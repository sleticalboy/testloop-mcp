#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-agent-decision-client-release-response-ci.sh [--json]

Create a realistic external-client repository smoke for release response consumption:
  1. Create or reuse an external client repository directory.
  2. Export the release response client package into that repository.
  3. Write a GitHub Actions workflow that runs the exported package's npm test.
  4. Run the same npm test locally and print a stable summary.

Environment:
  TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_REPO_DIR       External client repo directory.
                                                         Default: a fresh temp directory.
  TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_PACKAGE_DIR    Exported package directory.
                                                         Default: <repo-dir>/testloop-release-response-client
  TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_SUMMARY_JSON   Existing release smoke summary JSON.
                                                         Default: checked-in passed fixture.
  TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_WORKFLOW_PATH  Workflow path.
                                                         Default: <repo-dir>/.github/workflows/testloop-release-response-contract.yml

Examples:
  scripts/showcase-agent-decision-client-release-response-ci.sh
  scripts/showcase-agent-decision-client-release-response-ci.sh --json
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
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/testloop-release-response-client-ci.XXXXXX")"
client_repo_dir="${TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_REPO_DIR:-${tmp_dir}/external-client-repo}"
package_dir="${TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_PACKAGE_DIR:-${client_repo_dir}/testloop-release-response-client}"
summary_json="${TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_SUMMARY_JSON:-${repo_root}/docs/fixtures/agent-decision-client-release-smoke-summary/passed.json}"
workflow_path="${TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_WORKFLOW_PATH:-${client_repo_dir}/.github/workflows/testloop-release-response-contract.yml}"
agent_response_json="${package_dir}/testloop-release-response.json"

command -v node >/dev/null 2>&1 || fail "missing required command: node"
command -v npm >/dev/null 2>&1 || fail "missing required command: npm"

[[ ! -e "$client_repo_dir" || -d "$client_repo_dir" ]] || fail "repo dir path must be a directory: $client_repo_dir"
[[ ! -e "$package_dir" || -d "$package_dir" ]] || fail "package dir path must be a directory: $package_dir"
[[ ! -d "$summary_json" ]] || fail "summary JSON path must not be a directory: $summary_json"
[[ -f "$summary_json" ]] || fail "release smoke summary JSON does not exist: $summary_json"

mkdir -p "$client_repo_dir" "$(dirname "$workflow_path")"

client_repo_real="$(cd "$client_repo_dir" && pwd)"
mkdir -p "$(dirname "$package_dir")"
package_parent_real="$(cd "$(dirname "$package_dir")" && pwd)"
case "${package_parent_real}/" in
  "${client_repo_real}/"*|"${client_repo_real}/")
    ;;
  *)
    fail "package dir must be inside repo dir: $package_dir"
    ;;
esac

node "${repo_root}/scripts/export-agent-decision-release-response-client.mjs" \
  "$package_dir" "$summary_json" >/dev/null

package_rel="$(
  node -e "const path=require('node:path'); process.stdout.write(path.relative(process.argv[1], process.argv[2]) || '.')" \
    "$client_repo_real" "$(cd "$package_dir" && pwd)"
)"

cat > "$workflow_path" <<YAML
name: testloop release response contract

on:
  workflow_dispatch:
  pull_request:

jobs:
  release-response-contract:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: 22

      - name: Verify release response contract
        run: |
          set -euo pipefail
          cd ${package_rel}
          npm test --silent

      - name: Upload release response artifacts
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: testloop-release-response-contract
          path: |
            ${package_rel}/testloop-release-smoke-summary.json
            ${package_rel}/testloop-release-response.json
            ${package_rel}/package.json
            ${package_rel}/docs/fixtures/agent-decision-client-release-response.schema.json
            ${package_rel}/docs/fixtures/agent-decision-client-release-response/*.json
YAML

set +e
(
  cd "$package_dir"
  npm test --silent >/dev/null 2>/dev/null
)
npm_exit_code=$?
set -e

node - \
  "$output_format" \
  "$client_repo_real" \
  "$(cd "$package_dir" && pwd)" \
  "$workflow_path" \
  "$agent_response_json" \
  "$npm_exit_code" <<'JS'
const fs = require('node:fs');
const [
  outputFormat,
  repoDir,
  packageDir,
  workflowPath,
  agentResponsePath,
  npmExitCodeRaw,
] = process.argv.slice(2);
const npmExitCode = Number(npmExitCodeRaw);
const failures = [];

function readJSONIfExists(filePath) {
  if (!fs.existsSync(filePath)) {
    return {};
  }
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

const response = readJSONIfExists(agentResponsePath);
const evidence = response.evidence || {};

if (!fs.existsSync(workflowPath)) {
  failures.push(`workflow path does not exist: ${workflowPath}`);
}
if (!fs.existsSync(agentResponsePath)) {
  failures.push(`agent response json does not exist: ${agentResponsePath}`);
}
if (response.status !== 'passed') {
  failures.push(`agent response status=${response.status || 'missing'}, want passed`);
}
if (response.agent_next_step !== 'ready') {
  failures.push(`agent_next_step=${response.agent_next_step || 'missing'}, want ready`);
}
if (npmExitCode !== 0) {
  failures.push(`npm test exit code=${npmExitCode}`);
}

const payload = {
  schema_version: 1,
  status: failures.length === 0 ? 'passed' : 'failed',
  repo_dir: repoDir,
  workflow_path: workflowPath,
  package_dir: packageDir,
  release_summary_json: `${packageDir}/testloop-release-smoke-summary.json`,
  agent_response_json: agentResponsePath,
  release_ref: evidence.release_ref || '',
  fixture_count: evidence.fixture_count || 0,
  decisions: evidence.decisions || [],
  agent_next_step: response.agent_next_step || '',
  npm_exit_code: npmExitCode,
  failures,
};

if (outputFormat === 'json') {
  console.log(JSON.stringify(payload, null, 2));
} else {
  console.log(`agent_decision_client_release_response_ci_status=${payload.status}`);
  console.log(`agent_decision_client_release_response_ci_repo_dir=${payload.repo_dir}`);
  console.log(`agent_decision_client_release_response_ci_workflow_path=${payload.workflow_path}`);
  console.log(`agent_decision_client_release_response_ci_package_dir=${payload.package_dir}`);
  console.log(`agent_decision_client_release_response_ci_release_ref=${payload.release_ref}`);
  console.log(`agent_decision_client_release_response_ci_fixture_count=${payload.fixture_count}`);
  console.log(`agent_decision_client_release_response_ci_decisions=${payload.decisions.join(',')}`);
  console.log(`agent_decision_client_release_response_ci_agent_next_step=${payload.agent_next_step}`);
}

if (payload.status !== 'passed') {
  process.exitCode = 1;
}
JS
