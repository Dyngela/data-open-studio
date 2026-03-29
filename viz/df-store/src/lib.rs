//! df-store: in-memory DataFrame store backed by Polars.
//!
//! Provides a thread-safe registry that maps string keys to Polars DataFrames,
//! with basic CRUD and query helpers.

pub mod store;
mod chunk;
mod time;
pub mod data_type;
pub mod connectors;
pub mod frame;
pub mod cedrus;
pub mod query;