package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type decisionManifest struct {
	Schema        string                 `json:"$schema"`
	SchemaVersion int                    `json:"schema_version"`
	Fixtures      []decisionFixtureEntry `json:"fixtures"`
}

type decisionFixtureEntry struct {
	Path             string `json:"path"`
	Status           string `json:"status"`
	Action           string `json:"action"`
	ExpectedDecision string `json:"expected_decision"`
}

type validationSample struct {
	Name      string          `json:"-"`
	Status    string          `json:"status"`
	Action    string          `json:"action"`
	RunResult json.RawMessage `json:"run_result,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "agent decision demo failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	samples, err := loadFixtureSamples("docs/fixtures")
	if err != nil {
		return err
	}
	if len(samples) == 0 {
		return fmt.Errorf("no fixture samples found")
	}

	decisions := make([]string, 0, len(samples))
	for i, sample := range samples {
		decision := agentDecision(sample)
		reason := actionReason(sample)
		decisions = append(decisions, decision)
		if reason != "" {
			fmt.Printf("%d. fixture=%s status=%s action=%s decision=%s reason=%q\n", i+1, sample.Name, sample.Status, sample.Action, decision, reason)
			continue
		}
		fmt.Printf("%d. fixture=%s status=%s action=%s decision=%s\n", i+1, sample.Name, sample.Status, sample.Action, decision)
	}
	fmt.Printf("agent_decisions=%s\n", strings.Join(decisions, ","))
	return nil
}

func loadFixtureSamples(dir string) ([]validationSample, error) {
	manifestPath := filepath.Join(dir, "agent-decision-fixtures.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest decisionManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("%s invalid JSON: %w", manifestPath, err)
	}
	if manifest.Schema != "./agent-decision-fixtures.schema.json" {
		return nil, fmt.Errorf("%s $schema must be ./agent-decision-fixtures.schema.json", manifestPath)
	}
	if manifest.SchemaVersion != 1 {
		return nil, fmt.Errorf("%s schema_version must be 1", manifestPath)
	}
	if len(manifest.Fixtures) == 0 {
		return nil, fmt.Errorf("%s fixtures must not be empty", manifestPath)
	}

	samples := make([]validationSample, 0, len(manifest.Fixtures))
	for _, entry := range manifest.Fixtures {
		if entry.Path == "" {
			return nil, fmt.Errorf("%s contains fixture without path", manifestPath)
		}
		path := filepath.FromSlash(entry.Path)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var sample validationSample
		if err := json.Unmarshal(data, &sample); err != nil {
			return nil, fmt.Errorf("%s invalid JSON: %w", path, err)
		}
		if sample.Status != entry.Status || sample.Action != entry.Action {
			return nil, fmt.Errorf("%s status/action = %s/%s, manifest wants %s/%s", path, sample.Status, sample.Action, entry.Status, entry.Action)
		}
		decision := agentDecision(sample)
		if decision != entry.ExpectedDecision {
			return nil, fmt.Errorf("%s decision = %s, manifest wants %s", path, decision, entry.ExpectedDecision)
		}
		name, err := filepath.Rel(dir, path)
		if err != nil {
			return nil, err
		}
		sample.Name = filepath.ToSlash(name)
		samples = append(samples, sample)
	}
	return samples, nil
}

func agentDecision(sample validationSample) string {
	switch {
	case sample.Status == "passed" && sample.Action == "ready":
		return "accept"
	case strings.HasPrefix(sample.Action, "manual_review_"):
		return "manual-review"
	case sample.Action == "apply_fix_suggestions":
		return "apply-repair"
	case sample.Action == "needs_better_input":
		return "needs-better-input"
	case sample.Status == "generation_error":
		return "inspect-generation"
	case sample.Status == "run_error":
		return "inspect-runner"
	case sample.Status == "failed":
		return "repair-generated-test"
	default:
		return "inspect"
	}
}

func actionReason(sample validationSample) string {
	if len(sample.Metadata) == 0 {
		return ""
	}
	var metadata map[string]any
	if err := json.Unmarshal(sample.Metadata, &metadata); err != nil {
		return ""
	}
	for _, key := range []string{
		"next_action_reason",
		"manual_review_reason",
		"needs_better_input_reason",
		"coverage_miss_reason",
		"external_service_reason",
		"environment_reason",
		"internal_reason",
	} {
		if reason, ok := metadata[key].(string); ok && strings.TrimSpace(reason) != "" {
			return strings.TrimSpace(reason)
		}
	}
	return ""
}
