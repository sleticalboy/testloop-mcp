# Changelog

## Unreleased

### Added

- 新增主 MCP 工具 handler 层结构化返回契约测试，固定 `generate_tests`、`run_tests`、`parse_results`、`parse_coverage` 和 `fix_suggestions` 的 `structuredContent`、handler 返回值与 `content[0].text` JSON 语义一致。
- 新增 laoxia 双栈验收报告入口和通用双项目报告 helper，支持把 server/web 或任意两条项目 smoke 汇总成嵌套子 summary 的 combined summary。
- 新增 `docs/fixtures/dual-project-summary.schema.json` 和 `docs/fixtures/dual-project-summary/laoxia-passed.json`，固定双项目 combined summary 的结构契约，并在脚本回归中校验实际输出。
- 新增真实项目双项目 showcase 记录，覆盖 laoxia server/web、QuickSmoke Go/Java、APK Info Rust/Words Java 等跨项目或跨语言 pair。
- 新增 verification summary artifact 自包含 schema：标准验收报告、first-run/onboarding CI artifact 和静态 Agent artifact fixture 都会随 `verification-summary.json` 提供 `verification-summary.schema.json`。
- 新增 dual-project summary artifact 自包含 schema：通用双项目报告和 laoxia wrapper 会把 `dual-project-summary.schema.json` 写到 combined summary 同目录。
- 新增 `test/ci_workflow_test.sh`，要求默认 CI 显式运行每个 `test/*_test.sh`。
- 新增 `test/repository_hygiene_test.sh`，拒绝被 `.gitignore` 忽略但仍被 Git 跟踪的文件，并防止重新提交 `__pycache__/` 或 `.pyc`。
- 新增 `scripts/verify-agent-artifact.sh` 和 `examples/agent-artifact-verify`，可离线校验下载后的 first-run/onboarding artifact 目录，并支持 manifest 批量模式和 `--json` 结构化输出。

### Changed

- Agent response artifact manifest 的每个 artifact 现在带有本地 `summary_schema=verification-summary.schema.json` 指针，客户端下载单个 artifact 目录后可以离线校验 summary。
- first-run Agent 回复现在会输出 `first_run_status` 和 `first_run_failed_count`，与 artifact contract 和 `first-run-context.txt` 保持一致。
- `run-first-run-ci.sh` 和 `run-onboarding-ci.sh` 会在 helper 支持时自动运行 artifact verifier，并在 GitHub step summary 写入 `Artifact verification`。
- README、showcase、CI 集成、接入指南、fixture 索引和 artifact contract 已从 first-run 六件套 / onboarding 四件套更新为 first-run 七件套 / onboarding 五件套。
- 默认 GitHub Actions CI 已补跑 first-run/onboarding Agent response、artifact manifest、artifact fixture、外部 dry-run 文档、接入指南、README snippet 和 MCP 客户端契约等所有现有 shell 契约测试。
- `.gitignore` 为有意保留的 demo 输出和 first-run fixture log 增加精确例外，避免 fixture 被通用忽略规则误伤。
- 真实接入案例已补充 laoxia server/web 最新 onboarding bootstrap 记录，确认 artifact verifier 在真实项目上输出 `passed`。

### Fixed

- 多个对外脚本现在会提前拒绝目录型二进制路径、项目路径、输出目录和文件输出路径，避免把错误推迟到底层 OS 写入失败。
- 双项目 combined summary 现在会正确汇总两个子项目的失败数量，并区分 verification summary 与 pair combined summary。
- `verification-summary-decision-demo` 会先校验 `overall_status`、`failed_count` 和 `sections` 必填字段，避免把非 verification summary JSON 误判成 `ready`。
- 公开 showcase、外部 onboarding/first-run bootstrap、regression smoke 和 regression sample 输出路径都增加了更早的目录/文件输入校验。
- `testloop-mcp --help` 和 `testgen --help` 现在会在打印帮助后返回退出码 0，方便直接作为发布门禁或安装自检命令。

### Removed

- 从仓库移除已跟踪的 `demo-python/__pycache__/*.pyc` Python bytecode 缓存文件。

## v0.5.13 - 2026-07-20

### Added

- `cmd/testgen` 生成成功时会输出 `action=ready` 或 `action=manual_review`，让 CLI 用户在运行测试前就能识别 TODO/skipped 手审草稿。
- 验收报告新增“独立 CLI 生成动作 smoke”，会运行 `testloop-testgen` 或 `go run ./cmd/testgen` 并断言输出 `action=manual_review`。
- 验收 summary JSON 新增可选 `sections[].signals.action`，用于给 Agent/CI 暴露 section 级动作信号。
- `examples/verification-summary-decision-demo`、first-run Agent 回复和 onboarding Agent 回复都会展示 `section_signal=<section> action=<action>`。
- 新增 `docs/fixtures/verification-summary.schema.json` 和 Go schema 回归测试，固定 `verification-summary.json` 的稳定字段和可选 `signals` 结构。
- Agent response artifact manifest 新增 `expected_section_signals`，用于让客户端 fixture 回归固定 summary 与 `agent-response.txt` 中的 section 级动作信号。
- `examples/agent-response-manifest-demo` 会校验 summary schema，并输出 `decision_action=...` 与 `summary_validated=verification-summary.json`，固定 `agent-decision.txt`、summary JSON 与 manifest 契约的一致性。

### Changed

- Agent response artifact contract、客户端契约测试说明、接入指南、客户端集成文档和 README 已补充 `verification-summary.schema.json` 与 `section_signal` 消费说明。
- first-run/onboarding wrapper 回归测试现在固定 summary JSON、`agent-decision.txt` 和 `agent-response.txt` 中的 action signal 链路。
- first-run/onboarding 失败 artifact fixture 已刷新为包含“独立 CLI 生成动作 smoke”的 `signals.action=manual_review` 与 `section_signal` 输出。
- MCP server implementation version 更新为 `0.5.13`。

## v0.5.12 - 2026-07-19

### Added

- Java regression smoke now includes repository-backed RocketMQ `StatusChecker.java` top4 ready-hit samples, covering line-specific protobuf status codes, checked exception branches, and JaCoCo target-hit assertions.
- Java regression smoke task inputs for Commons Lang ready/internal and Commons Codec unreachable samples are now stored under `testdata/`, reducing dependence on local `/tmp` JSONL files.
- JS ip2region regression smoke ready task inputs are now stored under `testdata/`, reducing dependence on local `/tmp` JSONL files.
- Python coverage top-task validation now supports `TESTLOOP_VALIDATE_PY_LIST_TASKS_ONLY`, allowing maintainers to regenerate candidate task JSONL without running generate/run validation.
- Python Click regression smoke ready task inputs are now stored under `testdata/py-click/`, rebuilt against Click `8.2.1` utils coverage tasks instead of stale parser task line ranges.
- JS no-runtime/internal and Python internal regression smoke task inputs are now stored under `testdata/`, reducing runtime JSONL generation for repository-backed manual-review fixtures.
- JS mcp-hub regression smoke task inputs are now stored under `testdata/js-mcp-hub/`, covering repair, env, DevWatcher, SSE, and workspace samples with relative paths.
- Python haoy-apk-station regression smoke task inputs are now stored under `testdata/py-haoy-apk-station/`, covering environment, external-service, and database manual-review samples with relative paths.
- `scripts/fixture-task-jsonl.py` help text and docs now clearly position it as a maintainer-only helper for rebuilding static regression fixture JSONL inputs.
- `scripts/validate-regression-smoke.sh` now runs a preflight check by default to report missing project directories, static JSONL fixtures, and common local commands before starting the longer regression smoke.
- Regression preflight now supports `TESTLOOP_REGRESSION_PREFLIGHT_FORMAT=json`, producing machine-readable `ok`, `missing_count`, `missing`, and `checks` fields for Agent workflows.
- Added `scripts/render-regression-preflight-report.py` to convert preflight JSON summaries into a Chinese Markdown preparation checklist.

### Changed

- MCP server implementation version updated to `0.5.12`.

### Fixed

- Java coverage task generation now maps RocketMQ `StatusChecker.check` line ranges to concrete protobuf `Code` values and expected exception branches, including the default switch path, so JaCoCo target-hit validation no longer reports weak ready tests for these error paths.

## v0.5.11 - 2026-07-19

### Added

- 新增 `docs/agent-response-artifact-contract.md`，统一 first-run/onboarding `agent-response.txt` 的结构、字段、读取顺序和客户端断言。
- 新增 `docs/fixtures/agent-response-artifact-manifest.json`，为客户端测试提供 first-run/onboarding artifact fixture 的机器可读索引。
- 新增 `docs/fixtures/agent-response-artifact-manifest.schema.json` 和 Go schema 回归测试，固定 artifact manifest 的机器可读结构。
- 新增 `examples/agent-response-manifest-demo`，演示客户端如何读取 manifest 自动枚举并校验 artifact fixture。

### Changed

- README、CI 失败 triage 和客户端集成说明已收敛为失败后优先读取 `agent-response.txt`，再按需下钻 decision、summary 和 report。
- `docs/client-integration.md` 和 `docs/fixtures.md` 已链接 Agent response artifact manifest，减少客户端手写 fixture 路径和字段映射。
- `docs/client-integration.md` 和 Agent response contract 已补 manifest demo 命令。
- `docs/mcp-client-contract-tests.md` 已补 CI artifact manifest/schema 回归模板，覆盖 schema 校验、fallback 顺序和 first-run/onboarding action 字段差异。
- README 的 CI 失败 artifact 段落已补 manifest/schema 入口，避免接入方只按单个 first-run fixture 目录做回归。
- README 已补 `agent-response-manifest-demo` 的最小正常输出，方便接入方快速核对 demo 是否运行正常。
- `docs/adopter-verification-guide.md` 已补 artifact manifest/schema 验收入口，把 CI artifact 消费纳入一页式接入清单。
- `docs/quickstart.md` 已补 artifact manifest/schema 快速验证入口，并新增 quickstart 文档测试固定关键命令和链接。
- `docs/installation.md` 已从安装后自检段落指向 artifact manifest/schema 消费回归，并新增 installation 文档测试。
- `docs/fixtures.md` 已补 artifact manifest/schema 维护规则，manifest 测试会固定 `$schema` 指针、schema 文件和维护命令。
- 新增 `docs/plan-release-notes-v0.5.11.md`，整理 v0.5.10 之后的 Agent/客户端 artifact 消费契约候选发布边界。
- 新增 `docs/plan-release-v0.5.11.md`，整理 v0.5.11 候选发布检查清单和正式发布前待办。
- v0.5.11 候选 release readiness 预检已通过，并已记录到发布说明草案和发布检查清单。
- MCP server implementation version 更新为 `0.5.11`。

