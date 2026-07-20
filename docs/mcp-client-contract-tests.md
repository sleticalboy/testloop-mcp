# MCP 客户端契约测试说明

这份文档面向 MCP 客户端、编辑器插件和 AI Coding Agent 的接入方。目标是把 testloop-mcp 当前的客户端消费约束压成一组可复制的 CI 检查，避免客户端只在人工试跑时才发现 `structuredContent`、`status/action` 或 fixture 映射漂移。

## 最小契约

客户端至少应固定这些行为：

1. 优先读取 MCP 返回的 `structuredContent`。
2. 如果 SDK 暂不暴露 `structuredContent`，再 fallback 到 `content[0].text` JSON。
3. 对 `validate_coverage_task`，必须按 `status/action` 组合分流，而不是只看 `status`。
4. 对 `failed/apply_fix_suggestions`，优先读取 `run_result.fix_suggestions[].repair_task`。
5. 对 `manual_review_*`，不要继续自动修同一个生成测试。
6. 对 `failed/needs_better_input`，读取 `metadata.coverage_miss_reason` 和 `metadata.coverage_missed_lines`，重新选择输入或公共入口。
7. 客户端必须忽略未知字段。

## 可复制的 fixture 回归

接入方可以把 [真实结构化 fixture](./fixtures.md) 复制到自己的客户端测试资源里，或直接在集成测试中从 testloop-mcp 仓库读取：

```bash
docs/fixtures/validate-coverage-task-ready.json
docs/fixtures/validate-coverage-task-manual-review-internal.json
docs/fixtures/validate-coverage-task-apply-fix-suggestions.json
docs/fixtures/validate-coverage-task-needs-better-input.json
docs/fixtures/real-project-agent-loop/laoxia-server-go-utils.json
docs/fixtures/real-project-agent-loop/mcp-hub-vitest-repair.json
docs/fixtures/real-project-agent-loop/haoy-apk-station-py-environment.json
docs/fixtures/real-project-agent-loop/haoy-apk-station-py-external-service.json
docs/fixtures/agent-decision-fixtures.json
docs/fixtures/agent-decision-fixtures.schema.json
```

这些 fixture 来自 handler 真实输出或真实项目验证摘要，不是手写示意样例。推荐客户端单元测试直接断言：

| status/action | 期望客户端动作 |
| --- | --- |
| `passed/ready` | `accept` |
| `passed/manual_review_internal` | `manual-review` |
| `passed/manual_review_environment` | `manual-review` |
| `failed/manual_review_external_service` | `manual-review` |
| `failed/apply_fix_suggestions` | `apply-repair` |
| `failed/needs_better_input` | `needs-better-input` |

## 仓库内参考校验

testloop-mcp 自身用这些脚本保护客户端契约：

```bash
sh test/fixtures_index_test.sh
sh test/agent_decision_fixtures_manifest_test.sh
sh test/agent_decision_fixture_validator_test.sh
sh test/fixture_decision_mapping_test.sh
sh test/client_integration_doc_test.sh
sh test/agent_decision_demo_test.sh
```

各脚本职责：

- `fixtures_index_test.sh`：确认每个 `docs/fixtures/*.json` 都登记到 fixture 索引，且 `status/action` 集合没有静默扩张。
- `agent_decision_fixtures_manifest_test.sh`：确认 `agent-decision-fixtures.json` 登记了全部最小决策 fixture，且 manifest 中的 `status/action/expected_decision` 与真实 JSON 一致。
- `agent_decision_fixture_validator_test.sh`：运行可复制 Node validator，确认外部客户端模板能按 manifest 校验全部 fixture。
- `fixture_decision_mapping_test.sh`：直接扫描真实 fixture，校验每个 `status/action` 映射到预期客户端动作。
- `client_integration_doc_test.sh`：确认客户端集成说明引用的 fixture 和 demo 入口仍然存在。
- `agent_decision_demo_test.sh`：确认 `go run ./examples/agent-decision-demo` 对真实 fixture 输出稳定决策。
- 真实项目 fixture 也应通过同一套 `status/action` 映射，不应因为包含 `task`、`regression_note` 或 `redaction_note` 等额外字段而失败。

## 接入方 CI 模板

如果客户端项目使用 JavaScript/TypeScript，可以把 `agent-decision-fixtures.json` 和其中列出的 fixture 放到 `test/fixtures/`，并在 CI 中加入类似检查：

```bash
node test/validate-testloop-fixtures.mjs
```

仓库内提供了一个无第三方依赖的可复制参考实现：

```bash
node scripts/validate-agent-decision-fixtures.mjs \
  docs/fixtures/agent-decision-fixtures.json \
  .
```

CI 里建议使用 JSON 输出，避免解析展示用文本：

```bash
node scripts/validate-agent-decision-fixtures.mjs --json \
  docs/fixtures/agent-decision-fixtures.json \
  .
```

JSON 输出会固定 `status`、`fixture_count`、`decisions[]`、`fixtures[]` 和 `failures[]`。验证失败时脚本仍输出可解析 JSON，并用 `status=failed` 与非 0 退出码同时表达失败。

如果不想复制整个仓库，可以先导出最小决策 fixture 包：

```bash
node scripts/export-agent-decision-fixtures.mjs /tmp/testloop-agent-decision-fixtures
```

导出包保留 `docs/fixtures/...` 路径和 validator 脚本，适合直接放进客户端仓库的契约测试目录。
导出包还包含无依赖 `package.json`，客户端 CI 可以直接执行 `npm test --silent`。

脚本最小逻辑应由 manifest 驱动，而不是硬编码 glob：

