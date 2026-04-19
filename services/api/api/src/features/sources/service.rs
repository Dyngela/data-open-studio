use df_store::connectors::csv::{csv_to_frame, CsvConfig};
use df_store::connectors::postgres::{postgres_to_frame, PostgresConfig};
use df_store::frame::Frame;
use sqlx::PgPool;
use uuid::Uuid;

use crate::error::AppError;
use crate::frame_json::frame_schema_json;
use crate::state::{AppState, WorkspaceFrames};
use crate::storage;
use super::dto::CreateSourceRequest;
use super::model::Source;

pub async fn list(db: &PgPool, workspace_id: Uuid) -> Result<Vec<Source>, AppError> {
    sqlx::query_as(
        "SELECT * FROM source WHERE workspace_id = $1 ORDER BY created_at ASC",
    )
    .bind(workspace_id)
    .fetch_all(db)
    .await
    .map_err(AppError::from)
}

pub async fn create(
    db: &PgPool,
    workspace_id: Uuid,
    req: CreateSourceRequest,
) -> Result<Source, AppError> {
    if req.source_type != "csv" && req.source_type != "postgres" {
        return Err(AppError::bad_request("source_type must be 'csv' or 'postgres'"));
    }

    let exists: Option<(bool,)> =
        sqlx::query_as("SELECT true FROM source WHERE workspace_id = $1 AND name = $2 LIMIT 1")
            .bind(workspace_id)
            .bind(&req.name)
            .fetch_optional(db)
            .await?;
    if exists.is_some() {
        return Err(AppError::conflict(format!(
            "a source named '{}' already exists in this workspace",
            req.name
        )));
    }

    sqlx::query_as(
        "INSERT INTO source (id, workspace_id, name, source_type, config)
         VALUES (gen_random_uuid(), $1, $2, $3, $4) RETURNING *",
    )
    .bind(workspace_id)
    .bind(&req.name)
    .bind(&req.source_type)
    .bind(&req.config)
    .fetch_one(db)
    .await
    .map_err(fk_to_not_found)
}

pub async fn delete(db: &PgPool, workspace_id: Uuid, source_id: Uuid) -> Result<(), AppError> {
    let result = sqlx::query("DELETE FROM source WHERE id = $1 AND workspace_id = $2")
        .bind(source_id)
        .bind(workspace_id)
        .execute(db)
        .await?;

    if result.rows_affected() == 0 {
        return Err(AppError::not_found(format!("source {source_id} not found")));
    }

    Ok(())
}

pub async fn load(
    db: &PgPool,
    workspace_id: Uuid,
    source_id: Uuid,
    state: &AppState,
) -> Result<serde_json::Value, AppError> {
    let src: Option<Source> = sqlx::query_as(
        "SELECT * FROM source WHERE id = $1 AND workspace_id = $2",
    )
    .bind(source_id)
    .bind(workspace_id)
    .fetch_optional(db)
    .await?;

    let src = src.ok_or_else(|| AppError::not_found(format!("source {source_id} not found")))?;
    let frame_name = src.name.clone();

    let frame = run_connector(&src).await?;
    let schema = frame_schema_json(&frame);

    storage::spawn_persist(workspace_id, frame.clone());

    {
        let mut guard = state.workspaces.write().unwrap();
        let ws = guard.entry(workspace_id).or_insert_with(WorkspaceFrames::default);
        ws.frames.insert(frame_name.clone(), frame);
    }

    tracing::info!(
        workspace_id = %workspace_id,
        source_id = %source_id,
        frame = %frame_name,
        "source loaded into frame"
    );

    Ok(serde_json::json!({ "loaded": schema }))
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

async fn run_connector(src: &Source) -> Result<Frame, AppError> {
    match src.source_type.as_str() {
        "csv" => {
            let cfg: CsvConfig = serde_json::from_value(src.config.clone())
                .map_err(|e| AppError::bad_request(format!("invalid csv config: {e}")))?;
            csv_to_frame(&cfg).map_err(|e| AppError::bad_request(e.to_string()))
        }
        "postgres" => {
            let cfg: PostgresConfig = serde_json::from_value(src.config.clone())
                .map_err(|e| AppError::bad_request(format!("invalid postgres config: {e}")))?;
            tokio::task::spawn_blocking(move || postgres_to_frame(cfg))
                .await
                .map_err(|e| AppError::internal(e.to_string()))?
                .map_err(|e| AppError::bad_request(e.to_string()))
        }
        other => Err(AppError::bad_request(format!("unknown source type: {other}"))),
    }
}

fn fk_to_not_found(err: sqlx::Error) -> AppError {
    if let sqlx::Error::Database(ref dbe) = err {
        if dbe.code().as_deref() == Some("23503") {
            return AppError::not_found("workspace not found");
        }
    }
    AppError::from(err)
}
