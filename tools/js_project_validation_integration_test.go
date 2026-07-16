package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sleticalboy/testloop-mcp/internal/generator"
	"github.com/sleticalboy/testloop-mcp/types"
)

func TestValidateJSCoverageTopTasks(t *testing.T) {
	projectDir := os.Getenv("TESTLOOP_VALIDATE_JS_PROJECT_DIR")
	if projectDir == "" {
		t.Skip("TESTLOOP_VALIDATE_JS_PROJECT_DIR is not set")
	}
	framework := strings.TrimSpace(os.Getenv("TESTLOOP_VALIDATE_JS_FRAMEWORK"))
	if framework == "" {
		framework = "vitest"
	}
	if framework != "vitest" && framework != "jest" && framework != "mocha" {
		t.Fatalf("unsupported TESTLOOP_VALIDATE_JS_FRAMEWORK=%q", framework)
	}
	limit := envPositiveInt(t, "TESTLOOP_VALIDATE_JS_TASK_LIMIT", 20)
	stageTimeout := envOptionalDurationSeconds(t, "TESTLOOP_VALIDATE_JS_STAGE_TIMEOUT_SECONDS")
	baselineTimeout := firstNonZeroDuration(envOptionalDurationSeconds(t, "TESTLOOP_VALIDATE_JS_BASELINE_TIMEOUT_SECONDS"), stageTimeout)
	taskTimeout := firstNonZeroDuration(envOptionalDurationSeconds(t, "TESTLOOP_VALIDATE_JS_TASK_TIMEOUT_SECONDS"), stageTimeout)
	outputPath := os.Getenv("TESTLOOP_VALIDATE_JS_OUTPUT")
	if outputPath == "" {
		outputPath = filepath.Join(os.TempDir(), fmt.Sprintf("testloop-js-coverage-top%d-%s.jsonl", limit, time.Now().Format("20060102150405")))
	}

	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		t.Fatalf("resolve project dir: %v", err)
	}
	tasksFile := os.Getenv("TESTLOOP_VALIDATE_JS_TASKS_FILE")
	taskIDFilter := os.Getenv("TESTLOOP_VALIDATE_JS_TASK_IDS")
	baselineRoot := projectDir
	var tasks []types.CoverageTestTask
	if strings.TrimSpace(tasksFile) != "" {
		tasks = readCoverageTasksJSONL(t, tasksFile)
		logJSValidationStage(t, "tasks.file.loaded path=%s count=%d", tasksFile, len(tasks))
	} else {
		baselineRoot = filepath.Join(t.TempDir(), "baseline")
		logJSValidationStage(t, "baseline.copy.start project=%s dest=%s", projectDir, baselineRoot)
		if err := copyJSProjectTree(projectDir, baselineRoot); err != nil {
			t.Fatalf("copy baseline project: %v", err)
		}
		logJSValidationStage(t, "baseline.copy.done dest=%s", baselineRoot)
		logJSValidationStage(t, "baseline.link.start extra=%q", os.Getenv("TESTLOOP_VALIDATE_JS_EXTRA_SYMLINKS"))
		if err := linkJSExtraResources(projectDir, baselineRoot, os.Getenv("TESTLOOP_VALIDATE_JS_EXTRA_SYMLINKS")); err != nil {
			t.Fatalf("link baseline extra resources: %v", err)
		}
		logJSValidationStage(t, "baseline.link.done")

		report := parseJSCoverageReportForProject(t, baselineRoot, framework, strings.Fields(os.Getenv("TESTLOOP_VALIDATE_JS_TEST_ARGS")), baselineTimeout)
		tasks = jsCoverageTasksForValidationFilter(report, baselineRoot, os.Getenv("TESTLOOP_VALIDATE_JS_FILE_FILTER"))
	}
	tasks = filterCoverageTasksByFileAndIDs(tasks, os.Getenv("TESTLOOP_VALIDATE_JS_FILE_FILTER"), taskIDFilter)
	if len(tasks) == 0 {
		t.Fatalf("coverage tasks after filter = 0, file_filter=%q task_ids=%q tasks_file=%q", os.Getenv("TESTLOOP_VALIDATE_JS_FILE_FILTER"), taskIDFilter, tasksFile)
	}
	if taskIDFilter != "" && len(tasks) < limit {
		limit = len(tasks)
	}
	if len(tasks) < limit {
		t.Fatalf("coverage tasks after filter = %d, want at least %d", len(tasks), limit)
	}
	logJSValidationStage(t, "tasks.selected count=%d limit=%d filter=%q task_ids=%q", len(tasks), limit, os.Getenv("TESTLOOP_VALIDATE_JS_FILE_FILTER"), taskIDFilter)

	outFile, err := os.Create(outputPath)
	if err != nil {
		t.Fatalf("create output jsonl: %v", err)
	}
	defer outFile.Close()

	summary := jsProjectValidationSummary{
		Limit:        limit,
		Framework:    framework,
		StatusCounts: map[string]int{},
		ActionCounts: map[string]int{},
	}
	var failures []string
	for i := 0; i < limit; i++ {
		task := tasks[i]
		taskRoot := filepath.Join(t.TempDir(), fmt.Sprintf("task-%02d", i+1))
		logJSValidationStage(t, "task.copy.start index=%d id=%s target=%s root=%s", i+1, task.ID, task.Target, taskRoot)
		if err := copyJSProjectTree(projectDir, taskRoot); err != nil {
			t.Fatalf("copy task worktree for %s: %v", task.ID, err)
		}
		logJSValidationStage(t, "task.copy.done index=%d id=%s", i+1, task.ID)
		if err := linkJSExtraResources(projectDir, taskRoot, os.Getenv("TESTLOOP_VALIDATE_JS_EXTRA_SYMLINKS")); err != nil {
			t.Fatalf("link task extra resources for %s: %v", task.ID, err)
		}
		task.File = rewriteJSValidationPath(baselineRoot, taskRoot, task.File)
		task.TestFile = rewriteJSValidationPath(baselineRoot, taskRoot, task.TestFile)

		includeFixSuggestions := false
		ctx := context.Background()
		cancel := func() {}
		if taskTimeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, taskTimeout)
		}
		logJSValidationStage(t, "task.validate.start index=%d id=%s target=%s file=%s timeout=%s", i+1, task.ID, task.Target, task.File, taskTimeout)
		validation, _, err := HandleValidateCoverageTask(ctx, nil, validateCoverageTaskInput{
			FilePath:              task.File,
			Framework:             framework,
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
		logJSValidationStage(t, "task.validate.done index=%d id=%s target=%s status=%s action=%s", i+1, task.ID, task.Target, out.Status, out.Action)
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

func TestSynthesizeJSTypeOnlyFileLevelTasks(t *testing.T) {
	projectRoot := t.TempDir()
	srcDir := filepath.Join(projectRoot, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("create src dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "tests"), 0o755); err != nil {
		t.Fatalf("create tests dir: %v", err)
	}
	typeOnly := filepath.Join(srcDir, "events.ts")
	if err := os.WriteFile(typeOnly, []byte(`export type ThreadStartedEvent = {
  type: "thread.started";
  thread_id: string;
};
`), 0o644); err != nil {
		t.Fatalf("write type-only source: %v", err)
	}
	runtime := filepath.Join(srcDir, "codex.ts")
	if err := os.WriteFile(runtime, []byte(`export function startThread() {
  return {};
}
`), 0o644); err != nil {
		t.Fatalf("write runtime source: %v", err)
	}
	barrel := filepath.Join(srcDir, "index.ts")
	if err := os.WriteFile(barrel, []byte(`export type {
  ThreadStartedEvent,
} from "./events";

export { startThread } from "./codex";
`), 0o644); err != nil {
		t.Fatalf("write barrel source: %v", err)
	}

	tasks := synthesizeJSTypeOnlyFileLevelTasks("jest", projectRoot, "src/events.ts")
	if len(tasks) != 1 {
		t.Fatalf("synthesizeJSTypeOnlyFileLevelTasks() got %d tasks, want 1: %+v", len(tasks), tasks)
	}
	task := tasks[0]
	if task.Target != "events.ts" || task.GapType != "no_runtime" || task.LineRange != "entire file" {
		t.Fatalf("unexpected synthesized task: %+v", task)
	}
	if task.TestFile != filepath.Join(projectRoot, "tests", "events.test.ts") {
		t.Fatalf("unexpected test file: %q", task.TestFile)
	}

	if got := synthesizeJSTypeOnlyFileLevelTasks("jest", projectRoot, "src/codex.ts"); len(got) != 0 {
		t.Fatalf("runtime file should not synthesize no-runtime task: %+v", got)
	}

	barrelTasks := synthesizeJSTypeOnlyFileLevelTasks("jest", projectRoot, "src/index.ts")
	if len(barrelTasks) != 1 {
		t.Fatalf("barrel file should synthesize no-runtime task, got %+v", barrelTasks)
	}
	if barrelTasks[0].Target != "index.ts" || barrelTasks[0].GapType != "no_runtime" {
		t.Fatalf("unexpected barrel task: %+v", barrelTasks[0])
	}
}

func TestJSCoverageCommandSupportsCustomTemplate(t *testing.T) {
	t.Setenv("TESTLOOP_VALIDATE_JS_COVERAGE_COMMAND", "npx egg-bin cov --timeout 60000 {args}")

	cmd := jsCoverageCommand(context.Background(), "mocha", []string{"test/index.test.ts", "test/space name.test.ts"})

	got := strings.Join(cmd.Args, " ")
	want := "sh -c npx egg-bin cov --timeout 60000 'test/index.test.ts' 'test/space name.test.ts'"
	if got != want {
		t.Fatalf("jsCoverageCommand args = %q, want %q", got, want)
	}
}

func envOptionalDurationSeconds(t *testing.T, name string) time.Duration {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return 0
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		t.Fatalf("%s=%q is not a positive integer second value", name, raw)
	}
	return time.Duration(seconds) * time.Second
}

func firstNonZeroDuration(values ...time.Duration) time.Duration {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func logJSValidationStage(t *testing.T, format string, args ...any) {
	t.Helper()
	message := fmt.Sprintf(format, args...)
	t.Log(message)
	fmt.Fprintf(os.Stderr, "testloop-js-validation: %s\n", message)
}

type jsProjectValidationSummary struct {
	Limit        int            `json:"limit"`
	Framework    string         `json:"framework"`
	StatusCounts map[string]int `json:"status_counts"`
	ActionCounts map[string]int `json:"action_counts"`
	ZeroSkip     int            `json:"zero_skip"`
	SkippedTotal int            `json:"skipped_total"`
	SkippedReady []string       `json:"skipped_ready,omitempty"`
}

func (s *jsProjectValidationSummary) record(index int, task types.CoverageTestTask, out types.CoverageTaskValidationOutput) {
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

func parseJSCoverageReportForProject(t *testing.T, projectRoot string, framework string, testArgs []string, timeout time.Duration) types.CoverageReport {
	t.Helper()
	ctx := context.Background()
	cancel := func() {}
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()
	cmd := jsCoverageCommand(ctx, framework, testArgs)
	cmd.Dir = projectRoot
	logJSValidationStage(t, "baseline.coverage.start root=%s framework=%s args=%q timeout=%s", projectRoot, framework, strings.Join(testArgs, " "), timeout)
	output, err := cmd.CombinedOutput()
	logJSValidationStage(t, "baseline.coverage.done root=%s framework=%s err=%v output_bytes=%d", projectRoot, framework, err, len(output))
	coverageFile := filepath.Join(projectRoot, "coverage", "coverage-final.json")
	if err != nil {
		if _, statErr := os.Stat(coverageFile); statErr != nil {
			t.Fatalf("baseline coverage failed and no coverage-final.json was produced: %v\n%s", err, output)
		}
		t.Logf("baseline coverage exited non-zero but produced coverage-final.json: %v\n%s", err, output)
	}
	data, err := os.ReadFile(coverageFile)
	if err != nil {
		t.Fatalf("read coverage-final.json: %v", err)
	}
	result, _, err := HandleParseCoverage(context.Background(), nil, parseCoverageInput{
		Data:      string(data),
		Framework: framework,
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

func jsCoverageCommand(ctx context.Context, framework string, testArgs []string) *exec.Cmd {
	if template := strings.TrimSpace(os.Getenv("TESTLOOP_VALIDATE_JS_COVERAGE_COMMAND")); template != "" {
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
	args := []string{}
	switch framework {
	case "vitest":
		args = append(args, "vitest", "run", "--coverage")
	case "mocha":
		args = append(args, "mocha")
	default:
		args = append(args, "jest", "--coverage")
	}
	args = append(args, testArgs...)
	return configureCommandProcessGroup(exec.CommandContext(ctx, "npx", args...))
}

func filterJSCoverageTasks(tasks []types.CoverageTestTask, filter string) []types.CoverageTestTask {
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

func jsCoverageTasksForValidationFilter(report types.CoverageReport, projectRoot string, filter string) []types.CoverageTestTask {
	tasks := filterJSCoverageTasks(report.TestTasks, filter)
	if len(tasks) > 0 || strings.TrimSpace(filter) == "" {
		return tasks
	}
	return synthesizeJSTypeOnlyFileLevelTasks(report.Framework, projectRoot, filter)
}

func synthesizeJSTypeOnlyFileLevelTasks(framework string, projectRoot string, filter string) []types.CoverageTestTask {
	filter = filepath.ToSlash(strings.TrimSpace(filter))
	if filter == "" {
		return nil
	}
	var tasks []types.CoverageTestTask
	_ = filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ts" && ext != ".tsx" {
			return nil
		}
		rel, relErr := filepath.Rel(projectRoot, path)
		if relErr != nil {
			return nil
		}
		slashPath := filepath.ToSlash(path)
		slashRel := filepath.ToSlash(rel)
		if !strings.Contains(slashPath, filter) && !strings.Contains(slashRel, filter) {
			return nil
		}
		if !jsValidationTSModuleHasNoRuntimeTargets(path) {
			return nil
		}
		target := filepath.Base(path)
		task := types.CoverageTestTask{
			ID:             fmt.Sprintf("%s-no-runtime-%d", sanitizeJSValidationTaskID(framework), len(tasks)+1),
			Framework:      framework,
			File:           path,
			Target:         target,
			Kind:           "file_level",
			LineRange:      "entire file",
			GapType:        "no_runtime",
			Goal:           fmt.Sprintf("确认 %s 没有本地可执行的 TypeScript 运行时代码覆盖任务", target),
			Command:        jsValidationCoverageTaskCommand(framework, path),
			TestFile:       jsValidationTestFileForSource(projectRoot, path),
			TestName:       "marks type-only module as no runtime coverage",
			AssertionFocus: []string{"纯类型或 re-export 声明不会产生有意义的本地 JavaScript coverage task；应通过消费方运行时测试或类型检查验证"},
			Priority:       90,
			PriorityReason: "file filter matched a TypeScript module with no local runtime targets that coverage data omits",
			Confidence:     0.9,
		}
		tasks = append(tasks, task)
		return nil
	})
	return tasks
}

func jsValidationTSModuleHasNoRuntimeTargets(path string) bool {
	ctx := generator.BuildGenerationContext(path)
	if ctx != nil {
		if len(ctx.Targets) > 0 {
			return false
		}
		if len(ctx.Types) > 0 {
			return true
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return jsValidationSourceIsImportExportOnly(string(data))
}

func jsValidationSourceIsImportExportOnly(source string) bool {
	inImportExportBlock := false
	for _, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if inImportExportBlock {
			if strings.Contains(trimmed, ";") {
				inImportExportBlock = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "export ") {
			if !strings.Contains(trimmed, ";") {
				inImportExportBlock = true
			}
			continue
		}
		return false
	}
	return true
}

func sanitizeJSValidationTaskID(framework string) string {
	framework = strings.TrimSpace(framework)
	if framework == "" {
		framework = "js"
	}
	return strings.NewReplacer(" ", "-", "_", "-", ".", "-").Replace(framework)
}

func jsValidationTestFileForSource(projectRoot string, sourcePath string) string {
	ext := filepath.Ext(sourcePath)
	name := strings.TrimSuffix(filepath.Base(sourcePath), ext) + ".test" + ext
	testsDir := filepath.Join(projectRoot, "tests")
	if info, err := os.Stat(testsDir); err == nil && info.IsDir() {
		return filepath.Join(testsDir, name)
	}
	return generator.TestFileName(sourcePath)
}

func jsValidationCoverageTaskCommand(framework string, file string) string {
	switch framework {
	case "jest":
		return "npx jest " + filepath.ToSlash(file)
	case "mocha":
		return "npx mocha " + filepath.ToSlash(file)
	default:
		return "npx vitest run " + filepath.ToSlash(file)
	}
}

func rewriteJSValidationPath(baselineRoot string, taskRoot string, value string) string {
	if value == "" {
		return value
	}
	roots := []string{filepath.Clean(baselineRoot)}
	if realRoot, err := filepath.EvalSymlinks(baselineRoot); err == nil {
		roots = append(roots, filepath.Clean(realRoot))
	}
	cleanValue := filepath.Clean(value)
	if realValue, err := filepath.EvalSymlinks(value); err == nil {
		cleanValue = filepath.Clean(realValue)
	}
	for _, root := range roots {
		if rel, err := filepath.Rel(root, cleanValue); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return filepath.Join(taskRoot, rel)
		}
	}
	if filepath.IsAbs(value) {
		if candidate := findValidationPathBySourceSuffix(taskRoot, value, []string{"/src/", "/test/", "/tests/"}, []string{"src", "test", "tests"}); candidate != "" {
			return candidate
		}
		if candidate := validationPathBySourceSuffix(taskRoot, value, []string{"/src/", "/test/", "/tests/"}, []string{"src", "test", "tests"}); candidate != "" {
			return candidate
		}
	}
	return rewriteGoValidationPath(baselineRoot, taskRoot, value)
}

func linkJSExtraResources(projectDir string, copyRoot string, spec string) error {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil
	}
	for _, raw := range strings.Split(spec, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid TESTLOOP_VALIDATE_JS_EXTRA_SYMLINKS entry %q, expected src:dst", raw)
		}
		src := strings.TrimSpace(parts[0])
		dst := strings.TrimSpace(parts[1])
		if src == "" || dst == "" {
			return fmt.Errorf("invalid TESTLOOP_VALIDATE_JS_EXTRA_SYMLINKS entry %q, src and dst are required", raw)
		}
		if !filepath.IsAbs(src) {
			src = filepath.Join(projectDir, src)
		}
		src, err := filepath.Abs(src)
		if err != nil {
			return fmt.Errorf("resolve extra source %q: %w", parts[0], err)
		}
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("stat extra source %q: %w", src, err)
		}
		linkPath := filepath.Clean(filepath.Join(copyRoot, dst))
		if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
			return fmt.Errorf("create extra resource parent for %q: %w", linkPath, err)
		}
		if existing, err := os.Readlink(linkPath); err == nil {
			if existing == src {
				continue
			}
			return fmt.Errorf("extra resource link %q already points to %q, want %q", linkPath, existing, src)
		} else if !os.IsNotExist(err) {
			if _, statErr := os.Stat(linkPath); statErr == nil {
				continue
			}
			return fmt.Errorf("inspect extra resource link %q: %w", linkPath, err)
		}
		if err := os.Symlink(src, linkPath); err != nil {
			return fmt.Errorf("link extra resource %q -> %q: %w", linkPath, src, err)
		}
	}
	return nil
}

func copyJSProjectTree(src string, dst string) error {
	if err := copyTreeSkipping(src, dst, map[string]bool{
		".git":          true,
		"coverage":      true,
		"dist":          true,
		"node_modules":  true,
		".turbo":        true,
		".vite":         true,
		".vitest-cache": true,
	}); err != nil {
		return err
	}
	srcNodeModules := filepath.Join(src, "node_modules")
	if _, err := os.Stat(srcNodeModules); err == nil {
		if err := os.Symlink(srcNodeModules, filepath.Join(dst, "node_modules")); err != nil {
			return err
		}
	}
	return nil
}

func copyTreeSkipping(src string, dst string, skipNames map[string]bool) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		if skipNames[entry.Name()] {
			continue
		}
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(srcPath)
			if err != nil {
				return err
			}
			if err := os.Symlink(target, dstPath); err != nil {
				return err
			}
			continue
		}
		if entry.IsDir() {
			if err := copyTreeSkipping(srcPath, dstPath, skipNames); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dstPath, data, info.Mode()); err != nil {
			return err
		}
	}
	return nil
}
