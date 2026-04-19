use serde::Serialize;
use sqlx::FromRow;

#[derive(Debug, Clone, Serialize, FromRow)]
pub struct MetadataDatabase {
    pub id:            i64,
    pub name:          String,
    pub host:          String,
    pub port:          i32,
    pub user:          String,
    pub password:      String,
    pub database_name: String,
    pub ssl_mode:      String,
    pub extra:         String,
    pub db_type:       String,
}

#[derive(Debug, Clone, Serialize, FromRow)]
pub struct MetadataSftp {
    pub id:          i64,
    pub name:        String,
    pub host:        String,
    pub port:        i32,
    pub user:        String,
    pub password:    String,
    pub private_key: String,
    pub base_path:   String,
    pub extra:       String,
}

#[derive(Debug, Clone, Serialize, FromRow)]
pub struct MetadataEmail {
    pub id:        i64,
    pub name:      String,
    pub imap_host: String,
    pub imap_port: i32,
    pub smtp_host: String,
    pub smtp_port: i32,
    pub username:  String,
    pub password:  String,
    pub use_tls:   bool,
}
