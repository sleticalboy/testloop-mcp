# first-run artifact Agent 消费演示

这份文档把 `first-run-context.txt` 和 `verification-summary.json` 的消费方式固定成可运行 demo，避免 Agent 只按自然语言说明猜下一步。

## 运行

使用内置失败 fixture：

```bash
go run ./examples/first-run-agent-response-demo \
  docs/fixtures/first-run-artifacts/user-project-smoke-failed/first-run-context.txt \
  docs/fixtures/first-run-artifacts/user-project-smoke-failed/verification-summary.json
```

输出应固定为四段：

```text
结论：testloop-mcp 接入链路本身是通的，失败发生在用户项目 smoke。

证据：
- first_run_agent_next_step=inspect-user-project
- failed_section=用户项目 smoke
- exit_code=7
- first_run_report=/tmp/testloop-user-project-failed-report.md

下一步：
- 打开 verification-report.md 中“用户项目 smoke”这一节，先看项目测试/构建命令的 stdout / stderr。
- 在用户项目目录复跑同一条 smoke 命令，确认依赖、环境变量或测试本身是否失败。

暂不做：
- 不先修改 testloop-mcp 安装或 MCP transport。
- 不先生成/修改测试，除非项目 smoke 的失败日志明确指向测试缺失或断言失败。
```

## 用在真实 CI artifact

当 GitHub Actions first-run 失败后，先下载 artifact：

```bash
gh run download <run-id> --name testloop-first-run --dir /tmp/testloop-first-run
```

如果已经有完整 artifact 目录，可以直接使用目录入口：

```bash
sh scripts/render-first-run-agent-response.sh /tmp/testloop-first-run
```

脚本会自动读取 `first-run-context.txt`，并在存在 `verification-summary.json` 时一起传给 demo。

从当前 `run-first-run-ci.sh` 生成的 artifact 开始，目录里也会包含 `agent-response.txt`，内容就是这条目录入口的输出。这个文件适合直接作为 Agent 回复草稿；如果它不存在，说明 artifact 由旧版脚本生成，可以用上面的目录入口补渲染。

也可以手动把真实 artifact 喂给 demo：

```bash
go run ./examples/first-run-agent-response-demo \
  /tmp/testloop-first-run/first-run-context.txt \
  /tmp/testloop-first-run/verification-summary.json
```

如果暂时没有 `verification-summary.json`，也可以只传 `first-run-context.txt`。此时 demo 仍会按 `first_run_agent_next_step` 输出结论和下一步，但不会输出 `failed_section` 和 `exit_code`。

## 端到端回归

仓库测试 `test/first_run_agent_response_demo_test.sh` 会先运行 `scripts/run-first-run-ci.sh` 构造一个失败的用户项目 smoke，再把输出目录里的两份真实 artifact 传给 demo：

```bash
go run ./examples/first-run-agent-response-demo \
  "$artifact_dir/first-run-context.txt" \
  "$artifact_dir/verification-summary.json"
```

这个测试固定了从 CI 失败 artifact 到 Agent 四段回复的整条链路。

目录入口由 `test/render_first_run_agent_response_test.sh` 固定回归：它覆盖完整 artifact 目录、只有 `first-run-context.txt` 的目录，以及缺少上下文文件时的错误提示。

## fixture 包

仓库提供一份完整的失败 artifact 包：

```text
docs/fixtures/first-run-artifacts/user-project-smoke-failed/
```

它包含 first-run 失败时应上传的七件套，适合客户端或 Agent 集成测试直接读取，不需要每次都重新构造失败项目。

## 边界

这个 demo 不调用 LLM，也不会修改用户项目。它只做一件事：把 first-run artifact 转成 Agent 应该先回复给用户的稳定四段结构。

相关文档：

- [first-run Agent 回复格式](./first-run-agent-response.md)
- [CI 失败后交给 Agent](./ci-agent-triage.md)
- [首跑诊断失败样例](./first-run-failures.md)
