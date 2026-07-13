package generator

import (
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// ============================================================
// JS / TS tree-sitter parser
// ============================================================

// parseJSWithTreeSitter 用 tree-sitter 解析 JS/TS 源码，返回函数、类和是否 ES Module
func parseJSWithTreeSitter(source []byte, ext string) (funcs []jsFuncInfo, classes []jsClassInfo, isESModule bool) {
	var lang *sitter.Language
	switch ext {
	case ".ts":
		lang = typescript.GetLanguage()
	case ".tsx", ".jsx":
		lang = tsx.GetLanguage()
	default: // .js, .mjs, .cjs
		lang = javascript.GetLanguage()
	}

	parser := sitter.NewParser()
	parser.SetLanguage(lang)
	tree := parser.Parse(nil, source)
	defer tree.Close()

	root := tree.RootNode()
	ctx := &jsParseCtx{tsTypes: jsExtractTSTypeDecls(string(source))}
	jsWalkNode(root, source, ctx)
	jsMarkExportMetadata(string(source), ctx)

	return ctx.funcs, ctx.classes, ctx.isESModule
}

type jsParseCtx struct {
	funcs      []jsFuncInfo
	classes    []jsClassInfo
	isESModule bool
	exported   bool
	tsTypes    map[string]string
}

func jsWalkNode(node *sitter.Node, source []byte, ctx *jsParseCtx) {
	n := int(node.NamedChildCount())
	for i := 0; i < n; i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		jsHandleNode(child, source, ctx)
	}
}

func jsHandleNode(node *sitter.Node, source []byte, ctx *jsParseCtx) {
	switch node.Type() {
	case "export_statement":
		ctx.isESModule = true
		old := ctx.exported
		ctx.exported = true
		jsWalkNode(node, source, ctx)
		ctx.exported = old

	case "function_declaration":
		fn := jsExtractFunction(node, source, ctx.exported)
		jsAttachTSTypeDeclsToFunc(&fn, ctx.tsTypes)
		if fn.Name != "" && !isTestHelper(fn.Name) {
			ctx.funcs = append(ctx.funcs, fn)
		}

	case "lexical_declaration", "variable_declaration":
		n := int(node.NamedChildCount())
		for i := 0; i < n; i++ {
			child := node.NamedChild(i)
			if child == nil {
				continue
			}
			if child.Type() == "variable_declarator" {
				jsHandleDeclarator(child, source, ctx)
			}
		}

	case "class_declaration":
		cls := jsExtractClass(node, source)
		cls.IsExported = ctx.exported
		jsAttachTSTypeDeclsToClass(&cls, ctx.tsTypes)
		if cls.Name != "" {
			ctx.classes = append(ctx.classes, cls)
		}
	}
}

func jsHandleDeclarator(node *sitter.Node, source []byte, ctx *jsParseCtx) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil || nameNode.Type() != "identifier" {
		return
	}
	name := nameNode.Content(source)
	if isTestHelper(name) {
		return
	}

	valueNode := node.ChildByFieldName("value")
	if valueNode == nil {
		return
	}

	switch valueNode.Type() {
	case "arrow_function":
		fn := jsExtractArrowFunction(valueNode, source, name, ctx.exported)
		jsAttachTSTypeDeclsToFunc(&fn, ctx.tsTypes)
		ctx.funcs = append(ctx.funcs, fn)
	case "function_expression":
		fn := jsExtractFunction(valueNode, source, ctx.exported)
		fn.Name = name // 匿名函数表达式，名字来自变量名
		fn.IsArrow = false
		jsAttachTSTypeDeclsToFunc(&fn, ctx.tsTypes)
		ctx.funcs = append(ctx.funcs, fn)
	}
}

