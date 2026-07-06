# v0.4.8 发布说明草案

## 标题

testloop-mcp v0.4.8

## 摘要

v0.4.8 是编辑器接入体验增强版本。这个版本不改变 MCP 工具协议，重点降低 Codex、Claude Code / Claude Desktop 和 Cursor 的首次配置成本，并补齐 Agent 端到端测试闭环示例。

## 主要变化

- MCP server implementation version 更新为 `0.4.8`。
- 主二进制新增 `--print-config`，可输出 Codex、Codex HTTP、Claude Code / Claude Desktop 和 Cursor 的 MCP 配置片段。
- `--print-config` 支持 `--config-command` 和 `--config-http-url`，便于生成指定二进制路径或 HTTP endpoint 的配置。
- 主二进制新增 `--check-config`，可读取配置文件或 stdin，检查 MCP server 的 `command` 是否存在且可执行，或 `url` 是否是合法 HTTP endpoint。
- `--check-config` 支持校验 `--print-config=all` 的混合 TOML/JSON 输出，便于直接管道验证配置片段。
- 主二进制新增 `--doctor-config`，可输出当前二进制路径、PATH 解析结果、推荐配置路径，并对已存在的 Codex、Claude 和 Cursor 配置做只读校验；配置存在但缺少 `testloop` server 时，会明确列出已发现的其他 MCP server。
- 新增 `scripts/generate-client-config.sh`，作为源码仓库里的配置片段生成辅助入口。
- 新增 `docs/agent-workflow.md`，展示 `run_tests -> parse_results -> parse_coverage -> generate_tests -> run_tests` 的 Agent 闭环顺序。
- README、安装文档、路线图和质量评估同步当前配置体验与解析能力状态。

## 验证

- [x] `go test ./...`
- [x] `go build -o /tmp/testloop-mcp-v0.4.8 .`
- [x] `go build -o /tmp/testloop-testgen-v0.4.8 ./cmd/testgen`
- [x] `git diff --check`
- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-client-config.sh`
- [x] `go run . --print-config=all --config-command="$(command -v testloop-mcp || command -v true)" | go run . --check-config -`
- [x] `go run . --doctor-config`
- [x] `/tmp/testloop-mcp-v0.4.8 --print-config=codex --config-command=/tmp/testloop-mcp-v0.4.8 | /tmp/testloop-mcp-v0.4.8 --check-config -`
- [x] `/tmp/testloop-mcp-v0.4.8 --doctor-config`
- [x] `ruby -e 'require "yaml"; Dir[".github/workflows/*.yml"].each { |f| YAML.load_file(f) }'`
- [x] `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/*.yml`
- [x] 非法 URL 校验失败分支验证
- [x] `go test ./demo -coverprofile=/tmp/testloop-demo-coverage.out`
- [x] MCP server implementation version 已更新到 `0.4.8`
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
- 发布前不要把 README 和安装文档里的 release 下载链接切到 `v0.4.8`，直到 Release Artifacts 已经上传资产。
