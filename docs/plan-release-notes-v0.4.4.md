# v0.4.4 发布说明

## 标题

testloop-mcp v0.4.4

## 摘要

v0.4.4 是 Windows 分发和安装体验完善版本。这个版本不改变 MCP 工具协议或测试生成能力，重点把 Windows amd64 预构建 zip 纳入正式 Release Artifacts 和安装脚本路径，并清理临时探针 workflow。

## 主要变化

- Release 资产打包逻辑抽到 `scripts/package-release-asset.sh`，正式 workflow 和本地调试复用同一套 tarball/zip 打包逻辑。
- Release Artifacts workflow 新增 `windows_amd64` matrix 项，后续 tag release 会上传 Windows zip 和 `.sha256`。
- 安装脚本支持在 Git Bash/MSYS/Cygwin 等 Windows shell 下下载、校验、解压并安装 `windows_amd64` zip。
- 安装脚本在缺少匹配资产或缺少 zip 解压工具时仍会回退到 `go install`。
- 移除临时 Windows Release Probe workflow，Windows 打包链路统一由正式 Release Artifacts matrix 维护。
- MCP server implementation version 更新为 `0.4.4`。

## 验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `ruby` 解析 `.github/workflows/*.yml`
- [x] `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/*.yml`
- [x] `TESTLOOP_MCP_VERSION=v0.4.3 sh scripts/install.sh` 已验证可直接下载 macOS arm64 release 资产并安装
- [x] `TESTLOOP_MCP_OS=windows TESTLOOP_MCP_ARCH=amd64 TESTLOOP_MCP_VERSION=v0.4.3 sh scripts/install.sh` 已验证可直接下载 Windows amd64 zip 并安装 `.exe`
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.4.4-local-package scripts/package-release-asset.sh v0.4.4 darwin_arm64 darwin arm64` 已验证本地打包、checksum 和 tarball 内容
- [x] `go test ./...`
- [x] `git diff --check`
- [x] 远端 CI run `28764570560` passed
- [x] Tag `v0.4.4` 已推送并指向 `c91ae92a7e95eed2c7c674225699125143671066`
- [x] Release Artifacts run `28764619084` 已通过
- [x] `v0.4.4` Release 已包含 Linux amd64、Linux arm64、macOS arm64 和 Windows amd64 四类资产及各自 `.sha256`
- [x] `TESTLOOP_MCP_VERSION=v0.4.4 sh scripts/install.sh` 已验证可直接下载 release 资产并安装
- [x] Windows amd64 zip 已下载并通过 `.sha256` 校验，内容包含 `testloop-mcp.exe`、`testloop-testgen.exe`、`README.md` 和 `LICENSE`
- [x] `sleticalboy/homebrew-tap` 已更新 `testloop-mcp` formula 到 `0.4.4`，commit `39e2ce3`
- [x] `brew fetch --force --formula sleticalboy/tap/testloop-mcp`
- [x] `brew audit --strict --new sleticalboy/tap/testloop-mcp`
- [x] `brew upgrade --formula sleticalboy/tap/testloop-mcp` 已验证可从 `0.4.3` 升级到 `0.4.4`
- [x] `brew test sleticalboy/tap/testloop-mcp` 已验证 `0.4.4`

## 发布信息

- Tag: `v0.4.4`
- Release: https://github.com/sleticalboy/testloop-mcp/releases/tag/v0.4.4
