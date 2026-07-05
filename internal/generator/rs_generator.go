package generator

import (
	"fmt"
	"strings"

	"github.com/sleticalboy/testloop-mcp/types"
)

// ============================================================
// Rust test generator
// ============================================================

// GenerateRustTests 为 Rust 源码生成 #[test] 测试
func GenerateRustTests(source []byte, filePath string) (string, string, error) {
	return generateRustTests(source, filePath, nil)
}

func GenerateRustTestsForCoverageTask(source []byte, filePath string, task *types.CoverageTestTask) (string, string, error) {
	if task == nil {
		return GenerateRustTests(source, filePath)
	}
	return generateRustTests(source, filePath, task)
}

func generateRustTests(source []byte, filePath string, task *types.CoverageTestTask) (string, string, error) {
	funcs, structs := parseRustWithTreeSitter(source)
	if task != nil {
		funcs = filterRustFuncsForCoverageTask(funcs, task)
	}

	if len(funcs) == 0 {
		return "", "", fmt.Errorf("no testable functions found in %s", filePath)
	}

	// 判断是模块测试（#![cfg(test)]）还是独立测试文件
	baseName := baseName(filePath)
	// Rust 测试文件名：xxx_test.rs 或 xxx.rs -> tests/xxx_test.rs
	testFileName := strings.TrimSuffix(baseName, ".rs") + "_test.rs"

	var b strings.Builder

	// 生成测试文件头部
	b.WriteString(fmt.Sprintf("// Generated tests for %s\n", baseName))
	b.WriteString("// Run with: cargo test\n\n")
	b.WriteString("#[cfg(test)]\n")
	b.WriteString("mod tests {\n")
	b.WriteString("    use super::*;\n\n")

	// 为每个函数生成测试
	for _, f := range funcs {
		// 跳过 private 且不是方法的函数（无法从测试模块访问）
		if !f.IsPub && !f.IsMethod {
			continue
		}

		// 普通函数测试
		rsWriteFuncTestForCoverageTask(&b, f, structs, task)

		// 如果返回 Result，额外生成 Err 分支测试
		if f.HasResult && task == nil {
			rsWriteResultErrTest(&b, f, structs)
		}
	}

	b.WriteString("}\n")

	return testFileName, b.String(), nil
}

func filterRustFuncsForCoverageTask(funcs []rsFuncInfo, task *types.CoverageTestTask) []rsFuncInfo {
	target := strings.TrimSpace(task.Target)
	if target == "" {
		return funcs
	}
	filtered := make([]rsFuncInfo, 0, len(funcs))
	for _, f := range funcs {
		if taskTargetMatches(target, f.Owner, f.Name) {
			filtered = append(filtered, f)
		}
	}
	if len(filtered) == 0 {
		return funcs
	}
	return filtered
}

// rsWriteFuncTest 为单个函数写一个 #[test] 块
func rsWriteFuncTest(b *strings.Builder, f rsFuncInfo, structs []rsStructInfo) {
	rsWriteFuncTestForCoverageTask(b, f, structs, nil)
}

func rsWriteFuncTestForCoverageTask(b *strings.Builder, f rsFuncInfo, structs []rsStructInfo, task *types.CoverageTestTask) {
	indent := "    "

	b.WriteString(fmt.Sprintf("\n    #[test]\n"))
	testName := "test_" + f.Name
	if task != nil && strings.TrimSpace(task.TestName) != "" {
		testName = sanitizeRustTestName(task.TestName, testName)
	}
	b.WriteString(fmt.Sprintf("    fn %s() {\n", testName))
	if comment := coverageTaskComment(task); comment != "" {
		b.WriteString(fmt.Sprintf("%s    // coverage task: %s\n", indent, comment))
	}

	// 构造调用参数
	args := rsBuildArgsForCoverageTask(f, structs, task)

	var callExpr string
	if f.IsMethod {
		// 方法调用：let instance = TypeName::new(); instance.method(args);
		typeName := rsInferTypeName(f, structs)
		if typeName != "" {
			b.WriteString(fmt.Sprintf("%s    let instance = %s::new();\n", indent, typeName))
			callExpr = fmt.Sprintf("instance.%s(%s)", f.Name, args)
		} else {
			callExpr = fmt.Sprintf("self.%s(%s)", f.Name, args)
		}
	} else {
		// 关联函数或普通函数：TypeName::func() 或 func()
		if f.HasSelf && !f.IsMethod {
			// 自由函数
			callExpr = fmt.Sprintf("%s(%s)", f.Name, args)
		} else {
			// 关联函数
			typeName := rsInferTypeName(f, structs)
			if typeName != "" && !f.IsMethod {
				callExpr = fmt.Sprintf("%s::%s(%s)", typeName, f.Name, args)
			} else {
				callExpr = fmt.Sprintf("%s(%s)", f.Name, args)
			}
		}
	}

	// 根据返回类型写断言
	if f.ReturnType == "" || f.ReturnType == "()" {
		b.WriteString(fmt.Sprintf("%s    %s;\n", indent, callExpr))
	} else if f.HasResult {
		b.WriteString(fmt.Sprintf("%s    let result = %s;\n", indent, callExpr))
		b.WriteString(fmt.Sprintf("%s    assert!(result.is_ok() || result.is_err());\n", indent))
	} else if f.HasOption {
		b.WriteString(fmt.Sprintf("%s    let result = %s;\n", indent, callExpr))
		b.WriteString(fmt.Sprintf("%s    // result may be Some(...) or None\n", indent))
		b.WriteString(fmt.Sprintf("%s    match result {\n", indent))
		b.WriteString(fmt.Sprintf("%s        Some(v) => println!(\"Got Some({{:?}})\", v),\n", indent))
		b.WriteString(fmt.Sprintf("%s        None => println!(\"Got None\"),\n", indent))
		b.WriteString(fmt.Sprintf("%s    }\n", indent))
	} else {
		b.WriteString(fmt.Sprintf("%s    let result = %s;\n", indent, callExpr))
		b.WriteString(fmt.Sprintf("%s    // TODO: replace with actual expected value\n", indent))
		if strings.Contains(f.ReturnType, "bool") {
			b.WriteString(fmt.Sprintf("%s    assert!(result == true || result == false);\n", indent))
		} else {
			b.WriteString(fmt.Sprintf("%s    assert!(result == %s);\n", indent, rsInferReturnValue(f.ReturnType)))
		}
	}

	b.WriteString("    }\n")
}

