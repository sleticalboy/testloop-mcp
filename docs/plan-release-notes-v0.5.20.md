# v0.5.20 发布说明

## 标题

testloop-mcp v0.5.20

## 发布状态

- [x] 创建 v0.5.20 候选发布说明草案。
- [x] 梳理 v0.5.19 之后围绕 release response 独立客户端消费、真实外部仓库安装、安装 summary 契约、release readiness 门禁和接入 checklist 的改动边界。
- [x] 最新已完成的远端 CI：`a704d4e` run `29844159254` passed，覆盖 v0.5.20 候选发布边界整理。
- [x] 候选 release readiness 已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.19-goal-readiness-dist scripts/verify-release-candidate.sh v0.5.19` 输出 `release_candidate_status=passed`。
- [x] 已进入正式版本准备：`main.go` implementation version 已更新为 `0.5.20`，`CHANGELOG.md` 已收敛到 `v0.5.20 - 2026-07-21` 并保留空 Unreleased。
- [x] 正式版本准备后的 release readiness 已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.20-release-prep-dist scripts/verify-release-candidate.sh v0.5.20` 输出 `release_candidate_status=passed`。
- [ ] 尚未打 `v0.5.20` tag。
- [ ] 尚未创建 GitHub Release、正式 assets 或 Homebrew tap 更新。

## 摘要

v0.5.20 候选边界继续围绕项目定位推进：面向 AI 编程代理的测试反馈闭环 MCP 服务。

这个候选版本不扩语言、不承诺测试生成算法质量提升。重点是把发布后 release smoke 的消费链路从“仓库内 renderer 和 fixture”推进到“外部客户端可以安装、校验、运行 CI 并交给 Agent 分流”的完整路径。

接入方现在可以把 release response 客户端包和 `.github/workflows/testloop-release-response-contract.yml` 写入真实仓库，生成稳定安装 summary，再用无依赖 validator 固定 `status=written`、`release_ref`、`fixture_count`、`agent_next_step`、`npm_exit_code` 和决策序列。

## 主要变化

### release response 独立客户端消费

- 新增 `scripts/render-agent-decision-client-release-response.mjs`。
- 新增 `scripts/showcase-agent-decision-client-release-response-smoke.sh --json`。
- 新增 release response schema、通过态 fixture 和失败态 fixture。
- 失败分流覆盖 `inspect-release-installer`、`inspect-release-client-response`、`inspect-release-consumer-response`、`inspect-agent-decision-fixtures` 和 `inspect-release-smoke-summary`。

### 最小包导出与外部 CI 形态

- 新增 `scripts/export-agent-decision-release-response-client.mjs`。
- 导出包包含 renderer、断言脚本、`package.json`、response schema 和通过/失败态 fixture。
- 导出目录可直接运行 `npm test --silent`。
- 新增 `scripts/showcase-agent-decision-client-release-response-ci.sh --json`，模拟外部仓库 GitHub Actions 形态。

### 真实仓库安装

- 新增 `scripts/install-agent-decision-release-response-client.sh`。
- 安装脚本会写入 `testloop-release-response-client/` 和 `.github/workflows/testloop-release-response-contract.yml`。
- 安装后会在目标包目录运行 `npm test --silent`。
- 默认不覆盖已有 workflow 或包目录；需要覆盖时显式传 `--force`。
- 支持 `--summary-json`、`--package-dir`、`--workflow-path`、`--dry-run` 和 `--json`。

### 安装 summary 契约

- 新增 `docs/fixtures/agent-decision-release-response-client-install-summary.schema.json`。
- 新增通过态样例 `docs/fixtures/agent-decision-release-response-client-install-summary/passed.json`。
- 新增 `scripts/validate-agent-decision-release-response-client-install-summary.mjs`。
- validator 固定 `status=written`、`release_ref=v0.5.20`、`fixture_count=8`、`agent_next_step=ready`、`npm_exit_code=0` 和决策序列。

### readiness 与接入文档

- `scripts/verify-release-candidate.sh` 现在会同时验证 release response 导出包和真实仓库安装 summary。
- 新增 [Agent 决策 release response 接入 Checklist](./agent-decision-release-response-checklist.md)。
- 新增 checklist 命令回归测试，按文档顺序真实执行安装、summary validator 和导出包 `npm test --silent`。
- `docs/real-integration-cases.md` 新增 release response 真实安装接入记录，展示外部仓库会得到的 workflow、导出包、summary 和 Agent response。

## 质量边界

- 当前候选聚焦 release response 消费合同，不改变 `generate_tests` 的生成策略。
- `agent_next_step=ready` 只表示 release response contract 通过，不代表用户项目业务测试已经通过。
- release smoke summary 仍依赖发布后 raw installer 路径；网络抖动应通过 retry 或本地 file URL 演练隔离。
- 客户端应把 `testloop-release-response.json` 和安装 summary JSON 作为分流入口，不解析日志文本。

## 推荐验证

- `sh test/agent_decision_client_release_response_smoke_test.sh`
- `sh test/agent_decision_client_release_response_test.sh`
- `sh test/agent_decision_client_release_response_fixtures_test.sh`
- `sh test/agent_decision_client_release_response_ci_test.sh`
- `sh test/agent_decision_release_response_client_export_test.sh`
- `sh test/install_agent_decision_release_response_client_test.sh`
- `sh test/agent_decision_release_response_client_install_summary_schema_test.sh`
- `sh test/agent_decision_release_response_client_install_summary_validator_test.sh`
- `sh test/agent_decision_release_response_checklist_doc_test.sh`
- `sh test/agent_decision_release_response_checklist_commands_test.sh`
- `sh test/real_integration_cases_doc_test.sh`
- `sh test/release_candidate_script_test.sh`
- `sh test/release_doc_index_test.sh`
- `sh test/docs_links_test.sh`
- `sh test/docs_json_examples_test.sh`
- `sh test/ci_workflow_test.sh`
- `for t in test/*_test.sh; do sh "$t"; done`
- `go test ./...`
- `git diff --check`
- `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.20-release-prep-dist scripts/verify-release-candidate.sh v0.5.20`
  - 输出：`release_candidate_status=passed`
  - 输出：`agent_decision_release_response_client_install_summary_status=passed release_ref=v0.5.20`
  - 输出：`testloop-mcp 0.5.20`
  - 输出：`testloop-mcp_v0.5.20_darwin_arm64.tar.gz: OK`

## 发布备注

- 对外文案应强调：v0.5.20 候选让外部客户端可以把 release response contract 安装到真实仓库，并得到机器可校验的安装 summary 和 Agent response。
- 推荐演示路径：运行 `scripts/install-agent-decision-release-response-client.sh --json /path/to/client-repo`，再运行 `node scripts/validate-agent-decision-release-response-client-install-summary.mjs /path/to/install-summary.json`。
- 发布前仍需完成本轮正式版本准备验证：复跑 release readiness、等待 main CI，再打 tag。
