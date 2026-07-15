package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sleticalboy/testloop-mcp/types"
)

func TestGenerateRustTestsForCoverageTaskTargetsFunction(t *testing.T) {
	source := []byte(`pub struct Validator;

impl Validator {
    pub fn new() -> Self {
        Validator
    }

    pub fn check(&self, value: i32) -> bool {
        value > 0
    }

    pub fn skip(&self, value: i32) -> bool {
        value < 0
    }
}

pub fn add(a: i32, b: i32) -> i32 {
    a + b
}
`)
	task := types.CoverageTestTask{
		ID:              "rust-1",
		Framework:       "cargo-test",
		Target:          "Validator.check",
		LineRange:       "8-8",
		GapType:         "branch",
		TestName:        "test_validator_check_covers_gap",
		SuggestedInputs: []string{"构造满足条件 `value == 0` 的输入"},
		AssertionFocus:  []string{"未覆盖 match 分支"},
	}

	_, code, err := GenerateRustTestsForCoverageTask(source, "src/lib.rs", &task)
	if err != nil {
		t.Fatalf("GenerateRustTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"fn test_validator_check_covers_gap()",
		"coverage task: rust-1 | lines 8-8 | 未覆盖 match 分支 | 构造满足条件 `value == 0` 的输入",
		"let instance = Validator::new();",
		"let result = instance.check(0);",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "test_add") || strings.Contains(code, "test_skip") {
		t.Fatalf("task-aware Rust generation should only target Validator.check:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskTargetsMethod(t *testing.T) {
	source := []byte(`public class Calculator {
    public int add(int a, int b) {
        return a + b;
    }

    public int sub(int a, int b) {
        return a - b;
    }
}
`)
	task := types.CoverageTestTask{
		ID:              "java-1",
		Framework:       "junit",
		Target:          "Calculator.add",
		LineRange:       "2-2",
		GapType:         "branch",
		TestName:        "shouldCoverCalculatorAddGap",
		SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入"},
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
	}

	_, code, err := GenerateJavaTestsForCoverageTask(source, "Calculator.java", &task)
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"void shouldCoverCalculatorAddGap()",
		"coverage task: java-1 | lines 2-2 | 断言未覆盖分支的返回值或副作用 | 构造满足条件 `a == 0` 的输入",
		"Calculator instance = new Calculator();",
		"int result = instance.add(0, 0);",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "void sub()") || strings.Contains(code, "instance.sub(") {
		t.Fatalf("task-aware Java generation should only target Calculator.add:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskUsesTaskTestFileClassName(t *testing.T) {
	source := []byte(`public class Base64 {
    public byte[] encode(byte[] in) {
        return in;
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "Base64.java", &types.CoverageTestTask{
		ID:        "junit-10",
		Framework: "junit",
		Target:    "Base64.encode",
		LineRange: "3-3",
		TestName:  "shouldCoverBase64EncodeGap",
		TestFile:  filepath.Join("src", "test", "java", "org", "example", "Base64TestLoopTest.java"),
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	if !strings.Contains(code, "public class Base64TestLoopTest") {
		t.Fatalf("expected generated class to match task test file:\n%s", code)
	}
	if strings.Contains(code, "public class Base64Test ") {
		t.Fatalf("generated class should not use source-derived test class when task test file is set:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskUsesNestedJavaClassTarget(t *testing.T) {
	source := []byte(`public class Base64 {
    public static class Builder {
        public Builder setDecodeTableFormat(DecodeTableFormat format) {
            if (format == null) {
                return this;
            }
            switch (format) {
                case STANDARD:
                    return this;
                case URL_SAFE:
                    return this;
                case MIXED:
                default:
                    return this;
            }
        }
    }

    public enum DecodeTableFormat {
        STANDARD,
        URL_SAFE,
        MIXED
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "Base64.java", &types.CoverageTestTask{
		ID:              "junit-38",
		Framework:       "junit",
		Target:          "Base64.Builder.setDecodeTableFormat",
		LineRange:       "141-141",
		TestName:        "shouldCoverBase64BuilderSetDecodeTableFormatGap",
		TestFile:        filepath.Join("src", "test", "java", "org", "example", "Base64TestLoopTest.java"),
		AssertionFocus:  []string{"断言未覆盖返回路径的具体结果"},
		UncoveredLines:  []int{141},
		MissingBranches: []string{"未覆盖返回路径"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"Base64.Builder instance = new Base64.Builder();",
		"Base64.Builder result = instance.setDecodeTableFormat(Base64.DecodeTableFormat.MIXED);",
		"Assertions.assertNotNull(result);",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "Builder instance = new Builder()") || strings.Contains(code, "setDecodeTableFormat(null)") {
		t.Fatalf("nested Base64.Builder task should use qualified class and line-specific enum value:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskHandlesDigestUtilsShakeMethods(t *testing.T) {
	source := []byte(`import java.io.IOException;
import java.io.InputStream;

public class DigestUtils {
    public static byte[] shake128_256(byte[] data) {
        return data;
    }

    public static byte[] shake128_256(InputStream data) throws IOException {
        return data.readAllBytes();
    }

    public static String shake256_512Hex(String data) {
        return data;
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "DigestUtils.java", &types.CoverageTestTask{
		ID:              "junit-43",
		Framework:       "junit",
		Target:          "DigestUtils.shake128_256",
		LineRange:       "5-5",
		TestName:        "shouldCoverDigestUtilsShake128256Gap",
		AssertionFocus:  []string{"断言未覆盖返回路径的具体结果"},
		UncoveredLines:  []int{5},
		MissingBranches: []string{"未覆盖返回路径"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"byte[] result = DigestUtils.shake128_256(new byte[] { 97, 98, 99 });",
		"Assertions.assertTrue(result.length > 0);",
		"} catch (IllegalArgumentException ex) {",
		"Assertions.assertTrue(ex.getMessage().contains(\"SHAKE\"));",
		"} catch (Exception ex) {",
		"Assertions.fail(ex);",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "shake128_256(null)") || strings.Contains(code, "assertThrows(IOException.class") {
		t.Fatalf("SHAKE generation should not emit ambiguous null or empty IOException assertion:\n%s", code)
	}

	_, code, err = GenerateJavaTestsForCoverageTask(source, "DigestUtils.java", &types.CoverageTestTask{
		ID:              "junit-44",
		Framework:       "junit",
		Target:          "DigestUtils.shake128_256",
		LineRange:       "9-9",
		TestName:        "shouldCoverDigestUtilsShake128256InputStreamGap",
		AssertionFocus:  []string{"断言未覆盖返回路径的具体结果"},
		UncoveredLines:  []int{9},
		MissingBranches: []string{"未覆盖返回路径"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	if !strings.Contains(code, "DigestUtils.shake128_256(new java.io.ByteArrayInputStream(new byte[] { 97, 98, 99 }))") {
		t.Fatalf("InputStream SHAKE overload should use typed ByteArrayInputStream input:\n%s", code)
	}
	if strings.Contains(code, "shake128_256(null)") {
		t.Fatalf("InputStream SHAKE overload should not emit ambiguous null:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskHandlesRulePhonemeAndGetInstance(t *testing.T) {
	source := []byte(`import java.util.List;

public class Rule {
    public static final class Phoneme {
        public Phoneme(CharSequence phonemeText, Languages.LanguageSet languages) {
        }

        public Phoneme join(Phoneme right) {
            return right;
        }

        public CharSequence getPhonemeText() {
            return "ab";
        }

        public String toString() {
            return "abc[any]";
        }
    }

    public static List<Rule> getInstance(NameType nameType, RuleType rt, Languages.LanguageSet langs) {
        return java.util.Collections.singletonList(new Rule());
    }

    public static List<Rule> getInstance(NameType nameType, RuleType rt, String lang) {
        return java.util.Collections.singletonList(new Rule());
    }
}

class Languages {
    static final LanguageSet ANY_LANGUAGE = null;
    static class LanguageSet {
        static LanguageSet from(java.util.Set<String> languages) {
            return null;
        }
    }
}

enum NameType {
    GENERIC
}

enum RuleType {
    RULES
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "Rule.java", &types.CoverageTestTask{
		ID:              "junit-78",
		Framework:       "junit",
		Target:          "Rule.Phoneme.join",
		LineRange:       "8-8",
		TestName:        "shouldCoverRulePhonemeJoinGap",
		AssertionFocus:  []string{"断言未覆盖返回路径的具体结果"},
		UncoveredLines:  []int{8},
		MissingBranches: []string{"未覆盖返回路径"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"Rule.Phoneme instance = new Rule.Phoneme(\"a\", Languages.ANY_LANGUAGE);",
		"Rule.Phoneme right = new Rule.Phoneme(\"b\", Languages.ANY_LANGUAGE);",
		"Rule.Phoneme result = instance.join(right);",
		"Assertions.assertEquals(\"ab\", result.getPhonemeText().toString());",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "new Rule.Phoneme()") || strings.Contains(code, "join(null)") {
		t.Fatalf("Rule.Phoneme join should not use no-arg constructor or null right:\n%s", code)
	}

	_, code, err = GenerateJavaTestsForCoverageTask(source, "Rule.java", &types.CoverageTestTask{
		ID:              "junit-80",
		Framework:       "junit",
		Target:          "Rule.getInstance",
		LineRange:       "24-24",
		TestName:        "shouldCoverRuleGetInstanceGap",
		AssertionFocus:  []string{"断言未覆盖返回路径的具体结果"},
		UncoveredLines:  []int{24},
		MissingBranches: []string{"未覆盖返回路径"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	if !strings.Contains(code, "Rule.getInstance(NameType.GENERIC, RuleType.RULES, Languages.LanguageSet.from(new java.util.HashSet<>(java.util.Arrays.asList(\"english\"))))") {
		t.Fatalf("Rule.getInstance LanguageSet overload should use real enum and language inputs:\n%s", code)
	}
	if strings.Contains(code, "Rule.getInstance(null") {
		t.Fatalf("Rule.getInstance should not use null enum inputs:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskCastsNullForOverloadedCollectionConstructor(t *testing.T) {
	source := []byte(`public class JSONArray {
    public JSONArray(String source) {
    }

    public JSONArray(JSONArray array) {
    }

    public JSONArray(Iterable<?> iter) {
        if (iter == null) {
            return;
        }
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "JSONArray.java", &types.CoverageTestTask{
		ID:              "junit-2",
		Framework:       "junit",
		Target:          "JSONArray.JSONArray",
		LineRange:       "8-8",
		TestName:        "shouldCoverJSONArrayJSONArrayGap",
		MissingBranches: []string{"未覆盖 if 分支: iter == null"},
		SuggestedInputs: []string{"构造满足条件 `iter == null` 的输入"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	if !strings.Contains(code, "JSONArray instance = new JSONArray((Iterable<?>) null);") {
		t.Fatalf("overloaded collection constructor should cast null arg:\n%s", code)
	}
	if strings.Contains(code, "new JSONArray(null)") {
		t.Fatalf("overloaded collection constructor should not emit ambiguous null:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskBuildsJSONArrayNumberState(t *testing.T) {
	source := []byte(`public class JSONArray {
    public JSONArray() {
    }

    public JSONArray put(Object value) {
        return this;
    }

    public Number getNumber(int index) throws JSONException {
        Object object = this.get(index);
        if (object instanceof Number) {
            return (Number) object;
        }
        throw new JSONException("bad");
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "JSONArray.java", &types.CoverageTestTask{
		ID:              "junit-3",
		Framework:       "junit",
		Target:          "JSONArray.getNumber",
		LineRange:       "10-10",
		TestName:        "shouldCoverJSONArrayGetNumberGap",
		MissingBranches: []string{"未覆盖 if 分支: object instanceof Number"},
		SuggestedInputs: []string{"构造满足条件 `object instanceof Number` 的输入"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"JSONArray instance = new JSONArray();",
		"instance.put(1);",
		"Number result = instance.getNumber(0);",
		"Assertions.assertEquals(1, result.intValue());",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "Number result = instance.getNumber(0);\n        Assertions.assertNotNull(result);") {
		t.Fatalf("JSONArray number task should not call getter on empty instance:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskBuildsJSONArrayErrorPaths(t *testing.T) {
	source := []byte(`import java.io.Writer;

public class JSONArray {
    public JSONArray() {
    }

    public JSONArray put(Object value) {
        return this;
    }

    public Number optNumber(int index, Number defaultValue) {
        return defaultValue;
    }

    public float getFloat(int index) throws JSONException {
        throw new JSONException("bad");
    }

    public Writer write(Writer writer, int indentFactor, int indent) throws JSONException {
        return writer;
    }
}
`)

	tests := []struct {
		name string
		task types.CoverageTestTask
		want []string
	}{
		{
			name: "optNumber invalid string default",
			task: types.CoverageTestTask{
				ID:             "junit-21",
				Framework:      "junit",
				Target:         "JSONArray.optNumber",
				LineRange:      "11-11",
				TestName:       "shouldCoverJSONArrayOptNumberGap",
				AssertionFocus: []string{"断言错误、异常或空值路径"},
				UncoveredLines: []int{1153},
			},
			want: []string{
				"instance.put(\"not-a-number\");",
				"Number result = instance.optNumber(0, 7);",
				"Assertions.assertEquals(7, result.intValue());",
			},
		},
		{
			name: "getFloat conversion error",
			task: types.CoverageTestTask{
				ID:             "junit-26",
				Framework:      "junit",
				Target:         "JSONArray.getFloat",
				LineRange:      "15-15",
				TestName:       "shouldCoverJSONArrayGetFloatGap",
				AssertionFocus: []string{"断言错误、异常或空值路径"},
				UncoveredLines: []int{400, 401},
			},
			want: []string{
				"instance.put(new Object());",
				"Assertions.assertThrows(JSONException.class, () -> instance.getFloat(0));",
			},
		},
		{
			name: "write IOException wrapper",
			task: types.CoverageTestTask{
				ID:             "junit-23",
				Framework:      "junit",
				Target:         "JSONArray.write",
				LineRange:      "19-19",
				TestName:       "shouldCoverJSONArrayWriteGap",
				AssertionFocus: []string{"断言错误、异常或空值路径"},
				UncoveredLines: []int{1835, 1836},
			},
			want: []string{
				"final java.io.Writer writer = new java.io.Writer()",
				"throw new java.io.IOException(\"boom\");",
				"Assertions.assertThrows(JSONException.class, () -> instance.write(writer, 0, 0));",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, code, err := GenerateJavaTestsForCoverageTask(source, "JSONArray.java", &tt.task)
			if err != nil {
				t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
			}
			for _, want := range tt.want {
				if !strings.Contains(code, want) {
					t.Fatalf("expected %q in generated code:\n%s", want, code)
				}
			}
			if strings.Contains(code, "instance.write(null") || strings.Contains(code, "instance.getFloat(0);\n        Assertions.assertEquals") {
				t.Fatalf("JSONArray error task should not emit empty/default failing call:\n%s", code)
			}
		})
	}
}

func TestGenerateJavaTestsForCoverageTaskBuildsXMLToJSONObjectFlags(t *testing.T) {
	source := []byte(`import java.io.Reader;

public class XML {
    public static JSONObject toJSONObject(Reader reader, boolean keepNumberAsString, boolean keepBooleanAsString) throws JSONException {
        if (keepNumberAsString) {
            return new JSONObject();
        }
        if (keepBooleanAsString) {
            return new JSONObject();
        }
        return new JSONObject();
    }
}
`)

	tests := []struct {
		name string
		task types.CoverageTestTask
		want []string
	}{
		{
			name: "keep number as string",
			task: types.CoverageTestTask{
				ID:              "junit-14",
				Framework:       "junit",
				Target:          "XML.toJSONObject",
				LineRange:       "5-5",
				TestName:        "shouldCoverXMLToJSONObjectGap",
				MissingBranches: []string{"未覆盖 if 分支: keepNumberAsString"},
			},
			want: []string{
				"final java.io.Reader reader = new java.io.StringReader(\"<root>42</root>\");",
				"JSONObject result = XML.toJSONObject(reader, true, false);",
				"Assertions.assertEquals(\"42\", result.get(\"root\"));",
			},
		},
		{
			name: "keep boolean as string",
			task: types.CoverageTestTask{
				ID:              "junit-15",
				Framework:       "junit",
				Target:          "XML.toJSONObject",
				LineRange:       "8-8",
				TestName:        "shouldCoverXMLToJSONObjectGap",
				MissingBranches: []string{"未覆盖 if 分支: keepBooleanAsString"},
			},
			want: []string{
				"final java.io.Reader reader = new java.io.StringReader(\"<root>true</root>\");",
				"JSONObject result = XML.toJSONObject(reader, false, true);",
				"Assertions.assertEquals(\"true\", result.get(\"root\"));",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, code, err := GenerateJavaTestsForCoverageTask(source, "XML.java", &tt.task)
			if err != nil {
				t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
			}
			for _, want := range tt.want {
				if !strings.Contains(code, want) {
					t.Fatalf("expected %q in generated code:\n%s", want, code)
				}
			}
			if strings.Contains(code, "XML.toJSONObject(null") {
				t.Fatalf("XML.toJSONObject task should not use null XML input:\n%s", code)
			}
		})
	}
}

func TestGenerateJavaTestsForCoverageTaskBuildsXMLNoSpaceErrorPath(t *testing.T) {
	source := []byte(`public class XML {
    public static void noSpace(String string) throws JSONException {
        if (string.length() == 0) {
            throw new JSONException("Empty string.");
        }
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "XML.java", &types.CoverageTestTask{
		ID:             "junit-82",
		Framework:      "junit",
		Target:         "XML.noSpace",
		LineRange:      "3-3",
		TestName:       "shouldCoverXMLNoSpaceGap",
		AssertionFocus: []string{"断言错误、异常或空值路径"},
		UncoveredLines: []int{225},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	if !strings.Contains(code, "Assertions.assertThrows(JSONException.class, () -> XML.noSpace(\"\"));") {
		t.Fatalf("XML.noSpace empty string task should assert exception:\n%s", code)
	}
	if strings.Contains(code, "XML.noSpace(\"test\");") {
		t.Fatalf("XML.noSpace error task should not use valid default string:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskUsesPackageAndJUnit4ProjectStyle(t *testing.T) {
	root := t.TempDir()
	srcPath := filepath.Join(root, "client", "src", "main", "java", "com", "example", "Calculator.java")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	pomPath := filepath.Join(root, "client", "pom.xml")
	if err := os.WriteFile(pomPath, []byte(`<project>
  <dependencies>
    <dependency>
      <groupId>junit</groupId>
      <artifactId>junit</artifactId>
      <version>4.13.2</version>
      <scope>test</scope>
    </dependency>
  </dependencies>
</project>
`), 0o644); err != nil {
		t.Fatalf("write pom: %v", err)
	}
	source := []byte(`package com.example;

public class Calculator {
    public int add(int a, int b) {
        return a + b;
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, srcPath, &types.CoverageTestTask{
		Target:   "Calculator.add",
		TestName: "should cover add",
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"package com.example;",
		"import org.junit.Assert;",
		"import org.junit.Test;",
		"void shouldcoveradd()",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "org.junit.jupiter") || strings.Contains(code, "import static org.junit.Assert.*;") {
		t.Fatalf("JUnit 4 project should not use Jupiter imports:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskUsesEqualsHintForConstructorExceptionBranch(t *testing.T) {
	source := []byte(`package com.example;

import java.util.List;

public class Endpoints {
    public Endpoints(AddressScheme scheme, List<Address> addresses) {
        if (AddressScheme.DOMAIN_NAME.equals(scheme) && addresses.size() > 1) {
            throw new UnsupportedOperationException("Multiple addresses not allowed");
        }
    }
}
`)
	task := types.CoverageTestTask{
		Target:          "Endpoints.Endpoints",
		LineRange:       "7-7",
		TestName:        "shouldCoverEndpointsEndpointsGap",
		MissingBranches: []string{"未覆盖 if 分支: AddressScheme.DOMAIN_NAME.equals(scheme"},
		SuggestedInputs: []string{"设置 scheme 覆盖未执行分支", "设置 addresses 覆盖未执行分支"},
	}

	_, code, err := GenerateJavaTestsForCoverageTask(source, "Endpoints.java", &task)
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"final java.util.List<Address> addresses = java.util.Arrays.asList(",
		"new Address(\"example.com\", 80), new Address(\"example.org\", 81));",
		"Assertions.assertThrows(RuntimeException.class, () ->",
		"new Endpoints(AddressScheme.DOMAIN_NAME, addresses));",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "new AddressScheme()") || strings.Contains(code, "Collections.emptyList()") {
		t.Fatalf("constructor branch should use coverage task values:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskUsesEmptyAddressConstructorBranch(t *testing.T) {
	source := []byte(`package com.example;

import java.util.List;

public class Endpoints {
    public Endpoints(AddressScheme scheme, List<Address> addresses) {
        if (addresses.isEmpty()) {
            throw new UnsupportedOperationException("No available address");
        }
    }
}
`)
	task := types.CoverageTestTask{
		Target:          "Endpoints.Endpoints",
		LineRange:       "7-7",
		TestName:        "shouldCoverEndpointsEndpointsGap",
		MissingBranches: []string{"未覆盖 if 分支: addresses.isEmpty"},
		SuggestedInputs: []string{"设置 scheme 覆盖未执行分支", "设置 addresses 覆盖未执行分支"},
	}

	_, code, err := GenerateJavaTestsForCoverageTask(source, "Endpoints.java", &task)
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"final java.util.List<Address> addresses = java.util.Collections.emptyList();",
		"Assertions.assertThrows(RuntimeException.class, () ->",
		"new Endpoints(AddressScheme.IPv4, addresses));",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "new AddressScheme()") || strings.Contains(code, "new Address(\"example.com\"") {
		t.Fatalf("empty address branch should use enum constant and empty list:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskTargetsGetterAndEquals(t *testing.T) {
	source := []byte(`public class Endpoints {
    private final String facade;

    public Endpoints(String endpoints) {
        this.facade = endpoints;
    }

    public String getGrpcTarget() {
        return facade;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) {
            return true;
        }
        return false;
    }
}
`)

	_, getterCode, err := GenerateJavaTestsForCoverageTask(source, "Endpoints.java", &types.CoverageTestTask{
		Target:          "Endpoints.getGrpcTarget",
		LineRange:       "8-8",
		TestName:        "shouldCoverEndpointsGetGrpcTargetGap",
		MissingBranches: []string{"未覆盖 if 分支: AddressScheme.DOMAIN_NAME.equals(scheme"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(getter) error = %v", err)
	}
	for _, want := range []string{
		"Endpoints instance = new Endpoints(\"example.com:80\");",
		"String result = instance.getGrpcTarget();",
	} {
		if !strings.Contains(getterCode, want) {
			t.Fatalf("expected %q in getter code:\n%s", want, getterCode)
		}
	}
	if strings.Contains(getterCode, "instance.equals(") {
		t.Fatalf("getter coverage task should not fall back to all helpers:\n%s", getterCode)
	}

	_, equalsCode, err := GenerateJavaTestsForCoverageTask(source, "Endpoints.java", &types.CoverageTestTask{
		Target:          "Endpoints.equals",
		LineRange:       "13-13",
		TestName:        "shouldCoverEndpointsEqualsGap",
		MissingBranches: []string{"未覆盖 if 分支: this == o"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(equals) error = %v", err)
	}
	for _, want := range []string{
		"Endpoints instance = new Endpoints(\"127.0.0.1:80\");",
		"Assertions.assertTrue(instance.equals(instance));",
	} {
		if !strings.Contains(equalsCode, want) {
			t.Fatalf("expected %q in equals code:\n%s", want, equalsCode)
		}
	}
	if strings.Contains(equalsCode, "getGrpcTarget") {
		t.Fatalf("equals coverage task should not fall back to all helpers:\n%s", equalsCode)
	}
}

func TestGenerateJavaTestsForCoverageTaskHandlesHashCodeAndProtobufConstructors(t *testing.T) {
	source := []byte(`public class Endpoints {
    private int hash;

    public Endpoints(apache.rocketmq.v2.Endpoints endpoints) {
    }

    public int hashCode() {
        if (hash == 0) {
            hash = 1;
        }
        return hash;
    }
}
`)

	_, hashCode, err := GenerateJavaTestsForCoverageTask(source, "Endpoints.java", &types.CoverageTestTask{
		Target:          "Endpoints.hashCode",
		LineRange:       "8-8",
		TestName:        "shouldCoverEndpointsHashCodeGap",
		MissingBranches: []string{"未覆盖 if 分支: hash == 0"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(hashCode) error = %v", err)
	}
	for _, want := range []string{
		"int result = instance.hashCode();",
		"Assertions.assertNotEquals(0, result);",
	} {
		if !strings.Contains(hashCode, want) {
			t.Fatalf("expected %q in hashCode task:\n%s", want, hashCode)
		}
	}
	if strings.Contains(hashCode, "Assertions.assertEquals(0, result);") {
		t.Fatalf("hashCode branch should not assert the initial zero value:\n%s", hashCode)
	}

	_, emptyCode, err := GenerateJavaTestsForCoverageTask(source, "Endpoints.java", &types.CoverageTestTask{
		Target:          "Endpoints.Endpoints",
		LineRange:       "4-4",
		TestName:        "shouldCoverEndpointsEndpointsGap",
		MissingBranches: []string{"未覆盖 if 分支: addresses.isEmpty"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(empty protobuf) error = %v", err)
	}
	for _, want := range []string{
		"final apache.rocketmq.v2.Endpoints endpoints = apache.rocketmq.v2.Endpoints.newBuilder()",
		".setScheme(apache.rocketmq.v2.AddressScheme.IPv4)",
		".build();",
		"Assertions.assertThrows(RuntimeException.class, () -> new Endpoints(endpoints));",
	} {
		if !strings.Contains(emptyCode, want) {
			t.Fatalf("expected %q in empty protobuf task:\n%s", want, emptyCode)
		}
	}

	_, switchCode, err := GenerateJavaTestsForCoverageTask(source, "Endpoints.java", &types.CoverageTestTask{
		Target:          "Endpoints.Endpoints",
		LineRange:       "4-4",
		TestName:        "shouldCoverEndpointsEndpointsGap",
		MissingBranches: []string{"未覆盖 switch/case 分支"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(switch protobuf) error = %v", err)
	}
	for _, want := range []string{
		".setScheme(apache.rocketmq.v2.AddressScheme.IPv4)",
		".addAddresses(apache.rocketmq.v2.Address.newBuilder().setHost(\"127.0.0.1\").setPort(80))",
		"Endpoints instance = new Endpoints(endpoints);",
	} {
		if !strings.Contains(switchCode, want) {
			t.Fatalf("expected %q in switch protobuf task:\n%s", want, switchCode)
		}
	}

	_, sizeCode, err := GenerateJavaTestsForCoverageTask(source, "Endpoints.java", &types.CoverageTestTask{
		Target:          "Endpoints.Endpoints",
		LineRange:       "4-4",
		TestName:        "shouldCoverEndpointsEndpointsGap",
		MissingBranches: []string{"未覆盖 if 分支: addresses.size"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(size protobuf) error = %v", err)
	}
	for _, want := range []string{
		".setScheme(apache.rocketmq.v2.AddressScheme.DOMAIN_NAME)",
		".addAddresses(apache.rocketmq.v2.Address.newBuilder().setHost(\"a.example\").setPort(80))",
		".addAddresses(apache.rocketmq.v2.Address.newBuilder().setHost(\"b.example\").setPort(81))",
		"Assertions.assertThrows(RuntimeException.class, () -> new Endpoints(endpoints));",
	} {
		if !strings.Contains(sizeCode, want) {
			t.Fatalf("expected %q in size protobuf task:\n%s", want, sizeCode)
		}
	}
}

func TestGenerateJavaTestsForCoverageTaskUsesNullAddressListForErrorPath(t *testing.T) {
	source := []byte(`package com.example;

import java.util.List;

public class Endpoints {
    public Endpoints(AddressScheme scheme, List<Address> addresses) {
        if (addresses == null) {
            throw new NullPointerException("addresses");
        }
    }
}
`)
	task := types.CoverageTestTask{
		Target:          "Endpoints.Endpoints",
		LineRange:       "7-7",
		TestName:        "shouldCoverEndpointsEndpointsGap",
		MissingBranches: []string{"未覆盖错误或空值返回路径"},
		SuggestedInputs: []string{"设置 scheme 覆盖未执行分支", "设置 addresses 覆盖未执行分支"},
	}

	_, code, err := GenerateJavaTestsForCoverageTask(source, "Endpoints.java", &task)
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"final java.util.List<Address> addresses = null;",
		"Assertions.assertThrows(RuntimeException.class, () ->",
		"new Endpoints(AddressScheme.IPv4, addresses));",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "new AddressScheme()") || strings.Contains(code, "java.util.Arrays.asList") {
		t.Fatalf("null-address branch should use enum constant and null list:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskDisambiguatesProtobufEndpointErrorLines(t *testing.T) {
	source := []byte(`public class Endpoints {
    public Endpoints(apache.rocketmq.v2.Endpoints endpoints) {
        if (addresses.isEmpty()) {
            throw new UnsupportedOperationException("No available address");
        }
    }

    public Endpoints(String endpoints) {
    }
}
`)
	task := types.CoverageTestTask{
		Target:          "Endpoints.Endpoints",
		LineRange:       "3-3",
		TestName:        "shouldCoverEndpointsEndpointsGap",
		MissingBranches: []string{"未覆盖错误或空值返回路径"},
		SuggestedInputs: []string{"设置 endpoints 覆盖未执行分支"},
	}

	_, code, err := GenerateJavaTestsForCoverageTask(source, "Endpoints.java", &task)
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"final apache.rocketmq.v2.Endpoints endpoints = apache.rocketmq.v2.Endpoints.newBuilder()",
		".setScheme(apache.rocketmq.v2.AddressScheme.IPv4)",
		"Assertions.assertThrows(RuntimeException.class, () -> new Endpoints(endpoints));",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "new Endpoints(null)") {
		t.Fatalf("protobuf constructor task should not emit ambiguous null overload call:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskUsesLineRangeForEqualsBranches(t *testing.T) {
	source := []byte(`public class Endpoints {
    public Endpoints(String endpoints) {
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) {
            return true;
        }
        if (o == null || getClass() != o.getClass()) {
            return false;
        }
        Endpoints endpoints = (Endpoints) o;
        return true;
    }
}
`)

	tests := []struct {
		lineRange string
		want      string
	}{
		{lineRange: "8-8", want: "Assertions.assertTrue(instance.equals(instance));"},
		{lineRange: "11-11", want: "Assertions.assertFalse(instance.equals(new Object()));"},
		{lineRange: "14-14", want: "Endpoints other = new Endpoints(\"127.0.0.1:80\");"},
	}
	for _, tt := range tests {
		_, code, err := GenerateJavaTestsForCoverageTask(source, "Endpoints.java", &types.CoverageTestTask{
			Target:          "Endpoints.equals",
			LineRange:       tt.lineRange,
			TestName:        "shouldCoverEndpointsEqualsGap",
			MissingBranches: []string{"未覆盖返回路径"},
		})
		if err != nil {
			t.Fatalf("GenerateJavaTestsForCoverageTask(%s) error = %v", tt.lineRange, err)
		}
		if !strings.Contains(code, tt.want) {
			t.Fatalf("expected %q in generated code for %s:\n%s", tt.want, tt.lineRange, code)
		}
	}
}

func TestGenerateJavaTestsForCoverageTaskHandlesToSocketAddressesBranches(t *testing.T) {
	source := []byte(`public class Endpoints {
    public Endpoints(String endpoints) {
    }

    public List<InetSocketAddress> toSocketAddresses() {
        switch (scheme) {
            case DOMAIN_NAME:
                return null;
            default:
                return new ArrayList<>();
        }
    }
}
`)
	_, switchCode, err := GenerateJavaTestsForCoverageTask(source, "Endpoints.java", &types.CoverageTestTask{
		Target:          "Endpoints.toSocketAddresses",
		LineRange:       "6-6",
		TestName:        "shouldCoverEndpointsToSocketAddressesGap",
		MissingBranches: []string{"未覆盖 switch/case 分支"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(switch) error = %v", err)
	}
	for _, want := range []string{
		"import java.util.List;",
		"import java.net.InetSocketAddress;",
		"List<InetSocketAddress> result = instance.toSocketAddresses();",
		"Assertions.assertFalse(result.isEmpty());",
	} {
		if !strings.Contains(switchCode, want) {
			t.Fatalf("expected %q in switch code:\n%s", want, switchCode)
		}
	}

	_, nullCode, err := GenerateJavaTestsForCoverageTask(source, "Endpoints.java", &types.CoverageTestTask{
		Target:          "Endpoints.toSocketAddresses",
		LineRange:       "8-8",
		TestName:        "shouldCoverEndpointsToSocketAddressesGap",
		MissingBranches: []string{"未覆盖错误或空值返回路径"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(null) error = %v", err)
	}
	for _, want := range []string{
		"Endpoints domainInstance = new Endpoints(\"example.com:80\");",
		"List<InetSocketAddress> result = domainInstance.toSocketAddresses();",
		"Assertions.assertNull(result);",
	} {
		if !strings.Contains(nullCode, want) {
			t.Fatalf("expected %q in null code:\n%s", want, nullCode)
		}
	}
}

func TestGenerateJavaTestsForCoverageTaskSplitsAddressListConstructorSuccess(t *testing.T) {
	source := []byte(`package com.example;

import java.util.List;

public class Endpoints {
    public Endpoints(AddressScheme scheme, List<Address> addresses) {
        checkNotNull(addresses, "addresses");
    }
}
`)
	task := types.CoverageTestTask{
		Target:          "Endpoints.Endpoints",
		LineRange:       "6-6",
		TestName:        "shouldCoverEndpointsEndpointsGap",
		MissingBranches: []string{"未覆盖普通语句块"},
		SuggestedInputs: []string{"设置 scheme 覆盖未执行分支", "设置 addresses 覆盖未执行分支"},
	}

	_, code, err := GenerateJavaTestsForCoverageTask(source, "Endpoints.java", &task)
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"final java.util.List<Address> addresses = java.util.Arrays.asList(",
		"new Address(\"127.0.0.1\", 80),",
		"new Address(\"127.0.0.2\", 81));",
		"Endpoints instance = new Endpoints(AddressScheme.IPv4, addresses);",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "new Endpoints(new AddressScheme()") {
		t.Fatalf("constructor success branch should use enum constant and split addresses:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskHandlesStatusCheckerCheck(t *testing.T) {
	source := []byte(`package org.apache.rocketmq.client.java.exception;

import apache.rocketmq.v2.Status;
import org.apache.rocketmq.client.apis.ClientException;
import org.apache.rocketmq.client.java.rpc.RpcFuture;

public class StatusChecker {
    public static void check(Status status, RpcFuture<?, ?> future) throws ClientException {
        switch (status.getCode()) {
            case OK:
                return;
            case MESSAGE_NOT_FOUND:
                if (future.getRequest() instanceof apache.rocketmq.v2.ReceiveMessageRequest) {
                    return;
                }
            default:
                throw new ClientException("failed");
        }
    }
}
`)

	_, okCode, err := GenerateJavaTestsForCoverageTask(source, "StatusChecker.java", &types.CoverageTestTask{
		Target:          "StatusChecker.check",
		LineRange:       "9-9",
		TestName:        "shouldCoverStatusCheckerCheckGap",
		MissingBranches: []string{"未覆盖 switch/case 分支"},
		SuggestedInputs: []string{"设置 status 覆盖未执行分支", "设置 future 覆盖未执行分支"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(ok) error = %v", err)
	}
	for _, want := range []string{
		".setCode(apache.rocketmq.v2.Code.OK)",
		"final Object request = new Object();",
		"new org.apache.rocketmq.client.java.rpc.Context(",
		"new org.apache.rocketmq.client.java.rpc.RpcFuture<>(context, request,",
		"StatusChecker.check(status, future);",
		"Assertions.fail(e.getMessage());",
	} {
		if !strings.Contains(okCode, want) {
			t.Fatalf("expected %q in OK generated code:\n%s", want, okCode)
		}
	}
	if strings.Contains(okCode, "new Status()") || strings.Contains(okCode, "new RpcFuture") && strings.Contains(okCode, "new RpcFuture<?, ?>()") {
		t.Fatalf("StatusChecker task should not use invalid default constructors:\n%s", okCode)
	}

	_, receiveCode, err := GenerateJavaTestsForCoverageTask(source, "StatusChecker.java", &types.CoverageTestTask{
		Target:          "StatusChecker.check",
		LineRange:       "13-13",
		TestName:        "shouldCoverStatusCheckerCheckGap",
		MissingBranches: []string{"未覆盖 if 分支: future.getRequest"},
		SuggestedInputs: []string{"构造满足条件 `future.getRequest` 的输入"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(receive) error = %v", err)
	}
	for _, want := range []string{
		".setCode(apache.rocketmq.v2.Code.MESSAGE_NOT_FOUND)",
		"final Object request = apache.rocketmq.v2.ReceiveMessageRequest.newBuilder().build();",
		"StatusChecker.check(status, future);",
	} {
		if !strings.Contains(receiveCode, want) {
			t.Fatalf("expected %q in receive generated code:\n%s", want, receiveCode)
		}
	}
}

func TestGenerateJavaTestsForCoverageTaskUsesStaticFactoryForPrivateConstructor(t *testing.T) {
	source := []byte(`public class AttributeKey<T> {
    private AttributeKey(String name) {
    }

    public static <T> AttributeKey<T> create(String name) {
        return new AttributeKey<>(name);
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) {
            return true;
        }
        if (o == null || getClass() != o.getClass()) {
            return false;
        }
        return true;
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "AttributeKey.java", &types.CoverageTestTask{
		Target:          "AttributeKey.equals",
		LineRange:       "11-11",
		TestName:        "shouldCoverAttributeKeyEqualsGap",
		MissingBranches: []string{"未覆盖 if 分支: this == o"},
		SuggestedInputs: []string{"构造满足条件 `this == o` 的输入"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"AttributeKey instance = AttributeKey.create(\"test\");",
		"Assertions.assertTrue(instance.equals(instance));",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "new AttributeKey(") {
		t.Fatalf("private constructor should not be called directly:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskHandlesEnumMethodBranches(t *testing.T) {
	source := []byte(`public enum ClientType {
    PRODUCER,
    PUSH_CONSUMER;

    public apache.rocketmq.v2.ClientType toProtobuf() {
        if (PRODUCER.equals(this)) {
            return apache.rocketmq.v2.ClientType.PRODUCER;
        }
        if (PUSH_CONSUMER.equals(this)) {
            return apache.rocketmq.v2.ClientType.PUSH_CONSUMER;
        }
        return apache.rocketmq.v2.ClientType.CLIENT_TYPE_UNSPECIFIED;
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "ClientType.java", &types.CoverageTestTask{
		Target:          "ClientType.toProtobuf",
		LineRange:       "6-6",
		TestName:        "shouldCoverClientTypeToProtobufGap",
		MissingBranches: []string{"未覆盖 if 分支: PRODUCER.equals(this"},
		SuggestedInputs: []string{"构造满足条件 `PRODUCER.equals(this` 的输入"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"public void shouldCoverClientTypeToProtobufGap()",
		"apache.rocketmq.v2.ClientType result = ClientType.PRODUCER.toProtobuf();",
		"Assertions.assertEquals(apache.rocketmq.v2.ClientType.PRODUCER, result);",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "new ClientType") {
		t.Fatalf("enum method task should use enum constants, not constructors:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskHandlesInflightRequestCountInterceptor(t *testing.T) {
	source := []byte(`public class InflightRequestCountInterceptor implements MessageInterceptor {
    @Override
    public void doBefore(MessageInterceptorContext context, java.util.List<GeneralMessage> messages) {
        if (context.getMessageHookPoints() == MessageHookPoints.RECEIVE) {
            inflightReceiveRequestCount.incrementAndGet();
        }
    }

    @Override
    public void doAfter(MessageInterceptorContext context, java.util.List<GeneralMessage> messages) {
        if (context.getMessageHookPoints() == MessageHookPoints.RECEIVE) {
            inflightReceiveRequestCount.decrementAndGet();
        }
    }

    public long getInflightReceiveRequestCount() {
        return inflightReceiveRequestCount.get();
    }
}
`)

	_, beforeCode, err := GenerateJavaTestsForCoverageTask(source, "InflightRequestCountInterceptor.java", &types.CoverageTestTask{
		Target:          "InflightRequestCountInterceptor.doBefore",
		LineRange:       "4-4",
		TestName:        "shouldCoverInflightRequestCountInterceptorDoBeforeGap",
		MissingBranches: []string{"未覆盖 if 分支: context.getMessageHookPoints"},
		SuggestedInputs: []string{"构造满足条件 `context.getMessageHookPoints` 的输入"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(doBefore) error = %v", err)
	}
	for _, want := range []string{
		"MessageInterceptorContext context = new MessageInterceptorContextImpl(MessageHookPoints.RECEIVE);",
		"instance.doBefore(context, java.util.Collections.emptyList());",
		"Assertions.assertEquals(1L, instance.getInflightReceiveRequestCount());",
	} {
		if !strings.Contains(beforeCode, want) {
			t.Fatalf("expected %q in doBefore generated code:\n%s", want, beforeCode)
		}
	}
	if strings.Contains(beforeCode, "new MessageInterceptorContext()") {
		t.Fatalf("interface context should not be instantiated directly:\n%s", beforeCode)
	}

	_, afterCode, err := GenerateJavaTestsForCoverageTask(source, "InflightRequestCountInterceptor.java", &types.CoverageTestTask{
		Target:          "InflightRequestCountInterceptor.doAfter",
		LineRange:       "10-10",
		TestName:        "shouldCoverInflightRequestCountInterceptorDoAfterGap",
		MissingBranches: []string{"未覆盖 if 分支: context.getMessageHookPoints"},
		SuggestedInputs: []string{"构造满足条件 `context.getMessageHookPoints` 的输入"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(doAfter) error = %v", err)
	}
	for _, want := range []string{
		"instance.doBefore(context, java.util.Collections.emptyList());",
		"Assertions.assertEquals(1L, instance.getInflightReceiveRequestCount());",
		"instance.doAfter(context, java.util.Collections.emptyList());",
		"Assertions.assertEquals(0L, instance.getInflightReceiveRequestCount());",
	} {
		if !strings.Contains(afterCode, want) {
			t.Fatalf("expected %q in doAfter generated code:\n%s", want, afterCode)
		}
	}
}

func TestGenerateJavaTestsForCoverageTaskHandlesCompositedMessageInterceptor(t *testing.T) {
	source := []byte(`public class CompositedMessageInterceptor implements MessageInterceptor {
    public CompositedMessageInterceptor(java.util.List<MessageInterceptor> interceptors) {
    }

    @Override
    public void doBefore(MessageInterceptorContext context0, java.util.List<GeneralMessage> messages) {
        if (context0 instanceof MessageInterceptorContextImpl) {
            ((MessageInterceptorContextImpl) context0).getAttributes().forEach(context::putAttribute);
        }
    }

    @Override
    public void doAfter(MessageInterceptorContext context0, java.util.List<GeneralMessage> messages) {
        if (context0 instanceof MessageInterceptorContextImpl) {
            ((MessageInterceptorContextImpl) context0).getAttributes().forEach(context::putAttribute);
        }
    }
}
`)

	_, beforeCode, err := GenerateJavaTestsForCoverageTask(source, "CompositedMessageInterceptor.java", &types.CoverageTestTask{
		Target:          "CompositedMessageInterceptor.doBefore",
		LineRange:       "7-7",
		TestName:        "shouldCoverCompositedMessageInterceptorDoBeforeGap",
		MissingBranches: []string{"未覆盖 if 分支: context0 instanceof MessageInterceptorContextImpl"},
		SuggestedInputs: []string{"构造满足条件 `context0 instanceof MessageInterceptorContextImpl` 的输入"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(doBefore) error = %v", err)
	}
	for _, want := range []string{
		"MessageInterceptor interceptor = new MessageInterceptor() {",
		"new CompositedMessageInterceptor(java.util.Collections.singletonList(interceptor));",
		"MessageInterceptorContextImpl context = new MessageInterceptorContextImpl(",
		"instance.doBefore(context, java.util.Collections.emptyList());",
		"Assertions.assertTrue(called[0]);",
	} {
		if !strings.Contains(beforeCode, want) {
			t.Fatalf("expected %q in doBefore generated code:\n%s", want, beforeCode)
		}
	}
	if strings.Contains(beforeCode, "new MessageInterceptorContext()") {
		t.Fatalf("interface context should not be instantiated directly:\n%s", beforeCode)
	}

	_, afterCode, err := GenerateJavaTestsForCoverageTask(source, "CompositedMessageInterceptor.java", &types.CoverageTestTask{
		Target:          "CompositedMessageInterceptor.doAfter",
		LineRange:       "14-14",
		TestName:        "shouldCoverCompositedMessageInterceptorDoAfterGap",
		MissingBranches: []string{"未覆盖 if 分支: context0 instanceof MessageInterceptorContextImpl"},
		SuggestedInputs: []string{"构造满足条件 `context0 instanceof MessageInterceptorContextImpl` 的输入"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(doAfter) error = %v", err)
	}
	for _, want := range []string{
		"instance.doBefore(context, java.util.Collections.emptyList());",
		"instance.doAfter(context, java.util.Collections.emptyList());",
		"Assertions.assertTrue(called[1]);",
	} {
		if !strings.Contains(afterCode, want) {
			t.Fatalf("expected %q in doAfter generated code:\n%s", want, afterCode)
		}
	}
}

func TestGenerateJavaTestsForCoverageTaskMarksPrivateJavaMethodManualReview(t *testing.T) {
	source := []byte(`public class ClientManagerImpl {
    private void clearIdleRpcClients() throws InterruptedException {
        if (idleDuration.compareTo(RPC_CLIENT_MAX_IDLE_DURATION) > 0) {
            rpcClient.shutdown();
        }
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "ClientManagerImpl.java", &types.CoverageTestTask{
		Target:          "ClientManagerImpl.clearIdleRpcClients",
		LineRange:       "3-3",
		TestName:        "shouldCoverClientManagerImplClearIdleRpcClientsGap",
		MissingBranches: []string{"未覆盖 if 分支: idleDuration.compareTo(RPC_CLIENT_MAX_IDLE_DURATION"},
		SuggestedInputs: []string{"构造满足条件 `idleDuration.compareTo(RPC_CLIENT_MAX_IDLE_DURATION` 的输入"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"public void shouldCoverClientManagerImplClearIdleRpcClientsGap()",
		"manual_review_internal: ",
		"ClientManagerImpl.clearIdleRpcClients",
		"org.junit.jupiter.api.Assumptions.assumeTrue(false, reason);",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "public class ClientManagerImplTest {\n}") {
		t.Fatalf("private task should not generate an empty test class:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskCoversConsumerImplFilterExpressionViaRequest(t *testing.T) {
	source := []byte(`package org.apache.rocketmq.client.java.impl.consumer;

import apache.rocketmq.v2.ReceiveMessageRequest;
import java.time.Duration;
import org.apache.rocketmq.client.apis.ClientConfiguration;
import org.apache.rocketmq.client.apis.consumer.FilterExpression;

abstract class ConsumerImpl {
    private apache.rocketmq.v2.FilterExpression wrapFilterExpression(FilterExpression filterExpression) {
        switch (filterExpression.getFilterExpressionType()) {
            case SQL92:
                return apache.rocketmq.v2.FilterExpression.newBuilder()
                    .setType(apache.rocketmq.v2.FilterType.SQL).build();
            case TAG:
            default:
                return apache.rocketmq.v2.FilterExpression.newBuilder()
                    .setType(apache.rocketmq.v2.FilterType.TAG).build();
        }
    }

    ReceiveMessageRequest wrapReceiveMessageRequest(int batchSize, Object mq,
        FilterExpression filterExpression, Duration longPollingTimeout, String attemptId) {
        return ReceiveMessageRequest.newBuilder()
            .setFilterExpression(wrapFilterExpression(filterExpression)).build();
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "ConsumerImpl.java", &types.CoverageTestTask{
		Target:          "ConsumerImpl.wrapFilterExpression",
		LineRange:       "10-10",
		TestName:        "shouldCoverConsumerImplWrapFilterExpressionGap",
		MissingBranches: []string{"未覆盖 switch/case 分支"},
		SuggestedInputs: []string{"设置 filterExpression 覆盖未执行分支"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"public class ConsumerImplTest extends TestBase",
		"import org.apache.rocketmq.client.java.tool.TestBase;",
		"final PushConsumerImpl consumer = new PushConsumerImpl(",
		"new FilterExpression(",
		"FilterExpressionType.SQL92",
		"consumer.wrapReceiveMessageRequest(",
		"Assertions.assertEquals(",
		"apache.rocketmq.v2.FilterType.SQL",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "manual_review_internal:") {
		t.Fatalf("filter expression task should use public request path, got manual review:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskCoversConsumerImplFilterExpressionTagPath(t *testing.T) {
	source := []byte(`package org.apache.rocketmq.client.java.impl.consumer;

import apache.rocketmq.v2.ReceiveMessageRequest;
import java.time.Duration;
import org.apache.rocketmq.client.apis.ClientConfiguration;
import org.apache.rocketmq.client.apis.consumer.FilterExpression;

abstract class ConsumerImpl {
    private apache.rocketmq.v2.FilterExpression wrapFilterExpression(FilterExpression filterExpression) {
        switch (filterExpression.getFilterExpressionType()) {
            case SQL92:
                return apache.rocketmq.v2.FilterExpression.newBuilder()
                    .setType(apache.rocketmq.v2.FilterType.SQL).build();
            case TAG:
            default:
                return apache.rocketmq.v2.FilterExpression.newBuilder()
                    .setType(apache.rocketmq.v2.FilterType.TAG).build();
        }
    }

    ReceiveMessageRequest wrapReceiveMessageRequest(int batchSize, Object mq,
        FilterExpression filterExpression, Duration longPollingTimeout, String attemptId) {
        return ReceiveMessageRequest.newBuilder()
            .setFilterExpression(wrapFilterExpression(filterExpression)).build();
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "ConsumerImpl.java", &types.CoverageTestTask{
		Target:    "ConsumerImpl.wrapFilterExpression",
		LineRange: "255-255",
		TestName:  "shouldCoverConsumerImplWrapFilterExpressionTagGap",
		GapType:   "return",
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"final FilterExpression filterExpression = new FilterExpression();",
		"apache.rocketmq.v2.FilterType.TAG",
		"consumer.wrapReceiveMessageRequest(",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "FilterExpressionType.SQL92") || strings.Contains(code, "manual_review_internal:") {
		t.Fatalf("tag path should not use SQL branch or manual review:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskCoversConsumerImplReceiveMessageRequestOverloads(t *testing.T) {
	source := []byte(`package org.apache.rocketmq.client.java.impl.consumer;

import apache.rocketmq.v2.ReceiveMessageRequest;
import com.google.protobuf.util.Durations;
import java.time.Duration;
import org.apache.rocketmq.client.apis.ClientConfiguration;
import org.apache.rocketmq.client.apis.consumer.FilterExpression;

abstract class ConsumerImpl {
    ReceiveMessageRequest wrapReceiveMessageRequest(int batchSize, Object mq,
        FilterExpression filterExpression, Duration longPollingTimeout, String attemptId) {
        return ReceiveMessageRequest.newBuilder().setAutoRenew(true).setAttemptId(attemptId).build();
    }

    ReceiveMessageRequest wrapReceiveMessageRequest(int batchSize, Object mq,
        FilterExpression filterExpression, Duration invisibleDuration, Duration longPollingTimeout) {
        return ReceiveMessageRequest.newBuilder()
            .setAutoRenew(false)
            .setInvisibleDuration(Durations.fromNanos(invisibleDuration.toNanos()))
            .build();
    }
}
`)

	_, autoRenewCode, err := GenerateJavaTestsForCoverageTask(source, "ConsumerImpl.java", &types.CoverageTestTask{
		Target:    "ConsumerImpl.wrapReceiveMessageRequest",
		LineRange: "12-12",
		TestName:  "shouldCoverConsumerImplWrapReceiveMessageRequestAutoRenewGap",
		GapType:   "return",
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(auto renew) error = %v", err)
	}
	for _, want := range []string{
		"consumer.wrapReceiveMessageRequest(",
		"Duration.ofSeconds(30), \"attempt-id\");",
		"Assertions.assertTrue(request.getAutoRenew());",
		"Assertions.assertEquals(\"attempt-id\", request.getAttemptId());",
	} {
		if !strings.Contains(autoRenewCode, want) {
			t.Fatalf("expected %q in auto-renew generated code:\n%s", want, autoRenewCode)
		}
	}

	_, invisibleCode, err := GenerateJavaTestsForCoverageTask(source, "ConsumerImpl.java", &types.CoverageTestTask{
		Target:    "ConsumerImpl.wrapReceiveMessageRequest",
		LineRange: "18-18",
		TestName:  "shouldCoverConsumerImplWrapReceiveMessageRequestInvisibleGap",
		GapType:   "return",
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask(invisible) error = %v", err)
	}
	for _, want := range []string{
		"final Duration invisibleDuration = Duration.ofSeconds(45);",
		"invisibleDuration, Duration.ofSeconds(30));",
		"Assertions.assertFalse(request.getAutoRenew());",
		"Durations.fromNanos(invisibleDuration.toNanos())",
	} {
		if !strings.Contains(invisibleCode, want) {
			t.Fatalf("expected %q in invisible generated code:\n%s", want, invisibleCode)
		}
	}
	if strings.Contains(autoRenewCode, "manual_review_internal:") || strings.Contains(invisibleCode, "manual_review_internal:") {
		t.Fatalf("wrapReceiveMessageRequest tasks should not use manual review:\nauto:\n%s\ninvisible:\n%s", autoRenewCode, invisibleCode)
	}
}

func TestGenerateJavaTestsForCoverageTaskCoversConsumerImplReceiveMessageResponses(t *testing.T) {
	source := []byte(`package org.apache.rocketmq.client.java.impl.consumer;

import apache.rocketmq.v2.ReceiveMessageRequest;
import apache.rocketmq.v2.ReceiveMessageResponse;
import com.google.common.util.concurrent.ListenableFuture;
import java.time.Duration;
import java.util.List;
import org.apache.rocketmq.client.apis.ClientConfiguration;
import org.apache.rocketmq.client.apis.consumer.FilterExpression;
import org.apache.rocketmq.client.java.impl.ClientManager;
import org.apache.rocketmq.client.java.message.MessageViewImpl;
import org.apache.rocketmq.client.java.route.Endpoints;
import org.apache.rocketmq.client.java.route.MessageQueueImpl;
import org.apache.rocketmq.client.java.rpc.RpcFuture;

abstract class ConsumerImpl {
    protected ListenableFuture<ReceiveMessageResult> receiveMessage(ReceiveMessageRequest request,
        MessageQueueImpl mq, Duration awaitDuration) {
        return null;
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "ConsumerImpl.java", &types.CoverageTestTask{
		Target:          "ConsumerImpl.receiveMessage",
		LineRange:       "104-106",
		TestName:        "shouldCoverConsumerImplReceiveMessageDeliveryTimestampGap",
		MissingBranches: []string{"未覆盖 switch/case 分支: DELIVERY_TIMESTAMP"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"public class ConsumerImplTest extends TestBase",
		"final PushConsumerImpl consumer = org.mockito.Mockito.spy(new PushConsumerImpl(",
		"final List<ReceiveMessageResponse> responses = new java.util.ArrayList<>();",
		".setDeliveryTimestamp(deliveryTimestamp)",
		"new RpcFuture<>(fakeRpcContext(), request,",
		"clientManager).receiveMessage(",
		"consumer.receiveMessage(request, mq, Duration.ofSeconds(15))",
		"final MessageViewImpl message = result.getMessageViewImpls().get(0);",
		"Assertions.assertTrue(message.getTransportDeliveryTimestamp().isPresent());",
		"Assertions.assertEquals(Long.valueOf(123000L), message.getTransportDeliveryTimestamp().get());",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "manual_review_internal:") {
		t.Fatalf("receiveMessage task should use ClientManager mock path, got manual review:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskCoversConsumerImplAckMessageViaClientManagerMock(t *testing.T) {
	source := []byte(`package org.apache.rocketmq.client.java.impl.consumer;

import apache.rocketmq.v2.AckMessageRequest;
import apache.rocketmq.v2.AckMessageResponse;
import java.time.Duration;
import org.apache.rocketmq.client.apis.ClientConfiguration;
import org.apache.rocketmq.client.java.impl.ClientManager;
import org.apache.rocketmq.client.java.message.MessageViewImpl;
import org.apache.rocketmq.client.java.route.Endpoints;
import org.apache.rocketmq.client.java.rpc.RpcFuture;

abstract class ConsumerImpl {
    protected RpcFuture<AckMessageRequest, AckMessageResponse> ackMessage(MessageViewImpl messageView) {
        return this.getClientManager().ackMessage(messageView.getEndpoints(), null, Duration.ofSeconds(3));
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "ConsumerImpl.java", &types.CoverageTestTask{
		Target:    "ConsumerImpl.ackMessage",
		LineRange: "13-13",
		TestName:  "shouldCoverConsumerImplAckMessageGap",
		GapType:   "return",
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"public class ConsumerImplTest extends TestBase",
		"final PushConsumerImpl consumer = org.mockito.Mockito.spy(new PushConsumerImpl(",
		"final ClientManager clientManager = org.mockito.Mockito.mock(ClientManager.class);",
		"okAckMessageResponseFuture()",
		"clientManager).ackMessage(",
		"consumer.ackMessage(messageView)",
		"Assertions.assertEquals(future.get(), result.get());",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "manual_review_internal:") {
		t.Fatalf("ackMessage task should use ClientManager mock path, got manual review:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskCoversConsumerImplChangeInvisibleDurationErrorPath(t *testing.T) {
	source := []byte(`package org.apache.rocketmq.client.java.impl.consumer;

import apache.rocketmq.v2.ChangeInvisibleDurationRequest;
import apache.rocketmq.v2.ChangeInvisibleDurationResponse;
import java.time.Duration;
import org.apache.rocketmq.client.apis.ClientConfiguration;
import org.apache.rocketmq.client.java.impl.ClientManager;
import org.apache.rocketmq.client.java.message.MessageViewImpl;
import org.apache.rocketmq.client.java.route.Endpoints;
import org.apache.rocketmq.client.java.rpc.RpcFuture;

abstract class ConsumerImpl {
    RpcFuture<ChangeInvisibleDurationRequest, ChangeInvisibleDurationResponse> changeInvisibleDuration(
        MessageViewImpl messageView, Duration invisibleDuration) {
        return this.getClientManager().changeInvisibleDuration(messageView.getEndpoints(), null, invisibleDuration);
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "ConsumerImpl.java", &types.CoverageTestTask{
		Target:          "ConsumerImpl.changeInvisibleDuration",
		LineRange:       "209-209",
		TestName:        "shouldCoverConsumerImplChangeInvisibleDurationGap",
		MissingBranches: []string{"未覆盖 if 分支: !Code.OK.equals(code"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"public class ConsumerImplTest extends TestBase",
		"changInvisibleDurationCtxFuture(apache.rocketmq.v2.Code.INTERNAL_SERVER_ERROR)",
		"clientManager).changeInvisibleDuration(",
		"consumer.changeInvisibleDuration(messageView, Duration.ofSeconds(15))",
		"Assertions.assertEquals(future.get(), result.get());",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "manual_review_internal:") {
		t.Fatalf("changeInvisibleDuration task should use ClientManager mock path, got manual review:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskMarksUnconstructibleJavaInstanceManualReview(t *testing.T) {
	source := []byte(`public class ClientSessionImpl {
    protected ClientSessionImpl(ClientSessionHandler sessionHandler, Duration tolerance, Endpoints endpoints) {
    }

    public void release() {
        if (requestObserver == null) {
            return;
        }
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "ClientSessionImpl.java", &types.CoverageTestTask{
		Target:          "ClientSessionImpl.release",
		LineRange:       "6-6",
		TestName:        "shouldCoverClientSessionImplReleaseGap",
		MissingBranches: []string{"未覆盖 if 分支: null == requestObserver"},
		SuggestedInputs: []string{"构造满足条件 `null == requestObserver` 的输入"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"public void shouldCoverClientSessionImplReleaseGap()",
		"manual_review_internal: ",
		"ClientSessionImpl.release",
		"requires complex constructor state",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "new ClientSessionImpl()") || strings.Contains(code, "instance.release()") {
		t.Fatalf("unconstructible instance task should not emit invalid direct construction/call:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskUsesNullForUnknownJavaConstructorArg(t *testing.T) {
	source := []byte(`public class Assignment {
    private final MessageQueueImpl messageQueue;

    public Assignment(MessageQueueImpl messageQueue) {
        this.messageQueue = messageQueue;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) {
            return true;
        }
        if (o == null || getClass() != o.getClass()) {
            return false;
        }
        return true;
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "Assignment.java", &types.CoverageTestTask{
		Target:          "Assignment.equals",
		LineRange:       "9-9",
		TestName:        "shouldCoverAssignmentEqualsGap",
		MissingBranches: []string{"未覆盖 if 分支: this == o"},
		SuggestedInputs: []string{"构造满足条件 `this == o` 的输入"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"Assignment instance = new Assignment(null);",
		"Assertions.assertTrue(instance.equals(instance));",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "new MessageQueueImpl()") || strings.Contains(code, "(MessageQueueImpl) null") {
		t.Fatalf("unknown constructor arg should not require missing imports:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskAddsReferencedSourceImports(t *testing.T) {
	source := []byte(`import org.apache.rocketmq.client.apis.consumer.ConsumeResult;

public class ConsumeTask {
    public ConsumeResult call() {
        return ConsumeResult.SUCCESS;
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "ConsumeTask.java", &types.CoverageTestTask{
		Target:          "ConsumeTask.call",
		LineRange:       "4-4",
		TestName:        "shouldCoverConsumeTaskCallGap",
		MissingBranches: []string{"未覆盖 if 分支: !ConsumeResult.SUCCESS.equals(consumeResult"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"import org.apache.rocketmq.client.apis.consumer.ConsumeResult;",
		"ConsumeResult result = instance.call();",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
}

func TestGenerateJavaTestsForCoverageTaskDoesNotImportCommentOnlyTypes(t *testing.T) {
	source := []byte(`import java.time.Duration;

public class Worker {
    public Object run(Object value) {
        return value;
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "Worker.java", &types.CoverageTestTask{
		Target:          "Worker.run",
		LineRange:       "4-4",
		TestName:        "shouldCoverWorkerRunGap",
		MissingBranches: []string{"未覆盖 if 分支: Duration.ZERO.compareTo(delay"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	if strings.Contains(code, "import java.time.Duration;") {
		t.Fatalf("comment-only source import should not be copied:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskHandlesRocketMQConsumeTaskCall(t *testing.T) {
	source := []byte(`import org.apache.rocketmq.client.apis.consumer.ConsumeResult;
import org.apache.rocketmq.client.apis.consumer.MessageListener;
import org.apache.rocketmq.client.java.hook.MessageInterceptor;
import org.apache.rocketmq.client.java.message.MessageViewImpl;
import org.apache.rocketmq.client.java.misc.ClientId;

public class ConsumeTask {
    public ConsumeTask(ClientId clientId, MessageListener messageListener, MessageViewImpl messageView,
        MessageInterceptor messageInterceptor) {
    }

    public ConsumeResult call() {
        return ConsumeResult.FAILURE;
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "ConsumeTask.java", &types.CoverageTestTask{
		Target:          "ConsumeTask.call",
		LineRange:       "13-13",
		TestName:        "shouldCoverConsumeTaskCallGap",
		MissingBranches: []string{"未覆盖 if 分支: !ConsumeResult.SUCCESS.equals(consumeResult"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"import org.apache.rocketmq.client.java.tool.TestBase;",
		"public class ConsumeTaskTest extends TestBase {",
		"final MessageViewImpl messageView = fakeMessageViewImpl();",
		"final MessageListener messageListener = message -> ConsumeResult.FAILURE;",
		"org.mockito.Mockito.mock(MessageInterceptor.class);",
		"Assertions.assertEquals(ConsumeResult.FAILURE, result);",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "new ConsumeTask(null") {
		t.Fatalf("ConsumeTask task should use real fakes instead of null constructor args:\n%s", code)
	}
}

func TestGenerateJavaTestsForCoverageTaskHandlesRocketMQConsumeServiceConsume(t *testing.T) {
	source := []byte(`import com.google.common.util.concurrent.ListenableFuture;
import java.util.List;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledThreadPoolExecutor;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;
import org.apache.rocketmq.client.apis.consumer.ConsumeResult;
import org.apache.rocketmq.client.apis.consumer.MessageListener;
import org.apache.rocketmq.client.java.hook.MessageInterceptor;
import org.apache.rocketmq.client.java.message.MessageViewImpl;
import org.apache.rocketmq.client.java.misc.ClientId;
import org.apache.rocketmq.client.java.misc.ThreadFactoryImpl;

public abstract class ConsumeService {
    public ConsumeService(ClientId clientId, MessageListener messageListener,
        ThreadPoolExecutor consumptionExecutor, MessageInterceptor interceptor,
        ScheduledExecutorService scheduler) {
    }

    public abstract void consume(ProcessQueue pq, List<MessageViewImpl> messageViews);

    public ListenableFuture<ConsumeResult> consume(MessageViewImpl messageView) {
        return null;
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "ConsumeService.java", &types.CoverageTestTask{
		Target:          "ConsumeService.consume",
		LineRange:       "23-23",
		TestName:        "shouldCoverConsumeServiceConsumeGap",
		MissingBranches: []string{"未覆盖 if 分支: Duration.ZERO.compareTo(delay"},
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"public class ConsumeServiceTest extends TestBase {",
		"final ThreadPoolExecutor consumptionExecutor = new ThreadPoolExecutor(",
		"new java.util.concurrent.LinkedBlockingQueue<>()",
		"new java.util.concurrent.ScheduledThreadPoolExecutor(",
		"final ConsumeService instance = new ConsumeService(",
		"public void consume(ProcessQueue pq, List<MessageViewImpl> messageViews) {",
		"final ListenableFuture<ConsumeResult> future = instance.consume(messageView);",
		"Assertions.assertEquals(ConsumeResult.SUCCESS, result);",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "new ConsumeService(null") || strings.Contains(code, "instance.consume(null, null)") {
		t.Fatalf("ConsumeService task should avoid abstract construction and null overload ambiguity:\n%s", code)
	}
	for _, forbidden := range []string{
		"import java.util.concurrent.LinkedBlockingQueue;",
		"import java.util.concurrent.ScheduledThreadPoolExecutor;",
		"import org.apache.rocketmq.client.java.misc.ThreadFactoryImpl;",
	} {
		if strings.Contains(code, forbidden) {
			t.Fatalf("fully qualified helper should not copy unused import %q:\n%s", forbidden, code)
		}
	}
}

func TestGenerateJavaTestsForCoverageTaskRemovesUnusedAssertionImport(t *testing.T) {
	source := []byte(`public class NoopHook {
    public void run() {
    }
}
`)

	_, code, err := GenerateJavaTestsForCoverageTask(source, "NoopHook.java", &types.CoverageTestTask{
		Target:   "NoopHook.run",
		TestName: "shouldCoverNoopHookRunGap",
	})
	if err != nil {
		t.Fatalf("GenerateJavaTestsForCoverageTask() error = %v", err)
	}
	if strings.Contains(code, "import org.junit.jupiter.api.Assertions;") || strings.Contains(code, "import org.junit.Assert;") {
		t.Fatalf("unused assertion import should be removed:\n%s", code)
	}
	if !strings.Contains(code, "import org.junit.jupiter.api.Test;") {
		t.Fatalf("test import should stay:\n%s", code)
	}
}

func TestCoverageTaskInputValuesPreservesJavaScriptUndefined(t *testing.T) {
	task := types.CoverageTestTask{
		SuggestedInputs: []string{"构造满足条件 `value === undefined` 的输入"},
	}
	values := coverageTaskInputValues(&task, "javascript")
	if values["value"] != "undefined" {
		t.Fatalf("expected JavaScript undefined, got %+v", values)
	}
}
