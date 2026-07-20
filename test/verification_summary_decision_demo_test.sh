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

passed_summary="${tmp_dir}/passed-summary.json"
cat > "$passed_summary" <<'JSON'
{
  "overall_status": "passed",
  "failed_count": 0,
  "markdown_report": "/tmp/testloop-report.md",
  "sections": [
    {
      "name": "基础安装验收",
      "status": "passed",
      "exit_code": 0,
      "reason": null
    },
    {
      "name": "用户项目 smoke",
      "status": "skipped",
      "exit_code": null,
      "reason": "未设置 TESTLOOP_REPORT_PROJECT_DIR 和 TESTLOOP_REPORT_PROJECT_COMMAND"
    },
    {
      "name": "独立 CLI 生成动作 smoke",
      "status": "passed",
      "exit_code": 0,
      "reason": null,
      "signals": {
        "action": "manual_review"
      }
    }
  ]
}
JSON

project_failed_summary="${tmp_dir}/project-failed-summary.json"
cat > "$project_failed_summary" <<'JSON'
{
  "overall_status": "failed",
  "failed_count": 1,
  "markdown_report": "/tmp/testloop-project-report.md",
  "sections": [
    {
      "name": "基础安装验收",
      "status": "passed",
      "exit_code": 0,
      "reason": null
    },
    {
      "name": "用户项目 smoke",
      "status": "failed",
      "exit_code": 7,
      "reason": null
    }
  ]
}
JSON

transport_failed_summary="${tmp_dir}/transport-failed-summary.json"
cat > "$transport_failed_summary" <<'JSON'
{
  "overall_status": "failed",
  "failed_count": 1,
  "markdown_report": "/tmp/testloop-transport-report.md",
  "sections": [
    {
      "name": "真实 MCP 协议 smoke",
      "status": "failed",
      "exit_code": 1,
      "reason": null
    }
  ]
}
JSON

missing_sections_summary="${tmp_dir}/missing-sections-summary.json"
cat > "$missing_sections_summary" <<'JSON'
{
  "overall_status": "passed",
  "failed_count": 0
}
JSON

invalid_status_summary="${tmp_dir}/invalid-status-summary.json"
cat > "$invalid_status_summary" <<'JSON'
{
  "overall_status": "unknown",
  "failed_count": 0,
  "sections": []
}
JSON

out="${tmp_dir}/decision.out"
err="${tmp_dir}/decision.err"

(cd "$repo_root" && go run ./examples/verification-summary-decision-demo "$passed_summary") > "$out"
assert_contains "$out" "verification_summary: status=passed failed=0 sections=3"
assert_contains "$out" "section_signal=独立 CLI 生成动作 smoke action=manual_review"
assert_contains "$out" "agent_next_step=ready"

(cd "$repo_root" && go run ./examples/verification-summary-decision-demo "$project_failed_summary") > "$out"
assert_contains "$out" "verification_summary: status=failed failed=1 sections=2"
assert_contains "$out" "1. failed_section=用户项目 smoke exit_code=7 decision=inspect-user-project"
assert_contains "$out" "agent_next_step=inspect-user-project"
assert_contains "$out" "markdown_report=/tmp/testloop-project-report.md"

(cd "$repo_root" && go run ./examples/verification-summary-decision-demo "$transport_failed_summary") > "$out"
assert_contains "$out" "1. failed_section=真实 MCP 协议 smoke exit_code=1 decision=inspect-mcp-transport"
assert_contains "$out" "agent_next_step=inspect-mcp-transport"

if (cd "$repo_root" && go run ./examples/verification-summary-decision-demo "$missing_sections_summary") >"$out" 2>"$err"; then
  echo "expected missing sections summary to fail" >&2
  exit 1
fi
assert_contains "$err" "missing sections"

if (cd "$repo_root" && go run ./examples/verification-summary-decision-demo "$invalid_status_summary") >"$out" 2>"$err"; then
  echo "expected invalid status summary to fail" >&2
  exit 1
fi
assert_contains "$err" "invalid overall_status"

echo "verification summary decision demo test passed"
