# 用户项目验收报告

`scripts/generate-verification-report.sh` 用于把 testloop-mcp 的安装验收、真实 MCP 协议 smoke、最小 Agent 闭环 demo，以及可选的公开 showcase / 用户项目 smoke 聚合成一份 Markdown 报告。

它适合发布后核验、接入方验收、录屏前检查和用户项目试跑记录。默认路径不访问公网；公开 Go / JS showcase 需要显式开启。

## 快速使用

```bash
scripts/generate-verification-report.sh "$(command -v testloop-mcp)" /tmp/testloop-report.md
```

默认会执行三段稳定验收：

1. `scripts/verify-client-setup.sh`：检查二进制、`--version`、`--doctor-config`、客户端配置 roundtrip 和 HTTP `/healthz`。
2. `scripts/verify-mcp-process-smoke.sh`：用真实 MCP SDK 客户端通过 stdio / Streamable HTTP 启动二进制并调用轻量工具。
3. `go run ./examples/mcp-client-demo`：验证 `run_tests -> repair_task -> rerun -> parse_coverage` 的最小 Agent 反馈闭环。

报告会写入指定 Markdown 文件；任一已执行验收项失败时，脚本仍会写出报告，但最终返回非零 exit code。

如果还需要给 Agent 或 CI 使用的机器可读结果，可以同时输出 summary JSON：

```bash
TESTLOOP_REPORT_SUMMARY_JSON=/tmp/testloop-report-summary.json \
  scripts/generate-verification-report.sh "$(command -v testloop-mcp)" /tmp/testloop-report.md
```

## 版本门禁

发布后或 Homebrew 安装后，建议加上期望版本，防止 PATH 指到旧二进制：

```bash
TESTLOOP_REPORT_EXPECT_VERSION=0.5.11 \
  scripts/generate-verification-report.sh "$(command -v testloop-mcp)" /tmp/testloop-report.md
```

## 用户项目 smoke

如果要把接入方自己的项目命令纳入同一份报告，设置项目目录和命令：

```bash
TESTLOOP_REPORT_PROJECT_DIR=/path/to/project \
TESTLOOP_REPORT_PROJECT_COMMAND='go test ./...' \
  scripts/generate-verification-report.sh "$(command -v testloop-mcp)" /tmp/testloop-report.md
```

Vue / JS 项目可以使用已有测试或构建命令，例如：

```bash
TESTLOOP_REPORT_PROJECT_DIR=/path/to/web \
TESTLOOP_REPORT_PROJECT_COMMAND='pnpm test -- --run' \
  scripts/generate-verification-report.sh "$(command -v testloop-mcp)" /tmp/testloop-web-report.md
```

项目 smoke 不会自动推断命令，原因是不同仓库的测试、数据库、外部服务和环境变量差异很大。让调用方显式传入命令，可以避免把不安全或耗时的任务误放进验收脚本。

## 公开 showcase

公开 showcase 依赖 GitHub、npm registry 或外部项目测试环境，因此默认跳过。需要时可以显式开启：

```bash
TESTLOOP_REPORT_PUBLIC_SHOWCASES=go \
  scripts/generate-verification-report.sh "$(command -v testloop-mcp)" /tmp/testloop-go-showcase-report.md

TESTLOOP_REPORT_PUBLIC_SHOWCASES=all \
  scripts/generate-verification-report.sh "$(command -v testloop-mcp)" /tmp/testloop-showcase-report.md
```

可选值：

- `none`：默认值，不执行公开 showcase。
- `go`：执行公开 Go showcase。
- `js`：执行公开 JS/TS showcase。
- `all`：依次执行公开 Go 和 JS/TS showcase。

公开 showcase 的 JSONL 明细仍写在脚本临时目录中，报告只记录命令输出摘要。

## 常用环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `TESTLOOP_MCP_COMMAND` | 空 | 未通过第一个参数传二进制时使用。 |
| `TESTLOOP_REPORT_OUTPUT` | `/tmp/testloop-mcp-verification-report.md` | 未通过第二个参数传报告路径时使用。 |
| `TESTLOOP_REPORT_TITLE` | `testloop-mcp 验收报告` | Markdown H1 标题。 |
| `TESTLOOP_REPORT_SUMMARY_JSON` | 空 | 可选 summary JSON 输出路径，适合 Agent / CI 消费。 |
| `TESTLOOP_REPORT_EXPECT_VERSION` | 空 | 透传给基础安装验收脚本，用于版本门禁。 |
| `TESTLOOP_REPORT_SKIP_BASIC` | `false` | 设为 `true` 跳过基础安装验收。 |
| `TESTLOOP_REPORT_SKIP_PROCESS_SMOKE` | `false` | 设为 `true` 跳过真实 MCP 协议 smoke。 |
| `TESTLOOP_REPORT_SKIP_AGENT_DEMO` | `false` | 设为 `true` 跳过最小 Agent 闭环 demo。 |
| `TESTLOOP_REPORT_PUBLIC_SHOWCASES` | `none` | 控制公开 showcase：`none`、`go`、`js`、`all`。 |
| `TESTLOOP_REPORT_PROJECT_DIR` | 空 | 用户项目目录。 |
| `TESTLOOP_REPORT_PROJECT_COMMAND` | 空 | 在用户项目目录中执行的 smoke 命令。 |

