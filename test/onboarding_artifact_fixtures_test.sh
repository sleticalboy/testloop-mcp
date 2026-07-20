#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
fixture_dir="${repo_root}/docs/fixtures/onboarding-artifacts/user-project-smoke-failed"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

assert_file() {
  path="$1"
  if [ ! -f "$path" ]; then
    echo "missing fixture file: $path" >&2
    exit 1
  fi
}

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

for name in \
  README.md \
  verification-report.md \
  verification-summary.json \
  verification-summary.schema.json \
  agent-decision.txt \
  agent-response.txt
do
  assert_file "$fixture_dir/$name"
done

cmp "${repo_root}/docs/fixtures/verification-summary.schema.json" "$fixture_dir/verification-summary.schema.json"
ruby -rjson -e 'JSON.parse(File.read(ARGV.fetch(0)));' "$fixture_dir/verification-summary.json"
assert_contains "$fixture_dir/agent-decision.txt" "agent_next_step=inspect-user-project"
assert_contains "$fixture_dir/agent-response.txt" "结论：testloop-mcp onboarding 链路本身是通的，失败发生在用户项目 smoke。"
assert_contains "$fixture_dir/agent-response.txt" "- failed_section=用户项目 smoke"
assert_contains "$fixture_dir/agent-response.txt" "- section_signal=独立 CLI 生成动作 smoke action=manual_review"
assert_contains "$fixture_dir/verification-summary.json" '"overall_status": "failed"'
assert_contains "$fixture_dir/verification-summary.schema.json" '"title": "testloop-mcp verification summary"'
assert_contains "$fixture_dir/verification-summary.json" '"failed_count": 1'
assert_contains "$fixture_dir/verification-summary.json" '"name": "独立 CLI 生成动作 smoke"'
assert_contains "$fixture_dir/verification-summary.json" '"signals": {'
assert_contains "$fixture_dir/verification-summary.json" '"action": "manual_review"'
assert_contains "$fixture_dir/verification-report.md" "project failed from onboarding fixture"
assert_contains "$fixture_dir/verification-report.md" "provider=static action=manual_review"
assert_contains "${repo_root}/docs/fixtures.md" "./fixtures/onboarding-artifacts/user-project-smoke-failed/"
assert_contains "${repo_root}/docs/fixtures.md" "onboarding artifact fixture"
assert_contains "${repo_root}/docs/fixtures.md" "agent-response.txt"
assert_contains "${repo_root}/docs/fixtures.md" "verification-summary.schema.json"

out="${tmp_dir}/response.out"
(cd "$repo_root" && go run ./examples/onboarding-agent-response-demo \
  "$fixture_dir/verification-summary.json") > "$out"

assert_contains "$out" "结论：testloop-mcp onboarding 链路本身是通的，失败发生在用户项目 smoke。"
assert_contains "$out" "- failed_section=用户项目 smoke"
assert_contains "$out" "- exit_code=7"
assert_contains "$out" "- section_signal=独立 CLI 生成动作 smoke action=manual_review"

sh "${repo_root}/scripts/render-onboarding-agent-response.sh" "$fixture_dir" > "${tmp_dir}/rendered-response.out"
cmp "$fixture_dir/agent-response.txt" "${tmp_dir}/rendered-response.out"

echo "onboarding artifact fixtures test passed"
