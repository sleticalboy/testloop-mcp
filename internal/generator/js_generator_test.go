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

type AuditFields = {
  traceId: string
  page: number
}

type ApiResponse = {
  data: User
  meta: Meta
  count: number
}

type Users = readonly User[]
type MaybeUsers = ReadonlyArray<User | null>
type UserTuple = readonly [user: User, meta?: Meta]
type PublicUser = Pick<User, 'userId' | 'email'>
type UserWithoutMeta = Omit<User, 'manager' | 'displayName'>
type UserMap = Record<string, User>
type FeaturedUsers = Record<'primary' | 'secondary', User>
type AuditedUser = User & AuditFields
type ResponseData = ApiResponse['data']

export async function parseUser(response: Response): Promise<User> {
  return await response.json();
}

export async function parseReadonlyUser(response: Response): Promise<Readonly<User>> {
  return await response.json();
}

export async function parsePublicUser(response: Response): Promise<PublicUser> {
  return await response.json();
}

export async function parseUserWithoutMeta(response: Response): Promise<UserWithoutMeta> {
  return await response.json();
}

export async function parseUserMap(response: Response): Promise<UserMap> {
  return await response.json();
}

export async function parseFeaturedUsers(response: Response): Promise<FeaturedUsers> {
  return await response.json();
}

export async function parseAuditedUser(response: Response): Promise<AuditedUser> {
  return await response.json();
}

export async function parseInlineIntersection(response: Response): Promise<{ id: number } & { email: string }> {
  return await response.json();
}

export async function parseResponseData(response: Response): Promise<ResponseData> {
  return await response.json();
}