## v0.5.10 - 2026-07-19

### Added

- 新增 `scripts/render-first-run-agent-response.sh`，可直接从 first-run artifact 目录渲染稳定 Agent 回复，减少接入方手动拼接 `first-run-context.txt` 和 `verification-summary.json` 的成本。
- `scripts/run-first-run-ci.sh` 现在会在 first-run artifact 目录内额外生成 `agent-response.txt`，作为可直接交给 Agent 的四段回复草稿。
- first-run 失败 artifact fixture 已升级为包含 `agent-response.txt` 的六件套，并用测试固定静态文件与目录入口实时渲染结果一致。
- CI 失败 triage 和 first-run CI 模板已收敛读取优先级：新版 artifact 先看 `agent-response.txt`，旧版或机器分流再看 decision/context/summary/report。
- 外部项目 first-run showcase 现在会校验并输出 `agent-response.txt`，确保复制型 bootstrap 的六件套 artifact 在真实 dry-run 中可用。
- 新增 onboarding Agent 回复 demo 和目录入口，`scripts/run-onboarding-ci.sh` 现在会自动生成 `agent-response.txt`，让 onboarding artifact 与 first-run 的 Agent 消费体验对齐。
- 外部项目 onboarding showcase 现在会校验并输出 `agent-response.txt`，确保复制型 onboarding bootstrap 的四件套 artifact 在真实 dry-run 中可用。
- 新增 onboarding 用户项目 smoke 失败 artifact fixture，方便客户端/Agent 不运行 CI 也能回归 onboarding 四件套消费逻辑。

### Changed

- MCP server implementation version 更新为 `0.5.10`。

## v0.5.9 - 2026-07-19

### Added

- 新增 `examples/first-run-agent-response-demo`，可读取 `first-run-context.txt` 和可选 `verification-summary.json`，输出 Agent 应回复的“结论 / 证据 / 下一步 / 暂不做”四段结构。
- 新增 `docs/first-run-agent-artifact-demo.md` 和 `test/first_run_agent_response_demo_test.sh`，固定 first-run artifact 到 Agent 回复的可运行演示和端到端回归。
- 新增 `docs/fixtures/first-run-artifacts/user-project-smoke-failed/`，沉淀 first-run 失败五件套 fixture：report、summary、decision、context 和 log。
- 新增 `test/first_run_artifact_fixtures_test.sh`，验证 first-run artifact fixture 文件完整、summary JSON 可解析，并能被 Agent 回复 demo 消费。
- 新增 `docs/plan-release-notes-v0.5.9.md` 和 `docs/plan-release-v0.5.9.md`，整理 v0.5.9 候选发布说明和发布检查清单。

### Changed

- `docs/fixtures.md` 扩展 first-run artifact fixture 索引，区分 MCP tool 结构化返回 fixture 和 CI artifact fixture。
- `docs/client-integration.md` 增加 CI artifact fixture 消费路径，说明 `agent-decision.txt`、`first-run-context.txt`、`verification-summary.json` 和 `verification-report.md` 的读取顺序。
- README 和 release 文档索引补充 first-run artifact Agent 消费演示、demo 命令和失败 artifact fixture 入口。
- MCP server implementation version 更新为 `0.5.9`。

## v0.5.8 - 2026-07-19

### Changed

- 新增 `docs/plan-release-notes-v0.5.8.md` 和 `docs/plan-release-v0.5.8.md`，整理 v0.5.8 候选发布说明和发布检查清单。
- 新增 `docs/first-run-agent-response.md` 和文档测试，固定 Agent 收到 `first-run-context.txt` 后的回复结构、分流动作和 `inspect-user-project` 示例。
- `docs/ci-agent-triage.md` 增加 first-run 失败态实跑记录，确认故意失败的外部项目 smoke 会分流到 `inspect-user-project`，并修正安装脚本在聚合 `checksums.txt` 缺少当前资产时不继续尝试单资产 `.sha256` 的问题。
- 新增 `docs/ci-agent-triage.md` 和文档测试，说明 CI 失败后如何下载 artifact、读取 `agent-decision.txt` / `first-run-context.txt`，并把最小上下文交给 AI Agent。
- README 的“用户项目接入：直接复制”入口补充最小 GitHub Actions first-run workflow 片段，并新增 README YAML snippet 测试，防止首页 CI 示例漂移。
- README 新增“用户项目接入：直接复制”入口，把 first-run bootstrap、onboarding bootstrap、Go/Vue smoke 命令和 artifact 解读放到首页，减少首次接入跳转成本。
- `docs/real-integration-cases.md` 更新为 v0.5.7 真实 first-run / onboarding CI bootstrap 实跑记录，覆盖 laoxia Go server 和 Vue web 项目，并保留 v0.5.4 onboarding 样例作为历史记录。
- `docs/adopter-verification-guide.md` 补充 `PATH` 版本漂移与 `TESTLOOP_MCP_VERSION` bootstrap 版本门禁的区别，避免手动 MCP 客户端配置误用旧二进制。
- MCP server implementation version 更新为 `0.5.8`。

## v0.5.7 - 2026-07-19

### Added

- 新增 `docs/adopter-verification-guide.md` 和文档测试，把安装、首跑诊断、CI bootstrap、artifact 上传和失败分流压成接入方一页式验证清单。
- 新增 `scripts/showcase-onboarding-ci-external-project.sh`、`docs/onboarding-ci-external-dry-run.md` 和文档测试，用临时非 testloop Go 或 Node 项目验证 Onboarding CI bootstrap 的复制路径不依赖本仓库当前工作目录。
- 新增 `scripts/doctor-first-run.sh`、`docs/first-run-diagnostics.md` 和脚本回归测试，把安装验收、真实 MCP transport、最小 Agent demo、可选用户项目 smoke 收敛成一条首跑诊断入口。
- 新增 `docs/first-run-failures.md`、`docs/fixtures/first-run/*.txt` 和 fixture 测试，固定首跑失败时可粘贴给 AI Agent 的最小上下文。
- 新增 `scripts/run-first-run-ci.sh`、`docs/first-run-ci-template.md` 和模板测试，为外部用户项目提供可复制的首跑诊断 CI bootstrap。
- 新增 `scripts/showcase-first-run-ci-external-project.sh`、`docs/first-run-ci-external-dry-run.md` 和文档测试，用临时非 testloop Go 或 Node 项目验证首跑诊断 CI bootstrap 能生成五件套 artifact。
- 新增 `docs/plan-release-notes-v0.5.7.md` 和 `docs/plan-release-v0.5.7.md`，整理 v0.5.7 候选发布说明和发布检查清单。

### Changed

- README 和 showcase 索引补充外部项目 Onboarding CI 演练入口，便于发布后复验接入文档。
- `scripts/verify-mcp-process-smoke.sh` 进入 testloop-mcp 仓库后再执行 Go smoke，避免从用户项目 Go module cwd 调用时触发 `outside main module`。
- quickstart 优先推荐首跑诊断脚本，用户可以直接按 `first_run_agent_next_step` 判断下一步。
- `scripts/doctor-first-run.sh` 会额外写出 `first-run-context.txt`，便于用户把首跑失败上下文交给 AI Agent。
- 验收 CI 文档补充首跑诊断 CI 模板入口，区分 onboarding 三件套和 first-run 五件套。
- README 和 showcase 索引补充首跑诊断 CI 外部项目演练入口。
- 验收 CI 文档补充 onboarding 与 first-run bootstrap 的选择规则，README 增加直达链接。
- README、showcase 索引和 release 文档索引补充接入方一页式验证指南入口。
- MCP server implementation version 更新为 `0.5.7`。

## v0.5.6 - 2026-07-18

### Added

- 新增 `scripts/run-onboarding-ci.sh` 和 `test/run_onboarding_ci_test.sh`，把外部用户项目 CI 中的安装、脚本准备、项目 smoke 和 onboarding artifact 生成收敛成一个 bootstrap 入口。
- 新增 `docs/onboarding-ci-failure-triage.md` 和 `test/onboarding_ci_failure_triage_doc_test.sh`，说明 onboarding CI 失败时如何读取 step summary、artifact 和 `agent_next_step`。
- 新增 `docs/onboarding-ci-template.md` 和 `test/onboarding_ci_template_doc_test.sh`，提供 Go server 与 Vue / Node 项目可复制的 onboarding CI workflow，固定 Markdown、summary JSON 和 `agent_next_step` artifact 路径。
- 新增 `test/onboarding_ci_template_yaml_test.sh`，解析文档里的完整 workflow YAML 示例，防止复制模板语法漂移。

### Changed

- README、showcase 和验收 CI 文档补充 Onboarding CI 复制模板入口，让首次接入用户优先复制最小 workflow，再按需阅读完整说明。
- Onboarding CI 复制模板改用 `scripts/run-onboarding-ci.sh` bootstrap，避免外部用户仓库直接引用不存在的 repo-local `scripts/showcase-agent-onboarding-report.sh`。
- 使用 `TESTLOOP_MCP_VERSION=v0.5.6 scripts/run-onboarding-ci.sh 'go test ./...'` 完成真实 dry-run，基础安装验收、真实 MCP 协议 smoke、最小 Agent demo 和用户项目 smoke 均通过，`agent_next_step=ready`。
- `scripts/run-onboarding-ci.sh` 在 GitHub Actions 中会写入 `$GITHUB_STEP_SUMMARY`，失败时直接展示状态、失败数量、artifact 路径和下一步建议。
- MCP server implementation version 更新为 `0.5.6`。

