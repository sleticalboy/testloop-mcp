package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/binlee/testloop-mcp/internal/detector"
	"github.com/binlee/testloop-mcp/internal/parser"
)

type runTestsInput struct {
	Path      string `json:"path" jsonschema:"测试文件或目录路径，必填"`
	Framework string `json:"framework,omitempty" jsonschema:"测试框架，可选值: go-test/cargo-test/jest/vitest/mocha/pytest/junit，默认自动检测"`
	Coverage  bool   `json:"coverage,omitempty" jsonschema:"是否收集覆盖率，默认 false"`
	Verbose   bool   `json:"verbose,omitempty" jsonschema:"是否输出详细日志，默认 true"`
}

// Register 把所有 Tools 注册到 server
func Register(s *mcp.Server) {
	mcp.AddTool[runTestsInput, any](s,
		&mcp.Tool{Name: "run_tests", Description: "执行测试并返回结构化结果。支持 Go/Rust/Node/Python/Java 项目。"},
		HandleRunTests,
	)
	mcp.AddTool[parseResultsInput, any](s,
		&mcp.Tool{Name: "parse_results", Description: "解析测试执行输出，提取失败用例详情（AI 友好结构化格式）。"},
		HandleParseResults,
	)
	mcp.AddTool[generateTestsInput, any](s,
		&mcp.Tool{Name: "generate_tests", Description: "根据指定源文件生成测试代码，支持 Go/Rust/Java/JavaScript/TypeScript/Python，可选静态或 LLM provider。"},
		HandleGenerateTests,
	)
	mcp.AddTool[fixSuggestionsInput, any](s,
		&mcp.Tool{Name: "fix_suggestions", Description: "根据测试失败信息和源代码，生成结构化修复建议（供 AI 消费）。"},
		HandleFixSuggestions,
	)
	mcp.AddTool[parseCoverageInput, any](s,
		&mcp.Tool{Name: "parse_coverage", Description: "解析覆盖率数据（Go coverprofile / Istanbul JSON / coverage.py JSON / cargo tarpaulin LCOV / JaCoCo XML），返回结构化覆盖率报告和改进建议。"},
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
		framework = detector.DetectFramework(path)
	}

	var output []byte
	var err error

	switch framework {
	case "go-test":
		args := []string{"test", "-json"}
		if verbose {
			args = append(args, "-v")
		}
		if coverage {
			args = append(args, "-cover")
		}
		args = append(args, normalizeGoTestPath(path))
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

	case "cargo-test":
		args := []string{"test"}
		if verbose {
			args = append(args, "--", "--nocapture")
		}
		cmd := exec.CommandContext(ctx, "cargo", args...)
		cmd.Dir = findProjectRoot(path, "Cargo.toml")
		output, err = cmd.CombinedOutput()

	case "junit":
		cmd := javaTestCommand(ctx, path)
		output, err = cmd.CombinedOutput()

	default:
		return nil, nil, fmt.Errorf("暂不支持框架: %s（当前支持: go-test, cargo-test, jest, vitest, mocha, pytest, junit）", framework)
	}

	// 测试失败是正常情况，继续解析输出
	_ = err
	result := parser.ParseTestOutput(string(output), framework)
	resultJSON, _ := json.Marshal(result)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, nil, nil
}

func normalizeGoTestPath(path string) string {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || filepath.Ext(path) != ".go" {
		return path
	}
	return filepath.Dir(path)
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

func javaTestCommand(ctx context.Context, path string) *exec.Cmd {
	root := findProjectRoot(path, "pom.xml", "build.gradle", "build.gradle.kts")
	if fileExists(filepath.Join(root, "mvnw")) {
		cmd := exec.CommandContext(ctx, "./mvnw", "test")
		cmd.Dir = root
		return cmd
	}
	if fileExists(filepath.Join(root, "pom.xml")) {
		cmd := exec.CommandContext(ctx, "mvn", "test")
		cmd.Dir = root
		return cmd
	}
	if fileExists(filepath.Join(root, "gradlew")) {
		cmd := exec.CommandContext(ctx, "./gradlew", "test")
		cmd.Dir = root
		return cmd
	}
	cmd := exec.CommandContext(ctx, "gradle", "test")
	cmd.Dir = root
	return cmd
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func findProjectRoot(path string, markers ...string) string {
	root := getProjectRoot(path)
	for i := 0; i < 8; i++ {
		for _, marker := range markers {
			if fileExists(filepath.Join(root, marker)) {
				return root
			}
		}
		parent := filepath.Dir(root)
		if parent == root {
			break
		}
		root = parent
	}
	return getProjectRoot(path)
}
