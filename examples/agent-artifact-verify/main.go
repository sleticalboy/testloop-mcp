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

type artifactVerification struct {
	Kind           artifactKind
	ArtifactDir    string
	Summary        verificationSummary
	DecisionAction string
	ExpectedAction string
	FailedSection  string
	ExitCode       string
}

type artifactManifest struct {
	SchemaVersion int                `json:"schema_version"`
	Artifacts     []manifestArtifact `json:"artifacts"`
}

type manifestArtifact struct {
	Kind                   string                  `json:"kind"`
	Directory              string                  `json:"directory"`
	ActionField            string                  `json:"action_field"`
	ExpectedAction         string                  `json:"expected_action"`
	ExpectedFailedSection  string                  `json:"expected_failed_section"`
	ExpectedExitCode       *int                    `json:"expected_exit_code"`
	ExpectedSectionSignals []manifestSectionSignal `json:"expected_section_signals"`
}

type manifestSectionSignal struct {
	Section string `json:"section"`
	Action  string `json:"action"`
}

type outputFormat string

const (
	outputText outputFormat = "text"
	outputJSON outputFormat = "json"
)

type artifactOutput struct {
	Status         string                  `json:"status"`
	ArtifactKind   string                  `json:"artifact_kind"`
	ArtifactDir    string                  `json:"artifact_dir"`
	SummarySchema  string                  `json:"summary_schema"`
	OverallStatus  string                  `json:"overall_status"`
	FailedCount    int                     `json:"failed_count"`
	DecisionAction string                  `json:"decision_action"`
	ResponseAction string                  `json:"response_action"`
	FailedSection  string                  `json:"failed_section,omitempty"`
	ExitCode       string                  `json:"exit_code,omitempty"`
	SectionSignals []manifestSectionSignal `json:"section_signals,omitempty"`
	RequiredFiles  int                     `json:"required_files"`
}

