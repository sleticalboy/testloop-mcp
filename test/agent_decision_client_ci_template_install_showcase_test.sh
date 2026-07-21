#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

assert_contains() {
  file="$1"
  needle="$2"
  if ! grep -F -- "$needle" "$file" >/dev/null 2>&1; then
    echo "expected $file to contain: $needle" >&2
    echo "--- $file ---" >&2
    cat "$file" >&2
    exit 1
  fi
}

assert_exit_code() {
  want="$1"
  got="$2"
  context="$3"
  if [ "$got" -ne "$want" ]; then
    echo "expected exit code $want, got $got: $context" >&2
    exit 1
  fi
}

run_expect_code() {
  want="$1"
  out="$2"
  shift 2
  set +e
  "$@" > "$out" 2>&1
  code=$?
  set -e
  assert_exit_code "$want" "$code" "$*"
}

script="${repo_root}/scripts/showcase-agent-decision-client-ci-template-install.sh"

out="${tmp_dir}/help.out"
run_expect_code 0 "$out" bash "$script" --help
assert_contains "$out" "Usage: scripts/showcase-agent-decision-client-ci-template-install.sh"
assert_contains "$out" "TESTLOOP_AGENT_DECISION_CI_INSTALLER_URL"
assert_contains "$out" "TESTLOOP_AGENT_DECISION_CI_HELPER_DIR"

client_dir="${tmp_dir}/external-client"
out="${tmp_dir}/showcase.json"
run_expect_code 0 "$out" env \
  TESTLOOP_AGENT_DECISION_CI_INSTALLER_PATH="${repo_root}/scripts/install-agent-decision-client-ci-template.sh" \
  TESTLOOP_AGENT_DECISION_CI_CLIENT_DIR="$client_dir" \
  TESTLOOP_AGENT_DECISION_CI_HELPER_DIR="$repo_root" \
  bash "$script" --json

workflow="${client_dir}/.github/workflows/testloop-agent-decision-contract.yml"
assert_contains "$workflow" "repository: sleticalboy/testloop-mcp"
assert_contains "$workflow" "ref: v0.5.16"
assert_contains "$workflow" ".testloop-mcp/scripts/showcase-agent-decision-client-ci.sh --json"

python3 - "$out" "$client_dir" "$workflow" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
client_dir = sys.argv[2]
workflow = sys.argv[3]
assert payload["schema_version"] == 1
assert payload["status"] == "passed"
assert payload["client_dir"] == client_dir
assert payload["workflow_path"] == workflow
assert payload["helper_ref"] == "v0.5.16"
assert payload["fixture_count"] == 8
assert payload["decisions"] == [
    "accept",
    "accept",
    "accept",
    "manual-review",
    "manual-review",
    "manual-review",
    "apply-repair",
    "needs-better-input",
]
assert payload["failures"] == []
assert payload["contract_exit_code"] == 0
assert payload["validator_exit_code"] == 0
assert Path(payload["summary_json"]).exists()
PY

url_client_dir="${tmp_dir}/url-client"
url_out="${tmp_dir}/url-showcase.out"
run_expect_code 0 "$url_out" env \
  TESTLOOP_AGENT_DECISION_CI_INSTALLER_URL="file://${repo_root}/scripts/install-agent-decision-client-ci-template.sh" \
  TESTLOOP_AGENT_DECISION_CI_CLIENT_DIR="$url_client_dir" \
  TESTLOOP_AGENT_DECISION_CI_HELPER_DIR="$repo_root" \
  bash "$script"
assert_contains "$url_out" "agent_decision_template_install_status=passed"
assert_contains "$url_out" "agent_decision_template_install_helper_ref=v0.5.16"

echo "Agent decision client CI template install showcase test passed"
