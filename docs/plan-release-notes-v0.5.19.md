# v0.5.19 发布说明草案

## 标题

testloop-mcp v0.5.19

## 发布状态

- [x] 创建 v0.5.19 候选发布说明草案。
- [x] 梳理 v0.5.18 之后围绕 Release Artifacts 并发加固、消费端 smoke Agent 分流、失败态 fixture、`agent_response_json` 和基础客户端 CI response artifact 的改动边界。
- [x] 最新已完成的远端 CI：`d026283` run `29827369739` passed，覆盖 v0.5.19 正式版本准备。
- [x] 正式版本准备已经开始：`main.go` implementation version、`CHANGELOG.md` 和当前安装/接入文档版本引用已同步到 `0.5.19` / `v0.5.19`。
- [x] 正式版本准备后的 release readiness 已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.19-release-prep-dist scripts/verify-release-candidate.sh v0.5.19` 输出 `release_candidate_status=passed`，`testloop-mcp --version` 输出 `testloop-mcp 0.5.19`。
- [ ] 尚未打 `v0.5.19` tag，尚未生成 Release assets，尚未更新 Homebrew Formula / tap。

## 摘要

v0.5.19 候选边界继续围绕项目定位推进：面向 AI 编程代理的测试反馈闭环 MCP 服务。

这个候选版本不扩语言、不承诺测试生成算法质量提升。重点是把 v0.5.18 已经建立的外部客户端消费端 smoke 继续补成可直接交给 Agent 的 artifact：接入方不只知道 contract 是否通过，还能拿到结构化 `agent_next_step`，并在失败时区分校验器问题、fixture/决策语义漂移和 summary 缺失。

同时，v0.5.18 发布时暴露的 Release Artifacts 并发创建重复空 Release 问题已经在 workflow 结构上修复，避免后续 tag 发布重复出现同类清理工作。

## 主要变化

### 发布流程加固

- Release Artifacts workflow 新增 `ensure-release` 前置 job，统一解析 tag 并只创建一次 GitHub Release。
- 矩阵 build job 改为依赖 `ensure-release`，只负责构建、校验和上传资产。
- Release Artifacts workflow 按 tag 增加 `concurrency`，避免同一 tag 的 tag push 和手动 dispatch 并发运行。
- 新增 release workflow 结构测试，固定 Release 创建只出现一次且不在矩阵 build job 内。

### 消费端 smoke Agent 分流

- 新增 `scripts/render-agent-decision-client-consumer-response.mjs`。
- 该脚本可把消费端 smoke summary 转成稳定 `agent_next_step`。
- 通过态输出 `ready`。
- validator 失败输出 `inspect-consumer-smoke-validator`。
- fixture 数量或决策序列漂移输出 `inspect-agent-decision-fixtures`。
- 其他结构问题输出 `inspect-consumer-smoke-summary`。

### 失败态 fixture

- 新增 `docs/fixtures/agent-decision-client-consumer-smoke-summary/validator-failed.json`。
- 新增 `docs/fixtures/agent-decision-client-consumer-smoke-summary/fixture-drift.json`。
- response renderer、schema 和 validator 测试均已覆盖通过态和失败态样例。

### Agent response artifact

- `scripts/showcase-agent-decision-client-consumer-smoke.sh --json` 现在会返回 `agent_response_json`，指向 renderer 生成的 Agent 下一步动作 JSON。
- 新增 `scripts/render-agent-decision-client-ci-response.mjs`，可把基础客户端 CI summary 转成 Agent response。
- 安装脚本生成的 `.github/workflows/testloop-agent-decision-contract.yml` 已新增 `Render Agent decision response` step。
- 客户端 CI 模板现在默认上传 `/tmp/testloop-agent-decision-client-response.json`。
- 模板 contract 命令显式启用 `set -euo pipefail`，避免 pipeline 隐式吞掉失败。

### 文档同步

- README、客户端集成说明、Agent 决策客户端 CI Checklist、Agent 决策客户端 CI 模板和 fixtures 索引已同步 response artifact、失败态 fixture 和分流命令。
- 默认 CI 已加入新增 response renderer 测试。

## 质量边界

- 当前版本候选仍聚焦外部 MCP 客户端和 AI Agent 消费合同，不改变 `generate_tests` 的生成策略。
- `agent_next_step=ready` 只表示 fixture contract 或 consumer smoke contract 通过，不代表用户项目业务测试已经通过。
- `manual_review_*` 仍不进入自动修复循环；只有 `failed/apply_fix_suggestions` 对应 repair task。
- 客户端应把 response JSON 作为分流入口，再按需下钻 summary、result JSON、fixture manifest 和 failures。

## 推荐验证

- `sh test/release_workflow_test.sh`
- `sh test/agent_decision_client_ci_response_test.sh`
- `sh test/agent_decision_client_consumer_response_test.sh`
- `sh test/agent_decision_client_ci_consumer_smoke_test.sh`
- `sh test/agent_decision_client_ci_consumer_smoke_summary_schema_test.sh`
- `sh test/agent_decision_client_ci_consumer_smoke_summary_validator_test.sh`
- `sh test/install_agent_decision_client_ci_template_test.sh`
- `sh test/agent_decision_client_ci_template_doc_test.sh`
- `sh test/agent_decision_client_ci_template_yaml_test.sh`
- `sh test/agent_decision_client_ci_template_dry_run_test.sh`
- `sh test/agent_decision_client_ci_checklist_doc_test.sh`
- `sh test/client_integration_doc_test.sh`
- `sh test/readme_ci_snippet_test.sh`
- `sh test/showcase_scripts_test.sh`
- `sh test/release_doc_index_test.sh`
- `sh test/docs_links_test.sh`
- `sh test/docs_json_examples_test.sh`
- `sh test/ci_workflow_test.sh`
- `for t in test/*_test.sh; do sh "$t"; done`
- `go test ./...`
- `git diff --check`

## 发布备注

- 对外文案应强调：v0.5.19 候选让外部 MCP 客户端 CI 直接产出 Agent 可消费的下一步动作 artifact。
- 推荐演示路径：运行 `scripts/showcase-agent-decision-client-consumer-smoke.sh --json`，读取 summary 中的 `agent_response_json`，确认 `agent_next_step=ready`。
- 基础客户端模板演示路径：运行 installer 生成 workflow，确认 artifact 清单包含 `testloop-agent-decision-client-response.json`。
- 正式发布前仍需完成 tag、Release assets、GitHub Release 正文和 Homebrew tap 更新。
