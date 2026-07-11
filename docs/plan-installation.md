# 安装与分发体验规划

## 目标

让新用户可以从 Homebrew、GitHub Release、安装脚本、源码构建或 Docker 安装 testloop-mcp，并能快速接入 Codex、Claude Code / Claude Desktop 和 Cursor。

## 当前状态

- [x] Go module path 和文档仓库地址已统一为 `github.com/sleticalboy/testloop-mcp`。
- [x] `LICENSE` 已补齐，README License badge 指向有效文件。
- [x] `docs/installation.md` 覆盖 Homebrew、Release 下载、checksum 校验、源码构建、Docker、stdio、Streamable HTTP 和常见客户端配置。
- [x] `scripts/install.sh` 支持检测平台、下载匹配 release 资产、校验 `checksums.txt` 或单资产 `.sha256`、安装 `testloop-mcp` / `testloop-testgen`，资产缺失、平台不支持或下载失败时回退到 `go install`，并对 release 下载设置重试和超时。
- [x] Release Artifacts workflow 覆盖 Linux amd64、Linux arm64、macOS arm64、Windows amd64 和 Windows arm64。
- [x] Release Artifacts workflow 上传前会校验 `.sha256`，检查 tarball/zip 内包含两个二进制、`README.md` 和 `LICENSE`，并在 Windows runner 上实际运行 zip 内两个 `.exe --help`。
- [x] Post-Release Verify workflow 可手动输入 tag，校验 release 资产清单，并对 Linux amd64、Linux arm64、macOS arm64、Windows amd64 和 Windows arm64 执行安装脚本 dry run。
- [x] Homebrew tap 已接入 `sleticalboy/tap`，公式由 `Formula/testloop-mcp.rb` 和 `scripts/generate-homebrew-formula.sh` 维护。
- [x] `test/install_script_test.sh` 离线覆盖安装脚本的 Windows zip、单资产 `.sha256` fallback、下载重试/超时参数、下载失败提示和 `go install` fallback。
- [x] `scripts/verify-release-assets.sh` 可校验指定 tag 的五平台 release 资产和对应 `.sha256` 是否齐全，并有离线回归测试。

## 当前发布

- 当前版本：`v0.4.14`
- Tag：`v0.4.14` -> `b58b99aa2e69fefa0ce2a944bd9c86d360dbfe79`
- Release：https://github.com/sleticalboy/testloop-mcp/releases/tag/v0.4.14
- CI run：`29157660797`
- Release Artifacts run：`29157722825`
- Post-Release Verify run：`29157901152`
- Homebrew tap commit：`6394533b9f999bd2125efab6ace6f3c1e81da180`

`v0.4.14` Release 已包含：

- `testloop-mcp_v0.4.14_linux_amd64.tar.gz`
- `testloop-mcp_v0.4.14_linux_amd64.tar.gz.sha256`
- `testloop-mcp_v0.4.14_linux_arm64.tar.gz`
- `testloop-mcp_v0.4.14_linux_arm64.tar.gz.sha256`
- `testloop-mcp_v0.4.14_darwin_arm64.tar.gz`
- `testloop-mcp_v0.4.14_darwin_arm64.tar.gz.sha256`
- `testloop-mcp_v0.4.14_windows_amd64.zip`
- `testloop-mcp_v0.4.14_windows_amd64.zip.sha256`
- `testloop-mcp_v0.4.14_windows_arm64.zip`
- `testloop-mcp_v0.4.14_windows_arm64.zip.sha256`

`v0.4.14` 已验证：

- [x] 远端 CI passed
- [x] Release Artifacts run `29157722825` passed
- [x] `scripts/verify-release-assets.sh v0.4.14` 验证 release 页面包含 10 个必需资产
- [x] 手动触发 `Post-Release Verify` workflow `29157901152`，五平台安装脚本 dry run 全部通过
- [x] `brew fetch sleticalboy/tap/testloop-mcp`
- [x] `brew audit --formula --strict sleticalboy/tap/testloop-mcp`
- [x] `brew upgrade sleticalboy/tap/testloop-mcp`
- [x] `brew test sleticalboy/tap/testloop-mcp`
- [x] GitHub Release 正文已更新为正式 v0.4.14 发布说明
- [x] `scripts/install.sh` 的 release 下载失败路径仍会清晰提示并 fallback 到 `go install`；本机跨平台 dry run 因 GitHub 下载链路超时未作为发布通过条件。

## 版本摘要

