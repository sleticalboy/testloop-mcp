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
  TESTLOOP_VALIDATE_JS_FILE_FILTER  Optional substring filter for task source files
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
limit="${3:-${TESTLOOP_VALIDATE_JS_TASK_LIMIT:-20}}"
output="${4:-${TESTLOOP_VALIDATE_JS_OUTPUT:-}}"

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
