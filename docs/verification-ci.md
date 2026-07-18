# 验收报告 CI 集成

这份文档说明如何在 GitHub Actions 中生成 testloop-mcp 的 Markdown + JSON 验收报告，并让 Agent / CI 根据 summary JSON 做下一步分流。

目标不是替代项目自己的测试流水线，而是给接入方一个固定反馈入口：

- Markdown 报告给人看，适合上传 artifact 或贴到 issue / release checklist。
- summary JSON 给 Agent / CI 看，适合判断失败归因和下一步动作。
- 用户项目 smoke 命令由调用方显式传入，避免脚本猜测数据库、外部服务或前端构建环境。

## 推荐 workflow

如果只是想把首次接入验收和用户项目 smoke 汇总成可上传制品，优先使用 `scripts/showcase-agent-onboarding-report.sh`。它会同时生成 Markdown、summary JSON 和 decision 输出，减少手写路径和决策命令。

下面示例假设当前仓库已经能通过 `go install` 或 release asset 安装 `testloop-mcp`。如果是在 testloop-mcp 源码仓库内验证，也可以先 `go build -o /tmp/testloop-mcp .`，再把二进制路径传给脚本。

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

      - name: Build testloop-mcp
        run: go build -o /tmp/testloop-mcp .

      - name: Generate onboarding report
        run: |
          TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.5 \
          TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-onboarding \
          TESTLOOP_REPORT_PROJECT_DIR="$PWD" \
          TESTLOOP_REPORT_PROJECT_COMMAND='go test ./...' \
            scripts/showcase-agent-onboarding-report.sh /tmp/testloop-mcp

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
- `if: always()`：无论验收通过还是失败，都上传 Markdown 和 JSON。
- `TESTLOOP_REPORT_PROJECT_COMMAND`：接入方显式指定自己的 smoke 命令。

## 高级 workflow

如果需要完全控制 Markdown、summary JSON 和 decision demo 的路径，可以直接调用底层验收报告脚本：

```yaml
- name: Generate verification report
  run: |
    set +e
    TESTLOOP_REPORT_EXPECT_VERSION=0.5.5 \
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
    TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-web-onboarding \
    TESTLOOP_REPORT_PROJECT_DIR="$PWD" \
    TESTLOOP_REPORT_PROJECT_COMMAND='pnpm install --frozen-lockfile && pnpm build' \
      scripts/showcase-agent-onboarding-report.sh /tmp/testloop-mcp
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

## 适用边界

这条 CI 示例适合接入验收和发布后 smoke，不适合把所有公开 showcase 或真实项目 top-N regression 全部塞进默认 PR CI。公开 showcase 依赖 GitHub、npm registry 和外部项目状态，应继续使用 `TESTLOOP_REPORT_PUBLIC_SHOWCASES=go|js|all` 显式开启；真实项目 top-N 生成质量回归继续走 [固定 smoke 回归说明](./regression-smoke.md)。
