package tools

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

func TestJavaTestCommandUsesCoverageArgs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0o644); err != nil {
		t.Fatalf("write pom.xml: %v", err)
	}

	cmd := javaTestCommand(context.Background(), dir, true)
	if filepath.Base(cmd.Path) != "mvn" {
		t.Fatalf("command = %s, want mvn", cmd.Path)
	}
	want := []string{"mvn", "test", "jacoco:report"}
	if !equalStrings(cmd.Args, want) {
		t.Fatalf("args = %v, want %v", cmd.Args, want)
	}
	if cmd.Dir != dir {
		t.Fatalf("dir = %s, want %s", cmd.Dir, dir)
	}
}

func TestJavaTestCommandPrefersWrappers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("wrapper command paths are unix-style")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0o644); err != nil {
		t.Fatalf("write pom.xml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mvnw"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write mvnw: %v", err)
	}

	cmd := javaTestCommand(context.Background(), dir, false)
	want := []string{"./mvnw", "test"}
	if !equalStrings(cmd.Args, want) {
		t.Fatalf("args = %v, want %v", cmd.Args, want)
	}
}

func TestJavaTestCommandPrefersGradleWrapper(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("wrapper command paths are unix-style")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "build.gradle"), []byte("plugins {}\n"), 0o644); err != nil {
		t.Fatalf("write build.gradle: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gradlew"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write gradlew: %v", err)
	}
	testFile := filepath.Join(dir, "src", "test", "CalculatorTest.java")
	if err := os.MkdirAll(filepath.Dir(testFile), 0o755); err != nil {
		t.Fatalf("mkdir test dir: %v", err)
	}
	if err := os.WriteFile(testFile, []byte("class CalculatorTest {}\n"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	cmd := javaTestCommand(context.Background(), testFile, true)
	want := []string{"./gradlew", "test", "jacocoTestReport"}
	if !equalStrings(cmd.Args, want) {
		t.Fatalf("args = %v, want %v", cmd.Args, want)
	}
	if cmd.Dir != dir {
		t.Fatalf("dir = %s, want %s", cmd.Dir, dir)
	}
}

func TestCollectJavaCoveragePercentReadsMavenReport(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0o644); err != nil {
		t.Fatalf("write pom.xml: %v", err)
	}
	reportDir := filepath.Join(dir, "target", "site", "jacoco")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatalf("mkdir report dir: %v", err)
	}
	report := `<?xml version="1.0" encoding="UTF-8"?>
<report name="demo">
  <package name="com/example">
    <sourcefile name="Calculator.java">
      <line nr="10" mi="0" ci="1"/>
      <line nr="11" mi="1" ci="0"/>
      <counter type="LINE" missed="1" covered="1"/>
    </sourcefile>
  </package>
  <counter type="LINE" missed="1" covered="1"/>
</report>`
	if err := os.WriteFile(filepath.Join(reportDir, "jacoco.xml"), []byte(report), 0o644); err != nil {
		t.Fatalf("write jacoco.xml: %v", err)
	}

	percent, ok := collectJavaCoveragePercent(dir)
	if !ok {
		t.Fatal("expected Java coverage report to be parsed")
	}
	if percent != 50 {
		t.Fatalf("percent = %.1f, want 50.0", percent)
	}

	if percent := collectCoveragePercent(context.Background(), "junit", dir, 12.5); percent != 50 {
		t.Fatalf("collectCoveragePercent junit = %.1f, want 50.0", percent)
	}
	if percent := collectCoveragePercent(context.Background(), "go-test", dir, 12.5); percent != 12.5 {
		t.Fatalf("collectCoveragePercent passthrough = %.1f, want 12.5", percent)
	}
}

func TestNormalizeGoTestPathUsesContainingDirForGoFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "calc_test.go")
	if err := os.WriteFile(file, []byte("package calc\n"), 0o644); err != nil {
		t.Fatalf("write go file: %v", err)
	}

	if got := normalizeGoTestPath(file); got != dir {
		t.Fatalf("normalizeGoTestPath(%q) = %q, want %q", file, got, dir)
	}
	if got := normalizeGoTestPath(dir); got != dir {
		t.Fatalf("normalizeGoTestPath(%q) = %q, want %q", dir, got, dir)
	}
}

