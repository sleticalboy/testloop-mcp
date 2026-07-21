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
node scripts/validate-agent-decision-client-consumer-smoke-summary.mjs > "$out"
assert_contains "$out" "agent_decision_client_consumer_smoke_summary_status=passed fixture_count=8"
assert_contains "$out" "agent_decision_client_consumer_smoke_summary_decisions=accept,accept,accept,manual-review,manual-review,manual-review,apply-repair,needs-better-input"

json_out="${tmp_dir}/validator.json"
node scripts/validate-agent-decision-client-consumer-smoke-summary.mjs --json > "$json_out"
python3 - "$json_out" <<'PY'
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

runtime_summary="${tmp_dir}/runtime-summary.json"
scripts/showcase-agent-decision-client-consumer-smoke.sh --json > "$runtime_summary"
node scripts/validate-agent-decision-client-consumer-smoke-summary.mjs "$runtime_summary" > "${tmp_dir}/runtime-validator.out"
assert_contains "${tmp_dir}/runtime-validator.out" "agent_decision_client_consumer_smoke_summary_status=passed fixture_count=8"

bad_summary="${tmp_dir}/bad-summary.json"
python3 - "$bad_summary" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path("docs/fixtures/agent-decision-client-consumer-smoke-summary/passed.json").read_text(encoding="utf-8"))
payload["status"] = "failed"
payload["helper_ref"] = "main"
payload["fixture_validator_exit_code"] = 1
payload["failures"] = ["boom"]
Path(sys.argv[1]).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY

if node scripts/validate-agent-decision-client-consumer-smoke-summary.mjs "$bad_summary" > "${tmp_dir}/bad.out" 2>&1; then
  echo "expected validator to fail for invalid consumer smoke summary" >&2
  exit 1
fi
assert_contains "${tmp_dir}/bad.out" "status must be passed"
assert_contains "${tmp_dir}/bad.out" "helper_ref must be v0.5.18"
assert_contains "${tmp_dir}/bad.out" "fixture_validator_exit_code must be 0"
assert_contains "${tmp_dir}/bad.out" "failures must be an empty array"

if node scripts/validate-agent-decision-client-consumer-smoke-summary.mjs --json "$bad_summary" > "${tmp_dir}/bad.json"; then
  echo "expected JSON validator to fail for invalid consumer smoke summary" >&2
  exit 1
fi
python3 - "${tmp_dir}/bad.json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "failed"
assert payload["fixture_count"] == 8
assert payload["failures"]
assert any("status must be passed" in item for item in payload["failures"])
PY

if node scripts/validate-agent-decision-client-consumer-smoke-summary.mjs docs/fixtures/agent-decision-client-consumer-smoke-summary/validator-failed.json > "${tmp_dir}/validator-fixture.out" 2>&1; then
  echo "expected validator-failed fixture to fail validation" >&2
  exit 1
fi
assert_contains "${tmp_dir}/validator-fixture.out" "status must be passed"
assert_contains "${tmp_dir}/validator-fixture.out" "fixture_validator_exit_code must be 0"
assert_contains "${tmp_dir}/validator-fixture.out" "failures must be an empty array"

echo "agent decision client consumer smoke summary validator test passed"
