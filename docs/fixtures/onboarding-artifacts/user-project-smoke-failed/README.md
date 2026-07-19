# onboarding 用户项目 smoke 失败 artifact fixture

这是一份稳定的 onboarding CI 失败 artifact 包，用于客户端、编辑器插件或 Agent 测试失败消费逻辑。场景固定为：

- testloop-mcp 安装、MCP transport、最小 Agent 闭环和独立 CLI 生成动作 smoke 通过。
- 用户项目 smoke 命令失败，exit code 为 `7`。
- `agent_next_step=inspect-user-project`。
- summary 中的 `signals.action=manual_review` 来自独立 CLI 生成动作 smoke，用于确认 skipped/TODO 测试草稿不会被误读为有效覆盖。

包含文件：

- `verification-report.md`
- `verification-summary.json`
- `agent-decision.txt`
- `agent-response.txt`

可直接运行目录入口验证回复草稿：

```bash
sh scripts/render-onboarding-agent-response.sh \
  docs/fixtures/onboarding-artifacts/user-project-smoke-failed/
```

fixture 已包含同样输出的 `agent-response.txt`，便于客户端不运行 demo 也能回归回复草稿消费逻辑。
