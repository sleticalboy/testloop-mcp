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

script="scripts/verify-release-response-adopter-artifact.mjs"
showcase="scripts/showcase-release-response-adopter.sh"
repo_dir="${tmp_dir}/adopter-repo"
artifact_dir="${tmp_dir}/adopter-artifacts"
showcase_json="${tmp_dir}/showcase.json"

run_expect_code 0 "${tmp_dir}/help.out" node "$script" --help
assert_contains "${tmp_dir}/help.out" "Usage: node scripts/verify-release-response-adopter-artifact.mjs"

TESTLOOP_RELEASE_RESPONSE_ADOPTER_REPO_DIR="$repo_dir" \
TESTLOOP_RELEASE_RESPONSE_ADOPTER_ARTIFACT_DIR="$artifact_dir" \
  "$showcase" --json > "$showcase_json"

verify_out="${tmp_dir}/verify.out"
run_expect_code 0 "$verify_out" node "$script" "$artifact_dir"
assert_contains "$verify_out" "release_response_adopter_artifact_status=passed"
assert_contains "$verify_out" "release_ref=v0.5.20"
assert_contains "$verify_out" "fixture_count=8"
assert_contains "$verify_out" "agent_next_step=ready"
assert_contains "$verify_out" "should_accept=true"
assert_contains "$verify_out" "required_files=6"

verify_json="${tmp_dir}/verify.json"
run_expect_code 0 "$verify_json" node "$script" --json "$artifact_dir"
python3 - "$verify_json" <<'PY'
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
assert payload["required_files"] == 6
assert len(payload["files"]) == 6
assert all(item["exists"] is True for item in payload["files"])
assert payload["failures"] == []
PY

downloaded_artifact_dir="${tmp_dir}/downloaded/testloop-release-response-adopter-artifacts"
mkdir -p "$(dirname "$downloaded_artifact_dir")"
cp -R "$artifact_dir" "$downloaded_artifact_dir"
run_expect_code 0 "${tmp_dir}/downloaded.out" node "$script" "$downloaded_artifact_dir"

missing_file_dir="${tmp_dir}/missing-file"
cp -R "$artifact_dir" "$missing_file_dir"
rm "$missing_file_dir/testloop-release-response-summary-consumer.json"
run_expect_code 1 "${tmp_dir}/missing.out" node "$script" "$missing_file_dir"
assert_contains "${tmp_dir}/missing.out" "release_response_adopter_artifact_status=failed"
assert_contains "${tmp_dir}/missing.out" "agent_next_step=inspect-release-response-adopter-artifact"
assert_contains "${tmp_dir}/missing.out" "should_accept=false"
assert_contains "${tmp_dir}/missing.out" "missing required file testloop-release-response-summary-consumer.json"

bad_consumer_dir="${tmp_dir}/bad-consumer"
cp -R "$artifact_dir" "$bad_consumer_dir"
python3 - "$bad_consumer_dir/testloop-release-response-consumer.json" <<'PY'
from pathlib import Path
import json
import sys

path = Path(sys.argv[1])
payload = json.loads(path.read_text(encoding="utf-8"))
payload["agent_next_step"] = "inspect-release-consumer-response"
payload["should_accept"] = False
payload["failures"] = ["forced consumer drift"]
path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY
run_expect_code 1 "${tmp_dir}/bad-consumer.out" node "$script" --json "$bad_consumer_dir"
assert_contains "${tmp_dir}/bad-consumer.out" '"status": "failed"'
assert_contains "${tmp_dir}/bad-consumer.out" '"schema_version": 1'
assert_contains "${tmp_dir}/bad-consumer.out" '"agent_next_step": "inspect-release-response-adopter-artifact"'
assert_contains "${tmp_dir}/bad-consumer.out" '"should_accept": false'
assert_contains "${tmp_dir}/bad-consumer.out" 'consumer response agent_next_step=\"inspect-release-consumer-response\", want \"ready\"'
assert_contains "${tmp_dir}/bad-consumer.out" "consumer response failures must be an empty array"

run_expect_code 2 "${tmp_dir}/usage.err" node "$script"
assert_contains "${tmp_dir}/usage.err" "Usage: node scripts/verify-release-response-adopter-artifact.mjs"

echo "release response adopter artifact verify test passed"
