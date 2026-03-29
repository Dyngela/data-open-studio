use std::collections::HashMap;
use std::fs;
use std::fs::File;
use std::io::{BufWriter, Write};

use crate::chunk::Value;
use crate::cedrus::{Cedrus, ColumnChunk, ColumnMetadata, ColumnSchema, Encoding, RowGroup, RowGroupMetadata};
use crate::data_type::{DataType, DataValue};
use crate::frame::Frame;
use crate::store::Series;

const ROW_GROUP_SIZE: usize = 65_536 * 3;

impl Cedrus {
    pub fn write(&mut self, frame: &Frame) -> Result<(), String> {
        let dir = crate::cedrus::store_path();
        fs::create_dir_all(&dir).map_err(|e| e.to_string())?;
        let path = dir.join(format!("{}.cedr", frame.name));
        let file = File::create(&path).map_err(|e| e.to_string())?;
        let mut w = BufWriter::new(file);

        // --- Header: magic + version ---
        w.write_all(b"CEDR").map_err(|e| e.to_string())?;
        w.write_all(&1u16.to_le_bytes()).map_err(|e| e.to_string())?;

        let total_rows = frame.columns.first().map(|s| s.len).unwrap_or(0);
        let mut byte_offset: u64 = 6; // 4 (magic) + 2 (version)
        let mut rg_metas: Vec<RowGroupMetadata> = Vec::new();

        self.schema = frame.columns.iter().map(|s| ColumnSchema {
            name: s.field.name.clone(),
            data_type: s.field.dtype.clone(),
            nullable: s.null_count > 0,
        }).collect();

        // --- Row groups ---
        let mut row_start = 0;
        while row_start < total_rows {
            let row_end = (row_start + ROW_GROUP_SIZE).min(total_rows);
            let mut col_metas: Vec<ColumnMetadata> = Vec::new();
            let mut rg_chunks: Vec<ColumnChunk> = Vec::new();

            for series in &frame.columns {
                let is_dict = !series.dict.is_empty();
                let encoding = if is_dict { Encoding::RleDictionary } else { Encoding::Plain };
                let mut null_count: u64 = 0;
                let mut col_bytes: Vec<u8> = Vec::new();

                for value in &series.chunks[row_start..row_end] {
                    let is_null = matches!(value.data, DataValue::Null);
                    col_bytes.push(!is_null as u8); // 1 = valid, 0 = null

                    if is_null {
                        null_count += 1;
                        write_null_placeholder(&series.field.dtype, is_dict, &mut col_bytes);
                    } else {
                        encode_value(&value.data, &mut col_bytes)?;
                    }
                }

                let length = col_bytes.len() as u64;
                w.write_all(&col_bytes).map_err(|e| e.to_string())?;

                col_metas.push(ColumnMetadata {
                    encoding,
                    offset: byte_offset,
                    length,
                    null_count,
                    min_value: series.min.clone().unwrap_or(DataValue::Null),
                    max_value: series.max.clone().unwrap_or(DataValue::Null),
                });
                byte_offset += length;
                rg_chunks.push(ColumnChunk { data: col_bytes });
            }

            rg_metas.push(RowGroupMetadata {
                num_rows: (row_end - row_start) as u64,
                columns: col_metas,
            });
            self.row_groups.push(RowGroup {
                columns: rg_chunks,
                num_rows: (row_end - row_start) as u64,
            });

            row_start = row_end;
        }

        // --- Footer ---
        let mut footer: Vec<u8> = Vec::new();
        write_footer(&mut footer, total_rows as u64, &self.schema, &frame.columns, &rg_metas);

        let footer_len = footer.len() as u32;
        w.write_all(&footer).map_err(|e| e.to_string())?;
        w.write_all(&footer_len.to_le_bytes()).map_err(|e| e.to_string())?;
        w.write_all(b"CEDR").map_err(|e| e.to_string())?; // end marker

        Ok(())
    }

