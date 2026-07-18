# v0.5.4 发布检查清单

## 当前目标

这是 v0.5.4 的候选发布准备和 release readiness 记录。当前阶段只整理候选内容和本地门禁，不打 tag、不更新 Homebrew tap。

v0.5.4 发布重点见 [v0.5.4 发布说明草案](./plan-release-notes-v0.5.4.md)：本轮主要是 Agent onboarding report wrapper、summary 失败分流 fixture 和 GitHub Actions 示例收敛。

## 当前差异核对

- [x] `docs/plan-release-notes-v0.5.4.md` 已创建。
- [x] `docs/plan-release-v0.5.4.md` 已创建。
- [x] `scripts/showcase-agent-onboarding-report.sh` 已加入。
- [x] `docs/verification-summary-failures.md` 和 `docs/fixtures/verification-summary/*.json` 已加入。
- [x] `docs/verification-ci.md` 已优先推荐 onboarding report wrapper。
- [x] README、quickstart、showcase、verification report、roadmap 和 release doc index 入口已同步。
- [x] `main.go` MCP implementation version 已更新到 `0.5.4`。
- [x] 仓库内 Homebrew Formula 已用正式 Release Artifacts 的真实 asset digest 更新到 `0.5.4`。

## 候选内容

- [x] Agent onboarding report wrapper：`scripts/showcase-agent-onboarding-report.sh`。
- [x] wrapper 回归测试：`test/showcase_agent_onboarding_report_test.sh`。
- [x] 验收 summary 失败分流文档：`docs/verification-summary-failures.md`。
- [x] 验收 summary 失败 fixture：`docs/fixtures/verification-summary/*.json`。
- [x] 失败 fixture 决策回归测试：`test/verification_summary_failure_fixtures_test.sh`。
- [x] CI 推荐 workflow 收敛：`docs/verification-ci.md`。
- [x] v0.5.4 onboarding demo 规划：`docs/plan-agent-onboarding-v0.5.4.md`。

## 已验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/verify-release-assets.sh`
- [x] `bash -n scripts/generate-verification-report.sh`
- [x] `bash -n scripts/showcase-agent-onboarding-report.sh`
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
- [x] `sh test/verification_summary_failure_fixtures_test.sh`
- [x] `sh test/showcase_agent_onboarding_report_test.sh`
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
- [x] `go build -o /tmp/testloop-mcp-v0.5.4-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.4-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.4-prep --help` 输出 usage。
- [x] `/tmp/testloop-testgen-v0.5.4-prep --help` 输出 usage。
- [x] 使用真实构建二进制运行 onboarding report wrapper，并用 summary decision 输出 `agent_next_step=ready`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.4-prep-dist scripts/package-release-asset.sh v0.5.4 darwin_arm64 darwin arm64`
- [x] `/tmp/testloop-v0.5.4-prep-dist/testloop-mcp_v0.5.4_darwin_arm64.tar.gz.sha256` 校验通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.5.4`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.4 - 2026-07-18`。
- [x] 同步 README 中当前 Release、手动下载示例、Windows 下载示例到 `v0.5.4`。
- [x] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.5.4`。
- [x] quickstart、onboarding、verification report、verification CI 示例中的版本门禁同步到 `0.5.4`。
- [x] 测试中的版本期望同步到 `0.5.4`。
- [x] 重新运行完整验证：`go test ./...`、所有默认 shell 校验、脚本语法检查、主服务/CLI 构建、打包 dry-run。
- [x] 提交版本准备改动后确认远端 CI run `29638973367` 通过。
- [x] 打 tag `v0.5.4` 并推送。
- [x] 等待 Release Artifacts workflow run `29639038941` 生成五平台资产和 `.sha256`。
- [x] 使用 `scripts/verify-release-assets.sh v0.5.4` 验证 Release 资产完整。
- [x] 更新 GitHub Release 正文为正式 v0.5.4 发布说明。
- [x] 使用 `scripts/generate-homebrew-formula.sh v0.5.4` 更新仓库内 Formula。
- [x] 更新 Homebrew tap 到 `0.5.4` 并推送到 `00b56f2`。
- [x] 手动触发 Post-Release Verify run `29639243485`，确认五平台安装脚本 dry run 全部通过。
- [x] 本机 Homebrew tap 快进后运行 `HOMEBREW_NO_AUTO_UPDATE=1 brew fetch --force sleticalboy/tap/testloop-mcp`，确认获取 `0.5.4`。

## 当前结论

v0.5.4 已完成正式发布、Release Artifacts、资产校验、GitHub Release 正文更新、Homebrew tap 更新和 Post-Release Verify。发布收尾只剩提交并推送本仓库的 Formula 与发布记录更新。
