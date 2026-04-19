use serde_json::Value;
use uuid::Uuid;

use crate::error::AppError;
use crate::frame_json::{frame_data_json, frame_schema_json};
use crate::state::AppState;
use crate::storage;

pub fn list(state: &AppState, workspace_id: Uuid) -> Vec<Value> {
    let guard = state.workspaces.read().unwrap();
    guard
        .get(&workspace_id)
        .map(|ws| ws.frames.values().map(frame_schema_json).collect())
        .unwrap_or_default()
}

pub fn preview(
    state: &AppState,
    workspace_id: Uuid,
    frame_name: &str,
    offset: usize,
    limit: usize,
) -> Result<Value, AppError> {
    let guard = state.workspaces.read().unwrap();
    let ws = guard
        .get(&workspace_id)
        .ok_or_else(|| AppError::not_found(format!("no frames loaded for workspace {workspace_id}")))?;
    let frame = ws.frames.get(frame_name)
        .ok_or_else(|| AppError::not_found(format!("frame '{frame_name}' not loaded")))?;
    Ok(frame_data_json(frame, offset, limit))
}

pub fn delete(state: &AppState, workspace_id: Uuid, frame_name: &str) -> Result<(), AppError> {
    let removed = {
        let mut guard = state.workspaces.write().unwrap();
        guard
            .get_mut(&workspace_id)
            .and_then(|ws| ws.frames.remove(frame_name))
            .is_some()
    };

    if !removed {
        return Err(AppError::not_found(format!("frame '{frame_name}' not found")));
    }

    let cedr_path = storage::frame_file_path(workspace_id, frame_name);
    if cedr_path.exists() {
        if let Err(e) = std::fs::remove_file(&cedr_path) {
            tracing::warn!("Could not delete .cedr for '{}': {}", frame_name, e);
        }
    }

    tracing::info!(workspace_id = %workspace_id, frame = %frame_name, "frame unloaded");
    Ok(())
}
