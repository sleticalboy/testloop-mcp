package generator

import "testing"

func TestGeneratedTestsAction(t *testing.T) {
	tests := []struct {
		name          string
		fileNameOrExt string
		code          string
		want          string
	}{
		{name: "go ready", fileNameOrExt: ".go", code: "func TestAdd(t *testing.T) {}", want: "ready"},
		{name: "go todo skip", fileNameOrExt: "calc_test.go", code: `func TestAdd(t *testing.T) { t.Skip("TODO: fill in meaningful test inputs and expected values") }`, want: "manual_review"},
		{name: "js manual review", fileNameOrExt: ".ts", code: "it.skip('manual', () => {})", want: "manual_review"},
		{name: "python manual review", fileNameOrExt: ".py", code: "__import__('pytest').skip('manual_review_internal: helper')", want: "manual_review"},
		{name: "java manual review", fileNameOrExt: ".java", code: "org.junit.jupiter.api.Assumptions.assumeTrue(false, \"manual_review_unreachable: line\")", want: "manual_review"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GeneratedTestsAction(tt.code, tt.fileNameOrExt); got != tt.want {
				t.Fatalf("GeneratedTestsAction = %q, want %q", got, tt.want)
			}
		})
	}
}
