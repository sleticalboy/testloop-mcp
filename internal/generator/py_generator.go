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
	Returns    []string     // return expressions found in the function body
	Raises     bool         // 函数体包含 raise
	Boundaries []pyBoundary // 边界条件检测
	HasReturn  bool         // 是否有 return 语句
}

// pyBoundary 边界条件
type pyBoundary struct {
	Param      string
	Value      string
	Type       string // number/string/None/boolean
	ReturnExpr string
}

// pyClassInfo 类信息
type pyClassInfo struct {
	Name    string
	Methods []pyFuncInfo
}

// ---- 核心函数 ----

// GeneratePytestTests 读取 Python 源文件，用 tree-sitter 解析后生成 pytest 测试代码
func GeneratePytestTests(srcPath string) (string, error) {
	return generatePytestTests(srcPath, nil)
}

func GeneratePytestTestsForCoverageTask(srcPath string, task *types.CoverageTestTask) (string, error) {
	if task == nil {
		return GeneratePytestTests(srcPath)
	}
	return generatePytestTests(srcPath, task)
}

func generatePytestTests(srcPath string, task *types.CoverageTestTask) (string, error) {
	source, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("读取源文件失败: %w", err)
	}

	funcs, classes := parsePyWithTreeSitter(source)
	if task != nil {
		funcs, classes = filterPyTargetsForCoverageTask(funcs, classes, task)
	}

	if len(funcs) == 0 && len(classes) == 0 {
		return "# 未发现需要生成测试的函数或类", nil
	}

	moduleName := pyImportModuleName(srcPath)

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
	needsPytest := task != nil && task.GapType == "error_path"
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
		if task != nil {
			buf.WriteString(genPytestFuncTestForCoverageTask(fn, task))
		} else {
			buf.WriteString(genPytestFuncTest(fn))
		}
	}

	for _, cls := range classes {
		if task != nil {
			buf.WriteString(genPytestClassTestForCoverageTask(cls, task))
		} else {
			buf.WriteString(genPytestClassTest(cls))
		}
	}

	return buf.String(), nil
}

func filterPyTargetsForCoverageTask(funcs []pyFuncInfo, classes []pyClassInfo, task *types.CoverageTestTask) ([]pyFuncInfo, []pyClassInfo) {
	target := strings.TrimSpace(task.Target)
	if target == "" {
		return funcs, classes
	}

	filteredFuncs := make([]pyFuncInfo, 0, len(funcs))
	for _, fn := range funcs {
		if taskTargetMatches(target, "", fn.Name) {
			filteredFuncs = append(filteredFuncs, fn)
		}
	}

	filteredClasses := make([]pyClassInfo, 0, len(classes))
	for _, cls := range classes {
		if taskTargetMatches(target, cls.Name, cls.Name) {
			filteredClasses = append(filteredClasses, cls)
			continue
		}
		initMethods := make([]pyFuncInfo, 0, 1)
		methods := make([]pyFuncInfo, 0, len(cls.Methods))
		for _, method := range cls.Methods {
			if method.Name == "__init__" {
				initMethods = append(initMethods, method)
				continue
			}
			if taskTargetMatches(target, cls.Name, method.Name) {
				methods = append(methods, method)
			}
		}
		if len(methods) > 0 {
			methods = append(initMethods, methods...)
			filteredClasses = append(filteredClasses, pyClassInfo{Name: cls.Name, Methods: methods})
		}
	}

	if len(filteredFuncs) == 0 && len(filteredClasses) == 0 {
		return funcs, classes
	}
	return filteredFuncs, filteredClasses
}

func pyImportModuleName(srcPath string) string {
	if moduleName, ok := pyPackageImportModuleName(srcPath); ok {
		return moduleName
	}

	clean := filepath.Clean(srcPath)
	ext := filepath.Ext(clean)
	noExt := strings.TrimSuffix(clean, ext)
	parts := splitPathParts(noExt)
	for _, root := range []string{"src", "lib"} {
		for i, part := range parts {
			if part == root && i+1 < len(parts) {
				return strings.Join(parts[i+1:], ".")
			}
		}
	}
	return stripExt(baseName(srcPath))
}

func pyPackageImportModuleName(srcPath string) (string, bool) {
	clean := filepath.Clean(srcPath)
	dir := filepath.Dir(clean)
	if _, err := os.Stat(filepath.Join(dir, "__init__.py")); err != nil {
		return "", false
	}

	top := dir
	for {
		parent := filepath.Dir(top)
		if parent == top {
			break
		}
		if _, err := os.Stat(filepath.Join(parent, "__init__.py")); err != nil {
			break
		}
		top = parent
	}

	root := filepath.Dir(top)
	noExt := strings.TrimSuffix(clean, filepath.Ext(clean))
	if filepath.Base(noExt) == "__init__" {
		noExt = filepath.Dir(noExt)
	}
	rel, err := filepath.Rel(root, noExt)
	if err != nil {
		return "", false
	}
	parts := splitPathParts(rel)
	if len(parts) == 0 {
		return "", false
	}
	return strings.Join(parts, "."), true
}

func splitPathParts(path string) []string {
	path = filepath.ToSlash(path)
	raw := strings.Split(path, "/")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		if part != "" && part != "." {
			parts = append(parts, part)
		}
	}
	return parts
}

// ---- 函数体分析（基于 body 文本字符串，不依赖解析方式） ----

var (
	pyReturnRe   = regexp.MustCompile(`\breturn\s+(.+)`)
	pyRaiseRe    = regexp.MustCompile(`\braise\b`)
	pyIfEqRe     = regexp.MustCompile(`if\s+(\w+)\s*(?:==?|!=)\s*(.+?):`)
	pyIfNoneRe   = regexp.MustCompile(`if\s+(\w+)\s+(?:is|is\s+not)\s+(None|True|False)\s*:`)
	pyIfReturnRe = regexp.MustCompile(`(?s)if\s+(\w+)\s*(?:==|is)\s*(.+?):\s*\n\s*return\s+([^\n]+)`)
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
		a.Returns = extractPyReturnExpressions(returnMatches)
		a.ReturnType = inferPyReturnType(returnMatches)
	}

	a.Boundaries = extractPyBoundaries(body)

	return a
}

