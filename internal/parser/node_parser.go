package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
)

var (
	nodeTestSummaryRe       = regexp.MustCompile(`^#\s+(tests|pass|fail|skipped|todo|cancelled)\s+(\d+)\s*$`)
	nodeTestCoverageAllRe   = regexp.MustCompile(`^#\s+all files\s+\|\s+([0-9]+(?:\.[0-9]+)?)\s+\|`)
	nodeTestSubtestRe       = regexp.MustCompile(`^#\s+Subtest:\s+(.+)$`)
	nodeTestNotOKRe         = regexp.MustCompile(`^not ok\s+\d+\s+-\s+(.+)$`)
	nodeTestLocationLineRe  = regexp.MustCompile(`^location:\s+'?([^']+):(\d+):(\d+)'?\s*$`)
	nodeTestLocationParenRe = regexp.MustCompile(`\(([^():]+(?:/[^():]+)*):(\d+):(\d+)\)`)
	nodeTestLocationAtRe    = regexp.MustCompile(`^\s*at\s+(?:.+?\s+)?([^():]+(?:/[^():]+)*):(\d+):(\d+)`)
)

// ParseNodeTest parses Node.js built-in test runner TAP output from `node --test`.
func ParseNodeTest(output string) types.TestResult {
	result := types.TestResult{
		Status:    "pass",
		Framework: "node-test",
		Failures:  []types.TestFailure{},
		RawOutput: output,
	}

	lines := strings.Split(output, "\n")
	parseNodeTestSummary(lines, &result)
	result.CoveragePercent = parseNodeTestCoveragePercent(lines)
	result.Failures = parseNodeTestFailures(lines)
	if result.Failed == 0 && len(result.Failures) > 0 {
		result.Failed = len(result.Failures)
	}
	if result.Total == 0 {
		result.Total = result.Passed + result.Failed + result.Skipped
	}
	if result.Failed > 0 {
		result.Status = "fail"
	}
	if result.Total == 0 && strings.TrimSpace(output) != "" && looksLikeNodeTestCommandError(output) {
		result.Status = "fail"
		result.Total = 1
		result.Failed = 1
		result.Failures = []types.TestFailure{{
			TestName: "test command",
			Error:    summarizeCommandError(output),
		}}
	}
	return result
}

func parseNodeTestSummary(lines []string, result *types.TestResult) {
	for _, line := range lines {
		match := nodeTestSummaryRe.FindStringSubmatch(strings.TrimSpace(line))
		if len(match) != 3 {
			continue
		}
		count, _ := strconv.Atoi(match[2])
		switch match[1] {
		case "tests":
			result.Total = count
		case "pass":
			result.Passed = count
		case "fail":
			result.Failed = count
		case "skipped", "todo", "cancelled":
			result.Skipped += count
		}
	}
}

func parseNodeTestCoveragePercent(lines []string) float64 {
	for _, line := range lines {
		match := nodeTestCoverageAllRe.FindStringSubmatch(strings.TrimSpace(line))
		if len(match) != 2 {
			continue
		}
		percent, _ := strconv.ParseFloat(match[1], 64)
		return percent
	}
	return 0
}

func parseNodeTestFailures(lines []string) []types.TestFailure {
	failures := []types.TestFailure{}
	lastSubtest := ""
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if match := nodeTestSubtestRe.FindStringSubmatch(trimmed); len(match) == 2 {
			lastSubtest = match[1]
			continue
		}
		match := nodeTestNotOKRe.FindStringSubmatch(trimmed)
		if len(match) != 2 {
			continue
		}
		next := i + 1
		for next < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[next]), "not ok ") && !strings.HasPrefix(strings.TrimSpace(lines[next]), "ok ") && !strings.HasPrefix(strings.TrimSpace(lines[next]), "1..") {
			next++
		}
		block := lines[i+1 : next]
		if nodeTestFailureBlockIsSuiteAggregate(block) {
			i = next - 1
			continue
		}
		failure := types.TestFailure{
			TestName: firstNonEmptyString(match[1], lastSubtest),
			Error:    "测试失败，请查看详细输出",
		}
		for _, blockLine := range block {
			consumeNodeTestFailureLine(blockLine, &failure)
		}
		if failure.Error == "测试失败，请查看详细输出" {
			failure.Error = summarizeNodeTestFailure(block)
		}
		failures = append(failures, failure)
		i = next - 1
	}
	return failures
}

