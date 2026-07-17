# 公开 Go 项目覆盖率闭环案例

这个案例用于展示 testloop-mcp 在外部公开项目上的 coverage task 闭环。它默认使用 [google/uuid](https://github.com/google/uuid) 的固定 commit `2d3c2a9cc518326daf99a383f07c4d3c44317e4d`，只验证一个明确的覆盖率任务，避免公开演示被外部项目后续变更或网络波动放大。

## 运行方式

```bash
scripts/showcase-go-public-project.sh
```

脚本会执行三步：

1. 克隆公开 Go 项目到临时目录，并 checkout 固定 commit。
2. 调用 `scripts/validate-go-coverage-top-tasks.sh`，用 `TESTLOOP_VALIDATE_GO_TASK_IDS=go-test-1` 精确筛选目标任务。
3. 输出 JSONL 结果，并打印 `showcase_summary=...` 摘要。
4. 校验默认期望 `go-test-1=ready`，如果公开案例的决策信号漂移会直接失败。

默认输出文件：

```text
/tmp/testloop-google-uuid-showcase.jsonl
```

## 当前验证结果

本地验证命令：

```bash
TESTLOOP_VALIDATE_GO_TASK_IDS=go-test-1 \
  scripts/validate-go-coverage-top-tasks.sh /tmp/testloop-showcase-google-uuid /tmp/testloop-google-uuid-task1.jsonl
```

当前结果：

```json
{
  "status_counts": {"passed": 1},
  "action_counts": {"ready": 1},
  "task": "go-test-1 clockSequence 87-90",
  "run_result.skipped": 1,
  "showcase_expectations": "pass"
}
```

这个结果说明 Agent 可以把公开 Go 项目的覆盖率缺口转成单个 `validate_coverage_task`，并收到稳定的 `status=passed` / `action=ready` 决策信号。脚本仍保留 JSONL 明细，便于继续检查 `coverage_task`、`generated`、`run_result` 和 `metadata`。

## 可配置项

```bash
TESTLOOP_SHOWCASE_GO_REPO=https://github.com/google/uuid.git \
TESTLOOP_SHOWCASE_GO_REF=2d3c2a9cc518326daf99a383f07c4d3c44317e4d \
TESTLOOP_SHOWCASE_GO_TASK_IDS=go-test-1 \
TESTLOOP_SHOWCASE_GO_EXPECT_ACTIONS=go-test-1=ready \
TESTLOOP_SHOWCASE_GO_OUTPUT=/tmp/testloop-google-uuid-showcase.jsonl \
scripts/showcase-go-public-project.sh
```

如果只想打印摘要、不做期望断言，可以把 `TESTLOOP_SHOWCASE_GO_EXPECT_ACTIONS` 设为空字符串。

`scripts/validate-go-coverage-top-tasks.sh` 也支持直接复用已有任务文件：

```bash
TESTLOOP_VALIDATE_GO_TASKS_FILE=/tmp/tasks.jsonl \
TESTLOOP_VALIDATE_GO_TASK_IDS=go-test-1 \
scripts/validate-go-coverage-top-tasks.sh /path/to/go/project /tmp/result.jsonl
```

## 边界说明

这个 showcase 是 opt-in 脚本，不进入默认 CI，因为它依赖 GitHub 网络和外部仓库可达性。默认 CI 仍通过仓库内 fixture、MCP 传输 smoke、客户端配置 smoke 和最小 Agent demo 回归保护稳定性。

该案例的重点不是宣称静态生成器已经能理解 `google/uuid` 的全部业务语义，而是展示真实公开项目中 `parse_coverage -> validate_coverage_task -> run_tests` 的结构化反馈链路可以被 Agent 稳定消费。
