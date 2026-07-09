# v0.4.11 发布前检查清单

## 当前目标

这是 v0.4.11 的发布前差异检查记录。当前只做准备与核对，不打 tag、不创建 GitHub Release、不更新 Homebrew tap。

v0.4.11 候选发布重点见 [v0.4.11 发布说明草案](./plan-release-notes-v0.4.11.md)：本轮主要是 JS/TS 静态生成质量增强，不新增 MCP 工具协议。

## 当前差异核对

- [x] `docs/plan-release-notes-v0.4.11.md` 已创建，未回写已发布的 `docs/plan-release-notes-v0.4.10.md`。
- [x] `CHANGELOG.md` 已新增 `v0.4.11 - 2026-07-09` 小节，收敛本轮候选能力点。
- [x] `main.go` MCP implementation version 已更新为 `0.4.11`。
- [x] README 当前 Release、手动下载示例和 Windows 下载示例已同步到 `v0.4.11`。
- [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.4.11`。
- [x] `scripts/install.sh` 默认使用 `latest`，不需要为 v0.4.11 预先改脚本。
- [x] `scripts/package-release-asset.sh`、`scripts/generate-homebrew-formula.sh`、`scripts/update-homebrew-tap.sh` 和 `scripts/verify-release-assets.sh` 都按 tag 参数工作，不需要为 v0.4.11 改脚本。
- [x] GitHub Actions release workflow 仍按 tag 触发生成五平台资产，不需要为 v0.4.11 改 workflow。

## 已验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-client-config.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/verify-release-assets.sh`
- [x] `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/*.yml`
- [x] `go test ./...`
- [x] `go build -o /tmp/testloop-mcp-v0.4.11-precheck .`
- [x] `go build -o /tmp/testloop-testgen-v0.4.11-precheck ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.4.11-precheck --help` 输出 usage；当前 flag 行为返回 exit code 2。
- [x] `/tmp/testloop-testgen-v0.4.11-precheck --help` 输出 usage；当前 flag 行为返回 exit code 2。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.4.11-precheck scripts/package-release-asset.sh v0.4.11 darwin_arm64 darwin arm64`
- [x] `/tmp/testloop-v0.4.11-precheck/testloop-mcp_v0.4.11_darwin_arm64.tar.gz.sha256` 校验通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] 版本切换后重新运行 `git diff --check`。
- [x] 版本切换后重新运行脚本语法检查：`sh -n scripts/install.sh`、`sh -n scripts/package-release-asset.sh`、`bash -n scripts/update-homebrew-tap.sh`、`bash -n scripts/generate-client-config.sh`、`bash -n scripts/generate-homebrew-formula.sh`、`bash -n scripts/verify-release-assets.sh`。
- [x] 版本切换后重新运行 `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/*.yml`。
- [x] 版本切换后重新运行 `go test ./...`。
- [x] 版本切换后重新构建 `/tmp/testloop-mcp-v0.4.11-prep` 和 `/tmp/testloop-testgen-v0.4.11-prep`。
- [x] 版本切换后重新检查两个二进制的 `--help` 输出。
- [x] 版本切换后重新执行 `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.4.11-prep scripts/package-release-asset.sh v0.4.11 darwin_arm64 darwin arm64`，sha256 和 tarball 内容校验通过。

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.4.11`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.4.11 - 2026-07-09`。
- [x] 同步 README 中当前 Release、手动下载示例、Windows 下载示例到 `v0.4.11`。
- [x] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.4.11`。
- [x] 重新运行完整验证：`go test ./...`、脚本语法检查、actionlint、主服务/CLI 构建、打包 dry-run。
- [x] 提交版本准备改动后确认远端 CI 通过：`8232e6b` 对应 CI run `28995406760` 已通过。
- [ ] 打 tag `v0.4.11` 并推送。
- [ ] 等待 Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.4.11` 验证 Release 资产。
- [ ] 验证安装脚本下载、checksum 校验和安装路径。
- [ ] 更新 Homebrew tap 到 `0.4.11` 并跑 `brew fetch` / `brew audit` / `brew upgrade` / `brew test`。

## 当前结论

v0.4.11 的版本准备改动已经通过本地完整验证，并且提交 `8232e6b` 的远端 CI 已通过；发布脚本和 workflow 无需为本版本做结构性改动。当前仍不打 tag，下一步进入 tag、Release Artifacts 和 Homebrew tap 发布核验阶段。
