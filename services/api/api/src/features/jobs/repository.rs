use std::collections::HashMap;

use sqlx::PgPool;

use crate::error::AppError;
use super::{
    dto::{Connexion, NodeInput},
    model::{Job, JobUserInfo, NodeRow, PortRow, UserBasic},
};

pub async fn list(db: &PgPool, user_id: i64) -> Result<Vec<Job>, AppError> {
    sqlx::query_as(
        "SELECT DISTINCT j.* FROM job j
         LEFT JOIN job_user_access jua ON jua.job_id = j.id AND jua.user_id = $1
         WHERE j.creator_id = $1 OR jua.user_id = $1 OR j.visibility = 'public'
         ORDER BY j.created_at DESC",
    )
    .bind(user_id)
    .fetch_all(db)
    .await
    .map_err(AppError::from)
}

pub async fn list_by_file_path(db: &PgPool, user_id: i64, file_path: &str) -> Result<Vec<Job>, AppError> {
    sqlx::query_as(
        "SELECT DISTINCT j.* FROM job j
         LEFT JOIN job_user_access jua ON jua.job_id = j.id AND jua.user_id = $1
         WHERE (j.creator_id = $1 OR jua.user_id = $1 OR j.visibility = 'public')
           AND j.file_path = $2
         ORDER BY j.created_at DESC",
    )
    .bind(user_id)
    .bind(file_path)
    .fetch_all(db)
    .await
    .map_err(AppError::from)
}

pub async fn find_by_id(db: &PgPool, id: i64) -> Result<Option<Job>, AppError> {
    sqlx::query_as("SELECT * FROM job WHERE id = $1")
        .bind(id)
        .fetch_optional(db)
        .await
        .map_err(AppError::from)
}

pub async fn create(db: &PgPool, req: &super::dto::CreateJobRequest, creator_id: i64) -> Result<Job, AppError> {
    sqlx::query_as(
        "INSERT INTO job (name, description, file_path, creator_id, active, visibility, output_path)
         VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING *",
    )
    .bind(&req.name)
    .bind(&req.description)
    .bind(&req.file_path)
    .bind(creator_id)
    .bind(req.active)
    .bind(&req.visibility)
    .bind(&req.output_path)
    .fetch_one(db)
    .await
    .map_err(AppError::from)
}

pub async fn update(
    db: &PgPool,
    id: i64,
    req: &super::dto::UpdateJobRequest,
    current: &Job,
) -> Result<(), AppError> {
    sqlx::query(
        "UPDATE job SET name=$2, description=$3, file_path=$4, active=$5, visibility=$6, output_path=$7, updated_at=now()
         WHERE id=$1",
    )
    .bind(id)
    .bind(req.name.as_deref().unwrap_or(&current.name))
    .bind(req.description.as_deref().unwrap_or(&current.description))
    .bind(req.file_path.as_deref().unwrap_or(&current.file_path))
    .bind(req.active.unwrap_or(current.active))
    .bind(req.visibility.as_deref().unwrap_or(&current.visibility))
    .bind(req.output_path.as_deref().unwrap_or(&current.output_path))
    .execute(db)
    .await?;
    Ok(())
}

pub async fn delete(db: &PgPool, id: i64) -> Result<u64, AppError> {
    let result = sqlx::query("DELETE FROM job WHERE id = $1")
        .bind(id)
        .execute(db)
        .await?;
    Ok(result.rows_affected())
}

pub async fn get_access(db: &PgPool, job_id: i64, user_id: i64) -> Result<Option<String>, AppError> {
    let job: Option<(i64, String)> = sqlx::query_as(
        "SELECT creator_id, visibility FROM job WHERE id = $1",
    )
    .bind(job_id)
    .fetch_optional(db)
    .await?;

    let (creator_id, visibility) = match job {
        None    => return Ok(None),
        Some(j) => j,
    };

    if creator_id == user_id {
        return Ok(Some("owner".into()));
    }

    let access: Option<String> = sqlx::query_scalar(
        "SELECT role FROM job_user_access WHERE job_id = $1 AND user_id = $2",
    )
    .bind(job_id)
    .bind(user_id)
    .fetch_optional(db)
    .await?;

    if let Some(role) = access {
        return Ok(Some(role));
    }

    if visibility == "public" {
        return Ok(Some("viewer".into()));
    }

    Ok(None)
}

