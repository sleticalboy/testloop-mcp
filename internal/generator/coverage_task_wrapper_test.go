package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCoverageTaskGeneratorsFallBackWhenTaskNil(t *testing.T) {
	goPath := writeCoverageWrapperSource(t, "calc.go", `package sample

func Add(a, b int) int {
	return a + b
}
`)
	goCode, err := GenerateGoTestsForCoverageTask(goPath, nil)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask(nil) error = %v", err)
	}
	if !strings.Contains(goCode, "func TestAdd") {
		t.Fatalf("expected regular Go output, got:\n%s", goCode)
	}

	pyPath := writeCoverageWrapperSource(t, "calc.py", `def add(a, b):
    return a + b
`)
	pyCode, err := GeneratePytestTestsForCoverageTask(pyPath, nil)
	if err != nil {
		t.Fatalf("GeneratePytestTestsForCoverageTask(nil) error = %v", err)
	}
	if !strings.Contains(pyCode, "def test_add():") {
		t.Fatalf("expected regular pytest output, got:\n%s", pyCode)
	}

	jsPath := writeCoverageWrapperSource(t, "calc.js", `function add(a, b) {
  return a + b;
}

module.exports = { add };
`)
	jsCode, err := GenerateJestTestsForCoverageTask(jsPath, nil)
	if err != nil {
		t.Fatalf("GenerateJestTestsForCoverageTask(nil) error = %v", err)
	}
	if !strings.Contains(jsCode, "describe('add'") {
		t.Fatalf("expected regular Jest output, got:\n%s", jsCode)
	}

	_, javaCode, err := GenerateJavaTestsForCoverageTask([]byte(`public class Calculator {
    public int add(int a, int b) {
        return a + b;
    }
}
`), "Calculator.java", nil)
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(nil) error = %v", err)
	}
	if !strings.Contains(javaCode, "class CalculatorTest") || !strings.Contains(javaCode, "void add()") {
		t.Fatalf("expected regular Java output, got:\n%s", javaCode)
	}

	_, rustCode, err := GenerateRustTestsForCoverageTask([]byte(`pub fn add(a: i32, b: i32) -> i32 {
    a + b
}
`), "lib.rs", nil)
	if err != nil {
		t.Fatalf("GenerateRustTestsForCoverageTask(nil) error = %v", err)
	}
	if !strings.Contains(rustCode, "fn test_add()") {
		t.Fatalf("expected regular Rust output, got:\n%s", rustCode)
	}
}

func writeCoverageWrapperSource(t *testing.T, name, source string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
