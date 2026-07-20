#!/usr/bin/env sh
set -eu

usage() {
  cat <<'USAGE'
Usage: scripts/verify-agent-artifact.sh <first-run|onboarding> <artifact-dir>
       scripts/verify-agent-artifact.sh manifest <agent-response-artifact-manifest.json>

Verify a downloaded testloop-mcp Agent artifact directory.

The verifier checks required files, validates verification-summary.json against
the local verification-summary.schema.json, and confirms decision/response
actions agree with the failed section.
Manifest mode verifies every artifact listed in the manifest and checks the
manifest expectations against each artifact directory.

Environment:
  TESTLOOP_MCP_REPO_DIR  Path to a testloop-mcp checkout. Defaults to this script's repo.
USAGE
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

if [ "$#" -ne 2 ]; then
  usage >&2
  exit 2
fi

kind="$1"
artifact_dir="$2"
case "$artifact_dir" in
  /*) ;;
  *) artifact_dir="$(pwd)/$artifact_dir" ;;
esac

repo_root="${TESTLOOP_MCP_REPO_DIR:-$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)}"

(cd "$repo_root" && go run ./examples/agent-artifact-verify "$kind" "$artifact_dir")
