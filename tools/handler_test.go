package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/internal/generator"
	"github.com/sleticalboy/testloop-mcp/types"
)

func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil {
		t.Fatal("expected tool result")
	}
	if len(result.Content) != 1 {
		t.Fatalf("content len = %d, want 1", len(result.Content))
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content type = %T, want *mcp.TextContent", result.Content[0])
	}
	return text.Text
}

func structuredContentAs[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()
	if result == nil {
		t.Fatal("expected tool result")
	}
	typed, ok := result.StructuredContent.(T)
	if !ok {
		var zero T
		t.Fatalf("structured content type = %T, want %T", result.StructuredContent, zero)
	}
	return typed
}

func assertGenerateTestsProviderError(t *testing.T, result *mcp.CallToolResult, structured any, wantKind, wantAction string) {
	t.Helper()
	if result == nil {
		t.Fatal("expected provider error tool result")
	}
	if !result.IsError {
		t.Fatal("result.IsError = false, want true")
	}
	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &generated); err != nil {
		t.Fatalf("unmarshal provider error result: %v", err)
	}
	if generated.Status != "error" {
		t.Fatalf("status = %q, want error", generated.Status)
	}
	if generated.ProviderError == nil {
		t.Fatalf("provider_error missing in result: %+v", generated)
	}
	if generated.ProviderError.Kind != wantKind || generated.ProviderError.Action != wantAction {
		t.Fatalf("provider_error = %+v, want kind=%s action=%s", generated.ProviderError, wantKind, wantAction)
	}
	wantText := "provider_error kind=" + wantKind + " action=" + wantAction
	if !strings.Contains(generated.Error, wantText) {
		t.Fatalf("error text missing %q: %q", wantText, generated.Error)
	}
	structuredOutput, ok := structured.(types.GenerateTestsOutput)
	if !ok {
		t.Fatalf("structured output type = %T, want types.GenerateTestsOutput", structured)
	}
	if structuredOutput.ProviderError == nil ||
		structuredOutput.ProviderError.Kind != wantKind ||
		structuredOutput.ProviderError.Action != wantAction {
		t.Fatalf("structured provider_error = %+v, want kind=%s action=%s", structuredOutput.ProviderError, wantKind, wantAction)
	}
	if result.StructuredContent == nil {
		t.Fatal("result.StructuredContent is nil")
	}
}

func findGeneratedTarget(targets []types.TestTarget, name string) *types.TestTarget {
	for i := range targets {
		if targets[i].Name == name {
			return &targets[i]
		}
	}
	return nil
}

func assertStringSliceContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("expected %q in %+v", want, values)
}

func TestHandleParseResultsDefaultsToGoTest(t *testing.T) {
	output := strings.Join([]string{
		`{"Action":"run","Package":"example.com/calc","Test":"TestAdd"}`,
		`{"Action":"pass","Package":"example.com/calc","Test":"TestAdd","Elapsed":0}`,
		`{"Action":"pass","Package":"example.com/calc","Elapsed":0}`,
	}, "\n")

	result, _, err := HandleParseResults(context.Background(), nil, parseResultsInput{Output: output})
	if err != nil {
		t.Fatalf("HandleParseResults returned error: %v", err)
	}

	var parsed types.TestResult
	if err := json.Unmarshal([]byte(resultText(t, result)), &parsed); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if parsed.Framework != "go-test" || parsed.Status != "pass" || parsed.Passed != 1 || parsed.Failed != 0 {
		t.Fatalf("unexpected parsed result: %+v", parsed)
	}
	if parsed.Action != "ready" {
		t.Fatalf("action = %q, want ready", parsed.Action)
	}
	structured := structuredContentAs[types.TestResult](t, result)
	if structured.Framework != parsed.Framework || structured.Status != parsed.Status || structured.Action != parsed.Action || structured.Passed != parsed.Passed {
		t.Fatalf("structured content mismatch: %+v vs %+v", structured, parsed)
	}
}

func TestHandleParseResultsRequiresOutput(t *testing.T) {
	if _, _, err := HandleParseResults(context.Background(), nil, parseResultsInput{}); err == nil {
		t.Fatal("expected missing output error")
	}
}

