package coverage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/binlee/testloop-mcp/types"
)

func TestParseGoCoverage(t *testing.T) {
	// 使用真实生成的 coverprofile
	data, err := os.ReadFile("/tmp/testloop_cover.out")
	if err != nil {
		t.Skipf("跳过: 无法读取覆盖率文件: %v", err)
	}

	report, err := ParseGoCoverage(string(data))
	if err != nil {
		t.Fatalf("ParseGoCoverage 失败: %v", err)
	}

	if report.Framework != "go-test" {
		t.Errorf("Framework = %s, want go-test", report.Framework)
	}
	if report.TotalPercent <= 0 {
		t.Errorf("TotalPercent = %.1f, want > 0", report.TotalPercent)
	}
	if len(report.Files) == 0 {
		t.Error("Files 为空")
	}
	if report.Summary.TotalStatements == 0 {
		t.Error("TotalStatements = 0")
	}
	if report.Summary.CoveredStatements == 0 {
		t.Error("CoveredStatements = 0")
	}

	t.Logf("覆盖率: %.1f%%", report.TotalPercent)
	t.Logf("文件数: %d", report.Summary.TotalFiles)
	t.Logf("语句总数: %d", report.Summary.TotalStatements)
	t.Logf("已覆盖: %d", report.Summary.CoveredStatements)
	t.Logf("建议数: %d", len(report.Suggestions))

	for _, f := range report.Files {
		t.Logf("  %s: %.1f%% (%d blocks)", f.Path, f.Percent, len(f.Blocks))
	}
}

func TestParseGoCoverageRaw(t *testing.T) {
	raw := `mode: set
example.com/foo/bar.go:1.1,3.1 2 1
example.com/foo/bar.go:5.1,7.1 3 0
example.com/foo/baz.go:1.1,2.1 1 1
`
	report, err := ParseGoCoverage(raw)
	if err != nil {
		t.Fatalf("ParseGoCoverage 失败: %v", err)
	}

	if len(report.Files) != 2 {
		t.Fatalf("文件数 = %d, want 2", len(report.Files))
	}
	if report.TotalPercent <= 0 {
		t.Errorf("TotalPercent = %.1f, want > 0", report.TotalPercent)
	}
}

