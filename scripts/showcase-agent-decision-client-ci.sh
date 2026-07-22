#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-agent-decision-client-ci.sh [--json]

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
  scripts/showcase-agent-decision-client-ci.sh --json
USAGE
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

output_format="text"
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    --json)
      output_format="json"
      shift
      ;;
    *)
      usage >&2
      exit 2
      ;;
  esac
done

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
set +e
(
  cd "$fixture_dir"
  npm test --silent > "$result_json"
)
npm_status=$?
set -e

node - "$result_json" "$client_dir" "$fixture_dir" "$output_format" "$npm_status" <<'JS'
const fs = require('node:fs');
const path = require('node:path');
const [resultPath, clientDir, fixtureDir, outputFormat, npmStatusRaw] = process.argv.slice(2);
const npmStatus = Number(npmStatusRaw);
const payload = JSON.parse(fs.readFileSync(resultPath, 'utf8'));
const resultSchemaPath = path.join(fixtureDir, 'docs/fixtures/agent-decision-fixtures-result.schema.json');
const summary = {
  schema_version: 1,
  status: payload.status,
  client_dir: clientDir,
  fixture_dir: fixtureDir,
  result_json: resultPath,
  result_schema: resultSchemaPath,
  fixture_count: payload.fixture_count,
  decisions: payload.decisions,
  failures: payload.failures || [],
  validator_exit_code: npmStatus,
};
if (outputFormat === 'json') {
  console.log(JSON.stringify(summary, null, 2));
} else {
  console.log(`agent_decision_client_status=${summary.status}`);
  console.log(`agent_decision_client_dir=${summary.client_dir}`);
  console.log(`agent_decision_fixture_dir=${summary.fixture_dir}`);
  console.log(`agent_decision_result_json=${summary.result_json}`);
  console.log(`agent_decision_result_schema=${summary.result_schema}`);
  console.log(`agent_decision_fixture_count=${summary.fixture_count}`);
  console.log(`agent_decision_decisions=${summary.decisions.join(',')}`);
}
if (payload.status !== 'passed' || npmStatus !== 0) {
  process.exitCode = 1;
}
JS
