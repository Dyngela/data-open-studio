use chrono::Utc;
use df_store::connectors::postgres::{postgres_to_frame, PostgresConfig};
use serde_json::{json, Value};
use sqlx::PgPool;
use tokio_postgres::NoTls;
use uuid::Uuid;

use crate::{
    error::AppError,
    features::metadata::service::dec,
    frame_json::frame_schema_json,
    state::{AppState, WorkspaceFrames},
    storage,
};
use super::{
    dto::*,
    model::DatasetRow,
    repository,
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

async fn get_meta(
    db: &PgPool,
    id: i64,
) -> Result<crate::features::metadata::model::MetadataDatabase, AppError> {
    use crate::features::metadata::model::MetadataDatabase;
    let row: Option<MetadataDatabase> =
        sqlx::query_as("SELECT * FROM metadata_database WHERE id = $1")
            .bind(id)
            .fetch_optional(db)
            .await?;
    row.ok_or_else(|| AppError::not_found("metadata database not found"))
}

fn check_access(ds: &DatasetRow, user_id: i64) -> Result<(), AppError> {
    if ds.creator_id != user_id {
        return Err(AppError::forbidden("access denied"));
    }
    Ok(())
}

fn dataset_to_json(ds: &DatasetRow) -> Value {
    json!({
        "id":                   ds.id,
        "name":                 ds.name,
        "description":          ds.description,
        "creator_id":           ds.creator_id,
        "metadata_database_id": ds.metadata_database_id,
        "status":               ds.status,
        "last_refreshed_at":    ds.last_refreshed_at,
        "last_error":           ds.last_error,
        "created_at":           ds.created_at,
        "updated_at":           ds.updated_at,
    })
}

fn dataset_detail_to_json(ds: &DatasetRow) -> Value {
    let mut v = dataset_to_json(ds);
    v["query"]  = json!(ds.query);
    v["schema"] = ds.schema.clone();
    v
}

async fn connect_pg(
    meta: &crate::features::metadata::model::MetadataDatabase,
    key: &[u8; 32],
) -> Result<tokio_postgres::Client, AppError> {
    let password = dec(key, &meta.password)?;
    let conn_str = format!(
        "host={} port={} user={} password={} dbname={} sslmode={}",
        meta.host, meta.port, meta.user, password, meta.database_name, meta.ssl_mode
    );
    let (client, conn) = tokio_postgres::connect(&conn_str, NoTls)
        .await
        .map_err(|e| AppError::bad_request(format!("db connect failed: {e}")))?;
    tokio::spawn(conn);
    Ok(client)
}

async fn detect_schema(
    meta: &crate::features::metadata::model::MetadataDatabase,
    query: &str,
    key: &[u8; 32],
) -> Result<(Value, Option<String>), AppError> {
    let client = match connect_pg(meta, key).await {
        Ok(c) => c,
        Err(e) => return Ok((json!({"columns": []}), Some(e.message))),
    };

    let wrapped = format!("SELECT * FROM ({query}) AS _ds_schema LIMIT 0");
    let stmt = match client.prepare(&wrapped).await {
        Ok(s) => s,
        Err(e) => return Ok((json!({"columns": []}), Some(e.to_string()))),
    };

    let columns: Vec<Value> = stmt.columns().iter().map(|c| {
        json!({
            "name":      c.name(),
            "data_type": map_pg_type(c.type_()),
            "nullable":  true,
        })
    }).collect();

    Ok((json!({ "columns": columns }), None))
}

fn map_pg_type(t: &tokio_postgres::types::Type) -> &'static str {
    use tokio_postgres::types::Type;
    match *t {
        Type::INT2 | Type::INT4 | Type::INT8 => "integer",
        Type::FLOAT4 | Type::FLOAT8 | Type::NUMERIC => "float",
        Type::BOOL => "boolean",
        Type::DATE => "date",
        Type::TIMESTAMP | Type::TIMESTAMPTZ => "datetime",
        _ => "string",
    }
}

async fn execute_query_rows(
    meta: &crate::features::metadata::model::MetadataDatabase,
    query: &str,
    filters: &[QueryFilter],
    limit: i64,
    key: &[u8; 32],
) -> Result<(Vec<String>, Vec<Value>), AppError> {
    let client = connect_pg(meta, key).await?;

    let (where_sql, params) = build_where_clause(filters);
    let limited = format!("SELECT * FROM ({query}) AS _ds {where_sql} LIMIT {limit}");

    let stmt = client.prepare(&limited).await
        .map_err(|e| AppError::bad_request(format!("query prepare failed: {e}")))?;

    let col_names: Vec<String> = stmt.columns().iter().map(|c| c.name().to_string()).collect();

    let param_refs: Vec<&(dyn tokio_postgres::types::ToSql + Sync)> =
        params.iter().map(|s| s as &(dyn tokio_postgres::types::ToSql + Sync)).collect();

    let rows = client.query(&stmt, param_refs.as_slice())
        .await
        .map_err(|e| AppError::internal(format!("query failed: {e}")))?;

    let result_rows: Vec<Value> = rows.iter().map(|row| {
        let mut obj = serde_json::Map::new();
        for (i, col) in col_names.iter().enumerate() {
            obj.insert(col.clone(), row_col_to_json(row, i));
        }
        Value::Object(obj)
    }).collect();

    Ok((col_names, result_rows))
}

