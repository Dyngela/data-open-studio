use sqlx::PgPool;
use uuid::Uuid;

use crate::error::AppError;
use super::model::Workspace;

pub async fn list(db: &PgPool) -> Result<Vec<Workspace>, AppError> {
    sqlx::query_as("SELECT * FROM workspace ORDER BY created_at DESC")
        .fetch_all(db)
        .await
        .map_err(AppError::from)
}

pub async fn find_by_id(db: &PgPool, id: Uuid) -> Result<Option<Workspace>, AppError> {
    sqlx::query_as("SELECT * FROM workspace WHERE id = $1")
        .bind(id)
        .fetch_optional(db)
        .await
        .map_err(AppError::from)
}

pub async fn create(db: &PgPool, name: &str) -> Result<Workspace, AppError> {
    sqlx::query_as(
        "INSERT INTO workspace (id, name) VALUES (gen_random_uuid(), $1) RETURNING *",
    )
    .bind(name)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn update(db: &PgPool, id: Uuid, name: &str) -> Result<Option<Workspace>, AppError> {
    sqlx::query_as(
        "UPDATE workspace SET name = $1 WHERE id = $2 RETURNING *",
    )
    .bind(name)
    .bind(id)
    .fetch_optional(db)
    .await
    .map_err(AppError::from)
}

pub async fn delete(db: &PgPool, id: Uuid) -> Result<u64, AppError> {
    let result = sqlx::query("DELETE FROM workspace WHERE id = $1")
        .bind(id)
        .execute(db)
        .await?;
    Ok(result.rows_affected())
}
