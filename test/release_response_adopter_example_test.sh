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

script="scripts/showcase-release-response-adopter.sh"
consumer="examples/release-response-adopter/scripts/read-testloop-release-response.mjs"
summary_consumer="examples/release-response-adopter/scripts/read-testloop-release-response-summary.mjs"
summary="docs/fixtures/agent-decision-client-release-smoke-summary/passed.json"
response="docs/fixtures/agent-decision-client-release-response/passed.json"
adopter_summary="docs/fixtures/release-response-adopter-summary/passed.json"
bad_adopter_summary="docs/fixtures/release-response-adopter-summary/invalid-response.json"
readme="examples/release-response-adopter/README.md"

help_out="${tmp_dir}/help.out"
run_expect_code 0 "$help_out" "$script" --help
assert_contains "$help_out" "Usage: scripts/showcase-release-response-adopter.sh"
assert_contains "$help_out" "TESTLOOP_RELEASE_RESPONSE_ADOPTER_REPO_DIR"

consumer_help="${tmp_dir}/consumer-help.out"
run_expect_code 0 "$consumer_help" node "$consumer" --help
assert_contains "$consumer_help" "Usage: node scripts/read-testloop-release-response.mjs"

summary_consumer_help="${tmp_dir}/summary-consumer-help.out"
run_expect_code 0 "$summary_consumer_help" node "$summary_consumer" --help
assert_contains "$summary_consumer_help" "Usage: node scripts/read-testloop-release-response-summary.mjs"

consumer_out="${tmp_dir}/consumer.out"
run_expect_code 0 "$consumer_out" node "$consumer" "$response"
assert_contains "$consumer_out" "testloop_release_response_status=passed"
assert_contains "$consumer_out" "testloop_release_response_next_step=ready"
assert_contains "$consumer_out" "testloop_release_response_release_ref=v0.5.20"
assert_contains "$consumer_out" "testloop_release_response_fixture_count=8"
assert_contains "$consumer_out" "testloop_release_response_should_accept=true"

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
assert payload["evidence"]["release_ref"] == "v0.5.20"
assert payload["evidence"]["fixture_count"] == 8
assert payload["failures"] == []
PY

summary_consumer_out="${tmp_dir}/summary-consumer.out"
run_expect_code 0 "$summary_consumer_out" node "$summary_consumer" "$adopter_summary"
assert_contains "$summary_consumer_out" "testloop_release_response_summary_status=passed"
assert_contains "$summary_consumer_out" "testloop_release_response_summary_next_step=ready"
assert_contains "$summary_consumer_out" "testloop_release_response_summary_release_ref=v0.5.20"
assert_contains "$summary_consumer_out" "testloop_release_response_summary_fixture_count=8"
assert_contains "$summary_consumer_out" "testloop_release_response_summary_should_accept=true"

bad_summary_consumer_out="${tmp_dir}/bad-summary-consumer.out"
run_expect_code 1 "$bad_summary_consumer_out" node "$summary_consumer" "$bad_adopter_summary"
assert_contains "$bad_summary_consumer_out" "testloop_release_response_summary_status=failed"
assert_contains "$bad_summary_consumer_out" "testloop_release_response_summary_next_step=inspect-release-smoke-summary"
assert_contains "$bad_summary_consumer_out" "testloop_release_response_summary_should_accept=false"
assert_contains "$bad_summary_consumer_out" "consumer agent_next_step=inspect-release-smoke-summary, want ready"

summary_consumer_json="${tmp_dir}/summary-consumer.json"
run_expect_code 1 "$summary_consumer_json" node "$summary_consumer" --json "$bad_adopter_summary"
python3 - "$summary_consumer_json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["schema_version"] == 1
assert payload["status"] == "failed"
assert payload["agent_next_step"] == "inspect-release-smoke-summary"
assert payload["should_accept"] is False
assert payload["release_ref"] == "v0.5.20"
assert payload["fixture_count"] == 8
assert payload["failures"]
PY

repo_dir="${tmp_dir}/adopter-repo"
showcase_json="${tmp_dir}/showcase.json"
TESTLOOP_RELEASE_RESPONSE_ADOPTER_REPO_DIR="$repo_dir" \
TESTLOOP_RELEASE_RESPONSE_ADOPTER_SUMMARY_JSON="$summary" \
  "$script" --json > "$showcase_json"

python3 - "$showcase_json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["schema_version"] == 1
assert payload["status"] == "passed"
assert payload["release_ref"] == "v0.5.20"
assert payload["fixture_count"] == 8
assert payload["agent_next_step"] == "ready"
assert payload["should_accept"] is True
assert payload["npm_exit_code"] == 0
assert payload["failures"] == []
for key in [
    "repo_dir",
    "readme_path",
    "workflow_path",
    "package_dir",
    "install_summary_json",
    "agent_response_json",
    "consumer_json",
]:
    assert Path(payload[key]).exists(), f"{key} does not exist: {payload[key]}"

workflow = Path(payload["workflow_path"]).read_text(encoding="utf-8")
assert "npm test --silent" in workflow
assert "testloop-release-response-contract" in workflow

consumer = json.loads(Path(payload["consumer_json"]).read_text(encoding="utf-8"))
assert consumer["agent_next_step"] == "ready"
assert consumer["should_accept"] is True
PY

assert_contains "${repo_dir}/README.md" "node scripts/read-testloop-release-response.mjs"
assert_contains "${repo_dir}/scripts/read-testloop-release-response.mjs" "testloop_release_response_next_step"
assert_contains "${repo_dir}/scripts/read-testloop-release-response-summary.mjs" "testloop_release_response_summary_next_step"
assert_contains "$readme" "scripts/install-agent-decision-release-response-client.sh"
assert_contains "$readme" "scripts/showcase-release-response-adopter.sh --json"
assert_contains "$readme" "node scripts/validate-release-response-adopter-summary.mjs"
assert_contains "$readme" "node scripts/read-testloop-release-response-summary.mjs"
assert_contains "$readme" "docs/fixtures/release-response-adopter-summary/invalid-response.json"
assert_contains "$readme" "testloop_release_response_status"
assert_contains "$readme" "agent_next_step"
assert_contains "$readme" "failures[]"
assert_contains "$readme" "testloop_release_response_summary_status"
assert_contains "$readme" "testloop_release_response_summary_next_step"
assert_contains "$readme" "testloop_release_response_summary_should_accept"
assert_contains "$readme" "testloop_release_response_summary_failures"
assert_contains "$readme" "inspect-release-installer"

bad_out="${tmp_dir}/missing.out"
run_expect_code 1 "$bad_out" node "$consumer" "${tmp_dir}/missing-response.json"
assert_contains "$bad_out" "testloop_release_response_next_step=inspect-release-smoke-summary"

echo "release response adopter example test passed"
