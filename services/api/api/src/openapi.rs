use utoipa::{Modify, OpenApi};
use utoipa::openapi::security::{HttpAuthScheme, HttpBuilder, SecurityScheme};

use crate::features;

#[derive(OpenApi)]
#[openapi(
    info(
        title = "Data Open Studio API",
        version = "1.0.0",
        description = "Visualization workspace, source, frame, and Resin query execution API"
    ),
    modifiers(&BearerAuth),
    tags(
        (name = "auth_util",       description = "Authentication and user management"),
        (name = "workspaces", description = "Workspace CRUD"),
        (name = "sources",    description = "Data source registration and loading"),
        (name = "frames",     description = "In-memory DataFrame management"),
        (name = "execute",    description = "Resin query execution"),
        (name = "jobs",       description = "Pipeline job management"),
        (name = "metadata",   description = "Connection metadata"),
        (name = "triggers",   description = "Event triggers"),
        (name = "datasets",   description = "Named datasets"),
        (name = "sql",        description = "SQL utilities"),
    )
)]
struct RootDoc;

pub struct ApiDoc;

impl ApiDoc {
    pub fn openapi() -> utoipa::openapi::OpenApi {
        let mut doc = RootDoc::openapi();
        doc.merge(features::auth::handler::ApiDoc::openapi());
        doc.merge(features::workspaces::handler::ApiDoc::openapi());
        doc.merge(features::sources::handler::ApiDoc::openapi());
        doc.merge(features::frames::handler::ApiDoc::openapi());
        doc.merge(features::execute::handler::ApiDoc::openapi());
        doc.merge(features::jobs::handler::ApiDoc::openapi());
        doc.merge(features::metadata::handler::ApiDoc::openapi());
        doc.merge(features::triggers::handler::ApiDoc::openapi());
        doc.merge(features::datasets::handler::ApiDoc::openapi());
        doc.merge(features::sql::handler::ApiDoc::openapi());
        doc
    }
}

struct BearerAuth;

impl Modify for BearerAuth {
    fn modify(&self, openapi: &mut utoipa::openapi::OpenApi) {
        if let Some(components) = openapi.components.as_mut() {
            components.add_security_scheme(
                "bearer_auth",
                SecurityScheme::Http(
                    HttpBuilder::new()
                        .scheme(HttpAuthScheme::Bearer)
                        .bearer_format("JWT")
                        .build(),
                ),
            );
        }
    }
}
