//! connector: pluggable data-source connectors that produce Polars DataFrames.
//!
//! Each connector implements the [`Connector`] trait and loads data into the
//! shared [`DataFrameStore`].

pub mod csv;
pub mod postgres;

use async_trait::async_trait;
use df_store::{DataFrameStore, StoreError};
use polars::prelude::DataFrame;

#[derive(Debug, thiserror::Error)]
pub enum ConnectorError {
    #[error("store error: {0}")]
    Store(#[from] StoreError),
    #[error("io error: {0}")]
    Io(#[from] std::io::Error),
    #[error("postgres error: {0}")]
    Postgres(#[from] tokio_postgres::Error),
    #[error("polars error: {0}")]
    Polars(#[from] polars::prelude::PolarsError),
    #[error("{0}")]
    Other(String),
}

/// Common interface for all data-source connectors.
#[async_trait]
pub trait Connector: Send + Sync {
    /// Load data and return a DataFrame.
    async fn load(&self) -> Result<DataFrame, ConnectorError>;

    /// Load data and persist it in the store under `key`.
    async fn load_into(
        &self,
        store: &DataFrameStore,
        key: impl Into<String> + Send,
    ) -> Result<(), ConnectorError> {
        let df = self.load().await?;
        store.put(key, df).await;
        Ok(())
    }
}