use axum::{
    extract::{Query, State},
    http::StatusCode,
    response::IntoResponse,
    Extension,
    Json,
};

use crate::{error::AppError, state::AppState};
use super::{
    dto::{AuthResponse, LoginRequest, RefreshRequest, RegisterRequest, SearchQuery, UserResponse},
    service,
};

#[utoipa::path(
    post, path = "/api/v1/auth/register",
    request_body = RegisterRequest,
    responses(
        (status = 201, description = "Registered", body = AuthResponse),
        (status = 400, description = "Email already registered"),
    ),
    tag = "auth_util"
)]
pub async fn register(
    State(state): State<AppState>,
    Json(body): Json<RegisterRequest>,
) -> Result<impl IntoResponse, AppError> {
    let resp = service::register(&state.db, &state.config, body).await?;
    Ok((StatusCode::CREATED, Json(resp)))
}

#[utoipa::path(
    post, path = "/api/v1/auth/login",
    request_body = LoginRequest,
    responses(
        (status = 200, description = "Login successful", body = AuthResponse),
        (status = 401, description = "Invalid credentials"),
    ),
    tag = "auth_util"
)]
pub async fn login(
    State(state): State<AppState>,
    Json(body): Json<LoginRequest>,
) -> Result<Json<AuthResponse>, AppError> {
    Ok(Json(service::login(&state.db, &state.config, body).await?))
}

#[utoipa::path(
    post, path = "/api/v1/auth/refresh",
    request_body = RefreshRequest,
    responses(
        (status = 200, description = "New tokens issued", body = AuthResponse),
        (status = 401, description = "Invalid or expired refresh token"),
    ),
    tag = "auth_util"
)]
pub async fn refresh_token(
    State(state): State<AppState>,
    Json(body): Json<RefreshRequest>,
) -> Result<Json<AuthResponse>, AppError> {
    Ok(Json(service::refresh(&state.db, &state.config, body).await?))
}

#[utoipa::path(
    get, path = "/api/v1/me",
    responses(
        (status = 200, description = "Current user", body = UserResponse),
        (status = 401, description = "Unauthorized"),
    ),
    security(("bearer_auth" = [])),
    tag = "auth_util"
)]
pub async fn me(
    State(state): State<AppState>,
    Extension(caller): Extension<AuthUser>,
) -> Result<Json<UserResponse>, AppError> {
    Ok(Json(service::me(&state.db, caller.user_id).await?))
}

#[utoipa::path(
    get, path = "/api/v1/users/search",
    params(("q" = String, Query, description = "Search term (email, prenom, nom)")),
    responses(
        (status = 200, description = "Matching users", body = Vec<UserResponse>),
        (status = 401, description = "Unauthorized"),
    ),
    security(("bearer_auth" = [])),
    tag = "auth_util"
)]
pub async fn search_users(
    State(state): State<AppState>,
    Query(params): Query<SearchQuery>,
) -> Result<Json<Vec<UserResponse>>, AppError> {
    Ok(Json(service::search_users(&state.db, &params.q).await?))
}

// ---------------------------------------------------------------------------
// OpenAPI doc for this feature
// ---------------------------------------------------------------------------

use utoipa::OpenApi;
use crate::features::auth::auth_util::middleware::AuthUser;

#[derive(OpenApi)]
#[openapi(
    paths(register, login, refresh_token, me, search_users),
    components(schemas(
        RegisterRequest, LoginRequest, RefreshRequest, SearchQuery,
        AuthResponse, UserResponse,
    ))
)]
pub struct ApiDoc;

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

pub fn routes() -> axum::Router<AppState> {
    use axum::routing::{get, post};
    axum::Router::new()
        .route("/api/v1/auth/register", post(register))
        .route("/api/v1/auth/login",    post(login))
        .route("/api/v1/auth/refresh",  post(refresh_token))
        .route("/api/v1/me",            get(me))
        .route("/api/v1/users/search",  get(search_users))
}
