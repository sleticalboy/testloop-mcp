#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
doc="${repo_root}/docs/real-integration-cases.md"

assert_contains() {
  needle="$1"
  if ! grep -F -- "$needle" "$doc" >/dev/null 2>&1; then
    echo "expected $doc to contain: $needle" >&2
    exit 1
  fi
}

assert_contains "scripts/showcase-agent-onboarding-report.sh"
assert_contains "TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.6"
assert_contains "TESTLOOP_REPORT_PROJECT_DIR"
assert_contains "TESTLOOP_REPORT_PROJECT_COMMAND"
assert_contains "/tmp/testloop-laoxia-server-onboarding-v0.5.4"
assert_contains "/tmp/testloop-laoxia-web-onboarding-v0.5.4"
assert_contains "overall_status=passed"
assert_contains "failed_count=0"
assert_contains "agent_next_step=ready"
assert_contains "fix-installation"
assert_contains "inspect-user-project"

echo "real integration cases doc test passed"
