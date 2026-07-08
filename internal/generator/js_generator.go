package generator

import (
	"encoding/json"
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
	ReturnType     string // number/string/array/object/boolean/null/undefined/unknown
	ReturnTypeExpr string // TypeScript return annotation, if available
	TSTypeDecls    map[string]string
	Returns        []string     // return expressions found in the function body
	Throws         bool         // 函数体包含 throw
	Boundaries     []jsBoundary // 边界条件检测
	HasReturn      bool         // 是否有 return 语句（非 void）
	IsGetter       bool         // 是否是简单的 getter（return expression 只有一个变量/字面量）
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

// GenerateJestTests reads JS/TS source and generates default Jest-style tests.
func GenerateJestTests(srcPath string) (string, error) {
	return GenerateJavaScriptTestsWithFramework(srcPath, "jest")
}

// GenerateJavaScriptTestsWithFramework reads JS/TS source and generates tests
// for the requested JavaScript framework. Empty, Jest, and Vitest use Jest-style
// matchers; Mocha uses Chai assertions.
func GenerateJavaScriptTestsWithFramework(srcPath, framework string) (string, error) {
	framework = normalizeJavaScriptTestFramework(framework)
	return generateJavaScriptTests(srcPath, &types.CoverageTestTask{Framework: framework}, false)
}

// GenerateJavaScriptTestsForCoverageTask generates JS/TS tests from a coverage
// task. The task framework controls matcher/import style for Jest, Vitest, and
// Mocha while keeping the same static JS/TS parser pipeline.
func GenerateJavaScriptTestsForCoverageTask(srcPath string, task *types.CoverageTestTask) (string, error) {
	if task == nil {
		return GenerateJestTests(srcPath)
	}
	return generateJavaScriptTests(srcPath, task, true)
}

func normalizeJavaScriptTestFramework(framework string) string {
	switch strings.ToLower(strings.TrimSpace(framework)) {
	case "mocha":
		return "mocha"
	case "vitest":
		return "vitest"
	default:
		return "jest"
	}
}

// GenerateJestTestsForCoverageTask is kept for compatibility with existing
// callers. New code should call GenerateJavaScriptTestsForCoverageTask.
func GenerateJestTestsForCoverageTask(srcPath string, task *types.CoverageTestTask) (string, error) {
	return GenerateJavaScriptTestsForCoverageTask(srcPath, task)
}

func generateJavaScriptTests(srcPath string, task *types.CoverageTestTask, coverageMode bool) (string, error) {
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
	moduleImportPath := jsSourceModuleImportPath(srcPath)

	var buf strings.Builder

	mochaTask := task != nil && strings.EqualFold(task.Framework, "mocha")
	vitestTask := task != nil && strings.EqualFold(task.Framework, "vitest")
	if mochaTask && isESModule {
		buf.WriteString("import { expect } from 'chai';\n")
	} else if mochaTask {
		buf.WriteString("const { expect } = require('chai');\n")
	} else if vitestTask && isESModule {
		buf.WriteString("import { describe, it, expect } from 'vitest';\n")
	}
	if isESModule {
		buf.WriteString(fmt.Sprintf("import { %s } from '%s';\n\n", joinExportNames(funcs, classes), moduleImportPath))
	} else {
		buf.WriteString(fmt.Sprintf("const { %s } = require('./%s');\n\n", joinExportNames(funcs, classes), moduleName))
	}

	for _, fn := range funcs {
		if coverageMode {
			buf.WriteString(genJSFuncTestForCoverageTask(fn, task))
		} else if task != nil {
			buf.WriteString(genJSFuncTest(fn, jsAssertionStyleForTask(task)))
		} else {
			buf.WriteString(genJSFuncTest(fn, jsAssertionStyleJest))
		}
	}

	for _, cls := range classes {
		if coverageMode {
			buf.WriteString(genJSClassTestForCoverageTask(cls, task))
		} else if task != nil {
			buf.WriteString(genJSClassTest(cls, jsAssertionStyleForTask(task)))
		} else {
			buf.WriteString(genJSClassTest(cls, jsAssertionStyleJest))
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
	jsIfReturnRe = regexp.MustCompile(`(?s)if\s*\(\s*(\w+)\s*(?:===?|==)\s*([^)]+?)\s*\)\s*(?:\{\s*)?return\s+(\{[^;\n]*\}|\[[^;\n]*\]|.+?)(?:;|\n|\})`)
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

func genJSFuncTest(fn jsFuncInfo, assertions jsAssertionStyle) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", fn.Name))

	sb.WriteString(fmt.Sprintf("  it('should return expected result for normal input', %s => {\n", jsAsyncArrow(fn.IsAsync)))
	setup, callValues, clientCall := genJSInjectedClientSetup(fn.Params, fn.Analysis, "    ")
	sb.WriteString(setup)
	args := jsArgListWithValuesForAnalysis(fn.Params, callValues, &fn.Analysis)
	if fn.IsAsync {
		sb.WriteString(fmt.Sprintf("    const result = await %s(%s);\n", fn.Name, args))
	} else {
		sb.WriteString(fmt.Sprintf("    const result = %s(%s);\n", fn.Name, args))
	}
	sb.WriteString(genJSResultAssertionWithArgsStyle(fn.Analysis, fn.Params, nil, "    ", assertions))
	sb.WriteString(genJSInjectedClientCallAssertion(clientCall, "    ", assertions))
	sb.WriteString("  });\n\n")

	for _, b := range fn.Analysis.Boundaries {
		if !jsParamExists(fn.Params, b.Param) {
			continue
		}
		sb.WriteString(fmt.Sprintf("  it('should handle %s = %s', %s => {\n",
			b.Param, jsEscapeTestNameValue(b.Value), jsAsyncArrow(fn.IsAsync)))
		args := jsArgListWithBoundary(fn.Params, b)
		if fn.Analysis.Throws {
			sb.WriteString(genJSErrorAssertion(assertions, fn.IsAsync, fmt.Sprintf("%s(%s)", fn.Name, args), "    "))
		} else {
			if fn.IsAsync {
				sb.WriteString(fmt.Sprintf("    const result = await %s(%s);\n", fn.Name, jsArgListWithBoundaryForAnalysis(fn.Params, b, fn.Analysis)))
			} else {
				sb.WriteString(fmt.Sprintf("    const result = %s(%s);\n", fn.Name, jsArgListWithBoundaryForAnalysis(fn.Params, b, fn.Analysis)))
			}
			boundary := b
			sb.WriteString(genJSResultAssertionWithArgsStyle(fn.Analysis, fn.Params, &boundary, "    ", assertions))
		}
		sb.WriteString("  });\n\n")
	}

	if fn.Analysis.Throws {
		args := jsInvalidArgList(fn.Params, fn.Analysis.Boundaries)
		if !jsErrorBoundaryArgsExist(fn.Params, fn.Analysis.Boundaries, args) {
			sb.WriteString(fmt.Sprintf("  it('should throw on invalid input', %s => {\n", jsAsyncArrow(fn.IsAsync)))
			sb.WriteString(genJSErrorAssertion(assertions, fn.IsAsync, fmt.Sprintf("%s(%s)", fn.Name, args), "    "))
			sb.WriteString("  });\n\n")
		}
	}

	if len(fn.Params) == 0 && fn.Analysis.HasReturn {
		sb.WriteString(fmt.Sprintf("  it('should work with no arguments', %s => {\n", jsAsyncArrow(fn.IsAsync)))
		if fn.IsAsync {
			sb.WriteString(fmt.Sprintf("    const result = await %s();\n", fn.Name))
		} else {
			sb.WriteString(fmt.Sprintf("    const result = %s();\n", fn.Name))
		}
		sb.WriteString(genJSResultAssertionWithArgsStyle(fn.Analysis, fn.Params, nil, "    ", assertions))
		sb.WriteString("  });\n\n")
	}

	sb.WriteString("});\n\n")

	return sb.String()
}

func genJestFuncTest(fn jsFuncInfo) string {
	return genJSFuncTest(fn, jsAssertionStyleJest)
}

func genJSFuncTestForCoverageTask(fn jsFuncInfo, task *types.CoverageTestTask) string {
	var sb strings.Builder
	testName := jsCoverageTaskTestName(task, "should cover "+fn.Name+" coverage gap")
	boundary := jsBoundaryForCoverageTask(fn.Analysis.Boundaries, task)
	args := jsArgListForCoverageTask(fn.Params, task, boundary, fn.Analysis, nil)
	assertions := jsAssertionStyleForTask(task)

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

	setup, callValues, clientCall := genJSInjectedClientSetup(fn.Params, fn.Analysis, "    ")
	args = jsArgListForCoverageTask(fn.Params, task, boundary, fn.Analysis, callValues)
	sb.WriteString(setup)
	if fn.IsAsync {
		sb.WriteString(fmt.Sprintf("    const result = await %s(%s);\n", fn.Name, args))
	} else {
		sb.WriteString(fmt.Sprintf("    const result = %s(%s);\n", fn.Name, args))
	}
	sb.WriteString(genJSResultAssertionWithTaskArgsStyle(fn.Analysis, fn.Params, boundary, coverageTaskInputValues(task, "javascript"), "    ", assertions))
	sb.WriteString(genJSInjectedClientCallAssertion(clientCall, "    ", assertions))
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func genJSClassTest(cls jsClassInfo, assertions jsAssertionStyle) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", cls.Name))

	sb.WriteString(fmt.Sprintf("  describe('constructor', () => {\n"))
	sb.WriteString(fmt.Sprintf("    it('should create an instance', () => {\n"))
	sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
	if assertions == jsAssertionStyleChai {
		sb.WriteString(fmt.Sprintf("      expect(instance).to.be.instanceOf(%s);\n", cls.Name))
	} else {
		sb.WriteString(fmt.Sprintf("      expect(instance).toBeInstanceOf(%s);\n", cls.Name))
	}
	sb.WriteString("    });\n")
	sb.WriteString("  });\n\n")

	for _, method := range cls.Methods {
		sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))

		sb.WriteString(fmt.Sprintf("    it('should return expected result', %s => {\n", jsAsyncArrow(method.IsAsync)))
		sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
		if method.IsAsync {
			sb.WriteString(fmt.Sprintf("      const result = await instance.%s(%s);\n", method.Name, jsArgListForAnalysis(method.Params, method.Analysis)))
		} else {
			sb.WriteString(fmt.Sprintf("      const result = instance.%s(%s);\n", method.Name, jsArgListForAnalysis(method.Params, method.Analysis)))
		}
		sb.WriteString(genJSResultAssertionWithArgsStyle(method.Analysis, method.Params, nil, "      ", assertions))
		sb.WriteString("    });\n\n")

		if method.Analysis.Throws {
			args := jsInvalidArgList(method.Params, method.Analysis.Boundaries)
			if !jsErrorBoundaryArgsExist(method.Params, method.Analysis.Boundaries, args) {
				sb.WriteString(fmt.Sprintf("    it('should throw on invalid input', %s => {\n", jsAsyncArrow(method.IsAsync)))
				sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
				sb.WriteString(genJSErrorAssertion(assertions, method.IsAsync, fmt.Sprintf("instance.%s(%s)", method.Name, args), "      "))
				sb.WriteString("    });\n\n")
			}
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
				sb.WriteString(genJSErrorAssertion(assertions, method.IsAsync, fmt.Sprintf("instance.%s(%s)", method.Name, args), "      "))
			} else {
				if method.IsAsync {
					sb.WriteString(fmt.Sprintf("      const result = await instance.%s(%s);\n", method.Name, jsArgListWithBoundaryForAnalysis(method.Params, b, method.Analysis)))
				} else {
					sb.WriteString(fmt.Sprintf("      const result = instance.%s(%s);\n", method.Name, jsArgListWithBoundaryForAnalysis(method.Params, b, method.Analysis)))
				}
				boundary := b
				sb.WriteString(genJSResultAssertionWithArgsStyle(method.Analysis, method.Params, &boundary, "      ", assertions))
			}
			sb.WriteString("    });\n\n")
		}

		sb.WriteString("  });\n\n")
	}

	sb.WriteString("});\n\n")

	return sb.String()
}

