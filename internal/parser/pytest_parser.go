package parser

import (
	"strings"

	"github.com/binlee/testloop-mcp/types"
)

// ParsePytestTest 解析 pytest 测试输出
//
// pytest -v 输出格式示例：
// test_demo.py::test_add PASSED
// test_demo.py::test_subtract PASSED
// test_demo.py::test_divide FAILED
// 
// =================================== FAILURES ===================================
// _______________________________ test_divide ________________________________
// def test_divide():
//     assert divide(1, 0) == 0
// AssertionError: division by zero
func ParsePytestTest(output string) types.TestResult {
	result := types.TestResult{
		Status:    "pass",
		Framework: "pytest",
		Failures:  []types.TestFailure{},
		RawOutput: output,
	}

	lines := strings.Split(output, "\n")
	var currentTest string
	
	for _, line := range lines {
		// 检测测试结果
		// 格式: test_file.py::test_name PASSED/FAILED
		if strings.Contains(line, "PASSED") {
			result.Passed++
			result.Total++
		} else if strings.Contains(line, "FAILED") {
			result.Status = "fail"
			result.Failed++
			result.Total++
			
			// 提取测试名
			if idx := strings.Index(line, "FAILED"); idx > 0 {
				testInfo := strings.TrimSpace(line[:idx])
				parts := strings.Split(testInfo, "::")
				if len(parts) > 1 {
					currentTest = parts[1]
				}
			}
		}
		
		// 检测失败详情
		if strings.Contains(line, "AssertionError") || strings.Contains(line, "Error") {
			if currentTest != "" {
				result.Failures = append(result.Failures, types.TestFailure{
					TestName: currentTest,
					Error:    strings.TrimSpace(line),
				})
			}
		}
	}

	return result
}
