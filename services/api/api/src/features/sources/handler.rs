use axum::{
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    Json,
};
use serde_json::{json, Value};
use uuid::Uuid;

use crate::error::AppError;
use crate::state::AppState;
use super::{dto::CreateSourceRequest, service};

#[utoipa::path(
    get, path = "/api/v1/workspaces/{workspace_id}/sources",
    params(("workspace_id" = Uuid, Path, description = "Workspace UUID")),
    responses(
        (status = 200, description = "List of sources"),
        (status = 401, description = "Unauthorized"),
    ),
    security(("bearer_auth" = [])),
    tag = "sources"
)]
pub async fn list(
    Path(workspace_id): Path<Uuid>,
    State(state): State<AppState>,
) -> Result<Json<Value>, AppError> {
    let rows = service::list(&state.db, workspace_id).await?;
    Ok(Json(json!({ "sources": rows })))
}

#[utoipa::path(
    post, path = "/api/v1/workspaces/{workspace_id}/sources",
    params(("workspace_id" = Uuid, Path, description = "Workspace UUID")),
    request_body = CreateSourceRequest,
    responses(
        (status = 201, description = "Source created"),
        (status = 400, description = "Invalid source_type"),
        (status = 409, description = "Source name already exists in workspace"),
        (status = 401, description = "Unauthorized"),
    ),
    security(("bearer_auth" = [])),
    tag = "sources"
)]
pub async fn create(
    Path(workspace_id): Path<Uuid>,
    State(state): State<AppState>,
    Json(body): Json<CreateSourceRequest>,
) -> Result<impl IntoResponse, AppError> {
    let src = service::create(&state.db, workspace_id, body).await?;
    tracing::info!(workspace_id = %workspace_id, source_id = %src.id, name = %src.name, "source created");
    Ok((StatusCode::CREATED, Json(json!(src))))
}

#[utoipa::path(
    delete, path = "/api/v1/workspaces/{workspace_id}/sources/{source_id}",
    params(
        ("workspace_id" = Uuid, Path, description = "Workspace UUID"),
        ("source_id" = Uuid, Path, description = "Source UUID"),
    ),
    responses(
        (status = 204, description = "Source deleted"),
        (status = 404, description = "Not found"),
        (status = 401, description = "Unauthorized"),
    ),
    security(("bearer_auth" = [])),
    tag = "sources"
)]
pub async fn delete(
    Path((workspace_id, source_id)): Path<(Uuid, Uuid)>,
    State(state): State<AppState>,
) -> Result<StatusCode, AppError> {
    service::delete(&state.db, workspace_id, source_id).await?;
    Ok(StatusCode::NO_CONTENT)
}

#[utoipa::path(
    post, path = "/api/v1/workspaces/{workspace_id}/sources/{source_id}/load",
    params(
        ("workspace_id" = Uuid, Path, description = "Workspace UUID"),
        ("source_id" = Uuid, Path, description = "Source UUID"),
    ),
    responses(
        (status = 200, description = "Frame loaded; returns schema"),
        (status = 400, description = "Connector error"),
        (status = 404, description = "Source not found"),
        (status = 401, description = "Unauthorized"),
    ),
    security(("bearer_auth" = [])),
    tag = "sources"
)]
pub async fn load(
    Path((workspace_id, source_id)): Path<(Uuid, Uuid)>,
    State(state): State<AppState>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::load(&state.db, workspace_id, source_id, &state).await?))
}

// ---------------------------------------------------------------------------
// OpenAPI doc for this feature
// ---------------------------------------------------------------------------

use utoipa::OpenApi;

#[derive(OpenApi)]
#[openapi(
    paths(list, create, delete, load),
    components(schemas(CreateSourceRequest))
)]
pub struct ApiDoc;

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

pub fn routes() -> axum::Router<AppState> {
    use axum::routing::{delete, get, post};
    axum::Router::new()
        .route("/api/v1/workspaces/{workspace_id}/sources",                      get(list).post(create))
        .route("/api/v1/workspaces/{workspace_id}/sources/{source_id}",          delete(self::delete))
        .route("/api/v1/workspaces/{workspace_id}/sources/{source_id}/load",     post(load))
}
