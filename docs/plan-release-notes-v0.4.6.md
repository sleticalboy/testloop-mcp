# v0.4.6 发布说明

## 标题

testloop-mcp v0.4.6

## 摘要

v0.4.6 是 Homebrew formula 测试修复和发布资料同步版本。这个版本不改变 MCP 工具协议或测试生成行为，重点把 `v0.4.5` 发布后已经验证通过的 formula `--help` 测试修复纳入正式 release source archive，并把 README、安装文档和发布维护记录同步到 `v0.4.6`。

## 主要变化

- MCP server implementation version 更新为 `0.4.6`。
- Homebrew formula 测试改为使用 `shell_output(..., 2)` 断言 help 输出，修复 Go `flag` 的 `--help` 退出码导致 `brew test` 失败的问题。
- `scripts/generate-homebrew-formula.sh` 模板同步上述 test block，后续生成公式不会回退到失败写法。
- README、`docs/installation.md` 和 `docs/plan-installation.md` 同步当前发布版本。

## 验证

- [x] `go test ./...`
- [x] `git diff --check`
- [x] `sh -n scripts/generate-homebrew-formula.sh`
- [x] `ruby -c Formula/testloop-mcp.rb`
- [x] 远端 CI passed
- [x] Tag `v0.4.6` 已推送
- [x] Release Artifacts run `28782811885` 通过
- [x] `v0.4.6` Release 已包含 Linux amd64、Linux arm64、macOS arm64 和 Windows amd64 四类资产及各自 `.sha256`
- [x] `TESTLOOP_MCP_VERSION=v0.4.6 sh scripts/install.sh` 已验证可直接下载 release 资产并安装
- [x] Windows amd64 zip 已下载并通过 `.sha256` 校验，内容包含 `testloop-mcp.exe`、`testloop-testgen.exe`、`README.md` 和 `LICENSE`
- [x] `sleticalboy/homebrew-tap` 已更新 `testloop-mcp` formula 到 `0.4.6`
- [x] `brew fetch --force --formula sleticalboy/tap/testloop-mcp`
- [x] `brew audit --strict --new sleticalboy/tap/testloop-mcp`
- [x] `brew upgrade --formula sleticalboy/tap/testloop-mcp` 已验证可从 `0.4.5` 升级到 `0.4.6`
- [x] `brew test sleticalboy/tap/testloop-mcp`

## 发布信息

- Tag: `v0.4.6` -> `343ba12496ed1a08147bd7efeea805333250e08e`
- Release: https://github.com/sleticalboy/testloop-mcp/releases/tag/v0.4.6
- Release Artifacts run: `28782811885`
- Homebrew tap commit: `f449d28 Update testloop-mcp to v0.4.6`
