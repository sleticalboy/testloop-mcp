# 首跑诊断 CI 复制模板

这份模板面向希望在 GitHub Actions 中定期验证 testloop-mcp 接入状态的项目。它和 Onboarding CI 的区别是：首跑诊断会额外生成 `first-run-context.txt` 和 `first-run.log`，方便失败时直接把上下文交给 AI Agent。

生成的 artifact：

- `verification-report.md`：给人看的完整验收报告。
- `verification-summary.json`：给 Agent / CI 读取的结构化结果。
- `verification-summary.schema.json`：`verification-summary.json` 的结构契约。
- `agent-decision.txt`：最小下一步动作，核心字段是 `agent_next_step`。
- `first-run-context.txt`：可直接粘贴给 AI Agent 的最小上下文。
- `agent-response.txt`：由 `first-run-context.txt` 和可选 summary 渲染出的 Agent 四段回复草稿。
- `first-run.log`：底层 onboarding 命令完整输出。

bootstrap 在 helper checkout 支持时会自动运行 `sh scripts/verify-agent-artifact.sh first-run <output-dir>`，并在 GitHub step summary 写入 `Artifact verification`。如果 helper 固定到旧 tag 且没有 verifier，脚本只会给 warning，不影响已发布模板继续使用。

`TESTLOOP_MCP_VERSION` 控制安装的二进制版本；helper checkout 默认使用 `main`，这样当前 main 上新增的首跑诊断脚本可以搭配已发布的稳定二进制使用。需要固定 helper 版本时，再显式设置 `TESTLOOP_MCP_REPO_REF`。

## Go server 模板

```yaml
name: testloop first run

on:
  workflow_dispatch:
  pull_request:

jobs:
  first-run:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.24.x"

      - name: Generate first-run diagnostics
        run: |
          curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/run-first-run-ci.sh -o /tmp/testloop-first-run-ci.sh
          TESTLOOP_MCP_VERSION=v0.5.19 \
          TESTLOOP_FIRST_RUN_OUTPUT_DIR=/tmp/testloop-first-run \
          TESTLOOP_FIRST_RUN_PROJECT_DIR="$PWD" \
            bash /tmp/testloop-first-run-ci.sh 'go test ./...'

      - name: Upload first-run artifacts
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: testloop-first-run
          path: |
            /tmp/testloop-first-run/verification-report.md
            /tmp/testloop-first-run/verification-summary.json
            /tmp/testloop-first-run/verification-summary.schema.json
            /tmp/testloop-first-run/agent-decision.txt
            /tmp/testloop-first-run/first-run-context.txt
            /tmp/testloop-first-run/agent-response.txt
            /tmp/testloop-first-run/first-run.log
```

## Vue / Node 模板

```yaml
name: testloop web first run

on:
  workflow_dispatch:
  pull_request:

jobs:
  first-run:
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

      - name: Generate web first-run diagnostics
        run: |
          curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/run-first-run-ci.sh -o /tmp/testloop-first-run-ci.sh
          TESTLOOP_MCP_VERSION=v0.5.19 \
          TESTLOOP_FIRST_RUN_OUTPUT_DIR=/tmp/testloop-web-first-run \
          TESTLOOP_FIRST_RUN_PROJECT_DIR="$PWD" \
            bash /tmp/testloop-first-run-ci.sh 'pnpm install --frozen-lockfile && pnpm build'

      - name: Upload first-run artifacts
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: testloop-web-first-run
          path: |
            /tmp/testloop-web-first-run/verification-report.md
            /tmp/testloop-web-first-run/verification-summary.json
            /tmp/testloop-web-first-run/verification-summary.schema.json
            /tmp/testloop-web-first-run/agent-decision.txt
            /tmp/testloop-web-first-run/first-run-context.txt
            /tmp/testloop-web-first-run/agent-response.txt
            /tmp/testloop-web-first-run/first-run.log
```

## 失败时看什么

CI 失败时先打开 GitHub step summary。如果仍需要更多上下文，下载 artifact 后按顺序看：

1. `Artifact verification`：如果是 `passed`，说明目录必备文件、summary schema、decision 和 Agent response 已自检通过。
2. `agent-response.txt`：查看脚本已经渲染出的 Agent 四段回复草稿。
3. `first-run-context.txt`：旧版 artifact 没有回复草稿时，直接粘给 AI Agent。
4. `agent-decision.txt`：查看 `agent_next_step`，用于机器分流或复核。
5. `verification-summary.json`：查看失败 section。
6. `verification-report.md`：查看失败 section 的 stdout / stderr。
7. `first-run.log`：排查 bootstrap 或脚本入口问题。

下载 artifact 后也可以手动复跑：

```bash
sh scripts/verify-agent-artifact.sh first-run /tmp/testloop-first-run
```

首跑失败 action 的含义见 [首跑诊断失败样例](./first-run-failures.md)。如果不需要 `first-run-context.txt` 和完整日志，只想保留 onboarding 三件套，可以使用 [Onboarding CI 复制模板](./onboarding-ci-template.md)。

## 当前实跑记录

2026-07-18 使用当前仓库本地构建二进制完成两类 dry-run。

Go 项目：

```bash
go build -o /tmp/testloop-mcp-run-first-run-ci .
TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp-run-first-run-ci \
TESTLOOP_MCP_VERSION=v0.5.19 \
TESTLOOP_MCP_REPO_DIR="$PWD" \
TESTLOOP_FIRST_RUN_OUTPUT_DIR=/tmp/testloop-run-first-run-ci-go \
TESTLOOP_FIRST_RUN_PROJECT_DIR="$PWD" \
  scripts/run-first-run-ci.sh 'go test ./...'
```

结果：

- `first_run_status=passed`
- `first_run_failed_count=0`
- `first_run_agent_next_step=ready`
- `first_run_context=/tmp/testloop-run-first-run-ci-go/first-run-context.txt`
- `/tmp/testloop-run-first-run-ci-go/agent-response.txt` 包含 Agent 四段回复草稿。
- `agent_artifact_status=passed`

Node/web 项目：

```bash
TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp-run-first-run-ci \
TESTLOOP_MCP_VERSION=v0.5.19 \
TESTLOOP_MCP_REPO_DIR="$PWD" \
TESTLOOP_FIRST_RUN_OUTPUT_DIR=/tmp/testloop-run-first-run-ci-node \
TESTLOOP_FIRST_RUN_PROJECT_DIR=/tmp/testloop-run-first-run-ci-node-project \
  scripts/run-first-run-ci.sh 'pnpm install --frozen-lockfile && pnpm build'
```

结果同样为 `first_run_status=passed`、`first_run_failed_count=0`、`first_run_agent_next_step=ready`、`agent_artifact_status=passed`。
