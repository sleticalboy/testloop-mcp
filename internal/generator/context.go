package generator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
)

// BuildGenerationContext extracts source structure for semantic test generation.
func BuildGenerationContext(srcPath string) *types.TestGenerationContext {
	return BuildGenerationContextWithOptions(srcPath, GenerateTestsOptions{})
}

func BuildGenerationContextWithOptions(srcPath string, opts GenerateTestsOptions) *types.TestGenerationContext {
	ext := strings.ToLower(filepath.Ext(srcPath))
	var ctx *types.TestGenerationContext
	switch ext {
	case ".go":
		ctx = buildGoGenerationContext(srcPath, opts)
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		ctx = buildJSGenerationContext(srcPath, ext)
	case ".py":
		ctx = buildPyGenerationContext(srcPath)
	}
	if opts.CoverageTask == nil {
		if framework := effectiveGenerationFramework(srcPath, opts); framework != "" {
			if ctx != nil {
				ctx.Framework = normalizedFrameworkForPath(srcPath, framework)
			}
		}
		return ctx
	}
	if ctx == nil {
		ctx = &types.TestGenerationContext{
			Language:   languageNameForPath(srcPath),
			Framework:  opts.CoverageTask.Framework,
			SourceFile: srcPath,
		}
	}
	ctx.CoverageTask = opts.CoverageTask
	return ctx
}

func normalizedFrameworkForPath(srcPath, framework string) string {
	if isJavaScriptPath(srcPath) {
		return normalizeJavaScriptTestFramework(framework)
	}
	return strings.TrimSpace(framework)
}

func languageNameForPath(srcPath string) string {
	switch strings.ToLower(filepath.Ext(srcPath)) {
	case ".go":
		return "go"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".py":
		return "python"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	default:
		return ""
	}
}

