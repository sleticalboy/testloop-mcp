#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/verify-mcp-process-smoke.sh [testloop-mcp-binary]

Start a real testloop-mcp process through an MCP SDK client and call a
lightweight tool. This verifies the installed binary, transport wiring,
tools/list, parse_results, structuredContent, and text JSON fallback.

Arguments:
  testloop-mcp-binary  Optional binary path. Defaults to TESTLOOP_MCP_COMMAND,
                       then the testloop-mcp binary found on PATH.

Environment:
  TESTLOOP_MCP_COMMAND                  Binary path or command name to verify.
  TESTLOOP_MCP_CLIENT_SMOKE_TRANSPORT   all, stdio, or http. Default: all.

Examples:
  scripts/verify-mcp-process-smoke.sh
  scripts/verify-mcp-process-smoke.sh /opt/homebrew/bin/testloop-mcp
  TESTLOOP_MCP_CLIENT_SMOKE_TRANSPORT=stdio scripts/verify-mcp-process-smoke.sh
USAGE
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
command_path="${1:-${TESTLOOP_MCP_COMMAND:-}}"
transport="${TESTLOOP_MCP_CLIENT_SMOKE_TRANSPORT:-all}"

if [ -z "$command_path" ]; then
  command_path="$(command -v testloop-mcp 2>/dev/null || true)"
fi
[ -n "$command_path" ] || fail "testloop-mcp not found on PATH; pass a binary path or set TESTLOOP_MCP_COMMAND"

case "$command_path" in
  */*)
    [ -x "$command_path" ] || fail "binary is not executable: $command_path"
    ;;
  *)
    command -v "$command_path" >/dev/null 2>&1 || fail "binary not found on PATH: $command_path"
    ;;
esac

go run "$repo_root/examples/mcp-process-smoke" --command "$command_path" --transport "$transport"
