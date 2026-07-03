package generator

import (
	"fmt"
	"os"
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
	ReturnType  string           // number/string/array/object/boolean/null/undefined/unknown
	Throws      bool             // 函数体包含 throw
	Boundaries  []jsBoundary     // 边界条件检测
	HasReturn   bool             // 是否有 return 语句（非 void）
	IsGetter    bool             // 是否是简单的 getter（return expression 只有一个变量/字面量）
}

// jsBoundary 边界条件
type jsBoundary struct {
	Param string // 参数名
	Value string // 边界值（原始字面量）
	Type  string // 值类型：number/string/null/undefined/boolean
}

// jsReturnTypeJS 返回 JS 类型字符串用于 typeof 断言
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

// ---- 正则模式 ----

// function 声明: [export ] [async ] function name(params) {
var jsFuncRe = regexp.MustCompile(
	`(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(([^)]*)\)`,
)

// 箭头函数: [export ] const name = [async ](params) =>
var jsArrowRe = regexp.MustCompile(
	`(?:export\s+)?const\s+(\w+)\s*=\s*(?:async\s*)?\(([^)]*)\)\s*=>`,
)

// 命名函数表达式: [export ] const name = [async ] function(params) {
var jsNamedFnRe = regexp.MustCompile(
	`(?:export\s+)?const\s+(\w+)\s*=\s*(?:async\s+)?function\s*\(([^)]*)\)`,
)

// class 声明: [export ] class Name [extends Base] {
var jsClassRe = regexp.MustCompile(
	`(?:export\s+)?class\s+(\w+)(?:\s+extends\s+\w+)?\s*\{`,
)

// 对象方法简写 (module.exports = { name(params) { ... } })
var jsObjMethodRe = regexp.MustCompile(
	`^\s*(?:async\s+)?(\w+)\s*\(([^)]*)\)\s*\{`,
)

// export 检测
var jsExportRe = regexp.MustCompile(`^\s*export\s+`)

// ---- 核心函数 ----

// GenerateJestTests 读取 JS/TS 源文件，生成 Jest 测试代码
func GenerateJestTests(srcPath string) (string, error) {
	source, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("读取源文件失败: %w", err)
	}
	src := string(source)

	// 检测模块系统
	isESModule := jsExportRe.MatchString(src)

	// 提取所有函数
	var funcs []jsFuncInfo
	funcs = append(funcs, extractJSFunctions(src)...)
	funcs = append(funcs, extractJSArrowFunctions(src)...)
	funcs = append(funcs, extractJSNamedFunctions(src)...)

	// 提取类和方法
	classes := extractJSClasses(src)

	// 去重（箭头函数和 function 声明可能匹配同一个）
	funcs = dedupJSFuncs(funcs)

	if len(funcs) == 0 && len(classes) == 0 {
		return "// 未发现需要生成测试的函数或类", nil
	}

	// 推导模块名（不含扩展名）
	moduleName := stripExt(baseName(srcPath))

	var buf strings.Builder

	// 生成导入语句
	if isESModule {
		buf.WriteString(fmt.Sprintf("import { %s } from './%s';\n\n", joinExportNames(funcs, classes), moduleName))
	} else {
		buf.WriteString(fmt.Sprintf("const { %s } = require('./%s');\n\n", joinExportNames(funcs, classes), moduleName))
	}

	// 为每个函数生成测试
	for _, fn := range funcs {
		buf.WriteString(genJestFuncTest(fn))
	}

	// 为每个类生成测试
	for _, cls := range classes {
		buf.WriteString(genJestClassTest(cls, isESModule, moduleName))
	}

	return buf.String(), nil
}

// ---- 提取函数 ----

func extractJSFunctions(src string) []jsFuncInfo {
	var funcs []jsFuncInfo
	matches := jsFuncRe.FindAllStringSubmatchIndex(src, -1)
	for _, idx := range matches {
		fullMatch := src[idx[0]:idx[1]]
		name := src[idx[2]:idx[3]]
		paramStr := src[idx[4]:idx[5]]

		fn := jsFuncInfo{
			Name:       name,
			Params:     parseJSParams(paramStr),
			IsExported: jsExportRe.MatchString(fullMatch),
		}
		fn.IsAsync = strings.Contains(fullMatch, "async")

		if isTestHelper(name) {
			continue
		}

		// 提取函数体：idx[5] 是 ')' 在 src 中的位置
		fn.Body = extractJSBodyAfter(src, idx[5])
		fn.Analysis = analyzeJSBody(fn.Body)
		funcs = append(funcs, fn)
	}
	return funcs
}

