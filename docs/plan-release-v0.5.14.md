# v0.5.14 发布检查清单

## 当前目标

这是 v0.5.14 的候选发布检查清单。目标是把 v0.5.13 之后围绕 CI artifact 自检、manifest 批量校验、JSON 输出、summary schema 自包含、默认 CI 覆盖和真实项目证据的改动整理成一个可发布边界。

发布重点见 [v0.5.14 发布说明草案](./plan-release-notes-v0.5.14.md)。

当前发布状态：正式版本准备中。已更新版本号、切 CHANGELOG 正式段并同步当前安装/接入文档版本引用；尚未打 tag、生成正式 Release assets 或更新 Homebrew tap。

## 当前差异核对

- [x] Agent artifact fixture 已自包含 `verification-summary.schema.json`。
- [x] `agent-response-artifact-manifest.json` 已为每个 artifact 增加本地 `summary_schema` 指针。
- [x] 新增 `scripts/verify-agent-artifact.sh` 和 `examples/agent-artifact-verify`。
- [x] verifier 支持 first-run/onboarding 单目录校验。
- [x] verifier 支持 manifest 批量校验。
- [x] verifier 支持 `--json` 结构化输出。
- [x] first-run Agent response 已补齐 `first_run_status` 和 `first_run_failed_count`。
- [x] first-run/onboarding bootstrap 会在 helper 支持时自动运行 artifact verifier。
- [x] GitHub step summary 会记录 `Artifact verification`。
- [x] `testloop-mcp --help` 和 `testgen --help` 会以退出码 0 返回。
- [x] 新增 `scripts/verify-release-candidate.sh`，把本地 release readiness 门禁固化为维护者一键入口。
- [x] 默认 CI 显式运行全部 `test/*_test.sh`。
- [x] 仓库卫生测试已防止重新提交 ignored tracked 文件和 Python bytecode。
- [x] laoxia server/web 最新真实 bootstrap 已证明 `agent_artifact_status=passed`。
- [x] `CHANGELOG.md` 的 `Unreleased` 已收敛到 `v0.5.14 - 2026-07-20`。
- [x] `main.go` MCP implementation version 已更新到 `0.5.14`。
- [x] 当前安装、quickstart、first-run/onboarding/verification CI 文档和对应测试期望已同步到 `0.5.14` / `v0.5.14`。

## 候选内容

- [x] CI artifact 下载目录可离线自检。
- [x] first-run/onboarding artifact 可通过同一个 wrapper 验证。
- [x] manifest 可一条命令批量验证全部 Agent artifact fixture。
- [x] verifier 输出既支持人类可读文本，也支持客户端 JSON。
- [x] bootstrap 生成 artifact 后会自动自检，减少上传后才发现 artifact 不自洽的风险。
- [x] 真实项目案例记录证明 server/web 两类用户项目均可得到 `ready` 和 `Artifact verification=passed`。

## 已验证

- [x] `sh test/agent_artifact_verify_test.sh`
- [x] `sh test/agent_response_artifact_manifest_test.sh`
- [x] `sh test/agent_response_manifest_demo_test.sh`
- [x] `sh test/run_first_run_ci_test.sh`
- [x] `sh test/run_onboarding_ci_test.sh`
- [x] `sh test/real_integration_cases_doc_test.sh`
- [x] `sh test/repository_hygiene_test.sh`
- [x] `sh test/ci_workflow_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `for script in test/*_test.sh; do sh "$script"; done`
- [x] `go test ./...`
- [x] `git diff --check`
- [x] `6c7b742` 远端 CI run `29735364285` passed，覆盖 Agent artifact 目录校验入口。
- [x] `80d8030` 远端 CI run `29735864478` passed，覆盖 bootstrap 自动 artifact 自检。
- [x] `e9530c1` 远端 CI run `29736423222` passed，覆盖 manifest 批量校验。
- [x] `4764823` 远端 CI run `29736802986` passed，覆盖 verifier JSON 输出。
- [x] `c36758b` 远端 CI run `29737179225` passed，覆盖 laoxia artifact 自检复验证据。
- [x] `ab81926` 远端 CI run `29737938722` passed，覆盖 v0.5.14 候选发布边界文档。
- [x] `27a0410` 远端 CI run `29738560911` passed，覆盖 CLI help 退出码修复。
- [x] `7173228` 远端 CI run `29739075425` passed，覆盖候选发布门禁脚本。
- [ ] 最新 main CI 尚待最终确认；`b6ef1a8` 远端 CI run `29739151454` 仍在 GitHub Actions 队列中。

## 发布前门禁

- [ ] 等版本准备提交后的最新 main CI 通过。
- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `sh test/release_candidate_script_test.sh`
- [x] `go build -o /tmp/testloop-mcp-v0.5.14-candidate .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.14-candidate ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.14-candidate --version` 正式版本准备后输出 `testloop-mcp 0.5.14`。
- [x] `/tmp/testloop-mcp-v0.5.14-candidate --help`
- [x] `/tmp/testloop-testgen-v0.5.14-candidate --help`
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.14-candidate-dist scripts/package-release-asset.sh v0.5.14 darwin_arm64 darwin arm64`
- [x] 校验 darwin arm64 `.sha256` 和 tarball 内容，tarball 包含 `LICENSE`、`README.md`、`testloop-mcp` 和 `testloop-testgen`。
- [x] `scripts/verify-release-candidate.sh v0.5.14`
- [x] `git diff --check`

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.5.14`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.14 - 2026-07-20`。
- [x] 同步 README、installation、quickstart 和必要版本引用到 `0.5.14` / `v0.5.14`。
- [x] 测试中的版本期望同步到 `0.5.14`。
- [x] 重新运行完整本地验证，确认版本准备改动可发布：`scripts/verify-release-candidate.sh v0.5.14` 输出 `release_candidate_status=passed`，`testloop-mcp --version` 输出 `testloop-mcp 0.5.14`。
- [ ] 提交版本准备改动后确认远端 CI passed。
- [ ] 打 tag `v0.5.14` 并推送。
- [ ] Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.14` 验证 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.14 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.14` 更新仓库内 Formula。
- [ ] 更新 Homebrew tap 到 `0.5.14` 并推送。
- [ ] 手动触发 Post-Release Verify。

## 当前结论

v0.5.14 已完成正式版本准备的本地验证，但还不是正式发布状态。下一步提交版本准备；提交后的 main CI 通过后，才进入 tag、Release assets、GitHub Release、Homebrew tap 和 Post-Release Verify。
