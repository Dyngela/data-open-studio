use std::collections::HashMap;
use std::sync::{Arc, RwLock};

use df_store::frame::Frame;
use sqlx::PgPool;
use uuid::Uuid;

/// In-memory frames for a single workspace.
#[derive(Default)]
pub struct WorkspaceFrames {
    pub frames: HashMap<String, Frame>,
}

/// Shared application state threaded through every Axum handler.
#[derive(Clone)]
pub struct AppState {
    pub db: PgPool,
    /// workspace_id → loaded frames
    pub workspaces: Arc<RwLock<HashMap<Uuid, WorkspaceFrames>>>,
}

impl AppState {
    pub fn new(db: PgPool) -> Self {
        Self {
            db,
            workspaces: Arc::new(RwLock::new(HashMap::new())),
        }
    }
}
