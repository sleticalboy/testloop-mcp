package generator

import (
	"regexp"
	"strings"
)

func replaceIdentifier(expr string, name string, value string) string {
	if name == "" {
		return expr
	}
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
	return re.ReplaceAllString(expr, value)
}

func stripQuotedLiterals(expr string) string {
	var sb strings.Builder
	inQuote := rune(0)
	escaped := false
	for _, ch := range expr {
		if inQuote != 0 {
			if escaped {
				escaped = false
				sb.WriteRune(' ')
				continue
			}
			if ch == '\\' {
				escaped = true
				sb.WriteRune(' ')
				continue
			}
			if ch == inQuote {
				inQuote = 0
			}
			sb.WriteRune(' ')
			continue
		}
		if ch == '\'' || ch == '"' || ch == '`' {
			inQuote = ch
			sb.WriteRune(' ')
			continue
		}
		sb.WriteRune(ch)
	}
	return sb.String()
}

var exprIdentifierRe = regexp.MustCompile(`[A-Za-z_$][A-Za-z0-9_$]*`)

func hasUnknownIdentifiers(expr string, allowed map[string]bool) bool {
	for _, match := range exprIdentifierRe.FindAllString(expr, -1) {
		if allowed[match] {
			continue
		}
		return true
	}
	return false
}
