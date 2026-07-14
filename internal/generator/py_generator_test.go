package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sleticalboy/testloop-mcp/types"
)

func TestGeneratePytestTests(t *testing.T) {
	code, err := GeneratePytestTests("../../demo/calc.py")
	if err != nil {
		t.Fatalf("GeneratePytestTests 失败: %v", err)
	}

	if !strings.Contains(code, "from calc import") {
		t.Error("缺少 Python import 导入")
	}

	expectedFuncs := []string{"add", "divide", "format_text", "fetch_data"}
	for _, name := range expectedFuncs {
		if !strings.Contains(code, "def test_"+name+"(") {
			t.Errorf("缺少函数测试: test_%s", name)
		}
	}

	if !strings.Contains(code, "class TestCalculator:") {
		t.Error("缺少 TestCalculator 类测试")
	}

	if !strings.Contains(code, "def test_init(self):") {
		t.Error("缺少 __init__ 测试")
	}

	if !strings.Contains(code, "def test_add(self):") {
		t.Error("缺少 Calculator.add 方法测试")
	}
	if !strings.Contains(code, "def test_divide(self):") {
		t.Error("缺少 Calculator.divide 方法测试")
	}

	// 验证 staticmethod
	if !strings.Contains(code, "Calculator.version(") {
		t.Error("缺少 staticmethod 测试")
	}

	// 验证 self 参数被正确剥离
	if strings.Contains(code, "instance.add(self") {
		t.Error("方法调用中不应包含 self 参数")
	}

	t.Logf("生成的 pytest 测试:\n%s", code)
}

func TestTreeSitterPython_ParsesAllDeclarations(t *testing.T) {
	source := []byte(`def add(a, b):
    return a + b

class Calculator:
    def add(self, a, b):
        return a + b

    @staticmethod
    def version():
        return "1.0.0"
`)
	funcs, classes := parsePyWithTreeSitter(source)

	if len(funcs) != 1 {
		t.Fatalf("期望 1 个顶层函数, got %d", len(funcs))
	}
	if funcs[0].Name != "add" {
		t.Errorf("funcs[0].Name = %s, want add", funcs[0].Name)
	}

	if len(classes) != 1 {
		t.Fatalf("期望 1 个类, got %d", len(classes))
	}
	cls := classes[0]
	if cls.Name != "Calculator" {
		t.Errorf("class name = %s, want Calculator", cls.Name)
	}
	// add + divide + version (not __init__)
	if len(cls.Methods) != 2 {
		t.Errorf("期望 2 个方法 (add, version), got %d", len(cls.Methods))
	}

	// 验证 staticmethod
	foundStatic := false
	for _, m := range cls.Methods {
		if m.Name == "version" && m.IsStatic {
			foundStatic = true
		}
	}
	if !foundStatic {
		t.Error("version 应该是 static method")
	}
}

func TestTreeSitterPython_AsyncDetection(t *testing.T) {
	source := []byte(`async def fetch_data(url):
    return await fetch(url)

def sync_fn(x):
    return x * 2
`)
	funcs, _ := parsePyWithTreeSitter(source)

	if len(funcs) != 2 {
		t.Fatalf("期望 2 个函数, got %d", len(funcs))
	}

	foundAsync := false
	foundSync := false
	for _, fn := range funcs {
		if fn.Name == "fetch_data" && fn.IsAsync {
			foundAsync = true
		}
		if fn.Name == "sync_fn" && !fn.IsAsync {
			foundSync = true
		}
	}
	if !foundAsync {
		t.Error("fetch_data 应该是 async")
	}
	if !foundSync {
		t.Error("sync_fn 不应该是 async")
	}
}

func TestTreeSitterPython_ParamsExtraction(t *testing.T) {
	source := []byte(`def format_text(text: str, prefix: str = "", *args, **kwargs) -> str:
    return prefix + text
`)
	funcs, _ := parsePyWithTreeSitter(source)

	if len(funcs) != 1 {
		t.Fatalf("期望 1 个函数, got %d", len(funcs))
	}
	fn := funcs[0]
	if len(fn.Params) != 4 {
		t.Fatalf("期望 4 个参数, got %d: %+v", len(fn.Params), fn.Params)
	}
	if fn.Params[0].Name != "text" {
		t.Errorf("param[0].Name = %s, want text", fn.Params[0].Name)
	}
	if !fn.Params[1].HasDefault {
		t.Error("param[1] 应该有默认值")
	}
	if !fn.Params[2].IsArgs {
		t.Error("param[2] 应该是 *args")
	}
	if !fn.Params[3].IsKwargs {
		t.Error("param[3] 应该是 **kwargs")
	}
}

func TestTreeSitterPython_SelfStripped(t *testing.T) {
	source := []byte(`class Foo:
    def bar(self, a, b):
        return a + b
`)
	_, classes := parsePyWithTreeSitter(source)

	if len(classes) != 1 || len(classes[0].Methods) != 1 {
		t.Fatal("期望 1 个类 1 个方法")
	}
	method := classes[0].Methods[0]
	if len(method.Params) != 2 {
		t.Errorf("期望 2 个参数 (self 被剥离), got %d: %+v", len(method.Params), method.Params)
	}
	if method.Params[0].Name != "a" {
		t.Errorf("param[0].Name = %s, want a", method.Params[0].Name)
	}
}

func TestTreeSitterPython_DecoratedDefinitionsAndHelperFiltering(t *testing.T) {
	source := []byte(`@trace
def traced(value=1):
    return value

def setUp():
    return None

@service
class Worker:
    def __init__(self):
        pass

    def tearDown(self):
        pass

    @staticmethod
    def build(cls, value: int = 1, *args, **kwargs):
        return value

    def run(cls, mode):
        return mode
`)
	funcs, classes := parsePyWithTreeSitter(source)
	if len(funcs) != 1 || funcs[0].Name != "traced" {
		t.Fatalf("expected only decorated top-level traced function, got %+v", funcs)
	}
	if len(funcs[0].Params) != 1 || funcs[0].Params[0].Name != "value" || !funcs[0].Params[0].HasDefault {
		t.Fatalf("unexpected traced params: %+v", funcs[0].Params)
	}

	if len(classes) != 1 || classes[0].Name != "Worker" {
		t.Fatalf("expected decorated Worker class, got %+v", classes)
	}
	if len(classes[0].Methods) != 3 {
		t.Fatalf("__init__ should be retained for constructor args and tearDown should be skipped, got methods %+v", classes[0].Methods)
	}
	var init *pyFuncInfo
	var build *pyFuncInfo
	var run *pyFuncInfo
	for i := range classes[0].Methods {
		switch classes[0].Methods[i].Name {
		case "__init__":
			init = &classes[0].Methods[i]
		case "build":
			build = &classes[0].Methods[i]
		case "run":
			run = &classes[0].Methods[i]
		case "tearDown":
			t.Fatalf("helper method should be skipped: %+v", classes[0].Methods[i])
		}
	}
	if init == nil {
		t.Fatalf("__init__ should be retained in class metadata: %+v", classes[0].Methods)
	}
	if build == nil || !build.IsStatic {
		t.Fatalf("build should be parsed as static method: %+v", build)
	}
	if len(build.Params) != 4 || build.Params[0].Name != "cls" ||
		build.Params[1].Name != "value" || !build.Params[1].HasDefault ||
		!build.Params[2].IsArgs || !build.Params[3].IsKwargs {
		t.Fatalf("static method params should not strip cls and should keep varargs: %+v", build.Params)
	}
	if run == nil || len(run.Params) != 1 || run.Params[0].Name != "mode" {
		t.Fatalf("instance method should strip cls/self receiver, got %+v", run)
	}
}

func TestPytestArgsUseSemanticDefaults(t *testing.T) {
	params := []pyParamInfo{
		{Name: "url"},
		{Name: "count"},
		{Name: "enabled"},
		{Name: "items"},
		{Name: "options"},
		{Name: "name"},
	}

	got := pyArgList(params)
	for _, want := range []string{
		"'https://example.com'",
		"1",
		"True",
		"[]",
		"{}",
		"'test'",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("pyArgList() = %q, want value %q", got, want)
		}
	}
	if strings.Contains(got, "None") {
		t.Fatalf("pyArgList() = %q, should avoid None for recognized params", got)
	}
}

func TestPytestArgsKeepSemanticDefaultsForBoundaryCases(t *testing.T) {
	params := []pyParamInfo{
		{Name: "url"},
		{Name: "count"},
		{Name: "mode"},
	}

	got := pyArgListWithBoundary(params, pyBoundary{Param: "mode", Value: "None", Type: "None"})
	if got != "'https://example.com', 1, None" {
		t.Fatalf("pyArgListWithBoundary() = %q", got)
	}
}

func TestPytestRaiseArgsPreferInvalidBoundary(t *testing.T) {
	params := []pyParamInfo{
		{Name: "url"},
		{Name: "count"},
	}
	boundaries := []pyBoundary{{Param: "url", Value: "None", Type: "None"}}

	got := pyInvalidArgList(params, boundaries)
	if got != "None, 1" {
		t.Fatalf("pyInvalidArgList() = %q", got)
	}
}

func TestPytestBoundaryUsesRaisesForErrorPath(t *testing.T) {
	fn := pyFuncInfo{
		Name:   "divide",
		Params: []pyParamInfo{{Name: "a"}, {Name: "b"}},
		Analysis: pyFuncAnalysis{
			ReturnType: "float",
			HasReturn:  true,
			Raises:     true,
			Boundaries: []pyBoundary{{Param: "b", Value: "0", Type: "number"}},
		},
	}

	code := genPytestFuncTest(fn)
	if !strings.Contains(code, "def test_divide_b_boundary():") {
		t.Fatalf("missing boundary test:\n%s", code)
	}
	if !strings.Contains(code, "with pytest.raises(Exception):\n        divide(1, 0)") {
		t.Fatalf("boundary should assert raise, got:\n%s", code)
	}
	if strings.Contains(code, "result = divide(1, 0)") {
		t.Fatalf("boundary should not call raising input as normal result, got:\n%s", code)
	}
}

func TestPytestGeneratesExactAssertionForSimpleReturn(t *testing.T) {
	fn := pyFuncInfo{
		Name:   "add",
		Params: []pyParamInfo{{Name: "a"}, {Name: "b"}},
		Analysis: pyFuncAnalysis{
			ReturnType: "int",
			Returns:    []string{"a + b"},
			HasReturn:  true,
		},
	}

	code := genPytestFuncTest(fn)
	if !strings.Contains(code, "result = add(1, 2)") {
		t.Fatalf("expected semantic args, got:\n%s", code)
	}
	if !strings.Contains(code, "assert result == (1 + 2)") {
		t.Fatalf("expected exact assertion, got:\n%s", code)
	}
	if strings.Contains(code, "assert isinstance(result, int)") {
		t.Fatalf("exact assertion should replace broad type assertion, got:\n%s", code)
	}
}

func TestPytestExactAssertionUsesBoundaryValue(t *testing.T) {
	fn := pyFuncInfo{
		Name:   "normalize",
		Params: []pyParamInfo{{Name: "prefix"}, {Name: "text"}},
		Analysis: pyFuncAnalysis{
			ReturnType: "str",
			Returns:    []string{"prefix + text"},
			HasReturn:  true,
			Boundaries: []pyBoundary{{Param: "prefix", Value: "'>'", Type: "string"}},
		},
	}

	code := genPytestFuncTest(fn)
	if !strings.Contains(code, "assert result == ('test' + 'test')") {
		t.Fatalf("expected exact normal assertion, got:\n%s", code)
	}
	if !strings.Contains(code, "assert result == ('>' + 'test')") {
		t.Fatalf("expected exact boundary assertion, got:\n%s", code)
	}
}

func TestPytestExactAssertionUsesBranchReturnForBoundary(t *testing.T) {
	analysis := analyzePyBody(`if mode == "short":
    return prefix
return prefix + text`)
	fn := pyFuncInfo{
		Name:     "format_text",
		Params:   []pyParamInfo{{Name: "mode"}, {Name: "prefix"}, {Name: "text"}},
		Analysis: analysis,
	}

	code := genPytestFuncTest(fn)
	if !strings.Contains(code, "result = format_text('test', 'test', 'test')") {
		t.Fatalf("expected normal call, got:\n%s", code)
	}
	if !strings.Contains(code, "assert result == ('test' + 'test')") {
		t.Fatalf("expected default-path assertion, got:\n%s", code)
	}
	if !strings.Contains(code, "result = format_text(\"short\", 'test', 'test')") {
		t.Fatalf("expected boundary call, got:\n%s", code)
	}
	if !strings.Contains(code, "assert result == ('test')") {
		t.Fatalf("expected branch assertion, got:\n%s", code)
	}
}

