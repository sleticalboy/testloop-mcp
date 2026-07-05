package generator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/binlee/testloop-mcp/types"
)

// BuildGenerationContext extracts source structure for semantic test generation.
func BuildGenerationContext(srcPath string) *types.TestGenerationContext {
	return BuildGenerationContextWithOptions(srcPath, GenerateTestsOptions{})
}

func BuildGenerationContextWithOptions(srcPath string, opts GenerateTestsOptions) *types.TestGenerationContext {
	ext := strings.ToLower(filepath.Ext(srcPath))
	var ctx *types.TestGenerationContext
	switch ext {
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		ctx = buildJSGenerationContext(srcPath, ext)
	case ".py":
		ctx = buildPyGenerationContext(srcPath)
	}
	if opts.CoverageTask == nil {
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

func buildJSGenerationContext(srcPath, ext string) *types.TestGenerationContext {
	source, err := os.ReadFile(srcPath)
	if err != nil {
		return nil
	}

	funcs, classes, _ := parseJSWithTreeSitter(source, ext)
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
		ctx.Targets = append(ctx.Targets, jsTarget(fn, "function"))
	}
	for _, cls := range classes {
		for _, method := range cls.Methods {
			ctx.Targets = append(ctx.Targets, jsTarget(method, "method"))
		}
	}

	if len(ctx.Targets) == 0 {
		return nil
	}
	return ctx
}

func jsTarget(fn jsFuncInfo, kind string) types.TestTarget {
	target := types.TestTarget{
		Name:              fn.Name,
		Kind:              kind,
		ClassName:         fn.ClassName,
		Params:            jsParamNames(fn.Params),
		Async:             fn.IsAsync,
		ReturnType:        fn.Analysis.ReturnType,
		ReturnExpressions: fn.Analysis.Returns,
		HasErrorPath:      fn.Analysis.Throws,
		BoundaryCases:     jsBoundaryLabels(fn.Analysis.Boundaries),
	}
	if target.ReturnType == "" {
		target.ReturnType = "unknown"
	}
	return target
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
