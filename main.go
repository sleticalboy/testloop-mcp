package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/tools"
)

type serverConfig struct {
	transport     string
	addr          string
	stateless     bool
	printConfig   string
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
	configCommand := flags.String("config-command", "", "配置片段中的 testloop-mcp 二进制路径，默认使用当前可执行文件路径")
	configHTTPURL := flags.String("config-http-url", "http://localhost:8080/mcp", "Codex HTTP 配置片段中的 MCP endpoint")
	if err := flags.Parse(args); err != nil {
		return serverConfig{}, 2
	}

	cfg := serverConfig{
		transport:     *transport,
		addr:          *addr,
		stateless:     *stateless,
		printConfig:   *printConfig,
		configCommand: *configCommand,
		configHTTPURL: *configHTTPURL,
	}
	switch cfg.transport {
	case "stdio", "http":
		return cfg, 0
	default:
		fmt.Fprintf(stderr, "不支持的传输模式: %s\n可用值: stdio, http\n", cfg.transport)
		return cfg, 1
	}
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

func newTestloopServer() *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "testloop-mcp", Version: "0.4.7"},
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
