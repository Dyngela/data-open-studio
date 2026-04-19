use chrono::{DateTime, Utc};
use serde_json::Value;
use sqlx::FromRow;

#[derive(Debug, Clone, FromRow)]
pub struct TriggerRow {
    pub id:               i64,
    pub name:             String,
    pub description:      String,
    #[sqlx(rename = "type")]
    pub trigger_type:     String,
    pub status:           String,
    pub creator_id:       i64,
    pub polling_interval: i32,
    pub last_polled_at:   Option<DateTime<Utc>>,
    pub last_error:       String,
    pub config:           Value,
    pub created_at:       DateTime<Utc>,
    pub updated_at:       DateTime<Utc>,
    pub deleted_at:       Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, FromRow)]
pub struct TriggerRuleRow {
    pub id:         i64,
    pub trigger_id: i64,
    pub name:       String,
    pub conditions: Value,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub deleted_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, FromRow)]
pub struct TriggerJobRow {
    pub id:              i64,
    pub trigger_id:      i64,
    pub job_id:          i64,
    pub priority:        i32,
    pub active:          bool,
    pub pass_event_data: bool,
    pub created_at:      DateTime<Utc>,
    pub deleted_at:      Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, FromRow)]
pub struct TriggerExecutionRow {
    pub id:             i64,
    pub trigger_id:     i64,
    pub started_at:     DateTime<Utc>,
    pub finished_at:    Option<DateTime<Utc>>,
    pub status:         String,
    pub event_count:    i32,
    pub jobs_triggered: i32,
    pub error:          String,
    pub event_sample:   Option<Value>,
}
