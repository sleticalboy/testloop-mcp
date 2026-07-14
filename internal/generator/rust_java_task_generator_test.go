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

func TestCoverageTaskInputValuesPreservesJavaScriptUndefined(t *testing.T) {
	task := types.CoverageTestTask{
		SuggestedInputs: []string{"构造满足条件 `value === undefined` 的输入"},
	}
	values := coverageTaskInputValues(&task, "javascript")
	if values["value"] != "undefined" {
		t.Fatalf("expected JavaScript undefined, got %+v", values)
	}
}