func TestGeneratePytestTestsForCoverageTaskUsesTaskNameAndInputs(t *testing.T) {
	src := `def add(a, b):
    return a + b

def sub(a, b):
    return a - b
`
	srcPath := filepath.Join(t.TempDir(), "calc.py")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "pytest-1",
		Framework:       "pytest",
		Target:          "add",
		LineRange:       "2-2",
		GapType:         "return_path",
		TestName:        "test_add_zero_left_operand",
		SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入"},
		AssertionFocus:  []string{"断言未覆盖返回路径的具体结果"},
	}

	code, err := GeneratePytestTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GeneratePytestTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"def test_add_zero_left_operand():",
		"coverage task: pytest-1 | lines 2-2 | 断言未覆盖返回路径的具体结果 | 构造满足条件 `a == 0` 的输入",
		"result = add(0, 2)",
		"assert result == (0 + 2)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "test_sub") || strings.Contains(code, "sub(") {
		t.Fatalf("task-aware generation should only target add:\n%s", code)
	}
}

func TestGeneratePytestCoverageTaskUsesPackageImportForSrcLayout(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src", "click", "utils.py")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatal(err)
	}
	src := `def get_app_dir(app_name):
    return app_name
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:        "pytest-src-1",
		Framework: "pytest",
		Target:    "get_app_dir",
		LineRange: "2-2",
		GapType:   "return_path",
		TestName:  "test_get_app_dir_covers_gap",
	}

	code, err := GeneratePytestTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GeneratePytestTestsForCoverageTask() error = %v", err)
	}
	if !strings.Contains(code, "from click.utils import get_app_dir") {
		t.Fatalf("expected src-layout package import, got:\n%s", code)
	}
	if strings.Contains(code, "from utils import") {
		t.Fatalf("should not use basename import for src-layout package:\n%s", code)
	}
}

func TestGeneratePytestCoverageTaskUsesPackageImportForRootPackageLayout(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "starlette", "_utils.py")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "starlette", "__init__.py"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	src := `def get_route_path(scope):
    return scope["path"]
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:        "pytest-root-package-1",
		Framework: "pytest",
		Target:    "get_route_path",
		LineRange: "2-2",
		GapType:   "return_path",
		TestName:  "test_get_route_path_covers_gap",
	}

	code, err := GeneratePytestTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GeneratePytestTestsForCoverageTask() error = %v", err)
	}
	if !strings.Contains(code, "from starlette._utils import get_route_path") {
		t.Fatalf("expected root-package import, got:\n%s", code)
	}
	if strings.Contains(code, "from _utils import") {
		t.Fatalf("should not use basename import for root package layout:\n%s", code)
	}
}

func TestGeneratePytestCoverageTaskSanitizesMultilineComment(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "service.py")
	src := `def status(value):
    if value == "active":
        return "ok"
    return "idle"
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "pytest-comment-1",
		Framework:       "pytest",
		Target:          "status",
		LineRange:       "2-3",
		GapType:         "branch",
		TestName:        "test_status_covers_gap",
		MissingBranches: []string{"未覆盖 if 分支: value == \"active\":\n        return \"ok\""},
		SuggestedInputs: []string{"构造满足条件 `value == \"active\":\n        return \"ok\"` 的输入"},
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
	}

	code, err := GeneratePytestTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GeneratePytestTestsForCoverageTask() error = %v", err)
	}
	for _, line := range strings.Split(code, "\n") {
		if strings.Contains(line, "coverage task:") && strings.Contains(line, "\n") {
			t.Fatalf("coverage comment should stay on one line:\n%s", code)
		}
	}
	if strings.Contains(code, "\n        return \"ok\"") {
		t.Fatalf("coverage comment leaked multiline branch body:\n%s", code)
	}
}

func TestFilterPyTargetsForCoverageTaskBranches(t *testing.T) {
	funcs := []pyFuncInfo{{Name: "add"}, {Name: "sub"}}
	classes := []pyClassInfo{
		{
			Name: "Calculator",
			Methods: []pyFuncInfo{
				{Name: "add", ClassName: "Calculator", IsMethod: true},
				{Name: "divide", ClassName: "Calculator", IsMethod: true},
			},
		},
	}

	gotFuncs, gotClasses := filterPyTargetsForCoverageTask(funcs, classes, &types.CoverageTestTask{})
	if len(gotFuncs) != 2 || len(gotClasses) != 1 {
		t.Fatalf("empty target should keep all targets: funcs=%+v classes=%+v", gotFuncs, gotClasses)
	}

	gotFuncs, gotClasses = filterPyTargetsForCoverageTask(funcs, classes, &types.CoverageTestTask{Target: "add"})
	if len(gotFuncs) != 1 || gotFuncs[0].Name != "add" || len(gotClasses) != 1 ||
		len(gotClasses[0].Methods) != 1 || gotClasses[0].Methods[0].Name != "add" {
		t.Fatalf("function target filtered incorrectly: funcs=%+v classes=%+v", gotFuncs, gotClasses)
	}

	gotFuncs, gotClasses = filterPyTargetsForCoverageTask(funcs, classes, &types.CoverageTestTask{Target: "Calculator"})
	if len(gotFuncs) != 0 || len(gotClasses) != 1 || len(gotClasses[0].Methods) != 2 {
		t.Fatalf("class target should keep whole class: funcs=%+v classes=%+v", gotFuncs, gotClasses)
	}

	gotFuncs, gotClasses = filterPyTargetsForCoverageTask(funcs, classes, &types.CoverageTestTask{Target: "Calculator.divide"})
	if len(gotFuncs) != 0 || len(gotClasses) != 1 || len(gotClasses[0].Methods) != 1 || gotClasses[0].Methods[0].Name != "divide" {
		t.Fatalf("method target filtered incorrectly: funcs=%+v classes=%+v", gotFuncs, gotClasses)
	}

	gotFuncs, gotClasses = filterPyTargetsForCoverageTask(funcs, classes, &types.CoverageTestTask{Target: "missing"})
	if len(gotFuncs) != 2 || len(gotClasses) != 1 {
		t.Fatalf("missing target should fall back to all targets: funcs=%+v classes=%+v", gotFuncs, gotClasses)
	}
}

func TestPytestClassCoverageTaskCoversNormalAndErrorMethods(t *testing.T) {
	task := types.CoverageTestTask{
		ID:              "pytest-class-1",
		Target:          "Widget.load",
		LineRange:       "10-12",
		GapType:         "branch",
		TestName:        "test_widget_load_gap",
		SuggestedInputs: []string{"构造满足条件 `mode == 'short'` 的输入"},
		AssertionFocus:  []string{"断言 class 方法分支"},
	}
	cls := pyClassInfo{
		Name: "Widget",
		Methods: []pyFuncInfo{
			{Name: "__init__"},
			{
				Name:   "load",
				Params: []pyParamInfo{{Name: "mode"}, {Name: "count"}},
				Analysis: pyFuncAnalysis{
					ReturnType: "int",
					Returns:    []string{"count + 1"},
					HasReturn:  true,
					Boundaries: []pyBoundary{{Param: "mode", Value: "'short'", Type: "string", ReturnExpr: "count"}},
				},
			},
			{
				Name:     "save",
				IsAsync:  true,
				IsStatic: true,
				Params:   []pyParamInfo{{Name: "payload"}},
				Analysis: pyFuncAnalysis{
					Raises: true,
				},
			},
		},
	}

	code := genPytestClassTestForCoverageTask(cls, &task)
	for _, want := range []string{
		"class TestWidget:",
		"def test_widget_load_gap(self):",
		"coverage task: pytest-class-1 | lines 10-12 | 断言 class 方法分支 | 构造满足条件 `mode == 'short'` 的输入",
		"instance = Widget()",
		"result = instance.load('short', 1)",
		"assert result == (1)",
		"with pytest.raises(Exception):\n            asyncio.run(Widget.save({}))",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in class task output:\n%s", want, code)
		}
	}
	if strings.Contains(code, "__init__") {
		t.Fatalf("__init__ should be skipped in class task output:\n%s", code)
	}
}

func TestPytestClassCoverageTaskKeepsEmptyClassExecutable(t *testing.T) {
	task := types.CoverageTestTask{
		ID:        "pytest-pydantic-dto",
		Target:    "ReleaseNotesUpdate",
		LineRange: "entire file",
		TestName:  "test_release_notes_update_covers_gap",
	}
	cls := pyClassInfo{Name: "ReleaseNotesUpdate"}

	code := genPytestClassTestForCoverageTask(cls, &task)
	assertPyGenerated(t, code, []string{
		"class TestReleaseNotesUpdate:",
		"def test_release_notes_update_covers_gap(self):",
		"coverage task: pytest-pydantic-dto | lines entire file",
		"assert ReleaseNotesUpdate is not None",
	}, []string{
		"class TestReleaseNotesUpdate:\n}",
		"class TestReleaseNotesUpdate:\n\n",
	})
}

func TestPytestCoverageTaskUsesConstructorAndErrorPathInputs(t *testing.T) {
	optionTask := types.CoverageTestTask{
		ID:              "pytest-option-1",
		Target:          "_Option.process",
		GapType:         "error_path",
		LineRange:       "181-181",
		TestName:        "test_option_process_covers_gap",
		SuggestedInputs: []string{"设置 value 覆盖未执行分支", "设置 state 覆盖未执行分支"},
	}
	optionClass := pyClassInfo{
		Name: "_Option",
		Methods: []pyFuncInfo{
			{Name: "__init__", Params: []pyParamInfo{{Name: "obj"}, {Name: "opts"}, {Name: "dest"}}},
			{Name: "process", Params: []pyParamInfo{{Name: "value"}, {Name: "state"}}, Analysis: pyFuncAnalysis{Raises: true}},
		},
	}
	code := genPytestClassTestForCoverageTask(optionClass, &optionTask)
	assertPyGenerated(t, code, []string{
		"instance = _Option(None, ['--test'], None, action='unknown')",
		"with pytest.raises(Exception):",
		"instance.process('test', None)",
	}, []string{
		"instance = _Option()",
	})

	flushTask := types.CoverageTestTask{
		ID:              "pytest-flush-1",
		Target:          "PacifyFlushWrapper.flush",
		GapType:         "branch",
		LineRange:       "535-536",
		TestName:        "test_pacifyflushwrapper_flush_covers_gap",
		MissingBranches: []string{"未覆盖 if 分支: e.errno != errno.EPIPE"},
	}
	flushClass := pyClassInfo{
		Name: "PacifyFlushWrapper",
		Methods: []pyFuncInfo{
			{Name: "__init__", Params: []pyParamInfo{{Name: "wrapped"}}},
			{Name: "flush", Analysis: pyFuncAnalysis{Raises: true}},
		},
	}
	code = genPytestClassTestForCoverageTask(flushClass, &flushTask)
	assertPyGenerated(t, code, []string{
		"instance = PacifyFlushWrapper(type('BrokenFlush'",
		"with pytest.raises(Exception):",
		"instance.flush()",
	}, []string{
		"PacifyFlushWrapper()",
	})

	unpackTask := types.CoverageTestTask{
		ID:        "pytest-unpack-1",
		Target:    "_unpack_args",
		GapType:   "error_path",
		LineRange: "96-96",
		TestName:  "test_unpack_args_covers_gap",
	}
	unpackFunc := pyFuncInfo{
		Name:   "_unpack_args",
		Params: []pyParamInfo{{Name: "args"}, {Name: "nargs_spec"}},
		Analysis: pyFuncAnalysis{
			Raises: true,
		},
	}
	code = genPytestFuncTestForCoverageTask(unpackFunc, &unpackTask)
	assertPyGenerated(t, code, []string{
		"_unpack_args(['a'], [-1, -1])",
	}, []string{
		"_unpack_args([], [])",
	})
}

