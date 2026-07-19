# Onboarding CI 复制模板

这份模板面向第一次把 `testloop-mcp` 接入真实项目的团队。目标是少解释、少分支，让接入方复制后只改项目 smoke 命令，就能在 GitHub Actions 里拿到四类制品：

- `verification-report.md`：给人看的完整验收报告。
- `verification-summary.json`：给 Agent / CI 读取的结构化结果。
- `agent-decision.txt`：最小下一步动作，核心字段是 `agent_next_step`。
- `agent-response.txt`：按 summary 渲染出的 Agent 四段回复草稿。

## Go server 模板

适合 Go API、CLI、server 项目。接入方通常只需要改 `TESTLOOP_REPORT_PROJECT_COMMAND`。

```yaml
name: testloop onboarding

on:
  workflow_dispatch:
  pull_request:

jobs:
  onboarding:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.24.x"

      - name: Generate onboarding report
        run: |
          curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/run-onboarding-ci.sh -o /tmp/testloop-onboarding-ci.sh
          TESTLOOP_MCP_VERSION=v0.5.9 \
          TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-onboarding \
          TESTLOOP_ONBOARDING_TITLE='Go server testloop onboarding' \
          TESTLOOP_ONBOARDING_PROJECT_DIR="$PWD" \
            bash /tmp/testloop-onboarding-ci.sh 'go test ./...'

      - name: Upload onboarding artifacts
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: testloop-onboarding
          path: |
            /tmp/testloop-onboarding/verification-report.md
            /tmp/testloop-onboarding/verification-summary.json
            /tmp/testloop-onboarding/agent-decision.txt
            /tmp/testloop-onboarding/agent-response.txt
```

## Vue / Node 模板

适合 Vue、React、Node CLI 或库项目。仓库有 `pnpm-lock.yaml` 时优先使用 pnpm。

```yaml
name: testloop web onboarding

on:
  workflow_dispatch:
  pull_request:

jobs:
  onboarding:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.24.x"

      - uses: pnpm/action-setup@v4
        with:
          version: 10

      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: pnpm

      - name: Generate onboarding report
        run: |
          curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/run-onboarding-ci.sh -o /tmp/testloop-onboarding-ci.sh
          TESTLOOP_MCP_VERSION=v0.5.9 \
          TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-web-onboarding \
          TESTLOOP_ONBOARDING_TITLE='Vue web testloop onboarding' \
          TESTLOOP_ONBOARDING_PROJECT_DIR="$PWD" \
            bash /tmp/testloop-onboarding-ci.sh 'pnpm install --frozen-lockfile && pnpm build'

      - name: Upload onboarding artifacts
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: testloop-web-onboarding
          path: |
            /tmp/testloop-web-onboarding/verification-report.md
            /tmp/testloop-web-onboarding/verification-summary.json
            /tmp/testloop-web-onboarding/agent-decision.txt
            /tmp/testloop-web-onboarding/agent-response.txt
```

## 接入后看什么

CI 失败时不要只看最后一行日志。先下载 artifact，再按这个顺序看：

1. `agent-response.txt`：先看脚本已经渲染出的 Agent 四段回复草稿。
2. `agent-decision.txt`：如果 `agent_next_step=ready`，说明 testloop-mcp 自检和用户项目 smoke 都通过。
3. `verification-summary.json`：看 `failed_count` 和失败 section 的 `name/status/exit_code`。
4. `verification-report.md`：看失败 section 的 stdout / stderr 明细。

常见 `agent_next_step`：

| action | 下一步 |
| --- | --- |
| `ready` | 进入真实生成、补测、修复或覆盖率闭环。 |
| `fix-installation` | 先修二进制路径、版本漂移、配置 roundtrip 或 HTTP `/healthz`。 |
| `inspect-mcp-transport` | 先查 stdio / Streamable HTTP MCP 启动和客户端传输配置。 |
| `inspect-agent-demo` | 先查结构化返回、demo runner 和仓库自身构建。 |
| `inspect-user-project` | 先查项目自己的测试/构建命令、依赖、环境变量和日志。 |

更完整的 CI 说明见 [验收报告 CI 集成](./verification-ci.md)，真实 server / web 接入记录见 [真实接入案例模板](./real-integration-cases.md)。
