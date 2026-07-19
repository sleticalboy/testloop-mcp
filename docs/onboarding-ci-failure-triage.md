# Onboarding CI 失败排查

这份文档说明 `scripts/run-onboarding-ci.sh` 在 CI 失败时应该先看什么。目标是让接入方把失败信息交给 AI Agent 时，不再只贴一段红色日志，而是提供稳定的结构化上下文。

## 先看 Step Summary

在 GitHub Actions 中，`run-onboarding-ci.sh` 会自动向 `$GITHUB_STEP_SUMMARY` 写入：

- `Status`
- `Failed sections`
- `agent_next_step`
- `Markdown report`
- `Summary JSON`
- `Agent decision`
- `Agent response`

如果 `agent_next_step=ready`，说明 onboarding 链路和用户项目 smoke 都通过。否则按 `agent_next_step` 分流。

## 下载 Artifact

模板默认上传 artifact：

- `verification-report.md`
- `verification-summary.json`
- `agent-decision.txt`
- `agent-response.txt`

排查顺序：

1. 先读 `agent-response.txt`，查看脚本已经渲染出的 Agent 四段回复草稿。
2. 再读 `agent-decision.txt`，确认 `agent_next_step`。
3. 再读 `verification-summary.json`，确认 `overall_status`、`failed_count` 和失败 section。
4. 最后读 `verification-report.md`，查看失败 section 的 stdout / stderr。

## 分流表

| `agent_next_step` | 先看哪里 | 常见原因 |
| --- | --- | --- |
| `fix-installation` | `verification-report.md` 的“基础安装验收” | 二进制路径错、版本漂移、配置 roundtrip 失败、HTTP `/healthz` 不通。 |
| `inspect-mcp-transport` | “真实 MCP 协议 smoke” | stdio / Streamable HTTP 启动失败、端口冲突、客户端传输配置不一致。 |
| `inspect-agent-demo` | “最小 Agent 闭环 demo” | 本仓库 demo runner、结构化返回或 Go 运行环境异常。 |
| `inspect-user-project` | “用户项目 smoke” | 项目测试失败、依赖未安装、环境变量缺失、构建命令不对。 |
| `inspect-showcase` | “公开 showcase” | GitHub/npm 网络、外部项目 checkout、默认 action 期望漂移。 |

## 粘给 AI Agent 的最小上下文

建议把下面三段一起给 AI Agent：

```text
agent-response.txt:
<粘贴完整内容；新版 artifact 优先使用这份 Agent 四段回复草稿>

agent-decision.txt:
<粘贴完整内容>

verification-summary.json:
<粘贴 overall_status、failed_count、失败 section>

verification-report.md:
<只粘贴失败 section 的明细>
```

如果失败是 `inspect-user-project`，还要补充项目 smoke 命令，例如：

```text
TESTLOOP_ONBOARDING_PROJECT_DIR=/path/to/project
project-smoke-command='go test ./...'
```

## 不要先做什么

- 不要先重跑全部 CI，先看 artifact。
- 不要只贴 GitHub Actions 最后一行错误。
- 不要把安装问题误判成测试生成质量问题。
- 不要把用户项目 smoke 失败归因到 MCP 传输，除非 summary JSON 指向 `inspect-mcp-transport`。

复制模板见 [Onboarding CI 复制模板](./onboarding-ci-template.md)，完整 CI 集成说明见 [验收报告 CI 集成](./verification-ci.md)。
