use std::collections::HashMap;
use std::sync::{Arc, RwLock};

use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    routing::{get, post},
    Json, Router,
};
use df_store::connectors::csv::{csv_to_frame, CsvConfig};
use df_store::connectors::postgres::{postgres_to_frame, PostgresConfig};
use df_store::data_type::{DataType, DataValue};
use df_store::frame::Frame;
use df_store::query::relationship::Relationship;
use df_store::query::resolver::resolve_frame;
use df_store::store::Series;
use serde::Deserialize;
use serde_json::{json, Value};
use tower_http::cors::CorsLayer;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

// ---------------------------------------------------------------------------
// Shared state
// ---------------------------------------------------------------------------

#[derive(Clone)]
struct AppState {
    frames:        Arc<RwLock<HashMap<String, Frame>>>,
    relationships: Arc<RwLock<Vec<Relationship>>>,
}

impl AppState {
    fn new() -> Self {
        Self {
            frames:        Arc::new(RwLock::new(HashMap::new())),
            relationships: Arc::new(RwLock::new(Vec::new())),
        }
    }
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

#[tokio::main]
async fn main() {
    tracing_subscriber::registry()
        .with(EnvFilter::try_from_default_env().unwrap_or_else(|_| "info".into()))
        .with(tracing_subscriber::fmt::layer())
        .init();

    let app = Router::new()
        // Frame management
        .route("/frames",               get(list_frames))
        .route("/frames/csv",           post(add_csv_frame))
        .route("/frames/postgres",      post(add_postgres_frame))
        .route("/frames/:name",         get(get_frame_schema).delete(delete_frame))
        .route("/frames/:name/data",    get(get_frame_data))
        // Relationship management
        .route("/relationships",        get(list_relationships).post(add_relationship))
        .route("/relationships/:id",    get(get_relationship).delete(delete_relationship))
        // Resolve
        .route("/resolve",              post(resolve))
        .layer(CorsLayer::permissive())
        .with_state(AppState::new());

    let addr = "0.0.0.0:3030";
    tracing::info!("viz api listening on {addr}");
    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

// ---------------------------------------------------------------------------
// Frame handlers
// ---------------------------------------------------------------------------

/// `GET /frames` — list all loaded frames with name and row count.
async fn list_frames(State(state): State<AppState>) -> Json<Value> {
    let frames = state.frames.read().unwrap();
    let list: Vec<Value> = frames
        .values()
        .map(|f| json!({ "name": f.name, "row_count": frame_row_count(f) }))
        .collect();
    Json(json!({ "frames": list }))
}

/// `POST /frames/csv`
/// Body: `{ "path": "...", "delimiter": 44, "has_header": true, "frame_name": "sales" }`
async fn add_csv_frame(
    State(state): State<AppState>,
    Json(config): Json<CsvConfig>,
) -> Result<Json<Value>, AppError> {
    let name  = config.frame_name.clone();
    let frame = csv_to_frame(&config).map_err(|e| AppError(e.to_string()))?;
    let schema = frame_schema_json(&frame);
    state.frames.write().unwrap().insert(name, frame);
    Ok(Json(schema))
}

/// `POST /frames/postgres`
/// Body: `{ "host":"...", "port":5432, "username":"...", "password":"...",
///          "database":"...", "query":"SELECT ...", "frame_name":"orders" }`
async fn add_postgres_frame(
    State(state): State<AppState>,
    Json(config): Json<PostgresConfig>,
) -> Result<Json<Value>, AppError> {
    let name  = config.frame_name.clone();
    let frame = tokio::task::spawn_blocking(move || postgres_to_frame(config))
        .await
        .map_err(|e| AppError(e.to_string()))?
        .map_err(|e| AppError(e.to_string()))?;
    let schema = frame_schema_json(&frame);
    state.frames.write().unwrap().insert(name, frame);
    Ok(Json(schema))
}

/// `GET /frames/:name` — return the frame schema.
async fn get_frame_schema(
    Path(name): Path<String>,
    State(state): State<AppState>,
) -> Result<Json<Value>, AppError> {
    let frames = state.frames.read().unwrap();
    let frame = frames.get(&name)
        .ok_or_else(|| AppError(format!("frame '{name}' not found")))?;
    Ok(Json(frame_schema_json(frame)))
}

#[derive(Deserialize)]
struct DataQuery {
    #[serde(default)]
    offset: usize,
    #[serde(default = "default_limit")]
    limit:  usize,
}
fn default_limit() -> usize { 100 }

/// `GET /frames/:name/data?offset=0&limit=100` — paginated column-oriented data.
async fn get_frame_data(
    Path(name): Path<String>,
    Query(q): Query<DataQuery>,
    State(state): State<AppState>,
) -> Result<Json<Value>, AppError> {
    let frames    = state.frames.read().unwrap();
    let frame     = frames.get(&name)
        .ok_or_else(|| AppError(format!("frame '{name}' not found")))?;
    let row_count = frame_row_count(frame);
    let offset    = q.offset.min(row_count);
    let end       = (offset + q.limit).min(row_count);

    let mut columns = serde_json::Map::new();
    for series in &frame.columns {
        let values: Vec<Value> = (offset..end).map(|i| series_get(series, i)).collect();
        columns.insert(series.field.name.clone(), Value::Array(values));
    }

    Ok(Json(json!({
        "name":      frame.name,
        "offset":    offset,
        "limit":     end - offset,
        "row_count": row_count,
        "columns":   columns,
    })))
}

/// `DELETE /frames/:name`
async fn delete_frame(
    Path(name): Path<String>,
    State(state): State<AppState>,
) -> Result<StatusCode, AppError> {
    state.frames.write().unwrap().remove(&name)
        .ok_or_else(|| AppError(format!("frame '{name}' not found")))?;
    Ok(StatusCode::NO_CONTENT)
}

// ---------------------------------------------------------------------------
// Relationship handlers
// ---------------------------------------------------------------------------

/// `GET /relationships`
async fn list_relationships(State(state): State<AppState>) -> Json<Value> {
    let rels = state.relationships.read().unwrap();
    Json(json!({ "relationships": *rels }))
}

/// `POST /relationships`
/// Body: `{ "id":"...", "from_frame":"sales", "from_col":"customer_id",
///          "to_frame":"customers", "to_col":"id", "join_type":"left" }`
/// `id` is optional — omit it and it is auto-generated from the column endpoints.
async fn add_relationship(
    State(state): State<AppState>,
    Json(mut rel): Json<Relationship>,
) -> Result<Json<Value>, AppError> {
    if rel.id.is_empty() {
        rel.id = Relationship::auto_id(&rel.from_frame, &rel.from_col, &rel.to_frame, &rel.to_col);
    }

    // Validate that both frames exist.
    {
        let frames = state.frames.read().unwrap();
        if !frames.contains_key(&rel.from_frame) {
            return Err(AppError(format!("frame '{}' not found", rel.from_frame)));
        }
        if !frames.contains_key(&rel.to_frame) {
            return Err(AppError(format!("frame '{}' not found", rel.to_frame)));
        }
    }

    let mut rels = state.relationships.write().unwrap();
    if rels.iter().any(|r| r.id == rel.id) {
        return Err(AppError(format!("relationship '{}' already exists", rel.id)));
    }
    let out = json!(rel);
    rels.push(rel);
    Ok(Json(out))
}

/// `GET /relationships/:id`
async fn get_relationship(
    Path(id): Path<String>,
    State(state): State<AppState>,
) -> Result<Json<Value>, AppError> {
    let rels = state.relationships.read().unwrap();
    let rel  = rels.iter().find(|r| r.id == id)
        .ok_or_else(|| AppError(format!("relationship '{id}' not found")))?;
    Ok(Json(json!(rel)))
}

/// `DELETE /relationships/:id`
async fn delete_relationship(
    Path(id): Path<String>,
    State(state): State<AppState>,
) -> Result<StatusCode, AppError> {
    let mut rels = state.relationships.write().unwrap();
    let before   = rels.len();
    rels.retain(|r| r.id != id);
    if rels.len() == before {
        return Err(AppError(format!("relationship '{id}' not found")));
    }
    Ok(StatusCode::NO_CONTENT)
}

// ---------------------------------------------------------------------------
// Resolve handler
// ---------------------------------------------------------------------------

#[derive(Deserialize)]
struct ResolveRequest {
    /// The fact frame to start from.
    base: String,
    /// Name for the resulting materialized frame.
    result_name: String,
}

/// `POST /resolve`
/// Body: `{ "base": "sales", "result_name": "sales_enriched" }`
///
/// Walks every relationship that starts from `base`, left/inner joins the
/// referenced dimension frames, and stores the flat result in the frame store.
async fn resolve(
    State(state): State<AppState>,
    Json(req): Json<ResolveRequest>,
) -> Result<Json<Value>, AppError> {
    let frames = state.frames.read().unwrap();
    let base   = frames.get(&req.base)
        .ok_or_else(|| AppError(format!("frame '{}' not found", req.base)))?;
    let rels   = state.relationships.read().unwrap();

    let resolved = resolve_frame(base, &rels, &frames, req.result_name.clone());
    let schema   = frame_schema_json(&resolved);

    drop(rels);
    drop(frames);
    state.frames.write().unwrap().insert(req.result_name, resolved);

    Ok(Json(schema))
}

// ---------------------------------------------------------------------------
// Serialization helpers
// ---------------------------------------------------------------------------

fn frame_row_count(frame: &Frame) -> usize {
    frame.columns.first().map(|s| s.len).unwrap_or(0)
}

fn frame_schema_json(frame: &Frame) -> Value {
    let columns: Vec<Value> = frame.columns.iter().map(|s| json!({
        "name":       s.field.name,
        "dtype":      dtype_str(&s.field.dtype),
        "len":        s.len,
        "null_count": s.null_count,
        "dict_size":  s.dict.len(),
        "min":        s.min.as_ref().map(dv_to_json).unwrap_or(Value::Null),
        "max":        s.max.as_ref().map(dv_to_json).unwrap_or(Value::Null),
    })).collect();
    json!({ "name": frame.name, "row_count": frame_row_count(frame), "columns": columns })
}

/// Resolve the value at row `i` — dereferences dict encoding.
fn series_get(series: &Series, i: usize) -> Value {
    match series.chunks.get(i) {
        None => Value::Null,
        Some(chunk) => match &chunk.data {
            DataValue::Null => Value::Null,
            DataValue::Int64(idx) if !series.dict.is_empty() => {
                series.dict.get(idx).map(dv_to_json).unwrap_or(Value::Null)
            }
            other => dv_to_json(other),
        },
    }
}

fn dv_to_json(v: &DataValue) -> Value {
    match v {
        DataValue::Null              => Value::Null,
        DataValue::Boolean(b)        => json!(b),
        DataValue::Int8(n)           => json!(n),
        DataValue::Int16(n)          => json!(n),
        DataValue::Int32(n)          => json!(n),
        DataValue::Int64(n)          => json!(n),
        DataValue::UInt8(n)          => json!(n),
        DataValue::UInt16(n)         => json!(n),
        DataValue::UInt32(n)         => json!(n),
        DataValue::UInt64(n)         => json!(n),
        DataValue::Float32(f)        => if f.is_finite() { json!(f) } else { Value::Null },
        DataValue::Float64(f)        => if f.is_finite() { json!(f) } else { Value::Null },
        DataValue::String(s)         => json!(s),
        DataValue::Date(d)           => json!(d),
        DataValue::Time(t)           => json!(t),
        DataValue::Datetime(ts,_,tz) => json!({ "ts": ts, "tz": tz }),
        DataValue::Binary(_)         => Value::Null,
        DataValue::Duration(d, _)    => json!(d),
        DataValue::List(_)           => Value::Null,
        DataValue::Struct(_)         => Value::Null,
    }
}

fn dtype_str(dt: &DataType) -> &'static str {
    match dt {
        DataType::Boolean  => "Boolean",
        DataType::UInt8    => "UInt8",
        DataType::UInt16   => "UInt16",
        DataType::UInt32   => "UInt32",
        DataType::UInt64   => "UInt64",
        DataType::Int8     => "Int8",
        DataType::Int16    => "Int16",
        DataType::Int32    => "Int32",
        DataType::Int64    => "Int64",
        DataType::Float32  => "Float32",
        DataType::Float64  => "Float64",
        DataType::String   => "String",
        DataType::Binary   => "Binary",
        DataType::Date     => "Date",
        DataType::Time     => "Time",
        DataType::Datetime => "Datetime",
        DataType::Duration => "Duration",
        DataType::List     => "List",
        DataType::Struct   => "Struct",
        DataType::Null     => "Null",
    }
}

// ---------------------------------------------------------------------------
// Error type
// ---------------------------------------------------------------------------

struct AppError(String);

impl axum::response::IntoResponse for AppError {
    fn into_response(self) -> axum::response::Response {
        (StatusCode::BAD_REQUEST, Json(json!({ "error": self.0 }))).into_response()
    }
}
