package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/internal/coverage"
	"github.com/sleticalboy/testloop-mcp/types"
)

type validateCoverageTaskInput struct {
	FilePath              string                  `json:"file_path" jsonschema:"源文件路径，例如 internal/calc/calc.go，必填"`
	Framework             string                  `json:"framework,omitempty" jsonschema:"测试框架，可选值: go-test/cargo-test/jest/vitest/mocha/node-test/pytest/junit，默认使用 coverage_task.framework 或自动检测"`
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
	status := coverageTaskValidationStatus(runResult)
	if metadata["unreachable"] == true {
		action = "manual_review_unreachable"
	} else if metadata["environment_dependent"] == true {
		action = "manual_review_environment"
	} else if metadata["protocol_dependent"] == true {
		action = "manual_review_protocol"
	} else if metadata["database_dependent"] == true {
		action = "manual_review_database"
	} else if metadata["external_service_dependent"] == true {
		action = "manual_review_external_service"
	} else if metadata["private_method"] == true {
		action = "manual_review_private"
	} else if metadata["internal_symbol"] == true {
		action = "manual_review_internal"
	} else if metadata["no_runtime"] == true {
		action = "manual_review_no_runtime"
	}
	if action == "ready" && metadata["coverage_target_hit"] == false {
		action = "needs_better_input"
		status = "failed"
	}
	out := types.CoverageTaskValidationOutput{
		Status:       status,
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
	if hit, reportPath, hitLines, missedLines, ok := coverageTaskTargetLineHit(framework, generated, result, task); ok {
		metadata["coverage_report"] = reportPath
		metadata["coverage_target_lines"] = append(append([]int{}, hitLines...), missedLines...)
		metadata["coverage_hit_lines"] = hitLines
		metadata["coverage_missed_lines"] = missedLines
		metadata["coverage_target_hit"] = hit
		if !hit {
			metadata["coverage_miss_reason"] = fmt.Sprintf("%s did not cover target line range %s; generate stronger inputs or cover the target through a better public entry point", strings.TrimSpace(task.Target), strings.TrimSpace(task.LineRange))
		}
	}
	if reason := coverageTaskUnreachableReason(task, generated, result); reason != "" {
		metadata["unreachable"] = true
		metadata["unreachable_reason"] = reason
	}
	if reason := coverageTaskEnvironmentReason(task, generated, result); reason != "" {
		metadata["environment_dependent"] = true
		metadata["environment_reason"] = reason
	}
	if reason := coverageTaskProtocolReason(task, generated, result); reason != "" {
		metadata["protocol_dependent"] = true
		metadata["protocol_reason"] = reason
	}
	if reason := coverageTaskDatabaseReason(task, generated, result); reason != "" {
		metadata["database_dependent"] = true
		metadata["database_reason"] = reason
	}
	if reason := coverageTaskExternalServiceReason(task, generated, result); reason != "" {
		metadata["external_service_dependent"] = true
		metadata["external_service_reason"] = reason
	}
	if reason := coverageTaskPrivateMethodReason(task, generated, result); reason != "" {
		metadata["private_method"] = true
		metadata["private_reason"] = reason
		if candidates := coverageTaskPrivatePublicEntries(generated); len(candidates) > 0 {
			metadata["public_entry_candidates"] = candidates
		}
	}
	if reason := coverageTaskInternalSymbolReason(task, generated, result); reason != "" {
		metadata["internal_symbol"] = true
		metadata["internal_reason"] = reason
	}
	if reason := coverageTaskNoRuntimeReason(task, generated, result); reason != "" {
		metadata["no_runtime"] = true
		metadata["no_runtime_reason"] = reason
	}
	return metadata
}

func coverageTaskTargetLineHit(framework string, generated *types.GenerateTestsOutput, result *types.TestResult, task *types.CoverageTestTask) (bool, string, []int, []int, bool) {
	framework = strings.ToLower(strings.TrimSpace(framework))
	if (framework != "junit" && framework != "node-test" && framework != "pytest") || generated == nil || result == nil || result.Status != "pass" || task == nil {
		return false, "", nil, nil, false
	}
	start, end, ok := coverageTaskLineRange(task.LineRange)
	if !ok {
		return false, "", nil, nil, false
	}
	if framework == "node-test" {
		return nodeCoverageTaskTargetLineHit(result, task, start, end)
	}
	if framework == "pytest" {
		report, reportPath, ok := pytestCoverageReportForValidation(generated.TestFile, task.File)
		if !ok {
			return false, "", nil, nil, false
		}
		return coverageReportTaskTargetLineHit(report, reportPath, task, start, end)
	}
	report, reportPath, ok := javaCoverageReportForValidation(generated.TestFile, task.File)
	if !ok {
		return false, "", nil, nil, false
	}
	return coverageReportTaskTargetLineHit(report, reportPath, task, start, end)
}

func coverageReportTaskTargetLineHit(report *types.CoverageReport, reportPath string, task *types.CoverageTestTask, start int, end int) (bool, string, []int, []int, bool) {
	file := coverageTaskFindCoverageFile(report, task.File)
	if file == nil {
		return false, reportPath, nil, coverageTaskLineNumbers(start, end), true
	}
	coveredByLine := make(map[int]bool, len(file.Blocks))
	for _, block := range file.Blocks {
		for line := block.StartLine; line <= block.EndLine; line++ {
			coveredByLine[line] = block.Covered
		}
	}
	var hitLines []int
	var missedLines []int
	for line := start; line <= end; line++ {
		if coveredByLine[line] {
			hitLines = append(hitLines, line)
		} else {
			missedLines = append(missedLines, line)
		}
	}
	return len(missedLines) == 0, reportPath, hitLines, missedLines, true
}

func nodeCoverageTaskTargetLineHit(result *types.TestResult, task *types.CoverageTestTask, start int, end int) (bool, string, []int, []int, bool) {
	if result == nil || strings.TrimSpace(result.RawOutput) == "" {
		return false, "", nil, nil, false
	}
	report, err := coverage.ParseNodeTestCoverage(result.RawOutput)
	if err != nil {
		return false, "", nil, nil, false
	}
	file := coverageTaskFindCoverageFile(report, task.File)
	if file == nil {
		return false, "node-test raw_output", nil, coverageTaskLineNumbers(start, end), true
	}
	missedByLine := make(map[int]bool)
	for _, block := range file.Blocks {
		if block.Covered {
			continue
		}
		for line := block.StartLine; line <= block.EndLine; line++ {
			missedByLine[line] = true
		}
	}
	var hitLines []int
	var missedLines []int
	for line := start; line <= end; line++ {
		if missedByLine[line] {
			missedLines = append(missedLines, line)
		} else {
			hitLines = append(hitLines, line)
		}
	}
	return len(missedLines) == 0, "node-test raw_output", hitLines, missedLines, true
}

func javaCoverageReportForValidation(paths ...string) (*types.CoverageReport, string, bool) {
	root := ""
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		root = findProjectRoot(path, "pom.xml", "build.gradle", "build.gradle.kts")
		break
	}
	if root == "" {
		return nil, "", false
	}
	for _, reportPath := range []string{
		filepath.Join(root, "target", "site", "jacoco", "jacoco.xml"),
		filepath.Join(root, "build", "reports", "jacoco", "test", "jacocoTestReport.xml"),
	} {
		if !fileExists(reportPath) {
			continue
		}
		report, err := coverage.ParseJaCoCoCoverage(reportPath)
		if err != nil {
			continue
		}
		return report, reportPath, true
	}
	return nil, "", false
}

