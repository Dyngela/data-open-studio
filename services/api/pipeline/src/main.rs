use clap::Parser;
use sqlx::postgres::PgPoolOptions;

use pipeline::{
    config::WorkerConfig,
    executor,
    job_loader,
    metrics::WorkerMetrics,
    progress::ProgressReporter,
};

#[derive(Parser)]
#[command(name = "worker")]
struct Args {
    #[arg(long)]
    job_id: i64,
}

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt()
        .with_env_filter(tracing_subscriber::EnvFilter::from_default_env())
        .init();

    let _ = dotenvy::dotenv();
    let args = Args::parse();
    let cfg  = WorkerConfig::from_env(args.job_id);

    tracing::info!(job_id = cfg.job_id, "worker starting");

    // Connect to database
    let pool = PgPoolOptions::new()
        .max_connections(4)
        .connect(&cfg.database_url)
        .await
        .unwrap_or_else(|e| {
            tracing::error!("db connect failed: {e}");
            std::process::exit(1);
        });

    // Connect to NATS (optional)
    let nats = if let Some(url) = &cfg.nats_url {
        match async_nats::connect(url).await {
            Ok(c)  => { tracing::info!("NATS connected"); Some(c) }
            Err(e) => { tracing::warn!("NATS connect failed: {e}"); None }
        }
    } else {
        None
    };

    // Load job from DB
    let job = job_loader::load(&pool, cfg.job_id).await.unwrap_or_else(|e| {
        tracing::error!("job load failed: {e}");
        std::process::exit(1);
    });

    tracing::info!(job_id = cfg.job_id, job_name = %job.name,
        nodes = job.nodes.len(), "job loaded");

    let progress = ProgressReporter::new(cfg.job_id, nats, cfg.tenant_id.clone());
    let metrics  = WorkerMetrics::new();

    // match executor::run_dag(&job, &pool, &progress, metrics.clone()).await {
    //     Ok(()) => {
    //         tracing::info!(job_id = cfg.job_id, rows = metrics.rows(), "worker completed");
    //         std::process::exit(0);
    //     }
    //     Err(e) => {
    //         tracing::error!(job_id = cfg.job_id, error = %e, "worker failed");
    //         std::process::exit(1);
    //     }
    // }
}
