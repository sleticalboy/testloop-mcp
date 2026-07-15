#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/validate-java-coverage-top-tasks.sh <java-project-dir> [limit] [output-jsonl]

Runs an opt-in integration helper that:
  1. copies the Java project to a temporary baseline worktree,
  2. runs a coverage command that writes a JaCoCo XML report,
  3. parses top coverage tasks,
  4. validates each selected task in an isolated fresh copy,
  5. writes one validation JSON object per line.

Environment:
  TESTLOOP_VALIDATE_JAVA_TASK_LIMIT Overrides [limit], default 20
  TESTLOOP_VALIDATE_JAVA_OUTPUT     Overrides [output-jsonl]
  TESTLOOP_VALIDATE_JAVA_COVERAGE_COMMAND
                                    Optional shell command that writes a JaCoCo XML file.
                                    Default uses Maven/Gradle detection:
                                    mvn test jacoco:report or gradle test jacocoTestReport.
  TESTLOOP_VALIDATE_JAVA_COVERAGE_FILE
                                    JaCoCo XML path relative to project root. Default checks:
                                    target/site/jacoco/jacoco.xml and
                                    build/reports/jacoco/test/jacocoTestReport.xml.
  TESTLOOP_VALIDATE_JAVA_FILE_FILTER
                                    Optional substring filter for task source files.
  TESTLOOP_VALIDATE_JAVA_LIST_TASKS_ONLY
                                    If true, only writes selected coverage tasks to
                                    output-jsonl and skips per-task validation.
  TESTLOOP_VALIDATE_JAVA_STAGE_TIMEOUT_SECONDS
                                    Optional timeout in seconds used as the default for
                                    baseline coverage and each task validation stage.
  TESTLOOP_VALIDATE_JAVA_BASELINE_TIMEOUT_SECONDS
                                    Optional timeout in seconds for baseline coverage only.
  TESTLOOP_VALIDATE_JAVA_TASK_TIMEOUT_SECONDS
                                    Optional timeout in seconds for each generated task
                                    validation only.
  TESTLOOP_VALIDATE_JAVA_GO_TEST_TIMEOUT
                                    Optional go test timeout for this integration helper.
                                    Default: 30m.
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
limit="${2:-${TESTLOOP_VALIDATE_JAVA_TASK_LIMIT:-20}}"
output="${3:-${TESTLOOP_VALIDATE_JAVA_OUTPUT:-}}"

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

export TESTLOOP_VALIDATE_JAVA_PROJECT_DIR="$project_dir"
export TESTLOOP_VALIDATE_JAVA_TASK_LIMIT="$limit"
if [[ -n "$output" ]]; then
  export TESTLOOP_VALIDATE_JAVA_OUTPUT="$output"
fi

go_test_timeout="${TESTLOOP_VALIDATE_JAVA_GO_TEST_TIMEOUT:-30m}"
go test ./tools -run '^TestValidateJavaCoverageTopTasks$' -count=1 -timeout "$go_test_timeout" -v
