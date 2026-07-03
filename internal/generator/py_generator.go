package generator

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ---- 类型定义 ----

type pyFuncInfo struct {
	Name       string
	Params     []pyParamInfo
	IsAsync    bool
	IsMethod   bool
	ClassName  string
	IsStatic   bool
	Body       string          // 函数体源码
	Analysis   pyFuncAnalysis  // 函数体分析结果
}

type pyParamInfo struct {
	Name       string
	HasDefault bool
	IsArgs     bool // *args
	IsKwargs   bool // **kwargs
}

// pyFuncAnalysis 函数体分析结果
type pyFuncAnalysis struct {
	ReturnType string       // int/float/str/list/dict/bool/None/unknown
	Raises     bool         // 函数体包含 raise
	Boundaries []pyBoundary // 边界条件检测
	HasReturn  bool         // 是否有 return 语句
}

// pyBoundary 边界条件
type pyBoundary struct {
	Param string
	Value string
	Type  string // number/string/None/boolean
}

// ---- 正则模式 ----

// def 声明: [async ] def name(params):
var pyFuncRe = regexp.MustCompile(
	`^\s*(async\s+)?def\s+(\w+)\s*\(([^)]*)\)\s*(?:->\s*[^:]+)?\s*:`,
)

// class 声明: class Name[(Base)]:
var pyClassRe = regexp.MustCompile(
	`^class\s+(\w+)\s*(?:\([^)]*\))?\s*:`,
)

// ---- 核心函数 ----

// GeneratePytestTests 读取 Python 源文件，生成 pytest 测试代码
func GeneratePytestTests(srcPath string) (string, error) {
	source, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("读取源文件失败: %w", err)
	}
	src := string(source)

	// 提取函数和类
	funcs, classes := extractPyDecls(src)

	if len(funcs) == 0 && len(classes) == 0 {
		return "# 未发现需要生成测试的函数或类", nil
	}

	// 推导模块名（不含扩展名）
	moduleName := stripExt(baseName(srcPath))

	var buf strings.Builder

	// 检测是否有 async 函数（函数和方法都要检查）
	hasAsync := false
	for _, fn := range funcs {
		if fn.IsAsync {
			hasAsync = true
			break
		}
	}
	if !hasAsync {
		for _, cls := range classes {
			for _, m := range cls.Methods {
				if m.IsAsync {
					hasAsync = true
					break
				}
			}
			if hasAsync {
				break
			}
		}
	}

	// 生成导入语句
	buf.WriteString(fmt.Sprintf("from %s import %s\n", moduleName, joinPyExportNames(funcs, classes)))
	if hasAsync {
		buf.WriteString("import asyncio\n")
	}
	// 检测是否有 raise 测试需要 pytest
	needsPytest := false
	for _, fn := range funcs {
		if fn.Analysis.Raises {
			needsPytest = true
			break
		}
	}
	if !needsPytest {
		for _, cls := range classes {
			for _, m := range cls.Methods {
				if m.Analysis.Raises {
					needsPytest = true
					break
				}
			}
			if needsPytest {
				break
			}
		}
	}
	if needsPytest {
		buf.WriteString("import pytest\n")
	}
	buf.WriteString("\n")

	// 为每个函数生成测试
	for _, fn := range funcs {
		buf.WriteString(genPytestFuncTest(fn))
	}

	// 为每个类生成测试
	for _, cls := range classes {
		buf.WriteString(genPytestClassTest(cls))
	}

	return buf.String(), nil
}

// ---- 提取声明 ----

type pyClassInfo struct {
	Name    string
	Methods []pyFuncInfo
}