func TestPytestCoverageTaskUsesRealProjectFallbackInputs(t *testing.T) {
	safecallTask := types.CoverageTestTask{
		ID:        "pytest-safecall-1",
		Target:    "safecall",
		GapType:   "error_path",
		LineRange: "42-44",
		TestName:  "test_safecall_covers_gap",
	}
	code := genPytestFuncTestForCoverageTask(pyFuncInfo{
		Name:   "safecall",
		Params: []pyParamInfo{{Name: "func"}},
	}, &safecallTask)
	assertPyGenerated(t, code, []string{
		"def boom():",
		"raise RuntimeError('boom')",
		"result = safecall(boom)()",
		"assert result is None",
	}, []string{
		"with pytest.raises(Exception):",
		"safecall(None)",
	})

	makeStrTask := types.CoverageTestTask{
		ID:        "pytest-make-str-1",
		Target:    "make_str",
		GapType:   "error_path",
		LineRange: "52-55",
		TestName:  "test_make_str_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(pyFuncInfo{
		Name:   "make_str",
		Params: []pyParamInfo{{Name: "value"}},
	}, &makeStrTask)
	assertPyGenerated(t, code, []string{
		"_sys.getfilesystemencoding = lambda: 'ascii'",
		"result = make_str(b'\\xff')",
		"_sys.getfilesystemencoding = _original",
		"assert isinstance(result, str)",
	}, []string{
		"with pytest.raises(Exception):",
		"make_str('test')",
	})

	stderrTask := types.CoverageTestTask{
		ID:              "pytest-stderr-1",
		Target:          "get_binary_stderr",
		GapType:         "branch",
		LineRange:       "334-337",
		TestName:        "test_get_binary_stderr_covers_gap",
		MissingBranches: []string{"未覆盖 if 分支: writer is None: raise RuntimeError"},
	}
	code = genPytestFuncTestForCoverageTask(pyFuncInfo{
		Name:     "get_binary_stderr",
		Analysis: pyFuncAnalysis{Raises: true},
	}, &stderrTask)
	assertPyGenerated(t, code, []string{
		"pytest.skip(\"manual_review_environment: get_binary_stderr depends on process std stream binary-wrapper state",
	}, []string{
		"with pytest.raises(Exception):",
		"get_binary_stderr()",
	})
}

func TestPytestClassCoverageTaskUsesRealProjectInstances(t *testing.T) {
	readableTask := types.CoverageTestTask{
		ID:              "pytest-readable-1",
		Target:          "_FixupStream.readable",
		GapType:         "branch",
		LineRange:       "119-126",
		TestName:        "test_fixupstream_readable_covers_gap",
		MissingBranches: []string{"未覆盖 if 分支: try: self._stream.read(0) except Exception: return False"},
	}
	readableClass := pyClassInfo{
		Name: "_FixupStream",
		Methods: []pyFuncInfo{
			{Name: "__init__", Params: []pyParamInfo{{Name: "stream"}, {Name: "force_readable", HasDefault: true}, {Name: "force_writable", HasDefault: true}}},
			{Name: "readable", Analysis: pyFuncAnalysis{ReturnType: "bool", Returns: []string{"False", "True"}, HasReturn: true}},
		},
	}
	code := genPytestClassTestForCoverageTask(readableClass, &readableTask)
	assertPyGenerated(t, code, []string{
		"instance = _FixupStream(type('Unreadable'",
		"result = instance.readable()",
		"assert result is False",
	}, []string{
		"__import__('io').BytesIO",
	})

	formatTask := types.CoverageTestTask{
		ID:              "pytest-progress-format-1",
		Target:          "ProgressBar.format_bar",
		GapType:         "branch",
		LineRange:       "218-220",
		TestName:        "test_progressbar_format_bar_covers_gap",
		MissingBranches: []string{"未覆盖 if 分支: self.time_per_iteration != 0: chars["},
	}
	progressClass := pyClassInfo{
		Name: "ProgressBar",
		Methods: []pyFuncInfo{
			{Name: "__init__", Params: []pyParamInfo{{Name: "iterable"}, {Name: "length", HasDefault: true}, {Name: "width", HasDefault: true}}},
			{Name: "format_bar", Analysis: pyFuncAnalysis{ReturnType: "str", Returns: []string{"bar"}, HasReturn: true}},
		},
	}
	code = genPytestClassTestForCoverageTask(progressClass, &formatTask)
	assertPyGenerated(t, code, []string{
		"instance = ProgressBar(type('UnknownLength'",
		"instance.avg = [1.0]",
		"instance.pos = 1",
		"result = instance.format_bar()",
		"assert isinstance(result, str)",
	}, []string{
		"ProgressBar([], None",
	})

	renderTask := types.CoverageTestTask{
		ID:              "pytest-progress-render-1",
		Target:          "ProgressBar.render_progress",
		GapType:         "branch",
		LineRange:       "272-280",
		TestName:        "test_progressbar_render_progress_covers_gap",
		MissingBranches: []string{"未覆盖 if 分支: new_width < old_width and self.max_width is not None"},
	}
	progressClass.Methods[1] = pyFuncInfo{Name: "render_progress", Analysis: pyFuncAnalysis{HasReturn: false}}
	code = genPytestClassTestForCoverageTask(progressClass, &renderTask)
	assertPyGenerated(t, code, []string{
		"instance.autowidth = True",
		"_shutil.get_terminal_size = lambda: _os.terminal_size((5, 24))",
		"finally:",
		"_shutil.get_terminal_size = _original_get_terminal_size",
		"assert result is None",
	}, []string{
		"assert result is not None",
	})
}

func TestPytestCoverageTaskUsesCodexGoalStateInputs(t *testing.T) {
	goalTask := types.CoverageTestTask{
		ID:              "pytest-goal-1",
		Target:          "_GoalOperationState.active_turn",
		GapType:         "branch",
		LineRange:       "156-166",
		TestName:        "test_goaloperationstate_active_turn_covers_gap",
		MissingBranches: []string{"未覆盖 if 分支: self.current_turn_id is not None and self.current_turn_id != after: return self.current_turn_id"},
	}
	goalClass := pyClassInfo{
		Name: "_GoalOperationState",
		Methods: []pyFuncInfo{
			{Name: "active_turn", Params: []pyParamInfo{{Name: "after", HasDefault: true}}, Analysis: pyFuncAnalysis{Raises: true, HasReturn: true, ReturnType: "unknown", Returns: []string{"self.current_turn_id"}}},
		},
	}
	code := genPytestClassTestForCoverageTask(goalClass, &goalTask)
	assertPyGenerated(t, code, []string{
		"instance = _GoalOperationState('thread-1')",
		"instance.current_turn_id = 'turn-1'",
		"result = instance.active_turn()",
		"assert result == 'turn-1'",
	}, []string{
		"_GoalOperationState()",
		"with pytest.raises(Exception):",
		"instance.active_turn(None)",
	})

	beginTask := types.CoverageTestTask{
		ID:        "pytest-goal-2",
		Target:    "_GoalOperationState.begin_interrupt",
		GapType:   "branch",
		LineRange: "131-135",
		TestName:  "test_goaloperationstate_begin_interrupt_covers_gap",
	}
	goalClass.Methods[0] = pyFuncInfo{Name: "begin_interrupt", Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "bool", Returns: []string{"False", "True"}}}
	code = genPytestClassTestForCoverageTask(goalClass, &beginTask)
	assertPyGenerated(t, code, []string{
		"result = instance.begin_interrupt()",
		"assert result is True",
		"assert instance.interrupt_requested is True",
	}, []string{
		"_GoalOperationState()",
	})

	observeTask := types.CoverageTestTask{
		ID:        "pytest-goal-3",
		Target:    "_GoalOperationState.observe",
		GapType:   "branch",
		LineRange: "64-68",
		TestName:  "test_goaloperationstate_observe_covers_gap",
	}
	goalClass.Methods[0] = pyFuncInfo{Name: "observe", Params: []pyParamInfo{{Name: "notification"}}, Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "bool", Returns: []string{"True"}}}
	code = genPytestClassTestForCoverageTask(goalClass, &observeTask)
	assertPyGenerated(t, code, []string{
		"models.Notification('turn/started'",
		"v2.TurnStartedNotification",
		"result = instance.observe(notification)",
		"assert instance.logical_turn_id == 'physical-turn'",
	}, []string{
		"instance.observe(None)",
		"_GoalOperationState()",
	})
}

func TestPytestCoverageTaskUsesCodexGoalNotificationInputs(t *testing.T) {
	notificationTask := types.CoverageTestTask{
		ID:        "pytest-goal-notification-1",
		Target:    "_logical_notification",
		GapType:   "branch",
		LineRange: "208-212",
		TestName:  "test_logical_notification_covers_gap",
	}
	code := genPytestFuncTestForCoverageTask(pyFuncInfo{
		Name:   "_logical_notification",
		Params: []pyParamInfo{{Name: "notification"}, {Name: "logical_turn_id"}},
		Analysis: pyFuncAnalysis{
			HasReturn:  true,
			ReturnType: "unknown",
			Returns:    []string{"notification"},
		},
	}, &notificationTask)
	assertPyGenerated(t, code, []string{
		"v2.AgentMessageDeltaNotification",
		"models.Notification('agent_message_delta', payload)",
		"result = _logical_notification(notification, 'logical-turn')",
		"assert result.payload.turn_id == 'logical-turn'",
	}, []string{
		"_logical_notification(None, 1)",
	})

	completionTask := types.CoverageTestTask{
		ID:        "pytest-goal-completion-1",
		Target:    "_logical_completion",
		GapType:   "branch",
		LineRange: "243-245",
		TestName:  "test_logical_completion_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(pyFuncInfo{
		Name: "_logical_completion",
		Params: []pyParamInfo{
			{Name: "completed"},
			{Name: "logical_turn_id"},
			{Name: "started"},
			{Name: "interrupted"},
		},
		Analysis: pyFuncAnalysis{
			HasReturn:  true,
			ReturnType: "unknown",
			Returns:    []string{"completed.model_copy(update={\"turn\": final_turn.model_copy(update=updates)})"},
		},
	}, &completionTask)
	assertPyGenerated(t, code, []string{
		"completed = v2.TurnCompletedNotification(threadId='thread-1', turn=turn)",
		"_logical_completion(completed, logical_turn_id='logical-turn', started=started, interrupted=True)",
		"assert result.turn.status == v2.TurnStatus.interrupted",
	}, []string{
		"_logical_completion(None, 1, None, None)",
	})

	wireTask := types.CoverageTestTask{
		ID:        "pytest-wire-input-1",
		Target:    "_to_wire_input",
		GapType:   "branch",
		LineRange: "65-67",
		TestName:  "test_to_wire_input_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(pyFuncInfo{
		Name:   "_to_wire_input",
		Params: []pyParamInfo{{Name: "input"}},
		Analysis: pyFuncAnalysis{
			HasReturn:  true,
			ReturnType: "list",
			Returns:    []string{"[_to_wire_item(i) for i in input]", "[_to_wire_item(input)]"},
		},
	}, &wireTask)
	assertPyGenerated(t, code, []string{
		"inputs = __import__('openai_codex._inputs', fromlist=['TextInput'])",
		"result = _to_wire_input([inputs.TextInput('hello')])",
		"assert result == [{'type': 'text', 'text': 'hello'}]",
	}, []string{
		"_to_wire_input(None)",
	})
}

func TestPytestCoverageTaskUsesCodexGoalStreamInputs(t *testing.T) {
	processTask := types.CoverageTestTask{
		ID:        "pytest-goal-cursor-1",
		Target:    "_GoalStreamCursor.process",
		GapType:   "branch",
		LineRange: "265-271",
		TestName:  "test_goalstreamcursor_process_covers_gap",
	}
	cursorClass := pyClassInfo{
		Name: "_GoalStreamCursor",
		Methods: []pyFuncInfo{
			{Name: "process", Params: []pyParamInfo{{Name: "notification"}}, Analysis: pyFuncAnalysis{Raises: true, HasReturn: true}},
		},
	}
	code := genPytestClassTestForCoverageTask(cursorClass, &processTask)
	assertPyGenerated(t, code, []string{
		"state = goal._GoalOperationState('thread-1')",
		"state.logical_turn_id = 'logical-turn'",
		"instance = _GoalStreamCursor(state)",
		"v2.TurnStartedNotification",
		"events, completed = instance.process(notification)",
		"assert events[0].payload.turn.id == 'logical-turn'",
	}, []string{
		"_GoalStreamCursor()",
		"instance.process(None)",
	})

	completionTask := types.CoverageTestTask{
		ID:        "pytest-goal-cursor-2",
		Target:    "_GoalStreamCursor._completion",
		GapType:   "branch",
		LineRange: "333-336",
		TestName:  "test_goalstreamcursor_completion_covers_gap",
	}
	cursorClass.Methods[0] = pyFuncInfo{Name: "_completion", Params: []pyParamInfo{{Name: "method"}, {Name: "payload"}}, Analysis: pyFuncAnalysis{Raises: true, HasReturn: true}}
	code = genPytestClassTestForCoverageTask(cursorClass, &completionTask)
	assertPyGenerated(t, code, []string{
		"instance = _GoalStreamCursor(state)",
		"with pytest.raises(RuntimeError):",
		"instance._completion('turn/completed', completed)",
	}, []string{
		"_GoalStreamCursor()",
		"instance._completion(None, {})",
	})

	finishTask := types.CoverageTestTask{
		ID:        "pytest-goal-stream-1",
		Target:    "_GoalNotificationStream._finish",
		GapType:   "branch",
		LineRange: "388-393",
		TestName:  "test_goalnotificationstream_finish_covers_gap",
	}
	streamClass := pyClassInfo{
		Name: "_GoalNotificationStream",
		Methods: []pyFuncInfo{
			{Name: "_finish", Analysis: pyFuncAnalysis{HasReturn: false}},
		},
	}
	code = genPytestClassTestForCoverageTask(streamClass, &finishTask)
	assertPyGenerated(t, code, []string{
		"instance = _GoalNotificationStream(state, lambda: None, lambda: calls.append('unregister'), lambda: calls.append('cancel'))",
		"result = instance._finish()",
		"assert instance._closed is True",
		"assert calls == ['unregister']",
	}, []string{
		"_GoalNotificationStream()",
		"assert result is not None",
	})
}

