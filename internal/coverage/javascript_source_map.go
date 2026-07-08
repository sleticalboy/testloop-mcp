package coverage

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

var (
	jsFunctionRe   = regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\(([^)]*)\)`)
	jsVariableFnRe = regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*(?:async\s*)?(?:function\s*)?(?:\(([^)]*)\)|([A-Za-z_$][A-Za-z0-9_$]*))\s*(?:=>|\{)`)
)

func parseJavaScriptFunctionRangesWithTreeSitter(path string) []sourceRange {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	parser := sitter.NewParser()
	parser.SetLanguage(javascriptLanguageForPath(path))
	tree := parser.Parse(nil, source)
	defer tree.Close()

	lines := strings.Split(string(source), "\n")
	var ranges []sourceRange
	walkJavaScriptSource(tree.RootNode(), source, lines, "", &ranges)
	return ranges
}

func javascriptLanguageForPath(path string) *sitter.Language {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ts":
		return typescript.GetLanguage()
	case ".tsx", ".jsx":
		return tsx.GetLanguage()
	default:
		return javascript.GetLanguage()
	}
}

func walkJavaScriptSource(node *sitter.Node, source []byte, lines []string, className string, ranges *[]sourceRange) {
	if node == nil {
		return
	}
	switch node.Type() {
	case "export_statement":
		for i := 0; i < int(node.NamedChildCount()); i++ {
			walkJavaScriptSource(node.NamedChild(i), source, lines, className, ranges)
		}
		return
	case "function_declaration":
		if r := javaScriptFunctionSourceRange(node, source, lines, "", className); r != nil {
			*ranges = append(*ranges, *r)
		}
		return
	case "lexical_declaration", "variable_declaration":
		walkJavaScriptVariableDeclaration(node, source, lines, className, ranges)
		return
	case "class_declaration":
		nextClass := javaScriptNodeName(node, source)
		body := node.ChildByFieldName("body")
		if body == nil {
			return
		}
		for i := 0; i < int(body.NamedChildCount()); i++ {
			walkJavaScriptSource(body.NamedChild(i), source, lines, nextClass, ranges)
		}
		return
	case "method_definition":
		if r := javaScriptMethodSourceRange(node, source, lines, className); r != nil {
			*ranges = append(*ranges, *r)
		}
		return
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		walkJavaScriptSource(node.NamedChild(i), source, lines, className, ranges)
	}
}

func walkJavaScriptVariableDeclaration(node *sitter.Node, source []byte, lines []string, className string, ranges *[]sourceRange) {
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child == nil || child.Type() != "variable_declarator" {
			continue
		}
		nameNode := child.ChildByFieldName("name")
		valueNode := child.ChildByFieldName("value")
		if nameNode == nil || valueNode == nil {
			continue
		}
		switch valueNode.Type() {
		case "arrow_function", "function_expression":
			if r := javaScriptFunctionSourceRange(valueNode, source, lines, nameNode.Content(source), className); r != nil {
				*ranges = append(*ranges, *r)
			}
		}
	}
}

func javaScriptFunctionSourceRange(node *sitter.Node, source []byte, lines []string, fallbackName string, className string) *sourceRange {
	name := fallbackName
	if name == "" {
		name = javaScriptNodeName(node, source)
	}
	if name == "" || isJavaScriptTestHelper(name) {
		return nil
	}
	kind := "function"
	if className != "" {
		name = className + "." + name
		kind = "method"
	}
	start := int(node.StartPoint().Row) + 1
	end := int(node.EndPoint().Row) + 1
	return &sourceRange{
		Name:      name,
		Kind:      kind,
		StartLine: start,
		EndLine:   end,
		Params:    javaScriptFunctionParamNames(node, source),
		Lines:     rangeSourceLines(lines, start, end),
	}
}

