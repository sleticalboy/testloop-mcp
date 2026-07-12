# 路线图

## 产品方向

testloop-mcp 应定位为面向 AI 编程代理的测试反馈闭环 MCP 服务。项目优先级应该放在测试执行、结构化失败解析、覆盖率驱动定位和修复上下文打包上，而不是为每门语言手写完整静态测试生成器。

详细定位见 [项目定位](./product-positioning.md)。后续路线以“测试反馈与质量控制 MCP 层”为主线，避免退回到单纯模板测试生成器。

## 第一阶段：稳定测试反馈闭环

- [x] 增加 `go test -json` 的结构化解析。
- [x] 保留旧版 `go test -v` 文本解析作为回退。
- [x] 改善失败对象，包含测试名、文件、行号和精确错误输出。
- [x] 让 `run_tests` 默认使用 Go 结构化输出。
- [x] 增加 pass、fail、skip、包级失败和覆盖率输出的测试。

## 第二阶段：减少低价值生成测试

- [x] 停止生成会立刻失败的零值业务测试。
- [x] 当无法推断有意义输入时，把生成用例标记为 TODO 或 skip。
- [x] Go 方向评估把 gotests 作为主要静态骨架生成器，本地生成器作为回退。
- [x] 准确返回生成用例数量。

已完成补充：Go 测试生成现在会优先尝试调用外部 `gotests -all` 从 stdout 生成标准表驱动测试骨架；当本机未安装 `gotests`、命令执行失败或输出为空时，自动回退到项目内置 AST 生成器。这样既借用 Go 社区成熟工具，又不让 MCP 服务增加硬依赖。

## 第三阶段：提升 JS/TS/Python 语义质量

- [x] 继续使用 tree-sitter 做结构提取。
- [x] 提供面向 LLM 的基础上下文包，包含测试目标、参数、返回类型、错误路径和边界条件。
- [x] 用 LLM 上下文包替代占位参数测试。
- [x] 提取更完整的导入、邻近类型和返回表达式信息。
- [x] 在 provider 接口后面增加可选 LLM 生成能力。

已完成补充：`generate_tests` 的上下文现在会返回 JS/TS/Python 源文件中的导入语句、邻近类型声明，以及每个测试目标的 return 表达式，便于后续生成器或 LLM 生成更贴近源码语义的断言和构造代码。

已完成补充：JS/TS/Python 生成器已先用参数名、默认值、边界条件和异常路径信息替代常见占位参数。正常路径会优先生成 URL、数字、布尔、数组、对象、字符串等可运行示例值；异常路径会优先使用 `null`、`undefined`、`None` 等边界输入，避免与正常路径互相干扰。真正接入外部 LLM provider 仍作为独立能力保留。

已完成补充：新增测试生成 provider 接口。`generate_tests` 默认继续使用 `static` 静态生成；传入 `provider: "llm"` 或 `provider: "auto"` 时，可以通过服务端环境变量 `TESTLOOP_LLM_PROVIDER_CMD` 接入外部 LLM 命令。provider 从 stdin 接收源码上下文和静态生成结果，stdout 返回测试代码或 `{"code":"..."}`，当前不绑定具体厂商。

## 第四阶段：强化非 Go 解析器

1. [x] 解析 Jest/Vitest 的断言消息、expected/received 和栈位置。
2. [x] 解析 pytest 的失败段落和源码位置，避免依赖宽泛字符串匹配。
3. [x] 解析 Mocha 的摘要和失败段落，避免重复计数。
4. [x] 基于真实框架输出增加 fixture 测试。

已完成补充：Jest/Vitest 失败结果现在会提取断言消息、`expected`、`received`、失败文件、行号和列号，并通过 `TestFailure` 结构化返回。

已完成补充：pytest 失败结果现在会解析测试结果行、汇总行、失败段落、断言细节、异常信息和源码位置，避免通过宽泛 `FAILED` / `Error` 字符串重复计数。

已完成补充：Mocha 解析现在优先使用 summary 统计，避免 spec 行和 summary 重复计数；失败段落会提取完整测试名、错误消息、失败文件、行号和列号。

已完成补充：新增 Jest、Vitest、pytest、Mocha 失败输出 fixture，统一验证解析出的测试名、错误消息、源码位置和断言字段。

## 第五阶段：覆盖率驱动测试规划

1. [x] 把 Go 覆盖率缺口关联到源码函数和方法。
2. [x] 生成针对性测试任务，而不是整文件泛化测试。
3. [x] 返回给 AI 代理的建议应包含目标函数、缺失分支、建议输入形状和置信度。

已完成补充：Go coverprofile 的未覆盖 block 会通过 AST 映射到函数或方法，并在 coverage suggestion 中返回 `function`、`kind`、`uncovered_lines` 和参数相关的 `suggested_inputs`。

已完成补充：coverage report 现在会返回 `test_tasks`，把 coverage suggestion 转成面向 AI Agent 的测试任务，包含目标、行段、推荐命令、建议输入和置信度。

已完成补充：Go 覆盖率任务现在会基于源码行推断 `gap_type` 和 `missing_branches`，例如未覆盖 if 分支、switch/case 分支、错误路径或返回路径，并生成更贴近条件表达式的输入建议。

已完成补充：Go 覆盖率任务的分支条件抽取已从源码文本猜测改为 AST 节点匹配。`parse_coverage` 现在会从 `if` 条件表达式中剥离 `if init` 语句，并只在未覆盖 block 与真实 `if`、`switch`、`select` 或 `return` 语法节点重叠时标记分支或返回路径。用 `laoxia-scaffold-v1.0.0/car-admin-server` 复验后，函数签名、router 初始化普通语句、for 语句和 `if init` 泄漏这几类伪分支条件均降为 0；剩余重复分支任务属于后续任务聚合优化范围。

已完成补充：coverage suggestion 和 `test_tasks` 已支持相邻/重叠未覆盖 block 聚合。同文件、同函数/方法、同缺口类型、同分支条件的连续行段会合并成一个任务，并合并 `line_range`、`uncovered_lines`、`suggested_inputs` 和断言关注点。用 `laoxia-scaffold-v1.0.0/car-admin-server` 复验后，任务数从 1206 降到 715，相邻/重叠重复任务为 0；非相邻位置的同类 `err != nil` 仍保留为独立任务。

已完成补充：coverage task 排序新增路径环境成本启发式。`utils`、helper、parser、validator、time/date/format 等低依赖路径会加权提前；controller、router、service、middleware、initialize、cmd、db/cache/redis/email/captcha 等高初始化成本路径会降权。用 `laoxia-scaffold-v1.0.0/car-admin-server` 复验后，前 30 个任务均为 `utils` 目录任务，第一个 controller 任务排到第 72 位，第一个 service 任务排到第 224 位，更适合 Agent 先处理可自动补测、可快速提升覆盖率的目标。

## 第六阶段：补齐 Rust/Java 覆盖率解析

1. [x] 解析 Rust `cargo tarpaulin --out Lcov` 生成的 LCOV。
2. [x] 解析 Java JaCoCo XML。
3. [x] 将 Rust/Java 覆盖率结果转换为统一 `CoverageReport`、`suggestions` 和 `test_tasks`。
4. [x] 在 `run_tests` 的 coverage 模式中进一步集成 tarpaulin/JaCoCo 报告生成命令。
5. [x] 补充 Rust/Java 覆盖率闭环 e2e，验证 `run_tests` 与 `parse_coverage` 联动。

已完成补充：`parse_coverage` 现在支持 `cargo-test` 的 LCOV 覆盖率数据和 `junit` 的 JaCoCo XML 数据。二者会返回文件级覆盖率、未覆盖行 block、改进建议和面向 AI Agent 的测试任务。`run_tests coverage=true` 对 Rust 会额外调用 tarpaulin 生成 LCOV 并回填 `coverage_percent`；对 Java Maven/Gradle 项目会执行 JaCoCo report 任务并从 XML 报告回填 `coverage_percent`。

## 第七阶段：提升生成测试的业务断言质量

1. [x] Python 生成器对简单 return 表达式生成精确断言。
2. [x] Jest 生成器对简单 return 表达式生成精确断言。
3. [x] Python/Jest 边界用例将边界值带入简单 return 表达式。
4. [x] Go 内置生成器减少 TODO/skip，优先为简单纯函数生成可执行表驱动断言。
5. [x] Python/Jest 进一步识别多分支 return，为正常路径和边界路径分别生成期望值。
6. [x] 为 Go/Python/Jest 代表性生成结果补充 golden tests，防止输出质量回退。

已完成补充：Python/Jest 对 `a + b`、`prefix + text` 等单一安全 return 表达式，会基于语义默认参数生成精确断言，而不是只断言返回类型。Go 内置生成器对简单纯函数会生成 `skip: false` 的表驱动 case，例如 `Add(a, b int) int { return a + b }` 会生成 `a: 1`、`b: 2`、`ret0: 1 + 2`。Python/Jest 也会识别简单 `if param == value: return ...` 分支，让普通路径和边界路径分别断言对应返回值。这些代表性输出已用 golden tests 固定。

## 第八阶段：增强 Rust/Java 覆盖率建议上下文

1. [x] Rust LCOV 未覆盖行映射到所在 `fn` 范围。
2. [x] Java JaCoCo 未覆盖行映射到所在类方法范围。
3. [x] 覆盖率建议和 `test_tasks` 填充 `function`、`kind`、`uncovered_lines` 和参数输入提示。
4. [x] Java 源码路径解析覆盖常见 `src/main/java` 目录结构。
5. [x] 进一步解析 Rust/Java 分支语义，例如 `if`、`match`、`switch`、异常路径和错误返回路径。

已完成补充：Rust/Java 覆盖率建议不再只停留在文件和行号层面。`parse_coverage` 会在能读取源码时，把 Rust `fn` 和 Java 类方法范围映射到未覆盖 block，并把具体目标、参数输入提示和未覆盖行写入 suggestion/test task，方便 AI Agent 直接为目标函数或方法补测试。当前也会对 `if`、`match`、`switch`、错误/空值返回和普通返回做轻量语义分类，输出更具体的 `gap_type`、`missing_branches` 和输入提示。

## 第九阶段：提升源码结构解析稳健性

1. [x] 用 tree-sitter 或语言 AST 替换 Rust/Java 当前的轻量正则源码范围扫描。
2. [x] 正确处理 Java 多类文件、内部类、注解、多行方法签名和构造函数。
3. [x] 正确处理 Rust impl 方法、trait 方法、多行函数签名和属性标注函数。
4. [x] 为真实开源项目覆盖率报告增加 fixture，验证路径解析和源码映射在复杂目录下不退化。

已完成补充：Java 覆盖率源码映射已优先使用 tree-sitter，不再依赖单行方法签名正则。JaCoCo 未覆盖行现在可以映射到带注解的方法、多行参数列表、构造函数和内部类方法；tree-sitter 解析未产出范围时仍保留轻量正则回退。

已完成补充：Rust 覆盖率源码映射已优先使用 tree-sitter，不再依赖单行 `fn` 正则。LCOV 未覆盖行现在可以映射到属性标注函数、多行函数签名、`impl` 方法和 trait 默认方法；tree-sitter 解析未产出范围时仍保留轻量正则回退。

已完成补充：新增 Rust workspace 风格和 Java Maven 风格覆盖率 fixture，覆盖相对报告路径、复杂源码目录、内部类、`impl` 和 trait 场景，防止路径解析和源码映射在真实项目布局下退化。

## 第十阶段：让覆盖率任务更容易被 Agent 消费

1. [x] 为 `test_tasks` 增加更明确的生成指令字段，例如推荐测试文件路径、测试函数名和断言重点。
2. [x] 按风险和收益对覆盖率任务排序，优先暴露低覆盖率高价值目标。
3. [x] 支持按单个任务触发 `generate_tests`，让覆盖率缺口可以直接转成增量测试草稿。
4. [x] 为 Rust/Java/Go 的 coverage task 输出增加快照测试，固定面向 Agent 的 JSON 契约。

已完成补充：`test_tasks` 现在会输出 `test_file`、`test_name` 和 `assertion_focus`，分别给出推荐测试文件路径、推荐测试函数名和断言重点，便于 AI Agent 直接把覆盖率缺口转成测试草稿。

已完成补充：`test_tasks` 现在会输出 `priority` 和 `priority_reason`，并按任务价值排序。具体函数/方法、分支或错误路径、有建议输入、有未覆盖行列表和高置信度的任务会优先出现；整文件泛化任务会靠后。

已完成补充：`generate_tests` 现在支持传入单个 `coverage_task`。工具会优先写入任务推荐的 `test_file`，在输出中回显 `coverage_task`，并把任务放入 `context.coverage_task` 传给 LLM provider；普通静态生成保持兼容，task 模式已在后续阶段增强为目标函数/方法级增量生成。

已完成补充：新增 Go/Rust/Java 的 coverage task JSON golden 快照测试，固定 `id`、`target`、`gap_type`、`test_file`、`test_name`、`assertion_focus`、`priority` 和 `priority_reason` 等面向 Agent 的输出契约。

## 第十一阶段：提升按覆盖率任务生成测试的静态质量

1. [x] 让 Go 静态生成器在收到 `coverage_task` 时优先生成覆盖目标函数/方法和指定缺口的 case。
2. [x] 让 Python/Jest 静态生成器把 `coverage_task.assertion_focus` 和 `suggested_inputs` 转成更具体的测试名与输入。
3. [x] 让 Rust/Java 静态生成器在 task 模式下减少整文件泛化测试，优先生成目标函数/方法测试骨架。
4. [x] 为 task-aware 静态生成增加 golden tests，防止增量测试草稿退化。

已完成补充：Go 静态生成器在收到 `coverage_task` 时会跳过 `gotests` 路径，使用内置 AST 生成器过滤到目标函数/方法，并把 task 的测试名、缺口类型和断言重点写入测试函数、case 名称和注释。

已完成补充：Python/Jest 静态生成器在收到 `coverage_task` 时会过滤到目标函数/方法，使用任务推荐测试名，把 `assertion_focus` 和 `suggested_inputs` 写入注释，并从建议输入中的条件表达式提取参数值生成更贴近覆盖率缺口的调用和精确断言。

已完成补充：Rust/Java 静态生成器在收到 `coverage_task` 时会过滤到目标函数或方法，使用任务推荐测试名，把 task 上下文写入注释，并将建议输入中的条件值代入生成的调用参数，减少整文件泛化测试草稿。

已完成补充：新增 Go/Python/JS/TS/Rust/Java 的 task-aware 静态生成 golden tests，固定目标过滤、任务推荐测试名、coverage task 注释和建议输入代入后的代表性输出。

## 第十二阶段：发布前质量收敛

1. [x] 补齐 v0.4.0 发布说明草案，明确覆盖率任务到增量测试生成的闭环。
2. [x] 同步 README 和 LLM provider 文档，说明 `coverage_task` 在 static/LLM provider 中的行为。
3. [x] 更新质量评估，移除 Rust/Java 仍属实验能力等过时判断。
4. [x] 发布前执行完整 release checklist，确认构建、Docker、CLI、CI 和文档状态一致。
5. [x] 视 CI 和版本策略决定是否打 `v0.4.0` tag 并创建 GitHub Release。

已完成补充：新增 `docs/plan-release-notes-v0.4.0.md`，并同步 README、LLM provider 文档和质量评估，确保用户文档描述的 `coverage_task -> generate_tests -> run_tests` 闭环与当前代码能力一致。

已完成补充：已执行 v0.4.0 发布前 release checklist 复验，包括 `go test ./...`、主服务构建、CLI 构建、`docker compose config`、本地 HTTP `/healthz`、Docker 镜像构建和容器 `/healthz`。

已完成补充：`v0.4.0` 已发布到 GitHub Release，tag 指向 `77c5107d22b013e0042b6788394dd0015b4b9294`。

## 第十三阶段：发布后维护体验

1. [x] 为 CI workflow 增加 `workflow_dispatch`，便于后续手动触发验证和恢复 queued 状态。
2. [x] 清理发布说明中的发布前操作痕迹，并同步已发布的 GitHub Release 页面。
3. [x] 评估是否需要为 release workflow 增加二进制产物构建和校验文件。

已完成补充：已用更新后的 `docs/plan-release-notes-v0.4.0.md` 同步 GitHub Release 页面，移除发布前操作命令，保留发布信息和验证结果。

已完成补充：新增 `Release Artifacts` workflow。后续 tag push 或手动指定 tag 时，会构建 Linux amd64 的 `testloop-mcp` 和 `testloop-testgen` 压缩包，生成 `checksums.txt`，并上传到对应 GitHub Release。已用手动 run `28739120265` 回填 `v0.4.0` 资产。macOS、Windows 和多架构产物因 CGO 交叉编译成本暂缓。

## 第十四阶段：安装与分发体验

1. [x] 修正仓库地址和 Go module path，确保 README、Go Report Card、`go install` 与当前 GitHub 远端一致。
2. [x] 补齐 MIT `LICENSE` 文件。
3. [x] 新增安装与接入文档，覆盖 Release 下载、checksum 校验、源码构建、Docker、stdio、Streamable HTTP、Codex、Claude 和 Cursor。
4. [x] README 安装部分改为快速路径，并链接详细安装文档。
5. [x] 验证远端 `go install @main` 安装主服务和 `testgen` CLI 可用。
6. [x] 准备 v0.4.1 patch release，让 module path 修正进入正式 tag，并把安装文档切回 `@latest`。
7. [x] 发布 v0.4.1，并验证 Release 资产和 `go install @latest`。
8. [x] 评估 macOS、Windows 和多架构二进制发布。
9. [x] 评估 Homebrew tap 或一键安装脚本。

详细规划见 [安装与分发体验规划](./plan-installation.md)。

已完成补充：安装与分发工作已收敛到 `v0.4.10`。当前 release matrix 覆盖 Linux amd64、Linux arm64、macOS arm64、Windows amd64 和 Windows arm64，安装脚本支持 release 资产下载、checksum 校验、Windows zip、下载重试/超时和 `go install` 回退，`sleticalboy/tap` 已升级到 `0.4.10` 并通过 Homebrew 验证。逐版本发布证据和后续维护流程统一记录在 `docs/plan-installation.md`。

## 第十五阶段：生态集成和编辑器体验

1. [x] 评估是否需要 VS Code Extension 配套；当前 MCP 服务器已能通过通用 MCP 客户端接入，扩展应聚焦更低摩擦的本地配置、命令发现和结果展示，而不是重复 MCP 协议能力。
2. [x] 提供面向 Codex、Claude Code / Claude Desktop 和 Cursor 的本地配置生成和校验入口，降低首次接入成本。
3. [x] 为 `run_tests -> parse_results -> parse_coverage -> generate_tests` 增加一个端到端示例工作流，便于编辑器侧直接展示闭环步骤。

已完成补充：暂不把 VS Code Extension 作为近期主线。当前项目的差异化仍在 MCP 工具层和结构化测试反馈；编辑器体验先通过配置生成、配置校验和示例工作流降低接入摩擦，避免维护一个重复 MCP 能力的扩展壳。

已完成补充：主二进制新增 `--print-config`，可输出 Codex、Codex HTTP、Claude Code / Claude Desktop 和 Cursor 的配置片段；源码仓库也提供 `scripts/generate-client-config.sh` 辅助入口。两者都只打印配置，不直接修改用户全局配置文件。

已完成补充：主二进制新增 `--check-config`，可读取配置文件或 stdin，检查 MCP server 的 `command` 是否存在且可执行，或 `url` 是否是合法 HTTP endpoint。校验失败时会输出对应的配置生成或诊断建议。

已完成补充：主二进制新增 `--doctor-config`，可输出当前二进制路径、PATH 解析结果、推荐配置路径，并对已存在的 Codex、Claude 和 Cursor 配置做只读校验。诊断发现缺少 `testloop` server、配置项无效或 PATH 缺失时，会给出可执行的下一步建议。

已完成补充：新增 `docs/agent-workflow.md`，用 Go demo 展示 `run_tests`、`parse_results`、`fix_suggestions`、`parse_coverage`、`generate_tests` 和重新运行测试的闭环顺序，并明确先修真实失败、再补覆盖率缺口，覆盖率报告需要先由生态工具生成。

已完成补充：`fix_suggestions` 现在会返回 `category`、`context_file` 和 `context_line`，并在建议文本中带上 actual/want、越界 index/length、panic 类型和匹配到的源码或测试行上下文，便于 Agent 在失败修复阶段先分类再跳转。

已完成补充：`v0.4.9` 已完成发布，tag 指向 `be0e18028b3994693f82b8b4cc5547c965588d5c`。Release Artifacts run `28833047972` 已通过，Release 资产覆盖 Linux amd64、Linux arm64、macOS arm64、Windows amd64 和 Windows arm64，Homebrew tap 已升级到 `0.4.9` 并通过 `brew test`。

## 第十六阶段：失败修复闭环落地

1. [x] 为 `fix_suggestions` 增加更多真实框架失败 fixture，覆盖 pytest、Jest/Vitest、Mocha 的断言失败和异常场景。
2. [x] 增加面向 Agent 的 repair task 输出，把失败分类、源码上下文、建议命令和可编辑文件聚合为稳定 JSON 契约。
3. [x] 让 `run_tests` 在失败结果中可选附带 `fix_suggestions` 摘要，减少 Agent 额外往返。
4. [x] 为 repair task 增加 golden tests，固定字段、排序和降级行为。

已完成补充：新增 `parse_results` 真实 fixture 到 `fix_suggestions` 的联动测试，覆盖 Jest、Vitest、Mocha 的 expected/received 断言失败，以及 pytest 的 `division by zero` 异常。`fix_suggestions` 现在会利用 `TestFailure.Expected` / `Received` 和 JS 常见 AssertionError 文本识别 `expectation_mismatch`，避免真实 JS/TS 测试失败被降级为 generic 建议。

已完成补充：`fix_suggestions` 现在会为每条建议返回 `repair_task`，包含稳定 `id`、失败分类、目标文件和行号、上下文片段、可编辑文件列表、建议复跑命令和断言关注点，方便 Agent 把失败修复变成单个可执行任务。

已完成补充：`run_tests` 新增 `include_fix_suggestions`、`source_code` 和 `test_code` 输入。开启后，如果测试失败，返回结果会内联 `fix_suggestions[]` 和 `repair_task`，Agent 可以直接进入修复任务，减少一次 `parse_results -> fix_suggestions` 往返。

已完成补充：新增 `tools/testdata/golden/repair_tasks.golden` 和 repair task golden test，固定 `id`、`target_file`、`context_snippet`、`editable_files`、`suggested_commands` 和 `assertion_focus` 等字段，同时覆盖有测试名的断言失败和无测试名的越界失败降级路径。

## 第十七阶段：v0.4.10 发布收敛

