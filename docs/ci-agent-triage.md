# CI 失败后交给 Agent

这份文档只解决一个问题：GitHub Actions 里 testloop first-run 失败后，怎么把最小上下文交给 AI Agent，而不是让 Agent 猜一整段 CI 红色日志。

## 1. 下载 artifact

在失败 run 页面可以手动下载 artifact。使用 GitHub CLI 时：

```bash
gh run list --workflow "testloop first-run smoke" --limit 5
gh run download <run-id> -n testloop-first-run -D /tmp/testloop-first-run-artifacts
```

如果 workflow 或 artifact 名称改过，把命令里的 `"testloop first-run smoke"` 和 `testloop-first-run` 换成你的实际名称。

## 2. 先读 decision

```bash
cat /tmp/testloop-first-run-artifacts/agent-decision.txt
```

常见结果：

| `agent_next_step` | 交给 Agent 的任务 |
| --- | --- |
| `ready` | 不需要排查 testloop 接入，继续生成测试、补覆盖率或修业务失败。 |
| `fix-installation` | 先修二进制路径、版本漂移、配置 roundtrip 或 HTTP `/healthz`。 |
| `inspect-mcp-transport` | 先查 stdio / Streamable HTTP MCP 启动、端口和客户端传输配置。 |
| `inspect-agent-demo` | 先查最小 Agent demo、结构化返回或本仓库 Go 运行环境。 |
| `inspect-user-project` | 先查用户项目 smoke 命令、依赖、环境变量和测试/构建输出。 |

## 3. 粘贴最小上下文

first-run 失败时优先把 `first-run-context.txt` 原样交给 Agent：

```bash
cat /tmp/testloop-first-run-artifacts/first-run-context.txt
```

推荐粘贴格式：

```text
这是 testloop-mcp first-run CI 的失败上下文。请先读取 agent_next_step，再决定下一步：

<粘贴 first-run-context.txt 全文>
```

如果 Agent 需要更细日志，再补充：

```bash
cat /tmp/testloop-first-run-artifacts/verification-summary.json
cat /tmp/testloop-first-run-artifacts/verification-report.md
cat /tmp/testloop-first-run-artifacts/first-run.log
```

## 4. 不要先贴什么

- 不要只贴 GitHub Actions 最后一行错误。
- 不要先贴完整 `first-run.log`，除非 Agent 已经要求看底层日志。
- 不要在 `agent_next_step=fix-installation` 时让 Agent 先改业务测试。
- 不要在 `agent_next_step=inspect-user-project` 时继续排查 MCP transport，先看用户项目 smoke。

## 5. onboarding artifact

稳定接入后的 onboarding CI 没有 `first-run-context.txt` 和 `first-run.log`。这时按顺序粘：

1. `agent-decision.txt`
2. `verification-summary.json` 中的 `overall_status`、`failed_count` 和失败 section
3. `verification-report.md` 中失败 section 的 stdout / stderr

更完整的分流说明见 [Onboarding CI 失败排查](./onboarding-ci-failure-triage.md) 和 [首跑诊断失败样例](./first-run-failures.md)。
