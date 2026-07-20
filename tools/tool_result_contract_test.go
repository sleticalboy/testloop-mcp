package tools

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func assertStructuredContentMatchesTextJSON(t *testing.T, result *mcp.CallToolResult, returned any) {
	t.Helper()
	textPayload := normalizedJSONValue(t, []byte(resultText(t, result)))
	structuredPayload := normalizedStructuredJSONValue(t, result.StructuredContent)
	if !reflect.DeepEqual(structuredPayload, textPayload) {
		t.Fatalf("structuredContent mismatch\nstructured: %#v\ntext: %#v", structuredPayload, textPayload)
	}
	if returned == nil {
		return
	}
	returnedPayload := normalizedStructuredJSONValue(t, returned)
	if !reflect.DeepEqual(returnedPayload, textPayload) {
		t.Fatalf("handler return mismatch\nreturned: %#v\ntext: %#v", returnedPayload, textPayload)
	}
}

func TestPrimaryToolResultsKeepStructuredContentAndTextJSONInSync(t *testing.T) {
	t.Run("parse_results", func(t *testing.T) {
		output := strings.Join([]string{
			`{"Action":"run","Package":"example.com/calc","Test":"TestAdd"}`,
			`{"Action":"pass","Package":"example.com/calc","Test":"TestAdd","Elapsed":0}`,
			`{"Action":"pass","Package":"example.com/calc","Elapsed":0}`,
		}, "\n")
		result, returned, err := HandleParseResults(context.Background(), nil, parseResultsInput{Output: output})
		if err != nil {
			t.Fatalf("HandleParseResults returned error: %v", err)
		}
		assertStructuredContentMatchesTextJSON(t, result, returned)
	})

	t.Run("parse_coverage", func(t *testing.T) {
		result, returned, err := HandleParseCoverage(context.Background(), nil, parseCoverageInput{
			Data:      "mode: set\ncalc.go:1.1,2.1 1 1\ncalc.go:3.1,4.1 1 0\n",
			Framework: "go-test",
		})
		if err != nil {
			t.Fatalf("HandleParseCoverage returned error: %v", err)
		}
		assertStructuredContentMatchesTextJSON(t, result, returned)
	})

	t.Run("generate_tests", func(t *testing.T) {
		dir := t.TempDir()
		source := writeTestFile(t, dir, "calc.go", strings.Join([]string{
			"package calc",
			"",
			"func Add(a, b int) int {",
			"	return a + b",
			"}",
		}, "\n")+"\n")
		result, returned, err := HandleGenerateTests(context.Background(), nil, generateTestsInput{FilePath: source})
		if err != nil {
			t.Fatalf("HandleGenerateTests returned error: %v", err)
		}
		assertStructuredContentMatchesTextJSON(t, result, returned)
	})

	t.Run("run_tests", func(t *testing.T) {
		dir := t.TempDir()
		writeTestFile(t, dir, "go.mod", "module example.com/calc\n\ngo 1.23\n")
		writeTestFile(t, dir, "calc.go", strings.Join([]string{
			"package calc",
			"",
			"func Add(a, b int) int {",
			"	return a + b",
			"}",
		}, "\n")+"\n")
		writeTestFile(t, dir, "calc_test.go", strings.Join([]string{
			"package calc",
			"",
			"import \"testing\"",
			"",
			"func TestAdd(t *testing.T) {",
			"	if got := Add(1, 2); got != 3 {",
			"		t.Fatalf(\"got %d, want 3\", got)",
			"	}",
			"}",
		}, "\n")+"\n")
		result, returned, err := HandleRunTests(context.Background(), nil, runTestsInput{
			Path:      dir,
			Framework: "go-test",
		})
		if err != nil {
			t.Fatalf("HandleRunTests returned error: %v", err)
		}
		assertStructuredContentMatchesTextJSON(t, result, returned)
	})

	t.Run("fix_suggestions", func(t *testing.T) {
		dir := t.TempDir()
		source := writeTestFile(t, dir, "calc.go", strings.Join([]string{
			"package calc",
			"",
			"func Add(a, b int) int {",
			"	return a + b",
			"}",
		}, "\n")+"\n")
		failures := `[{"test":"TestAdd","file":"calc_test.go","line":6,"error":"got 4, want 3"}]`
		result, returned, err := HandleFixSuggestions(context.Background(), nil, fixSuggestionsInput{
			Failures:   failures,
			SourceCode: source,
		})
		if err != nil {
			t.Fatalf("HandleFixSuggestions returned error: %v", err)
		}
		assertStructuredContentMatchesTextJSON(t, result, returned)
	})
}
