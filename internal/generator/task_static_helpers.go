package generator

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/sleticalboy/testloop-mcp/types"
)

var (
	taskBacktickRe      = regexp.MustCompile("`([^`]+)`")
	taskConditionRe     = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*(===|!==|==|!=|>=|<=|>|<|=|is\s+not|is)\s*(.+?)\s*$`)
	taskConditionPartRe = regexp.MustCompile(`\s*(?:&&|\band\b)\s*`)
	taskEqualsCallRe    = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_.]*)\.equals\(\s*([A-Za-z_][A-Za-z0-9_]*)`)
	taskIdentifierRe    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
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
			for _, condition := range taskConditionParts(candidate) {
				if language == "java" {
					if matches := taskEqualsCallRe.FindStringSubmatch(condition); len(matches) == 3 {
						values[strings.TrimSpace(matches[2])] = strings.TrimSpace(matches[1])
						continue
					}
				}
				param, value, ok := parseTaskConditionValue(condition, language)
				if ok {
					values[param] = value
				}
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

func taskConditionParts(candidate string) []string {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return nil
	}
	parts := taskConditionPartRe.Split(candidate, -1)
	if len(parts) == 0 {
		return []string{candidate}
	}
	conditions := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			conditions = append(conditions, part)
		}
	}
	if len(conditions) == 0 {
		return []string{candidate}
	}
	return conditions
}

func parseTaskConditionValue(condition, language string) (string, string, bool) {
	condition = strings.TrimSpace(condition)
	if strings.HasPrefix(condition, "!") {
		param := strings.TrimSpace(strings.TrimPrefix(condition, "!"))
		if taskIdentifierRe.MatchString(param) {
			return param, normalizeTaskLiteral("false", language), true
		}
	}
	if strings.HasPrefix(condition, "not ") {
		param := strings.TrimSpace(strings.TrimPrefix(condition, "not "))
		if taskIdentifierRe.MatchString(param) {
			return param, normalizeTaskLiteral("false", language), true
		}
	}
	if taskIdentifierRe.MatchString(condition) {
		return condition, normalizeTaskLiteral("true", language), true
	}

	matches := taskConditionRe.FindStringSubmatch(condition)
	if len(matches) != 4 {
		return "", "", false
	}
	param := strings.TrimSpace(matches[1])
	value := normalizeTaskConditionLiteral(strings.TrimSpace(matches[3]), strings.TrimSpace(matches[2]), language)
	return param, value, param != "" && value != ""
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

func normalizeTaskConditionLiteral(value, operator, language string) string {
	normalized := normalizeTaskLiteral(value, language)
	switch strings.ToLower(strings.TrimSpace(operator)) {
	case ">", ">=":
		if operator == ">" {
			return incrementIntegerLiteral(normalized)
		}
	case "<", "<=":
		if operator == "<" {
			return decrementIntegerLiteral(normalized)
		}
	case "!=", "!==", "is not":
		return differentTaskLiteral(normalized, language)
	}
	return normalized
}

func incrementIntegerLiteral(value string) string {
	if parsed, ok := parseIntegerLiteral(value); ok {
		return strconv.FormatInt(parsed+1, 10)
	}
	return value
}

func decrementIntegerLiteral(value string) string {
	if parsed, ok := parseIntegerLiteral(value); ok {
		return strconv.FormatInt(parsed-1, 10)
	}
	return value
}

func parseIntegerLiteral(value string) (int64, bool) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return parsed, err == nil
}

func differentTaskLiteral(value, language string) string {
	switch value {
	case "true", "True":
		if language == "python" {
			return "False"
		}
		return "false"
	case "false", "False":
		if language == "python" {
			return "True"
		}
		return "true"
	case "null", "None", "undefined":
		return nonNullTaskLiteral(language)
	}
	if parsed, ok := parseIntegerLiteral(value); ok {
		return strconv.FormatInt(parsed+1, 10)
	}
	if len(value) >= 2 {
		quote := value[:1]
		if (quote == `"` || quote == `'`) && strings.HasSuffix(value, quote) {
			return quote + "__testloop_other__" + quote
		}
	}
	return value
}

func nonNullTaskLiteral(language string) string {
	switch language {
	case "python":
		return "object()"
	case "javascript":
		return "{}"
	case "java":
		return "\"__testloop_other__\""
	case "rust":
		return "Some(0)"
	default:
		return "1"
	}
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
