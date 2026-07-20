package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

type verificationSummary struct {
	OverallStatus  string                `json:"overall_status"`
	FailedCount    int                   `json:"failed_count"`
	MarkdownReport string                `json:"markdown_report"`
	Sections       []verificationSection `json:"sections"`
}

type verificationSection struct {
	Name     string            `json:"name"`
	Status   string            `json:"status"`
	ExitCode *int              `json:"exit_code"`
	Signals  map[string]string `json:"signals"`
}

type artifactKind struct {
	Name             string
	ActionField      string
	StatusField      string
	FailedCountField string
	RequiredFiles    []string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "agent artifact verification failed: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: go run ./examples/agent-artifact-verify <first-run|onboarding> <artifact-dir>")
	}

	kind, err := parseKind(args[0])
	if err != nil {
		return err
	}
	artifactDir, err := filepath.Abs(args[1])
	if err != nil {
		return err
	}
	if info, err := os.Stat(artifactDir); err != nil {
		return fmt.Errorf("artifact directory is not readable: %w", err)
	} else if !info.IsDir() {
		return fmt.Errorf("artifact path must be a directory: %s", artifactDir)
	}

	for _, name := range kind.RequiredFiles {
		if err := requireFile(artifactDir, name); err != nil {
			return err
		}
	}

	summary, err := loadAndValidateSummary(
		filepath.Join(artifactDir, "verification-summary.json"),
		filepath.Join(artifactDir, "verification-summary.schema.json"),
	)
	if err != nil {
		return err
	}
	if err := validateSummarySemantics(summary); err != nil {
		return err
	}

	expectedAction := expectedAction(summary)
	decisionAction, err := loadDecisionAction(filepath.Join(artifactDir, "agent-decision.txt"))
	if err != nil {
		return err
	}
	if decisionAction != expectedAction {
		return fmt.Errorf("agent-decision.txt agent_next_step=%q, want %q", decisionAction, expectedAction)
	}

	if kind.Name == "first-run" {
		contextAction, err := loadContextAction(filepath.Join(artifactDir, "first-run-context.txt"))
		if err != nil {
			return err
		}
		if contextAction != expectedAction {
			return fmt.Errorf("first-run-context.txt first_run_agent_next_step=%q, want %q", contextAction, expectedAction)
		}
	}

	responseText, err := os.ReadFile(filepath.Join(artifactDir, "agent-response.txt"))
	if err != nil {
		return err
	}
	if err := validateAgentResponse(string(responseText), kind, summary, expectedAction); err != nil {
		return err
	}

	failedSection, exitCode := firstFailedSection(summary)
	fmt.Println("agent_artifact_status=passed")
	fmt.Printf("artifact_kind=%s\n", kind.Name)
	fmt.Printf("artifact_dir=%s\n", artifactDir)
	fmt.Println("summary_schema=verification-summary.schema.json")
	fmt.Printf("overall_status=%s\n", summary.OverallStatus)
	fmt.Printf("failed_count=%d\n", summary.FailedCount)
	fmt.Printf("decision_action=%s\n", decisionAction)
	fmt.Printf("response_action=%s\n", expectedAction)
	if failedSection != "" {
		fmt.Printf("failed_section=%s\n", failedSection)
	}
	if exitCode != "" {
		fmt.Printf("exit_code=%s\n", exitCode)
	}
	printSectionSignals(summary.Sections)
	fmt.Printf("required_files=%d\n", len(kind.RequiredFiles))
	return nil
}

func parseKind(name string) (artifactKind, error) {
	switch name {
	case "first-run":
		return artifactKind{
			Name:             "first-run",
			ActionField:      "first_run_agent_next_step",
			StatusField:      "first_run_status",
			FailedCountField: "first_run_failed_count",
			RequiredFiles: []string{
				"verification-report.md",
				"verification-summary.json",
				"verification-summary.schema.json",
				"agent-decision.txt",
				"first-run-context.txt",
				"agent-response.txt",
				"first-run.log",
			},
		}, nil
	case "onboarding":
		return artifactKind{
			Name:             "onboarding",
			ActionField:      "agent_next_step",
			StatusField:      "overall_status",
			FailedCountField: "failed_count",
			RequiredFiles: []string{
				"verification-report.md",
				"verification-summary.json",
				"verification-summary.schema.json",
				"agent-decision.txt",
				"agent-response.txt",
			},
		}, nil
	default:
		return artifactKind{}, fmt.Errorf("unknown artifact kind %q, want first-run or onboarding", name)
	}
}

func requireFile(dir, name string) error {
	path := filepath.Join(dir, name)
	if info, err := os.Stat(path); err != nil {
		return fmt.Errorf("missing required file %s: %w", name, err)
	} else if info.IsDir() {
		return fmt.Errorf("required file %s is a directory", name)
	}
	return nil
}

