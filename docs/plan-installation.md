# 安装与分发体验规划

## 目标

让新用户可以从 Homebrew、GitHub Release、安装脚本、源码构建或 Docker 安装 testloop-mcp，并能快速接入 Codex、Claude Code / Claude Desktop 和 Cursor。

## 当前状态

- [x] Go module path 和文档仓库地址已统一为 `github.com/sleticalboy/testloop-mcp`。
- [x] `LICENSE` 已补齐，README License badge 指向有效文件。
- [x] `docs/installation.md` 覆盖 Homebrew、Release 下载、checksum 校验、源码构建、Docker、stdio、Streamable HTTP 和常见客户端配置。
- [x] `scripts/install.sh` 支持检测平台、下载匹配 release 资产、校验 `checksums.txt` 或单资产 `.sha256`、安装 `testloop-mcp` / `testloop-testgen`，资产缺失时回退到 `go install`。
- [x] Release Artifacts workflow 覆盖 Linux amd64、Linux arm64、macOS arm64 和 Windows amd64。
- [x] Release Artifacts workflow 上传前会校验 `.sha256`，并检查 tarball/zip 内包含两个二进制、`README.md` 和 `LICENSE`。
- [x] Homebrew tap 已接入 `sleticalboy/tap`，公式由 `Formula/testloop-mcp.rb` 和 `scripts/generate-homebrew-formula.sh` 维护。

## 当前发布

- 当前版本：`v0.4.4`
- Tag：`v0.4.4` -> `c91ae92a7e95eed2c7c674225699125143671066`
- Release：https://github.com/sleticalboy/testloop-mcp/releases/tag/v0.4.4
- Release Artifacts run：`28764619084`
- Release asset verification backfill run：`28765386761`
- Homebrew tap commit：`39e2ce3 Update testloop-mcp to v0.4.4`

`v0.4.4` Release 已包含：

- `testloop-mcp_v0.4.4_linux_amd64.tar.gz`
- `testloop-mcp_v0.4.4_linux_amd64.tar.gz.sha256`
- `testloop-mcp_v0.4.4_linux_arm64.tar.gz`
- `testloop-mcp_v0.4.4_linux_arm64.tar.gz.sha256`
- `testloop-mcp_v0.4.4_darwin_arm64.tar.gz`
- `testloop-mcp_v0.4.4_darwin_arm64.tar.gz.sha256`
- `testloop-mcp_v0.4.4_windows_amd64.zip`
- `testloop-mcp_v0.4.4_windows_amd64.zip.sha256`

`v0.4.4` 已验证：

- [x] 远端 CI run `28764570560` passed
- [x] Release Artifacts run `28764619084` passed
- [x] Release Artifacts run `28765386761` 验证四平台上传前资产校验均通过
- [x] `TESTLOOP_MCP_VERSION=v0.4.4 sh scripts/install.sh` 可直接下载 release 资产并安装
- [x] Windows amd64 zip 已下载并通过 `.sha256` 校验，内容包含 `testloop-mcp.exe`、`testloop-testgen.exe`、`README.md` 和 `LICENSE`
- [x] `brew fetch --force --formula sleticalboy/tap/testloop-mcp`
- [x] `brew audit --strict --new sleticalboy/tap/testloop-mcp`
- [x] `brew upgrade --formula sleticalboy/tap/testloop-mcp` 可从 `0.4.3` 升级到 `0.4.4`
- [x] `brew test sleticalboy/tap/testloop-mcp`

## 准备中版本

- 目标版本：`v0.4.5`
- Tag：`v0.4.5` -> `9a903470aea214f34a356ab63e6aefa0eaade833`
- Release Artifacts run：`28777039765`（当前排队）
- 发布说明草案：`docs/plan-release-notes-v0.4.5.md`
- 重点：补强内置静态测试生成器的回归测试，覆盖 Go、Python、Jest、Java 和 Rust 的 coverage-task、parser、参数推断和 helper 分支。
- 本地验证：`go test ./...`、`git diff --check`、release 脚本语法检查、workflow lint 和本机 `darwin_arm64` 打包模拟已通过。
- 当前远端 CI 多个 push run 仍在排队；按维护流程，排队状态不阻塞本地验证和后续 release prep。

## 版本摘要

