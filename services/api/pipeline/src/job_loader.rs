use serde_json::Value;
use sqlx::PgPool;
use thiserror::Error;

use crate::node::NodeConfig;

#[derive(Debug, Error)]
pub enum LoadError {
    #[error("job {0} not found")]
    JobNotFound(i64),
    #[error("database error: {0}")]
    Db(#[from] sqlx::Error),
    #[error("invalid node config for node {0}: {1}")]
    InvalidNodeConfig(i64, serde_json::Error),
}

#[derive(Debug, Clone)]
pub struct LoadedNode {
    pub id:     i64,
    pub name:   String,
    pub config: NodeConfig,
}

#[derive(Debug, Clone)]
pub struct LoadedEdge {
    pub from_node_id: i64,
    pub to_node_id:   i64,
}

#[derive(Debug, Clone)]
pub struct LoadedJob {
    pub id:    i64,
    pub name:  String,
    pub nodes: Vec<LoadedNode>,
    pub edges: Vec<LoadedEdge>,
}

pub async fn load(pool: &PgPool, job_id: i64) -> Result<LoadedJob, LoadError> {
    // Verify job exists
    let job_name: Option<String> = sqlx::query_scalar("SELECT name FROM job WHERE id = $1")
        .bind(job_id)
        .fetch_optional(pool)
        .await?;
    let job_name = job_name.ok_or(LoadError::JobNotFound(job_id))?;

    // Load nodes
    let rows: Vec<(i64, String, String, Value)> = sqlx::query_as(
        "SELECT id, name, type, data FROM node WHERE job_id = $1",
    )
    .bind(job_id)
    .fetch_all(pool)
    .await?;

    let mut nodes = Vec::with_capacity(rows.len());
    for (id, name, node_type, data) in rows {
        let config = deserialize_node_config(&node_type, data)
            .map_err(|e| LoadError::InvalidNodeConfig(id, e))?;
        nodes.push(LoadedNode { id, name, config });
    }

    // Load edges via ports
    let edges: Vec<(i64, i64)> = sqlx::query_as(
        "SELECT p.node_id, p.connected_node_id
         FROM port p
         JOIN node n ON n.id = p.node_id
         WHERE n.job_id = $1
           AND p.type IN ('node_flow_output', 'output')
           AND p.connected_node_id <> 0",
    )
    .bind(job_id)
    .fetch_all(pool)
    .await?;

    let edges = edges.into_iter()
        .map(|(from, to)| LoadedEdge { from_node_id: from, to_node_id: to })
        .collect();

    Ok(LoadedJob { id: job_id, name: job_name, nodes, edges })
}

fn deserialize_node_config(node_type: &str, data: Value) -> Result<NodeConfig, serde_json::Error> {
    match node_type {
        "db_input"     => Ok(NodeConfig::DbInput(serde_json::from_value(data)?)),
        "db_output"    => Ok(NodeConfig::DbOutput(serde_json::from_value(data)?)),
        "map"          => Ok(NodeConfig::Map(serde_json::from_value(data)?)),
        "log"          => Ok(NodeConfig::Log(serde_json::from_value(data)?)),
        "email_output" => Ok(NodeConfig::EmailOutput(serde_json::from_value(data)?)),
        _              => Ok(NodeConfig::Start),
    }
}
