#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
用法：scripts/validate-java-regression-samples.sh

运行一组固定的 Java/JUnit 小回归样本，复用已经生成过的 coverage task
或 validation JSONL。它用于低成本守住 coverage_task -> generate_tests ->
run_tests -> JaCoCo 目标行命中校验闭环，避免每次都重跑完整 top-N 覆盖率窗口。

环境变量：
  TESTLOOP_JAVA_REGRESSION_OUTPUT_DIR
                                    每组样本输出 JSONL 的目录。
                                    默认：/tmp/testloop-java-regression-<timestamp>
  TESTLOOP_JAVA_REGRESSION_LANG_DIR
                                    Apache Commons Lang 项目目录。
                                    默认：/tmp/testloop-commons-lang
  TESTLOOP_JAVA_REGRESSION_CODEC_DIR
                                    Apache Commons Codec 项目目录。
                                    默认：/tmp/testloop-commons-codec
  TESTLOOP_JAVA_REGRESSION_ROCKETMQ_DIR
                                    RocketMQ Java client 项目目录。
                                    默认：/Users/binlee/code/free-works/haoying/rocketmq-clients/java
  TESTLOOP_JAVA_REGRESSION_LANG_READY_TASKS_FILE
                                    包含 Commons Lang ready 任务的 JSONL。
                                    默认：testdata/java-commons-lang/ready-hit-tasks.jsonl
  TESTLOOP_JAVA_REGRESSION_LANG_MANUAL_TASKS_FILE
                                    包含 Commons Lang 手审任务的 JSONL。
                                    默认：testdata/java-commons-lang/manual-internal-tasks.jsonl
  TESTLOOP_JAVA_REGRESSION_CODEC_UNREACHABLE_TASKS_FILE
                                    包含 Commons Codec 不可达任务的 JSONL。
                                    默认：testdata/java-commons-codec/unreachable-tasks.jsonl
  TESTLOOP_JAVA_REGRESSION_ROCKETMQ_STATUSCHECKER_TASKS_FILE
                                    包含 RocketMQ StatusChecker ready 任务的 JSONL。
                                    默认：testdata/java-rocketmq-statuschecker/statuschecker-tasks.jsonl
  TESTLOOP_JAVA_REGRESSION_READY_IDS
                                    默认：junit-44,junit-50
  TESTLOOP_JAVA_REGRESSION_MANUAL_IDS
                                    默认：junit-52
  TESTLOOP_JAVA_REGRESSION_UNREACHABLE_IDS
                                    默认：junit-130
  TESTLOOP_JAVA_REGRESSION_ROCKETMQ_STATUSCHECKER_IDS
                                    默认：junit-272,junit-273,junit-418,junit-826
  TESTLOOP_JAVA_REGRESSION_ROCKETMQ_COVERAGE_COMMAND
                                    RocketMQ StatusChecker 复验用 Maven 命令。
                                    默认只跑 client 模块的 StatusCheckerTest 并生成 JaCoCo report。
  TESTLOOP_JAVA_REGRESSION_ROCKETMQ_COVERAGE_FILE
                                    RocketMQ JaCoCo XML 相对路径。
                                    默认：client/target/site/jacoco/jacoco.xml
  TESTLOOP_VALIDATE_JAVA_STAGE_TIMEOUT_SECONDS
  TESTLOOP_VALIDATE_JAVA_TASK_TIMEOUT_SECONDS
  TESTLOOP_VALIDATE_JAVA_GO_TEST_TIMEOUT
                                    透传给 validate-java-coverage-top-tasks.sh。
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
validator="$script_dir/validate-java-coverage-top-tasks.sh"

output_dir="${TESTLOOP_JAVA_REGRESSION_OUTPUT_DIR:-/tmp/testloop-java-regression-$(date +%Y%m%d%H%M%S)}"
lang_dir="${TESTLOOP_JAVA_REGRESSION_LANG_DIR:-/tmp/testloop-commons-lang}"
codec_dir="${TESTLOOP_JAVA_REGRESSION_CODEC_DIR:-/tmp/testloop-commons-codec}"
rocketmq_dir="${TESTLOOP_JAVA_REGRESSION_ROCKETMQ_DIR:-/Users/binlee/code/free-works/haoying/rocketmq-clients/java}"
lang_ready_tasks="${TESTLOOP_JAVA_REGRESSION_LANG_READY_TASKS_FILE:-$repo_root/testdata/java-commons-lang/ready-hit-tasks.jsonl}"
lang_manual_tasks="${TESTLOOP_JAVA_REGRESSION_LANG_MANUAL_TASKS_FILE:-$repo_root/testdata/java-commons-lang/manual-internal-tasks.jsonl}"
codec_unreachable_tasks="${TESTLOOP_JAVA_REGRESSION_CODEC_UNREACHABLE_TASKS_FILE:-$repo_root/testdata/java-commons-codec/unreachable-tasks.jsonl}"
rocketmq_statuschecker_tasks="${TESTLOOP_JAVA_REGRESSION_ROCKETMQ_STATUSCHECKER_TASKS_FILE:-$repo_root/testdata/java-rocketmq-statuschecker/statuschecker-tasks.jsonl}"
rocketmq_coverage_command="${TESTLOOP_JAVA_REGRESSION_ROCKETMQ_COVERAGE_COMMAND:-mvn -pl client -am -Dtest=StatusCheckerTest -DfailIfNoTests=false test jacoco:report -Dcheckstyle.skip -Dspotbugs.skip -DskipITs}"
rocketmq_coverage_file="${TESTLOOP_JAVA_REGRESSION_ROCKETMQ_COVERAGE_FILE:-client/target/site/jacoco/jacoco.xml}"

