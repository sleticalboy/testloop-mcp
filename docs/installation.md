# 安装与接入

本文面向想直接使用 testloop-mcp 的用户，覆盖二进制下载、校验、本地构建、Docker 运行和常见 MCP 客户端接入方式。

## 推荐方式：安装脚本

安装脚本会检测当前系统和 CPU 架构，优先下载匹配的 GitHub Release 资产并校验 `checksums.txt`。当前 release 没有对应资产时，会自动回退到 `go install`。

```bash
curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/install.sh | sh
```

可选环境变量：

```bash
TESTLOOP_MCP_VERSION=v0.4.2 sh scripts/install.sh
TESTLOOP_MCP_INSTALL_DIR=/usr/local/bin sh scripts/install.sh
```

脚本会安装两个命令：

- `testloop-mcp`
- `testloop-testgen`

## 手动下载 Release 二进制

当前 `v0.4.2` Release 已提供以下产物：

- `testloop-mcp_v0.4.2_linux_amd64.tar.gz`
- `testloop-mcp_v0.4.2_linux_arm64.tar.gz`
- `testloop-mcp_v0.4.2_darwin_arm64.tar.gz`
- `checksums.txt`

```bash
curl -LO https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.2/testloop-mcp_v0.4.2_linux_amd64.tar.gz
curl -LO https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.2/checksums.txt
sha256sum -c checksums.txt
tar -xzf testloop-mcp_v0.4.2_linux_amd64.tar.gz
chmod +x testloop-mcp testloop-testgen
./testloop-mcp --help
./testloop-testgen --help
```

后续自动发布产物会同时提供单资产 `.sha256` 文件。安装脚本会优先使用聚合 `checksums.txt`，不存在时自动使用对应 tarball 的 `.sha256`。

Windows 和当前 release 未覆盖的平台可以使用安装脚本的 `go install` 回退，或按下文从源码构建。

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

也可以直接用 `go install` 安装到 `$GOBIN` 或 `$GOPATH/bin`：

```bash
go install github.com/sleticalboy/testloop-mcp@latest
go install github.com/sleticalboy/testloop-mcp/cmd/testgen@latest
```

## Homebrew tap 准备

仓库内提供了可复制到 tap 仓库的公式草案：

```bash
Formula/testloop-mcp.rb
```

更新到最新 GitHub Release：

```bash
scripts/generate-homebrew-formula.sh v0.4.2
ruby -c Formula/testloop-mcp.rb
```

正式 tap 仓库创建后，可以把 `Formula/testloop-mcp.rb` 同步过去，再提供 `brew install sleticalboy/tap/testloop-mcp` 路径。

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
