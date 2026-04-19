use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;

use super::model::Job;

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateJobRequest {
    pub name:        String,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub file_path:   String,
    #[serde(default)]
    pub output_path: String,
    #[serde(default = "default_true")]
    pub active:      bool,
    #[serde(default = "default_private")]
    pub visibility:  String,
    #[serde(default)]
    pub shared_with: Vec<i64>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateJobRequest {
    pub name:        Option<String>,
    pub description: Option<String>,
    pub file_path:   Option<String>,
    pub output_path: Option<String>,
    pub active:      Option<bool>,
    pub visibility:  Option<String>,
    pub shared_with: Option<Vec<i64>>,
    pub nodes:       Option<Vec<NodeInput>>,
    #[serde(default)]
    pub connexions:  Vec<Connexion>,
}

#[derive(Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct NodeInput {
    pub id:        i64,
    pub r#type:    String,
    pub name:      String,
    pub xpos:      f32,
    pub ypos:      f32,
    pub data:      Value,
}

#[derive(Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
pub struct Connexion {
    pub source_node_id:   i64,
    pub source_port:      i32,
    pub source_port_type: String,
    pub target_node_id:   i64,
    pub target_port:      i32,
    pub target_port_type: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ShareRequest {
    pub user_ids: Vec<i64>,
    #[serde(default = "default_viewer")]
    pub role:     String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AddNotificationContactRequest {
    pub user_id: i64,
}

#[derive(Deserialize)]
pub struct ListQuery {
    #[serde(rename = "filePath")]
    pub file_path: Option<String>,
}

fn default_true()    -> bool   { true }
fn default_private() -> String { "private".into() }
fn default_viewer()  -> String { "viewer".into() }

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct JobResponse {
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

impl From<Job> for JobResponse {
    fn from(j: Job) -> Self {
        Self {
            id: j.id, name: j.name, description: j.description,
            file_path: j.file_path, creator_id: j.creator_id,
            active: j.active, visibility: j.visibility,
            output_path: j.output_path,
            created_at: j.created_at, updated_at: j.updated_at,
        }
    }
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct JobWithNodesResponse {
    pub id:                    i64,
    pub name:                  String,
    pub description:           String,
    pub file_path:             String,
    pub creator_id:            i64,
    pub active:                bool,
    pub visibility:            String,
    pub output_path:           String,
    pub created_at:            DateTime<Utc>,
    pub updated_at:            DateTime<Utc>,
    pub nodes:                 Vec<NodeResponse>,
    pub connexions:            Vec<ConnexionResponse>,
    pub shared_user:           Vec<SharedUserResponse>,
    pub notification_contacts: Vec<UserBasicResponse>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct NodeResponse {
    pub id:      i64,
    pub r#type:  String,
    pub name:    String,
    pub xpos:    f32,
    pub ypos:    f32,
    pub data:    Value,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ConnexionResponse {
    pub source_node_id:   i64,
    pub source_port:      i32,
    pub source_port_type: String,
    pub target_node_id:   i64,
    pub target_port:      i32,
    pub target_port_type: String,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct SharedUserResponse {
    pub id:     i64,
    pub email:  String,
    pub prenom: String,
    pub nom:    String,
    pub role:   String,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct UserBasicResponse {
    pub id:     i64,
    pub email:  String,
    pub prenom: String,
    pub nom:    String,
}
