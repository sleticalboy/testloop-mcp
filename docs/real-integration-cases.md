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

## v0.5.19 客户端 release smoke 实跑记录

这条记录面向 MCP 客户端和 AI Agent 接入方，不依赖 laoxia 项目。目标是验证正式 release tag 上的 raw installer、基础客户端 CI response 和 consumer smoke response 可以合成一份 Agent 可消费 JSON evidence。

运行命令：

```bash
scripts/showcase-agent-decision-client-release-smoke.sh --json
```

实跑过程中 raw.githubusercontent.com 出现过两次 transient 传输错误：

```text
curl: (56) Recv failure: Operation timed out
curl: (56) Send failure: Broken pipe
```

`showcase-agent-decision-client-ci-template-install.sh` 已使用 `curl --retry`、`--retry-all-errors`、`--retry-connrefused` 和 `--max-time` 处理这类网络抖动；最终 release smoke 通过。

关键结果：

```text
status=passed
release_ref=v0.5.19
installer_url=https://raw.githubusercontent.com/sleticalboy/testloop-mcp/v0.5.19/scripts/install-agent-decision-client-ci-template.sh
helper_refs.install=v0.5.19
helper_refs.consumer=v0.5.19
fixture_count=8
agent_next_steps.client=ready
agent_next_steps.consumer=ready
```

决策序列：

```text
accept,accept,accept,manual-review,manual-review,manual-review,apply-repair,needs-better-input
```

这条 smoke 和 Post-Release Verify 的分工不同：Post-Release Verify 验证正式二进制资产下载、安装和 help/version 自检；release smoke 验证外部 MCP 客户端能从 release tag raw installer 安装 workflow，并把 fixture contract summary 转成 Agent 下一步动作。

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
- `dual-project-summary.schema.json`，用于离线校验双项目 combined summary。

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
这个 `laoxia-summary.json` 是双项目 combined summary，不是 `verification-summary.json`；结构契约见 [dual-project-summary.schema.json](./fixtures/dual-project-summary.schema.json)。直接喂给 `examples/verification-summary-decision-demo` 会因为缺少 `sections` 被拒绝。

## laoxia 最新 bootstrap artifact 自检

2026-07-20 使用当前源码构建临时二进制，并分别对 laoxia server/web 跑 onboarding bootstrap，确认新加入的 artifact verifier 在真实项目上输出 `passed`。

构建二进制：

```bash
go build -o /tmp/testloop-mcp-latest .
/tmp/testloop-mcp-latest --version
```

版本输出：

```text
testloop-mcp 0.5.13
```

Server onboarding：

```bash
TESTLOOP_MCP_REPO_DIR="$PWD" \
TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp-latest \
TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.13 \
TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-laoxia-server-onboarding-artifact-verify \
TESTLOOP_ONBOARDING_TITLE='laoxia server onboarding artifact verify' \
TESTLOOP_ONBOARDING_PROJECT_DIR=/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server \
GITHUB_STEP_SUMMARY=/tmp/testloop-laoxia-server-onboarding-artifact-verify-step.md \
  bash scripts/run-onboarding-ci.sh 'go test ./...'
```

Web onboarding：

```bash
TESTLOOP_MCP_REPO_DIR="$PWD" \
TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp-latest \
TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.13 \
TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-laoxia-web-onboarding-artifact-verify \
TESTLOOP_ONBOARDING_TITLE='laoxia web onboarding artifact verify' \
TESTLOOP_ONBOARDING_PROJECT_DIR=/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-web \
GITHUB_STEP_SUMMARY=/tmp/testloop-laoxia-web-onboarding-artifact-verify-step.md \
  bash scripts/run-onboarding-ci.sh 'pnpm install --frozen-lockfile && pnpm build:prod'
```

两边命令输出均包含：

```text
agent_next_step=ready
agent_artifact_status=passed
artifact_kind=onboarding
overall_status=passed
failed_count=0
decision_action=ready
response_action=ready
section_signal=独立 CLI 生成动作 smoke action=manual_review
required_files=5
```

两份 step summary 均包含：

```text
- Status: `passed`
- Failed sections: `0`
- agent_next_step: `ready`
- Artifact verification: `passed`
```

summary JSON 结果：

| 项目 | 输出目录 | overall_status | failed_count | agent_next_step | Artifact verification |
| --- | --- | --- | --- | --- | --- |
| server | `/tmp/testloop-laoxia-server-onboarding-artifact-verify` | `passed` | `0` | `ready` | `passed` |
| web | `/tmp/testloop-laoxia-web-onboarding-artifact-verify` | `passed` | `0` | `ready` | `passed` |

