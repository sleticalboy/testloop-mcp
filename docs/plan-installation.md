# 安装与分发体验规划

## 目标

让新用户可以从 GitHub Release、源码构建或 Docker 三条路径安装 testloop-mcp，并能快速接入 Codex、Claude Code / Claude Desktop 和 Cursor。

## 范围

1. [x] 修正仓库地址和 Go module path，避免 README 与远端仓库不一致。
2. [x] 补齐 MIT `LICENSE` 文件，修复 README License badge 指向空文件的问题。
3. [x] 新增详细安装文档，覆盖 Release 下载、checksum 校验、源码构建、Docker、stdio、Streamable HTTP 和常见客户端配置。
4. [x] README 安装部分改成快速路径，并链接详细安装文档。
5. [x] 验证 `go install github.com/sleticalboy/testloop-mcp@main` 和 `go install github.com/sleticalboy/testloop-mcp/cmd/testgen@main` 可从远端安装并正常显示 help。
6. [x] 准备 v0.4.1 patch release，把 module path 修正纳入正式版本，并将安装文档切回 `@latest`。
7. [x] 发布 v0.4.1，并验证 Release 资产和 `go install @latest`。
8. [x] 评估并准备 Linux arm64、macOS arm64 和多架构二进制发布。
9. [x] 新增一键安装脚本，降低本地安装门槛。
10. [x] 准备 Homebrew Formula 草案和本地生成脚本。
11. [x] 接入 `sleticalboy/homebrew-tap` 同步脚本，支持本地验证后可选提交推送。
12. [x] 新增独立 Homebrew Tap workflow，按 release tag 创建或更新 tap PR，避免阻塞 Release Artifacts workflow。
13. [x] README 和安装文档将 Homebrew tap 更新为可用安装路径。

## 暂缓项

- Windows arm64 预构建二进制：项目使用 CGO 和 tree-sitter，当前先发布 Windows amd64；Windows arm64 暂缓到工具链需求明确后再评估。
- Homebrew tap 发布结果：独立 Homebrew Tap workflow 依赖仓库 secret `HOMEBREW_TAP_TOKEN`。没有配置时不能自动开 PR，但不影响 Release Artifacts workflow 上传资产。

## v0.4.1 发布验证

- [x] Tag `v0.4.1` 指向 `c69e717bf9da5c783bad5d1928d29d97b89deb79`。
- [x] GitHub Release 已创建：https://github.com/sleticalboy/testloop-mcp/releases/tag/v0.4.1
- [x] Release Artifacts run `28739889556` 通过。
- [x] Release 资产包含 `testloop-mcp_v0.4.1_linux_amd64.tar.gz` 和 `checksums.txt`。
- [x] `go install github.com/sleticalboy/testloop-mcp@latest` 验证通过。
- [x] `go install github.com/sleticalboy/testloop-mcp/cmd/testgen@latest` 验证通过。

## 下一版分发改进

- [x] Release Artifacts workflow 改为 matrix build，准备生成 `linux_amd64`、`linux_arm64` 和 `darwin_arm64` 三类 tarball。
- [x] checksums 改为在发布 job 中统一生成，避免多平台 job 并发上传时互相覆盖。
- [x] 新增 `scripts/install.sh`，支持检测平台、下载 release 资产、校验 checksum、安装到 `~/.local/bin`，并在资产缺失时回退到 `go install`。

## v0.4.2 发布验证

- [x] Tag `v0.4.2` 指向 `f119ca0d505d738b7fb7bb00d1b59f722b4ca972`。
- [x] GitHub Release 已创建：https://github.com/sleticalboy/testloop-mcp/releases/tag/v0.4.2
- [x] Release 资产包含 `testloop-mcp_v0.4.2_linux_amd64.tar.gz`、`testloop-mcp_v0.4.2_linux_arm64.tar.gz`、`testloop-mcp_v0.4.2_darwin_arm64.tar.gz` 和 `checksums.txt`。
- [x] `checksums.txt` 已校验 macOS arm64 资产。
- [x] `TESTLOOP_MCP_VERSION=v0.4.2 sh scripts/install.sh` 已验证可直接下载 macOS arm64 release 资产并安装 `testloop-mcp` / `testloop-testgen`。
- [x] Release Artifacts build jobs 已在 run `28746080130` 中验证 Linux amd64、Linux arm64 和 macOS arm64 均可构建；publish job 因 runner 队列取消，最终资产使用同一批成功构建 artifact 手动上传。

## 下一版自动发布修正

- [x] Release Artifacts workflow 去掉单独 publish job，改为每个 matrix build job 直接上传本平台 tarball 和 `.sha256`。
- [x] `scripts/install.sh` 兼容聚合 `checksums.txt` 和单资产 `.sha256`，下一版 release 即使不生成聚合 checksum 也能正常安装。

## Homebrew tap 准备

- [x] 新增 `Formula/testloop-mcp.rb`，基于 `v0.4.2` 三平台 release 资产填写 URL 和 sha256。
- [x] 新增 `scripts/generate-homebrew-formula.sh`，可从 GitHub Release asset digest 重新生成 formula。
- [x] 新增 `scripts/update-homebrew-tap.sh`，可更新 `sleticalboy/homebrew-tap` 工作区并运行 Ruby/Homebrew style 校验。
- [x] 新增 `.github/workflows/homebrew-tap.yml`，手动输入 tag 后创建或更新 `sleticalboy/homebrew-tap` 的 formula PR。
- [x] `brew install --formula sleticalboy/tap/testloop-mcp` 和 `brew test sleticalboy/tap/testloop-mcp` 已验证 `v0.4.2` 可用。

