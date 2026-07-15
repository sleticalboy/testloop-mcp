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
	allFuncs := funcs
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

	b.WriteString(fmt.Sprintf("public class %sTest%s {\n", className, javaTestClassExtends(funcs, task)))

	// 为每个方法生成测试
	constructors := javaConstructorsByClass(allFuncs)
	factories := javaStaticFactoriesByClass(allFuncs)
	usedNames := map[string]int{}
	for _, m := range funcs {
		if !m.IsPublic {
			if task != nil {
				testName := javaCoverageTaskMethodName(m, task)
				usedNames[testName]++
				if usedNames[testName] > 1 {
					testName = fmt.Sprintf("%s%d", testName, usedNames[testName])
				}
				if javaWriteConsumerImplFilterExpressionTask(&b, m, task, testName, style) {
					continue
				}
				if javaWriteConsumerImplAckMessageTask(&b, m, task, testName, style) {
					continue
				}
				if javaWriteConsumerImplChangeInvisibleDurationTask(&b, m, task, testName, style) {
					continue
				}
				if javaWriteConsumerImplReceiveMessageRequestTask(&b, m, task, testName, style) {
					continue
				}
				javaWriteInternalManualReviewTest(&b, m, task, testName, style)
			}
			continue
		}
		if task == nil && javaIsTestHelper(m.Name) {
			continue
		}
		testName := javaCoverageTaskMethodName(m, task)
		usedNames[testName]++
		if usedNames[testName] > 1 {
			testName = fmt.Sprintf("%s%d", testName, usedNames[testName])
		}
		javaWriteMethodTestForCoverageTaskWithName(&b, m, className, task, testName, style, constructors, factories)
	}

	b.WriteString("}\n")

	return testFileName, javaAddGeneratedImports(b.String(), source), nil
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
	javaWriteMethodTestForCoverageTaskWithName(b, m, className, task, javaCoverageTaskMethodName(m, task), javaJUnit5, nil, nil)
}

func javaWriteMethodTestForCoverageTaskWithName(b *strings.Builder, m javaFuncInfo, className string, task *types.CoverageTestTask, testName string, style javaJUnitStyle, constructors map[string][]javaFuncInfo, factories map[string][]javaFuncInfo) {
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
	if javaWriteStatusCheckerCheckTask(b, m, task, assertions, indent) {
		b.WriteString("    }\n")
		return
	}

	if m.IsConstructor {
		// 构造函数测试
		if !javaWriteProtobufEndpointsConstructorTask(b, m, task, callClassName, assertions, indent) &&
			!javaWriteConstructorAssertThrows(b, m, task, callClassName, assertions, indent) &&
			!javaWriteAddressListConstructorTask(b, m, task, callClassName, assertions, indent) {
			b.WriteString(fmt.Sprintf("%s    %s instance = new %s(%s);\n", indent, callClassName, callClassName, args))
			b.WriteString(fmt.Sprintf("%s    %s.assertNotNull(instance);\n", indent, assertions))
		}
	} else if m.IsStatic {
		// 静态方法调用：ClassName.method(...)
		callExpr := fmt.Sprintf("%s.%s(%s)", callClassName, m.Name, args)
		javaWriteCallAndAssert(b, callExpr, m, indent, assertions)
	} else {
		// 实例方法：先创建实例
		if javaWriteEnumMethodTaskAssertion(b, m, task, assertions, indent) {
			b.WriteString("    }\n")
			return
		}
		if javaWriteInflightRequestCountInterceptorTask(b, m, task, assertions, indent) {
			b.WriteString("    }\n")
			return
		}
		if javaWriteCompositedMessageInterceptorTask(b, m, task, assertions, indent) {
			b.WriteString("    }\n")
			return
		}
		if javaWriteConsumeTaskCallTask(b, m, task, assertions, indent) {
			b.WriteString("    }\n")
			return
		}
		if javaWriteConsumeServiceConsumeTask(b, m, task, assertions, indent) {
			b.WriteString("    }\n")
			return
		}
		instanceExpr, canConstruct := javaInstanceConstructionForCoverageTask(callClassName, constructors, factories, task)
		if !canConstruct && task != nil {
			javaWriteManualReviewAssumption(b, indent, style, strings.TrimSpace(task.Target),
				"requires complex constructor state; cover it through a public entry point or review manually.")
			b.WriteString("    }\n")
			return
		}
		b.WriteString(fmt.Sprintf("%s    %s instance = %s;\n", indent, callClassName, instanceExpr))
		if javaWriteEqualsTaskAssertion(b, m, task, assertions, indent) {
			b.WriteString("    }\n")
			return
		}
		if javaWriteHashCodeTaskAssertion(b, m, task, assertions, indent) {
			b.WriteString("    }\n")
			return
		}
		if javaWriteToSocketAddressesTaskAssertion(b, m, task, assertions, indent) {
			b.WriteString("    }\n")
			return
		}
		callExpr := fmt.Sprintf("instance.%s(%s)", m.Name, args)
		javaWriteCallAndAssert(b, callExpr, m, indent, assertions)
	}

	b.WriteString("    }\n")
}

func javaWriteInternalManualReviewTest(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, testName string, style javaJUnitStyle) {
	indent := "    "
	target := strings.TrimSpace(task.Target)
	if target == "" {
		target = strings.TrimSpace(m.ClassName + "." + m.Name)
	}
	b.WriteString(fmt.Sprintf("\n    @Test\n"))
	b.WriteString(fmt.Sprintf("    public void %s() {\n", testName))
	if comment := coverageTaskComment(task); comment != "" {
		b.WriteString(fmt.Sprintf("%s    // coverage task: %s\n", indent, truncateJavaComment(comment, 88)))
	}
	javaWriteManualReviewAssumption(b, indent, style, target,
		"is private/internal; cover it through a public entry point or review manually.")
	b.WriteString("    }\n")
}

func javaTestClassExtends(funcs []javaFuncInfo, task *types.CoverageTestTask) string {
	for _, m := range funcs {
		if javaNeedsRocketMQTestBase(m, task) {
			return " extends TestBase"
		}
	}
	return ""
}

func javaNeedsRocketMQTestBase(m javaFuncInfo, task *types.CoverageTestTask) bool {
	return task != nil && (m.ClassName == "ConsumeTask" && m.Name == "call" ||
		m.ClassName == "ConsumeService" && m.Name == "consume" ||
		m.ClassName == "ConsumerImpl" && m.Name == "wrapFilterExpression" ||
		m.ClassName == "ConsumerImpl" && m.Name == "ackMessage" ||
		m.ClassName == "ConsumerImpl" && m.Name == "changeInvisibleDuration" ||
		m.ClassName == "ConsumerImpl" && m.Name == "wrapReceiveMessageRequest")
}

