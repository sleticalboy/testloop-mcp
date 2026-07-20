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

assert_contains "$out" "1. fixture=validate-coverage-task-ready.json status=passed action=ready decision=accept"
assert_contains "$out" "2. fixture=real-project-agent-loop/laoxia-server-go-utils.json status=passed action=ready decision=accept"
assert_contains "$out" "3. fixture=real-project-agent-loop/mcp-hub-vitest-repair.json status=passed action=ready decision=accept"
assert_contains "$out" "4. fixture=validate-coverage-task-manual-review-internal.json status=passed action=manual_review_internal decision=manual-review"
assert_contains "$out" "5. fixture=real-project-agent-loop/haoy-apk-station-py-environment.json status=passed action=manual_review_environment decision=manual-review"
assert_contains "$out" "6. fixture=real-project-agent-loop/haoy-apk-station-py-external-service.json status=failed action=manual_review_external_service decision=manual-review"
assert_contains "$out" "7. fixture=validate-coverage-task-apply-fix-suggestions.json status=failed action=apply_fix_suggestions decision=apply-repair"
assert_contains "$out" "8. fixture=validate-coverage-task-needs-better-input.json status=failed action=needs_better_input decision=needs-better-input"
assert_contains "$out" "agent_decisions=accept,accept,accept,manual-review,manual-review,manual-review,apply-repair,needs-better-input"

echo "agent decision demo test passed"
