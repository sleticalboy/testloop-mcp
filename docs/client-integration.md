# 客户端集成说明

这份文档面向 MCP 客户端、编辑器插件和 AI Coding Agent 的接入方。目标不是重新解释所有字段，而是给出一条可回归的消费流程：优先读取 `structuredContent`，按 `status/action` 分流，再用真实 fixture 固定自己的客户端行为。

只接入 Agent 决策 fixture CI 时，可以先看 [Agent 决策客户端 CI 接入 Checklist](./agent-decision-client-ci-checklist.md)。

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

该示例读取 [agent-decision-fixtures.json](./fixtures/agent-decision-fixtures.json)，并用 [agent-decision-fixtures.schema.json](./fixtures/agent-decision-fixtures.schema.json) 固定最小决策样本清单。客户端可以直接复制这个 manifest，而不是自己维护 fixture glob 顺序。

该示例读取 [真实结构化 fixture](./fixtures.md) 中的 JSON，演示如何把：

- `passed/ready` 映射为 `accept`
- `passed/manual_review_internal` 映射为 `manual-review`
- `passed/manual_review_environment` 映射为 `manual-review`
- `failed/manual_review_external_service` 映射为 `manual-review`
- `failed/apply_fix_suggestions` 映射为 `apply-repair`
- `failed/needs_better_input` 映射为 `needs-better-input`

当 fixture 提供 `metadata.next_action_reason`、`metadata.manual_review_reason`、`metadata.needs_better_input_reason` 或兼容原因字段时，demo 输出也会带 `reason="..."`，方便 Agent 把分流原因直接写入任务记录，而不是只看到动作名。

这个 demo 会通过 manifest 同时读取 `docs/fixtures/validate-coverage-task-*.json` 和 `docs/fixtures/real-project-agent-loop/*.json`。后者来自 laoxia server、mcp-hub 这类真实项目的脱敏验证摘要，用来确认真实项目证据也走同一套 `status/action` 分流，而不是另写一套客户端逻辑。

如果你在做自己的客户端，建议把同样的映射逻辑做成单元测试，而不是只在真实项目里手动观察。

接入方测试应优先读取 manifest 的 `fixtures[].expected_decision` 做断言；这样新增 `failed/manual_review_*` 这类真实项目分流时，客户端只需要同步 manifest 和 fixture，而不是维护另一份文件名白名单。

如果客户端项目使用 Node，可以先复制仓库内的无依赖参考实现，再按本项目的 fixture 目录调整路径：

```bash
node scripts/validate-agent-decision-fixtures.mjs \
  docs/fixtures/agent-decision-fixtures.json \
  .
```

CI 中需要机器断言时使用 JSON 输出：

```bash
node scripts/validate-agent-decision-fixtures.mjs --json \
  docs/fixtures/agent-decision-fixtures.json \
  .
```

该 JSON 固定包含 `schema_version`、`status`、`fixture_count`、`decisions[]`、`fixtures[]` 和 `failures[]`；`fixtures[]` 会在可用时带 `reason`，来源同样优先使用 `metadata.next_action_reason`。输出结构见 [Agent 决策 fixture validator result schema](./fixtures/agent-decision-fixtures-result.schema.json)，通过态样例见 [passed.json](./fixtures/agent-decision-fixtures-result/passed.json)。失败时仍输出 JSON，并以非 0 退出码让 CI 失败。
validator 不依赖 JSON Schema 工具链，也会检查 manifest 条目的 `kind`、`source`、`status`、`action`、`expected_decision` 和 `client_expectation`。

如果接入方只想复制最小决策 fixture 包，而不是整个仓库，可以先导出：

```bash
node scripts/export-agent-decision-fixtures.mjs /tmp/testloop-agent-decision-fixtures
```

导出目录会包含 `docs/fixtures/agent-decision-fixtures.json`、manifest schema、validator result schema、通过态 result fixture、manifest 中列出的 8 个 fixture，以及 `scripts/validate-agent-decision-fixtures.mjs`。路径保持为 `docs/fixtures/...`，所以复制到目标项目后仍可直接运行同一条 `--json` 校验命令。
导出包也包含无依赖 `package.json`，可以在客户端 CI 中用 `npm test --silent` 直接运行同一套契约测试。

如果要模拟外部客户端 CI 的完整接入路径，可以直接运行：

```bash
scripts/showcase-agent-decision-client-ci.sh
```