func TestFindProjectRootWalksParents(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "pkg", "calc")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\n"), 0o644); err != nil {
		t.Fatalf("write Cargo.toml: %v", err)
	}
	source := filepath.Join(nested, "calc.rs")
	if err := os.WriteFile(source, []byte("fn add() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if got := findProjectRoot(source, "Cargo.toml"); got != dir {
		t.Fatalf("findProjectRoot = %q, want %q", got, dir)
	}
}

func TestHandleRunTestsValidatesInput(t *testing.T) {
	if _, _, err := HandleRunTests(context.Background(), nil, runTestsInput{}); err == nil {
		t.Fatal("expected missing path error")
	}
	if _, _, err := HandleRunTests(context.Background(), nil, runTestsInput{Path: ".", Framework: "unknown"}); err == nil {
		t.Fatal("expected unsupported framework error")
	}
}

func TestHandleRunTestsExecutesGoTest(t *testing.T) {
	dir := writeTempGoTestPackage(t, `func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("bad add")
	}
}`)

	result, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:      dir,
		Framework: "go-test",
		Coverage:  true,
	})
	if err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}

	var parsed types.TestResult
	if err := json.Unmarshal([]byte(resultText(t, result)), &parsed); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if parsed.Framework != "go-test" || parsed.Status != "pass" || parsed.Passed != 1 || parsed.Failed != 0 {
		t.Fatalf("unexpected result: %+v", parsed)
	}
	if parsed.CoveragePercent != 100 {
		t.Fatalf("coverage = %.1f, want 100.0", parsed.CoveragePercent)
	}
}

func TestHandleRunTestsParsesGoFailure(t *testing.T) {
	dir := writeTempGoTestPackage(t, `func TestAdd(t *testing.T) {
	t.Fatalf("boom")
}`)

	result, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:      dir,
		Framework: "go-test",
	})
	if err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}

	var parsed types.TestResult
	if err := json.Unmarshal([]byte(resultText(t, result)), &parsed); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if parsed.Status != "fail" || parsed.Failed != 1 || len(parsed.Failures) != 1 {
		t.Fatalf("unexpected failure result: %+v", parsed)
	}
	if parsed.Failures[0].TestName != "TestAdd" || !strings.Contains(parsed.Failures[0].Error, "boom") {
		t.Fatalf("unexpected failure: %+v", parsed.Failures[0])
	}
	if len(parsed.FixSuggestions) != 0 {
		t.Fatalf("fix_suggestions should be omitted by default: %+v", parsed.FixSuggestions)
	}
}

func TestHandleRunTestsCanIncludeFixSuggestions(t *testing.T) {
	dir := writeTempGoTestPackage(t, `func TestAdd(t *testing.T) {
	if got := Add(1, 2); got != 4 {
		t.Fatalf("got %d, want 4", got)
	}
}`)

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

	var parsed types.TestResult
	if err := json.Unmarshal([]byte(resultText(t, result)), &parsed); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if parsed.Status != "fail" || len(parsed.Failures) != 1 {
		t.Fatalf("unexpected failure result: %+v", parsed)
	}
	if len(parsed.FixSuggestions) != 1 {
		t.Fatalf("fix_suggestions len = %d, want 1: %+v", len(parsed.FixSuggestions), parsed.FixSuggestions)
	}
	suggestion := parsed.FixSuggestions[0]
	if suggestion.Category != "expectation_mismatch" ||
		suggestion.ContextFile != filepath.Join(dir, "calc_test.go") ||
		suggestion.RepairTask == nil ||
		suggestion.RepairTask.Category != "expectation_mismatch" ||
		!strings.Contains(suggestion.SuggestedFix, "实际值: 3") ||
		!strings.Contains(suggestion.SuggestedFix, "期望值: 4") {
		t.Fatalf("unexpected fix suggestion: %+v", suggestion)
	}
	if suggestion.RepairTask.TargetFile != filepath.Join(dir, "calc_test.go") ||
		len(suggestion.RepairTask.EditableFiles) != 2 ||
		suggestion.RepairTask.SuggestedCommands[0] != "go test ./..." {
		t.Fatalf("unexpected repair task: %+v", suggestion.RepairTask)
	}
}

