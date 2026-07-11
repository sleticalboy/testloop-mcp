package coverage

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
)

type goFuncRange struct {
	Name     string
	Kind     string
	Params   []string
	Branches []goBranchRange
	Returns  []goReturnRange

	StartLine int
	EndLine   int
	Lines     []string
}

type goBranchRange struct {
	Kind      string
	Condition string
	StartLine int
	EndLine   int
}

type goReturnRange struct {
	StartLine int
	EndLine   int
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
			Branches:  collectGoBranchRanges(fs, fn),
			Returns:   collectGoReturnRanges(fs, fn),
		})
	}
	return ranges
}

func collectGoBranchRanges(fs *token.FileSet, fn *ast.FuncDecl) []goBranchRange {
	var branches []goBranchRange
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch stmt := node.(type) {
		case *ast.IfStmt:
			branches = append(branches, goBranchRange{
				Kind:      "if",
				Condition: goExprString(fs, stmt.Cond),
				StartLine: fs.Position(stmt.Pos()).Line,
				EndLine:   fs.Position(stmt.End()).Line,
			})
		case *ast.SwitchStmt:
			branches = append(branches, goBranchRange{
				Kind:      "switch",
				StartLine: fs.Position(stmt.Pos()).Line,
				EndLine:   fs.Position(stmt.End()).Line,
			})
		case *ast.TypeSwitchStmt:
			branches = append(branches, goBranchRange{
				Kind:      "switch",
				StartLine: fs.Position(stmt.Pos()).Line,
				EndLine:   fs.Position(stmt.End()).Line,
			})
		case *ast.SelectStmt:
			branches = append(branches, goBranchRange{
				Kind:      "select",
				StartLine: fs.Position(stmt.Pos()).Line,
				EndLine:   fs.Position(stmt.End()).Line,
			})
		}
		return true
	})
	return branches
}

func collectGoReturnRanges(fs *token.FileSet, fn *ast.FuncDecl) []goReturnRange {
	var returns []goReturnRange
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		stmt, ok := node.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		returns = append(returns, goReturnRange{
			StartLine: fs.Position(stmt.Pos()).Line,
			EndLine:   fs.Position(stmt.End()).Line,
		})
		return true
	})
	return returns
}

func goExprString(fs *token.FileSet, expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fs, expr); err != nil {
		return ""
	}
	return strings.TrimSpace(buf.String())
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
	if branch := findGoBranchForBlock(fn.Branches, block); branch != nil {
		switch branch.Kind {
		case "if":
			condition := branch.Condition
			if condition == "" {
				condition = "条件表达式"
			}
			return "branch", []string{"未覆盖 if 分支: " + condition}, suggestedGoBranchInputs(fn.Params, condition)
		case "select":
			return "branch", []string{"未覆盖 select/case 分支"}, suggestedGoInputs(fn.Params)
		default:
			return "branch", []string{"未覆盖 switch/case 分支"}, suggestedGoInputs(fn.Params)
		}
	}
	if findGoReturnForBlock(fn.Returns, block) != nil {
		lines := goBlockSourceLines(fn, block)
		trimmed := strings.TrimSpace(strings.Join(lines, "\n"))
		if strings.Contains(trimmed, "return nil") || strings.Contains(trimmed, "error") || strings.Contains(trimmed, "err") {
			return "error_path", []string{"未覆盖错误或空值返回路径"}, suggestedGoInputs(fn.Params)
		}
		return "return_path", []string{"未覆盖返回路径"}, suggestedGoInputs(fn.Params)
	}
	lines := goBlockSourceLines(fn, block)
	joined := strings.Join(lines, "\n")
	trimmed := strings.TrimSpace(joined)
	switch {
	case strings.Contains(trimmed, "return nil") || strings.Contains(trimmed, "error") || strings.Contains(trimmed, "err"):
		return "error_path", []string{"未覆盖错误或空值返回路径"}, suggestedGoInputs(fn.Params)
	case strings.Contains(trimmed, "return"):
		return "return_path", []string{"未覆盖返回路径"}, suggestedGoInputs(fn.Params)
	default:
		return "statement", []string{"未覆盖普通语句块"}, suggestedGoInputs(fn.Params)
	}
}

func findGoBranchForBlock(branches []goBranchRange, block types.CoverageBlock) *goBranchRange {
	var best *goBranchRange
	for i := range branches {
		branch := &branches[i]
		if !lineRangesOverlap(block.StartLine, block.EndLine, branch.StartLine, branch.EndLine) {
			continue
		}
		if best == nil || lineSpan(branch.StartLine, branch.EndLine) < lineSpan(best.StartLine, best.EndLine) {
			best = branch
		}
	}
	return best
}

func findGoReturnForBlock(returns []goReturnRange, block types.CoverageBlock) *goReturnRange {
	for i := range returns {
		ret := &returns[i]
		if lineRangesOverlap(block.StartLine, block.EndLine, ret.StartLine, ret.EndLine) {
			return ret
		}
	}
	return nil
}

func lineRangesOverlap(startA int, endA int, startB int, endB int) bool {
	if endA < startA {
		endA = startA
	}
	if endB < startB {
		endB = startB
	}
	return startA <= endB && startB <= endA
}

func lineSpan(start int, end int) int {
	if end < start {
		return 0
	}
	return end - start
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

func suggestedGoBranchInputs(params []string, condition string) []string {
	inputs := suggestedGoInputs(params)
	if condition != "" && condition != "条件表达式" {
		inputs = append([]string{"构造满足条件 `" + condition + "` 的输入"}, inputs...)
	}
	return inputs
}
