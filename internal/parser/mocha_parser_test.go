package parser

import "testing"

func TestParseMochaTestPass(t *testing.T) {
	output := `  calc
    ✓ add() should add numbers
    ✓ subtract() should subtract numbers

  2 passing (9ms)`

	result := ParseMochaTest(output)

	if result.Framework != "mocha" {
		t.Errorf("Expected framework mocha, got %s", result.Framework)
	}
	if result.Status != "pass" {
		t.Errorf("Expected pass status, got %s", result.Status)
	}
	if result.Total != 2 || result.Passed != 2 || result.Failed != 0 {
		t.Errorf("Unexpected counts: total=%d passed=%d failed=%d", result.Total, result.Passed, result.Failed)
	}
}

func TestParseMochaTestFailureDetails(t *testing.T) {
	output := `  calc
    ✓ add() should add numbers
    1) divide() should handle division by zero

  1 passing (12ms)
  1 failing

  1) calc
       divide() should handle division by zero:
     AssertionError: expected 4 to equal 3
      at Context.<anonymous> (test/calc.test.js:12:18)`

	result := ParseMochaTest(output)

	if result.Status != "fail" {
		t.Errorf("Expected fail status, got %s", result.Status)
	}
	if result.Total != 2 || result.Passed != 1 || result.Failed != 1 {
		t.Errorf("Unexpected counts: total=%d passed=%d failed=%d", result.Total, result.Passed, result.Failed)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("Expected one failure, got %d", len(result.Failures))
	}
	failure := result.Failures[0]
	if failure.TestName != "calc divide() should handle division by zero" {
		t.Errorf("Expected full failure name, got %q", failure.TestName)
	}
	if failure.Error != "AssertionError: expected 4 to equal 3" {
		t.Errorf("Expected assertion detail, got %q", failure.Error)
	}
	if failure.File != "test/calc.test.js" || failure.Line != 12 || failure.Column != 18 {
		t.Errorf("Expected location test/calc.test.js:12:18, got %s:%d:%d", failure.File, failure.Line, failure.Column)
	}
}

func TestParseMochaTestFallbackCountsWithoutSummary(t *testing.T) {
	output := `  calc
    ✓ add() should add numbers
    1) divide() should handle division by zero`

	result := ParseMochaTest(output)

	if result.Status != "fail" {
		t.Errorf("Expected fail status, got %s", result.Status)
	}
	if result.Total != 2 || result.Passed != 1 || result.Failed != 1 {
		t.Errorf("Unexpected fallback counts: total=%d passed=%d failed=%d", result.Total, result.Passed, result.Failed)
	}
}
