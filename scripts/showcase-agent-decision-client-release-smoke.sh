#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-agent-decision-client-release-smoke.sh [--json]

Run the post-release external-client Agent decision smoke bundle:
  1. Install the client CI workflow from the release tag raw installer.
  2. Render the basic client CI summary into an Agent response.
  3. Run the consumer smoke and verify its Agent response.

Environment:
  TESTLOOP_AGENT_DECISION_RELEASE_VERSION        Release ref. Default: v<main.go appVersion>.
  TESTLOOP_AGENT_DECISION_RELEASE_INSTALLER_URL  Installer URL. Default: raw GitHub URL for the release ref.
  TESTLOOP_AGENT_DECISION_RELEASE_HELPER_DIR     testloop-mcp helper checkout directory. Default: this repository.
  TESTLOOP_AGENT_DECISION_RELEASE_CLIENT_DIR     External client directory for installer smoke.
  TESTLOOP_AGENT_DECISION_RELEASE_CONSUMER_DIR   External client directory for consumer smoke.
  TESTLOOP_AGENT_DECISION_CI_DOWNLOAD_RETRIES    curl retry count for raw installer. Default: 3.
  TESTLOOP_AGENT_DECISION_CI_DOWNLOAD_MAX_TIME    curl max time in seconds. Default: 120.

Examples:
  scripts/showcase-agent-decision-client-release-smoke.sh
  scripts/showcase-agent-decision-client-release-smoke.sh --json
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
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/testloop-agent-decision-release-smoke.XXXXXX")"

command -v node >/dev/null 2>&1 || fail "missing required command: node"
command -v npm >/dev/null 2>&1 || fail "missing required command: npm"

