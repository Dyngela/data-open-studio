/// Helpers for serialising `df_store` frames to JSON-friendly values.
use df_store::data_type::{DataType, DataValue};
use df_store::frame::Frame;
use df_store::store::Series;
use serde_json::{json, Value};

pub fn frame_row_count(frame: &Frame) -> usize {
    frame.columns.first().map(|s| s.len).unwrap_or(0)
}

pub fn frame_schema_json(frame: &Frame) -> Value {
    let columns: Vec<Value> = frame.columns.iter().map(|s| json!({
        "name":       s.field.name,
        "dtype":      dtype_str(&s.field.dtype),
        "len":        s.len,
        "null_count": s.null_count,
    })).collect();
    json!({
        "name":      frame.name,
        "row_count": frame_row_count(frame),
        "columns":   columns,
    })
}

/// Return a paginated column-oriented JSON view of a frame.
pub fn frame_data_json(frame: &Frame, offset: usize, limit: usize) -> Value {
    let row_count = frame_row_count(frame);
    let offset    = offset.min(row_count);
    let end       = (offset + limit).min(row_count);

    let mut columns = serde_json::Map::new();
    for series in &frame.columns {
        let values: Vec<Value> = (offset..end).map(|i| series_get(series, i)).collect();
        columns.insert(series.field.name.clone(), Value::Array(values));
    }

    json!({
        "name":      frame.name,
        "offset":    offset,
        "limit":     end - offset,
        "row_count": row_count,
        "columns":   columns,
    })
}

pub fn series_get(series: &Series, i: usize) -> Value {
    match series.chunks.get(i) {
        None        => Value::Null,
        Some(chunk) => match &chunk.data {
            DataValue::Null => Value::Null,
            DataValue::Int64(idx) if !series.dict.is_empty() => {
                series.dict.get(idx).map(dv_to_json).unwrap_or(Value::Null)
            }
            other => dv_to_json(other),
        },
    }
}

pub fn dv_to_json(v: &DataValue) -> Value {
    match v {
        DataValue::Null              => Value::Null,
        DataValue::Boolean(b)        => json!(b),
        DataValue::Int8(n)           => json!(n),
        DataValue::Int16(n)          => json!(n),
        DataValue::Int32(n)          => json!(n),
        DataValue::Int64(n)          => json!(n),
        DataValue::UInt8(n)          => json!(n),
        DataValue::UInt16(n)         => json!(n),
        DataValue::UInt32(n)         => json!(n),
        DataValue::UInt64(n)         => json!(n),
        DataValue::Float32(f)        => if f.is_finite() { json!(f) } else { Value::Null },
        DataValue::Float64(f)        => if f.is_finite() { json!(f) } else { Value::Null },
        DataValue::String(s)         => json!(s),
        DataValue::Date(d)           => json!(d),
        DataValue::Time(t)           => json!(t),
        DataValue::Datetime(ts, _, tz) => json!({ "ts": ts, "tz": tz }),
        DataValue::Binary(_)         => Value::Null,
        DataValue::Duration(d, _)    => json!(d),
        DataValue::List(_)           => Value::Null,
        DataValue::Struct(_)         => Value::Null,
    }
}

pub fn dtype_str(dt: &DataType) -> &'static str {
    match dt {
        DataType::Boolean  => "Boolean",
        DataType::UInt8    => "UInt8",
        DataType::UInt16   => "UInt16",
        DataType::UInt32   => "UInt32",
        DataType::UInt64   => "UInt64",
        DataType::Int8     => "Int8",
        DataType::Int16    => "Int16",
        DataType::Int32    => "Int32",
        DataType::Int64    => "Int64",
        DataType::Float32  => "Float32",
        DataType::Float64  => "Float64",
        DataType::String   => "String",
        DataType::Binary   => "Binary",
        DataType::Date     => "Date",
        DataType::Time     => "Time",
        DataType::Datetime => "Datetime",
        DataType::Duration => "Duration",
        DataType::List     => "List",
        DataType::Struct   => "Struct",
        DataType::Null     => "Null",
    }
}
