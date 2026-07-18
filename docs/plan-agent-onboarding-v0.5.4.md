# v0.5.4 Agent Onboarding Demo 规划

## 目标

v0.5.4 优先收敛公开可复现的 Agent onboarding demo。目标不是继续扩语言，也不是继续强调“自动生成测试”，而是让新用户和接入方用一个命令看到完整反馈闭环：

1. 安装和客户端配置可验收。
2. 真实 MCP stdio / Streamable HTTP 传输可验收。
3. 最小 Agent 测试反馈闭环可运行。
4. Markdown 报告可转发给人看。
5. summary JSON 和 `agent_next_step` 可给 Agent / CI 消费。

## 当前阶段

- [x] 新增 `scripts/showcase-agent-onboarding-report.sh`。
- [x] 默认输出 Markdown、summary JSON 和 decision 文本。
- [x] 复用 `scripts/generate-verification-report.sh`，避免重复维护验收 section。
- [x] 复用 `examples/verification-summary-decision-demo`，避免新增另一套决策规则。
- [x] 新增 `test/showcase_agent_onboarding_report_test.sh` 固定 artifact 路径和 `agent_next_step=ready` 输出。
- [x] README、quickstart、`docs/showcase.md` 和 `docs/showcase-onboarding.md` 已补入口。
- [x] 使用当前源码构建的真实二进制 `/tmp/testloop-mcp-onboarding-demo` 跑通完整 wrapper，summary JSON 为 `overall_status=passed`、`failed_count=0`，decision 输出 `agent_next_step=ready`。
- [x] 新增 `docs/verification-summary-failures.md`，展示五类验收失败如何映射到 `agent_next_step`。
- [x] 新增 `docs/fixtures/verification-summary/*.json`，固定安装、MCP 协议、Agent demo、公开 showcase 和用户项目 smoke 失败样例。
- [x] 新增 `test/verification_summary_failure_fixtures_test.sh`，逐个 fixture 运行 decision demo 并校验 action。

## 后续任务

- [x] 在 quickstart 中把“首次接入验收”升级成两条路径：只看终端输出用 `showcase-onboarding.sh`，需要制品用 `showcase-agent-onboarding-report.sh`。
- [x] 增加一个失败样例文档，展示 summary JSON 如何把安装、协议、Agent demo、公开 showcase 和用户项目 smoke 分流到不同 `agent_next_step`。
- [ ] 评估是否把 onboarding report wrapper 放进 GitHub Actions 示例，减少接入方手写 env 和 artifact 路径。
- [ ] v0.5.4 候选发布前跑一次真实安装二进制的 onboarding report，并把摘要记录到发布说明草案。