func TestParseResultsCompatibilityDefaultsToGoTest(t *testing.T) {
	output := strings.Join([]string{
		`{"Action":"run","Package":"example.com/calc","Test":"TestAdd"}`,
		`{"Action":"fail","Package":"example.com/calc","Test":"TestAdd","Elapsed":0}`,
		`{"Action":"fail","Package":"example.com/calc","Elapsed":0}`,
	}, "\n")

	result, err := parseResults(parseResultsInput{Output: output})
	if err != nil {
		t.Fatalf("parseResults returned error: %v", err)
	}
	if result.Framework != "go-test" || result.Status != "fail" || result.Failed != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestHandleParseCoverageParsesGoCoverprofile(t *testing.T) {
	data := strings.Join([]string{
		"mode: set",
		"calc.go:1.1,2.1 1 1",
		"calc.go:3.1,4.1 1 0",
	}, "\n")

	result, _, err := HandleParseCoverage(context.Background(), nil, parseCoverageInput{Data: data})
	if err != nil {
		t.Fatalf("HandleParseCoverage returned error: %v", err)
	}

	var report types.CoverageReport
	if err := json.Unmarshal([]byte(resultText(t, result)), &report); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if report.Framework != "go-test" || report.TotalPercent != 50 {
		t.Fatalf("unexpected report: %+v", report)
	}
	structured := structuredContentAs[*types.CoverageReport](t, result)
	if structured.Framework != report.Framework || structured.TotalPercent != report.TotalPercent {
		t.Fatalf("structured content mismatch: %+v vs %+v", structured, report)
	}
}

func TestHandleParseCoverageValidatesInput(t *testing.T) {
	if _, _, err := HandleParseCoverage(context.Background(), nil, parseCoverageInput{}); err == nil {
		t.Fatal("expected missing data error")
	}
	if _, _, err := HandleParseCoverage(context.Background(), nil, parseCoverageInput{Data: "not coverage data"}); err == nil {
		t.Fatal("expected invalid coverage data error")
	}
}

type coverageTaskGenerateLoopGolden struct {
	ParsedTask types.CoverageTestTask                `json:"parsed_task"`
	Generated  coverageTaskGenerateLoopGeneratedPart `json:"generated"`
}

type coverageTaskGenerateLoopGeneratedPart struct {
	Status              string                 `json:"status"`
	Provider            string                 `json:"provider"`
	TestFile            string                 `json:"test_file"`
	GeneratedCases      int                    `json:"generated_cases"`
	CoverageTask        types.CoverageTestTask `json:"coverage_task"`
	ContextCoverageTask types.CoverageTestTask `json:"context_coverage_task"`
	TestFileContent     string                 `json:"test_file_content"`
}

func TestHandleParseCoverageGenerateTestsLoopGolden(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, filepath.Join("pkg", "calc.go"), strings.Join([]string{
		"package calc",
		"",
		"func Add(a, b int) int {",
		"	if a == 0 {",
		"		return b",
		"	}",
		"	return a + b",
		"}",
	}, "\n")+"\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir fixture: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	coverageProfile := strings.Join([]string{
		"mode: set",
		"pkg/calc.go:4.2,4.10 1 0",
		"pkg/calc.go:7.2,7.14 1 1",
	}, "\n")
	coverageResult, _, err := HandleParseCoverage(context.Background(), nil, parseCoverageInput{
		Data:      coverageProfile,
		Framework: "go-test",
	})
	if err != nil {
		t.Fatalf("HandleParseCoverage returned error: %v", err)
	}

	var report types.CoverageReport
	if err := json.Unmarshal([]byte(resultText(t, coverageResult)), &report); err != nil {
		t.Fatalf("unmarshal coverage report: %v", err)
	}
	if len(report.TestTasks) != 1 {
		t.Fatalf("test tasks len = %d, want 1: %+v", len(report.TestTasks), report.TestTasks)
	}
	task := report.TestTasks[0]

	generateResult, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     task.File,
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, generateResult)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	if generated.CoverageTask == nil || generated.Context == nil || generated.Context.CoverageTask == nil {
		t.Fatalf("expected coverage task in generated output and context: %+v", generated)
	}
	content, err := os.ReadFile(task.TestFile)
	if err != nil {
		t.Fatalf("read generated test file: %v", err)
	}

	gotBytes, err := json.MarshalIndent(coverageTaskGenerateLoopGolden{
		ParsedTask: task,
		Generated: coverageTaskGenerateLoopGeneratedPart{
			Status:              generated.Status,
			Provider:            generated.Provider,
			TestFile:            generated.TestFile,
			GeneratedCases:      generated.GeneratedCases,
			CoverageTask:        *generated.CoverageTask,
			ContextCoverageTask: *generated.Context.CoverageTask,
			TestFileContent:     string(content),
		},
	}, "", "  ")
	if err != nil {
		t.Fatalf("marshal loop golden: %v", err)
	}
	got := string(gotBytes) + "\n"
	wantBytes, err := os.ReadFile(filepath.Join(oldWD, "testdata", "golden", "coverage_task_generate_loop.golden"))
	if err != nil {
		t.Fatalf("read golden: %v\n--- got ---\n%s", err, got)
	}
	want := string(wantBytes)
	if strings.TrimSpace(got) != strings.TrimSpace(want) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestHandleParseCoverageGenerateTestsPytestLoopGolden(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, filepath.Join("src", "service.py"), strings.Join([]string{
		"def status(value):",
		"    if value == \"active\":",
		"        return \"ok\"",
		"    return \"idle\"",
	}, "\n")+"\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir fixture: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	coverageJSON := `{
  "totals": {
    "covered_lines": 3,
    "num_statements": 4,
    "percent_covered": 75.0,
    "missing_lines": 1
  },
  "files": {
    "src/service.py": {
      "executed_lines": [1, 3, 4],
      "missing_lines": [2],
      "summary": {
        "covered_lines": 3,
        "num_statements": 4,
        "percent_covered": 75.0,
        "missing_lines": 1
      }
    }
  }
}`
	coverageResult, _, err := HandleParseCoverage(context.Background(), nil, parseCoverageInput{
		Data:      coverageJSON,
		Framework: "pytest",
	})
	if err != nil {
		t.Fatalf("HandleParseCoverage returned error: %v", err)
	}

	var report types.CoverageReport
	if err := json.Unmarshal([]byte(resultText(t, coverageResult)), &report); err != nil {
		t.Fatalf("unmarshal coverage report: %v", err)
	}
	if len(report.TestTasks) != 1 {
		t.Fatalf("test tasks len = %d, want 1: %+v", len(report.TestTasks), report.TestTasks)
	}
	task := report.TestTasks[0]

	generateResult, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     task.File,
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, generateResult)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	if generated.CoverageTask == nil || generated.Context == nil || generated.Context.CoverageTask == nil {
		t.Fatalf("expected coverage task in generated output and context: %+v", generated)
	}
	content, err := os.ReadFile(task.TestFile)
	if err != nil {
		t.Fatalf("read generated test file: %v", err)
	}

	gotBytes, err := json.MarshalIndent(coverageTaskGenerateLoopGolden{
		ParsedTask: task,
		Generated: coverageTaskGenerateLoopGeneratedPart{
			Status:              generated.Status,
			Provider:            generated.Provider,
			TestFile:            generated.TestFile,
			GeneratedCases:      generated.GeneratedCases,
			CoverageTask:        *generated.CoverageTask,
			ContextCoverageTask: *generated.Context.CoverageTask,
			TestFileContent:     string(content),
		},
	}, "", "  ")
	if err != nil {
		t.Fatalf("marshal loop golden: %v", err)
	}
	got := string(gotBytes) + "\n"
	wantBytes, err := os.ReadFile(filepath.Join(oldWD, "testdata", "golden", "coverage_task_generate_pytest_loop.golden"))
	if err != nil {
		t.Fatalf("read golden: %v\n--- got ---\n%s", err, got)
	}
	want := string(wantBytes)
	if strings.TrimSpace(got) != strings.TrimSpace(want) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestHandleParseCoverageGenerateTestsJestLoopGolden(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, filepath.Join("src", "sum.js"), strings.Join([]string{
		"export function add(a, b) {",
		"  if (a === 0) return b;",
		"  return a + b;",
		"}",
	}, "\n")+"\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir fixture: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	coverageJSON := `{
  "src/sum.js": {
    "path": "src/sum.js",
    "statementMap": {
      "0": {"start": {"line": 1, "column": 0}, "end": {"line": 1, "column": 28}},
      "1": {"start": {"line": 2, "column": 2}, "end": {"line": 2, "column": 26}},
      "2": {"start": {"line": 3, "column": 2}, "end": {"line": 3, "column": 15}}
    },
    "s": {"0": 1, "1": 0, "2": 1},
    "fnMap": {},
    "f": {},
    "branchMap": {},
    "b": {}
  }
}`
	coverageResult, _, err := HandleParseCoverage(context.Background(), nil, parseCoverageInput{
		Data:      coverageJSON,
		Framework: "jest",
	})
	if err != nil {
		t.Fatalf("HandleParseCoverage returned error: %v", err)
	}

	var report types.CoverageReport
	if err := json.Unmarshal([]byte(resultText(t, coverageResult)), &report); err != nil {
		t.Fatalf("unmarshal coverage report: %v", err)
	}
	if len(report.TestTasks) != 1 {
		t.Fatalf("test tasks len = %d, want 1: %+v", len(report.TestTasks), report.TestTasks)
	}
	task := report.TestTasks[0]

	generateResult, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     task.File,
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, generateResult)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	if generated.CoverageTask == nil || generated.Context == nil || generated.Context.CoverageTask == nil {
		t.Fatalf("expected coverage task in generated output and context: %+v", generated)
	}
	content, err := os.ReadFile(task.TestFile)
	if err != nil {
		t.Fatalf("read generated test file: %v", err)
	}

	gotBytes, err := json.MarshalIndent(coverageTaskGenerateLoopGolden{
		ParsedTask: task,
		Generated: coverageTaskGenerateLoopGeneratedPart{
			Status:              generated.Status,
			Provider:            generated.Provider,
			TestFile:            generated.TestFile,
			GeneratedCases:      generated.GeneratedCases,
			CoverageTask:        *generated.CoverageTask,
			ContextCoverageTask: *generated.Context.CoverageTask,
			TestFileContent:     string(content),
		},
	}, "", "  ")
	if err != nil {
		t.Fatalf("marshal loop golden: %v", err)
	}
	got := string(gotBytes) + "\n"
	wantBytes, err := os.ReadFile(filepath.Join(oldWD, "testdata", "golden", "coverage_task_generate_jest_loop.golden"))
	if err != nil {
		t.Fatalf("read golden: %v\n--- got ---\n%s", err, got)
	}
	want := string(wantBytes)
	if strings.TrimSpace(got) != strings.TrimSpace(want) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestHandleParseCoverageGenerateTestsVitestLoopGolden(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, filepath.Join("src", "sum.ts"), strings.Join([]string{
		"export function add(a: number, b: number): number {",
		"  if (a === 0) return b;",
		"  return a + b;",
		"}",
	}, "\n")+"\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir fixture: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	coverageJSON := `{
  "src/sum.ts": {
    "path": "src/sum.ts",
    "statementMap": {
      "0": {"start": {"line": 1, "column": 0}, "end": {"line": 1, "column": 52}},
      "1": {"start": {"line": 2, "column": 2}, "end": {"line": 2, "column": 26}},
      "2": {"start": {"line": 3, "column": 2}, "end": {"line": 3, "column": 15}}
    },
    "s": {"0": 1, "1": 0, "2": 1},
    "fnMap": {},
    "f": {},
    "branchMap": {},
    "b": {}
  }
}`
	coverageResult, _, err := HandleParseCoverage(context.Background(), nil, parseCoverageInput{
		Data:      coverageJSON,
		Framework: "vitest",
	})
	if err != nil {
		t.Fatalf("HandleParseCoverage returned error: %v", err)
	}

	var report types.CoverageReport
	if err := json.Unmarshal([]byte(resultText(t, coverageResult)), &report); err != nil {
		t.Fatalf("unmarshal coverage report: %v", err)
	}
	if len(report.TestTasks) != 1 {
		t.Fatalf("test tasks len = %d, want 1: %+v", len(report.TestTasks), report.TestTasks)
	}
	task := report.TestTasks[0]

	generateResult, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     task.File,
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, generateResult)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	if generated.CoverageTask == nil || generated.Context == nil || generated.Context.CoverageTask == nil {
		t.Fatalf("expected coverage task in generated output and context: %+v", generated)
	}
	content, err := os.ReadFile(task.TestFile)
	if err != nil {
		t.Fatalf("read generated test file: %v", err)
	}

	gotBytes, err := json.MarshalIndent(coverageTaskGenerateLoopGolden{
		ParsedTask: task,
		Generated: coverageTaskGenerateLoopGeneratedPart{
			Status:              generated.Status,
			Provider:            generated.Provider,
			TestFile:            generated.TestFile,
			GeneratedCases:      generated.GeneratedCases,
			CoverageTask:        *generated.CoverageTask,
			ContextCoverageTask: *generated.Context.CoverageTask,
			TestFileContent:     string(content),
		},
	}, "", "  ")
	if err != nil {
		t.Fatalf("marshal loop golden: %v", err)
	}
	got := string(gotBytes) + "\n"
	wantBytes, err := os.ReadFile(filepath.Join(oldWD, "testdata", "golden", "coverage_task_generate_vitest_loop.golden"))
	if err != nil {
		t.Fatalf("read golden: %v\n--- got ---\n%s", err, got)
	}
	want := string(wantBytes)
	if strings.TrimSpace(got) != strings.TrimSpace(want) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestHandleParseCoverageGenerateTestsMochaLoopGolden(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, filepath.Join("lib", "calc.js"), strings.Join([]string{
		"function divide(a, b) {",
		"  if (b === 0) return 0;",
		"  return a / b;",
		"}",
		"",
		"module.exports = { divide };",
	}, "\n")+"\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir fixture: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	coverageJSON := `{
  "lib/calc.js": {
    "path": "lib/calc.js",
    "statementMap": {
      "0": {"start": {"line": 1, "column": 0}, "end": {"line": 1, "column": 22}},
      "1": {"start": {"line": 2, "column": 2}, "end": {"line": 2, "column": 28}},
      "2": {"start": {"line": 3, "column": 2}, "end": {"line": 3, "column": 15}}
    },
    "s": {"0": 1, "1": 0, "2": 1},
    "fnMap": {},
    "f": {},
    "branchMap": {},
    "b": {}
  }
}`
	coverageResult, _, err := HandleParseCoverage(context.Background(), nil, parseCoverageInput{
		Data:      coverageJSON,
		Framework: "mocha",
	})
	if err != nil {
		t.Fatalf("HandleParseCoverage returned error: %v", err)
	}

	var report types.CoverageReport
	if err := json.Unmarshal([]byte(resultText(t, coverageResult)), &report); err != nil {
		t.Fatalf("unmarshal coverage report: %v", err)
	}
	if len(report.TestTasks) != 1 {
		t.Fatalf("test tasks len = %d, want 1: %+v", len(report.TestTasks), report.TestTasks)
	}
	task := report.TestTasks[0]

	generateResult, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     task.File,
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, generateResult)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	if generated.CoverageTask == nil || generated.Context == nil || generated.Context.CoverageTask == nil {
		t.Fatalf("expected coverage task in generated output and context: %+v", generated)
	}
	content, err := os.ReadFile(task.TestFile)
	if err != nil {
		t.Fatalf("read generated test file: %v", err)
	}

	gotBytes, err := json.MarshalIndent(coverageTaskGenerateLoopGolden{
		ParsedTask: task,
		Generated: coverageTaskGenerateLoopGeneratedPart{
			Status:              generated.Status,
			Provider:            generated.Provider,
			TestFile:            generated.TestFile,
			GeneratedCases:      generated.GeneratedCases,
			CoverageTask:        *generated.CoverageTask,
			ContextCoverageTask: *generated.Context.CoverageTask,
			TestFileContent:     string(content),
		},
	}, "", "  ")
	if err != nil {
		t.Fatalf("marshal loop golden: %v", err)
	}
	got := string(gotBytes) + "\n"
	wantBytes, err := os.ReadFile(filepath.Join(oldWD, "testdata", "golden", "coverage_task_generate_mocha_loop.golden"))
	if err != nil {
		t.Fatalf("read golden: %v\n--- got ---\n%s", err, got)
	}
	want := string(wantBytes)
	if strings.TrimSpace(got) != strings.TrimSpace(want) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestHandleGenerateTestsValidatesInput(t *testing.T) {
	if _, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{}); err == nil {
		t.Fatal("expected missing file_path error")
	}
	if _, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{FilePath: "missing.go"}); err == nil {
		t.Fatal("expected missing file error")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(source, []byte("package calc\nfunc Add(a, b int) int { return a + b }\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{FilePath: source, Provider: "unknown"}); err == nil {
		t.Fatal("expected unsupported provider error")
	}
}

func TestHandleGenerateTestsClassifiesLLMProviderConfigError(t *testing.T) {
	t.Setenv(generator.EnvLLMProviderCommand, "")
	dir := t.TempDir()
	source := writeTestFile(t, dir, "calc.go", "package calc\nfunc Add(a, b int) int { return a + b }\n")

	result, structured, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath: source,
		Provider: "llm",
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned protocol error: %v", err)
	}
	assertGenerateTestsProviderError(t, result, structured, "llm_config_missing", "configure_provider")
}

func TestHandleGenerateTestsClassifiesLLMProviderBadOutputs(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		wantKind   string
		wantAction string
	}{
		{
			name:       "empty output",
			output:     "",
			wantKind:   "llm_empty_output",
			wantAction: "retry_model_or_fallback_static",
		},
		{
			name:       "json parse error",
			output:     "{not-json",
			wantKind:   "llm_json_error",
			wantAction: "retry_model_or_fallback_static",
		},
		{
			name:       "json missing code",
			output:     `{"message":"no code today"}`,
			wantKind:   "llm_missing_code",
			wantAction: "retry_model_or_fallback_static",
		},
		{
			name:       "explanation only",
			output:     "I would test the add function by checking a simple happy path.",
			wantKind:   "llm_output_cleaning_failed",
			wantAction: "retry_model_or_fallback_static",
		},
		{
			name:       "code but not test",
			output:     `{"code":"package calc\nfunc Add(a, b int) int { return a + b }\n"}`,
			wantKind:   "llm_output_validation_failed",
			wantAction: "adjust_prompt_or_fallback_static",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			source := writeTestFile(t, dir, "calc.go", "package calc\nfunc Add(a, b int) int { return a + b }\n")
			providerPath := writeFakeLLMProviderOutput(t, tt.output)
			t.Setenv(generator.EnvLLMProviderCommand, providerPath)

			result, structured, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
				FilePath: source,
				Provider: "llm",
			})
			if err != nil {
				t.Fatalf("HandleGenerateTests returned protocol error: %v", err)
			}
			assertGenerateTestsProviderError(t, result, structured, tt.wantKind, tt.wantAction)
		})
	}
}

func TestHandleGenerateTestsProviderErrorFallsBackToStaticAndRunTests(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"vitest run"},"devDependencies":{"vitest":"^1.0.0"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	source := writeTestFile(t, dir, filepath.Join("src", "sum.ts"), "export function add(a: number, b: number): number { return a + b; }\n")

	t.Setenv(generator.EnvLLMProviderCommand, writeFakeLLMProviderOutput(t, "I would test the add function with a simple happy path."))
	llmResult, llmStructured, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:  source,
		Framework: "vitest",
		Provider:  "llm",
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned protocol error: %v", err)
	}
	assertGenerateTestsProviderError(t, llmResult, llmStructured, "llm_output_cleaning_failed", "retry_model_or_fallback_static")
	llmOutput := llmStructured.(types.GenerateTestsOutput)
	if _, err := os.Stat(llmOutput.TestFile); !os.IsNotExist(err) {
		t.Fatalf("llm provider error should not write test file, stat error = %v", err)
	}

	staticResult, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:  source,
		Framework: "vitest",
		Provider:  "static",
	})
	if err != nil {
		t.Fatalf("static fallback HandleGenerateTests returned error: %v", err)
	}
	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, staticResult)), &generated); err != nil {
		t.Fatalf("unmarshal static fallback output: %v", err)
	}
	if generated.Status != "ok" || generated.Provider != "static" {
		t.Fatalf("static fallback output = %+v, want ok static", generated)
	}
	if generated.TestFile != filepath.Join(dir, "src", "sum.test.ts") {
		t.Fatalf("test file = %q, want src/sum.test.ts", generated.TestFile)
	}

	logPath := installFakeNpxContentChecker(t, []string{
		"import { describe, it, expect } from 'vitest';",
		"import { add } from './sum';",
		"expect(result).toBe((1 + 2));",
	}, strings.Join([]string{
		" ✓ src/sum.test.ts (1 test)",
		" Test Files  1 passed (1)",
		"      Tests  1 passed (1)",
	}, "\n"))
	runResult, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:                  generated.TestFile,
		Framework:             "vitest",
		IncludeFixSuggestions: true,
		SourceCode:            source,
		TestCode:              generated.TestFile,
	})
	if err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}
	var parsed types.TestResult
	if err := json.Unmarshal([]byte(resultText(t, runResult)), &parsed); err != nil {
		t.Fatalf("unmarshal run result: %v", err)
	}
	if parsed.Framework != "vitest" || parsed.Status != "pass" || parsed.Passed != 1 || parsed.Failed != 0 {
		t.Fatalf("unexpected run result: %+v", parsed)
	}
	logText := readTextFile(t, logPath)
	wantDir := absCleanPath(t, dir)
	if !strings.Contains(logText, "PWD="+wantDir+"\n") {
		t.Fatalf("fake npx cwd log = %q, want PWD=%s", logText, wantDir)
	}
	if !strings.Contains(logText, "ARGS=vitest run src/sum.test.ts\n") {
		t.Fatalf("fake npx args log = %q, want Vitest args", logText)
	}
}

func TestHandleGenerateTestsStaticGo(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(source, []byte("package calc\nfunc Add(a, b int) int { return a + b }\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{FilePath: source})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	if generated.Status != "ok" || generated.Provider != "static" || generated.GeneratedCases == 0 {
		t.Fatalf("unexpected generated output: %+v", generated)
	}
	if generated.TestFile != filepath.Join(dir, "calc_test.go") {
		t.Fatalf("test file = %q, want %q", generated.TestFile, filepath.Join(dir, "calc_test.go"))
	}
	structured := structuredContentAs[types.GenerateTestsOutput](t, result)
	if structured.Status != generated.Status || structured.TestFile != generated.TestFile || structured.Provider != generated.Provider {
		t.Fatalf("structured content mismatch: %+v vs %+v", structured, generated)
	}
	if _, err := os.Stat(generated.TestFile); err != nil {
		t.Fatalf("expected generated test file: %v", err)
	}
}

func TestHandleGenerateTestsStaticGoAvoidsDuplicateAcrossPackageFiles(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(source, []byte("package calc\nfunc Add(a, b int) int { return a + b }\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	existing := `package calc

