use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    response::IntoResponse,
    Extension,
    Json,
};
use serde_json::Value;

use crate::{error::AppError, state::AppState};
use super::{
    dto::*,
    service,
};

pub async fn list(
    State(state): State<AppState>,
    Extension(caller): Extension<AuthUser>,
    Query(params): Query<ListQuery>,
) -> Result<Json<Vec<JobResponse>>, AppError> {
    Ok(Json(service::list(&state.db, caller.user_id, params.file_path.as_deref()).await?))
}

pub async fn get_by_id(
    State(state): State<AppState>,
    Extension(caller): Extension<AuthUser>,
    Path(id): Path<i64>,
) -> Result<Json<JobWithNodesResponse>, AppError> {
    Ok(Json(service::get(&state.db, id, caller.user_id).await?))
}

pub async fn create(
    State(state): State<AppState>,
    Extension(caller): Extension<AuthUser>,
    Json(body): Json<CreateJobRequest>,
) -> Result<impl IntoResponse, AppError> {
    let resp = service::create(&state.db, body, caller.user_id).await?;
    Ok((StatusCode::CREATED, Json(resp)))
}

pub async fn update(
    State(state): State<AppState>,
    Extension(caller): Extension<AuthUser>,
    Path(id): Path<i64>,
    Json(body): Json<UpdateJobRequest>,
) -> Result<Json<JobWithNodesResponse>, AppError> {
    Ok(Json(service::update(&state.db, id, body, caller.user_id).await?))
}

pub async fn delete(
    State(state): State<AppState>,
    Extension(caller): Extension<AuthUser>,
    Path(id): Path<i64>,
) -> Result<StatusCode, AppError> {
    service::delete(&state.db, id, caller.user_id).await?;
    Ok(StatusCode::NO_CONTENT)
}

pub async fn share(
    State(state): State<AppState>,
    Extension(caller): Extension<AuthUser>,
    Path(id): Path<i64>,
    Json(body): Json<ShareRequest>,
) -> Result<Json<JobWithNodesResponse>, AppError> {
    Ok(Json(service::share(&state.db, id, body, caller.user_id).await?))
}

pub async fn unshare(
    State(state): State<AppState>,
    Extension(caller): Extension<AuthUser>,
    Path(id): Path<i64>,
    Json(body): Json<ShareRequest>,
) -> Result<Json<JobWithNodesResponse>, AppError> {
    Ok(Json(service::unshare(&state.db, id, body, caller.user_id).await?))
}

pub async fn add_notification_contact(
    State(state): State<AppState>,
    Extension(caller): Extension<AuthUser>,
    Path(id): Path<i64>,
    Json(body): Json<AddNotificationContactRequest>,
) -> Result<Json<JobWithNodesResponse>, AppError> {
    Ok(Json(service::add_notification_contact(&state.db, id, body.user_id, caller.user_id).await?))
}

pub async fn remove_notification_contact(
    State(state): State<AppState>,
    Extension(caller): Extension<AuthUser>,
    Path((id, uid)): Path<(i64, i64)>,
) -> Result<Json<JobWithNodesResponse>, AppError> {
    Ok(Json(service::remove_notification_contact(&state.db, id, uid, caller.user_id).await?))
}

pub async fn execute(
    State(state): State<AppState>,
    Extension(caller): Extension<AuthUser>,
    Path(id): Path<i64>,
) -> Result<StatusCode, AppError> {
    service::execute_job(&state, id, caller.user_id).await?;
    Ok(StatusCode::ACCEPTED)
}

pub async fn stop(
    State(state): State<AppState>,
    Extension(caller): Extension<AuthUser>,
    Path(id): Path<i64>,
) -> Result<StatusCode, AppError> {
    service::stop_job(&state, id, caller.user_id).await?;
    Ok(StatusCode::OK)
}

pub async fn print_code(
    State(state): State<AppState>,
    Extension(caller): Extension<AuthUser>,
    Path(id): Path<i64>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::print_code(&state.db, id, caller.user_id).await?))
}

// ---------------------------------------------------------------------------
// OpenAPI doc for this feature
// ---------------------------------------------------------------------------

use utoipa::OpenApi;
use crate::features::auth::auth_util::middleware::AuthUser;

#[derive(OpenApi)]
#[openapi()]
pub struct ApiDoc;

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

pub fn routes() -> axum::Router<AppState> {
    use axum::routing::{delete, get, post, put};
    axum::Router::new()
        .route("/api/v1/jobs",     get(list).post(create))
        .route("/api/v1/jobs/{id}", get(get_by_id).put(update).delete(self::delete))
        .route("/api/v1/jobs/{id}/share",                    post(share).delete(unshare))
        .route("/api/v1/jobs/{id}/execute",                  post(execute))
        .route("/api/v1/jobs/{id}/stop",                     post(stop))
        .route("/api/v1/jobs/{id}/print-code",               post(print_code))
        .route("/api/v1/jobs/{id}/notification-contacts",     post(add_notification_contact))
        .route("/api/v1/jobs/{id}/notification-contacts/{user_id}", delete(remove_notification_contact))
}
