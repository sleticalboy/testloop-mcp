package coverage

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
)

type goFuncRange struct {
	Name      string
	Kind      string
	StartLine int
	EndLine   int
	Params    []string
	Lines     []string
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
	source, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, path, source, 0)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(source), "\n")

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
			Lines:     sourceLines(lines, start, end),
		})
	}
	return ranges
}

func sourceLines(lines []string, start int, end int) []string {
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end {
		return nil
	}
	copied := make([]string, end-start+1)
	copy(copied, lines[start-1:end])
	return copied
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

func analyzeGoCoverageGap(fn *goFuncRange, block types.CoverageBlock) (string, []string, []string) {
	if fn == nil {
		return "", nil, nil
	}
	lines := goBlockSourceLines(fn, block)
	joined := strings.Join(lines, "\n")
	trimmed := strings.TrimSpace(joined)
	switch {
	case strings.Contains(trimmed, "if ") || strings.HasPrefix(trimmed, "if"):
		condition := extractGoCondition(trimmed, "if")
		return "branch", []string{"未覆盖 if 分支: " + condition}, suggestedGoBranchInputs(fn.Params, condition)
	case strings.Contains(trimmed, "switch ") || strings.HasPrefix(trimmed, "switch"):
		return "branch", []string{"未覆盖 switch/case 分支"}, suggestedGoInputs(fn.Params)
	case strings.Contains(trimmed, "return nil") || strings.Contains(trimmed, "error") || strings.Contains(trimmed, "err"):
		return "error_path", []string{"未覆盖错误或空值返回路径"}, suggestedGoInputs(fn.Params)
	case strings.Contains(trimmed, "return"):
		return "return_path", []string{"未覆盖返回路径"}, suggestedGoInputs(fn.Params)
	default:
		return "statement", []string{"未覆盖普通语句块"}, suggestedGoInputs(fn.Params)
	}
}

func goBlockSourceLines(fn *goFuncRange, block types.CoverageBlock) []string {
	start := block.StartLine - fn.StartLine
	end := block.EndLine - fn.StartLine
	if start < 0 {
		start = 0
	}
	if end >= len(fn.Lines) {
		end = len(fn.Lines) - 1
	}
	if start > end || len(fn.Lines) == 0 {
		return nil
	}
	return fn.Lines[start : end+1]
}

func extractGoCondition(line string, keyword string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, keyword)
	line = strings.TrimSpace(line)
	if idx := strings.Index(line, "{"); idx >= 0 {
		line = line[:idx]
	}
	line = strings.Trim(line, "() ")
	if line == "" {
		return "条件表达式"
	}
	return line
}

func suggestedGoBranchInputs(params []string, condition string) []string {
	inputs := suggestedGoInputs(params)
	if condition != "" && condition != "条件表达式" {
		inputs = append([]string{"构造满足条件 `" + condition + "` 的输入"}, inputs...)
	}
	return inputs
}
