package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/tools"
)

const appVersion = "0.5.16"

type serverConfig struct {
	transport     string
	addr          string
	stateless     bool
	printConfig   string
	checkConfig   string
	doctorConfig  bool
	version       bool
	configCommand string
	configHTTPURL string
}

func parseServerConfig(args []string, stderr io.Writer) (serverConfig, int) {
	flags := flag.NewFlagSet("testloop-mcp", flag.ContinueOnError)
	flags.SetOutput(stderr)
	transport := flags.String("transport", "stdio", "传输模式: stdio 或 http")
	addr := flags.String("addr", ":8080", "HTTP 模式监听地址 (仅 --transport=http 时生效)")
	stateless := flags.Bool("stateless", false, "HTTP 无状态模式 (仅 --transport=http 时生效)")
	printConfig := flags.String("print-config", "", "打印 MCP 客户端配置片段: all、codex、codex-http、claude 或 cursor")
	checkConfig := flags.String("check-config", "", "检查 MCP 客户端配置文件路径；使用 - 从 stdin 读取")
	doctorConfig := flags.Bool("doctor-config", false, "诊断本机 MCP 客户端配置路径和 testloop-mcp 可执行文件")
	version := flags.Bool("version", false, "打印 testloop-mcp 版本并退出")
	configCommand := flags.String("config-command", "", "配置片段中的 testloop-mcp 二进制路径，默认使用当前可执行文件路径")
	configHTTPURL := flags.String("config-http-url", "http://localhost:8080/mcp", "Codex HTTP 配置片段中的 MCP endpoint")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return serverConfig{}, 0
		}
		return serverConfig{}, 2
	}

	cfg := serverConfig{
		transport:     *transport,
		addr:          *addr,
		stateless:     *stateless,
		printConfig:   *printConfig,
		checkConfig:   *checkConfig,
		doctorConfig:  *doctorConfig,
		version:       *version,
		configCommand: *configCommand,
		configHTTPURL: *configHTTPURL,
	}
	if countConfigActions(cfg) > 1 {
		fmt.Fprintln(stderr, "--print-config、--check-config、--doctor-config 和 --version 不能同时使用")
		return cfg, 1
	}
	switch cfg.transport {
	case "stdio", "http":
		return cfg, 0
	default:
		fmt.Fprintf(stderr, "不支持的传输模式: %s\n可用值: stdio, http\n", cfg.transport)
		return cfg, 1
	}
}

func countConfigActions(cfg serverConfig) int {
	count := 0
	if cfg.printConfig != "" {
		count++
	}
	if cfg.checkConfig != "" {
		count++
	}
	if cfg.doctorConfig {
		count++
	}
	if cfg.version {
		count++
	}
	return count
}

func printClientConfig(cfg serverConfig, stdout, stderr io.Writer) int {
	command := cfg.configCommand
	if command == "" {
		exe, err := os.Executable()
		if err != nil {
			fmt.Fprintf(stderr, "获取当前可执行文件路径失败: %v\n", err)
			return 1
		}
		command = exe
	}

	emitCodex := func() {
		fmt.Fprintf(stdout, "# ~/.codex/config.toml\n[mcp_servers.testloop]\ncommand = %s\n", strconv.Quote(command))
	}
	emitCodexHTTP := func() {
		fmt.Fprintf(stdout, "# ~/.codex/config.toml\n[mcp_servers.testloop]\nurl = %s\n", strconv.Quote(cfg.configHTTPURL))
	}
	emitClaude := func() {
		fmt.Fprintf(stdout, "# ~/.claude/claude_desktop_config.json\n{\n  \"mcpServers\": {\n    \"testloop\": {\n      \"command\": %s\n    }\n  }\n}\n", strconv.Quote(command))
	}
	emitCursor := func() {
		fmt.Fprintf(stdout, "# .cursor/mcp.json\n{\n  \"mcpServers\": {\n    \"testloop\": {\n      \"command\": %s\n    }\n  }\n}\n", strconv.Quote(command))
	}
	separator := func() {
		fmt.Fprint(stdout, "\n---\n\n")
	}

	switch cfg.printConfig {
	case "all":
		emitCodex()
		separator()
		emitCodexHTTP()
		separator()
		emitClaude()
		separator()
		emitCursor()
	case "codex":
		emitCodex()
	case "codex-http", "http":
		emitCodexHTTP()
	case "claude", "claude-code", "claude-desktop":
		emitClaude()
	case "cursor":
		emitCursor()
	default:
		fmt.Fprintf(stderr, "不支持的客户端配置类型: %s\n可用值: all, codex, codex-http, claude, cursor\n", cfg.printConfig)
		return 1
	}
	return 0
}

