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
TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.7 \
  scripts/showcase-agent-onboarding-report.sh "$(command -v testloop-mcp)"
```

如果本机安装版本和源码版本不一致，可以先使用源码构建临时二进制跑案例，避免把安装漂移误判成项目接入失败：

```bash
go build -o /tmp/testloop-mcp-v0.5.7-case .
/tmp/testloop-mcp-v0.5.7-case --version
```

接入真实项目时，固定四个变量：

```bash
TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.7 \
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

## laoxia v0.5.7 CI bootstrap 实跑记录

样例项目：

- 根目录：`/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0`
- Server：`car-admin-server`，Go 项目，本地仓库状态干净。
- Web：`car-admin-web`，Vue 项目，使用 `pnpm-lock.yaml`，本地仓库状态干净。
- 使用版本：`TESTLOOP_MCP_VERSION=v0.5.7`
- 实际二进制：`/Users/binlee/.local/bin/testloop-mcp`
- 版本输出：`testloop-mcp 0.5.7`

这次演练按 [接入方一页式验证指南](./adopter-verification-guide.md) 执行，覆盖 first-run CI 和 onboarding CI 两条 bootstrap。执行后两个外部项目工作区仍保持干净，说明脚本只在指定 `/tmp/testloop-*` 输出目录写入验收制品，不会污染用户仓库。

### Server first-run

运行命令：

```bash
TESTLOOP_MCP_VERSION=v0.5.7 \
TESTLOOP_FIRST_RUN_OUTPUT_DIR=/tmp/testloop-laoxia-server-first-run \
TESTLOOP_FIRST_RUN_PROJECT_DIR=/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server \
  scripts/run-first-run-ci.sh 'go test ./...'
```

验收结果：

```text
first_run_status=passed
first_run_failed_count=0
first_run_agent_next_step=ready
```

summary 结果：

```text
overall_status=passed
failed_count=0
version_output=testloop-mcp 0.5.7
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

- `/tmp/testloop-laoxia-server-first-run/verification-report.md`
- `/tmp/testloop-laoxia-server-first-run/verification-summary.json`
- `/tmp/testloop-laoxia-server-first-run/agent-decision.txt`
- `/tmp/testloop-laoxia-server-first-run/first-run-context.txt`
- `/tmp/testloop-laoxia-server-first-run/first-run.log`

项目侧 smoke 输出包含 macOS `IOMasterPort` deprecated warning，但 `go test ./...` 退出码为 0，因此只作为 warning 记录，不影响 testloop 接入链路判定。

### Web first-run

运行命令：

```bash
TESTLOOP_MCP_VERSION=v0.5.7 \
TESTLOOP_FIRST_RUN_OUTPUT_DIR=/tmp/testloop-laoxia-web-first-run \
TESTLOOP_FIRST_RUN_PROJECT_DIR=/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-web \
  scripts/run-first-run-ci.sh 'pnpm install --frozen-lockfile && pnpm build:prod'
```

验收结果：

```text
first_run_status=passed
first_run_failed_count=0
first_run_agent_next_step=ready
```

summary 结果：

```text
overall_status=passed
failed_count=0
version_output=testloop-mcp 0.5.7
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

- `/tmp/testloop-laoxia-web-first-run/verification-report.md`
- `/tmp/testloop-laoxia-web-first-run/verification-summary.json`
- `/tmp/testloop-laoxia-web-first-run/agent-decision.txt`
- `/tmp/testloop-laoxia-web-first-run/first-run-context.txt`
- `/tmp/testloop-laoxia-web-first-run/first-run.log`

项目侧 smoke 输出包含 `pnpm approve-builds`、Browserslist 和 Webpack asset size warning，但 `pnpm install --frozen-lockfile && pnpm build:prod` 退出码为 0，因此只作为项目 warning 记录。

## laoxia 双栈报告入口

前面的 server/web 验收已经证明两条 smoke 都能通过。为了后续复用更方便，可以直接使用新的双栈入口一次性产出两份报告：

```bash
scripts/showcase-laoxia-scaffold-report.sh "$(command -v testloop-mcp)"
```

默认输出目录为 `/tmp/testloop-laoxia-scaffold`，里面会分别落下：

- `server/verification-report.md`
- `server/verification-summary.json`
- `web/verification-report.md`
- `web/verification-summary.json`
- `laoxia-summary.json`，其中会嵌套 server/web 子 summary，方便 CI 直接读取总状态和子状态。

这个入口适合做项目级回归和发布前复验。它不替代 `generate-verification-report.sh`，只是把 laoxia 这类已经确认过的真实项目路径收成一个更省心的命令。

最新源码复验记录：

```bash
go build -o /tmp/testloop-mcp-latest .

TESTLOOP_LAOXIA_OUTPUT_DIR=/tmp/testloop-laoxia-scaffold-live-20260720154617 \
TESTLOOP_REPORT_SKIP_BASIC=true \
TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true \
TESTLOOP_REPORT_SKIP_AGENT_DEMO=true \
TESTLOOP_REPORT_SKIP_TESTGEN_SMOKE=true \
  scripts/showcase-laoxia-scaffold-report.sh /tmp/testloop-mcp-latest
```

验收结果：

```text
laoxia_server_status=passed
laoxia_server_command=go test ./...
laoxia_web_status=passed
laoxia_web_command=pnpm install --frozen-lockfile && pnpm build:prod
laoxia_status=passed
```

`/tmp/testloop-laoxia-scaffold-live-20260720154617/laoxia-summary.json` 的顶层、server 子 summary 和 web 子 summary 均为 `overall_status=passed`、`failed_count=0`。

