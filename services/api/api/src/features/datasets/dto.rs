use serde::{Deserialize, Serialize};
use serde_json::Value;

#[derive(Deserialize)]
pub struct CreateDatasetReq {
    pub name:                 String,
    pub description:          Option<String>,
    pub metadata_database_id: i64,
    pub query:                String,
}

#[derive(Deserialize)]
pub struct UpdateDatasetReq {
    pub name:                 Option<String>,
    pub description:          Option<String>,
    pub metadata_database_id: Option<i64>,
    pub query:                Option<String>,
}

#[derive(Deserialize)]
pub struct PreviewReq {
    pub limit: Option<i64>,
}

#[derive(Deserialize)]
pub struct QueryFilter {
    pub column:   String,
    pub operator: String,
    pub value:    Value,
}

#[derive(Deserialize)]
pub struct QueryReq {
    pub filters: Option<Vec<QueryFilter>>,
    pub limit:   Option<i64>,
}

#[derive(Deserialize)]
pub struct LoadAsFrameReq {
    pub workspace_id: uuid::Uuid,
    pub frame_name:   Option<String>,
}

#[derive(Serialize)]
pub struct DatasetSchema {
    pub columns: Vec<DatasetColumn>,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct DatasetColumn {
    pub name:      String,
    pub data_type: String,
    pub nullable:  bool,
}
