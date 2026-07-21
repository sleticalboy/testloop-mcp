#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-agent-decision-client-consumer-smoke.sh [--json]

Run a real external-client smoke for the Agent decision CI contract:
  1. Create a temporary external client repository.
  2. Install the GitHub Actions workflow with the local installer.
  3. Run the installed helper dry-run summary path.
  4. Validate the install summary and exported fixture package.
  5. Verify the generated result JSON can be consumed by clients.

Environment:
  TESTLOOP_AGENT_DECISION_CI_CLIENT_DIR      External client directory.
                                             Default: a fresh temp directory.
  TESTLOOP_AGENT_DECISION_CI_VERSION         Helper ref passed to installer.
                                             Default: installer default.

Examples:
  scripts/showcase-agent-decision-client-consumer-smoke.sh
  scripts/showcase-agent-decision-client-consumer-smoke.sh --json
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
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/testloop-agent-decision-consumer-smoke.XXXXXX")"
client_dir="${TESTLOOP_AGENT_DECISION_CI_CLIENT_DIR:-${tmp_dir}/external-client}"
install_summary_json="${tmp_dir}/install-summary.json"
install_validation_json="${tmp_dir}/install-summary-validation.json"
fixture_validation_json="${tmp_dir}/fixture-validation.json"

command -v node >/dev/null 2>&1 || fail "missing required command: node"
command -v npm >/dev/null 2>&1 || fail "missing required command: npm"

[[ ! -e "$client_dir" || -d "$client_dir" ]] || fail "client dir path must be a directory: $client_dir"
mkdir -p "$client_dir"

TESTLOOP_AGENT_DECISION_CI_INSTALLER_PATH="${repo_root}/scripts/install-agent-decision-client-ci-template.sh" \
TESTLOOP_AGENT_DECISION_CI_CLIENT_DIR="$client_dir" \
TESTLOOP_AGENT_DECISION_CI_HELPER_DIR="$repo_root" \
  bash "${repo_root}/scripts/showcase-agent-decision-client-ci-template-install.sh" --json > "$install_summary_json"

set +e
node "${repo_root}/scripts/validate-agent-decision-client-ci-install-summary.mjs" \
  --json "$install_summary_json" > "$install_validation_json"
install_summary_validator_exit_code=$?
set -e

client_summary_json="$(
  node -e "const fs=require('node:fs'); const payload=JSON.parse(fs.readFileSync(process.argv[1], 'utf8')); process.stdout.write(payload.summary_json || '');" \
    "$install_summary_json"
)"
[[ -n "$client_summary_json" ]] || fail "install summary is missing summary_json: $install_summary_json"
[[ -f "$client_summary_json" ]] || fail "client summary JSON does not exist: $client_summary_json"

fixture_dir="$(
  node -e "const fs=require('node:fs'); const payload=JSON.parse(fs.readFileSync(process.argv[1], 'utf8')); process.stdout.write(payload.fixture_dir || '');" \
    "$client_summary_json"
)"
[[ -n "$fixture_dir" ]] || fail "client summary is missing fixture_dir: $client_summary_json"
[[ -d "$fixture_dir" ]] || fail "fixture dir does not exist: $fixture_dir"
fixture_manifest="${fixture_dir}/docs/fixtures/agent-decision-fixtures.json"
[[ -f "$fixture_manifest" ]] || fail "fixture manifest does not exist: $fixture_manifest"

set +e
node "${repo_root}/scripts/validate-agent-decision-fixtures.mjs" \
  --json "$fixture_manifest" "$fixture_dir" > "$fixture_validation_json"
fixture_validator_exit_code=$?
set -e

node - \
  "$output_format" \
  "$install_summary_json" \
  "$install_validation_json" \
  "$client_summary_json" \
  "$fixture_validation_json" \
  "$install_summary_validator_exit_code" \
  "$fixture_validator_exit_code" <<'JS'
