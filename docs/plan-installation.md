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
7. [ ] 发布 v0.4.1，并验证 Release 资产和 `go install @latest`。
8. [ ] 后续评估是否发布 macOS、Windows 和多架构二进制。
9. [ ] 后续评估 Homebrew tap 或一键安装脚本，降低本地安装门槛。

## 暂缓项

- 多平台二进制：项目使用 CGO，跨平台构建需要额外工具链验证，先不在第十四阶段一次性铺开。
- 包管理器发布：需要稳定版本节奏和产物命名后再接入 Homebrew 或其他分发渠道。
