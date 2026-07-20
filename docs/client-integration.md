# 客户端集成说明

这份文档面向 MCP 客户端、编辑器插件和 AI Coding Agent 的接入方。目标不是重新解释所有字段，而是给出一条可回归的消费流程：优先读取 `structuredContent`，按 `status/action` 分流，再用真实 fixture 固定自己的客户端行为。

## 消费顺序

1. 调用 MCP tool 后，优先读取 `structuredContent`。
2. 如果客户端 SDK 暂不暴露 `structuredContent`，再 fallback 到 `content[0].text` 并按 JSON 解析。
3. 对 `validate_coverage_task`，不要只看 `status`；必须用 `status/action` 组合做下一步决策。
4. 对失败结果，优先读取 `run_result.fix_suggestions[].repair_task`，不要把 `suggested_fix` 当补丁直接套用。
5. 客户端必须忽略未知字段；新增字段不应导致旧客户端失败。

## 最小决策回归

仓库提供一个最小示例：

```bash
go run ./examples/agent-decision-demo
```

该示例读取 [真实结构化 fixture](./fixtures.md) 中的 JSON，演示如何把：

- `passed/ready` 映射为 `accept`
- `passed/manual_review_internal` 映射为 `manual-review`
- `failed/apply_fix_suggestions` 映射为 `apply-repair`
- `failed/needs_better_input` 映射为 `needs-better-input`

如果你在做自己的客户端，建议把同样的映射逻辑做成单元测试，而不是只在真实项目里手动观察。

## 使用真实 fixture

[真实结构化 fixture](./fixtures.md) 提供了来自 handler 的稳定 JSON 投影，适合直接放进客户端测试用例：

| fixture | 期望客户端动作 |
| --- | --- |
| [run-tests/apply-fix-suggestions.json](./fixtures/run-tests/apply-fix-suggestions.json) | 对普通 `run_tests` 失败结果读取 `action`、`fix_suggestions[0].category` 和 `repair_task`。 |
| [validate-coverage-task-ready.json](./fixtures/validate-coverage-task-ready.json) | 接受生成测试，进入下一个 coverage task。 |
| [validate-coverage-task-manual-review-internal.json](./fixtures/validate-coverage-task-manual-review-internal.json) | 记录手审原因，不继续自动修同一个生成测试。 |
| [validate-coverage-task-apply-fix-suggestions.json](./fixtures/validate-coverage-task-apply-fix-suggestions.json) | 读取 `repair_task`，按限定文件和命令执行修复闭环。 |
| [validate-coverage-task-needs-better-input.json](./fixtures/validate-coverage-task-needs-better-input.json) | 读取覆盖率未命中原因，重新选择输入或公共入口。 |

建议客户端测试至少断言：

- 能从 fixture 解析 `status` 和 `action`。
- `run_tests` 的 `fail/apply_fix_suggestions` 能定位 `fix_suggestions[0].category` 和 `fix_suggestions[0].repair_task`。
- `passed/ready` 不读取 `run_result.fix_suggestions`。
- `manual_review_*` 不触发自动修复循环。
- `failed/apply_fix_suggestions` 能定位 `run_result.fix_suggestions[0].repair_task.target_file`、`editable_files` 和 `suggested_commands`。
- `failed/needs_better_input` 能定位 `metadata.coverage_miss_reason` 和 `metadata.coverage_missed_lines`。
- 遇到未知字段不会报错。

## CI artifact fixture

除了 `validate_coverage_task` 的 MCP 结构化返回，接入方还需要测试“CI 失败后把 artifact 交给 Agent”的路径。这类输入不是 MCP tool 返回值，而是 `run-first-run-ci.sh` 或 `run-onboarding-ci.sh` 产出的文件包。

仓库提供两类完整 fixture：

```text
docs/fixtures/first-run-artifacts/user-project-smoke-failed/
docs/fixtures/onboarding-artifacts/user-project-smoke-failed/
```

first-run fixture 包含：

- `verification-report.md`
- `verification-summary.json`
- `agent-decision.txt`
- `first-run-context.txt`
- `agent-response.txt`
- `first-run.log`

onboarding fixture 包含：

- `verification-report.md`
- `verification-summary.json`
- `agent-decision.txt`
- `agent-response.txt`

推荐客户端或 Agent 测试断言：

- 先读取 `agent-decision.txt`，识别 `agent_next_step=inspect-user-project`。
- first-run artifact 再读取 `first-run-context.txt`，识别 `first_run_agent_next_step=inspect-user-project`。
- 需要失败细节时读取 `verification-summary.json`，定位 `failed_section=用户项目 smoke` 和 `exit_code=7`。
- 如果存在 `agent-response.txt`，可以直接把它作为 Agent 回复草稿；不存在时用目录入口补渲染。
- 最后打开 `verification-report.md` 的失败 section，而不是只消费 CI 最后一行错误。

