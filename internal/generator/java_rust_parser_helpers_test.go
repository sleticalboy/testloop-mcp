package generator

import (
	"testing"

	"github.com/sleticalboy/testloop-mcp/types"
)

func TestJavaParserExtractsConstructorsThrowsVarargsAndGenerics(t *testing.T) {
	source := []byte(`public class Service {
    public Service(String name) {
    }

    public static <T> T identity(T value) throws java.io.IOException, IllegalArgumentException {
        return value;
    }

    public void log(String... messages) {
    }
}
`)

	funcs, classes := parseJavaWithTreeSitter(source)

	if len(classes) != 1 || classes[0].Name != "Service" || !classes[0].IsPublic {
		t.Fatalf("unexpected classes: %+v", classes)
	}

	constructor := findJavaFunc(funcs, "Service")
	if constructor == nil {
		t.Fatalf("constructor not found in funcs: %+v", funcs)
	}
	if !constructor.IsConstructor || constructor.ClassName != "Service" || len(constructor.Params) != 1 || constructor.Params[0].Type != "String" {
		t.Fatalf("unexpected constructor info: %+v", constructor)
	}

	identity := findJavaFunc(funcs, "identity")
	if identity == nil {
		t.Fatalf("identity not found in funcs: %+v", funcs)
	}
	if !identity.IsStatic || !identity.IsGeneric || identity.ReturnType != "T" {
		t.Fatalf("unexpected identity flags: %+v", identity)
	}
	if len(identity.Throws) != 2 || identity.Throws[0] != "java.io.IOException" || identity.Throws[1] != "IllegalArgumentException" {
		t.Fatalf("unexpected throws: %+v", identity.Throws)
	}

	logMethod := findJavaFunc(funcs, "log")
	if logMethod == nil {
		t.Fatalf("log not found in funcs: %+v", funcs)
	}
	if !logMethod.IsVoid || len(logMethod.Params) != 1 || logMethod.Params[0].Type != "String..." || logMethod.Params[0].Name != "messages" {
		t.Fatalf("unexpected log method: %+v", logMethod)
	}
}

func TestJavaParserExtractsInterfaceEnumInnerClassAndSkipsHelpers(t *testing.T) {
	source := []byte(`interface Worker {
    void run(String job);
}

enum Mode {
    FAST;

    public String label() {
        return "fast";
    }
}

class Outer {
    public String getName() {
        return "outer";
    }

    public void setName(String name) {
    }

    public boolean equals(Object other) {
        return false;
    }

    public int value() {
        return 1;
    }

    class Inner {
        int nested() {
            return 2;
        }
    }
}
`)

	funcs, classes := parseJavaWithTreeSitter(source)

	for _, want := range []string{"Worker", "Mode", "Outer"} {
		found := false
		for _, cls := range classes {
			if cls.Name == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("class-like declaration %q not found in %+v", want, classes)
		}
	}

	run := findJavaFunc(funcs, "run")
	if run == nil || run.ClassName != "Worker" || !run.IsVoid || len(run.Params) != 1 || run.Params[0].Name != "job" {
		t.Fatalf("unexpected interface method: %+v", run)
	}
	value := findJavaFunc(funcs, "value")
	if value == nil || value.ClassName != "Outer" || value.ReturnType != "int" {
		t.Fatalf("unexpected outer method: %+v", value)
	}
	nested := findJavaFunc(funcs, "nested")
	if nested == nil || nested.ClassName != "Inner" || nested.ReturnType != "int" {
		t.Fatalf("unexpected inner class method: %+v", nested)
	}
	for _, helper := range []string{"getName", "setName", "equals"} {
		if got := findJavaFunc(funcs, helper); got != nil {
			t.Fatalf("helper %s should be skipped, got %+v in funcs %+v", helper, got, funcs)
		}
	}
}

