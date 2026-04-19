use std::collections::HashMap;

use sqlx::PgPool;

use crate::{error::AppError, pipeline, state::AppState};
use super::{
    dto::*,
    model::{Job, NodeRow, PortRow},
    repository,
};

// ---------------------------------------------------------------------------
// Access control
// ---------------------------------------------------------------------------

pub async fn user_role(db: &PgPool, job_id: i64, user_id: i64) -> Result<Option<String>, AppError> {
    repository::get_access(db, job_id, user_id).await
}

pub fn require_role(role: Option<String>, needed: &str) -> Result<String, AppError> {
    let role = role.ok_or_else(|| AppError::forbidden("no access to this job"))?;
    let ok = match needed {
        "owner"  => role == "owner",
        "editor" => role == "owner" || role == "editor",
        _        => true,
    };
    if !ok {
        return Err(AppError::forbidden(format!("role '{role}' is insufficient, need '{needed}'")));
    }
    Ok(role)
}

// ---------------------------------------------------------------------------
// Internal helper: load full job detail
// ---------------------------------------------------------------------------

pub async fn load_with_details(db: &PgPool, job_id: i64) -> Result<JobWithNodesResponse, AppError> {
    let job: Job = repository::find_by_id(db, job_id)
        .await?
        .ok_or_else(|| AppError::not_found("job not found"))?;

    let nodes = repository::get_nodes(db, job_id).await?;
    let node_ids: Vec<i64> = nodes.iter().map(|n| n.id).collect();
    let ports = repository::get_ports_for_nodes(db, &node_ids).await?;
    let shared = repository::get_shared_users(db, job_id).await?;
    let contacts = repository::get_notification_contacts(db, job_id).await?;

    let mut ports_by_node: HashMap<i64, Vec<&PortRow>> = HashMap::new();
    for p in &ports {
        ports_by_node.entry(p.node_id).or_default().push(p);
    }

    let connexions = build_connexions(&nodes, &ports_by_node);

    let node_responses = nodes.iter().map(|n| NodeResponse {
        id: n.id, r#type: n.node_type.clone(), name: n.name.clone(),
        xpos: n.xpos, ypos: n.ypos,
        data: n.data.clone().unwrap_or_else(|| serde_json::json!({})),
    }).collect();

    Ok(JobWithNodesResponse {
        id: job.id, name: job.name, description: job.description,
        file_path: job.file_path, creator_id: job.creator_id,
        active: job.active, visibility: job.visibility,
        output_path: job.output_path,
        created_at: job.created_at, updated_at: job.updated_at,
        nodes: node_responses,
        connexions,
        shared_user: shared.into_iter().map(|u| SharedUserResponse {
            id: u.user_id, email: u.email, prenom: u.prenom, nom: u.nom, role: u.role,
        }).collect(),
        notification_contacts: contacts.into_iter().map(|u| UserBasicResponse {
            id: u.id, email: u.email, prenom: u.prenom, nom: u.nom,
        }).collect(),
    })
}

fn build_connexions(nodes: &[NodeRow], ports_by_node: &HashMap<i64, Vec<&PortRow>>) -> Vec<ConnexionResponse> {
    let mut node_input_ports: HashMap<i64, (Vec<i64>, Vec<i64>)> = HashMap::new();
    for node in nodes {
        let empty = vec![];
        let node_ports = ports_by_node.get(&node.id).unwrap_or(&empty);
        let flow_inputs: Vec<i64> = node_ports.iter()
            .filter(|p| p.port_type == "node_flow_input")
            .map(|p| p.connected_node_id)
            .collect();
        let data_inputs: Vec<i64> = node_ports.iter()
            .filter(|p| p.port_type == "input")
            .map(|p| p.connected_node_id)
            .collect();
        node_input_ports.insert(node.id, (flow_inputs, data_inputs));
    }

    let mut connexions = Vec::new();
    for node in nodes {
        let empty = vec![];
        let node_ports = ports_by_node.get(&node.id).unwrap_or(&empty);

        let mut flow_out_idx = 0i32;
        let mut data_out_idx = 0i32;

        for port in node_ports.iter().filter(|p| matches!(p.port_type.as_str(), "node_flow_output" | "output")) {
            if port.connected_node_id == 0 { continue; }

            let (conn_type, src_idx) = if port.port_type == "node_flow_output" {
                let idx = flow_out_idx;
                flow_out_idx += 1;
                ("flow", idx)
            } else {
                let idx = data_out_idx;
                data_out_idx += 1;
                ("data", idx)
            };

            let target_idx = if let Some((flow_ins, data_ins)) = node_input_ports.get(&port.connected_node_id) {
                let search = if conn_type == "flow" { flow_ins } else { data_ins };
                search.iter().position(|&src| src == node.id).unwrap_or(0) as i32
            } else {
                0
            };

            connexions.push(ConnexionResponse {
                source_node_id:   node.id,
                source_port:      src_idx,
                source_port_type: conn_type.into(),
                target_node_id:   port.connected_node_id,
                target_port:      target_idx,
                target_port_type: conn_type.into(),
            });
        }
    }
    connexions
}

// ---------------------------------------------------------------------------
// Public service functions
// ---------------------------------------------------------------------------

