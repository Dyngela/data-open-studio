use std::sync::Arc;

use futures_util::StreamExt;
use pipeline::progress::ProgressEvent;
use serde_json::json;

use crate::websocket::hub::Hub;

/// Subscribes to NATS `tenant.*.job.*.progress` and forwards messages to the Hub.
/// No-op if NATS URL is not configured.
pub async fn run(hub: Arc<Hub>, nats_url: Option<String>) {
    let Some(url) = nats_url else {
        tracing::info!("NATS not configured, skipping WebSocket bridge");
        return;
    };

    let client = match async_nats::connect(&url).await {
        Ok(c) => c,
        Err(e) => {
            tracing::error!("NATS connect failed: {e}");
            return;
        }
    };

    let mut sub = match client.subscribe("tenant.*.job.*.progress").await {
        Ok(s) => s,
        Err(e) => {
            tracing::error!("NATS subscribe failed: {e}");
            return;
        }
    };

    tracing::info!("NATS bridge running on {url}");

    while let Some(msg) = sub.next().await {
        let job_id = parse_job_id(msg.subject.as_str());
        if let Some(id) = job_id {
            if let Ok(event) = serde_json::from_slice::<ProgressEvent>(&msg.payload) {
                let envelope = json!({
                    "type": "job.progress",
                    "jobId": event.job_id,
                    "payload": {
                        "nodeId":   event.node_id,
                        "nodeName": event.node_name,
                        "status":   event.status,
                        "rowCount": event.row_count,
                        "message":  event.message
                    }
                });
                hub.broadcast(id, envelope.to_string());
            }
        }
    }
}

/// Extracts job_id from `tenant.<tid>.job.<jid>.progress`.
fn parse_job_id(subject: &str) -> Option<i64> {
    let parts: Vec<&str> = subject.split('.').collect();
    if parts.len() >= 5 && parts[2] == "job" {
        parts[3].parse().ok()
    } else {
        None
    }
}
