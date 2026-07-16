package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sleticalboy/testloop-mcp/types"
)

func readCoverageTasksJSONL(t *testing.T, path string) []types.CoverageTestTask {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open coverage tasks JSONL %s: %v", path, err)
	}
	defer file.Close()

	var tasks []types.CoverageTestTask
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		task, err := decodeCoverageTaskJSONLine(line)
		if err != nil {
			t.Fatalf("decode coverage task %s:%d: %v", path, lineNo, err)
		}
		tasks = append(tasks, task)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan coverage tasks JSONL %s: %v", path, err)
	}
	if len(tasks) == 0 {
		t.Fatalf("coverage tasks JSONL %s did not contain any tasks", path)
	}
	return tasks
}

func decodeCoverageTaskJSONLine(line string) (types.CoverageTestTask, error) {
	var wrapped struct {
		CoverageTask *types.CoverageTestTask `json:"coverage_task"`
	}
	if err := json.Unmarshal([]byte(line), &wrapped); err != nil {
		return types.CoverageTestTask{}, err
	}
	if wrapped.CoverageTask != nil {
		return *wrapped.CoverageTask, nil
	}
	var task types.CoverageTestTask
	if err := json.Unmarshal([]byte(line), &task); err != nil {
		return types.CoverageTestTask{}, err
	}
	if strings.TrimSpace(task.ID) == "" && strings.TrimSpace(task.Target) == "" {
		return types.CoverageTestTask{}, fmt.Errorf("JSON line is neither a coverage task nor a validation output")
	}
	return task, nil
}

func filterCoverageTasksByFileAndIDs(tasks []types.CoverageTestTask, filter string, taskIDs string) []types.CoverageTestTask {
	filter = strings.TrimSpace(filter)
	ids := coverageTaskIDSet(taskIDs)
	if filter == "" && len(ids) == 0 {
		return tasks
	}
	filtered := make([]types.CoverageTestTask, 0, len(tasks))
	for _, task := range tasks {
		if filter != "" && !strings.Contains(filepath.ToSlash(task.File), filter) {
			continue
		}
		if len(ids) > 0 {
			if _, ok := ids[strings.TrimSpace(task.ID)]; !ok {
				continue
			}
		}
		filtered = append(filtered, task)
	}
	return filtered
}

func coverageTaskIDSet(raw string) map[string]struct{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	ids := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		id := strings.TrimSpace(part)
		if id != "" {
			ids[id] = struct{}{}
		}
	}
	return ids
}

func findValidationPathBySourceSuffix(taskRoot string, value string, markers []string, roots []string) string {
	for _, candidate := range validationPathCandidatesBySourceSuffix(taskRoot, value, markers, roots) {
		if fileExists(candidate) {
			return candidate
		}
	}
	slash := filepath.ToSlash(value)
	for _, marker := range markers {
		idx := strings.LastIndex(slash, marker)
		if idx < 0 {
			continue
		}
		rel := slash[idx+len(marker):]
		if candidate := findValidationNestedPath(taskRoot, rel, roots); candidate != "" {
			return candidate
		}
	}
	return ""
}

func validationPathBySourceSuffix(taskRoot string, value string, markers []string, roots []string) string {
	candidates := validationPathCandidatesBySourceSuffix(taskRoot, value, markers, roots)
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0]
}

func validationPathCandidatesBySourceSuffix(taskRoot string, value string, markers []string, roots []string) []string {
	slash := filepath.ToSlash(value)
	var candidates []string
	for _, marker := range markers {
		idx := strings.LastIndex(slash, marker)
		if idx < 0 {
			continue
		}
		rel := slash[idx+len(marker):]
		for _, root := range roots {
			candidates = append(candidates, filepath.Join(taskRoot, filepath.FromSlash(root), filepath.FromSlash(rel)))
		}
		return candidates
	}
	return nil
}

func findValidationNestedPath(taskRoot string, value string, roots []string) string {
	slash := filepath.ToSlash(value)
	for _, root := range roots {
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

func TestDecodeCoverageTaskJSONLineSupportsTaskAndValidationOutput(t *testing.T) {
	task, err := decodeCoverageTaskJSONLine(`{"id":"task-1","target":"Foo.bar"}`)
	if err != nil {
		t.Fatalf("decode task line: %v", err)
	}
	if task.ID != "task-1" || task.Target != "Foo.bar" {
		t.Fatalf("decoded task = %+v", task)
	}

	task, err = decodeCoverageTaskJSONLine(`{"status":"passed","coverage_task":{"id":"task-2","target":"Bar.baz"}}`)
	if err != nil {
		t.Fatalf("decode validation output line: %v", err)
	}
	if task.ID != "task-2" || task.Target != "Bar.baz" {
		t.Fatalf("decoded wrapped task = %+v", task)
	}
}

func TestFilterCoverageTasksByFileAndIDsKeepsCoverageOrder(t *testing.T) {
	tasks := []types.CoverageTestTask{
		{ID: "task-1", File: "src/foo.go"},
		{ID: "task-2", File: "src/bar.go"},
		{ID: "task-3", File: "src/foo.go"},
	}

	filtered := filterCoverageTasksByFileAndIDs(tasks, "foo.go", "task-3,task-1")

	if len(filtered) != 2 || filtered[0].ID != "task-1" || filtered[1].ID != "task-3" {
		t.Fatalf("filterCoverageTasksByFileAndIDs() = %+v, want task-1 then task-3", filtered)
	}
}
