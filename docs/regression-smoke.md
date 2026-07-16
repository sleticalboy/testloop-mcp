# 固定 smoke 回归说明

固定 smoke 用于低成本验证真实项目里的测试反馈闭环：

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
| JS | `scripts/validate-js-regression-samples.sh` | ip2region JavaScript binding `jest-1/jest-2` | `ready` |
| Python | `scripts/validate-py-regression-samples.sh` | Click `pytest-1/pytest-3` | `ready` |

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
| Python / Click | `/tmp/testloop-click-sample` | `/tmp/testloop-click-pytest-top5-regression.jsonl` |

路径不一致时，用对应环境变量覆盖：

```bash
TESTLOOP_JAVA_REGRESSION_LANG_DIR=/path/to/commons-lang \
TESTLOOP_JAVA_REGRESSION_CODEC_DIR=/path/to/commons-codec \
TESTLOOP_JS_REGRESSION_IP2REGION_DIR=/path/to/ip2region/binding/javascript \
TESTLOOP_PY_REGRESSION_CLICK_DIR=/path/to/click \
scripts/validate-regression-smoke.sh
```

## 跳过单个语言

```bash
TESTLOOP_REGRESSION_SKIP_JAVA=true scripts/validate-regression-smoke.sh
TESTLOOP_REGRESSION_SKIP_JS=true scripts/validate-regression-smoke.sh
TESTLOOP_REGRESSION_SKIP_PY=true scripts/validate-regression-smoke.sh
```

## 关键 runner

JS/ip2region 使用 Jest ESM，需要固定到单个生成测试文件，否则 `jest util.test.js` 会误匹配项目已有 `tests/util.test.js`：

```bash
TESTLOOP_JS_TEST_COMMAND="NODE_OPTIONS='--experimental-vm-modules --no-warnings' npx jest --runTestsByPath {path}"
```

Python/Click 默认使用 `uv`：

```bash
TESTLOOP_PYTEST_COMMAND="uv run python -m pytest {verbose} {coverage} {path}"
```

## 当前边界

- JS 默认 smoke 只覆盖 `ready` 样本。ip2region 扩大窗口会暴露 `repair_generated_test`，但这类普通失败不应进入默认 smoke。
- 旧 ufo JSONL 包含 `manual_review_no_runtime`，但本机当前 ufo 目录只有发布产物，没有对应 `src/*.ts`，不适合作为固定样本。
- Codex SDK TypeScript 的旧 JSONL 包含 `manual_review_internal`，但当前本地 workspace 的独立 `node_modules` 不包含 Jest，复用时会被 runner 依赖污染，不适合作为默认样本。
- GitHub Actions 偶尔会长时间停在 `queued`。这种状态表示 runner 尚未开始执行，不能等同于测试失败。
