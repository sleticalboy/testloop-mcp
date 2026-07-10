# v0.4.13 发布说明草案

## 标题

testloop-mcp v0.4.13

## 发布状态

- [x] 创建 v0.4.13 发布说明草案。
- [x] 确认 v0.4.12 之后的 `Unreleased` 范围主要是 LLM provider 接入质量、错误可观测性、Agent fallback 闭环和安装脚本 fallback 日志修正。
- [x] 更新 `main.go` MCP implementation version 到 `0.4.13`。
- [x] 将 `CHANGELOG.md` 的 Unreleased 内容收敛为 `v0.4.13 - 2026-07-10`。
- [x] 更新 README、安装文档和必要的版本引用。
- [x] 跑完整本地发布前验证。
- [ ] 推送 `v0.4.13` tag 并等待 Release Artifacts workflow 完成。
- [ ] 验证 Release 资产、checksum、安装脚本和 Homebrew tap。
- [ ] 发布 GitHub Release 正文。

## 摘要

v0.4.13 是 v0.4.12 之后的 LLM provider 质量和 Agent 闭环增强版本。这个版本不内置具体模型厂商，也不把测试生成目标改成“替代 Claude/Cursor/Codex 写测试”；重点是把外部 LLM 生成纳入 testloop-mcp 的测试反馈闭环：

- 给外部 provider 更完整的上下文和默认 prompt。
- 对 provider 输出做清洗、轻量校验和坏输出分类。
- 让 Agent 能用结构化 `provider_error` 决定重试、降级 static 或提示用户修配置。
- 确保无论 LLM provider 成功还是失败，最终都回到 `generate_tests -> run_tests -> repair_task` 的可验证路径。

## 主要变化

- JS/TS `payload_notes` 在遇到 imported type 时会追加 import 来源和候选源码文件提示，帮助 Agent/LLM provider 读取跨文件类型上下文。
- `examples/llm-provider.sh` 会消费 `payload_notes` 中的候选源码文件，并把 imported type context 拼入 prompt。
- 新增默认 prompt 模板 `examples/llm-provider-prompt.md`，包含源码文件、语言、框架、coverage task、静态草稿、imported type context 和完整请求 JSON。
- 新增 Ollama 和 OpenAI CLI 模型命令包装示例，降低真实模型接入成本。
- 外部 LLM provider 输出支持清洗常见 Markdown 代码围栏和前后解释性文本。
- 外部 LLM provider 输出增加按目标语言的轻量测试代码校验，Go/Python/JS/TS/Rust/Java 会拒绝明显不是测试的代码片段。
- `cmd/testgen` 新增 `-provider-check`，用于在真正生成测试前诊断 provider 模式、`TESTLOOP_LLM_PROVIDER_CMD` 和命令可执行性。
- MCP `generate_tests` 的 LLM provider 失败新增稳定 `provider_error.kind` / `provider_error.action`，覆盖配置缺失、命令失败、空输出、JSON 错误、缺少 `code`、清洗失败和语言校验失败。
- `generate_tests` 的 LLM provider 失败会返回 `isError=true` 的结构化工具结果，并在 JSON / `structuredContent` 中提供 `provider_error.kind`、`provider_error.action`、`provider_error.provider` 和 `provider_error.message`。
- 默认 LLM provider prompt 新增输出契约，要求模型只返回可直接写盘的完整测试文件，无法安全增强时回退静态草稿。
- Agent workflow 明确 provider 错误策略：哪些错误可重试一次，哪些应降级 static，哪些应提示用户修 provider 配置。
- 新增 provider 成功输出进入 `run_tests include_fix_suggestions=true` 的 handler 闭环测试。
- 新增 provider 坏输出分类回归测试，固定空输出、JSON 错误、缺少 `code`、解释文本和非测试代码的 `provider_error kind/action`。
- 新增结构化 `provider_error -> static fallback -> run_tests` 的 handler 闭环测试，固定 Agent fallback 序列可执行。
- `scripts/install.sh` 的 `go install` fallback 日志会根据实际落盘文件名输出安装路径，避免跨平台 dry run 下载失败时把当前主机二进制误报为 `.exe`。