func nodeTestFailureBlockIsSuiteAggregate(lines []string) bool {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "type: 'suite'" || trimmed == `type: "suite"` ||
			trimmed == "failureType: 'subtestsFailed'" || trimmed == `failureType: "subtestsFailed"` {
			return true
		}
	}
	return false
}

func consumeNodeTestFailureLine(line string, failure *types.TestFailure) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed == "---" || trimmed == "..." {
		return
	}
	if strings.HasPrefix(trimmed, "error: ") {
		value := strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "error:")), "'\"")
		if !nodeTestYAMLBlockScalar(value) {
			failure.Error = value
		}
		return
	}
	if strings.HasPrefix(trimmed, "message: ") {
		value := strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "message:")), "'\"")
		if !nodeTestYAMLBlockScalar(value) {
			failure.Error = value
		}
		return
	}
	if strings.HasPrefix(trimmed, "expected: ") {
		failure.Expected = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "expected:")), "'\"")
		return
	}
	if strings.HasPrefix(trimmed, "actual: ") {
		failure.Received = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "actual:")), "'\"")
		return
	}
	if file, lineNo, column := parseNodeTestLocation(trimmed); file != "" {
		failure.File = file
		failure.Line = lineNo
		failure.Column = column
	}
}

func parseNodeTestLocation(line string) (string, int, int) {
	for _, re := range []*regexp.Regexp{nodeTestLocationLineRe, nodeTestLocationParenRe, nodeTestLocationAtRe} {
		match := re.FindStringSubmatch(line)
		if len(match) != 4 {
			continue
		}
		lineNo, _ := strconv.Atoi(match[2])
		column, _ := strconv.Atoi(match[3])
		return match[1], lineNo, column
	}
	return "", 0, 0
}

func summarizeNodeTestFailure(lines []string) string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if nodeTestLineShouldSkipInSummary(trimmed) {
			continue
		}
		if strings.HasPrefix(trimmed, "error: ") || strings.HasPrefix(trimmed, "message: ") {
			value := strings.Trim(strings.TrimSpace(strings.SplitN(trimmed, ":", 2)[1]), "'\"")
			if nodeTestYAMLBlockScalar(value) {
				continue
			}
			return value
		}
		return strings.Trim(trimmed, "'\"")
	}
	return "测试失败，请查看详细输出"
}

func nodeTestLineShouldSkipInSummary(line string) bool {
	return line == "" ||
		line == "---" ||
		line == "..." ||
		strings.HasPrefix(line, "at ") ||
		strings.Contains(line, "|-") ||
		strings.HasPrefix(line, "duration_ms:") ||
		strings.HasPrefix(line, "failureType:") ||
		strings.HasPrefix(line, "code:") ||
		strings.HasPrefix(line, "name:") ||
		strings.HasPrefix(line, "expected:") ||
		strings.HasPrefix(line, "actual:") ||
		strings.HasPrefix(line, "operator:") ||
		strings.HasPrefix(line, "location:") ||
		strings.HasPrefix(line, "stack:")
}

func nodeTestYAMLBlockScalar(value string) bool {
	return value == "|-" || value == "|" || value == ">" || value == ">-"
}

func looksLikeNodeTestCommandError(output string) bool {
	trimmed := strings.TrimSpace(output)
	return strings.Contains(trimmed, "Could not find") ||
		strings.Contains(trimmed, "Cannot find module") ||
		strings.Contains(trimmed, "SyntaxError:") ||
		strings.Contains(trimmed, "TypeError:") ||
		strings.Contains(trimmed, "ERR_")
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