func jsMarkExportMetadata(source string, ctx *jsParseCtx) {
	if ctx == nil {
		return
	}
	defaultName := jsDefaultExportName(source)
	for i := range ctx.funcs {
		fn := &ctx.funcs[i]
		if fn.Name == defaultName || jsHasDefaultFunctionExport(source, fn.Name) {
			fn.IsDefault = true
			fn.IsExported = true
			ctx.isESModule = true
		}
	}
	for i := range ctx.classes {
		cls := &ctx.classes[i]
		if cls.Name == defaultName || jsHasDefaultClassExport(source, cls.Name) {
			cls.IsDefault = true
			cls.IsExported = true
			ctx.isESModule = true
		}
		if instance := jsDefaultInstanceForClass(source, cls.Name, defaultName); instance != "" {
			cls.DefaultInstance = instance
			ctx.isESModule = true
		}
	}
	for i := range ctx.classes {
		ctx.classes[i].SourceIsESModule = ctx.isESModule
	}
}

func jsDefaultExportName(source string) string {
	re := regexp.MustCompile(`(?m)\bexport\s+default\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`)
	if match := re.FindStringSubmatch(source); len(match) == 2 {
		return match[1]
	}
	return ""
}

func jsHasDefaultFunctionExport(source, name string) bool {
	if name == "" {
		return false
	}
	re := regexp.MustCompile(`(?m)\bexport\s+default\s+(?:async\s+)?function\s+` + regexp.QuoteMeta(name) + `\b`)
	return re.MatchString(source)
}

func jsHasDefaultClassExport(source, name string) bool {
	if name == "" {
		return false
	}
	re := regexp.MustCompile(`(?m)\bexport\s+default\s+class\s+` + regexp.QuoteMeta(name) + `\b`)
	return re.MatchString(source)
}

func jsDefaultInstanceForClass(source, className, defaultName string) string {
	if className == "" {
		return ""
	}
	inlineRe := regexp.MustCompile(`(?m)\bexport\s+default\s+new\s+` + regexp.QuoteMeta(className) + `\s*\(`)
	if inlineRe.MatchString(source) {
		return strings.ToLower(className[:1]) + className[1:]
	}
	instanceRe := regexp.MustCompile(`(?m)\b(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*new\s+` + regexp.QuoteMeta(className) + `\s*\(`)
	for _, match := range instanceRe.FindAllStringSubmatch(source, -1) {
		if len(match) == 2 && match[1] == defaultName {
			return match[1]
		}
	}
	return ""
}

func jsExtractFunction(node *sitter.Node, source []byte, isExported bool) jsFuncInfo {
	fn := jsFuncInfo{IsExported: isExported}

	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		fn.Name = nameNode.Content(source)
	}
	if paramsNode := node.ChildByFieldName("parameters"); paramsNode != nil {
		fn.Params = jsParseParams(paramsNode, source)
	}

	content := node.Content(source)
	fn.IsAsync = strings.HasPrefix(content, "async ")

	bodyNode := node.ChildByFieldName("body")
	fn.Body = jsExtractBody(bodyNode, source)
	fn.Analysis = analyzeJSBody(fn.Body)
	fn.Analysis.ReturnTypeExpr = jsExtractTSReturnTypeExpr(content)

	return fn
}

func jsExtractArrowFunction(node *sitter.Node, source []byte, name string, isExported bool) jsFuncInfo {
	fn := jsFuncInfo{
		Name:       name,
		IsArrow:    true,
		IsExported: isExported,
	}

	if paramsNode := node.ChildByFieldName("parameters"); paramsNode != nil {
		fn.Params = jsParseParams(paramsNode, source)
	}

	content := node.Content(source)
	fn.IsAsync = strings.HasPrefix(content, "async ")

	bodyNode := node.ChildByFieldName("body")
	fn.Body = jsExtractBody(bodyNode, source)
	fn.Analysis = analyzeJSBody(fn.Body)
	fn.Analysis.ReturnTypeExpr = jsExtractTSReturnTypeExpr(content)

	return fn
}