func TestPytestCoverageTaskUsesStarletteServiceInputs(t *testing.T) {
	routeTask := types.CoverageTestTask{
		ID:        "pytest-starlette-route-1",
		Target:    "get_route_path",
		GapType:   "return_path",
		LineRange: "111-111",
		TestName:  "test_get_route_path_covers_gap",
	}
	code := genPytestFuncTestForCoverageTask(pyFuncInfo{
		Name:   "get_route_path",
		Params: []pyParamInfo{{Name: "scope"}},
		Analysis: pyFuncAnalysis{
			HasReturn:  true,
			ReturnType: "str",
			Returns:    []string{"path", "path[len(root_path) :]"},
		},
	}, &routeTask)
	assertPyGenerated(t, code, []string{
		"scope = {'type': 'http', 'path': '/app/home', 'root_path': '/app'}",
		"result = get_route_path(scope)",
		"assert result == '/home'",
	}, []string{
		"get_route_path(None)",
	})

	scopeTask := types.CoverageTestTask{
		ID:        "pytest-starlette-auth-1",
		Target:    "has_required_scope",
		GapType:   "branch",
		LineRange: "18-22",
		TestName:  "test_has_required_scope_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(pyFuncInfo{
		Name:   "has_required_scope",
		Params: []pyParamInfo{{Name: "conn"}, {Name: "scopes"}},
		Analysis: pyFuncAnalysis{
			HasReturn:  true,
			ReturnType: "bool",
			Returns:    []string{"False", "True"},
		},
	}, &scopeTask)
	assertPyGenerated(t, code, []string{
		"auth = type('Auth', (), {'scopes': ['authenticated']})()",
		"conn = type('Conn', (), {'auth': auth})()",
		"assert has_required_scope(conn, ['missing']) is False",
		"result = has_required_scope(conn, ['authenticated'])",
		"assert result is True",
	}, []string{
		"has_required_scope(None, None)",
	})
}

func TestPytestCoverageTaskUsesStarletteConfigInputs(t *testing.T) {
	readTask := types.CoverageTestTask{
		ID:        "pytest-starlette-config-1",
		Target:    "Config._read_file",
		GapType:   "branch",
		LineRange: "111-121",
		TestName:  "test_config_read_file_covers_gap",
	}
	configClass := pyClassInfo{
		Name: "Config",
		Methods: []pyFuncInfo{
			{Name: "_read_file", Params: []pyParamInfo{{Name: "file_name"}, {Name: "encoding"}}, Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "dict", Returns: []string{"file_values"}}},
		},
	}
	code := genPytestClassTestForCoverageTask(configClass, &readTask)
	assertPyGenerated(t, code, []string{
		"handle = tempfile.NamedTemporaryFile('w', delete=False, encoding='utf-8')",
		"handle.write(\"# ignored\\nAPI_KEY='secret'\\nDEBUG=true\\n\")",
		"result = instance._read_file(handle.name, 'utf-8')",
		"assert result == {'API_KEY': 'secret', 'DEBUG': 'true'}",
		"os.unlink(handle.name)",
	}, []string{
		"Config(None, None, 'test', None)",
		"instance._read_file('test', None)",
	})

	castTask := types.CoverageTestTask{
		ID:        "pytest-starlette-config-2",
		Target:    "Config._perform_cast",
		GapType:   "branch",
		LineRange: "129-140",
		TestName:  "test_config_perform_cast_covers_gap",
	}
	configClass.Methods[0] = pyFuncInfo{Name: "_perform_cast", Params: []pyParamInfo{{Name: "key"}, {Name: "value"}, {Name: "cast", HasDefault: true}}, Analysis: pyFuncAnalysis{Raises: true, HasReturn: true, ReturnType: "unknown"}}
	code = genPytestClassTestForCoverageTask(configClass, &castTask)
	assertPyGenerated(t, code, []string{
		"assert instance._perform_cast('DEBUG', 'true', bool) is True",
		"assert instance._perform_cast('COUNT', '3', int) == 3",
		"with pytest.raises(ValueError):",
		"instance._perform_cast('DEBUG', 'maybe', bool)",
	}, []string{
		"instance._perform_cast('test', 'test', None)",
	})
}

func TestPytestCoverageTaskUsesStarletteMultiDictInputs(t *testing.T) {
	multiDictClass := pyClassInfo{
		Name: "MultiDict",
		Methods: []pyFuncInfo{
			{Name: "pop", Params: []pyParamInfo{{Name: "key"}, {Name: "default", HasDefault: true}}, Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "unknown"}},
		},
	}
	popTask := types.CoverageTestTask{
		ID:        "pytest-starlette-multidict-1",
		Target:    "MultiDict.pop",
		GapType:   "branch",
		LineRange: "331-332",
		TestName:  "test_multidict_pop_covers_gap",
	}
	code := genPytestClassTestForCoverageTask(multiDictClass, &popTask)
	assertPyGenerated(t, code, []string{
		"instance = MultiDict([('a', '123'), ('a', '456'), ('b', '789')])",
		"result = instance.pop('a')",
		"assert result == '456'",
		"assert instance.multi_items() == [('b', '789')]",
	}, []string{
		"MultiDict()",
		"instance.pop('test', None)",
	})

	popitemTask := types.CoverageTestTask{
		ID:        "pytest-starlette-multidict-2",
		Target:    "MultiDict.popitem",
		GapType:   "branch",
		LineRange: "335-337",
		TestName:  "test_multidict_popitem_covers_gap",
	}
	multiDictClass.Methods[0] = pyFuncInfo{Name: "popitem", Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "tuple"}}
	code = genPytestClassTestForCoverageTask(multiDictClass, &popitemTask)
	assertPyGenerated(t, code, []string{
		"result = instance.popitem()",
		"assert result == ('b', '789')",
		"assert instance.multi_items() == [('a', '123'), ('a', '456')]",
	}, []string{
		"MultiDict()",
	})

	setdefaultTask := types.CoverageTestTask{
		ID:        "pytest-starlette-multidict-3",
		Target:    "MultiDict.setdefault",
		GapType:   "branch",
		LineRange: "349-351",
		TestName:  "test_multidict_setdefault_covers_gap",
	}
	multiDictClass.Methods[0] = pyFuncInfo{Name: "setdefault", Params: []pyParamInfo{{Name: "key"}, {Name: "default", HasDefault: true}}, Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "unknown"}}
	code = genPytestClassTestForCoverageTask(multiDictClass, &setdefaultTask)
	assertPyGenerated(t, code, []string{
		"assert instance.setdefault('a', '456') == '123'",
		"result = instance.setdefault('b', '456')",
		"assert result == '456'",
		"assert instance.multi_items() == [('a', '123'), ('b', '456')]",
	}, []string{
		"MultiDict()",
		"instance.setdefault('test', None)",
	})
}

func TestPytestCoverageTaskUsesStarletteUploadFileInputs(t *testing.T) {
	uploadClass := pyClassInfo{
		Name: "UploadFile",
		Methods: []pyFuncInfo{
			{Name: "_will_roll", Params: []pyParamInfo{{Name: "size_to_add"}}, Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "bool"}},
		},
	}
	willRollDiskTask := types.CoverageTestTask{
		ID:        "pytest-starlette-upload-1",
		Target:    "UploadFile._will_roll",
		GapType:   "branch",
		LineRange: "444-445",
		TestName:  "test_upload_file_will_roll_disk_covers_gap",
	}
	code := genPytestClassTestForCoverageTask(uploadClass, &willRollDiskTask)
	assertPyGenerated(t, code, []string{
		"stream = tempfile.SpooledTemporaryFile(max_size=1)",
		"stream.rollover()",
		"instance = UploadFile(file=stream, filename='file', size=2)",
		"result = instance._will_roll(1)",
		"assert result is True",
	}, []string{
		"UploadFile(None, 1, 'test', None)",
		"instance._will_roll(None)",
	})

	willRollSizeTask := types.CoverageTestTask{
		ID:        "pytest-starlette-upload-2",
		Target:    "UploadFile._will_roll",
		GapType:   "branch",
		LineRange: "448-449",
		TestName:  "test_upload_file_will_roll_size_covers_gap",
	}
	code = genPytestClassTestForCoverageTask(uploadClass, &willRollSizeTask)
	assertPyGenerated(t, code, []string{
		"stream = tempfile.SpooledTemporaryFile(max_size=4)",
		"stream.write(b'abc')",
		"result = instance._will_roll(2)",
		"assert result is True",
	}, []string{
		"stream.rollover()",
		"UploadFile(None, 1, 'test', None)",
	})

	writeTask := types.CoverageTestTask{
		ID:        "pytest-starlette-upload-3",
		Target:    "UploadFile.write",
		GapType:   "statement",
		LineRange: "452-454",
		TestName:  "test_upload_file_write_size_covers_gap",
	}
	uploadClass.Methods[0] = pyFuncInfo{Name: "write", Params: []pyParamInfo{{Name: "data"}}, IsAsync: true, Analysis: pyFuncAnalysis{HasReturn: false}}
	code = genPytestClassTestForCoverageTask(uploadClass, &writeTask)
	assertPyGenerated(t, code, []string{
		"instance = UploadFile(file=stream, filename='file', size=0)",
		"result = asyncio.run(instance.write(b'hi'))",
		"assert instance.size == 2",
		"assert stream.read() == b'hi'",
	}, []string{
		"instance.write({})",
		"UploadFile(None, 1, 'test', None)",
	})

	writeRollTask := types.CoverageTestTask{
		ID:        "pytest-starlette-upload-4",
		Target:    "UploadFile.write",
		GapType:   "branch",
		LineRange: "456-457",
		TestName:  "test_upload_file_write_roll_covers_gap",
	}
	code = genPytestClassTestForCoverageTask(uploadClass, &writeRollTask)
	assertPyGenerated(t, code, []string{
		"stream = tempfile.SpooledTemporaryFile(max_size=1)",
		"result = asyncio.run(instance.write(b'hi'))",
		"assert instance._in_memory is False",
	}, []string{
		"instance.write({})",
	})

	readTask := types.CoverageTestTask{
		ID:        "pytest-starlette-upload-5",
		Target:    "UploadFile.read",
		GapType:   "branch",
		LineRange: "462-464",
		TestName:  "test_upload_file_read_covers_gap",
	}
	uploadClass.Methods[0] = pyFuncInfo{Name: "read", Params: []pyParamInfo{{Name: "size", HasDefault: true}}, IsAsync: true, Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "bytes"}}
	code = genPytestClassTestForCoverageTask(uploadClass, &readTask)
	assertPyGenerated(t, code, []string{
		"stream.write(b'hello')",
		"result = asyncio.run(instance.read(2))",
		"assert result == b'he'",
	}, []string{
		"instance.read(None)",
	})

	seekTask := types.CoverageTestTask{
		ID:        "pytest-starlette-upload-6",
		Target:    "UploadFile.seek",
		GapType:   "branch",
		LineRange: "467-468",
		TestName:  "test_upload_file_seek_covers_gap",
	}
	uploadClass.Methods[0] = pyFuncInfo{Name: "seek", Params: []pyParamInfo{{Name: "offset"}}, IsAsync: true, Analysis: pyFuncAnalysis{HasReturn: false}}
	code = genPytestClassTestForCoverageTask(uploadClass, &seekTask)
	assertPyGenerated(t, code, []string{
		"result = asyncio.run(instance.seek(1))",
		"assert stream.tell() == 1",
	}, []string{
		"instance.seek(None)",
	})

	closeTask := types.CoverageTestTask{
		ID:        "pytest-starlette-upload-7",
		Target:    "UploadFile.close",
		GapType:   "branch",
		LineRange: "473-474",
		TestName:  "test_upload_file_close_covers_gap",
	}
	uploadClass.Methods[0] = pyFuncInfo{Name: "close", IsAsync: true, Analysis: pyFuncAnalysis{HasReturn: false}}
	code = genPytestClassTestForCoverageTask(uploadClass, &closeTask)
	assertPyGenerated(t, code, []string{
		"result = asyncio.run(instance.close())",
		"assert stream.closed is True",
	}, []string{
		"UploadFile(None, 1, 'test', None)",
	})
}

