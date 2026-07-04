package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/binlee/testloop-mcp/types"
)

// ParseJestTest 解析 Jest 测试输出
//
// Jest 输出格式示例：
// PASS  ./sum.test.js
//
//	✓ adds 1 + 2 to equal 3 (1 ms)
//	✓ adds 1 + 1 to equal 2
//
// FAIL  ./sum.test.js
//
//	✕ adds 1 + 2 to equal 3 (1 ms)
//	  expect(received).toBe(expected)
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

	result.Failures = parseJestFailures(lines)
	if result.Failed == 0 && len(result.Failures) > 0 {
		result.Failed = len(result.Failures)
	}
	if result.Total == 0 && len(result.Failures) > 0 {
		result.Total = len(result.Failures)
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

var (
	jestLocationParenRe = regexp.MustCompile(`\(([^():]+(?:/[^():]+)*):(\d+):(\d+)\)`)
	jestLocationAtRe    = regexp.MustCompile(`^\s*at\s+([^():]+(?:/[^():]+)*):(\d+):(\d+)`)
	vitestLocationRe    = regexp.MustCompile(`^[\s❯]*([^():]+(?:/[^():]+)*):(\d+):(\d+)`)
	jestCodeFrameRe     = regexp.MustCompile(`^\s*>\s*(\d+)\s*\|`)
)

func parseJestFailures(lines []string) []types.TestFailure {
	var failures []types.TestFailure
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !isJestFailureHeader(trimmed) {
			continue
		}

		failure := types.TestFailure{
			TestName: normalizeJestTestName(trimmed),
			Error:    "测试失败，请查看详细输出",
		}

		next := i + 1
		for next < len(lines) && !isJestFailureHeader(strings.TrimSpace(lines[next])) && !isJestSummaryStart(lines[next]) {
			consumeJestFailureLine(lines[next], &failure)
			next++
		}
		if strings.TrimSpace(failure.Error) == "" || failure.Error == "测试失败，请查看详细输出" {
			failure.Error = summarizeJestFailure(lines[i+1 : next])
		}
		failures = append(failures, failure)
		i = next - 1
	}
	return failures
}

func isJestFailureHeader(trimmed string) bool {
	if strings.HasPrefix(trimmed, "●") {
		return true
	}
	return strings.HasPrefix(trimmed, "FAIL") && strings.Contains(trimmed, " > ")
}

func isJestSummaryStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "Test Suites:") ||
		strings.HasPrefix(trimmed, "Tests:") ||
		strings.HasPrefix(trimmed, "Snapshots:") ||
		strings.HasPrefix(trimmed, "Time:") ||
		strings.HasPrefix(trimmed, "Ran all test")
}

func normalizeJestTestName(header string) string {
	name := strings.TrimSpace(header)
	name = strings.TrimPrefix(name, "●")
	name = strings.TrimPrefix(name, "FAIL")
	name = strings.TrimSpace(name)
	if idx := strings.LastIndex(name, "›"); idx >= 0 {
		name = strings.TrimSpace(name[idx+len("›"):])
	} else if strings.Contains(name, " > ") {
		parts := strings.Split(name, " > ")
		if len(parts) > 1 {
			name = strings.Join(parts[1:], " > ")
		}
	}
	return name
}

func consumeJestFailureLine(line string, failure *types.TestFailure) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}

	if strings.HasPrefix(trimmed, "Expected:") {
		failure.Expected = strings.TrimSpace(strings.TrimPrefix(trimmed, "Expected:"))
		return
	}
	if strings.HasPrefix(trimmed, "Received:") {
		failure.Received = strings.TrimSpace(strings.TrimPrefix(trimmed, "Received:"))
		return
	}
	if strings.HasPrefix(trimmed, "expect(") || strings.HasPrefix(trimmed, "AssertionError:") || strings.HasPrefix(trimmed, "Error:") {
		failure.Error = trimmed
	}
	if file, lineNo, column := parseJestLocation(trimmed); file != "" {
		failure.File = file
		failure.Line = lineNo
		failure.Column = column
		return
	}
	if failure.Line == 0 {
		if matches := jestCodeFrameRe.FindStringSubmatch(line); len(matches) == 2 {
			if lineNo, err := strconv.Atoi(matches[1]); err == nil {
				failure.Line = lineNo
			}
		}
	}
}

func parseJestLocation(line string) (string, int, int) {
	for _, re := range []*regexp.Regexp{jestLocationParenRe, jestLocationAtRe, vitestLocationRe} {
		if matches := re.FindStringSubmatch(line); len(matches) == 4 {
			lineNo, _ := strconv.Atoi(matches[2])
			column, _ := strconv.Atoi(matches[3])
			return matches[1], lineNo, column
		}
	}
	return "", 0, 0
}

func summarizeJestFailure(lines []string) string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "at ") || strings.HasPrefix(trimmed, ">") || strings.Contains(trimmed, "|") {
			continue
		}
		return trimmed
	}
	return "测试失败，请查看详细输出"
}
