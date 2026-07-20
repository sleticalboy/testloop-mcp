# Agent response artifact contract

这份文档面向 MCP 客户端、编辑器插件和 AI Coding Agent。`agent-response.txt` 是 CI artifact 里的确定性回复草稿，用来把用户从“整段红色日志”带到可执行的下一步。

## 适用入口

| CI 入口 | artifact 套件 | 生成脚本 | 主要用途 |
| --- | --- | --- | --- |
| `scripts/run-first-run-ci.sh` | first-run 七件套 | `scripts/render-first-run-agent-response.sh` | 首次安装、版本漂移、MCP transport、Agent demo 和用户项目 smoke 诊断。 |
| `scripts/run-onboarding-ci.sh` | onboarding 五件套 | `scripts/render-onboarding-agent-response.sh` | 稳定接入后的 PR / 发布后 smoke。 |

## 固定结构

`agent-response.txt` 固定为四段，顺序不能漂移：

1. `结论：...`
2. `证据：`
3. `下一步：`
4. `暂不做：`

客户端可以把它作为自然语言草稿展示给 Agent 或用户；需要机器分流时，仍以 `agent-decision.txt` 和 `verification-summary.json` 为准。

## 固定证据字段

first-run response 至少包含：

- `first_run_agent_next_step=<action>`
- `first_run_status=<passed|failed>`
- `first_run_failed_count=<number>`
- `first_run_report=<path>`

onboarding response 至少包含：

- `agent_next_step=<action>`
- `overall_status=<passed|failed>`
- `failed_count=<number>`
- `markdown_report=<path>`

失败场景通常还包含：

- `failed_section=<section name>`
- `exit_code=<code>`

通过或失败场景都可能包含：

- `section_signal=<section name> action=<action>`

`section_signal` 来自 `verification-summary.json` 的 `sections[].signals`。例如独立 CLI 生成动作 smoke 会输出 `action=manual_review`，表示测试生成草稿需要补真实输入和断言；这不是整体验收失败信号，整体结果仍以 `overall_status` / `failed_count` 为准。

## 读取顺序

CI 失败时按这个顺序读取：

1. 先读 `agent-response.txt`，直接获得结论和下一步。
2. 需要机器分流或复核 action 来源时，读 `agent-decision.txt`。
3. 需要定位失败 section 时，读 `verification-summary.json`。
4. 需要 stdout / stderr 细节时，读 `verification-report.md`。
5. first-run 旧版 artifact 没有 `agent-response.txt` 时，再读 `first-run-context.txt` 或用目录入口补渲染。

## 客户端断言

客户端或 Agent 回归测试建议固定这些行为：

- 看到 `agent-response.txt` 时，不要求用户先粘完整 CI 日志。
- `agent-response.txt` 缺失时，first-run 可以 fallback 到 `first-run-context.txt`，onboarding 可以 fallback 到 summary + decision。
- `agent_next_step=ready` 或 `first_run_agent_next_step=ready` 不触发排障流程。
- `section_signal=... action=manual_review` 只作为 section 级证据记录，不把整体通过报告误判为失败。
- `inspect-user-project` 优先检查用户项目 smoke，不先修改 testloop-mcp 安装或 MCP transport。
- 未知 action 进入人工复核，不自动生成或修改测试。

可复用 fixture：

- [first-run 用户项目 smoke 失败](./fixtures/first-run-artifacts/user-project-smoke-failed/)
- [onboarding 用户项目 smoke 失败](./fixtures/onboarding-artifacts/user-project-smoke-failed/)

机器可读索引：

- [agent-response-artifact-manifest.json](./fixtures/agent-response-artifact-manifest.json)
- [agent-response-artifact-manifest.schema.json](./fixtures/agent-response-artifact-manifest.schema.json)
- [verification-summary.schema.json](./fixtures/verification-summary.schema.json)

最小消费 demo：

```bash
go run ./examples/agent-response-manifest-demo \
  docs/fixtures/agent-response-artifact-manifest.json
```

## 相关文档

- [CI 失败后交给 Agent](./ci-agent-triage.md)
- [客户端集成说明](./client-integration.md)
- [first-run artifact Agent 消费演示](./first-run-agent-artifact-demo.md)
- [Onboarding CI 失败排查](./onboarding-ci-failure-triage.md)
