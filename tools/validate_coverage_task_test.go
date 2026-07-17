package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sleticalboy/testloop-mcp/types"
)

func normalizedJSONValue(t *testing.T, data []byte) any {
	t.Helper()
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("unmarshal JSON value: %v\n%s", err, data)
	}
	return value
}

func normalizedStructuredJSONValue(t *testing.T, value any) any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal structured value %T: %v", value, err)
	}
	return normalizedJSONValue(t, data)
}

func TestHandleValidateCoverageTaskGeneratesAndRunsGoTask(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/calc\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "calc.go")
	src := `package calc

func Add(a, b int) int {
	if a == 0 {
		return b
	}
	return a + b
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-1",
		Framework:       "go-test",
		File:            source,
		Target:          "Add",
		Kind:            "function",
		LineRange:       "4-6",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: a == 0"},
		UncoveredLines:  []int{4, 5, 6},
		SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入", "设置 a 覆盖未执行分支", "设置 b 覆盖未执行分支"},
		Goal:            "为 Add 补充测试，覆盖未执行行段 4-6",
		Command:         "go test ./...",
		TestFile:        filepath.Join(dir, "calc_test.go"),
		TestName:        "TestAdd",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
		Confidence:      0.95,
	}

	result, structured, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}
	if out.Status != "passed" || out.Action != "ready" {
		t.Fatalf("unexpected validation status: %+v", out)
	}
	if out.Generated == nil || out.Generated.TestFile != task.TestFile || out.Generated.GeneratedCases == 0 {
		t.Fatalf("unexpected generated output: %+v", out.Generated)
	}
	if out.RunResult == nil || out.RunResult.Status != "pass" || out.RunResult.Failed != 0 {
		t.Fatalf("unexpected run result: %+v", out.RunResult)
	}
	if out.Metadata["test_file"] != task.TestFile || out.Metadata["framework"] != "go-test" {
		t.Fatalf("unexpected metadata: %+v", out.Metadata)
	}
	structuredOutput, ok := structured.(types.CoverageTaskValidationOutput)
	if !ok || structuredOutput.Status != "passed" {
		t.Fatalf("structured output = %#v, want passed validation output", structured)
	}
}

func TestHandleValidateCoverageTaskStructuredContentMatchesTextJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/calc\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "calc.go")
	src := `package calc

func Add(a, b int) int {
	if a == 0 {
		return b
	}
	return a + b
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-1",
		Framework:       "go-test",
		File:            source,
		Target:          "Add",
		Kind:            "function",
		LineRange:       "4-6",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: a == 0"},
		UncoveredLines:  []int{4, 5, 6},
		SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入"},
		Goal:            "为 Add 补充测试，覆盖未执行行段 4-6",
		Command:         "go test ./...",
		TestFile:        filepath.Join(dir, "calc_test.go"),
		TestName:        "TestAdd",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
		Confidence:      0.95,
	}

	result, structured, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}

	textPayload := normalizedJSONValue(t, []byte(resultText(t, result)))
	structuredPayload := normalizedStructuredJSONValue(t, result.StructuredContent)
	if !reflect.DeepEqual(structuredPayload, textPayload) {
		t.Fatalf("StructuredContent mismatch\nstructured: %#v\ntext: %#v", structuredPayload, textPayload)
	}
	returnedPayload := normalizedStructuredJSONValue(t, structured)
	if !reflect.DeepEqual(returnedPayload, textPayload) {
		t.Fatalf("handler structured return mismatch\nstructured: %#v\ntext: %#v", returnedPayload, textPayload)
	}

	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}
	if out.Status != "passed" || out.Action != "ready" {
		t.Fatalf("unexpected status/action: %+v", out)
	}
	if out.CoverageTask == nil || out.CoverageTask.ID != task.ID || out.CoverageTask.Target != task.Target {
		t.Fatalf("coverage_task missing or changed: %+v", out.CoverageTask)
	}
	if out.Generated == nil || out.Generated.Status != "ok" || out.Generated.TestFile != task.TestFile {
		t.Fatalf("generated missing or changed: %+v", out.Generated)
	}
	if out.RunResult == nil || out.RunResult.Status != "pass" || out.RunResult.Failed != 0 {
		t.Fatalf("run_result missing or changed: %+v", out.RunResult)
	}
	if out.Metadata == nil || out.Metadata["framework"] != "go-test" || out.Metadata["test_file"] != task.TestFile {
		t.Fatalf("metadata missing or changed: %+v", out.Metadata)
	}
}

type validateCoverageTaskFixture struct {
	Status       string                               `json:"status"`
	Action       string                               `json:"action"`
	CoverageTask types.CoverageTestTask               `json:"coverage_task"`
	Generated    validateCoverageTaskGeneratedFixture `json:"generated"`
	RunResult    validateCoverageTaskRunResultFixture `json:"run_result"`
	Metadata     map[string]any                       `json:"metadata"`
}

type validateCoverageTaskGeneratedFixture struct {
	Status         string                  `json:"status"`
	TestFile       string                  `json:"test_file"`
	GeneratedCases int                     `json:"generated_cases"`
	Provider       string                  `json:"provider"`
	CoverageTask   *types.CoverageTestTask `json:"coverage_task,omitempty"`
}

