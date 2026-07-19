package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sleticalboy/testloop-mcp/internal/generator"
)

func TestRunTestgenRequiresSourceFile(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runTestgen(nil, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: testgen") {
		t.Fatalf("stderr missing usage: %q", stderr.String())
	}
}

func TestRunTestgenInvalidFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runTestgen([]string{"-bad"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("stderr missing flag error: %q", stderr.String())
	}
}

func TestRunTestgenRejectsUnknownProvider(t *testing.T) {
	dir := t.TempDir()
	source := writeSource(t, dir)
	var stdout, stderr bytes.Buffer

	code := runTestgen([]string{"-provider", "unknown", source}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "Provider error") {
		t.Fatalf("stderr missing provider error: %q", stderr.String())
	}
}

func TestRunTestgenProviderCheckStatic(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runTestgen([]string{"-provider-check"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "provider=static") || !strings.Contains(stdout.String(), "status=ok") {
		t.Fatalf("stdout missing static diagnostics: %q", stdout.String())
	}
}

func TestRunTestgenProviderCheckAutoFallsBackToStatic(t *testing.T) {
	t.Setenv(generator.EnvLLMProviderCommand, "")
	var stdout, stderr bytes.Buffer

	code := runTestgen([]string{"-provider", "auto", "-provider-check"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "will fall back to static") {
		t.Fatalf("stdout missing auto fallback detail: %q", stdout.String())
	}
}

func TestRunTestgenProviderCheckLLMRequiresCommand(t *testing.T) {
	t.Setenv(generator.EnvLLMProviderCommand, "")
	var stdout, stderr bytes.Buffer

	code := runTestgen([]string{"-provider", "llm", "-provider-check"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), generator.EnvLLMProviderCommand+"=<unset>") {
		t.Fatalf("stdout missing env diagnostic: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "provider llm requires "+generator.EnvLLMProviderCommand) {
		t.Fatalf("stderr missing missing-command diagnostic: %q", stderr.String())
	}
}

func TestRunTestgenProviderCheckReportsMissingExecutable(t *testing.T) {
	t.Setenv(generator.EnvLLMProviderCommand, filepath.Join(t.TempDir(), "missing-provider"))
	var stdout, stderr bytes.Buffer

	code := runTestgen([]string{"-provider", "llm", "-provider-check"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "provider command executable not found") {
		t.Fatalf("stderr missing executable diagnostic: %q", stderr.String())
	}
}

func TestRunTestgenProviderCheckReportsConfiguredCommand(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	t.Setenv(generator.EnvLLMProviderCommand, exe+" --fake-provider")
	var stdout, stderr bytes.Buffer

	code := runTestgen([]string{"-provider", "llm", "-provider-check"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "status=ok") || !strings.Contains(stdout.String(), "command="+exe+" --fake-provider") {
		t.Fatalf("stdout missing configured-command diagnostic: %q", stdout.String())
	}
}

func TestRunTestgenLLMFailureSuggestsProviderCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake provider is unix-only")
	}
	dir := t.TempDir()
	source := writeSource(t, dir)
	provider := filepath.Join(t.TempDir(), "provider")
	if err := os.WriteFile(provider, []byte(`#!/usr/bin/env sh
cat <<'EOF'
{"code":"package calc\n\nfunc Add(a, b int) int { return a + b }\n"}
EOF
`), 0o755); err != nil {
		t.Fatalf("write provider: %v", err)
	}
	t.Setenv(generator.EnvLLMProviderCommand, provider)
	var stdout, stderr bytes.Buffer

	code := runTestgen([]string{"-provider", "llm", source}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "did not look like go test code") {
		t.Fatalf("stderr missing validation error: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "-provider llm -provider-check") {
		t.Fatalf("stderr missing provider-check hint: %q", stderr.String())
	}
}

func TestRunTestgenWritesDefaultOutput(t *testing.T) {
	dir := t.TempDir()
	source := writeSource(t, dir)
	var stdout, stderr bytes.Buffer

	code := runTestgen([]string{source}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	output := filepath.Join(dir, "calc_test.go")
	if !strings.Contains(stdout.String(), "Generated: "+output+" (provider=static action=manual_review)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read generated test: %v", err)
	}
	if !strings.Contains(string(content), "func TestAdd") {
		t.Fatalf("generated test missing TestAdd:\n%s", content)
	}
}

func TestRunTestgenReportsManualReviewActionForGoTodoSkeleton(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "alias.go")
	if err := os.WriteFile(source, []byte(`package alias

func SliceMapper[T any, U any](src []T, mapper func(T) U) []U {
	dst := make([]U, 0, len(src))
	for _, v := range src {
		dst = append(dst, mapper(v))
	}
	return dst
}
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	output := filepath.Join(dir, "alias_test.go")
	var stdout, stderr bytes.Buffer

	code := runTestgen([]string{source, output}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Generated: "+output+" (provider=static action=manual_review)") {
		t.Fatalf("stdout missing manual_review action: %q", stdout.String())
	}
}

func TestRunTestgenWritesExplicitOutput(t *testing.T) {
	dir := t.TempDir()
	source := writeSource(t, dir)
	output := filepath.Join(dir, "custom_test.go")
	var stdout, stderr bytes.Buffer

	code := runTestgen([]string{source, output}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("expected explicit output file: %v", err)
	}
}

func TestRunTestgenJavaScriptExplicitOutputUsesRelativeSourceImport(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	testDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("mkdir tests: %v", err)
	}
	source := filepath.Join(srcDir, "user.js")
	if err := os.WriteFile(source, []byte(`export function listUsers(params) {
  return request({ url: '/users', method: 'get', params });
}
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	output := filepath.Join(testDir, "user.test.js")
	var stdout, stderr bytes.Buffer

	code := runTestgen([]string{source, output}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(content), "import { listUsers } from '../src/user';") {
		t.Fatalf("generated JS test import should be relative to explicit output path:\n%s", content)
	}
}

func TestRunTestgenGoExplicitOutputAvoidsDuplicatePackageTestNames(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(source, []byte(`package calc

func Add(a, b int) int { return a + b }
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "calc_test.go"), []byte(`package calc

import "testing"

func TestAdd(t *testing.T) {}
`), 0o644); err != nil {
		t.Fatalf("write existing test: %v", err)
	}
	output := filepath.Join(dir, "calc_testloop_test.go")
	var stdout, stderr bytes.Buffer

	code := runTestgen([]string{source, output}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "func TestAddTestLoop(t *testing.T)") {
		t.Fatalf("generated Go test should avoid existing TestAdd:\n%s", text)
	}
	if strings.Contains(text, "func TestAdd(t *testing.T)") {
		t.Fatalf("generated Go test still contains duplicate TestAdd:\n%s", text)
	}
}

func TestRunTestgenReportsWriteError(t *testing.T) {
	dir := t.TempDir()
	source := writeSource(t, dir)
	output := filepath.Join(dir, "missing", "calc_test.go")
	var stdout, stderr bytes.Buffer

	code := runTestgen([]string{source, output}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "Write error") {
		t.Fatalf("stderr missing write error: %q", stderr.String())
	}
}

func writeSource(t *testing.T, dir string) string {
	t.Helper()
	source := filepath.Join(dir, "calc.go")
	code := `package calc

func Add(a, b int) int { return a + b }
`
	if err := os.WriteFile(source, []byte(code), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	return source
}
