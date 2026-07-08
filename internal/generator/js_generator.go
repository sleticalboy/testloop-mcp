package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
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
	Param      string // 参数名
	Value      string // 边界值（原始字面量）
	Type       string // 值类型：number/string/null/undefined/boolean
	ReturnExpr string // 命中边界分支时的简单 return 表达式
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
	return generateJestTests(srcPath, nil)
}

func GenerateJestTestsForCoverageTask(srcPath string, task *types.CoverageTestTask) (string, error) {
	if task == nil {
		return GenerateJestTests(srcPath)
	}
	return generateJestTests(srcPath, task)
}

func generateJestTests(srcPath string, task *types.CoverageTestTask) (string, error) {
	source, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("读取源文件失败: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(srcPath))
	funcs, classes, isESModule := parseJSWithTreeSitter(source, ext)
	if task != nil {
		funcs, classes = filterJSTargetsForCoverageTask(funcs, classes, task)
	}

	if len(funcs) == 0 && len(classes) == 0 {
		return "// 未发现需要生成测试的函数或类", nil
	}

	moduleName := stripExt(baseName(srcPath))

	var buf strings.Builder

	mochaTask := task != nil && strings.EqualFold(task.Framework, "mocha")
	if mochaTask && isESModule {
		buf.WriteString("import { expect } from 'chai';\n")
	} else if mochaTask {
		buf.WriteString("const { expect } = require('chai');\n")
	}
	if isESModule {
		buf.WriteString(fmt.Sprintf("import { %s } from './%s';\n\n", joinExportNames(funcs, classes), moduleName))
	} else {
		buf.WriteString(fmt.Sprintf("const { %s } = require('./%s');\n\n", joinExportNames(funcs, classes), moduleName))
	}

	for _, fn := range funcs {
		if task != nil {
			buf.WriteString(genJestFuncTestForCoverageTask(fn, task))
		} else {
			buf.WriteString(genJestFuncTest(fn))
		}
	}

	for _, cls := range classes {
		if task != nil {
			buf.WriteString(genJestClassTestForCoverageTask(cls, task))
		} else {
			buf.WriteString(genJestClassTest(cls, isESModule, moduleName))
		}
	}

	return buf.String(), nil
}

func filterJSTargetsForCoverageTask(funcs []jsFuncInfo, classes []jsClassInfo, task *types.CoverageTestTask) ([]jsFuncInfo, []jsClassInfo) {
	target := strings.TrimSpace(task.Target)
	if target == "" {
		return funcs, classes
	}

	filteredFuncs := make([]jsFuncInfo, 0, len(funcs))
	for _, fn := range funcs {
		if taskTargetMatches(target, "", fn.Name) {
			filteredFuncs = append(filteredFuncs, fn)
		}
	}

	filteredClasses := make([]jsClassInfo, 0, len(classes))
	for _, cls := range classes {
		if taskTargetMatches(target, cls.Name, cls.Name) {
			filteredClasses = append(filteredClasses, cls)
			continue
		}
		methods := make([]jsFuncInfo, 0, len(cls.Methods))
		for _, method := range cls.Methods {
			if taskTargetMatches(target, cls.Name, method.Name) {
				methods = append(methods, method)
			}
		}
		if len(methods) > 0 {
			filteredClasses = append(filteredClasses, jsClassInfo{Name: cls.Name, Methods: methods})
		}
	}

	if len(filteredFuncs) == 0 && len(filteredClasses) == 0 {
		return funcs, classes
	}
	return filteredFuncs, filteredClasses
}

// ---- 函数体分析（基于 body 文本字符串，不依赖解析方式） ----

var (
	jsReturnRe   = regexp.MustCompile(`\breturn\s+(.+?)(?:;|\n|$)`)
	jsThrowRe    = regexp.MustCompile(`\bthrow\b`)
	jsIfEqRe     = regexp.MustCompile(`if\s*\(\s*(\w+)\s*(?:===?|!==?)\s*([^)]+?)\s*\)`)
	jsIfNullRe   = regexp.MustCompile(`if\s*\(\s*(\w+)\s*(?:===?|!==?)\s*(null|undefined)\s*\)`)
	jsIfReturnRe = regexp.MustCompile(`(?s)if\s*\(\s*(\w+)\s*(?:===?|==)\s*([^)]+?)\s*\)\s*(?:\{\s*)?return\s+(.+?)(?:;|\n|\})`)
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
	branchReturns := extractJSBranchReturns(body)

	nullMatches := jsIfNullRe.FindAllStringSubmatch(body, -1)
	for _, m := range nullMatches {
		param := m[1]
		val := m[2]
		key := param + ":" + val
		if !seen[key] {
			seen[key] = true
			boundaries = append(boundaries, jsBoundary{Param: param, Value: val, Type: val, ReturnExpr: branchReturns[key]})
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

		boundaries = append(boundaries, jsBoundary{Param: param, Value: val, Type: bType, ReturnExpr: branchReturns[key]})
	}

	return boundaries
}

func extractJSBranchReturns(body string) map[string]string {
	results := map[string]string{}
	for _, m := range jsIfReturnRe.FindAllStringSubmatch(body, -1) {
		if len(m) != 4 {
			continue
		}
		param := strings.TrimSpace(m[1])
		value := strings.TrimSpace(m[2])
		ret := strings.TrimSpace(m[3])
		if param == "" || value == "" || ret == "" {
			continue
		}
		results[param+":"+value] = ret
	}
	return results
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
	sb.WriteString(genJSResultAssertionWithArgs(fn.Analysis, fn.Params, nil, "    "))
	sb.WriteString("  });\n\n")

	for _, b := range fn.Analysis.Boundaries {
		if !jsParamExists(fn.Params, b.Param) {
			continue
		}
		sb.WriteString(fmt.Sprintf("  it('should handle %s = %s', %s => {\n",
			b.Param, jsEscapeTestNameValue(b.Value), jsAsyncArrow(fn.IsAsync)))
		args := jsArgListWithBoundary(fn.Params, b)
		if fn.Analysis.Throws {
			if fn.IsAsync {
				sb.WriteString(fmt.Sprintf("    await expect(%s(%s)).rejects.toThrow();\n", fn.Name, args))
			} else {
				sb.WriteString(fmt.Sprintf("    expect(() => %s(%s)).toThrow();\n", fn.Name, args))
			}
		} else {
			if fn.IsAsync {
				sb.WriteString(fmt.Sprintf("    const result = await %s(%s);\n", fn.Name, args))
			} else {
				sb.WriteString(fmt.Sprintf("    const result = %s(%s);\n", fn.Name, args))
			}
			boundary := b
			sb.WriteString(genJSResultAssertionWithArgs(fn.Analysis, fn.Params, &boundary, "    "))
		}
		sb.WriteString("  });\n\n")
	}

	if fn.Analysis.Throws {
		sb.WriteString(fmt.Sprintf("  it('should throw on invalid input', %s => {\n", jsAsyncArrow(fn.IsAsync)))
		args := jsInvalidArgList(fn.Params, fn.Analysis.Boundaries)
		if fn.IsAsync {
			sb.WriteString(fmt.Sprintf("    await expect(%s(%s)).rejects.toThrow();\n", fn.Name, args))
		} else {
			sb.WriteString(fmt.Sprintf("    expect(() => %s(%s)).toThrow();\n", fn.Name, args))
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
		sb.WriteString(genJSResultAssertionWithArgs(fn.Analysis, fn.Params, nil, "    "))
		sb.WriteString("  });\n\n")
	}

	sb.WriteString("});\n\n")

	return sb.String()
}

func genJestFuncTestForCoverageTask(fn jsFuncInfo, task *types.CoverageTestTask) string {
	var sb strings.Builder
	testName := jsCoverageTaskTestName(task, "should cover "+fn.Name+" coverage gap")
	boundary := jsBoundaryForCoverageTask(fn.Analysis.Boundaries, task)
	args := jsArgListForCoverageTask(fn.Params, task, boundary)
	assertions := jsAssertionStyleForCoverageTask(task)

	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", fn.Name))
	sb.WriteString(fmt.Sprintf("  it('%s', %s => {\n", jsEscapeTestNameValue(testName), jsAsyncArrow(fn.IsAsync)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	if fn.Analysis.Throws || task.GapType == "error_path" {
		sb.WriteString(genJSErrorAssertion(assertions, fn.IsAsync, fmt.Sprintf("%s(%s)", fn.Name, args), "    "))
		sb.WriteString("  });\n\n")
		sb.WriteString("});\n\n")
		return sb.String()
	}

	if fn.IsAsync {
		sb.WriteString(fmt.Sprintf("    const result = await %s(%s);\n", fn.Name, args))
	} else {
		sb.WriteString(fmt.Sprintf("    const result = %s(%s);\n", fn.Name, args))
	}
	sb.WriteString(genJSResultAssertionWithTaskArgsStyle(fn.Analysis, fn.Params, boundary, coverageTaskInputValues(task, "javascript"), "    ", assertions))
	sb.WriteString("  });\n\n")
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
		sb.WriteString(genJSResultAssertionWithArgs(method.Analysis, method.Params, nil, "      "))
		sb.WriteString("    });\n\n")

		if method.Analysis.Throws {
			sb.WriteString(fmt.Sprintf("    it('should throw on invalid input', %s => {\n", jsAsyncArrow(method.IsAsync)))
			sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
			args := jsInvalidArgList(method.Params, method.Analysis.Boundaries)
			if method.IsAsync {
				sb.WriteString(fmt.Sprintf("      await expect(instance.%s(%s)).rejects.toThrow();\n", method.Name, args))
			} else {
				sb.WriteString(fmt.Sprintf("      expect(() => instance.%s(%s)).toThrow();\n", method.Name, args))
			}
			sb.WriteString("    });\n\n")
		}

		for _, b := range method.Analysis.Boundaries {
			if !jsParamExists(method.Params, b.Param) {
				continue
			}
			sb.WriteString(fmt.Sprintf("    it('should handle %s = %s', %s => {\n",
				b.Param, jsEscapeTestNameValue(b.Value), jsAsyncArrow(method.IsAsync)))
			sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
			args := jsArgListWithBoundary(method.Params, b)
			if method.Analysis.Throws {
				if method.IsAsync {
					sb.WriteString(fmt.Sprintf("      await expect(instance.%s(%s)).rejects.toThrow();\n", method.Name, args))
				} else {
					sb.WriteString(fmt.Sprintf("      expect(() => instance.%s(%s)).toThrow();\n", method.Name, args))
				}
			} else {
				if method.IsAsync {
					sb.WriteString(fmt.Sprintf("      const result = await instance.%s(%s);\n", method.Name, args))
				} else {
					sb.WriteString(fmt.Sprintf("      const result = instance.%s(%s);\n", method.Name, args))
				}
				boundary := b
				sb.WriteString(genJSResultAssertionWithArgs(method.Analysis, method.Params, &boundary, "      "))
			}
			sb.WriteString("    });\n\n")
		}

		sb.WriteString("  });\n\n")
	}

	sb.WriteString("});\n\n")

	return sb.String()
}

func genJestClassTestForCoverageTask(cls jsClassInfo, task *types.CoverageTestTask) string {
	var sb strings.Builder
	assertions := jsAssertionStyleForCoverageTask(task)

	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", cls.Name))
	for _, method := range cls.Methods {
		testName := jsCoverageTaskTestName(task, "should cover "+method.Name+" coverage gap")
		boundary := jsBoundaryForCoverageTask(method.Analysis.Boundaries, task)
		args := jsArgListForCoverageTask(method.Params, task, boundary)

		sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
		sb.WriteString(fmt.Sprintf("    it('%s', %s => {\n", jsEscapeTestNameValue(testName), jsAsyncArrow(method.IsAsync)))
		if comment := coverageTaskComment(task); comment != "" {
			sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
		}
		sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
		if method.Analysis.Throws || task.GapType == "error_path" {
			sb.WriteString(genJSErrorAssertion(assertions, method.IsAsync, fmt.Sprintf("instance.%s(%s)", method.Name, args), "      "))
			sb.WriteString("    });\n\n")
			sb.WriteString("  });\n\n")
			continue
		}
		if method.IsAsync {
			sb.WriteString(fmt.Sprintf("      const result = await instance.%s(%s);\n", method.Name, args))
		} else {
			sb.WriteString(fmt.Sprintf("      const result = instance.%s(%s);\n", method.Name, args))
		}
		sb.WriteString(genJSResultAssertionWithTaskArgsStyle(method.Analysis, method.Params, boundary, coverageTaskInputValues(task, "javascript"), "      ", assertions))
		sb.WriteString("    });\n\n")
		sb.WriteString("  });\n\n")
	}
	sb.WriteString("});\n\n")

	return sb.String()
}

func genJSResultAssertion(a jsFuncAnalysis, indent string) string {
	return genJSResultAssertionWithArgs(a, nil, nil, indent)
}

func genJSResultAssertionWithArgs(a jsFuncAnalysis, params []jsParamInfo, boundary *jsBoundary, indent string) string {
	return genJSResultAssertionWithTaskArgs(a, params, boundary, nil, indent)
}

func genJSResultAssertionWithTaskArgs(a jsFuncAnalysis, params []jsParamInfo, boundary *jsBoundary, values map[string]string, indent string) string {
	return genJSResultAssertionWithTaskArgsStyle(a, params, boundary, values, indent, jsAssertionStyleJest)
}

type jsAssertionStyle string

const (
	jsAssertionStyleJest jsAssertionStyle = "jest"
	jsAssertionStyleChai jsAssertionStyle = "chai"
)

func jsAssertionStyleForCoverageTask(task *types.CoverageTestTask) jsAssertionStyle {
	if task != nil && strings.EqualFold(task.Framework, "mocha") {
		return jsAssertionStyleChai
	}
	return jsAssertionStyleJest
}

func genJSErrorAssertion(style jsAssertionStyle, isAsync bool, callExpr string, indent string) string {
	if style == jsAssertionStyleChai {
		if isAsync {
			return indent + "let caughtError;\n" +
				indent + "try {\n" +
				indent + "  await " + callExpr + ";\n" +
				indent + "} catch (err) {\n" +
				indent + "  caughtError = err;\n" +
				indent + "}\n" +
				indent + "expect(caughtError).to.exist;\n"
		}
		return fmt.Sprintf("%sexpect(() => %s).to.throw();\n", indent, callExpr)
	}
	if isAsync {
		return fmt.Sprintf("%sawait expect(%s).rejects.toThrow();\n", indent, callExpr)
	}
	return fmt.Sprintf("%sexpect(() => %s).toThrow();\n", indent, callExpr)
}

func genJSResultAssertionWithTaskArgsStyle(a jsFuncAnalysis, params []jsParamInfo, boundary *jsBoundary, values map[string]string, indent string, style jsAssertionStyle) string {
	var sb strings.Builder

	if !a.HasReturn {
		sb.WriteString(indent + "// void function, verify no exception\n")
		return sb.String()
	}

	if expected, ok := jsExpectedReturnExprWithValues(a, params, boundary, values); ok {
		if style == jsAssertionStyleChai {
			sb.WriteString(indent + "expect(result).to.equal(" + expected + ");\n")
		} else {
			sb.WriteString(indent + "expect(result).toBe(" + expected + ");\n")
		}
		return sb.String()
	}

	if style == jsAssertionStyleChai {
		switch a.ReturnType {
		case "number":
			sb.WriteString(indent + "expect(result).to.be.a('number');\n")
			sb.WriteString(indent + "expect(Number.isNaN(result)).to.equal(false);\n")
		case "string":
			sb.WriteString(indent + "expect(result).to.be.a('string');\n")
			sb.WriteString(indent + "expect(result.length).to.be.at.least(0);\n")
		case "boolean":
			sb.WriteString(indent + "expect(result).to.be.a('boolean');\n")
		case "array":
			sb.WriteString(indent + "expect(Array.isArray(result)).to.equal(true);\n")
		case "object":
			sb.WriteString(indent + "expect(result).to.be.an('object');\n")
			sb.WriteString(indent + "expect(result).to.not.equal(null);\n")
		case "null":
			sb.WriteString(indent + "expect(result).to.equal(null);\n")
		case "undefined":
			sb.WriteString(indent + "expect(result).to.equal(undefined);\n")
		default:
			sb.WriteString(indent + "expect(result).to.not.equal(undefined);\n")
		}
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

func jsExpectedReturnExpr(a jsFuncAnalysis, params []jsParamInfo, boundary *jsBoundary) (string, bool) {
	return jsExpectedReturnExprWithValues(a, params, boundary, nil)
}

func jsExpectedReturnExprWithValues(a jsFuncAnalysis, params []jsParamInfo, boundary *jsBoundary, values map[string]string) (string, bool) {
	expr := ""
	if boundary != nil && boundary.ReturnExpr != "" {
		expr = strings.TrimSpace(boundary.ReturnExpr)
	} else if len(a.Returns) == 1 {
		expr = strings.TrimSpace(a.Returns[0])
	} else if len(a.Returns) > 1 {
		expr = strings.TrimSpace(a.Returns[len(a.Returns)-1])
	}
	if expr == "" {
		return "", false
	}
	expr = strings.TrimSpace(strings.TrimSuffix(expr, ";"))
	if !jsReturnExprIsSafe(expr) {
		return "", false
	}

	for i, p := range params {
		if p.IsRest {
			continue
		}
		value := jsArgValue(p, i)
		if boundary != nil && p.Name == boundary.Param {
			value = boundary.Value
		}
		if values != nil && values[p.Name] != "" {
			value = values[p.Name]
		}
		expr = replaceIdentifier(expr, p.Name, value)
	}

	if hasUnknownIdentifiers(stripQuotedLiterals(expr), map[string]bool{
		"true": true, "false": true, "null": true, "undefined": true,
	}) {
		return "", false
	}
	return "(" + expr + ")", true
}

func jsReturnExprIsSafe(expr string) bool {
	if expr == "" || strings.ContainsAny(expr, "\n;{}[]") {
		return false
	}
	for _, blocked := range []string{"await ", "function", "=>", "new ", "this.", "(", ")("} {
		if strings.Contains(expr, blocked) {
			return false
		}
	}
	return true
}

// ---- 辅助函数 ----

func jsAsyncArrow(isAsync bool) string {
	if isAsync {
		return "async ()"
	}
	return "()"
}

func jsEscapeTestNameValue(value string) string {
	return strings.ReplaceAll(value, "'", "\\'")
}

func jsCoverageTaskTestName(task *types.CoverageTestTask, fallback string) string {
	if task == nil || strings.TrimSpace(task.TestName) == "" {
		return fallback
	}
	return strings.TrimSpace(task.TestName)
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
		} else {
			args[i] = jsArgValue(p, i)
		}
	}
	return strings.Join(args, ", ")
}

func jsArgListForCoverageTask(params []jsParamInfo, task *types.CoverageTestTask, boundary *jsBoundary) string {
	values := coverageTaskInputValues(task, "javascript")
	if boundary != nil {
		values[boundary.Param] = boundary.Value
	}
	return jsArgListWithValues(params, values)
}

func jsArgListWithValues(params []jsParamInfo, values map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	args := make([]string, len(params))
	for i, p := range params {
		if value := values[p.Name]; value != "" {
			args[i] = value
		} else {
			args[i] = jsArgValue(p, i)
		}
	}
	return strings.Join(args, ", ")
}

func jsBoundaryForCoverageTask(boundaries []jsBoundary, task *types.CoverageTestTask) *jsBoundary {
	values := coverageTaskInputValues(task, "javascript")
	for _, b := range boundaries {
		if values[b.Param] == b.Value {
			boundary := b
			return &boundary
		}
	}
	if task != nil && (task.GapType == "branch" || task.GapType == "error_path") && len(boundaries) == 1 {
		boundary := boundaries[0]
		return &boundary
	}
	return nil
}

func jsArgList(params []jsParamInfo) string {
	if len(params) == 0 {
		return ""
	}
	args := make([]string, len(params))
	for i, p := range params {
		args[i] = jsArgValue(p, i)
	}
	return strings.Join(args, ", ")
}

func jsInvalidArgList(params []jsParamInfo, boundaries []jsBoundary) string {
	for _, b := range boundaries {
		if b.Value == "null" || b.Value == "undefined" {
			return jsArgListWithBoundary(params, b)
		}
	}
	return jsPlaceholderArgList(params)
}

func jsPlaceholderArgList(params []jsParamInfo) string {
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

func jsArgValue(p jsParamInfo, _ int) string {
	if p.IsRest {
		return "[]"
	}

	name := strings.ToLower(p.Name)
	compact := strings.ReplaceAll(strings.ReplaceAll(name, "_", ""), "-", "")

	if jsNameHasPrefix(compact, "is", "has", "can", "should") ||
		jsNameHasAny(compact, "enabled", "active", "valid", "visible", "flag", "checked") {
		return "true"
	}
	if jsNameHasAny(compact, "items", "list", "array", "arr", "rows", "records", "args") {
		return "[]"
	}
	if jsNameIsNumeric(compact) {
		if compact == "b" || compact == "y" {
			return "2"
		}
		return "1"
	}
	if jsNameHasAny(compact, "options", "opts", "config", "payload", "data", "body", "params", "query", "user", "metadata") {
		return "{}"
	}
	if jsNameHasAny(compact, "url", "uri", "endpoint", "href") {
		return "'https://example.com'"
	}
	if jsNameHasAny(compact, "email") {
		return "'user@example.com'"
	}
	if jsNameHasAny(compact, "name", "title", "text", "message", "prefix", "suffix", "label", "path", "key", "value", "type", "mode") {
		return "'test'"
	}
	if p.HasDefault {
		return "undefined"
	}

	return "undefined"
}

func jsNameHasAny(name string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(name, part) {
			return true
		}
	}
	return false
}

func jsNameHasPrefix(name string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) && len(name) > len(prefix) {
			return true
		}
	}
	return false
}

func jsNameIsNumeric(name string) bool {
	switch name {
	case "a", "b", "x", "y", "n", "num", "number", "count", "size", "index", "idx",
		"age", "page", "limit", "offset", "total", "amount", "price", "id":
		return true
	}
	return strings.HasSuffix(name, "id") || strings.HasSuffix(name, "count") ||
		strings.HasSuffix(name, "index") || strings.HasSuffix(name, "size")
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
