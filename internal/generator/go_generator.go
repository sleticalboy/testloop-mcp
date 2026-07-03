package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

type funcInfo struct {
	Name       string
	Params     []paramInfo
	Returns    []paramInfo
	Receiver   string // 方法接收者（如果有）
	ReceiverType string // 接收者类型
	IsMethod   bool
}

type paramInfo struct {
	Name string
	Type string
}

// GenerateTests 读取源文件，用 AST 分析生成表驱动测试代码
func GenerateTests(srcPath string) (string, error) {
	fs := token.NewFileSet()
	node, err := parser.ParseFile(fs, srcPath, nil, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("解析源文件失败: %w", err)
	}

	pkgName := node.Name.Name

	var funcs []funcInfo
	var structs []string

	// 第一遍：收集结构体定义（用于生成测试数据）
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
			if _, ok := typeSpec.Type.(*ast.StructType); ok {
				structs = append(structs, typeSpec.Name.Name)
			}
		}
	}

	// 第二遍：收集函数和方法的 AST 信息
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}

		info := funcInfo{Name: fn.Name.Name}
		
		// 检查方法接收者
		if fn.Recv != nil {
			info.IsMethod = true
			for _, field := range fn.Recv.List {
				recvType := exprToString(field.Type)
				info.ReceiverType = recvType
				// 提取接收者变量名
				for _, name := range field.Names {
					info.Receiver = name.Name
				}
			}
		}
		
		// 解析参数
		if fn.Type.Params != nil {
			for _, p := range fn.Type.Params.List {
				typ := exprToString(p.Type)
				for _, name := range p.Names {
					info.Params = append(info.Params, paramInfo{
						Name: name.Name,
						Type: typ,
					})
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
				info.Returns = append(info.Returns, paramInfo{
					Name: name,
					Type: typ,
				})
			}
		}
		
		funcs = append(funcs, info)
	}

	if len(funcs) == 0 {
		return "// 未发现需要生成测试的 exported 函数", nil
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("package %s\n\n", pkgName))
	buf.WriteString("import (\n")
	buf.WriteString("\t\"testing\"\n")
	buf.WriteString(")\n\n")
	
	// 为结构体生成测试辅助函数（如果需要）
	for _, s := range structs {
		buf.WriteString(fmt.Sprintf("// makeTest%s 创建测试用的 %s 实例\n", s, s))
		buf.WriteString(fmt.Sprintf("func makeTest%s() %s {\n", s, s))
		buf.WriteString(fmt.Sprintf("\treturn %s{}\n", s))
		buf.WriteString("}\n\n")
	}

	for _, fn := range funcs {
		buf.WriteString(genTableDrivenTest(fn))
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.String(), fmt.Errorf("格式化失败: %w", err)
	}
	return string(formatted), nil
}

