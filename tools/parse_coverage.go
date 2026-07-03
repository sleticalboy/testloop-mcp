package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/binlee/testloop-mcp/internal/coverage"
)

type parseCoverageInput struct {
	Data      string `json:"data" jsonschema:"覆盖率数据（profile 文件内容或 JSON 字符串），必填"`
	Framework string `json:"framework,omitempty" jsonschema:"测试框架: go-test/jest/pytest，默认 go-test"`
}

func HandleParseCoverage(ctx context.Context, req *mcp.CallToolRequest, input parseCoverageInput) (*mcp.CallToolResult, any, error) {
	if input.Data == "" {
		return nil, nil, fmt.Errorf("data 参数必填")
	}
	framework := input.Framework
	if framework == "" {
		framework = "go-test"
	}

	report, err := coverage.ParseCoverage(input.Data, framework)
	if err != nil {
		return nil, nil, fmt.Errorf("解析覆盖率失败: %w", err)
	}

	reportJSON, _ := json.Marshal(report)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(reportJSON)}},
	}, nil, nil
}
