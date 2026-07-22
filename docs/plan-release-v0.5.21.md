# v0.5.21 发布检查清单

## 当前目标

这是 v0.5.21 的候选发布检查清单。目标是把 v0.5.20 之后围绕 release response 接入样板、artifact 打包、离线自检、JSON 契约和客户端消费 demo 的改动整理成一个可发布边界。

发布重点见 [v0.5.21 发布说明草案](./plan-release-notes-v0.5.21.md)。

当前发布状态：已进入正式版本准备。`main.go` implementation version 已更新为 `0.5.21`，`CHANGELOG.md` 已收敛到 `v0.5.21 - 2026-07-22` 并保留空 Unreleased；尚未打 `v0.5.21` tag，尚未创建 GitHub Release，尚未更新 Homebrew tap。

## 当前差异核对

- [x] 新增 release response 接入样板、临时外部仓库 showcase 和两个 consumer helper。
- [x] 新增 release response 接入样板 summary schema、通过态 fixture、失败态 fixture 和 validator。
- [x] 接入样板 showcase 会生成 `testloop-release-response-adopter-artifacts/`，用于外部 CI 上传 evidence。
- [x] 新增 artifact 离线自检 verifier，并纳入 release readiness。
- [x] artifact verifier 失败时固定分流到 `inspect-release-response-adopter-artifact`，避免误用旧 `ready`。
- [x] 新增 artifact verification JSON schema、通过态/失败态 fixture 和 validator。
- [x] 新增 artifact verification 客户端消费 demo，展示 `accept` 与 `inspect-artifact` 决策。
- [x] README、client integration、release response checklist、release response 客户端文档、fixtures 索引、CHANGELOG 和 roadmap 已同步。

## 候选内容

- [x] 接入方可以运行 `scripts/showcase-release-response-adopter.sh --json` 生成可上传 artifact。
- [x] 接入方可以运行 `node scripts/validate-release-response-adopter-summary.mjs /path/to/summary.json` 校验 adopter summary。
- [x] 接入方可以运行 `node scripts/verify-release-response-adopter-artifact.mjs --json /path/to/testloop-release-response-adopter-artifacts` 离线自检下载后的 artifact。
- [x] 接入方可以运行 `node scripts/validate-release-response-adopter-artifact-verification.mjs /path/to/verification.json` 固定 verifier JSON 契约。
- [x] 接入方可以运行 `go run ./examples/release-response-adopter-artifact-demo /path/to/verification.json` 把 verification JSON 映射成客户端决策。
- [x] 当前版本边界明确：不扩语言、不改测试生成算法，聚焦外部客户端/Agent 的 release response evidence 消费合同。

## 已验证

- [x] `sh test/release_response_adopter_example_test.sh`
- [x] `sh test/release_response_adopter_summary_schema_test.sh`
- [x] `sh test/release_response_adopter_summary_validator_test.sh`
- [x] `sh test/release_response_adopter_artifact_verify_test.sh`
- [x] `sh test/release_response_adopter_artifact_verification_schema_test.sh`
- [x] `sh test/release_response_adopter_artifact_verification_validator_test.sh`
- [x] `sh test/release_response_adopter_artifact_demo_test.sh`
- [x] `sh test/agent_decision_release_response_checklist_doc_test.sh`
- [x] `sh test/agent_decision_release_response_client_doc_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/fixtures_index_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/readme_ci_snippet_test.sh`
- [x] `sh test/ci_workflow_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/release_candidate_script_test.sh`
- [x] `go test ./...`
- [x] `git diff --check`
- [x] `cd44986` 远端 CI run `29893990689` passed，覆盖 artifact verification 客户端消费 demo。
- [x] `61e4e19` 远端 CI run `29894128846` passed，覆盖 artifact 消费 demo CI 记录。
- [x] `91c3498` 远端 CI run `29894309974` passed，覆盖 artifact 消费 demo 记录。
- [x] `5b197ed` 远端 CI run `29894452504` passed，覆盖 artifact 消费 demo 记录再验证。
- [x] `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.21-release-prep-dist scripts/verify-release-candidate.sh v0.5.21` 输出 `release_candidate_status=passed`。

## 发布前门禁

- [ ] 正式版本准备后的最新 main CI passed。
- [x] 正式版本准备后的本地 release readiness passed：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.21-release-prep-dist scripts/verify-release-candidate.sh v0.5.21`。
- [x] readiness 输出包含 release response adopter summary 校验：`release_response_adopter_summary_status=passed release_ref=v0.5.21`。
- [x] readiness 输出包含 release response adopter artifact 自检：`release_response_adopter_artifact_status=passed`。
- [x] readiness 输出包含 artifact verification validator：`release_response_adopter_artifact_verification_status=passed release_ref=v0.5.21`。
- [x] readiness 输出包含候选二进制版本：`testloop-mcp 0.5.21`。
- [x] readiness 输出包含 darwin arm64 打包 dry-run 和 sha256 校验：`testloop-mcp_v0.5.21_darwin_arm64.tar.gz: OK`。

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.5.21`。
- [x] 将 `CHANGELOG.md` 的 Unreleased 内容收敛到 `v0.5.21 - 2026-07-22`，并保留新的空 Unreleased。
- [x] 同步 README、installation、quickstart 和必要版本引用到 `0.5.21` / `v0.5.21`。
- [x] 测试中的版本期望同步到 `0.5.21`。
- [x] 重新运行完整 release readiness。
- [ ] 提交版本准备改动后确认远端 CI passed。
- [ ] 打 tag `v0.5.21` 并推送。
- [ ] 等 Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.21` 验证 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.21 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.21` 更新仓库内 Formula。
- [ ] 更新 Homebrew tap 到 `0.5.21`。
- [ ] Post-Release Verify。
- [ ] 发布后运行 release response adopter artifact smoke。

## 当前结论

v0.5.21 已进入正式版本准备：版本号、changelog、用户文档和测试期望已同步到 `0.5.21` / `v0.5.21`，完整本地 release readiness 已通过。下一步应提交版本准备改动并等待 main CI。
