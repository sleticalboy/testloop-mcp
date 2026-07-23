#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import json
import sys

schema = Path("docs/fixtures/agent-decision-client-adopter-summary.schema.json")
fixture_dir = Path("docs/fixtures/agent-decision-client-adopter-summary")
schema_payload = json.loads(schema.read_text(encoding="utf-8"))
fixtures = {
    "passed.json": {
        "status": "passed",
        "fixture_count": 8,
        "agent_next_step": "ready",
        "should_accept": True,
        "npm_exit_code": 0,
        "failures": [],
    },
    "invalid-response.json": {
        "status": "failed",
        "fixture_count": 8,
        "agent_next_step": "inspect-client-validator",
        "should_accept": False,
        "npm_exit_code": 0,
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

    fixture_payload = json.loads(fixture.read_text(encoding="utf-8"))

    for field in required:
        if field not in fixture_payload:
            failures.append(f"{fixture}: missing required field {field}")

    extra = sorted(set(fixture_payload) - set(required))
    if extra:
        failures.append(f"{fixture}: fixture has unexpected fields {extra}")

    want = {"schema_version": 1, **expected}
    for key, expected_value in want.items():
        if fixture_payload.get(key) != expected_value:
            failures.append(f"{fixture}: {key} = {fixture_payload.get(key)!r}, want {expected_value!r}")

    for key in [
        "repo_dir",
        "readme_path",
        "package_dir",
        "result_json",
        "response_json",
        "response_validation_json",
        "consumer_json",
    ]:
        if not isinstance(fixture_payload.get(key), str) or not fixture_payload[key]:
            failures.append(f"{fixture}: {key} must be a non-empty string")

    if fixture_payload.get("status") == "failed" and not fixture_payload.get("failures"):
        failures.append(f"{fixture}: failed fixture must include failures[]")

if schema_payload.get("additionalProperties") is not False:
    failures.append(f"{schema}: additionalProperties must be false")

if failures:
    print("Agent decision client adopter summary schema test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("Agent decision client adopter summary schema test passed")
PY
