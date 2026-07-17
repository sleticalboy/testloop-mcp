# v0.5.1 发布说明草案

## 标题

testloop-mcp v0.5.1

## 发布状态

- [x] 创建 v0.5.1 发布说明草案。
- [x] 梳理 v0.5.0 之后的 MCP 客户端接入、结构化契约和真实 fixture 回归能力。
- [x] 完成本地 release readiness 门禁。
- [x] 正式版本准备时更新 `main.go` MCP implementation version 到 `0.5.1`。
- [x] 正式版本准备时将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.5.1 - 2026-07-17`。
- [x] 正式版本准备时更新 README、安装文档和必要的版本引用。
- [x] 正式发布前重新跑远端 CI、Release Artifacts、资产校验和 Homebrew tap 更新。
- [x] Post-Release Verify run `29593507242` 已完成，资产清单和五平台安装脚本 dry run 全部通过。
- [ ] GitHub/Homebrew 网络恢复稳定后补跑本机安装级 `brew fetch`。

## 摘要

v0.5.1 候选重点是把 v0.5.0 的“AI Agent 测试反馈闭环”进一步产品化为可复制的客户端接入契约。

这个版本不追求新增语言清单，而是把 MCP 客户端真实会依赖的关键路径固定下来：

- 客户端优先消费 `structuredContent`，旧 SDK 再 fallback 到 `content[0].text` JSON。
- `validate_coverage_task` 的 `status/action` 分流有真实 handler fixture 保护。
- 最小 Agent 决策 demo、MCP 客户端端到端 demo、配置 roundtrip、自检脚本和 release 文档入口都进入 CI。
- 接入方可以把 fixture 和契约测试复制到自己的客户端项目中，降低集成漂移风险。

## 主要变化

### MCP 客户端端到端接入

- 新增 `examples/mcp-client-demo`，演示客户端优先读取 `structuredContent`，串联 `run_tests -> repair_task -> rerun -> parse_coverage`。
- 新增 `test/mcp_client_demo_test.sh`，固定最小 MCP 客户端闭环 demo 的预期输出。
- 新增真实 stdio 和 Streamable HTTP 进程级 e2e smoke，覆盖 MCP SDK 客户端真实接入路径。
- 新增 `scripts/verify-client-setup.sh`，把二进制可执行性、`--doctor-config`、配置 roundtrip 和 HTTP `/healthz` 收敛为安装后自检入口。
- 新增 `test/verify_client_setup_test.sh`，固定自检脚本的 skip HTTP 路径和缺失二进制错误提示。

### Agent 结构化契约

- 新增 `docs/agent-contract.md` 和 `types` 层 Agent JSON contract 测试，固定 `repair_task`、`provider_error`、`validate_coverage_task` 等关键结构化字段名。
- 新增 `docs/agent-action-guide.md`，整理 `validate_coverage_task.status/action` 到客户端下一步动作的决策表。
- 新增 `docs/validate-coverage-task-samples.md` 和 `test/docs_json_examples_test.sh`，固定典型结构化返回样例的 JSON 合法性。
- 新增 `docs/client-integration.md`，说明客户端消费 `structuredContent`、复用真实 fixture 和回归 `status/action` 分流的推荐流程。
- 新增 `docs/mcp-client-contract-tests.md`，说明接入方如何复制真实 fixture、demo 和契约校验到自己的客户端 CI。

### 真实 fixture 和客户端动作映射

- 新增四份真实 handler fixture：
  - `docs/fixtures/validate-coverage-task-ready.json`
  - `docs/fixtures/validate-coverage-task-manual-review-internal.json`
  - `docs/fixtures/validate-coverage-task-apply-fix-suggestions.json`
  - `docs/fixtures/validate-coverage-task-needs-better-input.json`
- 新增 handler 级 fixture 测试，分别固定 `ready`、`manual_review_internal`、`apply_fix_suggestions` 和 `needs_better_input`。
- `examples/agent-decision-demo` 改为直接读取 `docs/fixtures/*.json`，不再依赖 Markdown 中的手写样例。
- 新增 `test/fixtures_index_test.sh`、`test/fixture_decision_mapping_test.sh` 和 `test/client_integration_doc_test.sh`，校验 fixture 索引、客户端动作映射和文档引用持续一致。
- 新增 `test/release_doc_index_test.sh`，固定 README 中 Agent/客户端关键文档入口和 demo 命令。

### 文档和演示路径

- 新增 `docs/quickstart.md`，把安装、自检、Codex/Claude/Cursor 配置和最小 Agent 闭环收敛成 5 分钟接入路径。
- 新增 `docs/showcase-agent-loop.md`，说明最小 MCP 客户端闭环 demo 的价值和预期输出。
- 新增 `docs/showcase.md`，统一说明默认 CI、公开 opt-in showcase 和真实项目 regression smoke 的边界。
- 新增公开项目 showcase 脚本和文档：
  - `scripts/showcase-go-public-project.sh`
  - `docs/showcase-public-go.md`
  - `scripts/showcase-js-public-project.sh`
  - `docs/showcase-public-js.md`

## 质量边界

v0.5.1 保持以下边界：

- 当前已完成 tag、GitHub Release、Release Artifacts、资产校验和 Homebrew tap 更新。
- 本机 `brew fetch` 下载验证受 GitHub/Homebrew 网络队列影响未完成；资产存在性和 sha256 已由 Release API、`scripts/verify-release-assets.sh v0.5.1`、仓库 Formula 与 tap Formula 校验覆盖。
- fixture 是稳定投影，不包含 `raw_output`、耗时和临时目录绝对路径。
- 客户端动作映射当前覆盖四条核心分流；新增 action 时必须同步更新 fixture、决策映射和文档。
- 公开 showcase 脚本是 opt-in，不进入默认 CI，因为依赖 GitHub、npm registry 或外部仓库可达性。
- 真实项目 regression smoke 仍是代表性样本，不是完整 benchmark。

## 本地验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/verify-release-assets.sh`
- [x] `go test ./...`
- [x] `sh test/install_script_test.sh`
- [x] `sh test/release_assets_test.sh`
- [x] `sh test/llm_provider_example_test.sh`
- [x] `sh test/verify_client_setup_test.sh`
- [x] `sh test/mcp_client_demo_test.sh`
- [x] `sh test/agent_decision_demo_test.sh`
- [x] `sh test/showcase_scripts_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/docs_json_examples_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/fixtures_index_test.sh`
- [x] `sh test/fixture_decision_mapping_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `go build -o /tmp/testloop-mcp-v0.5.1-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.1-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.1-prep --help` 输出 usage；Go `flag` 包对 help 返回 exit code 2。
- [x] `/tmp/testloop-testgen-v0.5.1-prep --help` 输出 usage；Go `flag` 包对 help 返回 exit code 2。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.1-prep scripts/package-release-asset.sh v0.5.1 darwin_arm64 darwin arm64`
- [x] `cd /tmp/testloop-v0.5.1-prep && shasum -a 256 -c testloop-mcp_v0.5.1_darwin_arm64.tar.gz.sha256`
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`
- [x] 版本准备提交远端 CI run `29591849021` passed。
- [x] 版本准备文档提交远端 CI run `29592049170` passed。
- [x] Release Artifacts run `29592283968` passed。
- [x] `scripts/verify-release-assets.sh v0.5.1`
- [x] GitHub Release 正文已更新。
- [x] `scripts/generate-homebrew-formula.sh v0.5.1`
- [x] `ruby -c Formula/testloop-mcp.rb`
- [x] `TESTLOOP_MCP_TAP_COMMIT=1 TESTLOOP_MCP_TAP_PUSH=1 scripts/update-homebrew-tap.sh v0.5.1 /Users/binlee/code/open-source/homebrew-tap`
- [x] Homebrew tap 提交 `54d6e7a testloop-mcp 0.5.1` 已推送。
- [x] Post-Release Verify run `29593507242` passed，资产清单和五平台安装脚本 dry run 全部通过。
- [ ] 本机 `HOMEBREW_NO_AUTO_UPDATE=1 brew fetch --force sleticalboy/tap/testloop-mcp` 仍卡在下载阶段，已中止等待后续网络恢复。

## 发布备注

- v0.5.1 适合作为“客户端契约和真实 fixture 回归保护”的版本节点。
- 发布文案应突出 MCP 客户端可复制接入，而不是继续宣传“自动生成测试”。
- 正式发布前需要重新确认远端 CI、Release Artifacts、GitHub Release 正文、资产校验和 Homebrew tap。
