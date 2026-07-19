package parser

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
)

// ParseGoTest 解析 `go test -v` 输出，返回结构化结果
//
// go test -v 输出格式：
//
//	=== RUN   TestAdd
//	--- PASS: TestAdd (0.00s)
//	=== RUN   TestAdd_Negative
//	    calc_test.go:42: got -1, want 0
//	--- FAIL: TestAdd_Negative (0.00s)
//	FAIL
//	coverage: 50.0% of statements
//	FAIL	github.com/example/calc	0.001s
func ParseGoTest(output string) types.TestResult {
	if looksLikeGoJSON(output) {
		return parseGoTestJSON(output)
	}
	return parseGoTestText(output)
}

type goTestEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

type goFailureState struct {
	Name    string
	File    string
	Line    int
	Details []string
}

func looksLikeGoJSON(output string) bool {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		return strings.HasPrefix(line, "{") && strings.Contains(line, `"Action"`)
	}
	return false
}

func parseGoTestJSON(output string) types.TestResult {
	result := types.TestResult{
		Status:    "pass",
		Failures:  []types.TestFailure{},
		RawOutput: output,
	}

	outputs := map[string]*goFailureState{}
	failedTests := map[string]bool{}
	packageFailed := false
	packageName := ""
	var packageDetails []string
	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		var event goTestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		switch event.Action {
		case "run":
		case "build-output":
			if detail := cleanGoPackageOutput(event.Output); detail != "" {
				packageDetails = append(packageDetails, detail)
			}
		case "pass":
			if event.Test != "" {
				result.Passed++
			}
		case "fail":
			result.Status = "fail"
			if event.Test != "" {
				result.Failed++
				failedTests[event.Test] = true
				ensureGoFailure(outputs, event.Test)
			} else if result.Failed == 0 {
				packageFailed = true
				packageName = event.Package
			}
		case "skip":
			if event.Test != "" {
				result.Skipped++
			}
		case "output":
			if strings.Contains(event.Output, "coverage:") {
				result.CoveragePercent = parseGoCoverageLine(event.Output)
				continue
			}
			if event.Output == "" {
				continue
			}
			if event.Test == "" {
				if detail := cleanGoPackageOutput(event.Output); detail != "" {
					packageDetails = append(packageDetails, detail)
				}
				continue
			}
			detail := strings.TrimSpace(event.Output)
			if detail == "" || strings.HasPrefix(detail, "===") || strings.HasPrefix(detail, "---") {
				continue
			}
			state := ensureGoFailure(outputs, event.Test)
			file, lineNum, msg, ok := parseGoFailureDetail(detail)
			if ok {
				if state.File == "" {
					state.File = file
					state.Line = lineNum
				}
				if msg != "" {
					state.Details = append(state.Details, msg)
				}
			} else {
				state.Details = append(state.Details, detail)
			}
		}

	}

	for testName := range failedTests {
		failure := outputs[testName]
		errorMsg := strings.TrimSpace(strings.Join(failure.Details, "\n"))
		if errorMsg == "" {
			errorMsg = "test failed"
		}
		result.Failures = append(result.Failures, types.TestFailure{
			TestName: failure.Name,
			File:     failure.File,
			Line:     failure.Line,
			Error:    errorMsg,
		})
	}
	if packageFailed && len(result.Failures) == 0 {
		file, lineNum, errorMsg := summarizeGoPackageFailure(packageDetails)
		result.Failures = append(result.Failures, types.TestFailure{
			TestName: packageName,
			File:     file,
			Line:     lineNum,
			Error:    errorMsg,
		})
	}

	result.Total = result.Passed + result.Failed + result.Skipped
	if result.Failed > 0 || len(result.Failures) > 0 {
		result.Status = "fail"
	}
	return result
}

func cleanGoPackageOutput(output string) string {
	detail := strings.TrimSpace(output)
	if detail == "" || detail == "FAIL" || strings.HasPrefix(detail, "FAIL\t") {
		return ""
	}
	return detail
}

