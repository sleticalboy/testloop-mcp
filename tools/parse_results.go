package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/internal/parser"
	"github.com/sleticalboy/testloop-mcp/types"
)

type parseResultsInput struct {
	Output    string `json:"output" jsonschema:"测试执行的标准输出/错误输出原文"`
	Framework string `json:"framework,omitempty" jsonschema:"测试框架，可选值: go-test/cargo-test/jest/vitest/mocha/node-test/pytest/junit，默认 go-test"`
}

func HandleParseResults(ctx context.Context, req *mcp.CallToolRequest, input parseResultsInput) (*mcp.CallToolResult, any, error) {
	output := input.Output
	if output == "" {
		return nil, nil, fmt.Errorf("output 参数必填")
	}
	framework := input.Framework
	if framework == "" {
		framework = "go-test"
	}

	// 调用统一解析接口
	result := parser.ParseTestOutput(output, framework)
	annotateTestResultAction(&result)

	return structuredToolResult(result)
}

// 保留原有函数以兼容
func parseResults(input parseResultsInput) (types.TestResult, error) {
	framework := input.Framework
	if framework == "" {
		framework = "go-test"
	}

	result := parser.ParseTestOutput(input.Output, framework)
	annotateTestResultAction(&result)
	return result, nil
}
