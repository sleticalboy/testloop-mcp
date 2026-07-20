# Agent 结构化契约

testloop-mcp 的核心定位是给 AI Coding Agent 提供测试反馈闭环，因此 MCP 工具返回值必须优先面向机器消费。客户端应优先读取 `structuredContent`，旧客户端可以继续读取 `content[0].text` 中的 JSON；两者的 JSON 语义应保持一致。

## 兼容规则

- 已发布字段名默认稳定，不应重命名或改变含义。
- 可以追加新字段；Agent 客户端必须忽略未知字段。
- 可选字段缺失不代表失败，客户端应先根据 `status`、`action`、`provider_error.action` 等主入口字段决策。
- 文本 `error` 只作为人类可读摘要或旧客户端 fallback；自动决策应读取结构化字段。

## 关键输出

### `run_tests`

主入口字段：

- `status`：`pass` 或 `fail`。
- `action`：Agent 下一步动作入口。
- `framework`：实际执行/解析的测试框架。
- `failures[]`：失败测试结构。
- `fix_suggestions[]`：开启 `include_fix_suggestions=true` 且存在失败时返回。
- `raw_output`：原始测试输出，供人工排查或 parser fallback。

`run_tests.action` 的基础语义：

- `ready`：至少有一个真实测试通过，且当前结果没有失败。
- `manual_review`：命令通过但没有真实执行通过的用例，例如全部 skipped/TODO；不要把这类结果当作有效覆盖。
- `apply_fix_suggestions`：测试失败且已内联 `fix_suggestions[]`，优先执行 `repair_task`。
- `inspect_failures`：测试失败且有结构化 `failures[]`，但未请求或未生成修复建议。
- `inspect_test_runner`：测试命令失败但没有可归因到具体用例的结构化失败。

Agent 应优先读取 `fix_suggestions[].repair_task`。`repair_task` 的稳定字段包括：

- `id`
- `test_name`
- `category`
- `issue`
- `target_file`
- `target_line`
- `context_file`
- `context_line`
- `context_snippet`
- `editable_files`
- `suggested_commands`
- `assertion_focus`

`fix_suggestions[].category` 当前覆盖 `expectation_mismatch`、`runtime_panic`、`index_out_of_range`、`divide_by_zero`、`undefined_symbol`、`type_mismatch`、`module_resolution`、`python_import_error`、`compile_error` 和 `generic_failure`。客户端应把 category 当作分流提示，不应假设未来不会新增更细分类。

### `generate_tests`

主入口字段：

- `status`
- `test_file`
- `generated_cases`
- `action`
- `context`
- `coverage_task`
- `provider`
- `provider_error`

`action` 用于普通生成后的下一步分流：`ready` 表示可以直接进入 `run_tests`；`manual_review` 表示生成内容包含 TODO/skip 或手审标记，Agent 应先补充输入/断言或转入人工确认，不应把 skipped 测试当作有效覆盖。 当 LLM provider 或外部 provider 失败时，Agent 应读取 `provider_error.kind` 和 `provider_error.action` 决定是否重试、降级 `static` 或提示用户修配置。

### `validate_coverage_task`

主入口字段：

- `status`
- `action`
- `coverage_task`
- `generated`
- `run_result`
- `provider_error`
- `metadata`

Agent 决策应以 `status/action` 为准：

- `passed/ready`：生成测试已通过，可以进入下一个任务或重新统计覆盖率。
- `failed/apply_fix_suggestions`：读取 `run_result.fix_suggestions[].repair_task`。
- `failed/needs_better_input`：测试通过能力不足，目标覆盖行未命中，需要更强输入或公共入口。
- `generation_error`：读取 `provider_error.action` 或 `error`。
- `run_error`：先修测试命令、依赖或项目环境。
- `manual_review_*`：不要继续自动修同一个生成测试，应交给人工复核、公共入口测试或环境设计。

更完整的 action 决策建议见 [Agent Action 决策表](./agent-action-guide.md)，典型返回可对照 [validate_coverage_task 结构化返回样例](./validate-coverage-task-samples.md)，客户端回归建议见 [客户端集成说明](./client-integration.md) 和 [MCP 客户端契约测试说明](./mcp-client-contract-tests.md)。

## 回归保护

`types/agent_contract_test.go` 固定了上述关键 JSON 字段名。新增字段可以直接追加；如果确实需要改名或改变语义，应先新增兼容字段、更新文档和客户端迁移说明，再在后续主版本中移除旧字段。

`tools/tool_result_contract_test.go` 固定主 MCP 工具 handler 层的返回一致性：`generate_tests`、`run_tests`、`parse_results`、`parse_coverage` 和 `fix_suggestions` 的 `structuredContent`、handler 返回值与 `content[0].text` JSON 必须语义一致。这样客户端优先读 `structuredContent`，旧客户端 fallback 到 text JSON 时，不会拿到不同决策字段。

`test/e2e` 还包含真实 stdio 进程级 smoke：测试会先构建当前 `testloop-mcp` 二进制，再通过 MCP SDK `CommandTransport` 启动 `--transport=stdio` 进程，执行 `tools/list` 和一次 `parse_results` 调用，并复用 e2e helper 校验 `structuredContent` 与 text JSON 语义一致。这个测试覆盖的是客户端实际接入路径，不只是 in-memory server。

`test/e2e` 同时覆盖真实 Streamable HTTP 进程级 smoke：测试会启动 `testloop-mcp --transport=http`，等待 `/healthz` 返回 200，再通过 MCP SDK `StreamableClientTransport` 调用 `tools/list` 和 `parse_results`。这保证 HTTP 接入路径也能返回同一套结构化契约。
