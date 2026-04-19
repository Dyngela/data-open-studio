extern crate pipeline as pipeline_crate;

mod app;
mod config;
mod crypto;
mod migration;
mod error;
mod features;
mod frame_json;
mod openapi;
mod pipeline;
mod state;
mod storage;
mod trigger_poller;
mod websocket;

use sqlx::postgres::PgPoolOptions;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

use config::AppConfig;
use state::AppState;

/// CLI args — supports both server mode (default) and worker mode.
#[derive(Debug)]
enum Mode {
    Server,
    Worker { job_id: i64 },
}

fn parse_mode() -> Mode {
    let args: Vec<String> = std::env::args().collect();
    if args.contains(&"--worker".to_string()) {
        let job_id = args.windows(2)
            .find(|w| w[0] == "--job-id")
            .and_then(|w| w[1].parse().ok())
            .expect("--worker requires --job-id <id>");
        Mode::Worker { job_id }
    } else {
        Mode::Server
    }
}

#[tokio::main]
async fn main() {
    tracing_subscriber::registry()
        .with(EnvFilter::try_from_default_env().unwrap_or_else(|_| "info".into()))
        .with(tracing_subscriber::fmt::layer()
            .with_file(true)
            .with_line_number(true))
        .init();

    let _ = dotenvy::dotenv();

    match parse_mode() {
        Mode::Worker { job_id } => run_worker(job_id).await,
        Mode::Server            => run_server().await,
    }
}

async fn run_worker(job_id: i64) {
    use pipeline_crate::{
        config::WorkerConfig,
        // executor::run_dag,
        job_loader,
        metrics::WorkerMetrics,
        progress::ProgressReporter,
    };

    let cfg  = WorkerConfig::from_env(job_id);
    tracing::info!(job_id, "worker starting");

    let pool = PgPoolOptions::new()
        .max_connections(4)
        .connect(&cfg.database_url)
        .await
        .unwrap_or_else(|e| { tracing::error!("db connect: {e}"); std::process::exit(1); });

    let nats = if let Some(url) = &cfg.nats_url {
        async_nats::connect(url).await.ok()
    } else {
        None
    };

    let job = job_loader::load(&pool, job_id).await.unwrap_or_else(|e| {
        tracing::error!("job load failed: {e}");
        std::process::exit(1);
    });

    let progress = ProgressReporter::new(job_id, nats, cfg.tenant_id.clone());
    let metrics  = WorkerMetrics::new();

    // match run_dag(&job, &pool, &progress, metrics.clone()).await {
    //     Ok(()) => {
    //         tracing::info!(job_id, rows = metrics.rows(), "worker completed");
    //         std::process::exit(0);
    //     }
    //     Err(e) => {
    //         tracing::error!(job_id, error = %e, "worker failed");
    //         std::process::exit(1);
    //     }
    // }
}

async fn run_server() {
    let cfg = AppConfig::from_env();

    let db = PgPoolOptions::new()
        .max_connections(10)
        .connect(&cfg.database_url)
        .await
        .expect("failed to connect to database");

    // migration::migrate(&db).await;

    let addr  = format!("0.0.0.0:{}", cfg.port);
    tracing::info!("{:?}", cfg.clone());
    
    let state = AppState::new(db, cfg);
    // Spawn NATS → WebSocket bridge
    {
        let hub      = state.hub.clone();
        let nats_url = state.config.nats_url.clone();
        tokio::spawn(websocket::nats_bridge::run(hub, nats_url));
    }

    // Spawn trigger poller
    // {
    //     let s = state.clone();
    //     tokio::spawn(trigger_poller::poller::run(s));
    // }

    tracing::info!("api listening on {addr}");
    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app::build(state)).await.unwrap();
}
