# testloop-mcp 设计文档

## 1. 项目定位

testloop-mcp 是 AI Coding 工作流中「**写代码 → 验证 → 修复**」闭环的缺失环节。

当前状态：
- AI 能写代码 ✅（Claude/Cursor/GitHub Copilot）
- AI 能解释测试失败 ✅（有上下文时）
- **AI 不能自动触发测试、解析失败、驱动修复 ❌** ← 本工具补位

---

## 2. MCP 工具设计

### Tool 1：`generate_tests`

**目标**：给定源文件，生成对应的测试文件。支持 Go / Rust / Java / JavaScript / TypeScript / Python。

**输入：**
```json
{
  "file_path": "src/calc.rs",
  "framework": "cargo-test",
  "provider": "static"
}
```

**处理逻辑：**
1. 根据 `provider` 选择测试生成路径：`static`（默认）、`llm`、`auto`。
2. 静态路径按文件扩展名分发：`.go` → 优先 `gotests -all`，失败回退内置 Go AST；`.rs` → tree-sitter Rust；`.java` → tree-sitter Java；`.js`/`.ts` → tree-sitter JS/TS；`.py` → tree-sitter Python。
3. 提取 functions / methods / class methods / impl 块方法 / trait 方法，以及导入、邻近类型、返回表达式、错误路径和边界条件。
4. 根据返回类型（Result/Option/async）生成类型感知测试。
5. 当 `provider` 为 `llm` 或 `auto` 且配置了 `TESTLOOP_LLM_PROVIDER_CMD` 时，把源码上下文和静态生成结果传给外部 LLM provider，由 provider 返回最终测试代码。
6. 写入测试文件（`_test.go` / `test.rs` / `Test.java` / `.test.js` / `test_*.py`）。

**输出：**
```json
{
  "status": "ok",
  "test_file": "internal/calc/calc_test.go",
  "generated_cases": 12,
  "preview": "...",
  "context": { "language": "typescript", "targets": [] },
  "provider": "static"
}
```

**Go 生成器：** 优先调用本机 `gotests -all` 生成 Go 社区标准测试骨架；如果未安装 `gotests`、命令失败或输出为空，则回退到内置 `go/ast` 生成器。内置回退支持泛型类型参数实例化（`T → int`）、指针/值接收者方法、变参 `...T` → 切片、通道参数 nil-check + `t.Skip` 防阻塞、接口参数自动 mock、slice/map/struct 自动使用 `reflect.DeepEqual`。

**LLM provider：** 默认不依赖任何外部 LLM。需要启用时，在服务端配置 `TESTLOOP_LLM_PROVIDER_CMD`，并调用 `generate_tests` 时传 `provider: "llm"` 或 `provider: "auto"`。provider 从 stdin 接收 `{ source_file, context, static_code }`，stdout 可以直接返回测试代码，也可以返回 `{"code":"..."}`。

**JS/TS 生成器：** 正则解析函数声明、箭头函数、类方法，自动检测 CommonJS / ES Module 导入方式。支持 async 函数、变参 `...args`、默认值参数、TypeScript 类型注解剥离。

**Python 生成器：** 正则解析 `def`/`async def`/`class` 声明，自动剥离 `self`/`cls` 参数、类型注解、默认值。支持 `*args`/`**kwargs`、`@staticmethod`、pytest.raises 异常测试。

**Rust 生成器：** 基于 tree-sitter Rust grammar 解析 `fn`/impl 块方法/trait 方法，自动推断 `Result<T,E>`/`Option<T>` 返回类型并生成 `Ok`/`Err`/`Some`/`None` 分支测试，支持 `#[test]` 和 `#[tokio::test]` async 测试。

**Java 生成器：** 基于 tree-sitter Java grammar 解析 `class`/`method`/`constructor`，自动检测 `public`/`static` 修饰符，生成 JUnit 5 `@Test` 测试方法，支持 `assertAll`/`assertThrows` 异常测试。

---

### Tool 2：`run_tests`

**目标**：执行测试，返回结构化结果。

**输入：**
```json
{
  "path": "internal/calc/",
  "framework": "go test",
  "coverage": true,
  "verbose": true
}
```

**处理逻辑：**
1. 根据 `framework` 选择执行器（`go-test` / `cargo-test` / `jest` / `vitest` / `mocha` / `pytest` / `junit`）
2. 自动检测项目类型（存在 `go.mod` → go-test，`Cargo.toml` → cargo-test，`package.json` → Jest，等等）
3. 执行命令，捕获 stdout/stderr
4. 调用 `parser` 解析输出
5. 返回结构化 JSON

