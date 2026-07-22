package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/internal/coverage"
	"github.com/sleticalboy/testloop-mcp/internal/detector"
	"github.com/sleticalboy/testloop-mcp/internal/parser"
	"github.com/sleticalboy/testloop-mcp/types"
)

type runTestsInput struct {
	Path                  string `json:"path" jsonschema:"测试文件或目录路径，必填"`
	Framework             string `json:"framework,omitempty" jsonschema:"测试框架，可选值: go-test/cargo-test/jest/vitest/mocha/node-test/pytest/junit，默认自动检测"`
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
	mcp.AddTool[validateCoverageTaskInput, any](s,
		&mcp.Tool{Name: "validate_coverage_task", Description: "对单个 coverage task 执行 generate_tests -> run_tests 闭环，返回生成是否可运行、失败原因和下一步动作。"},
		HandleValidateCoverageTask,
	)
	mcp.AddTool[fixSuggestionsInput, any](s,
		&mcp.Tool{Name: "fix_suggestions", Description: "根据测试失败信息和源代码，生成结构化修复建议（供 AI 消费）。"},
		HandleFixSuggestions,
	)
	mcp.AddTool[parseCoverageInput, any](s,
		&mcp.Tool{Name: "parse_coverage", Description: "解析覆盖率数据（Go coverprofile / Istanbul JSON / Node TAP coverage report / coverage.py JSON / cargo tarpaulin LCOV / JaCoCo XML），返回结构化覆盖率报告和改进建议。"},
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
	input.Framework = framework

	var output []byte
	var err error

	switch framework {
	case "go-test":
		cmd := goTestCommand(ctx, path, verbose, coverage)
		output, err = cmd.CombinedOutput()

	case "jest", "vitest":
		cmd := jsTestCommand(ctx, framework, path, verbose, coverage)
		output, err = cmd.CombinedOutput()

	case "mocha":
		cmd := jsTestCommand(ctx, framework, path, verbose, coverage)
		output, err = cmd.CombinedOutput()

	case "node-test":
		cmd := nodeTestCommand(ctx, path, coverage)
		output, err = cmd.CombinedOutput()

	case "pytest":
		cmd := pytestCommand(ctx, path, verbose, coverage)
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
		return nil, nil, fmt.Errorf("暂不支持框架: %s（当前支持: go-test, cargo-test, jest, vitest, mocha, node-test, pytest, junit）", framework)
	}

	// 测试失败是正常情况，继续解析输出；如果解析器没有识别失败，再用非零退出码兜底。
	result := parser.ParseTestOutput(string(output), framework)
	if err != nil && result.Status == "pass" && result.Failed == 0 {
		result.Status = "fail"
		result.Failed = 1
		result.Total = result.Passed + result.Failed + result.Skipped
		result.Failures = []types.TestFailure{{
			TestName: "test runner",
			Error:    firstNonEmpty(firstNonBlankLine(string(output)), err.Error()),
		}}
	}
	if coverage {
		result.CoveragePercent = collectCoveragePercent(ctx, framework, path, result.CoveragePercent)
	}
	if input.IncludeFixSuggestions && result.Status == "fail" && len(result.Failures) > 0 {
		result.FixSuggestions = generateRunTestFixSuggestions(input, result.Failures)
	}
	annotateTestResultAction(&result)
	return structuredToolResult(result)
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
	case "node-test":
		if target != "" {
			return []string{"node --test " + filepath.ToSlash(target)}
		}
		return []string{"node --test"}
	default:
		return nil
	}
}

func jsTestCommand(ctx context.Context, framework, path string, verbose, coverage bool) *exec.Cmd {
	root := findProjectRoot(path, "package.json")
	if template := strings.TrimSpace(os.Getenv("TESTLOOP_JS_TEST_COMMAND")); template != "" {
		relPath := commandPathArg(path, root)
		command := expandJSCommandTemplate(template, framework, relPath, coverage)
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Dir = root
		return configureCommandProcessGroup(cmd)
	}
	args := jsTestArgs(framework, path, root, verbose, coverage)
	cmd := exec.CommandContext(ctx, "npx", args...)
	cmd.Dir = root
	return cmd
}

