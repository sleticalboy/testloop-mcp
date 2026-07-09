#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

fake_bin="${tmp_dir}/bin"
fixture_dir="${tmp_dir}/fixtures"
mkdir -p "$fake_bin" "$fixture_dir"

asset_name="testloop-mcp_v9.9.9_windows_amd64.zip"
asset_path="${fixture_dir}/${asset_name}"
printf 'fake windows zip payload\n' > "$asset_path"

if command -v sha256sum >/dev/null 2>&1; then
  asset_sha="$(sha256sum "$asset_path" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  asset_sha="$(shasum -a 256 "$asset_path" | awk '{print $1}')"
else
  echo "missing sha256sum or shasum" >&2
  exit 1
fi
printf '%s  %s\n' "$asset_sha" "$asset_name" > "${fixture_dir}/${asset_name}.sha256"

cat > "${fake_bin}/curl" <<'EOF'
#!/usr/bin/env sh
set -eu

printf '%s\n' "$*" >> "${TESTLOOP_FAKE_CURL_LOG:?}"

out=""
url=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o)
      shift
      out="$1"
      ;;
    http://*|https://*)
      url="$1"
      ;;
  esac
  shift || true
done

if [ -z "$out" ] || [ -z "$url" ]; then
  echo "fake curl requires URL and -o" >&2
  exit 2
fi

case "$url" in
  */testloop-mcp_v9.9.9_windows_amd64.zip)
    cp "${TESTLOOP_FAKE_FIXTURES}/testloop-mcp_v9.9.9_windows_amd64.zip" "$out"
    ;;
  */checksums.txt)
    exit 22
    ;;
  */testloop-mcp_v9.9.9_windows_amd64.zip.sha256)
    cp "${TESTLOOP_FAKE_FIXTURES}/testloop-mcp_v9.9.9_windows_amd64.zip.sha256" "$out"
    ;;
  *)
    exit 22
    ;;
esac
EOF
chmod +x "${fake_bin}/curl"

cat > "${fake_bin}/unzip" <<'EOF'
#!/usr/bin/env sh
set -eu

dest=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -d)
      shift
      dest="$1"
      ;;
  esac
  shift || true
done

if [ -z "$dest" ]; then
  echo "fake unzip requires -d" >&2
  exit 2
fi

printf '#!/usr/bin/env sh\nexit 0\n' > "${dest}/testloop-mcp.exe"
printf '#!/usr/bin/env sh\nexit 0\n' > "${dest}/testloop-testgen.exe"
chmod +x "${dest}/testloop-mcp.exe" "${dest}/testloop-testgen.exe"
EOF
chmod +x "${fake_bin}/unzip"

cat > "${fake_bin}/go" <<'EOF'
#!/usr/bin/env sh
set -eu

printf '%s\n' "$*" >> "${TESTLOOP_FAKE_GO_LOG:?}"
case "$*" in
  *github.com/sleticalboy/testloop-mcp/cmd/testgen@*)
    printf '#!/usr/bin/env sh\nexit 0\n' > "${GOBIN:?}/testgen"
    chmod +x "${GOBIN}/testgen"
    ;;
  *github.com/sleticalboy/testloop-mcp@*)
    printf '#!/usr/bin/env sh\nexit 0\n' > "${GOBIN:?}/testloop-mcp"
    chmod +x "${GOBIN}/testloop-mcp"
    ;;
  *)
    echo "unexpected fake go invocation: $*" >&2
    exit 2
    ;;
esac
EOF
chmod +x "${fake_bin}/go"

assert_file() {
  file="$1"
  if [ ! -f "$file" ]; then
    echo "expected file to exist: $file" >&2
    exit 1
  fi
}

assert_log_contains() {
  file="$1"
  needle="$2"
  if ! grep -F -- "$needle" "$file" >/dev/null 2>&1; then
    echo "expected $file to contain: $needle" >&2
    echo "--- $file ---" >&2
    cat "$file" >&2
    exit 1
  fi
}

test_windows_zip_sha256_fallback() {
  install_dir="${tmp_dir}/install-windows"
  curl_log="${tmp_dir}/curl-windows.log"
  : > "$curl_log"

  PATH="${fake_bin}:$PATH" \
    TESTLOOP_FAKE_CURL_LOG="$curl_log" \
    TESTLOOP_FAKE_FIXTURES="$fixture_dir" \
    TESTLOOP_MCP_VERSION="v9.9.9" \
    TESTLOOP_MCP_OS="windows" \
    TESTLOOP_MCP_ARCH="amd64" \
    TESTLOOP_MCP_INSTALL_DIR="$install_dir" \
    TESTLOOP_MCP_DOWNLOAD_RETRIES="7" \
    TESTLOOP_MCP_CONNECT_TIMEOUT="4" \
    TESTLOOP_MCP_DOWNLOAD_MAX_TIME="9" \
    sh "${repo_root}/scripts/install.sh" >/tmp/testloop-install-windows.out

  assert_file "${install_dir}/testloop-mcp.exe"
  assert_file "${install_dir}/testloop-testgen.exe"
  assert_log_contains "$curl_log" "--retry 7"
  assert_log_contains "$curl_log" "--connect-timeout 4"
  assert_log_contains "$curl_log" "--max-time 9"
  assert_log_contains "$curl_log" "checksums.txt"
  assert_log_contains "$curl_log" "${asset_name}.sha256"
}