const fs = require('node:fs');
const [
  outputFormat,
  installSummaryPath,
  installValidationPath,
  clientSummaryPath,
  fixtureValidationPath,
  installValidatorExitCodeRaw,
  fixtureValidatorExitCodeRaw,
] = process.argv.slice(2);

function readJSON(filePath) {
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

const installSummary = readJSON(installSummaryPath);
const installValidation = readJSON(installValidationPath);
const clientSummary = readJSON(clientSummaryPath);
const fixtureValidation = readJSON(fixtureValidationPath);
const resultJSONPath = clientSummary.result_json || '';
const resultPayload = resultJSONPath ? readJSON(resultJSONPath) : {};
const installValidatorExitCode = Number(installValidatorExitCodeRaw);
const fixtureValidatorExitCode = Number(fixtureValidatorExitCodeRaw);
const failures = [];

function requirePassed(payload, label) {
  if (!payload || payload.status !== 'passed') {
    failures.push(`${label}: expected status=passed`);
  }
}

requirePassed(installSummary, 'install summary');
requirePassed(installValidation, 'install summary validator');
requirePassed(clientSummary, 'client summary');
requirePassed(fixtureValidation, 'fixture validator');
requirePassed(resultPayload, 'result json');

if (installValidatorExitCode !== 0) {
  failures.push(`install summary validator exit code=${installValidatorExitCode}`);
}
if (fixtureValidatorExitCode !== 0) {
  failures.push(`fixture validator exit code=${fixtureValidatorExitCode}`);
}
if (clientSummary.validator_exit_code !== 0) {
  failures.push(`client npm validator exit code=${clientSummary.validator_exit_code}`);
}
if (installSummary.fixture_count !== clientSummary.fixture_count || clientSummary.fixture_count !== resultPayload.fixture_count) {
  failures.push('fixture_count mismatch across install summary, client summary and result json');
}
if (JSON.stringify(installSummary.decisions) !== JSON.stringify(clientSummary.decisions) ||
    JSON.stringify(clientSummary.decisions) !== JSON.stringify(resultPayload.decisions)) {
  failures.push('decisions mismatch across install summary, client summary and result json');
}
if (!fs.existsSync(installSummary.workflow_path)) {
  failures.push(`workflow path does not exist: ${installSummary.workflow_path}`);
}
if (!resultJSONPath || !fs.existsSync(resultJSONPath)) {
  failures.push(`result json does not exist: ${resultJSONPath}`);
}

const payload = {
  schema_version: 1,
  status: failures.length === 0 ? 'passed' : 'failed',
  client_dir: installSummary.client_dir,
  workflow_path: installSummary.workflow_path,
  helper_ref: installSummary.helper_ref,
  install_summary_json: installSummaryPath,
  install_summary_validator_json: installValidationPath,
  client_summary_json: clientSummaryPath,
  fixture_dir: clientSummary.fixture_dir,
  fixture_validation_json: fixtureValidationPath,
  result_json: resultJSONPath,
  fixture_count: installSummary.fixture_count,
  decisions: installSummary.decisions,
  failures,
  install_summary_validator_exit_code: installValidatorExitCode,
  fixture_validator_exit_code: fixtureValidatorExitCode,
  npm_validator_exit_code: clientSummary.validator_exit_code,
};

if (outputFormat === 'json') {
  console.log(JSON.stringify(payload, null, 2));
} else {
  console.log(`agent_decision_client_consumer_smoke_status=${payload.status}`);
  console.log(`agent_decision_client_consumer_smoke_client_dir=${payload.client_dir}`);
  console.log(`agent_decision_client_consumer_smoke_workflow_path=${payload.workflow_path}`);
  console.log(`agent_decision_client_consumer_smoke_helper_ref=${payload.helper_ref}`);
  console.log(`agent_decision_client_consumer_smoke_fixture_count=${payload.fixture_count}`);
  console.log(`agent_decision_client_consumer_smoke_decisions=${payload.decisions.join(',')}`);
  console.log(`agent_decision_client_consumer_smoke_result_json=${payload.result_json}`);
}

if (payload.status !== 'passed') {
  process.exitCode = 1;
}
JS
