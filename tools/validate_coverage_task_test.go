package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sleticalboy/testloop-mcp/types"
)

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
