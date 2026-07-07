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
- [ ] 发布前重新运行完整 release checklist
- [ ] 更新 `main.go` MCP implementation version
- [ ] 更新 README、安装文档和 CHANGELOG 版本号
- [ ] Tag `v0.4.10` 已推送
- [ ] Release Artifacts run 通过
- [ ] `v0.4.10` Release 资产验证
- [ ] Homebrew tap 更新到 `0.4.10`
- [ ] `brew test sleticalboy/tap/testloop-mcp`

## 发布信息

- Tag: 待发布
- Release: 待发布
- Release Artifacts run: 待发布
- Homebrew tap commit: 待发布

## 发布前注意

- 这是 post-v0.4.9 的候选发布资料，不回写已经发布的 `docs/plan-release-notes-v0.4.9.md`。
- `run_tests.include_fix_suggestions` 默认为 `false`，因此保持旧调用兼容；发布前需要确认 README 和 agent workflow 已明确说明开启条件。
- CI 如果因 GitHub runner 资源排队，应继续完成本地验证和发布资料准备；只有失败结论才需要阻塞发布。
