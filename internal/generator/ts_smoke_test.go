package generator

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
)

func TestTreeSitterSmokeJS(t *testing.T) {
	source := []byte(`function add(a, b) { return a + b; }`)

	parser := sitter.NewParser()
	parser.SetLanguage(javascript.GetLanguage())
	tree := parser.Parse(nil, source)
	defer tree.Close()

	root := tree.RootNode()
	if root.Type() != "program" {
		t.Fatalf("expected root=program, got %s", root.Type())
	}

	// 找 function_declaration
	funcNode := root.NamedChild(0)
	if funcNode.Type() != "function_declaration" {
		t.Fatalf("expected function_declaration, got %s", funcNode.Type())
	}

	nameNode := funcNode.ChildByFieldName("name")
	if nameNode == nil {
		t.Fatal("name field is nil")
	}
	if nameNode.Content(source) != "add" {
		t.Errorf("expected name=add, got %s", nameNode.Content(source))
	}

	paramsNode := funcNode.ChildByFieldName("parameters")
	if paramsNode == nil {
		t.Fatal("parameters field is nil")
	}
	if paramsNode.Type() != "formal_parameters" {
		t.Errorf("expected formal_parameters, got %s", paramsNode.Type())
	}

	bodyNode := funcNode.ChildByFieldName("body")
	if bodyNode == nil {
		t.Fatal("body field is nil")
	}
	t.Logf("JS body: %q", bodyNode.Content(source))
}

func TestTreeSitterSmokePython(t *testing.T) {
	source := []byte("def add(a, b):\n    return a + b\n")

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree := parser.Parse(nil, source)
	defer tree.Close()

	root := tree.RootNode()
	if root.Type() != "module" {
		t.Fatalf("expected root=module, got %s", root.Type())
	}

	funcNode := root.NamedChild(0)
	if funcNode.Type() != "function_definition" {
		t.Fatalf("expected function_definition, got %s", funcNode.Type())
	}

	nameNode := funcNode.ChildByFieldName("name")
	if nameNode == nil {
		t.Fatal("name field is nil")
	}
	if nameNode.Content(source) != "add" {
		t.Errorf("expected name=add, got %s", nameNode.Content(source))
	}

	bodyNode := funcNode.ChildByFieldName("body")
	if bodyNode == nil {
		t.Fatal("body field is nil")
	}
	t.Logf("Python body: %q", bodyNode.Content(source))
}
