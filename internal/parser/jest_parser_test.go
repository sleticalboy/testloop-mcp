package parser

import (
	"testing"
	"strings"
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
}
