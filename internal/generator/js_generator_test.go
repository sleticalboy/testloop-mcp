package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sleticalboy/testloop-mcp/types"
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

func TestTreeSitterJS_ParsesTSParamsAndSkipsHelpers(t *testing.T) {
	source := []byte(`export function typed(text: string, count?: number, enabled = true, ...items: string[]) {
  return text;
}
function test(name: string) { return name; }
const destructured = ({ id } = {}) => id;
class Widget {
  constructor(name: string) {}
  static(value: string) { return value; }
  render(props?: Record<string, unknown>) { return props; }
}
`)
	funcs, classes, isESModule := parseJSWithTreeSitter(source, ".ts")
	if !isESModule {
		t.Fatal("export statement should mark source as ES module")
	}

	var typedFn *jsFuncInfo
	var destructuredFn *jsFuncInfo
	for i := range funcs {
		switch funcs[i].Name {
		case "typed":
			typedFn = &funcs[i]
		case "destructured":
			destructuredFn = &funcs[i]
		case "test":
			t.Fatalf("test helper should be skipped: %+v", funcs[i])
		}
	}
	if typedFn == nil {
		t.Fatalf("typed function not parsed: %+v", funcs)
	}
	if len(typedFn.Params) != 4 {
		t.Fatalf("typed params = %+v, want 4 params", typedFn.Params)
	}
	if typedFn.Params[0].Name != "text" || typedFn.Params[1].Name != "count" || !typedFn.Params[1].HasDefault ||
		typedFn.Params[2].Name != "enabled" ||
		typedFn.Params[3].Name != "...items" || typedFn.Params[3].IsRest {
		t.Fatalf("unexpected typed params: %+v", typedFn.Params)
	}
	if destructuredFn == nil || len(destructuredFn.Params) != 1 || destructuredFn.Params[0].Name != "id" || destructuredFn.Params[0].HasDefault {
		t.Fatalf("unexpected destructured params: %+v", destructuredFn)
	}

	if len(classes) != 1 || classes[0].Name != "Widget" {
		t.Fatalf("classes = %+v, want Widget", classes)
	}
	if len(classes[0].Methods) != 1 || classes[0].Methods[0].Name != "render" {
		t.Fatalf("constructor and keyword method should be skipped, got methods: %+v", classes[0].Methods)
	}
	if len(classes[0].Methods[0].Params) != 1 || classes[0].Methods[0].Params[0].Name != "props" || !classes[0].Methods[0].Params[0].HasDefault {
		t.Fatalf("unexpected render params: %+v", classes[0].Methods[0].Params)
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

func TestJestGeneratesExactAssertionForSimpleReturn(t *testing.T) {
	fn := jsFuncInfo{
		Name:   "add",
		Params: []jsParamInfo{{Name: "a"}, {Name: "b"}},
		Analysis: jsFuncAnalysis{
			ReturnType: "number",
			Returns:    []string{"a + b"},
			HasReturn:  true,
		},
	}

	code := genJestFuncTest(fn)
	if !strings.Contains(code, "const result = add(1, 2);") {
		t.Fatalf("expected semantic args, got:\n%s", code)
	}
	if !strings.Contains(code, "expect(result).toBe((1 + 2));") {
		t.Fatalf("expected exact assertion, got:\n%s", code)
	}
	if strings.Contains(code, "expect(typeof result).toBe('number');") {
		t.Fatalf("exact assertion should replace broad type assertion, got:\n%s", code)
	}
}

func TestJestExactAssertionUsesBoundaryValue(t *testing.T) {
	fn := jsFuncInfo{
		Name:   "normalize",
		Params: []jsParamInfo{{Name: "prefix"}, {Name: "text"}},
		Analysis: jsFuncAnalysis{
			ReturnType: "string",
			Returns:    []string{"prefix + text"},
			HasReturn:  true,
			Boundaries: []jsBoundary{{Param: "prefix", Value: "'>'", Type: "string"}},
		},
	}

	code := genJestFuncTest(fn)
	if !strings.Contains(code, "expect(result).toBe(('test' + 'test'));") {
		t.Fatalf("expected exact normal assertion, got:\n%s", code)
	}
	if !strings.Contains(code, "expect(result).toBe(('>' + 'test'));") {
		t.Fatalf("expected exact boundary assertion, got:\n%s", code)
	}
}

func TestJestExactAssertionUsesBranchReturnForBoundary(t *testing.T) {
	analysis := analyzeJSBody(`if (mode === 'short') {
  return prefix;
}
return prefix + text;`)
	fn := jsFuncInfo{
		Name:     "formatText",
		Params:   []jsParamInfo{{Name: "mode"}, {Name: "prefix"}, {Name: "text"}},
		Analysis: analysis,
	}

	code := genJestFuncTest(fn)
	if !strings.Contains(code, "const result = formatText('test', 'test', 'test');") {
		t.Fatalf("expected normal call, got:\n%s", code)
	}
	if !strings.Contains(code, "expect(result).toBe(('test' + 'test'));") {
		t.Fatalf("expected default-path assertion, got:\n%s", code)
	}
	if !strings.Contains(code, "const result = formatText('short', 'test', 'test');") {
		t.Fatalf("expected boundary call, got:\n%s", code)
	}
	if !strings.Contains(code, "expect(result).toBe(('test'));") {
		t.Fatalf("expected branch assertion, got:\n%s", code)
	}
	if !strings.Contains(code, "it('should handle mode = \\'short\\''") {
		t.Fatalf("expected escaped boundary test name, got:\n%s", code)
	}
}

func TestGenerateJestTestsForCoverageTaskUsesTaskNameAndInputs(t *testing.T) {
	src := `function add(a, b) {
  return a + b;
}

function sub(a, b) {
  return a - b;
}

module.exports = { add, sub };
`
	srcPath := filepath.Join(t.TempDir(), "calc.js")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "jest-1",
		Framework:       "jest",
		Target:          "add",
		LineRange:       "2-2",
		GapType:         "return_path",
		TestName:        "covers add zero left operand",
		SuggestedInputs: []string{"构造满足条件 `a === 0` 的输入"},
		AssertionFocus:  []string{"断言未覆盖返回路径的具体结果"},
	}

	code, err := GenerateJestTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateJestTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"it('covers add zero left operand'",
		"coverage task: jest-1 | lines 2-2 | 断言未覆盖返回路径的具体结果 | 构造满足条件 `a === 0` 的输入",
		"const result = add(0, 2);",
		"expect(result).toBe((0 + 2));",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "describe('sub'") || strings.Contains(code, "sub(") {
		t.Fatalf("task-aware generation should only target add:\n%s", code)
	}
}

