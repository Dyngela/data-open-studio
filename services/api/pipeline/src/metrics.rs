use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;

/// Lightweight in-process metrics updated by the worker.
/// Written to an mmap file periodically so the server can read them
/// even after a SIGKILL.
#[derive(Default)]
pub struct WorkerMetrics {
    pub rows_processed: AtomicU64,
}

impl WorkerMetrics {
    pub fn new() -> Arc<Self> {
        Arc::new(Self::default())
    }

    pub fn add_rows(&self, n: u64) {
        self.rows_processed.fetch_add(n, Ordering::Relaxed);
    }

    pub fn rows(&self) -> u64 {
        self.rows_processed.load(Ordering::Relaxed)
    }
}
