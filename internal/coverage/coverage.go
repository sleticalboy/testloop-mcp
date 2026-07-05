package coverage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/binlee/testloop-mcp/types"
)

// ParseCoverage 解析覆盖率数据
func ParseCoverage(profileData, framework string) (*types.CoverageReport, error) {
	switch framework {
	case "go-test":
		return ParseGoCoverage(profileData)
	case "jest":
		return ParseJestCoverage(profileData, "jest")
	case "vitest":
		return ParseJestCoverage(profileData, "vitest")
	case "mocha":
		return ParseJestCoverage(profileData, "mocha")
	case "pytest":
		return ParsePytestCoverage(profileData)
	case "cargo-test":
		return ParseRustTarpaulinCoverage(profileData)
	case "junit":
		return ParseJaCoCoCoverage(profileData)
	default:
		return nil, fmt.Errorf("不支持的覆盖率框架: %s", framework)
	}
}

func coverageInputContent(profileData string) (string, error) {
	if _, err := os.Stat(profileData); err == nil {
		data, err := os.ReadFile(profileData)
		if err != nil {
			return "", fmt.Errorf("读取覆盖率文件失败: %w", err)
		}
		return string(data), nil
	}
	return profileData, nil
}

// GenerateSuggestions 根据覆盖率报告生成改进建议
func GenerateSuggestions(report *types.CoverageReport) []types.CoverageSuggestion {
	var suggestions []types.CoverageSuggestion
	var goFunctions map[string][]goFuncRange
	var sourceRanges map[string][]sourceRange
	if report.Framework == "go-test" {
		goFunctions = mapGoFunctionsByFile(report.Files)
	}
	if report.Framework == "cargo-test" || report.Framework == "junit" {
		sourceRanges = mapSourceRangesByFile(report.Files, report.Framework)
	}

	for _, file := range report.Files {
		if file.Percent >= 100 {
			continue
		}

		// 找出未覆盖的块
		for _, block := range file.Blocks {
			if !block.Covered {
				suggestion := types.CoverageSuggestion{
					File:       file.Path,
					LineRange:  fmt.Sprintf("%d-%d", block.StartLine, block.EndLine),
					Reason:     "此代码块未被测试覆盖",
					Confidence: 0.9,
				}
				enrichGoCoverageSuggestion(&suggestion, block, goFunctions[file.Path])
				enrichSourceCoverageSuggestion(&suggestion, block, sourceRanges[file.Path])
				suggestions = append(suggestions, suggestion)
			}
		}

		// 如果文件覆盖率低于 50%
		if file.Percent < 50 {
			suggestions = append(suggestions, types.CoverageSuggestion{
				File:       file.Path,
				LineRange:  "entire file",
				Reason:     fmt.Sprintf("文件覆盖率仅 %.1f%%，建议优先补充测试", file.Percent),
				Confidence: 0.8,
			})
		}
	}

	return suggestions
}

func enrichSourceCoverageSuggestion(suggestion *types.CoverageSuggestion, block types.CoverageBlock, ranges []sourceRange) {
	fn := findSourceRangeForBlock(ranges, block)
	if fn == nil {
		return
	}
	suggestion.Function = fn.Name
	suggestion.Kind = fn.Kind
	suggestion.UncoveredLines = lineRange(block.StartLine, block.EndLine)
	suggestion.SuggestedInputs = suggestedGoInputs(fn.Params)
	suggestion.GapType = "statement"
	suggestion.MissingBranches = []string{"未覆盖函数或方法内的语句"}
	suggestion.Reason = fmt.Sprintf("%s 中的代码行未被测试覆盖", fn.Name)
	suggestion.Confidence = 0.9
}

func enrichGoCoverageSuggestion(suggestion *types.CoverageSuggestion, block types.CoverageBlock, ranges []goFuncRange) {
	fn := findGoFunctionForBlock(ranges, block)
	if fn == nil {
		return
	}
	suggestion.Function = fn.Name
	suggestion.Kind = fn.Kind
	suggestion.UncoveredLines = lineRange(block.StartLine, block.EndLine)
	suggestion.SuggestedInputs = suggestedGoInputs(fn.Params)
	suggestion.GapType, suggestion.MissingBranches, suggestion.SuggestedInputs = analyzeGoCoverageGap(fn, block)
	suggestion.Reason = fmt.Sprintf("%s 中的代码块未被测试覆盖", fn.Name)
	suggestion.Confidence = 0.95
}

func lineRange(start int, end int) []int {
	if end < start {
		end = start
	}
	lines := make([]int, 0, end-start+1)
	for line := start; line <= end; line++ {
		lines = append(lines, line)
	}
	return lines
}

func suggestedGoInputs(params []string) []string {
	if len(params) == 0 {
		return nil
	}
	inputs := make([]string, 0, len(params))
	for _, param := range params {
		if param == "" || param == "arg" {
			inputs = append(inputs, "构造覆盖未执行分支的参数")
			continue
		}
		inputs = append(inputs, fmt.Sprintf("设置 %s 覆盖未执行分支", param))
	}
	return inputs
}

func GenerateTestTasks(report *types.CoverageReport) []types.CoverageTestTask {
	tasks := make([]types.CoverageTestTask, 0, len(report.Suggestions))
	for i, suggestion := range report.Suggestions {
		target := suggestion.Function
		if target == "" {
			target = filepath.Base(suggestion.File)
		}
		task := types.CoverageTestTask{
			ID:              fmt.Sprintf("%s-%d", sanitizeTaskID(report.Framework), i+1),
			Framework:       report.Framework,
			File:            suggestion.File,
			Target:          target,
			Kind:            suggestion.Kind,
			LineRange:       suggestion.LineRange,
			GapType:         suggestion.GapType,
			MissingBranches: suggestion.MissingBranches,
			UncoveredLines:  suggestion.UncoveredLines,
			SuggestedInputs: suggestion.SuggestedInputs,
			Goal:            coverageTaskGoal(target, suggestion.LineRange),
			Command:         coverageTaskCommand(report.Framework, suggestion.File),
			Confidence:      suggestion.Confidence,
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func coverageTaskGoal(target string, lineRange string) string {
	return fmt.Sprintf("为 %s 补充测试，覆盖未执行行段 %s", target, lineRange)
}

func coverageTaskCommand(framework string, file string) string {
	switch framework {
	case "go-test":
		dir := filepath.Dir(file)
		if dir == "." || dir == "" {
			return "go test ./..."
		}
		return "go test ./" + filepath.ToSlash(dir)
	case "jest":
		return "npx jest " + file
	case "vitest":
		return "npx vitest run " + file
	case "mocha":
		return "npx mocha"
	case "pytest":
		return "pytest " + file
	case "cargo-test":
		return "cargo test"
	case "junit":
		return "mvn test"
	default:
		return ""
	}
}

func sanitizeTaskID(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
