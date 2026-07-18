# v0.5.3 发布检查清单

## 当前目标

这是 v0.5.3 的候选发布准备和 release readiness 记录。当前阶段只做候选准备和本地门禁，不切版本号、不打 tag、不更新 Homebrew tap。

v0.5.3 发布重点见 [v0.5.3 发布说明草案](./plan-release-notes-v0.5.3.md)：本轮主要是验收报告脚本、summary JSON、真实用户项目 smoke、Agent/CI 决策示例和 GitHub Actions 集成文档。

## 当前差异核对

- [x] `docs/plan-release-notes-v0.5.3.md` 已创建。
- [x] `docs/plan-release-v0.5.3.md` 已创建。
- [x] `main.go` MCP implementation version 仍为 `0.5.2`，正式版本准备时再更新到 `0.5.3`。
- [x] `CHANGELOG.md` 的 `Unreleased` 暂不收敛，正式版本准备时再移动到 `v0.5.3 - 2026-07-18`。
- [x] README、安装文档、quickstart 和 onboarding 文档中的当前 release 仍保留 `v0.5.2`，正式版本准备时再同步。
- [x] 仓库内 Homebrew Formula 暂不更新；正式 Release Artifacts 生成后再使用真实 asset digest 更新。

## 候选内容

- [x] 用户项目 Markdown 验收报告：`scripts/generate-verification-report.sh`。
- [x] summary JSON 输出：`TESTLOOP_REPORT_SUMMARY_JSON`。
- [x] 真实 Go server / Vue web 用户项目 smoke 样例。
- [x] summary JSON 决策示例：`examples/verification-summary-decision-demo`。
- [x] GitHub Actions 集成文档：`docs/verification-ci.md`。
- [x] README、showcase、verification report 和 release doc index 入口同步。

## 已验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/verify-release-assets.sh`
- [x] `bash -n scripts/generate-verification-report.sh`
- [x] `bash -n scripts/showcase-go-public-project.sh scripts/showcase-js-public-project.sh scripts/showcase-onboarding.sh`
- [x] `python3 -m py_compile scripts/summarize-showcase-output.py`
- [x] `go test ./...`
- [x] `sh test/install_script_test.sh`
- [x] `sh test/release_assets_test.sh`
- [x] `sh test/llm_provider_example_test.sh`
- [x] `sh test/verify_client_setup_test.sh`
- [x] `sh test/mcp_process_smoke_test.sh`
- [x] `sh test/mcp_client_demo_test.sh`
- [x] `sh test/agent_decision_demo_test.sh`
- [x] `sh test/verification_summary_decision_demo_test.sh`
- [x] `sh test/showcase_scripts_test.sh`
- [x] `sh test/showcase_summary_test.sh`
- [x] `sh test/verification_report_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/docs_json_examples_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/fixtures_index_test.sh`
- [x] `sh test/fixture_decision_mapping_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/verification_ci_doc_test.sh`
- [x] `go build -o /tmp/testloop-mcp-v0.5.3-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.3-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.3-prep --help` 输出 usage。
- [x] `/tmp/testloop-testgen-v0.5.3-prep --help` 输出 usage。
- [x] 生成 Markdown + JSON 验收报告并用决策 demo 输出 `agent_next_step=ready`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.3-prep-dist scripts/package-release-asset.sh v0.5.3 darwin_arm64 darwin arm64`
- [x] `/tmp/testloop-v0.5.3-prep-dist/testloop-mcp_v0.5.3_darwin_arm64.tar.gz.sha256` 校验通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 正式发布前待办

- [ ] 更新 `main.go` MCP implementation version 到 `0.5.3`。
- [ ] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.3 - 2026-07-18`。
- [ ] 同步 README 中当前 Release、手动下载示例、Windows 下载示例到 `v0.5.3`。
- [ ] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.5.3`。
- [ ] quickstart、onboarding showcase 和验收报告示例中的版本门禁同步到 `0.5.3`。
- [ ] 重新运行完整验证：`go test ./...`、所有默认 shell 校验、脚本语法检查、主服务/CLI 构建、打包 dry-run。
- [ ] 提交版本准备改动后确认远端 CI 通过。
- [ ] 打 tag `v0.5.3` 并推送。
- [ ] 等待 Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.3` 验证 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.3 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.3` 更新仓库内 Formula。
- [ ] 更新 Homebrew tap 到 `0.5.3` 并推送。
- [ ] 手动触发 Post-Release Verify，确认五平台安装脚本 dry run 全部通过。
- [ ] 本机 Homebrew tap 快进后运行 `HOMEBREW_NO_AUTO_UPDATE=1 brew fetch --force sleticalboy/tap/testloop-mcp`。

## 当前结论

v0.5.3 候选内容已经明确，本地 release readiness 门禁已通过。下一步可以进入正式版本准备：更新版本号、收敛 CHANGELOG、同步文档版本引用，提交后等待远端 CI，再决定是否打 `v0.5.3` tag。
