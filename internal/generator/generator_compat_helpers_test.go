package generator

import (
	"strings"
	"testing"
)

func TestGoGeneratorCompatHelpers(t *testing.T) {
	mock := genMock(interfaceInfo{
		Name: "Store",
		Methods: []methodSig{
			{
				Name: "Save",
				Params: []paramInfo{
					{Name: "id", Type: "string"},
				},
				Returns: []paramInfo{
					{Name: "ok", Type: "bool"},
					{Name: "err", Type: "error"},
				},
			},
			{Name: "Close"},
		},
	})
	for _, want := range []string{
		"type StoreMock struct",
		"SaveFn func(string) (bool, error)",
		"func (m *StoreMock) Save(id string) (bool, error)",
		"ret0, ret1 := m.SaveFn(id)",
		"return ret0, ret1",
		"return false, nil",
		"func (m *StoreMock) Close()",
	} {
		if !strings.Contains(mock, want) {
			t.Fatalf("expected %q in mock:\n%s", want, mock)
		}
	}

	test := genTableDrivenTest(funcInfo{
		Name:         "Publish",
		Receiver:     "svc",
		ReceiverType: "*Service",
		IsMethod:     true,
		Params: []paramInfo{
			{Name: "items", Type: "[]string"},
			{Name: "done", Type: "chan bool"},
		},
		Returns: []paramInfo{
			{Name: "ret0", Type: "error"},
		},
	})
	for _, want := range []string{
		"func TestService_Publish(t *testing.T)",
		"skip: true",
		"done chan bool",
		"svc := &Service{}",
		"if tt.done == nil",
		"err := svc.Publish(tt.items, tt.done)",
	} {
		if !strings.Contains(test, want) {
			t.Fatalf("expected %q in table-driven test:\n%s", want, test)
		}
	}
}

func TestJavaGeneratorSourceAndCompatHelpers(t *testing.T) {
	source := []byte(`public class Service {
    public static int add(int a, int b) {
        return a + b;
    }

    public void ping(String name) {
    }
}
`)
	name, code, err := GenerateJavaTestsForSource(source, "Service.java")
	if err != nil {
		t.Fatalf("GenerateJavaTestsForSource() error = %v", err)
	}
	if name != "ServiceTest.java" {
		t.Fatalf("unexpected Java test file name: %q", name)
	}
	for _, want := range []string{
		"class ServiceTest",
		"int result = Service.add(0, 0);",
		"Service instance = new Service();",
		"instance.ping(\"test\");",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in Java output:\n%s", want, code)
		}
	}

	var b strings.Builder
	javaWriteMethodTest(&b, javaFuncInfo{
		Name:       "load",
		ClassName:  "Service",
		IsPublic:   true,
		IsStatic:   true,
		ReturnType: "String",
		Params: []javaParamInfo{
			{Name: "id", Type: "String"},
		},
		Throws: []string{"IOException"},
	}, "Fallback")
	out := b.String()
	for _, want := range []string{
		"void load()",
		"String result = Service.load(\"test\");",
		"assertNotNull(result);",
		"assertThrows(IOException.class",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in Java method test:\n%s", want, out)
		}
	}

	if got := javaBuildArgs([]javaParamInfo{{Name: "name", Type: "String"}, {Name: "count", Type: "int"}}); got != "\"test\", 0" {
		t.Fatalf("javaBuildArgs() = %q", got)
	}
}

func TestRustGeneratorSourceAndCompatHelpers(t *testing.T) {
	source := []byte(`pub fn fetch_data(name: &str) -> Result<String, Error> {
    Ok(name.to_string())
}
`)
	name, code, err := GenerateRustTestsForSource(source, "src/lib.rs")
	if err != nil {
		t.Fatalf("GenerateRustTestsForSource() error = %v", err)
	}
	if name != "lib_test.rs" {
		t.Fatalf("unexpected Rust test file name: %q", name)
	}
	for _, want := range []string{
		"fn test_fetch_data()",
		"let result = fetch_data(\"test\");",
		"assert!(result.is_ok() || result.is_err());",
		"fn test_fetch_data_returns_err_for_invalid_input()",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in Rust output:\n%s", want, code)
		}
	}

	f := rsFuncInfo{
		Name:       "maybe_find",
		ReturnType: "Option<i32>",
		HasOption:  true,
		Params: []rsParamInfo{
			{IsSelf: true},
			{Name: "name", Type: "&str"},
		},
	}
	var b strings.Builder
	rsWriteFuncTest(&b, f, nil)
	out := b.String()
	for _, want := range []string{
		"fn test_maybe_find()",
		"let result = maybe_find(\"test\");",
		"match result",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in Rust func test:\n%s", want, out)
		}
	}

	if got := rsBuildArgs(f, nil); got != "\"test\"" {
		t.Fatalf("rsBuildArgs() = %q", got)
	}
}
