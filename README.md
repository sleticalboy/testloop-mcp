# testloop-mcp

[![Go Report Card](https://goreportcard.com/badge/github.com/binlee/testloop-mcp)](https://goreportcard.com/report/github.com/binlee/testloop-mcp)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**testloop-mcp** 是一个基于 [MCP (Model Context Protocol)](https://modelcontextprotocol.io) 的智能测试生成与执行反馈闭环服务器。让 AI Coding 工具（Claude Code / Cursor / VS Code Copilot 等）能够自动生成测试、执行测试、解析失败原因、生成修复建议，并分析覆盖率——形成完整的测试闭环。

## 核心能力

- **智能生成测试** — 多语言 AST 分析（Go `go/ast` + tree-sitter Rust/Java/JS/TS/Python），自动生成类型感知测试。支持泛型、async、Result/Option 返回类型、JUnit 5 断言
- **执行测试** — 支持 `go test` / `cargo test` / Jest / Vitest / Mocha / pytest / JUnit 5（Maven/Gradle），自动检测项目类型，可选收集覆盖率
- **解析失败** — 结构化解析测试输出（Go/`cargo test`/Jest/Vitest/Mocha/JUnit），提取失败用例的文件、行号、错误信息，AI 友好 JSON 格式
- **修复建议** — 根据失败类型（期望值不匹配 / nil pointer / 数组越界 / 除零 / 类型不匹配等）生成结构化修复建议
- **覆盖率分析** — 解析 Go coverprofile / Jest coverage JSON / pytest coverage JSON，输出文件级覆盖率、未覆盖 block 定位和改进建议

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
  本地项目（Go / Node.js / Python）
```

## 支持的框架

| 语言 | 测试框架 | 生成 | 执行 | 解析 | 覆盖率 |
|------|---------|:----:|:----:|:----:|:------:|
| Go | `go test` | ✅ | ✅ | ✅ | ✅ |
| Rust | `cargo test` | ✅ | ✅ | ✅ | 🔲 |
| Node.js | Jest | ✅ | ✅ | ✅ | ✅ |
| Node.js | Vitest | ✅ | ✅ | ✅ | ✅ |
| Node.js | Mocha | ✅ | ✅ | ✅ | ✅ |
| Python | pytest | ✅ | ✅ | ✅ | ✅ |
| Java | JUnit 5 (Maven/Gradle) | ✅ | ✅ | ✅ | 🔲 |

> 测试生成：Go 基于 `go/ast` 原生 AST；JS/TS/Python/Rust/Java 基于 tree-sitter 多语言 AST 解析。
> 覆盖率：Rust (`cargo tarpaulin`)、Java (JaCoCo) 的支持正在路上。

## 安装

```bash
git clone https://github.com/binlee/testloop-mcp.git
cd testloop-mcp
go build -o testloop-mcp .
```

**前置要求：** Go 1.25+

## 配置接入

### Claude Code / Claude Desktop

`~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "testloop": {
      "command": "/path/to/testloop-mcp"
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
      "command": "path/to/testloop-mcp"
    }
  }
}
```

## MCP Tools

### `generate_tests`

根据源文件生成测试代码。支持 Go（优先 `gotests`，回退内置 AST 分析）、JavaScript/TypeScript（Jest）、Python（pytest）。

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `file_path` | string | ✅ | 源文件路径（`.go` / `.js` / `.ts` / `.jsx` / `.tsx` / `.py`） |
| `framework` | string | — | 测试框架，默认根据文件扩展名自动选择 |
| `provider` | string | — | 测试生成 provider：`static` / `llm` / `auto`，默认 `static` |

**返回：** `{ status, test_file, generated_cases, preview, context, provider }`

**LLM provider：** 默认不依赖任何外部 LLM。需要启用时，在服务端配置 `TESTLOOP_LLM_PROVIDER_CMD`，并调用 `generate_tests` 时传 `provider: "llm"` 或 `provider: "auto"`。命令会从 stdin 接收 JSON（`source_file`、`context`、`static_code`），stdout 可以直接返回测试代码，也可以返回 `{"code":"..."}`。`auto` 在未配置命令时会自动回退到 `static`。

**Go 生成器：** 优先调用本机 `gotests -all` 生成 Go 社区标准测试骨架；如果未安装 `gotests`、命令失败或输出为空，则回退到内置 `go/ast` 生成器。内置回退支持泛型类型参数实例化（`T → int`）、指针/值接收者方法、变参 `...T` → 切片、通道参数 nil-check + `t.Skip` 防阻塞、接口参数自动 mock、slice/map/struct 自动使用 `reflect.DeepEqual`。

**JS/TS 生成器：** 正则 + 花括号匹配解析函数体，分析 `return` 语句推断返回类型（number/string/array/object/boolean）、检测 `throw` 生成 `toThrow()` 测试、检测 `if (param === value)` 边界条件生成针对性用例。支持 async 函数（`await` + `resolves`）、箭头函数表达式体、CommonJS / ES Module 导入。

**Python 生成器：** 正则 + 缩进感知解析函数体，分析 `return` 语句推断返回类型（int/float/str/list/dict/bool）、检测 `raise` 生成 `pytest.raises()` 测试、检测 `if param == value` 边界条件。支持 `async def`（`asyncio.run()`）、`*args`/`**kwargs`、`@staticmethod`。

---

### `run_tests`

执行测试并返回结构化结果。

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `path` | string | ✅ | 测试文件或目录路径 |
| `framework` | string | — | `go-test` / `jest` / `vitest` / `mocha` / `pytest`，默认自动检测 |
| `coverage` | bool | — | 是否收集覆盖率，默认 `false` |
| `verbose` | bool | — | 详细输出，默认 `true` |

**返回：** `{ status, framework, total, passed, failed, skipped, coverage_percent, failures[], raw_output }`

---

### `parse_results`

解析测试执行输出，提取失败用例详情。

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `output` | string | ✅ | 测试执行的标准输出/错误输出原文 |
| `framework` | string | — | 测试框架，默认 `go-test` |

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
| `data` | string | ✅ | 覆盖率数据（coverprofile 文件路径/内容、Jest coverage JSON、pytest coverage JSON） |
| `framework` | string | — | `go-test` / `jest` / `pytest`，默认 `go-test` |

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
  ]
}
```