    /// Read a Frame from a .cedr file in the store directory.
    pub fn read(name: &str) -> Result<Frame, String> {
        let path = crate::cedrus::store_path().join(format!("{}.cedr", name));
        let bytes = fs::read(&path).map_err(|e| e.to_string())?;

        if bytes.len() < 10 {
            return Err("File too small".to_string());
        }
        if &bytes[0..4] != b"CEDR" {
            return Err("Invalid start magic".to_string());
        }
        if &bytes[bytes.len() - 4..] != b"CEDR" {
            return Err("Invalid end magic".to_string());
        }

        let end = bytes.len();
        let footer_len   = u32::from_le_bytes(bytes[end-8..end-4].try_into().unwrap()) as usize;
        let footer_start = end - 8 - footer_len;
        let footer       = &bytes[footer_start..footer_start + footer_len];

        let mut pos = 0usize;
        let _total_rows = read_u64(footer, &mut pos);
        let num_columns = read_u32(footer, &mut pos) as usize;

        let mut col_names:  Vec<String>                  = Vec::new();
        let mut col_dtypes: Vec<DataType>                = Vec::new();
        let mut col_dicts:  Vec<HashMap<i64, DataValue>> = Vec::new();

        for _ in 0..num_columns {
            let name_len  = read_u32(footer, &mut pos) as usize;
            let col_name  = String::from_utf8(footer[pos..pos + name_len].to_vec())
                .map_err(|e| e.to_string())?;
            pos += name_len;
            let dtype     = byte_to_dtype(footer[pos]); pos += 1;
            let _nullable = footer[pos];                pos += 1;
            let has_dict  = footer[pos] != 0;           pos += 1;

            let mut dict: HashMap<i64, DataValue> = HashMap::new();
            if has_dict {
                let dict_len = read_u32(footer, &mut pos) as usize;
                for _ in 0..dict_len {
                    let key   = read_i64(footer, &mut pos);
                    let value = read_data_value(footer, &mut pos)?;
                    dict.insert(key, value);
                }
            }
            col_names.push(col_name);
            col_dtypes.push(dtype);
            col_dicts.push(dict);
        }

        let num_rgs = read_u32(footer, &mut pos) as usize;
        let mut rg_col_metas:  Vec<Vec<(u8, usize, usize, u64)>> = Vec::new();
        let mut rg_row_counts: Vec<usize>                         = Vec::new();

        for _ in 0..num_rgs {
            let rg_rows = read_u64(footer, &mut pos) as usize;
            rg_row_counts.push(rg_rows);
            let mut cols = Vec::new();
            for _ in 0..num_columns {
                let encoding   = footer[pos]; pos += 1;
                let offset     = read_u64(footer, &mut pos) as usize;
                let length     = read_u64(footer, &mut pos) as usize;
                let null_count = read_u64(footer, &mut pos);
                cols.push((encoding, offset, length, null_count));
            }
            rg_col_metas.push(cols);
        }

        let mut all_chunks:  Vec<Vec<Value>> = (0..num_columns).map(|_| Vec::new()).collect();
        let mut null_counts: Vec<usize>      = vec![0; num_columns];

        for (rg_idx, col_metas) in rg_col_metas.iter().enumerate() {
            let rg_rows = rg_row_counts[rg_idx];
            for (col_idx, &(_, offset, length, _)) in col_metas.iter().enumerate() {
                let col_bytes = &bytes[offset..offset + length];
                let is_dict   = !col_dicts[col_idx].is_empty();
                let mut values = decode_column(col_bytes, &col_dtypes[col_idx], is_dict, rg_rows)?;
                null_counts[col_idx] += values.iter().filter(|v| matches!(v.data, DataValue::Null)).count();
                all_chunks[col_idx].append(&mut values);
            }
        }

        let columns: Vec<Series> = (0..num_columns).map(|i| {
            let len = all_chunks[i].len();
            Series {
                field:      crate::store::Column { name: col_names[i].clone(), dtype: col_dtypes[i].clone() },
                chunks:     std::mem::take(&mut all_chunks[i]),
                len,
                null_count: null_counts[i],
                dict:       std::mem::take(&mut col_dicts[i]),
                min:        None,
                max:        None,
            }
        }).collect();

        Ok(Frame { name: name.to_string(), columns })
    }
}

// ---------------------------------------------------------------------------
// Value encoding
// ---------------------------------------------------------------------------

/// Encode a single DataValue to its raw bytes (plain encoding).
fn encode_value(value: &DataValue, buf: &mut Vec<u8>) -> Result<(), String> {
    match value {
        DataValue::Boolean(b)        => buf.push(*b as u8),
        DataValue::UInt8(v)          => buf.push(*v),
        DataValue::UInt16(v)         => buf.extend_from_slice(&v.to_le_bytes()),
        DataValue::UInt32(v)         => buf.extend_from_slice(&v.to_le_bytes()),
        DataValue::UInt64(v)         => buf.extend_from_slice(&v.to_le_bytes()),
        DataValue::Int8(v)           => buf.extend_from_slice(&v.to_le_bytes()),
        DataValue::Int16(v)          => buf.extend_from_slice(&v.to_le_bytes()),
        DataValue::Int32(v)          => buf.extend_from_slice(&v.to_le_bytes()),
        DataValue::Int64(v)          => buf.extend_from_slice(&v.to_le_bytes()),
        DataValue::Float32(v)        => buf.extend_from_slice(&v.to_le_bytes()),
        DataValue::Float64(v)        => buf.extend_from_slice(&v.to_le_bytes()),
        DataValue::Date(v)           => buf.extend_from_slice(&v.to_le_bytes()),
        DataValue::Time(v)           => buf.extend_from_slice(&v.to_le_bytes()),
        DataValue::Datetime(v, _, _) => buf.extend_from_slice(&v.to_le_bytes()),
        DataValue::Duration(v, _)    => buf.extend_from_slice(&v.to_le_bytes()),
        DataValue::String(s) => {
            let bytes = s.as_bytes();
            buf.extend_from_slice(&(bytes.len() as u32).to_le_bytes());
            buf.extend_from_slice(bytes);
        }
        DataValue::Binary(b) => {
            buf.extend_from_slice(&(b.len() as u32).to_le_bytes());
            buf.extend_from_slice(b);
        }
        DataValue::Null => {}
        DataValue::List(_) | DataValue::Struct(_) => {
            return Err("List and Struct encoding not yet supported".to_string());
        }
    }
    Ok(())
}