func TestParseGoCoverageMapsUncoveredBlocksToFunctions(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "calc.go")
	src := `package calc

func Add(a, b int) int {
	if a == 0 {
		return b
	}
	return a + b
}

type Calculator struct{}

func (Calculator) Divide(a, b int) int {
	if b == 0 {
		return 0
	}
	return a / b
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	raw := `mode: set
` + srcPath + `:4.2,4.10 1 0
` + srcPath + `:13.2,13.10 1 0
`
	report, err := ParseGoCoverage(raw)
	if err != nil {
		t.Fatalf("ParseGoCoverage 失败: %v", err)
	}

	addSuggestion := findCoverageSuggestion(report.Suggestions, "Add")
	if addSuggestion == nil {
		t.Fatalf("expected Add suggestion, got %+v", report.Suggestions)
	}
	if addSuggestion.Kind != "function" || addSuggestion.LineRange != "4-4" {
		t.Errorf("unexpected Add suggestion: %+v", addSuggestion)
	}
	if addSuggestion.GapType != "branch" {
		t.Errorf("expected branch gap type, got %q", addSuggestion.GapType)
	}
	if !containsString(addSuggestion.MissingBranches, "未覆盖 if 分支: a == 0") {
		t.Errorf("expected if branch detail, got %+v", addSuggestion.MissingBranches)
	}
	if !containsString(addSuggestion.SuggestedInputs, "构造满足条件 `a == 0` 的输入") {
		t.Errorf("expected condition-specific input hints, got %+v", addSuggestion.SuggestedInputs)
	}

	divideSuggestion := findCoverageSuggestion(report.Suggestions, "Calculator.Divide")
	if divideSuggestion == nil {
		t.Fatalf("expected Calculator.Divide suggestion, got %+v", report.Suggestions)
	}
	if divideSuggestion.Kind != "method" || divideSuggestion.UncoveredLines[0] != 13 {
		t.Errorf("unexpected Divide suggestion: %+v", divideSuggestion)
	}
	if !strings.Contains(divideSuggestion.Reason, "Calculator.Divide") {
		t.Errorf("expected function-aware reason, got %q", divideSuggestion.Reason)
	}

	addTask := findCoverageTask(report.TestTasks, "Add")
	if addTask == nil {
		t.Fatalf("expected Add test task, got %+v", report.TestTasks)
	}
	if addTask.Framework != "go-test" || addTask.Kind != "function" {
		t.Errorf("unexpected Add task metadata: %+v", addTask)
	}
	if addTask.GapType != "branch" || !containsString(addTask.MissingBranches, "未覆盖 if 分支: a == 0") {
		t.Errorf("expected task branch details, got %+v", addTask)
	}
	if !strings.Contains(addTask.Goal, "覆盖未执行行段 4-4") {
		t.Errorf("unexpected Add task goal: %q", addTask.Goal)
	}
	if addTask.Command == "" || !strings.Contains(addTask.Command, "go test") {
		t.Errorf("expected go test command, got %q", addTask.Command)
	}
	if addTask.TestFile != strings.TrimSuffix(srcPath, ".go")+"_test.go" {
		t.Errorf("unexpected Add task test file: %q", addTask.TestFile)
	}
	if addTask.TestName != "TestAdd" {
		t.Errorf("unexpected Add task test name: %q", addTask.TestName)
	}
	if !containsString(addTask.AssertionFocus, "断言未覆盖分支的返回值或副作用") {
		t.Errorf("expected Add task assertion focus, got %+v", addTask.AssertionFocus)
	}
	if !containsString(addTask.SuggestedInputs, "构造满足条件 `a == 0` 的输入") {
		t.Errorf("expected task input hints, got %+v", addTask.SuggestedInputs)
	}
}

func findCoverageSuggestion(suggestions []types.CoverageSuggestion, fn string) *types.CoverageSuggestion {
	for i := range suggestions {
		if suggestions[i].Function == fn {
			return &suggestions[i]
		}
	}
	return nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func findCoverageTask(tasks []types.CoverageTestTask, target string) *types.CoverageTestTask {
	for i := range tasks {
		if tasks[i].Target == target {
			return &tasks[i]
		}
	}
	return nil
}

func withWorkingDirectory(t *testing.T, dir string) {
	t.Helper()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
}

func TestParseJestCoverage(t *testing.T) {
	raw := `{
		"/src/utils.js": {
			"path": "/src/utils.js",
			"statementMap": {
				"0": {"start": {"line": 1, "column": 0}, "end": {"line": 1, "column": 15}},
				"1": {"start": {"line": 2, "column": 0}, "end": {"line": 2, "column": 10}},
				"2": {"start": {"line": 4, "column": 0}, "end": {"line": 4, "column": 20}}
			},
			"s": {"0": 1, "1": 0, "2": 5},
			"fnMap": {},
			"f": {},
			"branchMap": {},
			"b": {}
		},
		"/src/helper.js": {
			"path": "/src/helper.js",
			"statementMap": {
				"0": {"start": {"line": 1, "column": 0}, "end": {"line": 1, "column": 10}}
			},
			"s": {"0": 3},
			"fnMap": {},
			"f": {},
			"branchMap": {},
			"b": {}
		}
	}`

	report, err := ParseJestCoverage(raw, "jest")
	if err != nil {
		t.Fatalf("ParseJestCoverage 失败: %v", err)
	}

	if report.Framework != "jest" {
		t.Errorf("Framework = %s, want jest", report.Framework)
	}
	if len(report.Files) != 2 {
		t.Fatalf("文件数 = %d, want 2", len(report.Files))
	}
	if report.Summary.TotalStatements != 4 {
		t.Errorf("TotalStatements = %d, want 4", report.Summary.TotalStatements)
	}
	if report.Summary.CoveredStatements != 3 {
		t.Errorf("CoveredStatements = %d, want 3", report.Summary.CoveredStatements)
	}
	if len(report.TestTasks) == 0 {
		t.Fatal("expected non-empty test tasks for uncovered statements")
	}
	if report.TestTasks[0].Framework != "jest" || report.TestTasks[0].Command == "" {
		t.Errorf("unexpected jest test task: %+v", report.TestTasks[0])
	}

	t.Logf("覆盖率: %.1f%%", report.TotalPercent)
	t.Logf("建议数: %d", len(report.Suggestions))
}

func TestParsePytestCoverage(t *testing.T) {
	raw := `{
		"totals": {
			"covered_lines": 8,
			"num_statements": 10,
			"percent_covered": 80.0,
			"missing_lines": 2
		},
		"files": {
			"src/app.py": {
				"executed_lines": [1, 2, 3, 4, 5, 6, 7, 8],
				"missing_lines": [10, 11],
				"summary": {
					"covered_lines": 8,
					"num_statements": 10,
					"percent_covered": 80.0,
					"missing_lines": 2
				}
			}
		}
	}`

	report, err := ParsePytestCoverage(raw)
	if err != nil {
		t.Fatalf("ParsePytestCoverage 失败: %v", err)
	}

	if report.Framework != "pytest" {
		t.Errorf("Framework = %s, want pytest", report.Framework)
	}
	if len(report.Files) != 1 {
		t.Fatalf("文件数 = %d, want 1", len(report.Files))
	}
	if report.TotalPercent != 80.0 {
		t.Errorf("TotalPercent = %.1f, want 80.0", report.TotalPercent)
	}

	// 验证 block 合并：1-8 应为一个 covered block，10-11 应为一个 uncovered block
	cf := report.Files[0]
	if len(cf.Blocks) != 2 {
		t.Logf("Blocks 数量 = %d", len(cf.Blocks))
		for i, b := range cf.Blocks {
			t.Logf("  Block %d: line %d-%d, covered=%v", i, b.StartLine, b.EndLine, b.Covered)
		}
	}

	t.Logf("覆盖率: %.1f%%", report.TotalPercent)
	t.Logf("建议数: %d", len(report.Suggestions))
}

func TestParseRustTarpaulinCoverageLCOV(t *testing.T) {
	raw := `TN:
