package parser

import (
	"strings"
	"testing"

	"github.com/sleticalboy/testloop-mcp/types"
)

func TestParsePytestTest(t *testing.T) {
	output := `test_calc.py::test_add PASSED                                            [ 33%]
test_calc.py::test_add_negative PASSED                                   [ 66%]
test_calc.py::test_subtract PASSED                                       [100%]

============================== 3 passed in 0.00s ===============================`

	result := ParsePytestTest(output)

	if result.Framework != "pytest" {
		t.Errorf("Expected framework 'pytest', got '%s'", result.Framework)
	}

	if result.Passed != 3 {
		t.Errorf("Expected 3 passed, got %d", result.Passed)
	}

	if result.Failed != 0 {
		t.Errorf("Expected 0 failed, got %d", result.Failed)
	}

	if result.Total != 3 {
		t.Errorf("Expected 3 total, got %d", result.Total)
	}

	if result.Status != "pass" {
		t.Errorf("Expected status 'pass', got '%s'", result.Status)
	}
}

func TestParsePytestTestErrorSkipAndDuplicateResultLines(t *testing.T) {
	output := `test_calc.py::test_add PASSED                                            [ 25%]
test_calc.py::test_add PASSED                                            [ 25%]
test_calc.py::test_todo SKIPPED                                          [ 50%]
test_calc.py::test_setup ERROR                                           [100%]

========================= 1 passed, 1 skipped, 1 error in 0.01s =========================`

	result := ParsePytestTest(output)

	if result.Status != "fail" {
		t.Fatalf("Expected fail status, got %s", result.Status)
	}
	if result.Total != 3 || result.Passed != 1 || result.Skipped != 1 || result.Failed != 1 {
		t.Fatalf("Unexpected counts: total=%d passed=%d skipped=%d failed=%d", result.Total, result.Passed, result.Skipped, result.Failed)
	}
}

func TestParsePytestTestFailure(t *testing.T) {
	output := `test_calc.py::test_add FAILED                                            [ 33%]
test_calc.py::test_add_negative PASSED                                   [ 66%]
test_calc.py::test_subtract PASSED                                       [100%]

=================================== FAILURES ===================================
_______________________________ test_add ________________________________

    def test_add():
>       assert add(1, 2) == 4
E       assert 3 == 4

test_calc.py:4: AssertionError
============================== 1 failed, 2 passed in 0.01s ===============================`

	result := ParsePytestTest(output)

	if result.Status != "fail" {
		t.Errorf("Expected status 'fail', got '%s'", result.Status)
	}

	if result.Failed != 1 {
		t.Errorf("Expected 1 failed, got %d", result.Failed)
	}

	if result.Passed != 2 {
		t.Errorf("Expected 2 passed, got %d", result.Passed)
	}

	if len(result.Failures) == 0 {
		t.Error("Expected at least 1 failure, got 0")
	} else {
		// 检查是否捕获了失败信息
		found := false
		for _, f := range result.Failures {
			if strings.Contains(f.TestName, "test_add") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected failure for test_add")
		}
	}

	failure := result.Failures[0]
	if failure.File != "test_calc.py" || failure.Line != 4 {
		t.Errorf("Expected location test_calc.py:4, got %s:%d", failure.File, failure.Line)
	}
	if failure.Error != "assert 3 == 4" {
		t.Errorf("Expected assertion detail, got %q", failure.Error)
	}
	if failure.Expected != "assert add(1, 2) == 4" {
		t.Errorf("Expected failing source expression, got %q", failure.Expected)
	}
}

