# Agent decision client 接入方样板

这个目录提供一份可复制到外部客户端仓库的最小样板，用来消费 testloop-mcp 基础 Agent decision client response。它不涉及 release 流程，只验证“导出 fixture 包 -> 运行客户端契约 -> 渲染 Agent response -> 接入方读取 `agent_next_step`”这条基础闭环。

## 需要复制的文件

```text
README.md
scripts/read-testloop-agent-decision-response.mjs
```

## 安装到客户端仓库

在 `testloop-mcp` 仓库 checkout 中运行：

```bash
node scripts/export-agent-decision-fixtures.mjs \
  /absolute/path/to/client-repo/testloop-agent-decision-fixtures
```

然后复制接入方消费 helper：

```bash
mkdir -p /absolute/path/to/client-repo/scripts
cp examples/agent-decision-client-adopter/scripts/read-testloop-agent-decision-response.mjs \
  /absolute/path/to/client-repo/scripts/read-testloop-agent-decision-response.mjs
```

## 本地验证

在客户端仓库内运行：

```bash
cd testloop-agent-decision-fixtures
npm test --silent > ../agent-decision-fixtures-result.json
npm run render:client-response --silent > ../testloop-agent-decision-client-response.json
npm run validate:client-response --silent
cd ..
node scripts/read-testloop-agent-decision-response.mjs \
  testloop-agent-decision-client-response.json
```

通过态输出应包含：

```text
testloop_agent_decision_response_status=passed
testloop_agent_decision_response_next_step=ready
testloop_agent_decision_response_fixture_count=8
testloop_agent_decision_response_should_accept=true
```

维护 testloop-mcp 仓库时，也可以直接验证完整接入样板：

```bash
scripts/showcase-agent-decision-client-adopter.sh --json \
  > /tmp/testloop-agent-decision-client-adopter-summary.json
```

失败态会返回非 0。Agent 应读取 `testloop_agent_decision_response_next_step` 和 `testloop_agent_decision_response_failures`，再决定排查 summary、validator，还是 fixture 漂移。

## Helper 输出字段

| 字段 | 来源 | Agent 动作 |
| --- | --- | --- |
| `testloop_agent_decision_response_status` | response `status` | 判断基础 response 是否通过。 |
| `testloop_agent_decision_response_next_step` | response `agent_next_step` | 主分流字段；`ready` 才接受结果。 |
| `testloop_agent_decision_response_fixture_count` | `evidence.fixture_count` | 核对 fixture 数量是否漂移。 |
| `testloop_agent_decision_response_validator_exit_code` | `evidence.validator_exit_code` | 非 0 时先排查客户端 validator。 |
| `testloop_agent_decision_response_should_accept` | helper 归一化结果 | `true` 才接受结果。 |
| `testloop_agent_decision_response_failures` | `failures[]` | 失败时交给 Agent 排查。 |

稳定 `agent_next_step` 值：

| agent_next_step | Agent 动作 |
| --- | --- |
| `ready` | 接受基础 Agent decision client response。 |
| `inspect-client-validator` | 检查导出包 validator 或运行时退出码。 |
| `inspect-agent-decision-fixtures` | 检查 fixture 数量或决策序列漂移。 |
| `inspect-agent-decision-client-summary` | 检查缺失、损坏或结构不兼容的 client CI summary。 |

消费 helper 不解析日志，只读取 `render-agent-decision-client-ci-response.mjs --json` 生成的 JSON response。
