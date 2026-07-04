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
	FilePath  string `json:"file_path" jsonschema:"源文件路径，例如 internal/calc/calc.go"`
	Framework string `json:"framework,omitempty" jsonschema:"测试框架，默认 go test"`
	Provider  string `json:"provider,omitempty" jsonschema:"测试生成 provider: static、llm 或 auto，默认 static"`
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

	provider, err := generator.NewTestProvider(input.Provider)
	if err != nil {
		return nil, nil, err
	}

	code, err := generator.GenerateTestsWithProvider(ctx, filePath, provider)
	if err != nil {
		return nil, nil, fmt.Errorf("生成测试失败: %w", err)
	}

	testFile := generator.TestFileName(filePath)
	if err := os.WriteFile(testFile, []byte(code), 0644); err != nil {
		return nil, nil, fmt.Errorf("写入测试文件失败: %w", err)
	}

	out := types.GenerateTestsOutput{
		Status:         "ok",
		TestFile:       testFile,
		GeneratedCases: countGeneratedCases(code, filepath.Ext(filePath)),
		Preview:        code,
		Context:        generator.BuildGenerationContext(filePath),
		Provider:       provider.Name(),
	}

	resultJSON, _ := json.Marshal(out)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, nil, nil
}

func countGeneratedCases(code, ext string) int {
	ext = strings.ToLower(ext)
	switch ext {
	case ".go":
		return countLinePrefixes(code, "func Test")
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		return strings.Count(code, "it('") + strings.Count(code, "it(\"")
	case ".py":
		return countLinePrefixes(code, "def test_", "    def test_")
	case ".rs":
		return strings.Count(code, "#[test]")
	case ".java":
		return strings.Count(code, "@Test")
	default:
		return 0
	}
}

func countLinePrefixes(code string, prefixes ...string) int {
	count := 0
	for _, line := range strings.Split(code, "\n") {
		for _, prefix := range prefixes {
			if strings.HasPrefix(line, prefix) {
				count++
				break
			}
		}
	}
	return count
}