func jsExtractClass(node *sitter.Node, source []byte) jsClassInfo {
	cls := jsClassInfo{}

	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		cls.Name = nameNode.Content(source)
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return cls
	}

	n := int(bodyNode.NamedChildCount())
	for i := 0; i < n; i++ {
		methodNode := bodyNode.NamedChild(i)
		if methodNode == nil || methodNode.Type() != "method_definition" {
			continue
		}

		methodNameNode := methodNode.ChildByFieldName("name")
		if methodNameNode == nil {
			continue
		}
		methodName := methodNameNode.Content(source)
		if methodName == "constructor" {
			if paramsNode := methodNode.ChildByFieldName("parameters"); paramsNode != nil {
				cls.ConstructorParams = jsParseParams(paramsNode, source)
			}
			continue
		}
		if isTestHelper(methodName) || isJSKeyword(methodName) {
			continue
		}

		method := jsFuncInfo{
			Name:      methodName,
			IsMethod:  true,
			IsPrivate: jsMethodDefinitionIsPrivate(methodNode, source, methodName),
			IsStatic:  jsMethodDefinitionIsStatic(methodNode, source, methodName),
			ClassName: cls.Name,
		}

		if paramsNode := methodNode.ChildByFieldName("parameters"); paramsNode != nil {
			method.Params = jsParseParams(paramsNode, source)
		}

		content := methodNode.Content(source)
		method.IsAsync = jsMethodDefinitionIsAsync(content, methodName)

		bodyNode := methodNode.ChildByFieldName("body")
		method.Body = jsExtractBody(bodyNode, source)
		method.Analysis = analyzeJSBody(method.Body)
		method.Analysis.ReturnTypeExpr = jsExtractTSReturnTypeExpr(content)
		method.Analysis.IsGetter = jsMethodDefinitionIsGetter(methodNode, source, methodName)

		cls.Methods = append(cls.Methods, method)
	}

	cls.PrivateEntries = jsPrivateEntryCandidates(cls)
	return cls
}

func jsPrivateEntryCandidates(cls jsClassInfo) map[string][]string {
	entries := map[string][]string{}
	for _, privateMethod := range cls.Methods {
		if !privateMethod.IsPrivate && !strings.HasPrefix(privateMethod.Name, "#") {
			continue
		}
		for _, method := range cls.Methods {
			if method.IsPrivate || strings.HasPrefix(method.Name, "#") {
				continue
			}
			if strings.Contains(method.Body, privateMethod.Name) {
				entries[privateMethod.Name] = append(entries[privateMethod.Name], cls.Name+"."+method.Name)
			}
		}
	}
	if len(entries) == 0 {
		return nil
	}
	return entries
}

func jsMethodDefinitionIsPrivate(node *sitter.Node, source []byte, methodName string) bool {
	if strings.HasPrefix(methodName, "#") {
		return true
	}
	prefix := jsMethodDefinitionPrefix(node, source, methodName)
	return strings.Contains(prefix, "private ") || strings.Contains(prefix, "protected ")
}

func jsMethodDefinitionIsStatic(node *sitter.Node, source []byte, methodName string) bool {
	prefix := jsMethodDefinitionPrefix(node, source, methodName)
	return strings.Contains(prefix, "static")
}

func jsMethodDefinitionIsAsync(content string, methodName string) bool {
	idx := strings.Index(content, methodName)
	if idx < 0 {
		return strings.HasPrefix(strings.TrimSpace(content), "async ")
	}
	prefix := content[:idx]
	return strings.Contains(prefix, "async")
}

func jsMethodDefinitionIsGetter(node *sitter.Node, source []byte, methodName string) bool {
	prefix := jsMethodDefinitionPrefix(node, source, methodName)
	return strings.Contains(prefix, "get ")
}

func jsMethodDefinitionPrefix(node *sitter.Node, source []byte, methodName string) string {
	if node == nil {
		return ""
	}
	content := node.Content(source)
	idx := strings.Index(content, methodName)
	if idx < 0 {
		return content
	}
	return content[:idx]
}

