package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
)

func TestAgentResponseArtifactManifestSchema(t *testing.T) {
	resolved := loadAgentResponseManifestSchema(t)
	manifest := loadJSONMap(t, "../docs/fixtures/agent-response-artifact-manifest.json")

	if err := resolved.Validate(manifest); err != nil {
		t.Fatalf("manifest should validate against schema: %v", err)
	}

	t.Run("rejects unsupported schema version", func(t *testing.T) {
		candidate := cloneJSONMap(t, manifest)
		candidate["schema_version"] = float64(2)
		if err := resolved.Validate(candidate); err == nil {
			t.Fatalf("expected schema validation to reject schema_version=2")
		}
	})

	t.Run("rejects missing artifact file pointer", func(t *testing.T) {
		candidate := cloneJSONMap(t, manifest)
		artifact := firstManifestArtifact(t, candidate)
		delete(artifact, "agent_response")
		if err := resolved.Validate(candidate); err == nil {
			t.Fatalf("expected schema validation to reject missing agent_response")
		}
	})

	t.Run("rejects missing summary schema pointer", func(t *testing.T) {
		candidate := cloneJSONMap(t, manifest)
		delete(candidate, "summary_schema")
		if err := resolved.Validate(candidate); err == nil {
			t.Fatalf("expected schema validation to reject missing summary_schema")
		}
	})

	t.Run("rejects missing expected section signals", func(t *testing.T) {
		candidate := cloneJSONMap(t, manifest)
		artifact := firstManifestArtifact(t, candidate)
		delete(artifact, "expected_section_signals")
		if err := resolved.Validate(candidate); err == nil {
			t.Fatalf("expected schema validation to reject missing expected_section_signals")
		}
	})

	t.Run("rejects invalid artifact kind", func(t *testing.T) {
		candidate := cloneJSONMap(t, manifest)
		artifact := firstManifestArtifact(t, candidate)
		artifact["kind"] = "smoke"
		if err := resolved.Validate(candidate); err == nil {
			t.Fatalf("expected schema validation to reject invalid kind")
		}
	})

	t.Run("rejects fallback order without agent response first", func(t *testing.T) {
		candidate := cloneJSONMap(t, manifest)
		artifact := firstManifestArtifact(t, candidate)
		artifact["fallback_order"] = []any{"agent-decision.txt", "agent-response.txt"}
		if err := resolved.Validate(candidate); err == nil {
			t.Fatalf("expected schema validation to reject fallback_order without agent-response.txt first")
		}
	})
}

func loadAgentResponseManifestSchema(t *testing.T) *jsonschema.Resolved {
	t.Helper()

	data, err := os.ReadFile(filepath.FromSlash("../docs/fixtures/agent-response-artifact-manifest.schema.json"))
	if err != nil {
		t.Fatalf("read manifest schema: %v", err)
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("decode manifest schema: %v", err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("resolve manifest schema: %v", err)
	}
	return resolved
}

func loadJSONMap(t *testing.T, path string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(filepath.FromSlash(path))
	if err != nil {
		t.Fatalf("read JSON %s: %v", path, err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode JSON %s: %v", path, err)
	}
	return value
}

func cloneJSONMap(t *testing.T, value map[string]any) map[string]any {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("clone JSON map: marshal: %v", err)
	}
	var cloned map[string]any
	if err := json.Unmarshal(data, &cloned); err != nil {
		t.Fatalf("clone JSON map: unmarshal: %v", err)
	}
	return cloned
}

func firstManifestArtifact(t *testing.T, manifest map[string]any) map[string]any {
	t.Helper()

	artifacts, ok := manifest["artifacts"].([]any)
	if !ok || len(artifacts) == 0 {
		t.Fatalf("manifest artifacts should be a non-empty array")
	}
	artifact, ok := artifacts[0].(map[string]any)
	if !ok {
		t.Fatalf("manifest first artifact should be an object")
	}
	return artifact
}
