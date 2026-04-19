use sqlx::PgPool;

use crate::error::AppError;
use super::model::{TriggerExecutionRow, TriggerJobRow, TriggerRow, TriggerRuleRow};

pub async fn list_by_creator(db: &PgPool, user_id: i64) -> Result<Vec<TriggerRow>, AppError> {
    sqlx::query_as(
        "SELECT * FROM trigger WHERE creator_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC",
    )
    .bind(user_id)
    .fetch_all(db)
    .await
    .map_err(AppError::from)
}

pub async fn find_by_id(db: &PgPool, id: i64) -> Result<Option<TriggerRow>, AppError> {
    sqlx::query_as::<_, TriggerRow>(
        "SELECT * FROM trigger WHERE id = $1 AND deleted_at IS NULL",
    )
    .bind(id)
    .fetch_optional(db)
    .await
    .map_err(AppError::from)
}

pub async fn create(
    db: &PgPool,
    name: &str,
    description: &str,
    trigger_type: &str,
    creator_id: i64,
    polling_interval: i32,
    config: &serde_json::Value,
) -> Result<TriggerRow, AppError> {
    sqlx::query_as(
        "INSERT INTO trigger (name, description, type, creator_id, polling_interval, config)
         VALUES ($1, $2, $3, $4, $5, $6) RETURNING *",
    )
    .bind(name)
    .bind(description)
    .bind(trigger_type)
    .bind(creator_id)
    .bind(polling_interval)
    .bind(config)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn update(
    db: &PgPool,
    id: i64,
    name: &str,
    description: &str,
    polling_interval: i32,
    config: &serde_json::Value,
) -> Result<TriggerRow, AppError> {
    sqlx::query_as(
        "UPDATE trigger SET name=$1, description=$2, polling_interval=$3, config=$4, updated_at=now()
         WHERE id=$5 AND deleted_at IS NULL RETURNING *",
    )
    .bind(name)
    .bind(description)
    .bind(polling_interval)
    .bind(config)
    .bind(id)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn soft_delete(db: &PgPool, id: i64) -> Result<(), AppError> {
    sqlx::query("UPDATE trigger SET deleted_at = now() WHERE id = $1")
        .bind(id)
        .execute(db)
        .await?;
    Ok(())
}

pub async fn set_status(db: &PgPool, id: i64, status: &str) -> Result<TriggerRow, AppError> {
    if status == "active" {
        sqlx::query_as(
            "UPDATE trigger SET status='active', last_error='', updated_at=now() WHERE id=$1 RETURNING *",
        )
        .bind(id)
        .fetch_one(db)
        .await
        .map_err(AppError::from)
    } else {
        sqlx::query_as(
            "UPDATE trigger SET status='paused', updated_at=now() WHERE id=$1 RETURNING *",
        )
        .bind(id)
        .fetch_one(db)
        .await
        .map_err(AppError::from)
    }
}

pub async fn get_rules(db: &PgPool, trigger_id: i64) -> Result<Vec<TriggerRuleRow>, AppError> {
    sqlx::query_as(
        "SELECT * FROM trigger_rule WHERE trigger_id = $1 AND deleted_at IS NULL ORDER BY created_at ASC",
    )
    .bind(trigger_id)
    .fetch_all(db)
    .await
    .map_err(AppError::from)
}

pub async fn add_rule(
    db: &PgPool,
    trigger_id: i64,
    name: &str,
    conditions: &serde_json::Value,
) -> Result<TriggerRuleRow, AppError> {
    sqlx::query_as(
        "INSERT INTO trigger_rule (trigger_id, name, conditions) VALUES ($1, $2, $3) RETURNING *",
    )
    .bind(trigger_id)
    .bind(name)
    .bind(conditions)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn find_rule(db: &PgPool, rule_id: i64, trigger_id: i64) -> Result<Option<TriggerRuleRow>, AppError> {
    sqlx::query_as(
        "SELECT * FROM trigger_rule WHERE id = $1 AND trigger_id = $2 AND deleted_at IS NULL",
    )
    .bind(rule_id)
    .bind(trigger_id)
    .fetch_optional(db)
    .await
    .map_err(AppError::from)
}

pub async fn update_rule(
    db: &PgPool,
    rule_id: i64,
    name: &str,
    conditions: &serde_json::Value,
) -> Result<TriggerRuleRow, AppError> {
    sqlx::query_as(
        "UPDATE trigger_rule SET name=$1, conditions=$2, updated_at=now() WHERE id=$3 RETURNING *",
    )
    .bind(name)
    .bind(conditions)
    .bind(rule_id)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn delete_rule(db: &PgPool, rule_id: i64, trigger_id: i64) -> Result<(), AppError> {
    sqlx::query(
        "UPDATE trigger_rule SET deleted_at=now() WHERE id=$1 AND trigger_id=$2",
    )
    .bind(rule_id)
    .bind(trigger_id)
    .execute(db)
    .await?;
    Ok(())
}

#[derive(sqlx::FromRow)]
pub struct TriggerJobJoin {
    pub id:              i64,
    pub trigger_id:      i64,
    pub job_id:          i64,
    pub priority:        i32,
    pub active:          bool,
    pub pass_event_data: bool,
    pub job_name:        String,
}

pub async fn get_jobs(db: &PgPool, trigger_id: i64) -> Result<Vec<TriggerJobJoin>, AppError> {
    sqlx::query_as(
        "SELECT tj.id, tj.trigger_id, tj.job_id, tj.priority, tj.active, tj.pass_event_data,
                j.name AS job_name
         FROM trigger_job tj
         JOIN job j ON j.id = tj.job_id
         WHERE tj.trigger_id = $1 AND tj.deleted_at IS NULL
         ORDER BY tj.priority ASC, tj.created_at ASC",
    )
    .bind(trigger_id)
    .fetch_all(db)
    .await
    .map_err(AppError::from)
}

pub async fn link_job(
    db: &PgPool,
    trigger_id: i64,
    job_id: i64,
    priority: i32,
    pass_event_data: bool,
) -> Result<TriggerJobRow, AppError> {
    sqlx::query_as(
        "INSERT INTO trigger_job (trigger_id, job_id, priority, pass_event_data)
         VALUES ($1, $2, $3, $4) RETURNING *",
    )
    .bind(trigger_id)
    .bind(job_id)
    .bind(priority)
    .bind(pass_event_data)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn job_exists(db: &PgPool, job_id: i64) -> Result<bool, AppError> {
    let exists: bool = sqlx::query_scalar("SELECT EXISTS(SELECT 1 FROM job WHERE id = $1)")
        .bind(job_id)
        .fetch_one(db)
        .await?;
    Ok(exists)
}

pub async fn get_job_name(db: &PgPool, job_id: i64) -> Result<String, AppError> {
    sqlx::query_scalar("SELECT name FROM job WHERE id = $1")
        .bind(job_id)
        .fetch_one(db)
        .await
        .map_err(AppError::from)
}

pub async fn unlink_job(db: &PgPool, trigger_id: i64, job_id: i64) -> Result<(), AppError> {
    sqlx::query(
        "UPDATE trigger_job SET deleted_at=now() WHERE trigger_id=$1 AND job_id=$2",
    )
    .bind(trigger_id)
    .bind(job_id)
    .execute(db)
    .await?;
    Ok(())
}

pub async fn get_executions(
    db: &PgPool,
    trigger_id: i64,
    limit: i64,
) -> Result<Vec<TriggerExecutionRow>, AppError> {
    sqlx::query_as(
        "SELECT * FROM trigger_execution WHERE trigger_id = $1 ORDER BY started_at DESC LIMIT $2",
    )
    .bind(trigger_id)
    .bind(limit)
    .fetch_all(db)
    .await
    .map_err(AppError::from)
}