// jsExtractBody 提取函数体文本
// statement_block → 去掉花括号返回内部文本
// 表达式体（箭头函数）→ 补上 return 前缀
func jsExtractBody(bodyNode *sitter.Node, source []byte) string {
	if bodyNode == nil {
		return ""
	}
	if bodyNode.Type() == "statement_block" {
		content := bodyNode.Content(source)
		content = strings.TrimPrefix(content, "{")
		content = strings.TrimSuffix(content, "}")
		return strings.TrimSpace(content)
	}
	// 箭头函数表达式体: (a, b) => a * b
	return "return " + strings.TrimSpace(bodyNode.Content(source))
}

func jsParseParams(node *sitter.Node, source []byte) []jsParamInfo {
	var params []jsParamInfo
	n := int(node.NamedChildCount())
	for i := 0; i < n; i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "identifier":
			params = append(params, jsParamInfo{Name: child.Content(source), TypeExpr: jsParamTypeExprFromNode(child, source)})

		case "assignment_pattern":
			// param = defaultValue
			leftNode := child.ChildByFieldName("left")
			if leftNode != nil {
				name := jsParamNameFromNode(leftNode, source)
				if name != "" {
					params = append(params, jsParamInfo{Name: name, TypeExpr: jsParamTypeExprFromNode(child, source), HasDefault: true})
				}
			}

		case "rest_parameter", "rest_pattern":
			// ...args — 没有统一 name 字段，直接找 identifier 子节点
			nameNode := jsFindIdentifierChild(child, source)
			if nameNode != "" {
				params = append(params, jsParamInfo{Name: nameNode, TypeExpr: jsParamTypeExprFromNode(child, source), IsRest: true})
			}

		// TypeScript 专用参数类型
		case "required_parameter":
			patternNode := child.ChildByFieldName("pattern")
			if patternNode != nil {
				name := jsParamNameFromNode(patternNode, source)
				if name != "" {
					params = append(params, jsParamInfo{Name: name, TypeExpr: jsParamTypeExprFromNode(child, source)})
				}
			}

		case "optional_parameter":
			patternNode := child.ChildByFieldName("pattern")
			if patternNode != nil {
				name := jsParamNameFromNode(patternNode, source)
				if name != "" {
					params = append(params, jsParamInfo{Name: name, TypeExpr: jsParamTypeExprFromNode(child, source), HasDefault: true})
				}
			}
		}
	}
	return params
}

func jsParamTypeExprFromNode(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	return jsParamTypeExprFromContent(node.Content(source))
}

func jsParamTypeExprFromContent(content string) string {
	colon := strings.Index(content, ":")
	if colon < 0 {
		return ""
	}
	typeExpr := strings.TrimSpace(content[colon+1:])
	if eq := strings.Index(typeExpr, "="); eq >= 0 {
		typeExpr = strings.TrimSpace(typeExpr[:eq])
	}
	typeExpr = strings.TrimSuffix(typeExpr, ",")
	return strings.TrimSpace(typeExpr)
}

func jsParamNameFromNode(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	if node.Type() == "identifier" {
		return node.Content(source)
	}
	// 解构模式: { a, b } 或 [a, b] → 取个可用名字
	content := node.Content(source)
	content = strings.Trim(content, "{}[]")
	return strings.TrimSpace(content)
}

// jsFindIdentifierChild 在节点的命名子节点中找第一个 identifier
func jsFindIdentifierChild(node *sitter.Node, source []byte) string {
	n := int(node.NamedChildCount())
	for i := 0; i < n; i++ {
		child := node.NamedChild(i)
		if child != nil && child.Type() == "identifier" {
			return child.Content(source)
		}
	}
	return ""
}

func jsAttachTSTypeDeclsToFunc(fn *jsFuncInfo, decls map[string]string) {
	if len(decls) == 0 {
		return
	}
	fn.Analysis.TSTypeDecls = decls
}

