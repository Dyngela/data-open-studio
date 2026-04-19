use serde_json::Value;
use sqlx::PgPool;

use crate::{error::AppError, state::AppState};
use super::dto::*;

pub fn build_pg_connstr(host: &str, port: i32, user: &str, password: &str, dbname: &str, ssl_mode: &str) -> String {
    format!("host={host} port={port} user={user} password={password} dbname={dbname} sslmode={ssl_mode}")
}

pub async fn resolve_connection(state: &AppState, body: &IntrospectRequest) -> Result<String, AppError> {
    if let Some(id) = body.metadata_database_id {
        use crate::features::metadata::model::MetadataDatabase;
        let meta: MetadataDatabase = sqlx::query_as("SELECT * FROM metadata_database WHERE id = $1")
            .bind(id)
            .fetch_optional(&state.db)
            .await?
            .ok_or_else(|| AppError::not_found("database metadata not found"))?;
        return Ok(build_pg_connstr(&meta.host, meta.port, &meta.user, &meta.password, &meta.database_name, &meta.ssl_mode));
    }
    if let Some(conn) = &body.connection {
        return Ok(build_pg_connstr(&conn.host, conn.port, &conn.user, &conn.password, &conn.database_name, &conn.ssl_mode));
    }
    Err(AppError::bad_request("provide metadata_database_id or connection"))
}

pub async fn test_connection(body: IntrospectRequest) -> Result<TestConnectionResult, AppError> {
    let conn = body.connection.ok_or_else(|| AppError::bad_request("connection required"))?;
    let conn_str = build_pg_connstr(&conn.host, conn.port, &conn.user, &conn.password, &conn.database_name, &conn.ssl_mode);
    match test_pg(&conn_str).await {
        Ok(v)  => Ok(TestConnectionResult { success: true,  message: "Connected".into(), version: Some(v) }),
        Err(e) => Ok(TestConnectionResult { success: false, message: e,                  version: None }),
    }
}

pub async fn get_tables(state: &AppState, body: IntrospectRequest) -> Result<DatabaseIntrospection, AppError> {
    let conn_str = resolve_connection(state, &body).await?;
    let tables = introspect_tables(&conn_str).await?;
    Ok(DatabaseIntrospection { tables, columns: vec![] })
}

pub async fn get_columns(state: &AppState, body: IntrospectRequest) -> Result<DatabaseIntrospection, AppError> {
    let conn_str = resolve_connection(state, &body).await?;
    let table = body.table_name.as_deref().unwrap_or("");
    let columns = introspect_columns(&conn_str, table).await?;
    Ok(DatabaseIntrospection { tables: vec![], columns })
}

pub async fn guess_query(state: &AppState, body: GuessQueryRequest) -> Result<GuessQueryResponse, AppError> {
    let schema_ctx = if let Some(id) = body.connection_id {
        fetch_schema_context(state, id).await.unwrap_or_default()
    } else {
        String::new()
    };

    let system_prompt = format!(
        "You are an expert SQL assistant. Generate a SQL query based on the user's request.\
        {}\
        Return ONLY the SQL query, no explanation.",
        if schema_ctx.is_empty() { String::new() } else { format!("\n\nDatabase schema:\n{schema_ctx}") }
    );

    let query = call_ollama(&state.config.ollama_url, &system_prompt, &body.prompt).await?;
    let query = clean_sql_response(&query);
    Ok(GuessQueryResponse { query })
}

