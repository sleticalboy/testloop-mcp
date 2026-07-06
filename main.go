package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/tools"
)

func main() {
	transport := flag.String("transport", "stdio", "传输模式: stdio 或 http")
	addr := flag.String("addr", ":8080", "HTTP 模式监听地址 (仅 --transport=http 时生效)")
	stateless := flag.Bool("stateless", false, "HTTP 无状态模式 (仅 --transport=http 时生效)")
	flag.Parse()

	// 创建 MCP server
	server := mcp.NewServer(
		&mcp.Implementation{Name: "testloop-mcp", Version: "0.4.3"},
		nil,
	)

	// 注册所有 Tools
	tools.Register(server)

	switch *transport {
	case "stdio":
		// stdio 模式：从 stdin/stdout 读取 JSON-RPC
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			log.Fatal(err)
		}

	case "http":
		// Streamable HTTP 模式
		handler := mcp.NewStreamableHTTPHandler(
			func(req *http.Request) *mcp.Server { return server },
			&mcp.StreamableHTTPOptions{Stateless: *stateless},
		)
		mux := http.NewServeMux()
		mux.Handle("/mcp", handler)
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok\n"))
		})

		httpServer := &http.Server{
			Addr:    *addr,
			Handler: mux,
		}

		// 优雅退出
		go func() {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh
			log.Println("正在关闭...")
			_ = httpServer.Shutdown(context.Background())
		}()

		log.Printf("testloop-mcp Streamable HTTP 服务启动，监听 %s (stateless=%v)", *addr, *stateless)
		log.Println("端点: POST/GET/DELETE http://" + *addr + "/mcp")
		log.Println("健康检查: GET http://" + *addr + "/healthz")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}

	default:
		fmt.Fprintf(os.Stderr, "不支持的传输模式: %s\n可用值: stdio, http\n", *transport)
		os.Exit(1)
	}
}
