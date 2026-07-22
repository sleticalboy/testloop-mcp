package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/internal/coverage"
)

type parseCoverageInput struct {
	Data      string `json:"data" jsonschema:"覆盖率数据（profile 文件内容或 JSON 字符串），必填"`
	Framework string `json:"framework,omitempty" jsonschema:"测试框架: go-test/jest/vitest/mocha/node-test/pytest/cargo-test/junit，默认 go-test"`
}

func HandleParseCoverage(ctx context.Context, req *mcp.CallToolRequest, input parseCoverageInput) (*mcp.CallToolResult, any, error) {
	if input.Data == "" {
		return nil, nil, fmt.Errorf("data 参数必填")
	}
	framework := normalizeFrameworkName(input.Framework)
	if framework == "" {
		framework = "go-test"
	}

	report, err := coverage.ParseCoverage(input.Data, framework)
	if err != nil {
		return nil, nil, fmt.Errorf("解析覆盖率失败: %w", err)
	}

	return structuredToolResult(report)
}