/// Write zero bytes for null values so fixed-width columns stay aligned.
/// Variable-width types (String, Binary) write a zero length prefix instead.
fn write_null_placeholder(dtype: &DataType, is_dict: bool, buf: &mut Vec<u8>) {
    if is_dict {
        buf.extend_from_slice(&0i64.to_le_bytes()); // dict index placeholder
        return;
    }
    match dtype {
        DataType::Boolean                                          => buf.push(0),
        DataType::UInt8  | DataType::Int8                         => buf.push(0),
        DataType::UInt16 | DataType::Int16                        => buf.extend_from_slice(&[0u8; 2]),
        DataType::UInt32 | DataType::Int32
        | DataType::Float32 | DataType::Date                      => buf.extend_from_slice(&[0u8; 4]),
        DataType::UInt64 | DataType::Int64 | DataType::Float64
        | DataType::Time | DataType::Datetime | DataType::Duration => buf.extend_from_slice(&[0u8; 8]),
        DataType::String | DataType::Binary                       => buf.extend_from_slice(&0u32.to_le_bytes()),
        _ => {}
    }
}

// ---------------------------------------------------------------------------
// Footer serialization
// ---------------------------------------------------------------------------
//
// Layout:
//   [8]  total_rows
//   [4]  num_columns
//   for each column:
//     [4]  name_len  +  [..] name utf8
//     [1]  DataType byte
//     [1]  nullable
//     [1]  has_dict
//     if has_dict:
//       [4]  dict entry count
//       for each entry (sorted by key):
//         [8]  key i64  +  [1] value type byte  +  [..] value bytes
//   [4]  num_row_groups
//   for each row_group:
//     [8]  num_rows
//     for each column:
//       [1]  encoding  [8]  offset  [8]  length  [8]  null_count

fn write_footer(
    buf: &mut Vec<u8>,
    total_rows: u64,
    schema: &[ColumnSchema],
    columns: &[Series],
    rg_metas: &[RowGroupMetadata],
) {
    buf.extend_from_slice(&total_rows.to_le_bytes());
    buf.extend_from_slice(&(schema.len() as u32).to_le_bytes());

    for (col_schema, series) in schema.iter().zip(columns.iter()) {
        let name = col_schema.name.as_bytes();
        buf.extend_from_slice(&(name.len() as u32).to_le_bytes());
        buf.extend_from_slice(name);
        buf.push(dtype_to_byte(&col_schema.data_type));
        buf.push(col_schema.nullable as u8);

        if series.dict.is_empty() {
            buf.push(0); // has_dict = false
        } else {
            buf.push(1); // has_dict = true
            buf.extend_from_slice(&(series.dict.len() as u32).to_le_bytes());
            // Sort by key so the file is deterministic
            let mut entries: Vec<(&i64, &DataValue)> = series.dict.iter().collect();
            entries.sort_by_key(|(k, _)| *k);
            for (key, value) in entries {
                buf.extend_from_slice(&key.to_le_bytes());
                write_data_value(buf, value);
            }
        }
    }

    buf.extend_from_slice(&(rg_metas.len() as u32).to_le_bytes());
    for rg in rg_metas {
        buf.extend_from_slice(&rg.num_rows.to_le_bytes());
        for col in &rg.columns {
            buf.push(encoding_to_byte(&col.encoding));
            buf.extend_from_slice(&col.offset.to_le_bytes());
            buf.extend_from_slice(&col.length.to_le_bytes());
            buf.extend_from_slice(&col.null_count.to_le_bytes());
        }
    }
}

