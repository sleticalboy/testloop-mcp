# v0.5.17 发布检查清单

## 当前目标

这是 v0.5.17 的发布检查清单。目标是把 v0.5.16 之后围绕 Agent 决策客户端 CI installer、接入 Checklist、安装 dry-run 摘要 schema/sample/validator 的改动整理成一个可发布边界，并为正式发布、资产校验和 Homebrew 分发更新做准备。

发布重点见 [v0.5.17 发布说明](./plan-release-notes-v0.5.17.md)。

当前发布状态：正式版本准备中。`main.go` implementation version、`CHANGELOG.md` 正式版本段和当前安装/接入文档版本引用已同步到 `0.5.17` / `v0.5.17`，本地 release readiness 已通过；版本准备提交、tag、Release assets、Homebrew 和 Post-Release Verify 待执行。

## 当前差异核对

- [x] 新增 `scripts/install-agent-decision-client-ci-template.sh`，可向外部 MCP 客户端仓库写入 Agent 决策契约 GitHub Actions workflow。
- [x] installer 支持 `--version`、`--dry-run` 和 `--force`，并可脱离仓库单文件运行。
- [x] 新增 `scripts/showcase-agent-decision-client-ci-template-install.sh`，覆盖下载安装脚本、生成 workflow、模拟 helper checkout 和执行 contract。
- [x] 新增安装 dry-run JSON summary schema 和通过态 sample。
- [x] 新增 `scripts/validate-agent-decision-client-ci-install-summary.mjs`，提供无依赖安装摘要校验入口。
- [x] 新增 [Agent 决策客户端 CI 接入 Checklist](./agent-decision-client-ci-checklist.md)，把外部客户端接入步骤压成一页式执行清单。
- [x] Checklist 命令回归测试会实际执行 Markdown 中的安装、contract 和安装 dry-run 命令。
- [x] README、客户端集成说明、MCP 客户端契约测试说明、Agent 决策客户端 CI 模板、CHANGELOG 和 roadmap 已同步。

## 候选内容

- [x] 外部客户端可以用 installer 一键生成 `.github/workflows/testloop-agent-decision-contract.yml`。
- [x] 接入方可以按一页式 checklist 完成 helper ref、CI 命令、artifact 和失败分流配置。
- [x] 维护者可以用完整安装 dry-run 验证“installer 写出的 workflow 能实际跑 Agent 决策 contract”。
- [x] 客户端可以用 schema、passed sample 或无依赖 validator 固定安装 dry-run JSON 输出。

## 已验证

- [x] `sh test/install_agent_decision_client_ci_template_test.sh`
- [x] `sh test/agent_decision_client_ci_template_install_showcase_test.sh`
- [x] `sh test/agent_decision_client_ci_template_install_summary_schema_test.sh`
- [x] `sh test/agent_decision_client_ci_install_summary_validator_test.sh`
- [x] `sh test/agent_decision_client_ci_checklist_doc_test.sh`
- [x] `sh test/agent_decision_client_ci_checklist_commands_test.sh`
- [x] `sh test/agent_decision_client_ci_template_doc_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/mcp_client_contract_doc_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/ci_workflow_test.sh`
- [x] `for t in test/*_test.sh; do sh "$t"; done`
- [x] `go test ./...`
- [x] `git diff --check`
- [x] `d02c3b3` 远端 CI run `29804100097` passed，覆盖 Agent 决策客户端 CI 接入 checklist。
- [x] `e361a89` 远端 CI run `29804258910` passed，覆盖安装脚本默认版本同步。
- [x] `31a0a13` 远端 CI run `29806137273` passed，覆盖 checklist 命令回归。
- [x] `9f5c970` 远端 CI run `29807044703` passed，覆盖安装摘要 passed sample。
- [x] `25d0278` 远端 CI run `29807556910` passed，覆盖安装摘要 validator。
- [x] `f351e7c` 远端 CI run `29808013697` passed，覆盖 v0.5.17 候选发布边界文档。
- [x] 正式版本准备后的完整本地门禁已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.17-release-prep-dist scripts/verify-release-candidate.sh v0.5.17` 输出 `release_candidate_status=passed`，候选二进制 `--version` 输出 `testloop-mcp 0.5.17`。

## 发布前门禁

- [x] 候选边界整理提交后的 main CI 已通过：`f351e7c` run `29808013697` passed。
- [x] `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.17-release-prep-dist scripts/verify-release-candidate.sh v0.5.17`
- [x] `git diff --check`
- [x] `main.go` implementation version 更新到 `0.5.17`。

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.5.17`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.17 - 2026-07-21`。
- [x] 同步 README、installation、quickstart 和必要版本引用到 `0.5.17` / `v0.5.17`。
- [x] 测试中的版本期望同步到 `0.5.17`。
- [x] 重新运行完整 release readiness。
- [ ] 提交版本准备改动后确认远端 CI passed。
- [ ] 打 tag `v0.5.17` 并推送。
- [ ] 等 Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.17` 验证 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.17 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.17` 更新仓库内 Formula。
- [ ] 更新 Homebrew tap 到 `0.5.17` 并推送。
- [ ] 手动触发 Post-Release Verify。

## 当前结论

v0.5.17 正式版本准备文件和本地门禁已经完成，但尚未提交版本准备、打 tag、生成 Release assets 或更新 Homebrew。这个版本不是扩语言或提升测试生成算法，而是把外部客户端 CI 接入路径从“复制模板”推进到“installer + checklist + 安装 dry-run + JSON schema/sample/validator + 命令回归”。下一步应提交版本准备并等待 main CI；通过后再进入正式发布动作。
