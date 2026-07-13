package generator

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/sleticalboy/testloop-mcp/types"
)

var (
	taskBacktickRe  = regexp.MustCompile("`([^`]+)`")
	taskConditionRe = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*(?:===|==|=|is)\s*(.+?)\s*$`)
)

func taskTargetMatches(target, className, name string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return true
	}
	if target == name || target == className {
		return true
	}
	if className == "" {
		return false
	}
	return target == className+"."+name ||
		target == className+"_"+name ||
		strings.HasSuffix(target, "."+className+"."+name)
}

func coverageTaskInputValues(task *types.CoverageTestTask, language string) map[string]string {
	values := map[string]string{}
	if task == nil {
		return values
	}

	hints := make([]string, 0, len(task.SuggestedInputs)+len(task.MissingBranches))
	hints = append(hints, task.SuggestedInputs...)
	hints = append(hints, task.MissingBranches...)
	for _, hint := range hints {
		for _, candidate := range taskConditionCandidates(hint) {
			matches := taskConditionRe.FindStringSubmatch(candidate)
			if len(matches) != 3 {
				continue
			}
			param := strings.TrimSpace(matches[1])
			value := normalizeTaskLiteral(strings.TrimSpace(matches[2]), language)
			if param != "" && value != "" {
				values[param] = value
			}
		}
	}
	return values
}

func taskConditionCandidates(hint string) []string {
	var candidates []string
	for _, m := range taskBacktickRe.FindAllStringSubmatch(hint, -1) {
		if len(m) == 2 {
			candidates = append(candidates, strings.TrimSpace(m[1]))
		}
	}
	if len(candidates) == 0 {
		candidates = append(candidates, strings.TrimSpace(hint))
	}
	return candidates
}

func normalizeTaskLiteral(value, language string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimRightFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || r == ',' || r == ';' || r == ':'
	})
	switch strings.ToLower(value) {
	case "undefined":
		if language == "javascript" {
			return "undefined"
		}
		if language == "python" || language == "rust" {
			return "None"
		}
		return "null"
	case "null", "none":
		if language == "python" || language == "rust" {
			return "None"
		}
		return "null"
	case "true":
		if language == "python" {
			return "True"
		}
		return "true"
	case "false":
		if language == "python" {
			return "False"
		}
		return "false"
	}
	return value
}

func coverageTaskComment(task *types.CoverageTestTask) string {
	if task == nil {
		return ""
	}
	parts := []string{}
	if task.ID != "" {
		parts = append(parts, task.ID)
	}
	if task.LineRange != "" {
		parts = append(parts, "lines "+task.LineRange)
	}
	if len(task.AssertionFocus) > 0 {
		parts = append(parts, strings.Join(task.AssertionFocus, "; "))
	}
	if len(task.SuggestedInputs) > 0 {
		parts = append(parts, strings.Join(task.SuggestedInputs, "; "))
	}
	return sanitizeCoverageTaskComment(strings.Join(parts, " | "))
}

func sanitizeCoverageTaskComment(comment string) string {
	fields := strings.Fields(comment)
	if len(fields) == 0 {
		return ""
	}
	comment = strings.Join(fields, " ")
	const maxCoverageTaskCommentLen = 400
	if len(comment) > maxCoverageTaskCommentLen {
		comment = strings.TrimSpace(comment[:maxCoverageTaskCommentLen]) + "..."
	}
	return comment
}

func sanitizePythonTestName(name, fallback string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = fallback
	}
	name = strings.TrimPrefix(name, "def ")
	name = strings.TrimSuffix(name, "()")
	var sb strings.Builder
	for _, r := range name {
		if r == '_' || unicode.IsLetter(r) || (unicode.IsDigit(r) && sb.Len() > 0) {
			sb.WriteRune(r)
			continue
		}
		if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "_") {
			sb.WriteByte('_')
		}
	}
	got := strings.Trim(sb.String(), "_")
	if got == "" {
		got = fallback
	}
	if !strings.HasPrefix(got, "test_") {
		got = "test_" + got
	}
	return got
}
