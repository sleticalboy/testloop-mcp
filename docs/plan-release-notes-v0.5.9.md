# v0.5.9 发布说明草案

## 标题

testloop-mcp v0.5.9

## 发布状态

- [x] 创建 v0.5.9 发布说明草案。
- [x] 梳理 v0.5.8 之后的 first-run artifact Agent 消费 demo、端到端回归、失败 artifact fixture 包、客户端集成文档和 README 入口。
- [x] 远端 CI 已通过到 `75e1c41`，最新成功 run 为 `29670275988`。
- [x] 完成本地候选验证：脚本语法、`go test ./...`、完整 shell 矩阵、文档链接、release doc index 和 `git diff --check`。
- [x] 完成本地发布前门禁：主服务/testgen 构建、help 输出、darwin arm64 打包 dry-run、sha256 校验和 tarball 内容检查。
- [ ] 正式版本准备时更新 `main.go` MCP implementation version 到 `0.5.9`。
- [ ] 正式版本准备时将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.5.9 - 2026-07-19`。
- [ ] 正式版本准备时同步 README、安装文档和必要版本引用到 `v0.5.9`。

## 摘要

v0.5.9 候选重点是把 v0.5.8 的 first-run / onboarding 接入路径继续推进到“Agent/客户端可以稳定消费 CI artifact”的层面。

这个版本不扩语言、不改变 MCP tool 协议，也不调整测试生成算法。核心变化是让失败 artifact 不只停留在文档说明，而是具备可运行 demo、端到端测试和可复用 fixture：

- `examples/first-run-agent-response-demo` 可以读取 `first-run-context.txt` 和 `verification-summary.json`，输出 Agent 回复的四段结构。
- `test/first_run_agent_response_demo_test.sh` 覆盖从 `run-first-run-ci.sh` 失败五件套到 demo 输出的端到端链路。
- `docs/fixtures/first-run-artifacts/user-project-smoke-failed/` 固定一份 first-run 失败 artifact 包，便于客户端/Agent 不跑脚本也能回归消费逻辑。
- `docs/client-integration.md` 明确区分 MCP tool 结构化返回 fixture 和 CI artifact fixture。
- README 从首页链接 artifact demo、demo 命令和失败 artifact fixture。

## 主要变化

### first-run artifact Agent 消费 demo

- 新增 `examples/first-run-agent-response-demo`。
- 输入：
  - `first-run-context.txt`
  - 可选 `verification-summary.json`
- 输出固定四段：
  - 结论
  - 证据
  - 下一步
  - 暂不做
- 当前覆盖 `ready`、`fix-installation`、`inspect-mcp-transport`、`inspect-agent-demo`、`inspect-user-project`、`inspect-showcase` 和未知 action。

### 端到端回归

- `test/first_run_agent_response_demo_test.sh` 不只读取静态 fixture，也会先运行 `scripts/run-first-run-ci.sh` 构造一个用户项目 smoke 失败。
- 测试确认 first-run 输出：
  - `first_run_agent_next_step=inspect-user-project`
  - `verification-summary.json` 中失败 section 为 `用户项目 smoke`
  - exit code 为 `7`
- 随后把真实输出目录里的 `first-run-context.txt` 和 `verification-summary.json` 喂给 demo，固定 Agent 回复内容。

### first-run 失败 artifact fixture

- 新增 `docs/fixtures/first-run-artifacts/user-project-smoke-failed/`。
- fixture 包含 first-run 失败五件套：
  - `verification-report.md`
  - `verification-summary.json`
  - `agent-decision.txt`
  - `first-run-context.txt`
  - `first-run.log`
- `test/first_run_artifact_fixtures_test.sh` 验证文件完整、JSON 可解析、decision/context 字段正确，并能被 demo 消费。

### 客户端集成入口

- `docs/fixtures.md` 新增 first-run artifact fixture 索引。
- `docs/client-integration.md` 明确两类输入：
  - MCP tool 结构化返回 fixture：用于验证 `structuredContent` 和 `status/action` 分流。
  - CI artifact fixture：用于验证 CI 失败后 Agent 如何消费 `agent-decision.txt`、`first-run-context.txt`、summary 和 report。
- README 和 release doc index 固定 first-run artifact demo 文档、demo 命令和 fixture 路径。

## 质量边界

- v0.5.9 是 Agent artifact 消费体验 patch，不是生成质量或覆盖率算法版本。
- demo 不调用 LLM，不修改用户项目，只把 first-run artifact 映射成稳定回复结构。
- fixture 包是客户端/Agent 回归输入，不代表 benchmark。

## 本地验证

- [x] `sh test/first_run_agent_response_demo_test.sh`
- [x] `sh test/first_run_artifact_fixtures_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `git diff --check`
- [x] `go build -o /tmp/testloop-mcp-v0.5.9-candidate .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.9-candidate ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.9-candidate --help` 输出 usage，exit code 为 `2`。
- [x] `/tmp/testloop-testgen-v0.5.9-candidate --help` 输出 usage，exit code 为 `2`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.9-candidate-dist scripts/package-release-asset.sh v0.5.9 darwin_arm64 darwin arm64`
- [x] `cd /tmp/testloop-v0.5.9-candidate-dist && shasum -a 256 -c testloop-mcp_v0.5.9_darwin_arm64.tar.gz.sha256`
- [x] tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。

## 发布备注

- v0.5.9 适合作为“first-run artifact 可被 Agent/客户端稳定消费”的 patch 版本。
- 发布文案应突出：CI 失败后的 artifact 不只是日志，而是能被 demo、fixture 和 Agent 回复格式稳定消费。
