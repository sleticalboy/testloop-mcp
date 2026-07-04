package generator

import (
	"path/filepath"
	"strings"
)

// TestFileName returns the conventional generated test file path for a source file.
func TestFileName(srcPath string) string {
	ext := strings.ToLower(filepath.Ext(srcPath))
	dir := filepath.Dir(srcPath)
	name := strings.TrimSuffix(filepath.Base(srcPath), ext)

	switch ext {
	case ".go":
		return filepath.Join(dir, name+"_test.go")
	case ".js", ".mjs", ".cjs":
		return filepath.Join(dir, name+".test.js")
	case ".jsx":
		return filepath.Join(dir, name+".test.jsx")
	case ".ts":
		return filepath.Join(dir, name+".test.ts")
	case ".tsx":
		return filepath.Join(dir, name+".test.tsx")
	case ".py":
		return filepath.Join(dir, "test_"+name+".py")
	default:
		return filepath.Join(dir, name+".test"+ext)
	}
}
