package parser

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestParseFrameworkFailureFixtures(t *testing.T) {
	tests := []struct {
		name      string
		framework string
		file      string
		testName  string
		error     string
		location  string
		expected  string
		received  string
	}{
		{
			name:      "jest",
			framework: "jest",
			file:      "jest_failure.txt",
			testName:  "adds 1 + 2 to equal 3",
			error:     "expect(received).toBe(expected)",
			location:  "sum.test.js:5:15",
			expected:  "3",
			received:  "4",
		},
		{
			name:      "vitest",
			framework: "vitest",
			file:      "vitest_failure.txt",
			testName:  "sum > adds values",
			error:     "AssertionError: expected 4 to be 3 // Object.is equality",
			location:  "src/sum.test.ts:8:18",
			expected:  "3",
			received:  "4",
		},
		{
			name:      "pytest",
			framework: "pytest",
			file:      "pytest_failure.txt",
			testName:  "test_divide",
			error:     "ValueError: division by zero",
			location:  "calc.py:7:0",
			expected:  "divide(1, 0)",
		},
		{
			name:      "mocha",
			framework: "mocha",
			file:      "mocha_failure.txt",
			testName:  "calc divide() should handle division by zero",
			error:     "AssertionError: expected 4 to equal 3",
			location:  "test/calc.test.js:12:18",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := readParserFixture(t, tt.file)
			result := ParseTestOutput(output, tt.framework)
			if result.Status != "fail" || result.Failed == 0 {
				t.Fatalf("expected failed result, got status=%s failed=%d", result.Status, result.Failed)
			}
			if len(result.Failures) != 1 {
				t.Fatalf("expected one structured failure, got %d", len(result.Failures))
			}
			failure := result.Failures[0]
			if failure.TestName != tt.testName {
				t.Errorf("test_name = %q, want %q", failure.TestName, tt.testName)
			}
			if failure.Error != tt.error {
				t.Errorf("error = %q, want %q", failure.Error, tt.error)
			}
			if got := formatFailureLocation(failure.File, failure.Line, failure.Column); got != tt.location {
				t.Errorf("location = %q, want %q", got, tt.location)
			}
			if tt.expected != "" && failure.Expected != tt.expected {
				t.Errorf("expected = %q, want %q", failure.Expected, tt.expected)
			}
			if tt.received != "" && failure.Received != tt.received {
				t.Errorf("received = %q, want %q", failure.Received, tt.received)
			}
		})
	}
}

func TestParseTestOutputDefaultsUnknownFrameworkToGoTest(t *testing.T) {
	output := `=== RUN   TestAdd
--- PASS: TestAdd (0.00s)
PASS
ok  	example.com/calc	0.001s`

	result := ParseTestOutput(output, "unknown")

	if result.Framework != "go-test" {
		t.Fatalf("Framework = %q, want go-test", result.Framework)
	}
	if result.Status != "pass" || result.Passed != 1 || result.Total != 1 {
		t.Fatalf("Unexpected go-test result: status=%s passed=%d total=%d", result.Status, result.Passed, result.Total)
	}
}

func readParserFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func formatFailureLocation(file string, line int, column int) string {
	return file + ":" + strconv.Itoa(line) + ":" + strconv.Itoa(column)
}