1. [x] 更新质量评估，补充 `repair_task` 和 `run_tests.include_fix_suggestions` 对修复闭环质量的影响。
2. [x] 新增 `docs/plan-release-notes-v0.4.10.md`，汇总失败修复闭环增强内容和发布前 checklist。
3. [x] 发布前重新运行 release checklist，确认 workflow、CLI、打包脚本和文档状态一致。
4. [x] 准备并发布 `v0.4.10`，验证 Release 资产、安装脚本和 Homebrew tap。

已完成补充：第十六阶段产出的 repair task 和内联修复建议已经整理进质量评估与 v0.4.10 发布说明草案。下一步进入发布前复验和版本号更新。

已完成补充：v0.4.10 发布前 checklist 已重新跑通，包括脚本语法、workflow YAML 解析、actionlint、`go test ./...`、主服务和 testgen CLI 构建、help 输出、本地 darwin arm64 release asset dry run、sha256 校验和 tarball 内容检查。详细证据记录在 `docs/plan-release-notes-v0.4.10.md`。

已完成补充：`v0.4.10` 已发布，tag 指向 `4816c291bdadf320f356218eac7f35b48ebec094`。Release Artifacts run `28845299697` 已通过，Release 资产覆盖 Linux amd64、Linux arm64、macOS arm64、Windows amd64 和 Windows arm64，Homebrew tap 已升级到 `0.4.10` 并通过 `brew test`。发布后安装验证暴露 GitHub release 下载偶发卡住，已为 `scripts/install.sh` 增加 curl/wget 重试和超时控制。

## 第十八阶段：安装链路回归测试

1. [x] 为 `scripts/install.sh` 增加离线回归测试，覆盖 Windows zip 安装、单资产 `.sha256` fallback 和下载重试/超时参数。
2. [x] 覆盖 `go install` fallback，并验证 `testgen` 会重命名为 `testloop-testgen`。
3. [x] 将安装脚本回归测试接入 CI，避免后续发布改动只靠人工验证。
4. [x] 继续把 release 资产清单校验沉淀到维护脚本。
5. [x] 继续评估多平台安装 dry run 是否适合做成手动 workflow，避免普通 CI 依赖 release 下载网络。

已完成补充：新增 `test/install_script_test.sh`，通过 fake `curl`、`unzip` 和 `go` 完全离线验证安装脚本关键路径。CI 的 `Run tests` 步骤现在会同时运行 `go test ./...` 和安装脚本回归测试。

已完成补充：新增 `scripts/verify-release-assets.sh` 和 `test/release_assets_test.sh`，可用 `gh release view` 校验指定 tag 是否包含五平台二进制资产和对应 `.sha256`，并用 fake `gh` 固定完整清单与缺失清单两种路径。`v0.4.10` 真实 release 已通过 10 个必需资产校验。

已完成补充：新增 `Post-Release Verify` 手动 workflow。发布后输入 tag 后，会先校验 release 资产清单，再对 Linux amd64、Linux arm64、macOS arm64、Windows amd64 和 Windows arm64 执行安装脚本 dry run；Linux/macOS 会进一步运行已安装二进制的 `--help`，Windows 在 Linux runner 上完成 zip 下载、`.sha256` 校验和 `.exe` 安装检查。已用 `v0.4.10` 触发 run `28851402908`，五个平台 dry run 全部通过。

## 第十九阶段：失败修复闭环契约固定

1. [x] 增加真实 `run_tests` 失败用例，开启 `include_fix_suggestions` 并验证结果中直接包含 `repair_task`。
2. [x] 用 golden 固定 `failures[]`、`fix_suggestions[]` 和 `repair_task` 的稳定 JSON 契约。
3. [x] 同步 Agent 工作流文档，说明内联 repair task 已有回归测试保护。
4. [x] 补齐 pytest 的内联 repair task 端到端 fixture，避免 Python 失败闭环退化。
5. [x] 补齐 Jest 的内联 repair task 端到端 fixture，避免主流 JS 断言失败闭环退化。
6. [x] 补齐 Vitest 的内联 repair task 端到端 fixture，避免 TS/Vite 测试失败闭环退化。
7. [x] 补齐 Mocha 的内联 repair task 端到端 fixture，避免其余 JS 测试失败闭环退化。

已完成补充：新增 `TestHandleRunTestsRepairTaskGolden` 和 `tools/testdata/golden/run_tests_repair_task.golden`。测试会运行临时 Go 项目的真实失败测试，调用 `run_tests include_fix_suggestions=true`，并把临时路径规范成 `fixture/...` 后固定 `repair_task` 的目标文件、上下文行、可编辑文件、建议命令和断言关注点。

已完成补充：新增 `TestHandleRunTestsPytestRepairTaskGolden` 和 `tools/testdata/golden/run_tests_pytest_repair_task.golden`。测试通过 fake `python3 -m pytest` 输出真实 pytest 失败日志，仍走完整 `run_tests -> pytest parser -> include_fix_suggestions -> repair_task` 路径，并固定除零异常场景下的源码定位、`pytest fixture/test_calc.py` 复跑命令和断言关注点。

已完成补充：新增 `TestHandleRunTestsJestRepairTaskGolden` 和 `tools/testdata/golden/run_tests_jest_repair_task.golden`。测试通过 fake `npx jest` 输出真实 Jest 失败日志，仍走完整 `run_tests -> Jest parser -> include_fix_suggestions -> repair_task` 路径，并固定 expected/received 断言失败、测试文件定位、`npx jest fixture/sum.test.js` 复跑命令和断言关注点。

已完成补充：新增 `TestHandleRunTestsVitestRepairTaskGolden` 和 `tools/testdata/golden/run_tests_vitest_repair_task.golden`。测试通过 fake `npx vitest run` 输出真实 Vitest 失败日志，仍走完整 `run_tests -> Vitest parser -> include_fix_suggestions -> repair_task` 路径，并固定 TS 子目录测试文件定位、`npx vitest run fixture/src/sum.test.ts` 复跑命令和断言关注点。

已完成补充：新增 `TestHandleRunTestsMochaRepairTaskGolden` 和 `tools/testdata/golden/run_tests_mocha_repair_task.golden`。测试通过 fake `npx mocha` 输出真实 Mocha 失败日志，仍走完整 `run_tests -> Mocha parser -> include_fix_suggestions -> repair_task` 路径，并固定 `test/calc.test.js` 断言失败定位、`npx mocha fixture/test/calc.test.js` 复跑命令和断言关注点。

## 第二十阶段：解析字段质量补齐

1. [x] 增强 Mocha parser，从 Chai 风格 `AssertionError: expected actual to equal expected` 中提取结构化 `expected` / `received`。
2. [x] 同步 parser fixture 测试，确保 Mocha 与 Jest/Vitest 的断言字段质量一致。
3. [x] 同步 `run_tests` Mocha repair golden，固定端到端输出中的 `expected` / `received` 字段。

已完成补充：Mocha 解析现在会把 `AssertionError: expected 4 to equal 3` 解析为 `received=4`、`expected=3`，并覆盖 `equal`、`deep equal`、`be` 三种常见 Chai 断言文本。对应 parser fixture 和 `run_tests` repair task golden 已更新，避免 Mocha 失败回路在结构化断言字段上落后于 Jest/Vitest。

## 第二十一阶段：JS 复跑命令质量提升

1. [x] 在 `run_tests` 内联 `repair_task` 中按 JS 测试框架生成复跑命令。
2. [x] Jest 使用 `npx jest <test-file>`，Vitest 使用 `npx vitest run <test-file>`，Mocha 使用 `npx mocha <test-file>`。
3. [x] 同步 JS repair golden，固定新的框架级命令契约。

已完成补充：`run_tests include_fix_suggestions=true` 现在会利用已知 `framework` 覆盖 JS repair task 的 `suggested_commands`，减少代理执行修复任务时对项目 `npm test` 脚本的依赖。独立 `fix_suggestions` 工具仍保留通用扩展名兜底命令，避免在不知道框架时过度猜测。

## 第二十二阶段：自动检测框架下的修复命令闭环

1. [x] 修复 `Framework` 为空时，自动检测出的框架没有传入 `repair_task` 命令生成的问题。
2. [x] 增加 Jest/Vitest/Mocha 自动检测场景回归测试，确保 `run_tests` 通过 `package.json` 识别框架后仍生成框架级复跑命令。
3. [x] 保持显式 `framework` 场景的 repair golden 不变，避免重复扩大契约快照。

已完成补充：`HandleRunTests` 现在会把 `detector.DetectFramework(path)` 得到的有效框架回写到本次 `runTestsInput`，因此 `include_fix_suggestions=true` 在自动检测 Jest/Vitest/Mocha 时，也能生成 `npx jest`、`npx vitest run`、`npx mocha` 这类精确复跑命令。新增 `TestHandleRunTestsAutoDetectedJSRepairCommands` 覆盖三种 JS 框架的自动检测闭环。

## 第二十三阶段：JS 测试执行目录与路径契约

1. [x] 将 Jest/Vitest/Mocha 的执行目录从测试文件所在目录改为向上查找到的 `package.json` 所在目录。
2. [x] 将传给 JS 测试框架的测试文件参数规范为相对项目根的路径，避免 `cmd.Dir` 改为项目根后仍携带重复根目录片段。
3. [x] 增加 fake `npx` 记录 `$PWD` 和参数的回归测试，固定子目录测试文件的执行契约。

已完成补充：JS `run_tests` 现在通过 `findProjectRoot(path, "package.json")` 在项目根执行 `npx`，并把 `src/sum.test.ts`、`test/calc.test.js` 这类子目录测试文件以相对项目根路径传给 Jest/Vitest/Mocha。新增 `TestHandleRunTestsJSUsesPackageRootAndRelativePath` 固定执行目录和参数顺序，避免未来路径归一化改动导致 JS 测试跑不到指定文件。

## 第二十四阶段：pytest 执行目录与路径契约

1. [x] 将 pytest 的执行目录从测试文件所在目录改为向上查找到的 Python 项目根。
2. [x] 支持通过 `pyproject.toml`、`setup.py`、`pytest.ini`、`tox.ini`、`setup.cfg` 定位 pytest 项目根。
3. [x] 将传给 pytest 的测试文件参数规范为相对项目根路径，并用 fake `python3` 固定 `$PWD` 和参数。

已完成补充：pytest `run_tests` 现在通过 Python 项目配置文件定位执行根，并把 `tests/test_calc.py` 这类子目录测试文件以相对项目根路径传给 `python3 -m pytest`。新增 `TestHandleRunTestsPytestUsesProjectRootAndRelativePath` 和 `TestPytestArgsUsesRelativePath` 固定该契约，避免子目录测试在错误目录下执行。

## 第二十五阶段：覆盖率模式下的执行目录契约

1. [x] 固定 Jest/Vitest/Mocha 在 `coverage=true` 时仍从 `package.json` 所在目录执行。
2. [x] 固定 pytest 在 `coverage=true` 时仍从 Python 项目根执行。
3. [x] 固定覆盖率参数与相对测试路径的参数顺序，避免覆盖率模式和普通测试模式出现路径行为分叉。

已完成补充：新增 coverage 模式下的 fake `npx` / fake `python3` 执行记录测试，确认 Jest/Vitest/Mocha 会在项目根收到 `--coverage` 与相对测试路径，pytest 会在 Python 项目根收到 `--cov` 与相对测试路径。这样普通测试与覆盖率测试共享同一套项目根和路径契约。

## 第二十六阶段：coverage task 建议命令一致性

1. [x] 将 coverage task 的 Jest/Vitest 命令统一为 `npx jest <file>` / `npx vitest run <file>` 并规范 slash。
2. [x] 将 Mocha coverage task 从宽泛 `npx mocha` 改为 `npx mocha <file>`，避免无法定位目标测试文件。
3. [x] 将 pytest coverage task 从 `pytest <file>` 对齐为 `python3 -m pytest <file>`，与 `run_tests` 执行入口保持一致。

已完成补充：新增 `TestCoverageTaskCommandMatchesRunTestsFrameworkCommands`，固定 Go/Jest/Vitest/Mocha/pytest/Rust/Java 的 coverage task 复跑命令。coverage task 现在不会再让 Mocha 丢失目标文件，也不会让 pytest 使用和 `run_tests` 不一致的入口命令。

## 第二十七阶段：coverage task 测试文件路径建议

1. [x] 将 pytest coverage task 的源文件建议路径从 `src/tests/test_*.py` 调整为更常见的项目根 `tests/.../test_*.py`。
2. [x] 保留已有 `test_*.py` 文件路径，不再重复包装。
3. [x] 增加 Jest/Vitest/Mocha/pytest 的 `test_file` 推荐路径契约测试，避免后续路径策略漂移。

已完成补充：pytest coverage task 现在会把 `src/service.py` 推荐为 `tests/test_service.py`，把 `src/billing/invoice.py` 推荐为 `tests/billing/test_invoice.py`，而已有 `tests/test_service.py` 会原样保留。新增 `TestCoverageTaskTestFileRecommendations` 同时固定 JS/TS/Mocha 的既有推荐文件策略。

## 第二十八阶段：generate_tests 消费 coverage task 契约

1. [x] 补齐 Python coverage task 的 handler 级端到端测试，固定 `coverage_task.test_file` 优先写入行为。
2. [x] 补齐 JavaScript coverage task 的 handler 级端到端测试，固定生成文件、预览和返回 JSON 契约。
3. [x] 统一断言 `coverage_task` 同时出现在顶层输出和 `context.coverage_task`，避免代理链路丢失任务上下文。

已完成补充：`generate_tests` 现在已有 Go/Python/JavaScript 三条 handler 级 coverage task 契约保护。测试会从真实临时源码触发静态生成，确认工具按 `coverage_task.test_file` 创建目标文件，并在输出 preview、顶层 `coverage_task` 与 `context.coverage_task` 中保留任务信息。

## 第二十九阶段：补齐 Java/Rust coverage task 生成闭环

1. [x] 补齐 Java coverage task 的 handler 级端到端测试，固定 JUnit 测试文件写入和任务上下文回传。
2. [x] 补齐 Rust coverage task 的 handler 级端到端测试，固定 Cargo 测试文件写入和任务上下文回传。
3. [x] 将 Go/Python/JavaScript/Java/Rust 的 `generate_tests` coverage task 工具边界统一到同一套断言 helper。

已完成补充：`generate_tests` 的 coverage task handler 级契约已覆盖当前主要静态生成语言。Java/Rust 测试会分别从临时源码触发 JUnit/Cargo 静态生成，确认 `coverage_task.test_file` 被优先使用，生成内容进入 preview，并且顶层输出和 generation context 都保留原始 coverage task。

## 第三十阶段：coverage task 跨工具闭环 golden

1. [x] 新增 `parse_coverage -> generate_tests` 的 handler 级 golden，用真实临时 Go 项目固定跨工具 JSON 契约。
2. [x] 固定 `parse_coverage.test_tasks[]` 可直接作为 `generate_tests.coverage_task` 输入。
3. [x] 固定生成输出中的 `coverage_task`、`context.coverage_task` 和落盘测试文件内容，避免覆盖率任务在工具串联时丢失上下文。

已完成补充：新增 `TestHandleParseCoverageGenerateTestsLoopGolden` 和 `tools/testdata/golden/coverage_task_generate_loop.golden`。测试会解析 Go coverprofile，取第一个 `test_tasks[]` 任务直接调用 `generate_tests`，并固定任务字段、生成输出、上下文回传和最终测试文件内容，确保 “覆盖率缺口 -> 测试任务 -> 测试草稿” 可以被 Agent 稳定串联。

## 第三十一阶段：pytest coverage task 跨工具闭环 golden

1. [x] 新增 pytest 的 `parse_coverage -> generate_tests` handler 级 golden，覆盖 `coverage.py json` 输入。
2. [x] 固定 `src/service.py -> tests/test_service.py` 的推荐测试文件路径在跨工具链路中保持稳定。
3. [x] 固定 pytest task 进入生成器后的顶层输出、`context.coverage_task` 和落盘测试文件内容。

已完成补充：新增 `TestHandleParseCoverageGenerateTestsPytestLoopGolden` 和 `tools/testdata/golden/coverage_task_generate_pytest_loop.golden`。测试确认 pytest 的 coverage task 可以直接驱动 `generate_tests`，并记录当前 pytest 任务仍停留在文件级目标 `service.py`，后续应补源码映射能力，把覆盖率缺口提升到具体函数和未覆盖行语义。

## 第三十二阶段：pytest coverage task 源码映射

1. [x] 为 pytest coverage.py JSON 增加 Python 源码映射，支持用 tree-sitter 定位函数和类方法范围，并保留正则兜底。
2. [x] 将 pytest 覆盖率建议从文件级 `service.py` 提升到函数级 `status`，补齐 `kind`、`gap_type`、`missing_branches`、`uncovered_lines` 和 `suggested_inputs`。
3. [x] 更新 pytest 跨工具 golden，固定 `parse_coverage -> generate_tests` 生成函数级测试名和分支输入。

已完成补充：pytest coverage task 现在能读取同目录源码，把 `src/service.py` 的未覆盖分支映射到 `status` 函数，并生成 `test_status_covers_gap`。对应生成测试会使用覆盖分支的输入 `status("active")`，断言 `"ok"`，不再停留在文件级泛化任务。

## 第三十三阶段：JS coverage task 源码映射

1. [x] 为 Istanbul coverage JSON 增加 JavaScript/TypeScript 源码映射，支持函数声明、箭头函数、函数表达式和类方法。
2. [x] 将 Jest/Vitest/Mocha 覆盖率建议从文件级提升到函数级，补齐分支缺口、未覆盖行和建议输入。
3. [x] 新增 Jest 的 `parse_coverage -> generate_tests` handler 级 golden，固定函数级任务能直接生成命中分支的测试。

已完成补充：Jest/Vitest/Mocha 的 Istanbul coverage task 现在能读取源码，把 `src/sum.js` 的未覆盖分支映射到 `add` 函数，并生成 `covers add coverage gap`。对应生成测试会使用 `add(0, 2)` 覆盖 `a === 0` 分支，断言返回值 `2`。

## 第三十四阶段：Vitest/TypeScript coverage task 闭环

1. [x] 补齐 TypeScript 源码下的 Vitest coverage task 源码映射测试，固定 `.ts` 文件能定位到具体函数。
2. [x] 固定 Vitest task 的 `test_file` 为 `src/sum.test.ts`，命令为 `npx vitest run src/sum.ts`。
3. [x] 新增 Vitest 的 `parse_coverage -> generate_tests` handler 级 golden，确认 `.ts` coverage task 能直接生成命中分支的测试草稿。

已完成补充：Vitest/TypeScript coverage task 现在有专门回归测试保护。`src/sum.ts` 的未覆盖分支会映射到 `add` 函数，生成 `covers add coverage gap`，并落盘到 `src/sum.test.ts`；生成测试使用 `add(0, 2)` 覆盖 `a === 0` 分支。

## 第三十五阶段：Mocha/nyc coverage task 闭环

1. [x] 补齐 Mocha/nyc Istanbul coverage task 的 CommonJS 源码映射测试，固定 `.js` 文件能定位到具体函数。
2. [x] 固定 Mocha task 的 `test_file` 为 `lib/calc.spec.js`，命令为 `npx mocha lib/calc.js`。
3. [x] 新增 Mocha 的 `parse_coverage -> generate_tests` handler 级 golden，并修正 Mocha coverage task 生成内容为 Chai 断言风格。

已完成补充：Mocha/nyc coverage task 现在有 CommonJS 闭环回归测试保护。`lib/calc.js` 的未覆盖分支会映射到 `divide` 函数，生成 `covers divide coverage gap`，并落盘到 `lib/calc.spec.js`；生成测试会引入 `chai` 的 `expect`，使用 `expect(result).to.equal((0))`，不再输出 Jest 专用的 `toBe` 断言。

## 第三十六阶段：Mocha error-path 断言契约

1. [x] 将 Mocha coverage task 的同步错误路径断言固定为 Chai `expect(() => call()).to.throw()`。
2. [x] 将 Mocha coverage task 的异步错误路径断言改为原生 `try/catch` 捕获变量 + Chai `expect(caughtError).to.exist`，避免生成 Jest 专用的 `rejects.toThrow()`。
3. [x] 增加 CommonJS Mocha 同步/异步 error-path 生成回归测试，固定导入 `chai` 与断言风格。

已完成补充：Mocha coverage task 在 `gap_type=error_path` 时已不再混入 Jest matcher。同步异常生成 `expect(() => divide(1, 0)).to.throw()`，异步异常通过 `let caughtError`、`try/catch` 和 `expect(caughtError).to.exist` 校验拒绝，无需额外依赖 `chai-as-promised`，也不会把未拒绝的 Promise 误判为通过。

## 第三十七阶段：Mocha class method error-path 入口契约

1. [x] 补齐 `GenerateJestTestsForCoverageTask` 入口级的 Mocha class 同步错误路径测试，固定 `Widget.method` 目标过滤和 Chai `to.throw()` 断言。
2. [x] 补齐入口级的 Mocha class 异步错误路径测试，固定 `try/catch` 捕获变量断言。
3. [x] 固定 class method coverage task 不生成同类中的非目标方法，并持续禁止输出 Jest 专用 `toThrow()` / `rejects.toThrow()`。

已完成补充：Mocha coverage task 的类方法错误路径现在不只由内部 helper 测试保护，而是通过真实 JS 源码解析、`Widget.save` / `Widget.load` 目标过滤和测试代码生成入口共同验证。同步方法会生成 `expect(() => instance.save(null)).to.throw()`，异步方法会生成 `await instance.load(undefined)` 后校验 `caughtError` 存在。

## 第三十八阶段：Mocha ES module error-path 入口契约

1. [x] 补齐 ES module 函数级 Mocha error-path 入口测试，固定 `import { expect } from 'chai'` 和源模块命名导入。
2. [x] 补齐 ES module class method 异步 error-path 入口测试，固定 class 目标过滤和 `caughtError` 断言。
3. [x] 固定 ESM Mocha coverage task 不生成 CommonJS `require(...)`，也不生成 Jest 专用 `toThrow()` / `rejects.toThrow()`。

已完成补充：Mocha coverage task 现在同时覆盖 CommonJS 与 ES module 生成路径。ESM 源码会生成 `import { expect } from 'chai';`，函数任务生成 `import { divide } from './calc';` 和 Chai `to.throw()`，类方法任务生成 `import { Widget } from './widget';`、实例化目标类并用 `caughtError` 校验异步拒绝。

## 第三十九阶段：Mocha ES module 非异常断言契约

1. [x] 补齐 ES module 函数级 Mocha `return_path` 入口测试，固定 Chai `expect(result).to.equal(...)`。
2. [x] 补齐 ES module class method `branch` 入口测试，固定分支输入、目标过滤和 Chai 等值断言。
3. [x] 固定 ESM Mocha 非异常 coverage task 不生成 CommonJS `require(...)`，也不生成 Jest 专用 `toBe(...)` / `toThrow()`。

