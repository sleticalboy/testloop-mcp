package tools

import (
	"encoding/json"
	"os"
	"os/exec"
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

func TestDualProjectSummaryScriptOutputMatchesSchema(t *testing.T) {
	resolved := loadDualProjectSummarySchema(t)
	tmpDir := t.TempDir()

	fakeBinary := filepath.Join(tmpDir, "testloop-mcp")
	if err := os.WriteFile(fakeBinary, []byte(`#!/usr/bin/env sh
case "${1:-}" in
  --version)
    echo "testloop-mcp 0.5.13"
    ;;
  *)
    echo "fake testloop-mcp"
    ;;
esac
`), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	apiDir := filepath.Join(tmpDir, "api")
	webDir := filepath.Join(tmpDir, "web")
	outputDir := filepath.Join(tmpDir, "artifacts")
	for _, dir := range []string{apiDir, webDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	cmd := exec.Command("bash", filepath.FromSlash("../scripts/showcase-dual-project-report.sh"), fakeBinary)
	cmd.Env = append(os.Environ(),
		"TESTLOOP_PAIR_PREFIX=pair",
		"TESTLOOP_PAIR_OUTPUT_DIR="+outputDir,
		"TESTLOOP_PAIR_FIRST_NAME=api",
		"TESTLOOP_PAIR_FIRST_DIR="+apiDir,
		"TESTLOOP_PAIR_FIRST_COMMAND=printf \"api smoke ok\\n\"",
		"TESTLOOP_PAIR_SECOND_NAME=web",
		"TESTLOOP_PAIR_SECOND_DIR="+webDir,
		"TESTLOOP_PAIR_SECOND_COMMAND=printf \"web smoke ok\\n\"",
		"TESTLOOP_REPORT_SKIP_BASIC=true",
		"TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true",
		"TESTLOOP_REPORT_SKIP_AGENT_DEMO=true",
		"TESTLOOP_REPORT_SKIP_TESTGEN_SMOKE=true",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("showcase-dual-project-report.sh failed: %v\n%s", err, output)
	}

	summaryPath := filepath.Join(outputDir, "pair-summary.json")
	summary := loadJSONMap(t, summaryPath)
	if err := resolved.Validate(summary); err != nil {
		t.Fatalf("generated pair summary should validate against dual project summary schema: %v\n%s", err, output)
	}
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
