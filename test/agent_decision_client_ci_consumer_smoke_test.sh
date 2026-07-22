#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

cd "$repo_root"

out="${tmp_dir}/consumer-smoke.out"
scripts/showcase-agent-decision-client-consumer-smoke.sh > "$out"

grep -F "agent_decision_client_consumer_smoke_status=passed" "$out" >/dev/null
grep -F "agent_decision_client_consumer_smoke_helper_ref=v0.5.21" "$out" >/dev/null
grep -F "agent_decision_client_consumer_smoke_fixture_count=8" "$out" >/dev/null
grep -F "agent_decision_client_consumer_smoke_decisions=accept,accept,accept,manual-review,manual-review,manual-review,apply-repair,needs-better-input" "$out" >/dev/null
grep -F "agent_decision_client_consumer_smoke_agent_response_json=" "$out" >/dev/null
grep -F "agent_decision_client_consumer_smoke_agent_next_step=ready" "$out" >/dev/null

json_out="${tmp_dir}/consumer-smoke.json"
scripts/showcase-agent-decision-client-consumer-smoke.sh --json > "$json_out"

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
assert payload["helper_ref"] == "v0.5.21"
assert payload["fixture_count"] == 8
assert payload["decisions"] == expected_decisions
assert payload["failures"] == []
assert payload["install_summary_validator_exit_code"] == 0
assert payload["fixture_validator_exit_code"] == 0
assert payload["npm_validator_exit_code"] == 0

for key in [
    "client_dir",
    "workflow_path",
    "install_summary_json",
    "install_summary_validator_json",
    "client_summary_json",
    "fixture_dir",
    "fixture_validation_json",
    "result_json",
    "agent_response_json",
]:
    value = Path(payload[key])
    assert value.exists(), f"{key} does not exist: {value}"

install_summary = json.loads(Path(payload["install_summary_json"]).read_text(encoding="utf-8"))
client_summary = json.loads(Path(payload["client_summary_json"]).read_text(encoding="utf-8"))
result_payload = json.loads(Path(payload["result_json"]).read_text(encoding="utf-8"))
fixture_validation = json.loads(Path(payload["fixture_validation_json"]).read_text(encoding="utf-8"))
agent_response = json.loads(Path(payload["agent_response_json"]).read_text(encoding="utf-8"))

assert install_summary["status"] == "passed"
assert client_summary["status"] == "passed"
assert result_payload["status"] == "passed"
assert fixture_validation["status"] == "passed"
assert agent_response["status"] == "passed"
assert agent_response["agent_next_step"] == "ready"
assert install_summary["decisions"] == expected_decisions
assert client_summary["decisions"] == expected_decisions
assert result_payload["decisions"] == expected_decisions
assert install_summary["fixture_count"] == client_summary["fixture_count"] == result_payload["fixture_count"] == 8
PY

echo "agent decision client CI consumer smoke test passed"
