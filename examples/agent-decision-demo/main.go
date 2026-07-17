package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type validationSample struct {
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
	data, err := os.ReadFile("docs/validate-coverage-task-samples.md")
	if err != nil {
		return err
	}
	blocks := jsonCodeBlocks(string(data))
	if len(blocks) == 0 {
		return fmt.Errorf("no json samples found")
	}

	decisions := make([]string, 0, len(blocks))
	for i, block := range blocks {
		var sample validationSample
		if err := json.Unmarshal([]byte(block), &sample); err != nil {
			return fmt.Errorf("sample %d invalid JSON: %w", i+1, err)
		}
		decision := agentDecision(sample)
		decisions = append(decisions, decision)
		fmt.Printf("%d. status=%s action=%s decision=%s\n", i+1, sample.Status, sample.Action, decision)
	}
	fmt.Printf("agent_decisions=%s\n", strings.Join(decisions, ","))
	return nil
}

func jsonCodeBlocks(text string) []string {
	re := regexp.MustCompile("(?s)```json\\n(.*?)\\n```")
	matches := re.FindAllStringSubmatch(text, -1)
	blocks := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) == 2 {
			blocks = append(blocks, match[1])
		}
	}
	return blocks
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
