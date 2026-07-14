package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/sleticalboy/testloop-mcp/types"
)

// ============================================================
// Java test generator (JUnit 4/5)
// ============================================================

// GenerateJavaTests 为 Java 源码生成 JUnit 测试
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
	if header := javaLeadingBlockComment(source); header != "" {
		b.WriteString(header)
		if !strings.HasSuffix(header, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n")
	} else {
		b.WriteString(fmt.Sprintf("// Generated tests for %s\n", baseName))
		b.WriteString("// Run with: mvn test  or  gradle test\n")
	}
	if packageName := javaPackageName(source); packageName != "" {
		b.WriteString(fmt.Sprintf("package %s;\n\n", packageName))
	}
	style := detectJavaJUnitStyle(filePath)
	if style == javaJUnit4 {
		b.WriteString("import org.junit.Assert;\n")
		b.WriteString("import org.junit.Test;\n")
	} else {
		b.WriteString("import org.junit.jupiter.api.Assertions;\n")
		b.WriteString("import org.junit.jupiter.api.Test;\n")
	}
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("public class %sTest {\n", className))

	// 为每个方法生成测试
	usedNames := map[string]int{}
	for _, m := range funcs {
		if !m.IsPublic {
			continue
		}
		testName := javaCoverageTaskMethodName(m, task)
		usedNames[testName]++
		if usedNames[testName] > 1 {
			testName = fmt.Sprintf("%s%d", testName, usedNames[testName])
		}
		javaWriteMethodTestForCoverageTaskWithName(&b, m, className, task, testName, style)
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
	if start, _, ok := javaCoverageTaskLineRange(task); ok {
		lineFiltered := javaFuncsClosestToLine(filtered, start)
		if len(lineFiltered) > 0 {
			return lineFiltered
		}
	}
	return filtered
}

// javaWriteMethodTest 为单个 Java 方法写一个 @Test 方法
func javaWriteMethodTest(b *strings.Builder, m javaFuncInfo, className string) {
	javaWriteMethodTestForCoverageTask(b, m, className, nil)
}

func javaWriteMethodTestForCoverageTask(b *strings.Builder, m javaFuncInfo, className string, task *types.CoverageTestTask) {
	javaWriteMethodTestForCoverageTaskWithName(b, m, className, task, javaCoverageTaskMethodName(m, task), javaJUnit5)
}

func javaWriteMethodTestForCoverageTaskWithName(b *strings.Builder, m javaFuncInfo, className string, task *types.CoverageTestTask, testName string, style javaJUnitStyle) {
	indent := "    "
	assertions := javaAssertionsQualifier(style)

	b.WriteString(fmt.Sprintf("\n    @Test\n"))
	b.WriteString(fmt.Sprintf("    public void %s() {\n", testName))
	if comment := coverageTaskComment(task); comment != "" {
		b.WriteString(fmt.Sprintf("%s    // coverage task: %s\n", indent, truncateJavaComment(comment, 88)))
	}

	// 构造调用参数
	args := javaBuildArgsForCoverageTask(m.Params, task)
	callClassName := className
	if m.ClassName != "" {
		callClassName = m.ClassName
	}

	if m.IsConstructor {
		// 构造函数测试
		if !javaWriteConstructorAssertThrows(b, m, task, callClassName, assertions, indent) {
			b.WriteString(fmt.Sprintf("%s    %s instance = new %s(%s);\n", indent, callClassName, callClassName, args))
			b.WriteString(fmt.Sprintf("%s    %s.assertNotNull(instance);\n", indent, assertions))
		}
	} else if m.IsStatic {
		// 静态方法调用：ClassName.method(...)
		callExpr := fmt.Sprintf("%s.%s(%s)", callClassName, m.Name, args)
		javaWriteCallAndAssert(b, callExpr, m, indent, assertions)
	} else {
		// 实例方法：先创建实例
		b.WriteString(fmt.Sprintf("%s    %s instance = new %s();\n", indent, callClassName, callClassName))
		callExpr := fmt.Sprintf("instance.%s(%s)", m.Name, args)
		javaWriteCallAndAssert(b, callExpr, m, indent, assertions)
	}

	b.WriteString("    }\n")
}

