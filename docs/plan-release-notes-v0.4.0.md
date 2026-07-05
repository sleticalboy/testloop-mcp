# v0.4.0 发布说明

## 标题

testloop-mcp v0.4.0

## 摘要

v0.4.0 聚焦覆盖率驱动的测试生成闭环。这个版本让 `parse_coverage` 输出的 `test_tasks` 不再只是建议文本，而是可以直接作为 `generate_tests.coverage_task` 输入，生成面向单个覆盖率缺口的增量测试草稿。

## 主要变化

- `test_tasks` 新增 `test_file`、`test_name`、`assertion_focus`、`priority` 和 `priority_reason`，让 AI Agent 更容易按收益排序并定位要补的测试。
- `generate_tests` 支持接收单个 `coverage_task`，优先写入任务推荐的测试文件，并在返回结果和 provider context 中回显任务上下文。
- Go 静态生成器在 task 模式下会聚焦目标函数或方法，使用任务推荐测试名，并把缺口类型写入 case 名和注释。
- Python/Jest 静态生成器在 task 模式下会消费 `assertion_focus` 和 `suggested_inputs`，生成更具体的测试名、调用参数和断言。
- Rust/Java 静态生成器在 task 模式下会减少整文件泛化输出，优先生成目标函数或方法的测试骨架。
- Go/Rust/Java coverage task 输出和 Go/Python/Jest/Rust/Java task-aware 静态生成输出都新增 golden tests，降低 Agent 契约和生成草稿退化风险。

## 典型闭环

1. 用 `run_tests` 或生态工具生成覆盖率数据。
2. 调用 `parse_coverage` 获取按优先级排序的 `test_tasks`。
3. 选择一个任务作为 `generate_tests.coverage_task` 传入。
4. 让 static provider 生成增量测试草稿，或让 LLM provider 基于 `static_code` 和 `context.coverage_task` 增强断言。
5. 再次调用 `run_tests`，根据结果继续修复或进入下一个覆盖率任务。

## 示例输入

```json
{
  "file_path": "src/calc.py",
  "coverage_task": {
    "id": "pytest-1",
    "framework": "pytest",
    "file": "src/calc.py",
    "target": "add",
    "line_range": "2-2",
    "gap_type": "return_path",
    "test_file": "tests/test_calc.py",
    "test_name": "test_add_covers_gap",
    "suggested_inputs": ["构造满足条件 `a == 0` 的输入"],
    "assertion_focus": ["断言未覆盖返回路径的具体结果"]
  }
}
```

## 已知限制

- task-aware 静态生成仍是测试草稿，不保证直接覆盖复杂业务依赖、外部 IO、mock 和 fixture 构造。
- `suggested_inputs` 的参数提取目前主要覆盖简单条件表达式，例如 `a == 0`、`mode === 'short'`。
- Rust/Java 生成器仍偏骨架化，适合给 Agent 提供目标明确的增量测试起点。

## 发布前验证

- [x] `go test ./...`
- [x] GitHub Actions CI passed

## 建议发布命令

```bash
git tag v0.4.0
git push origin v0.4.0
gh release create v0.4.0 --title "testloop-mcp v0.4.0" --notes-file docs/plan-release-notes-v0.4.0.md
```
