package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
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
//
//	assert divide(1, 0) == 0
//
// AssertionError: division by zero
func ParsePytestTest(output string) types.TestResult {
	result := types.TestResult{
		Status:    "pass",
		Framework: "pytest",
		Failures:  []types.TestFailure{},
		RawOutput: output,
	}

	lines := strings.Split(output, "\n")
	seenResults := make(map[string]bool)

	for _, line := range lines {
		testID, status := parsePytestResultLine(line)
		if testID == "" || seenResults[testID] {
			continue
		}
		seenResults[testID] = true
		switch status {
		case "PASSED":
			result.Passed++
			result.Total++
		case "FAILED", "ERROR":
			result.Status = "fail"
			result.Failed++
			result.Total++
		case "SKIPPED":
			result.Skipped++
			result.Total++
		}
	}

	result.Failures = parsePytestFailures(lines)
	if result.Total == 0 {
		parsePytestSummary(output, &result)
	}
	if result.Failed == 0 && len(result.Failures) > 0 {
		result.Failed = len(result.Failures)
		result.Status = "fail"
	}
	if result.Total == 0 && len(result.Failures) > 0 {
		result.Total = len(result.Failures)
	}
	if result.Failed > 0 {
		result.Status = "fail"
	}

	return result
}

var (
	pytestResultLineRe   = regexp.MustCompile(`^(\S+::\S+(?:\[[^\]]+\])?)\s+(PASSED|FAILED|ERROR|SKIPPED)\b`)
	pytestSummaryCountRe = regexp.MustCompile(`(\d+)\s+(passed|failed|errors?|skipped)`)
	pytestFailureHeadRe  = regexp.MustCompile(`^_+\s+(.+?)\s+_+$`)
	pytestLocationRe     = regexp.MustCompile(`^([^:\s]+\.py):(\d+):\s*(.+)$`)
	pytestTracebackRe    = regexp.MustCompile(`^>\s+(.+)$`)
	pytestErrorLineRe    = regexp.MustCompile(`^E\s+(.+)$`)
)

func parsePytestResultLine(line string) (string, string) {
	matches := pytestResultLineRe.FindStringSubmatch(strings.TrimSpace(line))
	if len(matches) != 3 {
		return "", ""
	}
	return matches[1], matches[2]
}

func parsePytestSummary(output string, result *types.TestResult) {
	lines := strings.Split(output, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.Trim(lines[i], "= ")
		matches := pytestSummaryCountRe.FindAllStringSubmatch(line, -1)
		if len(matches) == 0 {
			continue
		}
		for _, match := range matches {
			count, _ := strconv.Atoi(match[1])
			switch match[2] {
			case "passed":
				result.Passed = count
			case "failed", "error", "errors":
				result.Failed += count
			case "skipped":
				result.Skipped = count
			}
		}
		result.Total = result.Passed + result.Failed + result.Skipped
		return
	}
}

func parsePytestFailures(lines []string) []types.TestFailure {
	var failures []types.TestFailure
	for i := 0; i < len(lines); i++ {
		name := parsePytestFailureHeader(lines[i])
		if name == "" {
			continue
		}

		failure := types.TestFailure{
			TestName: name,
			Error:    "测试失败，请查看详细输出",
		}

		next := i + 1
		for next < len(lines) && parsePytestFailureHeader(lines[next]) == "" && !isPytestSummaryLine(lines[next]) {
			consumePytestFailureLine(lines[next], &failure)
			next++
		}
		if failure.Error == "测试失败，请查看详细输出" {
			failure.Error = summarizePytestFailure(lines[i+1 : next])
		}
		failures = append(failures, failure)
		i = next - 1
	}
	return failures
}

func parsePytestFailureHeader(line string) string {
	trimmed := strings.TrimSpace(line)
	matches := pytestFailureHeadRe.FindStringSubmatch(trimmed)
	if len(matches) != 2 {
		return ""
	}
	name := strings.TrimSpace(matches[1])
	if name == "" || strings.TrimSpace(strings.ReplaceAll(name, "_", "")) == "" ||
		strings.EqualFold(name, "FAILURES") || strings.EqualFold(name, "ERRORS") {
		return ""
	}
	return name
}

func isPytestSummaryLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "====") && strings.Contains(trimmed, " in ")
}

func consumePytestFailureLine(line string, failure *types.TestFailure) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	if matches := pytestLocationRe.FindStringSubmatch(trimmed); len(matches) == 4 {
		failure.File = matches[1]
		failure.Line, _ = strconv.Atoi(matches[2])
		if failure.Error == "测试失败，请查看详细输出" {
			failure.Error = strings.TrimSpace(matches[3])
		}
		return
	}
	if matches := pytestErrorLineRe.FindStringSubmatch(trimmed); len(matches) == 2 {
		failure.Error = strings.TrimSpace(matches[1])
		return
	}
	if matches := pytestTracebackRe.FindStringSubmatch(trimmed); len(matches) == 2 {
		if failure.Expected == "" {
			failure.Expected = strings.TrimSpace(matches[1])
		}
		return
	}
	if strings.HasPrefix(trimmed, "AssertionError:") || strings.HasPrefix(trimmed, "ValueError:") || strings.HasPrefix(trimmed, "TypeError:") {
		failure.Error = trimmed
	}
}

func summarizePytestFailure(lines []string) string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "|") || strings.HasPrefix(trimmed, "_") {
			continue
		}
		return trimmed
	}
	return "测试失败，请查看详细输出"
}