该脚本会在临时客户端目录中导出最小 fixture 包，进入导出目录执行 `npm test --silent`，把 validator JSON 写到客户端目录，并输出稳定摘要。正常输出应包含 `agent_decision_client_status=passed`、`agent_decision_fixture_count=8` 和完整 `agent_decision_decisions=accept,accept,accept,manual-review,manual-review,manual-review,apply-repair,needs-better-input`。
CI 中需要机器断言时可以直接使用 JSON 输出：

```bash
scripts/showcase-agent-decision-client-ci.sh --json
```

该 JSON 固定包含 `schema_version`、`status`、`client_dir`、`fixture_dir`、`result_json`、`result_schema`、`fixture_count`、`decisions[]`、`failures[]` 和 `validator_exit_code`；结构契约见 [Agent 决策客户端 CI summary schema](./fixtures/agent-decision-client-ci-summary.schema.json)，通过态样例见 [passed.json](./fixtures/agent-decision-client-ci-summary/passed.json)。客户端不想引入 JSON Schema 工具链时，可以运行 `node scripts/validate-agent-decision-client-ci-summary.mjs /path/to/testloop-agent-decision-client-summary.json` 做无依赖校验。如果要把基础客户端 CI summary 转成 Agent 下一步动作，可运行 `node scripts/render-agent-decision-client-ci-response.mjs /path/to/testloop-agent-decision-client-summary.json`；通过态输出 `agent_next_step=ready`，失败时分流到 `inspect-client-validator`、`inspect-agent-decision-fixtures` 或 `inspect-agent-decision-client-summary`。基础客户端 CI response 的结构契约见 [Agent 决策客户端 CI response schema](./fixtures/agent-decision-client-ci-response.schema.json)，通过态和失败态 fixture 见 [passed.json](./fixtures/agent-decision-client-ci-response/passed.json)、[validator-failed.json](./fixtures/agent-decision-client-ci-response/validator-failed.json) 和 [fixture-drift.json](./fixtures/agent-decision-client-ci-response/fixture-drift.json)；无依赖校验入口是 `node scripts/validate-agent-decision-client-ci-response.mjs /path/to/client-ci-response.json`。
如果要直接安装 GitHub Actions job 到外部客户端仓库：

```bash
scripts/install-agent-decision-client-ci-template.sh /absolute/path/to/client-repo
```

该脚本默认生成 `.github/workflows/testloop-agent-decision-contract.yml`，并把 helper 固定到当前版本 tag；手动复制版本见 [Agent 决策客户端 CI 模板](./agent-decision-client-ci-template.md)。
外部接入方也可以从 `main` raw URL 下载单脚本运行，不需要 clone 整个 testloop-mcp 仓库；生成的 workflow 仍会固定到稳定 helper tag。
维护者可以运行 `scripts/showcase-agent-decision-client-ci-template-install.sh --json`，验证下载安装脚本、生成 workflow 和执行 contract 的完整外部客户端 dry-run；JSON 输出结构见 [Agent 决策客户端 CI 模板安装 summary schema](./fixtures/agent-decision-client-ci-template-install-summary.schema.json)，通过态样例见 [passed.json](./fixtures/agent-decision-client-ci-template-install-summary/passed.json)。客户端不想引入 JSON Schema 工具链时，可以运行 `node scripts/validate-agent-decision-client-ci-install-summary.mjs /path/to/install-summary.json` 做无依赖校验。
如果要继续验证接入方能否消费安装后的 artifact 链路，可运行 `scripts/showcase-agent-decision-client-consumer-smoke.sh --json`。该命令会校验安装 summary、基础客户端 CI summary、基础客户端 CI response、导出的 fixture manifest 和 `agent-decision-fixtures-result.json` 互相一致，并在 summary 中返回 `client_response_json`、`client_response_validator_json` 和 `agent_response_json`；JSON 输出结构见 [Agent 决策客户端消费端 smoke summary schema](./fixtures/agent-decision-client-consumer-smoke-summary.schema.json)，通过态样例见 [passed.json](./fixtures/agent-decision-client-consumer-smoke-summary/passed.json)。无依赖校验入口是 `node scripts/validate-agent-decision-client-consumer-smoke-summary.mjs /path/to/consumer-smoke-summary.json`。
如果客户端希望把消费端 smoke summary 直接喂给 Agent，可运行 `node scripts/render-agent-decision-client-consumer-response.mjs /path/to/consumer-smoke-summary.json`。该脚本会输出稳定的 `agent_next_step`：通过态为 `ready`；validator 失败分流到 `inspect-consumer-smoke-validator`；fixture 数量或决策序列漂移分流到 `inspect-agent-decision-fixtures`；其他结构问题分流到 `inspect-consumer-smoke-summary`。
consumer response 的结构契约见 [Agent 决策客户端 consumer response schema](./fixtures/agent-decision-client-consumer-response.schema.json)，通过态和失败态 fixture 见 [passed.json](./fixtures/agent-decision-client-consumer-response/passed.json)、[client-summary-validator-failed.json](./fixtures/agent-decision-client-consumer-response/client-summary-validator-failed.json)、[validator-failed.json](./fixtures/agent-decision-client-consumer-response/validator-failed.json) 和 [fixture-drift.json](./fixtures/agent-decision-client-consumer-response/fixture-drift.json)；无依赖校验入口是 `node scripts/validate-agent-decision-client-consumer-response.mjs /path/to/consumer-response.json`。消费端 summary 失败态 fixture 见 [client-summary-validator-failed.json](./fixtures/agent-decision-client-consumer-smoke-summary/client-summary-validator-failed.json)、[validator-failed.json](./fixtures/agent-decision-client-consumer-smoke-summary/validator-failed.json) 和 [fixture-drift.json](./fixtures/agent-decision-client-consumer-smoke-summary/fixture-drift.json)，适合客户端固定失败分流测试。
正式发布后需要一次性确认 release tag raw installer、基础客户端 CI response 和 consumer smoke response 时，可运行：

