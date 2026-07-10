# v0.4.13 发布检查清单

## 当前目标

这是 v0.4.13 的发布记录。当前阶段已完成版本准备、本地发布前验证、tag、GitHub Release、Release Artifacts、Post-Release Verify 和 Homebrew tap 发布核验。

v0.4.13 发布重点见 [v0.4.13 发布说明草案](./plan-release-notes-v0.4.13.md)：本轮主要是 LLM provider 接入质量、输出校验、结构化 provider error、Agent static fallback 闭环，以及安装脚本 fallback 日志修正。

## 当前差异核对

- [x] `docs/plan-release-notes-v0.4.13.md` 已创建，未回写已发布的 `docs/plan-release-notes-v0.4.12.md`。
- [x] `CHANGELOG.md` 已新增 `v0.4.13 - 2026-07-10` 小节，收敛本轮候选能力点。
- [x] `main.go` MCP implementation version 已更新为 `0.4.13`。
- [x] README 当前 Release、手动下载示例和 Windows 下载示例已同步到 `v0.4.13`。
- [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.4.13`。
- [x] `scripts/install.sh` 默认使用 `latest`，不需要为 v0.4.13 预先改脚本。
- [x] `scripts/package-release-asset.sh`、`scripts/generate-homebrew-formula.sh`、`scripts/update-homebrew-tap.sh` 和 `scripts/verify-release-assets.sh` 都按 tag 参数工作，不需要为 v0.4.13 改脚本。
- [x] GitHub Actions release workflow 仍按 tag 触发生成五平台资产，不需要为 v0.4.13 改 workflow。

## 已验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-client-config.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/verify-release-assets.sh`
- [x] `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/*.yml`
- [x] `sh test/llm_provider_example_test.sh`
- [x] `sh test/install_script_test.sh`
- [x] `sh test/release_assets_test.sh`
- [x] `go test ./...`
- [x] `go build -o /tmp/testloop-mcp-v0.4.13-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.4.13-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.4.13-prep --help` 输出 usage。
- [x] `/tmp/testloop-testgen-v0.4.13-prep --help` 输出 usage。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.4.13-prep scripts/package-release-asset.sh v0.4.13 darwin_arm64 darwin arm64`
- [x] `/tmp/testloop-v0.4.13-prep/testloop-mcp_v0.4.13_darwin_arm64.tar.gz.sha256` 校验通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.4.13`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.4.13 - 2026-07-10`。
- [x] 同步 README 中当前 Release、手动下载示例、Windows 下载示例到 `v0.4.13`。
- [x] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.4.13`。
- [x] 重新运行完整验证：`go test ./...`、脚本语法检查、actionlint、主服务/CLI 构建、打包 dry-run。
- [x] 提交版本准备改动后确认远端 CI 通过：run `29087539959`。
- [x] 打 tag `v0.4.13` 并推送，tag 指向 `cebb4832ef9a7b8a84dbbb71e19f2989c1c74599`。
- [x] 等待 Release Artifacts workflow `29089692602` 生成五平台资产和 `.sha256`。
- [x] 使用 `scripts/verify-release-assets.sh v0.4.13` 验证 Release 资产，确认 10 个必需资产完整。
- [x] 更新 GitHub Release 正文为正式 v0.4.13 发布说明。
- [x] 手动触发 Post-Release Verify run `29090486292`，五平台安装脚本 dry run 全部通过。
- [x] 更新 Homebrew tap 到 `0.4.13`，tap commit `25b8018454c1b73cf259c08b13db06f59dcfc234`；并通过 `brew fetch`、`brew audit --formula --strict`、`brew upgrade --formula`、`brew test`。

## 当前结论

v0.4.13 已完成正式发布和发布后核验。Release 页面包含 10 个必需资产，Post-Release Verify 五平台安装 dry run 通过，Homebrew tap 已升级到 `0.4.13` 并通过本机 `brew test`。
