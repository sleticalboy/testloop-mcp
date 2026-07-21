# v0.5.16 发布检查清单

## 当前目标

这是 v0.5.16 的发布检查清单。目标是把 v0.5.15 之后围绕 Agent 决策 fixture 外部客户端 CI 接入的改动整理成一个可发布边界，并完成正式发布、资产校验和 Homebrew 分发更新。

发布重点见 [v0.5.16 发布说明](./plan-release-notes-v0.5.16.md)。

当前发布状态：已正式发布。`v0.5.16` tag 已推送，GitHub Release 已创建，五个平台 Release assets 和 `.sha256` 已上传并校验，仓库内 Formula 与 `sleticalboy/homebrew-tap` 已更新到 `0.5.16`。

## 当前差异核对

- [x] 新增 `scripts/showcase-agent-decision-client-ci.sh`，模拟外部客户端 CI：导出 Agent 决策 fixture 包并运行导出包 `npm test --silent`。
- [x] showcase 支持默认文本摘要和 `--json` 机器输出。
- [x] 新增 [Agent 决策客户端 CI 模板](./agent-decision-client-ci-template.md)，提供 `.github/workflows/testloop-agent-decision-contract.yml` 可复制 workflow。
- [x] 新增 `test/agent_decision_client_ci_template_dry_run_test.sh`，在临时外部客户端目录中模拟 `.testloop-mcp` helper checkout。
- [x] 修复 `scripts/export-agent-decision-fixtures.mjs` 仓库根目录定位，外部客户端从 `.testloop-mcp/scripts/...` 调用 helper 时不再按客户端 cwd 查找 fixture。
- [x] README、客户端集成说明、MCP 客户端契约测试说明、CHANGELOG 和 roadmap 已同步。

## 候选内容

- [x] 外部客户端可以一条命令完成“导出 fixture 包 -> 运行 validator -> 获取 summary JSON”。
- [x] 客户端仓库可以复制 GitHub Actions 模板，把 `status/action -> decision` 合同纳入自己的 CI。
- [x] 模板不只做 YAML 解析，也有本地 dry-run 覆盖 `.testloop-mcp` helper checkout 的真实相对路径。
- [x] CI / Agent 可以直接读取 `status`、`fixture_count`、`decisions[]`、`failures[]` 和 `validator_exit_code`，不用解析 key/value 日志。

## 已验证

- [x] `sh test/agent_decision_client_ci_showcase_test.sh`
- [x] `sh test/agent_decision_client_ci_template_doc_test.sh`
- [x] `sh test/agent_decision_client_ci_template_yaml_test.sh`
- [x] `sh test/agent_decision_client_ci_template_dry_run_test.sh`
- [x] `sh test/agent_decision_fixture_export_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/mcp_client_contract_doc_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/ci_workflow_test.sh`
- [x] `for t in test/*_test.sh; do sh "$t"; done`
- [x] `go test ./...`
- [x] `git diff --check`
- [x] `9a5f63e` 远端 CI run `29795870585` passed，覆盖 Agent 决策客户端 CI showcase。
- [x] `9d4e0cb` 远端 CI run `29796127281` passed，覆盖 Agent 决策客户端 CI 模板。
- [x] `76a1be0` 远端 CI run `29796336606` passed，覆盖 showcase JSON 输出。
- [x] `ebfe245` 远端 CI run `29796478139` passed，覆盖 CHANGELOG 收敛。
- [x] `08ff2a4` 远端 CI run `29797835817` passed，覆盖外部客户端模板 dry-run 和导出脚本定位修复。
- [x] `63409a6` 远端 CI run `29798075470` passed，覆盖 v0.5.16 候选边界文档。
- [x] 正式版本准备后的完整本地门禁已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.16-release-prep-dist scripts/verify-release-candidate.sh v0.5.16` 输出 `release_candidate_status=passed`，候选二进制 `--version` 输出 `testloop-mcp 0.5.16`。
- [x] `64995fc` 远端 CI run `29801283144` passed，覆盖 v0.5.16 正式版本准备。
- [x] `v0.5.16` Release Artifacts run `29801398746` passed，五个平台 10 个资产已上传。
- [x] `TESTLOOP_MCP_REPO=sleticalboy/testloop-mcp scripts/verify-release-assets.sh v0.5.16` 已验证正式 Release 资产完整。
- [x] Homebrew tap 已更新到 `0.5.16` 并推送：tap commit `1de9ae4`。
- [x] Post-Release Verify run `29801687152` passed，覆盖资产清单和五个平台安装验证。

## 发布前门禁

- [x] 候选边界整理提交后的 main CI 已通过：`63409a6` run `29798075470` passed。
- [x] `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.16-release-prep-dist scripts/verify-release-candidate.sh v0.5.16`
- [x] `git diff --check`
- [x] `main.go` implementation version 已更新到 `0.5.16`。

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.5.16`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.16 - 2026-07-21`。
- [x] 同步 README、installation、quickstart 和必要版本引用到 `0.5.16` / `v0.5.16`。
- [x] 测试中的版本期望同步到 `0.5.16`。
- [x] 重新运行完整 release readiness。
- [x] 提交版本准备改动后确认远端 CI passed：`64995fc` run `29801283144` passed。
- [x] 打 tag `v0.5.16` 并推送。
- [x] 等 Release Artifacts workflow 生成五平台资产和 `.sha256`：run `29801398746` passed。
- [x] 使用 `scripts/verify-release-assets.sh v0.5.16` 验证 Release 资产完整。
- [x] 更新 GitHub Release 正文为正式 v0.5.16 发布说明。
- [x] 使用 `scripts/generate-homebrew-formula.sh v0.5.16` 更新仓库内 Formula。
- [x] 更新 Homebrew tap 到 `0.5.16` 并推送：tap commit `1de9ae4`。
- [x] 手动触发 Post-Release Verify：run `29801687152` passed。

## 当前结论

v0.5.16 已完成正式 GitHub Release、五平台资产发布、资产清单校验、GitHub Release 正文、仓库内 Formula、Homebrew tap 和 Post-Release Verify。这个版本不是扩语言或提升测试生成算法，而是把 v0.5.15 的 Agent 决策 fixture 导出包继续推进到外部客户端可复制、可 dry-run、可 JSON 断言的 CI 接入路径。