func buildGoGenerationContext(srcPath string, opts GenerateTestsOptions) *types.TestGenerationContext {
	fs := token.NewFileSet()
	node, err := parser.ParseFile(fs, srcPath, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	ctx := &types.TestGenerationContext{
		Language:   "go",
		Framework:  "go-test",
		SourceFile: srcPath,
		Imports:    goContextImports(node),
	}
	if opts.CoverageTask != nil && opts.CoverageTask.Framework != "" {
		ctx.Framework = opts.CoverageTask.Framework
	}

	for _, decl := range node.Decls {
		fnDecl, ok := decl.(*ast.FuncDecl)
		if !ok || strings.HasPrefix(fnDecl.Name.Name, "Test") {
			continue
		}
		fn := goFuncInfoFromDecl(fs, fnDecl)
		ctx.Targets = append(ctx.Targets, goTarget(fn, opts.CoverageTask))
	}
	if len(ctx.Targets) == 0 && len(ctx.Types) == 0 {
		return nil
	}
	return ctx
}

func goContextImports(node *ast.File) []string {
	if node == nil {
		return nil
	}
	imports := make([]string, 0, len(node.Imports))
	for _, spec := range node.Imports {
		if spec.Path != nil {
			imports = append(imports, spec.Path.Value)
		}
	}
	return imports
}

func goFuncInfoFromDecl(fs *token.FileSet, fn *ast.FuncDecl) funcInfo {
	info := funcInfo{Name: fn.Name.Name}
	if fn.Recv != nil {
		info.IsMethod = true
		for _, field := range fn.Recv.List {
			recvType := exprToString(field.Type)
			info.ReceiverType = recvType
			for _, name := range field.Names {
				info.Receiver = name.Name
			}
		}
	}
	if fn.Type.Params != nil {
		for _, p := range fn.Type.Params.List {
			typ := exprToString(p.Type)
			if ell, ok := p.Type.(*ast.Ellipsis); ok {
				typ = "[]" + exprToString(ell.Elt)
			}
			for _, name := range p.Names {
				info.Params = append(info.Params, paramInfo{Name: name.Name, Type: typ})
			}
		}
	}
	if fn.Type.Results != nil {
		for i, r := range fn.Type.Results.List {
			typ := exprToString(r.Type)
			name := "ret" + strconv.Itoa(i)
			if len(r.Names) > 0 {
				name = r.Names[0].Name
			}
			info.Returns = append(info.Returns, paramInfo{Name: name, Type: typ})
		}
	}
	info.ReturnExpr = singleReturnExpr(fn.Body)
	info.FinalReturn = finalReturnExpr(fn.Body)
	info.Boundaries = extractGoBoundaries(fs, fn.Body)
	return info
}

func goTarget(fn funcInfo, task *types.CoverageTestTask) types.TestTarget {
	kind := "function"
	className := ""
	if fn.IsMethod {
		kind = "method"
		className = strings.TrimPrefix(fn.ReceiverType, "*")
	}
	target := types.TestTarget{
		Name:              fn.Name,
		Kind:              kind,
		ClassName:         className,
		Params:            goContextParams(fn.Params),
		ReturnType:        goContextReturnType(fn.Returns),
		ReturnExpressions: goContextReturnExpressions(fn),
		BoundaryCases:     goContextBoundaryCases(fn.Boundaries),
	}
	if task != nil && goFuncMatchesTarget(fn, strings.TrimSpace(task.Target)) {
		target.PayloadNotes = goCoverageTaskFallbackNotes(fn, task)
	}
	if target.ReturnType == "" {
		target.ReturnType = "unknown"
	}
	return target
}

func goContextParams(params []paramInfo) []string {
	out := make([]string, 0, len(params))
	for _, p := range params {
		if p.Type == "" {
			out = append(out, p.Name)
			continue
		}
		out = append(out, strings.TrimSpace(p.Name+" "+p.Type))
	}
	return out
}

func goContextReturnType(returns []paramInfo) string {
	if len(returns) == 0 {
		return ""
	}
	types := make([]string, 0, len(returns))
	for _, r := range returns {
		types = append(types, r.Type)
	}
	return strings.Join(types, ", ")
}

func goContextReturnExpressions(fn funcInfo) []string {
	var expressions []string
	seen := map[string]bool{}
	for _, expr := range append([]string{fn.ReturnExpr, fn.FinalReturn}, goBoundaryReturnExpressions(fn.Boundaries)...) {
		expr = strings.TrimSpace(expr)
		if expr == "" || seen[expr] {
			continue
		}
		seen[expr] = true
		expressions = append(expressions, expr)
	}
	return expressions
}

func goBoundaryReturnExpressions(boundaries []goBoundary) []string {
	expressions := make([]string, 0, len(boundaries))
	for _, boundary := range boundaries {
		expressions = append(expressions, boundary.ReturnExpr)
	}
	return expressions
}

func goContextBoundaryCases(boundaries []goBoundary) []string {
	cases := make([]string, 0, len(boundaries))
	for _, boundary := range boundaries {
		cases = append(cases, boundary.Condition)
	}
	return cases
}

func buildJSGenerationContext(srcPath, ext string) *types.TestGenerationContext {
	source, err := os.ReadFile(srcPath)
	if err != nil {
		return nil
	}

	funcs, classes, _ := parseJSWithTreeSitter(source, ext)
	typeMocks := jsImportedTypeMocks(srcPath, string(source))
	jsAttachImportedTypeMocks(funcs, classes, typeMocks)
	ctx := &types.TestGenerationContext{
		Language:   jsLanguageName(ext),
		Framework:  "jest",
		SourceFile: srcPath,
		Imports:    extractJSImports(string(source)),
		Types:      extractJSTypes(string(source)),
	}

	for _, fn := range funcs {
		if fn.IsMethod {
			continue
		}
		ctx.Targets = append(ctx.Targets, jsTarget(fn, "function", ctx.Imports, srcPath, typeMocks))
	}
	for _, cls := range classes {
		for _, method := range cls.Methods {
			ctx.Targets = append(ctx.Targets, jsTarget(method, "method", ctx.Imports, srcPath, typeMocks))
		}
	}

	if len(ctx.Targets) == 0 && len(ctx.Types) == 0 {
		return nil
	}
	return ctx
}

func jsTarget(fn jsFuncInfo, kind string, imports []string, srcPath string, typeMocks map[string]jsImportedTypeMock) types.TestTarget {
	target := types.TestTarget{
		Name:              fn.Name,
		Kind:              kind,
		ClassName:         fn.ClassName,
		Params:            jsParamNames(fn.Params),
		Async:             fn.IsAsync,
		ReturnType:        fn.Analysis.ReturnType,
		ReturnTypeExpr:    fn.Analysis.ReturnTypeExpr,
		ReturnExpressions: fn.Analysis.Returns,
		PayloadNotes:      jsPayloadFallbackNotes(fn.Analysis, imports, srcPath, typeMocks),
		HasErrorPath:      fn.Analysis.Throws,
		BoundaryCases:     jsBoundaryLabels(fn.Analysis.Boundaries),
	}
	if target.ReturnType == "" {
		target.ReturnType = "unknown"
	}
	return target
}

func jsPayloadFallbackNotes(analysis jsFuncAnalysis, imports []string, srcPath string, typeMocks map[string]jsImportedTypeMock) []string {
	typeExpr := strings.TrimSpace(analysis.ReturnTypeExpr)
	if typeExpr == "" {
		return nil
	}
	inner := jsPayloadNoteTypeExpr(typeExpr)
	importNotes := jsPayloadImportNotes(inner, imports, srcPath, typeMocks)
	if _, ok := jsMockPayloadFromTSTypeWithDecls(typeExpr, analysis.TSTypeDecls); ok {
		return importNotes
	}

	reason := jsExplainTSPayloadFallback(inner, analysis.TSTypeDecls)
	if reason == "" {
		reason = "return annotation is outside the static payload support boundary"
	}
	notes := []string{reason + "; static payload falls back to { ok: true }"}
	notes = append(notes, importNotes...)
	return notes
}

func jsPayloadNoteTypeExpr(typeExpr string) string {
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	typeExpr = jsUnwrapTSGeneric(typeExpr, "Promise")
	typeExpr = jsUnwrapTSGeneric(typeExpr, "PromiseLike")
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	typeExpr = jsUnwrapTSUtilityWrappers(typeExpr)
	if branch, ok := jsPreferredTSTypeUnionBranch(typeExpr); ok {
		typeExpr = jsUnwrapTSUtilityWrappers(branch)
	}
	return jsNormalizeTSTypeExpr(typeExpr)
}

func jsExplainTSPayloadFallback(typeExpr string, decls map[string]string) string {
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	if typeExpr == "" {
		return ""
	}
	if strings.Contains(typeExpr, "keyof") || regexp.MustCompile(`\[[A-Za-z_$][A-Za-z0-9_$]*\]`).MatchString(typeExpr) {
		return "return annotation " + typeExpr + " uses dynamic indexed access or keyof, which static payload generation does not expand"
	}
	if jsTSIdentifierRe.MatchString(typeExpr) {
		if strings.TrimSpace(decls[typeExpr]) == "" {
			return "return annotation " + typeExpr + " is not declared in the same source file"
		}
		return "return annotation " + typeExpr + " resolves to a non-object or unsupported alias"
	}
	if name, args, ok := jsTSNamedGenericParts(typeExpr); ok {
		foundBase := false
		foundArity := false
		for declName := range decls {
			declBase, params, ok := jsTSNamedGenericParts(declName)
			if !ok || declBase != name {
				continue
			}
			foundBase = true
			if len(params) != len(args) {
				continue
			}
			foundArity = true
			if !jsTSGenericParamsAreSimple(params) {
				return "generic return annotation " + typeExpr + " uses constrained or defaulted type parameters"
			}
		}
		if !foundBase {
			return "generic return annotation " + typeExpr + " is not declared in the same source file"
		}
		if !foundArity {
			return "generic return annotation " + typeExpr + " does not match the same-file generic declaration arity"
		}
		return "generic return annotation " + typeExpr + " resolves to an unsupported payload shape"
	}
	return ""
}

type jsImportTypeHint struct {
	Name      string
	Module    string
	Namespace bool
}

func jsPayloadImportNotes(typeExpr string, imports []string, srcPath string, typeMocks map[string]jsImportedTypeMock) []string {
	typeExpr = jsNormalizeTSTypeExpr(typeExpr)
	if typeExpr == "" || len(imports) == 0 {
		return nil
	}
	hints := jsImportTypeHints(imports)
	if len(hints) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var notes []string
	for _, identifier := range jsIdentifiersInTSType(typeExpr) {
		hint, ok := hints[identifier]
		if !ok {
			continue
		}
		key := hint.Name + "\x00" + hint.Module
		if seen[key] {
			continue
		}
		seen[key] = true
		notes = append(notes, jsImportTypeHintNote(hint, srcPath, typeMocks))
	}
	return notes
}

func jsImportTypeHintNote(hint jsImportTypeHint, srcPath string, typeMocks map[string]jsImportedTypeMock) string {
	if hint.Namespace {
		if candidates := jsImportCandidateFiles(srcPath, hint.Module); len(candidates) > 0 {
			return "return annotation references namespace import " + hint.Name + " from '" + hint.Module + "'; read candidate source files: " + strings.Join(candidates, ", ")
		}
		return "return annotation references namespace import " + hint.Name + " from package '" + hint.Module + "'; static payload does not inspect package types"
	}
	if mock, ok := typeMocks[hint.Name]; ok && mock.Module == hint.Module && mock.FilePath != "" {
		return "return annotation imported type " + hint.Name + " from '" + hint.Module + "' resolved from " + jsImportedTypeMockRelPath(srcPath, mock.FilePath)
	}
	if candidates := jsImportCandidateFiles(srcPath, hint.Module); len(candidates) > 0 {
		return "return annotation references imported type " + hint.Name + " from '" + hint.Module + "'; read candidate source files: " + strings.Join(candidates, ", ")
	}
	return "return annotation references imported type " + hint.Name + " from package '" + hint.Module + "'; static payload does not inspect package types"
}

func jsImportedTypeMockRelPath(srcPath string, mockPath string) string {
	if rel, err := filepath.Rel(filepath.Dir(srcPath), mockPath); err == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(mockPath)
}

func jsImportTypeHints(imports []string) map[string]jsImportTypeHint {
	hints := make(map[string]jsImportTypeHint)
	for _, line := range imports {
		module := jsImportModule(line)
		if module == "" {
			continue
		}
		for _, hint := range jsImportTypeHintsFromLine(line, module) {
			if hint.Name == "" {
				continue
			}
			hints[hint.Name] = hint
		}
	}
	return hints
}

func jsImportTypeHintsFromLine(line, module string) []jsImportTypeHint {
	var hints []jsImportTypeHint
	if match := jsNamespaceImportRe.FindStringSubmatch(line); len(match) > 1 {
		hints = append(hints, jsImportTypeHint{Name: strings.TrimSpace(match[1]), Module: module, Namespace: true})
	}
	for _, block := range jsImportNamedBlocks(line) {
		for _, part := range strings.Split(block, ",") {
			name := jsImportedLocalName(part)
			if name != "" {
				hints = append(hints, jsImportTypeHint{Name: name, Module: module})
			}
		}
	}
	if match := jsDefaultImportRe.FindStringSubmatch(line); len(match) > 1 {
		name := strings.TrimSpace(match[1])
		if name != "" && name != "type" {
			hints = append(hints, jsImportTypeHint{Name: name, Module: module})
		}
	}
	return hints
}

func jsImportModule(line string) string {
	if match := jsImportFromModuleRe.FindStringSubmatch(line); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	if match := jsRequireModuleRe.FindStringSubmatch(line); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func jsImportNamedBlocks(line string) []string {
	matches := jsNamedImportBlockRe.FindAllStringSubmatch(line, -1)
	blocks := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			blocks = append(blocks, match[1])
		}
	}
	return blocks
}