type validateCoverageTaskRunResultFixture struct {
	Status          string                `json:"status"`
	Framework       string                `json:"framework"`
	Total           int                   `json:"total"`
	Passed          int                   `json:"passed"`
	Failed          int                   `json:"failed"`
	Skipped         int                   `json:"skipped"`
	CoveragePercent float64               `json:"coverage_percent"`
	Failures        []types.TestFailure   `json:"failures"`
	FixSuggestions  []types.FixSuggestion `json:"fix_suggestions,omitempty"`
}

func TestHandleValidateCoverageTaskReadyFixture(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/calc\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "calc.go")
	src := `package calc

func Add(a, b int) int {
	if a == 0 {
		return b
	}
	return a + b
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-1",
		Framework:       "go-test",
		File:            source,
		Target:          "Add",
		Kind:            "function",
		LineRange:       "4-6",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: a == 0"},
		UncoveredLines:  []int{4, 5, 6},
		SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入"},
		Goal:            "为 Add 补充测试，覆盖未执行行段 4-6",
		Command:         "go test ./...",
		TestFile:        filepath.Join(dir, "calc_test.go"),
		TestName:        "TestAdd",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
		Confidence:      0.95,
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}

	gotBytes, err := json.MarshalIndent(validateCoverageTaskFixtureFromOutput(t, dir, out), "", "  ")
	if err != nil {
		t.Fatalf("marshal ready fixture: %v", err)
	}
	got := string(gotBytes) + "\n"
	wantBytes, err := os.ReadFile(filepath.Join("..", "docs", "fixtures", "validate-coverage-task-ready.json"))
	if err != nil {
		t.Fatalf("read ready fixture: %v\n--- got ---\n%s", err, got)
	}
	want := string(wantBytes)
	if got != want {
		t.Fatalf("ready fixture mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func validateCoverageTaskFixtureFromOutput(t *testing.T, root string, out types.CoverageTaskValidationOutput) validateCoverageTaskFixture {
	t.Helper()
	if out.CoverageTask == nil {
		t.Fatalf("coverage_task missing in output: %+v", out)
	}
	if out.Generated == nil {
		t.Fatalf("generated missing in output: %+v", out)
	}
	if out.RunResult == nil {
		t.Fatalf("run_result missing in output: %+v", out)
	}
	coverageTask := *out.CoverageTask
	normalizeCoverageTaskFixturePaths(root, &coverageTask)
	var generatedCoverageTask *types.CoverageTestTask
	if out.Generated.CoverageTask != nil {
		generated := *out.Generated.CoverageTask
		normalizeCoverageTaskFixturePaths(root, &generated)
		generatedCoverageTask = &generated
	}
	metadata := map[string]any{}
	if framework, ok := out.Metadata["framework"]; ok {
		metadata["framework"] = framework
	}
	if testFile, ok := out.Metadata["test_file"].(string); ok {
		metadata["test_file"] = normalizeFixturePath(root, testFile)
	}
	for _, key := range []string{
		"unreachable",
		"unreachable_reason",
		"environment_dependent",
		"environment_reason",
		"protocol_dependent",
		"protocol_reason",
		"database_dependent",
		"database_reason",
		"external_service_dependent",
		"external_service_reason",
		"private_method",
		"private_reason",
		"public_entry_candidates",
		"internal_symbol",
		"internal_reason",
		"no_runtime",
		"no_runtime_reason",
		"coverage_target_hit",
		"coverage_miss_reason",
	} {
		if value, ok := out.Metadata[key]; ok {
			metadata[key] = value
		}
	}
	return validateCoverageTaskFixture{
		Status:       out.Status,
		Action:       out.Action,
		CoverageTask: coverageTask,
		Generated: validateCoverageTaskGeneratedFixture{
			Status:         out.Generated.Status,
			TestFile:       normalizeFixturePath(root, out.Generated.TestFile),
			GeneratedCases: out.Generated.GeneratedCases,
			Provider:       out.Generated.Provider,
			CoverageTask:   generatedCoverageTask,
		},
		RunResult: validateCoverageTaskRunResultFixture{
			Status:          out.RunResult.Status,
			Framework:       out.RunResult.Framework,
			Total:           out.RunResult.Total,
			Passed:          out.RunResult.Passed,
			Failed:          out.RunResult.Failed,
			Skipped:         out.RunResult.Skipped,
			CoveragePercent: out.RunResult.CoveragePercent,
			Failures:        normalizeFixtureFailures(root, out.RunResult.Failures),
			FixSuggestions:  normalizeFixtureFixSuggestions(root, out.RunResult.FixSuggestions),
		},
		Metadata: metadata,
	}
}

func normalizeCoverageTaskFixturePaths(root string, task *types.CoverageTestTask) {
	task.File = normalizeFixturePath(root, task.File)
	task.TestFile = normalizeFixturePath(root, task.TestFile)
}

func normalizeFixturePath(root string, path string) string {
	if strings.TrimSpace(path) == "" {
		return path
	}
	if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}

func normalizeFixtureFailures(root string, failures []types.TestFailure) []types.TestFailure {
	if failures == nil {
		return nil
	}
	normalized := append([]types.TestFailure(nil), failures...)
	for i := range normalized {
		normalized[i].File = normalizeFixturePath(root, normalized[i].File)
	}
	return normalized
}

func normalizeFixtureFixSuggestions(root string, suggestions []types.FixSuggestion) []types.FixSuggestion {
	if suggestions == nil {
		return nil
	}
	normalized := append([]types.FixSuggestion(nil), suggestions...)
	for i := range normalized {
		normalized[i].File = normalizeFixturePath(root, normalized[i].File)
		normalized[i].ContextFile = normalizeFixturePath(root, normalized[i].ContextFile)
		if normalized[i].RepairTask != nil {
			repair := *normalized[i].RepairTask
			repair.TargetFile = normalizeFixturePath(root, repair.TargetFile)
			repair.ContextFile = normalizeFixturePath(root, repair.ContextFile)
			for j := range repair.EditableFiles {
				repair.EditableFiles[j] = normalizeFixturePath(root, repair.EditableFiles[j])
			}
			normalized[i].RepairTask = &repair
		}
	}
	return normalized
}

func TestHandleValidateCoverageTaskReturnsAdjustedCoverageTaskName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/calc\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(source, []byte("package calc\nfunc Add(a, b int) int { return a + b }\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	testFile := filepath.Join(dir, "calc_test.go")
	existing := `package calc

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("unexpected Add result")
	}
}
`
	if err := os.WriteFile(testFile, []byte(existing), 0o644); err != nil {
		t.Fatalf("write existing test: %v", err)
	}
	task := types.CoverageTestTask{
		ID:        "go-test-1",
		Framework: "go-test",
		File:      source,
		Target:    "Add",
		Kind:      "function",
		LineRange: "2-2",
		TestFile:  testFile,
		TestName:  "TestAdd",
		Command:   "go test ./...",
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}
	if out.CoverageTask == nil || out.CoverageTask.TestName != "TestAddCoverage2_2" {
		t.Fatalf("validation coverage task test_name = %+v, want adjusted name", out.CoverageTask)
	}
	if out.Generated == nil || out.Generated.CoverageTask == nil || out.Generated.CoverageTask.TestName != out.CoverageTask.TestName {
		t.Fatalf("generated coverage task was not aligned with validation output: %+v", out.Generated)
	}
}

func TestHandleValidateCoverageTaskMarksJavaReadyLineMiss(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project></project>\n"), 0o644); err != nil {
		t.Fatalf("write pom.xml: %v", err)
	}
	source := filepath.Join(dir, "src", "main", "java", "com", "example", "Calculator.java")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	src := `package com.example;

