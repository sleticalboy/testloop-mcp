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

validator="scripts/validate-release-response-adopter-summary.mjs"

out="${tmp_dir}/validator.out"
node "$validator" > "$out"
assert_contains "$out" "release_response_adopter_summary_status=passed release_ref=v0.5.20"
assert_contains "$out" "release_response_adopter_summary_fixture_count=8"
assert_contains "$out" "release_response_adopter_summary_agent_next_step=ready"
assert_contains "$out" "release_response_adopter_summary_should_accept=true"

json_out="${tmp_dir}/validator.json"
node "$validator" --json > "$json_out"
python3 - "$json_out" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "passed"
assert payload["release_ref"] == "v0.5.20"
assert payload["fixture_count"] == 8
assert payload["agent_next_step"] == "ready"
assert payload["should_accept"] is True
assert payload["npm_exit_code"] == 0
assert payload["failures"] == []
PY

showcase_json="${tmp_dir}/showcase.json"
TESTLOOP_RELEASE_RESPONSE_ADOPTER_REPO_DIR="${tmp_dir}/adopter-repo" \
TESTLOOP_RELEASE_RESPONSE_ADOPTER_SUMMARY_JSON="docs/fixtures/agent-decision-client-release-smoke-summary/passed.json" \
  scripts/showcase-release-response-adopter.sh --json > "$showcase_json"
node "$validator" "$showcase_json" > "${tmp_dir}/showcase-validator.out"
assert_contains "${tmp_dir}/showcase-validator.out" "release_response_adopter_summary_status=passed release_ref=v0.5.20"

bad_summary="${tmp_dir}/bad-summary.json"
python3 - "$bad_summary" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path("docs/fixtures/release-response-adopter-summary/passed.json").read_text(encoding="utf-8"))
payload["status"] = "failed"
payload["release_ref"] = "v0.0.0"
payload["fixture_count"] = 7
payload["agent_next_step"] = "inspect-release-installer"
payload["should_accept"] = False
payload["npm_exit_code"] = 1
payload["failures"] = ["boom"]
Path(sys.argv[1]).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY

if node "$validator" "$bad_summary" > "${tmp_dir}/bad.out" 2>&1; then
  echo "expected validator to fail for invalid release response adopter summary" >&2
  exit 1
fi
assert_contains "${tmp_dir}/bad.out" "status must be passed"
assert_contains "${tmp_dir}/bad.out" "release_ref must be v0.5.20"
assert_contains "${tmp_dir}/bad.out" "fixture_count must be 8"
assert_contains "${tmp_dir}/bad.out" "agent_next_step must be ready"
assert_contains "${tmp_dir}/bad.out" "should_accept must be true"
assert_contains "${tmp_dir}/bad.out" "npm_exit_code must be 0"
assert_contains "${tmp_dir}/bad.out" "failures must be an empty array"

if node "$validator" --json "$bad_summary" > "${tmp_dir}/bad.json"; then
  echo "expected JSON validator to fail for invalid release response adopter summary" >&2
  exit 1
fi
python3 - "${tmp_dir}/bad.json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "failed"
assert payload["release_ref"] == "v0.0.0"
assert payload["fixture_count"] == 7
assert payload["agent_next_step"] == "inspect-release-installer"
assert payload["should_accept"] is False
assert payload["npm_exit_code"] == 1
assert payload["failures"]
assert any("status must be passed" in item for item in payload["failures"])
PY

echo "release response adopter summary validator test passed"
