use axum::{
    extract::{Path, State},
    Json,
};
use resin::executor::Executor;
use resin::{Lexer, FrameExt};
use resin::parser::parse;
use serde::Deserialize;
use serde_json::{json, Value};
use uuid::Uuid;

use crate::error::AppError;
use crate::frame_json::frame_schema_json;
use crate::state::AppState;

#[derive(Deserialize)]
pub struct ExecuteRequest {
    /// Resin script to run.
    pub script: String,
    /// Optional name of a single result frame to return data for.
    /// If absent, returns schemas of all materialized frames.
    pub result: Option<String>,
    /// Max rows to return per frame (default 100).
    #[serde(default = "default_limit")]
    pub limit: usize,
}
fn default_limit() -> usize { 100 }

/// POST /workspaces/:workspace_id/execute
///
/// Runs a Resin script against the workspace's loaded frames.
/// Returns the schemas of all newly materialized frames, plus data
/// for the frame named in `result` (if provided).
pub async fn handle_execute(
    Path(workspace_id): Path<Uuid>,
    State(state): State<AppState>,
    Json(body): Json<ExecuteRequest>,
) -> Result<Json<Value>, AppError> {
    // 1. Lex + parse the Resin script.
    let tokens = Lexer::new(&body.script)
        .tokenize()
        .map_err(|errs| AppError::bad_request(format!("lex errors: {:?}", errs)))?;
    let (program, parse_errs) = parse(tokens);
    if !parse_errs.is_empty() {
        let msg = parse_errs.iter().map(|e| e.to_string()).collect::<Vec<_>>().join("; ");
        return Err(AppError::bad_request(format!("parse error: {msg}")));
    }

    // 2. Build an executor seeded with the workspace's current frames.
    let mut executor = Executor::new();
    {
        let guard = state.workspaces.read().unwrap();
        if let Some(ws) = guard.get(&workspace_id) {
            for (name, frame) in &ws.frames {
                // clone_with_name gives us an owned Frame from a reference
                executor.load(name, frame.clone_with_name(name));
            }
        }
    }

    // 3. Execute.
    executor
        .run(&program)
        .map_err(|e| AppError::bad_request(e.to_string()))?;

    // 4. Build response: schema list + optional data for requested frame.
    // Collect the names of all frames that are now in the executor.
    // We expose the schemas via the executor's public `get` API by iterating
    // over the program's materialized statement names.
    let materialized: Vec<String> = program
        .statements
        .iter()
        .filter_map(|s| match s {
            resin::ast::Statement::Query(q) => q.materialize.as_ref().map(|m| m.name.clone()),
            resin::ast::Statement::Frame(f) => Some(f.name.name.clone()),
            _ => None,
        })
        .collect();

    let schemas: Vec<Value> = materialized
        .iter()
        .filter_map(|name| executor.get(name).map(frame_schema_json))
        .collect();

    let result_data: Option<Value> = body.result.as_ref().and_then(|name| {
        executor.get(name).map(|frame| {
            use crate::frame_json::frame_data_json;
            frame_data_json(frame, 0, body.limit)
        })
    });

    Ok(Json(json!({
        "materialized": schemas,
        "result":       result_data,
    })))
}
