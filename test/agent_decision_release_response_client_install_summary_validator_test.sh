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

validator="scripts/validate-agent-decision-release-response-client-install-summary.mjs"

out="${tmp_dir}/validator.out"
node "$validator" > "$out"
assert_contains "$out" "agent_decision_release_response_client_install_summary_status=passed release_ref=v0.5.21"
assert_contains "$out" "agent_decision_release_response_client_install_summary_fixture_count=8"
assert_contains "$out" "agent_decision_release_response_client_install_summary_agent_next_step=ready"

json_out="${tmp_dir}/validator.json"
node "$validator" --json > "$json_out"
python3 - "$json_out" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "passed"
assert payload["release_ref"] == "v0.5.21"
assert payload["fixture_count"] == 8
assert payload["agent_next_step"] == "ready"
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

bad_summary="${tmp_dir}/bad-summary.json"
python3 - "$bad_summary" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path("docs/fixtures/agent-decision-release-response-client-install-summary/passed.json").read_text(encoding="utf-8"))
payload["status"] = "failed"
payload["agent_next_step"] = "inspect-release-client-response"
payload["decisions"][-1] = "inspect"
payload["npm_exit_code"] = 1
payload["failures"] = ["boom"]
Path(sys.argv[1]).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY

if node "$validator" "$bad_summary" > "${tmp_dir}/bad.out" 2>&1; then
  echo "expected validator to fail for invalid release response client install summary" >&2
  exit 1
fi
assert_contains "${tmp_dir}/bad.out" "status must be written"
assert_contains "${tmp_dir}/bad.out" "decisions must be accept,accept,accept,manual-review,manual-review,manual-review,apply-repair,needs-better-input"
assert_contains "${tmp_dir}/bad.out" "agent_next_step must be ready"
assert_contains "${tmp_dir}/bad.out" "npm_exit_code must be 0"
assert_contains "${tmp_dir}/bad.out" "failures must be an empty array"

if node "$validator" --json "$bad_summary" > "${tmp_dir}/bad.json"; then
  echo "expected JSON validator to fail for invalid release response client install summary" >&2
  exit 1
fi
python3 - "${tmp_dir}/bad.json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "failed"
assert payload["fixture_count"] == 8
assert payload["agent_next_step"] == "inspect-release-client-response"
assert payload["failures"]
assert any("status must be written" in item for item in payload["failures"])
PY

echo "agent decision release response client install summary validator test passed"