func expandJSCommandTemplate(template, framework, relPath string, coverage bool) string {
	coverageArg := ""
	if coverage {
		coverageArg = "--coverage"
	}
	replacements := map[string]string{
		"{framework}": framework,
		"{path}":      shellQuote(relPath),
		"{coverage}":  coverageArg,
	}
	command := template
	for placeholder, value := range replacements {
		command = strings.ReplaceAll(command, placeholder, value)
	}
	if !strings.Contains(template, "{path}") && relPath != "." {
		command = strings.TrimSpace(command + " " + shellQuote(relPath))
	}
	return strings.Join(strings.Fields(command), " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func jsTestArgs(framework, path, root string, verbose, coverage bool) []string {
	var args []string
	switch framework {
	case "vitest":
		args = []string{"vitest", "run"}
	case "mocha":
		args = []string{"mocha"}
	default:
		args = []string{"jest"}
	}

	if verbose {
		if framework == "mocha" {
			args = append(args, "--reporter", "spec")
		} else if framework == "vitest" {
			// Vitest 3 rejects --verbose as an unknown option; the default reporter
			// still exposes enough detail for parser/fix-suggestion extraction.
		} else {
			args = append(args, "--verbose")
		}
	}
	if coverage {
		args = append(args, "--coverage")
	}
	if path != "." {
		args = append(args, commandPathArg(path, root))
	}
	return args
}

func commandPathArg(path, dir string) string {
	absPath, pathErr := filepath.Abs(path)
	absDir, dirErr := filepath.Abs(dir)
	if pathErr == nil && dirErr == nil {
		if rel, err := filepath.Rel(absDir, absPath); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(path)
}

func nodeTestCommand(ctx context.Context, path string, coverage bool) *exec.Cmd {
	root := findProjectRoot(path, "package.json")
	args := []string{"--test"}
	if coverage {
		args = append(args, "--experimental-test-coverage")
	}
	if path != "." {
		args = append(args, commandPathArg(path, root))
	}
	cmd := exec.CommandContext(ctx, "node", args...)
	cmd.Dir = root
	return cmd
}

func pytestCommand(ctx context.Context, path string, verbose, coverage bool) *exec.Cmd {
	root := findPytestProjectRoot(path)
	if template := strings.TrimSpace(os.Getenv("TESTLOOP_PYTEST_COMMAND")); template != "" {
		relPath := commandPathArg(path, root)
		command := expandPytestCommandTemplate(template, relPath, verbose, coverage)
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Dir = root
		return configureCommandProcessGroup(cmd)
	}
	args := pytestArgs(path, root, verbose, coverage)
	cmd := exec.CommandContext(ctx, "python3", args...)
	cmd.Dir = root
	return cmd
}

func expandPytestCommandTemplate(template, relPath string, verbose, coverage bool) string {
	verboseArg := ""
	if verbose {
		verboseArg = "-v"
	}
	coverageArg := ""
	if coverage {
		coverageArg = "--cov"
	}
	replacements := map[string]string{
		"{path}":     shellQuote(relPath),
		"{verbose}":  verboseArg,
		"{coverage}": coverageArg,
	}
	command := template
	for placeholder, value := range replacements {
		command = strings.ReplaceAll(command, placeholder, value)
	}
	if !strings.Contains(template, "{path}") && relPath != "." {
		command = strings.TrimSpace(command + " " + shellQuote(relPath))
	}
	return strings.Join(strings.Fields(command), " ")
}

func findPytestProjectRoot(path string) string {
	start := getProjectRoot(path)
	root := findProjectRoot(path, "pyproject.toml", "setup.py", "pytest.ini", "tox.ini", "setup.cfg")
	if root == start {
		if parent := pytestTestsParentRoot(start); parent != "" {
			return parent
		}
	}
	return root
}

func pytestTestsParentRoot(start string) string {
	root := start
	for i := 0; i < 8; i++ {
		if filepath.Base(root) == "tests" {
			parent := filepath.Dir(root)
			if hasPythonPackageChild(parent) {
				return parent
			}
			return ""
		}
		parent := filepath.Dir(root)
		if parent == root {
			break
		}
		root = parent
	}
	return ""
}

func hasPythonPackageChild(root string) bool {
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || entry.Name() == "tests" {
			continue
		}
		if fileExists(filepath.Join(root, entry.Name(), "__init__.py")) {
			return true
		}
	}
	return false
}

func pytestArgs(path, root string, verbose, coverage bool) []string {
	args := []string{"-m", "pytest"}
	if verbose {
		args = append(args, "-v")
	}
	if coverage {
		args = append(args, "--cov")
	}
	if path != "." {
		args = append(args, commandPathArg(path, root))
	}
	return args
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

func firstNonBlankLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeGoTestPath(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return path
	}
	normalized := path
	if !info.IsDir() && filepath.Ext(path) == ".go" {
		normalized = filepath.Dir(path)
	}
	if filepath.IsAbs(normalized) || strings.HasPrefix(normalized, ".") {
		return normalized
	}
	return "." + string(filepath.Separator) + normalized
}

func goTestCommand(ctx context.Context, path string, verbose bool, coverage bool) *exec.Cmd {
	args := []string{"test", "-json"}
	if verbose {
		args = append(args, "-v")
	}
	if coverage {
		args = append(args, "-cover")
	}

	testPath := normalizeGoTestPath(path)
	root := findProjectRoot(path, "go.mod")
	if fileExists(filepath.Join(root, "go.mod")) {
		if rel, ok := goRelativeTestPath(root, testPath); ok {
			testPath = rel
		}
	}
	args = append(args, testPath)

	cmd := exec.CommandContext(ctx, "go", args...)
	if fileExists(filepath.Join(root, "go.mod")) {
		cmd.Dir = root
	}
	return cmd
}

func goRelativeTestPath(root string, testPath string) (string, bool) {
	if !filepath.IsAbs(testPath) {
		return testPath, true
	}
	rel, err := filepath.Rel(root, testPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	if rel == "." {
		return ".", true
	}
	return "./" + filepath.ToSlash(rel), true
}

func getProjectRoot(path string) string {
	// 简单实现：返回路径所在的目录
	info, err := os.Stat(path)
	if err != nil {
		if strings.TrimSpace(path) == "" {
			return "."
		}
		return filepath.Dir(path)
	}

	if info.IsDir() {
		return path
	}
	return filepath.Dir(path)
}

func javaTestCommand(ctx context.Context, path string, withCoverage bool) *exec.Cmd {
	root := findProjectRoot(path, "pom.xml", "build.gradle", "build.gradle.kts")
	testClass := javaTestClassFilter(path)
	if fileExists(filepath.Join(root, "pom.xml")) {
		commandRoot, args := javaMavenCommandRootAndArgs(root, withCoverage, testClass)
		if fileExists(filepath.Join(commandRoot, "mvnw")) {
			cmd := exec.CommandContext(ctx, "./mvnw", args...)
			cmd.Dir = commandRoot
			return cmd
		}
		cmd := exec.CommandContext(ctx, "mvn", args...)
		cmd.Dir = commandRoot
		return cmd
	}
	if fileExists(filepath.Join(root, "gradlew")) {
		args := javaGradleArgs(withCoverage, testClass)
		cmd := exec.CommandContext(ctx, "./gradlew", args...)
		cmd.Dir = root
		return cmd
	}
	args := javaGradleArgs(withCoverage, testClass)
	cmd := exec.CommandContext(ctx, "gradle", args...)
	cmd.Dir = root
	return cmd
}

func javaMavenCommandRootAndArgs(moduleRoot string, withCoverage bool, testClass string) (string, []string) {
	args := javaMavenArgs(withCoverage, testClass)
	if aggregator := findMavenAggregatorRoot(moduleRoot); aggregator != "" && aggregator != moduleRoot {
		rel, err := filepath.Rel(aggregator, moduleRoot)
		if err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			args = append([]string{"-pl", filepath.ToSlash(rel), "-am", "-DfailIfNoTests=false"}, args...)
			return aggregator, args
		}
	}
	return moduleRoot, args
}

func findMavenAggregatorRoot(moduleRoot string) string {
	dir := filepath.Dir(moduleRoot)
	moduleName := filepath.Base(moduleRoot)
	for i := 0; i < 8; i++ {
		pom := filepath.Join(dir, "pom.xml")
		if fileExists(pom) {
			data, err := os.ReadFile(pom)
			if err == nil && mavenPomDeclaresModule(string(data), moduleName) {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		moduleName = filepath.Join(filepath.Base(dir), moduleName)
		dir = parent
	}
	return ""
}

func mavenPomDeclaresModule(pom string, modulePath string) bool {
	normalized := filepath.ToSlash(modulePath)
	return strings.Contains(pom, "<module>"+normalized+"</module>") ||
		strings.Contains(pom, "<module>./"+normalized+"</module>")
}

func javaMavenArgs(withCoverage bool, testClass string) []string {
	args := []string{}
	if strings.TrimSpace(testClass) != "" {
		args = append(args, "-Dtest="+strings.TrimSpace(testClass))
	}
	if withCoverage {
		return append(args, "test", "jacoco:report")
	}
	return append(args, "test")
}

func javaGradleArgs(withCoverage bool, testClass string) []string {
	args := []string{"test"}
	if strings.TrimSpace(testClass) != "" {
		args = append(args, "--tests", strings.TrimSpace(testClass))
	}
	if withCoverage {
		return append(args, "jacocoTestReport")
	}
	return args
}

func javaTestClassFilter(path string) string {
	if strings.TrimSpace(path) == "" || strings.ToLower(filepath.Ext(path)) != ".java" {
		return ""
	}
	slash := filepath.ToSlash(path)
	if !strings.Contains(slash, "/src/test/") && !strings.HasPrefix(slash, "src/test/") {
		return ""
	}
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func findProjectRoot(path string, markers ...string) string {
	root := getProjectRoot(path)
	for i := 0; i < 16; i++ {
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
