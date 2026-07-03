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

	// 验证导入
	if !strings.Contains(code, "from calc import") {
		t.Error("缺少 Python import 导入")
	}

	// 验证函数测试
	expectedFuncs := []string{"add", "divide", "format_text", "fetch_data"}
	for _, name := range expectedFuncs {
		if !strings.Contains(code, "def test_"+name+"(") {
			t.Errorf("缺少函数测试: test_%s", name)
		}
	}

	// 验证类测试
	if !strings.Contains(code, "class TestCalculator:") {
		t.Error("缺少 TestCalculator 类测试")
	}

	// 验证 __init__ 测试
	if !strings.Contains(code, "def test_init(self):") {
		t.Error("缺少 __init__ 测试")
	}

	// 验证方法测试
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

func TestParsePyParams(t *testing.T) {
	tests := []struct {
		input    string
		isMethod bool
		wantLen  int
	}{
		{"", false, 0},
		{"a, b", false, 2},
		{"a, b, c", false, 3},
		{"a: int, b: str", false, 2},    // 类型注解
		{"a = 1, b = 'hello'", false, 2}, // 默认值
		{"*args", false, 1},              // *args
		{"**kwargs", false, 1},           // **kwargs
		{"self, a, b", true, 2},          // 方法：self 被剥离
		{"cls, a", true, 1},              // 类方法：cls 被剥离
	}

	for _, tt := range tests {
		params := parsePyParams(tt.input, tt.isMethod, false)
		if len(params) != tt.wantLen {
			t.Errorf("parsePyParams(%q, isMethod=%v) = %d params, want %d",
				tt.input, tt.isMethod, len(params), tt.wantLen)
		}
	}
}

func TestParsePyParams_RestAndKwargs(t *testing.T) {
	params := parsePyParams("a, *args, **kwargs", false, false)
	if len(params) != 3 {
		t.Fatalf("len = %d, want 3", len(params))
	}
	if params[1].IsArgs != true {
		t.Errorf("param[1] should be *args")
	}
	if params[2].IsKwargs != true {
		t.Errorf("param[2] should be **kwargs")
	}
}

func TestParsePyParams_TypeHints(t *testing.T) {
	params := parsePyParams("a: int, b: str = 'hello'", false, false)
	if len(params) != 2 {
		t.Fatalf("len = %d, want 2", len(params))
	}
	if params[0].Name != "a" {
		t.Errorf("param[0].Name = %q, want 'a'", params[0].Name)
	}
	if params[1].Name != "b" || !params[1].HasDefault {
		t.Errorf("param[1] = %+v, want name='b' hasDefault=true", params[1])
	}
}

func TestIndentLevel(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"    def foo():", 4},
		{"  def foo():", 2},
		{"def foo():", 0},
		{"\tdef foo():", 4},
		{"        def foo():", 8},
	}

	for _, tt := range tests {
		got := indentLevel(tt.input)
		if got != tt.want {
			t.Errorf("indentLevel(%q) = %d, want %d", tt.input, got, tt.want)
		}
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