func genTableDrivenTest(fn funcInfo) string {
	var sb strings.Builder

	// 生成函数注释
	sb.WriteString(fmt.Sprintf("// Test%s 测试 %s\n", fn.Name, fn.Name))
	sb.WriteString(fmt.Sprintf("func Test%s(t *testing.T) {\n", fn.Name))

	// 定义测试用例结构体
	sb.WriteString("\ttype testCase struct {\n")
	for _, p := range fn.Params {
		sb.WriteString(fmt.Sprintf("\t\t%s %s\n", p.Name, p.Type))
	}
	for _, r := range fn.Returns {
		sb.WriteString(fmt.Sprintf("\t\t%s %s\n", r.Name, r.Type))
	}
	sb.WriteString("\t}\n\n")

	// 测试用例列表
	sb.WriteString("\ttests := []testCase{\n")
	sb.WriteString("\t\t{\n")
	sb.WriteString("\t\t\t// TODO: 填写测试用例\n")
	for _, p := range fn.Params {
		sb.WriteString(fmt.Sprintf("\t\t\t%s: %s,\n", p.Name, zeroValue(p.Type)))
	}
	for _, r := range fn.Returns {
		sb.WriteString(fmt.Sprintf("\t\t\t%s: %s,\n", r.Name, zeroValue(r.Type)))
	}
	sb.WriteString("\t\t},\n")
	sb.WriteString("\t}\n\n")

	// 执行测试
	sb.WriteString("\tfor _, tt := range tests {\n")
	
	// 测试方法或函数
	if fn.IsMethod {
		sb.WriteString(fmt.Sprintf("\t\tt.Run(%q, func(t *testing.T) {\n", fmt.Sprintf("%s_%s", fn.ReceiverType, fn.Name)))
		// 创建接收者实例 - 正确处理指针类型
		recvType := fn.ReceiverType
		if strings.HasPrefix(recvType, "*") {
			// 指针接收者: s := &UserService{}
			sb.WriteString(fmt.Sprintf("\t\t\t%s := &%s{}\n", fn.Receiver, recvType[1:]))
		} else {
			// 值接收者: s := UserService{}
			sb.WriteString(fmt.Sprintf("\t\t\t%s := %s{}\n", fn.Receiver, recvType))
		}
	} else {
		sb.WriteString(fmt.Sprintf("\t\tt.Run(%q, func(t *testing.T) {\n", fn.Name))
	}
	
	// 调用函数/方法
	args := make([]string, len(fn.Params))
	for i, p := range fn.Params {
		args[i] = fmt.Sprintf("tt.%s", p.Name)
	}
	
	callExpr := ""
	if fn.IsMethod {
		callExpr = fmt.Sprintf("%s.%s(%s)", fn.Receiver, fn.Name, strings.Join(args, ", "))
	} else {
		callExpr = fmt.Sprintf("%s(%s)", fn.Name, strings.Join(args, ", "))
	}
	
	if len(fn.Returns) == 0 {
		sb.WriteString(fmt.Sprintf("\t\t\t%s\n", callExpr))
	} else if len(fn.Returns) == 1 {
		sb.WriteString(fmt.Sprintf("\t\t\tgot := %s\n", callExpr))
		sb.WriteString(fmt.Sprintf("\t\t\tif got != tt.%s {\n", fn.Returns[0].Name))
		sb.WriteString(fmt.Sprintf("\t\t\t\tt.Errorf(\"got %%v, want %%v\", got, tt.%s)\n", fn.Returns[0].Name))
		sb.WriteString("\t\t\t}\n")
		
		// 如果有 error 返回值，检查错误
		if fn.Returns[0].Type == "error" {
			sb.WriteString("\t\t\tif got != nil {\n")
			sb.WriteString("\t\t\t\tt.Errorf(\"unexpected error: %v\", got)\n")
			sb.WriteString("\t\t\t}\n")
		}
	} else {
		// 多个返回值
		returnVars := make([]string, len(fn.Returns))
		for i := range fn.Returns {
			returnVars[i] = fmt.Sprintf("got%d", i)
		}
		sb.WriteString(fmt.Sprintf("\t\t\t%s := %s\n", strings.Join(returnVars, ", "), callExpr))
		
		for i, r := range fn.Returns {
			if r.Type == "error" {
				// error 返回值特殊处理
				sb.WriteString(fmt.Sprintf("\t\t\tif %s != nil {\n", returnVars[i]))
				sb.WriteString(fmt.Sprintf("\t\t\t\tt.Errorf(\"unexpected error: %%v\", %s)\n", returnVars[i]))
				sb.WriteString("\t\t\t}\n")
			} else {
				sb.WriteString(fmt.Sprintf("\t\t\tif %s != tt.%s {\n", returnVars[i], r.Name))
				sb.WriteString(fmt.Sprintf("\t\t\t\tt.Errorf(\"%s got %%v, want %%v\", %s, tt.%s)\n", r.Name, returnVars[i], r.Name))
				sb.WriteString("\t\t\t}\n")
			}
		}
	}
	
	sb.WriteString("\t\t})\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	return sb.String()
}

func zeroValue(typ string) string {
	switch {
	case strings.HasPrefix(typ, "int"), strings.HasPrefix(typ, "int8"), strings.HasPrefix(typ, "int16"), strings.HasPrefix(typ, "int32"), strings.HasPrefix(typ, "int64"):
		return "0"
	case strings.HasPrefix(typ, "uint"):
		return "0"
	case strings.HasPrefix(typ, "float"):
		return "0.0"
	case typ == "string":
		return "\"\""
	case typ == "bool":
		return "false"
	case typ == "error":
		return "nil"
	case strings.HasPrefix(typ, "*"):
		return "nil"
	case strings.HasPrefix(typ, "[]"):
		return "nil"
	case strings.HasPrefix(typ, "map["):
		return "nil"
	case typ == "any":
		return "nil"
	default:
		// 假设是结构体类型
		return fmt.Sprintf("%s{}", typ)
	}
}

func exprToString(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return exprToString(v.X) + "." + v.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(v.X)
	case *ast.ArrayType:
		return "[]" + exprToString(v.Elt)
	case *ast.MapType:
		return "map[" + exprToString(v.Key) + "]" + exprToString(v.Value)
	case *ast.ChanType:
		return "chan " + exprToString(v.Value)
	case *ast.InterfaceType:
		return "any"
	case *ast.FuncType:
		return "func()"
	default:
		return "any"
	}
}
