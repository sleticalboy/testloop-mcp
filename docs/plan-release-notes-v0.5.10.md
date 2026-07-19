# v0.5.10 发布说明草案

## 标题

testloop-mcp v0.5.10

## 发布状态

- [x] 创建 v0.5.10 发布说明草案。
- [x] 梳理 v0.5.9 之后的 first-run / onboarding Agent response artifact 收敛改动。
- [x] `101b14e` 远端 CI run `29673054100` passed，覆盖 onboarding Agent 回复 artifact。
- [x] `d3dbb86` 远端 CI run `29673143525` passed，覆盖外部 onboarding 四件套校验。
- [x] `44071a0` 远端 CI run `29673246805` passed，覆盖 onboarding 失败 artifact fixture。
- [x] 完成本地候选验证：脚本语法、`go test ./...`、完整 shell 矩阵、文档链接、release doc index 和 `git diff --check`。
- [x] 完成本地 release readiness 门禁：主服务/testgen 构建、help 输出、darwin arm64 打包 dry-run、sha256 校验和 tarball 内容检查。
- [x] 候选计划提交 `13ea54b` 远端 CI run `29673325435` passed。
- [x] 正式版本准备已更新 `main.go` MCP implementation version 到 `0.5.10`。
- [x] 正式版本准备已将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.5.10 - 2026-07-19`。
- [x] 正式版本准备已同步 README、安装文档和必要版本引用到 `v0.5.10`。
- [x] 正式版本准备本地完整验证已通过。
- [ ] 正式发布：提交版本准备改动、打 tag、生成 Release Artifacts、更新 GitHub Release 和 Homebrew tap。

## 摘要

v0.5.10 候选重点是把 v0.5.9 已发布的 first-run artifact 消费体验继续下沉到 onboarding CI。这个版本不扩语言、不调整测试生成算法，也不改变 MCP tool 协议；核心是让 CI artifact 更适合被 Codex / Claude / Cursor 这类 Agent 直接消费。

本轮变化围绕一个目标：失败后用户不必先理解 report、summary、decision 的组合关系，先看 `agent-response.txt` 即可获得结构化排查入口。

## 主要变化

### first-run artifact 目录入口与自动回复

- 新增 `scripts/render-first-run-agent-response.sh`，接收 first-run artifact 目录并自动读取 `first-run-context.txt` 和可选 `verification-summary.json`。
- `scripts/run-first-run-ci.sh` 会 best-effort 生成 `agent-response.txt`。
- first-run 失败 artifact fixture 升级为六件套：report、summary、decision、context、Agent response 和 log。
- 外部 first-run showcase 校验六件套 artifact，确认复制型 bootstrap 在非 testloop 项目目录中也产出 Agent response。

### onboarding artifact Agent 回复

- 新增 `examples/onboarding-agent-response-demo`，从 `verification-summary.json` 渲染 Agent 四段回复。
- 新增 `scripts/render-onboarding-agent-response.sh`，接收 onboarding artifact 目录并自动读取 summary。
- `scripts/run-onboarding-ci.sh` 会 best-effort 生成 `agent-response.txt`，并在 GitHub step summary 中列出路径。
- onboarding CI 模板、失败排查、一页式接入指南、验收 CI 文档和 README 已从三件套更新为四件套。

### 外部 onboarding 四件套校验

- `scripts/showcase-onboarding-ci-external-project.sh` 校验 `agent-response.txt` 存在且包含 `agent_next_step=ready`。
- showcase 输出新增 `external_onboarding_*_agent_response` 路径。
- `docs/onboarding-ci-external-dry-run.md` 已补四件套 artifact 和 Go / Node 路径说明。

### onboarding 失败 artifact fixture

- 新增 `docs/fixtures/onboarding-artifacts/user-project-smoke-failed/`。
- fixture 固定用户项目 smoke exit code `7`，`agent_next_step=inspect-user-project`。
- `test/onboarding_artifact_fixtures_test.sh` 验证 fixture 文件完整、summary JSON 可解析、decision/response 字段正确，并能被 onboarding 回复 demo 和目录入口消费。
- `docs/client-integration.md` 已把 CI artifact fixture 扩展为 first-run 和 onboarding 两类输入。

## 质量边界

- v0.5.10 是 Agent artifact 消费体验 patch，不是测试生成质量或覆盖率算法版本。
- `agent-response.txt` 是确定性草稿，不调用 LLM，不修改用户项目。
- first-run 更适合首次安装和 transport 诊断；onboarding 更适合稳定接入后的 PR / 发布后 smoke。

## 本地验证

- [x] `sh test/onboarding_agent_response_demo_test.sh`
- [x] `sh test/run_onboarding_ci_test.sh`
- [x] `sh test/onboarding_ci_template_doc_test.sh`
- [x] `sh test/onboarding_ci_failure_triage_doc_test.sh`
- [x] `sh test/onboarding_ci_external_dry_run_doc_test.sh`
- [x] `sh test/onboarding_artifact_fixtures_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `go build -o /tmp/testloop-mcp-external-onboarding-fourpack .`
- [x] `TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp-external-onboarding-fourpack TESTLOOP_MCP_VERSION=v0.5.9 scripts/showcase-onboarding-ci-external-project.sh`
- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `git diff --check`
- [x] 正式版本准备复跑：脚本语法、`go test ./...`、完整 shell 矩阵、主服务/testgen 构建、`testloop-mcp 0.5.10` 版本输出、help 输出、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。

## 发布备注

- v0.5.10 发布文案应突出：CI artifact 现在直接携带 Agent 回复草稿，first-run 和 onboarding 两条 bootstrap 的失败消费路径已统一。
- 不要把它包装成测试生成能力升级；它是接入体验、失败分流和客户端回归能力升级。
