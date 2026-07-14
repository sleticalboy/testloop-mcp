#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/validate-rust-coverage-top-tasks.sh <rust-project-dir> [limit] [output-jsonl]

Runs an opt-in integration helper that:
  1. copies the Rust project to a temporary baseline worktree,
  2. runs a coverage command that writes LCOV data,
  3. parses top coverage tasks,
  4. validates each selected task in an isolated fresh copy,
  5. writes one validation JSON object per line.

Environment:
  TESTLOOP_VALIDATE_RUST_TASK_LIMIT Overrides [limit], default 20
  TESTLOOP_VALIDATE_RUST_OUTPUT     Overrides [output-jsonl]
  TESTLOOP_VALIDATE_RUST_COVERAGE_COMMAND
                                    Optional shell command that writes an LCOV file.
                                    Default:
                                    "cargo tarpaulin --out Lcov --output-dir target/tarpaulin"
  TESTLOOP_VALIDATE_RUST_COVERAGE_FILE
                                    LCOV file path relative to project root, default
                                    "target/tarpaulin/lcov.info".
  TESTLOOP_VALIDATE_RUST_FILE_FILTER
                                    Optional substring filter for task source files.
  TESTLOOP_VALIDATE_RUST_STAGE_TIMEOUT_SECONDS
                                    Optional timeout in seconds used as the default for
                                    baseline coverage and each task validation stage.
  TESTLOOP_VALIDATE_RUST_BASELINE_TIMEOUT_SECONDS
                                    Optional timeout in seconds for baseline coverage only.
  TESTLOOP_VALIDATE_RUST_TASK_TIMEOUT_SECONDS
                                    Optional timeout in seconds for each generated task
                                    validation only.
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -lt 1 || $# -gt 3 ]]; then
  usage
  exit 2
fi

project_dir="$1"
limit="${2:-${TESTLOOP_VALIDATE_RUST_TASK_LIMIT:-20}}"
output="${3:-${TESTLOOP_VALIDATE_RUST_OUTPUT:-}}"

if [[ ! -d "$project_dir" ]]; then
  echo "project directory does not exist: $project_dir" >&2
  exit 1
fi

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

export TESTLOOP_VALIDATE_RUST_PROJECT_DIR="$project_dir"
export TESTLOOP_VALIDATE_RUST_TASK_LIMIT="$limit"
if [[ -n "$output" ]]; then
  export TESTLOOP_VALIDATE_RUST_OUTPUT="$output"
fi

go test ./tools -run '^TestValidateRustCoverageTopTasks$' -count=1 -v