import "testing"

func TestAdd(t *testing.T) {}
`
	if err := os.WriteFile(filepath.Join(dir, "existing_test.go"), []byte(existing), 0o644); err != nil {
		t.Fatalf("write existing test: %v", err)
	}

	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{FilePath: source})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	if generated.TestFile != filepath.Join(dir, "calc_test.go") {
		t.Fatalf("test file = %q, want %q", generated.TestFile, filepath.Join(dir, "calc_test.go"))
	}
	if !strings.Contains(generated.Preview, "func TestAddTestLoop(t *testing.T)") {
		t.Fatalf("generated preview should avoid existing TestAdd:\n%s", generated.Preview)
	}
	if strings.Contains(generated.Preview, "func TestAdd(t *testing.T)") {
		t.Fatalf("generated preview still contains duplicate TestAdd:\n%s", generated.Preview)
	}
}

func TestHandleGenerateTestsStaticGoTodoSkeletonActionManualReview(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "alias.go")
	src := `package utils

func SliceMapper[T any, U any](src []T, mapper func(T) U) []U {
	dst := make([]U, 0, len(src))
	for _, v := range src {
		dst = append(dst, mapper(v))
	}
	return dst
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{FilePath: source})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	if generated.Action != "manual_review" {
		t.Fatalf("action = %q, want manual_review; preview:\n%s", generated.Action, generated.Preview)
	}
	if !strings.Contains(generated.Preview, `t.Skip("TODO: fill in meaningful test inputs and expected values")`) {
		t.Fatalf("expected TODO skip preview:\n%s", generated.Preview)
	}
}

func TestHandleGenerateTestsUsesJavaScriptFramework(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "calc.js")
	if err := os.WriteFile(source, []byte("function add(a, b) {\n  return a + b;\n}\n\nmodule.exports = { add };\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{FilePath: source, Framework: "mocha"})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	for _, want := range []string{
		"const { expect } = require('chai');",
		"expect(result).to.equal((1 + 2));",
	} {
		if !strings.Contains(generated.Preview, want) {
			t.Fatalf("expected %q in generated preview:\n%s", want, generated.Preview)
		}
	}
	if strings.Contains(generated.Preview, "toBe((1 + 2))") {
		t.Fatalf("expected Mocha/Chai assertions, got:\n%s", generated.Preview)
	}
	if generated.Context == nil || generated.Context.Framework != "mocha" {
		t.Fatalf("context framework = %+v, want mocha", generated.Context)
	}
}

func TestHandleGenerateTestsReturnsJavaScriptPayloadFallbackNotes(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "api.ts")
	src := `import type { ExternalUser } from './types';

export async function loadUser(response: Response): Promise<ExternalUser> {
  return await response.json();
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{FilePath: source, Framework: "vitest"})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	if generated.Context == nil {
		t.Fatal("expected generation context")
	}
	target := findGeneratedTarget(generated.Context.Targets, "loadUser")
	if target == nil {
		t.Fatalf("loadUser target not found: %+v", generated.Context.Targets)
	}
	if target.ReturnTypeExpr != "Promise<ExternalUser>" {
		t.Fatalf("return_type_expr = %q, want Promise<ExternalUser>", target.ReturnTypeExpr)
	}
	assertStringSliceContains(t, target.PayloadNotes, "return annotation ExternalUser is not declared in the same source file; static payload falls back to { ok: true }")
	assertStringSliceContains(t, target.PayloadNotes, "return annotation references imported type ExternalUser from './types'; read candidate source files: types.ts, types.tsx, types.d.ts, types.js, types.jsx, types.mjs, types.cjs, types/index.ts, types/index.tsx, types/index.d.ts, types/index.js, types/index.jsx, types/index.mjs, types/index.cjs")
	for _, want := range []string{
		"const result = await loadUser({ json: async () => ({ ok: true }) });",
		"expect(result).toEqual({ ok: true });",
	} {
		if !strings.Contains(generated.Preview, want) {
			t.Fatalf("generated preview missing %q:\n%s", want, generated.Preview)
		}
	}
}

func TestHandleGenerateTestsReturnsResolvedBarrelPayloadInStructuredContent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
  "scripts": { "test": "vitest run" },
  "devDependencies": { "vitest": "^3.0.0" }
}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	modelDir := filepath.Join(dir, "src", "models")
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatalf("mkdir models: %v", err)
	}
	source := filepath.Join(dir, "src", "api.ts")
	src := `import type { ExternalUser } from './models';