**输出：**
```json
{
  "status": "fail",
  "framework": "go-test",
  "total": 11,
  "passed": 8,
  "failed": 3,
  "skipped": 0,
  "coverage_percent": 67.2,
  "failures": [
    {
      "test_name": "TestAdd/negative_inputs",
      "file": "calc_test.go",
      "line": 42,
      "error": "got -1, want 0"
    }
  ],
  "raw_output": "..."
}
```

---

### Tool 3：`parse_results`

**目标**：将原始测试输出解析为结构化失败信息（AI 友好）。

**输入：** 测试执行的原始 stdout/stderr + 框架名

**输出：** 同 `run_tests` 的 failures 结构（可独立使用）

支持 7 种框架的输出解析：`go test` / `cargo test` / Jest / Vitest / Mocha / pytest / JUnit。

---

### Tool 4：`fix_suggestions`

**目标**：将失败信息 + 源代码结构化打包，生成修复建议供 AI 消费。

> 注意：此工具本身不调用 AI，而是将上下文**结构化打包**，让调用方（Claude/Cursor）直接使用。

**输入：**
```json
{
  "failures": "[...]",
  "source_code": "internal/calc/calc.go",
  "test_code": "internal/calc/calc_test.go"
}
```

**输出：**
```json
{
  "suggestions": [
    {
      "file": "calc.go",
      "line": 15,
      "issue": "Add 函数未处理负数溢出",
      "suggested_fix": "添加溢出检查：if a > math.MaxInt64 - b { return 0, ErrOverflow }",
      "confidence": 0.92
    }
  ]
}
```

识别的失败类型：期望值不匹配（`got X, want Y`）、nil pointer panic、数组越界、除零错误、未定义引用、类型不匹配。

---

### Tool 5：`parse_coverage`

**目标**：解析覆盖率数据，返回结构化报告和改进建议。

**输入：**
```json
{
  "data": "mode: set\ncalc.go:1.1,3.2 1 1\n...",
  "framework": "go-test"
}
```

**输出：**
```json
{
  "framework": "go-test",
  "total_percent": 58.8,
  "files": [
    {
      "path": "calc.go",
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
    { "file": "calc.go", "line_range": "5-7", "reason": "此代码块未被测试覆盖", "confidence": 0.9 }
  ]
}
```

支持 Go coverprofile / Jest (Istanbul) coverage JSON / pytest coverage JSON 三种格式。

---

## 3. 内部模块设计

```
testloop-mcp/
├── main.go                          # 入口，启动 MCP server（stdio / Streamable HTTP）
├── go.mod                           # github.com/binlee/testloop-mcp, go 1.25
├── types/
│   └── types.go                     # 公共类型定义（TestResult/TestFailure/FixSuggestion/CoverageReport 等）
├── tools/
│   ├── run_tests.go                 # run_tests 工具 + Register() 注册所有工具
│   ├── generate_tests.go            # generate_tests 工具
│   ├── parse_results.go             # parse_results 工具
│   ├── fix_suggestions.go           # fix_suggestions 工具
│   └── parse_coverage.go            # parse_coverage 工具
├── internal/
│   ├── generator/
│   │   ├── generator.go             # 多语言分发入口（按扩展名路由 .go/.rs/.java/.js/.ts/.py）
│   │   ├── provider.go              # 测试生成 provider 接口 + 可选 LLM command provider
│   │   ├── go_gotests.go            # gotests 优先生成器，失败回退内置 Go AST
│   │   ├── go_generator.go          # Go AST 回退测试生成器（泛型/通道/接口/变参）
│   │   ├── context.go               # 面向 LLM/AI Agent 的测试生成上下文
│   │   ├── ts_parser.go            # JS/TS tree-sitter AST 解析器（函数/箭头/类/async）
│   │   ├── js_generator.go          # JS/TS Jest 测试生成器（类型推断/throw/边界条件）
│   │   ├── py_generator.go          # Python pytest 测试生成器（def/class/async/*args）
│   │   ├── rs_parser.go            # Rust tree-sitter AST 解析器（fn/impl/trait/async）
│   │   ├── rs_generator.go          # Rust #[test] 测试生成器（Result/Option/async）
│   │   ├── java_parser.go          # Java tree-sitter AST 解析器（class/method/constructor）
│   │   └── java_generator.go        # Java JUnit 5 测试生成器（@Test/assertAll/assertThrows）
│   ├── parser/
│   │   ├── parser.go                # 统一解析入口（按框架名分发）
│   │   ├── go_parser.go             # go test 输出解析
│   │   ├── jest_parser.go         # Jest 输出解析
│   │   ├── pytest_parser.go         # pytest 输出解析
│   │   ├── mocha_parser.go          # Mocha 输出解析
│   │   ├── rust.go                 # cargo test 输出解析
│   │   └── java.go                 # JUnit 输出解析（Maven/Gradle/JUnit5）
│   ├── coverage/
│   │   ├── coverage.go              # 统一入口 + 改进建议生成
│   │   ├── go_coverage.go           # Go coverprofile 解析
│   │   ├── jest_coverage.go         # Jest/Istanbul coverage JSON 解析（Vitest/Mocha 共用）
│   │   └── pytest_coverage.go       # coverage.py JSON 解析
│   └── detector/
│       └── detector.go              # 框架自动检测（package.json/pyproject.toml/go.mod，向上递归）
├── cmd/
│   └── testgen/main.go              # 独立 CLI 工具，脱离 MCP 直接生成测试
├── demo/                            # 示例代码（calc, service, advanced）
├── Dockerfile                       # 多阶段构建（Go builder → alpine runtime）
└── docker-compose.yml               # HTTP 模式一键部署
```

