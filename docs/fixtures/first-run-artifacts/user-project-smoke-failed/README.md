# user-project-smoke-failed

这是一份 first-run 失败 artifact 六件套 fixture，用于测试 AI Agent 或客户端如何消费 `run-first-run-ci.sh` 的失败输出。

场景：testloop-mcp 安装、MCP transport、最小 Agent demo 和独立 CLI 生成动作 smoke 均通过，用户项目 smoke 命令失败，`agent_next_step` 应分流到 `inspect-user-project`。独立 CLI 生成动作 smoke 会在 summary 中保留 `signals.action=manual_review`，用于确认 skipped/TODO 测试草稿不会被误读为有效覆盖。

文件：

- `verification-report.md`
- `verification-summary.json`
- `agent-decision.txt`
- `first-run-context.txt`
- `first-run.log`

可用 demo 消费：

```bash
sh scripts/render-first-run-agent-response.sh \
  docs/fixtures/first-run-artifacts/user-project-smoke-failed/
```

也可以手动指定文件：

```bash
go run ./examples/first-run-agent-response-demo \
  docs/fixtures/first-run-artifacts/user-project-smoke-failed/first-run-context.txt \
  docs/fixtures/first-run-artifacts/user-project-smoke-failed/verification-summary.json
```

fixture 已包含同样输出的 `agent-response.txt`，便于客户端不运行 demo 也能回归回复草稿消费逻辑。
