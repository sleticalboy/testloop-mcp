# 验收报告 CI 集成

这份文档说明如何在 GitHub Actions 中生成 testloop-mcp 的 Markdown + JSON 验收报告，并让 Agent / CI 根据 summary JSON 做下一步分流。

目标不是替代项目自己的测试流水线，而是给接入方一个固定反馈入口。如果只需要最短可复制版本，先看 [Onboarding CI 复制模板](./onboarding-ci-template.md)。如果希望失败时额外上传可粘贴给 AI Agent 的 `first-run-context.txt` 和完整日志，使用 [首跑诊断 CI 复制模板](./first-run-ci-template.md)。

- Markdown 报告给人看，适合上传 artifact 或贴到 issue / release checklist。
- summary JSON 给 Agent / CI 看，适合判断失败归因和下一步动作。
- 用户项目 smoke 命令由调用方显式传入，避免脚本猜测数据库、外部服务或前端构建环境。

## 怎么选入口

首次接入、刚安装完、或者希望失败时直接把上下文交给 AI Agent，优先用 `scripts/run-first-run-ci.sh`。它在 onboarding 三件套之外额外生成 `first-run-context.txt` 和 `first-run.log`，更适合排查安装、MCP transport、Agent demo 和用户项目 smoke 的综合问题。

已经稳定接入，只想在 PR 或发布后确认当前项目 smoke 和 testloop-mcp 自检是否通过，使用 `scripts/run-onboarding-ci.sh`。它输出 Markdown、summary JSON 和 decision，artifact 更少，适合作为持续验收入口。

维护者改 onboarding 或 first-run 模板后，使用外部项目演练脚本复验复制路径：

```bash
scripts/showcase-onboarding-ci-external-project.sh
scripts/showcase-first-run-ci-external-project.sh
```

这两条演练会从非 testloop 项目目录执行下载版 bootstrap，防止模板不小心依赖仓库内路径。

## 推荐 workflow

如果只是在用户项目 CI 中接入，优先使用 `scripts/run-onboarding-ci.sh` bootstrap。它会安装或解析 `testloop-mcp`，准备报告脚本，再同时生成 Markdown、summary JSON 和 decision 输出，减少手写路径和决策命令。

下面示例适合直接复制到用户项目。更短的 Go / Vue 模板见 [Onboarding CI 复制模板](./onboarding-ci-template.md)。

```yaml
name: testloop verification

on:
  workflow_dispatch:
  pull_request:

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.24.x"

      - name: Generate onboarding report
        run: |
          curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/run-onboarding-ci.sh -o /tmp/testloop-onboarding-ci.sh
          TESTLOOP_MCP_VERSION=v0.5.8 \
          TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-onboarding \
          TESTLOOP_ONBOARDING_PROJECT_DIR="$PWD" \
            bash /tmp/testloop-onboarding-ci.sh 'go test ./...'

      - name: Upload verification report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: testloop-verification-report
          path: |
            /tmp/testloop-onboarding/verification-report.md
            /tmp/testloop-onboarding/verification-summary.json
            /tmp/testloop-onboarding/agent-decision.txt
```

这段 workflow 的关键点：

- `scripts/showcase-agent-onboarding-report.sh`：失败时也会尽量保留 Markdown / JSON / decision 输出。
- `scripts/run-onboarding-ci.sh`：适合外部用户项目，负责准备 testloop-mcp 二进制和报告脚本。
- `if: always()`：无论验收通过还是失败，都上传 Markdown 和 JSON。
- `project-smoke-command`：接入方显式指定自己的 smoke 命令。

## 高级 workflow

如果需要完全控制 Markdown、summary JSON 和 decision demo 的路径，可以直接调用底层验收报告脚本：

```yaml
- name: Generate verification report
  run: |
    set +e
    TESTLOOP_REPORT_EXPECT_VERSION=0.5.8 \
    TESTLOOP_REPORT_SUMMARY_JSON=/tmp/testloop-summary.json \
    TESTLOOP_REPORT_PROJECT_DIR="$PWD" \
    TESTLOOP_REPORT_PROJECT_COMMAND='go test ./...' \
      scripts/generate-verification-report.sh /tmp/testloop-mcp /tmp/testloop-report.md
    code=$?

    go run ./examples/verification-summary-decision-demo /tmp/testloop-summary.json
    exit "$code"
```

高级用法的关键点：

- `set +e`：验收失败时仍继续运行决策 demo，并上传报告。
- `code=$?` 和 `exit "$code"`：最后保留原始验收结果，CI 仍会正确失败。

## 前端项目示例

Vue / React / Node 项目可以把用户项目 smoke 换成包管理器命令。仓库有 `pnpm-lock.yaml` 时优先用 pnpm：

```yaml
- uses: pnpm/action-setup@v4
  with:
    version: 10

- uses: actions/setup-node@v4
  with:
    node-version: 22
    cache: pnpm

- name: Generate web verification report
  run: |
    curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/run-onboarding-ci.sh -o /tmp/testloop-onboarding-ci.sh
    TESTLOOP_MCP_VERSION=v0.5.8 \
    TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-web-onboarding \
    TESTLOOP_ONBOARDING_PROJECT_DIR="$PWD" \
      bash /tmp/testloop-onboarding-ci.sh 'pnpm install --frozen-lockfile && pnpm build'
```

## Agent 分流建议

CI 日志里优先看 `agent_next_step`：

| action | 说明 |
| --- | --- |
| `ready` | testloop-mcp 自检和用户项目 smoke 均通过，可以进入下一步任务。 |
| `fix-installation` | 优先修二进制路径、版本、客户端配置 roundtrip 或 HTTP `/healthz`。 |
| `inspect-mcp-transport` | 优先排查 stdio / Streamable HTTP MCP 客户端启动。 |
| `inspect-agent-demo` | 优先排查结构化反馈闭环和 demo runner。 |
| `inspect-showcase` | 优先排查外部网络、本地 checkout 或 action 期望。 |
| `inspect-user-project` | 优先查看用户项目 smoke 输出，通常是依赖、环境变量、测试命令或项目自身失败。 |

失败时不要只看 CI 最后一行。应下载 `testloop-verification-report` artifact，先读 summary JSON 的 failed section，再打开 Markdown 对应明细。

更具体的失败排查顺序见 [Onboarding CI 失败排查](./onboarding-ci-failure-triage.md)。

首跑诊断 CI 的 artifact 多包含 `first-run-context.txt` 和 `first-run.log`，适合失败时直接把上下文交给 AI Agent。复制模板见 [首跑诊断 CI 复制模板](./first-run-ci-template.md)。

## 适用边界

这条 CI 示例适合接入验收和发布后 smoke，不适合把所有公开 showcase 或真实项目 top-N regression 全部塞进默认 PR CI。公开 showcase 依赖 GitHub、npm registry 和外部项目状态，应继续使用 `TESTLOOP_REPORT_PUBLIC_SHOWCASES=go|js|all` 显式开启；真实项目 top-N 生成质量回归继续走 [固定 smoke 回归说明](./regression-smoke.md)。
