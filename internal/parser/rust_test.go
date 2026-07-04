package parser

import "testing"

func TestParseCargoTestSetsTotalsAndRawOutput(t *testing.T) {
	output := `running 2 tests
test tests::test_add ... ok
test tests::test_subtract ... FAILED

failures:

---- tests::test_subtract stdout ----
thread 'tests::test_subtract' panicked at 'assertion failed', src/lib.rs:12:9

test result: FAILED. 1 passed; 1 failed; 0 ignored; 0 measured; 0 filtered out`

	result := ParseCargoTest(output)
	if result.Status != "fail" {
		t.Fatalf("status = %s, want fail", result.Status)
	}
	if result.Total != 2 || result.Passed != 1 || result.Failed != 1 {
		t.Fatalf("unexpected counts: total=%d passed=%d failed=%d", result.Total, result.Passed, result.Failed)
	}
	if result.RawOutput != output {
		t.Fatal("raw output was not preserved")
	}
}