func extractJSArrowFunctions(src string) []jsFuncInfo {
	var funcs []jsFuncInfo
	matches := jsArrowRe.FindAllStringSubmatchIndex(src, -1)
	for _, idx := range matches {
		fullMatch := src[idx[0]:idx[1]]
		name := src[idx[2]:idx[3]]
		paramStr := src[idx[4]:idx[5]]

		fn := jsFuncInfo{
			Name:       name,
			Params:     parseJSParams(paramStr),
			IsArrow:    true,
			IsExported: jsExportRe.MatchString(fullMatch),
		}
		fn.IsAsync = strings.Contains(fullMatch, "async")

		if isTestHelper(name) {
			continue
		}

		// 提取箭头函数体：fullMatch 以 => 结尾，body 在 src 中 idx[1] 之后
		fn.Body = extractJSBodyAfter(src, idx[1]-1)
		fn.Analysis = analyzeJSBody(fn.Body)
		funcs = append(funcs, fn)
	}
	return funcs
}

func extractJSNamedFunctions(src string) []jsFuncInfo {
	var funcs []jsFuncInfo
	matches := jsNamedFnRe.FindAllStringSubmatchIndex(src, -1)
	for _, idx := range matches {
		fullMatch := src[idx[0]:idx[1]]
		name := src[idx[2]:idx[3]]
		paramStr := src[idx[4]:idx[5]]

		fn := jsFuncInfo{
			Name:       name,
			Params:     parseJSParams(paramStr),
			IsExported: jsExportRe.MatchString(fullMatch),
		}
		fn.IsAsync = strings.Contains(fullMatch, "async")

		if isTestHelper(name) {
			continue
		}

		// 提取函数体：idx[5] 是 ')' 在 src 中的位置
		fn.Body = extractJSBodyAfter(src, idx[5])
		fn.Analysis = analyzeJSBody(fn.Body)
		funcs = append(funcs, fn)
	}
	return funcs
}

// ---- 提取类 ----

type jsClassInfo struct {
	Name    string
	Methods []jsFuncInfo
}

func extractJSClasses(src string) []jsClassInfo {
	var classes []jsClassInfo

	matches := jsClassRe.FindAllStringSubmatchIndex(src, -1)
	for _, idx := range matches {
		className := src[idx[2]:idx[3]]

		// 找到 class body 的起止位置（花括号匹配）
		braceStart := idx[1] - 1 // 指向 '{'
		braceEnd := findMatchingBrace(src, braceStart)
		if braceEnd < 0 {
			continue
		}

		body := src[braceStart+1 : braceEnd]

		// 在 class body 中查找方法
		var methods []jsFuncInfo
		lines := strings.Split(body, "\n")
		bodyOffset := braceStart + 1 // body 在 src 中的起始偏移
		for _, line := range lines {
			line = strings.TrimRight(line, "\r")
			// 跳过注释行
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "/*") {
				continue
			}

			m := jsObjMethodRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			methodName := m[1]
			paramStr := m[2]

			// 跳过 constructor（单独处理）
			if methodName == "constructor" {
				continue
			}
			// 跳过测试辅助
			if isTestHelper(methodName) {
				continue
			}
			// 跳过 JS 关键字（if/for/while 等不应被当作方法）
			if isJSKeyword(methodName) {
				continue
			}

			method := jsFuncInfo{
				Name:      methodName,
				Params:    parseJSParams(paramStr),
				IsMethod:  true,
				ClassName: className,
				IsAsync:   strings.Contains(line, "async"),
			}

			// 提取方法体：在 class body 中找该行，再找花括号
			lineIdx := strings.Index(body, line)
			if lineIdx >= 0 {
				braceInBody := strings.IndexByte(line, '{')
				if braceInBody >= 0 {
					absBrace := bodyOffset + lineIdx + braceInBody
					if absBrace < len(src) {
						bodyEnd := findMatchingBrace(src, absBrace)
						if bodyEnd > absBrace {
							method.Body = src[absBrace+1 : bodyEnd]
						}
					}
				}
			}
			method.Analysis = analyzeJSBody(method.Body)
			methods = append(methods, method)
		}

		if len(methods) > 0 {
			classes = append(classes, jsClassInfo{
				Name:    className,
				Methods: methods,
			})
		}
	}

	return classes
}

