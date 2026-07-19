# v0.5.11 发布检查清单

## 当前目标

这是 v0.5.11 的候选发布检查清单。当前目标是把 v0.5.10 之后围绕 Agent/客户端 CI artifact 消费契约的改动归档为一个 patch 版本。

发布重点见 [v0.5.11 发布说明草案](./plan-release-notes-v0.5.11.md)。

## 当前差异核对

- [x] Agent response artifact contract 已加入。
- [x] first-run/onboarding artifact manifest 已加入。
- [x] artifact manifest JSON Schema 已加入。
- [x] manifest 声明 `$schema`。
- [x] manifest demo 已加入，并覆盖 first-run/onboarding artifact fixture。
- [x] README 已加入 manifest demo 命令和最小正常输出。
- [x] 客户端契约测试说明已加入 manifest/schema 回归模板。
- [x] 接入方一页式验证指南已加入 artifact manifest/schema 验收入口。
- [x] quickstart 已加入 artifact manifest/schema 快速验证入口。
- [x] installation 已从安装后自检段落指向 artifact manifest/schema 消费回归。
- [x] fixture 维护规则已要求同步 manifest、schema、Go schema 测试、demo 输出断言和入口文档。
- [x] CHANGELOG 和 roadmap 已记录候选内容。

## 候选内容

- [x] `agent-response.txt` 的统一 contract。
- [x] CI artifact fixture 的机器可读 manifest。
- [x] artifact manifest v1 JSON Schema。
- [x] manifest 消费 demo。
- [x] 客户端/Agent 可复制的 manifest/schema 回归模板。
- [x] README、quickstart、installation 和一页式接入指南的入口同步。
- [x] manifest/schema 维护规则。

## 已验证

- [x] `sh test/agent_response_artifact_contract_doc_test.sh`
- [x] `sh test/agent_response_artifact_manifest_test.sh`
- [x] `sh test/agent_response_manifest_demo_test.sh`
- [x] `go test ./tools -run TestAgentResponseArtifactManifestSchema -count=1`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/mcp_client_contract_doc_test.sh`
- [x] `sh test/adopter_verification_guide_doc_test.sh`
- [x] `sh test/quickstart_doc_test.sh`
- [x] `sh test/installation_doc_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `git diff --check`
- [x] `da2efc9` 远端 CI run `29673942518` passed。
- [x] `a002aea` 远端 CI run `29674019795` passed。
- [x] `c1b72db` 远端 CI run `29674090490` passed。
- [x] `012ae16` 远端 CI run `29674150664` passed。
- [x] `2adcf2c` 远端 CI run `29674342374` passed。
- [x] `79fb125` 远端 CI run `29674445146` passed。
- [x] `8f9cd99` 远端 CI run `29674535626` passed。
- [x] `292c8bf` 远端 CI run `29674625675` passed。
- [x] `d0827c2` 远端 CI run `29674719845` passed。
- [x] `fcdda6f` 远端 CI run `29674820980` passed。
- [x] `d7d24da` 远端 CI run `29674914207` passed。
- [x] `7519cf2` 远端 CI run `29675007824` passed。
- [x] 发布说明草案提交 `e7ca8a1` 远端 CI run `29675124697` passed。

## 发布前门禁

- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `go build -o /tmp/testloop-mcp-v0.5.11-candidate .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.11-candidate ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.11-candidate --version` 输出 `testloop-mcp 0.5.10`，正式版本准备前不提前切版本号。
- [x] `/tmp/testloop-mcp-v0.5.11-candidate --help` 输出 `Usage of`，exit code 为 `2`。
- [x] `/tmp/testloop-testgen-v0.5.11-candidate --help` 输出 `Usage: testgen`，exit code 为 `2`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.11-candidate-dist scripts/package-release-asset.sh v0.5.11 darwin_arm64 darwin arm64`
- [x] 在 dist 目录内校验 `testloop-mcp_v0.5.11_darwin_arm64.tar.gz.sha256` 通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 正式发布前待办

- [ ] 更新 `main.go` MCP implementation version 到 `0.5.11`。
- [ ] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.11 - 2026-07-19`。
- [ ] 同步 README 中当前 Release、手动下载示例和 Windows 下载示例到 `v0.5.11`。
- [ ] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.5.11`。
- [ ] 同步 quickstart、first-run、verification CI、onboarding CI 和接入指南中的版本门禁到 `0.5.11`。
- [ ] 测试中的版本期望同步到 `0.5.11`。
- [ ] 重新运行完整本地验证，确认版本准备改动可发布。
- [ ] 提交版本准备改动后确认远端 CI passed。
- [ ] 打 tag `v0.5.11` 并推送。
- [ ] Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.11` 验证 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.11 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.11` 更新仓库内 Formula。
- [ ] 更新 Homebrew tap 到 `0.5.11` 并推送。
- [ ] 手动触发 Post-Release Verify，确认资产清单和五平台安装脚本 dry run 通过。

## 当前结论

v0.5.11 候选边界和 release readiness 预检已通过：这是一次 Agent/客户端 artifact 消费契约 patch。下一步可以进入正式版本准备，更新版本号与文档引用后重新跑完整发布前验证；在正式版本准备提交通过远端 CI 前不要打 tag。
