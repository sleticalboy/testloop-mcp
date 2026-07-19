# v0.5.12 发布检查清单

## 当前目标

这是 v0.5.12 的候选发布检查清单。当前目标是把 v0.5.11 之后围绕 regression smoke 可复跑性、静态 fixture 和 preflight 诊断层的改动归档为一个 patch 版本。

发布重点见 [v0.5.12 发布说明草案](./plan-release-notes-v0.5.12.md)。

当前阶段只整理候选内容和本地门禁，不切版本号、不打 tag、不更新 Homebrew tap。

## 当前差异核对

- [x] Java regression smoke 默认任务输入已迁入仓库 `testdata/`。
- [x] JS regression smoke 默认任务输入已迁入仓库 `testdata/`。
- [x] Python regression smoke 默认任务输入已迁入仓库 `testdata/`。
- [x] Python Click ready fixture 已基于 Click `8.2.1` 重建。
- [x] Python top-task 验证已支持 `TESTLOOP_VALIDATE_PY_LIST_TASKS_ONLY`。
- [x] `scripts/fixture-task-jsonl.py` 已明确定位为维护者重建 fixture 的辅助工具。
- [x] `scripts/validate-regression-preflight.sh` 已加入。
- [x] `scripts/validate-regression-smoke.sh` 默认先运行 preflight。
- [x] preflight JSON summary 已加入。
- [x] preflight 中文 Markdown 渲染器已加入。
- [x] README、regression smoke 文档、showcase 文档、CHANGELOG 和 roadmap 已记录候选内容。

## 候选内容

- [x] 仓库内静态 regression fixture。
- [x] Java/JS/Python fixture 结构测试。
- [x] Python Click list-only 候选任务导出。
- [x] regression smoke 默认 preflight。
- [x] preflight text / JSON 双输出。
- [x] preflight JSON 到中文准备清单的渲染脚本。
- [x] 带 preflight 的全量 regression smoke 验证记录。
- [x] release readiness 复验记录。

## 已验证

