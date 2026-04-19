use chrono::{DateTime, Utc};
use serde::Serialize;
use serde_json::Value;
use sqlx::FromRow;
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, FromRow)]
pub struct Source {
    pub id:           Uuid,
    pub workspace_id: Uuid,
    pub name:         String,
    pub source_type:  String,
    pub config:       Value,
    pub created_at:   DateTime<Utc>,
}
