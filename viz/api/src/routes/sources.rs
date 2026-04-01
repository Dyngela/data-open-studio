use axum::{
    extract::{Path, State},
    http::StatusCode,
    Json,
};
use df_store::connectors::csv::{csv_to_frame, CsvConfig};
use df_store::connectors::postgres::{postgres_to_frame, PostgresConfig};
use serde_json::{json, Value};
use uuid::Uuid;

use crate::error::AppError;
use crate::frame_json::frame_schema_json;
use crate::models::{CreateSource, Source};
use crate::state::{AppState, WorkspaceFrames};

fn workspace_exists_check(err: sqlx::Error) -> AppError {
    // foreign-key violation → workspace not found
    if let sqlx::Error::Database(ref dbe) = err {
        if dbe.code().as_deref() == Some("23503") {
            return AppError::not_found("workspace not found");
        }
    }
    AppError::from(err)
}

/// POST /workspaces/:workspace_id/sources
pub async fn create_source(
    Path(workspace_id): Path<Uuid>,
    State(state): State<AppState>,
    Json(body): Json<CreateSource>,
) -> Result<(StatusCode, Json<Value>), AppError> {
    if body.source_type != "csv" && body.source_type != "postgres" {
        return Err(AppError::bad_request("source_type must be 'csv' or 'postgres'"));
    }

    let src: Source = sqlx::query_as(
        "INSERT INTO source (id, workspace_id, name, source_type, config)
         VALUES (gen_random_uuid(), $1, $2, $3, $4)
         RETURNING *",
    )
    .bind(workspace_id)
    .bind(&body.name)
    .bind(&body.source_type)
    .bind(&body.config)
    .fetch_one(&state.db)
    .await
    .map_err(workspace_exists_check)?;

    Ok((StatusCode::CREATED, Json(json!(src))))
}

/// GET /workspaces/:workspace_id/sources
pub async fn list_sources(
    Path(workspace_id): Path<Uuid>,
    State(state): State<AppState>,
) -> Result<Json<Value>, AppError> {
    let rows: Vec<Source> = sqlx::query_as(
        "SELECT * FROM source WHERE workspace_id = $1 ORDER BY created_at ASC",
    )
    .bind(workspace_id)
    .fetch_all(&state.db)
    .await?;

    Ok(Json(json!({ "sources": rows })))
}

/// DELETE /workspaces/:workspace_id/sources/:source_id
pub async fn delete_source(
    Path((workspace_id, source_id)): Path<(Uuid, Uuid)>,
    State(state): State<AppState>,
) -> Result<StatusCode, AppError> {
    let result = sqlx::query(
        "DELETE FROM source WHERE id = $1 AND workspace_id = $2",
    )
    .bind(source_id)
    .bind(workspace_id)
    .execute(&state.db)
    .await?;

    if result.rows_affected() == 0 {
        return Err(AppError::not_found(format!("source {source_id} not found")));
    }

    Ok(StatusCode::NO_CONTENT)
}

/// POST /workspaces/:workspace_id/sources/:source_id/load
///
/// Loads the source's data into an in-memory frame named after the source.
pub async fn load_source(
    Path((workspace_id, source_id)): Path<(Uuid, Uuid)>,
    State(state): State<AppState>,
) -> Result<Json<Value>, AppError> {
    let src: Option<Source> = sqlx::query_as(
        "SELECT * FROM source WHERE id = $1 AND workspace_id = $2",
    )
    .bind(source_id)
    .bind(workspace_id)
    .fetch_optional(&state.db)
    .await?;

    let src = src.ok_or_else(|| AppError::not_found(format!("source {source_id} not found")))?;

    let frame_name = src.name.clone();

    let frame = match src.source_type.as_str() {
        "csv" => {
            let cfg: CsvConfig = serde_json::from_value(src.config.clone())
                .map_err(|e| AppError::bad_request(format!("invalid csv config: {e}")))?;
            csv_to_frame(&cfg)
                .map_err(|e| AppError::bad_request(e.to_string()))?
        }
        "postgres" => {
            let cfg: PostgresConfig = serde_json::from_value(src.config.clone())
                .map_err(|e| AppError::bad_request(format!("invalid postgres config: {e}")))?;
            tokio::task::spawn_blocking(move || postgres_to_frame(cfg))
                .await
                .map_err(|e| AppError::internal(e.to_string()))?
                .map_err(|e| AppError::bad_request(e.to_string()))?
        }
        other => return Err(AppError::bad_request(format!("unknown source type: {other}"))),
    };

    let schema = frame_schema_json(&frame);

    // Store in-memory under the workspace.
    {
        let mut guard = state.workspaces.write().unwrap();
        let ws_frames = guard.entry(workspace_id).or_insert_with(WorkspaceFrames::default);
        ws_frames.frames.insert(frame_name, frame);
    }

    Ok(Json(json!({ "loaded": schema })))
}
