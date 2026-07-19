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

assert_not_contains() {
  file="$1"
  needle="$2"
  if grep -F -- "$needle" "$file" >/dev/null 2>&1; then
    echo "expected $file not to contain: $needle" >&2
    echo "--- $file ---" >&2
    cat "$file" >&2
    exit 1
  fi
}

out="${tmp_dir}/response.out"
err="${tmp_dir}/response.err"

sh "${repo_root}/scripts/render-first-run-agent-response.sh" \
  "${repo_root}/docs/fixtures/first-run-artifacts/user-project-smoke-failed" > "$out"

assert_contains "$out" "结论：testloop-mcp 接入链路本身是通的，失败发生在用户项目 smoke。"
assert_contains "$out" "- first_run_agent_next_step=inspect-user-project"
assert_contains "$out" "- failed_section=用户项目 smoke"
assert_contains "$out" "- exit_code=7"
assert_contains "$out" "打开 verification-report.md 中“用户项目 smoke”这一节"

context_only_dir="${tmp_dir}/context-only"
mkdir -p "$context_only_dir"
cp "${repo_root}/docs/fixtures/first-run/fix-installation.txt" "${context_only_dir}/first-run-context.txt"

sh "${repo_root}/scripts/render-first-run-agent-response.sh" "$context_only_dir" > "$out"
assert_contains "$out" "结论：失败发生在 testloop-mcp 安装或版本门禁，还没进入用户项目测试。"
assert_contains "$out" "- first_run_agent_next_step=fix-installation"
assert_not_contains "$out" "failed_section="
assert_not_contains "$out" "exit_code="

sh "${repo_root}/scripts/render-first-run-agent-response.sh" --help > "$out"
assert_contains "$out" "Usage: scripts/render-first-run-agent-response.sh <first-run-artifact-dir>"

missing_context_dir="${tmp_dir}/missing-context"
mkdir -p "$missing_context_dir"
set +e
sh "${repo_root}/scripts/render-first-run-agent-response.sh" "$missing_context_dir" > "$out" 2> "$err"
status=$?
set -e
if [ "$status" -ne 1 ]; then
  echo "expected missing context exit code 1, got $status" >&2
  exit 1
fi
assert_contains "$err" "missing first-run-context.txt"

echo "render first-run Agent response test passed"
