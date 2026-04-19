use axum::{middleware, response::{Html, IntoResponse}, routing::get, Json, Router};
use tower_http::trace::TraceLayer;
use utoipa::OpenApi;

use crate::{features, openapi::ApiDoc, state::AppState, websocket};
use crate::features::auth::auth_util::middleware::require_auth;

pub fn build(state: AppState) -> Router {
    let public = Router::new()
        .merge(features::auth::routes())
        .route("/ws", get(websocket::handler::handle))
        .route("/docs/openapi.json", get(openapi_json))
        .route("/docs", get(swagger_ui));

    let protected = Router::new()
        .merge(features::workspaces::routes())
        .merge(features::sources::routes())
        .merge(features::frames::routes())
        .merge(features::execute::routes())
        .merge(features::metadata::routes())
        .merge(features::triggers::routes())
        .merge(features::datasets::routes())
        .merge(features::sql::routes())
        .merge(features::jobs::routes())
        .route_layer(middleware::from_fn_with_state(state.clone(), require_auth));

    Router::new()
        .merge(public)
        .merge(protected)
        .layer(TraceLayer::new_for_http())
        .with_state(state)
}

async fn openapi_json() -> impl IntoResponse {
    Json(ApiDoc::openapi())
}

async fn swagger_ui() -> impl IntoResponse {
    Html(r##"<!DOCTYPE html><html><head><title>API</title><link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css"></head><body><div id="swagger-ui"></div><script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script><script>SwaggerUIBundle({url:"/docs/openapi.json",dom_id:"#swagger-ui",presets:[SwaggerUIBundle.presets.apis],layout:"BaseLayout",persistAuthorization:true});</script></body></html>"##)
}
