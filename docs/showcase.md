# 展示与验收路径

testloop-mcp 的展示路径分三层：默认 CI 保护基础契约，公开 showcase 适合演示外部项目闭环，真实项目 regression smoke 用于维护生成质量边界。不要把这三类验证混在一起，否则会让用户误以为所有外部项目验证都应该进入默认 CI。

## 快速选择

| 场景 | 命令 | 是否进默认 CI | 说明 |
| --- | --- | --- | --- |
| 接入方一页式验证 | `docs/adopter-verification-guide.md` | 是，文档回归 | 面向用户项目接入，把安装、首跑、CI bootstrap、artifact 和失败分流压成一页执行清单。 |
| 安装后首跑诊断 | `scripts/doctor-first-run.sh "$(command -v testloop-mcp)"` | 否，脚本入口和文档回归 | 聚合安装验收、真实 MCP transport、最小 Agent demo、可选用户项目 smoke，并输出稳定 artifact 路径、可粘贴上下文和 `first_run_agent_next_step`。 |
| 首跑诊断 CI 模板 | `scripts/run-first-run-ci.sh 'go test ./...'` | 否，脚本入口和文档回归 | 面向外部用户项目 CI，自动准备 helper checkout 并上传 report、summary、decision、context、log。 |
| 渲染 first-run Agent 回复 | `sh scripts/render-first-run-agent-response.sh /tmp/testloop-first-run` | 是，脚本回归 | 从 first-run artifact 目录自动读取 context 和 summary，输出 Agent 可直接回复用户的四段结构。 |
| 演练外部项目首跑诊断 CI | `scripts/showcase-first-run-ci-external-project.sh` | 否，脚本入口回归 | 在 `/tmp` 创建非 testloop Go 或 Node 项目，从该项目目录运行 first-run bootstrap，验证复制路径能产出六件套 artifact。 |
| 演示首次接入全路径 | `scripts/showcase-onboarding.sh "$(command -v testloop-mcp)"` | 否，脚本入口回归 | 串联基础安装验收、真实 MCP 进程协议验收和最小 Agent 闭环 demo，只看终端输出。 |
| 生成 Agent onboarding 演示制品 | `scripts/showcase-agent-onboarding-report.sh "$(command -v testloop-mcp)"` | 否，脚本入口回归 | 在完整首次接入路径基础上输出 Markdown、summary JSON 和 `agent_next_step` 决策文本。 |
| 渲染 onboarding Agent 回复 | `sh scripts/render-onboarding-agent-response.sh /tmp/testloop-onboarding` | 是，脚本回归 | 从 onboarding artifact 目录自动读取 summary，输出 Agent 可直接回复用户的四段结构。 |
| 演练外部项目 Onboarding CI | `scripts/showcase-onboarding-ci-external-project.sh` | 否，脚本入口回归 | 在 `/tmp` 创建非 testloop Go 或 Node 项目，从该项目目录运行 bootstrap，验证复制路径能产出四件套 artifact。 |
| 生成用户项目验收报告 | `scripts/generate-verification-report.sh "$(command -v testloop-mcp)" /tmp/testloop-report.md` | 否，脚本入口回归 | 聚合基础安装验收、真实 MCP 协议 smoke、最小 Agent demo，可 opt-in 公开 showcase 和用户项目 smoke。 |
| 生成通用双项目报告 | `scripts/showcase-dual-project-report.sh "$(command -v testloop-mcp)"` | 否，脚本入口回归 | 提供任意两条 user-project smoke 的通用底座，输出两份报告和嵌套子 summary 的总状态文件。 |
| 生成 laoxia 双栈验收报告 | `scripts/showcase-laoxia-scaffold-report.sh "$(command -v testloop-mcp)"` | 否，脚本入口回归 | 一次生成 `car-admin-server` 和 `car-admin-web` 两份验收报告，以及一份嵌套子 summary 的总 `laoxia-summary.json`，适合作为真实项目双栈 smoke 的固定入口。 |
| 生成 QuickSmoke Go/Java 双项目报告 | `TESTLOOP_PAIR_PREFIX=quicksmoke ... scripts/showcase-dual-project-report.sh /tmp/testloop-mcp` | 否，脚本入口回归 | 用干净的 Go 与 Java 项目验证 shared helper 的跨语言复用性，输出 `quicksmoke-summary.json` 及两份子 summary。 |
| 验证 Agent 最小闭环 | `go run ./examples/mcp-client-demo` | 是，脚本回归 | 不依赖外部项目，验证 `run_tests -> repair_task -> rerun -> parse_coverage` 和 `structuredContent` 消费路径。 |
| 演示公开 Go 项目 | `scripts/showcase-go-public-project.sh` | 否 | 克隆固定 commit 的 `google/uuid`，验证 `go-test-1`，并断言 `passed/ready` 决策信号。 |
| 演示公开 JS/TS 项目 | `scripts/showcase-js-public-project.sh` | 否 | 克隆固定 commit 的 `unjs/ufo`，验证 `vitest-1,vitest-2`，并断言 `ready` 与 `manual_review_internal` 分流。 |
| 维护跨语言质量边界 | `scripts/validate-regression-smoke.sh` | 否 | 复用本机真实项目和仓库内静态 JSONL 样本，覆盖 Java + JS + Python 的 ready / manual-review / external-service / database 等分类。 |

## 默认 CI 保护什么

默认 CI 只保护稳定、低网络依赖、可在普通 GitHub runner 上重复执行的路径：

- Go 单元测试和 e2e smoke。
- stdio / Streamable HTTP MCP 传输兼容性。
- 客户端配置生成和校验。
- 安装脚本、验收报告脚本、release 资产检查脚本、LLM provider 示例脚本。
- 最小 MCP 客户端 demo 输出回归。

