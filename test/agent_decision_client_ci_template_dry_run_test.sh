#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

cd "$repo_root"

assert_exists() {
  path="$1"
  if [ ! -e "$path" ]; then
    echo "expected path to exist: $path" >&2
    exit 1
  fi
}

external_client_dir="${tmp_dir}/external-client"
client_output_dir="${tmp_dir}/testloop-agent-decision-client"
summary_json="${tmp_dir}/testloop-agent-decision-client-summary.json"
mkdir -p "$external_client_dir"
ln -s "$repo_root" "${external_client_dir}/.testloop-mcp"

(
  cd "$external_client_dir"
  TESTLOOP_AGENT_DECISION_CLIENT_DIR="$client_output_dir" \
    .testloop-mcp/scripts/showcase-agent-decision-client-ci.sh --json \
    | tee "$summary_json" >/dev/null
)

assert_exists "$summary_json"
assert_exists "${client_output_dir}/agent-decision-fixtures-result.json"
assert_exists "${client_output_dir}/testloop-agent-decision-fixtures/package.json"
assert_exists "${client_output_dir}/testloop-agent-decision-fixtures/docs/fixtures/agent-decision-fixtures.json"

python3 - "$summary_json" "$client_output_dir" <<'PY'
from pathlib import Path
import json
import sys

summary = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
client_output_dir = sys.argv[2]
assert summary["status"] == "passed"
assert summary["client_dir"] == client_output_dir
assert summary["fixture_dir"] == f"{client_output_dir}/testloop-agent-decision-fixtures"
assert summary["result_json"] == f"{client_output_dir}/agent-decision-fixtures-result.json"
assert summary["fixture_count"] == 8
assert summary["decisions"] == [
    "accept",
    "accept",
    "accept",
    "manual-review",
    "manual-review",
    "manual-review",
    "apply-repair",
    "needs-better-input",
]
assert summary["failures"] == []
assert summary["validator_exit_code"] == 0

validator_payload = json.loads(Path(summary["result_json"]).read_text(encoding="utf-8"))
assert validator_payload["status"] == "passed"
assert validator_payload["fixture_count"] == 8
assert validator_payload["failures"] == []
PY

echo "Agent decision client CI template dry-run test passed"
