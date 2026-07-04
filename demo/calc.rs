// demo/calc.rs - Testloop MCP Rust demo
// Run: cargo test

/// Adds two numbers.
pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

/// Divides two numbers.
/// Returns an error if b is zero.
pub fn divide(a: i32, b: i32) -> Result<i32, String> {
    if b == 0 {
        Err("division by zero".to_string())
    } else {
        Ok(a / b)
    }
}

/// Returns a greeting string.
pub fn greet(name: &str) -> String {
    format!("Hello, {}!", name)
}

/// Multiplies two numbers.
pub fn multiply(a: i32, b: i32) -> i32 {
    a * b
}

/// An async function example.
pub async fn fetch_data(url: &str) -> Result<String, String> {
    // In real code, this would make an HTTP request.
    Ok(format!("fetched: {}", url))
}

/// A struct with methods.
pub struct Calculator {
    value: i32,
}

impl Calculator {
    pub fn new() -> Self {
        Calculator { value: 0 }
    }

    pub fn add(&mut self, x: i32) -> i32 {
        self.value += x;
        self.value
    }

    pub fn divide(&self, b: i32) -> Result<i32, String> {
        if b == 0 {
            Err("division by zero".to_string())
        } else {
            Ok(self.value / b)
        }
    }

    pub fn clear(&mut self) {
        self.value = 0;
    }
}
