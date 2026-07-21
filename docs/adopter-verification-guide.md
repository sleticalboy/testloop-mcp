# 接入方一页式验证指南

这份指南面向要把 testloop-mcp 接到自己项目里的用户。目标是少做选择：先确认本机安装可用，再把同一套反馈入口放进 CI，失败时把稳定上下文交给 AI Agent。

## 1. 安装并确认版本

推荐 Homebrew：

```bash
brew tap sleticalboy/tap
brew install testloop-mcp
testloop-mcp --version
```

当前文档期望版本是 `0.5.19`。如果版本不对：

```bash
brew update
brew upgrade sleticalboy/tap/testloop-mcp
brew reinstall sleticalboy/tap/testloop-mcp
```

源码 checkout 或临时二进制也可以直接传绝对路径给后续脚本。

注意区分两类版本检查：`testloop-mcp --version` 检查当前 `PATH` 上的二进制；CI bootstrap 中的 `TESTLOOP_MCP_VERSION=v0.5.19` 会下载并使用指定发布版本。真实接入时如果 bootstrap 通过但本机 `PATH` 仍是旧版本，CI 链路可以继续使用，但手动配置 Codex / Claude / Cursor 前仍应先升级或修正 `PATH`，避免客户端启动到旧二进制。

## 2. 本机首跑诊断

首次接入先跑首跑诊断：

```bash
TESTLOOP_FIRST_RUN_EXPECT_VERSION=0.5.19 \
  scripts/doctor-first-run.sh "$(command -v testloop-mcp)"
```

如果输出：

```text
first_run_agent_next_step=ready
```

说明安装、配置生成、真实 MCP transport 和最小 Agent 闭环都已通过。

如果不是 `ready`，优先保留这些文件：

- `verification-report.md`
- `verification-summary.json`
- `verification-summary.schema.json`
- `agent-decision.txt`
- `first-run-context.txt`
- `agent-response.txt`
- `first-run.log`

其中 `first-run-context.txt` 可以直接交给 AI Agent。

## 3. CI 里怎么选

首次接入、安装漂移排查、或者希望失败时直接给 Agent 上下文，用 first-run CI：

```bash
curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/run-first-run-ci.sh -o /tmp/testloop-first-run-ci.sh
TESTLOOP_MCP_VERSION=v0.5.19 \
TESTLOOP_FIRST_RUN_OUTPUT_DIR=/tmp/testloop-first-run \
TESTLOOP_FIRST_RUN_PROJECT_DIR="$PWD" \
  bash /tmp/testloop-first-run-ci.sh 'go test ./...'
```

稳定接入后的 PR / 发布后 smoke，用 onboarding CI：

```bash
curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/run-onboarding-ci.sh -o /tmp/testloop-onboarding-ci.sh
TESTLOOP_MCP_VERSION=v0.5.19 \
TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-onboarding \
TESTLOOP_ONBOARDING_PROJECT_DIR="$PWD" \
  bash /tmp/testloop-onboarding-ci.sh 'go test ./...'
```

前端项目把 smoke 命令换成：

```bash
pnpm install --frozen-lockfile && pnpm build
```

## 4. CI artifact 固定上传

first-run CI 上传七件套：

- `/tmp/testloop-first-run/verification-report.md`
- `/tmp/testloop-first-run/verification-summary.json`
- `/tmp/testloop-first-run/verification-summary.schema.json`
- `/tmp/testloop-first-run/agent-decision.txt`
- `/tmp/testloop-first-run/first-run-context.txt`
- `/tmp/testloop-first-run/agent-response.txt`
- `/tmp/testloop-first-run/first-run.log`

onboarding CI 上传五件套：

- `/tmp/testloop-onboarding/verification-report.md`
- `/tmp/testloop-onboarding/verification-summary.json`
- `/tmp/testloop-onboarding/verification-summary.schema.json`
- `/tmp/testloop-onboarding/agent-decision.txt`
- `/tmp/testloop-onboarding/agent-response.txt`

GitHub Actions 里上传 artifact 时使用 `if: always()`，否则失败时最需要的上下文可能不会被保存。

如果要把 artifact 消费也纳入客户端或 Agent 回归，优先读取机器可读索引：

```bash
go run ./examples/agent-response-manifest-demo \
  docs/fixtures/agent-response-artifact-manifest.json
```

manifest 的结构契约见 [agent-response-artifact-manifest.schema.json](./fixtures/agent-response-artifact-manifest.schema.json)，其中固定了 first-run/onboarding artifact 目录、必备文件、期望 action 和 `fallback_order`。`verification-summary.json` 的结构契约见 [verification-summary.schema.json](./fixtures/verification-summary.schema.json)，客户端可用它校验 `sections[].signals.action` 这类可选动作信号。

下载 artifact 后也可以直接跑目录自检：

```bash
sh scripts/verify-agent-artifact.sh first-run /tmp/testloop-first-run
sh scripts/verify-agent-artifact.sh onboarding /tmp/testloop-onboarding
```

正常输出包含 `agent_artifact_status=passed`。复制型 bootstrap 在 helper checkout 支持时会自动执行同类校验，并在 GitHub step summary 写入 `Artifact verification`。

## 5. 失败时看哪个字段

优先看 `agent-decision.txt`：

```text
agent_next_step=ready
```

常见 action：

| action | 下一步 |
| --- | --- |
| `ready` | 接入链路通过，可以开始真实生成、修复或覆盖率补测。 |
| `fix-installation` | 检查二进制路径、版本、Homebrew 链接或安装脚本输出。 |
| `inspect-mcp-transport` | 检查 stdio / Streamable HTTP MCP 启动和客户端配置。 |
| `inspect-agent-demo` | 检查结构化返回、最小 Agent demo 和 demo runner。 |
| `inspect-user-project` | 检查用户项目 smoke，通常是依赖、环境变量、测试命令或项目自身失败。 |

first-run CI 失败时，优先把 `agent-response.txt` 交给 AI Agent；旧版 artifact 没有回复草稿时再交 `first-run-context.txt`。onboarding CI 失败时，也优先把 `agent-response.txt` 交给 Agent；需要下钻时再补 `agent-decision.txt`、`verification-summary.json` 和 Markdown 报告里失败 section。

## 6. 改模板后如何自证

维护者或团队改 CI 模板后，先用临时外部项目演练复制路径：

```bash
go build -o /tmp/testloop-mcp .
TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp \
TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE=all \
  scripts/showcase-first-run-ci-external-project.sh

TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp \
TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE=all \
  scripts/showcase-onboarding-ci-external-project.sh
```

两条都应输出 `external_*_status=passed`。

## 相关文档

- [5 分钟接入向导](./quickstart.md)
- [验收报告 CI 集成](./verification-ci.md)
- [首跑诊断](./first-run-diagnostics.md)
- [首跑诊断 CI 复制模板](./first-run-ci-template.md)
- [Onboarding CI 复制模板](./onboarding-ci-template.md)
- [MCP 客户端契约测试说明](./mcp-client-contract-tests.md)
- [真实结构化 fixture](./fixtures.md)
