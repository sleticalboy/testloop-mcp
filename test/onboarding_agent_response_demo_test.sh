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

out="${tmp_dir}/response.out"
err="${tmp_dir}/response.err"

(cd "$repo_root" && go run ./examples/onboarding-agent-response-demo \
  docs/fixtures/verification-summary/user-project-failed.json) > "$out"
assert_contains "$out" "结论：testloop-mcp onboarding 链路本身是通的，失败发生在用户项目 smoke。"
assert_contains "$out" "- agent_next_step=inspect-user-project"
assert_contains "$out" "- overall_status=failed"
assert_contains "$out" "- failed_count=1"
assert_contains "$out" "- failed_section=用户项目 smoke"
assert_contains "$out" "- exit_code=7"
assert_contains "$out" "- markdown_report=/tmp/testloop-user-project-failed-report.md"

(cd "$repo_root" && go run ./examples/onboarding-agent-response-demo \
  docs/fixtures/verification-summary/install-failed.json) > "$out"
assert_contains "$out" "结论：失败发生在 testloop-mcp 安装或版本门禁，还没进入用户项目 smoke。"
assert_contains "$out" "- agent_next_step=fix-installation"
assert_contains "$out" "不修改用户项目测试"

passed_dir="${tmp_dir}/passed"
mkdir -p "$passed_dir"
cat > "${passed_dir}/verification-summary.json" <<'JSON'
{
  "overall_status": "passed",
  "failed_count": 0,
  "markdown_report": "/tmp/testloop-onboarding-report.md",
  "sections": []
}
JSON

sh "${repo_root}/scripts/render-onboarding-agent-response.sh" "$passed_dir" > "$out"
assert_contains "$out" "结论：testloop-mcp onboarding 链路通过，可以继续真实生成、修复或覆盖率闭环。"
assert_contains "$out" "- agent_next_step=ready"
assert_contains "$out" "- markdown_report=/tmp/testloop-onboarding-report.md"

sh "${repo_root}/scripts/render-onboarding-agent-response.sh" --help > "$out"
assert_contains "$out" "Usage: scripts/render-onboarding-agent-response.sh <onboarding-artifact-dir>"

missing_summary_dir="${tmp_dir}/missing-summary"
mkdir -p "$missing_summary_dir"
set +e
sh "${repo_root}/scripts/render-onboarding-agent-response.sh" "$missing_summary_dir" > "$out" 2> "$err"
status=$?
set -e
if [ "$status" -ne 1 ]; then
  echo "expected missing summary exit code 1, got $status" >&2
  exit 1
fi
assert_contains "$err" "missing verification-summary.json"

echo "onboarding Agent response demo test passed"
