use sqlx::PgPool;

use crate::error::AppError;
use super::model::{MetadataDatabase, MetadataEmail, MetadataSftp};

// ---------------------------------------------------------------------------
// Database
// ---------------------------------------------------------------------------

pub async fn db_list(db: &PgPool) -> Result<Vec<MetadataDatabase>, AppError> {
    sqlx::query_as("SELECT * FROM metadata_database ORDER BY id")
        .fetch_all(db)
        .await
        .map_err(AppError::from)
}

pub async fn db_find_by_id(db: &PgPool, id: i64) -> Result<Option<MetadataDatabase>, AppError> {
    sqlx::query_as("SELECT * FROM metadata_database WHERE id = $1")
        .bind(id)
        .fetch_optional(db)
        .await
        .map_err(AppError::from)
}

pub async fn db_create(
    db: &PgPool,
    name: &str,
    host: &str,
    port: i32,
    user: &str,
    password: &str,
    database_name: &str,
    ssl_mode: &str,
    extra: &str,
    db_type: &str,
) -> Result<MetadataDatabase, AppError> {
    sqlx::query_as(
        "INSERT INTO metadata_database (name, host, port, \"user\", password, database_name, ssl_mode, extra, db_type)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING *",
    )
    .bind(name)
    .bind(host)
    .bind(port)
    .bind(user)
    .bind(password)
    .bind(database_name)
    .bind(ssl_mode)
    .bind(extra)
    .bind(db_type)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn db_update(
    db: &PgPool,
    id: i64,
    name: &str,
    host: &str,
    port: i32,
    user: &str,
    password: &str,
    database_name: &str,
    ssl_mode: &str,
    extra: &str,
    db_type: &str,
) -> Result<MetadataDatabase, AppError> {
    sqlx::query_as(
        "UPDATE metadata_database
         SET name=$2, host=$3, port=$4, \"user\"=$5, password=$6,
             database_name=$7, ssl_mode=$8, extra=$9, db_type=$10
         WHERE id=$1 RETURNING *",
    )
    .bind(id)
    .bind(name)
    .bind(host)
    .bind(port)
    .bind(user)
    .bind(password)
    .bind(database_name)
    .bind(ssl_mode)
    .bind(extra)
    .bind(db_type)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn db_delete(db: &PgPool, id: i64) -> Result<u64, AppError> {
    let result = sqlx::query("DELETE FROM metadata_database WHERE id = $1")
        .bind(id)
        .execute(db)
        .await?;
    Ok(result.rows_affected())
}

// ---------------------------------------------------------------------------
// SFTP
// ---------------------------------------------------------------------------

pub async fn sftp_list(db: &PgPool) -> Result<Vec<MetadataSftp>, AppError> {
    sqlx::query_as("SELECT * FROM metadata_sftp ORDER BY id")
        .fetch_all(db)
        .await
        .map_err(AppError::from)
}

pub async fn sftp_find_by_id(db: &PgPool, id: i64) -> Result<Option<MetadataSftp>, AppError> {
    sqlx::query_as("SELECT * FROM metadata_sftp WHERE id = $1")
        .bind(id)
        .fetch_optional(db)
        .await
        .map_err(AppError::from)
}

pub async fn sftp_create(
    db: &PgPool,
    name: &str,
    host: &str,
    port: i32,
    user: &str,
    password: &str,
    private_key: &str,
    base_path: &str,
    extra: &str,
) -> Result<MetadataSftp, AppError> {
    sqlx::query_as(
        "INSERT INTO metadata_sftp (name, host, port, \"user\", password, private_key, base_path, extra)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING *",
    )
    .bind(name)
    .bind(host)
    .bind(port)
    .bind(user)
    .bind(password)
    .bind(private_key)
    .bind(base_path)
    .bind(extra)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn sftp_update(
    db: &PgPool,
    id: i64,
    name: &str,
    host: &str,
    port: i32,
    user: &str,
    password: &str,
    private_key: &str,
    base_path: &str,
    extra: &str,
) -> Result<MetadataSftp, AppError> {
    sqlx::query_as(
        "UPDATE metadata_sftp
         SET name=$2, host=$3, port=$4, \"user\"=$5, password=$6, private_key=$7, base_path=$8, extra=$9
         WHERE id=$1 RETURNING *",
    )
    .bind(id)
    .bind(name)
    .bind(host)
    .bind(port)
    .bind(user)
    .bind(password)
    .bind(private_key)
    .bind(base_path)
    .bind(extra)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn sftp_delete(db: &PgPool, id: i64) -> Result<u64, AppError> {
    let result = sqlx::query("DELETE FROM metadata_sftp WHERE id = $1")
        .bind(id)
        .execute(db)
        .await?;
    Ok(result.rows_affected())
}

// ---------------------------------------------------------------------------
// Email
// ---------------------------------------------------------------------------

pub async fn email_list(db: &PgPool) -> Result<Vec<MetadataEmail>, AppError> {
    sqlx::query_as("SELECT * FROM metadata_email ORDER BY id")
        .fetch_all(db)
        .await
        .map_err(AppError::from)
}

pub async fn email_find_by_id(db: &PgPool, id: i64) -> Result<Option<MetadataEmail>, AppError> {
    sqlx::query_as("SELECT * FROM metadata_email WHERE id = $1")
        .bind(id)
        .fetch_optional(db)
        .await
        .map_err(AppError::from)
}

pub async fn email_create(
    db: &PgPool,
    name: &str,
    imap_host: &str,
    imap_port: i32,
    smtp_host: &str,
    smtp_port: i32,
    username: &str,
    password: &str,
    use_tls: bool,
) -> Result<MetadataEmail, AppError> {
    sqlx::query_as(
        "INSERT INTO metadata_email (name, imap_host, imap_port, smtp_host, smtp_port, username, password, use_tls)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING *",
    )
    .bind(name)
    .bind(imap_host)
    .bind(imap_port)
    .bind(smtp_host)
    .bind(smtp_port)
    .bind(username)
    .bind(password)
    .bind(use_tls)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn email_update(
    db: &PgPool,
    id: i64,
    name: &str,
    imap_host: &str,
    imap_port: i32,
    smtp_host: &str,
    smtp_port: i32,
    username: &str,
    password: &str,
    use_tls: bool,
) -> Result<MetadataEmail, AppError> {
    sqlx::query_as(
        "UPDATE metadata_email
         SET name=$2, imap_host=$3, imap_port=$4, smtp_host=$5, smtp_port=$6,
             username=$7, password=$8, use_tls=$9
         WHERE id=$1 RETURNING *",
    )
    .bind(id)
    .bind(name)
    .bind(imap_host)
    .bind(imap_port)
    .bind(smtp_host)
    .bind(smtp_port)
    .bind(username)
    .bind(password)
    .bind(use_tls)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn email_delete(db: &PgPool, id: i64) -> Result<u64, AppError> {
    let result = sqlx::query("DELETE FROM metadata_email WHERE id = $1")
        .bind(id)
        .execute(db)
        .await?;
    Ok(result.rows_affected())
}
