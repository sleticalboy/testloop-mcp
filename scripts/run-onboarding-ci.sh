#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/run-onboarding-ci.sh [project-smoke-command]

Bootstrap a CI-friendly testloop-mcp onboarding report for a user project.
The script installs or resolves testloop-mcp, prepares a testloop-mcp source
checkout for report helpers, then writes:
  - verification-report.md
  - verification-summary.json
  - agent-decision.txt
  - agent-response.txt

Arguments:
  project-smoke-command  Optional smoke command. Defaults to
                         TESTLOOP_ONBOARDING_PROJECT_COMMAND.

Environment:
  TESTLOOP_ONBOARDING_PROJECT_DIR      Project directory. Default: current dir.
  TESTLOOP_ONBOARDING_PROJECT_COMMAND  Smoke command run in the project dir.
  TESTLOOP_ONBOARDING_OUTPUT_DIR       Output dir. Default: /tmp/testloop-onboarding
  TESTLOOP_ONBOARDING_AGENT_RESPONSE_OUT
                                       Agent response output path. Defaults to
                                       $TESTLOOP_ONBOARDING_OUTPUT_DIR/agent-response.txt
  TESTLOOP_ONBOARDING_TITLE            Report title.

  TESTLOOP_MCP_COMMAND                 Existing testloop-mcp binary path/command.
  TESTLOOP_MCP_VERSION                 Binary version to install. Default: latest.
  TESTLOOP_MCP_VERIFY_EXPECT_VERSION   Expected version gate. Defaults to
                                       TESTLOOP_MCP_VERSION without a leading v
                                       when TESTLOOP_MCP_VERSION is not latest.
  TESTLOOP_MCP_REPO_DIR                Existing testloop-mcp source checkout.
  TESTLOOP_MCP_REPO_REF                Source ref to clone. Default: main, or
                                       TESTLOOP_MCP_VERSION when it is not latest.
  TESTLOOP_MCP_INSTALL_DIR             Install dir. Default: $HOME/.local/bin
  TESTLOOP_MCP_REPO_URL                Source repo URL.

Examples:
  TESTLOOP_MCP_VERSION=v0.5.6 scripts/run-onboarding-ci.sh 'go test ./...'
  TESTLOOP_ONBOARDING_PROJECT_COMMAND='pnpm install --frozen-lockfile && pnpm build' \
    scripts/run-onboarding-ci.sh
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
project_dir="${TESTLOOP_ONBOARDING_PROJECT_DIR:-$PWD}"
project_command="${1:-${TESTLOOP_ONBOARDING_PROJECT_COMMAND:-}}"
output_dir="${TESTLOOP_ONBOARDING_OUTPUT_DIR:-/tmp/testloop-onboarding}"
title="${TESTLOOP_ONBOARDING_TITLE:-testloop-mcp onboarding report}"
version="${TESTLOOP_MCP_VERSION:-latest}"
install_dir="${TESTLOOP_MCP_INSTALL_DIR:-$HOME/.local/bin}"
repo_url="${TESTLOOP_MCP_REPO_URL:-https://github.com/sleticalboy/testloop-mcp.git}"
repo_ref="${TESTLOOP_MCP_REPO_REF:-}"
repo_dir="${TESTLOOP_MCP_REPO_DIR:-}"
command_path="${TESTLOOP_MCP_COMMAND:-}"
expect_version="${TESTLOOP_MCP_VERIFY_EXPECT_VERSION:-}"

[[ -n "$project_command" ]] || fail "project smoke command is required; pass it as the first argument or set TESTLOOP_ONBOARDING_PROJECT_COMMAND"
[[ -d "$project_dir" ]] || fail "project directory does not exist: $project_dir"

if [[ -z "$repo_ref" ]]; then
  if [[ -n "$version" && "$version" != "latest" ]]; then
    repo_ref="$version"
  else
    repo_ref="main"
  fi
fi

if [[ -z "$expect_version" && -n "$version" && "$version" != "latest" ]]; then
  expect_version="${version#v}"
fi

has_report_helpers() {
  local candidate="$1"
  [[ -x "$candidate/scripts/showcase-agent-onboarding-report.sh" && -x "$candidate/scripts/install.sh" ]]
}

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

if [[ -n "$repo_dir" ]]; then
  has_report_helpers "$repo_dir" || fail "TESTLOOP_MCP_REPO_DIR is not a testloop-mcp checkout with required scripts: $repo_dir"
