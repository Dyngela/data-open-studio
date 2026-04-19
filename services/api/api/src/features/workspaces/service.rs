use sqlx::PgPool;
use uuid::Uuid;

use crate::error::AppError;
use super::{
    dto::{CreateWorkspaceRequest, UpdateWorkspaceRequest, WorkspaceResponse},
    repository,
};

pub async fn list(db: &PgPool) -> Result<Vec<WorkspaceResponse>, AppError> {
    let rows = repository::list(db).await?;
    Ok(rows.into_iter().map(WorkspaceResponse::from).collect())
}

pub async fn get(db: &PgPool, id: Uuid) -> Result<WorkspaceResponse, AppError> {
    repository::find_by_id(db, id)
        .await?
        .map(WorkspaceResponse::from)
        .ok_or_else(|| AppError::not_found(format!("workspace {id} not found")))
}

pub async fn create(db: &PgPool, req: CreateWorkspaceRequest) -> Result<WorkspaceResponse, AppError> {
    let ws = repository::create(db, &req.name).await?;
    tracing::info!(workspace_id = %ws.id, name = %ws.name, "workspace created");
    Ok(WorkspaceResponse::from(ws))
}

pub async fn update(db: &PgPool, id: Uuid, req: UpdateWorkspaceRequest) -> Result<WorkspaceResponse, AppError> {
    repository::update(db, id, &req.name)
        .await?
        .map(WorkspaceResponse::from)
        .ok_or_else(|| AppError::not_found(format!("workspace {id} not found")))
}

pub async fn delete(
    db: &PgPool,
    id: Uuid,
    state: &crate::state::AppState,
) -> Result<(), AppError> {
    let n = repository::delete(db, id).await?;
    if n == 0 {
        return Err(AppError::not_found(format!("workspace {id} not found")));
    }

    state.workspaces.write().unwrap().remove(&id);

    let store_dir = crate::storage::frame_store_dir(id);
    if store_dir.exists() {
        if let Err(e) = std::fs::remove_dir_all(&store_dir) {
            tracing::warn!("Could not remove store dir {:?}: {e}", store_dir);
        }
    }

    tracing::info!(workspace_id = %id, "workspace deleted");
    Ok(())
}
