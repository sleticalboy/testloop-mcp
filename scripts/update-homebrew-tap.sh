#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/update-homebrew-tap.sh [version] [tap-dir]

Updates the Homebrew tap formula from the selected GitHub Release.

Arguments:
  version  Release tag to use, for example v0.4.2. Defaults to latest.
  tap-dir  Existing tap checkout path. Defaults to TESTLOOP_MCP_TAP_DIR,
           or a temporary clone under ${TMPDIR:-/tmp}.

Environment:
  TESTLOOP_MCP_REPO             Source repo. Default: sleticalboy/testloop-mcp
  TESTLOOP_MCP_TAP_REPO         Tap repo. Default: sleticalboy/homebrew-tap
  TESTLOOP_MCP_TAP_DIR          Tap checkout path when tap-dir is omitted
  TESTLOOP_MCP_TAP_ALLOW_DIRTY  Set to 1 to allow an already-dirty tap checkout
  TESTLOOP_MCP_TAP_COMMIT       Set to 1 to commit Formula/testloop-mcp.rb
  TESTLOOP_MCP_TAP_PUSH         Set to 1 to push the tap commit
USAGE
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="${1:-${TESTLOOP_MCP_VERSION:-latest}}"
tap_dir="${2:-${TESTLOOP_MCP_TAP_DIR:-}}"
tap_repo="${TESTLOOP_MCP_TAP_REPO:-sleticalboy/homebrew-tap}"
allow_dirty="${TESTLOOP_MCP_TAP_ALLOW_DIRTY:-0}"
commit_changes="${TESTLOOP_MCP_TAP_COMMIT:-0}"
push_changes="${TESTLOOP_MCP_TAP_PUSH:-0}"

if ! command -v gh >/dev/null 2>&1; then
  echo "error: gh is required" >&2
  exit 1
fi

if [ -z "$tap_dir" ]; then
  tap_dir="${TMPDIR:-/tmp}/testloop-mcp-homebrew-tap"
fi

if [ ! -d "$tap_dir/.git" ]; then
  if [ -e "$tap_dir" ]; then
    echo "error: $tap_dir exists but is not a git checkout" >&2
    exit 1
  fi
  gh repo clone "$tap_repo" "$tap_dir"
else
  git -C "$tap_dir" fetch origin --prune
fi

if [ "$allow_dirty" != "1" ] && [ -n "$(git -C "$tap_dir" status --porcelain)" ]; then
  echo "error: tap checkout is dirty: $tap_dir" >&2
  echo "set TESTLOOP_MCP_TAP_ALLOW_DIRTY=1 to update it anyway" >&2
  exit 1
fi

formula_path="$tap_dir/Formula/testloop-mcp.rb"
mkdir -p "$(dirname "$formula_path")"

TESTLOOP_MCP_FORMULA_PATH="$formula_path" "$repo_root/scripts/generate-homebrew-formula.sh" "$version"
ruby -c "$formula_path"

if command -v brew >/dev/null 2>&1; then
  brew style "$formula_path"
else
  echo "warning: brew is not available; skipped Homebrew style check" >&2
fi

if [ "$commit_changes" = "1" ]; then
  git -C "$tap_dir" add Formula/testloop-mcp.rb
  if git -C "$tap_dir" diff --cached --quiet; then
    echo "No tap formula changes to commit"
  else
    formula_version="$(ruby -ne 'puts $1 if /^  version "([^"]+)"/' "$formula_path")"
    git -C "$tap_dir" commit -m "testloop-mcp ${formula_version}"
  fi
fi

if [ "$push_changes" = "1" ]; then
  git -C "$tap_dir" push origin HEAD
fi

echo "Updated $formula_path"
