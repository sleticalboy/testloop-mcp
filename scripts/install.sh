#!/usr/bin/env sh
set -eu

repo="sleticalboy/testloop-mcp"
version="${TESTLOOP_MCP_VERSION:-latest}"
install_dir="${TESTLOOP_MCP_INSTALL_DIR:-$HOME/.local/bin}"
binary_suffix=""
download_retries="${TESTLOOP_MCP_DOWNLOAD_RETRIES:-3}"
download_connect_timeout="${TESTLOOP_MCP_CONNECT_TIMEOUT:-15}"
download_max_time="${TESTLOOP_MCP_DOWNLOAD_MAX_TIME:-300}"

log() {
  printf '%s\n' "$*"
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

download() {
  url="$1"
  out="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fL --retry "$download_retries" --retry-delay 2 \
      --connect-timeout "$download_connect_timeout" \
      --max-time "$download_max_time" \
      -sS "$url" -o "$out"
  elif command -v wget >/dev/null 2>&1; then
    wget -q --tries="$download_retries" --timeout="$download_connect_timeout" "$url" -O "$out"
  else
    fail "missing curl or wget"
  fi
}

checksum_cmd() {
  if command -v sha256sum >/dev/null 2>&1; then
    printf 'sha256sum'
  elif command -v shasum >/dev/null 2>&1; then
    printf 'shasum -a 256'
  else
    fail "missing sha256sum or shasum"
  fi
}

extract_archive() {
  archive="$1"
  dest="$2"
  case "$archive" in
    *.zip)
      if command -v unzip >/dev/null 2>&1; then
        unzip -q "$archive" -d "$dest"
      elif command -v bsdtar >/dev/null 2>&1; then
        bsdtar -xf "$archive" -C "$dest"
      else
        return 1
      fi
      ;;
    *.tar.gz)
      tar -xzf "$archive" -C "$dest"
      ;;
    *)
      return 1
      ;;
  esac
}

go_install_fallback() {
  need_cmd go
  mkdir -p "$install_dir"
  log "No matching release asset was found; falling back to go install."
  GOBIN="$install_dir" go install "github.com/${repo}@${version}"
  GOBIN="$install_dir" go install "github.com/${repo}/cmd/testgen@${version}"
  if [ -x "${install_dir}/testgen${binary_suffix}" ] && [ ! -e "${install_dir}/testloop-testgen${binary_suffix}" ]; then
    mv "${install_dir}/testgen${binary_suffix}" "${install_dir}/testloop-testgen${binary_suffix}"
  fi
  log "Installed testloop-mcp to ${install_dir}/testloop-mcp${binary_suffix}"
  log "Installed testloop-testgen to ${install_dir}/testloop-testgen${binary_suffix}"
}

os="${TESTLOOP_MCP_OS:-}"
arch="${TESTLOOP_MCP_ARCH:-}"
if [ -z "$os" ]; then
  os="$(uname -s)"
fi
if [ -z "$arch" ]; then
  arch="$(uname -m)"
fi
os="$(printf '%s' "$os" | tr '[:upper:]' '[:lower:]')"

case "$os" in
  linux) os="linux" ;;
  darwin) os="darwin" ;;
  windows) os="windows" ;;
  mingw*|msys*|cygwin*) os="windows" ;;
  *) go_install_fallback; exit 0 ;;
esac

if [ "$os" = "windows" ]; then
  binary_suffix=".exe"
fi

case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) go_install_fallback; exit 0 ;;
esac

archive_ext="tar.gz"
if [ "$os" = "windows" ]; then
  archive_ext="zip"
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

if [ "$version" = "latest" ]; then
  if download "https://api.github.com/repos/${repo}/releases/latest" "${tmp_dir}/latest.json"; then
    version="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "${tmp_dir}/latest.json" | head -n 1)"
  fi
  if [ -z "$version" ] || [ "$version" = "latest" ]; then
    go_install_fallback
    exit 0
  fi
fi

asset="testloop-mcp_${version}_${os}_${arch}.${archive_ext}"
base_url="https://github.com/${repo}/releases/download/${version}"

if ! download "${base_url}/${asset}" "${tmp_dir}/${asset}" 2>/dev/null; then
  go_install_fallback
  exit 0
fi

(
  cd "$tmp_dir"
  if download "${base_url}/checksums.txt" checksums.txt 2>/dev/null; then
    grep "  ${asset}\$" checksums.txt > selected-checksum.txt || fail "checksum for ${asset} not found"
  elif download "${base_url}/${asset}.sha256" selected-checksum.txt 2>/dev/null; then
    :
  else
    fail "checksum for ${asset} not found"
  fi
  cmd="$(checksum_cmd)"
  $cmd -c selected-checksum.txt
)

mkdir -p "$install_dir"
if ! extract_archive "${tmp_dir}/${asset}" "$tmp_dir"; then
  log "No supported extractor was found for ${asset}; falling back to go install."
  go_install_fallback
  exit 0
fi
install -m 755 "${tmp_dir}/testloop-mcp${binary_suffix}" "${install_dir}/testloop-mcp${binary_suffix}"
install -m 755 "${tmp_dir}/testloop-testgen${binary_suffix}" "${install_dir}/testloop-testgen${binary_suffix}"

log "Installed testloop-mcp to ${install_dir}/testloop-mcp${binary_suffix}"
log "Installed testloop-testgen to ${install_dir}/testloop-testgen${binary_suffix}"
