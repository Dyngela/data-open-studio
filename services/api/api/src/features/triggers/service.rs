use serde_json::{json, Value};
use sqlx::PgPool;

use crate::error::AppError;
use super::{
    dto::*,
    model::{TriggerExecutionRow, TriggerRow, TriggerRuleRow},
    repository::{self, TriggerJobJoin},
};

// ---------------------------------------------------------------------------
// Serialisation helpers
// ---------------------------------------------------------------------------

fn trigger_to_json(t: &TriggerRow) -> Value {
    json!({
        "id":               t.id,
        "name":             t.name,
        "description":      t.description,
        "type":             t.trigger_type,
        "status":           t.status,
        "creator_id":       t.creator_id,
        "polling_interval": t.polling_interval,
        "last_polled_at":   t.last_polled_at,
        "last_error":       t.last_error,
        "created_at":       t.created_at,
        "updated_at":       t.updated_at,
    })
}

fn trigger_detail_to_json(
    t: &TriggerRow,
    rules: &[TriggerRuleRow],
    jobs: &[TriggerJobJoin],
) -> Value {
    let mut v = trigger_to_json(t);
    v["config"] = t.config.clone();
    v["rules"] = json!(rules.iter().map(rule_to_json).collect::<Vec<_>>());
    v["jobs"]  = json!(jobs.iter().map(job_link_to_json).collect::<Vec<_>>());
    v
}

fn rule_to_json(r: &TriggerRuleRow) -> Value {
    json!({
        "id":         r.id,
        "trigger_id": r.trigger_id,
        "name":       r.name,
        "conditions": r.conditions,
        "created_at": r.created_at,
        "updated_at": r.updated_at,
    })
}

fn job_link_to_json(j: &TriggerJobJoin) -> Value {
    json!({
        "id":              j.id,
        "trigger_id":      j.trigger_id,
        "job_id":          j.job_id,
        "job_name":        j.job_name,
        "priority":        j.priority,
        "active":          j.active,
        "pass_event_data": j.pass_event_data,
    })
}

fn check_access(t: &TriggerRow, user_id: i64) -> Result<(), AppError> {
    if t.creator_id != user_id {
        return Err(AppError::forbidden("access denied"));
    }
    Ok(())
}

async fn load_detail(
    db: &PgPool,
    t: &TriggerRow,
) -> Result<(Vec<TriggerRuleRow>, Vec<TriggerJobJoin>), AppError> {
    let rules = repository::get_rules(db, t.id).await?;
    let jobs = repository::get_jobs(db, t.id).await.unwrap_or_default();
    Ok((rules, jobs))
}

// ---------------------------------------------------------------------------
// Public service functions
// ---------------------------------------------------------------------------

pub async fn list(db: &PgPool, user_id: i64) -> Result<Value, AppError> {
    let rows = repository::list_by_creator(db, user_id).await.unwrap_or_default();
    Ok(json!({ "triggers": rows.iter().map(trigger_to_json).collect::<Vec<_>>() }))
}

pub async fn get_by_id(db: &PgPool, id: i64, user_id: i64) -> Result<Value, AppError> {
    let t = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("trigger not found"))?;
    check_access(&t, user_id)?;
    let (rules, jobs) = load_detail(db, &t).await?;
    Ok(trigger_detail_to_json(&t, &rules, &jobs))
}

pub async fn create(db: &PgPool, body: CreateTriggerReq, user_id: i64) -> Result<Value, AppError> {
    let valid_types = ["database", "email", "webhook", "cron"];
    if !valid_types.contains(&body.trigger_type.as_str()) {
        return Err(AppError::bad_request(format!(
            "type must be one of: {}",
            valid_types.join(", ")
        )));
    }

    let t = repository::create(
        db,
        &body.name,
        body.description.as_deref().unwrap_or(""),
        &body.trigger_type,
        user_id,
        body.polling_interval.unwrap_or(60),
        body.config.as_ref().unwrap_or(&serde_json::Value::Object(Default::default())),
    ).await?;

    Ok(trigger_detail_to_json(&t, &[], &[]))
}

pub async fn update(db: &PgPool, id: i64, body: UpdateTriggerReq, user_id: i64) -> Result<Value, AppError> {
    let t = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("trigger not found"))?;
    check_access(&t, user_id)?;

    let name             = body.name.as_deref().unwrap_or(&t.name).to_string();
    let description      = body.description.as_deref().unwrap_or(&t.description).to_string();
    let polling_interval = body.polling_interval.unwrap_or(t.polling_interval);
    let config           = body.config.as_ref().unwrap_or(&t.config).clone();

    let updated = repository::update(db, id, &name, &description, polling_interval, &config).await?;
    let (rules, jobs) = load_detail(db, &updated).await?;
    Ok(trigger_detail_to_json(&updated, &rules, &jobs))
}

