package generator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sleticalboy/testloop-mcp/types"
)

func TestNewTestProviderAutoFallsBackToStaticWhenCommandMissing(t *testing.T) {
	t.Setenv(EnvLLMProviderCommand, "")

	provider, err := NewTestProvider("auto")
	if err != nil {
		t.Fatalf("NewTestProvider(auto) error = %v", err)
	}
	if provider.Name() != "static" {
		t.Fatalf("provider.Name() = %q, want static", provider.Name())
	}
}

func TestNewTestProviderLLMRequiresCommand(t *testing.T) {
	t.Setenv(EnvLLMProviderCommand, "")

	_, err := NewTestProvider("llm")
	if err == nil || !strings.Contains(err.Error(), EnvLLMProviderCommand) {
		t.Fatalf("NewTestProvider(llm) error = %v, want missing command error", err)
	}
}

func TestNewTestProviderReturnsLLMAndRejectsUnsupportedMode(t *testing.T) {
	t.Setenv(EnvLLMProviderCommand, "testloop-fake-provider")

	provider, err := NewTestProvider(" llm ")
	if err != nil {
		t.Fatalf("NewTestProvider(llm) error = %v", err)
	}
	if provider.Name() != "llm-command" {
		t.Fatalf("provider.Name() = %q, want llm-command", provider.Name())
	}

	_, err = NewTestProvider("unknown")
	if err == nil || !strings.Contains(err.Error(), "unsupported test provider") {
		t.Fatalf("NewTestProvider(unknown) error = %v, want unsupported provider error", err)
	}
}

func TestStaticProviderUsesProvidedStaticCode(t *testing.T) {
	code, err := StaticProvider{}.GenerateTests(context.Background(), TestGenerationRequest{
		SourceFile: "missing.py",
		StaticCode: "def test_static():\n    assert True\n",
	})
	if err != nil {
		t.Fatalf("StaticProvider.GenerateTests() error = %v", err)
	}
	if !strings.Contains(code, "test_static") {
		t.Fatalf("expected static code passthrough, got:\n%s", code)
	}
}

func TestStaticProviderGeneratesCoverageTaskWhenNoStaticCode(t *testing.T) {
	srcPath := writeProviderSource(t)
	task := testCoverageTask()

	code, err := StaticProvider{}.GenerateTests(context.Background(), TestGenerationRequest{
		SourceFile: srcPath,
		Context: &types.TestGenerationContext{
			CoverageTask: &task,
		},
	})
	if err != nil {
		t.Fatalf("StaticProvider.GenerateTests() error = %v", err)
	}
	if !strings.Contains(code, "def test_add_covers_gap():") || !strings.Contains(code, "coverage task: pytest-1") {
		t.Fatalf("expected task-aware static output, got:\n%s", code)
	}
}

func TestGenerateTestsWithProviderOptionsDefaultsToStaticProvider(t *testing.T) {
	srcPath := writeProviderSource(t)

	code, err := GenerateTestsWithProviderOptions(context.Background(), srcPath, nil, GenerateTestsOptions{})
	if err != nil {
		t.Fatalf("GenerateTestsWithProviderOptions() error = %v", err)
	}
	if !strings.Contains(code, "def test_add():") {
		t.Fatalf("expected default static provider output, got:\n%s", code)
	}
}

func TestGenerateTestsWithProviderOptionsUsesJavaScriptFramework(t *testing.T) {
	srcPath := writeProviderJavaScriptSource(t)

	code, err := GenerateTestsWithProviderOptions(context.Background(), srcPath, StaticProvider{}, GenerateTestsOptions{Framework: "mocha"})
	if err != nil {
		t.Fatalf("GenerateTestsWithProviderOptions() error = %v", err)
	}
	for _, want := range []string{
		"const { expect } = require('chai');",
		"const result = add(1, 2);",
		"expect(result).to.equal((1 + 2));",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in Mocha output:\n%s", want, code)
		}
	}
	for _, forbidden := range []string{"toBe((1 + 2))", "toThrow()"} {
		if strings.Contains(code, forbidden) {
			t.Fatalf("did not expect %q in Mocha output:\n%s", forbidden, code)
		}
	}
}