## v0.5.5 - 2026-07-18

### Added

- 新增 `docs/real-integration-cases.md` 和 `test/real_integration_cases_doc_test.sh`，把 laoxia Go server / Vue web 的 onboarding report 实跑结果整理成可复用真实接入案例模板。
- 新增 `docs/plan-release-notes-v0.5.5.md` 和 `docs/plan-release-v0.5.5.md`，整理 v0.5.5 候选发布说明和本地 release readiness 门禁。

### Changed

- `scripts/verify-client-setup.sh` 在旧二进制缺少 `--version` 或版本不匹配时输出 Homebrew 升级/重装建议，降低安装漂移排查成本。
- MCP server implementation version 更新为 `0.5.5`。

## v0.5.4 - 2026-07-18

### Added

- 新增 `scripts/showcase-agent-onboarding-report.sh`，把 onboarding 验收报告、summary JSON 和 `agent_next_step` 决策输出收敛成一个公开演示入口。
- 新增 `test/showcase_agent_onboarding_report_test.sh`，固定一键演示报告脚本的 artifact 路径、summary JSON 和 decision 输出契约。
- 新增 `docs/verification-summary-failures.md` 和 summary JSON 失败 fixture，展示安装、MCP 协议、Agent demo、公开 showcase、用户项目 smoke 失败时的 `agent_next_step` 分流。
- 新增 `docs/plan-agent-onboarding-v0.5.4.md`，规划 v0.5.4 的公开 onboarding demo 收敛方向。
- 新增 `docs/plan-release-notes-v0.5.4.md` 和 `docs/plan-release-v0.5.4.md`，整理 v0.5.4 候选发布说明和本地 release readiness 门禁。

### Changed

- `docs/verification-ci.md` 优先推荐 `scripts/showcase-agent-onboarding-report.sh`，让接入方用更少环境变量生成 Markdown、summary JSON 和 decision artifact。
- MCP server implementation version 更新为 `0.5.4`。

## v0.5.3 - 2026-07-18

### Added

- 新增 `scripts/generate-verification-report.sh`，可把基础安装验收、真实 MCP 协议 smoke、最小 Agent 闭环 demo、可选公开 showcase 和用户项目 smoke 聚合成 Markdown 验收报告。
- 验收报告脚本支持 `TESTLOOP_REPORT_SUMMARY_JSON`，可额外输出机器可读 summary JSON，方便 Agent / CI 直接读取 `overall_status`、`failed_count` 和 section 状态。
- 新增 `examples/verification-summary-decision-demo` 和 `test/verification_summary_decision_demo_test.sh`，演示 Agent / CI 如何读取 summary JSON 并区分安装、协议、Agent demo、公开 showcase 和用户项目 smoke 失败。
- 新增 `docs/verification-ci.md` 和 `test/verification_ci_doc_test.sh`，提供 GitHub Actions 中生成 Markdown + JSON 验收报告、执行决策 demo、失败时上传 artifact 的可复制示例。
- 新增 `docs/verification-report.md` 和 README / showcase 索引入口，说明默认离线验收、公开 showcase opt-in、用户项目 smoke 命令和报告解读方式。
- 新增 `test/verification_report_test.sh` 并纳入 CI，固定验收报告脚本的跳过项、用户项目成功项、失败 exit code 和报告内容。
- `docs/verification-report.md` 新增 laoxia Go server 与 Vue web 的真实项目 smoke 记录，确认报告脚本可覆盖 server `go test ./...` 和 web `pnpm build:prod` 两类接入路径。
- 新增 `docs/plan-release-notes-v0.5.3.md` 和 `docs/plan-release-v0.5.3.md`，整理 v0.5.3 候选发布说明和本地 release readiness 门禁。

### Changed

- MCP server implementation version 更新为 `0.5.3`。

## v0.5.2 - 2026-07-18

### Added

- 主二进制新增 `--version`，安装后自检脚本可通过 `TESTLOOP_MCP_VERIFY_EXPECT_VERSION` 校验当前 PATH 或指定二进制是否为预期版本。
- 新增 `scripts/verify-mcp-process-smoke.sh` 和 `examples/mcp-process-smoke`，用真实 MCP SDK 客户端通过 stdio / Streamable HTTP 启动安装后的二进制并调用轻量工具。
- 公开 Go / JS showcase 脚本新增默认 action 期望校验，外部项目案例的 `ready` / `manual_review_internal` 决策信号漂移时会直接失败。
- 公开 Go / JS showcase 脚本支持通过 `TESTLOOP_SHOWCASE_*_PROJECT_DIR` 复用本地 checkout，JS 脚本还支持 `TESTLOOP_SHOWCASE_JS_SKIP_INSTALL=true` 跳过依赖安装。
- 公开 Go / JS showcase 脚本新增远端 `git clone/fetch` 超时控制，默认 60 秒，可通过 `TESTLOOP_SHOWCASE_*_GIT_TIMEOUT` 调整，避免 GitHub 网络不可达时长时间挂起。
- 公开 JS/TS showcase 已用远端 `unjs/ufo` 固定 commit 复验通过，默认 `vitest-1=manual_review_internal`、`vitest-2=ready` 决策信号保持稳定。
- 新增 `scripts/summarize-showcase-output.py`，统一公开 showcase 的 JSONL summary 输出和 action 断言逻辑，避免 Go / JS 脚本重复维护。
- 新增 `docs/plan-release-notes-v0.5.2.md` 和 `docs/plan-release-v0.5.2.md`，整理 v0.5.2 候选发布说明和本地 release readiness 门禁。

### Changed

- MCP server implementation version 更新为 `0.5.2`。

## v0.5.1 - 2026-07-17

### Added

- 新增 `examples/mcp-client-demo` 最小 MCP 客户端端到端 demo，演示客户端优先消费 `structuredContent`，串联 `run_tests -> repair_task -> rerun -> parse_coverage`。
- 新增 `docs/agent-contract.md` 和 `types` 层 Agent JSON contract 测试，固定 `repair_task`、`provider_error`、`validate_coverage_task` 等关键结构化字段名。
- 新增真实 stdio 进程级 e2e smoke，通过 MCP SDK `CommandTransport` 构建并启动当前 `testloop-mcp` 二进制，验证 `tools/list` 和 `parse_results` 的结构化返回一致性。
- 新增真实 Streamable HTTP 进程级 e2e smoke，验证 `/healthz`、`StreamableClientTransport`、`tools/list` 和 `parse_results` 的结构化返回一致性。
- 新增真实二进制级客户端配置 smoke，验证 `--print-config=all` 生成的 Codex / Codex HTTP / Claude / Cursor 配置可被同一二进制 `--check-config -` 校验通过。
- 新增 `scripts/verify-client-setup.sh`，把二进制可执行性、`--doctor-config`、配置 roundtrip 和 HTTP `/healthz` 收敛为安装后自检入口。
- 新增 `test/verify_client_setup_test.sh` 并纳入 CI，固定安装后自检脚本的 skip HTTP 路径和缺失二进制错误提示。
- 新增 `docs/quickstart.md`，把安装、自检、Codex/Claude/Cursor 配置和最小 Agent 闭环收敛成 5 分钟接入路径。
- 新增 `docs/showcase-agent-loop.md` 和 `test/mcp_client_demo_test.sh`，固定最小 MCP 客户端闭环 demo 的预期输出，并纳入 CI 回归。
- 新增 `scripts/showcase-go-public-project.sh` 和 `docs/showcase-public-go.md`，用固定 commit 的 `google/uuid` 展示公开 Go 项目的 `validate_coverage_task` 闭环。
- 新增 `scripts/showcase-js-public-project.sh` 和 `docs/showcase-public-js.md`，用固定 commit 的 `unjs/ufo` 展示 JS/TS 项目的 `ready` 与 `manual_review_internal` 决策分流。
- 新增 `docs/showcase.md`，统一说明默认 CI、公开 opt-in showcase 和真实项目 regression smoke 的边界。
- 新增 `test/showcase_scripts_test.sh` 并纳入 CI，固定公开 showcase 脚本的帮助输出、参数错误和缺少 `pnpm` 的提示。
- 新增 `validate_coverage_task` 结构化返回一致性测试，固定 `structuredContent` 与文本 JSON 中的 `status/action/coverage_task/generated/run_result/metadata` 不漂移。
- 新增 `docs/agent-action-guide.md`，整理 `validate_coverage_task.status/action` 到客户端下一步动作的决策表。
- 新增 `test/docs_links_test.sh` 并纳入 CI，检查 README 与 docs 下 Markdown 相对链接的文件目标是否存在。
- 新增 `docs/validate-coverage-task-samples.md` 和 `test/docs_json_examples_test.sh`，固定 `validate_coverage_task` 典型结构化返回样例的 JSON 合法性。
- 新增 `examples/agent-decision-demo` 和 `test/agent_decision_demo_test.sh`，演示客户端如何把 `validate_coverage_task.status/action` 映射成下一步动作。
- 新增 `docs/fixtures/validate-coverage-task-ready.json` 和 handler 级 fixture 测试，用真实临时 Go 项目固定 `validate_coverage_task` 的 ready 样例投影。
- 新增 `docs/fixtures/validate-coverage-task-manual-review-internal.json` 和 handler 级 fixture 测试，固定 `passed/manual_review_internal` 的真实结构化样例投影。
- 新增 `docs/fixtures/validate-coverage-task-apply-fix-suggestions.json` 和 handler 级 fixture 测试，固定 `failed/apply_fix_suggestions` 的真实修复闭环样例投影。
- 新增 `docs/fixtures.md`，集中说明真实结构化 fixture 的来源、Agent 分流、稳定字段和过滤规则。
- 新增 `test/fixtures_index_test.sh` 并纳入 CI，校验 `docs/fixtures/*.json` 的 `status/action` 覆盖清单已登记到 fixture 索引。
- 新增 `docs/client-integration.md`，说明客户端消费 `structuredContent`、复用真实 fixture 和回归 `status/action` 分流的推荐流程。
- 新增 `test/client_integration_doc_test.sh` 并纳入 CI，校验客户端集成说明引用的 fixture 文件和 `agent-decision-demo` 入口持续存在。
- 新增 `docs/fixtures/validate-coverage-task-needs-better-input.json` 和 handler 级 fixture 测试，固定 `failed/needs_better_input` 的真实 JaCoCo 目标行未命中样例投影。
- `examples/agent-decision-demo` 改为直接读取 `docs/fixtures/*.json`，让最小客户端决策 demo 与真实 handler fixture 保持一致。
- 新增 `test/fixture_decision_mapping_test.sh` 并纳入 CI，直接校验每个真实 fixture 的 `status/action` 到客户端动作映射。
- 新增 `docs/mcp-client-contract-tests.md`，说明接入方如何复制真实 fixture、demo 和契约校验到自己的客户端 CI。
- 新增 `test/release_doc_index_test.sh` 并纳入 CI，固定 README 中 Agent/客户端关键文档入口和 demo 命令。
- 新增 `docs/plan-release-notes-v0.5.1.md` 和 `docs/plan-release-v0.5.1.md`，整理 v0.5.1 候选发布说明、发布前门禁和正式发布待办。

