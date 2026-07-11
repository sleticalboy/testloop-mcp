package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sleticalboy/testloop-mcp/types"
)

func TestBuildGenerationContext_PythonMetadata(t *testing.T) {
	src := `import math
from decimal import Decimal

class Calculator:
    def add(self, a, b):
        return a + b

def normalize(value):
    if value is None:
        raise ValueError("value is required")
    return Decimal(value)
`
	srcPath := filepath.Join(t.TempDir(), "calc.py")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := BuildGenerationContext(srcPath)
	if ctx == nil {
		t.Fatal("expected generation context")
	}
	if ctx.Language != "python" || ctx.Framework != "pytest" {
		t.Fatalf("unexpected metadata: %+v", ctx)
	}
	assertContains(t, ctx.Imports, "import math")
	assertContains(t, ctx.Imports, "from decimal import Decimal")
	assertContains(t, ctx.Types, "Calculator")

	target := findTarget(ctx.Targets, "normalize")
	if target == nil {
		t.Fatalf("normalize target not found: %+v", ctx.Targets)
	}
	assertContains(t, target.ReturnExpressions, "Decimal(value)")
	if !target.HasErrorPath {
		t.Fatal("expected normalize to expose error path")
	}
	assertContains(t, target.BoundaryCases, "value=None")
}

func TestBuildGenerationContext_JSMetadata(t *testing.T) {
	src := `import { trim } from './text';
import type { ExternalUser } from './types';
const path = require('path');

export type User = { name: string };
type Box<T> = { data: T };
type Constrained<T extends User> = { data: T };
export class Greeter {
  greet(name) {
    return trim(name);
  }
}

export function add(a, b) {
  return a + b;
}

export async function loadBox(response: Response): Promise<Box<User>> {
  return await response.json();
}

export async function loadExternal(response: Response): Promise<ExternalUser> {
  return await response.json();
}

export async function loadImportedBox(response: Response): Promise<Box<ExternalUser>> {
  return await response.json();
}

export async function loadConstrained(response: Response): Promise<Constrained<User>> {
  return await response.json();
}
`
	srcPath := filepath.Join(t.TempDir(), "greeter.ts")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := BuildGenerationContext(srcPath)
	if ctx == nil {
		t.Fatal("expected generation context")
	}
	if ctx.Language != "typescript" || ctx.Framework != "jest" {
		t.Fatalf("unexpected metadata: %+v", ctx)
	}
	assertContains(t, ctx.Imports, "import { trim } from './text'")
	assertContains(t, ctx.Imports, "import type { ExternalUser } from './types'")
	assertContains(t, ctx.Imports, "const path = require('path')")
	assertContains(t, ctx.Types, "User")
	assertContains(t, ctx.Types, "Box")
	assertContains(t, ctx.Types, "Constrained")
	assertContains(t, ctx.Types, "Greeter")

	target := findTarget(ctx.Targets, "greet")
	if target == nil {
		t.Fatalf("greet target not found: %+v", ctx.Targets)
	}
	if target.ClassName != "Greeter" || target.Kind != "method" {
		t.Fatalf("unexpected greet target: %+v", target)
	}
	assertContains(t, target.ReturnExpressions, "trim(name)")

	loadBox := findTarget(ctx.Targets, "loadBox")
	if loadBox == nil {
		t.Fatalf("loadBox target not found: %+v", ctx.Targets)
	}
	if loadBox.ReturnTypeExpr != "Promise<Box<User>>" {
		t.Fatalf("unexpected loadBox return type expr: %+v", loadBox)
	}
	if len(loadBox.PayloadNotes) != 0 {
		t.Fatalf("supported same-file generic should not expose payload notes: %+v", loadBox.PayloadNotes)
	}

	loadExternal := findTarget(ctx.Targets, "loadExternal")
	if loadExternal == nil {
		t.Fatalf("loadExternal target not found: %+v", ctx.Targets)
	}
	assertContains(t, loadExternal.PayloadNotes, "return annotation ExternalUser is not declared in the same source file; static payload falls back to { ok: true }")
	assertContains(t, loadExternal.PayloadNotes, "return annotation references imported type ExternalUser from './types'; read candidate source files: types.ts, types.tsx, types.d.ts, types.js, types.jsx, types.mjs, types.cjs, types/index.ts, types/index.tsx, types/index.d.ts, types/index.js, types/index.jsx, types/index.mjs, types/index.cjs")

	loadImportedBox := findTarget(ctx.Targets, "loadImportedBox")
	if loadImportedBox == nil {
		t.Fatalf("loadImportedBox target not found: %+v", ctx.Targets)
	}
	assertContains(t, loadImportedBox.PayloadNotes, "return annotation references imported type ExternalUser from './types'; read candidate source files: types.ts, types.tsx, types.d.ts, types.js, types.jsx, types.mjs, types.cjs, types/index.ts, types/index.tsx, types/index.d.ts, types/index.js, types/index.jsx, types/index.mjs, types/index.cjs")

	loadConstrained := findTarget(ctx.Targets, "loadConstrained")
	if loadConstrained == nil {
		t.Fatalf("loadConstrained target not found: %+v", ctx.Targets)
	}
	assertContains(t, loadConstrained.PayloadNotes, "generic return annotation Constrained<User> uses constrained or defaulted type parameters; static payload falls back to { ok: true }")
}

