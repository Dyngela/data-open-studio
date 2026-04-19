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
use super::{
    dto::{CreateWorkspaceRequest, UpdateWorkspaceRequest, WorkspaceResponse},
    service,
};

#[utoipa::path(
    get, path = "/api/v1/workspaces",
    responses(
        (status = 200, description = "List of workspaces"),
        (status = 401, description = "Unauthorized"),
    ),
    security(("bearer_auth" = [])),
    tag = "workspaces"
)]
pub async fn list(State(state): State<AppState>) -> Result<Json<Value>, AppError> {
    let rows = service::list(&state.db).await?;
    Ok(Json(json!({ "workspaces": rows })))
}

#[utoipa::path(
    post, path = "/api/v1/workspaces",
    request_body = CreateWorkspaceRequest,
    responses(
        (status = 201, description = "Workspace created"),
        (status = 401, description = "Unauthorized"),
    ),
    security(("bearer_auth" = [])),
    tag = "workspaces"
)]
pub async fn create(
    State(state): State<AppState>,
    Json(body): Json<CreateWorkspaceRequest>,
) -> Result<impl IntoResponse, AppError> {
    let ws = service::create(&state.db, body).await?;
    Ok((StatusCode::CREATED, Json(json!(ws))))
}

#[utoipa::path(
    get, path = "/api/v1/workspaces/{id}",
    params(("id" = Uuid, Path, description = "Workspace UUID")),
    responses(
        (status = 200, description = "Workspace details"),
        (status = 404, description = "Not found"),
        (status = 401, description = "Unauthorized"),
    ),
    security(("bearer_auth" = [])),
    tag = "workspaces"
)]
pub async fn get(
    Path(id): Path<Uuid>,
    State(state): State<AppState>,
) -> Result<Json<Value>, AppError> {
    let ws = service::get(&state.db, id).await?;
    Ok(Json(json!(ws)))
}

#[utoipa::path(
    patch, path = "/api/v1/workspaces/{id}",
    params(("id" = Uuid, Path, description = "Workspace UUID")),
    request_body = UpdateWorkspaceRequest,
    responses(
        (status = 200, description = "Workspace updated"),
        (status = 404, description = "Not found"),
        (status = 401, description = "Unauthorized"),
    ),
    security(("bearer_auth" = [])),
    tag = "workspaces"
)]
pub async fn update(
    Path(id): Path<Uuid>,
    State(state): State<AppState>,
    Json(body): Json<UpdateWorkspaceRequest>,
) -> Result<Json<Value>, AppError> {
    let ws = service::update(&state.db, id, body).await?;
    Ok(Json(json!(ws)))
}

#[utoipa::path(
    delete, path = "/api/v1/workspaces/{id}",
    params(("id" = Uuid, Path, description = "Workspace UUID")),
    responses(
        (status = 204, description = "Workspace deleted"),
        (status = 404, description = "Not found"),
        (status = 401, description = "Unauthorized"),
    ),
    security(("bearer_auth" = [])),
    tag = "workspaces"
)]
pub async fn delete(
    Path(id): Path<Uuid>,
    State(state): State<AppState>,
) -> Result<StatusCode, AppError> {
    service::delete(&state.db, id, &state).await?;
    Ok(StatusCode::NO_CONTENT)
}

// ---------------------------------------------------------------------------
// OpenAPI doc for this feature
// ---------------------------------------------------------------------------

use utoipa::OpenApi;

#[derive(OpenApi)]
#[openapi(
    paths(list, create, get, update, delete),
    components(schemas(
        CreateWorkspaceRequest, UpdateWorkspaceRequest, WorkspaceResponse,
    ))
)]
pub struct ApiDoc;

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

pub fn routes() -> axum::Router<AppState> {
    use axum::routing::get;
    axum::Router::new()
        .route("/api/v1/workspaces",      get(list).post(create))
        .route("/api/v1/workspaces/{id}", get(self::get).patch(update).delete(self::delete))
}