public class Calculator {
    public int add(int a, int b) {
        if (a == 0) {
            return b;
        }
        return a + b;
    }
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	installFakeMvnSuccessWithJaCoCo(t, `<report>
  <package name="com/example">
    <sourcefile name="Calculator.java">
      <line nr="4" mi="1" ci="0"/>
      <counter type="LINE" missed="1" covered="0"/>
    </sourcefile>
  </package>
  <counter type="LINE" missed="1" covered="0"/>
</report>`)
	task := types.CoverageTestTask{
		ID:              "junit-line-miss",
		Framework:       "junit",
		File:            source,
		Target:          "Calculator.add",
		Kind:            "method",
		LineRange:       "4-4",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: a == 0"},
		SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入"},
		TestFile:        filepath.Join(dir, "src", "test", "java", "com", "example", "CalculatorTestLoopTest.java"),
		TestName:        "shouldCoverCalculatorAddGap",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		Framework:    "junit",
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}
	if out.Status != "failed" || out.Action != "needs_better_input" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.RunResult == nil || out.RunResult.Status != "pass" {
		t.Fatalf("expected passing test run with failed coverage validation, got %+v", out.RunResult)
	}
	if out.Metadata["coverage_target_hit"] != false {
		t.Fatalf("expected coverage_target_hit=false metadata, got %+v", out.Metadata)
	}
	missed, ok := out.Metadata["coverage_missed_lines"].([]any)
	if !ok || len(missed) != 1 || missed[0].(float64) != 4 {
		t.Fatalf("unexpected missed lines metadata: %+v", out.Metadata["coverage_missed_lines"])
	}
}

