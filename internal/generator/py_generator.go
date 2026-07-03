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
}

type pyParamInfo struct {
	Name       string
	HasDefault bool
	IsArgs     bool // *args
	IsKwargs   bool // **kwargs
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

	// 生成导入语句
	buf.WriteString(fmt.Sprintf("from %s import %s\n\n", moduleName, joinPyExportNames(funcs, classes)))

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
			// currentClass 指向的是 slices 中的副本，需要用索引
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
				// 如果在类内，且是 __init__，不生成方法测试但保留类
				continue
			}

			// 跳过测试辅助
			if isPyTestHelper(name) {
				continue
			}

			// 判断是否是类方法（缩进大于 0 且在类上下文中）
			isMethod := false
			isStatic := false
			className := ""

			if currentClass != nil && lineIndent > currentClassIndent {
				isMethod = true
				className = currentClass.Name
				// 检查是否有 @staticmethod 装饰器
				if i > 0 {
					prevLine := strings.TrimSpace(lines[i-1])
					if strings.HasPrefix(prevLine, "@staticmethod") {
						isStatic = true
					}
				}
			} else {
				// 离开类上下文
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

			if isMethod && currentClass != nil {
				// 找到 classes 中对应的类并添加方法
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

	sb.WriteString(fmt.Sprintf("def test_%s():\n", fn.Name))
	sb.WriteString(fmt.Sprintf("    result = %s(%s)\n", fn.Name, pyArgList(fn.Params)))
	sb.WriteString("    assert result is not None\n\n")

	// 边界用例
	sb.WriteString(fmt.Sprintf("def test_%s_edge_cases():\n", fn.Name))
	if len(fn.Params) == 0 {
		sb.WriteString(fmt.Sprintf("    result = %s()\n", fn.Name))
	} else {
		sb.WriteString(fmt.Sprintf("    result = %s(%s)\n", fn.Name, pyDefaultArgs(fn.Params)))
	}
	sb.WriteString("    assert result is not None\n\n");

	return sb.String()
}

func genPytestClassTest(cls pyClassInfo) string {
	var sb strings.Builder

	// 实例化测试
	sb.WriteString(fmt.Sprintf("class Test%s:\n", cls.Name))
	sb.WriteString("    def test_init(self):\n")
	sb.WriteString(fmt.Sprintf("        instance = %s()\n", cls.Name))
	sb.WriteString(fmt.Sprintf("        assert instance is not None\n\n"))

	// 方法测试
	for _, method := range cls.Methods {
		if method.Name == "__init__" {
			continue
		}

		testName := method.Name
		if method.IsStatic {
			sb.WriteString(fmt.Sprintf("    def test_%s(self):\n", testName))
			sb.WriteString(fmt.Sprintf("        result = %s.%s(%s)\n", cls.Name, method.Name, pyArgList(method.Params)))
		} else {
			sb.WriteString(fmt.Sprintf("    def test_%s(self):\n", testName))
			sb.WriteString(fmt.Sprintf("        instance = %s()\n", cls.Name))
			sb.WriteString(fmt.Sprintf("        result = instance.%s(%s)\n", method.Name, pyArgList(method.Params)))
		}
		sb.WriteString("        assert result is not None\n\n")
	}

	return sb.String()
}

// pyArgList 生成调用参数列表（用 None 占位）
func pyArgList(params []pyParamInfo) string {
	if len(params) == 0 {
		return ""
	}
	args := make([]string, len(params))
	for i, p := range params {
		if p.IsArgs {
			args[i] = "()" // *args 传空元组
		} else if p.IsKwargs {
			args[i] = "{}" // **kwargs 传空字典
		} else if p.HasDefault {
			args[i] = "None" // 有默认值，传 None 让默认值生效
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
