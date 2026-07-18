# Onboarding CI 外部项目演练

这个演练用于验证 `scripts/run-onboarding-ci.sh` 的复制路径不依赖 testloop-mcp 仓库作为当前工作目录。

脚本默认会在 `/tmp` 创建一个最小 Go 项目，把 bootstrap 脚本复制到临时路径，然后从这个外部项目目录执行：

```bash
go build -o /tmp/testloop-mcp .
TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp \
  scripts/showcase-onboarding-ci-external-project.sh
```

预期输出：

```text
external_onboarding_status=passed
```

演练成功时会生成：

- `/tmp/testloop-external-onboarding/artifacts/verification-report.md`
- `/tmp/testloop-external-onboarding/artifacts/verification-summary.json`
- `/tmp/testloop-external-onboarding/artifacts/agent-decision.txt`

其中 `verification-summary.json` 应为 `overall_status=passed`、`failed_count=0`，`agent-decision.txt` 应包含 `agent_next_step=ready`。

如果要验证 web 模板的命令形态，可以切换到 Node 模式：

```bash
go build -o /tmp/testloop-mcp .
TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp \
TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE=node \
  scripts/showcase-onboarding-ci-external-project.sh
```

Node 模式会创建一个无第三方依赖的临时项目，先生成 `pnpm-lock.yaml`，再通过 `pnpm install --frozen-lockfile && pnpm build` 跑用户项目 smoke。也可以用 `TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE=all` 连续验证 Go 和 Node 两类项目。

## 当前实跑记录

2026-07-18 使用当前仓库本地构建二进制完成一次 Go 演练：

```bash
go build -o /tmp/testloop-mcp-external-onboarding .
TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp-external-onboarding \
TESTLOOP_MCP_VERSION=v0.5.6 \
  scripts/showcase-onboarding-ci-external-project.sh
```

结果：

- `external_onboarding_project=/tmp/testloop-external-onboarding/project-go`
- `external_onboarding_output_dir=/tmp/testloop-external-onboarding/artifacts`
- `external_onboarding_status=passed`
- `agent_next_step=ready`

同日补充 Node/web 命令形态演练：

```bash
go build -o /tmp/testloop-mcp-external-onboarding .
TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp-external-onboarding \
TESTLOOP_MCP_VERSION=v0.5.6 \
TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE=node \
  scripts/showcase-onboarding-ci-external-project.sh
```

结果：

- `external_onboarding_node_project=/tmp/testloop-external-onboarding/project-node`
- `external_onboarding_node_output_dir=/tmp/testloop-external-onboarding/artifacts`
- `external_onboarding_node_status=passed`
- `external_onboarding_status=passed`
- `agent_next_step=ready`

最终也复验了 `TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE=all`，连续生成：

- `/tmp/testloop-external-onboarding/artifacts/go/verification-summary.json`
- `/tmp/testloop-external-onboarding/artifacts/node/verification-summary.json`

两条路径均输出 `agent_next_step=ready`，最终输出 `external_onboarding_mode=all`、`external_onboarding_status=passed`。

## 适用边界

- 这条路径面向维护者和接入方演示，不进入默认 CI 的完整执行矩阵。
- 默认 CI 只保护脚本语法、帮助输出和文档入口，避免让常规提交依赖本机网络或额外下载。
- 如果要模拟 GitHub Actions 复制模板，可以把 `scripts/run-onboarding-ci.sh` 下载或复制到项目外路径，再从用户项目目录执行。
- Node 模式需要本机有 `pnpm`。
- 如果本机无法下载 GitHub Release 资产，可以先构建本地二进制并传入 `TESTLOOP_MCP_COMMAND`。
