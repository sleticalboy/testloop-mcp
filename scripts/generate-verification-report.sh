#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/generate-verification-report.sh [testloop-mcp-binary] [output-md]

Generate a Markdown verification report for a local testloop-mcp setup.

Default sections:
  1. Basic install verification through scripts/verify-client-setup.sh.
  2. Real MCP process smoke through scripts/verify-mcp-process-smoke.sh.
  3. Minimal Agent feedback loop through examples/mcp-client-demo.
  4. Standalone testgen CLI action smoke.

Optional sections:
  - Public Go / JS showcase, enabled by TESTLOOP_REPORT_PUBLIC_SHOWCASES.
  - A user project smoke command, enabled by TESTLOOP_REPORT_PROJECT_DIR and
    TESTLOOP_REPORT_PROJECT_COMMAND.

Environment:
  TESTLOOP_MCP_COMMAND                  Binary path or command name to verify.
  TESTLOOP_REPORT_OUTPUT                Output Markdown path.
  TESTLOOP_REPORT_TITLE                 Report title. Default: testloop-mcp 验收报告
  TESTLOOP_REPORT_SUMMARY_JSON          Optional machine-readable summary JSON path.
  TESTLOOP_REPORT_EXPECT_VERSION        Optional expected binary version.
  TESTLOOP_REPORT_SKIP_BASIC            Set to true to skip install verification.
  TESTLOOP_REPORT_SKIP_PROCESS_SMOKE    Set to true to skip MCP process smoke.
  TESTLOOP_REPORT_SKIP_AGENT_DEMO       Set to true to skip minimal Agent demo.
  TESTLOOP_REPORT_SKIP_TESTGEN_SMOKE    Set to true to skip testgen CLI action smoke.
  TESTLOOP_REPORT_TESTGEN_COMMAND       Optional testgen command. Defaults to a sibling
                                        testloop-testgen binary, go run, then PATH.
  TESTLOOP_REPORT_PUBLIC_SHOWCASES      none, go, js, or all. Default: none.
  TESTLOOP_REPORT_PROJECT_DIR           Optional user project directory.
  TESTLOOP_REPORT_PROJECT_COMMAND       Optional smoke command run inside project dir.

Examples:
  scripts/generate-verification-report.sh "$(command -v testloop-mcp)"
  TESTLOOP_REPORT_EXPECT_VERSION=0.5.6 scripts/generate-verification-report.sh
  TESTLOOP_REPORT_PROJECT_DIR=/path/to/project \
    TESTLOOP_REPORT_PROJECT_COMMAND='go test ./...' \
    scripts/generate-verification-report.sh "$(command -v testloop-mcp)" report.md
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

