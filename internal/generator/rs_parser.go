package generator

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/rust"
)

// ============================================================
// Rust tree-sitter parser
// ============================================================

// rsFuncInfo Rust 函数信息
type rsFuncInfo struct {
	Name       string
	Owner      string // impl/trait owner, e.g. Validator for Validator.check
	Params     []rsParamInfo
	ReturnType string // 原始返回类型字符串，如 "i32", "Result<String, Error>", "Option<i32>"
	IsAsync    bool
	IsMethod   bool // 是否是 impl 块里的方法（第一个参数是 self）
	IsPub      bool
	Generics   string // 泛型参数，如 "<T>"，无则为 ""
	HasResult  bool   // 返回类型是否是 Result<...>
	HasOption  bool   // 返回类型是否是 Option<...>
	HasSelf    bool   // 是否有 self 参数
	Line       int
}

type rsParamInfo struct {
	Name      string
	Type      string
	IsSelf    bool // self / &self / &mut self
	IsMutSelf bool // &mut self
}

// rsStructInfo Rust 结构体/enum/trait 信息（用于生成测试时的实例化）
type rsStructInfo struct {
	Name   string
	IsPub  bool
	HasNew bool // 是否有 new() 关联函数
	Line   int
}

// parseRustWithTreeSitter 用 tree-sitter 解析 Rust 源码
func parseRustWithTreeSitter(source []byte) (funcs []rsFuncInfo, structs []rsStructInfo) {
	lang := rust.GetLanguage()
	parser := sitter.NewParser()
	parser.SetLanguage(lang)
	tree := parser.Parse(nil, source)
	defer tree.Close()

	root := tree.RootNode()
	rsWalk(root, source, &funcs, &structs)
	return
}

// rsWalk 遍历 Rust AST
func rsWalk(node *sitter.Node, source []byte, funcs *[]rsFuncInfo, structs *[]rsStructInfo) {
	if node == nil {
		return
	}

	switch node.Type() {
	case "function_item":
		if info := rsExtractFuncInfo(node, source); !rsIsTestHelper(info.Name) {
			*funcs = append(*funcs, info)
		}
	case "impl_item":
		// impl 块里的方法
		rsWalkImpl(node, source, funcs, structs)
		return // impl_item 自己处理子节点
	case "struct_item", "enum_item":
		if info := rsExtractStructInfo(node, source); info.Name != "" {
			*structs = append(*structs, info)
		}
	case "trait_item":
		// trait 里的方法声明（默认实现也要测）
		rsWalkTrait(node, source, funcs)
		return
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		rsWalk(node.Child(i), source, funcs, structs)
	}
}

// rsWalkImpl 处理 impl 块，提取方法
func rsWalkImpl(node *sitter.Node, source []byte, funcs *[]rsFuncInfo, structs *[]rsStructInfo) {
	// impl 块的类型名
	typeNode := node.ChildByFieldName("type")
	typeName := ""
	if typeNode != nil {
		typeName = typeNode.Content(source)
	}

	// 检查是否有 new() 关联函数
	hasNew := false
	body := node.ChildByFieldName("body")
	if body != nil {
		for i := 0; i < int(body.ChildCount()); i++ {
			child := body.Child(i)
			if child.Type() == "function_item" {
				nameNode := child.ChildByFieldName("name")
				if nameNode != nil && nameNode.Content(source) == "new" {
					hasNew = true
				}
			}
		}
	}

	// 记录结构体信息
	if typeName != "" {
		*structs = append(*structs, rsStructInfo{
			Name:   typeName,
			IsPub:  true,
			HasNew: hasNew,
		})
	}

	// 提取方法
	if body != nil {
		for i := 0; i < int(body.ChildCount()); i++ {
			child := body.Child(i)
			if child.Type() == "function_item" {
				info := rsExtractFuncInfo(child, source)
				info.Owner = typeName
				info.IsMethod = info.HasSelf
				// 关联函数（非方法）用 TypeName::func() 调用
				if !info.IsMethod && info.Name != "new" && info.Name != "default" {
					// 保留，关联函数也需要测试
				}
				if !rsIsTestHelper(info.Name) {
					*funcs = append(*funcs, info)
				}
			}
		}
	}
}

// rsWalkTrait 处理 trait 块
func rsWalkTrait(node *sitter.Node, source []byte, funcs *[]rsFuncInfo) {
	body := node.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)
		if child.Type() == "function_item" {
			info := rsExtractFuncInfo(child, source)
			info.IsMethod = info.HasSelf
			if !rsIsTestHelper(info.Name) {
				*funcs = append(*funcs, info)
			}
		}
	}
}

