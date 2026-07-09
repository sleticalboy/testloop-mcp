# v0.4.11 发布说明草案

## 标题

testloop-mcp v0.4.11

## 摘要

v0.4.11 是 v0.4.10 之后的 JS/TS 静态生成质量增强版本。这个版本不新增 MCP 工具，重点提升 TypeScript DTO payload 的可解释生成能力，并把复杂 payload 从 helper 回归推进到 `generate_tests -> run_tests` 的 handler 闭环检查。

## 主要变化

- JS/TS payload 支持继续扩展到常见 DTO 组合：utility wrapper、`Pick`、`Omit`、`Record`、对象交叉、indexed access、数组、tuple、labeled/rest tuple。
- 对象字段内部的数组、tuple、Record、投影类型和组合 alias 现在会继续展开，而不是退化成空数组、空对象或浅层 `{ ok: true }`。
- `response.json()` 和注入式 client 两条真实生成路径都覆盖复杂 DTO payload，mock return、结构断言和 client call 断言保持一致。
- coverage task 生成路径补齐复杂 Vitest payload 的 handler 闭环检查，确保覆盖率驱动增量测试不会落后于普通生成路径。
- 新增 `docs/js-ts-payload-quality.md`，集中记录 JS/TS payload 的支持范围、保守回退、不支持边界和后续演进原则。

## 质量边界

本版本仍不尝试实现完整 TypeScript 类型系统。静态生成器只展开同文件可见且可解释的对象结构；跨文件类型、泛型实例化、条件类型、mapped type、动态 indexed access、复杂 discriminated union 和运行时 schema 推导仍交给 Agent 或可选 LLM provider 增强。

保守回退策略保持稳定：

- 未知命名类型或跨文件类型回退 `{}` 或外层保守 payload。
- 自引用类型在递归位置回退 `{}`。
- 无法解释的数组元素回退 `[]`。
- 无法解释的 Record key 在字段值中回退 `{}`。
- 非对象交叉分支和动态 indexed access 不生成半截对象。

## 回归保护

- helper 级 payload 测试覆盖边界和负例。
- 普通生成级测试固定最终 Vitest/Jest/Mocha 测试文本。
- coverage task 级测试固定任务上下文、建议输入和框架断言风格。
- handler 闭环测试覆盖普通生成与 coverage task 的 `generate_tests -> run_tests` 路径，并通过 fake `npx` 校验生成文件内容、执行路径和解析结果。

## 验证

- [x] `go test ./internal/generator -run 'TestJestAssertionAndDedupeCompatHelpers' -count=1`
- [x] `go test ./internal/generator -run 'TestGenerateJavaScriptTestsComplexTypeCompositionPayloads' -count=1`
- [x] `go test ./tools -run 'TestHandleGenerateTestsComplexVitestOutputIsRunnerChecked' -count=1`
- [x] `go test ./tools -run 'TestHandleGenerateCoverageTaskComplexVitestOutputIsRunnerChecked' -count=1`
- [x] `git diff --check`
- [x] `go test ./...`
- [x] 远端 CI passed：`28994273935`

## 发布前注意

- 这是 post-v0.4.10 的候选发布资料，不回写已经发布的 `docs/plan-release-notes-v0.4.10.md`。
- 本轮主要是静态生成质量和回归保护增强，不改变 MCP 工具协议。
- fake `npx` handler 闭环用于 CI 中的可运行性契约检查，不代表已经引入真实 npm/Vitest 安装依赖。
- 发布前仍需重新跑完整 release checklist、更新 MCP implementation version、README/安装文档/CHANGELOG 版本号，并验证 Release Artifacts 与 Homebrew tap。
