package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ---- 类型定义 ----

type jsFuncInfo struct {
	Name       string
	Params     []jsParamInfo
	IsAsync    bool
	IsExported bool
	IsArrow    bool
	IsMethod   bool
	ClassName  string
	Body       string         // 函数体源码
	Analysis   jsFuncAnalysis // 函数体分析结果
}

type jsParamInfo struct {
	Name       string
	HasDefault bool
	IsRest     bool // ...args
}

// jsFuncAnalysis 函数体分析结果
type jsFuncAnalysis struct {
	ReturnType string       // number/string/array/object/boolean/null/undefined/unknown
	Returns    []string     // return expressions found in the function body
	Throws     bool         // 函数体包含 throw
	Boundaries []jsBoundary // 边界条件检测
	HasReturn  bool         // 是否有 return 语句（非 void）
	IsGetter   bool         // 是否是简单的 getter（return expression 只有一个变量/字面量）
}

// jsBoundary 边界条件
type jsBoundary struct {
	Param string // 参数名
	Value string // 边界值（原始字面量）
	Type  string // 值类型：number/string/null/undefined/boolean
}

// jsClassInfo 类信息
type jsClassInfo struct {
	Name    string
	Methods []jsFuncInfo
}

// returnTypeForAssert 返回 JS 类型字符串用于 typeof 断言
func (a jsFuncAnalysis) returnTypeForAssert() string {
	switch a.ReturnType {
	case "number":
		return "number"
	case "string":
		return "string"
	case "boolean":
		return "boolean"
	case "array":
		return "object" // JS 中数组 typeof 是 object
	case "object":
		return "object"
	case "null":
		return "object" // null 的 typeof 是 object
	default:
		return ""
	}
}

// ---- 核心函数 ----

// GenerateJestTests 读取 JS/TS 源文件，用 tree-sitter 解析后生成 Jest 测试代码
func GenerateJestTests(srcPath string) (string, error) {
	source, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("读取源文件失败: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(srcPath))
	funcs, classes, isESModule := parseJSWithTreeSitter(source, ext)

	if len(funcs) == 0 && len(classes) == 0 {
		return "// 未发现需要生成测试的函数或类", nil
	}

	moduleName := stripExt(baseName(srcPath))

	var buf strings.Builder

	if isESModule {
		buf.WriteString(fmt.Sprintf("import { %s } from './%s';\n\n", joinExportNames(funcs, classes), moduleName))
	} else {
		buf.WriteString(fmt.Sprintf("const { %s } = require('./%s');\n\n", joinExportNames(funcs, classes), moduleName))
	}

	for _, fn := range funcs {
		buf.WriteString(genJestFuncTest(fn))
	}

	for _, cls := range classes {
		buf.WriteString(genJestClassTest(cls, isESModule, moduleName))
	}

	return buf.String(), nil
}

// ---- 函数体分析（基于 body 文本字符串，不依赖解析方式） ----

var (
	jsReturnRe = regexp.MustCompile(`\breturn\s+(.+?)(?:;|\n|$)`)
	jsThrowRe  = regexp.MustCompile(`\bthrow\b`)
	jsIfEqRe   = regexp.MustCompile(`if\s*\(\s*(\w+)\s*(?:===?|!==?)\s*([^)]+?)\s*\)`)
	jsIfNullRe = regexp.MustCompile(`if\s*\(\s*(\w+)\s*(?:===?|!==?)\s*(null|undefined)\s*\)`)
)

// analyzeJSBody 分析 JS 函数体，推断返回类型、检测 throw 和边界条件
func analyzeJSBody(body string) jsFuncAnalysis {
	a := jsFuncAnalysis{}

	if body == "" {
		return a
	}

	a.Throws = jsThrowRe.MatchString(body)

	returnMatches := jsReturnRe.FindAllStringSubmatch(body, -1)
	a.HasReturn = len(returnMatches) > 0

	if a.HasReturn {
		a.Returns = extractJSReturnExpressions(returnMatches)
		a.ReturnType = inferJSReturnType(returnMatches)
	}

	a.Boundaries = extractJSBoundaries(body)

	return a
}

func extractJSReturnExpressions(matches [][]string) []string {
	seen := make(map[string]bool)
	returns := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		expr := strings.TrimSpace(m[1])
		if expr == "" || seen[expr] {
			continue
		}
		seen[expr] = true
		returns = append(returns, expr)
	}
	return returns
}