func jsAttachTSTypeDeclsToClass(cls *jsClassInfo, decls map[string]string) {
	if len(decls) == 0 {
		return
	}
	for i := range cls.Methods {
		cls.Methods[i].Analysis.TSTypeDecls = decls
	}
}

var (
	jsTSInterfaceDeclRe = regexp.MustCompile(`(?m)(?:^|\s)(?:export\s+)?interface\s+([A-Za-z_$][A-Za-z0-9_$]*)(?:\s*<([^>{}]*)>)?[^{]*\{`)
	jsTSTypeAliasDeclRe = regexp.MustCompile(`(?m)(?:^|\s)(?:export\s+)?type\s+([A-Za-z_$][A-Za-z0-9_$]*)(?:\s*<([^>=]*)>)?\s*=`)
)

func jsExtractTSTypeDecls(source string) map[string]string {
	decls := map[string]string{}
	for _, match := range jsTSInterfaceDeclRe.FindAllStringSubmatchIndex(source, -1) {
		if len(match) < 4 {
			continue
		}
		name := source[match[2]:match[3]]
		params := ""
		if len(match) >= 6 && match[4] >= 0 && match[5] >= 0 {
			params = source[match[4]:match[5]]
		}
		open := match[1] - 1
		if typeExpr := jsExtractBracedTypeExpr(source, open); typeExpr != "" {
			if parents := jsExtractTSInterfaceExtends(source[match[0]:match[1]]); len(parents) > 0 {
				typeExpr = strings.Join(parents, " & ") + " & " + typeExpr
			}
			decls[jsTSTypeDeclKey(name, params)] = typeExpr
		}
	}
	for _, match := range jsTSTypeAliasDeclRe.FindAllStringSubmatchIndex(source, -1) {
		if len(match) < 4 {
			continue
		}
		name := source[match[2]:match[3]]
		params := ""
		if len(match) >= 6 && match[4] >= 0 && match[5] >= 0 {
			params = source[match[4]:match[5]]
		}
		if typeExpr := jsExtractTSTypeAliasExpr(source, match[1]); typeExpr != "" {
			decls[jsTSTypeDeclKey(name, params)] = typeExpr
		}
	}
	if len(decls) == 0 {
		return nil
	}
	return decls
}

func jsExtractTSInterfaceExtends(header string) []string {
	idx := strings.Index(header, "extends")
	if idx < 0 {
		return nil
	}
	part := header[idx+len("extends"):]
	if brace := strings.Index(part, "{"); brace >= 0 {
		part = part[:brace]
	}
	parents := jsSplitTopLevelGenericArgs(part)
	result := make([]string, 0, len(parents))
	for _, parent := range parents {
		parent = strings.TrimSpace(parent)
		if parent != "" {
			result = append(result, parent)
		}
	}
	return result
}

func jsTSTypeDeclKey(name, params string) string {
	params = strings.TrimSpace(params)
	if params == "" {
		return name
	}
	parts := jsSplitTopLevelGenericArgs(params)
	if len(parts) == 0 {
		return name
	}
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return name + "<" + strings.Join(parts, ",") + ">"
}

func jsExtractBracedTypeExpr(source string, open int) string {
	if open < 0 || open >= len(source) || source[open] != '{' {
		return ""
	}
	depth := 0
	for i := open; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(source[open : i+1])
			}
		}
	}
	return ""
}

