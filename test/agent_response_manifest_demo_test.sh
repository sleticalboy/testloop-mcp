#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

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

out="${tmp_dir}/manifest-demo.out"
err="${tmp_dir}/manifest-demo.err"

(cd "$repo_root" && go run ./examples/agent-response-manifest-demo \
  docs/fixtures/agent-response-artifact-manifest.json) > "$out"

assert_contains "$out" "manifest_schema_version=1"
assert_contains "$out" "summary_schema=./verification-summary.schema.json"
assert_contains "$out" "artifact_count=2"
assert_contains "$out" "kind=first-run action_field=first_run_agent_next_step expected_action=inspect-user-project"
assert_contains "$out" "kind=onboarding action_field=agent_next_step expected_action=inspect-user-project"
assert_contains "$out" "decision_action=inspect-user-project"
assert_contains "$out" "summary_validated=verification-summary.json"
assert_contains "$out" "local_summary_schema=verification-summary.schema.json"
assert_contains "$out" "expected_section_signals=独立 CLI 生成动作 smoke:manual_review"
assert_contains "$out" "required_response_fields=first_run_agent_next_step,failed_section,exit_code,first_run_report"
assert_contains "$out" "required_response_fields=agent_next_step,overall_status,failed_count,failed_section,exit_code,markdown_report"
assert_contains "$out" "fallback_order=agent-response.txt > agent-decision.txt"

set +e
(cd "$repo_root" && go run ./examples/agent-response-manifest-demo) > "$out" 2> "$err"
status=$?
set -e
if [ "$status" -ne 1 ]; then
  echo "expected missing argument exit code 1, got $status" >&2
  exit 1
fi
assert_contains "$err" "usage: go run ./examples/agent-response-manifest-demo <agent-response-artifact-manifest.json>"

echo "agent response manifest demo test passed"