已完成补充：Mocha ESM coverage task 现在不仅覆盖错误路径，也覆盖返回路径和分支路径。函数任务会生成 `import { add } from './calc';`、`const result = add(0, 2);` 和 `expect(result).to.equal((0 + 2));`；类方法分支任务会生成 `instance.load('short', 1)` 并用 Chai 校验返回值。

## 第四十阶段：Mocha TypeScript 非异常断言契约

1. [x] 补齐 TypeScript 函数级 Mocha `return_path` 入口测试，固定类型标注源码下的参数解析和 Chai 等值断言。
2. [x] 补齐 TypeScript class method `branch` 入口测试，固定类型标注方法的目标过滤、分支输入和返回值断言。
3. [x] 固定 TypeScript Mocha coverage task 继续使用 ESM import，不生成 CommonJS `require(...)` 或 Jest 专用 matcher。

已完成补充：Mocha coverage task 在 `.ts` 源码下现在有函数和类方法非异常路径保护。生成器会从带类型标注的参数中提取 `a`、`b`、`mode`、`count`，生成 `import { add } from './calc';` / `import { Widget } from './widget';`，并使用 Chai `expect(result).to.equal(...)` 断言返回值。

## 第四十一阶段：Mocha TypeScript error-path 入口契约

1. [x] 补齐 TypeScript 函数级 Mocha `error_path` 入口测试，固定带类型标注参数下的 Chai `to.throw()` 断言。
2. [x] 补齐 TypeScript class method 异步 `error_path` 入口测试，固定可选参数解析、目标过滤和 `caughtError` 断言。
3. [x] 固定 TypeScript Mocha 异常路径继续使用 ESM import，不生成 CommonJS `require(...)` 或 Jest 专用 `toThrow()` / `rejects.toThrow()`。

已完成补充：Mocha coverage task 在 `.ts` 源码下的异常路径现在有入口级保护。函数任务会生成 `import { divide } from './calc';` 和 `expect(() => divide(1, 0)).to.throw()`；类方法任务会生成 `import { Widget } from './widget';`、`await instance.load(undefined)`，并用 `expect(caughtError).to.exist` 校验异步拒绝。

## 第四十二阶段：Mocha 生成测试矩阵维护性收拢

1. [x] 抽取 `assertGeneratedJS` 测试 helper，统一正向生成片段和 forbidden matcher 断言。
2. [x] 将 CommonJS、ESM、TypeScript Mocha coverage task 入口测试中的重复断言循环替换为 helper 调用。
3. [x] 保持既有断言内容不变，避免维护性重构改变生成契约。

已完成补充：Mocha 生成测试矩阵现在通过统一 helper 表达“必须生成哪些片段”和“禁止生成哪些 Jest/CommonJS 片段”。这降低了后续继续补 Vitest/Jest 或更多 TypeScript 场景时的重复成本，也让 forbidden matcher 契约更集中。

## 第四十三阶段：Mocha 生成矩阵 table-driven 化

1. [x] 将 CommonJS、ESM、TypeScript 的 Mocha coverage task 入口测试合并为 table-driven 测试。
2. [x] 保留同步/异步 error-path、return_path、branch、function、class method 的 12 个既有矩阵 case。
3. [x] 统一每个 case 的源码文件名、源码内容、coverage task、正向片段和 forbidden 片段表达，降低后续扩展成本。

已完成补充：Mocha 生成矩阵现在集中在 `TestGenerateMochaCoverageTaskUsesChaiMatrixAssertions` 中，每个 case 明确声明输入源码、coverage task 和输出契约。这样新增 Vitest/Jest 或更多 JS/TS case 时，不再需要复制整段测试函数。

## 第四十四阶段：Jest/Vitest 生成矩阵收拢

1. [x] 将 Jest coverage task 入口测试改为 table-driven 矩阵，复用 `assertGeneratedJS`。
2. [x] 补齐 Vitest TypeScript `branch` 入口 case，固定 ESM import、分支输入和 Jest/Vitest matcher 风格。
3. [x] 补齐 Vitest TypeScript 异步 `error_path` 入口 case，固定 `rejects.toThrow()`，避免误用 Chai 或 Mocha 的 `caughtError` 模式。

已完成补充：Jest/Vitest coverage task 生成入口现在由 `TestGenerateJestVitestCoverageTaskMatrixAssertions` 统一表达。矩阵覆盖 CommonJS Jest return_path、TypeScript Vitest branch、TypeScript Vitest async error-path，明确禁止 Chai `to.equal(...)`、`require('chai')` 和非目标函数输出。

## 第四十五阶段：Jest/Vitest class method 生成矩阵

1. [x] 补齐 Jest CommonJS class method `branch` 入口 case，固定目标过滤、实例化和 `toBe(...)` 断言。
2. [x] 补齐 Vitest TypeScript class method 异步 `error_path` 入口 case，固定 ESM import、实例化和 `rejects.toThrow()`。
3. [x] 持续禁止非 Mocha 框架生成 Chai `to.equal(...)`、`require('chai')` 或 Mocha 的 `caughtError` 模式。

已完成补充：Jest/Vitest 生成矩阵现在覆盖 class method 路径。Jest CommonJS 会生成 `const { Widget } = require('./widget');`、`instance.load('short', 1)` 和 `expect(result).toBe((1))`；Vitest TypeScript 会生成 `import { Widget } from './widget';`、`await expect(instance.load(undefined)).rejects.toThrow()`，和 Mocha class 生成路径形成对照保护。

## 第四十六阶段：JavaScript coverage task 入口命名澄清

1. [x] 新增 `GenerateJavaScriptTestsForCoverageTask` 作为 JS/TS coverage task 的语义化入口。
2. [x] 将内部 provider 分发、golden 和主要生成器测试切到新入口，避免继续把 Mocha/Vitest 误称为 Jest 生成路径。
3. [x] 保留 `GenerateJestTestsForCoverageTask` 兼容 wrapper，并补充 wrapper 与新入口输出一致的回归断言。

已完成补充：JS/TS coverage task 的入口命名现在能表达它服务 Jest、Vitest、Mocha 三种框架，而旧 `GenerateJestTestsForCoverageTask` 仍可用，不破坏既有调用方。内部 `generateTestsForCoverageTask` 已改为调用新的 JavaScript 入口。

## 第四十七阶段：JS/TS 生成语义和文档收口

1. [x] 将 coverage task 私有生成 helper 从 `genJest*ForCoverageTask` 改为 `genJS*ForCoverageTask`，避免内部实现继续暗示只支持 Jest。
2. [x] 将 coverage task 断言风格选择 helper 改为 `jsAssertionStyleForTask`，表达它服务 Jest/Vitest/Mocha 任务。
3. [x] 同步 README 中 `generate_tests`、能力概览和项目结构说明，明确 JS/TS 默认 Jest 风格，coverage task 会按 Jest/Vitest/Mocha 输出对应断言风格。

已完成补充：JS/TS 生成链路现在从公开入口、内部 helper 到用户文档都统一成“JavaScript/TypeScript coverage task”语义。带 task 时会根据 `framework` 选择 Jest/Vitest matcher 或 Mocha/Chai 断言；普通无 task 生成在后续阶段已接入框架自动检测，无法识别时再回退到 Jest 风格。

## 第四十八阶段：普通 JS/TS 生成框架参数生效

1. [x] 新增 `GenerateJavaScriptTestsWithFramework`，让无 `coverage_task` 的 JS/TS 生成也能按 `framework=jest/vitest/mocha` 选择断言风格。
2. [x] 将 `tools.generate_tests.framework` 传入 `GenerateTestsOptions`，并同步到 `TestGenerationContext.Framework`，避免 MCP 工具和 LLM provider 上下文继续固定显示 Jest。
3. [x] 补充 generator/provider/handler 测试，固定 Vitest 普通路径使用 Jest/Vitest matcher，Mocha 普通路径生成 Chai import 和 `to.equal(...)` 断言。

已完成补充：普通 `generate_tests` 现在不再只在 coverage task 场景下区分 JS 框架。调用方传 `framework: "mocha"` 会生成 `const { expect } = require('chai')` 和 Chai 断言；传 `framework: "vitest"` 会保持 Jest/Vitest matcher 风格，同时返回 context 中的 `framework` 也会显示为 `vitest` 或 `mocha`。

## 第四十九阶段：普通 JS/TS 生成框架自动检测

1. [x] 复用 `internal/detector.DetectFramework`，让无 `coverage_task` 且未显式传 `framework` 的 JS/TS 生成自动读取同目录或上级 `package.json`。
2. [x] 将自动检测到的 Jest/Vitest/Mocha 同步到静态生成输出和 `TestGenerationContext.Framework`，避免 preview 与上下文不一致。
3. [x] 补充 provider/handler 回归测试，固定 Mocha 自动生成 Chai 断言、Vitest 自动生成 Jest/Vitest matcher，并确保显式 framework 与 coverage task 路径不回退。

已完成补充：普通 `generate_tests` 现在和 `run_tests` 一样会自动识别 JS/TS 项目的测试框架。调用方只传 `file_path` 时，如果 `package.json` 的 `scripts.test` 是 `vitest run`，返回 context 会标记 `vitest`，生成测试也会使用 Vitest 可执行的 matcher；如果检测到 Mocha，则会生成 Chai import 和 Chai 断言。

## 第五十阶段：普通 JS/TS 生成到执行闭环契约

1. [x] 明确普通 JS/TS 生成文件继续使用源文件同目录的 `*.test.*` 策略，不为 Mocha 强制迁移到 `test/` 目录。
2. [x] 补充 `generate_tests -> run_tests` handler 级闭环测试，固定 Vitest 生成 `src/sum.test.ts` 后会以 `vitest run --verbose src/sum.test.ts` 执行。
3. [x] 补充 Mocha 闭环测试，固定 `lib/calc.js` 生成 `lib/calc.test.js` 后会从 `package.json` 根目录以 `mocha --reporter spec lib/calc.test.js` 执行。

已完成补充：普通 JS/TS 生成路径现在有端到端契约保护。生成器保持“测试文件靠近源文件”的低惊扰策略，`run_tests` 负责在执行时定位 package root 并传相对路径；这让 Jest/Vitest/Mocha 都能跑同一个生成路径策略，同时不破坏 coverage task 推荐的 `test_file` 覆盖能力。

## 第五十一阶段：普通 JS/TS 生成 golden 快照

1. [x] 为普通 Vitest 生成新增 golden 快照，固定它和 Jest 一样使用 `expect(...).toBe(...)` matcher 风格。
2. [x] 为普通 Mocha 生成新增 golden 快照，固定 Chai `require('chai')` 导入和 `expect(...).to.equal(...)` 断言风格。
3. [x] 复用同一份 `js_branch.js` 源码，确保 Jest/Vitest/Mocha 的普通生成差异集中在框架断言和 import 契约上。

已完成补充：普通 JS/TS 生成现在不仅有 handler 闭环测试，也有 generator 级 golden 快照。后续修改 JS parser、断言 helper 或 framework 分发时，如果影响 Jest/Vitest/Mocha 代表性输出，会直接触发快照差异。

## 第五十二阶段：普通 JS/TS ESM 生成 golden 快照

1. [x] 新增 TypeScript ESM 源码快照输入，覆盖 `export function`、类型标注参数和简单分支返回。
2. [x] 为 Jest/Vitest ESM 普通生成新增 golden，固定 `import { target } from './module'` 与 `expect(...).toBe(...)` 契约。
3. [x] 为 Mocha ESM 普通生成新增 golden，固定 `import { expect } from 'chai'`、源模块命名 import 和 Chai `to.equal(...)` 断言。

已完成补充：普通 JS/TS 生成的 CommonJS 与 ESM 代表性输出现在都有 golden 保护。后续无论是改 tree-sitter 解析、TypeScript 参数处理，还是调整框架断言风格，都能更早发现 Jest/Vitest/Mocha 在 ESM 路径上的契约漂移。

## 第五十三阶段：普通 JS/TS class 与 async error-path golden

1. [x] 新增普通 JS 源码快照输入，覆盖顶层 async 函数、class constructor、同步分支方法和异步抛错方法。
2. [x] 为 Jest/Vitest 普通生成新增 golden，固定 `rejects.toThrow()`、`toBeInstanceOf`、对象返回断言和 class method 分支断言。
3. [x] 为 Mocha 普通生成新增 golden，固定 Chai `instanceOf`、`caughtError` 异步错误断言和 class method 的 `to.equal(...)` 输出。

已完成补充：普通 JS/TS 生成的核心复杂路径现在有快照保护。顶层 async error path 与 class async method 的错误路径输出已经在后续阶段去重，当前 golden 固定的是“优先保留边界错误用例”的契约。

## 第五十四阶段：普通 JS/TS error-path 去重

1. [x] 在普通 JS/TS 生成中识别 generic invalid-input 调用是否已经被 `null` / `undefined` 边界错误用例覆盖。
2. [x] 当调用参数完全相同时，保留语义更明确的 `should handle param = undefined/null` 用例，省略重复的 `should throw on invalid input`。
3. [x] 更新 class/async/error-path golden，并补充 focused test 固定顶层 async 函数和 class async 方法的去重行为。

已完成补充：普通 JS/TS error-path 输出现在更精简。对于 `fetchData(undefined)`、`instance.save(undefined)` 这类边界错误路径，不再同时生成两个完全相同调用的测试，减少 Agent 后续需要清理的重复草稿。

## 第五十五阶段：普通 Vitest ESM 显式 API 导入

1. [x] 普通 JS/TS ESM Vitest 生成增加 `import { describe, it, expect } from 'vitest';`，适配未开启 globals 的 Vitest 项目。
2. [x] 保持 CommonJS Vitest 普通生成不引入 `require('vitest')`，避免改变现有 CJS runner 注入策略。
3. [x] 更新 ESM Vitest golden 和 focused test，固定 Vitest API import、源模块 import 与 Jest/Vitest matcher 风格。

已完成补充：普通 ESM/TS Vitest 输出现在不再隐含 `globals: true`。CommonJS 路径仍保持现有低惊扰输出，后续如需支持 CJS 显式 Vitest API，可以单独评估项目实际运行器兼容性。

## 第五十六阶段：Vitest ESM coverage task 显式 API 导入

1. [x] 将 Vitest ESM API import 扩展到 coverage task 生成路径，避免增量测试草稿依赖 `globals: true`。
2. [x] 补充 Vitest TypeScript coverage task focused 断言，固定 `describe` / `it` / `expect` 显式导入。
3. [x] 新增 Vitest ESM coverage golden，保护 task 注释、建议输入提取、源模块 import 和 matcher 风格。

已完成补充：普通生成和 coverage task 的 ESM/TS Vitest 输出现在保持一致，都会显式导入 Vitest 测试 API；CommonJS 路径仍保持 runner 全局注入假设，避免引入不稳定的 `require('vitest')` 兼容问题。

## 第五十七阶段：Vitest ESM 生成到执行闭环

1. [x] 加强普通 Vitest 生成到执行闭环测试，固定 ESM/TS 输出包含 Vitest API 显式导入。
2. [x] 新增 coverage task 级 Vitest ESM 闭环测试，覆盖 `generate_tests` 写入 `src/sum.test.ts` 后由 `run_tests` 自动检测 Vitest 执行。
3. [x] 固定 `run_tests` 从 `package.json` 根目录运行，并传入相对测试文件路径 `src/sum.test.ts`。

已完成补充：Vitest ESM/TS 现在不仅有 generator 和 golden 保护，也有 handler 级生成到执行闭环保护。这个测试不依赖真实 npm 安装，通过 fake `npx` 固定命令、工作目录和生成内容契约。

## 第五十八阶段：TypeScript NodeNext import 路径策略

1. [x] 为 TS/TSX ESM 源文件增加最近 `tsconfig.json` 检测，识别 `module` 或 `moduleResolution` 为 `node16` / `nodenext` 的项目。
2. [x] Node16/NodeNext 项目生成 `./module.js` 源模块 import，适配 TypeScript 原生 ESM 对显式扩展名的约束。
3. [x] 默认、bundler、无 tsconfig 场景继续保持 extensionless `./module`，避免破坏 Vitest/Vite 常见配置。
4. [x] 补充普通生成和 coverage task 生成测试，覆盖 JSON 与带注释 JSONC 风格的 tsconfig。

已完成补充：JS/TS ESM import 路径现在有最小检测策略。它不会全局强推 `.js` 后缀，只在 Node16/NodeNext TypeScript 项目中切换为 emitted JS extension；这降低了真实项目中 TS typecheck 与 Vitest/Vite resolution 之间的冲突概率。

## 第五十九阶段：JS/TS 对象数组结构断言

1. [x] 对简单 object / array literal 返回值生成结构断言，Jest/Vitest 使用 `toEqual`，Mocha/Chai 使用 `to.deep.equal`。
2. [x] 对 object shorthand 做安全展开，例如 `return { url }` 会结合生成参数得到 `{ url: 'https://example.com' }`。
3. [x] coverage task 的 `suggested_inputs` 会参与对象返回期望值推导，例如分支输入 `mode === 'short'` 和 `count = 1` 会生成对应对象断言。
4. [x] 复杂或不安全表达式仍回退到类型/非空断言，避免把函数调用、`new`、索引访问等高风险表达式写进期望值。

已完成补充：JS/TS 生成结果对简单对象和数组返回不再只断言类型。常见的 API 返回对象、状态摘要对象和数组返回现在能生成更可保留的结构化断言，同时保留安全回退策略。

## 第六十阶段：JS/TS response.json 返回草稿

1. [x] 识别 `return response.json()` 与 `return await response.json()` 这类低风险 API helper 返回路径。
2. [x] 当 `response` / `res` / `resp` 是函数参数时，生成 `{ json: async () => ({ ok: true }) }` 轻量 mock 参数。
3. [x] 对 response JSON 返回生成结构断言，Jest/Vitest 使用 `toEqual({ ok: true })`，Mocha/Chai 复用 `deep.equal` 路径。
4. [x] 补充普通生成与 coverage task focused test，确保 async 返回不再只生成对象类型断言。

已完成补充：JS/TS async API helper 的常见 `response.json()` 返回现在有可执行的 setup 草稿和结构断言。当前只覆盖 response 作为显式参数的场景，暂不自动 mock 全局 `fetch`，避免引入跨测试框架的全局状态清理问题。

## 第六十一阶段：JS/TS 注入式 client 返回草稿

1. [x] 识别 `client.get()`、`api.fetch()`、`http.request()` 这类参数注入的请求调用返回。
2. [x] 对 `client` / `api` / `http` / `fetcher` / `requester` 参数生成局部 mock，提供 `get` / `fetch` / `request` 三个 async 方法。
3. [x] 对注入式 client 返回生成结构断言 `toEqual({ ok: true })`，coverage task 同步复用该推导。
4. [x] 保持不 mock 全局 `fetch`，继续避免跨测试框架的全局状态污染。

已完成补充：JS/TS generator 现在覆盖了更多业务代码常见的依赖注入 API client 模式。对于显式传入的 client/fetcher/http 参数，生成测试会构造局部 mock 参数并断言返回结构；全局请求函数仍留给后续单独评估。

## 第六十二阶段：JS/TS 注入式 client mock 收窄

1. [x] 将注入式 client mock 从固定生成 `get` / `fetch` / `request` 三个方法，改为根据实际 return 调用的方法生成。
2. [x] 普通生成中 `client.get()` 只生成 `get` mock，不再带出未使用的 `fetch` / `request`。
3. [x] coverage task 中 `api.fetch()` 只生成 `fetch` mock，不再带出未使用的 `get` / `request`。
4. [x] 补充 helper 级测试，固定 `get` / `fetch` / `request` 三种方法的参数 mock 收窄行为。

已完成补充：JS/TS 注入式 API client 草稿更干净了。生成器仍保留无上下文时的保守默认值，但只要 return 表达式能定位到实际调用方法，测试参数就只包含用到的 mock 方法。

## 第六十三阶段：JS/TS 注入式 client 调用断言

1. [x] 注入式 client mock 改为命名局部 spy 对象，记录实际调用参数。
2. [x] 普通生成中 `client.get('/users/1')` 会生成 `client.getCalls` 调用断言。
3. [x] coverage task 中 `api.fetch('/users/1')` 会生成 `api.fetchCalls` 调用断言。
4. [x] Jest/Vitest 使用 `toEqual`，Mocha/Chai 使用 `to.deep.equal`，不引入 `vi.fn` / `jest.fn`，降低跨框架耦合。

已完成补充：JS/TS 注入式 API client 草稿现在不仅验证返回结构，也会验证被测函数是否按预期调用了注入依赖。普通生成和 coverage task 都复用同一套局部 spy 输出，避免绑定到特定 runner 的 mock API。

## 第六十四阶段：JS/TS API mock payload 类型化

1. [x] 从 TypeScript 函数、箭头函数和 class method 的返回注解中提取内联返回类型。
2. [x] 对 `Promise<{ id: number; name: string }>` 这类 API helper 返回生成 `{ id: 1, name: 'test' }` mock payload。
3. [x] `response.json()` 和注入式 client mock 共用同一套 payload 推导，断言值与 mock 返回值保持一致。
4. [x] 未知类型、命名接口和无类型注解继续回退 `{ ok: true }`，避免静态生成器凭空展开不可见结构。

已完成补充：JS/TS API helper 草稿现在能利用内联 TypeScript 返回注解生成更像业务对象的测试数据。这个阶段仍保持保守边界，只处理当前源码里可见的内联类型，不跨文件解析接口或类型别名。

## 第六十五阶段：JS/TS 同文件命名类型 payload

1. [x] 解析同文件对象型 `interface User { ... }` 声明，并用于 `Promise<User>` 返回 payload 推导。
2. [x] 解析同文件对象型 `type Profile = { ... }` 声明，并用于注入式 client mock 返回。
3. [x] 支持省略分号、按行书写的 interface/type 字段。
4. [x] 继续限制在同文件可见对象类型，不追踪 import、不展开非对象 alias，避免静态生成器越界猜测。

已完成补充：JS/TS API mock payload 现在能覆盖更常见的 `Promise<User>` / `Promise<Profile>` 写法。生成器会从同文件类型声明中取字段结构，生成 response JSON mock、client mock 和结果断言；跨文件类型仍保守回退。

## 第六十六阶段：JS/TS mock payload 字段值质量

1. [x] 对 string 字段按字段名生成更贴近业务的值，例如 `email`、`url`、`status`、`createdAt`。
2. [x] 对 TypeScript string/boolean/number literal union 取第一个安全字面量作为 mock 值。
3. [x] 保持字段类型兜底策略，未识别字段名仍按 `string` / `number` / `boolean` 等基础类型生成稳定值。
4. [x] 补充命名类型生成测试，固定 `userId`、`email`、`status`、`createdAt`、`avatarUrl` 等代表字段输出。