func pytestCoverageReportForValidation(paths ...string) (*types.CoverageReport, string, bool) {
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		root := findPytestProjectRoot(path)
		reportPath := filepath.Join(root, "coverage.json")
		if !fileExists(reportPath) {
			continue
		}
		report, err := coverage.ParsePytestCoverage(reportPath)
		if err != nil {
			continue
		}
		return report, reportPath, true
	}
	return nil, "", false
}

func coverageTaskFindCoverageFile(report *types.CoverageReport, taskFile string) *types.CoverageFile {
	if report == nil {
		return nil
	}
	normalizedTaskFile := filepath.ToSlash(strings.TrimSpace(taskFile))
	if idx := strings.LastIndex(normalizedTaskFile, "/src/main/java/"); idx >= 0 {
		normalizedTaskFile = normalizedTaskFile[idx+len("/src/main/java/"):]
	}
	for i := range report.Files {
		path := filepath.ToSlash(strings.TrimSpace(report.Files[i].Path))
		if path == normalizedTaskFile || strings.HasSuffix(normalizedTaskFile, "/"+path) || strings.HasSuffix(path, "/"+normalizedTaskFile) {
			return &report.Files[i]
		}
	}
	return nil
}

func coverageTaskLineRange(raw string) (int, int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "entire file") {
		return 0, 0, false
	}
	parts := strings.SplitN(raw, "-", 2)
	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || start <= 0 {
		return 0, 0, false
	}
	end := start
	if len(parts) == 2 {
		parsedEnd, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || parsedEnd <= 0 {
			return 0, 0, false
		}
		end = parsedEnd
	}
	if end < start {
		start, end = end, start
	}
	return start, end, true
}

func coverageTaskLineNumbers(start, end int) []int {
	if start <= 0 || end < start {
		return nil
	}
	lines := make([]int, 0, end-start+1)
	for line := start; line <= end; line++ {
		lines = append(lines, line)
	}
	return lines
}

