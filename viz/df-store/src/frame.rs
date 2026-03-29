use crate::store::Series;


pub enum FrameError {
    CompressionError(String),
    DecompressionError(String),
    ArbitraryError(String),
}

#[derive(Debug)]
pub struct Frame {
    pub name: String,
    pub columns: Vec<Series>,
}

impl Frame {

}