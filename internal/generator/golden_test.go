package generator

import (
	"os"
	"strings"
	"testing"
)

func TestGeneratorGoldenOutputs(t *testing.T) {
	tests := []struct {
		name   string
		source string
		golden string
		run    func(string) (string, error)
	}{
		{
			name:   "go simple pure function",
			source: "testdata/golden/go_simple.go",
			golden: "testdata/golden/go_simple.golden",
			run:    GenerateGoTests,
		},
		{
			name:   "python branch return",
			source: "testdata/golden/python_branch.py",
			golden: "testdata/golden/python_branch.golden",
			run:    GeneratePytestTests,
		},
		{
			name:   "js branch return",
			source: "testdata/golden/js_branch.js",
			golden: "testdata/golden/js_branch.golden",
			run:    GenerateJestTests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.run(tt.source)
			if err != nil {
				t.Fatalf("generate: %v", err)
			}
			wantBytes, err := os.ReadFile(tt.golden)
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			want := string(wantBytes)
			if strings.TrimSpace(got) != strings.TrimSpace(want) {
				t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
			}
		})
	}
}