已完成补充：JS/TS payload 现在不只是“类型正确”，也更像真实业务数据。这个阶段仍是确定性启发式生成，不引入随机值，方便 golden、CI 和 Agent 后续修复流程稳定复现。

## 第六十七阶段：JS/TS coverage task 命名类型回归

1. [x] 补充 coverage task 下 `response.json()` + `Promise<User>` 的命名类型 payload 回归。
2. [x] 补充 coverage task 下注入式 `api.fetch()` + `Promise<User>` 的命名类型 payload 回归。
3. [x] 固定 target filter 后仍只导入目标函数，不生成同文件无关函数测试。
4. [x] 固定 coverage task 输出继续包含结构断言、调用参数断言，并避免回退 `{ ok: true }`。

已完成补充：coverage task 的 JS/TS 增量生成现在有命名类型 payload 的矩阵保护。后续即使调整 parser、target filter 或 task-aware 生成入口，也能及时发现类型上下文丢失。

## 第六十八阶段：JS/TS nullable union payload

1. [x] 对 `User | null`、`null | User` 这类 nullable union 优先选择非 null/undefined 分支生成 mock。
2. [x] 对 `displayName?: string | null` 这类可选 nullable 字段继续生成稳定字符串值，而不是低价值 `null`。
3. [x] 对嵌套命名类型字段，例如 `owner?: User | null`，继续复用同文件类型声明生成对象 payload。
4. [x] 使用顶层 union 拆分，避免误拆对象、数组、泛型内部的 `|`。

已完成补充：JS/TS payload 对 nullable union 的处理更适合测试草稿。生成器会优先构造可断言的非空数据，保留字段结构与业务含义，同时不跨文件追踪类型。

## 第六十九阶段：JS/TS 自引用类型保护

1. [x] 为命名类型 payload 展开增加 `visited` 防护，避免自引用 interface/type 递归展开。
2. [x] `manager?: User | null` 这类字段在再次遇到已访问的 `User` 时回退 `{}`。
3. [x] 数组、泛型、nullable union 分支继续复用同一套递归上下文。
4. [x] 补充普通生成和 helper 级测试，固定自引用类型不会卡死且输出稳定。

已完成补充：JS/TS payload 展开现在对递归类型有明确边界。生成器仍会展开第一层可见业务字段，但遇到循环引用时会停止在 `{}`，避免生成无限嵌套测试草稿。

## 第七十阶段：JS/TS 数组 payload 质量

1. [x] 支持同文件数组 alias，例如 `type Users = User[]` 和 `type MaybeUsers = Array<User | null>`。
2. [x] `Promise<Users>`、`Promise<Array<User | null>>` 会生成 `[{ ... }]` 结构断言，而不是空数组或 `{}`。
3. [x] 数组元素复用命名类型、nullable union 和递归保护规则。
4. [x] 补充普通生成和 coverage task 回归，固定 response JSON 与注入式 client 的数组 payload 输出。

已完成补充：JS/TS API helper 的数组返回现在和对象返回使用同一套类型上下文。生成器能为命名数组 alias 和 nullable 数组元素生成稳定的一元素数组，便于 Agent 后续补充更具体的业务断言。

## 第七十一阶段：JS/TS readonly 数组 payload

1. [x] 支持 `ReadonlyArray<User>`，按普通数组生成一元素 payload。
2. [x] 支持 `readonly User[]`，复用命名类型、nullable union 和递归保护规则。
3. [x] 数组 alias 支持 readonly 写法，例如 `type Users = readonly User[]`。
4. [x] 补充普通生成、coverage task 和 helper 级测试，固定 readonly 数组输出。

已完成补充：JS/TS 数组 payload 现在覆盖真实项目常见的不可变数组类型写法。生成器会把 readonly 数组视为测试数据构造层面的普通数组，继续输出稳定、可断言的一元素数组。

## 第七十二阶段：JS/TS tuple payload

1. [x] 支持 `[User, Meta]` 和 `readonly [User, Meta]` 这类 tuple 返回。
2. [x] tuple 元素复用命名类型、字段名值、nullable union 和递归保护规则。
3. [x] 支持同文件 tuple alias，例如 `type UserTuple = readonly [User, Meta]`。
4. [x] 补充普通生成、coverage task 和 helper 级测试，固定 tuple JSON payload 输出。

已完成补充：JS/TS payload 现在能为 tuple 返回生成确定性的数组结构，例如 `[{ userId: 1 }, { total: 1 }]`。复杂或未知 tuple 元素仍按已有保守规则回退，避免生成不可控草稿。

## 第七十三阶段：JS/TS tuple label/rest 边界

1. [x] 支持 labeled tuple 元素，例如 `[user: User, meta?: Meta]` 会剥离 label 后生成 payload。
2. [x] 支持 rest tuple 元素，例如 `readonly [User, ...Meta[]]` 会生成单个代表性 `Meta` payload。
3. [x] 保持 tuple 元素继续复用命名类型、nullable union、递归保护与字段名值规则。
4. [x] 补充普通生成、coverage task 和 helper 级测试，固定 label/rest tuple 输出。

已完成补充：JS/TS tuple payload 对真实项目常见的 label 与 rest 写法更稳。生成器会把 label 当作类型注解信息剥离，rest 元素生成一个代表性样例，避免退化成嵌套空数组或对象。

## 第七十四阶段：JS/TS utility wrapper payload

1. [x] 支持 `Readonly<T>`，把 wrapper 视为测试数据构造层面的透明类型。
2. [x] 支持 `Required<T>`，继续按内部对象、数组、tuple 或命名类型生成 payload。
3. [x] 支持 `Partial<T>`，保守复用内部类型字段生成稳定样例，不随机省略字段。
4. [x] 支持 wrapper 嵌套，例如 `Partial<Readonly<User>>` 和 `Readonly<Users>`。
5. [x] 补充普通生成、coverage task 和 helper 级测试，固定 utility wrapper 输出。

已完成补充：JS/TS payload 现在能处理真实项目常见的类型工具包装。当前阶段只处理可安全透明化的 wrapper；`Pick<T, K>`、`Omit<T, K>` 这类字段投影类型暂不展开，避免静态生成器在缺少完整类型系统时误猜字段。

## 第七十五阶段：JS/TS Pick 投影 payload

1. [x] 支持同文件对象类型上的 `Pick<T, 'a' | 'b'>`，按选中字段生成子集 payload。
2. [x] 支持 `Readonly<Pick<T, ...>>` 这类透明 wrapper 嵌套。
3. [x] 保持输出字段顺序跟随源对象声明顺序，而不是 key union 顺序。
4. [x] 对 `Pick<T, keyof T>`、非字符串字面量 key、无法解析的源类型保守回退，不猜测字段集合。
5. [x] 补充普通生成、coverage task 和 helper 级测试，固定 Pick 输出和负例边界。

已完成补充：JS/TS payload 现在能覆盖常见的 DTO 子集类型，例如 `Pick<User, 'userId' | 'email'>`。这个阶段仍只在同文件类型上下文内工作，不引入完整 TypeScript 类型系统，也不展开 `keyof`、条件类型或跨文件类型。

## 第七十六阶段：JS/TS Omit 投影 payload

1. [x] 支持同文件对象类型上的 `Omit<T, 'a' | 'b'>`，按排除字段生成剩余 payload。
2. [x] 支持 `Readonly<Omit<T, ...>>` 这类透明 wrapper 嵌套。
3. [x] 未命中的字符串 key 不会误删字段，例如 `Omit<User, 'unknown'>` 仍生成完整对象。
4. [x] 对 `Omit<T, keyof T>`、非字符串字面量 key、无法解析的源类型保守回退，不展开不确定字段集合。
5. [x] 支持所有字段被 omit 后生成 `{}`，避免回退到无关的 `{ ok: true }`。
6. [x] 补充普通生成、coverage task 和 helper 级测试，固定 Omit 输出和负例边界。

已完成补充：JS/TS payload 现在覆盖 DTO 子集的两类常见投影：`Pick` 取字段与 `Omit` 排除字段。两者都只在同文件可见对象类型上工作，保持静态生成器的可解释边界。

## 第七十七阶段：JS/TS Record payload

1. [x] 支持 `Record<string, T>`，使用稳定的 `key` 字段生成代表性对象 payload。
2. [x] 支持 `Record<'a' | 'b', T>`，按字符串字面量 key union 的声明顺序生成对象 payload。
3. [x] Record value 继续复用命名类型、字段名值、nullable union、递归保护和投影类型规则。
4. [x] 支持同文件 Record alias，例如 `type UserMap = Record<string, User>`。
5. [x] 对 `Record<number, T>`、复杂 key、非字符串字面量 key 保守回退，不猜测对象键。
6. [x] 补充普通生成、coverage task 和 helper 级测试，固定 Record 输出和负例边界。

已完成补充：JS/TS payload 现在能覆盖字典型 DTO 返回，例如 `Record<string, User>` 和 `Record<'primary' | 'secondary', User>`。Record 仍只生成少量代表性 key，保持测试草稿稳定可读。

## 第七十八阶段：JS/TS 交叉类型 payload

1. [x] 支持同文件对象类型交叉，例如 `User & AuditFields`，按分支顺序合并字段 payload。
2. [x] 支持内联对象交叉，例如 `{ id: number } & { email: string }`。
3. [x] 支持交叉类型 alias，例如 `type AuditedUser = User & AuditFields`。
4. [x] 字段值中的交叉类型继续复用命名类型、nullable union、递归保护和字段名值规则。
5. [x] 对非对象分支、函数类型、未知类型保守回退，例如 `User & string` 不生成半截对象。
6. [x] 补充普通生成、coverage task 和 helper 级测试，固定交叉类型输出和负例边界。

已完成补充：JS/TS payload 现在能处理常见的对象组合 DTO。交叉类型仍只在所有分支都能解析成对象时合并，避免把复杂 TypeScript 类型系统中的非对象语义误生成为测试数据。

## 第七十九阶段：JS/TS indexed access payload

1. [x] 支持同文件对象类型字段访问，例如 `ApiResponse['data']`。
2. [x] 支持 indexed access alias，例如 `type ResponseData = ApiResponse['data']`。
3. [x] 支持内联对象字段访问，例如 `{ data: User }['data']`。
4. [x] 字段值中的 indexed access 继续复用命名类型、nullable union、递归保护和字段名值规则。
5. [x] 对 union key、`keyof`、泛型 `T[K]`、缺失字段和跨文件未知源类型保守回退。
6. [x] 补充普通生成、coverage task 和 helper 级测试，固定 indexed access 输出和负例边界。

已完成补充：JS/TS payload 现在能处理常见响应包装类型的字段抽取，例如 `ApiResponse['data']`。当前阶段只接受单个字符串字面量字段 key，不展开 TypeScript 类型系统里的动态索引语义。

## 第八十阶段：JS/TS 组合类型压力回归

1. [x] 补充组合 fixture，覆盖 `Pick` / `Omit` / `Record` / intersection / indexed access 混合使用。
2. [x] 固定普通生成入口不回退 `{ ok: true }`，并继续生成结构断言和 client call 断言。
3. [x] 固定 helper 级复杂组合 payload，防止后续调整类型解析时丢字段。
4. [x] 补齐链式投影源解析，例如 `Pick<DirectoryEnvelope, ...>` 中 `DirectoryEnvelope` 是 `Omit<...>` alias。
5. [x] 固定递归组合不会死循环；当时对象字段数组仍按保守 `[]` 策略记录，后续阶段已提升。

已完成补充：JS/TS payload 现在有一组跨类型能力的压力回归。它不扩展新的 TypeScript 语法范围，而是保证已支持的类型组合在同一个真实 DTO 场景里能稳定协作。

## 第八十一阶段：JS/TS 对象字段数组 payload

1. [x] 对对象字段中的数组类型生成一元素 payload，例如 `reports: User[]`。
2. [x] 字段数组元素复用命名类型、nullable union、递归保护、字段名值和组合类型规则。
3. [x] 无法解析数组元素时继续回退 `[]`，避免生成不可解释数据。
4. [x] 补充 helper 级测试，固定普通对象字段数组和递归组合数组字段输出。

已完成补充：JS/TS payload 现在不只在顶层数组返回时生成一元素样例，对 DTO 对象内部的数组字段也会给出更有信息量的结构断言，同时保留递归保护边界。

## 第八十二阶段：JS/TS 对象字段 tuple payload

1. [x] 固定对象字段中的普通 tuple，例如 `{ pair: [User, Meta] }`。
2. [x] 固定对象字段中的 readonly/labeled tuple，例如 `{ pair: readonly [user: User, meta?: Meta] }`。
3. [x] 固定对象字段中的 rest tuple，例如 `{ pair: readonly [User, ...Meta[]] }`。
4. [x] 字段 tuple 元素继续复用命名类型、nullable union、递归保护和字段名值规则。

已完成补充：JS/TS payload 的 tuple 能力现在不只覆盖顶层返回，也覆盖 DTO 对象内部字段，防止后续改动把 tuple 字段退回 `[]`。

## 第八十三阶段：JS/TS 对象字段投影类型 payload

1. [x] 固定对象字段中的 `Pick<T, K>` payload，例如 `{ owner: Pick<User, 'userId' | 'email'> }`。
2. [x] 固定对象字段中的 `Omit<T, K>` payload，例如 `{ owner: Omit<User, 'manager' | 'displayName'> }`。
3. [x] 固定对象字段中的 indexed access payload，例如 `{ data: ApiResponse['data'] }`。
4. [x] 固定对象字段中的组合投影 alias，例如 `{ summary: DirectorySummary }`。
5. [x] 字段投影继续复用命名类型、组合类型、字段名值规则和递归保护。

已完成补充：JS/TS payload 的投影类型能力现在覆盖 DTO 对象内部字段，避免 `Pick`、`Omit`、indexed access 只在顶层返回场景稳定。

## 第八十四阶段：JS/TS 对象字段 Record payload

1. [x] 固定对象字段中的字符串字面量 key Record，例如 `{ owners: Record<'primary' | 'secondary', User> }`。
2. [x] 固定对象字段中的 `Record<string, Pick<T, K>>` 组合 value，例如 `{ directory: Record<string, Pick<User, 'userId' | 'email'>> }`。
3. [x] 固定对象字段中的 unsupported Record key 保守回退，例如 `{ owners: Record<number, User> }` 生成 `{}`。
4. [x] 字段 Record value 继续复用投影类型、命名类型、字段名值规则和递归保护。

已完成补充：JS/TS payload 的 Record 能力现在覆盖 DTO 对象内部字段，能生成 map-like 字段的代表性样例，同时对无法解释的 key 类型保持保守输出。

## 第八十五阶段：JS/TS 字段级能力真实生成回归

1. [x] 在真实 TS fixture 中新增混合 DTO 返回类型，覆盖 `reports: User[]`。
2. [x] 覆盖真实生成入口中的 tuple 字段，例如 `pair: readonly [user: User, meta?: Meta]`。
3. [x] 覆盖真实生成入口中的 `Record<string, Pick<T, K>>` 字段。
4. [x] 覆盖真实生成入口中的组合投影 alias 字段，例如 `summary: DirectorySummary`。
5. [x] 固定最终生成的 Vitest 测试文本，不只验证底层 payload helper。

已完成补充：最近补齐的字段级数组、tuple、Record 和投影类型能力现在进入真实生成 fixture，能防止最终测试文本回退到 `{ ok: true }` 或浅层 object 断言。

## 第八十六阶段：JS/TS 字段级能力 client mock 回归

1. [x] 在真实 TS fixture 中新增 `api.fetch(): Promise<DirectoryBundle>` 的 client 注入函数。
2. [x] 固定 mock client 的 `return` payload，覆盖数组、tuple、Record 和组合投影字段。
3. [x] 固定 `expect(result).toEqual(...)` 对混合 DTO 的结构断言。
4. [x] 固定 `fetchCalls` 路径断言，确保 client 注入路径没有丢掉调用追踪。

已完成补充：字段级复杂 DTO 现在同时覆盖 `response.json()` 和注入 client 两条真实生成路径，降低生成器在 mock client 分支回退浅层数据的风险。

## 第八十七阶段：JS/TS payload 质量边界文档

1. [x] 新增 `docs/js-ts-payload-quality.md`，集中说明当前 JS/TS payload 质量目标。
2. [x] 记录已支持类型形态、对象字段组合能力和真实生成路径。
3. [x] 明确保守回退策略和暂不支持的 TypeScript 类型系统范围。
4. [x] 补充后续演进原则，要求新增能力同时覆盖 helper、真实生成入口和负例边界。
5. [x] 在 README 与质量评估文档中增加入口，避免后续能力说明散落。

已完成补充：JS/TS payload 这一轮能力收束为可维护的质量边界文档，后续推进时可以按文档判断是否属于静态生成器职责，避免继续无边界扩张。

## 第八十八阶段：JS/TS 复杂 payload 生成到运行检查

1. [x] 新增 handler 级 Vitest 临时项目 fixture，生成复杂 DTO 的测试文件。
2. [x] 在 `run_tests` 路径中通过 fake `npx` 读取生成文件并校验关键片段。
3. [x] 固定复杂 payload 的 Vitest API import、源模块 import、mock return、结果断言和 `fetchCalls` 断言。
4. [x] 固定 `run_tests` 对生成文件的项目根、相对测试路径和 pass 结果解析。

已完成补充：JS/TS 复杂 payload 现在不只在 generator preview 中受保护，也进入 `generate_tests -> run_tests` handler 闭环。CI 不依赖真实 npm 安装，但 fake runner 会检查生成文件内容，降低最终测试文本不可执行或关键断言丢失的风险。

## 第八十九阶段：JS/TS coverage task 复杂 payload 运行检查

1. [x] 新增 coverage task 版 Vitest 临时项目 fixture，目标函数返回复杂 `DirectoryBundle`。
2. [x] 固定 coverage task 的测试名、任务注释、Vitest API import 和源模块 import。
3. [x] 在 `run_tests` 路径中通过 fake `npx` 校验复杂 mock return、结构断言和 `fetchCalls` 断言。
4. [x] 固定 coverage task 生成文件的项目根、相对测试路径和 pass 结果解析。

已完成补充：复杂 JS/TS payload 的可运行性检查现在同时覆盖普通生成和 coverage task 生成路径，避免覆盖率驱动增量测试在类型化 mock、任务上下文或执行路径上落后。

## 第九十阶段：v0.4.11 JS/TS 生成质量发布草案

1. [x] 新增 `docs/plan-release-notes-v0.4.11.md`，归纳 v0.4.10 之后的 JS/TS payload 质量增强。
2. [x] 记录对象字段组合、真实生成路径、coverage task 路径和 handler 闭环保护。
3. [x] 明确本轮仍不改变 MCP 工具协议，也不引入完整 TypeScript 类型系统。
4. [x] 同步 `CHANGELOG.md` 的 Unreleased 条目，方便后续正式发版时收敛。

已完成补充：v0.4.11 候选发布资料已经覆盖本轮 JS/TS 静态生成质量增强，后续发版时可以在此基础上跑完整 release checklist、更新版本号和生成正式 release。

## 第九十一阶段：v0.4.11 发布前差异检查

1. [x] 核对当前版本号、README、安装文档、release 脚本和 GitHub Actions workflow。
2. [x] 新增 `docs/plan-release-v0.4.11.md`，记录已验证项、仍需正式发版前完成的版本切换和发布步骤。
3. [x] 跑通脚本语法检查、actionlint、`go test ./...`、主服务/CLI 构建和本地 darwin_arm64 打包 dry-run。
4. [x] 明确本阶段不打 tag、不创建 GitHub Release、不更新 Homebrew tap。

已完成补充：v0.4.11 现在有独立发布前检查清单。当前结论是候选资料和脚本可用，但正式发版前仍需要把 `main.go`、README、安装文档和 CHANGELOG 从 `v0.4.10` 切到 `v0.4.11`。

## 第九十二阶段：v0.4.11 版本准备改动

1. [x] 将 `main.go` MCP implementation version 更新为 `0.4.11`。
2. [x] 将 README 和 `docs/installation.md` 中的当前 Release、下载示例和安装维护示例同步到 `v0.4.11`。
3. [x] 将 `CHANGELOG.md` 的 Unreleased 内容收敛为 `v0.4.11 - 2026-07-09`。
4. [x] 更新 `docs/plan-release-v0.4.11.md`，标记已完成的版本准备项，保留 tag/release/Homebrew 待办。
5. [x] 版本切换后重新跑完整本地验证：diff 空白检查、脚本语法、actionlint、`go test ./...`、主服务/CLI 构建和本地 darwin_arm64 打包 dry-run。

已完成补充：v0.4.11 的代码版本号和用户安装文档已经切到新版本，版本切换后的本地验证已通过，提交 `8232e6b` 对应的远端 CI run `28995406760` 已通过。下一步进入 tag、Release Artifacts 和 Homebrew tap 发布核验阶段。

## 第九十三阶段：v0.4.11 正式发布核验

1. [x] 推送 `v0.4.11` tag。
2. [x] 等待 Release Artifacts workflow `28995989142` 完成，确认 Linux amd64、Linux arm64、macOS arm64、Windows amd64、Windows arm64 五个平台资产和 `.sha256` 上传成功。
3. [x] 运行 `scripts/verify-release-assets.sh v0.4.11`，确认 10 个 Release 资产完整。
4. [x] 更新 GitHub Release 正文为正式 v0.4.11 发布说明。
5. [x] 更新 `sleticalboy/homebrew-tap` 到 `testloop-mcp 0.4.11`，提交 `513d843` 并推送。
6. [x] 本机 Homebrew tap 快进到 `513d843`，通过 `brew fetch`、`brew audit --strict`、`brew upgrade`、`brew test`。
7. [x] 验证 release tarball 直连下载、checksum 和解包内容；记录 `scripts/install.sh` 在本机 GitHub 443 网络不稳定时回退 `go install` 且安装命令可运行。

已完成补充：v0.4.11 已正式发布并完成 Release/Homebrew 核验。下一步回到功能开发侧，优先处理安装脚本在下载失败时提示过于笼统的问题，或者进入下一批 JS/TS 静态生成边界增强。

## 第九十四阶段：安装脚本 fallback 提示细化

1. [x] 将 `scripts/install.sh` 的 `go install` fallback 改为接收明确原因。
2. [x] 区分不支持的 OS/架构、latest release tag 解析失败、Release 资产下载失败和缺少解压器。
3. [x] 补充离线安装脚本测试，固定下载失败时的提示，不再把网络问题笼统描述成“没有匹配资产”。
4. [x] 同步 `CHANGELOG.md`、`docs/plan-installation.md` 和主仓库 `Formula/testloop-mcp.rb` 到当前发布状态。