func TestBuildGenerationContextCoverageTaskForStaticLanguages(t *testing.T) {
	task := types.CoverageTestTask{
		ID:        "go-1",
		Framework: "go-test",
		Target:    "Add",
	}

	ctx := BuildGenerationContextWithOptions(filepath.Join(t.TempDir(), "calc.go"), GenerateTestsOptions{CoverageTask: &task})

	if ctx == nil {
		t.Fatal("expected coverage task context")
	}
	if ctx.Language != "go" || ctx.Framework != "go-test" || ctx.CoverageTask == nil {
		t.Fatalf("unexpected context: %+v", ctx)
	}
}

func TestBuildGenerationContextGoCoverageFallbackNotes(t *testing.T) {
	src := `package sample

type User struct {
	Name string
}

func UserName(user *User) string {
	if user != nil {
		return user.Name
	}
	return "missing"
}
`
	srcPath := filepath.Join(t.TempDir(), "user.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-branch-UserName",
		Framework:       "go-test",
		Target:          "UserName",
		LineRange:       "8-10",
		GapType:         "branch",
		SuggestedInputs: []string{"构造满足条件 `user != nil` 的输入"},
		AssertionFocus:  []string{"断言分支返回值"},
	}

	ctx := BuildGenerationContextWithOptions(srcPath, GenerateTestsOptions{CoverageTask: &task})
	if ctx == nil || ctx.Language != "go" || ctx.Framework != "go-test" || ctx.CoverageTask == nil {
		t.Fatalf("unexpected Go context: %+v", ctx)
	}
	target := findTarget(ctx.Targets, "UserName")
	if target == nil {
		t.Fatalf("UserName target not found: %+v", ctx.Targets)
	}
	assertContains(t, target.Params, "user *User")
	assertContains(t, target.ReturnExpressions, "user.Name")
	assertContains(t, target.ReturnExpressions, `"missing"`)
	assertContains(t, target.BoundaryCases, "user != nil")
	assertContains(t, target.PayloadNotes, `Static generator cannot infer exact coverage case: branch "user != nil" returns "user.Name", which needs manual expected value review.`)
}

func TestBuildGenerationContextGoAndCompoundCondition(t *testing.T) {
	src := `package sample

func Score(a, b int) int {
	if a > 0 && b > 0 {
		return 1
	}
	return 0
}
`
	srcPath := filepath.Join(t.TempDir(), "score.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-branch-Score",
		Framework:       "go-test",
		Target:          "Score",
		LineRange:       "4-6",
		GapType:         "branch",
		SuggestedInputs: []string{"构造满足条件 `a > 0 && b > 0` 的输入"},
		AssertionFocus:  []string{"断言分支返回值"},
	}

	ctx := BuildGenerationContextWithOptions(srcPath, GenerateTestsOptions{CoverageTask: &task})
	if ctx == nil || ctx.Language != "go" || ctx.Framework != "go-test" || ctx.CoverageTask == nil {
		t.Fatalf("unexpected Go context: %+v", ctx)
	}
	target := findTarget(ctx.Targets, "Score")
	if target == nil {
		t.Fatalf("Score target not found: %+v", ctx.Targets)
	}
	assertContains(t, target.Params, "a int")
	assertContains(t, target.Params, "b int")
	assertContains(t, target.ReturnExpressions, "1")
	assertContains(t, target.ReturnExpressions, "0")
	assertContains(t, target.BoundaryCases, "a > 0 && b > 0")
	if len(target.PayloadNotes) != 0 {
		t.Fatalf("did not expect fallback notes for supported && compound condition: %+v", target.PayloadNotes)
	}
}

