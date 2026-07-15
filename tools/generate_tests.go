package tools

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/imports"

	"github.com/sleticalboy/testloop-mcp/internal/generator"
	"github.com/sleticalboy/testloop-mcp/types"
)

type generateTestsInput struct {
	FilePath     string                  `json:"file_path" jsonschema:"源文件路径，例如 internal/calc/calc.go"`
	Framework    string                  `json:"framework,omitempty" jsonschema:"测试框架，可选值: go-test/cargo-test/jest/vitest/mocha/pytest/junit，默认自动检测，JS/TS 会读取 package.json"`
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

	testFile := targetTestFile(filePath, input.CoverageTask)
	coverageTask, err := coverageTaskForGeneration(filePath, testFile, input.CoverageTask)
	if err != nil {
		return nil, nil, err
	}
	input.CoverageTask = coverageTask

	provider, err := generator.NewTestProvider(input.Provider)
	if err != nil {
		if result, out, ok := generateTestsProviderErrorResult(filePath, nil, input, err); ok {
			return result, out, nil
		}
		return nil, nil, formatGenerateTestsError(err)
	}

	opts := generator.GenerateTestsOptions{CoverageTask: input.CoverageTask, Framework: input.Framework}
	code, err := generator.GenerateTestsWithProviderOptions(ctx, filePath, provider, opts)
	if err != nil {
		if result, out, ok := generateTestsProviderErrorResult(filePath, provider, input, err); ok {
			return result, out, nil
		}
		return nil, nil, formatGenerateTestsError(err)
	}

	if err := os.MkdirAll(filepath.Dir(testFile), 0755); err != nil && filepath.Dir(testFile) != "." {
		return nil, nil, fmt.Errorf("创建测试目录失败: %w", err)
	}
	writtenCode, err := writeGeneratedTestFile(testFile, code, filepath.Ext(filePath))
	if err != nil {
		return nil, nil, fmt.Errorf("写入测试文件失败: %w", err)
	}
	code = writtenCode

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

	return structuredToolResult(out)
}

func targetTestFile(filePath string, task *types.CoverageTestTask) string {
	testFile := generator.TestFileName(filePath)
	if task != nil && strings.TrimSpace(task.TestFile) != "" {
		testFile = task.TestFile
	}
	if task != nil && filepath.Ext(filePath) == ".java" {
		return nonCollidingJavaCoverageTestFile(testFile)
	}
	return testFile
}

func coverageTaskForGeneration(filePath, testFile string, task *types.CoverageTestTask) (*types.CoverageTestTask, error) {
	if task == nil {
		return task, nil
	}
	if strings.TrimSpace(task.TestFile) != testFile {
		adjusted := *task
		adjusted.TestFile = testFile
		task = &adjusted
	}
	if filepath.Ext(filePath) != ".go" || strings.TrimSpace(task.TestName) == "" {
		return task, nil
	}
	existing, err := existingGoPackageTestNames(testFile)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(task.TestName)
	if !existing[name] {
		return task, nil
	}
	adjusted := *task
	adjusted.TestName = uniqueGoCoverageTaskTestName(name, &adjusted, existing)
	return &adjusted, nil
}

func nonCollidingJavaCoverageTestFile(testFile string) string {
	if strings.TrimSpace(testFile) == "" || filepath.Ext(testFile) != ".java" {
		return testFile
	}
	info, err := os.Stat(testFile)
	if err != nil || info.IsDir() {
		return testFile
	}
	base := strings.TrimSuffix(testFile, filepath.Ext(testFile))
	for i := 0; i < 100; i++ {
		candidate := base + "LoopTest.java"
		if i > 0 {
			candidate = fmt.Sprintf("%sLoopTest%d.java", base, i+1)
		}
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return base + "LoopTest.java"
}

func existingGoPackageTestNames(testFile string) (map[string]bool, error) {
	existing := make(map[string]bool)
	pattern := filepath.Join(filepath.Dir(testFile), "*_test.go")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("查找已有 Go 测试文件失败: %w", err)
	}
	for _, file := range files {
		names, err := existingGoTestNames(file)
		if err != nil {
			return nil, err
		}
		for name := range names {
			existing[name] = true
		}
	}
	return existing, nil
}

func existingGoTestNames(testFile string) (map[string]bool, error) {
	existing := make(map[string]bool)
	content, err := os.ReadFile(testFile)
	if os.IsNotExist(err) {
		return existing, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取已有 Go 测试文件失败: %w", err)
	}
	if strings.TrimSpace(string(content)) == "" {
		return existing, nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, testFile, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("解析已有 Go 测试文件失败: %w", err)
	}
	return goTestFuncNames(file), nil
}

