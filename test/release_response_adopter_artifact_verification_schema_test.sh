#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import json
import sys

schema = Path("docs/fixtures/release-response-adopter-artifact-verification.schema.json")
fixture_dir = Path("docs/fixtures/release-response-adopter-artifact-verification")
schema_payload = json.loads(schema.read_text(encoding="utf-8"))
required_paths = [
    "testloop-release-response-adopter-summary.json",
    "testloop-release-response-install-summary.json",
    "testloop-release-response-client/testloop-release-smoke-summary.json",
    "testloop-release-response-client/testloop-release-response.json",
    "testloop-release-response-consumer.json",
    "testloop-release-response-summary-consumer.json",
]
fixtures = {
    "passed.json": {
        "status": "passed",
        "agent_next_step": "ready",
        "should_accept": True,
        "failures": [],
        "missing": set(),
    },
    "missing-summary-consumer.json": {
        "status": "failed",
        "agent_next_step": "inspect-release-response-adopter-artifact",
        "should_accept": False,
        "missing": {"testloop-release-response-summary-consumer.json"},
    },
}

failures = []
required = schema_payload.get("required", [])
properties = schema_payload.get("properties", {})

for field in required:
    if field not in properties:
        failures.append(f"{schema}: required field {field} has no property schema")

for name, expected in fixtures.items():
    fixture = fixture_dir / name
    if not fixture.exists():
        failures.append(f"{fixture}: missing fixture")
        continue

    payload = json.loads(fixture.read_text(encoding="utf-8"))
    for field in required:
        if field not in payload:
            failures.append(f"{fixture}: missing required field {field}")

    extra = sorted(set(payload) - set(required))
    if extra:
        failures.append(f"{fixture}: fixture has unexpected fields {extra}")

    for key, expected_value in {
        "schema_version": 1,
        "release_ref": "v0.5.20",
        "fixture_count": 8,
        "required_files": 6,
        "status": expected["status"],
        "agent_next_step": expected["agent_next_step"],
        "should_accept": expected["should_accept"],
    }.items():
        if payload.get(key) != expected_value:
            failures.append(f"{fixture}: {key} = {payload.get(key)!r}, want {expected_value!r}")

    for key in ["artifact_dir", "summary_json"]:
        if not isinstance(payload.get(key), str) or not payload[key]:
            failures.append(f"{fixture}: {key} must be a non-empty string")

    files = payload.get("files")
    if not isinstance(files, list) or len(files) != len(required_paths):
        failures.append(f"{fixture}: files must contain {len(required_paths)} entries")
    else:
        seen = [entry.get("path") for entry in files if isinstance(entry, dict)]
        if seen != required_paths:
            failures.append(f"{fixture}: files paths = {seen!r}, want {required_paths!r}")
        for entry in files:
            if not isinstance(entry, dict):
                failures.append(f"{fixture}: file entry must be object")
                continue
            missing = entry["path"] in expected["missing"]
            if entry.get("exists") is (not missing):
                continue
            failures.append(f"{fixture}: files entry {entry.get('path')} exists={entry.get('exists')!r}, want {not missing!r}")

    failure_items = payload.get("failures")
    if payload.get("status") == "passed" and failure_items != []:
        failures.append(f"{fixture}: passed fixture failures must be []")
    if payload.get("status") == "failed" and not failure_items:
        failures.append(f"{fixture}: failed fixture must include failures[]")

if schema_payload.get("additionalProperties") is not False:
    failures.append(f"{schema}: additionalProperties must be false")

if failures:
    print("release response adopter artifact verification schema test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("release response adopter artifact verification schema test passed")
PY
