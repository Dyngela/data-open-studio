use dashmap::DashMap;
use std::sync::Arc;
use tokio::process::Child;

/// Tracks running worker processes by job_id.
pub struct JobRegistry {
    children: DashMap<i64, Child>,
}

impl JobRegistry {
    pub fn new() -> Arc<Self> {
        Arc::new(Self { children: DashMap::new() })
    }

    pub fn insert(&self, job_id: i64, child: Child) {
        self.children.insert(job_id, child);
    }

    /// Kill the worker process for a job. Returns true if it was found.
    pub async fn kill(&self, job_id: i64) -> bool {
        if let Some((_, mut child)) = self.children.remove(&job_id) {
            let _ = child.kill().await;
            return true;
        }
        false
    }

    pub fn remove(&self, job_id: i64) {
        self.children.remove(&job_id);
    }

    pub fn is_running(&self, job_id: i64) -> bool {
        self.children.contains_key(&job_id)
    }
}

impl Default for JobRegistry {
    fn default() -> Self {
        Self { children: DashMap::new() }
    }
}