fn build_where_clause(filters: &[QueryFilter]) -> (String, Vec<String>) {
    if filters.is_empty() {
        return (String::new(), vec![]);
    }
    let mut parts = Vec::new();
    let mut params = Vec::new();
    for (i, f) in filters.iter().enumerate() {
        let op = match f.operator.as_str() {
            "eq"   => "=",
            "neq"  => "!=",
            "gt"   => ">",
            "lt"   => "<",
            "gte"  => ">=",
            "lte"  => "<=",
            "like" => "LIKE",
            _      => continue,
        };
        parts.push(format!("\"{}\" {} ${}", f.column, op, i + 1));
        params.push(match &f.value {
            Value::String(s) => s.clone(),
            other => other.to_string(),
        });
    }
    if parts.is_empty() {
        return (String::new(), vec![]);
    }
    (format!("WHERE {}", parts.join(" AND ")), params)
}

fn row_col_to_json(row: &tokio_postgres::Row, i: usize) -> Value {
    if let Ok(v) = row.try_get::<_, Option<i64>>(i) {
        return v.map(Value::from).unwrap_or(Value::Null);
    }
    if let Ok(v) = row.try_get::<_, Option<f64>>(i) {
        return v.and_then(|f| serde_json::Number::from_f64(f).map(Value::Number)).unwrap_or(Value::Null);
    }
    if let Ok(v) = row.try_get::<_, Option<bool>>(i) {
        return v.map(Value::Bool).unwrap_or(Value::Null);
    }
    if let Ok(v) = row.try_get::<_, Option<String>>(i) {
        return v.map(Value::String).unwrap_or(Value::Null);
    }
    Value::Null
}

// ---------------------------------------------------------------------------
// Public service functions
// ---------------------------------------------------------------------------

pub async fn list(db: &PgPool, user_id: i64) -> Result<Value, AppError> {
    let rows = repository::list_by_creator(db, user_id).await?;
    Ok(json!({ "datasets": rows.iter().map(dataset_to_json).collect::<Vec<_>>() }))
}

pub async fn get_by_id(db: &PgPool, id: i64, user_id: i64) -> Result<Value, AppError> {
    let ds = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("dataset not found"))?;
    check_access(&ds, user_id)?;
    Ok(dataset_detail_to_json(&ds))
}

pub async fn create(
    db: &PgPool,
    key: &[u8; 32],
    body: CreateDatasetReq,
    user_id: i64,
) -> Result<Value, AppError> {
    if body.query.trim().is_empty() {
        return Err(AppError::bad_request("query is required"));
    }

    let meta = get_meta(db, body.metadata_database_id).await?;
    let (schema, err_msg) = detect_schema(&meta, &body.query, key).await?;
    let status = if err_msg.is_none() { "ready" } else { "error" };
    let last_error = err_msg.unwrap_or_default();
    let now = Utc::now();

    let ds = repository::create(
        db,
        &body.name,
        body.description.as_deref().unwrap_or(""),
        user_id,
        body.metadata_database_id,
        &body.query,
        &schema,
        status,
        if status == "ready" { Some(now) } else { None },
        &last_error,
    ).await?;

    Ok(dataset_detail_to_json(&ds))
}

pub async fn update(
    db: &PgPool,
    key: &[u8; 32],
    id: i64,
    body: UpdateDatasetReq,
    user_id: i64,
) -> Result<Value, AppError> {
    let mut ds = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("dataset not found"))?;
    check_access(&ds, user_id)?;

    let query_changed = body.query.is_some() || body.metadata_database_id.is_some();

    if let Some(name) = body.name { ds.name = name; }
    if let Some(desc) = body.description { ds.description = desc; }
    if let Some(q) = body.query { ds.query = q; }
    if let Some(mid) = body.metadata_database_id { ds.metadata_database_id = mid; }

    if query_changed {
        let meta = get_meta(db, ds.metadata_database_id).await?;
        let (schema, err_msg) = detect_schema(&meta, &ds.query, key).await?;
        ds.schema = schema;
        ds.status = if err_msg.is_none() { "ready".to_string() } else { "error".to_string() };
        ds.last_error = err_msg.unwrap_or_default();
        ds.last_refreshed_at = if ds.status == "ready" { Some(Utc::now()) } else { None };
    }

    let updated = repository::update(
        db, id,
        &ds.name, &ds.description, &ds.query, ds.metadata_database_id,
        &ds.schema, &ds.status, ds.last_refreshed_at, &ds.last_error,
    ).await?;

    Ok(dataset_detail_to_json(&updated))
}

