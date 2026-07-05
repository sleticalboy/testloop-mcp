package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"

	"github.com/binlee/testloop-mcp/types"
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

		funcs = append(funcs, info)
	}

	if task != nil {
		funcs = filterGoFuncsForCoverageTask(funcs, task)
	}

	if len(funcs) == 0 && len(interfaces) == 0 {
		return "// 未发现需要生成测试的 exported 函数或接口", nil
	}

	// 检测是否需要 reflect
	needReflect := false
	for _, fn := range funcs {
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

	// 定义测试用例结构体
	sb.WriteString("\ttype testCase struct {\n")
	sb.WriteString("\t\tname string\n")
	sb.WriteString("\t\tskip bool\n")
	for _, p := range fn.Params {
		sb.WriteString(fmt.Sprintf("\t\t%s %s\n", p.Name, p.Type))
	}
	for _, r := range fn.Returns {
		sb.WriteString(fmt.Sprintf("\t\t%s %s\n", r.Name, r.Type))
	}
	sb.WriteString("\t}\n\n")

	// 测试用例列表
	seedCase, hasSeedCase := goSeedTestCase(fn)
	sb.WriteString("\ttests := []testCase{\n")
	sb.WriteString("\t\t{\n")
	if hasSeedCase {
		sb.WriteString(fmt.Sprintf("\t\t\tname: %q,\n", goCoverageTaskCaseName(task, "simple")))
		sb.WriteString("\t\t\tskip: false,\n")
		for _, p := range fn.Params {
			sb.WriteString(fmt.Sprintf("\t\t\t%s: %s,\n", p.Name, seedCase.Inputs[p.Name]))
		}
		for _, r := range fn.Returns {
			sb.WriteString(fmt.Sprintf("\t\t\t%s: %s,\n", r.Name, seedCase.Outputs[r.Name]))
		}
	} else {
		sb.WriteString(fmt.Sprintf("\t\t\tname: %q,\n", goCoverageTaskCaseName(task, "todo")))
		sb.WriteString("\t\t\tskip: true, // TODO: 填写有意义的输入和期望值后改为 false\n")
		for _, p := range fn.Params {
			sb.WriteString(fmt.Sprintf("\t\t\t%s: %s,\n", p.Name, zeroValue(p.Type)))
		}
		for _, r := range fn.Returns {
			sb.WriteString(fmt.Sprintf("\t\t\t%s: %s,\n", r.Name, zeroValue(r.Type)))
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
			sb.WriteString(fmt.Sprintf("\t\t\tif tt.%s == nil {\n\t\t\t\tt.Skip(\"%s is nil, fill in test data\")\n\t\t\t}\n", p.Name, p.Name))
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
			args[i] = fmt.Sprintf("tt.%s...", p.Name)
		} else {
			args[i] = fmt.Sprintf("tt.%s", p.Name)
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
	if len(fn.Returns) == 0 {
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
				sb.WriteString(fmt.Sprintf("\t\t\tif !reflect.DeepEqual(got, tt.%s) {\n", fn.Returns[0].Name))
			} else {
				sb.WriteString(fmt.Sprintf("\t\t\tif got != tt.%s {\n", fn.Returns[0].Name))
			}
			sb.WriteString(fmt.Sprintf("\t\t\t\tt.Errorf(\"got %%v, want %%v\", got, tt.%s)\n", fn.Returns[0].Name))
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
				sb.WriteString(fmt.Sprintf("\t\t\tif !reflect.DeepEqual(%s, tt.%s) {\n", returnVars[i], r.Name))
				sb.WriteString(fmt.Sprintf("\t\t\t\tt.Errorf(\"%s: got %%v, want %%v\", %s, tt.%s)\n", r.Name, returnVars[i], r.Name))
				sb.WriteString("\t\t\t}\n")
			} else {
				sb.WriteString(fmt.Sprintf("\t\t\tif %s != tt.%s {\n", returnVars[i], r.Name))
				sb.WriteString(fmt.Sprintf("\t\t\t\tt.Errorf(\"%s: got %%v, want %%v\", %s, tt.%s)\n", r.Name, returnVars[i], r.Name))
				sb.WriteString("\t\t\t}\n")
			}
		}
	}

	sb.WriteString("\t\t})\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	return sb.String()
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

type goSeedCase struct {
	Inputs  map[string]string
	Outputs map[string]string
}

func goSeedTestCase(fn funcInfo) (goSeedCase, bool) {
	if fn.IsMethod || fn.IsVariadic || len(fn.Returns) != 1 || fn.Returns[0].Type == "error" {
		return goSeedCase{}, false
	}
	if !goTypeSupportsExactSeed(fn.Returns[0].Type) || fn.ReturnExpr == "" || !goReturnExprIsSafe(fn.ReturnExpr) {
		return goSeedCase{}, false
	}

	seed := goSeedCase{Inputs: map[string]string{}, Outputs: map[string]string{}}
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
	case strings.HasPrefix(typ, "int"), strings.HasPrefix(typ, "uint"), strings.HasPrefix(typ, "float"):
		return "0"
	case typ == "string":
		return "\"\""
	case typ == "bool":
		return "false"
	case typ == "error":
		return "nil"
	case typ == "any", typ == "interface{}":
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
