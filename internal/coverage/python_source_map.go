package coverage

import (
	"os"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

var pyFunctionRe = regexp.MustCompile(`^\s*(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*:`)

func parsePythonFunctionRangesWithTreeSitter(path string) []sourceRange {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree := parser.Parse(nil, source)
	defer tree.Close()

	lines := strings.Split(string(source), "\n")
	var ranges []sourceRange
	walkPythonSource(tree.RootNode(), source, lines, "", &ranges)
	return ranges
}

func walkPythonSource(node *sitter.Node, source []byte, lines []string, className string, ranges *[]sourceRange) {
	if node == nil {
		return
	}
	switch node.Type() {
	case "function_definition":
		if r := pythonFunctionSourceRange(node, source, lines, className); r != nil {
			*ranges = append(*ranges, *r)
		}
		return
	case "decorated_definition":
		walkPythonDecoratedDefinition(node, source, lines, className, ranges)
		return
	case "class_definition":
		nextClass := pythonNodeName(node, source)
		body := node.ChildByFieldName("body")
		if body == nil {
			return
		}
		for i := 0; i < int(body.NamedChildCount()); i++ {
			walkPythonSource(body.NamedChild(i), source, lines, nextClass, ranges)
		}
		return
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		walkPythonSource(node.NamedChild(i), source, lines, className, ranges)
	}
}

func walkPythonDecoratedDefinition(node *sitter.Node, source []byte, lines []string, className string, ranges *[]sourceRange) {
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "function_definition", "class_definition":
			walkPythonSource(child, source, lines, className, ranges)
			return
		}
	}
}

func pythonFunctionSourceRange(node *sitter.Node, source []byte, lines []string, className string) *sourceRange {
	name := pythonNodeName(node, source)
	if name == "" || isPythonDunder(name) {
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
		Params:    pythonFunctionParamNames(node, source, className != ""),
		Lines:     rangeSourceLines(lines, start, end),
	}
}

func pythonNodeName(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return ""
	}
	return nameNode.Content(source)
}

func pythonFunctionParamNames(node *sitter.Node, source []byte, method bool) []string {
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
		name := pythonParamName(child, source)
		if name == "" {
			continue
		}
		params = append(params, name)
	}
	if method && len(params) > 0 && (params[0] == "self" || params[0] == "cls") {
		params = params[1:]
	}
	return params
}

func pythonParamName(node *sitter.Node, source []byte) string {
	switch node.Type() {
	case "identifier":
		return node.Content(source)
	case "default_parameter", "typed_parameter", "typed_default_parameter", "list_splat_pattern", "dictionary_splat_pattern":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return nameNode.Content(source)
		}
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(i)
			if child != nil && child.Type() == "identifier" {
				return child.Content(source)
			}
		}
	}
	return ""
}

func parsePythonFunctionRanges(path string) []sourceRange {
	lines, ok := readSourceLines(path)
	if !ok {
		return nil
	}
	var ranges []sourceRange
	classStack := []pythonIndentClass{}
	for i, line := range lines {
		indent := leadingSpaces(line)
		trimmed := strings.TrimSpace(line)
		for len(classStack) > 0 && indent <= classStack[len(classStack)-1].Indent && trimmed != "" {
			classStack = classStack[:len(classStack)-1]
		}
		if strings.HasPrefix(trimmed, "class ") {
			name := pythonClassName(trimmed)
			if name != "" {
				classStack = append(classStack, pythonIndentClass{Name: name, Indent: indent})
			}
			continue
		}
		matches := pyFunctionRe.FindStringSubmatch(line)
		if len(matches) != 3 || isPythonDunder(matches[1]) {
			continue
		}
		name := matches[1]
		kind := "function"
		params := parseParamNames(matches[2])
		if len(classStack) > 0 && indent > classStack[len(classStack)-1].Indent {
			name = classStack[len(classStack)-1].Name + "." + name
			kind = "method"
			if len(params) > 0 && (params[0] == "self" || params[0] == "cls") {
				params = params[1:]
			}
		}
		start := i + 1
		end := findPythonRangeEnd(lines, i, indent)
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

type pythonIndentClass struct {
	Name   string
	Indent int
}

func pythonClassName(line string) string {
	line = strings.TrimPrefix(strings.TrimSpace(line), "class ")
	for i, ch := range line {
		if ch == '(' || ch == ':' || ch == ' ' || ch == '\t' {
			return line[:i]
		}
	}
	return line
}

func findPythonRangeEnd(lines []string, startIdx int, indent int) int {
	end := startIdx + 1
	for i := startIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if leadingSpaces(lines[i]) <= indent {
			return end
		}
		end = i + 1
	}
	return end
}

func leadingSpaces(line string) int {
	count := 0
	for _, ch := range line {
		switch ch {
		case ' ':
			count++
		case '\t':
			count += 4
		default:
			return count
		}
	}
	return count
}

func isPythonDunder(name string) bool {
	return strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__")
}
