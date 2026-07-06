# v0.4.8 发布说明草案

## 标题

testloop-mcp v0.4.8

## 摘要

v0.4.8 是编辑器接入体验增强版本。这个版本不改变 MCP 工具协议，重点降低 Codex、Claude Code / Claude Desktop 和 Cursor 的首次配置成本，并补齐 Agent 端到端测试闭环示例。

## 主要变化

- 主二进制新增 `--print-config`，可输出 Codex、Codex HTTP、Claude Code / Claude Desktop 和 Cursor 的 MCP 配置片段。
- `--print-config` 支持 `--config-command` 和 `--config-http-url`，便于生成指定二进制路径或 HTTP endpoint 的配置。
- 主二进制新增 `--check-config`，可读取配置文件或 stdin，检查 MCP server 的 `command` 是否存在且可执行，或 `url` 是否是合法 HTTP endpoint。
- `--check-config` 支持校验 `--print-config=all` 的混合 TOML/JSON 输出，便于直接管道验证配置片段。
- 新增 `scripts/generate-client-config.sh`，作为源码仓库里的配置片段生成辅助入口。
- 新增 `docs/agent-workflow.md`，展示 `run_tests -> parse_results -> parse_coverage -> generate_tests -> run_tests` 的 Agent 闭环顺序。
- README、安装文档、路线图和质量评估同步当前配置体验与解析能力状态。

## 验证

- [x] `go test ./...`
- [x] `git diff --check`
- [x] `bash -n scripts/generate-client-config.sh`
- [x] `go run . --print-config=all --config-command="$(command -v testloop-mcp || command -v true)" | go run . --check-config -`
- [x] 非法 URL 校验失败分支验证
- [x] `go test ./demo -coverprofile=/tmp/testloop-demo-coverage.out`
- [ ] 远端 CI passed
- [ ] Tag `v0.4.8` 已推送
- [ ] Release Artifacts run 通过
- [ ] `v0.4.8` Release 资产验证
- [ ] Homebrew tap 更新到 `0.4.8`
- [ ] `brew test sleticalboy/tap/testloop-mcp`

## 发布信息

- Tag: 待发布
- Release: 待发布
- Release Artifacts run: 待发布
- Homebrew tap commit: 待发布

## 发布前注意

- CI 如果因 GitHub runner 资源排队，应继续完成本地验证和发布资料准备；只有失败结论才需要阻塞发布。
- 发布前需要把 MCP server implementation version 更新到 `0.4.8`，并同步 README、安装文档、CHANGELOG 和发布维护记录。