pub async fn optimize_query(state: &AppState, body: OptimizeQueryRequest) -> Result<OptimizeQueryResponse, AppError> {
    let system_prompt = "You are an expert SQL optimizer. Optimize the given SQL query. \
        Respond with JSON: {\"optimized_query\": \"...\", \"explanation\": \"...\"}";

    let prompt = format!("Optimize this SQL query:\n{}", body.query);
    let response = call_ollama(&state.config.ollama_url, system_prompt, &prompt).await?;

    if let Ok(parsed) = serde_json::from_str::<serde_json::Value>(&response) {
        return Ok(OptimizeQueryResponse {
            optimized_query: parsed["optimized_query"].as_str().unwrap_or(&body.query).into(),
            explanation:     parsed["explanation"].as_str().unwrap_or("").into(),
        });
    }

    Ok(OptimizeQueryResponse {
        optimized_query: body.query,
        explanation:     response,
    })
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

async fn test_pg(conn_str: &str) -> Result<String, String> {
    use tokio_postgres::NoTls;
    let (client, conn) = tokio_postgres::connect(conn_str, NoTls).await.map_err(|e| e.to_string())?;
    tokio::spawn(async move { let _ = conn.await; });
    let row = client.query_one("SELECT version()", &[]).await.map_err(|e| e.to_string())?;
    Ok(row.get::<_, String>(0))
}

async fn introspect_tables(conn_str: &str) -> Result<Vec<DatabaseTable>, AppError> {
    use tokio_postgres::NoTls;
    let (client, conn) = tokio_postgres::connect(conn_str, NoTls)
        .await
        .map_err(|e| AppError::internal(format!("db connect: {e}")))?;
    tokio::spawn(async move { let _ = conn.await; });

    let rows = client.query(
        "SELECT table_schema, table_name
         FROM information_schema.tables
         WHERE table_type = 'BASE TABLE'
           AND table_schema NOT IN ('pg_catalog','information_schema')
         ORDER BY table_schema, table_name",
        &[],
    )
    .await
    .map_err(|e| AppError::internal(format!("introspect tables: {e}")))?;

    Ok(rows.iter().map(|r| DatabaseTable {
        schema: r.get(0),
        name:   r.get(1),
    }).collect())
}

async fn introspect_columns(conn_str: &str, table_name: &str) -> Result<Vec<DatabaseColumn>, AppError> {
    use tokio_postgres::NoTls;
    let (client, conn) = tokio_postgres::connect(conn_str, NoTls)
        .await
        .map_err(|e| AppError::internal(format!("db connect: {e}")))?;
    tokio::spawn(async move { let _ = conn.await; });

    let (schema, table) = match table_name.split_once('.') {
        Some((s, t)) => (s.to_owned(), t.to_owned()),
        None         => ("public".to_owned(), table_name.to_owned()),
    };

    let rows = client.query(
        "SELECT c.column_name, c.data_type, c.is_nullable,
                COALESCE(
                    (SELECT true FROM information_schema.table_constraints tc
                     JOIN information_schema.key_column_usage kcu
                       ON kcu.constraint_name = tc.constraint_name
                      AND kcu.table_schema    = tc.table_schema
                     WHERE tc.constraint_type = 'PRIMARY KEY'
                       AND kcu.table_schema   = c.table_schema
                       AND kcu.table_name     = c.table_name
                       AND kcu.column_name    = c.column_name
                     LIMIT 1
                    ), false
                ) AS is_primary
         FROM information_schema.columns c
         WHERE c.table_schema = $1 AND c.table_name = $2
         ORDER BY c.ordinal_position",
        &[&schema, &table],
    )
    .await
    .map_err(|e| AppError::internal(format!("introspect columns: {e}")))?;

    Ok(rows.iter().map(|r| DatabaseColumn {
        name:        r.get(0),
        data_type:   r.get(1),
        is_nullable: r.get::<_, &str>(2) == "YES",
        is_primary:  r.get(3),
    }).collect())
}

async fn fetch_schema_context(state: &AppState, metadata_id: i64) -> Option<String> {
    use crate::features::metadata::model::MetadataDatabase;
    let meta: MetadataDatabase = sqlx::query_as("SELECT * FROM metadata_database WHERE id = $1")
        .bind(metadata_id)
        .fetch_optional(&state.db)
        .await
        .ok()??;

    let conn_str = build_pg_connstr(&meta.host, meta.port, &meta.user, &meta.password, &meta.database_name, &meta.ssl_mode);
    let tables = introspect_tables(&conn_str).await.ok()?;

    let lines: Vec<String> = tables.iter()
        .map(|t| format!("{}.{}", t.schema, t.name))
        .collect();
    Some(lines.join("\n"))
}

async fn call_ollama(ollama_url: &str, system: &str, user: &str) -> Result<String, AppError> {
    let client = reqwest::Client::new();
    let body = serde_json::json!({
        "model": "llama3.2",
        "messages": [
            { "role": "system",  "content": system },
            { "role": "user",    "content": user }
        ],
        "stream": false
    });

    let resp = client
        .post(format!("{ollama_url}/api/chat"))
        .json(&body)
        .send()
        .await
        .map_err(|e| AppError::internal(format!("ollama request: {e}")))?;

    if !resp.status().is_success() {
        return Err(AppError::internal(format!("ollama error: {}", resp.status())));
    }

    let json: Value = resp.json().await
        .map_err(|e| AppError::internal(format!("ollama parse: {e}")))?;

    Ok(json["message"]["content"].as_str().unwrap_or("").to_owned())
}

fn clean_sql_response(raw: &str) -> String {
    let s = raw.trim();
    let s = s.trim_start_matches("```sql").trim_start_matches("```").trim_end_matches("```");
    s.trim().to_owned()
}

// ---------------------------------------------------------------------------
// db_node (guess schema)
// ---------------------------------------------------------------------------

pub struct DataModel {
    pub column:      String,
    pub data_type:   String,
    pub nullable:    bool,
    pub primary_key: bool,
}

pub async fn guess_schema(
    db: &PgPool,
    query: &str,
    node_id: i64,
    connection_id: i64,
) -> Result<(i64, Vec<DataModel>), AppError> {
    use crate::features::metadata::model::MetadataDatabase;

    let meta: MetadataDatabase =
        sqlx::query_as("SELECT * FROM metadata_database WHERE id = $1")
            .bind(connection_id)
            .fetch_optional(db)
            .await?
            .ok_or_else(|| AppError::not_found("database metadata not found"))?;

    let conn_str = format!(
        "host={} port={} user={} password={} dbname={} sslmode={}",
        meta.host, meta.port, meta.user, meta.password, meta.database_name, meta.ssl_mode
    );

    let models = run_schema_introspection(&conn_str, query).await?;
    Ok((node_id, models))
}

async fn run_schema_introspection(conn_str: &str, query: &str) -> Result<Vec<DataModel>, AppError> {
    use tokio_postgres::NoTls;

    let (client, conn) = tokio_postgres::connect(conn_str, NoTls)
        .await
        .map_err(|e| AppError::internal(format!("db connect: {e}")))?;
    tokio::spawn(async move { let _ = conn.await; });

    let wrapped = format!("SELECT * FROM ({}) AS __schema_probe LIMIT 0", query.trim_end_matches(';'));
    let stmt = client
        .prepare(&wrapped)
        .await
        .map_err(|e| AppError::bad_request(format!("query parse error: {e}")))?;

    let models = stmt.columns().iter().map(|col| {
        let type_name = col.type_().name().to_owned();
        DataModel {
            column:      col.name().to_owned(),
            data_type:   pg_type_to_generic(&type_name),
            nullable:    true,
            primary_key: false,
        }
    }).collect();

    Ok(models)
}

fn pg_type_to_generic(pg_type: &str) -> String {
    match pg_type {
        "bool"                                       => "boolean",
        "int2" | "int4" | "int8"                    => "integer",
        "float4" | "float8" | "numeric"             => "float",
        "text" | "varchar" | "bpchar" | "uuid"
        | "json" | "jsonb"                          => "string",
        "date"                                       => "date",
        "timestamp" | "timestamptz"                 => "datetime",
        "bytea"                                      => "binary",
        _                                            => "string",
    }
    .to_owned()
}
