# Release response 接入方样板

这个目录提供一份可复制到外部客户端仓库的最小样板，用来消费 testloop-mcp 的 release response 证据。接入方不需要理解 testloop-mcp 内部实现，只需要读取稳定的 `agent_next_step` JSON，让 AI Agent 能判断下一步是接受结果、排查 installer，还是排查 fixture 漂移。

## 需要复制的文件

```text
README.md
scripts/read-testloop-release-response.mjs
scripts/read-testloop-release-response-summary.mjs
```

installer 会在目标客户端仓库里创建剩余文件：

```text
testloop-release-response-client/
.github/workflows/testloop-release-response-contract.yml
```

## 安装到客户端仓库

在 `testloop-mcp` 仓库 checkout 中运行：

```bash
scripts/showcase-agent-decision-client-release-smoke.sh --json > /tmp/testloop-release-smoke-summary.json
scripts/install-agent-decision-release-response-client.sh \
  --summary-json /tmp/testloop-release-smoke-summary.json \
  --json /absolute/path/to/client-repo \
  > /tmp/testloop-release-response-install-summary.json
node scripts/validate-agent-decision-release-response-client-install-summary.mjs \
  /tmp/testloop-release-response-install-summary.json
```

维护 testloop-mcp 仓库时，也可以直接验证完整接入样板 summary：

```bash
scripts/showcase-release-response-adopter.sh --json > /tmp/testloop-release-response-adopter-summary.json
node scripts/validate-release-response-adopter-summary.mjs \
  /tmp/testloop-release-response-adopter-summary.json
```

失败态可用 fixture 固定：

```bash
node scripts/validate-release-response-adopter-summary.mjs --json \
  docs/fixtures/release-response-adopter-summary/invalid-response.json
```

该命令应返回非 0；JSON 输出里的 `agent_next_step` 和 `failures[]` 可直接交给 Agent 分流。
`scripts/showcase-release-response-adopter.sh --json` 默认还会在接入方仓库下生成 `testloop-release-response-adopter-artifacts/`，其中包含总 summary、installer summary、release response、两个 helper 的 JSON 输出和 release smoke 输入。需要指定输出目录时设置 `TESTLOOP_RELEASE_RESPONSE_ADOPTER_ARTIFACT_DIR=/absolute/path/to/artifacts`。

然后复制接入方消费 helper：

```bash
mkdir -p /absolute/path/to/client-repo/scripts
cp examples/release-response-adopter/scripts/read-testloop-release-response.mjs \
  /absolute/path/to/client-repo/scripts/read-testloop-release-response.mjs
cp examples/release-response-adopter/scripts/read-testloop-release-response-summary.mjs \
  /absolute/path/to/client-repo/scripts/read-testloop-release-response-summary.mjs
```

## 本地验证

在客户端仓库内运行：

```bash
cd testloop-release-response-client
npm test --silent
cd ..
node scripts/read-testloop-release-response.mjs \
  testloop-release-response-client/testloop-release-response.json
```

通过态输出应包含：

```text
testloop_release_response_status=passed
testloop_release_response_next_step=ready
testloop_release_response_release_ref=v0.5.20
testloop_release_response_fixture_count=8
testloop_release_response_should_accept=true
```

如果要消费 showcase summary：

```bash
node scripts/read-testloop-release-response-summary.mjs \
  /tmp/testloop-release-response-adopter-summary.json
```

通过态输出应包含：

```text
testloop_release_response_summary_status=passed
testloop_release_response_summary_next_step=ready
testloop_release_response_summary_should_accept=true
```

失败态会返回非 0。Agent 应读取 `testloop_release_response_summary_next_step` 和 `testloop_release_response_summary_failures`，并停止继续发布。

## Helper 输出字段

`read-testloop-release-response.mjs` 输出字段：

| 字段 | 来源 | Agent 动作 |
| --- | --- | --- |
| `testloop_release_response_status` | `testloop-release-response.json.status` | 判断 release response 是否通过。 |
| `testloop_release_response_next_step` | `agent_next_step` | 主分流字段；`ready` 才继续发布。 |
| `testloop_release_response_release_ref` | `evidence.release_ref` | 核对 release tag。 |
| `testloop_release_response_fixture_count` | `evidence.fixture_count` | 核对 fixture 数量是否漂移。 |
| `testloop_release_response_should_accept` | helper 归一化结果 | `true` 才接受结果。 |
| `testloop_release_response_failures` | `failures[]` | 失败时交给 Agent 排查。 |

`read-testloop-release-response-summary.mjs` 输出字段：

| 字段 | 来源 | Agent 动作 |
| --- | --- | --- |
| `testloop_release_response_summary_status` | adopter summary `status` | 判断接入样板链路是否通过。 |
| `testloop_release_response_summary_next_step` | adopter summary `agent_next_step` | 主分流字段；非 `ready` 时停止发布。 |
| `testloop_release_response_summary_release_ref` | adopter summary `release_ref` | 核对 release tag。 |
| `testloop_release_response_summary_fixture_count` | adopter summary `fixture_count` | 核对 fixture 数量是否漂移。 |
| `testloop_release_response_summary_should_accept` | adopter summary `should_accept` | `false` 时停止发布。 |
| `testloop_release_response_summary_failures` | adopter summary `failures[]` | 失败时交给 Agent 排查。 |

## CI artifact 清单

`scripts/showcase-release-response-adopter.sh --json` 会默认生成一个名为 `testloop-release-response-adopter-artifacts` 的目录。外部仓库可以直接把该目录作为同名 CI artifact 上传，至少包含：

| 文件 | 作用 |
| --- | --- |
| `testloop-release-response-adopter-summary.json` | `scripts/showcase-release-response-adopter.sh --json` 输出，Agent 的总入口。 |
| `testloop-release-response-install-summary.json` | installer 输出，用来排查 workflow 或客户端包写入失败。 |
| `testloop-release-response-client/testloop-release-smoke-summary.json` | release smoke 输入，用来排查 release tag 或 helper refs 漂移。 |
| `testloop-release-response-client/testloop-release-response.json` | release response renderer 输出，用来读取 `agent_next_step`。 |
| `testloop-release-response-consumer.json` | `read-testloop-release-response.mjs --json` 输出，用来排查接入方消费 helper。 |
| `testloop-release-response-summary-consumer.json` | `read-testloop-release-response-summary.mjs --json` 输出，用来排查 summary helper。 |

如果只能上传一份文件，优先上传 `testloop-release-response-adopter-summary.json`；它包含 `install_summary_json`、`agent_response_json`、`consumer_json` 和 `summary_consumer_json` 路径，可让 Agent 继续定位缺失证据。

## Agent 契约

Agent 只读取 `testloop-release-response-client/testloop-release-response.json`，并基于这些字段分流：

- `status`
- `agent_next_step`
- `evidence.release_ref`
- `evidence.fixture_count`
- `evidence.decisions`
- `evidence.agent_next_steps`
- `failures[]`

稳定 `agent_next_step` 值：

| agent_next_step | Agent 动作 |
| --- | --- |
| `ready` | 接受 release response 契约结果。 |
| `inspect-release-installer` | 检查 release ref、installer URL 和 helper refs。 |
| `inspect-release-client-response` | 检查基础客户端 Agent response。 |
| `inspect-release-consumer-response` | 检查 consumer smoke Agent response。 |
| `inspect-agent-decision-fixtures` | 检查 fixture 数量或决策序列漂移。 |
| `inspect-release-smoke-summary` | 检查缺失、损坏或结构不兼容的 release smoke summary。 |

消费 helper 不解析日志，只读取 `npm test --silent` 生成的 JSON response。