func TestGenerateMochaCoverageTaskUsesChaiSyncErrorAssertion(t *testing.T) {
	src := `function divide(a, b) {
  if (b === 0) throw new Error('zero')
  return a / b
}

module.exports = { divide };
`
	srcPath := filepath.Join(t.TempDir(), "calc.js")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "mocha-error-1",
		Framework:       "mocha",
		Target:          "divide",
		LineRange:       "2-2",
		GapType:         "error_path",
		TestName:        "covers divide zero error",
		SuggestedInputs: []string{"构造满足条件 `b === 0` 的输入"},
	}

	code, err := GenerateJestTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateJestTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"const { expect } = require('chai');",
		"it('covers divide zero error'",
		"expect(() => divide(1, 0)).to.throw();",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, forbidden := range []string{"toThrow()", "rejects.toThrow()"} {
		if strings.Contains(code, forbidden) {
			t.Fatalf("Mocha error assertion should not contain %q:\n%s", forbidden, code)
		}
	}
}

func TestGenerateMochaCoverageTaskUsesChaiAsyncErrorAssertion(t *testing.T) {
	src := `async function fetchData(url) {
  if (url === undefined) throw new Error('missing')
  return { ok: true }
}

module.exports = { fetchData };
`
	srcPath := filepath.Join(t.TempDir(), "service.js")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "mocha-error-2",
		Framework:       "mocha",
		Target:          "fetchData",
		LineRange:       "2-2",
		GapType:         "error_path",
		TestName:        "covers fetchData missing url",
		SuggestedInputs: []string{"构造满足条件 `url === undefined` 的输入"},
	}

	code, err := GenerateJestTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateJestTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"const { expect } = require('chai');",
		"let caughtError;",
		"try {",
		"await fetchData(undefined);",
		"caughtError = err;",
		"expect(caughtError).to.exist;",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "rejects.toThrow()") || strings.Contains(code, "toThrow()") {
		t.Fatalf("Mocha async error assertion should not use Jest matchers:\n%s", code)
	}
}

