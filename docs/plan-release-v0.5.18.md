# v0.5.18 发布检查清单

## 当前目标

这是 v0.5.18 的候选发布检查清单。目标是把 v0.5.17 之后围绕 Agent 决策客户端消费端 smoke、summary schema/sample、无依赖 validator 和文档同步的改动整理成一个可发布边界，并为正式发布、资产校验和 Homebrew 分发更新做准备。

发布重点见 [v0.5.18 发布说明](./plan-release-notes-v0.5.18.md)。

当前发布状态：候选边界已整理，尚未正式发布。`v0.5.18` tag、GitHub Release、Release assets、Homebrew Formula / tap 和 Post-Release Verify 尚未执行。

## 当前差异核对

- [x] 新增 `scripts/showcase-agent-decision-client-consumer-smoke.sh`，用临时外部 client 串起 workflow 安装、helper dry-run、安装 summary 校验、导出 fixture manifest 校验和 result JSON 消费检查。
- [x] 新增 `docs/fixtures/agent-decision-client-consumer-smoke-summary.schema.json`，固定消费端 smoke 的 JSON 输出字段。
- [x] 新增 `docs/fixtures/agent-decision-client-consumer-smoke-summary/passed.json`，提供通过态 golden sample。
- [x] 新增 `scripts/validate-agent-decision-client-consumer-smoke-summary.mjs`，提供无依赖消费端 smoke summary 校验入口。
- [x] 新增消费端 smoke、summary schema 和 summary validator 回归测试。
- [x] 默认 CI 已加入消费端 smoke、summary schema 和 summary validator 测试。
- [x] README、Agent 决策客户端 CI Checklist、fixtures 索引、客户端集成说明、MCP 客户端契约测试说明、Agent 决策客户端 CI 模板、CHANGELOG 和 roadmap 已同步。

## 候选内容

- [x] 维护者可以用 `scripts/showcase-agent-decision-client-consumer-smoke.sh --json` 验证接入方能否稳定消费安装后的 artifact 链路。
- [x] 客户端可以用 schema、passed sample 或无依赖 validator 固定消费端 smoke JSON 输出。
- [x] 接入方文档已经说明从安装 dry-run 继续升级到消费端 smoke 的路径。
- [x] 当前版本边界明确：不扩语言、不改测试生成算法，只强化客户端 artifact 消费合同。

## 已验证

- [x] `sh test/agent_decision_client_ci_consumer_smoke_test.sh`
- [x] `sh test/agent_decision_client_ci_consumer_smoke_summary_schema_test.sh`
- [x] `sh test/agent_decision_client_ci_consumer_smoke_summary_validator_test.sh`
- [x] `sh test/agent_decision_client_ci_checklist_doc_test.sh`
- [x] `sh test/agent_decision_client_ci_checklist_commands_test.sh`
- [x] `sh test/agent_decision_client_ci_template_doc_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/mcp_client_contract_doc_test.sh`
- [x] `sh test/fixtures_index_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/ci_workflow_test.sh`
- [x] `for t in test/*_test.sh; do sh "$t"; done`
- [x] `go test ./...`
- [x] `git diff --check`
- [x] `2bc1c6d` 远端 CI run `29816097350` passed，覆盖消费端 smoke。
- [x] `22404ab` 远端 CI run `29816486971` passed，覆盖消费端 smoke summary schema/sample。
- [x] `dd68e98` 远端 CI run `29816908894` passed，覆盖消费端 smoke summary validator。
- [x] `f501697` 远端 CI run `29817455457` passed，覆盖消费端 smoke 文档同步。

## 发布前门禁

- [ ] 候选边界整理提交后的 main CI passed。
- [ ] `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.18-release-prep-dist scripts/verify-release-candidate.sh v0.5.18`
- [ ] `git diff --check`
- [ ] `main.go` implementation version 更新到 `0.5.18`。

## 正式发布前待办

- [ ] 更新 `main.go` MCP implementation version 到 `0.5.18`。
- [ ] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.18 - 2026-07-21`。
- [ ] 同步 README、installation、quickstart 和必要版本引用到 `0.5.18` / `v0.5.18`。
- [ ] 测试中的版本期望同步到 `0.5.18`。
- [ ] 重新运行完整 release readiness。
- [ ] 提交版本准备改动后确认远端 CI passed。
- [ ] 打 tag `v0.5.18` 并推送。
- [ ] 等 Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.18` 验证 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.18 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.18` 更新仓库内 Formula。
- [ ] 更新 Homebrew tap 到 `0.5.18` 并推送。
- [ ] 手动触发 Post-Release Verify。

## 当前结论

v0.5.18 候选边界适合作为一个小版本发布：它不承诺生成质量提升，而是把外部客户端 CI 接入路径从“installer + 安装 dry-run summary validator”推进到“消费端 smoke + summary schema/sample + 无依赖 validator + 客户端文档入口”，更贴合项目“AI Agent 测试反馈基础设施”的定位。
