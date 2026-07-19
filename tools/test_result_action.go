package tools

import "github.com/sleticalboy/testloop-mcp/types"

func annotateTestResultAction(result *types.TestResult) {
	if result == nil || result.Action != "" {
		return
	}
	result.Action = testResultAction(*result)
}

func testResultAction(result types.TestResult) string {
	if result.Status == "fail" {
		if len(result.FixSuggestions) > 0 {
			return "apply_fix_suggestions"
		}
		if len(result.Failures) > 0 {
			return "inspect_failures"
		}
		return "inspect_test_runner"
	}
	if result.Passed == 0 {
		return "manual_review"
	}
	return "ready"
}
