package parser

import (
	"strconv"
	"strings"

	"github.com/binlee/testloop-mcp/types"
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
	result := types.TestResult{
		Status:   "pass",
		Failures: []types.TestFailure{},
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
			idx1 := strings.Index(detail, ":")
			if idx1 > 0 {
				rest := detail[idx1+1:]
				idx2 := strings.Index(rest, ":")
				if idx2 > 0 {
					fileAndLine := detail[:idx1+1+idx2]
					errorMsg := strings.TrimSpace(rest[idx2+1:])

					// 解析行号
					lineNum := 0
					lastColon := strings.LastIndex(fileAndLine, ":")
					if lastColon > 0 {
						v, _ := strconv.Atoi(fileAndLine[lastColon+1:])
						lineNum = v
					}

					fileName := fileAndLine
					if lastColon > 0 {
						fileName = fileAndLine[:lastColon]
					}

					result.Failures = append(result.Failures, types.TestFailure{
						TestName: lastTest,
						File:     fileName,
						Line:     lineNum,
						Error:    errorMsg,
					})
				}
			}
			continue
		}

		// 4) 覆盖率
		if strings.Contains(line, "coverage:") {
			fields := strings.Fields(line)
			for i, f := range fields {
				if f == "coverage:" && i+1 < len(fields) {
					coverStr := strings.TrimSuffix(fields[i+1], "%")
					v, err := strconv.ParseFloat(coverStr, 64)
					if err == nil {
						result.CoveragePercent = v
					}
					break
				}
			}
		}
	}

	if result.Failed > 0 {
		result.Status = "fail"
	}
	return result
}
