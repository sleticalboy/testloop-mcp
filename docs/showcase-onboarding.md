# 安装到 Agent 闭环展示路径

这条路径用于公开演示 testloop-mcp 的首次接入体验：先确认安装产物和客户端配置没问题，再确认真实 MCP 进程传输可用，最后跑一条最小 Agent 测试反馈闭环。

## 一键演示

在源码 checkout 根目录执行：

```bash
scripts/showcase-onboarding.sh "$(command -v testloop-mcp)"
```

如果要生成可转发的演示制品，而不是只看终端输出：

```bash
scripts/showcase-agent-onboarding-report.sh "$(command -v testloop-mcp)"
```

默认会写出：

- `/tmp/testloop-mcp-onboarding/verification-report.md`
- `/tmp/testloop-mcp-onboarding/verification-summary.json`
- `/tmp/testloop-mcp-onboarding/agent-decision.txt`

如果要确认安装的是指定版本：

```bash
TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.20 scripts/showcase-onboarding.sh "$(command -v testloop-mcp)"
```

## 这条路径验证什么

脚本会依次执行三步：

1. `scripts/verify-client-setup.sh`：基础安装验收，确认二进制可执行、版本可读、客户端配置片段可 roundtrip 校验，并检查 HTTP `/healthz`。
2. `scripts/verify-mcp-process-smoke.sh`：深度协议验收，使用真实 MCP SDK 客户端通过 stdio 和 Streamable HTTP 启动二进制，调用 `tools/list` 和轻量 `parse_results`，并校验 `structuredContent` 与文本 JSON fallback 一致。
3. `go run ./examples/mcp-client-demo`：最小 Agent 闭环，演示 `run_tests.action -> fix_suggestions.category -> repair_task -> rerun.action -> parse_coverage`。

## 适用边界

这条路径适合 README、录屏和首次接入验收。它不依赖外部项目，也不会修改当前仓库业务文件。

如果只是想确认安装和配置，运行 `scripts/verify-client-setup.sh` 即可。如果要证明真实 MCP 协议链路可用，再运行 `scripts/verify-mcp-process-smoke.sh` 或本脚本。

如果面向接入方、README 录屏或 CI artifact，优先运行 `scripts/showcase-agent-onboarding-report.sh`。它会复用验收报告脚本和 summary 决策 demo，在失败时也尽量保留 Markdown / JSON / decision 输出，方便后续定位是安装、协议、Agent demo、公开 showcase 还是用户项目 smoke 的问题。
