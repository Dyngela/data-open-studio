use std::fs::File;
use std::io::{BufRead, BufReader};

use crate::connectors::ConnectorError;
use crate::connectors::extractor::{build_series, InferDtype};
use crate::data_type::{DataType, DataValue};
use crate::frame::Frame;
use crate::store::Series;

pub struct CsvConfig {
    pub path:       String,
    pub delimiter:  u8,
    pub has_header: bool,
    pub frame_name: String,
}

pub fn load_csv(config: CsvConfig, cedrus: &mut crate::cedrus::Cedrus) -> Result<(), ConnectorError> {
    let file   = File::open(&config.path)?;
    let reader = BufReader::new(file);
    let mut lines = reader.lines();

    let header = lines
        .next()
        .ok_or(ConnectorError::ParsingError("Empty file".to_string()))?
        .map_err(|e| ConnectorError::ParsingError(e.to_string()))?;

    let col_names: Vec<&str> = header.split(config.delimiter as char).collect();
    let n_cols = col_names.len();

    let mut col_raw: Vec<Vec<String>> = vec![vec![]; n_cols];
    for line in lines {
        let line = line.map_err(|e| ConnectorError::ParsingError(e.to_string()))?;
        if line.is_empty() { continue; }
        for (i, val) in line.split(config.delimiter as char).enumerate() {
            if i < n_cols { col_raw[i].push(val.trim().to_string()); }
        }
    }

    let columns: Vec<Series> = col_names
        .iter()
        .zip(col_raw)
        .map(|(name, raw)| infer_series(name, raw))
        .collect();

    cedrus.write(&Frame { columns, name: config.frame_name })
        .map_err(ConnectorError::ArbitraryError)
}

fn infer_series(name: &str, raw: Vec<String>) -> Series {
    let dtype = raw.infer_dtype();
    let data: Vec<Option<DataValue>> = raw.iter()
        .map(|s| if s.is_empty() { None } else { Some(parse_value(s, &dtype)) })
        .collect();
    build_series(name, dtype, data)
}

fn parse_value(s: &str, dtype: &DataType) -> DataValue {
    match dtype {
        DataType::Boolean => DataValue::Boolean(matches!(s.to_lowercase().as_str(), "true" | "1")),
        DataType::Int8    => DataValue::Int8(s.parse().unwrap_or(0)),
        DataType::Int16   => DataValue::Int16(s.parse().unwrap_or(0)),
        DataType::Int32   => DataValue::Int32(s.parse().unwrap_or(0)),
        DataType::Int64   => DataValue::Int64(s.parse().unwrap_or(0)),
        DataType::UInt8   => DataValue::UInt8(s.parse().unwrap_or(0)),
        DataType::UInt16  => DataValue::UInt16(s.parse().unwrap_or(0)),
        DataType::UInt32  => DataValue::UInt32(s.parse().unwrap_or(0)),
        DataType::Float32 => DataValue::Float32(s.parse().unwrap_or(0.0)),
        DataType::Float64 => DataValue::Float64(s.parse().unwrap_or(0.0)),
        _                 => DataValue::String(s.to_string()),
    }
}
