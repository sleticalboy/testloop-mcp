# CI 失败后交给 Agent

这份文档只解决一个问题：GitHub Actions 里 testloop first-run 失败后，怎么把最小上下文交给 AI Agent，而不是让 Agent 猜一整段 CI 红色日志。

## 1. 下载 artifact

在失败 run 页面可以手动下载 artifact。使用 GitHub CLI 时：

```bash
gh run list --workflow "testloop first-run smoke" --limit 5
gh run download <run-id> -n testloop-first-run -D /tmp/testloop-first-run-artifacts
```

如果 workflow 或 artifact 名称改过，把命令里的 `"testloop first-run smoke"` 和 `testloop-first-run` 换成你的实际名称。

## 2. 快速路径：先读 Agent 回复草稿

新版 `run-first-run-ci.sh` 会在 artifact 中生成 `agent-response.txt`。如果这个文件存在，先看它：

```bash
cat /tmp/testloop-first-run-artifacts/agent-response.txt
```

它已经按“结论 / 证据 / 下一步 / 暂不做”整理过，可直接作为 Agent 回复草稿。需要机器分流、旧版 artifact 没有这个文件，或者要确认 action 来源时，再继续读 `agent-decision.txt` 和 `first-run-context.txt`。

## 3. 再读 decision

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

## 4. 粘贴最小上下文

first-run 失败时，如果没有 `agent-response.txt`，或 Agent 需要自己重新判断，优先把 `first-run-context.txt` 原样交给 Agent：

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

如果 artifact 来自旧版脚本没有 `agent-response.txt`，可以在 testloop-mcp 仓库里运行：

```bash
sh scripts/render-first-run-agent-response.sh /tmp/testloop-first-run-artifacts
```

## 5. 不要先贴什么

- 不要只贴 GitHub Actions 最后一行错误。
- 不要先贴完整 `first-run.log`，除非 Agent 已经要求看底层日志。
- 不要在 `agent_next_step=fix-installation` 时让 Agent 先改业务测试。
- 不要在 `agent_next_step=inspect-user-project` 时继续排查 MCP transport，先看用户项目 smoke。

## 6. onboarding artifact

稳定接入后的 onboarding CI 没有 `first-run-context.txt` 和 `first-run.log`。这时按顺序粘：

1. `agent-decision.txt`
2. `verification-summary.json` 中的 `overall_status`、`failed_count` 和失败 section
3. `verification-report.md` 中失败 section 的 stdout / stderr

更完整的分流说明见 [Onboarding CI 失败排查](./onboarding-ci-failure-triage.md) 和 [首跑诊断失败样例](./first-run-failures.md)。

Agent 收到 `first-run-context.txt` 后的推荐回复格式见 [first-run Agent 回复格式](./first-run-agent-response.md)。

## 7. 失败态实跑记录

2026-07-19 使用外部临时项目和故意失败的 smoke 命令复验 first-run triage：

```bash
rm -rf /tmp/testloop-triage-failing-project /tmp/testloop-first-run-failure-triage
mkdir -p /tmp/testloop-triage-failing-project
printf 'intentional failure fixture\n' > /tmp/testloop-triage-failing-project/README.md

TESTLOOP_MCP_VERSION=v0.5.7 \
TESTLOOP_FIRST_RUN_OUTPUT_DIR=/tmp/testloop-first-run-failure-triage \
TESTLOOP_FIRST_RUN_PROJECT_DIR=/tmp/testloop-triage-failing-project \
  scripts/run-first-run-ci.sh 'echo testloop intentional project failure; exit 7'
```

结果符合预期：

```text
first_run_status=failed
first_run_failed_count=1
first_run_agent_next_step=inspect-user-project
first_run_context=/tmp/testloop-first-run-failure-triage/first-run-context.txt
agent_response=/tmp/testloop-first-run-failure-triage/agent-response.txt
```

`verification-summary.json` 中只有“用户项目 smoke”失败：

```text
overall_status=failed
failed_count=1
failed_section=用户项目 smoke
exit_code=7
```

`verification-report.md` 的失败 section 保留了项目输出：

```text
testloop intentional project failure
```

这说明 `agent-response.txt` 足够作为第一段 Agent 回复草稿；`agent-decision.txt` 和 `first-run-context.txt` 仍足够让 Agent 重新选择 `inspect-user-project`。只有需要查看项目 stdout / stderr 时，才需要继续打开 Markdown report。