release_ref="${TESTLOOP_AGENT_DECISION_RELEASE_VERSION:-}"
if [[ -z "$release_ref" ]]; then
  app_version="$(sed -n 's/^const appVersion = "\([^"]*\)"/\1/p' "$repo_root/main.go" | head -n 1)"
  [[ -n "$app_version" ]] || fail "could not read appVersion from main.go"
  release_ref="v${app_version}"
fi
[[ "$release_ref" == v* ]] || fail "release ref must start with v: $release_ref"

installer_url="${TESTLOOP_AGENT_DECISION_RELEASE_INSTALLER_URL:-https://raw.githubusercontent.com/sleticalboy/testloop-mcp/${release_ref}/scripts/install-agent-decision-client-ci-template.sh}"
helper_dir="${TESTLOOP_AGENT_DECISION_RELEASE_HELPER_DIR:-$repo_root}"
install_client_dir="${TESTLOOP_AGENT_DECISION_RELEASE_CLIENT_DIR:-${tmp_dir}/external-client-install}"
consumer_client_dir="${TESTLOOP_AGENT_DECISION_RELEASE_CONSUMER_DIR:-${tmp_dir}/external-client-consumer}"
install_summary_json="${tmp_dir}/install-summary.json"
client_summary_json="${tmp_dir}/client-summary.json"
client_response_json="${tmp_dir}/client-response.json"
consumer_summary_json="${tmp_dir}/consumer-summary.json"

TESTLOOP_AGENT_DECISION_CI_INSTALLER_URL="$installer_url" \
TESTLOOP_AGENT_DECISION_CI_VERSION="$release_ref" \
TESTLOOP_AGENT_DECISION_CI_HELPER_DIR="$helper_dir" \
TESTLOOP_AGENT_DECISION_CI_CLIENT_DIR="$install_client_dir" \
  bash "${repo_root}/scripts/showcase-agent-decision-client-ci-template-install.sh" --json > "$install_summary_json"

TESTLOOP_AGENT_DECISION_CLIENT_DIR="${tmp_dir}/basic-client" \
  bash "${repo_root}/scripts/showcase-agent-decision-client-ci.sh" --json > "$client_summary_json"

node "${repo_root}/scripts/render-agent-decision-client-ci-response.mjs" \
  --json "$client_summary_json" > "$client_response_json"

TESTLOOP_AGENT_DECISION_CI_INSTALLER_URL="$installer_url" \
TESTLOOP_AGENT_DECISION_CI_VERSION="$release_ref" \
TESTLOOP_AGENT_DECISION_CI_HELPER_DIR="$helper_dir" \
TESTLOOP_AGENT_DECISION_CI_CLIENT_DIR="$consumer_client_dir" \
  bash "${repo_root}/scripts/showcase-agent-decision-client-consumer-smoke.sh" --json > "$consumer_summary_json"

node - \
  "$output_format" \
  "$release_ref" \
  "$installer_url" \
  "$install_summary_json" \
  "$client_summary_json" \
  "$client_response_json" \
  "$consumer_summary_json" <<'JS'
const fs = require('node:fs');
const [
  outputFormat,
  releaseRef,
  installerURL,
  installSummaryPath,
  clientSummaryPath,
  clientResponsePath,
  consumerSummaryPath,
] = process.argv.slice(2);

function readJSON(filePath) {
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

const installSummary = readJSON(installSummaryPath);
const clientSummary = readJSON(clientSummaryPath);
const clientResponse = readJSON(clientResponsePath);
const consumerSummary = readJSON(consumerSummaryPath);
const consumerResponsePath = consumerSummary.agent_response_json || '';
const consumerResponse = consumerResponsePath ? readJSON(consumerResponsePath) : {};
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

function requirePassed(payload, label) {
  if (!payload || payload.status !== 'passed') {
    failures.push(`${label}: expected status=passed`);
  }
}

function requireDecisionSequence(payload, label) {
  if (JSON.stringify(payload.decisions || []) !== JSON.stringify(expectedDecisions)) {
    failures.push(`${label}: decisions drifted`);
  }
}

requirePassed(installSummary, 'install summary');
requirePassed(clientSummary, 'client summary');
requirePassed(clientResponse, 'client response');
requirePassed(consumerSummary, 'consumer summary');
requirePassed(consumerResponse, 'consumer response');
requireDecisionSequence(installSummary, 'install summary');
requireDecisionSequence(clientSummary, 'client summary');
requireDecisionSequence(consumerSummary, 'consumer summary');

if (installSummary.helper_ref !== releaseRef) {
  failures.push(`install summary helper_ref=${installSummary.helper_ref}, want ${releaseRef}`);
}
if (consumerSummary.helper_ref !== releaseRef) {
  failures.push(`consumer summary helper_ref=${consumerSummary.helper_ref}, want ${releaseRef}`);
}
if (installSummary.installer_url !== installerURL) {
  failures.push('install summary installer_url drifted');
}
if (clientResponse.agent_next_step !== 'ready') {
  failures.push(`client response agent_next_step=${clientResponse.agent_next_step}, want ready`);
}
if (consumerResponse.agent_next_step !== 'ready') {
  failures.push(`consumer response agent_next_step=${consumerResponse.agent_next_step}, want ready`);
}
if (installSummary.fixture_count !== 8 || clientSummary.fixture_count !== 8 || consumerSummary.fixture_count !== 8) {
  failures.push('fixture_count must be 8 across release smoke summaries');
}

const payload = {
  schema_version: 1,
  status: failures.length === 0 ? 'passed' : 'failed',
  release_ref: releaseRef,
  installer_url: installerURL,
  install_summary_json: installSummaryPath,
  client_summary_json: clientSummaryPath,
  client_response_json: clientResponsePath,
  consumer_summary_json: consumerSummaryPath,
  consumer_agent_response_json: consumerResponsePath,
  helper_refs: {
    install: installSummary.helper_ref || '',
    consumer: consumerSummary.helper_ref || '',
  },
  fixture_count: 8,
  decisions: expectedDecisions,
  agent_next_steps: {
    client: clientResponse.agent_next_step || '',
    consumer: consumerResponse.agent_next_step || '',
  },
  failures,
};

if (outputFormat === 'json') {
  console.log(JSON.stringify(payload, null, 2));
} else {
  console.log(`agent_decision_release_smoke_status=${payload.status}`);
  console.log(`agent_decision_release_smoke_ref=${payload.release_ref}`);
  console.log(`agent_decision_release_smoke_installer_url=${payload.installer_url}`);
  console.log(`agent_decision_release_smoke_fixture_count=${payload.fixture_count}`);
  console.log(`agent_decision_release_smoke_client_next_step=${payload.agent_next_steps.client}`);
  console.log(`agent_decision_release_smoke_consumer_next_step=${payload.agent_next_steps.consumer}`);
}

if (payload.status !== 'passed') {
  process.exitCode = 1;
}
JS
