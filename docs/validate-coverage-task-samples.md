# validate_coverage_task 结构化返回样例

这份文档给 Agent 集成方看，不替代完整字段契约。客户端应优先读取 MCP `structuredContent`；旧客户端可以 fallback 到 `content[0].text` JSON。自动决策不要只看 `status`，应以 `status/action` 组合为准。

相关文档：

- [Agent Action 决策表](./agent-action-guide.md)
- [Agent 结构化契约](./agent-contract.md)
- [Agent 工作流](./agent-workflow.md)
- [真实结构化 fixture](./fixtures.md)

仓库里的 `go run ./examples/agent-decision-demo` 会读取本文档中的 JSON 样例，并演示一个最小客户端如何把 `status/action` 映射成 `accept`、`manual-review`、`apply-repair` 和 `needs-better-input`。

如果需要更贴近真实 handler 的可复用样例，可以查看 [真实结构化 fixture](./fixtures.md)。这些 JSON 由 `tools` 层测试通过临时项目真实调用 `HandleValidateCoverageTask` 后生成稳定投影，并在 CI 中比对。

## passed / ready

含义：生成测试已经通过，且没有被标记为人工复核或弱覆盖。Agent 可以接受生成测试，进入下一个 coverage task，或重新统计覆盖率确认收益。

```json
{
  "status": "passed",
  "action": "ready",
  "coverage_task": {
    "id": "go-test-1",
    "framework": "go-test",
    "file": "internal/calc/calc.go",
    "target": "Add",
    "kind": "function",
    "line_range": "4-6",
    "gap_type": "branch",
    "uncovered_lines": [4, 5, 6],
    "goal": "为 Add 补充测试，覆盖未执行行段 4-6",
    "command": "go test ./...",
    "test_file": "internal/calc/calc_test.go",
    "test_name": "TestAddCoverage4_6",
    "confidence": 0.95
  },
  "generated": {
    "status": "ok",
    "test_file": "internal/calc/calc_test.go",
    "generated_cases": 1,
    "provider": "static"
  },
  "run_result": {
    "status": "pass",
    "framework": "go-test",
    "total": 1,
    "passed": 1,
    "failed": 0,
    "skipped": 0,
    "coverage_percent": 82.4,
    "failures": [],
    "raw_output": "ok example.com/project/internal/calc"
  },
  "metadata": {
    "framework": "go-test",
    "test_file": "internal/calc/calc_test.go"
  }
}
```

客户端下一步：

- 吸收生成测试或展示 diff 给用户确认。
- 继续处理下一个 `parse_coverage.test_tasks[]`。
- 如果需要严格验收，重新运行覆盖率并确认目标文件覆盖率提升。

## passed / manual_review_internal

含义：生成测试可以运行，但目标是未导出的内部符号、私有方法、无运行时代码或依赖特殊环境。`status=passed` 不代表可以把测试当作有效覆盖率补丁合入。

```json
{
  "status": "passed",
  "action": "manual_review_internal",
  "coverage_task": {
    "id": "vitest-2",
    "framework": "vitest",
    "file": "src/runtime/internal.ts",
    "target": "StorageManager.init",
    "kind": "method",
    "line_range": "18-27",
    "gap_type": "branch",
    "uncovered_lines": [21, 22, 23],
    "goal": "覆盖 StorageManager.init 的默认初始化分支",
    "command": "pnpm test -- --run",
    "test_file": "tests/runtime/internal.test.ts",
    "test_name": "StorageManager init manual review",
    "confidence": 0.7
  },
  "generated": {
    "status": "ok",
    "test_file": "tests/runtime/internal.test.ts",
    "generated_cases": 1,
    "provider": "static",
    "preview": "it.skip('manual_review_internal: StorageManager is not exported', () => {})"
  },
  "run_result": {
    "status": "pass",
    "framework": "vitest",
    "total": 1,
    "passed": 0,
    "failed": 0,
    "skipped": 1,
    "coverage_percent": 64.1,
    "failures": [],
    "raw_output": "Test Files 1 passed; Tests 1 skipped"
  },
  "metadata": {
    "framework": "vitest",
    "test_file": "tests/runtime/internal.test.ts",
    "internal_symbol": true,
    "internal_reason": "target is not exported from the module; cover it through a public API, test seam, or module-level integration test"
  }
}
```

客户端下一步：

- 不要继续对同一个 task 反复生成测试。
- 记录手审原因，寻找公共入口、测试 seam、依赖注入点或集成测试路径。
- 如果是 `manual_review_private`，优先看 `metadata.public_entry_candidates`。

## failed / apply_fix_suggestions

