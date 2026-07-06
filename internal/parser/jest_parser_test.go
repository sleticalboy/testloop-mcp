package parser

import (
	"strings"
	"testing"
)

func TestParseJestTest(t *testing.T) {
	output := `PASS  ./sum.test.js
  ✓ adds 1 + 2 to equal 3 (1 ms)
  ✓ adds 1 + 1 to equal 2

Test Suites: 1 passed, 1 total
Tests:       2 passed, 2 total
Snapshots:   0 total
Time:        0.145 s
Ran all test suites.`

	result := ParseJestTest(output)

	if result.Framework != "jest" {
		t.Errorf("Expected framework 'jest', got '%s'", result.Framework)
	}

	if result.Passed != 2 {
		t.Errorf("Expected 2 passed, got %d", result.Passed)
	}

	if result.Failed != 0 {
		t.Errorf("Expected 0 failed, got %d", result.Failed)
	}

	if result.Status != "pass" {
		t.Errorf("Expected status 'pass', got '%s'", result.Status)
	}
}

func TestParseJestTestFailure(t *testing.T) {
	output := `FAIL  ./sum.test.js
  ✕ adds 1 + 2 to equal 3 (1 ms)

  ● sum › adds 1 + 2 to equal 3
    expect(received).toBe(expected)
    Expected: 3
    Received: 4
      at Object.<anonymous> (sum.test.js:5:15)

Test Suites: 1 failed, 0 passed, 1 total
Tests:       1 failed, 0 passed, 1 total`

	result := ParseJestTest(output)

	if result.Status != "fail" {
		t.Errorf("Expected status 'fail', got '%s'", result.Status)
	}

	if result.Failed != 1 {
		t.Errorf("Expected 1 failed, got %d", result.Failed)
	}

	if len(result.Failures) == 0 {
		t.Error("Expected at least 1 failure, got 0")
	} else if !strings.Contains(result.Failures[0].TestName, "adds 1 + 2") {
		t.Errorf("Expected test name to contain 'adds 1 + 2', got '%s'", result.Failures[0].TestName)
	}

	failure := result.Failures[0]
	if failure.Expected != "3" {
		t.Errorf("Expected expected value '3', got %q", failure.Expected)
	}
	if failure.Received != "4" {
		t.Errorf("Expected received value '4', got %q", failure.Received)
	}
	if failure.File != "sum.test.js" || failure.Line != 5 || failure.Column != 15 {
		t.Errorf("Expected location sum.test.js:5:15, got %s:%d:%d", failure.File, failure.Line, failure.Column)
	}
	if !strings.Contains(failure.Error, "expect(received)") {
		t.Errorf("Expected assertion message, got %q", failure.Error)
	}
}

func TestParseJestTestFailureWithCodeFrameLocation(t *testing.T) {
	output := `FAIL  ./sum.test.js
  ● sum › adds values

    expect(received).toEqual(expected)

    Expected: {"total": 3}
    Received: {"total": 4}

      3 | test('adds values', () => {
      4 |   const result = add(1, 2)
    > 5 |   expect(result).toEqual({ total: 3 })
        |                  ^
      6 | })

      at Object.toEqual (sum.test.js:5:18)

Test Suites: 1 failed, 1 total
Tests:       1 failed, 1 total`

	result := ParseJestTest(output)

	if result.Status != "fail" || result.Failed != 1 {
		t.Fatalf("Expected one failed test, got status=%s failed=%d", result.Status, result.Failed)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("Expected one failure, got %d", len(result.Failures))
	}
	failure := result.Failures[0]
	if failure.TestName != "adds values" {
		t.Errorf("Expected normalized test name, got %q", failure.TestName)
	}
	if failure.Expected != `{"total": 3}` || failure.Received != `{"total": 4}` {
		t.Errorf("Expected structured assertion values, got expected=%q received=%q", failure.Expected, failure.Received)
	}
	if failure.File != "sum.test.js" || failure.Line != 5 || failure.Column != 18 {
		t.Errorf("Expected stack location sum.test.js:5:18, got %s:%d:%d", failure.File, failure.Line, failure.Column)
	}
}

func TestParseJestTestFailureUsesFallbackSummary(t *testing.T) {
	output := `FAIL  ./sum.test.js
  ● sum › reports custom matcher details

    custom matcher failed without structured assertion output
      at Object.<anonymous> (sum.test.js:4:3)

Test Suites: 1 failed, 1 total
Tests:       1 failed, 1 total`

	result := ParseJestTest(output)

	if result.Status != "fail" || result.Failed != 1 {
		t.Fatalf("Expected one failed test, got status=%s failed=%d", result.Status, result.Failed)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("Expected one failure detail, got %d", len(result.Failures))
	}
	failure := result.Failures[0]
	if failure.TestName != "reports custom matcher details" {
		t.Errorf("Expected normalized test name, got %q", failure.TestName)
	}
	if failure.Error != "custom matcher failed without structured assertion output" {
		t.Errorf("Expected fallback failure summary, got %q", failure.Error)
	}
	if failure.File != "sum.test.js" || failure.Line != 4 || failure.Column != 3 {
		t.Errorf("Expected stack location sum.test.js:4:3, got %s:%d:%d", failure.File, failure.Line, failure.Column)
	}
}

func TestParseVitestFailure(t *testing.T) {
	output := ` FAIL  src/sum.test.ts > sum > adds values
AssertionError: expected 4 to be 3 // Object.is equality

Expected: 3
Received: 4

 ❯ src/sum.test.ts:8:18
      6| test('adds values', () => {
      7|   const result = add(1, 2)
      8|   expect(result).toBe(3)
       |                  ^

 Test Files  1 failed (1)
      Tests  1 failed (1)`

	result := ParseTestOutput(output, "vitest")

	if result.Framework != "vitest" || result.Status != "fail" {
		t.Fatalf("Expected failed vitest result, got framework=%s status=%s", result.Framework, result.Status)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("Expected one failure, got %d", len(result.Failures))
	}
	failure := result.Failures[0]
	if failure.TestName != "sum > adds values" {
		t.Errorf("Expected vitest test name, got %q", failure.TestName)
	}
	if failure.Expected != "3" || failure.Received != "4" {
		t.Errorf("Expected values 3/4, got expected=%q received=%q", failure.Expected, failure.Received)
	}
	if failure.File != "src/sum.test.ts" || failure.Line != 8 || failure.Column != 18 {
		t.Errorf("Expected location src/sum.test.ts:8:18, got %s:%d:%d", failure.File, failure.Line, failure.Column)
	}
}