func TestGenerateTestsWithProviderOptionsAutoDetectsJavaScriptFramework(t *testing.T) {
	srcPath := writeProviderJavaScriptPackageSource(t, `{
  "scripts": { "test": "mocha --reporter spec" },
  "devDependencies": { "jest": "^29.0.0" }
}`)

	code, err := GenerateTestsWithProviderOptions(context.Background(), srcPath, StaticProvider{}, GenerateTestsOptions{})
	if err != nil {
		t.Fatalf("GenerateTestsWithProviderOptions() error = %v", err)
	}
	for _, want := range []string{
		"const { expect } = require('chai');",
		"expect(result).to.equal((1 + 2));",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in auto-detected Mocha output:\n%s", want, code)
		}
	}
	if strings.Contains(code, "toBe((1 + 2))") {
		t.Fatalf("expected auto-detected Mocha assertions, got:\n%s", code)
	}
}

func TestGenerateTestsWithProviderUsesExternalLLMCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake provider is unix-only")
	}

	srcPath := writeProviderSource(t)
	command := writeFakeProviderCommand(t, `#!/bin/sh
cat <<'EOF'
{"code":"from calc import add\n\n\ndef test_generated_by_llm():\n    assert add(1, 2) == 3\n"}
EOF
`)

	code, err := GenerateTestsWithProvider(context.Background(), srcPath, ExternalLLMProvider{Command: command})
	if err != nil {
		t.Fatalf("GenerateTestsWithProvider() error = %v", err)
	}
	if !strings.Contains(code, "test_generated_by_llm") {
		t.Fatalf("expected llm provider output, got:\n%s", code)
	}
}

func TestGenerateTestsWithProviderRejectsExternalLLMNonTestCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake provider is unix-only")
	}

	srcPath := writeProviderJavaScriptSource(t)
	command := writeFakeProviderCommand(t, `#!/bin/sh
cat <<'EOF'
{"code":"const value = add(1, 2);\nconsole.log(value);\n"}
EOF
`)

	_, err := GenerateTestsWithProvider(context.Background(), srcPath, ExternalLLMProvider{Command: command})
	if err == nil || !strings.Contains(err.Error(), "did not look like javascript test code") {
		t.Fatalf("GenerateTestsWithProvider() error = %v, want non-test code error", err)
	}
}

func TestParseLLMProviderOutputCleansMarkdownFence(t *testing.T) {
	raw := []byte("Here is the test:\n\n```python\nfrom calc import add\n\n\ndef test_add():\n    assert add(1, 2) == 3\n```\n\nThis covers the happy path.\n")

	code, err := parseLLMProviderOutput(raw)
	if err != nil {
		t.Fatalf("parseLLMProviderOutput() error = %v", err)
	}
	if strings.Contains(code, "Here is") || strings.Contains(code, "```") || strings.Contains(code, "This covers") {
		t.Fatalf("expected markdown prose to be stripped, got:\n%s", code)
	}
	if !strings.Contains(code, "def test_add():") || !strings.Contains(code, "assert add(1, 2) == 3") {
		t.Fatalf("expected fenced code, got:\n%s", code)
	}
}

func TestParseLLMProviderJSONOutputCleansMarkdownFence(t *testing.T) {
	raw := []byte("{\"code\":\"```javascript\\nconst { add } = require('./calc');\\n\\ntest('adds', () => {\\n  expect(add(1, 2)).toBe(3);\\n});\\n```\"}")

	code, err := parseLLMProviderOutput(raw)
	if err != nil {
		t.Fatalf("parseLLMProviderOutput() error = %v", err)
	}
	if strings.Contains(code, "```") {
		t.Fatalf("expected code fence to be stripped, got:\n%s", code)
	}
	if !strings.Contains(code, "const { add } = require('./calc');") || !strings.Contains(code, "expect(add(1, 2)).toBe(3);") {
		t.Fatalf("expected JS test code, got:\n%s", code)
	}
}

func TestParseLLMProviderOutputRejectsExplanationOnly(t *testing.T) {
	_, err := parseLLMProviderOutput([]byte("I would test the add function by checking a simple happy path."))
	if err == nil || !strings.Contains(err.Error(), "did not contain test code") {
		t.Fatalf("parseLLMProviderOutput() error = %v, want missing code error", err)
	}
}

