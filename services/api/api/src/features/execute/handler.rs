use axum::{
    extract::{Path, State},
    Json,
};
use serde_json::Value;
use uuid::Uuid;

use crate::error::AppError;
use crate::state::AppState;
use super::{dto::ExecuteRequest, service};

#[utoipa::path(
    post, path = "/api/v1/workspaces/{workspace_id}/execute",
    params(("workspace_id" = Uuid, Path, description = "Workspace UUID")),
    request_body = ExecuteRequest,
    responses(
        (status = 200, description = "Materialized frame schemas and optional result data"),
        (status = 400, description = "Lex, parse, or execution error"),
        (status = 401, description = "Unauthorized"),
    ),
    security(("bearer_auth" = [])),
    tag = "execute"
)]
pub async fn handle(
    Path(workspace_id): Path<Uuid>,
    State(state): State<AppState>,
    Json(body): Json<ExecuteRequest>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::run(&state, workspace_id, body)?))
}

// ---------------------------------------------------------------------------
// OpenAPI doc for this feature
// ---------------------------------------------------------------------------

use utoipa::OpenApi;

#[derive(OpenApi)]
#[openapi(
    paths(handle),
    components(schemas(ExecuteRequest))
)]
pub struct ApiDoc;

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

pub fn routes() -> axum::Router<AppState> {
    use axum::routing::post;
    axum::Router::new()
        .route("/api/v1/workspaces/{workspace_id}/execute", post(handle))
}
