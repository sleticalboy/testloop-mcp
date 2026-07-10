package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
)

type funcInfo struct {
	Name         string
	Params       []paramInfo
	Returns      []paramInfo
	Receiver     string
	ReceiverType string
	IsMethod     bool
	TypeParams   []string // 泛型类型参数名，如 T, K, V
	IsVariadic   bool     // 是否有变参（...T）
	ReturnExpr   string
	FinalReturn  string
	Boundaries   []goBoundary
}

type paramInfo struct {
	Name     string
	Type     string
	Variadic bool // 是否是变参
}

type structInfo struct {
	Name   string
	Fields []fieldInfo
}

type fieldInfo struct {
	Name string
	Type string
}

type interfaceInfo struct {
	Name    string
	Methods []methodSig
}

type methodSig struct {
	Name    string
	Params  []paramInfo
	Returns []paramInfo
}

// GenerateGoTests 读取 Go 源文件，用 AST 分析生成表驱动测试代码
func GenerateGoTests(srcPath string) (string, error) {
	return generateGoTests(srcPath, nil)
}

func GenerateGoTestsForCoverageTask(srcPath string, task *types.CoverageTestTask) (string, error) {
	if task == nil {
		return GenerateGoTests(srcPath)
	}
	return generateGoTests(srcPath, task)
}

func generateGoTests(srcPath string, task *types.CoverageTestTask) (string, error) {
	fs := token.NewFileSet()
	node, err := parser.ParseFile(fs, srcPath, nil, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("解析源文件失败: %w", err)
	}

	pkgName := node.Name.Name

	var funcs []funcInfo
	var structs []structInfo
	var interfaces []interfaceInfo

	// 第一遍：收集结构体和接口定义
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// 泛型结构体：跳过类型参数，只取名字
			structName := typeSpec.Name.Name

			switch t := typeSpec.Type.(type) {
			case *ast.StructType:
				si := structInfo{Name: structName}
				if t.Fields != nil {
					for _, field := range t.Fields.List {
						typ := exprToString(field.Type)
						for _, name := range field.Names {
							si.Fields = append(si.Fields, fieldInfo{Name: name.Name, Type: typ})
						}
					}
				}
				structs = append(structs, si)

			case *ast.InterfaceType:
				ii := interfaceInfo{Name: structName}
				if t.Methods != nil {
					for _, method := range t.Methods.List {
						if ft, ok := method.Type.(*ast.FuncType); ok {
							sig := methodSig{Name: method.Names[0].Name}
							if ft.Params != nil {
								for _, p := range ft.Params.List {
									typ := exprToString(p.Type)
									for _, name := range p.Names {
										sig.Params = append(sig.Params, paramInfo{Name: name.Name, Type: typ})
									}
								}
							}
							if ft.Results != nil {
								for i, r := range ft.Results.List {
									typ := exprToString(r.Type)
									sig.Returns = append(sig.Returns, paramInfo{
										Name: fmt.Sprintf("ret%d", i),
										Type: typ,
									})
								}
							}
							ii.Methods = append(ii.Methods, sig)
						}
					}
				}
				interfaces = append(interfaces, ii)
			}
		}
	}

	// 第二遍：收集函数和方法
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}

		info := funcInfo{Name: fn.Name.Name}

		// 解析泛型类型参数
		if fn.Type.TypeParams != nil {
			for _, tp := range fn.Type.TypeParams.List {
				for _, name := range tp.Names {
					info.TypeParams = append(info.TypeParams, name.Name)
				}
			}
		}

		// 检查方法接收者
		if fn.Recv != nil {
			info.IsMethod = true
			for _, field := range fn.Recv.List {
				recvType := exprToString(field.Type)
				info.ReceiverType = recvType
				for _, name := range field.Names {
					info.Receiver = name.Name
				}
			}
		}

		// 解析参数
		if fn.Type.Params != nil {
			for _, p := range fn.Type.Params.List {
				// 检查是否是变参 ...
				if ell, ok := p.Type.(*ast.Ellipsis); ok {
					typ := "[]" + exprToString(ell.Elt)
					for _, name := range p.Names {
						info.Params = append(info.Params, paramInfo{Name: name.Name, Type: typ, Variadic: true})
					}
					info.IsVariadic = true
					continue
				}
				typ := exprToString(p.Type)
				for _, name := range p.Names {
					info.Params = append(info.Params, paramInfo{Name: name.Name, Type: typ})
				}
			}
		}

		// 解析返回值
		if fn.Type.Results != nil {
			for i, r := range fn.Type.Results.List {
				typ := exprToString(r.Type)
				name := fmt.Sprintf("ret%d", i)
				if len(r.Names) > 0 {
					name = r.Names[0].Name
				}
				info.Returns = append(info.Returns, paramInfo{Name: name, Type: typ})
			}
		}

		// 泛型函数：把类型参数替换为具体类型（默认 int）
		if len(info.TypeParams) > 0 {
			substMap := make(map[string]string)
			for _, tp := range info.TypeParams {
				substMap[tp] = "int"
			}
			for i := range info.Params {
				info.Params[i].Type = substituteType(info.Params[i].Type, substMap)
			}
			for i := range info.Returns {
				info.Returns[i].Type = substituteType(info.Returns[i].Type, substMap)
			}
		}

		info.ReturnExpr = singleReturnExpr(fn.Body)
		info.FinalReturn = finalReturnExpr(fn.Body)
		info.Boundaries = extractGoBoundaries(fn.Body)

		funcs = append(funcs, info)
	}

	if task != nil {
		funcs = filterGoFuncsForCoverageTask(funcs, task)
	}

	if len(funcs) == 0 && len(interfaces) == 0 {
		return "// 未发现需要生成测试的 exported 函数或接口", nil
	}

	// 检测生成测试需要的额外依赖
	needReflect := false
	needTime := false
	needErrors := false
	for _, fn := range funcs {
		seedCase, hasSeedCase := goSeedTestCase(fn, task)
		if hasSeedCase && seedCase.Assert == goAssertTimeFormat {
			needTime = true
		}
		if hasSeedCase && goSeedCaseUsesPackage(seedCase, "errors") {
			needErrors = true
		}
		if hasSeedCase && seedCase.Assert != goAssertExact {
			continue
		}
		if !hasSeedCase && goCoverageSmokeCallable(fn, task) {
			continue
		}
		for _, r := range fn.Returns {
			if r.Type != "error" && needsDeepEqual(r.Type) {
				needReflect = true
				break
			}
		}
		if needReflect {
			break
		}
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("package %s\n\n", pkgName))
	buf.WriteString("import (\n")
	buf.WriteString("\t\"testing\"\n")
	if needErrors {
		buf.WriteString("\t\"errors\"\n")
	}
	if needTime {
		buf.WriteString("\t\"time\"\n")
	}
	if needReflect {
		buf.WriteString("\t\"reflect\"\n")
	}
	buf.WriteString(")\n\n")

	// 为结构体生成测试辅助函数
	for _, s := range structs {
		buf.WriteString(fmt.Sprintf("// makeTest%s 创建测试用的 %s 实例\n", s.Name, s.Name))
		buf.WriteString(fmt.Sprintf("func makeTest%s() %s {\n", s.Name, s.Name))
		buf.WriteString(fmt.Sprintf("\treturn %s{\n", s.Name))
		for _, f := range s.Fields {
			buf.WriteString(fmt.Sprintf("\t\t%s: %s,\n", f.Name, zeroValue(f.Type)))
		}
		buf.WriteString("\t}\n")
		buf.WriteString("}\n\n")
	}

	// 为接口生成 mock
	for _, iface := range interfaces {
		buf.WriteString(genMock(iface))
	}

	// 生成函数/方法测试
	for _, fn := range funcs {
		buf.WriteString(genTableDrivenTestForTask(fn, task))
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.String(), fmt.Errorf("格式化失败: %w", err)
	}
	return string(formatted), nil
}

