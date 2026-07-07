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