export async function loadUser(response: Response): Promise<ExternalUser> {
  return await response.json();
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, "index.ts"), []byte("export type * from './user';\n"), 0o644); err != nil {
		t.Fatalf("write barrel: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, "user.ts"), []byte(`export interface ExternalUser {
  userId: number;
  email: string;
}
`), 0o644); err != nil {
		t.Fatalf("write user type: %v", err)
	}

	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{FilePath: source})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	structured := structuredContentAs[types.GenerateTestsOutput](t, result)
	if structured.Status != generated.Status || structured.TestFile != generated.TestFile || structured.Provider != generated.Provider {
		t.Fatalf("structured content mismatch: %+v vs %+v", structured, generated)
	}
	for _, want := range []string{
		"import { describe, it, expect } from 'vitest';",
		"import { loadUser } from './api';",
		"json: async () => ({ userId: 1, email: 'user@example.com' })",
		"expect(result).toEqual({ userId: 1, email: 'user@example.com' });",
	} {
		if !strings.Contains(generated.Preview, want) {
			t.Fatalf("generated preview missing %q:\n%s", want, generated.Preview)
		}
	}
	if generated.Context == nil {
		t.Fatal("expected generation context")
	}
	target := findGeneratedTarget(generated.Context.Targets, "loadUser")
	if target == nil {
		t.Fatalf("loadUser target not found: %+v", generated.Context.Targets)
	}
	assertStringSliceContains(t, target.PayloadNotes, "return annotation imported type ExternalUser from './models' resolved from models/user.ts")
	for _, note := range target.PayloadNotes {
		if strings.Contains(note, "read candidate source files") || strings.Contains(note, "falls back to { ok: true }") {
			t.Fatalf("resolved barrel imported type should not emit fallback or candidate note: %+v", target.PayloadNotes)
		}
	}
}

func TestHandleGenerateTestsAutoDetectsJavaScriptFramework(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
  "scripts": { "test": "vitest run" },
  "devDependencies": { "jest": "^29.0.0" }
}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	source := filepath.Join(dir, "calc.ts")
	if err := os.WriteFile(source, []byte("export function add(a: number, b: number) { return a + b; }\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{FilePath: source})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	if generated.Context == nil || generated.Context.Framework != "vitest" {
		t.Fatalf("context framework = %+v, want vitest", generated.Context)
	}
	for _, want := range []string{
		"import { add } from './calc';",
		"expect(result).toBe((1 + 2));",
	} {
		if !strings.Contains(generated.Preview, want) {
			t.Fatalf("expected %q in generated preview:\n%s", want, generated.Preview)
		}
	}
	if strings.Contains(generated.Preview, "require('chai')") || strings.Contains(generated.Preview, "to.equal((1 + 2))") {
		t.Fatalf("expected Vitest matcher style, got:\n%s", generated.Preview)
	}
}

func TestHandleGenerateTestsOutputRunsWithDetectedJavaScriptFramework(t *testing.T) {
	tests := []struct {
		name        string
		packageJSON string
		sourceRel   string
		source      string
		wantTestRel string
		wantPreview []string
		wantArgs    string
		output      string
	}{
		{
			name:        "vitest keeps generated test beside source",
			packageJSON: `{"scripts":{"test":"vitest run"}}`,
			sourceRel:   filepath.Join("src", "sum.ts"),
			source:      "export function add(a: number, b: number) { return a + b; }\n",
			wantTestRel: filepath.Join("src", "sum.test.ts"),
			wantPreview: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { add } from './sum';",
				"expect(result).toBe((1 + 2));",
			},
			wantArgs: "vitest run src/sum.test.ts",
			output: strings.Join([]string{
				" ✓ src/sum.test.ts (1 test)",
				" Test Files  1 passed (1)",
				"      Tests  1 passed (1)",
			}, "\n"),
		},
		{
			name:        "mocha keeps generated test beside source",
			packageJSON: `{"scripts":{"test":"mocha --reporter spec"}}`,
			sourceRel:   filepath.Join("lib", "calc.js"),
			source:      "function add(a, b) {\n  return a + b;\n}\n\nmodule.exports = { add };\n",
			wantTestRel: filepath.Join("lib", "calc.test.js"),
			wantPreview: []string{
				"const { expect } = require('chai');",
				"const { add } = require('./calc');",
				"expect(result).to.equal((1 + 2));",
			},
			wantArgs: "mocha --reporter spec lib/calc.test.js",
			output: strings.Join([]string{
				"  add",
				"    ✓ should return expected result for normal input",
				"",
				"  1 passing (8ms)",
			}, "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(tt.packageJSON+"\n"), 0o644); err != nil {
				t.Fatalf("write package.json: %v", err)
			}
			source := filepath.Join(dir, tt.sourceRel)
			if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
				t.Fatalf("create source dir: %v", err)
			}
			if err := os.WriteFile(source, []byte(tt.source), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			generateResult, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{FilePath: source})
			if err != nil {
				t.Fatalf("HandleGenerateTests returned error: %v", err)
			}
			var generated types.GenerateTestsOutput
			if err := json.Unmarshal([]byte(resultText(t, generateResult)), &generated); err != nil {
				t.Fatalf("unmarshal generated output: %v", err)
			}
			wantTestFile := filepath.Join(dir, tt.wantTestRel)
			if generated.TestFile != wantTestFile {
				t.Fatalf("test file = %q, want %q", generated.TestFile, wantTestFile)
			}
			for _, want := range tt.wantPreview {
				if !strings.Contains(generated.Preview, want) {
					t.Fatalf("expected %q in generated preview:\n%s", want, generated.Preview)
				}
			}
			if _, err := os.Stat(generated.TestFile); err != nil {
				t.Fatalf("expected generated test file: %v", err)
			}

			logPath := installFakeNpxRecorder(t, tt.output)
			if _, _, err := HandleRunTests(context.Background(), nil, runTestsInput{Path: generated.TestFile}); err != nil {
				t.Fatalf("HandleRunTests returned error: %v", err)
			}
			logText := readTextFile(t, logPath)
			wantDir := absCleanPath(t, dir)
			if !strings.Contains(logText, "PWD="+wantDir+"\n") {
				t.Fatalf("fake npx cwd log = %q, want PWD=%s", logText, wantDir)
			}
			if !strings.Contains(logText, "ARGS="+tt.wantArgs+"\n") {
				t.Fatalf("fake npx args log = %q, want ARGS=%s", logText, tt.wantArgs)
			}
		})
	}
}

func TestHandleGenerateTestsComplexVitestOutputIsRunnerChecked(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"vitest run"},"devDependencies":{"vitest":"^1.0.0"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	source := filepath.Join(dir, "src", "api.ts")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(source, []byte(`interface User {
  userId: number
  email: string
  status: 'active' | 'disabled'
  manager?: User | null
}

type Meta = {
  total: number
  nextUrl?: string | null
}

type AuditFields = {
  traceId: string
  page: number
}

type ApiResponse = {
  data: User
  meta: Meta
  debug: string
}

type Directory = Readonly<Record<'primary' | 'secondary', ApiResponse['data'] & AuditFields>>
type DirectoryEnvelope = Omit<{ directory: Directory; meta: ApiResponse['meta']; debug: string }, 'debug'>
type DirectorySummary = Pick<DirectoryEnvelope, 'directory' | 'meta'>
type DirectoryBundle = {
  reports: User[]
  pair: readonly [user: User, meta?: Meta]
  directory: Record<string, Pick<User, 'userId' | 'email'>>
  summary: DirectorySummary
}

export async function loadDirectoryBundleClient(api: { fetch(path: string): Promise<DirectoryBundle> }): Promise<DirectoryBundle> {
  return await api.fetch('/directory/bundle')
}
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	generateResult, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{FilePath: source})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}
	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, generateResult)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	if generated.TestFile != filepath.Join(dir, "src", "api.test.ts") {
		t.Fatalf("test file = %q", generated.TestFile)
	}

	logPath := installFakeNpxContentChecker(t, []string{
		"import { describe, it, expect } from 'vitest';",
		"import { loadDirectoryBundleClient } from './api';",
		"return { reports: [{ userId: 1, email: 'user@example.com', status: 'active', manager: {} }], pair: [{ userId: 1, email: 'user@example.com', status: 'active', manager: {} }, { total: 1, nextUrl: 'https://example.com' }], directory: { key: { userId: 1, email: 'user@example.com' } }, summary: { directory: { primary: { userId: 1, email: 'user@example.com', status: 'active', manager: {}, traceId: 'id-1', page: 1 }, secondary: { userId: 1, email: 'user@example.com', status: 'active', manager: {}, traceId: 'id-1', page: 1 } }, meta: { total: 1, nextUrl: 'https://example.com' } } };",
		"const result = await loadDirectoryBundleClient(api);",
		"expect(result).toEqual({ reports: [{ userId: 1, email: 'user@example.com', status: 'active', manager: {} }], pair: [{ userId: 1, email: 'user@example.com', status: 'active', manager: {} }, { total: 1, nextUrl: 'https://example.com' }], directory: { key: { userId: 1, email: 'user@example.com' } }, summary: { directory: { primary: { userId: 1, email: 'user@example.com', status: 'active', manager: {}, traceId: 'id-1', page: 1 }, secondary: { userId: 1, email: 'user@example.com', status: 'active', manager: {}, traceId: 'id-1', page: 1 } }, meta: { total: 1, nextUrl: 'https://example.com' } } });",
		"expect(api.fetchCalls).toEqual([['/directory/bundle']]);",
	}, strings.Join([]string{
		" ✓ src/api.test.ts (1 test)",
		" Test Files  1 passed (1)",
		"      Tests  1 passed (1)",
	}, "\n"))
	runResult, _, err := HandleRunTests(context.Background(), nil, runTestsInput{Path: generated.TestFile})
	if err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}
	var parsed types.TestResult
	if err := json.Unmarshal([]byte(resultText(t, runResult)), &parsed); err != nil {
		t.Fatalf("unmarshal run result: %v", err)
	}
	if parsed.Framework != "vitest" || parsed.Status != "pass" || parsed.Passed != 1 || parsed.Failed != 0 {
		t.Fatalf("unexpected run result: %+v", parsed)
	}
	logText := readTextFile(t, logPath)
	wantDir := absCleanPath(t, dir)
	if !strings.Contains(logText, "PWD="+wantDir+"\n") {
		t.Fatalf("fake npx cwd log = %q, want PWD=%s", logText, wantDir)
	}
	if !strings.Contains(logText, "ARGS=vitest run src/api.test.ts\n") {
		t.Fatalf("fake npx args log = %q, want Vitest args", logText)
	}
}

func TestHandleGenerateTestsLLMProviderOutputFeedsRunTestsRepairLoop(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"vitest run"},"devDependencies":{"vitest":"^1.0.0"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	source := writeTestFile(t, dir, filepath.Join("src", "sum.ts"), strings.Join([]string{
		"export function add(a: number, b: number): number {",
		"  return a + b",
		"}",
	}, "\n")+"\n")

	providerPath := filepath.Join(t.TempDir(), "provider")
	providerScript := `#!/usr/bin/env sh
set -eu
cat >/dev/null
cat <<'EOF'
{"code":"import { describe, it, expect } from 'vitest';\nimport { add } from './sum';\n\ndescribe('sum', () => {\n  it('adds values', () => {\n    const result = add(1, 2);\n\n    expect(result).toBe(3);\n  });\n});\n"}
EOF
`
	if err := os.WriteFile(providerPath, []byte(providerScript), 0o755); err != nil {
		t.Fatalf("write fake llm provider: %v", err)
	}
	t.Setenv(generator.EnvLLMProviderCommand, providerPath)

	generateResult, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:  source,
		Framework: "vitest",
		Provider:  "llm",
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}
	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, generateResult)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	if generated.Provider != "llm-command" {
		t.Fatalf("provider = %q, want llm-command", generated.Provider)
	}
	if generated.TestFile != filepath.Join(dir, "src", "sum.test.ts") {
		t.Fatalf("test file = %q", generated.TestFile)
	}
	generatedCode := readTextFile(t, generated.TestFile)
	for _, want := range []string{
		"import { describe, it, expect } from 'vitest';",
		"expect(result).toBe(3);",
	} {
		if !strings.Contains(generatedCode, want) {
			t.Fatalf("generated LLM provider output missing %q:\n%s", want, generatedCode)
		}
	}

	logPath := installFakeNpxRecorder(t, vitestFailureOutput())
	runResult, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:                  generated.TestFile,
		Framework:             "vitest",
		IncludeFixSuggestions: true,
		SourceCode:            source,
		TestCode:              generated.TestFile,
	})
	if err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}
	var parsed types.TestResult
	if err := json.Unmarshal([]byte(resultText(t, runResult)), &parsed); err != nil {
		t.Fatalf("unmarshal run result: %v", err)
	}
	if parsed.Framework != "vitest" || parsed.Status != "fail" || len(parsed.Failures) != 1 {
		t.Fatalf("unexpected run result: %+v", parsed)
	}
	if len(parsed.FixSuggestions) != 1 {
		t.Fatalf("fix suggestions len = %d, want 1: %+v", len(parsed.FixSuggestions), parsed.FixSuggestions)
	}
	suggestion := parsed.FixSuggestions[0]
	if suggestion.Category != "expectation_mismatch" || suggestion.RepairTask == nil {
		t.Fatalf("unexpected fix suggestion: %+v", suggestion)
	}
	if suggestion.RepairTask.ContextSnippet != "    expect(result).toBe(3);" {
		t.Fatalf("context snippet = %q", suggestion.RepairTask.ContextSnippet)
	}
	wantCommand := "npx vitest run " + filepath.ToSlash(generated.TestFile)
	if !equalStrings(suggestion.RepairTask.SuggestedCommands, []string{wantCommand}) {
		t.Fatalf("suggested commands = %+v, want %q", suggestion.RepairTask.SuggestedCommands, wantCommand)
	}
	logText := readTextFile(t, logPath)
	wantDir := absCleanPath(t, dir)
	if !strings.Contains(logText, "PWD="+wantDir+"\n") {
		t.Fatalf("fake npx cwd log = %q, want PWD=%s", logText, wantDir)
	}
	if !strings.Contains(logText, "ARGS=vitest run src/sum.test.ts\n") {
		t.Fatalf("fake npx args log = %q, want Vitest args", logText)
	}
}