func filterGoFuncsForCoverageTask(funcs []funcInfo, task *types.CoverageTestTask) []funcInfo {
	target := strings.TrimSpace(task.Target)
	if target == "" {
		return funcs
	}
	filtered := make([]funcInfo, 0, len(funcs))
	for _, fn := range funcs {
		if goFuncMatchesTarget(fn, target) {
			filtered = append(filtered, fn)
		}
	}
	if len(filtered) == 0 {
		return funcs
	}
	return filtered
}

func goFuncMatchesTarget(fn funcInfo, target string) bool {
	if fn.Name == target {
		return true
	}
	if fn.IsMethod {
		recv := strings.TrimPrefix(fn.ReceiverType, "*")
		return recv+"."+fn.Name == target || recv+"_"+fn.Name == target
	}
	return false
}

func genMock(iface interfaceInfo) string {
	var sb strings.Builder
	mockName := iface.Name + "Mock"

	sb.WriteString(fmt.Sprintf("// %s 是 %s 的简单 mock 实现\n", mockName, iface.Name))
	sb.WriteString(fmt.Sprintf("type %s struct {\n", mockName))

	// 为每个方法存储一个函数字段
	for _, m := range iface.Methods {
		sb.WriteString(fmt.Sprintf("\t%sFn func(", m.Name))
		paramTypes := make([]string, len(m.Params))
		for i, p := range m.Params {
			paramTypes[i] = p.Type
		}
		sb.WriteString(strings.Join(paramTypes, ", "))
		sb.WriteString(")")

		returnTypes := make([]string, len(m.Returns))
		for i, r := range m.Returns {
			returnTypes[i] = r.Type
		}
		if len(returnTypes) > 0 {
			sb.WriteString(" (")
			sb.WriteString(strings.Join(returnTypes, ", "))
			sb.WriteString(")")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("}\n\n")

	// 为每个方法生成实现
	for _, m := range iface.Methods {
		sb.WriteString(fmt.Sprintf("func (m *%s) %s(", mockName, m.Name))
		params := make([]string, len(m.Params))
		for i, p := range m.Params {
			params[i] = fmt.Sprintf("%s %s", p.Name, p.Type)
		}
		sb.WriteString(strings.Join(params, ", "))
		sb.WriteString(")")

		returns := make([]string, len(m.Returns))
		for i, r := range m.Returns {
			returns[i] = r.Type
		}
		if len(returns) > 0 {
			sb.WriteString(" (")
			sb.WriteString(strings.Join(returns, ", "))
			sb.WriteString(")")
		}
		sb.WriteString(" {\n")

		// 检查是否有对应的 Fn 字段
		sb.WriteString(fmt.Sprintf("\tif m.%sFn != nil {\n", m.Name))
		args := make([]string, len(m.Params))
		for i, p := range m.Params {
			args[i] = p.Name
		}
		callExpr := fmt.Sprintf("m.%sFn(%s)", m.Name, strings.Join(args, ", "))

		if len(m.Returns) > 0 {
			returnVars := make([]string, len(m.Returns))
			for i := range m.Returns {
				returnVars[i] = fmt.Sprintf("ret%d", i)
			}
			sb.WriteString(fmt.Sprintf("\t\t%s := %s\n", strings.Join(returnVars, ", "), callExpr))
			sb.WriteString(fmt.Sprintf("\t\treturn %s\n", strings.Join(returnVars, ", ")))
		} else {
			sb.WriteString(fmt.Sprintf("\t\t%s\n", callExpr))
		}
		sb.WriteString("\t}\n")

		// 默认返回零值
		if len(m.Returns) > 0 {
			zeros := make([]string, len(m.Returns))
			for i, r := range m.Returns {
				zeros[i] = zeroValue(r.Type)
			}
			sb.WriteString(fmt.Sprintf("\treturn %s\n", strings.Join(zeros, ", ")))
		}
		sb.WriteString("}\n\n")
	}

	return sb.String()
}

func genTableDrivenTest(fn funcInfo) string {
	return genTableDrivenTestForTask(fn, nil)
}

func goTestCaseFieldName(name string) string {
	switch name {
	case "name", "skip":
		return name + "Value"
	default:
		return name
	}
}

func genTableDrivenTestForTask(fn funcInfo, task *types.CoverageTestTask) string {
	var sb strings.Builder

	// 泛型类型参数实例化（测试中用具体类型）
	typeArgs := ""
	if len(fn.TypeParams) > 0 {
		// 默认用 int 实例化
		concreteTypes := make([]string, len(fn.TypeParams))
		for i := range fn.TypeParams {
			concreteTypes[i] = "int"
		}
		typeArgs = "[" + strings.Join(concreteTypes, ", ") + "]"
	}

	testName := fn.Name
	if fn.IsMethod {
		testName = fn.ReceiverType + "_" + fn.Name
		// 去掉指针前缀
		testName = strings.TrimPrefix(testName, "*")
	}
	if task != nil && strings.TrimSpace(task.TestName) != "" {
		testName = strings.TrimPrefix(strings.TrimSpace(task.TestName), "Test")
	}

	sb.WriteString(fmt.Sprintf("// Test%s 测试 %s\n", testName, fn.Name))
	if task != nil {
		sb.WriteString(fmt.Sprintf("// coverage task: %s\n", goCoverageTaskComment(task)))
	}
	sb.WriteString(fmt.Sprintf("func Test%s(t *testing.T) {\n", testName))
	seedCase, hasSeedCase := goSeedTestCase(fn, task)
	smokeCase := !hasSeedCase && goCoverageSmokeCallable(fn, task)
	exactCase := hasSeedCase && seedCase.Assert == goAssertExact
	expectedReturnFields := !smokeCase && (!hasSeedCase || exactCase)

	// 定义测试用例结构体
	sb.WriteString("\ttype testCase struct {\n")
	sb.WriteString("\t\tname string\n")
	sb.WriteString("\t\tskip bool\n")
	for _, p := range fn.Params {
		sb.WriteString(fmt.Sprintf("\t\t%s %s\n", goTestCaseFieldName(p.Name), p.Type))
	}
	if !smokeCase {
		if expectedReturnFields {
			for _, r := range fn.Returns {
				sb.WriteString(fmt.Sprintf("\t\t%s %s\n", goTestCaseFieldName(r.Name), r.Type))
			}
		}
		if hasSeedCase && seedCase.Assert == goAssertTimeFormat {
			sb.WriteString("\t\tlayout string\n")
		}
	}
	sb.WriteString("\t}\n\n")

	// 测试用例列表
	sb.WriteString("\ttests := []testCase{\n")
	sb.WriteString("\t\t{\n")
	if hasSeedCase {
		sb.WriteString(fmt.Sprintf("\t\t\tname: %q,\n", goCoverageTaskCaseName(task, "simple")))
		sb.WriteString("\t\t\tskip: false,\n")
		for _, p := range fn.Params {
			sb.WriteString(fmt.Sprintf("\t\t\t%s: %s,\n", goTestCaseFieldName(p.Name), seedCase.Inputs[p.Name]))
		}
		if exactCase {
			for _, r := range fn.Returns {
				sb.WriteString(fmt.Sprintf("\t\t\t%s: %s,\n", goTestCaseFieldName(r.Name), seedCase.Outputs[r.Name]))
			}
		}
		if seedCase.Assert == goAssertTimeFormat {
			sb.WriteString(fmt.Sprintf("\t\t\tlayout: %q,\n", seedCase.TimeLayout))
		}
	} else if smokeCase {
		sb.WriteString(fmt.Sprintf("\t\t\tname: %q,\n", goCoverageTaskCaseName(task, "smoke")))
		sb.WriteString("\t\t\tskip: false,\n")
	} else {
		sb.WriteString(fmt.Sprintf("\t\t\tname: %q,\n", goCoverageTaskCaseName(task, "todo")))
		sb.WriteString("\t\t\tskip: true, // TODO: 填写有意义的输入和期望值后改为 false\n")
		for _, p := range fn.Params {
			sb.WriteString(fmt.Sprintf("\t\t\t%s: %s,\n", goTestCaseFieldName(p.Name), zeroValue(p.Type)))
		}
		for _, r := range fn.Returns {
			sb.WriteString(fmt.Sprintf("\t\t\t%s: %s,\n", goTestCaseFieldName(r.Name), zeroValue(r.Type)))
		}
	}
	sb.WriteString("\t\t},\n")
	sb.WriteString("\t}\n\n")

	// 执行测试
	sb.WriteString("\tfor _, tt := range tests {\n")
	sb.WriteString(fmt.Sprintf("\t\tt.Run(tt.name, func(t *testing.T) {\n"))
	sb.WriteString("\t\t\tif tt.skip {\n")
	sb.WriteString("\t\t\t\tt.Skip(\"TODO: fill in meaningful test inputs and expected values\")\n")
	sb.WriteString("\t\t\t}\n")

	// 为通道类型参数添加 nil 检查，避免阻塞
	for _, p := range fn.Params {
		if strings.Contains(p.Type, "chan") {
			sb.WriteString(fmt.Sprintf("\t\t\tif tt.%s == nil {\n\t\t\t\tt.Skip(\"%s is nil, fill in test data\")\n\t\t\t}\n", goTestCaseFieldName(p.Name), p.Name))
		}
	}

	// 创建接收者实例（如果是方法）
	if fn.IsMethod {
		recvType := fn.ReceiverType
		if strings.HasPrefix(recvType, "*") {
			sb.WriteString(fmt.Sprintf("\t\t\t%s := &%s{}\n", fn.Receiver, recvType[1:]))
		} else {
			sb.WriteString(fmt.Sprintf("\t\t\t%s := %s{}\n", fn.Receiver, recvType))
		}
	}

	// 构建调用参数
	args := make([]string, len(fn.Params))
	for i, p := range fn.Params {
		if p.Variadic {
			args[i] = fmt.Sprintf("tt.%s...", goTestCaseFieldName(p.Name))
		} else {
			args[i] = fmt.Sprintf("tt.%s", goTestCaseFieldName(p.Name))
		}
	}

	// 构建调用表达式
	callExpr := ""
	if fn.IsMethod {
		callExpr = fmt.Sprintf("%s.%s%s(%s)", fn.Receiver, fn.Name, typeArgs, strings.Join(args, ", "))
	} else {
		callExpr = fmt.Sprintf("%s%s(%s)", fn.Name, typeArgs, strings.Join(args, ", "))
	}

	// 处理返回值
	if smokeCase {
		writeGoSmokeCall(&sb, fn, callExpr)
	} else if hasSeedCase && seedCase.Assert == goAssertTimeFormat {
		sb.WriteString(fmt.Sprintf("\t\t\tgot := %s\n", callExpr))
		sb.WriteString("\t\t\tif _, err := time.Parse(tt.layout, got); err != nil {\n")
		sb.WriteString("\t\t\t\tt.Errorf(\"got %q, want layout %q: %v\", got, tt.layout, err)\n")
		sb.WriteString("\t\t\t}\n")
	} else if hasSeedCase && seedCase.Assert == goAssertTimeDateZero {
		sb.WriteString(fmt.Sprintf("\t\t\tgot := %s\n", callExpr))
		sb.WriteString("\t\t\tif got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Nanosecond() != 0 {\n")
		sb.WriteString("\t\t\t\tt.Errorf(\"got time component %02d:%02d:%02d.%09d, want date boundary\", got.Hour(), got.Minute(), got.Second(), got.Nanosecond())\n")
		sb.WriteString("\t\t\t}\n")
	} else if len(fn.Returns) == 0 {
		sb.WriteString(fmt.Sprintf("\t\t\t%s\n", callExpr))
	} else if len(fn.Returns) == 1 {
		if fn.Returns[0].Type == "error" {
			sb.WriteString(fmt.Sprintf("\t\t\terr := %s\n", callExpr))
			sb.WriteString("\t\t\tif err != nil {\n")
			sb.WriteString("\t\t\t\tt.Errorf(\"unexpected error: %v\", err)\n")
			sb.WriteString("\t\t\t}\n")
		} else {
			sb.WriteString(fmt.Sprintf("\t\t\tgot := %s\n", callExpr))
			if needsDeepEqual(fn.Returns[0].Type) {
				sb.WriteString(fmt.Sprintf("\t\t\tif !reflect.DeepEqual(got, tt.%s) {\n", goTestCaseFieldName(fn.Returns[0].Name)))
			} else {
				sb.WriteString(fmt.Sprintf("\t\t\tif got != tt.%s {\n", goTestCaseFieldName(fn.Returns[0].Name)))
			}
			sb.WriteString(fmt.Sprintf("\t\t\t\tt.Errorf(\"got %%v, want %%v\", got, tt.%s)\n", goTestCaseFieldName(fn.Returns[0].Name)))
			sb.WriteString("\t\t\t}\n")
		}
	} else {
		returnVars := make([]string, len(fn.Returns))
		for i := range fn.Returns {
			returnVars[i] = fmt.Sprintf("got%d", i)
		}
		sb.WriteString(fmt.Sprintf("\t\t\t%s := %s\n", strings.Join(returnVars, ", "), callExpr))

		for i, r := range fn.Returns {
			if r.Type == "error" {
				sb.WriteString(fmt.Sprintf("\t\t\tif %s != nil {\n", returnVars[i]))
				sb.WriteString(fmt.Sprintf("\t\t\t\tt.Errorf(\"unexpected error: %%v\", %s)\n", returnVars[i]))
				sb.WriteString("\t\t\t}\n")
			} else if needsDeepEqual(r.Type) {
				sb.WriteString(fmt.Sprintf("\t\t\tif !reflect.DeepEqual(%s, tt.%s) {\n", returnVars[i], goTestCaseFieldName(r.Name)))
				sb.WriteString(fmt.Sprintf("\t\t\t\tt.Errorf(\"%s: got %%v, want %%v\", %s, tt.%s)\n", r.Name, returnVars[i], goTestCaseFieldName(r.Name)))
				sb.WriteString("\t\t\t}\n")
			} else {
				sb.WriteString(fmt.Sprintf("\t\t\tif %s != tt.%s {\n", returnVars[i], goTestCaseFieldName(r.Name)))
				sb.WriteString(fmt.Sprintf("\t\t\t\tt.Errorf(\"%s: got %%v, want %%v\", %s, tt.%s)\n", r.Name, returnVars[i], goTestCaseFieldName(r.Name)))
				sb.WriteString("\t\t\t}\n")
			}
		}
	}

	sb.WriteString("\t\t})\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	return sb.String()
}

func goCoverageSmokeCallable(fn funcInfo, task *types.CoverageTestTask) bool {
	if task == nil || fn.IsMethod || fn.IsVariadic || len(fn.Params) > 0 {
		return false
	}
	for _, r := range fn.Returns {
		if r.Type == "error" ||
			strings.HasPrefix(r.Type, "chan ") ||
			strings.Contains(r.Type, "<-chan") ||
			strings.HasPrefix(r.Type, "func") {
			return false
		}
	}
	return true
}

func writeGoSmokeCall(sb *strings.Builder, fn funcInfo, callExpr string) {
	switch len(fn.Returns) {
	case 0:
		sb.WriteString(fmt.Sprintf("\t\t\t%s\n", callExpr))
	case 1:
		sb.WriteString(fmt.Sprintf("\t\t\tgot := %s\n", callExpr))
		sb.WriteString("\t\t\t_ = got\n")
	default:
		returnVars := make([]string, len(fn.Returns))
		for i := range fn.Returns {
			returnVars[i] = fmt.Sprintf("got%d", i)
		}
		sb.WriteString(fmt.Sprintf("\t\t\t%s := %s\n", strings.Join(returnVars, ", "), callExpr))
		for _, name := range returnVars {
			sb.WriteString(fmt.Sprintf("\t\t\t_ = %s\n", name))
		}
	}
}

func goCoverageTaskCaseName(task *types.CoverageTestTask, fallback string) string {
	if task == nil {
		return fallback
	}
	switch task.GapType {
	case "branch":
		return "coverage branch gap"
	case "error_path":
		return "coverage error path"
	case "return_path":
		return "coverage return path"
	case "statement":
		return "coverage statement gap"
	default:
		return "coverage gap"
	}
}

func goCoverageTaskComment(task *types.CoverageTestTask) string {
	parts := []string{}
	if task.ID != "" {
		parts = append(parts, task.ID)
	}
	if task.LineRange != "" {
		parts = append(parts, "lines "+task.LineRange)
	}
	if len(task.AssertionFocus) > 0 {
		parts = append(parts, strings.Join(task.AssertionFocus, "; "))
	}
	if len(parts) == 0 {
		return "coverage gap"
	}
	return strings.Join(parts, " | ")
}

// needsDeepEqual 判断类型是否需要用 reflect.DeepEqual 比较
func needsDeepEqual(typ string) bool {
	return strings.HasPrefix(typ, "[]") ||
		strings.HasPrefix(typ, "map[") ||
		(!strings.HasPrefix(typ, "int") &&
			!strings.HasPrefix(typ, "uint") &&
			!strings.HasPrefix(typ, "float") &&
			typ != "string" &&
			typ != "bool" &&
			typ != "error" &&
			typ != "any" &&
			typ != "interface{}" &&
			!strings.HasPrefix(typ, "*") &&
			!strings.HasPrefix(typ, "chan ") &&
			!strings.HasPrefix(typ, "func"))
}

type goAssertionKind string

const (
	goAssertExact        goAssertionKind = "exact"
	goAssertTimeFormat   goAssertionKind = "time_format"
	goAssertTimeDateZero goAssertionKind = "time_date_zero"
)

type goSeedCase struct {
	Assert     goAssertionKind
	Inputs     map[string]string
	Outputs    map[string]string
	TimeLayout string
}

type goBoundary struct {
	Param      string
	Op         string
	Value      string
	Condition  string
	ReturnExpr string
}

func goSeedTestCase(fn funcInfo, task *types.CoverageTestTask) (goSeedCase, bool) {
	if fn.IsMethod || fn.IsVariadic || len(fn.Returns) != 1 || fn.Returns[0].Type == "error" {
		return goSeedCase{}, false
	}
	if seed, ok := goBranchSeedTestCase(fn, task); ok {
		return seed, true
	}
	if fn.Returns[0].Type == "string" && len(fn.Params) == 0 {
		if layout, ok := goTimeFormatLayout(fn.ReturnExpr); ok {
			return goSeedCase{
				Assert:     goAssertTimeFormat,
				Inputs:     map[string]string{},
				Outputs:    map[string]string{},
				TimeLayout: layout,
			}, true
		}
	}
	if fn.Returns[0].Type == "time.Time" && len(fn.Params) == 0 && goTimeDateZeroExpr(fn.FinalReturn) {
		return goSeedCase{
			Assert:  goAssertTimeDateZero,
			Inputs:  map[string]string{},
			Outputs: map[string]string{},
		}, true
	}
	if !goTypeSupportsExactSeed(fn.Returns[0].Type) || fn.ReturnExpr == "" || !goReturnExprIsSafe(fn.ReturnExpr) {
		return goSeedCase{}, false
	}

	seed := goSeedCase{Assert: goAssertExact, Inputs: map[string]string{}, Outputs: map[string]string{}}
	expr := fn.ReturnExpr
	for i, p := range fn.Params {
		if !goTypeSupportsExactSeed(p.Type) {
			return goSeedCase{}, false
		}
		value := goArgValue(p, i)
		seed.Inputs[p.Name] = value
		expr = replaceIdentifier(expr, p.Name, value)
	}

	if hasUnknownIdentifiers(stripQuotedLiterals(expr), map[string]bool{
		"true": true, "false": true, "nil": true,
	}) {
		return goSeedCase{}, false
	}
	seed.Outputs[fn.Returns[0].Name] = expr
	return seed, true
}

func goBranchSeedTestCase(fn funcInfo, task *types.CoverageTestTask) (goSeedCase, bool) {
	if task == nil || task.GapType != "branch" || len(fn.Boundaries) == 0 || !goTypeSupportsExactSeed(fn.Returns[0].Type) {
		return goSeedCase{}, false
	}
	boundary := goBoundaryForCoverageTask(fn.Boundaries, task)
	if boundary == nil || boundary.ReturnExpr == "" || !goReturnExprIsSafe(boundary.ReturnExpr) {
		return goSeedCase{}, false
	}

	inputs := map[string]string{}
	for i, p := range fn.Params {
		value := goArgValue(p, i)
		if p.Name == boundary.Param {
			boundaryValue, ok := goBoundaryInputValue(*boundary, p.Type)
			if !ok {
				return goSeedCase{}, false
			}
			value = boundaryValue
		}
		inputs[p.Name] = value
	}

	expr := boundary.ReturnExpr
	for _, p := range fn.Params {
		expr = replaceIdentifier(expr, p.Name, inputs[p.Name])
	}
	if hasUnknownIdentifiers(stripQuotedLiterals(expr), map[string]bool{
		"true": true, "false": true, "nil": true,
	}) {
		return goSeedCase{}, false
	}
	return goSeedCase{
		Assert:  goAssertExact,
		Inputs:  inputs,
		Outputs: map[string]string{fn.Returns[0].Name: expr},
	}, true
}

func goBoundaryForCoverageTask(boundaries []goBoundary, task *types.CoverageTestTask) *goBoundary {
	hints := goCoverageTaskConditionHints(task)
	for i := range boundaries {
		for _, hint := range hints {
			if goBoundaryMatchesHint(boundaries[i], hint) {
				return &boundaries[i]
			}
		}
	}
	if task != nil && task.GapType == "branch" && len(boundaries) == 1 {
		return &boundaries[0]
	}
	return nil
}

func goCoverageTaskConditionHints(task *types.CoverageTestTask) []string {
	if task == nil {
		return nil
	}
	var hints []string
	for _, values := range [][]string{task.SuggestedInputs, task.MissingBranches, task.AssertionFocus} {
		for _, value := range values {
			if expr := firstBacktickExpr(value); expr != "" {
				hints = append(hints, expr)
				continue
			}
			if idx := strings.Index(value, ":"); idx >= 0 {
				trimmed := strings.TrimSpace(value[idx+1:])
				if trimmed != "" {
					hints = append(hints, trimmed)
				}
			}
		}
	}
	return hints
}

func firstBacktickExpr(s string) string {
	start := strings.Index(s, "`")
	if start < 0 {
		return ""
	}
	end := strings.Index(s[start+1:], "`")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(s[start+1 : start+1+end])
}

func goBoundaryMatchesHint(boundary goBoundary, hint string) bool {
	hint = strings.TrimSpace(hint)
	if hint == "" {
		return false
	}
	if hint == boundary.Condition {
		return true
	}
	compactHint := strings.Join(strings.Fields(hint), " ")
	return compactHint == boundary.Condition ||
		compactHint == strings.Join([]string{boundary.Param, boundary.Op, boundary.Value}, " ")
}

func goBoundaryInputValue(boundary goBoundary, typ string) (string, bool) {
	value := strings.TrimSpace(boundary.Value)
	switch boundary.Op {
	case "==", ">=", "<=":
		return goLiteralForType(value, typ)
	case "!=":
		return goAlternateLiteralForType(value, typ)
	case ">":
		return goNumericOffsetLiteral(value, typ, 1)
	case "<":
		return goNumericOffsetLiteral(value, typ, -1)
	default:
		return "", false
	}
}

func goLiteralForType(value string, typ string) (string, bool) {
	if value == "nil" {
		return goNilLiteralForType(typ)
	}
	switch {
	case typ == "string":
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			return value, true
		}
		return strconv.Quote(value), true
	case typ == "bool":
		if value == "true" || value == "false" {
			return value, true
		}
		return "", false
	case strings.HasPrefix(typ, "int"), strings.HasPrefix(typ, "uint"):
		if _, err := strconv.Atoi(value); err == nil {
			return value, true
		}
		return "", false
	case strings.HasPrefix(typ, "float"):
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return value, true
		}
		return "", false
	default:
		return "", false
	}
}

func goAlternateLiteralForType(value string, typ string) (string, bool) {
	if value == "nil" {
		return goNonNilLiteralForType(typ)
	}
	switch {
	case typ == "string":
		lit, ok := goLiteralForType(value, typ)
		if !ok {
			return "", false
		}
		if lit == `""` {
			return `"test"`, true
		}
		return `""`, true
	case typ == "bool":
		if value == "true" {
			return "false", true
		}
		if value == "false" {
			return "true", true
		}
		return "", false
	case strings.HasPrefix(typ, "int"), strings.HasPrefix(typ, "uint"), strings.HasPrefix(typ, "float"):
		return goNumericOffsetLiteral(value, typ, 1)
	default:
		return "", false
	}
}

func goNilLiteralForType(typ string) (string, bool) {
	switch {
	case typ == "any", typ == "interface{}", typ == "error",
		strings.HasPrefix(typ, "*"),
		strings.HasPrefix(typ, "[]"),
		strings.HasPrefix(typ, "map["),
		strings.HasPrefix(typ, "chan "),
		strings.Contains(typ, "<-chan"),
		strings.HasPrefix(typ, "func"):
		return "nil", true
	default:
		return "", false
	}
}

func goNonNilLiteralForType(typ string) (string, bool) {
	switch {
	case strings.HasPrefix(typ, "*"):
		return "&" + strings.TrimPrefix(typ, "*") + "{}", true
	case strings.HasPrefix(typ, "[]"), strings.HasPrefix(typ, "map["):
		return typ + "{}", true
	case typ == "error":
		return `errors.New("test")`, true
	case typ == "any", typ == "interface{}":
		return "struct{}{}", true
	default:
		return "", false
	}
}

func goNumericOffsetLiteral(value string, typ string, delta int) (string, bool) {
	if strings.HasPrefix(typ, "float") {
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return "", false
		}
		return strconv.FormatFloat(n+float64(delta), 'f', 1, 64), true
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return "", false
	}
	next := n + delta
	if strings.HasPrefix(typ, "uint") && next < 0 {
		return "", false
	}
	return strconv.Itoa(next), true
}

