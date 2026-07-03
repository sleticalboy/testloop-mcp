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
	case "pytest":
		return ParsePytestCoverage(profileData)
	default:
		return nil, fmt.Errorf("不支持的覆盖率框架: %s", framework)
	}
}

// GenerateSuggestions 根据覆盖率报告生成改进建议
func GenerateSuggestions(report *types.CoverageReport) []types.CoverageSuggestion {
	var suggestions []types.CoverageSuggestion

	for _, file := range report.Files {
		if file.Percent >= 100 {
			continue
		}

		// 找出未覆盖的块
		for _, block := range file.Blocks {
			if !block.Covered {
				suggestions = append(suggestions, types.CoverageSuggestion{
					File:       file.Path,
					LineRange:  fmt.Sprintf("%d-%d", block.StartLine, block.EndLine),
					Reason:     "此代码块未被测试覆盖",
					Confidence: 0.9,
				})
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
