use axum::{
    middleware,
    response::{Html, IntoResponse},
    routing::{delete, get, post},
    Json, Router,
};
use tower_http::trace::TraceLayer;
use utoipa::OpenApi;

use crate::{
    auth::{handlers as auth_handlers, middleware::require_auth},
    handlers::{datasets, db_node, execute, frames, jobs, metadata, sources, sql, triggers, workspaces},
    openapi::ApiDoc,
    state::AppState,
    websocket,
};

pub fn build(state: AppState) -> Router {
    // Public routes — no JWT required
    let public = Router::new()
        .route("/api/v1/auth_util/register", post(auth_handlers::register))
        .route("/api/v1/auth_util/login",    post(auth_handlers::login))
        .route("/api/v1/auth_util/refresh",  post(auth_handlers::refresh_token))
        // WebSocket — auth_util via query param token
        .route("/ws", get(websocket::handler::handle))
        // OpenAPI spec + Swagger UI
        .route("/docs/openapi.json", get(openapi_json))
        .route("/docs",              get(swagger_ui));

    // Protected routes — JWT required
    let protected = Router::new()
        // Auth utilities
        .route("/api/v1/me",           get(auth_handlers::me))
        .route("/api/v1/users/search", get(auth_handlers::search_users))
        // Workspaces
        .route("/api/v1/workspaces",      get(workspaces::list).post(workspaces::create))
        .route("/api/v1/workspaces/{id}", get(workspaces::get).patch(workspaces::update).delete(workspaces::delete))
        // Sources
        .route("/api/v1/workspaces/{workspace_id}/sources",
            get(sources::list).post(sources::create))
        .route("/api/v1/workspaces/{workspace_id}/sources/{source_id}",
            delete(sources::delete))
        .route("/api/v1/workspaces/{workspace_id}/sources/{source_id}/load",
            post(sources::load))
        // Frames
        .route("/api/v1/workspaces/{workspace_id}/frames",
            get(frames::list))
        .route("/api/v1/workspaces/{workspace_id}/frames/{frame_name}",
            delete(frames::delete))
        .route("/api/v1/workspaces/{workspace_id}/frames/{frame_name}/preview",
            get(frames::preview))
        // Execute (Resin)
        .route("/api/v1/workspaces/{workspace_id}/execute",
            post(execute::handle))
        // Metadata — Database
        .route("/api/v1/metadata/db",
            get(metadata::db_list).post(metadata::db_create))
        .route("/api/v1/metadata/db/{id}",
            get(metadata::db_get).put(metadata::db_update).delete(metadata::db_delete))
        .route("/api/v1/metadata/db/test-connection",
            post(metadata::db_test_connection))
        // Metadata — SFTP
        .route("/api/v1/metadata/sftp",
            get(metadata::sftp_list).post(metadata::sftp_create))
        .route("/api/v1/metadata/sftp/{id}",
            get(metadata::sftp_get).put(metadata::sftp_update).delete(metadata::sftp_delete))
        // Metadata — Email
        .route("/api/v1/metadata/email",
            get(metadata::email_list).post(metadata::email_create))
        .route("/api/v1/metadata/email/{id}",
            get(metadata::email_get).put(metadata::email_update).delete(metadata::email_delete))
        .route("/api/v1/metadata/email/test-connection",
            post(metadata::email_test_connection))
        // SQL utilities
        .route("/api/v1/sql/guess-query",        post(sql::guess_query))
        .route("/api/v1/sql/optimize-query",     post(sql::optimize_query))
        .route("/api/v1/sql/introspect/test-connection", post(sql::test_connection))
        .route("/api/v1/sql/introspect/tables",  post(sql::get_tables))
        .route("/api/v1/sql/introspect/columns", post(sql::get_columns))
        // DB node
        .route("/api/v1/db-node/guess-schema",   post(db_node::guess_schema))
        // Datasets
        .route("/api/v1/datasets",     get(datasets::list).post(datasets::create))
        .route("/api/v1/datasets/{id}", get(datasets::get_by_id).put(datasets::update).delete(datasets::delete))
        .route("/api/v1/datasets/{id}/refresh",      post(datasets::refresh))
        .route("/api/v1/datasets/{id}/preview",      post(datasets::preview))
        .route("/api/v1/datasets/{id}/query",        post(datasets::query))
        .route("/api/v1/datasets/{id}/load-as-frame", post(datasets::load_as_frame))
        // Jobs
        .route("/api/v1/jobs",     get(jobs::list).post(jobs::create))
        .route("/api/v1/jobs/{id}", get(jobs::get_by_id).put(jobs::update).delete(jobs::delete))
        .route("/api/v1/jobs/{id}/share",   post(jobs::share).delete(jobs::unshare))
        .route("/api/v1/jobs/{id}/execute", post(jobs::execute))
        .route("/api/v1/jobs/{id}/stop",    post(jobs::stop))
        .route("/api/v1/jobs/{id}/print-code", post(jobs::print_code))
        .route("/api/v1/jobs/{id}/notification-contacts",
            post(jobs::add_notification_contact))
        .route("/api/v1/jobs/{id}/notification-contacts/{user_id}",
            delete(jobs::remove_notification_contact))
        // Triggers
        .route("/api/v1/triggers",     get(triggers::list).post(triggers::create))
        .route("/api/v1/triggers/{id}", get(triggers::get_by_id).put(triggers::update).delete(triggers::delete))
        .route("/api/v1/triggers/{id}/activate",       post(triggers::activate))
        .route("/api/v1/triggers/{id}/pause",          post(triggers::pause))
        .route("/api/v1/triggers/{id}/rules",          post(triggers::add_rule))
        .route("/api/v1/triggers/{id}/rules/{rule_id}", axum::routing::put(triggers::update_rule).delete(triggers::delete_rule))
        .route("/api/v1/triggers/{id}/jobs",           post(triggers::link_job))
        .route("/api/v1/triggers/{id}/jobs/{job_id}",  delete(triggers::unlink_job))
        .route("/api/v1/triggers/{id}/executions",     get(triggers::get_executions))
        .layer(middleware::from_fn_with_state(state.clone(), require_auth));

    public
        .merge(protected)
        .layer(TraceLayer::new_for_http())
        .with_state(state)
}

async fn openapi_json() -> impl IntoResponse {
    Json(ApiDoc::openapi())
}

async fn swagger_ui() -> impl IntoResponse {
    Html(r#"<!DOCTYPE html>
<html>
<head>
  <title>Data Open Studio API</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script>
<script>
  SwaggerUIBundle({
    url: '/docs/openapi.json',
    dom_id: '#swagger-ui',
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
    layout: 'BaseLayout',
    persistAuthorization: true,
  });
</script>
</body>
</html>"#)
}
