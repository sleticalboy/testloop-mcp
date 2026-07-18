# v0.5.2 发布检查清单

## 当前目标

这是 v0.5.2 的候选发布准备、release readiness 和正式发布核验记录。当前阶段只做候选准备和本地门禁，不切版本号、不打 tag、不更新 Homebrew tap。

v0.5.2 发布重点见 [v0.5.2 发布说明草案](./plan-release-notes-v0.5.2.md)：本轮主要是安装后版本门禁、真实 MCP 进程级 smoke、首次接入 showcase、公开项目 action 断言、本地 checkout 复用、git 超时和 showcase summary 归档策略。

## 当前差异核对

- [x] `docs/plan-release-notes-v0.5.2.md` 已创建。
- [x] `docs/plan-release-v0.5.2.md` 已创建。
- [x] `main.go` MCP implementation version 已更新到 `0.5.2`。
- [x] `CHANGELOG.md` 的 `Unreleased` 内容已收敛到 `v0.5.2 - 2026-07-18`。
- [x] README、安装文档、quickstart 和 onboarding 文档已同步到 `v0.5.2`。
- [x] 本地 release readiness 门禁已完成，当前仍不打 `v0.5.2` tag。
- [x] 仓库内 Homebrew Formula 已更新到 `0.5.2`，使用 GitHub Release 真实 asset digest。

## 候选内容

- [x] `--version` 和安装后版本门禁。
- [x] `examples/mcp-process-smoke` 和 `scripts/verify-mcp-process-smoke.sh`。
- [x] 接入验收文档分层：基础安装验收与深度协议验收。
- [x] `scripts/showcase-onboarding.sh`。
- [x] 公开 Go / JS showcase action 断言。
- [x] 公开 Go / JS showcase 本地 checkout 复用。
- [x] 公开 Go / JS showcase git clone/fetch 超时。
- [x] JS/TS 公开 showcase 远端复验结果。
- [x] showcase summary 脚本和归档策略。

## 已验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/verify-release-assets.sh`
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
- [x] `sh test/showcase_scripts_test.sh`
- [x] `sh test/showcase_summary_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/docs_json_examples_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/fixtures_index_test.sh`
- [x] `sh test/fixture_decision_mapping_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `go build -o /tmp/testloop-mcp-v0.5.2-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.2-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.2-prep --help` 输出 usage。
- [x] `/tmp/testloop-testgen-v0.5.2-prep --help` 输出 usage。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.2-prep scripts/package-release-asset.sh v0.5.2 darwin_arm64 darwin arm64`
- [x] `/tmp/testloop-v0.5.2-prep/testloop-mcp_v0.5.2_darwin_arm64.tar.gz.sha256` 校验通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.5.2`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.2 - 2026-07-18`。
- [x] 同步 README 中当前 Release、手动下载示例、Windows 下载示例到 `v0.5.2`。
- [x] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.5.2`。
- [x] 更新仓库内 `Formula/testloop-mcp.rb` 到 `0.5.2`，使用 GitHub Release 真实 asset digest。
- [x] 重新运行完整验证：`go test ./...`、所有默认 shell 校验、脚本语法检查、主服务/CLI 构建、打包 dry-run。
- [x] 提交版本准备改动后确认远端 CI 通过：run `29629563807` passed。
- [x] 打 tag `v0.5.2` 并推送。
- [x] 等待 Release Artifacts workflow 生成五平台资产和 `.sha256`：run `29629630932` passed。
- [x] 使用 `scripts/verify-release-assets.sh v0.5.2` 验证 Release 资产完整：10 个必需资产已确认。
- [x] 更新 GitHub Release 正文为正式 v0.5.2 发布说明。
- [x] 手动触发 Post-Release Verify，确认五平台安装脚本 dry run 全部通过：run `29629793877` passed。
- [x] 更新 Homebrew tap 到 `0.5.2` 并推送：提交 `c1945e8 testloop-mcp 0.5.2`。
- [x] 本机 Homebrew tap 已快进到 `c1945e8`，`brew fetch --force sleticalboy/tap/testloop-mcp` 成功。

## 当前结论

v0.5.2 已正式发布。远端 CI、Release Artifacts、GitHub Release 正文、资产清单、仓库内 Formula、Homebrew tap 同步、tap style 校验、Post-Release Verify 五平台安装 dry run 和本机 `brew fetch` 下载验证均已完成。
