# Agent 决策 release response 客户端接入

这份文档面向已经接入 Agent 决策 fixture CI 的客户端项目。它只说明发布后 smoke summary 的消费方式：把 release smoke 的 JSON 汇总转成稳定 `agent_next_step`，让 Codex、Claude、Cursor 或自研 Agent 知道下一步该接受、排查 installer，还是排查 fixture 漂移。按步骤接入时优先看 [Agent 决策 release response 接入 Checklist](./agent-decision-release-response-checklist.md)。

## 最小目录

接入方项目可以只保留这几个文件：

```text
testloop-release-smoke-summary.json
testloop-release-response.json
package.json
scripts/
  render-agent-decision-client-release-response.mjs
  assert-release-response.mjs
```

其中：

- `testloop-release-smoke-summary.json` 来自 `scripts/showcase-agent-decision-client-release-smoke.sh --json`。
- `render-agent-decision-client-release-response.mjs` 可从 testloop-mcp 仓库复制。
- `assert-release-response.mjs` 是接入方自己的断言脚本，用来固定 `agent_next_step=ready`、`fixture_count=8` 和决策序列。
- `testloop-release-response.json` 是 renderer 输出，可以作为 CI artifact 上传给 Agent。

## 可复制命令

维护者可以先用内置 showcase 生成一个临时客户端项目：

```bash
scripts/showcase-agent-decision-client-release-response-smoke.sh --json
```

该命令会创建临时 Node 项目，复制 summary 和 renderer，并运行临时项目自己的：

```bash
npm test --silent
```

通过态 JSON 固定包含：

- `status=passed`
- `release_ref=v0.5.19`
- `fixture_count=8`
- `agent_next_step=ready`
- `npm_exit_code=0`
- `release_summary_json`
- `agent_response_json`

如果要把最小客户端包导出到指定目录，直接运行：

```bash
node scripts/export-agent-decision-release-response-client.mjs /tmp/testloop-release-response-client
```

导出目录可直接执行：

```bash
cd /tmp/testloop-release-response-client
npm test --silent
```

之后把 `testloop-release-smoke-summary.json` 替换成真实 `scripts/showcase-agent-decision-client-release-smoke.sh --json` 输出即可。导出包也会携带 release response schema 和通过/失败态 fixture，方便接入方把这些样例放进自己的单元测试。

如果要直接把 release response 客户端包和 GitHub Actions workflow 安装到真实外部仓库，运行：

```bash
scripts/install-agent-decision-release-response-client.sh /absolute/path/to/client-repo
```

如果已有正式 release smoke summary，可以指定输入：

```bash
scripts/install-agent-decision-release-response-client.sh --summary-json /path/to/release-smoke-summary.json /absolute/path/to/client-repo
```

安装脚本会写入 `testloop-release-response-client/` 和 `.github/workflows/testloop-release-response-contract.yml`，然后在目标包目录执行 `npm test --silent`。默认不会覆盖已有 workflow 或包目录；需要覆盖时显式传 `--force`。通过态输出会给出 `workflow_path`、`package_dir`、`agent_response_json`、`release_ref`、`fixture_count` 和 `agent_next_step=ready`，方便 Agent 直接判断接入是否可提交。

安装脚本的 JSON summary 结构见 [agent-decision-release-response-client-install-summary.schema.json](./fixtures/agent-decision-release-response-client-install-summary.schema.json)，通过态样例见 [passed.json](./fixtures/agent-decision-release-response-client-install-summary/passed.json)。接入方可以把安装输出保存后运行：

```bash
node scripts/validate-agent-decision-release-response-client-install-summary.mjs /path/to/install-summary.json
```

如果要验证外部仓库的 CI 形态，可以运行：

```bash
scripts/showcase-agent-decision-client-release-response-ci.sh --json
```

