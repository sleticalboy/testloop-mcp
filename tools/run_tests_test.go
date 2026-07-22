package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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

func TestJavaTestCommandFindsDeepNonexistentMavenTestPath(t *testing.T) {
	dir := t.TempDir()
	moduleRoot := filepath.Join(dir, "client")
	if err := os.MkdirAll(moduleRoot, 0o755); err != nil {
		t.Fatalf("mkdir module root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleRoot, "pom.xml"), []byte("<project/>"), 0o644); err != nil {
		t.Fatalf("write pom.xml: %v", err)
	}
	testFile := filepath.Join(moduleRoot, "src", "test", "java", "org", "apache", "rocketmq", "client", "java", "route", "EndpointsTest.java")

	cmd := javaTestCommand(context.Background(), testFile, false)

	if filepath.Base(cmd.Path) != "mvn" {
		t.Fatalf("command = %s, want mvn", cmd.Path)
	}
	if cmd.Dir != moduleRoot {
		t.Fatalf("dir = %s, want %s", cmd.Dir, moduleRoot)
	}
	want := []string{"mvn", "-Dtest=EndpointsTest", "test"}
	if !equalStrings(cmd.Args, want) {
		t.Fatalf("args = %v, want %v", cmd.Args, want)
	}
}

func TestJavaTestCommandRunsMavenModuleFromAggregator(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project><modules><module>client</module></modules></project>"), 0o644); err != nil {
		t.Fatalf("write root pom.xml: %v", err)
	}
	moduleRoot := filepath.Join(dir, "client")
	if err := os.MkdirAll(moduleRoot, 0o755); err != nil {
		t.Fatalf("mkdir module root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleRoot, "pom.xml"), []byte("<project/>"), 0o644); err != nil {
		t.Fatalf("write module pom.xml: %v", err)
	}
	testFile := filepath.Join(moduleRoot, "src", "test", "java", "org", "apache", "rocketmq", "client", "java", "route", "EndpointsTest.java")

	cmd := javaTestCommand(context.Background(), testFile, false)

	want := []string{"mvn", "-pl", "client", "-am", "-DfailIfNoTests=false", "-Dtest=EndpointsTest", "test"}
	if !equalStrings(cmd.Args, want) {
		t.Fatalf("args = %v, want %v", cmd.Args, want)
	}
	if cmd.Dir != dir {
		t.Fatalf("dir = %s, want %s", cmd.Dir, dir)
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
	want := []string{"./gradlew", "test", "--tests", "CalculatorTest", "jacocoTestReport"}
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

func TestNormalizeGoTestPathPrefixesRelativePackageDirs(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.MkdirAll("pkg", 0o755); err != nil {
		t.Fatalf("mkdir pkg: %v", err)
	}
	if err := os.WriteFile(filepath.Join("pkg", "calc_test.go"), []byte("package pkg\n"), 0o644); err != nil {
		t.Fatalf("write go file: %v", err)
	}

	if got := normalizeGoTestPath(filepath.Join("pkg", "calc_test.go")); got != "./pkg" {
		t.Fatalf("normalizeGoTestPath(relative file) = %q, want ./pkg", got)
	}
	if got := normalizeGoTestPath("pkg"); got != "./pkg" {
		t.Fatalf("normalizeGoTestPath(relative dir) = %q, want ./pkg", got)
	}
	if got := normalizeGoTestPath("./pkg"); got != "./pkg" {
		t.Fatalf("normalizeGoTestPath(dot relative dir) = %q, want ./pkg", got)
	}
}

func TestGoTestCommandUsesModuleRootForAbsolutePackagePath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	pkg := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatalf("mkdir pkg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "calc_test.go"), []byte("package pkg\n"), 0o644); err != nil {
		t.Fatalf("write go file: %v", err)
	}

	cmd := goTestCommand(context.Background(), pkg, true, true)
	want := []string{"go", "test", "-json", "-v", "-cover", "./pkg"}
	if !equalStrings(cmd.Args, want) {
		t.Fatalf("args = %v, want %v", cmd.Args, want)
	}
	if cmd.Dir != dir {
		t.Fatalf("dir = %s, want %s", cmd.Dir, dir)
	}
}

func TestGoTestCommandUsesModuleRootForAbsoluteTestFilePath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	pkg := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatalf("mkdir pkg: %v", err)
	}
	file := filepath.Join(pkg, "calc_test.go")
	if err := os.WriteFile(file, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatalf("write go file: %v", err)
	}

	cmd := goTestCommand(context.Background(), file, false, false)
	want := []string{"go", "test", "-json", "./pkg"}
	if !equalStrings(cmd.Args, want) {
		t.Fatalf("args = %v, want %v", cmd.Args, want)
	}
	if cmd.Dir != dir {
		t.Fatalf("dir = %s, want %s", cmd.Dir, dir)
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
	if parsed.Action != "ready" {
		t.Fatalf("action = %q, want ready", parsed.Action)
	}
	if parsed.CoveragePercent != 100 {
		t.Fatalf("coverage = %.1f, want 100.0", parsed.CoveragePercent)
	}
	structured := structuredContentAs[types.TestResult](t, result)
	if structured.Framework != parsed.Framework || structured.Status != parsed.Status || structured.Action != parsed.Action || structured.CoveragePercent != parsed.CoveragePercent {
		t.Fatalf("structured content mismatch: %+v vs %+v", structured, parsed)
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
	if parsed.Action != "inspect_failures" {
		t.Fatalf("action = %q, want inspect_failures", parsed.Action)
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
	if parsed.Action != "apply_fix_suggestions" {
		t.Fatalf("action = %q, want apply_fix_suggestions", parsed.Action)
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

func TestHandleRunTestsGoCompileFailureIncludesFixSuggestion(t *testing.T) {
	dir := writeTempGoTestPackage(t, `func TestAdd(t *testing.T) {
	_ = MissingSymbol
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
	if parsed.Status != "fail" || parsed.Action != "apply_fix_suggestions" || len(parsed.Failures) != 1 {
		t.Fatalf("unexpected compile failure result: %+v", parsed)
	}
	if !strings.Contains(parsed.Failures[0].Error, "undefined: MissingSymbol") {
		t.Fatalf("failure should preserve compile detail: %+v", parsed.Failures[0])
	}
	if len(parsed.FixSuggestions) != 1 || parsed.FixSuggestions[0].Category != "undefined_symbol" {
		t.Fatalf("unexpected fix suggestions: %+v", parsed.FixSuggestions)
	}
	if parsed.FixSuggestions[0].RepairTask == nil ||
		parsed.FixSuggestions[0].RepairTask.Category != "undefined_symbol" {
		t.Fatalf("unexpected repair task: %+v", parsed.FixSuggestions[0])
	}
}

func TestHandleRunTestsAllSkippedActionManualReview(t *testing.T) {
	dir := writeTempGoTestPackage(t, `func TestAdd(t *testing.T) {
	t.Skip("manual_review: add real assertions")
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
	if parsed.Status != "pass" || parsed.Passed != 0 || parsed.Skipped != 1 || parsed.Total != 1 {
		t.Fatalf("unexpected skipped result: %+v", parsed)
	}
	if parsed.Action != "manual_review" {
		t.Fatalf("action = %q, want manual_review", parsed.Action)
	}
	structured := structuredContentAs[types.TestResult](t, result)
	if structured.Action != parsed.Action || structured.Skipped != parsed.Skipped {
		t.Fatalf("structured content mismatch: %+v vs %+v", structured, parsed)
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

func TestHandleRunTestsPytestRepairTaskGolden(t *testing.T) {
	dir := writeTempPytestPackage(t)
	installFakePython3(t, pytestFailureOutput())

	result, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:                  filepath.Join(dir, "test_calc.py"),
		Framework:             "pytest",
		IncludeFixSuggestions: true,
		SourceCode:            filepath.Join(dir, "calc.py"),
		TestCode:              filepath.Join(dir, "test_calc.py"),
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
	wantBytes, err := os.ReadFile(filepath.Join("testdata", "golden", "run_tests_pytest_repair_task.golden"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	want := string(wantBytes)
	if strings.TrimSpace(got) != strings.TrimSpace(want) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestHandleRunTestsPytestImportErrorFixSuggestion(t *testing.T) {
	dir := writeTempPytestPackage(t)
	installFakePython3(t, pytestImportErrorOutput())

	result, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:                  filepath.Join(dir, "test_calc.py"),
		Framework:             "pytest",
		IncludeFixSuggestions: true,
		SourceCode:            filepath.Join(dir, "calc.py"),
		TestCode:              filepath.Join(dir, "test_calc.py"),
	})
	if err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}

	var parsed types.TestResult
	if err := json.Unmarshal([]byte(resultText(t, result)), &parsed); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if parsed.Status != "fail" || parsed.Action != "apply_fix_suggestions" || len(parsed.Failures) != 1 {
		t.Fatalf("unexpected pytest import error result: %+v", parsed)
	}
	if len(parsed.FixSuggestions) != 1 {
		t.Fatalf("fix_suggestions len = %d, want 1: %+v", len(parsed.FixSuggestions), parsed.FixSuggestions)
	}
	suggestion := parsed.FixSuggestions[0]
	if suggestion.Category != "python_import_error" ||
		suggestion.RepairTask == nil ||
		suggestion.RepairTask.Category != "python_import_error" ||
		!strings.Contains(suggestion.SuggestedFix, "Python import 失败") {
		t.Fatalf("unexpected python import suggestion: %+v", suggestion)
	}
}

func TestHandleRunTestsJestRepairTaskGolden(t *testing.T) {
	dir := writeTempJestPackage(t)
	installFakeNpx(t, jestFailureOutput())

	result, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:                  filepath.Join(dir, "sum.test.js"),
		Framework:             "jest",
		IncludeFixSuggestions: true,
		SourceCode:            filepath.Join(dir, "sum.js"),
		TestCode:              filepath.Join(dir, "sum.test.js"),
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
	wantBytes, err := os.ReadFile(filepath.Join("testdata", "golden", "run_tests_jest_repair_task.golden"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	want := string(wantBytes)
	if strings.TrimSpace(got) != strings.TrimSpace(want) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestHandleRunTestsVitestRepairTaskGolden(t *testing.T) {
	dir := writeTempVitestPackage(t)
	installFakeNpx(t, vitestFailureOutput())

	result, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:                  filepath.Join(dir, "src", "sum.test.ts"),
		Framework:             "vitest",
		IncludeFixSuggestions: true,
		SourceCode:            filepath.Join(dir, "src", "sum.ts"),
		TestCode:              filepath.Join(dir, "src", "sum.test.ts"),
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
	wantBytes, err := os.ReadFile(filepath.Join("testdata", "golden", "run_tests_vitest_repair_task.golden"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	want := string(wantBytes)
	if strings.TrimSpace(got) != strings.TrimSpace(want) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestHandleRunTestsVitestModuleResolutionFixSuggestion(t *testing.T) {
	dir := writeTempVitestPackage(t)
	installFakeNpx(t, strings.Join([]string{
		"failed to load config from vitest.config.ts",
		"Error: Cannot find module './missing-plugin'",
		"Require stack:",
		"- vitest.config.ts",
	}, "\n"))

	result, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:                  filepath.Join(dir, "src", "sum.test.ts"),
		Framework:             "vitest",
		IncludeFixSuggestions: true,
		SourceCode:            filepath.Join(dir, "src", "sum.ts"),
		TestCode:              filepath.Join(dir, "src", "sum.test.ts"),
	})
	if err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}

	var parsed types.TestResult
	if err := json.Unmarshal([]byte(resultText(t, result)), &parsed); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if parsed.Status != "fail" || parsed.Action != "apply_fix_suggestions" || len(parsed.Failures) != 1 {
		t.Fatalf("unexpected module resolution result: %+v", parsed)
	}
	if len(parsed.FixSuggestions) != 1 {
		t.Fatalf("fix_suggestions len = %d, want 1: %+v", len(parsed.FixSuggestions), parsed.FixSuggestions)
	}
	suggestion := parsed.FixSuggestions[0]
	if suggestion.Category != "module_resolution" ||
		suggestion.RepairTask == nil ||
		suggestion.RepairTask.Category != "module_resolution" ||
		!strings.Contains(suggestion.SuggestedFix, "模块或依赖解析失败") {
		t.Fatalf("unexpected module resolution suggestion: %+v", suggestion)
	}
}

func TestHandleRunTestsMochaRepairTaskGolden(t *testing.T) {
	dir := writeTempMochaPackage(t)
	installFakeNpx(t, mochaFailureOutput())

	result, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:                  filepath.Join(dir, "test", "calc.test.js"),
		Framework:             "mocha",
		IncludeFixSuggestions: true,
		SourceCode:            filepath.Join(dir, "lib", "calc.js"),
		TestCode:              filepath.Join(dir, "test", "calc.test.js"),
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
	wantBytes, err := os.ReadFile(filepath.Join("testdata", "golden", "run_tests_mocha_repair_task.golden"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	want := string(wantBytes)
	if strings.TrimSpace(got) != strings.TrimSpace(want) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestCoverageArgs(t *testing.T) {
	if got := javaMavenArgs(false, ""); !equalStrings(got, []string{"test"}) {
		t.Fatalf("maven no coverage args = %v", got)
	}
	if got := javaMavenArgs(true, ""); !equalStrings(got, []string{"test", "jacoco:report"}) {
		t.Fatalf("maven coverage args = %v", got)
	}
	if got := javaMavenArgs(true, "CalculatorTest"); !equalStrings(got, []string{"-Dtest=CalculatorTest", "test", "jacoco:report"}) {
		t.Fatalf("maven coverage filtered args = %v", got)
	}
	if got := javaGradleArgs(false, ""); !equalStrings(got, []string{"test"}) {
		t.Fatalf("gradle no coverage args = %v", got)
	}
	if got := javaGradleArgs(true, ""); !equalStrings(got, []string{"test", "jacocoTestReport"}) {
		t.Fatalf("gradle coverage args = %v", got)
	}
	if got := javaGradleArgs(true, "CalculatorTest"); !equalStrings(got, []string{"test", "--tests", "CalculatorTest", "jacocoTestReport"}) {
		t.Fatalf("gradle coverage filtered args = %v", got)
	}
}

func TestJavaTestClassFilterOnlyUsesTestJavaFiles(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: filepath.Join("src", "test", "java", "org", "example", "CalculatorTest.java"), want: "CalculatorTest"},
		{path: filepath.Join("src", "main", "java", "org", "example", "Calculator.java"), want: ""},
		{path: filepath.Join("src", "test", "resources", "data.txt"), want: ""},
		{path: "src/test/java/org/example", want: ""},
	}
	for _, tt := range tests {
		if got := javaTestClassFilter(tt.path); got != tt.want {
			t.Fatalf("javaTestClassFilter(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestRunTestRepairCommands(t *testing.T) {
	tests := []struct {
		name      string
		framework string
		source    string
		testFile  string
		want      []string
	}{
		{
			name:      "jest test file",
			framework: "jest",
			source:    "sum.js",
			testFile:  "sum.test.js",
			want:      []string{"npx jest sum.test.js"},
		},
		{
			name:      "vitest test file",
			framework: "vitest",
			source:    "src/sum.ts",
			testFile:  "src/sum.test.ts",
			want:      []string{"npx vitest run src/sum.test.ts"},
		},
		{
			name:      "mocha test file",
			framework: "mocha",
			source:    "lib/calc.js",
			testFile:  "test/calc.test.js",
			want:      []string{"npx mocha test/calc.test.js"},
		},
		{
			name:      "jest source fallback",
			framework: "jest",
			source:    "sum.js",
			want:      []string{"npx jest sum.js"},
		},
		{
			name:      "unsupported framework keeps generic commands",
			framework: "go-test",
			source:    "calc.go",
			testFile:  "calc_test.go",
			want:      nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runTestRepairCommands(tt.framework, tt.source, tt.testFile)
			if !equalStrings(got, tt.want) {
				t.Fatalf("runTestRepairCommands() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleRunTestsAutoDetectedJSRepairCommands(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		path        string
		source      string
		testFile    string
		output      string
		wantCommand string
	}{
		{
			name:        "jest",
			dir:         writeTempJestPackage(t),
			output:      jestFailureOutput(),
			wantCommand: "npx jest fixture/sum.test.js",
		},
		{
			name:        "vitest",
			dir:         writeTempVitestPackage(t),
			output:      vitestFailureOutput(),
			wantCommand: "npx vitest run fixture/src/sum.test.ts",
		},
		{
			name:        "mocha",
			dir:         writeTempMochaPackage(t),
			output:      mochaFailureOutput(),
			wantCommand: "npx mocha fixture/test/calc.test.js",
		},
	}
	tests[0].path = filepath.Join(tests[0].dir, "sum.test.js")
	tests[0].source = filepath.Join(tests[0].dir, "sum.js")
	tests[0].testFile = filepath.Join(tests[0].dir, "sum.test.js")
	tests[1].path = filepath.Join(tests[1].dir, "src", "sum.test.ts")
	tests[1].source = filepath.Join(tests[1].dir, "src", "sum.ts")
	tests[1].testFile = filepath.Join(tests[1].dir, "src", "sum.test.ts")
	tests[2].path = filepath.Join(tests[2].dir, "test", "calc.test.js")
	tests[2].source = filepath.Join(tests[2].dir, "lib", "calc.js")
	tests[2].testFile = filepath.Join(tests[2].dir, "test", "calc.test.js")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installFakeNpx(t, tt.output)
			result, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
				Path:                  tt.path,
				IncludeFixSuggestions: true,
				SourceCode:            tt.source,
				TestCode:              tt.testFile,
			})
			if err != nil {
				t.Fatalf("HandleRunTests returned error: %v", err)
			}
			var parsed types.TestResult
			if err := json.Unmarshal([]byte(resultText(t, result)), &parsed); err != nil {
				t.Fatalf("unmarshal result: %v", err)
			}
			if parsed.Framework != tt.name {
				t.Fatalf("framework = %q, want %q", parsed.Framework, tt.name)
			}
			contract := canonicalRunTestsRepairContract(parsed, tt.dir)
			if len(contract.FixSuggestions) != 1 ||
				contract.FixSuggestions[0].RepairTask == nil ||
				len(contract.FixSuggestions[0].RepairTask.SuggestedCommands) != 1 {
				t.Fatalf("unexpected fix suggestions: %+v", contract.FixSuggestions)
			}
			gotCommand := contract.FixSuggestions[0].RepairTask.SuggestedCommands[0]
			if gotCommand != tt.wantCommand {
				t.Fatalf("suggested command = %q, want %q", gotCommand, tt.wantCommand)
			}
		})
	}
}

func TestHandleRunTestsJSUsesPackageRootAndRelativePath(t *testing.T) {
	tests := []struct {
		name      string
		dir       string
		path      string
		framework string
		output    string
		wantArgs  string
	}{
		{
			name:      "jest",
			dir:       writeTempJestPackage(t),
			framework: "jest",
			output:    jestFailureOutput(),
			wantArgs:  "jest --verbose sum.test.js",
		},
		{
			name:      "vitest",
			dir:       writeTempVitestPackage(t),
			framework: "vitest",
			output:    vitestFailureOutput(),
			wantArgs:  "vitest run src/sum.test.ts",
		},
		{
			name:      "mocha",
			dir:       writeTempMochaPackage(t),
			framework: "mocha",
			output:    mochaFailureOutput(),
			wantArgs:  "mocha --reporter spec test/calc.test.js",
		},
	}
	tests[0].path = filepath.Join(tests[0].dir, "sum.test.js")
	tests[1].path = filepath.Join(tests[1].dir, "src", "sum.test.ts")
	tests[2].path = filepath.Join(tests[2].dir, "test", "calc.test.js")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logPath := installFakeNpxRecorder(t, tt.output)
			if _, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
				Path:      tt.path,
				Framework: tt.framework,
			}); err != nil {
				t.Fatalf("HandleRunTests returned error: %v", err)
			}

			logData, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatalf("read fake npx log: %v", err)
			}
			logText := string(logData)
			wantDir, err := filepath.Abs(filepath.Clean(tt.dir))
			if err != nil {
				t.Fatalf("abs dir: %v", err)
			}
			if !strings.Contains(logText, "PWD="+wantDir+"\n") {
				t.Fatalf("fake npx cwd log = %q, want PWD=%s", logText, wantDir)
			}
			if !strings.Contains(logText, "ARGS="+tt.wantArgs+"\n") {
				t.Fatalf("fake npx args log = %q, want ARGS=%s", logText, tt.wantArgs)
			}
		})
	}
}

func TestHandleRunTestsJSCoverageUsesPackageRootAndRelativePath(t *testing.T) {
	tests := []struct {
		name      string
		dir       string
		path      string
		framework string
		output    string
		wantArgs  string
	}{
		{
			name:      "jest",
			dir:       writeTempJestPackage(t),
			framework: "jest",
			output:    jestFailureOutput(),
			wantArgs:  "jest --verbose --coverage sum.test.js",
		},
		{
			name:      "vitest",
			dir:       writeTempVitestPackage(t),
			framework: "vitest",
			output:    vitestFailureOutput(),
			wantArgs:  "vitest run --coverage src/sum.test.ts",
		},
		{
			name:      "mocha",
			dir:       writeTempMochaPackage(t),
			framework: "mocha",
			output:    mochaFailureOutput(),
			wantArgs:  "mocha --reporter spec --coverage test/calc.test.js",
		},
	}
	tests[0].path = filepath.Join(tests[0].dir, "sum.test.js")
	tests[1].path = filepath.Join(tests[1].dir, "src", "sum.test.ts")
	tests[2].path = filepath.Join(tests[2].dir, "test", "calc.test.js")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logPath := installFakeNpxRecorder(t, tt.output)
			if _, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
				Path:      tt.path,
				Framework: tt.framework,
				Coverage:  true,
			}); err != nil {
				t.Fatalf("HandleRunTests returned error: %v", err)
			}

			logText := readTextFile(t, logPath)
			wantDir := absCleanPath(t, tt.dir)
			if !strings.Contains(logText, "PWD="+wantDir+"\n") {
				t.Fatalf("fake npx cwd log = %q, want PWD=%s", logText, wantDir)
			}
			if !strings.Contains(logText, "ARGS="+tt.wantArgs+"\n") {
				t.Fatalf("fake npx args log = %q, want ARGS=%s", logText, tt.wantArgs)
			}
		})
	}
}

func TestHandleRunTestsNodeTestUsesPackageRootAndRelativePath(t *testing.T) {
	dir := writeTempNodeTestPackage(t)
	testFile := filepath.Join(dir, "test", "sum.test.js")
	logPath := installFakeNodeRecorder(t, nodeTestFailureOutput())

	result, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:                  testFile,
		IncludeFixSuggestions: true,
	})
	if err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}

	var parsed types.TestResult
	if err := json.Unmarshal([]byte(resultText(t, result)), &parsed); err != nil {
		t.Fatalf("unmarshal run result: %v", err)
	}
	if parsed.Framework != "node-test" || parsed.Status != "fail" || len(parsed.Failures) != 1 {
		t.Fatalf("unexpected run result: %+v", parsed)
	}
	if len(parsed.FixSuggestions) != 1 || parsed.FixSuggestions[0].RepairTask == nil {
		t.Fatalf("expected repair task, got %+v", parsed.FixSuggestions)
	}
	wantCommand := "node --test " + filepath.ToSlash(testFile)
	if !equalStrings(parsed.FixSuggestions[0].RepairTask.SuggestedCommands, []string{wantCommand}) {
		t.Fatalf("suggested commands = %+v, want %q", parsed.FixSuggestions[0].RepairTask.SuggestedCommands, wantCommand)
	}

	logText := readTextFile(t, logPath)
	wantDir := absCleanPath(t, dir)
	if !strings.Contains(logText, "PWD="+wantDir+"\n") {
		t.Fatalf("fake node cwd log = %q, want PWD=%s", logText, wantDir)
	}
	if !strings.Contains(logText, "ARGS=--test test/sum.test.js\n") {
		t.Fatalf("fake node args log = %q, want node --test relative path", logText)
	}
}

