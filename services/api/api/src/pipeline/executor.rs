use std::sync::Arc;
use tokio::process::Command;

use crate::{pipeline::registry::JobRegistry, state::AppState};

/// Spawn a worker process for the given job_id.
/// The worker binary is the same executable invoked with --worker --job-id <id>.
pub async fn spawn_worker(job_id: i64, state: &AppState) -> Result<(), String> {
    if state.registry.is_running(job_id) {
        return Err(format!("job {job_id} is already running"));
    }

    let exe = std::env::current_exe()
        .map_err(|e| format!("cannot locate current executable: {e}"))?;

    let mut cmd = Command::new(&exe);
    cmd.arg("--worker")
        .arg("--job-id")
        .arg(job_id.to_string())
        .env("DATABASE_URL", &state.config.database_url)
        .env("TENANT_ID",    "default")
        .env("RUST_LOG",     "info");

    if let Some(url) = &state.config.nats_url {
        cmd.env("NATS_URL", url);
    }

    let child = cmd.kill_on_drop(false)
        .spawn()
        .map_err(|e| format!("spawn worker: {e}"))?;

    tracing::info!(job_id, pid = child.id(), "worker spawned");

    let registry = state.registry.clone();
    registry.insert(job_id, child);

    // Background monitor: clean up registry entry when the process exits
    let registry_monitor = state.registry.clone();
    tokio::spawn(async move {
        // Poll until the child is gone from the registry
        loop {
            tokio::time::sleep(std::time::Duration::from_secs(2)).await;
            if !registry_monitor.is_running(job_id) {
                break;
            }
        }
        registry_monitor.remove(job_id);
        tracing::info!(job_id, "worker process cleaned up from registry");
    });

    Ok(())
}
