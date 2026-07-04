package generator

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/binlee/testloop-mcp/types"
)

// BuildGenerationContext extracts source structure for semantic test generation.
func BuildGenerationContext(srcPath string) *types.TestGenerationContext {
	ext := strings.ToLower(filepath.Ext(srcPath))
	switch ext {
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		return buildJSGenerationContext(srcPath, ext)
	case ".py":
		return buildPyGenerationContext(srcPath)
	default:
		return nil
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
		Name:          fn.Name,
		Kind:          kind,
		ClassName:     fn.ClassName,
		Params:        jsParamNames(fn.Params),
		Async:         fn.IsAsync,
		ReturnType:    fn.Analysis.ReturnType,
		HasErrorPath:  fn.Analysis.Throws,
		BoundaryCases: jsBoundaryLabels(fn.Analysis.Boundaries),
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
		Name:          fn.Name,
		Kind:          kind,
		ClassName:     fn.ClassName,
		Params:        pyParamNames(fn.Params),
		Async:         fn.IsAsync,
		ReturnType:    fn.Analysis.ReturnType,
		HasErrorPath:  fn.Analysis.Raises,
		BoundaryCases: pyBoundaryLabels(fn.Analysis.Boundaries),
	}
	if target.ReturnType == "" {
		target.ReturnType = "unknown"
	}
	return target
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
