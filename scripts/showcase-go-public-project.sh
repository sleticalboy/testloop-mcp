#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/showcase-go-public-project.sh [output-jsonl]

Runs a small public Go project showcase:
  1. clones google/uuid at a pinned commit into a temporary directory,
  2. validates selected coverage task IDs through validate_coverage_task,
  3. writes JSONL validation output and prints a compact summary.

Environment:
  TESTLOOP_SHOWCASE_GO_REPO       Git repository URL, default https://github.com/google/uuid.git
  TESTLOOP_SHOWCASE_GO_REF        Git ref or commit, default 2d3c2a9cc518326daf99a383f07c4d3c44317e4d
  TESTLOOP_SHOWCASE_GO_TASK_IDS   Comma-separated task IDs, default go-test-1
  TESTLOOP_SHOWCASE_GO_EXPECT_ACTIONS
                                    Expected task actions, default go-test-1=ready.
                                    Set to empty to skip expectation checks.
  TESTLOOP_SHOWCASE_GO_OUTPUT     Output JSONL path, default /tmp/testloop-google-uuid-showcase.jsonl
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

repo="${TESTLOOP_SHOWCASE_GO_REPO:-https://github.com/google/uuid.git}"
ref="${TESTLOOP_SHOWCASE_GO_REF:-2d3c2a9cc518326daf99a383f07c4d3c44317e4d}"
task_ids="${TESTLOOP_SHOWCASE_GO_TASK_IDS:-go-test-1}"
expect_actions="${TESTLOOP_SHOWCASE_GO_EXPECT_ACTIONS-go-test-1=ready}"
output="${1:-${TESTLOOP_SHOWCASE_GO_OUTPUT:-/tmp/testloop-google-uuid-showcase.jsonl}}"

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

project_dir="${tmp_dir}/project"

echo "==> clone public Go project"
git clone --quiet --depth=1 "$repo" "$project_dir"
git -C "$project_dir" fetch --quiet --depth=1 origin "$ref"
git -C "$project_dir" checkout --quiet --detach "$ref"
echo "repo=$repo"
echo "ref=$(git -C "$project_dir" rev-parse HEAD)"

echo "==> validate selected coverage tasks"
TESTLOOP_VALIDATE_GO_TASK_IDS="$task_ids" \
  "${repo_root}/scripts/validate-go-coverage-top-tasks.sh" "$project_dir" "$output"

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
