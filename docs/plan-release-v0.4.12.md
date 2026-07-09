# v0.4.12 发布前检查清单

## 当前目标

这是 v0.4.12 的发布准备记录。当前目标是完成版本准备改动和本地验证；正式 tag、GitHub Release、Release Artifacts 资产验证和 Homebrew tap 更新仍待执行。

v0.4.12 发布重点见 [v0.4.12 发布说明草案](./plan-release-notes-v0.4.12.md)：本轮主要是 JS/TS 简单泛型 payload、payload 回退原因上下文化和安装脚本 fallback 提示增强，不新增 MCP 工具。

## 当前差异核对

- [x] `docs/plan-release-notes-v0.4.12.md` 已创建，未回写已发布的 `docs/plan-release-notes-v0.4.11.md`。
- [x] `CHANGELOG.md` 已新增 `v0.4.12 - 2026-07-09` 小节，收敛本轮候选能力点。
- [x] `main.go` MCP implementation version 已更新为 `0.4.12`。
- [x] README 当前 Release、手动下载示例和 Windows 下载示例已同步到 `v0.4.12`。
- [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.4.12`。
- [x] `scripts/install.sh` 默认使用 `latest`，不需要为 v0.4.12 预先改脚本。
- [x] `scripts/package-release-asset.sh`、`scripts/generate-homebrew-formula.sh`、`scripts/update-homebrew-tap.sh` 和 `scripts/verify-release-assets.sh` 都按 tag 参数工作，不需要为 v0.4.12 改脚本。
- [x] GitHub Actions release workflow 仍按 tag 触发生成五平台资产，不需要为 v0.4.12 改 workflow。
- [x] 仓库内 `Formula/testloop-mcp.rb` 仍保留已发布的 `0.4.11`，待 v0.4.12 Release 资产生成并计算 checksum 后再更新。

## 已验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-client-config.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/verify-release-assets.sh`
- [x] `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/*.yml`
- [x] `bash test/install_script_test.sh`
- [x] `go test ./...`
- [x] `go build -o /tmp/testloop-mcp-v0.4.12-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.4.12-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.4.12-prep --help` 输出 usage。
- [x] `/tmp/testloop-testgen-v0.4.12-prep --help` 输出 usage。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.4.12-prep scripts/package-release-asset.sh v0.4.12 darwin_arm64 darwin arm64`
- [x] `/tmp/testloop-v0.4.12-prep/testloop-mcp_v0.4.12_darwin_arm64.tar.gz.sha256` 校验通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.4.12`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.4.12 - 2026-07-09`。
- [x] 同步 README 中当前 Release、手动下载示例、Windows 下载示例到 `v0.4.12`。
- [x] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.4.12`。
- [x] 重新运行完整验证：`go test ./...`、脚本语法检查、actionlint、主服务/CLI 构建、打包 dry-run。
- [ ] 提交版本准备改动后确认远端 CI 通过。
- [ ] 打 tag `v0.4.12` 并推送。
- [ ] 等待 Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.4.12` 验证 Release 资产。
- [ ] 验证 Release 资产直连下载、checksum 校验和解包内容。
- [ ] 更新 Homebrew tap 到 `0.4.12` 并跑 `brew fetch` / `brew audit` / `brew upgrade` / `brew test`。

## 当前结论

v0.4.12 已完成本地版本准备和发布前验证。当前代码和用户安装文档已切到 `0.4.12`，但尚未打 tag、尚未生成 Release 资产，也尚未更新 Homebrew tap。
