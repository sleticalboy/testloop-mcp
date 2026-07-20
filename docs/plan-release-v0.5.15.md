# v0.5.15 候选发布检查清单

## 当前目标

这是 v0.5.15 的候选发布检查清单。目标是把 v0.5.14 之后围绕 Agent 决策 fixture、manifest 驱动客户端契约、JSON validator、最小导出包和 release readiness 门禁的改动整理成一个可发布边界。

发布重点见 [v0.5.15 发布说明](./plan-release-notes-v0.5.15.md)。

当前发布状态：正式发布完成。版本号、CHANGELOG、安装/接入文档、tag、GitHub Release、五平台 Release assets、资产校验、仓库内 Formula 和 Homebrew tap 均已完成。

## 当前差异核对

- [x] 新增真实项目 Agent 闭环 fixture，覆盖 Go ready、Vitest ready、Python environment 手审和 Python external-service 失败手审。
- [x] 新增 `docs/fixtures/agent-decision-fixtures.json` 和 schema。
- [x] `examples/agent-decision-demo` 已改为读取 manifest。
- [x] 新增 `scripts/validate-agent-decision-fixtures.mjs`，支持文本和 `--json` 输出。
- [x] validator 已校验 manifest 元数据和关键 payload 结构。
- [x] 新增 `scripts/export-agent-decision-fixtures.mjs`，导出可复制最小 fixture 包。
- [x] 导出包包含无依赖 `package.json`，可运行 `npm test --silent`。
- [x] `scripts/verify-release-candidate.sh` 已显式校验 Agent 决策 fixture 导出包。
- [x] README、客户端集成说明、MCP 客户端契约测试说明和 roadmap 已同步。

## 候选内容

- [x] 客户端可以从 manifest 读取全部最小 Agent 决策 fixture，而不是维护 glob 或手写白名单。
- [x] 客户端可以用 JSON validator 机器断言 `status/action -> decision` 合同。
- [x] 客户端可以复制最小 fixture 包，并用 `npm test --silent` 加入自己的 CI。
- [x] release readiness 会覆盖导出包，不再只依赖普通 shell 子测试间接验证。
- [x] 文档明确 `failed/manual_review_*` 不进入自动修复循环。

## 已验证

- [x] `sh test/agent_decision_fixture_validator_test.sh`
- [x] `sh test/agent_decision_fixture_export_test.sh`
- [x] `sh test/agent_decision_fixtures_manifest_test.sh`
- [x] `sh test/agent_decision_demo_test.sh`
- [x] `sh test/client_integration_doc_test.sh`
- [x] `sh test/mcp_client_contract_doc_test.sh`
- [x] `sh test/release_candidate_script_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/ci_workflow_test.sh`
- [x] `go test ./...`
- [x] `git diff --check`
- [x] `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.15-candidate-dist scripts/verify-release-candidate.sh v0.5.15` 输出 `release_candidate_status=passed`，导出包 step 输出 `fixture_count=8`。
- [x] `da69566` 远端 CI run `29748774219` passed，覆盖 Agent 决策 validator JSON 输出。
- [x] `0868676` 远端 CI run `29749188084` passed，覆盖 Agent 决策 fixture 导出包。
- [x] `d79b720` 远端 CI run `29749432210` passed，覆盖导出包 `package.json` / `npm test --silent`。
- [x] `85ce335` 远端 CI run `29749842485` passed，覆盖 manifest 元数据校验。
- [x] `153574f` 远端 CI run `29750125793` passed，覆盖 release readiness 显式校验 Agent 决策 fixture 导出包。
- [x] `34f0954` 远端 CI run `29750391251` passed，覆盖 v0.5.15 候选边界文档。
- [x] `f37b382` 远端 CI run `29751381326` passed，覆盖 v0.5.15 正式版本准备。
- [x] Release Artifacts tag run `29756859746` passed，覆盖 Linux amd64、Linux arm64、macOS arm64、Windows amd64 和 Windows arm64 五个平台资产构建与上传。
- [x] `TESTLOOP_MCP_REPO=sleticalboy/testloop-mcp scripts/verify-release-assets.sh v0.5.15` 已验证正式 Release 的 10 个资产完整。
- [x] `ruby -c Formula/testloop-mcp.rb` 和 `sh test/release_assets_test.sh` 已验证仓库内 Homebrew Formula。
- [x] Homebrew tap 已更新到 `0.5.15` 并推送：tap commit `d72ab7d`。
- [x] Post-Release Verify run `29757718773` passed，覆盖资产清单和五个平台安装脚本 dry run。

## 发布前门禁

- [x] 候选边界整理后的 main CI 已通过：`34f0954` run `29750391251` passed。
- [x] `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.15-candidate-dist scripts/verify-release-candidate.sh v0.5.15`
- [x] `git diff --check`
- [x] 确认 `testloop-mcp --version` 已在正式版本准备后输出 `testloop-mcp 0.5.15`。

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.5.15`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.15 - 2026-07-20`。
- [x] 同步 README、installation、quickstart 和必要版本引用到 `0.5.15` / `v0.5.15`。
- [x] 测试中的版本期望同步到 `0.5.15`。
- [x] 重新运行完整本地验证，确认版本准备改动可发布：`scripts/verify-release-candidate.sh v0.5.15` 输出 `release_candidate_status=passed`，`testloop-mcp --version` 输出 `testloop-mcp 0.5.15`。
- [x] 提交版本准备改动后确认远端 CI passed：`f37b382` run `29751381326` passed。
- [x] 打 tag `v0.5.15` 并推送。
- [x] 等 Release Artifacts workflow 生成五平台资产和 `.sha256`：run `29756859746` passed。
- [x] 使用 `scripts/verify-release-assets.sh v0.5.15` 验证 Release 资产完整。
- [x] 更新 GitHub Release 正文为正式 v0.5.15 发布说明。
- [x] 使用 `scripts/generate-homebrew-formula.sh v0.5.15` 更新仓库内 Formula。
- [x] 更新 Homebrew tap 到 `0.5.15` 并推送：tap commit `d72ab7d`。
- [x] 手动触发 Post-Release Verify：run `29757718773` passed。

## 当前结论

v0.5.15 已完成正式发布、Release Artifacts、资产清单校验、GitHub Release 正文、仓库内 Formula、Homebrew tap 和 Post-Release Verify 五平台安装 dry run。发布收尾只剩提交并推送本发布记录更新，然后回到主线产品价值。
