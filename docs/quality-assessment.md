# testloop-mcp 质量评估

## 当前定位

testloop-mcp 的价值不应该定位为“独立的多语言测试生成器”，而应该定位为“面向 AI 编程代理的测试反馈闭环 MCP 服务”。测试生成领域已经有比较成熟的工具和 LLM 方案，例如 Go 生态里的 gotests，以及多语言 MCP/AI 测试生成工具。

更完整的产品定位见 [项目定位](./product-positioning.md)。后续功能判断应优先服从该定位：本项目要解决的是测试反馈闭环、结构化结果和质量控制，而不是单纯追求生成更多测试代码。

更适合本项目的产品方向是成为 AI 编程工具的工作流层：

- 生成或请求生成测试
- 运行本地正确的测试框架
- 把失败结果解析成 AI 友好的 JSON
- 分析覆盖率缺口
- 为修复阶段打包上下文
- 重新运行测试，推动闭环收敛

当前真实项目样本的最新指标见 [真实项目验证质量报告](./real-project-validation.md)。该报告比 roadmap 更集中地记录了各语言链路的最新验证窗口、action 分布和手审边界。

## 生成能力质量

Go 生成器仍是最成熟的静态生成路径。普通生成会优先使用 `gotests`，回退内置 `go/ast`；内置路径能提取函数、方法、接收者、参数、返回值、接口和常见复合类型，并且已经能为简单纯函数生成可执行表驱动 case。传入 `coverage_task` 时，Go 会跳过 `gotests` 的整文件骨架，直接聚焦目标函数或方法，并写入任务测试名、case 名和 task 注释。当前 Go task 模式已经覆盖简单分支、返回路径、错误路径、HTTP/JSON/JWT/recover、泛型指针返回、包级变量临时覆盖和 receiver 字段变更断言；在 laoxia server top50 隔离验证中，50 个任务全部通过，45 个任务为 `skipped=0`，剩余 5 个被归类为不可达或环境依赖人工复核项。

JavaScript、TypeScript 和 Python 的声明提取已经通过 tree-sitter 得到明显改善。它们能更可靠地识别函数、类、异步函数、参数、静态方法和基础函数体，比正则扫描稳很多。简单 return 表达式、if-return 分支和常见边界输入已经能生成更具体的断言；传入 `coverage_task` 时，会按目标过滤测试草稿，并把 `assertion_focus`、`suggested_inputs` 转成测试名、注释、调用参数和精确断言。复杂业务构造仍需要 Agent 或 LLM provider 二次增强，但真实样本已经证明这类缺口可以被 `validate_coverage_task` 稳定暴露并转成回归规则。

JS/TS payload 已经从基础对象推进到同文件 DTO、数组、tuple、utility wrapper、Pick/Omit、Record、交叉类型、indexed access 和对象字段内嵌组合，并覆盖 `response.json()` 与注入式 client 两条真实生成路径。详细支持范围、保守回退和不支持边界见 [JS/TS payload 质量边界说明](./js-ts-payload-quality.md)。

