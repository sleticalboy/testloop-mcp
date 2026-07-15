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

func TestValidateJavaCoverageTopTasks(t *testing.T) {
	projectDir := os.Getenv("TESTLOOP_VALIDATE_JAVA_PROJECT_DIR")
	if projectDir == "" {
		t.Skip("TESTLOOP_VALIDATE_JAVA_PROJECT_DIR is not set")
	}
	limit := envPositiveInt(t, "TESTLOOP_VALIDATE_JAVA_TASK_LIMIT", 20)
	stageTimeout := envOptionalDurationSeconds(t, "TESTLOOP_VALIDATE_JAVA_STAGE_TIMEOUT_SECONDS")
	baselineTimeout := firstNonZeroDuration(envOptionalDurationSeconds(t, "TESTLOOP_VALIDATE_JAVA_BASELINE_TIMEOUT_SECONDS"), stageTimeout)
	taskTimeout := firstNonZeroDuration(envOptionalDurationSeconds(t, "TESTLOOP_VALIDATE_JAVA_TASK_TIMEOUT_SECONDS"), stageTimeout)
	outputPath := os.Getenv("TESTLOOP_VALIDATE_JAVA_OUTPUT")
	if outputPath == "" {
		outputPath = filepath.Join(os.TempDir(), fmt.Sprintf("testloop-java-coverage-top%d-%s.jsonl", limit, time.Now().Format("20060102150405")))
	}

	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		t.Fatalf("resolve project dir: %v", err)
	}
	baselineRoot := filepath.Join(t.TempDir(), "baseline")
	logJavaValidationStage(t, "baseline.copy.start project=%s dest=%s", projectDir, baselineRoot)
	if err := copyJavaProjectTree(projectDir, baselineRoot); err != nil {
		t.Fatalf("copy baseline project: %v", err)
	}
	logJavaValidationStage(t, "baseline.copy.done dest=%s", baselineRoot)

	report := parseJavaCoverageReportForProject(t, baselineRoot, baselineTimeout)
	tasks := filterJavaCoverageTasks(report.TestTasks, os.Getenv("TESTLOOP_VALIDATE_JAVA_FILE_FILTER"))
	if len(tasks) < limit {
		t.Fatalf("coverage tasks after filter = %d, want at least %d", len(tasks), limit)
	}
	logJavaValidationStage(t, "tasks.selected count=%d limit=%d filter=%q", len(tasks), limit, os.Getenv("TESTLOOP_VALIDATE_JAVA_FILE_FILTER"))

	outFile, err := os.Create(outputPath)
	if err != nil {
		t.Fatalf("create output jsonl: %v", err)
	}
	defer outFile.Close()
	if envBool("TESTLOOP_VALIDATE_JAVA_LIST_TASKS_ONLY") {
		for i := 0; i < limit; i++ {
			encoded, _ := json.Marshal(tasks[i])
			if _, err := outFile.Write(append(encoded, '\n')); err != nil {
				t.Fatalf("write task output jsonl: %v", err)
			}
		}
		t.Logf("tasks_jsonl=%s", outputPath)
		logJavaValidationStage(t, "tasks.list_only.done count=%d output=%s", limit, outputPath)
		return
	}

	summary := javaProjectValidationSummary{
		Limit:        limit,
		Framework:    "junit",
		StatusCounts: map[string]int{},
		ActionCounts: map[string]int{},
	}
	var failures []string
	for i := 0; i < limit; i++ {
		task := tasks[i]
		taskRoot := filepath.Join(t.TempDir(), fmt.Sprintf("task-%02d", i+1))
		logJavaValidationStage(t, "task.copy.start index=%d id=%s target=%s root=%s", i+1, task.ID, task.Target, taskRoot)
		if err := copyJavaProjectTree(projectDir, taskRoot); err != nil {
			t.Fatalf("copy task worktree for %s: %v", task.ID, err)
		}
		logJavaValidationStage(t, "task.copy.done index=%d id=%s", i+1, task.ID)
		task.File = rewriteJavaValidationPath(baselineRoot, taskRoot, task.File)
		task.TestFile = rewriteJavaValidationPath(baselineRoot, taskRoot, task.TestFile)
		task.TestFile = rewriteJavaValidationTestFileForSource(task.File, task.TestFile)

		includeFixSuggestions := false
		ctx := context.Background()
		cancel := func() {}
		if taskTimeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, taskTimeout)
		}
		logJavaValidationStage(t, "task.validate.start index=%d id=%s target=%s file=%s timeout=%s", i+1, task.ID, task.Target, task.File, taskTimeout)
		validation, _, err := HandleValidateCoverageTask(ctx, nil, validateCoverageTaskInput{
			FilePath:              task.File,
			Framework:             "junit",
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
		logJavaValidationStage(t, "task.validate.done index=%d id=%s target=%s status=%s action=%s", i+1, task.ID, task.Target, out.Status, out.Action)
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

type javaProjectValidationSummary struct {
	Limit        int            `json:"limit"`
	Framework    string         `json:"framework"`
	StatusCounts map[string]int `json:"status_counts"`
	ActionCounts map[string]int `json:"action_counts"`
	ZeroSkip     int            `json:"zero_skip"`
	SkippedTotal int            `json:"skipped_total"`
	SkippedReady []string       `json:"skipped_ready,omitempty"`
}

func (s *javaProjectValidationSummary) record(index int, task types.CoverageTestTask, out types.CoverageTaskValidationOutput) {
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

func parseJavaCoverageReportForProject(t *testing.T, projectRoot string, timeout time.Duration) types.CoverageReport {
	t.Helper()
	ctx := context.Background()
	cancel := func() {}
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()
	cmd := javaCoverageCommand(ctx, projectRoot)
	logJavaValidationStage(t, "baseline.coverage.start root=%s timeout=%s", projectRoot, timeout)
	output, err := cmd.CombinedOutput()
	logJavaValidationStage(t, "baseline.coverage.done root=%s err=%v output_bytes=%d", projectRoot, err, len(output))
	if err != nil {
		t.Fatalf("baseline coverage failed: %v\n%s", err, output)
	}
	coverageFile := javaCoverageFile(projectRoot)
	data, err := os.ReadFile(coverageFile)
	if err != nil {
		t.Fatalf("read Java JaCoCo file %s: %v", coverageFile, err)
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
		Framework: "junit",
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

func javaCoverageCommand(ctx context.Context, projectRoot string) *exec.Cmd {
	template := strings.TrimSpace(os.Getenv("TESTLOOP_VALIDATE_JAVA_COVERAGE_COMMAND"))
	if template != "" {
		cmd := exec.CommandContext(ctx, "sh", "-c", template)
		cmd.Dir = projectRoot
		return configureCommandProcessGroup(cmd)
	}
	return javaTestCommand(ctx, projectRoot, true)
}

func javaCoverageFile(projectRoot string) string {
	if coverageFile := strings.TrimSpace(os.Getenv("TESTLOOP_VALIDATE_JAVA_COVERAGE_FILE")); coverageFile != "" {
		if filepath.IsAbs(coverageFile) {
			return coverageFile
		}
		return filepath.Join(projectRoot, coverageFile)
	}
	for _, candidate := range []string{
		filepath.Join(projectRoot, "target", "site", "jacoco", "jacoco.xml"),
		filepath.Join(projectRoot, "build", "reports", "jacoco", "test", "jacocoTestReport.xml"),
	} {
		if fileExists(candidate) {
			return candidate
		}
	}
	return filepath.Join(projectRoot, "target", "site", "jacoco", "jacoco.xml")
}

func TestJavaCoverageCommandSupportsCustomTemplate(t *testing.T) {
	t.Setenv("TESTLOOP_VALIDATE_JAVA_COVERAGE_COMMAND", "mvn -q test jacoco:report")

	cmd := javaCoverageCommand(context.Background(), t.TempDir())

	got := strings.Join(cmd.Args, " ")
	want := "sh -c mvn -q test jacoco:report"
	if got != want {
		t.Fatalf("javaCoverageCommand args = %q, want %q", got, want)
	}
}

func TestJavaCoverageCommandKillsProcessGroupOnTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell process group cancellation is only configured on Unix platforms")
	}
	t.Setenv("TESTLOOP_VALIDATE_JAVA_COVERAGE_COMMAND", "sh -c 'sleep 5'")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	cmd := javaCoverageCommand(ctx, t.TempDir())
	_ = cmd.Run()
	elapsed := time.Since(start)
	if elapsed > 3*time.Second {
		t.Fatalf("Java coverage command timeout took %s, child process likely survived context cancellation", elapsed)
	}
}

func TestRewriteJavaValidationPathMapsJaCoCoPackagePathToMavenSource(t *testing.T) {
	baselineRoot := filepath.Join(t.TempDir(), "baseline")
	taskRoot := filepath.Join(t.TempDir(), "task")
	source := filepath.Join(taskRoot, "src", "main", "java", "com", "example", "Calculator.java")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.WriteFile(source, []byte("class Calculator {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	got := rewriteJavaValidationPath(baselineRoot, taskRoot, filepath.FromSlash("com/example/Calculator.java"))

	if got != source {
		t.Fatalf("rewriteJavaValidationPath = %q, want %q", got, source)
	}
}

func TestRewriteJavaValidationPathMapsJaCoCoPackagePathToNestedMavenModule(t *testing.T) {
	baselineRoot := filepath.Join(t.TempDir(), "baseline")
	taskRoot := filepath.Join(t.TempDir(), "task")
	source := filepath.Join(taskRoot, "client", "src", "main", "java", "org", "apache", "rocketmq", "client", "java", "route", "Endpoints.java")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.WriteFile(source, []byte("class Endpoints {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	got := rewriteJavaValidationPath(baselineRoot, taskRoot, filepath.FromSlash("org/apache/rocketmq/client/java/route/Endpoints.java"))

	if got != source {
		t.Fatalf("rewriteJavaValidationPath = %q, want %q", got, source)
	}
}

func TestRewriteJavaValidationTestFileForSourceKeepsNestedMavenModule(t *testing.T) {
	root := filepath.Join(t.TempDir(), "task")
	source := filepath.Join(root, "client", "src", "main", "java", "org", "apache", "rocketmq", "client", "java", "route", "Endpoints.java")
	current := filepath.Join(root, "src", "test", "java", "org", "apache", "rocketmq", "client", "java", "route", "EndpointsTest.java")
	want := filepath.Join(root, "client", "src", "test", "java", "org", "apache", "rocketmq", "client", "java", "route", "EndpointsTest.java")

	got := rewriteJavaValidationTestFileForSource(source, current)

	if got != want {
		t.Fatalf("rewriteJavaValidationTestFileForSource = %q, want %q", got, want)
	}
}

func filterJavaCoverageTasks(tasks []types.CoverageTestTask, filter string) []types.CoverageTestTask {
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

func copyJavaProjectTree(src string, dst string) error {
	return copyTreeSkipping(src, dst, map[string]bool{
		".git":    true,
		".gradle": true,
		"target":  true,
		"build":   true,
	})
}

func rewriteJavaValidationPath(baselineRoot string, taskRoot string, value string) string {
	rewritten := rewriteGoValidationPath(baselineRoot, taskRoot, value)
	if rewritten == "" || fileExists(rewritten) || filepath.IsAbs(value) {
		return rewritten
	}
	for _, root := range []string{filepath.Join("src", "main", "java"), filepath.Join("src", "test", "java")} {
		candidate := filepath.Join(taskRoot, root, filepath.FromSlash(value))
		if fileExists(candidate) {
			return candidate
		}
	}
	if candidate := findJavaValidationNestedPath(taskRoot, value); candidate != "" {
		return candidate
	}
	return rewritten
}

func findJavaValidationNestedPath(taskRoot string, value string) string {
	slash := filepath.ToSlash(value)
	for _, root := range []string{"src/main/java", "src/test/java"} {
		patterns := []string{
			filepath.Join(taskRoot, filepath.FromSlash("*/"+root+"/"+slash)),
			filepath.Join(taskRoot, filepath.FromSlash("*/*/"+root+"/"+slash)),
		}
		for _, pattern := range patterns {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				continue
			}
			for _, match := range matches {
				if fileExists(match) {
					return match
				}
			}
		}
	}
	return ""
}

func rewriteJavaValidationTestFileForSource(sourceFile string, testFile string) string {
	sourceSlash := filepath.ToSlash(sourceFile)
	const marker = "/src/main/java/"
	idx := strings.Index(sourceSlash, marker)
	if idx < 0 {
		return testFile
	}
	rel := sourceSlash[idx+len(marker):]
	if strings.ToLower(filepath.Ext(rel)) != ".java" {
		return testFile
	}
	base := strings.TrimSuffix(rel, filepath.Ext(rel))
	return filepath.FromSlash(sourceSlash[:idx] + "/src/test/java/" + base + "Test.java")
}

func logJavaValidationStage(t *testing.T, format string, args ...any) {
	t.Helper()
	message := fmt.Sprintf(format, args...)
	t.Log(message)
	fmt.Fprintf(os.Stderr, "testloop-java-validation: %s\n", message)
}