func extractPyDecls(src string) ([]pyFuncInfo, []pyClassInfo) {
	var funcs []pyFuncInfo
	var classes []pyClassInfo

	lines := strings.Split(src, "\n")

	// 追踪类上下文
	var currentClass *pyClassInfo
	currentClassIndent := -1

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimRight(line, "\r")

		// 跳过空行和注释
		if strings.TrimSpace(trimmed) == "" || strings.HasPrefix(strings.TrimSpace(trimmed), "#") {
			continue
		}

		// 检测 class 声明
		if m := pyClassRe.FindStringSubmatch(trimmed); m != nil {
			classIndent := indentLevel(trimmed)
			cls := pyClassInfo{Name: m[1]}
			currentClass = &cls
			currentClassIndent = classIndent
			classes = append(classes, cls)
			continue
		}

		// 检测 def 声明
		if m := pyFuncRe.FindStringSubmatch(trimmed); m != nil {
			isAsync := m[1] != ""
			name := m[2]
			paramStr := m[3]
			lineIndent := indentLevel(trimmed)

			// 跳过 __dunder__ 方法（除了 __init__）
			if isPyDunder(name) && name != "__init__" {
				continue
			}

			// 跳过测试辅助
			if isPyTestHelper(name) {
				continue
			}

			// 判断是否是类方法
			isMethod := false
			isStatic := false
			className := ""

			if currentClass != nil && lineIndent > currentClassIndent {
				isMethod = true
				className = currentClass.Name
				if i > 0 {
					prevLine := strings.TrimSpace(lines[i-1])
					if strings.HasPrefix(prevLine, "@staticmethod") {
						isStatic = true
					}
				}
			} else {
				currentClass = nil
				currentClassIndent = -1
			}

			fn := pyFuncInfo{
				Name:      name,
				Params:    parsePyParams(paramStr, isMethod, isStatic),
				IsAsync:   isAsync,
				IsMethod:  isMethod,
				ClassName: className,
				IsStatic:  isStatic,
			}

			// 提取函数体：从 def 行的下一行开始，收集缩进大于 def 行的行
			fn.Body = extractPyBody(lines, i+1, lineIndent)

			fn.Analysis = analyzePyBody(fn.Body)

			if isMethod && currentClass != nil {
				for idx := range classes {
					if classes[idx].Name == className {
						classes[idx].Methods = append(classes[idx].Methods, fn)
						break
					}
				}
			} else if !isMethod {
				funcs = append(funcs, fn)
			}
		}
	}

	return funcs, classes
}

// extractPyBody 提取 Python 函数体（基于缩进）
func extractPyBody(lines []string, startIdx, defIndent int) string {
	var bodyLines []string
	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		// 空行属于函数体
		if strings.TrimSpace(line) == "" {
			bodyLines = append(bodyLines, line)
			continue
		}
		indent := indentLevel(line)
		// 缩进大于 def 行的属于函数体
		if indent > defIndent {
			bodyLines = append(bodyLines, line)
		} else {
			break
		}
	}
	// 去掉尾部空行
	for len(bodyLines) > 0 && strings.TrimSpace(bodyLines[len(bodyLines)-1]) == "" {
		bodyLines = bodyLines[:len(bodyLines)-1]
	}
	return strings.Join(bodyLines, "\n")
}

// ---- 函数体分析 ----

var (
	pyReturnRe  = regexp.MustCompile(`\breturn\s+(.+)`)
	pyRaiseRe   = regexp.MustCompile(`\braise\b`)
	pyIfEqRe    = regexp.MustCompile(`if\s+(\w+)\s*(?:==?|!=)\s*(.+?):`)
	pyIfNoneRe  = regexp.MustCompile(`if\s+(\w+)\s+(?:is|is\s+not)\s+(None|True|False)\s*:`)
)

// analyzePyBody 分析 Python 函数体
func analyzePyBody(body string) pyFuncAnalysis {
	a := pyFuncAnalysis{}

	if body == "" {
		return a
	}

	// 检测 raise
	a.Raises = pyRaiseRe.MatchString(body)

	// 检测 return 语句
	returnMatches := pyReturnRe.FindAllStringSubmatch(body, -1)
	a.HasReturn = len(returnMatches) > 0

	// 推断返回类型
	if a.HasReturn {
		a.ReturnType = inferPyReturnType(returnMatches)
	}

	// 检测边界条件
	a.Boundaries = extractPyBoundaries(body)

	return a
}

// inferPyReturnType 根据返回值表达式推断类型
func inferPyReturnType(matches [][]string) string {
	for _, m := range matches {
		expr := strings.TrimSpace(m[1])

		// None
		if expr == "None" {
			return "None"
		}
		// bool
		if expr == "True" || expr == "False" {
			return "bool"
		}
		// 字符串字面量
		if (strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"")) ||
			(strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'")) {
			return "str"
		}
		// f-string
		if strings.HasPrefix(expr, "f\"") || strings.HasPrefix(expr, "f'") {
			return "str"
		}
		// 数字字面量
		if isPyNumericLiteral(expr) {
			if strings.Contains(expr, ".") {
				return "float"
			}
			return "int"
		}
		// 列表字面量
		if strings.HasPrefix(expr, "[") {
			return "list"
		}
		// 字典字面量
		if strings.HasPrefix(expr, "{") {
			return "dict"
		}
		// 元组字面量
		if strings.HasPrefix(expr, "(") {
			return "tuple"
		}
		// .json() → dict/list (通常 dict)
		if strings.Contains(expr, ".json()") {
			return "dict"
		}
		// 算术运算 → int/float
		if isPyArithmeticExpr(expr) {
			if strings.Contains(expr, " / ") || strings.Contains(expr, " // ") {
				return "float"
			}
			return "int"
		}
		// 字符串拼接 → str
		if strings.Contains(expr, " + ") && hasPyStringLiteral(expr) {
			return "str"
		}
		// "".join() → str
		if strings.Contains(expr, ".join(") {
			return "str"
		}
	}

	return "unknown"
}