// rsExtractFuncInfo 从 function_item 节点提取函数信息
func rsExtractFuncInfo(node *sitter.Node, source []byte) rsFuncInfo {
	info := rsFuncInfo{Line: int(node.StartPoint().Row) + 1}

	// 函数名
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		info.Name = nameNode.Content(source)
	}

	// async 检测
	info.IsAsync = rsHasAsyncModifier(node)

	// pub 检测
	info.IsPub = rsHasPubModifier(node)

	// 泛型参数
	genericsNode := node.ChildByFieldName("type_parameters")
	if genericsNode != nil {
		info.Generics = genericsNode.Content(source)
	}

	// 参数
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		info.Params = rsExtractParams(paramsNode, source)
	}

	// 返回类型
	returnTypeNode := node.ChildByFieldName("return_type")
	if returnTypeNode != nil {
		info.ReturnType = strings.TrimSpace(returnTypeNode.Content(source))
		info.HasResult = strings.HasPrefix(info.ReturnType, "Result<")
		info.HasOption = strings.HasPrefix(info.ReturnType, "Option<")
	}

	// self 检测
	for _, p := range info.Params {
		if p.IsSelf {
			info.HasSelf = true
			break
		}
	}

	return info
}

// rsExtractParams 提取参数列表
func rsExtractParams(node *sitter.Node, source []byte) []rsParamInfo {
	var params []rsParamInfo
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "parameter" {
			p := rsExtractParam(child, source)
			params = append(params, p)
		} else if child.Type() == "self_parameter" {
			params = append(params, rsExtractSelfParam(child, source))
		}
	}
	return params
}

// rsExtractParam 提取普通参数
func rsExtractParam(node *sitter.Node, source []byte) rsParamInfo {
	p := rsParamInfo{}
	nameNode := node.ChildByFieldName("pattern")
	if nameNode != nil {
		p.Name = nameNode.Content(source)
	}
	typeNode := node.ChildByFieldName("type")
	if typeNode != nil {
		p.Type = typeNode.Content(source)
	}
	return p
}

// rsExtractSelfParam 提取 self 参数
func rsExtractSelfParam(node *sitter.Node, source []byte) rsParamInfo {
	p := rsParamInfo{IsSelf: true}
	content := node.Content(source)
	if strings.Contains(content, "&mut self") {
		p.IsMutSelf = true
		p.Name = "&mut self"
	} else {
		p.Name = "&self"
	}
	return p
}

// rsHasAsyncModifier 检测 async fn
func rsHasAsyncModifier(node *sitter.Node) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "async" {
			return true
		}
	}
	return false
}

// rsHasPubModifier 检测 pub fn
func rsHasPubModifier(node *sitter.Node) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "visibility_modifier" {
			return true
		}
	}
	return false
}

// rsExtractStructInfo 提取结构体/enum 信息
func rsExtractStructInfo(node *sitter.Node, source []byte) rsStructInfo {
	info := rsStructInfo{Line: int(node.StartPoint().Row) + 1}
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		info.Name = nameNode.Content(source)
	}
	info.IsPub = rsHasPubModifier(node)
	return info
}

// rsIsTestHelper 判断是否是测试辅助函数（避免为测试函数本身生成测试）
func rsIsTestHelper(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, "test_") ||
		lower == "main" ||
		lower == "new" && strings.Contains(name, "") // new 是构造函数，可以测，不算 helper
}

// rsInferDefaultValue 根据 Rust 类型推断默认值，用于生成测试代码
func rsInferDefaultValue(typ string) string {
	if typ == "" {
		return "()"
	}
	if strings.HasPrefix(strings.TrimSpace(typ), "&str") {
		return "\"test\""
	}
	// 去掉 & 和 mut
	typ = strings.TrimPrefix(typ, "&")
	typ = strings.TrimPrefix(typ, "mut ")
	typ = strings.TrimSpace(typ)

	switch {
	case typ == "i8", typ == "i16", typ == "i32", typ == "i64", typ == "i128", typ == "isize":
		return "0"
	case typ == "u8", typ == "u16", typ == "u32", typ == "u64", typ == "u128", typ == "usize":
		return "0"
	case typ == "f32", typ == "f64":
		return "0.0"
	case typ == "bool":
		return "false"
	case typ == "char":
		return "'a'"
	case typ == "String":
		return "\"test\".to_string()"
	case strings.HasPrefix(typ, "Option<"):
		return "None"
	case strings.HasPrefix(typ, "Vec<"):
		return "vec![]"
	case strings.HasPrefix(typ, "HashMap<"):
		return "std::collections::HashMap::new()"
	case typ == "()" || typ == "":
		return "()"
	default:
		// 自定义类型，尝试调用 Default::default()
		if strings.Contains(typ, "::") || (len(typ) > 0 && typ[0] >= 'A' && typ[0] <= 'Z') {
			return fmt.Sprintf("%s::default()", typ)
		}
		return fmt.Sprintf("%s::new()", typ)
	}
}

// rsInferReturnValue 根据返回类型推断断言值
func rsInferReturnValue(returnType string) string {
	if returnType == "" {
		return "()"
	}
	switch {
	case returnType == "i32", returnType == "i64":
		return "0"
	case returnType == "f64", returnType == "f32":
		return "0.0"
	case returnType == "bool":
		return "true"
	case returnType == "String":
		return "\"test\".to_string()"
	case strings.HasPrefix(returnType, "Option<"):
		return "Some(0)" // 粗略推断
	case strings.HasPrefix(returnType, "Result<"):
		return "Ok(0)" // 粗略推断
	default:
		return "()"
	}
}
