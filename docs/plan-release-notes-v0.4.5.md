# v0.4.5 发布说明

## 标题

testloop-mcp v0.4.5

## 摘要

v0.4.5 是测试生成质量和发布前验证版本。这个版本不改变 MCP 工具协议，重点补强内置静态测试生成器的回归测试，覆盖 Go、Python、Jest、Java 和 Rust 的 coverage-task、parser、参数推断和 helper 分支，并把 MCP server implementation version 更新为 `0.4.5`。

## 主要变化

- MCP server implementation version 更新为 `0.4.5`。
- Go 静态生成器补充综合源码生成路径测试，覆盖结构体 fixture、接口 mock、泛型函数、变参函数、方法接收者和无可生成目标提示。
- Go seed helper 补充分支测试，覆盖单 return 表达式、表达式安全判断、不可精确 seed 的失败路径和未知标识符保护。
- Python/Jest coverage-task 生成器补充分支测试，覆盖 async、error_path、static/instance class 方法、边界输入、任务建议输入和 placeholder 参数。
- Java/Rust coverage-task helper 补充分支测试，覆盖目标过滤、测试名清洗、Rust 类型推断和 Java parser helper 行为。
- JS/Python parser 补充分支测试，覆盖 TypeScript 参数形态、解构参数、helper 过滤、decorated Python definition、staticmethod 和 receiver 参数剥离。
- Java parser 补充分支测试，覆盖 interface、enum、内部类和 helper 方法过滤。
- `internal/generator` 本地语句覆盖率提升到 `91.7%`。

## 验证

- [x] `go test ./...`
- [x] `git diff --check`
- [x] `go test -coverprofile=/tmp/generator-v045-prep.out ./internal/generator`，`internal/generator` coverage `91.7%`
- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `sh -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] `ruby -e 'require "yaml"; Dir[".github/workflows/*.yml"].each { |f| YAML.load_file(f) }'`
- [x] `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/*.yml`
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.4.5-local-package scripts/package-release-asset.sh v0.4.5 darwin_arm64 darwin arm64` 已验证本地打包、checksum 和 tarball 内容
- [ ] 远端 CI passed
- [x] Tag `v0.4.5` 已推送并指向 `9a903470aea214f34a356ab63e6aefa0eaade833`
- [ ] Release Artifacts run `28777039765` 通过（当前排队）
- [ ] `v0.4.5` Release 已包含 Linux amd64、Linux arm64、macOS arm64 和 Windows amd64 四类资产及各自 `.sha256`
- [ ] `TESTLOOP_MCP_VERSION=v0.4.5 sh scripts/install.sh` 已验证可直接下载 release 资产并安装
- [ ] Windows amd64 zip 已下载并通过 `.sha256` 校验，内容包含 `testloop-mcp.exe`、`testloop-testgen.exe`、`README.md` 和 `LICENSE`
- [ ] `sleticalboy/homebrew-tap` 已更新 `testloop-mcp` formula 到 `0.4.5`
- [ ] `brew fetch --force --formula sleticalboy/tap/testloop-mcp`
- [ ] `brew audit --strict --new sleticalboy/tap/testloop-mcp`
- [ ] `brew upgrade --formula sleticalboy/tap/testloop-mcp`
- [ ] `brew test sleticalboy/tap/testloop-mcp`

## 发布信息

- Tag: `v0.4.5` -> `9a903470aea214f34a356ab63e6aefa0eaade833`
- Release Artifacts run: `28777039765`（当前排队）
- Release: https://github.com/sleticalboy/testloop-mcp/releases/tag/v0.4.5（待创建）
