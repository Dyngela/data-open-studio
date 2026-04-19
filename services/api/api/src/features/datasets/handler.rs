use axum::{
    extract::{Extension, Path, State},
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
    Json(body): Json<CreateDatasetReq>,
) -> Result<(StatusCode, Json<Value>), AppError> {
    let val = service::create(&state.db, &state.config.credentials_key, body, user.user_id).await?;
    Ok((StatusCode::CREATED, Json(val)))
}

pub async fn update(
    Path(id): Path<i64>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
    Json(body): Json<UpdateDatasetReq>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::update(&state.db, &state.config.credentials_key, id, body, user.user_id).await?))
}

pub async fn delete(
    Path(id): Path<i64>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
) -> Result<StatusCode, AppError> {
    service::delete(&state.db, id, user.user_id).await?;
    Ok(StatusCode::NO_CONTENT)
}

pub async fn refresh(
    Path(id): Path<i64>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::refresh(&state.db, &state.config.credentials_key, id, user.user_id).await?))
}

pub async fn preview(
    Path(id): Path<i64>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
    Json(body): Json<PreviewReq>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::preview(&state.db, &state.config.credentials_key, id, body, user.user_id).await?))
}

pub async fn query(
    Path(id): Path<i64>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
    Json(body): Json<QueryReq>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::query(&state.db, &state.config.credentials_key, id, body, user.user_id).await?))
}

pub async fn load_as_frame(
    Path(id): Path<i64>,
    State(state): State<AppState>,
    Extension(user): Extension<AuthUser>,
    Json(body): Json<LoadAsFrameReq>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::load_as_frame(&state.db, &state.config.credentials_key, id, body, user.user_id, &state).await?))
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
        .route("/api/v1/datasets",                      get(list).post(create))
        .route("/api/v1/datasets/{id}",                 get(get_by_id).put(update).delete(self::delete))
        .route("/api/v1/datasets/{id}/refresh",         post(refresh))
        .route("/api/v1/datasets/{id}/preview",         post(preview))
        .route("/api/v1/datasets/{id}/query",           post(query))
        .route("/api/v1/datasets/{id}/load-as-frame",   post(load_as_frame))
}
