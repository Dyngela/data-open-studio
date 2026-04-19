use std::collections::HashMap;
use crate::chunk::Value;
use crate::data_type::{DataType, DataValue};
use crate::frame::{Frame, FrameError};

pub struct ColumnarStore {
    frames: HashMap<String, Frame>,
}

#[derive(Debug, Clone)]
pub struct Column {
    pub name: String,
    pub dtype: DataType,
}

#[derive(Debug, Clone)]
pub struct Series {
    pub field: Column,
    pub chunks: Vec<Value>,
    pub len: usize,
    pub null_count: usize,
    pub dict: HashMap<i64, DataValue>,
    pub min: Option<DataValue>,
    pub max: Option<DataValue>,
}

impl Series {
    pub fn compress(&self) -> Result<(), FrameError> {
        Ok(())
    }
}

impl ColumnarStore {
    pub fn new() -> Self {
        ColumnarStore {
            frames: HashMap::new(),
        }
    }

    pub fn load_frame(&mut self, frame: Frame, name: String) -> Result<(), FrameError> {
        self.frames.insert(name, frame);
        Ok(())
    }

    pub fn unload_frame(&mut self, _key: String) {}
}
