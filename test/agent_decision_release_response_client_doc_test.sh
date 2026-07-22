#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
doc="${repo_root}/docs/agent-decision-release-response-client.md"

assert_contains() {
  needle="$1"
  if ! grep -F -- "$needle" "$doc" >/dev/null 2>&1; then
    echo "expected $doc to contain: $needle" >&2
    exit 1
  fi
}

assert_contains "scripts/showcase-agent-decision-client-release-response-smoke.sh --json"
assert_contains "scripts/showcase-release-response-adopter.sh --json"
assert_contains "node scripts/validate-release-response-adopter-summary.mjs /path/to/release-response-adopter-summary.json"
assert_contains "node scripts/verify-release-response-adopter-artifact.mjs /path/to/testloop-release-response-adopter-artifacts"
assert_contains "node scripts/validate-release-response-adopter-artifact-verification.mjs /tmp/testloop-release-response-adopter-artifact-verification.json"
assert_contains "release-response-adopter-artifact-verification.schema.json"
assert_contains "release-response-adopter-artifact-verification/passed.json"
assert_contains "release-response-adopter-artifact-verification/missing-summary-consumer.json"
assert_contains "release-response-adopter-summary.schema.json"
assert_contains "release-response-adopter-summary/passed.json"
assert_contains "./agent-decision-release-response-checklist.md"
assert_contains "scripts/showcase-agent-decision-client-release-response-ci.sh --json"
assert_contains "scripts/install-agent-decision-release-response-client.sh /absolute/path/to/client-repo"
assert_contains "scripts/install-agent-decision-release-response-client.sh --summary-json /path/to/release-smoke-summary.json /absolute/path/to/client-repo"
assert_contains "node scripts/validate-agent-decision-release-response-client-install-summary.mjs /path/to/install-summary.json"
assert_contains "agent-decision-release-response-client-install-summary.schema.json"
assert_contains "agent-decision-release-response-client-install-summary/passed.json"
assert_contains "node scripts/export-agent-decision-release-response-client.mjs /tmp/testloop-release-response-client"
assert_contains ".github/workflows/testloop-release-response-contract.yml"
assert_contains "cd /tmp/testloop-release-response-client"
assert_contains "npm test --silent"
assert_contains "TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_SUMMARY_JSON=/path/to/release-smoke-summary.json"
assert_contains 'TESTLOOP_AGENT_DECISION_RELEASE_INSTALLER_URL="file://${PWD}/scripts/install-agent-decision-client-ci-template.sh"'
assert_contains "render-agent-decision-client-release-response.mjs"
assert_contains "assert-release-response.mjs"
assert_contains "testloop-release-smoke-summary.json"
assert_contains "testloop-release-response.json"
assert_contains "agent_next_step=ready"
assert_contains "inspect-release-installer"
assert_contains "inspect-release-client-response"
assert_contains "inspect-release-consumer-response"
assert_contains "inspect-agent-decision-fixtures"
assert_contains "inspect-release-smoke-summary"
assert_contains "evidence.release_ref"
assert_contains "evidence.helper_refs"
assert_contains "evidence.fixture_count"
assert_contains "evidence.decisions"
assert_contains "evidence.agent_next_steps"
assert_contains "failures[]"
assert_contains "sh test/agent_decision_client_release_response_smoke_test.sh"
assert_contains "./client-integration.md"
assert_contains "./mcp-client-contract-tests.md"
assert_contains "./agent-decision-client-ci-template.md"
assert_contains "./fixtures.md"
assert_contains "../examples/release-response-adopter/README.md"

for path in \
  "${repo_root}/scripts/showcase-agent-decision-client-release-response-smoke.sh" \
  "${repo_root}/scripts/showcase-release-response-adopter.sh" \
  "${repo_root}/scripts/validate-release-response-adopter-summary.mjs" \
  "${repo_root}/scripts/verify-release-response-adopter-artifact.mjs" \
  "${repo_root}/scripts/validate-release-response-adopter-artifact-verification.mjs" \
  "${repo_root}/docs/fixtures/release-response-adopter-artifact-verification.schema.json" \
  "${repo_root}/docs/fixtures/release-response-adopter-artifact-verification/passed.json" \
  "${repo_root}/docs/fixtures/release-response-adopter-artifact-verification/missing-summary-consumer.json" \
  "${repo_root}/docs/fixtures/release-response-adopter-summary.schema.json" \
  "${repo_root}/docs/fixtures/release-response-adopter-summary/passed.json" \
  "${repo_root}/examples/release-response-adopter/README.md" \
  "${repo_root}/examples/release-response-adopter/scripts/read-testloop-release-response.mjs" \
  "${repo_root}/docs/agent-decision-release-response-checklist.md" \
  "${repo_root}/scripts/showcase-agent-decision-client-release-response-ci.sh" \
  "${repo_root}/scripts/install-agent-decision-release-response-client.sh" \
  "${repo_root}/scripts/validate-agent-decision-release-response-client-install-summary.mjs" \
  "${repo_root}/scripts/export-agent-decision-release-response-client.mjs" \
  "${repo_root}/scripts/render-agent-decision-client-release-response.mjs" \
  "${repo_root}/test/agent_decision_client_release_response_smoke_test.sh" \
  "${repo_root}/docs/client-integration.md" \
  "${repo_root}/docs/mcp-client-contract-tests.md" \
  "${repo_root}/docs/agent-decision-client-ci-template.md" \
  "${repo_root}/docs/fixtures.md" \
  "${repo_root}/docs/fixtures/agent-decision-release-response-client-install-summary.schema.json" \
  "${repo_root}/docs/fixtures/agent-decision-release-response-client-install-summary/passed.json"
do
  if [ ! -e "$path" ]; then
    echo "missing referenced path: $path" >&2
    exit 1
  fi
done

echo "agent decision release response client doc test passed"
