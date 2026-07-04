# Changelog

## v0.2.0 - 2026-07-05

### Added

- `parse_coverage` 支持 Rust `cargo tarpaulin --out Lcov` 生成的 LCOV。
- `parse_coverage` 支持 Java JaCoCo XML。
- Rust/Java 覆盖率报告会生成统一的 `CoverageReport`、`suggestions` 和 `test_tasks`。
- `run_tests coverage=true` 支持为 Rust 调用 tarpaulin、为 Java Maven/Gradle 调用 JaCoCo report，并回填 `coverage_percent`。
- Rust/Java 覆盖率闭环新增 e2e 测试，覆盖 `run_tests` 与 `parse_coverage` 联动。

## v0.1.0 - 2026-07-04

首个可用版本，定位为面向 AI Coding Agent 的测试反馈与质量控制 MCP 层。

### Added

- MCP server 支持 stdio 和 Streamable HTTP 两种传输模式。
- `run_tests` 支持 Go、Rust、Jest、Vitest、Mocha、pytest、JUnit 5 的测试执行与自动检测。
- `parse_results` 支持 Go、Rust、Jest、Vitest、Mocha、pytest、JUnit 5 的结构化失败解析。
- `generate_tests` 支持 Go、Rust、Java、JavaScript/TypeScript、Python 测试生成。
- Go 测试生成优先调用 `gotests -all`，失败时回退内置 `go/ast` 生成器。
- JS/TS/Python 生成器支持参数名语义默认值、边界输入、异常路径和基础返回类型断言。
- 可选 LLM provider：`provider: "llm"` / `provider: "auto"`，通过 `TESTLOOP_LLM_PROVIDER_CMD` 接入外部命令。
- `parse_coverage` 支持 Go coverprofile、Istanbul coverage JSON、coverage.py JSON。
- Go 覆盖率缺口可映射到函数/方法，并生成面向 AI Agent 的 `test_tasks`。
- `fix_suggestions` 返回结构化修复建议。
- 独立 CLI：`cmd/testgen`，支持 `-provider static|llm|auto`。
- Docker 镜像和 `docker-compose.yml`，HTTP 模式提供 `/healthz` 健康检查。
- GitHub Actions CI：测试、主服务构建、CLI 构建、Docker build。

### Fixed

- 修正低价值零值测试生成策略：无法推断有效输入时标记 TODO/skip。
- 修正 JS/Python 生成器中异常边界输入仍按正常返回值断言的问题。
- 修正 Docker healthcheck 访问 `/mcp` 无 session 返回 400 的问题。
- 修正 Alpine 运行时镜像安装不存在的 `musl-libc` 包的问题。
- 修正 `.gitignore` 误伤 `cmd/testgen/main.go` 的问题。

### Known Limitations

- Rust `cargo tarpaulin` 覆盖率解析尚未实现。
- Java JaCoCo 覆盖率解析尚未实现。
- LLM provider 当前是命令协议适配层，不内置具体模型厂商。
- 静态测试生成仍以可运行骨架和上下文增强为主，不承诺替代通用 AI Agent 的完整语义测试生成。

### Verification

- `go test ./...`
- `go build -o /tmp/testloop-mcp .`
- `go build -o /tmp/testloop-testgen ./cmd/testgen`
- `docker build -t testloop-mcp:release-check .`
- Docker container `/healthz` smoke test
- GitHub Actions CI passed
