package coverage

import (
	"bufio"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
)

// ParseNodeTestCoverage parses the TAP coverage report printed by
// `node --test --experimental-test-coverage`.
func ParseNodeTestCoverage(profileData string) (*types.CoverageReport, error) {
	content, err := coverageInputContent(profileData)
	if err != nil {
		return nil, err
	}

	var files []types.CoverageFile
	var uncoveredFiles []string
	coveredFiles := 0
	totalPercent := 0.0
	sawAllFiles := false

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		row, ok := parseNodeCoverageRow(scanner.Text())
		if !ok {
			continue
		}
		if row.file == "all files" {
			totalPercent = row.linePercent
			sawAllFiles = true
			continue
		}
		cf := types.CoverageFile{
			Path:    row.file,
			Percent: row.linePercent,
			Blocks:  nodeCoverageBlocks(row.uncoveredLines),
		}
		files = append(files, cf)
		if cf.Percent > 0 {
			coveredFiles++
		} else {
			uncoveredFiles = append(uncoveredFiles, cf.Path)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("扫描 Node coverage 数据失败: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("未解析到任何 Node coverage 文件数据")
	}
	if !sawAllFiles {
		totalPercent = averageNodeCoveragePercent(files)
	}
	sort.SliceStable(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	report := &types.CoverageReport{
		Framework:    "node-test",
		TotalPercent: totalPercent,
		Files:        files,
		Summary: types.CoverageSummary{
			TotalFiles:     len(files),
			CoveredFiles:   coveredFiles,
			UncoveredFiles: uncoveredFiles,
		},
	}
	report.Suggestions = GenerateSuggestions(report)
	report.TestTasks = GenerateTestTasks(report)
	return report, nil
}

type nodeCoverageRow struct {
	file           string
	linePercent    float64
	uncoveredLines []int
}

func parseNodeCoverageRow(line string) (nodeCoverageRow, bool) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "#"))
	if !strings.Contains(trimmed, "|") {
		return nodeCoverageRow{}, false
	}
	parts := strings.Split(trimmed, "|")
	if len(parts) < 4 {
		return nodeCoverageRow{}, false
	}
	file := strings.TrimSpace(parts[0])
	if file == "" || file == "file" || strings.HasPrefix(file, "-") {
		return nodeCoverageRow{}, false
	}
	linePercent, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return nodeCoverageRow{}, false
	}
	uncovered := []int{}
	if len(parts) >= 5 {
		uncovered = parseNodeUncoveredLines(parts[4])
	}
	return nodeCoverageRow{file: file, linePercent: linePercent, uncoveredLines: uncovered}, true
}

func parseNodeUncoveredLines(value string) []int {
	seen := map[int]bool{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			start, err1 := strconv.Atoi(strings.TrimSpace(bounds[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err1 != nil || err2 != nil {
				continue
			}
			if end < start {
				end = start
			}
			for line := start; line <= end; line++ {
				seen[line] = true
			}
			continue
		}
		line, err := strconv.Atoi(part)
		if err == nil {
			seen[line] = true
		}
	}
	lines := make([]int, 0, len(seen))
	for line := range seen {
		lines = append(lines, line)
	}
	sort.Ints(lines)
	return lines
}

func nodeCoverageBlocks(lines []int) []types.CoverageBlock {
	if len(lines) == 0 {
		return nil
	}
	blocks := []types.CoverageBlock{}
	start := lines[0]
	prev := lines[0]
	for _, line := range lines[1:] {
		if line == prev+1 {
			prev = line
			continue
		}
		blocks = append(blocks, types.CoverageBlock{StartLine: start, EndLine: prev, Count: 0, Covered: false})
		start = line
		prev = line
	}
	blocks = append(blocks, types.CoverageBlock{StartLine: start, EndLine: prev, Count: 0, Covered: false})
	return blocks
}

func averageNodeCoveragePercent(files []types.CoverageFile) float64 {
	if len(files) == 0 {
		return 0
	}
	total := 0.0
	for _, file := range files {
		total += file.Percent
	}
	return total / float64(len(files))
}
