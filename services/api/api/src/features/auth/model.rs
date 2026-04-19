use chrono::{DateTime, Utc};
use sqlx::FromRow;

#[derive(Debug, Clone, FromRow)]
pub struct User {
    pub id:            i64,
    pub email:         String,
    pub password:      String,
    pub prenom:        String,
    pub nom:           String,
    pub role:          String,
    pub actif:         bool,
    pub refresh_token: Option<String>,
    pub created_at:    DateTime<Utc>,
    pub updated_at:    DateTime<Utc>,
    pub deleted_at:    Option<DateTime<Utc>>,
}
