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

script="scripts/export-agent-decision-release-response-client.mjs"
sample="docs/fixtures/agent-decision-client-release-smoke-summary/passed.json"

help_out="${tmp_dir}/help.out"
run_expect_code 0 "$help_out" node "$script" --help
assert_contains "$help_out" "Usage: node scripts/export-agent-decision-release-response-client.mjs"

export_dir="${tmp_dir}/release-response-client"
out="${tmp_dir}/export.out"
node "$script" "$export_dir" > "$out"
assert_contains "$out" "agent_decision_release_response_client_export_status=passed"
assert_contains "$out" "response_fixture_count=5"

assert_exists "$export_dir/package.json"
assert_exists "$export_dir/README.md"
assert_exists "$export_dir/testloop-release-smoke-summary.json"
assert_exists "$export_dir/scripts/render-agent-decision-client-release-response.mjs"
assert_exists "$export_dir/scripts/assert-release-response.mjs"
assert_exists "$export_dir/docs/fixtures/agent-decision-client-release-response.schema.json"
assert_exists "$export_dir/docs/fixtures/agent-decision-client-release-response/passed.json"
assert_exists "$export_dir/docs/fixtures/agent-decision-client-release-response/installer-drift.json"
assert_exists "$export_dir/docs/fixtures/agent-decision-client-release-response/client-response-drift.json"
assert_exists "$export_dir/docs/fixtures/agent-decision-client-release-response/consumer-response-drift.json"
assert_exists "$export_dir/docs/fixtures/agent-decision-client-release-response/fixture-drift.json"

(
  cd "$export_dir"
  npm test --silent
)
assert_exists "$export_dir/testloop-release-response.json"
python3 - "$export_dir/testloop-release-response.json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "passed"
assert payload["agent_next_step"] == "ready"
assert payload["evidence"]["release_ref"] == "v0.5.21"
assert payload["failures"] == []
PY

if node "$script" "$export_dir" > "${tmp_dir}/second.out" 2>&1; then
  echo "expected export to fail for non-empty output directory" >&2
  exit 1
fi
assert_contains "${tmp_dir}/second.out" "output directory already exists and is not empty"

bad_summary="${tmp_dir}/bad-summary.json"
python3 - "$sample" "$bad_summary" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
payload["agent_next_steps"]["consumer"] = "inspect-consumer-smoke-summary"
Path(sys.argv[2]).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY
bad_export_dir="${tmp_dir}/bad-release-response-client"
node "$script" "$bad_export_dir" "$bad_summary" > "${tmp_dir}/bad-export.out"
run_expect_code 1 "${tmp_dir}/bad-npm.out" sh -c "cd '$bad_export_dir' && npm test --silent"
python3 - "$bad_export_dir/testloop-release-response.json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "failed"
assert payload["agent_next_step"] == "inspect-release-consumer-response"
PY

echo "agent decision release response client export test passed"
