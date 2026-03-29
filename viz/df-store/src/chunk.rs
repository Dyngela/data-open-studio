use crate::data_type::DataValue;

#[derive(Debug)]

pub struct Value {
    pub data: DataValue,
    pub validity: Vec<u8>,  // 1 bit par élément, packed dans des octets
}


impl Value {
    /// Check si l'élément à l'index i est null
    fn is_null(&self, i: usize) -> bool {
        let byte = self.validity[i / 8];
        let bit = i % 8;
        (byte >> bit) & 1 == 0
    }

    /// Marquer un index comme null
    fn set_null(&mut self, i: usize) {
        let byte = &mut self.validity[i / 8];
        let bit = i % 8;
        *byte &= !(1 << bit);
    }
}

