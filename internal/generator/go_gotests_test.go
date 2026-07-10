package generator

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sleticalboy/testloop-mcp/types"
)

func TestGenerateGoTestsPreferredUsesGotestsWhenAvailable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gotests is unix-only")
	}

	srcPath := writeTempGoSource(t)
	fakeBin := writeFakeGotests(t, `#!/bin/sh
if [ "$1" != "-all" ]; then
  echo "missing -all" >&2
  exit 2
fi
if [ "$2" != "calc.go" ]; then
  echo "unexpected source: $2" >&2
  exit 3
fi
cat <<'EOF'
package sample

import "testing"

func TestFromGotests(t *testing.T) {}
EOF
`)

	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	code, err := GenerateGoTestsPreferred(srcPath)
	if err != nil {
		t.Fatalf("GenerateGoTestsPreferred() error = %v", err)
	}
	if !strings.Contains(code, "TestFromGotests") {
		t.Fatalf("expected gotests output, got:\n%s", code)
	}
}

func TestGenerateGoTestsPreferredFallsBackWhenGotestsFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gotests is unix-only")
	}

	srcPath := writeTempGoSource(t)
	fakeBin := writeFakeGotests(t, `#!/bin/sh
echo "boom" >&2
exit 42
`)

	t.Setenv("PATH", fakeBin)

	code, err := GenerateGoTestsPreferred(srcPath)
	if err != nil {
		t.Fatalf("GenerateGoTestsPreferred() fallback error = %v", err)
	}
	if strings.Contains(code, "TestFromGotests") {
		t.Fatalf("expected fallback output, got gotests output:\n%s", code)
	}
	if !strings.Contains(code, "func TestAdd") || !strings.Contains(code, "skip: false") {
		t.Fatalf("expected built-in fallback output, got:\n%s", code)
	}
	if !strings.Contains(code, "a:    1,") || !strings.Contains(code, "b:    2,") || !strings.Contains(code, "ret0: 1 + 2,") {
		t.Fatalf("expected seeded exact test case, got:\n%s", code)
	}
}

func TestGenerateGoTestsSeedsSimplePureFunction(t *testing.T) {
	srcPath := writeTempGoSource(t)

	code, err := GenerateGoTests(srcPath)
	if err != nil {
		t.Fatalf("GenerateGoTests() error = %v", err)
	}
	for _, want := range []string{
		"name: \"simple\"",
		"skip: false",
		"a:    1,",
		"b:    2,",
		"ret0: 1 + 2,",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "name: \"todo\"") {
		t.Fatalf("simple pure function should not generate TODO case:\n%s", code)
	}
}

func TestGenerateGoTestsCoversCompositeSourceBranches(t *testing.T) {
	src := `package sample

type Store struct {
	Name string
	Count int
}

type Writer interface {
	Write(data []byte) (int, error)
	Close()
}

func Identity[T any](value T) T {
	return value
}

func Join(prefix string, parts ...string) string {
	return prefix
}

func (s *Store) Names(limit int) []string {
	return []string{s.Name}
}
`
	srcPath := filepath.Join(t.TempDir(), "composite.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	code, err := GenerateGoTests(srcPath)
	if err != nil {
		t.Fatalf("GenerateGoTests() error = %v", err)
	}
	for _, want := range []string{
		"\"reflect\"",
		"func makeTestStore() Store",
		"Name:  \"\"",
		"Count: 0",
		"type WriterMock struct",
		"WriteFn func([]byte) (int, error)",
		"func (m *WriterMock) Close()",
		"func TestIdentity(t *testing.T)",
		"value int",
		"Identity[int](tt.value)",
		"func TestJoin(t *testing.T)",
		"parts  []string",
		"Join(tt.prefix, tt.parts...)",
		"func TestStore_Names(t *testing.T)",
		"s := &Store{}",
		"got := s.Names(tt.limit)",
		"if !reflect.DeepEqual(got, tt.ret0)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
}

func TestGenerateGoTestsReturnsMessageWhenNoFuncsOrInterfaces(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "empty.go")
	if err := os.WriteFile(srcPath, []byte("package sample\n\ntype Store struct{ Name string }\n"), 0644); err != nil {
		t.Fatal(err)
	}

	code, err := GenerateGoTests(srcPath)
	if err != nil {
		t.Fatalf("GenerateGoTests() error = %v", err)
	}
	if code != "// 未发现需要生成测试的 exported 函数或接口" {
		t.Fatalf("GenerateGoTests() = %q", code)
	}
}

func TestGenerateGoTestsForCoverageTaskTargetsFunction(t *testing.T) {
	src := `package sample

func Add(a, b int) int {
	return a + b
}

func Sub(a, b int) int {
	return a - b
}
`
	srcPath := filepath.Join(t.TempDir(), "calc.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:             "go-test-1",
		Framework:      "go-test",
		Target:         "Add",
		LineRange:      "3-3",
		GapType:        "branch",
		TestName:       "TestAddCoverageGap",
		AssertionFocus: []string{"断言未覆盖分支的返回值或副作用"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestAddCoverageGap",
		"coverage task: go-test-1 | lines 3-3",
		"name: \"coverage branch gap\"",
		"Add(tt.a, tt.b)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "TestSub") || strings.Contains(code, "Sub(tt.a, tt.b)") {
		t.Fatalf("task-aware generation should only target Add:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskUsesSmokeCaseForNoArgReturn(t *testing.T) {
	src := `package sample

import "time"

func GetCurrentDate() time.Time {
	return time.Now()
}
`
	srcPath := filepath.Join(t.TempDir(), "time.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:             "go-test-time",
		Framework:      "go-test",
		Target:         "GetCurrentDate",
		LineRange:      "5-7",
		GapType:        "return_path",
		TestName:       "TestGetCurrentDate",
		AssertionFocus: []string{"覆盖未执行返回路径"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestGetCurrentDate(t *testing.T)",
		"name: \"coverage return path\"",
		"skip: false",
		"got := GetCurrentDate()",
		"_ = got",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, notWant := range []string{
		"skip: true",
		"TODO: 填写有意义的输入",
		"ret0 string",
		"\"reflect\"",
		"if got != tt.ret0",
	} {
		if strings.Contains(code, notWant) {
			t.Fatalf("did not expect %q in generated code:\n%s", notWant, code)
		}
	}
}

func TestGenerateTestsWithProviderOptionsUsesGoCoverageTask(t *testing.T) {
	src := `package sample

func Add(a, b int) int {
	return a + b
}

func Sub(a, b int) int {
	return a - b
}
`
	srcPath := filepath.Join(t.TempDir(), "calc.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:        "go-test-1",
		Framework: "go-test",
		Target:    "Add",
		LineRange: "3-3",
		GapType:   "branch",
		TestName:  "TestAddCoverageGap",
	}

	code, err := GenerateTestsWithProviderOptions(context.Background(), srcPath, StaticProvider{}, GenerateTestsOptions{CoverageTask: &task})
	if err != nil {
		t.Fatalf("GenerateTestsWithProviderOptions() error = %v", err)
	}
	if !strings.Contains(code, "func TestAddCoverageGap") || strings.Contains(code, "TestSub") {
		t.Fatalf("expected task-aware static Go output, got:\n%s", code)
	}
}

func writeTempGoSource(t *testing.T) string {
	t.Helper()

	src := `package sample

func Add(a, b int) int {
	return a + b
}
`
	path := filepath.Join(t.TempDir(), "calc.go")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFakeGotests(t *testing.T, script string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "gotests")
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}
