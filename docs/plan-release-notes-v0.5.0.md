# v0.5.0 发布说明草案

## 标题

testloop-mcp v0.5.0

## 发布状态

- [x] 创建 v0.5.0 发布说明草案。
- [x] 梳理当前固定 smoke 矩阵和真实项目验证证据。
- [x] 在 README 补充面向 AI Agent 的快速演示路径。
- [x] 正式版本准备时更新 `main.go` MCP implementation version 到 `0.5.0`。
- [x] 正式版本准备时将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.5.0 - 2026-07-17`。
- [x] 正式版本准备时更新 README、安装文档和必要的版本引用。
- [x] 正式发布前重新跑完整本地验证、远端 CI、Release Artifacts、资产校验和 Homebrew 安装链路验证。

## 摘要

v0.5.0 候选重点不是继续扩大“能生成测试”的语言清单，而是把 testloop-mcp 收敛到更清晰的产品定位：面向 AI Coding Agent 的测试反馈闭环 MCP 服务。

这个版本把近几轮 Java、JS、Python 的真实项目样本固化为固定 smoke 矩阵，核心目标是证明 Agent 可以稳定获得以下反馈：

- 哪些覆盖率缺口可以直接生成并运行测试。
- 哪些测试虽然能运行，但没有命中目标行，应退回更好输入或人工复核。
- 哪些路径依赖内部实现、运行环境、外部服务或数据库事务，不应伪装成普通 ready 测试。
- 生成测试、运行测试、解析输出、分类下一步动作可以被低成本重复验证。

## 主要变化

### 固定 smoke 矩阵

- 新增 `scripts/validate-regression-smoke.sh`，串联 Java + JS + Python 代表性样本。
- 新增 `scripts/validate-java-regression-samples.sh`，固定 Java 真 ready、历史假 ready 降级和内部手审样本。
- 新增 `scripts/validate-js-regression-samples.sh`，固定 Jest/Vitest ready、TypeScript no-runtime、未导出 helper、mcp-hub 生命周期和环境依赖样本。
- 新增 `scripts/validate-py-regression-samples.sh`，固定 Click pytest ready、name-mangled private method、FastAPI 动态前端入口、外部服务 timeout 和 SQLAlchemy 事务错误样本。
- 新增 `scripts/fixture-task-jsonl.py`，统一生成 JS/Python fixture coverage task JSONL，避免 regression 脚本继续内联重复 JSON。

### Java/JUnit 质量收口

- Java `validate_coverage_task` 默认运行生成测试时收集 JaCoCo report，并校验 `coverage_task.line_range` 是否被真实命中。
- 测试通过但目标行未覆盖时，会返回 `failed/needs_better_input`，避免把弱 ready 暴露给 Agent。
- Java 验证脚本支持 `TESTLOOP_VALIDATE_JAVA_TASK_IDS` 和 `TESTLOOP_VALIDATE_JAVA_TASKS_FILE`，可以按 task id 精确回归并复用历史 JSONL。
- `run_tests` 对 Java 测试文件路径会尽量只运行生成的 `*TestLoopTest` 测试类，同时保留 JaCoCo report 生成。
- 固定样本覆盖 Commons Lang 和 Commons Codec 的 ready、`manual_review_unreachable`、`manual_review_internal` 三类路径。

### JS/TS 真实项目样本

- mcp-hub Vitest 样本覆盖 `ConfigManager.loadConfig` async throwing branch，防止历史 `repair_generated_test` 回退。
- mcp-hub `EnvResolver` 样本覆盖 placeholder 替换和缺失环境变量分支。
- mcp-hub `WorkspaceCacheManager` 样本覆盖 cache 状态更新、stale cleanup，以及真实 lock 文件/进程探测路径的 `manual_review_environment` 分类。
- mcp-hub `SSEManager` 样本覆盖自动关闭、连接断开、发送失败和定向发送。
- mcp-hub `DevWatcher` 样本覆盖 stop cleanup 和 watcher error lifecycle。
- 仓库内 fixture 固定 TypeScript 纯类型文件的 `manual_review_no_runtime` 和未导出 ESM helper 的 `manual_review_internal`。

### Python 真实项目样本

- Click pytest 样本固定真实 ready 路径。
- 仓库内 fixture 固定 name-mangled private method 的 `manual_review_internal`。
- haoy-apk-station FastAPI 样本固定动态前端入口依赖 `frontend/dist` 的 `manual_review_environment`。
- haoy-apk-station 下载代理样本固定对象存储 endpoint timeout 的 `manual_review_external_service`。
- haoy-apk-station 删除应用样本固定 SQLAlchemy `db.commit()` 事务错误的 `manual_review_database`。

### 文档和定位

- `docs/product-positioning.md` 明确项目定位：testloop-mcp 是 AI Coding Agent 的测试反馈与质量控制 MCP 层，不是独立测试生成器。
- `docs/regression-smoke.md` 记录固定 smoke 的默认项目路径、JSONL 依赖、跳过开关、runner 约束和当前质量边界。
- README 增加面向 Agent 的快速演示路径，强调 `generate_tests -> run_tests -> parse/fix/coverage feedback` 闭环，而不是单次生成测试。

## 真实项目验证证据

当前固定 smoke 覆盖三类样本来源：

- 成熟开源库：Apache Commons Lang、Apache Commons Codec、Click、ip2region JavaScript binding。
- 真实业务/工具项目：mcp-hub、haoy-apk-station。
- 仓库内最小 fixture：JS no-runtime、JS internal、Python internal。

这些样本的目标不是追求大覆盖率数字，而是固定容易让 Agent 跑偏的代表性路径：

- 生成测试通过但不命中目标行。
- 直接访问 private/internal 实现。
- 类型文件或 barrel re-export 没有运行时代码。
- 真实文件锁、进程探测、动态静态资源、外部对象存储和数据库事务无法靠普通静态单测安全构造。
- 长连接和 watcher 生命周期测试容易因为 mock 过度或启动真实资源而失稳。

## 质量边界

v0.5.0 候选仍保持以下边界：

- 固定 smoke 不是完整 top-N 验证，也不是性能 benchmark。
- Rust 和 Java 覆盖率解析已支持，但当前固定 smoke 主线只覆盖 Java、JS、Python；Rust 真实项目固定 smoke 仍未纳入默认矩阵。
- JS/Python 仓库内 fixture 只用于稳定验证手审分类，不代表真实业务项目质量。
- 外部服务、数据库、运行环境依赖路径会优先分类为手审或环境任务，不伪装成 ready 测试。
- 生成测试质量仍依赖源码结构、coverage task 粒度和可构造输入；项目重点是给 Agent 稳定反馈和动作分类。

## 验证

当前候选草案已经完成的本地验证：

- [x] `go test ./...`
- [x] `git diff --check`
- [x] `TESTLOOP_VALIDATE_JS_STAGE_TIMEOUT_SECONDS=180 TESTLOOP_VALIDATE_PY_STAGE_TIMEOUT_SECONDS=180 scripts/validate-regression-smoke.sh`
- [x] `go build -o /tmp/testloop-mcp-v0.5.0-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.0-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.0-prep --help` 输出 usage；Go `flag` 包对 help 返回 exit code 2。
- [x] `/tmp/testloop-testgen-v0.5.0-prep --help` 输出 usage；Go `flag` 包对 help 返回 exit code 2。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.0-prep scripts/package-release-asset.sh v0.5.0 darwin_arm64 darwin arm64`
- [x] 校验 `/tmp/testloop-v0.5.0-prep/testloop-mcp_v0.5.0_darwin_arm64.tar.gz.sha256`
- [x] 推送后等待远端 CI 通过：run `29557865650` passed。
- [x] 正式发布时验证 Release Artifacts 和 Homebrew 安装链路：Release Artifacts run `29558114233` passed，Post-Release Verify run `29559912737` passed，Homebrew tap commit `e201f8f` 已验证。

## 发布备注

- v0.5.0 适合作为“固定 smoke + Agent 闭环定位”的版本节点。
- 发布文案应突出测试反馈基础设施，而不是多语言测试生成器。
- README、release note 和 roadmap 中不应把 Rust 真实项目 smoke 描述成已完成。
- 如果正式发布前继续新增样本，应优先补到 `docs/regression-smoke.md` 和本文件，而不是只留在脚本注释中。
