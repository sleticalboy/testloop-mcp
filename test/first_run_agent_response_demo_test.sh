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

fake_binary="${tmp_dir}/testloop-mcp"
cat >"$fake_binary" <<'SH'
#!/usr/bin/env sh
case "${1:-}" in
  --version)
    echo "testloop-mcp 0.5.8"
    ;;
  *)
    echo "fake testloop-mcp"
    ;;
esac
SH
chmod +x "$fake_binary"

project_dir="${tmp_dir}/project"
artifact_dir="${tmp_dir}/first-run-artifacts"
mkdir -p "$project_dir"

set +e
(
  cd "$repo_root"
  TESTLOOP_MCP_REPO_DIR="$repo_root" \
  TESTLOOP_MCP_COMMAND="$fake_binary" \
  TESTLOOP_MCP_VERSION=v0.5.8 \
  TESTLOOP_FIRST_RUN_PROJECT_DIR="$project_dir" \
  TESTLOOP_FIRST_RUN_OUTPUT_DIR="$artifact_dir" \
  TESTLOOP_REPORT_SKIP_BASIC=true \
  TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true \
  TESTLOOP_REPORT_SKIP_AGENT_DEMO=true \
    bash scripts/run-first-run-ci.sh 'echo project failed from artifact e2e; exit 7'
) > "${tmp_dir}/first-run.out" 2>&1
first_run_code=$?
set -e

if [ "$first_run_code" -ne 1 ]; then
  echo "expected first-run failure exit code 1, got $first_run_code" >&2
  cat "${tmp_dir}/first-run.out" >&2
  exit 1
fi

assert_contains "${tmp_dir}/first-run.out" "first_run_agent_next_step=inspect-user-project"
assert_contains "$artifact_dir/first-run-context.txt" "first_run_agent_next_step=inspect-user-project"
assert_contains "$artifact_dir/verification-report.md" "project failed from artifact e2e"

(cd "$repo_root" && go run ./examples/first-run-agent-response-demo \
  "$artifact_dir/first-run-context.txt" \
  "$artifact_dir/verification-summary.json") > "$out"

assert_contains "$out" "结论：testloop-mcp 接入链路本身是通的，失败发生在用户项目 smoke。"
assert_contains "$out" "- first_run_agent_next_step=inspect-user-project"
assert_contains "$out" "- failed_section=用户项目 smoke"
assert_contains "$out" "- exit_code=7"

echo "first-run Agent response demo test passed"
