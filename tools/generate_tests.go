package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/binlee/testloop-mcp/internal/generator"
	"github.com/binlee/testloop-mcp/types"
)

type generateTestsInput struct {
	FilePath  string   `json:"file_path" jsonschema:"源文件路径，例如 internal/calc/calc.go"`
	Framework string   `json:"framework,omitempty" jsonschema:"测试框架，默认 go test"`
}

func HandleGenerateTests(ctx context.Context, req *mcp.CallToolRequest, input generateTestsInput) (*mcp.CallToolResult, any, error) {
	filePath := input.FilePath
	if filePath == "" {
		return nil, nil, fmt.Errorf("file_path 参数必填")
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); err != nil {
		return nil, nil, fmt.Errorf("文件不存在: %w", err)
	}

	// 调用 AST 分析生成测试代码
	code, err := generator.GenerateTests(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("生成测试失败: %w", err)
	}

	// 写入测试文件
	testFile := filePath[:len(filePath)-3] + "_test.go" // 替换 .go → _test.go
	if err := os.WriteFile(testFile, []byte(code), 0644); err != nil {
		return nil, nil, fmt.Errorf("写入测试文件失败: %w", err)
	}

	out := types.GenerateTestsOutput{
		Status:         "ok",
		TestFile:       testFile,
		GeneratedCases: 0, // TODO: 统计实际生成的测试用例数
		Preview:         code,
	}

	resultJSON, _ := json.Marshal(out)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, nil, nil
}