type configPathInfo struct {
	Name string
	Path string
}

type jsonClientConfig struct {
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
}

type clientConfigEntry struct {
	Name    string
	Command string
	URL     string
}

func checkClientConfig(cfg serverConfig, stdin io.Reader, stdout, stderr io.Writer) int {
	data, err := readConfigInput(cfg.checkConfig, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "读取配置失败: %v\n", err)
		return 1
	}
	entries := parseClientConfigEntries(data)
	if len(entries) == 0 {
		fmt.Fprintln(stderr, "未找到 MCP server 配置；需要 command 或 url")
		fmt.Fprintln(stderr, "suggestion: run `testloop-mcp --doctor-config` to locate client config files, then generate a testloop snippet with `testloop-mcp --print-config=codex`")
		return 1
	}

	ok := true
	clientName := inferClientConfigName(cfg.checkConfig)
	for _, entry := range entries {
		switch {
		case strings.TrimSpace(entry.Command) != "":
			if err := validateConfigCommand(entry.Command); err != nil {
				fmt.Fprintf(stderr, "error: %s command 无效: %v\n", entry.Name, err)
				fmt.Fprintf(stderr, "suggestion: update %s with `testloop-mcp --print-config=%s --config-command %s`, or run `testloop-mcp --doctor-config`\n", entry.Name, clientName, strconv.Quote(suggestedConfigCommand()))
				ok = false
			} else {
				fmt.Fprintf(stdout, "ok: %s command %s\n", entry.Name, entry.Command)
			}
		case strings.TrimSpace(entry.URL) != "":
			if err := validateConfigURL(entry.URL); err != nil {
				fmt.Fprintf(stderr, "error: %s url 无效: %v\n", entry.Name, err)
				fmt.Fprintf(stderr, "suggestion: use an http(s) Streamable HTTP endpoint, for example `testloop-mcp --print-config=codex-http --config-http-url http://localhost:8080/mcp`\n")
				ok = false
			} else {
				fmt.Fprintf(stdout, "ok: %s url %s\n", entry.Name, entry.URL)
			}
		}
	}
	if !ok {
		return 1
	}
	return 0
}

func inferClientConfigName(path string) string {
	normalized := filepath.ToSlash(strings.ToLower(path))
	switch {
	case strings.Contains(normalized, "claude"):
		return "claude"
	case strings.Contains(normalized, "cursor") || strings.HasSuffix(normalized, ".cursor/mcp.json"):
		return "cursor"
	case strings.Contains(normalized, ".codex") || strings.HasSuffix(normalized, "config.toml"):
		return "codex"
	default:
		return "codex"
	}
}

func suggestedConfigCommand() string {
	if pathBinary, err := exec.LookPath("testloop-mcp"); err == nil {
		return pathBinary
	}
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return "testloop-mcp"
}