func jsImportedLocalName(part string) string {
	part = strings.TrimSpace(part)
	part = strings.TrimPrefix(part, "type ")
	part = strings.TrimSpace(part)
	if part == "" {
		return ""
	}
	fields := strings.Fields(part)
	if len(fields) >= 3 && fields[len(fields)-2] == "as" {
		return fields[len(fields)-1]
	}
	if len(fields) > 0 {
		return fields[0]
	}
	return ""
}

func jsImportCandidateFiles(srcPath, module string) []string {
	if !strings.HasPrefix(module, ".") {
		return nil
	}
	base := filepath.Clean(module)
	exts := []string{".ts", ".tsx", ".d.ts", ".js", ".jsx", ".mjs", ".cjs"}
	candidates := make([]string, 0, len(exts)*2)
	for _, ext := range exts {
		candidates = append(candidates, filepath.ToSlash(base+ext))
	}
	for _, ext := range exts {
		candidates = append(candidates, filepath.ToSlash(filepath.Join(base, "index"+ext)))
	}
	return candidates
}

func jsIdentifiersInTSType(typeExpr string) []string {
	matches := jsTSIdentifierFindRe.FindAllString(typeExpr, -1)
	seen := make(map[string]bool)
	var identifiers []string
	for _, match := range matches {
		if jsBuiltinTSTypeIdentifier(match) || seen[match] {
			continue
		}
		seen[match] = true
		identifiers = append(identifiers, match)
	}
	return identifiers
}

