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

func TestParseLLMProviderOutputAcceptsRawCode(t *testing.T) {
	code, err := parseLLMProviderOutput([]byte("package demo\n\nfunc TestRaw(t *testing.T) {}\n"))
	if err != nil {
		t.Fatalf("parseLLMProviderOutput() error = %v", err)
	}
	if !strings.Contains(code, "TestRaw") {
		t.Fatalf("unexpected raw code output: %q", code)
	}
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

func writeFakeProviderCommand(t *testing.T, script string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "provider")
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}
