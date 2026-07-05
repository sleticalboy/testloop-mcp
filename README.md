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
        └── parse_coverage    → 覆盖率数据 → 报告 + 改进建议
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

> 测试生成：Go 优先使用 `gotests`，失败时回退内置 `go/ast`；JS/TS/Python/Rust/Java 基于 tree-sitter/轻量解析器。传入 `coverage_task` 时，Go/Python/Jest/Rust/Java 静态生成器会优先聚焦任务目标函数或方法，使用任务推荐测试名，并把建议输入代入生成的测试草稿。
> 覆盖率：当前支持 Go coverprofile、Istanbul coverage JSON（Jest/Vitest/Mocha）、coverage.py JSON、Rust `cargo tarpaulin --out Lcov` 生成的 LCOV，以及 Java JaCoCo XML。Go/Rust/Java 覆盖率建议会尽量定位到具体函数或方法，并输出常见分支/返回/错误路径分类，便于 AI Agent 直接补测试。

## 安装

推荐使用安装脚本。脚本会优先下载当前平台匹配的 GitHub Release 资产；当前 release 没有匹配资产时，会自动回退到 `go install`：

```bash
curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/install.sh | sh
```

当前 `v0.4.1` Release 已提供 Linux amd64 二进制：

```bash
curl -LO https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.1/testloop-mcp_v0.4.1_linux_amd64.tar.gz
curl -LO https://github.com/sleticalboy/testloop-mcp/releases/download/v0.4.1/checksums.txt
sha256sum -c checksums.txt
tar -xzf testloop-mcp_v0.4.1_linux_amd64.tar.gz
./testloop-mcp --help
```

后续 release workflow 已准备生成 Linux amd64、Linux arm64 和 macOS arm64 资产。Windows 或需要从源码构建：

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

根据源文件生成测试代码。支持 Go（优先 `gotests`，回退内置 AST 分析）、Rust、Java、JavaScript/TypeScript（Jest）、Python（pytest）。

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `file_path` | string | ✅ | 源文件路径（`.go` / `.rs` / `.java` / `.js` / `.ts` / `.jsx` / `.tsx` / `.py`） |
| `framework` | string | — | 测试框架，默认根据文件扩展名自动选择 |
| `provider` | string | — | 测试生成 provider：`static` / `llm` / `auto`，默认 `static` |
| `coverage_task` | object | — | `parse_coverage` 返回的单个 `test_tasks` 项，用于按覆盖率缺口生成测试 |

**返回：** `{ status, test_file, generated_cases, preview, context, coverage_task, provider }`

传入 `coverage_task` 时，工具会优先写入任务中的 `test_file`，并把任务回写到返回的 `context.coverage_task`。内置 static provider 会在 Go/Python/Jest/Rust/Java 中按目标函数或方法收窄生成范围，使用任务推荐测试名，把 `assertion_focus` 和 `suggested_inputs` 写入注释，并从建议输入中的条件表达式提取参数值生成更贴近覆盖率缺口的调用。LLM provider 也会收到同一份 task 上下文，便于在静态草稿基础上进一步增强断言。

**LLM provider：** 默认不依赖任何外部 LLM。需要启用时，在服务端配置 `TESTLOOP_LLM_PROVIDER_CMD`，并调用 `generate_tests` 时传 `provider: "llm"` 或 `provider: "auto"`。命令会从 stdin 接收 JSON（`source_file`、`context`、`static_code`），其中 `context.coverage_task` 会携带覆盖率任务上下文；stdout 可以直接返回测试代码，也可以返回 `{"code":"..."}`。`auto` 在未配置命令时会自动回退到 `static`。

LLM provider 示例见 [docs/llm-provider.md](./docs/llm-provider.md) 和 [examples/llm-provider.sh](./examples/llm-provider.sh)。

**Go 生成器：** 优先调用本机 `gotests -all` 生成 Go 社区标准测试骨架；如果未安装 `gotests`、命令失败或输出为空，则回退到内置 `go/ast` 生成器。内置回退支持泛型类型参数实例化（`T → int`）、指针/值接收者方法、变参 `...T` → 切片、通道参数 nil-check + `t.Skip` 防阻塞、接口参数自动 mock、slice/map/struct 自动使用 `reflect.DeepEqual`。

**JS/TS 生成器：** tree-sitter + 函数体分析，识别函数、类方法、async、参数、CommonJS / ES Module 导入，分析 `return` 语句推断返回类型（number/string/array/object/boolean）、检测 `throw` 生成 `toThrow()` 测试、检测 `if (param === value)` 边界条件生成针对性用例。

