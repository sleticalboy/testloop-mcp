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

func TestGenerateJavaScriptTestsWithFramework(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "calc.js")
	src := `function add(a, b) {
  return a + b;
}

module.exports = { add };
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	tests := []struct {
		name      string
		framework string
		want      []string
		forbidden []string
	}{
		{
			name:      "vitest keeps jest style matchers",
			framework: "vitest",
			want: []string{
				"const { add } = require('./calc');",
				"expect(result).toBe((1 + 2));",
			},
			forbidden: []string{"require('chai')", "to.equal((1 + 2))"},
		},
		{
			name:      "mocha uses chai assertions",
			framework: "mocha",
			want: []string{
				"const { expect } = require('chai');",
				"const { add } = require('./calc');",
				"expect(result).to.equal((1 + 2));",
			},
			forbidden: []string{"toBe((1 + 2))", "rejects.toThrow()"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := GenerateJavaScriptTestsWithFramework(srcPath, tt.framework)
			if err != nil {
				t.Fatalf("GenerateJavaScriptTestsWithFramework() error = %v", err)
			}
			assertGeneratedJS(t, code, tt.want, tt.forbidden)
		})
	}
}

func TestGenerateJavaScriptTestsWithFrameworkESMVitestImportsAPI(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "calc.ts")
	src := `export function add(a: number, b: number): number {
  return a + b;
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	code, err := GenerateJavaScriptTestsWithFramework(srcPath, "vitest")
	if err != nil {
		t.Fatalf("GenerateJavaScriptTestsWithFramework() error = %v", err)
	}

	assertGeneratedJS(t, code, []string{
		"import { describe, it, expect } from 'vitest';",
		"import { add } from './calc';",
		"expect(result).toBe((1 + 2));",
	}, []string{
		"from 'chai'",
		"require('vitest')",
		"to.equal((1 + 2))",
	})
}

func TestGenerateJavaScriptTestsESMImportPathFollowsTSConfig(t *testing.T) {
	tests := []struct {
		name       string
		tsconfig   string
		wantImport string
		forbidden  string
	}{
		{
			name: "nodenext uses emitted js extension",
			tsconfig: `{
  "compilerOptions": {
    "module": "NodeNext",
    "moduleResolution": "NodeNext"
  }
}`,
			wantImport: "import { add } from './sum.js';",
			forbidden:  "import { add } from './sum';",
		},
		{
			name: "jsonc node16 uses emitted js extension",
			tsconfig: `{
  // jsonc comments are common in tsconfig files
  "compilerOptions": {
    "moduleResolution": "node16"
  }
}`,
			wantImport: "import { add } from './sum.js';",
			forbidden:  "import { add } from './sum';",
		},
		{
			name: "bundler keeps extensionless import",
			tsconfig: `{
  "compilerOptions": {
    "moduleResolution": "bundler"
  }
}`,
			wantImport: "import { add } from './sum';",
			forbidden:  "import { add } from './sum.js';",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcDir := filepath.Join(dir, "src")
			if err := os.MkdirAll(srcDir, 0o755); err != nil {
				t.Fatalf("create source dir: %v", err)
			}
			if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(tt.tsconfig+"\n"), 0o644); err != nil {
				t.Fatalf("write tsconfig: %v", err)
			}
			srcPath := filepath.Join(srcDir, "sum.ts")
			src := `export function add(a: number, b: number): number {
  return a + b;
}
`
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			code, err := GenerateJavaScriptTestsWithFramework(srcPath, "vitest")
			if err != nil {
				t.Fatalf("GenerateJavaScriptTestsWithFramework() error = %v", err)
			}
			assertGeneratedJS(t, code, []string{tt.wantImport}, []string{tt.forbidden})
		})
	}
}

func TestGenerateJavaScriptTestsAsyncResponseJSON(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "api.ts")
	src := `export async function parseUser(response: Response): Promise<{ ok: boolean }> {
  return await response.json();
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	code, err := GenerateJavaScriptTestsWithFramework(srcPath, "vitest")
	if err != nil {
		t.Fatalf("GenerateJavaScriptTestsWithFramework() error = %v", err)
	}

	assertGeneratedJS(t, code, []string{
		"import { describe, it, expect } from 'vitest';",
		"const result = await parseUser({ json: async () => ({ ok: true }) });",
		"expect(result).toEqual({ ok: true });",
	}, []string{
		"expect(typeof result).toBe('object')",
		"expect(result).not.toBeNull()",
	})
}

func TestGenerateJavaScriptTestsResponseJSONReturnTypePayload(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "api.ts")
	src := `export async function parseUser(response: Response): Promise<{ id: number; name: string; active: boolean }> {
  return await response.json();
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	code, err := GenerateJavaScriptTestsWithFramework(srcPath, "vitest")
	if err != nil {
		t.Fatalf("GenerateJavaScriptTestsWithFramework() error = %v", err)
	}

	assertGeneratedJS(t, code, []string{
		"const result = await parseUser({ json: async () => ({ id: 1, name: 'test', active: true }) });",
		"expect(result).toEqual({ id: 1, name: 'test', active: true });",
	}, []string{
		"{ ok: true }",
		"expect(typeof result).toBe('object')",
	})
}