pub async fn list(db: &PgPool, user_id: i64, file_path: Option<&str>) -> Result<Vec<JobResponse>, AppError> {
    let jobs = if let Some(fp) = file_path {
        repository::list_by_file_path(db, user_id, fp).await?
    } else {
        repository::list(db, user_id).await?
    };
    Ok(jobs.into_iter().map(JobResponse::from).collect())
}

pub async fn get(db: &PgPool, job_id: i64, caller_id: i64) -> Result<JobWithNodesResponse, AppError> {
    let role = user_role(db, job_id, caller_id).await?;
    require_role(role, "viewer")?;
    load_with_details(db, job_id).await
}

pub async fn create(
    db: &PgPool,
    req: CreateJobRequest,
    caller_id: i64,
) -> Result<JobWithNodesResponse, AppError> {
    if req.name.trim().is_empty() {
        return Err(AppError::bad_request("name is required"));
    }

    let job = repository::create(db, &req, caller_id).await?;

    for user_id in &req.shared_with {
        let _ = sqlx::query(
            "INSERT INTO job_user_access (job_id, user_id, role) VALUES ($1,$2,'viewer') ON CONFLICT DO NOTHING",
        )
        .bind(job.id).bind(user_id)
        .execute(db).await;
    }

    load_with_details(db, job.id).await
}

pub async fn update(
    db: &PgPool,
    job_id: i64,
    req: UpdateJobRequest,
    caller_id: i64,
) -> Result<JobWithNodesResponse, AppError> {
    let role = user_role(db, job_id, caller_id).await?;
    require_role(role, "editor")?;

    let current = repository::find_by_id(db, job_id)
        .await?
        .ok_or_else(|| AppError::not_found("job not found"))?;

    repository::update(db, job_id, &req, &current).await?;

    if let Some(share_ids) = &req.shared_with {
        repository::set_shares(db, job_id, share_ids, "viewer").await?;
    }

    if let Some(nodes) = &req.nodes {
        let mut tx = db.begin().await?;
        repository::upsert_nodes(&mut tx, job_id, nodes, &req.connexions).await?;
        tx.commit().await?;
    }

    load_with_details(db, job_id).await
}

pub async fn delete(db: &PgPool, job_id: i64, caller_id: i64) -> Result<(), AppError> {
    let role = user_role(db, job_id, caller_id).await?;
    require_role(role, "owner")?;
    let n = repository::delete(db, job_id).await?;
    if n == 0 {
        return Err(AppError::not_found("job not found"));
    }
    Ok(())
}

pub async fn share(
    db: &PgPool,
    job_id: i64,
    req: ShareRequest,
    caller_id: i64,
) -> Result<JobWithNodesResponse, AppError> {
    let role = user_role(db, job_id, caller_id).await?;
    require_role(role, "owner")?;
    for uid in &req.user_ids {
        repository::add_share(db, job_id, *uid, &req.role).await?;
    }
    load_with_details(db, job_id).await
}

pub async fn unshare(
    db: &PgPool,
    job_id: i64,
    req: ShareRequest,
    caller_id: i64,
) -> Result<JobWithNodesResponse, AppError> {
    let role = user_role(db, job_id, caller_id).await?;
    require_role(role, "owner")?;
    for uid in &req.user_ids {
        repository::remove_share(db, job_id, *uid).await?;
    }
    load_with_details(db, job_id).await
}

pub async fn add_notification_contact(
    db: &PgPool,
    job_id: i64,
    user_id: i64,
    caller_id: i64,
) -> Result<JobWithNodesResponse, AppError> {
    let role = user_role(db, job_id, caller_id).await?;
    require_role(role, "editor")?;
    repository::add_notification_contact(db, job_id, user_id).await?;
    load_with_details(db, job_id).await
}

pub async fn remove_notification_contact(
    db: &PgPool,
    job_id: i64,
    contact_id: i64,
    caller_id: i64,
) -> Result<JobWithNodesResponse, AppError> {
    let role = user_role(db, job_id, caller_id).await?;
    require_role(role, "editor")?;
    repository::remove_notification_contact(db, job_id, contact_id).await?;
    load_with_details(db, job_id).await
}

pub async fn execute_job(
    state: &AppState,
    job_id: i64,
    caller_id: i64,
) -> Result<(), AppError> {
    let role = user_role(&state.db, job_id, caller_id).await?;
    require_role(role, "editor")?;
    pipeline::executor::spawn_worker(job_id, state)
        .await
        .map_err(AppError::internal)?;
    Ok(())
}

pub async fn stop_job(
    state: &AppState,
    job_id: i64,
    caller_id: i64,
) -> Result<(), AppError> {
    let role = user_role(&state.db, job_id, caller_id).await?;
    require_role(role, "editor")?;
    state.registry.kill(job_id).await;
    Ok(())
}

pub async fn print_code(
    db: &PgPool,
    job_id: i64,
    caller_id: i64,
) -> Result<serde_json::Value, AppError> {
    let role = user_role(db, job_id, caller_id).await?;
    require_role(role, "viewer")?;
    let detail = load_with_details(db, job_id).await?;
    Ok(serde_json::json!({
        "id":         detail.id,
        "name":       detail.name,
        "nodes":      detail.nodes,
        "connexions": detail.connexions,
    }))
}