func TestHandleGenerateCoverageTaskComplexVitestOutputIsRunnerChecked(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"vitest run"},"devDependencies":{"vitest":"^1.0.0"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	source := filepath.Join(dir, "src", "api.ts")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(source, []byte(complexDirectoryBundleSource()), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	task := &types.CoverageTestTask{
		ID:              "vitest-directory-bundle",
		Framework:       "vitest",
		File:            source,
		Target:          "loadDirectoryBundleClient",
		Kind:            "function",
		LineRange:       "31-33",
		GapType:         "return_path",
		Goal:            "cover directory bundle client payload",
		TestFile:        filepath.Join(dir, "src", "api.test.ts"),
		TestName:        "covers directory bundle client payload",
		SuggestedInputs: []string{"调用注入式 `api.fetch('/directory/bundle')`"},
		AssertionFocus:  []string{"断言复杂 DirectoryBundle payload 和 fetch 调用参数"},
		Confidence:      0.9,
	}
	generateResult, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     source,
		CoverageTask: task,
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}
	generated := assertGeneratedCoverageTaskOutput(t, generateResult, task, "it('covers directory bundle client payload'")

	logPath := installFakeNpxContentChecker(t, []string{
		"import { describe, it, expect } from 'vitest';",
		"import { loadDirectoryBundleClient } from './api';",
		"coverage task: vitest-directory-bundle | lines 31-33 | 断言复杂 DirectoryBundle payload 和 fetch 调用参数 | 调用注入式 `api.fetch('/directory/bundle')`",
		"return { reports: [{ userId: 1, email: 'user@example.com', status: 'active', manager: {} }], pair: [{ userId: 1, email: 'user@example.com', status: 'active', manager: {} }, { total: 1, nextUrl: 'https://example.com' }], directory: { key: { userId: 1, email: 'user@example.com' } }, summary: { directory: { primary: { userId: 1, email: 'user@example.com', status: 'active', manager: {}, traceId: 'id-1', page: 1 }, secondary: { userId: 1, email: 'user@example.com', status: 'active', manager: {}, traceId: 'id-1', page: 1 } }, meta: { total: 1, nextUrl: 'https://example.com' } } };",
		"const result = await loadDirectoryBundleClient(api);",
		"expect(result).toEqual({ reports: [{ userId: 1, email: 'user@example.com', status: 'active', manager: {} }], pair: [{ userId: 1, email: 'user@example.com', status: 'active', manager: {} }, { total: 1, nextUrl: 'https://example.com' }], directory: { key: { userId: 1, email: 'user@example.com' } }, summary: { directory: { primary: { userId: 1, email: 'user@example.com', status: 'active', manager: {}, traceId: 'id-1', page: 1 }, secondary: { userId: 1, email: 'user@example.com', status: 'active', manager: {}, traceId: 'id-1', page: 1 } }, meta: { total: 1, nextUrl: 'https://example.com' } } });",
		"expect(api.fetchCalls).toEqual([['/directory/bundle']]);",
	}, strings.Join([]string{
		" ✓ src/api.test.ts (1 test)",
		" Test Files  1 passed (1)",
		"      Tests  1 passed (1)",
	}, "\n"))
	runResult, _, err := HandleRunTests(context.Background(), nil, runTestsInput{Path: generated.TestFile})
	if err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}
	var parsed types.TestResult
	if err := json.Unmarshal([]byte(resultText(t, runResult)), &parsed); err != nil {
		t.Fatalf("unmarshal run result: %v", err)
	}
	if parsed.Framework != "vitest" || parsed.Status != "pass" || parsed.Passed != 1 || parsed.Failed != 0 {
		t.Fatalf("unexpected run result: %+v", parsed)
	}
	logText := readTextFile(t, logPath)
	wantDir := absCleanPath(t, dir)
	if !strings.Contains(logText, "PWD="+wantDir+"\n") {
		t.Fatalf("fake npx cwd log = %q, want PWD=%s", logText, wantDir)
	}
	if !strings.Contains(logText, "ARGS=vitest run src/api.test.ts\n") {
		t.Fatalf("fake npx args log = %q, want Vitest args", logText)
	}
}

func complexDirectoryBundleSource() string {
	return `interface User {
  userId: number
  email: string
  status: 'active' | 'disabled'
  manager?: User | null
}

type Meta = {
  total: number
  nextUrl?: string | null
}

type AuditFields = {
  traceId: string
  page: number
}

type ApiResponse = {
  data: User
  meta: Meta
  debug: string
}

type Directory = Readonly<Record<'primary' | 'secondary', ApiResponse['data'] & AuditFields>>
type DirectoryEnvelope = Omit<{ directory: Directory; meta: ApiResponse['meta']; debug: string }, 'debug'>
type DirectorySummary = Pick<DirectoryEnvelope, 'directory' | 'meta'>
type DirectoryBundle = {
  reports: User[]
  pair: readonly [user: User, meta?: Meta]
  directory: Record<string, Pick<User, 'userId' | 'email'>>
  summary: DirectorySummary
}

export async function loadDirectoryBundleClient(api: { fetch(path: string): Promise<DirectoryBundle> }): Promise<DirectoryBundle> {
  return await api.fetch('/directory/bundle')
}
`
}

func TestHandleGenerateCoverageTaskVitestESMOutputRunsWithDetectedFramework(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"vitest run"},"devDependencies":{"vitest":"^1.0.0"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	source := filepath.Join(dir, "src", "sum.ts")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(source, []byte(`export function add(a: number, b: number): number {
  if (a === 0) return b
  return a + b
}
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	task := &types.CoverageTestTask{
		ID:              "vitest-sum-branch",
		Framework:       "vitest",
		File:            source,
		Target:          "add",
		Kind:            "function",
		LineRange:       "2-2",
		GapType:         "branch",
		Goal:            "cover add zero branch",
		TestFile:        filepath.Join(dir, "src", "sum.test.ts"),
		TestName:        "covers add zero branch",
		SuggestedInputs: []string{"构造满足条件 `a === 0` 的输入"},
		AssertionFocus:  []string{"assert zero branch"},
		Confidence:      0.9,
	}

	generateResult, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     source,
		CoverageTask: task,
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}
	generated := assertGeneratedCoverageTaskOutput(t, generateResult, task, "it('covers add zero branch'")
	for _, want := range []string{
		"import { describe, it, expect } from 'vitest';",
		"import { add } from './sum';",
		"expect(result).toBe((2));",
	} {
		if !strings.Contains(generated.Preview, want) {
			t.Fatalf("generated preview missing %q:\n%s", want, generated.Preview)
		}
	}

	logPath := installFakeNpxRecorder(t, strings.Join([]string{
		" ✓ src/sum.test.ts (1 test)",
		" Test Files  1 passed (1)",
		"      Tests  1 passed (1)",
	}, "\n"))
	if _, _, err := HandleRunTests(context.Background(), nil, runTestsInput{Path: generated.TestFile}); err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}
	logText := readTextFile(t, logPath)
	wantDir := absCleanPath(t, dir)
	if !strings.Contains(logText, "PWD="+wantDir+"\n") {
		t.Fatalf("fake npx cwd log = %q, want PWD=%s", logText, wantDir)
	}
	if !strings.Contains(logText, "ARGS=vitest run src/sum.test.ts\n") {
		t.Fatalf("fake npx args log = %q, want Vitest args", logText)
	}
}

func assertGeneratedCoverageTaskOutput(t *testing.T, result *mcp.CallToolResult, task *types.CoverageTestTask, wantSnippet string) types.GenerateTestsOutput {
	t.Helper()

	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	if generated.TestFile != task.TestFile {
		t.Fatalf("test file = %q, want %q", generated.TestFile, task.TestFile)
	}
	if generated.CoverageTask == nil || generated.Context == nil || generated.Context.CoverageTask == nil {
		t.Fatalf("expected coverage task in output and context: %+v", generated)
	}
	if generated.CoverageTask.TestFile != task.TestFile || generated.Context.CoverageTask.TestFile != task.TestFile {
		t.Fatalf("coverage task test file was not preserved: %+v", generated)
	}
	content, err := os.ReadFile(task.TestFile)
	if err != nil {
		t.Fatalf("expected generated coverage task test file: %v", err)
	}
	if !strings.Contains(string(content), wantSnippet) {
		t.Fatalf("generated test file missing %q:\n%s", wantSnippet, content)
	}
	if !strings.Contains(generated.Preview, wantSnippet) {
		t.Fatalf("generated preview missing %q:\n%s", wantSnippet, generated.Preview)
	}
	return generated
}

func installFakeNpxContentChecker(t *testing.T, requiredSnippets []string, output string) string {
	t.Helper()
	fakeBin := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "npx.log")
	patternPath := filepath.Join(t.TempDir(), "required-patterns.txt")
	if err := os.WriteFile(patternPath, []byte(strings.Join(requiredSnippets, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write required patterns: %v", err)
	}
	script := "#!/usr/bin/env sh\n" +
		"{\n" +
		"  printf 'PWD=%s\\n' \"$PWD\"\n" +
		"  printf 'ARGS=%s\\n' \"$*\"\n" +
		"} > '" + logPath + "'\n" +
		"test_file=''\n" +
		"for arg in \"$@\"; do\n" +
		"  case \"$arg\" in\n" +
		"    *.test.ts|*.test.js) test_file=\"$arg\" ;;\n" +
		"  esac\n" +
		"done\n" +
		"if [ -z \"$test_file\" ]; then echo 'missing test file argument' >&2; exit 2; fi\n" +
		"if [ ! -f \"$test_file\" ]; then echo \"missing generated test file: $test_file\" >&2; exit 3; fi\n" +
		"while IFS= read -r pattern; do\n" +
		"  [ -z \"$pattern\" ] && continue\n" +
		"  if ! grep -F \"$pattern\" \"$test_file\" >/dev/null; then\n" +
		"    echo \"missing generated test snippet: $pattern\" >&2\n" +
		"    exit 4\n" +
		"  fi\n" +
		"done < '" + patternPath + "'\n" +
		"cat <<'NPX_OUTPUT'\n" + output + "\nNPX_OUTPUT\n" +
		"exit 0\n"
	path := filepath.Join(fakeBin, "npx")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake npx content checker: %v", err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

func TestHandleGenerateTestsUsesCoverageTaskTestFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(source, []byte("package calc\nfunc Add(a, b int) int { return a + b }\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	testFile := filepath.Join(dir, "generated", "calc_task_test.go")
	task := &types.CoverageTestTask{
		ID:             "go-calc-add",
		Framework:      "go-test",
		File:           source,
		Target:         "Add",
		Kind:           "function",
		LineRange:      "2-2",
		Goal:           "cover Add",
		TestFile:       testFile,
		TestName:       "TestAddCoverageTask",
		AssertionFocus: []string{"assert Add result"},
		Confidence:     0.9,
	}

	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     source,
		CoverageTask: task,
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	assertGeneratedCoverageTaskOutput(t, result, task, "TestAddCoverageTask")
}

func TestHandleGenerateTestsAppendsGoCoverageTaskToExistingTestFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "calc.go")
	src := `package calc

func Add(a, b int) int { return a + b }

func Sub(a, b int) int { return a - b }
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	testFile := filepath.Join(dir, "calc_test.go")
	addTask := &types.CoverageTestTask{
		ID:             "go-calc-add",
		Framework:      "go-test",
		File:           source,
		Target:         "Add",
		Kind:           "function",
		TestFile:       testFile,
		TestName:       "TestAddCoverageTask",
		AssertionFocus: []string{"assert Add result"},
	}
	subTask := &types.CoverageTestTask{
		ID:             "go-calc-sub",
		Framework:      "go-test",
		File:           source,
		Target:         "Sub",
		Kind:           "function",
		TestFile:       testFile,
		TestName:       "TestSubCoverageTask",
		AssertionFocus: []string{"assert Sub result"},
	}

	if _, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     source,
		CoverageTask: addTask,
	}); err != nil {
		t.Fatalf("first HandleGenerateTests returned error: %v", err)
	}
	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     source,
		CoverageTask: subTask,
	})
	if err != nil {
		t.Fatalf("second HandleGenerateTests returned error: %v", err)
	}

	generated := assertGeneratedCoverageTaskOutput(t, result, subTask, "TestSubCoverageTask")
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("read merged test file: %v", err)
	}
	text := string(content)
	for _, want := range []string{"func TestAddCoverageTask", "func TestSubCoverageTask"} {
		if !strings.Contains(text, want) {
			t.Fatalf("merged Go test file missing %q:\n%s", want, text)
		}
		if !strings.Contains(generated.Preview, want) {
			t.Fatalf("preview missing %q:\n%s", want, generated.Preview)
		}
	}
	if count := strings.Count(text, "\"testing\""); count != 1 {
		t.Fatalf("testing import count = %d, want 1:\n%s", count, text)
	}
}

