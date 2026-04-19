use chrono::Utc;
use jsonwebtoken::{Algorithm, DecodingKey, EncodingKey, Header, Validation};
use serde::{Deserialize, Serialize};

use crate::error::AppError;

#[derive(Debug, Serialize, Deserialize)]
pub struct Claims {
    pub sub:    i64,
    pub email:  String,
    pub prenom: String,
    pub nom:    String,
    pub role:   String,
    pub exp:    i64,
    pub iat:    i64,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct RefreshClaims {
    pub sub: i64,
    pub exp: i64,
    pub iat: i64,
}

pub fn encode_access(
    user_id: i64, email: &str, prenom: &str, nom: &str, role: &str,
    secret: &str, expiry_minutes: i64,
) -> Result<String, AppError> {
    let now = Utc::now().timestamp();
    let claims = Claims {
        sub: user_id, email: email.into(), prenom: prenom.into(),
        nom: nom.into(), role: role.into(), iat: now, exp: now + expiry_minutes * 60,
    };
    jsonwebtoken::encode(&Header::default(), &claims, &EncodingKey::from_secret(secret.as_bytes()))
        .map_err(|e| AppError::internal(format!("jwt encode: {e}")))
}

pub fn encode_refresh(user_id: i64, secret: &str, expiry_days: i64) -> Result<String, AppError> {
    let now = Utc::now().timestamp();
    let claims = RefreshClaims { sub: user_id, iat: now, exp: now + expiry_days * 86400 };
    jsonwebtoken::encode(&Header::default(), &claims, &EncodingKey::from_secret(secret.as_bytes()))
        .map_err(|e| AppError::internal(format!("jwt encode: {e}")))
}

pub fn decode_access(token: &str, secret: &str) -> Result<Claims, AppError> {
    let mut v = Validation::new(Algorithm::HS256);
    v.validate_exp = true;
    jsonwebtoken::decode::<Claims>(token, &DecodingKey::from_secret(secret.as_bytes()), &v)
        .map(|d| d.claims)
        .map_err(|e| AppError::unauthorized(format!("invalid token: {e}")))
}

pub fn decode_refresh(token: &str, secret: &str) -> Result<RefreshClaims, AppError> {
    let mut v = Validation::new(Algorithm::HS256);
    v.validate_exp = true;
    jsonwebtoken::decode::<RefreshClaims>(token, &DecodingKey::from_secret(secret.as_bytes()), &v)
        .map(|d| d.claims)
        .map_err(|e| AppError::unauthorized(format!("invalid refresh token: {e}")))
}
