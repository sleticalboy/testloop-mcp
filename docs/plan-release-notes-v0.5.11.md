# v0.5.11 发布说明草案

## 标题

testloop-mcp v0.5.11

## 发布状态

- [x] 创建 v0.5.11 发布说明草案。
- [x] 梳理 v0.5.10 之后的 Agent response artifact contract、manifest/schema、demo 和接入文档回归改动。
- [x] `da2efc9` 远端 CI run `29673942518` passed，覆盖 Agent response artifact contract。
- [x] `a002aea` 远端 CI run `29674019795` passed，覆盖 artifact manifest。
- [x] `c1b72db` 远端 CI run `29674090490` passed，覆盖 manifest demo。
- [x] `2adcf2c` 远端 CI run `29674342374` passed，覆盖 manifest JSON Schema。
- [x] `79fb125` 远端 CI run `29674445146` passed，覆盖客户端 manifest/schema 回归模板。
- [x] `8f9cd99` 远端 CI run `29674535626` passed，覆盖 README artifact manifest 入口。
- [x] `292c8bf` 远端 CI run `29674625675` passed，覆盖 README manifest demo 输出样例。
- [x] `d0827c2` 远端 CI run `29674719845` passed，覆盖一页式验收 manifest 入口。
- [x] `fcdda6f` 远端 CI run `29674820980` passed，覆盖 quickstart manifest 验证入口。
- [x] `d7d24da` 远端 CI run `29674914207` passed，覆盖 installation manifest 验证入口。
- [x] `7519cf2` 远端 CI run `29675007824` passed，覆盖 artifact manifest 维护规则。
- [x] 发布说明草案提交 `e7ca8a1` 远端 CI run `29675124697` passed。

## 摘要

v0.5.11 候选重点是把 v0.5.10 发布后的 CI artifact 消费能力从“有 artifact 和 Agent response”推进到“有稳定契约、机器可读 manifest、JSON Schema、demo、接入方回归模板和维护规则”。

这个版本仍然不扩语言、不调整测试生成算法，也不改变 MCP tool 协议。核心价值是让 Codex / Claude / Cursor 这类 Agent 或客户端能稳定消费失败 artifact，并用自动化测试防止字段、路径和 fallback 顺序漂移。

## 主要变化

### Agent response artifact contract

- 新增 `docs/agent-response-artifact-contract.md`，统一 first-run/onboarding `agent-response.txt` 的四段结构：结论、证据、下一步、暂不做。
- contract 固定 first-run 与 onboarding 的证据字段差异。
- contract 明确失败读取顺序：先 `agent-response.txt`，再 decision、summary、report；旧版 first-run 再 fallback 到 `first-run-context.txt`。

### Artifact manifest 与 JSON Schema

- 新增 `docs/fixtures/agent-response-artifact-manifest.json`，以机器可读形式列出 first-run 和 onboarding artifact fixture。
- 新增 `docs/fixtures/agent-response-artifact-manifest.schema.json`，固定 manifest v1 的必填字段、artifact kind、文件名、fallback 顺序和 first-run/onboarding 字段关系。
- manifest 声明 `$schema`，方便客户端定位契约文件。
- 新增 Go schema 回归测试，验证当前 manifest 正例，并覆盖 schema_version、缺必填字段、非法 kind、fallback 首项错误等负例。

### Manifest demo 与客户端消费路径

- 新增 `examples/agent-response-manifest-demo`，读取 manifest 自动枚举并校验 artifact fixture。
- README、客户端集成说明、Agent response contract、quickstart、installation 和接入方一页式指南都已链接 manifest/schema。
- README 补充 manifest demo 的最小正常输出，方便接入方快速判断 demo 是否运行正常。

### 客户端契约测试模板

- `docs/mcp-client-contract-tests.md` 新增 CI artifact manifest 回归章节。
- 文档提供 `agent-response-artifact-manifest.json` 与 schema 的下载/校验命令。
- 无 JSON Schema 校验器时，文档要求至少运行 manifest demo 来验证 artifact 路径、必备字段和 fallback 顺序。

### 接入与维护文档收敛

- `docs/adopter-verification-guide.md` 把 artifact manifest/schema 纳入一页式接入清单。
- `docs/quickstart.md` 补 artifact manifest/schema 快速验证入口。
- `docs/installation.md` 从安装后自检段落指向 artifact manifest/schema 消费回归。
- `docs/fixtures.md` 补 artifact manifest/schema 维护规则，要求修改 manifest 时同步 schema、Go schema 测试、manifest demo 输出断言和入口文档。

## 质量边界

- v0.5.11 是 Agent/客户端 artifact 消费契约 patch，不是测试生成质量升级。
- manifest/schema 面向 CI artifact fixture，不替代 MCP tool 的 `structuredContent` contract。
- JSON Schema 固定的是 artifact manifest v1 的机器可读结构；artifact 内容一致性仍由 fixture 测试、demo 测试和 Go schema 测试共同覆盖。

## 本地验证

最近一次完整本地 gate 已通过：

- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `git diff --check`

重点新增/变更测试：

- [x] `sh test/agent_response_artifact_contract_doc_test.sh`
- [x] `sh test/agent_response_artifact_manifest_test.sh`
- [x] `sh test/agent_response_manifest_demo_test.sh`
- [x] `go test ./tools -run TestAgentResponseArtifactManifestSchema -count=1`
- [x] `sh test/mcp_client_contract_doc_test.sh`
- [x] `sh test/adopter_verification_guide_doc_test.sh`
- [x] `sh test/quickstart_doc_test.sh`
- [x] `sh test/installation_doc_test.sh`

## 发布前待办

- [x] 完成候选发布检查清单 `docs/plan-release-v0.5.11.md`。
- [ ] 正式版本准备时更新 `main.go` MCP implementation version 到 `0.5.11`。
- [ ] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.5.11 - 2026-07-19`。
- [ ] 同步 README、installation、quickstart 和接入指南中的版本引用到 `0.5.11` / `v0.5.11`。
- [ ] 跑完整发布前门禁和 release readiness。
- [ ] 提交版本准备后等待远端 CI。
- [ ] 打 `v0.5.11` tag，生成 Release 资产，更新 GitHub Release 和 Homebrew tap。

## 发布备注

- 对外文案应突出“Agent/客户端可消费的 CI artifact 契约”。
- 不要宣传成“测试生成增强”或“新增语言支持”。
- 推荐示例路径：先运行 `go run ./examples/agent-response-manifest-demo docs/fixtures/agent-response-artifact-manifest.json`，再按 `docs/mcp-client-contract-tests.md` 把 manifest/schema 校验放进客户端 CI。
