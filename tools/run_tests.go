package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/binlee/testloop-mcp/internal/parser"
)

type runTestsInput struct {
	Path      string `json:"path" jsonschema:"测试文件或目录路径，必填"`
	Framework string `json:"framework,omitempty" jsonschema:"测试框架，可选值: go-test/jest/pytest，默认自动检测"`
	Coverage  bool   `json:"coverage,omitempty" jsonschema:"是否收集覆盖率，默认 false"`
	Verbose   bool   `json:"verbose,omitempty" jsonschema:"是否输出详细日志，默认 true"`
}

// Register 把所有 Tools 注册到 server
func Register(s *mcp.Server) {
	mcp.AddTool[runTestsInput, any](s,
		&mcp.Tool{Name: "run_tests", Description: "执行测试并返回结构化结果。支持 Go/Node/Python 项目。"},
		HandleRunTests,
	)
	mcp.AddTool[parseResultsInput, any](s,
		&mcp.Tool{Name: "parse_results", Description: "解析测试执行输出，提取失败用例详情（AI 友好结构化格式）。"},
		HandleParseResults,
	)
	mcp.AddTool[generateTestsInput, any](s,
		&mcp.Tool{Name: "generate_tests", Description: "根据指定源文件生成测试代码（当前仅支持 Go）。"},
		HandleGenerateTests,
	)
	mcp.AddTool[fixSuggestionsInput, any](s,
		&mcp.Tool{Name: "fix_suggestions", Description: "根据测试失败信息和源代码，生成结构化修复建议（供 AI 消费）。"},
		HandleFixSuggestions,
	)
	mcp.AddTool[parseCoverageInput, any](s,
		&mcp.Tool{Name: "parse_coverage", Description: "解析覆盖率数据（Go coverprofile / Jest coverage JSON / pytest coverage JSON），返回结构化覆盖率报告和改进建议。"},
		HandleParseCoverage,
	)
}

func HandleRunTests(ctx context.Context, req *mcp.CallToolRequest, input runTestsInput) (*mcp.CallToolResult, any, error) {
	path := input.Path
	if path == "" {
		return nil, nil, fmt.Errorf("path 参数必填")
	}
	
	framework := input.Framework
	coverage := input.Coverage
	verbose := input.Verbose
	if !verbose {
		verbose = true
	}

	// 自动检测框架
	if framework == "" {
		framework = detectFramework(path)
	}

	var output []byte
	var err error
	
	switch framework {
	case "go-test":
		args := []string{"test"}
		if verbose {
			args = append(args, "-v")
		}
		if coverage {
			args = append(args, "-cover")
		}
		args = append(args, path)
		output, err = exec.CommandContext(ctx, "go", args...).CombinedOutput()
		
	case "jest", "vitest":
		args := []string{"jest"}
		if framework == "vitest" {
			args = []string{"vitest", "run"}
		}
		if verbose {
			args = append(args, "--verbose")
		}
		if coverage {
			args = append(args, "--coverage")
		}
		if path != "." {
			args = append(args, path)
		}
		cmd := exec.CommandContext(ctx, "npx", args...)
		cmd.Dir = getProjectRoot(path)
		output, err = cmd.CombinedOutput()
		
	case "mocha":
		args := []string{"mocha"}
		if verbose {
			args = append(args, "--reporter", "spec")
		}
		if coverage {
			args = append(args, "--coverage")
		}
		args = append(args, path)
		cmd := exec.CommandContext(ctx, "npx", args...)
		cmd.Dir = getProjectRoot(path)
		output, err = cmd.CombinedOutput()
		
	case "pytest":
		args := []string{"-m", "pytest"}
		if verbose {
			args = append(args, "-v")
		}
		if coverage {
			args = append(args, "--cov")
		}
		args = append(args, path)
		cmd := exec.CommandContext(ctx, "python3", args...)
		cmd.Dir = getProjectRoot(path)
		output, err = cmd.CombinedOutput()
		
	default:
		return nil, nil, fmt.Errorf("暂不支持框架: %s（当前支持: go-test, jest, vitest, mocha, pytest）", framework)
	}

	// 测试失败是正常情况，继续解析输出
	_ = err
	result := parser.ParseTestOutput(string(output), framework)
	resultJSON, _ := json.Marshal(result)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, nil, nil
}

func detectFramework(path string) string {
	// 检查路径是否存在
	info, err := os.Stat(path)
	if err != nil {
		return "go-test" // 默认
	}

	// 如果是文件，检查扩展名
	if !info.IsDir() {
		ext := filepath.Ext(path)
		switch ext {
		case ".go":
			return "go-test"
		case ".js", ".ts", ".jsx", ".tsx":
			return "jest"
		case ".py":
			return "pytest"
		}
	}

	// 如果是目录，检查项目配置文件
	dir := path
	if !info.IsDir() {
		dir = filepath.Dir(path)
	}

	// 检查 package.json (Node.js)
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		// 读取 package.json 检查测试框架
		if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
			content := string(data)
			if strings.Contains(content, "vitest") {
				return "vitest"
			}
			if strings.Contains(content, "mocha") {
				return "mocha"
			}
		}
		return "jest"
	}
	
	// 检查 go.mod (Go)
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return "go-test"
	}
	
	// 检查 setup.py 或 pyproject.toml (Python)
	if _, err := os.Stat(filepath.Join(dir, "setup.py")); err == nil {
		return "pytest"
	}
	if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
		return "pytest"
	}

	return "go-test" // 默认
}

func getProjectRoot(path string) string {
	// 简单实现：返回路径所在的目录
	info, err := os.Stat(path)
	if err != nil {
		return "."
	}
	
	if info.IsDir() {
		return path
	}
	return filepath.Dir(path)
}
