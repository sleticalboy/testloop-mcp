# 安装与接入

本文面向想直接使用 testloop-mcp 的用户，覆盖 Homebrew、二进制下载、校验、本地构建、Docker 运行和常见 MCP 客户端接入方式。

如果只想尽快接入 Codex、Claude 或 Cursor，先看 [5 分钟接入向导](./quickstart.md)。

## 推荐方式：Homebrew

macOS / Linux 可以通过 `sleticalboy/tap` 安装：

```bash
brew tap sleticalboy/tap
brew install testloop-mcp
```

验证命令：

```bash
testloop-mcp --help
testloop-testgen --help
```

## 安装脚本

安装脚本会检测当前系统和 CPU 架构，优先下载匹配的 GitHub Release 资产并校验 `checksums.txt` 或单资产 `.sha256`。Linux/macOS 会安装 tarball，Git Bash/MSYS/Cygwin 等 Windows shell 会安装 `windows_amd64` 或 `windows_arm64` zip；当前 release 没有对应资产时，会自动回退到 `go install`。

```bash
curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/install.sh | sh
```

可选环境变量：

```bash
TESTLOOP_MCP_VERSION=v0.5.4 sh scripts/install.sh
TESTLOOP_MCP_INSTALL_DIR=/usr/local/bin sh scripts/install.sh
```

在 Git Bash/MSYS/Cygwin 等 Windows shell 下，脚本会安装 `testloop-mcp.exe` 和 `testloop-testgen.exe`。默认安装目录仍是 `$HOME/.local/bin`，需要确保该目录在 `PATH` 中：

```bash
mkdir -p "$HOME/.local/bin"
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
TESTLOOP_MCP_VERSION=v0.5.4 sh scripts/install.sh
testloop-mcp.exe --help
testloop-testgen.exe --help
```

也可以显式安装到已有的 Windows 用户 bin 目录：

```bash
TESTLOOP_MCP_INSTALL_DIR="$USERPROFILE/bin" TESTLOOP_MCP_VERSION=v0.5.4 sh scripts/install.sh
```

维护者调试平台选择时也可以显式覆盖检测结果：

```bash
TESTLOOP_MCP_OS=windows TESTLOOP_MCP_ARCH=amd64 TESTLOOP_MCP_VERSION=v0.5.4 sh scripts/install.sh
TESTLOOP_MCP_OS=windows TESTLOOP_MCP_ARCH=arm64 TESTLOOP_MCP_VERSION=v0.5.4 sh scripts/install.sh
```

脚本会安装两个命令：

- `testloop-mcp`
- `testloop-testgen`

Windows shell 下对应文件名为 `testloop-mcp.exe` 和 `testloop-testgen.exe`。

## 手动下载 Release 二进制

当前 `v0.5.4` Release 已提供以下产物：

- `testloop-mcp_v0.5.4_linux_amd64.tar.gz`
- `testloop-mcp_v0.5.4_linux_amd64.tar.gz.sha256`
- `testloop-mcp_v0.5.4_linux_arm64.tar.gz`
- `testloop-mcp_v0.5.4_linux_arm64.tar.gz.sha256`
- `testloop-mcp_v0.5.4_darwin_arm64.tar.gz`
- `testloop-mcp_v0.5.4_darwin_arm64.tar.gz.sha256`
- `testloop-mcp_v0.5.4_windows_amd64.zip`
- `testloop-mcp_v0.5.4_windows_amd64.zip.sha256`
- `testloop-mcp_v0.5.4_windows_arm64.zip`
- `testloop-mcp_v0.5.4_windows_arm64.zip.sha256`

```bash
curl -LO https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.4/testloop-mcp_v0.5.4_linux_amd64.tar.gz
curl -LO https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.4/testloop-mcp_v0.5.4_linux_amd64.tar.gz.sha256
sha256sum -c testloop-mcp_v0.5.4_linux_amd64.tar.gz.sha256
tar -xzf testloop-mcp_v0.5.4_linux_amd64.tar.gz
chmod +x testloop-mcp testloop-testgen
./testloop-mcp --help
./testloop-testgen --help
```

Release 产物会同时提供单资产 `.sha256` 文件。安装脚本会优先使用聚合 `checksums.txt`，不存在时自动使用对应资产的 `.sha256`。

Windows amd64/arm64 可直接下载 zip；将 `$arch` 设为 `amd64` 或 `arm64`：

```powershell
$arch = "amd64"
curl.exe -LO "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.4/testloop-mcp_v0.5.4_windows_$arch.zip"
curl.exe -LO "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.5.4/testloop-mcp_v0.5.4_windows_$arch.zip.sha256"
$expected = (Get-Content ".\testloop-mcp_v0.5.4_windows_$arch.zip.sha256").Split()[0]
$actual = (Get-FileHash ".\testloop-mcp_v0.5.4_windows_$arch.zip" -Algorithm SHA256).Hash.ToLower()
if ($actual -ne $expected) { throw "checksum mismatch" }
Expand-Archive ".\testloop-mcp_v0.5.4_windows_$arch.zip"
& ".\testloop-mcp_v0.5.4_windows_$arch\testloop-mcp.exe" --help
& ".\testloop-mcp_v0.5.4_windows_$arch\testloop-testgen.exe" --help
```

