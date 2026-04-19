use sqlx::PgPool;

use crate::error::AppError;
use super::model::User;

pub async fn find_by_email(db: &PgPool, email: &str) -> Result<Option<User>, AppError> {
    sqlx::query_as(
        "SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL",
    )
    .bind(email)
    .fetch_optional(db)
    .await
    .map_err(AppError::from)
}

pub async fn find_by_id(db: &PgPool, id: i64) -> Result<Option<User>, AppError> {
    sqlx::query_as(
        "SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL",
    )
    .bind(id)
    .fetch_optional(db)
    .await
    .map_err(AppError::from)
}

pub async fn email_exists(db: &PgPool, email: &str) -> Result<bool, AppError> {
    let exists: bool = sqlx::query_scalar(
        "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)",
    )
    .bind(email)
    .fetch_one(db)
    .await?;
    Ok(exists)
}

pub async fn create(
    db: &PgPool,
    email: &str,
    password_hash: &str,
    prenom: &str,
    nom: &str,
) -> Result<User, AppError> {
    sqlx::query_as(
        "INSERT INTO users (email, password_hash, prenom, nom)
         VALUES ($1, $2, $3, $4)
         RETURNING *",
    )
    .bind(email)
    .bind(password_hash)
    .bind(prenom)
    .bind(nom)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn update_refresh_token(db: &PgPool, id: i64, token: &str) -> Result<(), AppError> {
    sqlx::query("UPDATE users SET refresh_token = $1, updated_at = now() WHERE id = $2")
        .bind(token)
        .bind(id)
        .execute(db)
        .await?;
    Ok(())
}

pub async fn search(db: &PgPool, q: &str) -> Result<Vec<User>, AppError> {
    let pattern = format!("%{q}%");
    sqlx::query_as(
        "SELECT * FROM users
         WHERE deleted_at IS NULL
           AND (email ILIKE $1 OR prenom ILIKE $1 OR nom ILIKE $1)
         LIMIT 10",
    )
    .bind(&pattern)
    .fetch_all(db)
    .await
    .map_err(AppError::from)
}
