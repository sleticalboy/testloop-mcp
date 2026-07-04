// demo/Calculator.java - Testloop MCP Java demo
// Run: mvn test  or  gradle test

package com.example;

/**
 * A simple calculator class for demo purposes.
 */
public class Calculator {

    private int value;

    public Calculator() {
        this.value = 0;
    }

    public int add(int a, int b) {
        return a + b;
    }

    public int divide(int a, int b) throws ArithmeticException {
        if (b == 0) {
            throw new ArithmeticException("division by zero");
        }
        return a / b;
    }

    public String greet(String name) {
        return "Hello, " + name + "!";
    }

    public boolean isPositive(int n) {
        return n > 0;
    }

    public void clear() {
        this.value = 0;
    }

    public int getValue() {
        return this.value;
    }

    // Static factory method
    public static Calculator create() {
        return new Calculator();
    }
}
