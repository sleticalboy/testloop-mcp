# Agent Action 决策表

这份表面向 MCP 客户端和 AI Coding Agent。`validate_coverage_task` 的主入口不是单独的 `status`，而是 `status/action` 组合；尤其是 `status=passed` 并不总等于“可以吸收生成测试”。客户端应优先读取结构化返回中的 `action`，再决定下一步。

## 核心规则

- `action=ready`：可以吸收生成测试，进入下一个任务或重新统计覆盖率。
- `action=manual_review_*`：不要继续自动修同一个生成测试；记录原因，改走公共入口、环境设计、依赖注入或人工复核。该 action 可出现在 `passed` 或 `failed` 状态。
- `action=apply_fix_suggestions`：优先读取 `run_result.fix_suggestions[].repair_task`。
- `action=repair_generated_test`：读取 `run_result.failures[]`，判断修测试草稿还是修实现。
- `action=needs_better_input`：测试可能通过了，但目标覆盖行没命中；需要更强输入或更合适的公共入口。
- `generation_error` / `run_error`：先修生成器/provider 或测试运行环境，不要把它当业务测试失败处理。

覆盖率命中校验当前支持 Jest/Vitest/Mocha 的 Istanbul `coverage/coverage-final.json`、`node-test` 的 TAP coverage raw output、pytest 项目根目录的 `coverage.json` 和 Java/JUnit 的 JaCoCo XML。客户端看到 `metadata.coverage_target_hit=false` 时，应优先把它当作输入/入口不足，而不是测试运行器失败。

## 决策表

| status | action | 客户端下一步 |
| --- | --- | --- |
| `passed` | `ready` | 接受生成测试；继续下一个 coverage task，或重新运行覆盖率统计确认收益。 |
| `passed` | `manual_review_unreachable` | 记录不可达原因；不要继续生成同类输入。必要时由人工确认源码条件是否死分支。 |
| `passed` | `manual_review_environment` | 记录环境依赖；需要 OS/runtime/hardware fixture、依赖注入或集成测试环境。 |
| `passed` | `manual_review_protocol` | 记录协议依赖；需要 fake connection、stream/socket 注入点或协议层集成测试。 |
| `passed` | `manual_review_database` | 记录数据库依赖；先设计测试数据库、mock repository 或事务注入策略。 |
| `passed` | `manual_review_external_service` | 记录外部服务依赖；使用 fake client、route data、短超时 wrapper 或集成环境验证。 |
| `passed` | `manual_review_private` | 不要直接调用私有方法；优先查看 `metadata.public_entry_candidates`，改走公共入口或重构可见性。 |
| `passed` | `manual_review_internal` | 不要导入未导出内部符号；改走已导出的公共 API、测试 seam 或模块级集成测试。 |
| `passed` | `manual_review_no_runtime` | 不要为纯类型/barrel 文件生成运行时测试；通过消费方测试、类型检查或包入口测试验证。 |
| `failed` | `apply_fix_suggestions` | 读取 `run_result.fix_suggestions[].repair_task`，按 `target_file`、`editable_files` 和 `suggested_commands` 执行修复闭环。 |
| `failed` | `manual_review_external_service` | 失败来自 live RPC、对象存储、路由状态或长重试时序；不要自动修生成测试，改用 fake client、route data 或集成环境验证。 |
| `failed` | `repair_generated_test` | 读取 `run_result.failures[]`；如果失败来自测试草稿输入/断言，修生成测试；如果暴露真实 bug，再修实现。 |
| `failed` | `needs_better_input` | 当前测试没有命中目标行；根据 `metadata.coverage_miss_reason`、`coverage_hit_lines`、`coverage_missed_lines` 重新选择输入或公共入口。 |
| `generation_error` | `inspect_generation_error` | 读取 `error`；通常是文件缺失、源码不可解析或静态生成器无法处理。先修任务上下文或源码路径。 |
| `generation_error` | provider-specific action | 读取 `provider_error.action`；按 LLM/provider 策略决定重试、降级 `static` 或提示用户修配置。 |
| `run_error` | `inspect_test_runner` | 先修测试命令、依赖安装、工作目录、框架识别或项目环境。 |

## 不要做的事

- 不要只看 `status=passed` 就自动合入测试；`manual_review_*` 也常常是 `passed`，它表示生成了可运行的手审草稿或 skip。
- 不要对 `manual_review_*` 反复调用同一个 `generate_tests`；即使它的 `status=failed`，也不应直接进入 `apply_fix_suggestions` 修复循环。
- 不要把 `suggested_fix` 当补丁直接应用；应优先使用 `repair_task` 限定修复范围。
- 不要无限重试 provider；同一任务最多重试一次，仍失败时降级 `static` 或提示用户修配置。

## 字段来源

`validate_coverage_task` 返回的关键字段：

- `status`
- `action`
- `coverage_task`
- `generated`
- `run_result`
- `provider_error`
- `metadata`

这些字段已由 Agent 结构化契约和 handler 级一致性测试保护。客户端应优先读取 `structuredContent`，旧客户端可以 fallback 到 `content[0].text` JSON。典型返回可对照 [validate_coverage_task 结构化返回样例](./validate-coverage-task-samples.md)。