func TestHandleRunTestsRepairTaskGolden(t *testing.T) {
	dir := writeTempGoTestPackage(t, `func TestAdd(t *testing.T) {
	if got := Add(1, 2); got != 4 {
		t.Fatalf("got %d, want 4", got)
	}
}`)

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

	var parsed types.TestResult
	if err := json.Unmarshal([]byte(resultText(t, result)), &parsed); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	contract := canonicalRunTestsRepairContract(parsed, dir)
	gotBytes, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}
	got := string(gotBytes) + "\n"
	wantBytes, err := os.ReadFile(filepath.Join("testdata", "golden", "run_tests_repair_task.golden"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	want := string(wantBytes)
	if strings.TrimSpace(got) != strings.TrimSpace(want) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestCoverageArgs(t *testing.T) {
	if got := javaMavenArgs(false); !equalStrings(got, []string{"test"}) {
		t.Fatalf("maven no coverage args = %v", got)
	}
	if got := javaMavenArgs(true); !equalStrings(got, []string{"test", "jacoco:report"}) {
		t.Fatalf("maven coverage args = %v", got)
	}
	if got := javaGradleArgs(false); !equalStrings(got, []string{"test"}) {
		t.Fatalf("gradle no coverage args = %v", got)
	}
	if got := javaGradleArgs(true); !equalStrings(got, []string{"test", "jacocoTestReport"}) {
		t.Fatalf("gradle coverage args = %v", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type runTestsRepairContract struct {
	Status         string                `json:"status"`
	Framework      string                `json:"framework"`
	Failures       []types.TestFailure   `json:"failures"`
	FixSuggestions []types.FixSuggestion `json:"fix_suggestions"`
}

func canonicalRunTestsRepairContract(result types.TestResult, fixtureDir string) runTestsRepairContract {
	failures := append([]types.TestFailure(nil), result.Failures...)
	for i := range failures {
		failures[i].File = canonicalFixturePath(failures[i].File, fixtureDir)
	}

	suggestions := append([]types.FixSuggestion(nil), result.FixSuggestions...)
	for i := range suggestions {
		suggestions[i].File = canonicalFixturePath(suggestions[i].File, fixtureDir)
		suggestions[i].ContextFile = canonicalFixturePath(suggestions[i].ContextFile, fixtureDir)
		if suggestions[i].RepairTask != nil {
			task := *suggestions[i].RepairTask
			task.TargetFile = canonicalFixturePath(task.TargetFile, fixtureDir)
			task.ContextFile = canonicalFixturePath(task.ContextFile, fixtureDir)
			task.EditableFiles = canonicalFixturePaths(task.EditableFiles, fixtureDir)
			suggestions[i].RepairTask = &task
		}
	}

	return runTestsRepairContract{
		Status:         result.Status,
		Framework:      result.Framework,
		Failures:       failures,
		FixSuggestions: suggestions,
	}
}

func canonicalFixturePaths(paths []string, fixtureDir string) []string {
	result := make([]string, len(paths))
	for i, path := range paths {
		result[i] = canonicalFixturePath(path, fixtureDir)
	}
	return result
}

func canonicalFixturePath(path string, fixtureDir string) string {
	if path == "" {
		return ""
	}
	cleanPath := filepath.ToSlash(filepath.Clean(path))
	cleanFixture := filepath.ToSlash(filepath.Clean(fixtureDir))
	cleanBase := filepath.ToSlash(filepath.Base(fixtureDir))
	for _, prefix := range []string{cleanFixture + "/", "./" + cleanBase + "/", cleanBase + "/"} {
		if strings.HasPrefix(cleanPath, prefix) {
			return "fixture/" + strings.TrimPrefix(cleanPath, prefix)
		}
	}
	if !strings.Contains(cleanPath, "/") {
		return "fixture/" + cleanPath
	}
	return cleanPath
}

func writeTempGoTestPackage(t *testing.T, testBody string) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", "run-tests-")
	if err != nil {
		t.Fatalf("mkdir temp package: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatalf("remove temp package: %v", err)
		}
	})

	source := `package runtest

func Add(a, b int) int { return a + b }
`
	testSource := `package runtest

import "testing"

` + testBody + "\n"

	if err := os.WriteFile(filepath.Join(dir, "calc.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "calc_test.go"), []byte(testSource), 0o644); err != nil {
		t.Fatalf("write test source: %v", err)
	}
	return "./" + filepath.Base(dir)
}