func goSeedCaseUsesPackage(seed goSeedCase, pkg string) bool {
	prefix := pkg + "."
	for _, value := range seed.Inputs {
		if strings.Contains(value, prefix) {
			return true
		}
	}
	for _, value := range seed.Outputs {
		if strings.Contains(value, prefix) {
			return true
		}
	}
	return false
}

func goTimeFormatLayout(expr string) (string, bool) {
	expr = strings.TrimSpace(expr)
	const prefix = "time.Now().Format("
	if !strings.HasPrefix(expr, prefix) || !strings.HasSuffix(expr, ")") {
		return "", false
	}
	quoted := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(expr, prefix), ")"))
	layout, err := strconv.Unquote(quoted)
	if err != nil || layout == "" {
		return "", false
	}
	return layout, true
}

func goTimeDateZeroExpr(expr string) bool {
	expr = strings.TrimSpace(expr)
	const prefix = "time.Date("
	if !strings.HasPrefix(expr, prefix) || !strings.HasSuffix(expr, ")") {
		return false
	}
	args := splitGoCallArgs(strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(expr, prefix), ")")))
	if len(args) != 8 {
		return false
	}
	for _, idx := range []int{3, 4, 5, 6} {
		if strings.TrimSpace(args[idx]) != "0" {
			return false
		}
	}
	return true
}

