package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	// 根据语言约定生成测试文件名
	testFile := genTestFileName(filePath)
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

// genTestFileName 根据源文件扩展名生成对应的测试文件名
// .go → _test.go, .js → .test.js, .ts → .test.ts, .py → test_*.py
func genTestFileName(srcPath string) string {
	ext := strings.ToLower(filepath.Ext(srcPath))
	dir := filepath.Dir(srcPath)
	name := strings.TrimSuffix(filepath.Base(srcPath), ext)

	switch ext {
	case ".go":
		return filepath.Join(dir, name+"_test.go")
	case ".js", ".mjs", ".cjs":
		return filepath.Join(dir, name+".test.js")
	case ".jsx":
		return filepath.Join(dir, name+".test.jsx")
	case ".ts":
		return filepath.Join(dir, name+".test.ts")
	case ".tsx":
		return filepath.Join(dir, name+".test.tsx")
	case ".py":
		return filepath.Join(dir, "test_"+name+".py")
	default:
		return filepath.Join(dir, name+".test"+ext)
	}
}