已完成补充：安装脚本在 GitHub 443 网络波动或 release asset 下载失败时，现在会明确提示下载失败和 GitHub 网络可达性检查，再回退到 `go install`；不支持的平台和缺少解压器也会给出对应原因。

## 第九十五阶段：v0.4.11 发布后五平台安装验证

1. [x] 修正 `Post-Release Verify` workflow 的 fallback 检测，让它大小写不敏感匹配 `Falling back to go install`。
2. [x] 推送 workflow 修复提交 `c87d231`，并确认 main CI run `29010775969` 通过。
3. [x] 手动触发 `Post-Release Verify` workflow，输入 `v0.4.11`。
4. [x] run `29010902985` 中资产清单校验通过。
5. [x] linux_amd64、linux_arm64、windows_amd64、windows_arm64 安装 dry run 通过，且未走 `go install` fallback。
6. [x] darwin_arm64 首次等待 `macos-26` runner 后被取消，rerun failed job 后安装 dry run 和 help 检查通过。

已完成补充：v0.4.11 的发布后远端安装矩阵已经收口，五个平台安装脚本 dry run 全部通过。下一步回到功能开发侧，建议继续推进 JS/TS 静态生成边界增强，优先补跨文件类型/泛型边界的“明确不支持但可解释提示”或更小范围的静态展开能力。

## 第九十六阶段：JS/TS 同文件简单泛型 DTO 展开

1. [x] TypeScript 类型声明提取保留简单泛型参数，支持 `type ApiEnvelope<T> = ...` 和 `interface ApiEnvelope<T> { ... }`。
2. [x] JS/TS payload 解析支持同文件泛型 alias/interface 的直接实例化，例如 `ApiEnvelope<User>`、`Pair<User, Meta>`。
3. [x] 泛型参数只做简单标识符替换，继续拒绝约束、默认参数、`T[K]` 和跨文件类型推导。
4. [x] 补齐 helper 级和真实生成级回归测试，固定 `response.json()` 返回泛型 DTO 时生成结构化 payload。
5. [x] 同步 `docs/js-ts-payload-quality.md` 和 `CHANGELOG.md`，明确新支持范围与不支持边界。

已完成补充：JS/TS payload 现在能覆盖真实 API helper 常见的响应包装泛型，例如 `ApiEnvelope<User>`。这个阶段仍不引入完整 TypeScript 类型系统，只在当前文件、直接实例化、参数可简单替换的范围内展开，复杂泛型继续保守回退。

## 第九十七阶段：JS/TS payload 回退原因上下文化

1. [x] `TestTarget` 增加可选 `return_type_expr`，让 `generate_tests.context.targets[]` 保留 TypeScript 返回注解。
2. [x] `TestTarget` 增加可选 `payload_notes`，在静态 payload 无法展开时说明回退原因。
3. [x] 覆盖跨文件命名类型、约束泛型和同文件简单泛型三类 context 回归。
4. [x] 同步 README、LLM provider 文档、JS/TS payload 质量文档和 `CHANGELOG.md`。

已完成补充：JS/TS 静态生成仍保持保守边界，但现在能把“为什么退回 `{ ok: true }`”暴露给 Agent/LLM provider。后续 provider 可以基于 `return_type_expr` 和 `payload_notes` 决定是否读取更多项目上下文，而不是误以为静态生成器已经完整理解了 DTO。

## 第九十八阶段：JS/TS payload notes 工具输出契约

1. [x] 新增 `generate_tests` handler 级回归测试，从 MCP 工具输出 JSON 反序列化 `GenerateTestsOutput`。
2. [x] 固定跨文件 TypeScript 返回类型会出现在 `context.targets[].return_type_expr`。
3. [x] 固定静态 payload 回退原因会出现在 `context.targets[].payload_notes`。
4. [x] 固定生成 preview 仍保持可运行的 `{ ok: true }` 保守 mock。

已完成补充：`payload_notes` 不再只是 generator 内部 context 能力，已经进入 `generate_tests` 工具输出契约。后续外部 Agent 或 LLM provider 可以稳定依赖该字段判断静态草稿为何回退。

## 第九十九阶段：JS/TS payload notes provider 输入契约

1. [x] 新增外部 LLM provider 请求级回归测试，捕获 provider stdin JSON。
2. [x] 固定跨文件 TypeScript 返回类型会出现在 provider 请求的 `context.targets[].return_type_expr`。
3. [x] 固定静态 payload 回退原因会出现在 provider 请求的 `context.targets[].payload_notes`。
4. [x] 固定 provider 请求里的 `static_code` 仍包含 `{ ok: true }` 保守断言，方便 provider 在此基础上增强。

已完成补充：`payload_notes` 已经贯通到外部 LLM provider 输入契约。这样静态生成、MCP 工具输出和 provider stdin 三条链路都有回归保护，Agent 能可靠获知静态 payload 回退原因。

## 第一百阶段：v0.4.12 发布说明草案

1. [x] 新增 `docs/plan-release-notes-v0.4.12.md`，归纳 v0.4.11 之后的安装脚本 fallback 提示和 JS/TS payload 增强。
2. [x] 明确 v0.4.12 的范围是同文件简单泛型 DTO 展开、`payload_notes` 上下文化和 provider 输入契约，不新增 MCP 工具。
3. [x] 写清楚仍不支持完整 TypeScript 类型系统，跨文件类型、约束泛型、`keyof` / `T[K]` 等继续保守回退。
4. [x] 给出正式发布前验证清单，覆盖 generator、tools、安装脚本、空白检查、全量测试和远端 CI。

已完成补充：v0.4.12 候选发布资料已经建立。下一步如果进入正式发版，需要更新 `main.go` implementation version、收敛 `CHANGELOG.md` 的 Unreleased、同步版本文档，然后走 tag、Release Artifacts、资产校验和 Homebrew tap 流程。

## 第一百零一阶段：v0.4.12 版本准备改动

1. [x] 将 `main.go` MCP implementation version 更新为 `0.4.12`。
2. [x] 将 `CHANGELOG.md` 的 Unreleased 内容收敛为 `v0.4.12 - 2026-07-09`。
3. [x] 将 README 和 `docs/installation.md` 中的当前 Release、下载示例和安装维护示例同步到 `v0.4.12`。
4. [x] 新增 `docs/plan-release-v0.4.12.md`，记录版本准备、验证项和正式发布前待办。

已完成补充：v0.4.12 的版本号和用户安装文档已经切到新版本，发布检查清单已建立，本地发布前验证已通过。下一步需要确认远端 CI 通过，然后进入 tag、Release Artifacts 和 Homebrew tap 发布核验阶段。

## 第一百零二阶段：v0.4.12 正式发布核验

1. [x] 确认版本准备提交 `ccf38f2` 的远端 CI run `29021717743` 通过。
2. [x] 推送 `v0.4.12` tag，tag 指向 `ccf38f2b9f902b62e6c923a7017f31391e3a91fd`。
3. [x] 等待 Release Artifacts run `29022581976` 通过，确认 Linux amd64、Linux arm64、macOS arm64、Windows amd64、Windows arm64 五个平台资产和 `.sha256` 上传成功。
4. [x] 运行 `scripts/verify-release-assets.sh v0.4.12`，确认 10 个 Release 资产完整。
5. [x] 更新 GitHub Release 正文为正式 v0.4.12 发布说明。
6. [x] 手动触发 Post-Release Verify run `29025114403`，确认五平台安装脚本 dry run 全部通过。
7. [x] 更新 `sleticalboy/homebrew-tap` 到 `testloop-mcp 0.4.12`，提交 `1c62ce0` 并推送。
8. [x] 本机 Homebrew tap 快进到 `1c62ce0`，通过 `brew fetch`、`brew audit --strict`、`brew upgrade` 和 `brew test`。
9. [x] 发布后修正安装脚本跨平台 dry run 下载失败时 fallback 日志误报 `.exe` 路径的问题，并将其记录到 `CHANGELOG.md` 的 Unreleased。

已完成补充：v0.4.12 已正式发布并完成 Release/Homebrew 核验。下一步回到功能开发侧，优先继续推进 JS/TS 静态生成质量边界：跨文件类型的可解释上下文、LLM provider 增强输入消费示例，或更小范围的 TypeScript 类型展开能力。

## 第一百零三阶段：JS/TS imported type 上下文提示

1. [x] 解析 JS/TS import 行中的 named import、default import、namespace import 和 require module 来源。
2. [x] 当 `return_type_expr` 引用 imported type 时，在 `payload_notes` 中追加 import 来源和候选源码文件。
3. [x] 保持不做自动跨文件类型展开，候选文件只作为 Agent/LLM provider 的下一步读取建议。
4. [x] 覆盖 context、provider stdin 和 MCP handler 输出三条链路，固定 notes 会贯通到外部调用方。
5. [x] 同步 README、LLM provider 文档、JS/TS payload 质量文档和 `CHANGELOG.md`。

已完成补充：JS/TS 静态生成现在能告诉 Agent “这个 DTO 来自哪里”，而不是只说同文件没有声明。下一步建议推进 provider 示例消费这些 hints：让示例 provider 在收到候选文件后读取同目录类型文件，并在 prompt 中补充 imported DTO 定义；仍先作为外部增强，不把完整跨文件 TypeScript 类型系统塞进内置 static provider。

## 第一百零四阶段：LLM provider 示例消费 imported type hints

1. [x] 增强 `examples/llm-provider.sh`，解析 `payload_notes` 里的 `read candidate source files` 提示。
2. [x] 读取存在的同目录候选类型文件，并把内容写入 prompt 的 `Imported Type Context` 小节。
3. [x] 保持默认 stdout 只返回测试代码 JSON，不把 prompt 或解释性文本写入生成测试文件。
4. [x] 新增 `TESTLOOP_LLM_PROVIDER_PROMPT_FILE` 调试入口，便于查看最终 prompt。
5. [x] 新增 `TESTLOOP_LLM_PROVIDER_MODEL_CMD` 入口，允许示例脚本把 prompt 交给真实模型命令。
6. [x] 增加离线 shell 测试并接入 CI，固定示例 provider 会读取 `types.ts` 并保留 static code 回退行为。
7. [x] 同步 README、LLM provider 文档和 `CHANGELOG.md`。

已完成补充：imported type hints 现在不只是出现在 MCP/Provider JSON 中，也有示例 provider 展示如何消费这些 hints。下一步可以继续做更贴近真实模型的 provider 示例，例如为 Ollama/OpenAI CLI 组装 prompt 模板，或先补 provider 输出清洗/校验，避免模型返回解释性文本时污染测试文件。

## 第一百零五阶段：LLM provider 输出清洗与校验

1. [x] 在 `ExternalLLMProvider` 输出解析后增加统一清洗逻辑。
2. [x] 支持从直接 stdout 和 JSON `code` 字段中提取 Markdown 代码围栏内容。
3. [x] 没有代码围栏时，去掉常见前后解释性文本，只保留可识别代码行范围。
4. [x] 对纯解释性输出返回明确错误，避免写入测试文件。
5. [x] 补充 provider 单元测试，覆盖 raw markdown、JSON markdown 和 explanation-only 三类输出。
6. [x] 同步 README、LLM provider 文档和 `CHANGELOG.md`。

已完成补充：真实 LLM provider 现在即使返回常见的 Markdown 包装，也不会直接污染生成的测试文件。下一步可以继续做 provider 质量增强：增加可选的输出后验检查，例如按目标语言做轻量语法/框架关键字校验，或为 Ollama/OpenAI CLI 提供更具体的示例模板。

## 第一百零六阶段：LLM provider 输出语言级轻量校验

1. [x] 在外部 LLM provider 输出清洗后增加按目标语言的测试代码校验。
2. [x] Go/Python/JS/TS/Rust/Java 分别识别常见测试入口或测试框架信号，拒绝明显不是测试的业务实现片段。
3. [x] 保持校验只作用于外部 LLM provider 输出，不改变默认 static provider 行为。
4. [x] 补充 provider 单元测试，覆盖各语言通过/拒绝规则和 fake provider 非测试输出失败路径。
5. [x] 同步 README、LLM provider 文档和 `CHANGELOG.md`。

已完成补充：LLM provider 现在不只会剥掉 Markdown 包装，也会阻止明显不是测试代码的片段进入生成文件。下一步建议继续推进 provider 生态样例：给 Ollama/OpenAI CLI 准备可直接复用的 prompt 模板和命令包装，降低真实模型接入成本。

## 第一百零七阶段：LLM provider 真实模型接入样例

1. [x] 新增默认 prompt 模板 `examples/llm-provider-prompt.md`，把源码文件、语言、框架、coverage task、static draft、imported type context 和完整请求 JSON 显式组织给模型。
2. [x] `examples/llm-provider.sh` 支持 `TESTLOOP_LLM_PROVIDER_PROMPT_TEMPLATE`，允许用户替换模板而不改 provider 协议。
3. [x] 新增 `examples/model-ollama.sh`，把 prompt 转发给 `ollama run "$TESTLOOP_OLLAMA_MODEL"`。
4. [x] 新增 `examples/model-openai-cli.sh`，通过官方 `openai responses create` 调用 OpenAI CLI，并只抽取模型文本输出。
5. [x] 为模型命令包装加入 dry-run，并扩展 shell 测试覆盖模板渲染、模型命令覆盖 static draft、Ollama/OpenAI wrapper dry-run。
6. [x] 同步 README、LLM provider 文档和 `CHANGELOG.md`。

已完成补充：外部 LLM provider 现在从“协议示例”推进到“可落地接入真实模型”的样例层。下一步建议做生成后闭环：让 Agent 文档和测试路径明确推荐 `generate_tests(provider=auto/llm) -> run_tests -> fix_suggestions`，并补一个 provider 输出到 run_tests 的端到端 dry-run 契约。

## 第一百零八阶段：LLM provider 生成后反馈闭环

1. [x] 新增 handler 层 dry-run 回归，使用 fake LLM provider 生成 Vitest 测试文件。
2. [x] 使用 fake `npx vitest` 模拟生成测试失败，验证 `run_tests` 会解析 provider 产物的失败输出。
3. [x] 开启 `include_fix_suggestions=true`，固定失败结果会内联 `fix_suggestions[]` 和 `repair_task`。
4. [x] 校验 repair task 的上下文片段和建议复跑命令来自 LLM provider 写入的测试文件。
5. [x] 同步 README、LLM provider 文档、Agent workflow 和 `CHANGELOG.md`。

已完成补充：LLM provider 现在有了“生成 -> 执行 -> 失败解析 -> repair task”的本地契约保护。下一步建议补强真实用户体验：在 `cmd/testgen` 或文档中增加 provider 环境诊断，提前发现 `TESTLOOP_LLM_PROVIDER_CMD` 缺失、模型命令不可执行、输出被校验拒绝等常见接入问题。

## 第一百零九阶段：LLM provider 接入诊断

1. [x] `cmd/testgen` 新增 `-provider-check`，无需源码文件即可检查 provider 配置。
2. [x] 诊断 `static`、`auto` fallback、`llm` 缺少 `TESTLOOP_LLM_PROVIDER_CMD`、命令首个可执行文件不存在和命令可解析成功等场景。
3. [x] LLM provider 实际生成失败时，CLI 会提示用户先运行 `testgen -provider <mode> -provider-check` 排查基础配置。
4. [x] 补充 CLI 单元测试，固定诊断输出、退出码和输出校验失败时的提示。
5. [x] 同步 README、LLM provider 文档和 `CHANGELOG.md`。

已完成补充：真实模型接入现在有了生成前的基础配置检查，用户可以先排除环境变量和命令路径问题，再处理模型输出质量问题。下一步建议继续补 MCP 层可观测性：让 `generate_tests` 的错误信息对 provider 失败、输出为空、JSON 错误和语言校验失败给出更清晰的错误分类，方便 Agent 自动决定是重试模型、降级 static，还是提示用户修配置。

## 第一百一十阶段：MCP generate_tests provider 错误分类

1. [x] generator 层新增 `ProviderError` 和稳定 `ProviderErrorKind`，覆盖配置缺失、命令失败、空输出、JSON 错误、缺少 `code`、输出清洗失败和语言校验失败。
2. [x] 外部 LLM provider 的失败路径保留原错误语义，同时挂载 kind 和 provider 信息。
3. [x] `generate_tests` handler 在错误文本中输出 `provider_error kind=... action=...`，便于 Agent 做机器判断。
4. [x] 补充 generator 和 handler 回归测试，固定 provider error kind、action 和错误文本。
5. [x] 同步 README、LLM provider 文档和 `CHANGELOG.md`。

已完成补充：MCP 层现在能区分 LLM provider 是配置问题、命令问题、模型输出格式问题，还是输出不像测试代码。下一步建议把这套分类继续接到策略文档和示例 prompt：给 Agent 一个明确 fallback 策略表，例如哪些错误重试模型，哪些错误直接降级 static，哪些错误提示用户修 provider 配置。

## 第一百一十一阶段：LLM provider 错误处理策略

1. [x] 在 Agent workflow 中新增 `provider_error kind/action` 策略表。
2. [x] 明确 `llm_config_missing` 不应重试模型，应提示配置或降级 static。
3. [x] 明确 `llm_command_failed` 可根据 stderr 判断是否重试一次，鉴权、脚本或命令问题应提示用户修 provider。
4. [x] 明确空输出、JSON 错误、缺少 `code`、清洗失败和语言校验失败的自动 fallback 策略。
5. [x] 明确自动化 Agent 不应无限重试，同一任务最多重试一次 provider，仍失败则降级 static 并继续 `run_tests` 闭环。
6. [x] 同步 README 和 `CHANGELOG.md`。

已完成补充：provider 错误分类现在不只是技术字段，也有了 Agent 执行策略。下一步建议继续提升 provider 成功路径质量：把 prompt 模板里的输出约束和 fallback 规则写得更强，减少模型产生解释文本、非测试代码或错误格式的概率。

## 第一百一十二阶段：LLM provider prompt 成功路径约束

1. [x] 在默认 prompt 模板中新增 `Output Contract`，明确只返回一个可直接写盘的完整测试文件。
2. [x] 明确模型必须使用目标语言和测试框架，并保持静态草稿的导入、风格和文件布局。
3. [x] 明确无法安全增强时原样返回静态草稿，避免生成不可执行的“自信错误”测试。
4. [x] 明确禁止 JSON、解释文本、命令、伪代码、TODO-only 测试、生产代码 patch 和 Markdown 包装。
5. [x] 更新示例 provider 回归测试，固定 prompt 渲染文件和真实模型 stdin 都包含关键输出约束。
6. [x] 同步 README、LLM provider 文档和 `CHANGELOG.md`。

已完成补充：LLM provider 的成功路径现在不只靠输出后清洗和校验兜底，默认 prompt 已经前置约束模型的输出形态，并给出静态草稿回退规则。下一步建议继续补 provider 体验闭环：增加“坏模型输出”的端到端示例或 golden 测试，覆盖 Markdown/解释文本/非测试代码进入清洗、校验、错误分类和 Agent fallback 的完整路径。

## 第一百一十三阶段：LLM provider 坏输出闭环回归

1. [x] 在 MCP handler 层新增 LLM provider 坏输出表驱动测试，覆盖真实外部命令路径。
2. [x] 固定空 stdout 会映射为 `llm_empty_output` 和 `retry_model_or_fallback_static`。
3. [x] 固定坏 JSON 会映射为 `llm_json_error` 和 `retry_model_or_fallback_static`。
4. [x] 固定 JSON 缺少非空 `code` 会映射为 `llm_missing_code` 和 `retry_model_or_fallback_static`。
5. [x] 固定解释文本清洗失败会映射为 `llm_output_cleaning_failed` 和 `retry_model_or_fallback_static`。
6. [x] 固定“像代码但不是测试”的输出会映射为 `llm_output_validation_failed` 和 `adjust_prompt_or_fallback_static`。
7. [x] 同步 `CHANGELOG.md`。

已完成补充：provider 的坏输出不只在 generator 内部有单元测试，现在 MCP `generate_tests` 入口也固定了 Agent 实际可见的错误分类和动作建议。下一步建议继续提升 Agent 消费体验：把 `provider_error kind/action` 从错误字符串升级为结构化工具返回字段，减少 Agent 解析文本的需求。

## 第一百一十四阶段：LLM provider 错误结构化返回

1. [x] 扩展 `types.GenerateTestsOutput`，新增 `error` 和 `provider_error` 字段。
2. [x] 新增结构化 `ProviderErrorOutput`，包含 `kind`、`action`、`provider` 和 `message`。
3. [x] 将 MCP `generate_tests` 的 LLM provider 失败改为 `isError=true` 的工具结果，避免只走协议级 error。
4. [x] 在文本 JSON 和 `structuredContent` 中同时返回结构化 provider error，方便 Agent 直接消费。
5. [x] 保留 `error` 字段中的 `provider_error kind=... action=...` 文本片段，兼容旧 Agent。
6. [x] 更新 handler 回归测试，覆盖配置错误和坏输出场景的结构化结果。
7. [x] 同步 README、LLM provider 文档、Agent workflow 和 `CHANGELOG.md`。

已完成补充：Agent 现在不需要从中文错误文本里正则提取 provider 错误分类，可以优先读取 `provider_error.kind/action`；只读文本的旧路径也仍然可用。下一步建议继续补 provider 自动恢复能力：在 Agent workflow 文档和测试中明确“结构化 provider_error -> 自动降级 static -> run_tests”的工具调用序列。

## 第一百一十五阶段：LLM provider 失败后的 static fallback 闭环

1. [x] 新增 handler 级闭环测试，模拟 fake LLM provider 输出解释文本并触发结构化 provider error。
2. [x] 校验 `provider_error.action=retry_model_or_fallback_static` 可被 Agent 用作降级信号。
3. [x] 校验 LLM provider 失败时不会写入测试文件，避免坏输出污染工作区。
4. [x] 使用同一 `file_path` / `framework` 降级为 `provider=static` 并生成 Vitest 测试文件。
5. [x] 通过 fake `npx vitest` 执行 static fallback 生成的测试文件，固定 fallback 后仍必须进入 `run_tests`。
6. [x] 同步 Agent workflow 文档和 `CHANGELOG.md`。

已完成补充：provider 失败后的恢复路径现在不只是文档建议，而是有 handler 层回归测试保护的完整工具调用序列。下一步建议继续提升 release 准备度：收敛 Unreleased 内容，更新下一版发布说明草案，确认当前 LLM provider 系列改动是否进入 v0.4.13。

## 第一百一十六阶段：v0.4.13 发布说明草案

1. [x] 新增 `docs/plan-release-notes-v0.4.13.md`，归纳 v0.4.12 之后的 LLM provider 和安装脚本 fallback 日志改动。
2. [x] 明确 v0.4.13 的范围是 LLM provider 接入质量、输出校验、结构化错误、Agent fallback 闭环和安装脚本日志修正。
3. [x] 明确 v0.4.13 不内置模型厂商 SDK、不改变默认 static provider 行为、不把外部 provider 成功输出视为最终结论。
4. [x] 汇总当前已通过的本地验证和远端 CI run `29077553397`。
5. [x] 列出正式发布前仍需执行的版本号、CHANGELOG、README/安装文档、构建和资产预检事项。