// isPyNumericLiteral 判断是否是数字字面量
func isPyNumericLiteral(s string) bool {
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

// isPyArithmeticExpr 检测算术表达式
func isPyArithmeticExpr(s string) bool {
	for _, op := range []string{" + ", " - ", " * ", " / ", " // ", " % ", " ** "} {
		if strings.Contains(s, op) {
			if op == " + " && hasPyStringLiteral(s) {
				return false
			}
			return true
		}
	}
	return false
}

func hasPyStringLiteral(s string) bool {
	return strings.Contains(s, "\"") || strings.Contains(s, "'")
}

// extractPyBoundaries 从 if 条件中提取边界条件
func extractPyBoundaries(body string) []pyBoundary {
	var boundaries []pyBoundary
	seen := make(map[string]bool)

	// 先检测 None/True/False 检查
	noneMatches := pyIfNoneRe.FindAllStringSubmatch(body, -1)
	for _, m := range noneMatches {
		param := m[1]
		val := m[2]
		key := param + ":" + val
		if !seen[key] {
			seen[key] = true
			boundaries = append(boundaries, pyBoundary{Param: param, Value: val, Type: val})
		}
	}

	// 再检测其他 == 条件
	ifMatches := pyIfEqRe.FindAllStringSubmatch(body, -1)
	for _, m := range ifMatches {
		param := m[1]
		val := strings.TrimSpace(m[2])

		// 跳过 None/True/False（已在上面处理）
		if val == "None" || val == "True" || val == "False" {
			continue
		}

		key := param + ":" + val
		if seen[key] {
			continue
		}
		seen[key] = true

		bType := "unknown"
		if isPyNumericLiteral(val) {
			bType = "number"
		} else if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"")) ||
			(strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) {
			bType = "string"
		}

		boundaries = append(boundaries, pyBoundary{Param: param, Value: val, Type: bType})
	}

	return boundaries
}

// ---- 参数解析 ----

func parsePyParams(paramStr string, isMethod, isStatic bool) []pyParamInfo {
	paramStr = strings.TrimSpace(paramStr)
	if paramStr == "" {
		return nil
	}

	rawParams := splitParams(paramStr)
	var params []pyParamInfo

	for _, raw := range rawParams {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		info := pyParamInfo{}

		// **kwargs
		if strings.HasPrefix(raw, "**") {
			info.IsKwargs = true
			info.Name = strings.TrimPrefix(raw, "**")
			params = append(params, info)
			continue
		}

		// *args
		if strings.HasPrefix(raw, "*") {
			info.IsArgs = true
			info.Name = strings.TrimPrefix(raw, "*")
			params = append(params, info)
			continue
		}

		// 先检测默认值: param = value（必须在剥离类型之前，因为 Python 有 param: Type = default）
		if eqIdx := indexTopLevelEquals(raw); eqIdx > 0 {
			info.HasDefault = true
			raw = raw[:eqIdx]
		}

		// 再剥离类型注解: param: Type
		if colonIdx := indexTopLevelColon(raw); colonIdx > 0 {
			raw = raw[:colonIdx]
		}

		info.Name = strings.TrimSpace(raw)
		if info.Name != "" {
			params = append(params, info)
		}
	}

	// 如果是实例方法，去掉第一个参数（self/cls）
	if isMethod && !isStatic && len(params) > 0 {
		first := params[0].Name
		if first == "self" || first == "cls" {
			params = params[1:]
		}
	}

	return params
}

// ---- 测试生成 ----

