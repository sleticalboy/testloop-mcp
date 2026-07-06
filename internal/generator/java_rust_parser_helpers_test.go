package generator

import "testing"

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