func TestValidateLLMProviderTestCodeAcceptsLanguagePatterns(t *testing.T) {
	tests := []struct {
		name string
		path string
		code string
	}{
		{name: "go", path: "calc.go", code: "package calc\n\nfunc TestAdd(t *testing.T) {}\n"},
		{name: "python", path: "calc.py", code: "def test_add():\n    assert add(1, 2) == 3\n"},
		{name: "javascript", path: "calc.js", code: "test('adds', () => {\n  expect(add(1, 2)).toBe(3);\n});\n"},
		{name: "typescript", path: "calc.ts", code: "import { it, expect } from 'vitest';\n\nit('adds', () => expect(add(1, 2)).toBe(3));\n"},
		{name: "rust", path: "calc.rs", code: "#[test]\nfn test_add() {\n    assert_eq!(add(1, 2), 3);\n}\n"},
		{name: "java", path: "Calculator.java", code: "import org.junit.jupiter.api.Test;\n\nclass CalculatorTest {\n  @Test\n  void adds() {}\n}\n"},
		{name: "unknown", path: "calc.txt", code: "not test code"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateLLMProviderTestCode(tt.path, tt.code); err != nil {
				t.Fatalf("validateLLMProviderTestCode() error = %v", err)
			}
		})
	}
}

func TestValidateLLMProviderTestCodeRejectsLanguageNonTests(t *testing.T) {
	tests := []struct {
		name string
		path string
		code string
		want string
	}{
		{name: "go", path: "calc.go", code: "package calc\n\nfunc Add(a, b int) int { return a + b }\n", want: "go"},
		{name: "python", path: "calc.py", code: "def helper():\n    return add(1, 2)\n", want: "python"},
		{name: "javascript", path: "calc.js", code: "const value = add(1, 2);\nconsole.log(value);\n", want: "javascript"},
		{name: "typescript", path: "calc.ts", code: "export const value: number = add(1, 2);\n", want: "typescript"},
		{name: "rust", path: "calc.rs", code: "pub fn helper() -> i32 {\n    add(1, 2)\n}\n", want: "rust"},
		{name: "java", path: "Calculator.java", code: "class CalculatorTest {\n  void adds() {}\n}\n", want: "java"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLLMProviderTestCode(tt.path, tt.code)
			if err == nil || !strings.Contains(err.Error(), "did not look like "+tt.want+" test code") {
				t.Fatalf("validateLLMProviderTestCode() error = %v, want %s non-test error", err, tt.want)
			}
		})
	}
}

func TestExternalLLMProviderRejectsEmptyCommand(t *testing.T) {
	_, err := ExternalLLMProvider{}.GenerateTests(context.Background(), TestGenerationRequest{})
	if err == nil || !strings.Contains(err.Error(), EnvLLMProviderCommand+" is empty") {
		t.Fatalf("ExternalLLMProvider.GenerateTests() error = %v, want empty command error", err)
	}
}

func TestExternalLLMProviderReturnsCommandFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake provider is unix-only")
	}

	command := writeFakeProviderCommand(t, `#!/bin/sh
echo "provider exploded" >&2
exit 7
`)

	_, err := ExternalLLMProvider{Command: command}.GenerateTests(context.Background(), TestGenerationRequest{})
	if err == nil || !strings.Contains(err.Error(), "provider exploded") {
		t.Fatalf("ExternalLLMProvider.GenerateTests() error = %v, want stderr failure", err)
	}
	providerErr, ok := ProviderErrorInfo(err)
	if !ok || providerErr.Kind != ProviderErrorCommandFailed {
		t.Fatalf("ProviderErrorInfo() = %+v, %v; want command failed", providerErr, ok)
	}
}

