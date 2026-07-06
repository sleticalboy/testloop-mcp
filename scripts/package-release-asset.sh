#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/package-release-asset.sh TAG ASSET GOOS GOARCH

Builds release binaries and writes an archive plus .sha256 file to dist/.

Example:
  scripts/package-release-asset.sh v0.4.3 linux_amd64 linux amd64

Environment:
  TESTLOOP_MCP_DIST_DIR  Output directory. Default: dist
  CGO_ENABLED            CGO setting for go build. Default: 1
USAGE
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

if [ "$#" -ne 4 ]; then
  usage >&2
  exit 2
fi

tag="$1"
asset="$2"
goos="$3"
goarch="$4"
dist_dir="${TESTLOOP_MCP_DIST_DIR:-dist}"
cgo_enabled="${CGO_ENABLED:-1}"

checksum() {
  file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file"
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file"
  else
    echo "error: missing sha256sum or shasum" >&2
    exit 1
  fi
}

mkdir -p "$dist_dir"
package_dir="${dist_dir}/package-${asset}"
rm -rf "$package_dir"
mkdir -p "$package_dir"
trap 'rm -rf "$package_dir"' EXIT INT TERM

binary_suffix=""
archive_ext="tar.gz"
if [ "$goos" = "windows" ]; then
  binary_suffix=".exe"
  archive_ext="zip"
fi

CGO_ENABLED="$cgo_enabled" GOOS="$goos" GOARCH="$goarch" \
  go build -trimpath -ldflags="-s -w" -o "${package_dir}/testloop-mcp${binary_suffix}" .
CGO_ENABLED="$cgo_enabled" GOOS="$goos" GOARCH="$goarch" \
  go build -trimpath -ldflags="-s -w" -o "${package_dir}/testloop-testgen${binary_suffix}" ./cmd/testgen

cp README.md LICENSE "$package_dir/"

archive="testloop-mcp_${tag}_${asset}.${archive_ext}"
rm -f "${dist_dir}/${archive}" "${dist_dir}/${archive}.sha256"

if [ "$archive_ext" = "zip" ]; then
  (cd "$package_dir" && zip -qr "../${archive}" .)
else
  tar -czf "${dist_dir}/${archive}" -C "$package_dir" .
fi

(cd "$dist_dir" && checksum "$archive" > "${archive}.sha256")
echo "Wrote ${dist_dir}/${archive}"
echo "Wrote ${dist_dir}/${archive}.sha256"
