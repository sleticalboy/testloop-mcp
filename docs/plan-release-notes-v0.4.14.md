# v0.4.14 发布说明

## 标题

testloop-mcp v0.4.14

## 发布状态

- [x] 创建 v0.4.14 发布说明草案。
- [x] 确认 v0.4.13 之后的 `Unreleased` 范围主要是 Go coverage task 闭环质量、真实项目验证和 skipped task 分类。
- [x] 用 laoxia server 真实 Go 项目作为回归样本，完成 top50 隔离验证记录。
- [x] 正式版本准备时更新 `main.go` MCP implementation version 到 `0.4.14`。
- [x] 正式版本准备时将 `CHANGELOG.md` 的 Unreleased 内容收敛为 `v0.4.14 - 2026-07-11`。
- [x] 正式版本准备时更新 README、安装文档和必要的版本引用。
- [x] 正式发布前重新跑完整本地验证、远端 CI、Release Artifacts、资产校验和 Homebrew 安装链路验证。
- [ ] Post-Release Verify run `29157901152` 已触发但仍 queued，尚未完成五平台安装脚本 dry run。

## 摘要

v0.4.14 候选重点是把 Go coverage task 从“能生成并跑通”推进到“能针对真实覆盖率缺口生成更有价值的可执行断言”。这个版本不试图替代 Claude、Cursor 或 Codex 写完整业务测试，而是给 AI Agent 一个更稳定的测试反馈闭环：

- `parse_coverage` 产出的单个 task 可以直接交给 `validate_coverage_task`。
- `validate_coverage_task` 会执行 `generate_tests -> run_tests`，并返回下一步动作。
- Go static generator 对可静态构造的任务尽量生成 `skip: false` 的真实断言。
- 对不可达分支和运行环境依赖分支，工具给出明确人工复核分类，避免 Agent 反复尝试无效生成。

## 主要变化

- 新增 `validate_coverage_task` MCP 工具，把单个 coverage task 的生成、执行和反馈合成一次调用。
- `validate_coverage_task` 会将疑似不可达 skipped task 标记为 `manual_review_unreachable`，并在 metadata 中返回 `unreachable_reason`。
- `validate_coverage_task` 会将系统资源错误分支这类依赖 OS/runtime 的 skipped task 标记为 `manual_review_environment`，并在 metadata 中返回 `environment_reason`。
- Go coverage task 写入已有测试文件时支持追加新测试函数，不再覆盖已有测试。
- Go coverage task 推荐测试名冲突时会追加稳定后缀，例如基于覆盖率行段生成 `TestGetRawCoverage204_207`。
- Go `run_tests` 对 module 内绝对路径会自动切到 `go.mod` 根目录并转换为相对包路径。
- Go `init` coverage task 会生成明确人工复核 skip，不再写出不可调用的 `init()`。
- Go static generator 增强了多类可静态构造场景：
  - 时间字符串和 `time.Time` 日期边界断言。
  - 简单分支、`&&` 复合分支、整数范围、字符串/布尔/nil 条件输入。
  - `err == nil` / `err != nil` 分支输入。
  - URL/API 字符串参数错误路径。
  - `*http.Request` 分支输入，包括 `RemoteAddr`、`X-Forwarded-For`、`X-Real-IP`。
  - JSON/error 分支和 `FromJsonFile` 成功返回路径。
  - `ParseToken` JWT 成功分支。
  - `Recover` panic/recover 分支。
  - `GetJson` / `GetBytes` HTTP wrapper 本地 `httptest` 输入。
  - `TraceTransport.RoundTrip` 慢请求分支。
  - `Ptr` 泛型指针返回路径。
  - `RemoteIP` fallback return/statement path。
  - `BeforeSave(*gorm.DB) error` 这类 receiver 字段变更方法，可断言 trim 和默认值填充。
- 新增 `scripts/validate-go-coverage-top-tasks.sh` 开发辅助脚本，可对真实 Go 项目的前 N 个 coverage task 做隔离验证，并输出 JSONL 结果和 summary。

## 真实项目验证

本轮使用 `/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server` 作为真实 Go server 样本。验证方式采用隔离副本执行，避免污染原项目工作区。

阶段性结果：

