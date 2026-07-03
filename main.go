package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/binlee/testloop-mcp/tools"
)

func main() {
	// 创建 MCP server
	server := mcp.NewServer(
		&mcp.Implementation{Name: "testloop-mcp", Version: "0.1.0"},
		nil,
	)

	// 注册所有 Tools
	tools.Register(server)

	// 启动：从 stdin/stdout 读取 JSON-RPC（直到客户端断开）
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