```bash
scripts/showcase-agent-decision-client-release-smoke.sh --json
```

该 JSON 输出结构见 [Agent 决策客户端 release smoke summary schema](./fixtures/agent-decision-client-release-smoke-summary.schema.json)，通过态样例见 [passed.json](./fixtures/agent-decision-client-release-smoke-summary/passed.json)。正常结果会固定 `release_ref=v0.5.21`、`helper_refs.install=v0.5.21`、`helper_refs.consumer=v0.5.21`、`fixture_count=8`，并要求基础客户端和消费端的 `agent_next_step` 都是 `ready`。
如果客户端希望把发布后 smoke 汇总直接变成 Agent 下一步动作，可运行：

```bash
node scripts/render-agent-decision-client-release-response.mjs \
  /path/to/release-smoke-summary.json
```

通过态输出 `agent_next_step=ready`；release installer 或 helper tag 漂移分流到 `inspect-release-installer`；基础客户端 response 漂移分流到 `inspect-release-client-response`；consumer response 漂移分流到 `inspect-release-consumer-response`；fixture 数量或决策序列漂移分流到 `inspect-agent-decision-fixtures`。这给外部 Agent 一个可复制的最小消费样例：先跑 release smoke，再用 renderer 把 summary 转成稳定动作。
release response 的结构契约见 [Agent 决策客户端 release response schema](./fixtures/agent-decision-client-release-response.schema.json)，通过态和失败态 fixture 见 [passed.json](./fixtures/agent-decision-client-release-response/passed.json)、[installer-drift.json](./fixtures/agent-decision-client-release-response/installer-drift.json)、[client-response-drift.json](./fixtures/agent-decision-client-release-response/client-response-drift.json)、[consumer-response-drift.json](./fixtures/agent-decision-client-release-response/consumer-response-drift.json) 和 [fixture-drift.json](./fixtures/agent-decision-client-release-response/fixture-drift.json)。
如果要验证“复制到独立客户端项目后仍可用”，可以运行：

```bash
scripts/showcase-agent-decision-client-release-response-smoke.sh --json
```

