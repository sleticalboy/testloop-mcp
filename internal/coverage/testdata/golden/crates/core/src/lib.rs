pub struct Validator;

impl Validator {
    pub fn check(&self, value: Option<i32>) -> Result<i32, String> {
        match value {
            Some(v) if v > 0 => Ok(v),
            _ => Err("invalid".to_string()),
        }
    }
}
