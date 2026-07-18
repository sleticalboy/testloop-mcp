# v0.5.5 发布说明草案

## 标题

testloop-mcp v0.5.5

## 发布状态

- [x] 创建 v0.5.5 发布说明草案。
- [x] 梳理 v0.5.4 之后的真实接入案例模板、安装漂移诊断和 Homebrew 安装态验收。
- [x] 完成本地 release readiness 门禁。
- [x] 正式版本准备已更新 `main.go` MCP implementation version 到 `0.5.5`。
- [x] 正式版本准备已将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.5.5 - 2026-07-18`。
- [x] 正式版本准备已更新 README、安装文档和必要的版本引用。
- [ ] 正式发布前重新跑远端 CI、Release Artifacts、资产校验和 Homebrew tap 更新。

## 摘要

v0.5.5 候选重点是把 v0.5.4 的 onboarding report 能力落到真实接入与安装漂移排查上。

这个版本仍不扩语言、不新增测试生成策略，也不把卖点转回“自动生成测试”。核心变化是让接入方能更稳地完成：

- 用真实 Go server / Vue web 项目验证 onboarding report wrapper 的落地方式。
- 从 Markdown、summary JSON 和 `agent_next_step` 三类制品判断项目是否已可进入下一步闭环。
- 在 Homebrew 安装仍指向旧二进制时，得到明确的升级/重装建议，而不是只看到 Go flag usage。
- 用真实安装态 `testloop-mcp` 跑通基础安装验收、真实 MCP 协议 smoke 和最小 Agent demo。

## 主要变化

### 真实接入案例模板

- 新增 `docs/real-integration-cases.md`，沉淀接入方项目使用 `scripts/showcase-agent-onboarding-report.sh` 的推荐模板。
- 文档明确固定四个变量：
  - `TESTLOOP_MCP_VERIFY_EXPECT_VERSION`
  - `TESTLOOP_ONBOARDING_OUTPUT_DIR`
  - `TESTLOOP_REPORT_PROJECT_DIR`
  - `TESTLOOP_REPORT_PROJECT_COMMAND`
- 文档解释三类输出制品：
  - `verification-report.md`
  - `verification-summary.json`
  - `agent-decision.txt`
- 文档提供 `agent_next_step` 到下一步动作的判断表，覆盖 `ready`、`fix-installation`、`inspect-mcp-transport`、`inspect-agent-demo`、`inspect-showcase` 和 `inspect-user-project`。
- 新增 `test/real_integration_cases_doc_test.sh`，固定真实接入文档里的关键命令、环境变量、样例路径和决策字段。

### laoxia server / web 实跑记录

本轮使用 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0` 作为真实接入样例：

- Go server：`car-admin-server`，用户项目命令 `go test ./...`。
- Vue web：`car-admin-web`，用户项目命令 `pnpm install --frozen-lockfile && pnpm build:prod`。

两条链路都使用当前源码构建的 `/tmp/testloop-mcp-v0.5.4-case` 跑通 onboarding report：

- `overall_status=passed`
- `failed_count=0`
- `agent_next_step=ready`

这些记录不等同于 `validate_coverage_task` 的生成质量 benchmark；它们证明的是“真实项目 smoke 可以被纳入同一份 Agent/CI 可消费验收报告”。

### 安装漂移诊断

- 增强 `scripts/verify-client-setup.sh`：当旧二进制缺少 `--version`、`--version` 输出不合法或版本门禁不匹配时，输出原始版本命令结果和 Homebrew 升级/重装建议。
- 旧二进制常见错误 `flag provided but not defined: -version` 会被解释为安装漂移，而不是让用户只看到 Go flag usage。
- 版本不匹配时复用同一诊断提示，方便用户从 `TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.4` 的失败结果直接知道下一步命令。
- 扩充 `test/verify_client_setup_test.sh`，覆盖旧二进制缺少 `--version` 和版本不匹配两类提示。

### Homebrew 安装态验收

本机实测：

- `brew info testloop-mcp` 显示 tap stable 为 `0.5.4`，但 installed/linked 仍为 `0.5.0`。
- `brew update` auto-update 一度卡住，改用 `HOMEBREW_NO_AUTO_UPDATE=1 brew upgrade sleticalboy/tap/testloop-mcp` 成功升级到 `0.5.4`。
- `testloop-mcp --version` 输出 `testloop-mcp 0.5.4`。
- 真实安装二进制 `/opt/homebrew/bin/testloop-mcp` 跑通基础安装验收。
- 真实安装二进制跑通 `scripts/showcase-agent-onboarding-report.sh "$(command -v testloop-mcp)"`，summary JSON 为 `overall_status=passed`、`failed_count=0`，decision 输出 `agent_next_step=ready`。

## 质量边界

- v0.5.5 是接入体验和安装诊断 patch，不是生成质量、覆盖率算法或语言覆盖扩张版本。
- `docs/real-integration-cases.md` 的 laoxia 样例是 onboarding report 的真实接入记录，不是公开 benchmark。
- Homebrew auto-update 慢或卡住属于环境/网络问题；文档只提供 `HOMEBREW_NO_AUTO_UPDATE=1` 的可选绕行路径，不建议长期禁用更新。
- 正式版本准备已切 `main.go` 版本和文档版本引用；当前仍不打 tag、不更新 Homebrew tap，等 Release Artifacts 生成后再用真实 asset digest 更新。

## 本地验证

- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `bash -n scripts/verify-release-assets.sh`
- [x] `bash -n scripts/generate-verification-report.sh`
- [x] `bash -n scripts/showcase-agent-onboarding-report.sh`
- [x] `bash -n scripts/verify-client-setup.sh`
- [x] `bash -n scripts/showcase-go-public-project.sh scripts/showcase-js-public-project.sh scripts/showcase-onboarding.sh`
- [x] `python3 -m py_compile scripts/summarize-showcase-output.py`
- [x] `go test ./...`
- [x] 全部默认 shell 回归测试。
- [x] `go build -o /tmp/testloop-mcp-v0.5.5-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.5-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.5-prep --help` 输出 usage。
- [x] `/tmp/testloop-testgen-v0.5.5-prep --help` 输出 usage。
- [x] `/tmp/testloop-mcp-v0.5.5-prep --version` 输出 `testloop-mcp 0.5.5`。
- [x] 使用 v0.5.5 准备二进制运行 onboarding wrapper，输出 `agent_next_step=ready`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.5-prep-dist scripts/package-release-asset.sh v0.5.5 darwin_arm64 darwin arm64`
- [x] `/tmp/testloop-v0.5.5-prep-dist/testloop-mcp_v0.5.5_darwin_arm64.tar.gz.sha256` 校验通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 发布备注

- v0.5.5 适合作为“真实接入案例 + 安装漂移诊断 + Homebrew 安装态验收”的 patch 版本。
- 发布文案应突出：AI Agent 的测试反馈闭环要能稳定接入真实项目，也要能在安装/版本漂移时给出清晰下一步。
- v0.5.5 版本准备已完成；正式发布前还需要等待远端 CI，通过后再打 tag、生成 Release Artifacts，并用真实 asset digest 更新 Homebrew tap。
