# 固定 smoke 回归说明

固定 smoke 用于低成本验证真实项目和仓库内 fixture 的测试反馈闭环：

```text
coverage task -> generate_tests -> run_tests -> parse/fix/coverage feedback
```

它不替代完整 top-N 真实项目验证，也不适合作为性能 benchmark。目标是改动后快速确认几个代表性样本没有退化。

## 总入口

```bash
scripts/validate-regression-smoke.sh
```

默认会串联三组样本：

| 语言 | 脚本 | 默认样本 | 期望结果 |
| :--- | :--- | :--- | :--- |
| Java | `scripts/validate-java-regression-samples.sh` | Commons Lang `junit-44/junit-50`，Commons Codec `junit-130`，Commons Lang `junit-52` | `ready`、`manual_review_unreachable`、`manual_review_internal` |
| JS | `scripts/validate-js-regression-samples.sh` | ip2region JavaScript binding `jest-1/jest-2`，仓库内 `testdata/js-no-runtime` 的 `jest-no-runtime-1`，仓库内 `testdata/js-internal` 的 `jest-internal-1`，mcp-hub `vitest-mcp-hub-repair-1/2/3`、`vitest-mcp-hub-env-1/2`、`vitest-mcp-hub-devwatcher-1/2`、`vitest-mcp-hub-sse-1/2/3/4`、`vitest-mcp-hub-workspace-1/2/3` | `ready`、`manual_review_no_runtime`、`manual_review_internal`、`manual_review_environment` |
| Python | `scripts/validate-py-regression-samples.sh` | Click `pytest-1/pytest-3`，仓库内 `testdata/py-internal` 的 `pytest-internal-1`，haoy-apk-station backend 的 `pytest-apk-frontend-env-1/pytest-apk-download-external-1/pytest-apk-delete-db-1` | `ready`、`manual_review_internal`、`manual_review_environment`、`manual_review_external_service`、`manual_review_database` |

输出目录默认是：

```bash
/tmp/testloop-regression-smoke-<timestamp>
```

可以用 `TESTLOOP_REGRESSION_OUTPUT_DIR` 覆盖。

## 依赖路径

这些脚本默认复用本机已经准备好的真实项目目录和 JSONL：

| 语言 | 默认项目目录 | 默认 JSONL |
| :--- | :--- | :--- |
| Java / Commons Lang | `/tmp/testloop-commons-lang` | `/tmp/testloop-commons-lang-taskids-junit44-50-results.jsonl`、`/tmp/testloop-commons-lang-typeutils-top5-results.jsonl` |
| Java / Commons Codec | `/tmp/testloop-commons-codec` | `/tmp/testloop-commons-codec-taskids-junit130-results.jsonl` |
| JS / ip2region | `/Users/binlee/code/open-source/ip2region/binding/javascript` | `/tmp/testloop-ip2region-js-jest-top2-current.jsonl` |
| JS / no-runtime fixture | `./testdata/js-no-runtime` | 运行时临时生成到输出目录 |
| JS / internal fixture | `./testdata/js-internal` | 运行时临时生成到输出目录 |
| JS / mcp-hub | `/Users/binlee/code/open-source/mcp-hub` | 运行时临时生成到输出目录 |
| Python / Click | `/tmp/testloop-click-sample` | `/tmp/testloop-click-pytest-top5-regression.jsonl` |
| Python / internal fixture | `./testdata/py-internal` | 运行时临时生成到输出目录 |
| Python / haoy-apk-station | `/Users/binlee/code/free-works/haoy-apk-station/backend` | 运行时临时生成到输出目录 |

路径不一致时，用对应环境变量覆盖：

```bash
TESTLOOP_JAVA_REGRESSION_LANG_DIR=/path/to/commons-lang \
TESTLOOP_JAVA_REGRESSION_CODEC_DIR=/path/to/commons-codec \
TESTLOOP_JS_REGRESSION_IP2REGION_DIR=/path/to/ip2region/binding/javascript \
TESTLOOP_JS_REGRESSION_NO_RUNTIME_DIR=/path/to/js-no-runtime-fixture \
TESTLOOP_JS_REGRESSION_INTERNAL_DIR=/path/to/js-internal-fixture \
TESTLOOP_JS_REGRESSION_MCP_HUB_DIR=/path/to/mcp-hub \
TESTLOOP_PY_REGRESSION_CLICK_DIR=/path/to/click \
TESTLOOP_PY_REGRESSION_INTERNAL_DIR=/path/to/py-internal-fixture \
TESTLOOP_PY_REGRESSION_APK_STATION_DIR=/path/to/haoy-apk-station/backend \
scripts/validate-regression-smoke.sh
```

## 跳过单个语言

