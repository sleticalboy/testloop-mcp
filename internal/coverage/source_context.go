package coverage

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/binlee/testloop-mcp/types"
)

type sourceRange struct {
	Name      string
	Kind      string
	StartLine int
	EndLine   int
	Params    []string
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
		case "cargo-test":
			ranges = parseRustFunctionRanges(sourcePath)
		case "junit":
			ranges = parseJavaMethodRanges(sourcePath)
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
		ranges = append(ranges, sourceRange{
			Name:      matches[1],
			Kind:      "function",
			StartLine: start,
			EndLine:   findBraceRangeEnd(lines, i),
			Params:    parseParamNames(matches[2]),
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
		ranges = append(ranges, sourceRange{
			Name:      name,
			Kind:      "method",
			StartLine: start,
			EndLine:   findBraceRangeEnd(lines, i),
			Params:    parseParamNames(matches[2]),
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
