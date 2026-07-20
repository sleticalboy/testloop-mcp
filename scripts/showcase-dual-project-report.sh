#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-dual-project-report.sh [testloop-mcp-binary]

Run a dual-project verification report for two user projects. The script writes
two verification reports and one combined summary JSON.

Arguments:
  testloop-mcp-binary  Optional binary path. Defaults to TESTLOOP_MCP_COMMAND,
                       then the testloop-mcp binary found on PATH.

Environment:
  TESTLOOP_PAIR_PREFIX              Output key prefix. Default: dual
  TESTLOOP_PAIR_OUTPUT_DIR          Output dir. Default: /tmp/testloop-dual-project
  TESTLOOP_PAIR_SUMMARY_JSON        Optional combined summary JSON path.
  TESTLOOP_PAIR_FIRST_NAME          First project name used for output paths.
  TESTLOOP_PAIR_FIRST_DIR           First project directory.
  TESTLOOP_PAIR_FIRST_COMMAND       First project smoke command.
  TESTLOOP_PAIR_FIRST_TITLE         First report title.
  TESTLOOP_PAIR_SECOND_NAME         Second project name used for output paths.
  TESTLOOP_PAIR_SECOND_DIR          Second project directory.
  TESTLOOP_PAIR_SECOND_COMMAND      Second project smoke command.
  TESTLOOP_PAIR_SECOND_TITLE        Second report title.

All TESTLOOP_REPORT_* variables supported by generate-verification-report.sh
are forwarded to both reports.
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
      [[ -f "$candidate" && -x "$candidate" ]] || fail "binary must be an executable file: $candidate"
      printf '%s' "$candidate"
      ;;
    *)
      command -v "$candidate" 2>/dev/null || fail "binary not found on PATH: $candidate"
      ;;
  esac
}

binary="$(resolve_binary "$command_path")"
prefix="${TESTLOOP_PAIR_PREFIX:-dual}"
output_dir="${TESTLOOP_PAIR_OUTPUT_DIR:-/tmp/testloop-dual-project}"
summary_json="${TESTLOOP_PAIR_SUMMARY_JSON:-${output_dir}/${prefix}-summary.json}"
first_name="${TESTLOOP_PAIR_FIRST_NAME:-first}"
second_name="${TESTLOOP_PAIR_SECOND_NAME:-second}"
first_dir="${TESTLOOP_PAIR_FIRST_DIR:-}"
second_dir="${TESTLOOP_PAIR_SECOND_DIR:-}"
first_command="${TESTLOOP_PAIR_FIRST_COMMAND:-}"
second_command="${TESTLOOP_PAIR_SECOND_COMMAND:-}"
first_title="${TESTLOOP_PAIR_FIRST_TITLE:-${first_name} 接入验收报告}"
second_title="${TESTLOOP_PAIR_SECOND_TITLE:-${second_name} 接入验收报告}"

[[ -n "$first_dir" ]] || fail "TESTLOOP_PAIR_FIRST_DIR is required"
[[ -n "$second_dir" ]] || fail "TESTLOOP_PAIR_SECOND_DIR is required"
[[ -n "$first_command" ]] || fail "TESTLOOP_PAIR_FIRST_COMMAND is required"
[[ -n "$second_command" ]] || fail "TESTLOOP_PAIR_SECOND_COMMAND is required"
[[ -d "$first_dir" ]] || fail "first project directory does not exist: $first_dir"
[[ -d "$second_dir" ]] || fail "second project directory does not exist: $second_dir"

first_output_dir="${output_dir}/${first_name}"
second_output_dir="${output_dir}/${second_name}"
first_report_md="${first_output_dir}/verification-report.md"
first_summary_json="${first_output_dir}/verification-summary.json"
second_report_md="${second_output_dir}/verification-report.md"
second_summary_json="${second_output_dir}/verification-summary.json"

