# testloop-mcp

[![Go Report Card](https://goreportcard.com/badge/github.com/binlee/testloop-mcp)](https://goreportcard.com/report/github.com/binlee/testloop-mcp)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**testloop-mcp** 是一个基于 [MCP (Model Context Protocol)](https://modelcontextprotocol.io) 的 AI 测试辅助服务器。它让 AI Coding 工具（Claude Code / Cursor / VS Code Copilot）能够：

- 🧪 **智能生成测试**：根据代码变更自动生成单测 / 集成测试
- ▶️ **执行测试**：运行测试套件并实时获取结果
- 🔍 **解析失败**：结构化解析测试失败输出，提取失败原因和上下文
- 🔄 **反馈闭环**：将失败上下文喂回 AI，驱动自动修复

---

## 架构概览

```
AI IDE (Claude/Cursor)
        │  MCP JSON-RPC (stdio/SSE)
        ▼
  testloop-mcp server
        │
        ├── tools.generate_tests   → 分析代码，生成测试文件
        ├── tools.run_tests        → 执行测试框架命令
        ├── tools.parse_results    → 解析测试输出，提取失败详情
        └── tools.fix_suggestions → 根据失败输出生成修复建议
        │
        ▼
  本地项目（Go/Node/Python）
```

---

## 支持的语言 / 框架

| 语言 | 测试框架 | 状态 |
|------|---------|------|
| Go | `go test` / `testify` | ✅ 优先支持 |
| Node.js | Jest / Vitest | 🔜 后续支持 |
| Python | pytest | 🔜 后续支持 |

---

## 安装

```bash
# 从源码构建
git clone https://github.com/binlee/testloop-mcp.git
cd testloop-mcp
go build -o testloop-mcp .

# 或直接下载 release 二进制
# TODO: 添加 release 下载链接
```

---

## 配置（接入 AI IDE）

### Claude Code / Claude Desktop

在 `~/.claude/claude_desktop_config.json` 中添加：

```json
{
  "mcpServers": {
    "testloop": {
      "command": "/path/to/testloop-mcp",
      "args": ["--stdio"]
    }
  }
}
```

### Cursor

在 `.cursor/mcp.json` 中添加：

```json
{
  "mcpServers": {
    "testloop": {
      "command": "path/to/testloop-mcp",
      "args": ["--stdio"]
    }
  }
}
```

---

## MCP Tools 说明

### `generate_tests`
根据指定源文件生成测试代码。

**参数：**
- `file_path`（string，必填）：源文件路径
- `framework`（string，可选）：测试框架，默认自动检测

**返回：** 生成的测试代码内容

---

### `run_tests`
执行测试并返回结果。

**参数：**
- `path`（string，必填）：测试文件或目录路径
- `framework`（string，可选）：测试框架
- `coverage`（bool，可选）：是否收集覆盖率

**返回：** 结构化的测试结果 JSON

---

### `parse_results`
解析测试执行输出，提取失败用例详情。

**参数：**
- `output`（string，必填）：测试执行的标准输出/错误输出
- `framework`（string，可选）：测试框架

**返回：** 失败用例列表，含文件名、行号、错误信息

---

### `fix_suggestions`
根据测试失败信息，生成修复建议（供 AI 消费）。

**参数：**
- `failures`（string，必填）：`parse_results` 返回的失败 JSON
- `source_code`（string，必填）：原始源代码

**返回：** 结构化修复建议

---

## 开发

```bash
# 安装依赖
go mod tidy

# 本地运行（stdio 模式）
go run main.go --stdio

# 本地运行（SSE HTTP 模式，调试用）
go run main.go --sse --port 8080

# 运行测试
go test ./...

# 构建
go build -o testloop-mcp .
```

---

## Roadmap

- [x] MCP 服务器骨架（stdio + SSE 传输层）
- [x] Go `go test` 执行器
- [ ] 测试生成引擎（AST 分析 → 测试用例模板）
- [ ] 测试结果解析器（go test / Jest / pytest）
- [ ] 覆盖率收集与报告
- [ ] `fix_suggestions` 工具（失败 → 修复建议）
- [ ] Node.js / Jest 支持
- [ ] Python / pytest 支持
- [ ] VS Code Extension 配套

---

## License

MIT