```bash
TESTLOOP_REGRESSION_SKIP_JAVA=true scripts/validate-regression-smoke.sh
TESTLOOP_REGRESSION_SKIP_JS=true scripts/validate-regression-smoke.sh
TESTLOOP_REGRESSION_SKIP_PY=true scripts/validate-regression-smoke.sh
```

## 关键 runner

仓库内 fixture 的 coverage task JSONL 由统一 helper 生成：

```bash
scripts/fixture-task-jsonl.py js-no-runtime ./testdata/js-no-runtime /tmp/js-no-runtime.jsonl
scripts/fixture-task-jsonl.py js-internal ./testdata/js-internal /tmp/js-internal.jsonl
scripts/fixture-task-jsonl.py js-mcp-hub-repair /Users/binlee/code/open-source/mcp-hub /tmp/js-mcp-hub-repair.jsonl
scripts/fixture-task-jsonl.py js-mcp-hub-env /Users/binlee/code/open-source/mcp-hub /tmp/js-mcp-hub-env.jsonl
scripts/fixture-task-jsonl.py js-mcp-hub-devwatcher /Users/binlee/code/open-source/mcp-hub /tmp/js-mcp-hub-devwatcher.jsonl
scripts/fixture-task-jsonl.py js-mcp-hub-sse /Users/binlee/code/open-source/mcp-hub /tmp/js-mcp-hub-sse.jsonl
scripts/fixture-task-jsonl.py js-mcp-hub-workspace /Users/binlee/code/open-source/mcp-hub /tmp/js-mcp-hub-workspace.jsonl
scripts/fixture-task-jsonl.py py-internal ./testdata/py-internal /tmp/py-internal.jsonl
scripts/fixture-task-jsonl.py py-apk-station-environment /Users/binlee/code/free-works/haoy-apk-station/backend /tmp/py-apk-station-environment.jsonl
scripts/fixture-task-jsonl.py py-apk-station-external-service /Users/binlee/code/free-works/haoy-apk-station/backend /tmp/py-apk-station-external-service.jsonl
scripts/fixture-task-jsonl.py py-apk-station-database /Users/binlee/code/free-works/haoy-apk-station/backend /tmp/py-apk-station-database.jsonl
```

JS/ip2region 使用 Jest ESM，需要固定到单个生成测试文件，否则 `jest util.test.js` 会误匹配项目已有 `tests/util.test.js`：

```bash
TESTLOOP_JS_TEST_COMMAND="NODE_OPTIONS='--experimental-vm-modules --no-warnings' npx jest --runTestsByPath {path}"
```

JS/no-runtime 和 internal fixture 使用仓库内轻量 runner，只验证生成的手审 skip 能进入 `run_tests -> parse_results -> validate_coverage_task` 闭环，不依赖外部 Jest/Vitest 安装：

```bash
node scripts/js-manual-review-runner.js {path}
```

该 runner 会从生成测试文件中提取 `describe(...)` 与 `it.skip(...)` 名称，并输出包含 test file、skipped test 名称、summary 和耗时的 Jest 风格文本。

JS/mcp-hub 使用真实 Vitest 项目验证历史普通失败路径和环境依赖分类。`ConfigManager.loadConfig` 空 config paths 分支曾经会生成可运行但断言错误的测试；现在应识别该 `if (...) { throw ... }` 分支，并生成 `await expect(instance.loadConfig()).rejects.toThrow()`，预期结果是 `passed/ready`。`DevWatcher.stop` 固定未 watching 早返回和 watching cleanup 生命周期，断言 debounce timer、changed files、watcher close、watcher 引用和 watching 状态被清理；`DevWatcher.start` 固定 chokidar watcher `error` 事件路径，要求通过 mock watcher 触发事件而不是启动真实文件监听。`SSEManager.setupAutoShutdown` 需要用 `vi.useFakeTimers()`、fake `workspaceCache` 和一次性 `SIGTERM` listener 覆盖自动关闭 timer，不应 mock `process.emit`，否则会干扰 Vitest worker 内部通信；`SSEManager.addConnection` 的 close 生命周期需要用 `EventEmitter` request 触发 `req.emit('close')`，断言连接表清理、状态变更和 workspace cache 从 1 更新到 0；send failure 路径通过 throwing `res.write` 触发 `connection.send()` 返回 `false`、状态变为 `error`，再用 `broadcast` 清理 dead connection；`SSEManager.sendToClient` 固定缺失 client、disconnected client 返回 `false` 且不调用 `send`，connected client 委托 `connection.send`。`WorkspaceCacheManager.updateWorkspaceState` 和 `cleanupStaleEntries` 必须 mock `_withLock/_readCache/_writeCache`，其中 stale cleanup 还必须 mock `_isProcessRunning`，避免触碰真实 XDG cache/lock 文件和真实进程探测；`WorkspaceCacheManager._withLock` 依赖真实文件锁、重试时序和 stale lock 清理，固定样本预期生成 `manual_review_environment` skip，而不是直接调用真实 `_withLock`。

