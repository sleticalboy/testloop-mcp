# 首跑诊断 CI 外部项目演练

这个演练用于验证 `scripts/run-first-run-ci.sh` 的复制路径不依赖 testloop-mcp 仓库作为当前工作目录，并且能在外部用户项目里生成首跑诊断七件套。

脚本默认会在 `/tmp` 创建一个最小 Go 项目，把 bootstrap 脚本复制到临时路径，然后从这个外部项目目录执行：

```bash
go build -o /tmp/testloop-mcp .
TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp \
  scripts/showcase-first-run-ci-external-project.sh
```

预期输出：

```text
external_first_run_status=passed
```

演练成功时会生成：

- `/tmp/testloop-external-first-run/artifacts/verification-report.md`
- `/tmp/testloop-external-first-run/artifacts/verification-summary.json`
- `/tmp/testloop-external-first-run/artifacts/verification-summary.schema.json`
- `/tmp/testloop-external-first-run/artifacts/agent-decision.txt`
- `/tmp/testloop-external-first-run/artifacts/first-run-context.txt`
- `/tmp/testloop-external-first-run/artifacts/agent-response.txt`
- `/tmp/testloop-external-first-run/artifacts/first-run.log`

其中 `verification-summary.json` 应为 `overall_status=passed`、`failed_count=0`，`agent-decision.txt` 应包含 `agent_next_step=ready`，`first-run-context.txt` 和 `agent-response.txt` 都应包含 `first_run_agent_next_step=ready`。

如果要验证 web 模板的命令形态，可以切换到 Node 模式：

```bash
go build -o /tmp/testloop-mcp .
TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp \
TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE=node \
  scripts/showcase-first-run-ci-external-project.sh
```

Node 模式会创建一个无第三方依赖的临时项目，先生成 `pnpm-lock.yaml`，再通过 `pnpm install --frozen-lockfile && pnpm build` 跑用户项目 smoke。也可以用 `TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE=all` 连续验证 Go 和 Node 两类项目。

## 当前实跑记录

2026-07-19 使用当前仓库本地构建二进制完成一次 Go 演练：

```bash
go build -o /tmp/testloop-mcp-external-first-run .
TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp-external-first-run \
TESTLOOP_MCP_VERSION=v0.5.7 \
  scripts/showcase-first-run-ci-external-project.sh
```

结果：

- `external_first_run_project=/tmp/testloop-external-first-run/project-go`
- `external_first_run_output_dir=/tmp/testloop-external-first-run/artifacts`
- `external_first_run_agent_response=/tmp/testloop-external-first-run/artifacts/agent-response.txt`
- `external_first_run_status=passed`
- `first_run_agent_next_step=ready`

Node/web 命令形态演练：

```bash
go build -o /tmp/testloop-mcp-external-first-run .
TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp-external-first-run \
TESTLOOP_MCP_VERSION=v0.5.7 \
TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE=node \
  scripts/showcase-first-run-ci-external-project.sh
```

结果：

- `external_first_run_node_project=/tmp/testloop-external-first-run/project-node`
- `external_first_run_node_output_dir=/tmp/testloop-external-first-run/artifacts`
- `external_first_run_node_context=/tmp/testloop-external-first-run/artifacts/first-run-context.txt`
- `external_first_run_node_agent_response=/tmp/testloop-external-first-run/artifacts/agent-response.txt`
- `external_first_run_node_status=passed`
- `external_first_run_status=passed`
- `first_run_agent_next_step=ready`

同日复验了 `TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE=all`，连续生成：

- `/tmp/testloop-external-first-run/artifacts/go/verification-summary.json`
- `/tmp/testloop-external-first-run/artifacts/go/verification-summary.schema.json`
- `/tmp/testloop-external-first-run/artifacts/go/first-run-context.txt`
- `/tmp/testloop-external-first-run/artifacts/go/agent-response.txt`
- `/tmp/testloop-external-first-run/artifacts/node/verification-summary.json`
- `/tmp/testloop-external-first-run/artifacts/node/verification-summary.schema.json`
- `/tmp/testloop-external-first-run/artifacts/node/first-run-context.txt`
- `/tmp/testloop-external-first-run/artifacts/node/agent-response.txt`

两条路径均输出 `first_run_agent_next_step=ready`，最终输出 `external_first_run_mode=all`、`external_first_run_status=passed`。

## 适用边界

- 这条路径面向维护者和接入方演示，不进入默认 CI 的完整执行矩阵。
- 默认 CI 只保护脚本语法、帮助输出和文档入口，避免让常规提交依赖本机网络或额外下载。
- 如果要模拟 GitHub Actions 复制模板，可以把 `scripts/run-first-run-ci.sh` 下载或复制到项目外路径，再从用户项目目录执行。
- Node 模式需要本机有 `pnpm`。
- 如果本机无法下载 GitHub Release 资产，可以先构建本地二进制并传入 `TESTLOOP_MCP_COMMAND`。
