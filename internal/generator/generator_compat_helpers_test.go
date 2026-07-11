package generator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/sleticalboy/testloop-mcp/types"
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

func TestGoGeneratorTypeAndValueHelpers(t *testing.T) {
	if got := substituteType("map[K][]V", map[string]string{"K": "string", "V": "int"}); got != "map[string][]int" {
		t.Fatalf("substituteType() = %q", got)
	}

	zeroValues := map[string]string{
		"int":            "0",
		"uint64":         "0",
		"float64":        "0",
		"string":         "\"\"",
		"bool":           "false",
		"error":          "nil",
		"any":            "nil",
		"interface{}":    "nil",
		"chan int":       "nil",
		"*User":          "nil",
		"[]string":       "nil",
		"map[string]int": "nil",
		"func()":         "nil",
		"<-chan string":  "nil",
		"CustomResponse": "*new(CustomResponse)",
		"time.Duration":  "*new(time.Duration)",
	}
	for typ, want := range zeroValues {
		t.Run("zero_"+typ, func(t *testing.T) {
			if got := zeroValue(typ); got != want {
				t.Fatalf("zeroValue(%q) = %q, want %q", typ, got, want)
			}
		})
	}

	argValues := []struct {
		param paramInfo
		want  string
	}{
		{paramInfo{Name: "ok", Type: "bool"}, "true"},
		{paramInfo{Name: "name", Type: "string"}, "\"test\""},
		{paramInfo{Name: "y", Type: "float64"}, "2.0"},
		{paramInfo{Name: "x", Type: "float64"}, "1.0"},
		{paramInfo{Name: "b", Type: "int"}, "2"},
		{paramInfo{Name: "a", Type: "uint"}, "1"},
		{paramInfo{Name: "items", Type: "[]string"}, "nil"},
	}
	for _, tt := range argValues {
		t.Run(tt.param.Name+"_"+tt.param.Type, func(t *testing.T) {
			if got := goArgValue(tt.param, 0); got != tt.want {
				t.Fatalf("goArgValue(%+v) = %q, want %q", tt.param, got, tt.want)
			}
		})
	}
}

