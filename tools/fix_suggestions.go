package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/types"
)

type fixSuggestionsInput struct {
	Failures   string `json:"failures" jsonschema:"parse_results 返回的失败 JSON 字符串"`
	SourceCode string `json:"source_code" jsonschema:"原始源代码文件路径"`
	TestCode   string `json:"test_code,omitempty" jsonschema:"测试代码文件路径（可选）"`
}

var indexOutOfRangeRe = regexp.MustCompile(`index out of range \[?(-?\d+)\]?.*length (\d+)`)
var repairIDInvalidRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)

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
		return structuredToolResult([]types.FixSuggestion{})
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
	suggestions := generateFixSuggestions(failures, string(sourceCode), string(testCode), sourceFile, input.TestCode)

	return structuredToolResult(suggestions)
}

func generateFixSuggestions(failures []types.TestFailure, sourceCode, testCode, sourceFile, testFile string) []types.FixSuggestion {
	var suggestions []types.FixSuggestion

	for _, failure := range failures {
		if failure.File == "" {
			failure.File = sourceFile
		}
		suggestion := types.FixSuggestion{
			File:  failure.File,
			Line:  failure.Line,
			Issue: failure.Error,
		}

		errorMsg := failure.Error
		lowerError := strings.ToLower(errorMsg)
		context := failureContextLines(failure, sourceCode, testCode, sourceFile, testFile)
		suggestion.ContextFile = context.File
		suggestion.ContextLine = context.Line

		if isExpectationMismatch(failure, lowerError) {
			suggestion.Category = "expectation_mismatch"
			suggestion.SuggestedFix = analyzeExpectationMismatch(failure, context.SourceLine, context.TestLine)
			suggestion.Confidence = 0.8
		} else if strings.Contains(lowerError, "index out of range") {
			suggestion.Category = "index_out_of_range"
			suggestion.SuggestedFix = analyzeIndexOutOfRange(errorMsg, context.SourceLine)
			suggestion.Confidence = 0.9
		} else if strings.Contains(lowerError, "division by zero") || strings.Contains(lowerError, "divide by zero") {
			suggestion.Category = "divide_by_zero"
			suggestion.SuggestedFix = analyzeDivideByZero(context.SourceLine)
			suggestion.Confidence = 0.95
		} else if strings.Contains(lowerError, "nil pointer") ||
			strings.Contains(lowerError, "invalid memory address") ||
			strings.Contains(lowerError, "panic: runtime error") {
			suggestion.Category = "runtime_panic"
			suggestion.SuggestedFix = analyzeRuntimePanic(errorMsg, context.SourceLine)
			suggestion.Confidence = 0.9
		} else if strings.Contains(lowerError, "undefined:") {
			suggestion.Category = "undefined_symbol"
			suggestion.SuggestedFix = analyzeUndefined(errorMsg)
			suggestion.Confidence = 0.7
		} else if strings.Contains(lowerError, "type mismatch") || strings.Contains(lowerError, "cannot use") {
			suggestion.Category = "type_mismatch"
			suggestion.SuggestedFix = analyzeTypeMismatch(errorMsg, context.SourceLine)
			suggestion.Confidence = 0.7
		} else {
			suggestion.Category = "generic_failure"
			suggestion.SuggestedFix = analyzeGenericFailure(errorMsg, context.SourceLine, context.TestLine)
			suggestion.Confidence = 0.5
		}
		suggestion.RepairTask = buildRepairTask(failure, suggestion, context, sourceFile, testFile)

		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

func buildRepairTask(failure types.TestFailure, suggestion types.FixSuggestion, context failureContext, sourceFile, testFile string) *types.RepairTask {
	targetFile := firstNonEmpty(context.File, suggestion.File, sourceFile)
	targetLine := context.Line
	if targetLine == 0 {
		targetLine = suggestion.Line
	}
	return &types.RepairTask{
		ID:                repairTaskID(suggestion.Category, failure.TestName, targetFile, targetLine),
		TestName:          failure.TestName,
		Category:          suggestion.Category,
		Issue:             suggestion.Issue,
		TargetFile:        targetFile,
		TargetLine:        targetLine,
		ContextFile:       context.File,
		ContextLine:       context.Line,
		ContextSnippet:    firstNonEmpty(context.TestLine, context.SourceLine),
		EditableFiles:     editableRepairFiles(sourceFile, testFile, suggestion.File, context.File),
		SuggestedCommands: suggestedRepairCommands(sourceFile, testFile),
		AssertionFocus:    repairAssertionFocus(suggestion.Category),
	}
}

func repairTaskID(category, testName, targetFile string, targetLine int) string {
	seed := strings.TrimSpace(testName)
	if seed == "" {
		seed = filepath.Base(targetFile)
		if targetLine > 0 {
			seed = fmt.Sprintf("%s-%d", seed, targetLine)
		}
	}
	slug := strings.Trim(repairIDInvalidRe.ReplaceAllString(strings.ToLower(seed), "-"), "-")
	if slug == "" {
		slug = "failure"
	}
	if category == "" {
		category = "generic_failure"
	}
	return "repair-" + category + "-" + slug
}

func editableRepairFiles(files ...string) []string {
	var result []string
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		if hasEquivalentPath(result, file) {
			continue
		}
		result = append(result, file)
	}
	return result
}

func hasEquivalentPath(paths []string, candidate string) bool {
	for _, path := range paths {
		if samePath(path, candidate) {
			return true
		}
	}
	return false
}

func suggestedRepairCommands(sourceFile, testFile string) []string {
	target := firstNonEmpty(testFile, sourceFile)
	ext := strings.ToLower(filepath.Ext(target))
	switch ext {
	case ".go":
		return []string{"go test ./..."}
	case ".py":
		if testFile != "" {
			return []string{"pytest " + filepath.ToSlash(testFile)}
		}
		return []string{"pytest"}
	case ".js", ".jsx", ".ts", ".tsx":
		if testFile != "" {
			return []string{"npm test -- " + filepath.ToSlash(testFile)}
		}
		return []string{"npm test"}
	case ".rs":
		return []string{"cargo test"}
	case ".java":
		return []string{"mvn test"}
	default:
		return nil
	}
}

func repairAssertionFocus(category string) string {
	switch category {
	case "expectation_mismatch":
		return "对比实际值和期望值，判断应修正测试断言还是实现返回路径。"
	case "index_out_of_range":
		return "补充边界检查，并用空集合、单元素集合和最大索引验证。"
	case "divide_by_zero":
		return "明确除数为 0 时的业务语义，并断言错误返回或 fallback 行为。"
	case "runtime_panic":
		return "定位 panic 行的 nil、slice、map、接口或类型断言访问，并补充异常输入测试。"
	case "undefined_symbol":
		return "确认符号拼写、作用域和 import/package 前缀。"
	case "type_mismatch":
		return "对齐函数签名、变量声明和调用参数类型。"
	default:
		return "先复现失败，再判断问题属于测试期望、实现逻辑还是环境依赖。"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func isExpectationMismatch(failure types.TestFailure, lowerError string) bool {
	if strings.TrimSpace(failure.Expected) != "" && strings.TrimSpace(failure.Received) != "" {
		return true
	}
	return (strings.Contains(lowerError, "got") && strings.Contains(lowerError, "want")) ||
		(strings.Contains(lowerError, "expected") && (strings.Contains(lowerError, "received") ||
			strings.Contains(lowerError, " to be ") ||
			strings.Contains(lowerError, " to equal "))) ||
		strings.Contains(lowerError, "expect(received)")
}

func analyzeExpectationMismatch(failure types.TestFailure, sourceLine, testLine string) string {
	got, want := failure.Received, failure.Expected

	errorMsg := failure.Error
	lowerError := strings.ToLower(errorMsg)
	if got == "" && want == "" {
		got, want = extractExpectedActualFromAssertion(errorMsg)
	}
	if got == "" {
		got = valueAfterToken(errorMsg, lowerError, "got", ",")
	}
	if want == "" {
		want = valueAfterToken(errorMsg, lowerError, "want", "")
	}

	var sb strings.Builder
	sb.WriteString("期望值不匹配\n")
	sb.WriteString(fmt.Sprintf("  实际值: %s\n", valueOrUnknown(got)))
	sb.WriteString(fmt.Sprintf("  期望值: %s\n\n", valueOrUnknown(want)))
	sb.WriteString("可能的原因和修复建议：\n")
	sb.WriteString("1. 如果期望值错误，修正测试断言中的 want/expected 值。\n")
	sb.WriteString("2. 如果实际值错误，优先检查目标函数的返回路径和边界条件。\n")
	sb.WriteString("3. 对比失败输入下的实现分支，确认是否漏处理空值、零值、负值或错误返回。\n")
	appendContextLines(&sb, sourceLine, testLine)

	return sb.String()
}

func valueAfterToken(errorMsg, lowerError, token, endToken string) string {
	if idx := strings.Index(lowerError, token); idx >= 0 {
		rest := errorMsg[idx+len(token):]
		if endToken != "" {
			if endIdx := strings.Index(rest, endToken); endIdx > 0 {
				return strings.TrimRight(strings.TrimSpace(rest[:endIdx]), ".!;")
			}
		}
		return strings.TrimRight(strings.TrimSpace(rest), ".!;")
	}
	return ""
}

func extractExpectedActualFromAssertion(errorMsg string) (string, string) {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)expected\s+(.+?)\s+to\s+(?:be|equal)\s+(.+?)(?:\s*//.*)?$`),
		regexp.MustCompile(`(?i)expected\s+(.+?)\s+to\s+(?:deep\s+)?equal\s+(.+)$`),
	}
	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(strings.TrimSpace(errorMsg))
		if len(matches) == 3 {
			return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2])
		}
	}
	return "", ""
}

