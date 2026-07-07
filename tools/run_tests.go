package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/internal/coverage"
	"github.com/sleticalboy/testloop-mcp/internal/detector"
	"github.com/sleticalboy/testloop-mcp/internal/parser"
	"github.com/sleticalboy/testloop-mcp/types"
)

type runTestsInput struct {
	Path                  string `json:"path" jsonschema:"测试文件或目录路径，必填"`
	Framework             string `json:"framework,omitempty" jsonschema:"测试框架，可选值: go-test/cargo-test/jest/vitest/mocha/pytest/junit，默认自动检测"`
	Coverage              bool   `json:"coverage,omitempty" jsonschema:"是否收集覆盖率，默认 false"`
	Verbose               bool   `json:"verbose,omitempty" jsonschema:"是否输出详细日志，默认 true"`
	IncludeFixSuggestions bool   `json:"include_fix_suggestions,omitempty" jsonschema:"测试失败时是否附带 fix_suggestions 摘要，默认 false"`
	SourceCode            string `json:"source_code,omitempty" jsonschema:"源代码文件路径，用于 include_fix_suggestions 上下文"`
	TestCode              string `json:"test_code,omitempty" jsonschema:"测试代码文件路径，用于 include_fix_suggestions 上下文"`
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
		cmd := javaTestCommand(ctx, path, coverage)
		output, err = cmd.CombinedOutput()

	default:
		return nil, nil, fmt.Errorf("暂不支持框架: %s（当前支持: go-test, cargo-test, jest, vitest, mocha, pytest, junit）", framework)
	}

	// 测试失败是正常情况，继续解析输出
	_ = err
	result := parser.ParseTestOutput(string(output), framework)
	if coverage {
		result.CoveragePercent = collectCoveragePercent(ctx, framework, path, result.CoveragePercent)
	}
	if input.IncludeFixSuggestions && result.Status == "fail" && len(result.Failures) > 0 {
		result.FixSuggestions = generateRunTestFixSuggestions(input, result.Failures)
	}
	resultJSON, _ := json.Marshal(result)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, nil, nil
}

func generateRunTestFixSuggestions(input runTestsInput, failures []types.TestFailure) []types.FixSuggestion {
	sourceFile := firstNonEmpty(input.SourceCode, sourceCandidateFromRunPath(input.Path))
	testFile := firstNonEmpty(input.TestCode, testCandidateFromRunPath(input.Path))
	sourceCode := readOptionalText(sourceFile)
	testCode := readOptionalText(testFile)
	suggestions := generateFixSuggestions(failures, sourceCode, testCode, sourceFile, testFile)
	commands := runTestRepairCommands(input.Framework, sourceFile, testFile)
	if len(commands) == 0 {
		return suggestions
	}
	for i := range suggestions {
		if suggestions[i].RepairTask != nil {
			suggestions[i].RepairTask.SuggestedCommands = commands
		}
	}
	return suggestions
}

func runTestRepairCommands(framework, sourceFile, testFile string) []string {
	target := firstNonEmpty(testFile, sourceFile)
	switch framework {
	case "jest":
		if target != "" {
			return []string{"npx jest " + filepath.ToSlash(target)}
		}
		return []string{"npx jest"}
	case "vitest":
		if target != "" {
			return []string{"npx vitest run " + filepath.ToSlash(target)}
		}
		return []string{"npx vitest run"}
	case "mocha":
		if target != "" {
			return []string{"npx mocha " + filepath.ToSlash(target)}
		}
		return []string{"npx mocha"}
	default:
		return nil
	}
}

func sourceCandidateFromRunPath(path string) string {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || looksLikeTestFile(path) {
		return ""
	}
	return path
}

func testCandidateFromRunPath(path string) string {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || !looksLikeTestFile(path) {
		return ""
	}
	return path
}

func readOptionalText(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
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

func javaTestCommand(ctx context.Context, path string, withCoverage bool) *exec.Cmd {
	root := findProjectRoot(path, "pom.xml", "build.gradle", "build.gradle.kts")
	if fileExists(filepath.Join(root, "mvnw")) {
		args := javaMavenArgs(withCoverage)
		cmd := exec.CommandContext(ctx, "./mvnw", args...)
		cmd.Dir = root
		return cmd
	}
	if fileExists(filepath.Join(root, "pom.xml")) {
		args := javaMavenArgs(withCoverage)
		cmd := exec.CommandContext(ctx, "mvn", args...)
		cmd.Dir = root
		return cmd
	}
	if fileExists(filepath.Join(root, "gradlew")) {
		args := javaGradleArgs(withCoverage)
		cmd := exec.CommandContext(ctx, "./gradlew", args...)
		cmd.Dir = root
		return cmd
	}
	args := javaGradleArgs(withCoverage)
	cmd := exec.CommandContext(ctx, "gradle", args...)
	cmd.Dir = root
	return cmd
}

func javaMavenArgs(withCoverage bool) []string {
	if withCoverage {
		return []string{"test", "jacoco:report"}
	}
	return []string{"test"}
}

func javaGradleArgs(withCoverage bool) []string {
	if withCoverage {
		return []string{"test", "jacocoTestReport"}
	}
	return []string{"test"}
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

func collectCoveragePercent(ctx context.Context, framework string, path string, current float64) float64 {
	switch framework {
	case "cargo-test":
		if percent, ok := collectRustCoveragePercent(ctx, path); ok {
			return percent
		}
	case "junit":
		if percent, ok := collectJavaCoveragePercent(path); ok {
			return percent
		}
	}
	return current
}

func collectRustCoveragePercent(ctx context.Context, path string) (float64, bool) {
	root := findProjectRoot(path, "Cargo.toml")
	outputDir := filepath.Join(root, "target", "tarpaulin")
	cmd := exec.CommandContext(ctx, "cargo", "tarpaulin", "--out", "Lcov", "--output-dir", outputDir)
	cmd.Dir = root
	if _, err := cmd.CombinedOutput(); err != nil {
		return 0, false
	}
	report, err := coverage.ParseRustTarpaulinCoverage(filepath.Join(outputDir, "lcov.info"))
	if err != nil {
		return 0, false
	}
	return report.TotalPercent, true
}

func collectJavaCoveragePercent(path string) (float64, bool) {
	root := findProjectRoot(path, "pom.xml", "build.gradle", "build.gradle.kts")
	for _, reportPath := range []string{
		filepath.Join(root, "target", "site", "jacoco", "jacoco.xml"),
		filepath.Join(root, "build", "reports", "jacoco", "test", "jacocoTestReport.xml"),
	} {
		if !fileExists(reportPath) {
			continue
		}
		report, err := coverage.ParseJaCoCoCoverage(reportPath)
		if err != nil {
			continue
		}
		return report.TotalPercent, true
	}
	return 0, false
}
