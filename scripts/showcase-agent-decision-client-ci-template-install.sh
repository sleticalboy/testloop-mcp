#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-agent-decision-client-ci-template-install.sh [--json]

Run an external-client dry-run for the Agent decision CI template installer:
  1. Download or use the installer script.
  2. Install the GitHub Actions workflow into a client directory.
  3. Simulate the .testloop-mcp helper checkout.
  4. Run the Agent decision fixture contract command.

Environment:
  TESTLOOP_AGENT_DECISION_CI_INSTALLER_PATH  Local installer script path.
  TESTLOOP_AGENT_DECISION_CI_INSTALLER_URL   Installer URL when local path is not set.
                                             Default: raw GitHub main installer.
  TESTLOOP_AGENT_DECISION_CI_DOWNLOAD_RETRIES
                                             curl retry count. Default: 3.
  TESTLOOP_AGENT_DECISION_CI_DOWNLOAD_MAX_TIME
                                             curl max time in seconds. Default: 120.
  TESTLOOP_AGENT_DECISION_CI_CLIENT_DIR      External client directory.
                                             Default: a fresh temp directory.
  TESTLOOP_AGENT_DECISION_CI_HELPER_DIR      testloop-mcp helper checkout directory.
                                             Default: this repository.
  TESTLOOP_AGENT_DECISION_CI_VERSION         Helper ref passed to installer.
                                             Default: installer default.

Examples:
  scripts/showcase-agent-decision-client-ci-template-install.sh
  scripts/showcase-agent-decision-client-ci-template-install.sh --json
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
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/testloop-agent-decision-template-install.XXXXXX")"
installer_path="${TESTLOOP_AGENT_DECISION_CI_INSTALLER_PATH:-}"
installer_url="${TESTLOOP_AGENT_DECISION_CI_INSTALLER_URL:-https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/install-agent-decision-client-ci-template.sh}"
client_dir="${TESTLOOP_AGENT_DECISION_CI_CLIENT_DIR:-${tmp_dir}/external-client}"
helper_dir="${TESTLOOP_AGENT_DECISION_CI_HELPER_DIR:-$repo_root}"
workflow_path="${client_dir}/.github/workflows/testloop-agent-decision-contract.yml"
summary_json="${tmp_dir}/testloop-agent-decision-client-summary.json"
contract_client_dir="${tmp_dir}/testloop-agent-decision-client"

command -v node >/dev/null 2>&1 || fail "missing required command: node"
command -v npm >/dev/null 2>&1 || fail "missing required command: npm"

if [[ -z "$installer_path" ]]; then
  command -v curl >/dev/null 2>&1 || fail "missing required command: curl"
  installer_path="${tmp_dir}/install-agent-decision-client-ci-template.sh"
  curl_retries="${TESTLOOP_AGENT_DECISION_CI_DOWNLOAD_RETRIES:-3}"
  curl_max_time="${TESTLOOP_AGENT_DECISION_CI_DOWNLOAD_MAX_TIME:-120}"
  curl -fsSL \
    --retry "$curl_retries" \
    --retry-delay 2 \
    --retry-connrefused \
    --max-time "$curl_max_time" \
    "$installer_url" \
    -o "$installer_path"
fi

[[ -f "$installer_path" ]] || fail "installer script does not exist: $installer_path"
[[ -d "$helper_dir" ]] || fail "helper dir must be an existing directory: $helper_dir"
[[ -x "$helper_dir/scripts/showcase-agent-decision-client-ci.sh" ]] || fail "helper dir is missing scripts/showcase-agent-decision-client-ci.sh: $helper_dir"
mkdir -p "$client_dir"

installer_args=(--force)
if [[ -n "${TESTLOOP_AGENT_DECISION_CI_VERSION:-}" ]]; then
  installer_args+=(--version "$TESTLOOP_AGENT_DECISION_CI_VERSION")
fi
installer_output="${tmp_dir}/installer.out"
bash "$installer_path" "${installer_args[@]}" "$client_dir" > "$installer_output"
[[ -f "$workflow_path" ]] || fail "installer did not write workflow: $workflow_path"

if [[ ! -e "${client_dir}/.testloop-mcp" ]]; then
  ln -s "$helper_dir" "${client_dir}/.testloop-mcp"
fi

rm -rf "$contract_client_dir" "$summary_json"
set +e
(
  cd "$client_dir"
  TESTLOOP_AGENT_DECISION_CLIENT_DIR="$contract_client_dir" \
    .testloop-mcp/scripts/showcase-agent-decision-client-ci.sh --json \
    | tee "$summary_json" >/dev/null
)
contract_status=$?
set -e

node - "$output_format" "$installer_path" "$installer_url" "$client_dir" "$workflow_path" "$helper_dir" "$summary_json" "$installer_output" "$contract_status" <<'JS'
const fs = require('node:fs');
const [
  outputFormat,
  installerPath,
  installerUrl,
  clientDir,
  workflowPath,
  helperDir,
  summaryPath,
  installerOutputPath,
  contractStatusRaw,
] = process.argv.slice(2);
const contractExitCode = Number(contractStatusRaw);
const contractSummary = JSON.parse(fs.readFileSync(summaryPath, 'utf8'));
const installerOutput = fs.readFileSync(installerOutputPath, 'utf8');
const refMatch = installerOutput.match(/^agent_decision_client_ci_template_ref=(.+)$/m);
const payload = {
  schema_version: 1,
  status: contractSummary.status === 'passed' && contractExitCode === 0 ? 'passed' : 'failed',
  installer_path: installerPath,
  installer_url: installerUrl,
  client_dir: clientDir,
  workflow_path: workflowPath,
  helper_dir: helperDir,
  helper_ref: refMatch ? refMatch[1] : '',
  summary_json: summaryPath,
  fixture_count: contractSummary.fixture_count,
  decisions: contractSummary.decisions,
  failures: contractSummary.failures || [],
  contract_exit_code: contractExitCode,
  validator_exit_code: contractSummary.validator_exit_code,
};
if (outputFormat === 'json') {
  console.log(JSON.stringify(payload, null, 2));
} else {
  console.log(`agent_decision_template_install_status=${payload.status}`);
  console.log(`agent_decision_template_install_client_dir=${payload.client_dir}`);
  console.log(`agent_decision_template_install_workflow_path=${payload.workflow_path}`);
  console.log(`agent_decision_template_install_helper_ref=${payload.helper_ref}`);
  console.log(`agent_decision_template_install_fixture_count=${payload.fixture_count}`);
  console.log(`agent_decision_template_install_decisions=${payload.decisions.join(',')}`);
}
if (payload.status !== 'passed') {
  process.exitCode = 1;
}
JS
