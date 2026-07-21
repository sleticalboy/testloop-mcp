# Agent 决策客户端 CI 接入 Checklist

这份 checklist 面向 MCP 客户端、编辑器插件和 AI Coding Agent 集成方。目标是用最短路径把 testloop-mcp 的 `status/action -> decision` 合同放进客户端仓库 CI。

## 接入步骤

1. 选择 helper ref。
   - 默认推荐使用当前稳定 ref：`v0.5.16`。
   - 需要验证下一版变更时，再显式传入其他 tag 或临时 ref。

2. 生成 GitHub Actions workflow。

```bash
curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/install-agent-decision-client-ci-template.sh -o /tmp/install-testloop-agent-decision-ci.sh
bash /tmp/install-testloop-agent-decision-ci.sh /absolute/path/to/client-repo
```

默认会写入：

```text
.github/workflows/testloop-agent-decision-contract.yml
```

已有文件不会被覆盖；需要覆盖时显式加 `--force`。

3. 提交 workflow 后运行 CI。

CI 会 checkout `sleticalboy/testloop-mcp` helper，运行：

```bash
.testloop-mcp/scripts/showcase-agent-decision-client-ci.sh --json
```

成功时 summary JSON 应满足：

```json
{
  "status": "passed",
  "fixture_count": 8,
  "decisions": ["accept", "accept", "accept", "manual-review", "manual-review", "manual-review", "apply-repair", "needs-better-input"],
  "failures": [],
  "validator_exit_code": 0
}
```

4. 保存 artifact。

模板默认上传：

```text
/tmp/testloop-agent-decision-client-summary.json
/tmp/testloop-agent-decision-client/agent-decision-fixtures-result.json
/tmp/testloop-agent-decision-client/testloop-agent-decision-fixtures/package.json
/tmp/testloop-agent-decision-client/testloop-agent-decision-fixtures/docs/fixtures/agent-decision-fixtures.json
```

5. 客户端分流逻辑必须由 manifest 驱动。

读取 `agent-decision-fixtures.json` 的 `fixtures[].expected_decision`，不要硬编码文件名或 glob 顺序。当前最小动作集合：

| status/action | expected_decision |
| --- | --- |
| `passed/ready` | `accept` |
| `passed/manual_review_internal` | `manual-review` |
| `passed/manual_review_environment` | `manual-review` |
| `failed/manual_review_external_service` | `manual-review` |
| `failed/apply_fix_suggestions` | `apply-repair` |
| `failed/needs_better_input` | `needs-better-input` |

## 本地验收

维护者可运行完整安装 dry-run：

```bash
scripts/showcase-agent-decision-client-ci-template-install.sh --json
```

该命令会下载或读取 installer，生成 workflow，模拟 `.testloop-mcp` helper checkout，并执行 Agent 决策 fixture contract。JSON 输出结构见 [Agent 决策客户端 CI 模板安装 summary schema](./fixtures/agent-decision-client-ci-template-install-summary.schema.json)，通过态样例见 [passed.json](./fixtures/agent-decision-client-ci-template-install-summary/passed.json)。

## 失败排查

- `status=failed` 且 `failures[]` 非空：先读 `agent-decision-fixtures-result.json`，确认是 manifest、fixture 内容还是客户端期望漂移。
- `validator_exit_code` 非 0：优先检查 Node/npm 是否可用，以及导出包内 `npm test --silent` 输出。
- helper checkout 失败：确认 workflow 中 `repository: sleticalboy/testloop-mcp` 和 `ref: v0.5.16` 是否可访问。
- 不要把 `manual_review_*` 当成自动修复入口；只有 `failed/apply_fix_suggestions` 才进入 repair task 闭环。

完整背景见 [Agent 决策客户端 CI 模板](./agent-decision-client-ci-template.md)、[客户端集成说明](./client-integration.md) 和 [MCP 客户端契约测试说明](./mcp-client-contract-tests.md)。