func TestHandleRunTestsJSCustomCommandTemplateUsesPackageRootAndRelativePath(t *testing.T) {
	dir := writeTempMochaPackage(t)
	testFile := filepath.Join(dir, "test", "calc.test.js")
	t.Setenv("TESTLOOP_JS_TEST_COMMAND", "npx egg-bin test --timeout 60000 {path}")
	logPath := installFakeNpxRecorder(t, mochaFailureOutput())

	if _, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:      testFile,
		Framework: "mocha",
	}); err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}

	logText := readTextFile(t, logPath)
	wantDir := absCleanPath(t, dir)
	if !strings.Contains(logText, "PWD="+wantDir+"\n") {
		t.Fatalf("fake npx cwd log = %q, want PWD=%s", logText, wantDir)
	}
	if !strings.Contains(logText, "ARGS=egg-bin test --timeout 60000 test/calc.test.js\n") {
		t.Fatalf("fake npx args log = %q, want custom egg-bin command", logText)
	}
}

func TestHandleRunTestsJSCustomCommandTemplateKillsProcessGroupOnTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell process group cancellation is only configured on Unix platforms")
	}
	dir := writeTempMochaPackage(t)
	testFile := filepath.Join(dir, "test", "calc.test.js")
	t.Setenv("TESTLOOP_JS_TEST_COMMAND", "sh -c 'sleep 5'")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	result, _, err := HandleRunTests(ctx, nil, runTestsInput{
		Path:      testFile,
		Framework: "mocha",
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("custom command timeout took %s, child process likely survived context cancellation", elapsed)
	}

	var parsed types.TestResult
	if err := json.Unmarshal([]byte(resultText(t, result)), &parsed); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if parsed.Status != "fail" || parsed.Failed != 1 || len(parsed.Failures) != 1 {
		t.Fatalf("unexpected timeout fallback result: %+v", parsed)
	}
}