func TestGenerateTestsWithProviderIncludesCoverageTask(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake provider is unix-only")
	}

	srcPath := writeProviderSource(t)
	requestPath := filepath.Join(t.TempDir(), "request.json")
	command := writeFakeProviderCommand(t, `#!/bin/sh
cat > "`+requestPath+`"
cat <<'EOF'
{"code":"from calc import add\n\n\ndef test_coverage_task():\n    assert add(1, 2) == 3\n"}
EOF
`)

	coverageTask := testCoverageTask()
	code, err := GenerateTestsWithProviderOptions(context.Background(), srcPath, ExternalLLMProvider{Command: command}, GenerateTestsOptions{
		CoverageTask: &coverageTask,
	})
	if err != nil {
		t.Fatalf("GenerateTestsWithProviderOptions() error = %v", err)
	}
	if !strings.Contains(code, "test_coverage_task") {
		t.Fatalf("expected llm provider output, got:\n%s", code)
	}

	raw, err := os.ReadFile(requestPath)
	if err != nil {
		t.Fatal(err)
	}
	var req TestGenerationRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("parse provider request: %v\n%s", err, raw)
	}
	if req.Context == nil || req.Context.CoverageTask == nil {
		t.Fatalf("expected coverage task in provider context, got %+v", req.Context)
	}
	if req.Context.CoverageTask.Target != "add" || req.Context.CoverageTask.TestName != "test_add_covers_gap" {
		t.Fatalf("unexpected coverage task context: %+v", req.Context.CoverageTask)
	}
}

func TestGenerateTestsWithProviderIncludesJavaScriptPayloadFallbackNotes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake provider is unix-only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "api.ts")
	src := `import type { ExternalUser } from './types';

export async function loadUser(response: Response): Promise<ExternalUser> {
  return await response.json();
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	requestPath := filepath.Join(t.TempDir(), "request.json")
	command := writeFakeProviderCommand(t, `#!/bin/sh
cat > "`+requestPath+`"
cat <<'EOF'
{"code":"import { loadUser } from './api';\n\nit('uses llm output', async () => {\n  await loadUser({ json: async () => ({ ok: true }) });\n});\n"}
EOF
`)

	code, err := GenerateTestsWithProviderOptions(context.Background(), srcPath, ExternalLLMProvider{Command: command}, GenerateTestsOptions{
		Framework: "vitest",
	})
	if err != nil {
		t.Fatalf("GenerateTestsWithProviderOptions() error = %v", err)
	}
	if !strings.Contains(code, "uses llm output") {
		t.Fatalf("expected llm provider output, got:\n%s", code)
	}

	raw, err := os.ReadFile(requestPath)
	if err != nil {
		t.Fatal(err)
	}
	var req TestGenerationRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("parse provider request: %v\n%s", err, raw)
	}
	if req.Context == nil {
		t.Fatalf("expected provider context, got %+v", req)
	}
	target := findProviderTarget(req.Context.Targets, "loadUser")
	if target == nil {
		t.Fatalf("loadUser target not found: %+v", req.Context.Targets)
	}
	if target.ReturnTypeExpr != "Promise<ExternalUser>" {
		t.Fatalf("return_type_expr = %q, want Promise<ExternalUser>", target.ReturnTypeExpr)
	}
	assertProviderSliceContains(t, target.PayloadNotes, "return annotation ExternalUser is not declared in the same source file; static payload falls back to { ok: true }")
	assertProviderSliceContains(t, target.PayloadNotes, "return annotation references imported type ExternalUser from './types'; read candidate source files: types.ts, types.tsx, types.d.ts, types.js, types.jsx, types.mjs, types.cjs, types/index.ts, types/index.tsx, types/index.d.ts, types/index.js, types/index.jsx, types/index.mjs, types/index.cjs")
	if !strings.Contains(req.StaticCode, "expect(result).toEqual({ ok: true });") {
		t.Fatalf("provider static_code missing fallback assertion:\n%s", req.StaticCode)
	}
}

func TestGenerateTestsWithProviderOptionsRejectsProviderEmptyOutput(t *testing.T) {
	srcPath := writeProviderSource(t)

	_, err := GenerateTestsWithProviderOptions(context.Background(), srcPath, emptyProvider{}, GenerateTestsOptions{})
	if err == nil || !strings.Contains(err.Error(), "returned empty output") {
		t.Fatalf("GenerateTestsWithProviderOptions() error = %v, want empty provider output error", err)
	}
}

func TestGenerateTestsWithProviderOptionsPropagatesStaticGenerationError(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(srcPath, []byte("not source"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := GenerateTestsWithProviderOptions(context.Background(), srcPath, StaticProvider{}, GenerateTestsOptions{})
	if err == nil || !strings.Contains(err.Error(), "不支持的文件类型") {
		t.Fatalf("GenerateTestsWithProviderOptions() error = %v, want unsupported type error", err)
	}
}

func TestGenerateTestsWithProviderOptionsUsesPythonCoverageTask(t *testing.T) {
	srcPath := writeProviderSource(t)
	coverageTask := testCoverageTask()

	code, err := GenerateTestsWithProviderOptions(context.Background(), srcPath, StaticProvider{}, GenerateTestsOptions{
		CoverageTask: &coverageTask,
	})
	if err != nil {
		t.Fatalf("GenerateTestsWithProviderOptions() error = %v", err)
	}
	if !strings.Contains(code, "def test_add_covers_gap():") {
		t.Fatalf("expected task test name in static output, got:\n%s", code)
	}
	if !strings.Contains(code, "coverage task: pytest-1") {
		t.Fatalf("expected coverage task comment in static output, got:\n%s", code)
	}
}

func TestGenerateTestsWithProviderOptionsUsesJavaCoverageTask(t *testing.T) {
	src := `public class Calculator {
    public int add(int a, int b) {
        return a + b;
    }

    public int sub(int a, int b) {
        return a - b;
    }
}
`
	srcPath := filepath.Join(t.TempDir(), "Calculator.java")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	coverageTask := types.CoverageTestTask{
		ID:        "java-1",
		Framework: "junit",
		Target:    "Calculator.add",
		LineRange: "2-2",
		GapType:   "branch",
		TestName:  "shouldCoverCalculatorAddGap",
	}

	code, err := GenerateTestsWithProviderOptions(context.Background(), srcPath, StaticProvider{}, GenerateTestsOptions{
		CoverageTask: &coverageTask,
	})
	if err != nil {
		t.Fatalf("GenerateTestsWithProviderOptions() error = %v", err)
	}
	if !strings.Contains(code, "void shouldCoverCalculatorAddGap()") || strings.Contains(code, "instance.sub(") {
		t.Fatalf("expected task-aware Java static output, got:\n%s", code)
	}
}

func TestGenerateTestsWithProviderOptionsUsesJavaScriptCoverageTask(t *testing.T) {
	src := `function add(a, b) {
  return a + b;
}