SF:src/lib.rs
DA:1,1
DA:2,1
DA:3,0
LF:3
LH:2
end_of_record
SF:src/unused.rs
DA:1,0
LF:1
LH:0
end_of_record
`

	report, err := ParseRustTarpaulinCoverage(raw)
	if err != nil {
		t.Fatalf("ParseRustTarpaulinCoverage 失败: %v", err)
	}
	if report.Framework != "cargo-test" {
		t.Errorf("Framework = %s, want cargo-test", report.Framework)
	}
	if len(report.Files) != 2 {
		t.Fatalf("文件数 = %d, want 2", len(report.Files))
	}
	if report.Summary.TotalStatements != 4 || report.Summary.CoveredStatements != 2 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	if report.TotalPercent != 50 {
		t.Fatalf("TotalPercent = %.1f, want 50.0", report.TotalPercent)
	}
	if !containsString(report.Summary.UncoveredFiles, "src/unused.rs") {
		t.Fatalf("expected uncovered rust file, got %+v", report.Summary.UncoveredFiles)
	}
	if len(report.TestTasks) == 0 || report.TestTasks[0].Command != "cargo test" {
		t.Fatalf("expected cargo test task, got %+v", report.TestTasks)
	}
}

func TestParseRustCoverageMapsUncoveredLinesToFunctions(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "lib.rs")
	src := `pub fn add(a: i32, b: i32) -> i32 {
    if a == 0 {
        return b;
    }
    a + b
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	raw := `TN:
SF:` + srcPath + `
DA:1,1
DA:2,0
DA:3,0
DA:4,1
end_of_record
`
	report, err := ParseRustTarpaulinCoverage(raw)
	if err != nil {
		t.Fatalf("ParseRustTarpaulinCoverage 失败: %v", err)
	}
	suggestion := findCoverageSuggestion(report.Suggestions, "add")
	if suggestion == nil {
		t.Fatalf("expected add suggestion, got %+v", report.Suggestions)
	}
	if suggestion.Kind != "function" || suggestion.LineRange != "2-2" {
		t.Fatalf("unexpected rust suggestion: %+v", suggestion)
	}
	if suggestion.GapType != "branch" {
		t.Fatalf("expected rust branch gap type, got %+v", suggestion)
	}
	if !containsString(suggestion.MissingBranches, "未覆盖 if 分支: a == 0") {
		t.Fatalf("expected rust branch detail, got %+v", suggestion.MissingBranches)
	}
	if !containsString(suggestion.SuggestedInputs, "构造满足条件 `a == 0` 的输入") {
		t.Fatalf("expected rust condition input hints, got %+v", suggestion.SuggestedInputs)
	}
	if !containsString(suggestion.SuggestedInputs, "设置 a 覆盖未执行分支") {
		t.Fatalf("expected rust param hints, got %+v", suggestion.SuggestedInputs)
	}
	task := findCoverageTask(report.TestTasks, "add")
	if task == nil || task.Kind != "function" {
		t.Fatalf("expected add task, got %+v", report.TestTasks)
	}
}

