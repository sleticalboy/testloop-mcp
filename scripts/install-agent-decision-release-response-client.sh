#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/install-agent-decision-release-response-client.sh [options] [client-dir]

Install the testloop Agent decision release response client package and GitHub
Actions workflow into an external repository.

Options:
  --summary-json PATH    Release smoke summary JSON to copy into the package.
                         Default: docs/fixtures/agent-decision-client-release-smoke-summary/passed.json
  --package-dir PATH     Package path under client-dir.
                         Default: testloop-release-response-client
  --workflow-path PATH   Workflow path under client-dir.
                         Default: .github/workflows/testloop-release-response-contract.yml
  --force               Overwrite an existing workflow file or package directory.
  --dry-run             Print target paths without writing.
  --json                Print a JSON summary.
  -h, --help            Show this help.

Examples:
  scripts/install-agent-decision-release-response-client.sh /path/to/client
  scripts/install-agent-decision-release-response-client.sh --summary-json /tmp/release-smoke.json /path/to/client
USAGE
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

repo_root="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
client_dir="."
summary_json=""
package_dir="testloop-release-response-client"
workflow_path=".github/workflows/testloop-release-response-contract.yml"
force=0
dry_run=0
output_format="text"

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    --summary-json)
      [[ "$#" -ge 2 ]] || fail "--summary-json requires a value"
      summary_json="$2"
      shift 2
      ;;
    --package-dir)
      [[ "$#" -ge 2 ]] || fail "--package-dir requires a value"
      package_dir="$2"
      shift 2
      ;;
    --workflow-path)
      [[ "$#" -ge 2 ]] || fail "--workflow-path requires a value"
      workflow_path="$2"
      shift 2
      ;;
    --force)
      force=1
      shift
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    --json)
      output_format="json"
      shift
      ;;
    --*)
      usage >&2
      exit 2
      ;;
    *)
      if [[ "$client_dir" != "." ]]; then
        usage >&2
        exit 2
      fi
      client_dir="$1"
      shift
      ;;
  esac
done

if [[ -z "$summary_json" ]]; then
  summary_json="${repo_root}/docs/fixtures/agent-decision-client-release-smoke-summary/passed.json"
fi

[[ "$package_dir" != /* ]] || fail "--package-dir must be relative to client-dir"
[[ "$package_dir" != *..* ]] || fail "--package-dir must not contain .."
[[ "$workflow_path" != /* ]] || fail "--workflow-path must be relative to client-dir"
[[ "$workflow_path" != *..* ]] || fail "--workflow-path must not contain .."
[[ -d "$client_dir" ]] || fail "client dir must be an existing directory: $client_dir"
[[ ! -d "$summary_json" ]] || fail "summary JSON path must not be a directory: $summary_json"
[[ -f "$summary_json" ]] || fail "release smoke summary JSON does not exist: $summary_json"

client_dir_real="$(cd "$client_dir" && pwd)"
package_path="${client_dir_real}/${package_dir}"
workflow_target="${client_dir_real}/${workflow_path}"
agent_response_json="${package_path}/testloop-release-response.json"

[[ ! -d "$workflow_target" ]] || fail "workflow path must not be a directory: $workflow_target"
if [[ -e "$workflow_target" && "$force" != "1" ]]; then
  fail "workflow already exists: $workflow_target; pass --force to overwrite"
fi
if [[ -e "$package_path" && "$force" != "1" ]]; then
  fail "package path already exists: $package_path; pass --force to overwrite"
fi

package_rel="$(
  node -e "const path=require('node:path'); process.stdout.write(path.relative(process.argv[1], process.argv[2]) || '.')" \
    "$client_dir_real" "$package_path"
)"

if [[ "$dry_run" = "1" ]]; then
  if [[ "$output_format" = "json" ]]; then
    node - "$client_dir_real" "$workflow_target" "$package_path" "$summary_json" <<'JS'
const [clientDir, workflowPath, packageDir, summaryJson] = process.argv.slice(2);
console.log(JSON.stringify({
  schema_version: 1,
  status: 'dry-run',
  client_dir: clientDir,
  workflow_path: workflowPath,
  package_dir: packageDir,
  release_summary_json: summaryJson,
  agent_response_json: '',
  npm_exit_code: null,
  failures: [],
}, null, 2));
JS
  else
    printf 'agent_decision_release_response_client_install_status=dry-run\n'
    printf 'agent_decision_release_response_client_install_client_dir=%s\n' "$client_dir_real"
    printf 'agent_decision_release_response_client_install_workflow_path=%s\n' "$workflow_target"
    printf 'agent_decision_release_response_client_install_package_dir=%s\n' "$package_path"
    printf 'agent_decision_release_response_client_install_summary_json=%s\n' "$summary_json"
  fi
  exit 0
fi

command -v node >/dev/null 2>&1 || fail "missing required command: node"
command -v npm >/dev/null 2>&1 || fail "missing required command: npm"

if [[ "$force" = "1" ]]; then
  rm -rf "$package_path"
fi

mkdir -p "$(dirname "$workflow_target")"
node "${repo_root}/scripts/export-agent-decision-release-response-client.mjs" \
  "$package_path" "$summary_json" >/dev/null

cat > "$workflow_target" <<YAML
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
  cd "$package_path"
  npm test --silent >/dev/null 2>/dev/null
)
npm_exit_code=$?
set -e

node - \
  "$output_format" \
  "$client_dir_real" \
  "$workflow_target" \
  "$package_path" \
  "${package_path}/testloop-release-smoke-summary.json" \
  "$agent_response_json" \
  "$npm_exit_code" <<'JS'
const fs = require('node:fs');
const [
  outputFormat,
  clientDir,
  workflowPath,
  packageDir,
  releaseSummaryJson,
  agentResponseJson,
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

const response = readJSONIfExists(agentResponseJson);
const evidence = response.evidence || {};

if (!fs.existsSync(workflowPath)) {
  failures.push(`workflow path does not exist: ${workflowPath}`);
}
if (!fs.existsSync(releaseSummaryJson)) {
  failures.push(`release summary json does not exist: ${releaseSummaryJson}`);
}
if (!fs.existsSync(agentResponseJson)) {
  failures.push(`agent response json does not exist: ${agentResponseJson}`);
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
  status: failures.length === 0 ? 'written' : 'failed',
  client_dir: clientDir,
  workflow_path: workflowPath,
  package_dir: packageDir,
  release_summary_json: releaseSummaryJson,
  agent_response_json: agentResponseJson,
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
  console.log(`agent_decision_release_response_client_install_status=${payload.status}`);
  console.log(`agent_decision_release_response_client_install_client_dir=${payload.client_dir}`);
  console.log(`agent_decision_release_response_client_install_workflow_path=${payload.workflow_path}`);
  console.log(`agent_decision_release_response_client_install_package_dir=${payload.package_dir}`);
  console.log(`agent_decision_release_response_client_install_release_ref=${payload.release_ref}`);
  console.log(`agent_decision_release_response_client_install_fixture_count=${payload.fixture_count}`);
  console.log(`agent_decision_release_response_client_install_decisions=${payload.decisions.join(',')}`);
  console.log(`agent_decision_release_response_client_install_agent_next_step=${payload.agent_next_step}`);
  console.log(`agent_decision_release_response_client_install_npm_exit_code=${payload.npm_exit_code}`);
}

if (payload.status === 'failed') {
  process.exitCode = 1;
}
JS
