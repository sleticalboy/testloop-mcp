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
