package generator

import (
	"testing"

	"github.com/sleticalboy/testloop-mcp/types"
)

func TestTaskTargetMatches(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		className string
		funcName  string
		want      bool
	}{
		{name: "empty target", target: "", className: "Calculator", funcName: "add", want: true},
		{name: "function name", target: "add", className: "Calculator", funcName: "add", want: true},
		{name: "class name", target: "Calculator", className: "Calculator", funcName: "add", want: true},
		{name: "class dot method", target: "Calculator.add", className: "Calculator", funcName: "add", want: true},
		{name: "class underscore method", target: "Calculator_add", className: "Calculator", funcName: "add", want: true},
		{name: "qualified class dot method", target: "pkg.Calculator.add", className: "Calculator", funcName: "add", want: true},
		{name: "no class mismatch", target: "Calculator.add", className: "", funcName: "add", want: false},
		{name: "different name", target: "sub", className: "Calculator", funcName: "add", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := taskTargetMatches(tt.target, tt.className, tt.funcName); got != tt.want {
				t.Fatalf("taskTargetMatches(%q, %q, %q) = %v, want %v", tt.target, tt.className, tt.funcName, got, tt.want)
			}
		})
	}
}

func TestCoverageTaskInputValuesExtractsAndNormalizesHints(t *testing.T) {
	task := types.CoverageTestTask{
		SuggestedInputs: []string{
			"构造满足条件 `value == None` 和 `enabled == false` 的输入",
			"mode = 'short';",
			"构造满足条件 `positive > 0` 的输入",
			"构造满足条件 `offset < 10` 的输入",
			"构造满足条件 `limit >= 5` 的输入",
			"构造满足条件 `total <= 99` 的输入",
			"构造满足条件 `state != \"ready\"` 的输入",
			"构造满足条件 `missing is not None` 的输入",
			"构造满足条件 `left > 0 && right < 10` 的输入",
			"构造满足条件 `py_left > 1 and py_right <= 7` 的输入",
			"构造满足条件 `!disabled` 的输入",
			"构造满足条件 `not archived` 的输入",
			"构造满足条件 `visible` 的输入",
			"invalid text without condition",
		},
		MissingBranches: []string{
			"`count === 0`",
			"`fallback is undefined`",
		},
	}

	jsValues := coverageTaskInputValues(&task, "javascript")
	if jsValues["value"] != "null" || jsValues["enabled"] != "false" || jsValues["mode"] != "'short'" ||
		jsValues["count"] != "0" || jsValues["positive"] != "1" || jsValues["offset"] != "9" || jsValues["limit"] != "5" ||
		jsValues["total"] != "99" || jsValues["state"] != `"__testloop_other__"` ||
		jsValues["missing"] != "{}" || jsValues["fallback"] != "undefined" ||
		jsValues["left"] != "1" || jsValues["right"] != "9" || jsValues["py_left"] != "2" || jsValues["py_right"] != "7" ||
		jsValues["disabled"] != "false" || jsValues["archived"] != "false" || jsValues["visible"] != "true" {
		t.Fatalf("unexpected JavaScript values: %+v", jsValues)
	}

	pyValues := coverageTaskInputValues(&task, "python")
	if pyValues["value"] != "None" || pyValues["enabled"] != "False" || pyValues["missing"] != "object()" ||
		pyValues["offset"] != "9" || pyValues["state"] != `"__testloop_other__"` ||
		pyValues["left"] != "1" || pyValues["right"] != "9" || pyValues["py_left"] != "2" || pyValues["py_right"] != "7" ||
		pyValues["disabled"] != "False" || pyValues["archived"] != "False" || pyValues["visible"] != "True" {
		t.Fatalf("unexpected Python values: %+v", pyValues)
	}

	rsValues := coverageTaskInputValues(&task, "rust")
	if rsValues["value"] != "None" || rsValues["missing"] != "Some(0)" || rsValues["enabled"] != "false" ||
		rsValues["positive"] != "1" {
		t.Fatalf("unexpected Rust values: %+v", rsValues)
	}

	javaValues := coverageTaskInputValues(&task, "java")
	if javaValues["value"] != "null" || javaValues["missing"] != `"__testloop_other__"` ||
		javaValues["enabled"] != "false" || javaValues["positive"] != "1" {
		t.Fatalf("unexpected Java values: %+v", javaValues)
	}

	equalsTask := types.CoverageTestTask{
		MissingBranches: []string{"未覆盖 if 分支: AddressScheme.DOMAIN_NAME.equals(scheme"},
	}
	javaValues = coverageTaskInputValues(&equalsTask, "java")
	if javaValues["scheme"] != "AddressScheme.DOMAIN_NAME" {
		t.Fatalf("unexpected Java equals-call values: %+v", javaValues)
	}

	if got := coverageTaskInputValues(nil, "python"); len(got) != 0 {
		t.Fatalf("nil task should produce empty values, got %+v", got)
	}
}

func TestTaskConditionCandidates(t *testing.T) {
	got := taskConditionCandidates("try `a == 0`, then `b === undefined`")
	if len(got) != 2 || got[0] != "a == 0" || got[1] != "b === undefined" {
		t.Fatalf("taskConditionCandidates() = %+v", got)
	}

	got = taskConditionCandidates("  value is None  ")
	if len(got) != 1 || got[0] != "value is None" {
		t.Fatalf("taskConditionCandidates() fallback = %+v", got)
	}
}

func TestNormalizeTaskLiteral(t *testing.T) {
	tests := []struct {
		value    string
		language string
		want     string
	}{
		{value: " undefined, ", language: "javascript", want: "undefined"},
		{value: "undefined;", language: "python", want: "None"},
		{value: "undefined:", language: "java", want: "null"},
		{value: "null", language: "rust", want: "None"},
		{value: "none", language: "python", want: "None"},
		{value: "true", language: "python", want: "True"},
		{value: "false", language: "javascript", want: "false"},
		{value: "'short'", language: "python", want: "'short'"},
	}
	for _, tt := range tests {
		t.Run(tt.value+"_"+tt.language, func(t *testing.T) {
			if got := normalizeTaskLiteral(tt.value, tt.language); got != tt.want {
				t.Fatalf("normalizeTaskLiteral(%q, %q) = %q, want %q", tt.value, tt.language, got, tt.want)
			}
		})
	}
}

func TestSanitizePythonTestName(t *testing.T) {
	tests := map[string]string{
		"":                         "test_fallback",
		"def test_already_valid()": "test_already_valid",
		"covers add zero operand":  "test_covers_add_zero_operand",
		"123 invalid name":         "test_invalid_name",
		"---":                      "test_fallback",
		"test_keeps_existing_name": "test_keeps_existing_name",
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			if got := sanitizePythonTestName(input, "fallback"); got != want {
				t.Fatalf("sanitizePythonTestName(%q) = %q, want %q", input, got, want)
			}
		})
	}
}
