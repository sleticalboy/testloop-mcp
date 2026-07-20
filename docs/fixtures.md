# 真实结构化 fixture

这个目录给 Agent 客户端、编辑器插件和 MCP 集成测试复用。`docs/fixtures/*.json` 不是手写示意样例，而是 `tools` 层测试通过临时项目真实调用 `HandleValidateCoverageTask` 后生成的稳定投影。

接入方如何把这些 fixture 用到自己的客户端回归里，见 [客户端集成说明](./client-integration.md)。

Agent response artifact 的机器可读索引见 [agent-response-artifact-manifest.json](./fixtures/agent-response-artifact-manifest.json)，JSON Schema 见 [agent-response-artifact-manifest.schema.json](./fixtures/agent-response-artifact-manifest.schema.json)。客户端测试可以用它直接发现 first-run / onboarding artifact fixture、必备文件、固定字段、fallback 顺序和 `expected_section_signals`，并通过 `summary_schema` 找到 `verification-summary.json` 的结构契约：[verification-summary.schema.json](./fixtures/verification-summary.schema.json)。其中 `sections[].signals.action` 是可选的 section 级动作信号。双项目报告的 combined summary 使用独立结构契约：[dual-project-summary.schema.json](./fixtures/dual-project-summary.schema.json)，样例见 [laoxia-passed.json](./fixtures/dual-project-summary/laoxia-passed.json)。

## run_tests fixture 列表

| 文件 | status/action | category | 来源 | Agent 下一步 |
| --- | --- | --- | --- | --- |
| [run-tests/apply-fix-suggestions.json](./fixtures/run-tests/apply-fix-suggestions.json) | `fail/apply_fix_suggestions` | `expectation_mismatch` | 临时 Go 项目，失败断言触发 `run_tests include_fix_suggestions=true` | 读取 `fix_suggestions[0].repair_task`，按 `editable_files` 和 `suggested_commands` 修复后复跑。 |

## validate_coverage_task fixture 列表

| 文件 | status/action | 来源 | Agent 下一步 |
| --- | --- | --- | --- |
| [validate-coverage-task-ready.json](./fixtures/validate-coverage-task-ready.json) | `passed/ready` | 临时 Go 项目，`Add` 分支 coverage task 生成并运行通过 | 接受生成测试，继续下一个 coverage task 或重新统计覆盖率。 |
| [validate-coverage-task-manual-review-internal.json](./fixtures/validate-coverage-task-manual-review-internal.json) | `passed/manual_review_internal` | 临时 JS/Vitest 项目，未导出的 `LocalCache.get` 只能生成可运行手审 skip | 不要合入为有效覆盖率补丁；改走导出 API、test seam 或人工复核。 |
| [validate-coverage-task-apply-fix-suggestions.json](./fixtures/validate-coverage-task-apply-fix-suggestions.json) | `failed/apply_fix_suggestions` | 临时 Go 项目，已有失败测试触发 `failures[]`、`fix_suggestions[]` 和 `repair_task` | 优先读取 `run_result.fix_suggestions[].repair_task`，按限定文件和命令进入修复闭环。 |
| [validate-coverage-task-needs-better-input.json](./fixtures/validate-coverage-task-needs-better-input.json) | `failed/needs_better_input` | 临时 Java/JUnit 项目，测试命令通过但 JaCoCo 目标行未命中 | 不吸收该测试；读取 `metadata.coverage_miss_reason` 和未命中行，改用更强输入或更合适的公共入口。 |

## first-run artifact fixture

| 目录 | action | 内容 | Agent 下一步 |
| --- | --- | --- | --- |
| [user-project-smoke-failed](./fixtures/first-run-artifacts/user-project-smoke-failed/) | `inspect-user-project` | first-run 失败六件套：`verification-report.md`、`verification-summary.json`、`agent-decision.txt`、`first-run-context.txt`、`agent-response.txt`、`first-run.log`；summary 保留 `独立 CLI 生成动作 smoke` 的 `signals.action=manual_review` | 先打开用户项目 smoke 失败 section，再复跑同一条项目测试/构建命令；不要把 CLI 生成动作 smoke 的 `manual_review` 当作整体验收失败。 |

这类 fixture 面向 CI artifact 消费方：可以直接读取 `agent-response.txt`，也可以把 artifact 目录交给 [first-run artifact Agent 消费演示](./first-run-agent-artifact-demo.md)，不用每次都重新构造失败项目。

