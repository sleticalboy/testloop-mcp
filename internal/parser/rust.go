package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/binlee/testloop-mcp/types"
)

// ============================================================
// Rust cargo test output parser
// ============================================================

var (
	rsTestLineRE   = regexp.MustCompile(`^test\s+(\S+)\s+\.\.\.\s+(ok|FAILED)`)
	rsResultLineRE = regexp.MustCompile(`test result:\s*(ok|FAILED)\.\s*(\d+) passed;\s*(\d+) failed`)
	rsPanicRE      = regexp.MustCompile(`panicked at\s+'([^']+)'`)
)

// ParseCargoTest 解析 cargo test 输出，返回 TestResult
func ParseCargoTest(output string) types.TestResult {
	result := types.TestResult{
		Status:   "pass",
		Failures: []types.TestFailure{},
	}

	lines := strings.Split(output, "\n")
	inFailureBlock := false
	failIdx := 0 // 按顺序匹配失败详情

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 匹配 "test tests::test_add ... ok"
		if m := rsTestLineRE.FindStringSubmatch(line); m != nil {
			testName := m[1]
			status := m[2]
			if status == "ok" {
				result.Passed++
			} else {
				result.Failed++
				result.Status = "fail"
				result.Failures = append(result.Failures, types.TestFailure{
					TestName: testName,
				})
			}
			continue
		}

		// 匹配汇总行："test result: FAILED. 1 passed; 1 failed; ..."
		if m := rsResultLineRE.FindStringSubmatch(line); m != nil {
			if m[1] == "FAILED" {
				result.Status = "fail"
			} else {
				result.Status = "pass"
			}
			if n, err := strconv.Atoi(m[2]); err == nil {
				result.Passed = n
			}
			if n, err := strconv.Atoi(m[3]); err == nil {
				result.Failed = n
			}
			continue
		}

		// 失败详情块开始："---- tests::test_add stdout ----"
		if strings.HasPrefix(line, "---- ") && strings.Contains(line, " ----") {
			inFailureBlock = true
			continue
		}

		if inFailureBlock {
			// 匹配 panic 信息：panicked at 'assertion failed', src/lib.rs:10:5
			if m := rsPanicRE.FindStringSubmatch(line); m != nil {
				msg := m[1]
				if failIdx < len(result.Failures) {
					result.Failures[failIdx].Error = msg
				}
			}
			// 失败详情块结束
			if line == "" {
				inFailureBlock = false
				failIdx++
			}
		}
	}

	if result.Failed > 0 {
		result.Status = "fail"
	}

	return result
}
