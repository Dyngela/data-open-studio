mod cedrus;

use std::path::PathBuf;

/// Base directory where .cedr files are stored.
/// Override with the CEDRUS_STORE_PATH environment variable.
pub fn store_path() -> PathBuf {
    if let Ok(p) = std::env::var("CEDRUS_STORE_PATH") {
        return PathBuf::from(p);
    }
    #[cfg(target_os = "windows")]
    return PathBuf::from("C:/dev/cedrus");
    #[cfg(not(target_os = "windows"))]
    return PathBuf::from("/var/lib/cedrus");
}


use crate::data_type::{DataType, DataValue};

pub enum Encoding {
    Plain,
    RleDictionary,
}

/// La structure globale qui représente ton fichier en cours de création ou de lecture
pub struct Cedrus {
    pub header: FileHeader,
    pub schema: Vec<ColumnSchema>,
    pub row_groups: Vec<RowGroup>,
}

pub struct FileHeader {
    pub magic_number: [u8; 4], // Toujours b"CEDR"
    pub version: u16,
}

pub struct FileFooter {
    pub num_rows: u64,
    pub schema: Vec<ColumnSchema>,
    pub row_groups: Vec<RowGroupMetadata>,
}

pub struct ColumnSchema {
    pub name: String,
    pub data_type: DataType,
    pub nullable: bool,
}

pub struct RowGroupMetadata {
    pub num_rows: u64,
    pub columns: Vec<ColumnMetadata>,
}


pub struct RowGroup {
    pub columns: Vec<ColumnChunk>,
    pub num_rows: u64,
}

pub struct ColumnChunk {
    pub data: Vec<u8>,
}

pub struct ColumnMetadata {
    pub encoding: Encoding,
    pub offset: u64,
    pub length: u64,
    pub null_count: u64,
    pub min_value: DataValue,
    pub max_value: DataValue,
}

