# v0.5.18 发布说明

## 标题

testloop-mcp v0.5.18

## 发布状态

- [x] 创建 v0.5.18 候选发布说明草案。
- [x] 梳理 v0.5.17 之后围绕 Agent 决策客户端消费端 smoke、summary schema/sample、无依赖 validator 和文档同步的改动边界。
- [x] 最新已完成的远端 CI：`f501697` run `29817455457` passed，覆盖消费端 smoke 文档同步。
- [x] 正式版本准备文件已更新：implementation version、`CHANGELOG.md` 正式版本段和当前安装/接入文档版本引用同步到 `0.5.18` / `v0.5.18`。
- [x] 正式版本准备后的 release readiness 已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.18-release-prep-dist scripts/verify-release-candidate.sh v0.5.18` 输出 `release_candidate_status=passed`，`testloop-mcp --version` 输出 `testloop-mcp 0.5.18`。
- [ ] `v0.5.18` tag 已推送，Release Artifacts workflow 已完成。
- [ ] GitHub Release 正文已更新为正式 v0.5.18 发布说明。
- [ ] 仓库内 Homebrew Formula 和 `sleticalboy/homebrew-tap` 已更新到 `0.5.18`。
- [ ] Post-Release Verify 已通过。

## 摘要

v0.5.18 继续强化“面向 AI 编程代理的测试反馈闭环 MCP 服务”的客户端接入层。这个版本不扩语言，也不改变测试生成算法；重点是把 v0.5.17 的 installer / 安装 dry-run 继续推进到“接入方能稳定消费 artifact”的发布边界。

v0.5.17 解决的是“客户端仓库可以安装 workflow，并校验安装 dry-run summary”。v0.5.18 解决的是“客户端仓库可以把安装后的 summary、导出的 fixture manifest 和 `agent-decision-fixtures-result.json` 作为一条端到端消费合同验证”。

## 主要变化

### 消费端 smoke

- 新增 `scripts/showcase-agent-decision-client-consumer-smoke.sh`。
- 脚本会创建临时外部 client，使用本仓库 installer 生成 workflow，运行 helper dry-run。
- 在安装 dry-run 基础上继续校验安装 summary、导出的 fixture manifest 和 `agent-decision-fixtures-result.json` 互相一致。
- 支持 `--json`，输出 client 目录、workflow 路径、helper ref、summary 路径、fixture 路径、result JSON 路径、fixture 数量、决策序列和 validator 退出码。

### JSON 摘要契约

- 新增 `docs/fixtures/agent-decision-client-consumer-smoke-summary.schema.json`。
- 新增 `docs/fixtures/agent-decision-client-consumer-smoke-summary/passed.json` 通过态样例。
- 新增 `test/agent_decision_client_ci_consumer_smoke_summary_schema_test.sh`，同时校验实时 smoke 输出、schema 和通过态样例。

### 无依赖 validator

- 新增 `scripts/validate-agent-decision-client-consumer-smoke-summary.mjs`。
- validator 支持默认校验通过态 fixture，也可指定任意消费端 smoke summary JSON。
- 支持文本输出和 `--json` 输出。
- 固定 `helper_ref=v0.5.18`、`fixture_count=8`、决策序列、空 `failures[]`、安装 summary validator 退出码、fixture validator 退出码和 npm validator 退出码。

### 文档同步

- README、[Agent 决策客户端 CI 接入 Checklist](./agent-decision-client-ci-checklist.md)、[真实结构化 fixture](./fixtures.md)、[客户端集成说明](./client-integration.md)、[MCP 客户端契约测试说明](./mcp-client-contract-tests.md) 和 [Agent 决策客户端 CI 模板](./agent-decision-client-ci-template.md) 已同步消费端 smoke、summary schema/sample 和 validator 入口。
- 默认 CI 已加入消费端 smoke、summary schema 测试和 summary validator 测试。

## 质量边界

- 当前重点仍是 Agent 客户端测试反馈基础设施，不是“更会自动写单测”。
- 消费端 smoke 面向通过态接入合同；如果任一 summary、manifest 或 result JSON 漂移，validator 应失败并让客户端 CI 停下。
- 模板和 installer 仍默认固定到稳定 helper tag，避免客户端 CI 跟随 `main` 漂移。
- `failed/manual_review_*` 仍不进入自动修复循环。客户端应按 manifest 的 `expected_decision` 分流。

## 推荐验证

- `sh test/agent_decision_client_ci_consumer_smoke_test.sh`
- `sh test/agent_decision_client_ci_consumer_smoke_summary_schema_test.sh`
- `sh test/agent_decision_client_ci_consumer_smoke_summary_validator_test.sh`
- `sh test/agent_decision_client_ci_checklist_doc_test.sh`
- `sh test/agent_decision_client_ci_checklist_commands_test.sh`
- `sh test/agent_decision_client_ci_template_doc_test.sh`
- `sh test/client_integration_doc_test.sh`
- `sh test/mcp_client_contract_doc_test.sh`
- `sh test/fixtures_index_test.sh`
- `sh test/release_doc_index_test.sh`
- `sh test/docs_links_test.sh`
- `sh test/ci_workflow_test.sh`
- `for t in test/*_test.sh; do sh "$t"; done`
- `go test ./...`
- `git diff --check`
- `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.18-release-prep-dist scripts/verify-release-candidate.sh v0.5.18`

## 发布备注

- 对外文案应强调“外部 MCP 客户端可以用一条消费端 smoke 固定安装 summary、fixture manifest 和 result JSON 的 artifact 消费合同”。
- 推荐演示路径：运行 `scripts/showcase-agent-decision-client-consumer-smoke.sh --json`，再运行 `node scripts/validate-agent-decision-client-consumer-smoke-summary.mjs /path/to/consumer-smoke-summary.json`。
- 正式发布前仍需完成 Release assets、Homebrew Formula / tap 和 Post-Release Verify。
