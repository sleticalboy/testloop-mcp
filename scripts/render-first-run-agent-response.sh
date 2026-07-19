#!/usr/bin/env sh
set -eu

usage() {
  cat <<'USAGE'
Usage: scripts/render-first-run-agent-response.sh <first-run-artifact-dir>

Render the stable Agent response from a first-run artifact directory.

The directory must contain:
  - first-run-context.txt

The directory may also contain:
  - verification-summary.json

Environment:
  TESTLOOP_MCP_REPO_DIR  Path to a testloop-mcp checkout. Defaults to this script's repo.
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

artifact_dir="$1"
case "$artifact_dir" in
  /*) ;;
  *) artifact_dir="$(pwd)/$artifact_dir" ;;
esac

repo_root="${TESTLOOP_MCP_REPO_DIR:-$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)}"
context_file="${artifact_dir}/first-run-context.txt"
summary_file="${artifact_dir}/verification-summary.json"

if [ ! -d "$artifact_dir" ]; then
  echo "error: artifact directory does not exist: $artifact_dir" >&2
  exit 1
fi

if [ ! -f "$context_file" ]; then
  echo "error: artifact directory missing first-run-context.txt: $artifact_dir" >&2
  exit 1
fi

if [ -f "$summary_file" ]; then
  (cd "$repo_root" && go run ./examples/first-run-agent-response-demo "$context_file" "$summary_file")
else
  (cd "$repo_root" && go run ./examples/first-run-agent-response-demo "$context_file")
fi
