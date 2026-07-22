package tools

import (
	"os"
	"regexp"
	"testing"
)

func TestProjectValidationEntrypointsEnableCoverageTargetHit(t *testing.T) {
	coverageFlag := regexp.MustCompile(`Coverage:\s+true,`)
	for _, file := range []string{
		"go_project_validation_integration_test.go",
		"js_project_validation_integration_test.go",
		"py_project_validation_integration_test.go",
		"rust_project_validation_integration_test.go",
	} {
		t.Run(file, func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read %s: %v", file, err)
			}
			if !coverageFlag.Match(data) {
				t.Fatalf("%s must pass Coverage: true to validate_coverage_task", file)
			}
		})
	}
}