if [[ $# -gt 2 ]]; then
  usage >&2
  exit 2
fi

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
command_path="${1:-${TESTLOOP_MCP_COMMAND:-}}"
output="${2:-${TESTLOOP_REPORT_OUTPUT:-/tmp/testloop-mcp-verification-report.md}}"
title="${TESTLOOP_REPORT_TITLE:-testloop-mcp 验收报告}"
summary_json="${TESTLOOP_REPORT_SUMMARY_JSON:-}"
expected_version="${TESTLOOP_REPORT_EXPECT_VERSION:-}"
skip_basic="${TESTLOOP_REPORT_SKIP_BASIC:-false}"
skip_process_smoke="${TESTLOOP_REPORT_SKIP_PROCESS_SMOKE:-false}"
skip_agent_demo="${TESTLOOP_REPORT_SKIP_AGENT_DEMO:-false}"
skip_testgen_smoke="${TESTLOOP_REPORT_SKIP_TESTGEN_SMOKE:-false}"
testgen_command="${TESTLOOP_REPORT_TESTGEN_COMMAND:-}"
public_showcases="${TESTLOOP_REPORT_PUBLIC_SHOWCASES:-none}"
project_dir="${TESTLOOP_REPORT_PROJECT_DIR:-}"
project_command="${TESTLOOP_REPORT_PROJECT_COMMAND:-}"

resolve_binary() {
  if [[ -n "$command_path" ]]; then
    case "$command_path" in
      */*)
        [[ -f "$command_path" && -x "$command_path" ]] || fail "binary must be an executable file: $command_path"
        dir="$(cd "$(dirname "$command_path")" && pwd)"
        printf '%s/%s' "$dir" "$(basename "$command_path")"
        ;;
      *)
        resolved="$(command -v "$command_path" 2>/dev/null)" || fail "binary not found on PATH: $command_path"
        printf '%s' "$resolved"
        ;;
    esac
    return
  fi

  resolved="$(command -v testloop-mcp 2>/dev/null)" || fail "testloop-mcp not found on PATH; pass a binary path or set TESTLOOP_MCP_COMMAND"
  printf '%s' "$resolved"
}

markdown_escape_table_cell() {
  printf '%s' "$1" | sed 's/|/\\|/g'
}

write_output_block() {
  local file="$1"
  if [[ -s "$file" ]]; then
    sed 's/[[:cntrl:]]//g' "$file"
  else
    printf '(no output)\n'
  fi
}

write_summary_json() {
  local json_output="$1"
  mkdir -p "$(dirname "$json_output")"
  python3 - "$summary_file" "$json_output" "$title" "$generated_at" "$repo_root" "$git_ref" "$binary" "${version_output:-unknown}" "$output" "$failed_count" <<'PY'
import csv
import json
import sys
from pathlib import Path

summary_file, json_output, title, generated_at, repo_root, git_ref, binary, version_output, report_path, failed_count = sys.argv[1:]

sections = []
with open(summary_file, encoding="utf-8", newline="") as fh:
    for row in csv.reader(fh, delimiter="\t"):
        if not row:
            continue
        name, status, code = row[:3]
        reason = row[3] if len(row) > 3 else ""
        signals_raw = row[4] if len(row) > 4 else ""
        section = {
            "name": name,
            "status": status,
            "exit_code": None if code == "-" else int(code),
            "reason": reason or None,
        }
        if signals_raw:
            section["signals"] = json.loads(signals_raw)
        sections.append(section)

failed = int(failed_count)
payload = {
    "title": title,
    "generated_at": generated_at,
    "repository": repo_root,
    "git_ref": git_ref,
    "binary": binary,
    "version_output": version_output,
    "markdown_report": report_path,
    "overall_status": "failed" if failed else "passed",
    "failed_count": failed,
    "sections": sections,
}

Path(json_output).write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
}

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

summary_file="${tmp_dir}/summary.tsv"
sections_file="${tmp_dir}/sections.md"
: >"$summary_file"
: >"$sections_file"

binary="$(resolve_binary)"
version_output="$("$binary" --version 2>/dev/null || true)"
generated_at="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
git_ref="$(git -C "$repo_root" rev-parse --short HEAD 2>/dev/null || printf 'unknown')"

run_section() {
  local name="$1"
  shift
  local out_file="${tmp_dir}/section-$((section_index + 1)).out"
  section_index=$((section_index + 1))

  printf '==> %s\n' "$name"
  set +e
  "$@" >"$out_file" 2>&1
  local code=$?
  set -e

  local status="passed"
  if [[ "$code" -ne 0 ]]; then
    status="failed"
    failed_count=$((failed_count + 1))
  fi
  local signals
  signals="$(section_signals_json "$name" "$out_file")"
  printf '%s\t%s\t%s\t%s\t%s\n' "$name" "$status" "$code" "" "$signals" >>"$summary_file"

  {
    printf '### %s\n\n' "$name"
    printf '%s\n' "- 状态：\`$status\`"
    printf '%s\n\n' "- Exit code：\`$code\`"
    printf '```text\n'
    write_output_block "$out_file"
    printf '```\n\n'
  } >>"$sections_file"
}

section_signals_json() {
  local name="$1"
  local out_file="$2"
  local action
  case "$name" in
    "独立 CLI 生成动作 smoke")
      action="$(grep -Eo 'action=[A-Za-z0-9_]+' "$out_file" | head -n 1 | cut -d= -f2 || true)"
      if [[ -n "$action" ]]; then
        python3 - "$action" <<'PY'
import json
import sys

print(json.dumps({"action": sys.argv[1]}, ensure_ascii=False, sort_keys=True))
PY
      fi
      ;;
  esac
}

run_testgen_action_smoke() {
  local smoke_dir="${tmp_dir}/testgen-action-smoke"
  local source="${smoke_dir}/alias.go"
  local output="${smoke_dir}/alias_test.go"
  local command_line
  local quoted_source
  local quoted_output
  mkdir -p "$smoke_dir"
  cat >"$source" <<'GO'
package alias

func SliceMapper[T any, U any](src []T, mapper func(T) U) []U {
	dst := make([]U, 0, len(src))
	for _, v := range src {
		dst = append(dst, mapper(v))
	}
	return dst
}
GO

  printf -v quoted_source '%q' "$source"
  printf -v quoted_output '%q' "$output"
  if [[ -n "$testgen_command" ]]; then
    command_line="${testgen_command} ${quoted_source} ${quoted_output}"
  else
    local sibling
    sibling="$(dirname "$binary")/testloop-testgen"
    if [[ -x "$sibling" ]]; then
      printf -v command_line '%q %q %q' "$sibling" "$source" "$output"
    elif command -v go >/dev/null 2>&1; then
      command_line="go run ./cmd/testgen ${quoted_source} ${quoted_output}"
    elif command -v testloop-testgen >/dev/null 2>&1; then
      command_line="testloop-testgen ${quoted_source} ${quoted_output}"
    else
      printf 'error: testloop-testgen is not on PATH and go is not available for go run fallback\n' >&2
      return 1
    fi
  fi

  printf 'command: %s\n' "$command_line"
  local cli_output
  cli_output="$(cd "$repo_root" && bash -lc "$command_line")"
  printf '%s\n' "$cli_output"
  case "$cli_output" in
    *"action=manual_review"*) ;;
    *)
      printf 'error: expected testgen output to contain action=manual_review\n' >&2
      return 1
      ;;
  esac
  grep -F 't.Skip("TODO: fill in meaningful test inputs and expected values")' "$output" >/dev/null
  printf 'testgen action smoke passed\n'
}

skip_section() {
  local name="$1"
  local reason="$2"
  section_index=$((section_index + 1))
  printf '%s\t%s\t%s\t%s\t%s\n' "$name" "skipped" "-" "$reason" "" >>"$summary_file"
  {
    printf '### %s\n\n' "$name"
    printf '%s\n' '- 状态：`skipped`'
    printf '%s\n\n' "- 原因：$reason"
  } >>"$sections_file"
}

section_index=0
failed_count=0

if [[ "$skip_basic" == "true" ]]; then
  skip_section "基础安装验收" "TESTLOOP_REPORT_SKIP_BASIC=true"
else
  basic_env=()
  basic_env+=("TESTLOOP_MCP_VERIFY_HTTP_ADDR=${TESTLOOP_MCP_VERIFY_HTTP_ADDR:-127.0.0.1:18082}")
  if [[ -n "$expected_version" ]]; then
    basic_env+=("TESTLOOP_MCP_VERIFY_EXPECT_VERSION=$expected_version")
  fi
  run_section "基础安装验收" env "${basic_env[@]}" "$repo_root/scripts/verify-client-setup.sh" "$binary"
fi

if [[ "$skip_process_smoke" == "true" ]]; then
  skip_section "真实 MCP 协议 smoke" "TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true"
else
  run_section "真实 MCP 协议 smoke" "$repo_root/scripts/verify-mcp-process-smoke.sh" "$binary"
fi

if [[ "$skip_agent_demo" == "true" ]]; then
  skip_section "最小 Agent 闭环 demo" "TESTLOOP_REPORT_SKIP_AGENT_DEMO=true"
else
  run_section "最小 Agent 闭环 demo" bash -lc "cd '$repo_root' && go run ./examples/mcp-client-demo"
fi

if [[ "$skip_testgen_smoke" == "true" ]]; then
  skip_section "独立 CLI 生成动作 smoke" "TESTLOOP_REPORT_SKIP_TESTGEN_SMOKE=true"
else
  run_section "独立 CLI 生成动作 smoke" run_testgen_action_smoke
fi

case "$public_showcases" in
  none|"")
    skip_section "公开 showcase" "TESTLOOP_REPORT_PUBLIC_SHOWCASES=none"
    ;;
  go)
    run_section "公开 Go showcase" "$repo_root/scripts/showcase-go-public-project.sh" "${tmp_dir}/showcase-go.jsonl"
    ;;
  js)
    run_section "公开 JS/TS showcase" "$repo_root/scripts/showcase-js-public-project.sh" "${tmp_dir}/showcase-js.jsonl"
    ;;
  all)
    run_section "公开 Go showcase" "$repo_root/scripts/showcase-go-public-project.sh" "${tmp_dir}/showcase-go.jsonl"
    run_section "公开 JS/TS showcase" "$repo_root/scripts/showcase-js-public-project.sh" "${tmp_dir}/showcase-js.jsonl"
    ;;
  *)
    fail "unsupported TESTLOOP_REPORT_PUBLIC_SHOWCASES: $public_showcases"
    ;;
esac

if [[ -n "$project_dir" || -n "$project_command" ]]; then
  [[ -n "$project_dir" ]] || fail "TESTLOOP_REPORT_PROJECT_DIR is required when TESTLOOP_REPORT_PROJECT_COMMAND is set"
  [[ -n "$project_command" ]] || fail "TESTLOOP_REPORT_PROJECT_COMMAND is required when TESTLOOP_REPORT_PROJECT_DIR is set"
  [[ -d "$project_dir" ]] || fail "project path must be a directory: $project_dir"
  run_section "用户项目 smoke" bash -lc "cd '$project_dir' && $project_command"
else
  skip_section "用户项目 smoke" "未设置 TESTLOOP_REPORT_PROJECT_DIR 和 TESTLOOP_REPORT_PROJECT_COMMAND"
fi

mkdir -p "$(dirname "$output")"
{
  printf '# %s\n\n' "$title"
  printf '%s%s%s\n' '- 生成时间：`' "$generated_at" '`'
  printf '%s%s%s\n' '- 仓库：`' "$repo_root" '`'
  printf '%s%s%s\n' '- Git ref：`' "$git_ref" '`'
  printf '%s%s%s\n' '- 二进制：`' "$binary" '`'
  printf '%s%s%s\n\n' '- 版本输出：`' "${version_output:-unknown}" '`'

  printf '## 汇总\n\n'
  printf '| 验收项 | 状态 | Exit code |\n'
  printf '| --- | --- | --- |\n'
  while IFS="$(printf '\t')" read -r name status code reason; do
    printf '| %s | `%s` | `%s` |\n' "$(markdown_escape_table_cell "$name")" "$status" "$code"
  done <"$summary_file"

  printf '\n## 明细\n\n'
  cat "$sections_file"
} >"$output"

printf 'Wrote %s\n' "$output"
if [[ -n "$summary_json" ]]; then
  write_summary_json "$summary_json"
  printf 'Wrote %s\n' "$summary_json"
fi
if [[ "$failed_count" -gt 0 ]]; then
  exit 1
fi
