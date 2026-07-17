package types

import (
	"reflect"
	"strings"
	"testing"
)

func TestAgentJSONContractFields(t *testing.T) {
	tests := []struct {
		name   string
		typ    reflect.Type
		fields []string
	}{
		{
			name: "TestResult",
			typ:  reflect.TypeOf(TestResult{}),
			fields: []string{
				"status",
				"framework",
				"total",
				"passed",
				"failed",
				"skipped",
				"coverage_percent",
				"failures",
				"fix_suggestions",
				"raw_output",
			},
		},
		{
			name: "TestFailure",
			typ:  reflect.TypeOf(TestFailure{}),
			fields: []string{
				"test_name",
				"file",
				"line",
				"column",
				"error",
				"expected",
				"received",
			},
		},
		{
			name: "FixSuggestion",
			typ:  reflect.TypeOf(FixSuggestion{}),
			fields: []string{
				"file",
				"line",
				"issue",
				"category",
				"context_file",
				"context_line",
				"suggested_fix",
				"confidence",
				"repair_task",
			},
		},
		{
			name: "RepairTask",
			typ:  reflect.TypeOf(RepairTask{}),
			fields: []string{
				"id",
				"test_name",
				"category",
				"issue",
				"target_file",
				"target_line",
				"context_file",
				"context_line",
				"context_snippet",
				"editable_files",
				"suggested_commands",
				"assertion_focus",
			},
		},
		{
			name: "GenerateTestsOutput",
			typ:  reflect.TypeOf(GenerateTestsOutput{}),
			fields: []string{
				"status",
				"test_file",
				"generated_cases",
				"preview",
				"context",
				"coverage_task",
				"provider",
				"error",
				"provider_error",
			},
		},
		{
			name: "ProviderErrorOutput",
			typ:  reflect.TypeOf(ProviderErrorOutput{}),
			fields: []string{
				"kind",
				"action",
				"provider",
				"message",
			},
		},
		{
			name: "CoverageTaskValidationOutput",
			typ:  reflect.TypeOf(CoverageTaskValidationOutput{}),
			fields: []string{
				"status",
				"action",
				"coverage_task",
				"generated",
				"run_result",
				"error",
				"provider_error",
				"metadata",
			},
		},
		{
			name: "CoverageReport",
			typ:  reflect.TypeOf(CoverageReport{}),
			fields: []string{
				"framework",
				"total_percent",
				"files",
				"summary",
				"suggestions",
				"test_tasks",
			},
		},
		{
			name: "CoverageTestTask",
			typ:  reflect.TypeOf(CoverageTestTask{}),
			fields: []string{
				"id",
				"framework",
				"file",
				"target",
				"kind",
				"line_range",
				"gap_type",
				"missing_branches",
				"uncovered_lines",
				"suggested_inputs",
				"goal",
				"command",
				"test_file",
				"test_name",
				"assertion_focus",
				"priority",
				"priority_reason",
				"confidence",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireJSONFields(t, tt.typ, tt.fields)
		})
	}
}

func requireJSONFields(t *testing.T, typ reflect.Type, fields []string) {
	t.Helper()
	available := map[string]reflect.StructField{}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonName := jsonFieldName(field)
		if jsonName == "" {
			continue
		}
		available[jsonName] = field
	}
	for _, field := range fields {
		if _, ok := available[field]; !ok {
			t.Fatalf("%s missing stable json field %q; available=%v", typ.Name(), field, sortedMapKeys(available))
		}
	}
}

func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return ""
	}
	if tag == "" {
		return field.Name
	}
	name := strings.Split(tag, ",")[0]
	if name == "" {
		return field.Name
	}
	return name
}

func sortedMapKeys(values map[string]reflect.StructField) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
