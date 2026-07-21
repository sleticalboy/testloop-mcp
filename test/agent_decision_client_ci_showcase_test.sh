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

assert_exists() {
  path="$1"
  if [ ! -e "$path" ]; then
    echo "expected path to exist: $path" >&2
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

bash -n scripts/showcase-agent-decision-client-ci.sh

help_out="${tmp_dir}/help.out"
run_expect_code 0 "$help_out" bash scripts/showcase-agent-decision-client-ci.sh --help
assert_contains "$help_out" "Usage: scripts/showcase-agent-decision-client-ci.sh"
assert_contains "$help_out" "TESTLOOP_AGENT_DECISION_CLIENT_DIR"
assert_contains "$help_out" "TESTLOOP_AGENT_DECISION_RESULT_JSON"

args_out="${tmp_dir}/args.out"
run_expect_code 2 "$args_out" bash scripts/showcase-agent-decision-client-ci.sh extra
assert_contains "$args_out" "Usage: scripts/showcase-agent-decision-client-ci.sh"

client_file="${tmp_dir}/client-file"
printf 'not a directory\n' > "$client_file"
client_out="${tmp_dir}/client-file.out"
run_expect_code 1 "$client_out" env \
  TESTLOOP_AGENT_DECISION_CLIENT_DIR="$client_file" \
  bash scripts/showcase-agent-decision-client-ci.sh
assert_contains "$client_out" "client dir path must be a directory"

result_dir="${tmp_dir}/result-dir"
mkdir -p "$result_dir"
result_out="${tmp_dir}/result-dir.out"
run_expect_code 1 "$result_out" env \
  TESTLOOP_AGENT_DECISION_CLIENT_DIR="${tmp_dir}/client-for-result-dir" \
  TESTLOOP_AGENT_DECISION_RESULT_JSON="$result_dir" \
  bash scripts/showcase-agent-decision-client-ci.sh
assert_contains "$result_out" "result JSON path must not be a directory"

fixture_dir="${tmp_dir}/non-empty-fixture-dir"
mkdir -p "$fixture_dir"
printf 'existing\n' > "${fixture_dir}/existing.txt"
fixture_out="${tmp_dir}/non-empty-fixture.out"
run_expect_code 1 "$fixture_out" env \
  TESTLOOP_AGENT_DECISION_CLIENT_DIR="${tmp_dir}/client-for-non-empty-fixture" \
  TESTLOOP_AGENT_DECISION_FIXTURE_DIR="$fixture_dir" \
  bash scripts/showcase-agent-decision-client-ci.sh
assert_contains "$fixture_out" "fixture package directory must be empty"

default_first_out="${tmp_dir}/default-first.out"
run_expect_code 0 "$default_first_out" env TMPDIR="$tmp_dir" bash scripts/showcase-agent-decision-client-ci.sh
assert_contains "$default_first_out" "agent_decision_client_status=passed"
assert_contains "$default_first_out" "agent_decision_fixture_count=8"

default_second_out="${tmp_dir}/default-second.out"
run_expect_code 0 "$default_second_out" env TMPDIR="$tmp_dir" bash scripts/showcase-agent-decision-client-ci.sh
assert_contains "$default_second_out" "agent_decision_client_status=passed"
assert_contains "$default_second_out" "agent_decision_fixture_count=8"

client_dir="${tmp_dir}/client"
exported_fixture_dir="${client_dir}/testloop-agent-decision-fixtures"
result_json="${client_dir}/agent-decision-fixtures-result.json"
success_out="${tmp_dir}/success.out"
run_expect_code 0 "$success_out" env \
  TESTLOOP_AGENT_DECISION_CLIENT_DIR="$client_dir" \
  bash scripts/showcase-agent-decision-client-ci.sh

assert_contains "$success_out" "agent_decision_client_status=passed"
assert_contains "$success_out" "agent_decision_client_dir=$client_dir"
assert_contains "$success_out" "agent_decision_fixture_dir=$exported_fixture_dir"
assert_contains "$success_out" "agent_decision_result_json=$result_json"
assert_contains "$success_out" "agent_decision_fixture_count=8"
assert_contains "$success_out" "agent_decision_decisions=accept,accept,accept,manual-review,manual-review,manual-review,apply-repair,needs-better-input"
assert_exists "$exported_fixture_dir/package.json"
assert_exists "$exported_fixture_dir/scripts/validate-agent-decision-fixtures.mjs"
assert_exists "$exported_fixture_dir/docs/fixtures/agent-decision-fixtures.json"
assert_exists "$result_json"

python3 - "$result_json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "passed"
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
PY

echo "agent decision client CI showcase test passed"
