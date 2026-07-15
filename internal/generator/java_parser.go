package generator

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

// ============================================================
// Java tree-sitter parser
// ============================================================

// javaFuncInfo Java 方法信息
type javaFuncInfo struct {
	Name          string
	ClassName     string
	Params        []javaParamInfo
	ReturnType    string // void / int / String / boolean / 自定义类型
	IsStatic      bool
	IsPublic      bool
	IsVoid        bool
	IsConstructor bool
	Throws        []string // throws 的异常类型列表
	IsGeneric     bool     // 是否是泛型方法
	IsEnum        bool
	Line          int
}

type javaParamInfo struct {
	Name string
	Type string // 类型全名，如 "int", "String", "List<String>"
}

// javaClassInfo Java 类信息
type javaClassInfo struct {
	Name           string
	IsPublic       bool
	HasConstructor bool
	Constructors   []javaFuncInfo
	IsEnum         bool
	Line           int
}

// parseJavaWithTreeSitter 用 tree-sitter 解析 Java 源码
func parseJavaWithTreeSitter(source []byte) (funcs []javaFuncInfo, classes []javaClassInfo) {
	lang := java.GetLanguage()
	parser := sitter.NewParser()
	parser.SetLanguage(lang)
	tree := parser.Parse(nil, source)
	defer tree.Close()

	root := tree.RootNode()
	javaWalk(root, source, &funcs, &classes)
	return
}

// javaWalk 遍历 Java AST
func javaWalk(node *sitter.Node, source []byte, funcs *[]javaFuncInfo, classes *[]javaClassInfo) {
	if node == nil {
		return
	}

	switch node.Type() {
	case "class_declaration", "enum_declaration", "interface_declaration":
		info := javaExtractClassInfo(node, source)
		*classes = append(*classes, info)
		// 类体里的方法由 class_body 处理
		javaWalkClassBody(node, source, funcs, &info)
		return
	case "method_declaration":
		*funcs = append(*funcs, javaExtractMethodInfo(node, source))
		return
	case "constructor_declaration":
		info := javaExtractConstructorInfo(node, source)
		info.IsConstructor = true
		*funcs = append(*funcs, info)
		return
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		javaWalk(node.Child(i), source, funcs, classes)
	}
}

// javaWalkClassBody 遍历类体，提取方法（包括内部类的）
func javaWalkClassBody(node *sitter.Node, source []byte, funcs *[]javaFuncInfo, classInfo *javaClassInfo) {
	body := node.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := 0; i < int(body.ChildCount()); i++ {
		javaWalkClassMember(body.Child(i), source, funcs, classInfo)
	}
}

func javaWalkClassMember(node *sitter.Node, source []byte, funcs *[]javaFuncInfo, classInfo *javaClassInfo) {
	if node == nil {
		return
	}
	switch node.Type() {
	case "method_declaration":
		info := javaExtractMethodInfo(node, source)
		// 标记是否属于当前类（通过 IsStatic 等方式无法区分，这里简单处理）
		info.ClassName = classInfo.Name
		info.IsEnum = classInfo.IsEnum
		*funcs = append(*funcs, info)
		return
	case "constructor_declaration":
		info := javaExtractConstructorInfo(node, source)
		info.Name = classInfo.Name
		info.ClassName = classInfo.Name
		info.IsConstructor = true
		info.IsEnum = classInfo.IsEnum
		*funcs = append(*funcs, info)
		return
	case "class_declaration", "enum_declaration", "interface_declaration":
		// 内部类，递归处理
		javaWalk(node, source, funcs, &[]javaClassInfo{})
		return
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		javaWalkClassMember(node.Child(i), source, funcs, classInfo)
	}
}

// javaExtractClassInfo 提取类信息
func javaExtractClassInfo(node *sitter.Node, source []byte) javaClassInfo {
	info := javaClassInfo{Line: int(node.StartPoint().Row) + 1, IsEnum: node.Type() == "enum_declaration"}
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		info.Name = nameNode.Content(source)
	}
	info.IsPublic = javaHasModifier(node, "public")
	return info
}

// javaExtractMethodInfo 提取方法信息
func javaExtractMethodInfo(node *sitter.Node, source []byte) javaFuncInfo {
	info := javaFuncInfo{Line: int(node.StartPoint().Row) + 1}

	// 方法名
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		info.Name = nameNode.Content(source)
	}

	// 修饰符
	info.IsPublic = javaHasModifier(node, "public")
	info.IsStatic = javaHasModifier(node, "static")

	// 返回类型
	returnTypeNode := node.ChildByFieldName("type")
	if returnTypeNode != nil {
		info.ReturnType = strings.TrimSpace(returnTypeNode.Content(source))
	} else {
		// 检查是否是 void_type
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "void_type" {
				info.ReturnType = "void"
				info.IsVoid = true
				break
			}
		}
	}
	if info.ReturnType == "void" {
		info.IsVoid = true
	}

	// 参数
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		info.Params = javaExtractParams(paramsNode, source)
	}

	// throws
	info.Throws = javaExtractThrows(node, source)

	// 泛型方法检测（类型参数）
	typeParams := node.ChildByFieldName("type_parameters")
	if typeParams != nil {
		info.IsGeneric = true
	}

	return info
}

