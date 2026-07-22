# v0.5.21 发布说明草案

## 标题

testloop-mcp v0.5.21

## 发布状态

- [x] 创建 v0.5.21 候选发布说明草案。
- [x] 梳理 v0.5.20 之后围绕 release response 接入样板、artifact 打包、离线自检、JSON 契约和客户端消费 demo 的改动边界。
- [ ] 完成候选 release readiness。
- [ ] 进入正式版本准备：更新 implementation version，收敛 `CHANGELOG.md`，同步版本引用。
- [ ] 完成正式版本准备后的 release readiness 和远端 CI。
- [ ] 打 tag、发布 GitHub Release、更新 Homebrew tap 并完成发布后 smoke。

## 摘要

v0.5.21 候选边界继续围绕项目定位推进：面向 AI 编程代理的测试反馈闭环 MCP 服务。

这个候选版本不扩语言，也不把卖点转回“自动生成单测”。重点是把 v0.5.20 的 release response 接入能力继续向外部客户端落地：接入方可以生成一份可上传的 evidence artifact，下载后离线自检，再把 verifier 的稳定 JSON 输出纳入自己的客户端单元测试或 CI 分流。

对 Agent 来说，这个版本的价值是减少发布后验证链路里的猜测：artifact 缺文件、JSON 不完整、summary 与 consumer 输出不一致时，Agent 会停在 `inspect-release-response-adopter-artifact`，而不是误读内部 summary 的旧 `ready` 状态。

## 主要变化

### release response 接入样板

- 新增 `examples/release-response-adopter/`。
- 新增 `scripts/showcase-release-response-adopter.sh --json`，可创建临时接入方仓库并跑通 release response 安装、renderer 和 consumer helper。
- 新增 `read-testloop-release-response.mjs` 和 `read-testloop-release-response-summary.mjs`，固定接入方读取 release response JSON 与 adopter summary 的最小消费方式。
- 接入样板 README 记录 helper 输出字段、CI artifact 清单和离线排查入口。

### artifact 打包与离线自检

- `scripts/showcase-release-response-adopter.sh --json` 现在会生成 `testloop-release-response-adopter-artifacts/`。
- artifact 目录包含 adopter summary、install summary、release response client 输出、release smoke summary、release response JSON 和两个 consumer helper 输出。
- 新增 `scripts/verify-release-response-adopter-artifact.mjs`，可在下载后的 artifact 目录上离线校验 6 个必备文件、JSON 可解析性、`release_ref`、`fixture_count`、`agent_next_step` 和 `should_accept`。
- release readiness 已纳入 artifact 自检。

### artifact verification JSON 契约

- `scripts/verify-release-response-adopter-artifact.mjs --json` 输出新增 `schema_version=1`。
- 新增 `docs/fixtures/release-response-adopter-artifact-verification.schema.json`。
- 新增通过态 fixture `docs/fixtures/release-response-adopter-artifact-verification/passed.json`。
- 新增失败态 fixture `docs/fixtures/release-response-adopter-artifact-verification/missing-summary-consumer.json`。
- 新增 `scripts/validate-release-response-adopter-artifact-verification.mjs`，用于接入方 CI 固定 verifier JSON 契约。

### 客户端消费 demo

- 新增 `examples/release-response-adopter-artifact-demo`。
- 通过态 artifact verification JSON 会映射为 `client_decision=accept`。
- 缺文件失败态会映射为 `client_decision=inspect-artifact`，并输出 `missing_files` 和 `failures`。
- README、客户端集成说明、release response checklist、release response 客户端文档和 fixtures 索引已同步 demo、schema、validator 与 artifact 自检入口。

## 质量边界

- `agent_next_step=ready` 只表示 release response 接入样板和 artifact verification 契约通过，不代表用户业务测试已经通过。
- artifact verifier 校验的是下载后的 evidence 包完整性和自洽性，不替代 release asset 安装校验或用户项目 smoke。
- 当前候选不改 `generate_tests` 生成策略，也不新增语言支持。
- 客户端应优先读取 verifier JSON 和 `client_decision`，不要解析 shell 日志文本。

## 推荐验证

- `sh test/release_response_adopter_example_test.sh`
- `sh test/release_response_adopter_summary_schema_test.sh`
- `sh test/release_response_adopter_summary_validator_test.sh`
- `sh test/release_response_adopter_artifact_verify_test.sh`
- `sh test/release_response_adopter_artifact_verification_schema_test.sh`
- `sh test/release_response_adopter_artifact_verification_validator_test.sh`
- `sh test/release_response_adopter_artifact_demo_test.sh`
- `sh test/agent_decision_release_response_checklist_doc_test.sh`
- `sh test/agent_decision_release_response_client_doc_test.sh`
- `sh test/client_integration_doc_test.sh`
- `sh test/fixtures_index_test.sh`
- `sh test/release_doc_index_test.sh`
- `sh test/readme_ci_snippet_test.sh`
- `sh test/ci_workflow_test.sh`
- `sh test/docs_links_test.sh`
- `sh test/release_candidate_script_test.sh`
- `go test ./...`
- `git diff --check`
- `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.21-release-prep-dist scripts/verify-release-candidate.sh v0.5.21`

## 发布备注

- 对外文案应强调：v0.5.21 让外部客户端不仅能接入 release response，还能上传、下载、自检和消费 release response evidence artifact。
- 推荐演示路径：运行 `scripts/showcase-release-response-adopter.sh --json`，上传 `testloop-release-response-adopter-artifacts/`，下载后运行 `node scripts/verify-release-response-adopter-artifact.mjs --json /path/to/artifact`，再用 `go run ./examples/release-response-adopter-artifact-demo /path/to/verification.json` 查看客户端决策。
- 这个版本仍然服务于 Agent 测试反馈闭环，不应宣传成通用多语言测试生成器。