func jsBuiltinTSTypeIdentifier(identifier string) bool {
	switch identifier {
	case "Array", "ReadonlyArray", "Promise", "PromiseLike", "Readonly", "Required", "Partial", "Pick", "Omit", "Record",
		"string", "number", "boolean", "bigint", "symbol", "object", "unknown", "any", "void", "never", "null", "undefined",
		"true", "false", "Date":
		return true
	default:
		return false
	}
}

func jsParamNames(params []jsParamInfo) []string {
	names := make([]string, 0, len(params))
	for _, p := range params {
		name := p.Name
		if p.IsRest {
			name = "..." + name
		}
		if p.HasDefault {
			name += "?"
		}
		names = append(names, name)
	}
	return names
}

func jsBoundaryLabels(boundaries []jsBoundary) []string {
	labels := make([]string, 0, len(boundaries))
	for _, b := range boundaries {
		labels = append(labels, b.Param+"="+b.Value)
	}
	return labels
}

func jsLanguageName(ext string) string {
	switch ext {
	case ".ts", ".tsx":
		return "typescript"
	default:
		return "javascript"
	}
}

func buildPyGenerationContext(srcPath string) *types.TestGenerationContext {
	source, err := os.ReadFile(srcPath)
	if err != nil {
		return nil
	}

	funcs, classes := parsePyWithTreeSitter(source)
	ctx := &types.TestGenerationContext{
		Language:   "python",
		Framework:  "pytest",
		SourceFile: srcPath,
		Imports:    extractPyImports(string(source)),
		Types:      extractPyTypes(string(source)),
	}

	for _, fn := range funcs {
		if fn.IsMethod {
			continue
		}
		ctx.Targets = append(ctx.Targets, pyTarget(fn, "function"))
	}
	for _, cls := range classes {
		for _, method := range cls.Methods {
			ctx.Targets = append(ctx.Targets, pyTarget(method, "method"))
		}
	}

	if len(ctx.Targets) == 0 {
		return nil
	}
	return ctx
}

