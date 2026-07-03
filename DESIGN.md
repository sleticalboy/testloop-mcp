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

**目标**：给定源文件，生成对应的测试文件。支持 Go / JavaScript / TypeScript / Python。

**输入：**
```json
{
  "file_path": "internal/calc/calc.go",
  "framework": "go test"
}
```

**处理逻辑：**
1. 按文件扩展名分发（`.go` → Go AST，`.js`/`.ts`/`.jsx`/`.tsx` → JS 正则解析，`.py` → Python 正则解析）
2. 提取 exported functions / methods / class methods
3. 对每个函数生成表驱动测试模板
4. 写入测试文件（`_test.go` / `.test.js` / `test_*.py`）

**输出：**
```json
{
  "status": "ok",
  "test_file": "internal/calc/calc_test.go",
  "generated_cases": 12,
  "preview": "..."
}
```

**Go 生成器：** 基于 `go/ast` 原生 AST 分析，支持泛型类型参数实例化（`T → int`）、指针/值接收者方法、变参 `...T` → 切片、通道参数 nil-check + `t.Skip` 防阻塞、接口参数自动 mock、slice/map/struct 自动使用 `reflect.DeepEqual`。

**JS/TS 生成器：** 正则解析函数声明、箭头函数、类方法，自动检测 CommonJS / ES Module 导入方式。支持 async 函数、变参 `...args`、默认值参数、TypeScript 类型注解剥离。

**Python 生成器：** 正则解析 `def`/`async def`/`class` 声明，自动剥离 `self`/`cls` 参数、类型注解、默认值。支持 `*args`/`**kwargs`、`@staticmethod`。

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
1. 根据 `framework` 选择执行器（go-test / jest / vitest / mocha / pytest）
2. 自动检测项目类型（存在 `go.mod` → go-test，`package.json` 含 jest → jest，等等）
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

支持 5 种框架的输出解析：`go test` / Jest / Vitest / Mocha / pytest。

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
├── main.go                          # 入口，启动 MCP server（stdio）
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
│   │   ├── generator.go             # 多语言分发入口（按扩展名路由）
│   │   ├── go_generator.go          # Go AST 测试生成器（泛型/通道/接口/变参）
│   │   ├── js_generator.go          # JS/TS Jest 测试生成器（函数/箭头/类/async）
│   │   └── py_generator.go          # Python pytest 测试生成器（def/class/async）
│   ├── parser/
│   │   ├── parser.go                # 统一解析入口（按框架名分发）
│   │   ├── go_parser.go             # go test 输出解析
│   │   ├── jest_parser.go           # Jest 输出解析
│   │   ├── pytest_parser.go         # pytest 输出解析
│   │   └── mocha_parser.go          # Mocha 输出解析
│   └── coverage/
│       ├── coverage.go              # 统一入口 + 改进建议生成
│       ├── go_coverage.go           # Go coverprofile 解析
│       ├── jest_coverage.go         # Jest/Istanbul coverage JSON 解析（Vitest 共用）
│       └── pytest_coverage.go       # coverage.py JSON 解析
├── cmd/
│   └── testgen/main.go              # 独立 CLI 工具，脱离 MCP 直接生成测试
└── demo/                            # 示例代码（calc, service, advanced）
```

---

## 4. 传输层

支持两种 MCP 传输方式：

| 模式 | 参数 | 用途 |
|------|------|------|
| stdio | `--stdio` | 生产使用，接入 Claude/Cursor |
| Streamable HTTP | `--http --port 8080` | 开发调试，可用 Insomnia/Postman 测试 |

> 当前仅实现 stdio。Streamable HTTP 待后续迭代（go-sdk 已有 `StreamableHTTPRequestHandler`）。

---

## 5. 技术选型

| 组件 | 选择 | 理由 |
|------|------|------|
| MCP SDK | `github.com/modelcontextprotocol/go-sdk` v1.6.1 | MCP 官方 Go SDK |
| Go 版本 | 1.25+ | 泛型支持，标准库足够 |
| 测试解析 | 正则 + 行解析 | go test / Jest / pytest 输出格式稳定，无需复杂 Parser |
| 测试生成 | `go/ast` + 正则 + 模板 | Go 用标准库 AST；JS/Python 用正则解析签名 |
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
