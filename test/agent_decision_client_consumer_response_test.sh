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

script="scripts/render-agent-decision-client-consumer-response.mjs"

help_out="${tmp_dir}/help.out"
run_expect_code 0 "$help_out" node "$script" --help
assert_contains "$help_out" "Usage: node scripts/render-agent-decision-client-consumer-response.mjs"

out="${tmp_dir}/response.out"
node "$script" > "$out"
assert_contains "$out" "agent_decision_client_consumer_response_status=passed"
assert_contains "$out" "agent_next_step=ready"
assert_contains "$out" "helper_ref=v0.5.20"
assert_contains "$out" "fixture_count=8"
assert_contains "$out" "decisions=accept,accept,accept,manual-review,manual-review,manual-review,apply-repair,needs-better-input"

json_out="${tmp_dir}/response.json"
node "$script" --json > "$json_out"
python3 - "$json_out" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["schema_version"] == 1
assert payload["status"] == "passed"
assert payload["agent_next_step"] == "ready"
assert payload["evidence"]["helper_ref"] == "v0.5.20"
assert payload["evidence"]["fixture_count"] == 8
assert payload["failures"] == []
PY

runtime_summary="${tmp_dir}/runtime-summary.json"
scripts/showcase-agent-decision-client-consumer-smoke.sh --json > "$runtime_summary"
node "$script" "$runtime_summary" > "${tmp_dir}/runtime-response.out"
assert_contains "${tmp_dir}/runtime-response.out" "agent_decision_client_consumer_response_status=passed"
assert_contains "${tmp_dir}/runtime-response.out" "agent_next_step=ready"

bad_validator="docs/fixtures/agent-decision-client-consumer-smoke-summary/validator-failed.json"

run_expect_code 1 "${tmp_dir}/bad-validator.out" node "$script" "$bad_validator"
assert_contains "${tmp_dir}/bad-validator.out" "agent_decision_client_consumer_response_status=failed"
assert_contains "${tmp_dir}/bad-validator.out" "agent_next_step=inspect-consumer-smoke-validator"
assert_contains "${tmp_dir}/bad-validator.out" "fixture_validator_exit_code=1"

bad_decisions="docs/fixtures/agent-decision-client-consumer-smoke-summary/fixture-drift.json"

run_expect_code 1 "${tmp_dir}/bad-decisions.out" node "$script" "$bad_decisions"
assert_contains "${tmp_dir}/bad-decisions.out" "agent_next_step=inspect-agent-decision-fixtures"
assert_contains "${tmp_dir}/bad-decisions.out" "fixture_count=7, want 8"

echo "agent decision client consumer response test passed"
