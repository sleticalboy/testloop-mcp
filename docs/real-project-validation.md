# 真实项目验证质量报告

这份报告汇总当前已经跑过的真实项目样本。它不是性能 benchmark，也不等同于所有项目的承诺结果；它的作用是给后续开发一个质量边界：哪些闭环能力已经在真实仓库中验证过，哪些场景应该明确进入手审、依赖注入或集成环境，而不是继续让静态生成器硬猜。

## 样本指标

| 序号 | 样本 | 语言 / 框架 | 验证窗口 | 最新结果 | 普通 repair |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | laoxia server | Go / go test | top50 | `passed=50`，`ready=45`，`manual_review_unreachable=2`，`manual_review_environment=3` | 0 |
| 2 | lazy-mcp-wrapper | Go / go test | top20 | `passed=20`，`ready=16`，`manual_review_protocol=4` | 0 |
| 3 | QuickSmoke Backend | Go / go test | top20 | `passed=20`，`ready=14`，`manual_review_database=6` | 0 |
| 4 | mcp-hub | JavaScript / Vitest | top50 | `passed=50`，`ready=50`，`skipped_total=0` | 0 |
| 5 | ip2region JS binding | JavaScript / Jest | top30 | `passed=30`，`ready=30`，`skipped_total=0` | 0 |
| 6 | Codex SDK TypeScript | TypeScript / Jest | cross-file top101 | `passed=101`，`ready=83`，`manual_review_internal=18` | 0 |
| 7 | RocketMQ Node.js client | TypeScript / Mocha | top100 | `passed=97`，`failed=3`，`ready=64`，`manual_review_private=33`，`manual_review_external_service=3` | 0 |
| 8 | unjs/ufo | TypeScript / Vitest | top8 | `passed=8`，`ready=6`，`manual_review_internal=1`，`manual_review_no_runtime=1` | 0 |
| 9 | pallets/click | Python / pytest | top20 | `passed=20`，`ready=19`，`manual_review_environment=1` | 0 |
| 10 | Codex SDK Python | Python / pytest | top30 | `passed=30`，`ready=30`，`skipped_total=0` | 0 |
| 11 | Starlette | Python / pytest | top50 | `passed=50`，`ready=50`，`skipped_total=0` | 0 |
| 12 | haoy-apk-station backend | Python / FastAPI + pytest | top180 | `passed=180`，`ready=177`，`manual_review_environment=3` | 0 |

`普通 repair` 指最新验证结果中仍需要修生成测试本身的 `repair_generated_test` / `apply_fix_suggestions` / `generation_error` 数量。`manual_review_*` 不计入普通 repair，它表示工具已经给 Agent 一个稳定动作分类：不要继续盲修同一份生成测试，应改走公共入口、依赖注入、集成环境或人工复核。

## 已稳定的能力

Go 链路已经能覆盖低依赖 utility、HTTP wrapper、JSON/file helper、panic/recover、JWT、泛型指针、nil receiver、Gin response、全局 logger 初始化、Unix socket 协议读取和多返回值 error branch。`validate_coverage_task` 也能把不可达分支、OS/runtime 错误分支、socket write / streaming I/O 时序分支和数据库分支从普通 ready 队列里分离。

JS/TS 链路已经验证过 Vitest、Jest、Mocha、ESM、TypeScript strict 编译、`ts-jest`、项目自定义 runner、测试目录 include 规则、monorepo 外部资源 symlink、默认导出实例、JS `#private`、TS `private/protected`、getter、未导出内部 helper 的公共入口覆盖、跨模块 interface/type/class/enum mock、Map/Set、函数类型字段、fake timers、动态 import 和 Node process/platform 全局状态。

Python/pytest 链路已经从基础 pytest 草稿推进到四个真实项目样本：能处理 coverage report 相对路径映射、dotted package import、仓库根包布局、单行安全 task 注释、class `__init__` 元数据、constructor 入参、常见 fallback/error path 输入、wrapper swallow exception、Unicode fallback、stream fallback、内部状态对象构造、Pydantic 通知 payload、keyword-only 参数、dataclass 输入类型、同步/异步 stream finish 状态、ASGI scope、认证 scope、配置文件/cast 场景、可变多值字典状态断言、文件上传临时流、HTTP header 追加、ASGI endpoint 最小 scope / receive / send、WebSocket decode / dispatch、multipart parser 状态机、无 pyproject 的 `tests/...` 项目根推断、FastAPI TestClient 业务流、SQLAlchemy SQLite 临时库、API Key 上传路径、zip/apk 文件输入、外部 APK parser SDK fake、下载 URL helper、版本详情、下载重定向 fallback、删除图标、隐藏应用、下载统计、短链页面渲染、短链当前版本查询和最终 HTMLResponse 返回路径、auth refresh helper、JWT refresh/access token 分支、API Key request fake、auth service API Key fake DB、认证/API Key 列表 fake DB、应用列表搜索 fake DB、`build_app_out` fallback 版本查询、QR import fallback、数据库迁移 fake Session/inspect、Qiniu 对象存储 helper fallback、storage backend 选择和 TOS 对象存储 helper fallback，以及动态前端入口和根静态文件入口的环境手审分类。

## 仍需手审的边界

`manual_review_unreachable`：源码条件本身难以到达，例如由非空 slice 派生出的负索引分支，或 Go `init` 这类不能直接调用的入口。

`manual_review_environment`：依赖 OS/runtime 或当前进程环境的错误路径，例如磁盘/CPU/内存采集库返回错误、Python 标准流 binary wrapper 状态。

`manual_review_protocol`：依赖 socket write、streaming I/O 或协议时序的分支。除非源码提供 fake connection 注入点，否则不应靠竞态触发。

`manual_review_database`：依赖真实 DB、GORM 行为、事务或 RowsAffected 的分支。项目没有 sqlite/sqlmock/测试库策略时，不应自动引入第三方依赖。

`manual_review_internal` / `manual_review_private`：外部测试模块无法稳定访问的内部 helper、未导出 class/function 或语言级 private method。优先走公共入口覆盖；没有公共入口时应手审或重构可测性。

`manual_review_external_service`：依赖 live RPC、gRPC endpoint、路由状态或长重试时序的路径。这类结果可能表现为 `failed/manual_review_external_service`，含义不是生成器失败，而是需要 fake client、依赖注入或集成环境。

`manual_review_no_runtime`：TypeScript type-only、barrel re-export 或无本地 runtime statement 的文件。应通过消费方行为测试或类型检查验证，不应伪造单元测试。

## 当前判断

项目当前的核心价值已经不再是“能生成测试文件”，而是能在真实仓库里把覆盖率缺口拆成可执行测试、明确手审项和环境依赖项，并把普通生成失败持续压到 0。后续新增语言或框架前，应优先要求它们能进入同样的验证闭环：可重复 baseline、隔离 task worktree、结构化 action、普通 repair 清零或可解释归类。

下一步更有价值的是继续扩大业务型 Python 服务样本到 top200，重点观察上传后成功/失败回滚、更多应用详情更新路径、删除/移动文件后的数据库一致性，以及统计聚合的更深 SQLAlchemy 链是否能稳定归类为 ready 或 `manual_review_external_service` / `manual_review_database`。