## onboarding artifact fixture

| 目录 | action | 内容 | Agent 下一步 |
| --- | --- | --- | --- |
| [user-project-smoke-failed](./fixtures/onboarding-artifacts/user-project-smoke-failed/) | `inspect-user-project` | onboarding 失败四件套：`verification-report.md`、`verification-summary.json`、`agent-decision.txt`、`agent-response.txt`；summary 保留 `独立 CLI 生成动作 smoke` 的 `signals.action=manual_review` | 先打开用户项目 smoke 失败 section，再复跑同一条项目测试/构建命令；不要把 CLI 生成动作 smoke 的 `manual_review` 当作整体验收失败。 |

这类 fixture 面向已经稳定接入后的 PR / 发布后 smoke：可以直接读取 `agent-response.txt`，也可以把 artifact 目录交给 `scripts/render-onboarding-agent-response.sh`，不用每次都重新构造失败项目。

## 稳定字段

`run_tests` fixture 有意保留 `status/action`、测试统计、`failures[]`、`fix_suggestions[].category` 和 `fix_suggestions[].repair_task`，并过滤 `raw_output`。

`validate_coverage_task` fixture 有意保留 Agent 决策需要的字段：

- `status`
- `action`
- `coverage_task`
- `generated.status`
- `generated.test_file`
- `generated.generated_cases`
- `generated.provider`
- `generated.coverage_task`
- `run_result.status`
- `run_result.framework`
- `run_result.total/passed/failed/skipped`
- `run_result.failures[]`
- `run_result.fix_suggestions[].repair_task`
- `metadata.coverage_target_hit`
- `metadata.coverage_hit_lines`
- `metadata.coverage_missed_lines`
- `metadata.coverage_miss_reason`
- `metadata`

## 过滤规则

为了让 fixture 可以跨机器、跨 CI 稳定复用，测试会做稳定投影：

- 临时目录绝对路径会规范成 fixture 项目内相对路径，例如 `calc.go`、`cache.test.js`。
- `raw_output` 会被过滤，因为它包含 Go/Vitest 输出细节、耗时和临时路径。
- 覆盖率百分比只保留 handler 当前返回值；没有显式 coverage 的样例通常是 `0`。
- JaCoCo 报告路径会规范成项目内相对路径，例如 `target/site/jacoco/jacoco.xml`。
- `failures` 保留真实 JSON 形状；当前 ready Go 样例里该字段是 `null`，不是空数组。
- `fix_suggestions` 只在失败样例中保留，并包含可执行的 `repair_task`。

## 维护方式

修改 `validate_coverage_task`、`run_tests`、parser、fix suggestion 或静态生成器时，如果真实结构化输出语义变化，应同步更新对应 fixture 和文档。测试入口：

- `tools/run_tests_fixture_test.go` 中的 `TestHandleRunTestsActionCategoryFixture`
- `TestHandleValidateCoverageTaskReadyFixture`
- `TestHandleValidateCoverageTaskManualReviewInternalFixture`
- `TestHandleValidateCoverageTaskApplyFixSuggestionsFixture`
- `TestHandleValidateCoverageTaskNeedsBetterInputFixture`

如果只是 `raw_output`、测试耗时或临时路径变化，不应扩大 fixture 字段；优先在投影函数中继续过滤不稳定信息。

修改 first-run/onboarding artifact fixture 或 `agent-response-artifact-manifest.json` 时，还必须同步：

- `docs/fixtures/agent-response-artifact-manifest.schema.json`
- `docs/fixtures/verification-summary.schema.json`
- `docs/fixtures/dual-project-summary.schema.json`
- `tools/agent_response_artifact_manifest_schema_test.go`
- `tools/verification_summary_schema_test.go`
- `tools/dual_project_summary_schema_test.go`
- `test/verification_summary_decision_demo_test.sh`
- `examples/agent-response-manifest-demo` 的输出断言
- `expected_section_signals` 与 fixture summary / `agent-response.txt` 中的 `section_signal`
- README、quickstart、接入方一页式验证指南和 MCP 客户端契约测试说明里的 manifest/schema 入口

推荐至少运行：

```bash
sh test/agent_response_artifact_manifest_test.sh
sh test/agent_response_manifest_demo_test.sh
sh test/verification_summary_decision_demo_test.sh
go test ./tools -run TestAgentResponseArtifactManifestSchema -count=1
go test ./tools -run TestDualProjectSummarySchema -count=1
```
