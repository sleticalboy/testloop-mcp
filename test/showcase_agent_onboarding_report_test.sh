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
cat > "$fake_binary" <<'SH'
#!/usr/bin/env sh
case "${1:-}" in
  --version)
    echo "testloop-mcp 0.5.4"
    ;;
  *)
    echo "fake testloop-mcp"
    ;;
esac
SH
chmod +x "$fake_binary"

out="${tmp_dir}/onboarding.out"
output_dir="${tmp_dir}/artifacts"
run_expect_code 0 "$out" env \
  TESTLOOP_ONBOARDING_OUTPUT_DIR="$output_dir" \
  TESTLOOP_REPORT_SKIP_BASIC=true \
  TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true \
  TESTLOOP_REPORT_SKIP_AGENT_DEMO=true \
  bash "${repo_root}/scripts/showcase-agent-onboarding-report.sh" "$fake_binary"

report="${output_dir}/verification-report.md"
summary="${output_dir}/verification-summary.json"
decision="${output_dir}/agent-decision.txt"

assert_contains "$out" "onboarding_report=$report"
assert_contains "$out" "onboarding_summary_json=$summary"
assert_contains "$out" "onboarding_decision=$decision"
assert_contains "$out" "agent_next_step=ready"
assert_contains "$report" "# testloop-mcp 验收报告"
assert_contains "$summary" '"overall_status": "passed"'
assert_contains "$decision" "agent_next_step=ready"

run_expect_code 0 "$out" bash "${repo_root}/scripts/showcase-agent-onboarding-report.sh" --help
assert_contains "$out" "Usage: scripts/showcase-agent-onboarding-report.sh [testloop-mcp-binary]"

run_expect_code 2 "$out" bash "${repo_root}/scripts/showcase-agent-onboarding-report.sh" one two
assert_contains "$out" "Usage: scripts/showcase-agent-onboarding-report.sh [testloop-mcp-binary]"

echo "showcase agent onboarding report test passed"
