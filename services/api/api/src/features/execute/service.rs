use resin::executor::Executor;
use resin::{FrameExt, Lexer};
use resin::parser::parse;
use serde_json::Value;
use uuid::Uuid;

use crate::error::AppError;
use crate::frame_json::{frame_data_json, frame_schema_json};
use crate::state::AppState;
use super::dto::ExecuteRequest;

pub fn run(
    state: &AppState,
    workspace_id: Uuid,
    body: ExecuteRequest,
) -> Result<Value, AppError> {
    let tokens = Lexer::new(&body.script)
        .tokenize()
        .map_err(|errs| AppError::bad_request(format!("lex errors: {:?}", errs)))?;

    let (program, parse_errs) = parse(tokens);
    if !parse_errs.is_empty() {
        let msg = parse_errs.iter().map(|e| e.to_string()).collect::<Vec<_>>().join("; ");
        return Err(AppError::bad_request(format!("parse error: {msg}")));
    }

    let mut executor = Executor::new();
    {
        let guard = state.workspaces.read().unwrap();
        if let Some(ws) = guard.get(&workspace_id) {
            for (name, frame) in &ws.frames {
                executor.load(name, frame.clone_with_name(name));
            }
        }
    }

    executor.run(&program).map_err(|e| AppError::bad_request(e.to_string()))?;

    let materialized: Vec<String> = program.statements.iter()
        .filter_map(|s| match s {
            resin::ast::Statement::Query(q) => q.materialize.as_ref().map(|m| m.name.clone()),
            resin::ast::Statement::Frame(f) => Some(f.name.name.clone()),
            _ => None,
        })
        .collect();

    let schemas: Vec<Value> = materialized.iter()
        .filter_map(|name| executor.get(name).map(frame_schema_json))
        .collect();

    let result_data: Option<Value> = body.result.as_ref().and_then(|name| {
        executor.get(name).map(|frame| frame_data_json(frame, 0, body.limit))
    });

    tracing::info!(
        workspace_id = %workspace_id,
        materialized = ?materialized,
        "resin script executed"
    );

    Ok(serde_json::json!({
        "materialized": schemas,
        "result":       result_data,
    }))
}
