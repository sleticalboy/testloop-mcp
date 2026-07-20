#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

cd "$repo_root"

out="${tmp_dir}/validator.out"
node scripts/validate-agent-decision-fixtures.mjs > "$out"

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

echo "agent decision fixture validator test passed"
