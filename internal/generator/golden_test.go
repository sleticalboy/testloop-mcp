package generator

import (
	"os"
	"strings"
	"testing"

	"github.com/binlee/testloop-mcp/types"
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

func TestGeneratorCoverageTaskGoldenOutputs(t *testing.T) {
	tests := []struct {
		name   string
		source string
		golden string
		task   types.CoverageTestTask
		run    func(string, *types.CoverageTestTask) (string, error)
	}{
		{
			name:   "go coverage task",
			source: "testdata/golden/go_task.go",
			golden: "testdata/golden/go_task.golden",
			task: types.CoverageTestTask{
				ID:              "go-task-1",
				Framework:       "go-test",
				Target:          "Add",
				LineRange:       "4-4",
				GapType:         "branch",
				TestName:        "TestAddCoversGap",
				AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
				SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入"},
			},
			run: GenerateGoTestsForCoverageTask,
		},
		{
			name:   "python coverage task",
			source: "testdata/golden/python_task.py",
			golden: "testdata/golden/python_task.golden",
			task: types.CoverageTestTask{
				ID:              "pytest-task-1",
				Framework:       "pytest",
				Target:          "add",
				LineRange:       "2-2",
				GapType:         "return_path",
				TestName:        "test_add_zero_left_operand",
				AssertionFocus:  []string{"断言未覆盖返回路径的具体结果"},
				SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入"},
			},
			run: GeneratePytestTestsForCoverageTask,
		},
		{
			name:   "jest coverage task",
			source: "testdata/golden/js_task.js",
			golden: "testdata/golden/js_task.golden",
			task: types.CoverageTestTask{
				ID:              "jest-task-1",
				Framework:       "jest",
				Target:          "add",
				LineRange:       "2-2",
				GapType:         "return_path",
				TestName:        "covers add zero left operand",
				AssertionFocus:  []string{"断言未覆盖返回路径的具体结果"},
				SuggestedInputs: []string{"构造满足条件 `a === 0` 的输入"},
			},
			run: GenerateJestTestsForCoverageTask,
		},
		{
			name:   "rust coverage task",
			source: "testdata/golden/rust_task.rs",
			golden: "testdata/golden/rust_task.golden",
			task: types.CoverageTestTask{
				ID:              "rust-task-1",
				Framework:       "cargo-test",
				Target:          "Validator.check",
				LineRange:       "8-8",
				GapType:         "branch",
				TestName:        "test_validator_check_covers_gap",
				AssertionFocus:  []string{"未覆盖 match 分支"},
				SuggestedInputs: []string{"构造满足条件 `value == 0` 的输入"},
			},
			run: func(sourcePath string, task *types.CoverageTestTask) (string, error) {
				source, err := os.ReadFile(sourcePath)
				if err != nil {
					return "", err
				}
				_, code, err := GenerateRustTestsForCoverageTask(source, sourcePath, task)
				return code, err
			},
		},
		{
			name:   "java coverage task",
			source: "testdata/golden/java_task.java",
			golden: "testdata/golden/java_task.golden",
			task: types.CoverageTestTask{
				ID:              "java-task-1",
				Framework:       "junit",
				Target:          "Calculator.add",
				LineRange:       "2-2",
				GapType:         "branch",
				TestName:        "shouldCoverCalculatorAddGap",
				AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
				SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入"},
			},
			run: func(sourcePath string, task *types.CoverageTestTask) (string, error) {
				source, err := os.ReadFile(sourcePath)
				if err != nil {
					return "", err
				}
				_, code, err := GenerateJavaTestsForCoverageTask(source, sourcePath, task)
				return code, err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.run(tt.source, &tt.task)
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