func loadAndValidateSummary(summaryPath, schemaPath string) (verificationSummary, error) {
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return verificationSummary{}, fmt.Errorf("read verification-summary.schema.json: %w", err)
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		return verificationSummary{}, fmt.Errorf("verification-summary.schema.json invalid JSON: %w", err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return verificationSummary{}, fmt.Errorf("verification-summary.schema.json cannot be resolved: %w", err)
	}

	summaryData, err := os.ReadFile(summaryPath)
	if err != nil {
		return verificationSummary{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(summaryData, &raw); err != nil {
		return verificationSummary{}, fmt.Errorf("verification-summary.json invalid JSON: %w", err)
	}
	if err := resolved.Validate(raw); err != nil {
		return verificationSummary{}, fmt.Errorf("verification-summary.json does not validate against local schema: %w", err)
	}

	var summary verificationSummary
	if err := json.Unmarshal(summaryData, &summary); err != nil {
		return verificationSummary{}, fmt.Errorf("verification-summary.json invalid JSON: %w", err)
	}
	return summary, nil
}

func validateSummarySemantics(summary verificationSummary) error {
	failed := failedSections(summary)
	if summary.FailedCount != len(failed) {
		return fmt.Errorf("failed_count=%d, want failed sections=%d", summary.FailedCount, len(failed))
	}
	switch summary.OverallStatus {
	case "passed":
		if summary.FailedCount != 0 {
			return fmt.Errorf("overall_status=passed but failed_count=%d", summary.FailedCount)
		}
	case "failed":
		if summary.FailedCount == 0 {
			return fmt.Errorf("overall_status=failed but failed_count=0")
		}
	default:
		return fmt.Errorf("invalid overall_status: %s", summary.OverallStatus)
	}
	return nil
}

func loadDecisionAction(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || key != "agent_next_step" {
			continue
		}
		if value == "" {
			return "", fmt.Errorf("agent-decision.txt agent_next_step is empty")
		}
		return value, nil
	}
	return "", fmt.Errorf("agent-decision.txt missing agent_next_step")
}

func loadContextAction(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || key != "first_run_agent_next_step" {
			continue
		}
		if value == "" {
			return "", fmt.Errorf("first-run-context.txt first_run_agent_next_step is empty")
		}
		return value, nil
	}
	return "", fmt.Errorf("first-run-context.txt missing first_run_agent_next_step")
}

func validateAgentResponse(response string, kind artifactKind, summary verificationSummary, expectedAction string) error {
	if !strings.Contains(response, "结论：") ||
		!strings.Contains(response, "证据：") ||
		!strings.Contains(response, "下一步：") ||
		!strings.Contains(response, "暂不做：") {
		return fmt.Errorf("agent-response.txt must contain the four stable sections")
	}
	if !strings.Contains(response, "- "+kind.ActionField+"="+expectedAction) {
		return fmt.Errorf("agent-response.txt missing %s=%s", kind.ActionField, expectedAction)
	}
	if !strings.Contains(response, "- "+kind.StatusField+"="+summary.OverallStatus) {
		return fmt.Errorf("agent-response.txt missing %s=%s", kind.StatusField, summary.OverallStatus)
	}
	expectedFailedCount := fmt.Sprintf("- %s=%d", kind.FailedCountField, summary.FailedCount)
	if !strings.Contains(response, expectedFailedCount) {
		return fmt.Errorf("agent-response.txt missing %s=%d", kind.FailedCountField, summary.FailedCount)
	}
	failedSection, exitCode := firstFailedSection(summary)
	if failedSection != "" && !strings.Contains(response, "- failed_section="+failedSection) {
		return fmt.Errorf("agent-response.txt missing failed_section=%s", failedSection)
	}
	if exitCode != "" && !strings.Contains(response, "- exit_code="+exitCode) {
		return fmt.Errorf("agent-response.txt missing exit_code=%s", exitCode)
	}
	for _, section := range summary.Sections {
		action := strings.TrimSpace(section.Signals["action"])
		if action == "" {
			continue
		}
		expected := fmt.Sprintf("- section_signal=%s action=%s", section.Name, action)
		if !strings.Contains(response, expected) {
			return fmt.Errorf("agent-response.txt missing %s", expected)
		}
	}
	return nil
}

func expectedAction(summary verificationSummary) string {
	if summary.OverallStatus != "failed" && summary.FailedCount == 0 {
		return "ready"
	}
	failed := failedSections(summary)
	if len(failed) == 0 {
		return "inspect-verification-summary"
	}
	return decideSection(failed[0].Name)
}

func failedSections(summary verificationSummary) []verificationSection {
	failed := make([]verificationSection, 0)
	for _, section := range summary.Sections {
		if section.Status == "failed" {
			failed = append(failed, section)
		}
	}
	return failed
}

func decideSection(name string) string {
	switch {
	case strings.Contains(name, "基础安装"):
		return "fix-installation"
	case strings.Contains(name, "MCP 协议"):
		return "inspect-mcp-transport"
	case strings.Contains(name, "Agent 闭环"):
		return "inspect-agent-demo"
	case strings.Contains(name, "showcase"):
		return "inspect-showcase"
	case strings.Contains(name, "用户项目"):
		return "inspect-user-project"
	default:
		return "inspect-verification"
	}
}

func firstFailedSection(summary verificationSummary) (string, string) {
	failed := failedSections(summary)
	if len(failed) == 0 {
		return "", ""
	}
	if failed[0].ExitCode == nil {
		return failed[0].Name, ""
	}
	return failed[0].Name, fmt.Sprintf("%d", *failed[0].ExitCode)
}

func printSectionSignals(sections []verificationSection) {
	for _, section := range sections {
		action := strings.TrimSpace(section.Signals["action"])
		if action == "" {
			continue
		}
		fmt.Printf("section_signal=%s action=%s\n", section.Name, action)
	}
}
