package generator

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/sleticalboy/testloop-mcp/types"
)

// ============================================================
// Java test generator (JUnit 5)
// ============================================================

// GenerateJavaTests 为 Java 源码生成 JUnit 5 测试
func GenerateJavaTests(source []byte, filePath string) (string, string, error) {
	return generateJavaTests(source, filePath, nil)
}

func GenerateJavaTestsForCoverageTask(source []byte, filePath string, task *types.CoverageTestTask) (string, string, error) {
	if task == nil {
		return GenerateJavaTests(source, filePath)
	}
	return generateJavaTests(source, filePath, task)
}

func generateJavaTests(source []byte, filePath string, task *types.CoverageTestTask) (string, string, error) {
	funcs, classes := parseJavaWithTreeSitter(source)
	if task != nil {
		funcs = filterJavaFuncsForCoverageTask(funcs, task)
	}

	if len(funcs) == 0 && len(classes) == 0 {
		return "", "", fmt.Errorf("no testable methods found in %s", filePath)
	}

	baseName := baseName(filePath)
	// JUnit 5 测试文件：TypeNameTest.java
	className := strings.TrimSuffix(baseName, ".java")
	if len(className) > 0 {
		className = strings.ToUpper(className[:1]) + className[1:]
	}
	testFileName := className + "Test.java"

	var b strings.Builder

	// 生成测试文件头部
	b.WriteString(fmt.Sprintf("// Generated tests for %s\n", baseName))
	b.WriteString("// Run with: mvn test  or  gradle test\n")
	b.WriteString("import org.junit.jupiter.api.Test;\n")
	b.WriteString("import static org.junit.jupiter.api.Assertions.*;\n\n")

	b.WriteString(fmt.Sprintf("class %sTest {\n", className))

	// 为每个方法生成测试
	for _, m := range funcs {
		if !m.IsPublic {
			continue
		}
		javaWriteMethodTestForCoverageTask(&b, m, className, task)
	}

	b.WriteString("}\n")

	return testFileName, b.String(), nil
}

func filterJavaFuncsForCoverageTask(funcs []javaFuncInfo, task *types.CoverageTestTask) []javaFuncInfo {
	target := strings.TrimSpace(task.Target)
	if target == "" {
		return funcs
	}
	filtered := make([]javaFuncInfo, 0, len(funcs))
	for _, m := range funcs {
		if taskTargetMatches(target, m.ClassName, m.Name) {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) == 0 {
		return funcs
	}
	return filtered
}

// javaWriteMethodTest 为单个 Java 方法写一个 @Test 方法
func javaWriteMethodTest(b *strings.Builder, m javaFuncInfo, className string) {
	javaWriteMethodTestForCoverageTask(b, m, className, nil)
}

func javaWriteMethodTestForCoverageTask(b *strings.Builder, m javaFuncInfo, className string, task *types.CoverageTestTask) {
	indent := "    "

	b.WriteString(fmt.Sprintf("\n    @Test\n"))
	testName := javaTestMethodName(m.Name)
	if task != nil && strings.TrimSpace(task.TestName) != "" {
		testName = sanitizeJavaTestMethodName(task.TestName, testName)
	}
	b.WriteString(fmt.Sprintf("    void %s() {\n", testName))
	if comment := coverageTaskComment(task); comment != "" {
		b.WriteString(fmt.Sprintf("%s    // coverage task: %s\n", indent, comment))
	}

	// 构造调用参数
	args := javaBuildArgsForCoverageTask(m.Params, task)
	callClassName := className
	if m.ClassName != "" {
		callClassName = m.ClassName
	}

	if m.IsConstructor {
		// 构造函数测试
		b.WriteString(fmt.Sprintf("%s    %s instance = new %s(%s);\n", indent, callClassName, callClassName, args))
		b.WriteString(fmt.Sprintf("%s    assertNotNull(instance);\n", indent))
	} else if m.IsStatic {
		// 静态方法调用：ClassName.method(...)
		callExpr := fmt.Sprintf("%s.%s(%s)", callClassName, m.Name, args)
		javaWriteCallAndAssert(b, callExpr, m, indent)
	} else {
		// 实例方法：先创建实例
		b.WriteString(fmt.Sprintf("%s    %s instance = new %s();\n", indent, callClassName, callClassName))
		callExpr := fmt.Sprintf("instance.%s(%s)", m.Name, args)
		javaWriteCallAndAssert(b, callExpr, m, indent)
	}

	b.WriteString("    }\n")
}

// javaWriteCallAndAssert 写调用表达式和断言
func javaWriteCallAndAssert(b *strings.Builder, callExpr string, m javaFuncInfo, indent string) {
	if m.IsVoid {
		b.WriteString(fmt.Sprintf("%s    %s;\n", indent, callExpr))
	} else {
		varName := "result"
		b.WriteString(fmt.Sprintf("%s    %s %s = %s;\n", indent, m.ReturnType, varName, callExpr))
		assertion := javaInferAssert(m.ReturnType, varName)
		if assertion != "" {
			b.WriteString(fmt.Sprintf("%s    %s\n", indent, assertion))
		} else {
			b.WriteString(fmt.Sprintf("%s    // TODO: replace with actual expected value\n", indent))
			b.WriteString(fmt.Sprintf("%s    assertNotNull(%s);\n", indent, varName))
		}
	}

	// 如果有 throws，测试异常路径
	if len(m.Throws) > 0 {
		b.WriteString(fmt.Sprintf("\n%s    // Test exception path\n", indent))
		b.WriteString(fmt.Sprintf("%s    assertThrows(%s.class, () -> {\n", indent, m.Throws[0]))
		b.WriteString(fmt.Sprintf("%s        // TODO: call with invalid args\n", indent))
		b.WriteString(fmt.Sprintf("%s    });\n", indent))
	}
}

// javaBuildArgs 构造调用参数列表字符串
func javaBuildArgs(params []javaParamInfo) string {
	return javaBuildArgsForCoverageTask(params, nil)
}

func javaBuildArgsForCoverageTask(params []javaParamInfo, task *types.CoverageTestTask) string {
	values := coverageTaskInputValues(task, "java")
	var parts []string
	for _, p := range params {
		if value := values[p.Name]; value != "" {
			parts = append(parts, value)
		} else {
			parts = append(parts, javaInferDefaultValue(p.Type))
		}
	}
	return strings.Join(parts, ", ")
}

// javaTestMethodName 生成合法的 JUnit 测试方法名
func javaTestMethodName(methodName string) string {
	// 替换非法字符，保持可读性
	s := strings.ToLower(methodName)
	// Java 测试方法名可以包含下划线
	return s
}

// GenerateJavaTestsForSource 导出供 generator.go 调用
func GenerateJavaTestsForSource(source []byte, filePath string) (string, string, error) {
	return GenerateJavaTests(source, filePath)
}

func sanitizeJavaTestMethodName(name, fallback string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return fallback
	}
	var sb strings.Builder
	for i, r := range name {
		if r == '_' || unicode.IsLetter(r) || (unicode.IsDigit(r) && i > 0) {
			sb.WriteRune(r)
		}
	}
	got := sb.String()
	if got == "" {
		return fallback
	}
	if unicode.IsDigit([]rune(got)[0]) {
		return fallback
	}
	return got
}
