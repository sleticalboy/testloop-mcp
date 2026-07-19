package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

type manifest struct {
	SchemaVersion int        `json:"schema_version"`
	SummarySchema string     `json:"summary_schema"`
	Artifacts     []artifact `json:"artifacts"`
}

type artifact struct {
	Kind                   string          `json:"kind"`
	Directory              string          `json:"directory"`
	AgentResponse          string          `json:"agent_response"`
	Decision               string          `json:"decision"`
	Summary                string          `json:"summary"`
	Report                 string          `json:"report"`
	OptionalContext        string          `json:"optional_context"`
	OptionalLog            string          `json:"optional_log"`
	ActionField            string          `json:"action_field"`
	ExpectedAction         string          `json:"expected_action"`
	ExpectedFailedSection  string          `json:"expected_failed_section"`
	ExpectedExitCode       int             `json:"expected_exit_code"`
	ExpectedSectionSignals []sectionSignal `json:"expected_section_signals"`
	RequiredResponseFields []string        `json:"required_response_fields"`
	FallbackOrder          []string        `json:"fallback_order"`
}

type sectionSignal struct {
	Section string `json:"section"`
	Action  string `json:"action"`
}

type summary struct {
	Sections []summarySection `json:"sections"`
}

type summarySection struct {
	Name    string            `json:"name"`
	Signals map[string]string `json:"signals"`
}

type summaryContract struct {
	path     string
	resolved *jsonschema.Resolved
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "agent response manifest demo failed: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: go run ./examples/agent-response-manifest-demo <agent-response-artifact-manifest.json>")
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("%s invalid JSON: %w", args[0], err)
	}
	if m.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema_version: %d", m.SchemaVersion)
	}

	summarySchema, err := loadSummarySchema(args[0], m.SummarySchema)
	if err != nil {
		return err
	}

	fmt.Printf("manifest_schema_version=%d\n", m.SchemaVersion)
	fmt.Printf("summary_schema=%s\n", m.SummarySchema)
	fmt.Printf("artifact_count=%d\n", len(m.Artifacts))
	for i, artifact := range m.Artifacts {
		if err := validateArtifact(artifact, summarySchema); err != nil {
			return fmt.Errorf("%s: %w", artifact.Kind, err)
		}
		decisionAction, err := loadDecisionAction(artifact)
		if err != nil {
			return fmt.Errorf("%s: %w", artifact.Kind, err)
		}
		fmt.Printf("%d. kind=%s action_field=%s expected_action=%s failed_section=%s exit_code=%d\n",
			i+1,
			artifact.Kind,
			artifact.ActionField,
			artifact.ExpectedAction,
			artifact.ExpectedFailedSection,
			artifact.ExpectedExitCode,
		)
		fmt.Printf("   directory=%s\n", artifact.Directory)
		fmt.Printf("   decision_action=%s\n", decisionAction)
		fmt.Printf("   summary_validated=%s\n", artifact.Summary)
		fmt.Printf("   expected_section_signals=%s\n", formatSectionSignals(artifact.ExpectedSectionSignals))
		fmt.Printf("   required_response_fields=%s\n", strings.Join(artifact.RequiredResponseFields, ","))
		fmt.Printf("   fallback_order=%s\n", strings.Join(artifact.FallbackOrder, " > "))
	}
	return nil
}

