use serde_json::{json, Value};
use sqlx::PgPool;

use crate::{
    crypto::{decrypt_secret, encrypt_secret, MASKED},
    error::AppError,
};
use super::{
    dto::*,
    model::{MetadataDatabase, MetadataEmail, MetadataSftp},
    repository,
};

// ---------------------------------------------------------------------------
// Public helper re-exported for datasets feature
// ---------------------------------------------------------------------------

pub fn dec(key: &[u8; 32], stored: &str) -> Result<String, AppError> {
    decrypt_secret(key, stored).map_err(|e| AppError::internal(format!("decrypt credential: {e}")))
}

fn enc(key: &[u8; 32], val: &str) -> Result<String, AppError> {
    encrypt_secret(key, val).map_err(|e| AppError::internal(format!("encrypt credential: {e}")))
}

// ---------------------------------------------------------------------------
// Response masking
// ---------------------------------------------------------------------------

pub fn mask_db(row: &MetadataDatabase) -> Value {
    json!({
        "id":            row.id,
        "name":          row.name,
        "host":          row.host,
        "port":          row.port,
        "user":          row.user,
        "password":      MASKED,
        "database_name": row.database_name,
        "ssl_mode":      row.ssl_mode,
        "extra":         row.extra,
        "db_type":       row.db_type,
    })
}

pub fn mask_sftp(row: &MetadataSftp) -> Value {
    json!({
        "id":          row.id,
        "name":        row.name,
        "host":        row.host,
        "port":        row.port,
        "user":        row.user,
        "password":    MASKED,
        "private_key": MASKED,
        "base_path":   row.base_path,
        "extra":       row.extra,
    })
}

pub fn mask_email(row: &MetadataEmail) -> Value {
    json!({
        "id":        row.id,
        "name":      row.name,
        "imap_host": row.imap_host,
        "imap_port": row.imap_port,
        "smtp_host": row.smtp_host,
        "smtp_port": row.smtp_port,
        "username":  row.username,
        "password":  MASKED,
        "use_tls":   row.use_tls,
    })
}

// ---------------------------------------------------------------------------
// Database service functions
// ---------------------------------------------------------------------------

pub async fn db_list(db: &PgPool) -> Result<Value, AppError> {
    let rows = repository::db_list(db).await?;
    Ok(json!({ "items": rows.iter().map(mask_db).collect::<Vec<_>>() }))
}

pub async fn db_get(db: &PgPool, id: i64) -> Result<Value, AppError> {
    let row = repository::db_find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("database metadata not found"))?;
    Ok(mask_db(&row))
}

pub async fn db_create(
    db: &PgPool,
    key: &[u8; 32],
    body: CreateDbMetadata,
) -> Result<Value, AppError> {
    let row = repository::db_create(
        db,
        &body.name.unwrap_or_default(),
        &body.host,
        body.port,
        &body.user,
        &enc(key, &body.password)?,
        &body.database_name,
        &body.ssl_mode,
        &body.extra,
        &body.db_type,
    ).await?;
    Ok(mask_db(&row))
}

pub async fn db_update(
    db: &PgPool,
    key: &[u8; 32],
    id: i64,
    body: UpdateDbMetadata,
) -> Result<Value, AppError> {
    let current = repository::db_find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("database metadata not found"))?;

    let password = match body.password.as_deref() {
        Some(pw) if !pw.is_empty() => enc(key, pw)?,
        _ => current.password.clone(),
    };

    let row = repository::db_update(
        db, id,
        &body.name.unwrap_or(current.name),
        &body.host.unwrap_or(current.host),
        body.port.unwrap_or(current.port),
        &body.user.unwrap_or(current.user),
        &password,
        &body.database_name.unwrap_or(current.database_name),
        &body.ssl_mode.unwrap_or(current.ssl_mode),
        &body.extra.unwrap_or(current.extra),
        &body.db_type.unwrap_or(current.db_type),
    ).await?;
    Ok(mask_db(&row))
}

pub async fn db_delete(db: &PgPool, id: i64) -> Result<(), AppError> {
    let n = repository::db_delete(db, id).await?;
    if n == 0 {
        return Err(AppError::not_found("database metadata not found"));
    }
    Ok(())
}

pub async fn db_test_connection(body: TestDbConnectionRequest) -> Value {
    let conn_str = build_pg_connstr(&body.host, body.port, &body.user, &body.password, &body.database_name, &body.ssl_mode);
    match test_pg_connection(&conn_str).await {
        Ok(v)  => json!(TestConnectionResult { success: true,  message: "Connected".into(), version: Some(v) }),
        Err(e) => json!(TestConnectionResult { success: false, message: e, version: None }),
    }
}

// ---------------------------------------------------------------------------
// SFTP service functions
// ---------------------------------------------------------------------------

pub async fn sftp_list(db: &PgPool) -> Result<Value, AppError> {
    let rows = repository::sftp_list(db).await?;
    Ok(json!({ "items": rows.iter().map(mask_sftp).collect::<Vec<_>>() }))
}

pub async fn sftp_get(db: &PgPool, id: i64) -> Result<Value, AppError> {
    let row = repository::sftp_find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("sftp metadata not found"))?;
    Ok(mask_sftp(&row))
}

pub async fn sftp_create(
    db: &PgPool,
    key: &[u8; 32],
    body: CreateSftpMetadata,
) -> Result<Value, AppError> {
    let row = repository::sftp_create(
        db,
        &body.name.unwrap_or_default(),
        &body.host,
        body.port,
        &body.user,
        &enc(key, &body.password)?,
        &enc(key, &body.private_key)?,
        &body.base_path,
        &body.extra,
    ).await?;
    Ok(mask_sftp(&row))
}