func TestGenerateJavaScriptTestsInjectedClientCall(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "users.ts")
	src := `export async function loadUser(client: { get(path: string): Promise<{ ok: boolean }> }): Promise<{ ok: boolean }> {
  return await client.get('/users/1');
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	code, err := GenerateJavaScriptTestsWithFramework(srcPath, "vitest")
	if err != nil {
		t.Fatalf("GenerateJavaScriptTestsWithFramework() error = %v", err)
	}

	assertGeneratedJS(t, code, []string{
		"import { describe, it, expect } from 'vitest';",
		"const client = {",
		"getCalls: [],",
		"get: async (...args) => {",
		"client.getCalls.push(args);",
		"const result = await loadUser(client);",
		"expect(result).toEqual({ ok: true });",
		"expect(client.getCalls).toEqual([['/users/1']]);",
	}, []string{
		"fetch: async",
		"request: async",
		"expect(typeof result).toBe('object')",
		"expect(result).not.toBeNull()",
	})
}

func TestGenerateJavaScriptTestsInjectedClientReturnTypePayload(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "users.ts")
	src := `export async function loadUser(client: { get(path: string): Promise<{ id: number; email: string }> }): Promise<{ id: number; email: string }> {
  return await client.get('/users/1');
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	code, err := GenerateJavaScriptTestsWithFramework(srcPath, "vitest")
	if err != nil {
		t.Fatalf("GenerateJavaScriptTestsWithFramework() error = %v", err)
	}

	assertGeneratedJS(t, code, []string{
		"const client = {",
		"return { id: 1, email: 'user@example.com' };",
		"const result = await loadUser(client);",
		"expect(result).toEqual({ id: 1, email: 'user@example.com' });",
		"expect(client.getCalls).toEqual([['/users/1']]);",
	}, []string{
		"{ ok: true }",
		"expect(typeof result).toBe('object')",
	})
}

