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
9. [x] 新增一键安装脚本，降低本地安装门槛；Homebrew tap 继续暂缓。

## 暂缓项

- Windows 预构建二进制：项目使用 CGO 和 tree-sitter，Windows runner 需要额外 MinGW/MSYS2 工具链验证，暂时保留 `go install` 和源码构建路径。
- Homebrew tap：需要稳定版本节奏和产物命名后再接入。

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