func genJestClassTest(cls jsClassInfo, isESModule bool, moduleName string) string {
	return genJSClassTest(cls, jsAssertionStyleJest)
}

func genJSClassTestForCoverageTask(cls jsClassInfo, task *types.CoverageTestTask) string {
	var sb strings.Builder
	assertions := jsAssertionStyleForTask(task)

	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", cls.Name))
	for _, method := range cls.Methods {
		testName := jsCoverageTaskTestName(task, "should cover "+method.Name+" coverage gap")
		boundary := jsBoundaryForCoverageTask(method.Analysis.Boundaries, task)
		args := jsArgListForCoverageTask(method.Params, task, boundary, method.Analysis, nil)

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

func genJSResultAssertionWithArgsStyle(a jsFuncAnalysis, params []jsParamInfo, boundary *jsBoundary, indent string, style jsAssertionStyle) string {
	return genJSResultAssertionWithTaskArgsStyle(a, params, boundary, nil, indent, style)
}

type jsAssertionStyle string

const (
	jsAssertionStyleJest jsAssertionStyle = "jest"
	jsAssertionStyleChai jsAssertionStyle = "chai"
)

func jsAssertionStyleForTask(task *types.CoverageTestTask) jsAssertionStyle {
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

	if expected, ok, deepEqual := jsExpectedReturnExprWithValuesKind(a, params, boundary, values); ok {
		if style == jsAssertionStyleChai && deepEqual {
			sb.WriteString(indent + "expect(result).to.deep.equal(" + expected + ");\n")
		} else if style == jsAssertionStyleChai {
			sb.WriteString(indent + "expect(result).to.equal(" + expected + ");\n")
		} else if deepEqual {
			sb.WriteString(indent + "expect(result).toEqual(" + expected + ");\n")
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
	expr, ok, _ := jsExpectedReturnExprWithValuesKind(a, params, boundary, values)
	return expr, ok
}

func jsExpectedReturnExprWithValuesKind(a jsFuncAnalysis, params []jsParamInfo, boundary *jsBoundary, values map[string]string) (string, bool, bool) {
	expr := ""
	if boundary != nil && boundary.ReturnExpr != "" {
		expr = strings.TrimSpace(boundary.ReturnExpr)
	} else if len(a.Returns) == 1 {
		expr = strings.TrimSpace(a.Returns[0])
	} else if len(a.Returns) > 1 {
		expr = strings.TrimSpace(a.Returns[len(a.Returns)-1])
	}
	if expr == "" {
		return "", false, false
	}
	expr = strings.TrimSpace(strings.TrimSuffix(expr, ";"))
	if expected, ok := jsExpectedResponseJSONReturn(expr, params, a); ok {
		return expected, true, true
	}
	if expected, ok := jsExpectedInjectedClientReturn(expr, params, a); ok {
		return expected, true, true
	}
	if !jsReturnExprIsSafe(expr) {
		return "", false, false
	}

	deepEqual := jsReturnExprIsSimpleLiteral(expr)
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
		if deepEqual && jsReturnExprIsSimpleObjectLiteral(expr) {
			expr = replaceIdentifierInJSObjectLiteral(expr, p.Name, value)
		} else {
			expr = replaceIdentifier(expr, p.Name, value)
		}
	}

	identifierSource := stripQuotedLiterals(expr)
	if deepEqual && jsReturnExprIsSimpleObjectLiteral(expr) {
		identifierSource = stripJSObjectPropertyKeys(identifierSource)
	}
	if hasUnknownIdentifiers(identifierSource, map[string]bool{
		"true": true, "false": true, "null": true, "undefined": true,
	}) {
		return "", false, false
	}
	if deepEqual {
		return expr, true, true
	}
	return "(" + expr + ")", true, false
}

func jsExpectedResponseJSONReturn(expr string, params []jsParamInfo, analysis jsFuncAnalysis) (string, bool) {
	receiver, ok := jsResponseJSONReceiver(expr)
	if !ok {
		return "", false
	}
	for _, p := range params {
		if p.Name == receiver {
			return jsMockPayloadForAnalysis(analysis), true
		}
	}
	return "", false
}

func jsResponseJSONReceiver(expr string) (string, bool) {
	expr = strings.TrimSpace(strings.TrimSuffix(expr, ";"))
	expr = strings.TrimPrefix(expr, "await ")
	expr = strings.TrimSpace(expr)
	re := regexp.MustCompile(`^([A-Za-z_$][A-Za-z0-9_$]*)\.json\(\)$`)
	matches := re.FindStringSubmatch(expr)
	if len(matches) != 2 {
		return "", false
	}
	return matches[1], true
}

func jsExpectedInjectedClientReturn(expr string, params []jsParamInfo, analysis jsFuncAnalysis) (string, bool) {
	receiver, _, _, ok := jsInjectedClientCall(expr)
	if !ok {
		return "", false
	}
	for _, p := range params {
		if p.Name == receiver && jsNameLooksLikeClientParam(jsCompactName(p.Name)) {
			return jsMockPayloadForAnalysis(analysis), true
		}
	}
	return "", false
}

func jsInjectedClientCall(expr string) (receiver, method, args string, ok bool) {
	expr = strings.TrimSpace(strings.TrimSuffix(expr, ";"))
	expr = strings.TrimPrefix(expr, "await ")
	expr = strings.TrimSpace(expr)
	re := regexp.MustCompile(`^([A-Za-z_$][A-Za-z0-9_$]*)\.(get|fetch|request)\s*\((.*)\)$`)
	matches := re.FindStringSubmatch(expr)
	if len(matches) != 4 {
		return "", "", "", false
	}
	return matches[1], matches[2], strings.TrimSpace(matches[3]), true
}

func jsReturnExprIsSafe(expr string) bool {
	if expr == "" {
		return false
	}
	for _, blocked := range []string{"await ", "function", "=>", "new ", "this.", "(", ")("} {
		if strings.Contains(expr, blocked) {
			return false
		}
	}
	if jsReturnExprIsSimpleLiteral(expr) {
		return !strings.ContainsAny(expr, "\n;")
	}
	if strings.ContainsAny(expr, "\n;{}[]") {
		return false
	}
	return true
}

func jsReturnExprIsSimpleLiteral(expr string) bool {
	return jsReturnExprIsSimpleObjectLiteral(expr) || jsReturnExprIsSimpleArrayLiteral(expr)
}

func jsReturnExprIsSimpleObjectLiteral(expr string) bool {
	expr = strings.TrimSpace(expr)
	return strings.HasPrefix(expr, "{") && strings.HasSuffix(expr, "}")
}

func jsReturnExprIsSimpleArrayLiteral(expr string) bool {
	expr = strings.TrimSpace(expr)
	return strings.HasPrefix(expr, "[") && strings.HasSuffix(expr, "]")
}

func replaceIdentifierInJSObjectLiteral(expr, name, value string) string {
	if name == "" || !jsReturnExprIsSimpleObjectLiteral(expr) {
		return expr
	}
	inner := strings.TrimSpace(expr[1 : len(expr)-1])
	if inner == "" {
		return expr
	}
	parts := splitTopLevelJSCSV(inner)
	for i, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" || strings.HasPrefix(trimmed, "...") {
			continue
		}
		if key, val, ok := splitJSObjectProperty(trimmed); ok {
			parts[i] = key + ": " + replaceIdentifier(val, name, value)
			continue
		}
		if trimmed == name {
			parts[i] = name + ": " + value
		}
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func splitJSObjectProperty(prop string) (string, string, bool) {
	depth := 0
	inQuote := rune(0)
	escaped := false
	for i, ch := range prop {
		if inQuote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"', '`':
			inQuote = ch
		case '{', '[':
			depth++
		case '}', ']':
			if depth > 0 {
				depth--
			}
		case ':':
			if depth == 0 {
				return strings.TrimSpace(prop[:i]), strings.TrimSpace(prop[i+1:]), true
			}
		}
	}
	return "", "", false
}

func splitTopLevelJSCSV(input string) []string {
	var parts []string
	start := 0
	depth := 0
	inQuote := rune(0)
	escaped := false
	for i, ch := range input {
		if inQuote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"', '`':
			inQuote = ch
		case '{', '[':
			depth++
		case '}', ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(input[start:i]))
				start = i + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(input[start:]))
	return parts
}

func stripJSObjectPropertyKeys(expr string) string {
	re := regexp.MustCompile(`\b[A-Za-z_$][A-Za-z0-9_$]*\s*:`)
	return re.ReplaceAllString(expr, " ")
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

func jsArgListWithBoundaryForAnalysis(params []jsParamInfo, b jsBoundary, analysis jsFuncAnalysis) string {
	values := map[string]string{b.Param: b.Value}
	return jsArgListWithValuesForAnalysis(params, values, &analysis)
}

func jsArgListForCoverageTask(params []jsParamInfo, task *types.CoverageTestTask, boundary *jsBoundary, analysis jsFuncAnalysis, overrides map[string]string) string {
	values := coverageTaskInputValues(task, "javascript")
	if boundary != nil {
		values[boundary.Param] = boundary.Value
	}
	for name, value := range overrides {
		values[name] = value
	}
	return jsArgListWithValuesForAnalysis(params, values, &analysis)
}

func jsArgListWithValues(params []jsParamInfo, values map[string]string) string {
	return jsArgListWithValuesForAnalysis(params, values, nil)
}

func jsArgListWithValuesForAnalysis(params []jsParamInfo, values map[string]string, analysis *jsFuncAnalysis) string {
	if len(params) == 0 {
		return ""
	}
	args := make([]string, len(params))
	for i, p := range params {
		if value := values[p.Name]; value != "" {
			args[i] = value
		} else if method := jsInjectedClientMethodForParam(analysis, p.Name); method != "" {
			args[i] = jsInjectedClientMockWithPayload(method, jsMockPayloadForAnalysisPtr(analysis))
		} else if jsNameLooksLikeResponseParam(jsCompactName(p.Name)) {
			args[i] = jsResponseJSONMock(jsMockPayloadForAnalysisPtr(analysis))
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
	return jsArgListForAnalysis(params, jsFuncAnalysis{})
}

func jsArgListForAnalysis(params []jsParamInfo, analysis jsFuncAnalysis) string {
	if len(params) == 0 {
		return ""
	}
	return jsArgListWithValuesForAnalysis(params, nil, &analysis)
}

func jsInvalidArgList(params []jsParamInfo, boundaries []jsBoundary) string {
	for _, b := range boundaries {
		if b.Value == "null" || b.Value == "undefined" {
			return jsArgListWithBoundary(params, b)
		}
	}
	return jsPlaceholderArgList(params)
}

func jsErrorBoundaryArgsExist(params []jsParamInfo, boundaries []jsBoundary, args string) bool {
	for _, b := range boundaries {
		if !jsParamExists(params, b.Param) {
			continue
		}
		if b.Value != "null" && b.Value != "undefined" {
			continue
		}
		if jsArgListWithBoundary(params, b) == args {
			return true
		}
	}
	return false
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
	compact := jsCompactName(name)

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
	if jsNameLooksLikeResponseParam(compact) {
		return jsResponseJSONMock("{ ok: true }")
	}
	if jsNameLooksLikeClientParam(compact) {
		return jsInjectedClientMock("get")
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

func jsCompactName(name string) string {
	name = strings.ToLower(name)
	return strings.ReplaceAll(strings.ReplaceAll(name, "_", ""), "-", "")
}

func jsInjectedClientMethodForParam(analysis *jsFuncAnalysis, param string) string {
	if info := jsInjectedClientCallForParam(analysis, param); info != nil {
		return info.Method
	}
	return ""
}

type jsInjectedClientCallInfo struct {
	Param  string
	Method string
	Args   string
}

func genJSInjectedClientSetup(params []jsParamInfo, analysis jsFuncAnalysis, indent string) (string, map[string]string, *jsInjectedClientCallInfo) {
	info := jsInjectedClientCallForParams(params, analysis)
	if info == nil {
		return "", nil, nil
	}
	payload := jsMockPayloadForAnalysis(analysis)

	setup := fmt.Sprintf("%sconst %s = {\n", indent, info.Param) +
		fmt.Sprintf("%s  %sCalls: [],\n", indent, info.Method) +
		fmt.Sprintf("%s  %s: async (...args) => {\n", indent, info.Method) +
		fmt.Sprintf("%s    %s.%sCalls.push(args);\n", indent, info.Param, info.Method) +
		fmt.Sprintf("%s    return %s;\n", indent, payload) +
		fmt.Sprintf("%s  },\n", indent) +
		fmt.Sprintf("%s};\n", indent)

	return setup, map[string]string{info.Param: info.Param}, info
}

func jsInjectedClientCallForParams(params []jsParamInfo, analysis jsFuncAnalysis) *jsInjectedClientCallInfo {
	for _, p := range params {
		if !jsNameLooksLikeClientParam(jsCompactName(p.Name)) {
			continue
		}
		if info := jsInjectedClientCallForParam(&analysis, p.Name); info != nil {
			return info
		}
	}
	return nil
}

func jsInjectedClientCallForParam(analysis *jsFuncAnalysis, param string) *jsInjectedClientCallInfo {
	if analysis == nil || param == "" || !jsNameLooksLikeClientParam(jsCompactName(param)) {
		return nil
	}
	for _, expr := range analysis.Returns {
		if receiver, method, args, ok := jsInjectedClientCall(expr); ok && receiver == param {
			return &jsInjectedClientCallInfo{Param: param, Method: method, Args: args}
		}
	}
	for _, boundary := range analysis.Boundaries {
		if receiver, method, args, ok := jsInjectedClientCall(boundary.ReturnExpr); ok && receiver == param {
			return &jsInjectedClientCallInfo{Param: param, Method: method, Args: args}
		}
	}
	return nil
}

func genJSInjectedClientCallAssertion(info *jsInjectedClientCallInfo, indent string, style jsAssertionStyle) string {
	if info == nil {
		return ""
	}
	expectedArgs := "[]"
	if args := strings.TrimSpace(info.Args); args != "" {
		expectedArgs = "[" + args + "]"
	}
	expectedCalls := "[" + expectedArgs + "]"
	if style == jsAssertionStyleChai {
		return fmt.Sprintf("%sexpect(%s.%sCalls).to.deep.equal(%s);\n", indent, info.Param, info.Method, expectedCalls)
	}
	return fmt.Sprintf("%sexpect(%s.%sCalls).toEqual(%s);\n", indent, info.Param, info.Method, expectedCalls)
}

func jsInjectedClientMock(method string) string {
	return jsInjectedClientMockWithPayload(method, "{ ok: true }")
}

func jsInjectedClientMockWithPayload(method, payload string) string {
	switch method {
	case "fetch":
		return fmt.Sprintf("{ fetch: async () => (%s) }", payload)
	case "request":
		return fmt.Sprintf("{ request: async () => (%s) }", payload)
	default:
		return fmt.Sprintf("{ get: async () => (%s) }", payload)
	}
}

func jsResponseJSONMock(payload string) string {
	return fmt.Sprintf("{ json: async () => (%s) }", payload)
}

func jsMockPayloadForAnalysisPtr(analysis *jsFuncAnalysis) string {
	if analysis == nil {
		return "{ ok: true }"
	}
	return jsMockPayloadForAnalysis(*analysis)
}

func jsMockPayloadForAnalysis(analysis jsFuncAnalysis) string {
	if payload, ok := jsMockPayloadFromTSTypeWithDecls(analysis.ReturnTypeExpr, analysis.TSTypeDecls); ok {
		return payload
	}
	return "{ ok: true }"
}

func jsMockPayloadFromTSType(typeExpr string) (string, bool) {
	return jsMockPayloadFromTSTypeWithDecls(typeExpr, nil)
}

func jsMockPayloadFromTSTypeWithDecls(typeExpr string, decls map[string]string) (string, bool) {
	return jsMockPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, nil)
}

func jsMockPayloadFromTSTypeWithDeclsSeen(typeExpr string, decls map[string]string, seen map[string]bool) (string, bool) {
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	if typeExpr == "" {
		return "", false
	}
	typeExpr = jsUnwrapTSGeneric(typeExpr, "Promise")
	typeExpr = jsUnwrapTSGeneric(typeExpr, "PromiseLike")
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	if branch, ok := jsPreferredTSTypeUnionBranch(typeExpr); ok {
		typeExpr = branch
	}
	if name, resolved := jsResolveNamedTSType(typeExpr, decls); resolved != "" {
		if seen == nil {
			seen = map[string]bool{}
		}
		if seen[name] {
			return "{}", true
		}
		nextSeen := make(map[string]bool, len(seen)+1)
		for key, value := range seen {
			nextSeen[key] = value
		}
		nextSeen[name] = true
		typeExpr = strings.TrimSpace(resolved)
		seen = nextSeen
		if branch, ok := jsPreferredTSTypeUnionBranch(typeExpr); ok {
			typeExpr = branch
		}
	}
	if payload, ok := jsMockTuplePayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
	}
	if inner, ok := jsTSArrayElementType(typeExpr); ok {
		if payload, ok := jsMockPayloadFromTSTypeWithDeclsSeen(inner, decls, seen); ok {
			return "[" + payload + "]", true
		}
		return "[]", true
	}
	return jsObjectMockFromTSTypeWithDeclsSeen(typeExpr, decls, seen)
}

func jsObjectMockFromTSType(typeExpr string) (string, bool) {
	return jsObjectMockFromTSTypeWithDecls(typeExpr, nil)
}

func jsObjectMockFromTSTypeWithDecls(typeExpr string, decls map[string]string) (string, bool) {
	return jsObjectMockFromTSTypeWithDeclsSeen(typeExpr, decls, nil)
}

func jsObjectMockFromTSTypeWithDeclsSeen(typeExpr string, decls map[string]string, seen map[string]bool) (string, bool) {
	typeExpr = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(typeExpr), ";"))
	lookupType := jsNormalizeTSTypeExpr(typeExpr)
	if name, resolved := jsResolveNamedTSType(lookupType, decls); resolved != "" {
		if seen == nil {
			seen = map[string]bool{}
		}
		if seen[name] {
			return "{}", true
		}
		nextSeen := make(map[string]bool, len(seen)+1)
		for key, value := range seen {
			nextSeen[key] = value
		}
		nextSeen[name] = true
		typeExpr = strings.TrimSpace(resolved)
		seen = nextSeen
	}
	if !strings.HasPrefix(typeExpr, "{") || !strings.HasSuffix(typeExpr, "}") {
		return "", false
	}
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(typeExpr, "{"), "}"))
	fields := jsSplitTopLevelTypeFields(body)
	if len(fields) == 0 {
		return "", false
	}

	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		name, typ, ok := jsParseTSTypeField(field)
		if !ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s", name, jsMockValueForTSTypeWithDeclsSeen(name, typ, decls, seen)))
	}
	if len(parts) == 0 {
		return "", false
	}
	return "{ " + strings.Join(parts, ", ") + " }", true
}

