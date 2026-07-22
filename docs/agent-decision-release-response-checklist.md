# Agent 决策 release response 接入 Checklist

这份 checklist 面向已经接入基础 Agent 决策 fixture CI 的外部客户端。目标是把发布后 `release smoke summary -> release response -> agent_next_step` 的消费链路放进客户端仓库 CI。

## 接入步骤

1. 生成 release smoke summary。

```bash
scripts/showcase-agent-decision-client-release-smoke.sh --json > /tmp/testloop-release-smoke-summary.json
```

正式发布后建议使用 release tag raw installer 的默认路径；本地或 CI 网络不稳定时，可用 `TESTLOOP_AGENT_DECISION_RELEASE_INSTALLER_URL=file://...` 做离线验证。

2. 安装 release response 客户端包和 workflow。

```bash
scripts/install-agent-decision-release-response-client.sh --summary-json /tmp/testloop-release-smoke-summary.json --json /absolute/path/to/client-repo > /tmp/testloop-release-response-install-summary.json
```

默认写入：

```text
testloop-release-response-client/
.github/workflows/testloop-release-response-contract.yml
```

已有 workflow 或包目录不会被覆盖；需要覆盖时显式加 `--force`。

3. 校验安装 summary。

```bash
node scripts/validate-agent-decision-release-response-client-install-summary.mjs /tmp/testloop-release-response-install-summary.json
```

通过态 summary 应满足：

```json
{
  "status": "written",
  "release_ref": "v0.5.20",
  "fixture_count": 8,
  "agent_next_step": "ready",
  "npm_exit_code": 0,
  "failures": []
}
```

结构契约见 [agent-decision-release-response-client-install-summary.schema.json](./fixtures/agent-decision-release-response-client-install-summary.schema.json)，通过态样例见 [passed.json](./fixtures/agent-decision-release-response-client-install-summary/passed.json)。

4. 本地复验导出包。

```bash
cd /absolute/path/to/client-repo/testloop-release-response-client
npm test --silent
```

该命令会读取 `testloop-release-smoke-summary.json`，生成 `testloop-release-response.json`，并断言 `agent_next_step=ready`。

5. 提交 workflow 后运行客户端 CI。

workflow 会运行：

```bash
cd testloop-release-response-client
npm test --silent
```

失败时默认上传：

```text
testloop-release-response-client/testloop-release-smoke-summary.json
testloop-release-response-client/testloop-release-response.json
testloop-release-response-client/package.json
testloop-release-response-client/docs/fixtures/agent-decision-client-release-response.schema.json
testloop-release-response-client/docs/fixtures/agent-decision-client-release-response/*.json
```

## Agent 分流

客户端不要解析日志文本。只读取 `testloop-release-response.json` 的：

- `status`
- `agent_next_step`
- `evidence.release_ref`
- `evidence.fixture_count`
- `evidence.decisions`
- `evidence.agent_next_steps`
- `failures[]`

稳定动作：

| agent_next_step | 处理方式 |
| --- | --- |
| `ready` | 接受 release response 接入结果，继续发布或提交。 |
| `inspect-release-installer` | 检查 release ref、installer URL 和 helper refs 是否一致。 |
| `inspect-release-client-response` | 检查基础客户端 Agent response 是否仍为 `ready`。 |
| `inspect-release-consumer-response` | 检查 consumer smoke response 是否仍为 `ready`。 |
| `inspect-agent-decision-fixtures` | 检查 `fixture_count` 和 `decisions[]` 是否漂移。 |
| `inspect-release-smoke-summary` | 检查 summary 是否缺失、损坏或结构不兼容。 |

## 回归入口

维护者可用临时外部仓库验证 CI 形态：

```bash
scripts/showcase-agent-decision-client-release-response-ci.sh --json
```

如果要验证接入方仓库照抄样板，可以运行：

```bash
scripts/showcase-release-response-adopter.sh --json
```

该入口会把 [Release response 接入方样板](../examples/release-response-adopter/README.md) 复制到临时外部仓库，确认 installer、workflow、`npm test --silent` 和接入方消费 helper 都能跑通。
如果要把该 summary 纳入机器校验，保存 `--json` 输出后运行 `node scripts/validate-release-response-adopter-summary.mjs /path/to/release-response-adopter-summary.json`。结构契约见 [release-response-adopter-summary.schema.json](./fixtures/release-response-adopter-summary.schema.json)，通过态样例见 [passed.json](./fixtures/release-response-adopter-summary/passed.json)。
该入口默认生成 `testloop-release-response-adopter-artifacts/`；输出目录可用 `TESTLOOP_RELEASE_RESPONSE_ADOPTER_ARTIFACT_DIR` 覆盖。外部 CI 建议直接上传这个目录，至少包含 adopter summary、install summary、`testloop-release-response-client/testloop-release-response.json` 和两个 helper 的 `--json` 输出，方便 Agent 离线判断是 installer、renderer 还是接入方消费 helper 失败。
下载 artifact 后可运行 `node scripts/verify-release-response-adopter-artifact.mjs /path/to/testloop-release-response-adopter-artifacts` 做离线自检；通过态固定 `release_response_adopter_artifact_status=passed`、`agent_next_step=ready` 和 `should_accept=true`。

如果只想导出最小包：

```bash
node scripts/export-agent-decision-release-response-client.mjs /tmp/testloop-release-response-client
```

如果要在发版前跑完整本地门禁：

```bash
scripts/verify-release-candidate.sh v0.5.20
```

release readiness 会同时验证 release response 导出包和真实仓库安装 summary。

完整背景见 [Agent 决策 release response 客户端接入](./agent-decision-release-response-client.md)、[客户端集成说明](./client-integration.md) 和 [Agent 决策客户端 CI 接入 Checklist](./agent-decision-client-ci-checklist.md)。