当前 release 未覆盖的平台可以使用安装脚本的 `go install` 回退，或按下文从源码构建。

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

## Homebrew tap 维护

维护者可以用仓库内的公式和脚本更新 `sleticalboy/homebrew-tap`。

仓库内的公式草案：

```bash
Formula/testloop-mcp.rb
```

只更新当前仓库内的公式：

```bash
scripts/generate-homebrew-formula.sh v0.5.4
ruby -c Formula/testloop-mcp.rb
```

同步到 `sleticalboy/homebrew-tap` 工作区：

```bash
scripts/update-homebrew-tap.sh v0.5.4 ../homebrew-tap
```

不传 `tap-dir` 时，脚本会把 `sleticalboy/homebrew-tap` 克隆到临时目录并更新公式。默认不会自动提交；确认无误后可用以下环境变量提交和推送 tap 仓库：

```bash
TESTLOOP_MCP_TAP_COMMIT=1 TESTLOOP_MCP_TAP_PUSH=1 scripts/update-homebrew-tap.sh v0.5.4 ../homebrew-tap
```

也可以在 GitHub Actions 里手动触发 `Homebrew Tap` workflow，输入 release tag 后创建或更新 `sleticalboy/homebrew-tap` 的 formula PR。

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

可以先生成本机路径对应的配置片段：

```bash
testloop-mcp --print-config=codex
```

如果需要指定配置里的二进制路径，追加 `--config-command=/absolute/path/to/testloop-mcp`。

配置写入后可以校验 `command` 是否存在且可执行，或 `url` 是否是合法 HTTP endpoint：

```bash
testloop-mcp --check-config ~/.codex/config.toml
testloop-mcp --check-config ~/.claude/claude_desktop_config.json
testloop-mcp --check-config .cursor/mcp.json
```

校验失败时会输出对应的 `--print-config` 或 `--doctor-config` 建议，便于直接修复缺失、不可执行或 URL 不合法的配置。

也可以从 stdin 校验：

```bash
testloop-mcp --print-config=codex | testloop-mcp --check-config -
```

源码仓库的测试会构建当前 `testloop-mcp` 二进制，并验证 `--print-config=all --config-command=<built-binary>` 生成的 Codex、Codex HTTP、Claude 和 Cursor 片段可以被同一个二进制 `--check-config -` 校验通过。这个 smoke 用来防止配置生成、配置解析和安装接入文档之间发生漂移。

源码 checkout 中也提供用户侧安装后自检脚本：

```bash
scripts/verify-client-setup.sh /absolute/path/to/testloop-mcp
```

这个脚本适合做基础安装验收：它会依次验证二进制可执行、`--version` 可运行、`--doctor-config` 可运行、`--print-config=all` 与 `--check-config -` 可以闭环，并启动一次 HTTP 模式检查 `/healthz`。如果当前机器的 `127.0.0.1:18080` 已被占用，可以通过 `TESTLOOP_MCP_VERIFY_HTTP_ADDR=127.0.0.1:18081` 指定其他端口；只想验证 stdio 配置时可以设置 `TESTLOOP_MCP_VERIFY_SKIP_HTTP=true`。

如果需要确认安装产物没有指向旧版本，可以加版本门禁：

```bash
TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.4 scripts/verify-client-setup.sh /absolute/path/to/testloop-mcp
```

如果需要做深度协议验收，验证真实 MCP 客户端进程接入，而不仅是配置 roundtrip 和 HTTP 探活，可以运行：

```bash
scripts/verify-mcp-process-smoke.sh /absolute/path/to/testloop-mcp
```

该脚本会使用 MCP SDK 客户端通过 stdio 和 Streamable HTTP 启动指定二进制，调用 `tools/list` 和轻量 `parse_results`，并校验 `structuredContent` 与文本 JSON fallback 一致。只想验证单一路径时可以设置 `TESTLOOP_MCP_CLIENT_SMOKE_TRANSPORT=stdio` 或 `http`。

如果不确定应该写入哪个配置文件，先运行本机诊断：

```bash
testloop-mcp --doctor-config
```

诊断只读取配置，不会写入文件；如果配置文件存在但没有 `testloop` server，会列出已发现的其他 MCP server，并给出可复制的 `--print-config` 修复建议。

`~/.codex/config.toml`:

```toml
[mcp_servers.testloop]
command = "/absolute/path/to/testloop-mcp"
```

如果使用 HTTP 模式，并且当前 Codex 版本支持 URL 型 MCP server，可以配置：

```bash
testloop-mcp --print-config=codex-http
```

```toml
[mcp_servers.testloop]
url = "http://localhost:8080/mcp"
```

## Claude Code / Claude Desktop 配置示例

可以先生成本机路径对应的配置片段：

```bash
testloop-mcp --print-config=claude
```

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

可以先生成本机路径对应的配置片段：

```bash
testloop-mcp --print-config=cursor
```

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