func doctorClientConfig(stdout, stderr io.Writer) int {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(stderr, "获取当前可执行文件路径失败: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "binary: %s\n", exe)

	configCommand := exe
	if pathBinary, err := exec.LookPath("testloop-mcp"); err == nil {
		fmt.Fprintf(stdout, "path: ok %s\n", pathBinary)
		configCommand = pathBinary
	} else {
		fmt.Fprintf(stdout, "path: missing testloop-mcp\n")
		fmt.Fprintln(stdout, "  suggestion: install testloop-mcp or pass an absolute binary path with --config-command")
	}

	fmt.Fprintln(stdout, "recommended_config_paths:")
	for _, info := range recommendedConfigPaths() {
		fmt.Fprintf(stdout, "- %s: %s\n", info.Name, info.Path)
	}

	fmt.Fprintln(stdout, "existing_config_checks:")
	found := false
	allOK := true
	for _, info := range recommendedConfigPaths() {
		data, err := os.ReadFile(info.Path)
		if err != nil {
			continue
		}
		found = true
		fmt.Fprintf(stdout, "- %s: ", info.Name)

		entries := parseClientConfigEntries(data)
		if len(entries) == 0 {
			allOK = false
			fmt.Fprintln(stdout, "invalid MCP config; no command or url entries found")
			fmt.Fprintf(stdout, "  suggestion: run `testloop-mcp --print-config=%s --config-command %s` and merge it into %s\n", clientConfigName(info.Name), strconv.Quote(configCommand), info.Path)
			continue
		}
		testloopEntries := filterConfigEntries(entries, "testloop")
		if len(testloopEntries) == 0 {
			fmt.Fprintln(stdout, "missing testloop server")
			fmt.Fprintf(stdout, "  other_servers: %s\n", strings.Join(configEntryNames(entries), ", "))
			fmt.Fprintf(stdout, "  suggestion: run `testloop-mcp --print-config=%s --config-command %s` and merge it into %s\n", clientConfigName(info.Name), strconv.Quote(configCommand), info.Path)
			continue
		}
		if !validateConfigEntries(stdout, stdout, testloopEntries) {
			allOK = false
			fmt.Fprintf(stdout, "  suggestion: update the testloop server with `testloop-mcp --print-config=%s --config-command %s`\n", clientConfigName(info.Name), strconv.Quote(configCommand))
		}
	}
	if !found {
		fmt.Fprintln(stdout, "- none found")
		fmt.Fprintf(stdout, "  suggestion: start with `testloop-mcp --print-config=codex --config-command %s` or choose claude/cursor for another client\n", strconv.Quote(configCommand))
	}
	if !allOK {
		return 1
	}
	return 0
}

func clientConfigName(name string) string {
	switch strings.ToLower(name) {
	case "codex":
		return "codex"
	case "claude":
		return "claude"
	case "cursor":
		return "cursor"
	default:
		return "codex"
	}
}

func validateConfigEntries(stdout, stderr io.Writer, entries []clientConfigEntry) bool {
	ok := true
	for _, entry := range entries {
		switch {
		case strings.TrimSpace(entry.Command) != "":
			if err := validateConfigCommand(entry.Command); err != nil {
				fmt.Fprintf(stderr, "error: %s command 无效: %v\n", entry.Name, err)
				ok = false
			} else {
				fmt.Fprintf(stdout, "ok: %s command %s\n", entry.Name, entry.Command)
			}
		case strings.TrimSpace(entry.URL) != "":
			if err := validateConfigURL(entry.URL); err != nil {
				fmt.Fprintf(stderr, "error: %s url 无效: %v\n", entry.Name, err)
				ok = false
			} else {
				fmt.Fprintf(stdout, "ok: %s url %s\n", entry.Name, entry.URL)
			}
		}
	}
	return ok
}

func filterConfigEntries(entries []clientConfigEntry, name string) []clientConfigEntry {
	var filtered []clientConfigEntry
	for _, entry := range entries {
		if entry.Name == name {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func configEntryNames(entries []clientConfigEntry) []string {
	seen := map[string]bool{}
	var names []string
	for _, entry := range entries {
		if seen[entry.Name] {
			continue
		}
		seen[entry.Name] = true
		names = append(names, entry.Name)
	}
	return names
}

func recommendedConfigPaths() []configPathInfo {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}
	return []configPathInfo{
		{Name: "Codex", Path: filepath.Join(home, ".codex", "config.toml")},
		{Name: "Claude", Path: filepath.Join(home, ".claude", "claude_desktop_config.json")},
		{Name: "Cursor", Path: filepath.Join(".cursor", "mcp.json")},
	}
}

func readConfigInput(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(path)
}

func parseClientConfigEntries(data []byte) []clientConfigEntry {
	text := string(data)
	if strings.Contains(text, "\n---\n") {
		var entries []clientConfigEntry
		for _, part := range strings.Split(text, "\n---\n") {
			entries = append(entries, parseSingleClientConfigEntries([]byte(strings.TrimSpace(part)))...)
		}
		return entries
	}
	return parseSingleClientConfigEntries(data)
}

func parseSingleClientConfigEntries(data []byte) []clientConfigEntry {
	trimmed := strings.TrimSpace(string(data))
	if idx := strings.Index(trimmed, "{"); idx >= 0 {
		if entries := parseJSONClientConfigEntries([]byte(trimmed[idx:])); len(entries) > 0 {
			return entries
		}
	}
	if entries := parseJSONClientConfigEntries([]byte(trimmed)); len(entries) > 0 {
		return entries
	}
	return parseTOMLClientConfigEntries(trimmed)
}

func parseJSONClientConfigEntries(data []byte) []clientConfigEntry {
	var cfg jsonClientConfig
	if err := json.Unmarshal(data, &cfg); err != nil || len(cfg.MCPServers) == 0 {
		return nil
	}
	entries := make([]clientConfigEntry, 0, len(cfg.MCPServers))
	for name, raw := range cfg.MCPServers {
		var server struct {
			Command string `json:"command"`
			URL     string `json:"url"`
		}
		if err := json.Unmarshal(raw, &server); err != nil {
			continue
		}
		if strings.TrimSpace(server.Command) == "" && strings.TrimSpace(server.URL) == "" {
			continue
		}
		entries = append(entries, clientConfigEntry{Name: name, Command: server.Command, URL: server.URL})
	}
	return entries
}

func parseTOMLClientConfigEntries(data string) []clientConfigEntry {
	var entries []clientConfigEntry
	current := ""
	for _, line := range strings.Split(data, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			section := strings.Trim(trimmed, "[]")
			if strings.HasPrefix(section, "mcp_servers.") {
				current = strings.TrimPrefix(section, "mcp_servers.")
			} else {
				current = ""
			}
			continue
		}
		if current == "" {
			continue
		}
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		unquoted, err := strconv.Unquote(value)
		if err == nil {
			value = unquoted
		}
		if key == "command" && value != "" {
			entries = append(entries, clientConfigEntry{Name: current, Command: value})
		}
		if key == "url" && value != "" {
			entries = append(entries, clientConfigEntry{Name: current, URL: value})
		}
	}
	return entries
}

func validateConfigCommand(command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("command 为空")
	}
	if strings.Contains(command, "/") {
		info, err := os.Stat(command)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return fmt.Errorf("是目录，不是可执行文件")
		}
		if info.Mode()&0111 == 0 {
			return fmt.Errorf("文件不可执行")
		}
		return nil
	}
	_, err := exec.LookPath(command)
	return err
}

