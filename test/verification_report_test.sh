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
    echo "testloop-mcp 0.5.2"
    ;;
  *)
    echo "fake testloop-mcp"
    ;;
esac
SH
chmod +x "$fake_binary"

project_dir="${tmp_dir}/project"
mkdir -p "$project_dir"

out="${tmp_dir}/report.out"
report="${tmp_dir}/report.md"
run_expect_code 0 "$out" env \
  TESTLOOP_REPORT_SKIP_BASIC=true \
  TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true \
  TESTLOOP_REPORT_SKIP_AGENT_DEMO=true \
  TESTLOOP_REPORT_PROJECT_DIR="$project_dir" \
  TESTLOOP_REPORT_PROJECT_COMMAND='printf "project smoke ok\n"' \
  bash "${repo_root}/scripts/generate-verification-report.sh" "$fake_binary" "$report"

assert_contains "$out" "Wrote $report"
assert_contains "$report" "# testloop-mcp 验收报告"
assert_contains "$report" '| 基础安装验收 | `skipped` | `-` |'
assert_contains "$report" '| 用户项目 smoke | `passed` | `0` |'
assert_contains "$report" "project smoke ok"
assert_contains "$report" '版本输出：`testloop-mcp 0.5.2`'

failed_report="${tmp_dir}/failed-report.md"
run_expect_code 1 "$out" env \
  TESTLOOP_REPORT_SKIP_BASIC=true \
  TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true \
  TESTLOOP_REPORT_SKIP_AGENT_DEMO=true \
  TESTLOOP_REPORT_PROJECT_DIR="$project_dir" \
  TESTLOOP_REPORT_PROJECT_COMMAND='echo project failed; exit 7' \
  bash "${repo_root}/scripts/generate-verification-report.sh" "$fake_binary" "$failed_report"

assert_contains "$failed_report" '| 用户项目 smoke | `failed` | `7` |'
assert_contains "$failed_report" "project failed"

echo "verification report test passed"
