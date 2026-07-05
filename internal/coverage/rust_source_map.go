package coverage

import (
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/rust"
)

func parseRustFunctionRangesWithTreeSitter(path string) []sourceRange {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	parser := sitter.NewParser()
	parser.SetLanguage(rust.GetLanguage())
	tree := parser.Parse(nil, source)
	defer tree.Close()

	lines := strings.Split(string(source), "\n")
	var ranges []sourceRange
	walkRustSource(tree.RootNode(), source, lines, "", &ranges)
	return ranges
}

func walkRustSource(node *sitter.Node, source []byte, lines []string, owner string, ranges *[]sourceRange) {
	if node == nil {
		return
	}
	switch node.Type() {
	case "function_item":
		if r := rustFunctionSourceRange(node, source, lines, owner); r != nil {
			*ranges = append(*ranges, *r)
		}
		return
	case "impl_item":
		nextOwner := rustOwnerName(node.ChildByFieldName("type"), source)
		walkRustOwnedBody(node, source, lines, nextOwner, ranges)
		return
	case "trait_item":
		nextOwner := rustOwnerName(node.ChildByFieldName("name"), source)
		walkRustOwnedBody(node, source, lines, nextOwner, ranges)
		return
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		walkRustSource(node.Child(i), source, lines, owner, ranges)
	}
}

func walkRustOwnedBody(node *sitter.Node, source []byte, lines []string, owner string, ranges *[]sourceRange) {
	body := node.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := 0; i < int(body.ChildCount()); i++ {
		walkRustSource(body.Child(i), source, lines, owner, ranges)
	}
}

func rustFunctionSourceRange(node *sitter.Node, source []byte, lines []string, owner string) *sourceRange {
	name := rustOwnerName(node.ChildByFieldName("name"), source)
	if name == "" {
		return nil
	}
	if owner != "" {
		name = owner + "." + name
	}
	start := int(node.StartPoint().Row) + 1
	end := int(node.EndPoint().Row) + 1
	return &sourceRange{
		Name:      name,
		Kind:      rustRangeKind(owner),
		StartLine: start,
		EndLine:   end,
		Params:    rustFunctionParamNames(node, source),
		Lines:     rangeSourceLines(lines, start, end),
	}
}

func rustRangeKind(owner string) string {
	if owner == "" {
		return "function"
	}
	return "method"
}

func rustOwnerName(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	return strings.TrimSpace(node.Content(source))
}

func rustFunctionParamNames(node *sitter.Node, source []byte) []string {
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode == nil {
		return nil
	}
	var params []string
	for i := 0; i < int(paramsNode.ChildCount()); i++ {
		child := paramsNode.Child(i)
		switch child.Type() {
		case "parameter":
			nameNode := child.ChildByFieldName("pattern")
			if nameNode != nil {
				params = append(params, nameNode.Content(source))
			}
		case "self_parameter":
			continue
		}
	}
	return params
}
