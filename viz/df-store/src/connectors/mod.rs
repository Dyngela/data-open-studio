use crate::connectors::csv::CsvConfig;
use crate::connectors::excel::ExcelConfig;
use crate::connectors::postgres::PostgresConfig;

pub mod csv;
pub mod excel;
pub mod postgres;
pub mod extractor;

pub enum ConnectorType {
    Csv(CsvConfig),
    Excel(ExcelConfig),
    Postgres(PostgresConfig),
}

pub enum ConnectorError {
    FileNotFound(String),
    ConnectionError(String),
    ParsingError(String),
    ArbitraryError(String),
}

impl std::fmt::Display for ConnectorError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ConnectorError::FileNotFound(s)    => write!(f, "file not found: {s}"),
            ConnectorError::ConnectionError(s) => write!(f, "connection error: {s}"),
            ConnectorError::ParsingError(s)    => write!(f, "parsing error: {s}"),
            ConnectorError::ArbitraryError(s)  => write!(f, "{s}"),
        }
    }
}

impl From<std::io::Error> for ConnectorError {
    fn from(e: std::io::Error) -> Self {
        match e.kind() {
            std::io::ErrorKind::NotFound => ConnectorError::FileNotFound(e.to_string()),
            _ => ConnectorError::ArbitraryError(e.to_string()),
        }
    }
}


pub trait Connector {
    fn load(self, df: &mut crate::cedrus::Cedrus) -> Result<(), ConnectorError>;
}

impl Connector for ConnectorType {
    fn load(self, store: &mut crate::cedrus::Cedrus) -> Result<(), ConnectorError> {
        match self {
            ConnectorType::Csv(config) => {
                csv::load_csv(config, store)

            }
            ConnectorType::Excel(config) => {
                // Excel loading logic
                todo!()
            }
            ConnectorType::Postgres(config) => {
                postgres::load_postgres(config, store)
            }
        }
    }
}