func TestGenerateMochaCoverageTaskUsesChaiClassSyncErrorAssertion(t *testing.T) {
	src := `class Widget {
  save(payload) {
    if (payload === null) throw new Error('missing payload')
    return true
  }

  load(mode) {
    return mode
  }
}

module.exports = { Widget };
`
	srcPath := filepath.Join(t.TempDir(), "widget.js")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "mocha-class-error-1",
		Framework:       "mocha",
		Target:          "Widget.save",
		LineRange:       "3-3",
		GapType:         "error_path",
		TestName:        "covers widget save missing payload",
		SuggestedInputs: []string{"构造满足条件 `payload === null` 的输入"},
	}

	code, err := GenerateJestTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateJestTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"const { expect } = require('chai');",
		"const { Widget } = require('./widget');",
		"describe('Widget'",
		"describe('save'",
		"it('covers widget save missing payload'",
		"const instance = new Widget();",
		"expect(() => instance.save(null)).to.throw();",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, forbidden := range []string{"describe('load'", "toThrow()", "rejects.toThrow()"} {
		if strings.Contains(code, forbidden) {
			t.Fatalf("Mocha class sync error assertion should not contain %q:\n%s", forbidden, code)
		}
	}
}

func TestGenerateMochaCoverageTaskUsesChaiClassAsyncErrorAssertion(t *testing.T) {
	src := `class Widget {
  async load(url) {
    if (url === undefined) throw new Error('missing url')
    return { ok: true }
  }

  save(payload) {
    return payload
  }
}

module.exports = { Widget };
`
	srcPath := filepath.Join(t.TempDir(), "widget.js")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "mocha-class-error-2",
		Framework:       "mocha",
		Target:          "Widget.load",
		LineRange:       "3-3",
		GapType:         "error_path",
		TestName:        "covers widget load missing url",
		SuggestedInputs: []string{"构造满足条件 `url === undefined` 的输入"},
	}

	code, err := GenerateJestTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateJestTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"const { expect } = require('chai');",
		"const { Widget } = require('./widget');",
		"describe('Widget'",
		"describe('load'",
		"it('covers widget load missing url', async () => {",
		"const instance = new Widget();",
		"let caughtError;",
		"await instance.load(undefined);",
		"caughtError = err;",
		"expect(caughtError).to.exist;",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, forbidden := range []string{"describe('save'", "toThrow()", "rejects.toThrow()"} {
		if strings.Contains(code, forbidden) {
			t.Fatalf("Mocha class async error assertion should not contain %q:\n%s", forbidden, code)
		}
	}
}

func TestGenerateMochaCoverageTaskUsesChaiESMFunctionErrorAssertion(t *testing.T) {
	src := `export function divide(a, b) {
  if (b === 0) throw new Error('zero')
  return a / b
}

export function add(a, b) {
  return a + b
}
`
	srcPath := filepath.Join(t.TempDir(), "calc.js")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "mocha-esm-error-1",
		Framework:       "mocha",
		Target:          "divide",
		LineRange:       "2-2",
		GapType:         "error_path",
		TestName:        "covers esm divide zero error",
		SuggestedInputs: []string{"构造满足条件 `b === 0` 的输入"},
	}

	code, err := GenerateJestTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateJestTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"import { expect } from 'chai';",
		"import { divide } from './calc';",
		"it('covers esm divide zero error'",
		"expect(() => divide(1, 0)).to.throw();",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, forbidden := range []string{"require('chai')", "require('./calc')", "describe('add'", "toThrow()", "rejects.toThrow()"} {
		if strings.Contains(code, forbidden) {
			t.Fatalf("Mocha ESM function error assertion should not contain %q:\n%s", forbidden, code)
		}
	}
}