func TestPytestCoverageTaskUsesStarletteMutableHeadersInputs(t *testing.T) {
	headersClass := pyClassInfo{
		Name: "MutableHeaders",
		Methods: []pyFuncInfo{
			{Name: "add_vary_header", Params: []pyParamInfo{{Name: "vary"}}, Analysis: pyFuncAnalysis{HasReturn: false}},
		},
	}
	task := types.CoverageTestTask{
		ID:        "pytest-starlette-headers-1",
		Target:    "MutableHeaders.add_vary_header",
		GapType:   "branch",
		LineRange: "658-661",
		TestName:  "test_mutable_headers_add_vary_header_covers_gap",
	}
	code := genPytestClassTestForCoverageTask(headersClass, &task)
	assertPyGenerated(t, code, []string{
		"instance = MutableHeaders({'vary': 'Accept-Encoding'})",
		"result = instance.add_vary_header('Origin')",
		"assert result is None",
		"assert instance['vary'] == 'Accept-Encoding, Origin'",
	}, []string{
		"MutableHeaders()",
		"add_vary_header(None)",
	})
}

func TestPytestCoverageTaskUsesStarletteEndpointInputs(t *testing.T) {
	httpClass := pyClassInfo{
		Name: "HTTPEndpoint",
		Methods: []pyFuncInfo{
			{Name: "dispatch", IsAsync: true, Analysis: pyFuncAnalysis{HasReturn: false}},
		},
	}
	dispatchTask := types.CoverageTestTask{
		ID:        "pytest-starlette-endpoint-1",
		Target:    "HTTPEndpoint.dispatch",
		GapType:   "branch",
		LineRange: "32-43",
		TestName:  "test_http_endpoint_dispatch_covers_gap",
	}
	code := genPytestClassTestForCoverageTask(httpClass, &dispatchTask)
	assertPyGenerated(t, code, []string{
		"class AsyncEndpoint(HTTPEndpoint):",
		"class SyncEndpoint(HTTPEndpoint):",
		"get_scope = {'type': 'http', 'method': 'GET', 'path': '/', 'headers': []}",
		"head_scope = {'type': 'http', 'method': 'HEAD', 'path': '/', 'headers': []}",
		"post_scope = {'type': 'http', 'method': 'POST', 'path': '/', 'headers': []}",
		"assert messages[0]['status'] == 405",
	}, []string{
		"HTTPEndpoint(None, None, None)",
	})

	methodTask := types.CoverageTestTask{
		ID:        "pytest-starlette-endpoint-2",
		Target:    "HTTPEndpoint.method_not_allowed",
		GapType:   "branch",
		LineRange: "52-55",
		TestName:  "test_http_endpoint_method_not_allowed_covers_gap",
	}
	httpClass.Methods[0] = pyFuncInfo{Name: "method_not_allowed", Params: []pyParamInfo{{Name: "request"}}, IsAsync: true, Analysis: pyFuncAnalysis{Raises: true, HasReturn: true, ReturnType: "Response"}}
	code = genPytestClassTestForCoverageTask(httpClass, &methodTask)
	assertPyGenerated(t, code, []string{
		"request = requests.Request(scope, receive=receive)",
		"response = asyncio.run(instance.method_not_allowed(request))",
		"assert response.status_code == 405",
		"app_scope['app'] = object()",
		"except exceptions.HTTPException as exc:",
	}, []string{
		"HTTPEndpoint(None, None, None)",
		"method_not_allowed(None)",
	})

	websocketClass := pyClassInfo{
		Name: "WebSocketEndpoint",
		Methods: []pyFuncInfo{
			{Name: "decode", Params: []pyParamInfo{{Name: "websocket"}, {Name: "message"}}, IsAsync: true, Analysis: pyFuncAnalysis{Raises: true, HasReturn: true, ReturnType: "unknown"}},
		},
	}
	decodeTask := types.CoverageTestTask{
		ID:        "pytest-starlette-endpoint-3",
		Target:    "WebSocketEndpoint.decode",
		GapType:   "branch",
		LineRange: "91-117",
		TestName:  "test_websocket_endpoint_decode_covers_gap",
	}
	code = genPytestClassTestForCoverageTask(websocketClass, &decodeTask)
	assertPyGenerated(t, code, []string{
		"class DummyWebSocket:",
		"instance.encoding = 'json'",
		"assert asyncio.run(instance.decode(websocket, {'type': 'websocket.receive', 'text': '{\"ok\": true}'})) == {'ok': True}",
		"assert asyncio.run(instance.decode(websocket, {'type': 'websocket.receive', 'bytes': b'{\"ok\": true}'})) == {'ok': True}",
		"instance.encoding = 'text'",
		"instance.encoding = 'bytes'",
		"assert websocket.closed == [1003, 1003]",
	}, []string{
		"WebSocketEndpoint(None, None, None)",
		"instance.decode(None, 'test')",
	})

	dispatchWsTask := types.CoverageTestTask{
		ID:        "pytest-starlette-endpoint-4",
		Target:    "WebSocketEndpoint.dispatch",
		GapType:   "branch",
		LineRange: "76-87",
		TestName:  "test_websocket_endpoint_dispatch_covers_gap",
	}
	websocketClass.Methods[0] = pyFuncInfo{Name: "dispatch", IsAsync: true, Analysis: pyFuncAnalysis{Raises: true, HasReturn: false}}
	code = genPytestClassTestForCoverageTask(websocketClass, &dispatchWsTask)
	assertPyGenerated(t, code, []string{
		"class RecordingEndpoint(WebSocketEndpoint):",
		"{'type': 'websocket.connect'}",
		"{'type': 'websocket.receive', 'text': 'hello'}",
		"{'type': 'websocket.disconnect', 'code': 1001}",
		"assert instance.received == ['hello']",
		"assert failing.disconnected == [1011]",
	}, []string{
		"WebSocketEndpoint(None, None, None)",
	})
}

func TestPytestCoverageTaskUsesStarletteMultipartInputs(t *testing.T) {
	parserClass := pyClassInfo{
		Name: "MultiPartParser",
		Methods: []pyFuncInfo{
			{Name: "on_part_data", Params: []pyParamInfo{{Name: "data"}, {Name: "start"}, {Name: "end"}}, Analysis: pyFuncAnalysis{Raises: true, HasReturn: false}},
		},
	}
	dataTask := types.CoverageTestTask{
		ID:        "pytest-starlette-multipart-1",
		Target:    "MultiPartParser.on_part_data",
		GapType:   "branch",
		LineRange: "182-186",
		TestName:  "test_multipart_parser_on_part_data_covers_gap",
	}
	code := genPytestClassTestForCoverageTask(parserClass, &dataTask)
	assertPyGenerated(t, code, []string{
		"headers = datastructures.Headers({'content-type': 'multipart/form-data; boundary=x'})",
		"instance = MultiPartParser(headers, stream(), max_part_size=3)",
		"instance.on_part_data(b'ab', 0, 2)",
		"assert instance._current_part.data == bytearray(b'ab')",
		"instance.on_part_data(b'cdef', 0, 4)",
	}, []string{
		"MultiPartParser(None, __import__('io').BytesIO(b'test'), None, None, 1)",
		"on_part_data({}, None, None)",
	})

	endTask := types.CoverageTestTask{
		ID:        "pytest-starlette-multipart-2",
		Target:    "MultiPartParser.on_part_end",
		GapType:   "branch",
		LineRange: "191-192",
		TestName:  "test_multipart_parser_on_part_end_covers_gap",
	}
	parserClass.Methods[0] = pyFuncInfo{Name: "on_part_end", Analysis: pyFuncAnalysis{HasReturn: false}}
	code = genPytestClassTestForCoverageTask(parserClass, &endTask)
	assertPyGenerated(t, code, []string{
		"instance = MultiPartParser(headers, stream())",
		"instance._current_part.field_name = 'field'",
		"instance._current_part.data.extend(b'value')",
		"result = instance.on_part_end()",
		"assert instance.items == [('field', 'value')]",
	}, []string{
		"MultiPartParser(None, __import__('io').BytesIO(b'test'), None, None, 1)",
	})
}

