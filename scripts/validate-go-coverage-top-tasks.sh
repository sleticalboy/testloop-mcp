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
  TESTLOOP_VALIDATE_GO_FILE_FILTER  Optional substring filter for task source files
  TESTLOOP_VALIDATE_GO_TASK_IDS     Optional comma-separated coverage task IDs to
                                      validate exactly, for example: go-test-1,go-test-5.
  TESTLOOP_VALIDATE_GO_TASKS_FILE   Optional JSONL file containing coverage tasks or
                                      validate_coverage_task outputs. When set, skips
                                      baseline coverage generation and reads tasks from
                                      this file.
  TESTLOOP_VALIDATE_GO_COVERPROFILE Optional per-task coverprofile path used by
                                      validate_coverage_task. Relative paths are
                                      resolved from each isolated task worktree.
                                      Default: testloop-cover.out.
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
tasks_file="${TESTLOOP_VALIDATE_GO_TASKS_FILE:-}"
output="${3:-${TESTLOOP_VALIDATE_GO_OUTPUT:-}}"
if [[ -n "$output" && -e "$output" && -d "$output" ]]; then
  echo "output path must not be a directory: $output" >&2
  exit 1
fi
if [[ -n "$tasks_file" && ! -f "$tasks_file" ]]; then
  echo "tasks file does not exist: $tasks_file" >&2
  exit 1
fi

task_ids_count() {
  printf '%s' "${TESTLOOP_VALIDATE_GO_TASK_IDS:-}" | tr ',' '\n' | awk 'NF {count++} END {print count+0}'
}

tasks_file_count() {
  awk 'NF {count++} END {print count+0}' "$tasks_file"
}

inferred_limit() {
  if [[ -n "${TESTLOOP_VALIDATE_GO_TASK_IDS:-}" ]]; then
    task_ids_count
  elif [[ -n "$tasks_file" ]]; then
    tasks_file_count
  else
    printf '50\n'
  fi
}

if [[ $# -eq 2 && ( -n "${TESTLOOP_VALIDATE_GO_TASK_IDS:-}" || -n "$tasks_file" ) && ! "$2" =~ ^[0-9]+$ ]]; then
  limit="$(inferred_limit)"
  output="$2"
elif [[ $# -ge 2 ]]; then
  limit="$2"
elif [[ -n "${TESTLOOP_VALIDATE_GO_TASK_LIMIT:-}" ]]; then
  limit="$TESTLOOP_VALIDATE_GO_TASK_LIMIT"
else
  limit="$(inferred_limit)"
fi

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
if [[ -n "${TESTLOOP_VALIDATE_GO_COVERPROFILE:-}" ]]; then
  export TESTLOOP_GO_COVERPROFILE="$TESTLOOP_VALIDATE_GO_COVERPROFILE"
fi

go test ./tools -run '^TestValidateGoCoverageTopTasks$' -count=1 -v
