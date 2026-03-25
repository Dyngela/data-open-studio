//! PostgreSQL connector — executes a query and maps the result set to a Polars DataFrame.

use async_trait::async_trait;
use polars::prelude::*;
use tokio_postgres::{types::ToSql, NoTls};

use crate::{Connector, ConnectorError};

pub struct PostgresConnector {
    connection_string: String,
    query: String,
    #[allow(dead_code)] // reserved for parameterised queries
    params: Vec<Box<dyn ToSql + Sync + Send>>,
}

impl PostgresConnector {
    /// Create a connector from a `libpq`-style connection string and a SQL query.
    pub fn new(connection_string: impl Into<String>, query: impl Into<String>) -> Self {
        Self {
            connection_string: connection_string.into(),
            query: query.into(),
            params: vec![],
        }
    }
}

#[async_trait]
impl Connector for PostgresConnector {
    async fn load(&self) -> Result<DataFrame, ConnectorError> {
        let (client, connection) =
            tokio_postgres::connect(&self.connection_string, NoTls).await?;

        // Drive the connection in the background.
        tokio::spawn(async move {
            if let Err(e) = connection.await {
                tracing::error!("postgres connection error: {e}");
            }
        });

        let rows = client.query(&self.query, &[]).await?;

        if rows.is_empty() {
            return Ok(DataFrame::empty());
        }

        // Build one Vec<AnyValue> per column, then assemble a DataFrame.
        let columns = rows[0].columns();
        let series_vec: Vec<Column> = columns
            .iter()
            .map(|col| {
                let name = col.name().to_owned();
                // Collect as strings for a simple first implementation;
                // extend per-type matching as needed.
                let values: Vec<Option<String>> = rows
                    .iter()
                    .map(|row| row.try_get::<_, Option<String>>(col.name()).unwrap_or(None))
                    .collect();
                Column::new(name.into(), values)
            })
            .collect();

        DataFrame::new(series_vec).map_err(ConnectorError::Polars)
    }
}