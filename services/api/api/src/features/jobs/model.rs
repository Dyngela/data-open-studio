use chrono::{DateTime, Utc};
use serde_json::Value;
use sqlx::FromRow;

#[derive(Debug, Clone, FromRow)]
pub struct Job {
    pub id:          i64,
    pub name:        String,
    pub description: String,
    pub file_path:   String,
    pub creator_id:  i64,
    pub active:      bool,
    pub visibility:  String,
    pub output_path: String,
    pub created_at:  DateTime<Utc>,
    pub updated_at:  DateTime<Utc>,
}

#[derive(Debug, Clone, FromRow)]
pub struct NodeRow {
    pub id:        i64,
    pub job_id:    i64,
    #[sqlx(rename = "type")]
    pub node_type: String,
    pub name:      String,
    pub xpos:      f32,
    pub ypos:      f32,
    pub data:      Option<Value>,
}

#[derive(Debug, Clone, FromRow)]
pub struct PortRow {
    pub id:                i64,
    pub node_id:           i64,
    #[sqlx(rename = "type")]
    pub port_type:         String,
    pub connected_node_id: i64,
}

#[derive(Debug, Clone, FromRow)]
pub struct JobUserAccess {
    pub job_id:  i64,
    pub user_id: i64,
    pub role:    String,
}

#[derive(Debug, Clone, FromRow)]
pub struct JobUserInfo {
    pub user_id: i64,
    pub email:   String,
    pub prenom:  String,
    pub nom:     String,
    pub role:    String,
}

#[derive(Debug, Clone, FromRow)]
pub struct UserBasic {
    pub id:     i64,
    pub email:  String,
    pub prenom: String,
    pub nom:    String,
}
