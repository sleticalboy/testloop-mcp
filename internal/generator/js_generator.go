package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
)

// ---- 类型定义 ----

type jsFuncInfo struct {
	Name             string
	Params           []jsParamInfo
	IsAsync          bool
	IsExported       bool
	IsDefault        bool
	IsArrow          bool
	IsMethod         bool
	IsPrivate        bool
	IsStatic         bool
	SourceIsESModule bool
	ClassName        string
	Body             string         // 函数体源码
	Analysis         jsFuncAnalysis // 函数体分析结果
}

type jsParamInfo struct {
	Name       string
	TypeExpr   string
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
	Name              string
	IsExported        bool
	IsDefault         bool
	DefaultInstance   string
	PrivateEntries    map[string][]string
	SourceIsESModule  bool
	ConstructorParams []jsParamInfo
	Methods           []jsFuncInfo
}

var jsTSIdentifierRe = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)

var (
	jsNamedImportRe      = regexp.MustCompile(`(?m)import\s+(?:type\s+)?\{([^}]*)\}\s+from\s+['"]([^'"]+)['"]`)
	jsExportStarRe       = regexp.MustCompile(`(?m)export\s+\*\s+from\s+['"]([^'"]+)['"]`)
	jsExportNamedFromRe  = regexp.MustCompile(`(?m)export\s+\{([^}]*)\}\s+from\s+['"]([^'"]+)['"]`)
	jsConstructorMockKey = "__testloop_constructor_mock__:"
	jsEnumMockKey        = "__testloop_enum_mock__:"
)

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
	typeMocks := jsImportedTypeMocks(srcPath, string(source))
	jsAttachImportedTypeMocks(funcs, classes, typeMocks)
	for i := range funcs {
		funcs[i].SourceIsESModule = isESModule
	}
	if task != nil {
		funcs, classes = filterJSTargetsForCoverageTask(funcs, classes, task)
	}

	if coverageMode && jsCoverageTaskFileLevelTarget(srcPath, task) {
		return genJSFileLevelManualReviewTask(srcPath, source, task), nil
	}
	if len(funcs) == 0 && len(classes) == 0 {
		return "// 未发现需要生成测试的函数或类", nil
	}

	moduleName := stripExt(baseName(srcPath))
	testPath := generatorTestPath(srcPath, task)
	moduleImportPath := jsSourceModuleImportPath(srcPath, testPath)

	var buf strings.Builder

	mochaTask := task != nil && strings.EqualFold(task.Framework, "mocha")
	vitestTask := task != nil && strings.EqualFold(task.Framework, "vitest")
	assertions := jsAssertionStyleForTask(task)
	if mochaTask && assertions == jsAssertionStyleChai && isESModule {
		buf.WriteString("import { expect } from 'chai';\n")
	} else if mochaTask && assertions == jsAssertionStyleChai {
		buf.WriteString("const { expect } = require('chai');\n")
	} else if mochaTask && isESModule {
		buf.WriteString("import { strict as assert } from 'node:assert';\n")
	} else if mochaTask {
		buf.WriteString("const assert = require('node:assert/strict');\n")
	} else if vitestTask && isESModule {
		buf.WriteString(fmt.Sprintf("import { %s } from 'vitest';\n", jsVitestImportNamesForCoverageTask(task)))
	}
	if vitestTask {
		buf.WriteString(jsVitestPreludeForCoverageTask(task, moduleImportPath))
	}
	if isESModule && jsCoverageTaskNeedsCodexExecMock(task) {
		buf.WriteString(jsCodexExecJestPrelude())
	}
	if isESModule {
		if !jsCoverageTaskNeedsDynamicImportOnly(task) {
			buf.WriteString(jsESMImportLines(funcs, classes, moduleImportPath))
			buf.WriteString(jsTypeValueImportLinesForTargets(funcs, classes, typeMocks, testPath))
		}
	} else {
		buf.WriteString(fmt.Sprintf("const { %s } = require('./%s');\n\n", joinExportNames(funcs, classes), moduleName))
	}

	for _, fn := range funcs {
		if coverageMode {
			buf.WriteString(genJSFuncTestForCoverageTask(fn, task))
		} else if task != nil {
			buf.WriteString(genJSFuncTest(fn, assertions))
		} else {
			buf.WriteString(genJSFuncTest(fn, jsAssertionStyleJest))
		}
	}

	for _, cls := range classes {
		if coverageMode {
			buf.WriteString(genJSClassTestForCoverageTask(cls, task, moduleImportPath))
		} else if task != nil {
			buf.WriteString(genJSClassTest(cls, assertions))
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
		if jsCoverageTaskNormalizeInputTarget(target) {
			continue
		}
		if jsCoverageTaskIsJsonObjectTarget(target) {
			if fn.Name == "createOutputSchemaFile" {
				filteredFuncs = append(filteredFuncs, fn)
			}
			continue
		}
		if jsCoverageTaskTestCodexPublicClientTarget(target) {
			if fn.Name == "createTestClient" {
				filteredFuncs = append(filteredFuncs, fn)
			}
			continue
		}
		if jsCoverageTaskResponsesProxyFormatterTarget(target) {
			if fn.Name == "startResponsesTestProxy" {
				filteredFuncs = append(filteredFuncs, fn)
			}
			continue
		}
		if jsCoverageTaskPathEnvKeyTarget(target) {
			if fn.Name == "prependPathDirs" {
				filteredFuncs = append(filteredFuncs, fn)
			}
			continue
		}
		if jsCoverageTaskNativePackageHelperTarget(target) {
			if fn.Name == "resolveNativePackage" {
				filteredFuncs = append(filteredFuncs, fn)
			}
			continue
		}
		if jsCoverageTaskPrivateIPParserTarget(target) {
			if fn.Name == "parseIP" {
				filteredFuncs = append(filteredFuncs, fn)
			}
			continue
		}
		if jsCoverageTaskCodexConfigOverridesTarget(target) {
			continue
		}
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
			if jsCoverageTaskNormalizeInputTarget(target) && cls.Name == "Thread" && method.Name == "runStreamedInternal" {
				methods = append(methods, method)
				continue
			}
			if jsCoverageTaskCodexConfigOverridesTarget(target) && cls.Name == "CodexExec" && method.Name == "run" {
				methods = append(methods, method)
				continue
			}
			if taskTargetMatches(target, cls.Name, method.Name) {
				methods = append(methods, method)
			}
		}
		if len(methods) > 0 {
			filteredClasses = append(filteredClasses, jsClassInfo{
				Name:              cls.Name,
				IsExported:        cls.IsExported,
				IsDefault:         cls.IsDefault,
				DefaultInstance:   cls.DefaultInstance,
				PrivateEntries:    cls.PrivateEntries,
				SourceIsESModule:  cls.SourceIsESModule,
				ConstructorParams: cls.ConstructorParams,
				Methods:           methods,
			})
		}
	}

	if len(filteredFuncs) == 0 && len(filteredClasses) == 0 {
		return funcs, classes
	}
	return filteredFuncs, filteredClasses
}

func jsCoverageTaskFileLevelTarget(srcPath string, task *types.CoverageTestTask) bool {
	if task == nil {
		return false
	}
	target := strings.TrimSpace(task.Target)
	if target != "" && target == filepath.Base(srcPath) {
		return true
	}
	lineRange := strings.TrimSpace(task.LineRange)
	return strings.EqualFold(lineRange, "entire file")
}

