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

func TestHandleValidateCoverageTaskReturnsAdjustedCoverageTaskName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/calc\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(source, []byte("package calc\nfunc Add(a, b int) int { return a + b }\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	testFile := filepath.Join(dir, "calc_test.go")
	existing := `package calc

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("unexpected Add result")
	}
}
`
	if err := os.WriteFile(testFile, []byte(existing), 0o644); err != nil {
		t.Fatalf("write existing test: %v", err)
	}
	task := types.CoverageTestTask{
		ID:        "go-test-1",
		Framework: "go-test",
		File:      source,
		Target:    "Add",
		Kind:      "function",
		LineRange: "2-2",
		TestFile:  testFile,
		TestName:  "TestAdd",
		Command:   "go test ./...",
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
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
	if out.CoverageTask == nil || out.CoverageTask.TestName != "TestAddCoverage2_2" {
		t.Fatalf("validation coverage task test_name = %+v, want adjusted name", out.CoverageTask)
	}
	if out.Generated == nil || out.Generated.CoverageTask == nil || out.Generated.CoverageTask.TestName != out.CoverageTask.TestName {
		t.Fatalf("generated coverage task was not aligned with validation output: %+v", out.Generated)
	}
}

func TestHandleValidateCoverageTaskMarksUnreachableSkippedTask(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/remote\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "remote.go")
	src := `package remote

import (
	"net/http"
	"strings"
)

func RemoteIP(r *http.Request, fallback string) string {
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		partIndex := len(parts) - 1
		if partIndex < 0 {
			partIndex = 0
		}
		return parts[partIndex]
	}
	return fallback
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-unreachable",
		Framework:       "go-test",
		File:            source,
		Target:          "RemoteIP",
		Kind:            "function",
		LineRange:       "12-14",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: partIndex < 0"},
		SuggestedInputs: []string{"构造满足条件 `partIndex < 0` 的输入"},
		Goal:            "为 RemoteIP 补充测试，覆盖不可达行段",
		Command:         "go test ./...",
		TestFile:        filepath.Join(dir, "remote_test.go"),
		TestName:        "TestRemoteIPPartIndex",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
		Confidence:      0.95,
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
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
	if out.Status != "passed" || out.Action != "manual_review_unreachable" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.RunResult == nil || out.RunResult.Skipped == 0 {
		t.Fatalf("expected skipped generated TODO test, got run result: %+v", out.RunResult)
	}
	if out.Metadata["unreachable"] != true {
		t.Fatalf("expected unreachable metadata, got %+v", out.Metadata)
	}
	reason, _ := out.Metadata["unreachable_reason"].(string)
	if !strings.Contains(reason, "partIndex < 0") {
		t.Fatalf("unexpected unreachable reason: %q", reason)
	}
}

func TestHandleValidateCoverageTaskMarksInitAsManualReview(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "main.go")
	src := `package app

var initialized bool

func init() {
	initialized = true
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-init",
		Framework:       "go-test",
		File:            source,
		Target:          "init",
		Kind:            "function",
		LineRange:       "5-5",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: err != nil"},
		Goal:            "复核 init 初始化路径",
		Command:         "go test ./...",
		TestFile:        filepath.Join(dir, "main_test.go"),
		TestName:        "TestInit",
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
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
	if out.Status != "passed" || out.Action != "manual_review_unreachable" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.RunResult == nil || out.RunResult.Skipped == 0 {
		t.Fatalf("expected skipped init review test, got run result: %+v", out.RunResult)
	}
	reason, _ := out.Metadata["unreachable_reason"].(string)
	if !strings.Contains(reason, "init functions cannot be called directly") {
		t.Fatalf("unexpected init manual-review reason: %q", reason)
	}
	if out.Generated == nil || strings.Contains(out.Generated.Preview, "\t\t\tinit()\n") {
		t.Fatalf("generated init task should not call init directly: %+v", out.Generated)
	}
}

func TestHandleValidateCoverageTaskMarksSystemResourceErrorAsEnvironmentReview(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/sys\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "server.go")
	src := `package sys

type Cpu struct {
	Cores int
}

func InitCPU() (c Cpu, err error) {
	if err != nil {
		return c, err
	}
	c.Cores = 1
	return c, nil
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-init-cpu",
		Framework:       "go-test",
		File:            source,
		Target:          "InitCPU",
		Kind:            "function",
		LineRange:       "8-10",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: err != nil"},
		SuggestedInputs: []string{"构造满足条件 `err != nil` 的输入"},
		Goal:            "为 InitCPU 补充测试，覆盖系统资源错误分支",
		Command:         "go test ./...",
		TestFile:        filepath.Join(dir, "server_test.go"),
		TestName:        "TestInitCPU",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
		Confidence:      0.95,
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
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
	if out.Status != "passed" || out.Action != "manual_review_environment" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.RunResult == nil || out.RunResult.Skipped == 0 {
		t.Fatalf("expected skipped generated TODO test, got run result: %+v", out.RunResult)
	}
	if out.Metadata["environment_dependent"] != true {
		t.Fatalf("expected environment metadata, got %+v", out.Metadata)
	}
	reason, _ := out.Metadata["environment_reason"].(string)
	if !strings.Contains(reason, "InitCPU") || !strings.Contains(reason, "static tests cannot force it") {
		t.Fatalf("unexpected environment reason: %q", reason)
	}
}

func TestHandleValidateCoverageTaskMarksSocketWriteErrorAsProtocolReview(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/protocol\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "client.go")
	src := `package protocol

import (
	"encoding/json"
	"net"
)

type BindRequest struct {
	Control string ` + "`json:\"control\"`" + `
}

type Status struct{}

func QueryStatus(socketPath string) (Status, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return Status{}, err
	}
	bind, _ := json.Marshal(BindRequest{Control: "status"})
	if _, err := conn.Write(append(bind, '\n')); err != nil {
		return Status{}, err
	}
	return Status{}, nil
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-query-status-write",
		Framework:       "go-test",
		File:            source,
		Target:          "QueryStatus",
		Kind:            "function",
		LineRange:       "19-21",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: err != nil"},
		SuggestedInputs: []string{"构造满足条件 `err != nil` 的输入"},
		Goal:            "为 QueryStatus 补充 socket 写入错误分支",
		Command:         "go test ./...",
		TestFile:        filepath.Join(dir, "client_test.go"),
		TestName:        "TestQueryStatusWriteError",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
		Confidence:      0.95,
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
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
	if out.Status != "passed" || out.Action != "manual_review_protocol" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.RunResult == nil || out.RunResult.Skipped == 0 {
		t.Fatalf("expected skipped generated TODO test, got run result: %+v", out.RunResult)
	}
	if out.Metadata["protocol_dependent"] != true {
		t.Fatalf("expected protocol metadata, got %+v", out.Metadata)
	}
	reason, _ := out.Metadata["protocol_reason"].(string)
	if !strings.Contains(reason, "QueryStatus") || !strings.Contains(reason, "socket write") {
		t.Fatalf("unexpected protocol reason: %q", reason)
	}
}

func TestHandleValidateCoverageTaskMarksRepoDBBranchAsDatabaseReview(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/repo\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	source := filepath.Join(dir, "repo.go")
	src := `package repo

import "errors"

type CigaretteRepo struct{}

func (r *CigaretteRepo) query(userID int64) error {
	return nil
}

func (r *CigaretteRepo) ListByUserID(userID int64) ([]int, error) {
	err := r.query(userID)
	if err != nil {
		return nil, err
	}
	return []int{1}, nil
}

var _ = errors.New
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-repo-db",
		Framework:       "go-test",
		File:            source,
		Target:          "CigaretteRepo.ListByUserID",
		Kind:            "method",
		LineRange:       "12-14",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: result.Error != nil"},
		SuggestedInputs: []string{"构造满足条件 `result.Error != nil` 的输入"},
		Goal:            "为 CigaretteRepo.ListByUserID 补充数据库错误分支",
		Command:         "go test ./...",
		TestFile:        filepath.Join(dir, "repo_test.go"),
		TestName:        "TestCigaretteRepoListByUserID",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
		Confidence:      0.95,
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
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
	if out.Status != "passed" || out.Action != "manual_review_database" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.RunResult == nil || out.RunResult.Skipped == 0 {
		t.Fatalf("expected skipped generated TODO test, got run result: %+v", out.RunResult)
	}
	if out.Metadata["database_dependent"] != true {
		t.Fatalf("expected database metadata, got %+v", out.Metadata)
	}
	reason, _ := out.Metadata["database_reason"].(string)
	if !strings.Contains(reason, "CigaretteRepo.ListByUserID") || !strings.Contains(reason, "GORM DB behavior") {
		t.Fatalf("unexpected database reason: %q", reason)
	}
}

func TestHandleValidateCoverageTaskMarksJSPrivateMethodAsManualReview(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module","scripts":{"test":"vitest run"},"devDependencies":{"vitest":"^3.0.0"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	source := filepath.Join(dir, "config.js")
	src := `export class ConfigManager {
  #diffConfigs(oldServers, newServers) {
    for (const name of Object.keys(oldServers)) {
      if (!newServers[name]) {
        return true
      }
    }
    return false
  }

  loadConfig() {
    return this.#diffConfigs({ old: { command: 'node' } }, {})
  }
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	installFakeNpx(t, strings.Join([]string{
		" RUN  v3.2.4 " + dir,
		"",
		" ↓ config.test.js (1 test | 1 skipped)",
		"",
		" Test Files  1 skipped (1)",
		"      Tests  1 skipped (1)",
	}, "\n"))
	task := types.CoverageTestTask{
		ID:              "vitest-private-1",
		Framework:       "vitest",
		File:            source,
		Target:          "ConfigManager.#diffConfigs",
		Kind:            "method",
		LineRange:       "4-4",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: !newServers[name]"},
		SuggestedInputs: []string{"构造满足条件 `!newServers[name]` 的输入"},
		Goal:            "为 ConfigManager.#diffConfigs 补充私有方法分支测试",
		Command:         "npx vitest run config.js",
		TestFile:        filepath.Join(dir, "config.test.js"),
		TestName:        "covers ConfigManager private diff",
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
		Confidence:      0.95,
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		Framework:    "vitest",
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
	if out.Status != "passed" || out.Action != "manual_review_private" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.RunResult == nil || out.RunResult.Status != "pass" {
		t.Fatalf("expected passing private review test run, got %+v", out.RunResult)
	}
	if out.Metadata["private_method"] != true {
		t.Fatalf("expected private method metadata, got %+v", out.Metadata)
	}
	reason, _ := out.Metadata["private_reason"].(string)
	if !strings.Contains(reason, "ConfigManager.#diffConfigs") || !strings.Contains(reason, "private method") {
		t.Fatalf("unexpected private reason: %q", reason)
	}
	entries, ok := out.Metadata["public_entry_candidates"].([]any)
	if !ok || len(entries) != 1 || entries[0] != "ConfigManager.loadConfig" {
		t.Fatalf("unexpected public entry candidates: %+v", out.Metadata["public_entry_candidates"])
	}
	if out.Generated == nil || !strings.Contains(out.Generated.Preview, "manual_review_private: ConfigManager.#diffConfigs") ||
		!strings.Contains(out.Generated.Preview, "public_entry_candidates: ConfigManager.loadConfig") ||
		strings.Contains(out.Generated.Preview, "instance.#diffConfigs") {
		t.Fatalf("expected generated private method review skip, got %+v", out.Generated)
	}
}

func TestHandleValidateCoverageTaskMarksJSInternalClassAsManualReview(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module","scripts":{"test":"vitest run"},"devDependencies":{"vitest":"^3.0.0"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	source := filepath.Join(dir, "oauth-provider.js")
	src := `class StorageManager {
  get(serverUrl) {
    if (!serversStorage[serverUrl]) {
      serversStorage[serverUrl] = { tokens: null }
    }
    return serversStorage[serverUrl]
  }
}

const storage = new StorageManager()

export default class MCPHubOAuthProvider {
  async tokens() {
    return storage.get(this.serverUrl).tokens
  }
}
`
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	installFakeNpx(t, strings.Join([]string{
		" RUN  v3.2.4 " + dir,
		"",
		" ↓ oauth-provider.test.js (1 test | 1 skipped)",
		"",
		" Test Files  1 skipped (1)",
		"      Tests  1 skipped (1)",
	}, "\n"))
	task := types.CoverageTestTask{
		ID:              "vitest-internal-1",
		Framework:       "vitest",
		File:            source,
		Target:          "StorageManager.get",
		Kind:            "method",
		LineRange:       "3-3",
		GapType:         "branch",
		MissingBranches: []string{"未覆盖 if 分支: !serversStorage[serverUrl]"},
		SuggestedInputs: []string{"构造满足条件 `!serversStorage[serverUrl]` 的输入"},
		TestFile:        filepath.Join(dir, "oauth-provider.test.js"),
		TestName:        "covers StorageManager internal get",
	}

	result, _, err := HandleValidateCoverageTask(context.Background(), nil, validateCoverageTaskInput{
		FilePath:     source,
		Framework:    "vitest",
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
	if out.Status != "passed" || out.Action != "manual_review_internal" {
		t.Fatalf("unexpected validation output: %+v", out)
	}
	if out.Metadata["internal_symbol"] != true {
		t.Fatalf("expected internal symbol metadata, got %+v", out.Metadata)
	}
	reason, _ := out.Metadata["internal_reason"].(string)
	if !strings.Contains(reason, "StorageManager.get") || !strings.Contains(reason, "not exported") {
		t.Fatalf("unexpected internal reason: %q", reason)
	}
	if out.Generated == nil || !strings.Contains(out.Generated.Preview, "manual_review_internal: StorageManager") ||
		strings.Contains(out.Generated.Preview, "new StorageManager()") {
		t.Fatalf("expected generated internal review skip, got %+v", out.Generated)
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