func javaScriptMethodSourceRange(node *sitter.Node, source []byte, lines []string, className string) *sourceRange {
	name := javaScriptNodeName(node, source)
	if name == "" || name == "constructor" || isJavaScriptTestHelper(name) {
		return nil
	}
	if className != "" {
		name = className + "." + name
	}
	start := int(node.StartPoint().Row) + 1
	end := int(node.EndPoint().Row) + 1
	return &sourceRange{
		Name:      name,
		Kind:      "method",
		StartLine: start,
		EndLine:   end,
		Params:    javaScriptFunctionParamNames(node, source),
		Lines:     rangeSourceLines(lines, start, end),
	}
}

func javaScriptNodeName(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return ""
	}
	return nameNode.Content(source)
}

func javaScriptFunctionParamNames(node *sitter.Node, source []byte) []string {
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode == nil {
		return nil
	}
	var params []string
	for i := 0; i < int(paramsNode.NamedChildCount()); i++ {
		child := paramsNode.NamedChild(i)
		if child == nil {
			continue
		}
		name := javaScriptParamName(child, source)
		if name != "" {
			params = append(params, name)
		}
	}
	return params
}

func javaScriptParamName(node *sitter.Node, source []byte) string {
	switch node.Type() {
	case "identifier":
		return node.Content(source)
	case "assignment_pattern", "required_parameter", "optional_parameter":
		if leftNode := node.ChildByFieldName("left"); leftNode != nil {
			return javaScriptParamName(leftNode, source)
		}
		if patternNode := node.ChildByFieldName("pattern"); patternNode != nil {
			return javaScriptParamName(patternNode, source)
		}
	case "rest_pattern", "rest_parameter":
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(i)
			if child != nil && child.Type() == "identifier" {
				return child.Content(source)
			}
		}
	}
	return ""
}

func parseJavaScriptFunctionRanges(path string) []sourceRange {
	lines, ok := readSourceLines(path)
	if !ok {
		return nil
	}
	var ranges []sourceRange
	classStack := []javaScriptIndentClass{}
	for i, line := range lines {
		indent := leadingSpaces(line)
		trimmed := strings.TrimSpace(line)
		for len(classStack) > 0 && indent <= classStack[len(classStack)-1].Indent && trimmed != "" && strings.HasPrefix(trimmed, "}") {
			classStack = classStack[:len(classStack)-1]
		}
		if strings.HasPrefix(trimmed, "class ") {
			name := javaScriptClassName(trimmed)
			if name != "" {
				classStack = append(classStack, javaScriptIndentClass{Name: name, Indent: indent})
			}
			continue
		}
		name, params := javaScriptFallbackFunction(line)
		if name == "" || isJavaScriptTestHelper(name) {
			continue
		}
		kind := "function"
		if len(classStack) > 0 && indent > classStack[len(classStack)-1].Indent {
			name = classStack[len(classStack)-1].Name + "." + name
			kind = "method"
		}
		start := i + 1
		end := findBraceRangeEnd(lines, i)
		ranges = append(ranges, sourceRange{
			Name:      name,
			Kind:      kind,
			StartLine: start,
			EndLine:   end,
			Params:    params,
			Lines:     rangeSourceLines(lines, start, end),
		})
	}
	return ranges
}

type javaScriptIndentClass struct {
	Name   string
	Indent int
}

func javaScriptFallbackFunction(line string) (string, []string) {
	if matches := jsFunctionRe.FindStringSubmatch(line); len(matches) == 3 {
		return matches[1], parseParamNames(matches[2])
	}
	if matches := jsVariableFnRe.FindStringSubmatch(line); len(matches) == 4 {
		params := matches[2]
		if params == "" {
			params = matches[3]
		}
		return matches[1], parseParamNames(params)
	}
	return "", nil
}

func javaScriptClassName(line string) string {
	line = strings.TrimPrefix(strings.TrimSpace(line), "class ")
	for i, ch := range line {
		if ch == '{' || ch == ' ' || ch == '\t' {
			return line[:i]
		}
	}
	return line
}

func isJavaScriptTestHelper(name string) bool {
	switch name {
	case "describe", "it", "test", "beforeEach", "afterEach", "beforeAll", "afterAll", "expect":
		return true
	default:
		return strings.HasPrefix(name, "test") || strings.HasPrefix(name, "mock")
	}
}