Python/pytest coverage task 当前已验证过五个真实项目样本，能处理 pytest-cov `coverage.json`、包路径 import、仓库根包布局、无 pyproject 的 `tests/...` 项目根推断、自定义 pytest runner/venv 命令模板、class constructor 元数据、常见 fallback/error path 输入、环境依赖手审分类、内部状态机、Pydantic 通知对象、Pydantic/BaseModel DTO 空类 file-level smoke、dataclass 输入类型、keyword-only 参数、同步/异步 stream finish 状态、ASGI scope、认证 scope、配置文件/cast 场景、可变多值字典状态断言、文件上传临时流、HTTP header 追加、ASGI endpoint 最小 scope / receive / send、WebSocket decode / dispatch、multipart parser 状态机、轻量库 binding 的 bytes/buffer/vector index 参数、header/version-like 对象构造、IPv4/IPv6 解析和 in-memory/file/vector-index searcher 路径、FastAPI TestClient 业务流、SQLAlchemy SQLite 临时库、API Key 上传路径、zip/apk 文件输入、真实 APK 上传成功路径、同版本 build version、图标上传、上传失败数据库回滚、下载统计聚合、版本删除/应用删除数据库一致性、短链隐藏/无版本页面、refresh token 成功/禁用用户路径、发布说明长度限制和保存、设置当前版本、应用详情/版本列表 not found、下载流式成功路径、登录错误/禁用账号、删除用户 self/not found/成功路径、API Key 删除/不存在/list 更新、非 APK 重复上传版本递增、上传大小限制、storage facade qiniu/tos 分发、外部 APK parser SDK fake、下载 URL helper、版本详情、下载重定向 fallback、删除图标、隐藏应用、短链页面渲染、短链当前版本查询和最终 HTMLResponse 返回路径、短链缺失应用精确行段 `782-782` fake DB、FastAPI apps.py file-level 短链 fake DB、auth refresh helper、JWT refresh/access token 分支、API Key request fake、auth helper 吞异常返回 None、auth service API Key fake DB、认证/API Key 列表 fake DB、应用列表搜索 fake DB、`build_app_out` fallback 版本查询、QR import fallback、数据库迁移 fake Session/inspect、Qiniu 对象存储 helper fallback、storage backend 选择、Qiniu/TOS 对象存储错误返回值、fake SDK/client fallback、本地图标保存临时目录、Qiniu SDK 缺失路径、API Key 唯一性循环 fake DB、JWT `verify_exp=False` 解码分支、对象存储配置状态、storage facade 返回路径、对象存储模块级 file-level、FastAPI `lifespan` async context manager 启动钩子隔离，以及动态前端入口和根静态文件入口的环境手审分类。haoy-apk-station 在扩充 baseline 后，业务基线已到 `13 passed`，`apps.py` 覆盖率从 37.7% 提升到 80.6%，`auth.py` 为 88.3%，`storage.py` 为 82.1%；剩余 168 个覆盖任务全部通过，普通生成失败清零。ip2region Python binding 的临时断言基线为 `7 passed`、覆盖率约 `95.71%`，top9 已达到 `ready=9`、`skipped_total=0`。下一步建议继续挑选新的真实样本，重点验证这些通用启发式在非 ip2region 项目里的泛化。

Rust 和 Java 已经从实验能力推进到可用的闭环能力：`run_tests`、`parse_results`、`parse_coverage`、`test_tasks` 和 `generate_tests` 都已覆盖对应主路径。Rust 已新增 opt-in top-task 验证脚本，支持通过自定义 LCOV 命令接入 `cargo tarpaulin`、`cargo llvm-cov` 或项目自有覆盖率命令；同时修复了 Rust coverage task 写入源文件时覆盖原 `.rs` 源码的问题，现在会追加内联 `#[cfg(test)]` 测试模块。Java 也已新增 Maven/Gradle JaCoCo top-task 验证脚本，支持自定义 coverage 命令、JaCoCo XML 路径、文件过滤和阶段 timeout，并用最小 Maven/JUnit smoke 跑通 `JaCoCo XML -> coverage task -> generate_tests -> run_tests`。

