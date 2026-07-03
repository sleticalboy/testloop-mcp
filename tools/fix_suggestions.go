package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/binlee/testloop-mcp/types"
)

type fixSuggestionsInput struct {
	Failures   string `json:"failures" jsonschema:"parse_results 返回的失败 JSON 字符串"`
	SourceCode string `json:"source_code" jsonschema:"原始源代码文件路径"`
	TestCode   string `json:"test_code,omitempty" jsonschema:"测试代码文件路径（可选）"`
}

func HandleFixSuggestions(ctx context.Context, req *mcp.CallToolRequest, input fixSuggestionsInput) (*mcp.CallToolResult, any, error) {
	failuresStr := input.Failures
	sourceFile := input.SourceCode
	
	if failuresStr == "" || sourceFile == "" {
		return nil, nil, fmt.Errorf("failures 和 source_code 参数必填")
	}

	var failures []types.TestFailure
	if err := json.Unmarshal([]byte(failuresStr), &failures); err != nil {
		return nil, nil, fmt.Errorf("failures 参数解析失败: %w", err)
	}

	if len(failures) == 0 {
		resultJSON, _ := json.Marshal([]types.FixSuggestion{})
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
		}, nil, nil
	}

	// 读取源代码
	sourceCode, err := os.ReadFile(sourceFile)
	if err != nil {
		return nil, nil, fmt.Errorf("读取源文件失败: %w", err)
	}

	// 生成修复建议
	suggestions := generateFixSuggestions(failures, string(sourceCode), sourceFile)

	resultJSON, _ := json.Marshal(suggestions)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, nil, nil
}

func generateFixSuggestions(failures []types.TestFailure, sourceCode, sourceFile string) []types.FixSuggestion {
	var suggestions []types.FixSuggestion

	for _, failure := range failures {
		suggestion := types.FixSuggestion{
			File:  failure.File,
			Line:  failure.Line,
			Issue: failure.Error,
		}

		// 分析错误信息，生成修复建议
		errorMsg := failure.Error
		
		// 情况1: got X, want Y
		if strings.Contains(errorMsg, "got") && strings.Contains(errorMsg, "want") {
			suggestion.SuggestedFix = analyzeGotWant(errorMsg, sourceCode, failure.Line)
			suggestion.Confidence = 0.8
		} else if strings.Contains(errorMsg, "nil pointer") || strings.Contains(errorMsg, "panic") {
			// 情况2: 空指针错误
			suggestion.SuggestedFix = "检查是否为 nil 再访问，添加 nil 检查"
			suggestion.Confidence = 0.9
		} else if strings.Contains(errorMsg, "index out of range") {
			// 情况3: 数组越界
			suggestion.SuggestedFix = "检查数组索引是否在有效范围内，添加边界检查"
			suggestion.Confidence = 0.9
		} else {
			// 其他情况
			suggestion.SuggestedFix = fmt.Sprintf("测试失败: %s\n请检查相关代码逻辑", errorMsg)
			suggestion.Confidence = 0.5
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

func analyzeGotWant(errorMsg, sourceCode string, errorLine int) string {
	// 提取 got 和 want 的值
	// 格式: "got X, want Y" 或 "ret0 got X, want Y"
	
	// 简单分析：如果 got 和 want 不同，可能是逻辑错误或者测试用例值不对
	return fmt.Sprintf(
		"测试期望值不匹配。\n错误信息: %s\n建议：\n1. 检查测试用例的输入值和期望值是否正确\n2. 检查函数逻辑是否有错误\n3. 如果测试用例值正确，则函数实现可能有 bug",
	)
}
