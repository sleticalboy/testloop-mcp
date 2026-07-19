#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

out="${tmp_dir}/mcp-client-demo.out"

(cd "$repo_root" && go run ./examples/mcp-client-demo) > "$out"

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

assert_contains "$out" "1. run_tests: status=fail action=apply_fix_suggestions failed=1 suggestions=1"
assert_contains "$out" "2. repair_task:"
assert_contains "$out" "category=expectation_mismatch"
assert_contains "$out" "target=calc_test.go"
assert_contains "$out" "command=go test ./..."
assert_contains "$out" "3. rerun: status=pass action=ready passed=1 coverage=100.0"
assert_contains "$out" "4. parse_coverage: total=100.0 tasks=0"
assert_contains "$out" "agent_next_step=use structuredContent first"

echo "mcp client demo test passed"
