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
	outputPath := os.Getenv("TESTLOOP_VALIDATE_JS_OUTPUT")
	if outputPath == "" {
		outputPath = filepath.Join(os.TempDir(), fmt.Sprintf("testloop-js-coverage-top%d-%s.jsonl", limit, time.Now().Format("20060102150405")))
	}

	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		t.Fatalf("resolve project dir: %v", err)
	}
	baselineRoot := filepath.Join(t.TempDir(), "baseline")
	if err := copyJSProjectTree(projectDir, baselineRoot); err != nil {
		t.Fatalf("copy baseline project: %v", err)
	}
	if err := linkJSExtraResources(projectDir, baselineRoot, os.Getenv("TESTLOOP_VALIDATE_JS_EXTRA_SYMLINKS")); err != nil {
		t.Fatalf("link baseline extra resources: %v", err)
	}

	report := parseJSCoverageReportForProject(t, baselineRoot, framework, strings.Fields(os.Getenv("TESTLOOP_VALIDATE_JS_TEST_ARGS")))
	tasks := filterJSCoverageTasks(report.TestTasks, os.Getenv("TESTLOOP_VALIDATE_JS_FILE_FILTER"))
	if len(tasks) < limit {
		t.Fatalf("coverage tasks after filter = %d, want at least %d", len(tasks), limit)
	}

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
		if err := copyJSProjectTree(projectDir, taskRoot); err != nil {
			t.Fatalf("copy task worktree for %s: %v", task.ID, err)
		}
		if err := linkJSExtraResources(projectDir, taskRoot, os.Getenv("TESTLOOP_VALIDATE_JS_EXTRA_SYMLINKS")); err != nil {
			t.Fatalf("link task extra resources for %s: %v", task.ID, err)
		}
		task.File = rewriteJSValidationPath(baselineRoot, taskRoot, task.File)
		task.TestFile = rewriteJSValidationPath(baselineRoot, taskRoot, task.TestFile)

		includeFixSuggestions := false
		validation, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
			FilePath:              task.File,
			Framework:             framework,
			CoverageTask:          &task,
			IncludeFixSuggestions: &includeFixSuggestions,
		})
		if err != nil {
			t.Fatalf("validate task %d %s %s: %v", i+1, task.ID, task.Target, err)
		}
		var out types.CoverageTaskValidationOutput
		if err := json.Unmarshal([]byte(resultText(t, validation)), &out); err != nil {
			t.Fatalf("unmarshal validation output for %s: %v", task.ID, err)
		}
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

func parseJSCoverageReportForProject(t *testing.T, projectRoot string, framework string, testArgs []string) types.CoverageReport {
	t.Helper()
	cmd := jsCoverageCommand(framework, testArgs)
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
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

func jsCoverageCommand(framework string, testArgs []string) *exec.Cmd {
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
	return exec.Command("npx", args...)
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
