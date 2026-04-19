#[derive(Debug, Clone)]
pub struct WorkerConfig {
    pub job_id:       i64,
    pub database_url: String,
    pub nats_url:     Option<String>,
    pub tenant_id:    String,
    pub metrics_path: String,
}

impl WorkerConfig {
    pub fn from_env(job_id: i64) -> Self {
        Self {
            job_id,
            database_url: std::env::var("DATABASE_URL")
                .expect("DATABASE_URL required for worker"),
            nats_url: std::env::var("NATS_URL").ok(),
            tenant_id: std::env::var("TENANT_ID").unwrap_or_else(|_| "default".into()),
            metrics_path: format!("/tmp/job-{job_id}.metrics"),
        }
    }
}

impl std::fmt::Display for WorkerConfig {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        // Redact password from database URL before logging
        let db = redact_url(&self.database_url);
        write!(f,
            "WorkerConfig {{ job_id: {}, database_url: {}, nats_url: {:?}, tenant_id: {}, metrics_path: {} }}",
            self.job_id, db, self.nats_url, self.tenant_id, self.metrics_path
        )
    }
}

fn redact_url(url: &str) -> String {
    // postgres://user:password@host/db → postgres://user:***@host/db
    if let Some(at) = url.rfind('@') {
        if let Some(colon) = url[..at].rfind(':') {
            if url[..colon].contains("://") {
                return format!("{}:***@{}", &url[..colon], &url[at + 1..]);
            }
        }
    }
    url.to_string()
}
