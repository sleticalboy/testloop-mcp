# v0.5.1 发布检查清单

## 当前目标

这是 v0.5.1 的版本准备和 release readiness 记录。当前已完成发布说明草案、发布前本地门禁、版本号切换和安装文档同步；还没有打 tag、创建 GitHub Release、生成 Release Artifacts 或更新 Homebrew tap。

v0.5.1 发布重点见 [v0.5.1 发布说明草案](./plan-release-notes-v0.5.1.md)：本轮主要是 MCP 客户端接入、Agent 结构化契约、真实 handler fixture、客户端动作映射和 release 文档入口回归保护。

## 当前差异核对

- [x] `docs/plan-release-notes-v0.5.1.md` 已创建。
- [x] `CHANGELOG.md` 的 `Unreleased` 内容已收敛到 `v0.5.1 - 2026-07-17`。
- [x] `main.go` MCP implementation version 已更新为 `0.5.1`。
- [x] README 和 `docs/installation.md` 已同步到 `v0.5.1`。
- [x] `scripts/install.sh` 默认使用 `latest`，不需要为 v0.5.1 预先改脚本。
- [x] `scripts/package-release-asset.sh`、`scripts/generate-homebrew-formula.sh`、`scripts/update-homebrew-tap.sh` 和 `scripts/verify-release-assets.sh` 都按 tag 参数工作，不需要为 v0.5.1 改脚本。
- [x] GitHub Actions release workflow 仍按 tag 触发生成五平台资产，不需要为 v0.5.1 改 workflow。

## 已验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/verify-release-assets.sh`
- [x] `go test ./...`
- [x] `sh test/install_script_test.sh`
- [x] `sh test/release_assets_test.sh`
- [x] `sh test/llm_provider_example_test.sh`
- [x] `sh test/verify_client_setup_test.sh`
- [x] `sh test/mcp_client_demo_test.sh`
- [x] `sh test/agent_decision_demo_test.sh`
- [x] `sh test/showcase_scripts_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/docs_json_examples_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/fixtures_index_test.sh`
- [x] `sh test/fixture_decision_mapping_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `go build -o /tmp/testloop-mcp-v0.5.1-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.1-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.1-prep --help` 输出 usage。
- [x] `/tmp/testloop-testgen-v0.5.1-prep --help` 输出 usage。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.1-prep scripts/package-release-asset.sh v0.5.1 darwin_arm64 darwin arm64`
- [x] `/tmp/testloop-v0.5.1-prep/testloop-mcp_v0.5.1_darwin_arm64.tar.gz.sha256` 校验通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.5.1`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.1 - 2026-07-17`。
- [x] 同步 README 中当前 Release、手动下载示例、Windows 下载示例到 `v0.5.1`。
- [x] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.5.1`。
- [ ] 更新仓库内 `Formula/testloop-mcp.rb` 到 `0.5.1`，或在 Homebrew tap 发布步骤中生成并验证。
- [x] 重新运行完整验证：`go test ./...`、所有默认 shell 校验、脚本语法检查、主服务/CLI 构建、打包 dry-run。
- [ ] 提交版本准备改动后确认远端 CI 通过。
- [ ] 打 tag `v0.5.1` 并推送。
- [ ] 等待 Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.1` 验证 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.1 发布说明。
- [ ] 手动触发 Post-Release Verify，确认五平台安装脚本 dry run 全部通过。
- [ ] 更新 Homebrew tap 到 `0.5.1`，并通过 `brew fetch`、`brew audit --formula --strict`、`brew upgrade --formula`、`brew test`。

## 当前结论

v0.5.1 已完成候选发布资料、本地 release readiness 门禁、版本号切换和安装文档同步。当前不应直接打 tag；还需要提交版本准备改动后等待远端 CI，通过后再进入 tag、Release Artifacts、资产校验和 Homebrew tap 验证。
