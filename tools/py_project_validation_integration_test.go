package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sleticalboy/testloop-mcp/types"
)

func TestValidatePyCoverageTopTasks(t *testing.T) {
	projectDir := os.Getenv("TESTLOOP_VALIDATE_PY_PROJECT_DIR")
	if projectDir == "" {
		t.Skip("TESTLOOP_VALIDATE_PY_PROJECT_DIR is not set")
	}
	limit := envPositiveInt(t, "TESTLOOP_VALIDATE_PY_TASK_LIMIT", 20)
	stageTimeout := envOptionalDurationSeconds(t, "TESTLOOP_VALIDATE_PY_STAGE_TIMEOUT_SECONDS")
	baselineTimeout := firstNonZeroDuration(envOptionalDurationSeconds(t, "TESTLOOP_VALIDATE_PY_BASELINE_TIMEOUT_SECONDS"), stageTimeout)
	taskTimeout := firstNonZeroDuration(envOptionalDurationSeconds(t, "TESTLOOP_VALIDATE_PY_TASK_TIMEOUT_SECONDS"), stageTimeout)
	outputPath := os.Getenv("TESTLOOP_VALIDATE_PY_OUTPUT")
	if outputPath == "" {
		outputPath = filepath.Join(os.TempDir(), fmt.Sprintf("testloop-py-coverage-top%d-%s.jsonl", limit, time.Now().Format("20060102150405")))
	}

	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		t.Fatalf("resolve project dir: %v", err)
	}
	tasksFile := os.Getenv("TESTLOOP_VALIDATE_PY_TASKS_FILE")
	taskIDFilter := os.Getenv("TESTLOOP_VALIDATE_PY_TASK_IDS")
	baselineRoot := projectDir
	var tasks []types.CoverageTestTask
	if strings.TrimSpace(tasksFile) != "" {
		tasks = readCoverageTasksJSONL(t, tasksFile)
		logPyValidationStage(t, "tasks.file.loaded path=%s count=%d", tasksFile, len(tasks))
	} else {
		baselineRoot = filepath.Join(t.TempDir(), "baseline")
		logPyValidationStage(t, "baseline.copy.start project=%s dest=%s", projectDir, baselineRoot)
		if err := copyPyProjectTree(projectDir, baselineRoot); err != nil {
			t.Fatalf("copy baseline project: %v", err)
		}
		logPyValidationStage(t, "baseline.copy.done dest=%s", baselineRoot)

		report := parsePyCoverageReportForProject(t, baselineRoot, strings.Fields(os.Getenv("TESTLOOP_VALIDATE_PY_TEST_ARGS")), baselineTimeout)
		tasks = report.TestTasks
	}
	tasks = filterCoverageTasksByFileAndIDs(tasks, os.Getenv("TESTLOOP_VALIDATE_PY_FILE_FILTER"), taskIDFilter)
	if len(tasks) == 0 {
		t.Fatalf("coverage tasks after filter = 0, file_filter=%q task_ids=%q tasks_file=%q", os.Getenv("TESTLOOP_VALIDATE_PY_FILE_FILTER"), taskIDFilter, tasksFile)
	}
	if taskIDFilter != "" && len(tasks) < limit {
		limit = len(tasks)
	}
	if len(tasks) < limit {
		t.Fatalf("coverage tasks after filter = %d, want at least %d", len(tasks), limit)
	}
	logPyValidationStage(t, "tasks.selected count=%d limit=%d filter=%q task_ids=%q", len(tasks), limit, os.Getenv("TESTLOOP_VALIDATE_PY_FILE_FILTER"), taskIDFilter)

	outFile, err := os.Create(outputPath)
	if err != nil {
		t.Fatalf("create output jsonl: %v", err)
	}
	defer outFile.Close()

	summary := pyProjectValidationSummary{
		Limit:        limit,
		Framework:    "pytest",
		StatusCounts: map[string]int{},
		ActionCounts: map[string]int{},
	}
	var failures []string
	for i := 0; i < limit; i++ {
		task := tasks[i]
		taskRoot := filepath.Join(t.TempDir(), fmt.Sprintf("task-%02d", i+1))
		logPyValidationStage(t, "task.copy.start index=%d id=%s target=%s root=%s", i+1, task.ID, task.Target, taskRoot)
		if err := copyPyProjectTree(projectDir, taskRoot); err != nil {
			t.Fatalf("copy task worktree for %s: %v", task.ID, err)
		}
		logPyValidationStage(t, "task.copy.done index=%d id=%s", i+1, task.ID)
		task.File = rewritePyValidationPath(baselineRoot, taskRoot, task.File)
		task.TestFile = rewritePyValidationPath(baselineRoot, taskRoot, task.TestFile)

		includeFixSuggestions := false
		ctx := context.Background()
		cancel := func() {}
		if taskTimeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, taskTimeout)
		}
		logPyValidationStage(t, "task.validate.start index=%d id=%s target=%s file=%s timeout=%s", i+1, task.ID, task.Target, task.File, taskTimeout)
		validation, _, err := HandleValidateCoverageTask(ctx, nil, validateCoverageTaskInput{
			FilePath:              task.File,
			Framework:             "pytest",
			CoverageTask:          &task,
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
		logPyValidationStage(t, "task.validate.done index=%d id=%s target=%s status=%s action=%s", i+1, task.ID, task.Target, out.Status, out.Action)
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

func TestPyCoverageCommandSupportsCustomTemplate(t *testing.T) {
	t.Setenv("TESTLOOP_VALIDATE_PY_COVERAGE_COMMAND", "python3 -m pytest --cov=src --cov-report=json {args}")

	cmd := pyCoverageCommand(context.Background(), []string{"tests/test_api.py", "tests/test space.py"})

	got := strings.Join(cmd.Args, " ")
	want := "sh -c python3 -m pytest --cov=src --cov-report=json 'tests/test_api.py' 'tests/test space.py'"
	if got != want {
		t.Fatalf("pyCoverageCommand args = %q, want %q", got, want)
	}
}

type pyProjectValidationSummary struct {
	Limit        int            `json:"limit"`
	Framework    string         `json:"framework"`
	StatusCounts map[string]int `json:"status_counts"`
	ActionCounts map[string]int `json:"action_counts"`
	ZeroSkip     int            `json:"zero_skip"`
	SkippedTotal int            `json:"skipped_total"`
	SkippedReady []string       `json:"skipped_ready,omitempty"`
}

func (s *pyProjectValidationSummary) record(index int, task types.CoverageTestTask, out types.CoverageTaskValidationOutput) {
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

func parsePyCoverageReportForProject(t *testing.T, projectRoot string, testArgs []string, timeout time.Duration) types.CoverageReport {
	t.Helper()
	ctx := context.Background()
	cancel := func() {}
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()
	cmd := pyCoverageCommand(ctx, testArgs)
	cmd.Dir = projectRoot
	logPyValidationStage(t, "baseline.coverage.start root=%s args=%q timeout=%s", projectRoot, strings.Join(testArgs, " "), timeout)
	output, err := cmd.CombinedOutput()
	logPyValidationStage(t, "baseline.coverage.done root=%s err=%v output_bytes=%d", projectRoot, err, len(output))
	coverageFile := filepath.Join(projectRoot, "coverage.json")
	if err != nil {
		if _, statErr := os.Stat(coverageFile); statErr != nil {
			t.Fatalf("baseline coverage failed and no coverage.json was produced: %v\n%s", err, output)
		}
		t.Logf("baseline coverage exited non-zero but produced coverage.json: %v\n%s", err, output)
	}
	data, err := os.ReadFile(coverageFile)
	if err != nil {
		t.Fatalf("read coverage.json: %v", err)
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
		Framework: "pytest",
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

func pyCoverageCommand(ctx context.Context, testArgs []string) *exec.Cmd {
	if template := strings.TrimSpace(os.Getenv("TESTLOOP_VALIDATE_PY_COVERAGE_COMMAND")); template != "" {
		args := make([]string, 0, len(testArgs))
		for _, arg := range testArgs {
			args = append(args, shellQuote(arg))
		}
		command := strings.ReplaceAll(template, "{args}", strings.Join(args, " "))
		if !strings.Contains(template, "{args}") && len(args) > 0 {
			command = strings.TrimSpace(command + " " + strings.Join(args, " "))
		}
		return configureCommandProcessGroup(exec.CommandContext(ctx, "sh", "-c", command))
	}
	args := []string{"-m", "pytest", "--cov", "--cov-report=json"}
	args = append(args, testArgs...)
	return configureCommandProcessGroup(exec.CommandContext(ctx, "python3", args...))
}

func filterPyCoverageTasks(tasks []types.CoverageTestTask, filter string) []types.CoverageTestTask {
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

func copyPyProjectTree(src string, dst string) error {
	return copyTreeSkipping(src, dst, map[string]bool{
		".git":          true,
		".mypy_cache":   true,
		".pytest_cache": true,
		".ruff_cache":   true,
		".tox":          true,
		".venv":         true,
		"__pycache__":   true,
		"coverage.json": true,
		"htmlcov":       true,
		"venv":          true,
	})
}

func rewritePyValidationPath(baselineRoot string, taskRoot string, value string) string {
	if value == "" {
		return value
	}
	if filepath.IsAbs(value) {
		if rel, err := filepath.Rel(baselineRoot, value); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.Join(taskRoot, rel)
		}
		if candidate := findValidationPathBySourceSuffix(taskRoot, value, []string{"/src/", "/tests/", "/test/"}, []string{"src", "tests", "test"}); candidate != "" {
			return candidate
		}
		if candidate := validationPathBySourceSuffix(taskRoot, value, []string{"/src/", "/tests/", "/test/"}, []string{"src", "tests", "test"}); candidate != "" {
			return candidate
		}
		if candidate := findValidationPathByTail(taskRoot, value, 4); candidate != "" {
			return candidate
		}
		if candidate := validationPathByTail(taskRoot, value, 1); candidate != "" {
			return candidate
		}
		return value
	}
	return filepath.Join(taskRoot, filepath.FromSlash(value))
}

func logPyValidationStage(t *testing.T, format string, args ...any) {
	t.Helper()
	message := fmt.Sprintf(format, args...)
	t.Log(message)
	fmt.Fprintf(os.Stderr, "testloop-py-validation: %s\n", message)
}