/// Serialize a DataValue with a leading type-discriminant byte.
/// Used for dict entries in the footer.
fn write_data_value(buf: &mut Vec<u8>, v: &DataValue) {
    match v {
        DataValue::Boolean(b) => { buf.push(0);  buf.push(*b as u8); }
        DataValue::UInt8(n)   => { buf.push(1);  buf.push(*n); }
        DataValue::UInt16(n)  => { buf.push(2);  buf.extend_from_slice(&n.to_le_bytes()); }
        DataValue::UInt32(n)  => { buf.push(3);  buf.extend_from_slice(&n.to_le_bytes()); }
        DataValue::UInt64(n)  => { buf.push(4);  buf.extend_from_slice(&n.to_le_bytes()); }
        DataValue::Int8(n)    => { buf.push(5);  buf.extend_from_slice(&n.to_le_bytes()); }
        DataValue::Int16(n)   => { buf.push(6);  buf.extend_from_slice(&n.to_le_bytes()); }
        DataValue::Int32(n)   => { buf.push(7);  buf.extend_from_slice(&n.to_le_bytes()); }
        DataValue::Int64(n)   => { buf.push(8);  buf.extend_from_slice(&n.to_le_bytes()); }
        DataValue::Float32(n) => { buf.push(9);  buf.extend_from_slice(&n.to_le_bytes()); }
        DataValue::Float64(n) => { buf.push(10); buf.extend_from_slice(&n.to_le_bytes()); }
        DataValue::String(s)  => {
            buf.push(11);
            let b = s.as_bytes();
            buf.extend_from_slice(&(b.len() as u32).to_le_bytes());
            buf.extend_from_slice(b);
        }
        DataValue::Date(n)    => { buf.push(13); buf.extend_from_slice(&n.to_le_bytes()); }
        DataValue::Time(n)    => { buf.push(14); buf.extend_from_slice(&n.to_le_bytes()); }
        _                     => { buf.push(19); } // Null discriminant for unsupported types
    }
}

// ---------------------------------------------------------------------------
// Byte-mapping helpers
// ---------------------------------------------------------------------------

fn dtype_to_byte(dt: &DataType) -> u8 {
    match dt {
        DataType::Boolean  => 0,
        DataType::UInt8    => 1,
        DataType::UInt16   => 2,
        DataType::UInt32   => 3,
        DataType::UInt64   => 4,
        DataType::Int8     => 5,
        DataType::Int16    => 6,
        DataType::Int32    => 7,
        DataType::Int64    => 8,
        DataType::Float32  => 9,
        DataType::Float64  => 10,
        DataType::String   => 11,
        DataType::Binary   => 12,
        DataType::Date     => 13,
        DataType::Time     => 14,
        DataType::Datetime => 15,
        DataType::Duration => 16,
        DataType::List     => 17,
        DataType::Struct   => 18,
        DataType::Null     => 19,
    }
}

fn encoding_to_byte(enc: &Encoding) -> u8 {
    match enc {
        Encoding::Plain         => 0,
        Encoding::RleDictionary => 1,
    }
}

// ---------------------------------------------------------------------------
// Decoding helpers
// ---------------------------------------------------------------------------

fn decode_column(bytes: &[u8], dtype: &DataType, is_dict: bool, num_rows: usize) -> Result<Vec<Value>, String> {
    let mut values = Vec::with_capacity(num_rows);
    let mut pos = 0usize;
    for _ in 0..num_rows {
        let is_valid = bytes[pos] != 0; pos += 1;
        if !is_valid {
            pos += null_placeholder_size(dtype, is_dict);
            values.push(Value { data: DataValue::Null, validity: vec![] });
            continue;
        }
        let data = if is_dict {
            let idx = i64::from_le_bytes(bytes[pos..pos+8].try_into().unwrap());
            pos += 8;
            DataValue::Int64(idx)
        } else {
            decode_value(bytes, dtype, &mut pos)?
        };
        values.push(Value { data, validity: vec![] });
    }
    Ok(values)
}

fn decode_value(bytes: &[u8], dtype: &DataType, pos: &mut usize) -> Result<DataValue, String> {
    macro_rules! fixed {
        ($variant:expr, $t:ty, $n:expr) => {{
            let v = <$t>::from_le_bytes(bytes[*pos..*pos+$n].try_into().unwrap());
            *pos += $n;
            Ok($variant(v))
        }};
    }
    match dtype {
        DataType::Boolean  => { let v = bytes[*pos] != 0; *pos += 1; Ok(DataValue::Boolean(v)) }
        DataType::UInt8    => { let v = bytes[*pos];      *pos += 1; Ok(DataValue::UInt8(v)) }
        DataType::Int8     => { let v = bytes[*pos] as i8; *pos += 1; Ok(DataValue::Int8(v)) }
        DataType::UInt16   => fixed!(DataValue::UInt16,  u16, 2),
        DataType::UInt32   => fixed!(DataValue::UInt32,  u32, 4),
        DataType::UInt64   => fixed!(DataValue::UInt64,  u64, 8),
        DataType::Int16    => fixed!(DataValue::Int16,   i16, 2),
        DataType::Int32    => fixed!(DataValue::Int32,   i32, 4),
        DataType::Int64    => fixed!(DataValue::Int64,   i64, 8),
        DataType::Float32  => fixed!(DataValue::Float32, f32, 4),
        DataType::Float64  => fixed!(DataValue::Float64, f64, 8),
        DataType::Date     => fixed!(DataValue::Date,    i32, 4),
        DataType::Time     => fixed!(DataValue::Time,    i64, 8),
        DataType::Datetime => { let v = i64::from_le_bytes(bytes[*pos..*pos+8].try_into().unwrap()); *pos += 8; Ok(DataValue::Datetime(v, crate::time::TimeUnit::Milliseconds, None)) }
        DataType::Duration => { let v = u64::from_le_bytes(bytes[*pos..*pos+8].try_into().unwrap()); *pos += 8; Ok(DataValue::Duration(v, crate::time::TimeUnit::Milliseconds)) }
        DataType::String | DataType::Binary => {
            let len = u32::from_le_bytes(bytes[*pos..*pos+4].try_into().unwrap()) as usize;
            *pos += 4;
            let s = String::from_utf8(bytes[*pos..*pos+len].to_vec()).map_err(|e| e.to_string())?;
            *pos += len;
            Ok(DataValue::String(s))
        }
        _ => Err(format!("Unsupported decode type: {:?}", dtype)),
    }
}