func analyzeIndexOutOfRange(errorMsg, sourceLine string) string {
	idx, length := extractIndexAndLength(errorMsg)
	var sb strings.Builder
	sb.WriteString("数组或切片越界\n")
	if idx != "" || length != "" {
		sb.WriteString(fmt.Sprintf("  失败索引: %s\n", valueOrUnknown(idx)))
		sb.WriteString(fmt.Sprintf("  当前长度: %s\n", valueOrUnknown(length)))
	}
	sb.WriteString("建议：\n")
	sb.WriteString("1. 在访问 slice/array 前添加边界检查：idx >= 0 && idx < len(values)。\n")
	sb.WriteString("2. 如果索引来自输入参数，补充边界输入测试，例如空切片、单元素切片和最大索引。\n")
	sb.WriteString("3. 如果索引由循环计算得到，检查循环终止条件是否使用了 <= 或错误的长度来源。\n")
	appendContextLines(&sb, sourceLine, "")
	return sb.String()
}

func analyzeDivideByZero(sourceLine string) string {
	var sb strings.Builder
	sb.WriteString("除零错误\n")
	sb.WriteString("建议：\n")
	sb.WriteString("1. 在执行除法或取模前添加除零检查，确认除数是否为 0。\n")
	sb.WriteString("2. 根据业务语义返回错误、默认值或跳过该计算，不要静默吞掉异常输入。\n")
	sb.WriteString("3. 补充除数为 0 的测试，断言错误返回或约定的 fallback 行为。\n")
	appendContextLines(&sb, sourceLine, "")
	return sb.String()
}

