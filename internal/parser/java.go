package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/binlee/testloop-mcp/types"
)

// ============================================================
// Java JUnit test output parser
// Supports: Maven Surefire, Gradle, JUnit 5 console output
// ============================================================

var (
	// Maven Surefire: "Tests run: 5, Failures: 1, Errors: 0, Skipped: 0"
	javaMavenRE = regexp.MustCompile(`Tests run:\s*(\d+),\s*Failures:\s*(\d+),\s*Errors:\s*(\d+),\s*Skipped:\s*(\d+)`)
	// Gradle: "5 tests completed, 1 failed"
	javaGradleRE = regexp.MustCompile(`(\d+)\s+tests?\s+completed,\s*(\d+)\s+failed`)
	// JUnit 5 plain: "tests found: 5, tests succeeded: 4, tests failed: 1"
	javaJUnit5RE = regexp.MustCompile(`tests found:\s*(\d+).*tests succeeded:\s*(\d+).*tests failed:\s*(\d+)`)
	// Individual test failure: "   [FAILED] com.example.CalcTest::testAdd"
	javaFailureRE = regexp.MustCompile(`\[FAILED\]\s+(\S+)`)
	// Maven failure detail: "Failed tests:  \n  testAdd(com.example.CalcTest)"
	javaMavenFailureRE = regexp.MustCompile(`\s+(\w+)\((\S+)\)`)
)

// ParseJUnitTest 解析 Java JUnit 测试输出（Maven/Gradle/JUnit 5）
func ParseJUnitTest(output string) types.TestResult {
	result := types.TestResult{
		Status:   "pass",
		Failures: []types.TestFailure{},
	}

	lines := strings.Split(output, "\n")

	// 第一遍：解析汇总行
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Maven Surefire 格式
		if m := javaMavenRE.FindStringSubmatch(line); m != nil {
			total, _ := strconv.Atoi(m[1])
			failures, _ := strconv.Atoi(m[2])
			errors, _ := strconv.Atoi(m[3])
			skipped, _ := strconv.Atoi(m[4])
			result.Passed = total - failures - errors - skipped
			result.Failed = failures + errors
			if result.Failed > 0 {
				result.Status = "fail"
			}
			continue
		}

		// Gradle 格式
		if m := javaGradleRE.FindStringSubmatch(line); m != nil {
			total, _ := strconv.Atoi(m[1])
			failed, _ := strconv.Atoi(m[2])
			result.Passed = total - failed
			result.Failed = failed
			if result.Failed > 0 {
				result.Status = "fail"
			}
			continue
		}

		// JUnit 5 格式
		if m := javaJUnit5RE.FindStringSubmatch(line); m != nil {
			// found = m[1], succeeded = m[2], failed = m[3]
			passed, _ := strconv.Atoi(m[2])
			failed, _ := strconv.Atoi(m[3])
			result.Passed = passed
			result.Failed = failed
			if result.Failed > 0 {
				result.Status = "fail"
			}
			continue
		}
	}

	// 第二遍：提取失败详情
	inFailureSection := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Maven: "Failed tests:" 或 "Errors:"
		if strings.HasPrefix(line, "Failed tests:") || strings.HasPrefix(line, "Errors:") {
			inFailureSection = true
			continue
		}
		if inFailureSection {
			if line == "" || strings.HasPrefix(line, "Tests run:") {
				inFailureSection = false
				continue
			}
			if m := javaMavenFailureRE.FindStringSubmatch(line); m != nil {
				methodName := m[1]
				className := m[2]
				result.Failures = append(result.Failures, types.TestFailure{
					TestName: className + "#" + methodName,
				})
			}
			continue
		}

		// JUnit 5 / Gradle: "[FAILED] com.example.CalcTest::testAdd"
		if m := javaFailureRE.FindStringSubmatch(line); m != nil {
			fullName := m[1] // com.example.CalcTest::testAdd
			result.Failures = append(result.Failures, types.TestFailure{
				TestName: fullName,
			})
		}
	}

	if result.Failed > 0 {
		result.Status = "fail"
	}

	return result
}
