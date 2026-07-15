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
| 12 | haoy-apk-station backend | Python / FastAPI + pytest | expanded4 top168 | `passed=168`，`ready=165`，`manual_review_environment=3` | 0 |
| 13 | ip2region Python binding | Python / pytest | top9 | `passed=9`，`ready=9`，`skipped_total=0` | 0 |
| 14 | RocketMQ Java client | Java / Maven + JUnit + JaCoCo | `route/Endpoints.java` top20 + `exception/StatusChecker.java` top2 | `Endpoints: passed=20/ready=20`，`StatusChecker: passed=2/ready=2` | 0 |

`普通 repair` 指最新验证结果中仍需要修生成测试本身的 `repair_generated_test` / `apply_fix_suggestions` / `generation_error` 数量。`manual_review_*` 不计入普通 repair，它表示工具已经给 Agent 一个稳定动作分类：不要继续盲修同一份生成测试，应改走公共入口、依赖注入、集成环境或人工复核。

## 已稳定的能力

Go 链路已经能覆盖低依赖 utility、HTTP wrapper、JSON/file helper、panic/recover、JWT、泛型指针、nil receiver、Gin response、全局 logger 初始化、Unix socket 协议读取和多返回值 error branch。`validate_coverage_task` 也能把不可达分支、OS/runtime 错误分支、socket write / streaming I/O 时序分支和数据库分支从普通 ready 队列里分离。

JS/TS 链路已经验证过 Vitest、Jest、Mocha、ESM、TypeScript strict 编译、`ts-jest`、项目自定义 runner、测试目录 include 规则、monorepo 外部资源 symlink、默认导出实例、JS `#private`、TS `private/protected`、getter、未导出内部 helper 的公共入口覆盖、跨模块 interface/type/class/enum mock、Map/Set、函数类型字段、fake timers、动态 import 和 Node process/platform 全局状态。

Python/pytest 链路已经从基础 pytest 草稿推进到五个真实项目样本：能处理 coverage report 相对路径映射、dotted package import、仓库根包布局、单行安全 task 注释、class `__init__` 元数据、constructor 入参、常见 fallback/error path 输入、wrapper swallow exception、Unicode fallback、stream fallback、内部状态对象构造、Pydantic 通知 payload、Pydantic/BaseModel DTO 空类 file-level smoke、keyword-only 参数、dataclass 输入类型、同步/异步 stream finish 状态、ASGI scope、认证 scope、配置文件/cast 场景、可变多值字典状态断言、文件上传临时流、HTTP header 追加、ASGI endpoint 最小 scope / receive / send、WebSocket decode / dispatch、multipart parser 状态机、无 pyproject 的 `tests/...` 项目根推断、自定义 pytest runner/venv 命令模板、轻量库 binding 场景、bytes/buffer/vector index 参数、header/version-like 对象构造、IPv4/IPv6 解析和 in-memory/file/vector-index searcher 路径、FastAPI TestClient 业务流、SQLAlchemy SQLite 临时库、API Key 上传路径、zip/apk 文件输入、真实 APK 上传成功路径、同版本 build version、图标上传、上传失败数据库回滚、下载统计聚合、版本删除/应用删除数据库一致性、短链隐藏/无版本页面、refresh token 成功/禁用用户路径、发布说明长度限制和保存、设置当前版本、应用详情/版本列表 not found、下载流式成功路径、登录错误/禁用账号、删除用户 self/not found/成功路径、API Key 删除/不存在/list 更新、非 APK 重复上传版本递增、上传大小限制、storage facade qiniu/tos 分发、外部 APK parser SDK fake、下载 URL helper、版本详情、下载重定向 fallback、删除图标、隐藏应用、短链页面渲染、短链当前版本查询和最终 HTMLResponse 返回路径、短链缺失应用精确行段 `782-782` fake DB、FastAPI apps.py file-level 短链 fake DB、auth refresh helper、JWT refresh/access token 分支、API Key request fake、auth helper 吞异常返回 None、auth service API Key fake DB、认证/API Key 列表 fake DB、应用列表搜索 fake DB、`build_app_out` fallback 版本查询、QR import fallback、数据库迁移 fake Session/inspect、Qiniu 对象存储 helper fallback、storage backend 选择、Qiniu/TOS 对象存储错误返回值、fake SDK/client fallback、本地图标保存临时目录、Qiniu SDK 缺失路径、API Key 唯一性循环 fake DB、JWT `verify_exp=False` 解码分支、对象存储配置状态、storage facade 返回路径、对象存储模块级 file-level、FastAPI `lifespan` async context manager 启动钩子隔离，以及动态前端入口和根静态文件入口的环境手审分类。

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

haoy-apk-station 的 pytest baseline 已扩到 `13 passed`，关键文件覆盖率提升到 `apps.py=80.6%`、`auth.py=88.3%`、`storage.py=82.1%`；当前剩余 coverage task 池缩小到 168 个并已全部验证通过。第十三个样本 ip2region Python binding 则补上轻量库和二进制 buffer/searcher 场景：原始样本依赖外部 `data/*.xdb`，验证时用临时断言型测试构造最小 xdb buffer，baseline 为 `7 passed`、覆盖率约 `95.71%`；工具初跑因 pytest runner 环境和 `None` 参数退化只有 `ready=1/9`，修复后 top9 达到 `ready=9/9`、普通 repair 清零。下一步应选择新的 Python/JS/Go 真实样本继续验证跨项目泛化，优先挑选有项目自定义 runner、轻量二进制协议或无框架配置的仓库。

Rust 已补齐 opt-in top-task 验证入口，并用最小 Cargo crate + 自定义 LCOV writer 跑通 `passed=2/ready=2` 的 smoke；但由于本机 `cargo-tarpaulin` 不存在、`llvm-tools-preview` 下载未完成，该结果不计入真实项目样本表。Java 已从最小 Maven/JUnit smoke 推进到真实 Maven 多模块项目验证：RocketMQ Java client 的 `route/Endpoints.java` top20 使用真实 JaCoCo XML、真实 Maven reactor 和真实 JUnit runner 跑通 `passed=20/ready=20`，第二个文件 `exception/StatusChecker.java` top2 也达到 `passed=2/ready=2`、`zero_skip=2`。`Endpoints` 的 `skipped_total=20` 来自该项目现有测试套件每次固定 1 个 upstream skipped test，不是生成测试跳过；`StatusChecker` 任务没有 skipped。该样本暴露并修复了 JaCoCo package path 到嵌套 Maven module 的映射、深层 Java 包路径找不到 `pom.xml`、Maven module 需要从 aggregator 根目录使用 `-pl/-am` 执行、JUnit 4/5 风格识别、license header / Checkstyle、构造函数重载按行号过滤、getter/equals/hashCode 目标不能在 parser 阶段被丢弃、无默认构造器实例方法构造、protobuf builder 输入、`List<InetSocketAddress>` import 顺序、`toSocketAddresses` 集合/空值返回断言、line-range `equals` 分支断言、protobuf `Status` / `ReceiveMessageRequest` / `RpcFuture` 构造，以及 `AddressScheme.DOMAIN_NAME.equals(scheme)`、空集合、空值、多地址异常和普通地址列表构造路径输入问题。下一步应继续扩大 Java 真实验证窗口，观察 `toProtobuf`、Optional/外部 protobuf 类型和外部服务依赖的 action 分布。
