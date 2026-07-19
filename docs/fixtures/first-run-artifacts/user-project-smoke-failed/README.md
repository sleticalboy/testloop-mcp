# user-project-smoke-failed

这是一份 first-run 失败 artifact 五件套 fixture，用于测试 AI Agent 或客户端如何消费 `run-first-run-ci.sh` 的失败输出。

场景：testloop-mcp 安装、MCP transport 和最小 Agent demo 均已跳过或通过，用户项目 smoke 命令失败，`agent_next_step` 应分流到 `inspect-user-project`。

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
