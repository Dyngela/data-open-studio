//! api: Axum HTTP server exposing the DataFrame store and connectors to the frontend.

use axum::{
    extract::{Path, State},
    http::StatusCode,
    routing::get,
    Json, Router,
};
use df_store::DataFrameStore;
use serde_json::{json, Value};
use tower_http::cors::CorsLayer;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

// ── Shared application state ──────────────────────────────────────────────────

#[derive(Clone)]
struct AppState {
    store: DataFrameStore,
}

// ── Entry point ───────────────────────────────────────────────────────────────

#[tokio::main]
async fn main() {
    tracing_subscriber::registry()
        .with(EnvFilter::try_from_default_env().unwrap_or_else(|_| "info".into()))
        .with(tracing_subscriber::fmt::layer())
        .init();

    let state = AppState {
        store: DataFrameStore::new(),
    };

    let app = Router::new()
        .route("/frames", get(list_frames))
        .route("/frames/:key", get(get_frame))
        .layer(CorsLayer::permissive())
        .with_state(state);

    let addr = "0.0.0.0:3030";
    tracing::info!("viz api listening on {addr}");

    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

// ── Route handlers ────────────────────────────────────────────────────────────

/// List all frame keys currently in the store.
async fn list_frames(State(state): State<AppState>) -> Json<Value> {
    let keys = state.store.keys().await;
    Json(json!({ "frames": keys }))
}

/// Return a frame serialised as column-oriented JSON.
async fn get_frame(
    Path(key): Path<String>,
    State(state): State<AppState>,
) -> Result<Json<Value>, (StatusCode, Json<Value>)> {
    let df = state.store.get(&key).await.map_err(|_| {
        (
            StatusCode::NOT_FOUND,
            Json(json!({ "error": format!("frame '{key}' not found") })),
        )
    })?;

    // Serialise each column as a JSON array of values.
    let mut map = serde_json::Map::new();
    for col in df.get_columns() {
        let name = col.name().to_string();
        let values: Vec<Value> = (0..col.len())
            .map(|i| match col.get(i) {
                Ok(av) => format!("{av}").into(),
                Err(_) => Value::Null,
            })
            .collect();
        map.insert(name, Value::Array(values));
    }

    Ok(Json(Value::Object(map)))
}