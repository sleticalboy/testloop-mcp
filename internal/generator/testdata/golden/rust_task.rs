pub struct Validator;

impl Validator {
    pub fn new() -> Self {
        Validator
    }

    pub fn check(&self, value: i32) -> bool {
        value > 0
    }

    pub fn skip(&self, value: i32) -> bool {
        value < 0
    }
}

pub fn add(a: i32, b: i32) -> i32 {
    a + b
}
