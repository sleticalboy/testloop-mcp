package generator

import (
	"path/filepath"
	"strings"
)

// GeneratedTestsAction classifies generated test code before running it.
func GeneratedTestsAction(code, fileNameOrExt string) string {
	ext := strings.ToLower(fileNameOrExt)
	if !strings.HasPrefix(ext, ".") {
		ext = strings.ToLower(filepath.Ext(fileNameOrExt))
	}
	switch ext {
	case ".go":
		if strings.Contains(code, `t.Skip("TODO: fill in meaningful test inputs and expected values")`) {
			return "manual_review"
		}
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		if strings.Contains(code, ".skip(") || strings.Contains(code, "manual_review_") {
			return "manual_review"
		}
	case ".py":
		if strings.Contains(code, "pytest.skip(") || strings.Contains(code, "__import__('pytest').skip(") || strings.Contains(code, "manual_review_") {
			return "manual_review"
		}
	case ".java":
		if strings.Contains(code, "Assumptions.assumeTrue(false") || strings.Contains(code, "manual_review_") {
			return "manual_review"
		}
	}
	return "ready"
}
