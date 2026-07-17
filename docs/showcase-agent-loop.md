# Agent 闭环展示案例

这个案例用于公开演示 testloop-mcp 的核心价值：让 AI Coding Agent 稳定消费结构化测试反馈，并完成一轮“失败 -> 修复任务 -> 复跑 -> 覆盖率解析”的闭环。它不依赖外部项目，不会修改当前仓库文件，适合放在 README、接入验收和演示脚本里反复运行。

## 运行方式

在源码 checkout 根目录执行：

```bash
go run ./examples/mcp-client-demo
```

如果只想跑回归断言：

```bash
sh test/mcp_client_demo_test.sh
```

## 预期输出

输出应包含以下关键步骤：

```text
1. run_tests: status=fail failed=1 suggestions=1
2. repair_task: category=generic_failure target=calc_test.go command=go test ./...
3. rerun: status=pass passed=1 coverage=100.0
4. parse_coverage: total=100.0 tasks=0
agent_next_step=use structuredContent first; fall back to text JSON only for older clients
```

这些行分别对应 Agent 的四个决策点：

1. `run_tests` 返回失败结果，并在同一个响应中提供 `fix_suggestions[]`。
2. Agent 读取 `fix_suggestions[].repair_task`，用结构化字段决定目标文件、复跑命令和修复范围。
3. 修复后再次调用 `run_tests coverage=true`，用测试状态和覆盖率确认闭环收敛。
4. 调用 `parse_coverage` 读取覆盖率结果，确认没有新的补测任务。

## 验收边界

这个 demo 使用 in-memory MCP client/server，重点验证工具调用顺序、`structuredContent` 消费方式和 text JSON fallback 一致性。真实进程传输兼容性由 `test/e2e` 中的 stdio 和 Streamable HTTP smoke 覆盖；客户端配置生成和本机接入检查由 `scripts/verify-client-setup.sh` 覆盖。

因此，这个案例不是为了证明某个生成器能理解复杂业务，而是为了证明 Agent 可以稳定依赖 testloop-mcp 提供的测试反馈协议，把测试失败转成可执行修复任务。
