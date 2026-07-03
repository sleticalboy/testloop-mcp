package parser

import (
	"testing"
	"strings"
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
}
