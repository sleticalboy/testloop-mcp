#!/usr/bin/env bash
set -euo pipefail

repo="${TESTLOOP_MCP_REPO:-sleticalboy/testloop-mcp}"
version="${1:-${TESTLOOP_MCP_VERSION:-latest}}"
output="${TESTLOOP_MCP_FORMULA_PATH:-Formula/testloop-mcp.rb}"

if ! command -v gh >/dev/null 2>&1; then
  echo "error: gh is required" >&2
  exit 1
fi

if [ "$version" = "latest" ]; then
  version="$(gh release view --repo "$repo" --json tagName --jq '.tagName')"
fi

tag="$version"
plain_version="${version#v}"

asset_digest() {
  asset="$1"
  digest="$(gh release view "$tag" --repo "$repo" --json assets --jq ".assets[] | select(.name == \"$asset\") | .digest")"
  if [ -z "$digest" ]; then
    echo "error: missing asset digest for $asset" >&2
    exit 1
  fi
  printf '%s' "${digest#sha256:}"
}

darwin_arm64_asset="testloop-mcp_${tag}_darwin_arm64.tar.gz"
linux_amd64_asset="testloop-mcp_${tag}_linux_amd64.tar.gz"
linux_arm64_asset="testloop-mcp_${tag}_linux_arm64.tar.gz"

darwin_arm64_sha="$(asset_digest "$darwin_arm64_asset")"
linux_amd64_sha="$(asset_digest "$linux_amd64_asset")"
linux_arm64_sha="$(asset_digest "$linux_arm64_asset")"

mkdir -p "$(dirname "$output")"
cat > "$output" <<FORMULA
class TestloopMcp < Formula
  desc "MCP server for AI coding test feedback loops"
  homepage "https://github.com/${repo}"
  version "${plain_version}"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/${repo}/releases/download/${tag}/${darwin_arm64_asset}"
      sha256 "${darwin_arm64_sha}"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/${repo}/releases/download/${tag}/${linux_amd64_asset}"
      sha256 "${linux_amd64_sha}"
    end

    on_arm do
      url "https://github.com/${repo}/releases/download/${tag}/${linux_arm64_asset}"
      sha256 "${linux_arm64_sha}"
    end
  end

  def install
    bin.install "testloop-mcp"
    bin.install "testloop-testgen"
  end

  test do
    assert_match "Usage of testloop-mcp", shell_output("#{bin}/testloop-mcp --help 2>&1")
    assert_match "Usage: testgen", shell_output("#{bin}/testloop-testgen --help 2>&1")
  end
end
FORMULA

echo "Wrote $output for $tag"
