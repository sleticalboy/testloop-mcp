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

## 近期完成标准

第一个有价值的里程碑是：

- [x] `run_tests` 能从 `go test -json` 返回可靠的结构化 Go 失败信息
- [x] 旧版 Go 文本解析仍然可用
- [x] parser 测试覆盖 JSON 和文本输出
- [x] 已知 demo 生成测试即使失败，也能被准确报告
