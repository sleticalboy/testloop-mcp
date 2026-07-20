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
    TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.14 \
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
  assert_contains "$out" "error: version mismatch: expected 9.9.9, got 0.5.14"
  assert_contains "$out" "brew upgrade sleticalboy/tap/testloop-mcp"
}

test_verify_client_setup_explains_missing_version_flag() {
  old_binary="${tmp_dir}/old-testloop-mcp"
  cat > "$old_binary" <<'SH'
#!/usr/bin/env sh
case "${1:-}" in
  --print-config=codex)
    exit 0
    ;;
  --version)
    echo "flag provided but not defined: -version" >&2
    exit 2
    ;;
  *)
    exit 0
    ;;
esac
SH
  chmod +x "$old_binary"

  out="${tmp_dir}/old-version.out"
  set +e
  TESTLOOP_MCP_VERIFY_SKIP_HTTP=true \
    TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.14 \
    bash "${repo_root}/scripts/verify-client-setup.sh" "$old_binary" > "$out" 2>&1
  code=$?
  set -e

  if [ "$code" -eq 0 ]; then
    echo "expected old binary verification to fail" >&2
    exit 1
  fi
  assert_contains "$out" "error: --version failed for $old_binary"
  assert_contains "$out" "flag provided but not defined: -version"
  assert_contains "$out" "brew upgrade sleticalboy/tap/testloop-mcp"
  assert_contains "$out" "brew reinstall sleticalboy/tap/testloop-mcp"
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
  assert_contains "$out" "error: binary must be an executable file:"
}

test_verify_client_setup_passes_with_skip_http
test_verify_client_setup_checks_expected_version
test_verify_client_setup_rejects_version_mismatch
test_verify_client_setup_explains_missing_version_flag
test_verify_client_setup_rejects_missing_binary

echo "client setup verification tests passed"