func jsExtractTSTypeAliasExpr(source string, start int) string {
	if start < 0 || start >= len(source) {
		return ""
	}
	for start < len(source) && (source[start] == ' ' || source[start] == '\t' || source[start] == '\n' || source[start] == '\r') {
		start++
	}
	if start >= len(source) {
		return ""
	}
	if source[start] == '{' {
		return jsExtractBracedTypeExpr(source, start)
	}

	angleDepth, braceDepth, bracketDepth, parenDepth := 0, 0, 0, 0
	for i := start; i < len(source); i++ {
		ch := source[i]
		switch ch {
		case '<':
			angleDepth++
		case '>':
			if angleDepth > 0 {
				angleDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case ';', '\n', '\r':
			if angleDepth == 0 && braceDepth == 0 && bracketDepth == 0 && parenDepth == 0 {
				return strings.TrimSpace(source[start:i])
			}
		}
	}
	return strings.TrimSpace(source[start:])
}

func jsExtractTSReturnTypeExpr(content string) string {
	content = strings.TrimSpace(content)
	closeParen := jsFindFirstMatchingParen(content)
	if closeParen < 0 || closeParen+1 >= len(content) {
		return ""
	}
	rest := strings.TrimSpace(content[closeParen+1:])
	if !strings.HasPrefix(rest, ":") {
		return ""
	}
	rest = strings.TrimSpace(strings.TrimPrefix(rest, ":"))
	if rest == "" {
		return ""
	}

	started := false
	startedWithBrace := false
	angleDepth, braceDepth, bracketDepth, parenDepth := 0, 0, 0, 0
	for i := 0; i < len(rest); i++ {
		ch := rest[i]
		if !started {
			if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
				continue
			}
			started = true
			startedWithBrace = ch == '{'
		}
		if angleDepth == 0 && braceDepth == 0 && bracketDepth == 0 && parenDepth == 0 {
			if i+1 < len(rest) && rest[i:i+2] == "=>" {
				return strings.TrimSpace(rest[:i])
			}
			if ch == '{' && !startedWithBrace {
				return strings.TrimSpace(rest[:i])
			}
		}
		switch ch {
		case '<':
			angleDepth++
		case '>':
			if angleDepth > 0 {
				angleDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
				if startedWithBrace && angleDepth == 0 && braceDepth == 0 && bracketDepth == 0 && parenDepth == 0 {
					return strings.TrimSpace(rest[:i+1])
				}
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		}
	}
	return ""
}

func jsFindFirstMatchingParen(content string) int {
	open := strings.Index(content, "(")
	if open < 0 {
		return -1
	}
	depth := 0
	for i := open; i < len(content); i++ {
		switch content[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// ============================================================
// Python tree-sitter parser
// ============================================================

// parsePyWithTreeSitter 用 tree-sitter 解析 Python 源码，返回函数和类
func parsePyWithTreeSitter(source []byte) (funcs []pyFuncInfo, classes []pyClassInfo) {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree := parser.Parse(nil, source)
	defer tree.Close()

	root := tree.RootNode()

	n := int(root.NamedChildCount())
	for i := 0; i < n; i++ {
		child := root.NamedChild(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "function_definition":
			fn := pyExtractFunction(child, source, false, false, "")
			if pyShouldKeep(fn.Name) {
				funcs = append(funcs, fn)
			}
		case "decorated_definition":
			fn, cls := pyHandleDecorated(child, source, "")
			if fn != nil && pyShouldKeep(fn.Name) {
				funcs = append(funcs, *fn)
			} else if cls != nil && cls.Name != "" {
				classes = append(classes, *cls)
			}
		case "class_definition":
			cls := pyExtractClass(child, source)
			if cls.Name != "" {
				classes = append(classes, cls)
			}
		}
	}

	return funcs, classes
}

func pyShouldKeep(name string) bool {
	return name != "" && !isPyDunder(name) && !isPyTestHelper(name)
}

func pyHandleDecorated(node *sitter.Node, source []byte, className string) (*pyFuncInfo, *pyClassInfo) {
	isStatic := false
	isMethod := className != ""

	n := int(node.NamedChildCount())
	for i := 0; i < n; i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		if child.Type() == "decorator" {
			content := child.Content(source)
			if strings.Contains(content, "staticmethod") {
				isStatic = true
			}
		}
		if child.Type() == "function_definition" {
			fn := pyExtractFunction(child, source, isMethod, isStatic, className)
			return &fn, nil
		}
		if child.Type() == "class_definition" {
			cls := pyExtractClass(child, source)
			return nil, &cls
		}
	}
	return nil, nil
}

func pyExtractFunction(node *sitter.Node, source []byte, isMethod bool, isStatic bool, className string) pyFuncInfo {
	fn := pyFuncInfo{
		IsMethod:  isMethod,
		IsStatic:  isStatic,
		ClassName: className,
	}

	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		fn.Name = nameNode.Content(source)
	}
	if paramsNode := node.ChildByFieldName("parameters"); paramsNode != nil {
		fn.Params = pyParseParams(paramsNode, source, isMethod, isStatic)
	}

	content := node.Content(source)
	fn.IsAsync = strings.HasPrefix(content, "async ")

	if bodyNode := node.ChildByFieldName("body"); bodyNode != nil {
		fn.Body = bodyNode.Content(source)
	}

	fn.Analysis = analyzePyBody(fn.Body)

	return fn
}

func pyExtractClass(node *sitter.Node, source []byte) pyClassInfo {
	cls := pyClassInfo{}

	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		cls.Name = nameNode.Content(source)
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return cls
	}

	n := int(bodyNode.NamedChildCount())
	for i := 0; i < n; i++ {
		child := bodyNode.NamedChild(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "function_definition":
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}
			methodName := nameNode.Content(source)
			if !pyShouldKeep(methodName) && methodName != "__init__" {
				continue
			}
			if methodName == "__init__" {
				// __init__ 不生成测试，但保留在类信息中
				continue
			}
			fn := pyExtractFunction(child, source, true, false, cls.Name)
			cls.Methods = append(cls.Methods, fn)

		case "decorated_definition":
			fn, _ := pyHandleDecorated(child, source, cls.Name)
			if fn != nil && pyShouldKeep(fn.Name) {
				cls.Methods = append(cls.Methods, *fn)
			}
		}
	}

	return cls
}