func analyzeRuntimePanic(errorMsg, sourceLine string) string {
	var sb strings.Builder
	sb.WriteString("运行时 panic\n")
	sb.WriteString(fmt.Sprintf("  错误: %s\n", errorMsg))
	sb.WriteString("建议：\n")
	sb.WriteString("1. 先定位 panic 行访问的指针、map、slice、接口或类型断言。\n")
	sb.WriteString("2. 对可能为 nil 的值增加显式检查，并返回可断言的错误。\n")
	sb.WriteString("3. 补充 nil/空集合/缺失字段测试，避免只覆盖正常路径。\n")
	appendContextLines(&sb, sourceLine, "")
	return sb.String()
}

func analyzeUndefined(errorMsg string) string {
	symbol := ""
	if idx := strings.Index(strings.ToLower(errorMsg), "undefined:"); idx >= 0 {
		symbol = strings.TrimSpace(errorMsg[idx+len("undefined:"):])
	}
	var sb strings.Builder
	sb.WriteString("未定义引用\n")
	if symbol != "" {
		sb.WriteString(fmt.Sprintf("  符号: %s\n", symbol))
	}
	sb.WriteString("建议：\n")
	sb.WriteString("1. 检查变量、函数或类型名是否拼写正确。\n")
	sb.WriteString("2. 确认符号是否在当前 package 中导出或可见。\n")
	sb.WriteString("3. 如果来自其他包，补充正确 import 或使用包名前缀。\n")
	return sb.String()
}

