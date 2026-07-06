package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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
}

func TestHandleParseResultsRequiresOutput(t *testing.T) {
	if _, _, err := HandleParseResults(context.Background(), nil, parseResultsInput{}); err == nil {
		t.Fatal("expected missing output error")
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
}

func TestHandleParseCoverageValidatesInput(t *testing.T) {
	if _, _, err := HandleParseCoverage(context.Background(), nil, parseCoverageInput{}); err == nil {
		t.Fatal("expected missing data error")
	}
	if _, _, err := HandleParseCoverage(context.Background(), nil, parseCoverageInput{Data: "not coverage data"}); err == nil {
		t.Fatal("expected invalid coverage data error")
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
	if _, err := os.Stat(generated.TestFile); err != nil {
		t.Fatalf("expected generated test file: %v", err)
	}
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

	var generated types.GenerateTestsOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &generated); err != nil {
		t.Fatalf("unmarshal generated output: %v", err)
	}
	if generated.TestFile != testFile {
		t.Fatalf("test file = %q, want %q", generated.TestFile, testFile)
	}
	if generated.CoverageTask == nil || generated.Context == nil || generated.Context.CoverageTask == nil {
		t.Fatalf("expected coverage task in output and context: %+v", generated)
	}
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("expected generated coverage task test file: %v", err)
	}
	if !strings.Contains(string(content), "TestAddCoverageTask") {
		t.Fatalf("generated test file missing task test name:\n%s", content)
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
	if suggestions[0].Confidence != 0.8 || !strings.Contains(suggestions[0].SuggestedFix, "实际值: 2") {
		t.Fatalf("unexpected suggestion: %+v", suggestions[0])
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
		wantText       string
	}{
		{0.9, "nil"},
		{0.9, "边界检查"},
		{0.95, "除零检查"},
		{0.7, "拼写正确"},
		{0.7, "类型转换"},
	}
	for i, check := range checks {
		if suggestions[i].Confidence != check.wantConfidence || !strings.Contains(suggestions[i].SuggestedFix, check.wantText) {
			t.Fatalf("suggestion %d = %+v, want confidence %.1f and text %q", i, suggestions[i], check.wantConfidence, check.wantText)
		}
	}
}
