# 真实结构化 fixture

这个目录给 Agent 客户端、编辑器插件和 MCP 集成测试复用。`docs/fixtures/*.json` 不是手写示意样例，而是 `tools` 层测试通过临时项目真实调用 `HandleValidateCoverageTask` 后生成的稳定投影。

接入方如何把这些 fixture 用到自己的客户端回归里，见 [客户端集成说明](./client-integration.md)。

## fixture 列表

| 文件 | status/action | 来源 | Agent 下一步 |
| --- | --- | --- | --- |
| [validate-coverage-task-ready.json](./fixtures/validate-coverage-task-ready.json) | `passed/ready` | 临时 Go 项目，`Add` 分支 coverage task 生成并运行通过 | 接受生成测试，继续下一个 coverage task 或重新统计覆盖率。 |
| [validate-coverage-task-manual-review-internal.json](./fixtures/validate-coverage-task-manual-review-internal.json) | `passed/manual_review_internal` | 临时 JS/Vitest 项目，未导出的 `LocalCache.get` 只能生成可运行手审 skip | 不要合入为有效覆盖率补丁；改走导出 API、test seam 或人工复核。 |
| [validate-coverage-task-apply-fix-suggestions.json](./fixtures/validate-coverage-task-apply-fix-suggestions.json) | `failed/apply_fix_suggestions` | 临时 Go 项目，已有失败测试触发 `failures[]`、`fix_suggestions[]` 和 `repair_task` | 优先读取 `run_result.fix_suggestions[].repair_task`，按限定文件和命令进入修复闭环。 |

## 稳定字段

这些 fixture 有意保留 Agent 决策需要的字段：

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
- `metadata`

## 过滤规则

为了让 fixture 可以跨机器、跨 CI 稳定复用，测试会做稳定投影：

- 临时目录绝对路径会规范成 fixture 项目内相对路径，例如 `calc.go`、`cache.test.js`。
- `raw_output` 会被过滤，因为它包含 Go/Vitest 输出细节、耗时和临时路径。
- 覆盖率百分比只保留 handler 当前返回值；没有显式 coverage 的样例通常是 `0`。
- `failures` 保留真实 JSON 形状；当前 ready Go 样例里该字段是 `null`，不是空数组。
- `fix_suggestions` 只在失败样例中保留，并包含可执行的 `repair_task`。

## 维护方式

修改 `validate_coverage_task`、`run_tests`、parser、fix suggestion 或静态生成器时，如果真实结构化输出语义变化，应同步更新对应 fixture 和文档。测试入口在 `tools/validate_coverage_task_test.go`：

- `TestHandleValidateCoverageTaskReadyFixture`
- `TestHandleValidateCoverageTaskManualReviewInternalFixture`
- `TestHandleValidateCoverageTaskApplyFixSuggestionsFixture`

如果只是 `raw_output`、测试耗时或临时路径变化，不应扩大 fixture 字段；优先在投影函数中继续过滤不稳定信息。
