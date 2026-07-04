package parser

import (
	"strings"
	"testing"
)

func TestParseGoTest_Pass(t *testing.T) {
	output := `
=== RUN   TestAdd
--- PASS: TestAdd (0.00s)
=== RUN   TestSub
--- PASS: TestSub (0.00s)
PASS
coverage: 100.0% of statements
ok  	github.com/example/calc	0.001s
`
	result := ParseGoTest(output)
	if result.Status != "pass" {
		t.Errorf("期望 status=pass，实际 %s", result.Status)
	}
	if result.Passed != 2 {
		t.Errorf("期望 passed=2，实际 %d", result.Passed)
	}
	if result.Failed != 0 {
		t.Errorf("期望 failed=0，实际 %d", result.Failed)
	}
}

func TestParseGoTest_Fail(t *testing.T) {
	output := `
=== RUN   TestAdd
--- PASS: TestAdd (0.00s)
=== RUN   TestAdd_Negative
    calc_test.go:42: got -1, want 0
--- FAIL: TestAdd_Negative (0.00s)
FAIL
coverage: 50.0% of statements
FAIL	github.com/example/calc	0.001s
`
	result := ParseGoTest(output)
	if result.Status != "fail" {
		t.Errorf("期望 status=fail，实际 %s", result.Status)
	}
	if result.Passed != 1 {
		t.Errorf("期望 passed=1，实际 %d", result.Passed)
	}
	if result.Failed != 1 {
		t.Errorf("期望 failed=1，实际 %d", result.Failed)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("期望 1 个失败，实际 %d", len(result.Failures))
	}
	f := result.Failures[0]
	if f.TestName != "TestAdd_Negative" {
		t.Errorf("期望 TestName=TestAdd_Negative，实际 %s", f.TestName)
	}
	if !strings.Contains(f.Error, "got -1") {
		t.Errorf("期望 Error 包含 'got -1'，实际 %s", f.Error)
	}
}

func TestParseGoTest_JSONPassFailSkip(t *testing.T) {
	output := `
{"Time":"2026-07-04T10:00:00Z","Action":"run","Package":"example.com/calc","Test":"TestAdd"}
{"Time":"2026-07-04T10:00:00Z","Action":"output","Package":"example.com/calc","Test":"TestAdd","Output":"=== RUN   TestAdd\n"}
{"Time":"2026-07-04T10:00:00Z","Action":"output","Package":"example.com/calc","Test":"TestAdd","Output":"    calc_test.go:10: got 4, want 5\n"}
{"Time":"2026-07-04T10:00:00Z","Action":"fail","Package":"example.com/calc","Test":"TestAdd","Elapsed":0}
{"Time":"2026-07-04T10:00:00Z","Action":"run","Package":"example.com/calc","Test":"TestSub"}
{"Time":"2026-07-04T10:00:00Z","Action":"pass","Package":"example.com/calc","Test":"TestSub","Elapsed":0}
{"Time":"2026-07-04T10:00:00Z","Action":"run","Package":"example.com/calc","Test":"TestSlow"}
{"Time":"2026-07-04T10:00:00Z","Action":"skip","Package":"example.com/calc","Test":"TestSlow","Elapsed":0}
{"Time":"2026-07-04T10:00:00Z","Action":"output","Package":"example.com/calc","Output":"coverage: 66.7% of statements\n"}
{"Time":"2026-07-04T10:00:00Z","Action":"fail","Package":"example.com/calc","Elapsed":0}
`

	result := ParseGoTest(output)
	if result.Status != "fail" {
		t.Fatalf("期望 status=fail，实际 %s", result.Status)
	}
	if result.Passed != 1 || result.Failed != 1 || result.Skipped != 1 || result.Total != 3 {
		t.Fatalf("统计错误: passed=%d failed=%d skipped=%d total=%d", result.Passed, result.Failed, result.Skipped, result.Total)
	}
	if result.CoveragePercent != 66.7 {
		t.Fatalf("期望 coverage=66.7，实际 %.1f", result.CoveragePercent)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("期望 1 个失败，实际 %d: %+v", len(result.Failures), result.Failures)
	}
	f := result.Failures[0]
	if f.TestName != "TestAdd" {
		t.Errorf("期望 TestName=TestAdd，实际 %s", f.TestName)
	}
	if f.File != "calc_test.go" || f.Line != 10 {
		t.Errorf("期望 file/line=calc_test.go:10，实际 %s:%d", f.File, f.Line)
	}
	if !strings.Contains(f.Error, "got 4") {
		t.Errorf("期望 Error 包含 got 4，实际 %s", f.Error)
	}
}
