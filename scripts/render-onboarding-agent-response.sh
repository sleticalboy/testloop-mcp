#!/usr/bin/env sh
set -eu

usage() {
  cat <<'USAGE'
Usage: scripts/render-onboarding-agent-response.sh <onboarding-artifact-dir>

Render the stable Agent response from an onboarding artifact directory.

The directory must contain:
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
summary_file="${artifact_dir}/verification-summary.json"

if [ ! -d "$artifact_dir" ]; then
  echo "error: artifact directory does not exist: $artifact_dir" >&2
  exit 1
fi

if [ ! -f "$summary_file" ]; then
  echo "error: artifact directory missing verification-summary.json: $artifact_dir" >&2
  exit 1
fi

(cd "$repo_root" && go run ./examples/onboarding-agent-response-demo "$summary_file")
