package parser

import (
	"github.com/sleticalboy/testloop-mcp/types"
)

// ParseTestOutput 根据框架类型解析测试输出
func ParseTestOutput(output, framework string) types.TestResult {
	var result types.TestResult
	switch framework {
	case "go-test":
		result = ParseGoTest(output)
	case "jest", "vitest":
		result = ParseJestTest(output)
	case "node-test":
		result = ParseNodeTest(output)
	case "pytest":
		result = ParsePytestTest(output)
	case "mocha":
		result = ParseMochaTest(output)
	case "cargo-test":
		result = ParseCargoTest(output)
	case "junit":
		result = ParseJUnitTest(output)
	default:
		result = ParseGoTest(output)
		framework = "go-test"
	}
	result.Framework = framework
	return result
}
