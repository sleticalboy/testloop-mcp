package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
)

// ParseMochaTest 解析 Mocha 测试输出
//
// Mocha 输出格式示例：
//
//	calc
//	  ✓ add() should add numbers
//	  ✓ subtract() should subtract numbers
//	  1) divide() should handle division by zero
//
//	1 passing (9ms)
//	1 failing
func ParseMochaTest(output string) types.TestResult {
	result := types.TestResult{
		Status:    "pass",
		Framework: "mocha",
		Failures:  []types.TestFailure{},
		RawOutput: output,
	}

	lines := strings.Split(output, "\n")
	parseMochaSummary(lines, &result)
	result.Failures = parseMochaFailures(lines)

	if result.Total == 0 {
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if isMochaPassLine(trimmed) {
				result.Passed++
				result.Total++
			} else if isMochaSpecFailureLine(trimmed) {
				result.Failed++
				result.Total++
			}
		}
	}
	if result.Failed == 0 && len(result.Failures) > 0 {
		result.Failed = len(result.Failures)
		result.Total = result.Passed + result.Failed + result.Skipped
	}
	if result.Failed > 0 {
		result.Status = "fail"
	}

	return result
}

var (
	mochaPassingRe     = regexp.MustCompile(`^(\d+)\s+passing\b`)
	mochaFailingRe     = regexp.MustCompile(`^(\d+)\s+failing\b`)
	mochaPendingRe     = regexp.MustCompile(`^(\d+)\s+pending\b`)
	mochaFailureHeadRe = regexp.MustCompile(`^(\d+)\)\s+(.+)$`)
	mochaLocationRe    = regexp.MustCompile(`\(([^():]+(?:/[^():]+)*):(\d+):(\d+)\)`)
	mochaAtLocationRe  = regexp.MustCompile(`^\s*at\s+([^():]+(?:/[^():]+)*):(\d+):(\d+)`)
)

func parseMochaSummary(lines []string, result *types.TestResult) {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if matches := mochaPassingRe.FindStringSubmatch(trimmed); len(matches) == 2 {
			result.Passed, _ = strconv.Atoi(matches[1])
			continue
		}
		if matches := mochaFailingRe.FindStringSubmatch(trimmed); len(matches) == 2 {
			result.Failed, _ = strconv.Atoi(matches[1])
			continue
		}
		if matches := mochaPendingRe.FindStringSubmatch(trimmed); len(matches) == 2 {
			result.Skipped, _ = strconv.Atoi(matches[1])
		}
	}
	result.Total = result.Passed + result.Failed + result.Skipped
}

func isMochaPassLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "✓") || strings.HasPrefix(trimmed, "✔")
}

func isMochaSpecFailureLine(trimmed string) bool {
	if strings.HasPrefix(trimmed, "✗") || strings.HasPrefix(trimmed, "✘") {
		return true
	}
	return isMochaNumberedLine(trimmed)
}

func parseMochaFailures(lines []string) []types.TestFailure {
	var failures []types.TestFailure
	inFailureDetails := false
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if mochaFailingRe.MatchString(trimmed) {
			inFailureDetails = true
			continue
		}
		if !inFailureDetails || !mochaFailureHeadRe.MatchString(trimmed) {
			continue
		}
		failure := types.TestFailure{
			TestName: normalizeMochaFailureName(trimmed),
			Error:    "测试失败",
		}
		next := i + 1
		for next < len(lines) && !isMochaNumberedLine(strings.TrimSpace(lines[next])) && !isMochaSummaryLine(lines[next]) {
			consumeMochaFailureLine(lines[next], &failure)
			next++
		}
		failures = append(failures, failure)
		i = next - 1
	}
	if len(failures) == 0 {
		failures = parseMochaSpecFailures(lines)
	}
	return failures
}

func parseMochaSpecFailures(lines []string) []types.TestFailure {
	var failures []types.TestFailure
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !isMochaSpecFailureLine(trimmed) {
			continue
		}
		failures = append(failures, types.TestFailure{
			TestName: normalizeMochaFailureName(trimmed),
			Error:    "测试失败",
		})
	}
	return failures
}

func isMochaNumberedLine(trimmed string) bool {
	matches := mochaFailureHeadRe.FindStringSubmatch(trimmed)
	return len(matches) == 3
}

func normalizeMochaFailureName(header string) string {
	matches := mochaFailureHeadRe.FindStringSubmatch(header)
	if len(matches) != 3 {
		return strings.TrimSpace(header)
	}
	return strings.TrimSpace(matches[2])
}

func isMochaSummaryLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return mochaPassingRe.MatchString(trimmed) || mochaFailingRe.MatchString(trimmed) || mochaPendingRe.MatchString(trimmed)
}

func consumeMochaFailureLine(line string, failure *types.TestFailure) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	if strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "AssertionError") && !strings.HasPrefix(trimmed, "Error:") {
		name := strings.TrimSuffix(trimmed, ":")
		if failure.TestName == "" {
			failure.TestName = name
		} else if !strings.Contains(failure.TestName, name) {
			failure.TestName += " " + name
		}
		return
	}
	if strings.HasPrefix(trimmed, "AssertionError") || strings.HasPrefix(trimmed, "Error:") || strings.Contains(trimmed, " expected ") {
		failure.Error = trimmed
	}
	if file, lineNo, column := parseMochaLocation(trimmed); file != "" {
		failure.File = file
		failure.Line = lineNo
		failure.Column = column
	}
}

func parseMochaLocation(line string) (string, int, int) {
	for _, re := range []*regexp.Regexp{mochaLocationRe, mochaAtLocationRe} {
		if matches := re.FindStringSubmatch(line); len(matches) == 4 {
			lineNo, _ := strconv.Atoi(matches[2])
			column, _ := strconv.Atoi(matches[3])
			return matches[1], lineNo, column
		}
	}
	return "", 0, 0
}
