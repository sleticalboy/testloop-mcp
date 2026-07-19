#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
doc="${repo_root}/docs/first-run-agent-response.md"

assert_contains() {
  needle="$1"
  if ! grep -F -- "$needle" "$doc" >/dev/null 2>&1; then
    echo "expected $doc to contain: $needle" >&2
    exit 1
  fi
}

assert_contains "结论："
assert_contains "证据："
assert_contains "下一步："
assert_contains "暂不做："
assert_contains "agent-response.txt"
assert_contains "first_run_agent_next_step"
assert_contains "fix-installation"
assert_contains "inspect-mcp-transport"
assert_contains "inspect-agent-demo"
assert_contains "inspect-user-project"
assert_contains "inspect-showcase"
assert_contains "failed_section=用户项目 smoke"
assert_contains "exit_code=7"
assert_contains "不先修改 testloop-mcp 安装或 MCP transport"
assert_contains "不修改用户项目测试"
assert_contains "agent-decision.txt"
assert_contains "first-run-context.txt"
assert_contains "如果没有 agent-response.txt"
assert_contains "./ci-agent-triage.md"
assert_contains "./first-run-failures.md"
assert_contains "./onboarding-ci-failure-triage.md"

for path in \
  "${repo_root}/docs/ci-agent-triage.md" \
  "${repo_root}/docs/first-run-failures.md" \
  "${repo_root}/docs/onboarding-ci-failure-triage.md"
do
  if [ ! -f "$path" ]; then
    echo "missing referenced file: $path" >&2
    exit 1
  fi
done

echo "first-run Agent response doc test passed"
