# v0.5.13 发布检查清单

## 当前目标

这是 v0.5.13 的候选发布检查清单。当前目标是把 v0.5.12 之后围绕 Agent action 信号、verification summary schema、Agent response artifact manifest 和客户端契约回归的改动整理成一个 patch 版本。

发布重点见 [v0.5.13 发布说明草案](./plan-release-notes-v0.5.13.md)。

当前不做正式发布动作：不改版本号、不打 tag、不更新 Homebrew tap。完成候选门禁和远端 CI 后，再进入正式发布准备。

## 当前差异核对

- [x] `generate_tests`、`run_tests`、`parse_results` 和 `cmd/testgen` 已暴露动作信号。
- [x] `fix_suggestions` 已补强常见模块解析、Python import 和编译错误分类。
- [x] `examples/mcp-client-demo` 已展示 action/category/repair task 消费路径。
- [x] 验收报告已新增独立 CLI 生成动作 smoke。
- [x] verification summary 已新增 `sections[].signals.action`。
- [x] `docs/fixtures/verification-summary.schema.json` 已加入。
- [x] first-run/onboarding Agent 回复和 wrapper artifact 已固定 `section_signal`。
- [x] agent-response artifact manifest 已暴露 `summary_schema`。
- [x] first-run/onboarding 失败 artifact fixture 已刷新为包含 `signals.action=manual_review`。
- [x] manifest 已新增 `expected_section_signals`。
- [x] manifest demo 已校验 `agent-decision.txt` 的 `decision_action`。
- [x] README、客户端集成说明、MCP 客户端契约测试说明和 fixture 文档已同步。
- [x] `CHANGELOG.md` 的 `Unreleased` 已记录候选内容。

## 候选内容

- [x] Agent action 信号贯穿生成、运行、解析、CLI 和验收报告。
- [x] 常见失败分类更贴近真实项目的“测试还没跑起来”问题。
- [x] verification summary JSON schema 和 fixture 回归。
- [x] CI artifact manifest 的 summary schema、expected section signals 和 decision action 验证。
- [x] first-run/onboarding 失败 artifact fixture 刷新。
- [x] 客户端契约文档和 demo 输出同步。

## 已验证

- [x] `sh test/onboarding_agent_response_demo_test.sh`
- [x] `sh test/first_run_agent_response_demo_test.sh`
- [x] `sh test/onboarding_artifact_fixtures_test.sh`
- [x] `sh test/first_run_artifact_fixtures_test.sh`
- [x] `sh test/run_onboarding_ci_test.sh`
- [x] `sh test/run_first_run_ci_test.sh`
- [x] `sh test/agent_response_artifact_contract_doc_test.sh`
- [x] `sh test/agent_response_artifact_manifest_test.sh`
- [x] `sh test/agent_response_manifest_demo_test.sh`
- [x] `go test ./tools -run 'TestAgentResponseArtifactManifestSchema|TestVerificationSummarySchema' -count=1`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/mcp_client_contract_doc_test.sh`
- [x] `sh test/readme_ci_snippet_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `go test ./...`
- [x] `git diff --check`
- [x] `9400f01` 远端 CI run `29695128722` passed，覆盖 manifest 暴露 summary schema。
- [x] `4526c85` 远端 CI run `29695505354` passed，覆盖 artifact section signal fixture。
- [x] `515fd00` 远端 CI run `29695649219` passed，覆盖 artifact decision action 验证。
- [x] `1395715` 远端 CI run `29695820104` passed，覆盖 v0.5.13 候选发布边界文档。
- [x] 候选 release readiness 已通过：shell 语法、`go test ./...`、全部 `test/*_test.sh`、候选二进制构建、help/version、darwin arm64 打包 dry-run、sha256 和 tarball 内容检查。

## 发布前门禁

- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `go build -o /tmp/testloop-mcp-v0.5.13-candidate .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.13-candidate ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.13-candidate --version` 输出 `testloop-mcp 0.5.12`，正式版本准备前未提前切版本号。
- [x] `/tmp/testloop-mcp-v0.5.13-candidate --help` 输出 `Usage of testloop-mcp`，exit code 为 `2`。
- [x] `/tmp/testloop-testgen-v0.5.13-candidate --help` 输出 `Usage: testgen`，exit code 为 `2`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.13-candidate-dist scripts/package-release-asset.sh v0.5.13 darwin_arm64 darwin arm64`
- [x] 在 dist 目录内校验 `testloop-mcp_v0.5.13_darwin_arm64.tar.gz.sha256` 通过。
- [x] `tar -tzf /tmp/testloop-v0.5.13-candidate-dist/testloop-mcp_v0.5.13_darwin_arm64.tar.gz`，内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 正式发布前待办

- [ ] 更新 `main.go` MCP implementation version 到 `0.5.13`。
- [ ] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.13 - 2026-07-20`。
- [ ] 同步 README、installation、quickstart 和必要版本引用到 `0.5.13` / `v0.5.13`。
- [ ] 测试中的版本期望同步到 `0.5.13`。
- [ ] 重新运行完整本地验证，确认版本准备改动可发布。
- [ ] 提交版本准备改动后确认远端 CI passed。
- [ ] 打 tag `v0.5.13` 并推送。
- [ ] Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.13` 验证 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.13 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.13` 更新仓库内 Formula。
- [ ] 更新 Homebrew tap 到 `0.5.13` 并推送。
- [ ] 手动触发 Post-Release Verify，确认资产清单和五平台安装脚本 dry run 通过。

## 当前结论

v0.5.13 已具备候选发布边界，候选 release readiness 已通过：这一轮聚焦 Agent/客户端可消费契约，不扩语言、不改大方向。下一步如果要继续发布，应进入正式版本准备：切 `0.5.13` 版本号、收敛 changelog、同步版本引用并重新跑发布前门禁。