pub async fn delete(db: &PgPool, id: i64, user_id: i64) -> Result<(), AppError> {
    let ds = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("dataset not found"))?;
    check_access(&ds, user_id)?;
    repository::soft_delete(db, id).await
}

pub async fn refresh(db: &PgPool, key: &[u8; 32], id: i64, user_id: i64) -> Result<Value, AppError> {
    let ds = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("dataset not found"))?;
    check_access(&ds, user_id)?;

    let meta = get_meta(db, ds.metadata_database_id).await?;
    let (schema, err_msg) = detect_schema(&meta, &ds.query, key).await?;
    let status = if err_msg.is_none() { "ready" } else { "error" };
    let last_error = err_msg.unwrap_or_default();

    let updated = repository::update(
        db, id,
        &ds.name, &ds.description, &ds.query, ds.metadata_database_id,
        &schema, status,
        if status == "ready" { Some(Utc::now()) } else { None },
        &last_error,
    ).await?;

    Ok(dataset_detail_to_json(&updated))
}

pub async fn preview(
    db: &PgPool,
    key: &[u8; 32],
    id: i64,
    body: PreviewReq,
    user_id: i64,
) -> Result<Value, AppError> {
    let ds = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("dataset not found"))?;
    check_access(&ds, user_id)?;

    let limit = body.limit.unwrap_or(100).min(1000).max(1);
    let meta = get_meta(db, ds.metadata_database_id).await?;
    let (columns, rows) = execute_query_rows(&meta, &ds.query, &[], limit, key).await?;

    Ok(json!({
        "columns":   columns,
        "rows":      rows,
        "row_count": rows.len(),
    }))
}

pub async fn query(
    db: &PgPool,
    key: &[u8; 32],
    id: i64,
    body: QueryReq,
    user_id: i64,
) -> Result<Value, AppError> {
    let ds = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("dataset not found"))?;
    check_access(&ds, user_id)?;

    let limit = body.limit.unwrap_or(1000).min(10000).max(1);
    let filters = body.filters.unwrap_or_default();
    let meta = get_meta(db, ds.metadata_database_id).await?;
    let (columns, rows) = execute_query_rows(&meta, &ds.query, &filters, limit, key).await?;

    Ok(json!({
        "columns":   columns,
        "rows":      rows,
        "row_count": rows.len(),
    }))
}

pub async fn load_as_frame(
    db: &PgPool,
    key: &[u8; 32],
    id: i64,
    body: LoadAsFrameReq,
    user_id: i64,
    state: &AppState,
) -> Result<Value, AppError> {
    let ds = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("dataset not found"))?;
    check_access(&ds, user_id)?;

    let meta = get_meta(db, ds.metadata_database_id).await?;
    let workspace_id = body.workspace_id;
    let frame_name = body.frame_name.unwrap_or_else(|| ds.name.clone());

    {
        let guard = state.workspaces.read().unwrap();
        if guard.get(&workspace_id).map_or(false, |ws| ws.frames.contains_key(&frame_name)) {
            return Err(AppError::conflict(format!(
                "frame '{}' is already loaded in workspace {}; delete it first or choose a different name",
                frame_name, workspace_id
            )));
        }
    }

    let cfg = PostgresConfig {
        host:       meta.host.clone(),
        port:       meta.port as u16,
        username:   meta.user.clone(),
        password:   dec(key, &meta.password)?,
        database:   meta.database_name.clone(),
        query:      ds.query.clone(),
        frame_name: frame_name.clone(),
    };

    let frame = tokio::task::spawn_blocking(move || postgres_to_frame(cfg))
        .await
        .map_err(|e| AppError::internal(e.to_string()))?
        .map_err(|e| AppError::bad_request(e.to_string()))?;

    let schema = frame_schema_json(&frame);

    storage::spawn_persist(workspace_id, frame.clone());

    {
        let mut guard = state.workspaces.write().unwrap();
        let ws = guard.entry(workspace_id).or_insert_with(WorkspaceFrames::default);
        ws.frames.insert(frame_name, frame);
    }

    Ok(json!({ "loaded": schema }))
}
