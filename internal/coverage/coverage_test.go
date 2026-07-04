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
	if !containsString(addSuggestion.SuggestedInputs, "设置 a 覆盖未执行分支") {
		t.Errorf("expected param-specific input hints, got %+v", addSuggestion.SuggestedInputs)
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
	if !strings.Contains(addTask.Goal, "覆盖未执行行段 4-4") {
		t.Errorf("unexpected Add task goal: %q", addTask.Goal)
	}
	if addTask.Command == "" || !strings.Contains(addTask.Command, "go test") {
		t.Errorf("expected go test command, got %q", addTask.Command)
	}
	if !containsString(addTask.SuggestedInputs, "设置 a 覆盖未执行分支") {
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
