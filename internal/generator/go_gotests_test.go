package generator

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGenerateGoTestsPreferredUsesGotestsWhenAvailable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gotests is unix-only")
	}

	srcPath := writeTempGoSource(t)
	fakeBin := writeFakeGotests(t, `#!/bin/sh
if [ "$1" != "-all" ]; then
  echo "missing -all" >&2
  exit 2
fi
if [ "$2" != "calc.go" ]; then
  echo "unexpected source: $2" >&2
  exit 3
fi
cat <<'EOF'
package sample

import "testing"

func TestFromGotests(t *testing.T) {}
EOF
`)

	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	code, err := GenerateGoTestsPreferred(srcPath)
	if err != nil {
		t.Fatalf("GenerateGoTestsPreferred() error = %v", err)
	}
	if !strings.Contains(code, "TestFromGotests") {
		t.Fatalf("expected gotests output, got:\n%s", code)
	}
}

func TestGenerateGoTestsPreferredFallsBackWhenGotestsFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gotests is unix-only")
	}

	srcPath := writeTempGoSource(t)
	fakeBin := writeFakeGotests(t, `#!/bin/sh
echo "boom" >&2
exit 42
`)

	t.Setenv("PATH", fakeBin)

	code, err := GenerateGoTestsPreferred(srcPath)
	if err != nil {
		t.Fatalf("GenerateGoTestsPreferred() fallback error = %v", err)
	}
	if strings.Contains(code, "TestFromGotests") {
		t.Fatalf("expected fallback output, got gotests output:\n%s", code)
	}
	if !strings.Contains(code, "func TestAdd") || !strings.Contains(code, "skip: true") {
		t.Fatalf("expected built-in fallback output, got:\n%s", code)
	}
}

func writeTempGoSource(t *testing.T) string {
	t.Helper()

	src := `package sample

func Add(a, b int) int {
	return a + b
}
`
	path := filepath.Join(t.TempDir(), "calc.go")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFakeGotests(t *testing.T, script string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "gotests")
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}
