# Agent 闭环工作流示例

这个示例展示 AI Agent 或编辑器集成如何把 `run_tests`、`parse_results`、`parse_coverage` 和 `generate_tests` 串成一个可执行闭环。示例使用仓库内 Go demo，其他语言只需要替换测试命令和覆盖率格式。

## 1. 运行测试

先让 Agent 调用 MCP 工具：

```json
{
  "tool": "run_tests",
  "arguments": {
    "path": "./demo",
    "framework": "go-test",
    "coverage": false,
    "include_fix_suggestions": true,
    "source_code": "./demo/calc.go",
    "test_code": "./demo/calc_test.go"
  }
}
```

`run_tests` 会直接返回结构化结果：

```json
{
  "status": "pass",
  "framework": "go-test",
  "passed": 8,
  "failed": 0,
  "failures": [],
  "raw_output": "..."
}
```

如果测试失败且调用时传入了 `include_fix_suggestions=true`，Agent 应优先读取 `fix_suggestions[].repair_task`，减少一次额外的 `fix_suggestions` 调用。未开启该选项，或失败输出来自 CI、终端等非 MCP 来源时，可以把原始日志交给 `parse_results`：

```json
{
  "tool": "parse_results",
  "arguments": {
    "framework": "go-test",
    "output": "<raw go test -json output>"
  }
}
```

## 2. 修复真实失败

当 `run_tests` 或 `parse_results` 返回 `status: "fail"` 时，不要先补覆盖率。如果 `run_tests` 已返回 `fix_suggestions[]`，直接使用其中的 `repair_task`；否则把 `failures[]` 序列化成 JSON 字符串，连同源码路径交给 `fix_suggestions`：

```json
{
  "tool": "fix_suggestions",
  "arguments": {
    "failures": "[{\"test_name\":\"TestDivideByZero\",\"file\":\"./demo/calc.go\",\"line\":18,\"error\":\"got nil, want divide by zero error\"}]",
    "source_code": "./demo/calc.go",
    "test_code": "./demo/calc_test.go"
  }
}
```

`fix_suggestions` 返回结构化修复建议：

```json
[
  {
    "file": "./demo/calc.go",
    "line": 18,
    "issue": "got nil, want divide by zero error",
    "category": "expectation_mismatch",
    "context_file": "./demo/calc_test.go",
    "context_line": 12,
    "suggested_fix": "期望值不匹配...",
    "confidence": 0.8,
    "repair_task": {
      "id": "repair-expectation_mismatch-testdivide",
      "test_name": "TestDivide",
      "category": "expectation_mismatch",
      "target_file": "./demo/calc_test.go",
      "target_line": 12,
      "context_file": "./demo/calc_test.go",
      "context_line": 12,
      "context_snippet": "if got := Divide(1, 0); got == nil { ... }",
      "editable_files": ["./demo/calc.go", "./demo/calc_test.go"],
      "suggested_commands": ["go test ./..."],
      "assertion_focus": "对比实际值和期望值，判断应修正测试断言还是实现返回路径。"
    }
  }
]
```

Agent 应优先读取 `repair_task`，用 `target_file` / `target_line` 跳转，用 `editable_files` 限定改动范围，用 `suggested_commands` 复跑验证；`suggested_fix` 是修复线索，不应被直接当作补丁应用。只有当前失败闭环收敛后，才进入覆盖率缺口分析。

这个内联修复任务契约已经用 `tools/testdata/golden/run_tests_repair_task.golden` 固定：测试会运行真实 Go 失败用例，调用 `run_tests` 并开启 `include_fix_suggestions`，然后比对 `failures[]`、`fix_suggestions[]` 和 `repair_task` 的稳定 JSON 字段。后续修改字段名、路径选择、上下文行或建议命令时，需要同步更新该 golden。

## 3. 生成覆盖率报告

`parse_coverage` 解析已有报告文件，不负责替代生态工具生成报告。Go 项目可先让 Agent 执行：

```bash
go test ./demo -coverprofile=/tmp/testloop-demo-coverage.out
```

然后调用：

```json
{
  "tool": "parse_coverage",
  "arguments": {
    "framework": "go-test",
    "data": "/tmp/testloop-demo-coverage.out"
  }
}
```