func TestHandleRunTestsNonZeroExitFallsBackToRunnerFailure(t *testing.T) {
	dir := writeTempMochaPackage(t)
	testFile := filepath.Join(dir, "test", "calc.test.js")
	t.Setenv("TESTLOOP_JS_TEST_COMMAND", "npx egg-bin test --timeout 60000 {path}")
	installFakeNpx(t, "Exception during run: TypeScript compile failed")

	result, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:      testFile,
		Framework: "mocha",
	})
	if err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}

	var parsed types.TestResult
	if err := json.Unmarshal([]byte(resultText(t, result)), &parsed); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if parsed.Status != "fail" || parsed.Failed != 1 || len(parsed.Failures) != 1 {
		t.Fatalf("unexpected fallback result: %+v", parsed)
	}
	if !strings.Contains(parsed.Failures[0].Error, "Exception during run") {
		t.Fatalf("unexpected fallback failure: %+v", parsed.Failures[0])
	}
}

func TestPytestArgsUsesRelativePath(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "tests", "test_calc.py")
	got := pytestArgs(testFile, dir, true, true)
	want := []string{"-m", "pytest", "-v", "--cov", "tests/test_calc.py"}
	if !equalStrings(got, want) {
		t.Fatalf("pytestArgs() = %v, want %v", got, want)
	}
}

