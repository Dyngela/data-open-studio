use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    Json,
};
use serde_json::{json, Value};
use uuid::Uuid;

use crate::error::AppError;
use crate::state::AppState;
use super::{dto::PreviewQuery, service};

#[utoipa::path(
    get, path = "/api/v1/workspaces/{workspace_id}/frames",
    params(("workspace_id" = Uuid, Path, description = "Workspace UUID")),
    responses(
        (status = 200, description = "List of loaded frames with their schema"),
        (status = 401, description = "Unauthorized"),
    ),
    security(("bearer_auth" = [])),
    tag = "frames"
)]
pub async fn list(
    Path(workspace_id): Path<Uuid>,
    State(state): State<AppState>,
) -> Json<Value> {
    let frames = service::list(&state, workspace_id);
    Json(json!({ "frames": frames }))
}

#[utoipa::path(
    get, path = "/api/v1/workspaces/{workspace_id}/frames/{frame_name}/preview",
    params(
        ("workspace_id" = Uuid, Path, description = "Workspace UUID"),
        ("frame_name" = String, Path, description = "Frame name"),
        PreviewQuery,
    ),
    responses(
        (status = 200, description = "Paginated frame data"),
        (status = 404, description = "Frame or workspace not found"),
        (status = 401, description = "Unauthorized"),
    ),
    security(("bearer_auth" = [])),
    tag = "frames"
)]
pub async fn preview(
    Path((workspace_id, frame_name)): Path<(Uuid, String)>,
    Query(q): Query<PreviewQuery>,
    State(state): State<AppState>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::preview(&state, workspace_id, &frame_name, q.offset, q.limit)?))
}

#[utoipa::path(
    delete, path = "/api/v1/workspaces/{workspace_id}/frames/{frame_name}",
    params(
        ("workspace_id" = Uuid, Path, description = "Workspace UUID"),
        ("frame_name" = String, Path, description = "Frame name"),
    ),
    responses(
        (status = 204, description = "Frame unloaded"),
        (status = 404, description = "Frame not found"),
        (status = 401, description = "Unauthorized"),
    ),
    security(("bearer_auth" = [])),
    tag = "frames"
)]
pub async fn delete(
    Path((workspace_id, frame_name)): Path<(Uuid, String)>,
    State(state): State<AppState>,
) -> Result<StatusCode, AppError> {
    service::delete(&state, workspace_id, &frame_name)?;
    Ok(StatusCode::NO_CONTENT)
}

// ---------------------------------------------------------------------------
// OpenAPI doc for this feature
// ---------------------------------------------------------------------------

use utoipa::OpenApi;

#[derive(OpenApi)]
#[openapi(
    paths(list, preview, delete),
    components(schemas(PreviewQuery))
)]
pub struct ApiDoc;

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

pub fn routes() -> axum::Router<AppState> {
    use axum::routing::{delete, get};
    axum::Router::new()
        .route("/api/v1/workspaces/{workspace_id}/frames",                       get(list))
        .route("/api/v1/workspaces/{workspace_id}/frames/{frame_name}",          delete(self::delete))
        .route("/api/v1/workspaces/{workspace_id}/frames/{frame_name}/preview",  get(preview))
}
