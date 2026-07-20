# v0.5.15 候选发布检查清单

## 当前目标

这是 v0.5.15 的候选发布检查清单。目标是把 v0.5.14 之后围绕 Agent 决策 fixture、manifest 驱动客户端契约、JSON validator、最小导出包和 release readiness 门禁的改动整理成一个可发布边界。

发布重点见 [v0.5.15 候选发布说明](./plan-release-notes-v0.5.15.md)。

当前发布状态：候选边界整理中。尚未更新 implementation version、尚未收敛 `CHANGELOG.md` 正式版本段、尚未打 tag、尚未创建 GitHub Release、尚未更新 Homebrew tap。

## 当前差异核对

- [x] 新增真实项目 Agent 闭环 fixture，覆盖 Go ready、Vitest ready、Python environment 手审和 Python external-service 失败手审。
- [x] 新增 `docs/fixtures/agent-decision-fixtures.json` 和 schema。
- [x] `examples/agent-decision-demo` 已改为读取 manifest。
- [x] 新增 `scripts/validate-agent-decision-fixtures.mjs`，支持文本和 `--json` 输出。
- [x] validator 已校验 manifest 元数据和关键 payload 结构。
- [x] 新增 `scripts/export-agent-decision-fixtures.mjs`，导出可复制最小 fixture 包。
- [x] 导出包包含无依赖 `package.json`，可运行 `npm test --silent`。
- [x] `scripts/verify-release-candidate.sh` 已显式校验 Agent 决策 fixture 导出包。
- [x] README、客户端集成说明、MCP 客户端契约测试说明和 roadmap 已同步。

## 候选内容

- [x] 客户端可以从 manifest 读取全部最小 Agent 决策 fixture，而不是维护 glob 或手写白名单。
- [x] 客户端可以用 JSON validator 机器断言 `status/action -> decision` 合同。
- [x] 客户端可以复制最小 fixture 包，并用 `npm test --silent` 加入自己的 CI。
- [x] release readiness 会覆盖导出包，不再只依赖普通 shell 子测试间接验证。
- [x] 文档明确 `failed/manual_review_*` 不进入自动修复循环。

## 已验证

- [x] `sh test/agent_decision_fixture_validator_test.sh`
- [x] `sh test/agent_decision_fixture_export_test.sh`
- [x] `sh test/agent_decision_fixtures_manifest_test.sh`
- [x] `sh test/agent_decision_demo_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/mcp_client_contract_doc_test.sh`
- [x] `sh test/release_candidate_script_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/ci_workflow_test.sh`
- [x] `go test ./...`
- [x] `git diff --check`
- [x] `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.15-candidate-dist scripts/verify-release-candidate.sh v0.5.15` 输出 `release_candidate_status=passed`，导出包 step 输出 `fixture_count=8`。
- [x] `da69566` 远端 CI run `29748774219` passed，覆盖 Agent 决策 validator JSON 输出。
- [x] `0868676` 远端 CI run `29749188084` passed，覆盖 Agent 决策 fixture 导出包。
- [x] `d79b720` 远端 CI run `29749432210` passed，覆盖导出包 `package.json` / `npm test --silent`。
- [x] `85ce335` 远端 CI run `29749842485` passed，覆盖 manifest 元数据校验。
- [x] `153574f` 远端 CI run `29750125793` passed，覆盖 release readiness 显式校验 Agent 决策 fixture 导出包。

## 发布前门禁

- [ ] 等候选边界整理后的最新 main CI 通过。
- [ ] `scripts/verify-release-candidate.sh v0.5.15`
- [ ] `git diff --check`
- [ ] 确认 `testloop-mcp --version` 已在正式版本准备后输出 `testloop-mcp 0.5.15`。

## 正式发布前待办

- [ ] 更新 `main.go` MCP implementation version 到 `0.5.15`。
- [ ] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.15 - 2026-07-20`。
- [ ] 同步 README、installation、quickstart 和必要版本引用到 `0.5.15` / `v0.5.15`。
- [ ] 测试中的版本期望同步到 `0.5.15`。
- [ ] 重新运行完整本地验证，确认版本准备改动可发布。
- [ ] 提交版本准备改动后确认远端 CI passed。
- [ ] 打 tag `v0.5.15` 并推送。
- [ ] 等 Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.15` 验证 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.15 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.15` 更新仓库内 Formula。
- [ ] 更新 Homebrew tap 到 `0.5.15` 并推送。
- [ ] 手动触发 Post-Release Verify。

## 当前结论

v0.5.15 已具备候选范围草案和本地 dry-run 证据，但尚未进入正式版本准备。下一步应提交候选文档并等待 main CI；CI 通过后，再决定是否进入正式发布流程。