| Version | 重点 | 关键验证 |
| --- | --- | --- |
| `v0.4.1` | 修正 module path，补齐安装文档和 MIT license | Release run `28739889556`；`go install @latest` 验证通过 |
| `v0.4.2` | 增加 Linux arm64、macOS arm64 和安装脚本 | Release build run `28746080130`；macOS arm64 安装脚本验证通过 |
| `v0.4.3` | 移除 publish job 队列瓶颈，接入 Homebrew tap | Release run `28761435820`；Homebrew tap 升级到 `0.4.3` 并通过 `brew test` |
| `v0.4.4` | 正式覆盖 Windows amd64 zip，安装脚本支持 Windows zip，移除临时 probe | Release run `28764619084`；asset verification run `28765386761`；Homebrew tap 升级到 `0.4.4` 并通过 `brew test` |
| `v0.4.5` | 补强内置静态测试生成器和 parser/helper 回归测试 | Release run `28777039765`；Homebrew tap 升级到 `0.4.5` 并通过 `brew test` |
| `v0.4.6` | 将发布后验证通过的 Homebrew formula help 测试修复纳入正式 release source archive | Release run `28782811885`；Homebrew tap 升级到 `0.4.6` 并通过 `brew test` |
| `v0.4.7` | 正式发布 Windows arm64 预构建 zip，并增强 Windows zip 运行验证 | Release run `28784950785`；Homebrew tap 升级到 `0.4.7` 并通过 `brew test` |
| `v0.4.8` | 编辑器接入体验增强，补齐 MCP 客户端配置生成、校验、诊断和 Agent 闭环示例 | Release run `28793678783`；Homebrew tap 升级到 `0.4.8` 并通过 `brew test` |
| `v0.4.9` | Agent 修复闭环和配置诊断细化，补充 `fix_suggestions` 分类与源码上下文 | Release run `28833047972`；Homebrew tap 升级到 `0.4.9` 并通过 `brew test` |
| `v0.4.10` | 将 `repair_task` 和 `run_tests.include_fix_suggestions` 纳入正式发布，并补强安装脚本下载重试 | Release run `28845299697`；Homebrew tap 升级到 `0.4.10` 并通过 `brew test` |
| `v0.4.11` | JS/TS 静态生成质量增强，补强复杂 TypeScript DTO payload 和 handler 闭环检查 | Release run `28995989142`；Homebrew tap 升级到 `0.4.11` 并通过 `brew test` |
| `v0.4.12` | JS/TS 同文件简单泛型 DTO 展开，payload 回退原因贯通到工具输出和 LLM provider 输入 | Release run `29022581976`；Post-Release Verify run `29025114403`；Homebrew tap 升级到 `0.4.12` 并通过 `brew test` |
| `v0.4.13` | LLM provider 接入质量增强，补齐默认 prompt、输出清洗、结构化 `provider_error` 和 Agent static fallback 闭环 | Release run `29089692602`；Post-Release Verify run `29090486292`；Homebrew tap 升级到 `0.4.13` 并通过 `brew test` |
| `v0.4.14` | Go coverage task 闭环质量增强，补齐 `validate_coverage_task`、skipped task 分类和 laoxia top50 隔离验证 | Release run `29157722825`；Post-Release Verify run `29157901152`；Homebrew tap 升级到 `0.4.14` 并通过 `brew test` |

## 发布维护流程

1. 更新 `CHANGELOG.md`、`main.go` MCP implementation version、README 和 `docs/installation.md`。
2. 新增或更新对应 `docs/plan-release-notes-vX.Y.Z.md`。
3. 本地运行：

   ```bash
   sh -n scripts/install.sh
   sh -n scripts/package-release-asset.sh
   sh test/install_script_test.sh
   sh test/release_assets_test.sh
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

6. 等 Release Artifacts 完成，确认五平台资产和 `.sha256` 都存在。

   ```bash
   scripts/verify-release-assets.sh vX.Y.Z
   ```

   也可以手动触发 `Post-Release Verify` workflow，输入 `vX.Y.Z`，让 GitHub runner 验证资产清单和五平台安装脚本 dry run。
7. 验证安装脚本和 Windows zip：

   ```bash
   TESTLOOP_MCP_VERSION=vX.Y.Z sh scripts/install.sh
   TESTLOOP_MCP_OS=windows TESTLOOP_MCP_ARCH=amd64 TESTLOOP_MCP_VERSION=vX.Y.Z sh scripts/install.sh
   TESTLOOP_MCP_OS=windows TESTLOOP_MCP_ARCH=arm64 TESTLOOP_MCP_VERSION=vX.Y.Z sh scripts/install.sh
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

## 待跟进项

- Homebrew Tap workflow 自动开 PR：依赖仓库 secret `HOMEBREW_TAP_TOKEN`。没有配置时 workflow 会成功跳过 PR 步骤，不影响 Release Artifacts 上传资产，也不影响本地脚本同步 tap。

## Windows arm64 发布验证

Windows arm64 已在 `v0.4.7` 正式纳入 Release Artifacts matrix，并发布 `testloop-mcp_v0.4.7_windows_arm64.zip` 和对应 `.sha256`。

已验证路径：

- 项目依赖 `github.com/smacker/go-tree-sitter`，`go list` 可确认 `go-tree-sitter` 及 Java/Rust/JS/Python/TypeScript grammar 包都包含 CGO 文件。
- Windows amd64 已通过 `windows-latest` + MSYS2 UCRT64 + `mingw-w64-ucrt-x86_64-gcc` 验证。
- Windows arm64 已通过手动 `Windows ARM64 Probe` workflow `28784385589` 验证：`windows-11-arm` runner、MSYS2 `CLANGARM64`、`mingw-w64-clang-aarch64-clang`、`CC=clang`、`CXX=clang++` 可以打包 `windows_arm64` zip。
- Probe 已完成 `.sha256` 校验、zip 内容检查，并在 ARM64 runner 上运行 `testloop-mcp.exe --help` 和 `testloop-testgen.exe --help`。
- `v0.4.7` 的 `Release Artifacts` workflow `28784950785` 中 `windows_arm64` matrix 项通过，正式 release 页面已包含 Windows arm64 zip。

后续维护注意：

- GitHub-hosted `windows-11-arm` 仍处于 public preview 时，如果平台临时排队或不可用，不应阻塞 Linux/macOS/Windows amd64 资产发布；必要时可回退为 probe-only。