// rsWriteResultErrTest 为返回 Result 的函数生成 Err 分支测试
func rsWriteResultErrTest(b *strings.Builder, f rsFuncInfo, structs []rsStructInfo) {
	indent := "    "

	b.WriteString(fmt.Sprintf("\n    #[test]\n"))
	b.WriteString(fmt.Sprintf("    fn test_%s_returns_err_for_invalid_input() {\n", f.Name))

	args := rsBuildArgs(f, structs)
	// 用空值/零值调用，期望 Err
	if f.IsMethod {
		typeName := rsInferTypeName(f, structs)
		if typeName != "" {
			b.WriteString(fmt.Sprintf("%s    let instance = %s::new();\n", indent, typeName))
			b.WriteString(fmt.Sprintf("%s    let result = instance.%s(%s);\n", indent, f.Name, args))
		}
	} else {
		typeName := rsInferTypeName(f, structs)
		if typeName != "" {
			b.WriteString(fmt.Sprintf("%s    let result = %s::%s(%s);\n", indent, typeName, f.Name, args))
		} else {
			b.WriteString(fmt.Sprintf("%s    let result = %s(%s);\n", indent, f.Name, args))
		}
	}

	b.WriteString(fmt.Sprintf("%s    // This may return Err depending on input\n", indent))
	b.WriteString(fmt.Sprintf("%s    if let Err(e) = result {\n", indent))
	b.WriteString(fmt.Sprintf("%s        println!(\"Got expected error: {{:?}}\", e);\n", indent))
	b.WriteString(fmt.Sprintf("%s    }\n", indent))

	b.WriteString("    }\n")
}

// rsBuildArgs 构造调用参数列表字符串
func rsBuildArgs(f rsFuncInfo, structs []rsStructInfo) string {
	return rsBuildArgsForCoverageTask(f, structs, nil)
}

func rsBuildArgsForCoverageTask(f rsFuncInfo, structs []rsStructInfo, task *types.CoverageTestTask) string {
	values := coverageTaskInputValues(task, "rust")
	var parts []string
	for _, p := range f.Params {
		if p.IsSelf {
			continue
		}
		if value := values[p.Name]; value != "" {
			parts = append(parts, value)
		} else {
			parts = append(parts, rsInferDefaultValue(p.Type))
		}
	}
	return strings.Join(parts, ", ")
}

// rsInferTypeName 根据函数信息推断所属的类型名（用于 impl 块方法）
func rsInferTypeName(f rsFuncInfo, structs []rsStructInfo) string {
	if f.Owner != "" {
		return f.Owner
	}
	// 简单策略：从函数名推断，或者从 structs 列表里找
	// 实际项目中，解析器应该记录 impl 块对应的类型名
	// 这里用一个简化实现：如果 funcs 里有 self，返回第一个 struct 的名字
	if len(structs) > 0 {
		return structs[0].Name
	}
	return ""
}

// GenerateRustTestsForSource 导出供 generator.go 调用
func GenerateRustTestsForSource(source []byte, filePath string) (string, string, error) {
	return GenerateRustTests(source, filePath)
}

func sanitizeRustTestName(name, fallback string) string {
	return sanitizePythonTestName(name, fallback)
}
