#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

out="${tmp_dir}/agent-decision-demo.out"

(cd "$repo_root" && go run ./examples/agent-decision-demo) > "$out"

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

assert_contains "$out" "1. status=passed action=ready decision=accept"
assert_contains "$out" "2. status=passed action=manual_review_internal decision=manual-review"
assert_contains "$out" "3. status=failed action=apply_fix_suggestions decision=apply-repair"
assert_contains "$out" "4. status=failed action=needs_better_input decision=needs-better-input"
assert_contains "$out" "agent_decisions=accept,manual-review,apply-repair,needs-better-input"

echo "agent decision demo test passed"
