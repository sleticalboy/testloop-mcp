package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sleticalboy/testloop-mcp/types"
)

func TestValidateRustCoverageTopTasks(t *testing.T) {
	projectDir := os.Getenv("TESTLOOP_VALIDATE_RUST_PROJECT_DIR")
	if projectDir == "" {
		t.Skip("TESTLOOP_VALIDATE_RUST_PROJECT_DIR is not set")
	}
	limit := envPositiveInt(t, "TESTLOOP_VALIDATE_RUST_TASK_LIMIT", 20)
	stageTimeout := envOptionalDurationSeconds(t, "TESTLOOP_VALIDATE_RUST_STAGE_TIMEOUT_SECONDS")
	baselineTimeout := firstNonZeroDuration(envOptionalDurationSeconds(t, "TESTLOOP_VALIDATE_RUST_BASELINE_TIMEOUT_SECONDS"), stageTimeout)
	taskTimeout := firstNonZeroDuration(envOptionalDurationSeconds(t, "TESTLOOP_VALIDATE_RUST_TASK_TIMEOUT_SECONDS"), stageTimeout)
	outputPath := os.Getenv("TESTLOOP_VALIDATE_RUST_OUTPUT")
	if outputPath == "" {
		outputPath = filepath.Join(os.TempDir(), fmt.Sprintf("testloop-rust-coverage-top%d-%s.jsonl", limit, time.Now().Format("20060102150405")))
	}

	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		t.Fatalf("resolve project dir: %v", err)
	}
	baselineRoot := filepath.Join(t.TempDir(), "baseline")
	logRustValidationStage(t, "baseline.copy.start project=%s dest=%s", projectDir, baselineRoot)
	if err := copyRustProjectTree(projectDir, baselineRoot); err != nil {
		t.Fatalf("copy baseline project: %v", err)
	}
	logRustValidationStage(t, "baseline.copy.done dest=%s", baselineRoot)

	report := parseRustCoverageReportForProject(t, baselineRoot, baselineTimeout)
	tasks := filterRustCoverageTasks(report.TestTasks, os.Getenv("TESTLOOP_VALIDATE_RUST_FILE_FILTER"))
	if len(tasks) < limit {
		t.Fatalf("coverage tasks after filter = %d, want at least %d", len(tasks), limit)
	}
	logRustValidationStage(t, "tasks.selected count=%d limit=%d filter=%q", len(tasks), limit, os.Getenv("TESTLOOP_VALIDATE_RUST_FILE_FILTER"))

	outFile, err := os.Create(outputPath)
	if err != nil {
		t.Fatalf("create output jsonl: %v", err)
	}
	defer outFile.Close()

	summary := rustProjectValidationSummary{
		Limit:        limit,
		Framework:    "cargo-test",
		StatusCounts: map[string]int{},
		ActionCounts: map[string]int{},
	}
	var failures []string
	for i := 0; i < limit; i++ {
		task := tasks[i]
		taskRoot := filepath.Join(t.TempDir(), fmt.Sprintf("task-%02d", i+1))
		logRustValidationStage(t, "task.copy.start index=%d id=%s target=%s root=%s", i+1, task.ID, task.Target, taskRoot)
		if err := copyRustProjectTree(projectDir, taskRoot); err != nil {
			t.Fatalf("copy task worktree for %s: %v", task.ID, err)
		}
		logRustValidationStage(t, "task.copy.done index=%d id=%s", i+1, task.ID)
		task.File = rewriteRustValidationPath(baselineRoot, taskRoot, task.File)
		task.TestFile = rewriteRustValidationPath(baselineRoot, taskRoot, task.TestFile)

		includeFixSuggestions := false
		ctx := context.Background()
		cancel := func() {}
		if taskTimeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, taskTimeout)
		}
		logRustValidationStage(t, "task.validate.start index=%d id=%s target=%s file=%s timeout=%s", i+1, task.ID, task.Target, task.File, taskTimeout)
		validation, _, err := HandleValidateCoverageTask(ctx, nil, validateCoverageTaskInput{
			FilePath:              task.File,
			Framework:             "cargo-test",
			CoverageTask:          &task,
			Coverage:              true,
			IncludeFixSuggestions: &includeFixSuggestions,
		})
		cancel()
		if err != nil {
			t.Fatalf("validate task %d %s %s: %v", i+1, task.ID, task.Target, err)
		}
		var out types.CoverageTaskValidationOutput
		if err := json.Unmarshal([]byte(resultText(t, validation)), &out); err != nil {
			t.Fatalf("unmarshal validation output for %s: %v", task.ID, err)
		}
		logRustValidationStage(t, "task.validate.done index=%d id=%s target=%s status=%s action=%s", i+1, task.ID, task.Target, out.Status, out.Action)
		encoded, _ := json.Marshal(out)
		if _, err := outFile.Write(append(encoded, '\n')); err != nil {
			t.Fatalf("write output jsonl: %v", err)
		}
		summary.record(i+1, task, out)
		if out.Status != "passed" && !strings.HasPrefix(out.Action, "manual_review_") {
			failures = append(failures, fmt.Sprintf("task %d %s %s status=%s action=%s error=%s", i+1, task.ID, task.Target, out.Status, out.Action, out.Error))
		}
	}

	sort.Strings(summary.SkippedReady)
	summaryJSON, _ := json.Marshal(summary)
	t.Logf("result_jsonl=%s", outputPath)
	t.Logf("summary=%s", summaryJSON)
	if len(failures) > 0 {
		t.Fatalf("validation failures:\n%s", strings.Join(failures, "\n"))
	}
}

