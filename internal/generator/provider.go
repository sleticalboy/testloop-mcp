package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sleticalboy/testloop-mcp/internal/detector"
	"github.com/sleticalboy/testloop-mcp/types"
)

const EnvLLMProviderCommand = "TESTLOOP_LLM_PROVIDER_CMD"

type ProviderErrorKind string

const (
	ProviderErrorConfigMissing          ProviderErrorKind = "llm_config_missing"
	ProviderErrorCommandFailed          ProviderErrorKind = "llm_command_failed"
	ProviderErrorEmptyOutput            ProviderErrorKind = "llm_empty_output"
	ProviderErrorJSON                   ProviderErrorKind = "llm_json_error"
	ProviderErrorMissingCode            ProviderErrorKind = "llm_missing_code"
	ProviderErrorOutputCleaningFailed   ProviderErrorKind = "llm_output_cleaning_failed"
	ProviderErrorOutputValidationFailed ProviderErrorKind = "llm_output_validation_failed"
)

type ProviderError struct {
	Kind     ProviderErrorKind
	Provider string
	Message  string
	Err      error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" && e.Err != nil {
		msg = e.Err.Error()
	}
	if msg == "" {
		msg = string(e.Kind)
	}
	if e.Provider == "" {
		return msg
	}
	return e.Provider + ": " + msg
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func ProviderErrorInfo(err error) (*ProviderError, bool) {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr, true
	}
	return nil, false
}

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
	Framework    string
	TestFile     string
}

type StaticProvider struct{}

func (StaticProvider) Name() string {
	return "static"
}

func (StaticProvider) GenerateTests(_ context.Context, req TestGenerationRequest) (string, error) {
	if strings.TrimSpace(req.StaticCode) != "" {
		return req.StaticCode, nil
	}
	if req.Context != nil && req.Context.CoverageTask != nil {
		return generateTestsForCoverageTask(req.SourceFile, req.Context.CoverageTask)
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
		return "", llmProviderError(ProviderErrorConfigMissing, "%s is empty", EnvLLMProviderCommand)
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
		return "", llmProviderError(ProviderErrorCommandFailed, "llm provider failed: %s", msg)
	}

	code, err := parseLLMProviderOutput(out)
	if err != nil {
		return "", err
	}
	if err := validateLLMProviderTestCode(req.SourceFile, code); err != nil {
		return "", err
	}
	return code, nil
}

func llmProviderError(kind ProviderErrorKind, format string, args ...any) error {
	return &ProviderError{
		Kind:     kind,
		Provider: "llm-command",
		Message:  fmt.Sprintf(format, args...),
	}
}

type llmProviderResponse struct {
	Code string `json:"code"`
}

func parseLLMProviderOutput(out []byte) (string, error) {
	text := strings.TrimSpace(string(out))
	if text == "" {
		return "", llmProviderError(ProviderErrorEmptyOutput, "llm provider returned empty output")
	}

	if strings.HasPrefix(text, "{") {
		var resp llmProviderResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return "", &ProviderError{
				Kind:     ProviderErrorJSON,
				Provider: "llm-command",
				Message:  "parse llm provider json output: " + err.Error(),
				Err:      err,
			}
		}
		if strings.TrimSpace(resp.Code) == "" {
			return "", llmProviderError(ProviderErrorMissingCode, "llm provider json output missing code")
		}
		return cleanLLMProviderCode(resp.Code)
	}

	return cleanLLMProviderCode(text)
}

func cleanLLMProviderCode(text string) (string, error) {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if text == "" {
		return "", llmProviderError(ProviderErrorEmptyOutput, "llm provider returned empty output")
	}
	if fenced, ok := extractFirstCodeFence(text); ok {
		return fenced, nil
	}

	lines := strings.Split(text, "\n")
	start := -1
	for i, line := range lines {
		if llmProviderLineLooksLikeCode(line) {
			start = i
			break
		}
	}
	if start == -1 {
		return "", llmProviderError(ProviderErrorOutputCleaningFailed, "llm provider output did not contain test code")
	}
	end := start
	for i := len(lines) - 1; i >= start; i-- {
		if llmProviderLineLooksLikeCode(lines[i]) {
			end = i
			break
		}
	}
	code := strings.TrimSpace(strings.Join(lines[start:end+1], "\n"))
	if code == "" {
		return "", llmProviderError(ProviderErrorOutputCleaningFailed, "llm provider output did not contain test code")
	}
	return code, nil
}

func extractFirstCodeFence(text string) (string, bool) {
	start := strings.Index(text, "```")
	if start < 0 {
		return "", false
	}
	afterStart := text[start+3:]
	if newline := strings.IndexByte(afterStart, '\n'); newline >= 0 {
		afterStart = afterStart[newline+1:]
	} else {
		return "", false
	}
	end := strings.Index(afterStart, "```")
	if end < 0 {
		return "", false
	}
	code := strings.TrimSpace(afterStart[:end])
	return code, code != ""
}

