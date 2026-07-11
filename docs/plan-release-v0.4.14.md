# v0.4.14 发布检查清单

## 当前目标

这是 v0.4.14 的版本准备和正式发布记录。版本准备、tag、GitHub Release、Release Artifacts、资产校验、GitHub Release 正文和 Homebrew tap 发布核验均已完成；Post-Release Verify 已触发，但当前 GitHub runner 仍处于 queued 状态，尚未给出通过或失败结论。

v0.4.14 发布重点见 [v0.4.14 发布说明](./plan-release-notes-v0.4.14.md)：本轮主要是 Go coverage task 闭环质量、`validate_coverage_task`、skipped task 分类、真实 Go 项目 top task 隔离验证，以及多类 Go static generator seed 增强。

## 当前差异核对

- [x] `docs/plan-release-notes-v0.4.14.md` 已创建，未回写已发布的 `docs/plan-release-notes-v0.4.13.md`。
- [x] `CHANGELOG.md` 已新增 `v0.4.14 - 2026-07-11` 小节，收敛本轮候选能力点。
- [x] `main.go` MCP implementation version 已更新为 `0.4.14`。
- [x] README 当前 Release、手动下载示例和 Windows 下载示例已同步到 `v0.4.14`。
- [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.4.14`。
- [x] `scripts/install.sh` 默认使用 `latest`，不需要为 v0.4.14 预先改脚本。
- [x] `scripts/package-release-asset.sh`、`scripts/generate-homebrew-formula.sh`、`scripts/update-homebrew-tap.sh` 和 `scripts/verify-release-assets.sh` 都按 tag 参数工作，不需要为 v0.4.14 改脚本。
- [x] GitHub Actions release workflow 仍按 tag 触发生成五平台资产，不需要为 v0.4.14 改 workflow。

## 已验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-client-config.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/verify-release-assets.sh`
- [x] `bash -n scripts/validate-go-coverage-top-tasks.sh`
- [x] `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/*.yml`
- [x] `sh test/llm_provider_example_test.sh`
- [x] `sh test/install_script_test.sh`
- [x] `sh test/release_assets_test.sh`
- [x] `go test ./...`
- [x] `go build -o /tmp/testloop-mcp-v0.4.14-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.4.14-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.4.14-prep --help` 输出 usage。
- [x] `/tmp/testloop-testgen-v0.4.14-prep --help` 输出 usage。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.4.14-prep scripts/package-release-asset.sh v0.4.14 darwin_arm64 darwin arm64`
- [x] `/tmp/testloop-v0.4.14-prep/testloop-mcp_v0.4.14_darwin_arm64.tar.gz.sha256` 校验通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.4.14`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.4.14 - 2026-07-11`。
- [x] 同步 README 中当前 Release、手动下载示例、Windows 下载示例到 `v0.4.14`。
- [x] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.4.14`。
- [x] 重新运行完整验证：`go test ./...`、脚本语法检查、actionlint、主服务/CLI 构建、打包 dry-run。
- [x] 提交版本准备改动后确认远端 CI 通过：run `29157660797` passed。
- [x] 打 tag `v0.4.14` 并推送。
- [x] 等待 Release Artifacts workflow 生成五平台资产和 `.sha256`：run `29157722825` passed。
- [x] 使用 `scripts/verify-release-assets.sh v0.4.14` 验证 Release 资产，确认 10 个必需资产完整。
- [x] 更新 GitHub Release 正文为正式 v0.4.14 发布说明。
- [ ] 手动触发 Post-Release Verify，确认五平台安装脚本 dry run 全部通过：run `29157901152` 已触发但仍 queued，首个 `ubuntu-latest` job 尚未拿到 runner。
- [x] 更新 Homebrew tap 到 `0.4.14`，并通过 `brew fetch`、`brew audit --formula --strict`、`brew upgrade --formula`、`brew test`。tap commit：`6394533b9f999bd2125efab6ace6f3c1e81da180`。

## 发布后已验证

- [x] `gh run view 29157660797`：CI passed。
- [x] `gh run watch 29157722825 --exit-status`：Release Artifacts passed。
- [x] `scripts/verify-release-assets.sh v0.4.14`：GitHub Release 包含 10 个必需资产。
- [x] `gh release edit v0.4.14 --notes-file ...`：GitHub Release 正文已替换为正式发布说明。
- [x] `scripts/update-homebrew-tap.sh v0.4.14`：Homebrew tap 已提交并推送 `testloop-mcp 0.4.14`。
- [x] `brew fetch --force --formula sleticalboy/tap/testloop-mcp`
- [x] `brew audit --formula --strict sleticalboy/tap/testloop-mcp`
- [x] `brew upgrade --formula sleticalboy/tap/testloop-mcp`
- [x] `brew test sleticalboy/tap/testloop-mcp`
- [ ] `Post-Release Verify` workflow：run `29157901152` queued，尚未开始执行步骤。

## 当前结论

v0.4.14 已正式发布，GitHub Release 资产和 Homebrew 安装链路已验证通过。剩余唯一发布后核验项是等待 `Post-Release Verify` run `29157901152` 拿到 GitHub runner 并完成五平台安装脚本 dry run。
