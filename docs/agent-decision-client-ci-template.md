# Agent 决策客户端 CI 模板

这份模板面向 MCP 客户端、编辑器插件和 AI Coding Agent 集成方。它不验证用户项目构建，而是验证客户端能稳定消费 testloop-mcp 的 Agent 决策 fixture 包：导出最小 fixture 包，运行包内 `npm test --silent`，并断言 `status/action -> decision` 合同没有漂移。

适合放在客户端仓库的 smoke job 中，尤其是这些场景：

- 客户端实现了 `validate_coverage_task`、真实项目 summary 或 CI artifact 的分流逻辑。
- 客户端需要固定 `accept`、`manual-review`、`apply-repair`、`needs-better-input` 四类机器动作。
- 客户端不想复制整个 testloop-mcp 仓库，只想用最小 fixture 包做契约测试。

当前模板 checkout `main` 分支上的 helper，因为 `scripts/showcase-agent-decision-client-ci.sh` 是 v0.5.15 之后新增的接入脚本。后续发布正式版本后，接入方可以把 `ref: main` 固定到对应 tag。

## GitHub Actions 模板

保存为 `.github/workflows/testloop-agent-decision-contract.yml`：

```yaml
name: testloop agent decision contract

on:
  workflow_dispatch:
  pull_request:

jobs:
  agent-decision-contract:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: 22

      - name: Checkout testloop-mcp fixture helpers
        uses: actions/checkout@v4
        with:
          repository: sleticalboy/testloop-mcp
          ref: main
          path: .testloop-mcp

      - name: Verify Agent decision fixture contract
        run: |
          TESTLOOP_AGENT_DECISION_CLIENT_DIR=/tmp/testloop-agent-decision-client \
            .testloop-mcp/scripts/showcase-agent-decision-client-ci.sh --json \
            | tee /tmp/testloop-agent-decision-client-summary.json

      - name: Upload Agent decision result
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: testloop-agent-decision-contract
          path: |
            /tmp/testloop-agent-decision-client-summary.json
            /tmp/testloop-agent-decision-client/agent-decision-fixtures-result.json
            /tmp/testloop-agent-decision-client/testloop-agent-decision-fixtures/package.json
            /tmp/testloop-agent-decision-client/testloop-agent-decision-fixtures/docs/fixtures/agent-decision-fixtures.json
```

## 正常输出

成功时 `Verify Agent decision fixture contract` step 会输出：

```json
{
  "status": "passed",
  "fixture_count": 8,
  "decisions": ["accept", "accept", "accept", "manual-review", "manual-review", "manual-review", "apply-repair", "needs-better-input"],
  "failures": [],
  "validator_exit_code": 0
}
```

如果失败，先下载 `testloop-agent-decision-contract` artifact，查看 `testloop-agent-decision-client-summary.json` 和 `agent-decision-fixtures-result.json` 的 `failures[]`。失败通常意味着客户端同步的 fixture、manifest 元数据或 `manual_review_*` 分流语义已经漂移。

## 本地 dry-run

维护者可以用仓库内回归测试模拟外部客户端仓库的关键路径：

```bash
sh test/agent_decision_client_ci_template_dry_run_test.sh
```

这个测试会创建临时客户端目录，把当前仓库挂成 `.testloop-mcp` helper checkout，按模板中的相对路径运行 `.testloop-mcp/scripts/showcase-agent-decision-client-ci.sh --json | tee ...`，并验证 `testloop-agent-decision-client-summary.json`、`agent-decision-fixtures-result.json`、导出包 `package.json` 和 manifest 都真实存在。

更多背景见 [客户端集成说明](./client-integration.md) 和 [MCP 客户端契约测试说明](./mcp-client-contract-tests.md)。