### Changed

- MCP server implementation version 更新为 `0.5.1`。

## v0.5.0 - 2026-07-17

### Added

- 新增 `docs/plan-release-notes-v0.5.0.md`，把当前固定 smoke 矩阵、Agent 闭环定位、真实项目验证证据和 v0.5.0 发布前门禁整理成中文发布草案。
- README 新增“面向 Agent 的快速演示路径”，推荐用 `go test ./...` + 固定 smoke 展示 `generate_tests -> run_tests -> parse/fix/coverage feedback` 闭环。

### Changed

- MCP server implementation version 更新为 `0.5.0`。
- Go coverage task 写入测试文件前会扫描同包所有 `*_test.go`，当任务推荐的 `test_name` 已在其它测试文件中存在时，也会自动追加稳定后缀，避免生成后 `go test` 因包级 `Test*` 重名构建失败。
- Java coverage task 支持 Commons Codec `DigestUtils.sha` 重载和 `getShake*Digest` 运行时兼容路径：生成 typed `byte[]` / `InputStream` / `String` 输入，避免裸 `null` 重载歧义；SHAKE MessageDigest 不可用时断言 `IllegalArgumentException` 信息而不是误报生成失败。
- Java coverage task 支持 Commons Codec `language/bm` 资源规则与 nested value object 场景：`SomeLanguages` 使用 factory 构造，`PhoneticEngine.getLang` 使用合法 `RuleType.APPROX`，`Lang.loadFromResource` / `PhoneticEngine.encode` 归为资源手审边界，`Rule.RPattern` 等嵌套返回类型会自动限定，文件级 Java task 会生成可运行手审 smoke。
- Java coverage task 支持 `StringEncoder.encode(Object) throws EncoderException` 适配方法：错误路径生成非 String 对象触发 `EncoderException`，返回路径生成 String 输入断言结果，避免 `null` 参数和空 lambda `assertThrows` 导致真实项目失败。
- Java coverage task 命中 private nested class/interface/enum 时会生成 `manual_review_internal` 手审 smoke，避免直接构造私有嵌套类型导致真实 Maven/JUnit 项目编译失败。
- Java coverage task 命中确认不可达的公共路径时可生成 `manual_review_unreachable` 手审 smoke，`validate_coverage_task` 会识别该标记，避免把“测试通过但未命中目标缺口”的弱用例暴露为 ready。
- Java/JUnit `validate_coverage_task` 会在运行生成测试时默认收集 JaCoCo 覆盖率，并校验 `coverage_task.line_range` 是否真正命中；测试通过但目标行仍未覆盖时会返回 `failed/needs_better_input`，metadata 中包含 `coverage_hit_lines` 和 `coverage_missed_lines`。
- Java 真实项目验证脚本支持 `TESTLOOP_VALIDATE_JAVA_TASK_IDS` 精确筛选 task id；未显式传入 limit 时会按 id 数量自动收敛验证窗口，便于快速回归历史弱 ready 样本。
- Java 真实项目验证脚本支持 `TESTLOOP_VALIDATE_JAVA_TASKS_FILE` 从已有 coverage task / validation JSONL 直接读取任务，跳过 baseline coverage 生成，降低单任务回归成本。
- JS/TS 与 Python 真实项目验证脚本支持 `TESTLOOP_VALIDATE_JS_TASKS_FILE` / `TESTLOOP_VALIDATE_PY_TASKS_FILE` 和 `TESTLOOP_VALIDATE_JS_TASK_IDS` / `TESTLOOP_VALIDATE_PY_TASK_IDS`，可复用已有任务 JSONL 并跳过 baseline coverage。
- Java/JUnit `run_tests` 当 path 指向 `src/test/**/*.java` 时会只运行该测试类，同时保留 JaCoCo report 生成，减少 `validate_coverage_task` 单任务回归的 Maven/JUnit 运行耗时和上游 skipped tests 干扰。
- 新增 `scripts/validate-java-regression-samples.sh`，把 Java 真实 ready、历史假 ready 降级和内部手审三类样本固化为小型回归入口，并断言输出 JSONL 的 status/action/目标行命中元数据。
- 新增 `scripts/validate-js-regression-samples.sh`、`scripts/validate-py-regression-samples.sh` 和 `scripts/validate-regression-smoke.sh`，把 ip2region Jest ready、仓库内 TypeScript no-runtime/internal fixture、Click pytest ready 与仓库内 Python internal fixture 纳入固定 smoke 矩阵，并串联 Java + JS + Python 小回归。
- JS coverage task 对 `branch` 任务会识别命中的 `if (...) { throw ... }` 分支，并为 async 方法生成 `await expect(...).rejects.toThrow()`；mcp-hub `ConfigManager.loadConfig` 历史 `repair_generated_test` smoke 已收敛为真实 `ready`。
- JS validation helper 支持 `TESTLOOP_VALIDATE_JS_ALLOWED_FAILURE_ACTIONS`，只在脚本显式声明期望失败 action 时放行，默认 top-N 验证仍保持严格失败。
- Python regression smoke 新增 haoy-apk-station backend 真实 FastAPI `manual_review_environment` 样本，固定 `app.main` 动态前端入口依赖 `frontend/dist` 导入时环境的降级行为。
- Python regression smoke 新增 haoy-apk-station backend 真实 FastAPI `manual_review_external_service` 样本，固定 `download_apk` 对象存储 endpoint timeout 会输出 `failed/manual_review_external_service`，不进入普通 repair。
- Python coverage task 新增 SQLAlchemy/database 事务错误手审分类，并把 haoy-apk-station backend `delete_app` 的 `db.commit` 失败路径固定为 `manual_review_database` smoke。
- 新增 `docs/regression-smoke.md`，记录固定 smoke 的默认项目路径、JSONL 依赖、跳过开关、runner 约束以及 JS/Python fixture 样本边界。
- 新增仓库内 `testdata/js-no-runtime`、`testdata/js-internal` fixture 与 `scripts/js-manual-review-runner.js`，让 JS regression smoke 可稳定覆盖 `manual_review_no_runtime` 和 `manual_review_internal`，不再依赖已漂移的外部 TS 项目样本。
- 新增 Python name-mangled private method 生成规则、`testdata/py-internal` fixture 与 `scripts/py-manual-review-runner.py`，让 Python regression smoke 可稳定覆盖 `manual_review_internal`。
- 新增 `scripts/fixture-task-jsonl.py`，统一生成 JS/Python fixture coverage task JSONL，避免 regression 脚本继续内联重复 JSON 构造逻辑。
- JS/Python manual-review fixture runner 会从生成测试文件中提取真实 skipped test 名称或 pytest node id，并输出更接近 Jest/Pytest 的摘要，降低 parser smoke 对固定文案的依赖。
- Java coverage task 修正 Commons Codec `Metaphone.metaphone` 的弱 ready：目标 279 行被 JaCoCo 映射到被 `GN` 短路遮蔽的 `GNED` 侧，真实目标行命中校验后改为 `manual_review_unreachable`；`Soundex.getMaxLength` / `setMaxLength` 继续断言真实默认值和状态变化。
- Java coverage task 命中裸类型变量数组或 varargs（例如 `T[]` / `T...`）时会生成 `manual_review_internal` 手审 smoke，避免输出不可编译的 `T[] result` 或重载歧义的 `addAll(null, null)`。
- Java coverage task 补强 Apache Commons Lang `ClassUtils` 公共 helper 场景：`getShortClassName` 会用 JVM 数组编码输入覆盖对象数组和 primitive array 分支，`hierarchy` 会调用 `iterator.remove()` 覆盖 `UnsupportedOperationException` 路径，避免把未命中缺口的弱 ready 暴露给 Agent。
- Java coverage task 补强 Apache Commons Lang `CharSequenceUtils.toCharArray` 的 `StringBuffer` 分支：按 `line_range` 使用 `new StringBuffer("test")` 并断言字符数组内容，避免用普通 `String` 输入生成通过但不命中目标行的弱 ready。
- Java coverage task 补强 Apache Commons Lang `StopWatch` 状态路径：`split(String)` 和 `getStopInstant` 的未启动状态会生成 `IllegalStateException` 断言，`getNanoTime` 的 switch default 防御分支会归为 `manual_review_unreachable`，避免普通调用失败或弱 ready。
- Java coverage task 修正 public nested class 误判：`private enum SplitState` 这类前缀不再导致 `StopWatch.Split` 被归为私有嵌套类型，`StopWatch.Split.toString` 会生成真实构造和精确字符串断言。
- Java coverage task 补强 Apache Commons Lang `ExceptionUtils.throwUnchecked`：按 `line_range` 生成 `RuntimeException`、`Error` 和 checked-return 断言，避免泛型返回类型 `T` 泄漏到测试代码导致 Maven/JUnit 编译失败。
- Java coverage task 补强 Apache Commons Lang `ExceptionUtils.asRuntimeException` / `rethrow`：type-erasure 异常传播路径会断言原始 `RuntimeException` 被抛出，不再把泛型返回类型 `T` 当作测试局部变量类型。
- Java coverage task 补强 Apache Commons Lang `Failable` wrapper：`tryWithResources` 会按 `line_range` 生成 functional interface 输入、资源关闭异常和 errorHandler 断言；`get*` / `run` wrapper 会用 throwing lambda 断言原始 `RuntimeException` 被 rethrow，避免退化成 `tryWithResources(null, null, null)`、`T result` 或 `get*(null)`。
- Go static generator 支持普通参数校验触发的多返回值 error 分支，例如 `if socketPath == "" { return Status{}, fmt.Errorf(...) }` 会生成非 skipped 测试，断言非 error 返回为零值、error 返回非 nil；变参函数的参数校验分支也会进入同一生成路径。
- Go coverage task 分支匹配会优先使用 `line_range` 区分同一函数内重复的 `err != nil` 分支，并对 `net.Dial("unix", socketPath)` 连接失败分支生成缺失 socket 路径测试输入，避免把后续协议读写错误误判为连接失败。
- Go static generator 支持 Unix socket 协议错误路径输入合成，可用本地 `net.Listen("unix", ...)` 稳定触发 `ReadBytes` EOF 和 `json.Unmarshal` 非法 JSON 分支。
- Go static generator 支持 Unix socket JSON 响应分支输入合成，可覆盖 daemon client 的默认错误响应和 invalid status 复合分支。
- `validate_coverage_task` 会将静态生成器无法稳定构造的 socket write / streaming I/O 错误分支标记为 `manual_review_protocol`，避免继续以普通 `ready` skipped TODO 暴露给 Agent。
- `validate_coverage_task` 会将静态生成器无法安全构造的 GORM/数据库错误分支标记为 `manual_review_database`，避免在项目没有测试数据库策略时继续以普通 `ready` skipped TODO 暴露给 Agent。
- 新增 JS/Vitest 真实项目 top coverage task 验证脚本，支持测试子集参数和文件过滤，用于复用 `coverage_task -> generate_tests -> run_tests` 样本回归。
- `run_tests` 不再为 Vitest 追加已被 Vitest 3 拒绝的 `--verbose` 参数，并会把 Vitest/Jest 命令级错误解析为失败而不是误判通过。
- JS/Vitest coverage task 在项目已有 `tests/` 目录且源码位于 `src/` 下时，会把生成测试写入 `tests/` 镜像路径，并按测试文件位置生成相对 import，避免被真实项目的 Vitest `include` 配置排除。
- JS class method coverage task 支持从 `this.strict`、`this.maxPasses` 和 placeholder 返回分支推导实例构造参数与方法入参，并避免 return-path 因方法体存在其他 `throw` 分支而误生成错误断言。
- JS class coverage task 遇到 JavaScript `#private` method 时不再生成非法的 `instance.#method()` 外部调用，而是生成 `it.skip` 的 manual-review 草稿，并在 metadata 中返回可检测到的公共入口候选。
- JS class coverage task 可通过 `ConfigManager.loadConfig()` 公共入口覆盖 `ConfigManager.#diffConfigs` 私有分支，自动生成临时 config 文件、旧配置状态和 changes 断言。
- JS class coverage task 可通过 `DevWatcher.start()` 公共入口覆盖 `DevWatcher.#handleFileChange` 私有分支，自动生成 Vitest `chokidar` mock、fake timers、watcher 事件和 `filesChanged` 断言。
- JS class coverage task 可通过 `MCPHubOAuthProvider` 和模块动态导入覆盖未导出的 `StorageManager.init/get`，自动生成 `fs/promises`、logger mock 和默认导出 provider 断言。
- JS class coverage task 支持 `WorkspaceCacheManager.updateWorkspaceState` 这类缓存状态更新分支，自动预置 workspace cache、mock `_readCache/_writeCache/_withLock`，并断言写入的合并状态。
- JS coverage 验证脚本支持 `TESTLOOP_VALIDATE_JS_EXTRA_SYMLINKS`，可在隔离 worktree 中挂载 monorepo 父级资源，例如 `ip2region` JS 子包依赖的 `data/` 目录。
- JS function coverage task 支持 `versionFromHeader` 这类对象参数分支输入合成，并可通过公开 `parseIP()` 入口覆盖未导出的 `_parse_ipv4_addr/_parse_ipv6_addr` 错误分支。
- JS class coverage task 支持 `ip2region` 这类带状态 class 的最小实例构造：`Version.ipCompare` 会注入 compare callback，`Searcher.search/read/toString` 会用内存 buffer、临时 `fs.readSync` 替换和合法 version 结构覆盖二进制搜索、短读异常和字符串返回路径，避免 ESM Jest 下依赖不存在的全局 `jest`。
- JS/TS coverage task 支持通过 `CodexExec.run` 公共入口覆盖未导出的 `flattenConfigOverrides` / `toTomlValue` 配置序列化 helper，自动生成 ESM Jest `child_process.spawn` mock、合法 `CodexExecArgs`、config override 断言和 `@ts-nocheck` mock 草稿，避免 TypeScript/Jest 项目直接 import 内部函数失败。
- JS/TS coverage task 对未导出的 `findCodexPath` 会通过 `CodexExec` 构造器覆盖 unsupported platform/arch 分支；对依赖内部 platform package map 或 optional native package 布局的分支生成 `manual_review_internal` 草稿，避免继续生成非法命名 import。
- JS/TS coverage task 支持 `resolveNativePackage` 的缺失 native package 返回 `null` 分支，生成类型合法的字符串入参；未导出的 `serializeConfigOverrides` 会复用 `CodexExec.run` 公共入口覆盖返回路径。
- JS/TS coverage task 会通过 `CodexExec.run` 公共入口覆盖未导出的 `formatTomlKey` / `isPlainObject` helper，使用数组对象配置值触发 quoted TOML key formatter，并用数组配置值覆盖非 plain object 判定。
- JS/TS coverage task 遇到未导出的 `isDirectory` 这类内部文件系统 helper 时，会生成 `manual_review_internal` 草稿并标注 `findCodexPath` / `resolveNativePackage` 公共入口候选，避免错误生成非法命名 import。
- JS/TS coverage task 会为 `CodexExec.run` 参数分支生成分支专属断言，覆盖 `--model`、`--sandbox`、`--cd`、`--add-dir`、`--output-schema`、网络/搜索/审批配置、PATH prepend、`CODEX_API_KEY` 和缺失 stdin/stdout 错误路径，避免用泛化 spawn error 测试掩盖低价值覆盖。
- JS/TS coverage task 会为 `CodexExec.run` 的 stdout yield 分支生成 `for await` 收集断言，兼容不支持 `Array.fromAsync` 的 Jest/Node 环境。
- JS/TS coverage task 会为 Codex SDK 配置序列化分支生成更具体的 TOML 断言，覆盖 inline object、对象中 `undefined` child skip、unsupported value 错误路径，以及 `CodexExec.run` 内 config override loop。
- JS/TS coverage task 会通过 `CodexExec(null)` 公共入口和临时覆盖 `process.platform/process.arch` 覆盖 `findCodexPath` 的 linux/darwin/win32 平台映射分支，将仅依赖平台选择的任务从 `manual_review_internal` 转为 ready。
- JS/TS coverage task 遇到文件级目标任务时会生成 `manual_review_internal` 草稿，避免回退成全量导入并错误引用未导出的内部 helper。
- JS/TS coverage task 会通过 `Thread.runStreamed()` 公共入口覆盖 `Thread.runStreamedInternal` private 分支，自动生成 async generator `CodexExec.run` mock、`thread.started` id 更新断言和 JSON parse error 断言，把 Codex SDK `src/thread.ts` top18 的 private skipped 草稿转为真实 ready 测试。
- JS/TS coverage task 会通过 `Thread.runStreamed()` 公共入口覆盖未导出的 `normalizeInput`，生成 structured text + local image 输入，并断言传给 `CodexExec.run` 的 prompt 合并结果和 images 数组。
- JS/TS coverage task 会通过 `createOutputSchemaFile()` 覆盖未导出的 `isJsonObject` plain object 分支，并通过 Codex SDK 测试辅助模块的 `createTestClient()` 覆盖未导出的 `hasExplicitProviderConfig` 分支，避免直接 import 内部 helper。
- JS/TS coverage task 会通过 Codex SDK 测试辅助模块的 `createTestClient()` 覆盖未导出的 `getCurrentEnv` 分支，断言普通 env 会继承、`CODEX_INTERNAL_ORIGINATOR_OVERRIDE` 会被过滤，并在测试后恢复 `process.env`。
- JS/TS coverage task 支持 Codex SDK `tests/responsesProxy.ts` 辅助模块：`formatSseEvent` 会通过公开 `startResponsesTestProxy()` 的真实 HTTP POST/404/generator exhausted 行为间接覆盖，`responseFailed()` 会断言返回的 error event 对象；无法稳定触发的 Node `server.address()` / `server.close(err)` 内部分支会生成 `manual_review_internal` 草稿。
- JS/TS coverage task 在 ESM 源文件中遇到未导出的顶层函数时会生成 `manual_review_internal` 草稿，避免错误生成非法 named import；CommonJS coverage task 仍保留 `require()` 路径。
- JS/TS parser 会识别 TypeScript `private` / `protected` class method 和 `get` accessor；coverage task 对不可外部调用的 TS private method 会生成 `manual_review_private` 草稿和公共入口候选，对 getter 会生成属性访问而不是错误的函数调用。
- JS/TS class coverage task 会利用 TypeScript 参数类型生成更合法的 constructor / method 入参，例如为 `CodexExec` 注入最小 async generator mock、为 `Input` 生成字符串输入、为 options 生成对象、为 nullable id 生成 `null`，减少 TS 严格模式下的 `undefined` 编译失败。
- JS/TS coverage task 针对 `Thread.run` 这类消费 event stream 的错误路径，会让 `CodexExec` mock 产出 `turn.failed` 事件，从而稳定覆盖 reject 分支。
- JS/TS coverage task 支持 `createOutputSchemaFile` 的非法 schema 和写文件失败 cleanup 分支，使用 `node:fs` / `@jest/globals` 动态 import 的 spy 触发 `writeFile` reject 并断言 `rm` cleanup。
- JS/TS class coverage task 对 `Codex` 包装类会生成 `codexPathOverride`，避免 `resumeThread` 这类纯包装方法测试在构造阶段触发 native CLI package lookup。
- JS/TS coverage task 遇到 TypeScript 纯类型文件时会生成 `manual_review_no_runtime` 草稿，明确这类文件没有可执行运行时代码，应通过消费方测试或类型检查验证，而不是继续按覆盖率缺口生成伪单测。
- JS/TS generation context 对 type-only TS 文件会保留 `types` 信息；JS 真实项目验证脚本在文件过滤命中源码但 coverage report 没有任务时，会合成 no-runtime 文件级任务，并优先写入项目已有 `tests/` 目录以适配 Jest/Vitest `testMatch/include`。
- JS/TS no-runtime 文件级任务会覆盖 TypeScript barrel re-export 文件，例如 `index.ts` 只做 `export type` / `export { ... }` 时会生成 `manual_review_no_runtime`，提示通过消费方测试验证包入口。
- JS/TS coverage task 支持 `prependPathDirs` / `pathEnvKey` 的 PATH 归一化分支，生成类型合法的 env/pathDirs/platform 输入，并断言 Windows PATH key 合并和非 Windows `PATH` 保留行为。
- JS/TS coverage task 会通过 `resolveNativePackage` 公共入口覆盖未导出的 `existingDirs` / `isFile` helper，构造临时 native package vendor 目录并断言 `executablePath` 与 `pathDirs`，避免直接 import 内部函数。
- `validate_coverage_task` 会将 JavaScript `#private` method 任务标记为 `manual_review_private`，避免把语言访问性限制当成普通生成测试失败反复修。
- JS class coverage task 遇到 ESM 文件中未导出的内部 class 时，会生成 `manual_review_internal` 草稿而不是错误生成命名导入，例如 `StorageManager` 这类模块内部状态 helper。
- JS class coverage task 会解析 constructor 参数，并为 `serverName` / `devConfig` / `options` 这类常见参数生成最小实例化输入，例如 `new DevWatcher('test-server', { enabled: true, watch: [], cwd: process.cwd() })`。
- JS class coverage task 为 Express 风格 `req` / `res` 参数生成最小 mock，覆盖 `setHeader`、`write`、`end`、`on` 和 `writableEnded`，让 SSE handler 类方法可以先稳定跑通。
- JS class coverage task 支持默认导出实例，例如 `class Logger` + `export default logger` 会生成 `import logger from ...` 并通过实例调用方法，避免错误生成不可导出的 `new Logger()`。
- JS coverage task 对 `error` / `err` 参数会生成普通 `Error` 对象，并把 `new ErrorLike(...)` 返回路径识别为 object，减少错误包装 helper 的无效边界输入。
- Jest/Vitest parser 支持 Vitest 3 的 `Tests  1 skipped (1)` 摘要和 `↓` skipped 结果行，确保 validation summary 能准确统计 manual-review 草稿。
- Go 测试文件写入会对新建文件和合并文件统一执行 import 整理，避免 coverage task 只生成单个目标测试时保留未使用 import 导致构建失败。
- Go return 表达式提取支持空 composite literal，例如 `Status{}`，用于识别多返回值 error 分支中的零值返回。
- Go static generator 支持泛型 helper 的 `return &param`、nil 指针返回零值和非 nil 指针解引用返回路径，例如 `anyPtr[T]` / `derefAny[T]` 会生成真实指针值或返回值断言。
- Go static generator 支持 nil pointer receiver 的字符串分支，例如 `(*BizError).Error()` 的 `receiver == nil` 分支会生成非 skipped 测试并断言空字符串。
- Go static generator 支持 JWT `Parse(secret, raw)` 的常见错误分支，可生成错误签名算法 token 或非法 token 输入，并自动补齐 `/vN` 语义版本 import 的源码包名别名。
- Go static generator 支持 Gin `FailWithErr` 这类 response helper 分支，会生成 `gin.CreateTestContext`、`httptest.ResponseRecorder` 和 JSON response 断言；seed 也支持显式 import alias，避免业务 `errors` 包与标准库包名冲突。
- Go static generator 支持 `logx.Init(config.Log)` 这类全局 logger 初始化分支，会生成全局状态恢复、临时工作目录、日志级别断言、caller marshal 断言、目录创建错误路径和 dev writer 分支测试。