## QuickSmoke Go/Java 双项目报告入口

前面的 shared helper 也可以直接复用到跨语言的真实项目 pair。下面这组样本用一个干净的 Go 项目和一个干净的 Java 项目验证了 helper 的通用性：

```bash
TESTLOOP_PAIR_PREFIX=quicksmoke \
TESTLOOP_PAIR_OUTPUT_DIR=/tmp/testloop-quicksmoke-pair \
TESTLOOP_PAIR_FIRST_NAME=go \
TESTLOOP_PAIR_FIRST_TITLE='QuickSmoke Go' \
TESTLOOP_PAIR_FIRST_DIR=/Users/binlee/code/free-works/QuickSmoke-Backend-Go \
TESTLOOP_PAIR_FIRST_COMMAND='go test ./...' \
TESTLOOP_PAIR_SECOND_NAME=java \
TESTLOOP_PAIR_SECOND_TITLE='Words Java' \
TESTLOOP_PAIR_SECOND_DIR=/Users/binlee/code/free-works/words_java \
TESTLOOP_PAIR_SECOND_COMMAND='mvn -q test' \
  scripts/showcase-dual-project-report.sh /tmp/testloop-mcp
```

验收结果：

```text
quicksmoke_status=passed
```

本地制品路径：

- `/tmp/testloop-quicksmoke-pair/go/verification-report.md`
- `/tmp/testloop-quicksmoke-pair/go/verification-summary.json`
- `/tmp/testloop-quicksmoke-pair/java/verification-report.md`
- `/tmp/testloop-quicksmoke-pair/java/verification-summary.json`
- `/tmp/testloop-quicksmoke-pair/quicksmoke-summary.json`

`quicksmoke-summary.json` 会嵌套 go/java 子 summary，顶层 `overall_status` 和 `failed_count` 都可以直接给 Agent 或 CI 读取。

## APK Info Rust / Words Java 双项目报告入口

再补一组跨语言样本，确认 helper 在 Rust workspace 和 Java Maven 项目上也能稳定工作：

```bash
TESTLOOP_PAIR_PREFIX=rustjava \
TESTLOOP_PAIR_OUTPUT_DIR=/tmp/testloop-rustjava-pair \
TESTLOOP_PAIR_FIRST_NAME=rust \
TESTLOOP_PAIR_FIRST_TITLE='APK Info Rust Zip' \
TESTLOOP_PAIR_FIRST_DIR=/Users/binlee/code/free-works/apk-info \
TESTLOOP_PAIR_FIRST_COMMAND='cargo test -q -p apk-info-zip' \
TESTLOOP_PAIR_SECOND_NAME=java \
TESTLOOP_PAIR_SECOND_TITLE='Words Java' \
TESTLOOP_PAIR_SECOND_DIR=/Users/binlee/code/free-works/words_java \
TESTLOOP_PAIR_SECOND_COMMAND='mvn -q test' \
  scripts/showcase-dual-project-report.sh /tmp/testloop-mcp
```

验收结果：

```text
rustjava_status=passed
```

本地制品路径：

- `/tmp/testloop-rustjava-pair/rust/verification-report.md`
- `/tmp/testloop-rustjava-pair/rust/verification-summary.json`
- `/tmp/testloop-rustjava-pair/java/verification-report.md`
- `/tmp/testloop-rustjava-pair/java/verification-summary.json`
- `/tmp/testloop-rustjava-pair/rustjava-summary.json`

这里的 Rust 侧特意收窄到 `cargo test -q -p apk-info-zip`，因为整个 workspace 里还有 Python 绑定包，直接跑 workspace 容易被本机动态库环境卡住。

### Server onboarding

运行命令：

```bash
TESTLOOP_MCP_VERSION=v0.5.7 \
TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-laoxia-server-onboarding \
TESTLOOP_ONBOARDING_PROJECT_DIR=/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server \
  scripts/run-onboarding-ci.sh 'go test ./...'
```

验收结果：

```text
overall_status=passed
failed_count=0
agent_next_step=ready
```

本地制品路径：

- `/tmp/testloop-laoxia-server-onboarding/verification-report.md`
- `/tmp/testloop-laoxia-server-onboarding/verification-summary.json`
- `/tmp/testloop-laoxia-server-onboarding/agent-decision.txt`

### Web onboarding

运行命令：

```bash
TESTLOOP_MCP_VERSION=v0.5.7 \
TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-laoxia-web-onboarding \
TESTLOOP_ONBOARDING_PROJECT_DIR=/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-web \
  scripts/run-onboarding-ci.sh 'pnpm install --frozen-lockfile && pnpm build:prod'
```

验收结果：

```text
overall_status=passed
failed_count=0
agent_next_step=ready
```

本地制品路径：

- `/tmp/testloop-laoxia-web-onboarding/verification-report.md`
- `/tmp/testloop-laoxia-web-onboarding/verification-summary.json`
- `/tmp/testloop-laoxia-web-onboarding/agent-decision.txt`

这次实跑说明：first-run CI 更适合首次接入和失败上下文收集，onboarding CI 更适合稳定接入后的 PR / 发布后 smoke；二者可以覆盖同一类 server / web 项目，差异只在 artifact 数量和失败上下文详细程度。

## laoxia v0.5.4 历史 onboarding 样例

下面保留 v0.5.4 时期使用 `scripts/showcase-agent-onboarding-report.sh` 的历史样例，用来说明项目接入模板的演进。新接入项目应优先复制上面的 v0.5.7 CI bootstrap 示例。

### Server

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

### Web

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
