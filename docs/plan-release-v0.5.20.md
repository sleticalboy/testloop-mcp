# v0.5.20 发布检查清单

## 当前目标

这是 v0.5.20 的候选发布检查清单。目标是把 v0.5.19 之后围绕 release response 独立客户端消费、真实外部仓库安装、安装 summary 契约、release readiness 门禁和接入 checklist 的改动整理成一个可发布边界。

发布重点见 [v0.5.20 发布说明](./plan-release-notes-v0.5.20.md)。

当前发布状态：已进入正式版本准备。`main.go` implementation version 已更新为 `0.5.20`，`CHANGELOG.md` 已收敛到 `v0.5.20 - 2026-07-21` 并保留空 Unreleased；尚未打 `v0.5.20` tag，尚未创建 GitHub Release，尚未更新 Homebrew tap。

## 当前差异核对

- [x] 新增 release smoke summary 到 release response 的独立客户端消费链路。
- [x] 新增 release response renderer、schema、通过态 fixture 和失败态 fixture。
- [x] 新增 release response 客户端最小包导出脚本。
- [x] 新增 release response 外部客户端 CI 形态 showcase。
- [x] 新增 release response 客户端真实仓库安装脚本。
- [x] 新增 release response 客户端安装 summary schema、通过态 fixture 和 validator。
- [x] release readiness 已覆盖 release response 导出包和真实仓库安装 summary。
- [x] 新增 release response 接入 checklist，并用命令回归测试固定关键命令。
- [x] 真实接入案例已记录 release response 真实安装链路的 workflow、导出包、summary 和 Agent response。
- [x] README、客户端集成说明、fixtures 索引、real integration cases、CHANGELOG 和 roadmap 已同步。

## 候选内容

- [x] 接入方可以运行 `scripts/showcase-agent-decision-client-release-response-smoke.sh --json`，用独立临时 Node 客户端消费 release smoke summary。
- [x] 接入方可以运行 `node scripts/export-agent-decision-release-response-client.mjs /tmp/testloop-release-response-client` 导出最小客户端包。
- [x] 接入方可以运行 `scripts/showcase-agent-decision-client-release-response-ci.sh --json` 验证外部仓库 GitHub Actions 形态。
- [x] 接入方可以运行 `scripts/install-agent-decision-release-response-client.sh --json /path/to/client-repo` 把包和 workflow 写入真实仓库。
- [x] 接入方可以运行 `node scripts/validate-agent-decision-release-response-client-install-summary.mjs /path/to/install-summary.json` 机器校验安装结果。
- [x] `docs/agent-decision-release-response-checklist.md` 已把 release smoke summary、安装、summary validator、本地 npm 复验、CI artifact 和 Agent 分流整理成步骤。
- [x] 当前版本边界明确：不扩语言、不改测试生成算法，聚焦外部客户端/Agent 的 release response 消费合同。

## 已验证

- [x] `sh test/agent_decision_client_release_response_smoke_test.sh`
- [x] `sh test/agent_decision_client_release_response_test.sh`
- [x] `sh test/agent_decision_client_release_response_fixtures_test.sh`
- [x] `sh test/agent_decision_client_release_response_ci_test.sh`
- [x] `sh test/agent_decision_release_response_client_export_test.sh`
- [x] `sh test/install_agent_decision_release_response_client_test.sh`
- [x] `sh test/agent_decision_release_response_client_install_summary_schema_test.sh`
- [x] `sh test/agent_decision_release_response_client_install_summary_validator_test.sh`
- [x] `sh test/agent_decision_release_response_checklist_doc_test.sh`
- [x] `sh test/agent_decision_release_response_checklist_commands_test.sh`
- [x] `sh test/real_integration_cases_doc_test.sh`
- [x] `sh test/release_candidate_script_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/docs_json_examples_test.sh`
- [x] `sh test/ci_workflow_test.sh`
- [x] `for t in test/*_test.sh; do sh "$t"; done`
- [x] `go test ./...`
- [x] `git diff --check`
- [x] `461a3ce` 远端 CI run `29843072721` passed，覆盖 release response 真实安装案例记录。
- [x] `a704d4e` 远端 CI run `29844159254` passed，覆盖 v0.5.20 候选发布边界整理。
- [x] `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.19-goal-readiness-dist scripts/verify-release-candidate.sh v0.5.19` 输出 `release_candidate_status=passed`。
- [x] `go test ./...`
- [x] `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.20-release-prep-dist scripts/verify-release-candidate.sh v0.5.20` 输出 `release_candidate_status=passed`。
- [x] `353d255` 远端 CI run `29846178265` passed，覆盖 v0.5.20 正式版本准备。

## 发布前门禁

- [x] 最新 main CI passed：`353d255` run `29846178265` passed。
- [x] 本地 release readiness passed：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.19-goal-readiness-dist scripts/verify-release-candidate.sh v0.5.19`。
- [x] readiness 输出包含 release response 导出包验证：`response_fixture_count=5`。
- [x] readiness 输出包含真实仓库安装 summary 验证：`agent_decision_release_response_client_install_summary_status=passed release_ref=v0.5.19`。
- [x] readiness 输出包含候选二进制版本：`testloop-mcp 0.5.19`。
- [x] readiness 输出包含 darwin arm64 打包 dry-run 和 sha256 校验：`testloop-mcp_v0.5.19_darwin_arm64.tar.gz: OK`。
- [x] 正式版本准备后的 release readiness passed：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.20-release-prep-dist scripts/verify-release-candidate.sh v0.5.20`。
- [x] 正式版本准备后的 readiness 输出包含真实仓库安装 summary 验证：`agent_decision_release_response_client_install_summary_status=passed release_ref=v0.5.20`。
- [x] 正式版本准备后的 readiness 输出包含候选二进制版本：`testloop-mcp 0.5.20`。
- [x] 正式版本准备后的 readiness 输出包含 darwin arm64 打包 dry-run 和 sha256 校验：`testloop-mcp_v0.5.20_darwin_arm64.tar.gz: OK`。

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.5.20`。
- [x] 将 `CHANGELOG.md` 的 Unreleased 内容收敛到 `v0.5.20 - 2026-07-21`，并保留新的空 Unreleased。
- [x] 同步 README、installation、quickstart 和必要版本引用到 `0.5.20` / `v0.5.20`。
- [x] 测试中的版本期望同步到 `0.5.20`。
- [x] 重新运行完整 release readiness。
- [x] 提交版本准备改动后确认远端 CI passed。
- [ ] 打 tag `v0.5.20` 并推送。
- [ ] 等 Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.20` 验证 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.20 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.20` 更新仓库内 Formula。
- [ ] 更新 Homebrew tap 到 `0.5.20`。
- [ ] Post-Release Verify。
- [ ] 发布后运行 release response checklist 核心 smoke。

## 当前结论

v0.5.20 已完成正式版本准备的本地门禁和远端 CI：v0.5.19 之后的 release response 消费链路已经从 renderer/fixture 推进到最小包导出、外部仓库 CI 形态、真实仓库安装、安装 summary 契约、release readiness 门禁和 checklist 命令回归。当前仍不打 tag；下一步应进入正式 tag 与 Release assets 阶段。
