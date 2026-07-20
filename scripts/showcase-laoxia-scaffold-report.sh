#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-laoxia-scaffold-report.sh [testloop-mcp-binary]

Run a dual-stack laoxia scaffold report for the Go server and Vue web project.
The script writes two verification reports and summary JSON files:
  - server/verification-report.md
  - server/verification-summary.json
  - web/verification-report.md
  - web/verification-summary.json

Arguments:
  testloop-mcp-binary  Optional binary path. Defaults to TESTLOOP_MCP_COMMAND,
                       then the testloop-mcp binary found on PATH.

Environment:
  TESTLOOP_LAOXIA_OUTPUT_DIR           Output dir. Default: /tmp/testloop-laoxia-scaffold
  TESTLOOP_LAOXIA_SERVER_DIR           Go server project dir.
  TESTLOOP_LAOXIA_WEB_DIR              Vue web project dir.
  TESTLOOP_LAOXIA_SERVER_COMMAND       Server smoke command. Default: go test ./...
  TESTLOOP_LAOXIA_WEB_COMMAND          Web smoke command. Default: pnpm install --frozen-lockfile && pnpm build:prod
  TESTLOOP_LAOXIA_SUMMARY_JSON         Optional combined summary JSON path.
  TESTLOOP_LAOXIA_SERVER_TITLE         Optional report title override for the server report.
  TESTLOOP_LAOXIA_WEB_TITLE            Optional report title override for the web report.

All TESTLOOP_REPORT_* variables supported by generate-verification-report.sh
are forwarded to both reports.

Examples:
  scripts/showcase-laoxia-scaffold-report.sh "$(command -v testloop-mcp)"
  TESTLOOP_LAOXIA_OUTPUT_DIR=/tmp/testloop-laoxia \
    scripts/showcase-laoxia-scaffold-report.sh
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

resolve_binary() {
  local candidate="$1"
  case "$candidate" in
    */*)
      [[ -x "$candidate" ]] || fail "binary is not executable: $candidate"
      printf '%s' "$candidate"
      ;;
    *)
      command -v "$candidate" 2>/dev/null || fail "binary not found on PATH: $candidate"
      ;;
  esac
}

binary="$(resolve_binary "$command_path")"
output_dir="${TESTLOOP_LAOXIA_OUTPUT_DIR:-/tmp/testloop-laoxia-scaffold}"
summary_json="${TESTLOOP_LAOXIA_SUMMARY_JSON:-${output_dir}/laoxia-summary.json}"
server_dir="${TESTLOOP_LAOXIA_SERVER_DIR:-/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server}"
web_dir="${TESTLOOP_LAOXIA_WEB_DIR:-/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-web}"
server_command="${TESTLOOP_LAOXIA_SERVER_COMMAND:-go test ./...}"
web_command="${TESTLOOP_LAOXIA_WEB_COMMAND:-pnpm install --frozen-lockfile && pnpm build:prod}"
server_title="${TESTLOOP_LAOXIA_SERVER_TITLE:-laoxia car-admin-server 接入验收报告}"
web_title="${TESTLOOP_LAOXIA_WEB_TITLE:-laoxia car-admin-web 接入验收报告}"

[[ -d "$server_dir" ]] || fail "server project directory does not exist: $server_dir"
[[ -d "$web_dir" ]] || fail "web project directory does not exist: $web_dir"

server_output_dir="${output_dir}/server"
web_output_dir="${output_dir}/web"
server_report_md="${server_output_dir}/verification-report.md"
server_summary_json="${server_output_dir}/verification-summary.json"
web_report_md="${web_output_dir}/verification-report.md"
web_summary_json="${web_output_dir}/verification-summary.json"

mkdir -p "$server_output_dir" "$web_output_dir"

run_report_with_env() {
  local project_dir="$1"
  local project_command="$2"
  local report_md="$3"
  local summary_json="$4"
  local title="$5"
  set +e
  env \
    TESTLOOP_REPORT_OUTPUT="$report_md" \
    TESTLOOP_REPORT_SUMMARY_JSON="$summary_json" \
    TESTLOOP_REPORT_TITLE="$title" \
    TESTLOOP_REPORT_PROJECT_DIR="$project_dir" \
    TESTLOOP_REPORT_PROJECT_COMMAND="$project_command" \
    "$repo_root/scripts/generate-verification-report.sh" "$binary" "$report_md"
  local code=$?
  return "$code"
}

server_code=0
web_code=0
server_status="passed"
web_status="passed"

set +e
run_report_with_env "$server_dir" "$server_command" "$server_report_md" "$server_summary_json" "$server_title"
server_code=$?
set -e
if [[ "$server_code" -ne 0 ]]; then
  server_status="failed"
fi

set +e
run_report_with_env "$web_dir" "$web_command" "$web_report_md" "$web_summary_json" "$web_title"
web_code=$?
set -e
if [[ "$web_code" -ne 0 ]]; then
  web_status="failed"
fi

summary_status="passed"
if [[ "$server_code" -ne 0 || "$web_code" -ne 0 ]]; then
  summary_status="failed"
fi

mkdir -p "$(dirname "$summary_json")"
python3 - "$summary_json" "$output_dir" "$server_report_md" "$server_summary_json" "$server_status" "$server_command" "$web_report_md" "$web_summary_json" "$web_status" "$web_command" "$summary_status" <<'PY'
import json
import sys
from pathlib import Path

(
    summary_path,
    output_dir,
    server_report,
    server_summary,
    server_status,
    server_command,
    web_report,
    web_summary,
    web_status,
    web_command,
    summary_status,
) = sys.argv[1:]

payload = {
    "output_dir": output_dir,
    "overall_status": summary_status,
    "failed_count": 0 if summary_status == "passed" else 1,
    "server": {
        "status": server_status,
        "command": server_command,
        "report": server_report,
        "summary_json": server_summary,
    },
    "web": {
        "status": web_status,
        "command": web_command,
        "report": web_report,
        "summary_json": web_summary,
    },
}

Path(summary_path).write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY

printf 'laoxia_output_dir=%s\n' "$output_dir"
printf 'laoxia_summary_json=%s\n' "$summary_json"
printf 'laoxia_server_report=%s\n' "$server_report_md"
printf 'laoxia_server_summary=%s\n' "$server_summary_json"
printf 'laoxia_server_status=%s\n' "$server_status"
printf 'laoxia_server_command=%s\n' "$server_command"
printf 'laoxia_web_report=%s\n' "$web_report_md"
printf 'laoxia_web_summary=%s\n' "$web_summary_json"
printf 'laoxia_web_status=%s\n' "$web_status"
printf 'laoxia_web_command=%s\n' "$web_command"
printf 'laoxia_status=%s\n' "$summary_status"

if [[ "$server_code" -ne 0 ]]; then
  exit "$server_code"
fi
if [[ "$web_code" -ne 0 ]]; then
  exit "$web_code"
fi
