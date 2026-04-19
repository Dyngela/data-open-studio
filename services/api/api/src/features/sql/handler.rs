use axum::{extract::State, Json};
use serde::{Deserialize, Serialize};

use crate::{error::AppError, state::AppState};
use super::{dto::*, service};

pub async fn guess_query(
    State(state): State<AppState>,
    Json(body): Json<GuessQueryRequest>,
) -> Result<Json<GuessQueryResponse>, AppError> {
    Ok(Json(service::guess_query(&state, body).await?))
}

pub async fn optimize_query(
    State(state): State<AppState>,
    Json(body): Json<OptimizeQueryRequest>,
) -> Result<Json<OptimizeQueryResponse>, AppError> {
    Ok(Json(service::optimize_query(&state, body).await?))
}

pub async fn test_connection(
    Json(body): Json<IntrospectRequest>,
) -> Result<Json<TestConnectionResult>, AppError> {
    Ok(Json(service::test_connection(body).await?))
}

pub async fn get_tables(
    State(state): State<AppState>,
    Json(body): Json<IntrospectRequest>,
) -> Result<Json<DatabaseIntrospection>, AppError> {
    Ok(Json(service::get_tables(&state, body).await?))
}

pub async fn get_columns(
    State(state): State<AppState>,
    Json(body): Json<IntrospectRequest>,
) -> Result<Json<DatabaseIntrospection>, AppError> {
    Ok(Json(service::get_columns(&state, body).await?))
}

// ---------------------------------------------------------------------------
// db_node handler (combined here since sql feature has no model/repo)
// ---------------------------------------------------------------------------

#[derive(Deserialize)]
pub struct GuessSchemaRequest {
    pub query:         String,
    pub node_id:       i64,
    pub connection_id: i64,
}

#[derive(Serialize)]
pub struct DataModelResponse {
    pub column:      String,
    pub data_type:   String,
    pub nullable:    bool,
    pub primary_key: bool,
}

#[derive(Serialize)]
pub struct GuessSchemaResponse {
    pub node_id:     i64,
    pub data_models: Vec<DataModelResponse>,
}

pub async fn guess_schema(
    State(state): State<AppState>,
    Json(body): Json<GuessSchemaRequest>,
) -> Result<Json<GuessSchemaResponse>, AppError> {
    let (node_id, models) = service::guess_schema(&state.db, &body.query, body.node_id, body.connection_id).await?;
    Ok(Json(GuessSchemaResponse {
        node_id,
        data_models: models.into_iter().map(|m| DataModelResponse {
            column:      m.column,
            data_type:   m.data_type,
            nullable:    m.nullable,
            primary_key: m.primary_key,
        }).collect(),
    }))
}

// ---------------------------------------------------------------------------
// OpenAPI doc for this feature
// ---------------------------------------------------------------------------

use utoipa::OpenApi;

#[derive(OpenApi)]
#[openapi()]
pub struct ApiDoc;

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

pub fn routes() -> axum::Router<AppState> {
    use axum::routing::post;
    axum::Router::new()
        .route("/api/v1/sql/guess-query",               post(guess_query))
        .route("/api/v1/sql/optimize-query",            post(optimize_query))
        .route("/api/v1/sql/introspect/test-connection", post(test_connection))
        .route("/api/v1/sql/introspect/tables",         post(get_tables))
        .route("/api/v1/sql/introspect/columns",        post(get_columns))
        .route("/api/v1/db-node/guess-schema",          post(guess_schema))
}