elif [[ -n "$script_root" && -d "$script_root" ]] && has_report_helpers "$script_root"; then
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
report_md="${TESTLOOP_ONBOARDING_REPORT_MD:-${output_dir}/verification-report.md}"
summary_json="${TESTLOOP_ONBOARDING_SUMMARY_JSON:-${output_dir}/verification-summary.json}"
decision_out="${TESTLOOP_ONBOARDING_DECISION_OUT:-${output_dir}/agent-decision.txt}"
agent_response_out="${TESTLOOP_ONBOARDING_AGENT_RESPONSE_OUT:-${output_dir}/agent-response.txt}"

printf 'testloop_mcp_repo=%s\n' "$repo_dir"
printf 'testloop_mcp_binary=%s\n' "$binary"
printf 'testloop_onboarding_output_dir=%s\n' "$output_dir"
printf 'testloop_project_dir=%s\n' "$project_dir"
printf 'testloop_project_command=%s\n' "$project_command"

env_args=(
  "TESTLOOP_ONBOARDING_OUTPUT_DIR=$output_dir"
  "TESTLOOP_REPORT_TITLE=$title"
  "TESTLOOP_REPORT_PROJECT_DIR=$project_dir"
  "TESTLOOP_REPORT_PROJECT_COMMAND=$project_command"
)
if [[ -n "$expect_version" ]]; then
  env_args+=("TESTLOOP_MCP_VERIFY_EXPECT_VERSION=$expect_version")
fi

write_github_step_summary() {
  [[ -n "${GITHUB_STEP_SUMMARY:-}" ]] || return 0

  local overall_status="unknown"
  local failed_count="unknown"
  local agent_next_step="unknown"

  if [[ -s "$summary_json" ]]; then
    read -r overall_status failed_count < <(
      python3 - "$summary_json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    data = json.load(fh)
print(data.get("overall_status", "unknown"), data.get("failed_count", "unknown"))
PY
    )
  fi

  if [[ -s "$decision_out" ]]; then
    agent_next_step="$(grep -E '^agent_next_step=' "$decision_out" | tail -n 1 | cut -d= -f2- || true)"
    [[ -n "$agent_next_step" ]] || agent_next_step="unknown"
  fi

  {
    printf '## testloop-mcp onboarding\n\n'
    printf '%s%s%s\n' '- Status: `' "$overall_status" '`'
    printf '%s%s%s\n' '- Failed sections: `' "$failed_count" '`'
    printf '%s%s%s\n' '- agent_next_step: `' "$agent_next_step" '`'
    printf '%s%s%s\n' '- Markdown report: `' "$report_md" '`'
    printf '%s%s%s\n' '- Summary JSON: `' "$summary_json" '`'
    printf '%s%s%s\n' '- Agent decision: `' "$decision_out" '`'
    if [[ -s "$agent_response_out" ]]; then
      printf '%s%s%s\n' '- Agent response: `' "$agent_response_out" '`'
    fi
    printf '\n'
    case "$agent_next_step" in
      ready)
        printf '%s\n' 'Next: continue with the real generation, repair, or coverage loop.'
        ;;
      fix-installation)
        printf '%s\n' 'Next: inspect binary path, version gate, config roundtrip, and HTTP health output.'
        ;;
      inspect-user-project)
        printf '%s\n' 'Next: inspect the user project smoke section in the Markdown report.'
        ;;
      inspect-mcp-transport)
        printf '%s\n' 'Next: inspect stdio / Streamable HTTP MCP startup and transport configuration.'
        ;;
      inspect-agent-demo)
        printf '%s\n' 'Next: inspect structured feedback loop and demo runner output.'
        ;;
      inspect-showcase)
        printf '%s\n' 'Next: inspect external network, showcase checkout, or action expectation drift.'
        ;;
      *)
        printf '%s\n' 'Next: download the onboarding artifacts and inspect verification-summary.json first.'
        ;;
    esac
    printf '\n'
  } >>"$GITHUB_STEP_SUMMARY"
}

render_agent_response() {
  [[ -s "$summary_json" ]] || return 0
  [[ -x "$repo_dir/scripts/render-onboarding-agent-response.sh" ]] || return 0
  if ! command -v go >/dev/null 2>&1; then
    printf 'warning: go is not available; skipped agent-response.txt rendering\n' >&2
    return 0
  fi

  local tmp_response="$tmp_dir/agent-response.txt"
  if ! TESTLOOP_MCP_REPO_DIR="$repo_dir" \
    "$repo_dir/scripts/render-onboarding-agent-response.sh" "$output_dir" > "$tmp_response"; then
    printf 'warning: failed to render agent-response.txt; continuing with onboarding result\n' >&2
    return 0
  fi
  mv "$tmp_response" "$agent_response_out"
}

set +e
env "${env_args[@]}" "$repo_dir/scripts/showcase-agent-onboarding-report.sh" "$binary"
onboarding_code=$?
set -e

render_agent_response
write_github_step_summary
exit "$onboarding_code"
