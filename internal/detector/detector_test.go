package detector

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile 在 dir 下创建文件
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectFramework_GoFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main")
	path := filepath.Join(dir, "main.go")

	if fw := DetectFramework(path); fw != "go-test" {
		t.Errorf("got %s, want go-test", fw)
	}
}

func TestDetectFramework_PyFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.py", "print('hi')")
	path := filepath.Join(dir, "app.py")

	if fw := DetectFramework(path); fw != "pytest" {
		t.Errorf("got %s, want pytest", fw)
	}
}

func TestDetectFramework_RustFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "lib.rs", "pub fn add() {}")
	path := filepath.Join(dir, "lib.rs")

	if fw := DetectFramework(path); fw != "cargo-test" {
		t.Errorf("got %s, want cargo-test", fw)
	}
}

func TestDetectFramework_JavaFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Calculator.java", "class Calculator {}")
	path := filepath.Join(dir, "Calculator.java")

	if fw := DetectFramework(path); fw != "junit" {
		t.Errorf("got %s, want junit", fw)
	}
}

func TestDetectFramework_JSFile_VitestViaScript(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"scripts": { "test": "vitest run" },
		"devDependencies": { "jest": "^29.0.0" }
	}`)
	writeFile(t, dir, "app.test.ts", "")
	path := filepath.Join(dir, "app.test.ts")

	if fw := DetectFramework(path); fw != "vitest" {
		t.Errorf("got %s, want vitest (scripts.test 优先级最高)", fw)
	}
}

func TestDetectFramework_JSFile_MochaViaScript(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"scripts": { "test": "mocha --reporter spec" },
		"devDependencies": { "jest": "^29.0.0", "mocha": "^10.0.0" }
	}`)
	writeFile(t, dir, "app.js", "")
	path := filepath.Join(dir, "app.js")

	if fw := DetectFramework(path); fw != "mocha" {
		t.Errorf("got %s, want mocha (scripts.test 含 mocha)", fw)
	}
}

func TestDetectFramework_JSFile_JestViaDevDeps(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"scripts": {},
		"devDependencies": { "jest": "^29.0.0" }
	}`)
	writeFile(t, dir, "app.js", "")
	path := filepath.Join(dir, "app.js")

	if fw := DetectFramework(path); fw != "jest" {
		t.Errorf("got %s, want jest", fw)
	}
}

func TestDetectFramework_JSFile_VitestViaDevDeps(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"scripts": {},
		"devDependencies": { "vitest": "^1.0.0" }
	}`)
	writeFile(t, dir, "app.ts", "")
	path := filepath.Join(dir, "app.ts")

	if fw := DetectFramework(path); fw != "vitest" {
		t.Errorf("got %s, want vitest", fw)
	}
}

func TestDetectFramework_JSFile_MochaViaDependencies(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"scripts": {},
		"dependencies": { "mocha": "^10.0.0" }
	}`)
	writeFile(t, dir, "app.js", "")
	path := filepath.Join(dir, "app.js")

	if fw := DetectFramework(path); fw != "mocha" {
		t.Errorf("got %s, want mocha", fw)
	}
}

func TestDetectFramework_JSFile_JestViaDependencies(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"scripts": {},
		"dependencies": { "jest": "^29.0.0" }
	}`)
	writeFile(t, dir, "app.js", "")
	path := filepath.Join(dir, "app.js")

	if fw := DetectFramework(path); fw != "jest" {
		t.Errorf("got %s, want jest", fw)
	}
}

