#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

cd "$repo_root"

summary_json="${tmp_dir}/summary.json"
client_dir="${tmp_dir}/client"
TESTLOOP_AGENT_DECISION_CI_INSTALLER_PATH="${repo_root}/scripts/install-agent-decision-client-ci-template.sh" \
TESTLOOP_AGENT_DECISION_CI_CLIENT_DIR="$client_dir" \
TESTLOOP_AGENT_DECISION_CI_HELPER_DIR="$repo_root" \
  bash scripts/showcase-agent-decision-client-ci-template-install.sh --json > "$summary_json"

python3 - "$summary_json" docs/fixtures/agent-decision-client-ci-template-install-summary.schema.json <<'PY'
from pathlib import Path
import json
import sys

summary_path = Path(sys.argv[1])
schema_path = Path(sys.argv[2])
summary = json.loads(summary_path.read_text(encoding="utf-8"))
schema = json.loads(schema_path.read_text(encoding="utf-8"))

required = schema["required"]
properties = schema["properties"]
failures = []

if schema.get("additionalProperties") is not False:
    failures.append("schema must reject additional properties")

missing = [key for key in required if key not in summary]
if missing:
    failures.append("summary missing required fields: " + ", ".join(missing))

extra = sorted(set(summary) - set(properties))
if extra:
    failures.append("summary has fields not declared by schema: " + ", ".join(extra))

for key in required:
    if key not in properties:
        failures.append(f"schema required field is not declared in properties: {key}")

expected_exact = {
    "schema_version": 1,
    "status": "passed",
    "helper_ref": "v0.5.16",
    "fixture_count": 8,
    "contract_exit_code": 0,
    "validator_exit_code": 0,
}
for key, want in expected_exact.items():
    if summary.get(key) != want:
        failures.append(f"{key} = {summary.get(key)!r}, want {want!r}")

if summary.get("decisions") != [
    "accept",
    "accept",
    "accept",
    "manual-review",
    "manual-review",
    "manual-review",
    "apply-repair",
    "needs-better-input",
]:
    failures.append("unexpected decisions sequence")
if summary.get("failures") != []:
    failures.append("failures must be empty on passed showcase")
for key in ("installer_path", "client_dir", "workflow_path", "helper_dir", "summary_json"):
    if not isinstance(summary.get(key), str) or not summary[key]:
        failures.append(f"{key} must be a non-empty string")

if not Path(summary["workflow_path"]).exists():
    failures.append("workflow_path does not exist")
if not Path(summary["summary_json"]).exists():
    failures.append("summary_json does not exist")

if failures:
    print("Agent decision client CI template install summary schema test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("Agent decision client CI template install summary schema test passed")
PY