Java 真实项目链路已经用 RocketMQ Java client 从单文件扩展到多文件小窗口：`route/Endpoints.java` top20 在真实 Maven reactor、真实 JaCoCo XML 和真实 JUnit runner 下达到 `passed=20/ready=20`，`exception/StatusChecker.java` top2、`hook/AttributeKey.java` top2、`impl/ClientType.java` top4、`hook/InflightRequestCountInterceptor.java` top2 和 `hook/CompositedMessageInterceptor.java` top2 也全部达到 `ready`，`impl/ClientManagerImpl.java` top5 则稳定归为 `manual_review_internal=5`，普通 repair 清零。hook/route/enum 文件的 `skipped_total` 来自该项目现有测试套件每次固定 1 个 upstream skipped test，不是生成测试跳过；`StatusChecker` 没有 skipped。这一轮把 Java 生成器从骨架进一步推进到项目规范可运行：能复用源文件 license header 和 package，按 Maven 依赖识别 JUnit 4/5，避免 star import 并按项目 Checkstyle 要求排序 import，生成 `public class/public void` 以兼容旧 Surefire，按 coverage line range 选择重载构造函数，coverage task 下保留 getter/equals/hashCode 目标，为无默认构造器的实例方法选择可用构造器，并能在私有构造器场景下选择 public static factory。Java parser 现在也能递归进入 enum body declaration，生成器能把 enum 常量作为 receiver 覆盖 `toProtobuf` 这类分支；对 hook 状态型 void 方法，也能使用 `MessageInterceptorContextImpl` 最小实现和 `MessageHookPoints.RECEIVE` 输入生成状态副作用断言，并清理未使用 assertion import；对组合 hook，也能生成匿名 `MessageInterceptor`、非空 interceptor 列表和 `doAfter` 所需的 attribute map 前置状态；对 Java private/internal 方法，会生成可运行的手审 skip，避免伪造直接调用或输出空测试类。其他已验证能力包括 `AddressScheme.DOMAIN_NAME.equals(scheme)`、`addresses.isEmpty`、`addresses.size`、空值路径、protobuf `Endpoints` builder 输入、`toSocketAddresses` 集合/空值返回、`equals` 细分返回路径、protobuf `Status` 和 `RpcFuture` 上下文构造。当前 Java 样本窗口仍集中在 RocketMQ client，不能外推为通用 Java 项目质量；下一步应继续扩大到 RPC session、observer 状态、Optional/外部 protobuf 类型和外部服务依赖，观察普通 repair 或手审分类。

## 解析能力质量

解析层仍是测试闭环可靠性的关键面，但主流框架的失败解析已经从摘要级别推进到结构化失败对象级别。后续重点应放在更多真实输出 fixture、异常输出兼容和跨框架字段一致性上。

Go 解析已经优先支持 `go test -json`，能更稳健地处理结构化事件、编译失败、panic 细节、子测试状态和包级失败；旧版文本解析仍作为兼容路径保留。

Jest 和 Vitest 解析已经能提取失败测试名、断言消息、`expected`、`received`、栈位置里的文件、行号和列号，并有真实失败输出 fixture 防止回退。仍需持续补充快照、异步错误、自定义 matcher 和多文件失败输出样例。

pytest 解析已经能识别结果行、失败段落、断言详情、异常信息和源码位置，不再只依赖宽泛的 `PASSED` / `FAILED` 字符串扫描。

Mocha 解析已经能用 summary 统计避免重复计数，并提取常见失败段落里的测试名、错误消息、文件、行号和列号。它仍比 Go/Jest/pytest 更依赖文本结构，后续应继续补充真实项目输出样例。

## 修复闭环质量

失败修复闭环已经从“给出自然语言建议”推进到“生成 Agent 可消费的修复任务”。`fix_suggestions` 会输出 `category`、`context_file`、`context_line` 和 `repair_task`，把失败类型、目标位置、上下文片段、可编辑文件、建议复跑命令和断言关注点聚合成稳定 JSON。`run_tests` 也可以通过 `include_fix_suggestions=true` 在失败结果中内联 `fix_suggestions[]`，减少 `parse_results -> fix_suggestions` 的额外往返。

这一层的质量优势在于可编排性：Agent 不必重新从整段日志里推断修复入口，可以直接读取 `repair_task.target_file` / `target_line` 跳转，按 `editable_files` 控制改动范围，再用 `suggested_commands` 复跑。当前已通过真实 Jest/Vitest/Mocha/pytest fixture 和 repair task golden test 固定主路径契约。

仍需注意的风险是上下文来源依赖调用方传入 `source_code` / `test_code`。如果只运行目录而没有传文件路径，`repair_task` 仍会生成，但 `context_snippet` 和文件定位可能退化。因此下一阶段应增强源码/测试文件自动发现，特别是 Go package、pytest module、Jest/Vitest stack path 到本地文件的映射。

## 战略建议

项目应该继续推进，但核心使命要收窄：

> testloop-mcp 应该成为 AI 编程代理使用的测试反馈闭环与编排 MCP。

它应该在成熟工具擅长的地方集成外部工具，在结构化上下文上使用 tree-sitter，在语义测试生成上可选地委托 LLM，并重点投入测试执行、失败解析和覆盖率驱动能力。

这样可以避免直接和 gotests 或通用 LLM 测试生成器竞争，同时保留 MCP 原生工作流的差异化价值。