func TestDetectFramework_JSFile_NoTestDep_DefaultsJest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"name": "myapp",
		"dependencies": { "express": "^4.0.0" }
	}`)
	writeFile(t, dir, "app.js", "")
	path := filepath.Join(dir, "app.js")

	if fw := DetectFramework(path); fw != "jest" {
		t.Errorf("got %s, want jest (有 package.json 无测试框架时默认 jest)", fw)
	}
}

func TestDetectFramework_Dir_GoMod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app\ngo 1.25\n")

	if fw := DetectFramework(dir); fw != "go-test" {
		t.Errorf("got %s, want go-test", fw)
	}
}

func TestDetectFramework_Dir_CargoToml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"app\"\n")

	if fw := DetectFramework(dir); fw != "cargo-test" {
		t.Errorf("got %s, want cargo-test", fw)
	}
}

func TestDetectFramework_Dir_SetupPy(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "setup.py", "from setuptools import setup\n")

	if fw := DetectFramework(dir); fw != "pytest" {
		t.Errorf("got %s, want pytest", fw)
	}
}

func TestDetectFramework_Dir_PomXML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", "<project/>")

	if fw := DetectFramework(dir); fw != "junit" {
		t.Errorf("got %s, want junit", fw)
	}
}

func TestDetectFramework_Dir_GradleKts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle.kts", "plugins {}\n")

	if fw := DetectFramework(dir); fw != "junit" {
		t.Errorf("got %s, want junit", fw)
	}
}

func TestDetectFramework_Dir_PyprojectToml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", `[tool.pytest.ini_options]
testpaths = ["tests"]
`)

	if fw := DetectFramework(dir); fw != "pytest" {
		t.Errorf("got %s, want pytest", fw)
	}
}

func TestDetectFramework_Dir_PyprojectToml_PoetryDeps(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", `[tool.poetry.dependencies]
python = "^3.11"
pytest = "^7.0"
`)

	if fw := DetectFramework(dir); fw != "pytest" {
		t.Errorf("got %s, want pytest", fw)
	}
}

func TestDetectFramework_Dir_PyprojectToml_NoPytest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", `[project]
name = "myapp"
dependencies = ["flask"]
`)

	// 没有 pytest 也不该匹配
	if fw := DetectFramework(dir); fw == "pytest" {
		t.Errorf("不应检测到 pytest，got %s", fw)
	}
}

func TestDetectFramework_WalkUpParentDirs(t *testing.T) {
	root := t.TempDir()
	// package.json 在 root，测试文件在 root/src/utils/
	writeFile(t, root, "package.json", `{
		"scripts": { "test": "vitest" }
	}`)
	subDir := filepath.Join(root, "src", "utils")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, subDir, "util.test.ts", "")

	path := filepath.Join(subDir, "util.test.ts")
	if fw := DetectFramework(path); fw != "vitest" {
		t.Errorf("got %s, want vitest (应向上查找 package.json)", fw)
	}
}

func TestDetectFramework_WalkUpGoMod(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\ngo 1.25\n")
	subDir := filepath.Join(root, "pkg", "service")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	if fw := DetectFramework(subDir); fw != "go-test" {
		t.Errorf("got %s, want go-test (应向上查找 go.mod)", fw)
	}
}

func TestHasDep(t *testing.T) {
	deps := map[string]string{"jest": "^29.0.0", "vitest": "^1.0.0"}
	if !hasDep(deps, "jest") {
		t.Error("应找到 jest")
	}
	if hasDep(deps, "mocha") {
		t.Error("不应找到 mocha")
	}
	if hasDep(nil, "jest") {
		t.Error("nil map 应返回 false")
	}
}

func TestHasPytestInTOML(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			"tool.pytest section",
			`[tool.pytest.ini_options]
testpaths = ["tests"]`,
			true,
		},
		{
			"poetry deps with pytest",
			`[tool.poetry.dependencies]
python = "^3.11"
pytest = "^7.0"`,
			true,
		},
		{
			"project deps with pytest",
			`[project]
dependencies = ["pytest>=7.0", "flask"]`,
			true,
		},
		{
			"no pytest",
			`[project]
name = "app"
dependencies = ["flask"]`,
			false,
		},
		{
			"commented pytest",
			`# pytest is not used
[project]
dependencies = ["flask"]`,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasPytestInTOML(tt.content); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "")
	subDir := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 从子目录向上找
	path, foundDir := findFile(subDir, "go.mod")
	if path == "" {
		t.Fatal("应找到 go.mod")
	}
	if filepath.Dir(path) != root {
		t.Errorf("找到的目录 = %s, want %s", filepath.Dir(path), root)
	}
	_ = foundDir
}

func TestFindFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	path, _ := findFile(dir, "nonexistent.file")
	if path != "" {
		t.Errorf("不应找到文件，got %s", path)
	}
}
