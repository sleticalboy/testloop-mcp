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
| 14 | RocketMQ Java client | Java / Maven + JUnit + JaCoCo | `route/Endpoints.java` top20 + `exception/StatusChecker.java` top2 + `hook/AttributeKey.java` top2 + `impl/ClientType.java` top4 + `hook/InflightRequestCountInterceptor.java` top2 + `hook/CompositedMessageInterceptor.java` top2 + `impl/ClientManagerImpl.java` top5 + `impl/ClientSessionImpl.java` top7 + `impl/ClientImpl.java` top8 + consumer value/behavior top6 + `ConsumerImpl.java` top20 | `Endpoints: passed=20/ready=20`，`StatusChecker: passed=2/ready=2`，`AttributeKey: passed=2/ready=2`，`ClientType: passed=4/ready=4`，`Inflight: passed=2/ready=2`，`Composited: passed=2/ready=2`，`ClientManager: passed=5/manual_review_internal=5`，`ClientSession: passed=7/manual_review_internal=7`，`ClientImpl: passed=8/manual_review_internal=8`，`consumer: passed=6/ready=6`，`ConsumerImpl: passed=20/ready=17/manual_review_internal=3` | 0 |
| 15 | JSON-java | Java / Maven + JUnit + JaCoCo | `JSONArray.java` top10 + `JSONObject.java` top10 + `JSONML.java` top10 + `XML.java` top10 | `JSONArray: passed=10/ready=6/manual_review_internal=4`，`JSONObject: passed=10/ready=1/manual_review_internal=9`，`JSONML: passed=10/manual_review_internal=10`，`XML: 初跑 passed=7/failed=3；修复后 passed=10/ready=3/manual_review_internal=7` | 0 |
| 16 | Apache Commons Codec | Java / Maven + JUnit + JaCoCo | `Base64.java` top5 + `DigestUtils.java` top10 + `Rule.java` top10 + `BaseNCodec.java` top7 + `HmacUtils.java` top7 + `Digest.java` top10 + `Blake3.java` top4 | `Base64: 初跑 passed=3/failed=2；修复后 passed=5/ready=2/manual_review_internal=3`；`DigestUtils: 初跑 failed=10；修复后 passed=10/ready=10`；`Rule: 初跑 passed=6/failed=4；修复后 passed=10/ready=4/manual_review_internal=6`；`BaseNCodec: 候选仅 7 个；复跑 passed=7/manual_review_internal=7`；`HmacUtils: 初跑 passed=1/failed=6；修复后 passed=7/ready=7`；`Digest: passed=10/manual_review_internal=10`；`Blake3: 候选仅 4 个；复跑 passed=4/manual_review_internal=4` | 0 |

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

`manual_review_internal` / `manual_review_private`：外部测试模块无法稳定访问的内部 helper、未导出 class/function、语言级 private method，或 public 方法虽然可见但需要复杂构造依赖和内部生命周期状态才能安全触发。优先走公共入口覆盖；没有公共入口时应手审或重构可测性。

`manual_review_external_service`：依赖 live RPC、gRPC endpoint、路由状态或长重试时序的路径。这类结果可能表现为 `failed/manual_review_external_service`，含义不是生成器失败，而是需要 fake client、依赖注入或集成环境。

`manual_review_no_runtime`：TypeScript type-only、barrel re-export 或无本地 runtime statement 的文件。应通过消费方行为测试或类型检查验证，不应伪造单元测试。

## 当前判断

项目当前的核心价值已经不再是“能生成测试文件”，而是能在真实仓库里把覆盖率缺口拆成可执行测试、明确手审项和环境依赖项，并把普通生成失败持续压到 0。后续新增语言或框架前，应优先要求它们能进入同样的验证闭环：可重复 baseline、隔离 task worktree、结构化 action、普通 repair 清零或可解释归类。

haoy-apk-station 的 pytest baseline 已扩到 `13 passed`，关键文件覆盖率提升到 `apps.py=80.6%`、`auth.py=88.3%`、`storage.py=82.1%`；当前剩余 coverage task 池缩小到 168 个并已全部验证通过。第十三个样本 ip2region Python binding 则补上轻量库和二进制 buffer/searcher 场景：原始样本依赖外部 `data/*.xdb`，验证时用临时断言型测试构造最小 xdb buffer，baseline 为 `7 passed`、覆盖率约 `95.71%`；工具初跑因 pytest runner 环境和 `None` 参数退化只有 `ready=1/9`，修复后 top9 达到 `ready=9/9`、普通 repair 清零。下一步应选择新的 Python/JS/Go 真实样本继续验证跨项目泛化，优先挑选有项目自定义 runner、轻量二进制协议或无框架配置的仓库。

