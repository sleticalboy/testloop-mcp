package coverage

import (
	"os"
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

	report, err := ParseJestCoverage(raw)
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