func uniqueGoCoverageTaskTestName(base string, task *types.CoverageTestTask, existing map[string]bool) string {
	for _, suffix := range goCoverageTaskTestNameSuffixes(task) {
		candidate := base + suffix
		if !existing[candidate] {
			return candidate
		}
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%sCoverage%d", base, i)
		if !existing[candidate] {
			return candidate
		}
	}
}

func goCoverageTaskTestNameSuffixes(task *types.CoverageTestTask) []string {
	var suffixes []string
	if lineRange := goLineRangeSuffix(task.LineRange); lineRange != "" {
		suffixes = append(suffixes, "Coverage"+lineRange)
	}
	if id := goCamelSuffix(task.ID); id != "" {
		suffixes = append(suffixes, "Coverage"+id)
	}
	suffixes = append(suffixes, "Coverage")
	return suffixes
}

func goLineRangeSuffix(value string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range strings.TrimSpace(value) {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore && b.Len() > 0 {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func goCamelSuffix(value string) string {
	parts := strings.FieldsFunc(strings.TrimSpace(value), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	})
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		if runes[0] >= 'a' && runes[0] <= 'z' {
			runes[0] -= 'a' - 'A'
		}
		b.WriteString(string(runes))
	}
	return b.String()
}

func generateTestsProviderErrorResult(filePath string, provider generator.TestProvider, input generateTestsInput, err error) (*mcp.CallToolResult, any, bool) {
	providerErr, ok := generator.ProviderErrorInfo(err)
	if !ok {
		return nil, nil, false
	}
	action := providerErrorAction(providerErr.Kind)
	message := formatGenerateTestsError(err).Error()
	providerName := providerErr.Provider
	if providerName == "" && provider != nil {
		providerName = provider.Name()
	}
	out := types.GenerateTestsOutput{
		Status:       "error",
		TestFile:     generator.TestFileName(filePath),
		CoverageTask: input.CoverageTask,
		Provider:     providerName,
		Error:        message,
		ProviderError: &types.ProviderErrorOutput{
			Kind:     string(providerErr.Kind),
			Action:   action,
			Provider: providerErr.Provider,
			Message:  providerErr.Error(),
		},
	}
	if input.CoverageTask != nil && strings.TrimSpace(input.CoverageTask.TestFile) != "" {
		out.TestFile = input.CoverageTask.TestFile
	}
	result, err := structuredToolResultWithError(out, true)
	if err != nil {
		return nil, nil, false
	}
	return result, out, true
}

func formatGenerateTestsError(err error) error {
	if providerErr, ok := generator.ProviderErrorInfo(err); ok {
		return fmt.Errorf("生成测试失败: provider_error kind=%s action=%s: %w", providerErr.Kind, providerErrorAction(providerErr.Kind), err)
	}
	return fmt.Errorf("生成测试失败: %w", err)
}

func providerErrorAction(kind generator.ProviderErrorKind) string {
	switch kind {
	case generator.ProviderErrorConfigMissing:
		return "configure_provider"
	case generator.ProviderErrorCommandFailed:
		return "fix_provider_command_or_retry"
	case generator.ProviderErrorEmptyOutput, generator.ProviderErrorJSON, generator.ProviderErrorMissingCode, generator.ProviderErrorOutputCleaningFailed:
		return "retry_model_or_fallback_static"
	case generator.ProviderErrorOutputValidationFailed:
		return "adjust_prompt_or_fallback_static"
	default:
		return "inspect_provider"
	}
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

func writeGeneratedTestFile(testFile string, code string, sourceExt string) (string, error) {
	if sourceExt == ".rs" {
		merged, err := mergeRustGeneratedTestFile(testFile, code)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(testFile, []byte(merged), 0644); err != nil {
			return "", err
		}
		return merged, nil
	}
	if sourceExt != ".go" {
		if err := os.WriteFile(testFile, []byte(code), 0644); err != nil {
			return "", err
		}
		return code, nil
	}
	merged, err := mergeGoGeneratedTestFile(testFile, code)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(testFile, []byte(merged), 0644); err != nil {
		return "", err
	}
	return merged, nil
}

func mergeRustGeneratedTestFile(testFile string, generated string) (string, error) {
	existing, err := os.ReadFile(testFile)
	if os.IsNotExist(err) {
		return generated, nil
	}
	if err != nil {
		return "", err
	}
	existingText := strings.TrimRight(string(existing), "\n")
	generatedText := strings.TrimSpace(generated)
	if existingText == "" {
		return generatedText + "\n", nil
	}
	if strings.Contains(existingText, generatedText) {
		return existingText + "\n", nil
	}
	return existingText + "\n\n" + generatedText + "\n", nil
}

func mergeGoGeneratedTestFile(testFile string, generated string) (string, error) {
	existing, err := os.ReadFile(testFile)
	if os.IsNotExist(err) {
		return formatGoTestFile(testFile, generated)
	}
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(existing)) == "" {
		return formatGoTestFile(testFile, generated)
	}

	existingSet := token.NewFileSet()
	existingFile, err := parser.ParseFile(existingSet, testFile, existing, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("解析已有 Go 测试文件失败: %w", err)
	}
	generatedSet := token.NewFileSet()
	generatedFile, err := parser.ParseFile(generatedSet, testFile+".generated", generated, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("解析新生成 Go 测试代码失败: %w", err)
	}
	if existingFile.Name.Name != generatedFile.Name.Name {
		return "", fmt.Errorf("Go 测试文件 package 不一致: existing=%s generated=%s", existingFile.Name.Name, generatedFile.Name.Name)
	}

	existingTests := goTestFuncNames(existingFile)
	snippets, err := goGeneratedTestFuncSnippets(generatedSet, generatedFile, generated, existingTests)
	if err != nil {
		return "", err
	}
	if len(snippets) == 0 {
		return "", fmt.Errorf("新生成 Go 测试代码中没有可追加的 Test 函数")
	}

	merged := string(existing)
	missingImports := missingGoImports(existingFile, generatedFile)
	if len(missingImports) > 0 {
		merged, err = addGoImports(existingSet, existingFile, merged, missingImports)
		if err != nil {
			return "", err
		}
	}
	merged = strings.TrimRight(merged, "\n") + "\n\n" + strings.Join(snippets, "\n\n") + "\n"

	return formatGoTestFile(testFile, merged)
}

func formatGoTestFile(testFile string, source string) (string, error) {
	formatted, err := imports.Process(testFile, []byte(source), nil)
	if err != nil {
		return source, fmt.Errorf("格式化合并后的 Go 测试文件失败: %w", err)
	}
	return string(formatted), nil
}

func goTestFuncNames(file *ast.File) map[string]bool {
	names := make(map[string]bool)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		names[fn.Name.Name] = true
	}
	return names
}