已完成补充：v0.4.13 候选发布说明已经建立，当前只做发布资料准备，没有提前切版本号或改安装文档。下一步如果进入正式版本准备，需要更新 `main.go` implementation version、将 `CHANGELOG.md` 的 Unreleased 收敛为 `v0.4.13 - 2026-07-10`，同步 README/installation 版本引用，并跑发布前构建和资产预检。

## 第一百一十七阶段：v0.4.13 版本准备改动

1. [x] 将 `main.go` MCP implementation version 更新为 `0.4.13`。
2. [x] 将 `CHANGELOG.md` 的 Unreleased 内容收敛为 `v0.4.13 - 2026-07-10`。
3. [x] 将 README 和 `docs/installation.md` 中的当前 Release、下载示例和安装维护示例同步到 `v0.4.13`。
4. [x] 新增 `docs/plan-release-v0.4.13.md`，记录版本准备、验证项和正式发布前待办。
5. [x] 跑完整本地发布前验证，覆盖脚本语法、actionlint、`go test ./...`、provider 示例、安装脚本、release 资产脚本、二进制构建、help 输出和 darwin_arm64 打包 dry-run。

已完成补充：v0.4.13 的版本号和用户安装文档已经切到新版本，发布检查清单已建立，本地发布前验证已通过。下一步需要确认远端 CI 通过，然后进入 tag、Release Artifacts、资产校验、GitHub Release 正文和 Homebrew tap 发布核验阶段。

## 第一百一十八阶段：v0.4.13 正式发布和发布后核验

1. [x] 确认 release prep 远端 CI run `29087539959` 通过。
2. [x] 创建并推送 `v0.4.13` tag，tag 指向 `cebb4832ef9a7b8a84dbbb71e19f2989c1c74599`。
3. [x] Release Artifacts run `29089692602` 通过，生成 Linux amd64、Linux arm64、macOS arm64、Windows amd64 和 Windows arm64 五个平台资产及 `.sha256`。
4. [x] 运行 `scripts/verify-release-assets.sh v0.4.13`，确认 10 个 Release 资产完整。
5. [x] 将 GitHub Release 正文更新为正式 v0.4.13 发布说明。
6. [x] 手动触发 Post-Release Verify run `29090486292`，确认五平台安装脚本 dry run 全部通过。
7. [x] 更新并推送 Homebrew tap commit `25b8018454c1b73cf259c08b13db06f59dcfc234`，本机 `brew fetch`、`brew audit --formula --strict`、`brew upgrade --formula`、`brew test` 均通过。

已完成补充：v0.4.13 已正式发布并完成发布后核验。下一步建议回到功能质量推进，优先选择一个真实多文件 JS/TS 或 Go 样例项目，验证 `coverage_task -> generate_tests(llm/static) -> run_tests -> repair_task` 在真实仓库里的端到端收益，并把失败样本转成 parser/generator 回归测试。

## 第一百一十九阶段：真实 Go 项目覆盖率闭环样本

1. [x] 使用 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server` 作为真实 Go server 样本，确认 `go test ./...` 基线通过。
2. [x] 生成 Go coverprofile 并通过 `parse_coverage` 获取覆盖率任务：基线总覆盖率约 `4.39%`，`test_tasks` 数量为 1207。
3. [x] 发现真实摩擦点：Go coverprofile 中的文件路径是 module 路径（如 `car-svc/utils/time.go`），当前 `generate_tests` 需要真实文件路径，Agent 需要手动映射到 `utils/time.go`。
4. [x] 用 `utils/time.go` 的 `GetNowDate` 覆盖率任务触发 `generate_tests(provider=static)`，发现原生成结果为 `skip: true` 的 TODO 用例，`run_tests` 通过但 `skipped=1`，不能有效提升覆盖率。
5. [x] 修正 Go static coverage task 生成器：无参数、非方法、返回值不含 error/chan/func 且无法推导精确期望值时，生成可执行 smoke 调用，不再默认 skipped。
6. [x] 新增回归测试固定 `GetNowDate()` 这类真实样本会生成 `skip: false`、调用目标函数并丢弃返回值。
7. [x] 重新用 laoxia server 样本验证：`run_tests` 返回 `passed=20`、`skipped=0`、`utils` 覆盖率 `38.9%`，全量 `go test ./...` 通过；`GetNowDate` 覆盖率提升到 100%。

已完成补充：真实样本证明当前闭环能落地，但也暴露了两个后续方向。第一，`parse_coverage` 应该在 Go 项目中自动把 module 路径映射到本地文件路径，避免 Agent 手工改 `coverage_task.file/test_file`。第二，smoke 用例能消除 skipped 并覆盖返回路径，但断言价值有限，下一步应继续提升 Go 生成器对时间/日期、格式化字符串和纯函数返回值的语义断言能力。

## 第一百二十阶段：Go coverage task 本地路径映射

1. [x] 修正 Go coverprofile 解析，在当前工作目录或上级目录读取 `go.mod` module 名。
2. [x] 当 coverprofile 文件路径以 module 名开头且本地文件存在时，将路径归一化成本地相对路径，例如 `car-svc/utils/time.go` -> `utils/time.go`。
3. [x] 保留无本地源码的原始 profile 路径行为，避免破坏纯字符串解析和远端路径报告。
4. [x] 新增回归测试，固定 `parse_coverage` 产出的 `CoverageFile.Path`、`test_tasks[].file`、`test_tasks[].test_file` 和 `test_tasks[].command` 都使用本地可执行路径。
5. [x] 修正 Go coverage smoke 测试对 `time.Time` 等复合返回值的 import 判定，避免不比较返回值时仍引入未使用的 `reflect`。
6. [x] 使用 laoxia server 临时 worktree 重新验证完整链路：`parse_coverage` 直接返回 `utils/time.go`，`generate_tests` 生成 `GetCurrentDate` smoke 测试，`run_tests` 返回 `passed=20`、`skipped=0`、`coverage_percent=39.4`。

已完成补充：Go server 真实样本现在不需要 Agent 手动修 `coverage_task.file/test_file` 了，`parse_coverage -> generate_tests -> run_tests` 可以直接串起来。下一步建议处理生成文件合并能力：当前 `generate_tests` 会覆盖目标测试文件，连续补多个 coverage task 时会替换前一个生成用例；应改为在可解析的 Go test 文件中追加新测试函数或至少检测冲突并提示 Agent。

## 第一百二十一阶段：Go generate_tests 测试文件合并

1. [x] 将 Go 测试写入从直接覆盖改为保守合并：已有 `*_test.go` 可解析时，追加新生成的 `Test*` 函数。
2. [x] 合并新生成测试需要的 import，已有 import 复用，最后通过 `gofmt` 规范化输出。
3. [x] 遇到已有同名 `Test*` 函数时返回明确错误，避免静默覆盖或生成重复函数。
4. [x] 新增 handler 回归测试，固定连续两个 Go coverage task 写入同一个测试文件时，第二次会保留第一次生成的测试函数。
5. [x] 新增 handler 回归测试，固定重复生成同名 Go coverage task 会失败并提示已存在的测试函数。
6. [x] 使用 laoxia server 临时 worktree 验证真实项目连续生成两个 coverage task 后，`run_tests` 仍能通过且两个测试函数都保留。
7. [x] 修正 Go `run_tests` 相对路径归一化：`utils/time_test.go` 和 `utils` 会执行为 `./utils`，避免被 Go 当成标准库导入路径。
8. [x] 新增 `normalizeGoTestPath` 相对路径回归测试，固定相对文件、相对目录和显式 `./pkg` 的行为。

已完成补充：Go coverage task 连续生成现在不会覆盖已有测试，真实样本中 `utils/time_test.go` 同时保留 `TestGetNowDate` 和 `TestGetCurrentDate`，`run_tests` 单文件路径返回 `passed=22`、`skipped=0`、`coverage_percent=39.8`。下一步建议继续提升 Go static 生成质量，优先从时间/日期和简单格式化返回值生成可断言用例，减少只有 smoke 调用但没有断言的测试。

## 第一百二十二阶段：Go 时间格式返回值断言

1. [x] 扩展 Go AST 表达式序列化，支持 `CallExpr`，可识别 `time.Now().Format("2006-01-02")` 这类返回表达式。
2. [x] 扩展 Go seed case 类型，除精确期望值外新增时间格式断言模式。
3. [x] 当 coverage task 命中无参数 string 返回函数且返回表达式为 `time.Now().Format("layout")` 时，生成 `time.Parse(tt.layout, got)` 断言。
4. [x] 自动为生成测试补充 `time` import，并避免生成 `ret0` 字段或 `_ = got` smoke 形态。
5. [x] 新增生成器回归测试，固定 `GetNowDate()` 会生成 `layout: "2006-01-02"` 和 `time.Parse` 断言。
6. [x] 更新表达式 helper 回归测试，固定 `make([]int, 0)` 和 `time.Now().Format(...)` 的 call 表达式输出。
7. [x] 使用 laoxia server 初始提交临时 worktree 验证真实链路：`parse_coverage` 产出 `GetNowDate` task，`generate_tests` 生成 `time.Parse` 断言，`run_tests` 返回 `passed=20`、`skipped=0`、`coverage_percent=38.9`。

已完成补充：`GetNowDate()` 不再只是“调用一下证明覆盖到”，而是能验证返回值符合日期格式。下一步建议处理 `time.Time` 返回值的日期归零断言，例如 laoxia 的 `GetCurrentDate()` 可断言 hour/min/sec/nsec 全为 0，并保留 smoke fallback 作为兜底。

## 第一百二十三阶段：Go time.Time 日期边界断言

1. [x] 保持普通精确 seed 的严格单 return 规则，同时为时间日期模式单独读取最后一条 return，覆盖 `now := time.Now(); return time.Date(...)` 这类真实代码且不误伤普通分支函数。
2. [x] 新增 Go call 参数拆分工具，能识别 `time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())` 中嵌套调用参数。
3. [x] 扩展 Go seed case 类型，新增 `time_date_zero` 断言模式。
4. [x] 当 coverage task 命中无参数 `time.Time` 返回函数且返回表达式为 `time.Date(..., 0, 0, 0, 0, ...)` 时，生成 `Hour/Minute/Second/Nanosecond` 归零断言。
5. [x] 避免该模式误引入 `reflect`、`ret0 time.Time` 或 `_ = got` smoke 形态。
6. [x] 新增生成器回归测试，固定 laoxia 形态的 `GetCurrentDate()` 会生成日期边界断言。
7. [x] 使用 laoxia server 初始提交临时 worktree 验证真实链路：连续生成 `GetNowDate` 和 `GetCurrentDate`，同一 `utils/time_test.go` 保留两个测试，`run_tests` 返回 `passed=22`、`skipped=0`、`coverage_percent=39.8`。

已完成补充：laoxia 的两个时间辅助函数现在都能生成有断言的测试，覆盖率闭环从“能跑”推进到“有基本行为验证”。下一步建议转向 Go 分支输入质量：优先利用 `parse_coverage.suggested_inputs` 中的条件提示，为简单 `if a == 0` / `if x > n` 分支生成非 TODO 的输入和期望，而不是默认 skipped。

## 第一百二十四阶段：Go 简单分支输入和期望推导

1. [x] 在 Go AST 收集阶段记录顶层简单 `if` 分支 boundary，限定为 `参数 <op> 字面量` 且分支体内能找到单返回值 `return`。
2. [x] 支持 `==`、`!=`、`>`、`<`、`>=`、`<=` 的基础输入生成，并对 `0 < a` 这类右侧参数比较做操作符反转。
3. [x] 从 coverage task 的 `suggested_inputs`、`missing_branches` 和 `assertion_focus` 中提取反引号条件，选择匹配的 branch boundary。
4. [x] 对可安全替换参数的 branch return 表达式生成精确期望值，例如 `if a == 0 { return b }` 生成 `a: 0`、`b: 2`、`ret0: 2`。
5. [x] 保持保守策略：只有参数类型、分支条件和返回表达式都可安全推导时才生成非 skipped 用例，否则继续保留 TODO。
6. [x] 新增 Go generator 回归测试，固定 coverage task 命中 `a == 0` 分支时不会生成 skipped TODO。
7. [x] 更新 handler golden，固定 `parse_coverage -> generate_tests` 的 Go 分支闭环从 skipped TODO 变为可执行分支断言。
8. [x] 使用临时 Go module 验证端到端：已有默认路径测试覆盖 `Add(1,2)`，coverage task 生成 `Add(0,2)` 分支测试，`run_tests` 返回 `passed=3`、`skipped=0`、`coverage_percent=100`。

已完成补充：Go 分支 coverage task 的静态生成质量明显提升，简单 `if` 分支现在能直接产生有效测试，而不是让 Agent/用户手动填 TODO。下一步建议扩展到更常见的分支形态：字符串空值 `s == ""`、布尔 `enabled == false`、错误路径 `if err != nil` / `if value == nil`，并把不可安全推导的场景明确保守降级。

## 第一百二十五阶段：Go 字符串/布尔/nil 分支输入推导

1. [x] 固定字符串空值分支：`if name == "" { return "anonymous" }` 会生成 `nameValue: ""` 和精确期望。
2. [x] 固定字符串非空分支：`if name != "" { return name }` 会生成非空输入并将返回值替换为同一输入。
3. [x] 固定布尔分支：`if enabled == false { return "off" }` 会生成 `enabled: false` 和精确期望。
4. [x] 新增 nil literal 支持，`if user == nil { return "missing" }` 可为指针参数生成 `nil` 输入和精确期望。
5. [x] 为 `name` / `skip` 参数名避让测试表保留字段，生成 `nameValue` / `skipValue`，避免重复 struct 字段导致测试无法编译。
6. [x] 新增 Go generator 回归测试，覆盖字符串空值、字符串非空、布尔 false、nil pointer 和 `skip` 参数冲突。
7. [x] 使用临时 Go module 验证端到端：已有默认路径测试覆盖正常分支，连续生成 `Label`、`Toggle`、`UserName` 三个 coverage task，`run_tests` 返回 `passed=7`、`skipped=0`、`coverage_percent=100`。
8. [x] 补齐 nil 非空路径回归：`if user != nil { return "present" }` 会生成 `user: &User{}` 和精确期望；临时 Go module 中补测后 `run_tests ./pkg` 返回 `passed=3`、`skipped=0`、`coverage_percent=100`。

已完成补充：Go branch task 现在能覆盖更常见的业务条件分支，并且真实生成代码可编译运行。下一步建议处理错误路径，并修复本阶段验证时暴露的 Go `run_tests` 绝对路径 module 边界问题；错误路径可先评估 `if err != nil` 是否需要引入 `errors.New(...)` 和额外 import。

## 第一百二十六阶段：Go run_tests 绝对路径归一化

1. [x] 将 Go `run_tests` 命令构造抽成独立逻辑，保留 `-json`、`-v`、`-cover` 参数组合。
2. [x] 当输入路径位于 Go module 内时，向上查找 `go.mod`，将命令工作目录切到 module 根目录。
3. [x] 将 module 内绝对目录和绝对测试文件转换成 `./pkg` 形式的相对包路径，避免 `go test /abs/pkg` 触发 outside main module。
4. [x] 补充工具层单测，覆盖绝对包目录和绝对测试文件两种入口。
5. [x] 使用临时 Go module 端到端验证：从 testloop-mcp 仓库目录调用 `run_tests`，分别传入绝对包目录和绝对测试文件，均返回 `status=pass`、`coverage_percent=100`。

已完成补充：Go 测试执行现在更适合被 Agent 以绝对路径调用，尤其适合外部项目目录或用户直接传入源码/测试文件路径的场景。下一步建议继续回到 Go branch/error path 生成质量：先评估 `if err != nil` 的可控输入构造，再决定是否支持 `errors.New(...)` 及 import 自动补全。

## 第一百二十七阶段：Go error 分支输入推导

1. [x] 支持 error 参数的 nil 分支输入：`if err == nil { return "ok" }` 会生成 `err: nil` 和精确期望。
2. [x] 支持 error 参数的非 nil 分支输入：`if err != nil { return "failed" }` 会生成 `err: errors.New("test")` 和精确期望。
3. [x] 为生成测试文件增加按需 `errors` import 判断，仅当 seed case 实际使用 `errors.New(...)` 时引入。
4. [x] 补充 Go generator helper 和 coverage task 回归测试，覆盖 `err == nil`、`err != nil`、`errors` import。
5. [x] 使用临时 Go module 端到端验证：已有测试覆盖 `err == nil`，生成 `err != nil` 补测后，通过绝对路径 `run_tests` 返回 `passed=3`、`skipped=0`、`coverage_percent=100`。

已完成补充：Go branch task 已覆盖常见标量、字符串、bool、nil pointer、non-nil pointer、nil error 和 non-nil error 输入形态。下一步建议做“无法安全推导”的显式降级说明和任务上下文增强，例如 selector 返回值 `return user.Name`、多条件 `a > 0 && b > 0`、函数调用返回值等场景应明确保守生成 smoke/TODO，并在 context/payload notes 中告诉 Agent 缺什么信息。

## 第一百二十八阶段：Go 保守降级说明与 context 增强

1. [x] 增加 Go coverage task 降级原因分析：当 branch task 无法安全推导精确输入或期望值时，返回明确原因，而不是只生成裸 TODO。
2. [x] 生成测试 TODO case 中加入降级注释，例如 `return user.Name` 这类 selector 返回值会提示需要人工/Agent 复核 expected value。
3. [x] `BuildGenerationContextWithOptions` 支持 Go 源码解析，返回 Go target 的参数、返回类型、返回表达式和分支条件。
4. [x] 匹配 coverage task 的 Go target 会在 `payload_notes` 中携带同一条保守降级说明，方便 LLM provider 或调用方理解为什么静态生成没有直接给出精确断言。
5. [x] 补充单测覆盖生成代码 TODO 注释和 Go context payload notes。
6. [x] 使用临时 Go 文件端到端验证 `generate_tests`：`preview` 包含保守降级注释，`context.targets[0]` 包含 `params=["user *User"]`、`return_expressions=["\"missing\"", "user.Name"]`、`boundary_cases=["user != nil"]` 和 payload note。

已完成补充：静态生成器现在不仅能生成能跑的测试，也能在不能安全生成时给 Agent 明确的下一步线索。下一步建议做多条件分支识别的保守表达：先让 context 暴露 `a > 0 && b > 0` 这类条件原文和“compound condition unsupported”的 note，再决定是否支持有限的 `&&` 输入合成。

## 第一百二十九阶段：Go 复合条件分支保守表达

1. [x] Go boundary 提取会保留 `&&` / `||` 复合条件原文，例如 `a > 0 && b > 0`，用于 context 和降级说明。
2. [x] 复合条件 boundary 不参与精确 seed case 生成，避免误把多参数条件当作单参数条件合成。
3. [x] 生成测试 TODO case 会说明复合条件当前不支持多参数输入合成，并保守保持 `skip: true`。
4. [x] Go generation context 的 `boundary_cases` 会暴露复合条件原文，匹配 coverage task 的 target 会在 `payload_notes` 中带同一条说明。
5. [x] 补充生成器和 context 单测，覆盖 `a > 0 && b > 0` 的保守降级。
6. [x] 使用临时 Go 文件端到端验证 `generate_tests`：`preview` 中有 compound condition 降级注释，`context.targets[0]` 包含 `params=["a int","b int"]`、`return_expressions=["0","1"]`、`boundary_cases=["a > 0 && b > 0"]` 和 payload note。

已完成补充：复合分支现在不会在静态生成中静默丢失，Agent 可以从 context 中看到具体条件和当前限制。下一步建议做有限 `&&` 输入合成：仅支持每个子条件都是简单参数边界且返回表达式安全的场景，例如 `a > 0 && b > 0 { return 1 }` 生成 `a: 1, b: 1, ret0: 1`；`||` 和混合复杂表达式继续保守降级。

## 第一百三十阶段：Go 有限 && 复合条件输入合成

1. [x] 为 Go compound boundary 增加子条件列表和 compound operator，保留原始条件用于 context 与 hint 匹配。
2. [x] 支持 `&&` 复合条件精确 seed：每个子条件必须是简单参数边界，参数不能重复，且返回表达式必须安全。
3. [x] `a > 0 && b > 0 { return 1 }` 可生成 `a: 1`、`b: 1`、`ret0: 1`、`skip: false`。
4. [x] `||`、复杂子条件、重复参数或无法映射安全 literal 的子条件继续保守降级，并保留 payload note。
5. [x] 补充生成器和 context 回归测试，覆盖 `&&` 精确合成和 `||` 保守降级。
6. [x] 使用临时 Go module 端到端验证：已有默认分支测试覆盖 `Score(0,0)`，生成 `a > 0 && b > 0` 补测后，`run_tests` 返回 `passed=3`、`skipped=0`、`coverage_percent=100`。

已完成补充：Go branch task 已能覆盖一批实际业务里常见的多参数正向分支，同时保持复杂条件的保守边界。下一步建议处理重复参数或范围条件的有限支持，例如 `a > 0 && a < 10` 是否可以合成 `a: 1`；若暂不做，至少把重复参数的 note 文案和 context 测试补齐，避免 Agent 误判。

## 第一百三十一阶段：Go 简单整数范围条件合成

1. [x] 支持同一参数在 `&&` 中重复出现时按整数范围求交集，例如 `a > 0 && a < 10`。
2. [x] 可行范围会生成范围内输入，`a > 0 && a < 10 { return "inside" }` 生成 `a: 1`、`ret0: "inside"`、`skip: false`。
3. [x] 无交集范围，例如 `a > 10 && a < 5`，会保守降级，不生成伪精确用例。
4. [x] 非整数重复参数、无法解析 literal 或不支持的操作符继续保守降级，并在 note 中说明 repeated parameter 不在支持范围内。
5. [x] 补充 helper、generator 和 context 回归测试，覆盖可行范围、无交集范围和 supported range 无 payload note。
6. [x] 使用临时 Go module 端到端验证：已有外部分支测试覆盖 `InRange(0)`，生成 `a > 0 && a < 10` 补测后，`run_tests` 返回 `passed=3`、`skipped=0`、`coverage_percent=100`。

已完成补充：Go 分支输入合成已覆盖简单单条件、多参数 `&&`、以及单参数整数范围。下一步建议转向真实项目回归：用 `laoxia-scaffold` 的 `server` 目录跑一轮 parse coverage -> generate tests -> run tests，记录哪些任务能自动闭环，哪些仍需 Agent/人工补充。

## 第一百三十二阶段：laoxia server 真实项目回归

1. [x] 确认真实项目目录为 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server`，而不是 `server`；该目录是独立 Git 仓库，回归前工作区干净。
2. [x] 在真实项目原目录执行只读覆盖率命令：`go test ./... -coverprofile=/tmp/laoxia-server-cover.out`。命令通过，仅有 `gopsutil/disk` 在 macOS 上的 `IOMasterPort` deprecated warning；整体覆盖率约 `4.44%`，`utils` 包覆盖率约 `38.9%`。
3. [x] 用 `parse_coverage` 解析覆盖率，得到约 `1206` 个 coverage task；真实项目暴露出明显任务抽取噪声，例如部分任务把函数签名、普通语句片段或 `if` 前置赋值误当作分支条件。
4. [x] 使用临时拷贝验证 `AsJson` 的 `src == nil` branch task：生成 `TestAsJson`，`src: nil`、`ret0: ""`、`skip: false`；`run_tests ./utils` 返回 `status=pass`、`passed=22`、`skipped=0`、`coverage_percent=39.4`。
5. [x] 使用临时拷贝验证 `GetCurrentDate` 的 time.Date return_path task：生成非 skipped 时间边界断言，检查 hour/min/sec/nsec 归零；`run_tests ./utils` 返回 `status=pass`、`passed=22`、`skipped=0`、`coverage_percent=39.8`。
6. [x] 验证 `Ptr` 泛型指针 return_path 的当前边界：生成结果仍是 `skip: true` 的 TODO，用例不会失败但也不会提升覆盖；这是后续 generator 可改进点。
7. [x] 全程在临时目录生成测试，未修改真实 `car-admin-server` 仓库；真实项目保持干净。

