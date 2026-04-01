use axum::{
    extract::{Path, Query, State},
    Json,
};
use serde::Deserialize;
use serde_json::{json, Value};
use uuid::Uuid;

use crate::error::AppError;
use crate::frame_json::{frame_data_json, frame_schema_json};
use crate::state::AppState;

/// GET /workspaces/:workspace_id/frames
pub async fn list_frames(
    Path(workspace_id): Path<Uuid>,
    State(state): State<AppState>,
) -> Json<Value> {
    let guard = state.workspaces.read().unwrap();
    let frames: Vec<Value> = guard
        .get(&workspace_id)
        .map(|ws| ws.frames.values().map(frame_schema_json).collect())
        .unwrap_or_default();

    Json(json!({ "frames": frames }))
}

#[derive(Deserialize)]
pub struct PreviewQuery {
    #[serde(default)]
    pub offset: usize,
    #[serde(default = "default_limit")]
    pub limit: usize,
}
fn default_limit() -> usize { 100 }

/// GET /workspaces/:workspace_id/frames/:frame_name/preview?offset=0&limit=100
pub async fn preview_frame(
    Path((workspace_id, frame_name)): Path<(Uuid, String)>,
    Query(q): Query<PreviewQuery>,
    State(state): State<AppState>,
) -> Result<Json<Value>, AppError> {
    let guard = state.workspaces.read().unwrap();
    let ws = guard
        .get(&workspace_id)
        .ok_or_else(|| AppError::not_found(format!("no frames loaded for workspace {workspace_id}")))?;

    let frame = ws.frames.get(&frame_name)
        .ok_or_else(|| AppError::not_found(format!("frame '{frame_name}' not loaded")))?;

    Ok(Json(frame_data_json(frame, q.offset, q.limit)))
}
