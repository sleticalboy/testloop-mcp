#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

cd "$repo_root"

result_json="${tmp_dir}/agent-decision-fixtures-result.json"
node scripts/validate-agent-decision-fixtures.mjs --json > "$result_json"

python3 - \
  "$result_json" \
  docs/fixtures/agent-decision-fixtures-result.schema.json \
  docs/fixtures/agent-decision-fixtures-result/passed.json <<'PY'
from pathlib import Path
import json
import sys

result_path = Path(sys.argv[1])
schema_path = Path(sys.argv[2])
sample_path = Path(sys.argv[3])

result = json.loads(result_path.read_text(encoding="utf-8"))
schema = json.loads(schema_path.read_text(encoding="utf-8"))
sample = json.loads(sample_path.read_text(encoding="utf-8"))

required = schema["required"]
properties = schema["properties"]
fixture_properties = properties["fixtures"]["items"]["properties"]
fixture_required = properties["fixtures"]["items"]["required"]
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
    failures.append("schema must reject additional top-level properties")
if properties["fixtures"]["items"].get("additionalProperties") is not False:
    failures.append("schema must reject additional fixture properties")

for key in required:
    if key not in properties:
        failures.append(f"schema required field is not declared in properties: {key}")
for key in fixture_required:
    if key not in fixture_properties:
        failures.append(f"schema fixture required field is not declared in properties: {key}")

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

    fixtures = payload.get("fixtures")
    if not isinstance(fixtures, list) or len(fixtures) != payload.get("fixture_count"):
        failures.append(f"{label} fixtures length must match fixture_count")
        fixtures = []
    for index, item in enumerate(fixtures):
        item_label = f"{label} fixtures[{index}]"
        missing_item = [key for key in fixture_required if key not in item]
        if missing_item:
            failures.append(f"{item_label} missing required fields: " + ", ".join(missing_item))
        extra_item = sorted(set(item) - set(fixture_properties))
        if extra_item:
            failures.append(f"{item_label} has fields not declared by schema: " + ", ".join(extra_item))
        if item.get("status") != item.get("manifest_status"):
            failures.append(f"{item_label} status must match manifest_status")
        if item.get("action") != item.get("manifest_action"):
            failures.append(f"{item_label} action must match manifest_action")
        if not isinstance(item.get("path"), str) or not item["path"]:
            failures.append(f"{item_label} path must be a non-empty string")
        if item.get("action") in {"manual_review_internal", "manual_review_environment", "manual_review_external_service", "needs_better_input"}:
            if not isinstance(item.get("reason"), str) or not item["reason"]:
                failures.append(f"{item_label} reason must be present for reason-bearing actions")

validate_payload(result, "generated result")
validate_payload(sample, "fixture sample")

if result != sample:
    failures.append("generated result JSON must match passed fixture sample")

if failures:
    print("Agent decision fixtures result schema test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("Agent decision fixtures result schema test passed")
PY
