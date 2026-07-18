# v0.5.4 发布说明草案

## 标题

testloop-mcp v0.5.4

## 发布状态

- [x] 创建 v0.5.4 发布说明草案。
- [x] 梳理 v0.5.3 之后的 Agent onboarding report、失败分流 fixture 和 CI 简化示例。
- [x] 完成本地 release readiness 门禁。
- [x] 正式版本准备时更新 `main.go` MCP implementation version 到 `0.5.4`。
- [x] 正式版本准备时将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.5.4 - 2026-07-18`。
- [x] 正式版本准备时更新 README、安装文档和必要的版本引用。
- [ ] 正式发布前重新跑远端 CI、Release Artifacts、资产校验和 Homebrew tap 更新。

## 摘要

v0.5.4 候选重点是把 v0.5.3 的验收报告能力继续收敛成“公开 onboarding demo + Agent/CI 可执行分流样例”。

这个版本仍不新增语言支持，也不扩测试生成策略。核心变化是让接入方用更少命令完成：

- 生成 Markdown 验收报告，方便人工审阅和上传 artifact。
- 生成 summary JSON，方便 Agent / CI 读取 section 状态。
- 生成 decision 文本，直接暴露 `agent_next_step`。
- 通过 fixture 看懂安装、协议、Agent demo、公开 showcase 和用户项目 smoke 失败时的分流动作。

## 主要变化

### Agent onboarding report

- 新增 `scripts/showcase-agent-onboarding-report.sh`，把验收报告、summary JSON 和决策 demo 收敛成一个公开演示入口。
- 默认输出：
  - `/tmp/testloop-mcp-onboarding/verification-report.md`
  - `/tmp/testloop-mcp-onboarding/verification-summary.json`
  - `/tmp/testloop-mcp-onboarding/agent-decision.txt`
- 脚本复用 `scripts/generate-verification-report.sh` 和 `examples/verification-summary-decision-demo`，避免新增另一套 section 或决策规则。
- 脚本支持 `TESTLOOP_MCP_VERIFY_EXPECT_VERSION` 版本门禁，并透传 `TESTLOOP_REPORT_*` 选项给底层验收报告脚本。
- 新增 `test/showcase_agent_onboarding_report_test.sh`，固定 artifact 路径、summary JSON 和 `agent_next_step=ready` 输出。

### 失败分流样例

- 新增 `docs/verification-summary-failures.md`，说明验收报告失败时如何从 summary JSON 读取失败 section 并决定下一步动作。
- 新增 `docs/fixtures/verification-summary/*.json`，覆盖五类最小失败样例：
  - 基础安装验收失败 -> `fix-installation`
  - 真实 MCP 协议 smoke 失败 -> `inspect-mcp-transport`
  - 最小 Agent 闭环 demo 失败 -> `inspect-agent-demo`
  - 公开 showcase 失败 -> `inspect-showcase`
  - 用户项目 smoke 失败 -> `inspect-user-project`
- 新增 `test/verification_summary_failure_fixtures_test.sh`，逐个 fixture 运行 decision demo 并校验 `agent_next_step`。

### CI 集成收敛

- `docs/verification-ci.md` 优先推荐 `scripts/showcase-agent-onboarding-report.sh`，减少接入方手写 `TESTLOOP_REPORT_SUMMARY_JSON`、decision demo 和 artifact 路径。
- 底层 `scripts/generate-verification-report.sh` 示例保留为高级 workflow，适合需要完全自定义 Markdown / JSON 路径的接入方。
- 前端项目示例同步使用 wrapper 入口。
- `test/verification_ci_doc_test.sh` 已固定推荐 workflow 和高级 workflow 的关键片段。

## 质量边界

- v0.5.4 是 onboarding 和 CI 消费体验 patch，不是生成质量、覆盖率算法或语言覆盖扩张版本。
- `showcase-agent-onboarding-report.sh` 仍然只执行调用方显式配置的用户项目 smoke，不自动猜测测试命令。
- 失败 fixture 是 summary JSON 的最小可消费样例，不是完整 Markdown 报告。
- 公开 showcase 继续 opt-in，不进入默认 CI。
- `/tmp` 中的 Markdown / JSON / decision 输出是本地制品，不提交仓库。

## 本地验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/verify-release-assets.sh`
- [x] `bash -n scripts/generate-verification-report.sh`
- [x] `bash -n scripts/showcase-agent-onboarding-report.sh`
- [x] `bash -n scripts/showcase-go-public-project.sh scripts/showcase-js-public-project.sh scripts/showcase-onboarding.sh`
- [x] `python3 -m py_compile scripts/summarize-showcase-output.py`
- [x] `go test ./...`
- [x] 全部默认 shell 回归测试。
- [x] `go build -o /tmp/testloop-mcp-v0.5.4-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.4-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.4-prep --help` 输出 usage。
- [x] `/tmp/testloop-testgen-v0.5.4-prep --help` 输出 usage。
- [x] 使用真实构建二进制运行 `scripts/showcase-agent-onboarding-report.sh`，输出 `agent_next_step=ready`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.4-prep-dist scripts/package-release-asset.sh v0.5.4 darwin_arm64 darwin arm64`
- [x] `/tmp/testloop-v0.5.4-prep-dist/testloop-mcp_v0.5.4_darwin_arm64.tar.gz.sha256` 校验通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 发布备注

- v0.5.4 适合作为“公开 onboarding demo + 失败分流 fixture + CI wrapper 示例”的 patch 版本。
- 发布文案应突出：AI Agent 需要的不只是测试生成，而是能把安装、协议、demo 和用户项目 smoke 失败清楚分流的反馈入口。
- 正式版本准备前不更新 `main.go` 版本、不改安装文档当前 release、不打 tag。
