use std::collections::HashMap;

use crate::chunk::Value;
use crate::connectors::extractor::compute_min_max;
use crate::data_type::{DataType, DataValue};
use crate::frame::Frame;
use crate::store::{Column, Series};

use super::relationship::{JoinType, Relationship};

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/// Build a flat, denormalized frame by following every relationship that
/// starts from `base`.  The result is stored under `result_name`.
///
/// For each relationship `base.fk → dim.pk`:
/// - All columns of `dim` (except `pk`) are appended as `{dim_name}_{col}`.
/// - `Left`  join: unmatched fact rows get `Null` in the added columns.
/// - `Inner` join: unmatched fact rows are dropped from the result entirely.
///
/// Dict-encoded columns are decoded during resolution; the output frame uses
/// plain (uncompressed) encoding so it is trivial to inspect and serialize.
pub fn resolve_frame(
    base: &Frame,
    relationships: &[Relationship],
    all_frames: &HashMap<String, Frame>,
    result_name: String,
) -> Frame {
    let row_count = base.columns.first().map(|s| s.len).unwrap_or(0);

    // Materialise every base column into a plain Vec<DataValue>.
    let mut out: Vec<PlainCol> = base
        .columns
        .iter()
        .map(|s| PlainCol {
            name:   s.field.name.clone(),
            dtype:  s.field.dtype.clone(),
            values: decode_series(s),
        })
        .collect();

    // Track which rows to keep (for Inner joins).
    let mut keep: Vec<bool> = vec![true; row_count];

    for rel in relationships.iter().filter(|r| r.from_frame == base.name) {
        let dim = match all_frames.get(&rel.to_frame) {
            Some(f) => f,
            None    => {
                eprintln!("[resolver] relationship '{}' references unknown frame '{}'", rel.id, rel.to_frame);
                continue;
            }
        };

        let fk_series = match base.columns.iter().find(|s| s.field.name == rel.from_col) {
            Some(s) => s,
            None    => {
                eprintln!("[resolver] FK column '{}.{}' not found", rel.from_frame, rel.from_col);
                continue;
            }
        };

        let pk_series = match dim.columns.iter().find(|s| s.field.name == rel.to_col) {
            Some(s) => s,
            None    => {
                eprintln!("[resolver] PK column '{}.{}' not found", rel.to_frame, rel.to_col);
                continue;
            }
        };

        // Build pk_value → row index lookup for the dimension frame.
        let lookup = build_lookup(pk_series);

        // Prepare one output vec per dimension column we want to add.
        let dim_cols: Vec<&Series> = dim
            .columns
            .iter()
            .filter(|s| s.field.name != rel.to_col)
            .collect();

        let mut dim_vecs: Vec<Vec<DataValue>> = dim_cols
            .iter()
            .map(|_| Vec::with_capacity(row_count))
            .collect();

        for i in 0..row_count {
            if !keep[i] {
                // Row already dropped by a previous Inner join — push placeholders.
                for v in dim_vecs.iter_mut() { v.push(DataValue::Null); }
                continue;
            }

            let fk_val  = value_at(fk_series, i);
            let key     = dv_key(&fk_val);
            let matched = lookup.get(&key);

            match matched {
                Some(&j) => {
                    for (v, col) in dim_vecs.iter_mut().zip(dim_cols.iter()) {
                        v.push(value_at(col, j));
                    }
                }
                None => {
                    match rel.join_type {
                        JoinType::Left  => {
                            for v in dim_vecs.iter_mut() { v.push(DataValue::Null); }
                        }
                        JoinType::Inner => {
                            keep[i] = false;
                            for v in dim_vecs.iter_mut() { v.push(DataValue::Null); }
                        }
                    }
                }
            }
        }

        // Register the new dimension columns.
        for (col, values) in dim_cols.iter().zip(dim_vecs) {
            out.push(PlainCol {
                name:   format!("{}_{}", rel.to_frame, col.field.name),
                dtype:  col.field.dtype.clone(),
                values,
            });
        }
    }

    // Apply keep-mask (only relevant when at least one Inner join dropped rows).
    let final_out: Vec<PlainCol> = if keep.iter().all(|&k| k) {
        out
    } else {
        out.into_iter()
            .map(|mut c| {
                c.values = c.values.into_iter().enumerate()
                    .filter(|(i, _)| keep[*i])
                    .map(|(_, v)| v)
                    .collect();
                c
            })
            .collect()
    };

    let columns = final_out.into_iter().map(plain_col_to_series).collect();
    Frame { name: result_name, columns }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

struct PlainCol {
    name:   String,
    dtype:  DataType,
    values: Vec<DataValue>,
}

/// Decode every value in a series to its actual `DataValue` (resolves dict).
fn decode_series(series: &Series) -> Vec<DataValue> {
    (0..series.len).map(|i| value_at(series, i)).collect()
}

/// Return the real value at row `i` — deref dict encoding if present.
fn value_at(series: &Series, i: usize) -> DataValue {
    match series.chunks.get(i).map(|c| &c.data) {
        None | Some(DataValue::Null) => DataValue::Null,
        Some(DataValue::Int64(idx)) if !series.dict.is_empty() => {
            series.dict.get(idx).cloned().unwrap_or(DataValue::Null)
        }
        Some(v) => v.clone(),
    }
}

/// Stable string key for a `DataValue` used in HashMap lookups.
fn dv_key(v: &DataValue) -> String {
    format!("{v:?}")
}

/// Build a `pk_value_key → row_index` map for fast join lookups.
fn build_lookup(series: &Series) -> HashMap<String, usize> {
    let mut map = HashMap::with_capacity(series.len);
    for i in 0..series.len {
        let v = value_at(series, i);
        if !matches!(v, DataValue::Null) {
            map.entry(dv_key(&v)).or_insert(i);
        }
    }
    map
}

/// Convert a `PlainCol` into a `Series` with no dict encoding.
fn plain_col_to_series(col: PlainCol) -> Series {
    let len        = col.values.len();
    let null_count = col.values.iter().filter(|v| matches!(v, DataValue::Null)).count();
    let chunks: Vec<Value> = col.values.into_iter()
        .map(|d| Value { data: d, validity: vec![] })
        .collect();
    let (min, max) = compute_min_max(&chunks, &HashMap::new());
    Series {
        field:      Column { name: col.name, dtype: col.dtype },
        chunks,
        len,
        null_count,
        dict:       HashMap::new(),
        min,
        max,
    }
}