ready_ids="${TESTLOOP_JAVA_REGRESSION_READY_IDS:-junit-44,junit-50}"
manual_ids="${TESTLOOP_JAVA_REGRESSION_MANUAL_IDS:-junit-52}"
unreachable_ids="${TESTLOOP_JAVA_REGRESSION_UNREACHABLE_IDS:-junit-130}"
rocketmq_statuschecker_ids="${TESTLOOP_JAVA_REGRESSION_ROCKETMQ_STATUSCHECKER_IDS:-junit-272,junit-273,junit-418,junit-826}"

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
  local require_hit="$4"

  python3 - "$output" "$expected_ids" "$expected_action" "$require_hit" <<'PY'
import json
import sys

path, raw_ids, expected_action, require_hit = sys.argv[1:]
expected_ids = [item.strip() for item in raw_ids.split(",") if item.strip()]
require_hit = require_hit == "true"

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
    task = row.get("coverage_task", {})
    task_id = task.get("id", "")
    status = row.get("status")
    action = row.get("action")
    if status != "passed":
        raise SystemExit(f"{path}: {task_id} status={status}, want=passed")
    if action != expected_action:
        raise SystemExit(f"{path}: {task_id} action={action}, want={expected_action}")
    if require_hit:
        metadata = row.get("metadata") or {}
        if metadata.get("coverage_target_hit") is not True:
            raise SystemExit(f"{path}: {task_id} coverage_target_hit is not true")
        if not metadata.get("coverage_hit_lines"):
            raise SystemExit(f"{path}: {task_id} coverage_hit_lines is empty")
PY
}

run_sample() {
  local name="$1"
  local project_dir="$2"
  local tasks_file="$3"
  local ids="$4"
  local expected_action="$5"
  local require_hit="$6"
  local output="$output_dir/$name.jsonl"

  require_path dir "$project_dir"
  require_path file "$tasks_file"

  echo "==> $name ids=$ids expected_action=$expected_action"
  (
    cd "$repo_root"
    TESTLOOP_VALIDATE_JAVA_TASKS_FILE="$tasks_file" \
    TESTLOOP_VALIDATE_JAVA_TASK_IDS="$ids" \
    "$validator" "$project_dir" "$(task_count "$ids")" "$output"
  )
  assert_sample_output "$output" "$ids" "$expected_action" "$require_hit"
  echo "ok  $name output=$output"
}

run_rocketmq_statuschecker_sample() {
  local output="$output_dir/rocketmq-statuschecker-ready-hit.jsonl"

  require_path dir "$rocketmq_dir"
  require_path file "$rocketmq_statuschecker_tasks"

  echo "==> rocketmq-statuschecker-ready-hit ids=$rocketmq_statuschecker_ids expected_action=ready"
  (
    cd "$repo_root"
    TESTLOOP_VALIDATE_JAVA_COVERAGE_COMMAND="$rocketmq_coverage_command" \
    TESTLOOP_VALIDATE_JAVA_COVERAGE_FILE="$rocketmq_coverage_file" \
    TESTLOOP_VALIDATE_JAVA_FILE_FILTER="StatusChecker.java" \
    TESTLOOP_VALIDATE_JAVA_TASKS_FILE="$rocketmq_statuschecker_tasks" \
    TESTLOOP_VALIDATE_JAVA_TASK_IDS="$rocketmq_statuschecker_ids" \
    "$validator" "$rocketmq_dir" "$(task_count "$rocketmq_statuschecker_ids")" "$output"
  )
  assert_sample_output "$output" "$rocketmq_statuschecker_ids" "ready" "true"
  echo "ok  rocketmq-statuschecker-ready-hit output=$output"
}

mkdir -p "$output_dir"

run_sample "commons-lang-ready-hit" "$lang_dir" "$lang_ready_tasks" "$ready_ids" "ready" "true"
run_sample "commons-codec-unreachable" "$codec_dir" "$codec_unreachable_tasks" "$unreachable_ids" "manual_review_unreachable" "false"
run_sample "commons-lang-manual-internal" "$lang_dir" "$lang_manual_tasks" "$manual_ids" "manual_review_internal" "false"
run_rocketmq_statuschecker_sample

echo "java_regression_output_dir=$output_dir"
