package parser

import (
	"github.com/binlee/testloop-mcp/types"
)

// ParseTestOutput 根据框架类型解析测试输出
func ParseTestOutput(output, framework string) types.TestResult {
	switch framework {
	case "go-test":
		return ParseGoTest(output)
	case "jest":
		return ParseJestTest(output)
	case "pytest":
		return ParsePytestTest(output)
	default:
		// 默认按 go test 解析
		return ParseGoTest(output)
	}
}
