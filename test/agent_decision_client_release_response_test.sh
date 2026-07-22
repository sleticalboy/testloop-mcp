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

script="scripts/render-agent-decision-client-release-response.mjs"
sample="docs/fixtures/agent-decision-client-release-smoke-summary/passed.json"

help_out="${tmp_dir}/help.out"
run_expect_code 0 "$help_out" node "$script" --help
assert_contains "$help_out" "Usage: node scripts/render-agent-decision-client-release-response.mjs"

out="${tmp_dir}/response.out"
node "$script" > "$out"
assert_contains "$out" "agent_decision_client_release_response_status=passed"
assert_contains "$out" "agent_next_step=ready"
assert_contains "$out" "release_ref=v0.5.21"
assert_contains "$out" "helper_refs=v0.5.21,v0.5.21"
assert_contains "$out" "agent_next_steps=ready,ready"

json_out="${tmp_dir}/response.json"
node "$script" --json > "$json_out"
python3 - "$json_out" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["schema_version"] == 1
assert payload["status"] == "passed"
assert payload["agent_next_step"] == "ready"
assert payload["evidence"]["release_ref"] == "v0.5.21"
assert payload["evidence"]["helper_refs"] == {"install": "v0.5.21", "consumer": "v0.5.21"}
assert payload["evidence"]["agent_next_steps"] == {"client": "ready", "consumer": "ready"}
assert payload["failures"] == []
PY

runtime_summary="${tmp_dir}/runtime-summary.json"
TESTLOOP_AGENT_DECISION_RELEASE_INSTALLER_URL="file://${repo_root}/scripts/install-agent-decision-client-ci-template.sh" \
  scripts/showcase-agent-decision-client-release-smoke.sh --json > "$runtime_summary"
node "$script" "$runtime_summary" > "${tmp_dir}/runtime-response.out"
assert_contains "${tmp_dir}/runtime-response.out" "agent_decision_client_release_response_status=passed"
assert_contains "${tmp_dir}/runtime-response.out" "agent_next_step=ready"

bad_installer="${tmp_dir}/bad-installer.json"
python3 - "$sample" "$bad_installer" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
payload["helper_refs"]["consumer"] = "v0.5.18"
Path(sys.argv[2]).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY
run_expect_code 1 "${tmp_dir}/bad-installer.out" node "$script" "$bad_installer"
assert_contains "${tmp_dir}/bad-installer.out" "agent_next_step=inspect-release-installer"
assert_contains "${tmp_dir}/bad-installer.out" "helper_refs.consumer=v0.5.18, want v0.5.21"

bad_client="${tmp_dir}/bad-client.json"
python3 - "$sample" "$bad_client" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
payload["agent_next_steps"]["client"] = "inspect-agent-decision-client-summary"
Path(sys.argv[2]).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY
run_expect_code 1 "${tmp_dir}/bad-client.out" node "$script" "$bad_client"
assert_contains "${tmp_dir}/bad-client.out" "agent_next_step=inspect-release-client-response"

bad_consumer="${tmp_dir}/bad-consumer.json"
python3 - "$sample" "$bad_consumer" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
payload["agent_next_steps"]["consumer"] = "inspect-consumer-smoke-summary"
Path(sys.argv[2]).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY
run_expect_code 1 "${tmp_dir}/bad-consumer.out" node "$script" "$bad_consumer"
assert_contains "${tmp_dir}/bad-consumer.out" "agent_next_step=inspect-release-consumer-response"

run_expect_code 1 "${tmp_dir}/missing.out" node "$script" "${tmp_dir}/missing.json"
assert_contains "${tmp_dir}/missing.out" "agent_next_step=inspect-release-smoke-summary"

echo "agent decision client release response test passed"