func TestHandleValidateCoverageTaskMarksUnreachableSkippedTask(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/remote\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "remote.go")
	src := `package remote

import (
	"net/http"
	"strings"
)

func RemoteIP(r *http.Request, fallback string) string {
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		partIndex := len(parts) - 1
		if partIndex < 0 {
			partIndex = 0
		}
		return parts[partIndex]
	}
	return fallback
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-unreachable",
		Framework:       "go-test",
		File:            source,
		Target:          "RemoteIP",
		Kind:            "function",
		LineRange:       "12-14",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: partIndex < 0"},
		SuggestedInputs: []string{"构造满足条件 `partIndex < 0` 的输入"},
		Goal:            "为 RemoteIP 补充测试，覆盖不可达行段",
		Command:         "go test ./...",
		TestFile:        filepath.Join(dir, "remote_test.go"),
		TestName:        "TestRemoteIPPartIndex",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
		Confidence:      0.95,
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}
	if out.Status != "passed" || out.Action != "manual_review_unreachable" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.RunResult == nil || out.RunResult.Skipped == 0 {
		t.Fatalf("expected skipped generated TODO test, got run result: %+v", out.RunResult)
	}
	if out.Metadata["unreachable"] != true {
		t.Fatalf("expected unreachable metadata, got %+v", out.Metadata)
	}
	reason, _ := out.Metadata["unreachable_reason"].(string)
	if !strings.Contains(reason, "partIndex < 0") {
		t.Fatalf("unexpected unreachable reason: %q", reason)
	}
}

func TestHandleValidateCoverageTaskMarksInitAsManualReview(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "main.go")
	src := `package app

var initialized bool

func init() {
	initialized = true
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-init",
		Framework:       "go-test",
		File:            source,
		Target:          "init",
		Kind:            "function",
		LineRange:       "5-5",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: err != nil"},
		Goal:            "复核 init 初始化路径",
		Command:         "go test ./...",
		TestFile:        filepath.Join(dir, "main_test.go"),
		TestName:        "TestInit",
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}
	if out.Status != "passed" || out.Action != "manual_review_unreachable" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.RunResult == nil || out.RunResult.Skipped == 0 {
		t.Fatalf("expected skipped init review test, got run result: %+v", out.RunResult)
	}
	reason, _ := out.Metadata["unreachable_reason"].(string)
	if !strings.Contains(reason, "init functions cannot be called directly") {
		t.Fatalf("unexpected init manual-review reason: %q", reason)
	}
	if out.Generated == nil || strings.Contains(out.Generated.Preview, "\t\t\tinit()\n") {
		t.Fatalf("generated init task should not call init directly: %+v", out.Generated)
	}
}

func TestHandleValidateCoverageTaskMarksSystemResourceErrorAsEnvironmentReview(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/sys\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "server.go")
	src := `package sys

type Cpu struct {
	Cores int
}

func InitCPU() (c Cpu, err error) {
	if err != nil {
		return c, err
	}
	c.Cores = 1
	return c, nil
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-init-cpu",
		Framework:       "go-test",
		File:            source,
		Target:          "InitCPU",
		Kind:            "function",
		LineRange:       "8-10",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: err != nil"},
		SuggestedInputs: []string{"构造满足条件 `err != nil` 的输入"},
		Goal:            "为 InitCPU 补充测试，覆盖系统资源错误分支",
		Command:         "go test ./...",
		TestFile:        filepath.Join(dir, "server_test.go"),
		TestName:        "TestInitCPU",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
		Confidence:      0.95,
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}
	if out.Status != "passed" || out.Action != "manual_review_environment" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.RunResult == nil || out.RunResult.Skipped == 0 {
		t.Fatalf("expected skipped generated TODO test, got run result: %+v", out.RunResult)
	}
	if out.Metadata["environment_dependent"] != true {
		t.Fatalf("expected environment metadata, got %+v", out.Metadata)
	}
	reason, _ := out.Metadata["environment_reason"].(string)
	if !strings.Contains(reason, "InitCPU") || !strings.Contains(reason, "static tests cannot force it") {
		t.Fatalf("unexpected environment reason: %q", reason)
	}
}

func TestCoverageTaskEnvironmentReasonAcceptsManualReviewMarker(t *testing.T) {
	task := &types.CoverageTestTask{
		Target: "get_binary_stderr",
	}
	generated := &types.GenerateTestsOutput{
		Preview: "def test_get_binary_stderr_covers_gap():\n    pytest.skip(\"manual_review_environment: get_binary_stderr depends on process std stream binary-wrapper state\")\n",
	}
	result := &types.TestResult{
		Status:  "pass",
		Skipped: 1,
	}

	reason := coverageTaskEnvironmentReason(task, generated, result)
	if !strings.Contains(reason, "get_binary_stderr") || !strings.Contains(reason, "OS/runtime environment state") {
		t.Fatalf("unexpected environment reason: %q", reason)
	}
}

func TestHandleValidateCoverageTaskMarksSocketWriteErrorAsProtocolReview(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/protocol\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "client.go")
	src := `package protocol

import (
	"encoding/json"
	"net"
)

