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

script="scripts/render-agent-decision-client-ci-response.mjs"

help_out="${tmp_dir}/help.out"
run_expect_code 0 "$help_out" node "$script" --help
assert_contains "$help_out" "Usage: node scripts/render-agent-decision-client-ci-response.mjs"

summary_json="${tmp_dir}/summary.json"
TESTLOOP_AGENT_DECISION_CLIENT_DIR="${tmp_dir}/client" \
  bash scripts/showcase-agent-decision-client-ci.sh --json > "$summary_json"

out="${tmp_dir}/response.out"
node "$script" "$summary_json" > "$out"
assert_contains "$out" "agent_decision_client_response_status=passed"
assert_contains "$out" "agent_next_step=ready"
assert_contains "$out" "fixture_count=8"
assert_contains "$out" "validator_exit_code=0"
assert_contains "$out" "result_schema="

json_out="${tmp_dir}/response.json"
node "$script" --json "$summary_json" > "$json_out"
python3 - "$json_out" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["schema_version"] == 1
assert payload["status"] == "passed"
assert payload["agent_next_step"] == "ready"
assert payload["evidence"]["fixture_count"] == 8
assert payload["evidence"]["validator_exit_code"] == 0
assert payload["evidence"]["result_schema"].endswith("agent-decision-fixtures-result.schema.json")
assert payload["failures"] == []
PY

bad_validator="${tmp_dir}/bad-validator.json"
python3 - "$summary_json" "$bad_validator" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
payload["status"] = "failed"
payload["validator_exit_code"] = 1
payload["failures"] = ["npm test failed"]
Path(sys.argv[2]).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY
run_expect_code 1 "${tmp_dir}/bad-validator.out" node "$script" "$bad_validator"
assert_contains "${tmp_dir}/bad-validator.out" "agent_next_step=inspect-client-validator"
assert_contains "${tmp_dir}/bad-validator.out" "validator_exit_code=1"

bad_decisions="${tmp_dir}/bad-decisions.json"
python3 - "$summary_json" "$bad_decisions" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
payload["fixture_count"] = 7
payload["decisions"] = payload["decisions"][:-1]
Path(sys.argv[2]).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY
run_expect_code 1 "${tmp_dir}/bad-decisions.out" node "$script" "$bad_decisions"
assert_contains "${tmp_dir}/bad-decisions.out" "agent_next_step=inspect-agent-decision-fixtures"
assert_contains "${tmp_dir}/bad-decisions.out" "fixture_count=7, want 8"

run_expect_code 1 "${tmp_dir}/missing.out" node "$script" "${tmp_dir}/missing.json"
assert_contains "${tmp_dir}/missing.out" "agent_next_step=inspect-agent-decision-client-summary"

echo "agent decision client CI response test passed"
