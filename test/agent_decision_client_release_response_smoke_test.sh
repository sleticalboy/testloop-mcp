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

script="scripts/showcase-agent-decision-client-release-response-smoke.sh"
sample="docs/fixtures/agent-decision-client-release-smoke-summary/passed.json"

help_out="${tmp_dir}/help.out"
run_expect_code 0 "$help_out" "$script" --help
assert_contains "$help_out" "Usage: scripts/showcase-agent-decision-client-release-response-smoke.sh"

out="${tmp_dir}/response-smoke.out"
TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_SUMMARY_JSON="$sample" \
  "$script" > "$out"
assert_contains "$out" "agent_decision_client_release_response_smoke_status=passed"
assert_contains "$out" "agent_decision_client_release_response_smoke_release_ref=v0.5.19"
assert_contains "$out" "agent_decision_client_release_response_smoke_fixture_count=8"
assert_contains "$out" "agent_decision_client_release_response_smoke_agent_next_step=ready"

json_out="${tmp_dir}/response-smoke.json"
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
assert payload["release_ref"] == "v0.5.19"
assert payload["fixture_count"] == 8
assert payload["decisions"] == expected_decisions
assert payload["agent_next_step"] == "ready"
assert payload["npm_exit_code"] == 0
assert payload["failures"] == []

for key in [
    "client_dir",
    "release_summary_json",
    "agent_response_json",
]:
    assert Path(payload[key]).exists(), f"{key} does not exist: {payload[key]}"

client_dir = Path(payload["client_dir"])
assert (client_dir / "package.json").exists()
assert (client_dir / "scripts/render-agent-decision-client-release-response.mjs").exists()
assert (client_dir / "scripts/assert-release-response.mjs").exists()

response = json.loads(Path(payload["agent_response_json"]).read_text(encoding="utf-8"))
assert response["status"] == "passed"
assert response["agent_next_step"] == "ready"
assert response["evidence"]["release_ref"] == "v0.5.19"
PY

runtime_json="${tmp_dir}/runtime-response-smoke.json"
TESTLOOP_AGENT_DECISION_RELEASE_INSTALLER_URL="file://${repo_root}/scripts/install-agent-decision-client-ci-template.sh" \
  "$script" --json > "$runtime_json"
python3 - "$runtime_json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "passed"
assert payload["release_ref"] == "v0.5.19"
assert payload["fixture_count"] == 8
assert payload["agent_next_step"] == "ready"
assert payload["npm_exit_code"] == 0
assert Path(payload["release_summary_json"]).exists()
assert Path(payload["agent_response_json"]).exists()
PY

bad_summary="${tmp_dir}/bad-summary.json"
python3 - "$sample" "$bad_summary" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
payload["agent_next_steps"]["consumer"] = "inspect-consumer-smoke-summary"
Path(sys.argv[2]).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY
run_expect_code 1 "${tmp_dir}/bad.out" env TESTLOOP_AGENT_DECISION_RELEASE_RESPONSE_SUMMARY_JSON="$bad_summary" "$script"
assert_contains "${tmp_dir}/bad.out" "agent_decision_client_release_response_smoke_status=failed"

echo "agent decision client release response smoke test passed"
