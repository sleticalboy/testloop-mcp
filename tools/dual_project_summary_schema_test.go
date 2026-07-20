package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
)

func TestDualProjectSummarySchema(t *testing.T) {
	resolved := loadDualProjectSummarySchema(t)

	matches, err := filepath.Glob(filepath.FromSlash("../docs/fixtures/dual-project-summary/*.json"))
	if err != nil {
		t.Fatalf("glob dual project summaries: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("glob dual project summaries matched no fixtures")
	}
	for _, match := range matches {
		t.Run(filepath.ToSlash(match), func(t *testing.T) {
			summary := loadJSONMap(t, match)
			if err := resolved.Validate(summary); err != nil {
				t.Fatalf("%s should validate against dual project summary schema: %v", match, err)
			}
		})
	}

	t.Run("rejects missing second project", func(t *testing.T) {
		candidate := loadJSONMap(t, "../docs/fixtures/dual-project-summary/laoxia-passed.json")
		delete(candidate, "web")
		if err := resolved.Validate(candidate); err == nil {
			t.Fatalf("expected schema validation to reject missing second project")
		}
	})

	t.Run("rejects child summary without sections", func(t *testing.T) {
		candidate := loadJSONMap(t, "../docs/fixtures/dual-project-summary/laoxia-passed.json")
		server := candidate["server"].(map[string]any)
		summary := server["summary"].(map[string]any)
		delete(summary, "sections")
		if err := resolved.Validate(candidate); err == nil {
			t.Fatalf("expected schema validation to reject child summary without sections")
		}
	})
}

func loadDualProjectSummarySchema(t *testing.T) *jsonschema.Resolved {
	t.Helper()

	data, err := os.ReadFile(filepath.FromSlash("../docs/fixtures/dual-project-summary.schema.json"))
	if err != nil {
		t.Fatalf("read dual project summary schema: %v", err)
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("decode dual project summary schema: %v", err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("resolve dual project summary schema: %v", err)
	}
	return resolved
}
