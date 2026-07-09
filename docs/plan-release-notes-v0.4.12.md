# v0.4.12 发布说明草案

## 标题

testloop-mcp v0.4.12

## 发布状态

- [ ] 更新 `main.go` MCP implementation version 到 `0.4.12`。
- [ ] 将 `CHANGELOG.md` 的 Unreleased 内容收敛为 `v0.4.12 - 2026-07-09`。
- [ ] 更新 README、安装文档和必要的版本引用。
- [ ] 跑完整本地发布前验证。
- [ ] 推送 `v0.4.12` tag 并等待 Release Artifacts workflow 完成。
- [ ] 验证 Release 资产、checksum、安装脚本和 Homebrew tap。
- [ ] 发布 GitHub Release 正文。

## 摘要

v0.4.12 是 v0.4.11 之后的小版本收口，重点增强两条链路：

- JS/TS 静态 payload 对同文件简单泛型 DTO 的展开能力。
- 静态 payload 保守回退时，把原因贯通到 `generate_tests` 输出和外部 LLM provider 输入，方便 Agent 判断下一步是否需要读取更多项目上下文。

这个版本不新增 MCP 工具，不引入完整 TypeScript 类型系统，也不改变默认 static provider 行为。

## 主要变化

- JS/TS payload 支持同文件简单泛型 alias/interface 的直接实例化，例如 `ApiEnvelope<User>`、`Pair<User, Meta>`。
- TypeScript 类型声明提取会保留简单泛型参数，支持 `type ApiEnvelope<T> = ...` 和 `interface ApiEnvelope<T> { ... }`。
- `generate_tests.context.targets[]` 新增可选 `return_type_expr`，保留 TypeScript 返回注解。
- `generate_tests.context.targets[]` 新增可选 `payload_notes`，在跨文件类型、约束泛型、动态 indexed access / `keyof` 等场景导致静态 payload 回退时说明原因。
- 外部 LLM provider 的 stdin JSON 会携带同一份 `return_type_expr` 和 `payload_notes`，让 provider 可以基于静态草稿做增强，而不是误判 `{ ok: true }` 是完整业务 DTO。
- `scripts/install.sh` 的 `go install` fallback 提示更具体，会区分不支持的平台、latest 解析失败、Release 资产下载失败和缺少解压器。

## 质量边界

本版本只处理同文件、直接实例化、参数可简单替换的泛型 DTO。以下场景仍保持保守回退：

- 跨文件 import/export 类型解析。
- 泛型约束、默认参数和条件推导，例如 `T extends User`、`T = User`。
- `T[K]`、`keyof`、动态 indexed access。
- 条件类型、mapped type、template literal type。
- 运行时 schema 推导，例如 Zod、Yup、io-ts。

当静态生成器无法解释顶层 TypeScript 返回注解时，测试代码仍优先保持可运行，通常回退到 `{ ok: true }`；解释信息通过 `payload_notes` 暴露给 Agent 或 LLM provider。

## 回归保护

- helper 级测试覆盖同文件泛型 DTO 展开、嵌套投影参数、多泛型参数和约束泛型负例。
- parser 级测试固定 `interface Box<T>`、`type Pair<T, U>` 的声明 key。
- generator 级测试固定 `response.json()` 返回 `Promise<ApiEnvelope<User>>` 时生成结构化 payload。
- context 级测试固定 `return_type_expr` 和 `payload_notes` 的生成规则。
- handler 级测试固定 `generate_tests` MCP 输出 JSON 中保留 `payload_notes`。
- provider 请求级测试固定外部 LLM provider stdin JSON 中保留 `payload_notes`，并保留可增强的 static code。
- 安装脚本测试固定下载失败时的 fallback 提示。

## 验证

候选发布前至少需要重新执行：

- [ ] `go test ./internal/generator -run 'TestJSExtractTSTypeDeclsKeepsSimpleGenericParams|TestGenerateJavaScriptTestsComplexTypeCompositionPayloads|TestJestAssertionAndDedupeCompatHelpers' -count=1`
- [ ] `go test ./internal/generator -run 'TestGenerateTestsWithProviderIncludesJavaScriptPayloadFallbackNotes|TestGenerateTestsWithProviderIncludesCoverageTask' -count=1`
- [ ] `go test ./tools -run 'TestHandleGenerateTestsReturnsJavaScriptPayloadFallbackNotes|TestHandleGenerateTestsUsesJavaScriptFramework' -count=1`
- [ ] `bash test/install_script_test.sh`
- [ ] `git diff --check`
- [ ] `go test ./...`
- [ ] 远端 CI 通过。

## 发布备注

- 这是 v0.4.12 候选发布资料，当前还未正式发版。
- 本轮新增的 `return_type_expr` 和 `payload_notes` 是向后兼容字段；未使用该字段的调用方可以继续忽略。
- `payload_notes` 是给 Agent/LLM provider 的解释信息，不会写入生成的测试文件。
- v0.4.12 的正式发布仍需要走版本号更新、tag、Release Artifacts、资产校验和 Homebrew tap 更新流程。
