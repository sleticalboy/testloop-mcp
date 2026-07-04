# v0.2.0 发布说明草案

## 标题

testloop-mcp v0.2.0

## 摘要

v0.2.0 聚焦补齐 Rust/Java 覆盖率闭环，让 `parse_coverage` 和 `run_tests coverage=true` 不再只覆盖 Go/Node/Python。这个版本把 Rust `cargo tarpaulin` LCOV 与 Java JaCoCo XML 转换为统一的 `CoverageReport`、`suggestions` 和 `test_tasks`，并补充端到端测试保证 MCP 工具链联动稳定。

## 主要变化

- `parse_coverage` 支持 Rust `cargo tarpaulin --out Lcov` 生成的 LCOV。
- `parse_coverage` 支持 Java JaCoCo XML。
- Rust/Java 覆盖率报告会生成统一的 `CoverageReport`、`suggestions` 和 `test_tasks`。
- `run_tests coverage=true` 支持为 Rust 调用 tarpaulin，并从 `target/tarpaulin/lcov.info` 回填 `coverage_percent`。
- `run_tests coverage=true` 支持为 Java Maven/Gradle 调用 JaCoCo report，并从 XML 报告回填 `coverage_percent`。
- Rust/Java 覆盖率闭环新增 e2e 测试，覆盖 `run_tests` 与 `parse_coverage` 联动。

## 使用示例

Rust 覆盖率解析：

```json
{
  "framework": "cargo-test",
  "data": "target/tarpaulin/lcov.info"
}
```

Java 覆盖率解析：

```json
{
  "framework": "junit",
  "data": "target/site/jacoco/jacoco.xml"
}
```

`run_tests` 自动回填覆盖率：

```json
{
  "path": ".",
  "framework": "cargo-test",
  "coverage": true
}
```

```json
{
  "path": ".",
  "framework": "junit",
  "coverage": true
}
```

## 已知限制

- Rust 覆盖率生成依赖目标项目环境中已安装 `cargo tarpaulin`。
- Java 覆盖率生成依赖目标项目 Maven/Gradle 配置可执行 JaCoCo report。
- 当前 Rust/Java 覆盖率建议仍以文件和行级缺口为主，尚未像 Go 一样基于源码结构推断分支语义。

## 发布前验证

- [x] `go test ./...`
- [x] GitHub Actions CI passed

## 建议发布命令

```bash
git tag v0.2.0
git push origin v0.2.0
gh release create v0.2.0 --title "testloop-mcp v0.2.0" --notes-file docs/plan-release-notes-v0.2.0.md
```
