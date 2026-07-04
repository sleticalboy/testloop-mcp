package generator

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// GenerateGoTestsPreferred uses gotests when it is available and falls back to
// the built-in AST generator when the external tool cannot produce output.
func GenerateGoTestsPreferred(srcPath string) (string, error) {
	code, err := generateGoTestsWithGotests(srcPath, "gotests")
	if err == nil {
		return code, nil
	}
	return GenerateGoTests(srcPath)
}

func generateGoTestsWithGotests(srcPath, binary string) (string, error) {
	if strings.TrimSpace(binary) == "" {
		return "", fmt.Errorf("gotests binary is empty")
	}

	path, err := exec.LookPath(binary)
	if err != nil {
		return "", fmt.Errorf("gotests not found: %w", err)
	}

	absPath, err := filepath.Abs(srcPath)
	if err != nil {
		return "", fmt.Errorf("resolve source path: %w", err)
	}

	var stderr bytes.Buffer
	cmd := exec.Command(path, "-all", filepath.Base(absPath))
	cmd.Dir = filepath.Dir(absPath)
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("gotests failed: %s", msg)
	}
	if strings.TrimSpace(string(out)) == "" {
		return "", fmt.Errorf("gotests returned empty output")
	}

	return string(out), nil
}
