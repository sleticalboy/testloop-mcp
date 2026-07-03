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
	Name    string
	Params  []paramInfo
	Returns []paramInfo
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

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if !fn.Name.IsExported() {
			continue
		}
		if strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}

		info := funcInfo{Name: fn.Name.Name}
		
		// 解析参数
		if fn.Type.Params != nil {
			for i, p := range fn.Type.Params.List {
				typ := exprToString(p.Type)
				for _, name := range p.Names {
					info.Params = append(info.Params, paramInfo{
						Name: name.Name,
						Type: typ,
					})
				}
				_ = i
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
	sb.WriteString(fmt.Sprintf("\t\tt.Run(%q, func(t *testing.T) {\n", fn.Name))
	
	// 调用函数
	args := make([]string, len(fn.Params))
	for i, p := range fn.Params {
		args[i] = fmt.Sprintf("tt.%s", p.Name)
	}
	
	if len(fn.Returns) == 0 {
		sb.WriteString(fmt.Sprintf("\t\t\t%s(%s)\n", fn.Name, strings.Join(args, ", ")))
	} else if len(fn.Returns) == 1 {
		sb.WriteString(fmt.Sprintf("\t\t\tgot := %s(%s)\n", fn.Name, strings.Join(args, ", ")))
		sb.WriteString(fmt.Sprintf("\t\t\tif got != tt.%s {\n", fn.Returns[0].Name))
		sb.WriteString(fmt.Sprintf("\t\t\t\tt.Errorf(\"got %%v, want %%v\", got, tt.%s)\n", fn.Returns[0].Name))
		sb.WriteString("\t\t\t}\n")
	} else {
		// 多个返回值
		returnVars := make([]string, len(fn.Returns))
		for i := range fn.Returns {
			returnVars[i] = fmt.Sprintf("got%d", i)
		}
		sb.WriteString(fmt.Sprintf("\t\t\t%s := %s(%s)\n", strings.Join(returnVars, ", "), fn.Name, strings.Join(args, ", ")))
		for i, r := range fn.Returns {
			sb.WriteString(fmt.Sprintf("\t\t\tif %s != tt.%s {\n", returnVars[i], r.Name))
			sb.WriteString(fmt.Sprintf("\t\t\t\tt.Errorf(\"%s got %%v, want %%v\", %s, tt.%s)\n", r.Name, returnVars[i], r.Name))
			sb.WriteString("\t\t\t}\n")
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
	default:
		return "nil"
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
	default:
		return "any"
	}
}
