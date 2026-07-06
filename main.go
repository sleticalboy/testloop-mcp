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
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/tools"
)

type serverConfig struct {
	transport string
	addr      string
	stateless bool
}

func parseServerConfig(args []string, stderr io.Writer) (serverConfig, int) {
	flags := flag.NewFlagSet("testloop-mcp", flag.ContinueOnError)
	flags.SetOutput(stderr)
	transport := flags.String("transport", "stdio", "传输模式: stdio 或 http")
	addr := flags.String("addr", ":8080", "HTTP 模式监听地址 (仅 --transport=http 时生效)")
	stateless := flags.Bool("stateless", false, "HTTP 无状态模式 (仅 --transport=http 时生效)")
	if err := flags.Parse(args); err != nil {
		return serverConfig{}, 2
	}

	cfg := serverConfig{transport: *transport, addr: *addr, stateless: *stateless}
	switch cfg.transport {
	case "stdio", "http":
		return cfg, 0
	default:
		fmt.Fprintf(stderr, "不支持的传输模式: %s\n可用值: stdio, http\n", cfg.transport)
		return cfg, 1
	}
}

func newTestloopServer() *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "testloop-mcp", Version: "0.4.6"},
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
