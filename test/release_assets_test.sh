#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

fake_bin="${tmp_dir}/bin"
mkdir -p "$fake_bin"

cat > "${fake_bin}/gh" <<'EOF'
#!/usr/bin/env sh
set -eu

printf '%s\n' "$*" >> "${TESTLOOP_FAKE_GH_LOG:?}"

tag=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    view)
      shift
      tag="$1"
      ;;
  esac
  shift || true
done

case "${TESTLOOP_FAKE_RELEASE_MODE:-complete}" in
  complete)
    cat <<ASSETS
testloop-mcp_${tag}_darwin_arm64.tar.gz
testloop-mcp_${tag}_darwin_arm64.tar.gz.sha256
testloop-mcp_${tag}_linux_amd64.tar.gz
testloop-mcp_${tag}_linux_amd64.tar.gz.sha256
testloop-mcp_${tag}_linux_arm64.tar.gz
testloop-mcp_${tag}_linux_arm64.tar.gz.sha256
testloop-mcp_${tag}_windows_amd64.zip
testloop-mcp_${tag}_windows_amd64.zip.sha256
testloop-mcp_${tag}_windows_arm64.zip
testloop-mcp_${tag}_windows_arm64.zip.sha256
ASSETS
    ;;
  missing)
    cat <<ASSETS
testloop-mcp_${tag}_darwin_arm64.tar.gz
testloop-mcp_${tag}_darwin_arm64.tar.gz.sha256
testloop-mcp_${tag}_linux_amd64.tar.gz
testloop-mcp_${tag}_linux_amd64.tar.gz.sha256
ASSETS
    ;;
  *)
    echo "unknown fake release mode" >&2
    exit 2
    ;;
esac
EOF
chmod +x "${fake_bin}/gh"

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

assert_not_contains() {
  file="$1"
  needle="$2"
  if grep -F -- "$needle" "$file" >/dev/null 2>&1; then
    echo "expected $file to not contain: $needle" >&2
    echo "--- $file ---" >&2
    cat "$file" >&2
    exit 1
  fi
}

test_complete_release_assets_pass() {
  gh_log="${tmp_dir}/gh-complete.log"
  out="${tmp_dir}/complete.out"
  : > "$gh_log"

  PATH="${fake_bin}:$PATH" \
    TESTLOOP_FAKE_GH_LOG="$gh_log" \
    TESTLOOP_FAKE_RELEASE_MODE="complete" \
    TESTLOOP_MCP_REPO="example/testloop-mcp" \
    bash "${repo_root}/scripts/verify-release-assets.sh" v9.9.9 > "$out"

  assert_contains "$out" "Verified 10 release assets for example/testloop-mcp@v9.9.9"
  assert_contains "$gh_log" "release view v9.9.9 --repo example/testloop-mcp --json assets --jq .assets[].name"
}

test_missing_release_assets_fail() {
  gh_log="${tmp_dir}/gh-missing.log"
  out="${tmp_dir}/missing.out"
  : > "$gh_log"

  set +e
  PATH="${fake_bin}:$PATH" \
    TESTLOOP_FAKE_GH_LOG="$gh_log" \
    TESTLOOP_FAKE_RELEASE_MODE="missing" \
    bash "${repo_root}/scripts/verify-release-assets.sh" v9.9.9 > "$out" 2>&1
  code=$?
  set -e

  if [ "$code" -eq 0 ]; then
    echo "expected missing release assets check to fail" >&2
    exit 1
  fi
  assert_contains "$out" "error: release sleticalboy/testloop-mcp@v9.9.9 is missing required assets:"
  assert_contains "$out" "testloop-mcp_v9.9.9_windows_arm64.zip.sha256"
}

test_release_help_checks_expect_exit_zero() {
  assert_contains "${repo_root}/.github/workflows/release.yml" 'test "$mcp_status" -eq 0'
  assert_contains "${repo_root}/.github/workflows/release.yml" 'test "$testgen_status" -eq 0'
  assert_contains "${repo_root}/.github/workflows/post-release-verify.yml" 'test "$mcp_status" -eq 0'
  assert_contains "${repo_root}/.github/workflows/post-release-verify.yml" 'test "$testgen_status" -eq 0'
  assert_contains "${repo_root}/.github/workflows/windows-arm64-probe.yml" 'test "$mcp_status" -eq 0'
  assert_contains "${repo_root}/.github/workflows/windows-arm64-probe.yml" 'test "$testgen_status" -eq 0'
  assert_not_contains "${repo_root}/.github/workflows/release.yml" 'test "$mcp_status" -eq 2'
  assert_not_contains "${repo_root}/.github/workflows/post-release-verify.yml" 'test "$mcp_status" -eq 2'
  assert_not_contains "${repo_root}/.github/workflows/windows-arm64-probe.yml" 'test "$mcp_status" -eq 2'
  assert_contains "${repo_root}/scripts/generate-homebrew-formula.sh" 'shell_output("#{bin}/testloop-mcp --help 2>&1")'
  assert_contains "${repo_root}/scripts/generate-homebrew-formula.sh" 'shell_output("#{bin}/testloop-testgen --help 2>&1")'
  assert_not_contains "${repo_root}/scripts/generate-homebrew-formula.sh" 'shell_output("#{bin}/testloop-mcp --help 2>&1", 2)'
}

test_complete_release_assets_pass
test_missing_release_assets_fail
test_release_help_checks_expect_exit_zero

echo "release asset tests passed"