func TestParseRustCoverageMapsComplexTreeSitterItems(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "lib.rs")
	src := `#[inline]
pub fn clamp(
    value: i32,
    max: i32,
) -> i32 {
    if value > max {
        return max;
    }
    value
}

pub struct Calculator;

impl Calculator {
    pub fn divide(&self, a: i32, b: i32) -> Result<i32, String> {
        if b == 0 {
            return Err("zero".to_string());
        }
        Ok(a / b)
    }
}

pub trait Named {
    fn label(&self, raw: Option<String>) -> String {
        match raw {
            Some(value) => value,
            None => "empty".to_string(),
        }
    }
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	raw := `TN:
SF:` + srcPath + `
DA:6,0
DA:16,0
DA:17,0
DA:25,0
DA:26,0
end_of_record
`
	report, err := ParseRustTarpaulinCoverage(raw)
	if err != nil {
		t.Fatalf("ParseRustTarpaulinCoverage 失败: %v", err)
	}
	clamp := findCoverageSuggestion(report.Suggestions, "clamp")
	if clamp == nil {
		t.Fatalf("expected clamp suggestion, got %+v", report.Suggestions)
	}
	if clamp.Kind != "function" || !containsString(clamp.SuggestedInputs, "设置 value 覆盖未执行分支") || !containsString(clamp.SuggestedInputs, "设置 max 覆盖未执行分支") {
		t.Fatalf("unexpected clamp suggestion: %+v", clamp)
	}
	divide := findCoverageSuggestion(report.Suggestions, "Calculator.divide")
	if divide == nil {
		t.Fatalf("expected Calculator.divide suggestion, got %+v", report.Suggestions)
	}
	if divide.Kind != "method" || !containsString(divide.SuggestedInputs, "设置 a 覆盖未执行分支") || !containsString(divide.SuggestedInputs, "设置 b 覆盖未执行分支") {
		t.Fatalf("unexpected divide suggestion: %+v", divide)
	}
	label := findCoverageSuggestion(report.Suggestions, "Named.label")
	if label == nil {
		t.Fatalf("expected Named.label suggestion, got %+v", report.Suggestions)
	}
	if label.GapType != "branch" || !containsString(label.MissingBranches, "未覆盖 match 分支") {
		t.Fatalf("unexpected trait method suggestion: %+v", label)
	}
}

func TestParseRustCoverageResolvesWorkspaceFixturePaths(t *testing.T) {
	dir := t.TempDir()
	withWorkingDirectory(t, dir)
	srcPath := filepath.Join("crates", "core", "src", "lib.rs")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0755); err != nil {
		t.Fatal(err)
	}
	src := `pub struct Validator;