test_go_install_fallback_renames_testgen() {
  install_dir="${tmp_dir}/install-fallback"
  go_log="${tmp_dir}/go-fallback.log"
  curl_log="${tmp_dir}/curl-fallback.log"
  : > "$go_log"
  : > "$curl_log"

  PATH="${fake_bin}:$PATH" \
    TESTLOOP_FAKE_GO_LOG="$go_log" \
    TESTLOOP_FAKE_CURL_LOG="$curl_log" \
    TESTLOOP_FAKE_FIXTURES="$fixture_dir" \
    TESTLOOP_MCP_VERSION="v9.9.9" \
    TESTLOOP_MCP_OS="plan9" \
    TESTLOOP_MCP_ARCH="amd64" \
    TESTLOOP_MCP_INSTALL_DIR="$install_dir" \
    sh "${repo_root}/scripts/install.sh" >/tmp/testloop-install-fallback.out

  assert_file "${install_dir}/testloop-mcp"
  assert_file "${install_dir}/testloop-testgen"
  if [ -e "${install_dir}/testgen" ]; then
    echo "expected fallback to rename testgen to testloop-testgen" >&2
    exit 1
  fi
  assert_log_contains "$go_log" "install github.com/sleticalboy/testloop-mcp@v9.9.9"
  assert_log_contains "$go_log" "install github.com/sleticalboy/testloop-mcp/cmd/testgen@v9.9.9"
  assert_log_contains /tmp/testloop-install-fallback.out "Unsupported OS 'plan9'. Falling back to go install."
}

test_download_failure_fallback_message() {
  install_dir="${tmp_dir}/install-download-fallback"
  go_log="${tmp_dir}/go-download-fallback.log"
  curl_log="${tmp_dir}/curl-download-fallback.log"
  : > "$go_log"
  : > "$curl_log"

  PATH="${fake_bin}:$PATH" \
    TESTLOOP_FAKE_GO_LOG="$go_log" \
    TESTLOOP_FAKE_CURL_LOG="$curl_log" \
    TESTLOOP_FAKE_FIXTURES="$fixture_dir" \
    TESTLOOP_MCP_VERSION="v9.9.9" \
    TESTLOOP_MCP_OS="darwin" \
    TESTLOOP_MCP_ARCH="arm64" \
    TESTLOOP_MCP_INSTALL_DIR="$install_dir" \
    sh "${repo_root}/scripts/install.sh" >/tmp/testloop-install-download-fallback.out

  assert_file "${install_dir}/testloop-mcp"
  assert_file "${install_dir}/testloop-testgen"
  assert_log_contains "$curl_log" "testloop-mcp_v9.9.9_darwin_arm64.tar.gz"
  assert_log_contains "$go_log" "install github.com/sleticalboy/testloop-mcp@v9.9.9"
  assert_log_contains /tmp/testloop-install-download-fallback.out "Failed to download release asset testloop-mcp_v9.9.9_darwin_arm64.tar.gz"
  assert_log_contains /tmp/testloop-install-download-fallback.out "check that the asset exists and that the network can reach GitHub. Falling back to go install."
}

test_windows_download_failure_fallback_logs_actual_go_install_paths() {
  install_dir="${tmp_dir}/install-windows-download-fallback"
  go_log="${tmp_dir}/go-windows-download-fallback.log"
  curl_log="${tmp_dir}/curl-windows-download-fallback.log"
  : > "$go_log"
  : > "$curl_log"

  PATH="${fake_bin}:$PATH" \
    TESTLOOP_FAKE_GO_LOG="$go_log" \
    TESTLOOP_FAKE_CURL_LOG="$curl_log" \
    TESTLOOP_FAKE_FIXTURES="$fixture_dir" \
    TESTLOOP_MCP_VERSION="v9.9.9" \
    TESTLOOP_MCP_OS="windows" \
    TESTLOOP_MCP_ARCH="arm64" \
    TESTLOOP_MCP_INSTALL_DIR="$install_dir" \
    sh "${repo_root}/scripts/install.sh" >/tmp/testloop-install-windows-download-fallback.out

  assert_file "${install_dir}/testloop-mcp"
  assert_file "${install_dir}/testloop-testgen"
  if [ -e "${install_dir}/testloop-mcp.exe" ] || [ -e "${install_dir}/testloop-testgen.exe" ]; then
    echo "expected go install fallback to log and keep current-host binary names" >&2
    exit 1
  fi
  assert_log_contains "$curl_log" "testloop-mcp_v9.9.9_windows_arm64.zip"
  assert_log_contains /tmp/testloop-install-windows-download-fallback.out "Failed to download release asset testloop-mcp_v9.9.9_windows_arm64.zip"
  assert_log_contains /tmp/testloop-install-windows-download-fallback.out "Installed testloop-mcp to ${install_dir}/testloop-mcp"
  assert_log_contains /tmp/testloop-install-windows-download-fallback.out "Installed testloop-testgen to ${install_dir}/testloop-testgen"
}

test_windows_zip_sha256_fallback
test_go_install_fallback_renames_testgen
test_download_failure_fallback_message
test_windows_download_failure_fallback_logs_actual_go_install_paths

echo "install script tests passed"