## v0.4.3 发布准备

- [x] `CHANGELOG.md` 已整理 `v0.4.3 - 2026-07-06`。
- [x] 新增 `docs/plan-release-notes-v0.4.3.md`。
- [x] MCP server implementation version 更新为 `0.4.3`。
- [x] Tag `v0.4.3` 已推送并指向 `ddd645d5943cda338c5175882ddd4a873c239ffb`。
- [x] Release Artifacts run `28761435820` 已通过，并上传三平台 tarball 与 `.sha256`。
- [x] `TESTLOOP_MCP_VERSION=v0.4.3 sh scripts/install.sh` 已验证可直接下载 release 资产并安装。
- [x] `sleticalboy/homebrew-tap` 已更新 `testloop-mcp` formula 到 `0.4.3`。
- [x] `brew upgrade --formula sleticalboy/tap/testloop-mcp` 和 `brew test sleticalboy/tap/testloop-mcp` 已验证 `0.4.3` 可用。

## 下一版发布维护

- [x] 新增 `scripts/package-release-asset.sh`，将 release 资产构建、打包和 `.sha256` 生成从 workflow YAML 中抽出。
- [x] Release Artifacts workflow 改为调用 `scripts/package-release-asset.sh`，为后续调试单平台和扩展 Windows zip 资产降低改动面。
- [x] 新增 `.github/workflows/windows-release-probe.yml`，手动验证 `windows_amd64` zip 构建和 `.sha256`，不影响正式 release matrix。
- [x] Windows Release Probe 首次运行已确认 MSYS2 安装成功，但 `go mod download` 在 MSYS2 shell 中找不到 `go`；workflow 已调整为 `path-type: inherit`。
- [x] Windows Release Probe 显式安装 `zip` 和 `unzip`，避免压缩包检查步骤依赖 runner 预装工具。
- [x] Windows Release Probe run `28763453059` 已通过，下载 artifact 校验 `.sha256` 成功，zip 内包含 `testloop-mcp.exe`、`testloop-testgen.exe`、`README.md` 和 `LICENSE`。
- [x] Release Artifacts workflow 已加入 `windows_amd64` matrix 项，后续 tag release 会上传 Windows zip 和 `.sha256`。
- [x] Release Artifacts workflow 会把触发 ref 中的 `scripts/package-release-asset.sh` 复制到 `.release/` 后再 checkout 指定 tag，避免手动回填旧 tag 时目标 tag 缺少新脚本。
- [x] Release Artifacts workflow 已修复 Windows matrix 下 bash 步骤误用 PowerShell，以及非 Windows 上传时缺少 zip glob 导致失败的问题。
- [x] `v0.4.3` 已通过 Release Artifacts run `28763866528` 回填 Windows amd64 zip 和 `.sha256`。
- [x] `scripts/install.sh` 已支持 Windows shell 下下载、校验、解压并安装 `windows_amd64` zip；已用 `TESTLOOP_MCP_OS=windows TESTLOOP_MCP_ARCH=amd64 TESTLOOP_MCP_VERSION=v0.4.3` 在本机验证。
- [x] 临时 Windows Release Probe workflow 已移除；后续以正式 Release Artifacts matrix 维护 Windows 打包链路。

## v0.4.4 发布准备

- [x] `CHANGELOG.md` 已整理 `v0.4.4 - 2026-07-06`。
- [x] 新增 `docs/plan-release-notes-v0.4.4.md`。
- [x] MCP server implementation version 更新为 `0.4.4`。
- [x] README 和安装文档已更新到 `v0.4.4` release 资产命名。
- [x] Tag `v0.4.4` 已推送并指向 `c91ae92a7e95eed2c7c674225699125143671066`。
- [x] Release Artifacts run `28764619084` 已通过，并上传四平台资产与 `.sha256`。
- [x] `TESTLOOP_MCP_VERSION=v0.4.4 sh scripts/install.sh` 已验证可直接下载 release 资产并安装。
- [x] Windows amd64 zip 已下载并通过 `.sha256` 校验，内容包含 `testloop-mcp.exe`、`testloop-testgen.exe`、`README.md` 和 `LICENSE`。
- [x] `sleticalboy/homebrew-tap` 已更新 `testloop-mcp` formula 到 `0.4.4`，commit `39e2ce3`。
- [x] `brew upgrade --formula sleticalboy/tap/testloop-mcp` 和 `brew test sleticalboy/tap/testloop-mcp` 已验证 `0.4.4` 可用。

## 下一版发布维护

- [x] Release Artifacts workflow 已补充上传前资产验证：校验 `.sha256`，并检查 tarball/zip 内包含两个二进制、`README.md` 和 `LICENSE`。
- [x] Release Artifacts workflow run `28765386761` 已验证 Linux amd64、Linux arm64、macOS arm64 和 Windows amd64 的上传前资产校验均通过。
