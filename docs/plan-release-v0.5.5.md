# v0.5.5 发布检查清单

## 当前目标

这是 v0.5.5 的候选发布准备、正式版本准备和 release readiness 记录。当前阶段已切版本号并同步文档引用，但不打 tag、不更新 Homebrew tap。

v0.5.5 发布重点见 [v0.5.5 发布说明草案](./plan-release-notes-v0.5.5.md)：本轮主要是真实接入案例模板、安装漂移诊断、Homebrew 升级/重装提示，以及真实安装态 onboarding 验收。

## 当前差异核对

- [x] `docs/plan-release-notes-v0.5.5.md` 已创建。
- [x] `docs/plan-release-v0.5.5.md` 已创建。
- [x] `docs/real-integration-cases.md` 已加入。
- [x] `test/real_integration_cases_doc_test.sh` 已加入并纳入 CI。
- [x] `scripts/verify-client-setup.sh` 已增强旧二进制 `--version` 失败和版本不匹配提示。
- [x] `test/verify_client_setup_test.sh` 已覆盖安装漂移诊断提示。
- [x] `docs/installation.md` 和 `docs/quickstart.md` 已补 Homebrew 升级/重装路径。
- [x] README、showcase、CHANGELOG 和 roadmap 已同步本轮内容。
- [x] `main.go` MCP implementation version 已更新到 `0.5.5`。
- [x] Homebrew Formula 暂不改 sha256；正式 Release Artifacts 生成后再通过真实 asset digest 更新 tap。

## 候选内容

- [x] 真实接入案例模板：`docs/real-integration-cases.md`。
- [x] 真实接入文档回归测试：`test/real_integration_cases_doc_test.sh`。
- [x] 安装版本漂移诊断：`scripts/verify-client-setup.sh`。
- [x] 安装诊断回归测试：`test/verify_client_setup_test.sh`。
- [x] Homebrew 升级/重装文档：`docs/installation.md`、`docs/quickstart.md`。
- [x] laoxia Go server onboarding report 实跑记录。
- [x] laoxia Vue web onboarding report 实跑记录。
- [x] Homebrew 安装态 onboarding report 实跑记录。

## 已验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/verify-release-assets.sh`
- [x] `bash -n scripts/generate-verification-report.sh`
- [x] `bash -n scripts/showcase-agent-onboarding-report.sh`
- [x] `bash -n scripts/verify-client-setup.sh`
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
- [x] `sh test/real_integration_cases_doc_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/docs_json_examples_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/fixtures_index_test.sh`
- [x] `sh test/fixture_decision_mapping_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/verification_ci_doc_test.sh`
- [x] `go build -o /tmp/testloop-mcp-v0.5.5-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.5-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.5-prep --help` 输出 usage。
- [x] `/tmp/testloop-testgen-v0.5.5-prep --help` 输出 usage。
- [x] `/tmp/testloop-mcp-v0.5.5-prep --version` 输出 `testloop-mcp 0.5.5`。
- [x] 使用 `/tmp/testloop-mcp-v0.5.5-prep` 跑通 `TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.5 scripts/verify-client-setup.sh /tmp/testloop-mcp-v0.5.5-prep`。
- [x] `scripts/showcase-agent-onboarding-report.sh /tmp/testloop-mcp-v0.5.5-prep` 输出 `agent_next_step=ready`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.5-prep-dist scripts/package-release-asset.sh v0.5.5 darwin_arm64 darwin arm64`
- [x] `/tmp/testloop-v0.5.5-prep-dist/testloop-mcp_v0.5.5_darwin_arm64.tar.gz.sha256` 校验通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.5.5`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.5 - 2026-07-18`。
- [x] 同步 README 中当前 Release、手动下载示例、Windows 下载示例到 `v0.5.5`。
- [x] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.5.5`。
- [x] quickstart、onboarding、real integration cases、verification report、verification CI 示例中的版本门禁同步到 `0.5.5`。
- [x] 测试中的版本期望同步到 `0.5.5`。
- [x] 重新运行完整验证：`go test ./...`、所有默认 shell 校验、脚本语法检查、主服务/CLI 构建、打包 dry-run。
- [ ] 提交版本准备改动后确认远端 CI 通过。
- [ ] 打 tag `v0.5.5` 并推送。
- [ ] 等待 Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.5` 验证 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.5 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.5` 更新仓库内 Formula。
- [ ] 更新 Homebrew tap 到 `0.5.5`。
- [ ] 手动触发 Post-Release Verify，确认五平台安装脚本 dry run 全部通过。
- [ ] 本机 Homebrew tap 快进后运行 `HOMEBREW_NO_AUTO_UPDATE=1 brew fetch --force sleticalboy/tap/testloop-mcp`，确认获取 `0.5.5`。

## 当前结论

v0.5.5 版本准备已经完成：版本号、CHANGELOG 和文档版本引用已同步，本地完整门禁通过。当前阶段不发布；下一步应提交后等待远端 CI，通过后再打 `v0.5.5` tag，并进入 Release Artifacts / 资产校验 / GitHub Release 正文 / Homebrew tap 验证。
