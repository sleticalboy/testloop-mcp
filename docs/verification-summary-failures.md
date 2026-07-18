# 验收 Summary 失败分流样例

这组样例面向 AI Agent 和 CI。目标是说明：验收报告失败时，不应该只把它当成“测试失败”，而应该先看 summary JSON 里的失败 section，再决定下一步动作。

运行方式：

```bash
go run ./examples/verification-summary-decision-demo docs/fixtures/verification-summary/user-project-failed.json
```

输出里的 `agent_next_step` 就是 Agent / CI 应优先执行的动作。

## 决策映射

| fixture | 失败 section | agent_next_step | 下一步 |
| --- | --- | --- | --- |
| [`install-failed.json`](./fixtures/verification-summary/install-failed.json) | `基础安装验收` | `fix-installation` | 先检查二进制路径、版本门禁、配置 roundtrip 和 HTTP `/healthz`。 |
| [`mcp-transport-failed.json`](./fixtures/verification-summary/mcp-transport-failed.json) | `真实 MCP 协议 smoke` | `inspect-mcp-transport` | 先排查 stdio / Streamable HTTP 客户端启动、端口占用和协议返回。 |
| [`agent-demo-failed.json`](./fixtures/verification-summary/agent-demo-failed.json) | `最小 Agent 闭环 demo` | `inspect-agent-demo` | 先看最小 demo 的结构化返回、`repair_task` 和复跑流程。 |
| [`showcase-failed.json`](./fixtures/verification-summary/showcase-failed.json) | `公开 Go showcase` | `inspect-showcase` | 先区分外部网络、公开仓库 checkout、依赖安装和 action 期望漂移。 |
| [`user-project-failed.json`](./fixtures/verification-summary/user-project-failed.json) | `用户项目 smoke` | `inspect-user-project` | 先检查用户项目命令、依赖、环境变量和测试输出。 |

## 使用边界

- 这些 fixture 是 summary JSON 的最小可消费样例，不是完整 Markdown 报告。
- 失败 section 的 `exit_code` 保留原始命令退出码；skipped section 的 `exit_code` 使用 `null`。
- 多个 section 同时失败时，decision demo 会逐项打印失败原因，并把第一个失败 section 作为 `agent_next_step`。
- 自动化侧应优先读取 summary JSON；需要 stdout / stderr 明细时，再打开 `markdown_report` 指向的 Markdown 报告。

这组样例和 [用户项目验收报告](./verification-report.md) 配套使用：Markdown 负责给人看，summary JSON 负责给 Agent / CI 做分流。