func summarizeGoPackageFailure(details []string) (string, int, string) {
	for _, detail := range details {
		if strings.HasPrefix(detail, "# ") {
			continue
		}
		if file, lineNum, msg, ok := parseGoFailureDetail(detail); ok {
			return file, lineNum, strings.TrimSpace(msg)
		}
	}
	for _, detail := range details {
		if strings.TrimSpace(detail) != "" {
			return "", 0, strings.TrimSpace(detail)
		}
	}
	return "", 0, "package failed"
}

func ensureGoFailure(failures map[string]*goFailureState, testName string) *goFailureState {
	if failures[testName] == nil {
		failures[testName] = &goFailureState{Name: testName}
	}
	return failures[testName]
}

func parseGoTestText(output string) types.TestResult {
	result := types.TestResult{
		Status:    "pass",
		Failures:  []types.TestFailure{},
		RawOutput: output,
	}

	lines := strings.Split(output, "\n")
	var lastTest string // 最近一次 === RUN / --- FAIL 的测试名

	for _, line := range lines {
		s := strings.TrimSpace(line)

		// 1) === RUN TestName
		if strings.HasPrefix(s, "=== RUN") {
			fields := strings.Fields(s)
			if len(fields) >= 3 {
				lastTest = fields[2]
			}
			continue
		}

		// 2) --- PASS: / --- FAIL:
		if strings.HasPrefix(s, "--- PASS:") || strings.HasPrefix(s, "--- FAIL:") {
			fields := strings.Fields(s)
			if len(fields) >= 3 {
				testName := fields[2]
				lastTest = testName
				if strings.HasPrefix(s, "--- FAIL:") {
					result.Status = "fail"
					result.Failed++
				} else {
					result.Passed++
				}
			}
			continue
		}

		// 3) 失败详情行：缩进 + "filename:line: error"
		//    只有当行首有缩进（说明是测试输出）且 lastTest 不为空时才处理
		if lastTest != "" && len(line) > 0 && (line[0] == '\t' || line[0] == ' ') {
			// 去掉缩进
			detail := strings.TrimSpace(line)
			// 格式：calc_test.go:42: got -1, want 0
			fileName, lineNum, errorMsg, ok := parseGoFailureDetail(detail)
			if ok {
				result.Failures = append(result.Failures, types.TestFailure{
					TestName: lastTest,
					File:     fileName,
					Line:     lineNum,
					Error:    errorMsg,
				})
			}
			continue
		}

		// 4) 覆盖率
		if strings.Contains(line, "coverage:") {
			result.CoveragePercent = parseGoCoverageLine(line)
		}
	}

	if result.Failed > 0 {
		result.Status = "fail"
	}
	result.Total = result.Passed + result.Failed + result.Skipped
	return result
}

func parseGoFailureDetail(detail string) (string, int, string, bool) {
	idx1 := strings.Index(detail, ":")
	if idx1 <= 0 {
		return "", 0, detail, false
	}
	rest := detail[idx1+1:]
	idx2 := strings.Index(rest, ":")
	if idx2 <= 0 {
		return "", 0, detail, false
	}

	fileAndLine := detail[:idx1+1+idx2]
	errorMsg := strings.TrimSpace(rest[idx2+1:])
	lastColon := strings.LastIndex(fileAndLine, ":")
	if lastColon <= 0 {
		return "", 0, errorMsg, false
	}

	lineNum, err := strconv.Atoi(fileAndLine[lastColon+1:])
	if err != nil {
		return "", 0, errorMsg, false
	}

	return fileAndLine[:lastColon], lineNum, errorMsg, true
}

func parseGoCoverageLine(line string) float64 {
	fields := strings.Fields(line)
	for i, f := range fields {
		if f == "coverage:" && i+1 < len(fields) {
			coverStr := strings.TrimSuffix(fields[i+1], "%")
			v, err := strconv.ParseFloat(coverStr, 64)
			if err == nil {
				return v
			}
			break
		}
	}
	return 0
}