## v0.4.14 - 2026-07-11

### Added

- 新增 `validate_coverage_task` MCP 工具，可对单个 `parse_coverage.test_tasks[]` 执行 `generate_tests -> run_tests` 闭环，并返回 `passed` / `failed` / `generation_error`、建议动作、生成结果、测试结果和 provider/fix 反馈。
- 新增 `scripts/validate-go-coverage-top-tasks.sh` 开发辅助脚本，可对真实 Go 项目的前 N 个 coverage task 做隔离验证，并输出 JSONL 结果和 summary。

### Changed

- `validate_coverage_task` 会将疑似不可达的 skipped coverage task 标记为 `action: "manual_review_unreachable"`，并在 metadata 中返回 `unreachable` 与 `unreachable_reason`，避免 Agent 把不可达分支当普通 TODO 反复重试。
- `validate_coverage_task` 会将系统资源错误分支这类依赖运行环境且无法静态构造的 skipped task 标记为 `action: "manual_review_environment"`，并在 metadata 中返回 `environment_dependent` 与 `environment_reason`。
- Go coverage task static generator 在遇到无参数、非方法、返回值可安全丢弃但无法推导精确期望值的函数时，会生成可执行的 smoke 测试而不是默认 skipped TODO；真实样例验证覆盖 `GetNowDate()` 这类日期/时间辅助函数。
- Go coverprofile 解析会把当前 `go.mod` module 路径映射成本地源码路径，例如 `car-svc/utils/time.go` 会归一化为 `utils/time.go`，让 `parse_coverage.test_tasks[]` 可直接传给 `generate_tests`。
- Go `generate_tests` 写入已有测试文件时会合并追加新的 `Test*` 函数并复用 import，不再覆盖已有测试；普通 Go 合并遇到同名测试函数会返回明确错误。
- Go coverage task 写入已有测试文件时，如果任务推荐的 `test_name` 已存在，会基于覆盖率行段或 task id 自动追加稳定后缀，例如 `TestGetRawCoverage204_207`，避免重复任务卡在生成阶段。
- Go `run_tests` 使用相对测试文件或目录时会归一化为 `./pkg` 形式，避免 `utils/time_test.go` 被执行成标准库导入路径 `utils`。
- Go static generator 会识别 `time.Now().Format("layout")` 这类日期字符串返回值，生成 `time.Parse` 格式断言，不再退化成仅丢弃返回值的 smoke 测试。
- Go static generator 会识别 `time.Date(..., 0, 0, 0, 0, ...)` 这类 `time.Time` 日期边界返回值，生成 hour/min/sec/nsec 归零断言。
- Go static generator 会利用 coverage task 的简单分支条件提示，例如 `a == 0` / `x > 3`，为可推导返回值的分支生成非 skipped 用例和精确期望值。
- Go static generator 的分支输入推导扩展到字符串空值、布尔值以及 nil / 非 nil 指针；当源码参数名为 `name` / `skip` 时会避让测试表保留字段，避免生成重复字段。
- Go static generator 支持 `err == nil` / `err != nil` 分支输入；非 nil error 会生成 `errors.New("test")` 并自动加入 `errors` import。
- `run_tests` 的 Go 执行路径会在收到 module 内绝对目录或绝对测试文件时自动切到 `go.mod` 根目录，并转换为相对包路径，避免 `directory ... outside main module` 失败。
- Go coverage task 在无法安全生成精确断言时，会在 TODO case 和 `context.targets[].payload_notes` 中说明保守降级原因，并为 Go context 暴露参数、返回表达式和分支条件。
- Go context 会保留 `a > 0 && b > 0` / `a > 0 || b > 0` 这类复合分支条件原文，并在 coverage task 降级说明中标注当前不支持多参数输入合成。
- Go static generator 支持有限的 `&&` 复合条件输入合成；当每个子条件都是简单参数边界且返回表达式安全时，会生成非 skipped 精确用例。
- Go static generator 支持简单整数范围条件，例如 `a > 0 && a < 10` 会合成范围内输入；无交集或非整数重复参数条件继续保守降级。
- Go static generator 支持 URL/API 字符串参数触发的 `err != nil` 分支；对 `error` 或 `(..., error)` 返回值会生成非法 URL 输入、断言 error 非 nil，并对非 error 返回值做 nil/简单值断言。
- Go static generator 支持 `*http.Request` 字符串返回分支的常见输入合成，可为 `RemoteAddr`、`X-Forwarded-For`、`X-Real-IP` 和 RemoteAddr 解析错误生成可执行请求对象与精确断言。
- Go static generator 支持常见 JSON/error 分支输入合成：`AsJson` marshal error、`FromJson` 非法 JSON、`FromJsonFile` 缺失文件路径会生成可执行断言。
- Go static generator 支持 `FromJsonFile` 成功返回路径输入合成，会写入临时 JSON 文件并断言返回 error 为 nil。
- Go static generator 支持部分工具函数分支输入合成：`SliceMapper0` 去重分支、`UserDurationOf` switch/case 和 `TrimSpaceSlice` 非空分支会生成可执行断言。
- Go static generator 扩展工具函数 return/statement path 输入合成，覆盖 `SliceMapper0`、`TrimSpaceSlice` 和 `UserTypeOf` 的纯函数返回路径。
- Go static generator 支持 `ParseToken` JWT 成功分支输入合成，可用同包 `GenerateToken` 与 `global.Config.Jwt` 构造有效 token，并断言 claims 非 nil、error 为 nil。
- Go static generator 支持 `Recover` 的 panic/recover 分支输入合成，会用 `defer Recover(...); panic(...)` 覆盖 `recover() != nil` 路径。
- Go static generator 支持 `GetJson` / `GetBytes` 这类 HTTP wrapper 的本地 `httptest` 输入合成，可覆盖 JSON 解析错误路径和 body 成功返回路径。
- Go static generator 支持 `TraceTransport.RoundTrip` 慢请求分支输入合成，可用本地 `httptest.Server` 和负 `SlowThreshold` 稳定覆盖 defer 中的 slow branch。
- Go static generator 支持 `Ptr` 这类泛型指针返回路径断言，会检查返回指针非 nil 且 `*got` 等于输入值，避免把指针地址当作期望值比较。
- Go static generator 支持 `RemoteIP` 的剩余 return/statement path：可临时覆盖同包 `ipLookups` 触发 fallback 返回路径，并用 RemoteAddr 输入覆盖入口语句块。
- Go static generator 支持 `BeforeSave(*gorm.DB) error` 这类 receiver mutation 方法的字段归一化/默认值断言，可为 laoxia 模型的 `User`、`Role`、`Menu`、`DictItem` 等方法生成非 skipped 测试。
- Go static generator 会保留函数类型参数的完整签名，例如 `func(int) int` 不再退化为 `func()`。
- Go static generator 会根据源码参数/返回类型中的 selector 自动补测试文件 import，例如 `*http.Request` 会引入 `net/http`。
- Go static generator 对未知命名类型的零值改用 `*new(Type)`，避免 `time.Duration{}` 这类命名标量类型导致生成测试编译失败。
- Go static generator 在方法测试中会避让 `t` / `tt` 等测试模板保留名，避免源码 receiver 名与 `*testing.T` 参数冲突导致生成测试无法编译。
- Go `init` coverage task 会生成明确的人工复核 skip，不再直接写出不可调用的 `init()` 调用；`validate_coverage_task` 会将这类结果标记为 `manual_review_unreachable`。
- Go coverage task 的分支缺口改为基于 AST 抽取 `if` / `switch` / `return`，不再把函数签名、普通语句或 `if init` 误当作分支条件。
- Coverage suggestion/test task 会合并同目标、同缺口类型、同分支条件且行段相邻或重叠的未覆盖 block，减少 Go coverprofile 拆块导致的重复任务。
- Coverage task 排序新增路径环境成本启发式，优先暴露 `utils` / helper / parser 等低依赖任务，并降低 controller、router、service、middleware、db/cache 等高初始化成本任务的优先级。

