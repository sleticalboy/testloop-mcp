package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/types"
)

type validateCoverageTaskInput struct {
	FilePath              string                  `json:"file_path" jsonschema:"源文件路径，例如 internal/calc/calc.go，必填"`
	Framework             string                  `json:"framework,omitempty" jsonschema:"测试框架，可选值: go-test/cargo-test/jest/vitest/mocha/pytest/junit，默认使用 coverage_task.framework 或自动检测"`
	Provider              string                  `json:"provider,omitempty" jsonschema:"测试生成 provider: static、llm 或 auto，默认 static"`
	CoverageTask          *types.CoverageTestTask `json:"coverage_task" jsonschema:"parse_coverage 返回的单个 test_tasks 项，必填"`
	Coverage              bool                    `json:"coverage,omitempty" jsonschema:"执行测试时是否收集覆盖率，默认 false"`
	IncludeFixSuggestions *bool                   `json:"include_fix_suggestions,omitempty" jsonschema:"测试失败时是否附带 fix_suggestions 摘要，默认 true"`
}

func HandleValidateCoverageTask(ctx context.Context, req *mcp.CallToolRequest, input validateCoverageTaskInput) (*mcp.CallToolResult, any, error) {
	if input.FilePath == "" {
		return nil, nil, fmt.Errorf("file_path 参数必填")
	}
	if input.CoverageTask == nil {
		return nil, nil, fmt.Errorf("coverage_task 参数必填")
	}
	framework := firstNonEmpty(input.Framework, input.CoverageTask.Framework)

	generated, err := validateCoverageTaskGenerate(ctx, input, framework)
	if err != nil {
		out := types.CoverageTaskValidationOutput{
			Status:       "generation_error",
			Action:       "inspect_generation_error",
			CoverageTask: input.CoverageTask,
			Error:        err.Error(),
		}
		return coverageTaskValidationResult(out)
	}
	if generated.Status != "ok" {
		out := types.CoverageTaskValidationOutput{
			Status:        "generation_error",
			Action:        coverageTaskGenerationAction(generated),
			CoverageTask:  validationCoverageTask(input.CoverageTask, generated),
			Generated:     generated,
			Error:         generated.Error,
			ProviderError: generated.ProviderError,
		}
		return coverageTaskValidationResult(out)
	}

	runResult, err := validateCoverageTaskRun(ctx, input, framework, generated)
	if err != nil {
		out := types.CoverageTaskValidationOutput{
			Status:       "run_error",
			Action:       "inspect_test_runner",
			CoverageTask: validationCoverageTask(input.CoverageTask, generated),
			Generated:    generated,
			Error:        err.Error(),
		}
		return coverageTaskValidationResult(out)
	}

	coverageTask := validationCoverageTask(input.CoverageTask, generated)
	metadata := coverageTaskValidationMetadata(framework, generated, runResult, coverageTask)
	action := coverageTaskValidationAction(runResult)
	if metadata["unreachable"] == true {
		action = "manual_review_unreachable"
	} else if metadata["environment_dependent"] == true {
		action = "manual_review_environment"
	}
	out := types.CoverageTaskValidationOutput{
		Status:       coverageTaskValidationStatus(runResult),
		Action:       action,
		CoverageTask: coverageTask,
		Generated:    generated,
		RunResult:    runResult,
		Metadata:     metadata,
	}
	return coverageTaskValidationResult(out)
}

func coverageTaskValidationMetadata(framework string, generated *types.GenerateTestsOutput, result *types.TestResult, task *types.CoverageTestTask) map[string]any {
	metadata := map[string]any{
		"framework": framework,
	}
	if generated != nil {
		metadata["test_file"] = generated.TestFile
	}
	if reason := coverageTaskUnreachableReason(task, generated, result); reason != "" {
		metadata["unreachable"] = true
		metadata["unreachable_reason"] = reason
	}
	if reason := coverageTaskEnvironmentReason(task, generated, result); reason != "" {
		metadata["environment_dependent"] = true
		metadata["environment_reason"] = reason
	}
	return metadata
}

func coverageTaskUnreachableReason(task *types.CoverageTestTask, generated *types.GenerateTestsOutput, result *types.TestResult) string {
	if task == nil || generated == nil || result == nil || result.Status != "pass" || result.Skipped == 0 {
		return ""
	}
	if task.Target == "init" && strings.Contains(generated.Preview, `t.Skip("init functions cannot be called directly; review package initialization manually")`) {
		return "Go init functions cannot be called directly from tests; review package initialization side effects manually"
	}
	if !strings.Contains(generated.Preview, `t.Skip("TODO: fill in meaningful test inputs and expected values")`) {
		return ""
	}
	hintsList := make([]string, 0, len(task.MissingBranches)+len(task.SuggestedInputs)+len(task.AssertionFocus))
	hintsList = append(hintsList, task.MissingBranches...)
	hintsList = append(hintsList, task.SuggestedInputs...)
	hintsList = append(hintsList, task.AssertionFocus...)
	hints := strings.Join(hintsList, " ")
	if task.Target == "RemoteIP" && strings.Contains(hints, "partIndex < 0") {
		return `branch "partIndex < 0" appears unreachable because partIndex is derived from a non-empty X-Forwarded-For parts slice`
	}
	return ""
}

