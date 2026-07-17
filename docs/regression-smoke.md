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
| JS | `scripts/validate-js-regression-samples.sh` | ip2region JavaScript binding `jest-1/jest-2`，仓库内 `testdata/js-no-runtime` 的 `jest-no-runtime-1`，仓库内 `testdata/js-internal` 的 `jest-internal-1` | `ready`、`manual_review_no_runtime`、`manual_review_internal` |
| Python | `scripts/validate-py-regression-samples.sh` | Click `pytest-1/pytest-3`，仓库内 `testdata/py-internal` 的 `pytest-internal-1` | `ready`、`manual_review_internal` |

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
| Python / Click | `/tmp/testloop-click-sample` | `/tmp/testloop-click-pytest-top5-regression.jsonl` |
| Python / internal fixture | `./testdata/py-internal` | 运行时临时生成到输出目录 |

路径不一致时，用对应环境变量覆盖：

```bash
TESTLOOP_JAVA_REGRESSION_LANG_DIR=/path/to/commons-lang \
TESTLOOP_JAVA_REGRESSION_CODEC_DIR=/path/to/commons-codec \
TESTLOOP_JS_REGRESSION_IP2REGION_DIR=/path/to/ip2region/binding/javascript \
TESTLOOP_JS_REGRESSION_NO_RUNTIME_DIR=/path/to/js-no-runtime-fixture \
TESTLOOP_JS_REGRESSION_INTERNAL_DIR=/path/to/js-internal-fixture \
TESTLOOP_PY_REGRESSION_CLICK_DIR=/path/to/click \
TESTLOOP_PY_REGRESSION_INTERNAL_DIR=/path/to/py-internal-fixture \
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
scripts/fixture-task-jsonl.py py-internal ./testdata/py-internal /tmp/py-internal.jsonl
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

JS/mcp-hub 使用真实 Vitest 项目验证普通失败路径。`ConfigManager.loadConfig` 空 config paths 分支当前会生成可运行但断言错误的测试，预期结果是 `failed/repair_generated_test`。该样本通过 `TESTLOOP_VALIDATE_JS_ALLOWED_FAILURE_ACTIONS=repair_generated_test` 显式放行；没有这个开关时，普通失败仍会让 top-N 验证失败。

Python/Click 默认使用 `uv`：

```bash
TESTLOOP_PYTEST_COMMAND="uv run python -m pytest {verbose} {coverage} {path}"
```

Python/internal fixture 使用仓库内轻量 runner，只验证生成的手审 skip 能进入 `run_tests -> parse_results -> validate_coverage_task` 闭环，不依赖 fixture 自身安装 pytest：

```bash
python3 scripts/py-manual-review-runner.py {path}
```

该 runner 会定位包含 `manual_review_*` marker 的 pytest function 或 class method，并输出对应 pytest node id，例如 `tests/test_private_service.py::TestPrivateService::test_private_method_requires_internal_review`。

## 当前边界

- JS 默认 smoke 覆盖 `ready`、`manual_review_no_runtime`、`manual_review_internal` 和真实项目 `repair_generated_test`。仓库内 no-runtime/internal fixture 不是性能或真实业务样本，只用于稳定验证 TypeScript 纯类型文件、未导出 ESM helper 会被降级为可解析的手审任务；mcp-hub repair 样本用于验证真实 Vitest 项目里的生成测试失败能被结构化为下一步修复任务。
- Python 默认 smoke 覆盖 `ready` 和 `manual_review_internal`。仓库内 Python internal fixture 用于稳定验证 name-mangled private method 会被降级为可解析的手审任务；更复杂的真实项目 internal 场景仍需要后续样本。
- ip2region 扩大窗口也会暴露 `repair_generated_test`，但那类普通失败没有固定为默认样本；当前默认 repair 样本只使用 mcp-hub `ConfigManager.loadConfig` 的稳定错误路径。
- 旧 ufo JSONL 包含 `manual_review_no_runtime`，但本机当前 ufo 目录只有发布产物，没有对应 `src/*.ts`，不适合作为固定样本。
- Codex SDK TypeScript 的旧 JSONL 包含更真实的 `manual_review_internal`，但当前本地 workspace 的独立 `node_modules` 不包含 Jest，复用时会被 runner 依赖污染，不适合作为默认样本。
- GitHub Actions 偶尔会长时间停在 `queued`。这种状态表示 runner 尚未开始执行，不能等同于测试失败。