mkdir -p "$first_output_dir" "$second_output_dir"

run_report() {
  local project_dir="$1"
  local project_command="$2"
  local report_md="$3"
  local summary_json_path="$4"
  local title="$5"
  set +e
  env \
    TESTLOOP_REPORT_OUTPUT="$report_md" \
    TESTLOOP_REPORT_SUMMARY_JSON="$summary_json_path" \
    TESTLOOP_REPORT_TITLE="$title" \
    TESTLOOP_REPORT_PROJECT_DIR="$project_dir" \
    TESTLOOP_REPORT_PROJECT_COMMAND="$project_command" \
    "$repo_root/scripts/generate-verification-report.sh" "$binary" "$report_md"
  local code=$?
  return "$code"
}

first_code=0
second_code=0
first_status="passed"
second_status="passed"

set +e
run_report "$first_dir" "$first_command" "$first_report_md" "$first_summary_json" "$first_title"
first_code=$?
set -e
if [[ "$first_code" -ne 0 ]]; then
  first_status="failed"
fi

set +e
run_report "$second_dir" "$second_command" "$second_report_md" "$second_summary_json" "$second_title"
second_code=$?
set -e
if [[ "$second_code" -ne 0 ]]; then
  second_status="failed"
fi

summary_status="passed"
if [[ "$first_code" -ne 0 || "$second_code" -ne 0 ]]; then
  summary_status="failed"
fi

mkdir -p "$(dirname "$summary_json")"
python3 - "$summary_json" "$output_dir" "$first_report_md" "$first_summary_json" "$first_status" "$first_command" "$second_report_md" "$second_summary_json" "$second_status" "$second_command" "$summary_status" "$first_name" "$second_name" <<'PY'
import json
import sys
from pathlib import Path

(
    summary_path,
    output_dir,
    first_report,
    first_summary,
    first_status,
    first_command,
    second_report,
    second_summary,
    second_status,
    second_command,
    summary_status,
    first_name,
    second_name,
) = sys.argv[1:]

first_summary_payload = json.loads(Path(first_summary).read_text(encoding="utf-8"))
second_summary_payload = json.loads(Path(second_summary).read_text(encoding="utf-8"))
failed_count = int(first_summary_payload.get("failed_count", 0)) + int(second_summary_payload.get("failed_count", 0))

payload = {
    "output_dir": output_dir,
    "overall_status": summary_status,
    "failed_count": failed_count,
    first_name: {
        "status": first_status,
        "command": first_command,
        "report": first_report,
        "summary_json": first_summary,
        "summary": first_summary_payload,
    },
    second_name: {
        "status": second_status,
        "command": second_command,
        "report": second_report,
        "summary_json": second_summary,
        "summary": second_summary_payload,
    },
}

Path(summary_path).write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY

printf '%s_output_dir=%s\n' "$prefix" "$output_dir"
printf '%s_summary_json=%s\n' "$prefix" "$summary_json"
printf '%s_%s_report=%s\n' "$prefix" "$first_name" "$first_report_md"
printf '%s_%s_summary=%s\n' "$prefix" "$first_name" "$first_summary_json"
printf '%s_%s_status=%s\n' "$prefix" "$first_name" "$first_status"
printf '%s_%s_command=%s\n' "$prefix" "$first_name" "$first_command"
printf '%s_%s_report=%s\n' "$prefix" "$second_name" "$second_report_md"
printf '%s_%s_summary=%s\n' "$prefix" "$second_name" "$second_summary_json"
printf '%s_%s_status=%s\n' "$prefix" "$second_name" "$second_status"
printf '%s_%s_command=%s\n' "$prefix" "$second_name" "$second_command"
printf '%s_status=%s\n' "$prefix" "$summary_status"

if [[ "$first_code" -ne 0 ]]; then
  exit "$first_code"
fi
if [[ "$second_code" -ne 0 ]]; then
  exit "$second_code"
fi