可用内置 demo 验证 Agent 回复：

```bash
sh scripts/render-first-run-agent-response.sh \
  docs/fixtures/first-run-artifacts/user-project-smoke-failed/
```

onboarding artifact 用对应目录入口：

```bash
sh scripts/render-onboarding-agent-response.sh \
  docs/fixtures/onboarding-artifacts/user-project-smoke-failed/
```

也可以手动指定文件：

```bash
go run ./examples/first-run-agent-response-demo \
  docs/fixtures/first-run-artifacts/user-project-smoke-failed/first-run-context.txt \
  docs/fixtures/first-run-artifacts/user-project-smoke-failed/verification-summary.json
```

这两条路径都输出固定四段：结论、证据、下一步、暂不做。完整说明见 [first-run artifact Agent 消费演示](./first-run-agent-artifact-demo.md)。

first-run 和 onboarding 的 `agent-response.txt` 统一字段、读取顺序和客户端断言见 [Agent response artifact contract](./agent-response-artifact-contract.md)。

如果希望测试自动发现 artifact fixture，可以读取 [agent-response-artifact-manifest.json](./fixtures/agent-response-artifact-manifest.json)，其中固定了 first-run / onboarding 的目录、必备文件、期望 action、`expected_section_signals` 和 fallback 顺序。manifest 的 JSON Schema 见 [agent-response-artifact-manifest.schema.json](./fixtures/agent-response-artifact-manifest.schema.json)，适合客户端生成类型或做契约校验。summary JSON 的 canonical 结构契约见 [verification-summary.schema.json](./fixtures/verification-summary.schema.json)；每个 artifact 也有本地 `summary_schema=verification-summary.schema.json` 指针，下载 fixture 或 CI artifact 后可以离线校验同目录的 `verification-summary.json`。其中 `sections[].signals.action` 可用于读取 section 级动作信号。`verification-summary-decision-demo` 会先校验 `overall_status`、`failed_count` 和 `sections` 这些必填字段，再输出 `agent_next_step`，避免把非 verification summary 的 JSON 误判成 ready。双项目报告的 combined summary 使用 [dual-project-summary.schema.json](./fixtures/dual-project-summary.schema.json)，样例见 [laoxia-passed.json](./fixtures/dual-project-summary/laoxia-passed.json)，客户端应把它和 `verification-summary.json` 分开建模。当前 fixture 会要求客户端保留 `独立 CLI 生成动作 smoke:manual_review`，但该 signal 不等于整体验收失败。

仓库提供了一个最小 manifest 消费 demo：

```bash
go run ./examples/agent-response-manifest-demo \
  docs/fixtures/agent-response-artifact-manifest.json
```

该 demo 会同时验证 `agent-response.txt`、`agent-decision.txt`、`verification-summary.json`、summary schema 和 manifest 中的 `expected_section_signals`，并输出 `decision_action=...`、`summary_validated=verification-summary.json` 与 `local_summary_schema=verification-summary.schema.json`，方便客户端确认机器分流、回复草稿和 summary 契约一致。

## 推荐客户端伪代码

```text
payload = result.structuredContent
if payload is empty:
  payload = parse_json(result.content[0].text)

switch payload.status + "/" + payload.action:
  case "passed/ready":
    accept_generated_test(payload.generated.test_file)
  case starts_with(action, "manual_review_"):
    record_manual_review(payload.metadata)
  case "failed/apply_fix_suggestions":
    apply_repair_task(payload.run_result.fix_suggestions[0].repair_task)
  case "failed/needs_better_input":
    choose_better_input(payload.metadata)
  case "generation_error/*":
    inspect_provider_error(payload.provider_error, payload.error)
  case "run_error/*":
    inspect_test_runner(payload.error)
  default:
    show_structured_result_for_review(payload)
```

## 相关文档

- [Agent 结构化契约](./agent-contract.md)
- [Agent Action 决策表](./agent-action-guide.md)
- [validate_coverage_task 结构化返回样例](./validate-coverage-task-samples.md)
- [真实结构化 fixture](./fixtures.md)
- [Agent response artifact contract](./agent-response-artifact-contract.md)
- [agent-response-artifact-manifest.json](./fixtures/agent-response-artifact-manifest.json)
- [agent-response-artifact-manifest.schema.json](./fixtures/agent-response-artifact-manifest.schema.json)
- [first-run artifact Agent 消费演示](./first-run-agent-artifact-demo.md)
- [MCP 客户端契约测试说明](./mcp-client-contract-tests.md)
