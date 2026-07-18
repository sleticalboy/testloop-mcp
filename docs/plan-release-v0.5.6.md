# v0.5.6 发布检查清单

## 当前目标

这是 v0.5.6 的候选发布准备和 release readiness 记录。当前阶段只整理候选内容和本地门禁，不切版本号、不打 tag、不更新 Homebrew tap。

v0.5.6 发布重点见 [v0.5.6 发布说明草案](./plan-release-notes-v0.5.6.md)：本轮主要是 Onboarding CI 复制模板、bootstrap 脚本、workflow YAML 可解析性、GitHub step summary 和失败路径排查。

## 当前差异核对

- [x] `docs/onboarding-ci-template.md` 已加入。
- [x] `test/onboarding_ci_template_doc_test.sh` 已加入并纳入 CI。
- [x] `test/onboarding_ci_template_yaml_test.sh` 已加入并纳入 CI。
- [x] `scripts/run-onboarding-ci.sh` 已加入。
- [x] `test/run_onboarding_ci_test.sh` 已加入并纳入 CI。
- [x] `docs/onboarding-ci-failure-triage.md` 已加入。
- [x] `test/onboarding_ci_failure_triage_doc_test.sh` 已加入并纳入 CI。
- [x] README、showcase、verification CI 文档、CHANGELOG 和 roadmap 已同步本轮内容。
- [x] 当前不更新 `main.go` MCP implementation version；正式版本准备阶段再切到 `0.5.6`。
- [x] Homebrew Formula 暂不改 sha256；正式 Release Artifacts 生成后再通过真实 asset digest 更新 tap。

## 候选内容

- [x] Onboarding CI 复制模板：`docs/onboarding-ci-template.md`。
- [x] Workflow YAML 可解析性测试：`test/onboarding_ci_template_yaml_test.sh`。
- [x] 外部用户项目 CI bootstrap：`scripts/run-onboarding-ci.sh`。
- [x] Bootstrap 回归测试：`test/run_onboarding_ci_test.sh`。
- [x] GitHub step summary 输出。
- [x] Onboarding CI 失败排查文档：`docs/onboarding-ci-failure-triage.md`。
- [x] 失败排查文档测试：`test/onboarding_ci_failure_triage_doc_test.sh`。
- [x] 当前仓库真实 dry-run 记录。

## 已验证

- [x] `bash -n scripts/run-onboarding-ci.sh`
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
- [x] `sh test/run_onboarding_ci_test.sh`
- [x] `sh test/onboarding_ci_failure_triage_doc_test.sh`
- [x] `sh test/showcase_agent_onboarding_report_test.sh`
- [x] `sh test/showcase_scripts_test.sh`
- [x] `sh test/showcase_summary_test.sh`
- [x] `sh test/verification_report_test.sh`
- [x] `sh test/real_integration_cases_doc_test.sh`
- [x] `sh test/onboarding_ci_template_doc_test.sh`
- [x] `sh test/onboarding_ci_template_yaml_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/docs_json_examples_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/fixtures_index_test.sh`
- [x] `sh test/fixture_decision_mapping_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/verification_ci_doc_test.sh`
- [x] `go build -o /tmp/testloop-mcp-v0.5.6-candidate .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.6-candidate ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.6-candidate --help` 输出 usage。
- [x] `/tmp/testloop-testgen-v0.5.6-candidate --help` 输出 usage。
- [x] 使用 `scripts/run-onboarding-ci.sh 'go test ./...'` 跑通真实 dry-run，输出 `agent_next_step=ready`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.6-candidate-dist scripts/package-release-asset.sh v0.5.6 darwin_arm64 darwin arm64`
- [x] `/tmp/testloop-v0.5.6-candidate-dist/testloop-mcp_v0.5.6_darwin_arm64.tar.gz.sha256` 校验通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 正式发布前待办

- [ ] 更新 `main.go` MCP implementation version 到 `0.5.6`。
- [ ] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.6 - 2026-07-18`。
- [ ] 同步 README 中当前 Release、手动下载示例、Windows 下载示例到 `v0.5.6`。
- [ ] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.5.6`。
- [ ] quickstart、onboarding、verification report、verification CI 示例中的版本门禁同步到 `0.5.6`。
- [ ] 测试中的版本期望同步到 `0.5.6`。
- [ ] 重新运行完整验证：`go test ./...`、所有默认 shell 校验、脚本语法检查、主服务/CLI 构建、打包 dry-run。
- [ ] 提交版本准备改动后确认远端 CI 通过。
- [ ] 打 tag `v0.5.6` 并推送。
- [ ] 等待 Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.6` 验证 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.6 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.6` 更新仓库内 Formula。
- [ ] 更新 Homebrew tap 到 `0.5.6`。
- [ ] 手动触发 Post-Release Verify，确认五平台安装脚本 dry run 全部通过。
- [ ] 本机 Homebrew tap 快进后运行 `HOMEBREW_NO_AUTO_UPDATE=1 brew fetch --force sleticalboy/tap/testloop-mcp`，确认获取 `0.5.6`。

## 当前结论

v0.5.6 候选内容已收敛，本地 release readiness 门禁通过。当前阶段不发布；下一步应提交候选文档并等待远端 CI，通过后进入正式版本准备。
