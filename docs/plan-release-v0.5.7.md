# v0.5.7 发布检查清单

## 当前目标

这是 v0.5.7 的发布准备、正式发布和发布后核验记录。

v0.5.7 发布重点见 [v0.5.7 发布说明](./plan-release-notes-v0.5.7.md)：本轮主要是首跑诊断 CI bootstrap、失败上下文、外部项目复制演练和 CI 入口选择规则。

## 当前差异核对

- [x] `scripts/doctor-first-run.sh` 已加入。
- [x] `docs/first-run-diagnostics.md` 已加入。
- [x] `test/doctor_first_run_test.sh` 已加入并纳入 CI。
- [x] `docs/first-run-failures.md` 已加入。
- [x] `docs/fixtures/first-run/*.txt` 已加入。
- [x] `test/first_run_failure_fixtures_test.sh` 已加入并纳入 CI。
- [x] `scripts/run-first-run-ci.sh` 已加入。
- [x] `docs/first-run-ci-template.md` 已加入。
- [x] `test/run_first_run_ci_test.sh`、`test/first_run_ci_template_doc_test.sh` 和 `test/first_run_ci_template_yaml_test.sh` 已加入并纳入 CI。
- [x] `scripts/showcase-onboarding-ci-external-project.sh` 已加入。
- [x] `docs/onboarding-ci-external-dry-run.md` 已加入。
- [x] `test/onboarding_ci_external_dry_run_doc_test.sh` 已加入并纳入 CI。
- [x] `scripts/showcase-first-run-ci-external-project.sh` 已加入。
- [x] `docs/first-run-ci-external-dry-run.md` 已加入。
- [x] `test/first_run_ci_external_dry_run_doc_test.sh` 已加入并纳入 CI。
- [x] `docs/verification-ci.md` 已补 onboarding / first-run bootstrap 选择规则。
- [x] README、showcase、verification CI 文档、CHANGELOG 和 roadmap 已同步本轮内容。
- [ ] `main.go` MCP implementation version 暂未更新到 `0.5.7`，正式版本准备阶段再改。
- [ ] Homebrew Formula 暂不改 sha256；正式 Release Artifacts 生成后再通过真实 asset digest 更新 tap。

## 候选内容

- [x] 安装后首跑诊断：`scripts/doctor-first-run.sh`。
- [x] 首跑失败上下文：`first-run-context.txt`。
- [x] 首跑失败样例库：`docs/fixtures/first-run/*.txt`。
- [x] 首跑诊断 CI bootstrap：`scripts/run-first-run-ci.sh`。
- [x] 首跑诊断 CI 复制模板：`docs/first-run-ci-template.md`。
- [x] First-run helper ref 默认 `main`，支持当前 main helper 搭配 v0.5.6 二进制。
- [x] Onboarding CI 外部项目复制演练：`scripts/showcase-onboarding-ci-external-project.sh`。
- [x] 首跑诊断 CI 外部项目复制演练：`scripts/showcase-first-run-ci-external-project.sh`。
- [x] CI 入口选择规则：`docs/verification-ci.md#怎么选入口`。
- [x] Go / Node 外部项目真实 dry-run 记录。

## 已验证

- [x] `go test ./...`
- [x] 全部 shell 脚本语法检查。
- [x] 全部默认 shell 回归测试。
- [x] `TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE=all scripts/showcase-first-run-ci-external-project.sh` 真实 dry-run。
- [x] `git diff --check`
- [x] 主服务 / testgen 构建。
- [x] 主服务 / testgen `--help` 输出 usage；Go flag 当前 help exit code 为 `2`。
- [x] darwin arm64 打包 dry-run。
- [x] sha256 校验和 tarball 内容检查。
- [x] 远端 CI run `29651790811` 通过。

## 发布前门禁

- [x] `find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- [x] `go test ./...`
- [x] `for f in $(find test -maxdepth 1 -name '*_test.sh' -print | sort); do sh "$f"; done`
- [x] `go build -o /tmp/testloop-mcp-v0.5.7-candidate .`
- [x] `go build -o /tmp/testloop-testgen-v0.5.7-candidate ./cmd/testgen`
- [x] `/tmp/testloop-mcp-v0.5.7-candidate --help` 输出 usage；exit code 为 `2`。
- [x] `/tmp/testloop-testgen-v0.5.7-candidate --help` 输出 usage；exit code 为 `2`。
- [x] `TESTLOOP_MCP_DIST_DIR=/tmp/testloop-v0.5.7-candidate-dist scripts/package-release-asset.sh v0.5.7 darwin_arm64 darwin arm64`
- [x] 在 dist 目录内校验 `testloop-mcp_v0.5.7_darwin_arm64.tar.gz.sha256` 通过。
- [x] 本地 tarball 内容包含 `testloop-mcp`、`testloop-testgen`、`README.md` 和 `LICENSE`。
- [x] `git diff --check`

## 正式发布前待办

- [ ] 更新 `main.go` MCP implementation version 到 `0.5.7`。
- [ ] 将 `CHANGELOG.md` 的 `Unreleased` 内容收敛到 `v0.5.7 - 2026-07-19`。
- [ ] 同步 README 中当前 Release、手动下载示例、Windows 下载示例到 `v0.5.7`。
- [ ] 同步 `docs/installation.md` 中 `TESTLOOP_MCP_VERSION`、资产列表、下载示例和 Homebrew 维护示例到 `v0.5.7`。
- [ ] quickstart、onboarding、first-run、verification report、verification CI 示例中的版本门禁同步到 `0.5.7`。
- [ ] 测试中的版本期望同步到 `0.5.7`。
- [ ] 重新运行完整验证：`go test ./...`、所有默认 shell 校验、脚本语法检查、主服务/CLI 构建、打包 dry-run。
- [ ] 提交版本准备改动后确认远端 CI passed。
- [ ] 打 tag `v0.5.7` 并推送。
- [ ] Release Artifacts workflow passed，五平台资产和 `.sha256` 已生成。
- [ ] 使用 `scripts/verify-release-assets.sh v0.5.7` 验证 10 个 Release 资产完整。
- [ ] 更新 GitHub Release 正文为正式 v0.5.7 发布说明。
- [ ] 使用 `scripts/generate-homebrew-formula.sh v0.5.7` 更新仓库内 Formula。
- [ ] 更新 Homebrew tap 到 `0.5.7` 并推送。
- [ ] 手动触发 Post-Release Verify，资产清单和五平台安装脚本 dry run 全部通过。

## 当前结论

v0.5.7 候选发布资料已建立，release readiness 门禁和远端 CI 均已通过。当前尚未进入正式版本准备，因此 `main.go` 版本号、安装文档版本引用、CHANGELOG 版本归档、tag、Release Artifacts 和 Homebrew tap 都保持待办。
