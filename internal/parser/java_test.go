package parser

import "testing"

func TestParseJUnitTestSetsTotalsAndRawOutput(t *testing.T) {
	output := `[INFO] -------------------------------------------------------
Tests run: 3, Failures: 1, Errors: 0, Skipped: 1
Failed tests:
  testAdd(com.example.CalculatorTest)`

	result := ParseJUnitTest(output)
	if result.Status != "fail" {
		t.Fatalf("status = %s, want fail", result.Status)
	}
	if result.Total != 3 || result.Passed != 1 || result.Failed != 1 || result.Skipped != 1 {
		t.Fatalf("unexpected counts: total=%d passed=%d failed=%d skipped=%d", result.Total, result.Passed, result.Failed, result.Skipped)
	}
	if result.RawOutput != output {
		t.Fatal("raw output was not preserved")
	}
}
