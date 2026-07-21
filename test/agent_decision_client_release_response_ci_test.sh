#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

cd "$repo_root"

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

run_expect_code() {
  expected="$1"
  out="$2"
  shift 2

  set +e
  "$@" > "$out" 2>&1
  code=$?
  set -e

  if [ "$code" -ne "$expected" ]; then
    echo "expected exit code $expected, got $code: $*" >&2
    echo "--- $out ---" >&2
    cat "$out" >&2
    exit 1
  fi
}

script="scripts/showcase-agent-decision-client-release-response-ci.sh"
sample="docs/fixtures/agent-decision-client-release-smoke-summary/passed.json"

help_out="${tmp_dir}/help.out"
run_expect_code 0 "$help_out" "$script" --help
assert_contains "$help_out" "Usage: scripts/showcase-agent-decision-client-release-response-ci.sh"

repo_dir="${tmp_dir}/external-client"
out="${tmp_dir}/release-response-ci.out"
TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_REPO_DIR="$repo_dir" \
TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_SUMMARY_JSON="$sample" \
  "$script" > "$out"
assert_contains "$out" "agent_decision_client_release_response_ci_status=passed"
assert_contains "$out" "agent_decision_client_release_response_ci_release_ref=v0.5.20"
assert_contains "$out" "agent_decision_client_release_response_ci_fixture_count=8"
assert_contains "$out" "agent_decision_client_release_response_ci_agent_next_step=ready"

workflow="${repo_dir}/.github/workflows/testloop-release-response-contract.yml"
package_dir="${repo_dir}/testloop-release-response-client"
test -f "$workflow"
test -f "${package_dir}/package.json"
test -f "${package_dir}/testloop-release-response.json"
assert_contains "$workflow" "npm test --silent"
assert_contains "$workflow" "actions/upload-artifact@v4"
assert_contains "$workflow" "testloop-release-response-contract"

json_out="${tmp_dir}/release-response-ci.json"
json_repo="${tmp_dir}/external-client-json"
TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_REPO_DIR="$json_repo" \
TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_SUMMARY_JSON="$sample" \
  "$script" --json > "$json_out"
python3 - "$json_out" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
expected_decisions = [
    "accept",
    "accept",
    "accept",
    "manual-review",
    "manual-review",
    "manual-review",
    "apply-repair",
    "needs-better-input",
]

assert payload["schema_version"] == 1
assert payload["status"] == "passed"
assert payload["release_ref"] == "v0.5.20"
assert payload["fixture_count"] == 8
assert payload["decisions"] == expected_decisions
assert payload["agent_next_step"] == "ready"
assert payload["npm_exit_code"] == 0
assert payload["failures"] == []

for key in [
    "repo_dir",
    "workflow_path",
    "package_dir",
    "release_summary_json",
    "agent_response_json",
]:
    assert Path(payload[key]).exists(), f"{key} does not exist: {payload[key]}"

workflow = Path(payload["workflow_path"]).read_text(encoding="utf-8")
assert "npm test --silent" in workflow
assert "testloop-release-response-contract" in workflow

response = json.loads(Path(payload["agent_response_json"]).read_text(encoding="utf-8"))
assert response["status"] == "passed"
assert response["agent_next_step"] == "ready"
PY

bad_summary="${tmp_dir}/bad-summary.json"
python3 - "$sample" "$bad_summary" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
payload["agent_next_steps"]["client"] = "inspect-agent-decision-client-summary"
Path(sys.argv[2]).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY
bad_repo="${tmp_dir}/bad-external-client"
run_expect_code 1 "${tmp_dir}/bad.out" env \
  TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_REPO_DIR="$bad_repo" \
  TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_SUMMARY_JSON="$bad_summary" \
  "$script" --json
python3 - "${tmp_dir}/bad.out" "$bad_repo/testloop-release-response-client/testloop-release-response.json" <<'PY'
from pathlib import Path
import json
import sys

summary = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
response = json.loads(Path(sys.argv[2]).read_text(encoding="utf-8"))
assert summary["status"] == "failed"
assert summary["agent_next_step"] == "inspect-release-client-response"
assert summary["npm_exit_code"] != 0
assert response["status"] == "failed"
assert response["agent_next_step"] == "inspect-release-client-response"
PY

echo "agent decision client release response CI test passed"
