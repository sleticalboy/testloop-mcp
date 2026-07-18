# v0.5.7 发布说明

## 标题

testloop-mcp v0.5.7

## 发布状态

- [x] 创建 v0.5.7 发布说明草案。
- [x] 梳理 v0.5.6 之后的首跑诊断 CI、失败上下文、外部项目复制演练和入口选择规则。
- [x] 完成本地 release readiness 门禁。
- [x] 正式版本准备已更新 `main.go` MCP implementation version 到 `0.5.7`。
- [x] 正式版本准备已将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.5.7 - 2026-07-19`。
- [x] 正式版本准备已同步 README、安装文档和必要的版本引用。
- [ ] Release Artifacts workflow 通过并上传五平台资产和 `.sha256`。
- [ ] `scripts/verify-release-assets.sh v0.5.7` 验证 10 个 Release 资产完整。
- [ ] GitHub Release 正文更新为正式 v0.5.7 发布说明。
- [ ] 仓库内 Formula 使用 Release 真实 digest 更新到 `0.5.7`。
- [ ] Homebrew tap 更新到 `testloop-mcp 0.5.7`。
- [ ] Post-Release Verify 通过。

## 摘要

v0.5.7 候选重点是把 v0.5.6 的 Onboarding CI 继续推进成更完整的“首次接入诊断”路径。

这个版本仍不扩语言、不调整生成算法，也不改变 MCP 工具协议。核心变化是让外部用户项目第一次接入 testloop-mcp 时，能够在 CI 中拿到更适合交给 AI Agent 的诊断上下文：

- `run-first-run-ci.sh` 输出 report、summary、decision、context、log 五件套。
- 首跑失败时有可粘贴给 AI Agent 的 `first-run-context.txt`。
- Onboarding CI 和 first-run CI 都有外部项目复制演练，验证脚本不依赖 testloop-mcp 仓库作为当前工作目录。
- 文档明确首次接入和稳定验收分别该用哪个 bootstrap，减少用户选择成本。

## 主要变化

### 首跑诊断入口

- 新增 `scripts/doctor-first-run.sh`，聚合安装验收、真实 MCP transport、最小 Agent demo 和可选用户项目 smoke。
- 脚本稳定输出：
  - `first_run_status`
  - `first_run_failed_count`
  - `first_run_agent_next_step`
  - Markdown report 路径
  - summary JSON 路径
  - decision 路径
  - `first-run-context.txt`
  - `first-run.log`
- 新增 `docs/first-run-diagnostics.md` 和 `test/doctor_first_run_test.sh`，固定成功路径、用户项目失败路径、help 和参数错误。

### 首跑失败上下文

- `scripts/doctor-first-run.sh` 会写出 `first-run-context.txt`。
- 新增 `docs/first-run-failures.md` 和 `docs/fixtures/first-run/*.txt`，覆盖：
  - `fix-installation`
  - `inspect-mcp-transport`
  - `inspect-agent-demo`
  - `inspect-showcase`
  - `inspect-user-project`
- 新增 `test/first_run_failure_fixtures_test.sh`，固定 fixture 字段、action 和 AI Agent 粘贴提示。

### 首跑诊断 CI bootstrap

- 新增 `scripts/run-first-run-ci.sh`，面向外部用户项目 CI。
- 脚本会安装或解析 `testloop-mcp`，准备 helper checkout，并调用首跑诊断入口。
- 脚本输出五件套 artifact：
  - `verification-report.md`
  - `verification-summary.json`
  - `agent-decision.txt`
  - `first-run-context.txt`
  - `first-run.log`
- 新增 `docs/first-run-ci-template.md`、`test/first_run_ci_template_doc_test.sh` 和 `test/first_run_ci_template_yaml_test.sh`，提供 Go server 与 Vue / Node 两份可复制 workflow。
- `run-first-run-ci.sh` 的 helper checkout 默认 ref 为 `main`，确保 v0.5.6 二进制可搭配当前 main 的首跑诊断 helper。

### 外部项目复制演练

- 新增 `scripts/showcase-onboarding-ci-external-project.sh` 和 `docs/onboarding-ci-external-dry-run.md`，用临时 Go 或 Node 项目验证 Onboarding CI bootstrap 的复制路径。
- 新增 `scripts/showcase-first-run-ci-external-project.sh` 和 `docs/first-run-ci-external-dry-run.md`，用临时 Go 或 Node 项目验证首跑诊断 CI bootstrap 的复制路径。
- 两条演练都支持 `go`、`node` 和 `all` 模式。
- 首跑外部演练已用本地构建二进制和 `TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE=all` 完成真实验证，Go 与 Node 两条路径均输出 `first_run_agent_next_step=ready`。

### CI 入口选择规则

- `docs/verification-ci.md` 新增“怎么选入口”章节：
  - 首次接入、安装后排查、或希望失败时直接交给 AI Agent，优先使用 `run-first-run-ci.sh`。
  - 稳定接入后的 PR / 发布后 smoke，优先使用 `run-onboarding-ci.sh`。
  - 维护者改模板后，使用两条 external showcase 脚本复验复制路径。
- README 在 bootstrap 示例后增加直达链接，减少首次接入时的判断成本。

## 真实 dry-run

首跑诊断 CI 外部项目 all 模式：

```bash
go build -o /tmp/testloop-mcp-external-first-run .
TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp-external-first-run \
TESTLOOP_MCP_VERSION=v0.5.7 \
TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE=all \
  scripts/showcase-first-run-ci-external-project.sh
```

结果：

- Go 路径输出 `external_first_run_go_status=passed`。
- Node 路径输出 `external_first_run_node_status=passed`。
- 两条路径均输出 `first_run_agent_next_step=ready`。
- 最终输出 `external_first_run_mode=all`、`external_first_run_status=passed`。
- 正式版本准备二进制 `--version` 输出 `testloop-mcp 0.5.7`。
- Onboarding CI 外部项目 all 模式也已用同一 v0.5.7 candidate 二进制复验通过，最终输出 `external_onboarding_mode=all`、`external_onboarding_status=passed`。

## 质量边界

- v0.5.7 是首次接入诊断和 CI 复制体验 patch，不是生成质量或覆盖率算法版本。
- `run-first-run-ci.sh` 与 `run-onboarding-ci.sh` 都是外部 CI bootstrap；高级路径仍由 `generate-verification-report.sh` 和 `doctor-first-run.sh` 承担。
- 外部项目演练是 opt-in，不进入默认 CI 的完整执行矩阵，避免常规提交依赖包管理器网络或外部下载。
- Onboarding CI helper ref 仍默认跟随具体 `TESTLOOP_MCP_VERSION`，首跑诊断 CI helper ref 默认 `main`，因为首跑 helper 是 v0.5.6 之后新增能力。

## 本地验证

- [x] `bash -n scripts/run-first-run-ci.sh`
- [x] `bash -n scripts/showcase-first-run-ci-external-project.sh`
- [x] `go test ./...`
- [x] 全部默认 shell 回归测试。
- [x] `scripts/showcase-first-run-ci-external-project.sh` 真实 all 模式 dry-run。
- [x] 主服务 / testgen 构建。
- [x] 主服务 / testgen `--help` 输出 usage；Go flag 当前 help exit code 为 `2`。
- [x] darwin arm64 打包 dry-run。
- [x] sha256 校验和 tarball 内容检查。
- [x] `git diff --check`
- [x] 远端 CI run `29651790811` passed。

## 发布备注

- v0.5.7 适合作为“首次接入诊断 CI + 外部项目复制演练”的 patch 版本。
- 发布文案应突出：用户第一次接入时不仅能知道失败，还能拿到可交给 AI Agent 的上下文文件和下一步动作。
