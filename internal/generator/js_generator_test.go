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
	tmpPath := t.TempDir() + "/esmod.js"
	if err := writeFile(tmpPath, src); err != nil {
		t.Fatal(err)
	}

	code, err := GenerateJestTests(tmpPath)
	if err != nil {
		t.Fatalf("GenerateJestTests 失败: %v", err)
	}

	if !strings.Contains(code, "import {") {
		t.Error("缺少 ES module import 导入")
	}
	if !strings.Contains(code, "from './esmod'") {
		t.Error("缺少 from './esmod' 导入路径")
	}

	if !strings.Contains(code, "foo") || !strings.Contains(code, "bar") || !strings.Contains(code, "Baz") {
		t.Error("导入中缺少导出函数/类")
	}
}

func TestTreeSitterJS_ParsesAllDeclarations(t *testing.T) {
	source := []byte(`function add(a, b) { return a + b; }
const multiply = (a, b) => a * b;
const greet = function(name) { return "Hello, " + name; };
class Calculator {
  add(a, b) { return a + b; }
  async divide(a, b) { return a / b; }
}
`)

	funcs, classes, isESModule := parseJSWithTreeSitter(source, ".js")

	if isESModule {
		t.Error("不应检测到 ES module")
	}

	expectedFuncs := map[string]bool{"add": false, "multiply": false, "greet": false}
	for _, fn := range funcs {
		if _, ok := expectedFuncs[fn.Name]; ok {
			expectedFuncs[fn.Name] = true
		}
	}
	for name, found := range expectedFuncs {
		if !found {
			t.Errorf("未提取到函数: %s", name)
		}
	}

	if len(classes) != 1 || classes[0].Name != "Calculator" {
		t.Errorf("期望 1 个 Calculator 类, got %d classes", len(classes))
	}
	if len(classes[0].Methods) != 2 {
		t.Errorf("期望 2 个方法, got %d", len(classes[0].Methods))
	}
}

func TestTreeSitterJS_AsyncDetection(t *testing.T) {
	source := []byte(`async function fetchData(url) { return fetch(url); }
const syncFn = (x) => x * 2;
`)
	funcs, _, _ := parseJSWithTreeSitter(source, ".js")

	if len(funcs) != 2 {
		t.Fatalf("期望 2 个函数, got %d", len(funcs))
	}

	// fetchData 应该是 async
	foundAsync := false
	foundSync := false
	for _, fn := range funcs {
		if fn.Name == "fetchData" && fn.IsAsync {
			foundAsync = true
		}
		if fn.Name == "syncFn" && !fn.IsAsync {
			foundSync = true
		}
	}
	if !foundAsync {
		t.Error("fetchData 应该是 async")
	}
	if !foundSync {
		t.Error("syncFn 不应该是 async")
	}
}

func TestTreeSitterJS_ParamsExtraction(t *testing.T) {
	source := []byte(`function formatText(text, prefix = "", ...args) { return text; }
const arrow = (a, b) => a + b;
`)
	funcs, _, _ := parseJSWithTreeSitter(source, ".js")

	// formatText: 3 params (text, prefix=hasDefault, ...args=rest)
	var formatFn *jsFuncInfo
	for i := range funcs {
		if funcs[i].Name == "formatText" {
			formatFn = &funcs[i]
			break
		}
	}
	if formatFn == nil {
		t.Fatal("未找到 formatText 函数")
	}
	if len(formatFn.Params) != 3 {
		t.Fatalf("formatText 期望 3 个参数, got %d", len(formatFn.Params))
	}
	if formatFn.Params[0].Name != "text" {
		t.Errorf("param[0] = %s, want text", formatFn.Params[0].Name)
	}
	if !formatFn.Params[1].HasDefault {
		t.Error("param[1] 应该有默认值")
	}
	if !formatFn.Params[2].IsRest {
		t.Error("param[2] 应该是 rest 参数")
	}
}

func TestJSTestArgsUseSemanticDefaults(t *testing.T) {
	params := []jsParamInfo{
		{Name: "url"},
		{Name: "count"},
		{Name: "enabled"},
		{Name: "items"},
		{Name: "options"},
		{Name: "name"},
	}

	got := jsArgList(params)
	for _, want := range []string{
		"'https://example.com'",
		"1",
		"true",
		"[]",
		"{}",
		"'test'",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("jsArgList() = %q, want value %q", got, want)
		}
	}
	if strings.Contains(got, "undefined") {
		t.Fatalf("jsArgList() = %q, should avoid undefined for recognized params", got)
	}
}

func TestJSTestArgsKeepSemanticDefaultsForBoundaryCases(t *testing.T) {
	params := []jsParamInfo{
		{Name: "url"},
		{Name: "count"},
		{Name: "mode"},
	}

	got := jsArgListWithBoundary(params, jsBoundary{Param: "mode", Value: "null", Type: "null"})
	if got != "'https://example.com', 1, null" {
		t.Fatalf("jsArgListWithBoundary() = %q", got)
	}
}

func TestJSThrowArgsPreferInvalidBoundary(t *testing.T) {
	params := []jsParamInfo{
		{Name: "url"},
		{Name: "count"},
	}
	boundaries := []jsBoundary{{Param: "url", Value: "undefined", Type: "undefined"}}

	got := jsInvalidArgList(params, boundaries)
	if got != "undefined, 1" {
		t.Fatalf("jsInvalidArgList() = %q", got)
	}
}

func TestJSTestBoundaryUsesThrowForErrorPath(t *testing.T) {
	fn := jsFuncInfo{
		Name:   "divide",
		Params: []jsParamInfo{{Name: "a"}, {Name: "b"}},
		Analysis: jsFuncAnalysis{
			ReturnType: "number",
			HasReturn:  true,
			Throws:     true,
			Boundaries: []jsBoundary{{Param: "b", Value: "0", Type: "number"}},
		},
	}

	code := genJestFuncTest(fn)
	if !strings.Contains(code, "it('should handle b = 0'") {
		t.Fatalf("missing boundary test:\n%s", code)
	}
	if !strings.Contains(code, "expect(() => divide(1, 0)).toThrow();") {
		t.Fatalf("boundary should assert throw, got:\n%s", code)
	}
	if strings.Contains(code, "const result = divide(1, 0)") {
		t.Fatalf("boundary should not call throwing input as normal result, got:\n%s", code)
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

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
