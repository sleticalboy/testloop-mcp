package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/types"
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

	// 读取源代码和测试代码
	sourceCode, err := os.ReadFile(sourceFile)
	if err != nil {
		return nil, nil, fmt.Errorf("读取源文件失败: %w", err)
	}

	var testCode []byte
	if input.TestCode != "" {
		testCode, _ = os.ReadFile(input.TestCode)
	}

	// 生成修复建议
	suggestions := generateFixSuggestions(failures, string(sourceCode), string(testCode), sourceFile)

	resultJSON, _ := json.Marshal(suggestions)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
	}, nil, nil
}

func generateFixSuggestions(failures []types.TestFailure, sourceCode, testCode, sourceFile string) []types.FixSuggestion {
	var suggestions []types.FixSuggestion

	for _, failure := range failures {
		suggestion := types.FixSuggestion{
			File:  failure.File,
			Line:  failure.Line,
			Issue: failure.Error,
		}

		// 分析错误信息，生成修复建议
		errorMsg := failure.Error

		// 情况1: got X, want Y (期望值不匹配)
		if strings.Contains(errorMsg, "got") && strings.Contains(errorMsg, "want") {
			suggestion.SuggestedFix = analyzeGotWant(errorMsg, sourceCode, testCode, failure.Line)
			suggestion.Confidence = 0.8
		} else if strings.Contains(errorMsg, "nil pointer") || strings.Contains(errorMsg, "panic: runtime error") {
			// 情况2: 空指针错误
			suggestion.SuggestedFix = "检查是否为 nil 再访问，添加 nil 检查：\nif ptr != nil {\n    // 访问 ptr\n}"
			suggestion.Confidence = 0.9
		} else if strings.Contains(errorMsg, "index out of range") {
			// 情况3: 数组越界
			suggestion.SuggestedFix = "检查数组索引是否在有效范围内，添加边界检查：\nif idx >= 0 && idx < len(arr) {\n    // 访问 arr[idx]\n}"
			suggestion.Confidence = 0.9
		} else if strings.Contains(errorMsg, "division by zero") {
			// 情况4: 除零错误
			suggestion.SuggestedFix = "添加除零检查：\nif b != 0 {\n    result = a / b\n}"
			suggestion.Confidence = 0.95
		} else if strings.Contains(errorMsg, "undefined:") {
			// 情况5: 未定义的变量或函数
			suggestion.SuggestedFix = "检查变量或函数名是否拼写正确，或者是否忘记了 import"
			suggestion.Confidence = 0.7
		} else if strings.Contains(errorMsg, "type mismatch") || strings.Contains(errorMsg, "cannot use") {
			// 情况6: 类型不匹配
			suggestion.SuggestedFix = "检查类型是否匹配，可能需要类型转换"
			suggestion.Confidence = 0.7
		} else {
			// 其他情况
			suggestion.SuggestedFix = fmt.Sprintf("测试失败: %s\n建议：\n1. 检查测试用例的输入值和期望值是否正确\n2. 检查函数逻辑是否有错误\n3. 查看完整错误输出以获取更多线索", errorMsg)
			suggestion.Confidence = 0.5
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

func analyzeGotWant(errorMsg, sourceCode, testCode string, errorLine int) string {
	// 提取 got 和 want 的值
	// 常见格式: "got X, want Y" 或 "ret0 got X, want Y"

	var got, want string

	// 尝试提取 got 和 want 的值
	if idx := strings.Index(errorMsg, "got"); idx > 0 {
		rest := errorMsg[idx+3:]
		if endIdx := strings.Index(rest, ","); endIdx > 0 {
			got = strings.TrimSpace(rest[:endIdx])
		}
	}

	if idx := strings.Index(errorMsg, "want"); idx > 0 {
		rest := errorMsg[idx+4:]
		want = strings.TrimSpace(rest)
		// 去掉可能的标点符号
		want = strings.TrimRight(want, ".!;")
	}

	// 生成修复建议
	var sb strings.Builder
	sb.WriteString("期望值不匹配\n")
	sb.WriteString(fmt.Sprintf("  实际值: %s\n", got))
	sb.WriteString(fmt.Sprintf("  期望值: %s\n\n", want))
	sb.WriteString("可能的原因和修复建议：\n")
	sb.WriteString("1. 测试用例的期望值填写错误 → 修改测试代码中的期望值\n")
	sb.WriteString("2. 函数实现逻辑错误 → 检查并修复函数实现\n")
	sb.WriteString("3. 边界条件处理错误 → 检查边界情况（如空值、零值、负值等）\n")

	return sb.String()
}
