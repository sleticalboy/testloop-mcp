#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/showcase-js-public-project.sh [output-jsonl]

Runs a small public JS/TS project showcase:
  1. clones unjs/ufo at a pinned commit into a temporary directory,
  2. installs dependencies with pnpm,
  3. validates selected Vitest coverage task IDs through validate_coverage_task,
  4. writes JSONL validation output and prints a compact summary.

Environment:
  TESTLOOP_SHOWCASE_JS_REPO       Git repository URL, default https://github.com/unjs/ufo.git
  TESTLOOP_SHOWCASE_JS_REF        Git ref or commit, default f06c800d0c59f2a4a1b9ba65eb6cb61a84419be6
  TESTLOOP_SHOWCASE_JS_TASK_IDS   Comma-separated task IDs, default vitest-1,vitest-2
  TESTLOOP_SHOWCASE_JS_OUTPUT     Output JSONL path, default /tmp/testloop-ufo-showcase.jsonl
  TESTLOOP_SHOWCASE_JS_TIMEOUT    Timeout seconds for baseline and task validation, default 180
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
task_ids="${TESTLOOP_SHOWCASE_JS_TASK_IDS:-vitest-1,vitest-2}"
timeout_seconds="${TESTLOOP_SHOWCASE_JS_TIMEOUT:-180}"
output="${1:-${TESTLOOP_SHOWCASE_JS_OUTPUT:-/tmp/testloop-ufo-showcase.jsonl}}"

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

project_dir="${tmp_dir}/project"

echo "==> clone public JS/TS project"
git clone --quiet --depth=1 "$repo" "$project_dir"
git -C "$project_dir" fetch --quiet --depth=1 origin "$ref"
git -C "$project_dir" checkout --quiet --detach "$ref"
echo "repo=$repo"
echo "ref=$(git -C "$project_dir" rev-parse HEAD)"

echo "==> install dependencies"
(cd "$project_dir" && pnpm install --frozen-lockfile)

echo "==> validate selected coverage tasks"
TESTLOOP_VALIDATE_JS_TASK_IDS="$task_ids" \
TESTLOOP_VALIDATE_JS_STAGE_TIMEOUT_SECONDS="$timeout_seconds" \
  "${repo_root}/scripts/validate-js-coverage-top-tasks.sh" "$project_dir" vitest "$output"

echo "==> summarize validation output"
python3 - "$output" <<'PY'
import json
import sys
from collections import Counter

path = sys.argv[1]
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
PY
