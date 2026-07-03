package generator

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateJestTests(t *testing.T) {
	code, err := GenerateJestTests("../../demo/calc.js")
	if err != nil {
		t.Fatalf("GenerateJestTests 失败: %v", err)
	}

	// 验证 CommonJS 导入
	if !strings.Contains(code, "require('./calc')") {
		t.Error("缺少 CommonJS require 导入")
	}

	// 验证函数测试
	expectedFuncs := []string{"add", "divide", "fetchData", "greet", "multiply", "formatText"}
	for _, name := range expectedFuncs {
		if !strings.Contains(code, "describe('"+name+"'") {
			t.Errorf("缺少函数测试: %s", name)
		}
	}

	// 验证类测试
	if !strings.Contains(code, "describe('Calculator'") {
		t.Error("缺少 Calculator 类测试")
	}

	// 验证类方法测试
	if !strings.Contains(code, "describe('add'") {
		t.Error("缺少 Calculator.add 方法测试")
	}
	if !strings.Contains(code, "describe('divide'") {
		t.Error("缺少 Calculator.divide 方法测试")
	}

	// 验证实例化测试
	if !strings.Contains(code, "new Calculator()") {
		t.Error("缺少 Calculator 实例化")
	}

	// 验证 async 函数
	if !strings.Contains(code, "fetchData") {
		t.Error("缺少 async 函数测试")
	}

	// 验证变参
	if !strings.Contains(code, "formatText") {
		t.Error("缺少变参函数测试")
	}

	t.Logf("生成的 Jest 测试:\n%s", code)
}

func TestGenerateJestTests_ESModule(t *testing.T) {
	src := `export function foo(a, b) { return a + b; }
export const bar = (x) => x * 2;
export class Baz {
  method(p) { return p; }
}
`
	// 写到临时文件
	tmpPath := t.TempDir() + "/esmod.js"
	if err := writeFile(tmpPath, src); err != nil {
		t.Fatal(err)
	}

	code, err := GenerateJestTests(tmpPath)
	if err != nil {
		t.Fatalf("GenerateJestTests 失败: %v", err)
	}

	// 验证 ES module 导入
	if !strings.Contains(code, "import {") {
		t.Error("缺少 ES module import 导入")
	}
	if !strings.Contains(code, "from './esmod'") {
		t.Error("缺少 from './esmod' 导入路径")
	}

	// 验证导出函数都在导入中
	if !strings.Contains(code, "foo") || !strings.Contains(code, "bar") || !strings.Contains(code, "Baz") {
		t.Error("导入中缺少导出函数/类")
	}
}

func TestParseJSParams(t *testing.T) {
	tests := []struct {
		input   string
		wantLen int
		wantFirst string
	}{
		{"", 0, ""},
		{"a, b", 2, "a"},
		{"a, b, c", 3, "a"},
		{"a = 1, b = 'hello'", 2, "a"},
		{"...args", 1, "args"},
		{"a: number, b: string", 2, "a"}, // TS 类型注解
		{"{a, b}", 1, "a, b"}, // 解构（简化处理，strip {} 后保留原始内容）
	}

	for _, tt := range tests {
		params := parseJSParams(tt.input)
		if len(params) != tt.wantLen {
			t.Errorf("parseJSParams(%q) = %d params, want %d", tt.input, len(params), tt.wantLen)
			continue
		}
		if tt.wantLen > 0 && params[0].Name != tt.wantFirst {
			t.Errorf("parseJSParams(%q)[0].Name = %q, want %q", tt.input, params[0].Name, tt.wantFirst)
		}
	}
}

func TestParseJSParams_RestAndDefault(t *testing.T) {
	params := parseJSParams("a, b = 2, ...rest")
	if len(params) != 3 {
		t.Fatalf("len = %d, want 3", len(params))
	}
	if params[0].Name != "a" || params[0].HasDefault {
		t.Errorf("param[0] = %+v", params[0])
	}
	if !params[1].HasDefault {
		t.Errorf("param[1] should have default, got %+v", params[1])
	}
	if !params[2].IsRest {
		t.Errorf("param[2] should be rest, got %+v", params[2])
	}
}

func TestSplitParams(t *testing.T) {
	tests := []struct {
		input   string
		wantLen int
	}{
		{"a, b, c", 3},
		{"a, {b, c}, d", 3}, // 嵌套不分割
		{"a, [b, c]", 2},     // 嵌套不分割
		{"", 1},              // 空字符串返回 1 个空元素
	}

	for _, tt := range tests {
		parts := splitParams(tt.input)
		if len(parts) != tt.wantLen {
			t.Errorf("splitParams(%q) = %d parts, want %d", tt.input, len(parts), tt.wantLen)
		}
	}
}

func TestFindMatchingBrace(t *testing.T) {
	tests := []struct {
		input    string
		openIdx  int
		wantEnd  int
	}{
		{"{a: 1}", 0, 5},
		{"{{a: 1}, b: 2}", 0, 13},
		{"const x = {a: 1};", 10, 15},
		{`{a: "}"}`, 0, 7}, // 字符串中的 } 不算
	}

	for _, tt := range tests {
		end := findMatchingBrace(tt.input, tt.openIdx)
		if end != tt.wantEnd {
			t.Errorf("findMatchingBrace(%q, %d) = %d, want %d", tt.input, tt.openIdx, end, tt.wantEnd)
		}
	}
}

func TestIsTestHelper(t *testing.T) {
	helpers := []string{"test", "it", "describe", "beforeEach", "afterAll", "expect", "jest"}
	for _, h := range helpers {
		if !isTestHelper(h) {
			t.Errorf("isTestHelper(%q) = false, want true", h)
		}
	}
	nonHelpers := []string{"add", "fetchData", "Calculator", "formatText"}
	for _, h := range nonHelpers {
		if isTestHelper(h) {
			t.Errorf("isTestHelper(%q) = true, want false", h)
		}
	}
}

// 辅助函数
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