**Python 生成器：** tree-sitter + 函数体分析，识别函数、类方法、async、参数、`@staticmethod`、`*args`/`**kwargs`，分析 `return` 语句推断返回类型（int/float/str/list/dict/bool）、检测 `raise` 生成 `pytest.raises()` 测试、检测 `if param == value` 边界条件。

**覆盖率驱动生成闭环：**

1. 用 `run_tests` 或生态命令生成覆盖率报告。
2. 调用 `parse_coverage` 获取 `test_tasks`，每个任务包含目标、缺口类型、推荐测试文件、测试名、建议输入和断言重点。
3. 取单个 `test_tasks[]` 作为 `generate_tests.coverage_task` 传入，生成面向该缺口的增量测试草稿。
4. 调用 `run_tests` 重新执行测试，必要时把失败交给 `parse_results` / `fix_suggestions` 继续闭环。

---

### `run_tests`

执行测试并返回结构化结果。

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `path` | string | ✅ | 测试文件或目录路径 |
| `framework` | string | — | `go-test` / `cargo-test` / `jest` / `vitest` / `mocha` / `pytest` / `junit`，默认自动检测 |
| `coverage` | bool | — | 是否收集覆盖率，默认 `false` |
| `verbose` | bool | — | 详细输出，默认 `true` |

**返回：** `{ status, framework, total, passed, failed, skipped, coverage_percent, failures[], raw_output }`

`coverage=true` 时，Rust 会额外调用 `cargo tarpaulin --out Lcov --output-dir target/tarpaulin` 并回填 `coverage_percent`；Java Maven/Gradle 项目会执行 JaCoCo report 任务并从 XML 报告回填 `coverage_percent`。也可以通过 `parse_coverage` 直接解析已有 LCOV/JaCoCo XML 文件。

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

**返回：** `[{ file, line, issue, suggested_fix, confidence }]`

识别的失败类型：期望值不匹配（`got X, want Y`）、nil pointer panic、数组越界、除零错误、未定义引用、类型不匹配。

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
├── main.go                          # MCP server 入口，注册 5 个工具
├── go.mod                           # github.com/sleticalboy/testloop-mcp, go 1.25
├── types/
│   └── types.go                     # 所有共享类型定义
├── tools/
│   ├── run_tests.go                 # run_tests 工具 + Register() 注册入口
│   ├── generate_tests.go            # generate_tests 工具
│   ├── parse_results.go             # parse_results 工具
│   ├── fix_suggestions.go           # fix_suggestions 工具
│   └── parse_coverage.go            # parse_coverage 工具
├── internal/
│   ├── generator/
│   │   ├── generator.go              # 多语言静态生成分发入口
│   │   ├── provider.go               # 测试生成 provider 接口 + 可选 LLM command provider
│   │   ├── context.go                # 面向 LLM/AI Agent 的测试生成上下文
│   │   ├── go_gotests.go             # gotests 优先生成器
│   │   ├── go_generator.go           # Go AST 回退测试生成器（泛型/通道/接口/变参）
│   │   ├── js_generator.go           # JS/TS Jest 测试生成器（函数/箭头/类/async）
│   │   ├── py_generator.go           # Python pytest 测试生成器（def/class/async）
│   │   ├── rs_generator.go           # Rust 测试生成器
│   │   └── java_generator.go         # Java JUnit 5 测试生成器
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

## Roadmap

- [x] MCP 服务器骨架（stdio + Streamable HTTP 传输）
- [x] Go 测试生成器（优先 gotests，回退 AST → 表驱动测试）
- [x] 泛型 / 通道 / 接口 / 变参 / `reflect.DeepEqual` 支持
- [x] JavaScript/TypeScript 测试生成器（函数/箭头/类/async → Jest）
- [x] Python 测试生成器（def/class/async → pytest）
- [x] `go test` / Jest / Vitest / Mocha / pytest 执行器
- [x] 测试输出解析器（5 框架）
- [x] `fix_suggestions` 修复建议（6 种失败类型）
- [x] 覆盖率解析（Go / Jest / Vitest / Mocha / pytest / Rust tarpaulin LCOV / Java JaCoCo XML）
- [x] 框架自动检测（package.json scripts/dependencies + pyproject.toml + go.mod，向上递归查找）
- [x] Docker 部署（多阶段构建 + docker-compose）
- [ ] VS Code Extension 配套

## License

MIT
