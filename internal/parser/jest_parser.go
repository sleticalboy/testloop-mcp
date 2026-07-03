package parser

import (
	"fmt"
	"strings"

	"github.com/binlee/testloop-mcp/types"
)

// ParseJestTest 解析 Jest 测试输出
//
// Jest 输出格式示例：
// PASS  ./sum.test.js
//   ✓ adds 1 + 2 to equal 3 (1 ms)
//   ✓ adds 1 + 1 to equal 2
// FAIL  ./sum.test.js
//   ✕ adds 1 + 2 to equal 3 (1 ms)
//     expect(received).toBe(expected)
func ParseJestTest(output string) types.TestResult {
	result := types.TestResult{
		Status:    "pass",
		Framework: "jest",
		Failures:  []types.TestFailure{},
		RawOutput: output,
	}

	lines := strings.Split(output, "\n")
	
	// 第一遍：解析摘要行获取准确统计
	for _, line := range lines {
		if strings.Contains(line, "Tests:") {
			parseTestSummary(line, &result)
		}
	}
	
	// 第二遍：提取失败详情
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// 检测失败详情 - Jest 用 ● 标记失败测试
		if strings.HasPrefix(trimmed, "●") {
			testName := strings.TrimSpace(trimmed[1:])
			// 提取测试名（格式：● sum › adds 1 + 2 to equal 3）
			if idx := strings.Index(testName, "›"); idx > 0 {
				testName = strings.TrimSpace(testName[idx+1:])
			}
			result.Failures = append(result.Failures, types.TestFailure{
				TestName: testName,
				Error:    "测试失败，请查看详细输出",
			})
		}
	}

	// 如果没有摘要行，从测试结果行计数
	if result.Total == 0 {
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "✓") || strings.HasPrefix(trimmed, "√") {
				result.Total++
				result.Passed++
			} else if strings.HasPrefix(trimmed, "✕") || strings.HasPrefix(trimmed, "×") {
				result.Total++
				result.Failed++
				result.Status = "fail"
			}
		}
	}

	if result.Failed > 0 {
		result.Status = "fail"
	}

	return result
}

func parseTestSummary(line string, result *types.TestResult) {
	// 格式: "Tests:       2 passed, 1 failed, 3 total"
	// 或: "Tests:       2 passed, 2 total"
	
	// 去掉 "Tests:" 前缀
	summary := strings.TrimSpace(strings.Split(line, ":")[1])
	
	// 按逗号分割
	parts := strings.Split(summary, ",")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		fields := strings.Fields(part)
		if len(fields) < 2 {
			continue
		}
		
		var count int
		fmt.Sscanf(fields[0], "%d", &count)
		
		switch fields[1] {
		case "passed":
			result.Passed = count
		case "failed":
			result.Failed = count
			result.Status = "fail"
		case "skipped":
			result.Skipped = count
		case "total":
			result.Total = count
		}
	}
	
	// 如果没有 total，计算 total
	if result.Total == 0 {
		result.Total = result.Passed + result.Failed + result.Skipped
	}
}
