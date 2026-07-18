#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-agent-onboarding-report.sh [testloop-mcp-binary]

Run the onboarding showcase and write human-readable plus agent-readable outputs:
  1. Markdown verification report.
  2. Summary JSON for Agent / CI routing.
  3. Decision demo output with agent_next_step.

Arguments:
  testloop-mcp-binary  Optional binary path. Defaults to TESTLOOP_MCP_COMMAND,
                       then the testloop-mcp binary found on PATH.

Environment:
  TESTLOOP_MCP_COMMAND                  Binary path or command name to verify.
  TESTLOOP_MCP_VERIFY_EXPECT_VERSION    Optional expected version, for example 0.5.4.
  TESTLOOP_ONBOARDING_OUTPUT_DIR        Output directory. Default: /tmp/testloop-mcp-onboarding
  TESTLOOP_ONBOARDING_REPORT_MD         Markdown report path.
  TESTLOOP_ONBOARDING_SUMMARY_JSON      Summary JSON path.
  TESTLOOP_ONBOARDING_DECISION_OUT      Decision output path.

All TESTLOOP_REPORT_* variables supported by generate-verification-report.sh
are forwarded, including optional public showcases and user project smoke.

Examples:
  scripts/showcase-agent-onboarding-report.sh
  TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.4 scripts/showcase-agent-onboarding-report.sh
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

output_dir="${TESTLOOP_ONBOARDING_OUTPUT_DIR:-/tmp/testloop-mcp-onboarding}"
report_md="${TESTLOOP_ONBOARDING_REPORT_MD:-${output_dir}/verification-report.md}"
summary_json="${TESTLOOP_ONBOARDING_SUMMARY_JSON:-${output_dir}/verification-summary.json}"
decision_out="${TESTLOOP_ONBOARDING_DECISION_OUT:-${output_dir}/agent-decision.txt}"

mkdir -p "$output_dir" "$(dirname "$report_md")" "$(dirname "$summary_json")" "$(dirname "$decision_out")"

report_env=(
  "TESTLOOP_REPORT_SUMMARY_JSON=$summary_json"
)
if [[ -n "${TESTLOOP_MCP_VERIFY_EXPECT_VERSION:-}" && -z "${TESTLOOP_REPORT_EXPECT_VERSION:-}" ]]; then
  report_env+=("TESTLOOP_REPORT_EXPECT_VERSION=$TESTLOOP_MCP_VERIFY_EXPECT_VERSION")
fi

set +e
env "${report_env[@]}" "$repo_root/scripts/generate-verification-report.sh" "$command_path" "$report_md"
report_code=$?
set -e

decision_code=0
if [[ -s "$summary_json" ]]; then
  set +e
  (
    cd "$repo_root"
    go run ./examples/verification-summary-decision-demo "$summary_json"
  ) >"$decision_out" 2>&1
  decision_code=$?
  set -e
else
  printf 'summary JSON was not generated: %s\n' "$summary_json" >"$decision_out"
  decision_code=1
fi

printf 'onboarding_report=%s\n' "$report_md"
printf 'onboarding_summary_json=%s\n' "$summary_json"
printf 'onboarding_decision=%s\n' "$decision_out"
if [[ -s "$decision_out" ]]; then
  grep -E '^agent_next_step=' "$decision_out" || true
fi

if [[ "$report_code" -ne 0 ]]; then
  exit "$report_code"
fi
if [[ "$decision_code" -ne 0 ]]; then
  exit "$decision_code"
fi
