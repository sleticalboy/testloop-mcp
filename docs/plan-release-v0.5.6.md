# v0.5.6 发布检查清单

## 当前目标

这是 v0.5.6 的发布准备、正式发布和发布后核验记录。

v0.5.6 发布重点见 [v0.5.6 发布说明](./plan-release-notes-v0.5.6.md)：本轮主要是 Onboarding CI 复制模板、bootstrap 脚本、workflow YAML 可解析性、GitHub step summary 和失败路径排查。

## 当前差异核对

- [x] `docs/onboarding-ci-template.md` 已加入。
- [x] `test/onboarding_ci_template_doc_test.sh` 已加入并纳入 CI。
- [x] `test/onboarding_ci_template_yaml_test.sh` 已加入并纳入 CI。
- [x] `scripts/run-onboarding-ci.sh` 已加入。
- [x] `test/run_onboarding_ci_test.sh` 已加入并纳入 CI。
- [x] `docs/onboarding-ci-failure-triage.md` 已加入。
- [x] `test/onboarding_ci_failure_triage_doc_test.sh` 已加入并纳入 CI。
- [x] README、showcase、verification CI 文档、CHANGELOG 和 roadmap 已同步本轮内容。
- [x] `main.go` MCP implementation version 已更新到 `0.5.6`。
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

- [x] 更新 `main.go` MCP implementation version 到 `0.5.6`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.6 - 2026-07-18`。
- [x] 同步 README 中当前 Release、手动下载示例、Windows 下载示例到 `v0.5.6`。
- [x] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.5.6`。
- [x] quickstart、onboarding、verification report、verification CI 示例中的版本门禁同步到 `0.5.6`。
- [x] 测试中的版本期望同步到 `0.5.6`。
- [x] 重新运行完整验证：`go test ./...`、所有默认 shell 校验、脚本语法检查、主服务/CLI 构建、打包 dry-run。
- [x] 提交版本准备改动后确认远端 CI run `29648677800` passed。
- [x] 打 tag `v0.5.6` 并推送。
- [x] Release Artifacts workflow run `29648755666` passed，五平台资产和 `.sha256` 已生成。
- [x] 使用 `scripts/verify-release-assets.sh v0.5.6` 验证 10 个 Release 资产完整。
- [x] 更新 GitHub Release 正文为正式 v0.5.6 发布说明。
- [x] 使用 `scripts/generate-homebrew-formula.sh v0.5.6` 更新仓库内 Formula。
- [x] 更新 Homebrew tap 到 `0.5.6`，提交 `000a417` 并推送。
- [x] 手动触发 Post-Release Verify run `29648990368`，资产清单和五平台安装脚本 dry run 全部通过。
- [x] 本机 Homebrew tap 已快进到 `000a417420e03c8dc278de28bce6d318e4880a1b`，`brew info --json=v2 sleticalboy/tap/testloop-mcp` 返回 `version=0.5.6`。
- [ ] 本机 `HOMEBREW_NO_AUTO_UPDATE=1 brew fetch --force sleticalboy/tap/testloop-mcp` 下载阶段无输出卡住；直接 `curl` Release 资产也卡在下载阶段。远端 Post-Release Verify 已覆盖安装脚本 dry run，后续网络稳定后补跑本机 fetch 即可。

## 正式发布核验证据

- [x] Release Artifacts：`29648755666` passed。
- [x] Release 资产完整性：`scripts/verify-release-assets.sh v0.5.6` 输出 `Verified 10 release assets for sleticalboy/testloop-mcp@v0.5.6`。
- [x] GitHub Release：`https://github.com/sleticalboy/testloop-mcp/releases/tag/v0.5.6`。
- [x] 仓库内 Formula：`ruby -c Formula/testloop-mcp.rb` 通过，`brew style Formula/testloop-mcp.rb` 通过。
- [x] Homebrew tap：`sleticalboy/homebrew-tap` 提交 `000a417 testloop-mcp 0.5.6` 已推送。
- [x] 本机 tap 缓存：`brew info --json=v2 sleticalboy/tap/testloop-mcp` 返回 `version=0.5.6`、`tap_git_head=000a417420e03c8dc278de28bce6d318e4880a1b`。
- [x] Post-Release Verify：`29648990368` passed，覆盖 Release manifest 和 linux amd64、linux arm64、darwin arm64、windows amd64、windows arm64 安装脚本 dry run。
- [ ] 本机 fetch：`brew fetch` 和直接 `curl` 下载 Release asset 当前卡住，判断为本机到 GitHub Release 资产下载链路不稳定，不阻塞已通过的远端发布核验。

## 当前结论

v0.5.6 已完成正式发布和远端发布后核验：Release Artifacts、Release 资产清单、GitHub Release 正文、仓库 Formula、Homebrew tap 和 Post-Release Verify 均已完成。唯一残余项是本机 `brew fetch` 因 GitHub Release 资产下载卡住未完成；tap 版本和远端安装 dry run 已验证，网络恢复后可补跑本机 fetch。
