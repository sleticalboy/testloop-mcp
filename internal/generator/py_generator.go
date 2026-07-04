package generator

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ---- 类型定义 ----

type pyFuncInfo struct {
	Name      string
	Params    []pyParamInfo
	IsAsync   bool
	IsMethod  bool
	ClassName string
	IsStatic  bool
	Body      string         // 函数体源码
	Analysis  pyFuncAnalysis // 函数体分析结果
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

// pyClassInfo 类信息
type pyClassInfo struct {
	Name    string
	Methods []pyFuncInfo
}

// ---- 核心函数 ----

// GeneratePytestTests 读取 Python 源文件，用 tree-sitter 解析后生成 pytest 测试代码
func GeneratePytestTests(srcPath string) (string, error) {
	source, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("读取源文件失败: %w", err)
	}

	funcs, classes := parsePyWithTreeSitter(source)

	if len(funcs) == 0 && len(classes) == 0 {
		return "# 未发现需要生成测试的函数或类", nil
	}

	moduleName := stripExt(baseName(srcPath))

	var buf strings.Builder

	// 检测是否有 async 函数
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

	for _, fn := range funcs {
		buf.WriteString(genPytestFuncTest(fn))
	}

	for _, cls := range classes {
		buf.WriteString(genPytestClassTest(cls))
	}

	return buf.String(), nil
}

// ---- 函数体分析（基于 body 文本字符串，不依赖解析方式） ----

var (
	pyReturnRe = regexp.MustCompile(`\breturn\s+(.+)`)
	pyRaiseRe  = regexp.MustCompile(`\braise\b`)
	pyIfEqRe   = regexp.MustCompile(`if\s+(\w+)\s*(?:==?|!=)\s*(.+?):`)
	pyIfNoneRe = regexp.MustCompile(`if\s+(\w+)\s+(?:is|is\s+not)\s+(None|True|False)\s*:`)
)

func analyzePyBody(body string) pyFuncAnalysis {
	a := pyFuncAnalysis{}

	if body == "" {
		return a
	}

	a.Raises = pyRaiseRe.MatchString(body)

	returnMatches := pyReturnRe.FindAllStringSubmatch(body, -1)
	a.HasReturn = len(returnMatches) > 0

	if a.HasReturn {
		a.ReturnType = inferPyReturnType(returnMatches)
	}

	a.Boundaries = extractPyBoundaries(body)

	return a
}

func inferPyReturnType(matches [][]string) string {
	for _, m := range matches {
		expr := strings.TrimSpace(m[1])

		if expr == "None" {
			return "None"
		}
		if expr == "True" || expr == "False" {
			return "bool"
		}
		if (strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"")) ||
			(strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'")) {
			return "str"
		}
		if strings.HasPrefix(expr, "f\"") || strings.HasPrefix(expr, "f'") {
			return "str"
		}
		if isPyNumericLiteral(expr) {
			if strings.Contains(expr, ".") {
				return "float"
			}
			return "int"
		}
		if strings.HasPrefix(expr, "[") {
			return "list"
		}
		if strings.HasPrefix(expr, "{") {
			return "dict"
		}
		if strings.HasPrefix(expr, "(") {
			return "tuple"
		}
		if strings.Contains(expr, ".json()") {
			return "dict"
		}
		if isPyArithmeticExpr(expr) {
			if strings.Contains(expr, " / ") || strings.Contains(expr, " // ") {
				return "float"
			}
			return "int"
		}
		if strings.Contains(expr, " + ") && hasPyStringLiteral(expr) {
			return "str"
		}
		if strings.Contains(expr, ".join(") {
			return "str"
		}
	}

	return "unknown"
}

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

func extractPyBoundaries(body string) []pyBoundary {
	var boundaries []pyBoundary
	seen := make(map[string]bool)

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

	ifMatches := pyIfEqRe.FindAllStringSubmatch(body, -1)
	for _, m := range ifMatches {
		param := m[1]
		val := strings.TrimSpace(m[2])

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

// ---- 测试生成 ----

func genPytestFuncTest(fn pyFuncInfo) string {
	var sb strings.Builder

	if fn.IsAsync {
		sb.WriteString(fmt.Sprintf("def test_%s():\n", fn.Name))
		sb.WriteString(fmt.Sprintf("    result = asyncio.run(%s(%s))\n", fn.Name, pyArgList(fn.Params)))
	} else {
		sb.WriteString(fmt.Sprintf("def test_%s():\n", fn.Name))
		sb.WriteString(fmt.Sprintf("    result = %s(%s)\n", fn.Name, pyArgList(fn.Params)))
	}
	sb.WriteString(genPyResultAssertion(fn.Analysis, "    "))

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

	sb.WriteString("    def test_init(self):\n")
	sb.WriteString(fmt.Sprintf("        instance = %s()\n", cls.Name))
	sb.WriteString("        assert instance is not None\n\n")

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

// ---- 辅助函数 ----

func pyParamExists(params []pyParamInfo, name string) bool {
	for _, p := range params {
		if p.Name == name {
			return true
		}
	}
	return false
}

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