func inferJSReturnType(matches [][]string) string {
	for _, m := range matches {
		expr := strings.TrimSpace(m[1])

		if expr == "null" {
			return "null"
		}
		if expr == "undefined" {
			return "undefined"
		}
		if expr == "true" || expr == "false" {
			return "boolean"
		}
		if (strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"")) ||
			(strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'")) {
			return "string"
		}
		if strings.HasPrefix(expr, "`") {
			return "string"
		}
		if isNumericLiteral(expr) {
			return "number"
		}
		if strings.HasPrefix(expr, "[") {
			return "array"
		}
		if strings.HasPrefix(expr, "{") {
			return "object"
		}
		if strings.Contains(expr, "JSON.parse") {
			return "object"
		}
		if strings.Contains(expr, ".json()") {
			return "object"
		}
		if isArithmeticExpr(expr) {
			return "number"
		}
		if isLogicalExpr(expr) {
			return "boolean"
		}
		if strings.Contains(expr, " + ") && hasStringLiteral(expr) {
			return "string"
		}
	}

	return "unknown"
}

func isNumericLiteral(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	dotSeen := false
	for i, ch := range s {
		if ch == '.' {
			if dotSeen || i == 0 || i == len(s)-1 {
				return false
			}
			dotSeen = true
			continue
		}
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func isArithmeticExpr(s string) bool {
	for _, op := range []string{" + ", " - ", " * ", " / ", " % "} {
		if strings.Contains(s, op) {
			if op == " + " && hasStringLiteral(s) {
				return false
			}
			return true
		}
	}
	return false
}

func isLogicalExpr(s string) bool {
	for _, op := range []string{" && ", " || ", "!"} {
		if strings.Contains(s, op) {
			return true
		}
	}
	return false
}

func hasStringLiteral(s string) bool {
	return strings.Contains(s, "\"") || strings.Contains(s, "'") || strings.Contains(s, "`")
}

func extractJSBoundaries(body string) []jsBoundary {
	var boundaries []jsBoundary
	seen := make(map[string]bool)

	nullMatches := jsIfNullRe.FindAllStringSubmatch(body, -1)
	for _, m := range nullMatches {
		param := m[1]
		val := m[2]
		key := param + ":" + val
		if !seen[key] {
			seen[key] = true
			boundaries = append(boundaries, jsBoundary{Param: param, Value: val, Type: val})
		}
	}

	ifMatches := jsIfEqRe.FindAllStringSubmatch(body, -1)
	for _, m := range ifMatches {
		param := m[1]
		val := strings.TrimSpace(m[2])

		if val == "null" || val == "undefined" {
			continue
		}

		key := param + ":" + val
		if seen[key] {
			continue
		}
		seen[key] = true

		bType := "unknown"
		if isNumericLiteral(val) {
			bType = "number"
		} else if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"")) ||
			(strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) {
			bType = "string"
		} else if val == "true" || val == "false" {
			bType = "boolean"
		}

		boundaries = append(boundaries, jsBoundary{Param: param, Value: val, Type: bType})
	}

	return boundaries
}

// ---- 测试生成 ----