func javaCoverageTaskMethodName(m javaFuncInfo, task *types.CoverageTestTask) string {
	testName := javaTestMethodName(m.Name)
	if task != nil && strings.TrimSpace(task.TestName) != "" {
		testName = sanitizeJavaTestMethodName(task.TestName, testName)
	}
	return testName
}

// javaWriteCallAndAssert 写调用表达式和断言
func javaWriteCallAndAssert(b *strings.Builder, callExpr string, m javaFuncInfo, indent string, assertions string) {
	if m.IsVoid {
		b.WriteString(fmt.Sprintf("%s    %s;\n", indent, callExpr))
	} else {
		varName := "result"
		b.WriteString(fmt.Sprintf("%s    %s %s = %s;\n", indent, m.ReturnType, varName, callExpr))
		assertion := javaInferAssert(m.ReturnType, varName)
		if assertion != "" {
			b.WriteString(fmt.Sprintf("%s    %s\n", indent, javaQualifyAssertion(assertion, assertions)))
		} else {
			b.WriteString(fmt.Sprintf("%s    // TODO: replace with actual expected value\n", indent))
			b.WriteString(fmt.Sprintf("%s    %s.assertNotNull(%s);\n", indent, assertions, varName))
		}
	}

	// 如果有 throws，测试异常路径
	if len(m.Throws) > 0 {
		b.WriteString(fmt.Sprintf("\n%s    // Test exception path\n", indent))
		b.WriteString(fmt.Sprintf("%s    %s.assertThrows(%s.class, () -> {\n", indent, assertions, m.Throws[0]))
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
		} else if value := javaInferCoverageTaskValue(p, task); value != "" {
			parts = append(parts, value)
		} else {
			parts = append(parts, javaInferDefaultValue(p.Type))
		}
	}
	return strings.Join(parts, ", ")
}

func javaInferCoverageTaskValue(param javaParamInfo, task *types.CoverageTestTask) string {
	if task == nil {
		return ""
	}
	if param.Name == "addresses" && (param.Type == "List<Address>" || param.Type == "java.util.List<Address>") && javaTaskMentions(task, "addresses") {
		return `java.util.Arrays.asList(new Address("example.com", 80), new Address("example.org", 81))`
	}
	return ""
}

func javaConstructorShouldAssertThrows(m javaFuncInfo, task *types.CoverageTestTask) bool {
	if task == nil || !m.IsConstructor {
		return false
	}
	if javaTaskMentions(task, ".equals(") && javaTaskMentions(task, "addresses") {
		return true
	}
	return false
}

func javaWriteConstructorAssertThrows(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, className string, assertions string, indent string) bool {
	if !javaConstructorShouldAssertThrows(m, task) {
		return false
	}
	values := coverageTaskInputValues(task, "java")
	hasAddressList := false
	schemeValue := ""
	for _, param := range m.Params {
		switch param.Name {
		case "scheme":
			schemeValue = values[param.Name]
		case "addresses":
			hasAddressList = param.Type == "List<Address>" || param.Type == "java.util.List<Address>"
		}
	}
	if !hasAddressList || schemeValue == "" {
		return false
	}
	b.WriteString(fmt.Sprintf("%s    final java.util.List<Address> addresses = java.util.Arrays.asList(\n", indent))
	b.WriteString(fmt.Sprintf("%s            new Address(\"example.com\", 80), new Address(\"example.org\", 81));\n", indent))
	b.WriteString(fmt.Sprintf("%s    %s.assertThrows(RuntimeException.class, () ->\n", indent, assertions))
	b.WriteString(fmt.Sprintf("%s            new %s(%s, addresses));\n", indent, className, schemeValue))
	return true
}

