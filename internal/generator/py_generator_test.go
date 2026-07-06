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
