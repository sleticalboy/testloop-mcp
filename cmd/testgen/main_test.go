package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestRunTestgenWritesDefaultOutput(t *testing.T) {
	dir := t.TempDir()
	source := writeSource(t, dir)
	var stdout, stderr bytes.Buffer

	code := runTestgen([]string{source}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	output := filepath.Join(dir, "calc_test.go")
	if !strings.Contains(stdout.String(), "Generated: "+output+" (provider=static)") {
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
