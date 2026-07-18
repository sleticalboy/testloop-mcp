#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/run-first-run-ci.sh [project-smoke-command]

Bootstrap a CI-friendly first-run diagnostic for a user project.
The script installs or resolves testloop-mcp, prepares a testloop-mcp source
checkout for helper scripts, then writes:
  - verification-report.md
  - verification-summary.json
  - agent-decision.txt
  - first-run-context.txt
  - first-run.log

Arguments:
  project-smoke-command  Optional smoke command. Defaults to
                         TESTLOOP_FIRST_RUN_PROJECT_COMMAND.

Environment:
  TESTLOOP_FIRST_RUN_PROJECT_DIR      Project directory. Default: current dir.
  TESTLOOP_FIRST_RUN_PROJECT_COMMAND  Smoke command run in the project dir.
  TESTLOOP_FIRST_RUN_OUTPUT_DIR       Output dir. Default: /tmp/testloop-first-run
  TESTLOOP_FIRST_RUN_EXPECT_VERSION   Expected binary version. Defaults to
                                      TESTLOOP_MCP_VERSION without a leading v
                                      when TESTLOOP_MCP_VERSION is not latest.

  TESTLOOP_MCP_COMMAND                Existing testloop-mcp binary path/command.
  TESTLOOP_MCP_VERSION                Binary version to install. Default: latest.
  TESTLOOP_MCP_REPO_DIR               Existing testloop-mcp source checkout.
  TESTLOOP_MCP_REPO_REF               Source ref to clone for helper scripts.
                                      Default: main.
  TESTLOOP_MCP_INSTALL_DIR            Install dir. Default: $HOME/.local/bin
  TESTLOOP_MCP_REPO_URL               Source repo URL.

Examples:
  TESTLOOP_MCP_VERSION=v0.5.6 scripts/run-first-run-ci.sh 'go test ./...'
  TESTLOOP_FIRST_RUN_PROJECT_COMMAND='pnpm install --frozen-lockfile && pnpm build' \
    scripts/run-first-run-ci.sh
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

script_root="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." 2>/dev/null && pwd || true)"
project_dir="${TESTLOOP_FIRST_RUN_PROJECT_DIR:-$PWD}"
project_command="${1:-${TESTLOOP_FIRST_RUN_PROJECT_COMMAND:-}}"
output_dir="${TESTLOOP_FIRST_RUN_OUTPUT_DIR:-/tmp/testloop-first-run}"
version="${TESTLOOP_MCP_VERSION:-latest}"
install_dir="${TESTLOOP_MCP_INSTALL_DIR:-$HOME/.local/bin}"
repo_url="${TESTLOOP_MCP_REPO_URL:-https://github.com/sleticalboy/testloop-mcp.git}"
repo_ref="${TESTLOOP_MCP_REPO_REF:-}"
repo_dir="${TESTLOOP_MCP_REPO_DIR:-}"
command_path="${TESTLOOP_MCP_COMMAND:-}"
expect_version="${TESTLOOP_FIRST_RUN_EXPECT_VERSION:-}"

[[ -d "$project_dir" ]] || fail "project directory does not exist: $project_dir"

if [[ -z "$repo_ref" ]]; then
  repo_ref="main"
fi

if [[ -z "$expect_version" && -n "$version" && "$version" != "latest" ]]; then
  expect_version="${version#v}"
fi

has_first_run_helpers() {
  local candidate="$1"
  [[ -x "$candidate/scripts/doctor-first-run.sh" && -x "$candidate/scripts/install.sh" ]]
}

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

if [[ -n "$repo_dir" ]]; then
  has_first_run_helpers "$repo_dir" || fail "TESTLOOP_MCP_REPO_DIR is not a testloop-mcp checkout with required scripts: $repo_dir"
elif [[ -n "$script_root" && -d "$script_root" ]] && has_first_run_helpers "$script_root"; then
  repo_dir="$script_root"
else
  command -v git >/dev/null 2>&1 || fail "git is required to clone testloop-mcp helpers"
  repo_dir="$tmp_dir/testloop-mcp"
  git clone --depth 1 --branch "$repo_ref" "$repo_url" "$repo_dir" >/dev/null 2>&1 || {
    printf 'warning: failed to clone %s ref %s; retrying default branch\n' "$repo_url" "$repo_ref" >&2
    git clone --depth 1 "$repo_url" "$repo_dir" >/dev/null 2>&1
  }
fi