func pyTarget(fn pyFuncInfo, kind string) types.TestTarget {
	target := types.TestTarget{
		Name:              fn.Name,
		Kind:              kind,
		ClassName:         fn.ClassName,
		Params:            pyParamNames(fn.Params),
		Async:             fn.IsAsync,
		ReturnType:        fn.Analysis.ReturnType,
		ReturnExpressions: fn.Analysis.Returns,
		HasErrorPath:      fn.Analysis.Raises,
		BoundaryCases:     pyBoundaryLabels(fn.Analysis.Boundaries),
	}
	if target.ReturnType == "" {
		target.ReturnType = "unknown"
	}
	return target
}

var (
	jsImportLineRe   = regexp.MustCompile(`(?m)^\s*import\s+[^;\n]+;?`)
	jsRequireLineRe  = regexp.MustCompile(`(?m)^\s*(?:const|let|var)\s+[^=\n]+\s*=\s*require\([^)]+\)\s*;?`)
	jsTypeDeclLineRe = regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:class|interface|type|enum)\s+([A-Za-z_$][\w$]*)`)
	pyImportLineRe   = regexp.MustCompile(`(?m)^\s*(?:from\s+\S+\s+import\s+.+|import\s+.+)$`)
	pyTypeDeclLineRe = regexp.MustCompile(`(?m)^\s*class\s+([A-Za-z_]\w*)`)

	jsImportFromModuleRe = regexp.MustCompile(`\sfrom\s*['"]([^'"]+)['"]`)
	jsRequireModuleRe    = regexp.MustCompile(`require\(\s*['"]([^'"]+)['"]\s*\)`)
	jsNamedImportBlockRe = regexp.MustCompile(`\{([^}]*)\}`)
	jsDefaultImportRe    = regexp.MustCompile(`^\s*import\s+(?:type\s+)?([A-Za-z_$][\w$]*)\s*(?:,|\s+from)`)
	jsNamespaceImportRe  = regexp.MustCompile(`^\s*import\s+(?:type\s+)?\*\s+as\s+([A-Za-z_$][\w$]*)`)
	jsTSIdentifierFindRe = regexp.MustCompile(`[A-Za-z_$][A-Za-z0-9_$]*`)
)

func extractJSImports(source string) []string {
	return uniqueTrimmedMatchesFromRegexes(source, []regexMatchGroup{
		{re: jsImportLineRe, group: 0},
		{re: jsRequireLineRe, group: 0},
	})
}

func extractJSTypes(source string) []string {
	return uniqueTrimmedMatches(source, jsTypeDeclLineRe, 1)
}

func extractPyImports(source string) []string {
	return uniqueTrimmedMatches(source, pyImportLineRe, 0)
}

func extractPyTypes(source string) []string {
	return uniqueTrimmedMatches(source, pyTypeDeclLineRe, 1)
}

func uniqueTrimmedMatches(source string, re *regexp.Regexp, group int) []string {
	return uniqueTrimmedMatchesFromRegexes(source, []regexMatchGroup{{re: re, group: group}})
}

type regexMatchGroup struct {
	re    *regexp.Regexp
	group int
}

func uniqueTrimmedMatchesFromRegexes(source string, regexes []regexMatchGroup) []string {
	seen := make(map[string]bool)
	var values []string
	for _, matcher := range regexes {
		values = append(values, uniqueTrimmedMatchesWithSeen(source, matcher.re, matcher.group, seen)...)
	}
	return values
}

func uniqueTrimmedMatchesWithSeen(source string, re *regexp.Regexp, group int, seen map[string]bool) []string {
	matches := re.FindAllStringSubmatch(source, -1)
	values := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) <= group {
			continue
		}
		value := strings.TrimSpace(match[group])
		value = strings.TrimSuffix(value, ";")
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	return values
}

func pyParamNames(params []pyParamInfo) []string {
	names := make([]string, 0, len(params))
	for _, p := range params {
		name := p.Name
		if p.IsArgs {
			name = "*" + name
		}
		if p.IsKwargs {
			name = "**" + name
		}
		if p.HasDefault {
			name += "?"
		}
		names = append(names, name)
	}
	return names
}

func pyBoundaryLabels(boundaries []pyBoundary) []string {
	labels := make([]string, 0, len(boundaries))
	for _, b := range boundaries {
		labels = append(labels, b.Param+"="+b.Value)
	}
	return labels
}