func splitGoCallArgs(args string) []string {
	if strings.TrimSpace(args) == "" {
		return nil
	}
	var parts []string
	var current strings.Builder
	depth := 0
	inString := false
	escaped := false
	for _, r := range args {
		if inString {
			current.WriteRune(r)
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '"' {
				inString = false
			}
			continue
		}
		switch r {
		case '"':
			inString = true
			current.WriteRune(r)
		case '(', '[', '{':
			depth++
			current.WriteRune(r)
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
			current.WriteRune(r)
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(current.String()))
				current.Reset()
				continue
			}
			current.WriteRune(r)
		default:
			current.WriteRune(r)
		}
	}
	parts = append(parts, strings.TrimSpace(current.String()))
	return parts
}

func singleReturnExpr(body *ast.BlockStmt) string {
	if body == nil || len(body.List) != 1 {
		return ""
	}
	ret, ok := body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return ""
	}
	return exprToString(ret.Results[0])
}

func finalReturnExpr(body *ast.BlockStmt) string {
	if body == nil || len(body.List) == 0 {
		return ""
	}
	ret, ok := body.List[len(body.List)-1].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return ""
	}
	return exprToString(ret.Results[0])
}

func extractGoBoundaries(body *ast.BlockStmt) []goBoundary {
	if body == nil {
		return nil
	}
	var boundaries []goBoundary
	for _, stmt := range body.List {
		ifStmt, ok := stmt.(*ast.IfStmt)
		if !ok || ifStmt.Body == nil {
			continue
		}
		boundary, ok := goBoundaryFromIf(ifStmt)
		if ok {
			boundaries = append(boundaries, boundary)
		}
	}
	return boundaries
}