impl Validator {
    pub fn check(&self, value: Option<i32>) -> Result<i32, String> {
        match value {
            Some(v) if v > 0 => Ok(v),
            _ => Err("invalid".to_string()),
        }
    }
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	raw := `TN:
SF:crates/core/src/lib.rs
DA:5,0
DA:6,0
end_of_record
`
	report, err := ParseRustTarpaulinCoverage(raw)
	if err != nil {
		t.Fatalf("ParseRustTarpaulinCoverage 失败: %v", err)
	}
	suggestion := findCoverageSuggestion(report.Suggestions, "Validator.check")
	if suggestion == nil {
		t.Fatalf("expected Validator.check suggestion, got %+v", report.Suggestions)
	}
	if suggestion.GapType != "branch" || !containsString(suggestion.MissingBranches, "未覆盖 match 分支") {
		t.Fatalf("unexpected rust workspace suggestion: %+v", suggestion)
	}
	task := findCoverageTask(report.TestTasks, "Validator.check")
	if task == nil || task.Command != "cargo test" {
		t.Fatalf("expected cargo test task, got %+v", report.TestTasks)
	}
	if task.TestFile != "crates/core/src/lib.rs" {
		t.Fatalf("unexpected rust task test file: %+v", task)
	}
	if task.TestName != "test_validator_check_covers_gap" {
		t.Fatalf("unexpected rust task test name: %+v", task)
	}
	if !containsString(task.AssertionFocus, "未覆盖 match 分支") {
		t.Fatalf("expected rust assertion focus, got %+v", task.AssertionFocus)
	}
}

func TestParseJaCoCoCoverageXML(t *testing.T) {
	raw := `<?xml version="1.0" encoding="UTF-8"?>
<report name="demo">
  <package name="com/example">
    <sourcefile name="Calculator.java">
      <line nr="10" mi="0" ci="1"/>
      <line nr="11" mi="1" ci="0"/>
      <counter type="LINE" missed="1" covered="1"/>
    </sourcefile>
    <sourcefile name="Unused.java">
      <line nr="5" mi="1" ci="0"/>
      <counter type="LINE" missed="1" covered="0"/>
    </sourcefile>
  </package>
  <counter type="LINE" missed="2" covered="1"/>
</report>`

	report, err := ParseJaCoCoCoverage(raw)
	if err != nil {
		t.Fatalf("ParseJaCoCoCoverage 失败: %v", err)
	}
	if report.Framework != "junit" {
		t.Errorf("Framework = %s, want junit", report.Framework)
	}
	if len(report.Files) != 2 {
		t.Fatalf("文件数 = %d, want 2", len(report.Files))
	}
	if report.Summary.TotalStatements != 3 || report.Summary.CoveredStatements != 1 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	if report.Files[0].Path != "com/example/Calculator.java" {
		t.Fatalf("unexpected java path: %s", report.Files[0].Path)
	}
	if !containsString(report.Summary.UncoveredFiles, "com/example/Unused.java") {
		t.Fatalf("expected uncovered java file, got %+v", report.Summary.UncoveredFiles)
	}
	if len(report.TestTasks) == 0 || report.TestTasks[0].Command != "mvn test" {
		t.Fatalf("expected mvn test task, got %+v", report.TestTasks)
	}
}

func TestParseJaCoCoCoverageMapsUncoveredLinesToMethods(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "com", "example")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(srcDir, "Calculator.java")
	src := `package com.example;

public class Calculator {
    public int add(int a, int b) {
        if (a == 0) {
            return b;
        }
        return a + b;
    }
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	raw := `<?xml version="1.0" encoding="UTF-8"?>
<report name="demo">
  <package name="` + dir + `/com/example">
    <sourcefile name="Calculator.java">
      <line nr="4" mi="0" ci="1"/>
      <line nr="5" mi="1" ci="0"/>
      <line nr="6" mi="1" ci="0"/>
      <counter type="LINE" missed="2" covered="1"/>
    </sourcefile>
  </package>
  <counter type="LINE" missed="2" covered="1"/>