func TestWriteGeneratedTestFileCleansUnusedGoImportsAfterMerge(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "calc_test.go")
	existing := `package calc

import (
	"io"
	"testing"
)

func TestExisting(t *testing.T) {}
`
	if err := os.WriteFile(testFile, []byte(existing), 0o644); err != nil {
		t.Fatalf("write existing test file: %v", err)
	}
	generated := `package calc

import "testing"

func TestAdd(t *testing.T) {}
`

	merged, err := writeGeneratedTestFile(testFile, generated, ".go")
	if err != nil {
		t.Fatalf("writeGeneratedTestFile returned error: %v", err)
	}
	if strings.Contains(merged, "\"io\"") {
		t.Fatalf("unused io import was not removed:\n%s", merged)
	}
	for _, want := range []string{"func TestExisting(t *testing.T)", "func TestAdd(t *testing.T)", "\"testing\""} {
		if !strings.Contains(merged, want) {
			t.Fatalf("merged Go test file missing %q:\n%s", want, merged)
		}
	}
}

func TestWriteGeneratedTestFileCleansUnusedGoImportsForNewFile(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "calc_test.go")
	generated := `package calc

import (
	"io"
	"testing"
)

func TestAdd(t *testing.T) {}
`

	written, err := writeGeneratedTestFile(testFile, generated, ".go")
	if err != nil {
		t.Fatalf("writeGeneratedTestFile returned error: %v", err)
	}
	if strings.Contains(written, "\"io\"") {
		t.Fatalf("unused io import was not removed:\n%s", written)
	}
	if !strings.Contains(written, "func TestAdd(t *testing.T)") || !strings.Contains(written, "\"testing\"") {
		t.Fatalf("written Go test file missing expected content:\n%s", written)
	}
}

func TestHandleGenerateTestsRenamesDuplicateGoCoverageTaskFunction(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(source, []byte("package calc\nfunc Add(a, b int) int { return a + b }\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	testFile := filepath.Join(dir, "calc_test.go")
	task := &types.CoverageTestTask{
		ID:        "go-calc-add",
		Framework: "go-test",
		File:      source,
		Target:    "Add",
		Kind:      "function",
		LineRange: "2-2",
		TestFile:  testFile,
		TestName:  "TestAddCoverageTask",
	}

	if _, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     source,
		CoverageTask: task,
	}); err != nil {
		t.Fatalf("first HandleGenerateTests returned error: %v", err)
	}
	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     source,
		CoverageTask: task,
	})
	if err != nil {
		t.Fatalf("second HandleGenerateTests returned error: %v", err)
	}

	generated := assertGeneratedCoverageTaskOutput(t, result, task, "TestAddCoverageTaskCoverage2_2")
	if generated.CoverageTask.TestName != "TestAddCoverageTaskCoverage2_2" {
		t.Fatalf("coverage task test_name = %q, want adjusted suffix", generated.CoverageTask.TestName)
	}
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("read merged test file: %v", err)
	}
	text := string(content)
	for _, want := range []string{"func TestAddCoverageTask", "func TestAddCoverageTaskCoverage2_2"} {
		if !strings.Contains(text, want) {
			t.Fatalf("merged Go test file missing %q:\n%s", want, text)
		}
	}
}

func TestHandleGenerateTestsRenamesDuplicateGoCoverageTaskFunctionAcrossPackageFiles(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(source, []byte("package calc\nfunc Add(a, b int) int { return a + b }\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	existingTestFile := filepath.Join(dir, "existing_test.go")
	existing := `package calc

import "testing"

func TestAddCoverageTask(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("unexpected Add result")
	}
}
`
	if err := os.WriteFile(existingTestFile, []byte(existing), 0o644); err != nil {
		t.Fatalf("write existing test: %v", err)
	}
	testFile := filepath.Join(dir, "calc_task_test.go")
	task := &types.CoverageTestTask{
		ID:        "go-calc-add",
		Framework: "go-test",
		File:      source,
		Target:    "Add",
		Kind:      "function",
		LineRange: "2-2",
		TestFile:  testFile,
		TestName:  "TestAddCoverageTask",
	}

	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     source,
		CoverageTask: task,
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	generated := assertGeneratedCoverageTaskOutput(t, result, task, "TestAddCoverageTaskCoverage2_2")
	if generated.CoverageTask.TestName != "TestAddCoverageTaskCoverage2_2" {
		t.Fatalf("coverage task test_name = %q, want adjusted suffix", generated.CoverageTask.TestName)
	}
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("read generated test file: %v", err)
	}
	if text := string(content); !strings.Contains(text, "func TestAddCoverageTaskCoverage2_2") {
		t.Fatalf("generated Go test file missing adjusted function:\n%s", text)
	}
}

func TestWriteGeneratedTestFileRejectsDuplicateGoTestFunction(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "calc_test.go")
	existing := `package calc

import "testing"

func TestAdd(t *testing.T) {}
`
	if err := os.WriteFile(testFile, []byte(existing), 0o644); err != nil {
		t.Fatalf("write existing test file: %v", err)
	}
	generated := `package calc

import "testing"

func TestAdd(t *testing.T) {}
`

	_, err := writeGeneratedTestFile(testFile, generated, ".go")
	if err == nil {
		t.Fatal("expected duplicate Go test function error")
	}
	if !strings.Contains(err.Error(), "Go 测试函数已存在: TestAdd") {
		t.Fatalf("duplicate error = %v", err)
	}
}

func TestHandleGenerateTestsUsesPythonCoverageTaskTestFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src", "service.py")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(source, []byte("def status(value):\n    if value == \"active\":\n        return \"ok\"\n    return \"idle\"\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := &types.CoverageTestTask{
		ID:              "pytest-service-status",
		Framework:       "pytest",
		File:            source,
		Target:          "status",
		Kind:            "function",
		LineRange:       "2-2",
		GapType:         "branch",
		Goal:            "cover status active branch",
		TestFile:        filepath.Join(dir, "tests", "test_service.py"),
		TestName:        "test_status_active_branch",
		SuggestedInputs: []string{"构造满足条件 `value == \"active\"` 的输入"},
		AssertionFocus:  []string{"assert active branch"},
		Confidence:      0.9,
	}

	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     source,
		CoverageTask: task,
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	assertGeneratedCoverageTaskOutput(t, result, task, "def test_status_active_branch():")
}

func TestHandleGenerateTestsUsesJavaScriptCoverageTaskTestFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src", "sum.js")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(source, []byte("function add(a, b) {\n  if (a === 0) return b;\n  return a + b;\n}\n\nmodule.exports = { add };\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := &types.CoverageTestTask{
		ID:              "jest-add-branch",
		Framework:       "jest",
		File:            source,
		Target:          "add",
		Kind:            "function",
		LineRange:       "2-2",
		GapType:         "branch",
		Goal:            "cover add zero branch",
		TestFile:        filepath.Join(dir, "src", "sum.test.js"),
		TestName:        "should cover add zero branch",
		SuggestedInputs: []string{"构造满足条件 `a === 0` 的输入"},
		AssertionFocus:  []string{"assert zero branch"},
		Confidence:      0.9,
	}

	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     source,
		CoverageTask: task,
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	assertGeneratedCoverageTaskOutput(t, result, task, "it('should cover add zero branch'")
}

func TestHandleGenerateTestsUsesJavaCoverageTaskTestFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src", "main", "java", "Calculator.java")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(source, []byte("public class Calculator {\n    public int add(int a, int b) {\n        return a + b;\n    }\n\n    public int sub(int a, int b) {\n        return a - b;\n    }\n}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := &types.CoverageTestTask{
		ID:              "junit-calculator-add",
		Framework:       "junit",
		File:            source,
		Target:          "Calculator.add",
		Kind:            "method",
		LineRange:       "2-4",
		GapType:         "branch",
		Goal:            "cover Calculator.add branch",
		TestFile:        filepath.Join(dir, "src", "test", "java", "CalculatorTaskTest.java"),
		TestName:        "shouldCoverCalculatorAddGap",
		SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入"},
		AssertionFocus:  []string{"assert add result"},
		Confidence:      0.9,
	}

	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     source,
		CoverageTask: task,
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	assertGeneratedCoverageTaskOutput(t, result, task, "void shouldCoverCalculatorAddGap()")
}

func TestHandleGenerateTestsAvoidsExistingJavaCoverageTaskTestFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src", "main", "java", "org", "example", "Base64.java")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(source, []byte("package org.example;\n\npublic class Base64 {\n    public byte[] encode(byte[] in) {\n        return in;\n    }\n}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	testFile := filepath.Join(dir, "src", "test", "java", "org", "example", "Base64Test.java")
	if err := os.MkdirAll(filepath.Dir(testFile), 0o755); err != nil {
		t.Fatalf("create test dir: %v", err)
	}
	existing := "package org.example;\n\npublic class Base64Test {\n    static final String EXISTING_HELPER = \"keep\";\n}\n"
	if err := os.WriteFile(testFile, []byte(existing), 0o644); err != nil {
		t.Fatalf("write existing test: %v", err)
	}
	task := &types.CoverageTestTask{
		ID:        "junit-base64-encode",
		Framework: "junit",
		File:      source,
		Target:    "Base64.encode",
		Kind:      "method",
		LineRange: "4-4",
		TestFile:  testFile,
		TestName:  "shouldCoverBase64EncodeGap",
	}

	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     source,
		CoverageTask: task,
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	wantTestFile := filepath.Join(dir, "src", "test", "java", "org", "example", "Base64TestLoopTest.java")
	if generated.TestFile != wantTestFile {
		t.Fatalf("test file = %q, want %q", generated.TestFile, wantTestFile)
	}
	if generated.CoverageTask == nil || generated.CoverageTask.TestFile != wantTestFile ||
		generated.Context == nil || generated.Context.CoverageTask == nil || generated.Context.CoverageTask.TestFile != wantTestFile {
		t.Fatalf("coverage task test file was not rewritten consistently: %+v", generated)
	}
	content := readTextFile(t, wantTestFile)
	if !strings.Contains(content, "public class Base64TestLoopTest") || !strings.Contains(content, "void shouldCoverBase64EncodeGap()") {
		t.Fatalf("generated collision-free Java test missing expected content:\n%s", content)
	}
	if got := readTextFile(t, testFile); got != existing {
		t.Fatalf("existing Java test file was overwritten:\n%s", got)
	}
}

