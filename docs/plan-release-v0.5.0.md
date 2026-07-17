# v0.5.0 发布检查清单

## 当前目标

这是 v0.5.0 的版本准备和正式发布记录。当前阶段只做候选版本准备、本地验证和远端 CI 验证；tag、GitHub Release、Release Artifacts、GitHub Release 正文、Homebrew tap 和 Post-Release Verify 仍需要后续单独执行。

v0.5.0 发布重点见 [v0.5.0 发布说明草案](./plan-release-notes-v0.5.0.md)：本轮主要是固定 smoke 矩阵、Java 目标行命中校验、JS/Python 真实项目手审分类、以及面向 AI Agent 的测试反馈闭环定位。

## 当前差异核对

- [x] `docs/plan-release-notes-v0.5.0.md` 已创建。
- [x] `CHANGELOG.md` 已新增 `v0.5.0 - 2026-07-17` 小节，收敛本轮候选能力点。
- [x] `main.go` MCP implementation version 已更新为 `0.5.0`。
- [x] README 当前 Release、手动下载示例和 Windows 下载示例已同步到 `v0.5.0`。
- [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.5.0`。
- [x] `scripts/install.sh` 默认使用 `latest`，不需要为 v0.5.0 预先改脚本。
- [x] `scripts/package-release-asset.sh`、`scripts/generate-homebrew-formula.sh`、`scripts/update-homebrew-tap.sh` 和 `scripts/verify-release-assets.sh` 都按 tag 参数工作，不需要为 v0.5.0 改脚本。
- [x] GitHub Actions release workflow 仍按 tag 触发生成五平台资产，不需要为 v0.5.0 改 workflow。

## 已验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/verify-release-assets.sh`
- [x] `go test ./...`
- [x] `TESTLOOP_VALIDATE_JS_STAGE_TIMEOUT_SECONDS=180 TESTLOOP_VALIDATE_PY_STAGE_TIMEOUT_SECONDS=180 scripts/validate-regression-smoke.sh`
- [x] `go build -o /tmp/testloop-mcp-v0.5.0-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.0-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.0-prep --help` 输出 usage。
- [x] `/tmp/testloop-testgen-v0.5.0-prep --help` 输出 usage。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.0-prep scripts/package-release-asset.sh v0.5.0 darwin_arm64 darwin arm64`
- [x] `/tmp/testloop-v0.5.0-prep/testloop-mcp_v0.5.0_darwin_arm64.tar.gz.sha256` 校验通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`
- [ ] 提交版本准备改动后确认远端 CI 通过。

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.5.0`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.0 - 2026-07-17`。
- [x] 同步 README 中当前 Release、手动下载示例、Windows 下载示例到 `v0.5.0`。
- [x] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.5.0`。
- [x] 重新运行完整验证：`go test ./...`、固定 smoke、脚本语法检查、主服务/CLI 构建、打包 dry-run。
- [ ] 提交版本准备改动后确认远端 CI 通过。
- [ ] 打 tag `v0.5.0` 并推送。
- [ ] 等待 Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.0` 验证 Release 资产，确认 10 个必需资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.0 发布说明。
- [ ] 手动触发 Post-Release Verify，确认五平台安装脚本 dry run 全部通过。
- [ ] 更新 Homebrew tap 到 `0.5.0`，并通过 `brew fetch`、`brew audit --formula --strict`、`brew upgrade --formula`、`brew test`。

## 当前结论

v0.5.0 已进入候选版本准备阶段；当前尚未打 tag，也尚未创建或更新 GitHub Release / Homebrew tap。