func TestGenerateMochaCoverageTaskUsesChaiESMClassAsyncErrorAssertion(t *testing.T) {
	src := `export class Widget {
  async load(url) {
    if (url === undefined) throw new Error('missing url')
    return { ok: true }
  }

  save(payload) {
    return payload
  }
}
`
	srcPath := filepath.Join(t.TempDir(), "widget.js")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "mocha-esm-class-error-1",
		Framework:       "mocha",
		Target:          "Widget.load",
		LineRange:       "3-3",
		GapType:         "error_path",
		TestName:        "covers esm widget load missing url",
		SuggestedInputs: []string{"构造满足条件 `url === undefined` 的输入"},
	}

	code, err := GenerateJestTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateJestTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"import { expect } from 'chai';",
		"import { Widget } from './widget';",
		"describe('Widget'",
		"describe('load'",
		"it('covers esm widget load missing url', async () => {",
		"const instance = new Widget();",
		"let caughtError;",
		"await instance.load(undefined);",
		"caughtError = err;",
		"expect(caughtError).to.exist;",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, forbidden := range []string{"require('chai')", "require('./widget')", "describe('save'", "toThrow()", "rejects.toThrow()"} {
		if strings.Contains(code, forbidden) {
			t.Fatalf("Mocha ESM class async error assertion should not contain %q:\n%s", forbidden, code)
		}
	}
}

func TestGenerateMochaCoverageTaskUsesChaiESMFunctionReturnAssertion(t *testing.T) {
	src := `export function add(a, b) {
  return a + b
}

export function sub(a, b) {
  return a - b
}
`
	srcPath := filepath.Join(t.TempDir(), "calc.js")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "mocha-esm-return-1",
		Framework:       "mocha",
		Target:          "add",
		LineRange:       "2-2",
		GapType:         "return_path",
		TestName:        "covers esm add zero left operand",
		SuggestedInputs: []string{"构造满足条件 `a === 0` 的输入"},
		AssertionFocus:  []string{"断言 ESM 返回路径的具体结果"},
	}

	code, err := GenerateJestTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateJestTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"import { expect } from 'chai';",
		"import { add } from './calc';",
		"it('covers esm add zero left operand'",
		"coverage task: mocha-esm-return-1 | lines 2-2 | 断言 ESM 返回路径的具体结果 | 构造满足条件 `a === 0` 的输入",
		"const result = add(0, 2);",
		"expect(result).to.equal((0 + 2));",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, forbidden := range []string{"require('chai')", "require('./calc')", "describe('sub'", "toBe(", "toThrow()", "rejects.toThrow()"} {
		if strings.Contains(code, forbidden) {
			t.Fatalf("Mocha ESM function return assertion should not contain %q:\n%s", forbidden, code)
		}
	}
}