返回里的 `test_tasks[]` 是面向 Agent 的增量测试计划，包含目标函数、未覆盖行、建议测试文件、测试函数名、断言重点和建议输入。

## 4. 验证单个覆盖率任务

推荐优先用 `validate_coverage_task` 处理单个 `test_tasks[]`。它会把 `generate_tests -> run_tests` 合成一次工具调用，并返回生成结果、测试结果和下一步动作：

```json
{
  "tool": "validate_coverage_task",
  "arguments": {
    "file_path": "./demo/calc.go",
    "provider": "static",
    "coverage": true,
    "coverage_task": {
      "id": "go-test-1",
      "framework": "go-test",
      "file": "./demo/calc.go",
      "target": "Divide",
      "kind": "function",
      "line_range": "17-19",
      "gap_type": "error_path",
      "suggested_inputs": ["b == 0"],
      "goal": "为 Divide 补充测试，覆盖除零错误路径",
      "command": "go test ./demo",
      "test_file": "./demo/calc_test.go",
      "test_name": "TestDivideCoverageTask",
      "assertion_focus": ["断言除零错误返回"],
      "confidence": 0.9
    }
  }
}
```

返回中的 `status` 和 `action` 是 Agent 的主入口：

- `passed` / `ready`：生成的测试已经跑通，可以进入下一个 coverage task 或重新统计覆盖率。
- `passed` / `manual_review_unreachable`：生成结果能跑通，但目标分支疑似不可达，例如源码条件本身无法满足；记录复核原因后进入下一个任务。
- `passed` / `manual_review_environment`：生成结果能跑通，但目标错误分支依赖 OS、硬件、外部运行时或第三方库内部错误；通常需要依赖注入或人工重构后再补。
- `failed` / `apply_fix_suggestions`：生成成功但测试失败，优先读取 `run_result.fix_suggestions[].repair_task`。
- `failed` / `repair_generated_test`：生成成功但没有足够修复摘要，读取 `run_result.failures[]` 后修测试或实现。
- `generation_error`：测试未生成，读取 `provider_error.action` 或 `error`，必要时降级 `provider: "static"`。
- `run_error`：测试执行入口异常，先修命令、框架或项目环境。

`validate_coverage_task` 默认在测试失败时开启 `include_fix_suggestions`。如果 Agent 需要调试 provider 输出、分步审查生成内容，仍可使用下面的手动路径。

## 5. 手动按覆盖率任务生成增量测试

取 `parse_coverage` 返回的一个 `test_tasks[]` 项，作为 `generate_tests.coverage_task` 传入：

```json
{
  "tool": "generate_tests",
  "arguments": {
    "file_path": "./demo/calc.go",
    "provider": "static",
    "coverage_task": {
      "id": "go-test-1",
      "framework": "go-test",
      "file": "./demo/calc.go",
      "target": "Divide",
      "kind": "function",
      "line_range": "17-19",
      "gap_type": "error_path",
      "suggested_inputs": ["b == 0"],
      "goal": "为 Divide 补充测试，覆盖除零错误路径",
      "command": "go test ./demo",
      "test_file": "./demo/calc_test.go",
      "test_name": "TestDivideCoverageTask",
      "assertion_focus": ["断言除零错误返回"],
      "confidence": 0.9
    }
  }
}
```

`generate_tests` 会把任务写入 `context.coverage_task`，并优先使用任务里的 `test_file`、`test_name`、`suggested_inputs` 和 `assertion_focus` 收窄生成范围。

如果项目已配置外部 LLM provider，可以把 `provider` 改为 `auto` 或 `llm`：

```json
{
  "tool": "generate_tests",
  "arguments": {
    "file_path": "./src/sum.ts",
    "framework": "vitest",
    "provider": "auto"
  }
}
```

这只改变测试草稿的生成来源，不改变闭环终止条件。Agent 仍然需要读取返回的 `test_file`，并立即进入下一步 `run_tests`。