## 报告解读

报告包含三部分：

- 元数据：生成时间、仓库路径、Git ref、二进制路径和版本输出。
- 汇总表：每个验收项的 `passed`、`failed` 或 `skipped` 状态。
- 明细：每个已执行命令的 stdout / stderr。

对 Agent 或维护者来说，最重要的是先看汇总表。如果只有用户项目 smoke 失败，通常说明接入项目自身测试环境或命令需要调整；如果基础安装验收或真实 MCP 协议 smoke 失败，应先修 testloop-mcp 安装、PATH、客户端配置或传输链路。

summary JSON 包含同一批元数据和 section 汇总，推荐给自动化消费：

```json
{
  "overall_status": "passed",
  "failed_count": 0,
  "sections": [
    {
      "name": "基础安装验收",
      "status": "passed",
      "exit_code": 0,
      "reason": null
    }
  ]
}
```

Markdown 适合发给人看，JSON 适合 Agent/CI 做分流。自动化侧建议优先读取 `overall_status` 和 `sections[].status`；当某个 section 是 `failed` 时，再打开 Markdown 明细查看 stdout / stderr。

仓库提供一个最小决策示例，演示 Agent 如何消费 summary JSON：

```bash
go run ./examples/verification-summary-decision-demo /tmp/testloop-report-summary.json
```

决策规则保持刻意简单：

- `overall_status=passed`：输出 `agent_next_step=ready`。
- `基础安装验收` 失败：输出 `fix-installation`，优先检查二进制路径、版本、配置 roundtrip 和 `/healthz`。
- `真实 MCP 协议 smoke` 失败：输出 `inspect-mcp-transport`，优先排查 stdio / Streamable HTTP 客户端启动。
- `最小 Agent 闭环 demo` 失败：输出 `inspect-agent-demo`，优先排查结构化反馈和 demo runner。
- `公开 showcase` 失败：输出 `inspect-showcase`，优先排查外部网络、本地 checkout 和 action 期望。
- `用户项目 smoke` 失败：输出 `inspect-user-project`，优先排查用户项目命令、依赖、环境变量和测试输出。

CI 集成示例见 [验收报告 CI 集成](./verification-ci.md)。

失败分流样例见 [验收 Summary 失败分流样例](./verification-summary-failures.md)，其中包含安装、MCP 协议、Agent demo、公开 showcase 和用户项目 smoke 失败时的最小 summary JSON fixture。

## 真实项目 smoke 记录

2026-07-18 用本机 `laoxia-scaffold-v1.0.0` 做过一次 server + web 验收报告试跑。该记录用于验证报告脚本在多项目场景下的可读性，不等同于 `validate_coverage_task` 的 top-N 生成质量 benchmark。

Go server：

```bash
TESTLOOP_REPORT_EXPECT_VERSION=0.5.2 \
TESTLOOP_REPORT_PROJECT_DIR=/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server \
TESTLOOP_REPORT_PROJECT_COMMAND='go test ./...' \
  scripts/generate-verification-report.sh /tmp/testloop-mcp-report /tmp/testloop-laoxia-server-report.md
```

结果：基础安装验收、真实 MCP 协议 smoke、最小 Agent 闭环 demo、用户项目 `go test ./...` 全部 `passed`。Go 项目输出包含 `gopsutil/disk` 的 macOS `IOMasterPort` deprecated warning，但测试 exit code 为 `0`。

Vue web：

```bash
TESTLOOP_REPORT_EXPECT_VERSION=0.5.2 \
TESTLOOP_REPORT_PROJECT_DIR=/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-web \
TESTLOOP_REPORT_PROJECT_COMMAND='pnpm install --frozen-lockfile && pnpm build:prod' \
  scripts/generate-verification-report.sh /tmp/testloop-mcp-report /tmp/testloop-laoxia-web-report.md
```

结果：基础安装验收、真实 MCP 协议 smoke、最小 Agent 闭环 demo、用户项目 `pnpm install --frozen-lockfile && pnpm build:prod` 全部 `passed`。Vue 构建输出包含 browserslist 数据过期和 bundle size warning，但构建 exit code 为 `0`。

这次试跑确认两点：

- 报告脚本适合同时覆盖 testloop-mcp 自身接入链路和用户项目 smoke。
- 用户项目命令需要显式传入；不同项目的安装、构建、测试成本和环境依赖差异太大，不应由脚本默认猜测。