这个 showcase 会创建临时外部客户端仓库，写入 `.github/workflows/testloop-release-response-contract.yml`，导出 release response 客户端包，并按 workflow 的核心命令执行 `npm test --silent`。通过态表示接入方可以把导出包目录提交到自己的仓库，并用同一条 workflow 做 release response contract smoke。

如果已有 release smoke summary，可以复用它：

```bash
TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_SUMMARY_JSON=/path/to/release-smoke-summary.json \
  scripts/showcase-agent-decision-client-release-response-smoke.sh --json
```

如果要避免网络下载，测试环境可以把 installer 改成本地 file URL：

```bash
TESTLOOP_AGENT_DECISION_RELEASE_INSTALLER_URL="file://${PWD}/scripts/install-agent-decision-client-ci-template.sh" \
  scripts/showcase-agent-decision-client-release-response-smoke.sh --json
```

正式发布后默认会使用 release tag raw installer。这个路径更接近真实接入方，但会受 `raw.githubusercontent.com` 网络抖动影响。

## package.json

临时客户端使用的最小脚本如下：

```json
{
  "name": "testloop-agent-decision-release-response-client",
  "private": true,
  "type": "module",
  "scripts": {
    "test": "node scripts/render-agent-decision-client-release-response.mjs --json testloop-release-smoke-summary.json > testloop-release-response.json && node scripts/assert-release-response.mjs testloop-release-response.json"
  }
}
```

接入方可以直接把这条 `npm test --silent` 放进自己的 CI。CI 失败时，把 `testloop-release-response.json` 作为 artifact 上传给 Agent。

## Agent 分流

`render-agent-decision-client-release-response.mjs` 会输出这些稳定动作：

| agent_next_step | 含义 |
| --- | --- |
| `ready` | release installer、基础客户端 response、consumer response 和 fixture 决策序列都通过。 |
| `inspect-release-installer` | `installer_url`、`helper_refs.install` 或 `helper_refs.consumer` 与 release ref 不一致。 |
| `inspect-release-client-response` | 基础客户端 response 没有得到 `ready`。 |
| `inspect-release-consumer-response` | consumer smoke response 没有得到 `ready`。 |
| `inspect-agent-decision-fixtures` | `fixture_count` 或 `decisions[]` 漂移。 |
| `inspect-release-smoke-summary` | summary 缺失、结构错误或其他不可归类失败。 |

客户端不要解析日志文本来判断成功。只读取 renderer JSON 的：

- `status`
- `agent_next_step`
- `evidence.release_ref`
- `evidence.helper_refs`
- `evidence.fixture_count`
- `evidence.decisions`
- `evidence.agent_next_steps`
- `failures[]`

通过态和失败态 fixture 见：

- [passed.json](./fixtures/agent-decision-client-release-response/passed.json)
- [installer-drift.json](./fixtures/agent-decision-client-release-response/installer-drift.json)
- [client-response-drift.json](./fixtures/agent-decision-client-release-response/client-response-drift.json)
- [consumer-response-drift.json](./fixtures/agent-decision-client-release-response/consumer-response-drift.json)
- [fixture-drift.json](./fixtures/agent-decision-client-release-response/fixture-drift.json)

## 建议 CI artifact

失败时建议上传：

- `testloop-release-smoke-summary.json`
- `testloop-release-response.json`
- `package.json`
- `scripts/assert-release-response.mjs`

这些文件足够 Agent 判断是 release installer 漂移、客户端 response 漂移，还是 fixture 合同漂移。

## 回归入口

仓库内保护这条路径的测试是：

```bash
sh test/agent_decision_client_release_response_smoke_test.sh
```

该测试覆盖：

- help 输出。
- fixture summary 直接消费。
- 实时 file installer summary 消费。
- 失败 summary 的非 0 退出码。

相关上游路径：

- [客户端集成说明](./client-integration.md)
- [MCP 客户端契约测试说明](./mcp-client-contract-tests.md)
- [Agent 决策客户端 CI 模板](./agent-decision-client-ci-template.md)
- [真实结构化 fixture](./fixtures.md)
