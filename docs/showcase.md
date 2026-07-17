# 展示与验收路径

testloop-mcp 的展示路径分三层：默认 CI 保护基础契约，公开 showcase 适合演示外部项目闭环，真实项目 regression smoke 用于维护生成质量边界。不要把这三类验证混在一起，否则会让用户误以为所有外部项目验证都应该进入默认 CI。

## 快速选择

| 场景 | 命令 | 是否进默认 CI | 说明 |
| --- | --- | --- | --- |
| 演示首次接入全路径 | `scripts/showcase-onboarding.sh "$(command -v testloop-mcp)"` | 否，脚本入口回归 | 串联基础安装验收、真实 MCP 进程协议验收和最小 Agent 闭环 demo。 |
| 验证 Agent 最小闭环 | `go run ./examples/mcp-client-demo` | 是，脚本回归 | 不依赖外部项目，验证 `run_tests -> repair_task -> rerun -> parse_coverage` 和 `structuredContent` 消费路径。 |
| 演示公开 Go 项目 | `scripts/showcase-go-public-project.sh` | 否 | 克隆固定 commit 的 `google/uuid`，验证 `go-test-1`，展示 `passed/ready` 决策信号。 |
| 演示公开 JS/TS 项目 | `scripts/showcase-js-public-project.sh` | 否 | 克隆固定 commit 的 `unjs/ufo`，验证 `vitest-1,vitest-2`，展示 `ready` 与 `manual_review_internal` 分流。 |
| 维护跨语言质量边界 | `scripts/validate-regression-smoke.sh` | 否 | 复用本机真实项目和 JSONL 样本，覆盖 Java + JS + Python 的 ready / manual-review / external-service / database 等分类。 |

## 默认 CI 保护什么

默认 CI 只保护稳定、低网络依赖、可在普通 GitHub runner 上重复执行的路径：

- Go 单元测试和 e2e smoke。
- stdio / Streamable HTTP MCP 传输兼容性。
- 客户端配置生成和校验。
- 安装脚本、release 资产检查脚本、LLM provider 示例脚本。
- 最小 MCP 客户端 demo 输出回归。

这些路径用于证明仓库自身契约没有漂移，不用于证明外部项目和包管理器网络一定可达。

## 公开 showcase 证明什么

公开 showcase 是 opt-in，因为它们依赖 GitHub、npm registry 或外部项目测试环境。它们适合 README、录屏、演示和手动验收：

```bash
scripts/showcase-onboarding.sh "$(command -v testloop-mcp)"
scripts/showcase-go-public-project.sh
scripts/showcase-js-public-project.sh
```

Onboarding showcase 证明首次接入路径可以从安装验收走到 Agent 闭环。Go showcase 证明 `validate_coverage_task` 可以在外部 Go 项目上给出 `passed/ready` 决策信号。JS/TS showcase 证明 Agent 不应只看测试是否通过，还要读取 `action`：`ready` 可以进入下一个任务，`manual_review_internal` 应记录手审或寻找公共入口。

详细说明：

- [安装到 Agent 闭环展示路径](./showcase-onboarding.md)
- [Agent 闭环展示案例](./showcase-agent-loop.md)
- [公开 Go 项目覆盖率闭环案例](./showcase-public-go.md)
- [公开 JS/TS 项目覆盖率闭环案例](./showcase-public-js.md)

## 深度 regression smoke 证明什么

真实项目 regression smoke 面向维护者，不面向首次接入用户。它复用本机项目路径和历史 JSONL 样本，目的是守住生成质量边界：

```bash
TESTLOOP_VALIDATE_JS_STAGE_TIMEOUT_SECONDS=180 \
TESTLOOP_VALIDATE_PY_STAGE_TIMEOUT_SECONDS=180 \
scripts/validate-regression-smoke.sh
```

重点看 `status_counts` 和 `action_counts`：

- 真实 ready 样本应保持 `passed/ready`。
- 内部实现、无运行时代码、环境依赖、外部服务和数据库事务样本应保持对应 `manual_review_*` 分类。
- 历史失败修复样本不应退回普通 `repair_generated_test`。

详细说明见 [固定 smoke 回归说明](./regression-smoke.md) 和 [真实项目验证质量报告](./real-project-validation.md)。
