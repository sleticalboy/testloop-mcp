#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

jsonl="${tmp_dir}/showcase.jsonl"
out="${tmp_dir}/out.txt"

cat > "$jsonl" <<'JSONL'
{"coverage_task":{"id":"task-1","target":"alpha","line_range":"10-12"},"status":"passed","action":"ready","run_result":{"skipped":0}}
{"coverage_task":{"id":"task-2","target":"beta","line_range":"20-21"},"status":"passed","action":"manual_review_internal","run_result":{"skipped":1}}
JSONL

assert_contains() {
  file="$1"
  needle="$2"
  if ! grep -F -- "$needle" "$file" >/dev/null 2>&1; then
    echo "expected $file to contain: $needle" >&2
    echo "--- $file ---" >&2
    cat "$file" >&2
    exit 1
  fi
}

run_expect_code() {
  want="$1"
  output="$2"
  shift 2
  set +e
  "$@" > "$output" 2>&1
  code=$?
  set -e
  if [ "$code" -ne "$want" ]; then
    echo "expected exit code $want, got $code: $*" >&2
    echo "--- $output ---" >&2
    cat "$output" >&2
    exit 1
  fi
}

run_expect_code 0 "$out" python3 "${repo_root}/scripts/summarize-showcase-output.py" "$jsonl" "task-1=ready,task-2=manual_review_internal"
assert_contains "$out" 'showcase_summary={"action_counts": {"manual_review_internal": 1, "ready": 1}'
assert_contains "$out" '"tasks": [{"action": "ready", "id": "task-1", "line_range": "10-12", "skipped": 0, "status": "passed", "target": "alpha"}'
assert_contains "$out" "showcase_expectations=pass"

run_expect_code 1 "$out" python3 "${repo_root}/scripts/summarize-showcase-output.py" "$jsonl" "task-1=manual_review_internal"
assert_contains "$out" "showcase_expectations_failed:"
assert_contains "$out" "task-1: action='ready', expected 'manual_review_internal'"

run_expect_code 1 "$out" python3 "${repo_root}/scripts/summarize-showcase-output.py" "$jsonl" "bad-expectation"
assert_contains "$out" "invalid expectation 'bad-expectation', expected task-id=action"

echo "showcase summary test passed"