func TestGenerateMochaCoverageTaskUsesChaiESMClassBranchAssertion(t *testing.T) {
	src := `export class Widget {
  load(mode, count) {
    if (mode === 'short') return count
    return count + 1
  }

  save(payload) {
    return payload
  }
}
`
	srcPath := filepath.Join(t.TempDir(), "widget.js")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "mocha-esm-class-branch-1",
		Framework:       "mocha",
		Target:          "Widget.load",
		LineRange:       "3-3",
		GapType:         "branch",
		TestName:        "covers esm widget short mode",
		SuggestedInputs: []string{"构造满足条件 `mode === 'short'` 的输入"},
		AssertionFocus:  []string{"断言 ESM class 分支返回值"},
	}

	code, err := GenerateJestTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateJestTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"import { expect } from 'chai';",
		"import { Widget } from './widget';",
		"describe('Widget'",
		"describe('load'",
		"it('covers esm widget short mode'",
		"coverage task: mocha-esm-class-branch-1 | lines 3-3 | 断言 ESM class 分支返回值 | 构造满足条件 `mode === 'short'` 的输入",
		"const instance = new Widget();",
		"const result = instance.load('short', 1);",
		"expect(result).to.equal((1));",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, forbidden := range []string{"require('chai')", "require('./widget')", "describe('save'", "toBe(", "toThrow()", "rejects.toThrow()"} {
		if strings.Contains(code, forbidden) {
			t.Fatalf("Mocha ESM class branch assertion should not contain %q:\n%s", forbidden, code)
		}
	}
}

func TestFilterJSTargetsForCoverageTaskBranches(t *testing.T) {
	funcs := []jsFuncInfo{{Name: "add"}, {Name: "sub"}}
	classes := []jsClassInfo{
		{
			Name: "Calculator",
			Methods: []jsFuncInfo{
				{Name: "add", ClassName: "Calculator", IsMethod: true},
				{Name: "divide", ClassName: "Calculator", IsMethod: true},
			},
		},
	}

	gotFuncs, gotClasses := filterJSTargetsForCoverageTask(funcs, classes, &types.CoverageTestTask{})
	if len(gotFuncs) != 2 || len(gotClasses) != 1 {
		t.Fatalf("empty target should keep all targets: funcs=%+v classes=%+v", gotFuncs, gotClasses)
	}

	gotFuncs, gotClasses = filterJSTargetsForCoverageTask(funcs, classes, &types.CoverageTestTask{Target: "add"})
	if len(gotFuncs) != 1 || gotFuncs[0].Name != "add" || len(gotClasses) != 1 ||
		len(gotClasses[0].Methods) != 1 || gotClasses[0].Methods[0].Name != "add" {
		t.Fatalf("function target filtered incorrectly: funcs=%+v classes=%+v", gotFuncs, gotClasses)
	}

	gotFuncs, gotClasses = filterJSTargetsForCoverageTask(funcs, classes, &types.CoverageTestTask{Target: "Calculator"})
	if len(gotFuncs) != 0 || len(gotClasses) != 1 || len(gotClasses[0].Methods) != 2 {
		t.Fatalf("class target should keep whole class: funcs=%+v classes=%+v", gotFuncs, gotClasses)
	}

	gotFuncs, gotClasses = filterJSTargetsForCoverageTask(funcs, classes, &types.CoverageTestTask{Target: "Calculator.divide"})
	if len(gotFuncs) != 0 || len(gotClasses) != 1 || len(gotClasses[0].Methods) != 1 || gotClasses[0].Methods[0].Name != "divide" {
		t.Fatalf("method target filtered incorrectly: funcs=%+v classes=%+v", gotFuncs, gotClasses)
	}

	gotFuncs, gotClasses = filterJSTargetsForCoverageTask(funcs, classes, &types.CoverageTestTask{Target: "missing"})
	if len(gotFuncs) != 2 || len(gotClasses) != 1 {
		t.Fatalf("missing target should fall back to all targets: funcs=%+v classes=%+v", gotFuncs, gotClasses)
	}
}

func TestJSAnalysisReturnTypeForAssert(t *testing.T) {
	tests := map[string]string{
		"number":    "number",
		"string":    "string",
		"boolean":   "boolean",
		"array":     "object",
		"object":    "object",
		"null":      "object",
		"undefined": "",
		"unknown":   "",
	}
	for typ, want := range tests {
		t.Run(typ, func(t *testing.T) {
			if got := (jsFuncAnalysis{ReturnType: typ}).returnTypeForAssert(); got != want {
				t.Fatalf("returnTypeForAssert(%q) = %q, want %q", typ, got, want)
			}
		})
	}
}