func TestParsePytestTestExceptionFailure(t *testing.T) {
	output := `test_calc.py::test_divide FAILED                                           [100%]

=================================== FAILURES ===================================
________________________________ test_divide _________________________________

    def test_divide():
>       divide(1, 0)

test_calc.py:8: 
_ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _

a = 1, b = 0

    def divide(a, b):
        if b == 0:
>           raise ValueError("division by zero")
E           ValueError: division by zero

calc.py:7: ValueError
============================== 1 failed in 0.01s ===============================`

	result := ParsePytestTest(output)

	if result.Status != "fail" || result.Failed != 1 {
		t.Fatalf("Expected one failure, got status=%s failed=%d", result.Status, result.Failed)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("Expected one failure detail, got %d", len(result.Failures))
	}
	failure := result.Failures[0]
	if failure.TestName != "test_divide" {
		t.Errorf("Expected test_divide, got %q", failure.TestName)
	}
	if failure.File != "calc.py" || failure.Line != 7 {
		t.Errorf("Expected business source location calc.py:7, got %s:%d", failure.File, failure.Line)
	}
	if failure.Error != "ValueError: division by zero" {
		t.Errorf("Expected exception detail, got %q", failure.Error)
	}
	if !strings.Contains(failure.Expected, `divide(1, 0)`) {
		t.Errorf("Expected traceback expression, got %q", failure.Expected)
	}
}

func TestParsePytestTestFailureUsesFallbackSummary(t *testing.T) {
	output := `=================================== FAILURES ===================================
_______________________________ test_plain ________________________________

custom pytest failure detail without traceback prefixes

============================== 1 failed in 0.01s ===============================`

	result := ParsePytestTest(output)

	if result.Status != "fail" || result.Failed != 1 || result.Total != 1 {
		t.Fatalf("Expected one failed test, got status=%s failed=%d total=%d", result.Status, result.Failed, result.Total)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("Expected one failure detail, got %d", len(result.Failures))
	}
	failure := result.Failures[0]
	if failure.TestName != "test_plain" {
		t.Errorf("Expected test_plain, got %q", failure.TestName)
	}
	if failure.Error != "custom pytest failure detail without traceback prefixes" {
		t.Errorf("Expected fallback failure summary, got %q", failure.Error)
	}
}

func TestParsePytestTestFailureBlockWithoutSummary(t *testing.T) {
	output := `=================================== FAILURES ===================================
_______________________________ test_plain ________________________________

custom pytest failure detail without summary`

	result := ParsePytestTest(output)

	if result.Status != "fail" || result.Failed != 1 || result.Total != 1 {
		t.Fatalf("Expected one failed test, got status=%s failed=%d total=%d", result.Status, result.Failed, result.Total)
	}
	if len(result.Failures) != 1 || result.Failures[0].Error != "custom pytest failure detail without summary" {
		t.Fatalf("Unexpected failures: %+v", result.Failures)
	}
}

func TestConsumePytestFailureLineKeepsFirstTracebackExpression(t *testing.T) {
	failure := types.TestFailure{}

	consumePytestFailureLine(">       first_call()", &failure)
	consumePytestFailureLine(">       second_call()", &failure)

	if failure.Expected != "first_call()" {
		t.Fatalf("Expected first traceback expression, got %q", failure.Expected)
	}
}

func TestSummarizePytestFailureUsesDefaultWhenOnlyFramesRemain(t *testing.T) {
	got := summarizePytestFailure([]string{
		"",
		">       assert add(1, 2) == 4",
		"|       + where 3 = add(1, 2)",
		"_______________________________ captured stdout _______________________________",
	})

	if got != "测试失败，请查看详细输出" {
		t.Fatalf("Expected default summary, got %q", got)
	}
}

func TestParsePytestSummaryOnly(t *testing.T) {
	output := `============================== test session starts ==============================
collected 3 items

========================= 1 failed, 1 passed, 1 skipped in 0.02s =========================`

	result := ParsePytestTest(output)

	if result.Status != "fail" {
		t.Errorf("Expected fail status, got %s", result.Status)
	}
	if result.Total != 3 || result.Failed != 1 || result.Passed != 1 || result.Skipped != 1 {
		t.Errorf("Unexpected counts: total=%d failed=%d passed=%d skipped=%d", result.Total, result.Failed, result.Passed, result.Skipped)
	}
}