// ---- 参数解析 ----

func parseJSParams(paramStr string) []jsParamInfo {
	paramStr = strings.TrimSpace(paramStr)
	if paramStr == "" {
		return nil
	}

	// 简单逗号分割（不处理嵌套解构的逗号）
	rawParams := splitParams(paramStr)
	var params []jsParamInfo
	for _, raw := range rawParams {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		info := jsParamInfo{}

		// Rest params: ...args
		if strings.HasPrefix(raw, "...") {
			info.IsRest = true
			raw = strings.TrimPrefix(raw, "...")
		}

		// 先检测默认值: param = value（必须在剥离类型之前，因为 TS 有 param: Type = value）
		if eqIdx := indexTopLevelEquals(raw); eqIdx > 0 {
			info.HasDefault = true
			raw = raw[:eqIdx]
		}

		// 再剥离 TypeScript 类型注解: param: Type
		if colonIdx := indexTopLevelColon(raw); colonIdx > 0 {
			raw = raw[:colonIdx]
		}

		info.Name = strings.TrimSpace(raw)
		// 去除可能的解构符号
		info.Name = strings.Trim(info.Name, "{}[]")

		if info.Name != "" {
			params = append(params, info)
		}
	}
	return params
}

// splitParams 按顶层逗号分割参数（忽略括号和方括号内的逗号）
func splitParams(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, ch := range s {
		switch ch {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// indexTopLevelColon 找到顶层（非括号内）的第一个冒号位置（用于剥离 TS 类型）
func indexTopLevelColon(s string) int {
	depth := 0
	for i, ch := range s {
		switch ch {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ':':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// indexTopLevelEquals 找到顶层（非括号/方括号内）的第一个等号位置
func indexTopLevelEquals(s string) int {
	depth := 0
	for i, ch := range s {
		switch ch {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case '=':
			if depth == 0 && i > 0 {
				// 确保不是 ==, =>, >=, <=, !=
				prev := s[i-1]
				next := byte(0)
				if i+1 < len(s) {
					next = s[i+1]
				}
				if prev != '=' && prev != '<' && prev != '>' && prev != '!' && next != '=' && next != '>' {
					return i
				}
			}
		}
	}
	return -1
}

// extractJSBodyAfter 从 src 中 pos 位置之后提取函数体
// 支持 { ... } 花括号体和 => expr 表达式体
func extractJSBodyAfter(src string, pos int) string {
	for j := pos + 1; j < len(src); j++ {
		ch := src[j]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			continue
		}
		if ch == '{' {
			bodyEnd := findMatchingBrace(src, j)
			if bodyEnd > j {
				return src[j+1 : bodyEnd]
			}
			return ""
		}
		// 箭头函数: => expr
		if ch == '=' && j+1 < len(src) && src[j+1] == '>' {
			k := j + 2
			for k < len(src) && (src[k] == ' ' || src[k] == '\t') {
				k++
			}
			if k < len(src) && src[k] == '{' {
				bodyEnd := findMatchingBrace(src, k)
				if bodyEnd > k {
					return src[k+1 : bodyEnd]
				}
				return ""
			}
			// 表达式体：到行尾或分号
			start := k
			for k < len(src) && src[k] != '\n' && src[k] != ';' {
				k++
			}
			expr := strings.TrimSpace(src[start:k])
			if expr != "" {
				return "return " + expr
			}
		}
		// 其他非空白字符 → 箭头函数表达式体（如 `=> a * b` 中的 `a`）
		// 表达式体：到行尾或分号
		start := j
		for k := j; k < len(src); k++ {
			if src[k] == '\n' || src[k] == ';' {
				expr := strings.TrimSpace(src[start:k])
				if expr != "" {
					return "return " + expr
				}
				break
			}
		}
		// 到文件末尾
		expr := strings.TrimSpace(src[start:])
		if expr != "" {
			return "return " + expr
		}
		return ""
	}
	return ""
}

// ---- 函数体分析 ----

var (
	jsReturnRe  = regexp.MustCompile(`\breturn\s+(.+?)(?:;|\n|$)`)
	jsThrowRe   = regexp.MustCompile(`\bthrow\b`)
	jsIfEqRe    = regexp.MustCompile(`if\s*\(\s*(\w+)\s*(?:===?|!==?)\s*([^)]+?)\s*\)`)
	jsIfNullRe  = regexp.MustCompile(`if\s*\(\s*(\w+)\s*(?:===?|!==?)\s*(null|undefined)\s*\)`)
)

// analyzeJSBody 分析 JS 函数体，推断返回类型、检测 throw 和边界条件
func analyzeJSBody(body string) jsFuncAnalysis {
	a := jsFuncAnalysis{}

	if body == "" {
		return a
	}

	// 检测 throw
	a.Throws = jsThrowRe.MatchString(body)

	// 检测 return 语句
	returnMatches := jsReturnRe.FindAllStringSubmatch(body, -1)
	a.HasReturn = len(returnMatches) > 0

	// 推断返回类型
	if a.HasReturn {
		a.ReturnType = inferJSReturnType(returnMatches)
	}

	// 检测边界条件
	a.Boundaries = extractJSBoundaries(body)

	return a
}

// inferJSReturnType 根据返回值表达式推断类型
func inferJSReturnType(matches [][]string) string {
	for _, m := range matches {
		expr := strings.TrimSpace(m[1])

		// null
		if expr == "null" {
			return "null"
		}
		// undefined
		if expr == "undefined" {
			return "undefined"
		}
		// boolean
		if expr == "true" || expr == "false" {
			return "boolean"
		}
		// 字符串字面量
		if (strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"")) ||
			(strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'")) {
			return "string"
		}
		// 模板字符串
		if strings.HasPrefix(expr, "`") {
			return "string"
		}
		// 数字字面量
		if isNumericLiteral(expr) {
			return "number"
		}
		// 数组字面量
		if strings.HasPrefix(expr, "[") {
			return "array"
		}
		// 对象字面量
		if strings.HasPrefix(expr, "{") {
			return "object"
		}
		// JSON.parse() → object/array
		if strings.Contains(expr, "JSON.parse") {
			return "object"
		}
		// response.json() → Promise (async)
		if strings.Contains(expr, ".json()") {
			return "object"
		}
		// 算术运算 → number
		if isArithmeticExpr(expr) {
			return "number"
		}
		// 逻辑运算 → boolean
		if isLogicalExpr(expr) {
			return "boolean"
		}
		// 拼接 → string
		if strings.Contains(expr, " + ") && hasStringLiteral(expr) {
			return "string"
		}
	}

	// 有 return 但无法推断 → unknown
	return "unknown"
}

// isNumericLiteral 判断是否是数字字面量
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

// isArithmeticExpr 简单检测算术表达式
func isArithmeticExpr(s string) bool {
	// 包含 + - * / % 且操作数看起来像数字或变量
	for _, op := range []string{" + ", " - ", " * ", " / ", " % "} {
		if strings.Contains(s, op) {
			// 但字符串拼接也是 +，排除含引号的情况
			if op == " + " && hasStringLiteral(s) {
				return false
			}
			return true
		}
	}
	return false
}

// isLogicalExpr 检测逻辑表达式
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

// extractJSBoundaries 从 if 条件中提取边界条件
func extractJSBoundaries(body string) []jsBoundary {
	var boundaries []jsBoundary
	seen := make(map[string]bool)

	// 先检测 null/undefined 检查
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

	// 再检测其他条件
	ifMatches := jsIfEqRe.FindAllStringSubmatch(body, -1)
	for _, m := range ifMatches {
		param := m[1]
		val := strings.TrimSpace(m[2])

		// 跳过已处理的 null/undefined
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

	// 正常用例：基于返回类型生成断言
	sb.WriteString(fmt.Sprintf("  it('should return expected result for normal input', %s => {\n", jsAsyncArrow(fn.IsAsync)))
	if fn.IsAsync {
		sb.WriteString(fmt.Sprintf("    const result = await %s(%s);\n", fn.Name, jsArgList(fn.Params)))
	} else {
		sb.WriteString(fmt.Sprintf("    const result = %s(%s);\n", fn.Name, jsArgList(fn.Params)))
	}
	sb.WriteString(genJSResultAssertion(fn.Analysis, "    "))
	sb.WriteString("  });\n\n")

	// 边界用例：基于函数体中的 if 条件
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

	// 错误用例：如果函数体有 throw，生成 toThrow 测试
	if fn.Analysis.Throws {
		sb.WriteString(fmt.Sprintf("  it('should throw on invalid input', %s => {\n", jsAsyncArrow(fn.IsAsync)))
		if fn.IsAsync {
			sb.WriteString(fmt.Sprintf("    await expect(%s(%s)).rejects.toThrow();\n", fn.Name, jsArgList(fn.Params)))
		} else {
			sb.WriteString(fmt.Sprintf("    expect(() => %s(%s)).toThrow();\n", fn.Name, jsArgList(fn.Params)))
		}
		sb.WriteString("  });\n\n")
	}

	// 零参数用例
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

	// 实例化测试
	sb.WriteString(fmt.Sprintf("  describe('constructor', () => {\n"))
	sb.WriteString(fmt.Sprintf("    it('should create an instance', () => {\n"))
	sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
	sb.WriteString(fmt.Sprintf("      expect(instance).toBeInstanceOf(%s);\n", cls.Name))
	sb.WriteString("    });\n")
	sb.WriteString("  });\n\n")

	// 方法测试
	for _, method := range cls.Methods {
		sb.WriteString(fmt.Sprintf("  describe('%s', () => {\n", method.Name))

		// 正常用例
		sb.WriteString(fmt.Sprintf("    it('should return expected result', %s => {\n", jsAsyncArrow(method.IsAsync)))
		sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
		if method.IsAsync {
			sb.WriteString(fmt.Sprintf("      const result = await instance.%s(%s);\n", method.Name, jsArgList(method.Params)))
		} else {
			sb.WriteString(fmt.Sprintf("      const result = instance.%s(%s);\n", method.Name, jsArgList(method.Params)))
		}
		sb.WriteString(genJSResultAssertion(method.Analysis, "      "))
		sb.WriteString("    });\n\n")

		// 错误用例
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

		// 边界用例
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

// genJSResultAssertion 根据返回类型分析生成断言
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

// jsAsyncArrow 返回 async 箭头函数前缀
func jsAsyncArrow(isAsync bool) string {
	if isAsync {
		return "async ()"
	}
	return "()"
}

// jsParamExists 检查参数名是否存在
func jsParamExists(params []jsParamInfo, name string) bool {
	for _, p := range params {
		if p.Name == name {
			return true
		}
	}
	return false
}

// jsArgListWithBoundary 生成带边界值的参数列表
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

// jsArgList 生成调用参数列表（用 undefined 占位）
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

// ---- 辅助函数 ----

func isTestHelper(name string) bool {
	switch name {
	case "test", "it", "describe", "beforeEach", "beforeAll", "afterEach", "afterAll", "expect", "jest", "before", "after":
		return true
	}
	return false
}

// isJSKeyword 判断是否是 JS 关键字（避免把 if/for/while 等误认为方法）
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

func findMatchingBrace(src string, openIdx int) int {
	if openIdx < 0 || openIdx >= len(src) || src[openIdx] != '{' {
		return -1
	}
	depth := 0
	inString := false
	stringChar := byte(0)
	for i := openIdx; i < len(src); i++ {
		ch := src[i]
		if inString {
			if ch == stringChar && (i == 0 || src[i-1] != '\\') {
				inString = false
			}
			continue
		}
		switch ch {
		case '"', '\'', '`':
			inString = true
			stringChar = ch
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func baseName(path string) string {
	// 取最后一段路径
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
