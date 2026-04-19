use sqlx::PgPool;

use crate::{
    config::AppConfig,
    error::AppError,
};
use crate::features::auth::auth_util::{jwt, password};
use super::{
    dto::{AuthResponse, LoginRequest, RefreshRequest, RegisterRequest, UserResponse},
    model::User,
    repository,
};

fn make_tokens(user: &User, config: &AppConfig) -> Result<(String, String), AppError> {
    let access = jwt::encode_access(
        user.id, &user.email, &user.prenom, &user.nom, &user.role,
        &config.jwt.secret, config.jwt.expiry_minutes,
    )?;
    let refresh = jwt::encode_refresh(user.id, &config.jwt.secret, config.jwt.refresh_expiry_days)?;
    Ok((access, refresh))
}

fn to_auth_response(user: User, token: String, refresh_token: String) -> AuthResponse {
    AuthResponse {
        token,
        refresh_token,
        user: UserResponse::from(user),
    }
}

pub async fn register(db: &PgPool, config: &AppConfig, req: RegisterRequest) -> Result<AuthResponse, AppError> {
    if repository::email_exists(db, &req.email).await? {
        return Err(AppError::bad_request("email already registered"));
    }

    let hash = password::hash(&req.password)?;
    let user = repository::create(db, &req.email, &hash, &req.prenom, &req.nom).await?;

    let (token, refresh_token) = make_tokens(&user, config)?;
    repository::update_refresh_token(db, user.id, &refresh_token).await?;

    Ok(to_auth_response(user, token, refresh_token))
}

pub async fn login(db: &PgPool, config: &AppConfig, req: LoginRequest) -> Result<AuthResponse, AppError> {
    tracing::info!("login attempt for email: {}", req.email);

    let user = repository::find_by_email(db, &req.email)
        .await
        .map_err(|e| {
            tracing::error!("database error during login: {:?}", e);
            AppError::internal("database error")
        })?
        .ok_or_else(|| AppError::unauthorized("invalid credentials"))?;

    if !user.actif {
        return Err(AppError::unauthorized("account is disabled"));
    }

    if !password::verify(&req.password, &user.password)? {
        tracing::error!("password verify failed");
        return Err(AppError::unauthorized("invalid credentials"));
    }

    let (token, refresh_token) = make_tokens(&user, config)?;
    repository::update_refresh_token(db, user.id, &refresh_token).await?;

    Ok(to_auth_response(user, token, refresh_token))
}

pub async fn refresh(db: &PgPool, config: &AppConfig, req: RefreshRequest) -> Result<AuthResponse, AppError> {
    let claims = jwt::decode_refresh(&req.refresh_token, &config.jwt.secret)?;

    let user = repository::find_by_id(db, claims.sub)
        .await?
        .ok_or_else(|| AppError::unauthorized("user not found"))?;

    if user.refresh_token.as_deref() != Some(&req.refresh_token) {
        return Err(AppError::unauthorized("refresh token mismatch"));
    }

    let (token, new_refresh) = make_tokens(&user, config)?;
    repository::update_refresh_token(db, user.id, &new_refresh).await?;

    Ok(to_auth_response(user, token, new_refresh))
}

pub async fn me(db: &PgPool, user_id: i64) -> Result<UserResponse, AppError> {
    let user = repository::find_by_id(db, user_id)
        .await?
        .ok_or_else(|| AppError::not_found("user not found"))?;
    Ok(UserResponse::from(user))
}

pub async fn search_users(db: &PgPool, q: &str) -> Result<Vec<UserResponse>, AppError> {
    let users = repository::search(db, q).await?;
    Ok(users.into_iter().map(UserResponse::from).collect())
}
