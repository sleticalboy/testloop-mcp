package coverage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
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
	suggestion.GapType, suggestion.MissingBranches, suggestion.SuggestedInputs = analyzeSourceCoverageGap(fn, block)
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
	for _, suggestion := range report.Suggestions {
		target := suggestion.Function
		if target == "" {
			target = filepath.Base(suggestion.File)
		}
		testFile := coverageTaskTestFile(report.Framework, suggestion.File)
		priority, priorityReason := coverageTaskPriority(suggestion, testFile)
		task := types.CoverageTestTask{
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
			TestFile:        testFile,
			TestName:        coverageTaskTestName(report.Framework, target),
			AssertionFocus:  coverageTaskAssertionFocus(suggestion),
			Priority:        priority,
			PriorityReason:  priorityReason,
			Confidence:      suggestion.Confidence,
		}
		tasks = append(tasks, task)
	}
	sortCoverageTasks(tasks)
	for i := range tasks {
		tasks[i].ID = fmt.Sprintf("%s-%d", sanitizeTaskID(report.Framework), i+1)
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
		return "npx jest " + filepath.ToSlash(file)
	case "vitest":
		return "npx vitest run " + filepath.ToSlash(file)
	case "mocha":
		return "npx mocha " + filepath.ToSlash(file)
	case "pytest":
		return "python3 -m pytest " + filepath.ToSlash(file)
	case "cargo-test":
		return "cargo test"
	case "junit":
		return "mvn test"
	default:
		return ""
	}
}

func coverageTaskTestFile(framework string, file string) string {
	ext := filepath.Ext(file)
	base := strings.TrimSuffix(file, ext)
	switch framework {
	case "go-test":
		if strings.HasSuffix(file, "_test.go") {
			return file
		}
		if ext == ".go" {
			return base + "_test.go"
		}
	case "cargo-test":
		return file
	case "junit":
		return javaCoverageTestFile(file)
	case "jest", "vitest":
		if strings.Contains(base, ".test") || strings.Contains(base, ".spec") {
			return file
		}
		return base + ".test" + ext
	case "mocha":
		if strings.Contains(base, ".test") || strings.Contains(base, ".spec") {
			return file
		}
		return base + ".spec" + ext
	case "pytest":
		dir := filepath.Dir(file)
		name := filepath.Base(file)
		if strings.HasPrefix(name, "test_") {
			return file
		}
		if dir == "." || dir == "" {
			return filepath.Join("tests", "test_"+name)
		}
		return filepath.Join(dir, "tests", "test_"+name)
	}
	return ""
}

func javaCoverageTestFile(file string) string {
	slash := filepath.ToSlash(file)
	const mainRoot = "src/main/java/"
	const testRoot = "src/test/java/"
	if strings.HasPrefix(slash, testRoot) {
		return file
	}
	if strings.HasPrefix(slash, mainRoot) {
		withoutRoot := strings.TrimPrefix(slash, mainRoot)
		ext := filepath.Ext(withoutRoot)
		base := strings.TrimSuffix(withoutRoot, ext)
		return filepath.FromSlash(testRoot + base + "Test" + ext)
	}
	if !filepath.IsAbs(file) && strings.Contains(slash, "/") {
		ext := filepath.Ext(slash)
		base := strings.TrimSuffix(slash, ext)
		return filepath.FromSlash(testRoot + base + "Test" + ext)
	}
	ext := filepath.Ext(file)
	base := strings.TrimSuffix(file, ext)
	if ext == ".java" {
		return base + "Test" + ext
	}
	return ""
}

func coverageTaskTestName(framework string, target string) string {
	words := identifierWords(target)
	if len(words) == 0 {
		return ""
	}
	switch framework {
	case "go-test":
		return "Test" + pascalCase(words)
	case "cargo-test":
		return "test_" + snakeCase(words) + "_covers_gap"
	case "junit":
		return "shouldCover" + pascalCase(words) + "Gap"
	case "jest", "vitest", "mocha":
		return "covers " + target + " coverage gap"
	case "pytest":
		return "test_" + snakeCase(words) + "_covers_gap"
	default:
		return ""
	}
}

