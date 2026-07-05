# Changelog

## Unreleased

### Changed

- Release Artifacts workflow 改为由每个 matrix build job 直接上传对应 tarball 和 `.sha256`，避免单独 publish job 等不到 runner 时阻塞发版。
- 安装脚本兼容聚合 `checksums.txt` 和单资产 `.sha256` 两种校验文件。

## v0.4.2 - 2026-07-05

### Added

- Release Artifacts workflow 准备生成 Linux amd64、Linux arm64 和 macOS arm64 三类 tarball，并统一生成 `checksums.txt`。
- 新增 `scripts/install.sh`，支持检测平台、下载 release 资产、校验 checksum、安装 `testloop-mcp` / `testloop-testgen`，资产缺失时回退到 `go install`。

## v0.4.1 - 2026-07-05

### Added

- 新增 `docs/installation.md`，补齐 Release 下载、checksum 校验、源码构建、Docker 运行和 Codex / Claude / Cursor 接入说明。
- 新增 MIT `LICENSE` 文件。

### Changed

- Go module path 和文档仓库地址统一为 `github.com/sleticalboy/testloop-mcp`，为后续新版本支持 `go install github.com/sleticalboy/testloop-mcp@latest` 做准备。

## v0.4.0 - 2026-07-05

### Added

- Rust `cargo tarpaulin` LCOV 覆盖率建议会尝试把未覆盖行映射到具体 `fn`，并在 `test_tasks` 中使用函数目标。
- Java JaCoCo 覆盖率建议会尝试把未覆盖行映射到具体类方法，并支持常见 `src/main/java` 源码目录解析。
- Rust/Java 覆盖率建议会对 `if`、`match`、`switch`、错误/空值返回和普通返回做轻量语义分类，生成更具体的 `gap_type`、`missing_branches` 和输入提示。
- Java 覆盖率源码映射改用 tree-sitter，支持注解、多行方法签名、构造函数和内部类，并保留轻量正则回退。
- Rust 覆盖率源码映射改用 tree-sitter，支持属性标注函数、多行函数签名、`impl` 方法和 trait 默认方法，并保留轻量正则回退。
- 新增 Rust workspace 和 Java Maven 风格覆盖率 fixture，验证相对报告路径、复杂源码目录和源码映射不会退化。
- `test_tasks` 新增 `test_file`、`test_name` 和 `assertion_focus`，让 AI Agent 更容易把覆盖率缺口转成具体测试草稿。
- `test_tasks` 新增 `priority` 和 `priority_reason`，并按函数/方法级缺口、分支/错误路径、建议输入、未覆盖行和置信度排序。
- `generate_tests` 支持接收单个 `coverage_task`，并把任务上下文传给 LLM provider、回写到返回 context，同时优先写入任务推荐的 `test_file`。
- Go/Rust/Java coverage task 输出新增 JSON golden 快照测试，固定面向 Agent 的任务契约。
- Go 静态生成器支持 `coverage_task` 模式，会优先只生成目标函数或方法的测试，并把 task 信息写入测试名、case 名和注释。
- Python/Jest 静态生成器支持 `coverage_task` 模式，会按目标过滤测试草稿，并把建议输入转成更具体的调用参数和断言。
- Rust/Java 静态生成器支持 `coverage_task` 模式，会优先生成目标函数或方法的测试骨架，减少整文件泛化输出。
- 新增 Go/Python/Jest/Rust/Java task-aware 静态生成 golden tests，防止 coverage task 增量测试草稿退化。
- 补齐 v0.4.0 发布说明草案，并同步 README、LLM provider 文档和质量评估中的 coverage task 闭环说明。

## v0.3.0 - 2026-07-05

### Added

- Python/Jest 生成器会对简单 return 表达式生成精确断言，例如 `a + b` 会生成 `assert result == (1 + 2)` / `expect(result).toBe((1 + 2))`。
- 边界用例会把边界值带入简单 return 表达式，生成更具体的断言。
- Go 内置生成器会为简单纯函数生成可执行表驱动 case，不再默认只生成 TODO/skip。
- Python/Jest 生成器会识别简单 if-return 分支，为普通路径和边界路径分别生成期望值。
- Go/Python/Jest 生成器新增 golden tests，固定代表性输出。

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
