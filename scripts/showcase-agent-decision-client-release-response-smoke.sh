#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-agent-decision-client-release-response-smoke.sh [--json]

Create a standalone external-client project that consumes release smoke summary:
  1. Produce or reuse a release smoke summary JSON.
  2. Copy the release response renderer into a temporary client project.
  3. Run the client project's npm test script.
  4. Print a stable summary for CI and Agent consumption.

Environment:
  TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_CLIENT_DIR    External client directory.
                                                        Default: a fresh temp directory.
  TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_SUMMARY_JSON  Existing release smoke summary JSON.
                                                        Default: run release smoke and write a temp summary.
  TESTLOOP_AGENT_DECISION_RELEASE_INSTALLER_URL          Passed through when producing the release smoke summary.
  TESTLOOP_AGENT_DECISION_RELEASE_VERSION                Passed through when producing the release smoke summary.

Examples:
  scripts/showcase-agent-decision-client-release-response-smoke.sh
  scripts/showcase-agent-decision-client-release-response-smoke.sh --json
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
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/testloop-agent-decision-release-response-smoke.XXXXXX")"
client_dir="${TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_CLIENT_DIR:-${tmp_dir}/external-client}"
source_summary_json="${TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_SUMMARY_JSON:-${tmp_dir}/release-smoke-summary.json}"
client_summary_json="${client_dir}/testloop-release-smoke-summary.json"
agent_response_json="${client_dir}/testloop-release-response.json"

command -v node >/dev/null 2>&1 || fail "missing required command: node"
command -v npm >/dev/null 2>&1 || fail "missing required command: npm"

[[ ! -e "$client_dir" || -d "$client_dir" ]] || fail "client dir path must be a directory: $client_dir"
[[ ! -d "$source_summary_json" ]] || fail "summary JSON path must not be a directory: $source_summary_json"

if [[ -z "${TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_SUMMARY_JSON:-}" ]]; then
  bash "${repo_root}/scripts/showcase-agent-decision-client-release-smoke.sh" --json > "$source_summary_json"
fi
[[ -f "$source_summary_json" ]] || fail "release smoke summary JSON does not exist: $source_summary_json"

mkdir -p "$client_dir/scripts"
cp "$source_summary_json" "$client_summary_json"
cp "${repo_root}/scripts/render-agent-decision-client-release-response.mjs" \
  "$client_dir/scripts/render-agent-decision-client-release-response.mjs"
chmod +x "$client_dir/scripts/render-agent-decision-client-release-response.mjs"

cat > "$client_dir/package.json" <<'JSON'
{
  "name": "testloop-agent-decision-release-response-client",
  "private": true,
  "type": "module",
  "scripts": {
    "test": "node scripts/render-agent-decision-client-release-response.mjs --json testloop-release-smoke-summary.json > testloop-release-response.json && node scripts/assert-release-response.mjs testloop-release-response.json"
  }
}
JSON

cat > "$client_dir/scripts/assert-release-response.mjs" <<'JS'
import fs from 'node:fs';
import process from 'node:process';

const responsePath = process.argv[2];
if (!responsePath) {
  console.error('Usage: node scripts/assert-release-response.mjs <response-json>');
  process.exit(2);
}

const payload = JSON.parse(fs.readFileSync(responsePath, 'utf8'));
const expectedDecisions = [
  'accept',
  'accept',
  'accept',
  'manual-review',
  'manual-review',
  'manual-review',
  'apply-repair',
  'needs-better-input',
];
const failures = [];

if (payload.schema_version !== 1) {
  failures.push('schema_version must be 1');
}
if (payload.status !== 'passed') {
  failures.push(`status=${payload.status || 'missing'}, want passed`);
}
if (payload.agent_next_step !== 'ready') {
  failures.push(`agent_next_step=${payload.agent_next_step || 'missing'}, want ready`);
}
if (!payload.evidence || typeof payload.evidence.release_ref !== 'string' || payload.evidence.release_ref.length === 0) {
  failures.push('evidence.release_ref is required');
}
if (!payload.evidence || payload.evidence.fixture_count !== expectedDecisions.length) {
  failures.push(`evidence.fixture_count=${payload.evidence?.fixture_count}, want ${expectedDecisions.length}`);
}
if (JSON.stringify(payload.evidence?.decisions || []) !== JSON.stringify(expectedDecisions)) {
  failures.push('evidence.decisions drifted');
}
if (payload.evidence?.agent_next_steps?.client !== 'ready') {
  failures.push('evidence.agent_next_steps.client must be ready');
}
if (payload.evidence?.agent_next_steps?.consumer !== 'ready') {
  failures.push('evidence.agent_next_steps.consumer must be ready');
}
if (!Array.isArray(payload.failures) || payload.failures.length > 0) {
  failures.push('payload.failures must be an empty array');
}

if (failures.length > 0) {
  console.error(failures.join('\n'));
  process.exit(1);
}
JS

set +e
(
  cd "$client_dir"
  npm test --silent >/dev/null
)
npm_exit_code=$?
set -e

node - \
  "$client_dir" \
  "$client_summary_json" \
  "$agent_response_json" \
  "$output_format" \
  "$npm_exit_code" <<'JS'
const fs = require('node:fs');
const [clientDir, summaryPath, responsePath, outputFormat, npmExitCodeRaw] = process.argv.slice(2);
const npmExitCode = Number(npmExitCodeRaw);
const failures = [];

function readJSONIfExists(filePath) {
  if (!fs.existsSync(filePath)) {
    return {};
  }
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

const summary = readJSONIfExists(summaryPath);
const response = readJSONIfExists(responsePath);

if (summary.status !== 'passed') {
  failures.push(`release summary status=${summary.status || 'missing'}, want passed`);
}
if (response.status !== 'passed') {
  failures.push(`agent response status=${response.status || 'missing'}, want passed`);
}
if (response.agent_next_step !== 'ready') {
  failures.push(`agent response next step=${response.agent_next_step || 'missing'}, want ready`);
}
if (npmExitCode !== 0) {
  failures.push(`npm test exit code=${npmExitCode}`);
}

const payload = {
  schema_version: 1,
  status: failures.length === 0 ? 'passed' : 'failed',
  client_dir: clientDir,
  release_summary_json: summaryPath,
  agent_response_json: responsePath,
  release_ref: summary.release_ref || response.evidence?.release_ref || '',
  fixture_count: summary.fixture_count || response.evidence?.fixture_count || 0,
  decisions: summary.decisions || response.evidence?.decisions || [],
  agent_next_step: response.agent_next_step || '',
  npm_exit_code: npmExitCode,
  failures,
};

if (outputFormat === 'json') {
  console.log(JSON.stringify(payload, null, 2));
} else {
  console.log(`agent_decision_client_release_response_smoke_status=${payload.status}`);
  console.log(`agent_decision_client_release_response_smoke_client_dir=${payload.client_dir}`);
  console.log(`agent_decision_client_release_response_smoke_summary_json=${payload.release_summary_json}`);
  console.log(`agent_decision_client_release_response_smoke_agent_response_json=${payload.agent_response_json}`);
  console.log(`agent_decision_client_release_response_smoke_release_ref=${payload.release_ref}`);
  console.log(`agent_decision_client_release_response_smoke_fixture_count=${payload.fixture_count}`);
  console.log(`agent_decision_client_release_response_smoke_decisions=${payload.decisions.join(',')}`);
  console.log(`agent_decision_client_release_response_smoke_agent_next_step=${payload.agent_next_step}`);
}

if (payload.status !== 'passed') {
  process.exitCode = 1;
}
JS