Python/Click 默认使用 `uv`：

```bash
TESTLOOP_PYTEST_COMMAND="uv run python -m pytest {verbose} {coverage} {path}"
```

Python/internal fixture 使用仓库内轻量 runner，只验证生成的手审 skip 能进入 `run_tests -> parse_results -> validate_coverage_task` 闭环，不依赖 fixture 自身安装 pytest：

```bash
python3 scripts/py-manual-review-runner.py {path}
```

该 runner 会定位包含 `manual_review_*` marker 的 pytest function 或 class method，并输出对应 pytest node id，例如 `tests/test_private_service.py::TestPrivateService::test_private_method_requires_internal_review`。

Python/haoy-apk-station 使用真实 FastAPI 项目验证 environment 手审路径。`app.main` 中的 `serve_frontend` 只会在 `frontend/dist` 存在时于模块导入阶段动态定义，固定样本预期生成 `manual_review_environment` skip，并提示通过导入前创建 `frontend/dist/index.html` 的集成 fixture 覆盖，而不是直接调用 `lifespan` 或导入不存在的动态函数。

Python/haoy-apk-station 还使用真实 FastAPI 下载代理验证 external-service 路径。`app.api.apps.download_apk` 的代理下载分支依赖外部对象存储 endpoint 和 `urllib.request.urlopen(..., timeout=60)`；固定 runner 会输出 pytest 风格 timeout 失败，预期 `validate_coverage_task` 返回 `failed/manual_review_external_service`，表示应通过 fake storage client、route data 或集成环境验证，而不是继续普通修生成测试。

Python/haoy-apk-station 还使用真实 FastAPI 删除应用路径验证 database 手审路径。`app.api.apps.delete_app` 同时删除版本、下载日志和应用记录，`db.commit()` 失败依赖 SQLAlchemy session/事务行为；固定样本预期生成 `manual_review_database` skip，表示应通过测试数据库、注入 session/repository 或集成 fixture 验证，而不是把事务错误分支伪造成普通 ready。

## 当前边界

- JS 默认 smoke 覆盖 `ready`、`manual_review_no_runtime`、`manual_review_internal` 和真实项目 `manual_review_environment`。仓库内 no-runtime/internal fixture 不是性能或真实业务样本，只用于稳定验证 TypeScript 纯类型文件、未导出 ESM helper 会被降级为可解析的手审任务；mcp-hub 样本用于防止真实 Vitest 项目里的 async throwing branch 从 `ready` 回退成 `repair_generated_test`，也用于防止 workspace cache 这类 XDG 文件锁/进程探测路径被误判成可安全直接运行的 ready 测试，并固定 DevWatcher stop 不能退化成只测未 watching 早返回、watcher error 不能启动真实 chokidar、SSE 自动关闭 timer 不能通过 mock `process.emit` 破坏测试 runner、连接断开不能退化成只注册空 `req.on` 的弱测试、发送失败不能退化成不可触达的空 `res.write` mock、定向发送不能退化成只测 missing client 的弱断言。
- Python 默认 smoke 覆盖 `ready`、`manual_review_internal`、真实项目 `manual_review_environment`、真实项目 `manual_review_external_service` 和真实项目 `manual_review_database`。仓库内 Python internal fixture 用于稳定验证 name-mangled private method 会被降级为可解析的手审任务；haoy-apk-station 样本用于验证 FastAPI 动态前端入口这类导入时环境依赖不会被误当成普通 ready，也用于验证对象存储 endpoint timeout 会被归类为外部服务手审、SQLAlchemy 事务错误会被归类为数据库手审，而不是普通 repair。
- ip2region 扩大窗口也会暴露 `repair_generated_test`，但那类普通失败没有固定为默认样本；当前默认 mcp-hub 样本固定的是历史 repair 已收敛的 `ConfigManager.loadConfig` 稳定错误路径。
- 旧 ufo JSONL 包含 `manual_review_no_runtime`，但本机当前 ufo 目录只有发布产物，没有对应 `src/*.ts`，不适合作为固定样本。
- Codex SDK TypeScript 的旧 JSONL 包含更真实的 `manual_review_internal`，但当前本地 workspace 的独立 `node_modules` 不包含 Jest，复用时会被 runner 依赖污染，不适合作为默认样本。
- GitHub Actions 偶尔会长时间停在 `queued`。这种状态表示 runner 尚未开始执行，不能等同于测试失败。