func TestJestClassCoverageTaskCoversNormalAndErrorMethods(t *testing.T) {
	task := types.CoverageTestTask{
		ID:              "jest-class-1",
		Target:          "Widget.load",
		LineRange:       "10-12",
		GapType:         "branch",
		TestName:        "covers widget load",
		SuggestedInputs: []string{"构造满足条件 `mode === 'short'` 的输入"},
		AssertionFocus:  []string{"断言 class 方法分支"},
	}
	cls := jsClassInfo{
		Name: "Widget",
		Methods: []jsFuncInfo{
			{
				Name:   "load",
				Params: []jsParamInfo{{Name: "mode"}, {Name: "count"}},
				Analysis: jsFuncAnalysis{
					ReturnType: "number",
					Returns:    []string{"count + 1"},
					HasReturn:  true,
					Boundaries: []jsBoundary{{Param: "mode", Value: "'short'", Type: "string", ReturnExpr: "count"}},
				},
			},
			{
				Name:    "save",
				IsAsync: true,
				Params:  []jsParamInfo{{Name: "payload"}},
				Analysis: jsFuncAnalysis{
					Throws: true,
				},
			},
		},
	}

	code := genJestClassTestForCoverageTask(cls, &task)
	for _, want := range []string{
		"describe('Widget'",
		"it('covers widget load'",
		"coverage task: jest-class-1 | lines 10-12 | 断言 class 方法分支 | 构造满足条件 `mode === 'short'` 的输入",
		"const instance = new Widget();",
		"const result = instance.load('short', 1);",
		"expect(result).toBe((1));",
		"await expect(instance.save({})).rejects.toThrow();",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in class task output:\n%s", want, code)
		}
	}
}

func TestJestFuncCoverageTaskCoversAsyncErrorFallbackName(t *testing.T) {
	fn := jsFuncInfo{
		Name:    "fetchData",
		IsAsync: true,
		Params:  []jsParamInfo{{Name: "url"}},
		Analysis: jsFuncAnalysis{
			ReturnType: "object",
			HasReturn:  true,
			Throws:     true,
			Boundaries: []jsBoundary{{Param: "url", Value: "undefined", Type: "undefined"}},
		},
	}
	task := types.CoverageTestTask{
		ID:              "jest-error-1",
		GapType:         "error_path",
		LineRange:       "4-6",
		SuggestedInputs: []string{"构造满足条件 `url === undefined` 的输入"},
	}

	code := genJestFuncTestForCoverageTask(fn, &task)
	for _, want := range []string{
		"describe('fetchData'",
		"it('should cover fetchData coverage gap', async () => {",
		"coverage task: jest-error-1 | lines 4-6 | 构造满足条件 `url === undefined` 的输入",
		"await expect(fetchData(undefined)).rejects.toThrow();",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in coverage task output:\n%s", want, code)
		}
	}
	if strings.Contains(code, "const result =") {
		t.Fatalf("error-path coverage task should not assign result:\n%s", code)
	}
}

