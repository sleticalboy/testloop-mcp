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

out="${tmp_dir}/validator.out"
node scripts/validate-agent-decision-client-consumer-response.mjs > "$out"
assert_contains "$out" "agent_decision_client_consumer_response_status=passed agent_next_step=ready"
assert_contains "$out" "agent_decision_client_consumer_response_fixture_count=8"
assert_contains "$out" "agent_decision_client_consumer_response_decisions=accept,accept,accept,manual-review,manual-review,manual-review,apply-repair,needs-better-input"

json_out="${tmp_dir}/validator.json"
node scripts/validate-agent-decision-client-consumer-response.mjs --json > "$json_out"
python3 - "$json_out" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "passed"
assert payload["agent_next_step"] == "ready"
assert payload["fixture_count"] == 8
assert payload["failures"] == []
PY

runtime_summary="${tmp_dir}/runtime-summary.json"
runtime_response="${tmp_dir}/runtime-response.json"
scripts/showcase-agent-decision-client-consumer-smoke.sh --json > "$runtime_summary"
node scripts/render-agent-decision-client-consumer-response.mjs --json "$runtime_summary" > "$runtime_response"
node scripts/validate-agent-decision-client-consumer-response.mjs "$runtime_response" > "${tmp_dir}/runtime-validator.out"
assert_contains "${tmp_dir}/runtime-validator.out" "agent_decision_client_consumer_response_status=passed agent_next_step=ready"

bad_response="docs/fixtures/agent-decision-client-consumer-response/client-summary-validator-failed.json"
if node scripts/validate-agent-decision-client-consumer-response.mjs "$bad_response" > "${tmp_dir}/bad.out" 2>&1; then
  echo "expected validator to fail for invalid consumer response" >&2
  exit 1
fi
assert_contains "${tmp_dir}/bad.out" "status must be passed"
assert_contains "${tmp_dir}/bad.out" "agent_next_step must be ready"
assert_contains "${tmp_dir}/bad.out" "failures must be an empty array"
assert_contains "${tmp_dir}/bad.out" "evidence.client_summary_validator_exit_code must be 0"
assert_contains "${tmp_dir}/bad.out" "evidence.client_response_validator_exit_code must be 0"

if node scripts/validate-agent-decision-client-consumer-response.mjs --json "$bad_response" > "${tmp_dir}/bad.json"; then
  echo "expected JSON validator to fail for invalid consumer response" >&2
  exit 1
fi
python3 - "${tmp_dir}/bad.json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "failed"
assert payload["agent_next_step"] == "inspect-consumer-smoke-validator"
assert payload["fixture_count"] == 8
assert payload["failures"]
assert any("status must be passed" in item for item in payload["failures"])
PY

echo "agent decision client consumer response validator test passed"
