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

run_expect_code() {
  want="$1"
  output="$2"
  shift 2
  set +e
  "$@" > "$output" 2>&1
  code=$?
  set -e
  if [ "$code" -ne "$want" ]; then
    echo "expected exit code $want, got $code: $*" >&2
    echo "--- $output ---" >&2
    cat "$output" >&2
    exit 1
  fi
}

fake_binary="${tmp_dir}/testloop-mcp"
cat >"$fake_binary" <<'SH'
#!/usr/bin/env sh
case "${1:-}" in
  --version)
    echo "testloop-mcp 0.5.15"
    ;;
  *)
    echo "fake testloop-mcp"
    ;;
esac
SH
chmod +x "$fake_binary"

out="${tmp_dir}/first-run.out"
output_dir="${tmp_dir}/first-run"
run_expect_code 0 "$out" env \
  TESTLOOP_FIRST_RUN_OUTPUT_DIR="$output_dir" \
  TESTLOOP_FIRST_RUN_EXPECT_VERSION=0.5.15 \
  TESTLOOP_REPORT_SKIP_BASIC=true \
  TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true \
  TESTLOOP_REPORT_SKIP_AGENT_DEMO=true \
  bash "${repo_root}/scripts/doctor-first-run.sh" "$fake_binary"

assert_contains "$out" "first_run_status=passed"
assert_contains "$out" "first_run_failed_count=0"
assert_contains "$out" "first_run_agent_next_step=ready"
assert_contains "$out" "first_run_report=$output_dir/verification-report.md"
assert_contains "$out" "first_run_summary_json=$output_dir/verification-summary.json"
assert_contains "$out" "first_run_decision=$output_dir/agent-decision.txt"
assert_contains "$out" "first_run_context=$output_dir/first-run-context.txt"
assert_contains "$out" "first_run_log=$output_dir/first-run.log"
assert_contains "$out" "first_run_next=install path is ready"
assert_contains "$output_dir/verification-summary.json" '"overall_status": "passed"'
assert_contains "$output_dir/agent-decision.txt" "agent_next_step=ready"
assert_contains "$output_dir/first-run-context.txt" "testloop-mcp first-run diagnostic context"
assert_contains "$output_dir/first-run-context.txt" "first_run_agent_next_step=ready"
assert_contains "$output_dir/first-run-context.txt" "Suggested prompt:"
assert_contains "$output_dir/first-run.log" "onboarding_report=$output_dir/verification-report.md"

dir_binary_out="${tmp_dir}/dir-binary.out"
run_expect_code 1 "$dir_binary_out" env \
  TESTLOOP_FIRST_RUN_OUTPUT_DIR="${tmp_dir}/dir-binary" \
  bash "${repo_root}/scripts/doctor-first-run.sh" "$repo_root"
assert_contains "${tmp_dir}/dir-binary/first-run.log" "binary must be an executable file"

output_file="${tmp_dir}/output-file"
printf 'not a directory\n' > "$output_file"
run_expect_code 1 "$out" env \
  TESTLOOP_FIRST_RUN_OUTPUT_DIR="$output_file" \
  bash "${repo_root}/scripts/doctor-first-run.sh" "$fake_binary"
assert_contains "$out" "output path must be a directory"

project_dir="${tmp_dir}/project"
mkdir -p "$project_dir"
failed_dir="${tmp_dir}/failed-first-run"
run_expect_code 1 "$out" env \
  TESTLOOP_FIRST_RUN_OUTPUT_DIR="$failed_dir" \
  TESTLOOP_FIRST_RUN_PROJECT_DIR="$project_dir" \
  TESTLOOP_FIRST_RUN_PROJECT_COMMAND='echo project failed; exit 7' \
  TESTLOOP_REPORT_SKIP_BASIC=true \
  TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true \
  TESTLOOP_REPORT_SKIP_AGENT_DEMO=true \
  bash "${repo_root}/scripts/doctor-first-run.sh" "$fake_binary"

assert_contains "$out" "first_run_status=failed"
assert_contains "$out" "first_run_failed_count=1"
assert_contains "$out" "first_run_agent_next_step=inspect-user-project"
assert_contains "$out" "first_run_next=inspect user project smoke command"
assert_contains "$failed_dir/verification-report.md" "project failed"
assert_contains "$failed_dir/first-run-context.txt" "first_run_agent_next_step=inspect-user-project"
assert_contains "$failed_dir/first-run-context.txt" "不要直接改生成测试"

run_expect_code 0 "$out" bash "${repo_root}/scripts/doctor-first-run.sh" --help
assert_contains "$out" "Usage: scripts/doctor-first-run.sh [testloop-mcp-binary]"
assert_contains "$out" "TESTLOOP_FIRST_RUN_OUTPUT_DIR"

run_expect_code 2 "$out" bash "${repo_root}/scripts/doctor-first-run.sh" one two
assert_contains "$out" "Usage: scripts/doctor-first-run.sh [testloop-mcp-binary]"

echo "doctor first-run test passed"