```text
manifest = parse_json("agent-decision-fixtures.json")
for each item in manifest.fixtures:
  payload = parse_json(item.path)
  assert payload.status == item.status
  assert payload.action == item.action
  decision = map(payload.status, payload.action)
  assert decision == item.expected_decision
  assert unknown fields do not fail parsing
  if action == apply_fix_suggestions:
    assert payload.run_result.fix_suggestions[0].repair_task exists
  if action starts with manual_review_:
    assert decision == manual-review
    assert client does not run automatic repair for the same generated test
  if action == needs_better_input:
    assert payload.metadata.coverage_miss_reason exists
```

如果客户端直接调用 MCP server，还应增加一条进程级 smoke：

```text
start testloop-mcp via stdio or Streamable HTTP
call tools/list
call one lightweight tool, such as parse_results
assert structuredContent is present
assert content[0].text can be parsed as equivalent JSON fallback
```

仓库内对应参考是 `test/e2e`，它覆盖 stdio 和 Streamable HTTP 两条真实 MCP 接入路径。

## CI artifact manifest 回归

客户端如果消费 GitHub Actions artifact，还应把 `agent-response.txt` 这条路径纳入契约测试。这里测试的不是 MCP 返回值，而是 CI 失败后下载到本地的 artifact 目录。

推荐最小流程：

```bash
curl -fsSLO https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/docs/fixtures/agent-response-artifact-manifest.json
curl -fsSLO https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/docs/fixtures/agent-response-artifact-manifest.schema.json
curl -fsSLO https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/docs/fixtures/verification-summary.schema.json
curl -fsSLO https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/docs/fixtures/dual-project-summary.schema.json
npx --yes ajv-cli validate \
  -s agent-response-artifact-manifest.schema.json \
  -d agent-response-artifact-manifest.json
npx --yes ajv-cli validate \
  -s verification-summary.schema.json \
  -d docs/fixtures/verification-summary/user-project-failed.json
npx --yes ajv-cli validate \
  -s dual-project-summary.schema.json \
  -d docs/fixtures/dual-project-summary/laoxia-passed.json
```

如果客户端 CI 不想引入 JSON Schema 校验器，至少应运行仓库内 demo，确认 manifest 指向的 artifact fixture、必备字段和 fallback 顺序仍然可消费：

```bash
go run ./examples/agent-response-manifest-demo \
  docs/fixtures/agent-response-artifact-manifest.json
```

正常输出应包含 `decision_action=inspect-user-project` 和 `summary_validated=verification-summary.json`，用于确认 `agent-decision.txt` 的机器分流结果、summary schema 校验和 manifest 中的 `expected_action` 一致。

如果客户端已经拿到了单个 artifact 目录，推荐再跑目录级 verifier：

```bash
sh scripts/verify-agent-artifact.sh \
  first-run \
  docs/fixtures/first-run-artifacts/user-project-smoke-failed/

sh scripts/verify-agent-artifact.sh \
  onboarding \
  docs/fixtures/onboarding-artifacts/user-project-smoke-failed/
```

正常输出应包含 `agent_artifact_status=passed`、`decision_action=inspect-user-project` 和 `response_action=inspect-user-project`，用于确认必备文件、同目录 summary schema、decision、Agent 回复草稿和 section signal 没有漂移。

客户端测试也可以直接用 manifest 批量模式，避免手写两条 fixture 路径：

```bash
sh scripts/verify-agent-artifact.sh \
  manifest \
  docs/fixtures/agent-response-artifact-manifest.json

sh scripts/verify-agent-artifact.sh \
  --json \
  manifest \
  docs/fixtures/agent-response-artifact-manifest.json
```

正常文本输出应包含 `agent_artifact_manifest_status=passed` 和 `artifact_count=2`。客户端 CI 更推荐断言 JSON 输出里的 `status`、`artifact_count`、`artifacts[].artifact_kind`、`artifacts[].response_action` 和 `artifacts[].section_signals`。

客户端自己的测试建议额外断言：

- `schema_version=1`。
- 每个 artifact 都先读取 `agent-response.txt`。
- `fallback_order[0]` 固定为 `agent-response.txt`。
- `agent-decision.txt` 中的 `agent_next_step` 等于 manifest 的 `expected_action`。
- `verification-summary.json` 通过 manifest 顶层 `summary_schema` 指向的 canonical schema 校验。
- 每个 artifact 的本地 `summary_schema=verification-summary.schema.json` 指向同目录 schema，下载 artifact 后不依赖仓库路径也能离线校验。
- `showcase-dual-project-report.sh` 的 combined summary 通过 `dual-project-summary.schema.json` 校验，不能当成 `verification-summary.json` 直接喂给 decision demo。
- `first-run` 使用 `first_run_agent_next_step`，`onboarding` 使用 `agent_next_step`。
- `expected_action=inspect-user-project` 时，客户端先进入用户项目失败排查，不先重装 testloop-mcp。
- 按 manifest 的 `expected_section_signals` 校验 `verification-summary.json` 和 `agent-response.txt` 都保留 `独立 CLI 生成动作 smoke:manual_review`。
- `verification-summary.json` 允许可选 `sections[].signals.action`，例如 `manual_review`，但该信号不等于整体失败。

## 相关入口

- [客户端集成说明](./client-integration.md)
- [Agent 结构化契约](./agent-contract.md)
- [Agent Action 决策表](./agent-action-guide.md)
- [真实结构化 fixture](./fixtures.md)
- [agent-response-artifact-manifest.json](./fixtures/agent-response-artifact-manifest.json)
- [agent-response-artifact-manifest.schema.json](./fixtures/agent-response-artifact-manifest.schema.json)
- [verification-summary.schema.json](./fixtures/verification-summary.schema.json)
- [dual-project-summary.schema.json](./fixtures/dual-project-summary.schema.json)
- [validate_coverage_task 结构化返回样例](./validate-coverage-task-samples.md)
