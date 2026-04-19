use axum::{
    extract::{Extension, Path, Query, State},
    http::StatusCode,
    Json,
};
use serde_json::Value;

use crate::{error::AppError, state::AppState};
use super::{dto::*, service};

pub async fn list(
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::list(&state.db, user.user_id).await?))
}

pub async fn get_by_id(
    Path(id): Path<i64>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::get_by_id(&state.db, id, user.user_id).await?))
}

pub async fn create(
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
    Json(body): Json<CreateTriggerReq>,
) -> Result<(StatusCode, Json<Value>), AppError> {
    let val = service::create(&state.db, body, user.user_id).await?;
    Ok((StatusCode::CREATED, Json(val)))
}

pub async fn update(
    Path(id): Path<i64>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
    Json(body): Json<UpdateTriggerReq>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::update(&state.db, id, body, user.user_id).await?))
}

pub async fn delete(
    Path(id): Path<i64>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
) -> Result<StatusCode, AppError> {
    service::delete(&state.db, id, user.user_id).await?;
    Ok(StatusCode::NO_CONTENT)
}

pub async fn activate(
    Path(id): Path<i64>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::activate(&state.db, id, user.user_id).await?))
}

pub async fn pause(
    Path(id): Path<i64>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::pause(&state.db, id, user.user_id).await?))
}

pub async fn add_rule(
    Path(id): Path<i64>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
    Json(body): Json<CreateRuleReq>,
) -> Result<(StatusCode, Json<Value>), AppError> {
    let val = service::add_rule(&state.db, id, body, user.user_id).await?;
    Ok((StatusCode::CREATED, Json(val)))
}

pub async fn update_rule(
    Path((id, rule_id)): Path<(i64, i64)>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
    Json(body): Json<UpdateRuleReq>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::update_rule(&state.db, id, rule_id, body, user.user_id).await?))
}

pub async fn delete_rule(
    Path((id, rule_id)): Path<(i64, i64)>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
) -> Result<StatusCode, AppError> {
    service::delete_rule(&state.db, id, rule_id, user.user_id).await?;
    Ok(StatusCode::NO_CONTENT)
}

pub async fn link_job(
    Path(id): Path<i64>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
    Json(body): Json<LinkJobReq>,
) -> Result<(StatusCode, Json<Value>), AppError> {
    let val = service::link_job(&state.db, id, body, user.user_id).await?;
    Ok((StatusCode::CREATED, Json(val)))
}

pub async fn unlink_job(
    Path((id, job_id)): Path<(i64, i64)>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
) -> Result<StatusCode, AppError> {
    service::unlink_job(&state.db, id, job_id, user.user_id).await?;
    Ok(StatusCode::NO_CONTENT)
}

pub async fn get_executions(
    Path(id): Path<i64>,
    Query(params): Query<ExecutionsQuery>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::get_executions(&state.db, id, params, user.user_id).await?))
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
        .route("/api/v1/triggers",                          get(list).post(create))
        .route("/api/v1/triggers/{id}",                     get(get_by_id).put(update).delete(self::delete))
        .route("/api/v1/triggers/{id}/activate",            post(activate))
        .route("/api/v1/triggers/{id}/pause",               post(pause))
        .route("/api/v1/triggers/{id}/rules",               post(add_rule))
        .route("/api/v1/triggers/{id}/rules/{rule_id}",     put(update_rule).delete(delete_rule))
        .route("/api/v1/triggers/{id}/jobs",                post(link_job))
        .route("/api/v1/triggers/{id}/jobs/{job_id}",       delete(unlink_job))
        .route("/api/v1/triggers/{id}/executions",          get(get_executions))
}
