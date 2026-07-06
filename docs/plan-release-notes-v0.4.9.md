# v0.4.9 发布说明草案

## 标题

testloop-mcp v0.4.9

## 摘要

v0.4.9 是 v0.4.8 之后的 Agent 修复闭环与配置诊断细化版本。这个版本不改变 MCP 工具列表，重点是让配置检查失败和测试失败都能返回更结构化、更容易被 Agent 消费的修复线索。

## 主要变化

- `--doctor-config` 在发现 PATH、配置文件、`testloop` server 或 server 参数异常时，会输出下一步修复建议。
- `--check-config` 在配置校验失败时，会输出对应的配置生成、路径修复或 URL 修复建议。
- `fix_suggestions` 的建议文本补充 actual/want、越界 index/length、panic 类型和邻近源码上下文。
- `fix_suggestions` 会根据失败文件选择源码或测试文件上下文，并支持用相对路径匹配绝对 `test_code` 路径。
- `fix_suggestions` 返回新增 `category`、`context_file` 和 `context_line` 字段，便于 Agent 做失败分类和精准跳转。
- `docs/agent-workflow.md` 补充失败修复步骤，明确 `run_tests` / `parse_results` 失败后应先调用 `fix_suggestions`，再处理覆盖率缺口。

## 验证

- [x] `go test ./...`
- [x] `git diff --check`
- [ ] 远端 CI passed
- [ ] Tag `v0.4.9` 已推送
- [ ] Release Artifacts run 通过
- [ ] `v0.4.9` Release 资产验证
- [ ] Homebrew tap 更新到 `0.4.9`
- [ ] `brew test sleticalboy/tap/testloop-mcp`

## 发布信息

- Tag: 待发布
- Release: 待发布
- Release Artifacts run: 待发布
- Homebrew tap commit: 待发布

## 发布前注意

- 这是 post-v0.4.8 的候选发布资料，不回写已经发布的 `docs/plan-release-notes-v0.4.8.md`。
- CI 如果因 GitHub runner 资源排队，应继续完成本地验证和发布资料准备；只有失败结论才需要阻塞发布。