func validateConfigURL(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("scheme 必须是 http 或 https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("缺少 host")
	}
	return nil
}

func newTestloopServer() *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "testloop-mcp", Version: appVersion},
		nil,
	)
	tools.Register(server)
	return server
}

func newHTTPMux(server *mcp.Server, stateless bool) *http.ServeMux {
	handler := mcp.NewStreamableHTTPHandler(
		func(req *http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: stateless},
	)
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	return mux
}

func main() {
	cfg, code := parseServerConfig(os.Args[1:], os.Stderr)
	if code != 0 {
		os.Exit(code)
	}
	if cfg.printConfig != "" {
		os.Exit(printClientConfig(cfg, os.Stdout, os.Stderr))
	}
	if cfg.checkConfig != "" {
		os.Exit(checkClientConfig(cfg, os.Stdin, os.Stdout, os.Stderr))
	}
	if cfg.doctorConfig {
		os.Exit(doctorClientConfig(os.Stdout, os.Stderr))
	}
	if cfg.version {
		fmt.Fprintf(os.Stdout, "testloop-mcp %s\n", appVersion)
		os.Exit(0)
	}

	server := newTestloopServer()

	switch cfg.transport {
	case "stdio":
		// stdio 模式：从 stdin/stdout 读取 JSON-RPC
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			log.Fatal(err)
		}

	case "http":
		// Streamable HTTP 模式
		httpServer := &http.Server{
			Addr:    cfg.addr,
			Handler: newHTTPMux(server, cfg.stateless),
		}

		// 优雅退出
		go func() {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh
			log.Println("正在关闭...")
			_ = httpServer.Shutdown(context.Background())
		}()

		log.Printf("testloop-mcp Streamable HTTP 服务启动，监听 %s (stateless=%v)", cfg.addr, cfg.stateless)
		log.Println("端点: POST/GET/DELETE http://" + cfg.addr + "/mcp")
		log.Println("健康检查: GET http://" + cfg.addr + "/healthz")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}
}
