// Generated tests for Calculator.java
// Run with: mvn test  or  gradle test
package com.example;

import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

public class CalculatorTest {

    @Test
    public void calculator() {
        Calculator instance = new Calculator();
        Assertions.assertNotNull(instance);
    }

    @Test
    public void add() {
        Calculator instance = new Calculator();
        int result = instance.add(0, 0);
        Assertions.assertEquals(0, result);
    }

    @Test
    public void divide() {
        Calculator instance = new Calculator();
        int result = instance.divide(0, 0);
        Assertions.assertEquals(0, result);

        // Test exception path
        Assertions.assertThrows(ArithmeticException.class, () -> {
            // TODO: call with invalid args
        });
    }

    @Test
    public void greet() {
        Calculator instance = new Calculator();
        String result = instance.greet("test");
        Assertions.assertNotNull(result);
    }

    @Test
    public void ispositive() {
        Calculator instance = new Calculator();
        boolean result = instance.isPositive(0);
        Assertions.assertTrue(result);
    }

    @Test
    public void clear() {
        Calculator instance = new Calculator();
        instance.clear();
    }

    @Test
    public void create() {
        Calculator result = Calculator.create();
        Assertions.assertNotNull(result);
    }
}