func goBoundaryFromIf(ifStmt *ast.IfStmt) (goBoundary, bool) {
	cond, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok {
		return goBoundary{}, false
	}
	param, op, value, ok := goBoundaryParamValue(cond)
	if !ok {
		return goBoundary{}, false
	}
	ret := firstGoReturnExpr(ifStmt.Body)
	if ret == "" {
		return goBoundary{}, false
	}
	return goBoundary{
		Param:      param,
		Op:         op,
		Value:      value,
		Condition:  strings.Join([]string{param, op, value}, " "),
		ReturnExpr: ret,
	}, true
}

func goBoundaryParamValue(cond *ast.BinaryExpr) (string, string, string, bool) {
	op := cond.Op.String()
	if !goBoundaryOperator(op) {
		return "", "", "", false
	}
	if ident, ok := cond.X.(*ast.Ident); ok {
		if value, ok := goBoundaryLiteral(cond.Y); ok {
			return ident.Name, op, value, true
		}
	}
	if ident, ok := cond.Y.(*ast.Ident); ok {
		if value, ok := goBoundaryLiteral(cond.X); ok {
			inverted, ok := invertGoBoundaryOperator(op)
			if !ok {
				return "", "", "", false
			}
			return ident.Name, inverted, value, true
		}
	}
	return "", "", "", false
}