如果 `generate_tests` 返回 LLM provider 错误，Agent 应优先读取工具结果 JSON / `structuredContent` 中的 `provider_error.kind` 和 `provider_error.action` 选择处理策略。`error` 文本仍包含 `provider_error kind=... action=...`，只读文本的旧 Agent 可以继续作为 fallback 解析：

| kind | action | Agent 策略 |
| --- | --- | --- |
| `llm_config_missing` | `configure_provider` | 不重试模型；提示用户配置 `TESTLOOP_LLM_PROVIDER_CMD`，或改用 `provider: "auto"` / `static`。 |
| `llm_command_failed` | `fix_provider_command_or_retry` | 优先读取错误里的 stderr；如果像网络、限流或模型服务暂时失败，可以重试一次；如果像命令不存在、鉴权失败或脚本错误，提示用户修 provider。 |
| `llm_empty_output` | `retry_model_or_fallback_static` | 可重试一次；仍为空时降级 static，并保留错误摘要。 |
| `llm_json_error` | `retry_model_or_fallback_static` | 如果 provider 配置为 JSON 输出，先修 wrapper 或 prompt；Agent 自动流程可降级 static。 |
| `llm_missing_code` | `retry_model_or_fallback_static` | 要求 provider 返回非空 `code` 字段；自动流程可降级 static。 |
| `llm_output_cleaning_failed` | `retry_model_or_fallback_static` | 要求模型只输出测试代码；可重试一次，失败后降级 static。 |
| `llm_output_validation_failed` | `adjust_prompt_or_fallback_static` | 说明模型输出不是目标语言测试；优先调整 prompt，自动流程可降级 static。 |

自动化 Agent 不应无限重试 provider。建议同一任务最多重试一次 LLM provider；第二次仍失败时降级 static，继续执行 `run_tests`，并把 provider 错误作为上下文记录。

自动降级时不要重新推断目标文件或测试框架，复用同一个 `file_path`、`framework` 和 `coverage_task`，只把 provider 改为 `static`：

```json
{
  "tool": "generate_tests",
  "arguments": {
    "file_path": "./src/sum.ts",
    "framework": "vitest",
    "provider": "static"
  }
}
```

如果 static fallback 返回 `status: "ok"`，立即用返回的 `test_file` 进入 `run_tests`。这个降级闭环已有 handler 回归测试覆盖：fake LLM provider 返回解释文本，`generate_tests` 返回结构化 `provider_error.action=retry_model_or_fallback_static`，随后同一源文件降级 static 生成 Vitest 测试，并由 fake `npx vitest` 执行通过。

## 6. 重新运行并收敛

生成测试后再次调用：

```json
{
  "tool": "run_tests",
  "arguments": {
    "path": "./demo",
    "framework": "go-test",
    "coverage": true,
    "include_fix_suggestions": true,
    "source_code": "./demo/calc.go",
    "test_code": "./demo/calc_test.go"
  }
}
```

如果失败，Agent 应读取结构化 `failures[]` 和内联 `fix_suggestions[]`。未开启 `include_fix_suggestions` 或上下文不足时，再单独调用 `fix_suggestions` 获取修复建议。闭环终止条件不是“生成了测试”，而是测试结果、失败结构和覆盖率任务都达到可接受状态。

LLM provider 输出也必须走同样的路径：`generate_tests(provider=auto/llm) -> run_tests(include_fix_suggestions=true) -> repair_task -> rerun`。当前回归测试已固定一个 Vitest dry-run 链路：fake LLM provider 生成测试文件，fake `npx vitest` 返回断言失败，`run_tests` 解析失败并内联 `repair_task`。

## Agent 策略

- 先修真实失败，再补覆盖率缺口。
- 优先选择 `test_tasks[]` 中 `priority` 更高、`target` 更具体、`suggested_inputs` 更明确的任务。
- `generate_tests` 的 static 输出是草稿；复杂业务断言应由 Agent 或 LLM provider 结合源码继续增强。
- LLM provider 的产物不是最终结论；必须跑 `run_tests`，失败时使用 `repair_task` 限定修复范围。
- provider 错误要按 `provider_error kind/action` 处理；不要对配置错误或格式错误做无上限重试。
- 不要把整段日志直接塞给模型；优先使用 `run_tests` / `parse_results` 的结构化 JSON。
