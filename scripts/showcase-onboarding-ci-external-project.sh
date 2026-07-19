#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-onboarding-ci-external-project.sh

Create a temporary Go or Node project outside this repository and run the
onboarding CI bootstrap from that project directory. This verifies the
copy-paste onboarding path does not rely on the user project being the
testloop-mcp checkout.

Environment:
  TESTLOOP_EXTERNAL_ONBOARDING_WORKDIR     Parent temp dir. Default: /tmp/testloop-external-onboarding
  TESTLOOP_EXTERNAL_ONBOARDING_OUTPUT_DIR  Artifact dir. Default: <workdir>/artifacts
  TESTLOOP_EXTERNAL_ONBOARDING_BOOTSTRAP   Bootstrap script path. Default: <workdir>/testloop-onboarding-ci.sh
  TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE go, node, or all. Default: go.
  TESTLOOP_MCP_COMMAND                     Existing testloop-mcp binary path/command.
  TESTLOOP_MCP_VERSION                     Expected binary version. Default: v0.5.10
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
output_dir="${TESTLOOP_EXTERNAL_ONBOARDING_OUTPUT_DIR:-$workdir/artifacts}"
bootstrap="${TESTLOOP_EXTERNAL_ONBOARDING_BOOTSTRAP:-$workdir/testloop-onboarding-ci.sh}"
project_type="${TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE:-go}"
version="${TESTLOOP_MCP_VERSION:-v0.5.10}"
repo_dir="${TESTLOOP_MCP_REPO_DIR:-$repo_root}"

case "$project_type" in
  go|node|all)
    ;;
  *)
    printf 'error: unsupported TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE: %s\n' "$project_type" >&2
    exit 1
    ;;
esac

rm -rf "$workdir"
mkdir -p "$output_dir" "$(dirname -- "$bootstrap")"

cp "$repo_root/scripts/run-onboarding-ci.sh" "$bootstrap"
chmod +x "$bootstrap"

create_go_project() {
  local project_dir="$1"
  mkdir -p "$project_dir"
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
}

create_node_project() {
  local project_dir="$1"
  command -v pnpm >/dev/null 2>&1 || {
    printf 'error: pnpm is required for node external onboarding\n' >&2
    exit 1
  }
  mkdir -p "$project_dir"
  cat >"$project_dir/package.json" <<'EOF_PACKAGE_JSON'
{
  "name": "testloop-onboarding-external-web",
  "private": true,
  "type": "module",
  "scripts": {
    "build": "node build.js"
  }
}
EOF_PACKAGE_JSON

  cat >"$project_dir/build.js" <<'EOF_JS'
import { mkdirSync, writeFileSync } from "node:fs";

mkdirSync("dist", { recursive: true });
writeFileSync("dist/index.html", "<!doctype html><title>testloop onboarding</title>\n");
console.log("web build ok");
EOF_JS

  (
    cd "$project_dir"
    pnpm install --lockfile-only >/dev/null
  )
}

run_external_onboarding() {
  local kind="$1"
  local project_dir="$2"
  local project_command="$3"
  local artifacts_dir="$4"

  mkdir -p "$artifacts_dir"
  (
    cd "$project_dir"
    env \
      TESTLOOP_MCP_REPO_DIR="$repo_dir" \
      TESTLOOP_MCP_VERSION="$version" \
      TESTLOOP_ONBOARDING_PROJECT_DIR="$project_dir" \
      TESTLOOP_ONBOARDING_OUTPUT_DIR="$artifacts_dir" \
      TESTLOOP_REPORT_PUBLIC_SHOWCASES=none \
      "$bootstrap" "$project_command"
  )

  local summary_json="$artifacts_dir/verification-summary.json"
  local decision_out="$artifacts_dir/agent-decision.txt"
  local agent_response_out="$artifacts_dir/agent-response.txt"

  python3 - "$summary_json" "$decision_out" "$agent_response_out" <<'PY'
import json
import sys
from pathlib import Path

summary = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
decision = Path(sys.argv[2]).read_text(encoding="utf-8")
response_path = Path(sys.argv[3])

if summary.get("overall_status") != "passed":
    raise SystemExit(f"expected overall_status=passed, got {summary.get('overall_status')!r}")
if summary.get("failed_count") != 0:
    raise SystemExit(f"expected failed_count=0, got {summary.get('failed_count')!r}")
if "agent_next_step=ready" not in decision:
    raise SystemExit("expected agent_next_step=ready")
if not response_path.is_file():
    raise SystemExit(f"expected agent-response.txt to exist: {response_path}")
response = response_path.read_text(encoding="utf-8")
if "agent_next_step=ready" not in response:
    raise SystemExit("expected agent-response.txt to contain agent_next_step=ready")
PY

  printf 'external_onboarding_%s_project=%s\n' "$kind" "$project_dir"
  printf 'external_onboarding_%s_output_dir=%s\n' "$kind" "$artifacts_dir"
  printf 'external_onboarding_%s_summary=%s\n' "$kind" "$summary_json"
  printf 'external_onboarding_%s_decision=%s\n' "$kind" "$decision_out"
  printf 'external_onboarding_%s_agent_response=%s\n' "$kind" "$agent_response_out"
  printf 'external_onboarding_%s_status=passed\n' "$kind"

  if [[ "$project_type" != "all" ]]; then
    printf 'external_onboarding_project=%s\n' "$project_dir"
    printf 'external_onboarding_output_dir=%s\n' "$artifacts_dir"
    printf 'external_onboarding_summary=%s\n' "$summary_json"
    printf 'external_onboarding_decision=%s\n' "$decision_out"
    printf 'external_onboarding_agent_response=%s\n' "$agent_response_out"
  fi
}

case "$project_type" in
  go)
    create_go_project "$workdir/project-go"
    run_external_onboarding go "$workdir/project-go" 'go test ./...' "$output_dir"
    ;;
  node)
    create_node_project "$workdir/project-node"
    run_external_onboarding node "$workdir/project-node" 'pnpm install --frozen-lockfile && pnpm build' "$output_dir"
    ;;
  all)
    create_go_project "$workdir/project-go"
    create_node_project "$workdir/project-node"
    run_external_onboarding go "$workdir/project-go" 'go test ./...' "$output_dir/go"
    run_external_onboarding node "$workdir/project-node" 'pnpm install --frozen-lockfile && pnpm build' "$output_dir/node"
    ;;
esac

printf 'external_onboarding_mode=%s\n' "$project_type"
printf 'external_onboarding_status=passed\n'
