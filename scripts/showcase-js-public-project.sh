#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/showcase-js-public-project.sh [output-jsonl]

Runs a small public JS/TS project showcase:
  1. clones unjs/ufo at a pinned commit into a temporary directory,
     or reuses TESTLOOP_SHOWCASE_JS_PROJECT_DIR when set,
  2. installs dependencies with pnpm,
  3. validates selected Vitest coverage task IDs through validate_coverage_task,
  4. writes JSONL validation output and prints a compact summary.

Environment:
  TESTLOOP_SHOWCASE_JS_PROJECT_DIR Existing local checkout to reuse. When set,
                                    clone/fetch/checkout is skipped.
  TESTLOOP_SHOWCASE_JS_REPO       Git repository URL, default https://github.com/unjs/ufo.git
  TESTLOOP_SHOWCASE_JS_REF        Git ref or commit, default f06c800d0c59f2a4a1b9ba65eb6cb61a84419be6
  TESTLOOP_SHOWCASE_JS_TASK_IDS   Comma-separated task IDs, default vitest-1,vitest-2
  TESTLOOP_SHOWCASE_JS_EXPECT_ACTIONS
                                    Expected task actions, default
                                    vitest-1=manual_review_internal,vitest-2=ready.
                                    Set to empty to skip expectation checks.
  TESTLOOP_SHOWCASE_JS_OUTPUT     Output JSONL path, default /tmp/testloop-ufo-showcase.jsonl
  TESTLOOP_SHOWCASE_JS_TIMEOUT    Timeout seconds for baseline and task validation, default 180
  TESTLOOP_SHOWCASE_JS_GIT_TIMEOUT
                                    Timeout seconds for git clone/fetch, default 60.
  TESTLOOP_SHOWCASE_JS_SKIP_INSTALL
                                    Set to true to skip pnpm install for a prepared checkout.
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -gt 1 ]]; then
  usage
  exit 2
fi

if ! command -v pnpm >/dev/null 2>&1; then
  echo "error: pnpm is required for this showcase" >&2
  exit 1
fi

repo="${TESTLOOP_SHOWCASE_JS_REPO:-https://github.com/unjs/ufo.git}"
ref="${TESTLOOP_SHOWCASE_JS_REF:-f06c800d0c59f2a4a1b9ba65eb6cb61a84419be6}"
project_dir="${TESTLOOP_SHOWCASE_JS_PROJECT_DIR:-}"
task_ids="${TESTLOOP_SHOWCASE_JS_TASK_IDS:-vitest-1,vitest-2}"
expect_actions="${TESTLOOP_SHOWCASE_JS_EXPECT_ACTIONS-vitest-1=manual_review_internal,vitest-2=ready}"
timeout_seconds="${TESTLOOP_SHOWCASE_JS_TIMEOUT:-180}"
git_timeout="${TESTLOOP_SHOWCASE_JS_GIT_TIMEOUT:-60}"
skip_install="${TESTLOOP_SHOWCASE_JS_SKIP_INSTALL:-false}"
output="${1:-${TESTLOOP_SHOWCASE_JS_OUTPUT:-/tmp/testloop-ufo-showcase.jsonl}}"

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