func TestGoGeneratorExpressionHelpers(t *testing.T) {
	exprs := map[string]string{
		"value":                           "value",
		"42":                              "42",
		"a + b":                           "a + b",
		"-count":                          "-count",
		"(a)":                             "(a)",
		"pkg.Value":                       "pkg.Value",
		"*User":                           "*User",
		"[]string":                        "[]string",
		"map[string]int":                  "map[string]int",
		"chan int":                        "chan int",
		"chan<- int":                      "chan<- int",
		"<-chan string":                   "<-chan string",
		"interface{}":                     "any",
		"func()":                          "func()",
		"func(int) int":                   "func(int) int",
		"func(i int) string":              "func(i int) string",
		"Box[int]":                        "Box[int]",
		"Pair[int, string]":               "Pair[int, string]",
		"make([]int, 0)":                  "make([]int, 0)",
		`time.Now().Format("2006-01-02")`: `time.Now().Format("2006-01-02")`,
		"time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())": "time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())",
	}
	for src, want := range exprs {
		t.Run(src, func(t *testing.T) {
			expr, err := parser.ParseExpr(src)
			if err != nil {
				t.Fatalf("ParseExpr(%q): %v", src, err)
			}
			if got := exprToString(expr); got != want {
				t.Fatalf("exprToString(%q) = %q, want %q", src, got, want)
			}
		})
	}
	if got := exprToString(&ast.Ellipsis{Elt: ast.NewIdent("string")}); got != "...string" {
		t.Fatalf("exprToString(ellipsis) = %q", got)
	}
	if !goTimeDateZeroExpr("time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())") {
		t.Fatal("expected time.Date expression with zero clock components to match")
	}
	if goTimeDateZeroExpr("time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, now.Location())") {
		t.Fatal("expected non-zero hour time.Date expression not to match")
	}
	branchFn := funcInfo{
		Name:    "Add",
		Params:  []paramInfo{{Name: "a", Type: "int"}, {Name: "b", Type: "int"}},
		Returns: []paramInfo{{Name: "ret0", Type: "int"}},
		Boundaries: []goBoundary{
			{Param: "a", Op: "==", Value: "0", Condition: "a == 0", ReturnExpr: "b"},
		},
	}
	seed, ok := goSeedTestCase(branchFn, &types.CoverageTestTask{
		GapType:         "branch",
		SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入"},
	})
	if !ok || seed.Inputs["a"] != "0" || seed.Inputs["b"] != "2" || seed.Outputs["ret0"] != "2" {
		t.Fatalf("branch seed = %+v, %v; want a=0 b=2 ret0=2", seed, ok)
	}
	greater, ok := goBoundaryInputValue(goBoundary{Param: "count", Op: ">", Value: "3"}, "int")
	if !ok || greater != "4" {
		t.Fatalf("goBoundaryInputValue(count > 3) = %q, %v; want 4 true", greater, ok)
	}
	if op, ok := invertGoBoundaryOperator("<"); !ok || op != ">" {
		t.Fatalf("invertGoBoundaryOperator(<) = %q, %v; want > true", op, ok)
	}
	if got, ok := goBoundaryInputValue(goBoundary{Param: "user", Op: "==", Value: "nil"}, "*User"); !ok || got != "nil" {
		t.Fatalf("goBoundaryInputValue(user == nil) = %q, %v; want nil true", got, ok)
	}
	if got, ok := goBoundaryInputValue(goBoundary{Param: "user", Op: "!=", Value: "nil"}, "*User"); !ok || got != "&User{}" {
		t.Fatalf("goBoundaryInputValue(user != nil) = %q, %v; want &User{} true", got, ok)
	}
	if got, ok := goBoundaryInputValue(goBoundary{Param: "err", Op: "==", Value: "nil"}, "error"); !ok || got != "nil" {
		t.Fatalf("goBoundaryInputValue(err == nil) = %q, %v; want nil true", got, ok)
	}
	if got, ok := goBoundaryInputValue(goBoundary{Param: "err", Op: "!=", Value: "nil"}, "error"); !ok || got != `errors.New("test")` {
		t.Fatalf("goBoundaryInputValue(err != nil) = %q, %v; want errors.New true", got, ok)
	}
	if !goSeedCaseUsesPackage(goSeedCase{Inputs: map[string]string{"err": `errors.New("test")`}}, "errors") {
		t.Fatal("goSeedCaseUsesPackage did not detect errors input")
	}
	if got, ok := goCompoundParamInputValue([]goBoundary{
		{Param: "a", Op: ">", Value: "0"},
		{Param: "a", Op: "<", Value: "10"},
	}, "int"); !ok || got != "1" {
		t.Fatalf("goCompoundParamInputValue(0<a<10) = %q, %v; want 1 true", got, ok)
	}
	if got, ok := goCompoundParamInputValue([]goBoundary{
		{Param: "a", Op: ">", Value: "10"},
		{Param: "a", Op: "<", Value: "5"},
	}, "int"); ok || got != "" {
		t.Fatalf("goCompoundParamInputValue(a>10 && a<5) = %q, %v; want empty false", got, ok)
	}
	if got := goTestCaseFieldName("name"); got != "nameValue" {
		t.Fatalf("goTestCaseFieldName(name) = %q, want nameValue", got)
	}
	if got := goTestCaseFieldName("skip"); got != "skipValue" {
		t.Fatalf("goTestCaseFieldName(skip) = %q, want skipValue", got)
	}

	cases := map[string]string{
		"branch":      "coverage branch gap",
		"error_path":  "coverage error path",
		"return_path": "coverage return path",
		"statement":   "coverage statement gap",
		"other":       "coverage gap",
	}
	for gapType, want := range cases {
		if got := goCoverageTaskCaseName(&types.CoverageTestTask{GapType: gapType}, "fallback"); got != want {
			t.Fatalf("goCoverageTaskCaseName(%q) = %q, want %q", gapType, got, want)
		}
	}
	if got := goCoverageTaskCaseName(nil, "fallback"); got != "fallback" {
		t.Fatalf("goCoverageTaskCaseName(nil) = %q", got)
	}
}

