# 5 分钟接入向导

这条路径面向已经想把 testloop-mcp 接到 Codex、Claude 或 Cursor 的用户。完整安装、校验和 Docker 说明见 [安装与接入](./installation.md)。

## 1. 安装

macOS / Linux 推荐 Homebrew：

```bash
brew tap sleticalboy/tap
brew install testloop-mcp
```

也可以使用安装脚本：

```bash
curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/install.sh | sh
```

确认命令可用：

```bash
testloop-mcp --print-config=codex
testloop-testgen --help
```

## 2. 自检

### 基础安装验收

如果是源码 checkout，先运行安装后自检脚本：

```bash
scripts/verify-client-setup.sh "$(command -v testloop-mcp)"
```

这一步用于确认二进制可执行、`--version` / `--doctor-config` 可运行、客户端配置片段能 roundtrip 校验，并且 HTTP `/healthz` 可探活。

如果要确认当前 PATH 指向的就是预期版本：

```bash
TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.3 scripts/verify-client-setup.sh "$(command -v testloop-mcp)"
```

如果当前机器 `127.0.0.1:18080` 已被占用：

```bash
TESTLOOP_MCP_VERIFY_HTTP_ADDR=127.0.0.1:18081 scripts/verify-client-setup.sh "$(command -v testloop-mcp)"
```

如果只接 stdio 客户端，不需要 HTTP 探活：

```bash
TESTLOOP_MCP_VERIFY_SKIP_HTTP=true scripts/verify-client-setup.sh "$(command -v testloop-mcp)"
```

### 深度协议验收

如果要进一步确认真实 MCP 客户端可以通过 stdio 和 Streamable HTTP 启动该二进制并调用工具：

```bash
scripts/verify-mcp-process-smoke.sh "$(command -v testloop-mcp)"
```

这一步会调用 `tools/list` 和轻量 `parse_results`，并校验 `structuredContent` 与文本 JSON fallback 一致。只想验证单一路径时：

```bash
TESTLOOP_MCP_CLIENT_SMOKE_TRANSPORT=stdio scripts/verify-mcp-process-smoke.sh "$(command -v testloop-mcp)"
TESTLOOP_MCP_CLIENT_SMOKE_TRANSPORT=http scripts/verify-mcp-process-smoke.sh "$(command -v testloop-mcp)"
```

没有源码 checkout 时，可以手动执行同等检查：

```bash
testloop-mcp --doctor-config
testloop-mcp --print-config=all --config-command="$(command -v testloop-mcp)" | testloop-mcp --check-config -
```

## 3. 写入客户端配置

Codex：

```bash
testloop-mcp --print-config=codex --config-command="$(command -v testloop-mcp)"
```

把输出写入 `~/.codex/config.toml`。

Claude Code / Claude Desktop：

```bash
testloop-mcp --print-config=claude --config-command="$(command -v testloop-mcp)"
```

把输出合并到 `~/.claude/claude_desktop_config.json`。

Cursor：

```bash
testloop-mcp --print-config=cursor --config-command="$(command -v testloop-mcp)"
```

把输出写入 `.cursor/mcp.json`。

写入后校验：

```bash
testloop-mcp --check-config ~/.codex/config.toml
testloop-mcp --check-config ~/.claude/claude_desktop_config.json
testloop-mcp --check-config .cursor/mcp.json
```

## 4. 重启客户端

修改 MCP 配置后，重启 Codex、Claude 或 Cursor，让客户端重新加载 MCP server 列表。

## 5. 跑一个最小闭环

在支持 MCP 的客户端里优先试这条顺序：

1. 调用 `run_tests`，传入当前项目测试路径。
2. 如果失败，读取 `fix_suggestions[].repair_task`。
3. 修复后再次调用 `run_tests`。
4. 需要补覆盖率时，先生成覆盖率报告，再调用 `parse_coverage` 或 `validate_coverage_task`。

源码 checkout 可直接运行本地 demo：

```bash
go run ./examples/mcp-client-demo
```

这个 demo 会展示 `run_tests -> repair_task -> rerun -> parse_coverage` 的最小 Agent 消费路径。

预期输出和可重复验收方式见 [Agent 闭环展示案例](./showcase-agent-loop.md)。
