#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

cd "$repo_root"

summary_json="${tmp_dir}/agent-decision-client-ci-summary.json"
TESTLOOP_AGENT_DECISION_CLIENT_DIR="${tmp_dir}/client" \
  bash scripts/showcase-agent-decision-client-ci.sh --json > "$summary_json"

python3 - \
  "$summary_json" \
  docs/fixtures/agent-decision-client-ci-summary.schema.json \
  docs/fixtures/agent-decision-client-ci-summary/passed.json <<'PY'
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
expected_decisions = [
    "accept",
    "accept",
    "accept",
    "manual-review",
    "manual-review",
    "manual-review",
    "apply-repair",
    "needs-better-input",
]

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

    if payload.get("schema_version") != 1:
        failures.append(f"{label} schema_version must be 1")
    if payload.get("status") != "passed":
        failures.append(f"{label} status must be passed")
    if payload.get("fixture_count") != 8:
        failures.append(f"{label} fixture_count must be 8")
    if payload.get("decisions") != expected_decisions:
        failures.append(f"{label} decisions drifted")
    if payload.get("failures") != []:
        failures.append(f"{label} failures must be empty")
    if payload.get("validator_exit_code") != 0:
        failures.append(f"{label} validator_exit_code must be 0")
    for key in ("client_dir", "fixture_dir", "result_json", "result_schema"):
        if not isinstance(payload.get(key), str) or not payload[key]:
            failures.append(f"{label} {key} must be a non-empty string")

validate_payload(summary, "generated summary")
validate_payload(sample, "fixture sample")

for key in ("client_dir", "fixture_dir", "result_json", "result_schema"):
    if not Path(summary[key]).exists():
        failures.append(f"generated summary {key} does not exist: {summary[key]}")

if summary.get("result_schema") != str(Path(summary["fixture_dir"]) / "docs/fixtures/agent-decision-fixtures-result.schema.json"):
    failures.append("generated summary result_schema must point inside fixture_dir")

if failures:
    print("Agent decision client CI summary schema test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("Agent decision client CI summary schema test passed")
PY