含义：测试已经生成，但运行失败；工具返回了可执行修复任务。Agent 应先读取 `run_result.fix_suggestions[].repair_task`，按限定文件和建议命令进入修复闭环。

```json
{
  "status": "failed",
  "action": "apply_fix_suggestions",
  "coverage_task": {
    "id": "pytest-1",
    "framework": "pytest",
    "file": "src/calc.py",
    "target": "divide",
    "kind": "function",
    "line_range": "8-10",
    "gap_type": "branch",
    "uncovered_lines": [9],
    "goal": "覆盖 divide 的除零错误分支",
    "command": "pytest",
    "test_file": "tests/test_calc.py",
    "test_name": "test_divide_zero",
    "confidence": 0.9
  },
  "generated": {
    "status": "ok",
    "test_file": "tests/test_calc.py",
    "generated_cases": 1,
    "provider": "static"
  },
  "run_result": {
    "status": "fail",
    "framework": "pytest",
    "total": 1,
    "passed": 0,
    "failed": 1,
    "skipped": 0,
    "coverage_percent": 71.0,
    "failures": [
      {
        "test_name": "test_divide_zero",
        "file": "tests/test_calc.py",
        "line": 12,
        "error": "AssertionError: assert 'division by zero' == 'zero division'"
      }
    ],
    "fix_suggestions": [
      {
        "file": "tests/test_calc.py",
        "line": 12,
        "issue": "断言期望值与实际错误信息不一致",
        "category": "assertion_mismatch",
        "suggested_fix": "根据真实异常信息调整测试断言",
        "confidence": 0.86,
        "repair_task": {
          "id": "repair-test-divide-zero",
          "test_name": "test_divide_zero",
          "category": "assertion_mismatch",
          "issue": "测试断言应匹配实际错误信息",
          "target_file": "tests/test_calc.py",
          "target_line": 12,
          "editable_files": ["tests/test_calc.py"],
          "suggested_commands": ["pytest tests/test_calc.py -q"],
          "assertion_focus": "断言 ZeroDivisionError 的真实 message"
        }
      }
    ],
    "raw_output": "FAILED tests/test_calc.py::test_divide_zero"
  },
  "metadata": {
    "framework": "pytest",
    "test_file": "tests/test_calc.py"
  }
}
```

客户端下一步：

- 优先执行 `repair_task`，不要把 `suggested_fix` 当补丁直接应用。
- 如果失败来自测试输入或断言，修生成测试；如果暴露真实业务 bug，再修实现。
- 修完后重新调用 `run_tests` 或再次调用 `validate_coverage_task` 验证闭环。

## failed / needs_better_input

含义：测试命令可能通过了，但覆盖率校验发现目标行没有命中。Agent 不应吸收该测试，应根据 `metadata.coverage_miss_reason`、`coverage_hit_lines` 和 `coverage_missed_lines` 选择更强输入或更合适的公共入口。

```json
{
  "status": "failed",
  "action": "needs_better_input",
  "coverage_task": {
    "id": "junit-3",
    "framework": "junit",
    "file": "src/main/java/com/example/StopWatch.java",
    "target": "StopWatch.getNanoTime",
    "kind": "method",
    "line_range": "210-214",
    "gap_type": "branch",
    "uncovered_lines": [212, 213],
    "goal": "覆盖 getNanoTime 的目标状态分支",
    "command": "mvn test",
    "test_file": "src/test/java/com/example/StopWatchTest.java",
    "test_name": "testGetNanoTimeTargetState",
    "confidence": 0.8
  },
  "generated": {
    "status": "ok",
    "test_file": "src/test/java/com/example/StopWatchTest.java",
    "generated_cases": 1,
    "provider": "static"
  },
  "run_result": {
    "status": "pass",
    "framework": "junit",
    "total": 1,
    "passed": 1,
    "failed": 0,
    "skipped": 0,
    "coverage_percent": 73.2,
    "failures": [],
    "raw_output": "BUILD SUCCESS"
  },
  "metadata": {
    "framework": "junit",
    "test_file": "src/test/java/com/example/StopWatchTest.java",
    "coverage_report": "target/site/jacoco/jacoco.xml",
    "coverage_target_lines": [210, 211, 212, 213, 214],
    "coverage_hit_lines": [210, 211],
    "coverage_missed_lines": [212, 213, 214],
    "coverage_target_hit": false,
    "coverage_miss_reason": "StopWatch.getNanoTime did not cover target line range 210-214; generate stronger inputs or cover the target through a better public entry point"
  }
}
```

客户端下一步：

- 不要因为 `run_result.status=pass` 就接受测试。
- 优先换输入、换状态构造方式，或改走更合适的公共入口。
- 如果源码分支确认不可达，应记录为人工复核，而不是继续生成同类弱测试。
