use dashmap::DashMap;
use std::sync::Arc;
use tokio::sync::broadcast;

const CHANNEL_CAPACITY: usize = 256;

pub struct Hub {
    channels: DashMap<i64, broadcast::Sender<String>>,
}

impl Hub {
    pub fn new() -> Arc<Self> {
        Arc::new(Self { channels: DashMap::new() })
    }

    pub fn subscribe(&self, job_id: i64) -> broadcast::Receiver<String> {
        if let Some(tx) = self.channels.get(&job_id) {
            return tx.subscribe();
        }
        let (tx, rx) = broadcast::channel(CHANNEL_CAPACITY);
        self.channels.insert(job_id, tx);
        rx
    }

    pub fn broadcast(&self, job_id: i64, msg: String) {
        if let Some(tx) = self.channels.get(&job_id) {
            let _ = tx.send(msg);
        }
    }

    pub fn close(&self, job_id: i64) {
        self.channels.remove(&job_id);
    }
}

impl Default for Hub {
    fn default() -> Self {
        Self { channels: DashMap::new() }
    }
}
