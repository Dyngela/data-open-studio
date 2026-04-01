mod error;
mod frame_json;
mod models;
mod routes;
mod state;

use axum::{
    routing::{delete, get, post},
    Router,
};
use sqlx::postgres::PgPoolOptions;
use tower_http::cors::CorsLayer;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

use routes::{
    execute::handle_execute,
    frames::{list_frames, preview_frame},
    sources::{create_source, delete_source, list_sources, load_source},
    workspaces::{create_workspace, delete_workspace, get_workspace, list_workspaces},
};
use state::AppState;

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

#[tokio::main]
async fn main() {
    tracing_subscriber::registry()
        .with(EnvFilter::try_from_default_env().unwrap_or_else(|_| "info".into()))
        .with(tracing_subscriber::fmt::layer())
        .init();

    let database_url = std::env::var("DATABASE_URL")
        .unwrap_or_else(|_| "postgres://postgres:postgres@localhost:5432/data_open_studio".into());

    let db = PgPoolOptions::new()
        .max_connections(10)
        .connect(&database_url)
        .await
        .expect("failed to connect to database");

    // Run migrations on startup.
    migrate(&db).await;

    let state = AppState::new(db);

    let app = Router::new()
        // Workspaces
        .route("/workspaces",     get(list_workspaces).post(create_workspace))
        .route("/workspaces/:id", get(get_workspace).delete(delete_workspace))
        // Sources
        .route("/workspaces/:workspace_id/sources",
            get(list_sources).post(create_source))
        .route("/workspaces/:workspace_id/sources/:source_id",
            delete(delete_source))
        .route("/workspaces/:workspace_id/sources/:source_id/load",
            post(load_source))
        // Frames (in-memory)
        .route("/workspaces/:workspace_id/frames",
            get(list_frames))
        .route("/workspaces/:workspace_id/frames/:frame_name/preview",
            get(preview_frame))
        // Execute
        .route("/workspaces/:workspace_id/execute",
            post(handle_execute))
        .layer(CorsLayer::permissive())
        .with_state(state);

    let addr = "0.0.0.0:3030";
    tracing::info!("viz api listening on {addr}");
    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

// ---------------------------------------------------------------------------
// DB migrations (inline DDL)
// ---------------------------------------------------------------------------

async fn migrate(db: &sqlx::PgPool) {
    sqlx::query(
        "CREATE TABLE IF NOT EXISTS workspace (
            id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
            name       TEXT        NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT now()
        )",
    )
    .execute(db)
    .await
    .expect("failed to create workspace table");

    sqlx::query(
        "CREATE TABLE IF NOT EXISTS source (
            id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
            workspace_id UUID        NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
            name         TEXT        NOT NULL,
            source_type  TEXT        NOT NULL,
            config       JSONB       NOT NULL DEFAULT '{}',
            created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
        )",
    )
    .execute(db)
    .await
    .expect("failed to create source table");

    tracing::info!("migrations applied");
}