func javaWriteManualReviewAssumption(b *strings.Builder, indent string, style javaJUnitStyle, target string, detail string) {
	b.WriteString(fmt.Sprintf("%s    final String target = \"%s\";\n", indent, javaEscapeStringLiteral(target)))
	b.WriteString(fmt.Sprintf("%s    final String reason =\n", indent))
	b.WriteString(fmt.Sprintf("%s            \"manual_review_internal: \" + target\n", indent))
	segments := javaManualReviewDetailSegments(detail)
	for i, segment := range segments {
		suffix := ""
		if i == len(segments)-1 {
			suffix = ";"
		}
		b.WriteString(fmt.Sprintf("%s                    + \"%s\"%s\n", indent, javaEscapeStringLiteral(segment), suffix))
	}
	if style == javaJUnit4 {
		b.WriteString(fmt.Sprintf("%s    org.junit.Assume.assumeTrue(reason, false);\n", indent))
	} else {
		b.WriteString(fmt.Sprintf("%s    org.junit.jupiter.api.Assumptions.assumeTrue(false, reason);\n", indent))
	}
}

func javaManualReviewDetailSegments(detail string) []string {
	parts := strings.Split(detail, ";")
	segments := make([]string, 0, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		prefix := " "
		if i > 0 {
			prefix = ""
		}
		if i < len(parts)-1 {
			part += "; "
		}
		segments = append(segments, prefix+part)
	}
	if len(segments) == 0 {
		return []string{" requires manual review."}
	}
	return segments
}

func javaCoverageTaskMethodName(m javaFuncInfo, task *types.CoverageTestTask) string {
	testName := javaTestMethodName(m.Name)
	if task != nil && strings.TrimSpace(task.TestName) != "" {
		testName = sanitizeJavaTestMethodName(task.TestName, testName)
	}
	return testName
}

func javaEscapeStringLiteral(value string) string {
	return strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n", "\r", "\\r").Replace(value)
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
			parts = append(parts, javaInferDefaultArgValue(p.Type))
		}
	}
	return strings.Join(parts, ", ")
}

func javaInferDefaultArgValue(typ string) string {
	value := javaInferDefaultValue(typ)
	if value == "null" && strings.TrimSpace(typ) != "" {
		if !strings.Contains(typ, ".") {
			return "null"
		}
		return fmt.Sprintf("(%s) null", typ)
	}
	return value
}

