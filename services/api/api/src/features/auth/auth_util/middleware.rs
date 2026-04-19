use axum::{extract::{Request, State}, middleware::Next, response::Response};

use crate::{error::AppError, state::AppState};
use crate::features::auth::auth_util::jwt;

#[derive(Clone, Debug)]
pub struct AuthUser {
    pub user_id: i64,
    pub email:   String,
    pub role:    String,
    pub prenom:  String,
    pub nom:     String,
}

pub async fn require_auth(
    State(state): State<AppState>,
    mut req:      Request,
    next:         Next,
) -> Result<Response, AppError> {
    let token = extract_bearer(&req)
        .ok_or_else(|| AppError::unauthorized("missing or invalid Authorization header"))?;
    let claims = jwt::decode_access(&token, &state.config.jwt.secret)?;
    req.extensions_mut().insert(AuthUser {
        user_id: claims.sub,
        email:   claims.email,
        role:    claims.role,
        prenom:  claims.prenom,
        nom:     claims.nom,
    });
    Ok(next.run(req).await)
}

fn extract_bearer(req: &Request) -> Option<String> {
    req.headers()
        .get(axum::http::header::AUTHORIZATION)?
        .to_str().ok()?
        .strip_prefix("Bearer ")
        .map(str::to_owned)
}
