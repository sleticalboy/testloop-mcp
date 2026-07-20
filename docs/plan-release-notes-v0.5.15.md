# v0.5.15 发布说明

## 标题

testloop-mcp v0.5.15

## 发布状态

- [x] 创建 v0.5.15 候选发布说明草案。
- [x] 梳理 v0.5.14 之后围绕真实项目 Agent 决策 fixture、manifest 驱动客户端契约、JSON validator、最小导出包和 release readiness 门禁的改动边界。
- [x] 正式版本准备文件已更新：implementation version、`CHANGELOG.md` 正式版本段和当前安装/接入文档版本引用已同步到 `0.5.15` / `v0.5.15`。
- [x] 完整本地 dry-run 门禁已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.15-candidate-dist scripts/verify-release-candidate.sh v0.5.15` 输出 `release_candidate_status=passed`。
- [x] 最新已完成的远端 CI：`153574f` run `29750125793` passed，覆盖 release readiness 显式校验 Agent 决策 fixture 导出包。
- [x] 候选边界整理提交 `34f0954` 的远端 CI run `29750391251` passed。
- [x] 版本准备后的完整本地门禁已通过：`scripts/verify-release-candidate.sh v0.5.15` 输出 `release_candidate_status=passed`，`testloop-mcp --version` 输出 `testloop-mcp 0.5.15`。
- [x] 版本准备提交 `f37b382` 的远端 CI run `29751381326` passed。
- [x] `v0.5.15` tag 已推送，Release Artifacts run `29756859746` passed，五个平台 10 个资产已上传。
- [x] `TESTLOOP_MCP_REPO=sleticalboy/testloop-mcp scripts/verify-release-assets.sh v0.5.15` 已验证正式 Release 资产完整。
- [x] GitHub Release 正文已更新为正式 v0.5.15 发布说明。
- [x] 仓库内 Homebrew Formula 已用正式 Release asset digest 更新到 `0.5.15`。
- [x] Homebrew tap 已更新到 `0.5.15` 并推送，tap commit `d72ab7d`。
- [x] Post-Release Verify run `29757718773` passed，覆盖资产清单和五个平台安装脚本 dry run。

## 摘要

v0.5.15 继续围绕“面向 AI 编程代理的测试反馈闭环 MCP 服务”收敛，不把重点放在扩语言或包装成泛化测试生成器。

这轮改动把 `validate_coverage_task` 的 Agent 决策样本从文档示例推进到外部客户端可复制、可机器断言、可进入发布门禁的 fixture 包。接入方可以直接用 manifest 和 validator 校验 `status/action -> decision` 合同，避免自己维护隐含文件顺序、硬编码单个样例，或把 `failed/manual_review_*` 误当成自动修复任务。

## 主要变化

### Agent 决策 fixture manifest

- 新增 `docs/fixtures/agent-decision-fixtures.json` 和 `docs/fixtures/agent-decision-fixtures.schema.json`。
- manifest 固定 8 个最小决策样本，覆盖：
  - `passed/ready -> accept`
  - `passed/manual_review_internal -> manual-review`
  - `passed/manual_review_environment -> manual-review`
  - `failed/manual_review_external_service -> manual-review`
  - `failed/apply_fix_suggestions -> apply-repair`
  - `failed/needs_better_input -> needs-better-input`
- `examples/agent-decision-demo` 改为读取同一份 manifest，避免 demo 和文档 fixture 清单分叉。

### 真实项目 Agent 闭环证据

- 新增 laoxia Go server `utils` 包真实 coverage task fixture，证明真实 Go 项目可稳定得到 `passed/ready`。
- 新增 mcp-hub Vitest 历史 repair 回归 fixture，防止 async throwing branch 从 `passed/ready` 回退到普通 repair。
- 新增 haoy-apk-station FastAPI `manual_review_environment` fixture，证明环境依赖场景会被稳定分流到手审。
- 新增 haoy-apk-station FastAPI `failed/manual_review_external_service` fixture，明确外部对象存储 timeout 不应进入自动修生成测试循环。

### 客户端 validator 和导出包

- 新增 `scripts/validate-agent-decision-fixtures.mjs`，无第三方依赖校验 manifest 和全部决策 fixture。
- validator 支持默认文本输出和 `--json` 输出；失败时仍输出可解析 JSON，并通过非 0 退出码让 CI 失败。
- validator 不依赖 JSON Schema 工具链，也会检查 manifest 条目的 `kind`、`source`、`status`、`action`、`expected_decision` 和 `client_expectation`。
- 新增 `scripts/export-agent-decision-fixtures.mjs`，可导出最小 Agent 决策 fixture 包。
- 导出包包含 manifest、schema、8 个 fixture、validator、最小 README 和无依赖 `package.json`，复制到客户端仓库后可直接运行 `npm test --silent`。

### 发布门禁

- `scripts/verify-release-candidate.sh` 新增显式 `verify agent decision fixture export package` 步骤。
- release readiness 会导出最小 Agent 决策 fixture 包，并在导出目录运行 `npm test --silent`。
- release readiness 提前检查 `node` 和 `npm`，减少候选发布时的隐式环境失败。

## 质量边界

- 这轮提升的是 Agent/客户端消费确定性，不承诺测试生成算法本身大幅提升。
- 当前真实 `failed/apply_fix_suggestions` 仍没有比现有合成 fixture 更稳定的真实项目来源；短期不强行提交漂移样本。
- `failed` 不等于自动修复。只有 `failed/apply_fix_suggestions` 进入 repair task 闭环；`failed/manual_review_external_service` 等 `manual_review_*` 应转 fake client、依赖注入或集成环境验证。
- Rust/Java 覆盖率扩展不属于本候选范围。

## 推荐验证

- `sh test/agent_decision_fixtures_manifest_test.sh`
- `sh test/agent_decision_fixture_validator_test.sh`
- `sh test/agent_decision_fixture_export_test.sh`
- `sh test/agent_decision_demo_test.sh`
- `sh test/client_integration_doc_test.sh`
- `sh test/mcp_client_contract_doc_test.sh`
- `sh test/release_candidate_script_test.sh`
- `sh test/release_doc_index_test.sh`
- `sh test/docs_links_test.sh`
- `go test ./...`
- `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.15-candidate-dist scripts/verify-release-candidate.sh v0.5.15`
- `git diff --check`

## 发布备注

- 对外文案应强调“Agent 决策 fixture 可复制、可 JSON 校验、可进入客户端 CI”，而不是“多语言测试生成能力增强”。
- 推荐演示路径：`go run ./examples/mcp-client-demo` 展示最小失败修复闭环，再运行 `node scripts/export-agent-decision-fixtures.mjs /tmp/testloop-agent-decision-fixtures` 和导出包内的 `npm test --silent` 展示客户端契约回归。
- v0.5.15 已完成正式 GitHub Release、Release assets、资产校验、仓库内 Formula、Homebrew tap 和 Post-Release Verify。
- 发布后验证证据：Release Artifacts run `29756859746`、资产清单校验、tap commit `d72ab7d`、Post-Release Verify run `29757718773` 均已通过。