func goGeneratedTestFuncSnippets(fset *token.FileSet, file *ast.File, generated string, existing map[string]bool) ([]string, error) {
	var snippets []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if existing[fn.Name.Name] {
			return nil, fmt.Errorf("Go 测试函数已存在: %s", fn.Name.Name)
		}
		start := fset.Position(fn.Pos()).Offset
		if fn.Doc != nil {
			start = fset.Position(fn.Doc.Pos()).Offset
		}
		end := fset.Position(fn.End()).Offset
		if start < 0 || end > len(generated) || start >= end {
			return nil, fmt.Errorf("无法定位新生成 Go 测试函数: %s", fn.Name.Name)
		}
		snippets = append(snippets, strings.TrimSpace(generated[start:end]))
	}
	return snippets, nil
}

func missingGoImports(existingFile, generatedFile *ast.File) []string {
	existing := make(map[string]bool)
	for _, imp := range existingFile.Imports {
		existing[imp.Path.Value] = true
	}
	var missing []string
	for _, imp := range generatedFile.Imports {
		if !existing[imp.Path.Value] {
			existing[imp.Path.Value] = true
			missing = append(missing, imp.Path.Value)
		}
	}
	return missing
}

func addGoImports(fset *token.FileSet, file *ast.File, source string, imports []string) (string, error) {
	importDecl := firstGoImportDecl(file)
	if importDecl == nil {
		insertAt := fset.Position(file.Name.End()).Offset
		return source[:insertAt] + "\n\n" + goImportBlock(imports) + source[insertAt:], nil
	}
	if importDecl.Lparen.IsValid() {
		insertAt := fset.Position(importDecl.Rparen).Offset
		return source[:insertAt] + goImportLines(imports) + source[insertAt:], nil
	}

	start := fset.Position(importDecl.Pos()).Offset
	end := fset.Position(importDecl.End()).Offset
	if start < 0 || end > len(source) || start >= end {
		return "", fmt.Errorf("无法定位已有 Go import 声明")
	}
	allImports := make([]string, 0, len(importDecl.Specs)+len(imports))
	for _, spec := range importDecl.Specs {
		if importSpec, ok := spec.(*ast.ImportSpec); ok {
			allImports = append(allImports, importSpec.Path.Value)
		}
	}
	allImports = append(allImports, imports...)
	return source[:start] + goImportBlock(allImports) + source[end:], nil
}

func firstGoImportDecl(file *ast.File) *ast.GenDecl {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if ok && genDecl.Tok == token.IMPORT {
			return genDecl
		}
	}
	return nil
}

func goImportBlock(imports []string) string {
	return "import (\n" + goImportLines(imports) + ")\n"
}

func goImportLines(imports []string) string {
	var b strings.Builder
	for _, imp := range imports {
		b.WriteString("\t")
		b.WriteString(imp)
		b.WriteString("\n")
	}
	return b.String()
}