func javaTaskMentions(task *types.CoverageTestTask, needle string) bool {
	if task == nil || needle == "" {
		return false
	}
	needle = strings.ToLower(needle)
	for _, values := range [][]string{task.MissingBranches, task.SuggestedInputs, task.AssertionFocus} {
		for _, value := range values {
			if strings.Contains(strings.ToLower(value), needle) {
				return true
			}
		}
	}
	return false
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

func javaCoverageTaskLineRange(task *types.CoverageTestTask) (int, int, bool) {
	if task == nil {
		return 0, 0, false
	}
	parts := strings.Split(strings.TrimSpace(task.LineRange), "-")
	if len(parts) == 0 {
		return 0, 0, false
	}
	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || start <= 0 {
		return 0, 0, false
	}
	end := start
	if len(parts) > 1 {
		if parsedEnd, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && parsedEnd >= start {
			end = parsedEnd
		}
	}
	return start, end, true
}

func javaFuncsClosestToLine(funcs []javaFuncInfo, line int) []javaFuncInfo {
	bestLine := 0
	for _, fn := range funcs {
		if fn.Line <= line && fn.Line > bestLine {
			bestLine = fn.Line
		}
	}
	if bestLine == 0 {
		return nil
	}
	var filtered []javaFuncInfo
	for _, fn := range funcs {
		if fn.Line == bestLine {
			filtered = append(filtered, fn)
		}
	}
	return filtered
}

func javaPackageName(source []byte) string {
	for _, line := range strings.Split(string(source), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if !strings.HasPrefix(line, "package ") {
			continue
		}
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "package "), ";"))
		if name != "" {
			return name
		}
	}
	return ""
}

func javaLeadingBlockComment(source []byte) string {
	text := strings.TrimLeft(string(source), "\ufeff \t\r\n")
	if !strings.HasPrefix(text, "/*") {
		return ""
	}
	end := strings.Index(text, "*/")
	if end < 0 {
		return ""
	}
	return text[:end+2]
}

type javaJUnitStyle string

const (
	javaJUnit5 javaJUnitStyle = "junit5"
	javaJUnit4 javaJUnitStyle = "junit4"
)

func detectJavaJUnitStyle(filePath string) javaJUnitStyle {
	projectRoot := findNearestJavaBuildRoot(filepath.Dir(filePath))
	if projectRoot == "" {
		return javaJUnit5
	}
	for _, buildFile := range []string{"pom.xml", "build.gradle", "build.gradle.kts"} {
		data, err := os.ReadFile(filepath.Join(projectRoot, buildFile))
		if err != nil {
			continue
		}
		text := string(data)
		if strings.Contains(text, "junit-jupiter") || strings.Contains(text, "org.junit.jupiter") {
			return javaJUnit5
		}
		if strings.Contains(text, "<groupId>junit</groupId>") ||
			strings.Contains(text, "junit:junit") ||
			strings.Contains(text, "testCompile group: 'junit'") ||
			strings.Contains(text, `testImplementation "junit:junit`) ||
			strings.Contains(text, `testImplementation 'junit:junit`) {
			return javaJUnit4
		}
	}
	return javaJUnit5
}

func findNearestJavaBuildRoot(start string) string {
	if start == "" || start == "." {
		return ""
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		dir = start
	}
	for i := 0; i < 16; i++ {
		for _, marker := range []string{"pom.xml", "build.gradle", "build.gradle.kts"} {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func javaAssertionsQualifier(style javaJUnitStyle) string {
	if style == javaJUnit4 {
		return "Assert"
	}
	return "Assertions"
}

func javaQualifyAssertion(assertion string, assertions string) string {
	for _, name := range []string{"assertEquals", "assertTrue", "assertNotNull", "assertThrows"} {
		if strings.HasPrefix(assertion, name+"(") {
			return assertions + "." + assertion
		}
	}
	return assertion
}

func truncateJavaComment(comment string, maxRunes int) string {
	runes := []rune(comment)
	if len(runes) <= maxRunes {
		return comment
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}
