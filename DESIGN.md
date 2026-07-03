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

**目标**：给定源文件，生成对应的测试文件。

**输入：**
```json
{
  "file_path": "internal/calc/calc.go",
  "framework": "go test",
  "coverage_target": ["Add", "Sub"],
}
```

**处理逻辑：**
1. 读取源文件，解析 AST
2. 提取 exported functions / methods
3. 对每个函数生成表驱动测试模板
4. 写入 `{source}_test.go`

**输出：**
```json
{
  "status": "ok",
  "test_file": "internal/calc/calc_test.go",
  "generated_cases": 12,
  "preview": "..."
}
```

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
1. 根据 `framework` 选择执行器（Go/Node/Python）
2. 执行命令：`go test -v -cover ./internal/calc/...`
3. 捕获 stdout/stderr
4. 调用 `parser` 解析输出
5. 返回结构化 JSON

**输出：**
```json
{
  "status": "fail",
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

**输入：** 测试执行的原始 stdout/stderr

**输出：** 同 `run_tests` 的 failures 结构（可独立使用）

---

### Tool 4：`fix_suggestions`

**目标**：将失败信息 + 源代码喂给 AI，生成修复建议。

> 注意：此工具本身不调用 AI，而是将上下文**结构化打包**，让调用方（Claude/Cursor）直接使用。

**输入：**
```json
{
  "failures": [...],
  "source_code": "package calc\n func Add(...)",
  "test_code": "..."
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

---

## 3. 内部模块设计

```
testloop-mcp/
├── main.go                 # 入口，启动 MCP server
├── server/
│   └── server.go          # MCP 服务器初始化，注册 tools
├── tools/
│   ├── generate_tests.go  # Tool 1 实现
│   ├── run_tests.go       # Tool 2 实现
│   ├── parse_results.go   # Tool 3 实现
│   └── fix_suggestions.go # Tool 4 实现
├── internal/
│   ├── runner/
│   │   ├── runner.go     # 测试执行器接口
│   │   ├── go_runner.go  # Go test 执行器
│   │   ├── node_runner.go# Node.js 执行器（TODO）
│   │   └── python_runner.go # Python 执行器（TODO）
│   ├── parser/
│   │   ├── parser.go     # 解析器接口
│   │   ├── go_parser.go  # go test 输出解析
│   │   ├── jest_parser.go# Jest 输出解析（TODO）
│   │   └── pytest_parser.go # pytest 输出解析（TODO）
│   └── generator/
│       ├── generator.go   # 测试生成器接口
│       └── go_generator.go # Go 测试代码生成（AST 分析）
└── types/
    └── types.go           # 公共类型定义
```

---

## 4. 传输层

支持两种 MCP 传输方式：

| 模式 | 参数 | 用途 |
|------|------|------|
| stdio | `--stdio` | 生产使用，接入 Claude/Cursor |
| SSE HTTP | `--sse --port 8080` | 开发调试，可用 Insomnia/Postman 测试 |

---

## 5. 技术选型

| 组件 | 选择 | 理由 |
|------|------|------|
| MCP SDK | `github.com/mark3labs/mcp-go` | Go 生态最主流 MCP 实现 |
| Go 版本 | 1.22+ | 泛型支持，标准库足够 |
| 测试解析 | 正则 + 行解析 | go test 输出格式稳定，无需复杂 Parser |
| 测试生成 | `go/ast` + 模板 | 标准库即可，无额外依赖 |
| 配置 | 命令行参数 + 环境变量 | 保持简单，MCP 本身无状态 |

---

## 6. 核心类型定义（types/types.go）

```go
// TestResult 单次测试执行结果
type TestResult struct {
    Status          string        // "pass" / "fail" / "skip"
    Passed          int
    Failed          int
    Skipped         int
    CoveragePercent float64
    Failures        []TestFailure
    RawOutput       string
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
    File        string  `json:"file"`
    Line        int     `json:"line"`
    Issue       string  `json:"issue"`
    SuggestedFix string `json:"suggested_fix"`
    Confidence  float64 `json:"confidence"`
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
- ✅ 打通「生成 → 执行 → 解析 → 建议」闭环