Rust 已补齐 opt-in top-task 验证入口，并用最小 Cargo crate + 自定义 LCOV writer 跑通 `passed=2/ready=2` 的 smoke；但由于本机 `cargo-tarpaulin` 不存在、`llvm-tools-preview` 下载未完成，该结果不计入真实项目样本表。Java 已从最小 Maven/JUnit smoke 推进到真实 Maven 多模块项目验证：RocketMQ Java client 的 `route/Endpoints.java` top20 使用真实 JaCoCo XML、真实 Maven reactor 和真实 JUnit runner 跑通 `passed=20/ready=20`，后续又扩展到 `exception/StatusChecker.java` top2、`hook/AttributeKey.java` top2、`impl/ClientType.java` top4、`hook/InflightRequestCountInterceptor.java` top2、`hook/CompositedMessageInterceptor.java` top2、`impl/ClientManagerImpl.java` top5、`impl/ClientSessionImpl.java` top7、`impl/ClientImpl.java` top8，以及 consumer `Assignment.java` / `Assignments.java` / `ConsumeTask.java` / `ConsumeService.java` top6。前六组和 consumer value/behavior 小窗口全部为 `ready`，`ConsumerImpl.java` top20 已提升到 `passed=20/ready=17/manual_review_internal=3`，其中 `receiveMessage` 的 response switch、protobuf message list、transport delivery timestamp 和 StatusChecker 成功路径都已通过 `PushConsumerImpl` spy + mock `ClientManager.receiveMessage` 转为真实 ready；剩余手审只集中在 `wrapAckMessageRequest` 和 `wrapChangeInvisibleDuration` 两类 private wrapper。`ClientManagerImpl` 的五个 private/internal method 任务、`ClientSessionImpl` 的七个复杂 RPC session / observer / stream lifecycle 任务、`ClientImpl` 的八个抽象 client lifecycle / session / heartbeat 任务稳定归为 `manual_review_internal`。这些 hook/route/enum/session/client/consumer 文件的 `skipped_total` 来自该项目现有测试套件每次固定 1 个或每条手审测试 1 个 skipped test，不是普通生成测试跳过；`StatusChecker` 任务没有 skipped。该样本暴露并修复了 JaCoCo package path 到嵌套 Maven module 的映射、深层 Java 包路径找不到 `pom.xml`、Maven module 需要从 aggregator 根目录使用 `-pl/-am` 执行、JUnit 4/5 风格识别、license header / Checkstyle、构造函数重载按行号过滤、getter/equals/hashCode 目标不能在 parser 阶段被丢弃、无默认构造器实例方法构造、显式复杂构造器不可安全生成时的手审分类、未知自定义引用类型参数不应伪造无参构造、源文件 import 按实际引用复制、coverage 注释不应触发 import、私有构造器下选择 public static factory、enum body 方法递归解析、enum 常量 receiver、void 方法 assertion import 清理、hook 接口参数使用最小实现类、hook enum 输入和状态副作用断言、组合 hook 匿名接口实现、attribute map 前置状态、Java private/internal 方法手审分类、RocketMQ `TestBase` fixture、consumer listener/interceptor mock、匿名抽象类子类、protobuf builder 输入、`List<InetSocketAddress>` import 顺序、`toSocketAddresses` 集合/空值返回断言、line-range `equals` 分支断言、protobuf `Status` / `ReceiveMessageRequest` / `ReceiveMessageResponse` / `RpcFuture` 构造，以及 `AddressScheme.DOMAIN_NAME.equals(scheme)`、空集合、空值、多地址异常和普通地址列表构造路径输入问题。

第十五个样本 JSON-java 用非 RocketMQ 的 Maven/JUnit 项目验证 Java 生成器泛化边界。`JSONArray.java` top10 初跑为 `passed=4/failed=6`，失败集中在重载构造器 `new JSONArray(null)` 歧义、空数组直接调用 `getNumber/getFloat/optNumber`、以及 `write(null, ...)` 触发 NPE 而不是目标 `IOException -> JSONException` 分支。修复后复跑同一窗口达到 `passed=10`，其中 public task 为 `ready=6`，package/private/internal task 为 `manual_review_internal=4`，`zero_skip=0`，普通 repair 清零。这一轮补上了集合/泛型构造器 null cast、`JSONArray` 数值读取最小状态、invalid number fallback、writer IOException 包装路径和对应回归测试。随后扩到 `JSONObject.java` top10 和 `JSONML.java` top10，分别达到 `passed=10/ready=1/manual_review_internal=9`、`passed=10/manual_review_internal=10`，说明反射/parse helper 任务能稳定归类但 public ready 密度较低。`XML.java` top10 初跑为 `passed=7/failed=3`，普通失败集中在 `toJSONObject(Reader, keepNumberAsString, keepBooleanAsString)` 使用 `null` XML 输入，以及 `noSpace` 错误路径使用有效字符串；修复后复跑达到 `passed=10/ready=3/manual_review_internal=7`，普通 repair 清零。

