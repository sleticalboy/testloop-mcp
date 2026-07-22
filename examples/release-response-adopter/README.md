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
