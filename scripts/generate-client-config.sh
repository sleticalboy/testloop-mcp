#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/generate-client-config.sh [client] [command]

Print MCP client configuration snippets for testloop-mcp.

This source-checkout helper mirrors:
  testloop-mcp --print-config=<client>

Arguments:
  client   codex, claude, cursor, or all. Defaults to all.
  command  Path to the testloop-mcp binary. Defaults to TESTLOOP_MCP_COMMAND,
           or the testloop-mcp binary found on PATH.

Environment:
  TESTLOOP_MCP_COMMAND   Binary path used in stdio snippets.
  TESTLOOP_MCP_HTTP_URL  HTTP MCP endpoint for Codex URL snippets.
                         Default: http://localhost:8080/mcp

Examples:
  scripts/generate-client-config.sh
  scripts/generate-client-config.sh codex /opt/homebrew/bin/testloop-mcp
  TESTLOOP_MCP_HTTP_URL=http://localhost:8080/mcp scripts/generate-client-config.sh codex-http
USAGE
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

client="${1:-all}"
command_path="${2:-${TESTLOOP_MCP_COMMAND:-}}"
http_url="${TESTLOOP_MCP_HTTP_URL:-http://localhost:8080/mcp}"

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

toml_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

resolve_command() {
  if [ -n "$command_path" ]; then
    case "$command_path" in
      */*)
        dir="$(cd "$(dirname "$command_path")" && pwd)"
        printf '%s/%s' "$dir" "$(basename "$command_path")"
        ;;
      *)
        if resolved="$(command -v "$command_path" 2>/dev/null)"; then
          printf '%s' "$resolved"
        else
          printf '%s' "$command_path"
        fi
        ;;
    esac
    return
  fi

  if resolved="$(command -v testloop-mcp 2>/dev/null)"; then
    printf '%s' "$resolved"
    return
  fi

  fail "testloop-mcp not found on PATH; pass the binary path as the second argument or set TESTLOOP_MCP_COMMAND"
}

emit_codex() {
  cmd="$(toml_escape "$(resolve_command)")"
  cat <<CONFIG
# ~/.codex/config.toml
[mcp_servers.testloop]
command = "$cmd"
CONFIG
}

emit_codex_http() {
  url="$(toml_escape "$http_url")"
  cat <<CONFIG
# ~/.codex/config.toml
[mcp_servers.testloop]
url = "$url"
CONFIG
}

emit_claude() {
  cmd="$(json_escape "$(resolve_command)")"
  cat <<CONFIG
# ~/.claude/claude_desktop_config.json
{
  "mcpServers": {
    "testloop": {
      "command": "$cmd"
    }
  }
}
CONFIG
}

emit_cursor() {
  cmd="$(json_escape "$(resolve_command)")"
  cat <<CONFIG
# .cursor/mcp.json
{
  "mcpServers": {
    "testloop": {
      "command": "$cmd"
    }
  }
}
CONFIG
}

emit_all() {
  emit_codex
  printf '\n---\n\n'
  emit_codex_http
  printf '\n---\n\n'
  emit_claude
  printf '\n---\n\n'
  emit_cursor
}

case "$client" in
  all)
    emit_all
    ;;
  codex)
    emit_codex
    ;;
  codex-http | http)
    emit_codex_http
    ;;
  claude | claude-desktop | claude-code)
    emit_claude
    ;;
  cursor)
    emit_cursor
    ;;
  *)
    usage >&2
    fail "unknown client: $client"
    ;;
esac