func TestHandleGenerateTestsUsesRustCoverageTaskTestFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src", "lib.rs")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(source, []byte("pub fn add(a: i32, b: i32) -> i32 {\n    a + b\n}\n\npub fn sub(a: i32, b: i32) -> i32 {\n    a - b\n}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := &types.CoverageTestTask{
		ID:              "cargo-add-gap",
		Framework:       "cargo-test",
		File:            source,
		Target:          "add",
		Kind:            "function",
		LineRange:       "1-3",
		GapType:         "branch",
		Goal:            "cover add branch",
		TestFile:        filepath.Join(dir, "tests", "add_gap_test.rs"),
		TestName:        "test_add_gap",
		SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入"},
		AssertionFocus:  []string{"assert add result"},
		Confidence:      0.9,
	}

	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     source,
		CoverageTask: task,
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	assertGeneratedCoverageTaskOutput(t, result, task, "fn test_add_gap()")
}

func TestHandleGenerateTestsAppendsRustCoverageTaskToSourceFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src", "lib.rs")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	original := "pub fn add(a: i32, b: i32) -> i32 {\n    a + b\n}\n"
	if err := os.WriteFile(source, []byte(original), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := &types.CoverageTestTask{
		ID:              "cargo-add-gap",
		Framework:       "cargo-test",
		File:            source,
		Target:          "add",
		Kind:            "function",
		LineRange:       "1-3",
		GapType:         "branch",
		TestFile:        source,
		TestName:        "test_add_gap",
		SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入"},
	}

	result, _, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{
		FilePath:     source,
		CoverageTask: task,
	})
	if err != nil {
		t.Fatalf("HandleGenerateTests returned error: %v", err)
	}

	generated := assertGeneratedCoverageTaskOutput(t, result, task, "fn test_add_gap()")
	content := readTextFile(t, source)
	if !strings.Contains(content, original) {
		t.Fatalf("Rust source was not preserved:\n%s", content)
	}
	if !strings.Contains(content, "#[cfg(test)]") || !strings.Contains(content, "use super::*;") {
		t.Fatalf("Rust generated inline test module missing:\n%s", content)
	}
	if generated.GeneratedCases == 0 {
		t.Fatalf("generated cases should count appended Rust tests: %+v", generated)
	}
}

