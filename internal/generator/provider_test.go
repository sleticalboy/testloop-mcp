package generator

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewTestProviderAutoFallsBackToStaticWhenCommandMissing(t *testing.T) {
	t.Setenv(EnvLLMProviderCommand, "")

	provider, err := NewTestProvider("auto")
	if err != nil {
		t.Fatalf("NewTestProvider(auto) error = %v", err)
	}
	if provider.Name() != "static" {
		t.Fatalf("provider.Name() = %q, want static", provider.Name())
	}
}

func TestNewTestProviderLLMRequiresCommand(t *testing.T) {
	t.Setenv(EnvLLMProviderCommand, "")

	_, err := NewTestProvider("llm")
	if err == nil || !strings.Contains(err.Error(), EnvLLMProviderCommand) {
		t.Fatalf("NewTestProvider(llm) error = %v, want missing command error", err)
	}
}

func TestGenerateTestsWithProviderUsesExternalLLMCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake provider is unix-only")
	}

	srcPath := writeProviderSource(t)
	command := writeFakeProviderCommand(t, `#!/bin/sh
cat <<'EOF'
{"code":"from calc import add\n\n\ndef test_generated_by_llm():\n    assert add(1, 2) == 3\n"}
EOF
`)

	code, err := GenerateTestsWithProvider(context.Background(), srcPath, ExternalLLMProvider{Command: command})
	if err != nil {
		t.Fatalf("GenerateTestsWithProvider() error = %v", err)
	}
	if !strings.Contains(code, "test_generated_by_llm") {
		t.Fatalf("expected llm provider output, got:\n%s", code)
	}
}

func TestParseLLMProviderOutputAcceptsRawCode(t *testing.T) {
	code, err := parseLLMProviderOutput([]byte("package demo\n\nfunc TestRaw(t *testing.T) {}\n"))
	if err != nil {
		t.Fatalf("parseLLMProviderOutput() error = %v", err)
	}
	if !strings.Contains(code, "TestRaw") {
		t.Fatalf("unexpected raw code output: %q", code)
	}
}

func writeProviderSource(t *testing.T) string {
	t.Helper()

	src := `def add(a, b):
    return a + b
`
	path := filepath.Join(t.TempDir(), "calc.py")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFakeProviderCommand(t *testing.T, script string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "provider")
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}
