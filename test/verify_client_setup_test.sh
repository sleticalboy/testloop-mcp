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

test_verify_client_setup_passes_with_skip_http() {
  out="${tmp_dir}/verify.out"
  TESTLOOP_MCP_VERIFY_SKIP_HTTP=true \
    bash "${repo_root}/scripts/verify-client-setup.sh" "$binary" > "$out"

  assert_contains "$out" "==> binary: $binary"
  assert_contains "$out" "==> version"
  assert_contains "$out" "==> doctor-config"
  assert_contains "$out" "==> print-config/check-config roundtrip"
  assert_contains "$out" "==> HTTP health check skipped"
  assert_contains "$out" "==> client setup verification passed"
}

test_verify_client_setup_checks_expected_version() {
  out="${tmp_dir}/verify-version.out"
  TESTLOOP_MCP_VERIFY_SKIP_HTTP=true \
    TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.1 \
    bash "${repo_root}/scripts/verify-client-setup.sh" "$binary" > "$out"

  assert_contains "$out" "==> version"
  assert_contains "$out" "==> client setup verification passed"
}

test_verify_client_setup_rejects_version_mismatch() {
  out="${tmp_dir}/version-mismatch.out"
  set +e
  TESTLOOP_MCP_VERIFY_SKIP_HTTP=true \
    TESTLOOP_MCP_VERIFY_EXPECT_VERSION=9.9.9 \
    bash "${repo_root}/scripts/verify-client-setup.sh" "$binary" > "$out" 2>&1
  code=$?
  set -e

  if [ "$code" -eq 0 ]; then
    echo "expected version mismatch verification to fail" >&2
    exit 1
  fi
  assert_contains "$out" "error: version mismatch: expected 9.9.9, got 0.5.1"
}

test_verify_client_setup_rejects_missing_binary() {
  out="${tmp_dir}/missing.out"
  set +e
  bash "${repo_root}/scripts/verify-client-setup.sh" "${tmp_dir}/missing-testloop-mcp" > "$out" 2>&1
  code=$?
  set -e

  if [ "$code" -eq 0 ]; then
    echo "expected missing binary verification to fail" >&2
    exit 1
  fi
  assert_contains "$out" "error: binary is not executable:"
}

test_verify_client_setup_passes_with_skip_http
test_verify_client_setup_checks_expected_version
test_verify_client_setup_rejects_version_mismatch
test_verify_client_setup_rejects_missing_binary

echo "client setup verification tests passed"
