package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sleticalboy/testloop-mcp/types"
)

type runTestsFixture struct {
	Status          string                `json:"status"`
	Action          string                `json:"action"`
	Framework       string                `json:"framework"`
	Total           int                   `json:"total"`
	Passed          int                   `json:"passed"`
	Failed          int                   `json:"failed"`
	Skipped         int                   `json:"skipped"`
	CoveragePercent float64               `json:"coverage_percent"`
	Failures        []types.TestFailure   `json:"failures"`
	FixSuggestions  []types.FixSuggestion `json:"fix_suggestions,omitempty"`
}

func TestHandleRunTestsActionCategoryFixture(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "go.mod", "module example.com/calc\n\ngo 1.23\n")
	writeFixtureFile(t, dir, "calc.go", `package calc

func Add(a, b int) int {
	return a + b
}
`)
	writeFixtureFile(t, dir, "calc_test.go", `package calc

import "testing"

func TestAdd(t *testing.T) {
	if got := Add(2, 2); got != 5 {
		t.Fatalf("Add(2, 2) = %d, want 5", got)
	}
}
`)

	result, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:                  dir,
		Framework:             "go-test",
		IncludeFixSuggestions: true,
		SourceCode:            filepath.Join(dir, "calc.go"),
		TestCode:              filepath.Join(dir, "calc_test.go"),
	})
	if err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}

	var out types.TestResult
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal run_tests output: %v", err)
	}
	gotBytes, err := json.MarshalIndent(runTestsFixtureFromOutput(dir, out), "", "  ")
	if err != nil {
		t.Fatalf("marshal run_tests fixture: %v", err)
	}
	got := string(gotBytes) + "\n"
	wantBytes, err := os.ReadFile(filepath.Join("..", "docs", "fixtures", "run-tests", "apply-fix-suggestions.json"))
	if err != nil {
		t.Fatalf("read run_tests fixture: %v\n--- got ---\n%s", err, got)
	}
	want := string(wantBytes)
	if got != want {
		t.Fatalf("run_tests fixture mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func runTestsFixtureFromOutput(root string, out types.TestResult) runTestsFixture {
	return runTestsFixture{
		Status:          out.Status,
		Action:          out.Action,
		Framework:       out.Framework,
		Total:           out.Total,
		Passed:          out.Passed,
		Failed:          out.Failed,
		Skipped:         out.Skipped,
		CoveragePercent: out.CoveragePercent,
		Failures:        normalizeFixtureFailures(root, out.Failures),
		FixSuggestions:  normalizeFixtureFixSuggestions(root, out.FixSuggestions),
	}
}

func writeFixtureFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture file %s: %v", rel, err)
	}
}
