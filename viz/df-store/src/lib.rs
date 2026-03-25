//! df-store: in-memory DataFrame store backed by Polars.
//!
//! Provides a thread-safe registry that maps string keys to Polars DataFrames,
//! with basic CRUD and query helpers.

use std::collections::HashMap;
use std::sync::Arc;

use polars::prelude::*;
use tokio::sync::RwLock;

pub type StoreKey = String;

#[derive(Debug, thiserror::Error)]
pub enum StoreError {
    #[error("frame not found: {0}")]
    NotFound(StoreKey),
    #[error("polars error: {0}")]
    Polars(#[from] PolarsError),
}

/// Thread-safe, key-addressable collection of Polars DataFrames.
#[derive(Clone, Default)]
pub struct DataFrameStore {
    frames: Arc<RwLock<HashMap<StoreKey, DataFrame>>>,
}

impl DataFrameStore {
    pub fn new() -> Self {
        Self::default()
    }

    /// Insert or replace a DataFrame.
    pub async fn put(&self, key: impl Into<StoreKey>, df: DataFrame) {
        self.frames.write().await.insert(key.into(), df);
    }

    /// Return a clone of the stored DataFrame.
    pub async fn get(&self, key: &str) -> Result<DataFrame, StoreError> {
        self.frames
            .read()
            .await
            .get(key)
            .cloned()
            .ok_or_else(|| StoreError::NotFound(key.to_owned()))
    }

    /// Remove a DataFrame; returns true if it existed.
    pub async fn remove(&self, key: &str) -> bool {
        self.frames.write().await.remove(key).is_some()
    }

    /// List all registered keys.
    pub async fn keys(&self) -> Vec<StoreKey> {
        self.frames.read().await.keys().cloned().collect()
    }
}