func coverageTaskEnvironmentReason(task *types.CoverageTestTask, generated *types.GenerateTestsOutput, result *types.TestResult) string {
	if task == nil || generated == nil || result == nil || result.Status != "pass" || result.Skipped == 0 {
		return ""
	}
	if !strings.Contains(generated.Preview, `t.Skip("TODO: fill in meaningful test inputs and expected values")`) {
		return ""
	}
	hintsList := make([]string, 0, len(task.MissingBranches)+len(task.SuggestedInputs)+len(task.AssertionFocus))
	hintsList = append(hintsList, task.MissingBranches...)
	hintsList = append(hintsList, task.SuggestedInputs...)
	hintsList = append(hintsList, task.AssertionFocus...)
	hints := strings.Join(hintsList, " ")
	switch task.Target {
	case "InitDisk":
		if strings.Contains(hints, "err != nil") {
			return "InitDisk error branch depends on disk.Usage(\"/\") returning an OS/runtime error; static tests cannot force it without dependency injection"
		}
	case "InitCPU":
		if strings.Contains(hints, "err != nil") {
			return "InitCPU error branch depends on gopsutil cpu.Counts or cpu.Percent returning an OS/runtime error; static tests cannot force it without dependency injection"
		}
	case "InitRAM":
		if strings.Contains(hints, "err != nil") {
			return "InitRAM error branch depends on mem.VirtualMemory returning an OS/runtime error; static tests cannot force it without dependency injection"
		}
	}
	return ""
}

func validationCoverageTask(input *types.CoverageTestTask, generated *types.GenerateTestsOutput) *types.CoverageTestTask {
	if generated != nil && generated.CoverageTask != nil {
		return generated.CoverageTask
	}
	return input
}

func validateCoverageTaskGenerate(ctx context.Context, input validateCoverageTaskInput, framework string) (*types.GenerateTestsOutput, error) {
	result, _, err := HandleGenerateTests(ctx, nil, generateTestsInput{
		FilePath:     input.FilePath,
		Framework:    framework,
		Provider:     input.Provider,
		CoverageTask: input.CoverageTask,
	})
	if err != nil {
		return nil, err
	}
	var out types.GenerateTestsOutput
	if err := decodeToolResult(result, &out); err != nil {
		return nil, fmt.Errorf("解析 generate_tests 输出失败: %w", err)
	}
	return &out, nil
}

func validateCoverageTaskRun(ctx context.Context, input validateCoverageTaskInput, framework string, generated *types.GenerateTestsOutput) (*types.TestResult, error) {
	includeFixSuggestions := true
	if input.IncludeFixSuggestions != nil {
		includeFixSuggestions = *input.IncludeFixSuggestions
	}
	result, _, err := HandleRunTests(ctx, nil, runTestsInput{
		Path:                  generated.TestFile,
		Framework:             framework,
		Coverage:              input.Coverage,
		IncludeFixSuggestions: includeFixSuggestions,
		SourceCode:            input.FilePath,
		TestCode:              generated.TestFile,
	})
	if err != nil {
		return nil, err
	}
	var out types.TestResult
	if err := decodeToolResult(result, &out); err != nil {
		return nil, fmt.Errorf("解析 run_tests 输出失败: %w", err)
	}
	return &out, nil
}

func coverageTaskValidationResult(out types.CoverageTaskValidationOutput) (*mcp.CallToolResult, any, error) {
	resultJSON, _ := json.Marshal(out)
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
		StructuredContent: out,
		IsError:           out.Status == "generation_error" || out.Status == "run_error",
	}, out, nil
}

func decodeToolResult(result *mcp.CallToolResult, out any) error {
	if result == nil || len(result.Content) == 0 {
		return fmt.Errorf("工具输出为空")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return fmt.Errorf("工具输出不是文本内容: %T", result.Content[0])
	}
	if err := json.Unmarshal([]byte(text.Text), out); err != nil {
		return err
	}
	return nil
}

func coverageTaskGenerationAction(generated *types.GenerateTestsOutput) string {
	if generated != nil && generated.ProviderError != nil {
		return generated.ProviderError.Action
	}
	return "inspect_generation_error"
}

func coverageTaskValidationStatus(result *types.TestResult) string {
	if result == nil {
		return "run_error"
	}
	if result.Status == "pass" {
		return "passed"
	}
	return "failed"
}

func coverageTaskValidationAction(result *types.TestResult) string {
	if result == nil {
		return "inspect_test_runner"
	}
	if result.Status == "pass" {
		return "ready"
	}
	if len(result.FixSuggestions) > 0 {
		return "apply_fix_suggestions"
	}
	if len(result.Failures) > 0 {
		return "repair_generated_test"
	}
	return "manual_review"
}
