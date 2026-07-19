# v0.5.12 发布说明草案

## 标题

testloop-mcp v0.5.12

## 发布状态

- [x] 创建 v0.5.12 发布说明草案。
- [x] 梳理 v0.5.11 之后的 regression fixture 静态化、Click fixture 重建、全量 smoke 证据、preflight 诊断层、JSON summary 和中文准备清单渲染器。
- [x] 本地 release readiness 复验已通过，覆盖完整测试矩阵、候选二进制构建、help/version、darwin arm64 打包 dry-run、sha256 和 tarball 内容检查。
- [x] 最近 CI 已通过：`59124ba` 远端 CI run `29685059506` passed。
- [x] 正式版本准备已更新 `main.go` MCP implementation version 到 `0.5.12`。
- [x] 正式版本准备已将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.5.12 - 2026-07-19`。
- [x] 正式版本准备已同步 README、安装文档和必要版本引用到 `v0.5.12`。
- [x] 正式版本准备本地完整验证已通过。
- [ ] 尚未打 tag、生成 Release 资产或更新 Homebrew tap。

## 摘要

v0.5.12 候选重点是把真实项目 regression smoke 从“依赖临时 JSONL 和维护者记忆”推进到“仓库内静态 fixture、启动前诊断、机器可读缺失项和用户可读准备清单”。

这个版本仍然不扩语言、不改变 MCP tool 协议，也不把项目重新定位成单纯测试生成器。核心价值是提升 AI Agent 测试反馈闭环的可复跑性：默认 smoke 输入更稳定，跨机器运行前能快速知道缺什么，Agent 也能把缺失项转成可执行的准备步骤。

## 主要变化

### Regression fixture 静态化

- Java regression smoke 的 Commons Lang、Commons Codec 和 RocketMQ `StatusChecker.java` 任务输入已迁入 `testdata/`。
- JS regression smoke 的 ip2region、仓库内 no-runtime/internal、mcp-hub repair/env/DevWatcher/SSE/workspace 任务输入已迁入 `testdata/`。
- Python regression smoke 的 Click、仓库内 internal、haoy-apk-station environment/external-service/database 任务输入已迁入 `testdata/`。
- 新增 JS/Python/Java fixture 结构测试，固定 task id、framework、目标符号、行段和推荐测试文件。

### Python Click fixture 重建

- Python coverage top-task 验证支持 `TESTLOOP_VALIDATE_PY_LIST_TASKS_ONLY`。
- Click ready 样本基于 Click `8.2.1` 重新挑选，避免继续使用已漂移的旧 parser 私有 helper 行段。
- 当前固定七个已验证 ready 样本：`pytest-19/20/21/22/23/32/33`。

### Regression smoke preflight

- 新增 `scripts/validate-regression-preflight.sh`。
- `scripts/validate-regression-smoke.sh` 默认先运行 preflight，提前检查真实项目目录、仓库内静态 JSONL 和常用命令。
- 可通过 `TESTLOOP_REGRESSION_SKIP_PREFLIGHT=true` 临时跳过前置检查。
- preflight 支持 `TESTLOOP_REGRESSION_PREFLIGHT_FORMAT=json`，输出 `ok`、`missing_count`、`missing` 和 `checks`。

### Agent 可读准备清单

- 新增 `scripts/render-regression-preflight-report.py`。
- 该脚本可把 preflight JSON 转成中文 Markdown。
- 通过时输出继续运行 smoke 的命令。
- 未通过时按 command、dir、file 分组列出缺失项，并提示用 `TESTLOOP_*_REGRESSION_*` 环境变量改到本机路径。

## 质量边界

- regression smoke 仍面向维护者，不面向首次接入用户。
- 静态 JSONL 降低了临时文件漂移，但真实项目 checkout 仍是运行前提。
- preflight 只检查显式前置条件，不执行覆盖率、不生成测试，也不保证真实项目测试一定通过。
- Python haoy-apk-station external-service 样本的预期仍是 `failed/manual_review_external_service`，这是正确分流，不是普通失败。

## 本地验证

- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `scripts/validate-regression-preflight.sh`
- [x] `TESTLOOP_REGRESSION_PREFLIGHT_FORMAT=json scripts/validate-regression-preflight.sh`
- [x] `TESTLOOP_REGRESSION_PREFLIGHT_FORMAT=json scripts/validate-regression-preflight.sh | scripts/render-regression-preflight-report.py -`
- [x] `TESTLOOP_VALIDATE_JS_STAGE_TIMEOUT_SECONDS=180 TESTLOOP_VALIDATE_JS_TASK_TIMEOUT_SECONDS=180 TESTLOOP_VALIDATE_PY_STAGE_TIMEOUT_SECONDS=180 TESTLOOP_VALIDATE_PY_TASK_TIMEOUT_SECONDS=180 scripts/validate-regression-smoke.sh`
- [x] `go build -o /tmp/testloop-mcp-current-candidate .`
- [x] `go build -o /tmp/testloop-testgen-current-candidate ./cmd/testgen`
- [x] `/tmp/testloop-mcp-current-candidate --version` 输出 `testloop-mcp 0.5.11`，说明候选阶段尚未切新版本号。
- [x] `/tmp/testloop-mcp-current-candidate --help` 输出 usage，exit code 为 `2`。
- [x] `/tmp/testloop-testgen-current-candidate --help` 输出 usage，exit code 为 `2`。
- [x] 正式版本准备后 `/tmp/testloop-mcp-v0.5.12-release-prep --version` 输出 `testloop-mcp 0.5.12`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-current-candidate-dist scripts/package-release-asset.sh v0.5.12 darwin_arm64 darwin arm64`
- [x] `cd /tmp/testloop-current-candidate-dist && shasum -a 256 -c testloop-mcp_v0.5.12_darwin_arm64.tar.gz.sha256`
- [x] `tar -tzf /tmp/testloop-current-candidate-dist/testloop-mcp_v0.5.12_darwin_arm64.tar.gz`
- [x] 正式版本准备复跑：版本相关文档/脚本测试、文档 gate、`go test ./...`、主服务/testgen 构建、`testloop-mcp 0.5.12` 版本输出、help 输出、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。
- [x] `git diff --check`

## 发布前待办

- [x] 完成候选发布检查清单 `docs/plan-release-v0.5.12.md`。
- [x] 正式版本准备时更新 `main.go` MCP implementation version 到 `0.5.12`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛为 `v0.5.12 - 2026-07-19`。
- [x] 同步 README、installation、quickstart 和必要版本引用到 `0.5.12` / `v0.5.12`。
- [x] 跑完整正式发布前门禁和 release readiness。
- [ ] 提交版本准备后等待远端 CI。
- [ ] 打 `v0.5.12` tag，生成 Release 资产，更新 GitHub Release。
- [ ] 生成并更新 Homebrew Formula / tap。
- [ ] 触发 Post-Release Verify，确认五平台安装脚本 dry run 通过。

## 发布备注

- 对外文案应突出“维护者 regression smoke 更可复跑、Agent 可读诊断更稳定”。
- 不要宣传成新增语言支持或测试生成算法大改。
- 推荐演示路径：先运行 preflight JSON 到 Markdown 管道，再运行固定 regression smoke。
