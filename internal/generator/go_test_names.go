package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/imports"
)

// AvoidDuplicateGoTestNames rewrites generated Go Test functions when the
// target package already contains a test with the same name.
func AvoidDuplicateGoTestNames(srcPath, outputPath, code string) (string, error) {
	if strings.ToLower(filepath.Ext(srcPath)) != ".go" {
		return code, nil
	}
	existing, err := existingGoPackageTestNames(filepath.Dir(srcPath), outputPath)
	if err != nil {
		return "", err
	}
	if len(existing) == 0 {
		return code, nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, outputPath+".generated", code, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("解析新生成 Go 测试代码失败: %w", err)
	}

	changed := false
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if !existing[fn.Name.Name] {
			existing[fn.Name.Name] = true
			continue
		}
		fn.Name.Name = uniqueGoTestLoopName(fn.Name.Name, existing)
		existing[fn.Name.Name] = true
		changed = true
	}
	if !changed {
		return code, nil
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, file); err != nil {
		return "", fmt.Errorf("重写 Go 测试函数名失败: %w", err)
	}
	formatted, err := imports.Process(outputPath, buf.Bytes(), nil)
	if err != nil {
		return buf.String(), fmt.Errorf("格式化重写后的 Go 测试失败: %w", err)
	}
	return string(formatted), nil
}

func existingGoPackageTestNames(dir, outputPath string) (map[string]bool, error) {
	names := map[string]bool{}
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		return nil, err
	}
	outputAbs, _ := filepath.Abs(outputPath)
	for _, path := range files {
		pathAbs, _ := filepath.Abs(path)
		if outputAbs != "" && pathAbs == outputAbs {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, data, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("解析已有 Go 测试文件失败: %w", err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Name != nil && strings.HasPrefix(fn.Name.Name, "Test") {
				names[fn.Name.Name] = true
			}
		}
	}
	return names, nil
}

func uniqueGoTestLoopName(name string, used map[string]bool) string {
	base := name + "TestLoop"
	if !used[base] {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s%d", base, i)
		if !used[candidate] {
			return candidate
		}
	}
}
