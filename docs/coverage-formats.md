# 覆盖率格式说明

`parse_coverage` 负责解析已有覆盖率报告，不强制负责生成报告文件。不同语言推荐先用生态工具生成稳定格式，再把文件路径或文件内容传给 MCP。

## Go

推荐命令：

```bash
go test ./... -coverprofile=coverage.out
```

调用参数：

```json
{
  "framework": "go-test",
  "data": "coverage.out"
}
```

## JavaScript / TypeScript

Jest、Vitest、Mocha/nyc 都可以生成 Istanbul `coverage-final.json`。

调用参数：

```json
{
  "framework": "jest",
  "data": "coverage/coverage-final.json"
}
```

`framework` 可替换为 `vitest` 或 `mocha`。

## Python

推荐生成 coverage.py JSON：

```bash
coverage run -m pytest
coverage json -o coverage.json
```

调用参数：

```json
{
  "framework": "pytest",
  "data": "coverage.json"
}
```

## Rust

推荐使用 cargo tarpaulin 生成 LCOV：

```bash
cargo tarpaulin --out Lcov --output-dir target/tarpaulin
```

调用参数：

```json
{
  "framework": "cargo-test",
  "data": "target/tarpaulin/lcov.info"
}
```

## Java

推荐使用 JaCoCo XML。Maven 常见路径：

```bash
mvn test jacoco:report
```

调用参数：

```json
{
  "framework": "junit",
  "data": "target/site/jacoco/jacoco.xml"
}
```

Gradle 项目通常使用 `jacocoTestReport` 任务，报告路径一般在 `build/reports/jacoco/test/jacocoTestReport.xml`。
