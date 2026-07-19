# v0.5.9 发布检查清单

## 当前目标

这是 v0.5.9 的候选发布和正式版本准备记录。当前已完成正式版本准备的版本同步；tag、Release Artifacts 和 Homebrew tap 仍需等版本准备提交的远端 CI 通过后再推进。

v0.5.9 发布重点见 [v0.5.9 发布说明草案](./plan-release-notes-v0.5.9.md)：本轮主要是 first-run artifact Agent 消费 demo、端到端回归、失败 artifact fixture 包、客户端集成文档和 README 入口。

## 当前差异核对

- [x] `examples/first-run-agent-response-demo` 已加入。
- [x] `docs/first-run-agent-artifact-demo.md` 已加入。
- [x] `test/first_run_agent_response_demo_test.sh` 已覆盖静态 fixture 和 first-run 真实失败五件套到 demo 输出。
- [x] `docs/fixtures/first-run-artifacts/user-project-smoke-failed/` 已加入完整五件套 fixture。
- [x] `test/first_run_artifact_fixtures_test.sh` 已加入并纳入 shell 矩阵。
- [x] `docs/fixtures.md` 已索引 first-run artifact fixture。
- [x] `docs/client-integration.md` 已区分 MCP tool 结构化返回 fixture 和 CI artifact fixture。
- [x] README 和 release doc index 已补 first-run artifact demo、demo 命令和 fixture 路径。
- [x] CHANGELOG 和 roadmap 已记录候选内容。

## 候选内容

- [x] first-run artifact Agent 消费 demo。
- [x] first-run 失败 artifact 到 Agent 回复 demo 的端到端回归。
- [x] first-run 失败 artifact fixture 五件套。
- [x] 客户端集成文档中的 CI artifact fixture 消费路径。
- [x] README / release doc index 可发现入口。

## 已验证

- [x] `sh test/first_run_agent_response_demo_test.sh`
- [x] `sh test/first_run_artifact_fixtures_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `git diff --check`
- [x] 候选提交远端 CI run `29670477128` passed。
- [x] 正式版本准备本地验证已通过：脚本语法、`go test ./...`、完整 shell 矩阵、主服务/testgen 构建、`--version`、help 输出、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。

## 发布前门禁

- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `go build -o /tmp/testloop-mcp-v0.5.9-candidate .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.9-candidate ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.9-candidate --help` 输出 usage，exit code 为 `2`。
- [x] `/tmp/testloop-testgen-v0.5.9-candidate --help` 输出 usage，exit code 为 `2`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.9-candidate-dist scripts/package-release-asset.sh v0.5.9 darwin_arm64 darwin arm64`
- [x] 在 dist 目录内校验 `testloop-mcp_v0.5.9_darwin_arm64.tar.gz.sha256` 通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.5.9`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.9 - 2026-07-19`。
- [x] 同步 README 中当前 Release、手动下载示例和 Windows 下载示例到 `v0.5.9`。
- [x] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.5.9`。
- [x] 同步 quickstart、first-run、verification CI、onboarding CI 和接入指南中的版本门禁到 `0.5.9`。
- [x] 测试中的版本期望同步到 `0.5.9`。
- [x] 重新运行完整本地验证，确认版本准备改动可发布。
- [ ] 提交版本准备改动后确认远端 CI passed。
- [ ] 打 tag `v0.5.9` 并推送。
- [ ] 等待 Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.9` 验证 10 个 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.9 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.9` 更新仓库内 Formula。
- [ ] 更新 Homebrew tap 到 `0.5.9` 并推送。
- [ ] 手动触发 Post-Release Verify，确认资产清单和五平台安装脚本 dry run 通过。

## 当前结论

v0.5.9 正式版本准备改动和本地完整验证已完成，适合作为 Agent artifact 消费体验 patch。下一步提交版本准备改动并等待远端 CI，通过后再进入 tag、Release Artifacts、GitHub Release 和 Homebrew tap 流程。