type BindRequest struct {
	Control string ` + "`json:\"control\"`" + `
}

type Status struct{}

func QueryStatus(socketPath string) (Status, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return Status{}, err
	}
	bind, _ := json.Marshal(BindRequest{Control: "status"})
	if _, err := conn.Write(append(bind, '\n')); err != nil {
		return Status{}, err
	}
	return Status{}, nil
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-query-status-write",
		Framework:       "go-test",
		File:            source,
		Target:          "QueryStatus",
		Kind:            "function",
		LineRange:       "19-21",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: err != nil"},
		SuggestedInputs: []string{"构造满足条件 `err != nil` 的输入"},
		Goal:            "为 QueryStatus 补充 socket 写入错误分支",
		Command:         "go test ./...",
		TestFile:        filepath.Join(dir, "client_test.go"),
		TestName:        "TestQueryStatusWriteError",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
		Confidence:      0.95,
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}
	if out.Status != "passed" || out.Action != "manual_review_protocol" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.RunResult == nil || out.RunResult.Skipped == 0 {
		t.Fatalf("expected skipped generated TODO test, got run result: %+v", out.RunResult)
	}
	if out.Metadata["protocol_dependent"] != true {
		t.Fatalf("expected protocol metadata, got %+v", out.Metadata)
	}
	reason, _ := out.Metadata["protocol_reason"].(string)
	if !strings.Contains(reason, "QueryStatus") || !strings.Contains(reason, "socket write") {
		t.Fatalf("unexpected protocol reason: %q", reason)
	}
}

func TestHandleValidateCoverageTaskMarksRepoDBBranchAsDatabaseReview(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/repo\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "repo.go")
	src := `package repo

import "errors"

type CigaretteRepo struct{}

func (r *CigaretteRepo) query(userID int64) error {
	return nil
}

func (r *CigaretteRepo) ListByUserID(userID int64) ([]int, error) {
	err := r.query(userID)
	if err != nil {
		return nil, err
	}
	return []int{1}, nil
}

var _ = errors.New
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-repo-db",
		Framework:       "go-test",
		File:            source,
		Target:          "CigaretteRepo.ListByUserID",
		Kind:            "method",
		LineRange:       "12-14",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: result.Error != nil"},
		SuggestedInputs: []string{"构造满足条件 `result.Error != nil` 的输入"},
		Goal:            "为 CigaretteRepo.ListByUserID 补充数据库错误分支",
		Command:         "go test ./...",
		TestFile:        filepath.Join(dir, "repo_test.go"),
		TestName:        "TestCigaretteRepoListByUserID",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
		Confidence:      0.95,
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}
	if out.Status != "passed" || out.Action != "manual_review_database" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.RunResult == nil || out.RunResult.Skipped == 0 {
		t.Fatalf("expected skipped generated TODO test, got run result: %+v", out.RunResult)
	}
	if out.Metadata["database_dependent"] != true {
		t.Fatalf("expected database metadata, got %+v", out.Metadata)
	}
	reason, _ := out.Metadata["database_reason"].(string)
	if !strings.Contains(reason, "CigaretteRepo.ListByUserID") || !strings.Contains(reason, "GORM DB behavior") {
		t.Fatalf("unexpected database reason: %q", reason)
	}
}

func TestHandleValidateCoverageTaskMarksExternalServiceTimeoutAsManualReview(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module","devDependencies":{"mocha":"^10.0.0"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	source := filepath.Join(dir, "producer.ts")
	src := `export class Producer {
  rpcClientManager = {
    sendMessage: async () => ({ ok: true }),
  };

  constructor(options: { endpoints?: string } = {}) {
    void options;
  }

  async send(message: { topic: string }) {
    if (message.topic) {
      return await this.rpcClientManager.sendMessage();
    }
    return null;
  }
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	installFakeNpxFailure(t, "Error: Timeout of 60000ms exceeded while waiting for gRPC sendMessage")
	task := types.CoverageTestTask{
		ID:              "mocha-producer-external",
		Framework:       "mocha",
		File:            source,
		Target:          "Producer.send",
		Kind:            "method",
		LineRange:       "11-12",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: message.topic"},
		SuggestedInputs: []string{"构造满足条件 `message.topic` 的输入", "设置 endpoints 覆盖未执行分支"},
		Goal:            "为 Producer.send 补充发送路径测试",
		Command:         "npx mocha producer.ts",
		TestFile:        filepath.Join(dir, "producer.spec.ts"),
		TestName:        "covers Producer send external service path",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
		Confidence:      0.9,
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		Framework:    "mocha",
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}
	if out.Status != "failed" || out.Action != "manual_review_external_service" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.Metadata["external_service_dependent"] != true {
		t.Fatalf("expected external service metadata, got %+v", out.Metadata)
	}
	reason, _ := out.Metadata["external_service_reason"].(string)
	if !strings.Contains(reason, "Producer.send") || !strings.Contains(reason, "live RPC/external service") {
		t.Fatalf("unexpected external service reason: %q", reason)
	}
}

func TestCoverageTaskDatabaseReasonRecognizesManualReviewMarker(t *testing.T) {
	task := &types.CoverageTestTask{
		ID:        "pytest-db-1",
		Framework: "pytest",
		Target:    "delete_app",
		GapType:   "error_path",
		LineRange: "670-672",
	}
	generated := &types.GenerateTestsOutput{
		Preview: "def test_delete_app_database_commit_failure_requires_review():\n    __import__('pytest').skip('manual_review_database: delete_app depends on SQLAlchemy database transaction/session behavior')\n",
	}
	result := &types.TestResult{
		Status:  "pass",
		Skipped: 1,
	}

	reason := coverageTaskDatabaseReason(task, generated, result)
	if !strings.Contains(reason, "delete_app") || !strings.Contains(reason, "database transaction/session behavior") {
		t.Fatalf("unexpected database reason: %q", reason)
	}
}

func TestHandleValidateCoverageTaskMarksJSPrivateMethodAsManualReview(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module","scripts":{"test":"vitest run"},"devDependencies":{"vitest":"^3.0.0"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	source := filepath.Join(dir, "token-store.js")
	src := `export class TokenStore {
  #readSecret(name) {
    if (name) {
      return true
    }
    return false
  }

  get(name) {
    return this.#readSecret(name)
  }
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	installFakeNpxSuccess(t, strings.Join([]string{
		" RUN  v3.2.4 " + dir,
		"",
		" ↓ token-store.test.js (1 test | 1 skipped)",
		"",
		" Test Files  1 skipped (1)",
		"      Tests  1 skipped (1)",
	}, "\n"))
	task := types.CoverageTestTask{
		ID:              "vitest-private-1",
		Framework:       "vitest",
		File:            source,
		Target:          "TokenStore.#readSecret",
		Kind:            "method",
		LineRange:       "4-4",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: name"},
		SuggestedInputs: []string{"构造满足条件 `name` 的输入"},
		Goal:            "为 TokenStore.#readSecret 补充私有方法分支测试",
		Command:         "npx vitest run token-store.js",
		TestFile:        filepath.Join(dir, "token-store.test.js"),
		TestName:        "covers TokenStore private read secret",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
		Confidence:      0.95,
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		Framework:    "vitest",
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}
	if out.Status != "passed" || out.Action != "manual_review_private" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.RunResult == nil || out.RunResult.Status != "pass" {
		t.Fatalf("expected passing private review test run, got %+v", out.RunResult)
	}
	if out.Metadata["private_method"] != true {
		t.Fatalf("expected private method metadata, got %+v", out.Metadata)
	}
	reason, _ := out.Metadata["private_reason"].(string)
	if !strings.Contains(reason, "TokenStore.#readSecret") || !strings.Contains(reason, "private method") {
		t.Fatalf("unexpected private reason: %q", reason)
	}
	entries, ok := out.Metadata["public_entry_candidates"].([]any)
	if !ok || len(entries) != 1 || entries[0] != "TokenStore.get" {
		t.Fatalf("unexpected public entry candidates: %+v", out.Metadata["public_entry_candidates"])
	}
	if out.Generated == nil || !strings.Contains(out.Generated.Preview, "manual_review_private: TokenStore.#readSecret") ||
		!strings.Contains(out.Generated.Preview, "public_entry_candidates: TokenStore.get") ||
		strings.Contains(out.Generated.Preview, "instance.#readSecret") {
		t.Fatalf("expected generated private method review skip, got %+v", out.Generated)
	}
}

