#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-agent-decision-client-adopter.sh [--json]

Create a realistic external-client repository sample for base Agent decision response adoption:
  1. Create or reuse an external client repository directory.
  2. Export the Agent decision fixture package.
  3. Copy the adopter README and Agent next-step consumer helper.
  4. Run fixture validation, render client response, validate response, and read it as an adopter.

Environment:
  TESTLOOP_AGENT_DECISION_ADOPTER_REPO_DIR       External client repo directory.
                                                 Default: a fresh temp directory.

Examples:
  scripts/showcase-agent-decision-client-adopter.sh
  scripts/showcase-agent-decision-client-adopter.sh --json
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
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/testloop-agent-decision-client-adopter.XXXXXX")"
client_repo_dir="${TESTLOOP_AGENT_DECISION_ADOPTER_REPO_DIR:-${tmp_dir}/external-client-repo}"

command -v node >/dev/null 2>&1 || fail "missing required command: node"
command -v npm >/dev/null 2>&1 || fail "missing required command: npm"

[[ ! -e "$client_repo_dir" || -d "$client_repo_dir" ]] || fail "repo dir path must be a directory: $client_repo_dir"

mkdir -p "$client_repo_dir"
client_repo_real="$(cd "$client_repo_dir" && pwd)"
package_dir="${client_repo_real}/testloop-agent-decision-fixtures"
result_json="${client_repo_real}/agent-decision-fixtures-result.json"
response_json="${client_repo_real}/testloop-agent-decision-client-response.json"
response_validation_json="${client_repo_real}/testloop-agent-decision-client-response-validation.json"
consumer_json="${client_repo_real}/testloop-agent-decision-response-consumer.json"

cp "${repo_root}/examples/agent-decision-client-adopter/README.md" \
  "${client_repo_real}/README.md"
mkdir -p "${client_repo_real}/scripts"
cp "${repo_root}/examples/agent-decision-client-adopter/scripts/read-testloop-agent-decision-response.mjs" \
  "${client_repo_real}/scripts/read-testloop-agent-decision-response.mjs"
chmod +x "${client_repo_real}/scripts/read-testloop-agent-decision-response.mjs"

node "${repo_root}/scripts/export-agent-decision-fixtures.mjs" "$package_dir" >/dev/null

set +e
(
  cd "$package_dir"
  npm test --silent > "$result_json"
)
npm_status=$?
set -e

(
  cd "$package_dir"
  npm run render:client-response --silent > "$response_json"
  npm run validate:client-response --silent > "$response_validation_json"
)

node "${client_repo_real}/scripts/read-testloop-agent-decision-response.mjs" \
  --json \
  "$response_json" > "$consumer_json"

node - \
  "$output_format" \
  "$client_repo_real" \
  "$package_dir" \
  "$result_json" \
  "$response_json" \
  "$response_validation_json" \
  "$consumer_json" \
  "$npm_status" <<'JS'
const fs = require('node:fs');
const path = require('node:path');
const [
  outputFormat,
  repoDir,
  packageDir,
  resultPath,
  responsePath,
  responseValidationPath,
  consumerPath,
  npmStatusRaw,
] = process.argv.slice(2);
const npmStatus = Number(npmStatusRaw);
const consumer = JSON.parse(fs.readFileSync(consumerPath, 'utf8'));
const failures = [];

function exists(filePath, label) {
  if (!fs.existsSync(filePath)) {
    failures.push(`${label} does not exist: ${filePath}`);
  }
}

exists(path.join(repoDir, 'README.md'), 'README');
exists(path.join(repoDir, 'scripts/read-testloop-agent-decision-response.mjs'), 'consumer helper');
exists(path.join(packageDir, 'package.json'), 'exported package');
exists(path.join(packageDir, 'docs/fixtures/agent-decision-client-ci-response.schema.json'), 'client response schema');
exists(resultPath, 'fixture result JSON');
exists(responsePath, 'client response JSON');
exists(responseValidationPath, 'client response validation output');
exists(consumerPath, 'consumer JSON');

if (npmStatus !== 0) {
  failures.push(`npm_exit_code=${npmStatus}`);
}
if (consumer.agent_next_step !== 'ready') {
  failures.push(`consumer agent_next_step=${consumer.agent_next_step || 'missing'}, want ready`);
}
if (consumer.should_accept !== true) {
  failures.push('consumer should_accept must be true');
}

const payload = {
  schema_version: 1,
  status: failures.length === 0 ? 'passed' : 'failed',
  repo_dir: repoDir,
  package_dir: packageDir,
  result_json: resultPath,
  response_json: responsePath,
  response_validation_json: responseValidationPath,
  consumer_json: consumerPath,
  fixture_count: consumer.evidence?.fixture_count || 0,
  agent_next_step: consumer.agent_next_step || '',
  should_accept: consumer.should_accept === true,
  npm_exit_code: npmStatus,
  failures,
};

if (outputFormat === 'json') {
  console.log(JSON.stringify(payload, null, 2));
} else {
  console.log(`agent_decision_client_adopter_status=${payload.status}`);
  console.log(`agent_decision_client_adopter_repo_dir=${payload.repo_dir}`);
  console.log(`agent_decision_client_adopter_package_dir=${payload.package_dir}`);
  console.log(`agent_decision_client_adopter_fixture_count=${payload.fixture_count}`);
  console.log(`agent_decision_client_adopter_agent_next_step=${payload.agent_next_step}`);
  console.log(`agent_decision_client_adopter_should_accept=${payload.should_accept}`);
}

if (payload.status !== 'passed') {
  process.exitCode = 1;
}
JS