func pyParseParams(node *sitter.Node, source []byte, isMethod bool, isStatic bool) []pyParamInfo {
	var params []pyParamInfo

	n := int(node.NamedChildCount())
	for i := 0; i < n; i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "identifier":
			params = append(params, pyParamInfo{Name: child.Content(source)})

		case "default_parameter":
			nameNode := pyFindIdentifier(child, source)
			if nameNode != "" {
				params = append(params, pyParamInfo{Name: nameNode, HasDefault: true})
			}

		case "typed_parameter":
			nameNode := pyFindIdentifier(child, source)
			if nameNode != "" {
				params = append(params, pyParamInfo{Name: nameNode})
			}

		case "typed_default_parameter":
			nameNode := pyFindIdentifier(child, source)
			if nameNode != "" {
				params = append(params, pyParamInfo{Name: nameNode, HasDefault: true})
			}

		case "list_splat_pattern":
			// *args
			nameNode := pyFindIdentifier(child, source)
			if nameNode != "" {
				params = append(params, pyParamInfo{Name: nameNode, IsArgs: true})
			}

		case "dictionary_splat_pattern":
			// **kwargs
			nameNode := pyFindIdentifier(child, source)
			if nameNode != "" {
				params = append(params, pyParamInfo{Name: nameNode, IsKwargs: true})
			}
		}
	}

	// 实例方法去掉 self/cls；静态方法不去掉
	if isMethod && !isStatic && len(params) > 0 {
		first := params[0].Name
		if first == "self" || first == "cls" {
			params = params[1:]
		}
	}

	return params
}

// pyFindIdentifier 从节点中提取标识符名称
// 先尝试 name 字段，再退回到第一个 identifier 子节点
func pyFindIdentifier(node *sitter.Node, source []byte) string {
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return nameNode.Content(source)
	}
	n := int(node.NamedChildCount())
	for i := 0; i < n; i++ {
		child := node.NamedChild(i)
		if child != nil && child.Type() == "identifier" {
			return child.Content(source)
		}
	}
	return ""
}
