#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

cd "$repo_root"

export_dir="${tmp_dir}/agent-decision-fixtures"
out="${tmp_dir}/export.out"
node scripts/export-agent-decision-fixtures.mjs "$export_dir" > "$out"

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

assert_contains "$out" "agent_decision_fixture_export_status=passed"
assert_contains "$out" "fixture_count=8"
assert_exists "$export_dir/README.md"
assert_exists "$export_dir/package.json"
assert_exists "$export_dir/scripts/validate-agent-decision-fixtures.mjs"
assert_exists "$export_dir/docs/fixtures/agent-decision-fixtures.json"
assert_exists "$export_dir/docs/fixtures/agent-decision-fixtures.schema.json"
assert_exists "$export_dir/docs/fixtures/agent-decision-fixtures-result.schema.json"
assert_exists "$export_dir/docs/fixtures/agent-decision-fixtures-result/passed.json"
assert_exists "$export_dir/docs/fixtures/agent-decision-client-ci-summary.schema.json"
assert_exists "$export_dir/docs/fixtures/agent-decision-client-ci-summary/passed.json"
assert_exists "$export_dir/docs/fixtures/agent-decision-client-ci-response.schema.json"
assert_exists "$export_dir/docs/fixtures/agent-decision-client-ci-response/passed.json"
assert_exists "$export_dir/docs/fixtures/agent-decision-client-ci-response/validator-failed.json"
assert_exists "$export_dir/docs/fixtures/agent-decision-client-ci-response/fixture-drift.json"
assert_exists "$export_dir/scripts/validate-agent-decision-client-ci-summary.mjs"
assert_exists "$export_dir/scripts/render-agent-decision-client-ci-response.mjs"
assert_exists "$export_dir/scripts/validate-agent-decision-client-ci-response.mjs"
assert_exists "$export_dir/docs/fixtures/real-project-agent-loop/haoy-apk-station-py-external-service.json"

if [ -e "$export_dir/docs/fixtures/run-tests/apply-fix-suggestions.json" ]; then
  echo "export should contain only agent decision fixtures, not run_tests fixtures" >&2
  exit 1
fi

json_out="${tmp_dir}/validator.json"
(
  cd "$export_dir"
  node scripts/validate-agent-decision-fixtures.mjs --json \
    docs/fixtures/agent-decision-fixtures.json \
    . > "$json_out"
)

python3 - "$json_out" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["schema_version"] == 1
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

npm_json_out="${tmp_dir}/validator-npm.json"
(
  cd "$export_dir"
  npm test --silent > "$npm_json_out"
)
python3 - "$npm_json_out" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["schema_version"] == 1
assert payload["status"] == "passed"
assert payload["fixture_count"] == 8
assert payload["failures"] == []
PY

summary_validator_out="${tmp_dir}/client-summary-validator.out"
response_json="${tmp_dir}/client-response.json"
response_validator_out="${tmp_dir}/client-response-validator.out"
(
  cd "$export_dir"
  npm run validate:client-summary --silent > "$summary_validator_out"
  npm run render:client-response --silent > "$response_json"
  npm run validate:client-response --silent > "$response_validator_out"
)
assert_contains "$summary_validator_out" "agent_decision_client_ci_summary_status=passed fixture_count=8"
assert_contains "$response_validator_out" "agent_decision_client_ci_response_status=passed agent_next_step=ready"
python3 - "$response_json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["schema_version"] == 1
assert payload["status"] == "passed"
assert payload["agent_next_step"] == "ready"
assert payload["evidence"]["fixture_count"] == 8
assert payload["failures"] == []
PY

if node scripts/export-agent-decision-fixtures.mjs "$export_dir" > "${tmp_dir}/second.out" 2>&1; then
  echo "expected export to fail for a non-empty output directory" >&2
  exit 1
fi
assert_contains "${tmp_dir}/second.out" "output directory already exists and is not empty"

echo "agent decision fixture export test passed"