resolve_binary() {
  local candidate="$1"
  if [[ -n "$candidate" ]]; then
    case "$candidate" in
      */*)
        [[ -x "$candidate" ]] || fail "binary is not executable: $candidate"
        printf '%s' "$candidate"
        ;;
      *)
        command -v "$candidate" 2>/dev/null || fail "binary not found on PATH: $candidate"
        ;;
    esac
    return
  fi

  if [[ -n "$version" && "$version" != "latest" ]]; then
    TESTLOOP_MCP_VERSION="$version" TESTLOOP_MCP_INSTALL_DIR="$install_dir" "$repo_dir/scripts/install.sh" >/dev/null
    local installed="$install_dir/testloop-mcp"
    if [[ -x "$install_dir/testloop-mcp.exe" ]]; then
      installed="$install_dir/testloop-mcp.exe"
    fi
    [[ -x "$installed" ]] || fail "testloop-mcp install did not produce an executable at $installed"
    printf '%s' "$installed"
    return
  fi

  if command -v testloop-mcp >/dev/null 2>&1; then
    command -v testloop-mcp
    return
  fi

  TESTLOOP_MCP_VERSION="$version" TESTLOOP_MCP_INSTALL_DIR="$install_dir" "$repo_dir/scripts/install.sh" >/dev/null
  local installed="$install_dir/testloop-mcp"
  if [[ -x "$install_dir/testloop-mcp.exe" ]]; then
    installed="$install_dir/testloop-mcp.exe"
  fi
  [[ -x "$installed" ]] || fail "testloop-mcp install did not produce an executable at $installed"
  printf '%s' "$installed"
}

binary="$(resolve_binary "$command_path")"
report_md="${TESTLOOP_FIRST_RUN_REPORT_MD:-${output_dir}/verification-report.md}"
summary_json="${TESTLOOP_FIRST_RUN_SUMMARY_JSON:-${output_dir}/verification-summary.json}"
decision_out="${TESTLOOP_FIRST_RUN_DECISION_OUT:-${output_dir}/agent-decision.txt}"
context_out="${TESTLOOP_FIRST_RUN_CONTEXT_OUT:-${output_dir}/first-run-context.txt}"
log_out="${TESTLOOP_FIRST_RUN_LOG:-${output_dir}/first-run.log}"

printf 'testloop_mcp_repo=%s\n' "$repo_dir"
printf 'testloop_mcp_binary=%s\n' "$binary"
printf 'testloop_first_run_output_dir=%s\n' "$output_dir"
printf 'testloop_project_dir=%s\n' "$project_dir"
if [[ -n "$project_command" ]]; then
  printf 'testloop_project_command=%s\n' "$project_command"
fi

env_args=(
  "TESTLOOP_FIRST_RUN_OUTPUT_DIR=$output_dir"
  "TESTLOOP_FIRST_RUN_REPORT_MD=$report_md"
  "TESTLOOP_FIRST_RUN_SUMMARY_JSON=$summary_json"
  "TESTLOOP_FIRST_RUN_DECISION_OUT=$decision_out"
  "TESTLOOP_FIRST_RUN_CONTEXT_OUT=$context_out"
  "TESTLOOP_FIRST_RUN_LOG=$log_out"
)
if [[ -n "$expect_version" ]]; then
  env_args+=("TESTLOOP_FIRST_RUN_EXPECT_VERSION=$expect_version")
fi
if [[ -n "$project_command" ]]; then
  env_args+=("TESTLOOP_FIRST_RUN_PROJECT_DIR=$project_dir")
  env_args+=("TESTLOOP_FIRST_RUN_PROJECT_COMMAND=$project_command")
fi

write_github_step_summary() {
  [[ -n "${GITHUB_STEP_SUMMARY:-}" ]] || return 0

  local status="unknown"
  local failed_count="unknown"
  local agent_next_step="unknown"
  if [[ -s "$summary_json" ]]; then
    read -r status failed_count < <(
      python3 - "$summary_json" <<'PY'
import json
import sys
from pathlib import Path

data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
print(data.get("overall_status", "unknown"), data.get("failed_count", "unknown"))
PY
    )
  fi
  if [[ -s "$decision_out" ]]; then
    agent_next_step="$(grep -E '^agent_next_step=' "$decision_out" | tail -n 1 | cut -d= -f2- || true)"
    [[ -n "$agent_next_step" ]] || agent_next_step="unknown"
  fi

  {
    printf '## testloop-mcp first-run\n\n'
    printf '%s%s%s\n' '- Status: `' "$status" '`'
    printf '%s%s%s\n' '- Failed sections: `' "$failed_count" '`'
    printf '%s%s%s\n' '- first_run_agent_next_step: `' "$agent_next_step" '`'
    printf '%s%s%s\n' '- Markdown report: `' "$report_md" '`'
    printf '%s%s%s\n' '- Summary JSON: `' "$summary_json" '`'
    printf '%s%s%s\n' '- Agent decision: `' "$decision_out" '`'
    printf '%s%s%s\n' '- Agent context: `' "$context_out" '`'
    printf '%s%s%s\n\n' '- Full log: `' "$log_out" '`'
  } >>"$GITHUB_STEP_SUMMARY"
}

set +e
env "${env_args[@]}" "$repo_dir/scripts/doctor-first-run.sh" "$binary"
first_run_code=$?
set -e

write_github_step_summary
exit "$first_run_code"
