# v0.5.16 发布说明

## 标题

testloop-mcp v0.5.16

## 发布状态

- [x] 创建 v0.5.16 候选发布说明草案。
- [x] 梳理 v0.5.15 之后围绕 Agent 决策 fixture 外部客户端 CI 接入的改动边界。
- [x] 最新已完成的远端 CI：`08ff2a4` run `29797835817` passed，覆盖外部客户端模板 dry-run 和导出脚本定位修复。
- [x] 候选边界整理提交 `63409a6` 的远端 CI run `29798075470` passed。
- [x] 正式版本准备文件已更新：implementation version、`CHANGELOG.md` 正式版本段和当前安装/接入文档版本引用已同步到 `0.5.16` / `v0.5.16`。
- [x] 正式版本准备后的完整本地门禁已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.16-release-prep-dist scripts/verify-release-candidate.sh v0.5.16` 输出 `release_candidate_status=passed`，`testloop-mcp --version` 输出 `testloop-mcp 0.5.16`。
- [x] 版本准备提交 `64995fc` 的远端 CI run `29801283144` passed。
- [x] `v0.5.16` tag 已推送，Release Artifacts run `29801398746` passed，五个平台 10 个资产已上传。
- [x] `TESTLOOP_MCP_REPO=sleticalboy/testloop-mcp scripts/verify-release-assets.sh v0.5.16` 已验证正式 Release 资产完整。
- [x] GitHub Release 正文已更新为正式 v0.5.16 发布说明。
- [x] 仓库内 Homebrew Formula 已用正式 Release asset digest 更新到 `0.5.16`。
- [x] Homebrew tap 已更新到 `0.5.16` 并推送，tap commit `1de9ae4`。
- [x] Post-Release Verify run `29801687152` passed，覆盖资产清单和五个平台安装验证。

## 摘要

v0.5.16 继续沿着“面向 AI 编程代理的测试反馈闭环 MCP 服务”推进。这个版本不扩语言，也不承诺测试生成算法提升；重点是让外部 MCP 客户端、编辑器插件和 AI Coding Agent 集成方，可以把 v0.5.15 新增的 Agent 决策 fixture 包真正放进自己的 CI。

v0.5.15 解决的是“fixture 包能导出、能校验”。v0.5.16 解决的是“客户端仓库怎么复制、怎么 dry-run、怎么让 CI / Agent 直接读 JSON 结果”。

## 主要变化

### 外部客户端 CI showcase

- 新增 `scripts/showcase-agent-decision-client-ci.sh`。
- 脚本会创建或使用客户端目录，导出最小 Agent 决策 fixture 包，并在导出包内运行 `npm test --silent`。
- 默认文本输出包含 `agent_decision_client_status=passed`、`agent_decision_fixture_count=8` 和完整决策序列。
- `--json` 输出包含 `status`、`client_dir`、`fixture_dir`、`result_json`、`fixture_count`、`decisions[]`、`failures[]` 和 `validator_exit_code`，方便客户端 CI 和 Agent 直接机器消费。

### GitHub Actions 复制模板

- 新增 [Agent 决策客户端 CI 模板](./agent-decision-client-ci-template.md)。
- 模板可保存为 `.github/workflows/testloop-agent-decision-contract.yml`。
- 模板会 checkout 客户端仓库，设置 Node，checkout `sleticalboy/testloop-mcp` helper，并运行 `.testloop-mcp/scripts/showcase-agent-decision-client-ci.sh --json`。
- 模板会上传 `testloop-agent-decision-client-summary.json`、`agent-decision-fixtures-result.json`、导出包 `package.json` 和 manifest。

### 外部仓库 dry-run

- 新增 `test/agent_decision_client_ci_template_dry_run_test.sh`。
- 测试会创建临时外部客户端目录，把当前仓库挂成 `.testloop-mcp`，按模板中的相对路径执行 helper。
- dry-run 会验证 summary JSON、validator result JSON、导出包 `package.json` 和 manifest 都真实存在。
- dry-run 会解析 summary JSON 和 validator result JSON，固定 `status=passed`、`fixture_count=8`、完整决策序列和空 `failures[]`。

### 外部 helper 路径修复

- `scripts/export-agent-decision-fixtures.mjs` 现在按脚本自身位置定位仓库根目录。
- 外部客户端从 `.testloop-mcp/scripts/...` 调用 helper 时，不再误到客户端仓库目录查找 `docs/fixtures/...`。

## 质量边界

- 这轮提升的是客户端接入和 CI 消费确定性，不改变 `generate_tests` 的核心生成策略。
- 当前模板使用 `ref: v0.5.16` checkout helper，避免客户端 CI 跟随 `main` 漂移。
- `failed/manual_review_*` 仍不进入自动修复循环。客户端应按 manifest 的 `expected_decision` 分流，而不是只看 `status=failed`。

## 推荐验证

- `sh test/agent_decision_client_ci_showcase_test.sh`
- `sh test/agent_decision_client_ci_template_doc_test.sh`
- `sh test/agent_decision_client_ci_template_yaml_test.sh`
- `sh test/agent_decision_client_ci_template_dry_run_test.sh`
- `sh test/agent_decision_fixture_export_test.sh`
- `sh test/client_integration_doc_test.sh`
- `sh test/mcp_client_contract_doc_test.sh`
- `sh test/release_doc_index_test.sh`
- `sh test/docs_links_test.sh`
- `sh test/ci_workflow_test.sh`
- `for t in test/*_test.sh; do sh "$t"; done`
- `go test ./...`
- `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.16-release-prep-dist scripts/verify-release-candidate.sh v0.5.16`
- `git diff --check`
- `TESTLOOP_MCP_REPO=sleticalboy/testloop-mcp scripts/verify-release-assets.sh v0.5.16`
- `gh workflow run post-release-verify.yml -f tag=v0.5.16`

## 发布备注

- 对外文案应强调“外部 MCP 客户端可以复制 GitHub Actions 模板，并用 JSON summary 固定 Agent 决策合同”。
- 推荐演示路径：运行 `scripts/showcase-agent-decision-client-ci.sh --json` 展示结构化结果，再展示 [Agent 决策客户端 CI 模板](./agent-decision-client-ci-template.md) 中的 workflow。
- v0.5.16 已完成正式 GitHub Release、Release assets、资产校验、仓库内 Formula、Homebrew tap 和 Post-Release Verify。