func TestPytestCoverageTaskUsesApkParserInputs(t *testing.T) {
	findIconFunc := pyFuncInfo{
		Name:   "_find_icon_in_zip",
		Params: []pyParamInfo{{Name: "apk_path"}},
		Analysis: pyFuncAnalysis{
			HasReturn:  true,
			ReturnType: "tuple",
		},
	}
	iconTask := types.CoverageTestTask{
		ID:        "pytest-apk-icon-1",
		Target:    "_find_icon_in_zip",
		GapType:   "branch",
		LineRange: "113-117",
		TestName:  "test_find_icon_in_zip_covers_gap",
	}
	code := genPytestFuncTestForCoverageTask(findIconFunc, &iconTask)
	assertPyGenerated(t, code, []string{
		"handle = tempfile.NamedTemporaryFile(suffix='.apk', delete=False)",
		"zf.writestr(\"res/mipmap-xxxhdpi/ic_launcher.png\", b'icon')",
		"result = _find_icon_in_zip(handle.name)",
		"assert result == (b'icon', 'png')",
	}, []string{
		"_find_icon_in_zip('test')",
	})

	fallbackFunc := pyFuncInfo{
		Name:   "_fallback_from_filename",
		Params: []pyParamInfo{{Name: "apk_path"}, {Name: "result"}},
		Analysis: pyFuncAnalysis{
			HasReturn: false,
		},
	}
	fallbackTask := types.CoverageTestTask{
		ID:        "pytest-apk-fallback-1",
		Target:    "_fallback_from_filename",
		GapType:   "branch",
		LineRange: "135-136",
		TestName:  "test_fallback_from_filename_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(fallbackFunc, &fallbackTask)
	assertPyGenerated(t, code, []string{
		"result = {'package_name': '', 'app_name': ''}",
		"_fallback_from_filename('/tmp/My App.apk', result)",
		"assert result['package_name'] == 'my.app'",
		"assert result['app_name'] == 'My App'",
	}, []string{
		"_fallback_from_filename('test', None)",
	})

	packageFallbackTask := types.CoverageTestTask{
		ID:        "pytest-apk-fallback-2",
		Target:    "_fallback_from_filename",
		GapType:   "branch",
		LineRange: "138-139",
		TestName:  "test_fallback_from_filename_package_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(fallbackFunc, &packageFallbackTask)
	assertPyGenerated(t, code, []string{
		"_fallback_from_filename('/tmp/com.example.app.apk', result)",
		"assert result['package_name'] == 'com.example.app'",
	}, []string{
		"_fallback_from_filename('test', None)",
	})

	parseFunc := pyFuncInfo{
		Name:   "parse_apk",
		Params: []pyParamInfo{{Name: "apk_path"}},
		Analysis: pyFuncAnalysis{
			HasReturn:  true,
			ReturnType: "dict",
		},
	}
	unavailableTask := types.CoverageTestTask{
		ID:        "pytest-apk-parse-1",
		Target:    "parse_apk",
		GapType:   "branch",
		LineRange: "36-38",
		TestName:  "test_parse_apk_unavailable_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(parseFunc, &unavailableTask)
	assertPyGenerated(t, code, []string{
		"module = __import__('app.utils.apk_parser', fromlist=['parse_apk'])",
		"module.APK_INFO_AVAILABLE = False",
		"result = module.parse_apk('missing.apk')",
		"assert result['version_name'] == '1.0'",
	}, []string{
		"parse_apk('test')",
	})

	parseTask := types.CoverageTestTask{
		ID:        "pytest-apk-parse-2",
		Target:    "parse_apk",
		GapType:   "branch",
		LineRange: "44-70",
		TestName:  "test_parse_apk_metadata_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(parseFunc, &parseTask)
	assertPyGenerated(t, code, []string{
		"class FakeAPK:",
		"module.APK = FakeAPK",
		"module._extract_icon = lambda apk, path: (b'icon', 'png')",
		"result = module.parse_apk('sample.apk')",
		"assert result['package_name'] == 'com.example.app'",
		"assert result['version_code'] == 7",
		"assert result['icon_data'] == b'icon'",
	}, []string{
		"parse_apk('test')",
	})
}

func TestPytestCoverageTaskUsesFastAPIAppInputs(t *testing.T) {
	versionFunc := pyFuncInfo{
		Name: "build_version_out",
		Params: []pyParamInfo{
			{Name: "version"},
			{Name: "app_short_code", HasDefault: true},
		},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "dict"},
	}
	versionTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-version-1",
		Target:    "build_version_out",
		GapType:   "branch",
		LineRange: "68-70",
		TestName:  "test_build_version_out_covers_gap",
	}
	code := genPytestFuncTestForCoverageTask(versionFunc, &versionTask)
	assertPyGenerated(t, code, []string{
		"class DetachedVersion:",
		"def app(self):",
		"raise RuntimeError('detached')",
		"result = build_version_out(DetachedVersion())",
		"assert result['short_code'] is None",
		"assert result['share_url'] is None",
	}, []string{
		"build_version_out(None, None)",
	})

	buildAppFunc := pyFuncInfo{
		Name: "build_app_out",
		Params: []pyParamInfo{
			{Name: "app"},
			{Name: "db"},
		},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "dict"},
	}
	buildAppTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-build-app-1",
		Target:    "build_app_out",
		GapType:   "statement",
		LineRange: "101-101",
		TestName:  "test_build_app_out_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(buildAppFunc, &buildAppTask)
	assertPyGenerated(t, code, []string{
		"class FakeDB:",
		"result = build_app_out(app, FakeDB([None, version], 5))",
		"assert result['latest_version'] == '1.2.3'",
		"assert result['total_downloads'] == 5",
	}, []string{
		"build_app_out(None, None)",
	})

	listAppsFunc := pyFuncInfo{
		Name: "list_apps",
		Params: []pyParamInfo{
			{Name: "page", HasDefault: true},
			{Name: "page_size", HasDefault: true},
			{Name: "search", HasDefault: true},
			{Name: "current_user"},
			{Name: "db"},
		},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "dict"},
	}
	listAppsTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-list-apps-1",
		Target:    "list_apps",
		GapType:   "statement",
		LineRange: "410-410",
		TestName:  "test_list_apps_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(listAppsFunc, &listAppsTask)
	assertPyGenerated(t, code, []string{
		"class FakeDB:",
		"result = list_apps(1, 20, 'Example', SimpleNamespace(id=1), db)",
		"assert result['total'] == 1",
		"assert result['items'][0]['latest_version'] == '1.2.3'",
		"assert db.filtered is True",
	}, []string{
		"list_apps(1, 1, None, {}, None)",
	})

	deleteFunc := pyFuncInfo{
		Name:     "_delete_icon_file",
		Params:   []pyParamInfo{{Name: "icon_url"}},
		Analysis: pyFuncAnalysis{HasReturn: false},
	}
	deleteEmptyTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-delete-1",
		Target:    "_delete_icon_file",
		GapType:   "branch",
		LineRange: "682-684",
		TestName:  "test_delete_icon_file_empty_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(deleteFunc, &deleteEmptyTask)
	assertPyGenerated(t, code, []string{
		"result = _delete_icon_file('')",
		"assert result is None",
	}, []string{
		"_delete_icon_file('https://example.com')",
	})

	deleteKeyTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-delete-2",
		Target:    "_delete_icon_file",
		GapType:   "branch",
		LineRange: "691-700",
		TestName:  "test_delete_icon_file_key_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(deleteFunc, &deleteKeyTask)
	assertPyGenerated(t, code, []string{
		"module = __import__('app.api.apps', fromlist=['delete_file'])",
		"module.delete_file = lambda key: calls.append(key)",
		"_delete_icon_file('https://cdn.test/apps/example-icon.png')",
		"assert calls == ['apps/example-icon.png']",
	}, []string{
		"assert isinstance(result, int)",
	})

	shortLinkFunc := pyFuncInfo{
		Name: "short_link_page",
		Params: []pyParamInfo{
			{Name: "short_code"},
			{Name: "db"},
		},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "unknown"},
	}
	for _, tt := range []struct {
		name      string
		lineRange string
		wants     []string
	}{
		{
			name:      "missing app",
			lineRange: "780-782",
			wants: []string{
				"result = short_link_page('missing', FakeDB(None))",
				"assert result.status_code == 404",
				"assert '链接无效' in result.body.decode()",
			},
		},
		{
			name:      "hidden app",
			lineRange: "785-786",
			wants: []string{
				"app = SimpleNamespace(id=1, app_name='Hidden App', icon_url='', is_hidden=True)",
				"result = short_link_page('hidden', FakeDB(app))",
				"assert result.status_code == 410",
			},
		},
		{
			name:      "latest fallback",
			lineRange: "815-816",
			wants: []string{
				"result = short_link_page('abc123', FakeDB(app, [None, version]))",
				"assert result.status_code == 200",
			},
		},
		{
			name:      "no versions",
			lineRange: "819-820",
			wants: []string{
				"result = short_link_page('empty', FakeDB(app, [None, None]))",
				"assert '暂无可用版本' in result.body.decode()",
			},
		},
		{
			name:      "current version query",
			lineRange: "811-811",
			wants: []string{
				"result = short_link_page('rich', FakeDB(app, [version]))",
				"assert result.status_code == 200",
				"assert '&lt;Example&gt;' in body",
			},
		},
		{
			name:      "file size fallback",
			lineRange: "822-824",
			wants: []string{
				"version = SimpleNamespace(id=10, version_name='2.0.0', version_code=8, file_size=None, release_notes='')",
				"assert '大小：? MB' in body",
				"assert '/api/versions/10/download' in body",
			},
		},
		{
			name:      "rich html fields",
			lineRange: "826-832",
			wants: []string{
				"app = SimpleNamespace(id=1, app_name='<Example>', icon_url='https://cdn.test/icon.png', is_hidden=False)",
				"version = SimpleNamespace(id=11, version_name='3.0.0', version_code=9, file_size=2097152, release_notes='one\\ntwo')",
				"assert '&lt;Example&gt;' in body",
				"assert 'https://cdn.test/icon.png' in body",
			},
		},
		{
			name:      "html content assignment",
			lineRange: "835-835",
			wants: []string{
				"result = short_link_page('rich', FakeDB(app, [version]))",
				"body = result.body.decode()",
				"assert result.status_code == 200",
			},
		},
		{
			name:      "final html response",
			lineRange: "877-877",
			wants: []string{
				"result = short_link_page('rich', FakeDB(app, [version]))",
				"body = result.body.decode()",
				"assert result.status_code == 200",
				"assert '&lt;Example&gt;' in body",
			},
		},
		{
			name:      "file-level default",
			lineRange: "entire file",
			wants: []string{
				"result = short_link_page('rich', FakeDB(app, [version]))",
				"body = result.body.decode()",
				"assert result.status_code == 200",
				"assert '&lt;Example&gt;' in body",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			task := types.CoverageTestTask{
				ID:        "pytest-fastapi-short-link",
				Target:    "short_link_page",
				GapType:   "branch",
				LineRange: tt.lineRange,
				TestName:  "test_short_link_page_covers_gap",
			}
			code := genPytestFuncTestForCoverageTask(shortLinkFunc, &task)
			baseWants := []string{
				"class FakeQuery:",
				"class FakeDB:",
				"def query(self, model):",
			}
			assertPyGenerated(t, code, append(baseWants, tt.wants...), []string{
				"short_link_page(None, None)",
			})
		})
	}
}

