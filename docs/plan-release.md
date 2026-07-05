# 发布检查清单

## 当前目标

发布前先确认项目的构建、CLI、Docker/HTTP 参数、文档和测试状态一致，避免 README 展示的能力与实际二进制行为不一致。

## 已验证

- [x] 主服务二进制可构建：`go build -o /tmp/testloop-mcp .`
- [x] 独立 CLI 可构建：`go build -o /tmp/testloop-testgen ./cmd/testgen`
- [x] `Dockerfile` 默认启动 HTTP 模式：`--transport http --addr :8080`
- [x] `docker-compose.yml` 与 Dockerfile 默认 HTTP 参数一致。
- [x] `docker compose config` 可正常解析，并确认 healthcheck 指向 `/healthz`。
- [x] HTTP 模式可启动，并且 `/healthz` 健康检查返回 200。
- [x] Docker 镜像可构建：`docker build -t testloop-mcp:release-check .`
- [x] Docker 容器可启动，并且映射端口后的 `/healthz` 返回 200。
- [x] GitHub Actions CI 已通过：测试、主服务构建、CLI 构建、Docker build。
- [x] README 与 DESIGN 已说明 `generate_tests` 的 provider、gotests 优先路径和当前覆盖率支持范围。
- [x] README 已明确 Rust `cargo tarpaulin` 与 Java JaCoCo 覆盖率支持范围。
- [x] README、LLM provider 文档和 v0.4.0 发布说明已说明 `coverage_task -> generate_tests` 的增量测试生成闭环。

## 本轮收口项

- [x] 修正 `cmd/testgen` 只适配 Go 输出文件名的问题，改为复用统一测试文件命名规则。
- [x] 给 `cmd/testgen` 增加 `-provider static|llm|auto` 参数，默认 `static`。
- [x] 修正 `.gitignore` 中 `testgen` 规则误伤 `cmd/testgen/main.go` 的问题，改为只忽略根目录 `/testgen` 二进制。
- [x] 给统一测试文件命名规则增加单元测试。
- [x] 修正 JS/Python 生成器中“会抛异常的边界输入仍按正常返回值断言”的问题，改为生成 `toThrow` / `pytest.raises`。
- [x] 修正 Docker healthcheck 指向 `/mcp` 导致无 session GET 返回 400 的问题，新增 `/healthz` 探活端点。
- [x] 修正 Dockerfile 运行时镜像安装不存在的 `musl-libc` 包的问题，运行时仅安装 `ca-certificates`。
- [x] 优化 `.dockerignore`，排除根目录构建产物，Docker build context 从约 29MB 降到 KB 级。
- [x] 新增 `CHANGELOG.md` 和 `docs/plan-release-notes.md`，准备 v0.1.0 发布说明。

## 后续建议补充

- [x] Rust `cargo tarpaulin` LCOV 覆盖率解析已实现。
- [x] Java JaCoCo XML 覆盖率解析已实现。
- [x] `run_tests` 的 coverage 模式已集成 tarpaulin/JaCoCo 报告生成命令。
- [x] `generate_tests` 已支持接收单个 `coverage_task`，并在 Go/Python/Jest/Rust/Java 中使用 task-aware 静态生成草稿。

## v0.4.0 发布前复验

- [x] `go test ./...`
- [x] `go build -o /tmp/testloop-mcp .`
- [x] `go build -o /tmp/testloop-testgen ./cmd/testgen`
- [x] `docker compose config`
- [x] `/tmp/testloop-mcp --transport http --addr :18080` 后 `GET /healthz` 返回 `ok`
- [x] `docker build -t testloop-mcp:release-check .`
- [x] `docker run -p 18081:8080 testloop-mcp:release-check` 后 `GET /healthz` 返回 `ok`
- [x] 最新远端 push CI 已通过：`28738856671`
- [x] 手动触发 CI 已通过：`28738899059`

## 发布产物

- [x] 新增 `Release Artifacts` workflow，支持 tag push 和手动指定 tag 触发。
- [x] 当前先构建 Linux amd64 的 server/CLI 二进制压缩包，并生成 `checksums.txt`。
- [x] workflow 会把产物上传到对应 GitHub Release；如果 Release 尚不存在，会先按 tag 创建。
- [ ] macOS、Windows 或更多架构需要结合 CGO 交叉编译工具链再扩展，暂不纳入第一版。
