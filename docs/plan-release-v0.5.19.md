# v0.5.19 发布检查清单

## 当前目标

这是 v0.5.19 的候选发布检查清单。目标是把 v0.5.18 之后围绕发布流程加固、外部客户端 Agent response artifact、消费端 smoke 失败态 fixture 和文档同步的改动整理成一个可发布边界。

发布重点见 [v0.5.19 发布说明](./plan-release-notes-v0.5.19.md)。

当前发布状态：已正式发布。`v0.5.19` tag 已推送，Release Artifacts run `29827625494` 已通过，五个平台 10 个正式资产已上传并通过资产清单校验，GitHub Release 正文已更新，仓库内 Formula 与 `sleticalboy/homebrew-tap` 已更新到 `0.5.19`，Post-Release Verify run `29828306451` 已通过。

## 当前差异核对

- [x] Release Artifacts workflow 已拆出 `ensure-release` 前置 job，避免矩阵 job 并发创建重复空 Release。
- [x] Release Artifacts workflow 已按 tag 增加 `concurrency`。
- [x] 新增 `test/release_workflow_test.sh`，固定 Release 创建结构。
- [x] 新增 `scripts/render-agent-decision-client-consumer-response.mjs`，把 consumer smoke summary 转成 `agent_next_step`。
- [x] 新增消费端 smoke summary 失败态 fixture：`validator-failed.json` 和 `fixture-drift.json`。
- [x] `scripts/showcase-agent-decision-client-consumer-smoke.sh --json` 已返回 `agent_response_json`。
- [x] 新增 `scripts/render-agent-decision-client-ci-response.mjs`，把基础客户端 CI summary 转成 Agent response。
- [x] `scripts/install-agent-decision-client-ci-template.sh` 生成的 workflow 已新增 `Render Agent decision response` step，并上传 `/tmp/testloop-agent-decision-client-response.json`。
- [x] README、客户端集成说明、Agent 决策客户端 CI Checklist、Agent 决策客户端 CI 模板、fixtures 索引、CHANGELOG 和 roadmap 已同步。

## 候选内容

- [x] 客户端可以从基础 contract CI artifact 直接读取 `testloop-agent-decision-client-response.json`。
- [x] 客户端可以从 consumer smoke summary 直接读取 `agent_response_json`。
- [x] 失败态样例已经固定 validator 失败和 fixture 决策漂移两类分流。
- [x] 发布流程已经避免重复空 Release。
- [x] 当前版本边界明确：不扩语言、不改测试生成算法，只强化 Agent/客户端消费合同和发版稳定性。

## 已验证

- [x] `sh test/release_workflow_test.sh`
- [x] `sh test/agent_decision_client_ci_response_test.sh`
- [x] `sh test/agent_decision_client_consumer_response_test.sh`
- [x] `sh test/agent_decision_client_ci_consumer_smoke_test.sh`
- [x] `sh test/agent_decision_client_ci_consumer_smoke_summary_schema_test.sh`
- [x] `sh test/agent_decision_client_ci_consumer_smoke_summary_validator_test.sh`
- [x] `sh test/install_agent_decision_client_ci_template_test.sh`
- [x] `sh test/agent_decision_client_ci_template_doc_test.sh`
- [x] `sh test/agent_decision_client_ci_template_yaml_test.sh`
- [x] `sh test/agent_decision_client_ci_template_dry_run_test.sh`
- [x] `sh test/agent_decision_client_ci_checklist_doc_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/readme_ci_snippet_test.sh`
- [x] `sh test/showcase_scripts_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/docs_json_examples_test.sh`
- [x] `sh test/ci_workflow_test.sh`
- [x] `for t in test/*_test.sh; do sh "$t"; done`
- [x] `go test ./...`
- [x] `git diff --check`
- [x] `99430dd` 远端 CI run `29819978961` passed，覆盖 Release Artifacts 并发创建加固。
- [x] `48b48f2` 远端 CI run `29820554605` passed，覆盖 consumer smoke Agent 分流。
- [x] `878d352` 远端 CI run `29821061038` passed，覆盖消费端失败态分流 fixture。
- [x] `113e0af` 远端 CI run `29821447588` passed，覆盖客户端模板失败分流示例。
- [x] `d0d81d1` 远端 CI run `29821925051` passed，覆盖 consumer smoke `agent_response_json`。
- [x] `b78a375` 远端 CI run `29826190450` passed，覆盖基础客户端 CI response artifact。
- [x] `0f8d971` 远端 CI run `29826825652` passed，覆盖 v0.5.19 候选边界整理。
- [x] `d026283` 远端 CI run `29827369739` passed，覆盖 v0.5.19 正式版本准备。

## 发布前门禁

- [x] 候选边界整理提交后的 main CI passed：`0f8d971` run `29826825652` passed。
- [x] `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.19-release-prep-dist scripts/verify-release-candidate.sh v0.5.19`
- [x] `git diff --check`
- [x] `main.go` implementation version 更新到 `0.5.19`。
- [x] `CHANGELOG.md` 的 Unreleased 内容收敛到 `v0.5.19 - 2026-07-21`。

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.5.19`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.19 - 2026-07-21`，并保留新的空 Unreleased。
- [x] 同步 README、installation、quickstart 和必要版本引用到 `0.5.19` / `v0.5.19`。
- [x] 测试中的版本期望同步到 `0.5.19`。
- [x] 重新运行完整 release readiness。
- [x] 提交版本准备改动后确认远端 CI passed：`d026283` run `29827369739` passed。
- [x] 打 tag `v0.5.19` 并推送。
- [x] 等 Release Artifacts workflow 生成五平台资产和 `.sha256`：run `29827625494` passed。
- [x] 使用 `scripts/verify-release-assets.sh v0.5.19` 验证 Release 资产完整。
- [x] 更新 GitHub Release 正文为正式 v0.5.19 发布说明。
- [x] 使用 `scripts/generate-homebrew-formula.sh v0.5.19` 更新仓库内 Formula。
- [x] 更新 Homebrew tap 到 `0.5.19` 并推送：tap commit `72123db`。
- [x] Post-Release Verify：run `29828306451` passed。
- [x] 发布后运行 raw installer smoke、基础客户端 CI response smoke 和 consumer smoke。

## 当前结论

v0.5.19 已完成正式 tag、Release assets、资产清单校验、GitHub Release 正文、仓库内 Formula、Homebrew tap、Post-Release Verify 和发布后 smoke。这个版本把 v0.5.18 之后的客户端接入链路从“可校验 summary/result JSON”推进到“可直接产出 Agent 下一步动作 artifact”，同时修复了正式发布时暴露的 Release 创建并发问题。