该脚本会创建一个临时 Node 客户端项目，把 release smoke summary、release response renderer、`package.json` 和断言脚本放进去，再运行该客户端自己的 `npm test`。通过态表示接入方可以只消费 summary JSON 和 renderer，不依赖 testloop-mcp 仓库内部路径。可复制目录结构见 [Agent 决策 release response 客户端接入](./agent-decision-release-response-client.md)。
如果要导出可复制最小包，可运行 `node scripts/export-agent-decision-release-response-client.mjs /tmp/testloop-release-response-client`，导出目录可直接执行 `npm test --silent`。
如果要写入真实外部仓库，可运行 `scripts/install-agent-decision-release-response-client.sh /absolute/path/to/client-repo`；该 installer 会安装 `testloop-release-response-client/`、`.github/workflows/testloop-release-response-contract.yml`，并在目标包目录执行 `npm test --silent`。
安装 summary 可用 `node scripts/validate-agent-decision-release-response-client-install-summary.mjs /path/to/install-summary.json` 校验，结构契约见 [agent-decision-release-response-client-install-summary.schema.json](./fixtures/agent-decision-release-response-client-install-summary.schema.json)，通过态样例见 [passed.json](./fixtures/agent-decision-release-response-client-install-summary/passed.json)。
如果要模拟接入方仓库的 GitHub Actions 形态，可运行 `scripts/showcase-agent-decision-client-release-response-ci.sh --json`；该命令会写入 `.github/workflows/testloop-release-response-contract.yml` 并运行同一条 `npm test --silent`。
如果要看外部仓库可直接照抄的最小样板，见 [Release response 接入方样板](../examples/release-response-adopter/README.md)，也可以运行 `scripts/showcase-release-response-adopter.sh --json` 验证 installer、workflow、`npm test --silent` 和接入方消费 helper 的完整链路；summary 可用 `node scripts/validate-release-response-adopter-summary.mjs /path/to/release-response-adopter-summary.json` 校验，结构契约见 [release-response-adopter-summary.schema.json](./fixtures/release-response-adopter-summary.schema.json)，通过态样例见 [passed.json](./fixtures/release-response-adopter-summary/passed.json)，失败态样例见 [invalid-response.json](./fixtures/release-response-adopter-summary/invalid-response.json)。如果 validator 失败，接入方可以用 `--json` 输出读取 `agent_next_step` 和 `failures[]`，也可以复制 `examples/release-response-adopter/scripts/read-testloop-release-response-summary.mjs` 输出 `testloop_release_response_summary_next_step`，交给 Agent 做分流。
接入样板 README 固定了两组 helper 输出字段：`testloop_release_response_*` 面向 `testloop-release-response.json`，`testloop_release_response_summary_*` 面向 adopter summary。
`scripts/showcase-release-response-adopter.sh --json` 默认会生成 `testloop-release-response-adopter-artifacts/`，输出目录可用 `TESTLOOP_RELEASE_RESPONSE_ADOPTER_ARTIFACT_DIR` 覆盖。外部 CI 建议直接上传这个目录，至少包含 `testloop-release-response-adopter-summary.json`、`testloop-release-response-install-summary.json`、`testloop-release-response-client/testloop-release-smoke-summary.json`、`testloop-release-response-client/testloop-release-response.json`、`testloop-release-response-consumer.json` 和 `testloop-release-response-summary-consumer.json`。
下载 artifact 后可运行 `node scripts/verify-release-response-adopter-artifact.mjs /path/to/testloop-release-response-adopter-artifacts` 离线自检；该 verifier 会检查必备文件、通过态字段、`agent_next_step=ready`、`should_accept=true` 和 summary 路径后缀，不要求原始 CI 绝对路径仍然存在。失败态会返回非 0，并输出 `agent_next_step=inspect-release-response-adopter-artifact` 和 `should_accept=false`。
如果客户端要把 verifier JSON 纳入单元测试，可运行 `node scripts/verify-release-response-adopter-artifact.mjs --json /path/to/testloop-release-response-adopter-artifacts > /tmp/testloop-release-response-adopter-artifact-verification.json`，再用 `node scripts/validate-release-response-adopter-artifact-verification.mjs /tmp/testloop-release-response-adopter-artifact-verification.json` 校验。结构契约见 [release-response-adopter-artifact-verification.schema.json](./fixtures/release-response-adopter-artifact-verification.schema.json)，通过态 fixture 见 [passed.json](./fixtures/release-response-adopter-artifact-verification/passed.json)，失败态 fixture 见 [missing-summary-consumer.json](./fixtures/release-response-adopter-artifact-verification/missing-summary-consumer.json)。
如果要看最小消费逻辑，可运行 `go run ./examples/release-response-adopter-artifact-demo docs/fixtures/release-response-adopter-artifact-verification/passed.json`；通过态输出 `client_decision=accept`，artifact 缺文件时输出 `client_decision=inspect-artifact`。

## 使用真实 fixture

[真实结构化 fixture](./fixtures.md) 提供了来自 handler 的稳定 JSON 投影，适合直接放进客户端测试用例：

