#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-agent-decision-client-ci.sh

Create a minimal external-client CI example for Agent decision fixtures:
  1. Export the testloop-mcp Agent decision fixture package.
  2. Run the exported package's npm test script.
  3. Write the validator JSON result and print a stable summary.

Environment:
  TESTLOOP_AGENT_DECISION_CLIENT_DIR       Client project directory.
                                             Default: a fresh temp directory.
  TESTLOOP_AGENT_DECISION_FIXTURE_DIR      Exported fixture package directory.
                                             Default: <client-dir>/testloop-agent-decision-fixtures
  TESTLOOP_AGENT_DECISION_RESULT_JSON      Validator JSON output path.
                                             Default: <client-dir>/agent-decision-fixtures-result.json

Example:
  scripts/showcase-agent-decision-client-ci.sh
USAGE
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
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
if [[ -n "${TESTLOOP_AGENT_DECISION_CLIENT_DIR:-}" ]]; then
  client_dir="$TESTLOOP_AGENT_DECISION_CLIENT_DIR"
else
  client_dir="$(mktemp -d "${TMPDIR:-/tmp}/testloop-agent-decision-client.XXXXXX")"
fi
fixture_dir="${TESTLOOP_AGENT_DECISION_FIXTURE_DIR:-${client_dir}/testloop-agent-decision-fixtures}"
result_json="${TESTLOOP_AGENT_DECISION_RESULT_JSON:-${client_dir}/agent-decision-fixtures-result.json}"

command -v node >/dev/null 2>&1 || fail "missing required command: node"
command -v npm >/dev/null 2>&1 || fail "missing required command: npm"

[[ ! -e "$client_dir" || -d "$client_dir" ]] || fail "client dir path must be a directory: $client_dir"
[[ ! -e "$fixture_dir" || -d "$fixture_dir" ]] || fail "fixture package path must be a directory: $fixture_dir"
[[ ! -d "$result_json" ]] || fail "result JSON path must not be a directory: $result_json"
if [[ -d "$fixture_dir" && -n "$(find "$fixture_dir" -mindepth 1 -maxdepth 1 -print -quit)" ]]; then
  fail "fixture package directory must be empty: $fixture_dir"
fi

mkdir -p "$client_dir" "$(dirname "$result_json")"

node "$repo_root/scripts/export-agent-decision-fixtures.mjs" "$fixture_dir" >/dev/null
(
  cd "$fixture_dir"
  npm test --silent > "$result_json"
)

node - "$result_json" "$client_dir" "$fixture_dir" <<'JS'
const fs = require('node:fs');
const [resultPath, clientDir, fixtureDir] = process.argv.slice(2);
const payload = JSON.parse(fs.readFileSync(resultPath, 'utf8'));
console.log(`agent_decision_client_status=${payload.status}`);
console.log(`agent_decision_client_dir=${clientDir}`);
console.log(`agent_decision_fixture_dir=${fixtureDir}`);
console.log(`agent_decision_result_json=${resultPath}`);
console.log(`agent_decision_fixture_count=${payload.fixture_count}`);
console.log(`agent_decision_decisions=${payload.decisions.join(',')}`);
if (payload.status !== 'passed') {
  process.exitCode = 1;
}
JS
