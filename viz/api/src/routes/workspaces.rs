use axum::{
    extract::{Path, State},
    http::StatusCode,
    Json,
};
use serde_json::{json, Value};
use uuid::Uuid;

use crate::error::AppError;
use crate::models::{CreateWorkspace, UpdateWorkspace, Workspace};
use crate::state::AppState;

/// POST /workspaces
pub async fn create_workspace(
    State(state): State<AppState>,
    Json(body): Json<CreateWorkspace>,
) -> Result<(StatusCode, Json<Value>), AppError> {
    let ws: Workspace = sqlx::query_as(
        "INSERT INTO workspace (id, name) VALUES (gen_random_uuid(), $1) RETURNING *",
    )
    .bind(&body.name)
    .fetch_one(&state.db)
    .await?;

    Ok((StatusCode::CREATED, Json(json!(ws))))
}

/// GET /workspaces
pub async fn list_workspaces(
    State(state): State<AppState>,
) -> Result<Json<Value>, AppError> {
    let rows: Vec<Workspace> = sqlx::query_as(
        "SELECT * FROM workspace ORDER BY created_at DESC",
    )
    .fetch_all(&state.db)
    .await?;

    Ok(Json(json!({ "workspaces": rows })))
}

/// GET /workspaces/:id
pub async fn get_workspace(
    Path(id): Path<Uuid>,
    State(state): State<AppState>,
) -> Result<Json<Value>, AppError> {
    let ws: Option<Workspace> = sqlx::query_as(
        "SELECT * FROM workspace WHERE id = $1",
    )
    .bind(id)
    .fetch_optional(&state.db)
    .await?;

    match ws {
        Some(w) => Ok(Json(json!(w))),
        None    => Err(AppError::not_found(format!("workspace {id} not found"))),
    }
}

/// PATCH /workspaces/:id
pub async fn update_workspace(
    Path(id): Path<Uuid>,
    State(state): State<AppState>,
    Json(body): Json<UpdateWorkspace>,
) -> Result<Json<Value>, AppError> {
    let ws: Option<Workspace> = sqlx::query_as(
        "UPDATE workspace SET name = $1 WHERE id = $2 RETURNING *",
    )
    .bind(&body.name)
    .bind(id)
    .fetch_optional(&state.db)
    .await?;

    match ws {
        Some(w) => Ok(Json(json!(w))),
        None    => Err(AppError::not_found(format!("workspace {id} not found"))),
    }
}

/// DELETE /workspaces/:id
pub async fn delete_workspace(
    Path(id): Path<Uuid>,
    State(state): State<AppState>,
) -> Result<StatusCode, AppError> {
    let result = sqlx::query("DELETE FROM workspace WHERE id = $1")
        .bind(id)
        .execute(&state.db)
        .await?;

    if result.rows_affected() == 0 {
        return Err(AppError::not_found(format!("workspace {id} not found")));
    }

    // Drop in-memory frames for this workspace.
    state.workspaces.write().unwrap().remove(&id);

    Ok(StatusCode::NO_CONTENT)
}