| fixture | 期望客户端动作 |
| --- | --- |
| [run-tests/apply-fix-suggestions.json](./fixtures/run-tests/apply-fix-suggestions.json) | 对普通 `run_tests` 失败结果读取 `action`、`fix_suggestions[0].category` 和 `repair_task`。 |
| [validate-coverage-task-ready.json](./fixtures/validate-coverage-task-ready.json) | 接受生成测试，进入下一个 coverage task。 |
| [validate-coverage-task-manual-review-internal.json](./fixtures/validate-coverage-task-manual-review-internal.json) | 记录手审原因，不继续自动修同一个生成测试。 |
| [validate-coverage-task-apply-fix-suggestions.json](./fixtures/validate-coverage-task-apply-fix-suggestions.json) | 读取 `repair_task`，按限定文件和命令执行修复闭环。 |
| [validate-coverage-task-needs-better-input.json](./fixtures/validate-coverage-task-needs-better-input.json) | 读取覆盖率未命中原因，重新选择输入或公共入口。 |
| [real-project-agent-loop/laoxia-server-go-utils.json](./fixtures/real-project-agent-loop/laoxia-server-go-utils.json) | 对真实 Go server coverage task 的 `passed/ready` 摘要执行同样的 `accept` 分流。 |
| [real-project-agent-loop/mcp-hub-vitest-repair.json](./fixtures/real-project-agent-loop/mcp-hub-vitest-repair.json) | 对真实 Vitest 历史 repair 回归样本的 `passed/ready` 摘要执行同样的 `accept` 分流。 |
| [real-project-agent-loop/haoy-apk-station-py-environment.json](./fixtures/real-project-agent-loop/haoy-apk-station-py-environment.json) | 对真实 FastAPI 环境依赖样本的 `passed/manual_review_environment` 摘要执行同样的 `manual-review` 分流。 |
| [real-project-agent-loop/haoy-apk-station-py-external-service.json](./fixtures/real-project-agent-loop/haoy-apk-station-py-external-service.json) | 对真实 FastAPI 外部服务 timeout 样本的 `failed/manual_review_external_service` 摘要执行同样的 `manual-review` 分流。 |

建议客户端测试至少断言：

- 能从 fixture 解析 `status` 和 `action`。
- 能把 `real-project-agent-loop/*.json` 当成普通结构化结果消费，同时忽略额外的 `task`、`regression_note` 和 `redaction_note` 字段。
- `run_tests` 的 `fail/apply_fix_suggestions` 能定位 `fix_suggestions[0].category` 和 `fix_suggestions[0].repair_task`。
- `passed/ready` 不读取 `run_result.fix_suggestions`。
- `manual_review_*` 不触发自动修复循环。
- `failed/apply_fix_suggestions` 能定位 `run_result.fix_suggestions[0].repair_task.target_file`、`editable_files` 和 `suggested_commands`。
- `failed/needs_better_input` 能定位 `metadata.next_action_reason`、`metadata.needs_better_input_reason` 和 `metadata.coverage_missed_lines`。
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
- `verification-summary.schema.json`
- `agent-decision.txt`
- `first-run-context.txt`
- `agent-response.txt`
- `first-run.log`

onboarding fixture 包含：

- `verification-report.md`
- `verification-summary.json`
- `verification-summary.schema.json`
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

如果要直接校验下载后的 artifact 目录是否自洽，使用目录级 verifier：

```bash
sh scripts/verify-agent-artifact.sh \
  first-run \
  docs/fixtures/first-run-artifacts/user-project-smoke-failed/

sh scripts/verify-agent-artifact.sh \
  onboarding \
  docs/fixtures/onboarding-artifacts/user-project-smoke-failed/
```

正常输出会包含 `agent_artifact_status=passed`、`decision_action=inspect-user-project` 和 `response_action=inspect-user-project`。这个检查覆盖必备文件、同目录 summary schema、`agent-response.txt` 四段结构、失败 section、`exit_code` 和 `section_signal`。

如果要一次性校验 manifest 里登记的 first-run 和 onboarding artifact fixture：

```bash
sh scripts/verify-agent-artifact.sh \
  manifest \
  docs/fixtures/agent-response-artifact-manifest.json

sh scripts/verify-agent-artifact.sh \
  --json \
  manifest \
  docs/fixtures/agent-response-artifact-manifest.json
```

正常文本输出会包含 `agent_artifact_manifest_status=passed` 和 `artifact_count=2`。JSON 输出会包含 `status=passed`、`artifact_count=2`、`artifacts[].artifact_kind`、`artifacts[].decision_action`、`artifacts[].response_action` 和 `artifacts[].section_signals`。

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
