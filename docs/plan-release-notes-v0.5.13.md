# v0.5.13 发布说明草案

## 标题

testloop-mcp v0.5.13

## 发布状态

- [x] 创建 v0.5.13 发布说明草案。
- [x] 梳理 v0.5.12 之后围绕 `action` 信号、verification summary schema、Agent response artifact manifest 和客户端消费回归的改动边界。
- [x] 最近本地验证已通过：artifact fixture、manifest demo、客户端文档测试、文档 gate、`go test ./...` 和 `git diff --check`。
- [x] 最近远端 CI 已通过：`515fd00` run `29695649219` passed。
- [x] 候选 release readiness 已通过：shell 语法、`go test ./...`、全部 `test/*_test.sh`、候选二进制构建、help/version、darwin arm64 打包 dry-run、sha256 和 tarball 内容检查。
- [x] 候选发布边界提交 `1395715` 远端 CI run `29695820104` passed。
- [x] 正式版本准备已更新 `main.go` MCP implementation version 到 `0.5.13`。
- [x] 正式版本准备已将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.5.13 - 2026-07-20`。
- [x] 正式版本准备已同步 README、安装文档和必要版本引用到 `v0.5.13`。
- [x] 正式版本准备 release readiness 已通过：shell 语法、`go test ./...`、全部 shell 回归、候选二进制构建、help/version、darwin arm64 打包 dry-run、sha256 和 tarball 内容检查。
- [x] 已打 `v0.5.13` tag 并推送。
- [x] Release Artifacts workflow_dispatch run `29710581315` 已通过，五个平台 10 个资产已上传。
- [x] `scripts/verify-release-assets.sh v0.5.13` 已验证 10 个 Release 资产完整。
- [x] GitHub Release 正文已更新为正式发布说明。
- [x] 仓库内 Formula 已更新到 `0.5.13`，`ruby -c`、`brew style` 和 release asset 测试通过。
- [x] Homebrew tap 已更新到 `0.5.13`：tap commit `0cb590e`。
- [x] Post-Release Verify run `29711047138` 已通过，覆盖资产清单和五个平台安装校验。

## 摘要

v0.5.13 候选重点是把 Agent 可消费的动作信号从单个 MCP tool 输出扩展到 CLI、验收报告、summary JSON、Agent 回复和 CI artifact manifest。

这个版本仍然不扩语言，也不把定位转回“自动生成测试”。核心价值是让 Codex、Claude Code、Cursor 这类 Agent 能稳定判断下一步：测试草稿是否可直接吸收、是否需要人工补输入/断言、失败是否应该读取修复建议，还是应该先排查用户项目 smoke。

## 主要变化

### 生成与执行动作信号

- `generate_tests` 返回可选 `action` 字段，区分 `ready` 和 `manual_review`。
- `run_tests` / `parse_results` 返回 `action`，把 passed/skipped/failed 场景映射到 `ready`、`manual_review`、`apply_fix_suggestions`、`inspect_failures` 或 `inspect_test_runner`。
- `cmd/testgen` 成功输出新增 `action=ready|manual_review`，CLI 用户无需先跑测试也能识别 TODO/skipped 草稿。

### 修复建议分类

- `fix_suggestions` 细分 `module_resolution`、`python_import_error` 和 `compile_error`。
- JS/TS 模块解析失败、Python import error 和 Go 编译失败都已有 `run_tests include_fix_suggestions=true` 回归。
- `examples/mcp-client-demo` 展示 `run_tests.action -> fix_suggestions.category -> repair_task -> rerun.action` 的最小消费路径。

### 验收报告与 summary schema

- `scripts/generate-verification-report.sh` 新增“独立 CLI 生成动作 smoke”章节，确认静态生成 TODO/skipped 草稿会输出 `action=manual_review`。
- `verification-summary.json` 支持 `sections[].signals.action`，并通过 `docs/fixtures/verification-summary.schema.json` 固化结构。
- verification summary decision、first-run Agent response 和 onboarding Agent response demo 都会展示 `section_signal=<section> action=<action>`。

### CI artifact manifest

- `agent-response-artifact-manifest.json` 新增 `summary_schema`，客户端只读取 manifest 就能发现 summary JSON 契约。
- manifest 新增 `expected_section_signals`，固定 fixture 中必须保留的 section/action 组合。
- `examples/agent-response-manifest-demo` 会验证 artifact 必备文件、`agent-response.txt` 字段、`agent-decision.txt` 的 `decision_action`、summary schema 文件和 expected section signals。

## 质量边界

- `manual_review` 是动作信号，不等于整体失败；整体状态仍以 `overall_status` / `failed_count` 为准。
- `section_signal=独立 CLI 生成动作 smoke action=manual_review` 只表示生成草稿需要人工补输入/断言，不代表用户项目 smoke 失败。
- JSON Schema 固定的是 artifact/summary 的稳定字段，不承诺覆盖 Markdown 报告的完整格式。
- 当前候选聚焦 Agent 消费闭环，不引入新的 LLM provider 或测试生成算法承诺。

## 推荐验证

- `sh test/agent_response_manifest_demo_test.sh`
- `sh test/agent_response_artifact_manifest_test.sh`
- `sh test/first_run_artifact_fixtures_test.sh`
- `sh test/onboarding_artifact_fixtures_test.sh`
- `go test ./tools -run 'TestAgentResponseArtifactManifestSchema|TestVerificationSummarySchema' -count=1`
- `sh test/client_integration_doc_test.sh`
- `sh test/mcp_client_contract_doc_test.sh`
- `sh test/readme_ci_snippet_test.sh`
- `sh test/docs_links_test.sh`
- `sh test/release_doc_index_test.sh`
- `go test ./...`
- `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- `go build -o /tmp/testloop-mcp-v0.5.13-candidate .`
- `go build -o /tmp/testloop-testgen-v0.5.13-candidate ./cmd/testgen`
- `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.13-candidate-dist scripts/package-release-asset.sh v0.5.13 darwin_arm64 darwin arm64`
- `go build -o /tmp/testloop-mcp-v0.5.13-release-prep .`
- `go build -o /tmp/testloop-testgen-v0.5.13-release-prep ./cmd/testgen`
- `/tmp/testloop-mcp-v0.5.13-release-prep --version` 输出 `testloop-mcp 0.5.13`
- `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.13-release-prep-dist scripts/package-release-asset.sh v0.5.13 darwin_arm64 darwin arm64`
- `TESTLOOP_MCP_REPO=sleticalboy/testloop-mcp scripts/verify-release-assets.sh v0.5.13`
- `ruby -c Formula/testloop-mcp.rb`
- `brew style Formula/testloop-mcp.rb`
- `sh test/release_assets_test.sh`
- `brew info --json=v2 sleticalboy/tap/testloop-mcp` 显示 stable `0.5.13`
- `brew audit --formula --strict sleticalboy/tap/testloop-mcp`
- Post-Release Verify run `29711047138`
- `git diff --check`

## 发布备注

- 对外文案应突出“Agent 测试反馈闭环的机器可读契约更稳定”。
- 不要宣传成测试生成质量大幅提升；本轮提升的是动作分流、失败分类和 artifact 消费确定性。
- 推荐演示路径：先跑 `examples/mcp-client-demo`，再跑 `examples/agent-response-manifest-demo`，展示 MCP tool 结果和 CI artifact 结果都能被 Agent 稳定消费。