fn null_placeholder_size(dtype: &DataType, is_dict: bool) -> usize {
    if is_dict { return 8; }
    match dtype {
        DataType::Boolean | DataType::UInt8  | DataType::Int8                     => 1,
        DataType::UInt16  | DataType::Int16                                        => 2,
        DataType::UInt32  | DataType::Int32  | DataType::Float32 | DataType::Date  => 4,
        DataType::UInt64  | DataType::Int64  | DataType::Float64
        | DataType::Time  | DataType::Datetime | DataType::Duration                => 8,
        DataType::String  | DataType::Binary                                       => 4,
        _                                                                          => 0,
    }
}

fn read_u32(bytes: &[u8], pos: &mut usize) -> u32 {
    let v = u32::from_le_bytes(bytes[*pos..*pos+4].try_into().unwrap()); *pos += 4; v
}
fn read_u64(bytes: &[u8], pos: &mut usize) -> u64 {
    let v = u64::from_le_bytes(bytes[*pos..*pos+8].try_into().unwrap()); *pos += 8; v
}
fn read_i64(bytes: &[u8], pos: &mut usize) -> i64 {
    let v = i64::from_le_bytes(bytes[*pos..*pos+8].try_into().unwrap()); *pos += 8; v
}

fn byte_to_dtype(b: u8) -> DataType {
    match b {
        0  => DataType::Boolean,
        1  => DataType::UInt8,   2  => DataType::UInt16,  3  => DataType::UInt32, 4 => DataType::UInt64,
        5  => DataType::Int8,    6  => DataType::Int16,   7  => DataType::Int32,  8 => DataType::Int64,
        9  => DataType::Float32, 10 => DataType::Float64,
        11 => DataType::String,  12 => DataType::Binary,
        13 => DataType::Date,    14 => DataType::Time,
        15 => DataType::Datetime, 16 => DataType::Duration,
        17 => DataType::List,    18 => DataType::Struct,
        _  => DataType::Null,
    }
}

fn read_data_value(bytes: &[u8], pos: &mut usize) -> Result<DataValue, String> {
    let type_byte = bytes[*pos]; *pos += 1;
    match type_byte {
        0  => { let v = bytes[*pos] != 0; *pos += 1; Ok(DataValue::Boolean(v)) }
        1  => { let v = bytes[*pos];      *pos += 1; Ok(DataValue::UInt8(v)) }
        2  => { let v = u16::from_le_bytes(bytes[*pos..*pos+2].try_into().unwrap()); *pos += 2; Ok(DataValue::UInt16(v)) }
        3  => { let v = u32::from_le_bytes(bytes[*pos..*pos+4].try_into().unwrap()); *pos += 4; Ok(DataValue::UInt32(v)) }
        4  => { let v = u64::from_le_bytes(bytes[*pos..*pos+8].try_into().unwrap()); *pos += 8; Ok(DataValue::UInt64(v)) }
        5  => { let v = bytes[*pos] as i8;                                           *pos += 1; Ok(DataValue::Int8(v)) }
        6  => { let v = i16::from_le_bytes(bytes[*pos..*pos+2].try_into().unwrap()); *pos += 2; Ok(DataValue::Int16(v)) }
        7  => { let v = i32::from_le_bytes(bytes[*pos..*pos+4].try_into().unwrap()); *pos += 4; Ok(DataValue::Int32(v)) }
        8  => { let v = i64::from_le_bytes(bytes[*pos..*pos+8].try_into().unwrap()); *pos += 8; Ok(DataValue::Int64(v)) }
        9  => { let v = f32::from_le_bytes(bytes[*pos..*pos+4].try_into().unwrap()); *pos += 4; Ok(DataValue::Float32(v)) }
        10 => { let v = f64::from_le_bytes(bytes[*pos..*pos+8].try_into().unwrap()); *pos += 8; Ok(DataValue::Float64(v)) }
        11 => {
            let len = u32::from_le_bytes(bytes[*pos..*pos+4].try_into().unwrap()) as usize;
            *pos += 4;
            let s = String::from_utf8(bytes[*pos..*pos+len].to_vec()).map_err(|e| e.to_string())?;
            *pos += len;
            Ok(DataValue::String(s))
        }
        13 => { let v = i32::from_le_bytes(bytes[*pos..*pos+4].try_into().unwrap()); *pos += 4; Ok(DataValue::Date(v)) }
        14 => { let v = i64::from_le_bytes(bytes[*pos..*pos+8].try_into().unwrap()); *pos += 8; Ok(DataValue::Time(v)) }
        _  => Ok(DataValue::Null),
    }
}


