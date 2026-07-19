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

(cd "$repo_root" && go run ./examples/first-run-agent-response-demo \
  docs/fixtures/first-run/inspect-user-project.txt \
  docs/fixtures/verification-summary/user-project-failed.json) > "$out"

assert_contains "$out" "结论：testloop-mcp 接入链路本身是通的，失败发生在用户项目 smoke。"
assert_contains "$out" "证据："
assert_contains "$out" "- first_run_agent_next_step=inspect-user-project"
assert_contains "$out" "- failed_section=用户项目 smoke"
assert_contains "$out" "- exit_code=7"
assert_contains "$out" "- first_run_report=/tmp/testloop-user-project-failed-report.md"
assert_contains "$out" "下一步："
assert_contains "$out" "打开 verification-report.md 中“用户项目 smoke”这一节"
assert_contains "$out" "暂不做："
assert_contains "$out" "不先修改 testloop-mcp 安装或 MCP transport"

(cd "$repo_root" && go run ./examples/first-run-agent-response-demo \
  docs/fixtures/first-run/fix-installation.txt) > "$out"

assert_contains "$out" "结论：失败发生在 testloop-mcp 安装或版本门禁，还没进入用户项目测试。"
assert_contains "$out" "- first_run_agent_next_step=fix-installation"
assert_contains "$out" "不修改用户项目测试"

echo "first-run Agent response demo test passed"
