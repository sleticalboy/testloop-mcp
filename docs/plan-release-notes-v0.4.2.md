# v0.4.2 发布说明

## 标题

testloop-mcp v0.4.2

## 摘要

v0.4.2 是分发体验增强版本。这个版本不改变 MCP 工具协议或测试生成能力，重点让安装路径更顺滑，并为下一批 Release 产物补齐 Linux arm64 和 macOS arm64。

## 主要变化

- Release Artifacts workflow 改为 matrix build，准备生成 `linux_amd64`、`linux_arm64` 和 `darwin_arm64` 三类 tarball。
- `checksums.txt` 改为在单独 publish job 中统一生成，避免多平台 job 并发上传时互相覆盖。
- 新增 `scripts/install.sh`，支持自动检测系统和架构，优先下载匹配的 GitHub Release 资产并校验 checksum。
- 当前 release 没有匹配资产时，安装脚本会自动回退到 `go install`，并统一安装 `testloop-mcp` 和 `testloop-testgen` 两个命令。
- README 和安装文档已同步一键安装入口，以及 Windows 继续走 `go install` / 源码构建的说明。

## 验证

- [x] `go test ./...`
- [x] `sh -n scripts/install.sh`
- [x] `.github/workflows/release.yml` YAML 解析通过
- [x] `git diff --check`
- [x] 本机 `darwin_arm64` 打包模拟通过
- [x] 安装脚本在 `v0.4.1` 缺少 macOS arm64 资产时可回退到 `go install`
- [x] 远端 CI passed

## 发布信息

- Tag: `v0.4.2`
- Release: https://github.com/sleticalboy/testloop-mcp/releases/tag/v0.4.2