func TestHandleRunTestsPytestUsesProjectRootAndRelativePath(t *testing.T) {
	dir := writeTempNestedPytestPackage(t)
	testFile := filepath.Join(dir, "tests", "test_calc.py")
	logPath := installFakePython3Recorder(t, pytestFailureOutput())

	if _, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:      testFile,
		Framework: "pytest",
	}); err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake python3 log: %v", err)
	}
	logText := string(logData)
	wantDir, err := filepath.Abs(filepath.Clean(dir))
	if err != nil {
		t.Fatalf("abs dir: %v", err)
	}
	if !strings.Contains(logText, "PWD="+wantDir+"\n") {
		t.Fatalf("fake python3 cwd log = %q, want PWD=%s", logText, wantDir)
	}
	if !strings.Contains(logText, "ARGS=-m pytest -v tests/test_calc.py\n") {
		t.Fatalf("fake python3 args log = %q, want relative pytest args", logText)
	}
}

func TestHandleRunTestsPytestCoverageUsesProjectRootAndRelativePath(t *testing.T) {
	dir := writeTempNestedPytestPackage(t)
	testFile := filepath.Join(dir, "tests", "test_calc.py")
	logPath := installFakePython3Recorder(t, pytestFailureOutput())

	if _, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:      testFile,
		Framework: "pytest",
		Coverage:  true,
	}); err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}

	logText := readTextFile(t, logPath)
	wantDir := absCleanPath(t, dir)
	if !strings.Contains(logText, "PWD="+wantDir+"\n") {
		t.Fatalf("fake python3 cwd log = %q, want PWD=%s", logText, wantDir)
	}
	if !strings.Contains(logText, "ARGS=-m pytest -v --cov tests/test_calc.py\n") {
		t.Fatalf("fake python3 args log = %q, want coverage pytest args", logText)
	}
}

