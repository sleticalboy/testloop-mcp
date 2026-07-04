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
- [x] README 与 DESIGN 已说明 `generate_tests` 的 provider、gotests 优先路径和当前覆盖率支持范围。
- [x] README 已明确 Rust `cargo tarpaulin` 与 Java JaCoCo 覆盖率尚未实现。

## 本轮收口项

- [x] 修正 `cmd/testgen` 只适配 Go 输出文件名的问题，改为复用统一测试文件命名规则。
- [x] 给 `cmd/testgen` 增加 `-provider static|llm|auto` 参数，默认 `static`。
- [x] 修正 `.gitignore` 中 `testgen` 规则误伤 `cmd/testgen/main.go` 的问题，改为只忽略根目录 `/testgen` 二进制。
- [x] 给统一测试文件命名规则增加单元测试。
- [x] 修正 JS/Python 生成器中“会抛异常的边界输入仍按正常返回值断言”的问题，改为生成 `toThrow` / `pytest.raises`。
- [x] 修正 Docker healthcheck 指向 `/mcp` 导致无 session GET 返回 400 的问题，新增 `/healthz` 探活端点。
- [x] 修正 Dockerfile 运行时镜像安装不存在的 `musl-libc` 包的问题，运行时仅安装 `ca-certificates`。
- [x] 优化 `.dockerignore`，排除根目录构建产物，Docker build context 从约 29MB 降到 KB 级。

## 发布前仍建议补充

- [ ] 确认 GitHub Actions 或本地发布脚本是否需要补充。
- [ ] 如果准备打 tag，补充 changelog/release notes。
- [ ] Rust `cargo tarpaulin` 覆盖率解析仍未实现。
- [ ] Java JaCoCo 覆盖率解析仍未实现。
