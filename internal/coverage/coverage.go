package coverage

import (
	"fmt"

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
	default:
		return nil, fmt.Errorf("不支持的覆盖率框架: %s", framework)
	}
}

// GenerateSuggestions 根据覆盖率报告生成改进建议
func GenerateSuggestions(report *types.CoverageReport) []types.CoverageSuggestion {
	var suggestions []types.CoverageSuggestion
	var goFunctions map[string][]goFuncRange
	if report.Framework == "go-test" {
		goFunctions = mapGoFunctionsByFile(report.Files)
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

func enrichGoCoverageSuggestion(suggestion *types.CoverageSuggestion, block types.CoverageBlock, ranges []goFuncRange) {
	fn := findGoFunctionForBlock(ranges, block)
	if fn == nil {
		return
	}
	suggestion.Function = fn.Name
	suggestion.Kind = fn.Kind
	suggestion.UncoveredLines = lineRange(block.StartLine, block.EndLine)
	suggestion.SuggestedInputs = suggestedGoInputs(fn.Params)
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
