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
assert_contains "docs/fixtures/agent-decision-fixtures.json"
assert_contains "docs/fixtures/agent-decision-fixtures.schema.json"
assert_contains "manifest.fixtures"
assert_contains "item.expected_decision"
assert_contains "action starts with manual_review_"
assert_contains "docs/fixtures/validate-coverage-task-ready.json"
assert_contains "docs/fixtures/real-project-agent-loop/laoxia-server-go-utils.json"
assert_contains "docs/fixtures/real-project-agent-loop/mcp-hub-vitest-repair.json"
assert_contains "docs/fixtures/real-project-agent-loop/haoy-apk-station-py-environment.json"
assert_contains "docs/fixtures/real-project-agent-loop/haoy-apk-station-py-external-service.json"
assert_contains "regression_note"
assert_contains "redaction_note"
assert_contains "sh test/fixtures_index_test.sh"
assert_contains "sh test/agent_decision_fixtures_manifest_test.sh"
assert_contains "test/e2e"
assert_contains "CI artifact manifest 回归"
assert_contains "agent-response-artifact-manifest.json"
assert_contains "agent-response-artifact-manifest.schema.json"
assert_contains "verification-summary.schema.json"
assert_contains "dual-project-summary.schema.json"
assert_contains "npx --yes ajv-cli validate"
assert_contains "sections[].signals.action"
assert_contains "go run ./examples/agent-response-manifest-demo"
assert_contains "sh scripts/verify-agent-artifact.sh"
assert_contains "--json"
assert_contains "node scripts/validate-agent-decision-fixtures.mjs --json"
assert_contains "node scripts/export-agent-decision-fixtures.mjs /tmp/testloop-agent-decision-fixtures"
assert_contains "最小决策 fixture 包"
assert_contains "package.json"
assert_contains "npm test --silent"
assert_contains "fixture_count"
assert_contains "decisions[]"
assert_contains "fixtures[]"
assert_contains "failures[]"
assert_contains "agent_artifact_status=passed"
assert_contains "agent_artifact_manifest_status=passed"
assert_contains "response_action=inspect-user-project"
assert_contains "artifact_count=2"
assert_contains "artifacts[].section_signals"
assert_contains "fallback_order[0]"
assert_contains "summary_schema=verification-summary.schema.json"
assert_contains "first_run_agent_next_step"
assert_contains "agent_next_step"
assert_contains "inspect-user-project"

for path in \
  "${repo_root}/docs/fixtures/agent-response-artifact-manifest.json" \
  "${repo_root}/docs/fixtures/agent-response-artifact-manifest.schema.json" \
  "${repo_root}/docs/fixtures/agent-decision-fixtures.json" \
  "${repo_root}/docs/fixtures/agent-decision-fixtures.schema.json" \
  "${repo_root}/docs/fixtures/verification-summary.schema.json" \
  "${repo_root}/docs/fixtures/dual-project-summary.schema.json" \
  "${repo_root}/docs/fixtures/dual-project-summary/laoxia-passed.json" \
  "${repo_root}/docs/fixtures/real-project-agent-loop/laoxia-server-go-utils.json" \
  "${repo_root}/docs/fixtures/real-project-agent-loop/mcp-hub-vitest-repair.json" \
  "${repo_root}/docs/fixtures/real-project-agent-loop/haoy-apk-station-py-environment.json" \
  "${repo_root}/docs/fixtures/real-project-agent-loop/haoy-apk-station-py-external-service.json" \
  "${repo_root}/examples/agent-response-manifest-demo/main.go" \
  "${repo_root}/scripts/verify-agent-artifact.sh" \
  "${repo_root}/test/e2e"
do
  if [ ! -e "$path" ]; then
    echo "missing referenced path: $path" >&2
    exit 1
  fi
done

echo "MCP client contract doc test passed"
