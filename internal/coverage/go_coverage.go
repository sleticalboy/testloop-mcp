package coverage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
)

// Go coverage profile 行格式:
// mode: set|count|atomic
// path/to/file.go:startLine.startCol,endLine.endCol numStatements count

var goCoverageLineRe = regexp.MustCompile(
	`^(.+):(\d+)\.(\d+),(\d+)\.(\d+)\s+(\d+)\s+(\d+)$`,
)

// ParseGoCoverage 解析 go test -coverprofile 输出的覆盖率 profile 文件内容或文件路径
func ParseGoCoverage(profileData string) (*types.CoverageReport, error) {
	content, err := coverageInputContent(profileData)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 256*1024), 256*1024) // 支持长行

	fileMap := make(map[string]*types.CoverageFile)
	var totalStmts, coveredStmts int
	modulePath := goCoverageModulePath()

	modeLine := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// 第一行是 mode: xxx
		if !modeLine {
			if strings.HasPrefix(line, "mode:") {
				modeLine = true
				continue
			}
		}

		matches := goCoverageLineRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		filePath := matches[1]
		filePath = normalizeGoCoverageFilePath(filePath, modulePath)
		startLine, _ := strconv.Atoi(matches[2])
		endLine, _ := strconv.Atoi(matches[4])
		numStmts, _ := strconv.Atoi(matches[6])
		count, _ := strconv.Atoi(matches[7])

		cf, ok := fileMap[filePath]
		if !ok {
			cf = &types.CoverageFile{Path: filePath}
			fileMap[filePath] = cf
		}

		covered := count > 0
		cf.Blocks = append(cf.Blocks, types.CoverageBlock{
			StartLine: startLine,
			EndLine:   endLine,
			Count:     count,
			Covered:   covered,
		})

		totalStmts += numStmts
		if covered {
			coveredStmts += numStmts
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("扫描覆盖率数据失败: %w", err)
	}

	if len(fileMap) == 0 {
		return nil, fmt.Errorf("未解析到任何覆盖率数据")
	}

	// 构建文件列表并计算各文件覆盖率
	var files []types.CoverageFile
	var uncoveredFiles []string
	coveredFiles := 0

	for _, cf := range fileMap {
		cfStmts, cfCovered := computeFileStats(cf)
		cf.Percent = percent(cfCovered, cfStmts)
		if cf.Percent > 0 {
			coveredFiles++
		} else {
			uncoveredFiles = append(uncoveredFiles, cf.Path)
		}
		files = append(files, *cf)
	}

	report := &types.CoverageReport{
		Framework:    "go-test",
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

func normalizeGoCoverageFilePath(path string, modulePath string) string {
	if path == "" {
		return path
	}
	if fileExists(path) {
		return filepath.Clean(path)
	}
	clean := filepath.Clean(path)
	if fileExists(clean) {
		return clean
	}
	slash := filepath.ToSlash(clean)
	modulePath = strings.Trim(filepath.ToSlash(modulePath), "/")
	if modulePath != "" && strings.HasPrefix(slash, modulePath+"/") {
		withoutModule := strings.TrimPrefix(slash, modulePath+"/")
		candidate := filepath.FromSlash(withoutModule)
		if fileExists(candidate) {
			return filepath.Clean(candidate)
		}
	}
	if sourcePath := resolveGoSourcePath(clean); sourcePath != "" {
		return filepath.Clean(sourcePath)
	}
	return clean
}

func goCoverageModulePath() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		goMod := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(goMod)
		if err == nil {
			return parseGoModulePath(string(data))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func parseGoModulePath(goMod string) string {
	for _, line := range strings.Split(goMod, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// computeFileStats 按 block 的行数近似计算文件级语句覆盖
func computeFileStats(cf *types.CoverageFile) (total, covered int) {
	for _, b := range cf.Blocks {
		// 用行跨度近似语句数
		stmts := b.EndLine - b.StartLine + 1
		if stmts < 1 {
			stmts = 1
		}
		total += stmts
		if b.Covered {
			covered += stmts
		}
	}
	return
}

func percent(covered, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(covered) / float64(total) * 100
}
