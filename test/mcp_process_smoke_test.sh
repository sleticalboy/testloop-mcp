#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

binary="${tmp_dir}/testloop-mcp"
if [ "${GOOS:-}" = "windows" ]; then
  binary="${binary}.exe"
fi

(cd "$repo_root" && go build -o "$binary" .)

assert_contains() {
  file="$1"
  needle="$2"
  if ! grep -F -- "$needle" "$file" >/dev/null 2>&1; then
    echo "expected $file to contain: $needle" >&2
    echo "--- $file ---" >&2
    cat "$file" >&2
    exit 1
  fi
}

out="${tmp_dir}/smoke.out"
TESTLOOP_MCP_CLIENT_SMOKE_TRANSPORT=all \
  bash "${repo_root}/scripts/verify-mcp-process-smoke.sh" "$binary" > "$out"

assert_contains "$out" "stdio: tools="
assert_contains "$out" "http: tools="
assert_contains "$out" "parse_results=pass"
assert_contains "$out" "structuredContent=ok"
assert_contains "$out" "client_smoke=pass"

echo "mcp process smoke test passed"