func javaInferCoverageTaskValue(param javaParamInfo, task *types.CoverageTestTask) string {
	if task == nil {
		return ""
	}
	if param.Name == "addresses" && (param.Type == "List<Address>" || param.Type == "java.util.List<Address>") && javaTaskMentions(task, "addresses") {
		if javaTaskMentions(task, "addresses.isEmpty") {
			return "java.util.Collections.emptyList()"
		}
		return `java.util.Arrays.asList(new Address("example.com", 80), new Address("example.org", 81))`
	}
	if param.Name == "scheme" && strings.HasSuffix(param.Type, "Scheme") && javaTaskMentions(task, "addresses.isEmpty") {
		return param.Type + ".IPv4"
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
	if javaTaskMentions(task, "addresses.isEmpty") {
		return true
	}
	if javaTaskMentions(task, "空值") && javaTaskMentions(task, "addresses") {
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
	useEmptyAddresses := javaTaskMentions(task, "addresses.isEmpty")
	useNullAddresses := javaTaskMentions(task, "空值") && javaTaskMentions(task, "addresses")
	for _, param := range m.Params {
		switch param.Name {
		case "scheme":
			schemeValue = values[param.Name]
			if schemeValue == "" && strings.HasSuffix(param.Type, "Scheme") {
				schemeValue = param.Type + ".IPv4"
			}
		case "addresses":
			hasAddressList = param.Type == "List<Address>" || param.Type == "java.util.List<Address>"
		}
	}
	if !hasAddressList || schemeValue == "" {
		return false
	}
	if useNullAddresses {
		b.WriteString(fmt.Sprintf("%s    final java.util.List<Address> addresses = null;\n", indent))
	} else if useEmptyAddresses {
		b.WriteString(fmt.Sprintf("%s    final java.util.List<Address> addresses = java.util.Collections.emptyList();\n", indent))
	} else {
		b.WriteString(fmt.Sprintf("%s    final java.util.List<Address> addresses = java.util.Arrays.asList(\n", indent))
		b.WriteString(fmt.Sprintf("%s            new Address(\"example.com\", 80), new Address(\"example.org\", 81));\n", indent))
	}
	b.WriteString(fmt.Sprintf("%s    %s.assertThrows(RuntimeException.class, () ->\n", indent, assertions))
	b.WriteString(fmt.Sprintf("%s            new %s(%s, addresses));\n", indent, className, schemeValue))
	return true
}

func javaWriteAddressListConstructorTask(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, className string, assertions string, indent string) bool {
	if task == nil || !m.IsConstructor || !javaTaskMentions(task, "scheme") || !javaTaskMentions(task, "addresses") {
		return false
	}
	hasScheme := false
	hasAddressList := false
	schemeValue := ""
	for _, param := range m.Params {
		switch param.Name {
		case "scheme":
			hasScheme = strings.HasSuffix(param.Type, "Scheme")
			if hasScheme {
				schemeValue = param.Type + ".IPv4"
			}
		case "addresses":
			hasAddressList = param.Type == "List<Address>" || param.Type == "java.util.List<Address>"
		}
	}
	if !hasScheme || !hasAddressList || schemeValue == "" {
		return false
	}
	b.WriteString(fmt.Sprintf("%s    final java.util.List<Address> addresses = java.util.Arrays.asList(\n", indent))
	b.WriteString(fmt.Sprintf("%s            new Address(\"127.0.0.1\", 80),\n", indent))
	b.WriteString(fmt.Sprintf("%s            new Address(\"127.0.0.2\", 81));\n", indent))
	b.WriteString(fmt.Sprintf("%s    %s instance = new %s(%s, addresses);\n", indent, className, className, schemeValue))
	b.WriteString(fmt.Sprintf("%s    %s.assertNotNull(instance);\n", indent, assertions))
	return true
}

func javaWriteProtobufEndpointsConstructorTask(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, className string, assertions string, indent string) bool {
	if task == nil || !m.IsConstructor || len(m.Params) != 1 || m.Params[0].Type != "apache.rocketmq.v2.Endpoints" {
		return false
	}
	switch {
	case javaTaskMentions(task, "addresses.isEmpty"):
		return javaWriteProtobufEndpointsEmptyTask(b, className, assertions, indent)
	case javaTaskMentions(task, "addresses.size"):
		return javaWriteProtobufEndpointsDomainMultiAddressTask(b, className, assertions, indent)
	case javaTaskMentions(task, "switch/case"):
		b.WriteString(fmt.Sprintf("%s    final apache.rocketmq.v2.Endpoints endpoints = apache.rocketmq.v2.Endpoints.newBuilder()\n", indent))
		b.WriteString(fmt.Sprintf("%s            .setScheme(apache.rocketmq.v2.AddressScheme.IPv4)\n", indent))
		b.WriteString(fmt.Sprintf("%s            .addAddresses(apache.rocketmq.v2.Address.newBuilder().setHost(\"127.0.0.1\").setPort(80))\n", indent))
		b.WriteString(fmt.Sprintf("%s            .build();\n", indent))
		b.WriteString(fmt.Sprintf("%s    %s instance = new %s(endpoints);\n", indent, className, className))
		b.WriteString(fmt.Sprintf("%s    %s.assertNotNull(instance);\n", indent, assertions))
		return true
	default:
		if start, _, ok := javaCoverageTaskLineRange(task); ok {
			switch {
			case start <= m.Line+6:
				return javaWriteProtobufEndpointsEmptyTask(b, className, assertions, indent)
			case start >= m.Line+20:
				return javaWriteProtobufEndpointsDomainMultiAddressTask(b, className, assertions, indent)
			}
		}
		return false
	}
}

func javaWriteProtobufEndpointsEmptyTask(b *strings.Builder, className string, assertions string, indent string) bool {
	b.WriteString(fmt.Sprintf("%s    final apache.rocketmq.v2.Endpoints endpoints = apache.rocketmq.v2.Endpoints.newBuilder()\n", indent))
	b.WriteString(fmt.Sprintf("%s            .setScheme(apache.rocketmq.v2.AddressScheme.IPv4)\n", indent))
	b.WriteString(fmt.Sprintf("%s            .build();\n", indent))
	b.WriteString(fmt.Sprintf("%s    %s.assertThrows(RuntimeException.class, () -> new %s(endpoints));\n", indent, assertions, className))
	return true
}

func javaWriteProtobufEndpointsDomainMultiAddressTask(b *strings.Builder, className string, assertions string, indent string) bool {
	b.WriteString(fmt.Sprintf("%s    final apache.rocketmq.v2.Endpoints endpoints = apache.rocketmq.v2.Endpoints.newBuilder()\n", indent))
	b.WriteString(fmt.Sprintf("%s            .setScheme(apache.rocketmq.v2.AddressScheme.DOMAIN_NAME)\n", indent))
	b.WriteString(fmt.Sprintf("%s            .addAddresses(apache.rocketmq.v2.Address.newBuilder().setHost(\"a.example\").setPort(80))\n", indent))
	b.WriteString(fmt.Sprintf("%s            .addAddresses(apache.rocketmq.v2.Address.newBuilder().setHost(\"b.example\").setPort(81))\n", indent))
	b.WriteString(fmt.Sprintf("%s            .build();\n", indent))
	b.WriteString(fmt.Sprintf("%s    %s.assertThrows(RuntimeException.class, () -> new %s(endpoints));\n", indent, assertions, className))
	return true
}

func javaConstructorsByClass(funcs []javaFuncInfo) map[string][]javaFuncInfo {
	constructors := map[string][]javaFuncInfo{}
	for _, fn := range funcs {
		if fn.IsConstructor && fn.ClassName != "" {
			constructors[fn.ClassName] = append(constructors[fn.ClassName], fn)
		}
	}
	return constructors
}

func javaStaticFactoriesByClass(funcs []javaFuncInfo) map[string][]javaFuncInfo {
	factories := map[string][]javaFuncInfo{}
	for _, fn := range funcs {
		if !fn.IsPublic || !fn.IsStatic || fn.IsConstructor || fn.ClassName == "" {
			continue
		}
		if !javaFactoryReturnsClass(fn.ReturnType, fn.ClassName) {
			continue
		}
		switch fn.Name {
		case "create", "of", "from", "valueOf":
			factories[fn.ClassName] = append(factories[fn.ClassName], fn)
		}
	}
	return factories
}

func javaFactoryReturnsClass(returnType string, className string) bool {
	return javaRawTypeName(returnType) == className
}

func javaRawTypeName(typ string) string {
	typ = strings.TrimSpace(typ)
	if idx := strings.Index(typ, "<"); idx >= 0 {
		typ = typ[:idx]
	}
	if idx := strings.LastIndex(typ, "."); idx >= 0 {
		typ = typ[idx+1:]
	}
	return strings.TrimSpace(typ)
}

func javaInstanceConstruction(className string, constructors map[string][]javaFuncInfo, factories map[string][]javaFuncInfo, task *types.CoverageTestTask) string {
	instanceExpr, _ := javaInstanceConstructionForCoverageTask(className, constructors, factories, task)
	return instanceExpr
}

func javaInstanceConstructionForCoverageTask(className string, constructors map[string][]javaFuncInfo, factories map[string][]javaFuncInfo, task *types.CoverageTestTask) (string, bool) {
	for _, constructor := range constructors[className] {
		if constructor.IsPublic && len(constructor.Params) == 0 {
			return fmt.Sprintf("new %s()", className), true
		}
	}
	if javaTaskMentions(task, "DOMAIN_NAME") {
		for _, constructor := range constructors[className] {
			if constructor.IsPublic && len(constructor.Params) == 1 && constructor.Params[0].Type == "String" {
				return fmt.Sprintf("new %s(\"example.com:80\")", className), true
			}
		}
	}
	for _, constructor := range constructors[className] {
		if constructor.IsPublic && len(constructor.Params) == 1 && constructor.Params[0].Type == "String" {
			return fmt.Sprintf("new %s(\"127.0.0.1:80\")", className), true
		}
	}
	for _, factory := range factories[className] {
		return fmt.Sprintf("%s.%s(%s)", className, factory.Name, javaBuildArgsForCoverageTask(factory.Params, task)), true
	}
	for _, constructor := range constructors[className] {
		if constructor.IsPublic {
			return fmt.Sprintf("new %s(%s)", className, javaBuildArgsForCoverageTask(constructor.Params, task)), true
		}
	}
	if len(constructors[className]) > 0 {
		return "", false
	}
	return fmt.Sprintf("new %s()", className), true
}

func javaWriteEqualsTaskAssertion(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, assertions string, indent string) bool {
	if m.Name != "equals" || len(m.Params) != 1 {
		return false
	}
	if start, _, ok := javaCoverageTaskLineRange(task); ok {
		switch {
		case start <= m.Line+3:
			b.WriteString(fmt.Sprintf("%s    %s.assertTrue(instance.equals(instance));\n", indent, assertions))
			return true
		case start <= m.Line+6:
			b.WriteString(fmt.Sprintf("%s    %s.assertFalse(instance.equals(new Object()));\n", indent, assertions))
			return true
		default:
			className := m.ClassName
			if className == "" {
				className = "Endpoints"
			}
			b.WriteString(fmt.Sprintf("%s    %s other = new %s(\"127.0.0.1:80\");\n", indent, className, className))
			b.WriteString(fmt.Sprintf("%s    %s.assertTrue(instance.equals(other));\n", indent, assertions))
			return true
		}
	}
	if javaTaskMentions(task, "this == o") {
		b.WriteString(fmt.Sprintf("%s    %s.assertTrue(instance.equals(instance));\n", indent, assertions))
		return true
	}
	if javaTaskMentions(task, "o == null") {
		b.WriteString(fmt.Sprintf("%s    %s.assertFalse(instance.equals(null));\n", indent, assertions))
		return true
	}
	return false
}

func javaWriteToSocketAddressesTaskAssertion(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, assertions string, indent string) bool {
	if m.Name != "toSocketAddresses" || m.ReturnType != "List<InetSocketAddress>" {
		return false
	}
	className := m.ClassName
	if className == "" {
		className = "Endpoints"
	}
	if start, _, ok := javaCoverageTaskLineRange(task); ok && start >= m.Line+3 {
		b.WriteString(fmt.Sprintf("%s    %s domainInstance = new %s(\"example.com:80\");\n", indent, className, className))
		b.WriteString(fmt.Sprintf("%s    List<InetSocketAddress> result = domainInstance.toSocketAddresses();\n", indent))
		b.WriteString(fmt.Sprintf("%s    %s.assertNull(result);\n", indent, assertions))
		return true
	}
	if javaTaskMentions(task, "空值") || javaTaskMentions(task, "null") || javaTaskMentions(task, "DOMAIN_NAME") {
		b.WriteString(fmt.Sprintf("%s    %s domainInstance = new %s(\"example.com:80\");\n", indent, className, className))
		b.WriteString(fmt.Sprintf("%s    List<InetSocketAddress> result = domainInstance.toSocketAddresses();\n", indent))
		b.WriteString(fmt.Sprintf("%s    %s.assertNull(result);\n", indent, assertions))
		return true
	}
	b.WriteString(fmt.Sprintf("%s    List<InetSocketAddress> result = instance.toSocketAddresses();\n", indent))
	b.WriteString(fmt.Sprintf("%s    %s.assertNotNull(result);\n", indent, assertions))
	b.WriteString(fmt.Sprintf("%s    %s.assertFalse(result.isEmpty());\n", indent, assertions))
	return true
}

func javaWriteStatusCheckerCheckTask(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, assertions string, indent string) bool {
	if m.ClassName != "StatusChecker" || m.Name != "check" || len(m.Params) != 2 {
		return false
	}
	code := "apache.rocketmq.v2.Code.OK"
	request := "new Object()"
	if javaTaskMentions(task, "future.getRequest") {
		code = "apache.rocketmq.v2.Code.MESSAGE_NOT_FOUND"
		request = "apache.rocketmq.v2.ReceiveMessageRequest.newBuilder().build()"
	}
	b.WriteString(fmt.Sprintf("%s    final apache.rocketmq.v2.Status status = apache.rocketmq.v2.Status.newBuilder()\n", indent))
	b.WriteString(fmt.Sprintf("%s            .setCode(%s)\n", indent, code))
	b.WriteString(fmt.Sprintf("%s            .build();\n", indent))
	b.WriteString(fmt.Sprintf("%s    final Object request = %s;\n", indent, request))
	b.WriteString(fmt.Sprintf("%s    final io.grpc.Metadata metadata = new io.grpc.Metadata();\n", indent))
	b.WriteString(fmt.Sprintf("%s    metadata.put(\n", indent))
	b.WriteString(fmt.Sprintf("%s            io.grpc.Metadata.Key.of(\n", indent))
	b.WriteString(fmt.Sprintf("%s                    org.apache.rocketmq.client.java.rpc.Signature.REQUEST_ID_KEY,\n", indent))
	b.WriteString(fmt.Sprintf("%s                    io.grpc.Metadata.ASCII_STRING_MARSHALLER),\n", indent))
	b.WriteString(fmt.Sprintf("%s            \"request-id\");\n", indent))
	b.WriteString(fmt.Sprintf("%s    final org.apache.rocketmq.client.java.rpc.Context context =\n", indent))
	b.WriteString(fmt.Sprintf("%s            new org.apache.rocketmq.client.java.rpc.Context(\n", indent))
	b.WriteString(fmt.Sprintf("%s                    new org.apache.rocketmq.client.java.route.Endpoints(\"127.0.0.1:80\"), metadata);\n", indent))
	b.WriteString(fmt.Sprintf("%s    final org.apache.rocketmq.client.java.rpc.RpcFuture<Object, Object> future =\n", indent))
	b.WriteString(fmt.Sprintf("%s            new org.apache.rocketmq.client.java.rpc.RpcFuture<>(context, request,\n", indent))
	b.WriteString(fmt.Sprintf("%s                    com.google.common.util.concurrent.Futures.immediateFuture(new Object()));\n", indent))
	b.WriteString(fmt.Sprintf("%s    try {\n", indent))
	b.WriteString(fmt.Sprintf("%s        StatusChecker.check(status, future);\n", indent))
	b.WriteString(fmt.Sprintf("%s    } catch (org.apache.rocketmq.client.apis.ClientException e) {\n", indent))
	b.WriteString(fmt.Sprintf("%s        %s.fail(e.getMessage());\n", indent, assertions))
	b.WriteString(fmt.Sprintf("%s    }\n", indent))
	return true
}

func javaWriteInflightRequestCountInterceptorTask(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, assertions string, indent string) bool {
	if m.ClassName != "InflightRequestCountInterceptor" || len(m.Params) != 2 || !javaTaskMentions(task, "context.getMessageHookPoints") {
		return false
	}
	switch m.Name {
	case "doBefore":
		b.WriteString(fmt.Sprintf("%s    InflightRequestCountInterceptor instance = new InflightRequestCountInterceptor();\n", indent))
		b.WriteString(fmt.Sprintf("%s    MessageInterceptorContext context = new MessageInterceptorContextImpl(MessageHookPoints.RECEIVE);\n", indent))
		b.WriteString(fmt.Sprintf("%s    instance.doBefore(context, java.util.Collections.emptyList());\n", indent))
		b.WriteString(fmt.Sprintf("%s    %s.assertEquals(1L, instance.getInflightReceiveRequestCount());\n", indent, assertions))
		return true
	case "doAfter":
		b.WriteString(fmt.Sprintf("%s    InflightRequestCountInterceptor instance = new InflightRequestCountInterceptor();\n", indent))
		b.WriteString(fmt.Sprintf("%s    MessageInterceptorContext context = new MessageInterceptorContextImpl(MessageHookPoints.RECEIVE);\n", indent))
		b.WriteString(fmt.Sprintf("%s    instance.doBefore(context, java.util.Collections.emptyList());\n", indent))
		b.WriteString(fmt.Sprintf("%s    %s.assertEquals(1L, instance.getInflightReceiveRequestCount());\n", indent, assertions))
		b.WriteString(fmt.Sprintf("%s    instance.doAfter(context, java.util.Collections.emptyList());\n", indent))
		b.WriteString(fmt.Sprintf("%s    %s.assertEquals(0L, instance.getInflightReceiveRequestCount());\n", indent, assertions))
		return true
	default:
		return false
	}
}

func javaWriteCompositedMessageInterceptorTask(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, assertions string, indent string) bool {
	if m.ClassName != "CompositedMessageInterceptor" || len(m.Params) != 2 || !javaTaskMentions(task, "context0 instanceof MessageInterceptorContextImpl") {
		return false
	}
	switch m.Name {
	case "doBefore":
		javaWriteCompositedInterceptorSetup(b, indent)
		b.WriteString(fmt.Sprintf("%s    instance.doBefore(context, java.util.Collections.emptyList());\n", indent))
		b.WriteString(fmt.Sprintf("%s    %s.assertTrue(called[0]);\n", indent, assertions))
		return true
	case "doAfter":
		javaWriteCompositedInterceptorSetup(b, indent)
		b.WriteString(fmt.Sprintf("%s    instance.doBefore(context, java.util.Collections.emptyList());\n", indent))
		b.WriteString(fmt.Sprintf("%s    instance.doAfter(context, java.util.Collections.emptyList());\n", indent))
		b.WriteString(fmt.Sprintf("%s    %s.assertTrue(called[1]);\n", indent, assertions))
		return true
	default:
		return false
	}
}

func javaWriteCompositedInterceptorSetup(b *strings.Builder, indent string) {
	b.WriteString(fmt.Sprintf("%s    final boolean[] called = new boolean[2];\n", indent))
	b.WriteString(fmt.Sprintf("%s    MessageInterceptor interceptor = new MessageInterceptor() {\n", indent))
	b.WriteString(fmt.Sprintf("%s        @Override\n", indent))
	b.WriteString(fmt.Sprintf("%s        public void doBefore(\n", indent))
	b.WriteString(fmt.Sprintf("%s                MessageInterceptorContext context,\n", indent))
	b.WriteString(fmt.Sprintf("%s                java.util.List<org.apache.rocketmq.client.java.message.GeneralMessage> messages) {\n", indent))
	b.WriteString(fmt.Sprintf("%s            called[0] = true;\n", indent))
	b.WriteString(fmt.Sprintf("%s        }\n", indent))
	b.WriteString(fmt.Sprintf("\n%s        @Override\n", indent))
	b.WriteString(fmt.Sprintf("%s        public void doAfter(\n", indent))
	b.WriteString(fmt.Sprintf("%s                MessageInterceptorContext context,\n", indent))
	b.WriteString(fmt.Sprintf("%s                java.util.List<org.apache.rocketmq.client.java.message.GeneralMessage> messages) {\n", indent))
	b.WriteString(fmt.Sprintf("%s            called[1] = true;\n", indent))
	b.WriteString(fmt.Sprintf("%s        }\n", indent))
	b.WriteString(fmt.Sprintf("%s    };\n", indent))
	b.WriteString(fmt.Sprintf("%s    CompositedMessageInterceptor instance =\n", indent))
	b.WriteString(fmt.Sprintf("%s            new CompositedMessageInterceptor(java.util.Collections.singletonList(interceptor));\n", indent))
	b.WriteString(fmt.Sprintf("%s    MessageInterceptorContextImpl context = new MessageInterceptorContextImpl(\n", indent))
	b.WriteString(fmt.Sprintf("%s            MessageHookPoints.RECEIVE, MessageHookPointsStatus.OK);\n", indent))
}

func javaWriteConsumeTaskCallTask(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, assertions string, indent string) bool {
	if !javaNeedsRocketMQTestBase(m, task) {
		return false
	}
	if m.ClassName != "ConsumeTask" {
		return false
	}
	b.WriteString(fmt.Sprintf("%s    ClientId clientId = new ClientId();\n", indent))
	b.WriteString(fmt.Sprintf("%s    final MessageViewImpl messageView = fakeMessageViewImpl();\n", indent))
	b.WriteString(fmt.Sprintf("%s    final MessageListener messageListener = message -> ConsumeResult.FAILURE;\n", indent))
	b.WriteString(fmt.Sprintf("%s    final MessageInterceptor messageInterceptor =\n", indent))
	b.WriteString(fmt.Sprintf("%s            org.mockito.Mockito.mock(MessageInterceptor.class);\n", indent))
	b.WriteString(fmt.Sprintf("%s    final ConsumeTask instance = new ConsumeTask(\n", indent))
	b.WriteString(fmt.Sprintf("%s            clientId, messageListener, messageView, messageInterceptor);\n", indent))
	b.WriteString(fmt.Sprintf("%s    final ConsumeResult result = instance.call();\n", indent))
	b.WriteString(fmt.Sprintf("%s    %s.assertEquals(ConsumeResult.FAILURE, result);\n", indent, assertions))
	return true
}

func javaWriteConsumeServiceConsumeTask(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, assertions string, indent string) bool {
	if !javaNeedsRocketMQTestBase(m, task) || m.ClassName != "ConsumeService" {
		return false
	}
	b.WriteString(fmt.Sprintf("%s    ClientId clientId = new ClientId();\n", indent))
	b.WriteString(fmt.Sprintf("%s    final MessageInterceptor interceptor =\n", indent))
	b.WriteString(fmt.Sprintf("%s            org.mockito.Mockito.mock(MessageInterceptor.class);\n", indent))
	b.WriteString(fmt.Sprintf("%s    final ThreadPoolExecutor consumptionExecutor = new ThreadPoolExecutor(\n", indent))
	b.WriteString(fmt.Sprintf("%s            1, 1, 0L, TimeUnit.MILLISECONDS,\n", indent))
	b.WriteString(fmt.Sprintf("%s            new java.util.concurrent.LinkedBlockingQueue<>(),\n", indent))
	b.WriteString(fmt.Sprintf("%s            new org.apache.rocketmq.client.java.misc.ThreadFactoryImpl(\n", indent))
	b.WriteString(fmt.Sprintf("%s                    \"TestMessageConsumption\"));\n", indent))
	b.WriteString(fmt.Sprintf("%s    final ScheduledExecutorService scheduler =\n", indent))
	b.WriteString(fmt.Sprintf("%s            new java.util.concurrent.ScheduledThreadPoolExecutor(\n", indent))
	b.WriteString(fmt.Sprintf("%s                    1, new org.apache.rocketmq.client.java.misc.ThreadFactoryImpl(\n", indent))
	b.WriteString(fmt.Sprintf("%s                            \"TestScheduler\"));\n", indent))
	b.WriteString(fmt.Sprintf("%s    final MessageListener messageListener = messageView -> ConsumeResult.SUCCESS;\n", indent))
	b.WriteString(fmt.Sprintf("%s    final ConsumeService instance = new ConsumeService(\n", indent))
	b.WriteString(fmt.Sprintf("%s            clientId, messageListener, consumptionExecutor, interceptor, scheduler) {\n", indent))
	b.WriteString(fmt.Sprintf("%s        @Override\n", indent))
	b.WriteString(fmt.Sprintf("%s        public void consume(ProcessQueue pq, List<MessageViewImpl> messageViews) {\n", indent))
	b.WriteString(fmt.Sprintf("%s        }\n", indent))
	b.WriteString(fmt.Sprintf("%s    };\n", indent))
	b.WriteString(fmt.Sprintf("%s    final MessageViewImpl messageView = fakeMessageViewImpl();\n", indent))
	b.WriteString(fmt.Sprintf("%s    final ListenableFuture<ConsumeResult> future = instance.consume(messageView);\n", indent))
	b.WriteString(fmt.Sprintf("%s    try {\n", indent))
	b.WriteString(fmt.Sprintf("%s        final ConsumeResult result = future.get(1000, TimeUnit.MILLISECONDS);\n", indent))
	b.WriteString(fmt.Sprintf("%s        %s.assertEquals(ConsumeResult.SUCCESS, result);\n", indent, assertions))
	b.WriteString(fmt.Sprintf("%s    } catch (Exception e) {\n", indent))
	b.WriteString(fmt.Sprintf("%s        %s.fail(e.getMessage());\n", indent, assertions))
	b.WriteString(fmt.Sprintf("%s    } finally {\n", indent))
	b.WriteString(fmt.Sprintf("%s        consumptionExecutor.shutdownNow();\n", indent))
	b.WriteString(fmt.Sprintf("%s        scheduler.shutdownNow();\n", indent))
	b.WriteString(fmt.Sprintf("%s    }\n", indent))
	return true
}

func javaWriteConsumerImplFilterExpressionTask(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, testName string, style javaJUnitStyle) bool {
	if task == nil || m.ClassName != "ConsumerImpl" || m.Name != "wrapFilterExpression" {
		return false
	}
	indent := "    "
	assertions := javaAssertionsQualifier(style)
	lineStart, _, hasLine := javaCoverageTaskLineRange(task)
	useTagPath := hasLine && lineStart >= 253
	filterExpression := "new FilterExpression(\n" + indent + "            \"a > 1\",\n" + indent + "            org.apache.rocketmq.client.apis.consumer.FilterExpressionType.SQL92)"
	expectedFilterType := "apache.rocketmq.v2.FilterType.SQL"
	if useTagPath {
		filterExpression = "new FilterExpression()"
		expectedFilterType = "apache.rocketmq.v2.FilterType.TAG"
	}
	b.WriteString(fmt.Sprintf("\n    @Test\n"))
	b.WriteString(fmt.Sprintf("    public void %s() {\n", testName))
	if comment := coverageTaskComment(task); comment != "" {
		b.WriteString(fmt.Sprintf("%s    // coverage task: %s\n", indent, truncateJavaComment(comment, 88)))
	}
	b.WriteString(fmt.Sprintf("%s    final PushConsumerImpl consumer = new PushConsumerImpl(\n", indent))
	b.WriteString(fmt.Sprintf("%s            ClientConfiguration.newBuilder().setEndpoints(FAKE_ENDPOINTS).build(),\n", indent))
	b.WriteString(fmt.Sprintf("%s            FAKE_CONSUMER_GROUP_0,\n", indent))
	b.WriteString(fmt.Sprintf("%s            createSubscriptionExpressions(FAKE_TOPIC_0),\n", indent))
	b.WriteString(fmt.Sprintf("%s            messageView -> org.apache.rocketmq.client.apis.consumer.ConsumeResult.SUCCESS,\n", indent))
	b.WriteString(fmt.Sprintf("%s            8, 1024, 4);\n", indent))
	b.WriteString(fmt.Sprintf("%s    final FilterExpression filterExpression = %s;\n", indent, filterExpression))
	b.WriteString(fmt.Sprintf("%s    final ReceiveMessageRequest request = consumer.wrapReceiveMessageRequest(\n", indent))
	b.WriteString(fmt.Sprintf("%s            1, fakeMessageQueueImpl(FAKE_TOPIC_0), filterExpression,\n", indent))
	b.WriteString(fmt.Sprintf("%s            Duration.ofSeconds(15), \"attempt-id\");\n", indent))
	b.WriteString(fmt.Sprintf("%s    %s.assertEquals(\n", indent, assertions))
	b.WriteString(fmt.Sprintf("%s            %s,\n", indent, expectedFilterType))
	b.WriteString(fmt.Sprintf("%s            request.getFilterExpression().getType());\n", indent))
	b.WriteString("    }\n")
	return true
}

func javaWriteConsumerImplReceiveMessageRequestTask(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, testName string, style javaJUnitStyle) bool {
	if task == nil || m.ClassName != "ConsumerImpl" || m.Name != "wrapReceiveMessageRequest" {
		return false
	}
	indent := "    "
	assertions := javaAssertionsQualifier(style)
	useInvisibleDurationOverload := len(m.Params) >= 5 && m.Params[3].Name == "invisibleDuration"
	b.WriteString(fmt.Sprintf("\n    @Test\n"))
	b.WriteString(fmt.Sprintf("    public void %s() {\n", testName))
	if comment := coverageTaskComment(task); comment != "" {
		b.WriteString(fmt.Sprintf("%s    // coverage task: %s\n", indent, truncateJavaComment(comment, 88)))
	}
	b.WriteString(fmt.Sprintf("%s    final PushConsumerImpl consumer = new PushConsumerImpl(\n", indent))
	b.WriteString(fmt.Sprintf("%s            ClientConfiguration.newBuilder().setEndpoints(FAKE_ENDPOINTS).build(),\n", indent))
	b.WriteString(fmt.Sprintf("%s            FAKE_CONSUMER_GROUP_0,\n", indent))
	b.WriteString(fmt.Sprintf("%s            createSubscriptionExpressions(FAKE_TOPIC_0),\n", indent))
	b.WriteString(fmt.Sprintf("%s            messageView -> org.apache.rocketmq.client.apis.consumer.ConsumeResult.SUCCESS,\n", indent))
	b.WriteString(fmt.Sprintf("%s            8, 1024, 4);\n", indent))
	if useInvisibleDurationOverload {
		b.WriteString(fmt.Sprintf("%s    final Duration invisibleDuration = Duration.ofSeconds(45);\n", indent))
		b.WriteString(fmt.Sprintf("%s    final ReceiveMessageRequest request = consumer.wrapReceiveMessageRequest(\n", indent))
		b.WriteString(fmt.Sprintf("%s            2, fakeMessageQueueImpl(FAKE_TOPIC_0), new FilterExpression(),\n", indent))
		b.WriteString(fmt.Sprintf("%s            invisibleDuration, Duration.ofSeconds(30));\n", indent))
		b.WriteString(fmt.Sprintf("%s    %s.assertFalse(request.getAutoRenew());\n", indent, assertions))
		b.WriteString(fmt.Sprintf("%s    %s.assertEquals(\n", indent, assertions))
		b.WriteString(fmt.Sprintf("%s            Durations.fromNanos(invisibleDuration.toNanos()),\n", indent))
		b.WriteString(fmt.Sprintf("%s            request.getInvisibleDuration());\n", indent))
	} else {
		b.WriteString(fmt.Sprintf("%s    final ReceiveMessageRequest request = consumer.wrapReceiveMessageRequest(\n", indent))
		b.WriteString(fmt.Sprintf("%s            2, fakeMessageQueueImpl(FAKE_TOPIC_0), new FilterExpression(),\n", indent))
		b.WriteString(fmt.Sprintf("%s            Duration.ofSeconds(30), \"attempt-id\");\n", indent))
		b.WriteString(fmt.Sprintf("%s    %s.assertTrue(request.getAutoRenew());\n", indent, assertions))
		b.WriteString(fmt.Sprintf("%s    %s.assertEquals(\"attempt-id\", request.getAttemptId());\n", indent, assertions))
	}
	b.WriteString(fmt.Sprintf("%s    %s.assertEquals(2, request.getBatchSize());\n", indent, assertions))
	b.WriteString(fmt.Sprintf("%s    %s.assertEquals(\n", indent, assertions))
	b.WriteString(fmt.Sprintf("%s            apache.rocketmq.v2.FilterType.TAG,\n", indent))
	b.WriteString(fmt.Sprintf("%s            request.getFilterExpression().getType());\n", indent))
	b.WriteString("    }\n")
	return true
}

func javaWriteConsumerImplAckMessageTask(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, testName string, style javaJUnitStyle) bool {
	if task == nil || m.ClassName != "ConsumerImpl" || m.Name != "ackMessage" {
		return false
	}
	indent := "    "
	assertions := javaAssertionsQualifier(style)
	javaWriteConsumerImplRpcSetup(b, indent, task, testName)
	b.WriteString(fmt.Sprintf("%s    final RpcFuture<AckMessageRequest, AckMessageResponse> future =\n", indent))
	b.WriteString(fmt.Sprintf("%s            okAckMessageResponseFuture();\n", indent))
	b.WriteString(fmt.Sprintf("%s    org.mockito.Mockito.doReturn(future).when(clientManager).ackMessage(\n", indent))
	b.WriteString(fmt.Sprintf("%s            org.mockito.ArgumentMatchers.any(Endpoints.class),\n", indent))
	b.WriteString(fmt.Sprintf("%s            org.mockito.ArgumentMatchers.any(AckMessageRequest.class),\n", indent))
	b.WriteString(fmt.Sprintf("%s            org.mockito.ArgumentMatchers.any(Duration.class));\n", indent))
	b.WriteString(fmt.Sprintf("%s    final MessageViewImpl messageView = fakeMessageViewImpl();\n", indent))
	b.WriteString(fmt.Sprintf("%s    final RpcFuture<AckMessageRequest, AckMessageResponse> result =\n", indent))
	b.WriteString(fmt.Sprintf("%s            consumer.ackMessage(messageView);\n", indent))
	b.WriteString(fmt.Sprintf("%s    try {\n", indent))
	b.WriteString(fmt.Sprintf("%s        %s.assertEquals(future.get(), result.get());\n", indent, assertions))
	b.WriteString(fmt.Sprintf("%s    } catch (Exception e) {\n", indent))
	b.WriteString(fmt.Sprintf("%s        %s.fail(e.getMessage());\n", indent, assertions))
	b.WriteString(fmt.Sprintf("%s    }\n", indent))
	b.WriteString("    }\n")
	return true
}

func javaWriteConsumerImplChangeInvisibleDurationTask(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, testName string, style javaJUnitStyle) bool {
	if task == nil || m.ClassName != "ConsumerImpl" || m.Name != "changeInvisibleDuration" {
		return false
	}
	indent := "    "
	assertions := javaAssertionsQualifier(style)
	javaWriteConsumerImplRpcSetup(b, indent, task, testName)
	lineStart, _, hasLine := javaCoverageTaskLineRange(task)
	failurePath := hasLine && lineStart >= 217 && lineStart <= 224
	if failurePath {
		b.WriteString(fmt.Sprintf("%s    final RpcFuture<ChangeInvisibleDurationRequest, ChangeInvisibleDurationResponse> future =\n", indent))
		b.WriteString(fmt.Sprintf("%s            new RpcFuture<>(new RuntimeException(\"change invisible failed\"));\n", indent))
	} else if javaTaskMentions(task, "!Code.OK.equals") || (hasLine && lineStart >= 208 && lineStart <= 210) {
		b.WriteString(fmt.Sprintf("%s    final RpcFuture<ChangeInvisibleDurationRequest, ChangeInvisibleDurationResponse> future =\n", indent))
		b.WriteString(fmt.Sprintf("%s            changInvisibleDurationCtxFuture(apache.rocketmq.v2.Code.INTERNAL_SERVER_ERROR);\n", indent))
	} else {
		b.WriteString(fmt.Sprintf("%s    final RpcFuture<ChangeInvisibleDurationRequest, ChangeInvisibleDurationResponse> future =\n", indent))
		b.WriteString(fmt.Sprintf("%s            okChangeInvisibleDurationCtxFuture();\n", indent))
	}
	b.WriteString(fmt.Sprintf("%s    org.mockito.Mockito.doReturn(future).when(clientManager).changeInvisibleDuration(\n", indent))
	b.WriteString(fmt.Sprintf("%s            org.mockito.ArgumentMatchers.any(Endpoints.class),\n", indent))
	b.WriteString(fmt.Sprintf("%s            org.mockito.ArgumentMatchers.any(ChangeInvisibleDurationRequest.class),\n", indent))
	b.WriteString(fmt.Sprintf("%s            org.mockito.ArgumentMatchers.any(Duration.class));\n", indent))
	b.WriteString(fmt.Sprintf("%s    final MessageViewImpl messageView = fakeMessageViewImpl();\n", indent))
	b.WriteString(fmt.Sprintf("%s    final RpcFuture<ChangeInvisibleDurationRequest, ChangeInvisibleDurationResponse> result =\n", indent))
	b.WriteString(fmt.Sprintf("%s            consumer.changeInvisibleDuration(messageView, Duration.ofSeconds(15));\n", indent))
	if failurePath {
		b.WriteString(fmt.Sprintf("%s    %s.assertThrows(Exception.class, () -> result.get());\n", indent, assertions))
	} else {
		b.WriteString(fmt.Sprintf("%s    try {\n", indent))
		b.WriteString(fmt.Sprintf("%s        %s.assertEquals(future.get(), result.get());\n", indent, assertions))
		b.WriteString(fmt.Sprintf("%s    } catch (Exception e) {\n", indent))
		b.WriteString(fmt.Sprintf("%s        %s.fail(e.getMessage());\n", indent, assertions))
		b.WriteString(fmt.Sprintf("%s    }\n", indent))
	}
	b.WriteString("    }\n")
	return true
}

func javaWriteConsumerImplRpcSetup(b *strings.Builder, indent string, task *types.CoverageTestTask, testName string) {
	b.WriteString(fmt.Sprintf("\n    @Test\n"))
	b.WriteString(fmt.Sprintf("    public void %s() {\n", testName))
	if comment := coverageTaskComment(task); comment != "" {
		b.WriteString(fmt.Sprintf("%s    // coverage task: %s\n", indent, truncateJavaComment(comment, 88)))
	}
	b.WriteString(fmt.Sprintf("%s    final PushConsumerImpl consumer = org.mockito.Mockito.spy(new PushConsumerImpl(\n", indent))
	b.WriteString(fmt.Sprintf("%s            ClientConfiguration.newBuilder().setEndpoints(FAKE_ENDPOINTS).build(),\n", indent))
	b.WriteString(fmt.Sprintf("%s            FAKE_CONSUMER_GROUP_0,\n", indent))
	b.WriteString(fmt.Sprintf("%s            createSubscriptionExpressions(FAKE_TOPIC_0),\n", indent))
	b.WriteString(fmt.Sprintf("%s            messageView -> org.apache.rocketmq.client.apis.consumer.ConsumeResult.SUCCESS,\n", indent))
	b.WriteString(fmt.Sprintf("%s            8, 1024, 4));\n", indent))
	b.WriteString(fmt.Sprintf("%s    final ClientManager clientManager = org.mockito.Mockito.mock(ClientManager.class);\n", indent))
	b.WriteString(fmt.Sprintf("%s    org.mockito.Mockito.doReturn(clientManager).when(consumer).getClientManager();\n", indent))
}

func javaWriteEnumMethodTaskAssertion(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, assertions string, indent string) bool {
	if !m.IsEnum || m.ClassName == "" || m.Name == "" || m.ReturnType == "" {
		return false
	}
	constant := javaCoverageTaskEnumConstant(task)
	if constant == "" {
		return false
	}
	b.WriteString(fmt.Sprintf("%s    %s result = %s.%s.%s();\n", indent, m.ReturnType, m.ClassName, constant, m.Name))
	expectedPrefix := m.ReturnType
	b.WriteString(fmt.Sprintf("%s    %s.assertEquals(%s.%s, result);\n", indent, assertions, expectedPrefix, constant))
	return true
}

func javaCoverageTaskEnumConstant(task *types.CoverageTestTask) string {
	if task == nil {
		return ""
	}
	for _, values := range [][]string{task.MissingBranches, task.SuggestedInputs, task.AssertionFocus} {
		for _, value := range values {
			for _, token := range strings.FieldsFunc(value, func(r rune) bool {
				return !(r == '_' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
			}) {
				if token == "" {
					continue
				}
				if token == "PRODUCER" || token == strings.ToUpper(token) && strings.Contains(token, "_") {
					return token
				}
			}
		}
	}
	return ""
}

func javaWriteHashCodeTaskAssertion(b *strings.Builder, m javaFuncInfo, task *types.CoverageTestTask, assertions string, indent string) bool {
	if m.Name != "hashCode" || m.ReturnType != "int" || !javaTaskMentions(task, "hash == 0") {
		return false
	}
	b.WriteString(fmt.Sprintf("%s    int result = instance.hashCode();\n", indent))
	b.WriteString(fmt.Sprintf("%s    %s.assertNotEquals(0, result);\n", indent, assertions))
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

func javaAddGeneratedImports(code string, source []byte) string {
	code = javaRemoveUnusedAssertionImports(code)
	var additions []string
	for _, importLine := range javaReferencedSourceImports(source, code) {
		if !strings.Contains(code, importLine) {
			additions = append(additions, importLine)
		}
	}
	if strings.Contains(code, "extends TestBase") &&
		!strings.Contains(code, "import org.apache.rocketmq.client.java.tool.TestBase;") {
		additions = append(additions, "import org.apache.rocketmq.client.java.tool.TestBase;")
	}
	if strings.Contains(code, "InetSocketAddress") && !strings.Contains(code, "import java.net.InetSocketAddress;") {
		additions = append(additions, "import java.net.InetSocketAddress;")
	}
	if strings.Contains(code, "List<InetSocketAddress>") && !strings.Contains(code, "import java.util.List;") {
		additions = append(additions, "import java.util.List;")
	}
	if len(additions) == 0 {
		return code
	}
	for _, marker := range []string{"import org.junit.", "import org.junit.jupiter."} {
		if idx := strings.Index(code, marker); idx >= 0 {
			return code[:idx] + strings.Join(additions, "\n") + "\n" + code[idx:]
		}
	}
	return code
}

func javaReferencedSourceImports(source []byte, code string) []string {
	imports := javaSourceImportLines(source)
	referenceCode := javaCodeWithoutLineComments(code)
	referenced := make([]string, 0, len(imports))
	for _, importLine := range imports {
		simpleName := javaImportSimpleName(importLine)
		if simpleName == "" {
			continue
		}
		if javaCodeReferencesIdentifier(referenceCode, simpleName) {
			referenced = append(referenced, importLine)
		}
	}
	return referenced
}

func javaCodeWithoutLineComments(code string) string {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "//"); idx >= 0 {
			lines[i] = line[:idx]
		}
	}
	return strings.Join(lines, "\n")
}

func javaSourceImportLines(source []byte) []string {
	var imports []string
	for _, line := range strings.Split(string(source), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "import static ") {
			continue
		}
		if !strings.HasSuffix(line, ";") {
			continue
		}
		imports = append(imports, line)
	}
	return imports
}

func javaImportSimpleName(importLine string) string {
	name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(importLine, "import "), ";"))
	if strings.HasSuffix(name, ".*") || name == "" {
		return ""
	}
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

func javaCodeReferencesIdentifier(code string, identifier string) bool {
	for i := 0; i < len(code); i++ {
		idx := strings.Index(code[i:], identifier)
		if idx < 0 {
			return false
		}
		start := i + idx
		end := start + len(identifier)
		beforeOK := start == 0 || (!isJavaIdentifierPart(rune(code[start-1])) && code[start-1] != '.')
		afterOK := end == len(code) || !isJavaIdentifierPart(rune(code[end]))
		if beforeOK && afterOK {
			return true
		}
		i = end
	}
	return false
}

func isJavaIdentifierPart(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func javaRemoveUnusedAssertionImports(code string) string {
	if !strings.Contains(code, "Assert.") {
		code = strings.ReplaceAll(code, "import org.junit.Assert;\n", "")
	}
	if !strings.Contains(code, "Assertions.") {
		code = strings.ReplaceAll(code, "import org.junit.jupiter.api.Assertions;\n", "")
	}
	return code
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
	for _, name := range []string{"assertEquals", "assertNotEquals", "assertTrue", "assertFalse", "assertNotNull", "assertNull", "assertThrows"} {
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
