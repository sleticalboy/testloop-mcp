# JS/TS payload 质量边界说明

这份文档记录 JS/TS 静态生成器当前的 payload 质量边界。它的目标不是实现完整 TypeScript 类型系统，而是在常见 API helper、DTO 和 client 注入场景里生成稳定、可读、可运行的测试草稿。

## 当前质量目标

JS/TS payload 生成遵循三个原则：

- 稳定优先：同一份源码应生成确定性测试数据，方便 golden、CI 和 Agent 复跑。
- 可解释优先：只展开当前文件中能静态解释的类型结构，不跨文件猜测。
- 可运行优先：无法解释时回退到保守值，避免生成语法正确但语义任意的断言。

## 已覆盖场景

当前生成器已经覆盖以下主路径：

- `response.json()` 返回：会根据函数返回注解生成 mock JSON 和 `toEqual` / `deep.equal` 断言。
- 注入式 client 返回：支持 `client.get()`、`api.fetch()`、`http.request()` 等显式参数调用，生成局部 mock、返回 payload 和调用参数断言。
- 普通生成与 coverage task：两条入口共用同一套 payload 推导，coverage task 会额外使用目标函数、测试名、建议输入和断言关注点。
- Jest/Vitest/Mocha：按框架生成对应断言风格，Vitest ESM/TS 会显式导入测试 API。

## 支持的 TypeScript 类型形态

支持范围集中在同文件可见、能直接转成测试数据的 DTO 类型：

- 基础类型和字段名启发式值：`number`、`string`、`boolean`、`email`、`url`、`status`、`createdAt` 等。
- literal union：优先使用第一个稳定字面量，例如 `'active' | 'disabled'`。
- nullable union：`User | null`、`null | User` 优先选择非空分支。
- 同文件对象类型：`interface User { ... }`、`type User = { ... }`。
- 数组：`T[]`、`Array<T>`、`ReadonlyArray<T>`、`readonly T[]`。
- tuple：`[A, B]`、`readonly [A, B]`、labeled tuple、optional tuple element、rest tuple。
- utility wrapper：`Readonly<T>`、`Required<T>`、`Partial<T>`。
- 投影类型：`Pick<T, 'a' | 'b'>`、`Omit<T, 'a' | 'b'>`。
- 字典类型：`Record<string, T>`、`Record<'a' | 'b', T>`。
- 对象交叉：`A & B`，仅当所有分支都能解析为对象时合并。
- indexed access：`T['field']`，仅支持单个字符串字面量 key。
- 对象字段内嵌组合：数组、tuple、Record、Pick/Omit、indexed access 和组合 alias 可以作为 DTO 字段继续展开。

## 保守回退策略

以下回退是有意设计，不应轻易改成猜测：

- 未知命名类型或跨文件类型：回退 `{}` 或更外层的保守 payload。
- 自引用类型：展开第一层后在递归位置回退 `{}`，避免无限嵌套。
- 无法解释的数组元素：回退 `[]`。
- 无法解释的 Record key：字段值回退 `{}`；顶层 `Record<number, T>` 不生成 payload。
- 非对象交叉分支：例如 `User & string` 不生成半截对象。
- indexed access 的 union key、`keyof`、泛型 `T[K]`：不展开。

这些回退的核心目的是避免测试草稿看起来很具体，但其实来自不可解释的静态猜测。

## 明确不支持

当前阶段不承诺支持完整 TypeScript 语义，包括：

- 跨文件 import/export 类型解析。
- 泛型实例化和约束推导，例如 `ApiResponse<T>`、`T extends User`。
- 条件类型、mapped type、template literal type。
- `keyof`、动态 indexed access、复杂 union/discriminated union 完整分支枚举。
- 运行时 schema 解析，例如 Zod、Yup、io-ts。
- 业务语义断言，例如权限、排序、状态机或副作用验证。

这些能力更适合由 Agent 或可选 LLM provider 基于项目上下文增强，而不是由静态生成器无边界扩张。

## 当前回归保护

JS/TS payload 已经有三层测试保护：

- helper 级：直接固定 payload 推导函数，覆盖边界和负例。
- 普通生成级：固定最终生成的 Jest/Vitest/Mocha 测试文本。
- coverage task 级：固定目标过滤、任务上下文、建议输入和框架断言风格。
- handler 闭环级：通过 `generate_tests -> run_tests` 临时项目 fixture 校验生成文件、执行路径和解析结果。

最近补齐的字段级组合能力已经覆盖 `response.json()` 和注入式 client 两条真实生成路径。后续修改生成器时，至少应跑：

```bash
go test ./internal/generator -run 'TestGenerateJavaScriptTestsComplexTypeCompositionPayloads|TestJestAssertionAndDedupeCompatHelpers' -count=1
go test ./...
```

## 后续演进原则

后续增加 JS/TS 类型支持时，需要同时满足：

- 能解释：实现能说明来源，不依赖随意猜测。
- 有边界：负例和回退行为要写入测试。
- 进真实入口：不能只测 helper，还要覆盖最终生成文本。
- 不破坏闭环：生成结果要能被 `run_tests` 和失败解析继续消费。

如果一个能力需要完整类型检查器、跨文件工程图或业务语义推理，优先考虑把静态草稿交给 Agent/LLM provider 增强，而不是把静态生成器做成半套 TypeScript compiler。
