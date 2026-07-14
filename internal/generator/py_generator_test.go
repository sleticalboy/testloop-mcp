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
