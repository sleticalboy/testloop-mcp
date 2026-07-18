# v0.5.6 发布说明草案

## 标题

testloop-mcp v0.5.6

## 发布状态

- [x] 创建 v0.5.6 发布说明草案。
- [x] 梳理 v0.5.5 之后的 Onboarding CI 复制模板、bootstrap 脚本、YAML 可解析性校验和失败排查能力。
- [x] 完成本地 release readiness 门禁。
- [ ] 正式版本准备时再更新 `main.go` MCP implementation version 到 `0.5.6`。
- [ ] 正式版本准备时再收敛 `CHANGELOG.md` 的 `Unreleased` 内容。
- [ ] 正式版本准备时再同步 README、安装文档和必要的版本引用。

## 摘要

v0.5.6 候选重点是把 v0.5.5 的真实接入验收继续推进到“用户项目 CI 可复制、失败可分流”的路径。

这个版本仍不扩语言、不改测试生成策略，也不把定位转回“自动生成测试”。核心变化是让外部用户项目可以在 GitHub Actions 中更低成本地接入 testloop-mcp，并在失败时拿到稳定的下一步判断：

- 复制一份 Go server / Vue web workflow，就能生成 onboarding artifact。
- 外部用户仓库不需要本地拥有 testloop-mcp 源码里的 `scripts/` 目录。
- CI step summary 会直接展示状态、失败数量、artifact 路径和 `agent_next_step`。
- 失败时有明确排查顺序，便于把结构化上下文交给 AI Agent 继续修。

## 主要变化

### Onboarding CI 复制模板

- 新增 `docs/onboarding-ci-template.md`，提供 Go server 和 Vue / Node 项目的最小 GitHub Actions 模板。
- 模板默认上传三类 artifact：
  - `verification-report.md`
  - `verification-summary.json`
  - `agent-decision.txt`
- 新增 `test/onboarding_ci_template_doc_test.sh`，固定模板里的版本门禁、输出目录、项目 smoke 命令和 artifact 路径。

### Workflow YAML 可解析性

- 新增 `test/onboarding_ci_template_yaml_test.sh`，从 Markdown 中抽取完整 `yaml` fenced block。
- 测试要求保留 Go server 与 Vue / Node 两个完整 workflow 示例。
- 使用 Ruby 标准库 `yaml` 解析 workflow，并校验 `name`、`on` 和 `jobs.onboarding` 等关键结构。

### Onboarding CI bootstrap

- 新增 `scripts/run-onboarding-ci.sh`，用于外部用户项目 CI。
- 脚本会安装或解析 `testloop-mcp`，准备 testloop-mcp helper checkout，并调用 onboarding report wrapper。
- 脚本支持：
  - `TESTLOOP_ONBOARDING_PROJECT_DIR`
  - `TESTLOOP_ONBOARDING_PROJECT_COMMAND`
  - `TESTLOOP_ONBOARDING_OUTPUT_DIR`
  - `TESTLOOP_MCP_VERSION`
  - `TESTLOOP_MCP_COMMAND`
  - `TESTLOOP_MCP_REPO_DIR`
- 当设置具体 `TESTLOOP_MCP_VERSION` 时，脚本会安装并使用该版本，避免误复用 PATH 上的旧 Homebrew 二进制。
- 新增 `test/run_onboarding_ci_test.sh`，覆盖 fake binary、指定版本安装、成功路径、用户项目 smoke 失败路径和 GitHub step summary 输出。

### 失败路径排查

- `scripts/run-onboarding-ci.sh` 在 `GITHUB_STEP_SUMMARY` 存在时会写入 CI step summary。
- Step summary 包含：
  - `Status`
  - `Failed sections`
  - `agent_next_step`
  - Markdown report 路径
  - Summary JSON 路径
  - Agent decision 路径
- 新增 `docs/onboarding-ci-failure-triage.md`，说明失败时先看 step summary，再看 `agent-decision.txt`、`verification-summary.json` 和 `verification-report.md`。
- 新增 `test/onboarding_ci_failure_triage_doc_test.sh`，固定失败分流 action、artifact 文件名和 AI Agent 粘贴上下文。

## 真实 dry-run

本轮使用当前仓库作为用户项目，运行：

```bash
TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp-v0.5.6-candidate \
TESTLOOP_MCP_REPO_DIR="$PWD" \
TESTLOOP_ONBOARDING_PROJECT_DIR="$PWD" \
TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-v0.5.6-candidate-onboarding \
  scripts/run-onboarding-ci.sh 'go test ./...'
```

结果：

- `overall_status=passed`
- `failed_count=0`
- `agent_next_step=ready`
- 基础安装验收、真实 MCP 协议 smoke、最小 Agent demo 和用户项目 smoke 均为 `passed`。
- 公开 showcase 按默认策略 `skipped`。
- 候选阶段尚未切 implementation version，候选二进制 `--version` 仍输出 `testloop-mcp 0.5.5`。

## 质量边界

- v0.5.6 是 onboarding CI 接入体验 patch，不是生成质量或覆盖率算法版本。
- `scripts/run-onboarding-ci.sh` 面向 CI bootstrap，不替代底层 `generate-verification-report.sh` 的高级参数能力。
- 公开 showcase 仍默认关闭，避免首次接入 CI 依赖外部网络和公共仓库状态。
- 当前仍不打 tag、不更新 Homebrew tap；正式版本准备通过后再进入发布链路。

## 本地验证

- [x] `bash -n scripts/run-onboarding-ci.sh`
- [x] `go test ./...`
- [x] 全部默认 shell 回归测试。
- [x] `scripts/run-onboarding-ci.sh 'go test ./...'` 真实 dry-run。
- [x] 主服务 / testgen 构建。
- [x] darwin arm64 打包 dry-run。
- [x] `git diff --check`

## 发布备注

- v0.5.6 适合作为“外部用户项目 Onboarding CI bootstrap + 失败分流”的 patch 版本。
- 发布文案应突出：接入方可以复制最小 workflow，在成功和失败时都拿到 Agent 可消费的 artifact 与下一步动作。
