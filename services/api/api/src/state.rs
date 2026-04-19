use std::collections::HashMap;
use std::sync::{Arc, RwLock};

use df_store::frame::Frame;
use sqlx::PgPool;
use uuid::Uuid;

use crate::config::AppConfig;
use crate::pipeline::registry::JobRegistry;
use crate::websocket::hub::Hub;

#[derive(Default)]
pub struct WorkspaceFrames {
    pub frames: HashMap<String, Frame>,
}

#[derive(Clone)]
pub struct AppState {
    pub db:         PgPool,
    pub config:     AppConfig,
    pub workspaces: Arc<RwLock<HashMap<Uuid, WorkspaceFrames>>>,
    pub hub:        Arc<Hub>,
    pub registry:   Arc<JobRegistry>,
}

impl AppState {
    pub fn new(db: PgPool, config: AppConfig) -> Self {
        Self {
            db,
            config,
            workspaces: Arc::new(RwLock::new(HashMap::new())),
            hub:        Hub::new(),
            registry:   JobRegistry::new(),
        }
    }
}
