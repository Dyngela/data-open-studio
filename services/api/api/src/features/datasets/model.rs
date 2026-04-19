use chrono::{DateTime, Utc};
use serde_json::Value;
use sqlx::FromRow;

#[derive(Debug, Clone, FromRow)]
pub struct DatasetRow {
    pub id:                   i64,
    pub name:                 String,
    pub description:          String,
    pub creator_id:           i64,
    pub metadata_database_id: i64,
    pub query:                String,
    pub schema:               Value,
    pub status:               String,
    pub last_refreshed_at:    Option<DateTime<Utc>>,
    pub last_error:           String,
    pub created_at:           DateTime<Utc>,
    pub updated_at:           DateTime<Utc>,
    pub deleted_at:           Option<DateTime<Utc>>,
}
