package detector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DetectFramework 根据路径自动检测测试框架。
// 检测优先级：
//  1. 文件扩展名（.go → go-test, .py → pytest, .js/.ts/.jsx/.tsx → 需进一步看 package.json）
//  2. 项目配置文件（package.json scripts.test + dependencies / pyproject.toml / go.mod）
//  3. 向上递归查找配置文件（最多 5 层）
//  4. 默认 go-test
func DetectFramework(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "go-test"
	}

	// 文件 → 先看扩展名
	if !info.IsDir() {
		switch filepath.Ext(path) {
		case ".go":
			return "go-test"
		case ".py":
			return "pytest"
		case ".js", ".ts", ".jsx", ".tsx":
			// JS/TS 文件需要看 package.json 决定 jest/vitest/mocha
			if fw := detectJSFramework(path); fw != "" {
				return fw
			}
		}
	}

	// 目录或文件 → 向上查找配置文件
	dir := path
	if !info.IsDir() {
		dir = filepath.Dir(path)
	}
	return detectFromDir(dir)
}

// detectJSFramework 从 JS/TS 文件出发向上查找 package.json，返回 jest/vitest/mocha，空串表示未确定
func detectJSFramework(filePath string) string {
	pkg, dir := findPackageJSON(filepath.Dir(filePath))
	if pkg == nil {
		return ""
	}
	return resolveJSFramework(pkg, dir)
}

// detectFromDir 从目录出发检测框架
func detectFromDir(dir string) string {
	// package.json
	pkg, pkgDir := findPackageJSON(dir)
	if pkg != nil {
		return resolveJSFramework(pkg, pkgDir) // 已有兜底 "jest"
	}

	// go.mod
	if _, d := findFile(dir, "go.mod"); d != "" {
		return "go-test"
	}

	// pyproject.toml
	if fw, _ := detectPytest(dir); fw != "" {
		return fw
	}

	// setup.py
	if _, d := findFile(dir, "setup.py"); d != "" {
		return "pytest"
	}

	return "go-test"
}

// ---- package.json 解析 ----

type packageJSON struct {
	Name            string            `json:"name"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// findPackageJSON 从 dir 开始向上查找 package.json（最多 5 层），返回解析结果和所在目录
func findPackageJSON(dir string) (*packageJSON, string) {
	found, foundDir := findFile(dir, "package.json")
	if found == "" {
		return nil, ""
	}
	data, err := os.ReadFile(found)
	if err != nil {
		return nil, ""
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, ""
	}
	return &pkg, foundDir
}

// resolveJSFramework 根据 package.json 判断 JS 测试框架
func resolveJSFramework(pkg *packageJSON, _ string) string {
	// 优先看 scripts.test — 最可靠
	if testScript, ok := pkg.Scripts["test"]; ok {
		lower := strings.ToLower(testScript)
		switch {
		case strings.Contains(lower, "vitest"):
			return "vitest"
		case strings.Contains(lower, "mocha"):
			return "mocha"
		case strings.Contains(lower, "jest"):
			return "jest"
		}
	}

	// 其次看 devDependencies
	if hasDep(pkg.DevDependencies, "vitest") {
		return "vitest"
	}
	if hasDep(pkg.DevDependencies, "mocha") {
		return "mocha"
	}
	if hasDep(pkg.DevDependencies, "jest") {
		return "jest"
	}

	// 最后看 dependencies
	if hasDep(pkg.Dependencies, "vitest") {
		return "vitest"
	}
	if hasDep(pkg.Dependencies, "mocha") {
		return "mocha"
	}
	if hasDep(pkg.Dependencies, "jest") {
		return "jest"
	}

	// 有 package.json 但没测试框架 → 默认 jest
	return "jest"
}

// hasDep 检查 dependencies map 中是否有指定包（精确匹配 key）
func hasDep(deps map[string]string, name string) bool {
	if deps == nil {
		return false
	}
	_, ok := deps[name]
	return ok
}

// ---- pyproject.toml 解析 ----

// detectPytest 从 dir 开始向上查找 pyproject.toml，检测是否使用 pytest
func detectPytest(dir string) (string, string) {
	found, foundDir := findFile(dir, "pyproject.toml")
	if found == "" {
		return "", ""
	}
	data, err := os.ReadFile(found)
	if err != nil {
		return "", ""
	}
	if hasPytestInTOML(string(data)) {
		return "pytest", foundDir
	}
	return "", ""
}

// hasPytestInTOML 简单文本扫描：检查 pyproject.toml 是否包含 pytest 相关配置
func hasPytestInTOML(content string) bool {
	// [tool.pytest.ini_options] 或 pytest 出现在 dependencies 行
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// [tool.pytest...] section
		if strings.HasPrefix(trimmed, "[tool.pytest") {
			return true
		}
		// dependencies = ["pytest", ...] 或 "pytest" in a dep list
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "pytest") {
			// 排除注释
			if !strings.HasPrefix(trimmed, "#") {
				return true
			}
		}
	}
	return false
}

// ---- 通用文件查找 ----

// findFile 从 dir 开始向上查找名为 filename 的文件（最多 5 层），返回文件完整路径和所在目录
func findFile(dir, filename string) (string, string) {
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, filename)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", ""
}
