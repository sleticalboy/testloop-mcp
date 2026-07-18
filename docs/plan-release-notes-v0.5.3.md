# v0.5.3 发布说明草案

## 标题

testloop-mcp v0.5.3

## 发布状态

- [x] 创建 v0.5.3 发布说明草案。
- [x] 梳理 v0.5.2 之后的验收报告、summary JSON、Agent/CI 决策示例和 CI 集成文档。
- [x] 完成本地 release readiness 门禁。
- [x] 正式版本准备时更新 `main.go` MCP implementation version 到 `0.5.3`。
- [x] 正式版本准备时将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.5.3 - 2026-07-18`。
- [x] 正式版本准备时更新 README、安装文档和必要的版本引用。
- [x] 正式发布前重新跑远端 CI、Release Artifacts、资产校验和 Homebrew tap 更新。

## 摘要

v0.5.3 候选重点是把 v0.5.2 的安装验收和公开 showcase 继续推进成可交付的“验收报告闭环”。

这个版本仍不新增语言支持，也不把定位改成“测试生成器”。核心变化是让 AI Agent、CI 和接入方能围绕一份稳定报告完成：

- 生成 Markdown 验收报告，方便人工审阅和上传 artifact。
- 生成 summary JSON，方便 Agent / CI 不解析 Markdown 就能读取状态。
- 读取 summary JSON 输出下一步 action，把失败归因到安装、协议、Agent demo、公开 showcase 或用户项目 smoke。
- 给 GitHub Actions 提供可复制集成示例。

## 主要变化

### 用户项目验收报告

- 新增 `scripts/generate-verification-report.sh`，默认聚合基础安装验收、真实 MCP 协议 smoke 和最小 Agent 闭环 demo。
- 用户项目 smoke 通过 `TESTLOOP_REPORT_PROJECT_DIR` 和 `TESTLOOP_REPORT_PROJECT_COMMAND` 显式传入，避免脚本猜测项目测试命令。
- 公开 Go / JS showcase 通过 `TESTLOOP_REPORT_PUBLIC_SHOWCASES=go|js|all` 显式开启，默认不访问公网。
- 脚本会在任一已执行 section 失败时仍写出 Markdown 报告，并返回非零 exit code。
- `test/verification_report_test.sh` 已纳入 CI，固定成功、失败和 skipped 报告行为。

### Summary JSON 和 Agent 决策

- `scripts/generate-verification-report.sh` 支持 `TESTLOOP_REPORT_SUMMARY_JSON`，额外输出机器可读 summary JSON。
- JSON 包含 `overall_status`、`failed_count`、报告元数据和 `sections[]` 的 `name/status/exit_code/reason`。
- skipped section 的 `exit_code` 为 `null`，失败 section 保留真实 exit code。
- 新增 `examples/verification-summary-decision-demo`，读取 summary JSON 后输出 `agent_next_step`。
- 决策示例能区分：
  - `ready`
  - `fix-installation`
  - `inspect-mcp-transport`
  - `inspect-agent-demo`
  - `inspect-showcase`
  - `inspect-user-project`
- `test/verification_summary_decision_demo_test.sh` 已纳入 CI，覆盖整体通过、用户项目失败和 MCP 协议失败。

### 真实项目 smoke 样例

- 用 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server` 跑通 Go server 验收报告，用户项目 smoke 为 `go test ./...`。
- 用 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-web` 跑通 Vue web 验收报告，用户项目 smoke 为 `pnpm install --frozen-lockfile && pnpm build:prod`。
- 两份报告的基础安装验收、真实 MCP 协议 smoke、最小 Agent 闭环 demo 和用户项目 smoke 均为 `passed`。
- 文档记录 server 侧 macOS deprecated warning 和 web 侧 browserslist / bundle size warning，并明确这些是 warning 不是失败。

### CI 集成文档

- 新增 `docs/verification-report.md`，说明本地报告、版本门禁、用户项目 smoke、公开 showcase、summary JSON 和真实项目 smoke 记录。
- 新增 `docs/verification-ci.md`，提供 GitHub Actions 中生成 Markdown + JSON 报告、运行决策 demo、失败时上传 artifact 的可复制示例。
- README、showcase 索引和 release 文档索引已补验收报告入口。
- 新增 `test/verification_ci_doc_test.sh`，固定 CI 示例中的关键环境变量、命令、artifact 路径和决策 demo 入口。

## 质量边界

- v0.5.3 是接入验收和 Agent/CI 可消费性版本，不是生成质量或语言覆盖扩张版本。
- 验收报告的用户项目 smoke 只执行调用方显式传入的命令，不自动推断框架、数据库、外部服务或构建流程。
- 公开 showcase 仍保持 opt-in，不进入默认 CI。
- `/tmp` 中的 Markdown / JSON 报告是本地制品，不提交仓库；文档只记录摘要和可复现命令。
- summary JSON 是验收报告的汇总，不替代 `validate_coverage_task` 的结构化 MCP 返回。

## 本地验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/verify-release-assets.sh`
- [x] `bash -n scripts/generate-verification-report.sh`
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
- [x] `sh test/verification_summary_decision_demo_test.sh`
- [x] `sh test/showcase_scripts_test.sh`
- [x] `sh test/showcase_summary_test.sh`
- [x] `sh test/verification_report_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/docs_json_examples_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/fixtures_index_test.sh`
- [x] `sh test/fixture_decision_mapping_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/verification_ci_doc_test.sh`
- [x] `go build -o /tmp/testloop-mcp-v0.5.3-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.3-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.3-prep --help` 输出 usage。
- [x] `/tmp/testloop-testgen-v0.5.3-prep --help` 输出 usage。
- [x] `TESTLOOP_REPORT_EXPECT_VERSION=0.5.3 TESTLOOP_REPORT_SUMMARY_JSON=/tmp/testloop-v0.5.3-summary.json scripts/generate-verification-report.sh /tmp/testloop-mcp-v0.5.3-prep /tmp/testloop-v0.5.3-report.md`
- [x] `go run ./examples/verification-summary-decision-demo /tmp/testloop-v0.5.3-summary.json` 输出 `agent_next_step=ready`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.3-prep-dist scripts/package-release-asset.sh v0.5.3 darwin_arm64 darwin arm64`
- [x] `cd /tmp/testloop-v0.5.3-prep-dist && shasum -a 256 -c testloop-mcp_v0.5.3_darwin_arm64.tar.gz.sha256`
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`
- [x] 远端 CI run `29635368963` 通过。
- [x] `v0.5.3` tag 已推送，Release Artifacts run `29635462891` 通过。
- [x] `scripts/verify-release-assets.sh v0.5.3`
- [x] GitHub Release 正文已更新为正式 v0.5.3 发布说明。
- [x] 仓库内 `Formula/testloop-mcp.rb` 已使用真实 Release asset digest 更新到 `0.5.3`，并通过 `ruby -c` 和 `brew style`。
- [x] `sleticalboy/homebrew-tap` 已更新并推送到 `b099aba`。
- [x] 本机 Homebrew tap 已快进到 `b099aba`，`HOMEBREW_NO_AUTO_UPDATE=1 brew fetch --force sleticalboy/tap/testloop-mcp` 获取 `0.5.3` 成功。
- [x] Post-Release Verify run `29635745094` 通过，五平台安装验收全部成功。

## 发布备注

- v0.5.3 适合作为“验收报告 + summary JSON + Agent/CI 分流示例”的 patch 版本。
- 发布文案应突出：AI Agent 不只需要测试生成能力，也需要稳定的验收报告、结构化状态和可复用 CI 反馈入口。
- 正式版本准备前不更新 `main.go` 版本、不改安装文档当前 release、不打 tag。
