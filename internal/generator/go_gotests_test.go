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

func TestGenerateGoTestsForCoverageTaskUsesBranchSuggestedInputs(t *testing.T) {
	src := `package sample

func Add(a, b int) int {
	if a == 0 {
		return b
	}
	return a + b
}
`
	srcPath := filepath.Join(t.TempDir(), "calc.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-branch",
		Framework:       "go-test",
		Target:          "Add",
		LineRange:       "4-4",
		GapType:         "branch",
		TestName:        "TestAddZeroBranch",
		MissingBranches: []string{"未覆盖 if 分支: a == 0"},
		SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入"},
		AssertionFocus:  []string{"断言分支返回值"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestAddZeroBranch(t *testing.T)",
		"\"coverage branch gap\"",
		"skip: false",
		"a:    0,",
		"b:    2,",
		"ret0: 2,",
		"got := Add(tt.a, tt.b)",
		"if got != tt.ret0",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, notWant := range []string{
		"skip: true",
		"TODO: 填写有意义的输入",
	} {
		if strings.Contains(code, notWant) {
			t.Fatalf("did not expect %q in generated code:\n%s", notWant, code)
		}
	}
}

func TestGenerateGoTestsForCoverageTaskUsesStringBoolAndNilBranchInputs(t *testing.T) {
	src := `package sample

type User struct {
	Name string
}

func Label(name string) string {
	if name == "" {
		return "anonymous"
	}
	return name
}

func Normalize(name string) string {
	if name != "" {
		return name
	}
	return "anonymous"
}

func Toggle(enabled bool) string {
	if enabled == false {
		return "off"
	}
	return "on"
}

func UserName(user *User) string {
	if user == nil {
		return "missing"
	}
	return user.Name
}

func ActiveUser(user *User) string {
	if user != nil {
		return "present"
	}
	return "missing"
}

func ErrorMessage(err error) string {
	if err != nil {
		return "failed"
	}
	return "ok"
}

func NoError(err error) string {
	if err == nil {
		return "ok"
	}
	return "failed"
}

func SkipLabel(skip bool) string {
	if skip == false {
		return "run"
	}
	return "skip"
}
`
	srcPath := filepath.Join(t.TempDir(), "branches.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		target    string
		testName  string
		condition string
		wants     []string
	}{
		{
			name:      "empty string",
			target:    "Label",
			testName:  "TestLabelEmptyBranch",
			condition: `name == ""`,
			wants:     []string{`nameValue: ""`, `got := Label(tt.nameValue)`, `ret0:      "anonymous"`},
		},
		{
			name:      "non empty string",
			target:    "Normalize",
			testName:  "TestNormalizeNonEmptyBranch",
			condition: `name != ""`,
			wants:     []string{`nameValue: "test"`, `got := Normalize(tt.nameValue)`, `ret0:      "test"`},
		},
		{
			name:      "false bool",
			target:    "Toggle",
			testName:  "TestToggleFalseBranch",
			condition: "enabled == false",
			wants:     []string{`enabled: false`, `ret0:    "off"`},
		},
		{
			name:      "nil pointer",
			target:    "UserName",
			testName:  "TestUserNameNilBranch",
			condition: "user == nil",
			wants:     []string{`user: nil`, `ret0: "missing"`},
		},
		{
			name:      "non nil pointer",
			target:    "ActiveUser",
			testName:  "TestActiveUserNonNilBranch",
			condition: "user != nil",
			wants:     []string{`user: &User{}`, `ret0: "present"`},
		},
		{
			name:      "non nil error",
			target:    "ErrorMessage",
			testName:  "TestErrorMessageNonNilBranch",
			condition: "err != nil",
			wants:     []string{`"errors"`, `err:  errors.New("test")`, `ret0: "failed"`},
		},
		{
			name:      "nil error",
			target:    "NoError",
			testName:  "TestNoErrorNilBranch",
			condition: "err == nil",
			wants:     []string{`err:  nil`, `ret0: "ok"`},
		},
		{
			name:      "skip parameter",
			target:    "SkipLabel",
			testName:  "TestSkipLabelFalseBranch",
			condition: "skip == false",
			wants:     []string{`skipValue: false`, `got := SkipLabel(tt.skipValue)`, `ret0:      "run"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := types.CoverageTestTask{
				ID:              "go-test-branch-" + tt.target,
				Framework:       "go-test",
				Target:          tt.target,
				LineRange:       "1-1",
				GapType:         "branch",
				TestName:        tt.testName,
				SuggestedInputs: []string{"构造满足条件 `" + tt.condition + "` 的输入"},
				AssertionFocus:  []string{"断言分支返回值"},
			}

			code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
			if err != nil {
				t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
			}
			for _, want := range append([]string{
				"func " + tt.testName + "(t *testing.T)",
				"skip:",
				"got := " + tt.target + "(",
				"if got != tt.ret0",
			}, tt.wants...) {
				if !strings.Contains(code, want) {
					t.Fatalf("expected %q in generated code:\n%s", want, code)
				}
			}
			if strings.Contains(code, "skip: true") || strings.Contains(code, "TODO: 填写有意义的输入") {
				t.Fatalf("did not expect skipped TODO in generated code:\n%s", code)
			}
		})
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

func TestGenerateGoTestsForCoverageTaskAssertsTimeNowFormatReturn(t *testing.T) {
	src := `package sample

import "time"

func GetNowDate() string {
	return time.Now().Format("2006-01-02")
}
`
	srcPath := filepath.Join(t.TempDir(), "time.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:             "go-test-time-format",
		Framework:      "go-test",
		Target:         "GetNowDate",
		LineRange:      "5-7",
		GapType:        "return_path",
		TestName:       "TestGetNowDate",
		AssertionFocus: []string{"断言日期格式"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"\"time\"",
		"func TestGetNowDate(t *testing.T)",
		"\"coverage return path\"",
		"skip:   false",
		"layout: \"2006-01-02\"",
		"got := GetNowDate()",
		"if _, err := time.Parse(tt.layout, got); err != nil",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, notWant := range []string{
		"skip: true",
		"_ = got",
		"ret0 string",
		"if got != tt.ret0",
	} {
		if strings.Contains(code, notWant) {
			t.Fatalf("did not expect %q in generated code:\n%s", notWant, code)
		}
	}
}

func TestGenerateGoTestsForCoverageTaskAssertsTimeDateZeroReturn(t *testing.T) {
	src := `package sample

import "time"

func GetCurrentDate() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}
`
	srcPath := filepath.Join(t.TempDir(), "time.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:             "go-test-current-date",
		Framework:      "go-test",
		Target:         "GetCurrentDate",
		LineRange:      "5-7",
		GapType:        "return_path",
		TestName:       "TestGetCurrentDate",
		AssertionFocus: []string{"断言日期归零"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestGetCurrentDate(t *testing.T)",
		"\"coverage return path\"",
		"skip: false",
		"got := GetCurrentDate()",
		"got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Nanosecond() != 0",
		"want date boundary",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, notWant := range []string{
		"skip: true",
		"_ = got",
		"ret0 time.Time",
		"\"reflect\"",
		"if !reflect.DeepEqual",
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
