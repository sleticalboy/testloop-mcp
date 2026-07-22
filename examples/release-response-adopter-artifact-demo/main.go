package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type verification struct {
	SchemaVersion int      `json:"schema_version"`
	Status        string   `json:"status"`
	ReleaseRef    string   `json:"release_ref"`
	FixtureCount  int      `json:"fixture_count"`
	AgentNextStep string   `json:"agent_next_step"`
	ShouldAccept  bool     `json:"should_accept"`
	RequiredFiles int      `json:"required_files"`
	Files         []file   `json:"files"`
	Failures      []string `json:"failures"`
}

type file struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./examples/release-response-adopter-artifact-demo <artifact-verification.json>")
		os.Exit(1)
	}

	payload, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "read artifact verification: %v\n", err)
		os.Exit(1)
	}

	var result verification
	if err := json.Unmarshal(payload, &result); err != nil {
		fmt.Fprintf(os.Stderr, "parse artifact verification: %v\n", err)
		os.Exit(1)
	}

	missing := missingFiles(result.Files)
	fmt.Printf("artifact_verification_status=%s\n", result.Status)
	fmt.Printf("schema_version=%d\n", result.SchemaVersion)
	fmt.Printf("release_ref=%s\n", result.ReleaseRef)
	fmt.Printf("fixture_count=%d\n", result.FixtureCount)
	fmt.Printf("agent_next_step=%s\n", result.AgentNextStep)
	fmt.Printf("should_accept=%t\n", result.ShouldAccept)
	fmt.Printf("required_files=%d\n", result.RequiredFiles)
	fmt.Printf("missing_files=%s\n", strings.Join(missing, ","))
	fmt.Printf("client_decision=%s\n", clientDecision(result, missing))
	if len(result.Failures) > 0 {
		fmt.Printf("failures=%s\n", strings.Join(result.Failures, "; "))
	}
}

func missingFiles(files []file) []string {
	var missing []string
	for _, item := range files {
		if !item.Exists {
			missing = append(missing, item.Path)
		}
	}
	return missing
}

func clientDecision(result verification, missing []string) string {
	if result.Status == "passed" && result.AgentNextStep == "ready" && result.ShouldAccept && len(missing) == 0 {
		return "accept"
	}
	if result.AgentNextStep == "inspect-release-response-adopter-artifact" || len(missing) > 0 {
		return "inspect-artifact"
	}
	return "manual-review"
}
