package coverage

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
)

// ParseRustTarpaulinCoverage parses LCOV data produced by cargo tarpaulin.
// Recommended command: cargo tarpaulin --out Lcov --output-dir target/tarpaulin
func ParseRustTarpaulinCoverage(profileData string) (*types.CoverageReport, error) {
	content, err := coverageInputContent(profileData)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)

	var files []types.CoverageFile
	var uncoveredFiles []string
	var current *types.CoverageFile
	totalLines, coveredLines := 0, 0

	flush := func() {
		if current == nil {
			return
		}
		fileTotal, fileCovered := coverageFileLineStats(current.Blocks)
		current.Percent = percent(fileCovered, fileTotal)
		if current.Percent == 0 {
			uncoveredFiles = append(uncoveredFiles, current.Path)
		}
		files = append(files, *current)
		current = nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "TN:") {
			continue
		}
		if strings.HasPrefix(line, "SF:") {
			flush()
			current = &types.CoverageFile{Path: strings.TrimPrefix(line, "SF:")}
			continue
		}
		if line == "end_of_record" {
			flush()
			continue
		}
		if current == nil {
			continue
		}
		if strings.HasPrefix(line, "DA:") {
			block, ok := parseLCOVLine(line)
			if !ok {
				continue
			}
			current.Blocks = append(current.Blocks, block)
			totalLines++
			if block.Covered {
				coveredLines++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("扫描 Rust LCOV 覆盖率数据失败: %w", err)
	}
	flush()

	if len(files) == 0 {
		return nil, fmt.Errorf("未解析到任何 Rust LCOV 覆盖率数据")
	}

	report := &types.CoverageReport{
		Framework:    "cargo-test",
		TotalPercent: percent(coveredLines, totalLines),
		Files:        files,
		Summary: types.CoverageSummary{
			TotalStatements:   totalLines,
			CoveredStatements: coveredLines,
			TotalFiles:        len(files),
			CoveredFiles:      len(files) - len(uncoveredFiles),
			UncoveredFiles:    uncoveredFiles,
		},
	}
	report.Suggestions = GenerateSuggestions(report)
	report.TestTasks = GenerateTestTasks(report)
	return report, nil
}

func parseLCOVLine(line string) (types.CoverageBlock, bool) {
	value := strings.TrimPrefix(line, "DA:")
	parts := strings.Split(value, ",")
	if len(parts) < 2 {
		return types.CoverageBlock{}, false
	}
	lineNumber, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return types.CoverageBlock{}, false
	}
	count, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return types.CoverageBlock{}, false
	}
	return types.CoverageBlock{
		StartLine: lineNumber,
		EndLine:   lineNumber,
		Count:     count,
		Covered:   count > 0,
	}, true
}

func coverageFileLineStats(blocks []types.CoverageBlock) (total, covered int) {
	for _, block := range blocks {
		total++
		if block.Covered {
			covered++
		}
	}
	return total, covered
}