func genPytestFuncTest(fn pyFuncInfo) string {
	var sb strings.Builder

	// 正常用例
	if fn.IsAsync {
		sb.WriteString(fmt.Sprintf("def test_%s():\n", fn.Name))
		sb.WriteString(fmt.Sprintf("    result = asyncio.run(%s(%s))\n", fn.Name, pyArgList(fn.Params)))
	} else {
		sb.WriteString(fmt.Sprintf("def test_%s():\n", fn.Name))
		sb.WriteString(fmt.Sprintf("    result = %s(%s)\n", fn.Name, pyArgList(fn.Params)))
	}
	sb.WriteString(genPyResultAssertion(fn.Analysis, "    "))

	// 边界用例
	for _, b := range fn.Analysis.Boundaries {
		if !pyParamExists(fn.Params, b.Param) {
			continue
		}
		sb.WriteString(fmt.Sprintf("\ndef test_%s_%s_boundary():\n", fn.Name, b.Param))
		args := pyArgListWithBoundary(fn.Params, b)
		if fn.IsAsync {
			sb.WriteString(fmt.Sprintf("    result = asyncio.run(%s(%s))\n", fn.Name, args))
		} else {
			sb.WriteString(fmt.Sprintf("    result = %s(%s)\n", fn.Name, args))
		}
		if fn.Analysis.Raises {
			sb.WriteString("    # 边界条件可能触发异常\n")
		}
		sb.WriteString(genPyResultAssertion(fn.Analysis, "    "))
	}

	// 错误用例
	if fn.Analysis.Raises {
		sb.WriteString(fmt.Sprintf("\ndef test_%s_raises():\n", fn.Name))
		if fn.IsAsync {
			sb.WriteString(fmt.Sprintf("    with pytest.raises(Exception):\n"))
			sb.WriteString(fmt.Sprintf("        asyncio.run(%s(%s))\n", fn.Name, pyArgList(fn.Params)))
		} else {
			sb.WriteString(fmt.Sprintf("    with pytest.raises(Exception):\n"))
			sb.WriteString(fmt.Sprintf("        %s(%s)\n", fn.Name, pyArgList(fn.Params)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func genPytestClassTest(cls pyClassInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("class Test%s:\n", cls.Name))

	// 实例化测试
	sb.WriteString("    def test_init(self):\n")
	sb.WriteString(fmt.Sprintf("        instance = %s()\n", cls.Name))
	sb.WriteString("        assert instance is not None\n\n")

	// 方法测试
	for _, method := range cls.Methods {
		if method.Name == "__init__" {
			continue
		}

		testName := method.Name
		if method.IsStatic {
			sb.WriteString(fmt.Sprintf("    def test_%s(self):\n", testName))
			if method.IsAsync {
				sb.WriteString(fmt.Sprintf("        result = asyncio.run(%s.%s(%s))\n", cls.Name, method.Name, pyArgList(method.Params)))
			} else {
				sb.WriteString(fmt.Sprintf("        result = %s.%s(%s)\n", cls.Name, method.Name, pyArgList(method.Params)))
			}
			sb.WriteString(genPyResultAssertion(method.Analysis, "        "))
		} else {
			sb.WriteString(fmt.Sprintf("    def test_%s(self):\n", testName))
			sb.WriteString(fmt.Sprintf("        instance = %s()\n", cls.Name))
			if method.IsAsync {
				sb.WriteString(fmt.Sprintf("        result = asyncio.run(instance.%s(%s))\n", method.Name, pyArgList(method.Params)))
			} else {
				sb.WriteString(fmt.Sprintf("        result = instance.%s(%s)\n", method.Name, pyArgList(method.Params)))
			}
			sb.WriteString(genPyResultAssertion(method.Analysis, "        "))
		}

		// 错误用例
		if method.Analysis.Raises {
			sb.WriteString(fmt.Sprintf("\n    def test_%s_raises(self):\n", method.Name))
			if method.IsStatic {
				if method.IsAsync {
					sb.WriteString("        with pytest.raises(Exception):\n")
					sb.WriteString(fmt.Sprintf("            asyncio.run(%s.%s(%s))\n", cls.Name, method.Name, pyArgList(method.Params)))
				} else {
					sb.WriteString("        with pytest.raises(Exception):\n")
					sb.WriteString(fmt.Sprintf("            %s.%s(%s)\n", cls.Name, method.Name, pyArgList(method.Params)))
				}
			} else {
				sb.WriteString(fmt.Sprintf("        instance = %s()\n", cls.Name))
				if method.IsAsync {
					sb.WriteString("        with pytest.raises(Exception):\n")
					sb.WriteString(fmt.Sprintf("            asyncio.run(instance.%s(%s))\n", method.Name, pyArgList(method.Params)))
				} else {
					sb.WriteString("        with pytest.raises(Exception):\n")
					sb.WriteString(fmt.Sprintf("            instance.%s(%s)\n", method.Name, pyArgList(method.Params)))
				}
			}
			sb.WriteString("\n")
		}

		// 边界用例
		for _, b := range method.Analysis.Boundaries {
			if !pyParamExists(method.Params, b.Param) {
				continue
			}
			sb.WriteString(fmt.Sprintf("\n    def test_%s_%s_boundary(self):\n", method.Name, b.Param))
			args := pyArgListWithBoundary(method.Params, b)
			if method.IsStatic {
				if method.IsAsync {
					sb.WriteString(fmt.Sprintf("        result = asyncio.run(%s.%s(%s))\n", cls.Name, method.Name, args))
				} else {
					sb.WriteString(fmt.Sprintf("        result = %s.%s(%s)\n", cls.Name, method.Name, args))
				}
			} else {
				sb.WriteString(fmt.Sprintf("        instance = %s()\n", cls.Name))
				if method.IsAsync {
					sb.WriteString(fmt.Sprintf("        result = asyncio.run(instance.%s(%s))\n", method.Name, args))
				} else {
					sb.WriteString(fmt.Sprintf("        result = instance.%s(%s)\n", method.Name, args))
				}
			}
			sb.WriteString(genPyResultAssertion(method.Analysis, "        "))
		}
	}

	return sb.String()
}

// genPyResultAssertion 根据返回类型生成断言
func genPyResultAssertion(a pyFuncAnalysis, indent string) string {
	var sb strings.Builder

	if !a.HasReturn {
		sb.WriteString(indent + "# void function, verify no exception\n")
		return sb.String()
	}

	switch a.ReturnType {
	case "int":
		sb.WriteString(indent + "assert isinstance(result, int)\n")
	case "float":
		sb.WriteString(indent + "assert isinstance(result, (int, float))\n")
	case "str":
		sb.WriteString(indent + "assert isinstance(result, str)\n")
	case "bool":
		sb.WriteString(indent + "assert isinstance(result, bool)\n")
	case "list":
		sb.WriteString(indent + "assert isinstance(result, list)\n")
	case "dict":
		sb.WriteString(indent + "assert isinstance(result, dict)\n")
	case "tuple":
		sb.WriteString(indent + "assert isinstance(result, tuple)\n")
	case "None":
		sb.WriteString(indent + "assert result is None\n")
	default:
		sb.WriteString(indent + "assert result is not None\n")
	}

	return sb.String()
}

// pyParamExists 检查参数名是否存在
func pyParamExists(params []pyParamInfo, name string) bool {
	for _, p := range params {
		if p.Name == name {
			return true
		}
	}
	return false
}

// pyArgListWithBoundary 生成带边界值的参数列表
func pyArgListWithBoundary(params []pyParamInfo, b pyBoundary) string {
	if len(params) == 0 {
		return ""
	}
	args := make([]string, len(params))
	for i, p := range params {
		if p.Name == b.Param {
			args[i] = b.Value
		} else if p.IsArgs {
			args[i] = "()"
		} else if p.IsKwargs {
			args[i] = "{}"
		} else {
			args[i] = "None"
		}
	}
	return strings.Join(args, ", ")
}

// pyArgList 生成调用参数列表（用 None 占位）
func pyArgList(params []pyParamInfo) string {
	if len(params) == 0 {
		return ""
	}
	args := make([]string, len(params))
	for i, p := range params {
		if p.IsArgs {
			args[i] = "()"
		} else if p.IsKwargs {
			args[i] = "{}"
		} else {
			args[i] = "None"
		}
	}
	return strings.Join(args, ", ")
}

func pyDefaultArgs(params []pyParamInfo) string {
	return pyArgList(params)
}

// ---- 辅助函数 ----

func isPyDunder(name string) bool {
	return strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__")
}

func isPyTestHelper(name string) bool {
	switch name {
	case "setUp", "tearDown", "setUpClass", "tearDownClass":
		return true
	}
	return false
}

func indentLevel(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count += 4
		} else {
			break
		}
	}
	return count
}

func joinPyExportNames(funcs []pyFuncInfo, classes []pyClassInfo) string {
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
