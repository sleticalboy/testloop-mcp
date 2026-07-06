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
