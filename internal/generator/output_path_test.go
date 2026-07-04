package generator

import "testing"

func TestTestFileName(t *testing.T) {
	tests := map[string]string{
		"demo/calc.go":         "demo/calc_test.go",
		"demo/sum.js":          "demo/sum.test.js",
		"demo/button.jsx":      "demo/button.test.jsx",
		"demo/service.ts":      "demo/service.test.ts",
		"demo/widget.tsx":      "demo/widget.test.tsx",
		"demo/calc.py":         "demo/test_calc.py",
		"demo/calc.rs":         "demo/calc.test.rs",
		"demo/Calculator.java": "demo/Calculator.test.java",
	}

	for src, want := range tests {
		if got := TestFileName(src); got != want {
			t.Fatalf("TestFileName(%q) = %q, want %q", src, got, want)
		}
	}
}
