#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-onboarding.sh [testloop-mcp-binary]

Run the public onboarding showcase:
  1. Verify the installed binary and generated client config snippets.
  2. Verify real MCP stdio and Streamable HTTP process transports.
  3. Run the minimal Agent feedback-loop demo.

Arguments:
  testloop-mcp-binary  Optional binary path. Defaults to TESTLOOP_MCP_COMMAND,
                       then the testloop-mcp binary found on PATH.

Environment:
  TESTLOOP_MCP_COMMAND                  Binary path or command name to verify.
  TESTLOOP_MCP_VERIFY_EXPECT_VERSION    Optional expected version, for example 0.5.6.
  TESTLOOP_MCP_VERIFY_HTTP_ADDR         HTTP listen address for setup verification.
                                        Default: 127.0.0.1:18081
  TESTLOOP_MCP_VERIFY_SKIP_HTTP         Set to true to skip setup HTTP /healthz.
  TESTLOOP_MCP_CLIENT_SMOKE_TRANSPORT   all, stdio, or http. Default: all.

Examples:
  scripts/showcase-onboarding.sh
  scripts/showcase-onboarding.sh /opt/homebrew/bin/testloop-mcp
  TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.6 scripts/showcase-onboarding.sh
USAGE
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

section() {
  printf '\n==> %s\n' "$*"
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

if [ "$#" -gt 1 ]; then
  usage >&2
  exit 2
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
command_path="${1:-${TESTLOOP_MCP_COMMAND:-}}"

if [ -z "$command_path" ]; then
  command_path="$(command -v testloop-mcp 2>/dev/null || true)"
fi
[ -n "$command_path" ] || fail "testloop-mcp not found on PATH; pass a binary path or set TESTLOOP_MCP_COMMAND"

section "basic install verification"
TESTLOOP_MCP_VERIFY_HTTP_ADDR="${TESTLOOP_MCP_VERIFY_HTTP_ADDR:-127.0.0.1:18081}" \
  "$repo_root/scripts/verify-client-setup.sh" "$command_path"

section "real MCP process transport verification"
"$repo_root/scripts/verify-mcp-process-smoke.sh" "$command_path"

section "minimal Agent feedback loop"
(
  cd "$repo_root"
  go run ./examples/mcp-client-demo
)

section "onboarding showcase passed"