#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashMap;
    use std::io::Read;
    use std::sync::Mutex;
    use crate::cedrus::{Cedrus, FileHeader};
    use crate::chunk::Value;
    use crate::data_type::{DataType, DataValue};
    use crate::frame::Frame;
    use crate::store::{Column, Series};

    // Serialize env-var writes behind a mutex so parallel tests don't stomp each other.
    static ENV_LOCK: Mutex<()> = Mutex::new(());

    fn tmp_store() -> std::path::PathBuf {
        let dir = std::path::PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .parent().unwrap()
            .join("test-files");
        fs::create_dir_all(&dir).unwrap();
        dir
    }

    fn with_tmp_store<F: FnOnce()>(f: F) {
        let _guard = ENV_LOCK.lock().unwrap();
        std::env::set_var("CEDRUS_STORE_PATH", tmp_store());
        f();
        std::env::remove_var("CEDRUS_STORE_PATH");
    }

    fn make_cedrus() -> Cedrus {
        Cedrus {
            header: FileHeader { magic_number: *b"CEDR", version: 1 },
            schema: vec![],
            row_groups: vec![],
        }
    }

    fn make_series(name: &str, values: Vec<DataValue>, dtype: DataType) -> Series {
        let len = values.len();
        let null_count = values.iter().filter(|v| matches!(v, DataValue::Null)).count();
        let chunks = values.into_iter().map(|data| Value { data, validity: vec![] }).collect();
        Series {
            field: Column { name: name.to_string(), dtype },
            chunks,
            len,
            null_count,
            dict: HashMap::new(),
            min: None,
            max: None,
        }
    }

    fn make_frame(name: &str, columns: Vec<Series>) -> Frame {
        Frame { name: name.to_string(), columns }
    }

    // --- Serialization unit tests (no file I/O) ---

    #[test]
    fn test_encode_bool() {
        let mut buf = vec![];
        encode_value(&DataValue::Boolean(true), &mut buf).unwrap();
        assert_eq!(buf, vec![1u8]);

        let mut buf = vec![];
        encode_value(&DataValue::Boolean(false), &mut buf).unwrap();
        assert_eq!(buf, vec![0u8]);
    }

    #[test]
    fn test_encode_int64() {
        let mut buf = vec![];
        encode_value(&DataValue::Int64(42), &mut buf).unwrap();
        assert_eq!(buf, 42i64.to_le_bytes());
    }

    #[test]
    fn test_encode_float64() {
        let mut buf = vec![];
        encode_value(&DataValue::Float64(3.14), &mut buf).unwrap();
        assert_eq!(buf, 3.14f64.to_le_bytes());
    }

    #[test]
    fn test_encode_string() {
        let mut buf = vec![];
        encode_value(&DataValue::String("hi".to_string()), &mut buf).unwrap();
        // [4 bytes len][2 bytes "hi"]
        let expected: Vec<u8> = [2u32.to_le_bytes().as_slice(), b"hi"].concat();
        assert_eq!(buf, expected);
    }

    #[test]
    fn test_null_placeholder_fixed_width() {
        let mut buf = vec![];
        write_null_placeholder(&DataType::Int64, false, &mut buf);
        assert_eq!(buf.len(), 8);
        assert!(buf.iter().all(|b| *b == 0));
    }

    #[test]
    fn test_null_placeholder_string() {
        let mut buf = vec![];
        write_null_placeholder(&DataType::String, false, &mut buf);
        // Just a zero-length prefix, no string bytes
        assert_eq!(buf, 0u32.to_le_bytes());
    }

    #[test]
    fn test_null_placeholder_dict() {
        let mut buf = vec![];
        write_null_placeholder(&DataType::String, true, &mut buf);
        // Dict placeholder is always an i64 index
        assert_eq!(buf.len(), 8);
    }

    // --- Write integration tests ---

    #[test]
    fn test_write_creates_file() {
        with_tmp_store(|| {
            let series = make_series("val", vec![DataValue::Int64(1), DataValue::Int64(2)], DataType::Int64);
            let frame  = make_frame("test_creates", vec![series]);
            make_cedrus().write(&frame).unwrap();

            let path = tmp_store().join("test_creates.cedr");
            eprintln!("EXISTS: {}", path.exists());
            // Also print CEDRUS_STORE_PATH
            assert!(path.exists(), ".cedr file was not created");
        });
    }

    #[test]
    fn test_write_header_magic() {
        with_tmp_store(|| {
            let series = make_series("x", vec![DataValue::Int64(10)], DataType::Int64);
            let frame  = make_frame("test_header", vec![series]);
            make_cedrus().write(&frame).unwrap();

            let path = tmp_store().join("test_header.cedr");
            let mut file = std::fs::File::open(path).unwrap();
            let mut magic = [0u8; 4];
            file.read_exact(&mut magic).unwrap();
            assert_eq!(&magic, b"CEDR");
        });
    }

    #[test]
    fn test_write_version() {
        with_tmp_store(|| {
            let series = make_series("x", vec![DataValue::Int64(1)], DataType::Int64);
            let frame  = make_frame("test_version", vec![series]);
            make_cedrus().write(&frame).unwrap();

            let path = tmp_store().join("test_version.cedr");
            let mut file = std::fs::File::open(path).unwrap();
            let mut buf = [0u8; 6];
            file.read_exact(&mut buf).unwrap();
            let version = u16::from_le_bytes([buf[4], buf[5]]);
            assert_eq!(version, 1);
        });
    }

    #[test]
    fn test_write_end_magic() {
        with_tmp_store(|| {
            let series = make_series("x", vec![DataValue::Int64(1)], DataType::Int64);
            let frame  = make_frame("test_end", vec![series]);
            make_cedrus().write(&frame).unwrap();

            let path = tmp_store().join("test_end.cedr");
            let bytes = std::fs::read(path).unwrap();
            assert_eq!(&bytes[bytes.len() - 4..], b"CEDR", "end magic missing");
        });
    }

    #[test]
    fn test_write_null_flag_in_data() {
        with_tmp_store(|| {
            let series = make_series(
                "col",
                vec![DataValue::Int64(1), DataValue::Null, DataValue::Int64(3)],
                DataType::Int64,
            );
            let frame = make_frame("test_nulls", vec![series]);
            make_cedrus().write(&frame).unwrap();

            let bytes = std::fs::read(tmp_store().join("test_nulls.cedr")).unwrap();
            // After header (6 bytes): row 0 → [1][8 bytes], row 1 → [0][8 zeros], row 2 → [1][8 bytes]
            assert_eq!(bytes[6], 1, "row 0 should be valid");
            assert_eq!(bytes[6 + 9], 0, "row 1 should be null");
            assert_eq!(bytes[6 + 18], 1, "row 2 should be valid");
        });
    }

    #[test]
    fn test_write_multi_column() {
        with_tmp_store(|| {
            let s1 = make_series("a", vec![DataValue::Int64(1), DataValue::Int64(2)], DataType::Int64);
            let s2 = make_series("b", vec![DataValue::Float64(1.0), DataValue::Float64(2.0)], DataType::Float64);
            let frame = make_frame("test_multi", vec![s1, s2]);
            let result = make_cedrus().write(&frame);
            assert!(result.is_ok());
        });
    }

    #[test]
    fn test_write_empty_frame() {
        with_tmp_store(|| {
            let frame = make_frame("test_empty", vec![]);
            let result = make_cedrus().write(&frame);
            assert!(result.is_ok());
        });
    }
    // --- Visual inspection test ---

    #[test]
    fn test_inspect_roundtrip() {
        with_tmp_store(|| {
            let columns = vec![
                make_series("id",      vec![DataValue::Int64(1), DataValue::Int64(2), DataValue::Int64(3), DataValue::Int64(4)], DataType::Int64),
                make_series("score",   vec![DataValue::Float64(9.5), DataValue::Float64(7.0), DataValue::Null, DataValue::Float64(8.25)], DataType::Float64),
                make_series("label",   vec![DataValue::String("alpha".into()), DataValue::String("beta".into()), DataValue::String("alpha".into()), DataValue::String("gamma".into())], DataType::String),
                make_series("active",  vec![DataValue::Boolean(true), DataValue::Boolean(false), DataValue::Boolean(true), DataValue::Null], DataType::Boolean),
            ];
            let frame = make_frame("inspect", columns);
            make_cedrus().write(&frame).unwrap();

            let got = Cedrus::read("inspect").unwrap();

            // Print header
            println!("\n{:-<60}", "");
            println!(" Frame: {:?}  ({} rows, {} columns)",
                got.name, got.columns.first().map(|c| c.len).unwrap_or(0), got.columns.len());
            println!("{:-<60}", "");

            // Column widths
            let col_w = 14usize;
            let header: String = got.columns.iter()
                .map(|c| format!("{:>col_w$}", format!("{} ({:?})", c.field.name, c.field.dtype)))
                .collect::<Vec<_>>().join(" | ");
            println!("{}", header);
            println!("{:-<60}", "");

            // Rows
            let n_rows = got.columns.first().map(|c| c.len).unwrap_or(0);
            for row in 0..n_rows {
                let line: String = got.columns.iter().map(|col| {
                    let cell = match &col.chunks[row].data {
                        DataValue::Int64(v)   => format!("{}", v),
                        DataValue::Float64(v) => format!("{:.2}", v),
                        DataValue::String(v)  => format!("{:?}", v),
                        DataValue::Boolean(v) => format!("{}", v),
                        DataValue::Null       => "NULL".to_string(),
                        other                 => format!("{:?}", other),
                    };
                    format!("{:>col_w$}", cell)
                }).collect::<Vec<_>>().join(" | ");
                println!("{}", line);
            }

            println!("{:-<60}", "");
            println!(" null counts: {}", got.columns.iter()
                .map(|c| format!("{}={}", c.field.name, c.null_count))
                .collect::<Vec<_>>().join(", "));
            println!(" dict sizes:  {}", got.columns.iter()
                .map(|c| format!("{}={}", c.field.name, c.dict.len()))
                .collect::<Vec<_>>().join(", "));
            println!("{:-<60}\n", "");
        });
    }

    // --- Round-trip tests ---

    #[test]
    fn test_roundtrip_int64() {
        with_tmp_store(|| {
            let vals = vec![DataValue::Int64(1), DataValue::Int64(-42), DataValue::Int64(999)];
            let frame = make_frame("rt_int64", vec![make_series("n", vals, DataType::Int64)]);
            make_cedrus().write(&frame).unwrap();
            let got = Cedrus::read("rt_int64").unwrap();
            assert_eq!(got.columns[0].len, 3);
            assert!(matches!(got.columns[0].chunks[1].data, DataValue::Int64(-42)));
        });
    }

    #[test]
    fn test_roundtrip_string_plain() {
        with_tmp_store(|| {
            let vals: Vec<DataValue> = (0..10).map(|i| DataValue::String(format!("unique_{i}"))).collect();
            let frame = make_frame("rt_str", vec![make_series("s", vals, DataType::String)]);
            make_cedrus().write(&frame).unwrap();
            let got = Cedrus::read("rt_str").unwrap();
            assert_eq!(got.columns[0].len, 10);
            assert!(matches!(&got.columns[0].chunks[3].data, DataValue::String(s) if s == "unique_3"));
        });
    }

    #[test]
    fn test_roundtrip_with_nulls() {
        with_tmp_store(|| {
            let vals = vec![DataValue::Int64(10), DataValue::Null, DataValue::Int64(30)];
            let frame = make_frame("rt_nulls2", vec![make_series("c", vals, DataType::Int64)]);
            make_cedrus().write(&frame).unwrap();
            let got = Cedrus::read("rt_nulls2").unwrap();
            assert_eq!(got.columns[0].null_count, 1);
            assert!(matches!(got.columns[0].chunks[1].data, DataValue::Null));
            assert!(matches!(got.columns[0].chunks[2].data, DataValue::Int64(30)));
        });
    }

    #[test]
    fn test_roundtrip_float64() {
        with_tmp_store(|| {
            let vals = vec![DataValue::Float64(3.14), DataValue::Float64(-0.001)];
            let frame = make_frame("rt_f64", vec![make_series("f", vals, DataType::Float64)]);
            make_cedrus().write(&frame).unwrap();
            let got = Cedrus::read("rt_f64").unwrap();
            assert!(matches!(got.columns[0].chunks[0].data, DataValue::Float64(v) if (v - 3.14).abs() < 1e-10));
        });
    }

    #[test]
    fn test_roundtrip_schema_preserved() {
        with_tmp_store(|| {
            let s1 = make_series("id",   vec![DataValue::Int64(1)],            DataType::Int64);
            let s2 = make_series("name", vec![DataValue::String("x".into())],  DataType::String);
            let frame = make_frame("rt_schema", vec![s1, s2]);
            make_cedrus().write(&frame).unwrap();
            let got = Cedrus::read("rt_schema").unwrap();
            assert_eq!(got.columns.len(), 2);
            assert_eq!(got.columns[0].field.name,  "id");
            assert_eq!(got.columns[0].field.dtype, DataType::Int64);
            assert_eq!(got.columns[1].field.name,  "name");
            assert_eq!(got.columns[1].field.dtype, DataType::String);
        });
    }

    #[test]
    fn test_roundtrip_bool() {
        with_tmp_store(|| {
            let vals = vec![DataValue::Boolean(true), DataValue::Boolean(false), DataValue::Boolean(true)];
            let frame = make_frame("rt_bool", vec![make_series("b", vals, DataType::Boolean)]);
            make_cedrus().write(&frame).unwrap();
            let got = Cedrus::read("rt_bool").unwrap();
            assert!(matches!(got.columns[0].chunks[0].data, DataValue::Boolean(true)));
            assert!(matches!(got.columns[0].chunks[1].data, DataValue::Boolean(false)));
        });
    }

}