package parser

import (
	"fmt"
	"strings"

	"github.com/binlee/testloop-mcp/types"
)

// ParseMochaTest 解析 Mocha 测试输出
//
// Mocha 输出格式示例：
//   calc
//     ✓ add() should add numbers
//     ✓ subtract() should subtract numbers
//     1) divide() should handle division by zero
//
//   1 passing (9ms)
//   1 failing
func ParseMochaTest(output string) types.TestResult {
	result := types.TestResult{
		Status:    "pass",
		Framework: "mocha",
		Failures:  []types.TestFailure{},
		RawOutput: output,
	}

	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// 检测测试结果
		// Mocha 用 ✓ 表示通过，✗ 或数字) 表示失败
		if strings.HasPrefix(trimmed, "✓") || strings.HasPrefix(trimmed, "✔") {
			result.Total++
			result.Passed++
		} else if strings.HasPrefix(trimmed, "✗") || strings.HasPrefix(trimmed, "✘") || (len(trimmed) > 0 && trimmed[0] >= '0' && trimmed[0] <= '9' && strings.Contains(trimmed, ")")) {
			// Mocha 失败格式: "  1) test name"
			result.Total++
			result.Failed++
			result.Status = "fail"
			
			// 提取测试名
			if idx := strings.Index(trimmed, ")"); idx > 0 {
				testName := strings.TrimSpace(trimmed[idx+1:])
				result.Failures = append(result.Failures, types.TestFailure{
					TestName: testName,
					Error:    "测试失败",
				})
			}
		}
		
		// 从摘要行获取统计
		if strings.Contains(trimmed, "passing") {
			// 格式: "  3 passing (9ms)"
			fields := strings.Fields(trimmed)
			if len(fields) > 0 {
				// 提取数字
				fmt.Sscanf(fields[0], "%d", &result.Passed)
				result.Total += result.Passed
			}
		} else if strings.Contains(trimmed, "failing") {
			fields := strings.Fields(trimmed)
			if len(fields) > 0 {
				fmt.Sscanf(fields[0], "%d", &result.Failed)
				result.Total += result.Failed
				result.Status = "fail"
			}
		}
	}

	// 提取失败详情
	extractMochaFailures(output, &result)

	return result
}

func extractMochaFailures(output string, result *types.TestResult) {
	lines := strings.Split(output, "\n")
	inFailureSection := false
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// 检测失败详情开始
		if strings.Contains(trimmed, "Failures:") || (strings.HasPrefix(trimmed, "1)") || strings.HasPrefix(trimmed, "2)")) {
			inFailureSection = true
			continue
		}
		
		if inFailureSection {
			// 提取错误信息
			if strings.HasPrefix(trimmed, "Error:") || strings.HasPrefix(trimmed, "AssertionError:") {
				if len(result.Failures) > 0 {
					idx := len(result.Failures) - 1
					result.Failures[idx].Error = trimmed
				}
			}
		}
	}
}
