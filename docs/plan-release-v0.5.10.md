# v0.5.10 发布检查清单

## 当前目标

这是 v0.5.10 的候选发布检查清单。当前目标是把 v0.5.9 之后围绕 `agent-response.txt` 的 first-run / onboarding artifact 消费改动归档为一个 patch 版本。

发布重点见 [v0.5.10 发布说明草案](./plan-release-notes-v0.5.10.md)。

## 当前差异核对

- [x] first-run artifact 目录入口已加入。
- [x] first-run CI 自动生成 `agent-response.txt`。
- [x] first-run 失败 artifact fixture 已升级为六件套。
- [x] first-run 失败排查读取优先级已收敛为 `agent-response.txt` 优先。
- [x] 外部 first-run showcase 已校验六件套 artifact。
- [x] onboarding Agent 回复 demo 和目录入口已加入。
- [x] onboarding CI 自动生成 `agent-response.txt`。
- [x] onboarding CI 模板、失败排查、接入指南、验收 CI 和 README 已同步为四件套。
- [x] 外部 onboarding showcase 已校验四件套 artifact。
- [x] onboarding 失败 artifact fixture 已加入。
- [x] 客户端集成文档已区分 first-run 和 onboarding 两类 CI artifact fixture。
- [x] CHANGELOG 和 roadmap 已记录候选内容。

## 候选内容

- [x] first-run artifact 从目录到 Agent 回复的稳定入口。
- [x] first-run bootstrap 自动生成 Agent 回复草稿。
- [x] onboarding artifact 从 summary 到 Agent 回复的稳定入口。
- [x] onboarding bootstrap 自动生成 Agent 回复草稿。
- [x] 外部 first-run / onboarding showcase 分别校验六件套和四件套。
- [x] first-run / onboarding 失败 artifact fixture 供客户端回归。

## 已验证

- [x] `sh test/onboarding_agent_response_demo_test.sh`
- [x] `sh test/run_onboarding_ci_test.sh`
- [x] `sh test/onboarding_ci_template_doc_test.sh`
- [x] `sh test/onboarding_ci_failure_triage_doc_test.sh`
- [x] `sh test/onboarding_ci_external_dry_run_doc_test.sh`
- [x] `sh test/onboarding_artifact_fixtures_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `go build -o /tmp/testloop-mcp-external-onboarding-fourpack .`
- [x] `TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp-external-onboarding-fourpack TESTLOOP_MCP_VERSION=v0.5.9 scripts/showcase-onboarding-ci-external-project.sh`
- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `git diff --check`
- [x] `101b14e` 远端 CI run `29673054100` passed。
- [x] `d3dbb86` 远端 CI run `29673143525` passed。
- [x] `44071a0` 远端 CI run `29673246805` passed。

## 发布前门禁

- [ ] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [ ] `go test ./...`
- [ ] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [ ] `go build -o /tmp/testloop-mcp-v0.5.10-candidate .`
- [ ] `go build -o /tmp/testloop-testgen-v0.5.10-candidate ./cmd/testgen`
- [ ] `/tmp/testloop-mcp-v0.5.10-candidate --help` 输出 usage，exit code 为 `2`。
- [ ] `/tmp/testloop-testgen-v0.5.10-candidate --help` 输出 usage，exit code 为 `2`。
- [ ] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.10-candidate-dist scripts/package-release-asset.sh v0.5.10 darwin_arm64 darwin arm64`
- [ ] 在 dist 目录内校验 `testloop-mcp_v0.5.10_darwin_arm64.tar.gz.sha256` 通过。
- [ ] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [ ] `git diff --check`

## 正式发布前待办

- [ ] 更新 `main.go` MCP implementation version 到 `0.5.10`。
- [ ] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.10 - 2026-07-19`。
- [ ] 同步 README 中当前 Release、手动下载示例和 Windows 下载示例到 `v0.5.10`。
- [ ] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.5.10`。
- [ ] 同步 quickstart、first-run、verification CI、onboarding CI 和接入指南中的版本门禁到 `0.5.10`。
- [ ] 测试中的版本期望同步到 `0.5.10`。
- [ ] 重新运行完整本地验证，确认版本准备改动可发布。
- [ ] 提交版本准备改动后确认远端 CI passed。
- [ ] 打 tag `v0.5.10` 并推送。
- [ ] Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.10` 验证 10 个 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.10 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.10` 更新仓库内 Formula。
- [ ] 仓库内 Formula 和发布记录提交后确认远端 CI passed。
- [ ] 更新 Homebrew tap 到 `0.5.10` 并推送。
- [ ] 手动触发 Post-Release Verify，确认资产清单和五平台安装脚本 dry run 通过。

## 当前结论

v0.5.10 候选内容已经完成本地功能验证，最新远端 CI 也已通过；下一步补 release readiness 门禁，并决定是否进入正式版本准备。