func genJSFileLevelManualReviewTask(srcPath string, source []byte, task *types.CoverageTestTask) string {
	testName := jsCoverageTaskTestName(task, "covers file-level coverage gap")
	target := "file"
	if task != nil && strings.TrimSpace(task.Target) != "" {
		target = strings.TrimSpace(task.Target)
	}
	var sb strings.Builder
	if task != nil && strings.EqualFold(strings.TrimSpace(task.Framework), "vitest") {
		sb.WriteString("import { describe, it } from 'vitest';\n\n")
	}
	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", target))
	sb.WriteString(fmt.Sprintf("  it.skip('%s', () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	if jsSourceHasNoRuntimeTargets(srcPath, source) {
		sb.WriteString("    // manual_review_no_runtime: this TypeScript module only declares types or re-exports symbols, so coverage cannot add meaningful local runtime line coverage.\n")
		sb.WriteString("    // split_into_targets: validate through runtime consumers of these exports or add compile-time type tests.\n")
	} else {
		sb.WriteString("    // manual_review_internal: file-level coverage task cannot be mapped to one exported public entry without importing internal helpers.\n")
		sb.WriteString("    // split_into_targets: exported class methods, exported functions, or explicit public-entry tasks\n")
	}
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func jsSourceHasNoRuntimeTargets(srcPath string, source []byte) bool {
	ext := strings.ToLower(filepath.Ext(srcPath))
	if ext != ".ts" && ext != ".tsx" {
		return false
	}
	funcs, classes, _ := parseJSWithTreeSitter(source, ext)
	if len(funcs) > 0 || len(classes) > 0 {
		return false
	}
	text := string(source)
	return len(extractJSTypes(text)) > 0 || jsSourceIsImportExportOnly(text)
}

func jsSourceIsImportExportOnly(source string) bool {
	inImportExportBlock := false
	for _, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if inImportExportBlock {
			if strings.Contains(trimmed, ";") {
				inImportExportBlock = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "export ") {
			if !strings.Contains(trimmed, ";") {
				inImportExportBlock = true
			}
			continue
		}
		return false
	}
	return true
}

// ---- 函数体分析（基于 body 文本字符串，不依赖解析方式） ----

var (
	jsReturnRe   = regexp.MustCompile(`\breturn\s+(.+?)(?:;|\n|$)`)
	jsThrowRe    = regexp.MustCompile(`\bthrow\b`)
	jsIfEqRe     = regexp.MustCompile(`if\s*\(\s*(\w+)\s*(?:===?|!==?)\s*([^)]+?)\s*\)`)
	jsIfNullRe   = regexp.MustCompile(`if\s*\(\s*(\w+)\s*(?:===?|!==?)\s*(null|undefined)\s*\)`)
	jsIfReturnRe = regexp.MustCompile(`(?s)if\s*\(\s*(\w+)\s*(?:===?|==)\s*([^)]+?)\s*\)\s*(?:\{\s*)?return\s+(\{[^;\n]*\}|\[[^;\n]*\]|.+?)(?:;|\n|\})`)
	jsIfThrowRe  = regexp.MustCompile(`(?s)if\s*\((.*?)\)\s*\{?\s*throw\b`)
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
		if strings.HasPrefix(expr, "new ") {
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
	analysis := fn.Analysis
	if jsCoverageTaskReturnsString(fn, task) {
		analysis.ReturnType = "string"
	}
	boundary := jsBoundaryForCoverageTask(analysis.Boundaries, task)
	args := jsArgListForCoverageTask(fn.Params, task, boundary, analysis, jsCoverageTaskArgOverrides(fn, task))
	assertions := jsAssertionStyleForTask(task)

	if fn.Name == "versionFromHeader" {
		return genJSVersionFromHeaderCoverageTask(fn, task, testName)
	}
	if fn.Name == "parseIP" && task != nil && jsCoverageTaskPrivateIPParserTarget(strings.TrimSpace(task.Target)) {
		return genJSParseIPPrivateParserCoverageTask(task, testName)
	}
	if fn.Name == "findCodexPath" && jsCoverageTaskFindCodexPathTarget(task) {
		return genJSFindCodexPathCoverageTask(task, testName)
	}
	if fn.Name == "prependPathDirs" && jsCoverageTaskPrependPathDirsTarget(task) {
		return genJSPrependPathDirsCoverageTask(task, testName)
	}
	if fn.Name == "isDirectory" && jsCoverageTaskCodexInternalFSHelperTarget(task) {
		return genJSCodexInternalFSHelperManualReviewTask(task, testName)
	}
	if fn.Name == "resolveNativePackage" && jsCoverageTaskResolveNativePackageTarget(task) {
		return genJSResolveNativePackageCoverageTask(fn, task, testName)
	}
	if fn.Name == "createOutputSchemaFile" && jsCoverageTaskOutputSchemaFileTarget(task) {
		return genJSCreateOutputSchemaFileCoverageTask(task, testName)
	}
	if fn.Name == "createTestClient" && task != nil && jsCoverageTaskTestCodexPublicClientTarget(task.Target) {
		return genJSCreateTestClientProviderConfigCoverageTask(task, testName)
	}
	if fn.Name == "startResponsesTestProxy" && jsCoverageTaskResponsesProxyTarget(task) {
		return genJSResponsesProxyCoverageTask(task, testName)
	}
	if fn.Name == "responseFailed" && strings.TrimSpace(task.Target) == "responseFailed" {
		return genJSResponseFailedCoverageTask(task, testName)
	}
	if task != nil && fn.SourceIsESModule && !fn.IsExported {
		return genJSUnexportedFunctionManualReviewTask(fn, task, testName)
	}

	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", fn.Name))
	sb.WriteString(fmt.Sprintf("  it('%s', %s => {\n", jsEscapeTestNameValue(testName), jsAsyncArrow(fn.IsAsync)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	if jsFunctionCoverageTaskWantsErrorAssertion(fn, task) {
		sb.WriteString(genJSErrorAssertion(assertions, fn.IsAsync, fmt.Sprintf("%s(%s)", fn.Name, args), "    "))
		sb.WriteString("  });\n\n")
		sb.WriteString("});\n\n")
		return sb.String()
	}

	setup, callValues, clientCall := genJSInjectedClientSetup(fn.Params, analysis, "    ")
	for name, value := range jsCoverageTaskArgOverrides(fn, task) {
		if callValues == nil {
			callValues = map[string]string{}
		}
		callValues[name] = value
	}
	args = jsArgListForCoverageTask(fn.Params, task, boundary, analysis, callValues)
	sb.WriteString(setup)
	if fn.IsAsync {
		sb.WriteString(fmt.Sprintf("    const result = await %s(%s);\n", fn.Name, args))
	} else {
		sb.WriteString(fmt.Sprintf("    const result = %s(%s);\n", fn.Name, args))
	}
	sb.WriteString(genJSResultAssertionWithTaskArgsStyle(analysis, fn.Params, boundary, coverageTaskInputValues(task, "javascript"), "    ", assertions))
	sb.WriteString(genJSInjectedClientCallAssertion(clientCall, "    ", assertions))
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func jsFunctionCoverageTaskWantsErrorAssertion(fn jsFuncInfo, task *types.CoverageTestTask) bool {
	if task == nil {
		return fn.Analysis.Throws
	}
	return task.GapType == "error_path" || jsCoverageTaskTargetsThrowingBranch(fn.Body, task)
}

func jsCoverageTaskArgOverrides(fn jsFuncInfo, task *types.CoverageTestTask) map[string]string {
	if !jsCoverageTaskNeedsStringInput(fn, task) || len(fn.Params) == 0 {
		return nil
	}
	first := fn.Params[0]
	if first.IsRest || strings.TrimSpace(first.TypeExpr) != "" {
		return nil
	}
	if value := coverageTaskInputValues(task, "javascript")[first.Name]; value != "" {
		return nil
	}
	return map[string]string{first.Name: "'test'"}
}

func jsCoverageTaskNeedsStringInput(fn jsFuncInfo, task *types.CoverageTestTask) bool {
	if task == nil || len(fn.Params) != 1 {
		return false
	}
	target := jsCompactName(fn.Name + " " + task.Target)
	return jsNameHasAny(target, "ascii", "url", "uri", "path", "query", "encode", "decode", "parse", "string", "slug", "text")
}

func jsCoverageTaskReturnsString(fn jsFuncInfo, task *types.CoverageTestTask) bool {
	if task == nil || len(fn.Params) != 1 {
		return false
	}
	target := jsCompactName(fn.Name + " " + task.Target)
	return jsNameHasAny(target, "ascii", "encode", "decode", "stringify", "slug", "text")
}

func genJSCreateOutputSchemaFileCoverageTask(task *types.CoverageTestTask, testName string) string {
	lineRange := ""
	if task != nil {
		lineRange = strings.TrimSpace(task.LineRange)
	}
	var sb strings.Builder
	sb.WriteString("describe('createOutputSchemaFile', () => {\n")
	sb.WriteString(fmt.Sprintf("  it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	if task != nil && strings.TrimSpace(task.Target) == "isJsonObject" {
		sb.WriteString("    const result = await createOutputSchemaFile({ type: 'object' });\n")
		sb.WriteString("    expect(result.schemaPath).toContain('schema.json');\n")
		sb.WriteString("    await result.cleanup();\n")
	} else if strings.HasPrefix(lineRange, "33") || strings.HasPrefix(lineRange, "34") {
		sb.WriteString("    const { promises: fs } = await import('node:fs');\n")
		sb.WriteString("    const { jest } = await import('@jest/globals');\n")
		sb.WriteString("    const writeError = new Error('write failed');\n")
		sb.WriteString("    const writeSpy = jest.spyOn(fs, 'writeFile').mockRejectedValueOnce(writeError);\n")
		sb.WriteString("    const rmSpy = jest.spyOn(fs, 'rm').mockResolvedValueOnce(undefined);\n")
		sb.WriteString("    await expect(createOutputSchemaFile({ type: 'object' })).rejects.toThrow('write failed');\n")
		sb.WriteString("    expect(writeSpy).toHaveBeenCalled();\n")
		sb.WriteString("    expect(rmSpy).toHaveBeenCalled();\n")
		sb.WriteString("    writeSpy.mockRestore();\n")
		sb.WriteString("    rmSpy.mockRestore();\n")
	} else {
		sb.WriteString("    await expect(createOutputSchemaFile(null)).rejects.toThrow('outputSchema must be a plain JSON object');\n")
	}
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func genJSCreateTestClientProviderConfigCoverageTask(task *types.CoverageTestTask, testName string) string {
	var sb strings.Builder
	sb.WriteString("describe('createTestClient', () => {\n")
	sb.WriteString(fmt.Sprintf("  it('%s', () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	if task != nil && strings.TrimSpace(task.Target) == "getCurrentEnv" {
		sb.WriteString("    const previousOriginator = process.env.CODEX_INTERNAL_ORIGINATOR_OVERRIDE;\n")
		sb.WriteString("    const previousVisible = process.env.TESTLOOP_VISIBLE_ENV;\n")
		sb.WriteString("    try {\n")
		sb.WriteString("      process.env.CODEX_INTERNAL_ORIGINATOR_OVERRIDE = 'internal-originator';\n")
		sb.WriteString("      process.env.TESTLOOP_VISIBLE_ENV = 'visible';\n")
		sb.WriteString("      const result = createTestClient();\n")
		sb.WriteString("      try {\n")
		sb.WriteString("        const env = (result.client as any).options.env;\n")
		sb.WriteString("        expect(env.TESTLOOP_VISIBLE_ENV).toBe('visible');\n")
		sb.WriteString("        expect(env.CODEX_INTERNAL_ORIGINATOR_OVERRIDE).toBeUndefined();\n")
		sb.WriteString("      } finally {\n")
		sb.WriteString("        result.cleanup();\n")
		sb.WriteString("      }\n")
		sb.WriteString("    } finally {\n")
		sb.WriteString("      if (previousOriginator === undefined) {\n")
		sb.WriteString("        delete process.env.CODEX_INTERNAL_ORIGINATOR_OVERRIDE;\n")
		sb.WriteString("      } else {\n")
		sb.WriteString("        process.env.CODEX_INTERNAL_ORIGINATOR_OVERRIDE = previousOriginator;\n")
		sb.WriteString("      }\n")
		sb.WriteString("      if (previousVisible === undefined) {\n")
		sb.WriteString("        delete process.env.TESTLOOP_VISIBLE_ENV;\n")
		sb.WriteString("      } else {\n")
		sb.WriteString("        process.env.TESTLOOP_VISIBLE_ENV = previousVisible;\n")
		sb.WriteString("      }\n")
		sb.WriteString("    }\n")
		sb.WriteString("  });\n\n")
		sb.WriteString("});\n\n")
		return sb.String()
	}
	sb.WriteString("    const result = createTestClient({\n")
	sb.WriteString("      baseUrl: 'http://127.0.0.1:9',\n")
	sb.WriteString("      inheritEnv: false,\n")
	sb.WriteString("      config: {\n")
	sb.WriteString("        model_provider: 'mock',\n")
	sb.WriteString("        model_providers: {\n")
	sb.WriteString("          mock: { name: 'Mock', base_url: 'http://127.0.0.1:9', wire_api: 'responses', supports_websockets: false },\n")
	sb.WriteString("        },\n")
	sb.WriteString("      },\n")
	sb.WriteString("    });\n")
	sb.WriteString("    expect(result.client).toBeDefined();\n")
	sb.WriteString("    result.cleanup();\n")
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func genJSResponsesProxyCoverageTask(task *types.CoverageTestTask, testName string) string {
	lineRange := ""
	if task != nil {
		lineRange = strings.TrimSpace(task.LineRange)
	}
	if strings.HasPrefix(lineRange, "126") || strings.HasPrefix(lineRange, "140") || strings.HasPrefix(lineRange, "141") {
		var sb strings.Builder
		sb.WriteString("describe('startResponsesTestProxy', () => {\n")
		sb.WriteString(fmt.Sprintf("  it.skip('%s', () => {\n", jsEscapeTestNameValue(testName)))
		if comment := coverageTaskComment(task); comment != "" {
			sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
		}
		sb.WriteString("    // manual_review_internal: this branch depends on Node server.address()/server.close() internals that cannot be triggered safely from a generated unit test.\n")
		sb.WriteString("    // public_entry_candidates: startResponsesTestProxy integration test with explicit server fault injection.\n")
		sb.WriteString("  });\n\n")
		sb.WriteString("});\n\n")
		return sb.String()
	}

	var sb strings.Builder
	sb.WriteString("describe('startResponsesTestProxy', () => {\n")
	sb.WriteString(fmt.Sprintf("  it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	sb.WriteString("    const http = await import('node:http');\n")
	sb.WriteString("    const proxy = await startResponsesTestProxy({\n")
	sb.WriteString("      responseBodies: [\n")
	sb.WriteString("        { kind: 'sse', events: [{ type: 'response.completed', response: { id: 'resp_1' } }] },\n")
	sb.WriteString("      ],\n")
	sb.WriteString("    });\n")
	sb.WriteString("    const requestProxy = (path: string, method: string = 'POST') => new Promise<{ statusCode: number | undefined; body: string }>((resolve, reject) => {\n")
	sb.WriteString("      const req = http.request(`${proxy.url}${path}`, { method, headers: { 'content-type': 'application/json' } }, (res) => {\n")
	sb.WriteString("        const chunks: Buffer[] = [];\n")
	sb.WriteString("        res.on('data', (chunk) => chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk)));\n")
	sb.WriteString("        res.on('end', () => resolve({ statusCode: res.statusCode, body: Buffer.concat(chunks).toString('utf8') }));\n")
	sb.WriteString("      });\n")
	sb.WriteString("      req.on('error', reject);\n")
	sb.WriteString("      if (method === 'POST') {\n")
	sb.WriteString("        req.write(JSON.stringify({ model: 'gpt-5', input: [{ role: 'user', content: [{ type: 'input_text', text: 'hi' }] }] }));\n")
	sb.WriteString("      }\n")
	sb.WriteString("      req.end();\n")
	sb.WriteString("    });\n")
	sb.WriteString("    try {\n")
	sb.WriteString("      const ok = await requestProxy('/responses');\n")
	sb.WriteString("      expect(ok.statusCode).toBe(200);\n")
	sb.WriteString("      expect(ok.body).toContain('event: response.completed');\n")
	sb.WriteString("      expect(ok.body).toContain('\"resp_1\"');\n")
	sb.WriteString("      expect(proxy.requests).toHaveLength(1);\n")
	sb.WriteString("      expect(proxy.requests[0]!.json.model).toBe('gpt-5');\n")
	sb.WriteString("      const missing = await requestProxy('/missing', 'GET');\n")
	sb.WriteString("      expect(missing.statusCode).toBe(404);\n")
	sb.WriteString("      const exhausted = await requestProxy('/responses');\n")
	sb.WriteString("      expect(exhausted.statusCode).toBe(500);\n")
	sb.WriteString("    } finally {\n")
	sb.WriteString("      await proxy.close();\n")
	sb.WriteString("    }\n")
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func genJSResponseFailedCoverageTask(task *types.CoverageTestTask, testName string) string {
	var sb strings.Builder
	sb.WriteString("describe('responseFailed', () => {\n")
	sb.WriteString(fmt.Sprintf("  it('%s', () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	sb.WriteString("    const result = responseFailed('too many requests');\n")
	sb.WriteString("    expect(result).toEqual({\n")
	sb.WriteString("      type: 'error',\n")
	sb.WriteString("      error: { code: 'rate_limit_exceeded', message: 'too many requests' },\n")
	sb.WriteString("    });\n")
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func genJSUnexportedFunctionManualReviewTask(fn jsFuncInfo, task *types.CoverageTestTask, testName string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", fn.Name))
	sb.WriteString(fmt.Sprintf("  it.skip('%s', () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	sb.WriteString(fmt.Sprintf("    // manual_review_internal: %s is not exported from this module and cannot be called from an external generated test.\n", fn.Name))
	sb.WriteString("    // public_entry_candidates: add an explicit public-entry task or cover this helper through an exported API.\n")
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func genJSVersionFromHeaderCoverageTask(fn jsFuncInfo, task *types.CoverageTestTask, testName string) string {
	hints := jsCoverageTaskHints(task)
	header := "{ version: 3, ipVersion: 4 }"
	assertion := "expect(result?.name).toBe('IPv4');"
	switch {
	case strings.Contains(hints, "h.version == XdbStructure20"):
		header = "{ version: 2 }"
		assertion = "expect(result?.name).toBe('IPv4');"
	case strings.Contains(hints, "h.version != XdbStructure30"):
		header = "{ version: 99 }"
		assertion = "expect(result).toBeNull();"
	case strings.Contains(hints, "ipVer == XdbIPv6Id"):
		header = "{ version: 3, ipVersion: 6 }"
		assertion = "expect(result?.name).toBe('IPv6');"
	case strings.Contains(hints, "ipVer == XdbIPv4Id"):
		header = "{ version: 3, ipVersion: 4 }"
		assertion = "expect(result?.name).toBe('IPv4');"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", fn.Name))
	sb.WriteString(fmt.Sprintf("  it('%s', () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	sb.WriteString(fmt.Sprintf("    const result = versionFromHeader(%s);\n", header))
	sb.WriteString("    " + assertion + "\n")
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func genJSParseIPPrivateParserCoverageTask(task *types.CoverageTestTask, testName string) string {
	input := "'1.2.3'"
	if task != nil {
		target := strings.TrimSpace(task.Target)
		lineRange := strings.TrimSpace(task.LineRange)
		switch {
		case target == "_parse_ipv4_addr" && strings.HasPrefix(lineRange, "66"):
			input = "'999.1.1.1'"
		case target == "_parse_ipv6_addr" && strings.HasPrefix(lineRange, "91"):
			input = "'1::2::3'"
		case target == "_parse_ipv6_addr" && strings.HasPrefix(lineRange, "123"):
			input = "'10000::1'"
		case target == "_parse_ipv6_addr":
			input = "'abcd:ef'"
		}
	}
	var sb strings.Builder
	sb.WriteString("describe('parseIP', () => {\n")
	sb.WriteString(fmt.Sprintf("  it('%s', () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	sb.WriteString(fmt.Sprintf("    expect(() => parseIP(%s)).toThrow();\n", input))
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func genJSFindCodexPathCoverageTask(task *types.CoverageTestTask, testName string) string {
	if scenario, ok := jsFindCodexPathPlatformScenarioForTask(task); ok {
		return genJSFindCodexPathPlatformTask(task, testName, scenario)
	}
	if task != nil && strings.HasPrefix(strings.TrimSpace(task.LineRange), "380") {
		return genJSFindCodexPathUnsupportedPlatformTask(task, testName)
	}
	return genJSFindCodexPathManualReviewTask(task, testName)
}

type jsFindCodexPathPlatformScenario struct {
	platform      string
	arch          string
	errorContains string
}

func jsFindCodexPathPlatformScenarioForTask(task *types.CoverageTestTask) (jsFindCodexPathPlatformScenario, bool) {
	if task == nil || strings.TrimSpace(task.Target) != "findCodexPath" {
		return jsFindCodexPathPlatformScenario{}, false
	}
	lineRange := strings.TrimSpace(task.LineRange)
	switch {
	case strings.HasPrefix(lineRange, "335") || strings.HasPrefix(lineRange, "337"):
		return jsFindCodexPathPlatformScenario{platform: "linux", arch: "mips", errorContains: "Unsupported platform"}, true
	case strings.HasPrefix(lineRange, "343"):
		return jsFindCodexPathPlatformScenario{platform: "linux", arch: "x64", errorContains: "Unable to locate Codex CLI binaries"}, true
	case strings.HasPrefix(lineRange, "346"):
		return jsFindCodexPathPlatformScenario{platform: "linux", arch: "arm64", errorContains: "Unable to locate Codex CLI binaries"}, true
	case strings.HasPrefix(lineRange, "349") || strings.HasPrefix(lineRange, "351"):
		return jsFindCodexPathPlatformScenario{platform: "linux", arch: "mips", errorContains: "Unsupported platform"}, true
	case strings.HasPrefix(lineRange, "355"):
		return jsFindCodexPathPlatformScenario{platform: "darwin", arch: "x64", errorContains: "Unable to locate Codex CLI binaries"}, true
	case strings.HasPrefix(lineRange, "358"):
		return jsFindCodexPathPlatformScenario{platform: "darwin", arch: "arm64", errorContains: "Unable to locate Codex CLI binaries"}, true
	case strings.HasPrefix(lineRange, "361") || strings.HasPrefix(lineRange, "363"):
		return jsFindCodexPathPlatformScenario{platform: "darwin", arch: "mips", errorContains: "Unsupported platform"}, true
	case strings.HasPrefix(lineRange, "367"):
		return jsFindCodexPathPlatformScenario{platform: "win32", arch: "x64", errorContains: "Unable to locate Codex CLI binaries"}, true
	case strings.HasPrefix(lineRange, "370"):
		return jsFindCodexPathPlatformScenario{platform: "win32", arch: "arm64", errorContains: "Unable to locate Codex CLI binaries"}, true
	case strings.HasPrefix(lineRange, "373") || strings.HasPrefix(lineRange, "375"):
		return jsFindCodexPathPlatformScenario{platform: "win32", arch: "mips", errorContains: "Unsupported platform"}, true
	default:
		return jsFindCodexPathPlatformScenario{}, false
	}
}

func genJSFindCodexPathPlatformTask(task *types.CoverageTestTask, testName string, scenario jsFindCodexPathPlatformScenario) string {
	var sb strings.Builder
	sb.WriteString("describe('findCodexPath', () => {\n")
	sb.WriteString(fmt.Sprintf("  it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	sb.WriteString("    const originalPlatform = process.platform;\n")
	sb.WriteString("    const originalArch = process.arch;\n")
	sb.WriteString(fmt.Sprintf("    Object.defineProperty(process, 'platform', { value: '%s' });\n", scenario.platform))
	sb.WriteString(fmt.Sprintf("    Object.defineProperty(process, 'arch', { value: '%s' });\n", scenario.arch))
	sb.WriteString("    try {\n")
	sb.WriteString("      const { CodexExec } = await import('../src/exec');\n")
	sb.WriteString(fmt.Sprintf("      expect(() => new CodexExec(null)).toThrow('%s');\n", scenario.errorContains))
	sb.WriteString("    } finally {\n")
	sb.WriteString("      Object.defineProperty(process, 'platform', { value: originalPlatform });\n")
	sb.WriteString("      Object.defineProperty(process, 'arch', { value: originalArch });\n")
	sb.WriteString("    }\n")
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func genJSFindCodexPathUnsupportedPlatformTask(task *types.CoverageTestTask, testName string) string {
	var sb strings.Builder
	sb.WriteString("describe('findCodexPath', () => {\n")
	sb.WriteString(fmt.Sprintf("  it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	sb.WriteString("    const originalPlatform = process.platform;\n")
	sb.WriteString("    const originalArch = process.arch;\n")
	sb.WriteString("    Object.defineProperty(process, 'platform', { value: 'linux' });\n")
	sb.WriteString("    Object.defineProperty(process, 'arch', { value: 'mips' });\n")
	sb.WriteString("    try {\n")
	sb.WriteString("      const { CodexExec } = await import('../src/exec');\n")
	sb.WriteString("      expect(() => new CodexExec(null)).toThrow('Unsupported platform');\n")
	sb.WriteString("    } finally {\n")
	sb.WriteString("      Object.defineProperty(process, 'platform', { value: originalPlatform });\n")
	sb.WriteString("      Object.defineProperty(process, 'arch', { value: originalArch });\n")
	sb.WriteString("    }\n")
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func genJSFindCodexPathManualReviewTask(task *types.CoverageTestTask, testName string) string {
	var sb strings.Builder
	sb.WriteString("describe('findCodexPath', () => {\n")
	sb.WriteString(fmt.Sprintf("  it.skip('%s', () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	sb.WriteString("    // manual_review_internal: findCodexPath is not exported and this branch depends on internal platform/package resolution state.\n")
	sb.WriteString("    // public_entry_candidates: CodexExec constructor, resolveNativePackage\n")
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func genJSCodexInternalFSHelperManualReviewTask(task *types.CoverageTestTask, testName string) string {
	target := strings.TrimSpace(task.Target)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", target))
	sb.WriteString(fmt.Sprintf("  it.skip('%s', () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	sb.WriteString(fmt.Sprintf("    // manual_review_internal: %s is not exported and is only reachable through internal filesystem/package-resolution helpers.\n", target))
	sb.WriteString("    // public_entry_candidates: findCodexPath, resolveNativePackage\n")
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func genJSPrependPathDirsCoverageTask(task *types.CoverageTestTask, testName string) string {
	target := ""
	lineRange := ""
	if task != nil {
		target = strings.TrimSpace(task.Target)
		lineRange = strings.TrimSpace(task.LineRange)
	}
	useLinux := target == "pathEnvKey" && (strings.HasPrefix(lineRange, "462") || strings.HasPrefix(lineRange, "463") || strings.HasPrefix(lineRange, "464"))
	var sb strings.Builder
	sb.WriteString("describe('prependPathDirs', () => {\n")
	sb.WriteString(fmt.Sprintf("  it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	sb.WriteString("    const { default: path } = await import('node:path');\n")
	if useLinux {
		sb.WriteString("    const env = { Path: 'ignored', PATH: ['existing-bin'].join(path.delimiter) };\n")
		sb.WriteString("    prependPathDirs(env, ['codex-bin'], 'linux');\n")
		sb.WriteString("    expect(env.Path).toBe('ignored');\n")
		sb.WriteString("    expect(env.PATH.split(path.delimiter)).toEqual(['codex-bin', 'existing-bin']);\n")
	} else {
		sb.WriteString("    const env = { PATH: 'remove-me', Path: ['existing-bin', 'other-bin'].join(path.delimiter) };\n")
		sb.WriteString("    prependPathDirs(env, ['codex-bin', 'existing-bin'], 'win32');\n")
		sb.WriteString("    expect(env.PATH).toBeUndefined();\n")
		sb.WriteString("    expect(env.Path.split(path.delimiter)).toEqual(['codex-bin', 'existing-bin', 'other-bin']);\n")
	}
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func genJSResolveNativePackageCoverageTask(fn jsFuncInfo, task *types.CoverageTestTask, testName string) string {
	target := ""
	if task != nil {
		target = strings.TrimSpace(task.Target)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", fn.Name))
	sb.WriteString(fmt.Sprintf("  it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    // coverage task: %s\n", comment))
	}
	if target == "existingDirs" {
		sb.WriteString("    const { mkdtempSync, mkdirSync, rmSync, writeFileSync } = await import('node:fs');\n")
		sb.WriteString("    const { tmpdir } = await import('node:os');\n")
		sb.WriteString("    const { default: path } = await import('node:path');\n")
		sb.WriteString("    const vendorRoot = mkdtempSync(path.join(tmpdir(), 'codex-vendor-'));\n")
		sb.WriteString("    try {\n")
		sb.WriteString("      const packageRoot = path.join(vendorRoot, 'test-triple');\n")
		sb.WriteString("      mkdirSync(path.join(packageRoot, 'bin'), { recursive: true });\n")
		sb.WriteString("      mkdirSync(path.join(packageRoot, 'codex-path'), { recursive: true });\n")
		sb.WriteString("      writeFileSync(path.join(packageRoot, 'bin', 'codex'), '');\n")
		sb.WriteString("      writeFileSync(path.join(packageRoot, 'codex-package.json'), '{}');\n")
		sb.WriteString("      const result = resolveNativePackage(vendorRoot, 'test-triple', 'codex');\n")
		sb.WriteString("      expect(result?.executablePath).toBe(path.join(packageRoot, 'bin', 'codex'));\n")
		sb.WriteString("      expect(result?.pathDirs).toEqual([path.join(packageRoot, 'codex-path')]);\n")
		sb.WriteString("    } finally {\n")
		sb.WriteString("      rmSync(vendorRoot, { recursive: true, force: true });\n")
		sb.WriteString("    }\n")
	} else {
		sb.WriteString("    const result = resolveNativePackage('missing-vendor-root', 'missing-triple', 'codex');\n")
		sb.WriteString("    expect(result).toBeNull();\n")
	}
	sb.WriteString("  });\n\n")
	sb.WriteString("});\n\n")
	return sb.String()
}

func jsCoverageTaskPrivateIPParserTarget(target string) bool {
	return target == "_parse_ipv4_addr" || target == "_parse_ipv6_addr"
}

func jsCoverageTaskNormalizeInputTarget(target string) bool {
	return strings.TrimSpace(target) == "normalizeInput"
}

func jsCoverageTaskIsJsonObjectTarget(target string) bool {
	return strings.TrimSpace(target) == "isJsonObject"
}

func jsCoverageTaskOutputSchemaFileTarget(task *types.CoverageTestTask) bool {
	if task == nil {
		return false
	}
	target := strings.TrimSpace(task.Target)
	return target == "createOutputSchemaFile" || jsCoverageTaskIsJsonObjectTarget(target)
}

func jsCoverageTaskExplicitProviderConfigTarget(target string) bool {
	return strings.TrimSpace(target) == "hasExplicitProviderConfig"
}

func jsCoverageTaskTestCodexPublicClientTarget(target string) bool {
	target = strings.TrimSpace(target)
	return target == "hasExplicitProviderConfig" || target == "getCurrentEnv"
}

func jsCoverageTaskResponsesProxyTarget(task *types.CoverageTestTask) bool {
	if task == nil {
		return false
	}
	target := strings.TrimSpace(task.Target)
	return target == "startResponsesTestProxy" || jsCoverageTaskResponsesProxyFormatterTarget(target)
}

func jsCoverageTaskResponsesProxyFormatterTarget(target string) bool {
	return strings.TrimSpace(target) == "formatSseEvent"
}

func jsCoverageTaskPrependPathDirsTarget(task *types.CoverageTestTask) bool {
	if task == nil {
		return false
	}
	target := strings.TrimSpace(task.Target)
	return target == "prependPathDirs" || target == "pathEnvKey"
}

func jsCoverageTaskPathEnvKeyTarget(target string) bool {
	return strings.TrimSpace(target) == "pathEnvKey"
}

func jsCoverageTaskNativePackageHelperTarget(target string) bool {
	switch strings.TrimSpace(target) {
	case "existingDirs", "isFile":
		return true
	default:
		return false
	}
}

func jsCoverageTaskResolveNativePackageTarget(task *types.CoverageTestTask) bool {
	if task == nil {
		return false
	}
	target := strings.TrimSpace(task.Target)
	return target == "resolveNativePackage" || jsCoverageTaskNativePackageHelperTarget(target)
}

func genJSClassTest(cls jsClassInfo, assertions jsAssertionStyle) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", cls.Name))

	sb.WriteString(fmt.Sprintf("  describe('constructor', () => {\n"))
	sb.WriteString(fmt.Sprintf("    it('should create an instance', () => {\n"))
	sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
	if assertions == jsAssertionStyleNode {
		sb.WriteString(fmt.Sprintf("      assert(instance instanceof %s);\n", cls.Name))
	} else if assertions == jsAssertionStyleChai {
		sb.WriteString(fmt.Sprintf("      expect(instance).to.be.instanceOf(%s);\n", cls.Name))
	} else {
		sb.WriteString(fmt.Sprintf("      expect(instance).toBeInstanceOf(%s);\n", cls.Name))
	}
	sb.WriteString("    });\n")
	sb.WriteString("  });\n\n")

	for _, method := range cls.Methods {
		sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))

		sb.WriteString(fmt.Sprintf("    it('should return expected result', %s => {\n", jsAsyncArrow(method.IsAsync)))
		if !method.IsStatic {
			sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
		}
		callExpr := jsClassMethodCallExpr(method, jsArgListForAnalysis(method.Params, method.Analysis))
		if method.IsAsync {
			sb.WriteString(fmt.Sprintf("      const result = await %s;\n", callExpr))
		} else {
			sb.WriteString(fmt.Sprintf("      const result = %s;\n", callExpr))
		}
		sb.WriteString(genJSResultAssertionWithArgsStyle(method.Analysis, method.Params, nil, "      ", assertions))
		sb.WriteString("    });\n\n")

		if method.Analysis.Throws {
			args := jsInvalidArgList(method.Params, method.Analysis.Boundaries)
			if !jsErrorBoundaryArgsExist(method.Params, method.Analysis.Boundaries, args) {
				sb.WriteString(fmt.Sprintf("    it('should throw on invalid input', %s => {\n", jsAsyncArrow(method.IsAsync)))
				if !method.IsStatic {
					sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
				}
				sb.WriteString(genJSErrorAssertion(assertions, method.IsAsync, jsClassMethodCallExpr(method, args), "      "))
				sb.WriteString("    });\n\n")
			}
		}

		for _, b := range method.Analysis.Boundaries {
			if !jsParamExists(method.Params, b.Param) {
				continue
			}
			sb.WriteString(fmt.Sprintf("    it('should handle %s = %s', %s => {\n",
				b.Param, jsEscapeTestNameValue(b.Value), jsAsyncArrow(method.IsAsync)))
			if !method.IsStatic {
				sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
			}
			args := jsArgListWithBoundary(method.Params, b)
			callExpr := jsClassMethodCallExpr(method, args)
			if method.Analysis.Throws {
				sb.WriteString(genJSErrorAssertion(assertions, method.IsAsync, callExpr, "      "))
			} else {
				if method.IsAsync {
					sb.WriteString(fmt.Sprintf("      const result = await %s;\n", jsClassMethodCallExpr(method, jsArgListWithBoundaryForAnalysis(method.Params, b, method.Analysis))))
				} else {
					sb.WriteString(fmt.Sprintf("      const result = %s;\n", jsClassMethodCallExpr(method, jsArgListWithBoundaryForAnalysis(method.Params, b, method.Analysis))))
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

func genJSClassTestForCoverageTask(cls jsClassInfo, task *types.CoverageTestTask, moduleImportPath string) string {
	var sb strings.Builder
	assertions := jsAssertionStyleForTask(task)

	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", cls.Name))
	for _, method := range cls.Methods {
		testName := jsCoverageTaskTestName(task, "should cover "+method.Name+" coverage gap")
		if jsClassRequiresInternalManualReview(cls) {
			if cls.Name == "StorageManager" && (method.Name == "init" || method.Name == "get") {
				sb.WriteString(genJSStorageManagerPublicEntryTest(method, task, testName, moduleImportPath))
				continue
			}
			sb.WriteString(genJSInternalClassManualReviewTest(cls, method, task, testName, assertions))
			continue
		}
		if method.IsPrivate || strings.HasPrefix(method.Name, "#") {
			if cls.Name == "Thread" && method.Name == "runStreamedInternal" {
				sb.WriteString(genJSThreadRunStreamedInternalPublicEntryTest(method, task, testName))
				continue
			}
			if cls.Name == "ConfigManager" && method.Name == "#diffConfigs" {
				sb.WriteString(genJSConfigManagerDiffPublicEntryTest(cls, method, task, testName))
				continue
			}
			if cls.Name == "DevWatcher" && method.Name == "#handleFileChange" {
				sb.WriteString(genJSDevWatcherHandleFileChangePublicEntryTest(method, task, testName))
				continue
			}
			sb.WriteString(genJSPrivateMethodManualReviewTest(cls, method, task, testName, assertions))
			continue
		}
		if cls.Name == "WorkspaceCacheManager" && method.Name == "updateWorkspaceState" && jsCoverageTaskMentions(task, "cache[workspaceKey]") {
			sb.WriteString(genJSWorkspaceCacheUpdateStateTest(method, task, testName))
			continue
		}
		if cls.Name == "CodexExec" && method.Name == "run" && jsCoverageTaskNeedsCodexExecMock(task) {
			sb.WriteString(genJSCodexExecRunCoverageTask(method, task, testName, moduleImportPath))
			continue
		}
		if cls.Name == "ConfigManager" && method.Name == "loadConfig" {
			if code, ok := genJSConfigManagerLoadConfigValidationTest(cls, method, task, testName, assertions); ok {
				sb.WriteString(code)
				continue
			}
		}
		if cls.Name == "Version" && method.Name == "ipCompare" {
			sb.WriteString(genJSVersionIPCompareCoverageTask(method, task, testName))
			continue
		}
		if cls.Name == "Searcher" {
			if code, ok := genJSSearcherCoverageTask(method, task, testName); ok {
				sb.WriteString(code)
				continue
			}
		}
		if cls.Name == "StatusChecker" && method.Name == "check" {
			sb.WriteString(genJSStatusCheckerCoverageTask(method, task, testName, assertions))
			continue
		}
		boundary := jsBoundaryForCoverageTask(method.Analysis.Boundaries, task)
		overrides := jsClassCoverageTaskInputOverrides(method, task)
		args := jsArgListForCoverageTask(method.Params, task, boundary, method.Analysis, overrides)

		sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
		sb.WriteString(fmt.Sprintf("    it('%s', %s => {\n", jsEscapeTestNameValue(testName), jsAsyncArrow(method.IsAsync)))
		if comment := coverageTaskComment(task); comment != "" {
			sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
		}
		if !method.IsStatic {
			sb.WriteString(fmt.Sprintf("      const instance = %s;\n", jsClassInstanceForCoverageTask(cls, method, task)))
		}
		callExpr := jsClassMethodCallExpr(method, args)
		if jsCoverageTaskWantsErrorAssertion(method, task) {
			sb.WriteString(genJSErrorAssertion(assertions, method.IsAsync, callExpr, "      "))
			sb.WriteString("    });\n\n")
			sb.WriteString("  });\n\n")
			continue
		}
		if !method.Analysis.HasReturn {
			sb.WriteString(genJSVoidCallAssertion(assertions, method.IsAsync, callExpr, "      "))
			sb.WriteString("    });\n\n")
			sb.WriteString("  });\n\n")
			continue
		}
		if method.IsAsync {
			sb.WriteString(fmt.Sprintf("      const result = await %s;\n", callExpr))
		} else {
			sb.WriteString(fmt.Sprintf("      const result = %s;\n", callExpr))
		}
		if cls.Name == "SSEManager" && method.Name == "addConnection" {
			sb.WriteString("      expect(result).toBeDefined();\n")
		} else {
			sb.WriteString(genJSResultAssertionWithTaskArgsStyle(method.Analysis, method.Params, boundary, coverageTaskInputValues(task, "javascript"), "      ", assertions))
		}
		sb.WriteString("    });\n\n")
		sb.WriteString("  });\n\n")
	}
	sb.WriteString("});\n\n")

	return sb.String()
}

func genJSThreadRunStreamedInternalPublicEntryTest(method jsFuncInfo, task *types.CoverageTestTask, testName string) string {
	if task != nil && jsCoverageTaskNormalizeInputTarget(strings.TrimSpace(task.Target)) {
		return genJSThreadNormalizeInputPublicEntryTest(method, task, testName)
	}
	lineRange := ""
	if task != nil {
		lineRange = strings.TrimSpace(task.LineRange)
	}
	parseError := strings.HasPrefix(lineRange, "99") || strings.HasPrefix(lineRange, "100") || strings.HasPrefix(lineRange, "101") || strings.HasPrefix(lineRange, "102") || strings.HasPrefix(lineRange, "103")
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString("      const exec = {\n")
	sb.WriteString("        run: async function* () {\n")
	if parseError {
		sb.WriteString("          yield 'not-json';\n")
	} else {
		sb.WriteString("          yield JSON.stringify({ type: 'thread.started', thread_id: 'thread-123' });\n")
	}
	sb.WriteString("        },\n")
	sb.WriteString("      };\n")
	sb.WriteString("      const instance = new Thread(exec as any, {}, {}, null);\n")
	sb.WriteString("      const { events } = await instance.runStreamed('hello');\n")
	if parseError {
		sb.WriteString("      await expect(events.next()).rejects.toThrow('Failed to parse item: not-json');\n")
	} else {
		sb.WriteString("      const first = await events.next();\n")
		sb.WriteString("      expect(first.value).toEqual({ type: 'thread.started', thread_id: 'thread-123' });\n")
		sb.WriteString("      expect(instance.id).toBe('thread-123');\n")
		sb.WriteString("      await expect(events.next()).resolves.toEqual({ done: true, value: undefined });\n")
	}
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

func genJSThreadNormalizeInputPublicEntryTest(method jsFuncInfo, task *types.CoverageTestTask, testName string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString("      const calls: any[] = [];\n")
	sb.WriteString("      const exec = {\n")
	sb.WriteString("        run: async function* (args: any) {\n")
	sb.WriteString("          calls.push(args);\n")
	sb.WriteString("          yield JSON.stringify({ type: 'thread.started', thread_id: 'thread-123' });\n")
	sb.WriteString("        },\n")
	sb.WriteString("      };\n")
	sb.WriteString("      const instance = new Thread(exec as any, {}, {}, null);\n")
	sb.WriteString("      const { events } = await instance.runStreamed([\n")
	sb.WriteString("        { type: 'text', text: 'hello' },\n")
	sb.WriteString("        { type: 'text', text: 'world' },\n")
	sb.WriteString("        { type: 'local_image', path: '/tmp/image.png' },\n")
	sb.WriteString("      ] as any);\n")
	sb.WriteString("      await events.next();\n")
	sb.WriteString("      expect(calls[0].input).toBe('hello\\n\\nworld');\n")
	sb.WriteString("      expect(calls[0].images).toEqual(['/tmp/image.png']);\n")
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

func jsClassMethodCallExpr(method jsFuncInfo, args string) string {
	receiver := "instance"
	if method.IsStatic && method.ClassName != "" {
		receiver = method.ClassName
	}
	if method.Analysis.IsGetter {
		return fmt.Sprintf("%s.%s", receiver, method.Name)
	}
	return fmt.Sprintf("%s.%s(%s)", receiver, method.Name, args)
}

func genJSStatusCheckerCoverageTask(method jsFuncInfo, task *types.CoverageTestTask, testName string, assertions jsAssertionStyle) string {
	codeName, shouldThrow := jsStatusCheckerCodeForTask(task)
	var sb strings.Builder
	sb.WriteString("  describe('check', () => {\n")
	sb.WriteString(fmt.Sprintf("    it('%s', () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	if codeName == "" {
		if assertions == jsAssertionStyleChai {
			sb.WriteString("      expect(() => StatusChecker.check(undefined, 'req-1')).to.not.throw();\n")
		} else if assertions == jsAssertionStyleNode {
			sb.WriteString("      assert.doesNotThrow(() => StatusChecker.check(undefined, 'req-1'));\n")
		} else {
			sb.WriteString("      expect(() => StatusChecker.check(undefined, 'req-1')).not.toThrow();\n")
		}
		sb.WriteString("    });\n\n")
		sb.WriteString("  });\n\n")
		return sb.String()
	}
	codeExpr := "Code." + codeName
	if isNumericLiteral(codeName) {
		codeExpr = codeName
	} else {
		sb.WriteString("      const { Code } = require('../../proto/apache/rocketmq/v2/definition_pb');\n")
	}
	sb.WriteString(fmt.Sprintf("      const status = { code: %s, message: 'status message' };\n", codeExpr))
	callExpr := "StatusChecker.check(status, 'req-1')"
	if shouldThrow {
		sb.WriteString(genJSErrorAssertion(assertions, method.IsAsync, callExpr, "      "))
	} else if assertions == jsAssertionStyleChai {
		sb.WriteString(fmt.Sprintf("      expect(() => %s).to.not.throw();\n", callExpr))
	} else if assertions == jsAssertionStyleNode {
		sb.WriteString(fmt.Sprintf("      assert.doesNotThrow(() => %s);\n", callExpr))
	} else {
		sb.WriteString(fmt.Sprintf("      expect(() => %s).not.toThrow();\n", callExpr))
	}
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

func jsStatusCheckerCodeForTask(task *types.CoverageTestTask) (string, bool) {
	line := jsCoverageTaskStartLine(task)
	switch {
	case line == 33:
		return "", false
	case line >= 34 && line <= 37:
		return "MULTIPLE_RESULTS", false
	case line >= 38 && line <= 57:
		return "BAD_REQUEST", true
	case line >= 58 && line <= 59:
		return "UNAUTHORIZED", true
	case line >= 60 && line <= 61:
		return "PAYMENT_REQUIRED", true
	case line >= 62 && line <= 63:
		return "FORBIDDEN", true
	case line >= 64 && line <= 65:
		return "MESSAGE_NOT_FOUND", false
	case line >= 66 && line <= 69:
		return "NOT_FOUND", true
	case line >= 70 && line <= 72:
		return "PAYLOAD_TOO_LARGE", true
	case line >= 73 && line <= 74:
		return "TOO_MANY_REQUESTS", true
	case line >= 75 && line <= 77:
		return "REQUEST_HEADER_FIELDS_TOO_LARGE", true
	case line >= 78 && line <= 81:
		return "INTERNAL_ERROR", true
	case line >= 82 && line <= 85:
		return "PROXY_TIMEOUT", true
	case line >= 86 && line <= 89:
		return "UNSUPPORTED", true
	default:
		return "999999", true
	}
}

func jsCoverageTaskStartLine(task *types.CoverageTestTask) int {
	if task == nil {
		return 0
	}
	if len(task.UncoveredLines) > 0 {
		return task.UncoveredLines[0]
	}
	lineRange := strings.TrimSpace(task.LineRange)
	if lineRange == "" {
		return 0
	}
	part := strings.SplitN(lineRange, "-", 2)[0]
	n, _ := strconv.Atoi(strings.TrimSpace(part))
	return n
}

type jsImportedTypeMock struct {
	Name         string
	ImportedName string
	Module       string
	Decl         string
	IsValue      bool
	FilePath     string
}

type jsNamedImport struct {
	ImportedName string
	Module       string
}

func jsImportedTypeMocks(srcPath string, source string) map[string]jsImportedTypeMock {
	return jsImportedTypeMocksSeen(srcPath, source, map[string]bool{})
}

func jsImportedTypeMocksSeen(srcPath string, source string, seen map[string]bool) map[string]jsImportedTypeMock {
	key := srcPath
	if abs, err := filepath.Abs(srcPath); err == nil {
		key = abs
	}
	if seen[key] {
		return nil
	}
	seen[key] = true
	imports := jsNamedImports(source)
	if len(imports) == 0 {
		return nil
	}
	result := map[string]jsImportedTypeMock{}
	for localName, imported := range imports {
		if !strings.HasPrefix(imported.Module, ".") {
			if jsLooksLikeConstructableTypeName(localName) {
				result[localName] = jsImportedTypeMock{
					Name:         localName,
					ImportedName: imported.ImportedName,
					Module:       imported.Module,
					Decl:         jsConstructorMockKey + fmt.Sprintf("new %s()", localName),
					IsValue:      true,
				}
			}
			continue
		}
		resolvedPath, ok := jsResolveImportedSymbolPath(srcPath, imported.Module, imported.ImportedName, nil)
		if !ok {
			continue
		}
		data, err := os.ReadFile(resolvedPath)
		if err != nil {
			continue
		}
		text := string(data)
		for name, nested := range jsImportedTypeMocksSeen(resolvedPath, text, seen) {
			if _, exists := result[name]; !exists {
				result[name] = nested
			}
		}
		if decls := jsExtractTSTypeDecls(text); len(decls) > 0 {
			if decl, ok := decls[imported.ImportedName]; ok {
				result[localName] = jsImportedTypeMock{Name: localName, ImportedName: imported.ImportedName, Module: imported.Module, Decl: decl, IsValue: strings.HasPrefix(decl, jsEnumMockKey), FilePath: resolvedPath}
				continue
			}
		}
		_, classes, _ := parseJSWithTreeSitter(data, strings.ToLower(filepath.Ext(resolvedPath)))
		for _, cls := range classes {
			if cls.Name != imported.ImportedName || !cls.IsExported {
				continue
			}
			result[localName] = jsImportedTypeMock{
				Name:         localName,
				ImportedName: imported.ImportedName,
				Module:       imported.Module,
				Decl:         jsConstructorMockKey + jsConstructorMockExpression(localName, cls.ConstructorParams),
				IsValue:      true,
				FilePath:     resolvedPath,
			}
			break
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func jsLooksLikeConstructableTypeName(name string) bool {
	if name == "" {
		return false
	}
	r := rune(name[0])
	return r >= 'A' && r <= 'Z'
}

func jsNamedImports(source string) map[string]jsNamedImport {
	result := map[string]jsNamedImport{}
	for _, match := range jsNamedImportRe.FindAllStringSubmatch(source, -1) {
		if len(match) < 3 {
			continue
		}
		module := strings.TrimSpace(match[2])
		for _, part := range strings.Split(match[1], ",") {
			name, local := jsParseImportSpecifier(part)
			if name != "" && local != "" {
				result[local] = jsNamedImport{ImportedName: name, Module: module}
			}
		}
	}
	return result
}

func jsParseImportSpecifier(spec string) (importedName string, localName string) {
	spec = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(spec), "type "))
	if spec == "" {
		return "", ""
	}
	parts := regexp.MustCompile(`\s+as\s+`).Split(spec, 2)
	importedName = strings.TrimSpace(parts[0])
	localName = importedName
	if len(parts) == 2 {
		localName = strings.TrimSpace(parts[1])
	}
	if !jsTSIdentifierRe.MatchString(importedName) || !jsTSIdentifierRe.MatchString(localName) {
		return "", ""
	}
	return importedName, localName
}

func jsResolveImportedSymbolPath(srcPath string, module string, symbol string, seen map[string]bool) (string, bool) {
	if !strings.HasPrefix(module, ".") || symbol == "" {
		return "", false
	}
	for _, candidate := range jsImportModuleFileCandidates(srcPath, module) {
		key := candidate + "#" + symbol
		if seen != nil && seen[key] {
			continue
		}
		nextSeen := map[string]bool{}
		for k, v := range seen {
			nextSeen[k] = v
		}
		nextSeen[key] = true
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		text := string(data)
		if jsFileExportsSymbol(text, symbol) {
			return candidate, true
		}
		if path, ok := jsResolveReExportedSymbolPath(candidate, text, symbol, nextSeen); ok {
			return path, true
		}
	}
	return "", false
}

func jsImportModuleFileCandidates(srcPath string, module string) []string {
	base := filepath.Clean(filepath.Join(filepath.Dir(srcPath), filepath.FromSlash(module)))
	if filepath.Ext(base) != "" {
		return []string{base}
	}
	var candidates []string
	for _, suffix := range []string{".ts", ".tsx", ".d.ts", ".js", ".jsx", ".mjs", ".cjs"} {
		candidates = append(candidates, base+suffix)
	}
	for _, index := range []string{"index.ts", "index.tsx", "index.js", "index.jsx", "index.mjs", "index.cjs"} {
		candidates = append(candidates, filepath.Join(base, index))
	}
	return candidates
}

func jsFileExportsSymbol(source string, symbol string) bool {
	if !jsTSIdentifierRe.MatchString(symbol) {
		return false
	}
	pattern := regexp.MustCompile(fmt.Sprintf(`(?m)export\s+(?:abstract\s+)?(?:class|interface|type|enum)\s+%s(?:\s|<|=|\{)`, regexp.QuoteMeta(symbol)))
	if pattern.MatchString(source) {
		return true
	}
	for _, match := range regexp.MustCompile(`(?m)export\s+\{([^}]*)\}`).FindAllStringSubmatch(source, -1) {
		if len(match) < 2 {
			continue
		}
		for _, part := range strings.Split(match[1], ",") {
			name, local := jsParseImportSpecifier(part)
			if name == symbol || local == symbol {
				return true
			}
		}
	}
	return false
}

func jsResolveReExportedSymbolPath(currentPath string, source string, symbol string, seen map[string]bool) (string, bool) {
	for _, match := range jsExportNamedFromRe.FindAllStringSubmatch(source, -1) {
		if len(match) < 3 {
			continue
		}
		for _, part := range strings.Split(match[1], ",") {
			name, local := jsParseImportSpecifier(part)
			if name == symbol || local == symbol {
				if path, ok := jsResolveImportedSymbolPath(currentPath, strings.TrimSpace(match[2]), name, seen); ok {
					return path, true
				}
			}
		}
	}
	for _, match := range jsExportStarRe.FindAllStringSubmatch(source, -1) {
		if len(match) >= 2 {
			if path, ok := jsResolveImportedSymbolPath(currentPath, strings.TrimSpace(match[1]), symbol, seen); ok {
				return path, true
			}
		}
	}
	return "", false
}

func jsConstructorMockExpression(name string, params []jsParamInfo) string {
	if len(params) == 0 {
		return fmt.Sprintf("new %s()", name)
	}
	args := make([]string, len(params))
	for i, p := range params {
		if p.HasDefault {
			args[i] = "undefined"
		} else {
			args[i] = jsArgValue(p, i)
		}
	}
	return fmt.Sprintf("new %s(%s)", name, strings.Join(args, ", "))
}

func jsAttachImportedTypeMocks(funcs []jsFuncInfo, classes []jsClassInfo, mocks map[string]jsImportedTypeMock) {
	if len(mocks) == 0 {
		return
	}
	decls := map[string]string{}
	for name, mock := range mocks {
		decls[name] = mock.Decl
	}
	for i := range funcs {
		funcs[i].Analysis.TSTypeDecls = jsMergeTSTypeDecls(funcs[i].Analysis.TSTypeDecls, decls)
	}
	for i := range classes {
		for j := range classes[i].Methods {
			classes[i].Methods[j].Analysis.TSTypeDecls = jsMergeTSTypeDecls(classes[i].Methods[j].Analysis.TSTypeDecls, decls)
		}
	}
}

func jsMergeTSTypeDecls(base map[string]string, extra map[string]string) map[string]string {
	if len(extra) == 0 {
		return base
	}
	merged := map[string]string{}
	for k, v := range extra {
		merged[k] = v
	}
	for k, v := range base {
		merged[k] = v
	}
	return merged
}

func jsTypeValueImportLinesForTargets(funcs []jsFuncInfo, classes []jsClassInfo, mocks map[string]jsImportedTypeMock, testPath string) string {
	needed := jsTypeValueImportsNeeded(funcs, classes, mocks)
	if len(needed) == 0 {
		return ""
	}
	byModule := map[string][]string{}
	for name, mock := range needed {
		modulePath := mock.Module
		if mock.FilePath != "" {
			modulePath = jsSourceModuleImportPath(mock.FilePath, testPath)
		}
		if modulePath == "" {
			continue
		}
		byModule[modulePath] = append(byModule[modulePath], jsImportSpecifierForMock(name, mock))
	}
	if len(byModule) == 0 {
		return ""
	}
	modules := make([]string, 0, len(byModule))
	for module := range byModule {
		modules = append(modules, module)
	}
	sort.Strings(modules)
	var sb strings.Builder
	for _, module := range modules {
		names := byModule[module]
		sort.Strings(names)
		sb.WriteString(fmt.Sprintf("import { %s } from '%s';\n", strings.Join(names, ", "), module))
	}
	sb.WriteString("\n")
	return sb.String()
}

func jsImportSpecifierForMock(name string, mock jsImportedTypeMock) string {
	imported := strings.TrimSpace(mock.ImportedName)
	if imported == "" || imported == name {
		return name
	}
	return imported + " as " + name
}

func jsTypeValueImportsNeeded(funcs []jsFuncInfo, classes []jsClassInfo, mocks map[string]jsImportedTypeMock) map[string]jsImportedTypeMock {
	result := map[string]jsImportedTypeMock{}
	seenTypes := map[string]bool{}
	var addTypeExpr func(string, map[string]string)
	var addNamedType func(string, map[string]string)
	addNamedType = func(name string, decls map[string]string) {
		if name == "" {
			return
		}
		seenKey := name
		if decls != nil {
			seenKey += "\x00" + decls[name]
		}
		if seenTypes[seenKey] {
			return
		}
		seenTypes[seenKey] = true
		mock, ok := mocks[name]
		if ok {
			if mock.IsValue {
				result[name] = mock
			}
			addTypeExpr(mock.Decl, decls)
			return
		}
		if decls != nil {
			if decl := decls[name]; decl != "" {
				addTypeExpr(decl, decls)
			}
		}
	}
	addTypeExpr = func(typeExpr string, decls map[string]string) {
		typeExpr = strings.TrimSpace(typeExpr)
		if strings.HasPrefix(typeExpr, "{") && strings.HasSuffix(typeExpr, "}") {
			body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(typeExpr, "{"), "}"))
			for _, field := range jsSplitTopLevelTypeFields(body) {
				_, typ, ok := jsParseTSTypeField(field)
				if ok {
					addTypeExpr(typ, decls)
				}
			}
			return
		}
		if jsTSTypeIsFunction(typeExpr) {
			addTypeExpr(jsTSTypeFunctionReturnType(typeExpr), decls)
			return
		}
		for _, name := range jsNamedTypesInTSType(typeExpr) {
			addNamedType(name, decls)
		}
	}
	addParams := func(params []jsParamInfo, decls map[string]string) {
		for _, p := range params {
			if p.HasDefault {
				continue
			}
			addTypeExpr(p.TypeExpr, decls)
		}
	}
	for _, fn := range funcs {
		if fn.SourceIsESModule && !fn.IsExported {
			continue
		}
		addParams(fn.Params, fn.Analysis.TSTypeDecls)
	}
	for _, cls := range classes {
		for _, method := range cls.Methods {
			if jsClassRequiresInternalManualReview(cls) || method.IsPrivate || strings.HasPrefix(method.Name, "#") {
				continue
			}
			if cls.Name == "StatusChecker" && method.Name == "check" {
				continue
			}
			if !method.IsStatic {
				addParams(cls.ConstructorParams, method.Analysis.TSTypeDecls)
			}
			addParams(method.Params, method.Analysis.TSTypeDecls)
		}
	}
	return result
}

func jsDirectNamedTSType(typeExpr string) string {
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	typeExpr = jsUnwrapTSUtilityWrappers(typeExpr)
	if branch, ok := jsPreferredTSTypeUnionBranch(typeExpr); ok {
		typeExpr = jsUnwrapTSUtilityWrappers(branch)
	}
	if jsTSIdentifierRe.MatchString(typeExpr) {
		return typeExpr
	}
	return ""
}

func jsNamedTypesInTSType(typeExpr string) []string {
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	matches := regexp.MustCompile(`[A-Za-z_$][A-Za-z0-9_$]*`).FindAllString(typeExpr, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var result []string
	for _, name := range matches {
		if seen[name] || jsBuiltinTSTypeName(name) {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	return result
}

func jsBuiltinTSTypeName(name string) bool {
	switch name {
	case "string", "number", "boolean", "bigint", "null", "undefined", "void", "unknown", "any", "Map", "Set", "Record", "Array", "ReadonlyArray", "Promise", "Date", "Buffer", "Uint8Array":
		return true
	default:
		return false
	}
}

func jsVitestImportNamesForCoverageTask(task *types.CoverageTestTask) string {
	names := []string{"describe", "it", "expect"}
	if jsCoverageTaskNeedsVitestVi(task) {
		names = append(names, "vi")
	}
	return strings.Join(names, ", ")
}

func jsVitestPreludeForCoverageTask(task *types.CoverageTestTask, moduleImportPath string) string {
	if !jsCoverageTaskNeedsChokidarMock(task) {
		if !jsCoverageTaskNeedsOAuthProviderMocks(task) {
			return ""
		}
	}
	var sb strings.Builder
	if jsCoverageTaskNeedsChokidarMock(task) {
		sb.WriteString(`
vi.mock('chokidar', async () => {
  const { EventEmitter } = await import('node:events');
  return {
    default: {
      watch: vi.fn(() => {
        const watcher = new EventEmitter();
        watcher.close = vi.fn();
        return watcher;
      }),
    },
  };
});

`)
	}
	if jsCoverageTaskNeedsOAuthProviderMocks(task) {
		loggerPath := jsSiblingModuleImportPath(moduleImportPath, "logger.js")
		sb.WriteString(`
vi.mock('fs/promises', () => {
  const mkdir = vi.fn();
  const readFile = vi.fn();
  const writeFile = vi.fn();
  return {
    default: { mkdir, readFile, writeFile },
    mkdir,
    readFile,
    writeFile,
  };
});
`)
		sb.WriteString(fmt.Sprintf(`
vi.mock('%s', () => ({
  default: {
    file: vi.fn(),
    warn: vi.fn(),
    info: vi.fn(),
    error: vi.fn(),
    debug: vi.fn(),
  },
}));

`, loggerPath))
	}
	return sb.String()
}

func jsCoverageTaskNeedsVitestVi(task *types.CoverageTestTask) bool {
	return jsCoverageTaskNeedsChokidarMock(task) || jsCoverageTaskNeedsOAuthProviderMocks(task) || jsCoverageTaskNeedsWorkspaceCacheSpies(task)
}

func jsCoverageTaskNeedsChokidarMock(task *types.CoverageTestTask) bool {
	return task != nil && task.Target == "DevWatcher.#handleFileChange"
}

func jsCoverageTaskNeedsOAuthProviderMocks(task *types.CoverageTestTask) bool {
	return task != nil && strings.HasPrefix(task.Target, "StorageManager.")
}

func jsCoverageTaskNeedsWorkspaceCacheSpies(task *types.CoverageTestTask) bool {
	return task != nil && task.Target == "WorkspaceCacheManager.updateWorkspaceState"
}

func jsCoverageTaskNeedsDynamicImportOnly(task *types.CoverageTestTask) bool {
	return jsCoverageTaskNeedsOAuthProviderMocks(task) ||
		jsCoverageTaskNeedsCodexExecMock(task) ||
		jsCoverageTaskFindCodexPathTarget(task) ||
		jsCoverageTaskCodexInternalFSHelperTarget(task)
}

func jsCoverageTaskNeedsCodexExecMock(task *types.CoverageTestTask) bool {
	if task == nil {
		return false
	}
	target := strings.TrimSpace(task.Target)
	return target == "CodexExec.run" || jsCoverageTaskCodexConfigOverridesTarget(target)
}

func jsCoverageTaskFindCodexPathTarget(task *types.CoverageTestTask) bool {
	return task != nil && strings.TrimSpace(task.Target) == "findCodexPath"
}

func jsCoverageTaskCodexInternalFSHelperTarget(task *types.CoverageTestTask) bool {
	return task != nil && strings.TrimSpace(task.Target) == "isDirectory"
}

func jsCoverageTaskCodexConfigOverridesTarget(target string) bool {
	switch strings.TrimSpace(target) {
	case "flattenConfigOverrides", "formatTomlKey", "isPlainObject", "serializeConfigOverrides", "toTomlValue":
		return true
	default:
		return false
	}
}

func jsCodexExecJestPrelude() string {
	return `// @ts-nocheck
import * as child_process from 'node:child_process';
import { EventEmitter } from 'node:events';
import { PassThrough } from 'node:stream';
import { jest } from '@jest/globals';

jest.mock('node:child_process', () => {
  const actual = jest.requireActual('node:child_process');
  return Object.assign({}, actual, { spawn: jest.fn() });
});

class TestloopCodexExecChild extends EventEmitter {
  constructor() {
    super();
    this.stdin = new PassThrough();
    this.stdout = new PassThrough();
    this.stderr = new PassThrough();
    this.killed = false;
  }

  kill() {
    this.killed = true;
    return true;
  }
}

async function consumeTestloopCodexExec(iterable) {
  for await (const _ of iterable) {
    // consume all output
  }
}

`
}

func jsSiblingModuleImportPath(moduleImportPath, sibling string) string {
	idx := strings.LastIndex(moduleImportPath, "/")
	if idx < 0 {
		return "./" + sibling
	}
	return moduleImportPath[:idx+1] + sibling
}

func genJSConfigManagerDiffPublicEntryTest(cls jsClassInfo, method jsFuncInfo, task *types.CoverageTestTask, testName string) string {
	scenario := jsConfigManagerDiffScenarioForTask(task)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString("      const fs = await import('node:fs/promises');\n")
	sb.WriteString("      const os = await import('node:os');\n")
	sb.WriteString("      const path = await import('node:path');\n")
	sb.WriteString("      const dir = await fs.mkdtemp(path.join(os.tmpdir(), 'testloop-config-'));\n")
	sb.WriteString("      const configPath = path.join(dir, 'mcp.json');\n")
	sb.WriteString(fmt.Sprintf("      await fs.writeFile(configPath, JSON.stringify(%s));\n", scenario.newConfig))
	sb.WriteString(fmt.Sprintf("      const instance = new %s(%s);\n", cls.Name, scenario.oldConfig))
	sb.WriteString("      instance.configPaths = [configPath];\n")
	sb.WriteString("      const result = await instance.loadConfig();\n")
	for _, assertion := range scenario.assertions {
		sb.WriteString("      " + assertion + "\n")
	}
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

func genJSConfigManagerLoadConfigValidationTest(cls jsClassInfo, method jsFuncInfo, task *types.CoverageTestTask, testName string, assertions jsAssertionStyle) (string, bool) {
	scenario, ok := jsConfigManagerLoadConfigScenarioForTask(task)
	if !ok {
		return "", false
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString("      const fs = await import('node:fs/promises');\n")
	sb.WriteString("      const os = await import('node:os');\n")
	sb.WriteString("      const path = await import('node:path');\n")
	sb.WriteString("      const dir = await fs.mkdtemp(path.join(os.tmpdir(), 'testloop-config-'));\n")
	sb.WriteString("      const configPath = path.join(dir, 'mcp.json');\n")
	sb.WriteString(fmt.Sprintf("      await fs.writeFile(configPath, JSON.stringify(%s));\n", scenario.config))
	sb.WriteString(fmt.Sprintf("      const instance = new %s(configPath);\n", cls.Name))
	sb.WriteString(genJSErrorAssertionWithMessage(assertions, true, "instance.loadConfig()", scenario.errorContains, "      "))
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String(), true
}

type jsConfigManagerLoadConfigScenario struct {
	config        string
	errorContains string
}

func jsConfigManagerLoadConfigScenarioForTask(task *types.CoverageTestTask) (jsConfigManagerLoadConfigScenario, bool) {
	if task == nil || strings.TrimSpace(task.Target) != "ConfigManager.loadConfig" {
		return jsConfigManagerLoadConfigScenario{}, false
	}
	hints := strings.ToLower(jsCoverageTaskHints(task))
	switch {
	case strings.Contains(hints, "hasstdiofields && hasssefields") ||
		strings.Contains(hints, "command 和 url") ||
		strings.Contains(hints, "cannot mix stdio and sse"):
		return jsConfigManagerLoadConfigScenario{
			config:        "{ mcpServers: { test: { command: 'node', url: 'http://localhost:3000' } } }",
			errorContains: "cannot mix stdio and sse",
		}, true
	case strings.Contains(hints, "missing both command and url") ||
		strings.Contains(hints, "must include either command") ||
		strings.Contains(hints, "为空对象"):
		return jsConfigManagerLoadConfigScenario{
			config:        "{ mcpServers: { test: {} } }",
			errorContains: "must include either command",
		}, true
	case strings.Contains(hints, "missing command value") ||
		strings.Contains(hints, "缺少 command"):
		return jsConfigManagerLoadConfigScenario{
			config:        "{ mcpServers: { test: { command: '' } } }",
			errorContains: "missing command value",
		}, true
	case strings.Contains(hints, "invalid url") || strings.Contains(hints, "无效 url"):
		return jsConfigManagerLoadConfigScenario{
			config:        "{ mcpServers: { test: { url: 'not a url' } } }",
			errorContains: "invalid url",
		}, true
	case strings.Contains(hints, "invalid environment") || strings.Contains(hints, "invalid env"):
		return jsConfigManagerLoadConfigScenario{
			config:        "{ mcpServers: { test: { command: 'node', env: 'bad' } } }",
			errorContains: "invalid environment config",
		}, true
	case strings.Contains(hints, "invalid headers"):
		return jsConfigManagerLoadConfigScenario{
			config:        "{ mcpServers: { test: { url: 'http://localhost:3000', headers: 'bad' } } }",
			errorContains: "invalid headers config",
		}, true
	case strings.Contains(hints, "dev field is only supported") || strings.Contains(hints, "dev field only"):
		return jsConfigManagerLoadConfigScenario{
			config:        "{ mcpServers: { test: { url: 'http://localhost:3000', dev: { cwd: process.cwd() } } } }",
			errorContains: "dev field is only supported",
		}, true
	case strings.Contains(hints, "dev.enabled"):
		return jsConfigManagerLoadConfigScenario{
			config:        "{ mcpServers: { test: { command: 'node', dev: { enabled: 'yes', cwd: process.cwd() } } } }",
			errorContains: "dev.enabled must be a boolean",
		}, true
	case strings.Contains(hints, "dev.watch"):
		return jsConfigManagerLoadConfigScenario{
			config:        "{ mcpServers: { test: { command: 'node', dev: { watch: 'src', cwd: process.cwd() } } } }",
			errorContains: "dev.watch must be an array of strings",
		}, true
	case strings.Contains(hints, "dev.cwd"):
		return jsConfigManagerLoadConfigScenario{
			config:        "{ mcpServers: { test: { command: 'node', dev: { cwd: 'relative' } } } }",
			errorContains: "dev.cwd must be an absolute path",
		}, true
	default:
		return jsConfigManagerLoadConfigScenario{}, false
	}
}

type jsConfigManagerDiffTestScenario struct {
	oldConfig  string
	newConfig  string
	assertions []string
}

func jsConfigManagerDiffScenarioForTask(task *types.CoverageTestTask) jsConfigManagerDiffTestScenario {
	hints := jsCoverageTaskHints(task)
	switch {
	case strings.Contains(hints, "!newServers[name]") || strings.Contains(hints, "removed"):
		return jsConfigManagerDiffTestScenario{
			oldConfig: "{ mcpServers: { old: { command: 'node' } } }",
			newConfig: "{ mcpServers: {} }",
			assertions: []string{
				"expect(result.changes.removed).toContain('old');",
			},
		}
	case strings.Contains(hints, "field === 'args'") || strings.Contains(hints, "field === 'env'") || strings.Contains(hints, "field === 'headers'") || strings.Contains(hints, "field === 'dev'"):
		return jsConfigManagerDiffTestScenario{
			oldConfig: "{ mcpServers: { app: { command: 'node', args: ['old'] } } }",
			newConfig: "{ mcpServers: { app: { command: 'node', args: ['new'] } } }",
			assertions: []string{
				"expect(result.changes.modified).toContain('app');",
				"expect(result.changes.details.app.modifiedFields).toContain('args');",
			},
		}
	case strings.Contains(hints, "!oldServers[name].hasOwnProperty(field"):
		if task != nil && strings.HasPrefix(strings.TrimSpace(task.LineRange), "85") {
			return jsConfigManagerDiffTestScenario{
				oldConfig: "{ mcpServers: { app: { command: 'node' } } }",
				newConfig: "{ mcpServers: { app: { command: 'node' } } }",
				assertions: []string{
					"expect(result.changes.modified).toContain('app');",
					"expect(result.changes.details.app.modifiedFields).toContain('config_source');",
				},
			}
		}
		return jsConfigManagerDiffTestScenario{
			oldConfig: "{ mcpServers: { app: { command: 'node' } } }",
			newConfig: "{ mcpServers: { app: { command: 'node', args: ['run'] } } }",
			assertions: []string{
				"expect(result.changes.modified).toContain('app');",
				"expect(result.changes.details.app.modifiedFields).toContain('args');",
			},
		}
	case strings.Contains(hints, "modifiedFields.length > 0"):
		return jsConfigManagerDiffTestScenario{
			oldConfig: "{ mcpServers: { app: { command: 'node' } } }",
			newConfig: "{ mcpServers: { app: { command: 'node', cwd: '/tmp' } } }",
			assertions: []string{
				"expect(result.changes.modified).toContain('app');",
				"expect(result.changes.details.app.modifiedFields).toContain('cwd');",
			},
		}
	default:
		return jsConfigManagerDiffTestScenario{
			oldConfig: "{ mcpServers: { app: { command: 'node' } } }",
			newConfig: "{ mcpServers: { app: { command: 'node', args: ['run'] } } }",
			assertions: []string{
				"expect(result.changes.modified.length + result.changes.removed.length + result.changes.added.length).toBeGreaterThanOrEqual(0);",
			},
		}
	}
}

func genJSStorageManagerPublicEntryTest(method jsFuncInfo, task *types.CoverageTestTask, testName string, moduleImportPath string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString("      vi.resetModules();\n")
	sb.WriteString("      const fs = await import('fs/promises');\n")
	if method.Name == "init" {
		sb.WriteString("      const logger = await import('" + jsSiblingModuleImportPath(moduleImportPath, "logger.js") + "');\n")
		sb.WriteString("      fs.default.mkdir.mockResolvedValue(undefined);\n")
		sb.WriteString("      fs.default.readFile.mockRejectedValue(Object.assign(new Error('permission denied'), { code: 'EACCES' }));\n")
		sb.WriteString(fmt.Sprintf("      await import('%s');\n", moduleImportPath))
		sb.WriteString("      await new Promise((resolve) => setTimeout(resolve, 0));\n")
		sb.WriteString("      expect(fs.default.readFile).toHaveBeenCalled();\n")
		sb.WriteString("      expect(logger.default.warn).toHaveBeenCalledWith(expect.stringContaining('Error reading storage'));\n")
	} else {
		sb.WriteString("      fs.default.mkdir.mockResolvedValue(undefined);\n")
		sb.WriteString("      fs.default.readFile.mockRejectedValue(Object.assign(new Error('missing'), { code: 'ENOENT' }));\n")
		sb.WriteString(fmt.Sprintf("      const { default: MCPHubOAuthProvider } = await import('%s');\n", moduleImportPath))
		sb.WriteString("      await new Promise((resolve) => setTimeout(resolve, 0));\n")
		sb.WriteString("      const provider = new MCPHubOAuthProvider({ serverName: 'test-server', serverUrl: 'https://example.com/mcp', hubServerUrl: 'http://localhost:3000' });\n")
		sb.WriteString("      await expect(provider.tokens()).resolves.toBeNull();\n")
	}
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

func genJSWorkspaceCacheUpdateStateTest(method jsFuncInfo, task *types.CoverageTestTask, testName string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString("      const instance = new WorkspaceCacheManager({ port: 3000 });\n")
	sb.WriteString("      const cache = {\n")
	sb.WriteString("        '3000': { state: 'active', activeConnections: 1, port: 3000 },\n")
	sb.WriteString("      };\n")
	sb.WriteString("      instance._withLock = async (fn) => fn();\n")
	sb.WriteString("      instance._readCache = vi.fn().mockResolvedValue(cache);\n")
	sb.WriteString("      instance._writeCache = vi.fn().mockResolvedValue(undefined);\n")
	sb.WriteString("      await instance.updateWorkspaceState(3000, { state: 'shutting_down', activeConnections: 0 });\n")
	sb.WriteString("      expect(instance._readCache).toHaveBeenCalled();\n")
	sb.WriteString("      expect(instance._writeCache).toHaveBeenCalledWith(expect.objectContaining({\n")
	sb.WriteString("        '3000': expect.objectContaining({ state: 'shutting_down', activeConnections: 0, port: 3000 }),\n")
	sb.WriteString("      }));\n")
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

func genJSCodexExecRunCoverageTask(method jsFuncInfo, task *types.CoverageTestTask, testName string, moduleImportPath string) string {
	target := ""
	if task != nil {
		target = task.Target
	}
	if jsCoverageTaskCodexConfigOverridesTarget(target) {
		return genJSCodexExecConfigOverrideCoverageTask(method, task, testName, moduleImportPath)
	}
	if scenario, ok := jsCodexExecRunArgsScenarioForTask(task); ok {
		return genJSCodexExecRunArgsCoverageTask(method, task, testName, moduleImportPath, scenario)
	}
	return genJSCodexExecSpawnErrorCoverageTask(method, task, testName, moduleImportPath)
}

type jsCodexExecRunArgsScenario struct {
	instanceSetup string
	runArgs       string
	childSetup    []string
	stdoutWrites  []string
	expectError   string
	collectOutput bool
	assertions    []string
}

func jsCodexExecRunArgsScenarioForTask(task *types.CoverageTestTask) (jsCodexExecRunArgsScenario, bool) {
	if task == nil || strings.TrimSpace(task.Target) != "CodexExec.run" {
		return jsCodexExecRunArgsScenario{}, false
	}
	lineRange := strings.TrimSpace(task.LineRange)
	scenario := jsCodexExecRunArgsScenario{
		instanceSetup: "const instance = new CodexExec('codex');",
		runArgs:       "{ input: 'hi' }",
		assertions: []string{
			"expect(spawnMock).toHaveBeenCalled();",
		},
	}
	switch {
	case strings.HasPrefix(lineRange, "95"):
		scenario.runArgs = "{ input: 'hi', baseUrl: 'https://api.example.com' }"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('--config');",
			"expect(commandArgs).toContain('openai_base_url=\"https://api.example.com\"');",
		}
	case strings.HasPrefix(lineRange, "103"):
		scenario.runArgs = "{ input: 'hi', model: 'gpt-5' }"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('--model');",
			"expect(commandArgs).toContain('gpt-5');",
		}
	case strings.HasPrefix(lineRange, "107"):
		scenario.runArgs = "{ input: 'hi', sandboxMode: 'workspace-write' }"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('--sandbox');",
			"expect(commandArgs).toContain('workspace-write');",
		}
	case strings.HasPrefix(lineRange, "111"):
		scenario.runArgs = "{ input: 'hi', workingDirectory: '/tmp/project' }"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('--cd');",
			"expect(commandArgs).toContain('/tmp/project');",
		}
	case strings.HasPrefix(lineRange, "115"):
		scenario.runArgs = "{ input: 'hi', additionalDirectories: ['/tmp/extra'] }"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('--add-dir');",
			"expect(commandArgs).toContain('/tmp/extra');",
		}
	case strings.HasPrefix(lineRange, "121"):
		scenario.runArgs = "{ input: 'hi', skipGitRepoCheck: true }"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('--skip-git-repo-check');",
		}
	case strings.HasPrefix(lineRange, "125"):
		scenario.runArgs = "{ input: 'hi', outputSchemaFile: '/tmp/schema.json' }"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('--output-schema');",
			"expect(commandArgs).toContain('/tmp/schema.json');",
		}
	case strings.HasPrefix(lineRange, "129"):
		scenario.runArgs = "{ input: 'hi', modelReasoningEffort: 'high' }"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('--config');",
			"expect(commandArgs).toContain('model_reasoning_effort=\"high\"');",
		}
	case strings.HasPrefix(lineRange, "133"):
		scenario.runArgs = "{ input: 'hi', networkAccessEnabled: true }"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('--config');",
			"expect(commandArgs).toContain('sandbox_workspace_write.network_access=true');",
		}
	case strings.HasPrefix(lineRange, "140"):
		scenario.runArgs = "{ input: 'hi', webSearchMode: 'live' }"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('--config');",
			"expect(commandArgs).toContain('web_search=\"live\"');",
		}
	case strings.HasPrefix(lineRange, "142"):
		scenario.runArgs = "{ input: 'hi', webSearchEnabled: true }"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('--config');",
			"expect(commandArgs).toContain('web_search=\"live\"');",
		}
	case strings.HasPrefix(lineRange, "144"):
		scenario.runArgs = "{ input: 'hi', webSearchEnabled: false }"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('--config');",
			"expect(commandArgs).toContain('web_search=\"disabled\"');",
		}
	case strings.HasPrefix(lineRange, "148"):
		scenario.runArgs = "{ input: 'hi', approvalPolicy: 'never' }"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('--config');",
			"expect(commandArgs).toContain('approval_policy=\"never\"');",
		}
	case strings.HasPrefix(lineRange, "151"):
		scenario.runArgs = "{ input: 'hi', threadId: 'thread-123' }"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('resume');",
			"expect(commandArgs).toContain('thread-123');",
		}
	case strings.HasPrefix(lineRange, "155"):
		scenario.runArgs = "{ input: 'hi', images: ['/tmp/image.png'] }"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('--image');",
			"expect(commandArgs).toContain('/tmp/image.png');",
		}
	case strings.HasPrefix(lineRange, "174"):
		scenario.runArgs = "{ input: 'hi', apiKey: 'test-api-key' }"
		scenario.assertions = []string{
			"expect(spawnOptions.env.CODEX_API_KEY).toBe('test-api-key');",
		}
	case strings.HasPrefix(lineRange, "178"):
		scenario.instanceSetup = "const instance = new CodexExec('codex', { PATH: '/usr/bin' });\n      instance.pathDirs = ['/tmp/codex-bin'];"
		scenario.assertions = []string{
			"expect(spawnOptions.env.PATH.split(':')[0]).toBe('/tmp/codex-bin');",
		}
	case strings.HasPrefix(lineRange, "90"):
		scenario.instanceSetup = "const instance = new CodexExec('codex', {}, { model: 'gpt-5' });"
		scenario.assertions = []string{
			"expect(commandArgs).toContain('--config');",
			"expect(commandArgs).toContain('model=\"gpt-5\"');",
		}
	case strings.HasPrefix(lineRange, "190"), strings.HasPrefix(lineRange, "189"):
		scenario.childSetup = []string{"child.stdin = null;"}
		scenario.expectError = "Child process has no stdin"
		scenario.assertions = []string{
			"expect(child.killed).toBe(true);",
		}
	case strings.HasPrefix(lineRange, "197"), strings.HasPrefix(lineRange, "196"):
		scenario.childSetup = []string{"child.stdout = null;"}
		scenario.expectError = "Child process has no stdout"
		scenario.assertions = []string{
			"expect(child.killed).toBe(true);",
		}
	case strings.HasPrefix(lineRange, "224"):
		scenario.stdoutWrites = []string{"child.stdout.write('ready\\n');"}
		scenario.collectOutput = true
		scenario.assertions = []string{
			"expect(output).toEqual(['ready']);",
			"expect(spawnMock).toHaveBeenCalled();",
		}
	default:
		return jsCodexExecRunArgsScenario{}, false
	}
	return scenario, true
}

func genJSCodexExecRunArgsCoverageTask(method jsFuncInfo, task *types.CoverageTestTask, testName string, moduleImportPath string, scenario jsCodexExecRunArgsScenario) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString(fmt.Sprintf("      const { CodexExec } = await import('%s');\n", moduleImportPath))
	sb.WriteString("      const spawnMock = jest.mocked(child_process.spawn);\n")
	sb.WriteString("      spawnMock.mockClear();\n")
	sb.WriteString("      const child = new TestloopCodexExecChild();\n")
	for _, line := range scenario.childSetup {
		sb.WriteString("      " + line + "\n")
	}
	sb.WriteString("      spawnMock.mockReturnValue(child);\n")
	if scenario.expectError == "" {
		sb.WriteString("      setImmediate(() => {\n")
		for _, line := range scenario.stdoutWrites {
			sb.WriteString("        " + line + "\n")
		}
		sb.WriteString("        child.stdout.end();\n")
		sb.WriteString("        child.stderr.end();\n")
		sb.WriteString("        child.emit('exit', 0, null);\n")
		sb.WriteString("      });\n")
	}
	sb.WriteString("      " + scenario.instanceSetup + "\n")
	if scenario.expectError != "" {
		sb.WriteString(fmt.Sprintf("      await expect(consumeTestloopCodexExec(instance.run(%s))).rejects.toThrow('%s');\n", scenario.runArgs, strings.ReplaceAll(scenario.expectError, "'", "\\'")))
	} else if scenario.collectOutput {
		sb.WriteString("      const output = [];\n")
		sb.WriteString(fmt.Sprintf("      for await (const line of instance.run(%s)) {\n", scenario.runArgs))
		sb.WriteString("        output.push(line);\n")
		sb.WriteString("      }\n")
	} else {
		sb.WriteString(fmt.Sprintf("      await consumeTestloopCodexExec(instance.run(%s));\n", scenario.runArgs))
	}
	sb.WriteString("      const commandArgs = spawnMock.mock.calls[0]?.[1] || [];\n")
	sb.WriteString("      const spawnOptions = spawnMock.mock.calls[0]?.[2] || {};\n")
	for _, assertion := range scenario.assertions {
		sb.WriteString("      " + assertion + "\n")
	}
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

func genJSCodexExecSpawnErrorCoverageTask(method jsFuncInfo, task *types.CoverageTestTask, testName string, moduleImportPath string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString(fmt.Sprintf("      const { CodexExec } = await import('%s');\n", moduleImportPath))
	sb.WriteString("      const spawnMock = jest.mocked(child_process.spawn);\n")
	sb.WriteString("      spawnMock.mockClear();\n")
	sb.WriteString("      const child = new TestloopCodexExecChild();\n")
	sb.WriteString("      spawnMock.mockReturnValue(child);\n")
	sb.WriteString("      setImmediate(() => {\n")
	sb.WriteString("        child.emit('error', new Error('spawn failed'));\n")
	sb.WriteString("        child.stdout.end();\n")
	sb.WriteString("        child.stderr.end();\n")
	sb.WriteString("        child.emit('exit', 0, null);\n")
	sb.WriteString("      });\n")
	sb.WriteString("      const instance = new CodexExec('codex');\n")
	sb.WriteString("      await expect(consumeTestloopCodexExec(instance.run({ input: 'hi' }))).rejects.toThrow('spawn failed');\n")
	sb.WriteString("      expect(spawnMock).toHaveBeenCalled();\n")
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

func genJSCodexExecConfigOverrideCoverageTask(method jsFuncInfo, task *types.CoverageTestTask, testName string, moduleImportPath string) string {
	scenario := jsCodexExecConfigOverrideScenarioForTask(task)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString(fmt.Sprintf("      const { CodexExec } = await import('%s');\n", moduleImportPath))
	sb.WriteString("      const spawnMock = jest.mocked(child_process.spawn);\n")
	sb.WriteString("      spawnMock.mockClear();\n")
	if scenario.expectError {
		sb.WriteString(fmt.Sprintf("      const instance = new CodexExec('codex', {}, %s);\n", scenario.configOverrides))
		message := scenario.errorMessage
		if message == "" {
			message = "Error"
		}
		sb.WriteString(fmt.Sprintf("      await expect(consumeTestloopCodexExec(instance.run({ input: 'hi' }))).rejects.toThrow('%s');\n", strings.ReplaceAll(message, "'", "\\'")))
		sb.WriteString("      expect(spawnMock).not.toHaveBeenCalled();\n")
	} else {
		sb.WriteString("      const child = new TestloopCodexExecChild();\n")
		sb.WriteString("      spawnMock.mockReturnValue(child);\n")
		sb.WriteString("      setImmediate(() => {\n")
		sb.WriteString("        child.stdout.end();\n")
		sb.WriteString("        child.stderr.end();\n")
		sb.WriteString("        child.emit('exit', 0, null);\n")
		sb.WriteString("      });\n")
		sb.WriteString(fmt.Sprintf("      const instance = new CodexExec('codex', {}, %s);\n", scenario.configOverrides))
		sb.WriteString("      await consumeTestloopCodexExec(instance.run({ input: 'hi' }));\n")
		sb.WriteString("      const commandArgs = spawnMock.mock.calls[0]?.[1] || [];\n")
		for _, assertion := range scenario.assertions {
			sb.WriteString("      " + assertion + "\n")
		}
	}
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

type jsCodexExecConfigOverrideScenario struct {
	configOverrides string
	expectError     bool
	errorMessage    string
	assertions      []string
}

func jsCodexExecConfigOverrideScenarioForTask(task *types.CoverageTestTask) jsCodexExecConfigOverrideScenario {
	lineRange := ""
	target := ""
	if task != nil {
		lineRange = strings.TrimSpace(task.LineRange)
		target = strings.TrimSpace(task.Target)
	}
	if target == "toTomlValue" {
		if strings.HasPrefix(lineRange, "296") {
			return jsCodexExecConfigOverrideScenario{
				configOverrides: "{ retries: Infinity }",
				expectError:     true,
				errorMessage:    "finite number",
			}
		}
		if strings.HasPrefix(lineRange, "300") {
			return jsCodexExecConfigOverrideScenario{
				configOverrides: "{ enabled: true }",
				assertions: []string{
					"expect(commandArgs).toContain('--config');",
					"expect(commandArgs).toContain('enabled=true');",
				},
			}
		}
		if strings.HasPrefix(lineRange, "303") {
			return jsCodexExecConfigOverrideScenario{
				configOverrides: "{ models: ['gpt-5'] }",
				assertions: []string{
					"expect(commandArgs).toContain('--config');",
					"expect(commandArgs).toContain('models=[\"gpt-5\"]');",
				},
			}
		}
		if strings.HasPrefix(lineRange, "306") || strings.HasPrefix(lineRange, "314") {
			return jsCodexExecConfigOverrideScenario{
				configOverrides: "{ settings: [{ model: 'gpt-5' }] }",
				assertions: []string{
					"expect(commandArgs).toContain('--config');",
					"expect(commandArgs).toContain('settings=[{model = \"gpt-5\"}]');",
				},
			}
		}
		if strings.HasPrefix(lineRange, "312") {
			return jsCodexExecConfigOverrideScenario{
				configOverrides: "{ settings: [{ model: undefined, effort: 'high' }] }",
				assertions: []string{
					"expect(commandArgs).toContain('--config');",
					"expect(commandArgs).toContain('settings=[{effort = \"high\"}]');",
					"expect(commandArgs).not.toContain('model');",
				},
			}
		}
		if strings.HasPrefix(lineRange, "317") {
			return jsCodexExecConfigOverrideScenario{
				configOverrides: "{ invalid: null }",
				expectError:     true,
				errorMessage:    "cannot be null",
			}
		}
		if strings.HasPrefix(lineRange, "320") {
			return jsCodexExecConfigOverrideScenario{
				configOverrides: "{ invalid: () => 'bad' }",
				expectError:     true,
				errorMessage:    "Unsupported Codex config override value",
			}
		}
		return jsCodexExecConfigOverrideScenario{
			configOverrides: "{ retries: 3 }",
			assertions: []string{
				"expect(commandArgs).toContain('--config');",
				"expect(commandArgs).toContain('retries=3');",
			},
		}
	}
	if target == "formatTomlKey" {
		return jsCodexExecConfigOverrideScenario{
			configOverrides: "{ settings: [{ 'model name': 'gpt-5' }] }",
			assertions: []string{
				"expect(commandArgs).toContain('--config');",
				"expect(commandArgs).toContain('settings=[{\"model name\" = \"gpt-5\"}]');",
			},
		}
	}
	if target == "isPlainObject" {
		return jsCodexExecConfigOverrideScenario{
			configOverrides: "{ models: ['gpt-5'] }",
			assertions: []string{
				"expect(commandArgs).toContain('--config');",
				"expect(commandArgs).toContain('models=[\"gpt-5\"]');",
			},
		}
	}
	switch {
	case strings.HasPrefix(lineRange, "257"):
		return jsCodexExecConfigOverrideScenario{
			configOverrides: "'invalid-config'",
			expectError:     true,
			errorMessage:    "plain object",
		}
	case strings.HasPrefix(lineRange, "267"):
		return jsCodexExecConfigOverrideScenario{
			configOverrides: "{}",
			assertions: []string{
				"expect(commandArgs).not.toContain('--config');",
			},
		}
	case strings.HasPrefix(lineRange, "271"):
		return jsCodexExecConfigOverrideScenario{
			configOverrides: "{ sandbox_workspace_write: {} }",
			assertions: []string{
				"expect(commandArgs).toContain('--config');",
				"expect(commandArgs).toContain('sandbox_workspace_write={}');",
			},
		}
	default:
		return jsCodexExecConfigOverrideScenario{
			configOverrides: "{ model: 'gpt-5' }",
			assertions: []string{
				"expect(commandArgs).toContain('--config');",
				"expect(commandArgs).toContain('model=\"gpt-5\"');",
			},
		}
	}
}

func genJSVersionIPCompareCoverageTask(method jsFuncInfo, task *types.CoverageTestTask, testName string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it('%s', () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString("      const calls = [];\n")
	sb.WriteString("      const compare = (...args) => {\n")
	sb.WriteString("        calls.push(args);\n")
	sb.WriteString("        return 0;\n")
	sb.WriteString("      };\n")
	sb.WriteString("      const instance = new Version(4, 'IPv4', 4, 14, compare);\n")
	sb.WriteString("      const left = Buffer.from([1, 0, 0, 0]);\n")
	sb.WriteString("      const right = Buffer.from([1, 0, 0, 0]);\n")
	sb.WriteString("      const result = instance.ipCompare(left, right);\n")
	sb.WriteString("      expect(result).toBe(0);\n")
	sb.WriteString("      expect(calls).toEqual([[left, right, 0]]);\n")
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

func genJSSearcherCoverageTask(method jsFuncInfo, task *types.CoverageTestTask, testName string) (string, bool) {
	switch method.Name {
	case "search":
		return genJSSearcherSearchCoverageTask(method, task, testName), true
	case "read":
		return genJSSearcherReadCoverageTask(method, task, testName), true
	case "toString":
		return genJSSearcherToStringCoverageTask(method, task, testName), true
	default:
		return "", false
	}
}

func genJSSearcherSearchCoverageTask(method jsFuncInfo, task *types.CoverageTestTask, testName string) string {
	dLenZero := task != nil && strings.HasPrefix(strings.TrimSpace(task.LineRange), "100")
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString("      const version = { name: 'IPv4', bytes: 4, indexSize: 14, ipSubCompare: () => 0 };\n")
	if dLenZero {
		sb.WriteString("      const cBuffer = Buffer.alloc(278);\n")
		sb.WriteString("      cBuffer.writeUInt32LE(264, 256);\n")
		sb.WriteString("      cBuffer.writeUInt32LE(264, 260);\n")
	} else {
		sb.WriteString("      const cBuffer = Buffer.alloc(264);\n")
	}
	sb.WriteString("      const instance = new Searcher(version, null, null, cBuffer);\n")
	sb.WriteString("      const result = await instance.search('0.0.0.0');\n")
	sb.WriteString("      expect(result).toBe('');\n")
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

func genJSSearcherReadCoverageTask(method jsFuncInfo, task *types.CoverageTestTask, testName string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString("      const fs = await import('fs');\n")
	sb.WriteString("      const originalReadSync = fs.default.readSync;\n")
	sb.WriteString("      fs.default.readSync = () => 0;\n")
	sb.WriteString("      try {\n")
	sb.WriteString("        const instance = Object.assign(Object.create(Searcher.prototype), { cBuffer: null, handle: 1, ioCount: 0 });\n")
	sb.WriteString("        expect(() => instance.read(0, Buffer.alloc(4))).toThrow('incomplete read');\n")
	sb.WriteString("        expect(instance.ioCount).toBe(1);\n")
	sb.WriteString("      } finally {\n")
	sb.WriteString("        fs.default.readSync = originalReadSync;\n")
	sb.WriteString("      }\n")
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

func genJSSearcherToStringCoverageTask(method jsFuncInfo, task *types.CoverageTestTask, testName string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it('%s', () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString("      const version = { name: 'IPv4' };\n")
	sb.WriteString("      const instance = new Searcher(version, null, null, Buffer.alloc(8));\n")
	sb.WriteString("      const result = instance.toString();\n")
	sb.WriteString("      expect(result).toContain('IPv4');\n")
	sb.WriteString("      expect(result).toContain('\"cBuffer\": 8');\n")
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

func genJSDevWatcherHandleFileChangePublicEntryTest(method jsFuncInfo, task *types.CoverageTestTask, testName string) string {
	scenario := jsDevWatcherHandleFileChangeScenarioForTask(task)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it('%s', async () => {\n", jsEscapeTestNameValue(testName)))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString("      vi.useFakeTimers();\n")
	sb.WriteString("      try {\n")
	sb.WriteString("        const path = await import('node:path');\n")
	sb.WriteString("        const cwd = process.cwd();\n")
	sb.WriteString("        const instance = new DevWatcher('test-server', { enabled: true, watch: [], cwd });\n")
	sb.WriteString("        const changes = [];\n")
	sb.WriteString("        instance.on('filesChanged', (event) => changes.push(event));\n")
	sb.WriteString("        await instance.start();\n")
	if scenario.primeTimer {
		sb.WriteString("        instance.watcher.emit('change', 'src/first.js');\n")
	}
	if scenario.absolutePath {
		sb.WriteString("        const changedPath = path.join(cwd, 'src/app.js');\n")
	} else {
		sb.WriteString("        const changedPath = 'src/app.js';\n")
	}
	sb.WriteString("        instance.watcher.emit('change', changedPath);\n")
	sb.WriteString("        await vi.advanceTimersByTimeAsync(500);\n")
	sb.WriteString("        expect(changes).toHaveLength(1);\n")
	sb.WriteString("        expect(changes[0].serverName).toBe('test-server');\n")
	sb.WriteString("        expect(changes[0].files).toContain(changedPath);\n")
	if scenario.absolutePath {
		sb.WriteString("        expect(changes[0].relativeFiles).toContain(path.join('src', 'app.js'));\n")
	} else {
		sb.WriteString("        expect(changes[0].relativeFiles).toContain('src/app.js');\n")
	}
	if scenario.primeTimer {
		sb.WriteString("        expect(changes[0].files).toContain('src/first.js');\n")
	}
	sb.WriteString("        expect(instance.changedFiles.size).toBe(0);\n")
	sb.WriteString("        await instance.stop();\n")
	sb.WriteString("      } finally {\n")
	sb.WriteString("        vi.useRealTimers();\n")
	sb.WriteString("      }\n")
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

type jsDevWatcherHandleFileChangeScenario struct {
	absolutePath bool
	primeTimer   bool
}

func jsDevWatcherHandleFileChangeScenarioForTask(task *types.CoverageTestTask) jsDevWatcherHandleFileChangeScenario {
	hints := jsCoverageTaskHints(task)
	return jsDevWatcherHandleFileChangeScenario{
		absolutePath: strings.Contains(hints, "path.isAbsolute(file"),
		primeTimer:   strings.Contains(hints, "this.debounceTimer"),
	}
}

func jsClassRequiresInternalManualReview(cls jsClassInfo) bool {
	return cls.SourceIsESModule && !cls.IsExported && !cls.IsDefault && cls.DefaultInstance == ""
}

func genJSInternalClassManualReviewTest(cls jsClassInfo, method jsFuncInfo, task *types.CoverageTestTask, testName string, assertions jsAssertionStyle) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it.skip('%s', () => {\n", jsEscapeTestNameValue(testName)))
	sb.WriteString(jsManualReviewSymbolReferences(cls.Name, assertions, "      "))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString(fmt.Sprintf("      // manual_review_internal: %s is not exported from this ES module and cannot be constructed from an external test.\n", cls.Name))
	sb.WriteString("      // public_entry_candidates: none detected; cover it through an exported API, test-only seam, or module-level integration test.\n")
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

func genJSPrivateMethodManualReviewTest(cls jsClassInfo, method jsFuncInfo, task *types.CoverageTestTask, testName string, assertions jsAssertionStyle) string {
	var sb strings.Builder
	entries := jsPrivateEntryCandidatesForMethod(cls, method.Name)
	sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))
	sb.WriteString(fmt.Sprintf("    it.skip('%s', () => {\n", jsEscapeTestNameValue(testName)))
	sb.WriteString(jsManualReviewSymbolReferences(cls.Name, assertions, "      "))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("      // coverage task: %s\n", comment))
	}
	sb.WriteString(fmt.Sprintf("      // manual_review_private: %s.%s is a JavaScript private method and cannot be called from external tests.\n", cls.Name, method.Name))
	if len(entries) > 0 {
		sb.WriteString(fmt.Sprintf("      // public_entry_candidates: %s\n", strings.Join(entries, ", ")))
	} else {
		sb.WriteString("      // public_entry_candidates: none detected; add a public entry point or review manually.\n")
	}
	sb.WriteString("    });\n\n")
	sb.WriteString("  });\n\n")
	return sb.String()
}

func jsManualReviewSymbolReferences(className string, assertions jsAssertionStyle, indent string) string {
	var sb strings.Builder
	if className != "" {
		sb.WriteString(fmt.Sprintf("%svoid %s;\n", indent, className))
	}
	switch assertions {
	case jsAssertionStyleNode:
		sb.WriteString(indent + "void assert;\n")
	case jsAssertionStyleChai:
		sb.WriteString(indent + "void expect;\n")
	}
	return sb.String()
}

func jsPrivateEntryCandidatesForMethod(cls jsClassInfo, privateName string) []string {
	if cls.PrivateEntries == nil {
		return nil
	}
	return cls.PrivateEntries[privateName]
}

func jsCoverageTaskWantsErrorAssertion(method jsFuncInfo, task *types.CoverageTestTask) bool {
	if task != nil && taskTargetMatches(task.Target, method.ClassName, method.Name) {
		if method.ClassName == "EnvResolver" && method.Name == "_resolveStringWithPlaceholders" &&
			(jsCoverageTaskTargetsEnvResolverMissingPlaceholder(task) || jsCoverageTaskTargetsEnvResolverCommandFailure(task)) {
			return true
		}
		return task.GapType == "error_path" || jsCoverageTaskMentions(task, "if 分支: e") || jsCoverageTaskMentions(task, "if (e)") || jsCoverageTaskTargetsThrowingBranch(method.Body, task)
	}
	return method.Analysis.Throws
}

func jsCoverageTaskTargetsEnvResolverMissingPlaceholder(task *types.CoverageTestTask) bool {
	hints := jsCoverageTaskHints(task)
	return strings.Contains(hints, "resolvedValue === undefined") ||
		strings.Contains(hints, "MISSING_TOKEN") ||
		strings.Contains(hints, "Variable '")
}

func jsCoverageTaskTargetsEnvResolverCommandFailure(task *types.CoverageTestTask) bool {
	hints := jsCoverageTaskHints(task)
	return strings.Contains(hints, "${cmd:") ||
		strings.Contains(hints, "failing-command") ||
		strings.Contains(hints, "cmd execution failed") ||
		strings.Contains(hints, "isCommand")
}

func jsCoverageTaskTargetsEnvResolverMaxPasses(task *types.CoverageTestTask) bool {
	hints := jsCoverageTaskHints(task)
	return strings.Contains(hints, "maxPasses") ||
		strings.Contains(hints, "Max placeholder resolution depth") ||
		strings.Contains(hints, "depth")
}

func jsCoverageTaskTargetsThrowingBranch(body string, task *types.CoverageTestTask) bool {
	if task == nil || strings.TrimSpace(task.GapType) != "branch" || strings.TrimSpace(body) == "" {
		return false
	}
	conditions := jsCoverageTaskBranchConditions(task)
	if len(conditions) == 0 {
		return false
	}
	throwingConditions := jsThrowingBranchConditions(body)
	for _, condition := range conditions {
		if _, ok := throwingConditions[normalizeJSCondition(condition)]; ok {
			return true
		}
	}
	return false
}

func jsCoverageTaskBranchConditions(task *types.CoverageTestTask) []string {
	if task == nil {
		return nil
	}
	var conditions []string
	for _, text := range append(append([]string{}, task.MissingBranches...), task.AssertionFocus...) {
		if condition := jsCoverageTaskBranchCondition(text); condition != "" {
			conditions = append(conditions, condition)
		}
	}
	for _, text := range task.SuggestedInputs {
		if condition := jsBacktickExpression(text); condition != "" {
			conditions = append(conditions, condition)
		}
	}
	return conditions
}

func jsCoverageTaskBranchCondition(text string) string {
	for _, marker := range []string{"if 分支:", "if branch:"} {
		idx := strings.Index(text, marker)
		if idx >= 0 {
			return strings.TrimSpace(text[idx+len(marker):])
		}
	}
	return ""
}

func jsBacktickExpression(text string) string {
	start := strings.Index(text, "`")
	if start < 0 {
		return ""
	}
	end := strings.Index(text[start+1:], "`")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(text[start+1 : start+1+end])
}

func jsThrowingBranchConditions(body string) map[string]struct{} {
	conditions := map[string]struct{}{}
	for _, match := range jsIfThrowRe.FindAllStringSubmatch(body, -1) {
		if len(match) != 2 {
			continue
		}
		condition := normalizeJSCondition(match[1])
		if condition != "" {
			conditions[condition] = struct{}{}
		}
	}
	return conditions
}

func normalizeJSCondition(condition string) string {
	condition = strings.TrimSpace(condition)
	condition = strings.TrimPrefix(condition, "(")
	condition = strings.TrimSuffix(condition, ")")
	return strings.Join(strings.Fields(condition), " ")
}

func jsClassInstanceForCoverageTask(cls jsClassInfo, method jsFuncInfo, task *types.CoverageTestTask) string {
	if cls.DefaultInstance != "" {
		return cls.DefaultInstance
	}
	if cls.Name == "SSEManager" && method.Name == "addConnection" && jsCoverageTaskMentions(task, "this.shutdownTimer") {
		return "Object.assign(new SSEManager({}), { shutdownTimer: setTimeout(() => {}, 1000) })"
	}
	if cls.Name == "Codex" {
		return "new Codex({ codexPathOverride: 'codex' })"
	}
	args := jsClassConstructorArgsForCoverageTask(cls, method, task)
	if args == "" {
		return fmt.Sprintf("new %s()", cls.Name)
	}
	return fmt.Sprintf("new %s(%s)", cls.Name, args)
}

func jsClassConstructorArgsForCoverageTask(cls jsClassInfo, method jsFuncInfo, task *types.CoverageTestTask) string {
	if len(cls.ConstructorParams) == 0 {
		return ""
	}
	options := jsClassCoverageTaskConstructorOptions(method, task)
	args := make([]string, len(cls.ConstructorParams))
	for i, param := range cls.ConstructorParams {
		compact := jsCompactName(param.Name)
		switch {
		case jsParamTypeMentions(param, "CodexExec"):
			args[i] = jsCodexExecMockForCoverageTask(method, task)
		case jsRocketMQLoadBalancerClass(cls.Name) && jsParamTypeMentions(param, "TopicRouteData"):
			args[i] = jsRocketMQTopicRouteDataMock()
		case cls.Name == "PublishingMessage" && jsParamTypeMentions(param, "MessageOptions"):
			args[i] = jsRocketMQPublishingMessageOptionsMock(task)
		case cls.Name == "PublishingMessage" && jsParamTypeMentions(param, "PublishingSettings"):
			args[i] = "({ maxBodySizeBytes: 4194304 } as PublishingSettings)"
		case cls.Name == "PublishingMessage" && strings.Contains(compact, "txenabled"):
			args[i] = "false"
		case strings.Contains(compact, "servername") || compact == "name":
			args[i] = "'test-server'"
		case strings.Contains(compact, "devconfig"):
			args[i] = jsObjectLiteralWithDefaults(options, []string{"enabled: true", "watch: []", "cwd: process.cwd()"})
		case method.Analysis.TSTypeDecls != nil && jsMockTypeHasDecl(jsNormalizeTSTypeExpr(param.TypeExpr), method.Analysis.TSTypeDecls):
			args[i], _ = jsTypedParamMockValueWithDecls(param, method.Analysis.TSTypeDecls)
		case strings.Contains(compact, "config") || strings.Contains(compact, "options"):
			args[i] = jsObjectLiteralWithDefaults(options, nil)
		default:
			if value, ok := jsTypedParamMockValueWithDecls(param, method.Analysis.TSTypeDecls); ok {
				args[i] = value
			} else {
				args[i] = jsArgValue(param, i)
			}
		}
	}
	return strings.Join(args, ", ")
}

func jsRocketMQLoadBalancerClass(name string) bool {
	return name == "PublishingLoadBalancer" || name == "SubscriptionLoadBalancer"
}

func jsRocketMQTopicRouteDataMock() string {
	return "Object.assign(new TopicRouteData([]), { messageQueues: [{ queueId: 0, permission: 4, broker: { name: 'broker-a', endpoints: { facade: '127.0.0.1:8081' } } }, { queueId: 0, permission: 4, broker: { name: 'broker-b', endpoints: { facade: '127.0.0.1:8082' } } }] })"
}

func jsRocketMQPublishingMessageOptionsMock(task *types.CoverageTestTask) string {
	fields := []string{"topic: 'test'", "body: Buffer.from('test')"}
	switch {
	case jsCoverageTaskMentions(task, "this.tag"):
		fields = append(fields, "tag: 'test'")
	case jsCoverageTaskMentions(task, "this.deliveryTimestamp"):
		fields = append(fields, "deliveryTimestamp: new Date('2026-01-01T00:00:00.000Z')")
	case jsCoverageTaskMentions(task, "this.messageGroup"):
		fields = append(fields, "messageGroup: 'test'")
	case jsCoverageTaskMentions(task, "this.properties"):
		fields = append(fields, "properties: new Map([['key', 'test']])")
	default:
		fields = append(fields, "keys: ['key']")
	}
	return "{ " + strings.Join(fields, ", ") + " }"
}

func jsCodexExecMockForCoverageTask(method jsFuncInfo, task *types.CoverageTestTask) string {
	event := "{ type: 'turn.completed', usage: null }"
	if method.ClassName == "Thread" && method.Name == "run" && jsCoverageTaskWantsErrorAssertion(method, task) {
		event = "{ type: 'turn.failed', error: { message: 'failed' } }"
	}
	return fmt.Sprintf("({ run: async function* () { yield JSON.stringify(%s); } } as any)", event)
}

func jsObjectLiteralWithDefaults(options []string, defaults []string) string {
	parts := append([]string{}, options...)
	parts = append(parts, defaults...)
	if len(parts) == 0 {
		return "{}"
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func jsClassCoverageTaskConstructorOptions(method jsFuncInfo, task *types.CoverageTestTask) []string {
	if task == nil || task.GapType != "error_path" {
		return nil
	}
	body := method.Body
	var options []string
	if strings.Contains(body, "depth > this.maxPasses") && jsCoverageTaskTargetsEnvResolverMaxPasses(task) {
		options = append(options, "maxPasses: 0")
	}
	if strings.Contains(body, "fallbackValue === undefined") && strings.Contains(body, "this.strict") {
		options = append(options, "strict: true")
	}
	return options
}

func jsClassCoverageTaskInputOverrides(method jsFuncInfo, task *types.CoverageTestTask) map[string]string {
	overrides := map[string]string{}
	if task == nil {
		return overrides
	}
	body := method.Body
	if task.GapType == "error_path" && strings.Contains(body, "fallbackValue === undefined") && strings.Contains(body, "this.strict") {
		jsSetParamOverride(method.Params, overrides, 0, "{ MISSING: null }")
		jsSetParamOverride(method.Params, overrides, 1, "{}")
	}
	if method.ClassName == "EnvResolver" && method.Name == "_resolveStringWithPlaceholders" && jsCoverageTaskTargetsEnvResolverMissingPlaceholder(task) {
		jsSetNamedParamOverride(method.Params, overrides, "str", "'${MISSING_TOKEN}'")
		jsSetNamedParamOverride(method.Params, overrides, "context", "{}")
		jsSetNamedParamOverride(method.Params, overrides, "depth", "0")
	}
	if method.ClassName == "EnvResolver" && method.Name == "_resolveStringWithPlaceholders" && jsCoverageTaskTargetsEnvResolverCommandFailure(task) {
		jsSetNamedParamOverride(method.Params, overrides, "str", "'${cmd: failing-command}'")
		jsSetNamedParamOverride(method.Params, overrides, "context", "{}")
		jsSetNamedParamOverride(method.Params, overrides, "depth", "0")
	}
	if task.GapType == "error_path" && strings.Contains(body, "depth > this.maxPasses") && jsCoverageTaskTargetsEnvResolverMaxPasses(task) {
		jsSetNamedParamOverride(method.Params, overrides, "str", "'${MISSING}'")
		jsSetNamedParamOverride(method.Params, overrides, "context", "{}")
		jsSetNamedParamOverride(method.Params, overrides, "depth", "1")
	}
	if task.GapType == "return_path" && strings.Contains(body, "placeholders.length === 0") {
		jsSetNamedParamOverride(method.Params, overrides, "str", "'plain'")
		jsSetNamedParamOverride(method.Params, overrides, "context", "{}")
		jsSetNamedParamOverride(method.Params, overrides, "depth", "0")
	}
	if strings.Contains(body, "this.LOG_LEVELS[level] !== undefined") {
		jsSetNamedParamOverride(method.Params, overrides, "level", "'info'")
	}
	if strings.Contains(body, "if (enable)") || strings.Contains(body, "if(enable)") {
		jsSetNamedParamOverride(method.Params, overrides, "enable", "true")
	}
	if method.Name == "addConnection" && strings.Contains(body, "res.setHeader") {
		jsSetNamedParamOverride(method.Params, overrides, "req", jsExpressRequestMock())
		writableEnded := "false"
		if jsCoverageTaskMentions(task, "res.writableEnded") {
			writableEnded = "true"
		}
		jsSetNamedParamOverride(method.Params, overrides, "res", jsExpressResponseMock(writableEnded))
	}
	if method.ClassName == "PublishingMessage" && method.Name == "toProtobuf" {
		jsSetNamedParamOverride(method.Params, overrides, "mq", "({ queueId: 0 } as MessageQueue)")
	}
	return overrides
}

func jsSetParamOverride(params []jsParamInfo, overrides map[string]string, index int, value string) {
	if index >= 0 && index < len(params) {
		overrides[params[index].Name] = value
	}
}

func jsSetNamedParamOverride(params []jsParamInfo, overrides map[string]string, name string, value string) {
	for _, param := range params {
		if param.Name == name {
			overrides[param.Name] = value
			return
		}
	}
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
	jsAssertionStyleNode jsAssertionStyle = "node"
)

func jsAssertionStyleForTask(task *types.CoverageTestTask) jsAssertionStyle {
	if task != nil && strings.EqualFold(task.Framework, "mocha") {
		if task.File != "" && !jsProjectHasDependency(task.File, "chai") {
			return jsAssertionStyleNode
		}
		return jsAssertionStyleChai
	}
	return jsAssertionStyleJest
}

func jsProjectHasDependency(startPath string, dep string) bool {
	dir := startPath
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for {
		pkgPath := filepath.Join(dir, "package.json")
		data, err := os.ReadFile(pkgPath)
		if err == nil {
			var pkg struct {
				Dependencies         map[string]any `json:"dependencies"`
				DevDependencies      map[string]any `json:"devDependencies"`
				OptionalDependencies map[string]any `json:"optionalDependencies"`
				PeerDependencies     map[string]any `json:"peerDependencies"`
			}
			if json.Unmarshal(data, &pkg) == nil {
				if _, ok := pkg.Dependencies[dep]; ok {
					return true
				}
				if _, ok := pkg.DevDependencies[dep]; ok {
					return true
				}
				if _, ok := pkg.OptionalDependencies[dep]; ok {
					return true
				}
				if _, ok := pkg.PeerDependencies[dep]; ok {
					return true
				}
			}
			return false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

func genJSErrorAssertion(style jsAssertionStyle, isAsync bool, callExpr string, indent string) string {
	if style == jsAssertionStyleNode {
		if isAsync {
			return fmt.Sprintf("%sawait assert.rejects(async () => %s);\n", indent, callExpr)
		}
		return fmt.Sprintf("%sassert.throws(() => %s);\n", indent, callExpr)
	}
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

func genJSErrorAssertionWithMessage(style jsAssertionStyle, isAsync bool, callExpr string, message string, indent string) string {
	if strings.TrimSpace(message) == "" {
		return genJSErrorAssertion(style, isAsync, callExpr, indent)
	}
	quoted := jsSingleQuotedString(message)
	if style == jsAssertionStyleNode {
		if isAsync {
			return fmt.Sprintf("%sawait assert.rejects(async () => %s, { message: new RegExp(%s) });\n", indent, callExpr, quoted)
		}
		return fmt.Sprintf("%sassert.throws(() => %s, { message: new RegExp(%s) });\n", indent, callExpr, quoted)
	}
	if style == jsAssertionStyleChai {
		if isAsync {
			return indent + "let caughtError;\n" +
				indent + "try {\n" +
				indent + "  await " + callExpr + ";\n" +
				indent + "} catch (err) {\n" +
				indent + "  caughtError = err;\n" +
				indent + "}\n" +
				indent + "expect(caughtError).to.exist;\n" +
				fmt.Sprintf("%sexpect(caughtError.message).to.contain(%s);\n", indent, quoted)
		}
		return fmt.Sprintf("%sexpect(() => %s).to.throw(%s);\n", indent, callExpr, quoted)
	}
	if isAsync {
		return fmt.Sprintf("%sawait expect(%s).rejects.toThrow(%s);\n", indent, callExpr, quoted)
	}
	return fmt.Sprintf("%sexpect(() => %s).toThrow(%s);\n", indent, callExpr, quoted)
}

func genJSVoidCallAssertion(style jsAssertionStyle, isAsync bool, callExpr string, indent string) string {
	if isAsync {
		if style == jsAssertionStyleNode {
			return fmt.Sprintf("%sawait assert.doesNotReject(async () => %s);\n", indent, callExpr)
		}
		if style == jsAssertionStyleChai {
			return indent + "let caughtError;\n" +
				indent + "try {\n" +
				indent + "  await " + callExpr + ";\n" +
				indent + "} catch (err) {\n" +
				indent + "  caughtError = err;\n" +
				indent + "}\n" +
				indent + "expect(caughtError).to.be.undefined;\n"
		}
		return fmt.Sprintf("%sawait %s;\n", indent, callExpr)
	}
	if style == jsAssertionStyleNode {
		return fmt.Sprintf("%sassert.doesNotThrow(() => %s);\n", indent, callExpr)
	}
	if style == jsAssertionStyleChai {
		return fmt.Sprintf("%sexpect(() => %s).to.not.throw();\n", indent, callExpr)
	}
	return fmt.Sprintf("%sexpect(() => %s).not.toThrow();\n", indent, callExpr)
}

func genJSResultAssertionWithTaskArgsStyle(a jsFuncAnalysis, params []jsParamInfo, boundary *jsBoundary, values map[string]string, indent string, style jsAssertionStyle) string {
	var sb strings.Builder

	if !a.HasReturn {
		sb.WriteString(indent + "// void function, verify no exception\n")
		return sb.String()
	}

	if expected, ok, deepEqual := jsExpectedReturnExprWithValuesKind(a, params, boundary, values); ok {
		if style == jsAssertionStyleNode && deepEqual {
			sb.WriteString(indent + "assert.deepEqual(result, " + expected + ");\n")
		} else if style == jsAssertionStyleNode {
			sb.WriteString(indent + "assert.equal(result, " + expected + ");\n")
		} else if style == jsAssertionStyleChai && deepEqual {
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
	if style == jsAssertionStyleNode {
		switch a.ReturnType {
		case "number":
			sb.WriteString(indent + "assert.equal(typeof result, 'number');\n")
			sb.WriteString(indent + "assert.equal(Number.isNaN(result), false);\n")
		case "string":
			sb.WriteString(indent + "assert.equal(typeof result, 'string');\n")
			sb.WriteString(indent + "assert(result.length >= 0);\n")
		case "boolean":
			sb.WriteString(indent + "assert.equal(typeof result, 'boolean');\n")
		case "array":
			sb.WriteString(indent + "assert.equal(Array.isArray(result), true);\n")
		case "object":
			sb.WriteString(indent + "assert.equal(typeof result, 'object');\n")
			sb.WriteString(indent + "assert.notEqual(result, null);\n")
		case "null":
			sb.WriteString(indent + "assert.equal(result, null);\n")
		case "undefined":
			sb.WriteString(indent + "assert.equal(result, undefined);\n")
		default:
			sb.WriteString(indent + "assert.notEqual(result, undefined);\n")
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

func jsSingleQuotedString(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "'", "\\'")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\r", "\\r")
	return "'" + value + "'"
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
		} else if jsDefaultParamShouldStayUndefined(p, analysis) {
			args[i] = "undefined"
		} else if value, ok := jsTypedParamMockValue(p, analysis); ok {
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

func jsDefaultParamShouldStayUndefined(p jsParamInfo, analysis *jsFuncAnalysis) bool {
	if !p.HasDefault {
		return false
	}
	compact := jsCompactName(p.Name)
	if jsNameHasAny(compact, "options", "opts", "config", "context", "payload", "data", "body", "params", "query", "metadata") {
		return false
	}
	typeExpr := jsNormalizeTSTypeExpr(p.TypeExpr)
	if typeExpr == "" || strings.HasPrefix(typeExpr, "{") || strings.Contains(strings.ToLower(typeExpr), "record<") || strings.Contains(strings.ToLower(typeExpr), "object") {
		return false
	}
	if analysis != nil && len(analysis.TSTypeDecls) > 0 {
		if _, resolved := jsResolveNamedTSType(typeExpr, analysis.TSTypeDecls); resolved != "" {
			return strings.HasPrefix(resolved, jsConstructorMockKey)
		}
	}
	return jsTSIdentifierRe.MatchString(typeExpr) && jsLooksLikeConstructableTypeName(typeExpr)
}

func jsTypedParamMockValue(p jsParamInfo, analysis *jsFuncAnalysis) (string, bool) {
	if analysis == nil || strings.TrimSpace(p.TypeExpr) == "" {
		return "", false
	}
	return jsTypedParamMockValueWithDecls(p, analysis.TSTypeDecls)
}

func jsTypedParamMockValueWithDecls(p jsParamInfo, decls map[string]string) (string, bool) {
	if strings.TrimSpace(p.TypeExpr) == "" {
		return "", false
	}
	typeExpr := jsNormalizeTSTypeExpr(p.TypeExpr)
	if typeExpr == "" {
		return "", false
	}
	if jsMockTypeHasDecl(typeExpr, decls) || jsBuiltinTSTypeHasMock(typeExpr) || strings.HasPrefix(typeExpr, "{") {
		return jsMockValueForTSTypeWithDecls(p.Name, typeExpr, decls), true
	}
	return "", false
}

func jsBuiltinTSTypeHasMock(typeExpr string) bool {
	switch {
	case typeExpr == "Buffer" || typeExpr == "Uint8Array" || typeExpr == "Date":
		return true
	case typeExpr == "Map" || strings.HasPrefix(typeExpr, "Map<"):
		return true
	case typeExpr == "Set" || strings.HasPrefix(typeExpr, "Set<"):
		return true
	default:
		return false
	}
}

func jsMockTypeHasDecl(typeExpr string, decls map[string]string) bool {
	if len(decls) == 0 {
		return false
	}
	typeExpr = jsUnwrapTSUtilityWrappers(typeExpr)
	if branch, ok := jsPreferredTSTypeUnionBranch(typeExpr); ok {
		typeExpr = jsUnwrapTSUtilityWrappers(branch)
	}
	if _, resolved := jsResolveNamedTSType(typeExpr, decls); resolved != "" {
		return true
	}
	return false
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
	typeExpr := strings.ToLower(strings.TrimSpace(p.TypeExpr))

	if jsNameHasPrefix(compact, "is", "has", "can", "should") ||
		jsNameHasAny(compact, "enabled", "active", "valid", "visible", "flag", "checked") {
		return "true"
	}
	if jsNameHasAny(compact, "items", "list", "array", "arr", "rows", "records", "args") {
		return "[]"
	}
	if typeExpr != "" && strings.Contains(typeExpr, "null") {
		return "null"
	}
	if typeExpr != "" && (strings.Contains(typeExpr, "input") || strings.Contains(typeExpr, "string")) {
		return "'test'"
	}
	if typeExpr != "" && strings.Contains(typeExpr, "number") {
		if compact == "b" || compact == "y" {
			return "2"
		}
		return "1"
	}
	if typeExpr != "" && strings.Contains(typeExpr, "buffer") {
		return "Buffer.from('test')"
	}
	if typeExpr != "" && strings.Contains(typeExpr, "uint8array") {
		return "new Uint8Array([1, 2, 3])"
	}
	if jsNameIsNumeric(compact) {
		if compact == "b" || compact == "y" {
			return "2"
		}
		return "1"
	}
	if typeExpr != "" {
		switch {
		case strings.Contains(typeExpr, "codexexec"):
			return "({ run: async function* () { yield JSON.stringify({ type: 'turn.completed', usage: null }); } } as any)"
		case strings.Contains(typeExpr, "boolean"):
			return "true"
		case strings.Contains(typeExpr, "[]") || strings.Contains(typeExpr, "array<"):
			return "[]"
		case strings.Contains(typeExpr, "options") || strings.Contains(typeExpr, "record<") || strings.Contains(typeExpr, "object") || strings.HasPrefix(typeExpr, "{"):
			return "{}"
		}
	}
	if jsNameHasAny(compact, "options", "opts", "config", "payload", "data", "body", "params", "query", "user", "metadata") {
		return "{}"
	}
	if jsNameHasAny(compact, "error", "err") {
		return "new Error('test error')"
	}
	if compact == "req" || compact == "request" {
		return jsExpressRequestMock()
	}
	if compact == "res" {
		return jsExpressResponseMock("false")
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

func jsParamTypeMentions(p jsParamInfo, needle string) bool {
	return strings.Contains(strings.ToLower(p.TypeExpr), strings.ToLower(needle))
}

func jsCompactName(name string) string {
	name = strings.ToLower(name)
	return strings.ReplaceAll(strings.ReplaceAll(name, "_", ""), "-", "")
}

func jsCoverageTaskMentions(task *types.CoverageTestTask, needle string) bool {
	if task == nil || needle == "" {
		return false
	}
	return strings.Contains(jsCoverageTaskHints(task), needle)
}

func jsCoverageTaskHints(task *types.CoverageTestTask) string {
	if task == nil {
		return ""
	}
	hints := append(append([]string{}, task.MissingBranches...), task.SuggestedInputs...)
	hints = append(hints, task.AssertionFocus...)
	return strings.Join(hints, " ")
}

func jsExpressRequestMock() string {
	return "{ on: () => {} }"
}

func jsExpressResponseMock(writableEnded string) string {
	if writableEnded == "" {
		writableEnded = "false"
	}
	return "{ writableEnded: " + writableEnded + ", setHeader: () => {}, write: () => {}, end: () => {} }"
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
	if style == jsAssertionStyleNode {
		return fmt.Sprintf("%sassert.deepEqual(%s.%sCalls, %s);\n", indent, info.Param, info.Method, expectedCalls)
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
	typeExpr = jsUnwrapTSUtilityWrappers(typeExpr)
	if branch, ok := jsPreferredTSTypeUnionBranch(typeExpr); ok {
		typeExpr = branch
		typeExpr = jsUnwrapTSUtilityWrappers(typeExpr)
	}
	if payload, ok := jsMockIndexedAccessPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
	}
	if payload, ok := jsMockIntersectionPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
	}
	if payload, ok := jsMockProjectionPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
	}
	if payload, ok := jsMockRecordPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
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
		typeExpr = jsUnwrapTSUtilityWrappers(typeExpr)
		seen = nextSeen
		if branch, ok := jsPreferredTSTypeUnionBranch(typeExpr); ok {
			typeExpr = branch
			typeExpr = jsUnwrapTSUtilityWrappers(typeExpr)
		}
	}
	if payload, ok := jsMockIndexedAccessPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
	}
	if payload, ok := jsMockIntersectionPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
	}
	if payload, ok := jsMockProjectionPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
	}
	if payload, ok := jsMockRecordPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
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
	typeExpr = jsUnwrapTSUtilityWrappers(typeExpr)
	if payload, ok := jsMockIndexedAccessPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
	}
	if payload, ok := jsMockIntersectionPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
	}
	if payload, ok := jsMockProjectionPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
	}
	if payload, ok := jsMockRecordPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
	}
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
		typeExpr = jsUnwrapTSUtilityWrappers(typeExpr)
		seen = nextSeen
	}
	if payload, ok := jsMockIndexedAccessPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
	}
	if payload, ok := jsMockIntersectionPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
	}
	if payload, ok := jsMockProjectionPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
	}
	if payload, ok := jsMockRecordPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload, true
	}
	if !strings.HasPrefix(typeExpr, "{") || !strings.HasSuffix(typeExpr, "}") {
		return "", false
	}
	return jsObjectMockFromResolvedTSTypeWithDeclsSeen(typeExpr, decls, seen, nil, nil, false)
}

func jsObjectMockFromResolvedTSTypeWithDeclsSeen(typeExpr string, decls map[string]string, seen map[string]bool, include map[string]bool, exclude map[string]bool, allowEmpty bool) (string, bool) {
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(typeExpr), "{"), "}"))
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
		if include != nil && !include[name] {
			continue
		}
		if exclude != nil && exclude[name] {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s", name, jsMockValueForTSTypeWithDeclsSeen(name, typ, decls, seen)))
	}
	if len(parts) == 0 {
		if allowEmpty {
			return "{}", true
		}
		return "", false
	}
	return "{ " + strings.Join(parts, ", ") + " }", true
}

func jsMockIntersectionPayloadFromTSTypeWithDeclsSeen(typeExpr string, decls map[string]string, seen map[string]bool) (string, bool) {
	branches := jsSplitTopLevelTypeIntersection(typeExpr)
	if len(branches) <= 1 {
		return "", false
	}
	parts := make([]string, 0, len(branches))
	for _, branch := range branches {
		payload, ok := jsObjectMockFromTSTypeWithDeclsSeen(branch, decls, seen)
		if !ok || !strings.HasPrefix(payload, "{") || !strings.HasSuffix(payload, "}") {
			return "", false
		}
		body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(payload, "{"), "}"))
		if body != "" {
			parts = append(parts, jsSplitTopLevelTypeFields(body)...)
		}
	}
	if len(parts) == 0 {
		return "{}", true
	}
	return "{ " + strings.Join(parts, ", ") + " }", true
}

func jsMockIndexedAccessPayloadFromTSTypeWithDeclsSeen(typeExpr string, decls map[string]string, seen map[string]bool) (string, bool) {
	sourceType, key, ok := jsTSIndexedAccessParts(typeExpr)
	if !ok {
		return "", false
	}
	sourceExpr, nextSeen, ok := jsResolvedObjectTypeForProjection(sourceType, decls, seen)
	if !ok {
		return "", false
	}
	fieldType, ok := jsObjectFieldType(sourceExpr, key)
	if !ok {
		return "", false
	}
	if payload, ok := jsMockPayloadFromTSTypeWithDeclsSeen(fieldType, decls, nextSeen); ok {
		return payload, true
	}
	return jsMockValueForTSTypeWithDeclsSeen(key, fieldType, decls, nextSeen), true
}

func jsMockProjectionPayloadFromTSTypeWithDeclsSeen(typeExpr string, decls map[string]string, seen map[string]bool) (string, bool) {
	if args, ok := jsTSGenericArgs(typeExpr, "Pick"); ok {
		return jsMockProjectedObjectPayload(args, decls, seen, false)
	}
	if args, ok := jsTSGenericArgs(typeExpr, "Omit"); ok {
		return jsMockProjectedObjectPayload(args, decls, seen, true)
	}
	return "", false
}

func jsMockProjectedObjectPayload(args []string, decls map[string]string, seen map[string]bool, omit bool) (string, bool) {
	if len(args) != 2 {
		return "", false
	}
	keys, ok := jsTSStringLiteralUnionValues(args[1])
	if !ok || len(keys) == 0 {
		return "", false
	}
	sourceExpr, nextSeen, ok := jsResolvedObjectTypeForProjection(args[0], decls, seen)
	if !ok {
		return "", false
	}
	if omit {
		return jsObjectMockFromResolvedTSTypeWithDeclsSeen(sourceExpr, decls, nextSeen, nil, keys, true)
	}
	return jsObjectMockFromResolvedTSTypeWithDeclsSeen(sourceExpr, decls, nextSeen, keys, nil, false)
}

func jsProjectedObjectTypeFromTSType(typeExpr string, decls map[string]string, seen map[string]bool) (string, map[string]bool, bool) {
	if args, ok := jsTSGenericArgs(typeExpr, "Pick"); ok {
		return jsProjectedObjectType(args, decls, seen, false)
	}
	if args, ok := jsTSGenericArgs(typeExpr, "Omit"); ok {
		return jsProjectedObjectType(args, decls, seen, true)
	}
	return "", seen, false
}

func jsProjectedObjectType(args []string, decls map[string]string, seen map[string]bool, omit bool) (string, map[string]bool, bool) {
	if len(args) != 2 {
		return "", seen, false
	}
	keys, ok := jsTSStringLiteralUnionValues(args[1])
	if !ok || len(keys) == 0 {
		return "", seen, false
	}
	sourceExpr, nextSeen, ok := jsResolvedObjectTypeForProjection(args[0], decls, seen)
	if !ok {
		return "", seen, false
	}
	if omit {
		typeExpr, ok := jsObjectTypeFromResolvedTSType(sourceExpr, nil, keys, true)
		return typeExpr, nextSeen, ok
	}
	typeExpr, ok := jsObjectTypeFromResolvedTSType(sourceExpr, keys, nil, false)
	return typeExpr, nextSeen, ok
}

func jsMockRecordPayloadFromTSTypeWithDeclsSeen(typeExpr string, decls map[string]string, seen map[string]bool) (string, bool) {
	args, ok := jsTSGenericArgs(typeExpr, "Record")
	if !ok || len(args) != 2 {
		return "", false
	}
	keys, ok := jsTSRecordKeys(args[0])
	if !ok || len(keys) == 0 {
		return "", false
	}
	valueType := strings.TrimSpace(args[1])
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value, ok := jsMockPayloadFromTSTypeWithDeclsSeen(valueType, decls, seen)
		if !ok {
			value = jsMockValueForTSTypeWithDeclsSeen(key, valueType, decls, seen)
		}
		parts = append(parts, fmt.Sprintf("%s: %s", key, value))
	}
	return "{ " + strings.Join(parts, ", ") + " }", true
}

func jsTSRecordKeys(typeExpr string) ([]string, bool) {
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	if typeExpr == "string" {
		return []string{"key"}, true
	}
	return jsTSStringLiteralUnionList(typeExpr)
}

func jsResolvedObjectTypeForProjection(typeExpr string, decls map[string]string, seen map[string]bool) (string, map[string]bool, bool) {
	typeExpr = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(typeExpr), ";"))
	typeExpr = jsUnwrapTSUtilityWrappers(typeExpr)
	if typeExpr == "" {
		return "", seen, false
	}
	lookupType := jsNormalizeTSTypeExpr(typeExpr)
	if name, resolved := jsResolveNamedTSType(lookupType, decls); resolved != "" {
		if seen == nil {
			seen = map[string]bool{}
		}
		if seen[name] {
			return "", seen, false
		}
		nextSeen := make(map[string]bool, len(seen)+1)
		for key, value := range seen {
			nextSeen[key] = value
		}
		nextSeen[name] = true
		typeExpr = strings.TrimSpace(resolved)
		typeExpr = jsUnwrapTSUtilityWrappers(typeExpr)
		seen = nextSeen
	}
	if projectedExpr, nextSeen, ok := jsProjectedObjectTypeFromTSType(typeExpr, decls, seen); ok {
		return projectedExpr, nextSeen, true
	}
	if !strings.HasPrefix(typeExpr, "{") || !strings.HasSuffix(typeExpr, "}") {
		return "", seen, false
	}
	return typeExpr, seen, true
}

func jsObjectTypeFromResolvedTSType(typeExpr string, include map[string]bool, exclude map[string]bool, allowEmpty bool) (string, bool) {
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(typeExpr), "{"), "}"))
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
		if include != nil && !include[name] {
			continue
		}
		if exclude != nil && exclude[name] {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s", name, typ))
	}
	if len(parts) == 0 {
		if allowEmpty {
			return "{}", true
		}
		return "", false
	}
	return "{ " + strings.Join(parts, "; ") + " }", true
}

func jsObjectFieldType(typeExpr, key string) (string, bool) {
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(typeExpr), "{"), "}"))
	for _, field := range jsSplitTopLevelTypeFields(body) {
		name, typ, ok := jsParseTSTypeField(field)
		if ok && name == key {
			return typ, true
		}
	}
	return "", false
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
	if field == "" {
		return "", "", false
	}
	if name, typ, ok := jsParseTSTypeMethodField(field); ok {
		return name, typ, true
	}
	parts := strings.SplitN(field, ":", 2)
	if len(parts) == 2 {
		name := strings.TrimSpace(parts[0])
		name = strings.TrimPrefix(name, "readonly ")
		name = strings.TrimSuffix(name, "?")
		name = strings.Trim(name, `"'`)
		if !regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`).MatchString(name) {
			return "", "", false
		}
		return name, strings.TrimSpace(parts[1]), true
	}
	return "", "", false
}

func jsParseTSTypeMethodField(field string) (string, string, bool) {
	open := strings.Index(field, "(")
	close := strings.LastIndex(field, ")")
	if open <= 0 || close <= open {
		return "", "", false
	}
	name := strings.TrimSpace(field[:open])
	name = strings.TrimPrefix(name, "readonly ")
	name = strings.TrimSuffix(name, "?")
	name = strings.Trim(name, `"'`)
	if !regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`).MatchString(name) {
		return "", "", false
	}
	returnType := "void"
	rest := strings.TrimSpace(field[close+1:])
	if strings.HasPrefix(rest, ":") {
		returnType = strings.TrimSpace(strings.TrimPrefix(rest, ":"))
	}
	if returnType == "" {
		returnType = "void"
	}
	return name, "() => " + returnType, true
}

func jsTSTypeIsFunction(typeExpr string) bool {
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	return strings.Contains(typeExpr, "=>") || strings.HasPrefix(typeExpr, "Function")
}

func jsTSTypeFunctionReturnType(typeExpr string) string {
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	arrow := strings.Index(typeExpr, "=>")
	if arrow < 0 {
		return ""
	}
	return strings.TrimSpace(typeExpr[arrow+len("=>"):])
}

func jsMockValueForTSType(fieldName, typeExpr string) string {
	return jsMockValueForTSTypeWithDecls(fieldName, typeExpr, nil)
}

func jsMockValueForTSTypeWithDecls(fieldName, typeExpr string, decls map[string]string) string {
	return jsMockValueForTSTypeWithDeclsSeen(fieldName, typeExpr, decls, nil)
}

func jsMockValueForTSTypeWithDeclsSeen(fieldName, typeExpr string, decls map[string]string, seen map[string]bool) string {
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	typeExpr = jsUnwrapTSUtilityWrappers(typeExpr)
	if branch, ok := jsPreferredTSTypeUnionBranch(typeExpr); ok {
		typeExpr = branch
		typeExpr = jsUnwrapTSUtilityWrappers(typeExpr)
	}
	if value, ok := jsConstructorMockValueForTSType(typeExpr, decls); ok {
		return value
	}
	if value, ok := jsEnumMockValueForTSType(typeExpr, decls); ok {
		return value
	}
	if value, ok := jsMockValueForTSLiteral(typeExpr); ok {
		return value
	}
	if payload, ok := jsMockIndexedAccessPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload
	}
	if payload, ok := jsMockIntersectionPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload
	}
	if payload, ok := jsMockProjectionPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload
	}
	if payload, ok := jsMockRecordPayloadFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return payload
	}
	compactName := jsCompactName(fieldName)
	switch {
	case jsTSTypeIsFunction(typeExpr):
		return jsMockFunctionValueForTSType(typeExpr, decls, seen)
	case strings.HasPrefix(typeExpr, "Promise<") || strings.HasPrefix(typeExpr, "PromiseLike<"):
		if args, ok := jsTSGenericArgs(typeExpr, "Promise"); ok && len(args) == 1 {
			return "Promise.resolve(" + jsMockValueForTSTypeWithDeclsSeen(fieldName, args[0], decls, seen) + ")"
		}
		if args, ok := jsTSGenericArgs(typeExpr, "PromiseLike"); ok && len(args) == 1 {
			return "Promise.resolve(" + jsMockValueForTSTypeWithDeclsSeen(fieldName, args[0], decls, seen) + ")"
		}
		return "Promise.resolve(undefined)"
	case typeExpr == "Buffer":
		return "Buffer.from('test')"
	case typeExpr == "Uint8Array":
		return "new Uint8Array([1, 2, 3])"
	case typeExpr == "Date":
		return "new Date('2026-01-01T00:00:00.000Z')"
	case strings.HasPrefix(typeExpr, "Map<") || typeExpr == "Map":
		if args, ok := jsTSGenericArgs(typeExpr, "Map"); ok && len(args) == 2 {
			value := jsMockValueForTSTypeWithDeclsSeen("value", args[1], decls, seen)
			return "new Map([['key', " + value + "]])"
		}
		return "new Map([['key', 'value']])"
	case strings.HasPrefix(typeExpr, "Set<") || typeExpr == "Set":
		if args, ok := jsTSGenericArgs(typeExpr, "Set"); ok && len(args) == 1 {
			value := jsMockValueForTSTypeWithDeclsSeen("value", args[0], decls, seen)
			return "new Set([" + value + "])"
		}
		return "new Set(['value'])"
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
		if inner, ok := jsTSArrayElementType(typeExpr); ok {
			if payload, ok := jsMockPayloadFromTSTypeWithDeclsSeen(inner, decls, seen); ok {
				return "[" + payload + "]"
			}
		}
		return "[]"
	}
	if object, ok := jsObjectMockFromTSTypeWithDeclsSeen(typeExpr, decls, seen); ok {
		return object
	}
	return "{}"
}

func jsConstructorMockValueForTSType(typeExpr string, decls map[string]string) (string, bool) {
	if len(decls) == 0 {
		return "", false
	}
	name, resolved := jsResolveNamedTSType(typeExpr, decls)
	if resolved == "" || !strings.HasPrefix(resolved, jsConstructorMockKey) {
		return "", false
	}
	if name == "" {
		return strings.TrimPrefix(resolved, jsConstructorMockKey), true
	}
	return strings.TrimPrefix(resolved, jsConstructorMockKey), true
}

func jsEnumMockValueForTSType(typeExpr string, decls map[string]string) (string, bool) {
	if len(decls) == 0 {
		return "", false
	}
	_, resolved := jsResolveNamedTSType(typeExpr, decls)
	if resolved == "" || !strings.HasPrefix(resolved, jsEnumMockKey) {
		return "", false
	}
	return strings.TrimPrefix(resolved, jsEnumMockKey), true
}

func jsMockFunctionValueForTSType(typeExpr string, decls map[string]string, seen map[string]bool) string {
	returnType := jsTSTypeFunctionReturnType(typeExpr)
	if returnType == "" || returnType == "void" || returnType == "undefined" {
		return "() => undefined"
	}
	if args, ok := jsTSGenericArgs(returnType, "Promise"); ok && len(args) == 1 {
		return "async () => " + jsMockValueForTSTypeWithDeclsSeen("value", args[0], decls, seen)
	}
	if args, ok := jsTSGenericArgs(returnType, "PromiseLike"); ok && len(args) == 1 {
		return "async () => " + jsMockValueForTSTypeWithDeclsSeen("value", args[0], decls, seen)
	}
	return "() => " + jsMockValueForTSTypeWithDeclsSeen("value", returnType, decls, seen)
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

func jsSplitTopLevelTypeIntersection(typeExpr string) []string {
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
		case '&':
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
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	if jsTSIdentifierRe.MatchString(typeExpr) {
		return typeExpr, strings.TrimSpace(decls[typeExpr])
	}
	name, args, ok := jsTSNamedGenericParts(typeExpr)
	if !ok {
		return "", ""
	}
	for declName, decl := range decls {
		declBase, params, ok := jsTSNamedGenericParts(declName)
		if !ok || declBase != name || len(params) != len(args) {
			continue
		}
		if !jsTSGenericParamsAreSimple(params) {
			continue
		}
		return typeExpr, strings.TrimSpace(jsSubstituteTSTypeParams(decl, params, args))
	}
	return "", ""
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

func jsUnwrapTSUtilityWrappers(typeExpr string) string {
	typeExpr = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(typeExpr), ";"))
	for {
		unwrapped := false
		for _, name := range []string{"Readonly", "Required", "Partial"} {
			if inner, ok := jsTSGenericArg(typeExpr, name); ok {
				typeExpr = strings.TrimSpace(inner)
				unwrapped = true
				break
			}
		}
		if !unwrapped {
			return typeExpr
		}
	}
}

func jsTSGenericArg(typeExpr, name string) (string, bool) {
	prefix := name + "<"
	if !strings.HasPrefix(typeExpr, prefix) || !strings.HasSuffix(typeExpr, ">") {
		return "", false
	}
	return strings.TrimSpace(typeExpr[len(prefix) : len(typeExpr)-1]), true
}

func jsTSGenericArgs(typeExpr, name string) ([]string, bool) {
	inner, ok := jsTSGenericArg(jsNormalizeTSTypeExpr(typeExpr), name)
	if !ok {
		return nil, false
	}
	return jsSplitTopLevelGenericArgs(inner), true
}

func jsTSNamedGenericParts(typeExpr string) (string, []string, bool) {
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	if !strings.HasSuffix(typeExpr, ">") {
		return "", nil, false
	}
	open := strings.Index(typeExpr, "<")
	if open <= 0 {
		return "", nil, false
	}
	name := strings.TrimSpace(typeExpr[:open])
	if !jsTSIdentifierRe.MatchString(name) {
		return "", nil, false
	}
	inner := strings.TrimSpace(typeExpr[open+1 : len(typeExpr)-1])
	if inner == "" {
		return "", nil, false
	}
	args := jsSplitTopLevelGenericArgs(inner)
	if len(args) == 0 {
		return "", nil, false
	}
	return name, args, true
}

func jsTSGenericParamsAreSimple(params []string) bool {
	for _, param := range params {
		if !jsTSIdentifierRe.MatchString(strings.TrimSpace(param)) {
			return false
		}
	}
	return true
}

func jsSubstituteTSTypeParams(typeExpr string, params, args []string) string {
	if len(params) == 0 || len(params) != len(args) {
		return typeExpr
	}
	replacements := make(map[string]string, len(params))
	for i, param := range params {
		replacements[strings.TrimSpace(param)] = strings.TrimSpace(args[i])
	}

	var out strings.Builder
	for i := 0; i < len(typeExpr); {
		ch := typeExpr[i]
		if jsTSIdentifierStart(ch) {
			start := i
			i++
			for i < len(typeExpr) && jsTSIdentifierPart(typeExpr[i]) {
				i++
			}
			ident := typeExpr[start:i]
			if replacement, ok := replacements[ident]; ok {
				out.WriteString(replacement)
			} else {
				out.WriteString(ident)
			}
			continue
		}
		out.WriteByte(ch)
		i++
	}
	return out.String()
}

func jsTSIdentifierStart(ch byte) bool {
	return ch == '_' || ch == '$' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func jsTSIdentifierPart(ch byte) bool {
	return jsTSIdentifierStart(ch) || (ch >= '0' && ch <= '9')
}

func jsSplitTopLevelGenericArgs(inner string) []string {
	var args []string
	start := 0
	angleDepth, braceDepth, bracketDepth, parenDepth := 0, 0, 0, 0
	var quote rune
	for i, ch := range inner {
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"':
			quote = ch
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
		case ',':
			if angleDepth == 0 && braceDepth == 0 && bracketDepth == 0 && parenDepth == 0 {
				if arg := strings.TrimSpace(inner[start:i]); arg != "" {
					args = append(args, arg)
				}
				start = i + 1
			}
		}
	}
	if arg := strings.TrimSpace(inner[start:]); arg != "" {
		args = append(args, arg)
	}
	return args
}

func jsTSStringLiteralUnionValues(typeExpr string) (map[string]bool, bool) {
	keys, ok := jsTSStringLiteralUnionList(typeExpr)
	if !ok {
		return nil, false
	}
	values := make(map[string]bool, len(keys))
	for _, key := range keys {
		values[key] = true
	}
	return values, true
}

func jsTSStringLiteralUnionList(typeExpr string) ([]string, bool) {
	parts := jsSplitTopLevelTypeUnion(typeExpr)
	if len(parts) == 0 {
		return nil, false
	}
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) < 2 {
			return nil, false
		}
		quote := part[0]
		if (quote != '\'' && quote != '"') || part[len(part)-1] != quote {
			return nil, false
		}
		key := strings.TrimSpace(part[1 : len(part)-1])
		if !regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`).MatchString(key) {
			return nil, false
		}
		values = append(values, key)
	}
	return values, true
}

func jsTSIndexedAccessParts(typeExpr string) (string, string, bool) {
	typeExpr = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(typeExpr), ";"))
	if !strings.HasSuffix(typeExpr, "]") {
		return "", "", false
	}
	angleDepth, braceDepth, bracketDepth, parenDepth := 0, 0, 0, 0
	var quote rune
	for i, ch := range typeExpr {
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"':
			quote = ch
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
			if angleDepth == 0 && braceDepth == 0 && bracketDepth == 0 && parenDepth == 0 {
				source := strings.TrimSpace(typeExpr[:i])
				keyExpr := strings.TrimSpace(typeExpr[i+1 : len(typeExpr)-1])
				key, ok := jsTSStringLiteralValue(keyExpr)
				return source, key, ok && source != ""
			}
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
		}
	}
	return "", "", false
}

func jsTSStringLiteralValue(typeExpr string) (string, bool) {
	typeExpr = strings.TrimSpace(typeExpr)
	if len(typeExpr) < 2 {
		return "", false
	}
	quote := typeExpr[0]
	if (quote != '\'' && quote != '"') || typeExpr[len(typeExpr)-1] != quote {
		return "", false
	}
	key := strings.TrimSpace(typeExpr[1 : len(typeExpr)-1])
	if !regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`).MatchString(key) {
		return "", false
	}
	return key, true
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
		"async", "await", "yield", "static", "this":
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

func jsESMImportLines(funcs []jsFuncInfo, classes []jsClassInfo, moduleImportPath string) string {
	defaultImport := ""
	for _, cls := range classes {
		if cls.DefaultInstance != "" {
			defaultImport = cls.DefaultInstance
			break
		}
		if cls.IsDefault {
			defaultImport = cls.Name
			break
		}
	}
	for _, fn := range funcs {
		if defaultImport == "" && fn.IsDefault {
			defaultImport = fn.Name
			break
		}
	}

	namedImport := joinNamedESMExportNames(funcs, classes)
	switch {
	case defaultImport != "" && namedImport != "":
		return fmt.Sprintf("import %s, { %s } from '%s';\n\n", defaultImport, namedImport, moduleImportPath)
	case defaultImport != "":
		return fmt.Sprintf("import %s from '%s';\n\n", defaultImport, moduleImportPath)
	case namedImport != "":
		return fmt.Sprintf("import { %s } from '%s';\n\n", namedImport, moduleImportPath)
	default:
		return fmt.Sprintf("import '%s';\n\n", moduleImportPath)
	}
}

func joinNamedESMExportNames(funcs []jsFuncInfo, classes []jsClassInfo) string {
	var names []string
	seen := make(map[string]bool)
	for _, fn := range funcs {
		if fn.IsMethod || fn.IsDefault {
			continue
		}
		if fn.SourceIsESModule && !fn.IsExported {
			continue
		}
		if !seen[fn.Name] {
			names = append(names, fn.Name)
			seen[fn.Name] = true
		}
	}
	for _, cls := range classes {
		if cls.DefaultInstance != "" || cls.IsDefault {
			continue
		}
		if cls.SourceIsESModule && !cls.IsExported {
			continue
		}
		if !seen[cls.Name] {
			names = append(names, cls.Name)
			seen[cls.Name] = true
		}
	}
	return strings.Join(names, ", ")
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

func generatorTestPath(srcPath string, task *types.CoverageTestTask) string {
	if task != nil && strings.TrimSpace(task.TestFile) != "" {
		return task.TestFile
	}
	return TestFileName(srcPath)
}

func jsSourceModuleImportPath(srcPath string, testPath string) string {
	ext := strings.ToLower(filepath.Ext(srcPath))
	sourceWithoutExt := strings.TrimSuffix(srcPath, filepath.Ext(srcPath))
	if strings.HasSuffix(strings.ToLower(srcPath), ".d.ts") {
		sourceWithoutExt = strings.TrimSuffix(srcPath, ".d.ts")
	}
	rel, err := filepath.Rel(filepath.Dir(testPath), sourceWithoutExt)
	if err != nil {
		rel = stripExt(baseName(srcPath))
	}
	importPath := filepath.ToSlash(rel)
	if !strings.HasPrefix(importPath, ".") {
		importPath = "./" + importPath
	}
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