func TestPytestCoverageTaskUsesFastAPIAuthAndMainInputs(t *testing.T) {
	apiKeyFunc := pyFuncInfo{
		Name: "get_current_user_by_api_key",
		Params: []pyParamInfo{
			{Name: "request"},
			{Name: "db"},
		},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "unknown"},
	}
	apiKeyTask := types.CoverageTestTask{
		ID:              "pytest-fastapi-api-key-1",
		Target:          "get_current_user_by_api_key",
		GapType:         "branch",
		LineRange:       "148-150",
		TestName:        "test_get_current_user_by_api_key_covers_gap",
		MissingBranches: []string{"未覆盖 if 分支: not form"},
	}
	code := genPytestFuncTestForCoverageTask(apiKeyFunc, &apiKeyTask)
	assertPyGenerated(t, code, []string{
		"request = SimpleNamespace(headers={}, query_params={})",
		"result = get_current_user_by_api_key(request, None)",
		"assert result is None",
	}, []string{
		"get_current_user_by_api_key(None, None)",
	})

	authWithoutRaiseFunc := pyFuncInfo{
		Name: "get_current_user_without_raise",
		Params: []pyParamInfo{
			{Name: "token"},
			{Name: "db"},
		},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "unknown"},
	}
	authWithoutRaiseTask := types.CoverageTestTask{
		ID:              "pytest-fastapi-auth-without-raise-1",
		Target:          "get_current_user_without_raise",
		GapType:         "error_path",
		LineRange:       "136-138",
		TestName:        "test_get_current_user_without_raise_covers_gap",
		MissingBranches: []string{"未覆盖错误或空值返回路径"},
	}
	code = genPytestFuncTestForCoverageTask(authWithoutRaiseFunc, &authWithoutRaiseTask)
	assertPyGenerated(t, code, []string{
		"module = __import__('app.api.auth', fromlist=['get_current_user_without_raise'])",
		"assert get_current_user_without_raise(None, None) is None",
		"module.decode_token = lambda token: (_ for _ in ()).throw(RuntimeError('bad token'))",
		"assert get_current_user_without_raise('bad-token', object()) is None",
		"module.decode_token = lambda token: 'alice'",
		"module.get_user_by_username = lambda db, username: SimpleNamespace(username=username)",
		"result = get_current_user_without_raise('good-token', object())",
		"assert result.username == 'alice'",
	}, []string{
		"with pytest.raises(Exception):",
	})

	authServiceAPIKeyFunc := pyFuncInfo{
		Name: "get_user_by_api_key",
		Params: []pyParamInfo{
			{Name: "db"},
			{Name: "key"},
		},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "unknown"},
	}
	authServiceAPIKeyTask := types.CoverageTestTask{
		ID:              "pytest-fastapi-auth-service-api-key-1",
		Target:          "get_user_by_api_key",
		GapType:         "branch",
		LineRange:       "140-141",
		TestName:        "test_get_user_by_api_key_covers_gap",
		MissingBranches: []string{"未覆盖 if 分支: not api_key: return None"},
	}
	code = genPytestFuncTestForCoverageTask(authServiceAPIKeyFunc, &authServiceAPIKeyTask)
	assertPyGenerated(t, code, []string{
		"class FakeDB:",
		"missing_db = FakeDB(None)",
		"assert get_user_by_api_key(missing_db, 'sk_missing') is None",
		"api_key = SimpleNamespace(user=user, last_used_at=None)",
		"assert get_user_by_api_key(db, 'sk_live') is user",
		"assert db.committed is True",
	}, []string{
		"get_user_by_api_key(None, 'test')",
	})

	refreshFunc := pyFuncInfo{
		Name:     "verify_refresh_token",
		Params:   []pyParamInfo{{Name: "token"}},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "unknown"},
	}
	refreshTask := types.CoverageTestTask{
		ID:              "pytest-fastapi-refresh-token-1",
		Target:          "verify_refresh_token",
		GapType:         "branch",
		LineRange:       "72-80",
		TestName:        "test_verify_refresh_token_covers_gap",
		MissingBranches: []string{"未覆盖 if 分支: token_type != \"refresh\": return None"},
	}
	code = genPytestFuncTestForCoverageTask(refreshFunc, &refreshTask)
	assertPyGenerated(t, code, []string{
		"from app.services.auth_service import create_access_token, create_refresh_token",
		"access_token = create_access_token({'sub': 'alice'})",
		"assert verify_refresh_token(access_token) is None",
		"refresh_token = create_refresh_token({'sub': 'alice'})",
		"assert verify_refresh_token(refresh_token) == 'alice'",
		"assert verify_refresh_token('not-a-jwt') is None",
	}, []string{
		"verify_refresh_token(None)",
	})

	createAPIKeyFunc := pyFuncInfo{
		Name: "create_api_key",
		Params: []pyParamInfo{
			{Name: "db"},
			{Name: "user_id"},
			{Name: "name"},
		},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "unknown"},
	}
	createAPIKeyTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-create-api-key-1",
		Target:    "create_api_key",
		GapType:   "statement",
		File:      "/tmp/app/services/auth_service.py",
		LineRange: "126-126",
		TestName:  "test_create_api_key_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(createAPIKeyFunc, &createAPIKeyTask)
	assertPyGenerated(t, code, []string{
		"module = __import__('app.services.auth_service', fromlist=['create_api_key'])",
		"keys = iter(['sk_duplicate', 'sk_unique'])",
		"module.generate_api_key = lambda: next(keys)",
		"result = module.create_api_key(db, 7, 'deploy')",
		"assert result.key == 'sk_unique'",
		"assert db.first_calls == 2",
	}, []string{
		"create_api_key(None, 1, 'test')",
	})

	decodeTokenFunc := pyFuncInfo{
		Name: "decode_token",
		Params: []pyParamInfo{
			{Name: "token"},
			{Name: "verify_exp", HasDefault: true},
		},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "unknown"},
	}
	decodeTokenTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-decode-token-1",
		Target:    "decode_token",
		GapType:   "statement",
		File:      "/tmp/app/services/auth_service.py",
		LineRange: "63-63",
		TestName:  "test_decode_token_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(decodeTokenFunc, &decodeTokenTask)
	assertPyGenerated(t, code, []string{
		"from app.services.auth_service import create_access_token",
		"token = create_access_token({'sub': 'alice'})",
		"assert decode_token(token) == 'alice'",
		"assert decode_token(token, verify_exp=False) == 'alice'",
		"assert decode_token('not-a-jwt') is None",
	}, []string{
		"decode_token(None, None)",
	})

	listUsersFunc := pyFuncInfo{
		Name: "list_users",
		Params: []pyParamInfo{
			{Name: "current_user"},
			{Name: "db"},
		},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "list"},
	}
	listUsersTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-list-users-1",
		Target:    "list_users",
		GapType:   "return_path",
		LineRange: "84-84",
		TestName:  "test_list_users_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(listUsersFunc, &listUsersTask)
	assertPyGenerated(t, code, []string{
		"class FakeDB:",
		"user = SimpleNamespace(id=1, username='admin', is_admin=True)",
		"result = list_users(user, FakeDB([user]))",
		"assert result == [user]",
	}, []string{
		"list_users({}, None)",
	})

	listAPIKeysFunc := pyFuncInfo{
		Name: "list_api_keys",
		Params: []pyParamInfo{
			{Name: "current_user"},
			{Name: "db"},
		},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "list"},
	}
	listAPIKeysTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-list-api-keys-1",
		Target:    "list_api_keys",
		GapType:   "return_path",
		LineRange: "192-192",
		TestName:  "test_list_api_keys_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(listAPIKeysFunc, &listAPIKeysTask)
	assertPyGenerated(t, code, []string{
		"class FakeDB:",
		"api_key = SimpleNamespace(id=1, user_id=7, name='ci', key='secret', is_active=True, last_used_at=None, created_at=None)",
		"result = list_api_keys(user, db)",
		"assert result == [api_key]",
		"assert db.query_obj.filtered is True",
	}, []string{
		"list_api_keys({}, None)",
	})

	qrFunc := pyFuncInfo{
		Name:     "generate_qr_data_url",
		Params:   []pyParamInfo{{Name: "text"}, {Name: "size", HasDefault: true}},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "str", Raises: true},
	}
	qrTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-qr-1",
		Target:    "generate_qr_data_url",
		GapType:   "error_path",
		LineRange: "57-59",
		TestName:  "test_generate_qr_data_url_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(qrFunc, &qrTask)
	assertPyGenerated(t, code, []string{
		"def fake_import(name, *args, **kwargs):",
		"if name == 'qrcode':",
		"result = generate_qr_data_url('https://example.com/download')",
		"assert result == ''",
	}, []string{
		"with pytest.raises(Exception):",
		"generate_qr_data_url('test', 1)",
	})

	lifespanFunc := pyFuncInfo{
		Name:    "lifespan",
		Params:  []pyParamInfo{{Name: "app"}},
		IsAsync: true,
	}
	serveTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-frontend-1",
		Target:    "serve_frontend",
		GapType:   "branch",
		LineRange: "86-89",
		TestName:  "test_serve_frontend_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(lifespanFunc, &serveTask)
	assertPyGenerated(t, code, []string{
		"manual_review_environment: serve_frontend is defined only when frontend/dist exists at app import time",
	}, []string{
		"asyncio.run(lifespan(None))",
	})

	serveRootTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-frontend-2",
		Target:    "serve_root_file",
		GapType:   "return_path",
		LineRange: "73-79",
		TestName:  "test_serve_root_file_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(lifespanFunc, &serveRootTask)
	assertPyGenerated(t, code, []string{
		"manual_review_environment: serve_root_file is defined only when frontend/dist exists at app import time",
	}, []string{
		"asyncio.run(lifespan(None))",
	})

	mainTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-lifespan-1",
		Target:    "main.py",
		GapType:   "statement",
		File:      "/tmp/app/main.py",
		LineRange: "69-71",
		TestName:  "test_main_py_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(lifespanFunc, &mainTask)
	assertPyGenerated(t, code, []string{
		"module = __import__('app.main', fromlist=['lifespan'])",
		"module.init_db = lambda: calls.append('init_db')",
		"async def _run_lifespan():",
		"async with lifespan(None):",
		"asyncio.run(_run_lifespan())",
		"assert calls == ['init_db', 'ensure_admin_exists', 'close', 'yielded']",
	}, []string{
		"asyncio.run(lifespan(None))",
	})
}

func TestPytestCoverageTaskUsesFastAPIStorageInputs(t *testing.T) {
	saveIconFunc := pyFuncInfo{
		Name: "_save_icon_local",
		Params: []pyParamInfo{
			{Name: "key"},
			{Name: "data"},
		},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "str"},
	}
	saveIconTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-qiniu-save-icon-1",
		Target:    "_save_icon_local",
		GapType:   "return_path",
		File:      "/tmp/app/services/qiniu_service.py",
		LineRange: "44-51",
		TestName:  "test_save_icon_local_covers_gap",
	}
	code := genPytestFuncTestForCoverageTask(saveIconFunc, &saveIconTask)
	assertPyGenerated(t, code, []string{
		"module = __import__('app.services.qiniu_service', fromlist=['_save_icon_local'])",
		"with tempfile.TemporaryDirectory() as tmpdir:",
		"module.LOCAL_ICON_ROOT = tmpdir",
		"result = module._save_icon_local('icons/com.example.app/ic_launcher.png', b'png-data')",
		"assert handle.read() == b'png-data'",
	}, []string{
		"_save_icon_local('test', {})",
	})

	qiniuAuthFunc := pyFuncInfo{
		Name:     "get_qiniu_auth",
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "unknown", Raises: true},
	}
	qiniuAuthTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-qiniu-auth-1",
		Target:    "get_qiniu_auth",
		GapType:   "error_path",
		File:      "/tmp/app/services/qiniu_service.py",
		LineRange: "22-26",
		TestName:  "test_get_qiniu_auth_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(qiniuAuthFunc, &qiniuAuthTask)
	assertPyGenerated(t, code, []string{
		"module = __import__('app.services.qiniu_service', fromlist=['get_qiniu_auth'])",
		"def fake_import(name, *args, **kwargs):",
		"if name == 'qiniu':",
		"module.get_qiniu_auth()",
		"assert 'qiniu SDK' in str(exc)",
	}, []string{
		"with pytest.raises(Exception):",
	})

	uploadBytesFunc := pyFuncInfo{
		Name: "upload_bytes",
		Params: []pyParamInfo{
			{Name: "data"},
			{Name: "key"},
			{Name: "mime_type", HasDefault: true},
		},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "tuple", Raises: true},
	}
	qiniuUploadTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-qiniu-upload-bytes-1",
		Target:    "upload_bytes",
		GapType:   "error_path",
		File:      "/tmp/app/services/qiniu_service.py",
		LineRange: "118-125",
		TestName:  "test_upload_bytes_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(uploadBytesFunc, &qiniuUploadTask)
	assertPyGenerated(t, code, []string{
		"module = __import__('app.services.qiniu_service', fromlist=['upload_bytes'])",
		"module._is_qiniu_configured = lambda: True",
		"sys.modules['qiniu'] = SimpleNamespace(put_data=lambda *args, **kwargs: ({}, Info()))",
		"assert ok is True",
		"assert 'remote failed' in message and 'local fail' in message",
	}, []string{
		"with pytest.raises(Exception):",
		"upload_bytes({}, 'test', 'test')",
	})

	qiniuMoveFunc := pyFuncInfo{
		Name:     "move_file",
		Params:   []pyParamInfo{{Name: "src_key"}, {Name: "dest_key"}},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "tuple", Raises: true},
	}
	qiniuMoveTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-qiniu-move-1",
		Target:    "move_file",
		GapType:   "error_path",
		File:      "/tmp/app/services/qiniu_service.py",
		LineRange: "243-246",
		TestName:  "test_move_file_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(qiniuMoveFunc, &qiniuMoveTask)
	assertPyGenerated(t, code, []string{
		"module = __import__('app.services.qiniu_service', fromlist=['move_file'])",
		"def stat(self, *args, **kwargs):",
		"def move(self, *args, **kwargs):",
		"assert 'move failed' in message",
		"assert 'bucket boom' in message",
	}, []string{
		"with pytest.raises(Exception):",
		"move_file('test', 'test')",
	})

	qiniuDownloadFunc := pyFuncInfo{
		Name:     "download_to_temp",
		Params:   []pyParamInfo{{Name: "key"}},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "tuple", Raises: true},
	}
	qiniuDownloadTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-qiniu-download-1",
		Target:    "download_to_temp",
		GapType:   "error_path",
		File:      "/tmp/app/services/qiniu_service.py",
		LineRange: "279-282",
		TestName:  "test_download_to_temp_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(qiniuDownloadFunc, &qiniuDownloadTask)
	assertPyGenerated(t, code, []string{
		"module = __import__('app.services.qiniu_service', fromlist=['download_to_temp'])",
		"class FakeBucket:",
		"sys.modules['qiniu'] = SimpleNamespace(BucketManager=lambda auth: FakeBucket())",
		"assert 'download fail' in message",
	}, []string{
		"with pytest.raises(Exception):",
		"download_to_temp('test')",
	})

	tosClientFunc := pyFuncInfo{
		Name:     "_get_client",
		Params:   []pyParamInfo{{Name: "internal", HasDefault: true}},
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "unknown", Raises: true},
	}
	tosClientTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-tos-client-1",
		Target:    "_get_client",
		GapType:   "error_path",
		File:      "/tmp/app/services/tos_service.py",
		LineRange: "46-50",
		TestName:  "test_get_client_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(tosClientFunc, &tosClientTask)
	assertPyGenerated(t, code, []string{
		"module = __import__('app.services.tos_service', fromlist=['_get_client'])",
		"sys.modules['tos'] = SimpleNamespace(TosClientV2=lambda *args, **kwargs: (_ for _ in ()).throw(RuntimeError('client boom')))",
		"assert module._get_client(True) is None",
	}, []string{
		"with pytest.raises(Exception):",
		"_get_client(None)",
	})

	tosMoveTask := types.CoverageTestTask{
		ID:        "pytest-fastapi-tos-move-1",
		Target:    "move_file",
		GapType:   "error_path",
		File:      "/tmp/app/services/tos_service.py",
		LineRange: "160-164",
		TestName:  "test_move_file_covers_gap",
	}
	code = genPytestFuncTestForCoverageTask(qiniuMoveFunc, &tosMoveTask)
	assertPyGenerated(t, code, []string{
		"module = __import__('app.services.tos_service', fromlist=['move_file'])",
		"class SuccessClient:",
		"assert message == 'https://cdn.example.com/new.apk'",
		"assert 'copy failed' in message",
	}, []string{
		"with pytest.raises(Exception):",
		"move_file('test', 'test')",
	})
}

