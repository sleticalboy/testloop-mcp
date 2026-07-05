package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/binlee/testloop-mcp/types"
)

const EnvLLMProviderCommand = "TESTLOOP_LLM_PROVIDER_CMD"

// TestProvider generates test code from a source file and generation context.
type TestProvider interface {
	Name() string
	GenerateTests(context.Context, TestGenerationRequest) (string, error)
}

type TestGenerationRequest struct {
	SourceFile string                       `json:"source_file"`
	Context    *types.TestGenerationContext `json:"context,omitempty"`
	StaticCode string                       `json:"static_code,omitempty"`
}

type GenerateTestsOptions struct {
	CoverageTask *types.CoverageTestTask
}

type StaticProvider struct{}

func (StaticProvider) Name() string {
	return "static"
}

func (StaticProvider) GenerateTests(_ context.Context, req TestGenerationRequest) (string, error) {
	if strings.TrimSpace(req.StaticCode) != "" {
		return req.StaticCode, nil
	}
	if req.Context != nil && req.Context.CoverageTask != nil && strings.EqualFold(filepath.Ext(req.SourceFile), ".go") {
		return GenerateGoTestsForCoverageTask(req.SourceFile, req.Context.CoverageTask)
	}
	return GenerateTestsStatic(req.SourceFile)
}

type ExternalLLMProvider struct {
	Command string
}

func (p ExternalLLMProvider) Name() string {
	return "llm-command"
}

func (p ExternalLLMProvider) GenerateTests(ctx context.Context, req TestGenerationRequest) (string, error) {
	parts := strings.Fields(p.Command)
	if len(parts) == 0 {
		return "", fmt.Errorf("%s is empty", EnvLLMProviderCommand)
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal llm provider request: %w", err)
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("llm provider failed: %s", msg)
	}

	code, err := parseLLMProviderOutput(out)
	if err != nil {
		return "", err
	}
	return code, nil
}

type llmProviderResponse struct {
	Code string `json:"code"`
}

func parseLLMProviderOutput(out []byte) (string, error) {
	text := strings.TrimSpace(string(out))
	if text == "" {
		return "", fmt.Errorf("llm provider returned empty output")
	}

	if strings.HasPrefix(text, "{") {
		var resp llmProviderResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return "", fmt.Errorf("parse llm provider json output: %w", err)
		}
		if strings.TrimSpace(resp.Code) == "" {
			return "", fmt.Errorf("llm provider json output missing code")
		}
		return resp.Code, nil
	}

	return string(out), nil
}

func NewTestProvider(mode string) (TestProvider, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "static":
		return StaticProvider{}, nil
	case "llm":
		command := strings.TrimSpace(os.Getenv(EnvLLMProviderCommand))
		if command == "" {
			return nil, fmt.Errorf("provider llm requires %s", EnvLLMProviderCommand)
		}
		return ExternalLLMProvider{Command: command}, nil
	case "auto":
		command := strings.TrimSpace(os.Getenv(EnvLLMProviderCommand))
		if command == "" {
			return StaticProvider{}, nil
		}
		return ExternalLLMProvider{Command: command}, nil
	default:
		return nil, fmt.Errorf("unsupported test provider %q (supported: static, llm, auto)", mode)
	}
}

func GenerateTestsWithProvider(ctx context.Context, srcPath string, provider TestProvider) (string, error) {
	return GenerateTestsWithProviderOptions(ctx, srcPath, provider, GenerateTestsOptions{})
}

func GenerateTestsWithProviderOptions(ctx context.Context, srcPath string, provider TestProvider, opts GenerateTestsOptions) (string, error) {
	if provider == nil {
		provider = StaticProvider{}
	}

	staticCode, err := generateTestsStaticWithOptions(srcPath, opts)
	if err != nil {
		return "", err
	}

	code, err := provider.GenerateTests(ctx, TestGenerationRequest{
		SourceFile: srcPath,
		Context:    BuildGenerationContextWithOptions(srcPath, opts),
		StaticCode: staticCode,
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(code) == "" {
		return "", fmt.Errorf("test provider %s returned empty output", provider.Name())
	}
	return code, nil
}

func generateTestsStaticWithOptions(srcPath string, opts GenerateTestsOptions) (string, error) {
	if opts.CoverageTask != nil && strings.EqualFold(filepath.Ext(srcPath), ".go") {
		return GenerateGoTestsForCoverageTask(srcPath, opts.CoverageTask)
	}
	return GenerateTestsStatic(srcPath)
}
