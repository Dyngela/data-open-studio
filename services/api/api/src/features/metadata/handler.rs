use axum::{
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    Json,
};
use serde_json::Value;

use crate::error::AppError;
use crate::state::AppState;
use super::{dto::*, service};

// ---------------------------------------------------------------------------
// Database handlers
// ---------------------------------------------------------------------------

pub async fn db_list(State(state): State<AppState>) -> Result<Json<Value>, AppError> {
    Ok(Json(service::db_list(&state.db).await?))
}

pub async fn db_get(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::db_get(&state.db, id).await?))
}

pub async fn db_create(
    State(state): State<AppState>,
    Json(body): Json<CreateDbMetadata>,
) -> Result<impl IntoResponse, AppError> {
    let row = service::db_create(&state.db, &state.config.credentials_key, body).await?;
    Ok((StatusCode::CREATED, Json(row)))
}

pub async fn db_update(
    State(state): State<AppState>,
    Path(id): Path<i64>,
    Json(body): Json<UpdateDbMetadata>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::db_update(&state.db, &state.config.credentials_key, id, body).await?))
}

pub async fn db_delete(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<StatusCode, AppError> {
    service::db_delete(&state.db, id).await?;
    Ok(StatusCode::NO_CONTENT)
}

pub async fn db_test_connection(
    Json(body): Json<TestDbConnectionRequest>,
) -> Json<Value> {
    Json(service::db_test_connection(body).await)
}

// ---------------------------------------------------------------------------
// SFTP handlers
// ---------------------------------------------------------------------------

pub async fn sftp_list(State(state): State<AppState>) -> Result<Json<Value>, AppError> {
    Ok(Json(service::sftp_list(&state.db).await?))
}

pub async fn sftp_get(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::sftp_get(&state.db, id).await?))
}

pub async fn sftp_create(
    State(state): State<AppState>,
    Json(body): Json<CreateSftpMetadata>,
) -> Result<impl IntoResponse, AppError> {
    let row = service::sftp_create(&state.db, &state.config.credentials_key, body).await?;
    Ok((StatusCode::CREATED, Json(row)))
}

pub async fn sftp_update(
    State(state): State<AppState>,
    Path(id): Path<i64>,
    Json(body): Json<UpdateSftpMetadata>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::sftp_update(&state.db, &state.config.credentials_key, id, body).await?))
}

pub async fn sftp_delete(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<StatusCode, AppError> {
    service::sftp_delete(&state.db, id).await?;
    Ok(StatusCode::NO_CONTENT)
}

// ---------------------------------------------------------------------------
// Email handlers
// ---------------------------------------------------------------------------

pub async fn email_list(State(state): State<AppState>) -> Result<Json<Value>, AppError> {
    Ok(Json(service::email_list(&state.db).await?))
}

pub async fn email_get(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::email_get(&state.db, id).await?))
}

pub async fn email_create(
    State(state): State<AppState>,
    Json(body): Json<CreateEmailMetadata>,
) -> Result<impl IntoResponse, AppError> {
    let row = service::email_create(&state.db, &state.config.credentials_key, body).await?;
    Ok((StatusCode::CREATED, Json(row)))
}

pub async fn email_update(
    State(state): State<AppState>,
    Path(id): Path<i64>,
    Json(body): Json<UpdateEmailMetadata>,
) -> Result<Json<Value>, AppError> {
    Ok(Json(service::email_update(&state.db, &state.config.credentials_key, id, body).await?))
}

pub async fn email_delete(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<StatusCode, AppError> {
    service::email_delete(&state.db, id).await?;
    Ok(StatusCode::NO_CONTENT)
}

pub async fn email_test_connection(
    Json(body): Json<TestEmailConnectionRequest>,
) -> Json<Value> {
    Json(service::email_test_connection(body).await)
}

// ---------------------------------------------------------------------------
// OpenAPI doc for this feature
// ---------------------------------------------------------------------------

use utoipa::OpenApi;

#[derive(OpenApi)]
#[openapi()]
pub struct ApiDoc;

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

pub fn routes() -> axum::Router<AppState> {
    use axum::routing::{delete, get, post, put};
    axum::Router::new()
        .route("/api/v1/metadata/db",                     get(db_list).post(db_create))
        .route("/api/v1/metadata/db/{id}",                get(db_get).put(db_update).delete(db_delete))
        .route("/api/v1/metadata/db/test-connection",     post(db_test_connection))
        .route("/api/v1/metadata/sftp",                   get(sftp_list).post(sftp_create))
        .route("/api/v1/metadata/sftp/{id}",              get(sftp_get).put(sftp_update).delete(sftp_delete))
        .route("/api/v1/metadata/email",                  get(email_list).post(email_create))
        .route("/api/v1/metadata/email/{id}",             get(email_get).put(email_update).delete(email_delete))
        .route("/api/v1/metadata/email/test-connection",  post(email_test_connection))
}
