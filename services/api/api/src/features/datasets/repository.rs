use chrono::{DateTime, Utc};
use serde_json::Value;
use sqlx::PgPool;

use crate::error::AppError;
use super::model::DatasetRow;

pub async fn list_by_creator(db: &PgPool, user_id: i64) -> Result<Vec<DatasetRow>, AppError> {
    sqlx::query_as(
        "SELECT * FROM dataset WHERE creator_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC",
    )
    .bind(user_id)
    .fetch_all(db)
    .await
    .map_err(AppError::from)
}

pub async fn find_by_id(db: &PgPool, id: i64) -> Result<Option<DatasetRow>, AppError> {
    sqlx::query_as::<_, DatasetRow>(
        "SELECT * FROM dataset WHERE id = $1 AND deleted_at IS NULL",
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
    creator_id: i64,
    metadata_database_id: i64,
    query: &str,
    schema: &Value,
    status: &str,
    last_refreshed_at: Option<DateTime<Utc>>,
    last_error: &str,
) -> Result<DatasetRow, AppError> {
    sqlx::query_as(
        "INSERT INTO dataset
            (name, description, creator_id, metadata_database_id, query,
             schema, status, last_refreshed_at, last_error)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
         RETURNING *",
    )
    .bind(name)
    .bind(description)
    .bind(creator_id)
    .bind(metadata_database_id)
    .bind(query)
    .bind(schema)
    .bind(status)
    .bind(last_refreshed_at)
    .bind(last_error)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn update(
    db: &PgPool,
    id: i64,
    name: &str,
    description: &str,
    query: &str,
    metadata_database_id: i64,
    schema: &Value,
    status: &str,
    last_refreshed_at: Option<DateTime<Utc>>,
    last_error: &str,
) -> Result<DatasetRow, AppError> {
    sqlx::query_as(
        "UPDATE dataset SET
            name = $1, description = $2, query = $3, metadata_database_id = $4,
            schema = $5, status = $6, last_refreshed_at = $7, last_error = $8,
            updated_at = now()
         WHERE id = $9 AND deleted_at IS NULL
         RETURNING *",
    )
    .bind(name)
    .bind(description)
    .bind(query)
    .bind(metadata_database_id)
    .bind(schema)
    .bind(status)
    .bind(last_refreshed_at)
    .bind(last_error)
    .bind(id)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn soft_delete(db: &PgPool, id: i64) -> Result<(), AppError> {
    sqlx::query("UPDATE dataset SET deleted_at = now() WHERE id = $1")
        .bind(id)
        .execute(db)
        .await?;
    Ok(())
}
