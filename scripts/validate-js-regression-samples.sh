#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
用法：scripts/validate-js-regression-samples.sh

运行一组固定的 JS/Jest 小回归样本，复用已经生成过的 coverage task
或 validation JSONL。它用于低成本验证 JS coverage task 闭环，不重跑完整
top-N 覆盖率窗口。

环境变量：
  TESTLOOP_JS_REGRESSION_OUTPUT_DIR
                                    每组样本输出 JSONL 的目录。
                                    默认：/tmp/testloop-js-regression-<timestamp>
  TESTLOOP_JS_REGRESSION_IP2REGION_DIR
                                    ip2region JavaScript binding 项目目录。
                                    默认：/Users/binlee/code/open-source/ip2region/binding/javascript
  TESTLOOP_JS_REGRESSION_IP2REGION_TASKS_FILE
                                    包含 ip2region Jest 任务的 JSONL。
                                    默认：/tmp/testloop-ip2region-js-jest-top2-current.jsonl
  TESTLOOP_JS_REGRESSION_IP2REGION_READY_IDS
                                    默认：jest-1,jest-2
  TESTLOOP_JS_REGRESSION_NO_RUNTIME_DIR
                                    仓库内 TypeScript no-runtime fixture 目录。
                                    默认：testdata/js-no-runtime
  TESTLOOP_JS_REGRESSION_NO_RUNTIME_IDS
                                    默认：jest-no-runtime-1
  TESTLOOP_JS_TEST_COMMAND
                                    默认：NODE_OPTIONS='--experimental-vm-modules --no-warnings' npx jest --runTestsByPath {path}
  TESTLOOP_VALIDATE_JS_STAGE_TIMEOUT_SECONDS
  TESTLOOP_VALIDATE_JS_TASK_TIMEOUT_SECONDS
                                    透传给 validate-js-coverage-top-tasks.sh。
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -ne 0 ]]; then
  usage
  exit 2
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
validator="$script_dir/validate-js-coverage-top-tasks.sh"

output_dir="${TESTLOOP_JS_REGRESSION_OUTPUT_DIR:-/tmp/testloop-js-regression-$(date +%Y%m%d%H%M%S)}"
ip2region_dir="${TESTLOOP_JS_REGRESSION_IP2REGION_DIR:-/Users/binlee/code/open-source/ip2region/binding/javascript}"
ip2region_tasks="${TESTLOOP_JS_REGRESSION_IP2REGION_TASKS_FILE:-/tmp/testloop-ip2region-js-jest-top2-current.jsonl}"
ip2region_ready_ids="${TESTLOOP_JS_REGRESSION_IP2REGION_READY_IDS:-jest-1,jest-2}"
no_runtime_dir="${TESTLOOP_JS_REGRESSION_NO_RUNTIME_DIR:-$repo_root/testdata/js-no-runtime}"
no_runtime_ids="${TESTLOOP_JS_REGRESSION_NO_RUNTIME_IDS:-jest-no-runtime-1}"
js_test_command="${TESTLOOP_JS_TEST_COMMAND:-}"
if [[ -z "$js_test_command" ]]; then
  js_test_command="NODE_OPTIONS='--experimental-vm-modules --no-warnings' npx jest --runTestsByPath {path}"
fi
manual_review_command="node $repo_root/scripts/js-manual-review-runner.js {path}"

require_path() {
  local kind="$1"
  local path="$2"
  if [[ "$kind" == "dir" && ! -d "$path" ]]; then
    echo "required directory does not exist: $path" >&2
    exit 1
  fi
  if [[ "$kind" == "file" && ! -f "$path" ]]; then
    echo "required file does not exist: $path" >&2
    exit 1
  fi
}

task_count() {
  printf '%s' "$1" | tr ',' '\n' | awk 'NF {count++} END {print count+0}'
}

assert_sample_output() {
  local output="$1"
  local expected_ids="$2"
  local expected_action="$3"

  python3 - "$output" "$expected_ids" "$expected_action" <<'PY'
import json
import sys

path, raw_ids, expected_action = sys.argv[1:]
expected_ids = [item.strip() for item in raw_ids.split(",") if item.strip()]

rows = []
with open(path, "r", encoding="utf-8") as handle:
    for line in handle:
        line = line.strip()
        if line:
            rows.append(json.loads(line))

ids = [row.get("coverage_task", {}).get("id", "") for row in rows]
if ids != expected_ids:
    raise SystemExit(f"{path}: ids={ids}, want={expected_ids}")

for row in rows:
    task_id = row.get("coverage_task", {}).get("id", "")
    status = row.get("status")
    action = row.get("action")
    if status != "passed":
        raise SystemExit(f"{path}: {task_id} status={status}, want=passed")
    if action != expected_action:
        raise SystemExit(f"{path}: {task_id} action={action}, want={expected_action}")
PY
}

run_sample() {
  local name="$1"
  local project_dir="$2"
  local tasks_file="$3"
  local ids="$4"
  local expected_action="$5"
  local test_command="${6:-$js_test_command}"
  local output="$output_dir/$name.jsonl"

  require_path dir "$project_dir"
  require_path file "$tasks_file"

  echo "==> $name ids=$ids expected_action=$expected_action"
  (
    cd "$repo_root"
    TESTLOOP_VALIDATE_JS_TASKS_FILE="$tasks_file" \
    TESTLOOP_VALIDATE_JS_TASK_IDS="$ids" \
    TESTLOOP_JS_TEST_COMMAND="$test_command" \
    "$validator" "$project_dir" jest "$(task_count "$ids")" "$output"
  )
  assert_sample_output "$output" "$ids" "$expected_action"
  echo "ok  $name output=$output"
}

mkdir -p "$output_dir"

run_sample "ip2region-ready" "$ip2region_dir" "$ip2region_tasks" "$ip2region_ready_ids" "ready"

no_runtime_tasks="$output_dir/no-runtime-tasks.jsonl"
require_path dir "$no_runtime_dir"
python3 - "$no_runtime_dir" "$no_runtime_tasks" <<'PY'
import json
import os
import sys

project_dir, output = sys.argv[1:]
source = os.path.join(project_dir, "src", "events.ts")
task = {
    "id": "jest-no-runtime-1",
    "framework": "jest",
    "file": source,
    "target": "events.ts",
    "kind": "file_level",
    "line_range": "entire file",
    "gap_type": "no_runtime",
    "goal": "确认 events.ts 是 TypeScript 纯类型文件，没有可直接执行的运行时代码覆盖任务",
    "command": "node scripts/js-manual-review-runner.js tests/events.test.ts",
    "test_file": os.path.join(project_dir, "tests", "events.test.ts"),
    "test_name": "marks type-only module as no runtime coverage",
    "assertion_focus": [
        "纯类型声明不会产生有意义的本地 JavaScript coverage task，应通过消费方运行时测试或类型检查验证"
    ],
    "priority": 90,
    "priority_reason": "repository fixture for stable JS no-runtime manual-review smoke",
    "confidence": 0.9,
}
with open(output, "w", encoding="utf-8") as handle:
    handle.write(json.dumps(task, ensure_ascii=False) + "\n")
PY
run_sample "fixture-no-runtime" "$no_runtime_dir" "$no_runtime_tasks" "$no_runtime_ids" "manual_review_no_runtime" "$manual_review_command"

echo "js_regression_output_dir=$output_dir"