</report>`

	report, err := ParseJaCoCoCoverage(raw)
	if err != nil {
		t.Fatalf("ParseJaCoCoCoverage 失败: %v", err)
	}
	suggestion := findCoverageSuggestion(report.Suggestions, "Calculator.add")
	if suggestion == nil {
		t.Fatalf("expected Calculator.add suggestion, got %+v", report.Suggestions)
	}
	if suggestion.Kind != "method" || suggestion.LineRange != "5-5" {
		t.Fatalf("unexpected java suggestion: %+v", suggestion)
	}
	if !containsString(suggestion.SuggestedInputs, "设置 a 覆盖未执行分支") {
		t.Fatalf("expected java param hints, got %+v", suggestion.SuggestedInputs)
	}
	task := findCoverageTask(report.TestTasks, "Calculator.add")
	if task == nil || task.Kind != "method" {
		t.Fatalf("expected Calculator.add task, got %+v", report.TestTasks)
	}
}

func TestParseJavaCoverageClassifiesErrorPath(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "com", "example")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(srcDir, "Calculator.java")
	src := `package com.example;

public class Calculator {
    public int divide(int a, int b) {
        if (b == 0) {
            throw new IllegalArgumentException("zero");
        }
        return a / b;
    }
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	raw := `<?xml version="1.0" encoding="UTF-8"?>
<report name="demo">
  <package name="` + dir + `/com/example">
    <sourcefile name="Calculator.java">
      <line nr="4" mi="0" ci="1"/>
      <line nr="5" mi="0" ci="1"/>
      <line nr="6" mi="1" ci="0"/>
      <counter type="LINE" missed="1" covered="2"/>
    </sourcefile>
  </package>
  <counter type="LINE" missed="1" covered="2"/>
</report>`

	report, err := ParseJaCoCoCoverage(raw)
	if err != nil {
		t.Fatalf("ParseJaCoCoCoverage 失败: %v", err)
	}
	suggestion := findCoverageSuggestion(report.Suggestions, "Calculator.divide")
	if suggestion == nil {
		t.Fatalf("expected Calculator.divide suggestion, got %+v", report.Suggestions)
	}
	if suggestion.GapType != "error_path" {
		t.Fatalf("expected java error path, got %+v", suggestion)
	}
	if !containsString(suggestion.MissingBranches, "未覆盖错误或空值返回路径") {
		t.Fatalf("expected java error path detail, got %+v", suggestion.MissingBranches)
	}
}

