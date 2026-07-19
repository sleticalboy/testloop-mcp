package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	Reason   string            `json:"reason"`
	Signals  map[string]string `json:"signals"`
}

type sectionDecision struct {
	Action string
	Next   string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "verification summary decision demo failed: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: go run ./examples/verification-summary-decision-demo <summary-json>")
	}

	summary, err := loadSummary(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("verification_summary: status=%s failed=%d sections=%d\n", summary.OverallStatus, summary.FailedCount, len(summary.Sections))
	printSectionSignals(summary.Sections)
	if summary.OverallStatus != "failed" && summary.FailedCount == 0 {
		fmt.Println("agent_next_step=ready")
		return nil
	}

	failed := failedSections(summary.Sections)
	if len(failed) == 0 {
		fmt.Println("agent_next_step=inspect-verification-summary")
		return nil
	}

	for i, section := range failed {
		decision := decideSection(section)
		fmt.Printf("%d. failed_section=%s exit_code=%s decision=%s next=%s\n", i+1, section.Name, formatExitCode(section.ExitCode), decision.Action, decision.Next)
	}
	fmt.Printf("agent_next_step=%s\n", decideSection(failed[0]).Action)
	if summary.MarkdownReport != "" {
		fmt.Printf("markdown_report=%s\n", summary.MarkdownReport)
	}
	return nil
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

func loadSummary(path string) (verificationSummary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return verificationSummary{}, err
	}
	var summary verificationSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return verificationSummary{}, fmt.Errorf("%s invalid JSON: %w", path, err)
	}
	if summary.OverallStatus == "" {
		return verificationSummary{}, fmt.Errorf("%s missing overall_status", path)
	}
	return summary, nil
}

func failedSections(sections []verificationSection) []verificationSection {
	failed := make([]verificationSection, 0)
	for _, section := range sections {
		if section.Status == "failed" {
			failed = append(failed, section)
		}
	}
	return failed
}

func decideSection(section verificationSection) sectionDecision {
	name := section.Name
	switch {
	case strings.Contains(name, "基础安装"):
		return sectionDecision{
			Action: "fix-installation",
			Next:   "check binary path, version, client config roundtrip, and HTTP health",
		}
	case strings.Contains(name, "MCP 协议"):
		return sectionDecision{
			Action: "inspect-mcp-transport",
			Next:   "debug stdio or Streamable HTTP MCP client startup",
		}
	case strings.Contains(name, "Agent 闭环"):
		return sectionDecision{
			Action: "inspect-agent-demo",
			Next:   "check structuredContent feedback loop and demo project runner",
		}
	case strings.Contains(name, "showcase"):
		return sectionDecision{
			Action: "inspect-showcase",
			Next:   "check external network, cloned project state, and action expectations",
		}
	case strings.Contains(name, "用户项目"):
		return sectionDecision{
			Action: "inspect-user-project",
			Next:   "check user project command, dependencies, environment variables, and test output",
		}
	default:
		return sectionDecision{
			Action: "inspect-verification",
			Next:   "open markdown report and inspect the failed section output",
		}
	}
}

func formatExitCode(code *int) string {
	if code == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *code)
}
