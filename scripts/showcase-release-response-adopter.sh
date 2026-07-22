#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-release-response-adopter.sh [--json]

Create a realistic external-client repository sample for release response adoption:
  1. Create or reuse an external client repository directory.
  2. Install the release response client package and workflow.
  3. Copy the adopter README and Agent next-step consumer helper.
  4. Validate install summary, run npm test, and read testloop-release-response.json.

Environment:
  TESTLOOP_RELEASE_RESPONSE_ADOPTER_REPO_DIR       External client repo directory.
                                                   Default: a fresh temp directory.
  TESTLOOP_RELEASE_RESPONSE_ADOPTER_ARTIFACT_DIR   Artifact directory.
                                                   Default: <repo>/testloop-release-response-adopter-artifacts.
  TESTLOOP_RELEASE_RESPONSE_ADOPTER_SUMMARY_JSON   Existing release smoke summary JSON.
                                                   Default: checked-in passed fixture.

Examples:
  scripts/showcase-release-response-adopter.sh
  scripts/showcase-release-response-adopter.sh --json
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
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/testloop-release-response-adopter.XXXXXX")"
client_repo_dir="${TESTLOOP_RELEASE_RESPONSE_ADOPTER_REPO_DIR:-${tmp_dir}/external-client-repo}"
summary_json="${TESTLOOP_RELEASE_RESPONSE_ADOPTER_SUMMARY_JSON:-${repo_root}/docs/fixtures/agent-decision-client-release-smoke-summary/passed.json}"
install_summary_json="${tmp_dir}/install-summary.json"
consumer_json="${tmp_dir}/consumer-response.json"
artifact_dir_input="${TESTLOOP_RELEASE_RESPONSE_ADOPTER_ARTIFACT_DIR:-${client_repo_dir}/testloop-release-response-adopter-artifacts}"

command -v node >/dev/null 2>&1 || fail "missing required command: node"
command -v npm >/dev/null 2>&1 || fail "missing required command: npm"

[[ ! -e "$client_repo_dir" || -d "$client_repo_dir" ]] || fail "repo dir path must be a directory: $client_repo_dir"
[[ ! -d "$summary_json" ]] || fail "summary JSON path must not be a directory: $summary_json"
[[ -f "$summary_json" ]] || fail "release smoke summary JSON does not exist: $summary_json"

mkdir -p "$client_repo_dir"
client_repo_real="$(cd "$client_repo_dir" && pwd)"
mkdir -p "$artifact_dir_input"
artifact_dir_real="$(cd "$artifact_dir_input" && pwd)"
adopter_summary_json="${artifact_dir_real}/testloop-release-response-adopter-summary.json"
summary_consumer_json="${artifact_dir_real}/testloop-release-response-summary-consumer.json"

cp "${repo_root}/examples/release-response-adopter/README.md" \
  "${client_repo_real}/README.md"
mkdir -p "${client_repo_real}/scripts"
cp "${repo_root}/examples/release-response-adopter/scripts/read-testloop-release-response.mjs" \
  "${client_repo_real}/scripts/read-testloop-release-response.mjs"
cp "${repo_root}/examples/release-response-adopter/scripts/read-testloop-release-response-summary.mjs" \
  "${client_repo_real}/scripts/read-testloop-release-response-summary.mjs"
chmod +x "${client_repo_real}/scripts/read-testloop-release-response.mjs"
chmod +x "${client_repo_real}/scripts/read-testloop-release-response-summary.mjs"

"${repo_root}/scripts/install-agent-decision-release-response-client.sh" \
  --summary-json "$summary_json" \
  --json \
  "$client_repo_real" > "$install_summary_json"

node "${repo_root}/scripts/validate-agent-decision-release-response-client-install-summary.mjs" \
  "$install_summary_json" >/dev/null

(
  cd "${client_repo_real}/testloop-release-response-client"
  npm test --silent >/dev/null
)

node "${client_repo_real}/scripts/read-testloop-release-response.mjs" \
  --json \
  "${client_repo_real}/testloop-release-response-client/testloop-release-response.json" \
  > "$consumer_json"

node - \
  "$client_repo_real" \
  "$artifact_dir_real" \
  "$install_summary_json" \
  "$consumer_json" \
  "$adopter_summary_json" \
  "$summary_consumer_json" <<'JS'
