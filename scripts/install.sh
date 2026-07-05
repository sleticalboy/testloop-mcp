#!/usr/bin/env sh
set -eu

repo="sleticalboy/testloop-mcp"
version="${TESTLOOP_MCP_VERSION:-latest}"
install_dir="${TESTLOOP_MCP_INSTALL_DIR:-$HOME/.local/bin}"

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
    curl -fsSL "$url" -o "$out"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$out"
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

go_install_fallback() {
  need_cmd go
  mkdir -p "$install_dir"
  log "No matching release asset was found; falling back to go install."
  GOBIN="$install_dir" go install "github.com/${repo}@${version}"
  GOBIN="$install_dir" go install "github.com/${repo}/cmd/testgen@${version}"
  if [ -x "${install_dir}/testgen" ] && [ ! -e "${install_dir}/testloop-testgen" ]; then
    mv "${install_dir}/testgen" "${install_dir}/testloop-testgen"
  fi
  log "Installed testloop-mcp to ${install_dir}/testloop-mcp"
  log "Installed testloop-testgen to ${install_dir}/testloop-testgen"
}

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$os" in
  linux) os="linux" ;;
  darwin) os="darwin" ;;
  *) go_install_fallback; exit 0 ;;
esac

case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) go_install_fallback; exit 0 ;;
esac

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

asset="testloop-mcp_${version}_${os}_${arch}.tar.gz"
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
tar -xzf "${tmp_dir}/${asset}" -C "$tmp_dir"
install -m 755 "${tmp_dir}/testloop-mcp" "${install_dir}/testloop-mcp"
install -m 755 "${tmp_dir}/testloop-testgen" "${install_dir}/testloop-testgen"

log "Installed testloop-mcp to ${install_dir}/testloop-mcp"
log "Installed testloop-testgen to ${install_dir}/testloop-testgen"
