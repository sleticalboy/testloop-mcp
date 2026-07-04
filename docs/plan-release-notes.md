# v0.1.0 发布说明草案

## 标题

testloop-mcp v0.1.0

## 摘要

首个可用版本，提供面向 AI Coding Agent 的测试反馈闭环 MCP 服务。核心目标不是替代 Claude Code、Cursor 或 Codex 写测试，而是把测试执行、失败解析、覆盖率缺口和测试生成上下文整理成稳定、可复用的工具层。

## 主要能力

- MCP 工具：
  - `generate_tests`
  - `run_tests`
  - `parse_results`
  - `fix_suggestions`
  - `parse_coverage`
- 传输模式：
  - stdio
  - Streamable HTTP
- 生成能力：
  - Go：优先 `gotests -all`，失败回退内置 AST 生成器
  - Rust：生成 `#[test]` 测试骨架
  - Java：生成 JUnit 5 测试骨架
  - JS/TS：生成 Jest 测试，支持 async、throw、边界条件
  - Python：生成 pytest 测试，支持 async、raise、边界条件
- 解析能力：
  - Go test JSON 与文本回退
  - cargo test
  - Jest/Vitest/Mocha
  - pytest
  - JUnit 5
- 覆盖率：
  - Go coverprofile
  - Istanbul coverage JSON
  - coverage.py JSON
- 可选 LLM provider：
  - `provider: "static" | "llm" | "auto"`
  - `TESTLOOP_LLM_PROVIDER_CMD`
  - stdin JSON / stdout code 协议
- 部署：
  - Dockerfile
  - docker-compose
  - HTTP `/healthz`
- CI：
  - `go test ./...`
  - 主服务构建
  - `cmd/testgen` 构建
  - Docker build

## 已知限制

- Rust/Java 覆盖率解析已在后续版本规划中补齐；v0.1.0 发布时尚未包含。
- LLM provider 当前只提供命令协议，不内置具体模型厂商。
- 静态生成器更适合提供可运行骨架和上下文，不保证直接生成完整高价值业务断言。

## 发布前验证

- [x] `go test ./...`
- [x] `go build -o /tmp/testloop-mcp .`
- [x] `go build -o /tmp/testloop-testgen ./cmd/testgen`
- [x] `docker build -t testloop-mcp:release-check .`
- [x] Docker container `/healthz` smoke test
- [x] GitHub Actions CI passed

## 建议发布命令

```bash
git tag v0.1.0
git push origin v0.1.0
gh release create v0.1.0 --title "testloop-mcp v0.1.0" --notes-file docs/plan-release-notes.md
```
