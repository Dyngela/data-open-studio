use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use utoipa::ToSchema;
use uuid::Uuid;

use super::model::Workspace;

#[derive(Deserialize, ToSchema)]
pub struct CreateWorkspaceRequest {
    pub name: String,
}

#[derive(Deserialize, ToSchema)]
pub struct UpdateWorkspaceRequest {
    pub name: String,
}

#[derive(Serialize, ToSchema)]
pub struct WorkspaceResponse {
    pub id:         Uuid,
    pub name:       String,
    pub created_at: DateTime<Utc>,
}

impl From<Workspace> for WorkspaceResponse {
    fn from(w: Workspace) -> Self {
        Self {
            id:         w.id,
            name:       w.name,
            created_at: w.created_at,
        }
    }
}
