use std::sync::Arc;
use tokio::{sync::Semaphore, time};

use crate::{
    features::triggers::model::TriggerRow,
    pipeline,
    state::AppState,
};

const TICK_SECS:         u64 = 10;
const MAX_CONCURRENT:    usize = 10;

pub async fn run(state: AppState) {
    let semaphore = Arc::new(Semaphore::new(MAX_CONCURRENT));
    let mut interval = time::interval(time::Duration::from_secs(TICK_SECS));

    tracing::info!("trigger poller started");

    loop {
        interval.tick().await;
        dispatch_due_triggers(&state, semaphore.clone()).await;
    }
}

async fn dispatch_due_triggers(state: &AppState, semaphore: Arc<Semaphore>) {
    let triggers: Vec<TriggerRow> = sqlx::query_as(
        "SELECT * FROM trigger WHERE status = 'active' AND deleted_at IS NULL",
    )
    .fetch_all(&state.db)
    .await
    .unwrap_or_default();

    let now = chrono::Utc::now();

    for trigger in triggers {
        if !is_due(&trigger, now) {
            continue;
        }

        let permit = match semaphore.clone().try_acquire_owned() {
            Ok(p)  => p,
            Err(_) => {
                tracing::debug!("trigger poller: semaphore full, skipping trigger {}", trigger.id);
                continue;
            }
        };

        let state_clone = state.clone();
        tokio::spawn(async move {
            let _permit = permit; // dropped when task ends
            poll_trigger(state_clone, trigger).await;
        });
    }
}

fn is_due(trigger: &TriggerRow, now: chrono::DateTime<chrono::Utc>) -> bool {
    match trigger.trigger_type.as_str() {
        "cron" => is_cron_due(trigger, now),
        _      => is_interval_due(trigger, now),
    }
}

fn is_interval_due(trigger: &TriggerRow, now: chrono::DateTime<chrono::Utc>) -> bool {
    let interval_secs = trigger.polling_interval as i64;
    match trigger.last_polled_at {
        None       => true,
        Some(last) => (now - last).num_seconds() >= interval_secs,
    }
}

fn is_cron_due(trigger: &TriggerRow, now: chrono::DateTime<chrono::Utc>) -> bool {
    // Read cron config from trigger.config JSONB
    let mode = trigger.config.get("cron")
        .and_then(|c: &serde_json::Value| c.get("mode"))
        .and_then(|m: &serde_json::Value| m.as_str())
        .unwrap_or("interval");

    match mode {
        "interval" => {
            let interval_value = trigger.config
                .get("cron").and_then(|c: &serde_json::Value| c.get("intervalValue"))
                .and_then(|v: &serde_json::Value| v.as_i64()).unwrap_or(60);
            let unit = trigger.config
                .get("cron").and_then(|c: &serde_json::Value| c.get("intervalUnit"))
                .and_then(|u: &serde_json::Value| u.as_str()).unwrap_or("seconds");
            let secs = match unit {
                "minutes" => interval_value * 60,
                "hours"   => interval_value * 3600,
                "days"    => interval_value * 86400,
                _         => interval_value,
            };
            match trigger.last_polled_at {
                None       => true,
                Some(last) => (now - last).num_seconds() >= secs,
            }
        }
        "schedule" => {
            // For schedule mode: check if we passed the scheduled time since last poll
            let time_str = trigger.config
                .get("cron").and_then(|c: &serde_json::Value| c.get("scheduleTime"))
                .and_then(|t: &serde_json::Value| t.as_str()).unwrap_or("00:00");
            let parts: Vec<&str> = time_str.split(':').collect();
            let (h, m) = if parts.len() >= 2 {
                (parts[0].parse::<u32>().unwrap_or(0), parts[1].parse::<u32>().unwrap_or(0))
            } else {
                (0, 0)
            };
            let today_fire = now.date_naive().and_hms_opt(h, m, 0)
                .map(|dt| dt.and_utc())
                .unwrap_or(now);
            match trigger.last_polled_at {
                None       => now >= today_fire,
                Some(last) => now >= today_fire && last < today_fire,
            }
        }
        _ => is_interval_due(trigger, now),
    }
}

async fn poll_trigger(state: AppState, trigger: TriggerRow) {
    tracing::info!(trigger_id = trigger.id, trigger_type = %trigger.trigger_type, "polling trigger");

    // Mark as polled immediately to avoid double-dispatch
    let _ = sqlx::query(
        "UPDATE trigger SET last_polled_at = now() WHERE id = $1",
    )
    .bind(trigger.id)
    .execute(&state.db)
    .await;

    match trigger.trigger_type.as_str() {
        "cron" => poll_cron(&state, &trigger).await,
        _      => {
            tracing::debug!(trigger_id = trigger.id, "trigger type '{}' polling not yet implemented", trigger.trigger_type);
        }
    }
}

async fn poll_cron(state: &AppState, trigger: &TriggerRow) {
    // Load linked active jobs
    let job_ids: Vec<i64> = sqlx::query_scalar(
        "SELECT job_id FROM trigger_job WHERE trigger_id = $1 AND active = true AND deleted_at IS NULL ORDER BY priority ASC",
    )
    .bind(trigger.id)
    .fetch_all(&state.db)
    .await
    .unwrap_or_default();

    if job_ids.is_empty() {
        return;
    }

    // Record execution start
    let exec_id: i64 = sqlx::query_scalar(
        "INSERT INTO trigger_execution (trigger_id, started_at, status, event_count)
         VALUES ($1, now(), 'running', 1)
         RETURNING id",
    )
    .bind(trigger.id)
    .fetch_one(&state.db)
    .await
    .unwrap_or(0);

    let mut triggered = 0i32;
    for job_id in &job_ids {
        match pipeline::executor::spawn_worker(*job_id, state).await {
            Ok(())  => triggered += 1,
            Err(e)  => tracing::warn!(trigger_id = trigger.id, job_id, "spawn failed: {e}"),
        }
    }

    let _ = sqlx::query(
        "UPDATE trigger_execution SET finished_at = now(), status = 'completed', jobs_triggered = $1
         WHERE id = $2",
    )
    .bind(triggered)
    .bind(exec_id)
    .execute(&state.db)
    .await;
}