已完成补充：真实项目证明当前工具已经能对简单 nil 分支和 time.Date 返回路径自动闭环，但 parse_coverage 的任务抽取质量会直接影响推荐任务可用性。下一步建议优先修 Go coverage task 抽取：不要从覆盖率行段的任意源码片段猜条件，改为基于 Go AST 定位包含该行段的 `if` / `switch` / `return`，过滤函数签名、普通语句和 `if` 前置赋值噪声。

## 第一百三十三阶段：coverage task 生成后验证工具

1. [x] 新增 `validate_coverage_task` MCP 工具，接收单个 `parse_coverage.test_tasks[]` 和源文件路径。
2. [x] 工具内部复用 `generate_tests` 写入任务推荐测试文件，再复用 `run_tests` 执行生成结果。
3. [x] 输出稳定的 `status` / `action`，区分 `passed`、`failed`、`generation_error` 和 `run_error`。
4. [x] 测试失败时默认开启 `include_fix_suggestions`，把失败原因和 repair task 带回给 Agent。
5. [x] 补充 handler 级测试，覆盖临时 Go module 中 coverage task 生成并执行通过，以及源文件缺失时返回 `generation_error`。
6. [x] README 和 Agent workflow 增加 `validate_coverage_task` 说明，推荐 Agent 对单个 coverage task 优先使用该闭环工具。

已完成补充：coverage task 现在不只停留在“可生成测试草稿”，还可以由 MCP 工具直接验证生成结果是否可运行。Agent 可以根据 `status/action` 做下一步决策：`passed/ready` 进入下一个任务或重新统计覆盖率；`failed/apply_fix_suggestions` 读取内联 repair task；`generation_error` 根据 `provider_error.action` 重试或降级；`run_error` 先修测试入口。

## 第一百三十四阶段：laoxia 前排 coverage task 批量验证

1. [x] 使用 `validate_coverage_task` 在 `laoxia-scaffold-v1.0.0/car-admin-server` 的临时副本中验证前 20 个高优先级 coverage task，避免修改真实项目。
2. [x] 初始批量验证发现 10 个 passed、9 个 failed、1 个 generation_error；failed 主要是生成测试编译失败。
3. [x] 修复 Go function type 参数渲染：`func(int) int` 不再退化为 `func()`，解决 `SliceMapper0` 这类泛型 mapper 测试编译失败。
4. [x] 修复 Go selector 类型 import 推导：参数或返回值里出现 `http.Request` / `time.Duration` 这类源码 import selector 时，生成测试会自动引入对应包。
5. [x] 修复未知命名类型零值：默认从 `Type{}` 改为 `*new(Type)`，避免 `time.Duration{}` 编译失败。
6. [x] 重新批量验证前 20 个任务，最终 19 个 `passed/ready`，仅 `GetRaw` 因目标测试文件已有 `TestGetRaw` 返回 `generation_error`。

已完成补充：`validate_coverage_task` 在真实 Go 项目的前排低依赖任务上已经能暴露并驱动生成器质量修复。当前剩余主要问题不是测试代码不可编译，而是已有测试函数同名时 `generate_tests` 保守返回错误。下一步建议处理 coverage task 的重复测试名策略，例如基于 task id 或行段追加稳定后缀，或让 `validate_coverage_task` 在检测到重复时返回更明确的 `action` 供 Agent 自动调整。

## 第一百三十五阶段：coverage task 重复测试名策略

1. [x] 在 Go coverage task 生成前读取目标测试文件已有 `Test*` 函数名。
2. [x] 当任务推荐 `test_name` 已存在时，复制任务并生成稳定后缀名称，优先使用覆盖率行段，例如 `TestGetRawCoverage204_207`。
3. [x] 将调整后的 coverage task 同步传给生成器、输出 `coverage_task` 和 generation context，保证 Agent 看到的名称就是实际写入的名称。
4. [x] 保留普通 Go 合并的重复函数保护，底层 `writeGeneratedTestFile` 遇到同名 `Test*` 仍返回明确错误。
5. [x] 补充 handler 回归测试：同一个 coverage task 第二次生成不再失败，而是追加稳定后缀测试函数。
6. [x] 使用 laoxia `go-test-20 / GetRaw` 真实样本验证：原 `TestGetRaw` 已存在时生成 `TestGetRawCoverage204_207`，`validate_coverage_task` 返回 `passed/ready`，`go test ./utils` 通过，新增用例为保守 skipped TODO。

已完成补充：前一阶段唯一的 `generation_error` 已被消除，laoxia 前 20 个高优先级任务从生成链路角度已全部可进入验证执行。下一步建议重新批量跑前 20 个任务形成最新统计，再从结果中挑选“passed 但 skipped/TODO”的样本，推进多返回值 error path 的可执行输入合成，优先解决 `GetRaw` / `GetBytes` 这类 `(*http.Response, error)` 分支仍只能保守跳过的问题。

## 第一百三十六阶段：Go URL error path 可执行输入合成

1. [x] 重新批量验证 laoxia 前 20 个高优先级 coverage task，确认上一阶段后已达到 20/20 `passed/ready`。
2. [x] 定位主要质量缺口：大多数任务虽然通过，但仍是 skipped/TODO；其中 `GetJson`、`GetBytes`、`GetRaw` 的 `err != nil` 分支可通过非法 URL 在本地稳定触发。
3. [x] Go boundary 提取支持 `return nil, err` 这类多返回表达式，保留 `ReturnExprs`，不再只看单返回表达式。
4. [x] Go static generator 增加 `error_path` 断言模式：当分支条件是 `err != nil`、返回值包含 `error`，且存在 `api` / `url` / `uri` / `endpoint` 等字符串参数时，生成 `"://invalid-url"` 输入。
5. [x] 对 `error` 返回值断言非 nil；对非 error 返回值断言 nil 或安全简单值；`[]byte` 等需要深比较的返回值会自动引入 `reflect`。
6. [x] 补充生成器回归测试，覆盖 `([]byte, error)` error path 生成非法 URL、非 skipped case、`reflect.DeepEqual` 和 `expected error, got nil` 断言。
7. [x] 使用 laoxia 真实样本验证：`GetBytes` 和 `GetRaw` 均返回 `passed/ready`、`skipped=0`；重新跑前 20 后仍为 20/20 `passed/ready`，0 skipped 任务从 1 个提升到 6 个，剩余 skipped 任务数为 14。

已完成补充：Go 静态生成器现在能把常见 HTTP/API error path 从 TODO 变成可执行测试，且不依赖外网请求成功。下一步建议处理 `RemoteIP` 这类 `*http.Request` 参数分支：基于 header / RemoteAddr 构造 `httptest.NewRequest` 或 `&http.Request{Header: http.Header{}, RemoteAddr: ...}`，把前 20 中 7 个 `RemoteIP` skipped 任务转成真实断言。

## 第一百三十七阶段：Go `*http.Request` 分支输入合成

1. [x] 定位 RemoteIP skipped 根因：当前 Go boundary 只扫函数顶层 `if`，无法识别 `for` 内部和嵌套分支，因此 `RemoteIP` 任务全部退化为 `no simple if boundary was detected`。
2. [x] 增加窄范围 `*http.Request` seed：仅在 coverage branch、返回值为 `string`、参数包含 `*http.Request` 时启用，避免泛化到未知复杂对象。
3. [x] 支持 `RemoteAddr` 正常路径：构造 `&http.Request{Header: http.Header{}, RemoteAddr: "203.0.113.9:1234"}`，断言返回 `203.0.113.9`。
4. [x] 支持 RemoteAddr 解析错误路径：构造 `RemoteAddr: "bad-remote-addr"`，断言返回原始 remote addr。
5. [x] 支持 `X-Forwarded-For` 路径：构造 header `X-Forwarded-For`，断言返回第一个 IP。
6. [x] 支持 `X-Real-IP` 路径：构造 canonical header key `X-Real-Ip`，避免 `http.Header.Get("X-Real-IP")` 查不到值。
7. [x] 明确保留 `partIndex < 0` 为 skipped：该源码分支在 `forwardedFor != ""` 条件下不可达，静态生成不伪造不可执行断言。
8. [x] 补充生成器回归测试，覆盖上述 4 类可达请求分支以及不可达 `partIndex < 0` 的保守降级。
9. [x] 使用 laoxia 真实样本验证：7 个 RemoteIP task 全部 `passed/ready`，其中 6 个 `skipped=0`，仅不可达 `partIndex < 0` 保留 skipped。
10. [x] 重新跑 laoxia 前 20：仍为 20/20 `passed/ready`，0 skipped 任务从 6 个提升到 12 个，剩余 skipped 任务数从 14 降到 8。

已完成补充：真实项目中 HTTP request/header 类分支已经能自动补出可执行断言。下一步建议转向剩余 skipped 中的 JSON/文件/泛型工具函数：优先处理 `AsJson`、`FromJson`、`FromJsonFile` 的 sonic marshal/unmarshal 和临时文件路径输入，预计能继续把前 20 的 4 个 JSON 相关 skipped 降下来。

## 第一百三十八阶段：Go JSON / 文件错误分支输入合成

1. [x] 定位剩余 JSON skipped：`AsJson` 的 marshal error 分支、`FromJson` 的 unmarshal error 分支、`FromJsonFile` 的 read file error 分支均可用稳定本地输入触发。
2. [x] 为 `AsJson` 的 slice/array marshal error 分支生成 `[]func(){func() {}}`，断言返回 `"[]"`。
3. [x] 为 `AsJson` 的非 slice marshal error 分支生成 `func() {}`，断言返回 `"{}"`。
4. [x] 为 `FromJson` 生成 `[]byte("{")` 和 `&map[string]any{}`，断言返回 error 非 nil。
5. [x] 为 `FromJsonFile` 生成不存在的 `testdata/does-not-exist.json` 路径，断言返回 error 非 nil。
6. [x] 补充 Go generator 回归测试，覆盖上述 4 类 JSON/文件错误分支，并固定不再生成 skipped TODO。
7. [x] 使用 laoxia 真实样本验证：`go-test-5` 到 `go-test-8` 全部 `passed/ready` 且 `skipped=0`。
8. [x] 重新跑 laoxia 前 20：仍为 20/20 `passed/ready`，0 skipped 任务从 12 个提升到 16 个，剩余 skipped 任务从 8 个降到 4 个。

已完成补充：JSON marshal/unmarshal 和文件读取错误路径已经能自动生成真实断言。下一步建议处理剩余 3 个 alias/generic/string-slice 工具函数：`SliceMapper0` 去重分支、`UserDurationOf` switch case、`TrimSpaceSlice` 非空分支；`RemoteIP partIndex < 0` 当前判断为不可达，可继续保守 skipped 或在 coverage task 层标注 unreachable。

## 第一百三十九阶段：Go alias/generic 工具函数分支输入合成

1. [x] 定位剩余 alias skipped：`SliceMapper0` 的去重分支、`UserDurationOf` 的 switch/case、`TrimSpaceSlice` 的非空分支都可用稳定纯函数输入触发。
2. [x] 为 `SliceMapper0` 生成重复输入 `[]int{1, 1, 2}` 和 identity mapper，断言输出 `[]int{1, 2}`，覆盖 `filter[ret]` 分支。
3. [x] 为 `TrimSpaceSlice` 生成 `[]string{" a ", " ", "b"}`，断言输出 `[]string{"a", "b"}`，覆盖 trim 后非空分支。
4. [x] 为 `UserDurationOf` 生成 `tpy: 5`，断言返回 `time.Hour * 24 * 365 * 99`，覆盖 switch/case 路径。
5. [x] 补充 Go generator 回归测试，覆盖上述 3 类工具函数分支，并固定不再生成 skipped TODO。
6. [x] 修正 coverage task condition hint 对 `switch/case` 文案的保留，让 switch/case 类任务可进入 seed 判断。
7. [x] 使用 laoxia 真实样本验证：`go-test-1` 到 `go-test-3` 全部 `passed/ready` 且 `skipped=0`。
8. [x] 重新跑 laoxia 前 20：仍为 20/20 `passed/ready`，0 skipped 任务从 16 个提升到 19 个，仅剩 `RemoteIP partIndex < 0` 一个不可达分支保守 skipped。

已完成补充：laoxia 前 20 个高优先级任务中，可达路径已经基本全部转成真实执行测试。下一步建议不要伪造 `partIndex < 0`，而是在 coverage task 生成或验证结果中增加 `unreachable` / `manual_review_unreachable` 标记，用静态规则识别这类明显不可达分支，避免 Agent 继续反复尝试。

## 第一百四十阶段：coverage task 不可达分支标记

1. [x] 在 `validate_coverage_task` 中增加不可达识别后处理：当生成结果是 TODO skip、测试运行通过且 task 命中明确不可达规则时，不再返回普通 `ready`。
2. [x] 对 laoxia 暴露的 `RemoteIP partIndex < 0` 增加窄规则：该分支来自非空 `X-Forwarded-For` parts slice 的 `len(parts)-1`，负数路径疑似不可达。
3. [x] validation 输出保持 `status: "passed"`，但将 `action` 改为 `manual_review_unreachable`，避免被当作失败，也避免 Agent 继续生成伪测试。
4. [x] 在 metadata 中返回 `unreachable: true` 和 `unreachable_reason`，给 Agent/人工复核足够上下文。
5. [x] 补充 handler 回归测试，固定不可达 skipped task 会输出 `manual_review_unreachable`。
6. [x] 使用 laoxia `go-test-13` 验证：输出 `passed/manual_review_unreachable`，metadata 带不可达原因。
7. [x] 重新跑 laoxia 前 20：20/20 `passed`，其中 19 个 `ready`、1 个 `manual_review_unreachable`，剩余 skipped 全部被解释为不可达复核项。

已完成补充：前 20 个高优先级任务已经从“生成/运行修复”推进到“可达任务全部 ready，不可达任务明确复核”。下一步建议扩大真实项目批量验证窗口，例如跑前 50 或按 `utils` 包全部 task，观察新的失败类型是否来自更复杂依赖、全局状态、文件系统或泛型边界。

## 第一百四十一阶段：laoxia top50 扩窗验证与 Go 可编译性修复

1. [x] 将 laoxia 临时基线验证窗口从前 20 扩大到前 50 个高优先级 coverage task，继续在独立临时副本中执行，避免修改真实项目。
2. [x] 首轮 top50 暴露 2 个 `failed/apply_fix_suggestions`：`TraceTransport.RoundTrip` 和 `init`；其余任务为 47 个 `ready`、1 个 `manual_review_unreachable`。
3. [x] 修复 Go 方法 receiver 命名冲突：当源码 receiver 名为 `t` / `tt` 或为空时，生成测试改用 `receiver`，避免覆盖 `func(t *testing.T)` 的测试参数。
4. [x] 保持普通 receiver 名兼容：`svc` / `s` 等非冲突 receiver 仍按原名生成，避免不必要改变既有输出。
5. [x] 修复 Go `init` coverage task：生成测试不再直接调用不可访问的 `init()`，而是输出明确的 manual-review skip。
6. [x] `validate_coverage_task` 对 `init` manual-review skip 返回 `passed/manual_review_unreachable`，并在 metadata 中写入不可直接调用的原因。
7. [x] 补充 generator 和 handler 回归测试，固定 receiver 冲突避让、`init` 不直接调用、以及 `init` validation 合同。
8. [x] 重新验证原失败样本：`TraceTransport.RoundTrip` 返回 `passed/ready`，`init` 返回 `passed/manual_review_unreachable`。
9. [x] 重新跑 laoxia top50：50/50 `passed`，其中 48 个 `ready`、2 个 `manual_review_unreachable`；总 skipped 30，0 skipped 任务 20 个，失败数为 0。

已完成补充：扩窗到前 50 后，生成测试的“能否编译并跑通”问题已被压到 0，剩余质量问题主要是大量 `passed/ready` 仍为 skipped TODO。下一步建议从 top50 的 skipped ready 样本里按收益排序，优先处理低依赖、可稳定构造输入的任务，例如 `ParseToken`、`Recover`、`InitDisk/InitCPU/InitRAM` 或 `GetBytes/GetJson` 的返回路径，继续提高 0 skipped 比例。

## 第一百四十二阶段：Go JWT ParseToken 成功分支输入合成

1. [x] 从 laoxia top50 的 skipped ready 样本中选取低依赖任务 `ParseToken`，目标分支为 `ok && tc.Valid`。
2. [x] 增加窄范围 `ParseToken` seed：仅当任务为 branch、函数名为 `ParseToken`、签名为 `string -> (*T, error)`，且 coverage hint 命中 `ok && tc.Valid` 时启用。
3. [x] 使用同包 `GenerateToken` 构造合法 token，并在输入表达式中设置 `global.Config.Jwt.Key` 与 `global.Config.Jwt.ExpireTime`，避免依赖外部配置文件。
4. [x] 增加 seed 表达式 import 推导：当 seed 输入或输出引用源码 import alias，例如 `global.Config`，生成测试会自动带上对应 import。
5. [x] 增加 `non_nil_result` 断言模式：对非 error 且可 nil 的返回值断言非 nil，对 error 返回值断言 nil，避免比较 JWT Claims 中的动态过期时间。
6. [x] 补充 Go generator 回归测试，固定 `ParseToken` 成功分支生成非 skipped 用例、引入 `global` import，并输出 claims 非 nil / error nil 断言。
7. [x] 使用 laoxia `go-test-22 / ParseToken` 真实样本验证：`validate_coverage_task` 返回 `passed/ready`，`skipped=0`。
8. [x] 重新跑 laoxia top50：50/50 `passed`，其中 48 个 `ready`、2 个 `manual_review_unreachable`；0 skipped 任务从 20 提升到 21，skipped 总数从 30 降到 29。

已完成补充：top50 中第一个认证类成功路径已经能自动生成真实断言，并验证了 seed 表达式依赖源码 import 的通用能力。下一步建议继续处理 top50 中低依赖 skipped ready：优先看 `Recover` 的 panic/recover 分支和 `GetJson` / `GetBytes` 的 return path；系统资源类 `InitCPU/InitRAM/InitDisk` 可后置，因为它们更容易受运行环境影响。

## 第一百四十三阶段：Go Recover panic 分支输入合成

1. [x] 从 laoxia top50 的 skipped ready 样本中选取低依赖任务 `Recover`，目标分支为 `p != nil`。
2. [x] 增加窄范围 `Recover` seed：仅当任务为 branch、函数名为 `Recover`、签名为 `cleanups ...func()`，且 coverage hint 命中 `p != nil` 时启用。
3. [x] 增加 `recover_panic` 断言模式：在测试中用内层闭包执行 `defer Recover(tt.cleanups...)` 后触发 `panic("test panic")`，用 recover 是否吞掉 panic 作为可运行断言。
4. [x] 为 variadic `cleanups` 生成 `[]func(){func() {}}` 输入，复用现有 `tt.cleanups...` 调用路径。
5. [x] 补充 Go generator 回归测试，固定 `Recover` panic 分支生成非 skipped 用例、`defer Recover(tt.cleanups...)` 和 `panic("test panic")`。
6. [x] 使用 laoxia `go-test-23 / Recover` 真实样本验证：`validate_coverage_task` 返回 `passed/ready`，`skipped=0`。
7. [x] 重新跑 laoxia top50：50/50 `passed`，其中 48 个 `ready`、2 个 `manual_review_unreachable`；0 skipped 任务从 21 提升到 22，skipped 总数从 29 降到 28。

已完成补充：panic/recover 类控制流已经能自动生成真实执行路径，top50 的纯本地低依赖 skipped 继续下降。下一步建议处理 `GetJson` / `GetBytes` 的 return path skipped，目标是复用已有 HTTP error path 能力，给 `return GetBytes(...)` / `return FromJson(...)` 这类包装返回路径生成稳定本地断言；系统资源类继续后置。

## 第一百四十四阶段：Go HTTP wrapper 本地 server 输入合成

1. [x] 从 laoxia top50 的 skipped ready 样本中选取 `GetJson` 和 `GetBytes`，目标分别为 JSON 解析错误路径和 body 成功返回路径。
2. [x] 为 seed 增加 `HTTPServerBody` 元数据；命中本地 HTTP wrapper 路径时，生成测试会自动引入 `net/http` 和 `net/http/httptest`。
3. [x] 在 `t.Run` 内启动 `httptest.NewServer`，写入固定响应 body，并将 URL 注入到 `tt.api`，避免外网依赖和全局 client 改造。
4. [x] `GetJson` error path 使用非法 JSON body `{`，配合 `&map[string]any{}` 触发解析错误，并断言 error 非 nil。
5. [x] `GetBytes` return path 使用固定 body `test-body`，断言返回 `[]byte("test-body")` 且 error 为 nil。
6. [x] 补充 Go generator 回归测试，固定 `httptest` import、server setup、URL 注入、error path 断言和 body DeepEqual 断言。
7. [x] 使用 laoxia `go-test-29 / GetJson` 和 `go-test-35 / GetBytes` 真实样本验证：两者均返回 `passed/ready`，`skipped=0`。
8. [x] 重新跑 laoxia top50：50/50 `passed`，其中 48 个 `ready`、2 个 `manual_review_unreachable`；0 skipped 任务从 22 提升到 24，skipped 总数从 28 降到 26。

