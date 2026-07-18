# v0.5.2 发布说明草案

## 标题

testloop-mcp v0.5.2

## 发布状态

- [x] 创建 v0.5.2 发布说明草案。
- [x] 梳理 v0.5.1 之后的安装验收、真实 MCP 进程 smoke 和公开 showcase 收敛内容。
- [x] 完成本地 release readiness 门禁。
- [x] 正式版本准备时更新 `main.go` MCP implementation version 到 `0.5.2`。
- [x] 正式版本准备时将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.5.2 - 2026-07-18`。
- [x] 正式版本准备时更新 README、安装文档和必要的版本引用。
- [ ] 正式发布前重新跑远端 CI、Release Artifacts、资产校验和 Homebrew tap 更新。

## 摘要

v0.5.2 候选重点是把 v0.5.1 的客户端契约进一步落成可演示、可验收、可复验的公开路径。

这个版本仍不以“新增语言支持”为主线，而是围绕 AI Agent 的测试反馈闭环补强三类能力：

- 安装后可以用版本门禁和真实 MCP SDK 客户端做更深的接入验收。
- 首次接入可以从基础安装验收一路演示到最小 Agent 闭环。
- 公开 Go / JS showcase 不只打印结果，还会断言 `action` 决策信号，并能在网络不稳定时复用本地 checkout 或快速超时失败。

## 主要变化

### 安装后验收

- 主二进制新增 `--version`，输出 `testloop-mcp <version>`。
- MCP implementation version 与 CLI version 共用同一个 `appVersion` 常量，减少发版时版本漂移。
- `scripts/verify-client-setup.sh` 新增 `TESTLOOP_MCP_VERIFY_EXPECT_VERSION`，可检查当前 PATH 或指定二进制是否为预期版本。
- README、quickstart 和安装文档已区分基础安装验收与深度协议验收。

### 真实 MCP 进程级 smoke

- 新增 `examples/mcp-process-smoke`，使用 MCP SDK 客户端连接真实 `testloop-mcp` 进程。
- stdio 路径通过 `CommandTransport` 启动指定二进制，执行 `tools/list` 和轻量 `parse_results`。
- Streamable HTTP 路径启动指定二进制、等待 `/healthz`，再通过 `StreamableClientTransport` 调用同一组轻量工具。
- `scripts/verify-mcp-process-smoke.sh` 提供安装后单命令协议验收入口，支持 `TESTLOOP_MCP_CLIENT_SMOKE_TRANSPORT=stdio|http|all`。
- `test/mcp_process_smoke_test.sh` 已纳入 CI，构建当前仓库临时二进制后跑真实进程级客户端 smoke。

### Onboarding 和公开 showcase

- 新增 `scripts/showcase-onboarding.sh` 和 `docs/showcase-onboarding.md`，串联基础安装验收、真实 MCP 进程协议验收和最小 Agent 闭环 demo。
- 公开 Go / JS showcase 新增默认 action 期望校验：
  - Go 默认 `go-test-1=ready`。
  - JS 默认 `vitest-1=manual_review_internal,vitest-2=ready`。
- 公开 Go / JS showcase 支持 `TESTLOOP_SHOWCASE_*_PROJECT_DIR` 复用本地 checkout，避免每次演示都依赖 GitHub clone。
- JS showcase 支持 `TESTLOOP_SHOWCASE_JS_SKIP_INSTALL=true`，可在依赖已准备好时跳过 `pnpm install`。
- 公开 Go / JS showcase 支持 `TESTLOOP_SHOWCASE_*_GIT_TIMEOUT` 控制远端 clone/fetch 超时，默认 60 秒。
- 公开 JS/TS showcase 已用远端 `unjs/ufo` 固定 commit 复验通过，默认 action 决策信号保持稳定。

### Showcase 证据归档

- 新增 `scripts/summarize-showcase-output.py`，统一公开 showcase 的 JSONL summary 输出和 action 断言逻辑。
- Go / JS showcase 共用同一个 summary 脚本，减少重复维护。
- 新增 `test/showcase_summary_test.sh`，固定 summary 输出、action 漂移失败和非法期望失败。
- showcase 文档明确 JSONL 明细默认保留在 `/tmp` 或用户指定路径，仓库只归档精简 summary 和关键任务摘要。

## 质量边界

- v0.5.2 仍是 Agent 测试反馈闭环的接入体验版本，不宣传为“通用自动生成测试神器”。
- 公开 showcase 是 opt-in，不进入默认 CI，因为依赖 GitHub、npm registry 或外部项目可达性。
- 默认 CI 只保护仓库内稳定路径：Go 测试、MCP 传输 smoke、客户端配置 smoke、安装脚本、文档链接、fixture 和 showcase 脚本契约。
- 公开项目 JSONL 明细不提交进仓库，避免外部项目生成结果和本机路径污染仓库。
- Homebrew、本机下载和公开 showcase 远端 clone 仍可能受外部网络影响，脚本应快速失败并提示复用本地 checkout。

## 本地验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/verify-release-assets.sh`
- [x] `bash -n scripts/showcase-go-public-project.sh scripts/showcase-js-public-project.sh scripts/showcase-onboarding.sh`
- [x] `python3 -m py_compile scripts/summarize-showcase-output.py`
- [x] `go test ./...`
- [x] `sh test/install_script_test.sh`
- [x] `sh test/release_assets_test.sh`
- [x] `sh test/llm_provider_example_test.sh`
- [x] `sh test/verify_client_setup_test.sh`
- [x] `sh test/mcp_process_smoke_test.sh`
- [x] `sh test/mcp_client_demo_test.sh`
- [x] `sh test/agent_decision_demo_test.sh`
- [x] `sh test/showcase_scripts_test.sh`
- [x] `sh test/showcase_summary_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/docs_json_examples_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/fixtures_index_test.sh`
- [x] `sh test/fixture_decision_mapping_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `go build -o /tmp/testloop-mcp-v0.5.2-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.2-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.2-prep --help` 输出 usage。
- [x] `/tmp/testloop-testgen-v0.5.2-prep --help` 输出 usage。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.2-prep scripts/package-release-asset.sh v0.5.2 darwin_arm64 darwin arm64`
- [x] `cd /tmp/testloop-v0.5.2-prep && shasum -a 256 -c testloop-mcp_v0.5.2_darwin_arm64.tar.gz.sha256`
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`
- [x] 正式版本准备时 `main.go`、README、安装文档、quickstart 和 onboarding 文档已同步到 `0.5.2`。
- [x] `CHANGELOG.md` 已收敛为 `v0.5.2 - 2026-07-18`。
- [x] 版本准备后已重新运行 `go test ./...`、全部默认 shell 校验、脚本语法检查、主服务/testgen 构建和打包 dry-run。

## 发布备注

- v0.5.2 适合作为“安装验收 + 真实 MCP 协议 smoke + 公开 showcase 决策断言”的版本节点。
- 发布文案应突出 Agent 接入可验证、公开演示可复验，以及 action 决策信号不会静默漂移。
- 正式发布前需要重新确认远端 CI、Release Artifacts、GitHub Release 正文、资产校验和 Homebrew tap。
