#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/verify-release-assets.sh TAG

Verifies that a GitHub Release contains all required binary assets and .sha256
files for the supported release matrix.

Example:
  scripts/verify-release-assets.sh v0.4.10

Environment:
  TESTLOOP_MCP_REPO  GitHub repo. Default: sleticalboy/testloop-mcp
USAGE
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

if [ "$#" -ne 1 ]; then
  usage >&2
  exit 2
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "error: gh is required" >&2
  exit 1
fi

repo="${TESTLOOP_MCP_REPO:-sleticalboy/testloop-mcp}"
tag="$1"

asset_names="$(gh release view "$tag" --repo "$repo" --json assets --jq '.assets[].name')"

required_assets=(
  "testloop-mcp_${tag}_darwin_arm64.tar.gz"
  "testloop-mcp_${tag}_darwin_arm64.tar.gz.sha256"
  "testloop-mcp_${tag}_linux_amd64.tar.gz"
  "testloop-mcp_${tag}_linux_amd64.tar.gz.sha256"
  "testloop-mcp_${tag}_linux_arm64.tar.gz"
  "testloop-mcp_${tag}_linux_arm64.tar.gz.sha256"
  "testloop-mcp_${tag}_windows_amd64.zip"
  "testloop-mcp_${tag}_windows_amd64.zip.sha256"
  "testloop-mcp_${tag}_windows_arm64.zip"
  "testloop-mcp_${tag}_windows_arm64.zip.sha256"
)

missing=()
for asset in "${required_assets[@]}"; do
  if ! grep -Fx -- "$asset" <<<"$asset_names" >/dev/null 2>&1; then
    missing+=("$asset")
  fi
done

if [ "${#missing[@]}" -gt 0 ]; then
  echo "error: release $repo@$tag is missing required assets:" >&2
  for asset in "${missing[@]}"; do
    echo "  - $asset" >&2
  done
  exit 1
fi

echo "Verified ${#required_assets[@]} release assets for $repo@$tag"
