#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
doc="${repo_root}/docs/agent-response-artifact-contract.md"

assert_contains() {
  needle="$1"
  if ! grep -F -- "$needle" "$doc" >/dev/null 2>&1; then
    echo "expected $doc to contain: $needle" >&2
    exit 1
  fi
}

assert_contains "scripts/run-first-run-ci.sh"
assert_contains "scripts/run-onboarding-ci.sh"
assert_contains "scripts/render-first-run-agent-response.sh"
assert_contains "scripts/render-onboarding-agent-response.sh"
assert_contains "结论："
assert_contains "证据："
assert_contains "下一步："
assert_contains "暂不做："
assert_contains "first_run_agent_next_step=<action>"
assert_contains "agent_next_step=<action>"
assert_contains "failed_section=<section name>"
assert_contains "exit_code=<code>"
assert_contains '先读 `agent-response.txt`'
assert_contains "first-run-context.txt"
assert_contains "./fixtures/first-run-artifacts/user-project-smoke-failed/"
assert_contains "./fixtures/onboarding-artifacts/user-project-smoke-failed/"
assert_contains "./fixtures/agent-response-artifact-manifest.json"
assert_contains "./fixtures/agent-response-artifact-manifest.schema.json"
assert_contains "go run ./examples/agent-response-manifest-demo"
assert_contains "./ci-agent-triage.md"
assert_contains "./client-integration.md"

for path in \
  "${repo_root}/docs/fixtures/first-run-artifacts/user-project-smoke-failed" \
  "${repo_root}/docs/fixtures/onboarding-artifacts/user-project-smoke-failed" \
  "${repo_root}/docs/fixtures/agent-response-artifact-manifest.json" \
  "${repo_root}/docs/fixtures/agent-response-artifact-manifest.schema.json" \
  "${repo_root}/examples/agent-response-manifest-demo/main.go" \
  "${repo_root}/docs/ci-agent-triage.md" \
  "${repo_root}/docs/client-integration.md"
do
  if [ ! -e "$path" ]; then
    echo "missing referenced path: $path" >&2
    exit 1
  fi
done

echo "agent response artifact contract doc test passed"