| Version | 重点 | 关键验证 |
| --- | --- | --- |
| `v0.4.1` | 修正 module path，补齐安装文档和 MIT license | Release run `28739889556`；`go install @latest` 验证通过 |
| `v0.4.2` | 增加 Linux arm64、macOS arm64 和安装脚本 | Release build run `28746080130`；macOS arm64 安装脚本验证通过 |
| `v0.4.3` | 移除 publish job 队列瓶颈，接入 Homebrew tap | Release run `28761435820`；Homebrew tap 升级到 `0.4.3` 并通过 `brew test` |
| `v0.4.4` | 正式覆盖 Windows amd64 zip，安装脚本支持 Windows zip，移除临时 probe | Release run `28764619084`；asset verification run `28765386761`；Homebrew tap 升级到 `0.4.4` 并通过 `brew test` |
| `v0.4.5` | 准备中：补强内置静态测试生成器和 parser/helper 回归测试 | 本地 generator coverage `91.7%`；release checklist 本地验证通过，远端 CI 排队中 |

## 发布维护流程

1. 更新 `CHANGELOG.md`、`main.go` MCP implementation version、README 和 `docs/installation.md`。
2. 新增或更新对应 `docs/plan-release-notes-vX.Y.Z.md`。
3. 本地运行：

   ```bash
   sh -n scripts/install.sh
   sh -n scripts/package-release-asset.sh
   ruby -e 'require "yaml"; Dir[".github/workflows/*.yml"].each { |f| YAML.load_file(f) }'
   go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/*.yml
   go test ./...
   git diff --check
   ```

4. 推送 release prep commit，确认 CI；CI 排队时不要阻塞独立的本地验证和后续准备。
5. 创建并推送 tag：

   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

6. 等 Release Artifacts 完成，确认四平台资产和 `.sha256` 都存在。
7. 验证安装脚本和 Windows zip：

   ```bash
   TESTLOOP_MCP_VERSION=vX.Y.Z sh scripts/install.sh
   TESTLOOP_MCP_OS=windows TESTLOOP_MCP_ARCH=amd64 TESTLOOP_MCP_VERSION=vX.Y.Z sh scripts/install.sh
   ```

8. 更新 Homebrew formula：

   ```bash
   scripts/generate-homebrew-formula.sh vX.Y.Z
   scripts/update-homebrew-tap.sh vX.Y.Z ../homebrew-tap
   ```

9. 验证 Homebrew：

   ```bash
   brew fetch --force --formula sleticalboy/tap/testloop-mcp
   brew audit --strict --new sleticalboy/tap/testloop-mcp
   brew upgrade --formula sleticalboy/tap/testloop-mcp
   brew test sleticalboy/tap/testloop-mcp
   ```

10. 提交发布验证记录，保持 `raw.md` 不进提交。

## 暂缓项

- Windows arm64 预构建二进制：项目使用 CGO 和 tree-sitter，当前先发布 Windows amd64；Windows arm64 暂缓到工具链需求明确后再评估。
- Homebrew Tap workflow 自动开 PR：依赖仓库 secret `HOMEBREW_TAP_TOKEN`。没有配置时不影响 Release Artifacts 上传资产，也不影响本地脚本同步 tap。

## Windows arm64 评估

当前不把 Windows arm64 加入 Release Artifacts matrix。

原因：

- 项目依赖 `github.com/smacker/go-tree-sitter`，`go list` 可确认 `go-tree-sitter` 及 Java/Rust/JS/Python/TypeScript grammar 包都包含 CGO 文件。
- Windows amd64 已通过 `windows-latest` + MSYS2 UCRT64 + `mingw-w64-ucrt-x86_64-gcc` 验证。
- Windows arm64 不是只新增 `GOARCH=arm64` 就能可靠产出的目标；它需要可验证的 Windows ARM64 CGO 编译器、链接器和运行/解包校验链路。
- 当前 GitHub-hosted release workflow 没有现成的 Windows ARM64 runner 验证路径，盲目交叉编译会产生无法运行验证的资产。

重新评估条件：

- 有稳定的 Windows ARM64 runner，或能在 CI 中安装并验证 Windows ARM64 CGO toolchain。
- 能在 workflow 中完成 `go build`、`.zip` 打包、`.sha256` 校验、zip 内容检查，以及至少 `--help` 级别的二进制运行验证。
- 用户侧确实有 Windows ARM64 预构建二进制需求；否则继续使用源码构建或 `go install` 回退更稳。