func coverageTaskUnreachableReason(task *types.CoverageTestTask, generated *types.GenerateTestsOutput, result *types.TestResult) string {
	if task == nil || generated == nil || result == nil || result.Status != "pass" || result.Skipped == 0 {
		return ""
	}
	if strings.Contains(generated.Preview, "manual_review_unreachable:") {
		target := strings.TrimSpace(task.Target)
		if target == "" {
			target = "coverage task"
		}
		return fmt.Sprintf("%s appears unreachable through the generated public test path; review the coverage report or source branch manually", target)
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
	if strings.Contains(generated.Preview, "manual_review_environment:") {
		target := strings.TrimSpace(task.Target)
		if target == "" {
			target = "coverage task"
		}
		return fmt.Sprintf("%s depends on OS/runtime environment state; use injection or an integration environment instead of treating the generated skip as directly ready", target)
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

func coverageTaskProtocolReason(task *types.CoverageTestTask, generated *types.GenerateTestsOutput, result *types.TestResult) string {
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
	if !strings.Contains(hints, "err != nil") {
		return ""
	}
	switch task.Target {
	case "RunClient", "QueryStatus", "SendControl":
		return fmt.Sprintf("%s protocol error branch depends on socket write or streaming I/O failure; generate a deterministic fake connection or review manually instead of relying on connection timing", task.Target)
	default:
		return ""
	}
}

func coverageTaskDatabaseReason(task *types.CoverageTestTask, generated *types.GenerateTestsOutput, result *types.TestResult) string {
	if task == nil || generated == nil || result == nil || result.Status != "pass" || result.Skipped == 0 {
		return ""
	}
	if strings.Contains(generated.Preview, "manual_review_database:") {
		target := strings.TrimSpace(task.Target)
		if target == "" {
			target = "coverage task"
		}
		return fmt.Sprintf("%s depends on database transaction/session behavior; use a deterministic test database, injected repository/session, or integration fixture instead of treating the generated skip as directly ready", target)
	}
	if !strings.Contains(generated.Preview, `t.Skip("TODO: fill in meaningful test inputs and expected values")`) {
		return ""
	}
	hintsList := make([]string, 0, len(task.MissingBranches)+len(task.SuggestedInputs)+len(task.AssertionFocus))
	hintsList = append(hintsList, task.MissingBranches...)
	hintsList = append(hintsList, task.SuggestedInputs...)
	hintsList = append(hintsList, task.AssertionFocus...)
	hints := strings.Join(hintsList, " ")
	if !strings.Contains(hints, "err != nil") && !strings.Contains(hints, "Error != nil") && !strings.Contains(hints, "RowsAffected") && !strings.Contains(hints, "gorm") {
		return ""
	}
	if strings.Contains(task.Target, "Repo.") || strings.Contains(generated.Preview, "*Repo") || strings.Contains(task.File, "/repo/") || strings.Contains(task.File, "\\repo\\") {
		return fmt.Sprintf("%s database branch depends on GORM DB behavior; generate a deterministic test database or inject a fake repository/DB instead of adding third-party test dependencies implicitly", task.Target)
	}
	return ""
}

func coverageTaskExternalServiceReason(task *types.CoverageTestTask, generated *types.GenerateTestsOutput, result *types.TestResult) string {
	if task == nil || generated == nil || result == nil || result.Status != "fail" {
		return ""
	}
	if !coverageTaskRunLooksExternalServiceDependent(result) {
		return ""
	}
	target := strings.TrimSpace(task.Target)
	hintsList := make([]string, 0, len(task.MissingBranches)+len(task.SuggestedInputs)+len(task.AssertionFocus)+2)
	hintsList = append(hintsList, task.MissingBranches...)
	hintsList = append(hintsList, task.SuggestedInputs...)
	hintsList = append(hintsList, task.AssertionFocus...)
	hintsList = append(hintsList, target, generated.Preview)
	hints := strings.ToLower(strings.Join(hintsList, " "))
	if !coverageTaskLooksExternalServiceTarget(target, hints) {
		return ""
	}
	if target == "" {
		target = "coverage task"
	}
	return fmt.Sprintf("%s depends on a live RPC/external service, route state, or long retry timing; validate it with injected fake clients/route data or an integration environment instead of treating the generated unit test as directly repairable", target)
}

func coverageTaskRunLooksExternalServiceDependent(result *types.TestResult) bool {
	if result == nil {
		return false
	}
	parts := []string{result.RawOutput}
	for _, failure := range result.Failures {
		parts = append(parts, failure.Error)
	}
	text := strings.ToLower(strings.Join(parts, "\n"))
	for _, marker := range []string{
		"timeout", "timed out", "deadline exceeded", "context deadline",
		"signal: killed", "econnrefused", "connection refused",
		"unavailable", "grpc", "too many requests",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func coverageTaskLooksExternalServiceTarget(target string, hints string) bool {
	targetLower := strings.ToLower(strings.TrimSpace(target))
	if strings.Contains(targetLower, "producer.") {
		return true
	}
	for _, marker := range []string{
		"rpcclient", "rpc client", "grpc", "endpoint", "endpoints",
		"sendmessage", "send message", ".send(", "endtransaction",
		"recoverorphanedtransaction", "route data",
	} {
		if strings.Contains(hints, marker) {
			return true
		}
	}
	return false
}

func coverageTaskPrivateMethodReason(task *types.CoverageTestTask, generated *types.GenerateTestsOutput, result *types.TestResult) string {
	if task == nil || generated == nil || result == nil {
		return ""
	}
	target := strings.TrimSpace(task.Target)
	if !strings.Contains(target, ".#") && !strings.Contains(generated.Preview, "instance.#") {
		return ""
	}
	if strings.Contains(generated.Preview, "manual_review_private:") && result.Status == "pass" {
		return fmt.Sprintf("%s is a JavaScript private method; cover it through a public entry point or review manually instead of calling it directly", target)
	}
	if result.Status != "fail" {
		return ""
	}
	if !strings.Contains(result.RawOutput, "Private field") && !strings.Contains(result.RawOutput, "private field") {
		return ""
	}
	return fmt.Sprintf("%s is a JavaScript private method; generate a public-entry test or review manually instead of calling it directly", target)
}

func coverageTaskPrivatePublicEntries(generated *types.GenerateTestsOutput) []string {
	if generated == nil {
		return nil
	}
	const marker = "public_entry_candidates:"
	for _, line := range strings.Split(generated.Preview, "\n") {
		idx := strings.Index(line, marker)
		if idx < 0 {
			continue
		}
		raw := strings.TrimSpace(line[idx+len(marker):])
		if raw == "" || strings.HasPrefix(raw, "none detected") {
			return nil
		}
		parts := strings.Split(raw, ",")
		entries := make([]string, 0, len(parts))
		for _, part := range parts {
			entry := strings.TrimSpace(part)
			if entry != "" {
				entries = append(entries, entry)
			}
		}
		return entries
	}
	return nil
}

func coverageTaskInternalSymbolReason(task *types.CoverageTestTask, generated *types.GenerateTestsOutput, result *types.TestResult) string {
	if task == nil || generated == nil || result == nil {
		return ""
	}
	if !strings.Contains(generated.Preview, "manual_review_internal:") || result.Status != "pass" {
		return ""
	}
	target := strings.TrimSpace(task.Target)
	if target == "" {
		target = "target"
	}
	contextFramework := ""
	if generated.Context != nil {
		contextFramework = generated.Context.Framework
	}
	framework := strings.ToLower(strings.TrimSpace(firstNonEmpty(task.Framework, contextFramework)))
	if framework == "junit" || strings.HasSuffix(strings.ToLower(task.File), ".java") {
		return fmt.Sprintf("%s is private/internal Java code; cover it through a visible public or package entry point, add a test seam, or review manually", target)
	}
	if framework == "pytest" || strings.HasSuffix(strings.ToLower(task.File), ".py") {
		return fmt.Sprintf("%s is private/internal Python code; cover it through a public method, add a test seam, or review manually", target)
	}
	return fmt.Sprintf("%s is not exported from this JavaScript module; cover it through an exported API, add a test seam, or review manually", target)
}

func coverageTaskNoRuntimeReason(task *types.CoverageTestTask, generated *types.GenerateTestsOutput, result *types.TestResult) string {
	if task == nil || generated == nil || result == nil {
		return ""
	}
	if !strings.Contains(generated.Preview, "manual_review_no_runtime:") || result.Status != "pass" {
		return ""
	}
	target := strings.TrimSpace(task.Target)
	if target == "" {
		target = "file"
	}
	return fmt.Sprintf("%s has no runtime JavaScript statements to execute; cover behavior through runtime consumers or type-checking instead of generating a unit test for the type-only module", target)
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
	collectCoverage := input.Coverage || strings.ToLower(strings.TrimSpace(framework)) == "junit"
	if timeout := validateCoverageTaskTimeout(); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	result, _, err := HandleRunTests(ctx, nil, runTestsInput{
		Path:                  generated.TestFile,
		Framework:             framework,
		Coverage:              collectCoverage,
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

func validateCoverageTaskTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("TESTLOOP_VALIDATE_TASK_TIMEOUT_SECONDS"))
	if raw == "" {
		return 0
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func coverageTaskValidationResult(out types.CoverageTaskValidationOutput) (*mcp.CallToolResult, any, error) {
	result, err := structuredToolResultWithError(out, out.Status == "generation_error" || out.Status == "run_error")
	if err != nil {
		return nil, nil, err
	}
	return result, out, nil
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
		if result.Action == "manual_review" {
			return "manual_review"
		}
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
