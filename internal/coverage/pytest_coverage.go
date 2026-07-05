package coverage

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/sleticalboy/testloop-mcp/types"
)

// coverage.py JSON 输出结构
// {
//   "totals": {
//     "covered_lines": 80,
//     "num_statements": 100,
//     "percent_covered": 80.0,
//     "missing_lines": 20
//   },
//   "files": {
//     "path/file.py": {
//       "executed_lines": [1, 2, 3, ...],
//       "missing_lines": [5, 6, ...],
//       "excluded_lines": [],
//       "summary": {
//         "covered_lines": 80,
//         "num_statements": 100,
//         "percent_covered": 80.0,
//         "missing_lines": 20
//       }
//     }
//   }
// }

type coveragePyFile struct {
	ExecutedLines []int `json:"executed_lines"`
	MissingLines  []int `json:"missing_lines"`
	Summary       struct {
		CoveredLines   int     `json:"covered_lines"`
		NumStatements  int     `json:"num_statements"`
		PercentCovered float64 `json:"percent_covered"`
		MissingLines   int     `json:"missing_lines"`
	} `json:"summary"`
}

type coveragePyJSON struct {
	Totals struct {
		CoveredLines   int     `json:"covered_lines"`
		NumStatements  int     `json:"num_statements"`
		PercentCovered float64 `json:"percent_covered"`
		MissingLines   int     `json:"missing_lines"`
	} `json:"totals"`
	Files map[string]coveragePyFile `json:"files"`
}

// ParsePytestCoverage 解析 coverage.py 生成的 JSON 覆盖率数据
func ParsePytestCoverage(profileData string) (*types.CoverageReport, error) {
	var cov coveragePyJSON
	if err := json.Unmarshal([]byte(profileData), &cov); err != nil {
		return nil, fmt.Errorf("pytest 覆盖率数据不是有效的 JSON: %w", err)
	}

	if len(cov.Files) == 0 {
		return nil, fmt.Errorf("覆盖率 JSON 中没有文件数据")
	}

	var files []types.CoverageFile
	coveredFiles := 0
	var uncoveredFiles []string

	for path, fc := range cov.Files {
		cf := types.CoverageFile{
			Path:    path,
			Percent: fc.Summary.PercentCovered,
		}

		// 合并已执行行和未覆盖行，构建 block
		lineHits := make(map[int]bool)
		for _, line := range fc.ExecutedLines {
			lineHits[line] = true
		}
		for _, line := range fc.MissingLines {
			if _, exists := lineHits[line]; !exists {
				lineHits[line] = false
			}
		}

		// 收集并排序所有行号
		allLines := make([]int, 0, len(lineHits))
		for line := range lineHits {
			allLines = append(allLines, line)
		}
		sort.Ints(allLines)

		// 将连续行合并为 block
		if len(allLines) > 0 {
			blockStart := allLines[0]
			blockEnd := allLines[0]
			blockCovered := lineHits[allLines[0]]

			for i := 1; i < len(allLines); i++ {
				line := allLines[i]
				covered := lineHits[line]

				// 连续且覆盖状态相同 → 扩展当前 block
				if line == blockEnd+1 && covered == blockCovered {
					blockEnd = line
				} else {
					// 输出上一个 block
					cf.Blocks = append(cf.Blocks, makeBlock(blockStart, blockEnd, blockCovered))
					blockStart = line
					blockEnd = line
					blockCovered = covered
				}
			}
			// 输出最后一个 block
			cf.Blocks = append(cf.Blocks, makeBlock(blockStart, blockEnd, blockCovered))
		}

		if cf.Percent > 0 {
			coveredFiles++
		} else {
			uncoveredFiles = append(uncoveredFiles, path)
		}

		files = append(files, cf)
	}

	report := &types.CoverageReport{
		Framework:    "pytest",
		TotalPercent: cov.Totals.PercentCovered,
		Files:        files,
		Summary: types.CoverageSummary{
			TotalStatements:   cov.Totals.NumStatements,
			CoveredStatements: cov.Totals.CoveredLines,
			TotalFiles:        len(files),
			CoveredFiles:      coveredFiles,
			UncoveredFiles:    uncoveredFiles,
		},
	}

	report.Suggestions = GenerateSuggestions(report)
	report.TestTasks = GenerateTestTasks(report)
	return report, nil
}

func makeBlock(start, end int, covered bool) types.CoverageBlock {
	count := 0
	if covered {
		count = 1
	}
	return types.CoverageBlock{
		StartLine: start,
		EndLine:   end,
		Count:     count,
		Covered:   covered,
	}
}