## 质量边界

v0.4.13 仍保持 LLM provider 是外部命令协议适配层：

- 不内置 OpenAI、Claude、Ollama 或其他模型 SDK。
- 不在 MCP 请求中接收任意命令；provider 命令仍只能由服务端环境变量配置。
- 不对模型输出做完整语法编译或框架级执行校验；成功生成后仍必须调用 `run_tests`。
- 不自动无限重试 provider；Agent 推荐同一任务最多重试一次，仍失败则降级 static。
- `static_code` 是可用草稿和回退结果，不代表复杂业务断言已经完整。
- imported type 候选文件是给 Agent/LLM provider 的读取建议，不等同于内置跨文件 TypeScript 类型系统。

## 回归保护

- provider 请求级测试固定 JS/TS `payload_notes` 会随 stdin JSON 传给外部 provider。
- provider 示例脚本测试固定 imported type context 会进入 prompt，并覆盖 Ollama / OpenAI CLI dry run 包装。
- generator 级测试固定 LLM provider 输出清洗、JSON 解析、缺少 `code`、语言级轻量校验和 `ProviderErrorKind` 分类。
- CLI 测试固定 `testgen -provider-check` 的配置诊断、退出码和 provider 失败提示。
- handler 测试固定 MCP `generate_tests` 会把 provider 失败转成稳定的 `provider_error kind/action`。
- handler 测试固定 provider 成功生成的 Vitest 测试文件可以进入 `run_tests include_fix_suggestions=true`，并生成 repair task。
- handler 测试固定结构化 `provider_error` 可触发 Agent 降级 static，并继续执行 `run_tests`。
- 安装脚本测试固定跨平台下载失败时 `go install` fallback 的实际落盘路径日志。

## 验证

候选发布前至少需要重新执行：

- [x] `go test ./tools -run 'TestHandleGenerateTestsProviderErrorFallsBackToStaticAndRunTests|TestHandleGenerateTestsClassifiesLLMProviderBadOutputs' -count=1`
- [x] `go test ./tools -run 'TestHandleGenerateTestsClassifiesLLMProvider(BadOutputs|ConfigError)|TestHandleGenerateTestsLLMProviderOutputFeedsRunTestsRepairLoop' -count=1`
- [x] `go test ./types ./tools -count=1`
- [x] `go test ./...`
- [x] `sh test/llm_provider_example_test.sh`
- [x] `sh test/install_script_test.sh`
- [x] `sh test/release_assets_test.sh`
- [x] `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/*.yml`
- [x] `git diff --check`
- [x] 远端 CI 通过：run `29077553397`。

正式发布前还需要执行：

- [x] `go build -o /tmp/testloop-mcp-v0.4.13-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.4.13-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.4.13-prep --help`
- [x] `/tmp/testloop-testgen-v0.4.13-prep --help`
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.4.13-prep scripts/package-release-asset.sh v0.4.13 darwin_arm64 darwin arm64`
- [x] 校验 `/tmp/testloop-v0.4.13-prep/testloop-mcp_v0.4.13_darwin_arm64.tar.gz.sha256`

## 发布备注

- 这是 v0.4.13 的候选发布说明草案，不回写已经发布的 `docs/plan-release-notes-v0.4.12.md`。
- 本轮重点是 LLM provider 接入质量和 Agent 可恢复闭环，不改变默认 static provider 行为。
- 结构化 `provider_error` 是向后兼容增强；只读文本的旧 Agent 仍可从 `error` 字段解析 `provider_error kind=... action=...`。
- 本轮新增的 provider 示例和模型包装是接入参考，不代表 testloop-mcp 内置或绑定任何模型供应商。
