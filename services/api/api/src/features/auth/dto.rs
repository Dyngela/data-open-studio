use serde::{Deserialize, Serialize};
use utoipa::ToSchema;

use super::model::User;

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

#[derive(Deserialize, ToSchema)]
pub struct RegisterRequest {
    pub email:    String,
    pub password: String,
    pub prenom:   String,
    pub nom:      String,
}

#[derive(Deserialize, ToSchema)]
pub struct LoginRequest {
    pub email:    String,
    pub password: String,
}

#[derive(Deserialize, ToSchema)]
pub struct RefreshRequest {
    pub refresh_token: String,
}

#[derive(Deserialize, ToSchema)]
pub struct SearchQuery {
    pub q: String,
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

#[derive(Serialize, ToSchema)]
pub struct UserResponse {
    pub id:     i64,
    pub email:  String,
    pub prenom: String,
    pub nom:    String,
    pub actif:  bool,
}

impl From<User> for UserResponse {
    fn from(u: User) -> Self {
        Self {
            id:     u.id,
            email:  u.email,
            prenom: u.prenom,
            nom:    u.nom,
            actif:  u.actif,
        }
    }
}

#[derive(Serialize, ToSchema)]
pub struct AuthResponse {
    pub token:         String,
    pub refresh_token: String,
    pub user:          UserResponse,
}
