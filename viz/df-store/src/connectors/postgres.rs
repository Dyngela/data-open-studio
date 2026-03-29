use chrono::NaiveDate;
use postgres::types::Type;

use crate::cedrus::Cedrus;
use crate::connectors::ConnectorError;
use crate::connectors::extractor::build_series;
use crate::data_type::{DataType, DataValue};
use crate::frame::Frame;
use crate::store::Series;
use crate::time::TimeUnit;

pub struct PostgresConfig {
    pub host:       String,
    pub port:       u16,
    pub username:   String,
    pub password:   String,
    pub database:   String,
    pub query:      String,
    pub frame_name: String,
}

pub fn load_postgres(config: PostgresConfig, cedrus: &mut Cedrus) -> Result<(), ConnectorError> {
    let conn_str = format!(
        "host={} port={} user={} password={} dbname={}",
        config.host, config.port, config.username, config.password, config.database
    );
    let mut client = postgres::Client::connect(&conn_str, postgres::NoTls)
        .map_err(|e| ConnectorError::ConnectionError(e.to_string()))?;

    let rows = client
        .query(&config.query as &str, &[])
        .map_err(|e| ConnectorError::ArbitraryError(e.to_string()))?;

    if rows.is_empty() {
        return cedrus
            .write(&Frame { name: config.frame_name, columns: vec![] })
            .map_err(ConnectorError::ArbitraryError);
    }

    let col_meta: Vec<(String, Type)> = rows[0]
        .columns()
        .iter()
        .map(|c| (c.name().to_string(), c.type_().clone()))
        .collect();
    let n_cols = col_meta.len();
    let n_rows = rows.len();

    let mut col_data: Vec<Vec<Option<DataValue>>> = vec![Vec::with_capacity(n_rows); n_cols];
    for row in &rows {
        for (i, (_, t)) in col_meta.iter().enumerate() {
            col_data[i].push(pg_to_data_value(row, i, t)?);
        }
    }

    let columns: Vec<Series> = col_meta
        .iter()
        .zip(col_data)
        .map(|((name, t), data)| build_series(name, pg_type_to_dtype(t), data))
        .collect();

    cedrus
        .write(&Frame { name: config.frame_name, columns })
        .map_err(ConnectorError::ArbitraryError)
}

fn pg_type_to_dtype(t: &Type) -> DataType {
    match *t {
        Type::BOOL                                                          => DataType::Boolean,
        Type::INT2                                                          => DataType::Int16,
        Type::INT4                                                          => DataType::Int32,
        Type::INT8                                                          => DataType::Int64,
        Type::FLOAT4                                                        => DataType::Float32,
        Type::FLOAT8 | Type::NUMERIC                                        => DataType::Float64,
        Type::TEXT | Type::VARCHAR | Type::BPCHAR | Type::NAME | Type::UUID => DataType::String,
        Type::DATE                                                          => DataType::Date,
        Type::TIMESTAMP | Type::TIMESTAMPTZ                                 => DataType::Datetime,
        Type::BYTEA                                                         => DataType::Binary,
        _                                                                   => DataType::String,
    }
}

fn pg_to_data_value(row: &postgres::Row, idx: usize, t: &Type) -> Result<Option<DataValue>, ConnectorError> {
    let e = |err: postgres::Error| ConnectorError::ParsingError(err.to_string());

    Ok(match *t {
        Type::BOOL   => row.try_get::<_, Option<bool>>(idx).map_err(e)?.map(DataValue::Boolean),
        Type::INT2   => row.try_get::<_, Option<i16>>(idx).map_err(e)?.map(DataValue::Int16),
        Type::INT4   => row.try_get::<_, Option<i32>>(idx).map_err(e)?.map(DataValue::Int32),
        Type::INT8   => row.try_get::<_, Option<i64>>(idx).map_err(e)?.map(DataValue::Int64),
        Type::FLOAT4 => row.try_get::<_, Option<f32>>(idx).map_err(e)?.map(DataValue::Float32),
        Type::FLOAT8 | Type::NUMERIC => {
            row.try_get::<_, Option<f64>>(idx).map_err(e)?.map(DataValue::Float64)
        }
        Type::DATE => {
            let epoch = NaiveDate::from_ymd_opt(1970, 1, 1).unwrap();
            row.try_get::<_, Option<NaiveDate>>(idx)
                .map_err(e)?
                .map(|d| DataValue::Date((d - epoch).num_days() as i32))
        }
        Type::TIMESTAMP => {
            row.try_get::<_, Option<chrono::NaiveDateTime>>(idx)
                .map_err(e)?
                .map(|dt| DataValue::Datetime(dt.and_utc().timestamp_millis(), TimeUnit::Milliseconds, None))
        }
        Type::TIMESTAMPTZ => {
            row.try_get::<_, Option<chrono::DateTime<chrono::Utc>>>(idx)
                .map_err(e)?
                .map(|dt| DataValue::Datetime(dt.timestamp_millis(), TimeUnit::Milliseconds, Some("UTC".to_string())))
        }
        Type::BYTEA => {
            row.try_get::<_, Option<Vec<u8>>>(idx).map_err(e)?.map(DataValue::Binary)
        }
        _ => row.try_get::<_, Option<String>>(idx).map_err(e)?.map(DataValue::String),
    })
}