func TestHandleRunTestsPytestCustomCommandTemplateUsesProjectRootAndRelativePath(t *testing.T) {
	dir := writeTempNestedPytestPackage(t)
	testFile := filepath.Join(dir, "tests", "test_calc.py")
	t.Setenv("TESTLOOP_PYTEST_COMMAND", "python3 -m pytest {verbose} {coverage} {path}")
	logPath := installFakePython3Recorder(t, pytestFailureOutput())

	if _, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:      testFile,
		Framework: "pytest",
		Coverage:  true,
	}); err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}

	logText := readTextFile(t, logPath)
	wantDir := absCleanPath(t, dir)
	if !strings.Contains(logText, "PWD="+wantDir+"\n") {
		t.Fatalf("fake python3 cwd log = %q, want PWD=%s", logText, wantDir)
	}
	if !strings.Contains(logText, "ARGS=-m pytest -v --cov tests/test_calc.py\n") {
		t.Fatalf("fake python3 args log = %q, want custom pytest command", logText)
	}
}

func TestHandleRunTestsPytestUsesTestsParentWhenNoConfigMarker(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "app")
	testsDir := filepath.Join(dir, "tests", "app", "utils")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("mkdir app: %v", err)
	}
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("mkdir tests: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "__init__.py"), []byte("VALUE = 1\n"), 0o644); err != nil {
		t.Fatalf("write app init: %v", err)
	}
	testFile := filepath.Join(testsDir, "test_app.py")
	if err := os.WriteFile(testFile, []byte("from app import VALUE\n\ndef test_value():\n    assert VALUE == 1\n"), 0o644); err != nil {
		t.Fatalf("write pytest file: %v", err)
	}
	logPath := installFakePython3Recorder(t, pytestFailureOutput())

	if _, _, err := HandleRunTests(context.Background(), nil, runTestsInput{
		Path:      testFile,
		Framework: "pytest",
	}); err != nil {
		t.Fatalf("HandleRunTests returned error: %v", err)
	}

	logText := readTextFile(t, logPath)
	wantDir := absCleanPath(t, dir)
	if !strings.Contains(logText, "PWD="+wantDir+"\n") {
		t.Fatalf("fake python3 cwd log = %q, want PWD=%s", logText, wantDir)
	}
	if !strings.Contains(logText, "ARGS=-m pytest -v tests/app/utils/test_app.py\n") {
		t.Fatalf("fake python3 args log = %q, want relative pytest args", logText)
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

func readTextFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return string(data)
}