`car-admin-server` 和 `car-admin-web` 的本地 git 状态均为空，说明 bootstrap 和 artifact verifier 只写入 `/tmp/testloop-laoxia-*-onboarding-artifact-verify`，没有污染用户项目工作区。

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
- `/tmp/testloop-quicksmoke-pair/dual-project-summary.schema.json`

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
- `/tmp/testloop-rustjava-pair/dual-project-summary.schema.json`

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

## laoxia server 覆盖率任务闭环样例

这个样例用于验证“覆盖率任务 -> 生成增量测试 -> 运行测试 -> Agent 下一步动作”的真实项目闭环。它使用 laoxia `car-admin-server` 的 `utils` 包，脚本会复制项目到临时目录后执行，不会修改原项目工作区。

运行命令：

```bash
TESTLOOP_VALIDATE_GO_FILE_FILTER=utils \
  scripts/validate-go-coverage-top-tasks.sh \
  /Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server \
  1 \
  /tmp/testloop-laoxia-agent-loop.jsonl
```

本次输出摘要：

```text
result_jsonl=/tmp/testloop-laoxia-agent-loop.jsonl
summary={"limit":1,"status_counts":{"passed":1},"action_counts":{"ready":1},"zero_skip":1,"skipped_total":0}
```

脱敏后的稳定 fixture 见 [laoxia-server-go-utils.json](./fixtures/real-project-agent-loop/laoxia-server-go-utils.json)。关键字段：

```json
{
  "status": "passed",
  "action": "ready",
  "task": {
    "target": "SliceMapper0",
    "file": "utils/alias.go",
    "line_range": "23-26",
    "gap_type": "branch",
    "test_name": "TestSliceMapper0"
  },
  "generated": {
    "status": "ok",
    "generated_cases": 4,
    "action": "manual_review"
  },
  "run_result": {
    "status": "pass",
    "action": "ready",
    "total": 22,
    "passed": 22,
    "failed": 0,
    "skipped": 0
  }
}
```

这条证据的价值不在于证明所有 coverage task 都能自动通过，而是证明真实项目中至少可以把低依赖 `utils` 缺口稳定转成 `ready` 任务：Agent 可以读取 `coverage_task`、生成的测试摘要、`run_result.action=ready` 和 skipped 数量，决定吸收本次增量测试并进入下一个 coverage task。

原始 JSONL 不入仓。原因是 `run_result.raw_output` 会包含外部项目测试日志和本机环境变量；文档和 fixture 只保留可公开复查的结构化摘要。

## mcp-hub 历史 repair 回归样例

这个样例用于验证真实 JS/Vitest 项目里曾经容易退回 `repair_generated_test` 的 async throwing branch。当前期望不是失败，而是生成器可以构造 `await expect(...).rejects.toThrow(...)` 断言，并让 Agent 得到 `passed/ready`。

运行命令：

```bash
TESTLOOP_VALIDATE_JS_TASKS_FILE=/Users/binlee/code/open-source/testloop-mcp/testdata/js-mcp-hub/repair-tasks.jsonl \
TESTLOOP_VALIDATE_JS_TASK_IDS=vitest-mcp-hub-repair-1 \
TESTLOOP_JS_TEST_COMMAND='npx vitest run {path}' \
  scripts/validate-js-coverage-top-tasks.sh \
  /Users/binlee/code/open-source/mcp-hub \
  vitest \
  /tmp/testloop-mcp-hub-repair-agent-loop.jsonl
```

本次输出摘要：

```text
task.validate.done index=1 id=vitest-mcp-hub-repair-1 target=ConfigManager.loadConfig status=passed action=ready
summary={"limit":1,"framework":"vitest","status_counts":{"passed":1},"action_counts":{"ready":1},"zero_skip":1,"skipped_total":0}
```

脱敏后的稳定 fixture 见 [mcp-hub-vitest-repair.json](./fixtures/real-project-agent-loop/mcp-hub-vitest-repair.json)。关键字段：

```json
{
  "status": "passed",
  "action": "ready",
  "task": {
    "id": "vitest-mcp-hub-repair-1",
    "target": "ConfigManager.loadConfig",
    "file": "src/utils/config.js",
    "line_range": "136-136",
    "gap_type": "branch"
  },
  "generated": {
    "status": "ok",
    "generated_cases": 1,
    "action": "ready"
  },
  "run_result": {
    "status": "pass",
    "action": "ready",
    "total": 1,
    "passed": 1,
    "failed": 0,
    "skipped": 0
  }
}
```

这条证据补齐了 ready 分支之外的历史 repair 回归场景：Agent 不需要盲修生成测试，而是可以把该 coverage task 视为已收敛，继续处理下一个任务。

## haoy-apk-station 环境手审样例

