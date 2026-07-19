package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type manifest struct {
	SchemaVersion int        `json:"schema_version"`
	Artifacts     []artifact `json:"artifacts"`
}

type artifact struct {
	Kind                   string   `json:"kind"`
	Directory              string   `json:"directory"`
	AgentResponse          string   `json:"agent_response"`
	Decision               string   `json:"decision"`
	Summary                string   `json:"summary"`
	Report                 string   `json:"report"`
	OptionalContext        string   `json:"optional_context"`
	OptionalLog            string   `json:"optional_log"`
	ActionField            string   `json:"action_field"`
	ExpectedAction         string   `json:"expected_action"`
	ExpectedFailedSection  string   `json:"expected_failed_section"`
	ExpectedExitCode       int      `json:"expected_exit_code"`
	RequiredResponseFields []string `json:"required_response_fields"`
	FallbackOrder          []string `json:"fallback_order"`
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

	fmt.Printf("manifest_schema_version=%d\n", m.SchemaVersion)
	fmt.Printf("artifact_count=%d\n", len(m.Artifacts))
	for i, artifact := range m.Artifacts {
		if err := validateArtifact(artifact); err != nil {
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
		fmt.Printf("   required_response_fields=%s\n", strings.Join(artifact.RequiredResponseFields, ","))
		fmt.Printf("   fallback_order=%s\n", strings.Join(artifact.FallbackOrder, " > "))
	}
	return nil
}

func validateArtifact(artifact artifact) error {
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
	return nil
}
