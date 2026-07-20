#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
doc="${repo_root}/docs/mcp-client-contract-tests.md"

assert_contains() {
  needle="$1"
  if ! grep -F -- "$needle" "$doc" >/dev/null 2>&1; then
    echo "expected $doc to contain: $needle" >&2
    exit 1
  fi
}

assert_contains "structuredContent"
assert_contains "content[0].text"
assert_contains "docs/fixtures/validate-coverage-task-ready.json"
assert_contains "sh test/fixtures_index_test.sh"
assert_contains "test/e2e"
assert_contains "CI artifact manifest 回归"
assert_contains "agent-response-artifact-manifest.json"
assert_contains "agent-response-artifact-manifest.schema.json"
assert_contains "verification-summary.schema.json"
assert_contains "dual-project-summary.schema.json"
assert_contains "npx --yes ajv-cli validate"
assert_contains "sections[].signals.action"
assert_contains "go run ./examples/agent-response-manifest-demo"
assert_contains "fallback_order[0]"
assert_contains "first_run_agent_next_step"
assert_contains "agent_next_step"
assert_contains "inspect-user-project"

for path in \
  "${repo_root}/docs/fixtures/agent-response-artifact-manifest.json" \
  "${repo_root}/docs/fixtures/agent-response-artifact-manifest.schema.json" \
  "${repo_root}/docs/fixtures/verification-summary.schema.json" \
  "${repo_root}/docs/fixtures/dual-project-summary.schema.json" \
  "${repo_root}/docs/fixtures/dual-project-summary/laoxia-passed.json" \
  "${repo_root}/examples/agent-response-manifest-demo/main.go" \
  "${repo_root}/test/e2e"
do
  if [ ! -e "$path" ]; then
    echo "missing referenced path: $path" >&2
    exit 1
  fi
done

echo "MCP client contract doc test passed"