func llmProviderLineLooksLikeCode(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	lower := strings.ToLower(line)
	codePrefixes := []string{
		"package ", "import ", "from ", "func ", "def ", "class ", "public ", "private ", "protected ",
		"const ", "let ", "var ", "export ", "module.exports", "require(", "describe(", "it(", "test(",
		"assert ", "expect(", "return ", "use ", "#[", "@", "//", "/*", "*", "}", ")", "]",
	}
	for _, prefix := range codePrefixes {
		if strings.HasPrefix(line, prefix) || strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	codeMarkers := []string{"=>", ":=", "==", "!=", "<=", ">=", "{", "}", ";", "()", "&&", "||"}
	for _, marker := range codeMarkers {
		if strings.Contains(line, marker) {
			return true
		}
	}
	return false
}

var (
	llmProviderGoTestRe         = regexp.MustCompile(`(?m)^\s*func\s+Test[A-Za-z0-9_]*\s*\(`)
	llmProviderJavaScriptTestRe = regexp.MustCompile(`(?m)(?:^|[^\w$])(?:describe|it|test)(?:\s*\(|\.(?:each|only|skip|todo|concurrent)\s*\()|(?:^|[^\w$])expect\s*\(`)
	llmProviderPythonTestRe     = regexp.MustCompile(`(?m)^\s*(?:async\s+)?def\s+test_[A-Za-z0-9_]*\s*\(`)
	llmProviderRustTestRe       = regexp.MustCompile(`(?m)#\[(?:[A-Za-z0-9_]+::)?test\]|\bfn\s+test_[A-Za-z0-9_]*\s*\(`)
	llmProviderJavaTestRe       = regexp.MustCompile(`(?m)^\s*@(?:org\.junit\.jupiter\.api\.)?Test\b|\borg\.junit(?:\.jupiter)?\.api\.Test\b`)
)

func validateLLMProviderTestCode(srcPath, code string) error {
	language := languageNameForPath(srcPath)
	if language == "" {
		return nil
	}
	if llmProviderCodeLooksLikeTest(language, code) {
		return nil
	}
	return llmProviderError(ProviderErrorOutputValidationFailed, "llm provider output did not look like %s test code", language)
}

func llmProviderCodeLooksLikeTest(language, code string) bool {
	switch language {
	case "go":
		return llmProviderGoTestRe.MatchString(code)
	case "javascript", "typescript":
		return llmProviderJavaScriptTestRe.MatchString(code)
	case "python":
		return llmProviderPythonTestRe.MatchString(code)
	case "rust":
		return llmProviderRustTestRe.MatchString(code)
	case "java":
		return llmProviderJavaTestRe.MatchString(code)
	default:
		return true
	}
}

func NewTestProvider(mode string) (TestProvider, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "static":
		return StaticProvider{}, nil
	case "llm":
		command := strings.TrimSpace(os.Getenv(EnvLLMProviderCommand))
		if command == "" {
			return nil, &ProviderError{
				Kind:     ProviderErrorConfigMissing,
				Provider: "llm-command",
				Message:  fmt.Sprintf("provider llm requires %s", EnvLLMProviderCommand),
			}
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
	if opts.CoverageTask != nil {
		return generateTestsForCoverageTask(srcPath, opts.CoverageTask)
	}
	if framework := effectiveGenerationFramework(srcPath, opts); isJavaScriptPath(srcPath) && framework != "" {
		return GenerateJavaScriptTestsWithFrameworkAndTestFile(srcPath, framework, opts.TestFile)
	}
	return GenerateTestsStatic(srcPath)
}

func effectiveGenerationFramework(srcPath string, opts GenerateTestsOptions) string {
	if opts.CoverageTask != nil && strings.TrimSpace(opts.CoverageTask.Framework) != "" {
		return opts.CoverageTask.Framework
	}
	if !isJavaScriptPath(srcPath) {
		return strings.TrimSpace(opts.Framework)
	}
	if strings.TrimSpace(opts.Framework) != "" {
		return normalizeJavaScriptTestFramework(opts.Framework)
	}
	return normalizeJavaScriptTestFramework(detector.DetectFramework(srcPath))
}

func isJavaScriptPath(srcPath string) bool {
	switch strings.ToLower(filepath.Ext(srcPath)) {
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		return true
	default:
		return false
	}
}

func generateTestsForCoverageTask(srcPath string, task *types.CoverageTestTask) (string, error) {
	switch strings.ToLower(filepath.Ext(srcPath)) {
	case ".go":
		return GenerateGoTestsForCoverageTask(srcPath, task)
	case ".py":
		return GeneratePytestTestsForCoverageTask(srcPath, task)
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		return GenerateJavaScriptTestsForCoverageTask(srcPath, task)
	case ".rs":
		source, err := os.ReadFile(srcPath)
		if err != nil {
			return "", fmt.Errorf("读取 Rust 源文件失败: %w", err)
		}
		_, content, err := GenerateRustTestsForCoverageTask(source, srcPath, task)
		return content, err
	case ".java":
		source, err := os.ReadFile(srcPath)
		if err != nil {
			return "", fmt.Errorf("读取 Java 源文件失败: %w", err)
		}
		_, content, err := GenerateJavaTestsForCoverageTask(source, srcPath, task)
		return content, err
	default:
		return GenerateTestsStatic(srcPath)
	}
}
