package coverage

import (
	"encoding/json"
	"fmt"

	"github.com/sleticalboy/testloop-mcp/types"
)

// Jest / Istanbul coverage-final.json 结构
// {
//   "/abs/path/file.js": {
//     "path": "/abs/path/file.js",
//     "statementMap": {
//       "0": {"start": {"line": 1, "column": 0}, "end": {"line": 1, "column": 10}},
//       ...
//     },
//     "s": {"0": 1, "1": 0, ...},
//     "fnMap": {...},
//     "f": {...},
//     "branchMap": {...},
//     "b": {...}
//   }
// }

type istanbulPosition struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type istanbulRange struct {
	Start istanbulPosition `json:"start"`
	End   istanbulPosition `json:"end"`
}

type istanbulFileCoverage struct {
	Path         string                   `json:"path"`
	StatementMap map[string]istanbulRange `json:"statementMap"`
	S            map[string]int           `json:"s"` // statement hit counts
	FnMap        map[string]struct {
		Start istanbulPosition `json:"start"`
	} `json:"fnMap"`
	F map[string]int `json:"f"`
}

// ParseJestCoverage 解析 Jest / Vitest 的 coverage-final.json 格式
func ParseJestCoverage(profileData, framework string) (*types.CoverageReport, error) {
	// 尝试解析为 JSON
	var raw map[string]istanbulFileCoverage
	if err := json.Unmarshal([]byte(profileData), &raw); err != nil {
		return nil, fmt.Errorf("Jest 覆盖率数据不是有效的 JSON: %w", err)
	}

	if len(raw) == 0 {
		return nil, fmt.Errorf("覆盖率 JSON 为空")
	}

	var files []types.CoverageFile
	var totalStmts, coveredStmts int
	coveredFiles := 0
	var uncoveredFiles []string

	for _, fc := range raw {
		cf := types.CoverageFile{Path: fc.Path}

		fileStmts, fileCovered := 0, 0
		for id, hitCount := range fc.S {
			rng, ok := fc.StatementMap[id]
			if !ok {
				continue
			}
			covered := hitCount > 0
			cf.Blocks = append(cf.Blocks, types.CoverageBlock{
				StartLine: rng.Start.Line,
				EndLine:   rng.End.Line,
				Count:     hitCount,
				Covered:   covered,
			})
			fileStmts++
			if covered {
				fileCovered++
			}
		}

		cf.Percent = percent(fileCovered, fileStmts)
		if cf.Percent > 0 {
			coveredFiles++
		} else {
			uncoveredFiles = append(uncoveredFiles, cf.Path)
		}

		files = append(files, cf)
		totalStmts += fileStmts
		coveredStmts += fileCovered
	}

	report := &types.CoverageReport{
		Framework:    framework,
		TotalPercent: percent(coveredStmts, totalStmts),
		Files:        files,
		Summary: types.CoverageSummary{
			TotalStatements:   totalStmts,
			CoveredStatements: coveredStmts,
			TotalFiles:        len(files),
			CoveredFiles:      coveredFiles,
			UncoveredFiles:    uncoveredFiles,
		},
	}

	report.Suggestions = GenerateSuggestions(report)
	report.TestTasks = GenerateTestTasks(report)
	return report, nil
}
