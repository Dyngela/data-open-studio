use serde::{Deserialize, Serialize};

/// A directed relationship between two frames: many (from) → one (to).
/// Mirrors Power BI's relationship model.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Relationship {
    /// Unique name for this relationship (user-provided or auto-generated).
    pub id: String,

    /// Fact-side frame (holds the foreign key).
    pub from_frame: String,
    /// Column in the fact frame that contains the foreign key values.
    pub from_col: String,

    /// Dimension-side frame (holds the primary key).
    pub to_frame: String,
    /// Column in the dimension frame that the FK references.
    pub to_col: String,

    /// How to handle rows in the fact frame with no matching dimension row.
    #[serde(default)]
    pub join_type: JoinType,
}

impl Relationship {
    /// Canonical ID derived from the column endpoints.
    pub fn auto_id(from_frame: &str, from_col: &str, to_frame: &str, to_col: &str) -> String {
        format!("{from_frame}.{from_col}→{to_frame}.{to_col}")
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, Default, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum JoinType {
    /// Keep all fact rows; fill unmatched dimension columns with null.
    #[default]
    Left,
    /// Only keep fact rows that have a matching dimension row.
    Inner,
}
