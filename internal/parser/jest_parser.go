package parser

import (
	"encoding/json"
	"strings"

	"github.com/binlee/testloop-mcp/types"
)

// ParseJestTest 解析 Jest 测试输出
//
// Jest 输出格式示例：
// PASS  src/sum.test.js
// FAIL  src/sum.test.js
//   ● sum › adds 1 + 2 to equal 3
//     expect(received).toBe(expected)
//     Expected: 3
//     Received: 4
func ParseJestTest(output string) types.TestResult {
	result := types.TestResult{
		Status:    "pass",
		Framework: "jest",
		Failures:  []types.TestFailure{},
		RawOutput: output,
	}

	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		// 检测测试结果
		if strings.Contains(line, "PASS") {
			result.Passed++
		} else if strings.Contains(line, "FAIL") {
			result.Status = "fail"
			result.Failed++
		}
		
		// 检测失败详情
		if strings.Contains(line, "●") {
			// 提取测试名和错误信息
			testName := strings.TrimSpace(strings.TrimPrefix(line, "●"))
			result.Failures = append(result.Failures, types.TestFailure{
				TestName: testName,
				Error:    "测试失败，请查看详细输出",
			})
		}
	}

	// 尝试解析 Jest JSON 输出（如果可用）
	if strings.Contains(output, "{") {
		parseJestJSONOutput(output, &result)
	}

	return result
}

func parseJestJSONOutput(output string, result *types.TestResult) {
	// Jest 可以用 --json 输出 JSON 格式
	// 这里简单处理，实际应该解析完整的 JSON
	
	// 查找 JSON 开始位置
	start := strings.Index(output, "{")
	if start < 0 {
		return
	}
	
	jsonStr := output[start:]
	var jestResult map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jestResult); err != nil {
		return
	}
	
	// 解析测试结果
	if numTotalTests, ok := jestResult["numTotalTests"].(float64); ok {
		result.Total = int(numTotalTests)
	}
	if numPassedTests, ok := jestResult["numPassedTests"].(float64); ok {
		result.Passed = int(numPassedTests)
	}
	if numFailedTests, ok := jestResult["numFailedTests"].(float64); ok {
		result.Failed = int(numFailedTests)
		result.Status = "fail"
	}
}