function sub(a, b) {
  return a - b;
}

module.exports = { add, sub };
`
	srcPath := filepath.Join(t.TempDir(), "calc.cjs")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	coverageTask := types.CoverageTestTask{
		ID:        "jest-1",
		Framework: "jest",
		Target:    "add",
		LineRange: "2-2",
		GapType:   "return_path",
		TestName:  "covers add gap",
	}

	code, err := GenerateTestsWithProviderOptions(context.Background(), srcPath, StaticProvider{}, GenerateTestsOptions{
		CoverageTask: &coverageTask,
	})
	if err != nil {
		t.Fatalf("GenerateTestsWithProviderOptions() error = %v", err)
	}
	if !strings.Contains(code, "it('covers add gap'") || strings.Contains(code, "describe('sub'") {
		t.Fatalf("expected task-aware JavaScript static output, got:\n%s", code)
	}
}

func TestGenerateTestsWithProviderOptionsUsesRustCoverageTask(t *testing.T) {
	src := `pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

pub fn sub(a: i32, b: i32) -> i32 {
    a - b
}
`
	srcPath := filepath.Join(t.TempDir(), "calc.rs")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	coverageTask := types.CoverageTestTask{
		ID:        "rust-1",
		Framework: "cargo-test",
		Target:    "add",
		LineRange: "1-3",
		GapType:   "branch",
		TestName:  "test_add_gap",
	}

	code, err := GenerateTestsWithProviderOptions(context.Background(), srcPath, StaticProvider{}, GenerateTestsOptions{
		CoverageTask: &coverageTask,
	})
	if err != nil {
		t.Fatalf("GenerateTestsWithProviderOptions() error = %v", err)
	}
	if !strings.Contains(code, "fn test_add_gap()") || strings.Contains(code, "test_sub") {
		t.Fatalf("expected task-aware Rust static output, got:\n%s", code)
	}
}

func TestGenerateTestsForCoverageTaskReportsReadErrorsAndUnsupportedType(t *testing.T) {
	task := testCoverageTask()
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "missing rust", path: filepath.Join(t.TempDir(), "missing.rs"), want: "读取 Rust 源文件失败"},
		{name: "missing java", path: filepath.Join(t.TempDir(), "Missing.java"), want: "读取 Java 源文件失败"},
		{name: "unsupported", path: filepath.Join(t.TempDir(), "notes.txt"), want: "不支持的文件类型"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := generateTestsForCoverageTask(tt.path, &task)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("generateTestsForCoverageTask() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestParseLLMProviderOutputAcceptsRawCode(t *testing.T) {
	code, err := parseLLMProviderOutput([]byte("package demo\n\nfunc TestRaw(t *testing.T) {}\n"))
	if err != nil {
		t.Fatalf("parseLLMProviderOutput() error = %v", err)
	}
	if !strings.Contains(code, "TestRaw") {
		t.Fatalf("unexpected raw code output: %q", code)
	}
}

func TestParseLLMProviderOutputRejectsInvalidResponses(t *testing.T) {
	tests := []struct {
		name string
		out  []byte
		want string
	}{
		{name: "empty", out: []byte("  \n"), want: "empty output"},
		{name: "invalid json", out: []byte("{not-json"), want: "parse llm provider json output"},
		{name: "missing code", out: []byte(`{"code":"   "}`), want: "missing code"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseLLMProviderOutput(tt.out)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parseLLMProviderOutput() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestProviderErrorInfoClassifiesLLMProviderFailures(t *testing.T) {
	tests := []struct {
		name string
		err  error
		kind ProviderErrorKind
	}{
		{name: "empty output", err: mustProviderErr(parseLLMProviderOutput([]byte("  \n"))), kind: ProviderErrorEmptyOutput},
		{name: "json error", err: mustProviderErr(parseLLMProviderOutput([]byte("{not-json"))), kind: ProviderErrorJSON},
		{name: "missing code", err: mustProviderErr(parseLLMProviderOutput([]byte(`{"code":"   "}`))), kind: ProviderErrorMissingCode},
		{name: "cleaning failed", err: mustProviderErr(parseLLMProviderOutput([]byte("I would test the add function."))), kind: ProviderErrorOutputCleaningFailed},
		{name: "validation failed", err: validateLLMProviderTestCode("calc.go", "package calc\nfunc Add(a, b int) int { return a + b }\n"), kind: ProviderErrorOutputValidationFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerErr, ok := ProviderErrorInfo(tt.err)
			if !ok {
				t.Fatalf("ProviderErrorInfo() ok = false for %v", tt.err)
			}
			if providerErr.Kind != tt.kind {
				t.Fatalf("ProviderErrorInfo().Kind = %q, want %q", providerErr.Kind, tt.kind)
			}
			if providerErr.Provider != "llm-command" {
				t.Fatalf("ProviderErrorInfo().Provider = %q, want llm-command", providerErr.Provider)
			}
		})
	}
}

func mustProviderErr(_ string, err error) error {
	return err
}

type emptyProvider struct{}

func (emptyProvider) Name() string {
	return "empty"
}

func (emptyProvider) GenerateTests(context.Context, TestGenerationRequest) (string, error) {
	return " \n", nil
}

func testCoverageTask() types.CoverageTestTask {
	return types.CoverageTestTask{
		ID:             "pytest-1",
		Framework:      "pytest",
		File:           "calc.py",
		Target:         "add",
		LineRange:      "2-2",
		GapType:        "branch",
		TestFile:       filepath.Join("tests", "test_calc.py"),
		TestName:       "test_add_covers_gap",
		AssertionFocus: []string{"断言未覆盖分支"},
		Priority:       100,
	}
}

func writeProviderSource(t *testing.T) string {
	t.Helper()

	src := `def add(a, b):
    return a + b
`
	path := filepath.Join(t.TempDir(), "calc.py")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeProviderJavaScriptSource(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	return writeProviderJavaScriptSourceInDir(t, dir)
}

func writeProviderJavaScriptPackageSource(t *testing.T, packageJSON string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}
	return writeProviderJavaScriptSourceInDir(t, dir)
}

func writeProviderJavaScriptSourceInDir(t *testing.T, dir string) string {
	t.Helper()

	src := `function add(a, b) {
  return a + b;
}

module.exports = { add };
`
	path := filepath.Join(dir, "calc.js")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func findProviderTarget(targets []types.TestTarget, name string) *types.TestTarget {
	for i := range targets {
		if targets[i].Name == name {
			return &targets[i]
		}
	}
	return nil
}

func assertProviderSliceContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("expected %q in %+v", want, values)
}

func writeFakeProviderCommand(t *testing.T, script string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "provider")
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}
