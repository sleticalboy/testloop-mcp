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
assert_contains "agent-decision.txt"
assert_contains "first-run-context.txt"
assert_contains "verification-summary.json"
assert_contains "verification-report.md"
assert_contains "first-run.log"
assert_contains "agent_next_step=fix-installation"
assert_contains "agent_next_step=inspect-user-project"
assert_contains "不要只贴 GitHub Actions 最后一行错误"
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