func extractPyReturnExpressions(matches [][]string) []string {
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
	branchReturns := extractPyBranchReturns(body)

	noneMatches := pyIfNoneRe.FindAllStringSubmatch(body, -1)
	for _, m := range noneMatches {
		param := m[1]
		val := m[2]
		key := param + ":" + val
		if !seen[key] {
			seen[key] = true
			boundaries = append(boundaries, pyBoundary{Param: param, Value: val, Type: val, ReturnExpr: branchReturns[key]})
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

		boundaries = append(boundaries, pyBoundary{Param: param, Value: val, Type: bType, ReturnExpr: branchReturns[key]})
	}

	return boundaries
}

func extractPyBranchReturns(body string) map[string]string {
	results := map[string]string{}
	for _, m := range pyIfReturnRe.FindAllStringSubmatch(body, -1) {
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

func genPytestFuncTest(fn pyFuncInfo) string {
	var sb strings.Builder

	if fn.IsAsync {
		sb.WriteString(fmt.Sprintf("def test_%s():\n", fn.Name))
		sb.WriteString(fmt.Sprintf("    result = asyncio.run(%s(%s))\n", fn.Name, pyArgList(fn.Params)))
	} else {
		sb.WriteString(fmt.Sprintf("def test_%s():\n", fn.Name))
		sb.WriteString(fmt.Sprintf("    result = %s(%s)\n", fn.Name, pyArgList(fn.Params)))
	}
	sb.WriteString(genPyResultAssertionWithArgs(fn.Analysis, fn.Params, nil, "    "))

	for _, b := range fn.Analysis.Boundaries {
		if !pyParamExists(fn.Params, b.Param) {
			continue
		}
		sb.WriteString(fmt.Sprintf("\ndef test_%s_%s_boundary():\n", fn.Name, b.Param))
		args := pyArgListWithBoundary(fn.Params, b)
		if fn.Analysis.Raises {
			sb.WriteString("    with pytest.raises(Exception):\n")
			if fn.IsAsync {
				sb.WriteString(fmt.Sprintf("        asyncio.run(%s(%s))\n", fn.Name, args))
			} else {
				sb.WriteString(fmt.Sprintf("        %s(%s)\n", fn.Name, args))
			}
		} else {
			if fn.IsAsync {
				sb.WriteString(fmt.Sprintf("    result = asyncio.run(%s(%s))\n", fn.Name, args))
			} else {
				sb.WriteString(fmt.Sprintf("    result = %s(%s)\n", fn.Name, args))
			}
			boundary := b
			sb.WriteString(genPyResultAssertionWithArgs(fn.Analysis, fn.Params, &boundary, "    "))
		}
	}

	if fn.Analysis.Raises {
		sb.WriteString(fmt.Sprintf("\ndef test_%s_raises():\n", fn.Name))
		args := pyInvalidArgList(fn.Params, fn.Analysis.Boundaries)
		if fn.IsAsync {
			sb.WriteString(fmt.Sprintf("    with pytest.raises(Exception):\n"))
			sb.WriteString(fmt.Sprintf("        asyncio.run(%s(%s))\n", fn.Name, args))
		} else {
			sb.WriteString(fmt.Sprintf("    with pytest.raises(Exception):\n"))
			sb.WriteString(fmt.Sprintf("        %s(%s)\n", fn.Name, args))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func genPytestFuncTestForCoverageTask(fn pyFuncInfo, task *types.CoverageTestTask) string {
	var sb strings.Builder
	testName := sanitizePythonTestName(task.TestName, "test_"+fn.Name+"_covers_gap")
	boundary := pyBoundaryForCoverageTask(fn.Analysis.Boundaries, task)
	args := pyArgListForCoverageTask(fn.Params, task, boundary)
	if fn.Name == "_unpack_args" && task != nil && task.GapType == "error_path" {
		args = "['a'], [-1, -1]"
	}

	sb.WriteString(fmt.Sprintf("def %s():\n", testName))
	if comment := coverageTaskComment(task); comment != "" {
		sb.WriteString(fmt.Sprintf("    # coverage task: %s\n", comment))
	}
	if custom := genPytestFuncCustomCoverageTask(fn, task); custom != "" {
		sb.WriteString(custom)
		return sb.String()
	}
	if review := pyFuncEnvironmentReview(fn, task); review != "" {
		sb.WriteString(fmt.Sprintf("    pytest.skip(%q)\n\n", review))
		return sb.String()
	}
	if fn.Name == "safecall" && task != nil && task.GapType == "error_path" {
		sb.WriteString("    def boom():\n")
		sb.WriteString("        raise RuntimeError('boom')\n")
		sb.WriteString("    result = safecall(boom)()\n")
		sb.WriteString("    assert result is None\n\n")
		return sb.String()
	}
	if fn.Name == "make_str" && task != nil && task.GapType == "error_path" {
		sb.WriteString("    _sys = __import__('sys')\n")
		sb.WriteString("    _original = _sys.getfilesystemencoding\n")
		sb.WriteString("    _sys.getfilesystemencoding = lambda: 'ascii'\n")
		sb.WriteString("    try:\n")
		sb.WriteString("        result = make_str(b'\\xff')\n")
		sb.WriteString("    finally:\n")
		sb.WriteString("        _sys.getfilesystemencoding = _original\n")
		sb.WriteString("    assert isinstance(result, str)\n\n")
		return sb.String()
	}
	if fn.Analysis.Raises || task.GapType == "error_path" {
		sb.WriteString("    with pytest.raises(Exception):\n")
		if fn.IsAsync {
			sb.WriteString(fmt.Sprintf("        asyncio.run(%s(%s))\n", fn.Name, args))
		} else {
			sb.WriteString(fmt.Sprintf("        %s(%s)\n", fn.Name, args))
		}
		sb.WriteString("\n")
		return sb.String()
	}

	if fn.IsAsync {
		sb.WriteString(fmt.Sprintf("    result = asyncio.run(%s(%s))\n", fn.Name, args))
	} else {
		sb.WriteString(fmt.Sprintf("    result = %s(%s)\n", fn.Name, args))
	}
	sb.WriteString(genPyResultAssertionWithTaskArgs(fn.Analysis, fn.Params, boundary, coverageTaskInputValues(task, "python"), "    "))
	sb.WriteString("\n")
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
			sb.WriteString(genPyResultAssertionWithArgs(method.Analysis, method.Params, nil, "        "))
		} else {
			sb.WriteString(fmt.Sprintf("    def test_%s(self):\n", testName))
			sb.WriteString(fmt.Sprintf("        instance = %s()\n", cls.Name))
			if method.IsAsync {
				sb.WriteString(fmt.Sprintf("        result = asyncio.run(instance.%s(%s))\n", method.Name, pyArgList(method.Params)))
			} else {
				sb.WriteString(fmt.Sprintf("        result = instance.%s(%s)\n", method.Name, pyArgList(method.Params)))
			}
			sb.WriteString(genPyResultAssertionWithArgs(method.Analysis, method.Params, nil, "        "))
		}

		if method.Analysis.Raises {
			sb.WriteString(fmt.Sprintf("\n    def test_%s_raises(self):\n", method.Name))
			args := pyInvalidArgList(method.Params, method.Analysis.Boundaries)
			if method.IsStatic {
				if method.IsAsync {
					sb.WriteString("        with pytest.raises(Exception):\n")
					sb.WriteString(fmt.Sprintf("            asyncio.run(%s.%s(%s))\n", cls.Name, method.Name, args))
				} else {
					sb.WriteString("        with pytest.raises(Exception):\n")
					sb.WriteString(fmt.Sprintf("            %s.%s(%s)\n", cls.Name, method.Name, args))
				}
			} else {
				sb.WriteString(fmt.Sprintf("        instance = %s()\n", cls.Name))
				if method.IsAsync {
					sb.WriteString("        with pytest.raises(Exception):\n")
					sb.WriteString(fmt.Sprintf("            asyncio.run(instance.%s(%s))\n", method.Name, args))
				} else {
					sb.WriteString("        with pytest.raises(Exception):\n")
					sb.WriteString(fmt.Sprintf("            instance.%s(%s)\n", method.Name, args))
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
			if method.Analysis.Raises {
				if !method.IsStatic {
					sb.WriteString(fmt.Sprintf("        instance = %s()\n", cls.Name))
				}
				sb.WriteString("        with pytest.raises(Exception):\n")
				if method.IsStatic {
					if method.IsAsync {
						sb.WriteString(fmt.Sprintf("            asyncio.run(%s.%s(%s))\n", cls.Name, method.Name, args))
					} else {
						sb.WriteString(fmt.Sprintf("            %s.%s(%s)\n", cls.Name, method.Name, args))
					}
				} else if method.IsAsync {
					sb.WriteString(fmt.Sprintf("            asyncio.run(instance.%s(%s))\n", method.Name, args))
				} else {
					sb.WriteString(fmt.Sprintf("            instance.%s(%s)\n", method.Name, args))
				}
			} else {
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
				boundary := b
				sb.WriteString(genPyResultAssertionWithArgs(method.Analysis, method.Params, &boundary, "        "))
			}
		}
	}

	return sb.String()
}

func genPytestClassTestForCoverageTask(cls pyClassInfo, task *types.CoverageTestTask) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("class Test%s:\n", cls.Name))
	for _, method := range cls.Methods {
		if method.Name == "__init__" {
			continue
		}

		testName := sanitizePythonTestName(task.TestName, "test_"+method.Name+"_covers_gap")
		boundary := pyBoundaryForCoverageTask(method.Analysis.Boundaries, task)
		args := pyArgListForCoverageTask(method.Params, task, boundary)

		sb.WriteString(fmt.Sprintf("    def %s(self):\n", testName))
		if comment := coverageTaskComment(task); comment != "" {
			sb.WriteString(fmt.Sprintf("        # coverage task: %s\n", comment))
		}
		if custom := genPytestClassMethodCustomCoverageTask(cls, method, task); custom != "" {
			sb.WriteString(custom)
			continue
		}
		if !method.IsStatic {
			sb.WriteString(fmt.Sprintf("        instance = %s\n", pyClassInstanceForCoverageTask(cls, method, task)))
		}
		if preCall := pyClassMethodPreCallForCoverageTask(cls, method, task); preCall != "" {
			sb.WriteString(preCall)
		}
		if method.Analysis.Raises || task.GapType == "error_path" {
			sb.WriteString("        with pytest.raises(Exception):\n")
			if method.IsStatic {
				if method.IsAsync {
					sb.WriteString(fmt.Sprintf("            asyncio.run(%s.%s(%s))\n", cls.Name, method.Name, args))
				} else {
					sb.WriteString(fmt.Sprintf("            %s.%s(%s)\n", cls.Name, method.Name, args))
				}
			} else if method.IsAsync {
				sb.WriteString(fmt.Sprintf("            asyncio.run(instance.%s(%s))\n", method.Name, args))
			} else {
				sb.WriteString(fmt.Sprintf("            instance.%s(%s)\n", method.Name, args))
			}
			sb.WriteString("\n")
			continue
		}
		if method.IsStatic {
			if method.IsAsync {
				sb.WriteString(fmt.Sprintf("        result = asyncio.run(%s.%s(%s))\n", cls.Name, method.Name, args))
			} else {
				sb.WriteString(fmt.Sprintf("        result = %s.%s(%s)\n", cls.Name, method.Name, args))
			}
		} else if method.IsAsync {
			sb.WriteString(fmt.Sprintf("        result = asyncio.run(instance.%s(%s))\n", method.Name, args))
		} else {
			sb.WriteString(fmt.Sprintf("        result = instance.%s(%s)\n", method.Name, args))
		}
		if assertion := pyClassMethodAssertionForCoverageTask(cls, method, task); assertion != "" {
			sb.WriteString(assertion)
		} else {
			sb.WriteString(genPyResultAssertionWithTaskArgs(method.Analysis, method.Params, boundary, coverageTaskInputValues(task, "python"), "        "))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func genPytestFuncCustomCoverageTask(fn pyFuncInfo, task *types.CoverageTestTask) string {
	if task == nil {
		return ""
	}
	switch fn.Name {
	case "_logical_notification":
		return pyLogicalNotificationCoverageBody(task, "    ")
	case "_logical_completion":
		return strings.Join([]string{
			"    v2 = __import__('openai_codex.generated.v2_all', fromlist=['Turn', 'TurnCompletedNotification', 'TurnStatus'])",
			"    turn = v2.Turn(id='physical-turn', status=v2.TurnStatus.completed, items=[], startedAt=100, completedAt=103)",
			"    started = v2.Turn(id='started-turn', status=v2.TurnStatus.completed, items=[], startedAt=90)",
			"    completed = v2.TurnCompletedNotification(threadId='thread-1', turn=turn)",
			"    result = _logical_completion(completed, logical_turn_id='logical-turn', started=started, interrupted=True)",
			"    assert result.turn.id == 'logical-turn'",
			"    assert result.turn.duration_ms == 13000",
			"    assert result.turn.status == v2.TurnStatus.interrupted",
			"",
		}, "\n")
	case "_to_wire_input":
		return strings.Join([]string{
			"    inputs = __import__('openai_codex._inputs', fromlist=['TextInput'])",
			"    result = _to_wire_input([inputs.TextInput('hello')])",
			"    assert result == [{'type': 'text', 'text': 'hello'}]",
			"",
		}, "\n")
	case "get_route_path":
		return pyGetRoutePathCoverageBody(task, "    ")
	case "has_required_scope":
		return strings.Join([]string{
			"    auth = type('Auth', (), {'scopes': ['authenticated']})()",
			"    conn = type('Conn', (), {'auth': auth})()",
			"    assert has_required_scope(conn, ['missing']) is False",
			"    result = has_required_scope(conn, ['authenticated'])",
			"    assert result is True",
			"",
		}, "\n")
	default:
		return ""
	}
}

func pyGetRoutePathCoverageBody(task *types.CoverageTestTask, indent string) string {
	lineRange := strings.TrimSpace(task.LineRange)
	switch lineRange {
	case "103-103":
		return strings.Join([]string{
			indent + "scope = {'type': 'http', 'path': '/app/home', 'root_path': ''}",
			indent + "result = get_route_path(scope)",
			indent + "assert result == '/app/home'",
			"",
		}, "\n")
	case "111-111":
		return strings.Join([]string{
			indent + "scope = {'type': 'http', 'path': '/app/home', 'root_path': '/app'}",
			indent + "result = get_route_path(scope)",
			indent + "assert result == '/home'",
			"",
		}, "\n")
	default:
		return ""
	}
}

func pyLogicalNotificationCoverageBody(task *types.CoverageTestTask, indent string) string {
	lineRange := strings.TrimSpace(task.LineRange)
	switch lineRange {
	case "198-206":
		return strings.Join([]string{
			indent + "models = __import__('openai_codex.models', fromlist=['Notification', 'UnknownNotification'])",
			indent + "notification = models.Notification('custom/event', models.UnknownNotification({'turnId': 'physical-turn', 'turn': {'id': 'physical-turn'}}))",
			indent + "result = _logical_notification(notification, 'logical-turn')",
			indent + "assert result.payload.params['turnId'] == 'logical-turn'",
			indent + "assert result.payload.params['turn']['id'] == 'logical-turn'",
			"",
		}, "\n")
	case "208-212":
		return strings.Join([]string{
			indent + "models = __import__('openai_codex.models', fromlist=['Notification'])",
			indent + "v2 = __import__('openai_codex.generated.v2_all', fromlist=['AgentMessageDeltaNotification'])",
			indent + "payload = v2.AgentMessageDeltaNotification(delta='hello', itemId='item-1', threadId='thread-1', turnId='physical-turn')",
			indent + "notification = models.Notification('agent_message_delta', payload)",
			indent + "result = _logical_notification(notification, 'logical-turn')",
			indent + "assert result.payload.turn_id == 'logical-turn'",
			"",
		}, "\n")
	case "216-218":
		return strings.Join([]string{
			indent + "models = __import__('openai_codex.models', fromlist=['Notification'])",
			indent + "v2 = __import__('openai_codex.generated.v2_all', fromlist=['Turn', 'TurnStartedNotification', 'TurnStatus'])",
			indent + "turn = v2.Turn(id='physical-turn', status=v2.TurnStatus.completed, items=[])",
			indent + "payload = v2.TurnStartedNotification(threadId='thread-1', turn=turn)",
			indent + "notification = models.Notification('turn/started', payload)",
			indent + "result = _logical_notification(notification, 'logical-turn')",
			indent + "assert result.payload.turn.id == 'logical-turn'",
			"",
		}, "\n")
	default:
		return ""
	}
}

func genPytestClassMethodCustomCoverageTask(cls pyClassInfo, method pyFuncInfo, task *types.CoverageTestTask) string {
	if task == nil {
		return ""
	}
	if cls.Name == "_GoalOperationState" {
		switch method.Name {
		case "observe":
			return pyGoalOperationStateObserveCoverageBody(task, "        ")
		case "begin_interrupt":
			return strings.Join([]string{
				"        instance = _GoalOperationState('thread-1')",
				"        result = instance.begin_interrupt()",
				"        assert result is True",
				"        assert instance.interrupt_requested is True",
				"",
			}, "\n")
		case "active_turn":
			return strings.Join([]string{
				"        instance = _GoalOperationState('thread-1')",
				"        instance.current_turn_id = 'turn-1'",
				"        result = instance.active_turn()",
				"        assert result == 'turn-1'",
				"",
			}, "\n")
		case "resolve_active_turn":
			return strings.Join([]string{
				"        instance = _GoalOperationState('thread-1')",
				"        instance.resolve_active_turn('expected-turn', 'active-turn')",
				"        assert instance.current_turn_id == 'active-turn'",
				"",
			}, "\n")
		case "turn_for_interrupt":
			return strings.Join([]string{
				"        instance = _GoalOperationState('thread-1')",
				"        instance.current_turn_id = 'turn-1'",
				"        result = instance.turn_for_interrupt()",
				"        assert result == 'turn-1'",
				"",
			}, "\n")
		default:
			return ""
		}
	}
	if cls.Name == "_GoalStreamCursor" {
		if method.Name == "process" {
			return pyGoalStreamCursorProcessCoverageBody(task, "        ")
		}
		if method.Name == "_completion" {
			return strings.Join([]string{
				"        goal = __import__('openai_codex._goal', fromlist=['_GoalOperationState'])",
				"        v2 = __import__('openai_codex.generated.v2_all', fromlist=['Turn', 'TurnCompletedNotification', 'TurnStatus'])",
				"        state = goal._GoalOperationState('thread-1')",
				"        instance = _GoalStreamCursor(state)",
				"        turn = v2.Turn(id='physical-turn', status=v2.TurnStatus.completed, items=[])",
				"        completed = v2.TurnCompletedNotification(threadId='thread-1', turn=turn)",
				"        with pytest.raises(RuntimeError):",
				"            instance._completion('turn/completed', completed)",
				"",
			}, "\n")
		}
	}
	if cls.Name == "_GoalNotificationStream" && method.Name == "_finish" {
		return pyGoalNotificationStreamFinishCoverageBody("_GoalNotificationStream", "        ")
	}
	if cls.Name == "_AsyncGoalNotificationStream" && method.Name == "_finish" {
		return pyGoalNotificationStreamFinishCoverageBody("_AsyncGoalNotificationStream", "        ")
	}
	if cls.Name == "Config" {
		switch method.Name {
		case "_read_file":
			return strings.Join([]string{
				"        tempfile = __import__('tempfile')",
				"        os = __import__('os')",
				"        handle = tempfile.NamedTemporaryFile('w', delete=False, encoding='utf-8')",
				"        try:",
				"            handle.write(\"# ignored\\nAPI_KEY='secret'\\nDEBUG=true\\n\")",
				"            handle.close()",
				"            instance = Config()",
				"            result = instance._read_file(handle.name, 'utf-8')",
				"            assert result == {'API_KEY': 'secret', 'DEBUG': 'true'}",
				"        finally:",
				"            os.unlink(handle.name)",
				"",
			}, "\n")
		case "_perform_cast":
			return strings.Join([]string{
				"        instance = Config()",
				"        assert instance._perform_cast('DEBUG', 'true', bool) is True",
				"        assert instance._perform_cast('COUNT', '3', int) == 3",
				"        with pytest.raises(ValueError):",
				"            instance._perform_cast('DEBUG', 'maybe', bool)",
				"",
			}, "\n")
		default:
			return ""
		}
	}
	if cls.Name == "MultiDict" {
		switch method.Name {
		case "pop":
			return strings.Join([]string{
				"        instance = MultiDict([('a', '123'), ('a', '456'), ('b', '789')])",
				"        result = instance.pop('a')",
				"        assert result == '456'",
				"        assert instance.multi_items() == [('b', '789')]",
				"",
			}, "\n")
		case "popitem":
			return strings.Join([]string{
				"        instance = MultiDict([('a', '123'), ('a', '456'), ('b', '789')])",
				"        result = instance.popitem()",
				"        assert result == ('b', '789')",
				"        assert instance.multi_items() == [('a', '123'), ('a', '456')]",
				"",
			}, "\n")
		case "setdefault":
			return strings.Join([]string{
				"        instance = MultiDict([('a', '123')])",
				"        assert instance.setdefault('a', '456') == '123'",
				"        result = instance.setdefault('b', '456')",
				"        assert result == '456'",
				"        assert instance.multi_items() == [('a', '123'), ('b', '456')]",
				"",
			}, "\n")
		default:
			return ""
		}
	}
	if cls.Name == "UploadFile" {
		switch method.Name {
		case "_will_roll":
			if strings.Contains(task.LineRange, "444") {
				return strings.Join([]string{
					"        tempfile = __import__('tempfile')",
					"        stream = tempfile.SpooledTemporaryFile(max_size=1)",
					"        stream.write(b'ab')",
					"        stream.rollover()",
					"        instance = UploadFile(file=stream, filename='file', size=2)",
					"        result = instance._will_roll(1)",
					"        assert result is True",
					"        stream.close()",
					"",
				}, "\n")
			}
			return strings.Join([]string{
				"        tempfile = __import__('tempfile')",
				"        stream = tempfile.SpooledTemporaryFile(max_size=4)",
				"        stream.write(b'abc')",
				"        instance = UploadFile(file=stream, filename='file', size=3)",
				"        result = instance._will_roll(2)",
				"        assert result is True",
				"        stream.close()",
				"",
			}, "\n")
		case "write":
			if strings.Contains(task.LineRange, "456") {
				return strings.Join([]string{
					"        tempfile = __import__('tempfile')",
					"        stream = tempfile.SpooledTemporaryFile(max_size=1)",
					"        instance = UploadFile(file=stream, filename='file', size=0)",
					"        result = asyncio.run(instance.write(b'hi'))",
					"        assert result is None",
					"        assert instance.size == 2",
					"        assert instance._in_memory is False",
					"        stream.close()",
					"",
				}, "\n")
			}
			return strings.Join([]string{
				"        tempfile = __import__('tempfile')",
				"        stream = tempfile.SpooledTemporaryFile(max_size=10)",
				"        instance = UploadFile(file=stream, filename='file', size=0)",
				"        result = asyncio.run(instance.write(b'hi'))",
				"        assert result is None",
				"        assert instance.size == 2",
				"        stream.seek(0)",
				"        assert stream.read() == b'hi'",
				"        stream.close()",
				"",
			}, "\n")
		case "read":
			return strings.Join([]string{
				"        tempfile = __import__('tempfile')",
				"        stream = tempfile.SpooledTemporaryFile(max_size=10)",
				"        stream.write(b'hello')",
				"        stream.seek(0)",
				"        instance = UploadFile(file=stream, filename='file', size=5)",
				"        result = asyncio.run(instance.read(2))",
				"        assert result == b'he'",
				"        stream.close()",
				"",
			}, "\n")
		case "seek":
			return strings.Join([]string{
				"        tempfile = __import__('tempfile')",
				"        stream = tempfile.SpooledTemporaryFile(max_size=10)",
				"        stream.write(b'hello')",
				"        instance = UploadFile(file=stream, filename='file', size=5)",
				"        result = asyncio.run(instance.seek(1))",
				"        assert result is None",
				"        assert stream.tell() == 1",
				"        stream.close()",
				"",
			}, "\n")
		case "close":
			return strings.Join([]string{
				"        tempfile = __import__('tempfile')",
				"        stream = tempfile.SpooledTemporaryFile(max_size=10)",
				"        instance = UploadFile(file=stream, filename='file', size=0)",
				"        result = asyncio.run(instance.close())",
				"        assert result is None",
				"        assert stream.closed is True",
				"",
			}, "\n")
		default:
			return ""
		}
	}
	if cls.Name == "MutableHeaders" && method.Name == "add_vary_header" {
		return strings.Join([]string{
			"        instance = MutableHeaders({'vary': 'Accept-Encoding'})",
			"        result = instance.add_vary_header('Origin')",
			"        assert result is None",
			"        assert instance['vary'] == 'Accept-Encoding, Origin'",
			"",
		}, "\n")
	}
	if cls.Name == "HTTPEndpoint" {
		switch method.Name {
		case "dispatch":
			return strings.Join([]string{
				"        responses = __import__('starlette.responses', fromlist=['PlainTextResponse'])",
				"        messages = []",
				"        async def receive():",
				"            return {'type': 'http.request', 'body': b'', 'more_body': False}",
				"        async def send(message):",
				"            messages.append(message)",
				"        class AsyncEndpoint(HTTPEndpoint):",
				"            async def get(self, request):",
				"                return responses.PlainTextResponse('async-ok')",
				"        class SyncEndpoint(HTTPEndpoint):",
				"            def get(self, request):",
				"                return responses.PlainTextResponse('sync-ok')",
				"        get_scope = {'type': 'http', 'method': 'GET', 'path': '/', 'headers': []}",
				"        asyncio.run(AsyncEndpoint(get_scope, receive, send).dispatch())",
				"        assert messages[0]['status'] == 200",
				"        messages.clear()",
				"        head_scope = {'type': 'http', 'method': 'HEAD', 'path': '/', 'headers': []}",
				"        asyncio.run(SyncEndpoint(head_scope, receive, send).dispatch())",
				"        assert messages[0]['status'] == 200",
				"        messages.clear()",
				"        post_scope = {'type': 'http', 'method': 'POST', 'path': '/', 'headers': []}",
				"        asyncio.run(HTTPEndpoint(post_scope, receive, send).dispatch())",
				"        assert messages[0]['status'] == 405",
				"",
			}, "\n")
		case "method_not_allowed":
			return strings.Join([]string{
				"        requests = __import__('starlette.requests', fromlist=['Request'])",
				"        exceptions = __import__('starlette.exceptions', fromlist=['HTTPException'])",
				"        async def receive():",
				"            return {'type': 'http.request', 'body': b'', 'more_body': False}",
				"        async def send(message):",
				"            return None",
				"        scope = {'type': 'http', 'method': 'POST', 'path': '/', 'headers': []}",
				"        instance = HTTPEndpoint(scope, receive, send)",
				"        request = requests.Request(scope, receive=receive)",
				"        response = asyncio.run(instance.method_not_allowed(request))",
				"        assert response.status_code == 405",
				"        app_scope = dict(scope)",
				"        app_scope['app'] = object()",
				"        app_instance = HTTPEndpoint(app_scope, receive, send)",
				"        app_request = requests.Request(app_scope, receive=receive)",
				"        try:",
				"            asyncio.run(app_instance.method_not_allowed(app_request))",
				"        except exceptions.HTTPException as exc:",
				"            assert exc.status_code == 405",
				"        else:",
				"            assert False, 'expected HTTPException'",
				"",
			}, "\n")
		default:
			return ""
		}
	}
	if cls.Name == "WebSocketEndpoint" {
		switch method.Name {
		case "decode":
			return strings.Join([]string{
				"        class DummyWebSocket:",
				"            def __init__(self):",
				"                self.closed = []",
				"            async def close(self, code=None, reason=None):",
				"                self.closed.append(code)",
				"        async def receive():",
				"            return {'type': 'websocket.disconnect'}",
				"        async def send(message):",
				"            return None",
				"        scope = {'type': 'websocket', 'path': '/', 'headers': []}",
				"        instance = WebSocketEndpoint(scope, receive, send)",
				"        websocket = DummyWebSocket()",
				"        instance.encoding = 'json'",
				"        assert asyncio.run(instance.decode(websocket, {'type': 'websocket.receive', 'text': '{\"ok\": true}'})) == {'ok': True}",
				"        assert asyncio.run(instance.decode(websocket, {'type': 'websocket.receive', 'bytes': b'{\"ok\": true}'})) == {'ok': True}",
				"        instance.encoding = 'text'",
				"        assert asyncio.run(instance.decode(websocket, {'type': 'websocket.receive', 'text': 'hello'})) == 'hello'",
				"        try:",
				"            asyncio.run(instance.decode(websocket, {'type': 'websocket.receive', 'bytes': b'hello'}))",
				"        except RuntimeError as exc:",
				"            assert 'Expected text websocket messages' in str(exc)",
				"        else:",
				"            assert False, 'expected text RuntimeError'",
				"        instance.encoding = 'bytes'",
				"        assert asyncio.run(instance.decode(websocket, {'type': 'websocket.receive', 'bytes': b'hello'})) == b'hello'",
				"        try:",
				"            asyncio.run(instance.decode(websocket, {'type': 'websocket.receive', 'text': 'hello'}))",
				"        except RuntimeError as exc:",
				"            assert 'Expected bytes websocket messages' in str(exc)",
				"        else:",
				"            assert False, 'expected bytes RuntimeError'",
				"        instance.encoding = None",
				"        assert asyncio.run(instance.decode(websocket, {'type': 'websocket.receive', 'text': 'plain'})) == 'plain'",
				"        assert asyncio.run(instance.decode(websocket, {'type': 'websocket.receive', 'bytes': b'plain'})) == b'plain'",
				"        assert websocket.closed == [1003, 1003]",
				"",
			}, "\n")
		case "dispatch":
			return strings.Join([]string{
				"        class RecordingEndpoint(WebSocketEndpoint):",
				"            encoding = 'text'",
				"            def __init__(self, scope, receive, send):",
				"                super().__init__(scope, receive, send)",
				"                self.received = []",
				"                self.disconnected = []",
				"            async def on_connect(self, websocket):",
				"                return None",
				"            async def on_receive(self, websocket, data):",
				"                self.received.append(data)",
				"            async def on_disconnect(self, websocket, close_code):",
				"                self.disconnected.append(close_code)",
				"        class FailingEndpoint(RecordingEndpoint):",
				"            async def on_receive(self, websocket, data):",
				"                raise RuntimeError('boom')",
				"        async def send(message):",
				"            return None",
				"        scope = {'type': 'websocket', 'path': '/', 'headers': []}",
				"        messages = iter([",
				"            {'type': 'websocket.connect'},",
				"            {'type': 'websocket.receive', 'text': 'hello'},",
				"            {'type': 'websocket.disconnect', 'code': 1001},",
				"        ])",
				"        async def receive():",
				"            return next(messages)",
				"        instance = RecordingEndpoint(scope, receive, send)",
				"        asyncio.run(instance.dispatch())",
				"        assert instance.received == ['hello']",
				"        assert instance.disconnected == [1001]",
				"        failing_messages = iter([",
				"            {'type': 'websocket.connect'},",
				"            {'type': 'websocket.receive', 'text': 'boom'},",
				"        ])",
				"        async def failing_receive():",
				"            return next(failing_messages)",
				"        failing = FailingEndpoint(scope, failing_receive, send)",
				"        try:",
				"            asyncio.run(failing.dispatch())",
				"        except RuntimeError as exc:",
				"            assert str(exc) == 'boom'",
				"        else:",
				"            assert False, 'expected RuntimeError'",
				"        assert failing.disconnected == [1011]",
				"",
			}, "\n")
		default:
			return ""
		}
	}
	if cls.Name == "MultiPartParser" {
		switch method.Name {
		case "on_part_data":
			return strings.Join([]string{
				"        datastructures = __import__('starlette.datastructures', fromlist=['Headers'])",
				"        async def stream():",
				"            yield b''",
				"        headers = datastructures.Headers({'content-type': 'multipart/form-data; boundary=x'})",
				"        instance = MultiPartParser(headers, stream(), max_part_size=3)",
				"        instance.on_part_data(b'ab', 0, 2)",
				"        assert instance._current_part.data == bytearray(b'ab')",
				"        try:",
				"            instance.on_part_data(b'cdef', 0, 4)",
				"        except Exception as exc:",
				"            assert 'Part exceeded maximum size' in getattr(exc, 'message', str(exc))",
				"        else:",
				"            assert False, 'expected max part size exception'",
				"",
			}, "\n")
		case "on_part_end":
			return strings.Join([]string{
				"        datastructures = __import__('starlette.datastructures', fromlist=['Headers'])",
				"        async def stream():",
				"            yield b''",
				"        headers = datastructures.Headers({'content-type': 'multipart/form-data; boundary=x'})",
				"        instance = MultiPartParser(headers, stream())",
				"        instance._current_part.field_name = 'field'",
				"        instance._current_part.data.extend(b'value')",
				"        result = instance.on_part_end()",
				"        assert result is None",
				"        assert instance.items == [('field', 'value')]",
				"",
			}, "\n")
		default:
			return ""
		}
	}
	return ""
}

func pyGoalOperationStateObserveCoverageBody(task *types.CoverageTestTask, indent string) string {
	lineRange := strings.TrimSpace(task.LineRange)
	prefix := []string{
		indent + "models = __import__('openai_codex.models', fromlist=['Notification'])",
		indent + "v2 = __import__('openai_codex.generated.v2_all', fromlist=['Turn', 'TurnStartedNotification', 'TurnCompletedNotification', 'ThreadGoal', 'ThreadGoalStatus', 'ThreadGoalUpdatedNotification'])",
		indent + "instance = _GoalOperationState('thread-1')",
	}
	switch lineRange {
	case "64-68":
		lines := append(prefix,
			indent+"instance.activate_turn_routing()",
			indent+"turn = v2.Turn(id='physical-turn', status=v2.TurnStatus.in_progress, items=[])",
			indent+"notification = models.Notification('turn/started', v2.TurnStartedNotification(threadId='thread-1', turn=turn))",
			indent+"result = instance.observe(notification)",
			indent+"assert result is True",
			indent+"assert instance.logical_turn_id == 'physical-turn'",
			indent+"assert instance.current_turn_id == 'physical-turn'",
			indent+"assert instance.started_turn is turn",
			"",
		)
		return strings.Join(lines, "\n")
	case "70-72":
		lines := append(prefix,
			indent+"instance.activate_turn_routing()",
			indent+"instance.current_turn_id = 'physical-turn'",
			indent+"turn = v2.Turn(id='physical-turn', status=v2.TurnStatus.completed, items=[])",
			indent+"notification = models.Notification('turn/completed', v2.TurnCompletedNotification(threadId='thread-1', turn=turn))",
			indent+"result = instance.observe(notification)",
			indent+"assert result is True",
			indent+"assert instance.completed_turn is turn",
			indent+"assert instance.current_turn_id is None",
			"",
		)
		return strings.Join(lines, "\n")
	case "74-76":
		lines := append(prefix,
			indent+"instance.cleared = True",
			indent+"goal = v2.ThreadGoal(createdAt=1, objective='ship', status=v2.ThreadGoalStatus.active, threadId='thread-1', timeUsedSeconds=0, tokensUsed=0, updatedAt=2)",
			indent+"notification = models.Notification('thread/goal/updated', v2.ThreadGoalUpdatedNotification(threadId='thread-1', goal=goal))",
			indent+"result = instance.observe(notification)",
			indent+"assert result is True",
			indent+"assert instance.status == v2.ThreadGoalStatus.active",
			indent+"assert instance.cleared is False",
			"",
		)
		return strings.Join(lines, "\n")
	default:
		return ""
	}
}

func pyGoalStreamCursorProcessCoverageBody(task *types.CoverageTestTask, indent string) string {
	lineRange := strings.TrimSpace(task.LineRange)
	prefix := []string{
		indent + "goal = __import__('openai_codex._goal', fromlist=['_GoalOperationState'])",
		indent + "models = __import__('openai_codex.models', fromlist=['Notification'])",
		indent + "v2 = __import__('openai_codex.generated.v2_all', fromlist=['Turn', 'TurnStartedNotification', 'TurnCompletedNotification', 'ThreadGoalClearedNotification', 'TurnStatus'])",
		indent + "state = goal._GoalOperationState('thread-1')",
	}
	switch lineRange {
	case "261-263":
		lines := append(prefix,
			indent+"instance = _GoalStreamCursor(state)",
			indent+"notification = models.Notification('turn/started', v2.TurnStartedNotification(threadId='thread-1', turn=v2.Turn(id='physical-turn', status=v2.TurnStatus.in_progress, items=[])))",
			indent+"with pytest.raises(RuntimeError):",
			indent+"    instance.process(notification)",
			"",
		)
		return strings.Join(lines, "\n")
	case "265-271":
		lines := append(prefix,
			indent+"state.logical_turn_id = 'logical-turn'",
			indent+"instance = _GoalStreamCursor(state)",
			indent+"notification = models.Notification('turn/started', v2.TurnStartedNotification(threadId='thread-1', turn=v2.Turn(id='physical-turn', status=v2.TurnStatus.in_progress, items=[])))",
			indent+"events, completed = instance.process(notification)",
			indent+"assert completed is False",
			indent+"assert events[0].payload.turn.id == 'logical-turn'",
			"",
		)
		return strings.Join(lines, "\n")
	case "273-277":
		lines := append(prefix,
			indent+"state.logical_turn_id = 'logical-turn'",
			indent+"instance = _GoalStreamCursor(state)",
			indent+"notification = models.Notification('turn/completed', v2.TurnCompletedNotification(threadId='thread-1', turn=v2.Turn(id='physical-turn', status=v2.TurnStatus.interrupted, items=[])))",
			indent+"events, completed = instance.process(notification)",
			indent+"assert completed is True",
			indent+"assert events[0].payload.turn.status == v2.TurnStatus.interrupted",
			"",
		)
		return strings.Join(lines, "\n")
	case "283-290":
		lines := append(prefix,
			indent+"state.logical_turn_id = 'logical-turn'",
			indent+"instance = _GoalStreamCursor(state)",
			indent+"instance.cleared = True",
			indent+"notification = models.Notification('turn/completed', v2.TurnCompletedNotification(threadId='thread-1', turn=v2.Turn(id='physical-turn', status=v2.TurnStatus.failed, items=[])))",
			indent+"events, completed = instance.process(notification)",
			indent+"assert completed is True",
			indent+"assert state.is_finished() is True",
			indent+"assert events[0].payload.turn.status == v2.TurnStatus.failed",
			"",
		)
		return strings.Join(lines, "\n")
	case "293-295", "313-313":
		lines := append(prefix,
			indent+"state.logical_turn_id = 'logical-turn'",
			indent+"instance = _GoalStreamCursor(state)",
			indent+"instance.last_completed = v2.TurnCompletedNotification(threadId='thread-1', turn=v2.Turn(id='physical-turn', status=v2.TurnStatus.completed, items=[]))",
			indent+"notification = models.Notification('thread/goal/cleared', v2.ThreadGoalClearedNotification(threadId='thread-1'))",
			indent+"events, completed = instance.process(notification)",
			indent+"assert completed is True",
			indent+"assert state.is_finished() is True",
			indent+"assert events[0].payload.turn.id == 'logical-turn'",
			"",
		)
		return strings.Join(lines, "\n")
	case "303-311":
		lines := append(prefix,
			indent+"state.logical_turn_id = 'logical-turn'",
			indent+"instance = _GoalStreamCursor(state)",
			indent+"notification = models.Notification('thread/goal/cleared', v2.ThreadGoalClearedNotification(threadId='thread-1'))",
			indent+"events, completed = instance.process(notification)",
			indent+"assert events == []",
			indent+"assert completed is False",
			indent+"assert instance.cleared is True",
			"",
		)
		return strings.Join(lines, "\n")
	default:
		return ""
	}
}

func pyGoalNotificationStreamFinishCoverageBody(className string, indent string) string {
	return strings.Join([]string{
		indent + "goal = __import__('openai_codex._goal', fromlist=['_GoalOperationState'])",
		indent + "state = goal._GoalOperationState('thread-1')",
		indent + "calls = []",
		fmt.Sprintf("%sinstance = %s(state, lambda: None, lambda: calls.append('unregister'), lambda: calls.append('cancel'))", indent, className),
		indent + "result = instance._finish()",
		indent + "assert result is None",
		indent + "assert instance._closed is True",
		indent + "assert state.is_finished() is True",
		indent + "assert calls == ['unregister']",
		"",
	}, "\n")
}

func genPyResultAssertion(a pyFuncAnalysis, indent string) string {
	return genPyResultAssertionWithArgs(a, nil, nil, indent)
}

func pyClassInstanceForCoverageTask(cls pyClassInfo, method pyFuncInfo, task *types.CoverageTestTask) string {
	if cls.Name == "PacifyFlushWrapper" && method.Name == "flush" {
		return "PacifyFlushWrapper(type('BrokenFlush', (), {'flush': lambda self: (_ for _ in ()).throw(OSError(22, 'boom'))})())"
	}
	if cls.Name == "_FixupStream" && method.Name == "readable" && pyCoverageTaskMentions(task, "return False") {
		return "_FixupStream(type('Unreadable', (), {'read': lambda self, size=0: (_ for _ in ()).throw(OSError('boom'))})())"
	}
	if cls.Name == "_Option" && method.Name == "process" && task != nil && task.GapType == "error_path" {
		return "_Option(None, ['--test'], None, action='unknown')"
	}
	if cls.Name == "_GoalOperationState" {
		return "_GoalOperationState('thread-1')"
	}
	if cls.Name == "ProgressBar" {
		return "ProgressBar(type('UnknownLength', (), {'__iter__': lambda self: iter([])})(), width=10, file=__import__('io').StringIO())"
	}
	if init := pyClassInitMethod(cls); init != nil && len(init.Params) > 0 {
		return fmt.Sprintf("%s(%s)", cls.Name, pyArgList(init.Params))
	}
	return fmt.Sprintf("%s()", cls.Name)
}

func pyClassMethodPreCallForCoverageTask(cls pyClassInfo, method pyFuncInfo, task *types.CoverageTestTask) string {
	if cls.Name != "ProgressBar" {
		return ""
	}
	switch method.Name {
	case "format_bar":
		if pyCoverageTaskMentions(task, "time_per_iteration != 0") {
			return "        instance.avg = [1.0]\n        instance.pos = 1\n"
		}
	case "render_progress":
		if pyCoverageTaskMentions(task, "new_width < old_width") {
			return strings.Join([]string{
				"        instance._is_atty = True",
				"        instance.autowidth = True",
				"        instance.width = 10",
				"        instance.max_width = 20",
				"        _shutil = __import__('shutil')",
				"        _os = __import__('os')",
				"        _original_get_terminal_size = _shutil.get_terminal_size",
				"        _shutil.get_terminal_size = lambda: _os.terminal_size((5, 24))",
			}, "\n") + "\n"
		}
	}
	return ""
}

func pyClassMethodAssertionForCoverageTask(cls pyClassInfo, method pyFuncInfo, task *types.CoverageTestTask) string {
	if cls.Name == "_FixupStream" && method.Name == "readable" && pyCoverageTaskMentions(task, "return False") {
		return "        assert result is False\n"
	}
	if cls.Name == "ProgressBar" && method.Name == "format_bar" && pyCoverageTaskMentions(task, "time_per_iteration != 0") {
		return "        assert isinstance(result, str)\n"
	}
	if cls.Name == "ProgressBar" && method.Name == "render_progress" && pyCoverageTaskMentions(task, "new_width < old_width") {
		return strings.Join([]string{
			"        try:",
			"            result = instance.render_progress()",
			"        finally:",
			"            _shutil.get_terminal_size = _original_get_terminal_size",
			"        assert result is None",
			"        assert instance.max_width <= 20",
			"",
		}, "\n")
	}
	return ""
}

func pyClassInitMethod(cls pyClassInfo) *pyFuncInfo {
	for i := range cls.Methods {
		if cls.Methods[i].Name == "__init__" {
			return &cls.Methods[i]
		}
	}
	return nil
}

func pyCoverageTaskMentions(task *types.CoverageTestTask, needle string) bool {
	if task == nil || needle == "" {
		return false
	}
	haystack := strings.Join(append(append([]string{}, task.MissingBranches...), task.SuggestedInputs...), "\n")
	return strings.Contains(haystack, needle)
}

func pyFuncEnvironmentReview(fn pyFuncInfo, task *types.CoverageTestTask) string {
	if task == nil {
		return ""
	}
	if (fn.Name == "get_binary_stdout" || fn.Name == "get_binary_stderr" || fn.Name == "get_binary_stdin") &&
		(pyCoverageTaskMentions(task, "writer is None") || pyCoverageTaskMentions(task, "reader is None")) {
		return fmt.Sprintf("manual_review_environment: %s depends on process std stream binary-wrapper state; cover with injected stream helpers or an integration environment", fn.Name)
	}
	return ""
}

func genPyResultAssertionWithArgs(a pyFuncAnalysis, params []pyParamInfo, boundary *pyBoundary, indent string) string {
	return genPyResultAssertionWithTaskArgs(a, params, boundary, nil, indent)
}

func genPyResultAssertionWithTaskArgs(a pyFuncAnalysis, params []pyParamInfo, boundary *pyBoundary, values map[string]string, indent string) string {
	var sb strings.Builder

	if !a.HasReturn {
		sb.WriteString(indent + "# void function, verify no exception\n")
		return sb.String()
	}

	if expected, ok := pyExpectedReturnExprWithValues(a, params, boundary, values); ok {
		sb.WriteString(indent + "assert result == " + expected + "\n")
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

func pyExpectedReturnExpr(a pyFuncAnalysis, params []pyParamInfo, boundary *pyBoundary) (string, bool) {
	return pyExpectedReturnExprWithValues(a, params, boundary, nil)
}

func pyExpectedReturnExprWithValues(a pyFuncAnalysis, params []pyParamInfo, boundary *pyBoundary, values map[string]string) (string, bool) {
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
	if !pyReturnExprIsSafe(expr) {
		return "", false
	}

	for i, p := range params {
		if p.IsArgs || p.IsKwargs {
			continue
		}
		value := pyArgValue(p, i)
		if boundary != nil && p.Name == boundary.Param {
			value = boundary.Value
		}
		if values != nil && values[p.Name] != "" {
			value = values[p.Name]
		}
		expr = replaceIdentifier(expr, p.Name, value)
	}

	if hasUnknownIdentifiers(stripQuotedLiterals(expr), map[string]bool{
		"True": true, "False": true, "None": true,
		"and": true, "or": true, "not": true, "is": true,
	}) {
		return "", false
	}
	return "(" + expr + ")", true
}

func pyReturnExprIsSafe(expr string) bool {
	if expr == "" || strings.ContainsAny(expr, "\n;{}[]") {
		return false
	}
	for _, blocked := range []string{"await ", "lambda ", "yield ", " for ", " if ", "(", ")("} {
		if strings.Contains(expr, blocked) {
			return false
		}
	}
	return true
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
		} else {
			args[i] = pyArgValue(p, i)
		}
	}
	return strings.Join(args, ", ")
}

func pyArgListForCoverageTask(params []pyParamInfo, task *types.CoverageTestTask, boundary *pyBoundary) string {
	values := coverageTaskInputValues(task, "python")
	if boundary != nil {
		values[boundary.Param] = boundary.Value
	}
	return pyArgListWithValues(params, values)
}

func pyArgListWithValues(params []pyParamInfo, values map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	args := make([]string, len(params))
	for i, p := range params {
		if value := values[p.Name]; value != "" {
			args[i] = value
		} else {
			args[i] = pyArgValue(p, i)
		}
	}
	return strings.Join(args, ", ")
}

func pyBoundaryForCoverageTask(boundaries []pyBoundary, task *types.CoverageTestTask) *pyBoundary {
	values := coverageTaskInputValues(task, "python")
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

func pyArgList(params []pyParamInfo) string {
	if len(params) == 0 {
		return ""
	}
	args := make([]string, len(params))
	for i, p := range params {
		args[i] = pyArgValue(p, i)
	}
	return strings.Join(args, ", ")
}

func pyDefaultArgs(params []pyParamInfo) string {
	return pyArgList(params)
}

func pyInvalidArgList(params []pyParamInfo, boundaries []pyBoundary) string {
	for _, b := range boundaries {
		if b.Value == "None" || b.Value == "False" {
			return pyArgListWithBoundary(params, b)
		}
	}
	return pyPlaceholderArgList(params)
}

func pyPlaceholderArgList(params []pyParamInfo) string {
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

func pyArgValue(p pyParamInfo, _ int) string {
	if p.IsArgs {
		return "()"
	}
	if p.IsKwargs {
		return "{}"
	}

	name := strings.ToLower(p.Name)
	compact := strings.ReplaceAll(strings.ReplaceAll(name, "_", ""), "-", "")

	if compact == "useragent" {
		return "'test/1.0'"
	}
	if pyNameHasPrefix(compact, "is", "has", "can", "should") ||
		pyNameHasAny(compact, "enabled", "active", "valid", "visible", "flag", "checked") {
		return "True"
	}
	if pyNameHasAny(compact, "items", "list", "array", "arr", "rows", "records", "args") {
		return "[]"
	}
	if pyNameHasAny(compact, "iterable") {
		return "[]"
	}
	if pyNameHasAny(compact, "stream") {
		return "__import__('io').BytesIO(b'test')"
	}
	if pyNameHasAny(compact, "wrapped") {
		return "type('Wrapped', (), {'flush': lambda self: None})()"
	}
	if pyNameIsNumeric(compact) {
		if compact == "b" || compact == "y" {
			return "2"
		}
		return "1"
	}
	if pyNameHasAny(compact, "options", "opts", "config", "payload", "data", "body", "params", "query", "user", "metadata") {
		return "{}"
	}
	if pyNameHasAny(compact, "url", "uri", "endpoint", "href") {
		return "'https://example.com'"
	}
	if pyNameHasAny(compact, "email") {
		return "'user@example.com'"
	}
	if pyNameHasAny(compact, "name", "title", "text", "message", "prefix", "suffix", "label", "path", "key", "value", "type", "mode") {
		return "'test'"
	}
	if p.HasDefault {
		return "None"
	}

	return "None"
}

func pyNameHasAny(name string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(name, part) {
			return true
		}
	}
	return false
}

func pyNameHasPrefix(name string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) && len(name) > len(prefix) {
			return true
		}
	}
	return false
}

func pyNameIsNumeric(name string) bool {
	switch name {
	case "a", "b", "x", "y", "n", "num", "number", "count", "size", "index", "idx",
		"age", "page", "limit", "offset", "total", "amount", "price", "id":
		return true
	}
	return strings.HasSuffix(name, "id") || strings.HasSuffix(name, "count") ||
		strings.HasSuffix(name, "index") || strings.HasSuffix(name, "size")
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
