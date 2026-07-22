package parser

import "testing"

func TestParseNodeTestPass(t *testing.T) {
	output := `TAP version 13
# Subtest: adds values
ok 1 - adds values
  ---
  duration_ms: 1.2
  ...
1..1
# tests 1
# suites 0
# pass 1
# fail 0
# cancelled 0
# skipped 0
# todo 0
# duration_ms 12.3`

	result := ParseNodeTest(output)

	if result.Framework != "node-test" || result.Status != "pass" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Total != 1 || result.Passed != 1 || result.Failed != 0 || result.Skipped != 0 {
		t.Fatalf("unexpected counts: total=%d passed=%d failed=%d skipped=%d", result.Total, result.Passed, result.Failed, result.Skipped)
	}
}

func TestParseNodeTestCoverageSummary(t *testing.T) {
	output := `TAP version 13
# Subtest: adds values
ok 1 - adds values
1..1
# tests 1
# pass 1
# fail 0
# skipped 0
# start of coverage report
# -----------------------------------------------------------------
# file             | line % | branch % | funcs % | uncovered lines
# -----------------------------------------------------------------
# sum.js           | 100.00 |   100.00 |   50.00 |
# test/sum.test.js | 100.00 |   100.00 |  100.00 |
# -----------------------------------------------------------------
# all files        | 87.50 |   75.00 |   66.67 |
# -----------------------------------------------------------------
# end of coverage report`

	result := ParseNodeTest(output)

	if result.Status != "pass" || result.Total != 1 || result.Passed != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.CoveragePercent != 87.5 {
		t.Fatalf("coverage = %.2f, want 87.50", result.CoveragePercent)
	}
}

func TestParseNodeTestFailure(t *testing.T) {
	output := `TAP version 13
# Subtest: adds values
not ok 1 - adds values
  ---
  duration_ms: 2.1
  failureType: 'testCodeFailure'
  error: 'Expected values to be strictly equal:'
  code: 'ERR_ASSERTION'
  expected: 3
  actual: 4
  operator: 'strictEqual'
  stack: |-
    TestContext.<anonymous> (/tmp/project/sum.test.js:7:10)
    Test.runInAsyncScope (node:async_hooks:206:9)
  ...
1..1
# tests 1
# suites 0
# pass 0
# fail 1
# cancelled 0
# skipped 0
# todo 0
# duration_ms 18.4`

	result := ParseTestOutput(output, "node-test")

	if result.Framework != "node-test" || result.Status != "fail" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Total != 1 || result.Failed != 1 || result.Passed != 0 {
		t.Fatalf("unexpected counts: total=%d passed=%d failed=%d", result.Total, result.Passed, result.Failed)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("failures len = %d, want 1: %+v", len(result.Failures), result.Failures)
	}
	failure := result.Failures[0]
	if failure.TestName != "adds values" {
		t.Fatalf("test name = %q, want adds values", failure.TestName)
	}
	if failure.Expected != "3" || failure.Received != "4" {
		t.Fatalf("expected/received = %q/%q", failure.Expected, failure.Received)
	}
	if failure.File != "/tmp/project/sum.test.js" || failure.Line != 7 || failure.Column != 10 {
		t.Fatalf("location = %s:%d:%d", failure.File, failure.Line, failure.Column)
	}
	if failure.Error != "Expected values to be strictly equal:" {
		t.Fatalf("error = %q", failure.Error)
	}
}

func TestParseNodeTestNestedBDDFailureSkipsSuiteAggregate(t *testing.T) {
	output := readParserFixture(t, "node_test_failure.txt")

	result := ParseTestOutput(output, "node-test")

	if result.Framework != "node-test" || result.Status != "fail" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Total != 1 || result.Failed != 1 || result.Passed != 0 {
		t.Fatalf("unexpected counts: total=%d passed=%d failed=%d", result.Total, result.Passed, result.Failed)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("failures len = %d, want 1: %+v", len(result.Failures), result.Failures)
	}
	failure := result.Failures[0]
	if failure.TestName != "adds values" {
		t.Fatalf("test name = %q, want adds values", failure.TestName)
	}
	if failure.Error != "Expected values to be strictly equal:" {
		t.Fatalf("error = %q", failure.Error)
	}
	if failure.File != "test/sum.test.js" || failure.Line != 6 || failure.Column != 10 {
		t.Fatalf("location = %s:%d:%d", failure.File, failure.Line, failure.Column)
	}
}

func TestParseNodeTestCommandError(t *testing.T) {
	output := `node:internal/modules/cjs/loader:1210
  throw err;
  ^

Error: Cannot find module '/tmp/project/missing.test.js'
    at Module._resolveFilename (node:internal/modules/cjs/loader:1207:15)
code: 'MODULE_NOT_FOUND'`

	result := ParseTestOutput(output, "node-test")

	if result.Framework != "node-test" || result.Status != "fail" || result.Total != 1 || result.Failed != 1 {
		t.Fatalf("unexpected command error result: %+v", result)
	}
	if len(result.Failures) != 1 || result.Failures[0].TestName != "test command" {
		t.Fatalf("unexpected failures: %+v", result.Failures)
	}
}
