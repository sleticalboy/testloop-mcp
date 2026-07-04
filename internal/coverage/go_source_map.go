package coverage

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/binlee/testloop-mcp/types"
)

type goFuncRange struct {
	Name      string
	Kind      string
	StartLine int
	EndLine   int
	Params    []string
}

func mapGoFunctionsByFile(files []types.CoverageFile) map[string][]goFuncRange {
	result := make(map[string][]goFuncRange)
	for _, file := range files {
		sourcePath := resolveGoSourcePath(file.Path)
		if sourcePath == "" {
			continue
		}
		ranges := parseGoFunctionRanges(sourcePath)
		if len(ranges) > 0 {
			result[file.Path] = ranges
		}
	}
	return result
}

func resolveGoSourcePath(path string) string {
	if fileExists(path) {
		return path
	}
	clean := filepath.Clean(path)
	if fileExists(clean) {
		return clean
	}

	parts := strings.Split(filepath.ToSlash(clean), "/")
	for i := range parts {
		candidate := filepath.FromSlash(strings.Join(parts[i:], "/"))
		if fileExists(candidate) {
			return candidate
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func parseGoFunctionRanges(path string) []goFuncRange {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, path, nil, 0)
	if err != nil {
		return nil
	}

	var ranges []goFuncRange
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		start := fs.Position(fn.Pos()).Line
		end := fs.Position(fn.End()).Line
		ranges = append(ranges, goFuncRange{
			Name:      goFunctionName(fn),
			Kind:      goFunctionKind(fn),
			StartLine: start,
			EndLine:   end,
			Params:    goParamNames(fn.Type.Params),
		})
	}
	return ranges
}

func goFunctionName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	recv := goReceiverName(fn.Recv.List[0].Type)
	if recv == "" {
		return fn.Name.Name
	}
	return recv + "." + fn.Name.Name
}

func goFunctionKind(fn *ast.FuncDecl) string {
	if fn.Recv == nil {
		return "function"
	}
	return "method"
}

func goReceiverName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return goReceiverName(t.X)
	case *ast.IndexExpr:
		return goReceiverName(t.X)
	case *ast.IndexListExpr:
		return goReceiverName(t.X)
	default:
		return ""
	}
}

func goParamNames(fields *ast.FieldList) []string {
	if fields == nil {
		return nil
	}
	var params []string
	for _, field := range fields.List {
		if len(field.Names) == 0 {
			params = append(params, "arg")
			continue
		}
		for _, name := range field.Names {
			params = append(params, name.Name)
		}
	}
	return params
}

func findGoFunctionForBlock(ranges []goFuncRange, block types.CoverageBlock) *goFuncRange {
	for i := range ranges {
		fn := &ranges[i]
		if block.StartLine >= fn.StartLine && block.StartLine <= fn.EndLine {
			return fn
		}
		if block.StartLine <= fn.EndLine && block.EndLine >= fn.StartLine {
			return fn
		}
	}
	return nil
}
