package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateTestsStaticDispatchesRustAndJava(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		source   string
		want     []string
	}{
		{
			name:     "rust",
			fileName: "lib.rs",
			source: `pub fn add(a: i32, b: i32) -> i32 {
    a + b
}
`,
			want: []string{"#[test]", "fn test_add()", "let result = add(0, 0);"},
		},
		{
			name:     "java",
			fileName: "Calculator.java",
			source: `public class Calculator {
    public int add(int a, int b) {
        return a + b;
    }
}
`,
			want: []string{"class CalculatorTest", "@Test", "void add()"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcPath := writeGeneratorStaticSource(t, tt.fileName, tt.source)

			code, err := GenerateTestsStatic(srcPath)
			if err != nil {
				t.Fatalf("GenerateTestsStatic() error = %v", err)
			}
			for _, want := range tt.want {
				if !strings.Contains(code, want) {
					t.Fatalf("expected %q in generated code:\n%s", want, code)
				}
			}
		})
	}
}

func TestGenerateTestsDelegatesToStaticGenerator(t *testing.T) {
	srcPath := writeGeneratorStaticSource(t, "calc.py", `def add(a, b):
    return a + b
`)

	code, err := GenerateTests(srcPath)
	if err != nil {
		t.Fatalf("GenerateTests() error = %v", err)
	}
	if !strings.Contains(code, "def test_add():") {
		t.Fatalf("expected pytest output, got:\n%s", code)
	}
}

func TestGenerateTestsStaticRejectsUnsupportedExtension(t *testing.T) {
	srcPath := writeGeneratorStaticSource(t, "notes.txt", "not source")

	_, err := GenerateTestsStatic(srcPath)
	if err == nil || !strings.Contains(err.Error(), "不支持的文件类型") {
		t.Fatalf("GenerateTestsStatic() error = %v, want unsupported extension error", err)
	}
}

func TestGenerateTestsStaticReportsReadErrorsForRustAndJava(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		want     string
	}{
		{name: "rust", fileName: "missing.rs", want: "读取 Rust 源文件失败"},
		{name: "java", fileName: "Missing.java", want: "读取 Java 源文件失败"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcPath := filepath.Join(t.TempDir(), tt.fileName)

			_, err := GenerateTestsStatic(srcPath)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("GenerateTestsStatic() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func writeGeneratorStaticSource(t *testing.T, name string, source string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