pub async fn delete(db: &PgPool, id: i64, user_id: i64) -> Result<(), AppError> {
    let t = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("trigger not found"))?;
    check_access(&t, user_id)?;
    repository::soft_delete(db, id).await
}

pub async fn activate(db: &PgPool, id: i64, user_id: i64) -> Result<Value, AppError> {
    let t = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("trigger not found"))?;
    check_access(&t, user_id)?;
    let updated = repository::set_status(db, id, "active").await?;
    Ok(trigger_to_json(&updated))
}

pub async fn pause(db: &PgPool, id: i64, user_id: i64) -> Result<Value, AppError> {
    let t = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("trigger not found"))?;
    check_access(&t, user_id)?;
    let updated = repository::set_status(db, id, "paused").await?;
    Ok(trigger_to_json(&updated))
}

pub async fn add_rule(db: &PgPool, id: i64, body: CreateRuleReq, user_id: i64) -> Result<Value, AppError> {
    let t = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("trigger not found"))?;
    check_access(&t, user_id)?;
    let rule = repository::add_rule(db, id, body.name.as_deref().unwrap_or(""), &body.conditions).await?;
    Ok(rule_to_json(&rule))
}

pub async fn update_rule(
    db: &PgPool,
    id: i64,
    rule_id: i64,
    body: UpdateRuleReq,
    user_id: i64,
) -> Result<Value, AppError> {
    let t = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("trigger not found"))?;
    check_access(&t, user_id)?;

    let rule = repository::find_rule(db, rule_id, id)
        .await?
        .ok_or_else(|| AppError::not_found("rule not found"))?;

    let name       = body.name.as_deref().unwrap_or(&rule.name).to_string();
    let conditions = body.conditions.as_ref().unwrap_or(&rule.conditions).clone();

    let updated = repository::update_rule(db, rule_id, &name, &conditions).await?;
    Ok(rule_to_json(&updated))
}

pub async fn delete_rule(db: &PgPool, id: i64, rule_id: i64, user_id: i64) -> Result<(), AppError> {
    let t = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("trigger not found"))?;
    check_access(&t, user_id)?;
    repository::delete_rule(db, rule_id, id).await
}

pub async fn link_job(db: &PgPool, id: i64, body: LinkJobReq, user_id: i64) -> Result<Value, AppError> {
    let t = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("trigger not found"))?;
    check_access(&t, user_id)?;

    if !repository::job_exists(db, body.job_id).await? {
        return Err(AppError::not_found("job not found"));
    }

    let row = repository::link_job(
        db, id, body.job_id,
        body.priority.unwrap_or(0),
        body.pass_event_data.unwrap_or(false),
    ).await?;

    let job_name = repository::get_job_name(db, body.job_id).await?;

    Ok(job_link_to_json(&TriggerJobJoin {
        id:              row.id,
        trigger_id:      row.trigger_id,
        job_id:          row.job_id,
        priority:        row.priority,
        active:          row.active,
        pass_event_data: row.pass_event_data,
        job_name,
    }))
}

pub async fn unlink_job(db: &PgPool, id: i64, job_id: i64, user_id: i64) -> Result<(), AppError> {
    let t = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("trigger not found"))?;
    check_access(&t, user_id)?;
    repository::unlink_job(db, id, job_id).await
}

pub async fn get_executions(db: &PgPool, id: i64, params: ExecutionsQuery, user_id: i64) -> Result<Value, AppError> {
    let t = repository::find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("trigger not found"))?;
    check_access(&t, user_id)?;

    let limit = params.limit.unwrap_or(20).min(100);
    let rows = repository::get_executions(db, id, limit).await?;

    let executions: Vec<Value> = rows.iter().map(|e| json!({
        "id":             e.id,
        "trigger_id":     e.trigger_id,
        "started_at":     e.started_at,
        "finished_at":    e.finished_at,
        "status":         e.status,
        "event_count":    e.event_count,
        "jobs_triggered": e.jobs_triggered,
        "error":          e.error,
        "event_sample":   e.event_sample,
    })).collect();

    Ok(json!({ "executions": executions }))
}