func genJestFuncTest(fn jsFuncInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", fn.Name))

	sb.WriteString(fmt.Sprintf("  it('should return expected result for normal input', %s => {\n", jsAsyncArrow(fn.IsAsync)))
	if fn.IsAsync {
		sb.WriteString(fmt.Sprintf("    const result = await %s(%s);\n", fn.Name, jsArgList(fn.Params)))
	} else {
		sb.WriteString(fmt.Sprintf("    const result = %s(%s);\n", fn.Name, jsArgList(fn.Params)))
	}
	sb.WriteString(genJSResultAssertion(fn.Analysis, "    "))
	sb.WriteString("  });\n\n")

	for _, b := range fn.Analysis.Boundaries {
		if !jsParamExists(fn.Params, b.Param) {
			continue
		}
		sb.WriteString(fmt.Sprintf("  it('should handle %s = %s', %s => {\n",
			b.Param, b.Value, jsAsyncArrow(fn.IsAsync)))
		args := jsArgListWithBoundary(fn.Params, b)
		if fn.IsAsync {
			sb.WriteString(fmt.Sprintf("    const result = await %s(%s);\n", fn.Name, args))
		} else {
			sb.WriteString(fmt.Sprintf("    const result = %s(%s);\n", fn.Name, args))
		}

		if fn.Analysis.Throws {
			sb.WriteString("    expect(result).toBeDefined();\n")
		} else {
			sb.WriteString(genJSResultAssertion(fn.Analysis, "    "))
		}
		sb.WriteString("  });\n\n")
	}

	if fn.Analysis.Throws {
		sb.WriteString(fmt.Sprintf("  it('should throw on invalid input', %s => {\n", jsAsyncArrow(fn.IsAsync)))
		if fn.IsAsync {
			sb.WriteString(fmt.Sprintf("    await expect(%s(%s)).rejects.toThrow();\n", fn.Name, jsArgList(fn.Params)))
		} else {
			sb.WriteString(fmt.Sprintf("    expect(() => %s(%s)).toThrow();\n", fn.Name, jsArgList(fn.Params)))
		}
		sb.WriteString("  });\n\n")
	}

	if len(fn.Params) == 0 && fn.Analysis.HasReturn {
		sb.WriteString(fmt.Sprintf("  it('should work with no arguments', %s => {\n", jsAsyncArrow(fn.IsAsync)))
		if fn.IsAsync {
			sb.WriteString(fmt.Sprintf("    const result = await %s();\n", fn.Name))
		} else {
			sb.WriteString(fmt.Sprintf("    const result = %s();\n", fn.Name))
		}
		sb.WriteString(genJSResultAssertion(fn.Analysis, "    "))
		sb.WriteString("  });\n\n")
	}

	sb.WriteString("});\n\n")

	return sb.String()
}

func genJestClassTest(cls jsClassInfo, isESModule bool, moduleName string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", cls.Name))

	sb.WriteString(fmt.Sprintf("  describe('constructor', () => {\n"))
	sb.WriteString(fmt.Sprintf("    it('should create an instance', () => {\n"))
	sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
	sb.WriteString(fmt.Sprintf("      expect(instance).toBeInstanceOf(%s);\n", cls.Name))
	sb.WriteString("    });\n")
	sb.WriteString("  });\n\n")

	for _, method := range cls.Methods {
		sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))

		sb.WriteString(fmt.Sprintf("    it('should return expected result', %s => {\n", jsAsyncArrow(method.IsAsync)))
		sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
		if method.IsAsync {
			sb.WriteString(fmt.Sprintf("      const result = await instance.%s(%s);\n", method.Name, jsArgList(method.Params)))
		} else {
			sb.WriteString(fmt.Sprintf("      const result = instance.%s(%s);\n", method.Name, jsArgList(method.Params)))
		}
		sb.WriteString(genJSResultAssertion(method.Analysis, "      "))
		sb.WriteString("    });\n\n")

		if method.Analysis.Throws {
			sb.WriteString(fmt.Sprintf("    it('should throw on invalid input', %s => {\n", jsAsyncArrow(method.IsAsync)))
			sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
			if method.IsAsync {
				sb.WriteString(fmt.Sprintf("      await expect(instance.%s(%s)).rejects.toThrow();\n", method.Name, jsArgList(method.Params)))
			} else {
				sb.WriteString(fmt.Sprintf("      expect(() => instance.%s(%s)).toThrow();\n", method.Name, jsArgList(method.Params)))
			}
			sb.WriteString("    });\n\n")
		}

		for _, b := range method.Analysis.Boundaries {
			if !jsParamExists(method.Params, b.Param) {
				continue
			}
			sb.WriteString(fmt.Sprintf("    it('should handle %s = %s', %s => {\n",
				b.Param, b.Value, jsAsyncArrow(method.IsAsync)))
			sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
			args := jsArgListWithBoundary(method.Params, b)
			if method.IsAsync {
				sb.WriteString(fmt.Sprintf("      const result = await instance.%s(%s);\n", method.Name, args))
			} else {
				sb.WriteString(fmt.Sprintf("      const result = instance.%s(%s);\n", method.Name, args))
			}
			sb.WriteString(genJSResultAssertion(method.Analysis, "      "))
			sb.WriteString("    });\n\n")
		}

		sb.WriteString("  });\n\n")
	}

	sb.WriteString("});\n\n")

	return sb.String()
}

