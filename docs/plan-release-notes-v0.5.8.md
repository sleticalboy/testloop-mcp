# v0.5.8 发布说明草案

## 标题

testloop-mcp v0.5.8

## 发布状态

- [x] 创建 v0.5.8 发布说明草案。
- [x] 梳理 v0.5.7 之后的接入方一页式验证、真实 server/web 复验、README 复制入口、CI 最小 workflow、失败 triage、Agent 回复格式和安装 checksum fallback 修复。
- [x] 完成本地候选验证：`go test ./...`、完整 shell 矩阵、安装脚本离线回归、文档链接、发布文档索引和 `git diff --check`。
- [x] 完成本地发布前门禁：主服务/testgen 构建、help 输出、darwin arm64 打包 dry-run、sha256 校验和 tarball 内容检查。
- [x] 候选提交远端 CI 已通过到 `daf55c3`，最新成功 run 为 `29668735511`。
- [ ] 正式版本准备时更新 `main.go` MCP implementation version 到 `0.5.8`。
- [ ] 正式版本准备时将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.5.8 - 2026-07-19`。
- [ ] 正式版本准备时同步 README、安装文档和必要版本引用到 `v0.5.8`。

## 摘要

v0.5.8 候选重点是把 v0.5.7 的 first-run / onboarding CI 能力继续收敛成“接入方能直接复制、CI 失败能交给 Agent、Agent 能按固定格式行动”的完整体验。

这个版本不扩语言、不改变 MCP tool 协议，也不调整测试生成算法。核心变化是降低接入方从 README 到 CI 到失败排查的摩擦：

- README 首页直接给出 first-run / onboarding bootstrap 和最小 GitHub Actions workflow。
- `docs/adopter-verification-guide.md` 和 `docs/real-integration-cases.md` 用 v0.5.7 真实 Go server / Vue web 项目复验当前接入路径。
- `docs/ci-agent-triage.md` 固定 CI 失败后下载 artifact、读取 decision/context 和交给 Agent 的流程。
- `docs/first-run-agent-response.md` 固定 Agent 收到 `first-run-context.txt` 后的回复结构和分流动作。
- `scripts/install.sh` 在聚合 `checksums.txt` 存在但缺当前资产时，会继续尝试单资产 `.sha256`，避免旧聚合 checksum 干扰新 release 安装。

## 主要变化

### 接入方复制路径

- 新增 `docs/adopter-verification-guide.md`，把安装、版本确认、本机 first-run、CI bootstrap、artifact 上传和失败分流压成一页执行清单。
- README 新增“用户项目接入：直接复制”入口：
  - 首次接入 / 安装漂移排查 / 失败上下文收集使用 `run-first-run-ci.sh`。
  - 稳定 PR / 发布后 smoke 使用 `run-onboarding-ci.sh`。
  - Go 项目默认 smoke 为 `go test ./...`。
  - Vue / Node 项目 smoke 示例为 `pnpm install --frozen-lockfile && pnpm build`。
- README 补充最小 GitHub Actions first-run workflow，可直接保存为 `.github/workflows/testloop-first-run.yml`。
- 新增 `test/readme_ci_snippet_test.sh`，从 README 提取 YAML 片段并解析，防止首页 CI 示例漂移。

### 真实接入复验

- `docs/real-integration-cases.md` 更新为 v0.5.7 真实 first-run / onboarding CI bootstrap 实跑记录。
- 复验项目：
  - `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server`
  - `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-web`
- 四条路径均通过：
  - server first-run：`first_run_agent_next_step=ready`
  - web first-run：`first_run_agent_next_step=ready`
  - server onboarding：`agent_next_step=ready`
  - web onboarding：`agent_next_step=ready`
- 复验后两个外部项目工作区保持干净。

### CI 失败后交给 Agent

- 新增 `docs/ci-agent-triage.md`，说明：
  - 使用 `gh run download` 下载 `testloop-first-run` artifact。
  - 先读 `agent-decision.txt`。
  - first-run 失败时优先把 `first-run-context.txt` 交给 AI Agent。
  - 需要更细日志时再补 `verification-summary.json`、`verification-report.md` 和 `first-run.log`。
- 文档增加失败态实跑记录：故意失败的外部项目 smoke 会稳定分流到 `inspect-user-project`，summary 中只有“用户项目 smoke”失败，exit code 为 `7`。

### Agent 回复格式

- 新增 `docs/first-run-agent-response.md`，固定 Agent 收到 `first-run-context.txt` 后的四段回复结构：
  - 结论
  - 证据
  - 下一步
  - 暂不做
- 文档明确 `ready`、`fix-installation`、`inspect-mcp-transport`、`inspect-agent-demo`、`inspect-user-project` 和 `inspect-showcase` 的分流动作。
- `inspect-user-project` 示例使用真实失败演练中的 `failed_section=用户项目 smoke` 和 `exit_code=7`。

### 安装 checksum fallback 修复

- `scripts/install.sh` 现在只有在聚合 `checksums.txt` 存在且包含当前资产时才使用它。
- 如果聚合 checksum 存在但缺当前资产，会继续下载 `${asset}.sha256`。
- 新增 `test/install_script_test.sh` 离线用例覆盖该场景，避免后续 release 只上传单资产 `.sha256` 时出现误导性 checksum 错误。

## 真实 dry-run

失败态 first-run triage：

```bash
rm -rf /tmp/testloop-triage-failing-project /tmp/testloop-first-run-failure-triage
mkdir -p /tmp/testloop-triage-failing-project
printf 'intentional failure fixture\n' > /tmp/testloop-triage-failing-project/README.md