func absCleanPath(t *testing.T, path string) string {
	t.Helper()
	result, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		t.Fatalf("abs path %s: %v", path, err)
	}
	return result
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
			task.SuggestedCommands = canonicalFixtureCommands(task.SuggestedCommands, fixtureDir)
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

func canonicalFixtureCommands(commands []string, fixtureDir string) []string {
	result := make([]string, len(commands))
	for i, command := range commands {
		parts := strings.Fields(command)
		for j, part := range parts {
			if j == 0 {
				continue
			}
			parts[j] = canonicalFixtureCommandArg(part, fixtureDir)
		}
		result[i] = strings.Join(parts, " ")
	}
	return result
}

func canonicalFixtureCommandArg(arg string, fixtureDir string) string {
	cleanArg := filepath.ToSlash(filepath.Clean(arg))
	cleanFixture := filepath.ToSlash(filepath.Clean(fixtureDir))
	cleanBase := filepath.ToSlash(filepath.Base(fixtureDir))
	if strings.HasPrefix(cleanArg, cleanFixture+"/") ||
		strings.HasPrefix(cleanArg, "./"+cleanBase+"/") ||
		strings.HasPrefix(cleanArg, cleanBase+"/") {
		return canonicalFixturePath(arg, fixtureDir)
	}
	return arg
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
	if !filepath.IsAbs(path) && !strings.HasPrefix(cleanPath, "../") {
		return "fixture/" + cleanPath
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

func writeTempPytestPackage(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", "run-tests-pytest-")
	if err != nil {
		t.Fatalf("mkdir temp pytest package: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatalf("remove temp pytest package: %v", err)
		}
	})

	source := strings.Join([]string{
		"def add(a, b):",
		"    return a + b",
		"",
		"def divide(a, b):",
		"    if b == 0:",
		"        # keep line numbers aligned with pytest fixture",
		"        raise ValueError(\"division by zero\")",
		"    return a / b",
	}, "\n") + "\n"
	testSource := strings.Join([]string{
		"from calc import divide",
		"",
		"def test_divide():",
		"    divide(1, 0)",
	}, "\n") + "\n"

	if err := os.WriteFile(filepath.Join(dir, "calc.py"), []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test_calc.py"), []byte(testSource), 0o644); err != nil {
		t.Fatalf("write test source: %v", err)
	}
	return "./" + filepath.Base(dir)
}

func writeTempNestedPytestPackage(t *testing.T) string {
	t.Helper()
	dir := writeTempPytestPackage(t)
	testsDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("mkdir nested pytest tests: %v", err)
	}
	oldTest := filepath.Join(dir, "test_calc.py")
	newTest := filepath.Join(testsDir, "test_calc.py")
	data, err := os.ReadFile(oldTest)
	if err != nil {
		t.Fatalf("read root pytest test: %v", err)
	}
	if err := os.WriteFile(newTest, data, 0o644); err != nil {
		t.Fatalf("write nested pytest test: %v", err)
	}
	if err := os.Remove(oldTest); err != nil {
		t.Fatalf("remove root pytest test: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[tool.pytest.ini_options]\ntestpaths = [\"tests\"]\n"), 0o644); err != nil {
		t.Fatalf("write pyproject.toml: %v", err)
	}
	return dir
}

