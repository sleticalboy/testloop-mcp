# testloop-mcp

[![Go Report Card](https://goreportcard.com/badge/github.com/sleticalboy/testloop-mcp)](https://goreportcard.com/report/github.com/sleticalboy/testloop-mcp)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**testloop-mcp** 是一个基于 [MCP (Model Context Protocol)](https://modelcontextprotocol.io) 的智能测试生成与执行反馈闭环服务器。让 AI Coding 工具（Claude Code / Cursor / VS Code Copilot 等）能够自动生成测试、执行测试、解析失败原因、生成修复建议，并分析覆盖率——形成完整的测试闭环。

## 核心能力

- **智能生成测试** — Go 优先复用 `gotests` 并回退内置 `go/ast`，其他语言结合 tree-sitter/轻量解析器生成类型感知测试。支持泛型、async、Result/Option 返回类型、JUnit 5 断言，可选接入外部 LLM provider
- **执行测试** — 支持 `go test` / `cargo test` / Jest / Vitest / Mocha / pytest / JUnit 5（Maven/Gradle），自动检测项目类型，可选收集覆盖率
- **解析失败** — 结构化解析测试输出（Go/`cargo test`/Jest/Vitest/Mocha/pytest/JUnit），提取失败用例的文件、行号、错误信息，AI 友好 JSON 格式
- **修复建议** — 根据失败类型（期望值不匹配 / nil pointer / 数组越界 / 除零 / 类型不匹配等）生成结构化修复建议
- **覆盖率分析** — 解析 Go coverprofile / Istanbul coverage JSON / coverage.py JSON / cargo tarpaulin LCOV / JaCoCo XML，输出文件级覆盖率、未覆盖 block 定位和改进建议；Go/Rust/Java 会尽量把缺口映射到具体函数或方法，并识别常见分支、返回、错误路径

## 架构概览

```
AI IDE (Claude Code / Cursor / Copilot)
        │  MCP JSON-RPC (stdio / Streamable HTTP)
        ▼
  testloop-mcp server
        │
        ├── generate_tests    → 社区工具/AST 分析源码 → 生成测试文件
        ├── run_tests         → 执行测试框架命令 → 结构化结果
        ├── parse_results     → 解析测试输出 → 提取失败详情
        ├── fix_suggestions   → 失败信息 + 源码 → 修复建议
        ├── parse_coverage    → 覆盖率数据 → 报告 + 改进建议
        └── validate_coverage_task → 覆盖率任务 → 生成 + 执行 + 反馈
        │
        ▼
  本地项目（Go / Rust / Java / Node.js / Python）
```

## 支持的框架

| 语言 | 测试框架 | 生成 | 执行 | 解析 | 覆盖率 |
|------|---------|:----:|:----:|:----:|:------:|
| Go | `go test` | ✅ | ✅ | ✅ | ✅ |
| Rust | `cargo test` | ✅ | ✅ | ✅ | ✅ |
| Node.js | Jest | ✅ | ✅ | ✅ | ✅ |
| Node.js | Vitest | ✅ | ✅ | ✅ | ✅ |
| Node.js | Mocha | ✅ | ✅ | ✅ | ✅ |
| Python | pytest | ✅ | ✅ | ✅ | ✅ |
| Java | JUnit 5 (Maven/Gradle) | ✅ | ✅ | ✅ | ✅ |

> 测试生成：Go 优先使用 `gotests`，失败时回退内置 `go/ast`；JS/TS/Python/Rust/Java 基于 tree-sitter/轻量解析器。传入 `coverage_task` 时，Go/Python/JS/TS/Rust/Java 静态生成器会优先聚焦任务目标函数或方法，使用任务推荐测试名，并把建议输入代入生成的测试草稿；Go 已覆盖常见纯函数、HTTP/JSON/JWT/recover、指针返回、receiver 字段变更等高频可静态构造场景；JS/TS 会根据任务框架输出 Jest/Vitest 或 Mocha/Chai 风格断言。
> 覆盖率：当前支持 Go coverprofile、Istanbul coverage JSON（Jest/Vitest/Mocha）、coverage.py JSON、Rust `cargo tarpaulin --out Lcov` 生成的 LCOV，以及 Java JaCoCo XML。Go/Rust/Java 覆盖率建议会尽量定位到具体函数或方法，并输出常见分支/返回/错误路径分类，便于 AI Agent 直接补测试。

## 安装

macOS / Linux 推荐使用 Homebrew：

```bash
brew tap sleticalboy/tap
brew install testloop-mcp
```

也可以使用安装脚本。脚本会优先下载当前平台匹配的 GitHub Release 资产，支持 Linux/macOS tarball 和 Windows amd64/arm64 zip；当前 release 没有匹配资产时，会自动回退到 `go install`：

```bash
curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/install.sh | sh
```

Windows Git Bash/MSYS 用户需要确保安装目录在 `PATH` 中；详细说明见 [安装与接入](docs/installation.md)。

当前 `v0.4.14` Release 已提供 Linux amd64、Linux arm64、macOS arm64、Windows amd64 和 Windows arm64 二进制。手动下载示例：

```bash
curl -LO https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.14/testloop-mcp_v0.4.14_linux_amd64.tar.gz
curl -LO https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.14/testloop-mcp_v0.4.14_linux_amd64.tar.gz.sha256
sha256sum -c testloop-mcp_v0.4.14_linux_amd64.tar.gz.sha256
tar -xzf testloop-mcp_v0.4.14_linux_amd64.tar.gz
./testloop-mcp --help
```

Release 产物会同时提供单资产 `.sha256` 文件，安装脚本会自动选择可用的校验文件。

Windows amd64/arm64 可直接下载 zip；将 `$arch` 设为 `amd64` 或 `arm64`：

```powershell
$arch = "amd64"
curl.exe -LO "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.14/testloop-mcp_v0.4.14_windows_$arch.zip"
curl.exe -LO "https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.14/testloop-mcp_v0.4.14_windows_$arch.zip.sha256"
$expected = (Get-Content ".\testloop-mcp_v0.4.14_windows_$arch.zip.sha256").Split()[0]
$actual = (Get-FileHash ".\testloop-mcp_v0.4.14_windows_$arch.zip" -Algorithm SHA256).Hash.ToLower()
if ($actual -ne $expected) { throw "checksum mismatch" }
Expand-Archive ".\testloop-mcp_v0.4.14_windows_$arch.zip"
& ".\testloop-mcp_v0.4.14_windows_$arch\testloop-mcp.exe" --help
& ".\testloop-mcp_v0.4.14_windows_$arch\testloop-testgen.exe" --help
```

其他未覆盖平台或需要从源码构建：

```bash
git clone https://github.com/sleticalboy/testloop-mcp.git
cd testloop-mcp
go build -o testloop-mcp .
go build -o testloop-testgen ./cmd/testgen
```

也可以直接安装到 Go bin 目录：

```bash
go install github.com/sleticalboy/testloop-mcp@latest
go install github.com/sleticalboy/testloop-mcp/cmd/testgen@latest
```

**前置要求：** Go 1.25+；源码构建需要 CGO 可用的 C 编译工具链。

更完整的下载、校验、Docker 和客户端接入说明见 [安装与接入](./docs/installation.md)。

## 配置接入

可以先用命令生成当前机器上的配置片段：

```bash
testloop-mcp --print-config=all
```

如果需要指定配置里的二进制路径，追加 `--config-command=/absolute/path/to/testloop-mcp`。

配置写入后可以校验 `command` 是否存在且可执行，或 `url` 是否是合法 HTTP endpoint：

```bash
testloop-mcp --check-config ~/.codex/config.toml
```

校验失败时会输出对应的 `--print-config` 或 `--doctor-config` 建议，便于直接修复缺失或不可执行的配置。

也可以查看推荐配置路径和本机诊断：

```bash
testloop-mcp --doctor-config
```

诊断会区分“配置文件存在但缺少 `testloop` server”和“已有其他 MCP server 配置正常”，并给出可复制的 `--print-config` 修复建议。

### Codex

`~/.codex/config.toml`:

```toml
[mcp_servers.testloop]
command = "/absolute/path/to/testloop-mcp"
```

### Claude Code / Claude Desktop

`~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "testloop": {
      "command": "/absolute/path/to/testloop-mcp"
    }
  }
}
```

### Cursor

`.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "testloop": {
      "command": "/absolute/path/to/testloop-mcp"
    }
  }
}
```

## MCP Tools

### `generate_tests`

根据源文件生成测试代码。支持 Go（优先 `gotests`，回退内置 AST 分析）、Rust、Java、JavaScript/TypeScript（自动从 `package.json` 识别 Jest/Vitest/Mocha；也可显式传 `framework`）、Python（pytest）。

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `file_path` | string | ✅ | 源文件路径（`.go` / `.rs` / `.java` / `.js` / `.ts` / `.jsx` / `.tsx` / `.py`） |
| `framework` | string | — | 测试框架，默认自动检测；JS/TS 会向上查找 `package.json`，支持 `jest` / `vitest` / `mocha` |
| `provider` | string | — | 测试生成 provider：`static` / `llm` / `auto`，默认 `static` |
| `coverage_task` | object | — | `parse_coverage` 返回的单个 `test_tasks` 项，用于按覆盖率缺口生成测试 |

**返回：** `{ status, test_file, generated_cases, preview, context, coverage_task, provider, error, provider_error }`

传入 `coverage_task` 时，工具会优先写入任务中的 `test_file`，并把任务回写到返回的 `context.coverage_task`。内置 static provider 会在 Go/Python/JS/TS/Rust/Java 中按目标函数或方法收窄生成范围，使用任务推荐测试名，把 `assertion_focus` 和 `suggested_inputs` 写入注释，并从建议输入中的条件表达式提取参数值生成更贴近覆盖率缺口的调用。Go 任务会优先把可静态构造的分支、返回路径、错误路径和接收者字段变更转成非 skipped 断言；无法稳定构造的系统资源、协议 I/O 或数据库/GORM 分支会交给 `validate_coverage_task` 标记为人工复核。JS/TS 普通生成和 coverage task 都会按框架选择 Jest/Vitest 的 `expect(...).toBe(...)` / `toEqual(...)` 或 Mocha/Chai 的 `expect(...).to.equal(...)` / `to.deep.equal(...)` 风格；ESM/TS Vitest 生成会显式导入 `describe` / `it` / `expect`，CommonJS 仍沿用 runner 注入的全局 API；coverage task 会识别 `export default` 实例并通过默认导入调用实例方法，避免把未导出的内部 class 当成可构造 API；TS/TSX 源文件在最近的 `tsconfig.json` 使用 `module` 或 `moduleResolution` 为 `node16` / `nodenext` 时，会用 `./module.js` 形式导入源模块，其他 ESM 场景保持 `./module`；未显式传 `framework` 时会复用自动检测结果。LLM provider 也会收到同一份 task 上下文，便于在静态草稿基础上进一步增强断言。

普通生成的 JS/TS 测试文件默认写在源文件同目录，例如 `src/sum.ts` → `src/sum.test.ts`、`lib/calc.js` → `lib/calc.test.js`。`run_tests` 会从 `package.json` 所在目录执行，并把生成文件转成相对项目根的参数；覆盖率任务仍优先使用 `coverage_task.test_file` 推荐路径。

**LLM provider：** 默认不依赖任何外部 LLM。需要启用时，在服务端配置 `TESTLOOP_LLM_PROVIDER_CMD`，并调用 `generate_tests` 时传 `provider: "llm"` 或 `provider: "auto"`。命令会从 stdin 接收 JSON（`source_file`、`context`、`static_code`），其中 `context.coverage_task` 会携带覆盖率任务上下文；JS/TS 目标还会在 `return_type_expr` 和 `payload_notes` 中说明 TypeScript 返回注解、跨文件类型/复杂泛型等导致静态 payload 回退的原因，以及 imported type 的来源和候选源码文件；stdout 可以直接返回测试代码，也可以返回 `{"code":"..."}`。provider 输出会自动清洗常见 Markdown 代码围栏和前后解释性文本；如果输出不含可识别测试代码，或对 Go/Python/JS/TS/Rust/Java 来说明显不像测试文件，会返回错误。`auto` 在未配置命令时会自动回退到 `static`。

当 LLM provider 失败时，`generate_tests` 会返回 `isError=true` 的工具结果，JSON / `structuredContent` 中包含 `status: "error"`、`error` 和结构化 `provider_error` 字段；`provider_error.kind` / `provider_error.action` 用于区分配置缺失、命令失败、模型输出为空、JSON 错误和语言校验失败等场景，方便 Agent 决定重试、降级 static 或提示用户修配置。为兼容旧消费方，`error` 文本仍包含 `provider_error kind=... action=...` 片段；推荐策略见 [Agent 工作流](./docs/agent-workflow.md)。

LLM provider 示例见 [docs/llm-provider.md](./docs/llm-provider.md) 和 [examples/llm-provider.sh](./examples/llm-provider.sh)。示例脚本会根据 `payload_notes` 读取 imported type 的候选文件，并使用 [examples/llm-provider-prompt.md](./examples/llm-provider-prompt.md) 组装 prompt；默认模板包含严格输出契约，要求模型只返回一个可直接写盘的完整测试文件，无法安全增强时回退静态草稿。可通过 `TESTLOOP_LLM_PROVIDER_PROMPT_FILE` 调试 prompt，通过 `TESTLOOP_LLM_PROVIDER_PROMPT_TEMPLATE` 替换模板，或通过 `TESTLOOP_LLM_PROVIDER_MODEL_CMD` 接入真实模型命令。仓库提供了 Ollama 和 OpenAI CLI 的模型命令包装示例。

Agent 端到端闭环示例见 [docs/agent-workflow.md](./docs/agent-workflow.md)。无论使用 `static` 还是 `llm` provider，生成测试后都应继续调用 `run_tests`；失败时建议开启 `include_fix_suggestions=true`，让生成结果直接进入结构化修复闭环。

**Go 生成器：** 优先调用本机 `gotests -all` 生成 Go 社区标准测试骨架；如果未安装 `gotests`、命令失败或输出为空，则回退到内置 `go/ast` 生成器。内置回退支持泛型类型参数实例化（`T → int`）、指针/值接收者方法、变参 `...T` → 切片、通道参数 nil-check + `t.Skip` 防阻塞、接口参数自动 mock、slice/map/struct 自动使用 `reflect.DeepEqual`。

**JS/TS 生成器：** tree-sitter + 函数体分析，识别函数、类方法、async、参数、TypeScript `private/protected`、getter、CommonJS / ES Module 导入，分析 `return` 语句推断返回类型（number/string/array/object/boolean）、对简单对象/数组字面量返回生成 `toEqual` / `deep.equal` 结构断言、为 `response.json()` 和参数注入的 `client.get()` / `api.fetch()` / `http.request()` 返回生成按实际调用方法收窄并记录调用参数的轻量 mock；当 TypeScript 返回注解是内联对象、对象数组、tuple 或同文件对象型 `interface` / `type` / 数组 alias 时，会用字段类型和字段名生成更具体的 mock payload，例如 `{ id: 1, email: 'user@example.com', status: 'active' }`，并支持同文件简单泛型 `ApiEnvelope<User>`、`Array<T>` / `ReadonlyArray<T>` / `readonly T[]` / `readonly [A, B]` / labeled tuple / rest tuple / `Readonly<T>` / `Required<T>` / `Partial<T>` / `Pick<T, 'a' | 'b'>` / `Omit<T, 'a' | 'b'>` / `Record<string, T>` / `Record<'a' | 'b', T>` / `A & B` / `T['field']`；对可选字段、nullable union 优先生成非空稳定值；自引用类型会在递归处回退 `{}`；无法展开跨文件类型或复杂泛型时，`context.targets[].payload_notes` 会说明回退原因；当返回注解引用 imported type 时，`payload_notes` 会追加 import 来源和候选源码文件，方便 Agent/LLM provider 读取更多上下文；检测 `throw` 生成 `toThrow()` 测试、检测 `if (param === value)` 边界条件生成针对性用例。

JS/TS payload 的支持范围、保守回退和不支持边界见 [docs/js-ts-payload-quality.md](./docs/js-ts-payload-quality.md)。

**Python 生成器：** tree-sitter + 函数体分析，识别函数、类方法、async、参数、`@staticmethod`、`*args`/`**kwargs`，分析 `return` 语句推断返回类型（int/float/str/list/dict/bool）、检测 `raise` 生成 `pytest.raises()` 测试、检测 `if param == value` 边界条件。

**覆盖率驱动生成闭环：**

1. 用 `run_tests` 或生态命令生成覆盖率报告。
2. 调用 `parse_coverage` 获取 `test_tasks`，每个任务包含目标、缺口类型、推荐测试文件、测试名、建议输入和断言重点。
3. 优先取单个 `test_tasks[]` 调用 `validate_coverage_task`，让工具自动执行 `generate_tests -> run_tests` 并返回 `passed` / `failed` / `generation_error`、建议动作和失败修复摘要。
4. 需要手动控制 provider 或调试生成内容时，也可以把同一个任务作为 `generate_tests.coverage_task` 传入，再调用 `run_tests` 重新执行测试，必要时把失败交给 `parse_results` / `fix_suggestions` 继续闭环。

---

### `validate_coverage_task`

对单个覆盖率任务执行生成后验证闭环。工具会先调用 `generate_tests` 写入任务推荐的测试文件，再调用 `run_tests` 执行该测试文件；测试失败时默认开启 `include_fix_suggestions`，把失败原因和修复任务一起返回。

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `file_path` | string | ✅ | 源文件路径 |
| `coverage_task` | object | ✅ | `parse_coverage.test_tasks[]` 中的单个任务 |
| `framework` | string | — | 测试框架，默认使用 `coverage_task.framework` 或自动检测 |
| `provider` | string | — | 测试生成 provider：`static` / `llm` / `auto`，默认 `static` |
| `coverage` | bool | — | 执行测试时是否收集覆盖率，默认 `false` |
| `include_fix_suggestions` | bool | — | 测试失败时是否附带 `fix_suggestions[]`，默认 `true` |

**返回：** `{ status, action, coverage_task, generated, run_result, error, provider_error, metadata }`

`status` 为 `passed` 时，说明生成的测试已经通过当前测试命令；`action=ready` 表示可以进入下一个任务，`action=manual_review_unreachable` 表示目标分支疑似不可达，`action=manual_review_environment` 表示目标错误路径依赖 OS、硬件或第三方库内部错误，`action=manual_review_protocol` 表示目标依赖 socket write/streaming I/O 等协议时序错误，`action=manual_review_database` 表示目标依赖 GORM/数据库行为且当前项目没有安全的静态测试数据库策略，`action=manual_review_external_service` 表示目标依赖 live RPC、外部服务、路由状态或长重试时序，应该通过 fake client/route data 或集成环境验证，而不是继续修生成的单元测试，`action=manual_review_private` 表示目标是 JavaScript `#private` 或 TypeScript `private/protected` method，不能从外部测试直接调用；内置 static provider 会生成 skipped 复核草稿，并在可检测时通过 `metadata.public_entry_candidates` 给出公共入口候选。`action=manual_review_internal` 表示目标是 JavaScript ESM 内部未导出的 class/function，外部测试不能直接导入构造，应通过导出的公共 API、测试 seam 或模块级集成测试覆盖。`failed` 表示生成成功但测试未通过，Agent 应读取 `run_result.failures[]` 和 `run_result.fix_suggestions[]`；`generation_error` 表示测试草稿未能生成，优先查看 `provider_error.action` 或 `error`；`run_error` 表示测试命令执行入口本身异常。

---

### `run_tests`

执行测试并返回结构化结果。

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `path` | string | ✅ | 测试文件或目录路径 |
| `framework` | string | — | `go-test` / `cargo-test` / `jest` / `vitest` / `mocha` / `pytest` / `junit`，默认自动检测 |
| `coverage` | bool | — | 是否收集覆盖率，默认 `false` |
| `verbose` | bool | — | 详细输出，默认 `true` |
| `include_fix_suggestions` | bool | — | 测试失败时附带 `fix_suggestions[]` 摘要，默认 `false` |
| `source_code` | string | — | 源码文件路径，用于生成修复上下文 |
| `test_code` | string | — | 测试文件路径，用于生成修复上下文 |

**返回：** `{ status, framework, total, passed, failed, skipped, coverage_percent, failures[], fix_suggestions[], raw_output }`

`coverage=true` 时，Rust 会额外调用 `cargo tarpaulin --out Lcov --output-dir target/tarpaulin` 并回填 `coverage_percent`；Java Maven/Gradle 项目会执行 JaCoCo report 任务并从 XML 报告回填 `coverage_percent`。也可以通过 `parse_coverage` 直接解析已有 LCOV/JaCoCo XML 文件。

`include_fix_suggestions=true` 且测试失败时，`run_tests` 会把失败结果同步转换为 `fix_suggestions[]`，其中包含 `repair_task`。未传 `source_code` / `test_code` 时仍会返回基础分类和任务信息，但源码/测试行上下文可能不完整。

---

### `parse_results`

解析测试执行输出，提取失败用例详情。

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `output` | string | ✅ | 测试执行的标准输出/错误输出原文 |
| `framework` | string | — | `go-test` / `cargo-test` / `jest` / `vitest` / `mocha` / `pytest` / `junit`，默认 `go-test` |

**返回：** 同 `run_tests` 的结构化结果，聚焦失败用例的文件名、行号、错误信息。

---

### `fix_suggestions`

根据测试失败信息和源代码，生成结构化修复建议。

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `failures` | string | ✅ | `parse_results` 返回的失败 JSON 数组 |
| `source_code` | string | ✅ | 源代码文件路径 |
| `test_code` | string | — | 测试代码文件路径（可选，增强分析） |

**返回：** `[{ file, line, issue, category, context_file, context_line, suggested_fix, confidence, repair_task }]`

识别的失败类型：期望值不匹配（`got X, want Y`、Jest/Vitest/Mocha 的 expected/received）、nil pointer panic、数组越界、除零错误、未定义引用、类型不匹配。返回会用 `category` 标识失败类型，并在能匹配到源码或测试文件时填充 `context_file` / `context_line`；建议文本也会尽量带上实际/期望值、越界索引和长度、相关源码行等上下文，便于 Agent 判断是修测试还是修实现。

`repair_task` 是面向 Agent 的稳定子契约，包含 `id`、`test_name`、`category`、`target_file`、`target_line`、`context_snippet`、`editable_files`、`suggested_commands` 和 `assertion_focus`，用于把单个失败转成可执行修复任务。

---

### `parse_coverage`

解析覆盖率数据，返回结构化报告和改进建议。

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `data` | string | ✅ | 覆盖率数据（coverprofile、Istanbul JSON、coverage.py JSON、LCOV、JaCoCo XML 的文件路径或内容） |
| `framework` | string | — | `go-test` / `jest` / `vitest` / `mocha` / `pytest` / `cargo-test` / `junit`，默认 `go-test` |

各语言覆盖率报告生成命令见 [docs/coverage-formats.md](./docs/coverage-formats.md)。

**返回：**

```json
{
  "framework": "go-test",
  "total_percent": 58.8,
  "files": [
    {
      "path": "example.com/pkg/calc.go",
      "percent": 91.7,
      "blocks": [
        { "start_line": 1, "end_line": 3, "count": 1, "covered": true },
        { "start_line": 5, "end_line": 7, "count": 0, "covered": false }
      ]
    }
  ],
  "summary": {
    "total_statements": 34,
    "covered_statements": 20,
    "total_files": 3,
    "covered_files": 3,
    "uncovered_files": []
  },
  "suggestions": [
    { "file": "example.com/pkg/calc.go", "line_range": "5-7", "reason": "此代码块未被测试覆盖", "confidence": 0.9 }
  ],
  "test_tasks": [
    {
      "id": "go-test-1",
      "framework": "go-test",
      "file": "example.com/pkg/calc.go",
      "target": "Add",
      "line_range": "5-7",
      "goal": "为 Add 补充测试，覆盖未执行行段 5-7",
      "command": "go test ./example.com/pkg",
      "test_file": "example.com/pkg/calc_test.go",
      "test_name": "TestAdd",
      "assertion_focus": ["断言未覆盖分支的返回值或副作用"],
      "priority": 103,
      "priority_reason": "已定位到具体函数或方法；分支缺口通常能生成高价值断言；已有建议输入"
    }
  ]
}
```

## 项目结构

```
testloop-mcp/
├── main.go                          # MCP server 入口，注册 MCP 工具
├── go.mod                           # github.com/sleticalboy/testloop-mcp, go 1.25
├── types/
│   └── types.go                     # 所有共享类型定义
├── tools/
│   ├── run_tests.go                 # run_tests 工具 + Register() 注册入口
│   ├── generate_tests.go            # generate_tests 工具
│   ├── parse_results.go             # parse_results 工具
│   ├── fix_suggestions.go           # fix_suggestions 工具
│   ├── parse_coverage.go            # parse_coverage 工具
│   └── validate_coverage_task.go    # validate_coverage_task 工具
├── internal/
│   ├── generator/
│   │   ├── generator.go              # 多语言静态生成分发入口
│   │   ├── provider.go               # 测试生成 provider 接口 + 可选 LLM command provider
│   │   ├── context.go                # 面向 LLM/AI Agent 的测试生成上下文
│   │   ├── go_gotests.go             # gotests 优先生成器
│   │   ├── go_generator.go           # Go AST 回退测试生成器（泛型/通道/接口/变参）
│   │   ├── js_generator.go           # JS/TS 测试生成器（函数/箭头/类/async；支持 Jest/Vitest/Mocha 断言风格）
│   │   ├── py_generator.go           # Python pytest 测试生成器（def/class/async）
│   │   ├── rs_generator.go           # Rust 测试生成器
│   │   └── java_generator.go         # Java JUnit 4/5 测试生成器
│   ├── parser/
│   │   ├── parser.go                # 统一解析入口
│   │   ├── go_parser.go             # go test 输出解析
│   │   ├── jest_parser.go           # Jest 输出解析
│   │   ├── pytest_parser.go         # pytest 输出解析
│   │   └── mocha_parser.go          # Mocha 输出解析
│   ├── coverage/
│   │   ├── coverage.go              # 统一入口 + 改进建议生成
│   │   ├── go_coverage.go           # Go coverprofile 解析
│   │   ├── jest_coverage.go         # Jest/Istanbul coverage JSON 解析
│   │   ├── pytest_coverage.go       # coverage.py JSON 解析
│   │   ├── rust_coverage.go         # cargo tarpaulin LCOV 解析
│   │   └── java_coverage.go         # JaCoCo XML 解析
│   └── detector/
│       └── detector.go              # 框架自动检测（package.json/pyproject.toml/go.mod）
├── cmd/
│   └── testgen/main.go              # 独立 CLI 工具，脱离 MCP 直接生成测试
├── demo/                            # 示例代码（calc, service, advanced）
├── Dockerfile                       # 多阶段构建（Go builder → alpine runtime）
├── docker-compose.yml               # HTTP 模式一键部署
└── .dockerignore
```

## 开发

```bash
# 安装依赖
go mod tidy

# 构建
go build -o testloop-mcp .

# 运行全部测试
go test ./...

# 仅运行覆盖率解析测试
go test ./internal/coverage/ -v

# 仅运行解析器测试
go test ./internal/parser/ -v

# 用 CLI 工具对指定文件生成测试（脱离 MCP）
go run ./cmd/testgen demo/calc.go
go run ./cmd/testgen -provider llm -provider-check
go run ./cmd/testgen -provider auto demo/calc.py /tmp/test_calc.py

# 启动 MCP server
go run main.go                          # stdio 模式（默认）
go run main.go --transport http --addr :8080  # Streamable HTTP 模式

# Docker 部署
docker compose up -d                   # HTTP 模式，监听 :8080
curl http://localhost:8080/healthz     # 健康检查
docker compose logs -f                 # 查看日志
docker compose down                    # 停止
```

## 技术栈

- **语言：** Go 1.25+
- **MCP SDK：** [github.com/modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) v1.6.1（官方 SDK）
- **测试生成：** Go 优先复用 `gotests`，并以内置 `go/ast`、`go/parser`、`go/token`、`go/format` 作为回退；其他语言使用 tree-sitter/轻量解析器
- **传输层：** stdio（JSON-RPC over stdin/stdout）+ Streamable HTTP（`--transport http`）
- **部署：** GitHub Release 二进制 + Docker 多阶段构建（alpine 基础镜像，~8MB 二进制）

## 真实项目验证脚本

仓库提供 opt-in 脚本，用于把真实项目的覆盖率缺口批量送入 `validate_coverage_task`，复用 `coverage_task -> generate_tests -> run_tests` 闭环。Go 项目可使用：

```bash
scripts/validate-go-coverage-top-tasks.sh /path/to/go/project 20 /tmp/testloop-go-top20.jsonl
```

JS/Vitest/Jest/Mocha 项目可使用：

```bash
TESTLOOP_VALIDATE_JS_TEST_ARGS='tests/env-resolver.test.js tests/config.test.js' \
TESTLOOP_VALIDATE_JS_FILE_FILTER='src/utils/' \
scripts/validate-js-coverage-top-tasks.sh /path/to/js/project vitest 10 /tmp/testloop-js-top10.jsonl
```

JS 脚本会读取 Istanbul `coverage/coverage-final.json`；如果项目覆盖率阈值导致 baseline coverage 命令非零退出，但覆盖率 JSON 已生成，脚本仍会继续解析并验证任务。

Python/pytest 项目可使用：

```bash
PYTHONPATH=src \
TESTLOOP_VALIDATE_PY_COVERAGE_COMMAND='python3 -m pytest --cov=src --cov-report=json {args}' \
scripts/validate-py-coverage-top-tasks.sh /path/to/python/project 20 /tmp/testloop-py-top20.jsonl
```

Python 脚本会读取 pytest-cov 生成的 `coverage.json`，并在隔离副本中逐个验证 coverage task。常用变量包括 `TESTLOOP_VALIDATE_PY_TEST_ARGS`、`TESTLOOP_VALIDATE_PY_FILE_FILTER`、`TESTLOOP_VALIDATE_PY_STAGE_TIMEOUT_SECONDS` 和 `TESTLOOP_VALIDATE_PY_TASK_TIMEOUT_SECONDS`。

Rust/Cargo 项目可使用：

```bash
TESTLOOP_VALIDATE_RUST_COVERAGE_COMMAND='cargo llvm-cov --lcov --output-path target/llvm-cov/lcov.info' \
TESTLOOP_VALIDATE_RUST_COVERAGE_FILE='target/llvm-cov/lcov.info' \
scripts/validate-rust-coverage-top-tasks.sh /path/to/rust/project 20 /tmp/testloop-rust-top20.jsonl
```

Rust 脚本会读取 LCOV 文件，默认命令为 `cargo tarpaulin --out Lcov --output-dir target/tarpaulin`。当前也支持通过环境变量接入 `cargo llvm-cov` 或项目自定义覆盖率命令。

Java/Maven 或 Gradle 项目可使用：

```bash
scripts/validate-java-coverage-top-tasks.sh /path/to/java/project 20 /tmp/testloop-java-top20.jsonl
```

Java 脚本会读取 JaCoCo XML，默认检测 `target/site/jacoco/jacoco.xml` 或 `build/reports/jacoco/test/jacocoTestReport.xml`。JS/Python/Java 脚本都支持 `TESTLOOP_VALIDATE_*_TASK_IDS` 按 task id 精确筛选，也支持 `TESTLOOP_VALIDATE_*_TASKS_FILE` 从已有 coverage task / validation JSONL 读取任务并跳过 baseline coverage。Java 常用变量包括 `TESTLOOP_VALIDATE_JAVA_COVERAGE_COMMAND`、`TESTLOOP_VALIDATE_JAVA_COVERAGE_FILE`、`TESTLOOP_VALIDATE_JAVA_FILE_FILTER`、`TESTLOOP_VALIDATE_JAVA_TASK_IDS`、`TESTLOOP_VALIDATE_JAVA_TASKS_FILE`、`TESTLOOP_VALIDATE_JAVA_STAGE_TIMEOUT_SECONDS` 和 `TESTLOOP_VALIDATE_JAVA_TASK_TIMEOUT_SECONDS`。如果只想回归特定任务，可以使用逗号分隔的 task id：

```bash
TESTLOOP_VALIDATE_JAVA_TASK_IDS='junit-44,junit-130' \
scripts/validate-java-coverage-top-tasks.sh /path/to/java/project
```

如果已经有 `TESTLOOP_VALIDATE_JAVA_LIST_TASKS_ONLY=true` 导出的 task JSONL，或已有验证结果 JSONL，可以直接复用它跳过 baseline coverage：

```bash
TESTLOOP_VALIDATE_JAVA_TASKS_FILE=/tmp/testloop-java-tasks.jsonl \
TESTLOOP_VALIDATE_JAVA_TASK_IDS='junit-44' \
scripts/validate-java-coverage-top-tasks.sh /path/to/java/project /tmp/testloop-java-junit44.jsonl
```

验证单个 Java coverage task 时，`run_tests` 会优先只运行生成的 `*TestLoopTest` 测试类，并继续生成 JaCoCo report，因此目标行命中校验仍然有效。

如果要快速回归当前固定的 Java 质量样本，可以使用：

```bash
scripts/validate-java-regression-samples.sh
```

该脚本默认复用 `/tmp/testloop-commons-lang`、`/tmp/testloop-commons-codec` 以及已有 JSONL 任务文件，覆盖三类样本：真实 ready 且命中目标行、历史假 ready 降级为 `manual_review_unreachable`、内部路径 `manual_review_internal`。项目目录或 JSONL 路径不一致时，可通过 `TESTLOOP_JAVA_REGRESSION_*` 环境变量覆盖。

如果要运行当前维护的固定 smoke 矩阵，可以使用：

```bash
scripts/validate-regression-smoke.sh
```

当前默认矩阵覆盖 Java + JS + Python：Java 使用上述三类样本，JS 使用 ip2region JavaScript binding 的 Jest ready 样本和仓库内 TypeScript no-runtime 手审样本，Python 使用 Click pytest ready 样本。各语言项目目录或 JSONL 路径不一致时，可通过 `TESTLOOP_*_REGRESSION_*` 环境变量覆盖。详细运行说明见 [固定 smoke 回归说明](./docs/regression-smoke.md)。

真实项目样本和当前质量边界见 [真实项目验证质量报告](./docs/real-project-validation.md)。

## Roadmap

当前版本已经覆盖 stdio / Streamable HTTP MCP 服务、多语言测试生成、测试执行、失败解析、修复建议、覆盖率解析、Docker 部署和 GitHub Release / Homebrew 分发。

后续路线图和已完成阶段见 [docs/plan-roadmap.md](./docs/plan-roadmap.md)。

## License

MIT