func TestGenerateJavaScriptTestsNamedReturnTypePayloads(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "api.ts")
	src := `interface User {
  userId: number
  email: string
  status: 'active' | 'disabled'
  createdAt: string
  displayName?: string | null
  manager?: User | null
}

type Profile = {
  title: string
  active: boolean
  avatarUrl: string
  owner?: User | null
}

type Meta = {
  total: number
  nextUrl?: string | null
}

type Users = readonly User[]
type MaybeUsers = ReadonlyArray<User | null>
type UserTuple = readonly [user: User, meta?: Meta]

export async function parseUser(response: Response): Promise<User> {
  return await response.json();
}

export async function parseReadonlyUser(response: Response): Promise<Readonly<User>> {
  return await response.json();
}

export async function loadProfile(client: { get(path: string): Promise<Profile> }): Promise<Profile> {
  return await client.get('/profile');
}

export async function loadPartialProfile(client: { get(path: string): Promise<Partial<Profile>> }): Promise<Partial<Profile>> {
  return await client.get('/profile/partial');
}

export async function listUsers(response: Response): Promise<Users> {
  return await response.json();
}

export async function searchUsers(client: { fetch(path: string): Promise<MaybeUsers> }): Promise<MaybeUsers> {
  return await client.fetch('/users');
}

export async function loadUserTuple(response: Response): Promise<UserTuple> {
  return await response.json();
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	code, err := GenerateJavaScriptTestsWithFramework(srcPath, "vitest")
	if err != nil {
		t.Fatalf("GenerateJavaScriptTestsWithFramework() error = %v", err)
	}

	assertGeneratedJS(t, code, []string{
		"const result = await parseUser({ json: async () => ({ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }) });",
		"expect(result).toEqual({ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} });",
		"const result = await parseReadonlyUser({ json: async () => ({ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }) });",
		"expect(result).toEqual({ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} });",
		"return { title: 'test', active: true, avatarUrl: 'https://example.com', owner: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} } };",
		"const result = await loadProfile(client);",
		"expect(result).toEqual({ title: 'test', active: true, avatarUrl: 'https://example.com', owner: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} } });",
		"expect(client.getCalls).toEqual([['/profile']]);",
		"const result = await loadPartialProfile(client);",
		"expect(result).toEqual({ title: 'test', active: true, avatarUrl: 'https://example.com', owner: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} } });",
		"expect(client.getCalls).toEqual([['/profile/partial']]);",
		"const result = await listUsers({ json: async () => ([{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }]) });",
		"expect(result).toEqual([{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }]);",
		"return [{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }];",
		"const result = await searchUsers(client);",
		"expect(result).toEqual([{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }]);",
		"expect(client.fetchCalls).toEqual([['/users']]);",
		"const result = await loadUserTuple({ json: async () => ([{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }, { total: 1, nextUrl: 'https://example.com' }]) });",
		"expect(result).toEqual([{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }, { total: 1, nextUrl: 'https://example.com' }]);",
	}, []string{
		"{ ok: true }",
		"expect(typeof result).toBe('object')",
	})
}

func TestGenerateJavaScriptCoverageTaskESMImportPathFollowsTSConfig(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(`{
  "compilerOptions": {
    "module": "NodeNext"
  }
}
`), 0o644); err != nil {
		t.Fatalf("write tsconfig: %v", err)
	}
	srcPath := filepath.Join(srcDir, "sum.ts")
	src := `export function add(a: number, b: number): number {
  if (a === 0) return b;
  return a + b;
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	code, err := GenerateJavaScriptTestsForCoverageTask(srcPath, &types.CoverageTestTask{
		ID:              "vitest-nodenext-1",
		Framework:       "vitest",
		Target:          "add",
		LineRange:       "2-2",
		GapType:         "branch",
		TestName:        "covers add zero branch",
		SuggestedInputs: []string{"构造满足条件 `a === 0` 的输入"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaScriptTestsForCoverageTask() error = %v", err)
	}

	assertGeneratedJS(t, code, []string{
		"import { describe, it, expect } from 'vitest';",
		"import { add } from './sum.js';",
		"expect(result).toBe((2));",
	}, []string{
		"import { add } from './sum';",
	})
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
	source := []byte(`export function typed(text: string, count?: number, enabled = true, ...items: string[]): Promise<Array<{ id: number; name: string }>> {
  return text;
}
function test(name: string) { return name; }
const destructured = ({ id } = {}) => id;
class Widget {
  constructor(name: string) {}
  static(value: string) { return value; }
  render(props?: Record<string, unknown>): { ok: boolean } { return props; }
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
	if typedFn.Analysis.ReturnTypeExpr != "Promise<Array<{ id: number; name: string }>>" {
		t.Fatalf("typed return type = %q", typedFn.Analysis.ReturnTypeExpr)
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
	if classes[0].Methods[0].Analysis.ReturnTypeExpr != "{ ok: boolean }" {
		t.Fatalf("render return type = %q", classes[0].Methods[0].Analysis.ReturnTypeExpr)
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

func TestGenerateJestVitestCoverageTaskMatrixAssertions(t *testing.T) {
	tests := []struct {
		name      string
		fileName  string
		source    string
		task      types.CoverageTestTask
		wants     []string
		forbidden []string
	}{
		{
			name:     "jest commonjs function return",
			fileName: "calc.js",
			source: `function add(a, b) {
  return a + b;
}

function sub(a, b) {
  return a - b;
}

module.exports = { add, sub };
`,
			task: types.CoverageTestTask{
				ID:              "jest-1",
				Framework:       "jest",
				Target:          "add",
				LineRange:       "2-2",
				GapType:         "return_path",
				TestName:        "covers add zero left operand",
				SuggestedInputs: []string{"构造满足条件 `a === 0` 的输入"},
				AssertionFocus:  []string{"断言未覆盖返回路径的具体结果"},
			},
			wants: []string{
				"const { add } = require('./calc');",
				"it('covers add zero left operand'",
				"coverage task: jest-1 | lines 2-2 | 断言未覆盖返回路径的具体结果 | 构造满足条件 `a === 0` 的输入",
				"const result = add(0, 2);",
				"expect(result).toBe((0 + 2));",
			},
			forbidden: []string{"describe('sub'", "sub(", "to.equal(", "require('chai')"},
		},
		{
			name:     "vitest typescript function branch",
			fileName: "sum.ts",
			source: `export function add(a: number, b: number): number {
  if (a === 0) return b
  return a + b
}

export function sub(a: number, b: number): number {
  return a - b
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-1",
				Framework:       "vitest",
				Target:          "add",
				LineRange:       "2-2",
				GapType:         "branch",
				TestName:        "covers vitest add zero branch",
				SuggestedInputs: []string{"构造满足条件 `a === 0` 的输入"},
				AssertionFocus:  []string{"断言 Vitest TypeScript 分支返回值"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { add } from './sum';",
				"it('covers vitest add zero branch'",
				"coverage task: vitest-1 | lines 2-2 | 断言 Vitest TypeScript 分支返回值 | 构造满足条件 `a === 0` 的输入",
				"const result = add(0, 2);",
				"expect(result).toBe((2));",
			},
			forbidden: []string{"describe('sub'", "sub(", "to.equal(", "require('chai')"},
		},
		{
			name:     "vitest typescript function async error",
			fileName: "service.ts",
			source: `export async function fetchData(url?: string): Promise<{ ok: boolean }> {
  if (url === undefined) throw new Error('missing url')
  return { ok: true }
}

export function status(): string {
  return 'ok'
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-error-1",
				Framework:       "vitest",
				Target:          "fetchData",
				LineRange:       "2-2",
				GapType:         "error_path",
				TestName:        "covers vitest fetchData missing url",
				SuggestedInputs: []string{"构造满足条件 `url === undefined` 的输入"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { fetchData } from './service';",
				"it('covers vitest fetchData missing url', async () => {",
				"coverage task: vitest-error-1 | lines 2-2 | 构造满足条件 `url === undefined` 的输入",
				"await expect(fetchData(undefined)).rejects.toThrow();",
			},
			forbidden: []string{"describe('status'", "status(", "to.equal(", "require('chai')", "caughtError"},
		},
		{
			name:     "vitest typescript object branch",
			fileName: "summary.ts",
			source: `export function summarize(mode: string, count: number): { mode: string, count: number } {
  if (mode === 'short') return { mode, count }
  return { mode, count: count + 1 }
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-object-1",
				Framework:       "vitest",
				Target:          "summarize",
				LineRange:       "2-2",
				GapType:         "branch",
				TestName:        "covers vitest summarize short branch",
				SuggestedInputs: []string{"构造满足条件 `mode === 'short'` 的输入", "设置 `count = 1`"},
				AssertionFocus:  []string{"断言对象返回值"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { summarize } from './summary';",
				"const result = summarize('short', 1);",
				"expect(result).toEqual({ mode: 'short', count: 1 });",
			},
			forbidden: []string{"to.equal(", "require('chai')", "expect(typeof result).toBe('object')"},
		},
		{
			name:     "vitest typescript response json",
			fileName: "api.ts",
			source: `export async function parseUser(response: Response): Promise<{ ok: boolean }> {
  return await response.json()
}
`,
			task: types.CoverageTestTask{
				ID:             "vitest-json-1",
				Framework:      "vitest",
				Target:         "parseUser",
				LineRange:      "2-2",
				GapType:        "return_path",
				TestName:       "covers vitest parseUser json response",
				AssertionFocus: []string{"断言 JSON 响应结构"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { parseUser } from './api';",
				"const result = await parseUser({ json: async () => ({ ok: true }) });",
				"expect(result).toEqual({ ok: true });",
			},
			forbidden: []string{"to.equal(", "require('chai')", "expect(typeof result).toBe('object')"},
		},
		{
			name:     "vitest typescript named response json",
			fileName: "api.ts",
			source: `interface User {
  userId: number
  email: string
  status: 'active' | 'disabled'
}

export async function parseUser(response: Response): Promise<User> {
  return await response.json()
}

export function status(): string {
  return 'ok'
}
`,
			task: types.CoverageTestTask{
				ID:             "vitest-json-named-1",
				Framework:      "vitest",
				Target:         "parseUser",
				LineRange:      "7-7",
				GapType:        "return_path",
				TestName:       "covers vitest parseUser named response",
				AssertionFocus: []string{"断言命名类型 JSON 响应结构"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { parseUser } from './api';",
				"const result = await parseUser({ json: async () => ({ userId: 1, email: 'user@example.com', status: 'active' }) });",
				"expect(result).toEqual({ userId: 1, email: 'user@example.com', status: 'active' });",
			},
			forbidden: []string{"describe('status'", "status(", "{ ok: true }", "to.equal(", "require('chai')", "expect(typeof result).toBe('object')"},
		},
		{
			name:     "vitest typescript utility wrapped named response json",
			fileName: "api.ts",
			source: `type User = {
  userId: number
  email: string
}

export async function parseUser(response: Response): Promise<Required<User>> {
  return await response.json()
}

export function status(): string {
  return 'ok'
}
`,
			task: types.CoverageTestTask{
				ID:             "vitest-json-required-1",
				Framework:      "vitest",
				Target:         "parseUser",
				LineRange:      "6-6",
				GapType:        "return_path",
				TestName:       "covers vitest parseUser required response",
				AssertionFocus: []string{"断言 utility wrapped 命名类型 JSON 响应结构"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { parseUser } from './api';",
				"const result = await parseUser({ json: async () => ({ userId: 1, email: 'user@example.com' }) });",
				"expect(result).toEqual({ userId: 1, email: 'user@example.com' });",
			},
			forbidden: []string{"describe('status'", "status(", "{ ok: true }", "to.equal(", "require('chai')", "expect(typeof result).toBe('object')"},
		},
		{
			name:     "vitest typescript named response json array",
			fileName: "api.ts",
			source: `interface User {
  userId: number
  email: string
}

type Users = ReadonlyArray<User | null>

export async function listUsers(response: Response): Promise<Users> {
  return await response.json()
}

export function status(): string {
  return 'ok'
}
`,
			task: types.CoverageTestTask{
				ID:             "vitest-json-array-1",
				Framework:      "vitest",
				Target:         "listUsers",
				LineRange:      "8-8",
				GapType:        "return_path",
				TestName:       "covers vitest listUsers named array response",
				AssertionFocus: []string{"断言命名数组 JSON 响应结构"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { listUsers } from './api';",
				"const result = await listUsers({ json: async () => ([{ userId: 1, email: 'user@example.com' }]) });",
				"expect(result).toEqual([{ userId: 1, email: 'user@example.com' }]);",
			},
			forbidden: []string{"describe('status'", "status(", "{ ok: true }", "to.equal(", "require('chai')", "expect(typeof result).toBe('object')"},
		},
		{
			name:     "vitest typescript named response json tuple",
			fileName: "api.ts",
			source: `interface User {
  userId: number
  email: string
}

type Meta = {
  total: number
  nextUrl?: string | null
}

type UserTuple = readonly [user: User, meta?: Meta]

export async function loadUserTuple(response: Response): Promise<UserTuple> {
  return await response.json()
}

export function status(): string {
  return 'ok'
}
`,
			task: types.CoverageTestTask{
				ID:             "vitest-json-tuple-1",
				Framework:      "vitest",
				Target:         "loadUserTuple",
				LineRange:      "12-12",
				GapType:        "return_path",
				TestName:       "covers vitest loadUserTuple named tuple response",
				AssertionFocus: []string{"断言命名 tuple JSON 响应结构"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { loadUserTuple } from './api';",
				"const result = await loadUserTuple({ json: async () => ([{ userId: 1, email: 'user@example.com' }, { total: 1, nextUrl: 'https://example.com' }]) });",
				"expect(result).toEqual([{ userId: 1, email: 'user@example.com' }, { total: 1, nextUrl: 'https://example.com' }]);",
			},
			forbidden: []string{"describe('status'", "status(", "{ ok: true }", "to.equal(", "require('chai')", "expect(typeof result).toBe('object')"},
		},
		{
			name:     "vitest typescript injected api fetch",
			fileName: "users.ts",
			source: `export async function loadUser(api: { fetch(path: string): Promise<{ ok: boolean }> }): Promise<{ ok: boolean }> {
  return await api.fetch('/users/1')
}
`,
			task: types.CoverageTestTask{
				ID:             "vitest-client-1",
				Framework:      "vitest",
				Target:         "loadUser",
				LineRange:      "2-2",
				GapType:        "return_path",
				TestName:       "covers vitest loadUser injected api",
				AssertionFocus: []string{"断言注入 API 返回结构"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { loadUser } from './users';",
				"const api = {",
				"fetchCalls: [],",
				"fetch: async (...args) => {",
				"api.fetchCalls.push(args);",
				"const result = await loadUser(api);",
				"expect(result).toEqual({ ok: true });",
				"expect(api.fetchCalls).toEqual([['/users/1']]);",
			},
			forbidden: []string{"to.equal(", "require('chai')", "get: async", "request: async", "expect(typeof result).toBe('object')"},
		},
		{
			name:     "vitest typescript named injected api fetch",
			fileName: "users.ts",
			source: `type User = {
  userId: number
  email: string
  createdAt: string
}

export async function loadUser(api: { fetch(path: string): Promise<User> }): Promise<User> {
  return await api.fetch('/users/1')
}

export function status(): string {
  return 'ok'
}
`,
			task: types.CoverageTestTask{
				ID:             "vitest-client-named-1",
				Framework:      "vitest",
				Target:         "loadUser",
				LineRange:      "7-7",
				GapType:        "return_path",
				TestName:       "covers vitest loadUser named api",
				AssertionFocus: []string{"断言命名类型注入 API 返回结构"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { loadUser } from './users';",
				"const api = {",
				"fetchCalls: [],",
				"return { userId: 1, email: 'user@example.com', createdAt: '2026-01-01T00:00:00.000Z' };",
				"const result = await loadUser(api);",
				"expect(result).toEqual({ userId: 1, email: 'user@example.com', createdAt: '2026-01-01T00:00:00.000Z' });",
				"expect(api.fetchCalls).toEqual([['/users/1']]);",
			},
			forbidden: []string{"describe('status'", "status(", "{ ok: true }", "to.equal(", "require('chai')", "get: async", "request: async", "expect(typeof result).toBe('object')"},
		},
		{
			name:     "jest commonjs class branch",
			fileName: "widget.js",
			source: `class Widget {
  load(mode, count) {
    if (mode === 'short') return count
    return count + 1
  }

  save(payload) {
    return payload
  }
}

module.exports = { Widget };
`,
			task: types.CoverageTestTask{
				ID:              "jest-class-branch-1",
				Framework:       "jest",
				Target:          "Widget.load",
				LineRange:       "3-3",
				GapType:         "branch",
				TestName:        "covers jest widget short mode",
				SuggestedInputs: []string{"构造满足条件 `mode === 'short'` 的输入"},
				AssertionFocus:  []string{"断言 Jest class 分支返回值"},
			},
			wants: []string{
				"const { Widget } = require('./widget');",
				"describe('Widget'",
				"describe('load'",
				"it('covers jest widget short mode'",
				"coverage task: jest-class-branch-1 | lines 3-3 | 断言 Jest class 分支返回值 | 构造满足条件 `mode === 'short'` 的输入",
				"const instance = new Widget();",
				"const result = instance.load('short', 1);",
				"expect(result).toBe((1));",
			},
			forbidden: []string{"describe('save'", "to.equal(", "require('chai')", "caughtError"},
		},
		{
			name:     "vitest typescript class async error",
			fileName: "widget.ts",
			source: `export class Widget {
  async load(url?: string): Promise<{ ok: boolean }> {
    if (url === undefined) throw new Error('missing url')
    return { ok: true }
  }

  save(payload: unknown): unknown {
    return payload
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-class-error-1",
				Framework:       "vitest",
				Target:          "Widget.load",
				LineRange:       "3-3",
				GapType:         "error_path",
				TestName:        "covers vitest widget load missing url",
				SuggestedInputs: []string{"构造满足条件 `url === undefined` 的输入"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { Widget } from './widget';",
				"describe('Widget'",
				"describe('load'",
				"it('covers vitest widget load missing url', async () => {",
				"coverage task: vitest-class-error-1 | lines 3-3 | 构造满足条件 `url === undefined` 的输入",
				"const instance = new Widget();",
				"await expect(instance.load(undefined)).rejects.toThrow();",
			},
			forbidden: []string{"describe('save'", "to.equal(", "require('chai')", "caughtError"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcPath := filepath.Join(t.TempDir(), tt.fileName)
			if err := os.WriteFile(srcPath, []byte(tt.source), 0644); err != nil {
				t.Fatal(err)
			}
			code, err := GenerateJavaScriptTestsForCoverageTask(srcPath, &tt.task)
			if err != nil {
				t.Fatalf("GenerateJavaScriptTestsForCoverageTask() error = %v", err)
			}
			assertGeneratedJS(t, code, tt.wants, tt.forbidden)
		})
	}
}

func assertGeneratedJS(t *testing.T, code string, wants []string, forbidden []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, item := range forbidden {
		if strings.Contains(code, item) {
			t.Fatalf("generated code should not contain %q:\n%s", item, code)
		}
	}
}

func TestGenerateMochaCoverageTaskUsesChaiMatrixAssertions(t *testing.T) {
	tests := []struct {
		name      string
		fileName  string
		source    string
		task      types.CoverageTestTask
		wants     []string
		forbidden []string
	}{
		{
			name:     "commonjs function sync error",
			fileName: "calc.js",
			source: `function divide(a, b) {
  if (b === 0) throw new Error('zero')
  return a / b
}

module.exports = { divide };
`,
			task: types.CoverageTestTask{
				ID:              "mocha-error-1",
				Framework:       "mocha",
				Target:          "divide",
				LineRange:       "2-2",
				GapType:         "error_path",
				TestName:        "covers divide zero error",
				SuggestedInputs: []string{"构造满足条件 `b === 0` 的输入"},
			},
			wants: []string{
				"const { expect } = require('chai');",
				"it('covers divide zero error'",
				"expect(() => divide(1, 0)).to.throw();",
			},
			forbidden: []string{"toThrow()", "rejects.toThrow()"},
		},
		{
			name:     "commonjs function async error",
			fileName: "service.js",
			source: `async function fetchData(url) {
  if (url === undefined) throw new Error('missing')
  return { ok: true }
}

module.exports = { fetchData };
`,
			task: types.CoverageTestTask{
				ID:              "mocha-error-2",
				Framework:       "mocha",
				Target:          "fetchData",
				LineRange:       "2-2",
				GapType:         "error_path",
				TestName:        "covers fetchData missing url",
				SuggestedInputs: []string{"构造满足条件 `url === undefined` 的输入"},
			},
			wants: []string{
				"const { expect } = require('chai');",
				"let caughtError;",
				"try {",
				"await fetchData(undefined);",
				"caughtError = err;",
				"expect(caughtError).to.exist;",
			},
			forbidden: []string{"rejects.toThrow()", "toThrow()"},
		},
		{
			name:     "commonjs class sync error",
			fileName: "widget.js",
			source: `class Widget {
  save(payload) {
    if (payload === null) throw new Error('missing payload')
    return true
  }

  load(mode) {
    return mode
  }
}

module.exports = { Widget };
`,
			task: types.CoverageTestTask{
				ID:              "mocha-class-error-1",
				Framework:       "mocha",
				Target:          "Widget.save",
				LineRange:       "3-3",
				GapType:         "error_path",
				TestName:        "covers widget save missing payload",
				SuggestedInputs: []string{"构造满足条件 `payload === null` 的输入"},
			},
			wants: []string{
				"const { expect } = require('chai');",
				"const { Widget } = require('./widget');",
				"describe('Widget'",
				"describe('save'",
				"it('covers widget save missing payload'",
				"const instance = new Widget();",
				"expect(() => instance.save(null)).to.throw();",
			},
			forbidden: []string{"describe('load'", "toThrow()", "rejects.toThrow()"},
		},
		{
			name:     "commonjs class async error",
			fileName: "widget.js",
			source: `class Widget {
  async load(url) {
    if (url === undefined) throw new Error('missing url')
    return { ok: true }
  }

  save(payload) {
    return payload
  }
}

module.exports = { Widget };
`,
			task: types.CoverageTestTask{
				ID:              "mocha-class-error-2",
				Framework:       "mocha",
				Target:          "Widget.load",
				LineRange:       "3-3",
				GapType:         "error_path",
				TestName:        "covers widget load missing url",
				SuggestedInputs: []string{"构造满足条件 `url === undefined` 的输入"},
			},
			wants: []string{
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
			},
			forbidden: []string{"describe('save'", "toThrow()", "rejects.toThrow()"},
		},
		{
			name:     "esm function sync error",
			fileName: "calc.js",
			source: `export function divide(a, b) {
  if (b === 0) throw new Error('zero')
  return a / b
}

export function add(a, b) {
  return a + b
}
`,
			task: types.CoverageTestTask{
				ID:              "mocha-esm-error-1",
				Framework:       "mocha",
				Target:          "divide",
				LineRange:       "2-2",
				GapType:         "error_path",
				TestName:        "covers esm divide zero error",
				SuggestedInputs: []string{"构造满足条件 `b === 0` 的输入"},
			},
			wants: []string{
				"import { expect } from 'chai';",
				"import { divide } from './calc';",
				"it('covers esm divide zero error'",
				"expect(() => divide(1, 0)).to.throw();",
			},
			forbidden: []string{"require('chai')", "require('./calc')", "describe('add'", "toThrow()", "rejects.toThrow()"},
		},
		{
			name:     "esm class async error",
			fileName: "widget.js",
			source: `export class Widget {
  async load(url) {
    if (url === undefined) throw new Error('missing url')
    return { ok: true }
  }

  save(payload) {
    return payload
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "mocha-esm-class-error-1",
				Framework:       "mocha",
				Target:          "Widget.load",
				LineRange:       "3-3",
				GapType:         "error_path",
				TestName:        "covers esm widget load missing url",
				SuggestedInputs: []string{"构造满足条件 `url === undefined` 的输入"},
			},
			wants: []string{
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
			},
			forbidden: []string{"require('chai')", "require('./widget')", "describe('save'", "toThrow()", "rejects.toThrow()"},
		},
		{
			name:     "esm function return",
			fileName: "calc.js",
			source: `export function add(a, b) {
  return a + b
}

export function sub(a, b) {
  return a - b
}
`,
			task: types.CoverageTestTask{
				ID:              "mocha-esm-return-1",
				Framework:       "mocha",
				Target:          "add",
				LineRange:       "2-2",
				GapType:         "return_path",
				TestName:        "covers esm add zero left operand",
				SuggestedInputs: []string{"构造满足条件 `a === 0` 的输入"},
				AssertionFocus:  []string{"断言 ESM 返回路径的具体结果"},
			},
			wants: []string{
				"import { expect } from 'chai';",
				"import { add } from './calc';",
				"it('covers esm add zero left operand'",
				"coverage task: mocha-esm-return-1 | lines 2-2 | 断言 ESM 返回路径的具体结果 | 构造满足条件 `a === 0` 的输入",
				"const result = add(0, 2);",
				"expect(result).to.equal((0 + 2));",
			},
			forbidden: []string{"require('chai')", "require('./calc')", "describe('sub'", "toBe(", "toThrow()", "rejects.toThrow()"},
		},
		{
			name:     "esm class branch",
			fileName: "widget.js",
			source: `export class Widget {
  load(mode, count) {
    if (mode === 'short') return count
    return count + 1
  }

  save(payload) {
    return payload
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "mocha-esm-class-branch-1",
				Framework:       "mocha",
				Target:          "Widget.load",
				LineRange:       "3-3",
				GapType:         "branch",
				TestName:        "covers esm widget short mode",
				SuggestedInputs: []string{"构造满足条件 `mode === 'short'` 的输入"},
				AssertionFocus:  []string{"断言 ESM class 分支返回值"},
			},
			wants: []string{
				"import { expect } from 'chai';",
				"import { Widget } from './widget';",
				"describe('Widget'",
				"describe('load'",
				"it('covers esm widget short mode'",
				"coverage task: mocha-esm-class-branch-1 | lines 3-3 | 断言 ESM class 分支返回值 | 构造满足条件 `mode === 'short'` 的输入",
				"const instance = new Widget();",
				"const result = instance.load('short', 1);",
				"expect(result).to.equal((1));",
			},
			forbidden: []string{"require('chai')", "require('./widget')", "describe('save'", "toBe(", "toThrow()", "rejects.toThrow()"},
		},
		{
			name:     "typescript function return",
			fileName: "calc.ts",
			source: `export function add(a: number, b: number): number {
  return a + b
}

export function sub(a: number, b: number): number {
  return a - b
}
`,
			task: types.CoverageTestTask{
				ID:              "mocha-ts-return-1",
				Framework:       "mocha",
				Target:          "add",
				LineRange:       "2-2",
				GapType:         "return_path",
				TestName:        "covers ts add zero left operand",
				SuggestedInputs: []string{"构造满足条件 `a === 0` 的输入"},
				AssertionFocus:  []string{"断言 TypeScript 返回路径的具体结果"},
			},
			wants: []string{
				"import { expect } from 'chai';",
				"import { add } from './calc';",
				"it('covers ts add zero left operand'",
				"coverage task: mocha-ts-return-1 | lines 2-2 | 断言 TypeScript 返回路径的具体结果 | 构造满足条件 `a === 0` 的输入",
				"const result = add(0, 2);",
				"expect(result).to.equal((0 + 2));",
			},
			forbidden: []string{"require('chai')", "require('./calc')", "describe('sub'", "toBe(", "toThrow()", "rejects.toThrow()"},
		},
		{
			name:     "typescript class branch",
			fileName: "widget.ts",
			source: `export class Widget {
  load(mode: string, count: number): number {
    if (mode === 'short') return count
    return count + 1
  }

  save(payload: unknown): unknown {
    return payload
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "mocha-ts-class-branch-1",
				Framework:       "mocha",
				Target:          "Widget.load",
				LineRange:       "3-3",
				GapType:         "branch",
				TestName:        "covers ts widget short mode",
				SuggestedInputs: []string{"构造满足条件 `mode === 'short'` 的输入"},
				AssertionFocus:  []string{"断言 TypeScript class 分支返回值"},
			},
			wants: []string{
				"import { expect } from 'chai';",
				"import { Widget } from './widget';",
				"describe('Widget'",
				"describe('load'",
				"it('covers ts widget short mode'",
				"coverage task: mocha-ts-class-branch-1 | lines 3-3 | 断言 TypeScript class 分支返回值 | 构造满足条件 `mode === 'short'` 的输入",
				"const instance = new Widget();",
				"const result = instance.load('short', 1);",
				"expect(result).to.equal((1));",
			},
			forbidden: []string{"require('chai')", "require('./widget')", "describe('save'", "toBe(", "toThrow()", "rejects.toThrow()"},
		},
		{
			name:     "typescript function sync error",
			fileName: "calc.ts",
			source: `export function divide(a: number, b: number): number {
  if (b === 0) throw new Error('zero')
  return a / b
}

export function add(a: number, b: number): number {
  return a + b
}
`,
			task: types.CoverageTestTask{
				ID:              "mocha-ts-error-1",
				Framework:       "mocha",
				Target:          "divide",
				LineRange:       "2-2",
				GapType:         "error_path",
				TestName:        "covers ts divide zero error",
				SuggestedInputs: []string{"构造满足条件 `b === 0` 的输入"},
			},
			wants: []string{
				"import { expect } from 'chai';",
				"import { divide } from './calc';",
				"it('covers ts divide zero error'",
				"expect(() => divide(1, 0)).to.throw();",
			},
			forbidden: []string{"require('chai')", "require('./calc')", "describe('add'", "toBe(", "toThrow()", "rejects.toThrow()"},
		},
		{
			name:     "typescript class async error",
			fileName: "widget.ts",
			source: `export class Widget {
  async load(url?: string): Promise<{ ok: boolean }> {
    if (url === undefined) throw new Error('missing url')
    return { ok: true }
  }

  save(payload: unknown): unknown {
    return payload
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "mocha-ts-class-error-1",
				Framework:       "mocha",
				Target:          "Widget.load",
				LineRange:       "3-3",
				GapType:         "error_path",
				TestName:        "covers ts widget load missing url",
				SuggestedInputs: []string{"构造满足条件 `url === undefined` 的输入"},
			},
			wants: []string{
				"import { expect } from 'chai';",
				"import { Widget } from './widget';",
				"describe('Widget'",
				"describe('load'",
				"it('covers ts widget load missing url', async () => {",
				"const instance = new Widget();",
				"let caughtError;",
				"await instance.load(undefined);",
				"caughtError = err;",
				"expect(caughtError).to.exist;",
			},
			forbidden: []string{"require('chai')", "require('./widget')", "describe('save'", "toBe(", "toThrow()", "rejects.toThrow()"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcPath := filepath.Join(t.TempDir(), tt.fileName)
			if err := os.WriteFile(srcPath, []byte(tt.source), 0644); err != nil {
				t.Fatal(err)
			}
			code, err := GenerateJavaScriptTestsForCoverageTask(srcPath, &tt.task)
			if err != nil {
				t.Fatalf("GenerateJavaScriptTestsForCoverageTask() error = %v", err)
			}
			assertGeneratedJS(t, code, tt.wants, tt.forbidden)
		})
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

func TestJSClassCoverageTaskCoversNormalAndErrorMethods(t *testing.T) {
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

	code := genJSClassTestForCoverageTask(cls, &task)
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

func TestJSRegularGenerationDedupesDuplicateErrorPathInputs(t *testing.T) {
	fn := jsFuncInfo{
		Name:    "fetchData",
		IsAsync: true,
		Params:  []jsParamInfo{{Name: "url"}},
		Analysis: jsFuncAnalysis{
			Throws:    true,
			HasReturn: true,
			Boundaries: []jsBoundary{
				{Param: "url", Value: "undefined", Type: "undefined"},
			},
		},
	}
	code := genJSFuncTest(fn, jsAssertionStyleJest)
	if !strings.Contains(code, "it('should handle url = undefined'") {
		t.Fatalf("expected boundary error-path case:\n%s", code)
	}
	if strings.Contains(code, "it('should throw on invalid input'") {
		t.Fatalf("duplicate generic error-path case should be omitted:\n%s", code)
	}

	cls := jsClassInfo{
		Name: "Widget",
		Methods: []jsFuncInfo{
			{
				Name:    "save",
				IsAsync: true,
				Params:  []jsParamInfo{{Name: "payload"}},
				Analysis: jsFuncAnalysis{
					Throws:    true,
					HasReturn: true,
					Boundaries: []jsBoundary{
						{Param: "payload", Value: "undefined", Type: "undefined"},
					},
				},
			},
		},
	}
	code = genJSClassTest(cls, jsAssertionStyleChai)
	if !strings.Contains(code, "it('should handle payload = undefined'") {
		t.Fatalf("expected class boundary error-path case:\n%s", code)
	}
	if strings.Contains(code, "it('should throw on invalid input'") {
		t.Fatalf("duplicate class generic error-path case should be omitted:\n%s", code)
	}
}

func TestJSFuncCoverageTaskCoversAsyncErrorFallbackName(t *testing.T) {
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

	code := genJSFuncTestForCoverageTask(fn, &task)
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
	expr, ok = jsExpectedReturnExpr(
		jsFuncAnalysis{Returns: []string{"{ url, ok: true }"}},
		[]jsParamInfo{{Name: "url"}},
		nil,
	)
	if !ok || expr != "{ url: 'https://example.com', ok: true }" {
		t.Fatalf("object shorthand jsExpectedReturnExpr() = %q, %v", expr, ok)
	}
	expr, ok = jsExpectedReturnExpr(
		jsFuncAnalysis{Returns: []string{"[a, b, 3]"}},
		[]jsParamInfo{{Name: "a"}, {Name: "b"}},
		nil,
	)
	if !ok || expr != "[1, 2, 3]" {
		t.Fatalf("array literal jsExpectedReturnExpr() = %q, %v", expr, ok)
	}
	expr, ok = jsExpectedReturnExprWithValues(
		jsFuncAnalysis{Returns: []string{"{ mode, count: count + 1 }"}},
		[]jsParamInfo{{Name: "mode"}, {Name: "count"}},
		nil,
		map[string]string{"mode": "'short'", "count": "1"},
	)
	if !ok || expr != "{ mode: 'short', count: 1 + 1 }" {
		t.Fatalf("coverage object jsExpectedReturnExprWithValues() = %q, %v", expr, ok)
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
		"{ ok: true }": true,
		"[a, b]":       true,
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
	if got := jsArgListForAnalysis(
		[]jsParamInfo{{Name: "client"}},
		jsFuncAnalysis{Returns: []string{"await client.get('/users/1')"}},
	); got != "{ get: async () => ({ ok: true }) }" {
		t.Fatalf("jsArgListForAnalysis(get) = %q", got)
	}
	if got := jsArgListForAnalysis(
		[]jsParamInfo{{Name: "api"}},
		jsFuncAnalysis{Returns: []string{"await api.fetch('/users/1')"}},
	); got != "{ fetch: async () => ({ ok: true }) }" {
		t.Fatalf("jsArgListForAnalysis(fetch) = %q", got)
	}
	if got := jsArgListForAnalysis(
		[]jsParamInfo{{Name: "http"}},
		jsFuncAnalysis{Returns: []string{"await http.request('/users/1')"}},
	); got != "{ request: async () => ({ ok: true }) }" {
		t.Fatalf("jsArgListForAnalysis(request) = %q", got)
	}
	receiver, method, args, ok := jsInjectedClientCall("await http.request('/users/1', { method: 'GET' })")
	if !ok || receiver != "http" || method != "request" || args != "'/users/1', { method: 'GET' }" {
		t.Fatalf("jsInjectedClientCall() = %q, %q, %q, %v", receiver, method, args, ok)
	}
	callInfo := &jsInjectedClientCallInfo{Param: "http", Method: "request", Args: "'/users/1', { method: 'GET' }"}
	if got := genJSInjectedClientCallAssertion(callInfo, "  ", jsAssertionStyleChai); got != "  expect(http.requestCalls).to.deep.equal([['/users/1', { method: 'GET' }]]);\n" {
		t.Fatalf("Chai client call assertion = %q", got)
	}
	if got, ok := jsMockPayloadFromTSType("Promise<Array<{ id: number; name: string }>>"); !ok || got != "[{ id: 1, name: 'test' }]" {
		t.Fatalf("array payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSType("Promise<User>"); ok || got != "" {
		t.Fatalf("named type payload = %q, %v", got, ok)
	}
	typeDecls := map[string]string{"User": "{ userId: number; email: string; status: 'active' | 'disabled'; createdAt: string; displayName?: string | null; manager?: User | null }"}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<User | null>", typeDecls); !ok || got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }" {
		t.Fatalf("decl payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Readonly<User>>", typeDecls); !ok || got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }" {
		t.Fatalf("readonly utility payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Partial<Readonly<User>>>", typeDecls); !ok || got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }" {
		t.Fatalf("nested utility payload = %q, %v", got, ok)
	}
	if got := jsMockValueForTSTypeWithDecls("owner", "null | User", typeDecls); got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }" {
		t.Fatalf("nullable owner value = %q", got)
	}
	if got := jsMockValueForTSTypeWithDecls("owner", "Readonly<User>", typeDecls); got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }" {
		t.Fatalf("readonly owner value = %q", got)
	}
	typeDecls["Users"] = "ReadonlyArray<User | null>"
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Users>", typeDecls); !ok || got != "[{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }]" {
		t.Fatalf("array alias payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Readonly<Users>>", typeDecls); !ok || got != "[{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }]" {
		t.Fatalf("readonly utility array payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<readonly User[]>", typeDecls); !ok || got != "[{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }]" {
		t.Fatalf("readonly array payload = %q, %v", got, ok)
	}
	typeDecls["Meta"] = "{ total: number; nextUrl?: string | null }"
	typeDecls["UserTuple"] = "readonly [user: User, meta?: Meta]"
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<UserTuple>", typeDecls); !ok || got != "[{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }, { total: 1, nextUrl: 'https://example.com' }]" {
		t.Fatalf("tuple alias payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<readonly [User, ...Meta[]]>", typeDecls); !ok || got != "[{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }, { total: 1, nextUrl: 'https://example.com' }]" {
		t.Fatalf("rest tuple payload = %q, %v", got, ok)
	}
	if got := genJSResultAssertionWithArgsStyle(
		jsFuncAnalysis{HasReturn: true, ReturnType: "object", Returns: []string{"{ ok: true }"}},
		nil,
		nil,
		"  ",
		jsAssertionStyleJest,
	); got != "  expect(result).toEqual({ ok: true });\n" {
		t.Fatalf("Jest object assertion = %q", got)
	}
	if got := genJSResultAssertionWithArgsStyle(
		jsFuncAnalysis{HasReturn: true, ReturnType: "array", Returns: []string{"[a, b]"}},
		[]jsParamInfo{{Name: "a"}, {Name: "b"}},
		nil,
		"  ",
		jsAssertionStyleChai,
	); got != "  expect(result).to.deep.equal([1, 2]);\n" {
		t.Fatalf("Chai array assertion = %q", got)
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
