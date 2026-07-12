# Changelog

## Unreleased

### Changed

- Go coverage task 写入测试文件前会扫描同包所有 `*_test.go`，当任务推荐的 `test_name` 已在其它测试文件中存在时，也会自动追加稳定后缀，避免生成后 `go test` 因包级 `Test*` 重名构建失败。
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
