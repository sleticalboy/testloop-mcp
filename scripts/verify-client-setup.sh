#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/verify-client-setup.sh [testloop-mcp-binary]

Verify that a local testloop-mcp binary is ready for MCP client setup.

Checks:
  1. The binary exists and can print a client config snippet.
  2. --doctor-config can inspect local client config paths.
  3. --print-config=all output can be validated by --check-config -.
  4. HTTP mode can start and /healthz returns ok.

Arguments:
  testloop-mcp-binary  Optional binary path. Defaults to TESTLOOP_MCP_COMMAND,
                       then the testloop-mcp binary found on PATH.

Environment:
  TESTLOOP_MCP_COMMAND             Binary path to verify.
  TESTLOOP_MCP_VERIFY_HTTP_ADDR    HTTP listen address for the health check.
                                   Default: 127.0.0.1:18080
  TESTLOOP_MCP_VERIFY_SKIP_HTTP    Set to true to skip the HTTP health check.

Examples:
  scripts/verify-client-setup.sh
  scripts/verify-client-setup.sh /opt/homebrew/bin/testloop-mcp
  TESTLOOP_MCP_VERIFY_SKIP_HTTP=true scripts/verify-client-setup.sh
USAGE
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

log() {
  printf '==> %s\n' "$*"
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

command_path="${1:-${TESTLOOP_MCP_COMMAND:-}}"
http_addr="${TESTLOOP_MCP_VERIFY_HTTP_ADDR:-127.0.0.1:18080}"
skip_http="${TESTLOOP_MCP_VERIFY_SKIP_HTTP:-false}"

resolve_binary() {
  if [ -n "$command_path" ]; then
    case "$command_path" in
      */*)
        [ -x "$command_path" ] || fail "binary is not executable: $command_path"
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

wait_for_healthz() {
  url="$1"
  attempts=50
  i=1
  while [ "$i" -le "$attempts" ]; do
    if command -v curl >/dev/null 2>&1; then
      if [ "$(curl -fsS "$url" 2>/dev/null || true)" = "ok" ]; then
        return 0
      fi
    else
      if command -v python3 >/dev/null 2>&1; then
        if python3 - "$url" <<'PY' >/dev/null 2>&1
import sys
from urllib.request import urlopen

with urlopen(sys.argv[1], timeout=0.25) as response:
    body = response.read().decode("utf-8").strip()
    if response.status != 200 or body != "ok":
        raise SystemExit(1)
PY
        then
          return 0
        fi
      else
        fail "curl or python3 is required for HTTP health check; set TESTLOOP_MCP_VERIFY_SKIP_HTTP=true to skip"
      fi
    fi
    sleep 0.1
    i=$((i + 1))
  done
  return 1
}

binary="$(resolve_binary)"
http_url="http://${http_addr}/mcp"

log "binary: $binary"
"$binary" --print-config=codex --config-command="$binary" >/dev/null

log "doctor-config"
"$binary" --doctor-config >/dev/null

log "print-config/check-config roundtrip"
config_output="$("$binary" --print-config=all --config-command="$binary" --config-http-url="$http_url")"
printf '%s\n' "$config_output" | "$binary" --check-config - >/dev/null

if [ "$skip_http" = "true" ]; then
  log "HTTP health check skipped"
else
  log "HTTP health check: http://${http_addr}/healthz"
  "$binary" --transport=http --addr="$http_addr" >/tmp/testloop-mcp-verify-http.log 2>&1 &
  server_pid=$!
  cleanup() {
    if kill -0 "$server_pid" >/dev/null 2>&1; then
      kill "$server_pid" >/dev/null 2>&1 || true
      wait "$server_pid" >/dev/null 2>&1 || true
    fi
  }
  trap cleanup EXIT INT TERM
  if ! wait_for_healthz "http://${http_addr}/healthz"; then
    tail -n 50 /tmp/testloop-mcp-verify-http.log >&2 || true
    fail "HTTP health check failed"
  fi
  cleanup
  trap - EXIT INT TERM
fi

log "client setup verification passed"
