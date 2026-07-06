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
	if len(result.Failures) != 1 || result.Failures[0].TestName != "com.example.CalculatorTest#testAdd" {
		t.Fatalf("unexpected failures: %+v", result.Failures)
	}
}

func TestParseJUnitTestMavenErrorsSection(t *testing.T) {
	output := `[ERROR] Errors:
  testDivideByZero(com.example.CalculatorTest)

Tests run: 2, Failures: 0, Errors: 1, Skipped: 0`

	result := ParseJUnitTest(output)

	if result.Status != "fail" || result.Total != 2 || result.Passed != 1 || result.Failed != 1 || result.Skipped != 0 {
		t.Fatalf("unexpected counts: status=%s total=%d passed=%d failed=%d skipped=%d", result.Status, result.Total, result.Passed, result.Failed, result.Skipped)
	}
	if len(result.Failures) != 1 || result.Failures[0].TestName != "com.example.CalculatorTest#testDivideByZero" {
		t.Fatalf("unexpected failures: %+v", result.Failures)
	}
}

func TestParseJUnitTestGradleFailure(t *testing.T) {
	output := `CalculatorTest > adds numbers FAILED
    org.opentest4j.AssertionFailedError at CalculatorTest.java:12

3 tests completed, 1 failed
   [FAILED] com.example.CalculatorTest::addsNumbers`

	result := ParseJUnitTest(output)

	if result.Status != "fail" || result.Total != 3 || result.Passed != 2 || result.Failed != 1 {
		t.Fatalf("unexpected counts: status=%s total=%d passed=%d failed=%d", result.Status, result.Total, result.Passed, result.Failed)
	}
	if len(result.Failures) != 1 || result.Failures[0].TestName != "com.example.CalculatorTest::addsNumbers" {
		t.Fatalf("unexpected failures: %+v", result.Failures)
	}
}

func TestParseJUnitTestJUnit5Summary(t *testing.T) {
	output := `JUnit Jupiter
tests found: 4, tests started: 4, tests succeeded: 3, tests failed: 1
[FAILED] com.example.CalculatorTest::subtractsNumbers`

	result := ParseJUnitTest(output)

	if result.Status != "fail" || result.Total != 4 || result.Passed != 3 || result.Failed != 1 {
		t.Fatalf("unexpected counts: status=%s total=%d passed=%d failed=%d", result.Status, result.Total, result.Passed, result.Failed)
	}
	if len(result.Failures) != 1 || result.Failures[0].TestName != "com.example.CalculatorTest::subtractsNumbers" {
		t.Fatalf("unexpected failures: %+v", result.Failures)
	}
}

func TestParseJUnitTestPassWithoutExplicitSummary(t *testing.T) {
	result := ParseJUnitTest("BUILD SUCCESSFUL")

	if result.Status != "pass" || result.Total != 0 || result.Passed != 0 || result.Failed != 0 {
		t.Fatalf("unexpected result: status=%s total=%d passed=%d failed=%d", result.Status, result.Total, result.Passed, result.Failed)
	}
}
