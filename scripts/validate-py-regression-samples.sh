#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
用法：scripts/validate-py-regression-samples.sh

运行一组固定的 Python/pytest 小回归样本，复用已经生成过的 coverage task
或 validation JSONL。它用于低成本验证 pytest coverage task 闭环，不重跑完整
top-N 覆盖率窗口。

环境变量：
  TESTLOOP_PY_REGRESSION_OUTPUT_DIR
                                    每组样本输出 JSONL 的目录。
                                    默认：/tmp/testloop-py-regression-<timestamp>
  TESTLOOP_PY_REGRESSION_CLICK_DIR
                                    Click 项目目录。
                                    默认：/tmp/testloop-click-sample
  TESTLOOP_PY_REGRESSION_CLICK_TASKS_FILE
                                    包含 Click pytest 任务的 JSONL。
                                    默认：/tmp/testloop-click-pytest-top5-regression.jsonl
  TESTLOOP_PY_REGRESSION_CLICK_READY_IDS
                                    默认：pytest-1,pytest-3
  TESTLOOP_PYTEST_COMMAND
                                    默认：uv run python -m pytest {verbose} {coverage} {path}
  TESTLOOP_VALIDATE_PY_STAGE_TIMEOUT_SECONDS
  TESTLOOP_VALIDATE_PY_TASK_TIMEOUT_SECONDS
                                    透传给 validate-py-coverage-top-tasks.sh。
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
validator="$script_dir/validate-py-coverage-top-tasks.sh"

output_dir="${TESTLOOP_PY_REGRESSION_OUTPUT_DIR:-/tmp/testloop-py-regression-$(date +%Y%m%d%H%M%S)}"
click_dir="${TESTLOOP_PY_REGRESSION_CLICK_DIR:-/tmp/testloop-click-sample}"
click_tasks="${TESTLOOP_PY_REGRESSION_CLICK_TASKS_FILE:-/tmp/testloop-click-pytest-top5-regression.jsonl}"
click_ready_ids="${TESTLOOP_PY_REGRESSION_CLICK_READY_IDS:-pytest-1,pytest-3}"
pytest_command="${TESTLOOP_PYTEST_COMMAND:-}"
if [[ -z "$pytest_command" ]]; then
  pytest_command='uv run python -m pytest {verbose} {coverage} {path}'
fi

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
  local output="$output_dir/$name.jsonl"

  require_path dir "$project_dir"
  require_path file "$tasks_file"

  echo "==> $name ids=$ids expected_action=$expected_action"
  (
    cd "$repo_root"
    TESTLOOP_VALIDATE_PY_TASKS_FILE="$tasks_file" \
    TESTLOOP_VALIDATE_PY_TASK_IDS="$ids" \
    TESTLOOP_PYTEST_COMMAND="$pytest_command" \
    "$validator" "$project_dir" "$(task_count "$ids")" "$output"
  )
  assert_sample_output "$output" "$ids" "$expected_action"
  echo "ok  $name output=$output"
}

mkdir -p "$output_dir"

run_sample "click-ready" "$click_dir" "$click_tasks" "$click_ready_ids" "ready"

echo "py_regression_output_dir=$output_dir"
