//! CSV file connector.

use std::path::{Path, PathBuf};

use async_trait::async_trait;
use polars::prelude::*;

use crate::{Connector, ConnectorError};

pub struct CsvConnector {
    path: PathBuf,
    delimiter: u8,
    has_header: bool,
}

impl CsvConnector {
    pub fn new(path: impl AsRef<Path>) -> Self {
        Self {
            path: path.as_ref().to_owned(),
            delimiter: b',',
            has_header: true,
        }
    }

    pub fn delimiter(mut self, d: u8) -> Self {
        self.delimiter = d;
        self
    }

    pub fn has_header(mut self, v: bool) -> Self {
        self.has_header = v;
        self
    }
}

#[async_trait]
impl Connector for CsvConnector {
    async fn load(&self) -> Result<DataFrame, ConnectorError> {
        let path = self.path.clone();
        let delimiter = self.delimiter;
        let has_header = self.has_header;

        // Run blocking Polars I/O on a thread-pool thread.
        tokio::task::spawn_blocking(move || {
            CsvReadOptions::default()
                .with_has_header(has_header)
                .map_parse_options(|o| o.with_separator(delimiter))
                .try_into_reader_with_file_path(Some(path))?
                .finish()
                .map_err(ConnectorError::Polars)
        })
        .await
        .map_err(|e| ConnectorError::Other(e.to_string()))?
    }
}