package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
		decisions = append(decisions, decision)
		fmt.Printf("%d. fixture=%s status=%s action=%s decision=%s\n", i+1, sample.Name, sample.Status, sample.Action, decision)
	}
	fmt.Printf("agent_decisions=%s\n", strings.Join(decisions, ","))
	return nil
}

func loadFixtureSamples(dir string) ([]validationSample, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "validate-coverage-task-*.json"))
	if err != nil {
		return nil, err
	}
	realProjectPaths, err := filepath.Glob(filepath.Join(dir, "real-project-agent-loop", "*.json"))
	if err != nil {
		return nil, err
	}
	paths = append(paths, realProjectPaths...)
	sort.Strings(paths)
	samples := make([]validationSample, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var sample validationSample
		if err := json.Unmarshal(data, &sample); err != nil {
			return nil, fmt.Errorf("%s invalid JSON: %w", path, err)
		}
		name, err := filepath.Rel(dir, path)
		if err != nil {
			return nil, err
		}
		sample.Name = filepath.ToSlash(name)
		samples = append(samples, sample)
	}
	sort.SliceStable(samples, func(i, j int) bool {
		leftOrder := sampleOrder(samples[i])
		rightOrder := sampleOrder(samples[j])
		if leftOrder != rightOrder {
			return leftOrder < rightOrder
		}
		leftSource := sampleSourceOrder(samples[i])
		rightSource := sampleSourceOrder(samples[j])
		if leftSource != rightSource {
			return leftSource < rightSource
		}
		return samples[i].Name < samples[j].Name
	})
	return samples, nil
}

func sampleSourceOrder(sample validationSample) int {
	if strings.HasPrefix(sample.Name, "real-project-agent-loop/") {
		return 20
	}
	return 10
}

func sampleOrder(sample validationSample) int {
	switch {
	case sample.Status == "passed" && sample.Action == "ready":
		return 10
	case strings.HasPrefix(sample.Action, "manual_review_"):
		return 20
	case sample.Action == "apply_fix_suggestions":
		return 30
	case sample.Action == "needs_better_input":
		return 40
	default:
		return 100
	}
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
