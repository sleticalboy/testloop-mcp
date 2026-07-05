# 安装与接入

本文面向想直接使用 testloop-mcp 的用户，覆盖二进制下载、校验、本地构建、Docker 运行和常见 MCP 客户端接入方式。

## 推荐方式：下载 Release 二进制

当前 Release 已提供 Linux amd64 产物：

- `testloop-mcp_v0.4.0_linux_amd64.tar.gz`
- `checksums.txt`

```bash
curl -LO https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.0/testloop-mcp_v0.4.0_linux_amd64.tar.gz
curl -LO https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.0/checksums.txt
sha256sum -c checksums.txt
tar -xzf testloop-mcp_v0.4.0_linux_amd64.tar.gz
chmod +x testloop-mcp testloop-testgen
./testloop-mcp --help
./testloop-testgen --help
```

macOS、Windows 和其他架构暂未发布预构建二进制。可以先用源码构建方式安装。

## 从源码构建

前置要求：

- Go 1.25+
- CGO 可用的 C 编译工具链

```bash
git clone https://github.com/sleticalboy/testloop-mcp.git
cd testloop-mcp
go test ./...
go build -o testloop-mcp .
go build -o testloop-testgen ./cmd/testgen
```

当前 main 分支也可以直接用 `go install` 安装到 `$GOBIN` 或 `$GOPATH/bin`：

```bash
go install github.com/sleticalboy/testloop-mcp@main
go install github.com/sleticalboy/testloop-mcp/cmd/testgen@main
```

等包含 module path 修正的新版本发布后，可以把 `@main` 换成 `@latest`。

## Docker 运行

```bash
docker compose up -d
curl http://localhost:8080/healthz
docker compose logs -f
docker compose down
```

Docker 默认以 Streamable HTTP 模式启动：

- MCP endpoint: `http://localhost:8080/mcp`
- Health check: `http://localhost:8080/healthz`

## stdio 模式

stdio 是本地 MCP 客户端最常见的接入方式。直接把 `testloop-mcp` 二进制路径配置给客户端即可。

```bash
/absolute/path/to/testloop-mcp
```

## Streamable HTTP 模式

需要远程或容器化部署时可以使用 HTTP 模式：

```bash
./testloop-mcp --transport http --addr :8080
curl http://localhost:8080/healthz
```

MCP endpoint 是：

```text
http://localhost:8080/mcp
```

只有在客户端支持远程 MCP / Streamable HTTP 时才使用这个地址；否则优先使用 stdio。

## Codex 配置示例

`~/.codex/config.toml`:

```toml
[mcp_servers.testloop]
command = "/absolute/path/to/testloop-mcp"
```

如果使用 HTTP 模式，并且当前 Codex 版本支持 URL 型 MCP server，可以配置：

```toml
[mcp_servers.testloop]
url = "http://localhost:8080/mcp"
```

## Claude Code / Claude Desktop 配置示例

`~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "testloop": {
      "command": "/absolute/path/to/testloop-mcp"
    }
  }
}
```

## Cursor 配置示例

`.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "testloop": {
      "command": "/absolute/path/to/testloop-mcp"
    }
  }
}
```

## 可选依赖

testloop-mcp 能在没有这些依赖时运行，但相关语言能力会受限：

| 能力 | 推荐依赖 |
| --- | --- |
| Go 测试骨架生成 | `go install github.com/cweill/gotests/gotests@latest` |
| Rust 覆盖率 | `cargo install cargo-tarpaulin` |
| Python 测试/覆盖率 | `pytest`、`pytest-cov`、`coverage` |
| Node.js 测试/覆盖率 | Jest / Vitest / Mocha + Istanbul coverage |
| Java 覆盖率 | JaCoCo Maven/Gradle 配置 |

## 快速验证

```bash
# stdio 二进制可启动并显示参数
./testloop-mcp --help

# CLI 可直接生成测试草稿
./testloop-testgen demo/calc.go /tmp/calc_test.go

# HTTP 模式健康检查
./testloop-mcp --transport http --addr :18080 &
curl http://localhost:18080/healthz
```