func goBoundaryOperator(op string) bool {
	switch op {
	case "==", "!=", ">", "<", ">=", "<=":
		return true
	default:
		return false
	}
}

func invertGoBoundaryOperator(op string) (string, bool) {
	switch op {
	case "==", "!=":
		return op, true
	case ">":
		return "<", true
	case "<":
		return ">", true
	case ">=":
		return "<=", true
	case "<=":
		return ">=", true
	default:
		return "", false
	}
}

func goBoundaryLiteral(expr ast.Expr) (string, bool) {
	switch v := expr.(type) {
	case *ast.BasicLit:
		return v.Value, true
	case *ast.Ident:
		if v.Name == "true" || v.Name == "false" || v.Name == "nil" {
			return v.Name, true
		}
	}
	return "", false
}

func firstGoReturnExpr(body *ast.BlockStmt) string {
	if body == nil {
		return ""
	}
	for _, stmt := range body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			continue
		}
		return exprToString(ret.Results[0])
	}
	return ""
}

func goTypeSupportsExactSeed(typ string) bool {
	return strings.HasPrefix(typ, "int") ||
		strings.HasPrefix(typ, "uint") ||
		strings.HasPrefix(typ, "float") ||
		typ == "string" ||
		typ == "bool"
}

func goReturnExprIsSafe(expr string) bool {
	if expr == "" || strings.ContainsAny(expr, "\n;{}[]") {
		return false
	}
	for _, blocked := range []string{"(", ")", ".", "*", "&", "<-", "chan ", "func"} {
		if strings.Contains(expr, blocked) {
			return false
		}
	}
	return true
}