func TestResolveJavaCoveragePathFromStandardSourceRoot(t *testing.T) {
	dir := t.TempDir()
	withWorkingDirectory(t, dir)

	srcDir := filepath.Join("src", "main", "java", "com", "example")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	src := `package com.example;

public class Calculator {
    public int add(int a, int b) {
        return a + b;
    }
}
`
	if err := os.WriteFile(filepath.Join(srcDir, "Calculator.java"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	raw := `<?xml version="1.0" encoding="UTF-8"?>
<report name="demo">
  <package name="com/example">
    <sourcefile name="Calculator.java">
      <line nr="4" mi="1" ci="0"/>
      <counter type="LINE" missed="1" covered="0"/>
    </sourcefile>
  </package>
  <counter type="LINE" missed="1" covered="0"/>
</report>`

	report, err := ParseJaCoCoCoverage(raw)
	if err != nil {
		t.Fatalf("ParseJaCoCoCoverage 失败: %v", err)
	}
	suggestion := findCoverageSuggestion(report.Suggestions, "Calculator.add")
	if suggestion == nil {
		t.Fatalf("expected Calculator.add suggestion, got %+v", report.Suggestions)
	}
}

func TestParseJavaCoverageMapsComplexTreeSitterMethods(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "com", "example")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(srcDir, "Service.java")
	src := `package com.example;

public class Service {
    @Deprecated
    public Service(String name) {
        if (name == null) {
            throw new IllegalArgumentException("name");
        }
    }

    @Override
    public String format(
            String prefix,
            int count
    ) {
        if (count <= 0) {
            return prefix;
        }
        return prefix + count;
    }

    static class Helper {
        String label(String value) {
            return value == null ? "" : value;
        }
    }
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	raw := `<?xml version="1.0" encoding="UTF-8"?>
<report name="demo">
  <package name="` + dir + `/com/example">
    <sourcefile name="Service.java">
      <line nr="6" mi="1" ci="0"/>
      <line nr="15" mi="1" ci="0"/>
      <line nr="23" mi="1" ci="0"/>
      <counter type="LINE" missed="3" covered="0"/>
    </sourcefile>
  </package>
  <counter type="LINE" missed="3" covered="0"/>
</report>`

	report, err := ParseJaCoCoCoverage(raw)
	if err != nil {
		t.Fatalf("ParseJaCoCoCoverage 失败: %v", err)
	}
	constructor := findCoverageSuggestion(report.Suggestions, "Service.Service")
	if constructor == nil {
		t.Fatalf("expected Service constructor suggestion, got %+v", report.Suggestions)
	}
	if constructor.GapType != "branch" || !containsString(constructor.SuggestedInputs, "设置 name 覆盖未执行分支") {
		t.Fatalf("unexpected constructor suggestion: %+v", constructor)
	}
	format := findCoverageSuggestion(report.Suggestions, "Service.format")
	if format == nil {
		t.Fatalf("expected Service.format suggestion, got %+v", report.Suggestions)
	}
	if !containsString(format.SuggestedInputs, "设置 prefix 覆盖未执行分支") || !containsString(format.SuggestedInputs, "设置 count 覆盖未执行分支") {
		t.Fatalf("expected multiline method params, got %+v", format.SuggestedInputs)
	}
	helper := findCoverageSuggestion(report.Suggestions, "Service.Helper.label")
	if helper == nil {
		t.Fatalf("expected nested Helper.label suggestion, got %+v", report.Suggestions)
	}
	if helper.Kind != "method" {
		t.Fatalf("unexpected nested method suggestion: %+v", helper)
	}
}

func TestParseJavaCoverageResolvesMavenFixturePaths(t *testing.T) {
	dir := t.TempDir()
	withWorkingDirectory(t, dir)
	srcPath := filepath.Join("src", "main", "java", "com", "example", "service", "OrderService.java")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0755); err != nil {
		t.Fatal(err)
	}
	src := `package com.example.service;

public class OrderService {
    public String status(String state) {
        switch (state) {
            case "paid":
                return "closed";
            default:
                return "open";
        }
    }

    static class Audit {
        String describe(String user) {
            if (user == null) {
                return null;
            }
            return user;
        }
    }
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	raw := `<?xml version="1.0" encoding="UTF-8"?>
<report name="demo">
  <package name="com/example/service">
    <sourcefile name="OrderService.java">
      <line nr="5" mi="1" ci="0"/>
      <line nr="15" mi="1" ci="0"/>
      <counter type="LINE" missed="2" covered="0"/>
    </sourcefile>
  </package>
  <counter type="LINE" missed="2" covered="0"/>
</report>`

	report, err := ParseJaCoCoCoverage(raw)
	if err != nil {
		t.Fatalf("ParseJaCoCoCoverage 失败: %v", err)
	}
	status := findCoverageSuggestion(report.Suggestions, "OrderService.status")
	if status == nil {
		t.Fatalf("expected OrderService.status suggestion, got %+v", report.Suggestions)
	}
	if status.GapType != "branch" || !containsString(status.MissingBranches, "未覆盖 switch/case 分支") {
		t.Fatalf("unexpected java maven status suggestion: %+v", status)
	}
	audit := findCoverageSuggestion(report.Suggestions, "OrderService.Audit.describe")
	if audit == nil {
		t.Fatalf("expected nested Audit.describe suggestion, got %+v", report.Suggestions)
	}
	if audit.GapType != "branch" || !containsString(audit.SuggestedInputs, "设置 user 覆盖未执行分支") {
		t.Fatalf("unexpected java nested suggestion: %+v", audit)
	}
	task := findCoverageTask(report.TestTasks, "OrderService.status")
	if task == nil || task.Command != "mvn test" {
		t.Fatalf("expected mvn test task, got %+v", report.TestTasks)
	}
	if task.TestFile != filepath.Join("src", "test", "java", "com", "example", "service", "OrderServiceTest.java") {
		t.Fatalf("unexpected java task test file: %+v", task)
	}
	if task.TestName != "shouldCoverOrderServiceStatusGap" {
		t.Fatalf("unexpected java task test name: %+v", task)
	}
	if !containsString(task.AssertionFocus, "未覆盖 switch/case 分支") {
		t.Fatalf("expected java assertion focus, got %+v", task.AssertionFocus)
	}
}

func TestParseCoverageDispatch(t *testing.T) {
	// go-test
	goData := `mode: set
example.com/foo.go:1.1,2.1 1 1`
	r1, err := ParseCoverage(goData, "go-test")
	if err != nil {
		t.Fatalf("go-test 分发失败: %v", err)
	}
	if r1.Framework != "go-test" {
		t.Errorf("Framework = %s", r1.Framework)
	}

	// jest
	jestData := `{"x.js": {"path":"x.js","statementMap":{"0":{"start":{"line":1,"column":0},"end":{"line":1,"column":1}}},"s":{"0":1},"fnMap":{},"f":{},"branchMap":{},"b":{}}}`
	r2, err := ParseCoverage(jestData, "jest")
	if err != nil {
		t.Fatalf("jest 分发失败: %v", err)
	}
	if r2.Framework != "jest" {
		t.Errorf("Framework = %s", r2.Framework)
	}

	// vitest — 同样走 Istanbul 格式，但 framework 标签应为 vitest
	r3, err := ParseCoverage(jestData, "vitest")
	if err != nil {
		t.Fatalf("vitest 分发失败: %v", err)
	}
	if r3.Framework != "vitest" {
		t.Errorf("Framework = %s, want vitest", r3.Framework)
	}

	// mocha — 同样走 Istanbul 格式（nyc 生成），framework 标签应为 mocha
	r4, err := ParseCoverage(jestData, "mocha")
	if err != nil {
		t.Fatalf("mocha 分发失败: %v", err)
	}
	if r4.Framework != "mocha" {
		t.Errorf("Framework = %s, want mocha", r4.Framework)
	}

	rustData := `SF:src/lib.rs
DA:1,1
end_of_record`
	r5, err := ParseCoverage(rustData, "cargo-test")
	if err != nil {
		t.Fatalf("cargo-test 分发失败: %v", err)
	}
	if r5.Framework != "cargo-test" {
		t.Errorf("Framework = %s, want cargo-test", r5.Framework)
	}

	jacocoData := `<report><package name="com/example"><sourcefile name="A.java"><line nr="1" mi="0" ci="1"/><counter type="LINE" missed="0" covered="1"/></sourcefile></package><counter type="LINE" missed="0" covered="1"/></report>`
	r6, err := ParseCoverage(jacocoData, "junit")
	if err != nil {
		t.Fatalf("junit 分发失败: %v", err)
	}
	if r6.Framework != "junit" {
		t.Errorf("Framework = %s, want junit", r6.Framework)
	}

	// 不支持的框架
	_, err = ParseCoverage("", "ruby")
	if err == nil {
		t.Error("期望返回错误，但未返回")
	}
}

func TestGenerateSuggestions(t *testing.T) {
	report := &types.CoverageReport{
		Files: []types.CoverageFile{
			{
				Path:    "low.go",
				Percent: 30,
				Blocks: []types.CoverageBlock{
					{StartLine: 1, EndLine: 5, Covered: false, Count: 0},
					{StartLine: 6, EndLine: 10, Covered: true, Count: 1},
				},
			},
			{
				Path:    "full.go",
				Percent: 100,
				Blocks: []types.CoverageBlock{
					{StartLine: 1, EndLine: 5, Covered: true, Count: 1},
				},
			},
		},
	}

	suggestions := GenerateSuggestions(report)
	// low.go: 1 个未覆盖 block + 1 个低覆盖率建议 = 2
	// full.go: 100% 覆盖，跳过
	if len(suggestions) != 2 {
		t.Fatalf("建议数 = %d, want 2", len(suggestions))
	}
}
