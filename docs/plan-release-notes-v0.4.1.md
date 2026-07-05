# v0.4.1 发布说明

## 标题

testloop-mcp v0.4.1

## 摘要

v0.4.1 是安装与分发体验修复版本。这个版本不改变 MCP 工具能力，重点修正仓库路径、Go module path、License、安装文档和 Release 产物内容，让用户可以通过 GitHub Release 或 `go install @latest` 正常安装。

## 主要变化

- Go module path 统一为 `github.com/sleticalboy/testloop-mcp`，与当前 GitHub 远端仓库一致。
- README、DESIGN 和内部 import 同步使用 `github.com/sleticalboy/testloop-mcp`。
- 新增 MIT `LICENSE` 文件。
- 新增 `docs/installation.md`，覆盖 Release 下载、checksum 校验、源码构建、Docker、stdio、Streamable HTTP、Codex、Claude 和 Cursor 配置。
- README 安装部分补充 Release 下载、checksum 校验和 `go install @latest`。
- Release workflow 打包内容包含 `README.md` 和 `LICENSE`。
- MCP server implementation version 更新为 `0.4.1`。

## 验证

- [x] `go test ./...`
- [x] 主服务构建通过
- [x] CLI 构建通过
- [x] release 打包流程本机原生验证通过
- [x] 远端 `go install github.com/sleticalboy/testloop-mcp@main` 验证通过
- [x] 远端 CI passed

## 发布信息

- Tag: `v0.4.1`
- Release: https://github.com/sleticalboy/testloop-mcp/releases/tag/v0.4.1