export async function parseInlineIndexed(response: Response): Promise<{ data: { id: number; email: string } }['data']> {
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
		"const result = await parsePublicUser({ json: async () => ({ userId: 1, email: 'user@example.com' }) });",
		"expect(result).toEqual({ userId: 1, email: 'user@example.com' });",
		"const result = await parseUserWithoutMeta({ json: async () => ({ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z' }) });",
		"expect(result).toEqual({ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z' });",
		"const result = await parseUserMap({ json: async () => ({ key: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} } }) });",
		"expect(result).toEqual({ key: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} } });",
		"const result = await parseFeaturedUsers({ json: async () => ({ primary: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }, secondary: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} } }) });",
		"expect(result).toEqual({ primary: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }, secondary: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} } });",
		"const result = await parseAuditedUser({ json: async () => ({ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {}, traceId: 'id-1', page: 1 }) });",
		"expect(result).toEqual({ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {}, traceId: 'id-1', page: 1 });",
		"const result = await parseInlineIntersection({ json: async () => ({ id: 1, email: 'user@example.com' }) });",
		"expect(result).toEqual({ id: 1, email: 'user@example.com' });",
		"const result = await parseResponseData({ json: async () => ({ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }) });",
		"expect(result).toEqual({ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} });",
		"const result = await parseInlineIndexed({ json: async () => ({ id: 1, email: 'user@example.com' }) });",
		"expect(result).toEqual({ id: 1, email: 'user@example.com' });",
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

func TestGenerateJavaScriptTestsComplexTypeCompositionPayloads(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "api.ts")
	src := `interface User {
  userId: number
  email: string
  status: 'active' | 'disabled'
  manager?: User | null
}

type AuditFields = {
  traceId: string
  page: number
}

type Meta = {
  total: number
  nextUrl?: string | null
}

type ApiResponse = {
  data: User
  meta: Meta
  debug: string
}

type ApiEnvelope<T> = {
  data: T
  meta: Meta
}

type Directory = Readonly<Record<'primary' | 'secondary', ApiResponse['data'] & AuditFields>>
type DirectoryEnvelope = Omit<{ directory: Directory; meta: ApiResponse['meta']; debug: string }, 'debug'>
type DirectorySummary = Pick<DirectoryEnvelope, 'directory' | 'meta'>
type DirectoryBundle = {
  reports: User[]
  pair: readonly [user: User, meta?: Meta]
  directory: Record<string, Pick<User, 'userId' | 'email'>>
  summary: DirectorySummary
}

export async function loadDirectory(response: Response): Promise<DirectorySummary> {
  return await response.json()
}

export async function loadDirectoryBundle(response: Response): Promise<DirectoryBundle> {
  return await response.json()
}

export async function loadUserEnvelope(response: Response): Promise<ApiEnvelope<User>> {
  return await response.json()
}

export async function loadDirectoryBundleClient(api: { fetch(path: string): Promise<DirectoryBundle> }): Promise<DirectoryBundle> {
  return await api.fetch('/directory/bundle')
}

export async function loadDirectoryClient(api: { fetch(path: string): Promise<DirectoryEnvelope> }): Promise<DirectoryEnvelope> {
  return await api.fetch('/directory')
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	code, err := GenerateJavaScriptTestsWithFramework(srcPath, "vitest")
	if err != nil {
		t.Fatalf("GenerateJavaScriptTestsWithFramework() error = %v", err)
	}

	directoryPayload := "{ directory: { primary: { userId: 1, email: 'user@example.com', status: 'active', manager: {}, traceId: 'id-1', page: 1 }, secondary: { userId: 1, email: 'user@example.com', status: 'active', manager: {}, traceId: 'id-1', page: 1 } }, meta: { total: 1, nextUrl: 'https://example.com' } }"
	bundlePayload := "{ reports: [{ userId: 1, email: 'user@example.com', status: 'active', manager: {} }], pair: [{ userId: 1, email: 'user@example.com', status: 'active', manager: {} }, { total: 1, nextUrl: 'https://example.com' }], directory: { key: { userId: 1, email: 'user@example.com' } }, summary: " + directoryPayload + " }"
	envelopePayload := "{ data: { userId: 1, email: 'user@example.com', status: 'active', manager: {} }, meta: { total: 1, nextUrl: 'https://example.com' } }"
	assertGeneratedJS(t, code, []string{
		"const result = await loadDirectory({ json: async () => (" + directoryPayload + ") });",
		"expect(result).toEqual(" + directoryPayload + ");",
		"const result = await loadDirectoryBundle({ json: async () => (" + bundlePayload + ") });",
		"expect(result).toEqual(" + bundlePayload + ");",
		"const result = await loadUserEnvelope({ json: async () => (" + envelopePayload + ") });",
		"expect(result).toEqual(" + envelopePayload + ");",
		"return " + bundlePayload + ";",
		"const result = await loadDirectoryBundleClient(api);",
		"expect(result).toEqual(" + bundlePayload + ");",
		"expect(api.fetchCalls).toEqual([['/directory/bundle']]);",
		"return " + directoryPayload + ";",
		"const result = await loadDirectoryClient(api);",
		"expect(result).toEqual(" + directoryPayload + ");",
		"expect(api.fetchCalls).toEqual([['/directory']]);",
	}, []string{
		"{ ok: true }",
		"debug: 'test'",
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
			name:     "jest version from header branch",
			fileName: "util.js",
			source: `export function versionFromHeader(h) {
  if (h.version == 2) return { name: 'IPv4' }
  if (h.version != 3) return null
  if (h.ipVersion == 4) return { name: 'IPv4' }
  if (h.ipVersion == 6) return { name: 'IPv6' }
  return null
}
`,
			task: types.CoverageTestTask{
				ID:              "jest-version-header-1",
				Framework:       "jest",
				Target:          "versionFromHeader",
				LineRange:       "4-4",
				GapType:         "branch",
				TestName:        "covers versionFromHeader IPv4 header",
				MissingBranches: []string{"未覆盖 if 分支: ipVer == XdbIPv4Id"},
				SuggestedInputs: []string{"构造满足条件 `ipVer == XdbIPv4Id` 的输入"},
			},
			wants: []string{
				"import { versionFromHeader } from './util';",
				"const result = versionFromHeader({ version: 3, ipVersion: 4 });",
				"expect(result?.name).toBe('IPv4');",
			},
			forbidden: []string{"versionFromHeader(undefined)", "expect(result).toBeNull();"},
		},
		{
			name:     "jest internal ipv6 parser via parseIP",
			fileName: "util.js",
			source: `function _parse_ipv6_addr(v6String) {
  if (v6String === '1::2::3') throw new Error('invalid ipv6 address: multi double colon detected')
  return Buffer.alloc(16)
}

export function parseIP(ipString) {
  return _parse_ipv6_addr(ipString)
}
`,
			task: types.CoverageTestTask{
				ID:              "jest-ipv6-private-1",
				Framework:       "jest",
				Target:          "_parse_ipv6_addr",
				LineRange:       "91-91",
				GapType:         "error_path",
				TestName:        "covers internal ipv6 multi double colon",
				SuggestedInputs: []string{"设置 v6String 覆盖未执行分支"},
			},
			wants: []string{
				"import { parseIP } from './util';",
				"describe('parseIP'",
				"expect(() => parseIP('1::2::3')).toThrow();",
			},
			forbidden: []string{"_parse_ipv6_addr }", "_parse_ipv6_addr(undefined)", "import { _parse_ipv6_addr"},
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
			name:     "vitest typescript pick named response json",
			fileName: "api.ts",
			source: `interface User {
  userId: number
  email: string
  status: 'active' | 'disabled'
}

type PublicUser = Pick<User, 'userId' | 'email'>

export async function parsePublicUser(response: Response): Promise<PublicUser> {
  return await response.json()
}

export function status(): string {
  return 'ok'
}
`,
			task: types.CoverageTestTask{
				ID:             "vitest-json-pick-1",
				Framework:      "vitest",
				Target:         "parsePublicUser",
				LineRange:      "9-9",
				GapType:        "return_path",
				TestName:       "covers vitest parsePublicUser pick response",
				AssertionFocus: []string{"断言 Pick 命名类型 JSON 响应结构"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { parsePublicUser } from './api';",
				"const result = await parsePublicUser({ json: async () => ({ userId: 1, email: 'user@example.com' }) });",
				"expect(result).toEqual({ userId: 1, email: 'user@example.com' });",
			},
			forbidden: []string{"describe('status'", "status(", "status: 'active'", "{ ok: true }", "to.equal(", "require('chai')", "expect(typeof result).toBe('object')"},
		},
		{
			name:     "vitest typescript omit named response json",
			fileName: "api.ts",
			source: `interface User {
  userId: number
  email: string
  status: 'active' | 'disabled'
}

type UserWithoutStatus = Omit<User, 'status'>

export async function parseUserWithoutStatus(response: Response): Promise<UserWithoutStatus> {
  return await response.json()
}

export function status(): string {
  return 'ok'
}
`,
			task: types.CoverageTestTask{
				ID:             "vitest-json-omit-1",
				Framework:      "vitest",
				Target:         "parseUserWithoutStatus",
				LineRange:      "9-9",
				GapType:        "return_path",
				TestName:       "covers vitest parseUserWithoutStatus omit response",
				AssertionFocus: []string{"断言 Omit 命名类型 JSON 响应结构"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { parseUserWithoutStatus } from './api';",
				"const result = await parseUserWithoutStatus({ json: async () => ({ userId: 1, email: 'user@example.com' }) });",
				"expect(result).toEqual({ userId: 1, email: 'user@example.com' });",
			},
			forbidden: []string{"describe('status'", "status(", "status: 'active'", "{ ok: true }", "to.equal(", "require('chai')", "expect(typeof result).toBe('object')"},
		},
		{
			name:     "vitest typescript record named response json",
			fileName: "api.ts",
			source: `interface User {
  userId: number
  email: string
}

type FeaturedUsers = Record<'primary' | 'secondary', User>

export async function loadFeaturedUsers(response: Response): Promise<FeaturedUsers> {
  return await response.json()
}

export function status(): string {
  return 'ok'
}
`,
			task: types.CoverageTestTask{
				ID:             "vitest-json-record-1",
				Framework:      "vitest",
				Target:         "loadFeaturedUsers",
				LineRange:      "8-8",
				GapType:        "return_path",
				TestName:       "covers vitest loadFeaturedUsers record response",
				AssertionFocus: []string{"断言 Record 命名类型 JSON 响应结构"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { loadFeaturedUsers } from './api';",
				"const result = await loadFeaturedUsers({ json: async () => ({ primary: { userId: 1, email: 'user@example.com' }, secondary: { userId: 1, email: 'user@example.com' } }) });",
				"expect(result).toEqual({ primary: { userId: 1, email: 'user@example.com' }, secondary: { userId: 1, email: 'user@example.com' } });",
			},
			forbidden: []string{"describe('status'", "status(", "{ ok: true }", "to.equal(", "require('chai')", "expect(typeof result).toBe('object')"},
		},
		{
			name:     "vitest typescript intersection named response json",
			fileName: "api.ts",
			source: `interface User {
  userId: number
  email: string
}

type AuditFields = {
  traceId: string
  page: number
}

type AuditedUser = User & AuditFields

export async function loadAuditedUser(response: Response): Promise<AuditedUser> {
  return await response.json()
}

export function status(): string {
  return 'ok'
}
`,
			task: types.CoverageTestTask{
				ID:             "vitest-json-intersection-1",
				Framework:      "vitest",
				Target:         "loadAuditedUser",
				LineRange:      "13-13",
				GapType:        "return_path",
				TestName:       "covers vitest loadAuditedUser intersection response",
				AssertionFocus: []string{"断言交叉类型 JSON 响应结构"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { loadAuditedUser } from './api';",
				"const result = await loadAuditedUser({ json: async () => ({ userId: 1, email: 'user@example.com', traceId: 'id-1', page: 1 }) });",
				"expect(result).toEqual({ userId: 1, email: 'user@example.com', traceId: 'id-1', page: 1 });",
			},
			forbidden: []string{"describe('status'", "status(", "{ ok: true }", "to.equal(", "require('chai')", "expect(typeof result).toBe('object')"},
		},
		{
			name:     "vitest typescript indexed access response json",
			fileName: "api.ts",
			source: `interface User {
  userId: number
  email: string
}

type ApiResponse = {
  data: User
  count: number
}

type ResponseData = ApiResponse['data']

export async function loadResponseData(response: Response): Promise<ResponseData> {
  return await response.json()
}

export function status(): string {
  return 'ok'
}
`,
			task: types.CoverageTestTask{
				ID:             "vitest-json-indexed-1",
				Framework:      "vitest",
				Target:         "loadResponseData",
				LineRange:      "13-13",
				GapType:        "return_path",
				TestName:       "covers vitest loadResponseData indexed response",
				AssertionFocus: []string{"断言 indexed access JSON 响应结构"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { loadResponseData } from './api';",
				"const result = await loadResponseData({ json: async () => ({ userId: 1, email: 'user@example.com' }) });",
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
		{
			name:     "vitest class strict fallback error path",
			fileName: "env-resolver.js",
			source: `export class EnvResolver {
  constructor(options = {}) {
    this.strict = options.strict !== false
  }

  async _resolveEnvObject(envConfig, baseContext) {
    const resolved = {}
    for (const [key, value] of Object.entries(envConfig)) {
      if (value === null || value === '') {
        const fallbackValue = baseContext[key]
        if (fallbackValue === undefined && this.strict) {
          throw new Error('missing variable')
        }
        resolved[key] = fallbackValue || ''
      }
    }
    return resolved
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-env-strict-1",
				Framework:       "vitest",
				Target:          "EnvResolver._resolveEnvObject",
				LineRange:       "11-11",
				GapType:         "error_path",
				TestName:        "covers EnvResolver strict fallback",
				SuggestedInputs: []string{"设置 envConfig 覆盖未执行分支", "设置 baseContext 覆盖未执行分支"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { EnvResolver } from './env-resolver';",
				"const instance = new EnvResolver({ strict: true });",
				"await expect(instance._resolveEnvObject({ MISSING: null }, {})).rejects.toThrow();",
			},
			forbidden: []string{"_resolveEnvObject({}, 'test')"},
		},
		{
			name:     "vitest class max passes error path",
			fileName: "env-resolver.js",
			source: `export class EnvResolver {
  constructor(options = {}) {
    this.maxPasses = options.maxPasses || 10
  }

  async _resolveStringWithPlaceholders(str, context, depth = 0) {
    if (depth > this.maxPasses) {
      throw new Error('Max placeholder resolution depth exceeded')
    }
    if (typeof str !== 'string' || !str.includes('${')) {
      return str
    }
    return str
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-env-depth-1",
				Framework:       "vitest",
				Target:          "EnvResolver._resolveStringWithPlaceholders",
				LineRange:       "8-8",
				GapType:         "error_path",
				TestName:        "covers EnvResolver max passes",
				SuggestedInputs: []string{"设置 str 覆盖未执行分支", "设置 context 覆盖未执行分支", "设置 depth 覆盖未执行分支"},
			},
			wants: []string{
				"const instance = new EnvResolver({ maxPasses: 0 });",
				"await expect(instance._resolveStringWithPlaceholders('${MISSING}', {}, 1)).rejects.toThrow();",
			},
			forbidden: []string{"_resolveStringWithPlaceholders(undefined, 'test', undefined)"},
		},
		{
			name:     "vitest class placeholder plain return path",
			fileName: "env-resolver.js",
			source: `export class EnvResolver {
  async _resolveStringWithPlaceholders(str, context, depth = 0) {
    if (depth > 10) {
      throw new Error('too deep')
    }
    const placeholders = []
    if (placeholders.length === 0) {
      return str
    }
    return context.value
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-env-return-1",
				Framework:       "vitest",
				Target:          "EnvResolver._resolveStringWithPlaceholders",
				LineRange:       "8-8",
				GapType:         "return_path",
				TestName:        "covers EnvResolver placeholder return",
				SuggestedInputs: []string{"设置 str 覆盖未执行分支", "设置 context 覆盖未执行分支", "设置 depth 覆盖未执行分支"},
			},
			wants: []string{
				"const instance = new EnvResolver();",
				"const result = await instance._resolveStringWithPlaceholders('plain', {}, 0);",
				"expect(result).toBeDefined();",
			},
			forbidden: []string{"rejects.toThrow()", "_resolveStringWithPlaceholders(undefined, 'test', undefined)"},
		},
		{
			name:     "vitest class return path ignores other throw branches",
			fileName: "env-resolver.js",
			source: `export class EnvResolver {
  async _resolveFieldUniversal(fieldValue, context, fieldType) {
    if (fieldType === 'bad') {
      throw new Error('bad field')
    }
    return fieldValue
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-env-field-return-1",
				Framework:       "vitest",
				Target:          "EnvResolver._resolveFieldUniversal",
				LineRange:       "6-6",
				GapType:         "return_path",
				TestName:        "covers EnvResolver field fallback",
				SuggestedInputs: []string{"设置 fieldValue 覆盖未执行分支", "设置 context 覆盖未执行分支", "设置 fieldType 覆盖未执行分支"},
			},
			wants: []string{
				"const instance = new EnvResolver();",
				"const result = await instance._resolveFieldUniversal('test', 'test', 'test');",
				"expect(result).toBe(('test'));",
			},
			forbidden: []string{"rejects.toThrow()", "to.equal(", "require('chai')"},
		},
		{
			name:     "vitest class constructor config branch",
			fileName: "dev-watcher.js",
			source: `export class DevWatcher {
  constructor(serverName, devConfig) {
    this.serverName = serverName
    this.devConfig = {
      enabled: devConfig.enabled ?? true,
      watch: devConfig.watch ?? [],
      cwd: devConfig.cwd
    }
    this.isWatching = false
  }

  async stop() {
    if (!this.isWatching) {
      return
    }
    this.isWatching = false
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-dev-watcher-stop-1",
				Framework:       "vitest",
				Target:          "DevWatcher.stop",
				LineRange:       "13-13",
				GapType:         "branch",
				TestName:        "covers DevWatcher stop not watching",
				SuggestedInputs: []string{"构造满足条件 `!this.isWatching` 的输入"},
			},
			wants: []string{
				"import { DevWatcher } from './dev-watcher';",
				"const instance = new DevWatcher('test-server', { enabled: true, watch: [], cwd: process.cwd() });",
				"const result = await instance.stop();",
				"expect(result).toBeDefined();",
			},
			forbidden: []string{"new DevWatcher();", "rejects.toThrow()"},
		},
		{
			name:     "vitest function wraps ordinary error",
			fileName: "errors.js",
			source: `export class MCPHubError extends Error {
  constructor(code, message, data = {}) {
    super(message)
    this.code = code
    this.data = data
  }
}

export function isMCPHubError(error) {
  return error instanceof MCPHubError
}

export function wrapError(error, code = 'UNEXPECTED_ERROR', data = {}) {
  if (isMCPHubError(error)) {
    return error
  }

  return new MCPHubError(error.code || code, error.message, {
    ...data,
    originalError: error,
  })
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-wrap-error-1",
				Framework:       "vitest",
				Target:          "wrapError",
				LineRange:       "15-15",
				GapType:         "branch",
				TestName:        "covers wrapError ordinary error",
				SuggestedInputs: []string{"构造满足条件 `isMCPHubError(error` 的输入", "设置 error 覆盖未执行分支"},
			},
			wants: []string{
				"import { describe, it, expect } from 'vitest';",
				"import { wrapError } from './errors';",
				"const result = wrapError(new Error('test error'), undefined, {});",
				"expect(typeof result).toBe('object');",
				"expect(result).not.toBeNull();",
			},
			forbidden: []string{"wrapError(undefined", "toBe('boolean')"},
		},
		{
			name:     "vitest class default exported instance",
			fileName: "logger.js",
			source: `class Logger {
  constructor(options = {}) {
    this.logLevel = options.logLevel || 'info'
    this.enableFileLogging = options.enableFileLogging !== false
    this.LOG_LEVELS = { error: 0, warn: 1, info: 2, debug: 3 }
  }

  setLogLevel(level) {
    if (this.LOG_LEVELS[level] !== undefined) {
      this.logLevel = level
    }
  }
}

const logger = new Logger({ logLevel: 'debug' })
export default logger
`,
			task: types.CoverageTestTask{
				ID:              "vitest-logger-level-1",
				Framework:       "vitest",
				Target:          "Logger.setLogLevel",
				LineRange:       "8-8",
				GapType:         "branch",
				TestName:        "covers Logger setLogLevel valid level",
				SuggestedInputs: []string{"构造满足条件 `this.LOG_LEVELS[level] !== undefined` 的输入", "设置 level 覆盖未执行分支"},
			},
			wants: []string{
				"import logger from './logger';",
				"const instance = logger;",
				"const result = instance.setLogLevel('info');",
				"// void function, verify no exception",
			},
			forbidden: []string{"import { Logger }", "new Logger({})", "setLogLevel(undefined)"},
		},
		{
			name:     "vitest class private method manual review",
			fileName: "token-store.js",
			source: `export class TokenStore {
  #readSecret(name) {
    if (name) {
      return 'secret'
    }
    return ''
  }

  get(name) {
    return this.#readSecret(name)
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-private-secret-1",
				Framework:       "vitest",
				Target:          "TokenStore.#readSecret",
				LineRange:       "3-3",
				GapType:         "branch",
				TestName:        "covers TokenStore private read secret",
				SuggestedInputs: []string{"构造满足条件 `name` 的输入"},
				AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
			},
			wants: []string{
				"import { TokenStore } from './token-store';",
				"it.skip('covers TokenStore private read secret'",
				"manual_review_private: TokenStore.#readSecret is a JavaScript private method",
				"public_entry_candidates: TokenStore.get",
			},
			forbidden: []string{"instance.#readSecret", "const result ="},
		},
		{
			name:     "vitest dev watcher private file change via start",
			fileName: "dev-watcher.js",
			source: `import chokidar from "chokidar"
import path from "path"

export class DevWatcher {
  constructor(serverName, devConfig) {
    this.serverName = serverName
    this.devConfig = { enabled: true, watch: [], cwd: devConfig.cwd, debounce: 500 }
    this.watcher = null
    this.debounceTimer = null
    this.changedFiles = new Set()
  }

  async start() {
    this.watcher = chokidar.watch([])
    this.watcher.on('change', (filePath) => this.#handleFileChange(filePath, 'change'))
  }

  #handleFileChange(filePath, eventType) {
    this.changedFiles.add(filePath)
    if (this.debounceTimer) {
      clearTimeout(this.debounceTimer)
    }
    this.debounceTimer = setTimeout(() => {
      const changedFilesArray = Array.from(this.changedFiles)
      const relativeFiles = changedFilesArray.map(file => {
        if (path.isAbsolute(file)) {
          return path.relative(this.devConfig.cwd, file)
        }
        return file
      })
      this.emit('filesChanged', { serverName: this.serverName, files: changedFilesArray, relativeFiles })
      this.changedFiles.clear()
    }, this.devConfig.debounce)
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-dev-watcher-private-1",
				Framework:       "vitest",
				Target:          "DevWatcher.#handleFileChange",
				LineRange:       "101-101",
				GapType:         "branch",
				TestName:        "covers DevWatcher private absolute file change",
				MissingBranches: []string{"未覆盖 if 分支: path.isAbsolute(file"},
				SuggestedInputs: []string{"构造满足条件 `path.isAbsolute(file` 的输入"},
				AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
			},
			wants: []string{
				"import { describe, it, expect, vi } from 'vitest';",
				"vi.mock('chokidar'",
				"import { DevWatcher } from './dev-watcher';",
				"vi.useFakeTimers();",
				"const instance = new DevWatcher('test-server', { enabled: true, watch: [], cwd });",
				"await instance.start();",
				"instance.watcher.emit('change', changedPath);",
				"await vi.advanceTimersByTimeAsync(500);",
				"expect(changes).toHaveLength(1);",
				"expect(changes[0].relativeFiles).toContain(path.join('src', 'app.js'));",
			},
			forbidden: []string{"instance.#handleFileChange", "it.skip(", "manual_review_private"},
		},
		{
			name:     "vitest config manager private diff via loadConfig",
			fileName: "config.js",
			source: `export class ConfigManager {
  constructor(configPathOrObject) {
    this.configPaths = null
    this.config = null
    this.#previousConfig = null
    if (configPathOrObject && typeof configPathOrObject === 'object') {
      this.config = configPathOrObject
      this.#previousConfig = JSON.parse(JSON.stringify(configPathOrObject))
    }
  }

  #diffConfigs(oldServers, newServers) {
    if (!newServers.old) {
      return { removed: ['old'], modified: [], added: [], unchanged: [], details: {} }
    }
    return { removed: [], modified: [], added: [], unchanged: ['old'], details: {} }
  }

  loadConfig() {
    return { changes: this.#diffConfigs(this.#previousConfig?.mcpServers, {}) }
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-config-private-diff-1",
				Framework:       "vitest",
				Target:          "ConfigManager.#diffConfigs",
				LineRange:       "3-3",
				GapType:         "branch",
				TestName:        "covers ConfigManager private diff",
				SuggestedInputs: []string{"构造满足条件 `!newServers[name]` 的输入"},
				AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
			},
			wants: []string{
				"import { ConfigManager } from './config';",
				"const fs = await import('node:fs/promises');",
				"await fs.writeFile(configPath, JSON.stringify({ mcpServers: {} }));",
				"const instance = new ConfigManager({ mcpServers: { old: { command: 'node' } } });",
				"instance.configPaths = [configPath];",
				"const result = await instance.loadConfig();",
				"expect(result.changes.removed).toContain('old');",
			},
			forbidden: []string{"instance.#diffConfigs", "it.skip(", "manual_review_private"},
		},
		{
			name:     "vitest storage manager internal class via default provider",
			fileName: "oauth-provider.js",
			source: `import fs from 'fs/promises'

let serversStorage = {}

class StorageManager {
  async init() {
    await fs.mkdir('/tmp', { recursive: true })
  }

  get(serverUrl) {
    if (!serversStorage[serverUrl]) {
      serversStorage[serverUrl] = { clientInfo: null, tokens: null, codeVerifier: null }
    }
    return serversStorage[serverUrl]
  }
}

const storage = new StorageManager()

export default class MCPHubOAuthProvider {
  constructor({ serverName, serverUrl, hubServerUrl }) {
    this.serverName = serverName
    this.serverUrl = serverUrl
    this.hubServerUrl = hubServerUrl
  }

  async tokens() {
    return storage.get(this.serverUrl).tokens
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-storage-get-1",
				Framework:       "vitest",
				Target:          "StorageManager.get",
				LineRange:       "3-3",
				GapType:         "branch",
				TestName:        "covers StorageManager get missing server",
				SuggestedInputs: []string{"构造满足条件 `!serversStorage[serverUrl]` 的输入"},
			},
			wants: []string{
				"import { describe, it, expect, vi } from 'vitest';",
				"vi.mock('fs/promises'",
				"vi.mock('./logger.js'",
				"const { default: MCPHubOAuthProvider } = await import('./oauth-provider');",
				"const provider = new MCPHubOAuthProvider({ serverName: 'test-server', serverUrl: 'https://example.com/mcp', hubServerUrl: 'http://localhost:3000' });",
				"await expect(provider.tokens()).resolves.toBeNull();",
			},
			forbidden: []string{"import './oauth-provider';", "import { StorageManager }", "new StorageManager()", "it.skip(", "manual_review_internal"},
		},
		{
			name:     "vitest storage manager init via module import",
			fileName: "oauth-provider.js",
			source: `import fs from 'fs/promises'
import logger from './logger.js'

let serversStorage = {}

class StorageManager {
  constructor() {
    this.path = '/tmp/oauth-storage.json'
  }

  async init() {
    try {
      await fs.mkdir('/tmp', { recursive: true })
      try {
        const data = await fs.readFile(this.path, 'utf8')
        serversStorage = JSON.parse(data)
      } catch (err) {
        if (err.code !== 'ENOENT') {
          logger.warn('Error reading storage')
        }
      }
    } catch (err) {
      logger.warn('Storage initialization error')
    }
  }
}

const storage = new StorageManager()
storage.init()

export default class MCPHubOAuthProvider {}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-storage-init-1",
				Framework:       "vitest",
				Target:          "StorageManager.init",
				LineRange:       "16-16",
				GapType:         "branch",
				TestName:        "covers StorageManager init read warning",
				MissingBranches: []string{"未覆盖 if 分支: err.code !== 'ENOENT'"},
				SuggestedInputs: []string{"构造满足条件 `err.code !== 'ENOENT'` 的输入"},
			},
			wants: []string{
				"import { describe, it, expect, vi } from 'vitest';",
				"vi.mock('fs/promises'",
				"const logger = await import('./logger.js');",
				"fs.default.readFile.mockRejectedValue(Object.assign(new Error('permission denied'), { code: 'EACCES' }));",
				"await import('./oauth-provider');",
				"expect(logger.default.warn).toHaveBeenCalledWith(expect.stringContaining('Error reading storage'));",
			},
			forbidden: []string{"import './oauth-provider';", "import { StorageManager }", "new StorageManager()", "it.skip(", "manual_review_internal"},
		},
		{
			name:     "vitest generic internal esm class manual review",
			fileName: "cache.js",
			source: `class LocalCache {
  get(key) {
    if (!this.values[key]) {
      this.values[key] = null
    }
    return this.values[key]
  }
}

export default class CacheFacade {
  constructor() {
    this.cache = new LocalCache()
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-internal-cache-1",
				Framework:       "vitest",
				Target:          "LocalCache.get",
				LineRange:       "3-3",
				GapType:         "branch",
				TestName:        "covers LocalCache get missing key",
				SuggestedInputs: []string{"构造满足条件 `!this.values[key]` 的输入"},
			},
			wants: []string{
				"import './cache';",
				"it.skip('covers LocalCache get missing key'",
				"manual_review_internal: LocalCache is not exported from this ES module",
			},
			forbidden: []string{"import { LocalCache }", "new LocalCache()"},
		},
		{
			name:     "vitest workspace cache update state existing entry",
			fileName: "workspace-cache.js",
			source: `export class WorkspaceCacheManager {
  constructor(options = {}) {
    this.port = options.port || null
  }

  async updateWorkspaceState(port, updates) {
    const workspaceKey = port.toString()
    await this._withLock(async () => {
      const cache = await this._readCache()
      if (cache[workspaceKey]) {
        cache[workspaceKey] = { ...cache[workspaceKey], ...updates }
        await this._writeCache(cache)
      }
    })
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-workspace-update-1",
				Framework:       "vitest",
				Target:          "WorkspaceCacheManager.updateWorkspaceState",
				LineRange:       "10-10",
				GapType:         "branch",
				TestName:        "covers WorkspaceCacheManager update existing workspace",
				MissingBranches: []string{"未覆盖 if 分支: cache[workspaceKey]"},
				SuggestedInputs: []string{"构造满足条件 `cache[workspaceKey]` 的输入", "设置 port 覆盖未执行分支", "设置 updates 覆盖未执行分支"},
			},
			wants: []string{
				"import { describe, it, expect, vi } from 'vitest';",
				"import { WorkspaceCacheManager } from './workspace-cache';",
				"const instance = new WorkspaceCacheManager({ port: 3000 });",
				"'3000': { state: 'active', activeConnections: 1, port: 3000 },",
				"instance._withLock = async (fn) => fn();",
				"instance._readCache = vi.fn().mockResolvedValue(cache);",
				"instance._writeCache = vi.fn().mockResolvedValue(undefined);",
				"await instance.updateWorkspaceState(3000, { state: 'shutting_down', activeConnections: 0 });",
				"expect(instance._writeCache).toHaveBeenCalledWith(expect.objectContaining({",
			},
			forbidden: []string{"updateWorkspaceState(undefined", "const result = await instance.updateWorkspaceState"},
		},
		{
			name:     "vitest sse add connection mocks express req res",
			fileName: "sse-manager.js",
			source: `export class SSEManager {
  constructor(options = {}) {
    this.connections = new Map()
    this.shutdownTimer = null
  }

  async addConnection(req, res) {
    const connection = {
      send: (event, data) => {
        if (res.writableEnded) return false
        res.write(event)
        return true
      }
    }
    res.setHeader('Content-Type', 'text/event-stream')
    req.on('close', () => {})
    if (this.shutdownTimer) {
      clearTimeout(this.shutdownTimer)
      this.shutdownTimer = null
    }
    this.connections.set('id', connection)
    return connection
  }
}
`,
			task: types.CoverageTestTask{
				ID:              "vitest-sse-shutdown-1",
				Framework:       "vitest",
				Target:          "SSEManager.addConnection",
				LineRange:       "17-17",
				GapType:         "branch",
				TestName:        "covers SSEManager addConnection shutdown timer",
				SuggestedInputs: []string{"构造满足条件 `this.shutdownTimer` 的输入", "设置 req 覆盖未执行分支", "设置 res 覆盖未执行分支"},
			},
			wants: []string{
				"const instance = Object.assign(new SSEManager({}), { shutdownTimer: setTimeout(() => {}, 1000) });",
				"const result = await instance.addConnection({ on: () => {} }, { writableEnded: false, setHeader: () => {}, write: () => {}, end: () => {} });",
				"expect(result).toBeDefined();",
			},
			forbidden: []string{"undefined, { json", "res.setHeader is not a function"},
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

	code := genJSClassTestForCoverageTask(cls, &task, "./widget")
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

func TestJSClassCoverageTaskGeneratesIP2RegionStatefulClassCases(t *testing.T) {
	tests := []struct {
		name      string
		cls       jsClassInfo
		task      types.CoverageTestTask
		wants     []string
		forbidden []string
	}{
		{
			name: "version ipCompare supplies compare callback",
			cls: jsClassInfo{
				Name: "Version",
				ConstructorParams: []jsParamInfo{
					{Name: "id"},
					{Name: "name"},
					{Name: "bytes"},
					{Name: "indexSize"},
					{Name: "ipCompareFunc"},
				},
				Methods: []jsFuncInfo{{
					Name: "ipCompare",
					Params: []jsParamInfo{
						{Name: "ip1"},
						{Name: "ip2"},
					},
					Analysis: jsFuncAnalysis{HasReturn: true, ReturnType: "unknown", Returns: []string{"this.ipCompareFunc(ip1, ip2, 0)"}},
				}},
			},
			task: types.CoverageTestTask{
				ID:        "jest-version-compare",
				Framework: "jest",
				Target:    "Version.ipCompare",
				LineRange: "268-268",
				GapType:   "return_path",
				TestName:  "covers version compare",
			},
			wants: []string{
				"describe('Version'",
				"describe('ipCompare'",
				"const calls = [];",
				"const compare = (...args) => {",
				"const instance = new Version(4, 'IPv4', 4, 14, compare);",
				"const result = instance.ipCompare(left, right);",
				"expect(result).toBe(0);",
				"expect(calls).toEqual([[left, right, 0]]);",
			},
			forbidden: []string{"new Version(1, 'test-server', undefined"},
		},
		{
			name: "searcher search zero pointer branch uses cBuffer",
			cls: jsClassInfo{
				Name:              "Searcher",
				ConstructorParams: []jsParamInfo{{Name: "version"}, {Name: "dbPath"}, {Name: "vectorIndex"}, {Name: "cBuffer"}},
				Methods: []jsFuncInfo{{
					Name:     "search",
					IsAsync:  true,
					Params:   []jsParamInfo{{Name: "ip"}},
					Analysis: jsFuncAnalysis{HasReturn: true, ReturnType: "string", Returns: []string{`""`}},
				}},
			},
			task: types.CoverageTestTask{
				ID:        "jest-searcher-zero",
				Framework: "jest",
				Target:    "Searcher.search",
				LineRange: "72-72",
				GapType:   "return_path",
				TestName:  "covers zero pointer",
			},
			wants: []string{
				"const version = { name: 'IPv4', bytes: 4, indexSize: 14, ipSubCompare: () => 0 };",
				"const cBuffer = Buffer.alloc(264);",
				"const instance = new Searcher(version, null, null, cBuffer);",
				"const result = await instance.search('0.0.0.0');",
				"expect(result).toBe('');",
			},
			forbidden: []string{"new Searcher(undefined, 'test'"},
		},
		{
			name: "searcher search empty match branch seeds segment pointers",
			cls: jsClassInfo{
				Name:              "Searcher",
				ConstructorParams: []jsParamInfo{{Name: "version"}, {Name: "dbPath"}, {Name: "vectorIndex"}, {Name: "cBuffer"}},
				Methods: []jsFuncInfo{{
					Name:     "search",
					IsAsync:  true,
					Params:   []jsParamInfo{{Name: "ip"}},
					Analysis: jsFuncAnalysis{HasReturn: true, ReturnType: "string", Returns: []string{`""`}},
				}},
			},
			task: types.CoverageTestTask{
				ID:        "jest-searcher-empty-match",
				Framework: "jest",
				Target:    "Searcher.search",
				LineRange: "100-100",
				GapType:   "return_path",
				TestName:  "covers empty match",
			},
			wants: []string{
				"const cBuffer = Buffer.alloc(278);",
				"cBuffer.writeUInt32LE(264, 256);",
				"cBuffer.writeUInt32LE(264, 260);",
				"expect(result).toBe('');",
			},
		},
		{
			name: "searcher read incomplete read mocks fs after construction",
			cls: jsClassInfo{
				Name:              "Searcher",
				ConstructorParams: []jsParamInfo{{Name: "version"}, {Name: "dbPath"}, {Name: "vectorIndex"}, {Name: "cBuffer"}},
				Methods: []jsFuncInfo{{
					Name:   "read",
					Params: []jsParamInfo{{Name: "offset"}, {Name: "buff"}, {Name: "stats"}},
					Analysis: jsFuncAnalysis{
						Throws: true,
					},
				}},
			},
			task: types.CoverageTestTask{
				ID:        "jest-searcher-read",
				Framework: "jest",
				Target:    "Searcher.read",
				LineRange: "122-122",
				GapType:   "error_path",
				TestName:  "covers incomplete read",
			},
			wants: []string{
				"const fs = await import('fs');",
				"fs.default.readSync = () => 0;",
				"Object.assign(Object.create(Searcher.prototype), { cBuffer: null, handle: 1, ioCount: 0 });",
				"expect(() => instance.read(0, Buffer.alloc(4))).toThrow('incomplete read');",
				"fs.default.readSync = originalReadSync;",
			},
			forbidden: []string{"new Searcher(undefined, 'test'"},
		},
		{
			name: "searcher toString uses memory buffer constructor path",
			cls: jsClassInfo{
				Name:              "Searcher",
				ConstructorParams: []jsParamInfo{{Name: "version"}, {Name: "dbPath"}, {Name: "vectorIndex"}, {Name: "cBuffer"}},
				Methods: []jsFuncInfo{{
					Name:     "toString",
					Analysis: jsFuncAnalysis{HasReturn: true, ReturnType: "string", Returns: []string{"`json`"}},
				}},
			},
			task: types.CoverageTestTask{
				ID:        "jest-searcher-string",
				Framework: "jest",
				Target:    "Searcher.toString",
				LineRange: "137-137",
				GapType:   "return_path",
				TestName:  "covers searcher string",
			},
			wants: []string{
				"const version = { name: 'IPv4' };",
				"const instance = new Searcher(version, null, null, Buffer.alloc(8));",
				"const result = instance.toString();",
				"expect(result).toContain('IPv4');",
				"expect(result).toContain('\"cBuffer\": 8');",
			},
			forbidden: []string{"new Searcher(undefined, 'test'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := genJSClassTestForCoverageTask(tt.cls, &tt.task, "./searcher")
			assertGeneratedJS(t, code, tt.wants, tt.forbidden)
		})
	}
}

func TestJSCoverageTaskRoutesCodexConfigHelperThroughPublicRun(t *testing.T) {
	funcs := []jsFuncInfo{{Name: "flattenConfigOverrides", IsExported: false}, {Name: "toTomlValue", IsExported: false}}
	classes := []jsClassInfo{{
		Name:       "CodexExec",
		IsExported: true,
		Methods: []jsFuncInfo{{
			Name:    "run",
			IsAsync: true,
			Params:  []jsParamInfo{{Name: "args"}},
			Analysis: jsFuncAnalysis{
				HasReturn: true,
			},
		}},
	}}
	task := types.CoverageTestTask{
		ID:        "jest-flatten-1",
		Framework: "jest",
		Target:    "flattenConfigOverrides",
		LineRange: "271-274",
		GapType:   "branch",
		TestName:  "covers nested empty config",
	}

	filteredFuncs, filteredClasses := filterJSTargetsForCoverageTask(funcs, classes, &task)
	if len(filteredFuncs) != 0 {
		t.Fatalf("unexported helper should not be imported directly: %+v", filteredFuncs)
	}
	if len(filteredClasses) != 1 || filteredClasses[0].Name != "CodexExec" || len(filteredClasses[0].Methods) != 1 || filteredClasses[0].Methods[0].Name != "run" {
		t.Fatalf("expected CodexExec.run public entry, got %+v", filteredClasses)
	}

	code := genJSClassTestForCoverageTask(filteredClasses[0], &task, "../src/exec")
	assertGeneratedJS(t, code, []string{
		"const { CodexExec } = await import('../src/exec');",
		"const instance = new CodexExec('codex', {}, { sandbox_workspace_write: {} });",
		"await consumeTestloopCodexExec(instance.run({ input: 'hi' }));",
		"expect(commandArgs).toContain('sandbox_workspace_write={}');",
	}, []string{
		"flattenConfigOverrides(",
		"import { flattenConfigOverrides }",
	})

	task.Target = "toTomlValue"
	task.LineRange = "296-298"
	task.TestName = "covers finite number validation"
	filteredFuncs, filteredClasses = filterJSTargetsForCoverageTask(funcs, classes, &task)
	if len(filteredFuncs) != 0 {
		t.Fatalf("unexported toTomlValue should not be imported directly: %+v", filteredFuncs)
	}
	code = genJSClassTestForCoverageTask(filteredClasses[0], &task, "../src/exec")
	assertGeneratedJS(t, code, []string{
		"const instance = new CodexExec('codex', {}, { retries: Infinity });",
		"rejects.toThrow('finite number');",
		"expect(spawnMock).not.toHaveBeenCalled();",
	}, []string{
		"toTomlValue(",
		"import { toTomlValue }",
	})
}

func TestGenerateJSCoverageTaskCodexExecUsesDynamicJestMock(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "exec.ts")
	src := `import { spawn } from "node:child_process";

export class CodexExec {
  constructor(executablePath: string | null = null) {}
  async *run(args: { input: string }): AsyncGenerator<string> {
    const child = spawn("codex", []);
    let spawnError: unknown | null = null;
    child.once("error", (err) => (spawnError = err));
    if (spawnError) throw spawnError;
  }
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	code, err := GenerateJavaScriptTestsForCoverageTask(srcPath, &types.CoverageTestTask{
		ID:        "jest-run-spawn-error",
		Framework: "jest",
		Target:    "CodexExec.run",
		LineRange: "8-8",
		GapType:   "branch",
		TestName:  "covers spawn error",
	})
	if err != nil {
		t.Fatalf("GenerateJavaScriptTestsForCoverageTask() error = %v", err)
	}
	assertGeneratedJS(t, code, []string{
		"// @ts-nocheck",
		"import * as child_process from 'node:child_process';",
		"import { jest } from '@jest/globals';",
		"jest.mock('node:child_process'",
		"const { CodexExec } = await import('./exec');",
		"const instance = new CodexExec('codex');",
		"instance.run({ input: 'hi' })",
		"rejects.toThrow('spawn failed');",
	}, []string{
		"import { CodexExec } from './exec';",
		"instance.run([])",
	})
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
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Pick<User, 'userId' | 'email'>>", typeDecls); !ok || got != "{ userId: 1, email: 'user@example.com' }" {
		t.Fatalf("pick payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Readonly<Pick<User, \"email\" | \"status\">>>", typeDecls); !ok || got != "{ email: 'user@example.com', status: 'active' }" {
		t.Fatalf("readonly pick payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Pick<User, keyof User>>", typeDecls); ok || got != "" {
		t.Fatalf("unsupported pick payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Omit<User, 'manager' | 'displayName'>>", typeDecls); !ok || got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z' }" {
		t.Fatalf("omit payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Readonly<Omit<User, \"email\" | \"status\">>>", typeDecls); !ok || got != "{ userId: 1, createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }" {
		t.Fatalf("readonly omit payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Omit<User, 'unknown'>>", typeDecls); !ok || got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }" {
		t.Fatalf("unknown omit payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Omit<User, 'userId' | 'email' | 'status' | 'createdAt' | 'displayName' | 'manager'>>", typeDecls); !ok || got != "{}" {
		t.Fatalf("empty omit payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Omit<User, keyof User>>", typeDecls); ok || got != "" {
		t.Fatalf("unsupported omit payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<{ owner: Pick<User, 'userId' | 'email'> }>", typeDecls); !ok || got != "{ owner: { userId: 1, email: 'user@example.com' } }" {
		t.Fatalf("object pick field payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<{ owner: Omit<User, 'manager' | 'displayName'> }>", typeDecls); !ok || got != "{ owner: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z' } }" {
		t.Fatalf("object omit field payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Record<string, User>>", typeDecls); !ok || got != "{ key: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} } }" {
		t.Fatalf("record string payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Record<'primary' | 'secondary', User>>", typeDecls); !ok || got != "{ primary: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }, secondary: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} } }" {
		t.Fatalf("record literal payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Record<number, User>>", typeDecls); ok || got != "" {
		t.Fatalf("unsupported record payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<{ owners: Record<'primary' | 'secondary', User> }>", typeDecls); !ok || got != "{ owners: { primary: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }, secondary: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} } } }" {
		t.Fatalf("object record literal field payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<{ directory: Record<string, Pick<User, 'userId' | 'email'>> }>", typeDecls); !ok || got != "{ directory: { key: { userId: 1, email: 'user@example.com' } } }" {
		t.Fatalf("object record projection field payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<{ owners: Record<number, User> }>", typeDecls); !ok || got != "{ owners: {} }" {
		t.Fatalf("object unsupported record field payload = %q, %v", got, ok)
	}
	typeDecls["AuditFields"] = "{ traceId: string; page: number }"
	typeDecls["AuditedUser"] = "User & AuditFields"
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<User & AuditFields>", typeDecls); !ok || got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {}, traceId: 'id-1', page: 1 }" {
		t.Fatalf("intersection payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<AuditedUser>", typeDecls); !ok || got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {}, traceId: 'id-1', page: 1 }" {
		t.Fatalf("intersection alias payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<{ id: number } & { email: string }>", typeDecls); !ok || got != "{ id: 1, email: 'user@example.com' }" {
		t.Fatalf("inline intersection payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<User & string>", typeDecls); ok || got != "" {
		t.Fatalf("unsupported intersection payload = %q, %v", got, ok)
	}
	typeDecls["ApiResponse"] = "{ data: User; meta: Meta; count: number }"
	typeDecls["ResponseData"] = "ApiResponse['data']"
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<ApiResponse['data']>", typeDecls); !ok || got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }" {
		t.Fatalf("indexed access payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<ResponseData>", typeDecls); !ok || got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }" {
		t.Fatalf("indexed access alias payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<{ data: { id: number; email: string } }['data']>", typeDecls); !ok || got != "{ id: 1, email: 'user@example.com' }" {
		t.Fatalf("inline indexed access payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<{ data: ApiResponse['data'] }>", typeDecls); !ok || got != "{ data: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} } }" {
		t.Fatalf("object indexed access field payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<ApiResponse['data' | 'meta']>", typeDecls); ok || got != "" {
		t.Fatalf("unsupported indexed union key payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<ApiResponse[keyof ApiResponse]>", typeDecls); ok || got != "" {
		t.Fatalf("unsupported indexed keyof payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<ApiResponse['missing']>", typeDecls); ok || got != "" {
		t.Fatalf("missing indexed access payload = %q, %v", got, ok)
	}
	if got := jsMockValueForTSTypeWithDecls("owner", "null | User", typeDecls); got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }" {
		t.Fatalf("nullable owner value = %q", got)
	}
	if got := jsMockValueForTSTypeWithDecls("owner", "Readonly<User>", typeDecls); got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }" {
		t.Fatalf("readonly owner value = %q", got)
	}
	if got := jsMockValueForTSTypeWithDecls("owner", "Pick<User, 'userId'>", typeDecls); got != "{ userId: 1 }" {
		t.Fatalf("pick owner value = %q", got)
	}
	if got := jsMockValueForTSTypeWithDecls("owner", "Omit<User, 'manager'>", typeDecls); got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test' }" {
		t.Fatalf("omit owner value = %q", got)
	}
	if got := jsMockValueForTSTypeWithDecls("owners", "Record<'primary', User>", typeDecls); got != "{ primary: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} } }" {
		t.Fatalf("record owners value = %q", got)
	}
	if got := jsMockValueForTSTypeWithDecls("owner", "User & AuditFields", typeDecls); got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {}, traceId: 'id-1', page: 1 }" {
		t.Fatalf("intersection owner value = %q", got)
	}
	if got := jsMockValueForTSTypeWithDecls("data", "ApiResponse['data']", typeDecls); got != "{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }" {
		t.Fatalf("indexed access data value = %q", got)
	}
	typeDecls["Users"] = "ReadonlyArray<User | null>"
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Users>", typeDecls); !ok || got != "[{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }]" {
		t.Fatalf("array alias payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Readonly<Users>>", typeDecls); !ok || got != "[{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }]" {
		t.Fatalf("readonly utility array payload = %q, %v", got, ok)
	}
	typeDecls["UserMap"] = "Record<string, User>"
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<UserMap>", typeDecls); !ok || got != "{ key: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} } }" {
		t.Fatalf("record alias payload = %q, %v", got, ok)
	}
	typeDecls["Meta"] = "{ total: number; nextUrl?: string | null }"
	typeDecls["Directory"] = "Readonly<Record<'primary' | 'secondary', ApiResponse['data'] & AuditFields>>"
	typeDecls["DirectoryEnvelope"] = "Omit<{ directory: Directory; meta: ApiResponse['meta']; debug: string }, 'debug'>"
	typeDecls["DirectorySummary"] = "Pick<DirectoryEnvelope, 'directory' | 'meta'>"
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<DirectorySummary>", typeDecls); !ok || got != "{ directory: { primary: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {}, traceId: 'id-1', page: 1 }, secondary: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {}, traceId: 'id-1', page: 1 } }, meta: { total: 1, nextUrl: 'https://example.com' } }" {
		t.Fatalf("composed directory payload = %q, %v", got, ok)
	}
	typeDecls["ApiEnvelope<T>"] = "{ data: T; meta: Meta }"
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<ApiEnvelope<User>>", typeDecls); !ok || got != "{ data: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }, meta: { total: 1, nextUrl: 'https://example.com' } }" {
		t.Fatalf("generic envelope payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<ApiEnvelope<Pick<User, 'userId' | 'email'>>>", typeDecls); !ok || got != "{ data: { userId: 1, email: 'user@example.com' }, meta: { total: 1, nextUrl: 'https://example.com' } }" {
		t.Fatalf("generic projection payload = %q, %v", got, ok)
	}
	typeDecls["Pair<T,U>"] = "{ first: T; second: U }"
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Pair<User, Meta>>", typeDecls); !ok || got != "{ first: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }, second: { total: 1, nextUrl: 'https://example.com' } }" {
		t.Fatalf("generic pair payload = %q, %v", got, ok)
	}
	typeDecls["Constrained<T extends User>"] = "{ data: T }"
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<Constrained<User>>", typeDecls); ok || got != "" {
		t.Fatalf("unsupported constrained generic payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<{ summary: DirectorySummary }>", typeDecls); !ok || got != "{ summary: { directory: { primary: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {}, traceId: 'id-1', page: 1 }, secondary: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {}, traceId: 'id-1', page: 1 } }, meta: { total: 1, nextUrl: 'https://example.com' } } }" {
		t.Fatalf("object composed projection field payload = %q, %v", got, ok)
	}
	typeDecls["RecursiveDirectory"] = "Record<'primary', User & { reports: User[] }>"
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<RecursiveDirectory>", typeDecls); !ok || got != "{ primary: { userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {}, reports: [{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }] } }" {
		t.Fatalf("recursive directory payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<{ reports: User[] }>", typeDecls); !ok || got != "{ reports: [{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }] }" {
		t.Fatalf("object array field payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<readonly User[]>", typeDecls); !ok || got != "[{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }]" {
		t.Fatalf("readonly array payload = %q, %v", got, ok)
	}
	typeDecls["UserTuple"] = "readonly [user: User, meta?: Meta]"
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<UserTuple>", typeDecls); !ok || got != "[{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }, { total: 1, nextUrl: 'https://example.com' }]" {
		t.Fatalf("tuple alias payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<readonly [User, ...Meta[]]>", typeDecls); !ok || got != "[{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }, { total: 1, nextUrl: 'https://example.com' }]" {
		t.Fatalf("rest tuple payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<{ pair: [User, Meta] }>", typeDecls); !ok || got != "{ pair: [{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }, { total: 1, nextUrl: 'https://example.com' }] }" {
		t.Fatalf("object tuple field payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<{ pair: readonly [user: User, meta?: Meta] }>", typeDecls); !ok || got != "{ pair: [{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }, { total: 1, nextUrl: 'https://example.com' }] }" {
		t.Fatalf("object readonly tuple field payload = %q, %v", got, ok)
	}
	if got, ok := jsMockPayloadFromTSTypeWithDecls("Promise<{ pair: readonly [User, ...Meta[]] }>", typeDecls); !ok || got != "{ pair: [{ userId: 1, email: 'user@example.com', status: 'active', createdAt: '2026-01-01T00:00:00.000Z', displayName: 'test', manager: {} }, { total: 1, nextUrl: 'https://example.com' }] }" {
		t.Fatalf("object rest tuple field payload = %q, %v", got, ok)
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