const fs = require('node:fs');
const path = require('node:path');
const [repoDir, artifactDir, installSummaryPath, consumerPath, adopterSummaryPath, summaryConsumerPath] = process.argv.slice(2);
const installSummary = JSON.parse(fs.readFileSync(installSummaryPath, 'utf8'));
const consumer = JSON.parse(fs.readFileSync(consumerPath, 'utf8'));
const failures = [];

function exists(filePath, label) {
  if (!fs.existsSync(filePath)) {
    failures.push(`${label} does not exist: ${filePath}`);
  }
}

exists(path.join(repoDir, 'README.md'), 'README');
exists(path.join(repoDir, 'scripts/read-testloop-release-response.mjs'), 'consumer helper');
exists(path.join(repoDir, 'scripts/read-testloop-release-response-summary.mjs'), 'summary consumer helper');
exists(installSummary.workflow_path, 'workflow');
exists(installSummary.package_dir, 'package dir');
exists(installSummary.agent_response_json, 'agent response json');

if (installSummary.status !== 'written') {
  failures.push(`install summary status=${installSummary.status || 'missing'}, want written`);
}
if (installSummary.agent_next_step !== 'ready') {
  failures.push(`install summary agent_next_step=${installSummary.agent_next_step || 'missing'}, want ready`);
}
if (consumer.agent_next_step !== 'ready') {
  failures.push(`consumer agent_next_step=${consumer.agent_next_step || 'missing'}, want ready`);
}
if (consumer.should_accept !== true) {
  failures.push('consumer should_accept must be true');
}

const artifactClientDir = path.join(artifactDir, 'testloop-release-response-client');
fs.mkdirSync(artifactClientDir, {recursive: true});
const artifactInstallSummary = path.join(artifactDir, 'testloop-release-response-install-summary.json');
const artifactConsumer = path.join(artifactDir, 'testloop-release-response-consumer.json');
const artifactReleaseSummary = path.join(artifactClientDir, 'testloop-release-smoke-summary.json');
const artifactAgentResponse = path.join(artifactClientDir, 'testloop-release-response.json');

fs.copyFileSync(installSummaryPath, artifactInstallSummary);
fs.copyFileSync(consumerPath, artifactConsumer);
fs.copyFileSync(installSummary.release_summary_json, artifactReleaseSummary);
fs.copyFileSync(installSummary.agent_response_json, artifactAgentResponse);

const payload = {
  schema_version: 1,
  status: failures.length === 0 ? 'passed' : 'failed',
  repo_dir: repoDir,
  artifact_dir: artifactDir,
  readme_path: path.join(repoDir, 'README.md'),
  workflow_path: installSummary.workflow_path || '',
  package_dir: installSummary.package_dir || '',
  install_summary_json: artifactInstallSummary,
  agent_response_json: artifactAgentResponse,
  consumer_json: artifactConsumer,
  summary_consumer_json: summaryConsumerPath,
  release_ref: installSummary.release_ref || consumer.evidence?.release_ref || '',
  fixture_count: installSummary.fixture_count || consumer.evidence?.fixture_count || 0,
  agent_next_step: consumer.agent_next_step || '',
  should_accept: consumer.should_accept === true,
  npm_exit_code: installSummary.npm_exit_code,
  failures,
};

fs.writeFileSync(adopterSummaryPath, `${JSON.stringify(payload, null, 2)}\n`, 'utf8');
JS

node "${client_repo_real}/scripts/read-testloop-release-response-summary.mjs" \
  --json \
  "$adopter_summary_json" > "$summary_consumer_json"

node - "$output_format" "$adopter_summary_json" <<'JS'
const fs = require('node:fs');
const [outputFormat, summaryPath] = process.argv.slice(2);
const payload = JSON.parse(fs.readFileSync(summaryPath, 'utf8'));

if (outputFormat === 'json') {
  console.log(JSON.stringify(payload, null, 2));
} else {
  console.log(`release_response_adopter_status=${payload.status}`);
  console.log(`release_response_adopter_repo_dir=${payload.repo_dir}`);
  console.log(`release_response_adopter_artifact_dir=${payload.artifact_dir}`);
  console.log(`release_response_adopter_workflow_path=${payload.workflow_path}`);
  console.log(`release_response_adopter_package_dir=${payload.package_dir}`);
  console.log(`release_response_adopter_release_ref=${payload.release_ref}`);
  console.log(`release_response_adopter_fixture_count=${payload.fixture_count}`);
  console.log(`release_response_adopter_agent_next_step=${payload.agent_next_step}`);
  console.log(`release_response_adopter_should_accept=${payload.should_accept}`);
}

if (payload.status !== 'passed') {
  process.exitCode = 1;
}
JS
