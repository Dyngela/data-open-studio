use serde::Deserialize;
use utoipa::ToSchema;

#[derive(Deserialize, ToSchema)]
pub struct ExecuteRequest {
    pub script: String,
    pub result: Option<String>,
    #[serde(default = "default_limit")]
    pub limit:  usize,
}

fn default_limit() -> usize { 100 }