func jsSplitTopLevelTypeFields(body string) []string {
	var fields []string
	start := 0
	angleDepth, braceDepth, bracketDepth, parenDepth := 0, 0, 0, 0
	for i, ch := range body {
		switch ch {
		case '<':
			angleDepth++
		case '>':
			if angleDepth > 0 {
				angleDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case ';', ',', '\n', '\r':
			if angleDepth == 0 && braceDepth == 0 && bracketDepth == 0 && parenDepth == 0 {
				if field := strings.TrimSpace(body[start:i]); field != "" {
					fields = append(fields, field)
				}
				start = i + 1
			}
		}
	}
	if field := strings.TrimSpace(body[start:]); field != "" {
		fields = append(fields, field)
	}
	return fields
}

func jsParseTSTypeField(field string) (string, string, bool) {
	field = strings.TrimSpace(field)
	if field == "" || strings.Contains(field, "(") {
		return "", "", false
	}
	parts := strings.SplitN(field, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	name := strings.TrimSpace(parts[0])
	name = strings.TrimSuffix(name, "?")
	name = strings.Trim(name, `"'`)
	if !regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`).MatchString(name) {
		return "", "", false
	}
	return name, strings.TrimSpace(parts[1]), true
}

func jsMockValueForTSType(fieldName, typeExpr string) string {
	return jsMockValueForTSTypeWithDecls(fieldName, typeExpr, nil)
}

func jsMockValueForTSTypeWithDecls(fieldName, typeExpr string, decls map[string]string) string {
	return jsMockValueForTSTypeWithDeclsSeen(fieldName, typeExpr, decls, nil)
}

func jsMockValueForTSTypeWithDeclsSeen(fieldName, typeExpr string, decls map[string]string, seen map[string]bool) string {
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	if branch, ok := jsPreferredTSTypeUnionBranch(typeExpr); ok {
		typeExpr = branch
	}
	if value, ok := jsMockValueForTSLiteral(typeExpr); ok {
		return value
	}
	compactName := jsCompactName(fieldName)
	switch {
	case typeExpr == "number" || typeExpr == "bigint":
		return "1"
	case typeExpr == "string":
		if value, ok := jsMockStringValueForFieldName(compactName); ok {
			return value
		}
		return "'test'"
	case typeExpr == "boolean":
		return "true"
	case typeExpr == "null":
		return "null"
	case typeExpr == "undefined" || typeExpr == "void":
		return "undefined"
	case jsTSTypeIsTuple(typeExpr):
		if payload, ok := jsMockTuplePayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
			return payload
		}
		return "[]"
	case jsTSTypeIsArray(typeExpr):
		return "[]"
	}
	if object, ok := jsObjectMockFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return object
	}
	return "{}"
}

func jsPreferredTSTypeUnionBranch(typeExpr string) (string, bool) {
	parts := jsSplitTopLevelTypeUnion(typeExpr)
	if len(parts) <= 1 {
		return "", false
	}
	for _, part := range parts {
		part = jsNormalizeTSTypeExpr(part)
		if part != "" && part != "null" && part != "undefined" {
			return part, true
		}
	}
	return jsNormalizeTSTypeExpr(parts[0]), true
}

func jsSplitTopLevelTypeUnion(typeExpr string) []string {
	var parts []string
	start := 0
	angleDepth, braceDepth, bracketDepth, parenDepth := 0, 0, 0, 0
	for i, ch := range typeExpr {
		switch ch {
		case '<':
			angleDepth++
		case '>':
			if angleDepth > 0 {
				angleDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '|':
			if angleDepth == 0 && braceDepth == 0 && bracketDepth == 0 && parenDepth == 0 {
				if part := strings.TrimSpace(typeExpr[start:i]); part != "" {
					parts = append(parts, part)
				}
				start = i + 1
			}
		}
	}
	if part := strings.TrimSpace(typeExpr[start:]); part != "" {
		parts = append(parts, part)
	}
	return parts
}

func jsMockStringValueForFieldName(name string) (string, bool) {
	switch {
	case jsNameHasAny(name, "email"):
		return "'user@example.com'", true
	case name == "id" || strings.HasSuffix(name, "id"):
		return "'id-1'", true
	case jsNameHasAny(name, "url", "uri", "endpoint", "href"):
		return "'https://example.com'", true
	case jsNameHasAny(name, "createdat", "updatedat", "deletedat", "timestamp", "datetime"):
		return "'2026-01-01T00:00:00.000Z'", true
	case name == "date" || strings.HasSuffix(name, "date"):
		return "'2026-01-01'", true
	case jsNameHasAny(name, "status", "state"):
		return "'active'", true
	case jsNameHasAny(name, "name", "title", "label"):
		return "'test'", true
	default:
		return "", false
	}
}

func jsMockTuplePayloadFromTSTypeWithDeclsSeen(typeExpr string, decls map[string]string, seen map[string]bool) (string, bool) {
	elements, ok := jsTSTupleElementTypes(typeExpr)
	if !ok {
		return "", false
	}
	values := make([]string, 0, len(elements))
	for _, elem := range elements {
		elem = jsNormalizeTSTupleElementType(elem)
		if elem == "" {
			continue
		}
		if payload, ok := jsMockPayloadFromTSTypeWithDeclsSeen(elem, decls, seen); ok {
			values = append(values, payload)
			continue
		}
		values = append(values, jsMockValueForTSTypeWithDeclsSeen("", elem, decls, seen))
	}
	if len(values) == 0 {
		return "[]", true
	}
	return "[" + strings.Join(values, ", ") + "]", true
}

func jsNormalizeTSTupleElementType(elem string) string {
	elem = strings.TrimSpace(elem)
	isRest := strings.HasPrefix(elem, "...")
	elem = strings.TrimSpace(strings.TrimPrefix(elem, "..."))
	if elem == "" || strings.HasPrefix(elem, "{") || strings.HasPrefix(elem, "[") {
		return elem
	}
	if _, typ, ok := jsParseTSTypeField(elem); ok {
		elem = typ
	}
	if isRest {
		if inner, ok := jsTSArrayElementType(elem); ok {
			return inner
		}
	}
	return elem
}

func jsMockValueForTSLiteral(typeExpr string) (string, bool) {
	value := strings.TrimSpace(typeExpr)
	if (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) ||
		(strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) {
		return "'" + strings.Trim(value, `'"`) + "'", true
	}
	if value == "true" || value == "false" || value == "null" || value == "undefined" || isNumericLiteral(value) {
		return value, true
	}
	return "", false
}

func jsResolveNamedTSType(typeExpr string, decls map[string]string) (string, string) {
	if len(decls) == 0 {
		return "", ""
	}
	if !regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`).MatchString(typeExpr) {
		return "", ""
	}
	return typeExpr, strings.TrimSpace(decls[typeExpr])
}

func jsNormalizeTSTypeExpr(typeExpr string) string {
	typeExpr = strings.TrimSpace(typeExpr)
	typeExpr = strings.TrimSuffix(typeExpr, ";")
	typeExpr = strings.Join(strings.Fields(typeExpr), " ")
	typeExpr = strings.ReplaceAll(typeExpr, " ;", ";")
	typeExpr = strings.ReplaceAll(typeExpr, " ,", ",")
	typeExpr = strings.ReplaceAll(typeExpr, "{ ", "{ ")
	return typeExpr
}

func jsUnwrapTSGeneric(typeExpr, name string) string {
	if inner, ok := jsTSGenericArg(typeExpr, name); ok {
		return inner
	}
	return typeExpr
}

func jsTSGenericArg(typeExpr, name string) (string, bool) {
	prefix := name + "<"
	if !strings.HasPrefix(typeExpr, prefix) || !strings.HasSuffix(typeExpr, ">") {
		return "", false
	}
	return strings.TrimSpace(typeExpr[len(prefix) : len(typeExpr)-1]), true
}

func jsTSArrayElementType(typeExpr string) (string, bool) {
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	if strings.HasPrefix(typeExpr, "readonly ") {
		typeExpr = strings.TrimSpace(strings.TrimPrefix(typeExpr, "readonly "))
	}
	if strings.HasSuffix(typeExpr, "[]") {
		return strings.TrimSpace(strings.TrimSuffix(typeExpr, "[]")), true
	}
	if inner, ok := jsTSGenericArg(typeExpr, "Array"); ok {
		return inner, true
	}
	if inner, ok := jsTSGenericArg(typeExpr, "ReadonlyArray"); ok {
		return inner, true
	}
	return "", false
}

func jsTSTypeIsArray(typeExpr string) bool {
	_, ok := jsTSArrayElementType(typeExpr)
	return ok
}

func jsTSTupleElementTypes(typeExpr string) ([]string, bool) {
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	if strings.HasPrefix(typeExpr, "readonly ") {
		typeExpr = strings.TrimSpace(strings.TrimPrefix(typeExpr, "readonly "))
	}
	if !strings.HasPrefix(typeExpr, "[") || !strings.HasSuffix(typeExpr, "]") {
		return nil, false
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(typeExpr, "["), "]"))
	if inner == "" {
		return nil, true
	}
	return jsSplitTopLevelTypeFields(inner), true
}

func jsTSTypeIsTuple(typeExpr string) bool {
	_, ok := jsTSTupleElementTypes(typeExpr)
	return ok
}

func jsNameLooksLikeResponseParam(name string) bool {
	switch name {
	case "response", "res", "resp":
		return true
	default:
		return false
	}
}

func jsNameLooksLikeClientParam(name string) bool {
	switch name {
	case "client", "api", "http", "fetcher", "requester":
		return true
	default:
		return false
	}
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

func jsSourceModuleImportPath(srcPath string) string {
	moduleName := stripExt(baseName(srcPath))
	importPath := "./" + moduleName
	ext := strings.ToLower(filepath.Ext(srcPath))
	if (ext == ".ts" || ext == ".tsx") && jsUsesNodeNextResolution(srcPath) {
		return importPath + ".js"
	}
	return importPath
}

func jsUsesNodeNextResolution(srcPath string) bool {
	tsconfig := findNearestFile(filepath.Dir(srcPath), "tsconfig.json")
	if tsconfig == "" {
		return false
	}
	data, err := os.ReadFile(tsconfig)
	if err != nil {
		return false
	}
	moduleResolution, module := parseTSConfigModuleSettings(data)
	return isNodeNextTSSetting(moduleResolution) || isNodeNextTSSetting(module)
}

type tsConfigFile struct {
	CompilerOptions struct {
		ModuleResolution string `json:"moduleResolution"`
		Module           string `json:"module"`
	} `json:"compilerOptions"`
}

func parseTSConfigModuleSettings(data []byte) (moduleResolution, module string) {
	var cfg tsConfigFile
	if err := json.Unmarshal(data, &cfg); err == nil {
		return cfg.CompilerOptions.ModuleResolution, cfg.CompilerOptions.Module
	}
	return jsonStringField(data, "moduleResolution"), jsonStringField(data, "module")
}

func jsonStringField(data []byte, field string) string {
	pattern := fmt.Sprintf(`(?i)"%s"\s*:\s*"([^"]+)"`, regexp.QuoteMeta(field))
	re := regexp.MustCompile(pattern)
	matches := re.FindSubmatch(data)
	if len(matches) != 2 {
		return ""
	}
	return string(matches[1])
}

func isNodeNextTSSetting(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "node16", "nodenext":
		return true
	default:
		return false
	}
}

func findNearestFile(dir, name string) string {
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
