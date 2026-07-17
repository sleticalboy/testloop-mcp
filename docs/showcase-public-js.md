# 公开 JS/TS 项目覆盖率闭环案例

这个案例用于展示 testloop-mcp 在外部公开 JS/TS 项目上的 coverage task 闭环。它默认使用 [unjs/ufo](https://github.com/unjs/ufo) 的固定 commit `f06c800d0c59f2a4a1b9ba65eb6cb61a84419be6`，通过 Vitest coverage 生成 `coverage-final.json`，再把选中的覆盖率任务交给 `validate_coverage_task`。

## 运行方式

```bash
scripts/showcase-js-public-project.sh
```

脚本会执行四步：

1. 克隆公开 JS/TS 项目到临时目录，并 checkout 固定 commit。
2. 使用 `pnpm install --frozen-lockfile` 安装依赖。
3. 调用 `scripts/validate-js-coverage-top-tasks.sh`，用 `TESTLOOP_VALIDATE_JS_TASK_IDS=vitest-1,vitest-2` 精确筛选目标任务。
4. 输出 JSONL 结果，并打印 `showcase_summary=...` 摘要。
5. 校验默认期望 `vitest-1=manual_review_internal,vitest-2=ready`，如果公开案例的决策信号漂移会直接失败。

默认输出文件：

```text
/tmp/testloop-ufo-showcase.jsonl
```

## 当前验证结果

本地验证命令：

```bash
TESTLOOP_VALIDATE_JS_STAGE_TIMEOUT_SECONDS=180 \
  scripts/validate-js-coverage-top-tasks.sh /tmp/testloop-showcase-ufo vitest 5 /tmp/testloop-ufo-vitest-top5.jsonl
```

top5 结果：

```json
{
  "status_counts": {"passed": 5},
  "action_counts": {"manual_review_internal": 1, "ready": 4},
  "zero_skip": 4,
  "skipped_total": 1
}
```

默认 showcase 精确验证前两个任务：

```json
{
  "tasks": [
    {"id": "vitest-1", "target": "s", "status": "passed", "action": "manual_review_internal"},
    {"id": "vitest-2", "target": "resolveURL", "status": "passed", "action": "ready"}
  ],
  "showcase_expectations": "pass"
}
```

这个结果比单纯 `passed/ready` 更适合展示 Agent 决策：`manual_review_internal` 表示目标是内部符号或不可直接从外部测试稳定构造，Agent 应记录手审或寻找公共入口；`ready` 表示生成测试已经通过，可以进入下一个 coverage task。

## 可配置项

```bash
TESTLOOP_SHOWCASE_JS_REPO=https://github.com/unjs/ufo.git \
TESTLOOP_SHOWCASE_JS_REF=f06c800d0c59f2a4a1b9ba65eb6cb61a84419be6 \
TESTLOOP_SHOWCASE_JS_TASK_IDS=vitest-1,vitest-2 \
TESTLOOP_SHOWCASE_JS_EXPECT_ACTIONS=vitest-1=manual_review_internal,vitest-2=ready \
TESTLOOP_SHOWCASE_JS_OUTPUT=/tmp/testloop-ufo-showcase.jsonl \
TESTLOOP_SHOWCASE_JS_TIMEOUT=180 \
scripts/showcase-js-public-project.sh
```

如果只想打印摘要、不做期望断言，可以把 `TESTLOOP_SHOWCASE_JS_EXPECT_ACTIONS` 设为空字符串。

如果已经有本地 checkout 和依赖，也可以直接运行底层验证脚本：

```bash
TESTLOOP_VALIDATE_JS_TASK_IDS=vitest-1,vitest-2 \
TESTLOOP_VALIDATE_JS_STAGE_TIMEOUT_SECONDS=180 \
scripts/validate-js-coverage-top-tasks.sh /path/to/ufo vitest /tmp/testloop-ufo-showcase.jsonl
```

## 边界说明

这个 showcase 是 opt-in 脚本，不进入默认 CI，因为它依赖 GitHub 网络、npm registry 和外部仓库可达性。默认 CI 仍通过仓库内 fixture、MCP 传输 smoke、客户端配置 smoke 和最小 Agent demo 回归保护稳定性。

该案例的重点不是宣传静态生成器可以完整理解 `ufo` 的全部 URL 语义，而是展示真实公开 TS/Vitest 项目中，Agent 可以直接消费 `validate_coverage_task.status/action`，并区分可吸收的 ready 任务和应手审的内部实现任务。
