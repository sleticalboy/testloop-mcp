package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/binlee/testloop-mcp/internal/parser"
	"github.com/binlee/testloop-mcp/types"
)

type parseResultsInput struct {
	Output    string `json:"output" jsonschema:"测试执行的标准输出/错误输出原文"`
	Framework string `json:"framework,omitempty" jsonschema:"测试框架，可选值: go-test/cargo-test/jest/vitest/mocha/pytest/junit，默认 go-test"`
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

	resultJSON, _ := json.Marshal(result)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, nil, nil
}

// 保留原有函数以兼容
func parseResults(input parseResultsInput) (types.TestResult, error) {
	framework := input.Framework
	if framework == "" {
		framework = "go-test"
	}

	result := parser.ParseTestOutput(input.Output, framework)
	return result, nil
}