func TestPytestCoverageTaskUsesFastAPIDatabaseMigrationInputs(t *testing.T) {
	migrateFunc := pyFuncInfo{
		Name:     "_migrate_short_code_to_app",
		Analysis: pyFuncAnalysis{HasReturn: true, ReturnType: "unknown"},
	}
	for _, tt := range []struct {
		name      string
		lineRange string
		wants     []string
	}{
		{
			name:      "current version already has app code",
			lineRange: "126-128",
			wants: []string{
				"fake_db = FakeDB([(1, 'current-code')], [])",
				"assert fake_db.updates == []",
			},
		},
		{
			name:      "latest version already has app code",
			lineRange: "144-146",
			wants: []string{
				"fake_db = FakeDB([], [(2, 'latest-code')])",
				"assert fake_db.updates == []",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			task := types.CoverageTestTask{
				ID:        "pytest-fastapi-migrate-1",
				Target:    "_migrate_short_code_to_app",
				GapType:   "branch",
				LineRange: tt.lineRange,
				TestName:  "test_migrate_short_code_to_app_covers_gap",
			}
			code := genPytestFuncTestForCoverageTask(migrateFunc, &task)
			baseWants := []string{
				"sqlalchemy.inspect = lambda engine: FakeInspector()",
				"module.SessionLocal = lambda: fake_db",
				"result = _migrate_short_code_to_app()",
				"assert result is None",
				"assert fake_db.committed is True",
			}
			assertPyGenerated(t, code, append(baseWants, tt.wants...), []string{
				"assert result == (#",
			})
		})
	}
}

func assertPyGenerated(t *testing.T, code string, wants []string, forbidden []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, bad := range forbidden {
		if strings.Contains(code, bad) {
			t.Fatalf("did not expect %q in generated code:\n%s", bad, code)
		}
	}
}

func TestPytestFuncCoverageTaskCoversAsyncAndErrorBranches(t *testing.T) {
	asyncTask := types.CoverageTestTask{
		TestName: "covers async lookup",
		SuggestedInputs: []string{
			"构造满足条件 `user_id == 42` 的输入",
		},
	}
	asyncFunc := pyFuncInfo{
		Name:    "lookup",
		IsAsync: true,
		Params:  []pyParamInfo{{Name: "user_id"}},
		Analysis: pyFuncAnalysis{
			HasReturn: true,
			Returns:   []string{"user_id"},
		},
	}
	code := genPytestFuncTestForCoverageTask(asyncFunc, &asyncTask)
	for _, want := range []string{
		"def test_covers_async_lookup():",
		"result = asyncio.run(lookup(42))",
		"assert result == (42)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in async function task output:\n%s", want, code)
		}
	}

	errorTask := types.CoverageTestTask{
		GapType:  "error_path",
		TestName: "!!!",
		SuggestedInputs: []string{
			"构造满足条件 `url == None` 的输入",
		},
	}
	errorFunc := pyFuncInfo{
		Name:    "fetch",
		IsAsync: true,
		Params:  []pyParamInfo{{Name: "url"}},
	}
	code = genPytestFuncTestForCoverageTask(errorFunc, &errorTask)
	for _, want := range []string{
		"def test_fetch_covers_gap():",
		"with pytest.raises(Exception):\n        asyncio.run(fetch(None))",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in error-path function task output:\n%s", want, code)
		}
	}
	if strings.Contains(code, "result =") {
		t.Fatalf("error-path task should not assign result:\n%s", code)
	}
}

func TestPytestClassCoverageTaskCoversStaticAsyncAndInstanceErrorBranches(t *testing.T) {
	task := types.CoverageTestTask{
		GapType:  "branch",
		TestName: "test_worker_gap",
		SuggestedInputs: []string{
			"构造满足条件 `value == 7` 的输入",
		},
	}
	cls := pyClassInfo{
		Name: "Worker",
		Methods: []pyFuncInfo{
			{
				Name:     "stat",
				IsStatic: true,
				Params:   []pyParamInfo{{Name: "value"}},
				Analysis: pyFuncAnalysis{
					HasReturn: true,
					Returns:   []string{"value"},
				},
			},
			{
				Name:     "async_stat",
				IsStatic: true,
				IsAsync:  true,
				Params:   []pyParamInfo{{Name: "url"}},
				Analysis: pyFuncAnalysis{
					HasReturn: true,
					Returns:   []string{"url"},
				},
			},
			{
				Name:    "async_instance",
				IsAsync: true,
				Params:  []pyParamInfo{{Name: "flag"}},
				Analysis: pyFuncAnalysis{
					HasReturn: true,
					Returns:   []string{"flag"},
				},
			},
			{
				Name:   "fail",
				Params: []pyParamInfo{{Name: "enabled"}},
				Analysis: pyFuncAnalysis{
					Raises:     true,
					Boundaries: []pyBoundary{{Param: "enabled", Value: "False"}},
				},
			},
		},
	}

	code := genPytestClassTestForCoverageTask(cls, &task)
	for _, want := range []string{
		"result = Worker.stat(7)\n        assert result == (7)",
		"result = asyncio.run(Worker.async_stat('https://example.com'))\n        assert result == ('https://example.com')",
		"instance = Worker()\n        result = asyncio.run(instance.async_instance(True))\n        assert result == (True)",
		"instance = Worker()\n        with pytest.raises(Exception):\n            instance.fail(False)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in class task output:\n%s", want, code)
		}
	}
}

func TestPytestClassTestCoversAsyncStaticRaisesAndBoundaries(t *testing.T) {
	cls := pyClassInfo{
		Name: "Service",
		Methods: []pyFuncInfo{
			{Name: "__init__"},
			{
				Name:     "fetch",
				IsStatic: true,
				IsAsync:  true,
				Params:   []pyParamInfo{{Name: "url"}},
				Analysis: pyFuncAnalysis{
					ReturnType: "str",
					HasReturn:  true,
				},
			},
			{
				Name:   "normalize",
				Params: []pyParamInfo{{Name: "prefix"}, {Name: "text"}},
				Analysis: pyFuncAnalysis{
					ReturnType: "str",
					Returns:    []string{"prefix + text"},
					HasReturn:  true,
					Boundaries: []pyBoundary{{Param: "prefix", Value: "'>'", ReturnExpr: "text"}},
				},
			},
			{
				Name:    "load",
				IsAsync: true,
				Params:  []pyParamInfo{{Name: "url"}},
				Analysis: pyFuncAnalysis{
					Raises:     true,
					Boundaries: []pyBoundary{{Param: "url", Value: "None"}},
				},
			},
			{
				Name:     "parse",
				IsStatic: true,
				Params:   []pyParamInfo{{Name: "payload"}},
				Analysis: pyFuncAnalysis{
					Raises:     true,
					Boundaries: []pyBoundary{{Param: "payload", Value: "None"}},
				},
			},
		},
	}

	code := genPytestClassTest(cls)
	for _, want := range []string{
		"class TestService:",
		"def test_fetch(self):\n        result = asyncio.run(Service.fetch('https://example.com'))",
		"def test_normalize_prefix_boundary(self):\n        instance = Service()\n        result = instance.normalize('>', 'test')\n        assert result == ('test')",
		"def test_load_raises(self):\n        instance = Service()\n        with pytest.raises(Exception):\n            asyncio.run(instance.load(None))",
		"def test_load_url_boundary(self):\n        instance = Service()\n        with pytest.raises(Exception):\n            asyncio.run(instance.load(None))",
		"def test_parse_raises(self):\n        with pytest.raises(Exception):\n            Service.parse(None)",
		"def test_parse_payload_boundary(self):\n        with pytest.raises(Exception):\n            Service.parse(None)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in class pytest output:\n%s", want, code)
		}
	}
	if strings.Contains(code, "test___init__") {
		t.Fatalf("__init__ should be skipped:\n%s", code)
	}
}

func TestPytestAssertionAndArgCompatHelpers(t *testing.T) {
	if got := genPyResultAssertion(pyFuncAnalysis{}, "  "); got != "  # void function, verify no exception\n" {
		t.Fatalf("genPyResultAssertion() = %q", got)
	}
	expr, ok := pyExpectedReturnExpr(pyFuncAnalysis{Returns: []string{"a + b"}}, []pyParamInfo{{Name: "a"}, {Name: "b"}}, nil)
	if !ok || expr != "(1 + 2)" {
		t.Fatalf("pyExpectedReturnExpr() = %q, %v", expr, ok)
	}
	expr, ok = pyExpectedReturnExprWithValues(
		pyFuncAnalysis{Returns: []string{"url + suffix"}},
		[]pyParamInfo{{Name: "url"}, {Name: "suffix"}},
		nil,
		map[string]string{"url": "'https://example.com'", "suffix": "'/v1'"},
	)
	if !ok || expr != "('https://example.com' + '/v1')" {
		t.Fatalf("pyExpectedReturnExprWithValues() = %q, %v", expr, ok)
	}
	if expr, ok = pyExpectedReturnExpr(pyFuncAnalysis{Returns: []string{"unknown + 1"}}, nil, nil); ok || expr != "" {
		t.Fatalf("unsafe pyExpectedReturnExpr() = %q, %v", expr, ok)
	}
	for input, want := range map[string]bool{
		"":       false,
		"1":      true,
		"1.5":    true,
		".5":     false,
		"1.":     false,
		"1.2.3":  false,
		"123abc": false,
	} {
		if got := isPyNumericLiteral(input); got != want {
			t.Fatalf("isPyNumericLiteral(%q) = %v, want %v", input, got, want)
		}
	}
	for input, want := range map[string]bool{
		"a + b":       true,
		"await value": false,
		"lambda x: x": false,
		"value if ok": false,
		"items[0]":    false,
		"":            false,
	} {
		if got := pyReturnExprIsSafe(input); got != want {
			t.Fatalf("pyReturnExprIsSafe(%q) = %v, want %v", input, got, want)
		}
	}
	for input, want := range map[string]string{
		"None":             "None",
		"True":             "bool",
		"'ok'":             "str",
		"f'ok'":            "str",
		"12":               "int",
		"12.5":             "float",
		"[1]":              "list",
		"{'ok': True}":     "dict",
		"(1, 2)":           "tuple",
		"response.json()":  "dict",
		"a // b":           "float",
		"a * b":            "int",
		"name + '!'":       "str",
		"', '.join(items)": "str",
		"custom_value":     "unknown",
	} {
		if got := inferPyReturnType([][]string{{"", input}}); got != want {
			t.Fatalf("inferPyReturnType(%q) = %q, want %q", input, got, want)
		}
	}
	args := pyDefaultArgs([]pyParamInfo{{Name: "items"}, {Name: "kwargs", IsKwargs: true}})
	if args != "[], {}" {
		t.Fatalf("pyDefaultArgs() = %q", args)
	}
	if got := pyPlaceholderArgList([]pyParamInfo{{Name: "args", IsArgs: true}, {Name: "kwargs", IsKwargs: true}, {Name: "value"}}); got != "(), {}, None" {
		t.Fatalf("pyPlaceholderArgList() = %q", got)
	}
	if got := pyArgListWithValues([]pyParamInfo{{Name: "enabled"}, {Name: "title"}}, map[string]string{"title": "'custom'"}); got != "True, 'custom'" {
		t.Fatalf("pyArgListWithValues() = %q", got)
	}
	boundaries := []pyBoundary{{Param: "mode", Value: "'short'"}, {Param: "enabled", Value: "False"}}
	task := &types.CoverageTestTask{SuggestedInputs: []string{"构造满足条件 `enabled == False` 的输入"}}
	if got := pyBoundaryForCoverageTask(boundaries, task); got == nil || got.Param != "enabled" {
		t.Fatalf("pyBoundaryForCoverageTask(exact) = %+v", got)
	}
	if got := pyBoundaryForCoverageTask([]pyBoundary{{Param: "mode", Value: "'short'"}}, &types.CoverageTestTask{GapType: "branch"}); got == nil || got.Param != "mode" {
		t.Fatalf("pyBoundaryForCoverageTask(branch fallback) = %+v", got)
	}
	if got := pyBoundaryForCoverageTask(boundaries, nil); got != nil {
		t.Fatalf("pyBoundaryForCoverageTask(nil) = %+v", got)
	}
}

func TestIsPyDunder(t *testing.T) {
	if !isPyDunder("__init__") {
		t.Error("__init__ should be dunder")
	}
	if !isPyDunder("__str__") {
		t.Error("__str__ should be dunder")
	}
	if isPyDunder("init") {
		t.Error("init should not be dunder")
	}
	if isPyDunder("_private") {
		t.Error("_private should not be dunder")
	}
}

func TestIsPyTestHelper(t *testing.T) {
	for _, name := range []string{"setUp", "tearDown", "setUpClass", "tearDownClass"} {
		if !isPyTestHelper(name) {
			t.Fatalf("isPyTestHelper(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"helper", "test_case", "setup"} {
		if isPyTestHelper(name) {
			t.Fatalf("isPyTestHelper(%q) = true, want false", name)
		}
	}
}