pub async fn sftp_update(
    db: &PgPool,
    key: &[u8; 32],
    id: i64,
    body: UpdateSftpMetadata,
) -> Result<Value, AppError> {
    let current = repository::sftp_find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("sftp metadata not found"))?;

    let password = match body.password.as_deref() {
        Some(pw) if !pw.is_empty() => enc(key, pw)?,
        _ => current.password.clone(),
    };
    let private_key = match body.private_key.as_deref() {
        Some(pk) if !pk.is_empty() => enc(key, pk)?,
        _ => current.private_key.clone(),
    };

    let row = repository::sftp_update(
        db, id,
        &body.name.unwrap_or(current.name),
        &body.host.unwrap_or(current.host),
        body.port.unwrap_or(current.port),
        &body.user.unwrap_or(current.user),
        &password,
        &private_key,
        &body.base_path.unwrap_or(current.base_path),
        &body.extra.unwrap_or(current.extra),
    ).await?;
    Ok(mask_sftp(&row))
}

pub async fn sftp_delete(db: &PgPool, id: i64) -> Result<(), AppError> {
    let n = repository::sftp_delete(db, id).await?;
    if n == 0 {
        return Err(AppError::not_found("sftp metadata not found"));
    }
    Ok(())
}

// ---------------------------------------------------------------------------
// Email service functions
// ---------------------------------------------------------------------------

pub async fn email_list(db: &PgPool) -> Result<Value, AppError> {
    let rows = repository::email_list(db).await?;
    Ok(json!({ "items": rows.iter().map(mask_email).collect::<Vec<_>>() }))
}

pub async fn email_get(db: &PgPool, id: i64) -> Result<Value, AppError> {
    let row = repository::email_find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("email metadata not found"))?;
    Ok(mask_email(&row))
}

pub async fn email_create(
    db: &PgPool,
    key: &[u8; 32],
    body: CreateEmailMetadata,
) -> Result<Value, AppError> {
    let row = repository::email_create(
        db,
        &body.name.unwrap_or_default(),
        &body.imap_host,
        body.imap_port,
        &body.smtp_host,
        body.smtp_port,
        &body.username,
        &enc(key, &body.password)?,
        body.use_tls,
    ).await?;
    Ok(mask_email(&row))
}

pub async fn email_update(
    db: &PgPool,
    key: &[u8; 32],
    id: i64,
    body: UpdateEmailMetadata,
) -> Result<Value, AppError> {
    let current = repository::email_find_by_id(db, id)
        .await?
        .ok_or_else(|| AppError::not_found("email metadata not found"))?;

    let password = match body.password.as_deref() {
        Some(pw) if !pw.is_empty() => enc(key, pw)?,
        _ => current.password.clone(),
    };

    let row = repository::email_update(
        db, id,
        &body.name.unwrap_or(current.name),
        &body.imap_host.unwrap_or(current.imap_host),
        body.imap_port.unwrap_or(current.imap_port),
        &body.smtp_host.unwrap_or(current.smtp_host),
        body.smtp_port.unwrap_or(current.smtp_port),
        &body.username.unwrap_or(current.username),
        &password,
        body.use_tls.unwrap_or(current.use_tls),
    ).await?;
    Ok(mask_email(&row))
}

pub async fn email_delete(db: &PgPool, id: i64) -> Result<(), AppError> {
    let n = repository::email_delete(db, id).await?;
    if n == 0 {
        return Err(AppError::not_found("email metadata not found"));
    }
    Ok(())
}

pub async fn email_test_connection(body: TestEmailConnectionRequest) -> Value {
    let smtp_result = test_smtp(&body.smtp_host, body.smtp_port, &body.username, &body.password, body.use_tls).await;
    json!(TestEmailConnectionResult {
        imap_success: false,
        imap_message: "IMAP test not implemented yet".into(),
        smtp_success: smtp_result.is_ok(),
        smtp_message: smtp_result.unwrap_or_else(|e| e),
    })
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

fn build_pg_connstr(host: &str, port: i32, user: &str, password: &str, dbname: &str, ssl_mode: &str) -> String {
    format!("host={host} port={port} user={user} password={password} dbname={dbname} sslmode={ssl_mode}")
}

async fn test_pg_connection(conn_str: &str) -> Result<String, String> {
    use tokio_postgres::NoTls;
    let (client, connection) = tokio_postgres::connect(conn_str, NoTls)
        .await
        .map_err(|e| e.to_string())?;
    tokio::spawn(async move { let _ = connection.await; });
    let row = client.query_one("SELECT version()", &[]).await.map_err(|e| e.to_string())?;
    Ok(row.get::<_, String>(0))
}

async fn test_smtp(host: &str, port: i32, username: &str, password: &str, use_tls: bool) -> Result<String, String> {
    use lettre::{transport::smtp::authentication::Credentials, AsyncSmtpTransport, AsyncTransport, Tokio1Executor};
    let creds = Credentials::new(username.into(), password.into());
    if use_tls {
        let t: AsyncSmtpTransport<Tokio1Executor> =
            AsyncSmtpTransport::<Tokio1Executor>::relay(host)
                .map_err(|e| e.to_string())?
                .port(port as u16)
                .credentials(creds)
                .build();
        t.test_connection().await.map_err(|e| e.to_string())?;
    } else {
        let t: AsyncSmtpTransport<Tokio1Executor> =
            AsyncSmtpTransport::<Tokio1Executor>::starttls_relay(host)
                .map_err(|e| e.to_string())?
                .port(port as u16)
                .credentials(creds)
                .build();
        t.test_connection().await.map_err(|e| e.to_string())?;
    }
    Ok("SMTP connection successful".into())
}