func TestJavaIsTestHelperBranches(t *testing.T) {
	for _, name := range []string{"testValue", "TestValue", "main", "equals", "hashCode", "toString", "getName", "setName"} {
		if !javaIsTestHelper(name) {
			t.Fatalf("javaIsTestHelper(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"value", "getter", "set", "get", "getname", "compute"} {
		if javaIsTestHelper(name) {
			t.Fatalf("javaIsTestHelper(%q) = true, want false", name)
		}
	}
}

func TestJavaInferDefaultValueAndAssert(t *testing.T) {
	defaults := map[string]string{
		"":                    "null",
		"int":                 "0",
		"double":              "0.0",
		"boolean":             "false",
		"char":                "'a'",
		"String":              "\"test\"",
		"List<String>":        "java.util.Collections.emptyList()",
		"Map<String,Integer>": "java.util.Collections.emptyMap()",
		"Set<Long>":           "java.util.Collections.emptySet()",
		"Optional<String>":    "java.util.Optional.empty()",
		"com.example.User":    "null",
		"User":                "new User()",
	}
	for typ, want := range defaults {
		t.Run("default_"+typ, func(t *testing.T) {
			if got := javaInferDefaultValue(typ); got != want {
				t.Fatalf("javaInferDefaultValue(%q) = %q, want %q", typ, got, want)
			}
		})
	}

	asserts := map[string]string{
		"void":    "",
		"":        "",
		"int":     "assertEquals(0, result);",
		"double":  "assertEquals(0.0, result, 0.001);",
		"boolean": "assertTrue(result);",
		"String":  "assertNotNull(result);",
		"User":    "assertNotNull(result);",
	}
	for typ, want := range asserts {
		t.Run("assert_"+typ, func(t *testing.T) {
			if got := javaInferAssert(typ, "result"); got != want {
				t.Fatalf("javaInferAssert(%q) = %q, want %q", typ, got, want)
			}
		})
	}
}

func TestJavaGeneratorFilterAndNameHelpers(t *testing.T) {
	funcs := []javaFuncInfo{
		{Name: "add", ClassName: "Calculator"},
		{Name: "sub", ClassName: "Calculator"},
	}

	got := filterJavaFuncsForCoverageTask(funcs, &types.CoverageTestTask{})
	if len(got) != 2 {
		t.Fatalf("empty target should keep all funcs: %+v", got)
	}

	got = filterJavaFuncsForCoverageTask(funcs, &types.CoverageTestTask{Target: "Calculator.add"})
	if len(got) != 1 || got[0].Name != "add" {
		t.Fatalf("class method target filtered incorrectly: %+v", got)
	}

	got = filterJavaFuncsForCoverageTask(funcs, &types.CoverageTestTask{Target: "missing"})
	if len(got) != 2 {
		t.Fatalf("missing target should fall back to all funcs: %+v", got)
	}

	tests := map[string]string{
		"":                 "fallback",
		"should cover gap": "shouldcovergap",
		"valid_name_2":     "valid_name_2",
		"1bad":             "bad",
		"---":              "fallback",
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			if got := sanitizeJavaTestMethodName(input, "fallback"); got != want {
				t.Fatalf("sanitizeJavaTestMethodName(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestRustParserExtractsTraitMethodsSelfAndModifiers(t *testing.T) {
	source := []byte(`pub trait Validator {
    fn check(&self, value: i32) -> bool {
        value > 0
    }

    fn refresh(&mut self) -> Result<String, Error> {
        Ok(String::new())
    }
}

pub struct Counter;

impl Counter {
    pub fn new() -> Self {
        Counter
    }

    pub async fn increment(&mut self, amount: u32) -> Option<i32> {
        Some(amount as i32)
    }
}
`)

	funcs, structs := parseRustWithTreeSitter(source)

	if findRustStruct(structs, "Counter") == nil {
		t.Fatalf("Counter struct not found: %+v", structs)
	}

	check := findRustFunc(funcs, "check")
	if check == nil {
		t.Fatalf("trait check not found in funcs: %+v", funcs)
	}
	if !check.IsMethod || !check.HasSelf || check.ReturnType != "bool" || len(check.Params) != 2 {
		t.Fatalf("unexpected check method: %+v", check)
	}

	refresh := findRustFunc(funcs, "refresh")
	if refresh == nil {
		t.Fatalf("trait refresh not found in funcs: %+v", funcs)
	}
	if !refresh.HasResult || !refresh.Params[0].IsMutSelf {
		t.Fatalf("unexpected refresh method: %+v", refresh)
	}

	increment := findRustFunc(funcs, "increment")
	if increment == nil {
		t.Fatalf("increment not found in funcs: %+v", funcs)
	}
	if increment.Owner != "Counter" || !increment.IsPub || !increment.HasOption {
		t.Fatalf("unexpected increment method: %+v", increment)
	}
}

func TestRustGeneratorFilterAndTypeHelpers(t *testing.T) {
	funcs := []rsFuncInfo{
		{Name: "add"},
		{Name: "check", Owner: "Validator"},
		{Name: "skip", Owner: "Validator"},
	}

	got := filterRustFuncsForCoverageTask(funcs, &types.CoverageTestTask{})
	if len(got) != 3 {
		t.Fatalf("empty target should keep all funcs: %+v", got)
	}

	got = filterRustFuncsForCoverageTask(funcs, &types.CoverageTestTask{Target: "Validator.check"})
	if len(got) != 1 || got[0].Name != "check" {
		t.Fatalf("method target filtered incorrectly: %+v", got)
	}

	got = filterRustFuncsForCoverageTask(funcs, &types.CoverageTestTask{Target: "missing"})
	if len(got) != 3 {
		t.Fatalf("missing target should fall back to all funcs: %+v", got)
	}

	if got := rsInferTypeName(rsFuncInfo{Owner: "Counter"}, nil); got != "Counter" {
		t.Fatalf("rsInferTypeName(owner) = %q", got)
	}
	if got := rsInferTypeName(rsFuncInfo{}, []rsStructInfo{{Name: "Fallback"}}); got != "Fallback" {
		t.Fatalf("rsInferTypeName(fallback struct) = %q", got)
	}
	if got := rsInferTypeName(rsFuncInfo{}, nil); got != "" {
		t.Fatalf("rsInferTypeName(no owner) = %q", got)
	}
}

func TestRustInferDefaultAndReturnValues(t *testing.T) {
	defaults := map[string]string{
		"":              "()",
		"i32":           "0",
		"usize":         "0",
		"f64":           "0.0",
		"bool":          "false",
		"char":          "'a'",
		"String":        "\"test\".to_string()",
		"&str":          "\"test\"",
		"Option<i32>":   "None",
		"Vec<String>":   "vec![]",
		"HashMap<K,V>":  "std::collections::HashMap::new()",
		"()":            "()",
		"MyType":        "MyType::default()",
		"crate::Config": "crate::Config::default()",
		"builder":       "builder::new()",
	}
	for typ, want := range defaults {
		t.Run("default_"+typ, func(t *testing.T) {
			if got := rsInferDefaultValue(typ); got != want {
				t.Fatalf("rsInferDefaultValue(%q) = %q, want %q", typ, got, want)
			}
		})
	}

	returns := map[string]string{
		"":               "()",
		"i32":            "0",
		"f32":            "0.0",
		"bool":           "true",
		"String":         "\"test\".to_string()",
		"Option<String>": "Some(0)",
		"Result<i32,E>":  "Ok(0)",
		"Custom":         "()",
	}
	for typ, want := range returns {
		t.Run("return_"+typ, func(t *testing.T) {
			if got := rsInferReturnValue(typ); got != want {
				t.Fatalf("rsInferReturnValue(%q) = %q, want %q", typ, got, want)
			}
		})
	}
}

func findJavaFunc(funcs []javaFuncInfo, name string) *javaFuncInfo {
	for i := range funcs {
		if funcs[i].Name == name {
			return &funcs[i]
		}
	}
	return nil
}

func findRustFunc(funcs []rsFuncInfo, name string) *rsFuncInfo {
	for i := range funcs {
		if funcs[i].Name == name {
			return &funcs[i]
		}
	}
	return nil
}

func findRustStruct(structs []rsStructInfo, name string) *rsStructInfo {
	for i := range structs {
		if structs[i].Name == name {
			return &structs[i]
		}
	}
	return nil
}