这些路径用于证明仓库自身契约没有漂移，不用于证明外部项目和包管理器网络一定可达。

## 验收报告证明什么

验收报告面向接入方和维护者，目标是把一次本机验收沉淀成可复制的 Markdown 制品：

```bash
scripts/generate-verification-report.sh "$(command -v testloop-mcp)" /tmp/testloop-report.md
```

默认报告不访问公网，只执行基础安装验收、真实 MCP 协议 smoke 和最小 Agent 闭环 demo。公开 showcase 通过 `TESTLOOP_REPORT_PUBLIC_SHOWCASES=go|js|all` 显式开启；用户自己的 server / web / CLI 项目 smoke 通过 `TESTLOOP_REPORT_PROJECT_DIR` 和 `TESTLOOP_REPORT_PROJECT_COMMAND` 显式传入。

详细说明见 [用户项目验收报告](./verification-report.md)。

## 公开 showcase 证明什么

公开 showcase 是 opt-in，因为它们依赖 GitHub、npm registry 或外部项目测试环境。它们适合 README、录屏、演示和手动验收：

```bash
scripts/showcase-onboarding.sh "$(command -v testloop-mcp)"
scripts/doctor-first-run.sh "$(command -v testloop-mcp)"
scripts/run-first-run-ci.sh 'go test ./...'
sh scripts/render-first-run-agent-response.sh /tmp/testloop-first-run
scripts/showcase-first-run-ci-external-project.sh
scripts/showcase-agent-onboarding-report.sh "$(command -v testloop-mcp)"
sh scripts/render-onboarding-agent-response.sh /tmp/testloop-onboarding
scripts/showcase-onboarding-ci-external-project.sh
scripts/showcase-go-public-project.sh
scripts/showcase-js-public-project.sh
```

Onboarding showcase 证明首次接入路径可以从安装验收走到 Agent 闭环。`doctor-first-run.sh` 是更适合安装后直接运行的入口：它复用 onboarding report，但会稳定输出 `first_run_status`、`first_run_agent_next_step`、可粘贴上下文和 artifact 路径。`run-first-run-ci.sh` 是外部用户项目 CI 的 bootstrap 入口，会准备 testloop-mcp helper checkout 并上传首跑诊断六件套；`showcase-first-run-ci-external-project.sh` 则从非 testloop 项目目录验证这条复制路径确实能产出 report、summary、decision、context、Agent response 和 log，默认跑 Go，也可以用 `TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE=node|all` 验证 web/Node 命令形态。`showcase-agent-onboarding-report.sh` 适合公开录屏、接入方验收和 CI artifact：它复用验收报告脚本生成 Markdown 与 summary JSON，再运行 summary 决策 demo，把 `agent_next_step` 单独落盘；`render-onboarding-agent-response.sh` 可继续把 onboarding artifact 目录转成 Agent 四段回复。`showcase-onboarding-ci-external-project.sh` 证明 onboarding bootstrap 在非 testloop 项目目录中也能生成 report、summary、decision 和 Agent response 四件套，适合发布后或改 onboarding 模板时手动复验；默认跑 Go，也可以用 `TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE=node|all` 验证 web/Node 命令形态。`showcase-dual-project-report.sh` 提供任意两条 user-project smoke 的通用底座；`showcase-laoxia-scaffold-report.sh` 只是带默认值的 thin wrapper。`QuickSmoke` Go/Java 复验说明这条 shared helper 不只适用于同类项目，也能把跨语言 pair 收成统一 summary。Go showcase 证明 `validate_coverage_task` 可以在外部 Go 项目上给出 `passed/ready` 决策信号，并默认校验该信号不漂移。JS/TS showcase 证明 Agent 不应只看测试是否通过，还要读取 `action`：`ready` 可以进入下一个任务，`manual_review_internal` 应记录手审或寻找公共入口；脚本也会默认校验这两个 action。公开项目 showcase 都支持通过 `TESTLOOP_SHOWCASE_*_PROJECT_DIR` 复用本地 checkout，减少外网 clone 对演示的影响；远端 clone/fetch 默认 60 秒超时，可通过 `TESTLOOP_SHOWCASE_*_GIT_TIMEOUT` 调整。

公开 showcase 的 JSONL 明细默认写入 `/tmp` 或用户指定路径，不提交到仓库。脚本会通过 `scripts/summarize-showcase-output.py` 输出精简 `showcase_summary=...` 并执行 action 断言；文档只归档这类 summary 和关键任务摘要。

详细说明：

- [接入方一页式验证指南](./adopter-verification-guide.md)
- [安装到 Agent 闭环展示路径](./showcase-onboarding.md)
- [首跑诊断](./first-run-diagnostics.md)
- [首跑诊断 CI 复制模板](./first-run-ci-template.md)
- [首跑诊断 CI 外部项目演练](./first-run-ci-external-dry-run.md)
- [首跑诊断失败样例](./first-run-failures.md)
- [Agent 闭环展示案例](./showcase-agent-loop.md)
- [用户项目验收报告](./verification-report.md)
- [Onboarding CI 外部项目演练](./onboarding-ci-external-dry-run.md)
- [Onboarding CI 复制模板](./onboarding-ci-template.md)
- [Onboarding CI 失败排查](./onboarding-ci-failure-triage.md)
- [真实接入案例模板](./real-integration-cases.md)
- [验收 Summary 失败分流样例](./verification-summary-failures.md)
- [公开 Go 项目覆盖率闭环案例](./showcase-public-go.md)
- [公开 JS/TS 项目覆盖率闭环案例](./showcase-public-js.md)

## 深度 regression smoke 证明什么

真实项目 regression smoke 面向维护者，不面向首次接入用户。它复用本机项目路径和仓库内静态 JSONL 样本，目的是守住生成质量边界：

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