func goArgValue(p paramInfo, _ int) string {
	name := strings.ToLower(p.Name)
	compact := strings.ReplaceAll(strings.ReplaceAll(name, "_", ""), "-", "")

	if p.Type == "bool" {
		return "true"
	}
	if p.Type == "string" {
		return "\"test\""
	}
	if strings.HasPrefix(p.Type, "float") {
		if compact == "b" || compact == "y" {
			return "2.0"
		}
		return "1.0"
	}
	if strings.HasPrefix(p.Type, "int") || strings.HasPrefix(p.Type, "uint") {
		if compact == "b" || compact == "y" {
			return "2"
		}
		return "1"
	}
	return zeroValue(p.Type)
}

// substituteType 把类型参数名替换为具体类型
func substituteType(typ string, substMap map[string]string) string {
	for name, concrete := range substMap {
		if typ == name {
			return concrete
		}
		// 处理复合类型中的类型参数，如 []T, *T, map[K]V 等
		if strings.Contains(typ, name) {
			typ = strings.ReplaceAll(typ, name, concrete)
		}
	}
	return typ
}

func zeroValue(typ string) string {
	switch {
	case typ == "any", typ == "interface{}":
		return "nil"
	case strings.HasPrefix(typ, "int"), strings.HasPrefix(typ, "uint"), strings.HasPrefix(typ, "float"):
		return "0"
	case typ == "string":
		return "\"\""
	case typ == "bool":
		return "false"
	case typ == "error":
		return "nil"
	case strings.HasPrefix(typ, "chan "):
		return "nil"
	case strings.HasPrefix(typ, "*"):
		return "nil"
	case strings.HasPrefix(typ, "[]"):
		return "nil"
	case strings.HasPrefix(typ, "map["):
		return "nil"
	case strings.HasPrefix(typ, "func"):
		return "nil"
	case strings.Contains(typ, "<-chan"):
		return "nil"
	default:
		return fmt.Sprintf("%s{}", typ)
	}
}

