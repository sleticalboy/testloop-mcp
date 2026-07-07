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

已完成补充：新增 Go/Python/Jest/Rust/Java 的 task-aware 静态生成 golden tests，固定目标过滤、任务推荐测试名、coverage task 注释和建议输入代入后的代表性输出。

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

## 近期完成标准

第一个有价值的里程碑是：

- [x] `run_tests` 能从 `go test -json` 返回可靠的结构化 Go 失败信息
- [x] 旧版 Go 文本解析仍然可用
- [x] parser 测试覆盖 JSON 和文本输出
- [x] 已知 demo 生成测试即使失败，也能被准确报告
