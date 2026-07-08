package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/internal/generator"
	"github.com/sleticalboy/testloop-mcp/types"
)

type generateTestsInput struct {
	FilePath     string                  `json:"file_path" jsonschema:"源文件路径，例如 internal/calc/calc.go"`
	Framework    string                  `json:"framework,omitempty" jsonschema:"测试框架，可选值: go-test/cargo-test/jest/vitest/mocha/pytest/junit，默认按文件类型选择"`
	Provider     string                  `json:"provider,omitempty" jsonschema:"测试生成 provider: static、llm 或 auto，默认 static"`
	CoverageTask *types.CoverageTestTask `json:"coverage_task,omitempty" jsonschema:"parse_coverage 返回的单个 test_tasks 项，用于按覆盖率缺口生成测试"`
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

	opts := generator.GenerateTestsOptions{CoverageTask: input.CoverageTask, Framework: input.Framework}
	code, err := generator.GenerateTestsWithProviderOptions(ctx, filePath, provider, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("生成测试失败: %w", err)
	}

	testFile := generator.TestFileName(filePath)
	if input.CoverageTask != nil && strings.TrimSpace(input.CoverageTask.TestFile) != "" {
		testFile = input.CoverageTask.TestFile
	}
	if err := os.MkdirAll(filepath.Dir(testFile), 0755); err != nil && filepath.Dir(testFile) != "." {
		return nil, nil, fmt.Errorf("创建测试目录失败: %w", err)
	}
	if err := os.WriteFile(testFile, []byte(code), 0644); err != nil {
		return nil, nil, fmt.Errorf("写入测试文件失败: %w", err)
	}

	genCtx := generator.BuildGenerationContextWithOptions(filePath, opts)
	out := types.GenerateTestsOutput{
		Status:         "ok",
		TestFile:       testFile,
		GeneratedCases: countGeneratedCases(code, filepath.Ext(filePath)),
		Preview:        code,
		Context:        genCtx,
		CoverageTask:   input.CoverageTask,
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
