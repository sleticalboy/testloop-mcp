#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/validate-js-coverage-top-tasks.sh <js-project-dir> [framework] [limit] [output-jsonl]

Runs an opt-in integration helper that:
  1. copies the JS/TS project to a temporary baseline worktree,
  2. runs framework coverage and reads coverage/coverage-final.json,
  3. parses top coverage tasks,
  4. validates each selected task in an isolated fresh copy,
  5. writes one validation JSON object per line.

Framework:
  vitest, jest, or mocha. Default: vitest.

Environment:
  TESTLOOP_VALIDATE_JS_FRAMEWORK    Overrides [framework], default vitest
  TESTLOOP_VALIDATE_JS_TASK_LIMIT   Overrides [limit], default 20
  TESTLOOP_VALIDATE_JS_OUTPUT       Overrides [output-jsonl]
  TESTLOOP_VALIDATE_JS_TEST_ARGS    Extra baseline coverage args, e.g. "tests/foo.test.js"
  TESTLOOP_VALIDATE_JS_COVERAGE_COMMAND
                                      Optional shell command template for baseline coverage.
                                      Use {args} for TESTLOOP_VALIDATE_JS_TEST_ARGS, e.g.
                                      "npx egg-bin cov --timeout 60000 {args}".
  TESTLOOP_JS_TEST_COMMAND           Optional shell command template used by run_tests during
                                      generated task validation. Use {path} for the generated
                                      test file, e.g. "npx egg-bin test --timeout 60000 {path}".
  TESTLOOP_VALIDATE_JS_FILE_FILTER  Optional substring filter for task source files
  TESTLOOP_VALIDATE_JS_TASK_IDS     Optional comma-separated coverage task IDs to
                                      validate exactly, for example: vitest-1,vitest-8.
  TESTLOOP_VALIDATE_JS_TASKS_FILE   Optional JSONL file containing coverage tasks or
                                      validate_coverage_task outputs. When set, skips
                                      baseline coverage generation and reads tasks from
                                      this file.
  TESTLOOP_VALIDATE_JS_STAGE_TIMEOUT_SECONDS
                                      Optional timeout in seconds used as the default for
                                      baseline coverage and each task validation stage.
  TESTLOOP_VALIDATE_JS_BASELINE_TIMEOUT_SECONDS
                                      Optional timeout in seconds for baseline coverage only.
  TESTLOOP_VALIDATE_JS_TASK_TIMEOUT_SECONDS
                                      Optional timeout in seconds for each generated task
                                      validation only.
  TESTLOOP_VALIDATE_JS_EXTRA_SYMLINKS
                                      Optional comma-separated src:dst mappings.
                                      src is relative to <js-project-dir> unless absolute;
                                      dst is relative to the isolated project copy and may
                                      include ../ segments for monorepo parent resources.
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -lt 1 || $# -gt 4 ]]; then
  usage
  exit 2
fi

project_dir="$1"
framework="${2:-${TESTLOOP_VALIDATE_JS_FRAMEWORK:-vitest}}"
tasks_file="${TESTLOOP_VALIDATE_JS_TASKS_FILE:-}"
output="${4:-${TESTLOOP_VALIDATE_JS_OUTPUT:-}}"
if [[ -n "$tasks_file" && ! -f "$tasks_file" ]]; then
  echo "tasks file does not exist: $tasks_file" >&2
  exit 1
fi

task_ids_count() {
  printf '%s' "${TESTLOOP_VALIDATE_JS_TASK_IDS:-}" | tr ',' '\n' | awk 'NF {count++} END {print count+0}'
}

tasks_file_count() {
  awk 'NF {count++} END {print count+0}' "$tasks_file"
}

inferred_limit() {
  if [[ -n "${TESTLOOP_VALIDATE_JS_TASK_IDS:-}" ]]; then
    task_ids_count
  elif [[ -n "$tasks_file" ]]; then
    tasks_file_count
  else
    printf '20\n'
  fi
}

if [[ $# -eq 3 && ( -n "${TESTLOOP_VALIDATE_JS_TASK_IDS:-}" || -n "$tasks_file" ) && ! "$3" =~ ^[0-9]+$ ]]; then
  limit="$(inferred_limit)"
  output="$3"
elif [[ $# -ge 3 ]]; then
  limit="$3"
elif [[ -n "${TESTLOOP_VALIDATE_JS_TASK_LIMIT:-}" ]]; then
  limit="$TESTLOOP_VALIDATE_JS_TASK_LIMIT"
else
  limit="$(inferred_limit)"
fi

if [[ ! -d "$project_dir" ]]; then
  echo "project directory does not exist: $project_dir" >&2
  exit 1
fi

case "$framework" in
  vitest|jest|mocha)
    ;;
  *)
    echo "unsupported framework: $framework (expected vitest, jest, or mocha)" >&2
    exit 1
    ;;
esac

case "$limit" in
  ''|*[!0-9]*)
    echo "limit must be a positive integer: $limit" >&2
    exit 1
    ;;
esac
if [[ "$limit" -le 0 ]]; then
  echo "limit must be greater than 0: $limit" >&2
  exit 1
fi

export TESTLOOP_VALIDATE_JS_PROJECT_DIR="$project_dir"
export TESTLOOP_VALIDATE_JS_FRAMEWORK="$framework"
export TESTLOOP_VALIDATE_JS_TASK_LIMIT="$limit"
if [[ -n "$output" ]]; then
  export TESTLOOP_VALIDATE_JS_OUTPUT="$output"
fi

go test ./tools -run '^TestValidateJSCoverageTopTasks$' -count=1 -v
