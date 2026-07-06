# v0.4.7 发布说明

## 标题

testloop-mcp v0.4.7

## 摘要

v0.4.7 是 Windows ARM64 release 资产补充版本。这个版本不改变 MCP 工具协议或测试生成行为，重点是把已经通过 `Windows ARM64 Probe` 验证的 `windows_arm64` 预构建 zip 纳入正式 Release Artifacts matrix，并增强 Windows zip 的发布前运行验证。

## 主要变化

- MCP server implementation version 更新为 `0.4.7`。
- Release Artifacts workflow 新增 `windows_arm64` matrix 项，使用 `windows-11-arm` runner、MSYS2 `CLANGARM64` 和 `mingw-w64-clang-aarch64-clang` 构建 Windows ARM64 zip。
- Windows release 资产上传前会校验 `.sha256`、检查 zip 内容，并实际运行 `testloop-mcp.exe --help` 和 `testloop-testgen.exe --help`。
- README、`docs/installation.md` 和 `docs/plan-installation.md` 同步当前发布版本。

## 验证

- [x] `Windows ARM64 Probe` workflow `28784385589` 通过
- [x] `go test ./...`
- [x] `git diff --check`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `ruby -e 'require "yaml"; Dir[".github/workflows/*.yml"].each { |f| YAML.load_file(f) }'`
- [x] `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/*.yml`
- [ ] 远端 CI passed
- [ ] Tag `v0.4.7` 已推送
- [ ] Release Artifacts run 通过
- [ ] `v0.4.7` Release 已包含 Linux amd64、Linux arm64、macOS arm64、Windows amd64 和 Windows arm64 五类资产及各自 `.sha256`
- [ ] `TESTLOOP_MCP_VERSION=v0.4.7 sh scripts/install.sh` 已验证可直接下载 release 资产并安装
- [ ] Windows amd64 zip 已下载并通过 `.sha256` 校验，内容包含 `testloop-mcp.exe`、`testloop-testgen.exe`、`README.md` 和 `LICENSE`
- [ ] Windows arm64 zip 已下载并通过 `.sha256` 校验，内容包含 `testloop-mcp.exe`、`testloop-testgen.exe`、`README.md` 和 `LICENSE`
- [ ] `sleticalboy/homebrew-tap` 已更新 `testloop-mcp` formula 到 `0.4.7`
- [ ] `brew fetch --force --formula sleticalboy/tap/testloop-mcp`
- [ ] `brew audit --strict --new sleticalboy/tap/testloop-mcp`
- [ ] `brew upgrade --formula sleticalboy/tap/testloop-mcp` 已验证可从 `0.4.6` 升级到 `0.4.7`
- [ ] `brew test sleticalboy/tap/testloop-mcp`

## 发布信息

- Tag: `v0.4.7`（待推送）
- Release: https://github.com/sleticalboy/testloop-mcp/releases/tag/v0.4.7（待创建）