func TestCountGeneratedCasesByLanguage(t *testing.T) {
	tests := []struct {
		name string
		ext  string
		code string
		want int
	}{
		{name: "go", ext: ".go", code: "func TestAdd(t *testing.T) {}\nfunc helper() {}", want: 1},
		{name: "jest single", ext: ".js", code: "it('adds', () => {})\nit(\"subs\", () => {})", want: 2},
		{name: "python", ext: ".py", code: "def test_add():\n    pass\n    def test_nested():\n        pass", want: 2},
		{name: "rust", ext: ".rs", code: "#[test]\nfn adds() {}\n#[test]\nfn subs() {}", want: 2},
		{name: "java", ext: ".java", code: "@Test\nvoid adds() {}\n@Test\nvoid subs() {}", want: 2},
		{name: "unknown", ext: ".txt", code: "func TestAdd() {}", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countGeneratedCases(tt.code, tt.ext); got != tt.want {
				t.Fatalf("countGeneratedCases = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGeneratedTestsAction(t *testing.T) {
	tests := []struct {
		name string
		ext  string
		code string
		want string
	}{
		{name: "go ready", ext: ".go", code: "func TestAdd(t *testing.T) {}", want: "ready"},
		{name: "go todo skip", ext: ".go", code: `func TestAdd(t *testing.T) { t.Skip("TODO: fill in meaningful test inputs and expected values") }`, want: "manual_review"},
		{name: "js manual review", ext: ".ts", code: "it.skip('manual', () => {})", want: "manual_review"},
		{name: "python manual review", ext: ".py", code: "__import__('pytest').skip('manual_review_internal: helper')", want: "manual_review"},
		{name: "java manual review", ext: ".java", code: "org.junit.jupiter.api.Assumptions.assumeTrue(false, \"manual_review_unreachable: line\")", want: "manual_review"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generatedTestsAction(tt.code, tt.ext); got != tt.want {
				t.Fatalf("generatedTestsAction = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleFixSuggestionsReturnsEmptyForNoFailures(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(source, []byte("package calc\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	result, _, err := HandleFixSuggestions(context.Background(), nil, fixSuggestionsInput{
		Failures:   "[]",
		SourceCode: source,
	})
	if err != nil {
		t.Fatalf("HandleFixSuggestions returned error: %v", err)
	}

	var suggestions []types.FixSuggestion
	if err := json.Unmarshal([]byte(resultText(t, result)), &suggestions); err != nil {
		t.Fatalf("unmarshal suggestions: %v", err)
	}
	if len(suggestions) != 0 {
		t.Fatalf("suggestions len = %d, want 0", len(suggestions))
	}
	structured := structuredContentAs[[]types.FixSuggestion](t, result)
	if len(structured) != 0 {
		t.Fatalf("structured suggestions len = %d, want 0", len(structured))
	}
}

func TestHandleFixSuggestionsValidatesInput(t *testing.T) {
	if _, _, err := HandleFixSuggestions(context.Background(), nil, fixSuggestionsInput{}); err == nil {
		t.Fatal("expected missing failures/source error")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(source, []byte("package calc\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, _, err := HandleFixSuggestions(context.Background(), nil, fixSuggestionsInput{
		Failures:   "{",
		SourceCode: source,
	}); err == nil {
		t.Fatal("expected invalid failures json error")
	}

	missing := filepath.Join(dir, "missing.go")
	failuresJSON, err := json.Marshal([]types.TestFailure{{TestName: "Test", Error: "panic: runtime error"}})
	if err != nil {
		t.Fatalf("marshal failures: %v", err)
	}
	if _, _, err := HandleFixSuggestions(context.Background(), nil, fixSuggestionsInput{
		Failures:   string(failuresJSON),
		SourceCode: missing,
	}); err == nil {
		t.Fatal("expected missing source error")
	}
}

func TestHandleFixSuggestionsGotWant(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(source, []byte("package calc\nfunc Add(a, b int) int { return a + b }\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	failuresJSON, err := json.Marshal([]types.TestFailure{{
		TestName: "TestAdd",
		File:     source,
		Line:     2,
		Error:    "ret0 got 2, want 3",
	}})
	if err != nil {
		t.Fatalf("marshal failures: %v", err)
	}

	result, _, err := HandleFixSuggestions(context.Background(), nil, fixSuggestionsInput{
		Failures:   string(failuresJSON),
		SourceCode: source,
	})
	if err != nil {
		t.Fatalf("HandleFixSuggestions returned error: %v", err)
	}

	var suggestions []types.FixSuggestion
	if err := json.Unmarshal([]byte(resultText(t, result)), &suggestions); err != nil {
		t.Fatalf("unmarshal suggestions: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("suggestions len = %d, want 1", len(suggestions))
	}
	if suggestions[0].Confidence != 0.8 ||
		suggestions[0].Category != "expectation_mismatch" ||
		suggestions[0].ContextFile != source ||
		suggestions[0].ContextLine != 2 ||
		!strings.Contains(suggestions[0].SuggestedFix, "实际值: 2") ||
		!strings.Contains(suggestions[0].SuggestedFix, "源码附近行") ||
		!strings.Contains(suggestions[0].SuggestedFix, "func Add") {
		t.Fatalf("unexpected suggestion: %+v", suggestions[0])
	}
	task := suggestions[0].RepairTask
	if task == nil {
		t.Fatal("expected repair_task")
	}
	if task.ID != "repair-expectation_mismatch-testadd" ||
		task.Category != "expectation_mismatch" ||
		task.TargetFile != source ||
		task.TargetLine != 2 ||
		task.ContextFile != source ||
		task.ContextLine != 2 ||
		task.ContextSnippet != "func Add(a, b int) int { return a + b }" ||
		len(task.EditableFiles) != 1 ||
		task.EditableFiles[0] != source ||
		len(task.SuggestedCommands) != 1 ||
		task.SuggestedCommands[0] != "go test ./..." ||
		!strings.Contains(task.AssertionFocus, "实际值和期望值") {
		t.Fatalf("unexpected repair task: %+v", task)
	}
}

func TestHandleFixSuggestionsUsesTestLineForTestFailure(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(source, []byte("package calc\nfunc Add(a, b int) int { return a + b }\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	testFile := filepath.Join(dir, "calc_test.go")
	if err := os.WriteFile(testFile, []byte("package calc\nif got := Add(1, 1); got != 3 { t.Fatalf(\"got %d, want 3\", got) }\n"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	failuresJSON, err := json.Marshal([]types.TestFailure{{
		TestName: "TestAdd",
		File:     testFile,
		Line:     2,
		Error:    "got 2, want 3",
	}})
	if err != nil {
		t.Fatalf("marshal failures: %v", err)
	}

	result, _, err := HandleFixSuggestions(context.Background(), nil, fixSuggestionsInput{
		Failures:   string(failuresJSON),
		SourceCode: source,
		TestCode:   testFile,
	})
	if err != nil {
		t.Fatalf("HandleFixSuggestions returned error: %v", err)
	}

	var suggestions []types.FixSuggestion
	if err := json.Unmarshal([]byte(resultText(t, result)), &suggestions); err != nil {
		t.Fatalf("unmarshal suggestions: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("suggestions len = %d, want 1", len(suggestions))
	}
	if suggestions[0].Category != "expectation_mismatch" ||
		suggestions[0].ContextFile != testFile ||
		suggestions[0].ContextLine != 2 {
		t.Fatalf("unexpected structured context: %+v", suggestions[0])
	}
	got := suggestions[0].SuggestedFix
	if !strings.Contains(got, "测试附近行") || !strings.Contains(got, "Add(1, 1)") {
		t.Fatalf("suggestion missing test context: %+v", suggestions[0])
	}
	if strings.Contains(got, "源码附近行") || strings.Contains(got, "func Add") {
		t.Fatalf("suggestion should not use source context for test failure: %+v", suggestions[0])
	}
}

func TestHandleFixSuggestionsMatchesRelativeTestFailurePath(t *testing.T) {
	dir := t.TempDir()
	demoDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(demoDir, 0o755); err != nil {
		t.Fatalf("mkdir demo: %v", err)
	}
	source := filepath.Join(demoDir, "calc.go")
	if err := os.WriteFile(source, []byte("package demo\nfunc Add(a, b int) int { return a + b }\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	testFile := filepath.Join(demoDir, "calc_test.go")
	if err := os.WriteFile(testFile, []byte("package demo\nif got := Add(1, 1); got != 3 { t.Fatalf(\"got %d, want 3\", got) }\n"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	failuresJSON, err := json.Marshal([]types.TestFailure{{
		TestName: "TestAdd",
		File:     filepath.Join("demo", "calc_test.go"),
		Line:     2,
		Error:    "got 2, want 3",
	}})
	if err != nil {
		t.Fatalf("marshal failures: %v", err)
	}

	result, _, err := HandleFixSuggestions(context.Background(), nil, fixSuggestionsInput{
		Failures:   string(failuresJSON),
		SourceCode: source,
		TestCode:   testFile,
	})
	if err != nil {
		t.Fatalf("HandleFixSuggestions returned error: %v", err)
	}

	var suggestions []types.FixSuggestion
	if err := json.Unmarshal([]byte(resultText(t, result)), &suggestions); err != nil {
		t.Fatalf("unmarshal suggestions: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("suggestions len = %d, want 1", len(suggestions))
	}
	if suggestions[0].ContextFile != testFile || suggestions[0].ContextLine != 2 {
		t.Fatalf("unexpected relative path context: %+v", suggestions[0])
	}
	if got := suggestions[0].SuggestedFix; !strings.Contains(got, "测试附近行") || !strings.Contains(got, "Add(1, 1)") {
		t.Fatalf("suggestion missing relative test context: %+v", suggestions[0])
	}
}

func TestHandleFixSuggestionsClassifiesCommonFailures(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(source, []byte("package calc\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	failures := []types.TestFailure{
		{TestName: "TestNil", File: source, Line: 2, Error: "panic: runtime error: invalid memory address or nil pointer dereference"},
		{TestName: "TestIndex", File: source, Line: 3, Error: "panic: runtime error: index out of range [2] with length 1"},
		{TestName: "TestDivision", File: source, Line: 4, Error: "panic: runtime error: integer divide by zero"},
		{TestName: "TestUndefined", File: source, Line: 5, Error: "undefined: missingSymbol"},
		{TestName: "TestType", File: source, Line: 6, Error: "cannot use value as string value"},
	}
	failuresJSON, err := json.Marshal(failures)
	if err != nil {
		t.Fatalf("marshal failures: %v", err)
	}

	result, _, err := HandleFixSuggestions(context.Background(), nil, fixSuggestionsInput{
		Failures:   string(failuresJSON),
		SourceCode: source,
	})
	if err != nil {
		t.Fatalf("HandleFixSuggestions returned error: %v", err)
	}

	var suggestions []types.FixSuggestion
	if err := json.Unmarshal([]byte(resultText(t, result)), &suggestions); err != nil {
		t.Fatalf("unmarshal suggestions: %v", err)
	}
	if len(suggestions) != len(failures) {
		t.Fatalf("suggestions len = %d, want %d", len(suggestions), len(failures))
	}
	checks := []struct {
		wantConfidence float64
		wantCategory   string
		wantTexts      []string
	}{
		{0.9, "runtime_panic", []string{"nil"}},
		{0.9, "index_out_of_range", []string{"边界检查", "失败索引: 2", "当前长度: 1"}},
		{0.95, "divide_by_zero", []string{"除零检查", "除数是否为 0"}},
		{0.7, "undefined_symbol", []string{"拼写正确", "符号: missingSymbol"}},
		{0.7, "type_mismatch", []string{"类型转换", "函数签名"}},
	}
	for i, check := range checks {
		if suggestions[i].Confidence != check.wantConfidence {
			t.Fatalf("suggestion %d = %+v, want confidence %.1f", i, suggestions[i], check.wantConfidence)
		}
		if suggestions[i].Category != check.wantCategory {
			t.Fatalf("suggestion %d = %+v, want category %q", i, suggestions[i], check.wantCategory)
		}
		for _, wantText := range check.wantTexts {
			if !strings.Contains(suggestions[i].SuggestedFix, wantText) {
				t.Fatalf("suggestion %d = %+v, want text %q", i, suggestions[i], wantText)
			}
		}
	}
}

func TestHandleFixSuggestionsUsesParsedFrameworkFixtures(t *testing.T) {
	tests := []struct {
		name            string
		framework       string
		fixture         string
		sourceRel       string
		sourceLines     []string
		testRel         string
		testLines       []string
		wantCategory    string
		wantContextFile string
		wantContextLine int
		wantTexts       []string
	}{
		{
			name:      "jest expected received",
			framework: "jest",
			fixture:   "jest_failure.txt",
			sourceRel: "sum.js",
			sourceLines: []string{
				"export function add(a, b) {",
				"  return a + b + 1",
				"}",
			},
			testRel: "sum.test.js",
			testLines: []string{
				"import { add } from './sum'",
				"",
				"test('adds 1 + 2 to equal 3', () => {",
				"  const result = add(1, 2)",
				"  expect(result).toBe(3)",
				"})",
			},
			wantCategory:    "expectation_mismatch",
			wantContextFile: "sum.test.js",
			wantContextLine: 5,
			wantTexts:       []string{"实际值: 4", "期望值: 3", "测试附近行", "expect(result).toBe(3)"},
		},
		{
			name:      "vitest assertion error",
			framework: "vitest",
			fixture:   "vitest_failure.txt",
			sourceRel: filepath.Join("src", "sum.ts"),
			sourceLines: []string{
				"export function add(a: number, b: number): number {",
				"  return a + b + 1",
				"}",
			},
			testRel: filepath.Join("src", "sum.test.ts"),
			testLines: []string{
				"import { add } from './sum'",
				"",
				"describe('sum', () => {",
				"  test('adds values', () => {",
				"    const result = add(1, 2)",
				"    expect(result).toBe(3)",
				"  })",
				"  expect(add(1, 2)).toBe(3)",
			},
			wantCategory:    "expectation_mismatch",
			wantContextFile: filepath.Join("src", "sum.test.ts"),
			wantContextLine: 8,
			wantTexts:       []string{"实际值: 4", "期望值: 3", "测试附近行", "expect(add(1, 2)).toBe(3)"},
		},
		{
			name:      "mocha assertion error",
			framework: "mocha",
			fixture:   "mocha_failure.txt",
			sourceRel: "calc.js",
			sourceLines: []string{
				"exports.divide = function divide(a, b) {",
				"  return a + b",
				"}",
			},
			testRel: filepath.Join("test", "calc.test.js"),
			testLines: []string{
				"const { expect } = require('chai')",
				"const calc = require('../calc')",
				"",
				"describe('calc', () => {",
				"  it('add() should add numbers', () => {})",
				"  it('divide() should handle division by zero', () => {",
				"    const result = calc.divide(8, 2)",
				"    expect(result).to.equal(3)",
				"  })",
				"})",
				"",
				"expect(calc.divide(8, 2)).to.equal(3)",
			},
			wantCategory:    "expectation_mismatch",
			wantContextFile: filepath.Join("test", "calc.test.js"),
			wantContextLine: 12,
			wantTexts:       []string{"实际值: 4", "期望值: 3", "测试附近行", "expect(calc.divide(8, 2)).to.equal(3)"},
		},
		{
			name:      "pytest exception",
			framework: "pytest",
			fixture:   "pytest_failure.txt",
			sourceRel: "calc.py",
			sourceLines: []string{
				"def add(a, b):",
				"    return a + b",
				"",
				"def divide(a, b):",
				"    if b == 0:",
				"        # keep line numbers aligned with pytest fixture",
				"        raise ValueError(\"division by zero\")",
			},
			testRel: "test_calc.py",
			testLines: []string{
				"from calc import divide",
				"",
				"def test_divide():",
				"    divide(1, 0)",
			},
			wantCategory:    "divide_by_zero",
			wantContextFile: "calc.py",
			wantContextLine: 7,
			wantTexts:       []string{"除零检查", "源码附近行", "raise ValueError"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			source := writeTestFile(t, dir, tt.sourceRel, strings.Join(tt.sourceLines, "\n")+"\n")
			testFile := writeTestFile(t, dir, tt.testRel, strings.Join(tt.testLines, "\n")+"\n")
			parsed, err := parseResults(parseResultsInput{
				Output:    readToolParserFixture(t, tt.fixture),
				Framework: tt.framework,
			})
			if err != nil {
				t.Fatalf("parseResults returned error: %v", err)
			}
			if len(parsed.Failures) != 1 {
				t.Fatalf("parsed failures len = %d, want 1: %+v", len(parsed.Failures), parsed.Failures)
			}
			failuresJSON, err := json.Marshal(parsed.Failures)
			if err != nil {
				t.Fatalf("marshal failures: %v", err)
			}

			result, _, err := HandleFixSuggestions(context.Background(), nil, fixSuggestionsInput{
				Failures:   string(failuresJSON),
				SourceCode: source,
				TestCode:   testFile,
			})
			if err != nil {
				t.Fatalf("HandleFixSuggestions returned error: %v", err)
			}

			var suggestions []types.FixSuggestion
			if err := json.Unmarshal([]byte(resultText(t, result)), &suggestions); err != nil {
				t.Fatalf("unmarshal suggestions: %v", err)
			}
			if len(suggestions) != 1 {
				t.Fatalf("suggestions len = %d, want 1", len(suggestions))
			}
			got := suggestions[0]
			if got.Category != tt.wantCategory {
				t.Fatalf("category = %q, want %q: %+v", got.Category, tt.wantCategory, got)
			}
			if !strings.HasSuffix(filepath.ToSlash(got.ContextFile), filepath.ToSlash(tt.wantContextFile)) || got.ContextLine != tt.wantContextLine {
				t.Fatalf("context = %s:%d, want suffix %s:%d", got.ContextFile, got.ContextLine, tt.wantContextFile, tt.wantContextLine)
			}
			for _, wantText := range tt.wantTexts {
				if !strings.Contains(got.SuggestedFix, wantText) {
					t.Fatalf("suggestion missing %q: %+v", wantText, got)
				}
			}
			if got.RepairTask == nil {
				t.Fatal("expected repair_task")
			}
			task := got.RepairTask
			if task.Category != tt.wantCategory ||
				task.ContextFile != got.ContextFile ||
				task.ContextLine != got.ContextLine ||
				task.TargetFile != got.ContextFile ||
				task.TargetLine != got.ContextLine ||
				strings.TrimSpace(task.ContextSnippet) == "" ||
				len(task.EditableFiles) < 2 ||
				len(task.SuggestedCommands) == 0 ||
				strings.TrimSpace(task.AssertionFocus) == "" {
				t.Fatalf("unexpected repair task: %+v", task)
			}
			if !strings.HasPrefix(task.ID, "repair-"+tt.wantCategory+"-") {
				t.Fatalf("repair task id = %q, want category prefix %q", task.ID, "repair-"+tt.wantCategory+"-")
			}
		})
	}
}

func TestFixSuggestionsRepairTaskGolden(t *testing.T) {
	failures := []types.TestFailure{
		{
			TestName: "TestAdd",
			File:     "calc_test.go",
			Line:     5,
			Error:    "got 3, want 4",
		},
		{
			File:  "calc.go",
			Line:  3,
			Error: "panic: runtime error: index out of range [2] with length 1",
		},
	}
	sourceCode := strings.Join([]string{
		"package calc",
		"",
		"func At(values []int, idx int) int { return values[idx] }",
		"",
		"func Add(a, b int) int { return a + b }",
	}, "\n")
	testCode := strings.Join([]string{
		"package calc",
		"",
		"func TestAdd(t *testing.T) {",
		"	got := Add(1, 2)",
		"	t.Fatalf(\"got %d, want 4\", got)",
		"}",
	}, "\n")

	suggestions := generateFixSuggestions(failures, sourceCode, testCode, "calc.go", filepath.Join("pkg", "calc_test.go"))
	tasks := make([]types.RepairTask, 0, len(suggestions))
	for _, suggestion := range suggestions {
		if suggestion.RepairTask == nil {
			t.Fatalf("missing repair task for suggestion: %+v", suggestion)
		}
		tasks = append(tasks, *suggestion.RepairTask)
	}

	gotBytes, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		t.Fatalf("marshal repair tasks: %v", err)
	}
	got := string(gotBytes) + "\n"
	wantBytes, err := os.ReadFile(filepath.Join("testdata", "golden", "repair_tasks.golden"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	want := string(wantBytes)
	if strings.TrimSpace(got) != strings.TrimSpace(want) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func readToolParserFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "internal", "parser", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func writeTestFile(t *testing.T, dir, rel, content string) string {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func writeFakeLLMProviderOutput(t *testing.T, output string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "provider")
	script := "#!/usr/bin/env sh\nset -eu\ncat >/dev/null\ncat <<'EOF'\n" + output + "\nEOF\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake llm provider: %v", err)
	}
	return path
}
