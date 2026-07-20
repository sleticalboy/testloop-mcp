# v0.5.14 发布说明草案

## 标题

testloop-mcp v0.5.14

## 发布状态

- [x] 创建 v0.5.14 发布说明草案。
- [x] 梳理 v0.5.13 之后围绕 CI artifact 自检、summary schema 自包含、双项目 summary、默认 CI 覆盖、仓库卫生和真实项目证据的改动边界。
- [x] 最近本地验证已通过：shell 语法检查、全部 `test/*_test.sh`、`go test ./...`、候选二进制 build、`--version`、`--help`、darwin arm64 release asset dry-run、tarball `.sha256` 校验和 `git diff --check`。
- [x] 最近远端 CI 已通过：`c36758b` run `29737179225` passed，覆盖 laoxia artifact 自检复验证据。
- [ ] 最新 main CI 尚待最终确认；本地门禁补齐时，候选边界文档提交 `ab81926` 的 CI run `29737938722` 仍在 GitHub Actions 队列中。
- [ ] 尚未更新 implementation version、CHANGELOG 正式版本段、tag、Release assets 或 Homebrew tap。

## 摘要

v0.5.14 候选重点不是扩语言，也不是声称测试生成质量大幅提升，而是继续强化“AI Agent 测试反馈闭环”的可验证基础设施。

这个版本把 CI artifact 从“可以人工下载查看”推进到“下载后可机器自检、可批量校验、可 JSON 消费”。同时把 summary schema 和双项目 summary 做到 artifact 自包含，并用 laoxia server/web 真实项目重新证明 bootstrap 输出可以被 Agent 稳定消费。

## 主要变化

### Artifact 自检闭环

- 新增 `scripts/verify-agent-artifact.sh` 和 `examples/agent-artifact-verify`。
- 支持校验 first-run 七件套和 onboarding 五件套的必备文件。
- 使用 artifact 同目录的 `verification-summary.schema.json` 离线校验 `verification-summary.json`。
- 校验 `agent-decision.txt`、`agent-response.txt`、失败 section、`exit_code`、`section_signal` 和 summary 语义一致。
- 支持 `manifest` 模式，一条命令批量校验 `agent-response-artifact-manifest.json` 登记的 first-run/onboarding fixture。
- 支持 `--json` 输出，方便外部客户端断言 `status`、`artifact_count`、`artifacts[].response_action` 和 `section_signals`。

### Bootstrap 自动自检

- `run-first-run-ci.sh` 和 `run-onboarding-ci.sh` 在 helper 支持时会自动运行 artifact verifier。
- GitHub step summary 会写入 `Artifact verification`。
- helper 固定到旧 tag 或 CI 缺少 Go 时，脚本会 warning 跳过 verifier，不破坏已发布复制模板。

### Schema 和 summary 自包含

- 标准 verification summary、first-run/onboarding artifact fixture 和真实 CI artifact 都会携带 `verification-summary.schema.json`。
- 双项目报告会携带 `dual-project-summary.schema.json`。
- `agent-response-artifact-manifest.json` 每个 artifact 都带有本地 `summary_schema` 指针。
- first-run `agent-response.txt` 已补齐 `first_run_status` 和 `first_run_failed_count`，与 artifact contract 和 `first-run-context.txt` 对齐。

### CI 和仓库卫生

- 默认 GitHub Actions CI 现在显式运行每个 `test/*_test.sh`。
- 新增 `test/ci_workflow_test.sh` 防止后续新增 shell 契约测试但忘记放进 CI。
- 新增 `test/repository_hygiene_test.sh` 防止提交被 `.gitignore` 忽略的跟踪文件、`__pycache__/` 或 `.pyc`。
- 新增 `scripts/verify-release-candidate.sh`，把本地 release readiness 的 shell 语法、Go 测试、shell 契约测试、候选二进制构建、help/version、打包 dry-run、sha256 和 tarball 内容检查收敛成一个维护者入口。
- 已移除仓库里曾被跟踪的 Python bytecode 缓存。
- `testloop-mcp --help` 和 `testgen --help` 会以退出码 0 返回，发布门禁、安装自检和脚本 wrapper 不再需要为帮助输出特殊处理非 0 状态。

### 真实项目证据

- 新增/更新 laoxia server/web、QuickSmoke Go/Java、APK Info Rust/Words Java 等真实项目双项目报告记录。
- laoxia server/web 最新 onboarding bootstrap 复验通过，两个输出目录均为 `overall_status=passed`、`failed_count=0`、`agent_next_step=ready`、`agent_artifact_status=passed`。
- 两个 laoxia 外部项目本地 git 状态为空，确认 bootstrap 和 verifier 不污染用户项目工作区。

## 质量边界

- 这个候选版本提升的是 artifact 消费确定性和客户端回归能力，不承诺生成算法显著提升。
- `section_signal=... action=manual_review` 仍是 section 级动作信号，不等于整体验收失败。
- verifier JSON 输出是面向客户端断言的稳定摘要，不替代 `verification-report.md` 中的详细 stdout / stderr。
- 双项目 combined summary 和单项目 verification summary 是两类不同 schema，不能混用。

## 推荐验证

- `sh test/agent_artifact_verify_test.sh`
- `sh test/agent_response_artifact_manifest_test.sh`
- `sh test/agent_response_manifest_demo_test.sh`
- `sh test/run_first_run_ci_test.sh`
- `sh test/run_onboarding_ci_test.sh`
- `sh test/real_integration_cases_doc_test.sh`
- `sh test/repository_hygiene_test.sh`
- `sh test/ci_workflow_test.sh`
- `sh test/release_doc_index_test.sh`
- `sh test/docs_links_test.sh`
- `for script in test/*_test.sh; do sh "$script"; done`
- `go test ./...`
- `go build -o /tmp/testloop-mcp-v0.5.14-candidate .`
- `go build -o /tmp/testloop-testgen-v0.5.14-candidate ./cmd/testgen`
- `/tmp/testloop-mcp-v0.5.14-candidate --version`
- `/tmp/testloop-mcp-v0.5.14-candidate --help`
- `/tmp/testloop-testgen-v0.5.14-candidate --help`
- `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.14-candidate-dist scripts/package-release-asset.sh v0.5.14 darwin_arm64 darwin arm64`
- `cd /tmp/testloop-v0.5.14-candidate-dist && shasum -a 256 -c testloop-mcp_v0.5.14_darwin_arm64.tar.gz.sha256`
- `tar -tzf /tmp/testloop-v0.5.14-candidate-dist/testloop-mcp_v0.5.14_darwin_arm64.tar.gz`
- `scripts/verify-release-candidate.sh v0.5.14`
- `git diff --check`

## 发布备注

- 对外文案应强调“CI artifact 可机器自检、可批量校验、可 JSON 消费”。
- 推荐演示路径：先跑 `scripts/run-onboarding-ci.sh` 生成五件套，再跑 `scripts/verify-agent-artifact.sh --json onboarding <artifact-dir>` 展示结构化消费。
- 正式发布前需要先等最新 main CI 通过，再执行版本号、CHANGELOG 正式段、tag、Release assets、Homebrew tap 和 post-release verify。
