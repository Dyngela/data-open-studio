use serde::{Deserialize, Serialize};

// ---------------------------------------------------------------------------
// Database request DTOs
// ---------------------------------------------------------------------------

#[derive(Deserialize)]
pub struct CreateDbMetadata {
    pub name:          Option<String>,
    pub host:          String,
    pub port:          i32,
    pub user:          String,
    pub password:      String,
    pub database_name: String,
    #[serde(default = "default_ssl")]
    pub ssl_mode:      String,
    #[serde(default)]
    pub extra:         String,
    pub db_type:       String,
}

#[derive(Deserialize)]
pub struct UpdateDbMetadata {
    pub name:          Option<String>,
    pub host:          Option<String>,
    pub port:          Option<i32>,
    pub user:          Option<String>,
    pub password:      Option<String>,
    pub database_name: Option<String>,
    pub ssl_mode:      Option<String>,
    pub extra:         Option<String>,
    pub db_type:       Option<String>,
}

// ---------------------------------------------------------------------------
// SFTP request DTOs
// ---------------------------------------------------------------------------

#[derive(Deserialize)]
pub struct CreateSftpMetadata {
    pub name:        Option<String>,
    pub host:        String,
    pub port:        i32,
    pub user:        String,
    #[serde(default)]
    pub password:    String,
    #[serde(default)]
    pub private_key: String,
    #[serde(default)]
    pub base_path:   String,
    #[serde(default)]
    pub extra:       String,
}

#[derive(Deserialize)]
pub struct UpdateSftpMetadata {
    pub name:        Option<String>,
    pub host:        Option<String>,
    pub port:        Option<i32>,
    pub user:        Option<String>,
    pub password:    Option<String>,
    pub private_key: Option<String>,
    pub base_path:   Option<String>,
    pub extra:       Option<String>,
}

// ---------------------------------------------------------------------------
// Email request DTOs
// ---------------------------------------------------------------------------

#[derive(Deserialize)]
pub struct CreateEmailMetadata {
    pub name:      Option<String>,
    #[serde(default)]
    pub imap_host: String,
    #[serde(default = "default_imap_port")]
    pub imap_port: i32,
    #[serde(default)]
    pub smtp_host: String,
    #[serde(default = "default_smtp_port")]
    pub smtp_port: i32,
    pub username:  String,
    pub password:  String,
    #[serde(default = "default_true")]
    pub use_tls:   bool,
}

#[derive(Deserialize)]
pub struct UpdateEmailMetadata {
    pub name:      Option<String>,
    pub imap_host: Option<String>,
    pub imap_port: Option<i32>,
    pub smtp_host: Option<String>,
    pub smtp_port: Option<i32>,
    pub username:  Option<String>,
    pub password:  Option<String>,
    pub use_tls:   Option<bool>,
}

// ---------------------------------------------------------------------------
// Test-connection request/response
// ---------------------------------------------------------------------------

#[derive(Deserialize)]
pub struct TestDbConnectionRequest {
    pub host:          String,
    pub port:          i32,
    pub user:          String,
    pub password:      String,
    pub database_name: String,
    #[serde(default = "default_ssl")]
    pub ssl_mode:      String,
    pub db_type:       String,
}

#[derive(Serialize)]
pub struct TestConnectionResult {
    pub success: bool,
    pub message: String,
    pub version: Option<String>,
}

#[derive(Deserialize)]
pub struct TestEmailConnectionRequest {
    pub imap_host: String,
    pub imap_port: i32,
    pub smtp_host: String,
    pub smtp_port: i32,
    pub username:  String,
    pub password:  String,
    pub use_tls:   bool,
}

#[derive(Serialize)]
pub struct TestEmailConnectionResult {
    pub imap_success: bool,
    pub imap_message: String,
    pub smtp_success: bool,
    pub smtp_message: String,
}

fn default_ssl() -> String { "disable".into() }
fn default_imap_port() -> i32 { 993 }
fn default_smtp_port() -> i32 { 587 }
fn default_true() -> bool { true }