这个样例用于验证真实 Python/FastAPI 项目中的环境依赖分支。`app.main` 里的 `serve_frontend` 只有在 `frontend/dist` 存在时才会在模块导入阶段动态定义；这类任务不应该被当成可直接吸收的 ready 测试，而应给 Agent 一个稳定的 `manual_review_environment` 分流。

运行命令：

```bash
TESTLOOP_VALIDATE_PY_TASKS_FILE=/Users/binlee/code/open-source/testloop-mcp/testdata/py-haoy-apk-station/environment-tasks.jsonl \
TESTLOOP_VALIDATE_PY_TASK_IDS=pytest-apk-frontend-env-1 \
TESTLOOP_PYTEST_COMMAND='python3 /Users/binlee/code/open-source/testloop-mcp/scripts/py-manual-review-runner.py {path}' \
  scripts/validate-py-coverage-top-tasks.sh \
  /Users/binlee/code/free-works/haoy-apk-station/backend \
  /tmp/testloop-haoy-apk-manual-review-agent-loop.jsonl
```

本次输出摘要：

```text
task.validate.done index=1 id=pytest-apk-frontend-env-1 target=serve_frontend status=passed action=manual_review_environment
summary={"limit":1,"framework":"pytest","status_counts":{"passed":1},"action_counts":{"manual_review_environment":1},"zero_skip":0,"skipped_total":1}
```

脱敏后的稳定 fixture 见 [haoy-apk-station-py-environment.json](./fixtures/real-project-agent-loop/haoy-apk-station-py-environment.json)。关键字段：

```json
{
  "status": "passed",
  "action": "manual_review_environment",
  "task": {
    "id": "pytest-apk-frontend-env-1",
    "target": "serve_frontend",
    "file": "app/main.py",
    "line_range": "84-89",
    "gap_type": "branch"
  },
  "generated": {
    "status": "ok",
    "generated_cases": 1,
    "action": "manual_review"
  },
  "run_result": {
    "status": "pass",
    "action": "manual_review",
    "total": 1,
    "passed": 0,
    "failed": 0,
    "skipped": 1
  }
}
```

这条证据补齐真实项目 `manual_review_*` 分流：Agent 应记录环境依赖，改用导入前准备 `frontend/dist/index.html` 的集成 fixture 或人工复核，而不是继续对同一个 coverage task 反复生成测试。

## haoy-apk-station 外部服务手审样例

这个样例用于验证真实 Python/FastAPI 项目中的外部服务失败分支。`download_apk` 的代理下载路径依赖外部对象存储 endpoint 和 `urllib.request.urlopen(..., timeout=60)`；这类失败不应该被当成普通断言失败进入自动修复，而应给 Agent 一个稳定的 `manual_review_external_service` 分流。

运行命令：

```bash
TESTLOOP_VALIDATE_PY_TASKS_FILE=/Users/binlee/code/open-source/testloop-mcp/testdata/py-haoy-apk-station/external-service-tasks.jsonl \
TESTLOOP_VALIDATE_PY_TASK_IDS=pytest-apk-download-external-1 \
TESTLOOP_PYTEST_COMMAND='python3 /Users/binlee/code/open-source/testloop-mcp/scripts/py-external-service-runner.py {path}' \
  scripts/validate-py-coverage-top-tasks.sh \
  /Users/binlee/code/free-works/haoy-apk-station/backend \
  /tmp/testloop-haoy-apk-external-service-agent-loop.jsonl
```

本次输出摘要：

```text
task.validate.done index=1 id=pytest-apk-download-external-1 target=download_apk status=failed action=manual_review_external_service
summary={"limit":1,"framework":"pytest","status_counts":{"failed":1},"action_counts":{"manual_review_external_service":1},"zero_skip":1,"skipped_total":0}
```

脱敏后的稳定 fixture 见 [haoy-apk-station-py-external-service.json](./fixtures/real-project-agent-loop/haoy-apk-station-py-external-service.json)。关键字段：

```json
{
  "status": "failed",
  "action": "manual_review_external_service",
  "task": {
    "id": "pytest-apk-download-external-1",
    "target": "download_apk",
    "file": "app/api/apps.py",
    "line_range": "550-570",
    "gap_type": "error_path"
  },
  "run_result": {
    "status": "fail",
    "action": "inspect_failures",
    "total": 1,
    "passed": 0,
    "failed": 1,
    "skipped": 0
  },
  "metadata": {
    "external_service_dependent": true
  }
}
```

这条证据补齐“失败但不应自动修复”的真实项目分流：Agent 应记录外部服务依赖，改用 fake storage client、route data 或集成环境验证，而不是继续对同一个生成测试应用 `fix_suggestions`。

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