---

## 4. 传输层

支持两种 MCP 传输方式：

| 模式 | 参数 | 用途 |
|------|------|------|
| stdio | `--transport stdio`（默认） | 生产使用，接入 Claude/Cursor |
| Streamable HTTP | `--transport http --addr :8080` | 远程部署、多客户端、Web IDE 集成 |

> 两种模式均已实现。HTTP 模式基于 go-sdk 的 `StreamableHTTPHandler`，支持有状态（默认）和无状态（`--stateless`）两种会话模式。
>
> **Docker 部署：** `docker compose up -d` 一键启动 HTTP 模式。多阶段构建（Go builder → alpine runtime），最终镜像约 15MB。

---

## 5. 技术选型

| 组件 | 选择 | 理由 |
|------|------|------|
| MCP SDK | `github.com/modelcontextprotocol/go-sdk` v1.6.1 | MCP 官方 Go SDK |
| Go 版本 | 1.25+ | 泛型支持，标准库足够 |
| 测试解析 | 正则 + 行解析 | go test / Jest / pytest 输出格式稳定，无需复杂 Parser |
| 测试生成 | `gotests` + provider 接口 + `go/ast` + tree-sitter + 模板 | Go 优先复用成熟社区工具；LLM 可选接入；静态生成保持无外部强依赖 |
| 覆盖率解析 | 正则 + JSON 解析 | 各框架覆盖率格式不同，按框架分发 |
| 配置 | 命令行参数 + 环境变量 | 保持简单，MCP 本身无状态 |

---

## 6. 核心类型定义（types/types.go）

```go
// TestResult 单次测试执行结果
type TestResult struct {
    Status          string        `json:"status"`
    Framework       string        `json:"framework,omitempty"`
    Total           int           `json:"total,omitempty"`
    Passed          int           `json:"passed"`
    Failed          int           `json:"failed"`
    Skipped         int           `json:"skipped"`
    CoveragePercent float64       `json:"coverage_percent"`
    Failures        []TestFailure `json:"failures"`
    RawOutput       string        `json:"raw_output"`
}

// TestFailure 单个测试失败详情
type TestFailure struct {
    TestName string `json:"test_name"`
    File     string `json:"file"`
    Line     int    `json:"line"`
    Error    string `json:"error"`
}

// FixSuggestion 修复建议
type FixSuggestion struct {
    File         string  `json:"file"`
    Line         int     `json:"line"`
    Issue        string  `json:"issue"`
    SuggestedFix string  `json:"suggested_fix"`
    Confidence   float64 `json:"confidence"`
}

// CoverageReport 覆盖率报告
type CoverageReport struct {
    Framework    string               `json:"framework"`
    TotalPercent float64              `json:"total_percent"`
    Files        []CoverageFile       `json:"files"`
    Summary      CoverageSummary      `json:"summary"`
    Suggestions  []CoverageSuggestion `json:"suggestions,omitempty"`
}
```

---

## 7. 与非 AI 工具的边界

testloop-mcp **不做**的事：
- ❌ 不替代 `go test` / Jest / pytest——只是调用它们
- ❌ 不自己修复代码——只生成结构化修复建议供 AI 消费
- ❌ 不管理测试数据——测试数据由项目方维护

testloop-mcp **专注**的事：
- ✅ 桥接 AI 和测试执行环境
- ✅ 把测试结果变成 AI 能高效消费的 JSON
- ✅ 打通「生成 → 执行 → 解析 → 建议 → 覆盖率」闭环
