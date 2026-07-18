# 真实接入案例模板

这份文档用于记录接入方项目如何用 `testloop-mcp` 完成一次可复查的本机验收。重点不是证明“生成测试质量已经覆盖所有业务场景”，而是证明 AI Agent 可以拿到稳定的反馈闭环：安装验收、真实 MCP 协议 smoke、最小 Agent demo、用户项目 smoke，以及最终 `agent_next_step`。

## 适用场景

- 给 Codex / Claude Code / Cursor 接入 `testloop-mcp` 前，确认本机二进制和 MCP 传输链路可用。
- 在真实 server / web / CLI 项目中，把项目自己的测试或构建命令纳入同一份报告。
- 在 CI 或交付记录中保存 Markdown 报告、summary JSON 和 Agent 决策输出。
- 复盘失败时，先区分 testloop-mcp 安装问题、MCP 传输问题、Agent demo 问题、公开 showcase 问题，还是用户项目自身 smoke 问题。

## 推荐模板

先确认要验收的二进制版本。发布后建议加版本门禁，避免 `PATH` 指向旧版本：

```bash
TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.4 \
  scripts/showcase-agent-onboarding-report.sh "$(command -v testloop-mcp)"
```

如果本机安装版本和源码版本不一致，可以先使用源码构建临时二进制跑案例，避免把安装漂移误判成项目接入失败：

```bash
go build -o /tmp/testloop-mcp-v0.5.4-case .
/tmp/testloop-mcp-v0.5.4-case --version
```

接入真实项目时，固定四个变量：

```bash
TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.4 \
TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-my-project-onboarding \
TESTLOOP_REPORT_TITLE='my-project 接入验收报告' \
TESTLOOP_REPORT_PROJECT_DIR=/path/to/my-project \
TESTLOOP_REPORT_PROJECT_COMMAND='go test ./...' \
  scripts/showcase-agent-onboarding-report.sh /absolute/path/to/testloop-mcp
```

输出固定包含三类制品：

| 制品 | 用途 |
| --- | --- |
| `verification-report.md` | 给人看的完整 Markdown 验收报告，包含每个 section 的 stdout / stderr。 |
| `verification-summary.json` | 给 Agent / CI 读取的结构化汇总，包含 `overall_status`、`failed_count` 和 section 状态。 |
| `agent-decision.txt` | 最小决策输出，核心字段是 `agent_next_step`。 |

## 结果判断

优先看 summary JSON 和 decision：

```bash
jq '{overall_status, failed_count, sections: [.sections[] | {name,status,exit_code}]}' \
  /tmp/testloop-my-project-onboarding/verification-summary.json

cat /tmp/testloop-my-project-onboarding/agent-decision.txt
```

常见结论：

| `agent_next_step` | 含义 | 下一步 |
| --- | --- | --- |
| `ready` | testloop-mcp 接入链路和用户项目 smoke 都通过。 | 可以进入真实生成/补测/修复闭环。 |
| `fix-installation` | 基础安装验收失败。 | 先检查二进制路径、版本、`--print-config`、`--check-config` 和 `/healthz`。 |
| `inspect-mcp-transport` | 真实 MCP 协议 smoke 失败。 | 先排查 stdio / Streamable HTTP 启动、端口和客户端传输配置。 |
| `inspect-agent-demo` | 最小 Agent demo 失败。 | 先排查结构化返回、demo runner 和仓库自身构建。 |
| `inspect-showcase` | 公开 showcase 失败。 | 先排查 GitHub/npm 网络、本地 checkout 和 action 期望。 |
| `inspect-user-project` | 用户项目 smoke 失败。 | 先看用户项目测试/构建命令、依赖、环境变量和失败日志。 |

构建 warning 不等于失败。只有对应 section 的 exit code 非 0，才会让 summary 进入 failed。

## laoxia server 样例

样例项目：

- 项目：`/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server`
- 类型：Go server
- 命令：`go test ./...`
- 二进制：`/tmp/testloop-mcp-v0.5.4-case`
- 版本输出：`testloop-mcp 0.5.4`

运行命令：

```bash
rm -rf /tmp/testloop-laoxia-server-onboarding-v0.5.4
TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.4 \
TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-laoxia-server-onboarding-v0.5.4 \
TESTLOOP_REPORT_TITLE='laoxia car-admin-server 接入验收报告' \
TESTLOOP_REPORT_PROJECT_DIR=/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server \
TESTLOOP_REPORT_PROJECT_COMMAND='go test ./...' \
  scripts/showcase-agent-onboarding-report.sh /tmp/testloop-mcp-v0.5.4-case
```

验收结果：

```text
overall_status=passed
failed_count=0
agent_next_step=ready
```

section 结果：

| 验收项 | 状态 | Exit code |
| --- | --- | --- |
| 基础安装验收 | `passed` | `0` |
| 真实 MCP 协议 smoke | `passed` | `0` |
| 最小 Agent 闭环 demo | `passed` | `0` |
| 公开 showcase | `skipped` | `-` |
| 用户项目 smoke | `passed` | `0` |

本地制品路径：

- `/tmp/testloop-laoxia-server-onboarding-v0.5.4/verification-report.md`
- `/tmp/testloop-laoxia-server-onboarding-v0.5.4/verification-summary.json`
- `/tmp/testloop-laoxia-server-onboarding-v0.5.4/agent-decision.txt`

## laoxia web 样例

样例项目：

- 项目：`/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-web`
- 类型：Vue web
- 命令：`pnpm install --frozen-lockfile && pnpm build:prod`
- 二进制：`/tmp/testloop-mcp-v0.5.4-case`
- 版本输出：`testloop-mcp 0.5.4`

运行命令：

```bash
rm -rf /tmp/testloop-laoxia-web-onboarding-v0.5.4
TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.4 \
TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-laoxia-web-onboarding-v0.5.4 \
TESTLOOP_REPORT_TITLE='laoxia car-admin-web 接入验收报告' \
TESTLOOP_REPORT_PROJECT_DIR=/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-web \
TESTLOOP_REPORT_PROJECT_COMMAND='pnpm install --frozen-lockfile && pnpm build:prod' \
  scripts/showcase-agent-onboarding-report.sh /tmp/testloop-mcp-v0.5.4-case
```

验收结果：

```text
overall_status=passed
failed_count=0
agent_next_step=ready
```

section 结果：

| 验收项 | 状态 | Exit code |
| --- | --- | --- |
| 基础安装验收 | `passed` | `0` |
| 真实 MCP 协议 smoke | `passed` | `0` |
| 最小 Agent 闭环 demo | `passed` | `0` |
| 公开 showcase | `skipped` | `-` |
| 用户项目 smoke | `passed` | `0` |

本地制品路径：

- `/tmp/testloop-laoxia-web-onboarding-v0.5.4/verification-report.md`
- `/tmp/testloop-laoxia-web-onboarding-v0.5.4/verification-summary.json`
- `/tmp/testloop-laoxia-web-onboarding-v0.5.4/agent-decision.txt`

这两个样例说明：server 和 web 项目都可以复用同一个 onboarding report wrapper；差异只在 `TESTLOOP_REPORT_PROJECT_DIR` 和 `TESTLOOP_REPORT_PROJECT_COMMAND`。这正是 testloop-mcp 当前最应该强化的价值点：让 Agent 不靠猜测，而是读取结构化结果继续推进。
