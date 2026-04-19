use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum NodeConfig {
    DbInput(DbInputConfig),
    DbOutput(DbOutputConfig),
    Map(MapConfig),
    Log(LogConfig),
    EmailOutput(EmailOutputConfig),
    Start,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct DbInputConfig {
    pub query:              String,
    pub db_schema:          String,
    pub query_with_schema:  String,
    pub batch_size:         i32,
    pub connection:         DbConnectionConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct DbOutputConfig {
    pub mode:       String, // insert | update | delete | merge | truncate
    pub batch_size: i32,
    pub connection: DbConnectionConfig,
    pub table:      String,
    pub db_schema:  String,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct MapConfig {
    pub inputs:    Vec<MapInput>,
    pub outputs:   Vec<MapOutput>,
    pub join:      Option<JoinConfig>,
    pub variables: Vec<MapVariable>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct MapInput {
    pub node_id:    String,
    pub alias:      String,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct MapOutput {
    pub name:   String,
    pub fields: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct JoinConfig {
    pub join_type:    String, // inner | left | right | cross | union
    pub left_keys:    Vec<String>,
    pub right_keys:   Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct MapVariable {
    pub name:       String,
    pub expression: String,
    pub kind:       String, // computed | filter
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct LogConfig {
    pub separator: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct EmailOutputConfig {
    pub to:       Vec<String>,
    pub subject:  String,
    pub body:     String,
    pub metadata_email_id: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct DbConnectionConfig {
    pub db_type:       String, // postgres | sqlserver | mysql
    pub host:          String,
    pub port:          i32,
    pub database:      String,
    pub username:      String,
    pub password:      String,
    pub ssl_mode:      String,
    pub metadata_id:   Option<i64>,
}

impl DbConnectionConfig {
    pub fn connection_string(&self) -> String {
        match self.db_type.as_str() {
            "postgres" => format!(
                "host={} port={} dbname={} user={} password={} sslmode={}",
                self.host, self.port, self.database, self.username, self.password, self.ssl_mode
            ),
            _ => format!(
                "postgresql://{}:{}@{}:{}/{}",
                self.username, self.password, self.host, self.port, self.database
            ),
        }
    }
}
