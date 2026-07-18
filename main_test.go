package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParseServerConfigDefaults(t *testing.T) {
	var stderr bytes.Buffer

	cfg, code := parseServerConfig(nil, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if cfg.transport != "stdio" || cfg.addr != ":8080" || cfg.stateless {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.configHTTPURL != "http://localhost:8080/mcp" {
		t.Fatalf("configHTTPURL = %q", cfg.configHTTPURL)
	}
}

func TestParseServerConfigHTTP(t *testing.T) {
	var stderr bytes.Buffer

	cfg, code := parseServerConfig([]string{"--transport=http", "--addr=:18080", "--stateless"}, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if cfg.transport != "http" || cfg.addr != ":18080" || !cfg.stateless {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestParseServerConfigInvalidFlag(t *testing.T) {
	var stderr bytes.Buffer

	_, code := parseServerConfig([]string{"--bad"}, &stderr)

	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("stderr missing flag error: %q", stderr.String())
	}
}

func TestParseServerConfigRejectsUnsupportedTransport(t *testing.T) {
	var stderr bytes.Buffer

	cfg, code := parseServerConfig([]string{"--transport=grpc"}, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if cfg.transport != "grpc" {
		t.Fatalf("transport = %q, want grpc", cfg.transport)
	}
	if !strings.Contains(stderr.String(), "不支持的传输模式") {
		t.Fatalf("stderr missing transport error: %q", stderr.String())
	}
}

func TestParseServerConfigRejectsMultipleConfigActions(t *testing.T) {
	var stderr bytes.Buffer

	_, code := parseServerConfig([]string{"--print-config=codex", "--version"}, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "不能同时使用") {
		t.Fatalf("stderr missing mutual exclusion error: %q", stderr.String())
	}
}

func TestVersionFlag(t *testing.T) {
	var stderr bytes.Buffer

	cfg, code := parseServerConfig([]string{"--version"}, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if !cfg.version {
		t.Fatalf("version = false, want true")
	}
	if appVersion != "0.5.6" {
		t.Fatalf("appVersion = %q, want 0.5.6", appVersion)
	}
}

func TestPrintClientConfigCodex(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := serverConfig{
		printConfig:   "codex",
		configCommand: "/opt/testloop-mcp",
		configHTTPURL: "http://localhost:8080/mcp",
	}

	code := printClientConfig(cfg, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "[mcp_servers.testloop]") || !strings.Contains(got, `command = "/opt/testloop-mcp"`) {
		t.Fatalf("unexpected codex config:\n%s", got)
	}
}

func TestPrintClientConfigAll(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := serverConfig{
		printConfig:   "all",
		configCommand: `/opt/Test Loop/testloop-mcp`,
		configHTTPURL: "http://127.0.0.1:18080/mcp",
	}

	code := printClientConfig(cfg, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, stderr=%q", code, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{
		"# ~/.codex/config.toml",
		`command = "/opt/Test Loop/testloop-mcp"`,
		`url = "http://127.0.0.1:18080/mcp"`,
		"# ~/.claude/claude_desktop_config.json",
		"# .cursor/mcp.json",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in config:\n%s", want, got)
		}
	}
}

func TestPrintClientConfigRejectsUnknownClient(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := printClientConfig(serverConfig{printConfig: "vscode"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "不支持的客户端配置类型") {
		t.Fatalf("stderr missing client error: %q", stderr.String())
	}
}

func TestCheckClientConfigJSONCommand(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "testloop-mcp")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	config := `{"mcpServers":{"testloop":{"command":"` + binary + `"}}}`
	configPath := filepath.Join(dir, "claude.json")
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := checkClientConfig(serverConfig{checkConfig: configPath}, strings.NewReader(""), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok: testloop command "+binary) {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestCheckClientConfigTOMLURLFromStdin(t *testing.T) {
	config := `[mcp_servers.testloop]
url = "http://localhost:8080/mcp"
`
	var stdout, stderr bytes.Buffer

	code := checkClientConfig(serverConfig{checkConfig: "-"}, strings.NewReader(config), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok: testloop url http://localhost:8080/mcp") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestCheckClientConfigRejectsMissingCommand(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "cursor.json")
	if err := os.WriteFile(configPath, []byte(`{"mcpServers":{"testloop":{"command":"`+filepath.Join(dir, "missing")+`"}}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := checkClientConfig(serverConfig{checkConfig: configPath}, strings.NewReader(""), &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "command 无效") {
		t.Fatalf("stderr missing command error: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "suggestion: update testloop with `testloop-mcp --print-config=cursor --config-command") {
		t.Fatalf("stderr missing command suggestion: %q", stderr.String())
	}
}

func TestCheckClientConfigRejectsInvalidURL(t *testing.T) {
	config := `[mcp_servers.testloop]
url = "file:///tmp/testloop.sock"
`
	var stdout, stderr bytes.Buffer

	code := checkClientConfig(serverConfig{checkConfig: "-"}, strings.NewReader(config), &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "url 无效") {
		t.Fatalf("stderr missing url error: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "suggestion: use an http(s) Streamable HTTP endpoint") {
		t.Fatalf("stderr missing url suggestion: %q", stderr.String())
	}
}

func TestCheckClientConfigRejectsEmptyConfig(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := checkClientConfig(serverConfig{checkConfig: "-"}, strings.NewReader("{}"), &stdout, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "未找到 MCP server 配置") {
		t.Fatalf("stderr missing empty config error: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "suggestion: run `testloop-mcp --doctor-config`") {
		t.Fatalf("stderr missing empty config suggestion: %q", stderr.String())
	}
}

func TestCheckClientConfigParsesMixedPrintConfigOutput(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "testloop-mcp")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	config := `[mcp_servers.testloop]
command = "` + binary + `"

---

# ~/.claude/claude_desktop_config.json
{
  "mcpServers": {
    "testloop": {
      "command": "` + binary + `"
    }
  }
}

---

# .cursor/mcp.json
{
  "mcpServers": {
    "testloop": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
`
	var stdout, stderr bytes.Buffer

	code := checkClientConfig(serverConfig{checkConfig: "-"}, strings.NewReader(config), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, stderr=%q", code, stderr.String())
	}
	got := stdout.String()
	if strings.Count(got, "ok: testloop") != 3 {
		t.Fatalf("expected three validated entries, got:\n%s", got)
	}
}

func TestCLIPrintConfigOutputCanBeCheckedByBuiltBinary(t *testing.T) {
	binary := buildMainBinary(t)
	httpURL := "http://127.0.0.1:18080/mcp"

	printCmd := exec.Command(binary, "--print-config=all", "--config-command="+binary, "--config-http-url="+httpURL)
	configOutput, err := printCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("print config failed: %v\n%s", err, configOutput)
	}
	config := string(configOutput)
	for _, want := range []string{
		"# ~/.codex/config.toml",
		`command = "` + binary + `"`,
		`url = "` + httpURL + `"`,
		"# ~/.claude/claude_desktop_config.json",
		"# .cursor/mcp.json",
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("generated config missing %q:\n%s", want, config)
		}
	}

	checkCmd := exec.Command(binary, "--check-config", "-")
	checkCmd.Stdin = strings.NewReader(config)
	checkOutput, err := checkCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("check generated config failed: %v\n%s", err, checkOutput)
	}
	got := string(checkOutput)
	for _, want := range []string{
		"ok: testloop command " + binary,
		"ok: testloop url " + httpURL,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("check output missing %q:\n%s", want, got)
		}
	}
	if strings.Count(got, "ok: testloop") != 4 {
		t.Fatalf("expected four validated testloop entries, got:\n%s", got)
	}
}

func buildMainBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "testloop-mcp")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", binary, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build binary: %v\n%s", err, output)
	}
	return binary
}

func TestDoctorClientConfigReportsRecommendedPaths(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer

	code := doctorClientConfig(&stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, stderr=%q", code, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{
		"binary:",
		"recommended_config_paths:",
		filepath.Join(dir, ".codex", "config.toml"),
		filepath.Join(dir, ".claude", "claude_desktop_config.json"),
		filepath.Join(".cursor", "mcp.json"),
		"existing_config_checks:",
		"- none found",
		"suggestion: start with `testloop-mcp --print-config=codex --config-command",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in doctor output:\n%s", want, got)
		}
	}
}

func TestDoctorClientConfigChecksExistingConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)
	binary := filepath.Join(dir, "testloop-mcp")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	codexDir := filepath.Join(dir, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatalf("mkdir codex: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte("[mcp_servers.testloop]\ncommand = \""+binary+"\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := doctorClientConfig(&stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Codex: ok: testloop command "+binary) {
		t.Fatalf("unexpected doctor output:\n%s", stdout.String())
	}
}

func TestDoctorClientConfigReportsMissingTestloopServer(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)
	codexDir := filepath.Join(dir, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatalf("mkdir codex: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte("[mcp_servers.context7]\ncommand = \"true\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := doctorClientConfig(&stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, stderr=%q", code, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{
		"Codex: missing testloop server",
		"other_servers: context7",
		"suggestion: run `testloop-mcp --print-config=codex --config-command",
		filepath.Join(dir, ".codex", "config.toml"),
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in doctor output:\n%s", want, got)
		}
	}
}

func TestDoctorClientConfigReportsMissingPathSuggestion(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("PATH", dir)
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer

	code := doctorClientConfig(&stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, stderr=%q", code, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{
		"path: missing testloop-mcp",
		"suggestion: install testloop-mcp or pass an absolute binary path with --config-command",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in doctor output:\n%s", want, got)
		}
	}
}

func TestHTTPMuxHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	newHTTPMux(newTestloopServer(), false).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok\n" {
		t.Fatalf("body = %q, want ok", rec.Body.String())
	}
}