### Fixed

- 修复真实 Go 项目中已有测试函数与 coverage task 推荐 `test_name` 重名时 `validate_coverage_task` 返回 `generation_error` 的问题；laoxia `GetRaw` 样本已验证为 `passed/ready`。
- 修复 laoxia top50 扩窗验证中 `TraceTransport.RoundTrip` 因 receiver 名为 `t` 造成的编译失败；同轮验证中 `init` 任务改为人工复核后，top50 达到 50/50 `passed`。

## v0.4.13 - 2026-07-10

### Added

- JS/TS `payload_notes` 在遇到 imported type 时会追加 import 来源和候选源码文件提示，帮助 Agent/LLM provider 读取跨文件类型上下文，而不是误把保守 mock 当作完整 DTO。
- `examples/llm-provider.sh` 支持读取 `payload_notes` 中的候选源码文件并组装调试 prompt，可通过 `TESTLOOP_LLM_PROVIDER_MODEL_CMD` 接入真实模型命令。
- 外部 LLM provider 输出会清洗常见 Markdown 代码围栏和前后解释性文本；如果输出不含可识别测试代码，会返回明确错误。
- 外部 LLM provider 输出增加按目标语言的轻量测试代码校验，Go/Python/JS/TS/Rust/Java 会拒绝明显不是测试的代码片段。
- 新增 `examples/llm-provider-prompt.md`、Ollama 模型命令包装和 OpenAI CLI 模型命令包装，降低外部 LLM provider 的真实模型接入成本。
- 新增 LLM provider 生成结果进入 `run_tests include_fix_suggestions=true` 的 handler 回归测试，固定外部生成结果可进入失败解析和 repair task 闭环。
- `cmd/testgen` 新增 `-provider-check`，用于诊断 provider 模式、`TESTLOOP_LLM_PROVIDER_CMD` 和命令可执行性。
- MCP `generate_tests` 的 LLM provider 失败错误新增 `provider_error kind=... action=...` 分类，方便 Agent 区分配置、命令执行、输出格式和语言校验问题。
- Agent workflow 新增 LLM provider 错误策略表，明确哪些错误应重试模型、降级 static，或提示用户修 provider 配置。
- 默认 LLM provider prompt 新增输出契约，要求模型只返回可直接写盘的完整测试文件，无法安全增强时回退静态草稿。
- 新增 MCP handler 层的 LLM provider 坏输出回归测试，固定空输出、JSON 错误、缺少 `code`、解释文本和非测试代码的 `provider_error kind/action`。
- 新增结构化 `provider_error` 自动降级 static 并继续 `run_tests` 的 handler 闭环测试，固定 Agent fallback 序列可执行。

