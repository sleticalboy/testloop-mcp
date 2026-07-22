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
2. [x] 补充 `generate_tests -> run_tests` handler 级闭环测试，固定 Vitest 生成 `src/sum.test.ts` 后会以 `vitest run src/sum.test.ts` 执行。
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
9. [x] 放开变参函数的 branch/error-path seed，保留通用纯函数 seed 的变参保守边界，避免 `SendControl(socketPath, control string, opts ...ControlOptions)` 这类参数校验分支被提前跳过。
10. [x] 补充 generator 回归测试，固定变参多返回值 error 分支会生成 `opts []ControlOptions`、`tt.opts...` 调用、零值 response 断言和 error 非 nil 断言。
11. [x] 再次复跑 lazy top20：20/20 `passed`，`zero_skip` 从 3 提升到 5，`skipped_total` 从 17 降到 15；`SendControl` 的空 `socketPath` 和空 `control` 分支已转成真实测试。
12. [x] 增强 Go 分支匹配：为 AST `if` 边界记录源码行段和前置错误来源，coverage task 优先按 `line_range` 区分同一函数内重复的 `err != nil` 分支。
13. [x] 支持 `net.Dial("unix", socketPath)` 连接失败错误路径输入合成：生成 `filepath.Join(t.TempDir(), "missing.sock")`，并自动引入 `path/filepath`。
14. [x] 补充 generator 回归测试，固定首个 `net.Dial` 错误分支会生成非 skipped 测试，而后续 `conn.Write` 等协议错误分支不会被误当成缺失 socket 测试。
15. [x] 再次复跑 lazy top20：20/20 `passed`，`zero_skip` 从 5 提升到 8，`skipped_total` 从 15 降到 12；`RunClient`、`QueryStatus`、`SendControl` 的 `net.Dial` 连接失败分支已转成真实测试，剩余 skipped ready 集中在协议写入、读取、JSON 解码和响应默认错误分支。
16. [x] 支持 Unix socket 协议错误路径输入合成：测试内启动本地 `net.Listen("unix", ...)`，读完请求后关闭连接触发 `ReadBytes` EOF，或返回非法 JSON 触发 `json.Unmarshal` 错误。
17. [x] 修复 Go 生成测试写入路径：新建和合并 Go 测试文件都统一执行 import 整理，避免只生成单个目标测试时保留未使用 import，例如 lazy `RunClient` / `SendControl` 场景里的 `io`。
18. [x] 补充 generator 和 handler 回归测试，固定 Unix socket ReadBytes/JSON 错误分支生成非 skipped 用例，并固定 Go 测试文件新建/合并都会清理 unused import。
19. [x] 再次复跑 lazy top20：20/20 `passed`，`zero_skip` 从 8 提升到 13，`skipped_total` 从 12 降到 7；`RunClient`、`QueryStatus`、`SendControl` 的 `ReadBytes` 和 `json.Unmarshal` 错误分支已转成真实测试。
20. [x] 支持 Unix socket JSON 响应分支输入合成：用本地 socket 返回 `{"ok":false}` 或 `{}`，覆盖 `RunClient` 的 bind 默认错误、`SendControl` 的 control 默认错误和 `QueryStatus` 的 invalid status 复合分支。
21. [x] 将需要 `net.Listen("unix", ...)` 的测试 socket 文件名缩短为 `s`，避开 macOS Unix socket 路径长度限制；保留纯 `net.Dial` 缺失路径分支的 `missing.sock` 语义。
22. [x] 再次复跑 lazy top20：20/20 `passed`，`zero_skip` 从 13 提升到 16，`skipped_total` 从 7 降到 4；剩余 skipped ready 集中在 `RunClient` / `QueryStatus` / `SendControl` 的 `conn.Write` 错误分支和 `RunClient` 后续 `io.Copy` 错误分支。
23. [x] 为 skipped TODO 的 socket write / streaming I/O 错误分支增加 `manual_review_protocol` 分类和 metadata，避免这类时序敏感协议缺口继续以普通 `ready` 暴露。
24. [x] 补充 validate handler 回归测试，固定 `QueryStatus` 的 socket write 错误分支会返回 `manual_review_protocol`、`protocol_dependent=true` 和 protocol reason。
25. [x] 再次复跑 lazy top20：20/20 `passed`，`zero_skip=16`，`skipped_total=4`，`action_counts={"ready":16,"manual_review_protocol":4}`，`skipped_ready` 已清空。

已完成补充：第二真实样本证明 v0.4.14 后的验证脚本能发现 laoxia 未覆盖的问题类型：同包跨文件测试名冲突、普通参数校验下的多返回值 error 分支、变参函数的参数校验分支、同函数重复 `err != nil` 分支定位问题、协议错误路径输入合成、Go import 合并清理问题、平台相关的 Unix socket 路径长度限制，以及时序敏感协议分支的动作分类问题。下一步建议切换到第三个真实项目样本，验证当前 Go coverage task 闭环是否还能发现新的缺口类型；如果继续深挖 lazy，则应先设计可注入 fake conn 的重构方案，而不是在现有生产代码形态上用竞态触发 `conn.Write` / `io.Copy` 错误。

## 第一百五十七阶段：第三个真实 Go 项目样本验证

1. [x] 从本机 Go 项目候选中筛选第三样本，确认 `/Users/binlee/code/free-works/QuickSmoke-Backend-Go` 可本地 `go test ./...` 通过；同时排除存在 import cycle 的 `phone-filter` 和本地测试 panic 的 `ip2region/binding/golang`。
2. [x] 首轮运行 `scripts/validate-go-coverage-top-tasks.sh /Users/binlee/code/free-works/QuickSmoke-Backend-Go 20 /tmp/testloop-quicksmoke-top20.jsonl`，确认 top20 全部 `passed/ready`，但 `zero_skip=0`、`skipped_total=20`，主要缺口集中在泛型 helper、nil receiver、JWT parse、log 初始化、Gin response 和 MySQL repo 分支。
3. [x] 增强 Go static generator：支持 `return &param` 指针值断言，以及 `param == nil { return *new(T) }` / `return *param` 这类泛型指针解引用分支和返回路径。
4. [x] 补充 generator 回归测试，固定 `anyPtr[T]`、`derefAny[T]` 的 nil 分支和非 nil 返回路径均生成 `skip: false`，并使用 `anyPtr[int]` / `derefAny[int]` 调用。
5. [x] 复跑 QuickSmoke top20：20/20 `passed`，`zero_skip` 从 0 提升到 3，`skipped_total` 从 20 降到 17。
6. [x] 增强 Go static generator：支持 nil pointer receiver 的字符串分支，通过 setup 把 receiver 置为 nil 后断言返回值，例如 `(*BizError).Error()`。
7. [x] 补充 generator 回归测试，固定 `e == nil` 分支生成 `e = nil`、`ret0: ""` 和非 skipped 断言。
8. [x] 复跑 QuickSmoke top20：20/20 `passed`，`zero_skip` 从 3 提升到 4，`skipped_total` 从 17 降到 16。
9. [x] 增强 Go static generator：支持 JWT `Parse(secret, raw)` 的错误签名算法和非法 token 分支，生成 `jwt.SigningMethodHS256` token 或 `"not-a-token"` 输入，并断言 error 非 nil。
10. [x] 修复 Go import alias 解析：对 `github.com/.../pkg/vN` 这类语义版本路径，除 `vN` 外也记录上级目录包名，确保 seed 中的 `jwt.` 能自动补测试文件 import。
11. [x] 补充 generator 回归测试，固定 JWT Parse 两类错误分支的 import、输入和 error-path 断言。
12. [x] 复跑 QuickSmoke top20：20/20 `passed`，`zero_skip` 从 4 提升到 6，`skipped_total` 从 16 降到 14；剩余 skipped ready 主要集中在 log 初始化、Gin response 和 MySQL repo 分支。
13. [x] 增强 Go static generator：支持 `FailWithErr(*gin.Context, error)` 这类 Gin response helper，生成 `httptest.ResponseRecorder`、`gin.CreateTestContext`、JSON 解析和 response code/message/success 断言。
14. [x] 为 seed 增加显式 import alias 支持，修复 QuickSmoke 中业务 `berr "quicksmoke/backendgo/internal/errors"` 被写成无 alias import 后导致生成测试不可编译的问题。
15. [x] 补充 generator 回归测试，固定 `err == nil` 和 `errors.As(err, &biz)` 两个 `FailWithErr` 分支均生成非 skipped 响应断言，并且只引入 `net/http/httptest` 而不误引入未使用的 `net/http`。
16. [x] 复跑 QuickSmoke top20：20/20 `passed`，`zero_skip` 从 6 提升到 8，`skipped_total` 从 14 降到 12；剩余 skipped ready 集中在 `logx.Init` 和 MySQL repo 分支。
17. [x] 增强 Go static generator：支持 `logx.Init(config.Log)` 全局 logger 初始化分支，生成 `zerolog` global level/logger、`CallerMarshalFunc`、`config.Dev` 和工作目录恢复逻辑。
18. [x] 补充 `logx.Init` 的日志级别断言、`CallerMarshalFunc` 调用断言、`os.MkdirAll` 错误路径输入和 `config.Dev` writer 分支输入。
19. [x] 修复 seed 复用源码 import alias 的问题，确保 `zlog "github.com/rs/zerolog/log"` 这类别名在生成测试中保留。
20. [x] 补充 generator 回归测试，固定 `logx.Init` 四类分支均生成非 skipped 测试，并包含全局状态恢复和临时目录隔离。
21. [x] 复跑 QuickSmoke top20：20/20 `passed`，`zero_skip` 从 8 提升到 14，`skipped_total` 从 12 降到 6；剩余 skipped ready 全部集中在 MySQL repo/GORM 分支。
22. [x] 确认 QuickSmoke 当前 `go.mod` 没有 `sqlite` 或 `sqlmock` 依赖，直接生成带新第三方依赖的 GORM 测试不合适。
23. [x] 在 `validate_coverage_task` 后处理中新增 `manual_review_database` action，当 skipped TODO 命中 repo/GORM 数据库错误分支时，从普通 `ready` 队列中分离，并在 metadata 中写入 `database_dependent` 和 `database_reason`。
24. [x] 补充 handler 回归测试，固定 repo 数据库分支会输出 `passed/manual_review_database`。
25. [x] 复跑 QuickSmoke top20：20/20 `passed`，`zero_skip=14`，`skipped_total=6`，`action_counts={"ready":14,"manual_review_database":6}`，`skipped_ready` 已清空。

已完成补充：第三真实样本补到了 laoxia/lazy 都没有暴露的生成质量缺口：泛型 helper 指针值断言、泛型指针解引用、nil receiver 分支、JWT Parse 错误输入合成、Gin response helper 断言、全局 logger 初始化分支，以及 `/vN` 语义版本 import、源码 alias 复用和显式 import alias 问题。QuickSmoke top20 当前已经没有普通 `skipped_ready`，剩余数据库分支都被明确归类为需要测试数据库策略或依赖注入。下一步建议开始第四个真实样本，优先选择非 Go 或 Go 中已有数据库测试依赖的项目，验证 `manual_review_database` 不会掩盖可静态生成的普通分支。

## 第一百五十八阶段：第四个真实 JS/Vitest 项目样本验证

1. [x] 从本机 JS/TS 项目候选中筛选第四样本，确认 `/Users/binlee/code/open-source/mcp-hub` 使用 Vitest 和 `@vitest/coverage-v8`，但全量测试基线当前存在项目自身失败，因此先限定在已通过的 `tests/env-resolver.test.js` / `tests/config.test.js` 子集。
2. [x] 新增 `scripts/validate-js-coverage-top-tasks.sh` 和 opt-in 集成测试，支持 `vitest` / `jest` / `mocha`、baseline 测试子集参数、文件过滤、隔离 worktree 验证和 JSONL 输出。
3. [x] 首轮样本暴露 `run_tests` 对 Vitest 3 追加非法 `--verbose`，并且 parser 将命令级错误误判为 `pass`；已修复 Vitest 参数和命令级错误解析。
4. [x] 首轮样本暴露 macOS `/var` 与 `/private/var` 等价路径会导致隔离任务路径未重写；已在 JS 验证脚本中用真实路径重写修复。
5. [x] 首轮样本暴露真实 Vitest 项目可能配置 `include: ["tests/**/*.test.js"]`，源码旁生成的 `src/**/*.test.js` 不会被执行；已让 JS coverage task 在项目已有 `tests/` 且源码位于 `src/` 下时写入 `tests/` 镜像路径，并按测试文件位置生成相对 import。
6. [x] 复跑 `mcp-hub` `env-resolver.js` top5：测试文件路径和 import 已正确，`EnvResolver._executeCommand` 真实通过；剩余 4 个失败集中在 JS class method 错误/返回路径输入合成，典型问题是把未覆盖 `throw` 行误转成 `rejects.toThrow()`，但生成输入没有满足实际条件。
7. [x] 增强 JS class coverage task 输入合成：支持 `fallbackValue === undefined && this.strict`、`depth > this.maxPasses`、`placeholders.length === 0` 这类依赖实例字段或局部派生值的条件，生成 `new EnvResolver({ strict: true })`、`new EnvResolver({ maxPasses: 0 })`、`_resolveEnvObject({ MISSING: null }, {})`、`_resolveStringWithPlaceholders('${MISSING}', {}, 1)` 和 plain string 返回路径输入。
8. [x] 修复 class method coverage task 的断言选择：`return_path` 不再因为同一方法体存在其他 `throw` 分支而生成 `rejects.toThrow()`。
9. [x] 复跑 `mcp-hub` `env-resolver.js` top5：`5/5 passed`，`action_counts={"ready":5}`，`zero_skip=5`，`skipped_total=0`。
10. [x] 扩大到 `mcp-hub` `src/utils` top10：当前 `2/10 passed`，新失败集中在 `ConfigManager.#diffConfigs`、`DevWatcher.#handleFileChange` 这类 JS private method 外部不可调用，以及 `DevWatcher.stop` 的构造器配置缺失。
11. [x] 新增 `manual_review_private` action，把 JavaScript `#private` method 外部直接调用导致的语法失败从普通 `repair_generated_test` 中分离，metadata 写入 `private_method` 和 `private_reason`。
12. [x] 增强 JS class constructor 解析和实例化输入合成：保留 constructor 参数，并为 `serverName`、`devConfig`、`options` / `config` 这类常见参数生成最小可运行入参。
13. [x] 复跑 `mcp-hub` `src/utils` top10：脚本通过，`action_counts={"ready":4,"manual_review_private":6}`，`zero_skip=10`，`skipped_total=0`，已清空普通 `repair_generated_test`。
14. [x] 扩大到 `mcp-hub` `src/utils` top20：新增普通失败集中在 `wrapError` 的 `error` 参数输入，以及 `logger.js` 默认导出实例被误当成可命名导出的 `Logger` class。
15. [x] 增强 JS static generator：识别 `class Logger` + `const logger = new Logger(...)` + `export default logger` 这类默认导出实例，生成默认 import 并通过实例调用方法；同时为 `this.LOG_LEVELS[level] !== undefined` 和 `if (enable)` 分支合成可执行输入。
16. [x] 增强 JS 错误对象输入和返回类型推断：`error` / `err` 参数默认生成 `new Error('test error')`，`new MCPHubError(...)` 这类 return 表达式识别为 object，避免错误包装 helper 生成 `undefined` 输入或 boolean 断言。
17. [x] 复跑 `mcp-hub` `src/utils` top20：脚本通过，`action_counts={"ready":13,"manual_review_private":7}`，`zero_skip=20`，`skipped_total=0`，普通 `repair_generated_test` 已清空。
18. [x] 将 JavaScript `#private` coverage task 从非法直接调用改为 `it.skip` manual-review 草稿，并从同 class 方法体中提取公共入口候选，例如 `ConfigManager.loadConfig`、`DevWatcher.start`，写入 `metadata.public_entry_candidates`。
19. [x] 修复 Jest/Vitest parser 的 skipped 统计，支持 Vitest 3 的 `Tests  1 skipped (1)` 摘要和 `↓` 结果行。
20. [x] 复跑 `mcp-hub` `src/utils` top20：脚本通过，`status_counts={"passed":20}`，`action_counts={"ready":13,"manual_review_private":7}`，`zero_skip=13`，`skipped_total=7`，所有 private 任务都以稳定 manual-review skip 暴露公共入口候选。
21. [x] 扩大到 `mcp-hub` `src/utils` top30：新增普通失败集中在 ESM 内部未导出的 `StorageManager`，以及 `SSEManager.addConnection` 的 Express `req` / `res` mock 缺失。
22. [x] 增强 JS parser/generator：普通方法名 `get` / `set` 不再被关键字过滤；ESM 内部未导出 class 生成 `manual_review_internal` skipped 草稿；Express `req` / `res` 参数生成最小 mock，并为 `SSEManager.addConnection` 的 `shutdownTimer` 分支合成实例状态。
23. [x] 复跑 `mcp-hub` `src/utils` top30：脚本通过，`status_counts={"passed":30}`，`action_counts={"ready":21,"manual_review_private":7,"manual_review_internal":2}`，`zero_skip=21`，`skipped_total=9`，普通 `repair_generated_test` 已清空。
24. [x] 扩大到 `mcp-hub` `src/utils` top40：新增任务主要集中在 `SSEManager.broadcast`、`SSEManager.sendToClient`、`SSEManager.shutdown` 和 `SSEManager.setupAutoShutdown`，未暴露新的普通失败类型。
25. [x] 复跑 `mcp-hub` `src/utils` top40：脚本通过，`status_counts={"passed":40}`，`action_counts={"ready":31,"manual_review_private":7,"manual_review_internal":2}`，`zero_skip=31`，`skipped_total=9`，普通 `repair_generated_test` 继续保持清零。
26. [x] 完成 `ConfigManager.loadConfig` 公共入口自动生成试点：对 `ConfigManager.#diffConfigs` 任务生成临时 config 文件、旧配置状态和 `changes` 断言，覆盖 removed、field missing、deepEqual 和 modifiedFields 分支。
27. [x] 复跑 `mcp-hub` `src/utils` top40：脚本通过，`status_counts={"passed":40}`，`action_counts={"ready":36,"manual_review_private":2,"manual_review_internal":2}`，`zero_skip=36`，`skipped_total=4`；`ConfigManager.#diffConfigs` 5 个 private task 已转成真实 ready 测试，剩余 private task 仅为 `DevWatcher.#handleFileChange`。
28. [x] 完成 `DevWatcher.start` 公共入口自动生成试点：对 `DevWatcher.#handleFileChange` 任务生成 Vitest `chokidar` mock、fake timers、watcher `change` 事件、绝对路径/已有 debounce timer 场景和 `filesChanged` 事件断言。
29. [x] 复跑 `mcp-hub` `src/utils` top40：脚本通过，`status_counts={"passed":40}`，`action_counts={"ready":38,"manual_review_internal":2}`，`zero_skip=38`，`skipped_total=2`；JS private method 任务已全部从 `manual_review_private` 转成真实 ready 测试。
30. [x] 完成 `StorageManager` 内部 class 公共入口自动生成试点：对 `StorageManager.init/get` 任务生成 Vitest `fs/promises` mock、logger mock、动态 import 和默认导出 `MCPHubOAuthProvider` 断言，避免模块初始化早于 mock 注册。
31. [x] 复跑 `mcp-hub` `src/utils` top40：脚本通过，`status_counts={"passed":40}`，`action_counts={"ready":40}`，`zero_skip=40`，`skipped_total=0`；普通失败、private review 和 internal review 均已清零。
32. [x] 扩大到 `mcp-hub` `src/utils` top50：新增普通失败集中在 `WorkspaceCacheManager.updateWorkspaceState` 的 `cache[workspaceKey]` 分支，生成测试把 `port` 写成 `undefined`，未能预置 cache。
33. [x] 增强 JS class coverage task：支持 `WorkspaceCacheManager.updateWorkspaceState` 状态更新分支，生成 `port=3000`、预置 `"3000"` workspace cache、mock `_withLock/_readCache/_writeCache`，并断言写入的合并状态。
34. [x] 复跑 `mcp-hub` `src/utils` top50：脚本通过，`status_counts={"passed":50}`，`action_counts={"ready":50}`，`zero_skip=50`，`skipped_total=0`。

已完成补充：第四真实样本已经证明 JS/Vitest 链路不能只依赖 fixture，需要真实项目的 Vitest 版本、include 配置、测试目录约定、命令级错误解析、实例字段条件、class return-path 断言、constructor 入参、默认导出实例、错误对象输入、JS private method 访问性、ESM 内部 class 可见性、Express req/res mock、公共入口覆盖 private method、watcher/fake timer 事件驱动测试、模块初始化前 mock 注册、未导出内部 class 的默认导出公共入口覆盖，以及缓存状态更新方法的内部 I/O mock 一起验证。`env-resolver.js` top5 已从 `1/5 passed` 提升到 `5/5 passed`，`src/utils` top50 已清空普通修复失败；`ConfigManager.#diffConfigs` 已通过 `loadConfig` 公共入口从 private review 转成真实 ready，`DevWatcher.#handleFileChange` 已通过 `start` 公共入口从 private review 转成真实 ready，`StorageManager.init/get` 已通过动态 import 和 `MCPHubOAuthProvider` 默认导出从 internal review 转成真实 ready，`WorkspaceCacheManager.updateWorkspaceState` 已通过预置 cache 和内部 I/O mock 转成真实 ready。当前 `src/utils` top50 已达到 `ready=50`、`skipped_total=0`。下一步建议切换到第五个真实 JS/TS 项目样本，验证这些针对 `mcp-hub` 的专用规则没有掩盖其他项目的新失败类型。

## 第一百五十九阶段：第五个真实 JS/Jest 项目样本验证

1. [x] 从本机 JS/TS 候选中选择 `/Users/binlee/code/open-source/ip2region/binding/javascript` 作为第五样本；该项目是 ESM + Jest 子包，`npm test -- --runInBand` 通过，`npm test -- --coverage --runInBand` 可生成覆盖率。
2. [x] 首轮验证暴露 JS 子包隔离复制问题：`ip2region` 测试通过 `../../../data` 读取 monorepo 父级 xdb 数据，原验证脚本只复制 package 目录，导致 baseline 在临时 worktree 中缺少 `data/`。
3. [x] 为 JS 验证脚本新增 `TESTLOOP_VALIDATE_JS_EXTRA_SYMLINKS`，支持把外部资源以 symlink 形式挂载到 baseline/task worktree，例如 `../../data:../../data`，避免复制大体积测试数据。
4. [x] 复跑 `ip2region` top20 后暴露 9 个普通生成失败：`versionFromHeader` 对象参数被生成为 `undefined`，以及未导出的 `_parse_ipv4_addr/_parse_ipv6_addr` 被错误生成命名 import。
5. [x] 增强 JS function coverage task：支持 `versionFromHeader` 的 `h.version == XdbStructure20`、`h.version != XdbStructure30`、`ipVer == XdbIPv4Id`、`ipVer == XdbIPv6Id` 分支对象输入和返回断言。
6. [x] 增强 JS function coverage task：对未导出的 `_parse_ipv4_addr/_parse_ipv6_addr` 不再直接 import，而是通过公开 `parseIP()` 入口覆盖 IPv4 段数错误、IPv4 越界、IPv6 段数错误、IPv6 多重双冒号和 IPv6 越界错误分支。
7. [x] 复跑 `ip2region` top20：脚本通过，`status_counts={"passed":20}`，`action_counts={"ready":20}`，`zero_skip=20`，`skipped_total=0`。
8. [x] 扩大到 `ip2region` top30：新增失败集中在带状态 class 的实例构造，`Version.ipCompare` 缺 compare callback，`Searcher.search/read/toString` 因通用构造参数误走文件打开分支。
9. [x] 增强 JS class coverage task：为 `Version.ipCompare` 注入最小 compare callback；为 `Searcher.search` 生成合法 version + in-memory buffer，覆盖 zero pointer 和 empty match 返回路径；为 `Searcher.read` 用 `Object.create(Searcher.prototype)` 和临时替换 `fs.readSync` 覆盖短读异常；为 `Searcher.toString` 使用 cBuffer 构造路径避免真实文件依赖。
10. [x] 复跑 `ip2region` top30：脚本通过，`status_counts={"passed":30}`，`action_counts={"ready":30}`，`zero_skip=30`，`skipped_total=0`。

已完成补充：第五真实样本证明 JS/Jest 链路还需要覆盖 ESM Jest 的 `NODE_OPTIONS`、monorepo 子包外部测试资源、对象参数分支输入合成、未导出顶层 helper 的公共入口覆盖，以及带状态 class 的最小合法实例构造。`ip2region` top30 已达到 `ready=30`、`skipped_total=0`，并额外验证了 Buffer、二进制 searcher、文件 I/O 短读异常和 ESM Jest 无全局 `jest` 的场景。下一步建议切换到 TypeScript/Jest 项目样本，重点验证 TS 编译配置、类型擦除、路径别名、Jest transformer 和真实 TS test include 规则。

## 第一百六十阶段：第六个真实 TypeScript/Jest 项目样本验证

1. [x] 从本机候选中选择 `/Users/binlee/code/open-source/codex/sdk/typescript` 作为第六样本；该项目是 TypeScript ESM SDK，使用 Jest + `ts-jest`，并通过 `testMatch: ["**/tests/**/*.test.ts"]` 约束测试文件。
2. [x] 完成样本依赖准备：在 codex monorepo 根执行 `pnpm install`，样本依赖由 workspace 根 `node_modules` 提供；隔离验证使用 `TESTLOOP_VALIDATE_JS_EXTRA_SYMLINKS='../../node_modules:../node_modules'` 挂载 workspace 根依赖，避免 `npx` 临时下载错误版本 Jest。
3. [x] 确认可控基线：全量 SDK Jest 当前依赖缺失的 Rust `codex-rs/target/debug/codex` 二进制，不适合作为 baseline；`tests/exec.test.ts` 子集独立通过，并可生成 `src/exec.ts` 覆盖率。
4. [x] 首轮 top5 暴露 TypeScript/Jest 生成质量问题：`CodexExec.run` 被传入 `[]` 导致 TS 编译失败，未导出的 `flattenConfigOverrides` 被错误生成命名 import。
5. [x] 增强 JS/TS coverage task：对 `CodexExec.run` 生成合法 `{ input: 'hi' }` 参数，并用 ESM Jest `child_process.spawn` mock 触发 `spawnError` 分支；对 `flattenConfigOverrides` 改走 `CodexExec.run` 公共入口，覆盖顶层非法值、普通 primitive、顶层空对象、嵌套空对象和递归 flatten 分支。
6. [x] 复跑 Codex SDK TS/Jest top5：脚本通过，`status_counts={"passed":5}`，`action_counts={"ready":5}`，`zero_skip=5`，`skipped_total=0`。
7. [x] 扩大到 Codex SDK TS/Jest top10：新增失败集中在未导出的 `toTomlValue`，仍是内部 helper 直接 import 问题。
8. [x] 增强 JS/TS coverage task：对 `toTomlValue` 改走 `CodexExec.run` 公共入口，使用 `{ retries: 3 }` 覆盖 number TOML 序列化分支，使用 `{ retries: Infinity }` 覆盖 finite number 校验错误分支。
9. [x] 复跑 Codex SDK TS/Jest top10：脚本通过，`status_counts={"passed":10}`，`action_counts={"ready":10}`，`zero_skip=10`，`skipped_total=0`。
10. [x] 扩大到 Codex SDK TS/Jest top20：新增失败集中在未导出的 `findCodexPath`，通用生成器仍错误生成命名 import。
11. [x] 增强 JS/TS coverage task：对 `findCodexPath` 的 unsupported platform/arch 分支，通过临时覆盖 `process.platform/process.arch` 并调用 `new CodexExec(null)` 公共入口生成真实断言；对内部 platform package map 和 optional native package 布局分支生成 `manual_review_internal` 草稿。
12. [x] 复跑 Codex SDK TS/Jest top20：脚本通过，`status_counts={"passed":20}`，`action_counts={"ready":18,"manual_review_internal":2}`，`zero_skip=18`，`skipped_total=2`。
13. [x] 扩大到 Codex SDK TS/Jest top30：新增普通失败集中在 `resolveNativePackage` 的 null 返回路径使用 `undefined` 参数导致 TS 编译失败，以及未导出的 `serializeConfigOverrides` 直接 import。
14. [x] 增强 JS/TS coverage task：为 `resolveNativePackage` 生成类型合法的缺失 vendor root / target triple / binary name 字符串入参并断言 `null`；把 `serializeConfigOverrides` 纳入 `CodexExec.run` 公共入口覆盖规则。
15. [x] 复跑 Codex SDK TS/Jest top30：脚本通过，`status_counts={"passed":30}`，`action_counts={"ready":28,"manual_review_internal":2}`，`zero_skip=28`，`skipped_total=2`。
16. [x] 扩大到 Codex SDK TS/Jest top40：新增普通失败集中在未导出的 `formatTomlKey` / `isPlainObject`，通用生成器仍错误生成命名 import。
17. [x] 增强 JS/TS coverage task：把 `formatTomlKey` / `isPlainObject` 纳入 `CodexExec.run` 公共入口覆盖规则；`formatTomlKey` 使用数组对象配置值触发 `toTomlValue` object 分支里的 quoted TOML key formatter，`isPlainObject` 使用数组配置值覆盖非 plain object 判定。
18. [x] 复跑 Codex SDK TS/Jest top40：脚本通过，`status_counts={"passed":40}`，`action_counts={"ready":37,"manual_review_internal":3}`，`zero_skip=37`，`skipped_total=3`。
19. [x] 扩大到 Codex SDK TS/Jest top50：新增普通失败集中在未导出的 `isDirectory`，通用生成器仍错误生成命名 import；同时暴露更多 `findCodexPath` 内部 package resolution 分支应继续归为 `manual_review_internal`。
20. [x] 增强 JS/TS coverage task：对 `isDirectory` 这类内部文件系统 helper 生成 `manual_review_internal` 草稿，并标注 `findCodexPath` / `resolveNativePackage` 公共入口候选，避免伪造不可访问的外部调用。
21. [x] 复跑 Codex SDK TS/Jest top50：脚本通过，`status_counts={"passed":50}`，`action_counts={"ready":42,"manual_review_internal":8}`，`zero_skip=42`，`skipped_total=8`。
22. [x] 扩大到 Codex SDK TS/Jest top60：脚本直接通过，`status_counts={"passed":60}`，`action_counts={"ready":52,"manual_review_internal":8}`，`zero_skip=52`，`skipped_total=8`；新增任务集中在 `CodexExec.run` 参数分支和 stdin/stdout 错误路径。
23. [x] 补强 JS/TS coverage task 质量：将 `CodexExec.run` 的 `--model`、`--sandbox`、`--cd`、`--add-dir`、`--output-schema`、network/web search/approval config、PATH prepend、`CODEX_API_KEY`、缺失 stdin/stdout 等分支改为分支专属 args 和 `commandArgs` / `spawnOptions.env` / `child.killed` 断言，避免继续用泛化 spawn error 测试覆盖所有行。
24. [x] 复跑 Codex SDK TS/Jest top60：脚本通过，`status_counts={"passed":60}`，`action_counts={"ready":52,"manual_review_internal":8}`，并抽查确认 top60 新增测试已生成具体参数和错误路径断言。
25. [x] 扩大到 Codex SDK TS/Jest top70：脚本通过，`status_counts={"passed":70}`，`action_counts={"ready":62,"manual_review_internal":8}`，`zero_skip=62`，`skipped_total=8`；新增任务集中在 `CodexExec.run` stdout yield 和配置序列化细分行。
26. [x] 补强 JS/TS coverage task 质量：将 `CodexExec.run` 的 stdout yield 分支从泛化 spawn error 模板改为 `PassThrough` 写入、`for await` 收集输出并断言 yielded line；首次使用 `Array.fromAsync` 在真实 Jest/Node 环境失败后，改为更兼容的显式循环。
27. [x] 复跑 Codex SDK TS/Jest top70：脚本通过，`status_counts={"passed":70}`，`action_counts={"ready":62,"manual_review_internal":8}`，并抽查确认 `jest-61` 生成测试已断言 `output=['ready']`。
28. [x] 扩大到 Codex SDK TS/Jest top80：脚本通过，`status_counts={"passed":80}`，`action_counts={"ready":67,"manual_review_internal":13}`，`zero_skip=67`，`skipped_total=13`；新增任务集中在 `toTomlValue` 的 object / undefined skip / unsupported value 细分行、`CodexExec.run` config override loop，以及更多 `findCodexPath` 内部平台映射分支。
29. [x] 补强 JS/TS coverage task 质量：将 `toTomlValue` 的数组、boolean、inline object、对象中 `undefined` child skip、null、unsupported function value 分支改为专属 config override 输入和断言；将 `CodexExec.run` 90-92 行改为断言 config override loop 产生 `--config model="gpt-5"`，避免继续用 spawn error 模板。
30. [x] 复跑 Codex SDK TS/Jest top80：脚本通过，`status_counts={"passed":80}`，`action_counts={"ready":67,"manual_review_internal":13}`，并抽查确认 `jest-71` 到 `jest-76` 已生成具体 TOML / config loop 断言。
31. [x] 扩大到 Codex SDK TS/Jest top90：脚本初跑通过，`status_counts={"passed":90}`，但新增 `findCodexPath` 平台映射分支全部被归为 `manual_review_internal`，`action_counts={"ready":67,"manual_review_internal":23}`。
32. [x] 补强 JS/TS coverage task 质量：将 `findCodexPath` 的 linux/darwin/win32 x64/arm64/default arch 平台映射分支改为通过 `CodexExec(null)` 公共入口覆盖；supported target triple 分支断言 “Unable to locate Codex CLI binaries”，unsupported arch 分支断言 “Unsupported platform”，真正依赖 package map / native package 文件布局的分支继续手审。
33. [x] 复跑 Codex SDK TS/Jest top90：脚本通过，`status_counts={"passed":90}`，`action_counts={"ready":81,"manual_review_internal":9}`，`zero_skip=81`，`skipped_total=9`；剩余手审集中在 `findCodexPath` package map、native package lookup、`isDirectory` 内部 FS helper 和 native package return。
34. [x] 尝试扩大到 Codex SDK TS/Jest top100：当前 `src/exec.ts` 过滤后仅 96 个 coverage tasks，脚本正确拒绝 `limit=100`，因此改跑当前文件任务上限 top96。
35. [x] top96 初跑暴露两个文件级任务失败：`jest-95 exec.ts 78-80` 和 `jest-96 exec.ts entire file` 回退成全量导入，错误引用未导出的 `serializeConfigOverrides` / `findCodexPath` / `isFile` 等内部 helper。
36. [x] 补强 JS/TS coverage task：对文件级 target 或 `line_range=entire file` 生成 `manual_review_internal` 草稿，提示拆分为 exported class method、exported function 或 explicit public-entry task，避免伪造全文件测试。
37. [x] 复跑 Codex SDK TS/Jest top96：脚本通过，`status_counts={"passed":96}`，`action_counts={"ready":81,"manual_review_internal":15}`，`zero_skip=81`，`skipped_total=15`；当前 `src/exec.ts` 样本任务已跑到上限。
38. [x] 切换到 Codex SDK TS/Jest 的第二个源码文件 `src/thread.ts`：基线测试子集仍依赖外部 Rust `codex-rs/target/debug/codex`，本机安装版 `codex` 又与当前 SDK 的 trusted-directory 行为不完全匹配，因此继续采用覆盖率 JSON 可产出的隔离验证结果作为任务来源。
39. [x] `thread.ts` top18 初跑暴露 TS class 生成缺陷：TypeScript `private` 方法被当公开方法直接调用，constructor 的 `CodexExec` / `Input` / `TurnOptions` 入参被生成成 `undefined` 或错误类型，getter `id` 被当普通方法调用。
40. [x] 增强 JS/TS parser/generator：解析 TypeScript 参数类型、`private/protected` 方法和 `get` accessor；对 TS private method 生成 `manual_review_private` 草稿并给出公共入口候选；对 `CodexExec` constructor 参数生成最小 async generator mock；对 `Input` / `TurnOptions` / nullable id 生成类型合法输入；getter 生成属性访问。
41. [x] 增强 `Thread.run` error-path 任务：当覆盖目标需要错误路径时，`CodexExec` mock 会产出 `turn.failed` 事件，避免用正常 `turn.completed` mock 导致 `rejects.toThrow()` 断言失败。
42. [x] 复跑 Codex SDK TS/Jest `src/thread.ts` top18：脚本通过，`status_counts={"passed":18}`，`action_counts={"ready":18}`，`zero_skip=13`，`skipped_total=5`；5 个 skipped 均为 `runStreamedInternal` private 任务的明确 manual-review 草稿，普通 `repair_generated_test` 已清零。
43. [x] 切换到 `src/outputSchemaFile.ts`：top3 初跑全部失败，生成器把 `schema` 生成为 `undefined`，但该值对应正常返回路径，无法覆盖 plain object 校验和 write failure cleanup。
44. [x] 增强 JS/TS function coverage task：对 `createOutputSchemaFile` 的非法 schema 分支生成 `null` 输入并断言错误消息；对 `writeFile` 失败分支通过动态 import `node:fs` 和 `@jest/globals`，spy `fs.writeFile` reject、spy `fs.rm` cleanup 并断言二者调用。
45. [x] 复跑 Codex SDK TS/Jest `src/outputSchemaFile.ts` top3：脚本通过，`status_counts={"passed":3}`，`action_counts={"ready":3}`，`zero_skip=3`，`skipped_total=0`。
46. [x] 切换到 `src/codex.ts`：top1 初跑暴露 `resumeThread(id: string)` 被生成 `1` 的类型默认值问题，以及 `new Codex({})` 在构造阶段触发 native CLI package lookup。
47. [x] 增强 JS/TS class coverage task：明确 `string` 类型优先于 `id` 数字命名启发，`number` 类型仍保留 `b/y -> 2`；`Codex` 包装类生成 `new Codex({ codexPathOverride: 'codex' })`，避免纯 wrapper 方法测试被 native package 查找阻断。
48. [x] 复跑 Codex SDK TS/Jest `src/codex.ts` top1：脚本通过，`status_counts={"passed":1}`，`action_counts={"ready":1}`，`zero_skip=1`，`skipped_total=0`。
49. [x] 切换到 `src/events.ts`：初跑按文件过滤后 `coverage tasks after filter = 0`，原因是 Istanbul/Jest coverage 不会把 TypeScript 纯 type/union 文件写入 `coverage-final.json`，当前验证脚本无法区分“无任务”与“工具失败”。
50. [x] 增强 JS/TS file-level coverage task：当目标是 TypeScript 纯类型文件且源码只有 `type/interface/enum` 等声明、没有运行时函数或 class 时，生成 `manual_review_no_runtime` skipped 草稿，提示通过消费方运行时测试或类型检查验证；generation context 会保留 `types` 列表，避免把 type-only 文件当空文件。
51. [x] 增强 JS 真实项目验证脚本：当 `TESTLOOP_VALIDATE_JS_FILE_FILTER` 命中源码但 coverage report 没有任务时，合成 no-runtime 文件级任务；若项目存在 `tests/` 目录，合成测试优先写入 `tests/<name>.test.ts`，适配 Codex SDK 的 `testMatch: ["**/tests/**/*.test.ts"]`。
52. [x] 复跑 Codex SDK TS/Jest `src/events.ts` top1：脚本通过，`status_counts={"passed":1}`，`action_counts={"manual_review_no_runtime":1}`，`zero_skip=0`，`skipped_total=1`，生成 context 保留 `ThreadStartedEvent`、`ThreadEvent` 等类型声明。
53. [x] 复跑 Codex SDK TS/Jest `src/items.ts` top1：脚本通过，`status_counts={"passed":1}`，`action_counts={"manual_review_no_runtime":1}`，`zero_skip=0`，`skipped_total=1`，确认纯 type/union 文件策略不是 `events.ts` 单点特化。
54. [x] 复跑 Codex SDK TS/Jest `src/codexOptions.ts`、`src/threadOptions.ts`、`src/turnOptions.ts` top1：三者均通过，`action_counts={"manual_review_no_runtime":1}`，`skipped_total=1`，确认 options 类型文件同样稳定归类为 no-runtime。
55. [x] 切换到 `src/index.ts`：初跑按文件过滤后 0 个任务，原因是该文件是 TypeScript barrel re-export，没有本地函数/class；增强 no-runtime 判定后，`src/index.ts` top1 通过，`action_counts={"manual_review_no_runtime":1}`，`skipped_total=1`。
56. [x] 复查 `src/exec.ts` 当前任务上限：由于 no-runtime 和排序变化，当前过滤后为 top86；初跑暴露 11 个普通失败，集中在公开 `prependPathDirs` 参数类型错误，以及未导出的 `pathEnvKey` / `existingDirs` / `isFile` 被直接 import。
57. [x] 增强 JS/TS coverage task：`pathEnvKey` 通过公开 `prependPathDirs` 覆盖；`prependPathDirs` 生成合法 `Record<string,string>` env、`string[]` pathDirs 和 `NodeJS.Platform`，断言 Windows PATH key 去重和非 Windows `PATH` 保留；`existingDirs` / `isFile` 通过公开 `resolveNativePackage` 覆盖，构造临时 vendor/package/bin/codex/codex-path 目录并断言 `executablePath` 与 `pathDirs`。
58. [x] 复跑 Codex SDK TS/Jest `src/exec.ts` top86：脚本通过，`status_counts={"passed":86}`，`action_counts={"ready":71,"manual_review_internal":15}`，`zero_skip=71`，`skipped_total=15`，普通 `repair_generated_test` 已清零。
59. [x] 复查 `src/thread.ts` top18：虽然脚本通过，但 5 个 `Thread.runStreamedInternal` private task 仍是 skipped ready，覆盖 `104-106`、`99-103`、`100`、`105`、`107` 等行，质量上仍是低价值手审草稿。
60. [x] 增强 JS/TS coverage task：对 `Thread.runStreamedInternal` 改用公开 `Thread.runStreamed()` 覆盖，mock `CodexExec.run` async generator；`thread.started` 分支断言 yielded event 和 `instance.id`，JSON parse error 分支断言 `events.next()` reject。
61. [x] 复跑 Codex SDK TS/Jest `src/thread.ts` top18：脚本通过，`status_counts={"passed":18}`，`action_counts={"ready":18}`，`zero_skip=18`，`skipped_total=0`；5 个 private skipped task 全部转为真实 ready。
62. [x] 尝试按 `tests/run.test.ts` / `tests/runStreamed.test.ts` 跨文件扩大到 Codex SDK TS/Jest top120：初跑 `status_counts={"failed":17,"passed":103}`，失败全部集中在测试辅助文件 `tests/responsesProxy.ts`，包括 `startResponsesTestProxy({})` 类型非法、`responseFailed` 被误判为 throw、未导出的 `formatSseEvent` 被非法 named import。
63. [x] 增强 JS/TS coverage task：对 ESM 未导出顶层函数增加 `manual_review_internal` 兜底，CommonJS 仍保留 `require()` 生成；`formatSseEvent` 改路由到公开 `startResponsesTestProxy()`；`startResponsesTestProxy` 生成真实 HTTP POST、SSE 文本、recorded request、404 和 generator exhausted 500 断言；`responseFailed` 断言返回的 error event 对象。
64. [x] 对 `server.address()` 异常和 `server.close(err)` 这类无法稳定从生成测试触发的 Node 内部分支，生成 `manual_review_internal` 草稿，避免为了通过验证而生成不覆盖目标的伪测试。
65. [x] 复跑 Codex SDK TS/Jest `tests/run.test.ts` / `tests/runStreamed.test.ts` 跨文件 top120：脚本通过，`status_counts={"passed":120}`，`action_counts={"ready":108,"manual_review_internal":12}`，`zero_skip=108`，`skipped_total=12`；17 个 `repair_generated_test` 全部清零，剩余手审为 9 个 `findCodexPath/isDirectory` 内部 native package/FS 分支和 3 个 `responsesProxy` Node server 内部分支。
66. [x] 切换到 Codex SDK TS/Jest `tests/abort.test.ts` top80：初跑 `status_counts={"failed":5,"passed":75}`，5 个普通失败集中在未导出的 `normalizeInput`、`isJsonObject`、`hasExplicitProviderConfig`，虽然生成了 manual-review skip 内容，但 ESM named import 仍非法导致测试文件编译失败。
67. [x] 增强 JS/TS coverage task：ESM import 生成会跳过未导出的顶层函数；`normalizeInput` 改通过 `Thread.runStreamed()` 公开入口覆盖 structured text + local image，并断言传给 `CodexExec.run` 的 `input` / `images`；`isJsonObject` 改通过 `createOutputSchemaFile({ type: 'object' })` 覆盖；`hasExplicitProviderConfig` 改通过公开 `createTestClient()` 覆盖。
68. [x] 复跑 Codex SDK TS/Jest `tests/abort.test.ts` top80：脚本通过，`status_counts={"passed":80}`，`action_counts={"ready":70,"manual_review_internal":10}`，`zero_skip=70`，`skipped_total=10`；5 个 `repair_generated_test` 全部清零，剩余手审仍是 native package/FS 和 Node server 内部分支。
69. [x] 单独复跑 `tests/runStreamed.test.ts` top100：脚本通过，`status_counts={"passed":100}`，`action_counts={"ready":90,"manual_review_internal":10}`，`zero_skip=90`，`skipped_total=10`；resume streaming 和 output schema streaming 当前没有新增普通失败。
70. [x] 复跑 `tests/run.test.ts` / `tests/runStreamed.test.ts` 组合样本当前上限 top128：初跑已通过但新增 `getCurrentEnv` 仍为 `manual_review_internal`；增强后 `getCurrentEnv` 通过公开 `createTestClient()` 覆盖，断言普通 env 继承、`CODEX_INTERNAL_ORIGINATOR_OVERRIDE` 被过滤，并恢复 `process.env`。
71. [x] 复跑 Codex SDK TS/Jest `tests/run.test.ts` / `tests/runStreamed.test.ts` top128：脚本通过，`status_counts={"passed":128}`，`action_counts={"ready":109,"manual_review_internal":19}`，`zero_skip=109`，`skipped_total=19`；`getCurrentEnv` 从手审转为真实 ready，剩余手审集中在 native package/FS、Node server 内部分支和整文件泛化任务。
72. [x] 切换到 Codex SDK TS/Jest `tests/exec.test.ts` 当前上限 top96：脚本通过，`status_counts={"passed":96}`，`action_counts={"ready":81,"manual_review_internal":15}`，`zero_skip=81`，`skipped_total=15`；结果与 `src/exec.ts` 文件级验证一致，剩余 15 个手审全部为 native package/FS、平台包解析或整文件泛化任务，没有新增普通失败。
73. [x] 合并 Codex SDK TS/Jest `tests/abort.test.ts`、`tests/run.test.ts`、`tests/runStreamed.test.ts`、`tests/exec.test.ts` 组合样本当前上限 top101：脚本通过，`status_counts={"passed":101}`，`action_counts={"ready":83,"manual_review_internal":18}`，`zero_skip=83`，`skipped_total=18`；组合排序没有引入新的普通失败，剩余手审为 15 个 native package/FS/平台包解析或整文件泛化任务，以及 3 个 `responsesProxy.ts` Node server 内部分支。

已完成补充：第六真实样本证明 TypeScript/Jest 链路还需要覆盖 workspace pnpm 依赖布局、`ts-jest` diagnostics、ESM 动态导入、Jest mock hoisting、未导出顶层 helper 的公共入口覆盖、生成测试中的复杂 mock 类型约束、Node `process.platform/process.arch` 运行时全局状态分支、TS 严格模式下的函数入参类型合法性、TypeScript `private/protected` 访问性、getter 属性访问、constructor 参数类型驱动的 mock、间接触发内部 helper 时必须选择能真正走到目标分支的配置形状、无法从测试模块稳定触达的内部文件系统 helper 必须明确标记为手审，纯 type/union/options/barrel 文件不会进入 Istanbul/Jest runtime coverage、应生成 `manual_review_no_runtime` 而不是伪测试，以及“能跑通”不等于“有价值覆盖”，需要为参数分支生成可观察的命令行/env/错误路径断言。当前 `src/exec.ts` 按最新任务排序已完成 top86，达到 `passed=86`，其中 71 个为真实 ready 测试，15 个为明确的 `manual_review_internal`；`tests/exec.test.ts` 已完成 top96，达到 `passed=96`，其中 81 个为真实 ready 测试，15 个为明确的 `manual_review_internal`；`src/thread.ts` 已完成 top18，达到 `passed=18`，18 个均为真实 ready，`skipped_total=0`；`src/outputSchemaFile.ts` top3 和 `src/codex.ts` top1 已清零普通失败；`src/events.ts`、`src/items.ts`、`src/codexOptions.ts`、`src/threadOptions.ts`、`src/turnOptions.ts`、`src/index.ts` 已用 no-runtime 文件级任务确认 type-only/barrel 策略；按 `tests/run.test.ts` / `tests/runStreamed.test.ts` 跨文件 top128 已达到 `passed=128`，普通 `repair_generated_test` 清零；`tests/abort.test.ts` top80 已达到 `passed=80`，普通 `repair_generated_test` 清零；`abort + run/runStreamed + exec` 组合 top101 已达到 `passed=101`，普通 `repair_generated_test` 继续清零。下一步建议切到第七个真实 TypeScript/Jest 或 TypeScript/Vitest 样本，优先选择 Codex SDK 之外的项目，验证当前规则没有被 Codex SDK 的 API 形状过拟合。

## 第一百六十一阶段：第七个真实 TypeScript/Mocha 项目样本验证

1. [x] 从本机候选中选择 `/Users/binlee/code/free-works/haoying/rocketmq-clients/nodejs` 作为第七样本；该项目是 TypeScript + egg-bin/Mocha，源码包含 gRPC、protobuf、producer/consumer、message、retry、route 等真实业务模块。
2. [x] 采用临时 monorepo 副本 `/tmp/testloop-rocketmq-clients-sample` 验证，避免污染原项目；原项目无 lockfile 且已有 unrelated Python 改动，本轮只在 `/tmp` 安装依赖和生成 proto。
3. [x] 确认可控基线：全量 `npm test` 依赖本地 RocketMQ `127.0.0.1:8081`，不可作为稳定 baseline；`test/index.test.ts`、`test/message/MessageId.test.ts`、`test/util/index.test.ts` 子集通过，并且 `npx egg-bin cov --timeout 60000 ...` 能生成 `coverage/coverage-final.json`。
4. [x] 增强 JS 真实项目验证脚本：新增 `TESTLOOP_VALIDATE_JS_COVERAGE_COMMAND` baseline coverage 命令模板，支持 `npx egg-bin cov --timeout 60000 {args}` 这类项目自定义 coverage runner。
5. [x] 增强 `run_tests` JS runner：新增 `TESTLOOP_JS_TEST_COMMAND` 命令模板，支持 `npx egg-bin test --timeout 60000 {path}` 这类必须经项目脚本注册 TS runtime 的 Mocha 项目；否则直接 `npx mocha test/index.test.ts` 会触发 Node/ESM/TS 加载错误。
6. [x] 为自定义 runner 补充回归测试：覆盖 `run_tests` 从 package root 执行自定义命令并传入相对路径，以及 JS coverage baseline 命令模板展开 `{args}`。
7. [x] 复跑 RocketMQ Node.js TS/Mocha top10：脚本通过，`status_counts={"passed":10}`，`action_counts={"ready":10}`，`zero_skip=10`，`skipped_total=0`。
8. [x] 扩大到 RocketMQ Node.js TS/Mocha top60：脚本通过，`status_counts={"passed":60}`，`action_counts={"ready":38,"manual_review_private":22}`，`zero_skip=60`，`skipped_total=0`；手审集中在 `RpcClientManager`、`TelemetrySession`、`SimpleConsumer`、`Producer` 的 `#private` 方法。
9. [x] 扩大到 RocketMQ Node.js TS/Mocha top100 时发现先前通过结果是误判：Mocha/TS 编译失败输出为 `Exception during run`，解析器没有识别具体失败，`run_tests` 又忽略了 runner 的非零退出码，导致“0 failed”被错误归为 pass。
10. [x] 修复 `run_tests` 兜底语义：当测试命令非零退出且解析器没有提取失败时，合成 `test runner` 失败，避免将编译错误、运行器崩溃或加载失败误判为通过。
11. [x] 增强 TypeScript/Mocha 生成质量：识别 TS `static` class method，静态方法直接用 `ClassName.method()`；Mocha 项目缺少 `chai` 时改用 `node:assert`；async void 生成 `assert.doesNotReject` / Chai caught-error 断言；`manual_review_private/internal` 的 skipped body 主动引用导入符号，避免 `noUnusedLocals` 把手审草稿误报为生成失败。
12. [x] 增强 RocketMQ 纯逻辑覆盖：`StatusChecker.check` 根据覆盖行生成具体 `Code.*` 或 default numeric status，并区分 `assert.throws` / `assert.doesNotThrow`；`Buffer` / `Uint8Array` 参数生成 `Buffer.from('test')` / `new Uint8Array([1, 2, 3])`，避免 checksum 类纯函数生成 `undefined`。
13. [x] 复跑 RocketMQ `src/exception/StatusChecker.ts` top32：脚本通过，`status_counts={"passed":32}`，`action_counts={"ready":31,"manual_review_internal":1}`，`zero_skip=31`，`skipped_total=1`。
14. [x] 复跑 RocketMQ `src/util/index.ts` top20：脚本通过，`status_counts={"passed":20}`，`action_counts={"ready":20}`，`zero_skip=20`，`skipped_total=0`，确认 Buffer/Uint8Array 入参问题已清零。
15. [x] 复跑 RocketMQ Node.js TS/Mocha top100：当前真实结果为 `status_counts={"failed":33,"passed":67}`，`action_counts={"ready":34,"manual_review_private":33,"repair_generated_test":33}`，`zero_skip=67`，`skipped_total=33`；33 个剩余普通失败集中在复杂业务类型构造：`Endpoints` 14 个、`TopicRouteData` 7 个、`MessageOptions` 4 个、`ProducerOptions` 6 个，已不再包含 runner 误判、static 调用、二进制参数或 skipped manual-review 的 TS 未使用导入问题。

已完成补充：第七真实样本证明 JS/TS 闭环不能假设 Mocha 项目都能用 `npx mocha` 直接运行。真实 TypeScript/Mocha 项目经常通过 `egg-bin`、`ts-node` 或项目脚本注册运行时，coverage runner 也可能是 c8/egg-bin 包装命令。当前验证脚本和 `run_tests` 已支持命令模板，使 Agent 能在不污染原项目的前提下复用项目自己的测试入口。本阶段同时暴露了一个更重要的质量门槛：不能只看解析器的 failed 数，必须尊重 runner 的非零退出，否则 TypeScript 编译失败会被误报为通过。当前 RocketMQ 文件级纯逻辑样本已经稳定通过，top100 剩余问题收敛到复杂业务对象最小构造和跨模块类型解析，下一阶段应优先做 TypeScript constructor/type import-aware mock 生成，而不是为 RocketMQ 硬编码专用对象。

## 第一百六十二阶段：TypeScript 跨模块类型感知 mock 生成

1. [x] 从 RocketMQ 剩余失败中优先选择 `Endpoints`、`TopicRouteData`、`MessageOptions`、`ProducerOptions` 四类入参，按“读取同项目类型定义 -> 生成最小合法构造 -> 真实 Mocha 编译执行”闭环推进。
2. [x] 实现第一层通用能力：解析 TypeScript named import、barrel `export *` / named re-export、`.d.ts` 生成文件和外部包 named import；当参数类型是同项目导出的 class 时，生成 `new Type(...)` 并补充测试文件 import；当参数类型是 interface/type alias 时，合并到已有 TS object mock。
3. [x] 支持 `interface extends` 转成 intersection mock，支持 `Map<K,V>` 按 value type 生成 mock，支持外部 PascalCase 类型如 `Metadata` 生成 constructor mock；额外 value import 只对真实调用路径生效，避免污染 `manual_review_private/internal` 和 `StatusChecker` 特化测试。
4. [x] 复跑 RocketMQ `src/client/RpcClient.ts` top12：脚本通过，`status_counts={"passed":12}`，`action_counts={"ready":12}`；覆盖了 `Endpoints` constructor、protobuf `.d.ts` request class、`Metadata` 外部 constructor、gRPC callback `if (e)` error branch reject 断言。
5. [x] 复跑 RocketMQ `src/exception/StatusChecker.ts` top32：脚本仍通过，`status_counts={"passed":32}`，确认跨模块 value import 收窄后没有重新引入 TS `noUnusedLocals` 回归。
6. [x] 复跑 RocketMQ Node.js TS/Mocha top100：当前真实结果提升为 `status_counts={"failed":21,"passed":79}`，`action_counts={"ready":46,"manual_review_private":33,"repair_generated_test":21}`；相比上一阶段 `passed=67`，新增 12 个真实 ready，主要清掉 `RpcClient.*` 的 `Endpoints` / protobuf request / `Metadata` 类型构造和 error callback 断言问题。
7. [x] 修复 imported type alias 和嵌套 interface：额外 import 会保留 `Settings as SettingsPB` 这类 local alias；解析 imported interface 时会递归带入其依赖的 imported types，使 `BaseClientOptions -> SessionCredentials` 能生成 `{ accessKey, accessSecret }` 必填字段。
8. [x] 修复 constructor 默认参数和集合类型：imported class constructor mock 对默认参数传 `undefined`，避免 `FilterExpression(filterType = FilterType.TAG)` 被误填字符串；新增 `Set<T>` typed mock，保持 `Map<K,V>` / `Set<T>` 都按泛型 value 生成。
9. [x] 复跑 RocketMQ `src/producer/PublishingSettings.ts` top2：脚本通过，`status_counts={"passed":2}`，`action_counts={"ready":2}`，确认 `Settings as SettingsPB` alias、`Endpoints`、`ExponentialBackoffRetryPolicy` 和 `Set<string>` 构造链路有效。
10. [x] 复跑 RocketMQ `src/consumer/SimpleSubscriptionSettings.ts` top2：脚本通过，`status_counts={"passed":2}`，`action_counts={"ready":2}`，确认 `Settings as SettingsPB` alias、`Map<string, FilterExpression>`、`FilterExpression` 默认参数链路有效。
11. [x] 修复 RocketMQ load balancer 的 `TopicRouteData` 最小有效状态：针对 `PublishingLoadBalancer` / `SubscriptionLoadBalancer` 构造函数生成带 `READ_WRITE` 权限、非空 `messageQueues` 的 `TopicRouteData` mock，避免 `new TopicRouteData([])` 触发 “No writable/readable message queue found”。
12. [x] 复跑 RocketMQ `src/producer/PublishingLoadBalancer.ts` top6：脚本通过，`status_counts={"passed":6}`，`action_counts={"ready":6}`，`zero_skip=6`，`skipped_total=0`；复跑 `src/consumer/SubscriptionLoadBalancer.ts` top1：脚本通过，`status_counts={"passed":1}`，`action_counts={"ready":1}`，`zero_skip=1`，`skipped_total=0`。
13. [x] 修复 `PublishingMessage.toProtobuf` 最小有效构造：`MessageOptions` 按目标分支只生成 `tag` / `deliveryTimestamp` / `messageGroup` / `properties` 所需字段，`PublishingSettings` 使用 `{ maxBodySizeBytes } as PublishingSettings` 避免级联构造无关依赖，`mq` 使用 `{ queueId: 0 } as MessageQueue` 避免错误调用 protobuf wrapper 构造器。
14. [x] 复跑 RocketMQ `src/message/PublishingMessage.ts` top4：脚本通过，`status_counts={"passed":4}`，`action_counts={"ready":4}`，`zero_skip=4`，`skipped_total=0`。
15. [x] 将 `Producer` 剩余普通失败收敛为闭环分类问题，而不是继续堆 RocketMQ 专用生成规则：新增 `manual_review_external_service` action，识别 live RPC、外部服务、路由状态或长重试时序导致的 timeout / connection refused / gRPC unavailable / signal killed 等失败，并在 metadata 中返回 `external_service_dependent` 和 `external_service_reason`。
16. [x] 为 `validate_coverage_task` 新增可选 `TESTLOOP_VALIDATE_TASK_TIMEOUT_SECONDS`，并让自定义 JS 测试命令使用独立进程组和 `WaitDelay`，避免 `sh -c` 下的 `egg-bin/mocha` 子进程树在 context 超时后继续拖住 `run_tests`。
17. [x] 补充 handler 回归测试：`Producer.send` 遇到 Mocha timeout/gRPC sendMessage 形态时返回 `failed/manual_review_external_service`，不再暴露为普通 `repair_generated_test`；自定义 JS shell 命令超时时会快速返回结构化 runner failure。
18. [x] 补强 JS 真实项目验证脚本的阶段级可观测性：`TestValidateJSCoverageTopTasks` 会即时输出 baseline copy/link/coverage、任务 copy、任务 validate 的 start/done 日志；新增 `TESTLOOP_VALIDATE_JS_STAGE_TIMEOUT_SECONDS`、`TESTLOOP_VALIDATE_JS_BASELINE_TIMEOUT_SECONDS`、`TESTLOOP_VALIDATE_JS_TASK_TIMEOUT_SECONDS`，并让 baseline coverage runner 也使用 context 和进程组取消。
19. [x] 用新增阶段日志复验 RocketMQ `Producer.ts` top1，确认原卡点在 `task.validate -> generate_tests -> jsImportedTypeMocks`，不是 baseline coverage 或 runner；修复 imported type mock 的循环 import 访问集，避免 RocketMQ route/producer/client 等互相 re-export 时无限递归。
20. [x] 复跑 RocketMQ `src/producer/Producer.ts` top6：脚本通过，`status_counts={"failed":2,"passed":4}`，`action_counts={"manual_review_external_service":2,"manual_review_private":4}`，`zero_skip=2`，`skipped_total=4`；剩余失败已从普通 repair 队列收敛为外部服务/RPC 手审。
21. [x] 扩大到 RocketMQ top100 复验：第一轮脚本通过，`status_counts={"failed":6,"passed":94}`，`action_counts={"ready":61,"manual_review_private":33,"manual_review_external_service":6}`，`repair_generated_test=0`；但失败文本暴露其中部分并非真正外部服务，而是 `ILogger` 方法字段、`TransactionChecker.check(): Promise<TransactionResolution>` 返回值和可选 `Transaction` 参数导入污染这类通用 TS mock 缺口。
22. [x] 补强 TypeScript interface 方法字段 mock：interface 中的 `info(...): void`、`check(...): Promise<Enum>` 这类方法签名不再被字段解析跳过，会生成 no-op function；函数返回 Promise enum 时会返回 enum 成员，而不是 `undefined`。
23. [x] 支持 imported enum value mock 和递归 value import：`.d.ts` / TS 中的 `export enum` 会被解析为最小合法 enum 成员，value import 会递归扫描本地和 imported type decl，但不会把只出现在函数参数类型里的 class 误导入，避免 TS `noUnusedLocals`。
24. [x] 修复可选 typed 参数默认构造策略：没有覆盖任务显式指定值时，`transaction?: Transaction` 这类可选参数优先传 `undefined`，不再因为 typed constructor mock 生成 `new Transaction(undefined)` 或未使用 import。
25. [x] 复跑 RocketMQ `src/producer/Producer.ts` top6：脚本通过，`status_counts={"failed":2,"passed":4}`，失败已从 TS 编译错误收敛为运行期外部服务/RPC 断言或 gRPC deadline，继续归入 `manual_review_external_service`。
26. [x] 复跑 RocketMQ Node.js TS/Mocha top100：脚本通过，`status_counts={"failed":3,"passed":97}`，`action_counts={"ready":64,"manual_review_private":33,"manual_review_external_service":3}`，`repair_generated_test=0`；相比上一轮 `ready=61/external=6`，新增 3 个真实 ready，剩余 3 个 external 分别是 `Producer.onRecoverOrphanedTransactionCommand` 的 unwanted rejection 和 `Producer.send` 的 gRPC deadline/name resolution。

已完成补充：第七真实样本的 RocketMQ top100 已经把普通 `repair_generated_test` 清零，并进一步把 TS 编译类残留从 6 个 external 中剥离出来：interface 方法字段、Promise enum 返回值、嵌套 enum value import、函数参数类型误导入和可选 typed 参数构造都已通用化。当前剩余失败是更接近真实外部服务/运行期依赖的 Producer RPC 路径。下一步建议进入第八真实样本，优先选择 TypeScript/Vitest 或 TypeScript/Jest 的非 RocketMQ 项目，验证这些跨模块类型与函数 mock 规则没有对 Mocha/RocketMQ 过拟合。

## 第一百六十三阶段：第八个真实 TypeScript 项目样本验证

1. [x] 选择 RocketMQ 之外的 TypeScript/Vitest 样本：临时克隆 `unjs/ufo` 到 `/tmp/testloop-ufo-sample`，该项目是 URL/string utility 库，使用 Vitest，基线 `pnpm vitest run --coverage --coverage.reporter=json` 通过，13 个 test files、489 个 tests 通过。
2. [x] 初跑 UFO/Vitest top9：`status_counts={"failed":4,"generation_error":1,"passed":4}`，`action_counts={"inspect_generation_error":1,"manual_review_internal":1,"ready":3,"repair_generated_test":4}`；问题集中在 `resolveURL` statement gap 被误生成 error assertion、`toASCII(o)` 无类型参数传 `undefined`、barrel `index.ts` no-runtime 模板缺少 Vitest globals import、JSON fixture 被当成 coverage task。
3. [x] 补强 JS/TS coverage task 生成：JS coverage task 过滤 `.json` 等非代码文件；Vitest file-level manual-review/no-runtime 模板补 `import { describe, it } from 'vitest'`；函数级覆盖任务只有 `gap_type=error_path` 才按错误断言生成，不再因为函数体存在 `throw` 就让普通 statement gap 走 `toThrow()`。
4. [x] 补强无类型字符串工具函数：对明显需要字符串输入的单参数 URL/query/parse/encode/decode/ascii/text/slug 类函数生成 `'test'`；返回类型覆盖只收窄到 ascii/encode/decode/stringify/text/slug，避免把 `createURL`、`parseQuery` 这类 object 返回误断言为 string。
5. [x] 为新增能力补单元测试：覆盖 JS coverage task 跳过非代码文件、Vitest no-runtime 文件级模板导入 globals、statement gap 不强制 error assertion、`toASCII(o)` 使用字符串输入和 string 断言、`parseQuery(parametersString)` 保持 object 返回断言。
6. [x] 复跑 UFO/Vitest top8：脚本通过，`status_counts={"passed":8}`，`action_counts={"ready":6,"manual_review_internal":1,"manual_review_no_runtime":1}`，`repair_generated_test=0`，`zero_skip=6`，`skipped_total=2`；JSON fixture 被过滤后任务上限为 8。

已完成补充：第八真实样本证明当前 JS/TS 链路在非业务 SDK、纯工具库、Vitest runner 下也能形成稳定闭环。本阶段新增的价值点不是项目特化，而是三类通用质量门槛：覆盖率任务不能把非代码 fixture 当作可生成测试目标；file-level no-runtime/manual-review 草稿也必须是目标框架可执行的合法测试文件；函数体存在 `throw` 只能说明它有错误路径，不能把普通 statement/return gap 都改成错误断言。UFO 样本还暴露了 minified/untyped 函数的常见输入问题，已通过窄启发式处理字符串工具函数，且避免污染 `createURL`、`parseQuery` 这类 object 返回 API。下一步建议进入第九真实样本，优先验证 Python/pytest 或 Go 之外的 JS/TS 应用型项目，重点观察复杂对象 mock、异步依赖和框架入口的泛化能力。

## 第一百六十四阶段：第九个真实项目样本与闭环质量复核

1. [x] 选择第九个真实样本：临时克隆 `pallets/click` 到 `/tmp/testloop-click-sample`。该项目是 Python/pytest 库项目，源码采用 `src/click` layout，适合验证 Python coverage task 的包路径 import、pytest runner 和真实分支输入构造。
2. [x] 新增 Python 真实项目验证入口：`scripts/validate-py-coverage-top-tasks.sh` + `TestValidatePyCoverageTopTasks`，支持隔离复制样本、运行自定义 coverage 命令、解析 `coverage.json`、逐 task 生成并验证 pytest 测试；新增 `TESTLOOP_VALIDATE_PY_*` 环境变量，覆盖任务上限、输出 JSONL、baseline/task timeout、文件过滤和 coverage 命令模板。
3. [x] 确认可控基线：Click 在隔离副本中使用 `PYTHONPATH=src` 和 `python3 -m pytest --cov=src/click --cov-report=json {args}` 能生成 coverage JSON；当前可选 coverage tasks 为 370 个。
4. [x] 初跑 Click/Pytest top5 暴露 Python 链路基础问题：coverage 解析使用原项目路径导致源码映射失效，`src` layout 被生成成 `from parser import ...` 这类错误 import，多行 coverage task 注释会破坏 Python 缩进，constructor 参数缺失导致 `_Option.process` / `PacifyFlushWrapper.flush` 不能执行。
5. [x] 修复 Python coverage task 基础能力：coverage 解析切换到隔离 baseline cwd，`src/<pkg>/...` 生成 dotted package import；coverage 注释统一压缩为单行并截断；Python parser 保留 `__init__` 元数据但生成测试时不直接测试它；class method 任务只保留匹配 class 的 `__init__`。
6. [x] 补强 Python 真实分支输入：`_Option.process` 生成最小合法 `_Option(None, ['--test'], None, action='unknown')`，`PacifyFlushWrapper.flush` 生成会抛 `OSError` 的 wrapper，`_unpack_args` error path 生成 `['a'], [-1, -1]`，`stream/wrapped/iterable` 等参数生成更合理的默认值。
7. [x] 复跑 Click/Pytest top5：脚本通过，`status_counts={"passed":5}`，`action_counts={"ready":5}`，`zero_skip=5`，`skipped_total=0`。
8. [x] 扩大到 Click/Pytest top20：初跑剩余 6 个普通失败，集中在 `safecall` decorator wrapper、`make_str` Unicode fallback、`_FixupStream.readable` exception fallback、`get_binary_stderr` 进程 std stream 环境分支，以及 `ProgressBar.format_bar/render_progress` 的内部状态构造。
9. [x] 补强 Python coverage task 质量：`safecall` 通过抛错函数验证 wrapper swallow exception；`make_str` 临时 monkeypatch `sys.getfilesystemencoding` 覆盖 Unicode fallback；`_FixupStream.readable` 构造 `read(0)` 抛错的 stream 并断言 `False`；`ProgressBar` 使用未知长度 iterable、TTY/autowidth 状态和临时 terminal size monkeypatch 覆盖目标分支。
10. [x] 新增 Python 环境依赖手审分类：`get_binary_stdout/stderr/stdin` 这类依赖当前进程标准流 binary wrapper 状态的分支生成 `manual_review_environment` pytest skip；`validate_coverage_task` 识别 `manual_review_environment:` 标记并返回 `passed/manual_review_environment`。
11. [x] 为新增能力补回归测试：覆盖 Python `src` layout import、单行 coverage 注释、constructor/error-path 输入、Click 暴露的 `safecall` / `make_str` / `_FixupStream` / `ProgressBar` 模式，以及验证器对 Python `manual_review_environment` marker 的识别。
12. [x] 复跑 Click/Pytest top20：脚本通过，`status_counts={"passed":20}`，`action_counts={"ready":19,"manual_review_environment":1}`，`zero_skip=19`，`skipped_total=1`；普通 `repair_generated_test` 已清零。
13. [x] 新增 `docs/real-project-validation.md`，复查前九个样本的指标表，整理中文质量报告，说明当前稳定能力、普通 repair 清零情况，以及 `manual_review_*` 的手审边界。

已完成补充：第九真实样本把 Python/pytest 链路从“能生成基础 pytest 草稿”推进到真实项目 top20 闭环可用。Click 样本暴露的问题都具有通用意义：`src` layout 包路径、coverage 路径映射、注释安全、constructor 元数据、wrapper/fallback/error path 输入、内部状态对象构造，以及 OS/runtime 标准流分支的明确手审分类。当前 Click top20 已达到 `passed=20`，其中 19 个为真实 ready 测试，1 个为 `manual_review_environment`。

补充完成：九个真实样本指标已经沉淀到 `docs/real-project-validation.md`，README 也补充了 Python/pytest 验证脚本用法。下一步建议进入第二个 Python 应用型项目样本，重点验证 fixtures、monkeypatch、HTTP client、数据库策略和项目自定义 pytest runner。

## 第一百六十五阶段：第二个 Python/pytest 应用型项目样本验证

1. [x] 选择第二个 Python 样本：`/Users/binlee/code/open-source/codex/sdk/python`。该项目是 Python SDK，源码采用 `src/openai_codex` layout，pytest 测试覆盖运行时 API、客户端 RPC 方法、状态机通知流和输入转换等应用型逻辑。
2. [x] 建立可重复 baseline：复制到 `/tmp/testloop-codex-python-sample`，执行 `uv sync --group test` 并安装 `pytest-cov`；使用 `PYTHONPATH=src python3 -m pytest tests/test_public_api_runtime_behavior.py tests/test_client_rpc_methods.py --cov=src/openai_codex --cov-report=json`，基线 `21 passed`，coverage JSON 解析出 380 个 coverage tasks。
3. [x] 优化 Python 真实项目验证脚本：隔离复制从通用 `copyTree` 改为 Python 专用 `copyPyProjectTree`，跳过 `.venv`、`.pytest_cache`、`.ruff_cache`、`.tox`、`__pycache__`、`coverage.json`、`htmlcov` 等虚拟环境和缓存目录，避免每个 task 复制 249MB 级别虚拟环境。
4. [x] 初跑 Codex SDK Python top10：`status_counts={"failed":9,"passed":1}`，普通失败集中在 `_GoalOperationState` 构造、`_logical_notification` 和 `_logical_completion` 的通知对象/keyword-only 参数。补强后复跑 top10 达到 `status_counts={"passed":10}`，`action_counts={"ready":10}`，`zero_skip=10`。
5. [x] 扩大到 top20：新增普通失败集中在 `_GoalStreamCursor.process`、`_GoalStreamCursor._completion`、`_GoalNotificationStream._finish` 和 `_AsyncGoalNotificationStream._finish` 的状态机构造。补强真实 `TurnStartedNotification`、`TurnCompletedNotification`、`ThreadGoalClearedNotification`、stream finish unregister/cancel 断言后，top20 达到 `status_counts={"passed":20}`，`action_counts={"ready":20}`，`zero_skip=20`。
6. [x] 扩大到 top30：新增普通失败集中在 `_GoalOperationState.observe` 的通知 payload 构造、`_split_user_agent` 的字符串输入和 `_to_wire_input` 的 `TextInput` 输入。补强 `observe` 的 started/completed/goal-updated 通知分支、`user_agent` 语义默认值和 `_to_wire_input` 专用输入后，top30 达到 `status_counts={"passed":30}`，`action_counts={"ready":30}`，`zero_skip=30`，普通 `repair_generated_test` 清零。
7. [x] 为新增能力补回归测试：覆盖 Codex goal state、logical notification/completion、goal stream cursor/stream finish、`observe` 通知构造和 `_to_wire_input` 输入转换，防止重新退化为 `None` / `{}` 占位参数。
8. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 Codex SDK Python 作为第十个真实项目样本纳入当前质量边界。

已完成补充：第十真实样本证明 Python/pytest 链路已经不只适用于 Click 这类工具库，也能处理 SDK 内部状态机、Pydantic 通知对象、dataclass 输入类型、keyword-only 参数、同步/异步 stream finish 这类更接近应用运行时的逻辑。Codex SDK Python top30 已达到 `ready=30`、`skipped_total=0`，并把普通生成失败从 9 个逐步压到 0。下一步建议继续扩大 Python 验证窗口或切换到带 HTTP client / 数据库 / fixtures 的服务型 Python 项目，验证依赖注入和外部资源手审分类。

## 第一百六十六阶段：Python/pytest 服务型 HTTP 项目样本验证

1. [x] 选择第十一个真实样本：临时克隆 `encode/starlette` 到 `/tmp/testloop-starlette-sample`。该项目是 ASGI/HTTP 框架，pytest 子集覆盖 routing、requests、authentication、config、datastructures 等服务型入口。
2. [x] 建立可重复 baseline：执行 `uv sync --group dev --no-group docs` 并补安装 `pytest-cov`；使用 `python3 -m pytest tests/test_routing.py tests/test_requests.py --cov=starlette --cov-report=json --cov-config=/dev/null`，基线 `171 passed`，coverage JSON 只包含 `starlette` 源码包，解析出 744 个 coverage tasks。
3. [x] 初跑 Starlette top10：未加源码包 import 修复时 `status_counts={"failed":10}`，普通失败主要是仓库根包布局 `starlette/_utils.py` 被误生成 `from _utils import ...`，以及 coverage 配置把 `tests/` 文件纳入任务队列。
4. [x] 修复 Python root package import 推断：当源码路径位于连续带 `__init__.py` 的包目录中时，生成完整 dotted import，例如 `from starlette._utils import get_route_path`；保留 `src/` / `lib/` fallback。Starlette 验证命令显式使用 `--cov-config=/dev/null`，避免上游 coverage 配置把 `tests/` 当作待补测源码。
5. [x] 复跑 Starlette top10：提升到 `status_counts={"failed":3,"passed":7}`；剩余失败集中在 `get_route_path(scope)` 的 ASGI scope 输入和 `has_required_scope(conn, scopes)` 的认证 scope 输入。
6. [x] 补强服务型 Python 输入构造：为 `get_route_path` 按 line range 生成不同 `scope`，覆盖无 `root_path` 和根路径剥离返回；为 `has_required_scope` 生成带 `auth.scopes` 的最小连接对象，同时断言缺失 scope 返回 `False` 和满足 scope 返回 `True`。
7. [x] 复跑 Starlette top10：达到 `status_counts={"passed":10}`，`action_counts={"ready":10}`，`zero_skip=10`。
8. [x] 扩大到 Starlette top20：新增两个普通失败集中在 `Config._read_file` 临时配置文件输入和 `Config._perform_cast` bool/int cast 分支。
9. [x] 补强配置类输入构造：`Config._read_file` 生成临时 env 文件并断言注释过滤、引号剥离和值解析；`Config._perform_cast` 断言 bool、int 正常 cast 和非法 bool 值的 `ValueError`。
10. [x] 复跑 Starlette top20：达到 `status_counts={"passed":20}`，`action_counts={"ready":20}`，`zero_skip=20`，普通 `repair_generated_test` 清零。
11. [x] 扩大到 Starlette top30：新增三个普通失败集中在 `MultiDict.pop`、`MultiDict.popitem` 和 `MultiDict.setdefault`，原因是生成器使用空 `MultiDict()`，无法触发有状态的列表/字典同步分支。
12. [x] 补强可变多值字典输入构造：为 `MultiDict.pop`、`popitem` 和 `setdefault` 生成带重复 key 的初始状态，并断言返回值与 `multi_items()` 状态变化。
13. [x] 复跑 Starlette top30：达到 `status_counts={"passed":30}`，`action_counts={"ready":30}`，`zero_skip=30`，普通 `repair_generated_test` 继续清零。
14. [x] 为新增能力补回归测试：覆盖 root package import、ASGI route scope、auth scope、临时配置文件读取、bool/int cast 和 `MultiDict` 状态化断言，防止服务型 Python 项目重新退化为 `None`、空实例或 basename import。
15. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 Starlette top30 作为第十一个真实项目样本纳入当前质量边界。
16. [x] 扩大到 Starlette top50：初跑 `status_counts={"failed":19,"passed":31}`，新增普通失败集中在 `UploadFile` keyword-only 构造、`MutableHeaders.add_vary_header(None)`、`HTTPEndpoint` / `WebSocketEndpoint` 的 ASGI 协议对象，以及 `MultiPartParser` keyword-only 构造和当前 part 状态。
17. [x] 补强 Starlette 文件上传与 header 输入构造：`UploadFile` 使用 `SpooledTemporaryFile` 和 keyword 参数，分别覆盖已落盘、即将超过内存阈值、同步/线程池写入、read/seek/close；`MutableHeaders.add_vary_header` 使用已有 `Vary` header 并断言追加结果。
18. [x] 补强 ASGI endpoint 输入构造：`HTTPEndpoint.dispatch` 生成最小 HTTP scope / receive / send，并通过 async handler、sync handler、HEAD fallback 和 405 分支覆盖目标；`HTTPEndpoint.method_not_allowed` 同时覆盖 plain ASGI response 与带 `app` scope 的 `HTTPException`。
19. [x] 补强 WebSocket endpoint 输入构造：`WebSocketEndpoint.decode` 覆盖 text、bytes、json text、json bytes、默认编码和错误关闭分支；`dispatch` 使用子类记录 `on_receive` / `on_disconnect`，覆盖 receive、disconnect 和异常 close_code 分支。
20. [x] 补强 multipart parser 状态机输入：`MultiPartParser.on_part_data` 使用真实 `Headers`、async stream 和 `max_part_size`，同时覆盖正常 data append 与超限异常；`on_part_end` 设置当前 part 的 field/data 并断言 items。
21. [x] 为新增能力补回归测试：覆盖 `UploadFile`、`MutableHeaders`、`HTTPEndpoint`、`WebSocketEndpoint` 和 `MultiPartParser`，防止退回到 `None` 参数、错误位置参数、空实例或无协议状态调用。
22. [x] 复跑 Starlette top50：脚本通过，`status_counts={"passed":50}`，`action_counts={"ready":50}`，`zero_skip=50`，`skipped_total=0`，普通 `repair_generated_test` 清零。
23. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 Starlette 验证窗口从 top30 提升到 top50，并把文件上传、header、ASGI endpoint、WebSocket 和 multipart 状态机纳入 Python/pytest 当前质量边界。

已完成补充：第十一个真实样本把 Python/pytest 链路推进到 ASGI/HTTP 服务型场景。Starlette 样本暴露的通用问题包括仓库根包 import、coverage 配置污染测试文件、ASGI scope 最小输入、认证 scope 对象、配置文件读取和类型转换、可变多值字典的状态化断言、keyword-only 文件上传对象、已有 header 追加、ASGI endpoint 最小协议对象、WebSocket decode / dispatch 状态机，以及 multipart parser 当前 part 状态。当前 Starlette top50 已达到 `ready=50`、`skipped_total=0`，普通生成失败清零。下一步建议切换到带业务依赖的 Python 服务项目，重点验证 pytest fixtures、monkeypatch、HTTP client、数据库策略、外部服务依赖和项目自定义 runner；如果继续 Starlette，则应扩大到 responses、middleware、background task 和 TestClient 相关分支。

## 第一百六十七阶段：业务型 Python/FastAPI 服务样本验证

1. [x] 选择第十二个真实样本：`/Users/binlee/code/free-works/haoy-apk-station/backend`。该项目是 FastAPI + SQLAlchemy + SQLite + JWT/API Key + 文件上传 + 对象存储的业务后端，比框架样本更接近真实服务依赖。
2. [x] 建立隔离验证环境：复制业务后端到 `/tmp/testloop-haoy-apk-backend-sample`，创建 `/tmp/testloop-haoy-apk-backend-venv`，安装 FastAPI、SQLAlchemy、pytest、pytest-cov、httpx、qiniu、apk-info 等依赖；`ve-tos-python-sdk` 当前 PyPI 不可安装，因此测试基线将 `STORAGE_BACKEND` 切到 `qiniu` 并 monkeypatch 上传函数，避免真实对象存储依赖。
3. [x] 在 `/tmp` 样本中建立临时 pytest 业务基线：使用独立 SQLite 文件、FastAPI `TestClient`、JWT 登录、API Key 创建、非 APK 文件上传、应用列表和普通用户管理员权限校验；基线命令 `/tmp/testloop-haoy-apk-backend-venv/bin/python -m pytest tests/test_business_flow.py --cov=app --cov-report=json --cov-config=/dev/null` 通过，`2 passed`。
4. [x] 初跑 haoy-apk-station top10：`status_counts={"failed":10}`，全部失败都是 `ModuleNotFoundError: No module named 'app'`。根因是该业务后端没有 `pyproject.toml` / `setup.py` / `pytest.ini`，生成测试位于 `tests/app/utils/test_*.py`，`run_tests` 的 pytest root discovery 停在测试子目录，导致裸 `app` 包不在 import path。
5. [x] 修复 pytest 项目根推断：当没有配置标记且测试路径位于嵌套 `tests/...` 下时，向上找到最近的 `tests` 目录，并在其父目录存在本地 Python package（如 `app/__init__.py`）时使用父目录作为 pytest cwd；新增 `run_tests` 回归测试覆盖 `tests/app/utils/test_app.py` 这类无配置 FastAPI/Flask 布局。
6. [x] 复跑 top10：提升为 `status_counts={"failed":2,"passed":8}`，剩余普通失败集中在 `_fallback_from_filename(apk_path, result)`，原因是默认生成 `result=None`，不能覆盖会原地写入 dict 的 fallback helper。
7. [x] 补强 APK parser 输入构造：`_find_icon_in_zip` 生成临时 zip/apk 并写入 `xxxhdpi/ic_launcher.png` 或 `launcher.png`；`_fallback_from_filename` 生成可变 dict 并断言包名/应用名写入；`parse_apk` 对 `APK_INFO_AVAILABLE=False` 分支生成模块级临时覆盖，对正常解析分支生成 `FakeAPK` 和 `_extract_icon` fake，断言 package/version/app/icon 字段。
8. [x] 为新增能力补回归测试：覆盖无配置 pytest root fallback、APK zip 输入、fallback dict 和 fake APK SDK，防止回退到 `ModuleNotFoundError`、`None` 参数或无语义的 `parse_apk('test')`。
9. [x] 复跑 haoy-apk-station top10：脚本通过，`status_counts={"passed":10}`，`action_counts={"ready":10}`，`zero_skip=10`，普通 `repair_generated_test` 清零。
10. [x] 扩大到 haoy-apk-station top20：脚本通过，`status_counts={"passed":20}`，`action_counts={"ready":20}`，`zero_skip=20`，`skipped_total=0`；新增任务覆盖 `_extract_icon` 和 `upload_apk` 早期业务分支，没有新增普通失败。
11. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station backend 作为第十二个真实项目样本纳入当前质量边界，并明确这是基于 `/tmp` 临时 pytest 业务基线的隔离验证。
12. [x] 扩大到 haoy-apk-station top40：脚本通过，`status_counts={"passed":40}`，`action_counts={"ready":40}`，`zero_skip=40`，`skipped_total=0`；新增任务覆盖 `_get_download_url`、`get_version_detail`、`download_apk`、版本更新和删除等业务分支。
13. [x] 扩大到 top60 初跑发现 9 个普通失败，集中在 `build_version_out` detached app fallback、`_delete_icon_file` 空 URL / key 提取，以及 `short_link_page` 的 DB session / HTMLResponse 分支；这些失败说明生成器仍会把服务 helper 误当纯函数，用 `None` 参数调用。
14. [x] 补强 FastAPI 服务 helper 输入构造：`build_version_out` 生成带属性和 detached `app` property 的 fake version；`_delete_icon_file` 区分空 URL 和有效 CDN URL，并 monkeypatch `delete_file`；`short_link_page` 生成最小 fake SQLAlchemy query 链，覆盖链接不存在、应用下架、当前版本缺失时取最新版本、无版本、文件大小未知、图标和 release notes HTML 分支。
15. [x] 为新增能力补回归测试：覆盖 FastAPI version output、icon cleanup 和 short link page 的六类分支，防止重新退化为 `build_version_out(None, None)`、`_delete_icon_file('https://example.com')` 或 `short_link_page(None, None)`。
16. [x] 复跑 haoy-apk-station top60：脚本通过，`status_counts={"passed":60}`，`action_counts={"ready":60}`，`zero_skip=60`，`skipped_total=0`；普通 `repair_generated_test` 再次清零。
17. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station backend 验证窗口从 top20 提升到 top60，并把下载 URL、版本详情、下载重定向、删除图标、隐藏应用、下载统计、短链页面和 auth refresh helper 纳入 Python/pytest 当前质量边界。
18. [x] 扩大到 haoy-apk-station top80 初跑：`status_counts={"failed":5,"passed":75}`，`action_counts={"ready":75,"repair_generated_test":5}`；新增失败集中在 `get_current_user_by_api_key` 空 request、动态定义的 `serve_frontend`、`_migrate_short_code_to_app` 迁移 helper 和 `generate_qr_data_url` error path。
19. [x] 补强 FastAPI 认证、迁移和 QR 输入构造：`get_current_user_by_api_key` 生成带空 headers/query_params 的 fake request；`generate_qr_data_url` monkeypatch `builtins.__import__` 触发 qrcode import fallback 并断言空字符串；`_migrate_short_code_to_app` 生成 fake Session/inspect/text result，覆盖两个“app 已有短码则跳过”分支。
20. [x] 将动态前端入口归类为环境手审：`serve_frontend` 只在 `frontend/dist` 存在时才会在 `app.main` import 阶段定义，生成器不再调用 `lifespan(None)` 伪覆盖，而是生成 `manual_review_environment` skip，提示需要在导入 `app.main` 前创建前端 dist 的集成 fixture。
21. [x] 修复 Python return 表达式安全判断：`return  # comment` 不再被当成可断言表达式，避免生成 `assert result == (# comment)` 这类语法错误。
22. [x] 为新增能力补回归测试：覆盖 API Key fake request、QR import fallback、动态前端手审分类和迁移 helper fake DB，并防止回退到 `None` 参数或无效 comment 断言。
23. [x] 复跑 haoy-apk-station top80：脚本通过，`status_counts={"passed":80}`，`action_counts={"ready":79,"manual_review_environment":1}`，`zero_skip=79`，`skipped_total=1`；普通 `repair_generated_test` 清零。
24. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station backend 验证窗口从 top60 提升到 top80，并把 API Key request fake、QR import fallback、数据库迁移 fake Session/inspect 和动态前端入口环境手审纳入 Python/pytest 当前质量边界。
25. [x] 扩大到 haoy-apk-station top100 初跑：`status_counts={"failed":1,"passed":99}`，`action_counts={"ready":98,"manual_review_environment":1,"repair_generated_test":1}`；唯一普通失败是 `short_link_page` 第 877 行最终 `HTMLResponse` 返回路径退化为 `short_link_page(None, None)`。
26. [x] 补强 FastAPI 短链最终返回路径：将 `short_link_page` 第 877 行映射到已有 rich HTML fake DB 模板，复用带应用、版本、HTML escape 和 release notes 的 SQLAlchemy query fake，并补充回归测试防止回退到 `None` 参数。
27. [x] 复跑 haoy-apk-station top100：脚本通过，`status_counts={"passed":100}`，`action_counts={"ready":99,"manual_review_environment":1}`，`zero_skip=99`，`skipped_total=1`；普通 `repair_generated_test` 再次清零。
28. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station backend 验证窗口从 top80 提升到 top100，并把短链页面最终 HTMLResponse 返回路径纳入 Python/pytest 当前质量边界。
29. [x] 扩大到 haoy-apk-station top120 初跑：`status_counts={"failed":5,"passed":115}`，`action_counts={"ready":114,"manual_review_environment":1,"repair_generated_test":5}`；普通失败集中在 `list_api_keys`、`list_users`、`build_app_out`、`list_apps` 的 DB query 链，以及动态定义的 `serve_root_file`。
30. [x] 补强 FastAPI 业务列表和 app output 输入构造：`list_users` / `list_api_keys` 生成最小 fake DB query；`build_app_out` 生成两次 `.first()` 共享状态，覆盖当前版本缺失后 fallback 到最新版本；`list_apps` 生成支持 `filter/count/order_by/offset/limit/all` 的 fake query，并断言搜索分支和分页参数。
31. [x] 将动态根静态文件入口归类为环境手审：`serve_root_file` 与 `serve_frontend` 一样只在 `frontend/dist` 存在时于 import 阶段定义，生成器输出 `manual_review_environment`，提示需要导入前创建 dist 的集成 fixture。
32. [x] 为新增能力补回归测试：覆盖认证列表、API Key 列表、应用输出 fallback、应用列表搜索和动态根静态文件手审分类，防止回退到 `None` DB 或 `asyncio.run(lifespan(None))`。
33. [x] 复跑 haoy-apk-station top120：脚本通过，`status_counts={"passed":120}`，`action_counts={"ready":118,"manual_review_environment":2}`，`zero_skip=118`，`skipped_total=2`；普通 `repair_generated_test` 再次清零。
34. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station backend 验证窗口从 top100 提升到 top120，并把认证/API Key 列表 fake DB、应用列表搜索 fake DB、`build_app_out` fallback 查询和根静态文件环境手审纳入 Python/pytest 当前质量边界。
35. [x] 扩大到 haoy-apk-station top140 初跑：`status_counts={"failed":2,"passed":138}`，`action_counts={"ready":136,"manual_review_environment":2,"repair_generated_test":2}`；普通失败只剩 `short_link_page` 第 811 行当前版本查询和第 835 行 HTML 模板赋值两个细粒度 statement 行段。
36. [x] 补强短链页面细粒度 statement 映射：将 `short_link_page` 第 811 行和第 835 行复用已有 rich HTML fake DB 模板，确保当前版本查询、HTML escape、图标、二维码和 release notes 路径都能覆盖，并补充回归测试防止回退到 `short_link_page(None, None)`。
37. [x] 复跑 haoy-apk-station top140：脚本通过，`status_counts={"passed":140}`，`action_counts={"ready":138,"manual_review_environment":2}`，`zero_skip=138`，`skipped_total=2`；普通 `repair_generated_test` 再次清零。新增 top120-top140 覆盖应用详情、版本列表、下载 APK 多条 fallback、删除应用、下载统计聚合和短链页面细粒度 statement。
38. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station backend 验证窗口从 top120 提升到 top140，并把短链当前版本查询和 HTML 模板赋值纳入 Python/pytest 当前质量边界。
39. [x] 扩大到 haoy-apk-station top160 初跑：`status_counts={"failed":2,"passed":158}`，`action_counts={"ready":155,"manual_review_environment":3,"repair_generated_test":2}`；普通失败集中在 `auth_service.get_user_by_api_key` 的 API Key 查询假 DB 和 `verify_refresh_token` 的 token 类型/JWT 错误路径，新增一个动态静态资源入口继续归类为环境手审。
40. [x] 补强 FastAPI auth service 输入构造：`get_user_by_api_key` 生成支持 `query/filter/first/commit` 的最小 fake DB，同时覆盖 API Key 不存在和存在时更新 `last_used_at`；`verify_refresh_token` 生成 access token、refresh token 和非法 JWT 三条路径，避免继续用 `None` 触发第三方库内部 AttributeError。
41. [x] 为新增能力补回归测试：覆盖 auth service API Key fake DB 和 refresh/access token 分支，防止重新退化到 `get_user_by_api_key(None, 'test')` 或 `verify_refresh_token(None)`。
42. [x] 复跑 haoy-apk-station top160：脚本通过，`status_counts={"passed":160}`，`action_counts={"ready":157,"manual_review_environment":3}`，`zero_skip=157`，`skipped_total=3`；普通 `repair_generated_test` 再次清零。新增 top140-top160 覆盖 auth service、refresh token 和 Qiniu 对象存储 helper 分支。
43. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station backend 验证窗口从 top140 提升到 top160，并把 auth service API Key fake DB、JWT refresh/access token 分支和 Qiniu helper fallback 纳入 Python/pytest 当前质量边界。
44. [x] 扩大到 haoy-apk-station top180：脚本一次通过，`status_counts={"passed":180}`，`action_counts={"ready":177,"manual_review_environment":3}`，`zero_skip=177`，`skipped_total=3`；普通 `repair_generated_test` 保持清零。新增 top160-top180 覆盖 Qiniu `move_file/download_to_temp/upload_file/upload_bytes`、storage backend 选择、TOS client 初始化、public URL、delete/move/download/upload helper 等对象存储路径。
45. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station backend 验证窗口从 top160 提升到 top180，并把 storage backend 选择和 TOS 对象存储 helper fallback 纳入 Python/pytest 当前质量边界。
46. [x] 扩大到 haoy-apk-station top200 初跑：`status_counts={"failed":17,"passed":183}`，`action_counts={"ready":180,"manual_review_environment":3,"repair_generated_test":17}`；普通失败集中在 Qiniu/TOS 对象存储 helper 的错误路径，根因是这些 helper 多数返回 `(False, message)` 或 fallback URL，而不是抛异常，通用 `error_path` 模板误用了 `pytest.raises`。
47. [x] 补强 Qiniu/TOS 对象存储输入构造：按 `qiniu_service.py` / `tos_service.py` 识别 `upload_file`、`upload_bytes`、`get_private_url`、`delete_file`、`generate_upload_token`、`move_file`、`download_to_temp` 和 TOS `_get_client`，生成 fake SDK module、fake bucket/client、临时文件、fallback local save 和返回值断言，避免继续用 `None` 参数或错误异常断言硬跑。
48. [x] 为新增对象存储能力补回归测试：覆盖 Qiniu `upload_bytes/move_file/download_to_temp` 和 TOS `_get_client/move_file`，防止回退到 `pytest.raises(Exception)`、真实 SDK `BucketManager` 或 `move_file('test', 'test')` 这类无效输入。
49. [x] 复跑 haoy-apk-station top200：脚本通过，`status_counts={"passed":200}`，`action_counts={"ready":197,"manual_review_environment":3}`，`zero_skip=197`，`skipped_total=3`；普通 `repair_generated_test` 再次清零。新增 top180-top200 覆盖 auth decode/authenticate error path、Qiniu upload/delete/token/move/download 错误返回值，以及 TOS upload/private URL/token/move/download/client 初始化路径。
50. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station backend 验证窗口从 top180 提升到 top200，并把 Qiniu/TOS 对象存储错误返回值和 fake SDK/client fallback 纳入 Python/pytest 当前质量边界。
51. [x] 扩大到 haoy-apk-station top220 初跑：`status_counts={"failed":4,"passed":216}`，`action_counts={"ready":213,"manual_review_environment":3,"repair_generated_test":4}`；普通失败集中在 `_save_icon_local` bytes 参数误生成 dict、`get_qiniu_auth` 在已安装 qiniu 的环境下不会抛异常、`create_api_key` 缺少 fake DB，以及 `decode_token` statement 行段误传 `None`。
52. [x] 补强 Qiniu/auth service 输入构造：`_save_icon_local` 使用临时目录和 bytes 并断言写入文件；`get_qiniu_auth` monkeypatch `builtins.__import__` 触发 qiniu SDK 缺失路径；`create_api_key` 使用 fake DB 覆盖 API Key 唯一性循环；`decode_token` 使用真实 access token 覆盖默认和 `verify_exp=False` 解码分支。
53. [x] 为新增能力补回归测试：覆盖本地图标保存、Qiniu SDK import fallback、API Key 唯一性循环和 JWT 解码 statement 行段，防止回退到 `_save_icon_local('test', {})`、`pytest.raises` 裸跑、`create_api_key(None, 1, 'test')` 或 `decode_token(None, None)`。
54. [x] 复跑 haoy-apk-station top220：脚本通过，`status_counts={"passed":220}`，`action_counts={"ready":217,"manual_review_environment":3}`，`zero_skip=217`，`skipped_total=3`；普通 `repair_generated_test` 再次清零。新增 top200-top220 覆盖 storage facade 返回路径、TOS/Qiniu public URL、本地图标保存、Qiniu SDK 缺失路径、API Key 创建唯一性循环和 JWT `verify_exp=False` 分支。
55. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station backend 验证窗口从 top200 提升到 top220，并把本地图标保存、Qiniu SDK 缺失路径、API Key 唯一性循环 fake DB 和 JWT `verify_exp=False` 解码分支纳入 Python/pytest 当前质量边界。
56. [x] 扩大到 haoy-apk-station top240：脚本一次通过，`status_counts={"passed":240}`，`action_counts={"ready":237,"manual_review_environment":3}`，`zero_skip=237`，`skipped_total=3`；普通 `repair_generated_test` 保持清零。新增 top220-top240 主要覆盖 Qiniu token 区域推断、move/download statement 路径、`is_configured` / `_is_qiniu_configured`、storage facade `_get_backend/is_configured`，以及 TOS public URL/delete/token statement 路径。
57. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station backend 验证窗口从 top220 提升到 top240，并把对象存储配置状态和 storage facade 返回路径纳入 Python/pytest 当前质量边界。
58. [x] 扩大到 haoy-apk-station top260 初跑：`status_counts={"failed":1,"passed":259}`，`action_counts={"ready":256,"manual_review_environment":3,"repair_generated_test":1}`；唯一普通失败是 `main.py` 文件级任务被映射到 `lifespan` 后仍走通用 async 模板，生成了 `asyncio.run(lifespan(None))`，但 FastAPI `lifespan` 是 `@asynccontextmanager`，返回 async context manager 而不是 coroutine。
59. [x] 补强 FastAPI `lifespan` 输入构造：普通 `lifespan` coverage task 生成 `async with lifespan(None)` 包裹，并 monkeypatch `init_db`、`SessionLocal` 和 `ensure_admin_exists` 隔离启动副作用；动态 `serve_frontend` / `serve_root_file` 仍保持 `manual_review_environment`，因为它们取决于 import 前是否存在 `frontend/dist`。
60. [x] 为新增能力补回归测试：覆盖 `main.py` 文件级任务的 `lifespan` async context manager 模板，防止重新退化到 `asyncio.run(lifespan(None))`。
61. [x] 复跑 haoy-apk-station top260：脚本通过，`status_counts={"passed":260}`，`action_counts={"ready":257,"manual_review_environment":3}`，`zero_skip=257`，`skipped_total=3`；普通 `repair_generated_test` 再次清零。新增 top240-top260 覆盖 TOS move/download/client/bucket/config/upload 路径、Qiniu auth/config 文件级路径、APK parser 文件级路径，以及 FastAPI `lifespan` 启动钩子隔离。
62. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station backend 验证窗口从 top240 提升到 top260，并把 FastAPI `lifespan` async context manager 启动钩子隔离纳入 Python/pytest 当前质量边界。
63. [x] 尝试扩大到 haoy-apk-station top280：当前 pytest baseline 过滤后只有 267 个 coverage tasks，因此 top280 不能运行，说明该样本在现有基线下任务池已经接近耗尽。
64. [x] 扩大到 haoy-apk-station top267 初跑：`status_counts={"failed":1,"passed":266}`，`action_counts={"ready":263,"manual_review_environment":3,"repair_generated_test":1}`；唯一普通失败是 `ReleaseNotesUpdate` 这类 Pydantic/BaseModel DTO 空类 file-level 任务生成了没有测试方法的空测试类，导致 `IndentationError`。
65. [x] 修复 Python 空类 coverage task 生成：当 class 只有 `__init__` 或没有可测方法时，生成可执行的 import smoke 测试，并保留 coverage task 注释；新增回归测试防止重新生成空测试类。
66. [x] 复跑 haoy-apk-station top267 后仍有 1 个普通失败：`apps.py` file-level 任务映射到 `short_link_page` 后走了默认 `short_link_page(None, None)`，在真实 SQLAlchemy fake DB 需求下触发 `None.query`。
67. [x] 修复 `short_link_page` entire-file 任务：file-level 也复用 rich HTML fake DB 模板，覆盖当前版本查询、HTML escape 和 `HTMLResponse` 返回路径；新增回归测试固定 `short_link_page('rich', FakeDB(...))`。
68. [x] 复跑 haoy-apk-station top267：脚本通过，`status_counts={"passed":267}`，`action_counts={"ready":264,"manual_review_environment":3}`，`zero_skip=264`，`skipped_total=3`；普通 `repair_generated_test` 再次清零。最后 7 个任务覆盖 `tos_service.py` 模块导入/配置行、`apps.py` file-level 短链路径、`qiniu_service.py` file-level 和 `tos_service.py` file-level。
69. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station backend 验证窗口从 top260 提升到 top267，并明确当前 baseline 的 coverage task 池已跑满。
70. [x] 扩充 haoy-apk-station pytest baseline：在 `/tmp` 隔离样本中加入真实 APK 上传成功、同版本 build version、图标上传和非 APK 上传失败数据库回滚测试；baseline 从 `2 passed` 扩到 `4 passed`，`apps.py` 覆盖率从 37.7% 提升到 55.1%。
71. [x] 扩 baseline 后尝试 top280：由于新增测试吃掉旧缺口，剩余 coverage tasks 从 267 降到 248，top280 仍不能运行；改跑 expanded top248。
72. [x] expanded top248 初跑：`status_counts={"failed":1,"passed":247}`，`action_counts={"ready":244,"manual_review_environment":3,"repair_generated_test":1}`；唯一普通失败是 `get_current_user_without_raise` 的 error path 被误生成 `pytest.raises(Exception)`，但该 helper 设计上会吞异常并返回 `None`。
73. [x] 修复 auth helper 吞异常路径：`get_current_user_without_raise` coverage task 生成无 token 返回 `None`、`decode_token` 抛错返回 `None`、正常 token + fake DB 返回用户的三段断言，并补回归测试防止退化为 `pytest.raises`。
74. [x] 复跑 expanded top248：脚本通过，`status_counts={"passed":248}`，`action_counts={"ready":245,"manual_review_environment":3}`，`zero_skip=245`，`skipped_total=3`；普通 `repair_generated_test` 再次清零。
75. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station 最新验证窗口改为 expanded top248，并把真实 APK 上传、同版本 build、图标上传、上传失败回滚和 auth helper 吞异常路径纳入当前质量边界。
76. [x] 继续扩充 haoy-apk-station pytest baseline：在 `/tmp` 隔离样本中加入下载统计聚合、版本删除后版本和下载日志清理、应用删除后版本和图标清理、短链隐藏/无版本页面，以及 refresh token 成功/禁用用户路径；baseline 从 `4 passed` 扩到 `7 passed`。
77. [x] 更新 baseline 覆盖率观测：`apps.py` 覆盖率提升到 67.4%，`auth.py` 为 73.3%，`auth_service.py` 为 81.8%；因为新基线吃掉旧缺口，剩余 coverage task 池从 248 降到 231，expanded top248 不再可运行。
78. [x] expanded2 top231 初跑：`status_counts={"failed":1,"passed":230}`，`action_counts={"ready":227,"manual_review_environment":3,"repair_generated_test":1}`；唯一普通失败是 `short_link_page` 缺失应用返回路径的精确行段 `782-782` 没有命中已有 `780-782` fake DB 模板，退化为 `short_link_page(None, None)`。
79. [x] 修复短链缺失应用精确行段：`short_link_page` 的 missing app 模板同时匹配 `780-782` 和 `782-782`，并新增回归测试固定 `short_link_page('missing', FakeDB(None))`，防止细粒度 return line 再次退化到空 DB。
80. [x] 复跑 expanded2 top231：脚本通过，`status_counts={"passed":231}`，`action_counts={"ready":228,"manual_review_environment":3}`，`zero_skip=228`，`skipped_total=3`；普通 `repair_generated_test` 再次清零。同步更新 `docs/real-project-validation.md` 和质量评估，把最新窗口改为 expanded2 top231。
81. [x] 继续扩充 haoy-apk-station pytest baseline：在 `/tmp` 隔离样本中加入发布说明长度限制和保存、设置当前版本、应用详情/版本列表 not found、下载流式成功路径、登录错误/禁用账号、删除用户 self/not found/成功路径；baseline 从 `7 passed` 扩到 `10 passed`。
82. [x] 更新 baseline 覆盖率观测：`apps.py` 覆盖率提升到 79.9%，`auth.py` 为 82.5%，`auth_service.py` 为 83.0%；因为新基线继续吃掉旧缺口，剩余 coverage task 池从 231 降到 208，expanded2 top231 不再可运行。
83. [x] 复跑 expanded3 top208：脚本通过，`status_counts={"passed":208}`，`action_counts={"ready":205,"manual_review_environment":3}`，`zero_skip=205`，`skipped_total=3`；普通 `repair_generated_test` 保持清零，没有暴露新的生成器缺口。
84. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station 最新验证窗口改为 expanded3 top208，并把发布说明、当前版本、not found、下载流式成功、登录错误/禁用和删除用户边界纳入 Python/pytest 当前质量边界。
85. [x] 继续扩充 haoy-apk-station pytest baseline：在 `/tmp` 隔离样本中加入 API Key 删除/不存在/list 更新、非 APK 重复上传版本递增、上传大小限制，以及 storage facade 的 qiniu/tos 后端分发；baseline 从 `10 passed` 扩到 `13 passed`。
86. [x] 更新 baseline 覆盖率观测：`apps.py` 覆盖率提升到 80.6%，`auth.py` 为 88.3%，`storage.py` 为 82.1%；因为新基线继续吃掉旧缺口，剩余 coverage task 池从 208 降到 168，expanded3 top208 不再可运行。
87. [x] 复跑 expanded4 top168：脚本通过，`status_counts={"passed":168}`，`action_counts={"ready":165,"manual_review_environment":3}`，`zero_skip=165`，`skipped_total=3`；普通 `repair_generated_test` 保持清零，没有暴露新的生成器缺口。
88. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 haoy-apk-station 最新验证窗口改为 expanded4 top168，并把 API Key 删除、重复上传、大小限制和 storage facade 分发纳入 Python/pytest 当前质量边界。

已完成补充：第十二个真实样本把 Python/pytest 链路推进到带业务依赖的 FastAPI 后端。它暴露了无 pyproject/test config 的项目根推断问题，以及文件/zip 输入、原地写入 dict helper、外部 SDK fake、对象存储依赖隔离、服务 helper 不能用 `None` 参数硬调、SQLAlchemy query 链最小 fake、认证/API Key 列表 fake DB、auth service API Key fake DB、JWT refresh/access token 分支、应用列表搜索 fake DB、`build_app_out` fallback 版本查询、HTMLResponse 分支断言、短链当前版本查询、短链最终返回路径、模块级 monkeypatch、API Key request fake、QR import fallback、数据库迁移 fake Session/inspect、Qiniu helper fallback、storage backend 选择、Qiniu/TOS 对象存储错误返回值、fake SDK/client fallback、本地图标保存、Qiniu SDK 缺失路径、API Key 唯一性循环 fake DB、JWT `verify_exp=False` 解码分支、对象存储配置状态、storage facade 返回路径、Pydantic/BaseModel DTO 空类 file-level smoke、FastAPI `lifespan` async context manager 启动钩子隔离、短链 file-level fake DB、真实 APK 上传成功路径、同版本 build version、图标上传、上传失败数据库回滚、下载统计聚合、版本删除/应用删除数据库一致性、短链隐藏/无版本页面、refresh token 成功/禁用用户路径、发布说明长度限制和保存、设置当前版本、应用详情/版本列表 not found、下载流式成功路径、登录错误/禁用账号、删除用户 self/not found/成功路径、API Key 删除/不存在/list 更新、非 APK 重复上传版本递增、上传大小限制、storage facade qiniu/tos 分发、auth helper 吞异常返回 None、短链缺失应用精确行段 `782-782` fake DB，以及动态定义路由不能伪装普通 ready 这些真实服务中常见的生成质量缺口。当前 haoy-apk-station backend expanded4 top168 已达到 `ready=165`、`manual_review_environment=3`、`skipped_total=3`，普通生成失败清零。下一步建议切换第十三个真实样本，继续验证跨项目泛化能力。

## 第一百六十八阶段：轻量 Python 库绑定样本验证

1. [x] 选择第十三个真实样本：`/Users/binlee/code/open-source/ip2region/binding/python`。该样本是轻量 Python binding/library，包含 IP 解析、大小端 helper、Header/Version 对象和 in-memory/file/vector-index 三种 xdb searcher 路径，和前一个 FastAPI 业务样本形态明显不同。
2. [x] 建立隔离验证样本：复制到 `/tmp/testloop-ip2region-python-sample`，创建 `/tmp/testloop-ip2region-python-venv` 并安装 pytest/pytest-cov；原始 `util_test.py` 依赖 `../../data/*.xdb`，因此在 `/tmp` 样本中补充 `tests/test_core_flow.py`，用内存构造最小 IPv4 xdb buffer，避免依赖外部二进制数据文件。
3. [x] 建立 pytest baseline：`/tmp/testloop-ip2region-python-venv/bin/python -m pytest -q tests/test_core_flow.py --cov=ip2region --cov-report=json --cov-config=/dev/null` 通过，`7 passed`，coverage 约 `95.71%`。
4. [x] 初跑 top9 发现两个真实问题：per-task 验证阶段仍使用系统 `python3`，没有继承 baseline 的 venv pytest；环境修复后仍只有 `ready=1/9`，普通失败集中在 Python 静态生成器把 bytes/header/version/callable/searcher 构造参数退化成 `None` 或不可用文件路径。
5. [x] 修复 pytest 自定义 runner：`run_tests` 的 pytest 分支支持 `TESTLOOP_PYTEST_COMMAND`，可用 `{path}`、`{verbose}`、`{coverage}` 指定 venv 或项目自定义命令；`scripts/validate-py-coverage-top-tasks.sh` 同步记录该环境变量。
6. [x] 补强 Python 静态生成器通用默认值：按参数名生成 bytes/buffer/vector index、offset、callable compare、header-like 对象和 version-like 对象；为 `_v4_sub_compare`、`ip_sub_compare` 和 `version_from_header` 生成可运行且更贴近目标分支的输入。
7. [x] 为新增能力补回归测试：覆盖自定义 pytest 命令模板、bytes compare helper、header object 和 Searcher-like constructor，防止退回到系统 pytest、`None` 参数、无效文件路径或不可调用 compare。
8. [x] 复跑 ip2region Python binding top9：脚本通过，`status_counts={"passed":9}`，`action_counts={"ready":9}`，`zero_skip=9`，`skipped_total=0`；普通 `repair_generated_test` 清零。
9. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 ip2region Python binding 作为第十三个真实项目样本纳入当前质量边界。

已完成补充：第十三个真实样本把 Python/pytest 链路从业务型 FastAPI 服务扩展到轻量库和二进制 buffer/searcher 场景。它暴露了验证环境和生成器输入启发式两类问题：baseline 可以用自定义 venv 命令跑通，但 per-task runner 也必须显式跟随；静态生成器不能继续把 bytes、buffer、header、version、callable 和 searcher constructor 参数硬填 `None`。当前 ip2region Python binding top9 已达到 `ready=9`、`skipped_total=0`，普通生成失败清零。下一步建议选择第十四个真实样本，优先验证非 Python 场景或 Python 的另一类库形态，确认这批通用启发式不会只对 ip2region 有效。

## 第一百六十九阶段：Rust 覆盖率任务验证入口

1. [x] 梳理 Rust/Java 当前能力：`run_tests`、`parse_results`、`parse_coverage`、`generate_tests` 已有主路径，但真实项目验证脚本只有 Go/JS/Python，Rust 缺少和现有 top-task 闭环同级的验证入口。
2. [x] 修复 Rust coverage task 写入安全问题：Rust 生成器输出的是内联 `#[cfg(test)] mod tests { use super::*; }`，当 coverage task 的 `test_file` 指向源文件时，`generate_tests` 原逻辑会覆盖 `.rs` 源码；现在 `.rs` 目标文件已存在时会追加生成测试模块，保留原源码。
3. [x] 为 Rust 追加写入补回归测试：覆盖 `coverage_task.test_file == source` 的场景，断言原始 `pub fn add` 仍存在，生成的 `#[cfg(test)]` / `use super::*` 模块被追加，且生成用例数仍可统计。
4. [x] 新增 `scripts/validate-rust-coverage-top-tasks.sh`：接口对齐 Go/JS/Python 验证脚本，复制项目到隔离 worktree，运行 LCOV coverage 命令，解析 top coverage tasks，对每个 task 复制新 worktree 后执行 `validate_coverage_task`，并输出 JSONL。
5. [x] 新增 `TestValidateRustCoverageTopTasks`：支持 `TESTLOOP_VALIDATE_RUST_COVERAGE_COMMAND`、`TESTLOOP_VALIDATE_RUST_COVERAGE_FILE`、`TESTLOOP_VALIDATE_RUST_FILE_FILTER` 和 baseline/task timeout，默认 coverage 命令仍是 `cargo tarpaulin --out Lcov --output-dir target/tarpaulin`。
6. [x] 跑通最小 Rust smoke：在 `/tmp/testloop-rust-minimal-sample` 构造临时 Cargo crate，用自定义 LCOV writer 触发 `add` 的 top2 coverage task；验证结果为 `status_counts={"passed":2}`，`action_counts={"ready":2}`，`zero_skip=2`，`skipped_total=0`。
7. [x] 尝试准备真实 Rust coverage 工具：`cargo-tarpaulin` 本机未安装，`rustup component add llvm-tools-preview` 下载超过两分半未完成后中断；因此本阶段不把 Rust smoke 计入真实项目样本表，避免把合成 LCOV 误报成真实覆盖率。
8. [x] 安装 `cargo-llvm-cov v0.8.7`，并用 `cargo llvm-cov --help` 确认真实 LCOV 命令为 `cargo llvm-cov --lcov --output-path target/llvm-cov/lcov.info`。
9. [x] 尝试 CodeInsight-mcp 真实 Rust baseline：复制当前 checkout 后运行 `cargo llvm-cov`，但 coverage 编译超过 3 分钟无输出，手动中断；该样本当前不适合作为小窗口首个 Rust 真实验证。
10. [x] 尝试 apk-info workspace 的 `apk-info-zip` 包：`cargo test` 本地通过，`cargo llvm-cov -p apk-info-zip --lcov --output-path target/llvm-cov/lcov.info` 进入 `llvm-tools-preview` 安装流程；下载超过 2 分钟被 timeout 杀掉。
11. [x] 修复 Rust coverage command timeout：自定义 coverage 命令现在配置进程组，和 JS/Python 自定义命令一致，避免 shell 被杀但 cargo/rustup 子进程继续悬挂；新增 `TestRustCoverageCommandKillsProcessGroupOnTimeout` 回归测试。
12. [x] 复验 timeout：apk-info 的 `cargo llvm-cov` 在缺少 `llvm-tools-preview` 时能按 `TESTLOOP_VALIDATE_RUST_STAGE_TIMEOUT_SECONDS=120` 可靠退出，不再卡住验证进程。

已完成补充：Rust 现在具备和 Go/JS/Python 一致形态的 opt-in top-task 验证入口，并修掉了 coverage task 写回源文件时覆盖源码的风险。当前只完成最小 Cargo crate smoke，尚未完成第十四个真实项目样本；主要 blocker 是 `cargo llvm-cov` 需要的 `llvm-tools-preview` 组件在当前网络下无法完成下载。下一步应先完成 `rustup component add llvm-tools-preview --toolchain stable-aarch64-apple-darwin`，或改用可用的 `cargo tarpaulin`/预生成 LCOV，再选择 apk-info `apk-info-zip` 或 CodeInsight-mcp 跑真实 LCOV top window，并据结果决定是补 Rust 生成器输入推断，还是把复杂 Path/自定义类型场景归为手审/骨架。

## 第一百七十阶段：MCP 结构化输出统一

1. [x] 在 Rust 真实 LCOV 工具链继续被 `llvm-tools-preview` 下载阻塞时，转向不依赖外部工具链的 Agent 闭环基础设施改进。
2. [x] 新增 `structuredToolResult` / `structuredToolResultWithError` helper，统一生成 `TextContent` JSON 与 `StructuredContent`，保持旧客户端读文本 JSON 的兼容性，同时让新 Agent 可以直接消费结构化字段。
3. [x] 将 `generate_tests`、`run_tests`、`parse_results`、`fix_suggestions`、`parse_coverage` 和 `validate_coverage_task` 的成功/业务错误返回改为统一结构化结果；`generate_tests` provider error 和 `validate_coverage_task` generation/run error 仍保留 `IsError` 语义。
4. [x] 为 `parse_results`、`parse_coverage`、`generate_tests`、`run_tests` 和 `fix_suggestions` 补 handler 级结构化返回断言；`validate_coverage_task` 已有结构化输出断言，继续沿用。
5. [x] 本地验证：`go test ./...` 和 `git diff --check` 通过。
6. [x] 补 e2e 层结构化输出断言：`callTool` / `callToolRaw` 现在要求真实 MCP session 返回非空 `structuredContent`，并验证其 JSON 语义和 `content[0].text` 完全一致；`go test ./test/e2e -count=1` 通过。

已完成补充：主 MCP 工具现在不再只依赖文本 JSON 承载结构化数据，Agent 可以优先读 `structuredContent`，旧消费方仍可继续读 `content[0].text`。handler 层和真实 MCP session 层都已固定该契约，这比继续堆“测试生成器”能力更贴合项目定位，因为它降低了 Codex/Claude/Cursor 类 Agent 在测试闭环中的解析成本和字段漂移风险。下一步建议回到真实样本扩展：优先解除 Rust LCOV 工具链 blocker；如果 `llvm-tools-preview` 仍不可用，则启动 Java top-task 验证入口，补齐 Java 与 Go/JS/Python/Rust smoke 同级的验证脚本。

## 第一百七十一阶段：Java 覆盖率任务验证入口

1. [x] 复查 Rust 工具链状态：本机仍只有 `cargo-llvm-cov`，缺少 `llvm-tools-preview`，`cargo-tarpaulin` 也不可用，因此 Rust 真实 LCOV 样本继续阻塞。
2. [x] 新增 `scripts/validate-java-coverage-top-tasks.sh`：接口对齐 Go/JS/Python/Rust 验证脚本，复制 Java 项目到隔离 baseline，运行 JaCoCo XML coverage 命令，解析 top coverage tasks，对每个 task 复制 fresh worktree 后执行 `validate_coverage_task`，并输出 JSONL。
3. [x] 新增 `TestValidateJavaCoverageTopTasks`：支持 `TESTLOOP_VALIDATE_JAVA_COVERAGE_COMMAND`、`TESTLOOP_VALIDATE_JAVA_COVERAGE_FILE`、`TESTLOOP_VALIDATE_JAVA_FILE_FILTER`、baseline/task/stage timeout 和 Maven/Gradle 默认 coverage runner。
4. [x] 为 Java 自定义 coverage 命令补进程组 timeout 回归测试，避免 shell 被杀但 Maven/Gradle 子进程继续悬挂；同时补 `TESTLOOP_VALIDATE_JAVA_COVERAGE_COMMAND` 命令模板测试。
5. [x] 修复 Java 验证路径重写：JaCoCo XML 常见文件路径是 `com/example/Foo.java`，验证时会优先映射到隔离 worktree 的 `src/main/java/com/example/Foo.java` 或 `src/test/java/...`；新增回归测试固定该映射。
6. [x] 跑通最小 Java smoke：构造临时 Maven/JUnit 项目和自定义 JaCoCo XML writer，fake `mvn` 返回 JUnit 摘要，执行 `scripts/validate-java-coverage-top-tasks.sh <tmp-project> 1 <tmp-jsonl>`，结果为 `status_counts={"passed":1}`、`action_counts={"ready":1}`、`zero_skip=1`、`skipped_total=0`。
7. [x] 更新 README、真实项目验证质量报告和质量评估，说明 Java top-task 入口、环境变量和当前 smoke 边界；Java smoke 不计入真实项目样本表。

已完成补充：Java 现在具备和 Go/JS/Python/Rust 同形态的 opt-in top-task 验证入口，并且已通过最小 Maven/JUnit + JaCoCo XML smoke 证明 `parse_coverage -> validate_coverage_task -> generate_tests -> run_tests` 能闭环。当前还不是第十四个真实样本，因为 coverage 和 test runner 都用的是可控 fake；下一步应选择一个真实 Java 项目，优先 Maven + JaCoCo、依赖可本机解析、测试窗口较小的项目，跑真实 top window，观察 Java 生成器在 package/import、构造函数、非静态方法、异常路径和集合/Optional 入参上的普通 repair。

## 第一百七十二阶段：Java 真实 Maven 多模块样本验证

1. [x] 选择第十四个真实样本：`/Users/binlee/code/free-works/haoying/rocketmq-clients/java`。该项目是 Maven 多模块 Java client，使用 JUnit 4、JaCoCo、Checkstyle、SpotBugs 和 parent aggregator，比最小 smoke 更接近真实企业 Java 项目。
2. [x] 建立真实 baseline 命令：使用 `mvn -q -pl client -am -DfailIfNoTests=false -Dtest=EndpointsTest test` 生成 `client/target/site/jacoco/jacoco.xml`，并用 `TESTLOOP_VALIDATE_JAVA_FILE_FILTER=route/Endpoints.java` 缩小首个验证窗口。
3. [x] 修复 JaCoCo 多模块路径映射：`org/apache/.../Endpoints.java` 这类 package path 会映射到隔离 worktree 的 `client/src/main/java/...`；对应测试文件会写回同一 module 的 `client/src/test/java/...`，避免落到根目录。
4. [x] 修复 Java 项目根发现：`run_tests` 现在能从尚未创建的深层测试文件路径向上查找 `pom.xml`，并把查找深度放宽到 16 层，覆盖 `src/test/java/org/apache/...` 这类真实包路径。
5. [x] 支持 Maven aggregator 运行：当父级 `pom.xml` 声明当前 module 时，Java runner 会从聚合根目录执行 `mvn -pl <module> -am -DfailIfNoTests=false test`，避免子模块单跑时找不到 sibling module 依赖。
6. [x] 补强 Java 生成器项目适配：复用源文件 license header 和 package，按 Maven/Gradle 依赖识别 JUnit 4/5，改用显式 `Assert`/`Assertions` 导入，生成 `public class` 和 `public void` 以兼容旧 Surefire，并截断过长 coverage task 注释以通过 Checkstyle。
7. [x] 补强 Java coverage task 定位和输入：按 `line_range` 选择重载方法/构造函数；解析 Java `Enum.CONST.equals(param)` 提示；对 `AddressScheme.DOMAIN_NAME` + `List<Address>` 构造函数分支生成 `assertThrows` 和两元素地址列表，避免无效 `new AddressScheme()` 和反向 `assertNotNull`。
8. [x] 复跑 RocketMQ Java client `route/Endpoints.java` top1：脚本通过，`status_counts={"passed":1}`，`action_counts={"ready":1}`，`skipped_total=1`；真实 Maven 输出显示 `Tests run: 211, Failures: 0, Errors: 0, Skipped: 1`，普通 `repair_generated_test` 清零。
9. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 RocketMQ Java client 作为第十四个真实项目样本纳入当前质量边界。
10. [x] 扩大到 RocketMQ Java client `route/Endpoints.java` top5：初跑为 `status_counts={"failed":4,"passed":1}`，普通失败集中在 parser 过早丢弃 `getGrpcTarget/equals`、无默认构造器实例方法退化为 `new Endpoints()`、`addresses.isEmpty` 空集合分支仍生成过长单行和无效 enum 构造。
11. [x] 修复 Java coverage task helper 目标：parser 保留 getter/equals/hashCode，普通生成阶段仍跳过 helper；coverage task 精确目标可命中 `getGrpcTarget`、`equals` 和 `hashCode`，避免回退整类生成。
12. [x] 补强 Java 实例方法和常见 helper 断言：无默认构造器时选择 String 构造器；`equals` 生成 self/null 分支断言；`hashCode` 对 `hash == 0` 分支断言计算结果非 0，不再错误期待初始值。
13. [x] 补强 Java protobuf constructor 输入：对 `apache.rocketmq.v2.Endpoints` 构造函数按任务生成 builder，覆盖空地址异常、switch/case IPv4 和 DOMAIN_NAME 多地址异常；对 `AddressScheme + List<Address>` 的空值路径生成 null list + `assertThrows`。
14. [x] 复跑 RocketMQ Java client `route/Endpoints.java` top10：脚本通过，`status_counts={"passed":10}`，`action_counts={"ready":10}`，`skipped_total=10`。`skipped_total` 来自该项目现有套件每次固定 1 个 upstream skipped test，不是生成测试跳过；普通 `repair_generated_test` 清零。
15. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 RocketMQ Java client 验证窗口从 top1 提升到 top10，并记录 getter/equals/hashCode、protobuf builder、空集合和空值分支能力。
16. [x] 扩大到 RocketMQ Java client `route/Endpoints.java` top20 初跑：`status_counts={"failed":9,"passed":11}`，普通失败集中在 protobuf 构造函数裸 `null` 重载歧义、`toSocketAddresses` 缺少 `List/InetSocketAddress` import、`equals` 返回路径断言方向错误，以及 `AddressScheme + List<Address>` 普通构造路径生成过长单行。
17. [x] 补强 Java line-range 生成策略：显式 coverage hint 优先，行号只作兜底；protobuf `Endpoints` 构造函数按未覆盖行选择空地址或 DOMAIN_NAME 多地址异常；`equals` 按返回行生成 self、异类 false 或同值 true 断言；`toSocketAddresses` 生成 IPv4 集合断言或 DOMAIN_NAME `assertNull`；`AddressScheme + List<Address>` 普通路径拆成多行地址列表。
18. [x] 修复 Java 生成 import 后处理：仅在出现未限定 `List<InetSocketAddress>` 时补 `java.util.List` / `java.net.InetSocketAddress`，并把 Java 标准库 import 插入到 JUnit import 之前，满足 RocketMQ Checkstyle 的 CustomImportOrder。
19. [x] 复跑 RocketMQ Java client `route/Endpoints.java` top20：脚本通过，`status_counts={"passed":20}`，`action_counts={"ready":20}`，`zero_skip=0`，`skipped_total=20`。`skipped_total` 仍来自 upstream 固定 skipped test；普通 `repair_generated_test` 清零。
20. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 RocketMQ Java client 验证窗口从 top10 提升到 top20，并记录 `toSocketAddresses`、line-range `equals`、import order、重载 null 和普通地址列表构造能力。

已完成补充：第十四个真实样本把 Java/JUnit 链路从最小 smoke 推进到真实 Maven 多模块项目，并把 `route/Endpoints.java` 验证窗口扩大到 top20。它暴露并修复了 JaCoCo package path、嵌套 Maven module、深层测试路径、aggregator 执行、JUnit 4 风格、Checkstyle 约束、构造函数重载、getter/equals/hashCode parser 可见性、无默认构造器实例方法、enum 参数、protobuf builder、空集合、空值、多地址异常分支、`List<InetSocketAddress>` import 顺序、`toSocketAddresses` 返回断言、line-range `equals` 断言和普通地址列表构造这些真实 Java 项目中的高频问题。下一步应优先扩大到 RocketMQ Java client 的更多文件或更大的 Java top window，观察 `toProtobuf`、Optional/外部 protobuf 类型和外部服务依赖；若 top window 中出现外部服务或路由状态依赖，应按项目定位归类为 `manual_review_external_service`，不要让静态生成器继续硬猜。

## 第一百七十三阶段：Java 多文件真实样本扩展

1. [x] 为 Java top-task 验证入口增加 `TESTLOOP_VALIDATE_JAVA_LIST_TASKS_ONLY`，只复制 baseline、运行 JaCoCo、解析并输出选中的 coverage tasks，不逐个执行 per-task 验证，便于先看真实任务分布再选择文件窗口。
2. [x] 用 RocketMQ Java client 生成 unfiltered top80 task 清单：共有 2502 个 coverage task，前 80 不再集中于 `Endpoints.java`，高优先级文件包括 `StatusChecker.java`、hook interceptor、`ClientImpl`、`ClientSessionImpl`、`ClientType` 和 consumer 相关类。
3. [x] 选择第二个 Java 文件 `exception/StatusChecker.java`，初跑 top2 为 `status_counts={"failed":2}`、`action_counts={"repair_generated_test":2}`，失败集中在 protobuf `Status` 不能 `new`、`RpcFuture` 无零参构造、`ClientException` / `Status` / `RpcFuture` 缺 import 或类型构造。
4. [x] 补强 `StatusChecker.check` coverage task 生成：用 protobuf builder 构造 `Status`，对普通 switch 分支使用 `Code.OK`，对 `future.getRequest` 分支使用 `Code.MESSAGE_NOT_FOUND` + `ReceiveMessageRequest`，并构造真实 `Metadata`、`Context` 和 `RpcFuture`，避免零值构造和 import order 风险。
5. [x] 为 `StatusChecker.check` 增加 Java 生成器回归测试，固定 OK switch 分支和 `MESSAGE_NOT_FOUND` receive request 分支的输出形态。
6. [x] 复跑 RocketMQ Java client `exception/StatusChecker.java` top2：脚本通过，`status_counts={"passed":2}`，`action_counts={"ready":2}`，`zero_skip=2`，`skipped_total=0`；普通 `repair_generated_test` 清零。
7. [x] 更新 `docs/real-project-validation.md` 和质量评估，把 RocketMQ Java client 从单文件 `Endpoints.java` 扩展为 `Endpoints.java` top20 + `StatusChecker.java` top2。
8. [x] 选择轻量 hook 文件 `hook/AttributeKey.java`，初跑 top2 为 `status_counts={"failed":2}`、`action_counts={"repair_generated_test":2}`，失败集中在私有构造函数被直接 `new AttributeKey(...)` 调用。
9. [x] 补强 Java 实例构造策略：只使用 public 构造器；当构造器不可用时识别 `create/of/from/valueOf` 这类返回当前类的 public static factory；为 `AttributeKey.equals` 增加私有构造 + 静态工厂回归测试。
10. [x] 复跑 RocketMQ Java client `hook/AttributeKey.java` top2：脚本通过，`status_counts={"passed":2}`，`action_counts={"ready":2}`，`zero_skip=0`，`skipped_total=2`；`skipped_total` 来自该项目现有 upstream skipped test。
11. [x] 选择 enum 文件 `impl/ClientType.java`，初跑 top4 为 `status_counts={"failed":4}`、`action_counts={"repair_generated_test":4}`，生成 preview 为空 `ClientTypeTest`，根因是 Java parser 没有递归进入 enum 的 `enum_body_declarations`。
12. [x] 补强 Java enum 方法解析和生成：tree-sitter class body walker 递归读取 enum body 内方法，coverage task 生成时用 enum 常量作为 receiver，并按任务中的 `PRODUCER` / `*_CONSUMER` 分支生成返回值断言；复跑 `impl/ClientType.java` top4 通过，`status_counts={"passed":4}`，`action_counts={"ready":4}`，`zero_skip=0`，`skipped_total=4`。
13. [x] 选择轻量 hook 状态类 `hook/InflightRequestCountInterceptor.java`，初跑 top2 为 `status_counts={"failed":2}`、`action_counts={"repair_generated_test":2}`，失败先暴露为 `Assert` 未使用 import，生成代码本身也会把 `MessageInterceptorContext` 接口误当成可实例化类型。
14. [x] 补强 Java hook 状态型生成：对 `InflightRequestCountInterceptor.doBefore/doAfter` 使用 `MessageInterceptorContextImpl(MessageHookPoints.RECEIVE)` 构造上下文，并断言 inflight count 从 0 到 1、再从 1 回到 0；同时补充 assertion import 清理，避免无断言 `void` 方法生成未使用 `Assert/Assertions`。
15. [x] 复跑 RocketMQ Java client `hook/InflightRequestCountInterceptor.java` top2：脚本通过，`status_counts={"passed":2}`，`action_counts={"ready":2}`，`zero_skip=0`，`skipped_total=2`；普通 `repair_generated_test` 清零。
16. [x] 选择组合 hook 文件 `hook/CompositedMessageInterceptor.java`，初跑 top2 为 `status_counts={"failed":2}`、`action_counts={"repair_generated_test":2}`，先暴露 `MessageInterceptorContext` 接口误实例化；补最小组合场景后又暴露匿名 interceptor 方法签名超过 RocketMQ Checkstyle 120 字符限制。
17. [x] 补强 Java 组合 hook 生成：为 `CompositedMessageInterceptor.doBefore/doAfter` 生成匿名 `MessageInterceptor`、`MessageInterceptorContextImpl` 和非空 interceptor 列表；`doAfter` 先调用 `doBefore` 建立 attribute map，再调用 `doAfter`，并断言对应 before/after 回调被触发；长方法签名拆行以满足 Checkstyle。
18. [x] 复跑 RocketMQ Java client `hook/CompositedMessageInterceptor.java` top2：脚本通过，`status_counts={"passed":2}`，`action_counts={"ready":2}`，`zero_skip=0`，`skipped_total=2`；普通 `repair_generated_test` 清零。
19. [x] 选择小型 impl/manager 文件 `impl/ClientManagerImpl.java`，初跑 top3 为 `status_counts={"failed":3}`、`action_counts={"repair_generated_test":3}`，失败不是业务构造问题，而是 private 方法 task 被过滤后生成空测试类并留下未使用 `org.junit.Test` import。
20. [x] 补强 Java private/internal coverage task 分类：当 coverage task 命中非 public Java 方法时生成可运行的 `manual_review_internal` skipped 测试，preview 中保留目标方法和手审原因，避免伪造 private 直接调用或空测试类。
21. [x] 复跑 RocketMQ Java client `impl/ClientManagerImpl.java` top3：脚本通过，`status_counts={"passed":3}`，`action_counts={"manual_review_internal":3}`，`zero_skip=0`，`skipped_total=6`；普通 `repair_generated_test` 清零。
22. [x] 扩大到 RocketMQ Java client `impl/ClientManagerImpl.java` top5：新增两条 `shutDown` 线程池 await 分支同样命中内部方法路径，脚本通过，`status_counts={"passed":5}`，`action_counts={"manual_review_internal":5}`，`zero_skip=0`，`skipped_total=10`；普通 `repair_generated_test` 继续清零。
23. [x] 切入 RocketMQ Java client `impl/ClientSessionImpl.java` top4：初跑为 `status_counts={"failed":3,"passed":1}`，失败集中在 public `release/onNext` 方法生成 `new ClientSessionImpl()`，但该类只有带 RPC session 依赖的 protected 构造器；`write` 已按非 public 方法归为 `manual_review_internal`。
24. [x] 补强 Java 实例方法构造兜底：coverage task 需要实例方法但类显式定义构造器、且生成器找不到可证明安全的 public 构造器或 public static factory 时，生成可运行的 `manual_review_internal` skipped 测试，不再回退到无参构造；手审说明拆成多行字符串以满足 RocketMQ Checkstyle 行长。
25. [x] 复跑 RocketMQ Java client `impl/ClientSessionImpl.java` top7：脚本通过，`status_counts={"passed":7}`，`action_counts={"manual_review_internal":7}`，`zero_skip=0`，`skipped_total=14`；RPC session、request observer 和 stream lifecycle 分支均稳定归为手审，普通 `repair_generated_test` 清零。
26. [x] 切入 RocketMQ Java client `impl/ClientImpl.java` top8：脚本通过，`status_counts={"passed":8}`，`action_counts={"manual_review_internal":8}`，`zero_skip=0`，`skipped_total=16`；抽象 client、startup/shutdown lifecycle、session table 和 heartbeat 路径均稳定归为手审，普通 `repair_generated_test` 清零。
27. [x] 切入 consumer value object：`impl/consumer/Assignment.java` top2 初跑为 `status_counts={"failed":2}`，失败集中在 unknown constructor arg `MessageQueueImpl` 被伪造成 `new MessageQueueImpl()`，测试文件缺 import 且无需真实对象。
28. [x] 补强 Java 默认参数策略：未知自定义引用类型默认用 `null`，短名类型不再 cast，避免为了构造 receiver 引入缺失 import 或无参构造假设；保留全限定类型的 null cast 以降低重载歧义。
29. [x] 复跑 RocketMQ Java client `impl/consumer/Assignment.java` top2：脚本通过，`status_counts={"passed":2}`，`action_counts={"ready":2}`，`zero_skip=0`，`skipped_total=2`；普通 `repair_generated_test` 清零。
30. [x] 复跑 RocketMQ Java client `impl/consumer/Assignments.java` top2：脚本通过，`status_counts={"passed":2}`，`action_counts={"ready":2}`，`zero_skip=0`，`skipped_total=2`；List 包装类 equals 分支保持真实 ready 测试。
31. [x] 切入 consumer 行为类 `impl/consumer/ConsumeTask.java` top1：初跑缺少源文件 import 的 `ConsumeResult`，修复 source import 按实际代码引用复制后进入运行阶段，又暴露 listener/messageView/interceptor 为 null 导致 NPE。
32. [x] 补强 Java source import 和 RocketMQ consumer fixture：生成测试只复制代码实际引用的源 import，忽略 coverage task 注释里的类型名；对 `ConsumeTask.call` 复用 `TestBase.fakeMessageViewImpl()`、lambda `MessageListener` 和 Mockito `MessageInterceptor`，覆盖 `ConsumeResult.FAILURE` 分支。
33. [x] 复跑 RocketMQ Java client `impl/consumer/ConsumeTask.java` top1：脚本通过，`status_counts={"passed":1}`，`action_counts={"ready":1}`，`zero_skip=0`，`skipped_total=1`；普通 `repair_generated_test` 清零。
34. [x] 切入 `impl/consumer/ConsumeService.java` top1：初跑暴露抽象类直接实例化、`null,null` 触发重载歧义和 coverage 注释误复制 `Duration` 无用 import；补匿名子类、executor/scheduler 和无延迟 `consume(messageView)` 生成。
35. [x] 复跑 RocketMQ Java client `impl/consumer/ConsumeService.java` top1：脚本通过，`status_counts={"passed":1}`，`action_counts={"ready":1}`，`zero_skip=0`，`skipped_total=1`；延迟分支的无延迟路径可生成真实 ready 测试。
36. [x] 切入 `impl/consumer/ConsumerImpl.java` top4：初跑脚本通过，`status_counts={"passed":4}`，`action_counts={"manual_review_internal":4}`；ACK private helper、change invisible RPC callback、filter expression private helper 和 receive response switch 均先被保守归为手审，普通 `repair_generated_test` 清零。
37. [x] 补强 Java ConsumerImpl filter expression 生成：当 coverage task 命中 `ConsumerImpl.wrapFilterExpression` 时，不再直接手审 private helper，而是通过同包可见的 `wrapReceiveMessageRequest` 生成真实 SQL92 `FilterExpression` 输入，并断言 protobuf `FilterType.SQL`；同时把 Java `manual_review_internal` metadata 改为 Java 语境，避免残留 JavaScript “not exported” 文案。
38. [x] 复跑 RocketMQ Java client `impl/consumer/ConsumerImpl.java` top4：脚本通过，`status_counts={"passed":4}`，`action_counts={"manual_review_internal":3,"ready":1}`；`wrapFilterExpression` 转为真实 ready，生成的 `ConsumerImplTest` 自身 `Tests run: 1, Skipped: 0`，剩余 skipped 来自项目既有 `StatusCheckerTest` 和手审测试；普通 `repair_generated_test` 继续清零。
39. [x] 扩大到 RocketMQ Java client `impl/consumer/ConsumerImpl.java` top12：初跑脚本通过，`status_counts={"passed":12}`，`action_counts={"manual_review_internal":11,"ready":1}`；新增任务集中在 `receiveMessage` return/switch、`changeInvisibleDuration` callback、private request wrapper 和 `ackMessage` 返回路径。
40. [x] 补强 Java ConsumerImpl RPC fixture：对 `ConsumerImpl.ackMessage` 生成 `PushConsumerImpl` spy、mock `ClientManager.ackMessage`、`TestBase.okAckMessageResponseFuture()` 和返回值断言；对 `ConsumerImpl.changeInvisibleDuration` 按 line range 选择 OK、非 OK 或失败 `RpcFuture`，覆盖 success/error/failure callback，并断言返回或异常。
41. [x] 复跑 RocketMQ Java client `impl/consumer/ConsumerImpl.java` top12：脚本通过，`status_counts={"passed":12}`，`action_counts={"manual_review_internal":7,"ready":5}`；`changeInvisibleDuration` 的 3 条 callback/return 任务和 `ackMessage` 返回路径已转为真实 ready，`ConsumerImplTest` ready 任务自身均 `Skipped: 0`，普通 `repair_generated_test` 继续清零。
42. [x] 补强 Java ConsumerImpl request builder：`wrapFilterExpression` 按 line range 区分 SQL/TAG 分支；两个 `wrapReceiveMessageRequest` overload 分别生成 auto-renew / invisible-duration request 断言，覆盖 batch size、attempt id、auto renew、invisible duration 和 filter type。
43. [x] 修复 Java top-task 验证脚本大窗口超时：新增 `TESTLOOP_VALIDATE_JAVA_GO_TEST_TIMEOUT`，默认 `30m`，避免 top20 这类慢速真实 Maven 窗口被 Go test 默认 10 分钟总超时截断。
44. [x] 复跑 RocketMQ Java client `impl/consumer/ConsumerImpl.java` top20：脚本通过，`status_counts={"passed":20}`，`action_counts={"manual_review_internal":11,"ready":9}`；`wrapFilterExpression` TAG path 和两个 `wrapReceiveMessageRequest` overload 已转为真实 ready，剩余手审集中在 `receiveMessage` response switch/stream lifecycle、ACK lite topic private wrapper 和 change invisible private wrapper。
45. [x] 补强 Java ConsumerImpl receiveMessage fixture：针对 `ConsumerImpl.receiveMessage` 生成 `PushConsumerImpl` spy、mock `ClientManager.receiveMessage`、自构造 `STATUS + DELIVERY_TIMESTAMP + MESSAGE` 的 `ReceiveMessageResponse` 列表，并断言 `ReceiveMessageResult` 消息数量和 `MessageViewImpl.getTransportDeliveryTimestamp()`，覆盖 response switch、protobuf message list、StatusChecker 成功路径和返回结果。
46. [x] 复跑 RocketMQ Java client `impl/consumer/ConsumerImpl.java` top20：脚本通过，`status_counts={"passed":20}`，`action_counts={"manual_review_internal":3,"ready":17}`；全部 `receiveMessage` 相关任务已转为真实 ready，剩余手审只集中在 `wrapAckMessageRequest` 和 `wrapChangeInvisibleDuration` 两类 private wrapper。

已完成补充：Java 真实样本已经开始从单文件收敛进入多文件扩展。新增 list-only 模式让后续选择验证窗口更稳，不必为了看 task 分布先跑完整 per-task 验证；新增 Go test 总超时配置后，top20 慢速 Maven 窗口也能完整收敛。`StatusChecker.java` top2 暴露的是静态方法、checked exception、protobuf builder、gRPC metadata 和 generic future/context 构造；`AttributeKey.java` top2 暴露的是私有构造器和静态工厂选择；`ClientType.java` top4 暴露的是 enum body 递归解析和 enum 常量 receiver 生成；`InflightRequestCountInterceptor.java` top2 暴露的是接口参数最小实现、hook enum 输入和 void 方法状态副作用断言；`CompositedMessageInterceptor.java` top2 暴露的是组合 hook 的非空列表、匿名接口实现、attribute map 前置状态和 Checkstyle 行长约束；`ClientManagerImpl.java` top5 暴露的是 Java private/internal 方法和线程池 await 内部分支不能从外部测试直接覆盖；`ClientSessionImpl.java` top7 暴露的是 public 方法也可能因复杂构造依赖、RPC session、request observer 和 stream lifecycle 状态不可安全静态生成；`ClientImpl.java` top8 进一步确认抽象 client lifecycle、session table 和 heartbeat 路径应归为公共入口/手审任务；consumer `Assignment` / `Assignments` 补上轻量 value object equals 分支的真实 ready 路径；`ConsumeTask` / `ConsumeService` 则补上 source import 复制、注释误 import 过滤、`TestBase` fixture、listener/interceptor mock、匿名抽象类子类和 executor/scheduler 生成；`ConsumerImpl.wrapFilterExpression` 和两个 `wrapReceiveMessageRequest` overload 已通过公共 request builder 路径从手审推进到真实 ready，`ConsumerImpl.changeInvisibleDuration`、`ConsumerImpl.ackMessage` 和 `ConsumerImpl.receiveMessage` 已通过 `PushConsumerImpl` spy + mock `ClientManager` 的 RPC fixture 从手审推进到真实 ready。当前这些问题都已转成可执行 JUnit 断言或明确的 `manual_review_internal`；`ConsumerImpl.java` top20 当前为 `passed=20/ready=17/manual_review_internal=3`，普通 `repair_generated_test` 清零。下一步建议离开 RocketMQ 专用 consumer fixture，选择一个非 RocketMQ 的 Java/JUnit 项目验证构造器、Optional、集合、异常路径和外部依赖分类的泛化能力。

## 第一百七十四阶段：非 RocketMQ Java 真实样本验证

1. [x] 选择 JSON-java 作为非 RocketMQ Java/JUnit 样本；该项目是单模块 Maven + JUnit + JaCoCo，baseline 覆盖率命令 `mvn org.jacoco:jacoco-maven-plugin:prepare-agent test org.jacoco:jacoco-maven-plugin:report` 通过，测试基线为 `Tests run: 787, Failures: 0, Errors: 0, Skipped: 6`。
2. [x] 用 list-only 模式查看 JSON-java top40 任务分布：`tasks.selected count=259 limit=40`，热点集中在 `JSONArray.java`、`JSONML.java`、`JSONObject.java`、`XML.java`、`XMLTokener.java` 和 `Cookie.java`。
3. [x] 选择 `org/json/JSONArray.java` top10 初跑：结果为 `status_counts={"failed":6,"passed":4}`，`action_counts={"manual_review":5,"manual_review_internal":4,"repair_generated_test":1}`，失败集中在 `new JSONArray(null)` 重载歧义、空数组调用 `getNumber/getFloat/optNumber`、以及 `write(null, ...)` 触发 NPE。
4. [x] 补强 Java 构造器 null 输入：对集合/泛型构造器参数在 coverage task 要求 null 时生成显式 typed null，例如 `new JSONArray((Iterable<?>) null)`，避免多重引用类型构造器的 `null` 歧义。
5. [x] 补强 `JSONArray` coverage task 生成：对 `getNumber` 的 `object instanceof Number` 分支先 `put(1)` 再断言返回；对 `getNumber/getFloat` 错误分支使用非数字对象触发 `JSONException`；对 `optNumber` invalid string 路径断言默认值；对 `write` 的 IOException 分支生成抛错 `java.io.Writer`。
6. [x] 为上述 Java 泛化点增加回归测试：覆盖重载集合构造器 typed null、`JSONArray` 数值读取状态、optional number fallback、float conversion error 和 writer IOException wrapper。
7. [x] 复跑 JSON-java `JSONArray.java` top10：脚本通过，`status_counts={"passed":10}`，`action_counts={"ready":6,"manual_review_internal":4}`，`zero_skip=0`；普通 `repair_generated_test` 清零。
8. [x] 更新 `docs/real-project-validation.md` 和 `docs/quality-assessment.md`，把 JSON-java 作为第十五个真实项目样本记录，明确 Java 已开始离开 RocketMQ 专用 fixture，但仍需要更多非 RocketMQ 样本验证。
9. [x] 扩大到 JSON-java `JSONObject.java` top10：脚本通过，`status_counts={"passed":10}`，`action_counts={"manual_review_internal":9,"ready":1}`；反射/record helper 稳定归为内部手审，普通 repair 清零。
10. [x] 扩大到 JSON-java `JSONML.java` top10：脚本通过，`status_counts={"passed":10}`，`action_counts={"manual_review_internal":10}`；递归 parse helper 稳定归为内部手审，普通 repair 清零，但 public ready 密度不足。
11. [x] 切入 JSON-java `XML.java` top10：初跑为 `status_counts={"failed":3,"passed":7}`，`action_counts={"manual_review":1,"manual_review_internal":7,"repair_generated_test":2}`；失败集中在 `XML.toJSONObject(Reader, keepNumberAsString, keepBooleanAsString)` 使用 `null` XML 输入，以及 `XML.noSpace` 错误路径使用有效字符串。
12. [x] 补强 Java `XML` coverage task 生成：`toJSONObject` 的 keep-number / keep-boolean 分支生成 `StringReader("<root>42</root>")` / `StringReader("<root>true</root>")` 并断言保留字符串；`noSpace` 错误路径用空字符串或含空格字符串触发 `JSONException`。
13. [x] 为 `XML.toJSONObject` flags 和 `XML.noSpace` 错误路径增加 Java 生成器回归测试，避免重新退回 `null` XML 输入或有效字符串。
14. [x] 复跑 JSON-java `XML.java` top10：脚本通过，`status_counts={"passed":10}`，`action_counts={"manual_review_internal":7,"ready":3}`，`zero_skip=0`；普通 repair 清零。
15. [x] 更新 `docs/real-project-validation.md` 和 `docs/quality-assessment.md`，把 JSON-java 验证窗口扩展为 `JSONArray + JSONObject + JSONML + XML` 四个文件。

已完成补充：第十五个真实样本把 Java 验证从 RocketMQ 专用 fixture 推进到轻量通用库。JSON-java 暴露的是更普通的 Java API 问题：重载构造器 `null` 歧义、集合/索引 getter 需要最小对象状态、optional fallback 断言、writer I/O error path、XML parser flag 输入、tag name 错误路径和 package/private helper 手审分类。当前 `JSONArray.java` top10 已从 `passed=4/failed=6` 收敛到 `passed=10/ready=6/manual_review_internal=4`；`JSONObject.java` 和 `JSONML.java` 分别通过但主要是内部手审；`XML.java` top10 已从 `passed=7/failed=3` 收敛到 `passed=10/ready=3/manual_review_internal=7`。普通 repair 继续清零。下一步建议选择第二个非 RocketMQ Java/JUnit 项目，优先覆盖 Optional、Map/Collection、泛型 value object、I/O error path 和外部依赖分类；若继续在 JSON-java 内推进，可优先跑 `Cookie.java` 或 `XMLTokener.java` top10，寻找更多 public ready 任务。

## 第一百七十五阶段：第二个非 RocketMQ Java 样本验证

1. [x] 选择 Apache Commons Codec 作为第二个非 RocketMQ Java/JUnit 样本；该项目是成熟 Apache Maven 单模块库，使用 JUnit 5、JaCoCo、Apache RAT 和较大的既有测试套件，适合验证生成测试是否会破坏已有测试文件和 helper。
2. [x] 建立真实 baseline：复制到 `/tmp/testloop-commons-codec` 后运行 `mvn -q org.jacoco:jacoco-maven-plugin:prepare-agent test org.jacoco:jacoco-maven-plugin:report` 通过，生成 `target/site/jacoco/jacoco.xml`，Surefire 报告无失败。
3. [x] 用 list-only 查看 top60 任务分布：`tasks.selected count=172 limit=60`，热点集中在 `DigestUtils.java`、`Digest.java`、`Base64.java`、`DaitchMokotoffSoundex.java`、`Rule.java`、`BaseNCodec.java`、`Blake3.java` 和 `HmacUtils.java`。
4. [x] 选择 `org/apache/commons/codec/binary/Base64.java` top5 初跑：结果为 `status_counts={"failed":5}`，其中前三个失败的根因是生成器覆盖了项目既有 `Base64Test.java`，导致 upstream `Base64InputStreamTest` / `Base64OutputStreamTest` 找不到 `Base64Test.BASE64_IMPOSSIBLE_CASES`；后两个失败是嵌套类目标 `Base64.Builder.setDecodeTableFormat` 被生成成未限定 `Builder`。
5. [x] 修复 Java coverage task 测试文件碰撞：当 Java task 推荐测试文件已存在时，`generate_tests` 改写到 `*TestLoopTest.java`，并把调整后的 `test_file` 同步回 output、context 和 coverage task，避免覆盖上游测试 helper。
6. [x] 修复 Java 生成类名：Java 生成器优先使用最终 `coverage_task.test_file` 推导测试类名，例如 `Base64TestLoopTest.java` 生成 `public class Base64TestLoopTest`，不再继续用源类名派生 `Base64Test`。
7. [x] 修复 Java 嵌套类 coverage task：当目标形如 `Base64.Builder.setDecodeTableFormat` 时，生成器使用 `Base64.Builder` 作为 receiver，并在返回类型等于嵌套类短名时限定为 `Base64.Builder`。
8. [x] 补强 line-range 输入推断：对 `Base64.Builder.setDecodeTableFormat` 按未覆盖行选择 `Base64.DecodeTableFormat.STANDARD` / `URL_SAFE` / `MIXED` 或 `null`，避免所有分支都退化为 `null`。
9. [x] 为上述能力补回归测试：覆盖既有 Java test file 不被覆盖、coverage task test file/class name 保持一致、嵌套类 target 使用限定 receiver/返回类型和 line-specific enum 输入。
10. [x] 复跑 Commons Codec `Base64.java` top5：脚本通过，`status_counts={"passed":5}`，`action_counts={"manual_review_internal":3,"ready":2}`，`zero_skip=0`，普通 `repair_generated_test` 清零；两个 `Base64.Builder.setDecodeTableFormat` 分支均为真实 `ready`。
11. [x] 更新 `docs/real-project-validation.md` 和 `docs/quality-assessment.md`，把 Apache Commons Codec 作为第十六个真实项目样本记录。
12. [x] 扩大到 Commons Codec `DigestUtils.java` top10：初跑为 `status_counts={"failed":10}`，`action_counts={"manual_review":3,"repair_generated_test":7}`，失败集中在 SHAKE digest 方法族。
13. [x] 修复 Java SHAKE 方法族生成：对 `shake128_256`、`shake128_256Hex`、`shake256_512`、`shake256_512Hex` 按重载类型生成 typed input，避免 `null` 在 `byte[]` / `InputStream` / `String` 间编译歧义；对运行时算法不可用的 JDK，断言 `IllegalArgumentException` 信息包含 `SHAKE`；算法可用时仍断言返回非空。
14. [x] 为 SHAKE 生成补回归测试：覆盖 byte[] 和 InputStream 重载，断言不会生成 `shake*(null)` 或空的 `assertThrows(IOException.class)`。
15. [x] 复跑 Commons Codec `DigestUtils.java` top10：脚本通过，`status_counts={"passed":10}`，`action_counts={"ready":10}`，普通 `repair_generated_test` 清零。
16. [x] 更新 `docs/real-project-validation.md` 和 `docs/quality-assessment.md`，把 Commons Codec 验证窗口扩展为 `Base64.java` top5 + `DigestUtils.java` top10。
17. [x] 扩大到 Commons Codec `Rule.java` top10：初跑为 `status_counts={"failed":4,"passed":6}`，`action_counts={"manual_review":2,"manual_review_internal":6,"repair_generated_test":2}`；内部 parser helper 已稳定归类，普通失败集中在 `Rule.Phoneme` 无参构造假设和 `Rule.getInstance` 空 enum/language 输入。
18. [x] 修复 Java Rule 生成：`Rule.Phoneme.join/toString` 使用真实 `Rule.Phoneme("...", Languages.ANY_LANGUAGE)` 构造并断言 phoneme text / 字符串前缀；`Rule.getInstance` 使用 `NameType.GENERIC`、`RuleType.RULES`、`"english"` 或 `Languages.LanguageSet.from(...)`，避免 `null` 导致 NPE。
19. [x] 为 Rule 生成补回归测试：覆盖 nested `Rule.Phoneme.join` 不再使用无参构造或 `join(null)`，以及 `Rule.getInstance` 不再使用空 enum 输入。
20. [x] 复跑 Commons Codec `Rule.java` top10：脚本通过，`status_counts={"passed":10}`，`action_counts={"manual_review_internal":6,"ready":4}`，普通 `repair_generated_test` 清零。
21. [x] 更新 `docs/real-project-validation.md` 和 `docs/quality-assessment.md`，把 Commons Codec 验证窗口扩展为 `Base64.java` top5 + `DigestUtils.java` top10 + `Rule.java` top10。
22. [x] 尝试扩大到 Commons Codec `BaseNCodec.java` top10：list-only 阶段确认过滤后只有 7 个候选任务，因此按 top7 实测，避免把不存在的窗口写成失败。
23. [x] 复跑 Commons Codec `BaseNCodec.java` top7：脚本通过，`status_counts={"passed":7}`，`action_counts={"manual_review_internal":7}`，普通 `repair_generated_test` 清零；目标覆盖 `BaseNCodec.gte0`、`getLength`、`isInAlphabet`、`isWhiteSpace` 和 `BaseNCodec.AbstractBuilder` 的三个 getter。
24. [x] 更新 `docs/real-project-validation.md` 和 `docs/quality-assessment.md`，把 Commons Codec 验证窗口扩展为 `Base64.java` top5 + `DigestUtils.java` top10 + `Rule.java` top10 + `BaseNCodec.java` top7。
25. [x] 尝试扩大到 Commons Codec `HmacUtils.java` top10：list-only 阶段确认过滤后只有 7 个候选任务，因此按 top7 实测。
26. [x] 初跑 Commons Codec `HmacUtils.java` top7：结果为 `status_counts={"failed":6,"passed":1}`，`action_counts={"manual_review":3,"ready":1,"repair_generated_test":3}`；普通失败集中在无效算法名、默认构造器空 Mac、`null` 重载歧义和 ByteBuffer 输入缺失。
27. [x] 修复 Java HmacUtils 生成：`isAvailable` 使用 `HmacAlgorithms.HMAC_SHA_256` 或其 `getName()`；HmacUtils 构造器使用真实算法和稳定 key；`hmac` / `hmacHex` 使用有效实例和 typed `java.nio.ByteBuffer.wrap(...)` 输入。
28. [x] 为 HmacUtils 生成补回归测试：覆盖 `isAvailable` 不再使用 `"test"`，`String,String` 构造器不再使用无效算法名，`hmac` / `hmacHex` 不再使用默认实例和 `null` 参数。
29. [x] 复跑 Commons Codec `HmacUtils.java` top7：脚本通过，`status_counts={"passed":7}`，`action_counts={"ready":7}`，普通 `repair_generated_test` 清零。
30. [x] 更新 `docs/real-project-validation.md` 和 `docs/quality-assessment.md`，把 Commons Codec 验证窗口扩展为 `Base64.java` top5 + `DigestUtils.java` top10 + `Rule.java` top10 + `BaseNCodec.java` top7 + `HmacUtils.java` top7。
31. [x] 扩大到 Commons Codec `Digest.java` top10：list-only 显示过滤后有 20 个候选任务，top10 全部集中在 CLI `Digest.run`。
32. [x] 复跑 Commons Codec `Digest.java` top10：脚本通过，`status_counts={"passed":10}`，`action_counts={"manual_review_internal":10}`，普通 `repair_generated_test` 清零；目标覆盖 `Digest.run` 的 `93`、`105-114`、`143` 等行段。
33. [x] 更新 `docs/real-project-validation.md` 和 `docs/quality-assessment.md`，把 Commons Codec 验证窗口扩展为 `Base64.java` top5 + `DigestUtils.java` top10 + `Rule.java` top10 + `BaseNCodec.java` top7 + `HmacUtils.java` top7 + `Digest.java` top10。
34. [x] 尝试扩大到 Commons Codec `Blake3.java` top10：list-only 阶段确认过滤后只有 4 个候选任务，因此按 top4 实测。
35. [x] 复跑 Commons Codec `Blake3.java` top4：脚本通过，`status_counts={"passed":4}`，`action_counts={"manual_review_internal":4}`，普通 `repair_generated_test` 清零；目标覆盖 `Blake3.checkBufferArgs` 的三个 buffer 参数检查分支和 `Blake3.doFinalize` 内部分支。
36. [x] 更新 `docs/real-project-validation.md` 和 `docs/quality-assessment.md`，把 Commons Codec 验证窗口扩展为 `Base64.java` top5 + `DigestUtils.java` top10 + `Rule.java` top10 + `BaseNCodec.java` top7 + `HmacUtils.java` top7 + `Digest.java` top10 + `Blake3.java` top4。
37. [x] 扩大到 Commons Codec `org/apache/commons/codec/digest/` package top20：初跑为 `status_counts={"failed":1,"passed":19}`，`action_counts={"manual_review_internal":5,"ready":14,"repair_generated_test":1}`；唯一普通失败是 `DigestUtils.sha` 生成裸 `null`，在 `sha(InputStream)` 与 `sha(String)` 间产生编译歧义。
38. [x] 修复 Java `DigestUtils.sha` 生成：对 `byte[]`、`InputStream`、`String` 重载生成 typed input，并断言返回 byte array 非空，避免 `DigestUtils.sha(null)`。
39. [x] 复跑 Commons Codec `DigestUtils.java` top17：中间暴露 `getShake128_256Digest` 和 `getShake256_512Digest` 在当前 JDK 无 SHAKE MessageDigest 时不能直接断言非空；修复后两个 getter 使用兼容性 try/catch，算法可用时断言非空，不可用时断言异常信息包含 `SHAKE`。
40. [x] 为 `DigestUtils.sha` 和 `getShake*Digest` 生成补回归测试，固定 typed input、SHAKE 兼容异常断言和 unexpected exception fail 行为。
41. [x] 复跑 Commons Codec digest package top20：脚本通过，`status_counts={"passed":20}`，`action_counts={"manual_review_internal":5,"ready":15}`，普通 `repair_generated_test` 清零；同步更新 `docs/real-project-validation.md` 和 `docs/quality-assessment.md`。
42. [x] 用 list-only 查看 Commons Codec `org/apache/commons/codec/binary/` 包任务分布：过滤后只有 24 个候选任务，不存在 top40 窗口；目标覆盖 Base32/Base64 codec 内部方法、Base58、Base64 Builder、BaseNCodec helper、CharSequenceUtils、Base16 stream 构造器和 BaseNCodecInputStream。
43. [x] 复跑 Commons Codec `binary/` package top24：脚本通过，`status_counts={"passed":24}`，`action_counts={"manual_review_internal":19,"ready":5}`，普通 `repair_generated_test` 清零；ready 集中在两个 `Base64.Builder.setDecodeTableFormat` 分支和三个 Base16 stream 构造器分支。
44. [x] 更新 `docs/real-project-validation.md` 和 `docs/quality-assessment.md`，把 Commons Codec 验证窗口扩展为 digest package top20 + binary package top24。
45. [x] 用 list-only 查看 Commons Codec `org/apache/commons/codec/language/bm/` 包任务分布：过滤后只有 34 个候选任务，不存在 top40 窗口；目标覆盖 `Lang`、`PhoneticEngine`、`Languages.SomeLanguages`、`Rule` parser/value object、`ResourceConstants.java` 和 `Rule.java` 文件级任务。
46. [x] 初跑 Commons Codec `language/bm/` package top34：结果为 `status_counts={"failed":11,"passed":23}`，`action_counts={"manual_review":5,"manual_review_internal":17,"ready":6,"repair_generated_test":6}`；失败集中在资源规则/encode/lang 手审边界、`SomeLanguages` 私有构造器、`Rule.RPattern` 未限定、文件级任务误生成 whole-file 测试和 `PhoneticEngine` 非法 `RuleType.RULES` 构造。
47. [x] 修复 Java `language/bm` value object 与资源边界生成：`Languages.SomeLanguages` 通过 `Languages.LanguageSet.from(...)` factory 构造并限定 `Languages.LanguageSet`；`Lang.loadFromResource` 和 `PhoneticEngine.encode` 稳定归为 `manual_review_internal`；`PhoneticEngine.getLang` 使用合法 `RuleType.APPROX`。
48. [x] 修复 Java `Rule` 嵌套返回类型和文件级任务：`Rule.RPattern`、`Rule.Phoneme`、`Rule.PhonemeExpr`、`Rule.PhonemeList` 会按需限定为 `Rule.*`；文件级 Java coverage task 生成可运行的 `manual_review_internal` 手审 smoke，不再伪造整文件测试。
49. [x] 为 `language/bm` 生成补回归测试：覆盖 `SomeLanguages.merge/restrictTo/getLanguages` factory 构造、`PhoneticEngine.encode/getLang` 边界、`Lang.loadFromResource` 资源手审、`Rule.RPattern` 限定返回类型和文件级任务手审。
50. [x] 复跑 Commons Codec `language/bm/` package top34：脚本通过，`status_counts={"passed":34}`，`action_counts={"manual_review_internal":23,"ready":11}`，普通 `repair_generated_test` 清零；同步更新 `docs/real-project-validation.md`、`docs/quality-assessment.md` 和 `CHANGELOG.md`。
51. [x] 扩大到 Commons Codec `Caverphone.java` top7：初跑为 `status_counts={"failed":3,"passed":4}`，`action_counts={"manual_review":3,"ready":4}`；失败全部集中在 `Caverphone.encode(Object)`，生成器使用 `instance.encode(null)` 并追加空 lambda `assertThrows`，没有真正触发非 String 参数错误路径。
52. [x] 修复 Java `StringEncoder.encode(Object)` 生成：对 `throws EncoderException` 的 `encode(Object)`，错误路径生成 `instance.encode(new Object())` 的真实异常断言，返回路径生成 `instance.encode("test")` 的非空结果断言，不再追加空的 `assertThrows` TODO。
53. [x] 为 `StringEncoder.encode(Object)` 增加回归测试，覆盖 Caverphone 类似适配方法的异常路径和成功返回路径。
54. [x] 复跑 Commons Codec `Caverphone.java` top10：脚本通过，`status_counts={"passed":10}`，`action_counts={"manual_review_internal":2,"ready":8}`，普通 `repair_generated_test` 清零；两个手审项均为文件级 task。
55. [x] 扩大到 Commons Codec `DaitchMokotoffSoundex.java` top9：初跑为 `status_counts={"failed":3,"passed":6}`，`action_counts={"manual_review_internal":6,"repair_generated_test":3}`；失败全部来自直接构造 private nested `DaitchMokotoffSoundex.Branch` / `Rule`，测试编译阶段被 Java 访问控制拒绝。
56. [x] 修复 Java private nested class coverage task 分类：当 task 目标命中源码中的 `private ... class/interface/enum` 嵌套类型时，生成可运行的 `manual_review_internal` 手审 smoke，不再伪造外部直接构造。
57. [x] 为 private nested Java type 增加回归测试，覆盖 `DaitchMokotoffSoundex.Branch.equals` 这类 public method 但所属类型 private 的任务。
58. [x] 复跑 Commons Codec `DaitchMokotoffSoundex.java` top9：脚本通过，`status_counts={"passed":9}`，`action_counts={"manual_review_internal":9}`，普通 `repair_generated_test` 清零。
59. [x] 复跑 Commons Codec `DoubleMetaphone.java` top6：脚本通过，`status_counts={"passed":6}`，`action_counts={"manual_review_internal":6}`；目标集中在 private `handleCH`、`handleG`、`handleJ`、`handleS`，确认复杂字符串规则 handler 会稳定归为手审。
60. [x] 初跑 Commons Codec `MatchRatingApproachEncoder.java` top1：脚本通过但结果为 `status_counts={"passed":1}`、`action_counts={"ready":1}`、`skipped_ready=["1 junit-70 MatchRatingApproachEncoder.encode 145-145"]`；复核源码确认生成的 `instance.encode("test")` 没有命中 145 行目标缺口。
61. [x] 修复 Java `MatchRatingApproachEncoder.encode` 145 行不可达路径分类：该行在 `removeVowels` 后检查空值，但 public `encode(String)` 能变空的输入已在前置 cleanName 空值保护返回，`removeVowels` 又会保留首字母元音或非元音，因此静态生成器输出 `manual_review_unreachable`，不再暴露弱 ready。
62. [x] `validate_coverage_task` 识别生成预览中的 `manual_review_unreachable:` 标记，并把 action 改为 `manual_review_unreachable`，让 Agent 能区分“测试通过但不可达手审”和真正 ready。
63. [x] 为 `MatchRatingApproachEncoder.encode` 不可达分类和 `manual_review_unreachable` 标记识别增加回归测试。
64. [x] 初跑 Commons Codec `Metaphone.java` top1：脚本通过但生成输入为 `instance.metaphone("test")`，不能命中目标 279 行 silent `G` 分支；修复后改用 `instance.metaphone("agned")` 覆盖内部 `GN/GNED` 分支，复跑达到 `status_counts={"passed":1}`、`action_counts={"ready":1}`。
65. [x] 初跑 Commons Codec `Soundex.java` top2：结果为 `status_counts={"failed":1,"passed":1}`，失败来自 `getMaxLength` 通用 int 断言 `0`，真实默认值为 `4`；`setMaxLength` 虽通过但没有断言状态变化。
66. [x] 修复 Soundex getter/setter 生成：`getMaxLength` 断言默认 `4`，`setMaxLength(6)` 后断言 `getMaxLength()` 返回 `6`，复跑达到 `status_counts={"passed":2}`、`action_counts={"ready":2}`，普通 `repair_generated_test` 清零。
67. [x] 为 Commons Codec `Metaphone` silent `G` 输入和 `Soundex` getter/setter 状态断言增加回归测试。
68. [x] 检查 Commons Codec `ColognePhonetic.java` / `Nysiis.java` / `RefinedSoundex.java` 剩余候选：`ColognePhonetic` 只有 2 个，`Nysiis` / `RefinedSoundex` 当前为 0。
69. [x] 复跑 Commons Codec `ColognePhonetic.java` top2：`status_counts={"passed":2}`、`action_counts={"manual_review_internal":2}`，目标集中在内部 `CologneInputBuffer.copyData`。

已完成补充：第十六个真实样本证明 Java 生成器不能只考虑“推荐测试文件路径”，还必须保护成熟项目已有的测试文件和 helper。Commons Codec 初跑暴露的关键问题不是断言质量，而是生成测试覆盖 `Base64Test.java` 后破坏上游测试类契约；现在生成器会选择 `Base64TestLoopTest.java` 这类不碰撞文件，并让 coverage task、上下文和类名保持一致。该样本也补上了嵌套类目标的通用能力：`Base64.Builder.setDecodeTableFormat` 能生成 `Base64.Builder` receiver、限定返回类型和按 line range 选择的 `Base64.DecodeTableFormat` 输入。`DigestUtils.java` top10 进一步暴露了成熟库里常见的重载 API 与运行时能力差异：同名 digest 方法不能用裸 `null` 输入硬调，JDK 不支持 SHAKE 算法时也不能断言非空返回。digest package top20 继续把这个问题从单文件扩展到包级窗口：`DigestUtils.sha` 需要按重载生成 typed input，`getShake128_256Digest` / `getShake256_512Digest` 需要兼容运行时缺少 SHAKE MessageDigest 的 JDK。`Rule.java` top10 又补上 nested value object 和资源规则入口场景：`Rule.Phoneme` 需要真实构造器，`Rule.getInstance` 需要真实 enum/language 输入，内部 parser helper 则应稳定归为手审。`BaseNCodec.java` top7 没有新增 ready 测试，但确认抽象 codec 内部状态机和 nested abstract builder getter 会被稳定归为 `manual_review_internal`，并且生成的手审 skip 能通过 Apache RAT、JUnit 5 和既有大测试套件。`HmacUtils.java` top7 把 crypto helper 的有效算法、key/material 和 ByteBuffer 重载输入补成通用规则，窗口从 `passed=1/failed=6` 收敛到 `passed=7/ready=7`。`Digest.java` top10 则确认 CLI `run` 的内部循环、输出和文件/目录处理路径会稳定归为手审 skip，而不是伪造脆弱 ready 测试。`Blake3.java` top4 继续确认二进制状态对象内部 helper 会稳定归为手审 skip。`binary/` package top24 则把验证扩展到 Base32/Base64 streaming codec、Base58、BaseNCodec helper、CharSequenceUtils、Base16 stream 和 BaseNCodecInputStream。`language/bm/` package top34 进一步验证资源规则、Beider-Morse encode、多语言推断、`SomeLanguages` 私有构造 value object、`Rule` nested return type、内部 parser helper 和文件级 task：窗口从 `passed=23/failed=11` 收敛到 `passed=34/ready=11/manual_review_internal=23`。`Caverphone.java` top10 补上非 bm 语言 encoder 的 `StringEncoder.encode(Object)` 适配路径：错误路径用非 String 对象触发 `EncoderException`，返回路径用 String 输入触发委托 encode，窗口从 top7 `passed=4/failed=3` 收敛到 top10 `passed=10/ready=8/manual_review_internal=2`。`DaitchMokotoffSoundex.java` top9 确认复杂规则 parser 与 private nested branch state 会稳定归为手审：窗口从 `passed=6/failed=3` 收敛到 `passed=9/manual_review_internal=9`。`DoubleMetaphone.java` top6 确认 private handler 状态机稳定归为 `manual_review_internal`。`MatchRatingApproachEncoder.java` top1 进一步暴露并修正弱 ready：生成测试通过不等于命中 coverage 缺口，145 行 public 路径不可达时应输出 `manual_review_unreachable`。`Metaphone.java` top1 和 `Soundex.java` top2 又把问题收敛到 public encoder 的输入/断言质量：silent `G` 需要 `agned` 这类 line-specific 输入，deprecated getter/setter 需要断言真实默认值和状态变化。当前 `Base64.java` top5 已达到 `passed=5/ready=2/manual_review_internal=3`，`DigestUtils.java` top17 已达到 `passed=17/ready=17`，digest package top20 已达到 `passed=20/ready=15/manual_review_internal=5`，binary package top24 已达到 `passed=24/ready=5/manual_review_internal=19`，language/bm package top34 已达到 `passed=34/ready=11/manual_review_internal=23`，Caverphone top10 已达到 `passed=10/ready=8/manual_review_internal=2`，DaitchMokotoffSoundex top9 已达到 `passed=9/manual_review_internal=9`，DoubleMetaphone top6 已达到 `passed=6/manual_review_internal=6`，MatchRating top1 已达到 `passed=1/manual_review_unreachable=1`，Metaphone top1 已达到 `passed=1/ready=1`，Soundex top2 已达到 `passed=2/ready=2`，`Rule.java` top10 已达到 `passed=10/ready=4/manual_review_internal=6`，`BaseNCodec.java` top7 已达到 `passed=7/manual_review_internal=7`，`HmacUtils.java` top7 已达到 `passed=7/ready=7`，`Digest.java` top10 已达到 `passed=10/manual_review_internal=10`，`Blake3.java` top4 已达到 `passed=4/manual_review_internal=4`，普通 repair 清零。下一步建议继续验证 Commons Codec 剩余 `language/` 非 bm public encoder，或切到另一个 Java 库验证 Optional、Map/Collection、泛型 value object、I/O error path 和外部依赖分类。

补充收口：Commons Codec `language/` 非 bm 剩余高优先级候选已基本见底，`ColognePhonetic.java` 只有内部 buffer top2 且已稳定归为 `manual_review_internal`，`Nysiis.java` / `RefinedSoundex.java` 当前无候选。下一步应离开该局部窗口，优先选择另一个 Java/JUnit 库或 Commons Codec 其他包级窗口，继续验证 Optional、Map/Collection、泛型 value object、I/O error path 和外部依赖分类。

## 第一百七十六阶段：第三个非 RocketMQ Java 样本验证

1. [x] 选择 Apache Commons Lang 作为第三个非 RocketMQ Java/JUnit 样本；该项目是成熟 Apache Maven 单模块库，当前快照为 `9f57c08`，覆盖泛型反射、字符串、时间、数学、数组和 class-name helper。
2. [x] 建立真实 baseline：复制到 `/tmp/testloop-commons-lang` 后运行 `mvn -q org.jacoco:jacoco-maven-plugin:prepare-agent test org.jacoco:jacoco-maven-plugin:report` 通过，生成 `target/site/jacoco/jacoco.xml`；该项目每轮真实测试约 65k 个用例、17 个 upstream skip。
3. [x] 生成 Commons Lang top80 list-only 清单：共有 539 个 coverage task，top80 热点集中在 `TypeUtils.java`、`CharSequenceUtils.java`、`FastDatePrinter.java`、`Fraction.java`、`ClassUtils.java`，另有 `AppendableJoiner.java`、`ArrayUtils.java`、`Failable.java` 等适合后续小窗口验证。
4. [x] 复跑 Commons Lang `TypeUtils.java` top5：`status_counts={"passed":5}`、`action_counts={"manual_review_internal":5}`，目标覆盖 `isAssignable`、`unrollBounds` 和 `unrollVariables` 的泛型反射内部边界；普通 `repair_generated_test` 清零。
5. [x] 复跑 Commons Lang `AppendableJoiner.java` top4：`status_counts={"passed":4}`、`action_counts={"manual_review_internal":4}`，目标覆盖 `joinI` / `joinSB` 的 `IOException` error path；普通 `repair_generated_test` 继续清零。
6. [x] 初跑 Commons Lang `ArrayUtils.java` top2：`status_counts={"failed":1,"passed":1}`，`addAll(T[], T...)` 失败来自生成器输出不可编译的 `T[] result` 和重载歧义的 `addAll(null, null)`；`addExact` 已稳定归为 `manual_review_internal`。
7. [x] 修复 Java 裸类型变量数组/varargs coverage task：遇到 `T[]` / `T...` 这类静态生成器无法安全实例化的签名时生成 `manual_review_internal` 手审 smoke，不再伪造泛型数组或裸 `null` 调用；新增回归测试固定 `ArrayUtils.addAll` 场景。
8. [x] 复跑 Commons Lang `ArrayUtils.java` top2：`status_counts={"passed":2}`、`action_counts={"manual_review_internal":2}`，普通 `repair_generated_test` 清零。
9. [x] 初跑 Commons Lang `ClassUtils.java` top6：`status_counts={"passed":6}`、`action_counts={"manual_review_internal":2,"ready":4}`，但 `getShortClassName("test")` 和 `hierarchy(null, null)` 属于通过却不命中目标行的弱 ready；`toCleanName` 私有路径已稳定归为 `manual_review_internal`。
10. [x] 修复 Java `ClassUtils` coverage task 输入：`getShortClassName` 按 `line_range` 使用 `"[Ljava.lang.String;"` 和 `"[I"` 覆盖对象数组 / primitive array 分支；`hierarchy` 通过 `iterator.remove()` 触发 `UnsupportedOperationException` 路径，不再使用空参数调用。
11. [x] 复跑 Commons Lang `ClassUtils.java` top6：`status_counts={"passed":6}`、`action_counts={"manual_review_internal":2,"ready":4}`，普通 repair 清零；生成测试已命中目标分支，复跑摘要里的 skipped 来自 Commons Lang 上游 skipped tests，不是生成测试自身跳过。
12. [x] 初跑 Commons Lang `CharSequenceUtils.java` top8：`status_counts={"passed":8}`、`action_counts={"manual_review_internal":7,"ready":1}`；7 个 `regionMatches` surrogate / ignoreCase 分支稳定归为包内 helper 手审，但 `toCharArray` ready 使用 `CharSequenceUtils.toCharArray("test")`，会走 `String` 分支而不是 419 行 `StringBuffer` 分支。
13. [x] 修复 Java `CharSequenceUtils.toCharArray` coverage task 输入：line 419 使用 `new StringBuffer("test")` 并断言 `char[] {'t','e','s','t'}`，避免通过但不命中目标缺口的弱 ready；新增回归测试固定该场景。
14. [x] 复跑 Commons Lang `CharSequenceUtils.java` top8：`status_counts={"passed":8}`、`action_counts={"manual_review_internal":7,"ready":1}`，普通 repair 清零；`CharSequenceUtilsTestLoopTest` 自身 `Skipped: 0`，复跑摘要中的 skipped 来自 Commons Lang 上游 skipped tests。
15. [x] 初跑 Commons Lang `StopWatch.java` top3：`status_counts={"failed":2,"passed":1}`、`action_counts={"manual_review":2,"ready":1}`；`split(String)` 和 `getStopInstant` 对未启动实例普通调用导致 `IllegalStateException` 未被断言，`getNanoTime` line 407 用未启动状态断言 0，属于不命中 switch default 防御分支的弱 ready。
16. [x] 修复 Java `StopWatch` coverage task 输入：`split(String)` 和 `getStopInstant` 生成 `assertThrows(IllegalStateException.class, ...)`；`getNanoTime` line 407 归为 `manual_review_unreachable`，说明该 default 分支无法通过公共状态流转触达。
17. [x] 复跑 Commons Lang `StopWatch.java` top3：`status_counts={"passed":3}`、`action_counts={"manual_review_unreachable":1,"ready":2}`，普通 repair 清零；两个 ready 测试自身没有 skip，复跑摘要中的 skipped 来自 Commons Lang 上游 skipped tests。
18. [x] 生成 Commons Lang `StopWatch.java` top4 list-only 清单：过滤后实际只有 4 个候选，新增第 4 个为 `StopWatch.Split.toString` line 118。
19. [x] 初跑 Commons Lang `StopWatch.java` top4：`status_counts={"passed":4}`、`action_counts={"manual_review_internal":1,"manual_review_unreachable":1,"ready":2}`；新增 `Split.toString` 被误判为 private/internal，根因是 `private enum SplitState` 的 `enum Split` 前缀匹配污染了 public nested `Split`。
20. [x] 修复 Java public nested class 误判和构造器查找：nested type 私有判断增加 Java 标识符边界，限定名 `StopWatch.Split` 可回退到简单名 `Split` 查找 public 构造器，`Duration` 默认值使用 `java.time.Duration.ZERO`；`StopWatch.Split.toString` 生成精确字符串断言。
21. [x] 复跑 Commons Lang `StopWatch.java` top4：`status_counts={"passed":4}`、`action_counts={"manual_review_unreachable":1,"ready":3}`，普通 repair 清零；`Split.toString` 从手审推进为真实 ready。
22. [x] 生成 Commons Lang `ExceptionUtils.java` top10 list-only 清单：前 5 个候选集中在 `ExceptionUtils.throwUnchecked` 的 RuntimeException、Error、throw 和 unchecked wrapper 行段。
23. [x] 初跑 Commons Lang `ExceptionUtils.java` top5：`status_counts={"failed":5}`、`action_counts={"repair_generated_test":5}`；失败根因是通用 Java 生成器把泛型返回类型 `T` 直接写入测试代码，导致 Maven testCompile 找不到符号 `T`。
24. [x] 修复 Java `ExceptionUtils.throwUnchecked` coverage task：按 `line_range` 生成 `RuntimeException`、`Error`、checked-return 和 unchecked wrapper 的专用断言，不再生成 `T result = ExceptionUtils.throwUnchecked(null)`。
25. [x] 复跑 Commons Lang `ExceptionUtils.java` top5：`status_counts={"passed":5}`、`action_counts={"ready":5}`，普通 repair 清零；复跑摘要中的 skipped 来自 Commons Lang 上游 skipped tests。
26. [x] 初跑 Commons Lang `ExceptionUtils.java` top10：`status_counts={"failed":2,"passed":8}`、`action_counts={"manual_review_internal":1,"ready":7,"repair_generated_test":2}`；剩余失败集中在 `asRuntimeException` 和 `rethrow` 的 type-erasure 泛型返回类型 `T` 泄漏。
27. [x] 修复 Java `ExceptionUtils.asRuntimeException` / `rethrow` coverage task：这类 type-erasure 异常传播路径生成 `RuntimeException` 抛出断言并校验原始异常对象，不再生成泛型返回值变量。
28. [x] 复跑 Commons Lang `ExceptionUtils.java` top10：`status_counts={"passed":10}`、`action_counts={"manual_review_internal":1,"ready":9}`，普通 repair 清零；private `getCauseUsingMethodName` 稳定归为 `manual_review_internal`。
29. [x] 生成 Commons Lang `Failable.java` top16 list-only 清单：过滤后实际只有 16 个候选，`tryWithResources` 缺口集中在 637、650-652、659、661-662，另有 `get*` wrapper、`run` 和 `stream` 候选。
30. [x] 初跑 Commons Lang `Failable.java` top1：`status_counts={"failed":1}`、`action_counts={"manual_review":1}`；生成测试退化为 `Failable.tryWithResources(null, null, null)`，在 `Objects.requireNonNull` 抛 `NullPointerException`，没有命中 651 行 `th == null` 资源异常分支。
31. [x] 修复 Java `Failable.tryWithResources` coverage task：按 `line_range` 生成 no-op action、抛错 resource、`AtomicReference` errorHandler 和 handler 抛错断言，覆盖 637、650-652、659、661-662 这些 functional interface 资源关闭和 errorHandler 路径，不再使用 null varargs。
32. [x] 复跑 Commons Lang `Failable.java` top1：`status_counts={"passed":1}`、`action_counts={"ready":1}`，`junit-50 Failable.tryWithResources 651-651` 从 `failed/manual_review` 收敛到真实 ready；复跑摘要中的 skipped 来自 Commons Lang 上游 skipped tests。
33. [x] 扩大 Commons Lang `Failable.java` 到 top4 初跑：`status_counts={"failed":3,"passed":1}`、`action_counts={"manual_review":2,"ready":1,"repair_generated_test":1}`；`Failable.get` 失败来自 `T result = Failable.get(null)` 编译错误，`getAsBoolean/getAsDouble` 失败来自 `null` supplier 运行时 NPE。
34. [x] 修复 Java `Failable.get*` / `run` wrapper coverage task：catch/rethrow 行使用 throwing lambda 触发原始 `RuntimeException`，并断言 `assertSame(failure, thrown)`，不再生成泛型 `T result`、`get*(null)` 或 primitive 默认值断言。
35. [x] 复跑 Commons Lang `Failable.java` top4：`status_counts={"passed":4}`、`action_counts={"ready":4}`，`tryWithResources 651`、`get 412`、`getAsBoolean 427`、`getAsDouble 442` 全部收敛为真实 ready；普通 repair 清零。

已完成补充：第十七个真实样本把非 RocketMQ Java 验证扩展到 Apache Commons Lang。这个样本提供了十一条有价值的质量证据：第一，复杂泛型反射内部路径会稳定生成可运行的 `manual_review_internal`，不会伪造成外部可直接断言的测试；第二，Appendable I/O error path 也会稳定归为手审，不会硬造不可控 `IOException` 断言；第三，裸类型变量数组/varargs 现在会稳定归为手审，不再生成不可编译的 `T[] result` 或重载歧义的 `addAll(null, null)`；第四，class-name helper 的公共分支需要 line-specific 输入，生成测试通过不等于命中 coverage 缺口；第五，CharSequence helper 的 `StringBuffer` 分支同样需要 line-specific 输入，不能用普通 `String` ready 代替；第六，时间状态 helper 需要把公开可触达错误路径生成异常断言，把 enum switch default 防御路径归为不可达；第七，public nested value object 不能因为同名前缀的 private enum 被误判为内部手审，限定名构造器和精确断言能把 `StopWatch.Split.toString` 推进到 ready；第八，异常传播 helper 的泛型返回类型不能直接写成测试局部变量类型，应按行段生成具体异常或返回值断言；第九，functional interface 和 varargs 不能退化成全 null 调用，应生成真实 action/resource/errorHandler 来命中资源关闭错误路径；第十，wrapper catch/rethrow 行不能用泛型局部变量或 null supplier，应使用 throwing lambda 断言原始异常对象；第十一，成熟库的大测试套件会显著拉高 per-task 成本，因此后续必须继续使用 list-only 选窗和小批量验证。下一步建议继续扩大 `Failable.java` 到 top8/top12，验证 `getAsInt/getAsLong/getAsShort`、`run`、`stream` 和 `tryWithResources` errorHandler 抛错路径。

## 第一百七十七阶段：Java 目标行命中校验

1. [x] 在 `validate_coverage_task` 的 Java/JUnit 路径默认启用覆盖率运行，让生成测试执行后产出 JaCoCo XML。
2. [x] 新增 JaCoCo 目标行命中校验：按 `coverage_task.line_range` 查找目标源码文件，收集 `coverage_hit_lines` / `coverage_missed_lines`；测试通过但目标行未命中时，验证结果从 `passed/ready` 降级为 `failed/needs_better_input`。
3. [x] 为“测试通过但目标行未覆盖”的假 ready 增加 fake Maven 单元测试：fake `mvn` 写入未覆盖目标行的 JaCoCo XML，`validate_coverage_task` 必须返回 `needs_better_input`，并在 metadata 中暴露 missed line。
4. [x] 复跑 Commons Lang `Failable.tryWithResources 651-651` 真实样本：结果仍为 `status=passed/action=ready`，metadata 为 `coverage_target_hit=true`、`coverage_hit_lines=[651]`，证明新增校验能在真实 Maven/JUnit + JaCoCo 链路读取目标行命中结果。
5. [x] 回归历史弱 ready 的 Commons Lang `ClassUtils.java` top4：`junit-45/46/77/78` 全部保持 `status=passed/action=ready`，metadata 分别确认命中目标行 `[1111]`、`[1114]`、`[1222]`、`[1258]`。
6. [x] 回归 Commons Lang `CharSequenceUtils.java` top8：`junit-44 CharSequenceUtils.toCharArray 419-419` 保持 `status=passed/action=ready`，metadata 确认 `coverage_target_hit=true`、`coverage_hit_lines=[419]`；其余 7 个 `regionMatches` 仍稳定归为 `manual_review_internal`。
7. [x] 回归 Commons Codec `Metaphone.java` top1：目标行命中校验推翻旧 `agned` ready，确认 `Metaphone.java:279` 被 JaCoCo 映射到被 `GN` 短路遮蔽的 `GNED` 侧；生成器改为 `manual_review_unreachable` 后复跑达到 `status_counts={"passed":1}`、`action_counts={"manual_review_unreachable":1}`。
8. [x] 为 Java 验证脚本新增 `TESTLOOP_VALIDATE_JAVA_TASK_IDS`，支持逗号分隔 task id 精确筛选；未显式传入 limit 时，脚本会按 id 数量自动收敛验证窗口，避免为单个历史弱 ready 回归跑完整 topN。
9. [x] 用 task id 精确筛选复查 3 个历史样本：Commons Lang `junit-44` / `junit-50` 只选中 2 条任务并保持 `passed/ready`，metadata 分别确认命中 `[419]`、`[651]`；Commons Codec `junit-130` 只选中 1 条任务并保持 `passed/manual_review_unreachable`，确认假 ready 不会回退。
10. [x] 为 Java 验证脚本新增 `TESTLOOP_VALIDATE_JAVA_TASKS_FILE`，支持从已有 coverage task JSONL 或 validation JSONL 读取任务，跳过 baseline coverage 生成；同时支持旧临时绝对源码路径重写到当前隔离 task worktree。
11. [x] 用 Commons Codec 真实 smoke 验证 `TASKS_FILE + TASK_IDS`：复用 `/tmp/testloop-commons-codec-taskids-junit130-results.jsonl`，直接加载 1 条任务并跳过 baseline copy / coverage，`junit-130 Metaphone.metaphone` 约 25 秒完成，结果保持 `passed/manual_review_unreachable`。
12. [x] 优化 Java/JUnit `run_tests`：path 指向 `src/test/**/*.java` 时，Maven 使用 `-Dtest=<TestClass>`，Gradle 使用 `--tests <TestClass>`，只运行生成的 TestLoop 测试类但继续生成 JaCoCo report。Commons Codec `junit-130` 从约 25 秒降到约 12 秒，Commons Lang `junit-44` 只运行 `CharSequenceUtilsTestLoopTest`，`Tests run: 1, Skipped: 0`，仍确认命中 `[419]`。
13. [x] 新增 `scripts/validate-java-regression-samples.sh`，固定回归 Commons Lang `junit-44/junit-50` 真实 ready 命中、Commons Codec `junit-130` 历史假 ready 降级为 `manual_review_unreachable`、Commons Lang `junit-52` 内部手审三类样本，并断言输出 JSONL 的 `status/action/coverage_target_hit`。

已完成补充：Java ready 的定义从“生成测试可运行”推进到“生成测试可运行且目标 line_range 被 JaCoCo 确认覆盖”。当前已覆盖真实命中样本 `Failable`、`ClassUtils`、`CharSequenceUtils`，也覆盖了需要降级为手审的 `Metaphone`。Java 真实项目回归现在可以按 task id 精确筛选、复用已有 task JSONL 跳过 baseline coverage、只运行生成的测试类，并通过固定回归脚本快速覆盖“真 ready + 假 ready + 手审”三类样本。下一步应把这种小型固定回归扩展到 JS/Python 的代表性闭环样本，形成跨语言 smoke 矩阵。

## 第一百七十八阶段：跨语言固定 smoke 矩阵

1. [x] 为 JS/TS 真实项目验证入口新增 `TESTLOOP_VALIDATE_JS_TASKS_FILE` 和 `TESTLOOP_VALIDATE_JS_TASK_IDS`，支持从已有 coverage task / validation JSONL 读取任务，跳过 baseline coverage，并按 task id 精确回归。
2. [x] 为 Python 真实项目验证入口新增 `TESTLOOP_VALIDATE_PY_TASKS_FILE` 和 `TESTLOOP_VALIDATE_PY_TASK_IDS`，支持从已有 coverage task / validation JSONL 读取任务，跳过 baseline coverage，并按 task id 精确回归。
3. [x] 抽出通用 coverage task JSONL 读取、validation output 解包、task id 筛选和旧临时路径后缀重映射 helper，避免 JS/Python 复用历史 JSONL 时继续写入旧 `/var/folders/.../task-*` 路径。
4. [x] 新增 `scripts/validate-py-regression-samples.sh`，默认复用 `/tmp/testloop-click-sample` 与 `/tmp/testloop-click-pytest-top5-regression.jsonl`，固定回归 Click `pytest-1/pytest-3` 两个 ready 样本。
5. [x] 新增 `scripts/validate-regression-smoke.sh`，默认串联 Java + JS + Python 固定小回归，输出到统一 smoke 目录；Java 覆盖 ready / unreachable / internal 三类样本，JS 覆盖 Jest ready 样本，Python 覆盖 pytest ready 样本。
6. [x] 用 Click 真实项目 smoke 验证 `TESTLOOP_VALIDATE_PY_TASKS_FILE + TESTLOOP_VALIDATE_PY_TASK_IDS`：`pytest-1 get_text_stream` 在 `uv run python -m pytest` runner 下保持 `passed/ready`。
7. [x] 基于当前源码快照重新导出 JS/Jest 样本 JSONL：ip2region JavaScript binding `jest-1/jest-2 versionFromHeader` 在 `NODE_OPTIONS='--experimental-vm-modules --no-warnings' npx jest --runTestsByPath {path}` runner 下保持 `passed/ready`。
8. [x] 新增 `scripts/validate-js-regression-samples.sh`，默认复用 `/Users/binlee/code/open-source/ip2region/binding/javascript` 与 `/tmp/testloop-ip2region-js-jest-top2-current.jsonl`，固定回归 `jest-1/jest-2` 两个 ready 样本。
9. [x] 新增 `docs/regression-smoke.md`，记录固定 smoke 的总入口、默认真实项目路径、JSONL 依赖、语言跳过开关、JS/Python runner 约束和当前边界。
10. [x] 复查 JS 非 ready 候选：ip2region 扩大到 top12 后只有 `ready` 与 `repair_generated_test`，旧 ufo `manual_review_no_runtime` 样本对应源码已不存在，Codex SDK TypeScript 旧 `manual_review_internal` 样本当前缺 Jest 依赖，不适合作为默认 smoke。
11. [x] 新增仓库内 `testdata/js-no-runtime` TypeScript 纯类型 fixture 和 `scripts/js-manual-review-runner.js`，让 JS smoke 可稳定回归 `jest-no-runtime-1 -> manual_review_no_runtime`，不再依赖已漂移的外部 TS 样本。
12. [x] 新增仓库内 `testdata/js-internal` TypeScript 未导出 helper fixture，复用轻量 runner 固定回归 `jest-internal-1 -> manual_review_internal`，让 JS smoke 覆盖 ready / no-runtime / internal 三类输出。
13. [x] 新增 Python name-mangled private method 生成规则、`testdata/py-internal` fixture 和 `scripts/py-manual-review-runner.py`，固定回归 `pytest-internal-1 -> manual_review_internal`，让 Python smoke 也覆盖非 ready 手审路径。
14. [x] 新增 `scripts/fixture-task-jsonl.py`，把 JS/Python fixture coverage task JSONL 从 regression 脚本 heredoc 中抽出，统一维护 `js-no-runtime`、`js-internal`、`py-internal` 三个 preset。
15. [x] 强化 JS/Python manual-review fixture runner：JS 从生成测试中提取 `describe(...)` / `it.skip(...)` 名称输出 Jest 风格 skipped 摘要，Python 输出真实 pytest node id，并补 parser 回归测试固定这类输出。
16. [x] 新增 mcp-hub 真实 Vitest repair 样本：`ConfigManager.loadConfig` 空 config paths 分支先固定回归历史 `failed/repair_generated_test`，再补强 JS 生成器对 `if (...) { throw ... }` branch 的识别，当前已收敛为 `passed/ready`；`TESTLOOP_VALIDATE_JS_ALLOWED_FAILURE_ACTIONS` 保留为显式期望失败 action 的验证开关，默认 top-N 验证仍保持严格。
17. [x] 新增 haoy-apk-station backend 真实 FastAPI environment 样本：`app.main` 的 `serve_frontend` 依赖 `frontend/dist` 在模块导入阶段动态定义，固定回归 `pytest-apk-frontend-env-1 -> manual_review_environment`，让 Python smoke 不再只靠仓库 fixture 覆盖非 ready。
18. [x] 新增 haoy-apk-station backend 真实 FastAPI external-service 样本：`app.api.apps.download_apk` 的对象存储代理下载 timeout 固定回归 `pytest-apk-download-external-1 -> failed/manual_review_external_service`，并新增 pytest 风格 timeout runner 验证这类失败不会进入普通 repair。
19. [x] 新增 Python/SQLAlchemy 数据库手审分类：coverage task 命中 `db.commit` / SQLAlchemy 事务错误时生成 `manual_review_database` skip，`validate_coverage_task` 会转成 `manual_review_database` metadata；haoy-apk-station `delete_app` 固定回归 `pytest-apk-delete-db-1 -> manual_review_database`。
20. [x] 新增 mcp-hub 真实 Vitest EnvResolver placeholder 样本：`EnvResolver._resolveStringWithPlaceholders` 的普通占位符替换和缺失环境变量分支保持 `passed/ready`，用于覆盖真实 JS class 私有状态和环境变量输入隔离。
21. [x] 新增 mcp-hub 真实 Vitest workspace cache ready 样本：`WorkspaceCacheManager.updateWorkspaceState`、`cleanupStaleEntries` 通过 mock `_withLock/_readCache/_writeCache/_isProcessRunning` 隔离 XDG cache 文件、lock 文件和真实进程探测，固定回归 `passed/ready`。
22. [x] 新增 mcp-hub 真实 Vitest workspace lock 环境手审样本：`WorkspaceCacheManager._withLock` 依赖 fs exclusive lock、重试时序和 stale lock 清理，固定回归 `passed/manual_review_environment`，避免生成会写真实 lock 文件的假 ready 测试。
23. [x] 将 mcp-hub workspace 样本纳入 `scripts/validate-js-regression-samples.sh` 和 `scripts/validate-regression-smoke.sh` 默认矩阵，完整 smoke 已验证通过，CI 已通过。
24. [x] 新增 mcp-hub 真实 Vitest SSE lifecycle 样本：`SSEManager.setupAutoShutdown` 使用 fake timers、fake `workspaceCache` 和一次性 `SIGTERM` listener 覆盖自动关闭 timer，固定回归 `passed/ready`；初版 mock `process.emit` 会导致 Vitest worker `onTaskUpdate` timeout，已改为不 mock 进程事件分发。
25. [x] 将 mcp-hub SSE 样本纳入 `scripts/validate-js-regression-samples.sh` 默认矩阵，补上长连接/事件流生命周期类路径的低成本回归入口。
26. [x] 新增 mcp-hub 真实 Vitest SSE close lifecycle 样本：`SSEManager.addConnection` 使用 `EventEmitter` request 触发 `req.emit('close')`，断言连接状态变为 disconnected、`connections` 删除对应 id、workspace cache 从 1 更新到 0，并设置 auto-shutdown timer；固定回归 `passed/ready`。
27. [x] 新增 mcp-hub 真实 Vitest SSE send failure 样本：`SSEManager.addConnection` 返回的 `connection.send` 通过 throwing `res.write` 触发 `SSE_SEND_ERROR` 路径，断言 send 返回 `false`、连接状态变为 `error`，并通过 `broadcast` 清理 dead connection；固定回归 `passed/ready`。
28. [x] 新增 mcp-hub 真实 Vitest SSE directed send 样本：`SSEManager.sendToClient` 同时覆盖 missing client、disconnected client 和 connected send delegation，断言前两类返回 `false` 且不调用 `send`，connected 时委托 `connection.send`；固定回归 `passed/ready`。
29. [x] 新增 mcp-hub 真实 Vitest DevWatcher lifecycle 样本：`DevWatcher.stop` 覆盖 not watching 早返回和 watching cleanup，断言 debounce timer、changed files、watcher close、watcher 引用和 `isWatching` 状态被清理；`DevWatcher.start` 通过 chokidar mock 触发 watcher `error` 事件，固定回归 `passed/ready`。
30. [x] 整理 v0.5.0 release readiness：新增 `docs/plan-release-notes-v0.5.0.md`，把固定 smoke 矩阵、Agent 闭环定位、真实项目验证证据和发布前门禁收敛到中文发布草案；README 补充面向 Agent 的快速演示路径。

已完成补充：固定 smoke 矩阵已从单语言 Java 扩展到 Java + JS + Python。当前默认矩阵重点保证真实项目小样本可以低成本复用历史任务并跑完整 `generate_tests -> run_tests -> parse/fix/coverage` 闭环。JS 侧这次避开了旧 ufo JSONL 的源码漂移，改用当前本机可运行的 ip2region JavaScript binding；同时确认 Jest 项目需要用 `--runTestsByPath` 避免生成测试名与既有测试文件模糊匹配。当前 JS smoke 已补上仓库内 TypeScript no-runtime、未导出 helper fixture，以及 mcp-hub 真实 Vitest async throwing branch、EnvResolver placeholder、DevWatcher lifecycle、SSE lifecycle、WorkspaceCacheManager cache/lock 样本，稳定覆盖 `ready`、`manual_review_no_runtime`、`manual_review_internal` 和真实项目 `manual_review_environment`，并防止历史 `repair_generated_test` 回退、DevWatcher stop 退化成只测早返回、watcher error 路径启动真实 chokidar、mock `process.emit` 干扰 Vitest worker、SSE close 退化成空 `req.on` mock、SSE send failure 退化成空 `res.write` mock、SSE directed send 退化成只测 missing client，或把真实文件锁/进程探测路径误判为 ready；Python smoke 已覆盖 Click 真实 ready、仓库 name-mangled private internal fixture、haoy-apk-station 真实 FastAPI 动态前端入口 `manual_review_environment`、对象存储下载代理 timeout 的 `failed/manual_review_external_service`，以及 SQLAlchemy 删除事务错误的 `manual_review_database`，不再只靠仓库 fixture 证明非 ready。fixture task JSONL 已抽到统一 helper，manual-review runner 也已改成读取生成测试名并输出更接近真实框架的 skipped 摘要；v0.5.0 发布说明草案和 README Agent 快速演示路径也已收口。下一步应执行 v0.5.0 候选发布前门禁，确认 `go test ./...`、固定 smoke、二进制构建和帮助输出都稳定后，再决定是否打 tag 和创建 release draft。

## 第一百七十九阶段：v0.5.0 版本准备改动

1. [x] 更新 `main.go` MCP implementation version 到 `0.5.0`。
2. [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.5.0 - 2026-07-17`。
3. [x] 同步 README 中当前 Release、手动下载示例、Windows 下载示例到 `v0.5.0`。
4. [x] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.5.0`。
5. [x] 新增 `docs/plan-release-v0.5.0.md`，记录候选版本准备、验证项和正式发布前待办。

已完成补充：v0.5.0 的版本号和用户安装文档已切到新版本，发布检查清单已建立；当前还没有打 tag，也没有创建 GitHub Release 或更新 Homebrew tap。下一步应完成本地发布前验证和远端 CI，再进入 tag、Release Artifacts、资产校验、GitHub Release 正文和 Homebrew tap 发布核验阶段。

## 第一百八十阶段：v0.5.0 正式发布和发布后核验

1. [x] 创建并推送 tag `v0.5.0`，GitHub Release 已创建为 `testloop-mcp v0.5.0`。
2. [x] Release Artifacts workflow `29558114233` 已通过，生成五个平台资产和 `.sha256`。
3. [x] `scripts/verify-release-assets.sh v0.5.0` 验证 GitHub Release 包含 10 个必需资产。
4. [x] GitHub Release 正文已更新为正式 v0.5.0 发布说明。
5. [x] 仓库内 `Formula/testloop-mcp.rb` 和 `sleticalboy/homebrew-tap` 已更新到 `0.5.0`，tap commit `e201f8f` 已推送。
6. [x] 本机 Homebrew tap 已更新到 `e201f8f`，并完成 `brew fetch`、`brew audit --formula --strict`、`brew upgrade --formula`、`brew test`。
7. [x] Post-Release Verify workflow `29559912737` 已通过，资产清单和五平台安装脚本 dry run 全部通过。

已完成补充：v0.5.0 已正式发布，发布资产、GitHub Release 正文、Homebrew tap 和五平台安装脚本验证均已闭环。下一步应回到产品主线，不再继续堆发布流程：优先选择一个新的真实项目样本或补一个最小 MCP 客户端端到端 demo，用发布后的 `0.5.0` 能力展示 Agent 如何消费结构化反馈并完成一轮测试修复。

## 第一百八十一阶段：最小 MCP 客户端端到端 demo

1. [x] 新增 `examples/mcp-client-demo`，使用 in-memory MCP server/client 调用真实工具，而不是伪造 JSON。
2. [x] demo 会创建临时 Go 项目并故意制造失败断言，调用 `run_tests` 且开启 `include_fix_suggestions`，验证 `fix_suggestions[].repair_task` 可被客户端直接消费。
3. [x] demo 优先读取 `structuredContent`，同时校验 text JSON 与结构化内容一致，给 Agent/编辑器集成提供稳定消费契约样例。
4. [x] demo 会模拟 Agent 修复临时测试，复跑 `run_tests coverage=true`，再用 `parse_coverage` 解析 Go coverprofile，展示失败修复后再进入覆盖率反馈。
5. [x] README 和 `docs/agent-workflow.md` 已加入 `go run ./examples/mcp-client-demo` 入口。

已完成补充：项目现在有一条不依赖外部真实项目、不污染仓库文件、可直接运行的 MCP 客户端集成验收路径。下一步应继续强化“客户端可消费”这条主线：优先把 demo 输出或工具返回字段整理成稳定 schema/契约测试，确保后续新增字段不会破坏 Codex、Claude Code、Cursor 这类 Agent 的自动决策。

## 第一百八十二阶段：Agent 结构化契约固定

1. [x] 新增 `docs/agent-contract.md`，明确客户端优先读取 `structuredContent`，旧客户端 fallback 到 `content[0].text` JSON。
2. [x] 文档固定兼容规则：已发布字段名默认稳定，可以追加字段，Agent 必须忽略未知字段，自动决策不依赖自然语言 `error` 文本。
3. [x] 文档固定 `run_tests`、`generate_tests` 和 `validate_coverage_task` 的主入口字段，以及 `repair_task`、`provider_error`、`status/action/metadata` 的 Agent 决策用法。
4. [x] 新增 `types/agent_contract_test.go`，用反射检查关键输出类型的 JSON tag 字段，防止后续重命名或删除字段破坏 Agent 消费方。
5. [x] README 已补充 Agent 结构化契约入口。

已完成补充：客户端可消费契约已经从“demo 能跑”推进到“文档和测试固定字段名”。下一步建议继续沿这条主线做真实 MCP client 兼容性验证：用 stdio 或 Streamable HTTP 传输启动真实 `testloop-mcp` 进程，而不是只用 in-memory session，验证发布二进制在 Codex/Claude/Cursor 同类客户端场景下的初始化、工具列表、工具调用和结构化返回一致性。

## 第一百八十三阶段：真实 stdio 传输兼容性 smoke

1. [x] 在 `test/e2e` 新增真实 stdio 进程级 smoke，测试会先把当前仓库构建成临时 `testloop-mcp` 二进制。
2. [x] 使用 MCP SDK `CommandTransport` 启动 `testloop-mcp --transport=stdio`，覆盖客户端实际通过 stdin/stdout 连接服务端的路径。
3. [x] smoke 执行 `ListTools` 并确认 `parse_results` 工具存在，验证初始化和工具发现路径。
4. [x] smoke 调用 `parse_results`，复用现有 e2e helper 校验 `structuredContent` 与 `content[0].text` JSON 语义一致。
5. [x] `docs/agent-contract.md` 已记录该 smoke 的覆盖范围。

已完成补充：现在已经同时覆盖 in-memory MCP session、最小客户端 demo、字段级契约测试和真实 stdio 进程传输 smoke。下一步建议补 Streamable HTTP 传输 smoke：启动真实 `testloop-mcp --transport=http --addr=127.0.0.1:0` 或可控端口，先检查 `/healthz`，再用 `StreamableClientTransport` 完成 `tools/list` 和轻量 `parse_results` 调用，避免 HTTP 接入路径只停留在健康检查。

## 第一百八十四阶段：真实 Streamable HTTP 传输兼容性 smoke

1. [x] 在 `test/e2e` 新增真实 Streamable HTTP 进程级 smoke，测试会先构建当前仓库的临时 `testloop-mcp` 二进制。
2. [x] smoke 选择本机临时 TCP 端口，启动 `testloop-mcp --transport=http --addr=<addr>`，覆盖真实 HTTP server 启动路径。
3. [x] smoke 会轮询 `/healthz`，确认 HTTP 服务就绪后再创建 MCP client。
4. [x] 使用 MCP SDK `StreamableClientTransport` 连接 `/mcp`，执行 `ListTools` 并确认 `parse_results` 工具存在。
5. [x] smoke 调用 `parse_results`，复用现有 e2e helper 校验 `structuredContent` 与 `content[0].text` JSON 语义一致。
6. [x] `docs/agent-contract.md` 已记录 Streamable HTTP smoke 的覆盖范围。

已完成补充：stdio 和 Streamable HTTP 两条真实接入路径都已经进入 CI，Agent 结构化契约不再只靠 in-memory 测试保证。下一步建议从“传输能连通”推进到“客户端配置可验收”：补一个端到端配置诊断 smoke，生成 Codex/Claude/Cursor 配置片段后用 `--check-config` 验证本次构建出的真实二进制路径，确保安装接入文档和 CLI 诊断不会随发布漂移。

## 第一百八十五阶段：客户端配置可验收 smoke

1. [x] 新增真实二进制级 CLI smoke，测试会先构建当前仓库的临时 `testloop-mcp` 二进制。
2. [x] smoke 调用 `--print-config=all --config-command=<built-binary> --config-http-url=<http-url>`，生成 Codex stdio、Codex HTTP、Claude 和 Cursor 配置片段。
3. [x] smoke 校验生成内容包含 Codex、Claude、Cursor 标识、真实二进制 command 和 HTTP endpoint。
4. [x] smoke 将生成内容通过 stdin 传给同一个二进制的 `--check-config -`，确认 command 可执行、HTTP URL 合法，并检查 4 个 `testloop` entry 都通过。
5. [x] `docs/installation.md` 已记录该 smoke 的覆盖范围。

已完成补充：首次接入链路现在有真实进程级保护：安装后的二进制不仅能启动 stdio/HTTP MCP 服务，也能生成并校验客户端配置片段。下一步建议整理一个轻量“接入验收脚本”：把 `--doctor-config`、`--print-config`、`--check-config` 和 stdio/HTTP smoke 的用户侧命令收敛成文档或脚本，方便用户安装后一次性确认本机可接入。

## 第一百八十六阶段：用户侧接入验收脚本

1. [x] 新增 `scripts/verify-client-setup.sh`，默认验证 PATH 上的 `testloop-mcp`，也支持通过参数或 `TESTLOOP_MCP_COMMAND` 指定二进制。
2. [x] 脚本会执行二进制 `--print-config=codex --config-command=<binary>`，确认安装产物可执行且配置生成入口可用。
3. [x] 脚本会执行 `--doctor-config`，确认本机客户端配置诊断入口可运行。
4. [x] 脚本会执行 `--print-config=all --config-command=<binary> --config-http-url=<url>`，并通过 `--check-config -` 校验生成的 Codex、Codex HTTP、Claude 和 Cursor 配置片段。
5. [x] 脚本会启动一次 `--transport=http`，轮询 `/healthz`，验证 HTTP 模式探活；支持 `TESTLOOP_MCP_VERIFY_HTTP_ADDR` 换端口和 `TESTLOOP_MCP_VERIFY_SKIP_HTTP=true` 跳过。
6. [x] README 和 `docs/installation.md` 已加入安装后自检脚本入口。
7. [x] 新增 `test/verify_client_setup_test.sh`，构建当前仓库临时二进制后运行 `scripts/verify-client-setup.sh` 的 skip HTTP 路径，并覆盖缺失二进制失败提示。
8. [x] CI 已加入 `sh test/verify_client_setup_test.sh`，防止脚本参数解析、配置 roundtrip 和基础错误处理漂移。

已完成补充：用户从安装到接入前的自检路径已经收敛成一条脚本命令，并且已有 CI 回归覆盖 skip HTTP 路径；真实 HTTP 探活则由脚本手动验证和 e2e Streamable HTTP 进程级 smoke 共同覆盖。下一步建议补一份更短的用户向导，把“安装 -> 自检 -> 写入 Codex/Claude/Cursor 配置 -> 开始使用工具”的步骤收敛成 5 分钟接入文档，避免 README / installation / agent workflow 之间跳转过多。

## 第一百八十七阶段：5 分钟接入向导

1. [x] 新增 `docs/quickstart.md`，把安装、自检、客户端配置、重启客户端和最小闭环验证收敛成单页文档。
2. [x] quickstart 优先给出 Homebrew 和安装脚本两条安装路径，不重复完整 Release 下载细节。
3. [x] quickstart 加入 `scripts/verify-client-setup.sh` 自检路径，并提供换 HTTP 端口和跳过 HTTP 探活的命令。
4. [x] quickstart 分别给出 Codex、Claude、Cursor 的 `--print-config` 命令和写入位置。
5. [x] quickstart 收尾到 `go run ./examples/mcp-client-demo`，用最小 demo 展示 `run_tests -> repair_task -> rerun -> parse_coverage`。
6. [x] README 和 `docs/installation.md` 已加入 quickstart 入口，避免新用户在长文档里寻找最短路径。

已完成补充：接入体验现在形成了“快速向导 -> 详细安装 -> Agent 工作流/结构化契约”的文档层次。下一步建议回到产品能力本身：选一个新的真实项目样本，优先 JS/TS 或 Go 小项目，用 v0.5.0 之后的结构化契约和自检脚本做一条可公开展示的端到端案例。

## 第一百八十八阶段：Agent 闭环展示案例固定

1. [x] 新增 `docs/showcase-agent-loop.md`，把最小 MCP 客户端 demo 的运行方式、预期输出和验收边界整理成中文展示文档。
2. [x] 新增 `test/mcp_client_demo_test.sh`，运行 `go run ./examples/mcp-client-demo` 并断言 `run_tests -> repair_task -> rerun -> parse_coverage` 四个关键输出。
3. [x] 将 demo 回归脚本加入 CI，防止后续修改破坏展示案例或 Agent 消费路径。
4. [x] README、quickstart 和 Agent workflow 已链接展示案例，形成“接入向导 -> 展示验收 -> 工作流细节”的文档层次。

已完成补充：最小 Agent 闭环不再只是文档里的一条命令，而是有固定输出、验收边界和 CI 回归保护的展示案例。下一步建议选择一个外部可公开的小型 Go 或 JS/TS 项目，复用同一套结构化契约跑真实项目案例，重点展示 `validate_coverage_task` 的 `status/action` 如何指导 Agent 处理 ready、manual_review 和失败修复。

## 第一百八十九阶段：公开 Go 项目展示案例

1. [x] `scripts/validate-go-coverage-top-tasks.sh` 对齐 JS/Python/Java 验证入口，新增 `TESTLOOP_VALIDATE_GO_FILE_FILTER`、`TESTLOOP_VALIDATE_GO_TASK_IDS` 和 `TESTLOOP_VALIDATE_GO_TASKS_FILE`。
2. [x] 新增 `scripts/showcase-go-public-project.sh`，默认克隆 `google/uuid` 固定 commit `2d3c2a9cc518326daf99a383f07c4d3c44317e4d`，并精确验证 `go-test-1`。
3. [x] 新增 `docs/showcase-public-go.md`，记录公开项目案例的运行方式、当前验证结果、可配置项和不进入默认 CI 的原因。
4. [x] README 和 CHANGELOG 已加入公开 Go showcase 入口。

已完成补充：现在项目除了仓库内最小 MCP 客户端 demo，还有一条 opt-in 的公开 Go 项目覆盖率闭环案例。该案例展示 `validate_coverage_task` 在外部仓库上返回 `passed/ready` 的结构化决策信号，并且通过 task id 精确筛选避免 topN 里低价值任务干扰演示。下一步建议继续补一个 JS/TS 公开项目案例，优先选择依赖安装轻、coverage 命令稳定、能展示 `manual_review_internal` 或真实 `ready` 的仓库。

## 第一百九十阶段：公开 JS/TS 项目展示案例

1. [x] 评估 `nanoid`、`ufo` 和 `slugify` 三个公开 JS 项目；`ufo` 使用 Vitest + coverage-v8 + pnpm 锁文件，最适合作为公开 TS/Vitest showcase。
2. [x] 用 `unjs/ufo` 固定 commit `f06c800d0c59f2a4a1b9ba65eb6cb61a84419be6` 跑 top5：结果为 `status_counts={"passed":5}`、`action_counts={"manual_review_internal":1,"ready":4}`。
3. [x] 新增 `scripts/showcase-js-public-project.sh`，默认克隆 `ufo`、执行 `pnpm install --frozen-lockfile`，并精确验证 `vitest-1,vitest-2`。
4. [x] 新增 `docs/showcase-public-js.md`，记录 JS/TS showcase 的运行方式、当前验证结果、可配置项和 opt-in 边界。
5. [x] README 和 CHANGELOG 已加入公开 JS/TS showcase 入口。

已完成补充：公开 showcase 现在覆盖 Go 的 `passed/ready` 和 JS/TS 的 `ready/manual_review_internal` 决策分流。下一步建议把两个公开 showcase 收敛成一个统一索引文档，明确“默认 CI 保护什么、opt-in showcase 证明什么、真实项目 regression smoke 证明什么”，避免新用户在 README、quickstart、showcase 和 regression 文档之间来回跳。

## 第一百九十一阶段：展示路径索引收敛

1. [x] 新增 `docs/showcase.md`，统一解释默认 CI、公开 opt-in showcase 和真实项目 regression smoke 的区别。
2. [x] 在索引文档中列出最小 Agent demo、公开 Go showcase、公开 JS/TS showcase 和跨语言 regression smoke 的适用场景。
3. [x] README 的“面向 Agent 的快速演示路径”改为保留最小 demo，其他展示和深度回归入口统一链接到 `docs/showcase.md`。
4. [x] 保留各单项 showcase 文档，避免索引页承载过多命令细节。

已完成补充：展示文档层次现在更清晰：README 只保留最小路径和索引入口，`docs/showcase.md` 负责路线选择，单项 showcase 文档负责具体命令和边界。下一步建议补一个轻量脚本测试，固定 `scripts/showcase-*.sh --help` 输出和参数校验，避免 showcase 脚本后续改动时入口破坏但默认 CI 无感知。

## 第一百九十二阶段：showcase 脚本入口回归

1. [x] 新增 `test/showcase_scripts_test.sh`，对 `scripts/showcase-go-public-project.sh` 和 `scripts/showcase-js-public-project.sh` 执行 `bash -n`。
2. [x] 固定两个 showcase 脚本的 `--help` 输出和多余参数错误码。
3. [x] 固定 JS/TS showcase 在缺少 `pnpm` 时的错误提示，避免用户侧环境问题变成静默失败。
4. [x] CI 已加入 `sh test/showcase_scripts_test.sh`，只验证入口契约，不执行外部克隆或依赖安装。

已完成补充：公开 showcase 仍然保持 opt-in，不让默认 CI 依赖 GitHub/npm 网络；但脚本入口、帮助文案和基础错误处理已经有默认 CI 保护。下一步建议回到 MCP 工具本身，优先补一个 `validate_coverage_task` 的文本/结构化返回一致性 golden，确保 showcase 依赖的 `status/action` 在 handler 层不会漂移。

## 第一百九十三阶段：validate_coverage_task 结构化契约固定

1. [x] 新增 `TestHandleValidateCoverageTaskStructuredContentMatchesTextJSON`，用临时 Go 项目跑真实 `validate_coverage_task`。
2. [x] 将 `content[0].text` JSON、`result.StructuredContent` 和 handler 返回的 structured payload 归一化后做完整 JSON 等价比较。
3. [x] 额外断言 `status/action/coverage_task/generated/run_result/metadata` 等 showcase 依赖字段存在且语义正确。
4. [x] CHANGELOG 已记录该契约测试，防止后续只更新 showcase 文档而忽略底层 MCP 返回契约。

已完成补充：showcase 依赖的核心 `validate_coverage_task` 输出现在有 handler 层一致性回归保护。下一步建议补一份简短的 Agent 决策表，把 `validate_coverage_task.action` 的 ready、manual_review_*、needs_better_input、repair_generated_test 等动作整理成客户端执行建议，减少集成方误用 `status=passed`。

## 第一百九十四阶段：Agent action 决策表

1. [x] 新增 `docs/agent-action-guide.md`，按 `status/action` 组合整理客户端下一步动作。
2. [x] 明确 `passed/manual_review_*` 不是自动吸收测试，而是记录手审原因、改走公共入口、环境设计或依赖注入。
3. [x] 明确 `apply_fix_suggestions`、`repair_generated_test`、`needs_better_input`、`generation_error` 和 `run_error` 的处理优先级。
4. [x] README、Agent workflow 和 Agent contract 已链接该决策表。

已完成补充：Agent 集成方现在有了比长工作流更直接的 action 决策表，可以减少只看 `status=passed` 的误用。下一步建议补一个文档链接检查或轻量 markdown smoke，确保 README / docs 中新增的关键相对链接不会漂移。

## 第一百九十五阶段：文档相对链接回归

1. [x] 新增 `test/docs_links_test.sh`，扫描 `README.md` 和 `docs/*.md` 中的 Markdown 相对链接。
2. [x] 跳过外链、纯锚点、绝对路径和 fenced code block，只校验仓库内相对文件目标是否存在。
3. [x] 将文档链接检查加入 CI 的 `Run tests` 步骤，防止新增或移动文档后链接静默失效。
4. [x] CHANGELOG 已记录该 smoke，便于后续发布说明追踪。

已完成补充：文档展示路径和 Agent action 决策表现在有了基础链接回归保护。下一步建议回到真实 Agent 可消费性，补一个 `validate_coverage_task` 结果样例 JSON 文档或 golden fixture，让接入方可以直接看到 `ready/manual_review/failed` 三类结构化返回样例。

## 第一百九十六阶段：validate_coverage_task 返回样例

1. [x] 新增 `docs/validate-coverage-task-samples.md`，用中文说明 `passed/ready`、`passed/manual_review_internal`、`failed/apply_fix_suggestions` 和 `failed/needs_better_input` 四类典型返回。
2. [x] 每个样例都保留 Agent 决策所需的 `status/action/coverage_task/generated/run_result/metadata` 形状。
3. [x] 新增 `test/docs_json_examples_test.sh`，解析样例文档中的 fenced JSON，校验 JSON 合法性和关键字段存在。
4. [x] README、Agent action guide、Agent contract 已链接样例文档，CI 已纳入 JSON 示例检查。

已完成补充：接入方现在不需要只读长字段说明，就能直接对照结构化返回样例实现客户端分流。下一步建议补一个小型客户端消费 fixture：给定这几类 JSON 样例，验证示例 Agent 决策函数会输出 accept/manual-review/apply-repair/needs-better-input 四种动作。

## 第一百九十七阶段：Agent 决策消费 fixture

1. [x] 新增 `examples/agent-decision-demo`，读取 `docs/validate-coverage-task-samples.md` 中的 fenced JSON 样例。
2. [x] 示例客户端按 `status/action` 输出 `accept`、`manual-review`、`apply-repair` 和 `needs-better-input` 四类动作。
3. [x] 新增 `test/agent_decision_demo_test.sh`，固定示例输出，防止文档样例和客户端消费逻辑脱节。
4. [x] CI 已纳入 Agent 决策 demo 回归，README 和样例文档已补充运行入口。

已完成补充：结构化返回样例现在不只是静态文档，也有最小客户端消费代码和 CI 输出回归保护。下一步建议补一个更贴近真实 MCP 客户端的 `validate_coverage_task` fixture：用仓库内临时项目真实调用工具，输出一份可复用的 ready 样例 JSON，减少文档样例与真实 handler 输出之间的距离。

## 第一百九十八阶段：validate_coverage_task 真实 ready fixture

1. [x] 新增 `docs/fixtures/validate-coverage-task-ready.json`，作为可复用的 `passed/ready` 结构化样例。
2. [x] 新增 handler 级 fixture 测试，用临时 Go 项目真实调用 `HandleValidateCoverageTask`。
3. [x] 测试会把临时目录路径规范成相对路径，并过滤 `raw_output` 等机器相关字段，再与 fixture 比对。
4. [x] 样例文档、CHANGELOG 和 roadmap 已链接/记录该真实 fixture。

已完成补充：`validate_coverage_task` 现在既有手写说明样例，也有来自真实 handler 的 ready fixture 回归。下一步建议补 `manual_review_*` 的真实 handler fixture，优先选择仓库内 JS internal fixture，固定 `passed/manual_review_internal` 的结构化形状。

## 第一百九十九阶段：manual_review_internal 真实 fixture

1. [x] 新增 `docs/fixtures/validate-coverage-task-manual-review-internal.json`，固定 `passed/manual_review_internal` 的结构化样例。
2. [x] 复用临时 JS/Vitest internal class 场景真实调用 `HandleValidateCoverageTask`，生成稳定投影并与 fixture 比对。
3. [x] fixture 投影保留 `status/action/coverage_task/generated/run_result/metadata`，并包含 `internal_symbol/internal_reason`。
4. [x] 样例文档、CHANGELOG 和 roadmap 已链接/记录该真实 fixture。

已完成补充：真实 handler fixture 现在覆盖 `ready` 与 `manual_review_internal` 两类 Agent 关键分流。下一步建议补 `failed/apply_fix_suggestions` 的真实 fixture，复用已有失败测试和 repair task 输出，固定 Agent 修复闭环样例。

## 第二百阶段：apply_fix_suggestions 真实 fixture

1. [x] 新增 `docs/fixtures/validate-coverage-task-apply-fix-suggestions.json`，固定 `failed/apply_fix_suggestions` 的结构化样例。
2. [x] 新增 handler 级 fixture 测试，用临时 Go 项目真实触发失败测试、失败解析和 `repair_task` 生成。
3. [x] fixture 投影保留 `failures[]`、`fix_suggestions[]` 和 `repair_task`，并规范化临时路径。
4. [x] 样例文档、CHANGELOG 和 roadmap 已链接/记录该真实 fixture。

已完成补充：真实 handler fixture 现在覆盖 `ready`、`manual_review_internal` 和 `apply_fix_suggestions` 三条 Agent 核心分流。下一步建议补一个 fixture 索引文档，把 `docs/fixtures/*.json` 的来源、稳定字段和不稳定字段过滤规则说明清楚，方便接入方复用。

## 第二百零一阶段：真实 fixture 索引

1. [x] 新增 `docs/fixtures.md`，统一列出 `docs/fixtures/*.json` 覆盖的 `status/action`、来源和 Agent 下一步。
2. [x] 明确 fixture 稳定字段，包括 `coverage_task`、`generated`、`run_result.failures[]`、`fix_suggestions[].repair_task` 和 `metadata`。
3. [x] 明确过滤规则：临时绝对路径转相对路径，过滤 `raw_output`，保留真实 `failures` JSON 形状。
4. [x] README 和 `docs/validate-coverage-task-samples.md` 已链接 fixture 索引，文档链接检查会覆盖新增入口。

已完成补充：真实 handler fixture 现在有了统一入口，接入方可以直接选择 ready、manual-review 或 apply-fix 样例做客户端实现和回归。下一步建议补一个小脚本校验 `docs/fixtures/*.json` 的 `status/action` 覆盖清单，防止新增 fixture 后忘记更新索引。

## 第二百零二阶段：fixture 索引校验

1. [x] 新增 `test/fixtures_index_test.sh`，扫描 `docs/fixtures/*.json` 并读取每个 fixture 的 `status/action`。
2. [x] 校验每个 fixture 文件名和 ``status/action`` 都已登记到 `docs/fixtures.md`。
3. [x] 固定当前覆盖集合：`passed/ready`、`passed/manual_review_internal`、`failed/apply_fix_suggestions`。
4. [x] CI 已纳入 fixture 索引校验，CHANGELOG 和 roadmap 已记录。

已完成补充：真实 fixture 的索引不再只靠人工维护，新增 fixture 或新增 action 分流时会要求同步更新索引和校验集合。下一步建议补一个简短的客户端集成说明，把 `docs/fixtures.md`、`agent-decision-demo` 和 MCP `structuredContent` 消费顺序串成“接入方如何用这些 fixture 做自己的回归”的流程。

## 第二百零三阶段：客户端 fixture 集成说明

1. [x] 新增 `docs/client-integration.md`，说明客户端应优先读取 `structuredContent`，旧客户端再 fallback 到 `content[0].text` JSON。
2. [x] 文档串联 `examples/agent-decision-demo` 与真实 fixture，给出 `accept/manual-review/apply-repair/needs-better-input` 的最小回归建议。
3. [x] 明确客户端测试应覆盖未知字段兼容、`manual_review_*` 不自动修复、`apply_fix_suggestions` 读取 `repair_task` 等行为。
4. [x] README、Agent contract 和 fixture 索引已链接客户端集成说明。

已完成补充：接入方现在有了“字段契约 -> action 决策表 -> 真实 fixture -> 客户端回归”的完整文档路径。下一步建议补一个 CI 检查，确保 `docs/client-integration.md` 提到的 fixture 文件和 `examples/agent-decision-demo` 命令持续存在。

## 第二百零四阶段：客户端集成说明入口校验

1. [x] 新增 `test/client_integration_doc_test.sh`，校验 `docs/client-integration.md` 中的关键入口仍然存在。
2. [x] 固定 `go run ./examples/agent-decision-demo` 命令与 `examples/agent-decision-demo/main.go` 的对应关系。
3. [x] 校验文档中的三份 `docs/fixtures/*.json` 链接指向真实文件，并保留 `structuredContent` / `content[0].text` 消费顺序关键字。
4. [x] CI 已纳入客户端集成说明入口校验，CHANGELOG 和 roadmap 已记录。

已完成补充：客户端集成文档不再只是静态说明，关键示例入口和 fixture 链接都有 CI 保护。下一步建议回到工具能力本身，补一个 `needs_better_input` 的真实 handler fixture，让真实 fixture 覆盖第四条 Agent 分流。

## 第二百零五阶段：needs_better_input 真实 fixture

1. [x] 新增 `TestHandleValidateCoverageTaskNeedsBetterInputFixture`，用临时 Java/JUnit 项目真实调用 `HandleValidateCoverageTask`。
2. [x] 固定测试命令通过但 JaCoCo 目标行未命中的场景，输出 `failed/needs_better_input`。
3. [x] 新增 `docs/fixtures/validate-coverage-task-needs-better-input.json`，保留 `coverage_target_hit`、`coverage_missed_lines` 和 `coverage_miss_reason` 等客户端决策字段。
4. [x] 更新 fixture 索引、客户端集成说明和 fixture 索引校验，四条 Agent 分流都已有真实 handler fixture 覆盖。

已完成补充：真实 handler fixture 现在覆盖 `ready`、`manual_review_internal`、`apply_fix_suggestions` 和 `needs_better_input` 四条核心客户端分流。下一步建议补一个 fixture 驱动的客户端决策 demo，让 `examples/agent-decision-demo` 直接读取 `docs/fixtures/*.json`，减少文档样例与真实 fixture 的双轨维护。

## 第二百零六阶段：fixture 驱动的客户端决策 demo

1. [x] `examples/agent-decision-demo` 改为读取 `docs/fixtures/validate-coverage-task-*.json`。
2. [x] demo 输出加入 fixture 文件名，方便接入方确认每条客户端分流来自哪个真实 handler 投影。
3. [x] 保留 `accept/manual-review/apply-repair/needs-better-input` 的固定展示顺序，避免文件名字母序影响回归输出。
4. [x] 更新 `test/agent_decision_demo_test.sh`、客户端集成说明和结构化样例文档，确保文档描述与真实入口一致。

已完成补充：客户端最小 demo 已从“读取文档样例”升级为“读取真实 handler fixture”，接入方可以直接照着这个入口实现自己的回归。下一步建议把 `docs/fixtures/*.json` 的客户端决策映射抽成一个独立测试脚本，校验每个 fixture 都能被映射到预期动作，防止 demo 逻辑和 fixture 集合未来漂移。

## 第二百零七阶段：fixture 决策映射独立校验

1. [x] 新增 `test/fixture_decision_mapping_test.sh`，直接扫描 `docs/fixtures/validate-coverage-task-*.json`。
2. [x] 固定 `passed/ready -> accept`、`passed/manual_review_internal -> manual-review`、`failed/apply_fix_suggestions -> apply-repair`、`failed/needs_better_input -> needs-better-input` 的客户端动作映射。
3. [x] 校验 `docs/fixtures.md` 和 `docs/client-integration.md` 中仍然登记这些 action 与客户端动作说明。
4. [x] CI 已纳入 fixture 决策映射校验，新增 fixture 或新增 action 时必须同步更新映射和文档。

已完成补充：真实 fixture 现在不仅有 handler 级输出校验，也有客户端动作映射校验。下一步建议补一份短的 MCP 客户端契约测试说明，把 `structuredContent`、fixture、demo 和这些 shell 校验串成“接入方如何复制到自己项目 CI”的最小模板。

## 第二百零八阶段：MCP 客户端契约测试说明

1. [x] 新增 `docs/mcp-client-contract-tests.md`，面向 MCP 客户端、编辑器插件和 AI Coding Agent 接入方。
2. [x] 明确客户端最小契约：优先 `structuredContent`、fallback `content[0].text`、按 `status/action` 分流、忽略未知字段。
3. [x] 给出复制真实 fixture 到客户端 CI 的推荐断言，包括 `repair_task` 和 `coverage_miss_reason` 两类关键分支。
4. [x] 串联仓库内 `fixtures_index`、`fixture_decision_mapping`、`client_integration_doc`、`agent_decision_demo` 四个 shell 校验和 `test/e2e` 进程级 smoke。
5. [x] README、Agent contract 和客户端集成说明已链接该契约测试说明。

已完成补充：接入方现在有了从字段契约到真实 fixture、demo、CI 校验的最小复制模板。下一步建议补一个 release 前文档索引检查，把 README 中的 Agent/客户端关键文档入口集中列出来，避免后续新增文档只在 roadmap 或 CHANGELOG 出现。

## 第二百零九阶段：release 前文档索引检查

1. [x] 新增 `test/release_doc_index_test.sh`，固定 README 中 Agent/客户端关键文档入口。
2. [x] 校验 README 必须链接 `agent-workflow`、`agent-contract`、`agent-action-guide`、`validate-coverage-task-samples`、`fixtures`、`client-integration` 和 `mcp-client-contract-tests`。
3. [x] 校验 README 必须保留 `go run ./examples/agent-decision-demo`、`go run ./examples/mcp-client-demo` 和 `scripts/verify-client-setup.sh` 三个接入/演示命令。
4. [x] CI 已纳入 release 文档索引检查，CHANGELOG 和 roadmap 已记录。

已完成补充：release 前文档入口不再只靠人工检查，README 的 Agent/客户端关键路径会被 CI 固定。下一步建议进入 v0.5.1 release readiness 收口，跑一轮版本前门禁并整理 release note 草案。

## 第二百一十阶段：v0.5.1 release readiness

1. [x] 新增 `docs/plan-release-notes-v0.5.1.md`，归纳 v0.5.0 之后的 MCP 客户端接入、Agent 结构化契约、真实 fixture 和 CI 回归保护。
2. [x] 新增 `docs/plan-release-v0.5.1.md`，记录候选发布资料、本地门禁、资产 dry-run 和正式发布待办。
3. [x] 完成本地发布前门禁：脚本语法、`go test ./...`、全部默认 shell 校验、主服务/testgen 构建、help 输出、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。
4. [x] 明确当前不切 `main.go` implementation version、不更新安装文档、不打 tag；这些保留到正式版本准备阶段。
5. [x] CHANGELOG 和 roadmap 已记录 v0.5.1 候选发布资料。

已完成补充：v0.5.1 候选发布资料已就绪，本地 release readiness 门禁通过。下一步建议进入正式版本准备：更新 implementation version、收敛 CHANGELOG、同步 README/installation 版本引用，然后提交后等待远端 CI，再决定是否打 `v0.5.1` tag。

## 第二百一十一阶段：v0.5.1 正式版本准备

1. [x] `main.go` MCP implementation version 已更新为 `0.5.1`。
2. [x] `CHANGELOG.md` 的 `Unreleased` 内容已收敛为 `v0.5.1 - 2026-07-17`，并记录 implementation version 更新。
3. [x] README 中当前 Release、手动下载示例和 Windows 下载示例已同步到 `v0.5.1`。
4. [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.5.1`。
5. [x] `docs/plan-release-notes-v0.5.1.md` 和 `docs/plan-release-v0.5.1.md` 已标记版本准备完成项。
6. [x] Homebrew Formula 暂不改 sha256；正式 Release Artifacts 生成后再通过真实 asset digest 更新 tap。

已完成补充：v0.5.1 版本准备改动已完成，远端 CI run `29591849021` 已通过。下一步可以打 `v0.5.1` tag，并进入 Release Artifacts / 资产校验 / GitHub Release 正文 / Homebrew tap 验证。

## 第二百一十二阶段：v0.5.1 正式发布核验

1. [x] 推送 `v0.5.1` tag。
2. [x] Release Artifacts workflow `29592283968` 已通过，Linux amd64、Linux arm64、macOS arm64、Windows amd64、Windows arm64 五个平台资产和 `.sha256` 已上传。
3. [x] 运行 `scripts/verify-release-assets.sh v0.5.1`，确认 10 个 Release 资产完整。
4. [x] 更新 GitHub Release 正文为正式 v0.5.1 发布说明。
5. [x] 仓库内 `Formula/testloop-mcp.rb` 已更新到 `0.5.1`，使用 Release API 返回的真实 digest。
6. [x] 更新 `sleticalboy/homebrew-tap` 到 `testloop-mcp 0.5.1`，提交 `54d6e7a` 并推送。
7. [x] 本机 Homebrew tap 已快进到 `54d6e7a`，并通过 `ruby -c` 与 `brew style`。
8. [x] Post-Release Verify run `29593507242` passed，资产清单和五平台安装脚本 dry run 全部通过。
9. [ ] `brew fetch` 下载验证受 GitHub/Homebrew 网络队列影响未完成；后续网络稳定后补跑安装级验证。

已完成补充：v0.5.1 已正式发布，版本重点是 MCP 客户端接入契约、真实 fixture、结构化分流和 release 文档入口的回归保护。Release Artifacts、资产清单和 Post-Release Verify 五平台安装 dry run 已通过；本机 `brew fetch` 仍受 GitHub/Homebrew 下载链路影响，后续网络稳定后再补跑。

## 第二百一十三阶段：安装后自检版本门禁

1. [x] 主二进制新增 `--version`，输出 `testloop-mcp <version>`。
2. [x] MCP implementation version 与 CLI version 共用同一个 `appVersion` 常量，减少发版时两个版本漂移的风险。
3. [x] `scripts/verify-client-setup.sh` 新增 `TESTLOOP_MCP_VERIFY_EXPECT_VERSION`，可在安装后自检时要求当前二进制匹配预期版本。
4. [x] 自检脚本测试覆盖版本匹配和版本不匹配失败提示。
5. [x] README、quickstart 和安装文档已记录版本门禁用法。

已完成补充：安装后自检现在不仅能确认二进制可执行、客户端配置可生成和 HTTP 探活可用，也能发现 PATH 指向旧版本的问题。下一步建议继续增强客户端可复制接入，优先补一个真实进程级 “stdio 配置片段 -> 启动客户端 -> 调用轻量工具” 的单命令验收脚本或文档入口。

## 第二百一十四阶段：真实进程级 MCP 客户端验收脚本

1. [x] 新增 `examples/mcp-process-smoke`，使用 MCP SDK 客户端连接真实 `testloop-mcp` 进程。
2. [x] stdio 路径通过 `CommandTransport` 启动指定二进制，执行 `tools/list` 和轻量 `parse_results`。
3. [x] Streamable HTTP 路径启动指定二进制、等待 `/healthz`，再通过 `StreamableClientTransport` 调用同一组轻量工具。
4. [x] smoke 会校验 `structuredContent` 存在，并与 `content[0].text` JSON fallback 语义一致。
5. [x] 新增 `scripts/verify-mcp-process-smoke.sh`，给用户一个安装后单命令验收入口，支持 `TESTLOOP_MCP_CLIENT_SMOKE_TRANSPORT=stdio|http|all`。
6. [x] 新增 `test/mcp_process_smoke_test.sh` 并纳入 CI，构建当前仓库临时二进制后跑真实进程级客户端 smoke。
7. [x] README、quickstart 和安装文档已加入该脚本入口。
8. [x] 最新 main CI run `29596235889` passed，确认新增真实进程级客户端 smoke、构建和 Docker 镜像构建均通过。

已完成补充：客户端接入验证现在从“配置可生成、HTTP healthz 可用”推进到“真实 MCP SDK 客户端能启动安装后二进制并调用工具”。下一步建议把这个 smoke 与 `verify-client-setup.sh` 的定位进一步整理到 quickstart：基础安装验收用 `verify-client-setup`，深度协议验收用 `verify-mcp-process-smoke`。

## 第二百一十五阶段：接入验收文档分层

1. [x] quickstart 的自检步骤拆成“基础安装验收”和“深度协议验收”。
2. [x] 明确 `verify-client-setup.sh` 用于二进制、版本、配置 roundtrip 和 `/healthz` 基础检查。
3. [x] 明确 `verify-mcp-process-smoke.sh` 用于真实 MCP SDK 客户端 stdio / Streamable HTTP 协议验收。
4. [x] README 和安装文档同步使用“基础安装验收 / 深度协议验收”的口径。
5. [x] `test/release_doc_index_test.sh` 已把 `scripts/verify-mcp-process-smoke.sh` 纳入 README 入口保护。

已完成补充：新用户现在能更清楚地区分两类验收：先用基础脚本确认安装和配置没问题，再用进程级 smoke 确认真实 MCP 协议链路可用。下一步建议沉淀一个公开展示案例，把这两类验收和 `examples/mcp-client-demo` 串成完整“安装 -> 验收 -> Agent 闭环”的演示路径。

## 第二百一十六阶段：安装到 Agent 闭环展示路径

1. [x] 新增 `scripts/showcase-onboarding.sh`，串联基础安装验收、真实 MCP 进程协议验收和最小 Agent 闭环 demo。
2. [x] 新增 `docs/showcase-onboarding.md`，说明一键演示命令、验证内容和适用边界。
3. [x] `docs/showcase.md` 已把 onboarding showcase 放到快速选择表中。
4. [x] README 已加入完整首次接入路径演示命令。
5. [x] `test/showcase_scripts_test.sh` 覆盖 onboarding showcase 的 bash 语法、help 输出和参数错误。
6. [x] `test/release_doc_index_test.sh` 已把 `scripts/showcase-onboarding.sh` 纳入 README 入口保护。

已完成补充：现在已经有一条不依赖外部项目的公开演示路径，可以从安装验收一路走到 Agent 反馈闭环。下一步建议选择公开 Go 或 JS/TS 项目，做真实项目级 showcase，展示覆盖率任务如何指导 Agent 做 ready、manual-review 和失败修复分流。

## 第二百一十七阶段：公开项目 showcase 决策信号断言

1. [x] `scripts/showcase-go-public-project.sh` 新增 `TESTLOOP_SHOWCASE_GO_EXPECT_ACTIONS`，默认断言 `go-test-1=ready`。
2. [x] `scripts/showcase-js-public-project.sh` 新增 `TESTLOOP_SHOWCASE_JS_EXPECT_ACTIONS`，默认断言 `vitest-1=manual_review_internal,vitest-2=ready`。
3. [x] 两个 showcase 会在输出 `showcase_summary=...` 后输出 `showcase_expectations=pass`；决策信号漂移时直接失败。
4. [x] Go / JS showcase 文档已记录默认期望、跳过断言方式和当前验证结果。
5. [x] `test/showcase_scripts_test.sh` 已校验 help 输出包含新的期望断言环境变量。

已完成补充：公开项目 showcase 现在不只是打印摘要，而是能作为手动验收门禁使用。下一步建议实际跑一次 Go showcase，如果网络和外部仓库可达，就把最新输出证据同步到文档；如果 Go 稳定，再跑 JS/TS showcase。

## 第二百一十八阶段：公开项目 showcase 本地 checkout 复用

1. [x] Go showcase 新增 `TESTLOOP_SHOWCASE_GO_PROJECT_DIR`，可复用已有本地 checkout，跳过 clone/fetch/checkout。
2. [x] JS showcase 新增 `TESTLOOP_SHOWCASE_JS_PROJECT_DIR`，可复用已有本地 checkout，跳过 clone/fetch/checkout。
3. [x] JS showcase 新增 `TESTLOOP_SHOWCASE_JS_SKIP_INSTALL=true`，可在依赖已准备好时跳过 `pnpm install`。
4. [x] Go / JS showcase 新增 `TESTLOOP_SHOWCASE_*_GIT_TIMEOUT`，远端 clone/fetch 默认 60 秒超时，避免 GitHub 网络不可达时长时间挂起。
5. [x] Go / JS showcase 文档已记录本地 checkout 复用方式和远端 git 超时控制。
6. [x] `test/showcase_scripts_test.sh` 已校验 help 输出、本地 checkout、跳过安装和 git timeout 失败路径。

已完成补充：公开项目 showcase 现在可以避开每次演示都重新 clone 外部仓库的问题，更适合网络不稳定时复验。已用本地 `/tmp/testloop-showcase-google-uuid` checkout 跑通 Go showcase，`go-test-1` 保持 `passed/ready`，并输出 `showcase_expectations=pass`。下一步建议准备或复用本地 `unjs/ufo` checkout 跑 JS/TS showcase。

已完成补充：当前环境再次验证 GitHub 远端连接仍不可用，`git ls-remote https://github.com/unjs/ufo.git f06c800d0c59f2a4a1b9ba65eb6cb61a84419be6` 在 75 秒后返回 `Couldn't connect to server`。公开 showcase 已新增 git 超时门禁，避免这类外部网络故障阻塞本地推进；JS/TS showcase 仍保留为网络或本地 checkout 准备好后的下一项。

## 第二百一十九阶段：公开 JS/TS showcase 远端复验

1. [x] 网络恢复后执行默认 JS/TS 公开 showcase，使用远端 clone/fetch 路径 checkout `unjs/ufo` 固定 commit `f06c800d0c59f2a4a1b9ba65eb6cb61a84419be6`。
2. [x] 默认任务 `vitest-1,vitest-2` 均通过验证，`status_counts={"passed":2}`。
3. [x] 默认 action 断言保持稳定：`vitest-1=manual_review_internal`、`vitest-2=ready`，脚本输出 `showcase_expectations=pass`。
4. [x] `docs/showcase-public-js.md` 已记录最新复验命令、输出文件和任务摘要。

已完成补充：公开项目 showcase 的 Go 和 JS/TS 两条路径都已有最新真实复验证据。下一步建议做一个 showcase 输出样例归档策略：只归档精简 summary 到文档，JSONL 明细继续保留在 `/tmp` 或用户指定路径，避免把外部项目生成结果大文件提交进仓库。

## 第二百二十阶段：showcase summary 归档策略

1. [x] 新增 `scripts/summarize-showcase-output.py`，统一从 JSONL 明细输出精简 `showcase_summary=...`。
2. [x] Go / JS 公开 showcase 脚本改为复用同一个 summary 脚本，避免 action 断言逻辑重复维护。
3. [x] 新增 `test/showcase_summary_test.sh`，固定 summary 输出、action 漂移失败和非法期望失败。
4. [x] CI 已纳入 showcase summary 测试。
5. [x] showcase 文档明确 JSONL 明细默认保留在 `/tmp` 或用户指定路径，仓库只归档精简 summary 和关键任务摘要。

已完成补充：公开 showcase 的证据归档方式已经收敛为“文档记录 summary，JSONL 作为本地明细制品”。下一步建议进入 v0.5.2 候选准备，先整理 Unreleased 内容和 release readiness checklist，再决定是否发 patch。

## 第二百二十一阶段：v0.5.2 release readiness

1. [x] 新增 `docs/plan-release-notes-v0.5.2.md`，归纳 v0.5.1 之后的安装验收、真实 MCP 进程 smoke、onboarding showcase 和公开 showcase 收敛内容。
2. [x] 新增 `docs/plan-release-v0.5.2.md`，记录候选发布资料、本地门禁和正式发布待办。
3. [x] 完成本地发布前门禁：脚本语法、`go test ./...`、全部默认 shell 校验、主服务/testgen 构建、help 输出、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。
4. [x] 明确当前不切 `main.go` implementation version、不更新安装文档、不打 tag；这些保留到正式版本准备阶段。
5. [x] CHANGELOG 和 roadmap 已记录 v0.5.2 候选发布资料。

已完成补充：v0.5.2 候选发布资料已就绪，本地 release readiness 门禁通过。下一步建议进入正式版本准备：更新 implementation version、收敛 CHANGELOG、同步 README/installation 版本引用，然后提交后等待远端 CI，再决定是否打 `v0.5.2` tag。

## 第二百二十二阶段：v0.5.2 正式版本准备

1. [x] `main.go` MCP implementation version 已更新为 `0.5.2`。
2. [x] `CHANGELOG.md` 的 `Unreleased` 内容已收敛为 `v0.5.2 - 2026-07-18`，并记录 implementation version 更新。
3. [x] README 中当前 Release、手动下载示例和 Windows 下载示例已同步到 `v0.5.2`。
4. [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.5.2`。
5. [x] quickstart 和 onboarding showcase 中的 `TESTLOOP_MCP_VERIFY_EXPECT_VERSION` 已同步到 `0.5.2`。
6. [x] Homebrew Formula 暂不改 sha256；正式 Release Artifacts 生成后再通过真实 asset digest 更新 tap。

已完成补充：v0.5.2 版本准备改动已完成。下一步应提交后等待远端 CI，通过后打 `v0.5.2` tag，并进入 Release Artifacts / 资产校验 / GitHub Release 正文 / Homebrew tap 验证。

## 第二百二十三阶段：v0.5.2 正式发布核验

1. [x] 版本准备提交远端 CI run `29629563807` passed。
2. [x] 推送 `v0.5.2` tag。
3. [x] Release Artifacts workflow `29629630932` 已通过，Linux amd64、Linux arm64、macOS arm64、Windows amd64、Windows arm64 五个平台资产和 `.sha256` 已上传。
4. [x] 运行 `scripts/verify-release-assets.sh v0.5.2`，确认 10 个 Release 资产完整。
5. [x] 更新 GitHub Release 正文为正式 v0.5.2 发布说明。
6. [x] 仓库内 `Formula/testloop-mcp.rb` 已更新到 `0.5.2`，使用 Release API 返回的真实 digest。
7. [x] 更新 `sleticalboy/homebrew-tap` 到 `testloop-mcp 0.5.2`，提交 `c1945e8` 并推送。
8. [x] 本机 Homebrew tap 已快进到 `c1945e8`，并通过 `ruby -c`、`brew style` 和 `brew fetch`。
9. [x] Post-Release Verify run `29629793877` passed，资产清单和五平台安装脚本 dry run 全部通过。

已完成补充：v0.5.2 已正式发布，版本重点是安装后验收、真实 MCP 协议 smoke 和公开 showcase 决策断言。与 v0.5.1 不同，本轮本机 `brew fetch` 也已成功。下一步建议回到产品主线，优先做一个“用户项目一键验收报告”入口，把基础验收、协议 smoke、公开 showcase 和本机项目 smoke 的结果聚合成可复制 Markdown。

## 第二百二十四阶段：用户项目一键验收报告

1. [x] 新增 `scripts/generate-verification-report.sh`，输出 Markdown 验收报告。
2. [x] 默认聚合基础安装验收、真实 MCP 协议 smoke 和最小 Agent 闭环 demo。
3. [x] 公开 Go / JS showcase 通过 `TESTLOOP_REPORT_PUBLIC_SHOWCASES=go|js|all` 显式 opt-in，默认不访问公网。
4. [x] 用户项目 smoke 通过 `TESTLOOP_REPORT_PROJECT_DIR` 和 `TESTLOOP_REPORT_PROJECT_COMMAND` 显式传入，支持 server / web / CLI 等不同仓库命令。
5. [x] 新增 `docs/verification-report.md`，README 和 `docs/showcase.md` 已补入口。
6. [x] 新增 `test/verification_report_test.sh` 并纳入默认 CI，固定成功、失败和 skipped 报告行为。

已完成补充：接入方现在可以用一个命令留下完整验收报告，不需要在聊天记录里零散复制多个 smoke 输出。下一步建议做“真实用户项目验收样例”：对 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/server` 跑一份 Go server 报告，再对 web 侧选择一个可离线执行的 Vue 命令，验证报告在多项目场景下是否足够清晰。

## 第二百二十五阶段：真实用户项目验收样例

1. [x] 用 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server` 跑通验收报告，用户项目 smoke 命令为 `go test ./...`。
2. [x] 用 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-web` 跑通验收报告，用户项目 smoke 命令为 `pnpm install --frozen-lockfile && pnpm build:prod`。
3. [x] 两份报告的基础安装验收、真实 MCP 协议 smoke、最小 Agent 闭环 demo 和用户项目 smoke 均为 `passed`。
4. [x] 文档记录 server 侧 macOS deprecated warning 和 web 侧 browserslist / bundle size warning，明确这些是 warning 不是失败。
5. [x] `docs/verification-report.md` 已补真实项目 smoke 记录，报告明细仍保留在 `/tmp`，不提交生成制品。

已完成补充：验收报告脚本已经在 Go server 和 Vue web 两类真实项目上跑通，证明它不仅能做 testloop-mcp 自检，也能把用户项目命令纳入同一份报告。下一步建议增强报告可读性：在脚本里增加一个机器可读 summary JSON 输出，方便 Agent 或 CI 不解析 Markdown 就能拿到每个 section 的 `status/exit_code`。

## 第二百二十六阶段：验收报告机器可读 summary

1. [x] `scripts/generate-verification-report.sh` 新增 `TESTLOOP_REPORT_SUMMARY_JSON`，可在 Markdown 报告之外写出 summary JSON。
2. [x] JSON 包含 `overall_status`、`failed_count`、报告元数据和 `sections[]` 的 `name/status/exit_code/reason`。
3. [x] skipped section 的 `exit_code` 输出为 `null`，失败 section 保留真实 exit code。
4. [x] `test/verification_report_test.sh` 已覆盖成功、失败、skipped 三类 JSON 输出。
5. [x] `docs/verification-report.md` 已补 “Markdown 给人看，JSON 给 Agent/CI 看” 的使用说明。

已完成补充：验收报告现在既有适合人工转发的 Markdown，也有适合 Agent/CI 分流的 JSON。下一步建议把这份 JSON 输出接入一个最小 Agent 决策示例：读取 summary JSON 后判断是安装问题、协议问题、Agent demo 问题，还是用户项目 smoke 问题，并输出下一步动作。

## 第二百二十七阶段：验收 summary Agent 决策示例

1. [x] 新增 `examples/verification-summary-decision-demo`，读取 `TESTLOOP_REPORT_SUMMARY_JSON` 产出的 summary JSON。
2. [x] `overall_status=passed` 时输出 `agent_next_step=ready`。
3. [x] 失败 section 按安装、MCP 协议、Agent demo、公开 showcase、用户项目 smoke 分流到不同 action。
4. [x] 新增 `test/verification_summary_decision_demo_test.sh`，覆盖整体通过、用户项目失败和 MCP 协议失败。
5. [x] README、`docs/verification-report.md` 和 release 文档索引已补决策 demo 入口。

已完成补充：验收报告的闭环已经从“生成 Markdown 给人看”推进到“生成 JSON 给 Agent/CI 分流”。下一步建议把这条链路封装成 CI 示例片段：展示如何在 GitHub Actions 中生成 Markdown + JSON，并在 summary JSON 失败时上传报告制品，方便接入方直接复制。

## 第二百二十八阶段：验收报告 CI 集成示例

1. [x] 新增 `docs/verification-ci.md`，提供 GitHub Actions 中生成 Markdown + JSON 验收报告的可复制示例。
2. [x] 示例在验收失败时仍运行 `go run ./examples/verification-summary-decision-demo`，输出 `agent_next_step`。
3. [x] 示例使用 `actions/upload-artifact@v4` 和 `if: always()` 上传 Markdown / JSON 报告制品。
4. [x] 文档覆盖 Go 项目 smoke 和 pnpm 前端项目 smoke 两类命令形态。
5. [x] 新增 `test/verification_ci_doc_test.sh` 并纳入 CI，固定关键环境变量、命令、artifact 路径和决策 demo 入口。
6. [x] README、`docs/verification-report.md` 和 release 文档索引已补 CI 集成入口。

已完成补充：接入方现在可以从本地验收、真实项目 smoke、summary JSON 到 GitHub Actions artifact 形成完整复制路径。下一步建议进入 v0.5.3 候选收敛：整理 Unreleased、跑本地发布前门禁，判断这些验收报告能力是否值得发 patch 版本。

## 第二百二十九阶段：v0.5.3 release readiness

1. [x] 新增 `docs/plan-release-notes-v0.5.3.md`，归纳 v0.5.2 之后的验收报告、summary JSON、Agent/CI 决策示例和 CI 集成内容。
2. [x] 新增 `docs/plan-release-v0.5.3.md`，记录候选发布资料、本地门禁和正式发布待办。
3. [x] 完成本地发布前门禁：脚本语法、`go test ./...`、全部默认 shell 校验、主服务/testgen 构建、help 输出、验收报告 Markdown + JSON、summary 决策 demo、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。
4. [x] 明确当前不切 `main.go` implementation version、不更新安装文档当前 release、不打 tag；这些保留到正式版本准备阶段。
5. [x] CHANGELOG 和 roadmap 已记录 v0.5.3 候选发布资料。

已完成补充：v0.5.3 候选发布资料已就绪，本地 release readiness 门禁通过。下一步建议进入正式版本准备：更新 implementation version、收敛 CHANGELOG、同步 README/installation/quickstart/onboarding/verification report 版本引用，然后提交后等待远端 CI，再决定是否打 `v0.5.3` tag。

## 第二百三十阶段：v0.5.3 正式版本准备

1. [x] `main.go` MCP implementation version 已更新为 `0.5.3`。
2. [x] `CHANGELOG.md` 的 `Unreleased` 内容已收敛为 `v0.5.3 - 2026-07-18`，并记录 implementation version 更新。
3. [x] README 中当前 Release、手动下载示例和 Windows 下载示例已同步到 `v0.5.3`。
4. [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.5.3`。
5. [x] quickstart、onboarding、verification report 和 verification CI 示例中的版本门禁已同步到 `0.5.3`。
6. [x] 测试中的版本期望已同步到 `0.5.3`。
7. [x] Homebrew Formula 暂不改 sha256；正式 Release Artifacts 生成后再通过真实 asset digest 更新 tap。
8. [x] 正式版本准备后已重新运行完整本地验证：脚本语法、`go test ./...`、默认 shell 矩阵、主服务/testgen 构建、验收报告 Markdown + JSON、summary 决策 demo、打包 dry-run、sha256 校验和 tarball 内容检查。

已完成补充：v0.5.3 版本准备改动已完成。下一步应提交后等待远端 CI，通过后打 `v0.5.3` tag，并进入 Release Artifacts / 资产校验 / GitHub Release 正文 / Homebrew tap 验证。

## 第二百三十一阶段：v0.5.3 正式发布核验

1. [x] 版本准备提交远端 CI run `29635368963` passed。
2. [x] 推送 `v0.5.3` tag。
3. [x] Release Artifacts workflow `29635462891` 已通过，Linux amd64、Linux arm64、macOS arm64、Windows amd64、Windows arm64 五个平台资产和 `.sha256` 已上传。
4. [x] 运行 `scripts/verify-release-assets.sh v0.5.3`，确认 10 个 Release 资产完整。
5. [x] 更新 GitHub Release 正文为正式 v0.5.3 发布说明。
6. [x] 仓库内 `Formula/testloop-mcp.rb` 已更新到 `0.5.3`，使用 Release API 返回的真实 digest。
7. [x] 更新 `sleticalboy/homebrew-tap` 到 `testloop-mcp 0.5.3`，提交 `b099aba` 并推送。
8. [x] 本机 Homebrew tap 已快进到 `b099aba`，并通过 `brew fetch` 获取 `0.5.3`。
9. [x] Post-Release Verify run `29635745094` passed，资产清单和五平台安装脚本 dry run 全部通过。
10. [x] 发布收尾提交 `a538e5d` 的远端 CI run `29636125961` passed。

已完成补充：v0.5.3 已正式发布，版本重点是验收报告、summary JSON、Agent/CI 决策示例和 CI 集成文档。下一步建议进入 v0.5.4 主线：把公开可复现 onboarding demo 从多个命令收敛成一个可产出 Markdown、JSON 和 `agent_next_step` 的演示入口。

## 第二百三十二阶段：v0.5.4 Agent onboarding demo 收敛

1. [x] 新增 `docs/plan-agent-onboarding-v0.5.4.md`，明确 v0.5.4 先做公开 onboarding demo 收敛，不扩语言。
2. [x] 新增 `scripts/showcase-agent-onboarding-report.sh`，一键运行验收报告、summary JSON 和 Agent 决策 demo。
3. [x] 脚本默认输出 `/tmp/testloop-mcp-onboarding/verification-report.md`、`verification-summary.json` 和 `agent-decision.txt`。
4. [x] 脚本支持复用 `TESTLOOP_MCP_VERIFY_EXPECT_VERSION` 做版本门禁，并透传 `TESTLOOP_REPORT_*` 选项给验收报告脚本。
5. [x] 新增 `test/showcase_agent_onboarding_report_test.sh` 并纳入 CI，固定 artifact 路径、summary JSON 和 `agent_next_step=ready` 输出。
6. [x] README、quickstart、`docs/showcase.md`、`docs/showcase-onboarding.md` 和 release 文档索引已补新入口。
7. [x] 使用当前源码构建的真实二进制 `/tmp/testloop-mcp-onboarding-demo` 跑通完整 wrapper，summary JSON 为 `overall_status=passed`、`failed_count=0`，decision 输出 `agent_next_step=ready`。
8. [x] 新增 `docs/verification-summary-failures.md` 和 `docs/fixtures/verification-summary/*.json`，展示五类验收失败的 `agent_next_step` 分流。
9. [x] 新增 `test/verification_summary_failure_fixtures_test.sh` 并纳入 CI，逐个 fixture 运行 decision demo 并校验 action。
10. [x] `docs/verification-ci.md` 已把 `scripts/showcase-agent-onboarding-report.sh` 作为推荐 workflow，底层 `generate-verification-report.sh` 保留为高级用法。

已完成补充：公开 onboarding demo 现在不只是在终端串跑三个命令，还能留下 Markdown、summary JSON 和 Agent decision 三类制品；失败样例也已固定为可执行 fixture，CI 示例也已优先推荐 wrapper。下一步建议进入 v0.5.4 候选收敛：补发布说明草案并跑一次真实安装二进制的 onboarding report。

## 第二百三十三阶段：v0.5.4 release readiness

1. [x] 新增 `docs/plan-release-notes-v0.5.4.md`，归纳 v0.5.3 之后的 Agent onboarding report、失败分流 fixture 和 CI 简化示例。
2. [x] 新增 `docs/plan-release-v0.5.4.md`，记录候选发布资料、本地门禁和正式发布待办。
3. [x] 完成本地发布前门禁：脚本语法、`go test ./...`、全部默认 shell 校验、主服务/testgen 构建、help 输出、onboarding report wrapper、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。
4. [x] 明确当前不切 `main.go` implementation version、不更新安装文档当前 release、不打 tag；这些保留到正式版本准备阶段。
5. [x] CHANGELOG 和 roadmap 已记录 v0.5.4 候选发布资料。

已完成补充：v0.5.4 候选发布资料已就绪，本地 release readiness 门禁通过。下一步建议提交候选文档并等待远端 CI，通过后进入正式版本准备：更新 implementation version、收敛 CHANGELOG、同步 README/installation/quickstart/onboarding/verification report/verification CI 版本引用，然后决定是否打 `v0.5.4` tag。

## 第二百三十四阶段：v0.5.4 正式版本准备

1. [x] `main.go` MCP implementation version 已更新为 `0.5.4`。
2. [x] `CHANGELOG.md` 的 `Unreleased` 内容已收敛为 `v0.5.4 - 2026-07-18`，并记录 implementation version 更新。
3. [x] README 中当前 Release、手动下载示例和 Windows 下载示例已同步到 `v0.5.4`。
4. [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.5.4`。
5. [x] quickstart、onboarding、verification report 和 verification CI 示例中的版本门禁已同步到 `0.5.4`。
6. [x] 测试中的版本期望已同步到 `0.5.4`。
7. [x] Homebrew Formula 暂不改 sha256；正式 Release Artifacts 生成后再通过真实 asset digest 更新 tap。
8. [x] 正式版本准备后重新运行完整本地验证：脚本语法、`go test ./...`、默认 shell 矩阵、主服务/testgen 构建、onboarding report wrapper、打包 dry-run、sha256 校验和 tarball 内容检查。

已完成补充：v0.5.4 版本准备改动已完成。下一步应提交后等待远端 CI，通过后打 `v0.5.4` tag，并进入 Release Artifacts / 资产校验 / GitHub Release 正文 / Homebrew tap 验证。

## 第二百三十五阶段：v0.5.4 正式发布核验

1. [x] 版本准备提交远端 CI run `29638973367` passed。
2. [x] 推送 `v0.5.4` tag。
3. [x] Release Artifacts workflow `29639038941` 已通过，Linux amd64、Linux arm64、macOS arm64、Windows amd64、Windows arm64 五个平台资产和 `.sha256` 已上传。
4. [x] 运行 `scripts/verify-release-assets.sh v0.5.4`，确认 10 个 Release 资产完整。
5. [x] 更新 GitHub Release 正文为正式 v0.5.4 发布说明。
6. [x] 仓库内 `Formula/testloop-mcp.rb` 已更新到 `0.5.4`，使用 Release API 返回的真实 digest。
7. [x] 更新 `sleticalboy/homebrew-tap` 到 `testloop-mcp 0.5.4`，提交 `00b56f2` 并推送。
8. [x] 本机 Homebrew tap 已快进到 `00b56f2`，并通过 `brew fetch` 获取 `0.5.4`。
9. [x] Post-Release Verify run `29639243485` passed，资产清单和五平台安装脚本 dry run 全部通过。

已完成补充：v0.5.4 已正式发布，版本重点是公开 onboarding report、验收 summary 失败分流 fixture 和更短的 CI 复制路径。下一步建议回到产品主线，优先做接入方真实项目案例模板，把 server/web smoke、summary JSON 和 `agent_next_step` 结果整理成一份可复用案例。

## 第二百三十六阶段：真实接入案例模板

1. [x] 使用当前源码构建 `/tmp/testloop-mcp-v0.5.4-case`，确认 `--version` 输出 `testloop-mcp 0.5.4`，避免本机 Homebrew 仍停留在 `0.5.0` 时污染案例结论。
2. [x] 用 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server` 跑通 onboarding report，用户项目命令为 `go test ./...`。
3. [x] 用 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-web` 跑通 onboarding report，用户项目命令为 `pnpm install --frozen-lockfile && pnpm build:prod`。
4. [x] 两个样例的 summary JSON 均为 `overall_status=passed`、`failed_count=0`，decision 输出均为 `agent_next_step=ready`。
5. [x] 新增 `docs/real-integration-cases.md`，沉淀真实项目接入模板、结果判断表和 laoxia server/web 实跑记录。
6. [x] 新增 `test/real_integration_cases_doc_test.sh` 并纳入 CI，固定模板关键命令、环境变量、样例路径和决策字段。
7. [x] README、`docs/showcase.md` 和 CHANGELOG 已补真实接入案例入口。

已完成补充：真实接入文档现在能把“这个 MCP 工具怎么在真实 server/web 项目里落地”讲清楚，并给 Agent/CI 明确的 JSON 与 `agent_next_step` 消费路径。下一步建议处理安装漂移问题：本机 Homebrew 当前仍是 `0.5.0`，但 README/脚本已经要求 `--version`；应补一个安装后升级/重装验收路径，确保公开安装命令拿到的二进制真的是 `0.5.4`。

## 第二百三十七阶段：安装漂移诊断与升级提示

1. [x] 复查本机状态：`brew info testloop-mcp` 显示 tap stable 为 `0.5.4`，但 installed/linked 仍为 `0.5.0`。
2. [x] 增强 `scripts/verify-client-setup.sh`：旧二进制缺少 `--version` 时不再裸退出，而是输出原始版本命令输出和 Homebrew 升级/重装建议。
3. [x] 版本不匹配时复用同一诊断提示，方便用户从 `TESTLOOP_MCP_VERIFY_EXPECT_VERSION` 失败直接知道下一步命令。
4. [x] 扩充 `test/verify_client_setup_test.sh`，覆盖旧二进制 `flag provided but not defined: -version` 和版本不匹配提示。
5. [x] `docs/installation.md` 与 `docs/quickstart.md` 已补 `testloop-mcp --version` 验证、`brew upgrade` 和 `brew reinstall` 路径。
6. [x] 本机执行 `HOMEBREW_NO_AUTO_UPDATE=1 brew upgrade sleticalboy/tap/testloop-mcp`，实际从 `0.5.0` 升级到 `0.5.4`。
7. [x] 使用真实安装二进制 `/opt/homebrew/bin/testloop-mcp` 跑通 `TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.4 scripts/verify-client-setup.sh "$(command -v testloop-mcp)"`。
8. [x] 使用真实安装二进制跑通 `scripts/showcase-agent-onboarding-report.sh "$(command -v testloop-mcp)"`，输出目录 `/tmp/testloop-installed-onboarding-v0.5.4`，summary JSON 为 `overall_status=passed`、`failed_count=0`，decision 输出 `agent_next_step=ready`。

已完成补充：安装漂移现在可以被基础自检脚本明确识别并给出下一步操作，不会再让用户只看到 Go flag usage；本机 Homebrew 安装也已升到 `0.5.4`，并通过真实版本门禁和安装态 onboarding wrapper。下一步建议进入 v0.5.5 候选收敛：整理本轮真实接入案例、安装漂移诊断、Homebrew 安装态验收证据，并跑发布前门禁，判断是否作为 patch 版本发布。

## 第二百三十八阶段：v0.5.5 release readiness

1. [x] 新增 `docs/plan-release-notes-v0.5.5.md`，归纳 v0.5.4 之后的真实接入案例、安装漂移诊断和 Homebrew 安装态验收。
2. [x] 新增 `docs/plan-release-v0.5.5.md`，记录候选发布资料、本地门禁和正式发布待办。
3. [x] 完成本地发布前门禁：脚本语法、`go test ./...`、全部默认 shell 校验、主服务/testgen 构建、help 输出、真实安装态 onboarding wrapper、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。
4. [x] 明确当前不切 `main.go` implementation version、不更新安装文档当前 release、不打 tag；这些保留到正式版本准备阶段。
5. [x] CHANGELOG 和 roadmap 已记录 v0.5.5 候选发布资料。

已完成补充：v0.5.5 候选发布资料已就绪，本地 release readiness 门禁通过。下一步建议提交候选文档并等待远端 CI，通过后进入正式版本准备：更新 implementation version、收敛 CHANGELOG、同步 README/installation/quickstart/real integration cases/verification 文档版本引用，然后决定是否打 `v0.5.5` tag。

## 第二百三十九阶段：v0.5.5 正式版本准备

1. [x] `main.go` MCP implementation version 已更新为 `0.5.5`。
2. [x] `CHANGELOG.md` 的 `Unreleased` 内容已收敛为 `v0.5.5 - 2026-07-18`，并记录 implementation version 更新。
3. [x] README 中当前 Release、手动下载示例和 Windows 下载示例已同步到 `v0.5.5`。
4. [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.5.5`。
5. [x] quickstart、onboarding、real integration cases、verification report 和 verification CI 示例中的版本门禁已同步到 `0.5.5`；laoxia 历史实跑记录保留 `0.5.4` 证据。
6. [x] 测试中的版本期望已同步到 `0.5.5`。
7. [x] Homebrew Formula 暂不改 sha256；正式 Release Artifacts 生成后再通过真实 asset digest 更新 tap。
8. [x] 正式版本准备后重新运行完整本地验证：脚本语法、`go test ./...`、默认 shell 矩阵、主服务/testgen 构建、v0.5.5 准备二进制 onboarding wrapper、打包 dry-run、sha256 校验和 tarball 内容检查。

已完成补充：v0.5.5 版本准备改动已完成。下一步应提交后等待远端 CI，通过后打 `v0.5.5` tag，并进入 Release Artifacts / 资产校验 / GitHub Release 正文 / Homebrew tap 验证。

## 第二百四十阶段：v0.5.5 正式发布核验

1. [x] 版本准备提交远端 CI run `29644261865` passed。
2. [x] 推送 `v0.5.5` tag。
3. [x] Release Artifacts workflow `29644340675` 已通过，Linux amd64、Linux arm64、macOS arm64、Windows amd64、Windows arm64 五个平台资产和 `.sha256` 已上传。
4. [x] 运行 `scripts/verify-release-assets.sh v0.5.5`，确认 10 个 Release 资产完整。
5. [x] 更新 GitHub Release 正文为正式 v0.5.5 发布说明。
6. [x] 仓库内 `Formula/testloop-mcp.rb` 已更新到 `0.5.5`，使用 Release API 返回的真实 digest。
7. [x] 更新 `sleticalboy/homebrew-tap` 到 `testloop-mcp 0.5.5`，提交 `e945158` 并推送。
8. [x] 本机 Homebrew tap 已快进到 `e945158`，并通过 `brew fetch` 获取 `0.5.5`。
9. [x] Post-Release Verify run `29644550322` passed，资产清单和五平台安装脚本 dry run 全部通过。

已完成补充：v0.5.5 已正式发布，版本重点是真实接入案例模板、安装漂移诊断和 Homebrew 安装态验收。下一步建议回到产品主线，优先做接入方“复制即用”的 onboarding CI 模板，把真实项目 smoke、summary JSON、`agent_next_step` 和 artifact 上传路径收敛成一份最小可粘贴 workflow。

## 第二百四十一阶段：Onboarding CI 复制模板

1. [x] 新增 `docs/onboarding-ci-template.md`，面向首次接入方提供一屏可复制的 GitHub Actions workflow。
2. [x] 模板覆盖 Go server 和 Vue / Node 两类项目，分别固定 `go test ./...` 与 `pnpm install --frozen-lockfile && pnpm build` smoke 命令。
3. [x] 模板统一上传 `verification-report.md`、`verification-summary.json` 和 `agent-decision.txt` 三类 artifact。
4. [x] 新增 `test/onboarding_ci_template_doc_test.sh`，固定安装脚本、版本门禁、输出目录、项目 smoke 命令和 artifact 路径。
5. [x] README、showcase、验收 CI 文档、CHANGELOG 和文档索引测试已补入口。

已完成补充：接入方现在可以先复制最小 onboarding CI 模板，跑出稳定 artifact 后再阅读完整验收报告文档。下一步建议增强这条模板的可执行性：新增一个仓库内最小 workflow fixture 或脚本校验，确保 YAML 片段持续可解析，避免文档示例随手改坏。

## 第二百四十二阶段：Onboarding CI 模板 YAML 可解析性

1. [x] 新增 `test/onboarding_ci_template_yaml_test.sh`，从 `docs/onboarding-ci-template.md` 抽取完整 `yaml` fenced block。
2. [x] 测试要求文档中恰好保留 Go server 与 Vue / Node 两个完整 workflow 示例。
3. [x] 使用 Ruby 标准库 `yaml` 解析每个 workflow，校验 `name`、`on` 和 `jobs.onboarding` 等关键结构存在。
4. [x] `.github/workflows/ci.yml` 已纳入该测试，避免远端 CI 漏掉文档示例语法漂移。
5. [x] CHANGELOG 已记录模板 YAML 可解析性回归测试。

已完成补充：Onboarding CI 模板现在不只是字符串片段校验，还会做 workflow YAML 语法和关键结构校验。下一步建议继续把“复制即用”推进一步：把模板中的安装和报告生成命令抽成可运行脚本片段，减少用户在 CI 里手写环境变量的机会。

## 第二百四十三阶段：Onboarding CI bootstrap 脚本

1. [x] 新增 `scripts/run-onboarding-ci.sh`，把外部用户项目 CI 中的二进制安装/解析、testloop-mcp helper checkout、用户项目 smoke 和 onboarding artifact 生成收敛成一个入口。
2. [x] 脚本支持 `TESTLOOP_ONBOARDING_PROJECT_DIR`、`TESTLOOP_ONBOARDING_PROJECT_COMMAND`、`TESTLOOP_ONBOARDING_OUTPUT_DIR`、`TESTLOOP_MCP_VERSION`、`TESTLOOP_MCP_COMMAND` 和 `TESTLOOP_MCP_REPO_DIR`。
3. [x] 新增 `test/run_onboarding_ci_test.sh`，用 fake binary 和跳过项验证脚本输出 Markdown、summary JSON 和 `agent_next_step=ready`。
4. [x] Onboarding CI 复制模板和验收 CI 文档已改用 `run-onboarding-ci.sh`，避免外部用户仓库直接调用不存在的 repo-local showcase 脚本。
5. [x] 文档测试和 YAML 测试已固定 bootstrap 命令，防止模板退回不可复制路径。

已完成补充：外部用户现在可以在 CI 里下载一个 bootstrap 脚本并传入 smoke 命令，不需要理解内部 `TESTLOOP_REPORT_*` 变量。下一步建议对 bootstrap 入口补一份真实本地 dry-run 记录：用当前仓库或 laoxia server 跑一次不跳过的 `run-onboarding-ci.sh`，确认真实安装/协议/Agent demo/项目 smoke 全链路仍能通过。

## 第二百四十四阶段：Onboarding CI bootstrap 真实 dry-run

1. [x] 使用当前仓库作为用户项目，运行 `scripts/run-onboarding-ci.sh 'go test ./...'`。
2. [x] 设置 `TESTLOOP_MCP_VERSION=v0.5.5` 和临时 `TESTLOOP_MCP_INSTALL_DIR=/tmp/testloop-run-onboarding-ci-bin`，确认 bootstrap 不复用 PATH 上的旧 Homebrew 二进制。
3. [x] 输出目录为 `/tmp/testloop-run-onboarding-ci-v0.5.5`，生成 `verification-report.md`、`verification-summary.json` 和 `agent-decision.txt`。
4. [x] summary JSON 为 `overall_status=passed`、`failed_count=0`。
5. [x] 基础安装验收、真实 MCP 协议 smoke、最小 Agent 闭环 demo 和用户项目 smoke 均为 `passed`，公开 showcase 按默认策略 `skipped`。
6. [x] decision 输出 `agent_next_step=ready`。

已完成补充：bootstrap 脚本已经通过真实安装态 v0.5.5 二进制完成全链路 dry-run。下一步建议补一层失败路径文档：当 bootstrap 因安装、版本、项目 smoke 失败时，接入方应该优先下载哪些 artifact、看哪些字段，以及如何把失败粘给 AI Agent 继续修。

## 第二百四十五阶段：Onboarding CI 失败路径排查

1. [x] `scripts/run-onboarding-ci.sh` 在 `GITHUB_STEP_SUMMARY` 存在时会写入 CI step summary。
2. [x] step summary 包含 `Status`、`Failed sections`、`agent_next_step`、Markdown report、summary JSON 和 agent decision 路径。
3. [x] `test/run_onboarding_ci_test.sh` 已覆盖成功路径和用户项目 smoke 失败路径的 step summary 输出。
4. [x] 新增 `docs/onboarding-ci-failure-triage.md`，说明失败时先看 step summary，再看 `agent-decision.txt`、`verification-summary.json` 和 `verification-report.md`。
5. [x] 新增 `test/onboarding_ci_failure_triage_doc_test.sh`，固定失败分流 action、artifact 文件名和 AI Agent 粘贴上下文。
6. [x] README、showcase、验收 CI 文档和文档索引测试已补失败排查入口。

已完成补充：Onboarding CI 现在在成功和失败时都有稳定可读的 GitHub Actions 摘要，接入方可以直接按 `agent_next_step` 分流，不必先翻完整日志。下一步建议做一次 v0.5.6 候选收敛：整理本轮 onboarding CI bootstrap、YAML 可解析性、失败排查和真实 dry-run 证据，跑 release readiness，判断是否发 patch 版本。

## 第二百四十六阶段：v0.5.6 release readiness

1. [x] 新增 `docs/plan-release-notes-v0.5.6.md`，归纳 v0.5.5 之后的 Onboarding CI 复制模板、bootstrap、YAML 可解析性、step summary 和失败排查内容。
2. [x] 新增 `docs/plan-release-v0.5.6.md`，记录候选发布资料、本地门禁和正式发布待办。
3. [x] 完成本地发布前门禁：脚本语法、`go test ./...`、全部默认 shell 校验、主服务/testgen 构建、bootstrap 真实 dry-run、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。
4. [x] 明确当前不切 `main.go` implementation version、不更新安装文档当前 release、不打 tag；这些保留到正式版本准备阶段。
5. [x] CHANGELOG 和 roadmap 已记录 v0.5.6 候选发布资料。

已完成补充：v0.5.6 候选发布资料已就绪，本地 release readiness 门禁通过。下一步提交候选文档并等待远端 CI，通过后进入正式版本准备：更新 implementation version、收敛 CHANGELOG、同步 README/installation/quickstart/onboarding/verification 文档版本引用，然后决定是否打 `v0.5.6` tag。

## 第二百四十七阶段：v0.5.6 正式版本准备

1. [x] `main.go` MCP implementation version 已更新为 `0.5.6`。
2. [x] `CHANGELOG.md` 的 `Unreleased` 内容已收敛为 `v0.5.6 - 2026-07-18`，并记录 implementation version 更新。
3. [x] README 中当前 Release、手动下载示例和 Windows 下载示例已同步到 `v0.5.6`。
4. [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.5.6`。
5. [x] quickstart、onboarding、real integration cases、verification report、verification CI 和 onboarding CI 模板中的版本门禁已同步到 `0.5.6`。
6. [x] 测试中的版本期望已同步到 `0.5.6`。
7. [x] Homebrew Formula 暂不改 sha256；正式 Release Artifacts 生成后再通过真实 asset digest 更新 tap。
8. [x] 正式版本准备后重新运行完整本地验证：脚本语法、`go test ./...`、默认 shell 矩阵、主服务/testgen 构建、v0.5.6 准备二进制 bootstrap dry-run、打包 dry-run、sha256 校验和 tarball 内容检查。

已完成补充：v0.5.6 版本准备改动已完成，本地完整门禁通过。下一步提交后等待远端 CI，通过后打 `v0.5.6` tag，并进入 Release Artifacts / 资产校验 / GitHub Release 正文 / Homebrew tap 验证。

## 第二百四十八阶段：v0.5.6 正式发布核验

1. [x] 版本准备提交远端 CI run `29648677800` passed。
2. [x] 推送 `v0.5.6` tag，指向 `4ba3ef1cc7fc605463fb31ba4e4f4c18f8f43885`。
3. [x] Release Artifacts workflow run `29648755666` passed，Linux amd64、Linux arm64、macOS arm64、Windows amd64、Windows arm64 五个平台资产和 `.sha256` 已上传。
4. [x] 运行 `scripts/verify-release-assets.sh v0.5.6`，确认 10 个 Release 资产完整。
5. [x] 更新 GitHub Release 正文为正式 v0.5.6 发布说明。
6. [x] 仓库内 `Formula/testloop-mcp.rb` 已更新到 `0.5.6`，使用 Release API 返回的真实 digest。
7. [x] 更新 `sleticalboy/homebrew-tap` 到 `testloop-mcp 0.5.6`，提交 `000a417` 并推送。
8. [x] 本机 Homebrew tap 已快进到 `000a417420e03c8dc278de28bce6d318e4880a1b`，`brew info --json=v2 sleticalboy/tap/testloop-mcp` 返回 `version=0.5.6`。
9. [x] Post-Release Verify run `29648990368` passed，资产清单和五平台安装脚本 dry run 全部通过。
10. [ ] 本机 `brew fetch` 与直接 `curl` Release 资产当前卡在下载阶段；该项按本机到 GitHub Release 资产下载链路不稳定记录，网络稳定后补跑。

已完成补充：v0.5.6 已正式发布，版本重点是外部用户项目 Onboarding CI bootstrap、复制模板、YAML 可解析性和失败分流。下一步建议回到产品主线，补一条“外部仓库真实 CI 复制演练”：在临时非 testloop 仓库用下载版 `run-onboarding-ci.sh` 跑最小 Go/Vue smoke，确认接入文档没有依赖本仓库上下文。

## 第二百四十九阶段：Onboarding CI 外部项目演练

1. [x] 新增 `scripts/showcase-onboarding-ci-external-project.sh`，在 `/tmp` 创建最小 Go 或 Node 项目，并从该项目目录运行 onboarding CI bootstrap。
2. [x] 演练脚本会把 `scripts/run-onboarding-ci.sh` 复制到临时路径执行，验证 bootstrap 不依赖用户项目拥有 testloop-mcp 仓库内 `scripts/` 目录。
3. [x] 脚本输出 `external_onboarding_project`、`external_onboarding_output_dir`、summary JSON、decision 路径和 `external_onboarding_status=passed`。
4. [x] 新增 `docs/onboarding-ci-external-dry-run.md`，记录可复现命令、预期输出、artifact 路径和适用边界。
5. [x] 新增 `test/onboarding_ci_external_dry_run_doc_test.sh`，固定文档入口、artifact 名称和 `agent_next_step=ready` 期望。
6. [x] `test/showcase_scripts_test.sh` 已覆盖新脚本的 bash 语法、帮助输出和参数错误。
7. [x] README、showcase 索引和 CHANGELOG 已补外部项目演练入口。
8. [x] 修复 `scripts/verify-mcp-process-smoke.sh` 从外部 Go module cwd 调用时的 `outside main module` 问题：进入 testloop-mcp 仓库后再执行 `go run ./examples/mcp-process-smoke`。
9. [x] 使用 `/tmp/testloop-mcp-external-onboarding` 和临时 Go 项目完成真实演练，输出 `external_onboarding_status=passed`、`agent_next_step=ready`。
10. [x] 脚本新增 `TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE=node|all`，Node 模式使用无第三方依赖项目验证 `pnpm install --frozen-lockfile && pnpm build` 命令形态。
11. [x] 使用 Node/web 模式完成真实演练，输出 `external_onboarding_node_status=passed`、`external_onboarding_status=passed`、`agent_next_step=ready`。
12. [x] 使用 `TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE=all` 复验 Go 和 Node 连续演练，分别生成 `artifacts/go` 与 `artifacts/node`，最终输出 `external_onboarding_mode=all`、`external_onboarding_status=passed`。

已完成补充：外部项目 Onboarding CI 演练已经通过真实临时 Go 和 Node/web 项目验证，并修复了内部 MCP process smoke 对调用 cwd 的隐含依赖。下一步建议提交这一阶段，并等待远端 CI；之后进入“安装后首跑体验”方向，补一个 `testloop-mcp doctor` 或文档化诊断路径，把版本、配置、MCP transport、onboarding artifact 位置集中输出。

## 第二百五十阶段：安装后首跑诊断

1. [x] 新增 `scripts/doctor-first-run.sh`，聚合 onboarding report 流程，稳定输出 `first_run_status`、`first_run_failed_count`、`first_run_agent_next_step`、报告路径、summary JSON、decision 和完整日志路径。
2. [x] 脚本支持 `TESTLOOP_FIRST_RUN_EXPECT_VERSION` 做版本门禁，避免 PATH 指向旧二进制。
3. [x] 脚本支持 `TESTLOOP_FIRST_RUN_PROJECT_DIR` 和 `TESTLOOP_FIRST_RUN_PROJECT_COMMAND`，可把用户项目 smoke 纳入同一条首跑诊断。
4. [x] 新增 `docs/first-run-diagnostics.md`，说明快速使用、输出字段和诊断边界。
5. [x] 新增 `test/doctor_first_run_test.sh`，覆盖成功路径、用户项目 smoke 失败路径、help 和参数错误。
6. [x] README、quickstart、showcase 索引和 release 文档索引测试已加入首跑诊断入口。
7. [x] 使用 `/tmp/testloop-mcp-first-run` 完成真实本地二进制首跑诊断，输出 `first_run_status=passed`、`first_run_failed_count=0`、`first_run_agent_next_step=ready`。

已完成补充：首跑诊断入口已经把安装、配置、真实 MCP transport、最小 Agent 闭环和可选用户项目 smoke 聚合成一条用户可执行命令，并通过真实本地构建二进制验证。下一步建议提交并等待远端 CI；之后继续做“诊断失败样例库”，把首跑失败时的 `first_run_agent_next_step` 和用户可粘贴给 Agent 的上下文固定下来。

## 第二百五十一阶段：首跑诊断失败样例库

1. [x] `scripts/doctor-first-run.sh` 新增 `first_run_context` 输出，默认写入 `first-run-context.txt`。
2. [x] `first-run-context.txt` 固定包含 `first_run_status`、`first_run_failed_count`、`first_run_agent_next_step`、下一步建议、报告路径、summary JSON、decision 和完整日志路径。
3. [x] 新增 `docs/fixtures/first-run/*.txt`，覆盖 `fix-installation`、`inspect-mcp-transport`、`inspect-agent-demo`、`inspect-showcase` 和 `inspect-user-project` 五类失败上下文。
4. [x] 新增 `docs/first-run-failures.md`，把首跑失败 action 映射到用户和 AI Agent 的下一步动作。
5. [x] 新增 `test/first_run_failure_fixtures_test.sh`，固定 fixture 字段、action 和可粘贴提示。
6. [x] `test/doctor_first_run_test.sh` 已验证成功和用户项目失败路径都会生成 `first-run-context.txt`。
7. [x] README、showcase、首跑诊断文档和 release 文档索引已补失败样例入口。
8. [x] 使用本地构建二进制和故意失败的用户项目 smoke 复验真实失败路径，脚本 exit code 为 `1`，但仍输出 `first_run_context=/tmp/testloop-mcp-first-run-failed-check/first-run-context.txt` 和 `first_run_agent_next_step=inspect-user-project`。

已完成补充：首跑诊断失败时现在有稳定、可粘贴给 AI Agent 的上下文文件，不需要用户手动整理完整日志；真实失败路径也已验证上下文文件可用。下一步建议运行完整本地门禁，提交并等待远端 CI；之后继续做“首跑诊断 CI 模板”，让用户可以把 `doctor-first-run.sh` 结果上传为 artifact。

## 第二百五十二阶段：首跑诊断 CI 模板

1. [x] 新增 `scripts/run-first-run-ci.sh`，面向外部用户项目 CI，负责安装或解析 testloop-mcp、准备 helper checkout，并调用首跑诊断入口。
2. [x] 脚本输出 `verification-report.md`、`verification-summary.json`、`agent-decision.txt`、`first-run-context.txt` 和 `first-run.log`。
3. [x] 脚本支持 `TESTLOOP_MCP_VERSION`、`TESTLOOP_FIRST_RUN_EXPECT_VERSION`、`TESTLOOP_FIRST_RUN_PROJECT_DIR`、`TESTLOOP_FIRST_RUN_PROJECT_COMMAND`、`TESTLOOP_MCP_REPO_DIR` 和 `TESTLOOP_MCP_COMMAND`。
4. [x] 脚本在 GitHub Actions 中会写入 `$GITHUB_STEP_SUMMARY`，展示 `first_run_agent_next_step` 和五类 artifact 路径。
5. [x] 新增 `test/run_first_run_ci_test.sh`，覆盖成功路径、用户项目失败路径、step summary、help 和指定版本安装路径。
6. [x] 新增 `docs/first-run-ci-template.md`，提供 Go server 与 Vue / Node 两份可复制 workflow。
7. [x] 新增 `test/first_run_ci_template_doc_test.sh` 和 `test/first_run_ci_template_yaml_test.sh`，固定模板文档和 YAML 可解析性。
8. [x] README、showcase、verification CI 文档、CHANGELOG 和 release 文档索引已补首跑诊断 CI 入口。
9. [x] 使用当前仓库和本地构建二进制完成 Go dry-run，输出 `first_run_status=passed`、`first_run_failed_count=0`、`first_run_agent_next_step=ready`。
10. [x] 使用临时 Node/web 项目完成 dry-run，验证 `pnpm install --frozen-lockfile && pnpm build` 命令形态，输出 `first_run_agent_next_step=ready`。
11. [x] 修正 `run-first-run-ci.sh` 的 helper checkout 默认 ref 为 `main`，避免当前 main 新增的首跑诊断脚本搭配 `TESTLOOP_MCP_VERSION=v0.5.6` 时误 clone 缺少 helper 的旧 tag。

已完成补充：外部用户项目现在可以复制首跑诊断 CI 模板，失败时拿到可直接交给 AI Agent 的 `first-run-context.txt`；Go 和 Node/web dry-run 都已验证五件套 artifact 和 `ready` 信号。下一步建议运行完整本地门禁，提交并等待远端 CI；之后继续把 `run-first-run-ci.sh` 和 `run-onboarding-ci.sh` 的重复 bootstrap 逻辑收敛成共享 helper，降低维护成本。

## 第二百五十三阶段：CI bootstrap 单脚本回归保护

1. [x] `test/run_first_run_ci_test.sh` 已覆盖 `run-first-run-ci.sh` 被复制到临时路径执行时，会 clone helper checkout 并默认使用 `main` ref。
2. [x] `test/run_onboarding_ci_test.sh` 新增同类复制版 bootstrap 回归，验证外部 CI 下载单脚本后仍会 clone helper checkout、安装指定版本二进制并传递版本门禁。
3. [x] Onboarding CI 复制版测试固定 `TESTLOOP_MCP_VERSION=v8.8.8` 时 helper checkout 使用同版本 tag/ref，避免后续收敛公共逻辑时意外改变已发布 onboarding 语义。

已完成补充：两个面向外部用户项目的 bootstrap 入口现在都有“脱离仓库 scripts 目录、复制到临时路径执行”的单元级回归保护。下一步建议先跑相关 shell 测试和完整本地门禁，提交并等待远端 CI；之后再评估是否值得抽取公共 bootstrap helper，或继续优先增强真实接入样例。

## 第二百五十四阶段：首跑诊断 CI 外部项目演练

1. [x] 新增 `scripts/showcase-first-run-ci-external-project.sh`，在 `/tmp` 创建最小 Go 或 Node 项目，并从该项目目录运行首跑诊断 CI bootstrap。
2. [x] 演练脚本会把 `scripts/run-first-run-ci.sh` 复制到临时路径执行，验证 bootstrap 不依赖用户项目拥有 testloop-mcp 仓库内 `scripts/` 目录。
3. [x] 脚本输出 `external_first_run_project`、`external_first_run_output_dir`、summary JSON、decision、context、log 路径和 `external_first_run_status=passed`。
4. [x] 脚本支持 `TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE=node|all`，Node 模式使用无第三方依赖项目验证 `pnpm install --frozen-lockfile && pnpm build` 命令形态。
5. [x] 新增 `docs/first-run-ci-external-dry-run.md`，记录可复现命令、预期输出、五件套 artifact 路径和适用边界。
6. [x] 新增 `test/first_run_ci_external_dry_run_doc_test.sh`，固定文档入口、artifact 名称、`agent_next_step=ready` 和 `first_run_agent_next_step=ready` 期望。
7. [x] `test/showcase_scripts_test.sh` 已覆盖新脚本的 bash 语法、帮助输出和参数错误。
8. [x] README、showcase 索引和 CHANGELOG 已补首跑诊断 CI 外部项目演练入口。
9. [x] 使用本地构建二进制和 `TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE=all` 完成真实演练，Go 和 Node 两条路径均输出 `first_run_agent_next_step=ready`，最终输出 `external_first_run_mode=all`、`external_first_run_status=passed`。

已完成补充：首跑诊断 CI 现在和 Onboarding CI 一样具备外部项目复制演练路径，能证明 Go server 与 Node/web 项目从非 testloop 仓库目录也能拿到五件套诊断 artifact；Go + Node 连续演练已通过。下一步建议运行完整本地门禁，提交并等待远端 CI；之后优先收敛 README/文档里的首次接入路径，减少 onboarding 三件套与 first-run 五件套之间的选择成本。

## 第二百五十五阶段：首次接入 CI 入口选择收敛

1. [x] `docs/verification-ci.md` 新增“怎么选入口”章节，明确首次接入、安装后排查和需要交给 AI Agent 时优先使用 `run-first-run-ci.sh`。
2. [x] 同一章节明确稳定接入后的 PR / 发布后 smoke 优先使用 `run-onboarding-ci.sh`，避免用户误以为两条 bootstrap 必须同时接入。
3. [x] 文档说明维护者改模板后应运行 `showcase-onboarding-ci-external-project.sh` 和 `showcase-first-run-ci-external-project.sh` 复验外部项目复制路径。
4. [x] README 在两个 bootstrap 命令后补充直达链接，指向验收 CI 文档的选择规则。

已完成补充：首次接入路径现在有明确分流：first-run 负责“首次接入和失败上下文”，onboarding 负责“稳定持续验收”。下一步建议运行文档链接和完整本地门禁，提交并等待远端 CI；之后可以开始准备 v0.5.7 候选收敛，把 v0.5.6 后的 first-run CI、外部演练和入口选择规则整理成发布说明。

## 第二百五十六阶段：v0.5.7 release readiness

1. [x] 新增 `docs/plan-release-notes-v0.5.7.md`，归纳 v0.5.6 之后的首跑诊断 CI、失败上下文、外部项目复制演练和入口选择规则。
2. [x] 新增 `docs/plan-release-v0.5.7.md`，记录候选内容、已验证项、发布前门禁和正式发布前待办。
3. [x] CHANGELOG 已记录 v0.5.7 候选发布资料。
4. [x] 完成本地 release readiness 门禁：脚本语法、`go test ./...`、默认 shell 矩阵、主服务/testgen 构建、打包 dry-run、sha256 校验和 `git diff --check`。
5. [x] 最新远端 CI run `29651790811` 已通过。

已完成补充：v0.5.7 候选发布资料已就绪，本地 release readiness 门禁和远端 CI 都已通过，但还没有进入正式版本准备；`main.go` 版本号、安装文档版本引用、CHANGELOG 版本归档和 Release/Homebrew 流程都保持待办。下一步建议提交候选文档；之后如继续发版，进入 v0.5.7 正式版本准备。

## 第二百五十七阶段：v0.5.7 正式版本准备

1. [x] `main.go` MCP implementation version 已更新为 `0.5.7`。
2. [x] `CHANGELOG.md` 的 `Unreleased` 内容已收敛为 `v0.5.7 - 2026-07-19`，并记录 implementation version 更新。
3. [x] README 中当前 Release、手动下载示例和 Windows 下载示例已同步到 `v0.5.7`。
4. [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.5.7`。
5. [x] quickstart、onboarding、first-run、verification report、verification CI、onboarding CI、first-run CI 和真实接入案例文档中的版本门禁已同步到 `0.5.7`。
6. [x] 测试中的版本期望已同步到 `0.5.7`。
7. [x] 使用本地构建二进制完成 v0.5.7 first-run 外部项目 all 模式 dry-run，Go 和 Node 均输出 `first_run_agent_next_step=ready`。
8. [x] 使用同一 v0.5.7 二进制完成 onboarding 外部项目 all 模式 dry-run，Go 和 Node 均输出 `agent_next_step=ready`。
9. [x] 本地验证已通过：脚本语法、`go test ./...`、默认 shell 矩阵、主服务/testgen 构建、darwin arm64 打包 dry-run、sha256 校验和 tarball 内容检查。
10. [x] 提交版本准备改动后远端 CI run `29652321592` 已通过。
11. [x] 远端 CI 通过后已打 `v0.5.7` tag，并进入 Release Artifacts / 资产校验 / GitHub Release 正文 / Homebrew tap 验证。

已完成补充：v0.5.7 正式版本准备的本地改动已完成，本地门禁、两条外部项目 all 演练和版本准备远端 CI 都已通过；`v0.5.7` tag 已推送。下一步进入 v0.5.7 正式发布核验。

## 第二百五十八阶段：v0.5.7 正式发布核验

1. [x] 推送 `v0.5.7` tag，指向 `76d72934a040ac34dc2ca223cab678777e2de006`。
2. [x] Release Artifacts workflow run `29665920056` passed，Linux amd64、Linux arm64、macOS arm64、Windows amd64、Windows arm64 五个平台资产和 `.sha256` 已上传。
3. [x] 运行 `scripts/verify-release-assets.sh v0.5.7`，确认 10 个 Release 资产完整。
4. [x] 更新 GitHub Release 正文为正式 v0.5.7 发布说明。
5. [x] 仓库内 `Formula/testloop-mcp.rb` 已更新到 `0.5.7`，使用 Release API 返回的真实 digest。
6. [x] 更新 `sleticalboy/homebrew-tap` 到 `testloop-mcp 0.5.7`，提交 `5538b6b` 并推送。
7. [x] 本机 Homebrew tap 已快进到 `5538b6b60cc7ce31c47f1d39b75726ce92e523f3`，`brew info --json=v2 sleticalboy/tap/testloop-mcp` 返回 `version=0.5.7`。
8. [x] Post-Release Verify run `29666102074` passed，资产清单和五平台安装脚本 dry run 全部通过。

已完成补充：v0.5.7 已正式发布并完成发布后核验。Release Artifacts、资产清单、GitHub Release 正文、仓库 Formula、Homebrew tap、本机 tap 缓存和 Post-Release Verify 均已闭环。下一步建议回到产品主线，优先补一个“接入方一页式验证指南”或继续真实项目接入样本，避免继续只堆发布流程。

## 第二百五十九阶段：接入方一页式验证指南

1. [x] 新增 `docs/adopter-verification-guide.md`，把安装、版本确认、本机 first-run、CI bootstrap、artifact 上传和失败分流压成一页执行清单。
2. [x] 指南明确首次接入优先 `run-first-run-ci.sh`，稳定 PR / 发布后 smoke 使用 `run-onboarding-ci.sh`。
3. [x] 指南固定 first-run 五件套和 onboarding 三件套 artifact 名称，提示 GitHub Actions 上传 artifact 时使用 `if: always()`。
4. [x] 指南明确失败时优先读取 `agent-decision.txt`，first-run 失败直接把 `first-run-context.txt` 交给 AI Agent。
5. [x] 指南提供外部项目复制演练命令，覆盖 first-run 和 onboarding 的 `all` 模式。
6. [x] 新增 `test/adopter_verification_guide_doc_test.sh`，固定关键命令、版本门禁、artifact 名称和 action 分流。
7. [x] README、showcase 索引、release 文档索引测试和 CHANGELOG 已补新指南入口。

已完成补充：接入方现在有一份一页式验证指南，不需要在 quickstart、verification CI、first-run 和 onboarding 文档之间来回跳。下一步建议用一个真实外部项目按这份指南跑一遍，从用户视角找仍然绕的地方，再决定是否继续改文档或脚本。

## 第二百六十阶段：真实外部项目接入指南复验

1. [x] 使用 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server` 作为 Go server 真实外部项目，确认本地仓库状态干净，并用 `go test ./...` 完成项目侧 smoke。
2. [x] 使用 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-web` 作为 Vue web 真实外部项目，确认本地仓库状态干净，并用 `pnpm install --frozen-lockfile && pnpm build:prod` 完成项目侧 smoke。
3. [x] 使用 `TESTLOOP_MCP_VERSION=v0.5.7 scripts/run-first-run-ci.sh` 跑通 server first-run，输出 `/tmp/testloop-laoxia-server-first-run` 五件套，`first_run_agent_next_step=ready`。
4. [x] 使用 `TESTLOOP_MCP_VERSION=v0.5.7 scripts/run-first-run-ci.sh` 跑通 web first-run，输出 `/tmp/testloop-laoxia-web-first-run` 五件套，`first_run_agent_next_step=ready`。
5. [x] 使用 `TESTLOOP_MCP_VERSION=v0.5.7 scripts/run-onboarding-ci.sh` 跑通 server onboarding，输出 `/tmp/testloop-laoxia-server-onboarding` 三件套，`agent_next_step=ready`。
6. [x] 使用 `TESTLOOP_MCP_VERSION=v0.5.7 scripts/run-onboarding-ci.sh` 跑通 web onboarding，输出 `/tmp/testloop-laoxia-web-onboarding` 三件套，`agent_next_step=ready`。
7. [x] 更新 `docs/real-integration-cases.md`，把 v0.5.7 first-run / onboarding bootstrap 实跑记录设为当前主案例，v0.5.4 样例保留为历史记录。
8. [x] 更新 `docs/adopter-verification-guide.md`，补充 `PATH` 版本漂移和 bootstrap 版本门禁的区别。
9. [x] 更新 `test/real_integration_cases_doc_test.sh` 和 `test/adopter_verification_guide_doc_test.sh`，固定 v0.5.7 真实外部项目 artifact、版本输出和失败上下文字段。

已完成补充：接入指南已经用真实 Go server 和 Vue web 项目复验，first-run 与 onboarding 两条 CI bootstrap 都能在外部项目目录下输出稳定 artifact，且不会污染用户仓库。下一步建议把这条真实案例转成 README 中更短的“复制哪条命令”入口，减少用户第一次接入时在展示文档和真实案例文档之间跳转。

## 第二百六十一阶段：README 首次接入入口收敛

1. [x] README 新增“用户项目接入：直接复制”小节，把首次接入和稳定接入的推荐入口放到首页。
2. [x] 首次接入推荐 `run-first-run-ci.sh`，并给出外部用户可复制的 `curl` 下载脚本命令、`TESTLOOP_MCP_VERSION=v0.5.7`、输出目录和 `go test ./...` smoke。
3. [x] 稳定 PR / 发布后 smoke 推荐 `run-onboarding-ci.sh`，并给出同样可复制的 `curl` bootstrap 命令。
4. [x] README 明确 Vue / Node 项目把 smoke 命令换成 `pnpm install --frozen-lockfile && pnpm build`。
5. [x] README 明确 first-run 五件套、onboarding 三件套、`agent-decision.txt` 和 `first-run-context.txt` 的读取顺序，并链接一页式验证指南和真实接入案例。
6. [x] 更新 `test/release_doc_index_test.sh`，固定 README 中两个 `curl` bootstrap 命令和首次接入关键文案。
7. [x] CHANGELOG 已记录 README 首次接入入口收敛。

已完成补充：README 首页现在直接告诉用户“首次接入复制 first-run、稳定 CI 复制 onboarding”，不需要先阅读展示路径索引才能开始接入。下一步建议补一份真正可粘贴的 GitHub Actions 最小 workflow 片段到 README 或 quickstart，让用户从本机命令自然过渡到 CI 文件。

## 第二百六十二阶段：README 最小 CI workflow 片段

1. [x] README 在“用户项目接入：直接复制”小节补充最小 GitHub Actions first-run workflow，可直接保存为 `.github/workflows/testloop-first-run.yml`。
2. [x] workflow 片段固定 `actions/checkout@v4`、`actions/setup-go@v5`、`TESTLOOP_MCP_VERSION=v0.5.7`、`scripts/run-first-run-ci.sh` 和 `go test ./...`。
3. [x] workflow 片段使用 `actions/upload-artifact@v4` 和 `if: always()` 上传 first-run 五件套，确保失败时保留 `first-run-context.txt` 和 `first-run.log`。
4. [x] 新增 `test/readme_ci_snippet_test.sh`，从 README 提取 YAML 片段并用 Ruby YAML 解析，固定关键 action、环境变量、artifact 和 smoke 命令。
5. [x] 更新 `test/release_doc_index_test.sh`，固定 README 中 GitHub Actions 最小片段入口和 artifact 上传 action。
6. [x] CHANGELOG 已记录 README 最小 CI workflow 片段。

已完成补充：README 现在不仅告诉用户本机要跑什么，也给出可直接落地到 GitHub Actions 的最小 workflow。下一步建议从“复制接入”转向“失败后怎么交给 Agent”，补一个最短 triage 示例：下载 artifact 后读取 `agent-decision.txt` / `first-run-context.txt`，并给出 Agent 应执行的下一步。

## 第二百六十三阶段：CI 失败后 Agent triage 最短路径

1. [x] 新增 `docs/ci-agent-triage.md`，说明 GitHub Actions 失败后如何用 `gh run download` 下载 `testloop-first-run` artifact。
2. [x] 文档固定首读 `agent-decision.txt`，再根据 `agent_next_step` 分流 `ready`、`fix-installation`、`inspect-mcp-transport`、`inspect-agent-demo` 和 `inspect-user-project`。
3. [x] 文档明确 first-run 失败时优先把 `first-run-context.txt` 全文交给 AI Agent，只有 Agent 需要更细日志时再补 `verification-summary.json`、`verification-report.md` 和 `first-run.log`。
4. [x] 文档补充 onboarding artifact 的最小粘贴顺序：`agent-decision.txt`、`verification-summary.json`、`verification-report.md`。
5. [x] README 在用户项目接入段落补充 CI 失败后的最短排查入口，避免用户只贴 GitHub Actions 最后一行错误。
6. [x] 新增 `test/ci_agent_triage_doc_test.sh`，固定下载命令、artifact 文件名、分流字段和关联文档链接。
7. [x] 更新 `test/release_doc_index_test.sh`，把 triage 文档纳入 README 发布文档索引。
8. [x] CHANGELOG 已记录 CI 失败后 Agent triage 文档。

已完成补充：接入链路已经覆盖“复制命令 -> 落地 CI -> 失败后交给 Agent”的最短闭环。下一步建议把这条链路做一次失败态真实演练：构造一个故意失败的外部项目 smoke，运行 README first-run workflow 等价命令，确认 `agent-decision.txt` 和 `first-run-context.txt` 确实足够指导下一步。

## 第二百六十四阶段：first-run 失败态真实演练

1. [x] 在 `/tmp/testloop-triage-failing-project` 构造外部临时项目，使用 `echo testloop intentional project failure; exit 7` 作为故意失败的 smoke 命令。
2. [x] 运行 README first-run workflow 等价命令，输出目录为 `/tmp/testloop-first-run-failure-triage`。
3. [x] 确认 first-run 输出 `first_run_status=failed`、`first_run_failed_count=1`、`first_run_agent_next_step=inspect-user-project`。
4. [x] 确认 `verification-summary.json` 只有“用户项目 smoke”失败，exit code 为 `7`，基础安装验收、真实 MCP 协议 smoke 和最小 Agent 闭环 demo 均通过。
5. [x] 确认 `verification-report.md` 的失败 section 保留项目输出 `testloop intentional project failure`。
6. [x] 更新 `docs/ci-agent-triage.md`，记录失败态实跑命令、结果、summary 和 report 关键信息。
7. [x] 更新 `test/ci_agent_triage_doc_test.sh`，固定失败态实跑记录中的 artifact 路径、decision、失败 section 和项目输出。
8. [x] 修正 `scripts/install.sh` checksum fallback：当聚合 `checksums.txt` 存在但不包含当前资产时，继续尝试单资产 `.sha256`。
9. [x] 更新 `test/install_script_test.sh`，离线覆盖 `checksums.txt` 存在但缺当前资产的 fallback 场景。

已完成补充：失败态演练证明 first-run artifact 能把用户项目失败稳定分流给 Agent；同时安装脚本的 checksum fallback 更稳，不会因为旧版聚合 checksum 文件缺少新资产而输出误导性错误。下一步建议进入接入体验的最后一层：给 Agent 一个“拿到 first-run-context 后应该怎么回话/怎么行动”的样例，固定用户粘贴上下文后的 Agent 响应格式。

## 第二百六十五阶段：first-run Agent 回复格式

1. [x] 新增 `docs/first-run-agent-response.md`，定义 Agent 收到 `first-run-context.txt` 后的四段回复结构：结论、证据、下一步、暂不做。
2. [x] 文档固定 `ready`、`fix-installation`、`inspect-mcp-transport`、`inspect-agent-demo`、`inspect-user-project` 和 `inspect-showcase` 的分流动作。
3. [x] 文档补充 `inspect-user-project` 示例，使用失败态实跑中的 `failed_section=用户项目 smoke` 和 `exit_code=7`。
4. [x] 文档补充 `fix-installation` 示例，明确不要先修改用户项目测试或排查覆盖率。
5. [x] 文档补充用户只贴 CI 最后一行错误时的回复，要求先提供 `agent-decision.txt` 和 `first-run-context.txt`。
6. [x] README 和 `docs/ci-agent-triage.md` 已链接到 first-run Agent 回复格式。
7. [x] 新增 `test/first_run_agent_response_doc_test.sh`，固定回复结构、分流 action、示例证据和关联文档链接。
8. [x] 更新 `test/release_doc_index_test.sh`，把 first-run Agent 回复格式纳入 README 发布文档索引。
9. [x] CHANGELOG 已记录 first-run Agent 回复格式文档。

已完成补充：首次接入链路已经从“用户怎么运行”延伸到“CI 失败怎么粘给 Agent”以及“Agent 应如何回复和行动”。下一步建议把这些接入体验改动收敛成 v0.5.8 候选发布说明草案，并判断是否需要作为 patch 版本发布。

## 第二百六十六阶段：v0.5.8 release readiness

1. [x] 新增 `docs/plan-release-notes-v0.5.8.md`，归纳 v0.5.7 之后的接入方一页式验证、真实项目复验、README 复制入口、CI 最小 workflow、失败 triage、Agent 回复格式和安装 checksum fallback 修复。
2. [x] 新增 `docs/plan-release-v0.5.8.md`，记录候选内容、已验证项、发布前门禁和正式发布前待办。
3. [x] CHANGELOG 已记录 v0.5.8 候选发布资料。
4. [x] 最新候选提交远端 CI run `29668735511` 已通过。
5. [x] 本地发布前门禁已通过：脚本语法、`go test ./...`、完整 shell 矩阵、主服务/testgen 构建、help 输出、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。

已完成补充：v0.5.8 候选发布资料和本地 release readiness 门禁已就绪，但还没有进入正式版本准备；`main.go` 版本号、安装文档版本引用、CHANGELOG 归档、tag、Release Artifacts 和 Homebrew tap 都保持待办。下一步建议提交候选文档并等待远端 CI，通过后进入 v0.5.8 正式版本准备。

## 第二百六十七阶段：v0.5.8 正式版本准备

1. [x] `main.go` MCP implementation version 已更新为 `0.5.8`。
2. [x] `CHANGELOG.md` 的 `Unreleased` 内容已收敛为 `v0.5.8 - 2026-07-19`，并记录 implementation version 更新。
3. [x] README 中当前 Release、手动下载示例、Windows 下载示例和接入 bootstrap 示例已同步到 `v0.5.8`。
4. [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.5.8`。
5. [x] quickstart、first-run、verification report、verification CI、onboarding CI 和接入指南中的版本门禁已同步到 `0.5.8`。
6. [x] 测试中的版本期望已同步到 `0.5.8`。
7. [x] `docs/plan-release-notes-v0.5.8.md` 和 `docs/plan-release-v0.5.8.md` 已标记正式版本准备同步项。
8. [x] 正式版本准备本地完整验证已通过：脚本语法、`go test ./...`、完整 shell 矩阵、主服务/testgen 构建、help 输出、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。
9. [x] 版本准备提交 `5ae841a` 的远端 CI run `29669148638` 已通过。

已完成补充：v0.5.8 正式版本准备的版本同步、本地完整验证和远端 CI 已完成。下一步打 `v0.5.8` tag，并进入 Release Artifacts / 资产校验 / GitHub Release 正文 / Homebrew tap 验证。

## 第二百六十八阶段：v0.5.8 正式发布核验

1. [x] 推送 `v0.5.8` tag，指向 `c2e6a1873fd14ad45b3f5a6e88333b2842503ebc`。
2. [x] Release Artifacts workflow run `29669279828` 已通过，五个平台构建 job 均完成。
3. [x] 运行 `scripts/verify-release-assets.sh v0.5.8`，确认 10 个 Release 资产完整。
4. [x] 更新 GitHub Release 正文为正式 v0.5.8 发布说明。
5. [x] 仓库内 `Formula/testloop-mcp.rb` 已更新到 `0.5.8`，使用 Release API 返回的真实 digest。
6. [x] 更新 `sleticalboy/homebrew-tap` 到 `testloop-mcp 0.5.8`，tap commit `11b06d0dedfa9a7d31537136e62a0fd774638d3c` 已推送。
7. [x] 本机 Homebrew tap 验证通过：`brew info --json=v2` 返回 `version=0.5.8`、`tap_git_head=11b06d0dedfa9a7d31537136e62a0fd774638d3c`。
8. [x] 本机 Homebrew 验证通过：`brew fetch --force --formula`、`brew audit --formula --strict`、`brew test`；`/opt/homebrew/bin/testloop-mcp --version` 输出 `testloop-mcp 0.5.8`。
9. [x] 手动触发 Post-Release Verify run `29669664060`，资产清单和五平台安装脚本 dry run 均通过。

已完成补充：v0.5.8 已正式发布并完成发布后核验。Release Artifacts、资产清单、GitHub Release 正文、仓库 Formula、Homebrew tap、本机 tap 验证和 Post-Release Verify 均已闭环。下一步回到产品主线，优先做真实接入失败样本沉淀和 Agent 消费 artifact 的端到端演示，避免继续只堆发布流程。

## 第二百六十九阶段：first-run artifact Agent 消费演示

1. [x] 新增 `examples/first-run-agent-response-demo`，读取 `first-run-context.txt` 和可选 `verification-summary.json`。
2. [x] demo 输出固定四段 Agent 回复：结论、证据、下一步、暂不做。
3. [x] demo 覆盖 `inspect-user-project`、`fix-installation` 等已知 `first_run_agent_next_step` 分流。
4. [x] 新增 `docs/first-run-agent-artifact-demo.md`，说明如何用内置 fixture 或真实 CI artifact 运行 demo。
5. [x] README 和 `docs/first-run-agent-response.md` 已链接 artifact 消费演示。
6. [x] 新增 `test/first_run_agent_response_demo_test.sh`，固定用户项目失败和安装失败两条输出。
7. [x] `test/first_run_agent_response_demo_test.sh` 已扩展端到端路径：先运行 `scripts/run-first-run-ci.sh` 产出失败五件套，再把 `first-run-context.txt` 和 `verification-summary.json` 喂给 demo。

已完成补充：first-run artifact 现在不仅能被文档解释，也能被可运行 demo 转成 Agent 应回复的四段结构；端到端测试已经覆盖 `run-first-run-ci.sh` 失败五件套到 demo 输出的链路。下一步建议沉淀一份真实失败 artifact fixture 包，方便客户端/Agent 不跑脚本也能回归消费逻辑。

## 第二百七十阶段：first-run 失败 artifact fixture 包

1. [x] 新增 `docs/fixtures/first-run-artifacts/user-project-smoke-failed/`，沉淀 first-run 失败五件套。
2. [x] fixture 包包含 `verification-report.md`、`verification-summary.json`、`agent-decision.txt`、`first-run-context.txt` 和 `first-run.log`。
3. [x] fixture 场景固定为用户项目 smoke exit code `7`，`agent_next_step=inspect-user-project`。
4. [x] `docs/first-run-agent-artifact-demo.md` 已改用完整 artifact fixture 包作为默认演示输入。
5. [x] 新增 `test/first_run_artifact_fixtures_test.sh`，验证 fixture 文件完整、summary JSON 可解析、decision/context 字段正确，并能被 Agent 回复 demo 消费。
6. [x] `docs/fixtures.md` 已新增 first-run artifact fixture 小节，统一索引覆盖率 task fixture 和 CI artifact fixture。

已完成补充：客户端/Agent 现在可以直接读取一份稳定的 first-run 失败 artifact 包来回归消费逻辑，并能从 fixture 索引发现它。下一步建议把客户端集成文档扩展到 first-run artifact 消费，给出“coverage task fixture”和“CI artifact fixture”两类回归入口的区别。

## 第二百七十一阶段：客户端集成文档区分两类 fixture

1. [x] `docs/client-integration.md` 新增 CI artifact fixture 小节，明确 first-run artifact 不是 MCP tool 返回值。
2. [x] 文档固定 `docs/fixtures/first-run-artifacts/user-project-smoke-failed/` 作为客户端/Agent 测试输入。
3. [x] 文档说明 artifact 消费顺序：`agent-decision.txt`、`first-run-context.txt`、`verification-summary.json`、`verification-report.md`。
4. [x] 文档加入 `go run ./examples/first-run-agent-response-demo` 命令，展示如何从 artifact 输出 Agent 四段回复。
5. [x] `test/client_integration_doc_test.sh` 已固定 first-run artifact fixture、demo 命令、action 字段和文档链接。

已完成补充：客户端集成文档现在区分了 MCP tool 结构化返回 fixture 和 CI artifact fixture，两条回归入口都有可运行 demo 和测试保护。下一步建议把 release/readme 索引补上 first-run artifact fixture，确保新入口从首页可发现。

## 第二百七十二阶段：README 索引补齐 first-run artifact 入口

1. [x] README 的 Agent 集成入口已说明 `docs/fixtures.md` 同时包含真实 handler fixture 和 first-run artifact fixture。
2. [x] README 的 CI 失败排查入口已链接 [first-run artifact Agent 消费演示](./first-run-agent-artifact-demo.md) 和 first-run 失败 artifact fixture。
3. [x] README 的展示路径索引已补 first-run artifact Agent 消费演示。
4. [x] `test/release_doc_index_test.sh` 已固定 artifact demo 文档、demo 命令、artifact fixture 路径和首页关键词。

已完成补充：first-run artifact 消费入口已经从 README 可发现，并被 release doc index 测试保护。下一步建议整理一个小版本候选计划，判断这些 v0.5.8 之后的 Agent artifact 消费改动是否需要作为 v0.5.9 patch 发布。

## 第二百七十三阶段：v0.5.9 release readiness

1. [x] `CHANGELOG.md` 已新增 `Unreleased`，归纳 v0.5.8 之后的 Agent artifact 消费 demo、端到端回归、artifact fixture 包、客户端集成文档和 README 入口。
2. [x] 新增 `docs/plan-release-notes-v0.5.9.md`，整理 v0.5.9 候选发布说明草案。
3. [x] 新增 `docs/plan-release-v0.5.9.md`，记录候选内容、已验证项、发布前门禁和正式发布前待办。
4. [x] 当前最新远端 CI run `29670275988` 已通过。
5. [x] 本地 v0.5.9 release readiness 门禁已通过：脚本语法、`go test ./...`、完整 shell 矩阵、主服务/testgen 构建、help 输出、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。

已完成补充：v0.5.9 候选发布资料和本地 release readiness 门禁已完成，但还没有进入正式版本准备；`main.go` 版本号、安装文档版本引用、CHANGELOG 归档、tag、Release Artifacts 和 Homebrew tap 都保持待办。下一步提交候选资料并等待远端 CI；通过后再判断是否发布 v0.5.9。

## 第二百七十四阶段：v0.5.9 正式版本准备

1. [x] `main.go` MCP implementation version 已更新为 `0.5.9`。
2. [x] `CHANGELOG.md` 的 `Unreleased` 内容已收敛为 `v0.5.9 - 2026-07-19`，并记录 implementation version 更新。
3. [x] README 中当前 Release、手动下载示例、Windows 下载示例和接入 bootstrap 示例已同步到 `v0.5.9`。
4. [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.5.9`。
5. [x] quickstart、first-run、verification report、verification CI、onboarding CI 和接入指南中的版本门禁已同步到 `0.5.9`。
6. [x] 测试中的版本期望已同步到 `0.5.9`。
7. [x] `docs/plan-release-notes-v0.5.9.md` 和 `docs/plan-release-v0.5.9.md` 已标记正式版本准备同步项。
8. [x] 正式版本准备本地完整验证已通过：脚本语法、`go test ./...`、完整 shell 矩阵、主服务/testgen 构建、`--version`、help 输出、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。
9. [x] 正式版本准备提交 `c8a56cb` 远端 CI run `29670649066` 已通过。
10. [x] `v0.5.9` tag 已推送，Release Artifacts run `29670770869` 已通过。
11. [x] `scripts/verify-release-assets.sh v0.5.9` 已确认 10 个 Release 资产完整。
12. [x] GitHub Release 正文已更新为正式 v0.5.9 发布说明。
13. [x] 仓库内 `Formula/testloop-mcp.rb` 已生成 v0.5.9 版本和真实 sha256。
14. [x] 仓库内 Formula 和发布记录提交 `a3f73b6` 远端 CI run `29670948462` 已通过。
15. [x] Homebrew tap 已更新到 v0.5.9，tap commit `62280bf`。
16. [x] Post-Release Verify run `29671001523` 已通过，覆盖资产清单和五平台安装脚本 dry run。

已完成补充：v0.5.9 发布流程已完成，tag、Release Artifacts、资产校验、GitHub Release 正文、仓库内 Formula、Homebrew tap 和 Post-Release Verify 均已收口。下一步回到主线产品价值，继续打磨 Agent/客户端消费发布 artifact 的真实接入示例。

## 第二百七十五阶段：first-run artifact 目录入口

1. [x] 新增 `scripts/render-first-run-agent-response.sh`，接收 first-run artifact 目录并自动读取 `first-run-context.txt`。
2. [x] 当目录存在 `verification-summary.json` 时，脚本会一起传给 `examples/first-run-agent-response-demo`；不存在时按 context-only 路径渲染回复。
3. [x] 新增 `test/render_first_run_agent_response_test.sh`，覆盖完整 fixture 目录、context-only 目录、help 和缺少 context 的错误提示。
4. [x] README、`docs/client-integration.md`、`docs/first-run-agent-artifact-demo.md`、fixture README 和 showcase 文档已补目录入口。
5. [x] `CHANGELOG.md` 已在新的 `Unreleased` 记录该入口，避免继续改已发布 v0.5.9 内容作为未来变更。

已完成补充：接入方现在拿到 GitHub Actions artifact 目录后，可以直接运行目录入口生成 Agent 四段回复，不需要手动拼两个文件路径。下一步建议运行完整本地门禁，提交并等待远端 CI；通过后继续把这个目录入口沉到复制型 CI 模板或失败 triage 文档中，减少用户从 Actions artifact 到 Agent 回复之间的手工步骤。

## 第二百七十六阶段：first-run 自动生成 Agent 回复 artifact

1. [x] `scripts/run-first-run-ci.sh` 已新增 `agent-response.txt` 输出路径，默认写到 first-run artifact 目录。
2. [x] 当 helper checkout 包含 `scripts/render-first-run-agent-response.sh` 且 `first-run-context.txt` 已生成时，first-run CI 会自动渲染 Agent 四段回复草稿。
3. [x] GitHub step summary 会在 `agent-response.txt` 存在时列出该路径。
4. [x] `test/run_first_run_ci_test.sh` 已覆盖 passed 和 failed 两种路径的 `agent-response.txt` 内容与 step summary 链接。
5. [x] first-run CI 模板、CI 失败 triage、客户端集成、接入指南和 README 已同步说明 six-pack artifact。
6. [x] 静态失败 artifact fixture 已升级为包含 `agent-response.txt` 的六件套，并通过 `test/first_run_artifact_fixtures_test.sh` 固定与目录入口实时渲染结果一致。

已完成补充：first-run 失败后，用户不必先理解 context/summary 如何组合，artifact 里已经有 `agent-response.txt` 可以直接作为 Agent 回复草稿；静态失败 fixture 也已升级为同样的六件套。下一步建议运行完整本地门禁，提交并等待远端 CI；通过后继续把复制型 CI 文档收敛成“失败后读 agent-response.txt，必要时再看 context/summary/report”的优先级，减少接入方排查分叉。

## 第二百七十七阶段：first-run 失败排查读取优先级收敛

1. [x] `docs/ci-agent-triage.md` 已新增快速路径：新版 artifact 先读 `agent-response.txt`。
2. [x] `docs/first-run-agent-response.md` 已说明 `agent-response.txt` 可作为回复草稿，缺失时再要求 `agent-decision.txt` 和 `first-run-context.txt`。
3. [x] `docs/first-run-ci-template.md` 的失败读取顺序已调整为 `agent-response.txt` 优先，旧版 artifact 或机器分流再看 context/decision。
4. [x] 文档回归测试已补 `agent-response.txt` 优先级断言。

已完成补充：用户从 GitHub Actions artifact 到 Agent 回复的路径已经进一步缩短，首选入口变成 `agent-response.txt`，复杂排查才下钻到 context/summary/report。下一步建议运行完整本地门禁，提交并等待远端 CI；通过后再考虑是否把 onboarding 三件套也补一个 response 草稿，统一 first-run 和 onboarding 的 Agent 消费体验。

## 第二百七十八阶段：外部 first-run showcase 六件套校验

1. [x] `scripts/showcase-first-run-ci-external-project.sh` 已校验 `agent-response.txt` 存在且包含 `first_run_agent_next_step=ready`。
2. [x] showcase 输出已新增 `external_first_run_*_agent_response` 路径，Go、Node 和 all 模式都可定位回复草稿。
3. [x] `docs/first-run-ci-external-dry-run.md` 已从五件套更新为六件套，并补充 `agent-response.txt` 路径和实跑记录字段。
4. [x] README、showcase、接入指南、artifact demo 和 fixture README 的当前说明已从五件套收敛为六件套。
5. [x] 文档回归已补 `agent-response.txt` 外部 dry-run 断言。

已完成补充：复制型 first-run bootstrap 的外部项目演练现在也证明六件套 artifact 可用，不只是在单元测试里生成 `agent-response.txt`。下一步建议运行完整本地门禁，提交并等待远端 CI；通过后再评估 onboarding 是否需要对齐 response 草稿，避免把 first-run 和 onboarding 的失败消费体验拉开。

## 第二百七十九阶段：onboarding Agent 回复 artifact

1. [x] 新增 `examples/onboarding-agent-response-demo`，从 `verification-summary.json` 渲染 Agent 四段回复。
2. [x] 新增 `scripts/render-onboarding-agent-response.sh`，接收 onboarding artifact 目录并自动读取 summary。
3. [x] `scripts/run-onboarding-ci.sh` 已在 artifact 目录中 best-effort 生成 `agent-response.txt`。
4. [x] `test/onboarding_agent_response_demo_test.sh` 覆盖用户项目失败、安装失败、通过、help 和缺失 summary。
5. [x] `test/run_onboarding_ci_test.sh` 已覆盖 passed / failed 两种路径的 `agent-response.txt` 内容和 step summary 链接。
6. [x] README、onboarding CI 模板、失败排查、接入指南和验收 CI 文档已同步为 onboarding 四件套。

已完成补充：onboarding 和 first-run 现在都能在 CI artifact 中提供 Agent 可直接读取的 `agent-response.txt`，失败消费入口保持一致。下一步建议把外部 onboarding showcase 也升级为校验四件套，确保复制型 bootstrap 在非 testloop 项目目录中同样产出回复草稿。

## 第二百八十阶段：外部 onboarding showcase 四件套校验

1. [x] `scripts/showcase-onboarding-ci-external-project.sh` 已校验 `agent-response.txt` 存在且包含 `agent_next_step=ready`。
2. [x] showcase 输出已新增 `external_onboarding_*_agent_response` 路径，Go、Node 和 all 模式都可定位回复草稿。
3. [x] `docs/onboarding-ci-external-dry-run.md` 已从三件套更新为四件套，并补充 `agent-response.txt` 路径。
4. [x] README 和 showcase 文档已说明外部 onboarding 演练会证明四件套 artifact 可用。
5. [x] 文档回归已补 `agent-response.txt` 外部 dry-run 断言。

已完成补充：复制型 onboarding bootstrap 的外部项目演练现在也证明四件套 artifact 可用，onboarding 和 first-run 的 Agent 消费入口已经基本对齐。下一步建议沉淀一份 onboarding 失败 artifact fixture，供客户端/Agent 不跑脚本也能回归 onboarding 失败消费逻辑。

## 第二百八十一阶段：onboarding 失败 artifact fixture

1. [x] 新增 `docs/fixtures/onboarding-artifacts/user-project-smoke-failed/`，沉淀 onboarding 用户项目 smoke 失败四件套。
2. [x] fixture 包包含 `verification-report.md`、`verification-summary.json`、`agent-decision.txt` 和 `agent-response.txt`。
3. [x] fixture 场景固定为用户项目 smoke exit code `7`，`agent_next_step=inspect-user-project`。
4. [x] 新增 `test/onboarding_artifact_fixtures_test.sh`，验证文件完整、summary JSON 可解析、decision/response 字段正确，并能被 onboarding 回复 demo 和目录入口消费。
5. [x] `docs/fixtures.md` 已新增 onboarding artifact fixture 小节。
6. [x] `docs/client-integration.md` 已把 CI artifact fixture 扩展为 first-run 和 onboarding 两类输入。

已完成补充：客户端/Agent 现在既可以用 first-run 六件套 fixture，也可以用 onboarding 四件套 fixture 回归 CI artifact 消费逻辑。下一步建议整理一个 v0.5.10 候选计划，把 v0.5.9 之后围绕 `agent-response.txt` 的 first-run/onboarding 收敛改动归档成一个 patch 版本。

## 第二百八十二阶段：v0.5.10 release readiness

1. [x] 新增 `docs/plan-release-notes-v0.5.10.md`，整理 v0.5.10 候选发布说明草案。
2. [x] 新增 `docs/plan-release-v0.5.10.md`，记录候选内容、已验证项、发布前门禁和正式发布前待办。
3. [x] 候选范围明确为 v0.5.9 之后 first-run / onboarding `agent-response.txt` artifact 消费体验收敛。
4. [x] 发布说明明确不扩语言、不改生成算法、不改变 MCP tool 协议。
5. [x] 最新候选提交 `44071a0` 远端 CI run `29673246805` 已通过，发布检查清单中的远端验证状态已补齐。
6. [x] 完整 release readiness 门禁已通过：脚本语法、`go test ./...`、完整 shell 矩阵、主服务/testgen 构建、help 输出、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。

已完成补充：v0.5.10 候选计划已经具备发布说明和检查清单骨架，候选计划提交 `13ea54b` 的远端 CI 已通过，本地 release readiness 门禁也已通过。下一步建议推进正式版本准备。

## 第二百八十三阶段：v0.5.10 正式版本准备

1. [x] `main.go` MCP implementation version 已更新为 `0.5.10`。
2. [x] `CHANGELOG.md` 的 `Unreleased` 内容已收敛为 `v0.5.10 - 2026-07-19`，并记录 implementation version 更新。
3. [x] README 中当前 Release、手动下载示例、Windows 下载示例和接入 bootstrap 示例已同步到 `v0.5.10`。
4. [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.5.10`。
5. [x] quickstart、first-run、verification report、verification CI、onboarding CI 和接入指南中的版本门禁已同步到 `0.5.10`。
6. [x] 测试中的版本期望已同步到 `0.5.10`。
7. [x] `docs/plan-release-notes-v0.5.10.md` 和 `docs/plan-release-v0.5.10.md` 已标记正式版本准备同步项。
8. [x] 正式版本准备本地完整验证已通过：脚本语法、`go test ./...`、完整 shell 矩阵、主服务/testgen 构建、`testloop-mcp 0.5.10` 版本输出、help 输出、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。
9. [x] 正式版本准备提交 `df4a2c3` 远端 CI run `29673498767` 已通过。
10. [x] `v0.5.10` tag 已推送，Release Artifacts run `29673555807` 已通过。
11. [x] `scripts/verify-release-assets.sh v0.5.10` 已确认 10 个 Release 资产完整。
12. [x] GitHub Release 正文已更新为正式 v0.5.10 发布说明。
13. [x] 仓库内 `Formula/testloop-mcp.rb` 已生成 v0.5.10 版本和真实 sha256。
14. [x] 仓库内 Formula 和发布记录提交 `530007e` 远端 CI run `29673720034` 已通过。
15. [x] Homebrew tap 已更新到 v0.5.10，tap commit `54e7c91`。
16. [x] Post-Release Verify run `29673822611` 已通过，覆盖资产清单和五平台安装脚本 dry run。

已完成补充：v0.5.10 发布流程已完成，tag、Release Artifacts、资产校验、GitHub Release 正文、仓库内 Formula、Homebrew tap 和 Post-Release Verify 均已收口。下一步回到主线产品价值，继续打磨真实 Agent/客户端接入体验。

## 第二百八十四阶段：Agent response artifact contract

1. [x] 新增 `docs/agent-response-artifact-contract.md`，统一 first-run 和 onboarding `agent-response.txt` 的适用入口。
2. [x] contract 固定四段结构：结论、证据、下一步、暂不做。
3. [x] contract 明确 first-run 与 onboarding 的固定证据字段差异。
4. [x] contract 固定失败读取顺序：先 `agent-response.txt`，再 decision、summary、report，旧版 first-run 再 fallback 到 `first-run-context.txt`。
5. [x] README、`docs/ci-agent-triage.md` 和 `docs/client-integration.md` 已链接统一 contract，并修正旧的 decision/context 优先顺序。
6. [x] 新增 `test/agent_response_artifact_contract_doc_test.sh`，文档测试固定 contract 入口、字段和 fixture 链接。

已完成补充：Agent/客户端现在有一份统一 contract，而不是分别从 first-run/onboarding 文档里拼读取规则。下一步建议把 contract 中的字段约束落成机器可读 manifest fixture，减少客户端测试自己写路径和字段映射。

## 第二百八十五阶段：Agent response artifact manifest

1. [x] 新增 `docs/fixtures/agent-response-artifact-manifest.json`，以机器可读形式列出 first-run 和 onboarding artifact fixture。
2. [x] manifest 固定每类 artifact 的目录、必备文件、action 字段、期望 action、失败 section、exit code、response 字段和 fallback 顺序。
3. [x] 新增 `test/agent_response_artifact_manifest_test.sh`，验证 manifest JSON 可解析、路径存在、response/decision/summary 与期望一致。
4. [x] `docs/agent-response-artifact-contract.md` 已链接 manifest 作为机器可读索引。
5. [x] `docs/fixtures.md` 和 `docs/client-integration.md` 已说明客户端可读取 manifest 来发现 artifact fixture。

已完成补充：客户端/Agent 测试现在既可以读人类 contract，也可以读机器可用 manifest 自动发现 artifact fixture。下一步建议把 manifest 消费方式加到最小客户端示例或新增小 demo，证明客户端可以从 manifest 自动枚举并校验 artifact。

## 第二百八十六阶段：Agent response manifest demo

1. [x] 新增 `examples/agent-response-manifest-demo`，读取 `agent-response-artifact-manifest.json`。
2. [x] demo 校验 manifest schema、artifact 必备文件、可选文件、response 必备字段和 fallback 顺序。
3. [x] demo 输出 first-run/onboarding artifact 的 action 字段、期望 action、失败 section、exit code、response 字段和 fallback 顺序。
4. [x] 新增 `test/agent_response_manifest_demo_test.sh`，固定成功输出和缺少参数的错误提示。
5. [x] `docs/agent-response-artifact-contract.md` 和 `docs/client-integration.md` 已加入 manifest demo 命令。

已完成补充：客户端现在有可运行示例证明如何从 manifest 自动枚举 artifact fixture 并验证消费前置条件。下一步建议把这条 manifest demo 纳入 README 的 Agent/客户端入口，形成“结构化 MCP fixture + CI artifact manifest”两条客户端回归路径。

## 第二百八十七阶段：README 客户端回归入口补齐

1. [x] README 的 Agent/客户端入口已从“真实 handler fixture 和 first-run artifact fixture”扩展为“真实 handler fixture、CI artifact fixture 和 agent-response artifact manifest”。
2. [x] README 已新增 `go run ./examples/agent-response-manifest-demo docs/fixtures/agent-response-artifact-manifest.json` 命令。
3. [x] `test/release_doc_index_test.sh` 已固定 manifest demo 命令、manifest JSON 路径和首页关键词。

已完成补充：首页现在同时暴露结构化 MCP fixture 的客户端决策 demo，以及 CI artifact manifest 的客户端消费 demo。下一步建议运行本地 gate，提交并等待远端 CI；通过后再评估是否需要把 manifest schema 固化进 JSON Schema。

## 第二百八十八阶段：Artifact manifest JSON Schema 固化

1. [x] 新增 `docs/fixtures/agent-response-artifact-manifest.schema.json`，固定 manifest v1 的必填字段、artifact kind、文件名、fallback 顺序和 first-run/onboarding 字段关系。
2. [x] `agent-response-artifact-manifest.json` 已声明 `$schema`，方便客户端定位契约文件。
3. [x] 新增 Go schema 回归测试，验证当前 manifest 可通过 schema，并覆盖 schema_version、缺必填字段、非法 kind、fallback 首项错误等负例。
4. [x] README、客户端集成说明、fixture 索引和 Agent response artifact contract 已链接 schema。

已完成补充：CI artifact manifest 现在不仅有示例数据和 demo，还有可被客户端复用的 JSON Schema。下一步建议运行本地 gate，提交并等待远端 CI；通过后继续把 schema 消费方式扩展到客户端契约测试文档，给接入方一条“下载 schema + 校验 manifest + 执行 demo”的完整回归模板。

## 第二百八十九阶段：客户端契约测试文档补齐 manifest/schema 模板

1. [x] `docs/mcp-client-contract-tests.md` 新增 CI artifact manifest 回归章节，区分 MCP 结构化返回 fixture 和 CI artifact fixture。
2. [x] 文档补充 `agent-response-artifact-manifest.json` 与 `agent-response-artifact-manifest.schema.json` 下载/校验命令。
3. [x] 文档补充无 JSON Schema 校验器时的兜底 demo：`go run ./examples/agent-response-manifest-demo docs/fixtures/agent-response-artifact-manifest.json`。
4. [x] 新增 `test/mcp_client_contract_doc_test.sh`，固定契约测试文档的 schema、demo、fallback 和 action 字段要求。

已完成补充：接入方现在可以从 MCP 客户端契约测试说明里直接复制两类回归路径：MCP structuredContent fixture，以及 CI artifact manifest/schema。下一步建议运行本地 gate，提交并等待远端 CI；通过后再评估是否需要把 README 的 CI 失败 artifact 段落也指向 manifest/schema，减少用户只看到单个 fixture 目录的概率。

## 第二百九十阶段：README CI artifact manifest/schema 入口补齐

1. [x] README 的 CI 失败 artifact 段落已从单个 first-run fixture 目录扩展到 manifest/schema 入口。
2. [x] 段落明确可通过 `agent-response-artifact-manifest.json` 一次性覆盖 first-run 和 onboarding artifact fixture。
3. [x] `test/readme_ci_snippet_test.sh` 已固定 README 中的 manifest、schema 和 first-run fixture 链接。

已完成补充：首页的 CI 失败路径现在同时服务“手动读 artifact”和“客户端自动回归”两类读者。下一步建议运行本地 gate，提交并等待远端 CI；通过后继续评估是否需要为 `agent-response-manifest-demo` 增加 README 示例输出，降低接入方判断 demo 是否正常的成本。

## 第二百九十一阶段：README manifest demo 输出样例补齐

1. [x] README 已新增 `agent-response-manifest-demo` 的最小正常输出样例。
2. [x] 输出样例固定 `manifest_schema_version=1`、`artifact_count=2`、first-run/onboarding 两类 artifact 和对应 action 字段。
3. [x] `test/release_doc_index_test.sh` 已固定 README 中的 manifest demo 输出关键词。

已完成补充：接入方现在不只知道要运行 manifest demo，也知道成功输出应该长什么样。下一步建议运行本地 gate，提交并等待远端 CI；通过后继续评估是否需要把 manifest/schema 链路加入 `docs/adopter-verification-guide.md` 的接入方一页式验收清单。

## 第二百九十二阶段：接入方一页式验收补齐 manifest/schema

1. [x] `docs/adopter-verification-guide.md` 的 CI artifact 上传章节已补 artifact manifest 消费入口。
2. [x] 指南已链接 `agent-response-artifact-manifest.schema.json`，明确 schema 固定 first-run/onboarding artifact 目录、必备文件、期望 action 和 `fallback_order`。
3. [x] 相关文档已补 `MCP 客户端契约测试说明` 和 `真实结构化 fixture`，方便从一页式指南继续进入客户端回归模板。
4. [x] `test/adopter_verification_guide_doc_test.sh` 已固定 manifest demo、schema、fallback 和相关路径。

已完成补充：首次接入路径现在覆盖安装、首跑、CI bootstrap、artifact 上传、失败分流和 artifact 客户端消费回归。下一步建议运行本地 gate，提交并等待远端 CI；通过后继续检查 `docs/quickstart.md` 是否也需要把最新的 artifact manifest/schema 入口补进 5 分钟路径。

## 第二百九十三阶段：Quickstart artifact manifest/schema 入口补齐

1. [x] `docs/quickstart.md` 已新增 `agent-response-manifest-demo` 快速验证入口。
2. [x] quickstart 已链接 `agent-response-artifact-manifest.schema.json`、接入方一页式验证指南和 MCP 客户端契约测试说明。
3. [x] 新增 `test/quickstart_doc_test.sh`，固定 quickstart 的安装、自检、客户端配置、最小闭环、演示制品和 manifest/schema 入口。

已完成补充：5 分钟接入路径现在也覆盖 artifact manifest/schema，不再只停留在本机 MCP 闭环。下一步建议运行本地 gate，提交并等待远端 CI；通过后继续检查 `docs/installation.md` 是否需要从完整安装文档指向 quickstart 已新增的 artifact 消费回归，而不是重复展开。

## 第二百九十四阶段：Installation artifact 消费回归入口补齐

1. [x] `docs/installation.md` 已在安装后自检段落补充 `agent-response-manifest-demo`。
2. [x] installation 已链接 `agent-response-artifact-manifest.schema.json`、quickstart 和 MCP 客户端契约测试说明。
3. [x] 新增 `test/installation_doc_test.sh`，固定完整安装文档中的安装、下载、Homebrew、Docker、客户端配置、自检和 manifest/schema 入口。

已完成补充：完整安装文档现在不重复展开 artifact 回归细节，但能把用户导向 quickstart 与客户端契约测试说明。下一步建议运行本地 gate，提交并等待远端 CI；通过后继续检查是否需要为 `docs/fixtures.md` 补一段 schema 变更维护规则，明确修改 manifest 时必须同步 schema 和 Go schema 测试。

## 第二百九十五阶段：Fixture manifest/schema 维护规则补齐

1. [x] `docs/fixtures.md` 已补 first-run/onboarding artifact fixture 与 manifest 的维护规则。
2. [x] 维护规则明确修改 manifest 时要同步 JSON Schema、Go schema 测试、manifest demo 输出断言和入口文档。
3. [x] `test/agent_response_artifact_manifest_test.sh` 已固定 `$schema` 指针、schema 文件存在和维护命令文档。

已完成补充：artifact manifest/schema 现在有使用入口、客户端回归模板和维护规则，后续字段调整不容易只改一处。下一步建议运行本地 gate，提交并等待远端 CI；通过后检查是否需要把 artifact manifest/schema 纳入 `docs/plan-release-notes-v0.5.11.md` 候选发布说明，开始整理下一版发布边界。

## 第二百九十六阶段：v0.5.11 候选发布说明草案

1. [x] 新增 `docs/plan-release-notes-v0.5.11.md`。
2. [x] 草案已把 v0.5.10 之后的 Agent response artifact contract、manifest/schema、demo、客户端回归模板和接入文档入口收敛为候选发布边界。
3. [x] 草案已记录 `da2efc9` 到 `7519cf2` 的远端 CI 通过证据。
4. [x] 草案明确 v0.5.11 是 Agent/客户端 artifact 消费契约 patch，不是测试生成质量升级。

已完成补充：下一版发布边界已经有草案，后续可以继续补正式发布检查清单。下一步建议运行文档链接和本地 gate，提交并等待远端 CI；通过后新增 `docs/plan-release-v0.5.11.md`，把候选内容、验证命令、正式发布前待办和 release readiness 固定成清单。

## 第二百九十七阶段：v0.5.11 候选发布检查清单

1. [x] 新增 `docs/plan-release-v0.5.11.md`。
2. [x] 清单已整理当前差异核对、候选内容、已验证命令和远端 CI 证据。
3. [x] 清单已列出 release readiness 门禁和正式发布前待办。
4. [x] `docs/plan-release-notes-v0.5.11.md` 已标记候选发布检查清单完成，并补充发布说明草案提交的远端 CI 证据。

已完成补充：v0.5.11 的候选说明和发布检查清单都已就绪。下一步建议运行文档链接和本地 gate，提交并等待远端 CI；通过后进入 release readiness 预检，但仍不提前切版本号。

## 第二百九十八阶段：v0.5.11 release readiness 预检

1. [x] 复用当前完整本地 gate：脚本语法、`go test ./...`、完整 shell 测试矩阵和 `git diff --check` 均已通过。
2. [x] 候选主服务和 `testloop-testgen` 二进制已构建到 `/tmp/testloop-mcp-v0.5.11-candidate` 和 `/tmp/testloop-testgen-v0.5.11-candidate`。
3. [x] 候选主服务 `--version` 输出仍为 `testloop-mcp 0.5.10`，确认正式版本准备前没有提前切版本号。
4. [x] 主服务和 testgen `--help` 均以 exit code `2` 输出 usage。
5. [x] `v0.5.11` darwin arm64 打包 dry-run、sha256 校验和 tarball 内容检查已通过。
6. [x] `docs/plan-release-v0.5.11.md` 和 `docs/plan-release-notes-v0.5.11.md` 已记录 readiness 预检结果。

已完成补充：v0.5.11 候选 release readiness 已通过。下一步建议运行文档链接和本地 gate，提交并等待远端 CI；通过后进入正式版本准备：更新 implementation version、收敛 CHANGELOG、同步版本引用，再复跑发布前验证。

## 第二百九十九阶段：v0.5.11 正式版本准备

1. [x] `main.go` MCP implementation version 已更新为 `0.5.11`。
2. [x] `CHANGELOG.md` 的 `Unreleased` 内容已收敛为 `v0.5.11 - 2026-07-19`，并记录 implementation version 更新。
3. [x] README 中当前 Release、手动下载示例、Windows 下载示例和接入 bootstrap 示例已同步到 `v0.5.11`。
4. [x] `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例已同步到 `v0.5.11`。
5. [x] quickstart、first-run、verification report、verification CI、onboarding CI、接入指南和相关测试期望已同步到 `0.5.11`。
6. [x] `docs/plan-release-v0.5.11.md` 和 `docs/plan-release-notes-v0.5.11.md` 已标记正式版本准备同步项。

已完成补充：v0.5.11 正式版本准备的文件同步已完成。下一步建议重新运行完整本地验证和 release readiness，提交并等待远端 CI；CI 通过后再打 `v0.5.11` tag。

## 第三百阶段：v0.5.11 正式版本准备验证

1. [x] 正式版本准备后已复跑脚本语法检查、`go test ./...`、完整 shell 测试矩阵和 `git diff --check`。
2. [x] 正式版本准备后已构建 `/tmp/testloop-mcp-v0.5.11-release-prep` 和 `/tmp/testloop-testgen-v0.5.11-release-prep`。
3. [x] `/tmp/testloop-mcp-v0.5.11-release-prep --version` 输出 `testloop-mcp 0.5.11`。
4. [x] 主服务和 testgen `--help` 均以 exit code `2` 输出 usage。
5. [x] `v0.5.11` darwin arm64 打包 dry-run、sha256 校验和 tarball 内容检查已通过。
6. [x] `docs/plan-release-v0.5.11.md` 和 `docs/plan-release-notes-v0.5.11.md` 已记录正式版本准备验证结果。

已完成补充：v0.5.11 正式版本准备本地验证已经通过。下一步建议提交版本准备改动并等待远端 CI；通过后打 `v0.5.11` tag、等待 Release Artifacts、验证资产、更新 GitHub Release 和 Homebrew tap。

## 第三百零一阶段：v0.5.11 正式发布收敛

1. [x] 版本准备提交 `473e764` 已推送，远端 CI run `29675557908` passed。
2. [x] 创建并推送 tag `v0.5.11`。
3. [x] Release Artifacts tag-push run `29675619230` 因 runner 长时间 queued 已取消，改用手动 dispatch run `29676083347` 完成发布资产生成。
4. [x] Release Artifacts run `29676083347` passed，五个平台资产和对应 `.sha256` 已上传。
5. [x] `scripts/verify-release-assets.sh v0.5.11` 验证 release 页面包含 10 个必需资产。
6. [x] GitHub Release `v0.5.11` 正文已更新为正式发布说明，并标记为 latest。
7. [x] `scripts/generate-homebrew-formula.sh v0.5.11` 已更新仓库内 Formula，`ruby -c Formula/testloop-mcp.rb` 通过。
8. [x] Formula/发布记录提交 `aea0330` 已推送，远端 CI run `29677030693` passed。
9. [x] Homebrew tap 已升级到 `0.5.11`，tap commit `e810f60` 已推送；本机 tap 快进到该提交后 `brew audit --formula --strict sleticalboy/tap/testloop-mcp` 通过。
10. [x] Post-Release Verify run `29679705790` passed，资产清单和五平台安装脚本 dry run 全部通过。

已完成补充：v0.5.11 已完成正式 GitHub Release、仓库内 Formula、Homebrew tap 更新和 Post-Release Verify 五平台安装脚本 dry run。本机直连 GitHub release asset 下载仍有网络波动，因此发布安装门禁以 GitHub runner 上的 Post-Release Verify 为准。下一步建议提交本发布记录并等待 main CI，通过后回到功能侧继续推进真实项目闭环样本或下一版候选范围。

## 第三百零二阶段：Java StatusChecker 行号驱动输入修复

1. [x] 用 RocketMQ Java client `StatusCheckerTest` + JaCoCo 生成 `StatusChecker.java` top4 coverage task，候选覆盖 `MESSAGE_NOT_FOUND` receive-message return、`MESSAGE_BODY_EMPTY` error path 和 switch default。
2. [x] 初始验证暴露 4/4 `failed/needs_better_input`：生成测试虽然能编译运行，但默认使用 `Code.OK`，没有命中目标行。
3. [x] 生成器已按 `StatusChecker.check` 的 line range 映射真实 protobuf `Code`、request 类型和预期异常；default 分支使用合法 `NOT_IMPLEMENTED`，避免 `UNRECOGNIZED` 在 `getNumber()` 前提前抛错。
4. [x] 异常分支生成 `PayloadEmptyException` / `UnsupportedException` 等具体断言，并补通用 `ClientException` catch，避免 Java checked exception 编译失败。
5. [x] 复跑 RocketMQ Java client top4：脚本通过，`status_counts={"passed":4}`、`action_counts={"ready":4}`、`zero_skip=4`、`skipped_total=0`。
6. [x] 本仓库本地 gate 已通过：`go test ./...`、`sh test/docs_links_test.sh`、`sh test/release_doc_index_test.sh`、`git diff --check`。
7. [x] 提交 `c43c3e8` 已推送，远端 CI run `29680739466` passed。

已完成补充：RocketMQ `StatusChecker.check` 不再用默认 `Code.OK` 伪造通过测试；生成器现在能按 JaCoCo 行段选择真实 protobuf `Code`、request 类型和 checked exception，四个 top task 都确认目标行命中。下一步继续把这组样本纳入轻量 Java regression smoke，避免后续只靠临时 `/tmp` JSONL 复验。

## 第三百零三阶段：Java regression smoke 补齐 RocketMQ StatusChecker

1. [x] `scripts/validate-java-regression-samples.sh` 已新增 RocketMQ StatusChecker 样本配置：项目目录、任务 JSONL、任务 ID、Maven 覆盖率命令和 JaCoCo XML 路径均可通过 `TESTLOOP_JAVA_REGRESSION_ROCKETMQ_*` 覆盖。
2. [x] 默认 regression 矩阵已追加 `rocketmq-statuschecker-ready-hit`，固定 `junit-272/junit-273/junit-418/junit-826` 四个 ready-hit 样本。
3. [x] 完整 Java regression 已通过，输出目录 `/tmp/testloop-java-regression-20260719170220`；其中 RocketMQ StatusChecker summary 为 `status_counts={"passed":4}`、`action_counts={"ready":4}`、`zero_skip=4`、`skipped_total=0`。
4. [x] README、固定 smoke 文档和真实项目验证报告已同步新增 RocketMQ StatusChecker regression 入口。
5. [x] 本仓库本地 gate 已通过：`bash -n scripts/validate-java-regression-samples.sh scripts/validate-regression-smoke.sh`、`go test ./...`、`sh test/docs_links_test.sh`、`sh test/release_doc_index_test.sh`、`sh test/verification_ci_doc_test.sh`、`git diff --check`。
6. [x] 提交 `95a3fbb` 已推送，远端 CI run `29681019662` passed。

已完成补充：Java regression smoke 现在同时覆盖 Commons Lang ready-hit、Commons Codec unreachable、Commons Lang internal 和 RocketMQ StatusChecker line-specific ready-hit。下一步继续评估是否把 RocketMQ 任务 JSONL 从 `/tmp` 迁入仓库 fixture，降低跨机器复跑门槛。

## 第三百零四阶段：RocketMQ StatusChecker 任务 fixture 入仓

1. [x] 新增 `testdata/java-rocketmq-statuschecker/statuschecker-tasks.jsonl`，固定 `junit-272/junit-273/junit-418/junit-826` 四个 StatusChecker coverage task 输入。
2. [x] `scripts/validate-java-regression-samples.sh` 默认任务文件已从 `/tmp/testloop-rocketmq-java-statuschecker-tasks.jsonl` 改为仓库内 fixture。
3. [x] `docs/regression-smoke.md` 已同步 Java / RocketMQ StatusChecker 默认 JSONL 路径。
4. [x] 新增 `test/java_regression_fixture_test.sh` 并纳入 CI，固定仓库内 StatusChecker JSONL 的 4 个任务 ID、目标方法、行段和推荐测试文件。
5. [x] 复跑 Java regression smoke，确认仓库内 fixture 可替代 `/tmp` 任务文件；输出目录 `/tmp/testloop-java-regression-20260719171247`，RocketMQ summary 为 `status_counts={"passed":4}`、`action_counts={"ready":4}`、`zero_skip=4`、`skipped_total=0`。
6. [x] 本仓库本地 gate 已通过：`sh test/java_regression_fixture_test.sh`、`bash -n scripts/validate-java-regression-samples.sh scripts/validate-regression-smoke.sh test/java_regression_fixture_test.sh`、`go test ./...`、`sh test/docs_links_test.sh`、`sh test/release_doc_index_test.sh`、`sh test/verification_ci_doc_test.sh`、`git diff --check`。
7. [x] 提交 `2b33c5e` 已推送，远端 CI run `29681320483` passed；CI 已执行新增 `test/java_regression_fixture_test.sh`。

已完成补充：RocketMQ StatusChecker regression 已从本机临时 `/tmp` task 文件迁入仓库 `testdata`，并有 CI 级 JSONL 结构测试保护。下一步继续寻找下一组收益最高的 regression 输入。

## 第三百零五阶段：Java Commons regression task fixture 入仓

1. [x] 新增 `testdata/java-commons-lang/ready-hit-tasks.jsonl`，固定 Commons Lang `junit-44/junit-50` ready-hit 输入。
2. [x] 新增 `testdata/java-commons-lang/manual-internal-tasks.jsonl`，固定 Commons Lang `junit-52` internal 手审输入。
3. [x] 新增 `testdata/java-commons-codec/unreachable-tasks.jsonl`，固定 Commons Codec `junit-130` unreachable 输入。
4. [x] `scripts/validate-java-regression-samples.sh` 的 Commons Lang / Codec 默认任务文件已从 `/tmp` 改为仓库内 fixture。
5. [x] `docs/regression-smoke.md` 和 changelog 已同步新的默认 JSONL 路径。
6. [x] `test/java_regression_fixture_test.sh` 已扩展覆盖三组 Commons JSONL 和 RocketMQ JSONL。
7. [x] 完整 Java regression 已通过，输出目录 `/tmp/testloop-java-regression-20260719172422`；Commons Lang ready-hit 为 `passed=2/ready=2/zero_skip=2`，Commons Codec unreachable 为 `passed=1/manual_review_unreachable=1`，Commons Lang internal 为 `passed=1/manual_review_internal=1`，RocketMQ StatusChecker 为 `passed=4/ready=4/zero_skip=4`。
8. [x] 本仓库本地 gate 已通过：`sh test/java_regression_fixture_test.sh`、`bash -n scripts/validate-java-regression-samples.sh scripts/validate-regression-smoke.sh test/java_regression_fixture_test.sh`、`go test ./...`、`sh test/docs_links_test.sh`、`sh test/release_doc_index_test.sh`、`git diff --check`。
9. [x] 提交 `e317898` 已推送，远端 CI run `29681646727` passed。

已完成补充：Java regression smoke 的默认任务输入已经全部迁入仓库 `testdata`；跨机器复跑仍依赖真实项目 checkout，但不再依赖本机历史 `/tmp` JSONL。下一步继续评估是否将 JS/Python regression 中仍依赖 `/tmp` 或运行时生成的输入做同样收敛。

## 第三百零六阶段：JS ready task fixture 入仓

1. [x] 新增 `testdata/js-ip2region/ready-hit-tasks.jsonl`，固定 JS/ip2region `jest-1/jest-2` ready-hit 输入。
2. [x] `scripts/validate-js-regression-samples.sh` 的 ip2region 默认任务文件已从 `/tmp/testloop-ip2region-js-jest-top2-current.jsonl` 改为仓库内 fixture。
3. [x] `docs/regression-smoke.md` 和 changelog 已同步新的默认 JSONL 路径。
4. [x] 新增 `test/js_regression_fixture_test.sh` 并纳入 CI，固定 JS/ip2region fixture 的 ID、framework、目标方法、行段和推荐测试文件。
5. [x] JS regression smoke 已通过，输出目录 `/tmp/testloop-js-regression-20260719173527`；ip2region ready 样本从仓库内 `testdata/js-ip2region/ready-hit-tasks.jsonl` 读取并达到 `passed=2/ready=2/zero_skip=2`。
6. [x] 尝试迁入 Python/Click ready task fixture 时发现当前 Click HEAD 与旧 task 行段不兼容，`pytest-1/pytest-3` 从 ready 退化为 `repair_generated_test`；因此本阶段不切 Python 默认任务文件，避免把漂移样本放入默认 smoke。
7. [x] 本地 gate 已通过：`sh test/js_regression_fixture_test.sh`、`sh test/java_regression_fixture_test.sh`、`bash -n scripts/validate-js-regression-samples.sh scripts/validate-py-regression-samples.sh test/js_regression_fixture_test.sh`、`go test ./...`、`sh test/docs_links_test.sh`、`sh test/release_doc_index_test.sh`、`sh test/verification_ci_doc_test.sh`、`git diff --check`。
8. [x] 提交 `c1dd5ef` 已推送，远端 CI run `29682017836` passed。

已完成补充：JS/ip2region ready task 输入已迁入仓库 `testdata`，并有 CI 级 fixture 测试保护。Python/Click 在本轮尝试中暴露出旧 task 与当前 Click HEAD / 8.2.1 均不兼容的问题：当前 Click 已改动 public/private stream helper 和测试目录结构，旧 `pytest-1/pytest-3` 会退化为 `repair_generated_test`。下一步应单独做 Click regression 重建：先为 Click 选择固定 commit，再用当前生成器重新挑选 ready 样本，而不是直接迁旧 `/tmp` JSONL。

## 第三百零七阶段：Python Click regression task 重建入仓

1. [x] `tools/py_project_validation_integration_test.go` 新增 `TESTLOOP_VALIDATE_PY_LIST_TASKS_ONLY`，可只输出筛选后的 coverage task JSONL，不执行 `generate_tests -> run_tests`。
2. [x] `scripts/validate-py-coverage-top-tasks.sh` 帮助文档已补 list-only 开关，后续重建 Python fixture 时可先生成候选任务。
3. [x] 基于 Click `8.2.1` 的 `tests/test_utils.py` 覆盖率窗口生成 top40 候选任务：`/tmp/testloop-click-821-pytest-test-utils-top40-tasks.jsonl`。
4. [x] 验证 Click utils 七个 ready 样本通过：`pytest-19/pytest-20/pytest-21/pytest-22/pytest-23/pytest-32/pytest-33`，输出 `/tmp/testloop-click-821-ready7-validation.jsonl`，结果为 `passed=7/ready=7/zero_skip=7/skipped_total=0`。
5. [x] 新增 `testdata/py-click/ready-hit-tasks.jsonl`，固定 Click `get_binary_stream`、`get_text_stream`、`get_app_dir`、`make_str`、`PacifyFlushWrapper.flush`、`safecall` 和 `_expand_args` ready-hit 输入。
6. [x] `scripts/validate-py-regression-samples.sh` 的 Click 默认任务文件从 `/tmp/testloop-click-pytest-top5-regression.jsonl` 改为仓库内 `testdata/py-click/ready-hit-tasks.jsonl`，默认 ready ids 改为已验证的七个样本。
7. [x] 新增 `test/py_regression_fixture_test.sh` 并纳入 CI，固定 Python Click fixture 的 ID、framework、目标方法、行段和推荐测试文件。
8. [x] Python regression smoke 已通过，输出目录 `/tmp/testloop-py-regression-20260719175459`；Click ready 为 `passed=7/ready=7/zero_skip=7`，Python internal 为 `passed=1/manual_review_internal=1`，haoy-apk-station environment/external-service/database 样本均符合预期 action。
9. [x] 本地 gate 已通过：`go test ./...`、`sh test/py_regression_fixture_test.sh`、`sh test/js_regression_fixture_test.sh`、`sh test/java_regression_fixture_test.sh`、`bash -n scripts/validate-py-coverage-top-tasks.sh scripts/validate-py-regression-samples.sh test/py_regression_fixture_test.sh`、`sh test/docs_links_test.sh`、`sh test/release_doc_index_test.sh`、`sh test/verification_ci_doc_test.sh`、`git diff --check`。
10. [x] 提交 `e669ed9` 已推送，远端 CI run `29682473349` passed。

已完成补充：Python regression smoke 不再依赖旧 `/tmp` Click task JSONL，且 Click ready 样本已从漂移的 parser 私有 helper 行段迁到当前 Click `8.2.1` 可稳定生成并运行的 utils 任务。下一步继续收敛剩余 regression 输入中仍运行时生成或依赖外部路径的样本，优先评估 JS/Python manual-review fixture 是否值得迁入仓库静态 JSONL。

## 第三百零八阶段：仓库内 manual-review regression task 入仓

1. [x] 新增 `testdata/js-no-runtime/no-runtime-tasks.jsonl`，固定仓库内 TypeScript 纯类型文件 `jest-no-runtime-1 -> manual_review_no_runtime`。
2. [x] 新增 `testdata/js-internal/internal-tasks.jsonl`，固定仓库内未导出 helper `jest-internal-1 -> manual_review_internal`。
3. [x] 新增 `testdata/py-internal/internal-tasks.jsonl`，固定 Python name-mangled private method `pytest-internal-1 -> manual_review_internal`。
4. [x] `scripts/validate-js-regression-samples.sh` 的 JS no-runtime/internal 默认任务文件已从运行时生成改为仓库内静态 JSONL。
5. [x] `scripts/validate-py-regression-samples.sh` 的 Python internal 默认任务文件已从运行时生成改为仓库内静态 JSONL。
6. [x] `test/js_regression_fixture_test.sh` 和 `test/py_regression_fixture_test.sh` 已扩展覆盖新增 JSONL 的 ID、framework、目标、行段和推荐测试文件。
7. [x] JS regression smoke 已通过，输出目录 `/tmp/testloop-js-regression-20260719181014`；仓库内 no-runtime/internal 样本分别达到 `passed=1/manual_review_no_runtime=1`、`passed=1/manual_review_internal=1`。
8. [x] Python regression smoke 已通过，输出目录 `/tmp/testloop-py-regression-20260719181015`；仓库内 Python internal 样本达到 `passed=1/manual_review_internal=1`。
9. [x] 本地 gate 已通过：`go test ./...`、`sh test/js_regression_fixture_test.sh`、`sh test/py_regression_fixture_test.sh`、`sh test/java_regression_fixture_test.sh`、`bash -n scripts/validate-js-regression-samples.sh scripts/validate-py-regression-samples.sh test/js_regression_fixture_test.sh test/py_regression_fixture_test.sh`、`sh test/docs_links_test.sh`、`sh test/release_doc_index_test.sh`、`sh test/verification_ci_doc_test.sh`、`git diff --check`。
10. [x] 提交 `b74fe40` 已推送，远端 CI run `29682968710` passed。

已完成补充：仓库内 regression fixture 现在基本不再依赖运行时 JSONL 生成；`fixture-task-jsonl.py` 仍保留给外部真实项目样本。下一步继续评估 mcp-hub 这类真实项目任务是否也能用相对路径安全迁入仓库。

## 第三百零九阶段：JS mcp-hub regression task 入仓

1. [x] 新增 `testdata/js-mcp-hub/repair-tasks.jsonl`，固定 `ConfigManager.loadConfig` 三个历史 repair 回归样本。
2. [x] 新增 `testdata/js-mcp-hub/env-tasks.jsonl`，固定 `EnvResolver._resolveStringWithPlaceholders` 两个环境变量/命令错误路径样本。
3. [x] 新增 `testdata/js-mcp-hub/devwatcher-tasks.jsonl`，固定 `DevWatcher.stop/start` 生命周期样本。
4. [x] 新增 `testdata/js-mcp-hub/sse-tasks.jsonl`，固定 `SSEManager.setupAutoShutdown/addConnection/sendToClient` 四个 SSE 生命周期样本。
5. [x] 新增 `testdata/js-mcp-hub/workspace-tasks.jsonl`，固定 `WorkspaceCacheManager` ready 和 `manual_review_environment` 样本。
6. [x] `scripts/validate-js-regression-samples.sh` 的 mcp-hub 默认任务文件已从运行时生成改为仓库内静态 JSONL，并保留 `TESTLOOP_JS_REGRESSION_MCP_HUB_*_TASKS_FILE` 覆盖入口。
7. [x] `test/js_regression_fixture_test.sh` 已扩展覆盖五组 mcp-hub JSONL 的 ID、framework、目标、行段和推荐测试文件。
8. [x] JS regression smoke 已通过，输出目录 `/tmp/testloop-js-regression-20260719182143`；mcp-hub repair/env/devwatcher/sse/workspace-ready 全部为 `passed/ready`，workspace manual 为 `passed/manual_review_environment`。
9. [x] 本地 gate 已通过：`go test ./...`、`sh test/js_regression_fixture_test.sh`、`sh test/py_regression_fixture_test.sh`、`sh test/java_regression_fixture_test.sh`、`bash -n scripts/validate-js-regression-samples.sh test/js_regression_fixture_test.sh`、`sh test/docs_links_test.sh`、`sh test/release_doc_index_test.sh`、`sh test/verification_ci_doc_test.sh`、`git diff --check`。
10. [x] 提交 `ceca15c` 已推送，远端 CI run `29683297970` passed。

已完成补充：JS regression smoke 的默认任务输入已经全部迁入仓库 `testdata`，外部 mcp-hub 项目仍需要本机 checkout，但不再依赖 helper 临时生成 task JSONL。下一步运行本地 gate、提交并等待远端 CI。

## 第三百一十阶段：Python haoy-apk-station regression task 入仓

1. [x] 新增 `testdata/py-haoy-apk-station/environment-tasks.jsonl`，固定 `serve_frontend -> manual_review_environment`。
2. [x] 新增 `testdata/py-haoy-apk-station/external-service-tasks.jsonl`，固定 `download_apk -> manual_review_external_service`。
3. [x] 新增 `testdata/py-haoy-apk-station/database-tasks.jsonl`，固定 `delete_app -> manual_review_database`。
4. [x] `scripts/validate-py-regression-samples.sh` 的 haoy-apk-station 默认任务文件已从运行时生成改为仓库内静态 JSONL，并保留 `TESTLOOP_PY_REGRESSION_APK_STATION*_TASKS_FILE` 覆盖入口。
5. [x] `test/py_regression_fixture_test.sh` 已扩展覆盖三组 haoy-apk-station JSONL 的 ID、framework、目标、行段和推荐测试文件。
6. [x] Python regression smoke 已通过，输出目录 `/tmp/testloop-py-regression-20260719182652`；environment/database 分别为 `passed/manual_review_environment`、`passed/manual_review_database`，external-service 为 `failed/manual_review_external_service`。
7. [x] 本地 gate 已通过：`go test ./...`、`sh test/py_regression_fixture_test.sh`、`sh test/js_regression_fixture_test.sh`、`sh test/java_regression_fixture_test.sh`、`bash -n scripts/validate-py-regression-samples.sh test/py_regression_fixture_test.sh`、`sh test/docs_links_test.sh`、`sh test/release_doc_index_test.sh`、`sh test/verification_ci_doc_test.sh`、`git diff --check`。
8. [x] 提交 `fe92caa` 已推送，远端 CI run `29683433546` passed。

已完成补充：Java、JS、Python regression smoke 的默认任务输入已经全部迁入仓库 `testdata`，真实项目 checkout 仍是运行前提，但脚本不再依赖 `/tmp` 历史 JSONL 或运行时 helper 生成默认 task。下一步清理/弱化 `fixture-task-jsonl.py` 的默认路径定位，避免文档误导它仍是 smoke 必需步骤。

## 第三百一十一阶段：全量静态 regression smoke 证据归档

1. [x] 运行全量回归 smoke：`scripts/validate-regression-smoke.sh`，输出目录 `/tmp/testloop-regression-smoke-20260719184046`。
2. [x] Java 回归样本全部符合预期：Commons Lang ready `passed=2/ready=2`，Commons Codec unreachable `passed=1/manual_review_unreachable=1`，Commons Lang internal `passed=1/manual_review_internal=1`，RocketMQ StatusChecker ready `passed=4/ready=4`。
3. [x] JS 回归样本全部符合预期：ip2region ready `passed=2/ready=2`，仓库内 no-runtime/internal 分别为 `passed=1/manual_review_no_runtime=1`、`passed=1/manual_review_internal=1`，mcp-hub repair/env/devwatcher/sse/workspace-ready 共 `passed=13/ready=13`，workspace manual 为 `passed=1/manual_review_environment=1`。
4. [x] Python 回归样本符合预期：Click ready `passed=7/ready=7`，仓库内 internal 为 `passed=1/manual_review_internal=1`，haoy-apk-station environment/database 分别为 `passed=1/manual_review_environment=1`、`passed=1/manual_review_database=1`，external-service 保持预期分流 `failed=1/manual_review_external_service=1`。
5. [x] 这次全量 smoke 证明默认 regression 输入已经从临时 `/tmp` JSONL 收敛到仓库内 `testdata` fixture；后续维护重点转为 fixture 重建入口、文档语义和真实项目路径可配置性。

已完成补充：静态 fixture 迁移已经有一次跨 Java/JS/Python 的端到端验证证据。下一步继续清理 `scripts/fixture-task-jsonl.py` 的定位：它应被描述为维护者重建 fixture 的辅助工具，而不是默认 regression smoke 的前置步骤。

## 第三百一十二阶段：fixture 重建入口定位清理

1. [x] `scripts/fixture-task-jsonl.py` 新增脚本级说明，明确默认 regression smoke 读取仓库内 `testdata/` 静态 JSONL，本脚本只用于维护者重建或新增 fixture 输入。
2. [x] `scripts/fixture-task-jsonl.py --help` 已补充重建语义、示例命令和 `output` 通常应写入 `testdata/` 的参数说明。
3. [x] 新增 `test/fixture_task_jsonl_script_test.sh`，固定 help 输出中的维护者定位、默认 smoke 读取路径和输出路径语义，并纳入 GitHub Actions CI。
4. [x] `docs/regression-smoke.md` 将“重建 fixture 输入”和“关键 runner”拆成独立章节，避免把 helper 误读成默认 smoke 前置步骤。
5. [x] `docs/showcase.md` 已把 deep regression smoke 描述从“历史 JSONL 样本”调整为“仓库内静态 JSONL 样本”。
6. [x] `CHANGELOG.md` 已同步 fixture 重建工具定位收紧。

已完成补充：默认 smoke 的运行入口和 fixture 重建入口已经分清。下一步收益最高的是继续降低真实项目路径依赖：先补一个 regression preflight/doctor，启动前检查默认项目目录、任务 JSONL 和关键 runner 命令是否存在，并输出可操作的缺失项，而不是等到长 smoke 半途失败。

## 第三百一十三阶段：regression smoke preflight

1. [x] 新增 `scripts/validate-regression-preflight.sh`，在不运行覆盖率、不生成测试、不执行真实项目测试的前提下，快速检查启用语言需要的真实项目目录、仓库内静态 JSONL fixture 和常用命令。
2. [x] `scripts/validate-regression-smoke.sh` 默认先执行 preflight；新增 `TESTLOOP_REGRESSION_SKIP_PREFLIGHT=true` 可临时跳过。
3. [x] 新增 `test/regression_preflight_test.sh`，覆盖 help 文案、全语言 skip 成功路径和缺失 Java 项目目录的快速失败输出。
4. [x] GitHub Actions CI 已纳入 `test/regression_preflight_test.sh`。
5. [x] `docs/regression-smoke.md`、`README.md`、`docs/showcase.md` 和 `CHANGELOG.md` 已同步 preflight 默认行为。
6. [x] 本机 preflight 已通过：Java/JS/Python 默认项目目录、静态 JSONL 和常用命令均满足当前全量 smoke 前置条件。
7. [x] 带 preflight 的全量 regression smoke 已通过，输出目录 `/tmp/testloop-regression-smoke-20260719185952`。

已完成补充：长 smoke 现在有启动前诊断层，跨机器复跑时会先报缺失目录、fixture 或命令。下一步应继续增强 preflight 的机器可读输出：增加 JSON summary 模式，方便 Agent 在无法复跑全量 smoke 时直接把缺失项转成用户可执行的准备清单。

## 第三百一十四阶段：preflight JSON summary

1. [x] `scripts/validate-regression-preflight.sh` 新增 `TESTLOOP_REGRESSION_PREFLIGHT_FORMAT=text|json`，默认继续保持人类可读 text 输出。
2. [x] JSON 模式 stdout 输出单个 JSON summary，包含 `ok`、`missing_count`、`missing` 和 `checks`，退出码仍按是否缺失前置条件决定。
3. [x] text 模式保持原有 `preflight: <language>`、`missing: ...` 和 `regression preflight passed/failed` 输出。
4. [x] `test/regression_preflight_test.sh` 已覆盖 JSON 成功路径和缺失目录的结构化 `missing` 内容。
5. [x] `docs/regression-smoke.md` 和 `CHANGELOG.md` 已同步 JSON 模式。
6. [x] 本地 gate 已通过：`sh test/regression_preflight_test.sh`、`TESTLOOP_REGRESSION_PREFLIGHT_FORMAT=json scripts/validate-regression-preflight.sh`、`bash -n scripts/validate-regression-preflight.sh scripts/validate-regression-smoke.sh test/regression_preflight_test.sh`、`go test ./...`、文档 gate、`git diff --check`。

已完成补充：Agent 现在可以用 JSON 模式把 smoke 前置缺失项转成准备清单。下一步应补一个小的 markdown 渲染器或示例，把 JSON preflight 输出转换成中文可执行准备步骤，用于用户机器缺依赖时直接回复。

## 第三百一十五阶段：preflight 中文准备清单渲染

1. [x] 新增 `scripts/render-regression-preflight-report.py`，支持从文件或 stdin 读取 preflight JSON summary。
2. [x] preflight 通过时输出中文 Markdown，包含状态、缺失项数量和继续运行 `scripts/validate-regression-smoke.sh` 的命令。
3. [x] preflight 未通过时按 `command`、`dir`、`file` 分组输出缺失命令、缺失目录和缺失 JSONL fixture，并提示可通过 `TESTLOOP_*_REGRESSION_*` 环境变量改到本机路径。
4. [x] 新增 `test/regression_preflight_report_test.sh`，覆盖通过报告、缺失项分组和 stdin 输入。
5. [x] GitHub Actions CI 已纳入 `test/regression_preflight_report_test.sh`。
6. [x] `docs/regression-smoke.md` 和 `CHANGELOG.md` 已同步 JSON 到 Markdown 的使用方式。
7. [x] 本地 gate 已通过：`sh test/regression_preflight_report_test.sh`、`sh test/regression_preflight_test.sh`、`python3 -m py_compile scripts/render-regression-preflight-report.py scripts/fixture-task-jsonl.py`、shell 语法检查、文档 gate、`go test ./...`、`git diff --check`。
8. [x] 真实管道已验证：`TESTLOOP_REGRESSION_PREFLIGHT_FORMAT=json scripts/validate-regression-preflight.sh | scripts/render-regression-preflight-report.py -` 输出中文 Markdown 且状态为通过。

已完成补充：preflight 已形成“text 给人看、JSON 给 Agent 消费、Markdown 给用户回复”的三层输出。下一步应把 release readiness 检查重新跑一遍，确认最近新增的维护者工具没有破坏发布前门禁，并把 CI 结果归档到 roadmap。

## 第三百一十六阶段：维护者工具发布前门禁复验

1. [x] 远端 CI 已通过：提交 `f9295b4` 对应 GitHub Actions CI run `29684888366` passed。
2. [x] shell 语法检查已通过：`find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`。
3. [x] Go 全量测试已通过：`go test ./...`。
4. [x] 完整 shell 测试矩阵已通过：`for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`。
5. [x] 二进制构建已通过：`go build -o /tmp/testloop-mcp-current-candidate .` 和 `go build -o /tmp/testloop-testgen-current-candidate ./cmd/testgen`。
6. [x] help/version 检查已通过：`/tmp/testloop-mcp-current-candidate --version` 输出 `testloop-mcp 0.5.11`，说明当前 main 仍未切正式新版本号；两个 help 命令均输出 usage 且 exit code 为 `2`。
7. [x] darwin arm64 打包 dry-run 已通过：`TESTLOOP_MCP_DIST_DIR=/tmp/testloop-current-candidate-dist scripts/package-release-asset.sh v0.5.12 darwin_arm64 darwin arm64`。
8. [x] sha256 校验通过：`cd /tmp/testloop-current-candidate-dist && shasum -a 256 -c testloop-mcp_v0.5.12_darwin_arm64.tar.gz.sha256`。
9. [x] tarball 内容检查通过，包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
10. [x] `git diff --check` 通过；工作区只剩既有未跟踪 `raw.md`。

已完成补充：最近新增的 regression preflight、JSON summary 和中文报告渲染器没有破坏发布前门禁。下一步应整理 v0.5.12 候选发布说明草案，把本轮静态 fixture 迁移、preflight 诊断层和 Agent 可读报告归档为一个清晰的候选版本范围，但暂不切版本号、不打 tag。

## 第三百一十七阶段：v0.5.12 候选发布文档

1. [x] 新增 `docs/plan-release-notes-v0.5.12.md`，归档 v0.5.11 之后的 regression fixture 静态化、Click fixture 重建、preflight 诊断层、JSON summary 和中文准备清单渲染器。
2. [x] 新增 `docs/plan-release-v0.5.12.md`，记录候选内容、已验证命令、远端 CI run、发布前门禁和正式发布前待办。
3. [x] 候选文档明确当前阶段不切版本号、不打 tag、不更新 Homebrew tap。
4. [x] 候选发布重点保持项目定位：维护者 regression smoke 更可复跑，Agent 可读诊断更稳定；不宣传成新增语言支持或测试生成算法大改。

已完成补充：v0.5.12 候选发布范围已经成文。下一步应跑文档 gate、提交并等待 CI；通过后若继续推进，再进入正式版本准备前的最后核对，而不是直接发布。

## 第三百一十八阶段：v0.5.12 正式版本准备

1. [x] 已确认本地 tag 和 GitHub Release 最新均为 `v0.5.11`，`v0.5.12` 尚不存在。
2. [x] `main.go` MCP implementation version 已更新为 `0.5.12`，`main_test.go` 版本断言已同步。
3. [x] `CHANGELOG.md` 的 `Unreleased` 内容已收敛为 `v0.5.12 - 2026-07-19`，并记录 implementation version 更新。
4. [x] README、installation、quickstart、first-run、verification report、verification CI、onboarding CI、接入指南和相关测试期望已同步到 `0.5.12` / `v0.5.12`。
5. [x] `docs/fixtures/onboarding-artifacts/user-project-smoke-failed/verification-summary.json` 的版本输出 fixture 已同步到 `testloop-mcp 0.5.12`。
6. [x] `docs/plan-release-v0.5.12.md` 和 `docs/plan-release-notes-v0.5.12.md` 已标记正式版本准备同步项。
7. [x] 本地验证已通过：版本相关文档/脚本测试、文档 gate、`go test ./...`、完整 shell 矩阵、主服务/testgen 构建、`testloop-mcp 0.5.12` 版本输出、help 输出、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。
8. [x] 提交 `4815012` 已推送，远端 CI run `29685640306` passed。

已完成补充：v0.5.12 正式版本准备的文件同步、本地验证和远端 CI 已完成。下一步是打 `v0.5.12` tag、等待 Release Artifacts、验证资产、更新 GitHub Release 和 Homebrew tap；这属于正式发布操作，需要明确确认后再执行。

## 第三百一十九阶段：v0.5.12 正式发布完成

1. [x] `v0.5.12` tag 已推送，Release Artifacts run `29688663889` passed。
2. [x] `scripts/verify-release-assets.sh v0.5.12` 已验证 10 个 Release 资产完整。
3. [x] GitHub Release 正文已更新为正式说明。
4. [x] 仓库内 Formula 已更新到 `0.5.12`，提交 `2cea4b8` 远端 CI run `29688905594` passed。
5. [x] Homebrew Tap workflow run `29688974741` 因未配置 `HOMEBREW_TAP_TOKEN` 跳过 PR；随后使用本地脚本直接更新 `sleticalboy/homebrew-tap` 并推送 commit `7d78be8`。
6. [x] Post-Release Verify run `29689015033` passed，覆盖资产清单和五平台安装脚本 dry run。
7. [x] 本机 Homebrew tap 已快进到 `7d78be8`，`brew info` 显示 stable `0.5.12`，`brew audit --formula --strict` 通过。
8. [x] 发布完成记录提交 `0834f3f` 已推送，远端 CI run `29689234071` passed。

已完成补充：v0.5.12 已完成正式发布、Release Artifacts、资产清单校验、GitHub Release 正文、仓库内 Formula、Homebrew tap、Post-Release Verify 和本机 Homebrew tap 抽查。下一步回到功能侧，继续让 Agent/LLM provider 消费的上下文与静态生成结果保持一致。

## 第三百二十阶段：JS/TS imported type context 与 static 生成对齐

1. [x] `BuildGenerationContext` 的 JS/TS 路径复用相对 import 类型解析能力，将可解析的本地 imported type 声明合并到 `TSTypeDecls`。
2. [x] context 中已解析 imported type 不再输出 `{ ok: true }` fallback note。
3. [x] 外部 LLM provider 的 request JSON 中，`context.targets[].payload_notes` 和 `static_code` 对已解析 imported type 保持一致。
4. [x] README 与 JS/TS payload 质量边界文档已同步：相对 import 本地类型可展开，包类型、namespace import 和需要完整 TS project graph 的类型仍保守回退。

已完成补充：JS/TS 静态生成器、generation context 和 LLM provider request 现在对相对 import 本地类型的判断一致。真实 TS/Vue 项目里常见的 `Promise<ExternalUser>`、`response.json()` 和注入式 client 返回不再出现“静态草稿已能生成结构 payload，但 provider context 仍提示回退”的割裂。

## 第三百二十一阶段：JS/TS resolved imported type 提示语义收敛

1. [x] `payload_notes` 现在能区分相对 import 类型已解析和未解析两种状态。
2. [x] 已解析的本地 imported type 输出 `resolved from <file>` 正向提示，不再继续输出 `read candidate source files` 这种待人工检查语义。
3. [x] 未解析相对 import、package import 和 namespace import 仍保留原有保守提示。
4. [x] context 与 LLM provider request 均覆盖 resolved imported type 的正向提示断言。

已完成补充：Agent 读取 JS/TS generation context 时，不会再把已经可展开的本地 imported type 误解为需要人工读取候选文件。下一步应继续压测简单 barrel re-export、alias import 和 `index.ts` re-export 这类真实项目常见组织方式，确认当前解析能力和文档边界一致。

## 第三百二十二阶段：JS/TS 简单 barrel re-export 类型解析

1. [x] re-export 解析支持 `export type { T } from './x'` 和 `export type * from './x'` 形式。
2. [x] named re-export 会继续解析到真实声明文件，而不是误把 barrel 文件当成类型声明来源。
3. [x] alias re-export 会保留本地 import 名，同时用真实导出名读取源声明，例如 `export type { ExternalUser as UserDTO } from './user'`。
4. [x] generation context 已覆盖 `api.ts -> models/index.ts -> user.ts` 的 barrel alias imported type，`payload_notes` 输出 `resolved from models/user.ts`。
5. [x] README 和 JS/TS payload 质量文档已把简单 named/star barrel re-export 纳入支持范围，复杂 barrel 和完整 TypeScript project graph 仍明确不承诺。
6. [x] LLM provider request 已覆盖 barrel alias resolved type，确认 `static_code` 中的 `response.json()` payload 使用真实字段结构。

已完成补充：真实 TS/Vue 项目常见的 `models/index.ts` 转出口不会再导致 provider context 回退到候选文件提示，provider 收到的静态草稿也会按真实 DTO 字段生成 `response.json()` payload。下一步应继续补 `export * from './user'` 的 provider/static_code 回归，确认 star barrel 和 named alias barrel 一样稳定。

## 第三百二十三阶段：JS/TS star barrel provider 回归

1. [x] LLM provider request 已覆盖 `api.ts -> models/index.ts -> user.ts` 的 `export type * from './user'` star barrel 场景。
2. [x] star barrel imported type 的 `payload_notes` 输出 `resolved from models/user.ts`。
3. [x] star barrel imported type 的 `static_code` 使用真实 DTO 字段生成 `response.json()` mock 和 `toEqual` 断言。

已完成补充：简单 named alias barrel 和 star barrel 都已在 provider/static_code 层有回归保护。下一步应回到真实样本验证，优先选择用户提供的 `laoxia-scaffold-v1.0.0/web` Vue 项目或一个小型 TS/Vitest fixture，确认这些静态规则在 Vue/Vite 项目结构里能被实际触发。

## 第三百二十四阶段：laoxia Vue 项目 JS smoke 与显式输出路径修复

1. [x] 已检查 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0`，实际目录为 `car-admin-web` / `car-admin-server`，不存在独立 `web` 目录。
2. [x] `car-admin-web` 是 Vue CLI 2.x 风格 JS/Vue 项目，不是 TS/Vite 项目；源码中没有 `import type` / TypeScript DTO，不能作为 imported type/barrel 能力的真实样本。
3. [x] 使用 `car-admin-web/src/api/system/user.js` 做 JS smoke，静态生成能识别 9 个导出 API 函数。
4. [x] smoke 暴露 `testgen <source_file> <output_file>` 显式输出到源文件外部时，JS/TS import 路径仍按源文件同目录生成的问题。
5. [x] `cmd/testgen` 现在把显式 `output_file` 传入 generation options，JS/TS 普通生成会按测试文件位置计算 `import` / `require` 源模块路径。
6. [x] 新增 CLI 回归测试覆盖 `src/user.js -> tests/user.test.js` 输出时生成 `import { listUsers } from '../src/user';`。
7. [x] 用 laoxia 真实 JS 文件复验输出到 `/tmp/testloop-laoxia-user.test.js`，import 已改为相对 `/tmp` 的源文件路径，不再错误写成 `./user`。

已完成补充：这次真实 Vue 项目验证没有证明 TS imported type 能力，但发现并修复了 CLI 显式输出路径下 JS/TS 测试不可运行的实际问题。下一步应继续补一个小型 TS/Vitest fixture 项目级 smoke，专门验证 imported type/barrel 能力；laoxia 项目则可作为 JS/Vue API 函数识别和输出路径回归样本。

## 第三百二十五阶段：TS/Vitest barrel CLI 项目级 smoke

1. [x] 新增 `test/js_ts_barrel_cli_smoke_test.sh`，临时创建 TS/Vitest 项目结构。
2. [x] smoke 覆盖 `src/api.ts -> src/models/index.ts -> src/models/user.ts` 的 `export type * from './user'` barrel imported type。
3. [x] smoke 通过 `go run ./cmd/testgen <source> <tests/api.test.ts>` 走真实 CLI，而不是只调用内部函数。
4. [x] smoke 断言生成文件包含 Vitest import、相对源模块 import、真实 DTO 字段的 `response.json()` mock 和 `toEqual` 断言。
5. [x] GitHub Actions CI 已纳入 `test/js_ts_barrel_cli_smoke_test.sh`。

已完成补充：imported type/barrel 能力现在既有单元测试，也有接近用户使用方式的 CLI 项目级 smoke。下一步应扩展 `generate_tests` MCP 工具的结构化响应示例或 contract 测试，确认 Agent 通过 MCP 消费时也能稳定看到这些 `payload_notes` 与 `static_code`。

## 第三百二十六阶段：MCP generate_tests TS barrel 结构化契约

1. [x] `HandleGenerateTests` 现在把目标 `test_file` 传入 generation options，普通 JS/TS 生成和 coverage task 生成都能按最终测试文件位置计算 import 路径。
2. [x] 新增 handler 级回归测试，覆盖 TS/Vitest 项目中的 `src/api.ts -> src/models/index.ts -> src/models/user.ts` star barrel imported type。
3. [x] handler 测试断言 `structuredContent` 与 text JSON 中的 `GenerateTestsOutput` 关键字段一致。
4. [x] handler 测试断言 `preview/static_code` 包含 Vitest import、源模块 import、真实 DTO payload 和 `toEqual` 断言。
5. [x] handler 测试断言 `context.targets[].payload_notes` 输出 `resolved from models/user.ts`，且不再出现候选文件或 `{ ok: true }` fallback 提示。

已完成补充：Agent 通过 MCP `generate_tests` 消费时，已经能稳定拿到 TS barrel imported type 的正向 `payload_notes` 与真实字段静态草稿。下一步应复查 LLM provider 文档和示例脚本：已解析类型不再需要 `read candidate source files`，示例应避免只围绕未解析候选提示组织 prompt。

## 第三百二十七阶段：LLM provider imported type 文档语义同步

1. [x] 复查 `examples/llm-provider.sh`，确认它只消费 `read candidate source files` 未解析提示，不会把 `resolved from <file>` 当作待读取候选。
2. [x] `docs/llm-provider.md` 已区分两类语义：已解析相对 import 输出 `resolved from <file>`，`static_code` / `context` 已带真实类型结构；未解析相对 import 才输出候选源码文件。
3. [x] 文档已说明示例 provider 对 `resolved from <file>` 不重复读取文件，避免把已解析类型当成缺上下文场景。

已完成补充：LLM provider 接入文档现在和最新 `payload_notes` 语义一致。下一步应跑文档/LLM provider 示例 gate 并等待远端 CI；通过后可以转入下一条高收益线：真实项目 Go server 的 `generate_tests -> run_tests` 闭环 smoke，验证 laoxia server 是否能作为 Go 样本。

## 第三百二十八阶段：laoxia Go server 临时闭环评估

1. [x] `car-admin-server` 基线 `go test ./...` 已通过；macOS 上 `gopsutil/disk` 仅输出 `IOMasterPort` deprecated warning。
2. [x] 先用临时副本验证 `utils/alias.go`，发现普通 `testgen` 写入新文件时会生成 `TestSliceMapper`、`TestSplitSlice`、`TestUserTypeOf` 等同名测试，与包内已有测试冲突。
3. [x] 改用无现有测试的 `global/utils.go` 临时副本验证，`testgen -> go test ./global` 通过。
4. [x] 生成测试主要是 `skip: true` 的 TODO 骨架，证明 Go 普通生成在真实项目里更适合作为起点，不应被包装成高质量断言自动完成。
5. [x] 本阶段未写入 laoxia 原仓库，只操作临时副本。

已完成补充：laoxia server 可以作为 Go 真实项目 smoke，但更适合验证“生成 -> 运行 -> 解析/反馈”闭环，而不是证明单次生成测试质量。下一步应修复或至少契约化 Go 普通 CLI 在已有测试包中可能生成重复 `TestXxx` 名称的问题；这会直接提升真实项目可用性。

## 第三百二十九阶段：Go CLI 显式输出重复测试名规避

1. [x] 新增 `generator.AvoidDuplicateGoTestNames`，写 Go 测试前扫描同包已有 `_test.go` 文件。
2. [x] 当生成的 `TestXxx` 与已有测试冲突时，自动改名为 `TestXxxTestLoop`，必要时追加数字后缀。
3. [x] `cmd/testgen` 写文件前调用该逻辑，避免显式输出新文件时写出不可编译的重复测试名。
4. [x] 新增 CLI 回归测试：已有 `TestAdd` 时，显式输出文件生成 `TestAddTestLoop`。
5. [x] 用 laoxia server 临时副本复验 `utils/alias.go`，原先冲突的 `TestSliceMapper` 等测试已改名，`go test ./utils` 通过；原仓库未被写入。

已完成补充：Go 普通 CLI 在真实项目已有测试包中的可运行性提升了一档。下一步应把这个 laoxia Go 临时闭环固化成仓库内可复跑 smoke，使用临时 fixture 或复制最小源码，避免依赖用户本机路径。

## 第三百三十阶段：Go CLI 重名场景项目级 smoke

1. [x] 新增 `test/go_cli_duplicate_name_smoke_test.sh`，临时创建一个 laoxia-like Go module。
2. [x] smoke 覆盖同包已有 `TestSliceMapper` 时，`testgen` 生成新文件应改名为 `TestSliceMapperTestLoop`。
3. [x] smoke 同时确认未冲突的 `SplitSlice` 保持原测试名。
4. [x] smoke 运行 `go test ./utils`，验证生成文件不仅文本正确，而且包级测试可编译运行。
5. [x] GitHub Actions CI 已纳入该 smoke。

已完成补充：laoxia Go 临时闭环中暴露的问题已经固化为仓库内可复跑 smoke。下一步应等待 CI，并考虑把 Go 重名规避推广到 MCP 普通生成写入新文件的路径，确认 handler 和 CLI 行为一致。

## 第三百三十一阶段：MCP 普通 Go 生成重复测试名规避

1. [x] `HandleGenerateTests` 在写 Go 测试文件前复用 `generator.AvoidDuplicateGoTestNames`。
2. [x] 普通 MCP `generate_tests` 生成新 `calc_test.go` 时，会扫描同包其他 `_test.go` 文件。
3. [x] 新增 handler 回归测试：同包 `existing_test.go` 已有 `TestAdd` 时，普通生成输出 `TestAddTestLoop`。
4. [x] CLI 和 MCP 普通 Go 生成现在对跨文件重复 `TestXxx` 的处理一致。

已完成补充：Go 重名规避不再只覆盖 CLI，也进入 MCP 工具路径。下一步应跑完整验证并等待 CI；通过后可以转向结果解析/失败建议侧，检查这类 generated-but-skipped Go 测试是否能给 Agent 更明确的下一步动作。

## 第三百三十二阶段：generate_tests action 分流字段

1. [x] `GenerateTestsOutput` 新增可选 `action` 字段。
2. [x] 普通生成内容可直接运行时输出 `action: "ready"`。
3. [x] Go TODO `t.Skip`、JS/TS `it.skip` / `manual_review_`、Python `pytest.skip`、Java `Assumptions.assumeTrue(false, ...)` 等生成内容会输出 `action: "manual_review"`。
4. [x] provider error 顶层 `action` 与 `provider_error.action` 保持一致。
5. [x] Agent JSON contract 测试和 `docs/agent-contract.md` 已同步 `generate_tests.action`。
6. [x] handler 测试已覆盖 laoxia-like 泛型 Go TODO 骨架，确认 MCP `generate_tests` 输出 `action: "manual_review"`。

已完成补充：Agent 调用普通 `generate_tests` 后，不必等 `run_tests` 返回 skipped 才知道测试草稿需要补输入/断言。下一步应跑完整验证并等待 CI；通过后继续检查 `run_tests` / `fix_suggestions` 对 skipped/manual_review 场景的说明是否足够直观。

## 第三百三十三阶段：run_tests action 分流字段

1. [x] `TestResult` 新增可选 `action` 字段。
2. [x] `run_tests` 成功执行真实用例时输出 `action: "ready"`。
3. [x] `run_tests` 遇到全部 skipped/TODO 或无真实通过用例时输出 `action: "manual_review"`，避免 Agent 把空通过误判为有效测试。
4. [x] `run_tests` 失败时按是否存在内联修复建议输出 `apply_fix_suggestions` 或 `inspect_failures`。
5. [x] `parse_results` 复用同一套 `TestResult.action` 语义，保持同类型输出一致。
6. [x] Agent JSON contract、README 和 handler 测试已同步。

已完成补充：执行侧也有了 Agent 可直接消费的下一步动作，和 `generate_tests.action`、`validate_coverage_task.action` 形成一致的分流入口。下一步应跑完整验证并等待 CI；通过后继续检查 `fix_suggestions` 的失败分类是否能覆盖更多真实项目错误，例如 Go 编译错误、JS import/require 错误和 Python import error。

## 第三百三十四阶段：fix_suggestions 真实失败分类补强

1. [x] Go JSON parser 保留 package-level 编译失败细节，不再把可解析错误压成 `package failed`。
2. [x] `fix_suggestions` 新增 `module_resolution`，覆盖 JS/TS `Cannot find module`、`ERR_MODULE_NOT_FOUND`、`failed to resolve import` 和 Go 依赖解析失败。
3. [x] `fix_suggestions` 新增 `python_import_error`，覆盖 `ModuleNotFoundError`、`ImportError` 和常见相对 import 失败。
4. [x] `fix_suggestions` 新增 `compile_error`，覆盖编译/语法错误、未使用变量/import、参数数量和返回值数量错误。
5. [x] 真实 `run_tests include_fix_suggestions=true` 回归覆盖 Go 编译失败，确认 parser 细节能进入 `repair_task`。
6. [x] README 和 Agent 结构化契约已登记新增 category。

已完成补充：失败修复闭环对真实项目的常见“测试还没跑起来”问题更敏感了，不会把模块缺失、import 失败、编译错误都混成 generic。下一步应跑完整验证并等待 CI；通过后建议补一条 CLI/MCP smoke，展示 `run_tests.action + fix_suggestions.category` 的组合如何被客户端直接消费。

## 第三百三十五阶段：客户端 demo 显示 action/category 消费路径

1. [x] `examples/mcp-client-demo` 的首次 `run_tests` 输出新增 `action`。
2. [x] `examples/mcp-client-demo` 的复跑输出新增 `action`。
3. [x] demo 继续展示 `repair_task.category`，形成 `action=apply_fix_suggestions -> category=expectation_mismatch -> repair_task -> action=ready` 的最小客户端消费路径。
4. [x] `test/mcp_client_demo_test.sh` 固定 action/category 输出。
5. [x] README 和 `docs/showcase-agent-loop.md` 已同步新的预期输出。
6. [x] `fix_suggestions` 识别 Go 常见 `actual, want expected` 断言格式，避免 demo 和真实 Go 失败回退到 `generic_failure`。

已完成补充：公开最小 demo 不再只展示 pass/fail，而是展示 Agent 真正该消费的 `action` 和 `category` 字段。下一步应跑完整验证并等待 CI；通过后建议检查 `docs/quickstart.md` / onboarding 文档是否也需要引用 action/category 的新版输出。

## 第三百三十六阶段：接入文档同步 action/category 路径

1. [x] `docs/quickstart.md` 的最小闭环说明改为 `run_tests.action -> fix_suggestions.category -> repair_task -> rerun.action -> parse_coverage`。
2. [x] `docs/showcase-onboarding.md` 同步最小 Agent 闭环字段链路。
3. [x] `docs/first-run-diagnostics.md` 同步首跑诊断边界里的 demo 描述。
4. [x] `docs/verification-report.md` 同步用户项目验收报告里的 demo 描述。

已完成补充：新用户从 quickstart、onboarding、first-run diagnostics 或 verification report 进入，都能看到同一条结构化字段消费路径。下一步应跑文档 gate 并等待 CI；通过后建议回到真实项目样本，选一个小 Go/JS 失败样本生成 artifact fixture，固定 `run_tests.action + category` 的机器可读输出。

## 第三百三十七阶段：run_tests action/category fixture

1. [x] 新增 `docs/fixtures/run-tests/apply-fix-suggestions.json`，固定普通 `run_tests` 失败结果的 `status/action/failures/fix_suggestions/repair_task`。
2. [x] 新增 `tools/run_tests_fixture_test.go`，通过真实临时 Go 项目调用 `HandleRunTests include_fix_suggestions=true` 生成稳定投影并对比 fixture。
3. [x] `docs/fixtures.md` 新增 run_tests fixture 列表，并说明稳定字段和维护入口。
4. [x] `docs/client-integration.md` 新增普通 `run_tests` fixture 的客户端断言建议。
5. [x] 新增 `test/run_tests_fixture_index_test.sh` 并纳入 CI，固定 fixture 链接、`fail/apply_fix_suggestions`、`expectation_mismatch` 和 `repair_task` 关键字段。

已完成补充：客户端现在既有 `validate_coverage_task` 的 end-to-end fixture，也有普通 `run_tests` 的 action/category fixture。下一步应跑完整验证并等待 CI；通过后建议回到代码侧，补一个 JS/TS 模块解析失败的真实 `run_tests` fixture 或 handler 回归，验证 `module_resolution` 不是只靠直接调用 `fix_suggestions`。

## 第三百三十八阶段：JS/TS 模块解析失败 run_tests 回归

1. [x] 新增 `TestHandleRunTestsVitestModuleResolutionFixSuggestion`。
2. [x] 通过 fake `npx vitest` 输出真实命令级模块解析失败：`Error: Cannot find module './missing-plugin'`。
3. [x] 验证 `run_tests include_fix_suggestions=true` 返回 `action: "apply_fix_suggestions"`。
4. [x] 验证内联 `fix_suggestions[0].category` 和 `repair_task.category` 都是 `module_resolution`。

已完成补充：`module_resolution` 不再只被直接 `fix_suggestions` 单元测试覆盖，也进入了 JS/TS runner 输出到 `run_tests` 结构化结果的闭环。下一步应跑完整验证并等待 CI；通过后建议补 Python import error 的同类 `run_tests` 回归，覆盖 pytest collection/import 阶段失败。

## 第三百三十九阶段：Python import error run_tests 回归

1. [x] 新增 `TestHandleRunTestsPytestImportErrorFixSuggestion`。
2. [x] 通过 fake `python3 -m pytest` 输出 pytest collection 阶段 `ModuleNotFoundError`。
3. [x] 验证 `run_tests include_fix_suggestions=true` 返回 `action: "apply_fix_suggestions"`。
4. [x] 验证内联 `fix_suggestions[0].category` 和 `repair_task.category` 都是 `python_import_error`。

已完成补充：`python_import_error` 也进入了 pytest runner 输出到 `run_tests` 结构化结果的闭环。下一步应跑完整验证并等待 CI；通过后建议盘点 `fix_suggestions` 的 remaining generic 样本，决定是否继续细分，避免为了低频错误过度扩分类。

## 第三百四十阶段：generic_failure 继续细分边界

1. [x] 检查 `README.md`、`docs/agent-contract.md`、`tools/testdata/golden/*`、`docs/fixtures/**/*` 和测试代码里的 `generic_failure` 使用。
2. [x] 确认当前没有稳定 fixture/golden 仍期待 `generic_failure` 作为真实常见失败分类。
3. [x] 保留 `generic_failure` 作为未知失败兜底，不再凭想象扩展低频 category。
4. [x] 后续 category 细分改为真实样本驱动：先从 `run_tests` / `validate_coverage_task` 暴露的失败日志或公开项目 smoke 中取样，再补 parser/fix_suggestions 回归。

已完成补充：这条线先收住，不继续为没有样本的错误类型加规则。下一步应跑文档 gate、提交并等待 CI；通过后建议转向真实项目样本驱动的下一轮，例如选择一个小型 JS/TS 或 Go 项目构造可复跑的失败 fixture，或者回到 laoxia server/web 跑最新闭环 smoke。

## 第三百四十一阶段：laoxia v0.5.12 验收报告复验

1. [x] 用当前源码构建 `/tmp/testloop-mcp-laoxia-smoke`，版本输出 `testloop-mcp 0.5.12`。
2. [x] laoxia server 运行 `scripts/generate-verification-report.sh`，用户项目命令为 `go test ./...`，summary 为 `overall_status=passed`、`failed_count=0`、用户项目 smoke `passed`。
3. [x] laoxia web 运行 `scripts/generate-verification-report.sh`，用户项目命令为 `pnpm install --frozen-lockfile && pnpm build:prod`，summary 为 `overall_status=passed`、`failed_count=0`、用户项目 smoke `passed`。
4. [x] 两份报告都包含最新最小 Agent demo 输出：`run_tests.action=apply_fix_suggestions` 和 `rerun.action=ready`。
5. [x] 复验后 `car-admin-server` 和 `car-admin-web` 工作区均保持干净。
6. [x] `docs/verification-report.md` 已追加 v0.5.12 复验记录。

已完成补充：真实用户项目 smoke 仍能纳入同一份 Agent/CI 验收报告，并且最小 Agent demo 已展示最新 action 字段。下一步应跑文档 gate、提交并等待 CI；通过后建议选择一个真实失败样本，而不是继续只跑 passing smoke。

## 第三百四十二阶段：testgen CLI 暴露生成动作

1. [x] 将生成测试内容的 action 判定下沉到 `internal/generator.GeneratedTestsAction`，避免 MCP 和 CLI 各自维护规则。
2. [x] `generate_tests` 继续复用同一判定，Go TODO skip、JS/TS skip/manual_review、Python skip/manual_review、Java assumption/manual_review 会稳定标记为 `manual_review`。
3. [x] `cmd/testgen` 成功输出从 `provider=...` 扩展为 `provider=... action=...`，让 CLI 用户在运行测试前就能看到草稿是否需要手审。
4. [x] CLI 单测固定 Go 静态生成 TODO 骨架的 `action=manual_review`。
5. [x] Go duplicate-name smoke 和 JS/TS barrel smoke 已同步新输出，分别覆盖 `manual_review` 与 `ready`。
6. [x] README 和安装验证文档已说明 `action=manual_review` 不能当成有效覆盖。

已完成补充：独立 CLI 现在和 MCP `generate_tests` 一样能暴露 Agent 动作信号，避免 laoxia 这类 Go 静态骨架因为 `go test` skipped 通过而被误读为 ready。下一步应跑完整本地验证、提交并等待 CI；通过后建议补一个报告层检查，把 verification report 里生成测试 smoke 的 `manual_review` 展示出来。

## 第三百四十三阶段：验收报告展示 testgen action

1. [x] `scripts/generate-verification-report.sh` 新增默认“独立 CLI 生成动作 smoke”章节。
2. [x] 该章节会优先使用 sibling `testloop-testgen`，其次回退当前源码 `go run ./cmd/testgen`，最后才使用 PATH，避免旧 CLI 把报告误带偏。
3. [x] smoke 生成 Go 静态测试草稿并断言 CLI 输出包含 `action=manual_review`。
4. [x] 新增 `TESTLOOP_REPORT_SKIP_TESTGEN_SMOKE=true` 跳过开关，以及 `TESTLOOP_REPORT_TESTGEN_COMMAND` 覆盖入口。
5. [x] `test/verification_report_test.sh` 覆盖跳过路径和默认通过路径，并固定 Markdown/summary JSON 中的章节状态。
6. [x] `docs/verification-report.md` 已说明该 smoke 的目的：避免 TODO/skipped 草稿被误读为有效覆盖。
7. [x] 构建 `/tmp/testloop-report-action-bin/testloop-mcp` 和 sibling `/tmp/testloop-report-action-bin/testloop-testgen` 后运行真实报告通过，summary 为 `overall_status=passed`、`failed_count=0`，新章节输出 `provider=static action=manual_review`。

已完成补充：验收报告现在不仅展示 MCP Agent demo 的 `run_tests.action`，也展示独立 CLI 生成阶段的 `action=manual_review`。下一步应跑脚本语法、verification report 测试、文档 gate 和全量 Go 测试；通过后提交并继续观察远端 CI。

## 第三百四十四阶段：验收 summary JSON 暴露 action signal

1. [x] `scripts/generate-verification-report.sh` 的 summary TSV 内部格式增加 signals JSON 列，旧 section 保持空值。
2. [x] summary JSON 的 section 对象在存在信号时输出 `signals` 字段。
3. [x] 独立 CLI 生成动作 smoke 会从输出中提取 `action=...`，写入 `sections[].signals.action`。
4. [x] `test/verification_report_test.sh` 固定 `signals.action == manual_review`。
5. [x] `docs/verification-report.md` 已补充 Agent 可从 summary JSON 直接读取该动作。

已完成补充：Agent 不必解析 Markdown 明细，也能从验收 summary JSON 判断 CLI 生成草稿是否需要人工补输入/断言。下一步应跑报告专项、文档 gate、全量 Go 测试、提交并观察 CI。

## 第三百四十五阶段：决策 demo 展示 summary signals

1. [x] `examples/verification-summary-decision-demo` 的 section 模型支持可选 `signals`。
2. [x] 决策 demo 会打印 `section_signal=<name> action=<action>`，让 Agent/CI 不解析 Markdown 也能记录动作信号。
3. [x] 整体 `passed` 且包含 `manual_review` smoke signal 时仍输出 `agent_next_step=ready`，避免把工具自检信号误判为验收失败。
4. [x] `test/verification_summary_decision_demo_test.sh` 覆盖通过报告中的 `manual_review` signal。
5. [x] `docs/verification-report.md` 已说明 `signals` 是 section 级可选字段，不代表 overall failure。

已完成补充：验收 summary 的 action signal 现在可以被 demo 直接消费并展示，同时保持整体通过语义清晰。下一步应跑决策 demo 测试、报告测试、文档 gate、全量 Go 测试，随后提交并观察 CI。

## 第三百四十六阶段：onboarding / first-run 回复展示 summary signals

1. [x] `examples/onboarding-agent-response-demo` 的 section 模型支持可选 `signals`。
2. [x] onboarding 回复的证据区会打印 `section_signal=<name> action=<action>`。
3. [x] `examples/first-run-agent-response-demo` 的 section 模型支持可选 `signals`。
4. [x] first-run 回复的证据区会打印 `section_signal=<name> action=<action>`。
5. [x] `test/onboarding_agent_response_demo_test.sh` 覆盖通过报告里的 `manual_review` signal。
6. [x] `test/first_run_agent_response_demo_test.sh` 覆盖 first-run artifact 里由报告脚本生成的 `manual_review` signal。

已完成补充：CI artifact 的 Agent 回复现在也能直接展示 testgen 生成动作信号，不再只依赖 decision demo。下一步应跑 onboarding/first-run 回复测试、相关 artifact 测试、全量 Go 和文档 gate；通过后提交并观察 CI。

## 第三百四十七阶段：wrapper artifact 固定 signals 链路

1. [x] `test/run_onboarding_ci_test.sh` 的成功路径固定 summary JSON 中的 `signals.action=manual_review`。
2. [x] `test/run_onboarding_ci_test.sh` 的成功和失败路径固定 `agent-decision.txt` 与 `agent-response.txt` 中的 `section_signal=独立 CLI 生成动作 smoke action=manual_review`。
3. [x] `test/run_first_run_ci_test.sh` 的成功路径固定 summary JSON 中的 `signals.action=manual_review`。
4. [x] `test/run_first_run_ci_test.sh` 的成功和失败路径固定 `agent-decision.txt` 与 `agent-response.txt` 中的 `section_signal=独立 CLI 生成动作 smoke action=manual_review`。

已完成补充：完整 first-run/onboarding wrapper 产物现在会被测试要求保留 action signal，不只是单独 demo 能展示。下一步应跑 wrapper 测试、报告测试、全量 Go 和文档 gate；通过后提交并观察 CI。

## 第三百四十八阶段：verification summary schema 固化 signals 契约

1. [x] 新增 `docs/fixtures/verification-summary.schema.json`，固定 summary JSON 的 `overall_status`、`failed_count`、`sections[]` 和可选 `sections[].signals` 结构。
2. [x] `sections[].signals.action` 固定为非空字符串，其他 signals 也必须是非空字符串，方便后续扩展。
3. [x] 新增 `tools/verification_summary_schema_test.go`，验证现有 failure summary fixtures、first-run artifact summary 和 onboarding artifact summary 都符合 schema。
4. [x] schema 测试覆盖带 `signals.action=manual_review` 的正例和非字符串 signal 的负例。
5. [x] `docs/fixtures.md` 和 `docs/verification-report.md` 已链接 summary schema，并说明 `signals.action` 是可选 section 级动作信号。

已完成补充：summary JSON 的 action signal 不再只是脚本输出约定，而是有 schema、fixture 回归和文档入口。下一步应跑 schema 测试、fixture 文档测试、报告测试、全量 Go 和文档 gate；通过后提交并观察 CI。

## 第三百四十九阶段：客户端文档接入 summary schema

1. [x] `docs/mcp-client-contract-tests.md` 已加入 `verification-summary.schema.json` 下载和 AJV 校验示例。
2. [x] 客户端契约测试说明明确 `sections[].signals.action` 是可选动作信号，不等于整体失败。
3. [x] `docs/adopter-verification-guide.md` 已链接 summary schema，并说明可校验 section 级动作信号。
4. [x] `docs/client-integration.md` 已链接 summary schema，说明客户端读取 `sections[].signals.action`。
5. [x] `test/mcp_client_contract_doc_test.sh`、`test/adopter_verification_guide_doc_test.sh`、`test/client_integration_doc_test.sh` 和 README CI snippet 测试已固定新入口。

已完成补充：接入方现在能从客户端契约、接入指南和集成说明三条路径发现 verification summary schema。下一步应跑相关文档测试、文档 gate、全量 Go 和 diff check；通过后提交并观察 CI。

## 第三百五十阶段：Agent response contract 补充 section_signal

1. [x] `docs/agent-response-artifact-contract.md` 将 `section_signal=<section name> action=<action>` 固定为可选证据字段。
2. [x] 文档明确 `section_signal ... action=manual_review` 是 section 级证据，不代表整体验收失败。
3. [x] contract 文档链接 `verification-summary.schema.json`。
4. [x] `test/agent_response_artifact_contract_doc_test.sh` 固定 `section_signal`、`sections[].signals`、`action=manual_review` 和 summary schema 链接。

已完成补充：Agent response artifact 的自然语言契约现在和 summary schema 对齐，客户端不会把手审草稿信号误当整体失败。下一步应跑 contract 文档测试、文档 gate、全量 Go 和 diff check；通过后提交并观察 CI。

## 第三百五十一阶段：artifact manifest 暴露 summary schema

1. [x] `docs/fixtures/agent-response-artifact-manifest.json` 新增 `summary_schema: "./verification-summary.schema.json"`。
2. [x] artifact manifest JSON Schema 将 `summary_schema` 固定为必填字段。
3. [x] `examples/agent-response-manifest-demo` 会校验 summary schema 文件可读，并输出 `summary_schema=...`。
4. [x] manifest shell 测试和 Go schema 测试都固定 summary schema 指针。
5. [x] `docs/fixtures.md` 已说明客户端可通过 manifest 的 `summary_schema` 找到 verification summary 契约。

已完成补充：客户端只读取 artifact manifest 也能自动发现 summary schema，不需要额外硬编码路径。下一步应跑 manifest demo、manifest/schema 测试、文档 gate、全量 Go 和 diff check；通过后提交并观察 CI。

## 第三百五十二阶段：artifact fixture 固定 section signals

1. [x] first-run 失败 artifact fixture 的 summary/report/Agent response 已刷新为包含“独立 CLI 生成动作 smoke”的 `signals.action=manual_review`。
2. [x] onboarding 失败 artifact fixture 的 summary/report/Agent response 已刷新为包含同一 section 级动作信号。
3. [x] `agent-response-artifact-manifest.json` 新增 `expected_section_signals`，声明客户端回归必须保留的 section/action 组合。
4. [x] manifest JSON Schema 要求 artifact 声明 `expected_section_signals`，并限制每个 signal 必须包含非空 `section` 和 `action`。
5. [x] `examples/agent-response-manifest-demo` 会读取 summary 并校验 manifest 声明的 expected section signals，同时输出可检查的 `expected_section_signals=...`。
6. [x] first-run/onboarding fixture 测试、manifest 测试和文档说明已固定 `section_signal=独立 CLI 生成动作 smoke action=manual_review`。

已完成补充：artifact manifest 现在不只告诉客户端去哪找 summary schema，还能让客户端验证关键 section 级动作信号不会在 fixture 刷新时丢失。下一步应跑 artifact fixture、manifest/schema、文档 gate、全量 Go 和 diff check；通过后提交并观察 CI。

## 第三百五十三阶段：manifest demo 校验机器决策

1. [x] `examples/agent-response-manifest-demo` 会读取每个 artifact 的 `agent-decision.txt`。
2. [x] demo 校验 `agent-decision.txt` 中的 `agent_next_step` 必须等于 manifest 的 `expected_action`。
3. [x] demo 输出新增 `decision_action=...`，方便客户端 smoke 直接观察机器分流结果。
4. [x] README、客户端集成说明和 MCP 客户端契约测试说明已同步 `decision_action` 验收入口。
5. [x] `test/agent_response_manifest_demo_test.sh` 固定 `decision_action=inspect-user-project`。

已完成补充：manifest demo 现在同时覆盖自然语言回复、机器决策和 summary section signal，客户端不必拆开多个脚本才能确认 artifact 三条消费路径一致。下一步应跑 manifest demo、客户端文档测试、README snippet、全量 Go、文档 gate 和 diff check；通过后提交并观察 CI。

## 第三百五十四阶段：v0.5.13 候选发布边界

1. [x] 新增 `docs/plan-release-notes-v0.5.13.md`，整理 v0.5.12 之后 action 信号、summary schema、artifact manifest 和客户端契约回归的候选发布说明。
2. [x] 新增 `docs/plan-release-v0.5.13.md`，记录当前差异核对、候选内容、已验证命令、远端 CI 证据、发布前门禁和正式发布前待办。
3. [x] 候选边界明确不扩语言、不切测试生成定位，主线仍是 Agent 测试反馈闭环的机器可读契约。
4. [x] 正式发布动作保持未完成：不提前改版本号、不打 tag、不更新 Homebrew tap。

已完成补充：v0.5.13 的候选边界已经从 Unreleased 变化中抽出来，后续如果要发布，可以按检查清单跑 release readiness，而不是临时整理范围。下一步应跑 release 文档索引、文档链接、README snippet、全量 Go 和 diff check；通过后提交并观察 CI。

## 第三百五十五阶段：v0.5.13 候选门禁复验

1. [x] shell 语法门禁通过：`find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`。
2. [x] 全量 Go 测试通过：`go test ./...`。
3. [x] 全部 shell 回归通过：`for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`。
4. [x] 候选主服务和 `testloop-testgen` 二进制构建通过。
5. [x] 候选 help/version 验证通过；当前版本仍输出 `testloop-mcp 0.5.12`，说明尚未进入正式版本准备。
6. [x] darwin arm64 打包 dry-run、sha256 校验和 tarball 内容检查通过。
7. [x] `docs/plan-release-notes-v0.5.13.md` 和 `docs/plan-release-v0.5.13.md` 已回填候选门禁证据。

已完成补充：v0.5.13 已通过候选 release readiness，但还没有切版本号、收敛 changelog 或执行正式发布。下一步如果继续走发布路径，应进入正式版本准备；如果不发布，应回到产品主线继续打磨 Agent artifact 消费和真实项目闭环。

## 第三百五十六阶段：manifest demo 校验 summary schema

1. [x] `examples/agent-response-manifest-demo` 会加载 manifest 的 `summary_schema` 并解析 JSON Schema。
2. [x] demo 在校验每个 artifact 时，会先用 summary schema 验证对应 `verification-summary.json`。
3. [x] demo 输出新增 `summary_validated=verification-summary.json`，方便客户端 smoke 固定 summary schema 校验路径。
4. [x] README、客户端集成说明、MCP 客户端契约测试说明和 CHANGELOG 已同步该输出。
5. [x] `test/agent_response_manifest_demo_test.sh` 固定 `summary_validated=verification-summary.json`。

已完成补充：manifest demo 现在不只是发现 summary schema 文件，还实际验证 artifact summary，客户端可以用一个 demo 同时覆盖 response、decision、summary schema 和 section signal。下一步应跑 manifest demo、客户端文档测试、README snippet、schema 测试、全量 Go、文档 gate 和 diff check；通过后提交并观察 CI。

## 第三百五十七阶段：v0.5.13 正式版本准备

1. [x] `main.go` MCP implementation version 已更新到 `0.5.13`。
2. [x] `CHANGELOG.md` 已将 Unreleased 内容收敛为 `v0.5.13 - 2026-07-20`，并保留新的空 Unreleased。
3. [x] README、installation、quickstart、first-run/onboarding/verification CI 模板和测试期望已同步到 `0.5.13` / `v0.5.13`。
4. [x] shell 语法、`go test ./...`、全部 `test/*_test.sh` 已通过。
5. [x] 主服务和 `testloop-testgen` release-prep 二进制构建通过，`testloop-mcp --version` 输出 `testloop-mcp 0.5.13`。
6. [x] darwin arm64 打包 dry-run、sha256 校验和 tarball 内容检查通过。
7. [x] `docs/plan-release-v0.5.13.md` 和 `docs/plan-release-notes-v0.5.13.md` 已回填正式版本准备证据。

已完成补充：v0.5.13 已进入正式发布准备状态，但尚未提交版本准备、打 tag、生成 Release assets 或更新 Homebrew tap。下一步应提交版本准备并等待远端 CI；CI 通过后继续 tag 和 Release Artifacts。

## 第三百五十八阶段：Release Artifacts API 503 加固

1. [x] `v0.5.13` tag 已推送，并触发 Release Artifacts run `29710040572`。
2. [x] Release Artifacts attempt 1 失败在 GitHub Release API 503：`gh release view` 返回 503 后没有重试。
3. [x] Release Artifacts attempt 2 失败在 uploads.github.com 503：`gh release upload` 上传单个资产时返回 503。
4. [x] 构建、sha256 和 tarball/zip 内容校验在失败前已通过，问题集中在 GitHub Release API/Upload API 临时不可用。
5. [x] `.github/workflows/release.yml` 已给 `gh release view/create/upload` 增加重试，并改成逐个文件上传，降低单个资产 503 导致整组失败的概率。

已完成补充：发布流水线对 GitHub Release API 抖动更稳。下一步应提交 workflow 加固，等待 CI，然后用 workflow_dispatch 对 `v0.5.13` 重跑 Release Artifacts。

## 第三百五十九阶段：v0.5.13 Release assets 与仓库 Formula

1. [x] workflow_dispatch run `29710581315` 使用加固后的 release workflow 对 `v0.5.13` 重跑 Release Artifacts。
2. [x] 首轮 dispatch 上传过程中仍受 GitHub upload 503 影响，随后 rerun failed jobs 补齐失败平台。
3. [x] Release assets 已补齐 10 个文件：五个平台压缩包和对应 `.sha256`。
4. [x] `TESTLOOP_MCP_REPO=sleticalboy/testloop-mcp scripts/verify-release-assets.sh v0.5.13` 通过。
5. [x] GitHub Release 正文已更新为正式 v0.5.13 发布说明。
6. [x] `scripts/generate-homebrew-formula.sh v0.5.13` 已更新仓库内 Formula。
7. [x] `ruby -c Formula/testloop-mcp.rb`、`brew style Formula/testloop-mcp.rb` 和 `sh test/release_assets_test.sh` 通过。

已完成补充：v0.5.13 的 GitHub Release 资产和仓库内 Formula 已完成。下一步应提交这些发布记录和 Formula 改动；CI 通过后更新 Homebrew tap，再触发 Post-Release Verify。

## 第三百六十阶段：v0.5.13 Homebrew tap 与发布后验证

1. [x] `scripts/update-homebrew-tap.sh v0.5.13` 已将 `sleticalboy/homebrew-tap` 推进到 `testloop-mcp 0.5.13`。
2. [x] Homebrew tap 远端 main 为 `0cb590eda5dc7d75353c2005e4c6927ed34c81dd`。
3. [x] 本机 tap 目录 fast-forward 后，`brew info --json=v2 sleticalboy/tap/testloop-mcp` 显示 stable `0.5.13`。
4. [x] `brew audit --formula --strict sleticalboy/tap/testloop-mcp` 通过。
5. [x] `TESTLOOP_MCP_REPO=sleticalboy/testloop-mcp scripts/verify-release-assets.sh v0.5.13` 再次验证 10 个 Release 资产完整。
6. [x] Post-Release Verify run `29711047138` passed，覆盖 release asset manifest 和五个平台安装校验。
7. [x] 全局 `brew update` 因 Homebrew/brew 镜像队列和 pack 断连中断；该失败不影响 tap 远端、tap 本地 fast-forward、公式审计和 post-release workflow 结果。

已完成补充：v0.5.13 发布闭环已经完成，包含 tag、GitHub Release、Homebrew tap 和 post-release install 验证。下一步应把发布完成记录提交并等待主分支 CI；CI 通过后回到产品主线，优先继续打磨 Agent 测试反馈闭环的真实项目演示和结构化输出一致性。

## 第三百六十一阶段：主工具结构化返回一致性回归

1. [x] 新增 `tools/tool_result_contract_test.go`，抽出 `assertStructuredContentMatchesTextJSON` 测试 helper。
2. [x] 覆盖 `parse_results`，确认解析结果的 `structuredContent`、handler 返回值和 text JSON 一致。
3. [x] 覆盖 `parse_coverage`，确认覆盖率报告的三路返回一致。
4. [x] 覆盖 `generate_tests`，确认生成结果、预览和 action 字段三路返回一致。
5. [x] 覆盖 `run_tests`，用临时 Go 项目确认测试执行结果三路返回一致。
6. [x] 覆盖 `fix_suggestions`，确认修复建议数组三路返回一致。
7. [x] `docs/agent-contract.md` 已记录 handler 层一致性回归职责，`CHANGELOG.md` 已加入 Unreleased 记录。
8. [x] 专项验证已通过：`go test ./tools -run TestPrimaryToolResultsKeepStructuredContentAndTextJSONInSync -count=1`。

已完成补充：主 MCP 工具的结构化输出一致性现在有集中回归保护，降低 Agent 客户端因 fallback 路径字段漂移而误判下一步动作的风险。下一步应跑 tools 包和全量测试、文档 gate、diff check；通过后提交并观察 CI。随后继续推进真实项目演示制品，把 laoxia scaffold 或公开 fixture 的闭环输出沉淀成可复用案例。

## 第三百六十二阶段：laoxia 双栈报告入口收敛

1. [x] 新增 `scripts/showcase-laoxia-scaffold-report.sh`，一次生成 `car-admin-server` 和 `car-admin-web` 两份验收报告与 summary JSON。
2. [x] 新增嵌套子 summary 的顶层 `laoxia-summary.json`，把 server/web 两条 smoke 汇总成单一可机器消费的状态文件。
3. [x] `test/showcase_scripts_test.sh` 已固定该入口的 help、参数、成功路径和失败路径。
4. [x] `docs/showcase.md` 和 `docs/real-integration-cases.md` 已收录这条双栈入口，明确它是项目级回归命令，而不是通用 bootstrap。
5. [x] 真实 laoxia 样本已跑通：`/tmp/testloop-laoxia-scaffold-live/server/verification-summary.json` 与 `/tmp/testloop-laoxia-scaffold-live/web/verification-summary.json` 均为 `overall_status=passed`、`failed_count=0`。

已完成补充：laoxia 的 server/web 验收现在有一个固定的一键入口，后续做项目级回归时不必再手工维护两条 smoke 命令。下一步应把这条入口继续作为真实项目验证样本使用；如果后续还要扩展更多 project pair，再考虑提炼共享 helper。

## 第三百六十三阶段：双栈报告 shared helper 提炼

1. [x] 新增 `scripts/showcase-dual-project-report.sh`，把两条用户项目 smoke 的通用报告逻辑提到独立 helper。
2. [x] `scripts/showcase-laoxia-scaffold-report.sh` 变成 thin wrapper，只保留 laoxia 的默认路径、命令和输出前缀。
3. [x] helper 输出嵌套子 summary 的总 `laoxia-summary.json` 仍然保持 `overall_status`、`failed_count` 和子 summary 全量字段。
4. [x] `README.md`、`docs/showcase.md` 和 `test/release_doc_index_test.sh` 已把 shared helper 作为可发现入口曝光。
5. [x] `test/showcase_scripts_test.sh` 已覆盖 shared helper 的直跑成功/失败路径、语法回归以及 laoxia wrapper 的成功/失败路径。
6. [x] 真实 laoxia 样本复验通过：`/tmp/testloop-laoxia-scaffold-live2/laoxia-summary.json` 显示 server/web 两边都为 `passed`。
7. [x] 真实 astaway pair 复验：`astaway-server` 当前本机 smoke 失败、`astaway-web` 构建通过，helper 正常输出 mixed 状态的嵌套 summary。
8. [x] 真实 QuickSmoke Go/Java pair 复验：`QuickSmoke-Backend-Go` 与 `words_java` 都通过，`quicksmoke-summary.json` 顶层 `overall_status=passed`、`failed_count=0`。
9. [x] 真实 APK Info Rust/Words Java pair 复验：`apk-info-zip` 与 `words_java` 都通过，`rustjava-summary.json` 顶层 `overall_status=passed`、`failed_count=0`。

已完成补充：双栈报告逻辑现在不再只属于 laoxia，而是可以给后续其他成对项目复用。下一步若再接一个类似样本，优先复用 shared helper，再只提供项目默认值和名称。

## 第三百六十四阶段：入口输出目录输入合同统一

1. [x] `scripts/doctor-first-run.sh`、`scripts/run-first-run-ci.sh`、`scripts/run-onboarding-ci.sh`、`scripts/showcase-agent-onboarding-report.sh` 和 `scripts/showcase-dual-project-report.sh` 已统一在写入前校验 `*_OUTPUT_DIR` 必须是目录。
2. [x] `scripts/showcase-first-run-ci-external-project.sh` 和 `scripts/showcase-onboarding-ci-external-project.sh` 已补上相同的输出目录早失败保护，避免坏路径落到 `mkdir -p` 的底层报错。
3. [x] `scripts/validate-regression-smoke.sh` 以及 `scripts/validate-java-regression-samples.sh`、`scripts/validate-js-regression-samples.sh`、`scripts/validate-py-regression-samples.sh` 已统一在跑样本前校验输出目录输入。
4. [x] `test/showcase_scripts_test.sh`、`test/regression_smoke_test.sh` 和 `test/regression_samples_test.sh` 已覆盖这些坏路径早失败分支。
5. [x] `.github/workflows/ci.yml` 已把新回归测试纳入默认 CI。
6. [x] `docs/showcase.md` 和 `docs/regression-smoke.md` 已补充这类入口会先校验输出目录输入的说明。

已完成补充：这一轮把展示入口和回归入口的输出目录合同统一了，坏路径会更早、更一致地失败。下一步应先等 CI 消化这几条提交，再继续看是否还要把 file 输出类入口的父目录校验也收口。

## 第三百六十五阶段：文件输出路径输入合同统一

1. [x] `scripts/generate-verification-report.sh` 已在生成 Markdown 报告和 summary JSON 前校验输出路径不能是目录。
2. [x] `scripts/showcase-go-public-project.sh` 和 `scripts/showcase-js-public-project.sh` 已在生成 JSONL 之前先拦截坏的输出路径输入。
3. [x] `scripts/validate-go-coverage-top-tasks.sh`、`scripts/validate-js-coverage-top-tasks.sh`、`scripts/validate-java-coverage-top-tasks.sh` 和 `scripts/validate-py-coverage-top-tasks.sh` 已统一校验 `output-jsonl` 不能是目录。
4. [x] `test/verification_report_test.sh`、`test/showcase_scripts_test.sh` 和 `test/validate_coverage_top_tasks_output_test.sh` 已覆盖这些 file-output 负例。
5. [x] `.github/workflows/ci.yml` 已把新回归测试纳入默认 CI。

已完成补充：现在目录输出和文件输出这两层入口都不再把坏路径推迟到写文件时才失败。下一步若继续收口，优先看是否还有其他对外入口仍然只靠底层 OS 错误返回。

## 第三百六十六阶段：validate_coverage_task 结构化一致性说明对齐

1. [x] 确认 `tools/validate_coverage_task_test.go` 已有 `TestHandleValidateCoverageTaskStructuredContentMatchesTextJSON`，覆盖 `validate_coverage_task` 的 `structuredContent`、handler 返回值和 text JSON 一致性。
2. [x] 该测试已改为复用 `tools/tool_result_contract_test.go` 中的 `assertStructuredContentMatchesTextJSON` helper，减少重复断言逻辑。
3. [x] `docs/agent-contract.md` 已补充说明：集中契约测试覆盖五个主工具，`validate_coverage_task` 在专属测试中覆盖同类一致性。

已完成补充：结构化返回一致性并没有漏掉 `validate_coverage_task`，只是原文档表达不完整。下一步继续优先看真实项目演示与 Agent 消费路径里是否还有字段契约说明不一致。

## 第三百六十七阶段：laoxia 双栈入口最新源码复验

1. [x] 用最新源码构建 `/tmp/testloop-mcp-latest`。
2. [x] 用 `scripts/showcase-laoxia-scaffold-report.sh` 复验 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server`，用户项目命令为 `go test ./...`。
3. [x] 用同一入口复验 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-web`，用户项目命令为 `pnpm install --frozen-lockfile && pnpm build:prod`。
4. [x] 输出目录 `/tmp/testloop-laoxia-scaffold-live-20260720154617` 的顶层 `laoxia-summary.json`、server 子 summary 和 web 子 summary 均为 `overall_status=passed`、`failed_count=0`。
5. [x] 最新远端主线提交 `66ebb98 fix: check showcase output before temp setup` 的 GitHub Actions 已通过，证明上一轮脚本输出路径加固没有破坏默认 CI。
6. [x] `docs/real-integration-cases.md` 已记录本次复验命令、输出目录和 server/web 状态，后续可作为最新真实项目 smoke 证据复查。

已完成补充：真实 laoxia server/web 双栈 smoke 已用最新源码复验通过，和最新主线 CI 状态一致。下一步应继续围绕 Agent 可消费闭环推进，优先检查真实项目报告中的 `action`、`agent_next_step`、summary 嵌套字段是否有可自动校验的契约缺口。

## 第三百六十八阶段：双项目 combined summary 文件路径契约收口

1. [x] `scripts/showcase-dual-project-report.sh` 现在会显式拒绝 `TESTLOOP_PAIR_SUMMARY_JSON` 指向目录。
2. [x] `scripts/showcase-laoxia-scaffold-report.sh` 复用 shared helper 后，同样继承 combined summary JSON 的文件路径契约。
3. [x] `test/showcase_scripts_test.sh` 已新增双项目 helper 和 laoxia wrapper 的目录型 summary JSON 负例。
4. [x] `docs/showcase.md` 已把 `TESTLOOP_PAIR_SUMMARY_JSON` 和 `TESTLOOP_LAOXIA_SUMMARY_JSON` 的普通文件要求写入入口说明。

已完成补充：combined summary 现在也和其他 file-output 入口一样，坏路径会更早失败。下一步应继续看是否还有其他入口存在“目录已拦截、文件总结仍靠底层错误”的不对称合同。

## 第三百六十九阶段：verification summary 决策 demo 最小合同校验

1. [x] `examples/verification-summary-decision-demo` 现在会先校验 `overall_status`、`failed_count` 和 `sections` 必填字段。
2. [x] 决策 demo 已拒绝非法 `overall_status`，只接受 `passed` 或 `failed`。
3. [x] `test/verification_summary_decision_demo_test.sh` 已覆盖缺少 `sections` 和非法状态两类非标准 summary 输入。
4. [x] `docs/client-integration.md` 和 `docs/fixtures.md` 已说明 decision demo 会先做最小 summary 合同校验，再输出 `agent_next_step`。
5. [x] 真实 `laoxia-summary.json` 已复验为 pair combined summary，直接喂给 decision demo 会因缺少 `sections` 被拒绝，说明两类 summary 不应混用。

已完成补充：Agent 客户端不会再因为拿到缺字段 JSON 就从默认零值推导出 `ready`。下一步应继续检查 artifact manifest / summary schema 的文档和测试是否都把“verification summary”和“双项目 combined summary”区分清楚。

## 第三百七十阶段：双项目 combined summary schema 固化

1. [x] 新增 `docs/fixtures/dual-project-summary.schema.json`，定义 `showcase-dual-project-report.sh` 顶层 combined summary 的机器可读合同。
2. [x] 新增 `docs/fixtures/dual-project-summary/laoxia-passed.json`，作为 laoxia server/web passed 样例。
3. [x] 新增 `tools/dual_project_summary_schema_test.go`，验证 fixture 符合 schema，并拒绝缺第二个项目或子 summary 缺 `sections` 的输入。
4. [x] `docs/fixtures.md`、`docs/showcase.md` 和 `docs/real-integration-cases.md` 已指向新的 combined summary schema，明确它和 `verification-summary.schema.json` 是两类合同。
5. [x] `TestDualProjectSummaryScriptOutputMatchesSchema` 会运行一次 fake 双项目报告入口，并用 schema 校验实际生成的 `pair-summary.json`。
6. [x] `docs/mcp-client-contract-tests.md` 和客户端文档测试已把 `dual-project-summary.schema.json` 纳入接入方契约示例。

已完成补充：双项目报告现在不只是“脚本能输出 JSON”，而是有了可供客户端生成类型或做回归的 schema。下一步应跑 schema、文档和全量 Go 验证；通过后提交并观察 CI。

## 第三百七十一阶段：双项目报告 artifact schema 自包含

1. [x] `scripts/showcase-dual-project-report.sh` 会把 `docs/fixtures/dual-project-summary.schema.json` 复制到 combined summary 同目录。
2. [x] 脚本输出新增 `<prefix>_summary_schema=<path>`，让 Agent / CI 不必再从仓库路径猜 schema 位置。
3. [x] `test/showcase_scripts_test.sh` 已覆盖 generic pair 和 laoxia wrapper 都会输出并落盘 `dual-project-summary.schema.json`。
4. [x] `docs/showcase.md` 和 `docs/real-integration-cases.md` 已把 schema 文件列入双项目报告 artifact。

已完成补充：双项目报告 artifact 下载到本地后可以离线校验 combined summary，不依赖调用方知道仓库内 schema 路径。下一步应跑展示脚本、schema、文档和全量 Go 验证；通过后提交并观察 CI。

## 第三百七十二阶段：verification summary artifact schema 自包含

1. [x] `scripts/generate-verification-report.sh` 现在写 `verification-summary.json` 时，会同时复制 `verification-summary.schema.json` 到同目录。
2. [x] 标准报告输出会打印 schema 文件路径，方便 CI 日志直接暴露 artifact 合同位置。
3. [x] `test/verification_report_test.sh` 已覆盖标准 report 成功/失败路径都会落盘 schema。
4. [x] `test/showcase_agent_onboarding_report_test.sh`、`test/run_onboarding_ci_test.sh` 和 `test/run_first_run_ci_test.sh` 相关回归已覆盖 onboarding / first-run artifact 目录包含 schema。
5. [x] `README.md`、`docs/verification-report.md`、`docs/verification-ci.md`、`docs/first-run-ci-template.md`、`docs/onboarding-ci-template.md`、`docs/adopter-verification-guide.md` 和 triage 文档已同步说明 summary artifact 会自带 schema。

已完成补充：单项目 verification summary 和双项目 combined summary 现在都可以随 artifact 离线校验。下一步应跑标准报告、onboarding、first-run、文档和全量 Go 验证；通过后提交并观察 CI。

## 第三百七十三阶段：Agent artifact fixture schema 自包含

1. [x] `docs/fixtures/first-run-artifacts/user-project-smoke-failed/` 已补 `verification-summary.schema.json`，静态失败 fixture 和真实 first-run CI artifact 一样可离线校验。
2. [x] `docs/fixtures/onboarding-artifacts/user-project-smoke-failed/` 已补 `verification-summary.schema.json`，静态 onboarding fixture 和真实 onboarding CI artifact 一样可离线校验。
3. [x] `docs/fixtures/agent-response-artifact-manifest.json` 的每个 artifact 已新增本地 `summary_schema: "verification-summary.schema.json"` 指针。
4. [x] manifest schema、manifest demo 和 artifact fixture 回归已覆盖本地 summary schema 文件存在且可消费。
5. [x] README、showcase、CI 集成、接入指南、fixture 索引和 artifact contract 当前说明已从 first-run 六件套 / onboarding 四件套更新为 first-run 七件套 / onboarding 五件套。

已完成补充：真实 CI artifact、静态 artifact fixture 和 manifest 现在都能自包含 verification summary schema，Agent / 客户端下载 artifact 后不需要回到仓库路径猜合同。下一步应跑 artifact fixture、manifest、release doc index、文档链接、schema 和全量 Go 验证；通过后提交并观察 CI。

## 第三百七十四阶段：默认 CI 覆盖所有 shell 契约测试

1. [x] `.github/workflows/ci.yml` 已补跑 first-run / onboarding Agent response、artifact manifest、artifact fixture、外部 dry-run 文档、接入指南、README snippet 和 MCP 客户端契约等此前未纳入的 `test/*_test.sh`。
2. [x] 新增 `test/ci_workflow_test.sh`，自动检查每个 `test/*_test.sh` 都在默认 CI 中显式出现，避免新增测试后只在本地运行。
3. [x] 本地已用 `for script in test/*_test.sh; do sh "$script"; done` 跑完整 shell 回归，确认全量脚本集合可顺序执行。
4. [x] 保留 CI 中的 `go test ./...`、二进制 build 和 Docker build 步骤；本机 Docker daemon 未启动，本地只能验证到 Go build。

已完成补充：默认 CI 不再只覆盖部分契约测试，后续 artifact/文档/客户端契约漂移会更早在远端暴露。下一步应提交并观察最新 CI；通过后继续找 Agent 消费链路里还未机器校验的字段或示例。

## 第三百七十五阶段：Python demo 缓存文件出库

1. [x] 确认 `.gitignore` 已忽略 `__pycache__/`。
2. [x] 从 Git 索引移除 `demo-python/__pycache__/calc.cpython-313.pyc` 和 `demo-python/__pycache__/test_calc.cpython-313-pytest-9.1.1.pyc`。
3. [x] 使用 `git ls-files | rg '\.pyc$|__pycache__'` 确认仓库不再跟踪 Python bytecode 缓存。
4. [x] `sh test/py_regression_fixture_test.sh` 已通过，说明 Python regression fixture 不依赖这些缓存文件。

已完成补充：Python demo 不再把本机解释器生成物作为仓库内容维护。下一步继续做低风险巡检，优先找已忽略但仍被跟踪的构建产物或文档/CI 入口漂移。

## 第三百七十六阶段：仓库卫生检查入 CI

1. [x] `.gitignore` 已为有意保留的 `demo-node/jest_output.txt`、`demo-python/pytest_output.txt` 和 first-run fixture `first-run.log` 增加精确例外。
2. [x] 新增 `test/repository_hygiene_test.sh`，检查 `git ls-files -ci --exclude-standard` 为空，避免已跟踪 fixture 被 ignore 规则误伤。
3. [x] 同一测试还会拒绝重新跟踪 `__pycache__/` 或 `.pyc` 文件。
4. [x] 默认 CI 已加入该仓库卫生测试。

已完成补充：仓库现在不仅移除了 Python bytecode，也能防止后续重新提交被忽略的生成缓存。下一步应跑仓库卫生、CI workflow 自检、完整 shell 回归和 Go 测试；通过后提交并观察 CI。

## 第三百七十七阶段：Unreleased 记录 post-release 收口

1. [x] `CHANGELOG.md` 的 `Unreleased` 已补齐 v0.5.13 tag 之后的双项目报告、summary schema 自包含、Agent artifact fixture schema、默认 CI 覆盖和仓库卫生检查。
2. [x] `Unreleased` 已按 Added / Changed / Fixed / Removed 分组，避免把 post-release 改动误塞回已发布的 v0.5.13。
3. [x] 删除 Python bytecode、输出路径校验、combined summary failed count 和 verification summary decision contract 校验已进入未来版本变更记录。

已完成补充：后续如果准备 v0.5.14，不需要重新从 commit log 里捞这些 post-release 改动。下一步应跑 changelog 相关文档检查、文档链接、完整 shell 回归和 Go 测试；通过后提交并观察 CI。

## 第三百七十八阶段：Agent artifact 下载目录校验入口

1. [x] 新增 `examples/agent-artifact-verify`，可校验 first-run/onboarding artifact 目录的必备文件、同目录 `verification-summary.schema.json`、summary 语义、decision action、Agent response 四段结构、失败 section、`exit_code` 和 `section_signal`。
2. [x] 新增 `scripts/verify-agent-artifact.sh`，把 Go verifier 包装成接入方可直接复制的 shell 入口。
3. [x] 新增 `test/agent_artifact_verify_test.sh`，覆盖 first-run/onboarding 成功路径、缺 schema、decision 漂移、response signal 漂移和 first-run failed count 漂移。
4. [x] first-run Agent response demo 已补齐 `first_run_status` 和 `first_run_failed_count` 输出，静态 first-run artifact fixture 与 contract 恢复一致。
5. [x] README、artifact contract、客户端集成说明、MCP 客户端契约测试说明、showcase 和 fixture 维护说明已同步 verifier 入口。
6. [x] 默认 CI 已加入 `test/agent_artifact_verify_test.sh`，文档测试已固定 `sh scripts/verify-agent-artifact.sh`、`agent_artifact_status=passed` 和 `response_action=inspect-user-project`。

已完成补充：CI artifact 现在不只是“能渲染 Agent 回复”，还可以对下载后的目录做一次机器可读的完整自检。下一步应跑 artifact verifier、first-run response、artifact fixture、文档 gate、完整 shell 回归和 Go 测试；通过后提交并观察 CI。随后优先看是否要把这个 verifier 接入 first-run/onboarding 复制模板的可选 CI step。

## 第三百七十九阶段：复制型 bootstrap 自动 artifact 自检

1. [x] `scripts/run-first-run-ci.sh` 在渲染 `agent-response.txt` 后会调用 `scripts/verify-agent-artifact.sh first-run <output-dir>`，helper 不支持或没有 Go 时只 warning 跳过。
2. [x] `scripts/run-onboarding-ci.sh` 在渲染 `agent-response.txt` 后会调用 `scripts/verify-agent-artifact.sh onboarding <output-dir>`，并保留旧 tag helper 的兼容跳过路径。
3. [x] 两个 bootstrap 的 GitHub step summary 都会写入 `Artifact verification`，便于用户在 Actions 页面直接确认 artifact 合同是否自检通过。
4. [x] `test/run_first_run_ci_test.sh` 和 `test/run_onboarding_ci_test.sh` 已覆盖成功与用户项目失败两类路径都会输出 `agent_artifact_status=passed`、`artifact_kind` 和 `response_action`。
5. [x] first-run/onboarding 复制模板、验收 CI 文档、接入方一页式指南和 README 已同步自动 verifier 与手动复跑命令。

已完成补充：用户项目 CI 生成 artifact 后，现在会在能力可用时立刻验证目录合同，减少“上传成功但 Agent 消费失败”的延迟发现。下一步应跑 bootstrap、模板文档、README/release 索引、完整 shell 回归和 Go 测试；通过后提交并观察 CI。随后继续看是否要给 artifact verifier 增加 manifest 批量模式，减少接入方在客户端测试中手写两条命令。

## 第三百八十阶段：Agent artifact manifest 批量校验

1. [x] `examples/agent-artifact-verify` 支持 `manifest <agent-response-artifact-manifest.json>` 模式，一次读取 manifest 中的 first-run/onboarding artifact 列表。
2. [x] manifest 模式复用单目录 verifier，并额外核对 `action_field`、`expected_action`、`expected_failed_section`、`expected_exit_code` 和 `expected_section_signals`。
3. [x] `scripts/verify-agent-artifact.sh` 的帮助信息已补 manifest 用法。
4. [x] `test/agent_artifact_verify_test.sh` 已覆盖 manifest 成功输出和 manifest 期望 action 漂移失败场景。
5. [x] README、artifact contract、客户端集成说明、MCP 客户端契约测试说明和 fixture 维护说明已同步 manifest 批量命令。

已完成补充：客户端或维护者现在可以用一条 verifier 命令校验 manifest 登记的全部 Agent artifact fixture。下一步应跑 verifier、manifest demo、文档 gate、完整 shell 回归和 Go 测试；通过后提交并观察 CI。随后优先评估是否要把 artifact verifier 的输出转成 JSON，进一步降低外部客户端解析成本。

## 第三百八十一阶段：Agent artifact verifier JSON 输出

1. [x] `examples/agent-artifact-verify` 支持可选 `--json` 参数，单目录模式输出 `status`、`artifact_kind`、`decision_action`、`response_action`、`section_signals` 和 `required_files`。
2. [x] manifest 模式的 `--json` 输出包含 `status`、`manifest_schema_version`、`artifact_count` 和 `artifacts[]`，方便客户端直接解析批量校验结果。
3. [x] `scripts/verify-agent-artifact.sh` 已透传 `--json`，默认文本输出保持不变，bootstrap 现有日志消费不受影响。
4. [x] `test/agent_artifact_verify_test.sh` 已覆盖单目录 JSON 和 manifest JSON 输出。
5. [x] README、artifact contract、客户端集成说明、MCP 客户端契约测试说明和 fixture 维护说明已补 `--json` 用法。

已完成补充：artifact verifier 现在既适合人看日志，也适合外部客户端做结构化断言。下一步应跑 verifier、文档 gate、完整 shell 回归和 Go 测试；通过后提交并观察 CI。随后回到更高收益的真实接入链路，优先复验 laoxia server/web bootstrap 输出中 `Artifact verification=passed` 是否能作为真实案例证据。

## 第三百八十二阶段：laoxia 真实 bootstrap artifact 自检复验

1. [x] 使用当前源码构建 `/tmp/testloop-mcp-latest`，版本输出 `testloop-mcp 0.5.13`。
2. [x] 对 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server` 运行 onboarding bootstrap，用户项目命令为 `go test ./...`。
3. [x] server 输出目录 `/tmp/testloop-laoxia-server-onboarding-artifact-verify` 的 summary 为 `overall_status=passed`、`failed_count=0`，decision 为 `agent_next_step=ready`，verifier 输出 `agent_artifact_status=passed`。
4. [x] 对 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-web` 运行 onboarding bootstrap，用户项目命令为 `pnpm install --frozen-lockfile && pnpm build:prod`。
5. [x] web 输出目录 `/tmp/testloop-laoxia-web-onboarding-artifact-verify` 的 summary 为 `overall_status=passed`、`failed_count=0`，decision 为 `agent_next_step=ready`，verifier 输出 `agent_artifact_status=passed`。
6. [x] 两份本地 step summary 均包含 `Artifact verification: passed`，两个外部项目的 git 状态均为空。
7. [x] `docs/real-integration-cases.md` 已记录这次真实项目复验命令、输出路径和结果，文档测试已固定关键证据。

已完成补充：artifact verifier 已从静态 fixture 和 bootstrap 单元回归推进到真实 laoxia server/web 项目证据。下一步应跑真实接入案例文档测试、文档链接、完整 shell 回归和 Go 测试；通过后提交并观察 CI。随后可以开始整理 v0.5.14 候选边界，把 post-v0.5.13 的 artifact 自检、JSON 输出和真实项目证据收拢成 release note。

## 第三百八十三阶段：v0.5.14 候选发布边界整理

1. [x] 新增 `docs/plan-release-notes-v0.5.14.md`，整理 v0.5.13 之后的 artifact 自检、manifest 批量校验、JSON 输出、schema 自包含、CI 覆盖和真实项目证据。
2. [x] 新增 `docs/plan-release-v0.5.14.md`，记录候选内容、已验证命令、远端 CI 状态、发布前门禁和正式发布待办。
3. [x] 明确当前只是候选边界：不更新 implementation version、不切 CHANGELOG 正式段、不打 tag、不生成 Release assets、不更新 Homebrew tap。
4. [x] 候选文档已记录最新 `c36758b` CI run `29737179225` 仍在 GitHub Actions 队列中，等待 runner 执行。

已完成补充：v0.5.14 的发布范围已经可读、可复查，后续不会从零开始整理 release note。下一步应跑文档链接、release doc index、完整 shell 回归和 Go 测试；通过后提交并继续观察最新 CI。等 CI 通过后，再补 release readiness 门禁或进入正式版本准备。

## 第三百八十四阶段：候选发布门禁脚本化

1. [x] 新增 `scripts/verify-release-candidate.sh`，把候选发布本地门禁收敛成维护者一键入口。
2. [x] 脚本会执行 shell 语法检查、`go test ./...`、全部 `test/*_test.sh`、候选二进制构建、`--version`、`--help`、release asset dry-run、sha256 校验、archive 内容校验和 `git diff --check`。
3. [x] 脚本只做本地验证，不更新版本号、不打 tag、不创建 GitHub Release、不更新 Homebrew tap。
4. [x] 新增 `test/release_candidate_script_test.sh` 固定脚本帮助、参数校验和关键门禁步骤，并纳入默认 CI。
5. [x] README 和 v0.5.14 候选发布文档已暴露 `scripts/verify-release-candidate.sh v0.5.14` 入口。

已完成补充：v0.5.14 候选门禁不再只靠手工复制 checklist。下一步应用该脚本跑一轮真实候选验证；通过后提交并观察最新 CI，CI 通过后再进入正式版本准备。

## 第三百八十五阶段：v0.5.14 正式版本准备

1. [x] `main.go` MCP implementation version 已更新到 `0.5.14`。
2. [x] `CHANGELOG.md` 已将 Unreleased 内容收敛为 `v0.5.14 - 2026-07-20`，并保留新的空 Unreleased。
3. [x] README、installation、quickstart、first-run/onboarding/verification CI 文档和测试期望已同步到 `0.5.14` / `v0.5.14`。
4. [x] targeted 版本准备测试已通过：主模块和 `cmd/testgen` Go 测试、README/installation/quickstart/adopter/CI 模板文档测试、first-run/onboarding wrapper、doctor、verification report、showcase 和 client setup 测试。
5. [x] 完整本地门禁已通过：`scripts/verify-release-candidate.sh v0.5.14` 输出 `release_candidate_status=passed`，候选二进制 `--version` 输出 `testloop-mcp 0.5.14`。

已完成补充：v0.5.14 正式版本准备文件和本地门禁都已完成，但尚未提交版本准备、打 tag、生成 Release assets 或更新 Homebrew tap。下一步应提交版本准备并等待 main CI，通过后再进入发布动作。

## 第三百八十六阶段：v0.5.14 Artifact 自检发布收敛

1. [x] 固化 `scripts/verify-release-candidate.sh`，把候选发布本地门禁收敛为一条维护者命令。
2. [x] 修正 `testloop-mcp --help` 和 `testgen --help` 退出码为 0，并同步 Release Artifacts、Post-Release Verify、Windows ARM64 Probe 和 Homebrew Formula 生成器。
3. [x] 发布 `v0.5.14`，生成 Linux amd64、Linux arm64、macOS arm64、Windows amd64 和 Windows arm64 五个平台资产。
4. [x] 用 `scripts/verify-release-assets.sh v0.5.14` 校验正式 Release 的 10 个资产完整。
5. [x] 更新仓库内 Homebrew Formula 和 `sleticalboy/homebrew-tap` 到 `0.5.14`。
6. [x] Post-Release Verify run `29740930414` 已通过，覆盖资产清单和五个平台安装脚本 dry run。

已完成补充：v0.5.14 已完成正式 GitHub Release、五平台资产发布、资产清单校验、GitHub Release 正文、仓库内 Formula、Homebrew tap 和 Post-Release Verify。下一步回到主线产品价值，继续打磨真实项目 Agent 闭环。

## 第三百八十七阶段：真实项目 Agent 闭环演示

1. [x] 选一个真实项目路径做端到端 Agent 闭环演示：覆盖率任务选择、生成增量测试、运行测试、解析失败、生成 repair task。
2. [x] 优先打磨 `run_tests.include_fix_suggestions` 和 `coverage_task -> generate_tests` 的真实项目可用性，而不是继续扩语言宣传面。
3. [x] 为真实项目闭环输出最小可复制 demo 文档和 fixture，形成后续发布的核心卖点证据。
4. [x] 让客户端决策 demo 同时读取真实项目脱敏 fixture，确认真实项目证据也走同一套 `status/action` 分流。

已完成补充：用 laoxia `car-admin-server` 的 `utils` 包跑通 1 个真实 Go coverage task。`scripts/validate-go-coverage-top-tasks.sh` 输出 `status_counts.passed=1`、`action_counts.ready=1`、`zero_skip=1`、`skipped_total=0`；脱敏 fixture 固定在 `docs/fixtures/real-project-agent-loop/laoxia-server-go-utils.json`，避免提交外部项目原始日志和本机环境变量。

已完成补充：用 mcp-hub `ConfigManager.loadConfig` 跑通 1 个真实 Vitest 历史 repair 回归样本。`scripts/validate-js-coverage-top-tasks.sh` 输出 `vitest-mcp-hub-repair-1 status=passed action=ready`、`zero_skip=1`、`skipped_total=0`；脱敏 fixture 固定在 `docs/fixtures/real-project-agent-loop/mcp-hub-vitest-repair.json`，用于防止 async throwing branch 回退到 `repair_generated_test`。下一步应把这两个真实项目 fixture 接入更直接的客户端消费 demo，或者扩展一个 `manual_review_*` 真实项目样本。

已完成补充：`examples/agent-decision-demo` 现在会读取根目录 `validate-coverage-task-*.json` 和 `real-project-agent-loop/*.json`，并统一输出 `accept/manual-review/apply-repair/needs-better-input` 决策；客户端集成说明和契约测试文档已同步真实项目 fixture 入口。下一步优先补一个真实项目 `manual_review_*` fixture，让客户端 demo 同时覆盖 ready、repair、needs-better-input 和真实项目手审分流。

已完成补充：用 haoy-apk-station `serve_frontend` 跑通 1 个真实 pytest 环境手审样本。`scripts/validate-py-coverage-top-tasks.sh` 输出 `pytest-apk-frontend-env-1 status=passed action=manual_review_environment`、`zero_skip=0`、`skipped_total=1`；脱敏 fixture 固定在 `docs/fixtures/real-project-agent-loop/haoy-apk-station-py-environment.json`，客户端 demo 现在也会把真实项目手审样本映射成 `manual-review`。下一步可以继续补一个真实项目 `failed/apply_fix_suggestions` 样本，或开始把真实项目 fixture 打包成更适合外部客户端复制的最小目录。

已完成补充：新增 `docs/fixtures/agent-decision-fixtures.json` 和 schema，把最小 Agent 决策样本、真实项目 fixture、`status/action` 与期望客户端 decision 固定成机器可读清单；`examples/agent-decision-demo` 已改为读取该 manifest，`test/agent_decision_fixtures_manifest_test.sh` 已纳入 CI。下一步优先寻找可稳定复现的真实项目 `failed/apply_fix_suggestions` 样本；如果短期没有合适来源，则继续强化 manifest/schema 的外部客户端复制路径。

已完成补充：用 haoy-apk-station `download_apk` 跑通 1 个真实 pytest 外部服务失败样本。`scripts/validate-py-coverage-top-tasks.sh` 输出 `pytest-apk-download-external-1 status=failed action=manual_review_external_service`、`zero_skip=1`、`skipped_total=0`；脱敏 fixture 固定在 `docs/fixtures/real-project-agent-loop/haoy-apk-station-py-external-service.json`，并纳入 Agent decision manifest。下一步继续寻找真正可稳定产出 `failed/apply_fix_suggestions` 的真实项目样本；若没有，应先把 `manual_review_*` 失败分流在 README/客户端模板中讲清楚。

已完成补充：README、Agent Action 决策表、Agent 结构化契约和 validate_coverage_task 样例说明已明确 `failed` 不等于自动修复；`failed/apply_fix_suggestions` 才进入 repair task 闭环，`failed/manual_review_external_service` 等 `manual_review_*` 需要转 fake client、依赖注入或集成环境验证。下一步应把同一语义继续落到可复制客户端模板或示例脚本中，减少接入方自己写分流逻辑时漏掉 failed/manual-review 的概率。

已完成补充：MCP 客户端契约测试说明已把示例客户端检查改为 manifest 驱动：读取 `agent-decision-fixtures.json` 的 `fixtures[]`、逐项校验 `status/action` 和 `expected_decision`，并明确 `manual_review_*` 不触发同一生成测试的自动修复循环。下一步继续寻找真实 `failed/apply_fix_suggestions` 样本，或者把 manifest 驱动检查提炼成一个可执行客户端模板脚本。

已完成补充：新增无第三方依赖的 Node 参考脚本 `scripts/validate-agent-decision-fixtures.mjs`，默认校验仓库内 `agent-decision-fixtures.json`，也支持外部客户端传入 manifest 路径和 repo root；`test/agent_decision_fixture_validator_test.sh` 已覆盖成功路径和错误 `expected_decision` 失败路径，并纳入 CI。下一步继续寻找真实 `failed/apply_fix_suggestions` 样本，或把该 validator 的输出接入 README 的快速演示路径。

已完成补充：README 的“面向 Agent 的快速演示路径”已加入 `node scripts/validate-agent-decision-fixtures.mjs docs/fixtures/agent-decision-fixtures.json .`，并说明 `fixture_count=8` 与完整 decision 串；`test/release_doc_index_test.sh` 已固定该入口。下一步继续寻找真实 `failed/apply_fix_suggestions` 样本，或者为 validator 增加 JSON 输出，方便外部客户端 CI 直接断言。

已完成补充：短窗口复查真实 `failed/apply_fix_suggestions` 候选后，没有找到比现有 fixture 更稳定的真实项目来源；当前真实失败更适合归类为 `manual_review_external_service`，历史普通 repair 样本也已经收敛为 `passed/ready`。本阶段先不强行提交漂移样本，改为增强可复制客户端路径。

## 第三百八十八阶段：Agent 决策 validator JSON 输出

1. [x] `scripts/validate-agent-decision-fixtures.mjs` 新增 `--json` 选项，默认文本输出保持兼容。
2. [x] JSON 输出固定 `status`、`fixture_count`、`decisions[]`、`fixtures[]` 和 `failures[]`，方便外部客户端 CI 直接断言。
3. [x] 验证失败时仍输出可解析 JSON，同时用非 0 退出码表达失败。
4. [x] `test/agent_decision_fixture_validator_test.sh` 已覆盖文本成功、JSON 成功、文本失败和 JSON 失败路径。
5. [x] README、客户端集成说明和 MCP 客户端契约测试说明已同步 `--json` 推荐入口，文档测试固定关键字段。

已完成补充：Agent 决策 fixture 已从“可运行 demo”推进到“可被外部 CI 机器消费”的形态。下一步应跑完整本地门禁；通过后提交并观察 main CI。随后继续补一个更贴近外部客户端复制的最小 fixture 包或 npm/pnpm 项目模板，降低接入成本。

## 第三百八十九阶段：Agent 决策 fixture 最小导出包

1. [x] 新增 `scripts/export-agent-decision-fixtures.mjs`，把 Agent 决策 manifest、schema、8 个决策 fixture 和 validator 脚本导出到指定目录。
2. [x] 导出包保留 `docs/fixtures/...` 路径，避免改写 manifest 和 schema 后产生第二套路径合同。
3. [x] 导出目录会生成最小 `README.md`，提示接入方直接运行 `node scripts/validate-agent-decision-fixtures.mjs --json docs/fixtures/agent-decision-fixtures.json .`。
4. [x] 导出目录会生成无依赖 `package.json`，接入方可以直接运行 `npm test --silent`。
5. [x] 新增 `test/agent_decision_fixture_export_test.sh`，验证导出包内容、复制后的 JSON 校验、`npm test --silent`、非空目录拒绝覆盖，并纳入 CI。
6. [x] README、客户端集成说明和 MCP 客户端契约测试说明已暴露 `node scripts/export-agent-decision-fixtures.mjs /tmp/testloop-agent-decision-fixtures`。

已完成补充：外部客户端现在可以不复制整个仓库，只复制一个最小 fixture 包来验证 `status/action -> decision` 合同。下一步应跑完整本地门禁；通过后提交并观察 main CI。之后继续评估是否需要把导出包升级成 npm/pnpm 可执行模板，或者先补一份真实客户端仓库接入示例。

## 第三百九十阶段：Agent 决策 manifest 元数据校验

1. [x] `scripts/validate-agent-decision-fixtures.mjs` 增加无 AJV 依赖的 manifest 条目元数据校验。
2. [x] validator 现在会显式检查 `kind`、`source`、`status`、`action`、`expected_decision` 和 `client_expectation`。
3. [x] `test/agent_decision_fixture_validator_test.sh` 新增错误 `source` 和空 `client_expectation` 的 JSON 失败路径。
4. [x] README、客户端集成说明和 MCP 客户端契约测试说明已同步 validator 的轻量合同边界。

已完成补充：导出的客户端 fixture 包不再只校验 payload 内容，也会拦截 manifest 元数据漂移。下一步应跑完整本地门禁；通过后提交并观察 main CI。随后可开始补真实客户端仓库接入示例，或者把导出包命令接入 release readiness 检查。

## 第三百九十一阶段：发布候选门禁覆盖 Agent 决策导出包

1. [x] `scripts/verify-release-candidate.sh` 新增显式 `verify agent decision fixture export package` 步骤。
2. [x] release readiness 现在会导出最小 Agent 决策 fixture 包，并在导出目录运行 `npm test --silent`。
3. [x] release readiness 提前检查 `node` 和 `npm`，避免到 shell 子测试或导出包阶段才出现不清晰错误。
4. [x] `test/release_candidate_script_test.sh` 已固定导出包 release step、`node/npm` 检查和 npm JSON 校验输出路径。
5. [x] 完整 dry-run 门禁已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.15-candidate-dist scripts/verify-release-candidate.sh v0.5.15` 输出 `release_candidate_status=passed`，其中导出包 step 输出 `fixture_count=8`。

已完成补充：Agent 决策 fixture 导出包已经纳入正式候选发布门禁，不再只是普通 CI 子测试。下一步应提交并观察 main CI；通过后可以补真实客户端仓库接入示例，或者整理下一版候选范围。当前只是 v0.5.15 dry-run，不更新版本号、不打 tag。

## 第三百九十二阶段：v0.5.15 候选边界整理

1. [x] 新增 `docs/plan-release-notes-v0.5.15.md`，整理 v0.5.14 之后的真实项目 Agent 决策 fixture、manifest 驱动客户端契约、JSON validator、最小导出包和 release readiness 门禁。
2. [x] 新增 `docs/plan-release-v0.5.15.md`，记录候选内容、已验证命令、远端 CI 证据、发布前门禁和正式发布待办。
3. [x] 明确当前只是候选边界：不更新 implementation version、不收敛 `CHANGELOG.md` 正式版本段、不打 tag、不创建 GitHub Release、不更新 Homebrew tap。
4. [x] 候选文档已记录完整 dry-run 门禁：`scripts/verify-release-candidate.sh v0.5.15` 输出 `release_candidate_status=passed`，导出包 step 输出 `fixture_count=8`。
5. [x] 候选文档已记录 `153574f` 远端 CI run `29750125793` passed，覆盖 release readiness 显式校验 Agent 决策 fixture 导出包。

已完成补充：v0.5.15 的候选发布范围已经成文，后续如果进入正式发布，不需要重新从 commit log 整理边界。下一步应跑文档链接、release doc index、完整本地验证和 diff check；通过后提交并观察 main CI。

## 第三百九十三阶段：v0.5.15 正式版本准备

1. [x] `main.go` MCP implementation version 已更新到 `0.5.15`。
2. [x] `CHANGELOG.md` 已将 Unreleased 内容收敛为 `v0.5.15 - 2026-07-20`，并保留新的空 Unreleased。
3. [x] README、installation、quickstart、first-run/onboarding/verification CI 文档和测试期望已同步到 `0.5.15` / `v0.5.15`。
4. [x] `docs/plan-release-v0.5.15.md` 和 `docs/plan-release-notes-v0.5.15.md` 已更新为正式版本准备状态，并记录 `34f0954` 远端 CI run `29750391251` passed。
5. [x] 版本准备后的完整本地门禁已通过：`scripts/verify-release-candidate.sh v0.5.15` 输出 `release_candidate_status=passed`，候选二进制 `--version` 输出 `testloop-mcp 0.5.15`。
6. [x] 版本准备提交 `f37b382` 的远端 CI run `29751381326` passed。

已完成补充：v0.5.15 正式版本准备文件、本地门禁和版本准备后的 main CI 都已完成，但尚未打 tag、生成 Release assets 或更新 Homebrew tap。下一步需要确认是否执行正式发布动作：打 `v0.5.15` tag 并进入 Release Artifacts。

## 第三百九十四阶段：v0.5.15 Agent 决策导出包发布收敛

1. [x] 发布 `v0.5.15` tag，Release Artifacts run `29756859746` passed。
2. [x] 五个平台 10 个正式资产已上传：Linux amd64、Linux arm64、macOS arm64、Windows amd64、Windows arm64 及对应 `.sha256`。
3. [x] `TESTLOOP_MCP_REPO=sleticalboy/testloop-mcp scripts/verify-release-assets.sh v0.5.15` 已验证 Release 资产完整。
4. [x] GitHub Release 正文已更新为正式 v0.5.15 发布说明。
5. [x] 仓库内 `Formula/testloop-mcp.rb` 已更新到 `0.5.15`，并通过 `ruby -c Formula/testloop-mcp.rb` 和 `sh test/release_assets_test.sh`。
6. [x] `sleticalboy/homebrew-tap` 已更新到 `0.5.15` 并推送，tap commit `d72ab7d`。
7. [x] Post-Release Verify run `29757718773` passed，覆盖资产清单和五个平台安装脚本 dry run。

已完成补充：v0.5.15 已完成正式 GitHub Release、五平台资产发布、资产清单校验、GitHub Release 正文、仓库内 Formula、Homebrew tap 和 Post-Release Verify。下一步应提交发布完成记录并等待 main CI；CI 通过后回到主线产品价值，继续围绕 Agent/客户端闭环推进。

## 第三百九十五阶段：Agent 决策真实客户端 CI 示例

1. [x] 新增 `scripts/showcase-agent-decision-client-ci.sh`，模拟外部客户端目录，导出最小 Agent 决策 fixture 包并执行导出包内的 `npm test --silent`。
2. [x] 脚本会输出 `agent_decision_client_status=passed`、fixture 包目录、validator JSON 路径、`fixture_count=8` 和完整决策序列，方便客户端 CI 稳定断言。
3. [x] 新增 `test/agent_decision_client_ci_showcase_test.sh`，覆盖帮助、参数错误、非法路径、非空 fixture 目录拒绝覆盖和成功 JSON 校验。
4. [x] 默认 CI 已加入该客户端 CI showcase 测试，避免导出包复制路径或 `npm test --silent` 合同漂移。
5. [x] README、客户端集成说明、MCP 客户端契约测试说明和文档索引测试已同步该真实客户端 CI 示例入口。

已完成补充：Agent 决策 fixture 包现在不只是“能导出”，还具备一条接近外部客户端仓库的 CI 演练路径。下一步应跑该 showcase、文档 gate、CI workflow 自检和全量 Go 验证；通过后提交并观察 main CI。随后继续围绕客户端接入价值，优先评估是否要把这个示例扩成可复制的 GitHub Actions 片段。

## 第三百九十六阶段：Agent 决策客户端 GitHub Actions 复制模板

1. [x] 新增 `docs/agent-decision-client-ci-template.md`，提供客户端仓库可直接复制的 `.github/workflows/testloop-agent-decision-contract.yml`。
2. [x] 模板会 checkout 客户端仓库、设置 Node、checkout `sleticalboy/testloop-mcp` helper，并运行 `.testloop-mcp/scripts/showcase-agent-decision-client-ci.sh`。
3. [x] 模板上传 `agent-decision-fixtures-result.json`、导出包 `package.json` 和 manifest，方便 CI 失败时查看 `failures[]`。
4. [x] 新增文档测试和 YAML 解析测试，固定 workflow 名称、helper checkout、showcase 命令、artifact 路径和关键输出字段。
5. [x] README、客户端集成说明、MCP 客户端契约测试说明和 release 文档索引已同步该复制模板入口。

已完成补充：外部客户端接入方现在不仅能运行本仓库 showcase，也能直接复制一份 GitHub Actions job 放到自己的 MCP 客户端项目里。下一步应跑模板文档/YAML 测试、文档链接、CI workflow 自检、完整 shell 回归和 Go 测试；通过后提交并观察 main CI。

## 第三百九十七阶段：Agent 决策客户端 CI showcase JSON 输出

1. [x] `scripts/showcase-agent-decision-client-ci.sh` 新增 `--json`，输出 `status`、`client_dir`、`fixture_dir`、`result_json`、`fixture_count`、`decisions[]`、`failures[]` 和 `validator_exit_code`。
2. [x] showcase 运行导出包 `npm test --silent` 时会保留 validator 退出码；只要 validator 写出了 JSON，脚本就能把失败摘要结构化输出给 CI / Agent。
3. [x] `test/agent_decision_client_ci_showcase_test.sh` 已覆盖 `--json` 成功路径、输出字段和 result JSON 文件存在性。
4. [x] Agent 决策客户端 GitHub Actions 模板已改用 `--json | tee /tmp/testloop-agent-decision-client-summary.json`，并上传 summary JSON。
5. [x] README、客户端集成说明、MCP 客户端契约测试说明和文档索引测试已同步 JSON 输出字段。

已完成补充：客户端 CI showcase 现在既能给人看 key=value 摘要，也能给 Agent / CI 直接消费结构化 JSON。下一步应跑 showcase、模板、文档 gate、完整 shell 回归和 Go 测试；通过后提交并观察 main CI。

## 第三百九十八阶段：客户端接入变更收敛到 Changelog

1. [x] `CHANGELOG.md` 的 `Unreleased` 已记录 Agent 决策客户端 CI showcase。
2. [x] `CHANGELOG.md` 已记录 Agent 决策客户端 GitHub Actions 复制模板。
3. [x] `CHANGELOG.md` 已记录 `scripts/showcase-agent-decision-client-ci.sh --json` 的机器可读输出字段。
4. [x] `CHANGELOG.md` 已记录 README、客户端集成说明和 MCP 客户端契约测试说明同步到新接入路径。

已完成补充：v0.5.15 之后的客户端接入改动已经进入 changelog，后续准备下一个版本时不需要从三次提交里重新整理范围。下一步应跑 changelog 空白检查、完整 shell 回归和 Go 测试；通过后提交并观察 main CI。

## 第三百九十九阶段：Agent 决策客户端 CI 模板本地 dry-run

1. [x] 新增 `test/agent_decision_client_ci_template_dry_run_test.sh`，在临时外部客户端目录中模拟 `.testloop-mcp` helper checkout。
2. [x] dry-run 按模板相对路径执行 `.testloop-mcp/scripts/showcase-agent-decision-client-ci.sh --json | tee ...`，而不是直接从仓库根目录调用脚本。
3. [x] dry-run 会验证 summary JSON、validator result JSON、导出包 `package.json` 和 manifest 都真实存在。
4. [x] dry-run 会解析 summary JSON 和 validator result JSON，固定 `status=passed`、`fixture_count=8`、完整决策序列和空 `failures[]`。
5. [x] 默认 CI 已加入该 dry-run 测试，避免客户端复制模板只停留在 Markdown/YAML 层面。
6. [x] 修复 `scripts/export-agent-decision-fixtures.mjs` 的仓库根目录定位：外部客户端从 `.testloop-mcp/scripts/...` 调用时不再按客户端 cwd 查找 `docs/fixtures/...`。

已完成补充：Agent 决策客户端 CI 模板现在有了接近真实外部仓库的本地 dry-run 证据。下一步应跑 dry-run、模板文档/YAML、CI workflow 自检、完整 shell 回归和 Go 测试；通过后提交并观察 main CI。

## 第四百阶段：v0.5.16 客户端 CI 候选边界整理

1. [x] 新增 `docs/plan-release-v0.5.16.md`，整理 v0.5.15 之后的客户端 CI showcase、GitHub Actions 模板、JSON 输出、dry-run 和外部 helper 路径修复。
2. [x] 新增 `docs/plan-release-notes-v0.5.16.md`，说明候选重点、主要变化、质量边界、推荐验证和发布备注。
3. [x] 明确当前只是候选边界：不更新 implementation version、不收敛 `CHANGELOG.md` 正式版本段、不打 tag、不创建 GitHub Release、不更新 Homebrew tap。
4. [x] 候选文档已记录当前最新远端 CI 证据：`08ff2a4` run `29797835817` passed。

已完成补充：v0.5.16 的候选发布范围已经成文，后续如果进入正式发布，不需要重新从 commit log 整理客户端 CI 接入边界。下一步应跑文档链接、release doc index、完整 shell 回归和 Go 测试；通过后提交并观察 main CI。

## 第四百零一阶段：v0.5.16 正式版本准备

1. [x] `main.go` MCP implementation version 已更新到 `0.5.16`。
2. [x] `CHANGELOG.md` 已将 Unreleased 内容收敛为 `v0.5.16 - 2026-07-21`，并保留新的空 Unreleased。
3. [x] README、installation、quickstart、first-run/onboarding/verification CI、showcase-onboarding、Agent 决策客户端 CI 模板和测试期望已同步到 `0.5.16` / `v0.5.16`。
4. [x] `docs/plan-release-v0.5.16.md` 和 `docs/plan-release-notes-v0.5.16.md` 已更新为正式版本准备状态，并记录 `63409a6` 远端 CI run `29798075470` passed。
5. [x] 版本准备后的完整本地门禁已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.16-release-prep-dist scripts/verify-release-candidate.sh v0.5.16` 输出 `release_candidate_status=passed`，候选二进制 `--version` 输出 `testloop-mcp 0.5.16`。

已完成补充：v0.5.16 正式版本准备文件和本地门禁已完成，但尚未提交版本准备、打 tag、生成 Release assets 或更新 Homebrew tap。下一步应提交版本准备并等待 main CI；通过后再决定是否执行正式发布动作。

## 第四百零二阶段：v0.5.16 客户端 CI 正式发布收敛

1. [x] `64995fc` 版本准备提交已推送，main CI run `29801283144` passed。
2. [x] `v0.5.16` tag 已推送，Release Artifacts run `29801398746` passed。
3. [x] `TESTLOOP_MCP_REPO=sleticalboy/testloop-mcp scripts/verify-release-assets.sh v0.5.16` 已验证正式 Release 的 10 个资产完整。
4. [x] GitHub Release 正文已更新为正式 v0.5.16 发布说明。
5. [x] 仓库内 `Formula/testloop-mcp.rb` 已更新到 `0.5.16`，并通过 `ruby -c Formula/testloop-mcp.rb` 和 `sh test/release_assets_test.sh`。
6. [x] `sleticalboy/homebrew-tap` 已更新到 `0.5.16` 并推送，tap commit `1de9ae4`。
7. [x] Post-Release Verify run `29801687152` passed，覆盖 Release 资产清单和五个平台安装验证。

已完成补充：v0.5.16 已完成正式 GitHub Release、五平台资产发布、资产清单校验、GitHub Release 正文、仓库内 Formula、Homebrew tap 和 Post-Release Verify。下一步应提交发布完成记录并等待 main CI；CI 通过后回到主线产品价值，继续围绕外部客户端接入和 Agent 决策合同推进。

## 第四百零三阶段：Agent 决策客户端 CI 模板一键安装

1. [x] 新增 `scripts/install-agent-decision-client-ci-template.sh`，可向外部客户端仓库写入 `.github/workflows/testloop-agent-decision-contract.yml`。
2. [x] 脚本默认从 `main.go` 读取当前版本，生成固定 tag 的 `repository: sleticalboy/testloop-mcp` helper checkout。
3. [x] 支持 `--version`、`--workflow-path`、`--force` 和 `--dry-run`，避免接入方只能从 Markdown 手动复制 workflow。
4. [x] 新增 `test/install_agent_decision_client_ci_template_test.sh`，覆盖帮助、dry-run、默认写入、拒绝覆盖、强制覆盖、自定义路径和 YAML 可解析。
5. [x] README、客户端集成说明、MCP 客户端契约测试说明和 Agent 决策客户端 CI 模板文档已同步一键安装入口。
6. [x] 安装脚本测试会比较脚本生成 workflow 与文档 YAML 模板，避免两条接入路径静默分叉。
7. [x] 安装脚本支持脱离仓库单文件运行，默认回退到内置稳定 tag；测试覆盖从临时目录复制脚本后直接生成 workflow，文档使用 `main` raw URL 下载脚本，避免指向不包含该新增脚本的旧 release tag。
8. [x] 安装脚本测试会校验 `default_helper_ref` 与 `main.go appVersion` 同步，避免后续版本准备时 raw 单脚本 fallback 仍指向旧 tag。

已完成补充：外部客户端接入现在从“复制模板”推进到“脚本生成模板”，更接近真实接入动作。下一步应跑新脚本测试、模板文档/YAML、客户端集成文档、showcase 脚本语法、完整 shell 回归和 Go 测试；通过后提交并观察 main CI。

## 第四百零四阶段：Agent 决策客户端 CI 模板安装端到端 dry-run

1. [x] 新增 `scripts/showcase-agent-decision-client-ci-template-install.sh`，覆盖下载安装脚本、生成 workflow、模拟 `.testloop-mcp` helper checkout 和执行 Agent 决策 fixture contract。
2. [x] showcase 支持默认文本输出和 `--json`，输出 installer 来源、客户端目录、workflow 路径、helper ref、fixture 数量、决策序列和退出码。
3. [x] 新增 `test/agent_decision_client_ci_template_install_showcase_test.sh`，覆盖帮助、本地 installer 路径、`file://` installer URL、workflow 内容和 JSON 摘要。
4. [x] 默认 CI 已加入该安装 dry-run，避免 installer 只验证“写文件”，没有验证写出的 workflow 能实际跑 contract。
5. [x] README、客户端集成说明、MCP 客户端契约测试说明和 Agent 决策客户端 CI 模板文档已同步完整安装 dry-run 入口。
6. [x] 新增 `docs/fixtures/agent-decision-client-ci-template-install-summary.schema.json` 和 schema 测试，固定安装 dry-run 的 `--json` 输出字段。
7. [x] `7681645` 远端 CI run `29803269984` passed，覆盖安装 dry-run 脚本和默认 CI 接入。
8. [x] `c1eae25` 远端 CI run `29803596648` passed，覆盖安装 dry-run JSON summary schema。
9. [x] `3f69c58` 远端 CI run `29803739668` passed，覆盖安装脚本版本语义文档修正。
10. [x] 真实 raw 下载路径已验证：`scripts/showcase-agent-decision-client-ci-template-install.sh --json` 从 `https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/install-agent-decision-client-ci-template.sh` 下载 installer 后输出 `status=passed`、`fixture_count=8`、`helper_ref=v0.5.16`。

已完成补充：外部客户端接入链路现在覆盖到“下载 installer 后生成 workflow 并执行 contract”的实际动作，并有本地无网络回归、远端 CI 和真实 raw 下载验证。下一步应把这个接入路径进一步压成接入方可复制的一页式 checklist，减少客户端团队理解成本。

## 第四百零五阶段：Agent 决策客户端 CI 一页式接入 Checklist

1. [x] 新增 `docs/agent-decision-client-ci-checklist.md`，把 helper ref、installer 下载、workflow 生成、CI 运行、artifact、manifest 分流和失败排查压成一页式步骤。
2. [x] 新增 `test/agent_decision_client_ci_checklist_doc_test.sh`，固定 checklist 中的 raw URL、workflow 路径、contract 命令、artifact 路径、JSON 示例和相关文档链接。
3. [x] 新增 `test/agent_decision_client_ci_checklist_commands_test.sh`，从 Markdown 抽取 bash 命令块并实际执行安装、contract 和安装 dry-run 命令。
4. [x] 默认 CI 已加入 checklist 文档测试和命令回归测试。
5. [x] README、Agent 决策客户端 CI 模板、客户端集成说明和 MCP 客户端契约测试说明已同步 checklist 入口。

已完成补充：外部客户端接入现在有“长文档 + 复制模板 + installer + install dry-run + summary schema + 一页式 checklist + checklist 命令回归”七层入口。下一步应跑 checklist 文档/命令测试、文档链接、CI workflow 自检、完整 shell 回归和 Go 测试；通过后提交并观察 main CI。

## 第四百零六阶段：Agent 决策客户端 CI 安装摘要通过态 fixture

1. [x] 新增 `docs/fixtures/agent-decision-client-ci-template-install-summary/passed.json`，固定安装 dry-run 通过态 JSON 摘要样例。
2. [x] 安装 summary schema 测试同时校验实时 dry-run 输出和固定样例字段，避免 schema、脚本输出和文档样例分叉。
3. [x] `docs/fixtures.md`、Agent 决策客户端 CI 模板、Checklist 和客户端集成说明已链接通过态样例。

已完成补充：客户端现在可以同时消费安装 dry-run 的 schema 和固定 golden sample，不必只依赖一次临时目录输出理解字段。下一步应跑安装 summary schema 测试、三份客户端文档测试、文档链接、完整 shell 回归、Go 测试和 diff check；通过后提交并观察 main CI。

## 第四百零七阶段：Agent 决策客户端 CI 安装摘要无依赖校验器

1. [x] 新增 `scripts/validate-agent-decision-client-ci-install-summary.mjs`，默认校验安装 dry-run 通过态 fixture，也可指定任意 summary JSON。
2. [x] 校验器支持文本输出和 `--json` 输出，固定 `fixture_count=8`、决策序列、空 failures、退出码和 installer URL。
3. [x] 新增 `test/agent_decision_client_ci_install_summary_validator_test.sh`，覆盖通过态、JSON 输出和失败摘要分支。
4. [x] 默认 CI 已加入安装摘要 validator 测试，客户端文档、Checklist 和 fixture 索引已同步 validator 入口。

已完成补充：安装 dry-run 输出现在有 schema、golden sample 和无依赖 validator 三层合同。下一步应跑 validator 测试、schema 测试、相关文档测试、CI workflow 自检、完整 shell 回归、Go 测试和 diff check；通过后提交并观察 main CI。

## 第四百零八阶段：v0.5.17 候选发布边界整理

1. [x] 新增 `docs/plan-release-notes-v0.5.17.md`，整理 v0.5.16 之后围绕 Agent 决策客户端 CI installer、Checklist、安装 dry-run summary schema/sample/validator 的候选发布说明。
2. [x] 新增 `docs/plan-release-v0.5.17.md`，记录当前差异、候选内容、已验证命令、远端 CI run 和正式发布前待办。
3. [x] 明确 v0.5.17 仍不扩语言、不承诺生成算法提升，发布边界聚焦外部 MCP 客户端 CI 接入确定性。

已完成补充：v0.5.17 的候选边界已经文档化。下一步应跑 release doc index、docs links、完整 shell 回归、Go 测试和 diff check；通过后提交并观察 main CI，再进入正式版本准备。

## 第四百零九阶段：v0.5.17 正式版本准备

1. [x] `main.go` MCP implementation version 已更新到 `0.5.17`。
2. [x] `CHANGELOG.md` 已将 Unreleased 内容收敛为 `v0.5.17 - 2026-07-21`，并保留新的空 Unreleased。
3. [x] README、installation、quickstart、first-run/onboarding/verification CI、showcase-onboarding、Agent 决策客户端 CI 模板、安装 summary fixture 和测试期望已同步到 `0.5.17` / `v0.5.17`。
4. [x] `docs/plan-release-v0.5.17.md` 和 `docs/plan-release-notes-v0.5.17.md` 已更新为正式版本准备状态，并记录 `f351e7c` 远端 CI run `29808013697` passed。
5. [x] 版本准备后的完整本地门禁已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.17-release-prep-dist scripts/verify-release-candidate.sh v0.5.17` 输出 `release_candidate_status=passed`，候选二进制 `--version` 输出 `testloop-mcp 0.5.17`。

已完成补充：v0.5.17 正式版本准备文件和本地门禁已完成，但尚未提交版本准备、打 tag、生成 Release assets 或更新 Homebrew。下一步应提交版本准备并等待 main CI；通过后再执行正式发布动作。

## 第四百一十阶段：v0.5.17 客户端 CI installer 正式发布收敛

1. [x] `9e040ba` 远端 CI run `29808559072` passed，覆盖 v0.5.17 正式版本准备。
2. [x] `v0.5.17` tag 已推送，Release Artifacts run `29808977015` passed。
3. [x] `TESTLOOP_MCP_REPO=sleticalboy/testloop-mcp scripts/verify-release-assets.sh v0.5.17` 已验证正式 Release 的 10 个资产完整。
4. [x] 仓库内 `Formula/testloop-mcp.rb` 已更新到 `0.5.17`，并通过 `ruby -c Formula/testloop-mcp.rb` 和 `sh test/release_assets_test.sh`。
5. [x] `sleticalboy/homebrew-tap` 已更新到 `0.5.17` 并推送，tap commit `3fec8ad`。
6. [x] Post-Release Verify run `29809495498` passed，覆盖资产清单和五个平台安装验证。
7. [x] `docs/plan-release-v0.5.17.md` 和 `docs/plan-release-notes-v0.5.17.md` 已更新为正式发布完成状态。
8. [x] 发布后 raw installer smoke 已通过：`scripts/showcase-agent-decision-client-ci-template-install.sh --json` 从 `main` raw URL 下载 installer 后输出 `status=passed`、`helper_ref=v0.5.17`、`fixture_count=8`。

已完成补充：v0.5.17 已完成正式 GitHub Release、五平台资产发布、资产清单校验、仓库内 Formula、Homebrew tap、Post-Release Verify 和发布后 raw installer smoke。下一步应提交发布后 smoke 记录并等待 main CI；CI 通过后回到主线产品价值，继续围绕真实客户端/Agent 消费路径补证据。

## 第四百一十一阶段：Agent 决策客户端消费端 smoke

1. [x] 新增 `scripts/showcase-agent-decision-client-consumer-smoke.sh`，用临时外部 client 串起 workflow 安装、helper dry-run、安装 summary 校验、导出 fixture manifest 校验和 result JSON 消费检查。
2. [x] 新增 `test/agent_decision_client_ci_consumer_smoke_test.sh`，固定文本输出、JSON 输出、helper ref、决策序列、validator 退出码和 artifact 路径存在性。
3. [x] 默认 CI 已加入消费端 smoke，README 和 Agent 决策客户端 CI Checklist 已同步本地验收入口。

已完成补充：外部客户端接入验证不再只停在“installer 能写 workflow”和“helper 能跑 fixture”，而是覆盖到接入方能否稳定消费安装 summary、fixture manifest 和 result JSON。下一步应跑新增 smoke、相关文档测试、CI workflow 自检、完整 shell 回归、Go 测试和 diff check；通过后提交并观察 main CI。

## 第四百一十二阶段：Agent 决策客户端消费端 smoke 摘要契约

1. [x] 新增 `docs/fixtures/agent-decision-client-consumer-smoke-summary.schema.json`，固定消费端 smoke 的 JSON 输出字段。
2. [x] 新增 `docs/fixtures/agent-decision-client-consumer-smoke-summary/passed.json`，提供通过态 golden sample。
3. [x] 新增 `test/agent_decision_client_ci_consumer_smoke_summary_schema_test.sh`，同时校验实时 smoke 输出、schema 和通过态样例。
4. [x] README、fixtures 索引、Agent 决策客户端 CI Checklist 和默认 CI 已同步消费端 smoke summary 契约。

已完成补充：消费端 smoke 现在不只是一个维护者脚本，而是有 schema、golden sample 和 CI 回归固定的客户端可消费合同。下一步应跑新增 schema 测试、docs links、相关 checklist 测试、完整 shell 回归、Go 测试和 diff check；通过后提交并观察 main CI。

## 第四百一十三阶段：Agent 决策客户端消费端 smoke 无依赖校验器

1. [x] 新增 `scripts/validate-agent-decision-client-consumer-smoke-summary.mjs`，默认校验消费端 smoke 通过态 fixture，也可指定任意 summary JSON。
2. [x] 校验器支持文本输出和 `--json` 输出，固定 helper ref、fixture 数量、决策序列、空 failures 和三个 validator 退出码。
3. [x] 新增 `test/agent_decision_client_ci_consumer_smoke_summary_validator_test.sh`，覆盖通过态、实时 smoke 输出、JSON 输出和失败摘要分支。
4. [x] 默认 CI、README、fixtures 索引和 Agent 决策客户端 CI Checklist 已同步 validator 入口。

已完成补充：消费端 smoke 输出现在有 schema、golden sample、无依赖 validator 和 CI 回归四层合同。下一步应跑 validator/schema/smoke 相关测试、docs links、完整 shell 回归、Go 测试和 diff check；通过后提交并观察 main CI。

## 第四百一十四阶段：Agent 决策客户端消费端 smoke 文档同步

1. [x] `docs/client-integration.md` 已补充消费端 smoke、summary schema/sample 和 validator 入口。
2. [x] `docs/mcp-client-contract-tests.md` 已补充接入方 CI 如何把安装后的 artifact 消费路径纳入契约测试。
3. [x] `docs/agent-decision-client-ci-template.md` 已在本地 dry-run 段落补充消费端 smoke 和无依赖校验入口。
4. [x] 三份文档测试已同步必备片段和引用文件存在性。

已完成补充：消费端 smoke 的入口已经从 README/checklist 扩散到客户端集成、契约测试和模板文档，接入方不需要知道维护者内部脚本历史也能找到完整验证路径。下一步应跑三份文档测试、docs links、完整 shell 回归、Go 测试和 diff check；通过后提交并观察 main CI。

## 第四百一十五阶段：v0.5.18 候选发布边界整理

1. [x] 新增 `docs/plan-release-notes-v0.5.18.md`，整理 v0.5.17 之后围绕 Agent 决策客户端消费端 smoke、summary schema/sample、无依赖 validator 和文档同步的候选发布说明。
2. [x] 新增 `docs/plan-release-v0.5.18.md`，记录当前差异、候选内容、已验证命令、远端 CI run 和正式发布前待办。
3. [x] 明确 v0.5.18 仍不扩语言、不承诺生成算法提升，发布边界聚焦外部 MCP 客户端 artifact 消费合同确定性。

已完成补充：v0.5.18 的候选边界已经文档化。下一步应跑 release doc index、docs links、完整 shell 回归、Go 测试和 diff check；通过后提交并观察 main CI，再进入正式版本准备。

## 第四百一十六阶段：v0.5.18 正式版本准备

1. [x] `main.go` MCP implementation version 已更新到 `0.5.18`。
2. [x] `CHANGELOG.md` 已将 Unreleased 内容收敛为 `v0.5.18 - 2026-07-21`，并保留新的空 Unreleased。
3. [x] README、installation、quickstart、first-run/onboarding/verification CI、showcase-onboarding、Agent 决策客户端 CI 模板、消费端 smoke summary fixture 和测试期望已同步到 `0.5.18` / `v0.5.18`。
4. [x] `docs/plan-release-v0.5.18.md` 和 `docs/plan-release-notes-v0.5.18.md` 已更新为正式版本准备状态，并记录 `e2b9208` 远端 CI run `29818006076` passed。
5. [x] 版本准备后的完整本地门禁已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.18-release-prep-dist scripts/verify-release-candidate.sh v0.5.18` 输出 `release_candidate_status=passed`，候选二进制 `--version` 输出 `testloop-mcp 0.5.18`。

已完成补充：v0.5.18 正式版本准备文件和本地门禁已完成，但尚未提交版本准备、打 tag、生成 Release assets 或更新 Homebrew。下一步应提交版本准备并等待 main CI；通过后再执行正式发布动作。

## 第四百一十七阶段：v0.5.18 客户端消费端 smoke 正式发布收敛

1. [x] `c8b2096` 远端 CI run `29818535669` passed，覆盖 v0.5.18 正式版本准备。
2. [x] `v0.5.18` tag 已推送，Release Artifacts run `29818715613` passed。
3. [x] `TESTLOOP_MCP_REPO=sleticalboy/testloop-mcp scripts/verify-release-assets.sh v0.5.18` 已验证正式 Release 的 10 个资产完整。
4. [x] Release Artifacts 并发创建出的重复空 Release 已删除，仅保留带 10 个资产的正式 Release。
5. [x] 仓库内 `Formula/testloop-mcp.rb` 已更新到 `0.5.18`，并通过 `ruby -c Formula/testloop-mcp.rb` 和 `sh test/release_assets_test.sh`。
6. [x] `sleticalboy/homebrew-tap` 已更新到 `0.5.18` 并推送，tap commit `d125310`。
7. [x] Post-Release Verify run `29819216549` passed，覆盖资产清单和五个平台安装验证。
8. [x] 发布后 raw installer smoke 已通过：首次 raw 下载因网络超时失败，重试后输出 `status=passed`、`helper_ref=v0.5.18`、`fixture_count=8`。
9. [x] 发布后消费端 smoke 已通过：`scripts/showcase-agent-decision-client-consumer-smoke.sh --json` 输出 `status=passed`、`helper_ref=v0.5.18`、`fixture_count=8`。

已完成补充：v0.5.18 已完成正式 GitHub Release、五平台资产发布、资产清单校验、重复空 Release 清理、GitHub Release 正文、仓库内 Formula、Homebrew tap、Post-Release Verify、raw installer smoke 和消费端 smoke。下一步应提交发布后记录并等待 main CI；CI 通过后回到主线产品价值，继续围绕真实客户端/Agent 消费路径补证据。

## 第四百一十八阶段：Release Artifacts 并发创建 Release 加固

1. [x] `.github/workflows/release.yml` 已新增 `ensure-release` 前置 job，统一解析 tag 并只创建一次 GitHub Release。
2. [x] 矩阵 `build` job 已改为 `needs: ensure-release`，通过 job output 复用同一个 `TAG_NAME`，只负责构建、校验和上传资产。
3. [x] Release Artifacts workflow 已按 tag 增加 `concurrency`，避免同一 tag 的 tag push 和手动 dispatch 并发运行。
4. [x] 新增 `test/release_workflow_test.sh`，固定 Release 创建只出现一次且不在矩阵 build job 内。
5. [x] release workflow 测试、CI workflow 自检、release asset 测试、文档链接、完整 shell 回归、Go 测试和 diff check 已通过。

已完成补充：v0.5.18 发布时暴露的重复空 Release 根因已在 workflow 结构上修掉，并用测试防止回归。本地验证已通过。下一步应提交并等待 main CI；CI 通过后回到主线产品价值，继续补真实客户端/Agent 消费路径的证据。

## 第四百一十九阶段：消费端 smoke summary 的 Agent 分流

1. [x] 新增 `scripts/render-agent-decision-client-consumer-response.mjs`，读取消费端 smoke summary 并输出稳定的 `agent_next_step`。
2. [x] 通过态分流为 `ready`；validator 失败分流为 `inspect-consumer-smoke-validator`；fixture 数量或决策序列漂移分流为 `inspect-agent-decision-fixtures`；其他结构问题分流为 `inspect-consumer-smoke-summary`。
3. [x] 新增 `test/agent_decision_client_consumer_response_test.sh`，覆盖默认 fixture、JSON 输出、实时 consumer smoke 输出和失败分流。
4. [x] README、客户端集成说明、Agent 决策客户端 CI Checklist 和 fixture 索引已同步 renderer 入口。
5. [x] 新增 response 测试、consumer smoke/validator、相关文档测试、CI workflow 自检、完整 shell 回归、Go 测试和 diff check 已通过。

已完成补充：外部客户端现在不只知道 consumer smoke summary 是否合格，还能把该 summary 直接转换成 Agent 下一步动作。本地验证已通过。下一步应提交并等待 main CI；CI 通过后继续补客户端失败态 artifact 的可读证据。

## 第四百二十阶段：消费端 smoke 失败态 Agent 分流 fixture

1. [x] 新增 `docs/fixtures/agent-decision-client-consumer-smoke-summary/validator-failed.json`，固定 validator 失败时的消费端 summary 形状。
2. [x] 新增 `docs/fixtures/agent-decision-client-consumer-smoke-summary/fixture-drift.json`，固定 fixture 数量或决策序列漂移时的 summary 形状。
3. [x] response renderer 测试已改为读取失败态 fixture，分别断言 `inspect-consumer-smoke-validator` 和 `inspect-agent-decision-fixtures`。
4. [x] summary schema 测试会遍历失败态 sample 的结构字段，validator 测试会确认失败态 sample 不能误判为通过。
5. [x] README、客户端集成说明、Agent 决策客户端 CI Checklist、fixture 索引和文档测试已同步失败态 fixture 入口。
6. [x] consumer response、summary schema/validator、文档测试、docs links、完整 shell 回归、Go 测试和 diff check 已通过。

已完成补充：消费端 smoke 现在有通过态和失败态 golden sample，客户端可以用固定 fixture 测试 Agent 分流，不需要临时构造失败 JSON。本地验证已通过。下一步应提交并等待 main CI；CI 通过后继续补一个客户端 README/模板里的失败分流最小示例。

## 第四百二十一阶段：客户端模板失败分流最小示例

1. [x] `docs/agent-decision-client-ci-template.md` 新增 Agent 分流示例，展示 consumer smoke summary 生成后如何用 renderer 输出 `agent_next_step`。
2. [x] 模板文档新增两个失败态 fixture 的本地演示命令，分别覆盖 `inspect-consumer-smoke-validator` 和 `inspect-agent-decision-fixtures`。
3. [x] 文档测试已固定 renderer 命令、失败态 fixture 路径和三类 `agent_next_step` 输出。
4. [x] Agent 决策客户端 CI 模板文档测试、YAML 测试、安装脚本测试、docs links、完整 shell 回归、Go 测试和 diff check 已通过。

已完成补充：接入方现在可以在模板文档里直接看到“summary -> Agent 下一步”的最小通过态和失败态命令。本地验证已通过。下一步应提交并等待 main CI；CI 通过后继续评估是否把 renderer 接入外部客户端安装 dry-run summary。

## 第四百二十二阶段：消费端 smoke 输出 Agent response artifact

1. [x] `scripts/showcase-agent-decision-client-consumer-smoke.sh` 已自动调用 `render-agent-decision-client-consumer-response.mjs`，并在 summary 中返回 `agent_response_json`。
2. [x] consumer smoke 文本输出新增 `agent_response_json` 和 `agent_next_step`，通过态为 `ready`。
3. [x] consumer smoke summary schema、通过态样例、失败态样例和无依赖 validator 已同步 `agent_response_json` 字段。
4. [x] consumer smoke、summary schema、validator、renderer 和文档测试已覆盖该字段。
5. [x] README、客户端集成说明、CI Checklist、fixture 索引和模板文档已同步 `agent_response_json` 入口。
6. [x] docs links、完整 shell 回归、Go 测试和 diff check 已通过。

已完成补充：接入方运行 consumer smoke 后不必再手动二次转换 summary，能直接拿到可上传、可交给 Agent 的 response JSON。本地验证已通过。下一步应提交并等待 main CI；CI 通过后继续压缩外部客户端接入文档，把推荐 artifact 上传清单补齐。

## 第四百二十三阶段：基础客户端 CI 输出 Agent response artifact

1. [x] 新增 `scripts/render-agent-decision-client-ci-response.mjs`，可把 `showcase-agent-decision-client-ci.sh --json` 输出转成稳定 `agent_next_step`。
2. [x] 安装脚本生成的 GitHub Actions 模板已新增 `Render Agent decision response` step，并上传 `/tmp/testloop-agent-decision-client-response.json`。
3. [x] 模板中的 contract 命令已显式启用 `set -euo pipefail`，避免 pipeline 隐式吞掉失败。
4. [x] 新增 `test/agent_decision_client_ci_response_test.sh`，覆盖通过态、validator 失败、fixture 决策漂移和 summary 缺失分流。
5. [x] installer、模板 YAML、模板 dry-run、checklist、README、客户端集成说明和脚本语法测试已同步 response artifact。
6. [x] 完整 shell 回归、Go 测试和 diff check 已通过。

已完成补充：外部客户端最小 contract CI 现在也会产出可直接给 Agent 的 response JSON，不再只有 consumer smoke 才有下一步动作。本地验证已通过。下一步应提交并等待 main CI；CI 通过后继续收敛发布前 Unreleased 内容，评估是否进入下一个 patch 版本准备。

## 第四百二十四阶段：v0.5.19 候选发布边界整理

1. [x] 新增 `docs/plan-release-notes-v0.5.19.md`，整理 v0.5.18 之后的 Release Artifacts 并发加固、consumer smoke Agent 分流、失败态 fixture、`agent_response_json` 和基础客户端 CI response artifact。
2. [x] 新增 `docs/plan-release-v0.5.19.md`，记录当前差异、候选内容、已验证命令、远端 CI 证据和正式发布前待办。
3. [x] 明确 v0.5.19 仍不扩语言、不承诺生成算法提升，发布边界聚焦 Agent/客户端消费合同和发版稳定性。
4. [x] release doc index、docs links、docs JSON examples、完整 shell 回归、Go 测试和 diff check 已通过。

已完成补充：v0.5.19 的候选边界已经文档化。本地验证已通过。下一步应提交并等待 main CI；CI 通过后进入正式版本准备，或者继续补发布前 smoke 证据。

## 第四百二十五阶段：v0.5.19 正式版本准备

1. [x] `0f8d971` 远端 CI run `29826825652` passed，覆盖 v0.5.19 候选边界整理。
2. [x] `main.go` MCP implementation version 已更新到 `0.5.19`。
3. [x] `CHANGELOG.md` 已将 Unreleased 内容收敛为 `v0.5.19 - 2026-07-21`，并保留新的空 Unreleased。
4. [x] README、installation、quickstart、first-run/onboarding/verification CI、showcase-onboarding、Agent 决策客户端 CI 模板、消费端 smoke summary fixture、脚本默认版本和测试期望已同步到 `0.5.19` / `v0.5.19`。
5. [x] 版本准备后的完整本地门禁已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.19-release-prep-dist scripts/verify-release-candidate.sh v0.5.19` 输出 `release_candidate_status=passed`，候选二进制 `--version` 输出 `testloop-mcp 0.5.19`。
6. [x] `d026283` 远端 CI run `29827369739` passed，覆盖 v0.5.19 正式版本准备。

已完成补充：v0.5.19 正式版本准备文件、本地门禁和远端 CI 已完成，仓库内 Formula 仍保持 `0.5.18`，等待正式 Release assets 生成后再用真实 digest 更新。下一步应打 tag 并等待 Release Artifacts workflow 生成正式资产。

## 第四百二十六阶段：v0.5.19 客户端 Agent response 正式发布收敛

1. [x] `v0.5.19` tag 已推送，指向版本准备提交 `d026283`。
2. [x] Release Artifacts run `29827625494` passed，五个平台 10 个资产已上传。
3. [x] `TESTLOOP_MCP_REPO=sleticalboy/testloop-mcp scripts/verify-release-assets.sh v0.5.19` 已验证正式 Release 资产完整。
4. [x] GitHub Release 正文已更新为正式 v0.5.19 发布说明。
5. [x] 仓库内 `Formula/testloop-mcp.rb` 已更新到 `0.5.19`，并通过 `ruby -c Formula/testloop-mcp.rb` 和 `sh test/release_assets_test.sh`。
6. [x] `sleticalboy/homebrew-tap` 已更新到 `0.5.19` 并推送，tap commit `72123db`。
7. [x] 发布后 raw installer smoke 已通过：`scripts/showcase-agent-decision-client-ci-template-install.sh --json` 输出 `status=passed`、`helper_ref=v0.5.19`、`fixture_count=8`。
8. [x] 发布后基础客户端 CI response smoke 已通过：`scripts/render-agent-decision-client-ci-response.mjs --json <client-summary>` 输出 `status=passed`、`agent_next_step=ready`、`fixture_count=8`。
9. [x] 发布后消费端 smoke 已通过：`scripts/showcase-agent-decision-client-consumer-smoke.sh --json` 输出 `status=passed`、`helper_ref=v0.5.19`、`fixture_count=8`。
10. [x] Post-Release Verify run `29828306451` passed，覆盖资产 manifest、Linux amd64/arm64、macOS arm64 和 Windows amd64/arm64 安装校验。

已完成补充：v0.5.19 已完成正式 GitHub Release、五平台资产发布、资产清单校验、GitHub Release 正文、仓库内 Formula、Homebrew tap、Post-Release Verify 和发布后 smoke。下一步应提交发布完成记录并等待 main CI；CI 通过后回到主线产品价值，继续围绕外部客户端/Agent 消费路径补真实接入证据。

## 第四百二十七阶段：发布后客户端 Agent smoke 汇总证据

1. [x] 新增 `scripts/showcase-agent-decision-client-release-smoke.sh`，把 release tag raw installer、基础客户端 CI response 和 consumer smoke response 合成一份 JSON evidence。
2. [x] `scripts/showcase-agent-decision-client-consumer-smoke.sh` 支持通过 `TESTLOOP_AGENT_DECISION_CI_INSTALLER_URL` 使用 release tag raw installer。
3. [x] 新增 `docs/fixtures/agent-decision-client-release-smoke-summary.schema.json` 和通过态样例，固定 `release_ref`、`helper_refs`、`fixture_count`、决策序列和 Agent next step。
4. [x] 新增 `test/agent_decision_client_release_smoke_test.sh`，真实运行 release smoke 并校验输出合同。
5. [x] `100585c` 首轮 CI run `29830364794` 暴露新增测试未加入默认 CI 清单，已通过 `10d4718` 补齐。
6. [x] `10d4718` 远端 CI run `29830583522` passed，覆盖发布后客户端 Agent smoke 汇总入口。

已完成补充：发布后客户端证据从三条手工命令收敛为一个 Agent 可消费 JSON 入口，并已纳入默认 CI。下一步应回到主线产品价值，继续补真实外部客户端/Agent 消费路径的证据，而不是继续扩发布流程。

## 第四百二十八阶段：v0.5.19 release raw installer 客户端证据

1. [x] `scripts/showcase-agent-decision-client-release-smoke.sh --json` 已使用默认 `v0.5.19` raw installer URL 实跑通过。
2. [x] raw.githubusercontent.com 实跑中出现 `curl: (56) Recv failure: Operation timed out` 和 `curl: (56) Send failure: Broken pipe`，已通过 `curl --retry-all-errors` 增强下载重试。
3. [x] release smoke 最终输出 `status=passed`、`release_ref=v0.5.19`、`helper_refs.install=v0.5.19`、`helper_refs.consumer=v0.5.19`、`fixture_count=8`、`agent_next_steps.client=ready` 和 `agent_next_steps.consumer=ready`。
4. [x] `docs/real-integration-cases.md` 已记录 v0.5.19 客户端 release smoke 实跑证据和它与 Post-Release Verify 的分工。
5. [x] `b4ac070` 远端 CI run `29831583765` passed，覆盖 raw installer release smoke 证据、真实接入案例文档和下载重试增强。

已完成补充：外部客户端证据已经从本地 file URL 回归推进到正式 release tag raw installer 实跑，并明确处理网络抖动。下一步应继续补“客户端如何消费 release smoke summary”的更小接入样例，让第三方 MCP 客户端可以直接复制断言逻辑。

## 第四百二十九阶段：release smoke summary Agent response 消费样例

1. [x] 新增 `scripts/render-agent-decision-client-release-response.mjs`，把 `showcase-agent-decision-client-release-smoke.sh --json` 输出转成稳定 `agent_next_step`。
2. [x] 通过态固定 `release_ref`、`installer_url`、`helper_refs`、`fixture_count`、决策序列和基础/消费端 `agent_next_steps`。
3. [x] 失败态分流覆盖 release installer/helper tag 漂移、基础客户端 response 漂移、consumer response 漂移、fixture 决策漂移和 summary 缺失。
4. [x] 新增 `test/agent_decision_client_release_response_test.sh`，覆盖默认样例、本地 file installer 实跑 summary、JSON 输出和失败分流。
5. [x] CI workflow 已纳入 release response 消费测试，防止新增测试遗漏。
6. [x] 客户端集成说明和 fixture 索引已同步最小消费命令。
7. [x] `8ce1be9` 远端 CI run `29832169823` passed，覆盖 release smoke summary Agent response 消费样例。

已完成补充：发布后 release smoke 现在不仅能产出汇总证据，也有可复制的 Agent 消费器把证据转成下一步动作，并已通过 main CI。下一步应继续回到真实外部项目接入，把这套客户端契约放进一个独立临时项目或示例仓库路径中验证。

## 第四百三十阶段：独立客户端 release response smoke

1. [x] 新增 `scripts/showcase-agent-decision-client-release-response-smoke.sh`，创建临时 Node 客户端项目消费 release smoke summary。
2. [x] 临时客户端只复制 summary、`render-agent-decision-client-release-response.mjs`、`package.json` 和断言脚本，通过自己的 `npm test` 验证。
3. [x] smoke 支持复用已有 summary，也支持实时运行 `showcase-agent-decision-client-release-smoke.sh --json` 生成 summary。
4. [x] 新增 `test/agent_decision_client_release_response_smoke_test.sh`，覆盖 help、fixture summary、实时 file installer summary 和失败 summary。
5. [x] CI workflow 已纳入独立客户端 release response smoke 测试。
6. [x] 客户端集成说明已同步独立临时项目 smoke 命令。
7. [x] `9843e1d` 远端 CI run `29832765011` passed，覆盖独立客户端 release response smoke。

已完成补充：release response 消费路径已经从仓库内 renderer 推进到独立临时客户端项目验证，证明接入方复制 summary/renderer 后可以用自己的 `npm test` 固定 Agent 下一步动作，并已通过 main CI。下一步应继续做真实示例接入包的文档化，把最小外部客户端目录结构沉淀成可复制说明。

## 第四百三十一阶段：release response 独立客户端接入文档

1. [x] 新增 `docs/agent-decision-release-response-client.md`，说明接入方最小目录、`package.json`、CI artifact 和 Agent 分流。
2. [x] 文档固定 `scripts/showcase-agent-decision-client-release-response-smoke.sh --json`、`npm test --silent`、复用 summary 和本地 file installer 的命令。
3. [x] 新增 `test/agent_decision_release_response_client_doc_test.sh`，校验文档关键字段、命令和引用路径。
4. [x] 客户端集成说明、MCP 客户端契约测试说明和 README 已链接到独立客户端接入文档。
5. [x] CI workflow 已纳入 release response 客户端接入文档测试。
6. [x] `d351375` 远端 CI run `29833161729` passed，覆盖 release response 客户端接入文档和新增文档测试。

已完成补充：独立客户端 smoke 的脚本证据已经沉淀成可复制文档，接入方可以按目录结构把 release summary -> response -> Agent next step 固定到自己的 CI，并已通过 main CI。下一步应评估是否需要把这条 release response smoke 加入 README 的展示命令区或 release checklist。

## 第四百三十二阶段：README 展示命令补齐

1. [x] README 的“面向 Agent 的快速演示路径”已新增 `scripts/showcase-agent-decision-client-release-smoke.sh --json`。
2. [x] README 已新增 `scripts/showcase-agent-decision-client-release-response-smoke.sh --json`，说明独立临时 Node 客户端会运行自己的 `npm test --silent`。
3. [x] README 已链接 [Agent 决策 release response 客户端接入](./agent-decision-release-response-client.md)。
4. [x] `test/release_doc_index_test.sh` 已固定 README 中的 release smoke、release response smoke 和 release response 客户端文档入口。
5. [x] v0.5.19 release checklist 暂不加入这条 main 后续增强，避免把 tag 后能力误写成 tag 内发布内容。
6. [x] `2b49d20` 远端 CI run `29833494200` passed，覆盖 README 展示命令和 release doc index 断言。

已完成补充：当前 main 的展示路径已经能从“外部客户端 fixture CI”一路展示到“发布后 release summary 独立客户端消费”，并已通过 main CI。下一步回到产品主线，优先补 release response 的失败态样例，让接入方能固定 `inspect-release-*` 分流。

## 第四百三十三阶段：release response 失败态 fixture

1. [x] 新增 `docs/fixtures/agent-decision-client-release-response.schema.json`，固定 renderer JSON 输出结构。
2. [x] 新增 release response 通过态 fixture：`passed.json`。
3. [x] 新增失败态 fixture：`installer-drift.json`、`client-response-drift.json`、`consumer-response-drift.json` 和 `fixture-drift.json`。
4. [x] 新增 `test/agent_decision_client_release_response_fixtures_test.sh`，用真实 renderer 生成输出并与 fixture 关键字段比对。
5. [x] fixture 索引、客户端集成说明和 release response 客户端接入文档已同步失败态样例。
6. [x] CI workflow 已纳入 release response fixture 测试。
7. [x] `4ad69a2` 远端 CI run `29833986021` passed，覆盖 release response 失败态 fixture。

已完成补充：接入方现在不仅能看到 release response 的 ready 样例，也能固定 installer、基础客户端、consumer 和 fixture 漂移四类失败分流，并已通过 main CI。下一步继续降低接入成本，优先提供 release response 客户端最小包导出脚本，而不是再维护一个完整第三方模板仓库。

## 第四百三十四阶段：release response 客户端最小包导出

1. [x] 新增 `scripts/export-agent-decision-release-response-client.mjs`，导出 release response 客户端最小包。
2. [x] 导出包包含 `testloop-release-smoke-summary.json`、renderer、断言脚本、`package.json`、README、response schema 和 5 个 response fixture。
3. [x] 导出包可直接运行 `npm test --silent`，失败 summary 会生成可交给 Agent 的 `testloop-release-response.json`。
4. [x] 新增 `test/agent_decision_release_response_client_export_test.sh`，覆盖 help、导出、npm test、非空目录失败和失败 summary 分流。
5. [x] README、客户端集成说明和 release response 客户端接入文档已同步导出命令。
6. [x] CI workflow 已纳入 release response 客户端导出测试。
7. [x] `3741007` 远端 CI run `29834500473` passed，覆盖 release response 客户端最小包导出。

已完成补充：接入方现在可以从“文档复制”进一步降到“一条命令导出最小包”，不需要维护完整第三方模板仓库，并已通过 main CI。下一步评估是否需要把导出包接入 release candidate readiness。

## 第四百三十五阶段：release candidate 导出包门禁

1. [x] `scripts/verify-release-candidate.sh` 已新增 `verify agent decision release response client export package` 步骤。
2. [x] release readiness 会运行 `node scripts/export-agent-decision-release-response-client.mjs "$agent_decision_release_response_client_dir"`。
3. [x] release readiness 会进入导出目录执行 `npm test --silent`，确认导出包可直接消费默认 release smoke summary。
4. [x] `test/release_candidate_script_test.sh` 已固定新门禁命令，防止后续发版脚本遗漏。
5. [x] `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.19-export-readiness-dist scripts/verify-release-candidate.sh v0.5.19` 已通过，输出 `release_candidate_status=passed`，并覆盖 `response_fixture_count=5`。
6. [x] `7b39045` 远端 CI run `29835010452` passed，覆盖 release candidate 导出包门禁。

已完成补充：release response 客户端导出包已进入发版前本地 readiness，不再只依赖普通 CI；本地完整 readiness 和 main CI 都已通过。下一步评估是否需要把导出包命令写入通用 release checklist 模板。

## 第四百三十六阶段：Unreleased 变更记录同步

1. [x] `CHANGELOG.md` 的 Unreleased 已记录 release smoke 汇总、release response renderer、独立客户端 smoke 和最小包导出脚本。
2. [x] Unreleased 已记录 release response schema、通过/失败态 fixture，以及 `inspect-release-*` 分流覆盖。
3. [x] Unreleased 已记录 release readiness 显式导出 release response 客户端最小包并运行 `npm test --silent`。
4. [x] 确认 `docs/plan-release.md` 是 v0.4.0 历史记录，不作为通用模板修改；当前发版门禁统一通过 `scripts/verify-release-candidate.sh` 表达。
5. [x] `f61c58c` 远端 CI run `29835352790` passed，覆盖 Unreleased 变更记录同步。

已完成补充：下一版发布收敛不会漏掉本轮 release response 客户端消费链路，并已通过 main CI。下一步应回到更靠近项目价值的真实接入验证：选择一个实际外部客户端/Agent 使用场景，验证导出包是否能直接进入对方 CI，而不是继续在本仓库内扩文档。

## 第四百三十七阶段：release response 外部客户端 CI 形态验证

1. [x] 新增 `scripts/showcase-agent-decision-client-release-response-ci.sh`，创建临时外部客户端仓库并写入 `.github/workflows/testloop-release-response-contract.yml`。
2. [x] 该 showcase 会导出 release response 客户端最小包，并按 workflow 核心命令运行 `npm test --silent`。
3. [x] JSON summary 固定 `repo_dir`、`workflow_path`、`package_dir`、`agent_response_json`、`release_ref`、`fixture_count`、`decisions[]`、`agent_next_step` 和 `npm_exit_code`。
4. [x] 新增 `test/agent_decision_client_release_response_ci_test.sh`，覆盖 help、文本输出、JSON 输出、workflow 文件、artifact 路径和失败 summary 分流。
5. [x] README、客户端集成说明和 release response 客户端接入文档已同步外部仓库 CI 形态验证命令。
6. [x] CI workflow 已纳入 release response 外部客户端 CI 形态测试。
7. [x] `0bba613` 远端 CI run `29837951150` passed，覆盖 release response 外部客户端 CI 形态验证。

已完成补充：release response 导出包现在不仅能单独 `npm test`，也能放进临时外部客户端仓库的 GitHub Actions 结构中验证，并已通过 main CI。下一步把这条外部客户端 CI 形态补进 Unreleased，避免下一版漏记。

## 第四百三十八阶段：外部客户端 CI 形态变更记录同步

1. [x] `CHANGELOG.md` Unreleased 已记录 `scripts/showcase-agent-decision-client-release-response-ci.sh --json`。
2. [x] Unreleased 已说明该 showcase 会创建临时外部客户端仓库、写入 `.github/workflows/testloop-release-response-contract.yml`，并运行导出包 `npm test --silent`。
3. [x] `7513a0f` 远端 CI run `29838372323` passed，覆盖外部客户端 CI 形态变更记录。

已完成补充：下一版发布说明会覆盖 release response 外部客户端 CI 形态验证，并已通过 main CI。下一步应把 showcase 能力升级成可落到真实外部仓库的安装脚本，让 Agent 不只看演示，也能把 workflow 与客户端包写入目标 repo 并产生可审计输出。

## 第四百三十九阶段：release response 真实仓库安装脚本

1. [x] 新增 `scripts/install-agent-decision-release-response-client.sh`，可向真实外部仓库安装 release response 客户端包。
2. [x] 安装脚本会写入 `testloop-release-response-client/` 和 `.github/workflows/testloop-release-response-contract.yml`，并在目标包目录运行 `npm test --silent`。
3. [x] 安装脚本支持 `--summary-json`、`--package-dir`、`--workflow-path`、`--force`、`--dry-run` 和 `--json`。
4. [x] 默认不覆盖已有 workflow 或包目录，`--force` 才会覆盖目标 release response 客户端包。
5. [x] 新增 `test/install_agent_decision_release_response_client_test.sh`，覆盖 help、dry-run、写入、JSON summary、force、坏路径和坏 summary 失败分流。
6. [x] README、客户端集成说明和 release response 客户端接入文档已同步真实仓库安装命令。
7. [x] CI workflow 已纳入 release response 真实仓库安装脚本测试。
8. [x] `3033622` 远端 CI run `29839378286` passed，覆盖 release response 真实仓库安装脚本。

已完成补充：release response 接入路径已经从“导出最小包”和“临时 showcase”推进到“可直接写入真实外部仓库并本地验证”，并已通过 main CI。下一步应给安装脚本的 JSON 输出补 schema 与 validator，让接入方可以把安装 summary 纳入自己的机器校验，而不是只依赖日志文本。

## 第四百四十阶段：release response 安装 summary 契约

1. [x] `scripts/install-agent-decision-release-response-client.sh --json --dry-run` 已补齐完整 summary 字段，和写入态保持同一结构。
2. [x] 新增 `docs/fixtures/agent-decision-release-response-client-install-summary.schema.json`。
3. [x] 新增通过态 fixture：`docs/fixtures/agent-decision-release-response-client-install-summary/passed.json`。
4. [x] 新增 `scripts/validate-agent-decision-release-response-client-install-summary.mjs`，固定 `status=written`、`release_ref=v0.5.19`、`fixture_count=8`、`agent_next_step=ready`、`npm_exit_code=0` 和决策序列。
5. [x] 新增 schema 测试和 validator 测试，覆盖真实 installer JSON、fixture 样例、文本输出、JSON 输出和失败分支。
6. [x] README、fixtures 索引、客户端集成说明和 release response 客户端接入文档已同步 schema/validator 入口。
7. [x] CI workflow 已纳入 release response 安装 summary schema 与 validator 测试。
8. [x] `4432a50` 远端 CI run `29840277414` passed，覆盖 release response 安装 summary schema、fixture 和 validator。

已完成补充：真实外部仓库安装不再只有日志和文件副作用，而是有稳定 JSON 契约、通过态 fixture 和 validator，并已通过 main CI。下一步应把安装 summary validator 接入 release readiness，让发版前门禁同时验证导出包和真实仓库安装摘要契约。

## 第四百四十一阶段：release readiness 覆盖安装摘要

1. [x] `scripts/verify-release-candidate.sh` 已新增 `verify agent decision release response client install summary` 步骤。
2. [x] release readiness 会创建临时外部仓库目录，运行 `scripts/install-agent-decision-release-response-client.sh --json`。
3. [x] release readiness 会把安装 summary 写入临时 JSON，并运行 `node scripts/validate-agent-decision-release-response-client-install-summary.mjs`。
4. [x] `test/release_candidate_script_test.sh` 已固定新 readiness step、installer 命令和 validator 命令。
5. [x] `CHANGELOG.md` Unreleased 已说明 release readiness 会运行真实仓库 installer 并校验安装 summary。
6. [x] `c899bee` 远端 CI run `29840878872` passed，覆盖 release readiness 安装 summary 门禁。

已完成补充：发版前门禁现在会同时覆盖 release response 导出包和真实仓库安装 summary，并已通过 main CI。下一步应把 release response 真实安装路径整理成接入 checklist，让外部客户端能按“生成 release smoke summary -> 安装包与 workflow -> 校验安装 summary -> 提交 CI”执行。

## 第四百四十二阶段：release response 接入 checklist

1. [x] 新增 `docs/agent-decision-release-response-checklist.md`。
2. [x] checklist 已覆盖 release smoke summary 生成、真实仓库安装、安装 summary validator、本地 `npm test --silent` 复验、客户端 CI artifact 和 Agent 分流。
3. [x] 新增 `test/agent_decision_release_response_checklist_doc_test.sh`，固定 checklist 命令、JSON 示例、引用路径和分流动作。
4. [x] CI workflow 已纳入 release response checklist 文档测试。
5. [x] README 和 release response 客户端接入文档已链接 checklist。
6. [x] `CHANGELOG.md` Unreleased 已记录 release response 接入 checklist。
7. [x] `d8b3273` 远端 CI run `29841761363` passed，覆盖 release response checklist 文档入口和测试。

已完成补充：外部客户端现在有一份按步骤执行的 release response checklist，不需要从长文档里自行拼接命令，并已通过 main CI。下一步应把 checklist 中的关键命令做成可执行测试，确认安装、summary validator 和导出包 `npm test --silent` 真的能按文档顺序跑通。

## 第四百四十三阶段：release response checklist 命令回归

1. [x] 新增 `test/agent_decision_release_response_checklist_commands_test.sh`。
2. [x] 测试会解析 `docs/agent-decision-release-response-checklist.md` 的 bash block 数量和顺序。
3. [x] 测试会用固定 release smoke summary fixture 替代真实发布 smoke 输出，实际运行 installer 命令。
4. [x] 测试会校验安装 summary、运行 `validate-agent-decision-release-response-client-install-summary.mjs`，并执行导出包内 `npm test --silent`。
5. [x] 测试会复验 workflow 中的 `cd testloop-release-response-client && npm test --silent` 命令可在临时客户端仓库跑通。
6. [x] CI workflow 已纳入 release response checklist 命令回归。
7. [x] `3718ed9` 远端 CI run `29842445534` passed，覆盖 release response checklist 命令回归。

已完成补充：release response checklist 不再只做字符串存在性检查，关键命令会按文档顺序真实执行，并已通过 main CI。下一步应把这条 release response 安装路径沉淀进真实接入案例记录，展示接入方仓库能拿到哪些文件、summary 和 Agent 分流证据。

## 第四百四十四阶段：release response 真实安装案例记录

1. [x] 使用临时外部仓库目录运行 `scripts/install-agent-decision-release-response-client.sh --json`。
2. [x] 使用 `node scripts/validate-agent-decision-release-response-client-install-summary.mjs` 校验安装 summary。
3. [x] 实跑结果确认 `status=written`、`release_ref=v0.5.19`、`fixture_count=8`、`agent_next_step=ready` 和 `npm_exit_code=0`。
4. [x] 实跑结果确认 workflow、导出包目录和 `testloop-release-response.json` 均已生成。
5. [x] `docs/real-integration-cases.md` 已新增 `v0.5.19 release response 真实安装接入记录`。
6. [x] `test/real_integration_cases_doc_test.sh` 已固定真实安装案例的关键字段和制品路径。

已完成补充：release response 安装路径已有真实接入案例记录，展示了外部仓库会得到的 workflow、导出包、summary 和 Agent response。下一步应跑真实接入案例文档测试、release response checklist 命令测试、文档链接、Go 测试和 diff check；通过后提交并等待 main CI。

## 第四百四十五阶段：v0.5.20 候选发布边界整理

1. [x] 完整 release readiness 已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.19-goal-readiness-dist scripts/verify-release-candidate.sh v0.5.19` 输出 `release_candidate_status=passed`。
2. [x] readiness 覆盖 release response 导出包：`response_fixture_count=5`。
3. [x] readiness 覆盖真实仓库安装 summary：`agent_decision_release_response_client_install_summary_status=passed release_ref=v0.5.19`。
4. [x] readiness 覆盖候选二进制版本与打包 dry-run：`testloop-mcp 0.5.19`，`testloop-mcp_v0.5.19_darwin_arm64.tar.gz: OK`。
5. [x] 新增 `docs/plan-release-v0.5.20.md`，整理下一版候选发布检查清单。
6. [x] 新增 `docs/plan-release-notes-v0.5.20.md`，整理下一版候选发布说明草案。
7. [x] 明确当前仍是候选状态：不改 `main.go` 版本、不收敛 `CHANGELOG.md`、不打 tag、不创建 GitHub Release。
8. [x] `a704d4e` 远端 CI run `29844159254` passed，覆盖 v0.5.20 候选发布边界整理。

已完成补充：v0.5.20 候选边界已经整理出来，当前具备可发布候选状态，并已通过 main CI。正式发布前还需单独进入版本准备；当前 goal 不打 tag、不创建 GitHub Release。

## 第四百四十六阶段：v0.5.20 正式版本准备

1. [x] `main.go` MCP implementation version 已更新到 `0.5.20`。
2. [x] `CHANGELOG.md` 已将 Unreleased 内容收敛为 `v0.5.20 - 2026-07-21`，并保留新的空 Unreleased。
3. [x] README、installation、quickstart、first-run/onboarding/verification CI、Agent 决策客户端 CI 模板、release smoke、release response fixture、脚本默认版本和测试期望已同步到 `0.5.20` / `v0.5.20`。
4. [x] `go test ./...` 已通过。
5. [x] 版本准备后的完整本地门禁已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.20-release-prep-dist scripts/verify-release-candidate.sh v0.5.20` 输出 `release_candidate_status=passed`，候选二进制 `--version` 输出 `testloop-mcp 0.5.20`。
6. [x] readiness 覆盖真实仓库安装 summary：`agent_decision_release_response_client_install_summary_status=passed release_ref=v0.5.20`。
7. [x] readiness 覆盖 darwin arm64 打包 dry-run 和 sha256 校验：`testloop-mcp_v0.5.20_darwin_arm64.tar.gz: OK`。
8. [x] `353d255` 远端 CI run `29846178265` passed，覆盖 v0.5.20 正式版本准备。

已完成补充：v0.5.20 正式版本准备已完成本地门禁和 main CI，当前仍不打 tag、不创建 GitHub Release、不更新 Homebrew tap。下一步应进入正式 tag 与 Release assets 阶段：推送 `v0.5.20` tag，等待 Release Artifacts workflow 生成五平台资产，再验证资产、更新 Release 正文和 Homebrew tap。

## 第四百四十七阶段：v0.5.20 release response 正式发布

1. [x] `v0.5.20` tag 已推送，指向 `44c2344`。
2. [x] Release Artifacts workflow run `29847487312` passed，覆盖 `darwin_arm64`、`linux_amd64`、`linux_arm64`、`windows_amd64` 和 `windows_arm64`。
3. [x] `TESTLOOP_MCP_REPO=sleticalboy/testloop-mcp scripts/verify-release-assets.sh v0.5.20` 已验证正式 Release 的 10 个 assets。
4. [x] GitHub Release 正文已更新为正式 v0.5.20 发布说明。
5. [x] 仓库内 `Formula/testloop-mcp.rb` 已更新到 `0.5.20`，并通过 `ruby -c Formula/testloop-mcp.rb` 和 `sh test/release_assets_test.sh`。
6. [x] `sleticalboy/homebrew-tap` 已更新到 `0.5.20` 并推送，tap commit `bee0521`。
7. [x] 本机 Homebrew tap 已 fast-forward 到 `bee0521`，`brew fetch --force --formula sleticalboy/tap/testloop-mcp` 成功，`brew audit --formula --strict sleticalboy/tap/testloop-mcp` 通过。
8. [x] 发布后 release smoke 已通过：`status=passed`、`release_ref=v0.5.20`、`helper_refs.install=v0.5.20`、`helper_refs.consumer=v0.5.20`、`fixture_count=8`、`agent_next_steps.client=ready` 和 `agent_next_steps.consumer=ready`。
9. [x] 发布后 release response smoke 已通过：`status=passed`、`release_ref=v0.5.20`、`fixture_count=8`、`agent_next_step=ready` 和 `npm_exit_code=0`。
10. [x] Post-Release Verify run `29848148743` passed，覆盖 asset manifest、Linux amd64/arm64、macOS arm64、Windows amd64/arm64 安装校验。
11. [x] `c22cd07` 远端 CI run `29849305668` passed，覆盖 v0.5.20 正式发布记录。

已完成补充：v0.5.20 已完成正式 GitHub Release、五平台资产发布、资产清单校验、GitHub Release 正文、仓库内 Formula、Homebrew tap、Post-Release Verify、发布后 smoke 和发布记录 main CI。下一步回到产品主线，优先做真实外部客户端/Agent 接入样板，把 release response contract 从“工具可用”推进到“接入方可直接照抄”。

## 第四百四十八阶段：真实外部客户端接入样板

1. [x] 选定 `examples/release-response-adopter/` 作为最小外部客户端样板目录，不污染主仓库发布资产。
2. [x] 新增 `scripts/showcase-release-response-adopter.sh --json`，用 `v0.5.20` release response installer 生成临时外部仓库的 workflow、客户端包、安装 summary 和 Agent response。
3. [x] 固定接入方 README，说明如何运行 `npm test --silent`、如何读取 `testloop-release-response.json` 和如何按 `agent_next_step` 分流。
4. [x] 新增 `examples/release-response-adopter/scripts/read-testloop-release-response.mjs`，给接入方提供不解析日志、只消费 release response JSON 的 helper。
5. [x] 新增 `test/release_response_adopter_example_test.sh`，覆盖样板 help、fixture 消费、真实 installer dry-run、workflow 输出和 helper 分流。
6. [x] README、客户端集成说明、release response 客户端接入文档和 checklist 已同步样板入口。
7. [x] `docs/real-integration-cases.md` 已新增 `v0.5.20 release response 接入方样板记录`，记录 showcase 实跑结果和接入方 helper 输出。
8. [x] `0fbf850` 远端 CI run `29884065782` passed，覆盖 release response 接入样板、CI workflow 纳入、server build、testgen build 和 Docker image build。

当前目标：让真实接入方不需要理解 testloop-mcp 内部实现，也能按一页说明接入 release response contract，并把结果交给 AI Agent 稳定消费。

已完成补充：release response contract 现在有可复制的外部客户端样板、接入方 JSON 消费 helper、临时外部仓库 showcase、真实案例记录和文档回归测试，并已通过 main CI。下一步应给 `scripts/showcase-release-response-adopter.sh --json` 补稳定 schema、通过态 fixture 和无依赖 validator，让接入方也能把样板 summary 纳入自己的机器校验。

## 第四百四十九阶段：release response 接入样板 summary 契约

1. [x] 新增 `docs/fixtures/release-response-adopter-summary.schema.json`。
2. [x] 新增通过态 fixture，固定 `status=passed`、`release_ref=v0.5.20`、`fixture_count=8`、`agent_next_step=ready`、`should_accept=true` 和 `npm_exit_code=0`。
3. [x] 新增 `scripts/validate-release-response-adopter-summary.mjs`，校验 showcase summary 的关键路径、状态和 Agent 分流字段。
4. [x] README、客户端集成说明、release response 客户端文档和接入样板 README 已同步 schema/validator 入口。
5. [x] CI workflow 已纳入 summary schema/validator 测试。
6. [x] 本地已通过 `sh test/release_response_adopter_summary_schema_test.sh`、`sh test/release_response_adopter_summary_validator_test.sh`、`sh test/release_response_adopter_example_test.sh`、`sh test/client_integration_doc_test.sh`、`sh test/docs_links_test.sh` 和 `go test ./...`。

已完成补充：接入样板 showcase 的 JSON 输出已经从“可读 summary”升级为有 schema、passed fixture 和 validator 的机器契约。下一步应提交并观察 main CI；通过后可继续把该 validator 接入 release readiness，确保发版前也覆盖接入样板 summary。

## 第四百五十阶段：release readiness 覆盖接入样板 summary

1. [x] `scripts/verify-release-candidate.sh` 已新增 `verify release response adopter summary` 步骤。
2. [x] readiness 会运行 `scripts/showcase-release-response-adopter.sh --json`，并把 summary 写入临时 JSON。
3. [x] readiness 会运行 `node scripts/validate-release-response-adopter-summary.mjs "$release_response_adopter_summary"`，固定 `status=passed`、`release_ref=v0.5.20`、`fixture_count=8`、`agent_next_step=ready` 和 `should_accept=true`。
4. [x] `test/release_candidate_script_test.sh` 已固定新增 readiness step、showcase 命令和 validator 命令。
5. [x] 完整本地 readiness 已通过：`TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.20-adopter-readiness-dist scripts/verify-release-candidate.sh v0.5.20` 输出 `release_candidate_status=passed`，并包含 `release_response_adopter_summary_status=passed release_ref=v0.5.20`。

已完成补充：发版前门禁现在同时覆盖 release response 导出包、真实仓库安装 summary 和接入样板 summary。下一步应提交并观察 main CI；通过后继续收敛下一阶段，优先把接入样板 summary validator 链接进 release checklist 或发布准备文档，避免维护者只知道普通 showcase 命令。

## 第四百五十一阶段：接入样板 summary validator 文档收口

1. [x] `18e202b` 远端 CI run `29884470170` passed，覆盖 release response 接入样板 summary schema、fixture、validator 和 CI 纳入。
2. [x] `2429319` 远端 CI run `29884700633` passed，覆盖 release readiness 接入样板 summary 门禁。
3. [x] `docs/agent-decision-release-response-checklist.md` 已补充 `node scripts/validate-release-response-adopter-summary.mjs /path/to/release-response-adopter-summary.json`。
4. [x] checklist 已链接 `release-response-adopter-summary.schema.json` 和通过态 fixture。
5. [x] `9fbac6e` 远端 CI run `29884843719` passed，覆盖 checklist validator 入口、文档测试、server build、testgen build 和 Docker image build。

当前目标：让 release response 接入样板从 showcase、summary、validator 到 release readiness 的链路全部可发现、可执行、可机器校验。

已完成补充：接入样板 summary validator 已进入 README、客户端集成说明、release response 客户端文档、接入方样板 README、fixtures 索引、release checklist 和 release readiness，当前主线 CI 已通过。下一步收益较高的是补接入样板 summary 的失败态 fixture，让接入方可以固定 `failed` summary 的 Agent 处理方式，而不是只验证 happy path。

## 第四百五十二阶段：接入样板 summary 失败态 fixture

1. [x] 新增 `docs/fixtures/release-response-adopter-summary/invalid-response.json` 失败态 fixture。
2. [x] `scripts/validate-release-response-adopter-summary.mjs --json` 失败输出会合并原始 summary `failures[]` 和 validator failures，接入方可直接读取。
3. [x] 已扩展 schema/validator 测试，覆盖失败态 fixture 和动态坏 summary。
4. [x] README、客户端集成说明、fixtures 索引和接入样板 README 已同步失败态用法。
5. [x] CI workflow 已通过既有 `release_response_adopter_summary_*` 测试覆盖新增失败态。
6. [x] `53443e6` 远端 CI run `29888810717` passed，覆盖接入样板 summary 失败态 fixture、validator、文档测试、server build、testgen build 和 Docker image build。

已完成补充：接入样板 summary 现在同时有 passed/failed 两类 fixture；失败态固定 `agent_next_step=inspect-release-smoke-summary`、`should_accept=false` 和 `failures[]`，validator `--json` 会把这些失败原因带给 Agent，并已通过 main CI。

## 第四百五十三阶段：接入样板失败态 Agent 消费示例

1. [x] 新增 `examples/release-response-adopter/scripts/read-testloop-release-response-summary.mjs`，展示接入方如何消费 adopter summary 的 failed 输出。
2. [x] `examples/release-response-adopter/README.md` 已增加失败态本地命令，明确 `should_accept=false` 时不要继续发布。
3. [x] `test/release_response_adopter_example_test.sh` 已固定 failed summary -> `testloop_release_response_summary_next_step=inspect-release-smoke-summary`，不依赖 validator 文本。
4. [x] README、客户端集成说明和 fixtures 索引已同步该消费路径。
5. [x] `86f779c` 远端 CI run `29888932247` passed，覆盖上一阶段 roadmap 记录。
6. [x] `3f96248` 远端 CI run `29889098732` passed，覆盖 summary 消费 helper、接入样板测试、server build、testgen build 和 Docker image build。

当前目标：让接入方既能消费 release response JSON，也能消费 adopter summary JSON；通过态继续发布，失败态按 `agent_next_step` 和 `failures[]` 停止并分流。

已完成补充：外部接入方现在可以复制两个 helper：一个消费 `testloop-release-response.json`，一个消费 adopter summary JSON；summary helper 在 `should_accept=false` 时返回非 0，防止发布流程误继续，并已通过 main CI。下一步收益较高的是把两个 helper 的输出契约做成独立 fixture/文档索引，减少接入方从 README 中手工复制字段名。

## 第四百五十四阶段：接入样板 helper 输出契约索引

1. [x] 已在接入样板 README 整理 helper 输出字段说明，覆盖 `testloop_release_response_*` 和 `testloop_release_response_summary_*`。
2. [x] 接入样板 README 已增加两张字段表，说明 status、next_step、should_accept、failures 的消费动作。
3. [x] `test/release_response_adopter_example_test.sh` 已固定两组 helper 输出字段，避免后续改名。
4. [x] fixtures 索引已补充 helper 输出字段表入口。
5. [x] `906f8b4` 远端 CI run `29889331769` passed，覆盖 helper 输出字段文档、接入样板测试、server build、testgen build 和 Docker image build。

已完成补充：接入样板 helper 输出字段已经从 README 示例沉淀为字段表和回归测试，`testloop_release_response_*` 与 `testloop_release_response_summary_*` 两组输出不再是隐式约定。下一步应回到 release response 主线，补一个最小“接入方 CI artifact 包”示例，展示外部仓库应上传哪些 JSON/日志给 Agent。

## 第四百五十五阶段：接入方 CI artifact 包示例

1. [x] 在 `examples/release-response-adopter/README.md` 增加 CI artifact 清单，覆盖 summary、response、consumer JSON 和 helper 输出。
2. [x] README 已明确建议 artifact 名称 `testloop-release-response-adopter-artifacts`，以及最小可上传文件。
3. [x] `test/release_response_adopter_example_test.sh` 已固定 artifact 清单字段名和路径，避免接入方缺失关键证据。
4. [x] `docs/client-integration.md` 和 release response checklist 已同步 artifact 包说明。
5. [x] `6c0a7ec` 远端 CI run `29889567628` passed，覆盖 CI artifact 清单、文档测试、server build、testgen build 和 Docker image build。

已完成补充：外部接入方现在知道应该上传哪一组 release response artifact，Agent 离线排查时不只依赖一份 summary。下一步应把 artifact 清单变成一个可执行的打包 helper，减少接入方手写 `mkdir/cp` 的机会。

## 第四百五十六阶段：接入方 CI artifact 打包 helper

1. [x] `scripts/showcase-release-response-adopter.sh --json` 已扩展为默认生成 `testloop-release-response-adopter-artifacts/` 目录，输出目录可用 `TESTLOOP_RELEASE_RESPONSE_ADOPTER_ARTIFACT_DIR` 覆盖。
2. [x] artifact 目录已包含 adopter summary、install summary、release smoke 输入、release response JSON、consumer JSON 和 summary consumer JSON。
3. [x] `release-response-adopter-summary` schema、passed/failed fixture 和 validator 已新增 `artifact_dir`、`summary_consumer_json` 字段。
4. [x] `test/release_response_adopter_example_test.sh` 已固定 artifact 目录文件清单和 summary consumer JSON。
5. [x] 接入样板 README、README、client integration、release response 客户端文档、checklist 和 changelog 已同步打包 helper 行为。
6. [x] 本地已通过 `sh test/release_response_adopter_example_test.sh`、`sh test/release_response_adopter_summary_schema_test.sh`、`sh test/release_response_adopter_summary_validator_test.sh`、`sh test/client_integration_doc_test.sh`、`sh test/agent_decision_release_response_checklist_doc_test.sh`、`sh test/docs_links_test.sh`、`sh test/readme_ci_snippet_test.sh`、`sh test/release_candidate_script_test.sh`、`go test ./...`、`git diff --check` 和完整 `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.20-adopter-artifact-dist scripts/verify-release-candidate.sh v0.5.20`。
7. [x] `729efbf` 远端 CI run `29890037886` passed，覆盖接入样板 artifact 打包输出、server build、testgen build 和 Docker image build。

已完成补充：接入方样板不再只告诉外部仓库“应该上传哪些文件”，而是由 showcase 直接生成可上传的 artifact 目录，并把目录路径写回 summary，Agent 可以离线追踪 installer、renderer 和两个消费 helper 的证据。下一步继续收敛外部接入样板的失败排查体验，优先补一份“接入方 artifact 下载后如何离线自检”的 verifier 或 demo。

## 第四百五十七阶段：接入方 artifact 离线自检

1. [x] 新增 `scripts/verify-release-response-adopter-artifact.mjs`，校验下载后的 `testloop-release-response-adopter-artifacts/` 目录。
2. [x] verifier 已检查 6 个必备文件、JSON 可解析、`release_ref=v0.5.20`、`fixture_count=8`、`agent_next_step=ready`、`should_accept=true` 和空 `failures[]`。
3. [x] verifier 不依赖原始 CI 绝对路径，只检查 summary 中证据路径的文件名/相对后缀，适配 artifact 下载到新目录后的场景。
4. [x] `test/release_response_adopter_artifact_verify_test.sh` 已覆盖通过态、下载后换目录、缺文件、坏 consumer JSON 和 usage。
5. [x] GitHub CI 显式测试列表和 release readiness 已纳入 verifier。
6. [x] README、接入样板 README、client integration、release response 客户端文档、checklist、fixtures 索引和 changelog 已同步离线自检入口。
7. [x] 本地已通过 `sh test/release_response_adopter_artifact_verify_test.sh`、`sh test/release_response_adopter_example_test.sh`、`sh test/release_candidate_script_test.sh`、`sh test/client_integration_doc_test.sh`、`sh test/agent_decision_release_response_checklist_doc_test.sh`、`sh test/agent_decision_release_response_client_doc_test.sh`、`sh test/fixtures_index_test.sh`、`sh test/ci_workflow_test.sh`、`go test ./...`、`git diff --check` 和完整 `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.20-adopter-artifact-verify-dist scripts/verify-release-candidate.sh v0.5.20`。readiness 输出 `release_response_adopter_artifact_status=passed` 和 `release_candidate_status=passed`。
8. [x] `290abfd` 远端 CI run `29890637849` passed，覆盖接入样板 artifact 离线自检、server build、testgen build 和 Docker image build。

当前目标：让接入方从“生成并上传 artifact”继续推进到“下载 artifact 后能一条命令判断证据包是否完整、自洽、可交给 Agent”。

## 第四百五十八阶段：接入方 artifact 自检失败分流

1. [x] `scripts/verify-release-response-adopter-artifact.mjs` 失败时不再沿用 adopter summary 中可能存在的 `agent_next_step=ready`。
2. [x] verifier 失败输出固定 `agent_next_step=inspect-release-response-adopter-artifact` 和 `should_accept=false`。
3. [x] `test/release_response_adopter_artifact_verify_test.sh` 已覆盖缺文件文本输出和坏 consumer JSON 输出的失败分流字段。
4. [x] 接入样板 README、client integration、release response checklist 和 changelog 已同步失败分流语义。
5. [x] 本地已通过 `sh test/release_response_adopter_artifact_verify_test.sh`、`sh test/release_response_adopter_example_test.sh`、`sh test/client_integration_doc_test.sh`、`sh test/agent_decision_release_response_checklist_doc_test.sh`、`go test ./...`、`git diff --check` 和完整 `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.20-adopter-artifact-failure-routing-dist scripts/verify-release-candidate.sh v0.5.20`。readiness 输出 `release_response_adopter_artifact_status=passed` 和 `release_candidate_status=passed`。
6. [x] `e15f09c` 远端 CI run `29891017150` passed，覆盖接入样板 artifact 自检失败分流、server build、testgen build 和 Docker image build。

当前目标：让 Agent 在 artifact 包本身不完整或不自洽时明确停在 artifact 排查分支，而不是误读内部 summary 的通过态。

## 第四百五十九阶段：接入方 artifact 自检 JSON 契约

1. [x] `scripts/verify-release-response-adopter-artifact.mjs --json` 输出新增 `schema_version=1`。
2. [x] 新增 `docs/fixtures/release-response-adopter-artifact-verification.schema.json`。
3. [x] 新增通过态 fixture `docs/fixtures/release-response-adopter-artifact-verification/passed.json`。
4. [x] 新增失败态 fixture `docs/fixtures/release-response-adopter-artifact-verification/missing-summary-consumer.json`，固定 `agent_next_step=inspect-release-response-adopter-artifact` 和 `should_accept=false`。
5. [x] 新增 `scripts/validate-release-response-adopter-artifact-verification.mjs`，校验 verifier JSON 的通过态字段、6 个文件清单、`ready` 分流和空 `failures[]`。
6. [x] release readiness 已改为生成 verifier `--json`，再运行 artifact verification validator。
7. [x] GitHub CI 显式测试列表已纳入 schema/validator 测试。
8. [x] README、接入样板 README、client integration、release response checklist、fixtures 索引和 changelog 已同步 artifact verification schema/validator 入口。
9. [x] 本地已通过 `sh test/release_response_adopter_artifact_verification_schema_test.sh`、`sh test/release_response_adopter_artifact_verification_validator_test.sh`、`sh test/release_response_adopter_artifact_verify_test.sh`、`sh test/release_candidate_script_test.sh`、`sh test/release_response_adopter_example_test.sh`、`sh test/client_integration_doc_test.sh`、`sh test/agent_decision_release_response_checklist_doc_test.sh`、`sh test/fixtures_index_test.sh`、`sh test/readme_ci_snippet_test.sh`、`sh test/ci_workflow_test.sh`、`sh test/docs_links_test.sh`、`go test ./...`、`git diff --check` 和完整 `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.20-adopter-artifact-verification-contract-dist scripts/verify-release-candidate.sh v0.5.20`。readiness 输出 `release_response_adopter_artifact_verification_status=passed` 和 `release_candidate_status=passed`。
10. [x] `161a7ba` 远端 CI run `29893032205` passed，覆盖接入样板 artifact 自检 JSON 契约、server build、testgen build 和 Docker image build。

当前目标：让接入方不仅能离线自检 artifact 目录，还能把自检结果作为稳定 JSON 契约放进自己的客户端单元测试。

## 第四百六十阶段：artifact verification 文档索引收口

1. [x] `docs/agent-decision-release-response-client.md` 已补充 artifact verifier `--json` 和 `validate-release-response-adopter-artifact-verification.mjs` 入口。
2. [x] release response 客户端文档已链接 artifact verification schema、passed fixture 和 missing-summary-consumer 失败态 fixture。
3. [x] `test/agent_decision_release_response_client_doc_test.sh` 已固定上述命令和路径。
4. [x] `test/release_doc_index_test.sh` 已把 release response adopter showcase、summary validator、artifact verifier 和 artifact verification validator 纳入 README 命令索引。
5. [x] 本地已通过 `sh test/agent_decision_release_response_client_doc_test.sh`、`sh test/release_doc_index_test.sh`、`sh test/docs_links_test.sh`、`sh test/client_integration_doc_test.sh`、`sh test/agent_decision_release_response_checklist_doc_test.sh`、`sh test/readme_ci_snippet_test.sh`、`sh test/fixtures_index_test.sh`、`sh test/release_response_adopter_artifact_verification_validator_test.sh`、`sh test/release_response_adopter_artifact_verification_schema_test.sh`、`go test ./...` 和 `git diff --check`。
6. [x] `bfba45a` 远端 CI run `29893380743` passed，覆盖 artifact verification 文档索引、server build、testgen build 和 Docker image build。
7. [x] `e98a70a` 远端 CI run `29893515188` passed，覆盖 artifact 契约索引 CI 记录、server build、testgen build 和 Docker image build。

当前目标：让 release response 接入样板的新 JSON 契约可以从 README、release checklist、客户端集成说明、fixtures 索引和 release response 客户端文档五条路径发现。

## 第四百六十一阶段：artifact verification 客户端消费 demo

1. [x] 新增 `examples/release-response-adopter-artifact-demo`，读取 artifact verification JSON 并输出客户端决策。
2. [x] 通过态 fixture 映射为 `client_decision=accept`。
3. [x] 缺文件失败态 fixture 映射为 `client_decision=inspect-artifact`，并输出 `missing_files` 和 `failures`。
4. [x] 新增 `test/release_response_adopter_artifact_demo_test.sh` 覆盖通过态、失败态和 usage。
5. [x] GitHub CI 显式测试列表已纳入 demo 测试。
6. [x] README、client integration、fixtures 索引和 changelog 已同步 demo 入口。
7. [x] 本地已通过 `sh test/release_response_adopter_artifact_demo_test.sh`、`sh test/release_doc_index_test.sh`、`sh test/client_integration_doc_test.sh`、`sh test/fixtures_index_test.sh`、`sh test/readme_ci_snippet_test.sh`、`sh test/docs_links_test.sh`、`sh test/release_candidate_script_test.sh`、`sh test/ci_workflow_test.sh`、`go test ./...`、`git diff --check` 和完整 `TESTLOOP_RELEASE_CANDIDATE_DIST_DIR=/tmp/testloop-v0.5.20-adopter-artifact-demo-dist scripts/verify-release-candidate.sh v0.5.20`。

当前目标：让接入方不仅能校验 artifact verification JSON，还能看到最小客户端如何把结果映射成 Agent/CI 决策。

## 近期完成标准

第一个有价值的里程碑是：

- [x] `run_tests` 能从 `go test -json` 返回可靠的结构化 Go 失败信息
- [x] 旧版 Go 文本解析仍然可用
- [x] parser 测试覆盖 JSON 和文本输出
- [x] 已知 demo 生成测试即使失败，也能被准确报告
