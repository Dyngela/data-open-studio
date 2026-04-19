use serde::Deserialize;
use serde_json::Value;

#[derive(Deserialize)]
pub struct CreateTriggerReq {
    pub name:             String,
    pub description:      Option<String>,
    #[serde(rename = "type")]
    pub trigger_type:     String,
    pub polling_interval: Option<i32>,
    pub config:           Option<Value>,
}

#[derive(Deserialize)]
pub struct UpdateTriggerReq {
    pub name:             Option<String>,
    pub description:      Option<String>,
    pub polling_interval: Option<i32>,
    pub config:           Option<Value>,
}

#[derive(Deserialize)]
pub struct CreateRuleReq {
    pub name:       Option<String>,
    pub conditions: Value,
}

#[derive(Deserialize)]
pub struct UpdateRuleReq {
    pub name:       Option<String>,
    pub conditions: Option<Value>,
}

#[derive(Deserialize)]
pub struct LinkJobReq {
    pub job_id:          i64,
    pub priority:        Option<i32>,
    pub pass_event_data: Option<bool>,
}

#[derive(Deserialize)]
pub struct ExecutionsQuery {
    pub limit: Option<i64>,
}
