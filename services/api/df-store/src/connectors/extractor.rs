use std::collections::{HashMap, HashSet};

use crate::chunk::Value;
use crate::data_type::{DataType, DataValue};
use crate::store::{Column, Series};

// ---------------------------------------------------------------------------
// InferDtype trait
// ---------------------------------------------------------------------------

/// Any column source that can tell us its DataType implements this trait.
/// Each connector (CSV, Postgres, Excel) will implement it for its own raw type.
pub trait InferDtype {
    fn infer_dtype(&self) -> DataType;
}

impl InferDtype for Vec<String> {
    fn infer_dtype(&self) -> DataType {
        let non_empty: Vec<&str> = self.iter()
            .map(|s| s.as_str())
            .filter(|s| !s.is_empty())
            .collect();

        if non_empty.is_empty() { return DataType::String; }
        if is_boolean(&non_empty) { return DataType::Boolean; }
        if is_int64(&non_empty)   { return DataType::Int64; }
        if is_float64(&non_empty) { return DataType::Float64; }
        DataType::String
    }
}

fn is_boolean(values: &[&str]) -> bool {
    values.iter().all(|v| matches!(v.to_lowercase().as_str(), "true" | "false" | "1" | "0"))
}

fn is_int64(values: &[&str]) -> bool {
    values.iter().all(|v| v.parse::<i64>().is_ok())
}

fn is_float64(values: &[&str]) -> bool {
    values.iter().all(|v| v.parse::<f64>().is_ok())
}

impl InferDtype for Vec<i64>  { fn infer_dtype(&self) -> DataType { DataType::Int64 } }
impl InferDtype for Vec<f64>  { fn infer_dtype(&self) -> DataType { DataType::Float64 } }
impl InferDtype for Vec<bool> { fn infer_dtype(&self) -> DataType { DataType::Boolean } }
impl InferDtype for Vec<i32>  { fn infer_dtype(&self) -> DataType { DataType::Int32 } }

// ---------------------------------------------------------------------------
// Shared series-building helpers
// ---------------------------------------------------------------------------

/// Build a `Series` from already-typed optional values.
/// Handles dict encoding, null counting, and min/max in one pass.
pub fn build_series(name: &str, dtype: DataType, data: Vec<Option<DataValue>>) -> Series {
    let len        = data.len();
    let null_count = data.iter().filter(|v| v.is_none()).count();
    let dict       = if is_dict_eligible(&dtype) { build_dict(&data) } else { HashMap::new() };

    let chunks: Vec<Value> = data.iter().map(|opt| {
        let dv = match opt {
            None                        => DataValue::Null,
            Some(v) if !dict.is_empty() => DataValue::Int64(dict_index(&dict, v)),
            Some(v)                     => v.clone(),
        };
        Value { data: dv, validity: vec![] }
    }).collect();

    let (min, max) = compute_min_max(&chunks, &dict);
    Series { field: Column { name: name.to_string(), dtype }, chunks, len, null_count, dict, min, max }
}

/// Types where dictionary encoding makes sense (low cardinality expected).
pub fn is_dict_eligible(dtype: &DataType) -> bool {
    matches!(dtype,
        DataType::String
        | DataType::Boolean
        | DataType::Date
        | DataType::Int8  | DataType::Int16  | DataType::Int32
        | DataType::UInt8 | DataType::UInt16 | DataType::UInt32
    )
}

/// Build a `{index → DataValue}` dict; returns empty map if unique values
/// exceed 1/3 of non-null rows (not worth encoding).
pub fn build_dict(data: &[Option<DataValue>]) -> HashMap<i64, DataValue> {
    let mut seen:   HashSet<String> = HashSet::new();
    let mut unique: Vec<DataValue>  = Vec::new();
    for v in data.iter().flatten() {
        if seen.insert(format!("{:?}", v)) { unique.push(v.clone()); }
    }
    let non_null = data.iter().filter(|v| v.is_some()).count();
    if non_null == 0 || unique.len() > non_null / 3 { return HashMap::new(); }
    unique.into_iter().enumerate().map(|(i, v)| (i as i64, v)).collect()
}

/// Look up the dict index for a value (falls back to 0 if missing).
pub fn dict_index(dict: &HashMap<i64, DataValue>, value: &DataValue) -> i64 {
    let target = format!("{:?}", value);
    dict.iter()
        .find(|(_, v)| format!("{:?}", v) == target)
        .map(|(k, _)| *k)
        .unwrap_or(0)
}

/// Compute min and max over non-null values.
/// For dict-encoded columns uses dict values; for plain columns scans chunks.
pub fn compute_min_max(chunks: &[Value], dict: &HashMap<i64, DataValue>) -> (Option<DataValue>, Option<DataValue>) {
    let values: Vec<&DataValue> = if !dict.is_empty() {
        dict.values().collect()
    } else {
        chunks.iter()
            .filter(|v| !matches!(v.data, DataValue::Null))
            .map(|v| &v.data)
            .collect()
    };

    let mut min: Option<&DataValue> = None;
    let mut max: Option<&DataValue> = None;
    for v in &values {
        if min.is_none() || dv_lt(v, min.unwrap()) { min = Some(v); }
        if max.is_none() || dv_lt(max.unwrap(), v) { max = Some(v); }
    }
    (min.cloned(), max.cloned())
}

/// Less-than comparison for same-type `DataValue` pairs.
pub fn dv_lt(a: &DataValue, b: &DataValue) -> bool {
    match (a, b) {
        (DataValue::Int8(x),    DataValue::Int8(y))    => x < y,
        (DataValue::Int16(x),   DataValue::Int16(y))   => x < y,
        (DataValue::Int32(x),   DataValue::Int32(y))   => x < y,
        (DataValue::Int64(x),   DataValue::Int64(y))   => x < y,
        (DataValue::UInt8(x),   DataValue::UInt8(y))   => x < y,
        (DataValue::UInt16(x),  DataValue::UInt16(y))  => x < y,
        (DataValue::UInt32(x),  DataValue::UInt32(y))  => x < y,
        (DataValue::UInt64(x),  DataValue::UInt64(y))  => x < y,
        (DataValue::Float32(x), DataValue::Float32(y)) => x < y,
        (DataValue::Float64(x), DataValue::Float64(y)) => x < y,
        (DataValue::String(x),  DataValue::String(y))  => x < y,
        (DataValue::Boolean(x), DataValue::Boolean(y)) => !x & y,
        (DataValue::Date(x),    DataValue::Date(y))    => x < y,
        (DataValue::Time(x),    DataValue::Time(y))    => x < y,
        (DataValue::Datetime(x, _, _), DataValue::Datetime(y, _, _)) => x < y,
        _ => false,
    }
}
