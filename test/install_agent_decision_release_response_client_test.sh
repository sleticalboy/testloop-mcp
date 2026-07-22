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

assert_not_exists() {
  path="$1"
  if [ -e "$path" ]; then
    echo "expected path not to exist: $path" >&2
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

script="scripts/install-agent-decision-release-response-client.sh"
sample="docs/fixtures/agent-decision-client-release-smoke-summary/passed.json"

help_out="${tmp_dir}/help.out"
run_expect_code 0 "$help_out" "$script" --help
assert_contains "$help_out" "Usage: scripts/install-agent-decision-release-response-client.sh"
assert_contains "$help_out" "--summary-json PATH"
assert_contains "$help_out" "--package-dir PATH"
assert_contains "$help_out" "--workflow-path PATH"
assert_contains "$help_out" "--dry-run"
assert_contains "$help_out" "--json"

client_dir="${tmp_dir}/client"
mkdir -p "$client_dir"

dry_out="${tmp_dir}/dry-run.out"
run_expect_code 0 "$dry_out" "$script" --dry-run "$client_dir"
assert_contains "$dry_out" "agent_decision_release_response_client_install_status=dry-run"
assert_contains "$dry_out" "agent_decision_release_response_client_install_workflow_path=${client_dir}/.github/workflows/testloop-release-response-contract.yml"
assert_contains "$dry_out" "agent_decision_release_response_client_install_package_dir=${client_dir}/testloop-release-response-client"
assert_not_exists "${client_dir}/.github/workflows/testloop-release-response-contract.yml"
assert_not_exists "${client_dir}/testloop-release-response-client"

write_out="${tmp_dir}/write.out"
run_expect_code 0 "$write_out" "$script" --summary-json "$sample" "$client_dir"
workflow="${client_dir}/.github/workflows/testloop-release-response-contract.yml"
package_dir="${client_dir}/testloop-release-response-client"
assert_contains "$write_out" "agent_decision_release_response_client_install_status=written"
assert_contains "$write_out" "agent_decision_release_response_client_install_release_ref=v0.5.21"
assert_contains "$write_out" "agent_decision_release_response_client_install_fixture_count=8"
assert_contains "$write_out" "agent_decision_release_response_client_install_agent_next_step=ready"
assert_contains "$write_out" "agent_decision_release_response_client_install_npm_exit_code=0"
test -f "$workflow"
test -f "${package_dir}/package.json"
test -f "${package_dir}/testloop-release-response.json"
assert_contains "$workflow" "name: testloop release response contract"
assert_contains "$workflow" "cd testloop-release-response-client"
assert_contains "$workflow" "npm test --silent"
assert_contains "$workflow" "actions/upload-artifact@v4"
assert_contains "$workflow" "testloop-release-response-contract"

exists_out="${tmp_dir}/exists.out"
run_expect_code 1 "$exists_out" "$script" "$client_dir"
assert_contains "$exists_out" "workflow already exists"

custom_dir="${tmp_dir}/custom-client"
mkdir -p "$custom_dir"
json_out="${tmp_dir}/install.json"
run_expect_code 0 "$json_out" "$script" \
  --json \
  --package-dir tools/testloop-release-response-client \
  --workflow-path .github/workflows/custom-release-response.yml \
  "$custom_dir"
python3 - "$json_out" "$custom_dir" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
client_dir = Path(sys.argv[2])
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
assert payload["status"] == "written"
assert payload["client_dir"] == str(client_dir)
assert payload["workflow_path"] == str(client_dir / ".github/workflows/custom-release-response.yml")
assert payload["package_dir"] == str(client_dir / "tools/testloop-release-response-client")
assert payload["release_ref"] == "v0.5.21"
assert payload["fixture_count"] == 8
assert payload["decisions"] == expected_decisions
assert payload["agent_next_step"] == "ready"
assert payload["npm_exit_code"] == 0
assert payload["failures"] == []
for key in ["workflow_path", "package_dir", "release_summary_json", "agent_response_json"]:
    assert Path(payload[key]).exists(), f"{key} does not exist: {payload[key]}"

workflow = Path(payload["workflow_path"]).read_text(encoding="utf-8")
assert "cd tools/testloop-release-response-client" in workflow
assert "npm test --silent" in workflow
PY

force_out="${tmp_dir}/force.out"
run_expect_code 0 "$force_out" "$script" --force --summary-json "$sample" "$client_dir"
assert_contains "$force_out" "agent_decision_release_response_client_install_status=written"

bad_path_out="${tmp_dir}/bad-path.out"
run_expect_code 1 "$bad_path_out" "$script" --package-dir ../bad "$custom_dir"
assert_contains "$bad_path_out" "--package-dir must not contain .."

bad_summary="${tmp_dir}/bad-summary.json"
python3 - "$sample" "$bad_summary" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
payload["agent_next_steps"]["client"] = "inspect-agent-decision-client-summary"
Path(sys.argv[2]).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY
bad_client="${tmp_dir}/bad-client"
mkdir -p "$bad_client"
run_expect_code 1 "${tmp_dir}/bad.out" "$script" --json --summary-json "$bad_summary" "$bad_client"
python3 - "${tmp_dir}/bad.out" "$bad_client/testloop-release-response-client/testloop-release-response.json" <<'PY'
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

ruby -e 'require "yaml"; data = YAML.load_file(ARGV.fetch(0)); raise "missing jobs" unless data["jobs"] || data[true]' "$workflow"

echo "install Agent decision release response client test passed"