func writeTempJestPackage(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", "run-tests-jest-")
	if err != nil {
		t.Fatalf("mkdir temp jest package: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatalf("remove temp jest package: %v", err)
		}
	})

	source := strings.Join([]string{
		"function add(a, b) {",
		"  return a + b",
		"}",
		"",
		"module.exports = { add }",
	}, "\n") + "\n"
	testSource := strings.Join([]string{
		"const { add } = require('./sum')",
		"",
		"test('adds 1 + 2 to equal 3', () => {",
		"  const result = add(1, 2)",
		"  expect(result).toBe(3)",
		"})",
	}, "\n") + "\n"

	if err := os.WriteFile(filepath.Join(dir, "sum.js"), []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sum.test.js"), []byte(testSource), 0o644); err != nil {
		t.Fatalf("write test source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"jest"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	return "./" + filepath.Base(dir)
}

func writeTempVitestPackage(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", "run-tests-vitest-")
	if err != nil {
		t.Fatalf("mkdir temp vitest package: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatalf("remove temp vitest package: %v", err)
		}
	})

	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	source := strings.Join([]string{
		"export function add(a: number, b: number): number {",
		"  return a + b + 1",
		"}",
	}, "\n") + "\n"
	testSource := strings.Join([]string{
		"import { describe, expect, test } from 'vitest'",
		"import { add } from './sum'",
		"",
		"describe('sum', () => {",
		"  test('adds values', () => {",
		"    const result = add(1, 2)",
		"    // keep line numbers aligned with vitest fixture",
		"    expect(result).toBe(3)",
		"  })",
		"})",
	}, "\n") + "\n"

	if err := os.WriteFile(filepath.Join(srcDir, "sum.ts"), []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "sum.test.ts"), []byte(testSource), 0o644); err != nil {
		t.Fatalf("write test source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"vitest run"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	return "./" + filepath.Base(dir)
}

func writeTempMochaPackage(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", "run-tests-mocha-")
	if err != nil {
		t.Fatalf("mkdir temp mocha package: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatalf("remove temp mocha package: %v", err)
		}
	})

	libDir := filepath.Join(dir, "lib")
	testDir := filepath.Join(dir, "test")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("mkdir lib: %v", err)
	}
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("mkdir test: %v", err)
	}

	source := strings.Join([]string{
		"function add(a, b) {",
		"  return a + b",
		"}",
		"",
		"function divide(a, b) {",
		"  return a / b",
		"}",
		"",
		"module.exports = { add, divide }",
	}, "\n") + "\n"
	testSource := strings.Join([]string{
		"const { expect } = require('chai')",
		"const { add, divide } = require('../lib/calc')",
		"",
		"describe('calc', () => {",
		"  it('add() should add numbers', () => {",
		"    expect(add(1, 2)).to.equal(3)",
		"  })",
		"",
		"  it('divide() should handle division by zero', () => {",
		"    const result = divide(4, 1)",
		"    // keep line numbers aligned with mocha fixture",
		"    expect(result).to.equal(3)",
		"  })",
		"})",
	}, "\n") + "\n"

	if err := os.WriteFile(filepath.Join(libDir, "calc.js"), []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "calc.test.js"), []byte(testSource), 0o644); err != nil {
		t.Fatalf("write test source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"mocha"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	return "./" + filepath.Base(dir)
}

func writeTempNodeTestPackage(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", "run-tests-node-")
	if err != nil {
		t.Fatalf("mkdir temp node package: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatalf("remove temp node package: %v", err)
		}
	})

	testDir := filepath.Join(dir, "test")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("mkdir test: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sum.js"), []byte("exports.add = (a, b) => a + b + 1;\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	testSource := strings.Join([]string{
		"const test = require('node:test')",
		"const assert = require('node:assert/strict')",
		"const { add } = require('../sum')",
		"",
		"test('adds values', () => {",
		"  assert.equal(add(1, 2), 3)",
		"})",
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(testDir, "sum.test.js"), []byte(testSource), 0o644); err != nil {
		t.Fatalf("write test source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"node --test"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	return "./" + filepath.Base(dir)
}

func installFakePython3(t *testing.T, output string) {
	t.Helper()
	fakeBin := t.TempDir()
	script := "#!/usr/bin/env sh\ncat <<'PYTEST_OUTPUT'\n" + output + "\nPYTEST_OUTPUT\nexit 1\n"
	path := filepath.Join(fakeBin, "python3")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake python3: %v", err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func installFakePython3Recorder(t *testing.T, output string) string {
	t.Helper()
	fakeBin := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "python3.log")
	script := "#!/usr/bin/env sh\n" +
		"{\n" +
		"  printf 'PWD=%s\\n' \"$PWD\"\n" +
		"  printf 'ARGS=%s\\n' \"$*\"\n" +
		"} > '" + logPath + "'\n" +
		"cat <<'PYTEST_OUTPUT'\n" + output + "\nPYTEST_OUTPUT\n" +
		"exit 1\n"
	path := filepath.Join(fakeBin, "python3")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake python3: %v", err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

func installFakeNpx(t *testing.T, output string) {
	t.Helper()
	fakeBin := t.TempDir()
	script := "#!/usr/bin/env sh\ncat <<'JEST_OUTPUT'\n" + output + "\nJEST_OUTPUT\nexit 1\n"
	path := filepath.Join(fakeBin, "npx")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake npx: %v", err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func installFakeNpxRecorder(t *testing.T, output string) string {
	t.Helper()
	fakeBin := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "npx.log")
	script := "#!/usr/bin/env sh\n" +
		"{\n" +
		"  printf 'PWD=%s\\n' \"$PWD\"\n" +
		"  printf 'ARGS=%s\\n' \"$*\"\n" +
		"} > '" + logPath + "'\n" +
		"cat <<'NPX_OUTPUT'\n" + output + "\nNPX_OUTPUT\n" +
		"exit 1\n"
	path := filepath.Join(fakeBin, "npx")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake npx: %v", err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

func installFakeNodeRecorder(t *testing.T, output string) string {
	t.Helper()
	fakeBin := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "node.log")
	script := "#!/usr/bin/env sh\n" +
		"{\n" +
		"  printf 'PWD=%s\\n' \"$PWD\"\n" +
		"  printf 'ARGS=%s\\n' \"$*\"\n" +
		"} > '" + logPath + "'\n" +
		"cat <<'NODE_OUTPUT'\n" + output + "\nNODE_OUTPUT\n" +
		"exit 1\n"
	path := filepath.Join(fakeBin, "node")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake node: %v", err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

func pytestFailureOutput() string {
	return strings.Join([]string{
		"test_calc.py::test_divide FAILED                                           [100%]",
		"",
		"=================================== FAILURES ===================================",
		"________________________________ test_divide _________________________________",
		"",
		"    def test_divide():",
		">       divide(1, 0)",
		"",
		"test_calc.py:4:",
		"_ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _",
		"",
		"a = 1, b = 0",
		"",
		"    def divide(a, b):",
		"        if b == 0:",
		">           raise ValueError(\"division by zero\")",
		"E           ValueError: division by zero",
		"",
		"calc.py:7: ValueError",
		"============================== 1 failed in 0.01s ===============================",
	}, "\n")
}

func nodeTestFailureOutput() string {
	return strings.Join([]string{
		"TAP version 13",
		"# Subtest: adds values",
		"not ok 1 - adds values",
		"  ---",
		"  duration_ms: 2.1",
		"  failureType: 'testCodeFailure'",
		"  error: 'Expected values to be strictly equal:'",
		"  code: 'ERR_ASSERTION'",
		"  expected: 3",
		"  actual: 4",
		"  operator: 'strictEqual'",
		"  stack: |-",
		"    TestContext.<anonymous> (test/sum.test.js:6:10)",
		"    Test.runInAsyncScope (node:async_hooks:206:9)",
		"  ...",
		"1..1",
		"# tests 1",
		"# suites 0",
		"# pass 0",
		"# fail 1",
		"# cancelled 0",
		"# skipped 0",
		"# todo 0",
		"# duration_ms 18.4",
	}, "\n")
}

func pytestImportErrorOutput() string {
	return strings.Join([]string{
		"==================================== ERRORS ====================================",
		"________________________ ERROR collecting test_calc.py ________________________",
		"ImportError while importing test module '/tmp/project/test_calc.py'.",
		"Traceback:",
		"test_calc.py:1: in <module>",
		"    from missing_app import VALUE",
		"E   ModuleNotFoundError: No module named 'missing_app'",
		"=========================== short test summary info ============================",
		"ERROR test_calc.py",
		"!!!!!!!!!!!!!!!!!!!! Interrupted: 1 error during collection !!!!!!!!!!!!!!!!!!!!",
		"=============================== 1 error in 0.01s ===============================",
	}, "\n")
}

func jestFailureOutput() string {
	return strings.Join([]string{
		"FAIL  ./sum.test.js",
		"  ✕ adds 1 + 2 to equal 3 (1 ms)",
		"",
		"  ● sum › adds 1 + 2 to equal 3",
		"    expect(received).toBe(expected)",
		"    Expected: 3",
		"    Received: 4",
		"      at Object.<anonymous> (sum.test.js:5:15)",
		"",
		"Test Suites: 1 failed, 0 passed, 1 total",
		"Tests:       1 failed, 0 passed, 1 total",
	}, "\n")
}

func vitestFailureOutput() string {
	return strings.Join([]string{
		" FAIL  src/sum.test.ts > sum > adds values",
		"AssertionError: expected 4 to be 3 // Object.is equality",
		"",
		"Expected: 3",
		"Received: 4",
		"",
		" ❯ src/sum.test.ts:8:18",
		"      6| test('adds values', () => {",
		"      7|   const result = add(1, 2)",
		"      8|   expect(result).toBe(3)",
		"       |                  ^",
		"",
		" Test Files  1 failed (1)",
		"      Tests  1 failed (1)",
	}, "\n")
}

func mochaFailureOutput() string {
	return strings.Join([]string{
		"  calc",
		"    ✓ add() should add numbers",
		"    1) divide() should handle division by zero",
		"",
		"  1 passing (12ms)",
		"  1 failing",
		"",
		"  1) calc",
		"       divide() should handle division by zero:",
		"     AssertionError: expected 4 to equal 3",
		"      at Context.<anonymous> (test/calc.test.js:12:18)",
	}, "\n")
}