func genJSResultAssertion(a jsFuncAnalysis, indent string) string {
	var sb strings.Builder

	if !a.HasReturn {
		sb.WriteString(indent + "// void function, verify no exception\n")
		return sb.String()
	}

	switch a.ReturnType {
	case "number":
		sb.WriteString(indent + "expect(typeof result).toBe('number');\n")
		sb.WriteString(indent + "expect(result).not.toBeNaN();\n")
	case "string":
		sb.WriteString(indent + "expect(typeof result).toBe('string');\n")
		sb.WriteString(indent + "expect(result.length).toBeGreaterThanOrEqual(0);\n")
	case "boolean":
		sb.WriteString(indent + "expect(typeof result).toBe('boolean');\n")
	case "array":
		sb.WriteString(indent + "expect(Array.isArray(result)).toBe(true);\n")
	case "object":
		sb.WriteString(indent + "expect(typeof result).toBe('object');\n")
		sb.WriteString(indent + "expect(result).not.toBeNull();\n")
	case "null":
		sb.WriteString(indent + "expect(result).toBeNull();\n")
	case "undefined":
		sb.WriteString(indent + "expect(result).toBeUndefined();\n")
	default:
		sb.WriteString(indent + "expect(result).toBeDefined();\n")
	}

	return sb.String()
}

// ---- 辅助函数 ----

func jsAsyncArrow(isAsync bool) string {
	if isAsync {
		return "async ()"
	}
	return "()"
}

func jsParamExists(params []jsParamInfo, name string) bool {
	for _, p := range params {
		if p.Name == name {
			return true
		}
	}
	return false
}

func jsArgListWithBoundary(params []jsParamInfo, b jsBoundary) string {
	if len(params) == 0 {
		return ""
	}
	args := make([]string, len(params))
	for i, p := range params {
		if p.Name == b.Param {
			args[i] = b.Value
		} else if p.IsRest {
			args[i] = "[]"
		} else {
			args[i] = "undefined"
		}
	}
	return strings.Join(args, ", ")
}

func jsArgList(params []jsParamInfo) string {
	if len(params) == 0 {
		return ""
	}
	args := make([]string, len(params))
	for i, p := range params {
		if p.IsRest {
			args[i] = "[]"
		} else {
			args[i] = "undefined"
		}
	}
	return strings.Join(args, ", ")
}

func isTestHelper(name string) bool {
	switch name {
	case "test", "it", "describe", "beforeEach", "beforeAll", "afterEach", "afterAll", "expect", "jest", "before", "after":
		return true
	}
	return false
}

func isJSKeyword(name string) bool {
	switch name {
	case "if", "for", "while", "switch", "catch", "return", "else", "do", "try",
		"finally", "throw", "break", "continue", "new", "delete", "typeof",
		"instanceof", "void", "in", "of", "let", "const", "var", "function",
		"class", "extends", "super", "import", "export", "default", "from",
		"async", "await", "yield", "static", "get", "set", "this":
		return true
	}
	return false
}

func dedupJSFuncs(funcs []jsFuncInfo) []jsFuncInfo {
	seen := make(map[string]bool)
	var result []jsFuncInfo
	for _, fn := range funcs {
		key := fn.Name
		if fn.IsMethod {
			key = fn.ClassName + "." + fn.Name
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, fn)
		}
	}
	return result
}

func joinExportNames(funcs []jsFuncInfo, classes []jsClassInfo) string {
	var names []string
	seen := make(map[string]bool)
	for _, fn := range funcs {
		if !fn.IsMethod && !seen[fn.Name] {
			names = append(names, fn.Name)
			seen[fn.Name] = true
		}
	}
	for _, cls := range classes {
		if !seen[cls.Name] {
			names = append(names, cls.Name)
			seen[cls.Name] = true
		}
	}
	return strings.Join(names, ", ")
}

func baseName(path string) string {
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		path = path[idx+1:]
	}
	if idx := strings.LastIndex(path, "\\"); idx >= 0 {
		path = path[idx+1:]
	}
	return path
}

func stripExt(filename string) string {
	if idx := strings.LastIndex(filename, "."); idx > 0 {
		return filename[:idx]
	}
	return filename
}
