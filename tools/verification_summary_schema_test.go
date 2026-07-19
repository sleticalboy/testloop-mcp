package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
)

func TestVerificationSummarySchema(t *testing.T) {
	resolved := loadVerificationSummarySchema(t)

	for _, pattern := range []string{
		"../docs/fixtures/verification-summary/*.json",
		"../docs/fixtures/first-run-artifacts/*/verification-summary.json",
		"../docs/fixtures/onboarding-artifacts/*/verification-summary.json",
	} {
		matches, err := filepath.Glob(filepath.FromSlash(pattern))
		if err != nil {
			t.Fatalf("glob %s: %v", pattern, err)
		}
		if len(matches) == 0 {
			t.Fatalf("glob %s matched no verification summaries", pattern)
		}
		for _, match := range matches {
			t.Run(filepath.ToSlash(match), func(t *testing.T) {
				summary := loadJSONMap(t, match)
				if err := resolved.Validate(summary); err != nil {
					t.Fatalf("%s should validate against verification summary schema: %v", match, err)
				}
			})
		}
	}

	t.Run("accepts section action signal", func(t *testing.T) {
		candidate := loadJSONMap(t, "../docs/fixtures/verification-summary/user-project-failed.json")
		candidate["sections"] = append(candidate["sections"].([]any), map[string]any{
			"name":      "独立 CLI 生成动作 smoke",
			"status":    "passed",
			"exit_code": float64(0),
			"reason":    nil,
			"signals": map[string]any{
				"action": "manual_review",
			},
		})
		if err := resolved.Validate(candidate); err != nil {
			t.Fatalf("summary with action signal should validate: %v", err)
		}
	})

	t.Run("rejects non string signal", func(t *testing.T) {
		candidate := loadJSONMap(t, "../docs/fixtures/verification-summary/user-project-failed.json")
		candidate["sections"] = append(candidate["sections"].([]any), map[string]any{
			"name":      "独立 CLI 生成动作 smoke",
			"status":    "passed",
			"exit_code": float64(0),
			"reason":    nil,
			"signals": map[string]any{
				"action": false,
			},
		})
		if err := resolved.Validate(candidate); err == nil {
			t.Fatalf("expected schema validation to reject non-string action signal")
		}
	})
}

func loadVerificationSummarySchema(t *testing.T) *jsonschema.Resolved {
	t.Helper()

	data, err := os.ReadFile(filepath.FromSlash("../docs/fixtures/verification-summary.schema.json"))
	if err != nil {
		t.Fatalf("read verification summary schema: %v", err)
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("decode verification summary schema: %v", err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("resolve verification summary schema: %v", err)
	}
	return resolved
}