func coverageTaskAssertionFocus(suggestion types.CoverageSuggestion) []string {
	var focus []string
	switch suggestion.GapType {
	case "branch":
		focus = append(focus, "断言未覆盖分支的返回值或副作用")
	case "error_path":
		focus = append(focus, "断言错误、异常或空值路径")
	case "return_path":
		focus = append(focus, "断言未覆盖返回路径的具体结果")
	case "statement":
		focus = append(focus, "断言未覆盖语句执行后的可观察结果")
	}
	focus = append(focus, suggestion.MissingBranches...)
	if len(suggestion.UncoveredLines) > 0 {
		focus = append(focus, fmt.Sprintf("覆盖未执行行: %s", intsCSV(suggestion.UncoveredLines)))
	}
	return focus
}

func coverageTaskPriority(suggestion types.CoverageSuggestion, testFile string) (int, string) {
	score := 0
	var reasons []string
	if suggestion.Function != "" {
		score += 40
		reasons = append(reasons, "已定位到具体函数或方法")
	} else if suggestion.LineRange == "entire file" {
		score -= 20
		reasons = append(reasons, "整文件泛化任务靠后处理")
	}
	switch suggestion.GapType {
	case "branch":
		score += 30
		reasons = append(reasons, "分支缺口通常能生成高价值断言")
	case "error_path":
		score += 28
		reasons = append(reasons, "错误或空值路径通常容易补充明确断言")
	case "return_path":
		score += 20
		reasons = append(reasons, "返回路径可直接断言结果")
	case "statement":
		score += 10
		reasons = append(reasons, "普通语句缺口有明确行号")
	}
	if len(suggestion.SuggestedInputs) > 0 {
		score += 10
		reasons = append(reasons, "已有建议输入")
	}
	if len(suggestion.UncoveredLines) > 0 {
		score += 8
		reasons = append(reasons, "已有未覆盖行列表")
	}
	if testFile != "" {
		score += 6
		reasons = append(reasons, "已有推荐测试文件")
	}
	score += int(suggestion.Confidence * 10)
	if suggestion.Confidence > 0 {
		reasons = append(reasons, fmt.Sprintf("置信度 %.2f", suggestion.Confidence))
	}
	if score < 0 {
		score = 0
	}
	return score, strings.Join(reasons, "；")
}

func sortCoverageTasks(tasks []types.CoverageTestTask) {
	sort.SliceStable(tasks, func(i int, j int) bool {
		if tasks[i].Priority != tasks[j].Priority {
			return tasks[i].Priority > tasks[j].Priority
		}
		if tasks[i].Confidence != tasks[j].Confidence {
			return tasks[i].Confidence > tasks[j].Confidence
		}
		if tasks[i].File != tasks[j].File {
			return tasks[i].File < tasks[j].File
		}
		if tasks[i].LineRange != tasks[j].LineRange {
			return tasks[i].LineRange < tasks[j].LineRange
		}
		return tasks[i].Target < tasks[j].Target
	})
}

func identifierWords(value string) []string {
	var words []string
	var current strings.Builder
	flush := func() {
		if current.Len() == 0 {
			return
		}
		words = append(words, current.String())
		current.Reset()
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			current.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return words
}

func pascalCase(words []string) string {
	var b strings.Builder
	for _, word := range words {
		if word == "" {
			continue
		}
		b.WriteString(strings.ToUpper(word[:1]))
		if len(word) > 1 {
			b.WriteString(word[1:])
		}
	}
	return b.String()
}

func snakeCase(words []string) string {
	for i, word := range words {
		words[i] = strings.ToLower(word)
	}
	return strings.Join(words, "_")
}

func intsCSV(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return strings.Join(parts, ",")
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
