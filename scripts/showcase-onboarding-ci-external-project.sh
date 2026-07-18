#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-onboarding-ci-external-project.sh

Create a temporary Go project outside this repository and run the onboarding CI
bootstrap from that project directory. This verifies the copy-paste onboarding
path does not rely on the user project being the testloop-mcp checkout.

Environment:
  TESTLOOP_EXTERNAL_ONBOARDING_WORKDIR     Parent temp dir. Default: /tmp/testloop-external-onboarding
  TESTLOOP_EXTERNAL_ONBOARDING_OUTPUT_DIR  Artifact dir. Default: <workdir>/artifacts
  TESTLOOP_EXTERNAL_ONBOARDING_BOOTSTRAP   Bootstrap script path. Default: <workdir>/testloop-onboarding-ci.sh
  TESTLOOP_MCP_COMMAND                     Existing testloop-mcp binary path/command.
  TESTLOOP_MCP_VERSION                     Expected binary version. Default: v0.5.6
  TESTLOOP_MCP_REPO_DIR                    Existing testloop-mcp source checkout. Default: current repo.

Examples:
  go build -o /tmp/testloop-mcp .
  TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp scripts/showcase-onboarding-ci-external-project.sh
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ "$#" -ne 0 ]]; then
  usage >&2
  exit 2
fi

repo_root="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
workdir="${TESTLOOP_EXTERNAL_ONBOARDING_WORKDIR:-/tmp/testloop-external-onboarding}"
project_dir="$workdir/project-go"
output_dir="${TESTLOOP_EXTERNAL_ONBOARDING_OUTPUT_DIR:-$workdir/artifacts}"
bootstrap="${TESTLOOP_EXTERNAL_ONBOARDING_BOOTSTRAP:-$workdir/testloop-onboarding-ci.sh}"
version="${TESTLOOP_MCP_VERSION:-v0.5.6}"
repo_dir="${TESTLOOP_MCP_REPO_DIR:-$repo_root}"

rm -rf "$workdir"
mkdir -p "$project_dir" "$output_dir" "$(dirname -- "$bootstrap")"

cp "$repo_root/scripts/run-onboarding-ci.sh" "$bootstrap"
chmod +x "$bootstrap"

cat >"$project_dir/go.mod" <<'EOF_GO_MOD'
module example.com/testloop-onboarding-external

go 1.22
EOF_GO_MOD

cat >"$project_dir/calculator.go" <<'EOF_GO'
package calculator

func Add(left, right int) int {
	return left + right
}
EOF_GO

cat >"$project_dir/calculator_test.go" <<'EOF_GO_TEST'
package calculator

import "testing"

func TestAdd(t *testing.T) {
	if got := Add(2, 3); got != 5 {
		t.Fatalf("Add(2, 3) = %d, want 5", got)
	}
}
EOF_GO_TEST

(
  cd "$project_dir"
  env \
    TESTLOOP_MCP_REPO_DIR="$repo_dir" \
    TESTLOOP_MCP_VERSION="$version" \
    TESTLOOP_ONBOARDING_PROJECT_DIR="$project_dir" \
    TESTLOOP_ONBOARDING_OUTPUT_DIR="$output_dir" \
    TESTLOOP_REPORT_PUBLIC_SHOWCASES=none \
    "$bootstrap" 'go test ./...'
)

summary_json="$output_dir/verification-summary.json"
decision_out="$output_dir/agent-decision.txt"

python3 - "$summary_json" "$decision_out" <<'PY'
import json
import sys
from pathlib import Path

summary = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
decision = Path(sys.argv[2]).read_text(encoding="utf-8")

if summary.get("overall_status") != "passed":
    raise SystemExit(f"expected overall_status=passed, got {summary.get('overall_status')!r}")
if summary.get("failed_count") != 0:
    raise SystemExit(f"expected failed_count=0, got {summary.get('failed_count')!r}")
if "agent_next_step=ready" not in decision:
    raise SystemExit("expected agent_next_step=ready")
PY

printf 'external_onboarding_project=%s\n' "$project_dir"
printf 'external_onboarding_output_dir=%s\n' "$output_dir"
printf 'external_onboarding_summary=%s\n' "$summary_json"
printf 'external_onboarding_decision=%s\n' "$decision_out"
printf 'external_onboarding_status=passed\n'