func exprToString(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.BasicLit:
		return v.Value
	case *ast.BinaryExpr:
		return exprToString(v.X) + " " + v.Op.String() + " " + exprToString(v.Y)
	case *ast.UnaryExpr:
		return v.Op.String() + exprToString(v.X)
	case *ast.ParenExpr:
		return "(" + exprToString(v.X) + ")"
	case *ast.SelectorExpr:
		return exprToString(v.X) + "." + v.Sel.Name
	case *ast.CallExpr:
		args := make([]string, len(v.Args))
		for i, arg := range v.Args {
			args[i] = exprToString(arg)
		}
		return exprToString(v.Fun) + "(" + strings.Join(args, ", ") + ")"
	case *ast.StarExpr:
		return "*" + exprToString(v.X)
	case *ast.ArrayType:
		return "[]" + exprToString(v.Elt)
	case *ast.MapType:
		return "map[" + exprToString(v.Key) + "]" + exprToString(v.Value)
	case *ast.ChanType:
		switch v.Dir {
		case ast.SEND:
			return "chan<- " + exprToString(v.Value)
		case ast.RECV:
			return "<-chan " + exprToString(v.Value)
		default:
			return "chan " + exprToString(v.Value)
		}
	case *ast.InterfaceType:
		return "any"
	case *ast.FuncType:
		return "func()"
	case *ast.Ellipsis:
		// 变参 ...T 在参数列表中单独处理，这里返回底层类型
		return "..." + exprToString(v.Elt)
	case *ast.IndexExpr:
		// 泛型实例化：Foo[int]
		return exprToString(v.X) + "[" + exprToString(v.Index) + "]"
	case *ast.IndexListExpr:
		// 多类型参数泛型实例化：Foo[int, string]
		types := make([]string, len(v.Indices))
		for i, idx := range v.Indices {
			types[i] = exprToString(idx)
		}
		return exprToString(v.X) + "[" + strings.Join(types, ", ") + "]"
	default:
		return "any"
	}
}