run_with_timeout() {
  local seconds="$1"
  shift
  python3 - "$seconds" "$@" <<'PY'
import os
import signal
import subprocess
import sys

seconds = float(sys.argv[1])
cmd = sys.argv[2:]

try:
    proc = subprocess.Popen(cmd, start_new_session=True)
except FileNotFoundError:
    print(f"error: command not found: {cmd[0]}", file=sys.stderr)
    sys.exit(127)

try:
    sys.exit(proc.wait(timeout=seconds))
except subprocess.TimeoutExpired:
    try:
        os.killpg(proc.pid, signal.SIGTERM)
    except ProcessLookupError:
        pass
    try:
        proc.wait(timeout=5)
    except subprocess.TimeoutExpired:
        try:
            os.killpg(proc.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        proc.wait()
    print(f"error: command timed out after {seconds:g}s: {' '.join(cmd)}", file=sys.stderr)
    sys.exit(124)
PY
}

if [[ -n "$project_dir" ]]; then
  [[ -d "$project_dir" ]] || {
    echo "error: TESTLOOP_SHOWCASE_JS_PROJECT_DIR does not exist: $project_dir" >&2
    exit 1
  }
  echo "==> reuse local JS/TS project"
  echo "project_dir=$project_dir"
  if git -C "$project_dir" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "ref=$(git -C "$project_dir" rev-parse HEAD)"
  else
    echo "ref=non-git"
  fi
else
  project_dir="${tmp_dir}/project"
  echo "==> clone public JS/TS project"
  run_with_timeout "$git_timeout" git clone --quiet --depth=1 "$repo" "$project_dir"
  run_with_timeout "$git_timeout" git -C "$project_dir" fetch --quiet --depth=1 origin "$ref"
  git -C "$project_dir" checkout --quiet --detach "$ref"
  echo "repo=$repo"
  echo "ref=$(git -C "$project_dir" rev-parse HEAD)"
fi

if [[ "$skip_install" == "true" ]]; then
  echo "==> install dependencies skipped"
else
  echo "==> install dependencies"
  (cd "$project_dir" && pnpm install --frozen-lockfile)
fi

echo "==> validate selected coverage tasks"
TESTLOOP_VALIDATE_JS_TASK_IDS="$task_ids" \
TESTLOOP_VALIDATE_JS_STAGE_TIMEOUT_SECONDS="$timeout_seconds" \
  "${repo_root}/scripts/validate-js-coverage-top-tasks.sh" "$project_dir" vitest "$output"

echo "==> summarize validation output"
TESTLOOP_SHOWCASE_EXPECT_ACTIONS="$expect_actions" python3 - "$output" <<'PY'
import json
import os
import sys
from collections import Counter

path = sys.argv[1]
expect_actions = os.environ.get("TESTLOOP_SHOWCASE_EXPECT_ACTIONS", "")
rows = []
with open(path, "r", encoding="utf-8") as fh:
    for line in fh:
        line = line.strip()
        if line:
            rows.append(json.loads(line))

status_counts = Counter(row.get("status", "") for row in rows)
action_counts = Counter(row.get("action", "") for row in rows)
tasks = [
    {
        "id": (row.get("coverage_task") or {}).get("id", ""),
        "target": (row.get("coverage_task") or {}).get("target", ""),
        "line_range": (row.get("coverage_task") or {}).get("line_range", ""),
        "status": row.get("status", ""),
        "action": row.get("action", ""),
        "skipped": (row.get("run_result") or {}).get("skipped", 0),
    }
    for row in rows
]
summary = {
    "output": path,
    "total": len(rows),
    "status_counts": dict(status_counts),
    "action_counts": dict(action_counts),
    "tasks": tasks,
}
print("showcase_summary=" + json.dumps(summary, ensure_ascii=False, sort_keys=True))

by_id = {(row.get("coverage_task") or {}).get("id", ""): row for row in rows}
failures = []
for item in [part.strip() for part in expect_actions.split(",") if part.strip()]:
    if "=" not in item:
        failures.append(f"invalid expectation {item!r}, expected task-id=action")
        continue
    task_id, expected_action = [part.strip() for part in item.split("=", 1)]
    row = by_id.get(task_id)
    if not row:
        failures.append(f"missing expected task {task_id!r}")
        continue
    actual_action = row.get("action", "")
    actual_status = row.get("status", "")
    if actual_action != expected_action:
        failures.append(f"{task_id}: action={actual_action!r}, expected {expected_action!r}")
    if actual_status != "passed":
        failures.append(f"{task_id}: status={actual_status!r}, expected 'passed'")

if failures:
    print("showcase_expectations_failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)
if expect_actions:
    print("showcase_expectations=pass")
PY