func TestGoSeedAndReturnSafetyBranches(t *testing.T) {
	for input, want := range map[string]bool{
		"":          false,
		"a + b":     true,
		"items[0]":  false,
		"fn(a)":     false,
		"obj.Value": false,
		"ch <- v":   false,
		"chan int":  false,
	} {
		if got := goReturnExprIsSafe(input); got != want {
			t.Fatalf("goReturnExprIsSafe(%q) = %v, want %v", input, got, want)
		}
	}

	file, err := parser.ParseFile(token.NewFileSet(), "sample.go", `package sample

func Add(a, b int) int {
	return a + b
}

func Empty() {
}

func Multi(a int) int {
	a++
	return a
}

func Pair() (int, int) {
	return 1, 2
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	funcs := map[string]*ast.BlockStmt{}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			funcs[fn.Name.Name] = fn.Body
		}
	}
	if got := singleReturnExpr(funcs["Add"]); got != "a + b" {
		t.Fatalf("singleReturnExpr(Add) = %q", got)
	}
	for _, name := range []string{"Empty", "Multi", "Pair"} {
		if got := singleReturnExpr(funcs[name]); got != "" {
			t.Fatalf("singleReturnExpr(%s) = %q", name, got)
		}
	}
	if got := singleReturnExpr(nil); got != "" {
		t.Fatalf("singleReturnExpr(nil) = %q", got)
	}

	if _, ok := goSeedTestCase(funcInfo{Name: "Method", IsMethod: true}, nil); ok {
		t.Fatal("method should not get exact seed case")
	}
	if _, ok := goSeedTestCase(funcInfo{Name: "Variadic", IsVariadic: true}, nil); ok {
		t.Fatal("variadic function should not get exact seed case")
	}
	if _, ok := goSeedTestCase(funcInfo{Name: "Err", Returns: []paramInfo{{Name: "err", Type: "error"}}}, nil); ok {
		t.Fatal("error return should not get exact seed case")
	}
	if _, ok := goSeedTestCase(funcInfo{
		Name:       "Unsafe",
		Params:     []paramInfo{{Name: "items", Type: "[]int"}},
		Returns:    []paramInfo{{Name: "ret0", Type: "[]int"}},
		ReturnExpr: "items",
	}, nil); ok {
		t.Fatal("unsupported exact seed type should not get seed case")
	}
	if _, ok := goSeedTestCase(funcInfo{
		Name:       "Unknown",
		Returns:    []paramInfo{{Name: "ret0", Type: "int"}},
		ReturnExpr: "missing + 1",
	}, nil); ok {
		t.Fatal("unknown identifiers should not get seed case")
	}
}

func TestGoTableDrivenTestBranches(t *testing.T) {
	voidTest := genTableDrivenTestForTask(funcInfo{
		Name: "Notify",
		Params: []paramInfo{
			{Name: "message", Type: "string"},
		},
	}, nil)
	for _, want := range []string{
		"func TestNotify(t *testing.T)",
		"message string",
		"Notify(tt.message)",
	} {
		if !strings.Contains(voidTest, want) {
			t.Fatalf("expected %q in void test:\n%s", want, voidTest)
		}
	}

	errorTest := genTableDrivenTestForTask(funcInfo{
		Name: "Validate",
		Returns: []paramInfo{
			{Name: "err", Type: "error"},
		},
	}, nil)
	for _, want := range []string{
		"func TestValidate(t *testing.T)",
		"err := Validate()",
		"if err != nil",
		"unexpected error",
	} {
		if !strings.Contains(errorTest, want) {
			t.Fatalf("expected %q in error test:\n%s", want, errorTest)
		}
	}

	multiReturnTest := genTableDrivenTestForTask(funcInfo{
		Name: "Lookup",
		Returns: []paramInfo{
			{Name: "items", Type: "[]string"},
			{Name: "err", Type: "error"},
		},
	}, nil)
	for _, want := range []string{
		"got0, got1 := Lookup()",
		"if !reflect.DeepEqual(got0, tt.items)",
		"items: got %v, want %v",
		"if got1 != nil",
	} {
		if !strings.Contains(multiReturnTest, want) {
			t.Fatalf("expected %q in multi-return test:\n%s", want, multiReturnTest)
		}
	}

	taskTest := genTableDrivenTestForTask(funcInfo{
		Name:       "Sum",
		TypeParams: []string{"T"},
		Params: []paramInfo{
			{Name: "values", Type: "[]int", Variadic: true},
		},
		Returns: []paramInfo{
			{Name: "ret0", Type: "int"},
		},
	}, &types.CoverageTestTask{
		ID:        "go-task-2",
		GapType:   "statement",
		TestName:  "TestCustomSumGap",
		LineRange: "10-12",
	})
	for _, want := range []string{
		"func TestCustomSumGap(t *testing.T)",
		"coverage task: go-task-2 | lines 10-12",
		"name: \"coverage statement gap\"",
		"Sum[int](tt.values...)",
	} {
		if !strings.Contains(taskTest, want) {
			t.Fatalf("expected %q in task test:\n%s", want, taskTest)
		}
	}
}

func TestGoFunctionTargetMatching(t *testing.T) {
	if !goFuncMatchesTarget(funcInfo{Name: "Add"}, "Add") {
		t.Fatal("function name should match target")
	}
	if goFuncMatchesTarget(funcInfo{Name: "Add"}, "Calculator.Add") {
		t.Fatal("non-method should not match class-qualified target")
	}

	method := funcInfo{Name: "Save", IsMethod: true, ReceiverType: "*Store"}
	for _, target := range []string{"Store.Save", "Store_Save"} {
		if !goFuncMatchesTarget(method, target) {
			t.Fatalf("method should match target %q", target)
		}
	}
	if goFuncMatchesTarget(method, "Other.Save") {
		t.Fatal("method should not match unrelated receiver")
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