### Changed

- MCP `generate_tests` 的 LLM provider 失败会返回 `isError=true` 的结构化工具结果，并在 JSON / `structuredContent` 中提供 `provider_error.kind`、`provider_error.action`、`provider_error.provider` 和 `provider_error.message`；旧的 `provider_error kind=... action=...` 文本片段继续保留在 `error` 字段中。
- `scripts/install.sh` 的 `go install` fallback 日志会根据实际落盘文件名输出安装路径，避免跨平台 dry run 下载失败时把当前主机二进制误报为 `.exe`。

## v0.4.12 - 2026-07-09

### Added

- JS/TS payload 支持同文件简单泛型 alias/interface 的直接实例化，例如 `ApiEnvelope<User>`，会在可解释范围内展开为结构化 mock 数据。
- `generate_tests.context.targets[]` 新增 JS/TS `return_type_expr` 和 `payload_notes`，在跨文件类型或复杂泛型导致静态 payload 回退时给 Agent/LLM provider 明确原因。
- 新增 `generate_tests` handler 回归测试，固定 JS/TS `payload_notes` 会出现在 MCP 工具输出 JSON 中。
- 新增外部 LLM provider 请求回归测试，固定 JS/TS `payload_notes` 会随 stdin JSON 传给 provider。

### Changed

- `scripts/install.sh` 的 `go install` fallback 会区分不支持的平台、latest 解析失败、Release 资产下载失败和缺少解压器，避免把网络失败误报成没有匹配资产。

## v0.4.11 - 2026-07-09

### Added

- JS/TS 静态生成器补强 TypeScript DTO payload，覆盖 utility wrapper、Pick/Omit、Record、对象交叉、indexed access、数组和 tuple 组合。
- JS/TS 对象字段内部的数组、tuple、Record、投影类型和组合 alias 会继续生成结构化 payload。
- 新增 JS/TS 复杂 payload 的 `generate_tests -> run_tests` handler 闭环检查，覆盖普通生成和 coverage task 两条路径。
- 新增 `docs/js-ts-payload-quality.md`，记录 JS/TS payload 支持范围、保守回退和不支持边界。

## v0.4.10 - 2026-07-07

### Added

- `fix_suggestions` 每条建议新增 `repair_task`，聚合失败分类、目标位置、上下文片段、可编辑文件、建议复跑命令和断言关注点，便于 Agent 直接执行单个修复任务。
- `run_tests` 新增 `include_fix_suggestions`、`source_code` 和 `test_code` 输入，测试失败时可内联 `fix_suggestions[]` 和 `repair_task` 摘要。
- 新增 repair task golden test，固定面向 Agent 的修复任务 JSON 契约。

## v0.4.9 - 2026-07-07

### Added

- `fix_suggestions` 返回新增 `category`、`context_file` 和 `context_line`，便于 Agent 区分失败类型并定位源码或测试上下文。
- `--check-config` 和 `--doctor-config` 在配置异常时会输出可执行的修复建议，降低 MCP 客户端接入排查成本。

### Changed

- `fix_suggestions` 的建议文本补充 actual/want、越界 index/length、panic 类型和源码/测试行上下文，并支持相对路径匹配测试文件。
- Agent 闭环文档补充失败修复步骤，明确先用 `fix_suggestions` 收敛真实失败，再进入覆盖率任务生成。

## v0.4.8 - 2026-07-06

### Added

- 主二进制新增 `--print-config`，可输出 Codex、Codex HTTP、Claude Code / Claude Desktop 和 Cursor 的 MCP 配置片段。
- 主二进制新增 `--check-config`，可读取配置文件或 stdin，检查 MCP server 的 `command` 是否存在且可执行，或 `url` 是否是合法 HTTP endpoint。
- 主二进制新增 `--doctor-config`，可输出推荐配置路径、只读校验已存在的 Codex、Claude 和 Cursor 配置，并区分缺少 `testloop` server 与其他 MCP server 正常配置。
- 新增 `docs/agent-workflow.md`，展示 `run_tests -> parse_results -> parse_coverage -> generate_tests -> run_tests` 的 Agent 闭环顺序。
- 新增 `scripts/generate-client-config.sh`，作为源码仓库里的配置片段生成辅助入口。

## v0.4.7 - 2026-07-06

### Changed

