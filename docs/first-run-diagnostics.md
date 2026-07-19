# 首跑诊断

`scripts/doctor-first-run.sh` 用于把安装后第一次验收收敛成一条命令。它会复用 onboarding report 流程，生成 Markdown 报告、summary JSON、Agent decision、可粘贴上下文和完整日志，并在终端输出稳定字段。

## 快速使用

```bash
scripts/doctor-first-run.sh "$(command -v testloop-mcp)"
```

建议发布后或 Homebrew 安装后加版本门禁：

```bash
TESTLOOP_FIRST_RUN_EXPECT_VERSION=0.5.9 \
  scripts/doctor-first-run.sh "$(command -v testloop-mcp)"
```

如果要把用户项目 smoke 一起纳入诊断：

```bash
TESTLOOP_FIRST_RUN_PROJECT_DIR=/path/to/project \
TESTLOOP_FIRST_RUN_PROJECT_COMMAND='go test ./...' \
  scripts/doctor-first-run.sh "$(command -v testloop-mcp)"
```

## 输出字段

脚本无论成功还是失败，都会尽量输出这些字段：

```text
first_run_status=passed
first_run_failed_count=0
first_run_agent_next_step=ready
first_run_report=/tmp/testloop-mcp-first-run/verification-report.md
first_run_summary_json=/tmp/testloop-mcp-first-run/verification-summary.json
first_run_decision=/tmp/testloop-mcp-first-run/agent-decision.txt
first_run_context=/tmp/testloop-mcp-first-run/first-run-context.txt
first_run_log=/tmp/testloop-mcp-first-run/first-run.log
```

其中：

- `first_run_status` 来自 summary JSON 的 `overall_status`。
- `first_run_agent_next_step` 来自 `agent-decision.txt`，用于告诉用户或 AI Agent 下一步做什么。
- `first_run_context` 是可直接粘给 AI Agent 的最小上下文。
- `first_run_log` 保存底层 onboarding 命令输出，便于排查脚本入口问题。

## 诊断边界

这条路径不新增独立诊断逻辑，而是编排已有能力：

- 基础安装验收：二进制、版本、`--doctor-config`、配置 roundtrip 和 HTTP `/healthz`。
- 真实 MCP 协议 smoke：stdio / Streamable HTTP 启动和轻量工具调用。
- 最小 Agent 闭环 demo：`run_tests -> repair_task -> rerun -> parse_coverage`。
- 可选用户项目 smoke：由调用方显式传入命令。

如果 `first_run_agent_next_step=ready`，说明安装和 MCP 传输链路已经可用，可以继续配置 Codex、Claude 或 Cursor，或进入真实项目验证。其他 action 的排查含义见 [首跑诊断失败样例](./first-run-failures.md)、[用户项目验收报告](./verification-report.md) 和 [Onboarding CI 失败排查](./onboarding-ci-failure-triage.md)。

## 当前实跑记录

2026-07-18 使用当前仓库本地构建二进制完成一次首跑诊断：

```bash
go build -o /tmp/testloop-mcp-first-run .
TESTLOOP_FIRST_RUN_EXPECT_VERSION=0.5.9 \
TESTLOOP_FIRST_RUN_OUTPUT_DIR=/tmp/testloop-mcp-first-run-check \
  scripts/doctor-first-run.sh /tmp/testloop-mcp-first-run
```

结果：

- `first_run_status=passed`
- `first_run_failed_count=0`
- `first_run_agent_next_step=ready`
- `first_run_report=/tmp/testloop-mcp-first-run-check/verification-report.md`
- `first_run_summary_json=/tmp/testloop-mcp-first-run-check/verification-summary.json`
- `first_run_decision=/tmp/testloop-mcp-first-run-check/agent-decision.txt`
- `first_run_context=/tmp/testloop-mcp-first-run-check/first-run-context.txt`
- `first_run_log=/tmp/testloop-mcp-first-run-check/first-run.log`