// javaExtractConstructorInfo 提取构造函数信息
func javaExtractConstructorInfo(node *sitter.Node, source []byte) javaFuncInfo {
	info := javaFuncInfo{
		IsConstructor: true,
		IsPublic:      javaHasModifier(node, "public"),
		Line:          int(node.StartPoint().Row) + 1,
	}

	// 构造函数名从 class 名获取，这里用父节点的 class_declaration name
	// 简化处理：从 parameters 提取参数
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		info.Params = javaExtractParams(paramsNode, source)
	}

	return info
}

// javaExtractParams 提取参数列表
func javaExtractParams(node *sitter.Node, source []byte) []javaParamInfo {
	var params []javaParamInfo
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "formal_parameter" {
			p := javaParamInfo{}
			// 类型
			typeNode := child.ChildByFieldName("type")
			if typeNode != nil {
				p.Type = typeNode.Content(source)
			}
			// 名字
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				p.Name = nameNode.Content(source)
			}
			params = append(params, p)
		} else if child.Type() == "spread_parameter" {
			// varargs: String... args
			p := javaParamInfo{}
			typeNode := child.ChildByFieldName("type")
			if typeNode != nil {
				p.Type = typeNode.Content(source) + "..."
			}
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				p.Name = nameNode.Content(source)
			}
			if p.Type == "" || p.Name == "" {
				parts := strings.Fields(strings.TrimSpace(child.Content(source)))
				if len(parts) >= 2 {
					p.Type = parts[0]
					p.Name = parts[len(parts)-1]
				}
			}
			params = append(params, p)
		}
	}
	return params
}

// javaExtractThrows 提取 throws 声明
func javaExtractThrows(node *sitter.Node, source []byte) []string {
	// tree-sitter-java 中 throws 是 siblings 节点
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "throws" {
			// throws 后面的类型名
			var exceptions []string
			for j := 0; j < int(child.ChildCount()); j++ {
				exNode := child.Child(j)
				if exNode.Type() == "type_identifier" || exNode.Type() == "scoped_type_identifier" {
					exceptions = append(exceptions, exNode.Content(source))
				}
			}
			return exceptions
		}
	}
	return nil
}

// javaHasModifier 检测是否有指定修饰符（public/private/static/...）
func javaHasModifier(node *sitter.Node, mod string) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "modifiers" {
			for j := 0; j < int(child.ChildCount()); j++ {
				sub := child.Child(j)
				if sub.Type() == mod {
					return true
				}
			}
		}
	}
	return false
}

// javaIsTestHelper 判断是否是测试辅助方法
func javaIsTestHelper(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, "test") ||
		lower == "main" ||
		lower == "equals" ||
		lower == "hashcode" ||
		lower == "tostring" ||
		strings.HasPrefix(name, "get") && len(name) > 3 && name[3] >= 'A' && name[3] <= 'Z' ||
		strings.HasPrefix(name, "set") && len(name) > 3 && name[3] >= 'A' && name[3] <= 'Z'
}

// javaInferDefaultValue 根据 Java 类型推断默认值
func javaInferDefaultValue(typ string) string {
	if typ == "" {
		return "null"
	}
	switch {
	case typ == "int", typ == "long", typ == "short", typ == "byte":
		return "0"
	case typ == "float", typ == "double":
		return "0.0"
	case typ == "boolean":
		return "false"
	case typ == "char":
		return "'a'"
	case typ == "String" || typ == "CharSequence":
		return "\"test\""
	case typ == "List", strings.HasPrefix(typ, "List<"):
		return "java.util.Collections.emptyList()"
	case typ == "Map", strings.HasPrefix(typ, "Map<"):
		return "java.util.Collections.emptyMap()"
	case typ == "Set", strings.HasPrefix(typ, "Set<"):
		return "java.util.Collections.emptySet()"
	case typ == "Optional", strings.HasPrefix(typ, "Optional<"):
		return "java.util.Optional.empty()"
	default:
		// 自定义类型，尝试 new
		if strings.Contains(typ, ".") {
			return "null" // 外部类型，不好推断
		}
		return fmt.Sprintf("new %s()", typ)
	}
}

// javaInferAssert 根据返回类型推断断言方式
func javaInferAssert(returnType string, varName string) string {
	if returnType == "void" || returnType == "" {
		return ""
	}
	switch {
	case returnType == "int", returnType == "long", returnType == "short", returnType == "byte":
		return fmt.Sprintf("assertEquals(0, %s);", varName)
	case returnType == "float", returnType == "double":
		return fmt.Sprintf("assertEquals(0.0, %s, 0.001);", varName)
	case returnType == "boolean":
		return fmt.Sprintf("assertTrue(%s);", varName)
	case returnType == "String":
		return fmt.Sprintf("assertNotNull(%s);", varName)
	default:
		return fmt.Sprintf("assertNotNull(%s);", varName)
	}
}