- MCP server implementation version 更新为 `0.4.7`。
- Release Artifacts workflow 新增 `windows_arm64` matrix 项，使用 `windows-11-arm` runner、MSYS2 `CLANGARM64` 和 `mingw-w64-clang-aarch64-clang` 构建 Windows ARM64 zip。
- Windows release 资产上传前会校验 `.sha256`、检查 zip 内容，并实际运行 `testloop-mcp.exe --help` 和 `testloop-testgen.exe --help`。
- README、安装文档和发布维护记录同步到 `v0.4.7`。

## v0.4.6 - 2026-07-06

### Changed

- MCP server implementation version 更新为 `0.4.6`。
- 将 `v0.4.5` 发布后验证通过的 Homebrew formula `--help` 测试修复纳入正式 release source archive。
- README、安装文档和发布维护记录同步到 `v0.4.6`。

## v0.4.5 - 2026-07-06

### Changed

- MCP server implementation version 更新为 `0.4.5`。
- 内置静态测试生成器补充覆盖 Go、Python、Jest、Java 和 Rust 的 coverage-task、parser 和 helper 分支测试，降低 coverage task 草稿生成回归风险。
- `internal/generator` 本地语句覆盖率提升到 `91.7%`，覆盖 release 前最容易回归的目标过滤、参数推断、边界输入和 parser 分支。
- Release Artifacts workflow 会在上传前校验生成资产的 `.sha256`，并检查 tarball/zip 内包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。

## v0.4.4 - 2026-07-06

### Changed

- Release 资产打包逻辑抽到 `scripts/package-release-asset.sh`，workflow 复用同一脚本生成 tarball/zip 和 `.sha256`。
- Release Artifacts workflow 新增 `windows_amd64` matrix 项，从该版本起 tag release 会上传 Windows zip 和 `.sha256`。
- 安装脚本支持在 Git Bash/MSYS/Cygwin 等 Windows shell 下下载、校验并安装 `windows_amd64` zip 资产，缺少匹配资产或解压工具时仍回退到 `go install`。
- 移除临时 Windows Release Probe workflow；Windows 打包链路已合入正式 Release Artifacts matrix。
- MCP server implementation version 更新为 `0.4.4`。

## v0.4.3 - 2026-07-06

### Changed

- Release Artifacts workflow 改为由每个 matrix build job 直接上传对应 tarball 和 `.sha256`，避免单独 publish job 等不到 runner 时阻塞发版。
- 安装脚本兼容聚合 `checksums.txt` 和单资产 `.sha256` 两种校验文件。
- 新增 Homebrew Formula 草案、生成脚本和独立 Homebrew Tap workflow，用于按 tag 更新 `sleticalboy/homebrew-tap` PR，避免阻塞 release 资产发布。
- README 和安装文档新增 `brew tap sleticalboy/tap && brew install testloop-mcp` 安装路径。
- MCP server implementation version 更新为 `0.4.3`。

## v0.4.2 - 2026-07-05

### Added

- Release Artifacts workflow 准备生成 Linux amd64、Linux arm64 和 macOS arm64 三类 tarball，并统一生成 `checksums.txt`。
- 新增 `scripts/install.sh`，支持检测平台、下载 release 资产、校验 checksum、安装 `testloop-mcp` / `testloop-testgen`，资产缺失时回退到 `go install`。

## v0.4.1 - 2026-07-05

### Added

- 新增 `docs/installation.md`，补齐 Release 下载、checksum 校验、源码构建、Docker 运行和 Codex / Claude / Cursor 接入说明。
- 新增 MIT `LICENSE` 文件。

### Changed

- Go module path 和文档仓库地址统一为 `github.com/sleticalboy/testloop-mcp`，为后续新版本支持 `go install github.com/sleticalboy/testloop-mcp@latest` 做准备。

## v0.4.0 - 2026-07-05

### Added

- Rust `cargo tarpaulin` LCOV 覆盖率建议会尝试把未覆盖行映射到具体 `fn`，并在 `test_tasks` 中使用函数目标。
- Java JaCoCo 覆盖率建议会尝试把未覆盖行映射到具体类方法，并支持常见 `src/main/java` 源码目录解析。
- Rust/Java 覆盖率建议会对 `if`、`match`、`switch`、错误/空值返回和普通返回做轻量语义分类，生成更具体的 `gap_type`、`missing_branches` 和输入提示。
- Java 覆盖率源码映射改用 tree-sitter，支持注解、多行方法签名、构造函数和内部类，并保留轻量正则回退。
- Rust 覆盖率源码映射改用 tree-sitter，支持属性标注函数、多行函数签名、`impl` 方法和 trait 默认方法，并保留轻量正则回退。
- 新增 Rust workspace 和 Java Maven 风格覆盖率 fixture，验证相对报告路径、复杂源码目录和源码映射不会退化。
- `test_tasks` 新增 `test_file`、`test_name` 和 `assertion_focus`，让 AI Agent 更容易把覆盖率缺口转成具体测试草稿。
- `test_tasks` 新增 `priority` 和 `priority_reason`，并按函数/方法级缺口、分支/错误路径、建议输入、未覆盖行和置信度排序。
- `generate_tests` 支持接收单个 `coverage_task`，并把任务上下文传给 LLM provider、回写到返回 context，同时优先写入任务推荐的 `test_file`。
- Go/Rust/Java coverage task 输出新增 JSON golden 快照测试，固定面向 Agent 的任务契约。
- Go 静态生成器支持 `coverage_task` 模式，会优先只生成目标函数或方法的测试，并把 task 信息写入测试名、case 名和注释。
- Python/Jest 静态生成器支持 `coverage_task` 模式，会按目标过滤测试草稿，并把建议输入转成更具体的调用参数和断言。
- Rust/Java 静态生成器支持 `coverage_task` 模式，会优先生成目标函数或方法的测试骨架，减少整文件泛化输出。
- 新增 Go/Python/Jest/Rust/Java task-aware 静态生成 golden tests，防止 coverage task 增量测试草稿退化。
- 补齐 v0.4.0 发布说明草案，并同步 README、LLM provider 文档和质量评估中的 coverage task 闭环说明。

## v0.3.0 - 2026-07-05

### Added

- Python/Jest 生成器会对简单 return 表达式生成精确断言，例如 `a + b` 会生成 `assert result == (1 + 2)` / `expect(result).toBe((1 + 2))`。
- 边界用例会把边界值带入简单 return 表达式，生成更具体的断言。
- Go 内置生成器会为简单纯函数生成可执行表驱动 case，不再默认只生成 TODO/skip。
- Python/Jest 生成器会识别简单 if-return 分支，为普通路径和边界路径分别生成期望值。
- Go/Python/Jest 生成器新增 golden tests，固定代表性输出。

## v0.2.0 - 2026-07-05

### Added

- `parse_coverage` 支持 Rust `cargo tarpaulin --out Lcov` 生成的 LCOV。
- `parse_coverage` 支持 Java JaCoCo XML。
- Rust/Java 覆盖率报告会生成统一的 `CoverageReport`、`suggestions` 和 `test_tasks`。
- `run_tests coverage=true` 支持为 Rust 调用 tarpaulin、为 Java Maven/Gradle 调用 JaCoCo report，并回填 `coverage_percent`。
- Rust/Java 覆盖率闭环新增 e2e 测试，覆盖 `run_tests` 与 `parse_coverage` 联动。

## v0.1.0 - 2026-07-04

首个可用版本，定位为面向 AI Coding Agent 的测试反馈与质量控制 MCP 层。

### Added

- MCP server 支持 stdio 和 Streamable HTTP 两种传输模式。
- `run_tests` 支持 Go、Rust、Jest、Vitest、Mocha、pytest、JUnit 5 的测试执行与自动检测。
- `parse_results` 支持 Go、Rust、Jest、Vitest、Mocha、pytest、JUnit 5 的结构化失败解析。
- `generate_tests` 支持 Go、Rust、Java、JavaScript/TypeScript、Python 测试生成。
- Go 测试生成优先调用 `gotests -all`，失败时回退内置 `go/ast` 生成器。
- JS/TS/Python 生成器支持参数名语义默认值、边界输入、异常路径和基础返回类型断言。
- 可选 LLM provider：`provider: "llm"` / `provider: "auto"`，通过 `TESTLOOP_LLM_PROVIDER_CMD` 接入外部命令。
- `parse_coverage` 支持 Go coverprofile、Istanbul coverage JSON、coverage.py JSON。
- Go 覆盖率缺口可映射到函数/方法，并生成面向 AI Agent 的 `test_tasks`。
- `fix_suggestions` 返回结构化修复建议。
- 独立 CLI：`cmd/testgen`，支持 `-provider static|llm|auto`。
- Docker 镜像和 `docker-compose.yml`，HTTP 模式提供 `/healthz` 健康检查。
- GitHub Actions CI：测试、主服务构建、CLI 构建、Docker build。

### Fixed

- 修正低价值零值测试生成策略：无法推断有效输入时标记 TODO/skip。
- 修正 JS/Python 生成器中异常边界输入仍按正常返回值断言的问题。
- 修正 Docker healthcheck 访问 `/mcp` 无 session 返回 400 的问题。
- 修正 Alpine 运行时镜像安装不存在的 `musl-libc` 包的问题。
- 修正 `.gitignore` 误伤 `cmd/testgen/main.go` 的问题。

### Known Limitations at Release

- Rust `cargo tarpaulin` 覆盖率解析在 v0.1.0 发布时尚未实现。
- Java JaCoCo 覆盖率解析在 v0.1.0 发布时尚未实现。
- LLM provider 当前是命令协议适配层，不内置具体模型厂商。
- 静态测试生成仍以可运行骨架和上下文增强为主，不承诺替代通用 AI Agent 的完整语义测试生成。

### Verification

- `go test ./...`
- `go build -o /tmp/testloop-mcp .`
- `go build -o /tmp/testloop-testgen ./cmd/testgen`
- `docker build -t testloop-mcp:release-check .`
- Docker container `/healthz` smoke test
- GitHub Actions CI passed