第十六个样本 Apache Commons Codec 继续验证非 RocketMQ Java/JUnit 泛化。list-only top60 显示热点集中在 `DigestUtils.java`、`Digest.java`、`Base64.java`、`DaitchMokotoffSoundex.java` 和 `Rule.java`；首个 `Base64.java` top5 初跑失败 2 个，根因不是业务断言，而是生成器把测试写进项目既有 `Base64Test.java`，覆盖了 upstream helper 常量，导致 `Base64InputStreamTest` / `Base64OutputStreamTest` 编译失败。修复后 Java coverage task 会在目标测试文件已存在时写入 `Base64TestLoopTest.java`，并让生成类名跟随最终测试文件。第二个失败点是嵌套类目标 `Base64.Builder.setDecodeTableFormat` 被生成成未限定的 `Builder` 和 `null` 参数；修复后会使用 `Base64.Builder`、限定返回类型，并按 line range 选择 `Base64.DecodeTableFormat.MIXED` 等 enum 输入。复跑 `Base64.java` top5 达到 `passed=5`，其中两个 Builder 分支为真实 `ready`，三个 internal decode/encode/constructor 任务稳定归为 `manual_review_internal`，普通 repair 清零。

随后扩大到 `DigestUtils.java` top10。初跑 `failed=10`，失败高度集中在 SHAKE 方法族：`null` 参数在 `byte[]` / `InputStream` / `String` 重载间产生编译歧义，当前 JDK 不支持 `SHAKE128-256` / `SHAKE256-512` 时又会抛 `IllegalArgumentException`，而默认生成器错误地断言非空返回或追加空的 `IOException` 异常路径。修复后 SHAKE coverage task 会按目标重载生成 typed input，例如 `new byte[] { 97, 98, 99 }`、`new java.io.ByteArrayInputStream(...)` 或 `"abc"`，并生成兼容不同 JDK 的 try/catch：支持算法时断言返回非空，不支持算法时断言异常信息包含 `SHAKE`。复跑 `DigestUtils.java` top10 达到 `passed=10/ready=10`，普通 repair 清零。

`Rule.java` top10 初跑为 `passed=6/failed=4`：`parsePhoneme`、`parsePhonemeExpr` 和 `parseRules` 这类内部 parser helper 已稳定归为 `manual_review_internal`，普通失败集中在 `Rule.Phoneme` 无默认构造器、`join(null)`、以及 `Rule.getInstance(null, null, ...)` 导致运行期 NPE。修复后 `Rule.Phoneme.join/toString` 使用 `new Rule.Phoneme("...", Languages.ANY_LANGUAGE)` 构造真实 nested value object，并断言 phoneme text / 字符串前缀；`Rule.getInstance` 使用 `NameType.GENERIC`、`RuleType.RULES` 和 `"english"` 或 `Languages.LanguageSet.from(...)` 触发真实资源规则路径。复跑达到 `passed=10/ready=4/manual_review_internal=6`，普通 repair 清零。

`BaseNCodec.java` 过滤后只有 7 个候选任务，不存在 top10 窗口；实际 top7 覆盖 `BaseNCodec.gte0`、`getLength`、`isInAlphabet`、`isWhiteSpace` 和 `BaseNCodec.AbstractBuilder` 的三个 getter。复跑达到 `passed=7/manual_review_internal=7`，普通 repair 清零。这一轮没有新增 ready 能力，但确认抽象 codec 状态机和 nested abstract builder 的内部路径会被稳定分类为手审 skip，且生成文件仍能通过 Apache RAT、JUnit 5 和既有大测试套件。

`HmacUtils.java` 同样只有 7 个候选任务。初跑为 `passed=1/failed=6`，其中 `isAvailable("test")` 断言方向错误，`new HmacUtils("test", "test")` 使用了不存在算法名，`instance.hmac(null)` / `hmacHex(null)` 在 `Path`、`String` 或 `ByteBuffer` 重载间产生编译歧义或运行时空实例问题。修复后生成器对 HmacUtils 使用 `HmacAlgorithms.HMAC_SHA_256`、稳定 key 和 typed `java.nio.ByteBuffer.wrap(...)` 输入，复跑达到 `passed=7/ready=7`，普通 repair 清零。

`Digest.java` top10 全部集中在 CLI `Digest.run` 的多个行段：`93`、`105-114`、`143` 等。复跑达到 `passed=10/manual_review_internal=10`，普通 repair 清零。这一轮没有新增 ready 能力，但确认 CLI run 的内部循环、输出和文件/目录处理路径不会被静态生成器伪造成脆弱单测，而是稳定给出可运行手审 skip。

`Blake3.java` 过滤后只有 4 个候选任务，不存在 top10 窗口；实际 top4 覆盖 `Blake3.checkBufferArgs` 的三个 buffer 参数检查分支和 `Blake3.doFinalize` 的内部 finalize 分支。复跑达到 `passed=4/manual_review_internal=4`，普通 repair 清零。这一轮继续确认二进制状态对象的内部 helper 会被稳定归为手审 skip，而不是直接生成外部不可访问断言。下一步应切到 Commons Codec 更大的跨文件窗口，观察 `Path/File/InputStream`、外部资源输入和跨类 helper 是否还能保持普通 repair 清零。