已完成补充：HTTP wrapper 类任务已经可以在无外网、无全局 client patch 的条件下生成真实执行测试。下一步建议重新盘点 top50 剩余 skipped ready，优先处理仍低依赖的 `Ptr`、`UserTypeOf`、剩余 `SliceMapper0` / `TrimSpaceSlice` return path；模型 `BeforeSave` 和系统资源类任务暂时后置，因为它们更可能需要业务结构或运行环境上下文。

## 第一百四十五阶段：Go 纯工具函数 return/statement 路径输入合成

1. [x] 重新盘点 laoxia top50 剩余 skipped ready，优先选择低依赖纯函数任务：`SliceMapper0`、`TrimSpaceSlice` 和 `UserTypeOf`。
2. [x] 扩展 `goAliasUtilitySeedTestCase`，不再只处理 branch gap；对 `return_path` 和 `statement` gap 也允许使用可观察输出断言。
3. [x] `SliceMapper0` 的 return/statement path 复用重复输入 `[]int{1, 1, 2}` 和 identity mapper，断言去重后结果为 `[]int{1, 2}`。
4. [x] `TrimSpaceSlice` 的 return/statement path 复用 `[]string{" a ", " ", "b"}`，断言输出 `[]string{"a", "b"}`。
5. [x] `UserTypeOf` 的默认 return path 使用 `time.Minute`，断言返回 `1`。
6. [x] 补充 Go generator 回归测试，覆盖 branch、return_path、statement 三类 gap 下的 alias 工具函数生成。
7. [x] 使用 laoxia 真实样本验证：`go-test-30`、`go-test-32`、`go-test-33`、`go-test-47`、`go-test-48`、`go-test-49` 全部返回 `passed/ready`，`skipped=0`。
8. [x] 重新跑 laoxia top50：50/50 `passed`，其中 48 个 `ready`、2 个 `manual_review_unreachable`；0 skipped 任务从 24 提升到 30，skipped 总数从 26 降到 20。

已完成补充：top50 中纯工具函数的可达 skipped 已基本清掉，剩余 skipped 集中在四类：`Ptr` 需要指针值断言模式，`RemoteIP` 仍有返回/不可达路径，`InitCPU/InitRAM/InitDisk` 依赖运行环境，模型 `BeforeSave` 依赖业务结构。下一步建议优先处理 `Ptr` 的泛型指针返回断言，因为它仍是低依赖纯函数；随后再决定是否进入 `BeforeSave` 或系统资源类。

## 第一百四十六阶段：Go Ptr 泛型指针返回断言

1. [x] 从 laoxia top50 剩余 skipped ready 中选择低依赖纯函数 `Ptr` 的 `return_path` 作为本阶段目标。
2. [x] 为 Go 静态生成器新增 `pointer_value` 断言模式，避免为 `*T` 返回值生成 `ret0 *int` 这类不可稳定比较的期望字段。
3. [x] 将 `Ptr` seed 限定在 `return_path`、函数名为 `Ptr`、单参数、单指针返回值的安全边界内，避免泛化到语义不确定的任意指针返回函数。
4. [x] 生成测试使用 `Ptr[int](tt.v)`，先断言返回值非 nil，再断言 `*got == tt.v`。
5. [x] 补充 Go generator 回归测试，固定 `skip: false`、输入值、泛型调用、nil 检查、解引用比较，并防止退化为 `ret0 *int` 或指针地址比较。
6. [x] 使用 laoxia `go-test-31 / Ptr` 真实样本验证：返回 `passed/ready`，`skipped=0`。
7. [x] 重新跑 laoxia top50：50/50 `passed`，其中 48 个 `ready`、2 个 `manual_review_unreachable`；0 skipped 任务从 30 提升到 31，skipped 总数从 20 降到 19。

已完成补充：`Ptr` 这类低依赖泛型指针工具函数现在能生成真实行为断言，不再只是 skipped TODO。下一步建议优先处理 `RemoteIP` 剩余可达 return path，并继续保留明显不可达的 `partIndex < 0` 为 `manual_review_unreachable`；系统资源类和模型 `BeforeSave` 后置。

## 第一百四十七阶段：Go RemoteIP return/statement 路径输入合成

1. [x] 复核 laoxia top50 剩余 `RemoteIP` skipped：`go-test-34` 为第 150 行 `return fallback`，`go-test-50` 为函数入口和 header 读取语句块。
2. [x] 确认 `return fallback` 在 laoxia 当前源码中不能靠空 header 触发，因为包级 `ipLookups` 默认包含 `RemoteAddr`；需要在测试中临时覆盖 `ipLookups`。
3. [x] Go AST 解析增加包级变量名记录，让窄规则只有在源码中确实存在 `ipLookups` 时才生成覆盖代码。
4. [x] `RemoteIP return_path` 生成 `oldIPLookups := ipLookups`、`ipLookups = []string{"Unknown"}` 和 `t.Cleanup(...)`，断言返回 `fallback`。
5. [x] `RemoteIP statement` 使用普通 `RemoteAddr` 请求对象，断言返回解析出的 IP，从而覆盖函数入口和 header 读取语句。
6. [x] 补充 Go generator 回归测试，固定 fallback setup、cleanup、RemoteAddr statement 输入和非 skipped 输出。
7. [x] 使用 laoxia `go-test-34 / RemoteIP` 与 `go-test-50 / RemoteIP` 真实样本验证：两者均返回 `passed/ready`，`skipped=0`，对应生成测试单独执行 `go test ./utils` 通过。
8. [x] 重新跑 laoxia top50：50/50 `passed`，其中 48 个 `ready`、2 个 `manual_review_unreachable`；0 skipped 任务从 31 提升到 33，skipped 总数从 19 降到 17。

已完成补充：`RemoteIP` 的可达 skipped 已清掉，剩余 `RemoteIP partIndex < 0` 继续作为不可达复核项。下一步建议优先处理 `FromJsonFile` 第 39 行成功返回路径：用临时 JSON 文件触发 `FromJsonFile` happy path 并断言 error 为 nil；`TraceTransport`、系统资源类和模型 `BeforeSave` 后置。

## 第一百四十八阶段：Go FromJsonFile 成功返回路径输入合成

1. [x] 定位 laoxia top50 剩余低依赖 skipped：`go-test-28 / FromJsonFile` 覆盖第 39 行 `return nil`，当前 task 虽被标成 `error_path`，但实际需要 happy path 输入。
2. [x] 将 JSON seed 的入口限制从全局 `branch` 下沉到具体 case，保留 `AsJson` / `FromJson` 的 branch 限制，同时允许 `FromJsonFile` 处理 success/error path。
3. [x] 新增 success return path 判断：优先识别 `return_path`，并兼容 assertion focus 中包含“返回路径”但不包含 `err != nil` 的 coverage task。
4. [x] `FromJsonFile` success path 生成 `tt.path == ""` 时的临时 JSON 文件 setup，使用 `t.TempDir()` 和 `os.WriteFile` 写入 `{"ok":true}`。
5. [x] 自动把 setup 中的 `os.WriteFile` 纳入 seed import 推导，生成测试会补 `os` import。
6. [x] 补充 Go generator 回归测试，固定 `os` import、临时 JSON 文件 setup、`FromJsonFile(tt.path, tt.dst)` 调用和 error 为 nil 断言。
7. [x] 使用 laoxia `go-test-28 / FromJsonFile` 真实样本验证：返回 `passed/ready`，`skipped=0`，生成测试单独执行 `go test ./utils` 通过。
8. [x] 重新跑 laoxia top50：50/50 `passed`，其中 48 个 `ready`、2 个 `manual_review_unreachable`；0 skipped 任务从 33 提升到 34，skipped 总数从 17 降到 16。

已完成补充：JSON/文件类低依赖 skipped 已继续收窄，剩余 skipped ready 主要集中在 `TraceTransport.RoundTrip`、`InitDisk/InitCPU/InitRAM` 和模型 `BeforeSave`。下一步建议先评估 `TraceTransport.RoundTrip`，看是否能用本地 `http.NewRequest`、stub `RoundTripper` 和较低 `SlowThreshold` 稳定覆盖 defer 中的慢请求日志路径；如果需要依赖日志副作用或 httptrace 时序不可控，再转向模型 `BeforeSave`。

## 第一百四十九阶段：Go TraceTransport 慢请求分支输入合成

1. [x] 定位 laoxia top50 剩余低依赖 skipped：`go-test-21 / TraceTransport.RoundTrip` 覆盖 defer 中的 `totalCost > t.SlowThreshold` 分支。
2. [x] 确认当前通用 branch seed 因目标是方法且返回 `(*http.Response, error)` 多返回值而保守 skipped。
3. [x] 新增窄范围 method seed：只匹配 `TraceTransport.RoundTrip`、单个 `*http.Request` 参数、`(*http.Response, error)` 返回值和 `totalCost > t.SlowThreshold` 条件提示。
4. [x] 生成测试使用本地 `httptest.NewServer`，handler sleep 1ms 并返回 `204`，避免外网依赖。
5. [x] 生成测试设置 `receiver.Transport = http.DefaultTransport` 和 `receiver.SlowThreshold = -time.Nanosecond`，稳定触发 defer 中的 slow branch。
6. [x] 复用 `non_nil_result` 断言模式，断言 response 非 nil、error 为 nil，避免对日志副作用做脆弱断言。
7. [x] 补充 Go generator 回归测试，固定 `httptest` / `time` import、server setup、request 构造、receiver 配置和非 skipped 输出。
8. [x] 使用 laoxia `go-test-21 / TraceTransport.RoundTrip` 真实样本验证：返回 `passed/ready`，`skipped=0`，生成测试单独执行 `go test ./utils` 通过。
9. [x] 重新跑 laoxia top50：50/50 `passed`，其中 48 个 `ready`、2 个 `manual_review_unreachable`；0 skipped 任务从 34 提升到 35，skipped 总数从 16 降到 15。

已完成补充：top50 中低依赖 utility / HTTP / JSON / trace 类 skipped 已基本清理完。下一步建议评估系统资源类 `InitDisk`、`InitCPU`、`InitRAM` 是否能用稳定断言覆盖；如果这些依赖运行环境且断言价值有限，就转向模型 `BeforeSave`，优先找可无数据库执行的字段归一化/默认值逻辑。

## 第一百五十阶段：系统资源错误分支环境依赖标记

1. [x] 复核 laoxia top50 剩余系统资源 skipped：`InitDisk`、`InitCPU`、`InitRAM` 的目标均为 `err != nil` 分支。
2. [x] 确认这些错误分支直接依赖 `disk.Usage("/")`、`cpu.Counts`、`cpu.Percent` 和 `mem.VirtualMemory` 的 OS/runtime 错误；当前源码没有依赖注入点，静态测试不能稳定构造。
3. [x] 保持不伪造 happy-path 测试来冒充 error branch 覆盖，避免把 coverage task 目标语义做偏。
4. [x] 在 `validate_coverage_task` 后处理中新增 `manual_review_environment` action，当 skipped TODO 命中系统资源错误分支时从普通 `ready` 队列中分离。
5. [x] 在 metadata 中返回 `environment_dependent: true` 和 `environment_reason`，分别说明 `InitDisk`、`InitCPU`、`InitRAM` 的不可静态构造原因。
6. [x] 补充 handler 回归测试，固定系统资源错误分支会输出 `passed/manual_review_environment`，并保留 skipped 结果供人工复核。
7. [x] 使用 laoxia `go-test-24`、`go-test-25`、`go-test-26` 真实样本验证：三者均返回 `passed/manual_review_environment`，metadata 带环境依赖原因。
8. [x] 重新跑 laoxia top50：50/50 `passed`，其中 45 个 `ready`、2 个 `manual_review_unreachable`、3 个 `manual_review_environment`；0 skipped 任务保持 35，skipped 总数保持 15，剩余 skipped ready 全部集中在模型 `BeforeSave`。

已完成补充：系统资源类错误分支已经从普通待处理队列中剥离，后续不会反复尝试无法静态构造的 OS/runtime 错误。下一步建议转向模型 `BeforeSave`，优先分析 `Role`、`Menu`、`DictItem`、`User` 等方法是否只是字段默认值、路径归一化或排序值填充；如果可无数据库执行，就为这些方法生成非 skipped 单元测试。

## 第一百五十一阶段：Go BeforeSave receiver 字段变更断言

1. [x] 复核 laoxia top50 剩余模型 `BeforeSave` skipped：`Role`、`Menu`、`DictItem`、`User`、`Dept`、`Config`、`DictType`、`GlobalWhiteIp` 均为字段 trim 或默认值填充逻辑。
2. [x] 确认这些方法签名统一为 `BeforeSave(*gorm.DB) error`，且当前逻辑不依赖真实数据库连接，可以用 `nil` tx 直接调用。
3. [x] 为 Go 静态生成器新增 `receiver_mutation` 断言模式，支持调用方法后断言 receiver 字段被归一化。
4. [x] `Role.DataScope` 和 `Menu.Type` 使用空白输入触发默认值分支，避免只覆盖普通 trim 路径。
5. [x] 普通 trim-only 方法生成字段赋值、调用和逐字段断言，同时断言返回 error 为 nil。
6. [x] 补充 Go generator 回归测试，固定 `Role.BeforeSave` 默认值分支和 `User.BeforeSave` trim-only 路径均生成 `skip: false`，并自动引入 `gorm.io/gorm`。
7. [x] 本仓库验证通过：`go test ./internal/generator -run 'TestGenerateGoTestsForCoverageTaskAssertsBeforeSave'`、`go test ./...`、`git diff --check`。
8. [x] 使用 laoxia 隔离副本重新跑 top50：50/50 `passed`，其中 45 个 `ready`、2 个 `manual_review_unreachable`、3 个 `manual_review_environment`；0 skipped 任务从 35 提升到 45，skipped 总数从 15 降到 5，`skipped_ready=[]`。

已完成补充：laoxia top50 中所有可静态构造的 Go coverage task 已经转成真实执行测试；剩余 5 个 skipped 均有明确分类，不再属于普通生成质量缺口。下一步建议把这轮真实项目回归沉淀为发布前质量证据：补一份 v0.4.14 发布说明草稿，说明 top50 指标、人工复核分类和 Go 生成器新增场景；随后检查是否需要把隔离验证脚本沉淀为正式开发脚本，避免依赖一次性临时测试。

## 第一百五十二阶段：v0.4.14 发布说明草案

1. [x] 新增 `docs/plan-release-notes-v0.4.14.md`，归纳 v0.4.13 之后的 Go coverage task 闭环质量增强。
2. [x] 明确 v0.4.14 候选范围是 `validate_coverage_task`、skipped task 分类、Go static generator 多场景 seed 和 laoxia top50 真实项目验证。
3. [x] 记录 laoxia 隔离 top50 最新指标：50/50 `passed`，45 个 `skipped=0`，剩余 5 个分别归类为 `manual_review_unreachable` 或 `manual_review_environment`，`skipped_ready=[]`。
4. [x] 明确本轮仍不改变默认 provider 策略，不把复杂业务断言交给静态生成器强行猜测。
5. [x] 记录正式发布前待执行项，包括版本号、CHANGELOG 收敛、安装文档版本引用、构建、资产 dry run、Release Artifacts、资产校验和 Homebrew tap 核验。

已完成补充：v0.4.14 候选发布说明已经建立，当前只做发布资料准备，没有提前切版本号或改安装文档。下一步建议把 laoxia top50 隔离验证沉淀为正式开发脚本或测试辅助命令，避免后续只能靠一次性临时测试复现真实项目指标。

## 第一百五十三阶段：Go coverage top task 隔离验证脚本化

1. [x] 新增 opt-in 集成测试 `TestValidateGoCoverageTopTasks`，通过 `TESTLOOP_VALIDATE_GO_PROJECT_DIR` 指定真实 Go 项目；未设置时默认 skip，不影响常规测试。
2. [x] 集成测试会复制项目到 baseline worktree，运行 `go test ./... -coverprofile`，再解析 top coverage tasks。
3. [x] 每个 task 都会复制一份新的隔离 worktree 后调用 `validate_coverage_task`，避免前一个任务生成的测试污染后续 skipped 统计。
4. [x] 支持 `TESTLOOP_VALIDATE_GO_TASK_LIMIT` 和 `TESTLOOP_VALIDATE_GO_OUTPUT`，输出 JSONL 结果并打印 status/action/zero-skip/skipped summary。
5. [x] 新增 `scripts/validate-go-coverage-top-tasks.sh <go-project-dir> [limit] [output-jsonl]`，封装环境变量和测试入口。
6. [x] 使用 laoxia server 跑前 5 个任务验证脚本入口：5/5 `passed`，5 个 `skipped=0`。
7. [x] 使用 laoxia server 跑 top50 验证脚本化指标：50/50 `passed`，45 个 `skipped=0`，2 个 `manual_review_unreachable`，3 个 `manual_review_environment`，`skipped_total=5`。
8. [x] 更新 `docs/plan-release-notes-v0.4.14.md`，把脚本化验证纳入候选发布说明和回归保护。

已完成补充：真实项目 top task 验证现在有可重复入口，不再依赖一次性临时测试。下一步建议进入 v0.4.14 正式版本准备前检查：确认是否现在就切版本号和收敛 `CHANGELOG.md`，或先继续补更多真实项目样本以降低 laoxia 单样本偏差。

## 第一百五十四阶段：v0.4.14 版本准备改动

1. [x] 确认 v0.4.14 候选范围已经聚焦在 Go coverage task 闭环质量、真实项目验证和 skipped task 分类，具备切版本准备的完整性。
2. [x] 更新 `main.go` MCP implementation version 到 `0.4.14`。
3. [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.4.14 - 2026-07-11`，并补入 Go coverage top task 验证脚本。
4. [x] 同步 README 中当前 Release、手动下载示例、Windows 下载示例到 `v0.4.14`。
5. [x] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.4.14`。
6. [x] 新增 `docs/plan-release-v0.4.14.md`，记录版本准备、验证项和正式发布前待办。
7. [x] 更新 `docs/plan-release-notes-v0.4.14.md`，标记版本准备项已完成。
8. [x] 完成本地发布前验证：脚本语法、actionlint、示例脚本测试、安装脚本测试、release asset 脚本测试、`go test ./...`、主服务/CLI 构建、help 输出、darwin arm64 打包 dry run、sha256 校验和 tarball 内容检查均通过。

已完成补充：v0.4.14 的版本号和用户安装文档已切到新版本，发布检查清单已建立，本地发布前验证已通过。下一步需要提交版本准备改动并确认远端 CI 通过，然后进入 tag、Release Artifacts、资产校验、GitHub Release 正文和 Homebrew tap 发布核验阶段。

## 第一百五十五阶段：v0.4.14 正式发布和发布后核验

1. [x] 版本准备提交 `b58b99a` 已推送，远端 CI run `29157660797` passed。
2. [x] 创建并推送 tag `v0.4.14`，GitHub Release 已创建为 `testloop-mcp v0.4.14`。
3. [x] Release Artifacts run `29157722825` passed，五个平台资产和对应 `.sha256` 已上传。
4. [x] `scripts/verify-release-assets.sh v0.4.14` 验证 release 页面包含 10 个必需资产。
5. [x] GitHub Release 正文已更新为正式 v0.4.14 发布说明。
6. [x] Homebrew tap 已升级到 `0.4.14`，tap commit `6394533b9f999bd2125efab6ace6f3c1e81da180` 已推送。
7. [x] Homebrew 本地验证通过：`brew fetch --force --formula`、`brew audit --formula --strict`、`brew upgrade --formula`、`brew test`。
8. [x] Post-Release Verify run `29157901152` passed，资产清单和五平台安装脚本 dry run 全部通过。
9. [x] 发布记录提交 `42e5a85` 的 main CI run `29158887713` passed。

已完成补充：v0.4.14 已完成正式 GitHub Release、Homebrew 发布核验和 Post-Release Verify 五平台安装脚本 dry run。下一步建议回到真实项目样本扩展：选择第二个 Go 或 JS/TS 项目，跑通 `coverage_task -> generate_tests -> run_tests -> repair_task`，把 laoxia 单样本之外的失败样本沉淀为回归测试。

## 第一百五十六阶段：第二个真实 Go 项目样本验证

1. [x] 从本机 Go 项目候选中筛选第二样本，排除需要缺失 FFI 产物的 `apk-info/go`，选择可本地 `go test ./...` 通过的 `/Users/binlee/code/open-source/lazy-mcp-wrapper`。
2. [x] 首轮运行 `scripts/validate-go-coverage-top-tasks.sh /Users/binlee/code/open-source/lazy-mcp-wrapper 20 /tmp/testloop-lazy-mcp-wrapper-top20.jsonl`，暴露出 `QueryStatus` 任务生成 `client_test.go` 后与同包 `daemon_test.go` 既有 `TestQueryStatus` 重名，导致包级构建失败。
3. [x] 修复 Go coverage task 推荐测试名避让：从只扫描目标测试文件，扩展为扫描同目录所有 `*_test.go`。
4. [x] 补充 handler 回归测试，固定同包其它测试文件已存在推荐 `Test*` 时会追加行段后缀，例如 `TestAddCoverageTaskCoverage2_2`。
5. [x] 复跑 lazy top20 后确认 20/20 `passed`，但 20 个任务仍全为 skipped ready，说明第二样本主要缺口转向多返回值 error 分支和 Unix socket 协议交互。
6. [x] 增强 Go static generator：支持普通参数校验触发的多返回值 error 分支，并识别 `Status{}` 这类空 composite literal 零值返回。
7. [x] 补充 generator 回归测试，固定 `QueryStatus(socketPath string) (Status, error)` 的 `socketPath == ""` 分支会生成 `skip: false`、零值 `Status` 断言和 error 非 nil 断言。
8. [x] 再次复跑 lazy top20：20/20 `passed`，`zero_skip` 从 0 提升到 3，`skipped_total` 从 20 降到 17；已转成真实测试的任务包括 `RunClient` 的空参数错误分支和 `QueryStatus` 的空 `socketPath` 错误分支。

已完成补充：第二真实样本证明 v0.4.14 后的验证脚本能发现 laoxia 未覆盖的问题类型：同包跨文件测试名冲突，以及普通参数校验下的多返回值 error 分支。下一步建议继续沿 lazy 样本推进，优先处理剩余 `RunClient` / `QueryStatus` / `SendControl` skipped ready 中可用本地 Unix socket 或 `net.Pipe` 稳定构造的协议错误路径；对需要完整 daemon 协议上下文的任务则考虑给 `manual_review_protocol` 之类的动作分类，而不是长期停留在普通 ready skipped。

## 近期完成标准

第一个有价值的里程碑是：

- [x] `run_tests` 能从 `go test -json` 返回可靠的结构化 Go 失败信息
- [x] 旧版 Go 文本解析仍然可用
- [x] parser 测试覆盖 JSON 和文本输出
- [x] 已知 demo 生成测试即使失败，也能被准确报告