func analyzeTypeMismatch(errorMsg, sourceLine string) string {
	var sb strings.Builder
	sb.WriteString("类型不匹配\n")
	sb.WriteString(fmt.Sprintf("  错误: %s\n", errorMsg))
	sb.WriteString("建议：\n")
	sb.WriteString("1. 对比函数签名、变量声明和调用参数的实际类型。\n")
	sb.WriteString("2. 优先修正 API 使用方式；只有语义明确时才添加类型转换。\n")
	sb.WriteString("3. 如果是泛型、接口或指针/值接收者问题，确认约束和 nil 处理是否正确。\n")
	appendContextLines(&sb, sourceLine, "")
	return sb.String()
}

func analyzeGenericFailure(errorMsg, sourceLine, testLine string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("测试失败: %s\n", errorMsg))
	sb.WriteString("建议：\n")
	sb.WriteString("1. 先定位失败测试的输入、断言和目标函数。\n")
	sb.WriteString("2. 判断问题属于测试期望错误、实现逻辑错误，还是外部依赖/环境问题。\n")
	sb.WriteString("3. 修复后重新运行同一测试，再运行相关包测试防止回归。\n")
	appendContextLines(&sb, sourceLine, testLine)
	return sb.String()
}

func appendContextLines(sb *strings.Builder, sourceLine, testLine string) {
	if strings.TrimSpace(sourceLine) != "" {
		sb.WriteString("\n源码附近行：\n")
		sb.WriteString(strings.TrimSpace(sourceLine))
		sb.WriteByte('\n')
	}
	if strings.TrimSpace(testLine) != "" {
		sb.WriteString("\n测试附近行：\n")
		sb.WriteString(strings.TrimSpace(testLine))
		sb.WriteByte('\n')
	}
}

type failureContext struct {
	SourceLine string
	TestLine   string
	File       string
	Line       int
}

func failureContextLines(failure types.TestFailure, sourceCode, testCode, sourceFile, testFile string) failureContext {
	if failure.Line <= 0 {
		return failureContext{}
	}
	switch {
	case samePath(failure.File, sourceFile):
		return newFailureContext(lineAt(sourceCode, failure.Line), "", sourceFile, failure.Line)
	case testFile != "" && samePath(failure.File, testFile):
		return newFailureContext("", lineAt(testCode, failure.Line), testFile, failure.Line)
	case failure.File == "":
		return newFailureContext(lineAt(sourceCode, failure.Line), "", sourceFile, failure.Line)
	case looksLikeTestFile(failure.File):
		return newFailureContext("", lineAt(testCode, failure.Line), testFile, failure.Line)
	default:
		return failureContext{}
	}
}

func newFailureContext(sourceLine, testLine, file string, line int) failureContext {
	ctx := failureContext{SourceLine: sourceLine, TestLine: testLine}
	if strings.TrimSpace(sourceLine) != "" || strings.TrimSpace(testLine) != "" {
		ctx.File = file
		ctx.Line = line
	}
	return ctx
}

func samePath(a, b string) bool {
	if strings.TrimSpace(a) == "" || strings.TrimSpace(b) == "" {
		return false
	}
	cleanA := filepath.Clean(a)
	cleanB := filepath.Clean(b)
	if cleanA == cleanB {
		return true
	}
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA == nil && errB == nil && filepath.Clean(absA) == filepath.Clean(absB) {
		return true
	}
	return pathHasSuffix(cleanA, cleanB) || pathHasSuffix(cleanB, cleanA)
}

func pathHasSuffix(path, suffix string) bool {
	path = filepath.ToSlash(filepath.Clean(path))
	suffix = filepath.ToSlash(filepath.Clean(suffix))
	return path != "." && suffix != "." && path != suffix && strings.HasSuffix(path, "/"+suffix)
}

func looksLikeTestFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(base, "_test.") ||
		strings.HasPrefix(base, "test_") ||
		strings.HasSuffix(base, ".test.js") ||
		strings.HasSuffix(base, ".spec.js") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".spec.ts")
}

func lineAt(text string, line int) string {
	if line <= 0 || text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	if line > len(lines) {
		return ""
	}
	return lines[line-1]
}

func extractIndexAndLength(errorMsg string) (string, string) {
	matches := indexOutOfRangeRe.FindStringSubmatch(errorMsg)
	if len(matches) != 3 {
		return "", ""
	}
	return matches[1], matches[2]
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}