type rustProjectValidationSummary struct {
	Limit        int            `json:"limit"`
	Framework    string         `json:"framework"`
	StatusCounts map[string]int `json:"status_counts"`
	ActionCounts map[string]int `json:"action_counts"`
	ZeroSkip     int            `json:"zero_skip"`
	SkippedTotal int            `json:"skipped_total"`
	SkippedReady []string       `json:"skipped_ready,omitempty"`
}

func (s *rustProjectValidationSummary) record(index int, task types.CoverageTestTask, out types.CoverageTaskValidationOutput) {
	s.StatusCounts[out.Status]++
	s.ActionCounts[out.Action]++
	if out.RunResult == nil {
		return
	}
	if out.RunResult.Skipped == 0 {
		s.ZeroSkip++
	}
	s.SkippedTotal += out.RunResult.Skipped
	if out.Action == "ready" && out.RunResult.Skipped > 0 {
		s.SkippedReady = append(s.SkippedReady, fmt.Sprintf("%d %s %s %s", index, task.ID, task.Target, task.LineRange))
	}
}

func parseRustCoverageReportForProject(t *testing.T, projectRoot string, timeout time.Duration) types.CoverageReport {
	t.Helper()
	ctx := context.Background()
	cancel := func() {}
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()
	cmd := rustCoverageCommand(ctx)
	cmd.Dir = projectRoot
	logRustValidationStage(t, "baseline.coverage.start root=%s timeout=%s", projectRoot, timeout)
	output, err := cmd.CombinedOutput()
	logRustValidationStage(t, "baseline.coverage.done root=%s err=%v output_bytes=%d", projectRoot, err, len(output))
	if err != nil {
		t.Fatalf("baseline coverage failed: %v\n%s", err, output)
	}
	coverageFile := os.Getenv("TESTLOOP_VALIDATE_RUST_COVERAGE_FILE")
	if coverageFile == "" {
		coverageFile = filepath.Join("target", "tarpaulin", "lcov.info")
	}
	if !filepath.IsAbs(coverageFile) {
		coverageFile = filepath.Join(projectRoot, coverageFile)
	}
	data, err := os.ReadFile(coverageFile)
	if err != nil {
		t.Fatalf("read Rust LCOV file %s: %v", coverageFile, err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir project root for coverage parsing: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	}()
	result, _, err := HandleParseCoverage(context.Background(), nil, parseCoverageInput{
		Data:      string(data),
		Framework: "cargo-test",
	})
	if err != nil {
		t.Fatalf("parse coverage: %v", err)
	}
	var report types.CoverageReport
	if err := json.Unmarshal([]byte(resultText(t, result)), &report); err != nil {
		t.Fatalf("unmarshal coverage report: %v", err)
	}
	return report
}

func rustCoverageCommand(ctx context.Context) *exec.Cmd {
	template := strings.TrimSpace(os.Getenv("TESTLOOP_VALIDATE_RUST_COVERAGE_COMMAND"))
	if template == "" {
		template = "cargo tarpaulin --out Lcov --output-dir target/tarpaulin"
	}
	return configureCommandProcessGroup(exec.CommandContext(ctx, "sh", "-c", template))
}

func TestRustCoverageCommandKillsProcessGroupOnTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell process group cancellation is only configured on Unix platforms")
	}
	t.Setenv("TESTLOOP_VALIDATE_RUST_COVERAGE_COMMAND", "sh -c 'sleep 5'")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	cmd := rustCoverageCommand(ctx)
	cmd.Dir = t.TempDir()
	_ = cmd.Run()
	elapsed := time.Since(start)
	if elapsed > 3*time.Second {
		t.Fatalf("Rust coverage command timeout took %s, child process likely survived context cancellation", elapsed)
	}
}

func filterRustCoverageTasks(tasks []types.CoverageTestTask, filter string) []types.CoverageTestTask {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return tasks
	}
	filtered := make([]types.CoverageTestTask, 0, len(tasks))
	for _, task := range tasks {
		if strings.Contains(filepath.ToSlash(task.File), filter) {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

func copyRustProjectTree(src string, dst string) error {
	return copyTreeSkipping(src, dst, map[string]bool{
		".git":   true,
		"target": true,
	})
}

func rewriteRustValidationPath(baselineRoot string, taskRoot string, value string) string {
	return rewriteGoValidationPath(baselineRoot, taskRoot, value)
}

func logRustValidationStage(t *testing.T, format string, args ...any) {
	t.Helper()
	message := fmt.Sprintf(format, args...)
	t.Log(message)
	fmt.Fprintf(os.Stderr, "testloop-rust-validation: %s\n", message)
}
