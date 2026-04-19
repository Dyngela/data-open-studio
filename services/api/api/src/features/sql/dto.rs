use serde::{Deserialize, Serialize};
use serde_json::Value;

#[derive(Deserialize)]
pub struct GuessQueryRequest {
    pub prompt:                       String,
    #[serde(default)]
    pub schema_optimization_needed:   bool,
    pub connection_id:                Option<i64>,
    #[serde(default)]
    pub previous_messages:            Vec<Value>,
}

#[derive(Deserialize)]
pub struct OptimizeQueryRequest {
    pub query:         String,
    pub connection_id: Option<i64>,
}

#[derive(Deserialize)]
pub struct IntrospectRequest {
    pub metadata_database_id: Option<i64>,
    pub connection:            Option<InlineConnection>,
    pub table_name:            Option<String>,
}

#[derive(Deserialize)]
pub struct InlineConnection {
    pub host:          String,
    pub port:          i32,
    pub user:          String,
    pub password:      String,
    pub database_name: String,
    #[serde(default = "default_ssl")]
    pub ssl_mode:      String,
    pub db_type:       String,
}

fn default_ssl() -> String { "disable".into() }

#[derive(Serialize)]
pub struct GuessQueryResponse {
    pub query: String,
}

#[derive(Serialize)]
pub struct OptimizeQueryResponse {
    pub optimized_query: String,
    pub explanation:     String,
}

#[derive(Serialize)]
pub struct DatabaseTable {
    pub schema: String,
    pub name:   String,
}

#[derive(Serialize)]
pub struct DatabaseColumn {
    pub name:        String,
    pub data_type:   String,
    pub is_nullable: bool,
    pub is_primary:  bool,
}

#[derive(Serialize)]
pub struct DatabaseIntrospection {
    pub tables:  Vec<DatabaseTable>,
    pub columns: Vec<DatabaseColumn>,
}

#[derive(Serialize)]
pub struct TestConnectionResult {
    pub success: bool,
    pub message: String,
    pub version: Option<String>,
}
