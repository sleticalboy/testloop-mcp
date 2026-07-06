# v0.4.3 发布说明

## 标题

testloop-mcp v0.4.3

## 摘要

v0.4.3 是分发和发布流水线修正版本。这个版本不改变 MCP 工具协议或测试生成能力，重点修复 v0.4.2 发布后暴露出的 GitHub Actions runner 排队风险，并正式补齐 Homebrew 安装路径。

## 主要变化

- Release Artifacts workflow 去掉单独 publish job，改为每个 matrix build job 直接上传本平台 tarball 和 `.sha256`。
- 安装脚本兼容聚合 `checksums.txt` 和单资产 `.sha256`，后续 release 即使不生成聚合 checksum 也能安装。
- 新增 `Formula/testloop-mcp.rb` 和 `scripts/generate-homebrew-formula.sh`，可从 GitHub Release asset digest 生成 Homebrew formula。
- 新增 `scripts/update-homebrew-tap.sh`，可更新 `sleticalboy/homebrew-tap` 工作区并运行 Ruby/Homebrew style 校验。
- 新增独立 `Homebrew Tap` workflow，手动输入 release tag 后创建或更新 tap PR，避免 Homebrew 自动化阻塞 release 资产发布。
- README 和安装文档新增 `brew tap sleticalboy/tap && brew install testloop-mcp` 安装路径。
- MCP server implementation version 更新为 `0.4.3`。

## 验证

- [x] `go test ./...`
- [x] `ruby` 解析 `.github/workflows/release.yml` 和 `.github/workflows/homebrew-tap.yml`
- [x] `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/release.yml .github/workflows/homebrew-tap.yml`
- [x] `sh -n scripts/install.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `sh -n scripts/update-homebrew-tap.sh`
- [x] `git diff --check`
- [x] `scripts/update-homebrew-tap.sh v0.4.2` 已在临时 tap clone 上验证通过
- [x] `brew install --formula sleticalboy/tap/testloop-mcp` 已验证 `v0.4.2` 可安装
- [x] `brew test sleticalboy/tap/testloop-mcp`
- [x] 远端 CI passed
- [x] Tag `v0.4.3` 指向 `ddd645d5943cda338c5175882ddd4a873c239ffb`
- [x] Release Artifacts run `28761435820` 通过
- [x] `v0.4.3` Release 已包含 Linux amd64、Linux arm64 和 macOS arm64 三类 tarball 及各自 `.sha256`
- [x] Release Artifacts run `28763866528` 已回填 Windows amd64 zip 和 `.sha256`
- [x] `TESTLOOP_MCP_VERSION=v0.4.3 sh scripts/install.sh` 已验证可直接下载 macOS arm64 release 资产并安装
- [x] `sleticalboy/homebrew-tap` 已更新 `testloop-mcp` formula 到 `0.4.3`
- [x] `brew upgrade --formula sleticalboy/tap/testloop-mcp` 已验证可从 `0.4.2` 升级到 `0.4.3`
- [x] `brew test sleticalboy/tap/testloop-mcp` 已验证 `0.4.3`

## 发布信息

- Tag: `v0.4.3`
- Release: https://github.com/sleticalboy/testloop-mcp/releases/tag/v0.4.3
