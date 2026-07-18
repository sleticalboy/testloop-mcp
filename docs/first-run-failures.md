# 首跑诊断失败样例

这组样例面向用户和 AI Agent。目标是让首跑诊断失败时，用户可以直接粘贴一段稳定上下文，而不是把完整终端日志全部交给 Agent 猜。

运行首跑诊断后，优先看：

```text
first_run_agent_next_step=inspect-user-project
first_run_context=/tmp/testloop-mcp-first-run/first-run-context.txt
```

然后把 `first_run_context` 指向的文件内容粘给 Agent。这个文件会包含 summary JSON、Markdown report、decision 和完整日志路径。

## 决策映射

| fixture | first_run_agent_next_step | 下一步 |
| --- | --- | --- |
| [`fix-installation.txt`](./fixtures/first-run/fix-installation.txt) | `fix-installation` | 先检查二进制路径、版本门禁、生成配置、配置 roundtrip 和 HTTP `/healthz`。 |
| [`inspect-mcp-transport.txt`](./fixtures/first-run/inspect-mcp-transport.txt) | `inspect-mcp-transport` | 先排查 stdio / Streamable HTTP MCP 进程启动、端口占用和协议返回。 |
| [`inspect-agent-demo.txt`](./fixtures/first-run/inspect-agent-demo.txt) | `inspect-agent-demo` | 先看最小 Agent demo 的结构化返回、`repair_task` 和复跑流程。 |
| [`inspect-showcase.txt`](./fixtures/first-run/inspect-showcase.txt) | `inspect-showcase` | 先区分外部网络、公开仓库 checkout、依赖安装和 action 期望漂移。 |
| [`inspect-user-project.txt`](./fixtures/first-run/inspect-user-project.txt) | `inspect-user-project` | 先检查用户项目命令、依赖、环境变量和测试输出。 |

## 使用边界

- 这些 fixture 是 `doctor-first-run.sh` 输出的最小可粘贴上下文，不是完整 Markdown 报告。
- 自动化侧仍应优先读取 `first_run_summary_json` 和 `first_run_decision`。
- 人工排查时先打开 `first_run_report`，定位失败 section，再看 `first_run_log`。
- 如果 `first_run_agent_next_step=ready`，不需要进入失败样例库，可以继续配置客户端或跑真实项目验证。

## 当前实跑记录

2026-07-18 使用本地构建二进制和故意失败的用户项目 smoke 复验失败路径：

```bash
go build -o /tmp/testloop-mcp-first-run-failure .
TESTLOOP_FIRST_RUN_EXPECT_VERSION=0.5.7 \
TESTLOOP_FIRST_RUN_OUTPUT_DIR=/tmp/testloop-mcp-first-run-failed-check \
TESTLOOP_FIRST_RUN_PROJECT_DIR=/tmp/testloop-first-run-failed-project \
TESTLOOP_FIRST_RUN_PROJECT_COMMAND='echo first run project failed; exit 7' \
  scripts/doctor-first-run.sh /tmp/testloop-mcp-first-run-failure
```

结果：

- exit code 为 `1`。
- `first_run_status=failed`
- `first_run_failed_count=1`
- `first_run_agent_next_step=inspect-user-project`
- `first_run_context=/tmp/testloop-mcp-first-run-failed-check/first-run-context.txt`
- Markdown report 中保留了用户项目输出 `first run project failed`。

配套文档见 [首跑诊断](./first-run-diagnostics.md)、[用户项目验收报告](./verification-report.md) 和 [验收 Summary 失败分流样例](./verification-summary-failures.md)。
