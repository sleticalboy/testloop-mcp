#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/doctor-first-run.sh [testloop-mcp-binary]

Run a first-run diagnostic bundle for a local testloop-mcp binary.
The script wraps the onboarding report flow and always prints stable paths plus
agent-readable status fields:
  - first_run_status
  - first_run_agent_next_step
  - first_run_report
  - first_run_summary_json
  - first_run_decision
  - first_run_context
  - first_run_log

Arguments:
  testloop-mcp-binary  Optional binary path. Defaults to TESTLOOP_MCP_COMMAND,
                       then the testloop-mcp binary found on PATH.

Environment:
  TESTLOOP_FIRST_RUN_OUTPUT_DIR       Output dir. Default: /tmp/testloop-mcp-first-run
  TESTLOOP_FIRST_RUN_EXPECT_VERSION   Optional expected version, for example 0.5.6.
  TESTLOOP_FIRST_RUN_PROJECT_DIR      Optional user project directory.
  TESTLOOP_FIRST_RUN_PROJECT_COMMAND  Optional smoke command run inside project dir.

All TESTLOOP_REPORT_* variables supported by generate-verification-report.sh
are forwarded.

Examples:
  scripts/doctor-first-run.sh "$(command -v testloop-mcp)"
  TESTLOOP_FIRST_RUN_EXPECT_VERSION=0.5.6 scripts/doctor-first-run.sh
  TESTLOOP_FIRST_RUN_PROJECT_DIR=/path/to/project \
  TESTLOOP_FIRST_RUN_PROJECT_COMMAND='go test ./...' \
    scripts/doctor-first-run.sh "$(command -v testloop-mcp)"
USAGE
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ "$#" -gt 1 ]]; then
  usage >&2
  exit 2
fi

repo_root="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
command_path="${1:-${TESTLOOP_MCP_COMMAND:-}}"
if [[ -z "$command_path" ]]; then
  command_path="$(command -v testloop-mcp 2>/dev/null || true)"
fi
[[ -n "$command_path" ]] || fail "testloop-mcp not found on PATH; pass a binary path or set TESTLOOP_MCP_COMMAND"

output_dir="${TESTLOOP_FIRST_RUN_OUTPUT_DIR:-/tmp/testloop-mcp-first-run}"
report_md="${TESTLOOP_FIRST_RUN_REPORT_MD:-${output_dir}/verification-report.md}"
summary_json="${TESTLOOP_FIRST_RUN_SUMMARY_JSON:-${output_dir}/verification-summary.json}"
decision_out="${TESTLOOP_FIRST_RUN_DECISION_OUT:-${output_dir}/agent-decision.txt}"
context_out="${TESTLOOP_FIRST_RUN_CONTEXT_OUT:-${output_dir}/first-run-context.txt}"
log_out="${TESTLOOP_FIRST_RUN_LOG:-${output_dir}/first-run.log}"
expect_version="${TESTLOOP_FIRST_RUN_EXPECT_VERSION:-}"
project_dir="${TESTLOOP_FIRST_RUN_PROJECT_DIR:-}"
project_command="${TESTLOOP_FIRST_RUN_PROJECT_COMMAND:-}"

mkdir -p "$output_dir" "$(dirname "$report_md")" "$(dirname "$summary_json")" "$(dirname "$decision_out")" "$(dirname "$context_out")" "$(dirname "$log_out")"

env_args=(
  "TESTLOOP_ONBOARDING_OUTPUT_DIR=$output_dir"
  "TESTLOOP_ONBOARDING_REPORT_MD=$report_md"
  "TESTLOOP_ONBOARDING_SUMMARY_JSON=$summary_json"
  "TESTLOOP_ONBOARDING_DECISION_OUT=$decision_out"
)

if [[ -n "$expect_version" ]]; then
  env_args+=("TESTLOOP_MCP_VERIFY_EXPECT_VERSION=$expect_version")
fi
if [[ -n "$project_dir" || -n "$project_command" ]]; then
  [[ -n "$project_dir" ]] || fail "TESTLOOP_FIRST_RUN_PROJECT_DIR is required when TESTLOOP_FIRST_RUN_PROJECT_COMMAND is set"
  [[ -n "$project_command" ]] || fail "TESTLOOP_FIRST_RUN_PROJECT_COMMAND is required when TESTLOOP_FIRST_RUN_PROJECT_DIR is set"
  env_args+=("TESTLOOP_REPORT_PROJECT_DIR=$project_dir")
  env_args+=("TESTLOOP_REPORT_PROJECT_COMMAND=$project_command")
fi

set +e
env "${env_args[@]}" "$repo_root/scripts/showcase-agent-onboarding-report.sh" "$command_path" >"$log_out" 2>&1
onboarding_code=$?
set -e

status="failed"
failed_count="unknown"
agent_next_step="unknown"

if [[ -s "$summary_json" ]]; then
  read -r status failed_count < <(
    python3 - "$summary_json" <<'PY'
import json
import sys
from pathlib import Path

data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
print(data.get("overall_status", "failed"), data.get("failed_count", "unknown"))
PY
  )
fi

if [[ -s "$decision_out" ]]; then
  agent_next_step="$(grep -E '^agent_next_step=' "$decision_out" | tail -n 1 | cut -d= -f2- || true)"
  [[ -n "$agent_next_step" ]] || agent_next_step="unknown"
fi

printf 'first_run_status=%s\n' "$status"
printf 'first_run_failed_count=%s\n' "$failed_count"
printf 'first_run_agent_next_step=%s\n' "$agent_next_step"
printf 'first_run_report=%s\n' "$report_md"
printf 'first_run_summary_json=%s\n' "$summary_json"
printf 'first_run_decision=%s\n' "$decision_out"
printf 'first_run_context=%s\n' "$context_out"
printf 'first_run_log=%s\n' "$log_out"

case "$agent_next_step" in
  ready)
    next_text='install path is ready; continue with client config or project validation'
    ;;
  fix-installation)
    next_text='inspect binary path, version, generated config, and HTTP healthz'
    ;;
  inspect-mcp-transport)
    next_text='inspect stdio or Streamable HTTP MCP transport startup'
    ;;
  inspect-agent-demo)
    next_text='inspect structuredContent feedback loop and demo runner output'
    ;;
  inspect-showcase)
    next_text='inspect external network, showcase checkout, or action expectation drift'
    ;;
  inspect-user-project)
    next_text='inspect user project smoke command and report details'
    ;;
  *)
    next_text='open the summary JSON first, then inspect the Markdown report'
    ;;
esac
printf 'first_run_next=%s\n' "$next_text"

{
  printf 'testloop-mcp first-run diagnostic context\n'
  printf 'first_run_status=%s\n' "$status"
  printf 'first_run_failed_count=%s\n' "$failed_count"
  printf 'first_run_agent_next_step=%s\n' "$agent_next_step"
  printf 'first_run_next=%s\n' "$next_text"
  printf 'first_run_report=%s\n' "$report_md"
  printf 'first_run_summary_json=%s\n' "$summary_json"
  printf 'first_run_decision=%s\n' "$decision_out"
  printf 'first_run_log=%s\n' "$log_out"
  printf '\nSuggested prompt:\n'
  printf '请根据 first_run_agent_next_step 和 summary JSON，先判断失败属于安装、MCP transport、Agent demo、公开 showcase 还是用户项目 smoke，再给出下一步修复动作。不要直接改生成测试，先打开 Markdown report 里的失败 section。\n'
} >"$context_out"

exit "$onboarding_code"