func TestHandleValidateCoverageTaskMarksJSInternalClassAsManualReview(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module","scripts":{"test":"vitest run"},"devDependencies":{"vitest":"^3.0.0"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	source := filepath.Join(dir, "cache.js")
	src := `class LocalCache {
  get(key) {
    if (!this.values[key]) {
      this.values[key] = null
    }
    return this.values[key]
  }
}

export default class CacheFacade {
  constructor() {
    this.cache = new LocalCache()
  }
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	installFakeNpxSuccess(t, strings.Join([]string{
		" RUN  v3.2.4 " + dir,
		"",
		" ↓ cache.test.js (1 test | 1 skipped)",
		"",
		" Test Files  1 skipped (1)",
		"      Tests  1 skipped (1)",
	}, "\n"))
	task := types.CoverageTestTask{
		ID:              "vitest-internal-1",
		Framework:       "vitest",
		File:            source,
		Target:          "LocalCache.get",
		Kind:            "method",
		LineRange:       "3-3",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: !this.values[key]"},
		SuggestedInputs: []string{"构造满足条件 `!this.values[key]` 的输入"},
		TestFile:        filepath.Join(dir, "cache.test.js"),
		TestName:        "covers LocalCache internal get",
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		Framework:    "vitest",
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}
	if out.Status != "passed" || out.Action != "manual_review_internal" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.Metadata["internal_symbol"] != true {
		t.Fatalf("expected internal symbol metadata, got %+v", out.Metadata)
	}
	reason, _ := out.Metadata["internal_reason"].(string)
	if !strings.Contains(reason, "LocalCache.get") || !strings.Contains(reason, "not exported") {
		t.Fatalf("unexpected internal reason: %q", reason)
	}
	if out.Generated == nil || !strings.Contains(out.Generated.Preview, "manual_review_internal: LocalCache") ||
		strings.Contains(out.Generated.Preview, "new LocalCache()") {
		t.Fatalf("expected generated internal review skip, got %+v", out.Generated)
	}
}

func TestHandleValidateCoverageTaskManualReviewInternalFixture(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module","scripts":{"test":"vitest run"},"devDependencies":{"vitest":"^3.0.0"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	source := filepath.Join(dir, "cache.js")
	src := `class LocalCache {
  get(key) {
    if (!this.values[key]) {
      this.values[key] = null
    }
    return this.values[key]
  }
}

export default class CacheFacade {
  constructor() {
    this.cache = new LocalCache()
  }
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	installFakeNpxSuccess(t, strings.Join([]string{
		" RUN  v3.2.4 " + dir,
		"",
		" ↓ cache.test.js (1 test | 1 skipped)",
		"",
		" Test Files  1 skipped (1)",
		"      Tests  1 skipped (1)",
	}, "\n"))
	task := types.CoverageTestTask{
		ID:              "vitest-internal-1",
		Framework:       "vitest",
		File:            source,
		Target:          "LocalCache.get",
		Kind:            "method",
		LineRange:       "3-3",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: !this.values[key]"},
		SuggestedInputs: []string{"构造满足条件 `!this.values[key]` 的输入"},
		TestFile:        filepath.Join(dir, "cache.test.js"),
		TestName:        "covers LocalCache internal get",
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		Framework:    "vitest",
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}

	gotBytes, err := json.MarshalIndent(validateCoverageTaskFixtureFromOutput(t, dir, out), "", "  ")
	if err != nil {
		t.Fatalf("marshal manual-review fixture: %v", err)
	}
	got := string(gotBytes) + "\n"
	wantBytes, err := os.ReadFile(filepath.Join("..", "docs", "fixtures", "validate-coverage-task-manual-review-internal.json"))
	if err != nil {
		t.Fatalf("read manual-review fixture: %v\n--- got ---\n%s", err, got)
	}
	want := string(wantBytes)
	if got != want {
		t.Fatalf("manual-review fixture mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestHandleValidateCoverageTaskApplyFixSuggestionsFixture(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/calc\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "calc.go")
	src := `package calc

func Add(a, b int) int {
	if a == 0 {
		return b
	}
	return a + b
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	legacyTest := `package calc

import "testing"

func TestLegacyAddExpectation(t *testing.T) {
	if got := Add(1, 2); got != 4 {
		t.Fatalf("got %d, want 4", got)
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "legacy_test.go"), []byte(legacyTest), 0o644); err != nil {
		t.Fatalf("write legacy test: %v", err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-apply-fix-1",
		Framework:       "go-test",
		File:            source,
		Target:          "Add",
		Kind:            "function",
		LineRange:       "4-6",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: a == 0"},
		UncoveredLines:  []int{4, 5, 6},
		SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入"},
		Goal:            "为 Add 补充测试，覆盖未执行行段 4-6",
		Command:         "go test ./...",
		TestFile:        filepath.Join(dir, "calc_test.go"),
		TestName:        "TestAdd",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
		Confidence:      0.95,
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}

	gotBytes, err := json.MarshalIndent(validateCoverageTaskFixtureFromOutput(t, dir, out), "", "  ")
	if err != nil {
		t.Fatalf("marshal apply-fix fixture: %v", err)
	}
	got := string(gotBytes) + "\n"
	wantBytes, err := os.ReadFile(filepath.Join("..", "docs", "fixtures", "validate-coverage-task-apply-fix-suggestions.json"))
	if err != nil {
		t.Fatalf("read apply-fix fixture: %v\n--- got ---\n%s", err, got)
	}
	want := string(wantBytes)
	if got != want {
		t.Fatalf("apply-fix fixture mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestCoverageTaskInternalSymbolReasonUsesJavaWording(t *testing.T) {
	reason := coverageTaskInternalSymbolReason(
		&types.CoverageTestTask{
			Framework: "junit",
			File:      "src/main/java/example/Client.java",
			Target:    "Client.hidden",
		},
		&types.GenerateTestsOutput{
			Preview: "manual_review_internal: Client.hidden",
			Context: &types.TestGenerationContext{Framework: "junit"},
		},
		&types.TestResult{Status: "pass"},
	)
	if !strings.Contains(reason, "Client.hidden") ||
		!strings.Contains(reason, "private/internal Java code") ||
		strings.Contains(reason, "JavaScript module") {
		t.Fatalf("unexpected Java internal reason: %q", reason)
	}
}

func TestCoverageTaskInternalSymbolReasonUsesPythonWording(t *testing.T) {
	reason := coverageTaskInternalSymbolReason(
		&types.CoverageTestTask{
			Framework: "pytest",
			File:      "src/private_service.py",
			Target:    "PrivateService.__normalize",
		},
		&types.GenerateTestsOutput{
			Preview: "manual_review_internal: PrivateService.__normalize",
			Context: &types.TestGenerationContext{Framework: "pytest"},
		},
		&types.TestResult{Status: "pass"},
	)
	if !strings.Contains(reason, "PrivateService.__normalize") ||
		!strings.Contains(reason, "private/internal Python code") ||
		strings.Contains(reason, "JavaScript module") {
		t.Fatalf("unexpected Python internal reason: %q", reason)
	}
}

func TestHandleValidateCoverageTaskMarksTypeOnlyTSFileAsNoRuntimeManualReview(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module","scripts":{"test":"vitest run"},"devDependencies":{"vitest":"^3.0.0"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	source := filepath.Join(dir, "events.ts")
	src := `export type ThreadStartedEvent = {
  type: "thread.started";
  thread_id: string;
};

export type ThreadEvent = ThreadStartedEvent;
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	installFakeNpxSuccess(t, strings.Join([]string{
		" RUN  v3.2.4 " + dir,
		"",
		" ↓ events.test.ts (1 test | 1 skipped)",
		"",
		" Test Files  1 skipped (1)",
		"      Tests  1 skipped (1)",
	}, "\n"))
	task := types.CoverageTestTask{
		ID:             "vitest-no-runtime-1",
		Framework:      "vitest",
		File:           source,
		Target:         "events.ts",
		Kind:           "file_level",
		LineRange:      "entire file",
		GapType:        "no_runtime",
		Goal:           "确认 events.ts 是 TypeScript 纯类型文件，没有可直接执行的运行时代码覆盖任务",
		Command:        "npx vitest run events.ts",
		TestFile:       filepath.Join(dir, "events.test.ts"),
		TestName:       "marks type-only module as no runtime coverage",
		AssertionFocus: []string{"通过消费方运行时测试或类型检查验证"},
		Confidence:     0.9,
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		Framework:    "vitest",
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}
	if out.Status != "passed" || out.Action != "manual_review_no_runtime" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.Metadata["no_runtime"] != true {
		t.Fatalf("expected no runtime metadata, got %+v", out.Metadata)
	}
	reason, _ := out.Metadata["no_runtime_reason"].(string)
	if !strings.Contains(reason, "events.ts") || !strings.Contains(reason, "no runtime JavaScript statements") {
		t.Fatalf("unexpected no runtime reason: %q", reason)
	}
	if out.Generated == nil || !strings.Contains(out.Generated.Preview, "manual_review_no_runtime:") ||
		strings.Contains(out.Generated.Preview, "manual_review_internal:") {
		t.Fatalf("expected generated no-runtime review skip, got %+v", out.Generated)
	}
}

func installFakeNpxSuccess(t *testing.T, output string) {
	t.Helper()
	fakeBin := t.TempDir()
	script := "#!/usr/bin/env sh\ncat <<'NPX_OUTPUT'\n" + output + "\nNPX_OUTPUT\nexit 0\n"
	path := filepath.Join(fakeBin, "npx")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake npx: %v", err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func installFakeNpxFailure(t *testing.T, output string) {
	t.Helper()
	fakeBin := t.TempDir()
	script := "#!/usr/bin/env sh\ncat <<'NPX_OUTPUT'\n" + output + "\nNPX_OUTPUT\nexit 1\n"
	path := filepath.Join(fakeBin, "npx")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake npx: %v", err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func installFakeMvnSuccessWithJaCoCo(t *testing.T, report string) {
	t.Helper()
	fakeBin := t.TempDir()
	script := "#!/usr/bin/env sh\n" +
		"mkdir -p target/site/jacoco\n" +
		"cat > target/site/jacoco/jacoco.xml <<'JACOCO_XML'\n" + report + "\nJACOCO_XML\n" +
		"cat <<'MAVEN_OUTPUT'\n" +
		"[INFO] Results:\n" +
		"[INFO] Tests run: 1, Failures: 0, Errors: 0, Skipped: 0\n" +
		"[INFO] BUILD SUCCESS\n" +
		"MAVEN_OUTPUT\n" +
		"exit 0\n"
	path := filepath.Join(fakeBin, "mvn")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake mvn: %v", err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestHandleValidateCoverageTaskReportsGenerationError(t *testing.T) {
	task := types.CoverageTestTask{
		ID:        "go-test-1",
		Framework: "go-test",
		File:      "missing.go",
		Target:    "Missing",
		LineRange: "1-1",
	}

	result, structured, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     "missing.go",
		CoverageTask: &task,
	})
	if err != nil {
		t.Fatalf("HandleValidateCoverageTask returned protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("result.IsError = false, output: %s", resultText(t, result))
	}
	var out types.CoverageTaskValidationOutput
	if err := json.Unmarshal([]byte(resultText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal validation output: %v", err)
	}
	if out.Status != "generation_error" || out.Action != "inspect_generation_error" {
		t.Fatalf("unexpected validation error output: %+v", out)
	}
	if !strings.Contains(out.Error, "文件不存在") {
		t.Fatalf("expected missing file error, got %q", out.Error)
	}
	structuredOutput, ok := structured.(types.CoverageTaskValidationOutput)
	if !ok || structuredOutput.Status != "generation_error" {
		t.Fatalf("structured output = %#v, want generation_error validation output", structured)
	}
}

func TestCoverageTaskUnreachableReasonRecognizesGeneratedMarker(t *testing.T) {
	task := &types.CoverageTestTask{
		Framework: "junit",
		File:      "MatchRatingApproachEncoder.java",
		Target:    "MatchRatingApproachEncoder.encode",
	}
	generated := &types.GenerateTestsOutput{
		Preview: `org.junit.jupiter.api.Assumptions.assumeTrue(false, "manual_review_unreachable: MatchRatingApproachEncoder.encode");`,
	}
	result := &types.TestResult{
		Status:  "pass",
		Skipped: 1,
	}

	reason := coverageTaskUnreachableReason(task, generated, result)
	if !strings.Contains(reason, "MatchRatingApproachEncoder.encode appears unreachable") {
		t.Fatalf("unexpected unreachable reason: %q", reason)
	}
}