pub async fn get_nodes(db: &PgPool, job_id: i64) -> Result<Vec<NodeRow>, AppError> {
    sqlx::query_as("SELECT * FROM node WHERE job_id = $1 ORDER BY id")
        .bind(job_id)
        .fetch_all(db)
        .await
        .map_err(AppError::from)
}

pub async fn get_ports_for_nodes(db: &PgPool, node_ids: &[i64]) -> Result<Vec<PortRow>, AppError> {
    if node_ids.is_empty() {
        return Ok(vec![]);
    }
    sqlx::query_as(
        "SELECT * FROM port WHERE node_id = ANY($1) ORDER BY node_id, id",
    )
    .bind(node_ids)
    .fetch_all(db)
    .await
    .map_err(AppError::from)
}

pub async fn get_shared_users(db: &PgPool, job_id: i64) -> Result<Vec<JobUserInfo>, AppError> {
    sqlx::query_as(
        "SELECT jua.user_id, u.email, u.prenom, u.nom, jua.role
         FROM job_user_access jua
         JOIN users u ON u.id = jua.user_id
         WHERE jua.job_id = $1",
    )
    .bind(job_id)
    .fetch_all(db)
    .await
    .map_err(AppError::from)
}

pub async fn get_notification_contacts(db: &PgPool, job_id: i64) -> Result<Vec<UserBasic>, AppError> {
    sqlx::query_as(
        "SELECT u.id, u.email, u.prenom, u.nom
         FROM job_notification_contact jnc
         JOIN users u ON u.id = jnc.user_id
         WHERE jnc.job_id = $1",
    )
    .bind(job_id)
    .fetch_all(db)
    .await
    .map_err(AppError::from)
}

