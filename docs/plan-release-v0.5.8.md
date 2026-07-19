# v0.5.8 发布检查清单

## 当前目标

这是 v0.5.8 的候选发布和正式版本准备记录。当前已完成正式版本准备的本地验证；tag、Release Artifacts 和 Homebrew tap 仍需等版本准备提交的远端 CI 通过后再推进。

v0.5.8 发布重点见 [v0.5.8 发布说明草案](./plan-release-notes-v0.5.8.md)：本轮主要是接入方复制路径、README 最小 CI、CI 失败后 Agent triage、Agent 回复格式，以及安装 checksum fallback 修复。

## 当前差异核对

- [x] `docs/adopter-verification-guide.md` 已加入并在 README / showcase / release doc index 中可达。
- [x] `docs/real-integration-cases.md` 已更新为 v0.5.7 first-run / onboarding 真实复验记录。
- [x] README 已新增“用户项目接入：直接复制”入口。
- [x] README 已新增最小 GitHub Actions first-run workflow 片段。
- [x] `test/readme_ci_snippet_test.sh` 已加入并纳入 shell 矩阵。
- [x] `docs/ci-agent-triage.md` 已加入，并包含失败态实跑记录。
- [x] `test/ci_agent_triage_doc_test.sh` 已加入并纳入 shell 矩阵。
- [x] `docs/first-run-agent-response.md` 已加入。
- [x] `test/first_run_agent_response_doc_test.sh` 已加入并纳入 shell 矩阵。
- [x] `scripts/install.sh` checksum fallback 已修复。
- [x] `test/install_script_test.sh` 已覆盖 `checksums.txt` 存在但缺当前资产时回退单资产 `.sha256` 的场景。
- [x] CHANGELOG 和 roadmap 已记录候选内容。

## 候选内容

- [x] 接入方一页式验证指南。
- [x] laoxia Go server / Vue web 真实接入复验。
- [x] README first-run / onboarding bootstrap 直达入口。
- [x] README 最小 first-run GitHub Actions workflow。
- [x] CI 失败后交给 Agent 的最短 triage 文档。
- [x] first-run Agent 回复格式。
- [x] 安装脚本 checksum fallback 修复。

## 已验证

- [x] `sh test/ci_agent_triage_doc_test.sh`
- [x] `sh test/first_run_agent_response_doc_test.sh`
- [x] `sh test/install_script_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `git diff --check`
- [x] 候选提交远端 CI run `29668904180` passed。

## 正式版本准备验证

- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `go build -o /tmp/testloop-mcp-v0.5.8-prep .`
- [x] `/tmp/testloop-mcp-v0.5.8-prep --version` 输出 `testloop-mcp 0.5.8`。
- [x] `/tmp/testloop-mcp-v0.5.8-prep --help` 输出 usage，exit code 为 `2`。
- [x] `go build -o /tmp/testloop-testgen-v0.5.8-prep ./cmd/testgen`
- [x] `/tmp/testloop-testgen-v0.5.8-prep --help` 输出 usage，exit code 为 `2`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.8-prep-dist scripts/package-release-asset.sh v0.5.8 darwin_arm64 darwin arm64`
- [x] 在 dist 目录内校验 `testloop-mcp_v0.5.8_darwin_arm64.tar.gz.sha256` 通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 发布前门禁

- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `go build -o /tmp/testloop-mcp-v0.5.8-candidate .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.8-candidate ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.8-candidate --help` 输出 usage，exit code 为 `2`。
- [x] `/tmp/testloop-testgen-v0.5.8-candidate --help` 输出 usage，exit code 为 `2`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.8-candidate-dist scripts/package-release-asset.sh v0.5.8 darwin_arm64 darwin arm64`
- [x] 在 dist 目录内校验 `testloop-mcp_v0.5.8_darwin_arm64.tar.gz.sha256` 通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.5.8`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.8 - 2026-07-19`。
- [x] 同步 README 中当前 Release、手动下载示例和 Windows 下载示例到 `v0.5.8`。
- [x] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.5.8`。
- [x] 同步 quickstart、first-run、verification CI、onboarding CI 和接入指南中的版本门禁到 `0.5.8`。
- [x] 测试中的版本期望同步到 `0.5.8`。
- [x] 重新运行完整验证。
- [ ] 提交版本准备改动后确认远端 CI passed。
- [ ] 打 tag `v0.5.8` 并推送。
- [ ] 等待 Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.8` 验证 10 个 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.8 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.8` 更新仓库内 Formula。
- [ ] 更新 Homebrew tap 到 `0.5.8` 并推送。
- [ ] 手动触发 Post-Release Verify，确认资产清单和五平台安装脚本 dry run 通过。

## 当前结论

v0.5.8 正式版本准备改动和本地完整验证已完成，适合作为接入体验和安装 fallback patch。下一步需提交版本准备改动并等待远端 CI，通过后再进入 tag、Release Artifacts、GitHub Release 和 Homebrew tap 流程。
