#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
doc="${repo_root}/docs/ci-agent-triage.md"

assert_contains() {
  needle="$1"
  if ! grep -F -- "$needle" "$doc" >/dev/null 2>&1; then
    echo "expected $doc to contain: $needle" >&2
    exit 1
  fi
}

assert_contains "gh run download <run-id> -n testloop-first-run"
assert_contains "快速路径：先读 Agent 回复草稿"
assert_contains "cat /tmp/testloop-first-run-artifacts/agent-response.txt"
assert_contains "agent-response.txt"
assert_contains "agent-decision.txt"
assert_contains "first-run-context.txt"
assert_contains "verification-summary.json"
assert_contains "verification-report.md"
assert_contains "first-run.log"
assert_contains "/tmp/testloop-first-run-failure-triage"
assert_contains "first_run_status=failed"
assert_contains "first_run_failed_count=1"
assert_contains "first_run_agent_next_step=inspect-user-project"
assert_contains "agent_response=/tmp/testloop-first-run-failure-triage/agent-response.txt"
assert_contains "failed_section=用户项目 smoke"
assert_contains "exit_code=7"
assert_contains "testloop intentional project failure"
assert_contains "agent_next_step=fix-installation"
assert_contains "agent_next_step=inspect-user-project"
assert_contains "不要只贴 GitHub Actions 最后一行错误"
assert_contains "足够作为第一段 Agent 回复草稿"
assert_contains "./agent-response-artifact-contract.md"
assert_contains "./onboarding-ci-failure-triage.md"
assert_contains "./first-run-failures.md"

for path in \
  "${repo_root}/docs/onboarding-ci-failure-triage.md" \
  "${repo_root}/docs/first-run-failures.md"
do
  if [ ! -f "$path" ]; then
    echo "missing referenced file: $path" >&2
    exit 1
  fi
done

echo "CI agent triage doc test passed"