pub async fn upsert_nodes(
    tx: &mut sqlx::Transaction<'_, sqlx::Postgres>,
    job_id: i64,
    nodes: &[NodeInput],
    connexions: &[Connexion],
) -> Result<(), AppError> {
    // 1. Clear all ports for this job's nodes (rebuild from connexions)
    sqlx::query("DELETE FROM port WHERE node_id IN (SELECT id FROM node WHERE job_id = $1)")
        .bind(job_id)
        .execute(&mut **tx)
        .await?;

    // 2. Determine existing node IDs in DB
    let existing_ids: Vec<i64> = sqlx::query_scalar("SELECT id FROM node WHERE job_id = $1")
        .bind(job_id)
        .fetch_all(&mut **tx)
        .await?;
    let existing_set: std::collections::HashSet<i64> = existing_ids.into_iter().collect();

    // 3. Split nodes into update vs insert
    let mut id_map: HashMap<i64, i64> = HashMap::new();

    let incoming_existing: Vec<&NodeInput> = nodes.iter()
        .filter(|n| n.id > 0 && existing_set.contains(&n.id)).collect();
    let incoming_new: Vec<&NodeInput> = nodes.iter()
        .filter(|n| n.id <= 0 || !existing_set.contains(&n.id)).collect();

    // Delete nodes not in the incoming set
    let keep_ids: Vec<i64> = incoming_existing.iter().map(|n| n.id).collect();
    if keep_ids.is_empty() {
        sqlx::query("DELETE FROM node WHERE job_id = $1").bind(job_id).execute(&mut **tx).await?;
    } else {
        sqlx::query("DELETE FROM node WHERE job_id = $1 AND id <> ALL($2)")
            .bind(job_id)
            .bind(&keep_ids)
            .execute(&mut **tx)
            .await?;
    }

    // Update existing nodes
    for n in &incoming_existing {
        sqlx::query("UPDATE node SET name=$2, xpos=$3, ypos=$4, data=$5 WHERE id=$1")
            .bind(n.id).bind(&n.name).bind(n.xpos).bind(n.ypos)
            .bind(&n.data)
            .execute(&mut **tx)
            .await?;
        id_map.insert(n.id, n.id);
    }

    // Insert new nodes
    for n in &incoming_new {
        let new_id: i64 = sqlx::query_scalar(
            "INSERT INTO node (job_id, type, name, xpos, ypos, data) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id",
        )
        .bind(job_id).bind(&n.r#type).bind(&n.name).bind(n.xpos).bind(n.ypos)
        .bind(&n.data)
        .fetch_one(&mut **tx)
        .await?;
        id_map.insert(n.id, new_id);
    }

    // 4. Insert ports from connexions
    for c in connexions {
        let src_id = id_map.get(&c.source_node_id).copied().unwrap_or(c.source_node_id);
        let tgt_id = id_map.get(&c.target_node_id).copied().unwrap_or(c.target_node_id);
        if src_id == 0 || tgt_id == 0 { continue; }

        let out_type = match c.source_port_type.as_str() {
            "flow" => "node_flow_output",
            _      => "output",
        };
        let in_type = match c.target_port_type.as_str() {
            "flow" => "node_flow_input",
            _      => "input",
        };

        sqlx::query("INSERT INTO port (node_id, type, connected_node_id) VALUES ($1,$2,$3)")
            .bind(src_id).bind(out_type).bind(tgt_id)
            .execute(&mut **tx).await?;

        sqlx::query("INSERT INTO port (node_id, type, connected_node_id) VALUES ($1,$2,$3)")
            .bind(tgt_id).bind(in_type).bind(src_id)
            .execute(&mut **tx).await?;
    }

    Ok(())
}

pub async fn set_shares(
    db: &PgPool,
    job_id: i64,
    user_ids: &[i64],
    role: &str,
) -> Result<(), AppError> {
    sqlx::query("DELETE FROM job_user_access WHERE job_id = $1")
        .bind(job_id)
        .execute(db)
        .await?;
    for uid in user_ids {
        sqlx::query(
            "INSERT INTO job_user_access (job_id, user_id, role) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING",
        )
        .bind(job_id).bind(uid).bind(role)
        .execute(db).await?;
    }
    Ok(())
}

pub async fn add_share(
    db: &PgPool,
    job_id: i64,
    user_id: i64,
    role: &str,
) -> Result<(), AppError> {
    sqlx::query(
        "INSERT INTO job_user_access (job_id, user_id, role) VALUES ($1,$2,$3)
         ON CONFLICT (job_id, user_id) DO UPDATE SET role = EXCLUDED.role",
    )
    .bind(job_id).bind(user_id).bind(role)
    .execute(db).await?;
    Ok(())
}

pub async fn remove_share(db: &PgPool, job_id: i64, user_id: i64) -> Result<(), AppError> {
    sqlx::query("DELETE FROM job_user_access WHERE job_id = $1 AND user_id = $2")
        .bind(job_id).bind(user_id)
        .execute(db).await?;
    Ok(())
}

pub async fn add_notification_contact(db: &PgPool, job_id: i64, user_id: i64) -> Result<(), AppError> {
    sqlx::query(
        "INSERT INTO job_notification_contact (job_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING",
    )
    .bind(job_id).bind(user_id)
    .execute(db).await?;
    Ok(())
}

pub async fn remove_notification_contact(db: &PgPool, job_id: i64, user_id: i64) -> Result<(), AppError> {
    sqlx::query("DELETE FROM job_notification_contact WHERE job_id = $1 AND user_id = $2")
        .bind(job_id).bind(user_id)
        .execute(db).await?;
    Ok(())
}