TESTLOOP_MCP_VERSION=v0.5.7 \
TESTLOOP_FIRST_RUN_OUTPUT_DIR=/tmp/testloop-first-run-failure-triage \
TESTLOOP_FIRST_RUN_PROJECT_DIR=/tmp/testloop-triage-failing-project \
  scripts/run-first-run-ci.sh 'echo testloop intentional project failure; exit 7'
```

结果：

- `first_run_status=failed`
- `first_run_failed_count=1`
- `first_run_agent_next_step=inspect-user-project`
- `verification-summary.json` 只有“用户项目 smoke”失败，exit code 为 `7`
- `verification-report.md` 保留项目输出 `testloop intentional project failure`

## 质量边界

- v0.5.8 是接入体验和安装脚本 fallback patch，不是生成质量或覆盖率算法版本。
- README 和文档新增的是复制路径、CI artifact 消费路径和 Agent 回复格式，不改变 MCP tool schema。
- 失败态演练使用故意失败的外部 smoke，用于验证分流和上下文，不是 benchmark。

## 本地验证

- [x] `sh test/ci_agent_triage_doc_test.sh`
- [x] `sh test/first_run_agent_response_doc_test.sh`
- [x] `sh test/install_script_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `git diff --check`
- [x] `go build -o /tmp/testloop-mcp-v0.5.8-candidate .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.8-candidate ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.8-candidate --help` 输出 usage，exit code 为 `2`。
- [x] `/tmp/testloop-testgen-v0.5.8-candidate --help` 输出 usage，exit code 为 `2`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.8-candidate-dist scripts/package-release-asset.sh v0.5.8 darwin_arm64 darwin arm64`
- [x] `cd /tmp/testloop-v0.5.8-candidate-dist && shasum -a 256 -c testloop-mcp_v0.5.8_darwin_arm64.tar.gz.sha256`
- [x] tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] 远端 CI run `29668735511` passed。

## 发布备注

- v0.5.8 适合作为“接入方复制路径 + CI 失败交给 Agent + 安装 checksum fallback 修复”的 patch 版本。
- 发布文案应突出：用户不仅能复制 CI，还能在失败后把 artifact 交给 Agent，并让 Agent 按固定格式行动。