- [x] `sh test/java_regression_fixture_test.sh`
- [x] `sh test/js_regression_fixture_test.sh`
- [x] `sh test/py_regression_fixture_test.sh`
- [x] `sh test/fixture_task_jsonl_script_test.sh`
- [x] `sh test/regression_preflight_test.sh`
- [x] `sh test/regression_preflight_report_test.sh`
- [x] `scripts/validate-regression-preflight.sh`
- [x] `TESTLOOP_REGRESSION_PREFLIGHT_FORMAT=json scripts/validate-regression-preflight.sh`
- [x] `TESTLOOP_REGRESSION_PREFLIGHT_FORMAT=json scripts/validate-regression-preflight.sh | scripts/render-regression-preflight-report.py -`
- [x] `TESTLOOP_VALIDATE_JS_STAGE_TIMEOUT_SECONDS=180 TESTLOOP_VALIDATE_JS_TASK_TIMEOUT_SECONDS=180 TESTLOOP_VALIDATE_PY_STAGE_TIMEOUT_SECONDS=180 TESTLOOP_VALIDATE_PY_TASK_TIMEOUT_SECONDS=180 scripts/validate-regression-smoke.sh`
- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `sh test/docs_links_test.sh`
- [x] `sh test/release_doc_index_test.sh`
- [x] `sh test/verification_ci_doc_test.sh`
- [x] `go build -o /tmp/testloop-mcp-current-candidate .`
- [x] `go build -o /tmp/testloop-testgen-current-candidate ./cmd/testgen`
- [x] `/tmp/testloop-mcp-current-candidate --version` 输出 `testloop-mcp 0.5.11`，正式版本准备前未提前切版本号。
- [x] `/tmp/testloop-mcp-current-candidate --help` 输出 `Usage of`，exit code 为 `2`。
- [x] `/tmp/testloop-testgen-current-candidate --help` 输出 `Usage: testgen`，exit code 为 `2`。
- [x] 正式版本准备后 `/tmp/testloop-mcp-v0.5.12-release-prep --version` 输出 `testloop-mcp 0.5.12`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-current-candidate-dist scripts/package-release-asset.sh v0.5.12 darwin_arm64 darwin arm64`
- [x] `cd /tmp/testloop-current-candidate-dist && shasum -a 256 -c testloop-mcp_v0.5.12_darwin_arm64.tar.gz.sha256`
- [x] `tar -tzf /tmp/testloop-current-candidate-dist/testloop-mcp_v0.5.12_darwin_arm64.tar.gz`，内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] 正式版本准备复跑：版本相关文档/脚本测试、文档 gate、`go test ./...`、主服务/testgen 构建、`testloop-mcp 0.5.12` 版本输出、help 输出、darwin arm64 打包 dry-run、sha256 校验、tarball 内容检查和 `git diff --check`。
- [x] `git diff --check`
- [x] `e669ed9` 远端 CI run `29682473349` passed，覆盖 Python Click regression fixture 重建入仓。
- [x] `b74fe40` 远端 CI run `29682968710` passed，覆盖仓库内 manual-review regression fixture。
- [x] `ceca15c` 远端 CI run `29683297970` passed，覆盖 JS mcp-hub 静态 fixture。
- [x] `fe92caa` 远端 CI run `29683433546` passed，覆盖 Python haoy-apk-station 静态 fixture。
- [x] `802b040` 远端 CI run `29683682501` passed，覆盖静态 fixture CI 结果归档。
- [x] `08494e4` 远端 CI run `29683922243` passed，覆盖全量静态 regression smoke 证据归档。
- [x] `80987d3` 远端 CI run `29684096502` passed，覆盖 fixture task helper 定位清理。
- [x] `6e3076d` 远端 CI run `29684483190` passed，覆盖 regression smoke preflight。
- [x] `df6619c` 远端 CI run `29684677073` passed，覆盖 preflight JSON output。
- [x] `f9295b4` 远端 CI run `29684888366` passed，覆盖 preflight report 渲染。
- [x] `59124ba` 远端 CI run `29685059506` passed，覆盖 release readiness 记录。

## 发布前门禁

- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `go build -o /tmp/testloop-mcp-current-candidate .`
- [x] `go build -o /tmp/testloop-testgen-current-candidate ./cmd/testgen`
- [x] `/tmp/testloop-mcp-current-candidate --version` 输出 `testloop-mcp 0.5.11`，正式版本准备前未提前切版本号。
- [x] `/tmp/testloop-mcp-current-candidate --help` 输出 `Usage of`，exit code 为 `2`。
- [x] `/tmp/testloop-testgen-current-candidate --help` 输出 `Usage: testgen`，exit code 为 `2`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-current-candidate-dist scripts/package-release-asset.sh v0.5.12 darwin_arm64 darwin arm64`
- [x] 在 dist 目录内校验 `testloop-mcp_v0.5.12_darwin_arm64.tar.gz.sha256` 通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 正式发布前待办

- [x] 更新 `main.go` MCP implementation version 到 `0.5.12`。
- [x] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.12 - 2026-07-19`。
- [x] 同步 README、installation、quickstart 和必要版本引用到 `v0.5.12`。
- [x] 测试中的版本期望同步到 `0.5.12`。
- [x] 重新运行完整本地验证，确认版本准备改动可发布。
- [ ] 提交版本准备改动后确认远端 CI passed。
- [ ] 打 tag `v0.5.12` 并推送。
- [ ] Release Artifacts workflow 生成五平台资产和 `.sha256`。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.12` 验证 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.12 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.12` 更新仓库内 Formula。
- [ ] 更新 Homebrew tap 到 `0.5.12` 并推送。
- [ ] 手动触发 Post-Release Verify，确认资产清单和五平台安装脚本 dry run 通过。

## 当前结论

v0.5.12 正式版本准备的文件同步和本地验证已经完成。下一步是提交版本准备改动并等待 main CI；CI 通过后再决定是否打 `v0.5.12` tag 和进入 Release Artifacts 流程。
