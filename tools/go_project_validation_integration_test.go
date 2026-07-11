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

	"github.com/sleticalboy/testloop-mcp/types"
)

func TestValidateGoCoverageTopTasks(t *testing.T) {
	projectDir := os.Getenv("TESTLOOP_VALIDATE_GO_PROJECT_DIR")
	if projectDir == "" {
		t.Skip("TESTLOOP_VALIDATE_GO_PROJECT_DIR is not set")
	}
	limit := envPositiveInt(t, "TESTLOOP_VALIDATE_GO_TASK_LIMIT", 50)
	outputPath := os.Getenv("TESTLOOP_VALIDATE_GO_OUTPUT")
	if outputPath == "" {
		outputPath = filepath.Join(os.TempDir(), fmt.Sprintf("testloop-go-coverage-top%d-%s.jsonl", limit, time.Now().Format("20060102150405")))
	}

	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		t.Fatalf("resolve project dir: %v", err)
	}
	baselineRoot := filepath.Join(t.TempDir(), "baseline")
	if err := copyTree(projectDir, baselineRoot); err != nil {
		t.Fatalf("copy baseline project: %v", err)
	}

	report := parseGoCoverageReportForProject(t, baselineRoot)
	if len(report.TestTasks) < limit {
		t.Fatalf("coverage tasks = %d, want at least %d", len(report.TestTasks), limit)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		t.Fatalf("create output jsonl: %v", err)
	}
	defer outFile.Close()

	summary := goProjectValidationSummary{
		Limit:        limit,
		StatusCounts: map[string]int{},
		ActionCounts: map[string]int{},
	}
	for i := 0; i < limit; i++ {
		task := report.TestTasks[i]
		taskRoot := filepath.Join(t.TempDir(), fmt.Sprintf("task-%02d", i+1))
		if err := copyTree(projectDir, taskRoot); err != nil {
			t.Fatalf("copy task worktree for %s: %v", task.ID, err)
		}
		task.File = rewriteGoValidationPath(baselineRoot, taskRoot, task.File)
		task.TestFile = rewriteGoValidationPath(baselineRoot, taskRoot, task.TestFile)

		validation, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
			FilePath:     task.File,
			CoverageTask: &task,
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
		if out.Status != "passed" {
			t.Fatalf("task %d %s %s status=%s action=%s error=%s", i+1, task.ID, task.Target, out.Status, out.Action, out.Error)
		}
	}

	sort.Strings(summary.SkippedReady)
	summaryJSON, _ := json.Marshal(summary)
	t.Logf("result_jsonl=%s", outputPath)
	t.Logf("summary=%s", summaryJSON)
}

type goProjectValidationSummary struct {
	Limit        int            `json:"limit"`
	StatusCounts map[string]int `json:"status_counts"`
	ActionCounts map[string]int `json:"action_counts"`
	ZeroSkip     int            `json:"zero_skip"`
	SkippedTotal int            `json:"skipped_total"`
	SkippedReady []string       `json:"skipped_ready,omitempty"`
}

func (s *goProjectValidationSummary) record(index int, task types.CoverageTestTask, out types.CoverageTaskValidationOutput) {
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

func parseGoCoverageReportForProject(t *testing.T, projectRoot string) types.CoverageReport {
	t.Helper()
	coverProfile := filepath.Join(t.TempDir(), "coverage.out")
	cmd := exec.Command("go", "test", "./...", "-coverprofile", coverProfile)
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("baseline coverage failed: %v\n%s", err, out)
	}
	data, err := os.ReadFile(coverProfile)
	if err != nil {
		t.Fatalf("read coverage profile: %v", err)
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
		Framework: "go-test",
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

func rewriteGoValidationPath(baselineRoot string, taskRoot string, value string) string {
	if value == "" {
		return value
	}
	if filepath.IsAbs(value) {
		if rel, err := filepath.Rel(baselineRoot, value); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.Join(taskRoot, rel)
		}
		return value
	}
	return filepath.Join(taskRoot, filepath.FromSlash(value))
}

func envPositiveInt(t *testing.T, name string, fallback int) int {
	t.Helper()
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		t.Fatalf("%s=%q is not a positive integer", name, raw)
	}
	return value
}

func copyTree(src string, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == ".git" {
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
			if err := copyTree(srcPath, dstPath); err != nil {
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
