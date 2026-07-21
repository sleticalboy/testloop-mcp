# v0.5.17 发布说明

## 标题

testloop-mcp v0.5.17

## 发布状态

- [x] 创建 v0.5.17 候选发布说明草案。
- [x] 梳理 v0.5.16 之后围绕 Agent 决策客户端 CI 安装路径、接入 Checklist、安装 dry-run 摘要契约和无依赖 validator 的改动边界。
- [x] 最新已完成的远端 CI：`25d0278` run `29807556910` passed，覆盖安装摘要 validator。
- [ ] 正式版本准备文件待更新：implementation version、`CHANGELOG.md` 正式版本段和当前安装/接入文档版本引用需同步到 `0.5.17` / `v0.5.17`。
- [ ] 正式版本准备后的 release readiness 待运行：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.17-release-prep-dist scripts/verify-release-candidate.sh v0.5.17`。
- [ ] `v0.5.17` tag、GitHub Release、Release assets、资产校验、Homebrew Formula、Homebrew tap 和 Post-Release Verify 待执行。

## 摘要

v0.5.17 继续强化“面向 AI 编程代理的测试反馈闭环 MCP 服务”的客户端接入层。这个版本不扩语言，也不改变测试生成算法；重点是把 v0.5.16 的外部客户端 CI 模板继续推进到“可一键安装、可照 checklist 接入、可用 JSON sample/validator 固定输出”的发布边界。

v0.5.16 解决的是“客户端仓库可以复制 workflow 并跑 Agent 决策 fixture contract”。v0.5.17 解决的是“客户端仓库可以用 installer 生成 workflow，并对安装 dry-run 输出做机器校验”。

## 主要变化

### 一键安装客户端 CI 模板

- 新增 `scripts/install-agent-decision-client-ci-template.sh`。
- 脚本默认写入 `.github/workflows/testloop-agent-decision-contract.yml`。
- 支持 `--version` 固定 helper ref，支持 `--dry-run` 预览，支持 `--force` 覆盖已有 workflow。
- 脚本支持脱离仓库单文件运行；从 `main` raw URL 下载时会回退到内置稳定 helper tag。
- 安装脚本测试会比较脚本生成 workflow 与文档 YAML 模板，并检查默认 helper ref 与 `main.go` 的 `appVersion` 同步。

### 安装 dry-run 闭环

- 新增 `scripts/showcase-agent-decision-client-ci-template-install.sh`。
- 脚本覆盖“下载或读取 installer -> 生成 workflow -> 模拟 `.testloop-mcp` helper checkout -> 执行 Agent 决策 fixture contract”。
- 支持 `--json`，输出 installer 来源、客户端目录、workflow 路径、helper ref、fixture 数量、决策序列、failures 和退出码。
- 仓库测试会用本地 installer 路径和 `file://` installer URL 代替网络下载，保证 CI 稳定。

### JSON 摘要契约

- 新增 `docs/fixtures/agent-decision-client-ci-template-install-summary.schema.json`。
- 新增 `docs/fixtures/agent-decision-client-ci-template-install-summary/passed.json` 通过态样例。
- 新增 `scripts/validate-agent-decision-client-ci-install-summary.mjs`，可无依赖校验安装 dry-run JSON 输出。
- validator 支持文本输出和 `--json` 输出，固定 `fixture_count=8`、决策序列、空 `failures[]`、退出码和 installer URL。

### 一页式接入 Checklist

- 新增 [Agent 决策客户端 CI 接入 Checklist](./agent-decision-client-ci-checklist.md)。
- Checklist 把 helper ref、installer 下载、workflow 生成、CI 运行、artifact、manifest 分流和失败排查压成一页式步骤。
- 新增 checklist 命令回归测试，会从 Markdown 抽取安装、contract 和安装 dry-run 命令并实际执行，避免复制命令漂移。

## 质量边界

- 当前重点仍是 Agent 客户端测试反馈基础设施，不是“更会自动写单测”。
- 模板和 installer 仍默认固定到稳定 tag，避免客户端 CI 跟随 `main` 漂移。
- `failed/manual_review_*` 仍不进入自动修复循环。客户端应按 manifest 的 `expected_decision` 分流。
- 安装 summary validator 面向通过态 smoke；如果 dry-run 输出 `status=failed`，validator 会失败并让客户端 CI 停下。

## 推荐验证

- `sh test/install_agent_decision_client_ci_template_test.sh`
- `sh test/agent_decision_client_ci_template_install_showcase_test.sh`
- `sh test/agent_decision_client_ci_template_install_summary_schema_test.sh`
- `sh test/agent_decision_client_ci_install_summary_validator_test.sh`
- `sh test/agent_decision_client_ci_checklist_doc_test.sh`
- `sh test/agent_decision_client_ci_checklist_commands_test.sh`
- `sh test/agent_decision_client_ci_template_doc_test.sh`
- `sh test/client_integration_doc_test.sh`
- `sh test/mcp_client_contract_doc_test.sh`
- `sh test/release_doc_index_test.sh`
- `sh test/docs_links_test.sh`
- `sh test/ci_workflow_test.sh`
- `for t in test/*_test.sh; do sh "$t"; done`
- `go test ./...`
- `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.17-release-prep-dist scripts/verify-release-candidate.sh v0.5.17`
- `git diff --check`

## 发布备注

- 对外文案应强调“外部 MCP 客户端可以一键安装 GitHub Actions contract，并用 JSON summary validator 固定 Agent 决策合同”。
- 推荐演示路径：运行 installer 生成 workflow，再运行 `scripts/showcase-agent-decision-client-ci-template-install.sh --json`，最后运行 `node scripts/validate-agent-decision-client-ci-install-summary.mjs /path/to/install-summary.json`。
- 正式发布时需要把模板默认 `ref`、installer fallback、文档命令和测试期望统一更新到 `v0.5.17`。
