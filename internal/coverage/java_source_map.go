package coverage

import (
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

func parseJavaMethodRangesWithTreeSitter(path string) []sourceRange {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())
	tree := parser.Parse(nil, source)
	defer tree.Close()

	lines := strings.Split(string(source), "\n")
	var ranges []sourceRange
	walkJavaSource(tree.RootNode(), source, lines, nil, &ranges)
	return ranges
}

func walkJavaSource(node *sitter.Node, source []byte, lines []string, classStack []string, ranges *[]sourceRange) {
	if node == nil {
		return
	}
	switch node.Type() {
	case "class_declaration", "enum_declaration", "interface_declaration", "record_declaration":
		className := javaNodeName(node, source)
		nextStack := classStack
		if className != "" {
			nextStack = append(nextStack, className)
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			walkJavaSource(node.Child(i), source, lines, nextStack, ranges)
		}
		return
	case "method_declaration":
		if r := javaMethodSourceRange(node, source, lines, classStack, false); r != nil {
			*ranges = append(*ranges, *r)
		}
		return
	case "constructor_declaration":
		if r := javaMethodSourceRange(node, source, lines, classStack, true); r != nil {
			*ranges = append(*ranges, *r)
		}
		return
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		walkJavaSource(node.Child(i), source, lines, classStack, ranges)
	}
}

func javaMethodSourceRange(node *sitter.Node, source []byte, lines []string, classStack []string, constructor bool) *sourceRange {
	name := javaNodeName(node, source)
	if constructor && len(classStack) > 0 {
		name = classStack[len(classStack)-1]
	}
	if name == "" {
		return nil
	}
	if len(classStack) > 0 {
		name = strings.Join(append(append([]string{}, classStack...), name), ".")
	}
	start := int(node.StartPoint().Row) + 1
	end := int(node.EndPoint().Row) + 1
	return &sourceRange{
		Name:      name,
		Kind:      "method",
		StartLine: start,
		EndLine:   end,
		Params:    javaMethodParamNames(node, source),
		Lines:     rangeSourceLines(lines, start, end),
	}
}

func javaNodeName(node *sitter.Node, source []byte) string {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return ""
	}
	return nameNode.Content(source)
}

func javaMethodParamNames(node *sitter.Node, source []byte) []string {
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode == nil {
		return nil
	}
	var params []string
	for i := 0; i < int(paramsNode.ChildCount()); i++ {
		child := paramsNode.Child(i)
		if child.Type() != "formal_parameter" && child.Type() != "spread_parameter" {
			continue
		}
		nameNode := child.ChildByFieldName("name")
		if nameNode != nil {
			params = append(params, nameNode.Content(source))
		}
	}
	return params
}
