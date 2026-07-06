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
const path = require('path');

export type User = { name: string };
export class Greeter {
  greet(name) {
    return trim(name);
  }
}

export function add(a, b) {
  return a + b;
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
	assertContains(t, ctx.Imports, "const path = require('path')")
	assertContains(t, ctx.Types, "User")
	assertContains(t, ctx.Types, "Greeter")

	target := findTarget(ctx.Targets, "greet")
	if target == nil {
		t.Fatalf("greet target not found: %+v", ctx.Targets)
	}
	if target.ClassName != "Greeter" || target.Kind != "method" {
		t.Fatalf("unexpected greet target: %+v", target)
	}
	assertContains(t, target.ReturnExpressions, "trim(name)")
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
