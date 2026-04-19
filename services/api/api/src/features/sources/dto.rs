use serde::Deserialize;
use serde_json::Value;
use utoipa::ToSchema;

#[derive(Deserialize, ToSchema)]
pub struct CreateSourceRequest {
    /// Unique name within the workspace — becomes the frame name on load
    pub name:        String,
    /// `csv` or `postgres`
    pub source_type: String,
    pub config:      Value,
}