type manifestOutput struct {
	Status                string           `json:"status"`
	ManifestSchemaVersion int              `json:"manifest_schema_version"`
	ManifestPath          string           `json:"manifest_path"`
	ArtifactCount         int              `json:"artifact_count"`
	Artifacts             []artifactOutput `json:"artifacts"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "agent artifact verification failed: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	format := outputText
	if len(args) > 0 && args[0] == "--json" {
		format = outputJSON
		args = args[1:]
	}
	if len(args) != 2 {
		return fmt.Errorf("usage: go run ./examples/agent-artifact-verify [--json] <first-run|onboarding|manifest> <artifact-dir|manifest-json>")
	}

	if args[0] == "manifest" {
		return runManifest(args[1], format)
	}

	result, err := verifyArtifact(args[0], args[1])
	if err != nil {
		return err
	}
	if format == outputJSON {
		return writeJSON(artifactOutputFrom(result))
	}
	printArtifactVerification(result)
	return nil
}

func verifyArtifact(kindName, artifactDirArg string) (artifactVerification, error) {
	kind, err := parseKind(kindName)
	if err != nil {
		return artifactVerification{}, err
	}
	artifactDir, err := filepath.Abs(artifactDirArg)
	if err != nil {
		return artifactVerification{}, err
	}
	if info, err := os.Stat(artifactDir); err != nil {
		return artifactVerification{}, fmt.Errorf("artifact directory is not readable: %w", err)
	} else if !info.IsDir() {
		return artifactVerification{}, fmt.Errorf("artifact path must be a directory: %s", artifactDir)
	}

	for _, name := range kind.RequiredFiles {
		if err := requireFile(artifactDir, name); err != nil {
			return artifactVerification{}, err
		}
	}

	summary, err := loadAndValidateSummary(
		filepath.Join(artifactDir, "verification-summary.json"),
		filepath.Join(artifactDir, "verification-summary.schema.json"),
	)
	if err != nil {
		return artifactVerification{}, err
	}
	if err := validateSummarySemantics(summary); err != nil {
		return artifactVerification{}, err
	}

	expectedAction := expectedAction(summary)
	decisionAction, err := loadDecisionAction(filepath.Join(artifactDir, "agent-decision.txt"))
	if err != nil {
		return artifactVerification{}, err
	}
	if decisionAction != expectedAction {
		return artifactVerification{}, fmt.Errorf("agent-decision.txt agent_next_step=%q, want %q", decisionAction, expectedAction)
	}

	if kind.Name == "first-run" {
		contextAction, err := loadContextAction(filepath.Join(artifactDir, "first-run-context.txt"))
		if err != nil {
			return artifactVerification{}, err
		}
		if contextAction != expectedAction {
			return artifactVerification{}, fmt.Errorf("first-run-context.txt first_run_agent_next_step=%q, want %q", contextAction, expectedAction)
		}
	}

	responseText, err := os.ReadFile(filepath.Join(artifactDir, "agent-response.txt"))
	if err != nil {
		return artifactVerification{}, err
	}
	if err := validateAgentResponse(string(responseText), kind, summary, expectedAction); err != nil {
		return artifactVerification{}, err
	}

	failedSection, exitCode := firstFailedSection(summary)
	return artifactVerification{
		Kind:           kind,
		ArtifactDir:    artifactDir,
		Summary:        summary,
		DecisionAction: decisionAction,
		ExpectedAction: expectedAction,
		FailedSection:  failedSection,
		ExitCode:       exitCode,
	}, nil
}

func printArtifactVerification(result artifactVerification) {
	fmt.Println("agent_artifact_status=passed")
	fmt.Printf("artifact_kind=%s\n", result.Kind.Name)
	fmt.Printf("artifact_dir=%s\n", result.ArtifactDir)
	fmt.Println("summary_schema=verification-summary.schema.json")
	fmt.Printf("overall_status=%s\n", result.Summary.OverallStatus)
	fmt.Printf("failed_count=%d\n", result.Summary.FailedCount)
	fmt.Printf("decision_action=%s\n", result.DecisionAction)
	fmt.Printf("response_action=%s\n", result.ExpectedAction)
	if result.FailedSection != "" {
		fmt.Printf("failed_section=%s\n", result.FailedSection)
	}
	if result.ExitCode != "" {
		fmt.Printf("exit_code=%s\n", result.ExitCode)
	}
	printSectionSignals(result.Summary.Sections)
	fmt.Printf("required_files=%d\n", len(result.Kind.RequiredFiles))
}

func runManifest(path string, format outputFormat) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var manifest artifactManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("%s invalid JSON: %w", path, err)
	}
	if manifest.SchemaVersion != 1 {
		return fmt.Errorf("unsupported manifest schema_version: %d", manifest.SchemaVersion)
	}
	if len(manifest.Artifacts) == 0 {
		return fmt.Errorf("manifest contains no artifacts")
	}

	results := make([]artifactVerification, 0, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		artifactDir, err := resolveManifestArtifactDir(artifact.Directory)
		if err != nil {
			return fmt.Errorf("%s: %w", artifact.Kind, err)
		}
		result, err := verifyArtifact(artifact.Kind, artifactDir)
		if err != nil {
			return fmt.Errorf("%s: %w", artifact.Kind, err)
		}
		if err := validateManifestExpectation(artifact, result); err != nil {
			return fmt.Errorf("%s: %w", artifact.Kind, err)
		}
		results = append(results, result)
	}

	manifestPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	output := manifestOutput{
		Status:                "passed",
		ManifestSchemaVersion: manifest.SchemaVersion,
		ManifestPath:          manifestPath,
		ArtifactCount:         len(results),
		Artifacts:             make([]artifactOutput, 0, len(results)),
	}
	for _, result := range results {
		output.Artifacts = append(output.Artifacts, artifactOutputFrom(result))
	}
	if format == outputJSON {
		return writeJSON(output)
	}

	fmt.Println("agent_artifact_manifest_status=passed")
	fmt.Printf("manifest_schema_version=%d\n", manifest.SchemaVersion)
	fmt.Printf("manifest_path=%s\n", manifestPath)
	fmt.Printf("artifact_count=%d\n", len(results))
	for i, result := range results {
		fmt.Printf("%d. artifact_kind=%s expected_action=%s decision_action=%s response_action=%s\n",
			i+1,
			result.Kind.Name,
			result.ExpectedAction,
			result.DecisionAction,
			result.ExpectedAction,
		)
		fmt.Printf("   artifact_dir=%s\n", result.ArtifactDir)
		if result.FailedSection != "" {
			fmt.Printf("   failed_section=%s\n", result.FailedSection)
		}
		if result.ExitCode != "" {
			fmt.Printf("   exit_code=%s\n", result.ExitCode)
		}
		fmt.Printf("   required_files=%d\n", len(result.Kind.RequiredFiles))
	}
	return nil
}

func artifactOutputFrom(result artifactVerification) artifactOutput {
	return artifactOutput{
		Status:         "passed",
		ArtifactKind:   result.Kind.Name,
		ArtifactDir:    result.ArtifactDir,
		SummarySchema:  "verification-summary.schema.json",
		OverallStatus:  result.Summary.OverallStatus,
		FailedCount:    result.Summary.FailedCount,
		DecisionAction: result.DecisionAction,
		ResponseAction: result.ExpectedAction,
		FailedSection:  result.FailedSection,
		ExitCode:       result.ExitCode,
		SectionSignals: sectionSignalsFrom(result.Summary.Sections),
		RequiredFiles:  len(result.Kind.RequiredFiles),
	}
}

func sectionSignalsFrom(sections []verificationSection) []manifestSectionSignal {
	signals := make([]manifestSectionSignal, 0)
	for _, section := range sections {
		action := strings.TrimSpace(section.Signals["action"])
		if action == "" {
			continue
		}
		signals = append(signals, manifestSectionSignal{
			Section: section.Name,
			Action:  action,
		})
	}
	return signals
}

func writeJSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func resolveManifestArtifactDir(directory string) (string, error) {
	if directory == "" {
		return "", fmt.Errorf("missing directory")
	}
	if filepath.IsAbs(directory) {
		return directory, nil
	}
	repoRoot := os.Getenv("TESTLOOP_MCP_REPO_DIR")
	if repoRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		repoRoot = cwd
	}
	return filepath.Join(repoRoot, directory), nil
}

func validateManifestExpectation(artifact manifestArtifact, result artifactVerification) error {
	if artifact.ActionField != "" && artifact.ActionField != result.Kind.ActionField {
		return fmt.Errorf("action_field=%q, want %q", artifact.ActionField, result.Kind.ActionField)
	}
	if artifact.ExpectedAction != "" && artifact.ExpectedAction != result.ExpectedAction {
		return fmt.Errorf("expected_action=%q, want %q", artifact.ExpectedAction, result.ExpectedAction)
	}
	if artifact.ExpectedFailedSection != "" && artifact.ExpectedFailedSection != result.FailedSection {
		return fmt.Errorf("expected_failed_section=%q, want %q", artifact.ExpectedFailedSection, result.FailedSection)
	}
	if artifact.ExpectedExitCode != nil {
		expectedExitCode := fmt.Sprintf("%d", *artifact.ExpectedExitCode)
		if expectedExitCode != result.ExitCode {
			return fmt.Errorf("expected_exit_code=%s, want %s", expectedExitCode, result.ExitCode)
		}
	}
	for _, signal := range artifact.ExpectedSectionSignals {
		if !hasSectionSignal(result.Summary.Sections, signal.Section, signal.Action) {
			return fmt.Errorf("missing expected_section_signal %s:%s", signal.Section, signal.Action)
		}
	}
	return nil
}

func hasSectionSignal(sections []verificationSection, name, action string) bool {
	for _, section := range sections {
		if section.Name == name && section.Signals["action"] == action {
			return true
		}
	}
	return false
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
