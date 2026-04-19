use serde::Serialize;

#[derive(Debug, Clone, Serialize, serde::Deserialize)]
pub struct ProgressEvent {
    pub job_id:    i64,
    pub node_id:   i64,
    pub node_name: String,
    pub status:    String, // running | completed | failed
    pub row_count: u64,
    pub message:   String,
}

#[derive(Clone)]
pub struct ProgressReporter {
    job_id:    i64,
    nats:      Option<async_nats::Client>,
    tenant_id: String,
}

impl ProgressReporter {
    pub fn new(job_id: i64, nats: Option<async_nats::Client>, tenant_id: String) -> Self {
        Self { job_id, nats, tenant_id }
    }

    pub fn no_op(job_id: i64) -> Self {
        Self { job_id, nats: None, tenant_id: "default".into() }
    }

    pub async fn report(&self, node_id: i64, node_name: &str, status: &str, row_count: u64, message: &str) {
        let event = ProgressEvent {
            job_id:    self.job_id,
            node_id,
            node_name: node_name.into(),
            status:    status.into(),
            row_count,
            message:   message.into(),
        };

        if let Some(nats) = &self.nats {
            let subject = format!("tenant.{}.job.{}.progress", self.tenant_id, self.job_id);
            if let Ok(payload) = serde_json::to_vec(&event) {
                let _ = nats.publish(subject, payload.into()).await;
            }
        }
    }
}
