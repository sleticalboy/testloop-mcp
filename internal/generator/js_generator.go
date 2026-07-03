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
}

type jsParamInfo struct {
	Name       string
	HasDefault bool
	IsRest     bool // ...args
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
	matches := jsFuncRe.FindAllStringSubmatch(src, -1)
	for _, m := range matches {
		fullMatch := m[0]
		name := m[1]
		paramStr := m[2]

		fn := jsFuncInfo{
			Name:       name,
			Params:     parseJSParams(paramStr),
			IsExported: jsExportRe.MatchString(fullMatch),
		}
		// 检测 async
		fn.IsAsync = strings.Contains(fullMatch, "async")

		// 跳过以大写开头但不是构造函数的（可能是 React 组件，仍然生成）
		// 跳过 test/it/describe 等测试辅助函数
		if isTestHelper(name) {
			continue
		}
		funcs = append(funcs, fn)
	}
	return funcs
}

func extractJSArrowFunctions(src string) []jsFuncInfo {
	var funcs []jsFuncInfo
	matches := jsArrowRe.FindAllStringSubmatch(src, -1)
	for _, m := range matches {
		fullMatch := m[0]
		name := m[1]
		paramStr := m[2]

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
		funcs = append(funcs, fn)
	}
	return funcs
}

func extractJSNamedFunctions(src string) []jsFuncInfo {
	var funcs []jsFuncInfo
	matches := jsNamedFnRe.FindAllStringSubmatch(src, -1)
	for _, m := range matches {
		fullMatch := m[0]
		name := m[1]
		paramStr := m[2]

		fn := jsFuncInfo{
			Name:       name,
			Params:     parseJSParams(paramStr),
			IsExported: jsExportRe.MatchString(fullMatch),
		}
		fn.IsAsync = strings.Contains(fullMatch, "async")

		if isTestHelper(name) {
			continue
		}
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

// ---- 测试生成 ----

func genJestFuncTest(fn jsFuncInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("describe('%s', () => {\n", fn.Name))

	// 正常用例
	sb.WriteString(fmt.Sprintf("  it('should return expected result for normal input', () => {\n"))
	sb.WriteString(fmt.Sprintf("    const result = %s(%s);\n", fn.Name, jsArgList(fn.Params)))
	sb.WriteString("    expect(result).toBeDefined();\n")
	sb.WriteString("  });\n\n")

	// 边界用例
	sb.WriteString(fmt.Sprintf("  it('should handle edge cases', () => {\n"))
	if len(fn.Params) == 0 {
		sb.WriteString(fmt.Sprintf("    const result = %s();\n", fn.Name))
		sb.WriteString("    expect(result).toBeDefined();\n")
	} else {
		sb.WriteString(fmt.Sprintf("    const result = %s(%s);\n", fn.Name, jsDefaultArgs(fn.Params)))
		sb.WriteString("    expect(result).toBeDefined();\n")
	}
	sb.WriteString("  });\n")

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
		sb.WriteString(fmt.Sprintf("    it('should return expected result', () => {\n"))
		sb.WriteString(fmt.Sprintf("      const instance = new %s();\n", cls.Name))
		sb.WriteString(fmt.Sprintf("      const result = instance.%s(%s);\n", method.Name, jsArgList(method.Params)))
		sb.WriteString("      expect(result).toBeDefined();\n")
		sb.WriteString("    });\n")
		sb.WriteString("  });\n\n")
	}

	sb.WriteString("});\n\n");

	return sb.String()
}

// jsArgList 生成调用参数列表（用 undefined 占位）
func jsArgList(params []jsParamInfo) string {
	if len(params) == 0 {
		return ""
	}
	args := make([]string, len(params))
	for i, p := range params {
		if p.IsRest {
			args[i] = "[]" // rest 参数传空数组
		} else if p.HasDefault {
			args[i] = p.Name // 有默认值的参数不传，让默认值生效——但这里还是传 undefined
			args[i] = "undefined"
		} else {
			args[i] = "undefined"
		}
	}
	return strings.Join(args, ", ")
}

// jsDefaultArgs 生成边界测试的参数（全 undefined）
func jsDefaultArgs(params []jsParamInfo) string {
	return jsArgList(params)
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
