# v0.4.10 发布说明草案

## 标题

testloop-mcp v0.4.10

## 摘要

v0.4.10 是 v0.4.9 之后的失败修复闭环增强版本。这个版本不新增 MCP 工具，重点是把 `fix_suggestions` 从“结构化修复建议”推进到“Agent 可执行 repair task”，并允许 `run_tests` 在失败时直接内联修复摘要，减少工具往返。

## 主要变化

- `fix_suggestions` 每条建议新增 `repair_task`，包含稳定 `id`、失败分类、目标文件和行号、上下文片段、可编辑文件、建议复跑命令和断言关注点。
- `fix_suggestions` 会利用 `TestFailure.Expected` / `Received` 和 JS 常见 AssertionError 文本识别 `expectation_mismatch`，避免 Jest/Vitest/Mocha 的真实断言失败被降级为 generic 建议。
- `run_tests` 新增 `include_fix_suggestions`、`source_code` 和 `test_code` 输入；开启后，失败结果会内联 `fix_suggestions[]` 和 `repair_task`。
- repair task 新增 golden test，固定面向 Agent 的 JSON 契约、字段顺序和降级行为。
- `docs/agent-workflow.md`、README、DESIGN 和质量评估同步说明 `repair_task` 和 `include_fix_suggestions` 的使用方式。

## 验证

- [x] `go test ./...`
- [x] `git diff --check`
- [x] 远端 CI passed：`28835826433`
- [x] 远端 CI passed：`28836691629`
- [x] 远端 CI passed：`28839097063`
- [x] 远端 CI passed：`28840750740`
- [x] 发布前重新运行完整 release checklist
- [x] `sh -n scripts/install.sh`
- [x] `sh -n scripts/package-release-asset.sh`
- [x] `bash -n scripts/update-homebrew-tap.sh`
- [x] `bash -n scripts/generate-client-config.sh`
- [x] `bash -n scripts/generate-homebrew-formula.sh`
- [x] GitHub Actions workflow YAML 解析通过
- [x] `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/*.yml`
- [x] 主服务和 `testloop-testgen` CLI 构建通过
- [x] `testloop-mcp --help` 和 `testloop-testgen --help` 验证通过
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.4.10-local-package scripts/package-release-asset.sh v0.4.10 darwin_arm64 darwin arm64`
- [x] `/tmp/testloop-v0.4.10-local-package/testloop-mcp_v0.4.10_darwin_arm64.tar.gz.sha256` 校验通过
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`
- [x] 更新 `main.go` MCP implementation version
- [x] 更新 README、安装文档和 CHANGELOG 版本号
- [x] Tag `v0.4.10` 已推送
- [x] Release Artifacts run 通过
- [x] `v0.4.10` Release 资产验证
- [x] 安装脚本验证 macOS arm64 和 Windows amd64 资产下载、`.sha256` 校验和安装
- [x] Homebrew tap 更新到 `0.4.10`
- [x] `brew fetch --force --formula sleticalboy/tap/testloop-mcp`
- [x] `brew audit --strict --new sleticalboy/tap/testloop-mcp`
- [x] `brew upgrade --formula sleticalboy/tap/testloop-mcp`
- [x] `brew test sleticalboy/tap/testloop-mcp`

## 发布信息

- Tag: `v0.4.10` -> `4816c291bdadf320f356218eac7f35b48ebec094`
- Release: https://github.com/sleticalboy/testloop-mcp/releases/tag/v0.4.10
- CI run: `28845217140`
- Release Artifacts run: `28845299697`
- Homebrew tap commit: `0003c0c071c247c610cf8ed8f677f8f714610b17`

## 发布前注意

- 这是 post-v0.4.9 的候选发布资料，不回写已经发布的 `docs/plan-release-notes-v0.4.9.md`。
- `run_tests.include_fix_suggestions` 默认为 `false`，因此保持旧调用兼容；发布前需要确认 README 和 agent workflow 已明确说明开启条件。
- 发布前 checklist 于 2026-07-07 重新跑通。首次 `shasum -c` 在仓库根目录执行失败，因为 `.sha256` 使用相对产物文件名；切换到 `/tmp/testloop-v0.4.10-local-package` 后校验通过。
- CI 如果因 GitHub runner 资源排队，应继续完成本地验证和发布资料准备；只有失败结论才需要阻塞发布。
- 发布后安装验证中发现 GitHub release asset 下载偶发长时间无响应；`scripts/install.sh` 已增加 curl/wget 重试和超时控制，避免安装流程无限挂起或过早 fallback。
- Windows arm64 安装分支在本轮发布中已通过一次 zip 下载、`.sha256` 校验和安装；后续复跑时受 GitHub 下载链路影响触发 fallback，未发现 release 资产缺失。
