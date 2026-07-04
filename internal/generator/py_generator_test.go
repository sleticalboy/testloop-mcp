package generator

import (
	"strings"
	"testing"
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
