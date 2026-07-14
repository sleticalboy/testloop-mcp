package coverage

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
)

type sourceRange struct {
	Name      string
	Kind      string
	StartLine int
	EndLine   int
	Params    []string
	Lines     []string
}

var (
	rustFnRe     = regexp.MustCompile(`^\s*(?:pub(?:\([^)]*\))?\s+)?(?:async\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)`)
	javaMethodRe = regexp.MustCompile(`^\s*(?:(?:public|private|protected|static|final|abstract|synchronized|native|strictfp)\s+)*(?:<[^>]+>\s*)?[\w<>\[\].?,\s]+\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*(?:throws\s+[^{]+)?\{?\s*$`)
	javaClassRe  = regexp.MustCompile(`^\s*(?:(?:public|private|protected|static|final|abstract)\s+)*(?:class|interface|enum|record)\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
)

func mapSourceRangesByFile(files []types.CoverageFile, framework string) map[string][]sourceRange {
	result := make(map[string][]sourceRange)
	for _, file := range files {
		sourcePath := resolveSourcePath(file.Path)
		if sourcePath == "" {
			continue
		}
		var ranges []sourceRange
		switch framework {
		case "jest", "vitest", "mocha":
			ranges = parseJavaScriptFunctionRangesWithTreeSitter(sourcePath)
			if len(ranges) == 0 {
				ranges = parseJavaScriptFunctionRanges(sourcePath)
			}
		case "cargo-test":
			ranges = parseRustFunctionRangesWithTreeSitter(sourcePath)
			if len(ranges) == 0 {
				ranges = parseRustFunctionRanges(sourcePath)
			}
		case "junit":
			ranges = parseJavaMethodRangesWithTreeSitter(sourcePath)
			if len(ranges) == 0 {
				ranges = parseJavaMethodRanges(sourcePath)
			}
		case "pytest":
			ranges = parsePythonFunctionRangesWithTreeSitter(sourcePath)
			if len(ranges) == 0 {
				ranges = parsePythonFunctionRanges(sourcePath)
			}
		}
		if len(ranges) > 0 {
			result[file.Path] = ranges
		}
	}
	return result
}

func resolveSourcePath(path string) string {
	if fileExists(path) {
		return path
	}
	clean := filepath.Clean(path)
	if fileExists(clean) {
		return clean
	}
	for _, root := range []string{"src/main/java", "src/test/java", "src/main/rust", "src", "crates"} {
		candidate := filepath.Join(root, clean)
		if fileExists(candidate) {
			return candidate
		}
	}
	if candidate := resolveNestedSourcePath(clean); candidate != "" {
		return candidate
	}
	parts := strings.Split(filepath.ToSlash(clean), "/")
	for i := range parts {
		candidate := filepath.FromSlash(strings.Join(parts[i:], "/"))
		if fileExists(candidate) {
			return candidate
		}
		for _, root := range []string{"src/main/java", "src/test/java", "src/main/rust", "src", "crates"} {
			rooted := filepath.Join(root, candidate)
			if fileExists(rooted) {
				return rooted
			}
		}
		if nested := resolveNestedSourcePath(candidate); nested != "" {
			return nested
		}
	}
	return ""
}

func resolveNestedSourcePath(clean string) string {
	slash := filepath.ToSlash(clean)
	for _, root := range []string{"src/main/java", "src/test/java", "src/main/rust", "src", "crates"} {
		patterns := []string{
			filepath.FromSlash("*/" + root + "/" + slash),
			filepath.FromSlash("*/*/" + root + "/" + slash),
		}
		for _, pattern := range patterns {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				continue
			}
			for _, match := range matches {
				if fileExists(match) {
					return match
				}
			}
		}
	}
	return ""
}

func parseRustFunctionRanges(path string) []sourceRange {
	lines, ok := readSourceLines(path)
	if !ok {
		return nil
	}
	var ranges []sourceRange
	for i, line := range lines {
		matches := rustFnRe.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}
		start := i + 1
		end := findBraceRangeEnd(lines, i)
		ranges = append(ranges, sourceRange{
			Name:      matches[1],
			Kind:      "function",
			StartLine: start,
			EndLine:   end,
			Params:    parseParamNames(matches[2]),
			Lines:     rangeSourceLines(lines, start, end),
		})
	}
	return ranges
}

func parseJavaMethodRanges(path string) []sourceRange {
	lines, ok := readSourceLines(path)
	if !ok {
		return nil
	}
	classStack := []string{}
	var ranges []sourceRange
	for i, line := range lines {
		if matches := javaClassRe.FindStringSubmatch(line); len(matches) == 2 {
			classStack = append(classStack, matches[1])
			continue
		}
		matches := javaMethodRe.FindStringSubmatch(line)
		if len(matches) != 3 || isJavaControlLine(line) {
			continue
		}
		name := matches[1]
		if len(classStack) > 0 {
			name = classStack[len(classStack)-1] + "." + name
		}
		start := i + 1
		end := findBraceRangeEnd(lines, i)
		ranges = append(ranges, sourceRange{
			Name:      name,
			Kind:      "method",
			StartLine: start,
			EndLine:   end,
			Params:    parseParamNames(matches[2]),
			Lines:     rangeSourceLines(lines, start, end),
		})
	}
	return ranges
}

func readSourceLines(path string) ([]string, bool) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return strings.Split(string(source), "\n"), true
}

func rangeSourceLines(lines []string, startLine int, endLine int) []string {
	if startLine < 1 {
		startLine = 1
	}
	if endLine < startLine {
		endLine = startLine
	}
	if startLine > len(lines) {
		return nil
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	return lines[startLine-1 : endLine]
}

func findBraceRangeEnd(lines []string, startIdx int) int {
	depth := 0
	seenOpen := false
	for i := startIdx; i < len(lines); i++ {
		for _, ch := range lines[i] {
			switch ch {
			case '{':
				depth++
				seenOpen = true
			case '}':
				if depth > 0 {
					depth--
				}
				if seenOpen && depth == 0 {
					return i + 1
				}
			}
		}
	}
	return len(lines)
}

func parseParamNames(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	params := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "&self" || part == "self" || part == "&mut self" {
			continue
		}
		if idx := strings.Index(part, ":"); idx > 0 {
			params = append(params, strings.TrimSpace(part[:idx]))
			continue
		}
		fields := strings.Fields(part)
		if len(fields) > 0 {
			params = append(params, strings.Trim(fields[len(fields)-1], "..."))
		}
	}
	return params
}

func isJavaControlLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	for _, prefix := range []string{"if ", "for ", "while ", "switch ", "catch ", "return "} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return strings.Contains(trimmed, " -> ")
}

func findSourceRangeForBlock(ranges []sourceRange, block types.CoverageBlock) *sourceRange {
	for i := range ranges {
		r := &ranges[i]
		if block.StartLine >= r.StartLine && block.StartLine <= r.EndLine {
			return r
		}
		if block.StartLine <= r.EndLine && block.EndLine >= r.StartLine {
			return r
		}
	}
	return nil
}

func analyzeSourceCoverageGap(fn *sourceRange, block types.CoverageBlock) (string, []string, []string) {
	if fn == nil {
		return "", nil, nil
	}
	lines := sourceBlockLines(fn, block)
	joined := strings.Join(lines, "\n")
	trimmed := strings.TrimSpace(joined)
	switch {
	case containsBranchKeyword(trimmed, "if"):
		condition := extractSourceCondition(trimmed, "if")
		return "branch", []string{"未覆盖 if 分支: " + condition}, suggestedSourceBranchInputs(fn.Params, condition)
	case containsBranchKeyword(trimmed, "match"):
		return "branch", []string{"未覆盖 match 分支"}, suggestedGoInputs(fn.Params)
	case containsBranchKeyword(trimmed, "switch"):
		return "branch", []string{"未覆盖 switch/case 分支"}, suggestedGoInputs(fn.Params)
	case containsAny(trimmed, "Err(", "None", "return null", "throw ", "Exception", "error", "Error"):
		return "error_path", []string{"未覆盖错误或空值返回路径"}, suggestedGoInputs(fn.Params)
	case strings.Contains(trimmed, "return") || strings.Contains(trimmed, "Ok(") || strings.Contains(trimmed, "Some("):
		return "return_path", []string{"未覆盖返回路径"}, suggestedGoInputs(fn.Params)
	default:
		return "statement", []string{"未覆盖普通语句块"}, suggestedGoInputs(fn.Params)
	}
}

func sourceBlockLines(fn *sourceRange, block types.CoverageBlock) []string {
	start := block.StartLine - fn.StartLine
	end := block.EndLine - fn.StartLine
	if start < 0 {
		start = 0
	}
	if end >= len(fn.Lines) {
		end = len(fn.Lines) - 1
	}
	if start > end || len(fn.Lines) == 0 {
		return nil
	}
	return fn.Lines[start : end+1]
}

func containsBranchKeyword(source string, keyword string) bool {
	trimmed := strings.TrimSpace(source)
	return strings.HasPrefix(trimmed, keyword) || strings.Contains(trimmed, "\n"+keyword+" ") || strings.Contains(trimmed, " "+keyword+" ")
}

func extractSourceCondition(source string, keyword string) string {
	source = strings.TrimSpace(source)
	if idx := strings.Index(source, keyword); idx >= 0 {
		source = source[idx+len(keyword):]
	}
	source = strings.TrimSpace(source)
	if strings.HasPrefix(source, "(") {
		if idx := strings.Index(source, ")"); idx > 0 {
			source = source[1:idx]
		}
	}
	if idx := strings.Index(source, "{"); idx >= 0 {
		source = source[:idx]
	}
	source = strings.Trim(source, "() :")
	if source == "" {
		return "条件表达式"
	}
	return source
}

func suggestedSourceBranchInputs(params []string, condition string) []string {
	inputs := suggestedGoInputs(params)
	if condition != "" && condition != "条件表达式" {
		inputs = append([]string{"构造满足条件 `" + condition + "` 的输入"}, inputs...)
	}
	return inputs
}

func containsAny(source string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(source, value) {
			return true
		}
	}
	return false
}
