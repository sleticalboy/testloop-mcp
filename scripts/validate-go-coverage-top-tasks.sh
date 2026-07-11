#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/validate-go-coverage-top-tasks.sh <go-project-dir> [limit] [output-jsonl]

Runs an opt-in integration helper that:
  1. copies the Go project to a temporary baseline worktree,
  2. runs go test ./... -coverprofile,
  3. parses top coverage tasks,
  4. validates each selected task in an isolated fresh copy,
  5. writes one validation JSON object per line.

Environment:
  TESTLOOP_VALIDATE_GO_TASK_LIMIT   Overrides [limit], default 50
  TESTLOOP_VALIDATE_GO_OUTPUT       Overrides [output-jsonl]
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
limit="${2:-${TESTLOOP_VALIDATE_GO_TASK_LIMIT:-50}}"
output="${3:-${TESTLOOP_VALIDATE_GO_OUTPUT:-}}"

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

export TESTLOOP_VALIDATE_GO_PROJECT_DIR="$project_dir"
export TESTLOOP_VALIDATE_GO_TASK_LIMIT="$limit"
if [[ -n "$output" ]]; then
  export TESTLOOP_VALIDATE_GO_OUTPUT="$output"
fi

go test ./tools -run '^TestValidateGoCoverageTopTasks$' -count=1 -v
