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

script="scripts/showcase-agent-decision-client-adopter.sh"
consumer="examples/agent-decision-client-adopter/scripts/read-testloop-agent-decision-response.mjs"
response="docs/fixtures/agent-decision-client-ci-response/passed.json"
bad_response="docs/fixtures/agent-decision-client-ci-response/validator-failed.json"
readme="examples/agent-decision-client-adopter/README.md"

help_out="${tmp_dir}/showcase-help.out"
run_expect_code 0 "$help_out" "$script" --help
assert_contains "$help_out" "Usage: scripts/showcase-agent-decision-client-adopter.sh"
assert_contains "$help_out" "TESTLOOP_AGENT_DECISION_ADOPTER_REPO_DIR"

consumer_help="${tmp_dir}/consumer-help.out"
run_expect_code 0 "$consumer_help" node "$consumer" --help
assert_contains "$consumer_help" "Usage: node scripts/read-testloop-agent-decision-response.mjs"

consumer_out="${tmp_dir}/consumer.out"
run_expect_code 0 "$consumer_out" node "$consumer" "$response"
assert_contains "$consumer_out" "testloop_agent_decision_response_status=passed"
assert_contains "$consumer_out" "testloop_agent_decision_response_next_step=ready"
assert_contains "$consumer_out" "testloop_agent_decision_response_fixture_count=8"
assert_contains "$consumer_out" "testloop_agent_decision_response_should_accept=true"

consumer_json="${tmp_dir}/consumer.json"
run_expect_code 0 "$consumer_json" node "$consumer" --json "$response"
python3 - "$consumer_json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["schema_version"] == 1
assert payload["status"] == "passed"
assert payload["agent_next_step"] == "ready"
assert payload["should_accept"] is True
assert payload["evidence"]["fixture_count"] == 8
assert payload["failures"] == []
PY

bad_consumer_out="${tmp_dir}/bad-consumer.out"
run_expect_code 1 "$bad_consumer_out" node "$consumer" "$bad_response"
assert_contains "$bad_consumer_out" "testloop_agent_decision_response_status=failed"
assert_contains "$bad_consumer_out" "testloop_agent_decision_response_next_step=inspect-client-validator"
assert_contains "$bad_consumer_out" "testloop_agent_decision_response_should_accept=false"
assert_contains "$bad_consumer_out" "validator_exit_code=1"

missing_out="${tmp_dir}/missing.out"
run_expect_code 1 "$missing_out" node "$consumer" "${tmp_dir}/missing-response.json"
assert_contains "$missing_out" "testloop_agent_decision_response_next_step=inspect-agent-decision-client-summary"

repo_dir="${tmp_dir}/adopter-repo"
showcase_json="${tmp_dir}/showcase.json"
TESTLOOP_AGENT_DECISION_ADOPTER_REPO_DIR="$repo_dir" \
  "$script" --json > "$showcase_json"

python3 - "$showcase_json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["schema_version"] == 1
assert payload["status"] == "passed"
assert payload["fixture_count"] == 8
assert payload["agent_next_step"] == "ready"
assert payload["should_accept"] is True
assert payload["npm_exit_code"] == 0
assert payload["failures"] == []
for key in [
    "repo_dir",
    "package_dir",
    "result_json",
    "response_json",
    "response_validation_json",
    "consumer_json",
    "readme_path",
]:
    assert Path(payload[key]).exists(), f"{key} does not exist: {payload[key]}"

consumer = json.loads(Path(payload["consumer_json"]).read_text(encoding="utf-8"))
assert consumer["agent_next_step"] == "ready"
assert consumer["should_accept"] is True
PY

assert_contains "${repo_dir}/README.md" "node scripts/read-testloop-agent-decision-response.mjs"
assert_contains "${repo_dir}/scripts/read-testloop-agent-decision-response.mjs" "testloop_agent_decision_response_next_step"
assert_contains "$readme" "scripts/showcase-agent-decision-client-adopter.sh --json"
assert_contains "$readme" "node scripts/validate-agent-decision-client-adopter-summary.mjs"
assert_contains "$readme" "agent-decision-client-adopter-summary.schema.json"
assert_contains "$readme" "invalid-response.json"
assert_contains "$readme" "testloop_agent_decision_response_status"
assert_contains "$readme" "testloop_agent_decision_response_next_step"
assert_contains "$readme" "inspect-client-validator"
assert_contains "$readme" "inspect-agent-decision-fixtures"
assert_contains "$readme" "inspect-agent-decision-client-summary"

echo "agent decision client adopter example test passed"