func TestBuildGenerationContextReturnsNilForMissingOrEmptyTargets(t *testing.T) {
	if ctx := BuildGenerationContext(filepath.Join(t.TempDir(), "missing.ts")); ctx != nil {
		t.Fatalf("expected nil context for missing source, got %+v", ctx)
	}

	srcPath := filepath.Join(t.TempDir(), "empty.py")
	if err := os.WriteFile(srcPath, []byte("import math\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if ctx := BuildGenerationContext(srcPath); ctx != nil {
		t.Fatalf("expected nil context when no targets exist, got %+v", ctx)
	}
}

func TestLanguageNameForPath(t *testing.T) {
	tests := map[string]string{
		"calc.go":       "go",
		"lib.rs":        "rust",
		"Example.java":  "java",
		"calc.py":       "python",
		"widget.tsx":    "typescript",
		"component.jsx": "javascript",
		"module.mjs":    "javascript",
		"unknown.txt":   "",
	}

	for path, want := range tests {
		t.Run(path, func(t *testing.T) {
			if got := languageNameForPath(path); got != want {
				t.Fatalf("languageNameForPath(%q) = %q, want %q", path, got, want)
			}
		})
	}
}

func TestGenerationContextTargetHelpers(t *testing.T) {
	js := jsTarget(jsFuncInfo{
		Name:      "formatText",
		ClassName: "Formatter",
		IsAsync:   true,
		Params: []jsParamInfo{
			{Name: "text"},
			{Name: "prefix", HasDefault: true},
			{Name: "args", IsRest: true},
		},
		Analysis: jsFuncAnalysis{
			Throws:     true,
			Returns:    []string{"prefix + text"},
			Boundaries: []jsBoundary{{Param: "prefix", Value: "'>'"}},
		},
	}, "method", nil, "")
	if js.Name != "formatText" || js.Kind != "method" || js.ClassName != "Formatter" || !js.Async || !js.HasErrorPath {
		t.Fatalf("unexpected JS target metadata: %+v", js)
	}
	if js.ReturnType != "unknown" {
		t.Fatalf("empty JS return type should become unknown: %+v", js)
	}
	assertContains(t, js.Params, "text")
	assertContains(t, js.Params, "prefix?")
	assertContains(t, js.Params, "...args")
	assertContains(t, js.ReturnExpressions, "prefix + text")
	assertContains(t, js.BoundaryCases, "prefix='>'")

	py := pyTarget(pyFuncInfo{
		Name:      "format_text",
		ClassName: "Formatter",
		IsAsync:   true,
		Params: []pyParamInfo{
			{Name: "text"},
			{Name: "prefix", HasDefault: true},
			{Name: "args", IsArgs: true},
			{Name: "kwargs", IsKwargs: true},
		},
		Analysis: pyFuncAnalysis{
			Raises:     true,
			Returns:    []string{"prefix + text"},
			Boundaries: []pyBoundary{{Param: "prefix", Value: "'>'"}},
		},
	}, "method")
	if py.Name != "format_text" || py.Kind != "method" || py.ClassName != "Formatter" || !py.Async || !py.HasErrorPath {
		t.Fatalf("unexpected Python target metadata: %+v", py)
	}
	if py.ReturnType != "unknown" {
		t.Fatalf("empty Python return type should become unknown: %+v", py)
	}
	assertContains(t, py.Params, "text")
	assertContains(t, py.Params, "prefix?")
	assertContains(t, py.Params, "*args")
	assertContains(t, py.Params, "**kwargs")
	assertContains(t, py.ReturnExpressions, "prefix + text")
	assertContains(t, py.BoundaryCases, "prefix='>'")
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("expected %q in %+v", want, values)
}

func findTarget(targets []types.TestTarget, name string) *types.TestTarget {
	for i := range targets {
		if targets[i].Name == name {
			return &targets[i]
		}
	}
	return nil
}
