#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

cd "$repo_root"

summary_json="${tmp_dir}/summary.json"
client_dir="${tmp_dir}/client"
mkdir -p "$client_dir"
scripts/install-agent-decision-release-response-client.sh --json "$client_dir" > "$summary_json"

python3 - "$summary_json" docs/fixtures/agent-decision-release-response-client-install-summary.schema.json docs/fixtures/agent-decision-release-response-client-install-summary/passed.json <<'PY'
from pathlib import Path
import json
import sys

summary_path = Path(sys.argv[1])
schema_path = Path(sys.argv[2])
sample_path = Path(sys.argv[3])
summary = json.loads(summary_path.read_text(encoding="utf-8"))
schema = json.loads(schema_path.read_text(encoding="utf-8"))
sample = json.loads(sample_path.read_text(encoding="utf-8"))

required = schema["required"]
properties = schema["properties"]
failures = []

if schema.get("additionalProperties") is not False:
    failures.append("schema must reject additional properties")

for key in required:
    if key not in properties:
        failures.append(f"schema required field is not declared in properties: {key}")

def validate_payload(payload, label):
    missing = [key for key in required if key not in payload]
    if missing:
        failures.append(f"{label} missing required fields: " + ", ".join(missing))

    extra = sorted(set(payload) - set(properties))
    if extra:
        failures.append(f"{label} has fields not declared by schema: " + ", ".join(extra))

    expected_exact = {
        "schema_version": 1,
        "status": "written",
        "release_ref": "v0.5.19",
        "fixture_count": 8,
        "agent_next_step": "ready",
        "npm_exit_code": 0,
    }
    for key, want in expected_exact.items():
        if payload.get(key) != want:
            failures.append(f"{label} {key} = {payload.get(key)!r}, want {want!r}")

    if payload.get("decisions") != [
        "accept",
        "accept",
        "accept",
        "manual-review",
        "manual-review",
        "manual-review",
        "apply-repair",
        "needs-better-input",
    ]:
        failures.append(f"{label} unexpected decisions sequence")
    if payload.get("failures") != []:
        failures.append(f"{label} failures must be empty on written install")
    for key in ("client_dir", "workflow_path", "package_dir", "release_summary_json", "agent_response_json", "release_ref"):
        if not isinstance(payload.get(key), str) or not payload[key]:
            failures.append(f"{label} {key} must be a non-empty string")

validate_payload(summary, "generated summary")
validate_payload(sample, "fixture sample")

for key in ("workflow_path", "package_dir", "release_summary_json", "agent_response_json"):
    if not Path(summary[key]).exists():
        failures.append(f"generated summary {key} does not exist: {summary[key]}")

workflow = Path(summary["workflow_path"]).read_text(encoding="utf-8")
if "npm test --silent" not in workflow:
    failures.append("generated workflow missing npm test --silent")
if "testloop-release-response-contract" not in workflow:
    failures.append("generated workflow missing artifact name")

if failures:
    print("Agent decision release response client install summary schema test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("Agent decision release response client install summary schema test passed")
PY
