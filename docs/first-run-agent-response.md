# first-run Agent 回复格式

这份文档面向 AI Coding Agent。用户把 `first-run-context.txt` 粘过来后，Agent 不应该直接开始改测试或猜测日志含义，而应该先按 `first_run_agent_next_step` 分流。

## 回复结构

推荐固定成四段：

```text
结论：<一句话说明失败属于哪一层>

证据：
- first_run_agent_next_step=<action>
- failed_section=<失败 section>
- exit_code=<退出码，如果有>

下一步：
- <第一条可执行动作>
- <第二条可执行动作>

暂不做：
- <当前不应该做的事>
```

如果上下文不完整，先要求用户补 artifact，而不是猜：

```text
我需要先看到 agent-decision.txt 和 first-run-context.txt。只有 GitHub Actions 最后一行错误不足以判断是安装、MCP transport、Agent demo 还是用户项目 smoke 失败。
```

## 分流动作

| `first_run_agent_next_step` | 结论 | Agent 下一步 |
| --- | --- | --- |
| `ready` | testloop 接入链路通过。 | 继续真实生成测试、补覆盖率、修复业务失败或接入 MCP 客户端。 |
| `fix-installation` | 安装或版本门禁失败。 | 检查二进制路径、`testloop-mcp --version`、Homebrew upgrade/reinstall、配置 roundtrip 和 HTTP `/healthz`。 |
| `inspect-mcp-transport` | MCP transport 失败。 | 检查 stdio / Streamable HTTP 启动、端口占用、客户端配置和真实 MCP smoke 输出。 |
| `inspect-agent-demo` | 最小 Agent demo 失败。 | 检查结构化返回、demo runner、Go 运行环境和仓库自身构建。 |
| `inspect-user-project` | 用户项目 smoke 失败。 | 检查项目测试/构建命令、依赖、环境变量和失败 section 的 stdout / stderr。 |
| `inspect-showcase` | 公开 showcase 失败。 | 区分 GitHub/npm 网络、外部项目 checkout、依赖安装和 action 期望漂移。 |

## inspect-user-project 示例

用户粘贴的上下文包含：

```text
first_run_status=failed
first_run_failed_count=1
first_run_agent_next_step=inspect-user-project
first_run_report=/tmp/testloop-first-run-failure-triage/verification-report.md
```

`verification-summary.json` 显示：

```text
failed_section=用户项目 smoke
exit_code=7
```

Agent 应回复：

```text
结论：testloop-mcp 接入链路本身是通的，失败发生在用户项目 smoke。

证据：
- first_run_agent_next_step=inspect-user-project
- failed_section=用户项目 smoke
- exit_code=7

下一步：
- 打开 verification-report.md 中“用户项目 smoke”这一节，先看项目测试/构建命令的 stdout / stderr。
- 在用户项目目录复跑同一条 smoke 命令，确认依赖、环境变量或测试本身是否失败。

暂不做：
- 不先修改 testloop-mcp 安装或 MCP transport。
- 不先生成/修改测试，除非项目 smoke 的失败日志明确指向测试缺失或断言失败。
```

## fix-installation 示例

```text
结论：失败发生在 testloop-mcp 安装或版本门禁，还没进入用户项目测试。

证据：
- first_run_agent_next_step=fix-installation

下一步：
- 先运行 testloop-mcp --version，确认是否等于文档要求的版本。
- 如果是 Homebrew 安装，执行 brew update && brew upgrade sleticalboy/tap/testloop-mcp；仍旧版本时执行 brew reinstall。
- 重新运行 first-run 诊断，直到基础安装验收通过。

暂不做：
- 不修改用户项目测试。
- 不排查覆盖率或生成质量。
```

## 缺少上下文时

如果用户只贴了 CI 最后一行错误，Agent 应回复：

```text
这段日志不足以判断失败层级。请下载 testloop-first-run artifact，并至少粘贴：
1. agent-decision.txt
2. first-run-context.txt

如果没有 first-run-context.txt，再补 verification-summary.json 和 verification-report.md 的失败 section。
```

相关文档：

- [CI 失败后交给 Agent](./ci-agent-triage.md)
- [首跑诊断失败样例](./first-run-failures.md)
- [Onboarding CI 失败排查](./onboarding-ci-failure-triage.md)