- 首轮 top50 扩窗后：50/50 `passed`，但大量任务仍是 skipped TODO。
- 逐步增强 Go generator 后：低依赖 utility、HTTP、JSON、JWT、recover、trace、model hook 等可达任务被转为真实执行测试。
- 最新隔离 top50：50/50 `passed`，45 个任务 `skipped=0`。
- 剩余 5 个 skipped 均有明确分类：
  - 2 个 `manual_review_unreachable`。
  - 3 个 `manual_review_environment`。
  - `skipped_ready=[]`，即没有普通 ready 队列里的未解释 skipped TODO。

这说明当前 Go static generator 在该样本 top50 中已经清掉所有可静态构造的 skipped task，剩余任务不再是生成质量缺口，而是源码可达性或运行环境可控性问题。

## 质量边界

v0.4.14 候选仍保持明确边界：

- Go static generator 只覆盖可静态、可稳定构造的任务，不伪造 happy-path 测试冒充 error branch 覆盖。
- 系统资源类错误分支没有依赖注入点时，不通过 mock OS/runtime 内部错误来制造脆弱测试。
- `manual_review_unreachable` 和 `manual_review_environment` 是面向 Agent 的动作分类，不代表源码一定无需改动。
- receiver mutation 断言目前识别 `strings.TrimSpace(receiver.Field)` 和 `if Field == "" { Field = "default" }` 这类明确 AST 模式；复杂业务 hook 仍需要 Agent 或 LLM provider 读取更多上下文。
- laoxia top50 是真实项目证据，不等同于所有 Go 项目都能达到相同比例。

## 回归保护

- handler 测试固定 `validate_coverage_task` 的生成、执行、调整测试名、不可达分类和环境依赖分类。
- generator 测试固定 Go 分支输入、HTTP/JSON/JWT/recover/trace/Ptr/RemoteIP/BeforeSave 等新增 seed 场景。
- run_tests 测试固定 Go module 路径归一化和绝对路径执行。
- opt-in 集成测试 `TestValidateGoCoverageTopTasks` 固定真实 Go 项目 top task 验证流程；默认未设置 `TESTLOOP_VALIDATE_GO_PROJECT_DIR` 时跳过，不影响常规 `go test ./...`。
- roadmap 记录 laoxia top50 的阶段性指标，方便后续对比回归。

## 验证

当前候选草案已经完成的验证：

- [x] `go test ./internal/generator -run 'TestGenerateGoTestsForCoverageTaskAssertsBeforeSave'`
- [x] `go test ./...`
- [x] `git diff --check`
- [x] laoxia server 隔离 top50：50/50 `passed`，45 个 `skipped=0`，剩余 5 个均为人工复核分类。
- [x] `scripts/validate-go-coverage-top-tasks.sh /Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server 50`
- [x] 远端 CI 通过：run `29155820565`。

正式发布前至少需要重新执行：

- [x] `go test ./...`
- [x] `git diff --check`
- [x] `go build -o /tmp/testloop-mcp-v0.4.14-prep .`
- [x] `go build -o /tmp/testloop-testgen-v0.4.14-prep ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.4.14-prep --help`
- [x] `/tmp/testloop-testgen-v0.4.14-prep --help`
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.4.14-prep scripts/package-release-asset.sh v0.4.14 darwin_arm64 darwin arm64`
- [x] 校验 `/tmp/testloop-v0.4.14-prep/testloop-mcp_v0.4.14_darwin_arm64.tar.gz.sha256`
- [x] 推送 tag 后等待 Release Artifacts workflow 完成：run `29157722825` passed。
- [x] `scripts/verify-release-assets.sh v0.4.14`
- [x] Homebrew tap 核验：tap commit `6394533b9f999bd2125efab6ace6f3c1e81da180`，`brew fetch`、`brew audit --formula --strict`、`brew upgrade --formula`、`brew test` 均通过。
- [ ] Post-Release Verify run `29157901152` 仍 queued，首个 `ubuntu-latest` job 尚未拿到 runner。

## 发布备注

- 这是 post-v0.4.13 的正式发布资料，不回写已经发布的 `docs/plan-release-notes-v0.4.13.md`。
- 本轮重点是 Go coverage task 的真实项目闭环质量，不改变默认 provider 策略。
- `validate_coverage_task` 的 action 分类是给 AI Agent 的执行信号：`ready` 可继续吸收测试，`manual_review_unreachable` / `manual_review_environment` 应进入人工复核或源码可测性改造。
- laoxia top50 隔离验证已沉淀为 `scripts/validate-go-coverage-top-tasks.sh` 和 opt-in 集成测试，后续可继续扩展更多真实项目样本。