func TestJestAssertionAndDedupeCompatHelpers(t *testing.T) {
	if got := genJSResultAssertion(jsFuncAnalysis{}, "  "); got != "  // void function, verify no exception\n" {
		t.Fatalf("genJSResultAssertion() = %q", got)
	}
	expr, ok := jsExpectedReturnExpr(jsFuncAnalysis{Returns: []string{"a + b"}}, []jsParamInfo{{Name: "a"}, {Name: "b"}}, nil)
	if !ok || expr != "(1 + 2)" {
		t.Fatalf("jsExpectedReturnExpr() = %q, %v", expr, ok)
	}
	expr, ok = jsExpectedReturnExprWithValues(
		jsFuncAnalysis{Returns: []string{"url + suffix;"}},
		[]jsParamInfo{{Name: "url"}, {Name: "suffix"}},
		nil,
		map[string]string{"url": "'https://example.com'", "suffix": "'/v1'"},
	)
	if !ok || expr != "('https://example.com' + '/v1')" {
		t.Fatalf("jsExpectedReturnExprWithValues() = %q, %v", expr, ok)
	}
	if expr, ok = jsExpectedReturnExpr(jsFuncAnalysis{Returns: []string{"unknown + 1"}}, nil, nil); ok || expr != "" {
		t.Fatalf("unsafe jsExpectedReturnExpr() = %q, %v", expr, ok)
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
		if got := isNumericLiteral(input); got != want {
			t.Fatalf("isNumericLiteral(%q) = %v, want %v", input, got, want)
		}
	}
	for input, want := range map[string]bool{
		"a + b":        true,
		"new Widget()": false,
		"value => 1":   false,
		"items[0]":     false,
		"":             false,
	} {
		if got := jsReturnExprIsSafe(input); got != want {
			t.Fatalf("jsReturnExprIsSafe(%q) = %v, want %v", input, got, want)
		}
	}
	for input, want := range map[string]string{
		"null":             "null",
		"undefined":        "undefined",
		"true":             "boolean",
		"'ok'":             "string",
		"`ok`":             "string",
		"12.5":             "number",
		"[1]":              "array",
		"{ ok: true }":     "object",
		"JSON.parse(raw)":  "object",
		"response.json()":  "object",
		"a * b":            "number",
		"enabled && ready": "boolean",
		"name + 'suffix'":  "string",
		"customValue":      "unknown",
	} {
		if got := inferJSReturnType([][]string{{"", input}}); got != want {
			t.Fatalf("inferJSReturnType(%q) = %q, want %q", input, got, want)
		}
	}
	if got := jsPlaceholderArgList([]jsParamInfo{{Name: "args", IsRest: true}, {Name: "value"}}); got != "[], undefined" {
		t.Fatalf("jsPlaceholderArgList() = %q", got)
	}
	if got := jsArgListWithValues([]jsParamInfo{{Name: "enabled"}, {Name: "title"}}, map[string]string{"title": "'custom'"}); got != "true, 'custom'" {
		t.Fatalf("jsArgListWithValues() = %q", got)
	}
	boundaries := []jsBoundary{{Param: "mode", Value: "'short'"}, {Param: "enabled", Value: "false"}}
	task := &types.CoverageTestTask{SuggestedInputs: []string{"构造满足条件 `enabled == false` 的输入"}}
	if got := jsBoundaryForCoverageTask(boundaries, task); got == nil || got.Param != "enabled" {
		t.Fatalf("jsBoundaryForCoverageTask(exact) = %+v", got)
	}
	if got := jsBoundaryForCoverageTask([]jsBoundary{{Param: "mode", Value: "'short'"}}, &types.CoverageTestTask{GapType: "error_path"}); got == nil || got.Param != "mode" {
		t.Fatalf("jsBoundaryForCoverageTask(fallback) = %+v", got)
	}
	if got := jsBoundaryForCoverageTask(boundaries, nil); got != nil {
		t.Fatalf("jsBoundaryForCoverageTask(nil) = %+v", got)
	}
	if got := baseName(`C:\tmp\calc.test.js`); got != "calc.test.js" {
		t.Fatalf("baseName() = %q", got)
	}
	if got := stripExt("calc.test.js"); got != "calc.test" {
		t.Fatalf("stripExt(with ext) = %q", got)
	}
	if got := stripExt(".env"); got != ".env" {
		t.Fatalf("stripExt(dotfile) = %q", got)
	}
	if !isJSKeyword("await") || isJSKeyword("businessValue") {
		t.Fatalf("isJSKeyword returned unexpected result")
	}

	funcs := dedupJSFuncs([]jsFuncInfo{
		{Name: "load"},
		{Name: "load"},
		{Name: "load", IsMethod: true, ClassName: "Widget"},
		{Name: "load", IsMethod: true, ClassName: "Widget"},
		{Name: "load", IsMethod: true, ClassName: "Other"},
	})
	if len(funcs) != 3 {
		t.Fatalf("dedupJSFuncs() kept %d funcs: %+v", len(funcs), funcs)
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