func loadSummarySchema(manifestPath, rel string) (summaryContract, error) {
	if rel == "" {
		return summaryContract{}, fmt.Errorf("missing summary_schema")
	}
	if filepath.IsAbs(rel) {
		return summaryContract{}, fmt.Errorf("summary_schema must be relative")
	}
	path := filepath.Join(filepath.Dir(manifestPath), rel)
	data, err := os.ReadFile(path)
	if err != nil {
		return summaryContract{}, fmt.Errorf("summary_schema is not readable: %w", err)
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return summaryContract{}, fmt.Errorf("summary_schema invalid JSON: %w", err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return summaryContract{}, fmt.Errorf("summary_schema cannot be resolved: %w", err)
	}
	return summaryContract{path: path, resolved: resolved}, nil
}

func validateArtifact(artifact artifact, summarySchema summaryContract) error {
	if artifact.Kind == "" {
		return fmt.Errorf("missing kind")
	}
	for key, rel := range map[string]string{
		"agent_response": artifact.AgentResponse,
		"decision":       artifact.Decision,
		"summary":        artifact.Summary,
		"report":         artifact.Report,
	} {
		if rel == "" {
			return fmt.Errorf("missing %s", key)
		}
		if _, err := os.Stat(filepath.Join(artifact.Directory, rel)); err != nil {
			return fmt.Errorf("%s file is not readable: %w", key, err)
		}
	}
	for key, rel := range map[string]string{
		"optional_context": artifact.OptionalContext,
		"optional_log":     artifact.OptionalLog,
	} {
		if rel == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(artifact.Directory, rel)); err != nil {
			return fmt.Errorf("%s file is listed but not readable: %w", key, err)
		}
	}
	if artifact.ActionField == "" || artifact.ExpectedAction == "" {
		return fmt.Errorf("missing action field or expected action")
	}
	if len(artifact.ExpectedSectionSignals) == 0 {
		return fmt.Errorf("missing expected section signals")
	}
	if len(artifact.RequiredResponseFields) == 0 {
		return fmt.Errorf("missing required response fields")
	}
	if len(artifact.FallbackOrder) == 0 || artifact.FallbackOrder[0] != "agent-response.txt" {
		return fmt.Errorf("fallback order must start with agent-response.txt")
	}

	response, err := os.ReadFile(filepath.Join(artifact.Directory, artifact.AgentResponse))
	if err != nil {
		return err
	}
	responseText := string(response)
	for _, field := range artifact.RequiredResponseFields {
		if !strings.Contains(responseText, "- "+field+"=") {
			return fmt.Errorf("agent response missing field %s", field)
		}
	}
	if !strings.Contains(responseText, artifact.ActionField+"="+artifact.ExpectedAction) {
		return fmt.Errorf("agent response missing expected action")
	}
	decisionAction, err := loadDecisionAction(artifact)
	if err != nil {
		return err
	}
	if decisionAction != artifact.ExpectedAction {
		return fmt.Errorf("decision action = %q, want %q", decisionAction, artifact.ExpectedAction)
	}
	if err := validateExpectedSectionSignals(artifact, summarySchema); err != nil {
		return err
	}
	return nil
}

func loadDecisionAction(artifact artifact) (string, error) {
	data, err := os.ReadFile(filepath.Join(artifact.Directory, artifact.Decision))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || key != "agent_next_step" {
			continue
		}
		if value == "" {
			return "", fmt.Errorf("decision agent_next_step is empty")
		}
		return value, nil
	}
	return "", fmt.Errorf("decision missing agent_next_step")
}

func validateExpectedSectionSignals(artifact artifact, summarySchema summaryContract) error {
	data, err := os.ReadFile(filepath.Join(artifact.Directory, artifact.Summary))
	if err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("summary invalid JSON: %w", err)
	}
	if err := summarySchema.resolved.Validate(raw); err != nil {
		return fmt.Errorf("summary does not validate against %s: %w", summarySchema.path, err)
	}
	var parsed summary
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("summary invalid JSON: %w", err)
	}
	for _, expected := range artifact.ExpectedSectionSignals {
		if expected.Section == "" || expected.Action == "" {
			return fmt.Errorf("expected section signal is incomplete")
		}
		found := false
		for _, section := range parsed.Sections {
			if section.Name != expected.Section {
				continue
			}
			if section.Signals["action"] != expected.Action {
				return fmt.Errorf("summary section %s action signal = %q, want %q", expected.Section, section.Signals["action"], expected.Action)
			}
			found = true
			break
		}
		if !found {
			return fmt.Errorf("summary missing section signal %s action=%s", expected.Section, expected.Action)
		}
	}
	return nil
}

func formatSectionSignals(signals []sectionSignal) string {
	parts := make([]string, 0, len(signals))
	for _, signal := range signals {
		parts = append(parts, signal.Section+":"+signal.Action)
	}
	return strings.Join(parts, ",")
}
