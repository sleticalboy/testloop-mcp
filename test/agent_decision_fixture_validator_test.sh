#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

cd "$repo_root"

out="${tmp_dir}/validator.out"
node scripts/validate-agent-decision-fixtures.mjs > "$out"
json_out="${tmp_dir}/validator.json"
node scripts/validate-agent-decision-fixtures.mjs --json > "$json_out"

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

assert_contains "$out" "agent_decision_fixture_status=passed fixture_count=8"
assert_contains "$out" "agent_decision_fixture_decisions=accept,accept,accept,manual-review,manual-review,manual-review,apply-repair,needs-better-input"

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
assert any(
    item["action"] == "manual_review_external_service"
    and item["decision"] == "manual-review"
    for item in payload["fixtures"]
)
assert any(
    item["action"] == "apply_fix_suggestions"
    and item["expected_decision"] == "apply-repair"
    for item in payload["fixtures"]
)
assert payload["failures"] == []
PY

bad_manifest="${tmp_dir}/agent-decision-fixtures.json"
python3 - "$bad_manifest" <<'PY'
from pathlib import Path
import json
import sys

manifest = json.loads(Path("docs/fixtures/agent-decision-fixtures.json").read_text(encoding="utf-8"))
manifest["fixtures"][0]["expected_decision"] = "manual-review"
Path(sys.argv[1]).write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
PY

if node scripts/validate-agent-decision-fixtures.mjs "$bad_manifest" "$repo_root" > "${tmp_dir}/bad.out" 2>&1; then
  echo "expected validator to fail for wrong expected_decision" >&2
  exit 1
fi
assert_contains "${tmp_dir}/bad.out" "decision=accept, expected=manual-review"

if node scripts/validate-agent-decision-fixtures.mjs --json "$bad_manifest" "$repo_root" > "${tmp_dir}/bad.json"; then
  echo "expected JSON validator to fail for wrong expected_decision" >&2
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
assert any("decision=accept, expected=manual-review" in item for item in payload["failures"])
PY

bad_metadata_manifest="${tmp_dir}/agent-decision-fixtures-bad-metadata.json"
python3 - "$bad_metadata_manifest" <<'PY'
from pathlib import Path
import json
import sys

manifest = json.loads(Path("docs/fixtures/agent-decision-fixtures.json").read_text(encoding="utf-8"))
manifest["fixtures"][0]["source"] = "unknown"
manifest["fixtures"][0]["client_expectation"] = ""
Path(sys.argv[1]).write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
PY

if node scripts/validate-agent-decision-fixtures.mjs --json "$bad_metadata_manifest" "$repo_root" > "${tmp_dir}/bad-metadata.json"; then
  echo "expected JSON validator to fail for invalid manifest metadata" >&2
  exit 1
fi
python3 - "${tmp_dir}/bad-metadata.json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "failed"
assert payload["failures"]
assert any("fixtures[0]: source: expected one of synthetic, real_project" in item for item in payload["failures"])
assert any("fixtures[0]: client_expectation: expected non-empty string" in item for item in payload["failures"])
PY

echo "agent decision fixture validator test passed"
