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

func TestCoverageTaskInputValuesPreservesJavaScriptUndefined(t *testing.T) {
	task := types.CoverageTestTask{
		SuggestedInputs: []string{"构造满足条件 `value === undefined` 的输入"},
	}
	values := coverageTaskInputValues(&task, "javascript")
	if values["value"] != "undefined" {
		t.Fatalf("expected JavaScript undefined, got %+v", values)
	}
}