## 项目结构

```
testloop-mcp/
├── main.go                          # MCP server 入口，注册 5 个工具
├── go.mod                           # github.com/binlee/testloop-mcp, go 1.25
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
│   │   ├── generator.go              # 多语言分发入口（按扩展名路由）
│   │   ├── go_generator.go           # Go AST 测试生成器（泛型/通道/接口/变参）
│   │   ├── js_generator.go           # JS/TS Jest 测试生成器（函数/箭头/类/async）
│   │   └── py_generator.go           # Python pytest 测试生成器（def/class/async）
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
│   │   └── pytest_coverage.go       # coverage.py JSON 解析
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

# 启动 MCP server
go run main.go                          # stdio 模式（默认）
go run main.go --transport http --addr :8080  # Streamable HTTP 模式

# Docker 部署
docker compose up -d                   # HTTP 模式，监听 :8080
docker compose logs -f                 # 查看日志
docker compose down                    # 停止
```

## 技术栈

- **语言：** Go 1.25+
- **MCP SDK：** [github.com/modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) v1.6.1（官方 SDK）
- **测试生成：** Go 优先复用 `gotests`，并以内置 `go/ast`、`go/parser`、`go/token`、`go/format` 作为回退；其他语言使用 tree-sitter/轻量解析器
- **传输层：** stdio（JSON-RPC over stdin/stdout）+ Streamable HTTP（`--transport http`）
- **部署：** Docker 多阶段构建（alpine 基础镜像，~8MB 二进制）

## Roadmap

- [x] MCP 服务器骨架（stdio + Streamable HTTP 传输）
- [x] Go 测试生成器（优先 gotests，回退 AST → 表驱动测试）
- [x] 泛型 / 通道 / 接口 / 变参 / `reflect.DeepEqual` 支持
- [x] JavaScript/TypeScript 测试生成器（函数/箭头/类/async → Jest）
- [x] Python 测试生成器（def/class/async → pytest）
- [x] `go test` / Jest / Vitest / Mocha / pytest 执行器
- [x] 测试输出解析器（5 框架）
- [x] `fix_suggestions` 修复建议（6 种失败类型）
- [x] 覆盖率解析（Go / Jest / Vitest / Mocha / pytest）
- [x] 框架自动检测（package.json scripts/dependencies + pyproject.toml + go.mod，向上递归查找）
- [x] Docker 部署（多阶段构建 + docker-compose）
- [ ] VS Code Extension 配套

## License

MIT
